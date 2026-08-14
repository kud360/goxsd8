package conformance

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// fakeCases is a tiny synthetic suite used to exercise the runner's lane
// iteration and ratchet integration without the real submodule.
func fakeCases() []caseSpec {
	return []caseSpec{
		{id: "set/g/schema/a", kind: kindSchema, expect: expectValid()},
		{id: "set/g/schema/b", kind: kindSchema, expect: expectInvalid()},
		{id: "set/g/instance/c", kind: kindInstance, expect: expectValid()},
	}
}

// expectValidity renders one of the two DECIDED outcomes from the bool column a
// table-driven executor test naturally carries (and from its flipped polarity).
// It exists only for those tests; an indeterminate expectation is never
// dispatched to an executor, so no executor test has one to build.
func expectValidity(valid bool) expectation {
	if valid {
		return expectValid()
	}
	return expectInvalid()
}

// passSchema is a fake executor: schema cases pass, everything else fails.
func passSchema(c caseSpec) Status {
	if c.kind == kindSchema {
		return Pass()
	}
	return Fail()
}

// TestResolveExpectedClassifiesDeclaredValidity pins the mapping from a suite
// <expected validity="..."> declaration to the outcome the harness scores against
// (issue #277), across all three real spellings AND across the version-precedence
// rule they interact with.
//
// The load-bearing claim is that "indeterminate" is its OWN outcome: xsts.dtd
// declares it disjoint from valid|invalid, XSD 1.1 Structures makes [validity]
// three-valued (§3.2.5.1/§3.3.5.1) and §5.2 denies that schema validity is a
// binary predicate, so folding it into "invalid" — as this harness did — scored a
// case the Working Group left undecided as a pass whenever this processor
// rejected the document for ANY reason.
//
// The mixed-declaration rows pin how the two rules compose. Precedence is decided
// by @version alone, BEFORE validity is read: a version="1.1" declaration wins
// over an unversioned one whatever either says — so a 1.1 indeterminate beats an
// unversioned valid, and symmetrically a 1.1 valid beats an unversioned
// indeterminate. Declaration order does not matter, because the scan returns on
// the first 1.1 match and only remembers the first unversioned one.
func TestResolveExpectedClassifiesDeclaredValidity(t *testing.T) {
	cases := []struct {
		name string
		exps []expected
		want expectation
		ok   bool
	}{
		{"valid alone", []expected{{Validity: "valid"}}, expectValid(), true},
		{"invalid alone", []expected{{Validity: "invalid"}}, expectInvalid(), true},
		{"indeterminate alone", []expected{{Validity: "indeterminate"}}, expectIndeterminate(), true},
		{
			"a rarer catalog spelling keeps the not-valid reading",
			[]expected{{Validity: "notKnown"}},
			expectInvalid(), true,
		},
		{
			"version 1.1 indeterminate wins over an unversioned valid",
			[]expected{{Validity: "valid"}, {Validity: "indeterminate", Version: "1.1"}},
			expectIndeterminate(), true,
		},
		{
			"version 1.1 indeterminate wins over an unversioned invalid, declared first",
			[]expected{{Validity: "indeterminate", Version: "1.1"}, {Validity: "invalid"}},
			expectIndeterminate(), true,
		},
		{
			"version 1.1 valid wins over an unversioned indeterminate",
			[]expected{{Validity: "indeterminate"}, {Validity: "valid", Version: "1.1"}},
			expectValid(), true,
		},
		{
			"a 1.0-only indeterminate still falls back to the first declaration",
			[]expected{{Validity: "indeterminate", Version: "1.0"}},
			expectIndeterminate(), true,
		},
		{
			"an unversioned declaration beats a versioned non-1.1 one",
			[]expected{{Validity: "invalid", Version: "1.0"}, {Validity: "indeterminate"}},
			expectIndeterminate(), true,
		},
		{"no declaration at all", nil, expectation{}, false},
	}
	for _, tc := range cases {
		got, ok := resolveExpected(tc.exps)
		if ok != tc.ok {
			t.Errorf("%s: ok = %v, want %v", tc.name, ok, tc.ok)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: expectation = %s, want %s", tc.name, expectationName(got), expectationName(tc.want))
		}
	}
}

