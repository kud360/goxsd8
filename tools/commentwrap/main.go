// Command commentwrap reports `//` comment paragraphs whose line breaks no
// longer match the paragraph's own wrap width — the layout an edit leaves
// behind when the words it inserted or removed are not followed by a reflow.
//
// It is a gate step (CLAUDE.md's "Commands" block): a mechanical check for a
// mechanical defect. Judgment about what a comment SAYS stays with the arbiter
// and warden, per .golangci.yml's header; this tool only measures where the
// lines end. Two shapes are reported, the two an unreflowed edit leaves: a line
// stranded short of its paragraph's width with room to spare for the next
// line's first word, and a line pushed past that width on its own.
//
// Usage:
//
//	go tool commentwrap ./...        # whole module (the gate step)
//	go tool commentwrap ./value      # one package directory
//	go tool commentwrap -fix ./...   # reflow every reported paragraph in place
//
// It exits 0 when clean, 1 when it reports findings, and 2 on an operational
// error (unreadable or unparseable input).
//
// It exists because the defect it measures survived four sightings and a direct
// blocking review instruction naming the exact line (#329). That is the family
// #270 (advisory follow-ups with no tracker) and #315 (a body premise corrected
// only in its thread) belong to as well — a correction stated somewhere
// durable-looking but landing nowhere enforced. Their mechanisms differ from
// this one's and they are deliberately not merged with it.
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// shortSlack is how many columns short of its paragraph's wrap width a
	// line must break before the break counts as ragged rather than as an
	// ordinary hand-wrap choice. Tuned against this repo's comments: at 20
	// every reported line is an orphan left mid-paragraph, while at 10 the
	// report is dominated by breaks a human placed on purpose (before a long
	// parenthetical, after a clause) that no later edit disturbed.
	shortSlack = 20
	// longSlack is the mirror bound: how far past every other line of its
	// paragraph one line may run before it counts as an edit that grew a line
	// instead of reflowing it.
	longSlack = 12
	// maxFillWidth is the widest any paragraph in this repo is filled to. An
	// over-long line has to clear it as well as longSlack, because standing
	// clear of the paragraph is on its own a weak signal in a three-line
	// paragraph whose other two lines end on a clause; clearing a width no
	// paragraph here is wrapped to is what says the line was never wrapped at
	// all. Measured, not chosen: comment lines cluster up to 88 columns and
	// thin out immediately after, and every line in the repo past it is an
	// unreflowed insertion (#329's tabled mirror-defect site is 93).
	maxFillWidth = 88
	// minWrapWidth is the narrowest paragraph the check will judge. A paragraph
	// whose longest line stays under this was never filled to a wrap width —
	// a quoted lexical form, a short spec extract, a two-word aside — so there
	// is no width for its breaks to be ragged against.
	minWrapWidth = 60
	// verbatimIndent is the comment-body indentation (in columns after the
	// `//`) at which `go doc` stops treating text as prose and renders it
	// preformatted: spec quotes, XML fragments, code samples. A paragraph that
	// opens there is laid out by hand on purpose and is never judged.
	verbatimIndent = 4
	// hangingIndent is how far a continuation line may be indented past the
	// line that opened its paragraph and still belong to it — enough for a list
	// item's hanging indent ("//   - text" continued at "//     text"), not
	// enough for a nested block.
	hangingIndent = 3
)

func main() {
	fix := flag.Bool("fix", false, "reflow every reported paragraph in place instead of reporting it")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"./..."}
	}

	clean, err := run(args, *fix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commentwrap: %v\n", err)
		os.Exit(2)
	}
	if !clean {
		os.Exit(1)
	}
}

