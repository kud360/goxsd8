// Command gapaudit reconciles the `// GAP(<area>): <text>` markers left in
// the source tree against the `kind/gap` GitHub issues that are supposed to
// track them.
//
// STYLE P3 requires every deliberate incompleteness — a fail-open XPath
// construct, an unimplemented validation branch, anything else recorded with
// a `GAP(` marker — to name a still-open tracking issue, so debt is
// greppable and ratchetable rather than fading into the prose around it.
// Nothing machine-checked enforces that promise: docs/WORKFLOW.md leaves it
// to the cartographer's backlog pass, which today means running `grep -rn
// "GAP("` and eyeballing the result against the open `kind/gap` issue list —
// a set difference, done by hand, over a list that only grows. PRINCIPLES 27
// says repetitive deterministic work like that becomes a tool; this is that
// tool.
//
// # What it reports
//
// Three groups, each sorted for byte-identical reruns (STYLE D1):
//
//  1. Markers that cite no OPEN tracking issue — the leak the rule exists
//     to prevent. This includes a marker that only cites a CLOSED issue:
//     STYLE P3 says a marker pointing at a closed issue is a dead end, so it
//     is reported the same as an uncited one. Each row carries its reason,
//     and only an unannotated row is a candidate for filing: "candidate
//     owner" names an OPEN issue the marker resembles but does not cite, so
//     the fix is writing that number into the marker rather than filing
//     another, "dead end" names a CLOSED issue the marker actually cites and
//     the landing that closed it should have repointed, "resemblance" names
//     a CLOSED one it merely reads like, and an unresolved-citation line
//     names a number that is no issue in the fed list at all.
//  2. OPEN `kind/gap` issues no marker in the tree cites — a stale tracker:
//     the gap was closed in code without closing the issue that tracked it.
//     A marker that matches by file path or by phrase alone is printed under
//     the row rather than retiring it.
//  3. A per-area census of marker counts.
//
// # Matching: one certain signal, then two heuristic ones
//
// A marker that cites an issue number in its own text — STYLE P3's "names
// that issue in the text" — is matched to that issue and to no other. That
// is a citation, not a resemblance, and [matches] reports it as its own
// [matchKind] so the report can say "cites" where it means cites (#852).
//
// The two fallbacks are heuristic: the marker's file path, or a run of
// [minPhraseWords] of its words, appearing in the issue's title or body. A
// marker with no match at all is reported as "no tracking issue found",
// never as "untracked" — the matcher has no way to rule out an issue that
// describes the same gap in different words, so the reader is the one who
// gets to call it truly untracked.
//
// Both directions retire a row on a citation and on nothing else. The two
// fallbacks are reported, never charged: an issue names a file path as
// readily to EXCLUDE the site as to own it, and a five-word run of this
// repo's own boilerplate collides outright (#852, #1060). Each fallback is
// printed beside the row it did not retire, so the reader judges the
// resemblance instead of re-deriving it.
//
// # Usage
//
//	go run ./tools/gapaudit [dir] < issues.json
//	gh issue list --state all --limit 5000 --json number,title,state,body,labels | go run ./tools/gapaudit
//
// dir defaults to "." (the whole tree). Empty or absent stdin runs in
// marker-inventory-only mode: groups 1 and 2 are skipped (there is nothing
// to reconcile against) and the report says so.
//
// Feed the whole repository, not a `kind/gap` slice of it, and carry
// `labels` on every row: this tool makes the `kind/gap` selection itself,
// for group 2 alone (#1062). Nothing constrains a gap's owner to that one
// label, so a marker citing a `kind/feature` or `kind/story` owner resolves
// against it on the wide input and would only have read as an absent
// citation on the narrow one. What remains unresolvable on a wide input is a
// number naming no issue at all — a mistyped citation, reported in group 1
// beside the marker. A labelless input selects nothing into group 2, and the
// report says so rather than reading as "no tracker is stale".
//
// It exits 0 on a clean run (regardless of what it finds — the findings are
// the report, not a failure) and 2 on an operational error: an unreadable
// file, a source file this tool's scanner cannot tokenize, or stdin that
// does not decode as the documented JSON shape.
package main