// TestVersionApplicableUsesOrConnectedTokens pins the APPLICABILITY half of the
// suite's `version` attribute (issue #446) — a different rule from the
// resolveExpected precedence above, not a variant of it. xsts.xsd:1449-1458 gives
// testSuite/testSet/testGroup/schemaTest/instanceTest an implicit OR ("if a
// processor configuration supports any of them, the tests included are
// applicable"); `expected` alone gets an AND.
//
// Three rows carry the design and the rest guard the edges. Absence must be
// applicable — `version` is use="optional" at every declaration site and
// ts:version-info declares no default, and an empty token list satisfies no
// any-match loop, so absence has to be special-cased or discovery loses the
// unversioned majority of the suite. "1.0 1.1" must be applicable, which is what
// distinguishes the OR from an equality test on the whole attribute value. And
// "full-xpath-in-CTA" must NOT be: it is an xsts.xsd:1854-1855 FEATURE token, and
// this processor's XPath engine is unlanded.
func TestVersionApplicableUsesOrConnectedTokens(t *testing.T) {
	cases := []struct {
		name    string
		version string
		want    bool
	}{
		{"absent applies to everything", "", true},
		{"whitespace only is absent", "  \n\t ", true},
		{"the supported version alone", "1.1", true},
		{"XSD 1.0 only", "1.0", false},
		{"1.0 or 1.1 — the OR admits it", "1.0 1.1", true},
		{"1.1 or 1.0, token order irrelevant", "1.1 1.0", true},
		{"full-xpath-in-CTA is an unsupported feature", "full-xpath-in-CTA", false},
		{"restricted-xpath-in-CTA is unsupported too", "restricted-xpath-in-CTA", false},
		{"a supported version beside an unsupported feature", "1.1 full-xpath-in-CTA", true},
		{"two unsupported tokens", "1.0 full-xpath-in-CTA", false},
		{"an unknown token", "Unicode_4.0.0", false},
		{"several unsupported tokens", "1.0 Unicode_4.0.0 restricted-xpath-in-CTA", false},
		{"a supported token last in a long list", "1.0 Unicode_4.0.0 1.1", true},
		{"1.10 is not 1.1", "1.10", false},
		{"a token merely containing 1.1", "x1.1y", false},
	}
	for _, tc := range cases {
		if got := versionApplicable(tc.version); got != tc.want {
			t.Errorf("%s: versionApplicable(%q) = %v, want %v", tc.name, tc.version, got, tc.want)
		}
	}
}

// TestCasesFromSetWithholdsInapplicableLevels proves the filter removes cases
// from DISCOVERY rather than declining them (issue #446): an inapplicable level
// yields no caseSpec at all, so it cannot reach an expectation file even as a
// recorded fail. It exercises every level casesFromSet can see — the set, each
// group, and each schemaTest/instanceTest — on a synthetic catalog, so it needs
// no submodule. The retained rows are chosen so a filter that skipped a level (or
// filtered one it should not) changes the ID list rather than only its length.
//
// Every dropped case must ALSO be recorded withheld (issue #576), because that
// recording is the whole difference between a sanctioned Delta.Removed and a
// Vanished regression that refuses the ratchet. The two lists are asserted
// exactly and asserted DISJOINT: a filter site that skipped without recording, or
// a recorder that fired on a level the loop still descended into, both show up
// here rather than as a ratchet refusal months later.
func TestCasesFromSetWithholdsInapplicableLevels(t *testing.T) {
	vt := func(name, version string) validityTest {
		return validityTest{
			Name:        name,
			Version:     version,
			SchemaDocs:  []docRef{{Href: name + ".xsd"}},
			InstanceDoc: docRef{Href: name + ".xml"},
			Expected:    []expected{{Validity: "valid"}},
		}
	}
	set := testSet{
		Name: "set",
		Groups: []testGroup{
			{
				Name:          "keep",
				SchemaTests:   []validityTest{vt("s-keep", ""), vt("s-drop", "1.0")},
				InstanceTests: []validityTest{vt("i-keep", "1.0 1.1"), vt("i-drop", "1.0")},
			},
			{Name: "dropVersion", Version: "1.0", SchemaTests: []validityTest{vt("s", "1.1")}},
			{Name: "dropFeature", Version: "full-xpath-in-CTA", SchemaTests: []validityTest{vt("s", "")}},
		},
	}

	got, err := casesFromSet(set, "sets/saxonMeta", map[string]struct{}{})
	if err != nil {
		t.Fatalf("casesFromSet: %v", err)
	}
	var ids []string
	for _, c := range got.cases {
		ids = append(ids, c.id)
	}
	want := []string{"set/keep/schema/s-keep", "set/keep/instance/i-keep"}
	if !slices.Equal(ids, want) {
		t.Errorf("discovered %v, want %v", ids, want)
	}
	wantWithheld := []string{
		"set/keep/schema/s-drop",
		"set/keep/instance/i-drop",
		"set/dropVersion/schema/s",
		"set/dropFeature/schema/s",
	}
	if !slices.Equal(got.withheld, wantWithheld) {
		t.Errorf("withheld %v, want %v", got.withheld, wantWithheld)
	}
	assertDisjoint(t, ids, got.withheld)

	scoped := set
	scoped.Version = "1.0"
	dropped, err := casesFromSet(scoped, "sets/saxonMeta", map[string]struct{}{})
	if err != nil {
		t.Fatalf("casesFromSet on an inapplicable set: %v", err)
	}
	if len(dropped.cases) != 0 {
		t.Errorf("an XSD-1.0-only testSet must yield no cases, got %d: %v", len(dropped.cases), dropped.cases)
	}
	wantSetWithheld := []string{
		"set/keep/schema/s-keep",
		"set/keep/schema/s-drop",
		"set/keep/instance/i-keep",
		"set/keep/instance/i-drop",
		"set/dropVersion/schema/s",
		"set/dropFeature/schema/s",
	}
	if !slices.Equal(dropped.withheld, wantSetWithheld) {
		t.Errorf("an XSD-1.0-only testSet must withhold every case it would have produced, got %v, want %v",
			dropped.withheld, wantSetWithheld)
	}
}

