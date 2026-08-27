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

	if got := matches(m, iss); got != matchFile {
		t.Errorf("matches = %v, want matchFile (file path is named in the issue body)", got)
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

	if got := matches(m, iss); got != matchPhrase {
		t.Errorf("matches = %v, want matchPhrase (a 5-word run of the marker's text appears in the issue)", got)
	}
}

// TestMatchesFalseWhenNeitherSignalFires checks that unrelated marker and
// issue text do not match by either signal.
func TestMatchesFalseWhenNeitherSignalFires(t *testing.T) {
	m := marker{Area: "xsd", File: "xsd/wildcard.go", Line: 5, Text: "the wildcard case is not folded in yet"}
	iss := issue{Number: 3, Title: "unrelated parser bug", State: "OPEN", Body: "a completely different problem in the lexer"}

	if got := matches(m, iss); got != matchNone {
		t.Errorf("matches = %v, want matchNone", got)
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
	if len(rep.Untracked[0].ClosedMatches) != 1 || rep.Untracked[0].ClosedMatches[0].Issue.Number != 42 {
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
	if len(rep.Stale) != 1 || rep.Stale[0].Issue.Number != 55 {
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
		Stale:  []staleTracker{{Issue: issue{Number: 9, Title: "stale tracker"}}},
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

// TestCitations checks the citation extractor: every `#<digits>` run in the
// marker's text, first-appearance order, repeats dropped.
func TestCitations(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []int
	}{
		{name: "none", text: "no owner names this gap yet", want: nil},
		{name: "one, parenthesized", text: "declines instead (#774).", want: []int{774}},
		{name: "one, mid-sentence", text: "owned by #1002, which is blocked", want: []int{1002}},
		{name: "repeat is dropped", text: "see #774 above; charged at #774 too", want: []int{774}},
		{name: "two owners keep their order", text: "#900 folds what #56 declines", want: []int{900, 56}},
		{name: "spec clauses are not citations", text: "§3.17.5.2 clause 4, cvc-id.1", want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := citations(tc.text)
			if len(got) != len(tc.want) {
				t.Fatalf("citations = %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("citations = %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// TestMatchesByCitation is the #852 regression: four markers reproduced from
// the tree, each naming its OPEN tracker with a `(#N)` the matcher used to be
// blind to. The marker text is verbatim; the issue carries its real number
// and title and no body, because against the real full bodies neither
// heuristic signal fires either — the file path appears in none of them and
// no minPhraseWords run is shared, which is exactly why all four reported
// untracked. The stripped sub-case is that state: delete the citation and the
// pair goes back to matching nothing.
func TestMatchesByCitation(t *testing.T) {
	tests := []struct {
		name   string
		marker marker
		issue  issue
	}{
		{
			name: "cvcid.go idAttributes cites #414",
			marker: marker{
				Area: "xsd", File: "validate/cvcid.go", Line: 169,
				Text: "where the governing type is complex and its {attribute uses} were NOT " +
					"folded, the same attribute may match a use this package cannot see " +
					"(assess.go's attributePropertiesFolded, #414), so the type read off a " +
					"top-level declaration — or the absence of one — is not the governing " +
					"type, and it declines instead.",
			},
			issue: issue{
				Number: 414, State: "OPEN",
				Title: "xsd: BOTH finalize folds walk the Schema's TYPE DEFINITIONS only",
			},
		},
		{
			name: "cvcid.go idDefaultedAttributes cites #774",
			marker: marker{
				Area: "validate", File: "validate/cvcid.go", Line: 196,
				Text: "§3.17.5.2's own Note puts one in the ·eligible item set· — \"the use " +
					"of [schema actual value] ... means that default or fixed value " +
					"constraints may play a part\" — because cvc-complex-type clause 4 " +
					"supplies the item and its ·actual value· is the constraint's. This " +
					"package does not synthesize the item, so the id it would declare is one " +
					"cvc-id never saw, and clause 1 would charge an empty binding for it " +
					"(#774).",
			},
			issue: issue{
				Number: 774, State: "OPEN",
				Title: "validate: the cvc-attribute/cvc-au declines an undecidable value space leaves",
			},
		},
		{
			name: "cvcid.go idRecord cites #774",
			marker: marker{
				Area: "validate", File: "validate/cvcid.go", Line: 290,
				Text: "a value.ValidateLexical error that is not a VERDICT " +
					"(value.IsDatatypeVerdict) declines, on cvcattribute.go's terms — an " +
					"ungoverned type reports under cvc-datatype-valid exactly as a genuine " +
					"rejection does, and reading one as \"no id here\" would hide a " +
					"declaration clause 1 charges for the absence of (#774). validatingType's " +
					"member scan declines on the same class for the same reason.",
			},
			issue: issue{
				Number: 774, State: "OPEN",
				Title: "validate: the cvc-attribute/cvc-au declines an undecidable value space leaves",
			},
		},
		{
			name: "cvccomplexcontent.go stringValid cites #774",
			marker: marker{
				Area: "validate", File: "validate/cvccomplexcontent.go", Line: 460,
				Text: "a ValidateLexical error that is not a VERDICT is the same fail-open " +
					"cvcattribute.go's matchedAttribute states in full, over the same " +
					"[value.IsDatatypeVerdict] classification: an ungoverned simple type " +
					"reports under cvc-datatype-valid exactly as a genuine rejection does, " +
					"and charging it would reject every element whose character content this " +
					"backend cannot read (#774).",
			},
			issue: issue{
				Number: 774, State: "OPEN",
				Title: "validate: the cvc-attribute/cvc-au declines an undecidable value space leaves",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := matches(tc.marker, tc.issue); got != matchCited {
				t.Errorf("matches = %v, want matchCited", got)
			}
			if !anyOpenMatch(tc.marker, []issue{tc.issue}) {
				t.Error("anyOpenMatch = false: the marker still lands in group 1")
			}

			stripped := tc.marker
			stripped.Text = citationPattern.ReplaceAllString(stripped.Text, "")
			if got := matches(stripped, tc.issue); got != matchNone {
				t.Errorf("with the citation removed, matches = %v, want matchNone"+
					" (the citation is the only signal tying this pair)", got)
			}
		})
	}
}

// TestMatchesCitationOutranksResemblance checks the ordering [matches]
// promises: a marker that cites an issue reports matchCited even where the
// weaker signals would also have fired, so the report never downgrades a
// citation to a resemblance.
func TestMatchesCitationOutranksResemblance(t *testing.T) {
	m := marker{Area: "xsd", File: "xsd/wildcard.go", Line: 5,
		Text: "expanded name matching is not yet folded in (#7)"}
	iss := issue{Number: 7, State: "OPEN", Title: "wildcard gap",
		Body: "xsd/wildcard.go: expanded name matching is not yet folded in"}

	if got := matches(m, iss); got != matchCited {
		t.Errorf("matches = %v, want matchCited", got)
	}
}

// TestUnresolvedCitationIsReportedNotCalledUntracked is the #852 defect-2
// case: the documented input is `kind/gap` only, and a marker whose owner
// carries another label cites a number the fed list cannot contain. The
// marker still lands in group 1 — nothing open matched it — but with the
// number it names, so the reader files nothing against it.
func TestUnresolvedCitationIsReportedNotCalledUntracked(t *testing.T) {
	markers := []marker{
		{Area: "validate", File: "validate/assess.go", Line: 685,
			Text: "an element whose type carries one (#717)."},
	}
	issues := []issue{{Number: 999, State: "OPEN", Title: "unrelated", Body: "nothing in common"}}

	rep := reconcile(markers, issues, true)
	if len(rep.Untracked) != 1 {
		t.Fatalf("Untracked = %+v, want 1 entry", rep.Untracked)
	}
	got := rep.Untracked[0].Unresolved
	if len(got) != 1 || got[0] != 717 {
		t.Fatalf("Unresolved = %v, want [717]", got)
	}

	var buf strings.Builder
	if err := printReport(&buf, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	if !strings.Contains(buf.String(), "cites #717, absent from the fed list") {
		t.Errorf("report does not name the unresolved citation:\n%s", buf.String())
	}
}

// TestPhraseCollisionDoesNotRetireATracker is the #852 defect-4 regression,
// reproduced from real text: parser/conditional.go's GAP(parser) marker and
// issue #921 — a conformance-lane tracker that carries no marker anywhere —
// share exactly one five-word window, "CLAUDE.md's one rule", because both
// cite the same line of CLAUDE.md. That collision used to delete #921 from
// group 2 silently. It must now leave the row standing and print the marker
// that collided.
func TestPhraseCollisionDoesNotRetireATracker(t *testing.T) {
	m := marker{Area: "parser", File: "parser/conditional.go", Line: 208,
		Text: "vc:maxVersion does not prune. Owned by #1002, which is blocked on a human " +
			"ruling: downgrading a banked case is not an agent's call (CLAUDE.md's one rule)."}
	iss := issue{Number: 921, State: "OPEN",
		Title: "conformance: <current status=\"queried\"> is unmodeled",
		Body: "Downgrading it is not an agent's call (CLAUDE.md's one rule, " +
			"`.claude/agents/arbiter.md`'s ratchet-integrity section)."}

	if got := matches(m, iss); got != matchPhrase {
		t.Fatalf("matches = %v, want matchPhrase (the fixture no longer reproduces the collision)", got)
	}

	rep := reconcile([]marker{m}, []issue{iss}, true)
	if len(rep.Stale) != 1 || rep.Stale[0].Issue.Number != 921 {
		t.Fatalf("Stale = %+v, want #921 still listed", rep.Stale)
	}
	if len(rep.Stale[0].Weak) != 1 || rep.Stale[0].Weak[0].File != "parser/conditional.go" {
		t.Fatalf("Weak = %+v, want the colliding marker named", rep.Stale[0].Weak)
	}

	var buf strings.Builder
	if err := printReport(&buf, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	if !strings.Contains(buf.String(), "phrase-matches parser/conditional.go:208 — too weak") {
		t.Errorf("report does not name the colliding marker:\n%s", buf.String())
	}
}

// TestCitationRetiresATracker checks the other direction of the same bar: a
// marker that CITES an open issue is strong enough to retire it, so a gap
// whose marker still stands never reports stale.
func TestCitationRetiresATracker(t *testing.T) {
	m := marker{Area: "xsd", File: "xsd/wildcard.go", Line: 5, Text: "still open (#7)"}
	iss := issue{Number: 7, State: "OPEN", Title: "wildcard gap", Body: "unrelated prose"}

	rep := reconcile([]marker{m}, []issue{iss}, true)
	if len(rep.Stale) != 0 {
		t.Errorf("Stale = %+v, want none (the marker cites #7)", rep.Stale)
	}
	if len(rep.Untracked) != 0 {
		t.Errorf("Untracked = %+v, want none", rep.Untracked)
	}
}

// TestFileMatchRetiresATracker checks that the file-path signal still retires
// a tracker: only the phrase run was demoted.
func TestFileMatchRetiresATracker(t *testing.T) {
	m := marker{Area: "xsd", File: "xsd/wildcard.go", Line: 5, Text: "no owner named here"}
	iss := issue{Number: 8, State: "OPEN", Title: "wildcard gap", Body: "the site is xsd/wildcard.go"}

	rep := reconcile([]marker{m}, []issue{iss}, true)
	if len(rep.Stale) != 0 {
		t.Errorf("Stale = %+v, want none (the issue names the marker's file)", rep.Stale)
	}
}

// TestDeadEndDistinguishesCitationFromResemblance is the #852 defect-3 case:
// a CLOSED issue the marker actually cites is a STYLE P3 dead end and must be
// repointed; a CLOSED issue that merely shares a phrase is not, and calling
// it one sends the reader to repoint a marker that never pointed there.
func TestDeadEndDistinguishesCitationFromResemblance(t *testing.T) {
	m := marker{Area: "validate", File: "validate/assess.go", Line: 208,
		Text: "the folded attribute uses of a governing type are not consulted here (#761)"}
	issues := []issue{
		{Number: 761, State: "CLOSED", Title: "the cited owner", Body: "unrelated prose entirely"},
		{Number: 800, State: "CLOSED", Title: "a collision",
			Body: "the folded attribute uses of a governing type are described here too"},
	}

	rep := reconcile([]marker{m}, issues, true)
	if len(rep.Untracked) != 1 {
		t.Fatalf("Untracked = %+v, want 1 entry", rep.Untracked)
	}
	cm := rep.Untracked[0].ClosedMatches
	if len(cm) != 2 {
		t.Fatalf("ClosedMatches = %+v, want both", cm)
	}
	if cm[0].Issue.Number != 761 || cm[0].Kind != matchCited {
		t.Errorf("ClosedMatches[0] = %+v, want #761 matchCited", cm[0])
	}
	if cm[1].Issue.Number != 800 || cm[1].Kind != matchPhrase {
		t.Errorf("ClosedMatches[1] = %+v, want #800 matchPhrase", cm[1])
	}

	var buf strings.Builder
	if err := printReport(&buf, rep); err != nil {
		t.Fatalf("printReport: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "dead end: cites CLOSED #761") {
		t.Errorf("the cited dead end is not rendered as one:\n%s", out)
	}
	if !strings.Contains(out, "resemblance: phrase-matches CLOSED #800") {
		t.Errorf("a phrase collision is still rendered as a dead end:\n%s", out)
	}
}
