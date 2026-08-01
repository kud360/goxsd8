package conformance

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// This file is the M1 harness seam (issue #6): it discovers the W3C suite's
// cases from suite.xml (and its auxiliary extra-suite.xml sibling, which carries
// the precisionDecimal test sets, issue #135) and routes each to exactly one
// lane's executor. It is
// test-only support code — nothing outside package conformance references it —
// so it exports nothing; a later milestone wires in a real lane by extending
// defaultLanes, never by touching the runner's control flow.
//
// # Suite shape
//
// suite.xml is a testSuite whose testSetRef children xlink:href relative
// paths to testSet documents (.testSet or .xml). Each testSet holds testGroup
// elements; each group carries schemaTest and/or instanceTest children. A
// schemaTest/instanceTest has a name, its document reference(s)
// (xlink:href to the document under test), and one or more expected children
// declaring validity ("valid"|"invalid"|"indeterminate", plus rarer spellings
// xsts.dtd allows), optionally qualified by a version. An
// instanceTest names ONE instanceDocument; a schemaTest may name SEVERAL
// schemaDocuments, an ordered set to be loaded "one by one, in order"
// (testdata/xsdtests/common/xsts.xsd, the suite's own catalog schema), so
// discovery keeps every one of them (caseSpec.doc plus caseSpec.extraDocs).
//
// # Case IDs
//
// A case ID is `<testSet-name>/<testGroup-name>/<kind>/<test-name>` where kind
// is "schema" or "instance". The kind segment keeps a group's schemaTest and
// instanceTest of the same name distinct; IDs are asserted unique across the
// whole suite (parseSuite errors on a collision) so an expectation-file line
// maps to exactly one case. Discovery output is sorted by ID (STYLE D1/D2).

// suiteModulePath is the pinned W3C submodule directory as written from the
// module root (conformance/doc.go) — the path `git submodule update --init`
// takes. suiteRoot is the same directory as `go test` sees it: the package
// directory is the working directory, so it is one level up. One fact, one
// encoding (STYLE D3): the module-relative path is the fact.
const (
	suiteModulePath = "testdata/xsdtests"
	suiteRoot       = "../" + suiteModulePath
)

// expectationsDir holds the committed per-lane expectation files.
const expectationsDir = "testdata/expectations"

// suitePath is the suite index whose absence means the submodule is not
// initialized.
func suitePath() string { return filepath.Join(suiteRoot, "suite.xml") }

// suiteOptionalEnv names the explicit opt-out for an environment that
// legitimately has no suite checkout (issue #309): GOXSD_SUITE_OPTIONAL=1 turns
// the missing-suite failure back into a skip for a READ-ONLY run only. It never
// applies to a ratchet run, so an empty suite can never report "no movement".
const suiteOptionalEnv = "GOXSD_SUITE_OPTIONAL"

// checkSuitePresent reports whether the suite index is usable, naming the
// submodule-init command when it is not. A missing suite is a run-ending
// condition for TestConformance, never a silent skip (issue #309): a suite that
// executed zero cases must not be indistinguishable from a green run.
func checkSuitePresent(index string) error {
	if _, err := os.Stat(index); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("W3C suite not present at %s; run `git submodule update --init %s` from the module root", index, suiteModulePath)
		}
		return fmt.Errorf("stat suite index %s: %w", index, err)
	}
	return nil
}

