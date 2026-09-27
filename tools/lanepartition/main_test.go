package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureSuite is a miniature catalog: two test sets, each declaring valid,
// invalid and indeterminate schema cases, so every column of the report has a
// case in it.
var fixtureSuite = map[string]string{
	"suite.xml": `<testSuite xmlns:xlink="http://www.w3.org/1999/xlink">
  <testSetRef xlink:href="sets/a.testSet"/>
  <testSetRef xlink:href="sets/b.testSet"/>
</testSuite>`,
	"sets/a.testSet": `<testSet name="A" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="g">
    <schemaTest name="v1"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="valid"/></schemaTest>
    <schemaTest name="v2"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="valid"/></schemaTest>
    <schemaTest name="i1"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="invalid"/></schemaTest>
    <schemaTest name="i2"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="invalid"/></schemaTest>
    <schemaTest name="i3"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="invalid"/></schemaTest>
    <schemaTest name="n1"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="indeterminate"/></schemaTest>
    <schemaTest name="p1"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="valid"/></schemaTest>
  </testGroup>
</testSet>`,
	"sets/b.testSet": `<testSet name="B" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="g">
    <schemaTest name="i1"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="invalid"/></schemaTest>
    <schemaTest name="v1"><schemaDocument xlink:href="../d/x.xsd"/><expected validity="valid"/></schemaTest>
  </testGroup>
</testSet>`,
}

// fixtureLanes banks every fixture case fail except A's p1, which is banked
// pass and so is in no partition.
var fixtureLanes = map[string]string{
	"schema.txt": `A/g/schema/i1 fail
A/g/schema/i2 fail
A/g/schema/i3 fail
A/g/schema/n1 fail
A/g/schema/p1 pass
A/g/schema/v1 fail
A/g/schema/v2 fail
B/g/schema/i1 fail
B/g/schema/v1 fail
`,
}

// fixtureLog is a GOXSD_DECLINES=1 run's census over the fixture lane, inside
// the noise a real -v log carries around it: A's i1 and B's v1 declined, A's
// n1 indeterminate, three decided disagreements charged two ways, and B's i1
// in no list — the run passed it. X/g/schema/z is listed but not banked fail.
const fixtureLog = `=== RUN   TestConformance
    conformance_test.go:172: lane schema: 9 cases
    conformance_test.go:245: lane schema: decline candidates: [A/g/schema/i1 B/g/schema/v1]
    conformance_test.go:246: lane schema: indeterminate declines: [A/g/schema/n1]
    conformance_test.go:247: lane schema: decided disagreements: [A/g/schema/i2=(accepted) A/g/schema/i3=(accepted) A/g/schema/v1=src-ct X/g/schema/z=(accepted) A/g/schema/v2=src-ct]
    conformance_test.go:245: lane instance: decline candidates: []
    conformance_test.go:246: lane instance: indeterminate declines: []
    conformance_test.go:247: lane instance: decided disagreements: []
--- PASS: TestConformance (1.00s)
`

// writeTree materializes a name-to-body map under a fresh temp directory and
// returns its root.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return root
}

// runFixture drives the whole command against the fixture suite and lane,
// with args behind the two directory flags, and returns what it printed and
// the error it returned.
func runFixture(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out strings.Builder
	full := append([]string{"-suite", writeTree(t, fixtureSuite), "-expectations", writeTree(t, fixtureLanes)}, args...)
	err := run(&out, strings.NewReader(stdin), full)
	return out.String(), err
}

// logFile writes body to a temp file and returns its path.
func logFile(t *testing.T, body string) string {
	t.Helper()
	return filepath.Join(writeTree(t, map[string]string{"run.log": body}), "run.log")
}

// clusterLines returns the report's cluster rows, in printed order, with runs
// of spaces collapsed so a test compares content and not padding.
func clusterLines(got string) []string {
	_, after, _ := strings.Cut(got, "=== Clusters")
	var rows []string
	for _, line := range strings.Split(after, "\n")[1:] {
		fields := strings.Fields(line)
		if len(fields) == 0 || strings.HasPrefix(fields[0], "(") {
			continue
		}
		rows = append(rows, strings.Join(fields, " "))
	}
	return rows
}

// wantClusters fails unless the report's cluster rows are exactly want, in
// order.
func wantClusters(t *testing.T, got string, want ...string) {
	t.Helper()
	rows := clusterLines(got)
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Errorf("clusters =\n%s\nwant\n%s\nreport:\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"), got)
	}
}

