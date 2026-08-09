package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractMarkerBasic checks that a single-line marker yields the area,
// line, and trailing text a caller needs to cite it (docs/WORKFLOW.md
// prefers citing marker text over a line number).
func TestExtractMarkerBasic(t *testing.T) {
	src := "package p\n\n" +
		"// GAP(xsd): the wildcard case is not yet folded in.\n" +
		"func f() {}\n"

	got, err := extractMarkers("xsd/wildcard.go", []byte(src))
	if err != nil {
		t.Fatalf("extractMarkers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d markers, want 1: %+v", len(got), got)
	}
	m := got[0]
	if m.Area != "xsd" {
		t.Errorf("Area = %q, want xsd", m.Area)
	}
	if m.Line != 3 {
		t.Errorf("Line = %d, want 3", m.Line)
	}
	if m.File != "xsd/wildcard.go" {
		t.Errorf("File = %q, want xsd/wildcard.go", m.File)
	}
	want := "the wildcard case is not yet folded in."
	if m.Text != want {
		t.Errorf("Text = %q, want %q", m.Text, want)
	}
}

// TestExtractMarkerParagraphContinuation checks that a marker's text absorbs
// following comment lines that continue its prose, and stops at the first
// blank comment line — the paragraph boundary a human reading it would use.
func TestExtractMarkerParagraphContinuation(t *testing.T) {
	src := "package p\n\n" +
		"// GAP(xsd): this gap continues\n" +
		"// onto a second line before stopping\n" +
		"// at a blank comment line.\n" +
		"//\n" +
		"// This trailing paragraph is not part of the marker.\n" +
		"func f() {}\n"

	got, err := extractMarkers("x.go", []byte(src))
	if err != nil {
		t.Fatalf("extractMarkers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d markers, want 1: %+v", len(got), got)
	}
	want := "this gap continues onto a second line before stopping at a blank comment line."
	if got[0].Text != want {
		t.Errorf("Text = %q, want %q", got[0].Text, want)
	}
}

// TestExtractMarkerStopsAtBullet checks that a paragraph does not absorb a
// following numbered- or dash-list item: those are the shape #305/#286-style
// GAP comments use to enumerate several sub-cases under one heading, and
// each sub-case is not part of the FIRST marker's own sentence.
func TestExtractMarkerStopsAtBullet(t *testing.T) {
	src := "package p\n\n" +
		"// GAP(xsd): first sentence of the gap.\n" +
		"// 1. This is a new bullet and must not be absorbed.\n" +
		"func f() {}\n"

	got, err := extractMarkers("x.go", []byte(src))
	if err != nil {
		t.Fatalf("extractMarkers: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d markers, want 1: %+v", len(got), got)
	}
	want := "first sentence of the gap."
	if got[0].Text != want {
		t.Errorf("Text = %q, want %q", got[0].Text, want)
	}
}

// TestExtractMarkerStopsAtNextMarker checks that two markers on adjacent
// comment lines are extracted as two separate markers, neither absorbing the
// other's text — value/valuespace.go's numbered GAP(value) list is exactly
// this shape.
func TestExtractMarkerStopsAtNextMarker(t *testing.T) {
	src := "package p\n\n" +
		"// GAP(xsd): first marker text only.\n" +
		"// GAP(parser): second marker begins immediately.\n" +
		"func f() {}\n"

	got, err := extractMarkers("x.go", []byte(src))
	if err != nil {
		t.Fatalf("extractMarkers: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d markers, want 2: %+v", len(got), got)
	}
	if got[0].Area != "xsd" || got[0].Text != "first marker text only." {
		t.Errorf("marker 0 = %+v", got[0])
	}
	if got[1].Area != "parser" || got[1].Text != "second marker begins immediately." {
		t.Errorf("marker 1 = %+v", got[1])
	}
}