// run checks (or, with fix, reflows) every file named by args, reporting what
// it found. It returns false when it reported anything, which is the exit
// status main turns it into.
func run(args []string, fix bool) (bool, error) {
	files, err := collect(args)
	if err != nil {
		return false, err
	}

	used := make(map[int]bool, len(waivers))
	var findings []finding
	for _, path := range files {
		got, err := process(path, fix, used)
		if err != nil {
			return false, err
		}
		findings = append(findings, got...)
	}
	findings = append(findings, staleWaivers(files, used)...)

	if len(findings) == 0 {
		return true, nil
	}
	// Findings go to stderr, where this repo's tools report (and where the
	// lint gate's errcheck rules take a diagnostic write as unchecked by
	// design).
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "%s:%d: %s\n", f.file, f.line, f.msg)
	}
	fmt.Fprintf(os.Stderr, "commentwrap: %d ragged comment line(s); reflow each reported paragraph at its own wrap width (go tool commentwrap -fix)\n", len(findings))
	return false, nil
}

// finding is one ragged line: a break a reflow of its paragraph, at that
// paragraph's own wrap width, would not have placed there.
type finding struct {
	file string
	line int
	msg  string
}

// collect expands the command-line path arguments into a sorted, de-duplicated
// list of Go files, so a rerun on an unchanged tree prints byte-identical
// output (STYLE D1). A `...` suffix (as in `./...`) walks the directory
// recursively; a plain directory takes the Go files directly inside it; a file
// argument is taken as given.
func collect(args []string) ([]string, error) {
	seen := make(map[string]struct{})
	var files []string
	add := func(path string) {
		if _, dup := seen[path]; dup {
			return
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}

	for _, arg := range args {
		cleaned := filepath.Clean(arg)
		root, recursive := strings.CutSuffix(cleaned, string(filepath.Separator)+"...")
		if cleaned == "..." {
			root, recursive = ".", true
		}

		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("resolving path %s: %w", arg, err)
		}
		if !info.IsDir() {
			add(root)
			continue
		}
		if err := walk(root, recursive, add); err != nil {
			return nil, fmt.Errorf("collecting from %s: %w", arg, err)
		}
	}

	sort.Strings(files)
	return files, nil
}

// walk feeds every Go file under root to add. Directories holding no
// hand-written comment prose of ours — the git store, the vendored W3C suite
// and other testdata — are skipped wholesale.
func walk(root string, recursive bool, add func(string)) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if !recursive || skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			add(path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory tree: %w", err)
	}
	return nil
}

func skipDir(name string) bool {
	return name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".")
}

// process reports the ragged lines of one file, or reflows them when fix is
// set. Waivers matched along the way are recorded in used so a waiver that has
// stopped matching can be reported as stale.
func process(path string, fix bool, used map[int]bool) ([]finding, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(src), "\n")
	paras, err := comments(path, string(src), lines)
	if err != nil {
		return nil, err
	}

	var findings []finding
	var splices []splice
	for _, para := range paras {
		width, flaws := flaws(para)
		if len(flaws) == 0 {
			continue
		}
		if w, waived := waivedBy(path, para); waived {
			used[w] = true
			continue
		}
		if !fix {
			findings = append(findings, report(path, para, width, flaws)...)
			continue
		}
		from := flaws[0].line
		splices = append(splices, splice{
			start: para[from].num,
			end:   para[len(para)-1].num,
			lines: reflow(para, width, from),
		})
	}

	if !fix || len(splices) == 0 {
		return findings, nil
	}
	if err := rewrite(path, lines, splices); err != nil {
		return nil, err
	}
	return findings, nil
}

// comments parses src for its comments and returns the paragraphs a reflow
// acts on, in source order.
func comments(path, src string, lines []string) ([][]commentLine, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if isGenerated(lines) {
		// A generated file's layout belongs to its generator, not to whoever
		// last edited the tree.
		return nil, nil
	}

	var paras [][]commentLine
	for _, group := range file.Comments {
		// A comment group runs `//` lines and `/* */` blocks together, so a line
		// this check cannot carry paragraph text on ends the run rather than
		// being stepped over: a paragraph accumulated across it would span its
		// line, and a reflow splices over every line it spans, deleting it
		// (#329).
		block := make([]commentLine, 0, len(group.List))
		flush := func() {
			paras = append(paras, paragraphs(block)...)
			block = nil
		}
		for _, c := range group.List {
			pos := fset.Position(c.Slash)
			if !strings.HasPrefix(c.Text, "//") || pos.Line > len(lines) {
				flush()
				continue
			}
			raw := lines[pos.Line-1]
			// A comment sharing its line with code is not part of a wrapped
			// paragraph: the code to its left, not a wrap width, decides where
			// it starts and how much room is left.
			if strings.TrimSpace(raw) != c.Text {
				flush()
				continue
			}
			block = append(block, commentLine{
				num:  pos.Line,
				raw:  raw,
				body: strings.TrimPrefix(c.Text, "//"),
			})
		}
		flush()
	}
	return paras, nil
}