// assertDisjoint fails when one case ID appears in both the produced and the
// withheld list. Compare treats that overlap as a runner bug and errors the whole
// run (issue #576), so casesFromSet must never build one.
func assertDisjoint(t *testing.T, produced, withheld []string) {
	t.Helper()
	for _, id := range withheld {
		if slices.Contains(produced, id) {
			t.Errorf("case %q is both produced and withheld", id)
		}
	}
}

// expectationName renders an expectation for a test failure message.
func expectationName(e expectation) string {
	if e.isIndeterminate() {
		return "indeterminate"
	}
	if e.wantsValid() {
		return "valid"
	}
	return "invalid"
}

// TestRunLaneDeclinesIndeterminateWithoutExecuting proves the decline is total
// (issue #277): a case the suite declared indeterminate is recorded Fail() — the
// standing "known/recorded gap" status — and the lane's executor is NOT invoked
// at all. The fake executor passes everything, so a runLane that still dispatched
// would record a pass; and it records every case it is handed, so the assertion
// on the ids it saw cannot be satisfied by an executor that merely returned Fail.
func TestRunLaneDeclinesIndeterminateWithoutExecuting(t *testing.T) {
	var dispatched []string
	l := lane{
		name:    "fake",
		selects: selectsKind(kindSchema),
		exec: func(c caseSpec) Status {
			dispatched = append(dispatched, c.id)
			return Pass()
		},
	}
	cases := []caseSpec{
		{id: "set/g/schema/decided", kind: kindSchema, expect: expectInvalid()},
		{id: "set/g/schema/undecided", kind: kindSchema, expect: expectIndeterminate()},
	}

	actual := runLane(l, cases)
	if len(actual) != 2 {
		t.Fatalf("both claimed cases must be recorded, got %d", len(actual))
	}
	if !actual["set/g/schema/decided"].IsPass() {
		t.Errorf("a decided case must still be scored by the executor")
	}
	if actual["set/g/schema/undecided"].IsPass() {
		t.Errorf("an indeterminate case must be recorded Fail (declined), never a pass")
	}
	if !slices.Equal(dispatched, []string{"set/g/schema/decided"}) {
		t.Errorf("executor saw %v, want only the decided case — an indeterminate case must never be dispatched", dispatched)
	}
}

// TestMakeCaseCarriesIndeterminateThrough proves discovery preserves the third
// outcome end to end: a suite entry declaring validity="indeterminate" yields a
// caseSpec whose expectation reports itself indeterminate, which is what runLane
// declines on.
func TestMakeCaseCarriesIndeterminateThrough(t *testing.T) {
	vt := validityTest{
		Name:       "undecided",
		SchemaDocs: []docRef{{Href: "a.xsd"}},
		Expected:   []expected{{Validity: "indeterminate"}},
	}
	c, err := makeCase("set", kindSchema, vt, testGroup{Name: "g"}, "sets/msMeta", map[string]struct{}{})
	if err != nil {
		t.Fatalf("makeCase: %v", err)
	}
	if !c.expect.isIndeterminate() {
		t.Errorf("caseSpec.expect = %s, want indeterminate", expectationName(c.expect))
	}
}

