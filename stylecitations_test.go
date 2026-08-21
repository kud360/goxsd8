// This file guards, module-wide, the STYLE-ID citation convention documented
// in docs/STYLE.md: "A style rule is cited by its letter ID from this file
// (`STYLE D4`, `STYLE T2`) — never by a position in CLAUDE.md's 'Style
// headlines' list, which is a summary and carries no citable IDs." It is the
// sibling of citations_test.go, which guards `PRINCIPLES N` against the same
// mistake — a token that reads as a citation but does not name the thing its
// author meant — and reuses that file's moduleRootDir and goSourceFiles
// rather than re-implementing the walk (STYLE T4).
//
// Two halves, matching the sibling's:
//
//   - BOUND: every citation token that is not bare digits must be an ID
//     docs/STYLE.md actually defines, parsed from its own `**ID. ` headings.
//     Slash-joined citations (`D2/D3`) are split and each token checked.
//     This half has no allow-list and takes no judgement: #382 ruled that
//     every invented letter ID in the tree — `T7`, `T8`, `D6`, `L6` — meant
//     a rule docs/STYLE.md already states, and renumbered all 32 sites onto
//     it, so an ID-shaped token naming no rule has no legitimate form left.
//   - ALLOW-LIST: a bare-numeric token is a position in CLAUDE.md's headline
//     list, which is #540's still-open defect. Every instance is pinned in
//     allowedBadStyleCitations by file, token and count — exactly
//     allowedCollisionCitations' shape — so a NEW or COPIED one cannot land
//     without editing that list, and editing it is the review this file
//     exists to force. The list shrinking to empty is what closes #540; it
//     is not this file's job to shrink it. An entry whose token is NOT bare
//     digits is itself an error, so the list cannot re-admit #382's class.
//
// The bound half is exactly decidable and that is its whole value. It does
// not decide whether a citation is TOPICALLY right — whether a REAL ID's rule
// governs the code it decorates — which #543 and #548 own and no mechanical
// check can settle.
package goxsd8_test

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// styleCitationsGuardFile is this file, excluded from its own scan for the
// same reason citations_test.go excludes itself: it necessarily spells the
// pattern it searches for (its allow-list entries below), and that data is
// not a citation.
const styleCitationsGuardFile = "stylecitations_test.go"

// allowedBadStyleCitations is the reviewed state of every bare-numeric STYLE
// citation in the module: the sites #540 describes, confirmed by direct
// inspection of the cited comment. Counts are per file and per token, not per
// line, so ordinary editing above a citation does not trip the guard while a
// new or copied bad citation does. Every token here is bare digits; a
// letter-shaped one is rejected by TestStyleCitationsNameARealRule's sibling
// check rather than admitted here.
//
// #382's entries — `T7` ×29, `T8`, `D6`, `L6` across 19 files — are gone: that
// issue ruled each of them onto the rule docs/STYLE.md already states (T1, T2,
// T5, D3) and renumbered every site.
var allowedBadStyleCitations = []styleCitationAllowance{
	{file: "conformance/datatypes.go", token: "10", count: 1},
	{file: "xsd/resolve.go", token: "8", count: 2},
}

// The vacuity floors, sitting safely below the counts at the time of
// writing (273 .go files, 602 STYLE citation tokens, 23 defined rule IDs) so
// ordinary editing does not trip them.
const (
	minStyleCitationsFound = 400
	minStyleRuleIDs        = 15
	styleDocRelPath        = "docs/STYLE.md"
)

// styleCitationRe matches one `STYLE <body>` citation and captures the body.
// The optional `// ` between STYLE and the body is not a stray comment
// marker: `go tool commentwrap` reflows prose, and a citation that lands at
// the end of a wrapped line continues as `// <body>` on the next — the
// citation is one token stream split by a comment continuation, not two.
// `<body>` itself may slash-join several tokens (`D3/T4`, `T5/8`); each
// token is validated independently.
var styleCitationRe = regexp.MustCompile(`STYLE (?:// )?([A-Za-z0-9]+(?:/[A-Za-z0-9]+)*)`)

