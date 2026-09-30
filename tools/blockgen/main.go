// Command blockgen emits regex/gen_blocks.go: the unicodeBlocks table that
// resolves a block escape \p{IsX} (Datatypes §G.4.2.3) to its code points.
//
// It reads two inputs rather than carrying a transcription of either
// (PRINCIPLES 26):
//
//   - a Unicode Character Database Blocks.txt, pinned under docs/specs/ucd/
//     because §G.4.2.3 defines blocks by [Unicode Database], which the local
//     spec corpus does not carry. The Unicode version is read from the file's
//     own "# Blocks-<version>.txt" header line and named in the generated
//     header, since which version's blocks apply is ·implementation-defined·
//     (§G.4.2.3);
//   - the Datatypes spec Markdown, for §G.4.2.3's list of Unicode 3.1 block
//     names superseded in later versions, which implementors "are encouraged
//     to support … for compatibility". Only the FIRST range the list gives a
//     name is taken: PrivateUse is #xE000-#xF8FF, Unicode 3.0's Private Use
//     block, and its two further bullets (#xF0000-#xFFFFD, #x100000-#x10FFFD)
//     are skipped. §G.4.2.3 only encourages superseded names and makes the
//     choice of block definitions ·implementation-defined·, and the suite
//     expects \p{IsPrivateUse} to exclude those supplementary planes, so
//     folding them in regresses suite-invalid cases (#1473).
//
// Each block name is keyed by its ·normalized block name·: white space and
// underbars stripped, hyphens and case retained (§G.4.2.3). normalize and
// isBlockName restate regex's normalizeBlockName and matchesIsBlock, and a key
// is reachable only while the two pairs agree: a tools/ main cannot import
// those unexported functions, and exporting them from regex would add library
// surface with no library consumer (STYLE T5). A key outside
// production [96]'s [a-zA-Z0-9#x2D]+, or one that two entries share, is an
// error rather than a silent skip or overwrite, so an input whose shape changes
// stops the build instead of emitting a short or unreachable table. Output is
// gofmt'd and in input order, so running it twice is byte-identical (STYLE D1).
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

func main() {
	blocksPath := flag.String("blocks", "docs/specs/ucd/Blocks-15.0.0.txt", "Unicode Character Database Blocks.txt")
	datatypesPath := flag.String("datatypes", "docs/specs/md/xmlschema11-2.md", "Datatypes spec Markdown")
	outPath := flag.String("out", "regex/gen_blocks.go", "output Go file")
	flag.Parse()

	if err := run(*blocksPath, *datatypesPath, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "blockgen: %v\n", err)
		os.Exit(1)
	}
}

func run(blocksPath, datatypesPath, outPath string) error {
	src, err := generate(blocksPath, datatypesPath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, src, 0o644)
}