// caseSpec is one discovered conformance case: a stable ID, its kind, the
// resolved path to the document under test, the resolved paths of any FURTHER
// documents the case declares, and the suite's declared XSD 1.1 expectation. An
// executor reads doc, extraDocs and expect to observe a Status; the M1 stub
// ignores them and always reports Fail. Fields are unexported and set only by
// discovery (STYLE T1); nothing derivable is stored (STYLE D3).
//
// expect is THREE-valued, not a bool (issue #277). [validity] is a genuine
// three-valued PSVI outcome in XSD 1.1 Structures (§3.2.5.1/§3.3.5.1:
// valid/invalid/notKnown) and §5.2 states outright that "schema validity is not
// a binary predicate"; the suite's own catalog DTD
// (testdata/xsdtests/wgMeta/ancillary/xsts.dtd) matches that by declaring
// `indeterminate` as a category DISJOINT from valid|invalid — a case the Working
// Group deliberately left undecided. An indeterminate case is therefore DECLINED:
// runLane never dispatches it to an executor and always records Fail(), the
// codebase's standing "known/recorded gap" convention. No executor's answer,
// right or wrong, can earn a pass on a case that has no agreed right answer.
//
// extraDocs is empty for every instanceTest and for the single-document
// schemaTest that is the overwhelming majority. It is non-empty only for a
// schemaTest declaring several <schemaDocument> children, which xsts.xsd (the
// suite's own catalog schema) defines as "run as if the schema documents given
// were loaded one by one, in order" — so the whole list, in document order, is
// the case, not any one member of it. execSchemaCase decides such a case only
// when every extra document is provably part of the closure the parser itself
// walks from doc; see conformance/schema.go.
type caseSpec struct {
	id        string
	kind      string
	doc       string
	extraDocs []string
	expect    expectation
}

// expectation is the suite's declared XSD 1.1 outcome for one case, as a closed
// three-valued set (STYLE T7): the document is expected to be valid, expected to
// be invalid, or declared indeterminate — the Working Group could not agree on
// one right answer. It is a struct so values are constructed ONLY via
// expectValid, expectInvalid and expectIndeterminate, and so "valid or invalid"
// and "indeterminate" cannot be encoded as an illegal combination of two fields
// (STYLE D3: one fact, one encoding). The zero expectation is indeterminate, so
// a caseSpec built without a declared outcome declines rather than being silently
// scored against "invalid".
type expectation struct {
	outcome outcomeKind
}

// outcomeKind enumerates the three declared outcomes. Indeterminate is zero so
// it is the safe default (see expectation).
type outcomeKind uint8

const (
	outcomeIndeterminate outcomeKind = iota
	outcomeValid
	outcomeInvalid
)

// expectValid is the expectation for a suite case declared valid.
func expectValid() expectation { return expectation{outcome: outcomeValid} }

// expectInvalid is the expectation for a suite case declared invalid.
func expectInvalid() expectation { return expectation{outcome: outcomeInvalid} }

// expectIndeterminate is the expectation for a suite case the Working Group left
// undecided; runLane declines such a case without running an executor.
func expectIndeterminate() expectation { return expectation{outcome: outcomeIndeterminate} }

// wantsValid reports whether the suite declared the document valid. It is the
// derived bool view executors compare their observation against (STYLE D3), and
// is meaningful only for a DISPATCHED case: runLane never dispatches an
// indeterminate case, so an executor only ever sees the two real outcomes.
func (e expectation) wantsValid() bool { return e.outcome == outcomeValid }

// isIndeterminate reports whether the suite left this case undecided.
func (e expectation) isIndeterminate() bool { return e.outcome == outcomeIndeterminate }

// The two case kinds. A schemaTest asserts schema-document validity; an
// instanceTest asserts an instance document's validity against its schema.
const (
	kindSchema   = "schema"
	kindInstance = "instance"
)

// executor runs one case and reports whether this processor's observed outcome
// agrees with the suite's declared expectation (Pass) or not (Fail). A real
// engine arrives in a later milestone; until then stubFail records every case
// as a known gap so nothing surfaces as a spurious pass (acceptance #2).
type executor func(caseSpec) Status

// lane is one conformance lane: the subset of suite cases its selector claims,
// executed by exec and ratcheted against
// conformance/testdata/expectations/<name>.txt. Lanes are ordered and a case
// routes to the first lane that claims it, so lanes are disjoint. A later
// milestone activates a lane by giving it a real selector and exec in
// defaultLanes; the runner never changes (issue #6 seam, STYLE T2).
type lane struct {
	name    string
	selects func(caseSpec) bool
	exec    executor
}

// stubFail is the placeholder executor: no engine exists yet, so every case is
// a recorded gap (acceptance #2).
func stubFail(caseSpec) Status { return Fail() }

// selectsNone claims no cases; a lane awaiting its milestone uses it so its
// expectation file stays an empty lane.
func selectsNone(caseSpec) bool { return false }

// selectsKind claims every case of the given kind.
func selectsKind(k string) func(caseSpec) bool {
	return func(c caseSpec) bool { return c.kind == k }
}