import (
	"encoding/json"
	"fmt"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	markers, err := scanTree(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gapaudit: %v\n", err)
		os.Exit(2)
	}

	issues, haveIssues, err := readIssues(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gapaudit: %v\n", err)
		os.Exit(2)
	}

	rep := reconcile(markers, issues, haveIssues)
	if err := printReport(os.Stdout, rep); err != nil {
		fmt.Fprintf(os.Stderr, "gapaudit: %v\n", err)
		os.Exit(2)
	}
}

// scanTree walks root for `.go` files and extracts every marker from each,
// in a deterministic (path-sorted) order. It is the only filesystem-touching
// step in the marker pipeline; extraction itself ([extractMarkers]) is pure.
func scanTree(root string) ([]marker, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != root && skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", root, err)
	}
	sort.Strings(paths)

	var markers []marker
	for _, path := range paths {
		src, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = path
		}
		found, err := extractMarkers(filepath.ToSlash(rel), src)
		if err != nil {
			return nil, fmt.Errorf("scanning %s: %w", path, err)
		}
		markers = append(markers, found...)
	}
	return markers, nil
}

// skipDir reports whether name is a directory whose Go files, if any, are
// not this repo's hand-written source: the git store, vendored test suites,
// and hidden directories.
func skipDir(name string) bool {
	return name == "vendor" || name == "testdata" || name == "xsdtests" ||
		(len(name) > 1 && name[0] == '.')
}

// readIssues decodes the `gh issue list ... --json
// number,title,state,body,labels` shape from r. An empty or absent stdin is
// not an error: haveIssues comes back false, and the caller runs in
// marker-inventory-only mode.
func readIssues(r io.Reader) (issues []issue, haveIssues bool, err error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, false, fmt.Errorf("reading issue list from stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, false, nil
	}
	if err := json.Unmarshal(data, &issues); err != nil {
		return nil, false, fmt.Errorf("decoding issue list JSON: %w", err)
	}
	return issues, true, nil
}

// marker is one `GAP(<area>): <text>` site found in a comment. Text is the
// marker's own paragraph: the words after the colon on the marker's line,
// plus every immediately following comment line that continues the same
// prose (docs/WORKFLOW.md prefers citing this text over a line number,
// because later unrelated edits move line numbers but not prose).
//
// Text is also where the marker's issue citation lives, read back out by
// [citations], so [paragraph]'s boundary rules decide what counts as cited:
// a `#N` past a blank comment line, a bullet, a heading, or the next marker
// head belongs to a different paragraph and is not this marker's owner. A
// `#N` that merely OPENS a wrapped line is this marker's own (#1117).
type marker struct {
	Area string
	File string
	Line int
	Text string
}

// markerPattern finds a `GAP(<area>):` marker head. The colon immediately
// after the closing paren is load-bearing: it is what tells a genuine marker
// apart from a comment merely mentioning an earlier one ("see the GAP(value)
// gate above"), which this pattern does not match.
var markerPattern = regexp.MustCompile(`GAP\(([A-Za-z][A-Za-z0-9_]*)\):\s*`)

// citationPattern finds an issue citation inside a marker's text. Bare
// `#<digits>` is the only form this repo writes one in — CLAUDE.md's commit
// format and STYLE P3 both spell it that way, and spec clauses are cited
// with `§` and rule IDs, never with a hash.
//
// It is not a perfect discriminator: this repo's comments do contain a
// `#<digits>` that is not an issue ("acceptance #2", conformance/runner.go),
// and nothing but its position keeps such a token out of a marker's
// paragraph. One inside a marker would tie that marker to whatever issue
// carries the number, so a marker's own prose is the place to fix it —
// there is no signal here to tell the two apart.
var citationPattern = regexp.MustCompile(`#(\d+)`)