// isGenerated reports whether the file carries the conventional generated-code
// header (https://go.dev/s/generatedcode), which by convention precedes the
// package clause.
func isGenerated(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if strings.HasPrefix(line, "// Code generated ") && strings.HasSuffix(line, " DO NOT EDIT.") {
			return true
		}
	}
	return false
}

// commentLine is one whole-line `//` comment: its source line number, the raw
// line (indentation and marker included, since a reflow preserves both) and
// the text after the marker, which classifies the line.
type commentLine struct {
	num  int
	raw  string
	body string
}

// width is the line's total width in columns, the quantity a wrap width bounds.
func (l commentLine) width() int {
	return utf8.RuneCountInString(l.raw)
}

// indent is how far the line's text sits past its `//`, counting a tab as a
// full verbatim step since that is how `go doc` reads it.
func (l commentLine) indent() int {
	n := 0
	for _, r := range l.body {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += verbatimIndent
			continue
		}
		break
	}
	return n
}

// prefix is everything a reflowed line repeats verbatim: the line's
// indentation, its `//`, and the body indentation that follows.
func (l commentLine) prefix() string {
	marker := strings.Index(l.raw, "//") + len("//")
	body := l.raw[marker:]
	return l.raw[:marker] + body[:len(body)-len(strings.TrimLeft(body, " \t"))]
}

// isBreak reports whether a comment line can carry no wrapped paragraph text
// at all: a blank `//`, a directive (`//go:generate`, `//nolint:…` — no space
// after the marker), a `#` heading marker inside a package doc, or a markdown
// table row.
func (l commentLine) isBreak() bool {
	trimmed := strings.TrimSpace(l.body)
	if trimmed == "" {
		return true
	}
	if !strings.HasPrefix(l.body, " ") && !strings.HasPrefix(l.body, "\t") {
		return true
	}
	return isHeading(trimmed) || strings.HasPrefix(trimmed, "|")
}

// isHeading reports whether trimmed is a `#`-style section marker: one or more
// hashes, then a space or nothing. The space is load-bearing — this repo's
// comments open lines with "#329" issue references constantly, and those are
// ordinary paragraph text.
func isHeading(trimmed string) bool {
	hashes := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
	if hashes == 0 {
		return false
	}
	return len(trimmed) == hashes || trimmed[hashes] == ' '
}

// isExampleOutput reports whether a comment line opens the expected-output
// block of an [Example function]. Everything from there to the end of the
// comment is the string `go test` compares against, not prose: rewrapping it
// changes what the example asserts.
//
// [Example function]: https://pkg.go.dev/testing#hdr-Examples
func (l commentLine) isExampleOutput() bool {
	trimmed := strings.ToLower(strings.TrimSpace(l.body))
	trimmed = strings.TrimPrefix(trimmed, "unordered ")
	return strings.HasPrefix(trimmed, "output:")
}

// isListItem reports whether a comment line opens a list item — a bullet
// (`- `, `* `, `• `) or an enumerator (`1. `, `2) `, `(3) `, `a. `). Its
// continuation lines belong to it, so it opens a paragraph rather than
// breaking one, and the line above it is a legitimate short last line.
func (l commentLine) isListItem() bool {
	trimmed := strings.TrimSpace(l.body)
	for _, bullet := range []string{"- ", "* ", "• "} {
		if strings.HasPrefix(trimmed, bullet) {
			return true
		}
	}
	return isEnumerator(trimmed)
}