// TestExtractMarkerFalsePositives checks the two shapes that look like a
// marker but are not one: the marker syntax appearing inside a string
// literal (not a comment at all), and a comment that merely MENTIONS an
// earlier marker without repeating its head (no colon immediately after the
// closing paren). The fixtures below spell that head with a literal area
// name rather than reusing the phrase in this doc comment, so this test
// file does not become a marker site itself under gapaudit's own scan.
func TestExtractMarkerFalsePositives(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "marker syntax inside a string literal",
			src: "package p\n\n" +
				`const s = "GAP(oops): not a marker, just a string literal"` + "\n\n" +
				"// GAP(real): a genuine marker for comparison.\n" +
				"func f() {}\n",
		},
		{
			name: "prose mentioning an earlier marker, no colon",
			src: "package p\n\n" +
				"// See the GAP(value) gate above for details; there is no\n" +
				"// marker head on this line, so it must not be extracted.\n\n" +
				"// GAP(real): a genuine marker for comparison.\n" +
				"func f() {}\n",
		},
		{
			// A doc comment showing the marker convention in an indented
			// example quotes the comment rather than being one. Counting it
			// reports the same phantom every audit — xpath/doc.go is the
			// live instance.
			name: "doc comment quoting the marker convention",
			src: "package p\n\n" +
				"// Every fail-open site carries a greppable marker:\n" +
				"//\n" +
				"//\t// GAP(xpath): <construct>\n" +
				"//\n" +
				"// Direction matters.\n\n" +
				"// GAP(real): a genuine marker for comparison.\n" +
				"func f() {}\n",
		},
		{
			// The exclusion must not fire on angle brackets themselves:
			// genuine markers carry XSD element names.
			name: "genuine marker containing an XSD element name",
			src: "package p\n\n" +
				"// GAP(real): a top-level <simpleType> is still outside this\n" +
				"// producer's reach.\n" +
				"func f() {}\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := extractMarkers("x.go", []byte(tc.src))
			if err != nil {
				t.Fatalf("extractMarkers: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d markers, want exactly the 1 genuine one: %+v", len(got), got)
			}
			if got[0].Area != "real" {
				t.Errorf("Area = %q, want real (the false-positive site was still picked up)", got[0].Area)
			}
		})
	}
}

// TestScanTreeSkipsVendorAndSortsPaths exercises the one filesystem-touching
// entry point end to end: it writes a small tree with a vendor directory and
// two packages whose paths sort out of the order they're created in, and
// checks that scanTree both skips vendor and returns files in path order.
func TestScanTreeSkipsVendorAndSortsPaths(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	marker := func(area string) string {
		return "package p\n\n// GAP(" + area + "): a marker in this file.\n"
	}
	write("zzz/b.go", marker("zzz"))
	write("aaa/a.go", marker("aaa"))
	write("vendor/ignored/v.go", marker("vendored"))

	got, err := scanTree(root)
	if err != nil {
		t.Fatalf("scanTree: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d markers, want 2 (vendor must be skipped): %+v", len(got), got)
	}
	if got[0].File != "aaa/a.go" || got[1].File != "zzz/b.go" {
		t.Errorf("files not in sorted order: %q, %q", got[0].File, got[1].File)
	}
}

// TestReadIssues checks the three shapes readIssues must distinguish: no
// stdin at all (inventory-only mode), a valid issue list, and input that
// does not decode.
func TestReadIssues(t *testing.T) {
	t.Run("empty stdin means inventory-only", func(t *testing.T) {
		issues, have, err := readIssues(strings.NewReader(""))
		if err != nil {
			t.Fatalf("readIssues: %v", err)
		}
		if have {
			t.Error("haveIssues = true, want false for empty stdin")
		}
		if len(issues) != 0 {
			t.Errorf("issues = %+v, want none", issues)
		}
	})

	t.Run("decodes the gh issue list shape", func(t *testing.T) {
		in := `[{"number":447,"title":"track the xsd wildcard gap","state":"OPEN","body":"see xsd/wildcard.go"}]`
		issues, have, err := readIssues(strings.NewReader(in))
		if err != nil {
			t.Fatalf("readIssues: %v", err)
		}
		if !have {
			t.Fatal("haveIssues = false, want true")
		}
		if len(issues) != 1 || issues[0].Number != 447 {
			t.Errorf("issues = %+v", issues)
		}
	})

	t.Run("invalid JSON is an operational error", func(t *testing.T) {
		_, _, err := readIssues(strings.NewReader("not json"))
		if err == nil {
			t.Fatal("readIssues: want an error for undecodable input")
		}
	})
}

// TestMatchesByFilePath checks that an issue naming a marker's file, even
// with unrelated or too-short marker text, counts as a match.
func TestMatchesByFilePath(t *testing.T) {
	m := marker{Area: "xsd", File: "xsd/wildcard.go", Line: 111, Text: "short"}
	iss := issue{Number: 1, Title: "close the wildcard gap", State: "OPEN",
		Body: "the fail-open site is xsd/wildcard.go, clause 1 only"}

	if !matches(m, iss) {
		t.Error("matches = false, want true (file path is named in the issue body)")
	}
}