// citations returns every issue number text cites, in first-appearance
// order with repeats dropped, so a marker naming one owner twice and a
// marker naming two owners are both reported as written. It is derived
// from the marker's text on demand rather than stored on [marker]: one
// fact, one encoding (STYLE D3), and the whole audit runs a few thousand
// of these.
func citations(text string) []int {
	var out []int
	seen := make(map[int]bool)
	for _, m := range citationPattern.FindAllStringSubmatch(text, -1) {
		// The pattern guarantees digits, so Atoi fails only on a run too long
		// to be an issue number. Skipping is the whole disposition: such a
		// token is not a citation, and there is no caller to report it to.
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

// bulletPattern recognizes a comment line opening a new list item — a
// numbered entry ("1. ") or a dash/star bullet — which ends the current
// marker's paragraph rather than continuing it, the same way a blank comment
// line does.
var bulletPattern = regexp.MustCompile(`^(\d+\.|[-*])\s`)

// extractMarkers scans one file's source for GAP markers. It is pure: given
// the same path and src it always returns the same markers, with no
// filesystem or network access, so tests can hand it fixture bytes directly.
// That is also what lets STYLE D1's determinism be tested without a tree on
// disk.
//
// Only text inside actual `//` and `/* */` comments is considered — a
// go/scanner token stream distinguishes those from string literals and code,
// so a source file whose own string constants happen to contain the letters
// "GAP(" is not mistaken for a marker site.
func extractMarkers(path string, src []byte) ([]marker, error) {
	lines, err := commentLines(path, src)
	if err != nil {
		return nil, err
	}

	var found []marker
	for i, line := range lines {
		if quotesAComment(line.text) {
			continue
		}
		for _, loc := range markerPattern.FindAllStringSubmatchIndex(line.text, -1) {
			area := line.text[loc[2]:loc[3]]
			text := paragraph(lines, i, loc[1])
			found = append(found, marker{Area: area, File: path, Line: line.num, Text: text})
		}
	}
	return found, nil
}

// quotesAComment reports whether a comment line is itself quoting comment
// syntax rather than being that comment — its content starts with another
// `//`, the shape a doc comment uses to show the marker convention in an
// indented example. Such a line is documentation, not a fail-open site, and
// counting it would report the same phantom every audit. Real markers are
// not excluded by their angle brackets: XSD element names put `<simpleType>`
// and `<xs:override>` inside genuine marker prose.
func quotesAComment(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "//")
}

// hashEndsParagraph reports whether a comment line's leading `#` opens
// something other than an issue citation — a Markdown heading (`# Notes`,
// `## Group 1`), a bare `#`, a stray `#word` — and so ends the current
// marker's paragraph. A `#` immediately followed by digits does not: that is
// a citation as [citationPattern] reads one, and `go tool commentwrap`
// reflows marker prose to 79 columns without regard to which token starts a
// line, so an owner's number lands at a line start on its own schedule.
// Dropping it there scored a correctly-owned marker as citing nothing
// (#1117).
func hashEndsParagraph(text string) bool {
	if !strings.HasPrefix(text, "#") {
		return false
	}
	loc := citationPattern.FindStringIndex(text)
	return loc == nil || loc[0] != 0
}

// paragraph assembles a marker's trailing text: the remainder of its own
// line from headEnd, plus every contiguous following comment line that is
// neither blank, a new bullet, a new marker, nor a heading
// ([hashEndsParagraph]) — the same paragraph-boundary rules a human reading
// the comment would apply.
func paragraph(lines []commentLine, at, headEnd int) string {
	var parts []string
	if first := strings.TrimSpace(lines[at].text[headEnd:]); first != "" {
		parts = append(parts, first)
	}

	prevNum := lines[at].num
	for j := at + 1; j < len(lines); j++ {
		if lines[j].num != prevNum+1 {
			break
		}
		t := strings.TrimSpace(lines[j].text)
		if t == "" || hashEndsParagraph(t) ||
			bulletPattern.MatchString(t) || markerPattern.MatchString(t) {
			break
		}
		parts = append(parts, t)
		prevNum = lines[j].num
	}
	return strings.Join(parts, " ")
}

// commentLine is one physical line of comment text — the content after `//`,
// or one line of a `/* ... */` block comment — with the source line number
// it appears on.
type commentLine struct {
	num  int
	text string
}

// commentLines tokenizes src and returns every comment line in source order.
// It uses go/scanner directly (rather than go/parser) because a marker is
// meaningful line by line, not as an AST comment group: this repo wraps GAP
// prose across several `//` lines, and a plain AST CommentGroup does not
// expose the per-line blank/bullet boundaries [paragraph] needs.
func commentLines(path string, src []byte) ([]commentLine, error) {
	fset := token.NewFileSet()
	file := fset.AddFile(path, fset.Base(), len(src))

	var errs []error
	var s scanner.Scanner
	s.Init(file, src, func(pos token.Position, msg string) {
		errs = append(errs, fmt.Errorf("%s: %s", pos, msg))
	}, scanner.ScanComments)

	var lines []commentLine
	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if tok != token.COMMENT {
			continue
		}
		startLine := file.Position(pos).Line
		if strings.HasPrefix(lit, "//") {
			lines = append(lines, commentLine{num: startLine, text: lit[2:]})
			continue
		}
		// A block comment's lit runs from "/*" to "*/" inclusive and may span
		// several physical lines; each inner line needs its own line number
		// for the same contiguity check a `//` run gets.
		inner := lit[2 : len(lit)-2]
		for i, part := range strings.Split(inner, "\n") {
			lines = append(lines, commentLine{num: startLine + i, text: part})
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("tokenizing %s: %w", path, errs[0])
	}
	return lines, nil
}

// issue is one row of `gh issue list --json number,title,state,body,labels`:
// the fields this tool needs to decide whether a marker is tracked. State is
// "OPEN" or "CLOSED" as gh emits it; matching compares it case-insensitively
// rather than adding a second encoding of the same fact.
//
// Labels is what lets the input be the whole repository rather than the
// `kind/gap` slice of it: [reconcile] does the [gapLabel] selection itself,
// so group 1 can resolve a citation whose owner carries some other label
// (#1062).
type issue struct {
	Number int     `json:"number"`
	Title  string  `json:"title"`
	State  string  `json:"state"`
	Body   string  `json:"body"`
	Labels []label `json:"labels"`
}

// label is one entry of an issue's `labels` array. Only the name is read;
// gh emits id, description and color beside it and they decode away.
//
// tools/wipsurvey declares the same one field as `ghLabel`, and the two
// stay separate: both tools are `package main`, so sharing the shape means
// a new package under tools/ with its own surface, which is a refactor of
// the survey pipeline rather than part of widening this one's input (T4).
type label struct {
	Name string `json:"name"`
}

// gapLabel is the label docs/WORKFLOW.md puts on the issue that tracks a
// marked gap. It selects group 2 and nothing else: group 1 weighs a marker
// against every issue in the input, whatever it is labeled.
const gapLabel = "kind/gap"

func (iss issue) open() bool {
	return strings.EqualFold(iss.State, "OPEN")
}

func (iss issue) hasLabel(name string) bool {
	for _, l := range iss.Labels {
		if l.Name == name {
			return true
		}
	}
	return false
}

// areaCount is one row of the per-area marker census.
type areaCount struct {
	Area  string
	Count int
}

// issueMatch is an issue that matched a group-1 marker without retiring it,
// with the signal that tied them. The kind and the issue's state together
// separate the three annotations printed under a group-1 row: a CLOSED issue
// the marker cites is STYLE P3's dead end, a CLOSED one it merely reads like
// is a resemblance, and an OPEN one it reads like is a candidate owner whose
// number belongs in the marker.
type issueMatch struct {
	Issue issue
	Kind  matchKind
}

// untrackedMarker is a group-1 finding: a marker no OPEN issue retires.
// Matches records every issue that DID match without retiring it, for
// context — the reader deciding what to do next needs to know whether a
// plausible owner is already open (write its number into the marker) or
// already closed (STYLE P3's dead end) before filing another. Unresolved
// records issue numbers the marker cites that the fed list does not contain
// at all: the marker names an owner that does not exist, which is a
// different finding from naming no owner.
type untrackedMarker struct {
	Marker     marker
	Unresolved []int
	Matches    []issueMatch
}

// markerMatch is a marker that matched a group-2 issue without retiring it,
// with the signal that tied them.
type markerMatch struct {
	Marker marker
	Kind   matchKind
}

// staleTracker is a group-2 finding: an OPEN issue no marker cites. Weak
// names every marker that matched it by file path or phrase — too weak to
// retire a tracker, and printed rather than dropped so the reader sees the
// collision instead of re-deriving it (#852, #1060).
type staleTracker struct {
	Issue issue
	Weak  []markerMatch
}

// report is gapaudit's full output: the three groups described in the
// package doc, plus whether an issue list was supplied at all (haveIssues
// false means groups 1 and 2 are meaningless and were not computed).
//
// Labeled records whether any row of that list carried a label at all. An
// input reshaped without a `labels` field — the shape this tool documented
// before #1062 — selects nothing into group 2, and an empty group 2 then
// means "no tracker was seen" rather than "no tracker is stale". The
// report says which.
type report struct {
	HaveIssues bool
	Labeled    bool
	Untracked  []untrackedMarker
	Stale      []staleTracker
	Census     []areaCount
}

// anyLabeled reports whether any fed issue carries a label, which is how a
// pre-#1062 labelless input is told apart from a repository whose issues
// happen to carry no `kind/gap`.
func anyLabeled(issues []issue) bool {
	for _, iss := range issues {
		if len(iss.Labels) > 0 {
			return true
		}
	}
	return false
}

// minPhraseWords is how many consecutive words of a marker's text must
// appear, in order, inside an issue's title+body before the marker counts as
// matched by phrase rather than by file path. Below this a run of words is
// common enough (English filler, repeated spec vocabulary like "content
// model") to match issues that have nothing to do with the marker.
//
// Five is not enough to make a match distinctive, only enough to make it
// worth a look: this repo's own idioms run that long, and "CLAUDE.md's one
// rule" tied a parser marker to a conformance-lane tracker sharing nothing
// else (#852). Raising the number is not the answer — it would silence real
// matches to buy back a heuristic's certainty it never had — so the weight
// of a phrase match lives in [matchKind.retires] instead, which charges it
// nothing and prints it beside the row it did not retire.
const minPhraseWords = 5

// reconcile computes the three-group report from a marker inventory and an
// optional issue list. It is pure: no I/O, so tests exercise it directly
// with literal marker/issue slices. When haveIssues is false only the
// census (group 3) is populated, since there is nothing to reconcile markers
// against.
func reconcile(markers []marker, issues []issue, haveIssues bool) report {
	rep := report{HaveIssues: haveIssues, Labeled: anyLabeled(issues), Census: census(markers)}
	if !haveIssues {
		return rep
	}

	sorted := make([]marker, len(markers))
	copy(sorted, markers)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].File != sorted[j].File {
			return sorted[i].File < sorted[j].File
		}
		if sorted[i].Line != sorted[j].Line {
			return sorted[i].Line < sorted[j].Line
		}
		return sorted[i].Area < sorted[j].Area
	})

	for _, m := range sorted {
		if anyOpenMatch(m, issues) {
			continue
		}
		rep.Untracked = append(rep.Untracked, untrackedMarker{
			Marker:     m,
			Unresolved: unresolvedCitations(m, issues),
			Matches:    weakIssueMatches(m, issues),
		})
	}

	for _, iss := range issues {
		if !iss.open() || !iss.hasLabel(gapLabel) {
			continue
		}
		retired, weak := trackerEvidence(iss, sorted)
		if retired {
			continue
		}
		rep.Stale = append(rep.Stale, staleTracker{Issue: iss, Weak: weak})
	}
	sort.Slice(rep.Stale, func(i, j int) bool { return rep.Stale[i].Issue.Number < rep.Stale[j].Issue.Number })

	return rep
}