// styleIDShapeRe matches the SHAPE of a docs/STYLE.md ID (a letter, then
// digits, then an optional lowercase suffix like P3a's `a`), used to confirm
// the heading parse below yields IDs and not prose.
var styleIDShapeRe = regexp.MustCompile(`^[A-Z][0-9]+[a-z]?$`)

// styleRuleHeadingRe matches one rule-ID heading in docs/STYLE.md, whose
// rules are `**ID. Title.** …` at the start of a line.
var styleRuleHeadingRe = regexp.MustCompile(`^\*\*([A-Z][0-9]+[a-z]?)\. `)

// styleCitationSite is one citation token, located for the failure message
// that has to send a reader to the exact comment. token is a single ID after
// any slash-join has been split, never the raw multi-ID citation body.
type styleCitationSite struct {
	file  string // slash-separated, relative to the module root
	line  int    // 1-based; the line STYLE appears on, not the token's own
	token string
}

// styleCitationAllowance is one reviewed allow-list entry: file cites token
// exactly count times, and token is bare digits — a CLAUDE.md headline
// position docs/STYLE.md cannot define.
type styleCitationAllowance struct {
	file  string
	token string
	count int
}

// styleFileToken keys the per-file, per-token citation counts. It is a
// lookup index only — reporting iterates the sites and the allow-list, both
// of which are ordered.
type styleFileToken struct {
	file  string
	token string
}

// TestStyleCitationsNameARealRule is the BOUND half: every citation token that
// is not bare digits names a rule docs/STYLE.md defines. There is no allow-list
// — #382 settled that an ID-shaped token naming no rule always meant a rule the
// document already states.
func TestStyleCitationsNameARealRule(t *testing.T) {
	root := moduleRootDir(t)
	defined := styleRuleIDs(t, root)
	for _, s := range styleCitations(t, root) {
		if defined[s.token] || isBareNumericToken(s.token) {
			continue
		}
		t.Errorf("%s:%d: cites STYLE %s, which %s does not define — an invented ID "+
			"(#382's defect) or a typo. Read the comment, pick the rule it means, and "+
			"cite that ID; do not add the rule to %s to make this pass",
			s.file, s.line, s.token, styleDocRelPath, styleDocRelPath)
	}
}

// TestPositionalStyleCitationsAreAllowListed is the ALLOW-LIST half: every
// bare-numeric citation — a position in CLAUDE.md's headline list rather than
// an ID — was reviewed onto allowedBadStyleCitations, and no reviewed entry has
// gone stale.
func TestPositionalStyleCitationsAreAllowListed(t *testing.T) {
	root := moduleRootDir(t)

	allowed := make(map[styleFileToken]int, len(allowedBadStyleCitations))
	for _, a := range allowedBadStyleCitations {
		if !isBareNumericToken(a.token) {
			t.Errorf("allowedBadStyleCitations entry {file: %q, token: %q} is not a bare "+
				"number — this list pins #540's positional citations only, and a "+
				"letter-shaped ID answers to TestStyleCitationsNameARealRule instead",
				a.file, a.token)
			continue
		}
		allowed[styleFileToken{file: a.file, token: a.token}] = a.count
	}

	found := make(map[styleFileToken]int, len(allowedBadStyleCitations))
	for _, s := range styleCitations(t, root) {
		if !isBareNumericToken(s.token) {
			continue
		}
		key := styleFileToken{file: s.file, token: s.token}
		found[key]++
		if _, ok := allowed[key]; ok {
			continue
		}
		t.Errorf("%s:%d: cites STYLE %s, a bare number — that is a position in "+
			"CLAUDE.md's headline list, not a citable ID (see %s's \"Citing rules\"). "+
			"Confirm the intended letter ID, then add {file: %q, token: %q, count: N} "+
			"to allowedBadStyleCitations",
			s.file, s.line, s.token, styleDocRelPath, s.file, s.token)
	}

	for _, a := range allowedBadStyleCitations {
		got := found[styleFileToken{file: a.file, token: a.token}]
		if got == a.count {
			continue
		}
		t.Errorf("%s: allow-listed for %d citations of STYLE %s, found %d — "+
			"re-check the citations, then update allowedBadStyleCitations",
			a.file, a.count, a.token, got)
	}
}