// TestMatchesByPhrase checks that a distinctive run of the marker's own
// words, reproduced in the issue, counts as a match even when the file path
// is never mentioned (an issue that quotes the marker's prose rather than
// citing a path).
func TestMatchesByPhrase(t *testing.T) {
	m := marker{
		Area: "xsd",
		File: "xsd/wildcard.go",
		Line: 111,
		Text: "this implements cvc-wildcard clause 1 ONLY, expanded name matching is not yet folded in",
	}
	iss := issue{
		Number: 2,
		Title:  "wildcard gap",
		State:  "OPEN",
		Body:   "Tracking: expanded name matching is not yet folded in for this construct.",
	}

	if !matches(m, iss) {
		t.Error("matches = false, want true (a 5-word run of the marker's text appears in the issue)")
	}
}

// TestMatchesFalseWhenNeitherSignalFires checks that unrelated marker and
// issue text do not match by either signal.
func TestMatchesFalseWhenNeitherSignalFires(t *testing.T) {
	m := marker{Area: "xsd", File: "xsd/wildcard.go", Line: 5, Text: "the wildcard case is not folded in yet"}
	iss := issue{Number: 3, Title: "unrelated parser bug", State: "OPEN", Body: "a completely different problem in the lexer"}

	if matches(m, iss) {
		t.Error("matches = true, want false")
	}
}

// TestReconcileUnmatchedMarkerLandsInGroup1 checks that a marker matching no
// issue at all is reported as untracked.
func TestReconcileUnmatchedMarkerLandsInGroup1(t *testing.T) {
	markers := []marker{
		{Area: "xsd", File: "xsd/orphan.go", Line: 9, Text: "nobody has filed a tracking issue for this one yet"},
	}
	issues := []issue{
		{Number: 1, Title: "unrelated", State: "OPEN", Body: "nothing to do with the marker above"},
	}

	rep := reconcile(markers, issues, true)
	if len(rep.Untracked) != 1 {
		t.Fatalf("Untracked = %+v, want 1 entry", rep.Untracked)
	}
	if rep.Untracked[0].Marker.File != "xsd/orphan.go" {
		t.Errorf("Untracked[0].Marker.File = %q", rep.Untracked[0].Marker.File)
	}
	if len(rep.Stale) != 1 {
		t.Fatalf("Stale = %+v, want the one open issue with no marker match", rep.Stale)
	}
}

// TestReconcileClosedMatchIsStillUntracked checks the STYLE P3 "dead end"
// case: a marker whose only match is a CLOSED issue is reported the same as
// an unmatched marker, with the closed issue recorded for context.
func TestReconcileClosedMatchIsStillUntracked(t *testing.T) {
	markers := []marker{
		{Area: "xsd", File: "xsd/wildcard.go", Line: 5, Text: "the wildcard case is not folded in yet, see the note above"},
	}
	issues := []issue{
		{Number: 42, Title: "wildcard gap", State: "CLOSED", Body: "the wildcard case is not folded in yet"},
	}

	rep := reconcile(markers, issues, true)
	if len(rep.Untracked) != 1 {
		t.Fatalf("Untracked = %+v, want 1 entry (closed match is a dead end)", rep.Untracked)
	}
	if len(rep.Untracked[0].ClosedMatches) != 1 || rep.Untracked[0].ClosedMatches[0].Number != 42 {
		t.Errorf("ClosedMatches = %+v, want [#42]", rep.Untracked[0].ClosedMatches)
	}
}

// TestReconcileOpenIssueNoMarkerLandsInGroup2 checks that an OPEN issue whose
// marker has been removed from the tree (or never matches anything left in
// it) is reported as a stale tracker.
func TestReconcileOpenIssueNoMarkerLandsInGroup2(t *testing.T) {
	markers := []marker{
		{Area: "xsd", File: "xsd/still-here.go", Line: 1, Text: "an entirely different still-open gap in this file"},
	}
	issues := []issue{
		{Number: 55, Title: "gap that got fixed without closing the issue", State: "OPEN", Body: "xsd/long-gone.go had a fail-open branch"},
	}

	rep := reconcile(markers, issues, true)
	if len(rep.Stale) != 1 || rep.Stale[0].Number != 55 {
		t.Fatalf("Stale = %+v, want [#55]", rep.Stale)
	}
	// The unrelated marker must still surface in group 1: it matches nothing.
	if len(rep.Untracked) != 1 {
		t.Fatalf("Untracked = %+v, want 1 entry", rep.Untracked)
	}
}

// TestReconcileTrackedMarkerIsNotReported checks the ordinary passing case:
// a marker matched to an OPEN issue appears in neither group.
func TestReconcileTrackedMarkerIsNotReported(t *testing.T) {
	markers := []marker{
		{Area: "xsd", File: "xsd/wildcard.go", Line: 5, Text: "the wildcard case is not folded in yet"},
	}
	issues := []issue{
		{Number: 7, Title: "wildcard gap", State: "OPEN", Body: "tracked at xsd/wildcard.go"},
	}

	rep := reconcile(markers, issues, true)
	if len(rep.Untracked) != 0 {
		t.Errorf("Untracked = %+v, want none", rep.Untracked)
	}
	if len(rep.Stale) != 0 {
		t.Errorf("Stale = %+v, want none", rep.Stale)
	}
}