// isEnumerator reports whether trimmed opens with a numbered or lettered list
// marker: an optional "(", then digits or a single letter, then "." or ")" and
// a space.
func isEnumerator(trimmed string) bool {
	rest := strings.TrimPrefix(trimmed, "(")
	n := 0
	for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
		n++
	}
	if n == 0 && len(rest) > 0 && isLetter(rest[0]) {
		n = 1
	}
	if n == 0 || len(rest) < n+2 {
		return false
	}
	if rest[n] != '.' && rest[n] != ')' {
		return false
	}
	return rest[n+1] == ' '
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// paragraphs splits a run of whole-line comments into the units a reflow acts
// on, dropping the one-line ones (a single line has no wrap to be ragged). A
// paragraph ends at a break line, where a list item starts, and where the
// indentation leaves the opening line's band — each of those is a boundary a
// reflow must not cross, which is exactly what makes the line before it a
// legitimate short line.
func paragraphs(block []commentLine) [][]commentLine {
	var out [][]commentLine
	var cur []commentLine
	margin := 0
	flush := func() {
		if len(cur) > 1 {
			out = append(out, cur)
		}
		cur = nil
	}

	for _, line := range block {
		if line.isExampleOutput() {
			flush()
			return out
		}
		if line.isBreak() {
			flush()
			continue
		}
		indent := line.indent()
		if line.isListItem() || indent < margin || indent > margin+hangingIndent {
			flush()
		}
		if len(cur) == 0 {
			margin = indent
		}
		cur = append(cur, line)
	}
	flush()
	return out
}

// flaw is one line of a paragraph whose break a reflow would move: line is its
// index within the paragraph, over says which of the two shapes it is, and off
// is how many columns it misses the wrap width by.
type flaw struct {
	line int
	over bool
	off  int
}

// flaws returns the paragraph's wrap width and the lines that miss it. A
// paragraph whose opening line is verbatim-indented, or that was never filled
// to a wrap width at all, has none by construction.
func flaws(para []commentLine) (int, []flaw) {
	if para[0].indent() >= verbatimIndent && !para[0].isListItem() {
		return 0, nil
	}
	width := wrapWidth(para)
	if width < minWrapWidth {
		return 0, nil
	}

	var out []flaw
	for i, line := range para {
		if line.width() > width+longSlack && len(strings.Fields(line.body)) > 1 {
			out = append(out, flaw{line: i, over: true, off: line.width() - width})
			continue
		}
		if i == len(para)-1 || width-line.width() < shortSlack {
			continue
		}
		word := firstWord(para[i+1].body)
		if word == 0 || line.width()+1+word > width {
			continue
		}
		out = append(out, flaw{line: i, off: width - line.width()})
	}
	return width, out
}

// wrapWidth is the width the paragraph was filled to: its longest line, except
// where that line is itself the over-long defect — standing clear of every
// other line by more than longSlack AND past maxFillWidth. Such a line does not
// get to define the width it broke; the runner-up does. Standing clear takes at
// least two other lines to stand clear of, so a two-line paragraph is always
// its own width.
func wrapWidth(para []commentLine) int {
	widths := make([]int, len(para))
	for i, line := range para {
		widths[i] = line.width()
	}
	sort.Sort(sort.Reverse(sort.IntSlice(widths)))

	if len(widths) > 2 && widths[0]-widths[1] > longSlack && widths[0] > maxFillWidth {
		return widths[1]
	}
	return widths[0]
}

// firstWord is the width in columns of the first whitespace-delimited word of
// a comment body — the word a reflow would pull up onto the line above.
func firstWord(body string) int {
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return 0
	}
	return utf8.RuneCountInString(fields[0])
}

// report renders one paragraph's flaws as findings.
func report(path string, para []commentLine, width int, flaws []flaw) []finding {
	findings := make([]finding, 0, len(flaws))
	for _, f := range flaws {
		msg := fmt.Sprintf("comment line ends %d columns short of the paragraph's %d-column wrap width, and the next line's first word would fit", f.off, width)
		if f.over {
			msg = fmt.Sprintf("comment line runs %d columns past the paragraph's %d-column wrap width", f.off, width)
		}
		findings = append(findings, finding{file: path, line: para[f.line].num, msg: msg})
	}
	return findings
}