// wantTableRow fails unless the class-by-validity table carries the row
// class with exactly these counts, total last.
func wantTableRow(t *testing.T, got, class string, counts ...string) {
	t.Helper()
	want := strings.Join(append([]string{class}, counts...), " ")
	for _, line := range strings.Split(got, "\n") {
		if strings.Join(strings.Fields(line), " ") == want {
			return
		}
	}
	t.Errorf("no table row %q in:\n%s", want, got)
}

// TestWithoutALogTheCatalogSplitsValidFromTheRest is the two-way partition
// the catalog alone supports: DeclaredValid folds indeterminate into
// not-valid, so A's n1 counts with the invalid cases, and a banked pass is in
// no cell at all.
func TestWithoutALogTheCatalogSplitsValidFromTheRest(t *testing.T) {
	got, err := runFixture(t, "", "schema")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	wantTableRow(t, got, "banked fail", "3", "5", "8")
	wantClusters(t, got,
		"4 banked fail invalid|indeterminate A",
		"2 banked fail valid A",
		"1 banked fail valid B",
		"1 banked fail invalid|indeterminate B",
	)
	if !strings.HasPrefix(got, "lanepartition: lane schema — 8 banked fail(s) in ") {
		t.Errorf("header does not open with the lane and its banked-fail count:\n%s", got)
	}
}

// TestALogSplitsDeclinesFromDecidedAndClustersByCharge is the three-way
// partition a GOXSD_DECLINES=1 log adds: each banked fail classed by the
// census list naming it, indeterminate told from invalid, each decided case
// clustered by its charge, and the case the run passed counted in no list.
// Ties rank by class order, declared validity, then test set.
func TestALogSplitsDeclinesFromDecidedAndClustersByCharge(t *testing.T) {
	got, err := runFixture(t, "", "-log", logFile(t, fixtureLog), "schema")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	wantTableRow(t, got, "declined", "1", "1", "0", "2")
	wantTableRow(t, got, "indeterminate", "0", "0", "1", "1")
	wantTableRow(t, got, "decided and wrong", "2", "2", "0", "4")
	wantTableRow(t, got, "in no census list", "0", "1", "0", "1")
	wantClusters(t, got,
		"2 decided and wrong valid A [src-ct]",
		"2 decided and wrong invalid A [(accepted)]",
		"1 declined valid B",
		"1 declined invalid A",
		"1 indeterminate indeterminate A",
		"1 in no census list invalid B",
	)
	if !strings.Contains(got, "does not bank fail (1)") {
		t.Errorf("report does not count the listed case the file does not bank fail:\n%s", got)
	}
}

// TestAnIssueListNamesOpenIssuesByTestSetOrRule reconciles clusters against
// the fed list: an OPEN issue whose body names the test set or the charged
// rule as a whole token. A closed one is not named, `src-ct-extends` does not
// name `src-ct`, and a parenthesized charge is never searched for.
func TestAnIssueListNamesOpenIssuesByTestSetOrRule(t *testing.T) {
	issues := `[
  {"number": 9, "state": "OPEN", "title": "t", "body": "the src-ct.5 clause"},
  {"number": 3, "state": "OPEN", "title": "t", "body": "every case in set B"},
  {"number": 4, "state": "CLOSED", "title": "t", "body": "set A and src-ct"},
  {"number": 5, "state": "OPEN", "title": "t", "body": "src-ct-extends and (accepted) and ABC"}
]`
	got, err := runFixture(t, issues, "-log", logFile(t, fixtureLog), "schema")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	wantClusters(t, got,
		"2 decided and wrong valid A [src-ct] — src-ct: #9",
		"2 decided and wrong invalid A [(accepted)] — no open issue names it",
		"1 declined valid B — B: #3",
		"1 declined invalid A — no open issue names it",
		"1 indeterminate indeterminate A — no open issue names it",
		"1 in no census list invalid B — B: #3",
	)
	if !strings.Contains(got, "reconciled against 3 open issue(s)") {
		t.Errorf("report does not say what it reconciled against:\n%s", got)
	}
}

// TestNoIssueListReconcilesNothingAndSaysSo is the empty-stdin mode: a report,
// exit 0, and a line saying nothing was reconciled.
func TestNoIssueListReconcilesNothingAndSaysSo(t *testing.T) {
	got, err := runFixture(t, "", "schema")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !strings.Contains(got, "(no issue list on stdin: nothing reconciled)") {
		t.Errorf("report does not say it reconciled nothing:\n%s", got)
	}
}

