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
// cases from suite.xml and routes each to exactly one lane's executor. It is
// test-only support code — nothing outside package conformance references it —
// so it exports nothing; a later milestone wires in a real lane by extending
// defaultLanes, never by touching the runner's control flow.
//
// # Suite shape
//
// suite.xml is a testSuite whose testSetRef children xlink:href relative
// paths to testSet documents (.testSet or .xml). Each testSet holds testGroup
// elements; each group carries schemaTest and/or instanceTest children. A
// schemaTest/instanceTest has a name, a schemaDocument/instanceDocument child
// (xlink:href to the document under test), and one or more expected children
// declaring validity ("valid"|"invalid"), optionally qualified by a version.
//
// # Case IDs
//
// A case ID is `<testSet-name>/<testGroup-name>/<kind>/<test-name>` where kind
// is "schema" or "instance". The kind segment keeps a group's schemaTest and
// instanceTest of the same name distinct; IDs are asserted unique across the
// whole suite (parseSuite errors on a collision) so an expectation-file line
// maps to exactly one case. Discovery output is sorted by ID (STYLE D1/D2).

// suiteRoot is the pinned W3C submodule directory. The suite lives at the
// module root (testdata/xsdtests per conformance/doc.go); `go test` runs with
// the package directory as the working directory, so it is one level up.
const suiteRoot = "../testdata/xsdtests"

// expectationsDir holds the committed per-lane expectation files.
const expectationsDir = "testdata/expectations"

// suitePath is the suite index whose absence means the submodule is not
// initialized.
func suitePath() string { return filepath.Join(suiteRoot, "suite.xml") }

// caseSpec is one discovered conformance case: a stable ID, its kind, the
// resolved path to the document under test, and the suite's declared XSD 1.1
// expectation. An executor reads doc and expectValid to observe a Status; the
// M1 stub ignores them and always reports Fail. Fields are unexported and set
// only by discovery (STYLE T1); nothing derivable is stored (STYLE D3).
type caseSpec struct {
	id          string
	kind        string
	doc         string
	expectValid bool
}

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
		{name: "schema", selects: selectsKind(kindSchema), exec: stubFail},
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
func runLane(l lane, cases []caseSpec) map[string]Status {
	actual := map[string]Status{}
	for _, c := range cases {
		if !l.selects(c) {
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

// validityTest mirrors a schemaTest or instanceTest. Exactly one document ref
// is populated per kind; makeCase selects the right one.
type validityTest struct {
	Name        string     `xml:"name,attr"`
	SchemaDoc   docRef     `xml:"schemaDocument"`
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

// parseSuite discovers every case reachable from the suite index, sorted by ID
// (STYLE D1). It errors on a malformed reference, an unreadable set, a case
// with no declared expectation, or a duplicate case ID.
func parseSuite(indexPath string) ([]caseSpec, error) {
	idx, err := decodeSuiteIndex(indexPath)
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Dir(indexPath)
	seen := map[string]struct{}{}
	var cases []caseSpec
	for _, ref := range idx.Refs {
		if ref.Href == "" {
			continue
		}
		setPath := filepath.Join(baseDir, filepath.FromSlash(ref.Href))
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
	slices.SortFunc(cases, func(a, b caseSpec) int { return strings.Compare(a.id, b.id) })
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

// makeCase builds one caseSpec, resolving its document path relative to the
// set directory and its XSD 1.1 expected validity.
func makeCase(setName, groupName, kind string, t validityTest, setDir string, seen map[string]struct{}) (caseSpec, error) {
	id := setName + "/" + groupName + "/" + kind + "/" + t.Name
	if _, dup := seen[id]; dup {
		return caseSpec{}, fmt.Errorf("duplicate case id %q", id)
	}
	valid, ok := resolveExpected(t.Expected)
	if !ok {
		return caseSpec{}, fmt.Errorf("case %q has no declared expected validity", id)
	}
	href := t.SchemaDoc.Href
	if kind == kindInstance {
		href = t.InstanceDoc.Href
	}
	seen[id] = struct{}{}
	return caseSpec{
		id:          id,
		kind:        kind,
		doc:         filepath.Join(setDir, filepath.FromSlash(href)),
		expectValid: valid,
	}, nil
}

// resolveExpected picks the validity that applies to an XSD 1.1 processor:
// an explicit version="1.1" declaration wins, else an unversioned one (applies
// to all versions), else the first declaration deterministically. ok is false
// only when no expected element is present.
func resolveExpected(exps []expected) (valid bool, ok bool) {
	unversioned := -1
	for i := range exps {
		if exps[i].Version == "1.1" {
			return exps[i].Validity == "valid", true
		}
		if exps[i].Version == "" && unversioned < 0 {
			unversioned = i
		}
	}
	if unversioned >= 0 {
		return exps[unversioned].Validity == "valid", true
	}
	if len(exps) > 0 {
		return exps[0].Validity == "valid", true
	}
	return false, false
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