// census counts markers per area, sorted by area name so a rerun over an
// unchanged tree reports byte-identical output (STYLE D1) — the counts are
// aggregated through a map internally, but nothing ranges over that map into
// the output slice (STYLE D2).
func census(markers []marker) []areaCount {
	counts := make(map[string]int)
	for _, m := range markers {
		counts[m.Area]++
	}
	areas := make([]string, 0, len(counts))
	for area := range counts {
		areas = append(areas, area)
	}
	sort.Strings(areas)

	out := make([]areaCount, 0, len(areas))
	for _, area := range areas {
		out = append(out, areaCount{Area: area, Count: counts[area]})
	}
	return out
}

// anyOpenMatch reports whether any OPEN issue keeps m out of group 1. The
// bar is [matchKind.retires]: a citation, nothing weaker. A file mention
// does not qualify, because it hides the exact row STYLE P3 exists to
// surface — a marker naming no issue — and one open issue naming a busy file
// shadows every uncited marker in it: #853, #414, #1099 and #1102 name
// validate/cvcid.go, xsd/attributeusefold.go and parser/produce_complex.go
// in their bodies, and between them hid the uncited markers those files
// carry (#1060).
func anyOpenMatch(m marker, issues []issue) bool {
	for _, iss := range issues {
		if iss.open() && matches(m, iss).retires() {
			return true
		}
	}
	return false
}

