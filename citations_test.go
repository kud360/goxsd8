// This file guards, module-wide, the one documentation-citation convention
// that has already been broken at scale: `PRINCIPLES N` in a Go comment means
// item N of docs/PRINCIPLES.md — never position N in CLAUDE.md's "Style
// headlines" summary, which is a different list in a different order carrying
// no citable IDs (a style rule is cited by its STYLE.md letter ID). Both lists
// start at 1, so a number copied from the summary lands on a real but WRONG
// principle and then reads as correct forever; #299 corrected 38 such sites.
//
// Two halves, both mechanical and both cheap:
//
//   - BOUND: every cited N is a number docs/PRINCIPLES.md actually defines.
//     A citation of the summary's numbering that overshoots, or a plain typo,
//     fails here.
//   - ALLOW-LIST: the four numbers whose two lists disagree about the TOPIC —
//     4, 5, 7 and 9 — are pinned per file, with counts. A new citation of one
//     of them cannot land without editing the list below, and editing it is
//     the review this file exists to force. Numbers 11 and up are unambiguous
//     (the summary stops at 10) and answer to the bound check alone.
//
// What this does NOT do is decide whether a citation is TOPICALLY right; no
// mechanical check can. It makes the wrong-list mistake loud instead of silent.
package goxsd8_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// citationsGuardFile is this file, excluded from its own scan: it necessarily
// spells the pattern it searches for, and its allow-list is data, not citation.
const citationsGuardFile = "citations_test.go"

// collisionProne are the item numbers where docs/PRINCIPLES.md and CLAUDE.md's
// headline summary name DIFFERENT topics, so a citation is ambiguous about
// which list its author was reading: 4 (minimal capability views vs. one fact,
// one encoding), 5 (one fact, one encoding vs. no cycle checks), 7 (no
// concurrency vs. illegal states unrepresentable) and 9 (phased construction
// vs. fail-open XPath).
var collisionProne = []int{4, 5, 7, 9}

// allowedCollisionCitations is the reviewed state of every collision-prone
// citation in the module as of #299, which audited each site against the topic
// it actually argues. Counts are per file, not per line, so ordinary editing
// above a citation does not trip the guard while a new or copied citation
// does. 4 and 7 are absent deliberately: no site legitimately cites either
// today, so any appearance is a fresh mistake until reviewed onto this list.
var allowedCollisionCitations = []citationAllowance{
	{file: "builtin/strict/datetime.go", number: 5, count: 2},
	{file: "builtin/strict/duration.go", number: 5, count: 1},
	{file: "builtin/strict/gregorian.go", number: 5, count: 1},
	{file: "builtin/strict/strict.go", number: 5, count: 1},
	{file: "parser/produce.go", number: 9, count: 4},
	{file: "value/union.go", number: 5, count: 1},
	{file: "value/valuespace.go", number: 9, count: 1},
	{file: "xsd/attributeusefold.go", number: 9, count: 1},
	{file: "xsd/attributewildcardfold.go", number: 9, count: 1},
	{file: "xsd/complexderivation.go", number: 9, count: 1},
	{file: "xsd/complexextension.go", number: 9, count: 1},
	{file: "xsd/contentrestricts.go", number: 9, count: 2},
	{file: "xsd/effectivetotalrange.go", number: 9, count: 1},
	{file: "xsd/example_test.go", number: 9, count: 1},
	{file: "xsd/modelgroup.go", number: 9, count: 1},
	{file: "xsd/resolve.go", number: 9, count: 4},
	{file: "xsd/substitutiongroup.go", number: 9, count: 1},
	{file: "xsd/valueconstraintvalid.go", number: 9, count: 1},
	{file: "xsd/wildcardadmit.go", number: 9, count: 1},
}

// The vacuity floors. A scan that has stopped finding files passes every
// assertion it makes, so both halves declare the minimum they must see. Both
// sit safely below the counts at the time of writing (251 .go files, 94
// citations) so ordinary editing does not trip them.
const (
	minGoFilesScanned    = 200
	minCitationsFound    = 70
	minPrincipleItems    = 30
	principlesDocRelPath = "docs/PRINCIPLES.md"
)

// citationRe matches one `PRINCIPLES N` citation and captures N. Trailing text
// is deliberately unanchored: `(PRINCIPLES 9)`, `PRINCIPLES 9's` and
// `PRINCIPLES 26/27` all cite an item and all must be seen.
var citationRe = regexp.MustCompile(`PRINCIPLES ([0-9]+)`)

// principleItemRe matches a numbered item heading in docs/PRINCIPLES.md, whose
// items are `N. **Title.** …` at the start of a line.
var principleItemRe = regexp.MustCompile(`^([0-9]+)\. \*\*`)

// citationSite is one `PRINCIPLES N` occurrence, located for the failure
// message that has to send a reader to the exact comment.
type citationSite struct {
	file   string // slash-separated, relative to the module root
	line   int    // 1-based
	number int
}

// citationAllowance is one reviewed allow-list entry: file cites number
// exactly count times.
type citationAllowance struct {
	file   string
	number int
	count  int
}

// fileNumber keys the per-file citation counts. It is a lookup index only —
// reporting iterates the sites and the allow-list, both of which are ordered.
type fileNumber struct {
	file   string
	number int
}