func TestRunLaneSelectsOnlyClaimedCases(t *testing.T) {
	l := lane{name: "fake", selects: selectsKind(kindSchema), exec: passSchema}
	actual := runLane(l, fakeCases())

	if len(actual) != 2 {
		t.Fatalf("selector must claim the 2 schema cases, got %d: %v", len(actual), actual)
	}
	if _, ok := actual["set/g/instance/c"]; ok {
		t.Errorf("instance case must not be claimed by a schema-kind lane")
	}
	if !actual["set/g/schema/a"].IsPass() {
		t.Errorf("executor result not recorded for claimed case")
	}
}

// TestRunLaneRatchetRoundTrip exercises the runner's integration of runLane
// with Ratchet + WriteExpectations against a temp-dir lane file (never the
// real committed path): a fresh lane starts empty, records observed New cases,
// and reloads byte-stably.
func TestRunLaneRatchetRoundTrip(t *testing.T) {
	l := lane{name: "fake", selects: func(caseSpec) bool { return true }, exec: passSchema}
	actual := runLane(l, fakeCases())

	path := filepath.Join(t.TempDir(), "fake.txt")
	expected, err := LoadExpectations(path) // missing file => empty lane
	if err != nil {
		t.Fatalf("load empty lane: %v", err)
	}
	merged, err := Ratchet(expected, actual, nil, RemovalAssertion{})
	if err != nil {
		t.Fatalf("ratchet must accept all-New cases: %v", err)
	}
	if err := WriteExpectations(path, merged); err != nil {
		t.Fatalf("write: %v", err)
	}

	reloaded, err := LoadExpectations(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	want := map[string]Status{
		"set/g/schema/a":   Pass(),
		"set/g/schema/b":   Pass(),
		"set/g/instance/c": Fail(),
	}
	if len(reloaded) != len(want) {
		t.Fatalf("reloaded %d cases, want %d", len(reloaded), len(want))
	}
	for id, w := range want {
		if reloaded[id] != w {
			t.Errorf("case %q: got %v, want %v", id, reloaded[id], w)
		}
	}
}

// TestRunLaneRatchetRefusesRegression proves the runner surfaces a ratchet
// refusal: a case committed as pass that the lane now fails blocks the merge.
func TestRunLaneRatchetRefusesRegression(t *testing.T) {
	// Lane executor fails the instance case; commit it as an expected pass.
	l := lane{name: "fake", selects: func(caseSpec) bool { return true }, exec: passSchema}
	actual := runLane(l, fakeCases())

	expected := map[string]Status{"set/g/instance/c": Pass()}
	if _, err := Ratchet(expected, actual, nil, RemovalAssertion{}); err == nil {
		t.Fatal("ratchet must refuse when an executor regresses a committed pass")
	}
}

// TestValidityTestKeepsEverySchemaDocument pins the decoding bug this struct's
// slice field exists to prevent (issue #238): encoding/xml OVERWRITES a
// non-slice field on every repeated matching child, so a scalar SchemaDoc kept
// only the LAST <schemaDocument> of a multi-document schemaTest and the harness
// then decided the case against an arbitrary member of the declared set. The
// assertion is on the unmarshal itself, not on any downstream verdict, because
// the loss happened here.
func TestValidityTestKeepsEverySchemaDocument(t *testing.T) {
	const src = `<schemaTest xmlns:xlink="http://www.w3.org/1999/xlink" name="multi">
		<schemaDocument xlink:href="first.xsd"/>
		<schemaDocument xlink:href="second.xsd"/>
		<schemaDocument xlink:href="third.xsd"/>
		<expected validity="valid"/>
	</schemaTest>`

	var vt validityTest
	if err := xml.Unmarshal([]byte(src), &vt); err != nil {
		t.Fatalf("unmarshalling schemaTest: %v", err)
	}

	got := make([]string, 0, len(vt.SchemaDocs))
	for _, d := range vt.SchemaDocs {
		got = append(got, d.Href)
	}
	want := []string{"first.xsd", "second.xsd", "third.xsd"}
	if !slices.Equal(got, want) {
		t.Errorf("SchemaDocs = %v, want every declared href in document order %v", got, want)
	}
}

// TestMakeCaseSplitsSchemaDocuments proves makeCase treats a multi-document
// schemaTest as the ordered SET xsts.xsd declares it to be: the FIRST document is
// the case's doc (the one parser.Parse is rooted at) and every further one lands
// in extraDocs in document order, each resolved against the test set's directory.
// An instanceTest keeps its single document and no extras, and a schemaTest that
// names no document at all is a malformed catalog entry rather than a case with
// an invented document.
func TestMakeCaseSplitsSchemaDocuments(t *testing.T) {
	setDir := filepath.Join("sets", "ibmMeta")
	multi := validityTest{
		Name: "multi",
		SchemaDocs: []docRef{
			{Href: "../ibmData/a.xsd"},
			{Href: "../ibmData/b.xsd"},
			{Href: "../ibmData/c.xsd"},
		},
		Expected: []expected{{Validity: "valid"}},
	}

	c, err := makeCase("set", kindSchema, multi, testGroup{Name: "g"}, setDir, map[string]struct{}{})
	if err != nil {
		t.Fatalf("makeCase: %v", err)
	}
	if want := filepath.Join(setDir, "../ibmData/a.xsd"); c.doc != want {
		t.Errorf("doc = %q, want the FIRST declared document %q", c.doc, want)
	}
	wantExtra := []string{
		filepath.Join(setDir, "../ibmData/b.xsd"),
		filepath.Join(setDir, "../ibmData/c.xsd"),
	}
	if !slices.Equal(c.extraDocs, wantExtra) {
		t.Errorf("extraDocs = %v, want the remaining documents in order %v", c.extraDocs, wantExtra)
	}

	inst := validityTest{
		Name:        "inst",
		InstanceDoc: docRef{Href: "../ibmData/i.xml"},
		Expected:    []expected{{Validity: "valid"}},
	}
	ic, err := makeCase("set", kindInstance, inst, testGroup{Name: "g"}, setDir, map[string]struct{}{})
	if err != nil {
		t.Fatalf("makeCase (instance): %v", err)
	}
	if want := filepath.Join(setDir, "../ibmData/i.xml"); ic.doc != want {
		t.Errorf("instance doc = %q, want %q", ic.doc, want)
	}
	if len(ic.extraDocs) != 0 {
		t.Errorf("instance case must carry no extraDocs, got %v", ic.extraDocs)
	}

	empty := validityTest{Name: "none", Expected: []expected{{Validity: "valid"}}}
	if _, err := makeCase("set", kindSchema, empty, testGroup{Name: "g"}, setDir, map[string]struct{}{}); err == nil {
		t.Error("a schemaTest declaring no schemaDocument must error, not yield a case with an empty document path")
	}
}

// TestMakeCaseCarriesTheGroupSchemaToInstanceCases pins the discovery half of the
// instance lane (issue #713): the catalog names only the instance document, so an
// instanceTest's caseSpec must carry the <schemaDocument> list of its group's
// sibling schemaTest, resolved exactly as a schemaTest's own documents are —
// without it execInstanceCase has no schema to assess against and declines every
// case.
//
// The three rows are the whole rule. Exactly one sibling schemaTest yields the
// reference; a group with none and a group with two yield NOTHING, because every
// instance-bearing group of the pinned suite declares exactly one and any other
// count is a shape with no grounded reading — declining beats picking a sibling.
// A schemaTest's own case carries no schema reference at all, its doc and
// extraDocs being its schema documents already (STYLE D3).
func TestMakeCaseCarriesTheGroupSchemaToInstanceCases(t *testing.T) {
	setDir := filepath.Join("sets", "ibmMeta")
	inst := validityTest{
		Name:        "inst",
		InstanceDoc: docRef{Href: "i.xml"},
		Expected:    []expected{{Validity: "valid"}},
	}
	schemaTest := func(hrefs ...string) validityTest {
		var docs []docRef
		for _, h := range hrefs {
			docs = append(docs, docRef{Href: h})
		}
		return validityTest{Name: "s", SchemaDocs: docs, Expected: []expected{{Validity: "valid"}}}
	}

	one := testGroup{Name: "g", SchemaTests: []validityTest{schemaTest("a.xsd", "b.xsd")}}
	c, err := makeCase("set", kindInstance, inst, one, setDir, map[string]struct{}{})
	if err != nil {
		t.Fatalf("makeCase: %v", err)
	}
	if want := filepath.Join(setDir, "a.xsd"); c.schemaDoc != want {
		t.Errorf("schemaDoc = %q, want the group schemaTest's FIRST document %q", c.schemaDoc, want)
	}
	if want := []string{filepath.Join(setDir, "b.xsd")}; !slices.Equal(c.schemaExtraDocs, want) {
		t.Errorf("schemaExtraDocs = %v, want %v", c.schemaExtraDocs, want)
	}

	for _, tc := range []struct {
		why string
		g   testGroup
	}{
		{"a group declaring no schemaTest", testGroup{Name: "g"}},
		{"a group declaring two schemaTests", testGroup{Name: "g", SchemaTests: []validityTest{schemaTest("a.xsd"), schemaTest("b.xsd")}}},
		{"a group whose schemaTest names no document", testGroup{Name: "g", SchemaTests: []validityTest{schemaTest()}}},
	} {
		c, err := makeCase("set", kindInstance, inst, tc.g, setDir, map[string]struct{}{})
		if err != nil {
			t.Fatalf("%s: makeCase: %v", tc.why, err)
		}
		if c.schemaDoc != "" || len(c.schemaExtraDocs) != 0 {
			t.Errorf("%s: must carry NO schema reference, got %q %v", tc.why, c.schemaDoc, c.schemaExtraDocs)
		}
	}

	sc, err := makeCase("set", kindSchema, schemaTest("a.xsd"), one, setDir, map[string]struct{}{})
	if err != nil {
		t.Fatalf("makeCase (schema): %v", err)
	}
	if sc.schemaDoc != "" || len(sc.schemaExtraDocs) != 0 {
		t.Errorf("a schemaTest case must carry no separate schema reference, got %q %v", sc.schemaDoc, sc.schemaExtraDocs)
	}
}

// TestWithheldIDsUseTheSameConstructionAsProducedCases pins what makes a
// sanctioned applicability removal reach Compare at all (issue #576): the ID a
// withheld level records must be byte-identical to the ID the same catalog entry
// would have produced. Assemble it a second way and Compare would classify the
// removal as a Vanished regression, which is precisely the failure this class
// exists to prevent — so the assertion compares the recorders against
// casesFromSet's own output rather than against a hand-written literal.
func TestWithheldIDsUseTheSameConstructionAsProducedCases(t *testing.T) {
	valid := []expected{{Validity: "valid"}}
	set := testSet{
		Name: "saxonMeta/Missing",
		Groups: []testGroup{
			{
				Name:          "g1",
				SchemaTests:   []validityTest{{Name: "s1", SchemaDocs: []docRef{{Href: "a.xsd"}}, Expected: valid}},
				InstanceTests: []validityTest{{Name: "i1", InstanceDoc: docRef{Href: "a.xml"}, Expected: valid}},
			},
			{
				Name:        "g2",
				SchemaTests: []validityTest{{Name: "s2", SchemaDocs: []docRef{{Href: "b.xsd"}}, Expected: valid}},
			},
		},
	}

	produced, err := casesFromSet(set, "sets", map[string]struct{}{})
	if err != nil {
		t.Fatalf("casesFromSet: %v", err)
	}
	if len(produced.withheld) != 0 {
		t.Fatalf("no level of this fixture declares a version, so nothing may be withheld, got %v", produced.withheld)
	}
	ids := make([]string, 0, len(produced.cases))
	for _, c := range produced.cases {
		ids = append(ids, c.id)
	}
	if len(ids) != 3 {
		t.Fatalf("fixture must produce 3 cases, got %v", ids)
	}

	var wholeSet discovery
	wholeSet.withholdSet(set)
	if !slices.Equal(wholeSet.withheld, ids) {
		t.Errorf("withholdSet recorded %v, want the ids the same set produces %v", wholeSet.withheld, ids)
	}

	var oneGroup discovery
	oneGroup.withholdGroup(set.Name, set.Groups[0])
	if !slices.Equal(oneGroup.withheld, ids[:2]) {
		t.Errorf("withholdGroup recorded %v, want the first group's ids %v", oneGroup.withheld, ids[:2])
	}

	var oneTest discovery
	oneTest.withholdTest(set.Name, "g1", kindSchema, "s1")
	if !slices.Equal(oneTest.withheld, ids[:1]) {
		t.Errorf("withholdTest recorded %v, want %v", oneTest.withheld, ids[:1])
	}
}

// TestRemovalAssertionsAreReachableOnlyFromARatchetRun pins the gating boundary
// (issue #576): the per-lane removal assertion applies on the ratchet path and
// NOWHERE else. A value set without GOXSD_RATCHET=1 ends the run rather than
// being parsed and ignored — the endUnusableSuiteRun rule (issue #309) that an
// opt-in touching what the ratchet banks must never half-work.
func TestRemovalAssertionsAreReachableOnlyFromARatchetRun(t *testing.T) {
	if _, err := removalAssertions("schema=34", true, false); err == nil {
		t.Error("asserting removals without GOXSD_RATCHET=1 must end the run, not half-apply")
	}
	if _, err := removalAssertions("nonsense", true, false); err == nil {
		t.Error("the ratchet gate must be checked before the value is parsed")
	}

	unsetReadOnly, err := removalAssertions("", false, false)
	if err != nil {
		t.Errorf("an unset variable must be fine on a read-only run, got %v", err)
	}
	if len(unsetReadOnly) != 0 {
		t.Errorf("an unset variable must assert nothing, got %v", unsetReadOnly)
	}

	unsetRatchet, err := removalAssertions("", false, true)
	if err != nil {
		t.Errorf("an unset variable must be fine on a ratchet run, got %v", err)
	}
	if unsetRatchet["schema"] != (RemovalAssertion{}) {
		t.Errorf("an unset variable must leave every lane at the zero assertion, got %v", unsetRatchet["schema"])
	}

	asserted, err := removalAssertions("schema=34,instance=65", true, true)
	if err != nil {
		t.Fatalf("a well-formed assertion on a ratchet run: %v", err)
	}
	if asserted["schema"] != AssertRemovals(34) || asserted["instance"] != AssertRemovals(65) {
		t.Errorf("parsed assertions = %v, want schema=34 instance=65", asserted)
	}
}

// TestParseRemovalAssertionsRejectsAnythingThatWouldAssertNothing proves every
// malformed spelling is an error rather than a skipped entry: an assertion the
// ratchet never reads would let a run look like it predicted a count while the
// lane it meant still refuses (or banks against the zero assertion instead).
func TestParseRemovalAssertionsRejectsAnythingThatWouldAssertNothing(t *testing.T) {
	bad := map[string]string{
		"no such lane":        "schemata=1",
		"no count at all":     "schema",
		"non-numeric count":   "schema=lots",
		"negative count":      "schema=-1",
		"lane asserted twice": "schema=1,schema=2",
	}
	for name, raw := range bad {
		t.Run(name, func(t *testing.T) {
			if _, err := parseRemovalAssertions(raw); err == nil {
				t.Fatalf("%q must be rejected", raw)
			}
		})
	}

	got, err := parseRemovalAssertions(" schema = 34 , instance=65 ")
	if err != nil {
		t.Fatalf("surrounding whitespace must be tolerated: %v", err)
	}
	if got["schema"] != AssertRemovals(34) {
		t.Errorf("schema = %v, want 34", got["schema"])
	}
	if got["datatypes"] != (RemovalAssertion{}) {
		t.Errorf("a lane the value does not name must assert nothing, got %v", got["datatypes"])
	}
}

// TestCheckSuitePresent proves the missing-suite gate TestConformance depends
// on (issue #309): an index that does not exist yields an error naming the
// submodule-init command, and a present index yields none. The error is what
// turns a silent skip into a hard failure, so it must be non-nil regardless of
// whether this checkout has the submodule populated.
func TestCheckSuitePresent(t *testing.T) {
	dir := t.TempDir()
	present := filepath.Join(dir, "suite.xml")
	if err := os.WriteFile(present, []byte("<testSuite/>"), 0o600); err != nil {
		t.Fatalf("writing fake suite index: %v", err)
	}

	err := checkSuitePresent(filepath.Join(dir, "no-such-root", "suite.xml"))
	if err == nil {
		t.Fatal("a non-existent suite root must yield an error, not a nil check (a skip would hide an empty run)")
	}
	if !strings.Contains(err.Error(), "git submodule update --init "+suiteModulePath) {
		t.Errorf("error must name the init command, got %q", err)
	}

	if err := checkSuitePresent(present); err != nil {
		t.Errorf("an existing suite index must check clean, got %v", err)
	}
}
