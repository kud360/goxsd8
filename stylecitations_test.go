// This file guards, module-wide, the STYLE-ID citation convention documented
// in docs/STYLE.md: "A style rule is cited by its letter ID from this file
// (`STYLE D4`, `STYLE T2`) — never by a position in CLAUDE.md's 'Style
// headlines' list, which is a summary and carries no citable IDs." Two open
// issues describe the same defect at scale, neither caught by anything today:
//
//   - #382 — sites cite a letter ID docs/STYLE.md does not define: `T7`,
//     `T8`, `D6`, `L6`, plus a malformed `T5/8` (one real ID, one bare
//     number, slash-joined as if both were IDs).
//   - #540 — sites cite a bare-numeric "STYLE N", which is a position in
//     CLAUDE.md's headline list, not a citable ID; CLAUDE.md and
//     docs/STYLE.md both say so explicitly.
//
// Both defects are the same shape as the one citations_test.go already
// guards for PRINCIPLES: a number or token that reads as a citation but does
// not name the thing its author meant. This file is that guard's sibling for
// STYLE, reusing citations_test.go's moduleRootDir and goSourceFiles rather
// than re-implementing the walk (STYLE T4).
//
// One half, allow-listed rather than unconditional:
//
//   - Every `STYLE <token>` citation's token(s) — slash-joined multi-ID
//     citations are split — must be an ID docs/STYLE.md actually defines.
//     A token that is bare digits is reported as a CLAUDE.md headline
//     position (#540's shape); any other undefined token is reported as an
//     unknown letter ID (#382's shape).
//   - Unlike the PRINCIPLES guard's bound check, this is not an unconditional
//     fail: #382 and #540 are ALREADY 36 real, unfixed violations across 21
//     files, and this guard must land without regressing the gate. Every
//     violation observed at authoring time is pinned in
//     allowedBadStyleCitations, by file, token and count — exactly
//     allowedCollisionCitations' shape. A NEW or COPIED bad citation cannot
//     land without editing that list, and editing it is the review this file
//     exists to force. The list shrinking to empty is what closes #382/#540;
//     it is not this file's job to shrink it.
//
// What this does NOT do is decide whether a citation is TOPICALLY right, or
// whether a real ID's rule actually applies to the code it decorates; no
// mechanical check can. It makes the undefined-ID and positional-citation
// mistakes loud instead of silent.
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

// allowedBadStyleCitations is the reviewed state of every undefined or
// positional STYLE citation in the module as of #636's authoring: every site
// #382 and #540 describe, confirmed by direct inspection of the cited
// comment. Counts are per file and per token, not per line, so ordinary
// editing above a citation does not trip the guard while a new or copied bad
// citation does.
var allowedBadStyleCitations = []styleCitationAllowance{
	{file: "builtin/strict/datetime.go", token: "D6", count: 1},
	{file: "builtin/strict/datetime.go", token: "T7", count: 1},
	{file: "builtin/strict/duration.go", token: "T7", count: 1},
	{file: "builtin/strict/precisiondecimal.go", token: "T7", count: 1},
	{file: "conformance/datatypes.go", token: "10", count: 1},
	{file: "conformance/expectations.go", token: "T7", count: 1},
	{file: "conformance/runner.go", token: "T7", count: 1},
	{file: "loader/resolver.go", token: "L6", count: 1},
	{file: "parser/report.go", token: "T7", count: 2},
	{file: "xsd/attributedeclaration.go", token: "T7", count: 1},
	{file: "xsd/attributeuse.go", token: "T7", count: 1},
	{file: "xsd/closedsets.go", token: "T7", count: 1},
	{file: "xsd/complextype.go", token: "T7", count: 3},
	{file: "xsd/defaultbinding.go", token: "T7", count: 1},
	{file: "xsd/defaultbinding.go", token: "T8", count: 1},
	{file: "xsd/elementdeclaration.go", token: "T7", count: 1},
	{file: "xsd/namespaceconstraint.go", token: "T7", count: 1},
	{file: "xsd/namespaceconstraint.go", token: "T8", count: 1},
	{file: "xsd/resolve.go", token: "8", count: 2},
	{file: "xsd/schema.go", token: "T7", count: 3},
	{file: "xsd/simpletype.go", token: "T7", count: 3},
	// Landed by #636 while this guard was in review, so it grandfathers in
	// rather than being caught before merge — the one entry here that is
	// NEW debt, not inherited. Its `STYLE T2/T7` decorates a sealed sum,
	// and T2 (capabilities are interfaces, not type switches) is the rule
	// that governs sealed sums; T7 looks like CLAUDE.md's headline 7, the
	// positional-citation defect #540 names. Left uncorrected on purpose:
	// 30 other sites cite T7, and deciding what they all become is #382's
	// question, not this file's.
	{file: "xsd/simpletyperef.go", token: "T7", count: 1},
	{file: "xsd/term.go", token: "T7", count: 2},
	{file: "xsd/typedefinition.go", token: "T7", count: 4},
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

// styleIDShapeRe matches the SHAPE of a real docs/STYLE.md ID (a letter, then
// digits, then an optional lowercase suffix like P3a's `a`) without regard to
// whether docs/STYLE.md actually defines it — used only to choose which of
// the two failure messages a token gets.
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
// exactly count times, and docs/STYLE.md does not define token.
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

// TestBadStyleCitationsAreAllowListed is the guard's one test: every STYLE
// citation token that docs/STYLE.md does not define was reviewed onto
// allowedBadStyleCitations, and no reviewed entry has gone stale.
func TestBadStyleCitationsAreAllowListed(t *testing.T) {
	root := moduleRootDir(t)
	defined := styleRuleIDs(t, root)

	allowed := make(map[styleFileToken]int, len(allowedBadStyleCitations))
	for _, a := range allowedBadStyleCitations {
		allowed[styleFileToken{file: a.file, token: a.token}] = a.count
	}

	found := make(map[styleFileToken]int, len(allowedBadStyleCitations))
	for _, s := range styleCitations(t, root) {
		if defined[s.token] {
			continue
		}
		key := styleFileToken{file: s.file, token: s.token}
		found[key]++
		if _, ok := allowed[key]; ok {
			continue
		}
		if isBareNumericToken(s.token) {
			t.Errorf("%s:%d: cites STYLE %s, a bare number — that is a position in "+
				"CLAUDE.md's headline list, not a citable ID (see %s's \"Citing rules\"). "+
				"Confirm the intended letter ID, then add {file: %q, token: %q, count: N} "+
				"to allowedBadStyleCitations",
				s.file, s.line, s.token, styleDocRelPath, s.file, s.token)
			continue
		}
		t.Errorf("%s:%d: cites STYLE %s, which %s does not define. Confirm the intended "+
			"ID, then add {file: %q, token: %q, count: N} to allowedBadStyleCitations",
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
// (styleIDShapeRe) that is absent from this set is exactly #382's mistake.
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