// TestPrinciplesCitationsNameARealPrinciple is the BOUND half: every cited
// number is an item docs/PRINCIPLES.md defines.
func TestPrinciplesCitationsNameARealPrinciple(t *testing.T) {
	root := moduleRootDir(t)
	highest := highestPrincipleItem(t, root)
	for _, s := range principlesCitations(t, root) {
		if s.number >= 1 && s.number <= highest {
			continue
		}
		t.Errorf("%s:%d: cites PRINCIPLES %d, but %s defines items 1..%d — a number "+
			"copied from CLAUDE.md's headline summary is not a principle citation",
			s.file, s.line, s.number, principlesDocRelPath, highest)
	}
}

// TestCollisionProneCitationsAreAllowListed is the ALLOW-LIST half: every
// citation of a number the two lists disagree about was reviewed onto
// allowedCollisionCitations, and no reviewed entry has gone stale.
func TestCollisionProneCitationsAreAllowListed(t *testing.T) {
	root := moduleRootDir(t)
	allowed := make(map[fileNumber]int, len(allowedCollisionCitations))
	for _, a := range allowedCollisionCitations {
		allowed[fileNumber{file: a.file, number: a.number}] = a.count
	}
	found := make(map[fileNumber]int, len(allowedCollisionCitations))
	for _, s := range principlesCitations(t, root) {
		if !isCollisionProne(s.number) {
			continue
		}
		key := fileNumber{file: s.file, number: s.number}
		found[key]++
		if _, ok := allowed[key]; ok {
			continue
		}
		t.Errorf("%s:%d: cites PRINCIPLES %d, one of the numbers CLAUDE.md's headline "+
			"summary and %s disagree about. Confirm which list the comment means, then "+
			"add {file: %q, number: %d, count: N} to allowedCollisionCitations",
			s.file, s.line, s.number, principlesDocRelPath, s.file, s.number)
	}
	for _, a := range allowedCollisionCitations {
		got := found[fileNumber{file: a.file, number: a.number}]
		if got == a.count {
			continue
		}
		t.Errorf("%s: allow-listed for %d citations of PRINCIPLES %d, found %d — "+
			"re-check the citations, then update allowedCollisionCitations",
			a.file, a.count, a.number, got)
	}
}

// isCollisionProne reports whether n is one of the numbers the two lists
// disagree about.
func isCollisionProne(n int) bool {
	for _, c := range collisionProne {
		if c == n {
			return true
		}
	}
	return false
}

// principlesCitations returns every `PRINCIPLES N` citation in the module's Go
// sources, test files included, in walk order (lexical by path, then by line)
// so failures report identically on every run.
func principlesCitations(t *testing.T, root string) []citationSite {
	t.Helper()
	var out []citationSite
	files := goSourceFiles(t, root)
	if len(files) < minGoFilesScanned {
		t.Fatalf("scanned %d .go files under %s, expected at least %d — the walk is not "+
			"reaching the module", len(files), root, minGoFilesScanned)
	}
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			for _, m := range citationRe.FindAllStringSubmatch(line, -1) {
				n, err := strconv.Atoi(m[1])
				if err != nil {
					t.Fatalf("%s:%d: parsing citation %q: %v", rel, i+1, m[0], err)
				}
				out = append(out, citationSite{file: rel, line: i + 1, number: n})
			}
		}
	}
	if len(out) < minCitationsFound {
		t.Fatalf("found %d PRINCIPLES citations, expected at least %d — the scan is not "+
			"matching comments it used to", len(out), minCitationsFound)
	}
	return out
}

// goSourceFiles returns every .go file under root as a slash-separated relative
// path, skipping dot directories and testdata trees, in lexical walk order.
// This file is skipped: it spells the pattern it searches for.
func goSourceFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slashed := filepath.ToSlash(rel)
		if slashed == citationsGuardFile {
			return nil
		}
		out = append(out, slashed)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return out
}

// highestPrincipleItem returns the number of the last item in
// docs/PRINCIPLES.md, having checked the items run 1..N with no gap — the
// property that makes "N is in range" mean "N names that item".
func highestPrincipleItem(t *testing.T, root string) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(principlesDocRelPath)))
	if err != nil {
		t.Fatalf("reading %s: %v", principlesDocRelPath, err)
	}
	highest := 0
	for i, line := range strings.Split(string(b), "\n") {
		m := principleItemRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("%s:%d: parsing item number %q: %v", principlesDocRelPath, i+1, m[1], err)
		}
		if n != highest+1 {
			t.Fatalf("%s:%d: item %d follows item %d — the list must be numbered 1..N "+
				"with no gap for a citation's number to identify an item",
				principlesDocRelPath, i+1, n, highest)
		}
		highest = n
	}
	if highest < minPrincipleItems {
		t.Fatalf("%s defines %d numbered items, expected at least %d — the parse is not "+
			"matching item headings", principlesDocRelPath, highest, minPrincipleItems)
	}
	return highest
}

// moduleRootDir returns the working directory, which for this root-package test
// is the module root, after confirming go.mod is there.
func moduleRootDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
		t.Fatalf("go.mod not in %s: this test must run from the module root: %v", dir, err)
	}
	return dir
}