// reflow re-fills the paragraph from its first ragged line onward, greedily, at
// the paragraph's own wrap width. Everything above that line is left byte-identical:
// the break the edit disturbed is the first one that has to move, and a reflow
// that reaches further would rewrite lines no edit touched.
func reflow(para []commentLine, width, from int) []string {
	prefix := para[from].prefix()
	cont := prefix
	if from+1 < len(para) {
		cont = para[from+1].prefix()
	}

	var words []string
	for _, line := range para[from:] {
		words = append(words, strings.Fields(line.body)...)
	}

	var out []string
	cur := prefix
	empty := true
	for _, w := range words {
		if !empty && utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) > width {
			out = append(out, cur)
			cur, empty = cont, true
		}
		if !empty {
			cur += " "
		}
		cur += w
		empty = false
	}
	if !empty {
		out = append(out, cur)
	}
	return out
}

// splice is one reflowed run: source lines [start, end] (1-based, inclusive)
// become lines.
type splice struct {
	start int
	end   int
	lines []string
}

// rewrite applies the splices to the file's lines and writes it back. Splices
// are disjoint and rise through the file, so they are applied back to front and
// no line number shifts under one still to be applied.
func rewrite(path string, lines []string, splices []splice) error {
	for i := len(splices) - 1; i >= 0; i-- {
		s := splices[i]
		out := make([]string, 0, len(lines))
		out = append(out, lines[:s.start-1]...)
		out = append(out, s.lines...)
		out = append(out, lines[s.end:]...)
		lines = out
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// waiver suppresses one paragraph whose reflow belongs to another open issue,
// so this check does not land a fix that issue's branch is already carrying. A
// waiver names the line by its exact text rather than by number (line numbers
// drift under every landing) and names the issue that retires it. It is the
// same discipline STYLE P3 puts on `GAP(` markers: a waiver that no longer
// matches a ragged line is reported as stale and must be deleted, so it cannot
// outlive the issue that justified it.
type waiver struct {
	file  string
	text  string
	issue string
}

var waivers = []waiver{{
	file:  "value/list.go",
	text:  "// terminates structurally: neither the item type nor any",
	issue: "#564",
}}

// covers reports whether path is the file the waiver names. The waiver names a
// repo-relative path and the check runs on paths relative to wherever it was
// invoked, so the match is on whole path elements: a bare suffix match would
// let a waiver keyed "value/list.go" cover "xvalue/list.go" too.
func (w waiver) covers(path string) bool {
	slash := filepath.ToSlash(filepath.Clean(path))
	return slash == w.file || strings.HasSuffix(slash, "/"+w.file)
}

// waivedBy reports which waiver, if any, covers a paragraph. The whole
// paragraph is waived, not the single line: a reflow rewrites the paragraph
// from its first ragged break onward, so there is no way to fix a neighbouring
// break without touching the waived one.
func waivedBy(path string, para []commentLine) (int, bool) {
	for i, w := range waivers {
		if !w.covers(path) {
			continue
		}
		for _, line := range para {
			if strings.TrimSpace(line.raw) == w.text {
				return i, true
			}
		}
	}
	return 0, false
}

// staleWaivers reports every waiver whose file was checked but whose line is no
// longer a ragged one — the issue that owned it has landed, and the waiver has
// to go with it.
func staleWaivers(files []string, used map[int]bool) []finding {
	var findings []finding
	for i, w := range waivers {
		if used[i] {
			continue
		}
		for _, path := range files {
			if !w.covers(path) {
				continue
			}
			findings = append(findings, finding{
				file: path,
				line: 1,
				msg: fmt.Sprintf("stale waiver: %q is no longer a ragged line, so the waiver deferring it to %s must be deleted from tools/commentwrap",
					w.text, w.issue),
			})
			break
		}
	}
	return findings
}