// defaultLanes is the committed lane table, one lane per expectation file in
// conformance/doc.go order. Only schema and instance claim cases at M1; the
// remaining lanes are inert (selectsNone) until their milestone gives them a
// selector and executor here. Routing is first-match, so a milestone that
// inserts a narrower lane ahead of schema/instance reroutes those cases
// without editing the runner.
func defaultLanes() []lane {
	return []lane{
		{name: "datatypes", selects: selectsDatatypes, exec: newDatatypesExec()},
		{name: "schema", selects: selectsKind(kindSchema), exec: newSchemaExec()},
		{name: "instance", selects: selectsKind(kindInstance), exec: stubFail},
		{name: "xpath", selects: selectsNone, exec: stubFail},
		{name: "json", selects: selectsNone, exec: stubFail},
		{name: "ber", selects: selectsNone, exec: stubFail},
	}
}

// laneFile is the committed expectation file for a lane.
func laneFile(name string) string {
	return filepath.Join(expectationsDir, name+".txt")
}

// runLane executes every case the lane claims and returns the observed status
// keyed by case ID. The map is an internal lookup for Compare/Ratchet, never
// iterated into output (STYLE D2).
//
// A case the suite declared indeterminate is DECLINED here, before any executor
// runs (issue #277): it is recorded Fail() — the codebase's "known/recorded gap"
// status — and l.exec is not called at all. Declining at this seam rather than
// inside each executor is what makes the guarantee total: a case with no agreed
// right answer cannot be scored a pass by any executor, present or future,
// however it happens to decide the document.
func runLane(l lane, cases []caseSpec) map[string]Status {
	actual := map[string]Status{}
	for _, c := range cases {
		if !l.selects(c) {
			continue
		}
		if c.expect.isIndeterminate() {
			actual[c.id] = Fail()
			continue
		}
		actual[c.id] = l.exec(c)
	}
	return actual
}

// suiteIndex mirrors the testSuite root of suite.xml; only testSetRef hrefs
// matter to discovery.
type suiteIndex struct {
	Refs []testSetRef `xml:"testSetRef"`
}

type testSetRef struct {
	Href string `xml:"http://www.w3.org/1999/xlink href,attr"`
}

// testSet mirrors a referenced testSet document. XMLName is intentionally
// omitted so a set file with an unexpected root decodes to zero groups rather
// than erroring the whole run.
type testSet struct {
	Name   string      `xml:"name,attr"`
	Groups []testGroup `xml:"testGroup"`
}

type testGroup struct {
	Name          string         `xml:"name,attr"`
	SchemaTests   []validityTest `xml:"schemaTest"`
	InstanceTests []validityTest `xml:"instanceTest"`
}

// validityTest mirrors a schemaTest or instanceTest. Only the refs of the
// matching kind are populated; makeCase selects them.
//
// SchemaDocs is a SLICE because a schemaTest may declare ANY NUMBER of
// <schemaDocument> children (xsts.xsd, the suite's own catalog schema), and
// encoding/xml overwrites a non-slice field on each repeated match — so a scalar
// here silently kept only the LAST of them and decided the case against an
// arbitrary document. A slice keeps all of them in document order, which is the
// order xsts.xsd makes significant. InstanceDoc stays scalar: an instanceTest
// declares exactly one <instanceDocument>.
type validityTest struct {
	Name        string     `xml:"name,attr"`
	SchemaDocs  []docRef   `xml:"schemaDocument"`
	InstanceDoc docRef     `xml:"instanceDocument"`
	Expected    []expected `xml:"expected"`
}

type docRef struct {
	Href string `xml:"http://www.w3.org/1999/xlink href,attr"`
}

type expected struct {
	Validity string `xml:"validity,attr"`
	Version  string `xml:"version,attr"`
}

// parseSuite discovers every case reachable from the suite index (and its
// auxiliary extra-suite sibling), sorted by ID (STYLE D1). It errors on a
// malformed reference, an unreadable set, a case with no declared expectation,
// or a duplicate case ID.
func parseSuite(indexPath string) ([]caseSpec, error) {
	seen := map[string]struct{}{}
	seenSets := map[string]struct{}{}
	var cases []caseSpec
	for _, index := range suiteIndexPaths(indexPath) {
		found, err := casesFromIndex(index, seen, seenSets)
		if err != nil {
			return nil, err
		}
		cases = append(cases, found...)
	}
	slices.SortFunc(cases, func(a, b caseSpec) int { return strings.Compare(a.id, b.id) })
	return cases, nil
}