// isBareNumericToken reports whether token is all digits — a CLAUDE.md
// headline position rather than a docs/STYLE.md letter ID.
func isBareNumericToken(token string) bool {
	return strings.IndexFunc(token, func(r rune) bool { return r < '0' || r > '9' }) == -1
}

// styleCitations returns every `STYLE <token>` citation in the module's Go
// sources, test files included, one entry per slash-split token, in walk
// order (lexical by path, then by line) so failures report identically on
// every run. A citation split across a `go tool commentwrap`-reflowed line
// break is read as one citation, not silently dropped (see styleCitationRe).
func styleCitations(t *testing.T, root string) []styleCitationSite {
	t.Helper()
	var out []styleCitationSite
	for _, rel := range goSourceFiles(t, root) {
		if rel == styleCitationsGuardFile {
			continue
		}
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		lines := strings.Split(string(b), "\n")

		// joined re-glues the file's lines with a single space so a citation
		// split by a reflowed comment break (`...(STYLE\n// T7): ...`) reads
		// as contiguous text; lineOf maps each byte of joined back to the
		// 1-based original line, so a match's reported location is still the
		// line STYLE itself starts on.
		var joinedB strings.Builder
		lineOf := make([]int, 0, len(b)+len(lines))
		for i, line := range lines {
			joinedB.WriteString(line)
			for range len(line) {
				lineOf = append(lineOf, i+1)
			}
			joinedB.WriteByte(' ')
			lineOf = append(lineOf, i+1)
		}
		joined := joinedB.String()

		for _, m := range styleCitationRe.FindAllStringSubmatchIndex(joined, -1) {
			line := lineOf[m[0]]
			body := joined[m[2]:m[3]]
			for _, tok := range strings.Split(body, "/") {
				out = append(out, styleCitationSite{file: rel, line: line, token: tok})
			}
		}
	}
	if len(out) < minStyleCitationsFound {
		t.Fatalf("found %d STYLE citation tokens, expected at least %d — the scan is "+
			"not matching comments it used to", len(out), minStyleCitationsFound)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].file != out[j].file {
			return out[i].file < out[j].file
		}
		if out[i].line != out[j].line {
			return out[i].line < out[j].line
		}
		return out[i].token < out[j].token
	})
	return out
}

// styleRuleIDs returns the set of rule IDs docs/STYLE.md actually defines,
// parsed from its `**ID. Title.**` headings — the property that makes "ID is
// in this set" mean "ID names a real rule". A token merely SHAPED like an ID
// but absent from this set is exactly #382's mistake.
func styleRuleIDs(t *testing.T, root string) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(styleDocRelPath)))
	if err != nil {
		t.Fatalf("reading %s: %v", styleDocRelPath, err)
	}
	ids := make(map[string]bool)
	for i, line := range strings.Split(string(b), "\n") {
		m := styleRuleHeadingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if !styleIDShapeRe.MatchString(m[1]) {
			t.Fatalf("%s:%d: heading ID %q does not match the expected shape", styleDocRelPath, i+1, m[1])
		}
		ids[m[1]] = true
	}
	if len(ids) < minStyleRuleIDs {
		t.Fatalf("%s defines %d rule IDs, expected at least %d — the parse is not "+
			"matching heading lines", styleDocRelPath, len(ids), minStyleRuleIDs)
	}
	return ids
}