// generate reads both inputs and returns the formatted Go source.
func generate(blocksPath, datatypesPath string) ([]byte, error) {
	blocksText, err := os.ReadFile(blocksPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", blocksPath, err)
	}
	version, blocks, err := parseBlocks(strings.Split(string(blocksText), "\n"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", blocksPath, err)
	}
	specText, err := os.ReadFile(datatypesPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", datatypesPath, err)
	}
	superseded, err := parseSuperseded(strings.Split(string(specText), "\n"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", datatypesPath, err)
	}
	all := append(blocks, superseded...)
	if err := checkKeys(all); err != nil {
		return nil, err
	}
	return emit(version, len(blocks), all)
}

// runeRange is one inclusive code-point interval of a block.
type runeRange struct {
	lo, hi rune
}

// block is one table entry: a normalized block name and its code points.
type block struct {
	key string
	rng runeRange
}

var (
	// versionLine is Blocks.txt's first line, which names the UCD version.
	versionLine = regexp.MustCompile(`^# Blocks-([0-9]+\.[0-9]+\.[0-9]+)\.txt$`)
	// dataLine is one Blocks.txt entry, "Start Code..End Code; Block Name",
	// once the trailing comment is stripped.
	dataLine = regexp.MustCompile(`^([0-9A-F]{4,6})\.\.([0-9A-F]{4,6}); (.+)$`)
	// supersededLine is one bullet of §G.4.2.3's superseded-name list, e.g.
	// "- #x0370 - #x03FF: Greek".
	supersededLine = regexp.MustCompile(`^- #x([0-9A-F]+) - #x([0-9A-F]+): (\S+)$`)
)

// supersededIntro opens §G.4.2.3's superseded-name list; the bullets that
// follow it, up to the first non-bullet line, are the list.
const supersededIntro = "block names from Unicode 3.1 known to have been superseded in this way included:"

// parseBlocks reads a Blocks.txt: the UCD version from its first line, and
// every data line as one single-range block. A line that is neither blank, a
// comment, nor a well-formed entry is an error.
func parseBlocks(lines []string) (string, []block, error) {
	m := versionLine.FindStringSubmatch(strings.TrimSpace(lines[0]))
	if m == nil {
		return "", nil, fmt.Errorf("line 1: %q does not name the Blocks.txt version", lines[0])
	}
	version := m[1]
	var blocks []block
	for i, line := range lines {
		body, _, _ := strings.Cut(line, "#")
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		d := dataLine.FindStringSubmatch(body)
		if d == nil {
			return "", nil, fmt.Errorf("line %d: %q is not a Blocks.txt entry", i+1, line)
		}
		r, err := parseRange(d[1], d[2])
		if err != nil {
			return "", nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		blocks = append(blocks, block{key: normalize(d[3]), rng: r})
	}
	if len(blocks) == 0 {
		return "", nil, fmt.Errorf("no block entries")
	}
	return version, blocks, nil
}

// parseSuperseded reads §G.4.2.3's superseded-name list from the Datatypes
// Markdown. A bullet repeating an earlier bullet's name is skipped, so each
// name keeps the first range the list gives it (see the package doc).
func parseSuperseded(lines []string) ([]block, error) {
	start := -1
	for i, line := range lines {
		if strings.Contains(line, supersededIntro) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("§G.4.2.3's superseded block-name list not found (%q)", supersededIntro)
	}
	var out []block
	for i := start; i < len(lines) && strings.HasPrefix(lines[i], "- "); i++ {
		m := supersededLine.FindStringSubmatch(lines[i])
		if m == nil {
			return nil, fmt.Errorf("line %d: %q is not a superseded block-name bullet", i+1, lines[i])
		}
		r, err := parseRange(m[1], m[2])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		key := normalize(m[3])
		if slices.ContainsFunc(out, func(b block) bool { return b.key == key }) {
			continue
		}
		out = append(out, block{key: key, rng: r})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("§G.4.2.3's superseded block-name list is empty")
	}
	return out, nil
}

// parseRange reads an inclusive hexadecimal code-point interval.
func parseRange(loHex, hiHex string) (runeRange, error) {
	lo, err := strconv.ParseUint(loHex, 16, 32)
	if err != nil {
		return runeRange{}, fmt.Errorf("range start %q: %w", loHex, err)
	}
	hi, err := strconv.ParseUint(hiHex, 16, 32)
	if err != nil {
		return runeRange{}, fmt.Errorf("range end %q: %w", hiHex, err)
	}
	if lo > hi || hi > 0x10FFFF {
		return runeRange{}, fmt.Errorf("range %s..%s is not an interval of code points", loHex, hiHex)
	}
	return runeRange{rune(lo), rune(hi)}, nil
}

// normalize returns the ·normalized block name· of a block name: white space
// (#x9/#xA/#xD/#x20) and underbars stripped, hyphens and case retained
// (§G.4.2.3).
func normalize(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// checkKeys rejects a key production [96] cannot spell — IsBlock ::= 'Is'
// [a-zA-Z0-9#x2D]+, so no escape could ever reach it — and a key two entries
// share, which a map literal would reject or one entry would shadow.
func checkKeys(blocks []block) error {
	seen := make(map[string]bool, len(blocks))
	for _, b := range blocks {
		if !isBlockName(b.key) {
			return fmt.Errorf("normalized block name %q is outside production [96]", b.key)
		}
		if seen[b.key] {
			return fmt.Errorf("normalized block name %q appears twice", b.key)
		}
		seen[b.key] = true
	}
	return nil
}

// isBlockName reports whether name is non-empty and drawn from production
// [96]'s [a-zA-Z0-9#x2D].
func isBlockName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-'
		if !ok {
			return false
		}
	}
	return true
}

// emit renders the table: the first nUCD entries are Blocks.txt's, the rest
// §G.4.2.3's superseded names.
func emit(version string, nUCD int, blocks []block) ([]byte, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "// Code generated by tools/blockgen from Unicode %s Blocks.txt; DO NOT EDIT.\n\n", version)
	b.WriteString("package regex\n\n")
	fmt.Fprintf(&b, "// unicodeBlocks maps each ·normalized block name· (Datatypes §G.4.2.3) to\n"+
		"// the code points of its block: the %d blocks of Unicode %s's Blocks.txt\n"+
		"// (docs/specs/ucd), then the Unicode 3.1 names §G.4.2.3 lists as superseded\n"+
		"// in later versions, each at the first range it lists, which it encourages\n"+
		"// supporting for compatibility.\n", nUCD, version)
	b.WriteString("var unicodeBlocks = map[string]runeRange{\n")
	for i, blk := range blocks {
		if i == nUCD {
			b.WriteString("\n// Superseded Unicode 3.1 block names (§G.4.2.3).\n")
		}
		fmt.Fprintf(&b, "%q: {0x%04X, 0x%04X},\n", blk.key, blk.rng.lo, blk.rng.hi)
	}
	b.WriteString("}\n")
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("formatting generated source: %w", err)
	}
	return src, nil
}