// trackerEvidence weighs every marker against one OPEN issue: retired is
// true once a marker CITES it (the bar is [matchKind.retires]), and weak
// collects every match too weak for that. A file mention does not retire,
// because an issue names a path as readily to EXCLUDE the site as to own it:
// #972's body names parser/produce_complex.go to rule that the check must
// NOT go there, and that mention retired #972's own row on the strength of
// markers in that file it disclaims — one of which says in its own text
// "Unowned: no issue tracks it yet" (#1060). weak keeps the order
// markers arrives in, which [reconcile] has already sorted, so the report
// inherits its determinism (STYLE D1) rather than re-establishing it here.
func trackerEvidence(iss issue, markers []marker) (retired bool, weak []markerMatch) {
	for _, m := range markers {
		k := matches(m, iss)
		if k.retires() {
			return true, nil
		}
		if k.found() {
			weak = append(weak, markerMatch{Marker: m, Kind: k})
		}
	}
	return false, weak
}

// unresolvedCitations returns the issue numbers m cites that appear nowhere
// in the fed list, open or closed. On the whole-repository input this tool
// now documents, such a number names no issue at all — a mistyped citation,
// or a pull-request number the reshape dropped — so the report names the
// number rather than calling the marker untracked, and the number itself is
// the thing to fix (#1062).
func unresolvedCitations(m marker, issues []issue) []int {
	var out []int
	for _, n := range citations(m.Text) {
		known := false
		for _, iss := range issues {
			if iss.Number == n {
				known = true
				break
			}
		}
		if !known {
			out = append(out, n)
		}
	}
	return out
}

