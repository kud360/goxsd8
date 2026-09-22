package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// catalogFixtureSuite is a miniature suite catalog exercising every shape
// Catalog reports: one document named by more than one test group, the three
// levels versionApplicable withholds at, a schemaTest's ordered document list,
// the sibling-schemaTest documents an instance entry is assessed against, and
// the three declared outcomes.
//
// The hrefs climb out of the set directory (`../docs/…`) exactly as the
// pinned suite's own saxonMeta sets do, so the suite-relative spelling Catalog
// reports is the one a census of the corpus would print.
var catalogFixtureSuite = map[string]string{
	"suite.xml": `<testSuite xmlns:xlink="http://www.w3.org/1999/xlink">
  <testSetRef xlink:href="sets/applicable.testSet"/>
  <testSetRef xlink:href="sets/scoped.testSet"/>
</testSuite>`,

	"sets/applicable.testSet": `<testSet name="Applicable" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="g1">
    <schemaTest name="s1">
      <schemaDocument xlink:href="../docs/s1.xsd"/>
      <schemaDocument xlink:href="../docs/s1-extra.xsd"/>
      <expected validity="valid"/>
    </schemaTest>
    <instanceTest name="i1">
      <instanceDocument xlink:href="../docs/shared.xml"/>
      <expected validity="invalid"/>
    </instanceTest>
  </testGroup>
  <testGroup name="g2">
    <instanceTest name="i1">
      <instanceDocument xlink:href="../docs/shared.xml"/>
      <expected validity="valid"/>
    </instanceTest>
  </testGroup>
  <testGroup name="g3" version="1.0">
    <instanceTest name="i3">
      <instanceDocument xlink:href="../docs/shared.xml"/>
      <expected validity="invalid"/>
    </instanceTest>
  </testGroup>
  <testGroup name="g4">
    <instanceTest name="i4" version="1.0">
      <instanceDocument xlink:href="../docs/lonely.xml"/>
      <expected validity="invalid"/>
    </instanceTest>
    <instanceTest name="i5">
      <instanceDocument xlink:href="../docs/lonely.xml"/>
      <expected validity="valid" version="1.0"/>
    </instanceTest>
  </testGroup>
  <testGroup name="g5">
    <schemaTest name="s5">
      <expected validity="valid"/>
    </schemaTest>
  </testGroup>
</testSet>`,

	"sets/scoped.testSet": `<testSet name="Scoped" version="1.0" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="sg1">
    <instanceTest name="si1">
      <instanceDocument xlink:href="../docs/shared.xml"/>
      <expected validity="valid"/>
    </instanceTest>
  </testGroup>
</testSet>`,
}

// outcomelessSuite is a miniature catalog whose single instanceTest declares
// NO <expected> at all, scoped by the `version` value given — empty for an
// applicable test, "1.0" for one the suite scopes away from this processor.
// The two are the halves of catalogEntry's refusal: it refuses the first, as
// makeCase does, and describes the second, as discovery records it.
func outcomelessSuite(version string) map[string]string {
	return map[string]string{
		"suite.xml": `<testSuite xmlns:xlink="http://www.w3.org/1999/xlink">
  <testSetRef xlink:href="sets/outcomeless.testSet"/>
</testSuite>`,

		"sets/outcomeless.testSet": `<testSet name="Outcomeless" xmlns:xlink="http://www.w3.org/1999/xlink">
  <testGroup name="g1">
    <instanceTest name="i1" version="` + version + `">
      <instanceDocument xlink:href="../docs/i1.xml"/>
    </instanceTest>
  </testGroup>
</testSet>`,
	}
}

