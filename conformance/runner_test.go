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
	c, err := makeCase("set", "g", kindSchema, vt, "sets/msMeta", map[string]struct{}{})
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
	merged, err := Ratchet(expected, actual)
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
	if _, err := Ratchet(expected, actual); err == nil {
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

	c, err := makeCase("set", "g", kindSchema, multi, setDir, map[string]struct{}{})
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
	ic, err := makeCase("set", "g", kindInstance, inst, setDir, map[string]struct{}{})
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
	if _, err := makeCase("set", "g", kindSchema, empty, setDir, map[string]struct{}{}); err == nil {
		t.Error("a schemaTest declaring no schemaDocument must error, not yield a case with an empty document path")
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