// weakIssueMatches returns every issue that matched m without retiring it,
// in issue-number order (STYLE D1). Its callers reach it only for a marker
// already in group 1, so an OPEN issue here is by construction one m does
// not cite.
func weakIssueMatches(m marker, issues []issue) []issueMatch {
	var out []issueMatch
	for _, iss := range issues {
		k := matches(m, iss)
		if !k.found() || (iss.open() && k.retires()) {
			continue
		}
		out = append(out, issueMatch{Issue: iss, Kind: k})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Issue.Number < out[j].Issue.Number })
	return out
}

// matchKind is the closed set of signals [matches] can find, ordered by
// strength: a citation is what STYLE P3 asks a marker to carry, a file path
// says only that the issue mentions the file — which it does as readily to
// EXCLUDE the site as to own it (#1060) — and a phrase run is a resemblance.
// The zero value is [matchNone], so an unset kind is "no match" rather than
// a spurious one.
type matchKind int

const (
	matchNone matchKind = iota
	matchPhrase
	matchFile
	matchCited
)

// matches is the decision at the center of this tool: which signal, if any,
// ties marker m to issue iss. It tries them strongest first and reports the
// first that fires, so a caller can weigh a citation differently from a
// resemblance instead of taking every match on the same word.
//
// Only [matchCited] is proof. A file gets renamed and a marker's prose gets
// rephrased in the issue that files it, so callers must treat [matchNone] as
// "no match found", never as "definitely untracked" (see the package doc).
func matches(m marker, iss issue) matchKind {
	for _, n := range citations(m.Text) {
		if n == iss.Number {
			return matchCited
		}
	}
	haystack := strings.ToLower(iss.Title + "\n" + iss.Body)
	if m.File != "" && strings.Contains(haystack, strings.ToLower(m.File)) {
		return matchFile
	}
	if phraseMatch(m.Text, haystack) {
		return matchPhrase
	}
	return matchNone
}