// TestAFedRowLackingStateStillReportsThenFails mirrors wipsurvey's #1604
// rule: the list is discarded whole, the report prints unreconciled, and the
// run returns the error main exits 2 on, naming the issue.
func TestAFedRowLackingStateStillReportsThenFails(t *testing.T) {
	got, err := runFixture(t, `[{"number": 3, "state": "OPEN", "body": "B"}, {"number": 7, "body": "B"}]`, "schema")
	if !errors.Is(err, errMissingState) || !strings.Contains(err.Error(), "#7") {
		t.Fatalf("run error = %v, want errMissingState naming #7", err)
	}
	if !strings.Contains(got, "issue list discarded, nothing reconciled") || strings.Contains(got, "#3") {
		t.Errorf("report must print unreconciled:\n%s", got)
	}
	wantTableRow(t, got, "banked fail", "3", "5", "8")
}

// TestOperationalErrorsReturnAnError covers the exit-2 cases other than a
// stateless row: each returns an error, which main turns into exit status 2.
func TestOperationalErrorsReturnAnError(t *testing.T) {
	cases := []struct {
		name, stdin, want string
		args              []string
	}{
		{"malformed JSON", "[{", "parsing issue JSON", []string{"schema"}},
		{"no lane", "", "name exactly one lane", nil},
		{"no expectation file", "", "no expectation file", []string{"xpath"}},
		{"unreadable log", "", "reading run log", []string{"-log", "/nonexistent/run.log", "schema"}},
		{"log without the listing", "", `no "indeterminate declines" listing for lane schema`, []string{"-log", logFile(t, "lane schema: decline candidates: []\n"), "schema"}},
		{"log listing twice", "", "twice", []string{"-log", logFile(t, fixtureLog+fixtureLog), "schema"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := runFixture(t, tc.stdin, tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("run error = %v, want one containing %q", err, tc.want)
			}
		})
	}
}

// TestABankedCaseTheCatalogLacksIsAnError: a lane file and a suite that are
// not one tree's must not be partitioned as if they were.
func TestABankedCaseTheCatalogLacksIsAnError(t *testing.T) {
	var out strings.Builder
	lanes := map[string]string{"schema.txt": "Q/g/schema/nowhere fail\n"}
	err := run(&out, strings.NewReader(""), []string{"-suite", writeTree(t, fixtureSuite), "-expectations", writeTree(t, lanes), "schema"})
	if err == nil || !strings.HasPrefix(err.Error(), "banked case Q/g/schema/nowhere has no catalog entry") {
		t.Errorf("run error = %v, want the uncatalogued case named", err)
	}
}

// TestAnIndeterminateListingOfAValidCaseIsAnError: the catalog declares A's
// v1 valid, so a log listing it indeterminate is from another suite.
func TestAnIndeterminateListingOfAValidCaseIsAnError(t *testing.T) {
	log := strings.Replace(fixtureLog, "indeterminate declines: [A/g/schema/n1]", "indeterminate declines: [A/g/schema/n1 A/g/schema/v1]", 1)
	log = strings.Replace(log, " A/g/schema/v1=src-ct", "", 1)
	_, err := runFixture(t, "", "-log", logFile(t, log), "schema")
	if err == nil || !strings.HasPrefix(err.Error(), "the run log lists A/g/schema/v1 as indeterminate but the catalog declares it valid") {
		t.Errorf("run error = %v, want the contradiction named", err)
	}
}

// TestNamesTokenNeedsAWholeToken pins the token boundary: a rule ID names a
// clause of itself, never a longer ID it prefixes, and never an ID it ends.
func TestNamesTokenNeedsAWholeToken(t *testing.T) {
	cases := []struct {
		text, term string
		want       bool
	}{
		{"src-ct", "src-ct", true},
		{"see src-ct.5 here", "src-ct", true},
		{"see (src-ct)", "src-ct", true},
		{"src-ct-extends", "src-ct", false},
		{"cos-src-ct", "src-ct", false},
		{"cos-src-ct and src-ct", "src-ct", true},
		{"msData2", "msData", false},
		{"msData/foo", "msData", true},
	}
	for _, tc := range cases {
		if got := namesToken(tc.text, tc.term); got != tc.want {
			t.Errorf("namesToken(%q, %q) = %v, want %v", tc.text, tc.term, got, tc.want)
		}
	}
}