// TestReconcileInventoryOnlyMode checks that a false haveIssues flag skips
// groups 1 and 2 entirely, computing only the census.
func TestReconcileInventoryOnlyMode(t *testing.T) {
	markers := []marker{{Area: "xsd", File: "x.go", Line: 1, Text: "anything"}}

	rep := reconcile(markers, nil, false)
	if rep.Untracked != nil {
		t.Errorf("Untracked = %+v, want nil in inventory-only mode", rep.Untracked)
	}
	if rep.Stale != nil {
		t.Errorf("Stale = %+v, want nil in inventory-only mode", rep.Stale)
	}
	if len(rep.Census) != 1 || rep.Census[0].Area != "xsd" || rep.Census[0].Count != 1 {
		t.Errorf("Census = %+v, want [{xsd 1}]", rep.Census)
	}
}

// TestCensusCounts checks that the per-area census aggregates and sorts
// correctly, including an area with more than one marker.
func TestCensusCounts(t *testing.T) {
	markers := []marker{
		{Area: "xsd", File: "a.go", Line: 1, Text: "one"},
		{Area: "parser", File: "b.go", Line: 2, Text: "two"},
		{Area: "xsd", File: "c.go", Line: 3, Text: "three"},
		{Area: "xpath", File: "d.go", Line: 4, Text: "four"},
	}

	got := census(markers)
	want := []areaCount{
		{Area: "parser", Count: 1},
		{Area: "xpath", Count: 1},
		{Area: "xsd", Count: 2},
	}
	if len(got) != len(want) {
		t.Fatalf("census = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("census[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestReconcileOrderingIsStable checks that reconcile's group-1 and group-3
// output is sorted deterministically regardless of the input order — the
// requirement STYLE D1 places on every report this tool produces.
func TestReconcileOrderingIsStable(t *testing.T) {
	// Deliberately out of file/line order.
	markers := []marker{
		{Area: "xsd", File: "z.go", Line: 9, Text: "an orphaned gap in z"},
		{Area: "parser", File: "a.go", Line: 20, Text: "an orphaned gap in a, second"},
		{Area: "parser", File: "a.go", Line: 5, Text: "an orphaned gap in a, first"},
	}

	rep := reconcile(markers, nil, true)
	if len(rep.Untracked) != 3 {
		t.Fatalf("Untracked = %+v, want 3 entries", rep.Untracked)
	}
	gotOrder := []string{}
	for _, u := range rep.Untracked {
		gotOrder = append(gotOrder, u.Marker.File)
	}
	want := []string{"a.go", "a.go", "z.go"}
	for i := range want {
		if gotOrder[i] != want[i] {
			t.Fatalf("Untracked order = %v, want File order %v", gotOrder, want)
		}
	}
	if rep.Untracked[0].Marker.Line != 5 || rep.Untracked[1].Marker.Line != 20 {
		t.Errorf("within a.go, lines = %d, %d, want 5 then 20",
			rep.Untracked[0].Marker.Line, rep.Untracked[1].Marker.Line)
	}

	// Rerunning on the same (still out-of-order) input must produce the
	// byte-identical order again.
	rep2 := reconcile(markers, nil, true)
	for i := range rep.Untracked {
		if rep.Untracked[i].Marker != rep2.Untracked[i].Marker {
			t.Fatalf("reconcile is not deterministic across reruns: %+v vs %+v", rep, rep2)
		}
	}
}

// TestPrintReportDoesNotPanicAndNamesEachGroup is a light smoke test on the
// renderer: it must run over every combination this tool produces (findings
// present, findings absent, inventory-only) without panicking, and must
// label all three groups so a reader can find them.
func TestPrintReportDoesNotPanicAndNamesEachGroup(t *testing.T) {
	rep := report{
		HaveIssues: true,
		Untracked: []untrackedMarker{
			{Marker: marker{Area: "xsd", File: "x.go", Line: 1, Text: "an orphan"}},
		},
		Stale:  []issue{{Number: 9, Title: "stale tracker"}},
		Census: []areaCount{{Area: "xsd", Count: 1}},
	}

	var buf strings.Builder
	if err := printReport(&buf, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Per-area census", "Group 1", "Group 2", "x.go", "#9 stale tracker"} {
		if !strings.Contains(out, want) {
			t.Errorf("report output missing %q:\n%s", want, out)
		}
	}
}