// found reports whether any signal fired.
func (k matchKind) found() bool { return k != matchNone }

// retires reports whether k is strong enough to keep a row out of the
// report, in either direction. Only a citation is: STYLE P3's "names that
// issue in the text" is the one signal a human wrote on purpose, and both
// weaker signals have been caught retiring a row they had no claim to — see
// [anyOpenMatch] and [trackerEvidence] for the case that settles each.
func (k matchKind) retires() bool { return k == matchCited }

// String names the signal as the report prints it, in the verb form that
// completes "<marker> <verb> <issue>".
func (k matchKind) String() string {
	switch k {
	case matchCited:
		return "cites"
	case matchFile:
		return "file-matches"
	case matchPhrase:
		return "phrase-matches"
	default:
		return "does not match"
	}
}

// wordPattern extracts the alphanumeric runs normalizeWords tokenizes on.
var wordPattern = regexp.MustCompile(`[A-Za-z0-9]+`)

// normalizeWords lowercases s and splits it into alphanumeric words,
// discarding punctuation and markup so a marker's angle-bracketed
// `<simpleType>` and an issue's plain-prose "simpleType" tokenize the same.
func normalizeWords(s string) []string {
	return wordPattern.FindAllString(strings.ToLower(s), -1)
}

// phraseMatch reports whether any run of minPhraseWords consecutive words
// from text appears, in the same order, inside haystack. Comparing
// space-joined word runs (rather than the raw substrings) means differences
// in whitespace, punctuation, and comment wrapping between the marker's
// source line and the issue's markdown never cause a miss.
func phraseMatch(text, haystack string) bool {
	words := normalizeWords(text)
	if len(words) < minPhraseWords {
		return false
	}
	joined := strings.Join(normalizeWords(haystack), " ")
	for i := 0; i+minPhraseWords <= len(words); i++ {
		window := strings.Join(words[i:i+minPhraseWords], " ")
		if strings.Contains(joined, window) {
			return true
		}
	}
	return false
}

// latchWriter is an [io.Writer] that remembers its first failure and drops
// every write after it. A report is a long run of small writes to one
// destination, so checking each in place would bury the layout in error
// handling; the caller checks err once, at the end.
type latchWriter struct {
	w   io.Writer
	err error
}

func (l *latchWriter) Write(p []byte) (int, error) {
	if l.err != nil {
		return 0, l.err
	}
	n, err := l.w.Write(p)
	l.err = err
	return n, err
}

// printReport renders rep to w in the fixed, deterministic layout described
// in the package doc. Formatting lives here, separate from [reconcile], so
// the decision logic stays testable without string-matching rendered text.
func printReport(dst io.Writer, rep report) error {
	w := &latchWriter{w: dst}
	printReportTo(w, rep)
	if w.err != nil {
		return fmt.Errorf("writing report: %w", w.err)
	}
	return nil
}

