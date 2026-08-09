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
//  1. Markers with no OPEN tracking issue matched — the leak the rule
//     exists to prevent. This includes a marker that only matches a CLOSED
//     issue: STYLE P3 says a marker pointing at a closed issue is a dead
//     end, so it is reported the same as an unmatched one, with the closed
//     match named for context.
//  2. OPEN `kind/gap` issues whose marker is no longer anywhere in the
//     tree — a stale tracker: the gap was closed in code without closing
//     the issue that tracked it.
//  3. A per-area census of marker counts.
//
// # Matching is heuristic
//
// A marker is matched to an issue by its file path, or by a run of words
// from the marker's text, appearing in the issue's title or body (see
// [matches]). That is a resemblance test, not a citation graph: a marker
// with no match is reported as "no tracking issue found", never as
// "untracked" — the matcher has no way to rule out an issue that describes
// the same gap in different words, so the reader is the one who gets to
// call it truly untracked.
//
// # Usage
//
//	go run ./tools/gapaudit [dir] < issues.json
//	gh issue list --label kind/gap --state all --json number,title,state,body | go run ./tools/gapaudit
//
// dir defaults to "." (the whole tree). Empty or absent stdin runs in
// marker-inventory-only mode: groups 1 and 2 are skipped (there is nothing
// to reconcile against) and the report says so.
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

// readIssues decodes the `gh issue list ... --json number,title,state,body`
// shape from r. An empty or absent stdin is not an error: haveIssues comes
// back false, and the caller runs in marker-inventory-only mode.
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

// paragraph assembles a marker's trailing text: the remainder of its own
// line from headEnd, plus every contiguous following comment line that is
// neither blank, a new bullet, a new marker, nor a heading — the same
// paragraph-boundary rules a human reading the comment would apply.
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
		if t == "" || strings.HasPrefix(t, "#") ||
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

// issue is one row of `gh issue list --json number,title,state,body`: the
// fields this tool needs to decide whether a marker is tracked. State is
// "OPEN" or "CLOSED" as gh emits it; matching compares it case-insensitively
// rather than adding a second encoding of the same fact.
type issue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Body   string `json:"body"`
}

func (iss issue) open() bool {
	return strings.EqualFold(iss.State, "OPEN")
}

// areaCount is one row of the per-area marker census.
type areaCount struct {
	Area  string
	Count int
}

// untrackedMarker is a group-1 finding: a marker with no OPEN issue matched.
// closedMatches records any CLOSED issue that DID match, for context — STYLE
// P3 treats a marker pointing at a closed issue as a dead end, the same as
// no match at all, but the reader deciding what to file next benefits from
// knowing a plausible predecessor already exists and was closed.
type untrackedMarker struct {
	Marker        marker
	ClosedMatches []issue
}

// report is gapaudit's full output: the three groups described in the
// package doc, plus whether an issue list was supplied at all (haveIssues
// false means groups 1 and 2 are meaningless and were not computed).
type report struct {
	HaveIssues bool
	Untracked  []untrackedMarker
	Stale      []issue
	Census     []areaCount
}

// minPhraseWords is how many consecutive words of a marker's text must
// appear, in order, inside an issue's title+body before the marker counts as
// matched by phrase rather than by file path. Below this a run of words is
// common enough (English filler, repeated spec vocabulary like "content
// model") to match issues that have nothing to do with the marker; at this
// length a match is a run distinctive enough to be worth a human's second
// look, per this tool's documented heuristic-not-proof contract.
const minPhraseWords = 5

// reconcile computes the three-group report from a marker inventory and an
// optional issue list. It is pure: no I/O, so tests exercise it directly
// with literal marker/issue slices. When haveIssues is false only the
// census (group 3) is populated, since there is nothing to reconcile markers
// against.
func reconcile(markers []marker, issues []issue, haveIssues bool) report {
	rep := report{HaveIssues: haveIssues, Census: census(markers)}
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
			Marker:        m,
			ClosedMatches: closedMatches(m, issues),
		})
	}

	for _, iss := range issues {
		if !iss.open() {
			continue
		}
		if anyMarkerMatch(iss, markers) {
			continue
		}
		rep.Stale = append(rep.Stale, iss)
	}
	sort.Slice(rep.Stale, func(i, j int) bool { return rep.Stale[i].Number < rep.Stale[j].Number })

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

func anyOpenMatch(m marker, issues []issue) bool {
	for _, iss := range issues {
		if iss.open() && matches(m, iss) {
			return true
		}
	}
	return false
}

func anyMarkerMatch(iss issue, markers []marker) bool {
	for _, m := range markers {
		if matches(m, iss) {
			return true
		}
	}
	return false
}

func closedMatches(m marker, issues []issue) []issue {
	var out []issue
	for _, iss := range issues {
		if iss.open() {
			continue
		}
		if matches(m, iss) {
			out = append(out, iss)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// matches is the heuristic at the center of this tool: it reports whether
// issue iss plausibly tracks marker m, by one of two signals — m's file path
// named in the issue, or a distinctive run of m's own words appearing in the
// issue's title or body. Neither signal is proof: a file gets renamed, a
// marker's prose gets rephrased in the issue that files it. Callers must
// treat a false result as "no match found", never as "definitely untracked"
// (see the package doc's matching-is-heuristic section).
func matches(m marker, iss issue) bool {
	haystack := strings.ToLower(iss.Title + "\n" + iss.Body)
	if m.File != "" && strings.Contains(haystack, strings.ToLower(m.File)) {
		return true
	}
	return phraseMatch(m.Text, haystack)
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

	_, _ = fmt.Fprintln(w, "\n=== Group 1: markers with no OPEN tracking issue found ===")
	_, _ = fmt.Fprintln(w, "(matching is heuristic — file path or a distinctive phrase; treat a")
	_, _ = fmt.Fprintln(w, "listing here as \"needs a human look\", not as proven untracked)")
	if len(rep.Untracked) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, u := range rep.Untracked {
		_, _ = fmt.Fprintf(w, "  %s:%d [%s] %s\n", u.Marker.File, u.Marker.Line, u.Marker.Area, u.Marker.Text)
		for _, iss := range u.ClosedMatches {
			_, _ = fmt.Fprintf(w, "      dead end: matches CLOSED #%d %q\n", iss.Number, iss.Title)
		}
	}

	_, _ = fmt.Fprintln(w, "\n=== Group 2: OPEN kind/gap issues with no surviving marker ===")
	_, _ = fmt.Fprintln(w, "(a stale tracker if the gap was a marked fail-open site — but")
	_, _ = fmt.Fprintln(w, "kind/gap also labels conformance-lane gaps, which never carry a")
	_, _ = fmt.Fprintln(w, "marker and belong here permanently)")
	if len(rep.Stale) == 0 {
		_, _ = fmt.Fprintln(w, "(none)")
	}
	for _, iss := range rep.Stale {
		_, _ = fmt.Fprintf(w, "  #%d %s\n", iss.Number, iss.Title)
	}
}