// writeCatalogFixture materializes a name-to-body map as a suite in a temp
// directory and returns its root.
func writeCatalogFixture(t *testing.T, files map[string]string) string {
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

// catalogByID indexes a read catalog for assertion, failing the test on an ID
// the fixture does not carry.
func catalogByID(t *testing.T, entries []CatalogEntry, id string) CatalogEntry {
	t.Helper()
	for _, e := range entries {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("no catalog entry %q in %d entries", id, len(entries))
	return CatalogEntry{}
}

// TestCatalogReportsEveryEntryTheSuiteNames holds the reader to the whole
// catalog: one entry per declared schemaTest and instanceTest, withheld levels
// included, each with the documents it names spelled relative to the suite
// root.
func TestCatalogReportsEveryEntryTheSuiteNames(t *testing.T) {
	entries, err := Catalog(writeCatalogFixture(t, catalogFixtureSuite))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	var ids []string
	for _, e := range entries {
		ids = append(ids, e.ID)
	}
	want := []string{
		"Applicable/g1/instance/i1",
		"Applicable/g1/schema/s1",
		"Applicable/g2/instance/i1",
		"Applicable/g3/instance/i3",
		"Applicable/g4/instance/i4",
		"Applicable/g4/instance/i5",
		"Applicable/g5/schema/s5",
		"Scoped/sg1/instance/si1",
	}
	if !slices.Equal(ids, want) {
		t.Errorf("catalog IDs = %q, want %q (sorted by ID, withheld entries included)", ids, want)
	}
}

// TestCatalogNamesOneDocumentFromEveryGroupThatDeclaresIt is the direction the
// relation is one-to-many in: a fixture path is the one thing a census hands
// over, and the catalog names it from as many entries as declare it. A reader
// answering with one ID silently under-reports (issue #1642).
func TestCatalogNamesOneDocumentFromEveryGroupThatDeclaresIt(t *testing.T) {
	entries, err := Catalog(writeCatalogFixture(t, catalogFixtureSuite))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	var naming []string
	for _, e := range entries {
		if slices.Contains(e.Docs, "docs/shared.xml") {
			naming = append(naming, e.ID)
		}
	}
	want := []string{
		"Applicable/g1/instance/i1",
		"Applicable/g2/instance/i1",
		"Applicable/g3/instance/i3",
		"Scoped/sg1/instance/si1",
	}
	if !slices.Equal(naming, want) {
		t.Errorf("entries naming docs/shared.xml = %q, want %q", naming, want)
	}
}

// TestCatalogMarksEveryLevelTheSuiteScopesAway holds the withheld flag to all
// three levels versionApplicable governs, and to nothing else.
func TestCatalogMarksEveryLevelTheSuiteScopesAway(t *testing.T) {
	entries, err := Catalog(writeCatalogFixture(t, catalogFixtureSuite))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	cases := map[string]bool{
		"Applicable/g1/instance/i1": false,
		"Applicable/g1/schema/s1":   false,
		"Applicable/g2/instance/i1": false,
		"Applicable/g3/instance/i3": true, // testGroup version="1.0"
		"Applicable/g4/instance/i4": true, // instanceTest version="1.0"
		"Applicable/g4/instance/i5": false,
		"Scoped/sg1/instance/si1":   true, // testSet version="1.0"
	}
	for id, want := range cases {
		if got := catalogByID(t, entries, id).Withheld; got != want {
			t.Errorf("%s: Withheld = %v, want %v", id, got, want)
		}
	}
}

// TestCatalogSplitsDocumentsUnderTestFromSchemaDocuments holds the two
// document lists apart: a schemaTest's ordered <schemaDocument> list is its
// own Docs, and an instance entry's SchemaDocs are its group's sibling
// schemaTest's list. A group with no single schemaTest contributes none, and a
// schemaTest declaring no document at all yields an entry naming nothing
// rather than no entry.
func TestCatalogSplitsDocumentsUnderTestFromSchemaDocuments(t *testing.T) {
	entries, err := Catalog(writeCatalogFixture(t, catalogFixtureSuite))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	schema := catalogByID(t, entries, "Applicable/g1/schema/s1")
	if want := []string{"docs/s1.xsd", "docs/s1-extra.xsd"}; !slices.Equal(schema.Docs, want) {
		t.Errorf("schemaTest Docs = %q, want %q in document order", schema.Docs, want)
	}
	if len(schema.SchemaDocs) != 0 {
		t.Errorf("schemaTest SchemaDocs = %q, want none: its own Docs are its schema documents", schema.SchemaDocs)
	}

	instance := catalogByID(t, entries, "Applicable/g1/instance/i1")
	if want := []string{"docs/shared.xml"}; !slices.Equal(instance.Docs, want) {
		t.Errorf("instanceTest Docs = %q, want %q", instance.Docs, want)
	}
	if want := []string{"docs/s1.xsd", "docs/s1-extra.xsd"}; !slices.Equal(instance.SchemaDocs, want) {
		t.Errorf("instanceTest SchemaDocs = %q, want its group's schemaTest documents %q", instance.SchemaDocs, want)
	}

	lonely := catalogByID(t, entries, "Applicable/g2/instance/i1")
	if len(lonely.SchemaDocs) != 0 {
		t.Errorf("instance entry in a group with no schemaTest: SchemaDocs = %q, want none", lonely.SchemaDocs)
	}

	docless := catalogByID(t, entries, "Applicable/g5/schema/s5")
	if len(docless.Docs) != 0 || len(docless.SchemaDocs) != 0 {
		t.Errorf("schemaTest declaring no schemaDocument: Docs = %q, SchemaDocs = %q, want none of either",
			docless.Docs, docless.SchemaDocs)
	}
}

// TestCatalogReportsTheDeclaredOutcomeTheRunnerScoresAgainst holds
// DeclaredValid to resolveExpected's reading: an outcome scoped to a
// configuration this processor does not claim prescribes nothing, so its entry
// is not declared valid however the declaration reads.
func TestCatalogReportsTheDeclaredOutcomeTheRunnerScoresAgainst(t *testing.T) {
	entries, err := Catalog(writeCatalogFixture(t, catalogFixtureSuite))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	cases := map[string]bool{
		"Applicable/g1/schema/s1":   true,  // <expected validity="valid"/>
		"Applicable/g1/instance/i1": false, // invalid
		"Applicable/g2/instance/i1": true,  // valid
		"Applicable/g4/instance/i5": false, // valid, but scoped to version="1.0": indeterminate
		"Applicable/g5/schema/s5":   true,
	}
	for id, want := range cases {
		if got := catalogByID(t, entries, id).DeclaredValid; got != want {
			t.Errorf("%s: DeclaredValid = %v, want %v", id, got, want)
		}
	}
}

// TestCatalogRefusesAnAbsentSuite holds the reader to naming the init command
// rather than reporting an empty catalog, which a join would read as "no case
// names this fixture" (issue #659, checkSuitePresent).
func TestCatalogRefusesAnAbsentSuite(t *testing.T) {
	entries, err := Catalog(filepath.Join(t.TempDir(), "no-such-suite"))
	if err == nil {
		t.Fatalf("Catalog over an absent suite = %d entries, nil; want an error", len(entries))
	}
	if want := "git submodule update --init"; !strings.Contains(err.Error(), want) {
		t.Errorf("Catalog error = %q, want it to name %q", err, want)
	}
}

// TestCatalogEnumeratesTheWithheldMissingTestSet is issue #1642's own
// regression case, against the pinned suite: saxonData/Missing/missing001.v1.xml
// is named by an instanceTest in FOUR test groups of saxonMeta/Missing.testSet,
// so one path yields four distinct case IDs — and all four are withheld, the
// testSet declaring version="1.0", so none of them carries a line in any lane.
// Enumerating the catalog and joining against banked cases are two different
// questions, and this path is where a reader that conflates them answers four
// for the second one.
func TestCatalogEnumeratesTheWithheldMissingTestSet(t *testing.T) {
	skipWithoutSuite(t)
	entries, err := Catalog(suiteRoot)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	const doc = "saxonData/Missing/missing001.v1.xml"
	var naming []string
	for _, e := range entries {
		if !slices.Contains(e.Docs, doc) {
			continue
		}
		naming = append(naming, e.ID)
		if !e.Withheld {
			t.Errorf("%s: Withheld = false, want true: Missing.testSet declares version=\"1.0\"", e.ID)
		}
	}
	want := []string{
		"Missing/missing001/instance/missing001.v1.xml",
		"Missing/missing002/instance/missing001.v1.xml",
		"Missing/missing003/instance/missing003.v1.xml",
		"Missing/missing006/instance/missing006.v1.xml",
	}
	if !slices.Equal(naming, want) {
		t.Errorf("entries naming %s = %q, want %q", doc, naming, want)
	}

	for _, id := range naming {
		for _, l := range defaultLanes() {
			banked, err := LoadExpectations(laneFile(l.name))
			if err != nil {
				t.Fatalf("lane %s: %v", l.name, err)
			}
			if _, ok := banked[id]; ok {
				t.Errorf("%s carries a line in lane %s: a withheld case can never flip a score (#1412)", id, l.name)
			}
		}
	}
}

// idsMissingFrom reports the IDs of a that b does not carry. Both are sorted,
// so one linear pass suffices and the result is in ID order (STYLE D1).
func idsMissingFrom(a, b []string) []string {
	var out []string
	i := 0
	for _, id := range a {
		for i < len(b) && b[i] < id {
			i++
		}
		if i < len(b) && b[i] == id {
			continue
		}
		out = append(out, id)
	}
	return out
}

// wantSameIDs fails unless two sorted ID lists are equal, naming the two
// lengths and the first few IDs only one side carries. The lists run to tens
// of thousands of entries against the pinned suite, so printing them whole
// would bury the divergence the assertion exists to expose.
func wantSameIDs(t *testing.T, what string, got, want []string) {
	t.Helper()
	if slices.Equal(got, want) {
		return
	}
	const show = 5
	onlyGot := idsMissingFrom(got, want)
	onlyWant := idsMissingFrom(want, got)
	t.Errorf("%s: Catalog reports %d ID(s), discovery %d; only in Catalog: %q; only in discovery: %q",
		what, len(got), len(want),
		onlyGot[:min(len(onlyGot), show)], onlyWant[:min(len(onlyWant), show)])
}

// TestCatalogAgreesWithDiscoveryOverThePinnedSuite pins the reader's traversal
// to the runner's own, which nothing else does: caseID, resolveDocs,
// groupSchemaDocs, versionApplicable and resolveExpected are shared, so they
// pin each entry's FACTS, but which entries exist and which of them are
// withheld is decided a second time in catalogFromSet. A drift there is
// silent, and what it produces is a lane figure an arbiter quotes (#1642).
//
// The two sets are asserted separately because they answer different halves:
// an entry Catalog fails to mark withheld would otherwise cancel out against
// one it wrongly marks.
func TestCatalogAgreesWithDiscoveryOverThePinnedSuite(t *testing.T) {
	skipWithoutSuite(t)
	d, err := parseSuite(suitePath())
	if err != nil {
		t.Fatalf("parseSuite: %v", err)
	}
	entries, err := Catalog(suiteRoot)
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}

	var produced, withheld []string
	for _, e := range entries {
		if e.Withheld {
			withheld = append(withheld, e.ID)
			continue
		}
		produced = append(produced, e.ID)
	}
	var cases []string
	for _, c := range d.cases {
		cases = append(cases, c.id)
	}
	wantSameIDs(t, "entries Catalog does not withhold, against the cases discovery produces", produced, cases)
	wantSameIDs(t, "entries Catalog withholds, against the IDs discovery withholds", withheld, d.withheld)
}

// TestCatalogRefusesAnApplicableEntryDeclaringNoOutcome holds the reader to
// makeCase's own refusal. DeclaredValid has no encoding for "nothing
// declared", so reporting such an entry hands the join a fabricated outcome
// that decides the #1561 subtraction; and discovery refuses the same entry, so
// describing it would also part the two ID sets.
func TestCatalogRefusesAnApplicableEntryDeclaringNoOutcome(t *testing.T) {
	entries, err := Catalog(writeCatalogFixture(t, outcomelessSuite("")))
	if err == nil {
		t.Fatalf("Catalog over an entry declaring no outcome = %d entries, nil; want an error", len(entries))
	}
	if want := `case "Outcomeless/g1/instance/i1" has no declared expected validity`; !strings.Contains(err.Error(), want) {
		t.Errorf("Catalog error = %q, want it to carry %q: the wrapping names the test SET, so only "+
			"this clause names the entry refused (#1048)", err, want)
	}
}

// TestCatalogDescribesAWithheldEntryDeclaringNoOutcome is the other half of
// that refusal, and the half the equivalence with discovery rests on:
// casesFromSet withholds a scoped test without reading its declaration at all,
// recording its ID, so a reader refusing it would report fewer withheld
// entries than the runner withholds.
func TestCatalogDescribesAWithheldEntryDeclaringNoOutcome(t *testing.T) {
	entries, err := Catalog(writeCatalogFixture(t, outcomelessSuite("1.0")))
	if err != nil {
		t.Fatalf("Catalog: %v", err)
	}
	e := catalogByID(t, entries, "Outcomeless/g1/instance/i1")
	if !e.Withheld {
		t.Errorf("%s: Withheld = false, want true: the instanceTest declares version=\"1.0\"", e.ID)
	}
	if e.DeclaredValid {
		t.Errorf("%s: DeclaredValid = true, want false: the entry declares no outcome at all", e.ID)
	}
}