// printReportTo does the rendering itself. Its writes go to a [latchWriter],
// which is why they are not individually checked.
func printReportTo(w io.Writer, rep report) {
	total := 0
	for _, ac := range rep.Census {
		total += ac.Count
	}
	_, _ = fmt.Fprintf(w, "gapaudit: %d GAP marker(s) across %d area(s)\n", total, len(rep.Census))

	_, _ = fmt.Fprintln(w, "\n=== Per-area census ===")
	if len(rep.Census) == 0 {
		_, _ = fmt.Fprintln(w, "(no GAP markers found)")
	}
	for _, ac := range rep.Census {
		_, _ = fmt.Fprintf(w, "  %-16s %d\n", ac.Area, ac.Count)
	}

	if !rep.HaveIssues {
		_, _ = fmt.Fprintln(w, "\ngapaudit: no issue list on stdin — marker-inventory-only mode;"+
			" groups 1 and 2 (tracking reconciliation) were skipped.")
		return
	}

	_, _ = fmt.Fprintln(w, "\n=== Group 1: markers citing no OPEN tracking issue ===")
	_, _ = fmt.Fprintln(w, "(only a citation keeps a marker out of this group, so a row means \"needs a")
	_, _ = fmt.Fprintln(w, "look\", not proven untracked. Read every row's annotations first: a candidate")
	_, _ = fmt.Fprintln(w, "owner wants that issue's number written into the marker, and only a row with")
	_, _ = fmt.Fprintln(w, "NO annotation at all is a candidate for filing.)")
	if len(rep.Untracked) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, u := range rep.Untracked {
		_, _ = fmt.Fprintf(w, "  %s:%d [%s] %s\n", u.Marker.File, u.Marker.Line, u.Marker.Area, u.Marker.Text)
		for _, n := range u.Unresolved {
			_, _ = fmt.Fprintf(w, "      cites #%d, which names no issue in the fed list —"+
				" the citation itself is the defect\n", n)
		}
		for _, im := range u.Matches {
			if im.Issue.open() {
				_, _ = fmt.Fprintf(w, "      candidate owner: %s OPEN #%d %q — write the number into"+
					" the marker if it owns this\n", im.Kind, im.Issue.Number, im.Issue.Title)
				continue
			}
			// Not "label": that names the issue-label type since #1062.
			verdict := "resemblance"
			if im.Kind == matchCited {
				verdict = "dead end"
			}
			_, _ = fmt.Fprintf(w, "      %s: %s CLOSED #%d %q\n", verdict, im.Kind, im.Issue.Number, im.Issue.Title)
		}
	}

	_, _ = fmt.Fprintln(w, "\n=== Group 2: OPEN kind/gap issues no marker cites ===")
	_, _ = fmt.Fprintln(w, "(a stale tracker if the gap was a marked fail-open site — but")
	_, _ = fmt.Fprintln(w, "kind/gap also labels conformance-lane gaps, which never carry a")
	_, _ = fmt.Fprintln(w, "marker and belong here permanently)")
	if !rep.Labeled {
		_, _ = fmt.Fprintln(w, "gapaudit: no row of the fed list carries a label, so this group selected")
		_, _ = fmt.Fprintln(w, "nothing. Reshape the input with `labels: [.labels[] | {name}]` — see")
		_, _ = fmt.Fprintln(w, "docs/ROUTINES.md's \"Survey input\".")
	}
	if len(rep.Stale) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, st := range rep.Stale {
		_, _ = fmt.Fprintf(w, "  #%d %s\n", st.Issue.Number, st.Issue.Title)
		for _, wm := range st.Weak {
			_, _ = fmt.Fprintf(w, "      %s %s:%d — too weak to retire this tracker;"+
				" check whether it is the same gap\n", wm.Kind, wm.Marker.File, wm.Marker.Line)
		}
	}
}