// suiteIndexPaths returns the discovery indices rooted at primary: the primary
// suite index, plus the auxiliary extra-suite.xml sibling when it is present.
// The auxiliary index carries the precisionDecimal test sets (saxonData/PDecimal,
// ibmData/D3_3_4), which the W3C suite moved out of the main suite.xml when
// precisionDecimal was withdrawn from XSD 1.1 but retained as a Working Group
// Note (extra-suite.xml's own documentation). goxsd8 implements precisionDecimal
// as an implementation-defined primitive (xsd-precisionDecimal.md; strict #115,
// maxScale/minScale #133), so those cases are in scope. A missing auxiliary index
// is not an error — discovery falls back to the primary index alone.
func suiteIndexPaths(primary string) []string {
	paths := []string{primary}
	extra := filepath.Join(filepath.Dir(primary), "extra-suite.xml")
	if _, err := os.Stat(extra); err == nil {
		paths = append(paths, extra)
	}
	return paths
}

// casesFromIndex discovers every case reachable from one suite index, sharing
// the suite-wide seen (case-ID uniqueness) and seenSets (resolved set-path
// de-duplication) maps with any sibling index. A test set referenced by more
// than one index — common/introspection.testSet is listed in both suite.xml and
// extra-suite.xml — is processed by the FIRST index to reach it, so its cases are
// discovered once rather than surfacing as a spurious duplicate-ID error.
func casesFromIndex(indexPath string, seen, seenSets map[string]struct{}) ([]caseSpec, error) {
	idx, err := decodeSuiteIndex(indexPath)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(indexPath)
	var cases []caseSpec
	for _, ref := range idx.Refs {
		if ref.Href == "" {
			continue
		}
		setPath := filepath.Join(baseDir, filepath.FromSlash(ref.Href))
		if _, done := seenSets[setPath]; done {
			continue
		}
		seenSets[setPath] = struct{}{}
		set, err := decodeTestSet(setPath)
		if err != nil {
			return nil, fmt.Errorf("test set %s: %w", ref.Href, err)
		}
		found, err := casesFromSet(set, filepath.Dir(setPath), seen)
		if err != nil {
			return nil, fmt.Errorf("test set %s: %w", ref.Href, err)
		}
		cases = append(cases, found...)
	}
	return cases, nil
}

// casesFromSet flattens one testSet into cases, recording each ID in seen to
// enforce suite-wide uniqueness.
func casesFromSet(set testSet, setDir string, seen map[string]struct{}) ([]caseSpec, error) {
	var out []caseSpec
	for _, g := range set.Groups {
		for _, st := range g.SchemaTests {
			c, err := makeCase(set.Name, g.Name, kindSchema, st, setDir, seen)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
		}
		for _, it := range g.InstanceTests {
			c, err := makeCase(set.Name, g.Name, kindInstance, it, setDir, seen)
			if err != nil {
				return nil, err
			}
			out = append(out, c)
		}
	}
	return out, nil
}

// makeCase builds one caseSpec, resolving its document path(s) relative to the
// set directory and its XSD 1.1 expected validity. A schemaTest's FIRST
// <schemaDocument> is the case's doc and the rest, in document order, are its
// extraDocs; an instanceTest has its one <instanceDocument> and no extras.
func makeCase(setName, groupName, kind string, t validityTest, setDir string, seen map[string]struct{}) (caseSpec, error) {
	id := setName + "/" + groupName + "/" + kind + "/" + t.Name
	if _, dup := seen[id]; dup {
		return caseSpec{}, fmt.Errorf("duplicate case id %q", id)
	}
	want, ok := resolveExpected(t.Expected)
	if !ok {
		return caseSpec{}, fmt.Errorf("case %q has no declared expected validity", id)
	}
	href, extra, err := caseDocs(kind, t, setDir)
	if err != nil {
		return caseSpec{}, fmt.Errorf("case %q: %w", id, err)
	}
	seen[id] = struct{}{}
	return caseSpec{
		id:        id,
		kind:      kind,
		doc:       filepath.Join(setDir, filepath.FromSlash(href)),
		extraDocs: extra,
		expect:    want,
	}, nil
}

// caseDocs returns the href of the document under test and the set-relative
// resolved paths of any further declared documents, in document order (STYLE
// D1: no map iteration, the catalog's own order is the fact). A schemaTest with
// no <schemaDocument> at all names nothing to test, which is a malformed catalog
// entry rather than a case this harness may silently invent a document for.
func caseDocs(kind string, t validityTest, setDir string) (href string, extra []string, err error) {
	if kind == kindInstance {
		return t.InstanceDoc.Href, nil, nil
	}
	if len(t.SchemaDocs) == 0 {
		return "", nil, fmt.Errorf("schemaTest declares no schemaDocument")
	}
	for _, d := range t.SchemaDocs[1:] {
		extra = append(extra, filepath.Join(setDir, filepath.FromSlash(d.Href)))
	}
	return t.SchemaDocs[0].Href, extra, nil
}

// resolveExpected picks the declaration that applies to an XSD 1.1 processor and
// classifies it: an explicit version="1.1" declaration wins, else an unversioned
// one (applies to all versions), else the first declaration deterministically.
// Precedence is decided by the version attribute ALONE and before the validity
// is looked at, so a version="1.1" declaration wins over an unversioned one
// whatever either says. ok is false only when no expected element is present.
func resolveExpected(exps []expected) (expectation, bool) {
	unversioned := -1
	for i := range exps {
		if exps[i].Version == "1.1" {
			return classifyValidity(exps[i].Validity), true
		}
		if exps[i].Version == "" && unversioned < 0 {
			unversioned = i
		}
	}
	if unversioned >= 0 {
		return classifyValidity(exps[unversioned].Validity), true
	}
	if len(exps) > 0 {
		return classifyValidity(exps[0].Validity), true
	}
	return expectation{}, false
}

// classifyValidity maps one @validity token to the outcome the harness scores
// against. "indeterminate" is its OWN outcome, not a synonym for invalid (issue
// #277): xsts.dtd declares it disjoint from valid|invalid|notKnown|
// runtime-schema-error, and there is no spec basis for equating it with invalid —
// XSD 1.1 Structures §3.2.5.1/§3.3.5.1 make [validity] three-valued and §5.2 says
// "schema validity is not a binary predicate" and that "there is no requirement
// that input which is not schema-valid be rejected". Folding it into invalid
// scored a case the Working Group left undecided as a PASS whenever this
// processor happened to reject the document, for any reason, right or wrong.
// Declining it instead is a harness-scoring convention, not a spec requirement:
// no rule says a processor must not decide such a document, only that the suite
// cannot judge the answer.
//
// Every other token — "invalid", plus the catalog's rarer
// notKnown/runtime-schema-error/implementation-defined/implementation-dependent/
// invalid-latent spellings — keeps the pre-existing "not valid" reading. Giving
// those their own treatment is separate work; this function is the one place to
// do it when it is grounded.
func classifyValidity(v string) expectation {
	switch v {
	case "valid":
		return expectValid()
	case "indeterminate":
		return expectIndeterminate()
	default:
		return expectInvalid()
	}
}

// decodeSuiteIndex streams the suite index into its struct (STYLE P4: the XML
// decoder reads tokens from the file, never buffering the raw document).
func decodeSuiteIndex(path string) (suiteIndex, error) {
	f, err := os.Open(path)
	if err != nil {
		return suiteIndex{}, fmt.Errorf("opening suite index %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	var idx suiteIndex
	if err := xml.NewDecoder(bufio.NewReader(f)).Decode(&idx); err != nil {
		return suiteIndex{}, fmt.Errorf("decoding suite index %s: %w", path, err)
	}
	return idx, nil
}

// decodeTestSet streams one testSet document into its struct.
func decodeTestSet(path string) (testSet, error) {
	f, err := os.Open(path)
	if err != nil {
		return testSet{}, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect the parsed result
	var set testSet
	if err := xml.NewDecoder(bufio.NewReader(f)).Decode(&set); err != nil {
		return testSet{}, fmt.Errorf("decoding %s: %w", path, err)
	}
	return set, nil
}
