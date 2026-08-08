package conformance

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
// # Applicability
//
// A testSet, testGroup, schemaTest or instanceTest may carry a `version`
// attribute whose tokens are OR-connected APPLICABILITY filters — "is this test
// for me at all?" — and a level the suite scopes away from this processor yields
// no cases at all (versionApplicable, issue #446); the cases it would have
// produced are recorded as WITHHELD instead (discovery, issue #576). That is a
// different attribute job from `expected/@version`, whose tokens are
// AND-connected and merely pick which declared outcome binds (resolveExpected).
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
	return laneFileIn(expectationsDir, name)
}

// laneFileIn is the ONE construction of a lane's file name (STYLE D3), over the
// directory holding it. ratchetAll takes that directory as an argument so its
// write phase is exercisable against a temp directory rather than only against
// the committed expectations.
func laneFileIn(dir, name string) string {
	return filepath.Join(dir, name+".txt")
}

// ratchetRemovalsEnv names the arbiter's per-lane assertion of how many
// sanctioned applicability removals a ratchet run is expected to bank
// (issue #576):
//
//	GOXSD_RATCHET_REMOVALS=schema=34,instance=65
//
// It is arbiter-only and covers the ratchet path ALONE: TestConformance fails
// outright when it is set without GOXSD_RATCHET=1, on the same reasoning as
// suiteOptionalEnv (issue #309) — an opt-in that changes what the ratchet will
// bank must never half-apply to a read-only run. Absent, every lane asserts the
// zero RemovalAssertion, so any removal at all refuses the merge.
//
// The count is asserted PER LANE because the real figures are per lane: a
// removal drifting from one lane to another cannot net out to a passing total.
const ratchetRemovalsEnv = "GOXSD_RATCHET_REMOVALS"

// removalAssertions resolves one run's per-lane removal assertions. raw and set
// are ratchetRemovalsEnv's os.LookupEnv pair and ratcheting is whether
// GOXSD_RATCHET=1; taking them as arguments keeps the gate a pure decision the
// tests can exercise in both directions.
//
// The gate itself: an unset variable asserts nothing on either path, and a
// variable set WITHOUT the ratchet is an error that ends the run. It is never
// parsed-and-ignored, because a read-only run that accepted the assertion would
// report agreement with a figure it never checked and could not bank.
func removalAssertions(raw string, set, ratcheting bool) (map[string]RemovalAssertion, error) {
	if !set {
		return nil, nil
	}
	if !ratcheting {
		return nil, fmt.Errorf(
			"%s is set but GOXSD_RATCHET=1 is not: asserting sanctioned removals is arbiter-only and never applies to a read-only run",
			ratchetRemovalsEnv)
	}
	return parseRemovalAssertions(raw)
}

// parseRemovalAssertions parses ratchetRemovalsEnv's value into one
// RemovalAssertion per named lane. Lanes it does not name assert nothing.
//
// Every malformed spelling is an error rather than a skipped entry: an assertion
// nothing reads — a typo'd lane name, a repeated lane, a count that is not a
// non-negative number — would let a run appear to have asserted a figure while
// the lane it meant still refuses (or, worse, still banks against the zero
// assertion). The map is an internal lookup keyed by lane, never iterated into
// output (STYLE D2).
func parseRemovalAssertions(raw string) (map[string]RemovalAssertion, error) {
	out := map[string]RemovalAssertion{}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		name, count, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("entry %q: want `<lane>=<count>`", entry)
		}
		name = strings.TrimSpace(name)
		if !isLaneName(name) {
			return nil, fmt.Errorf("entry %q: no lane is named %q", entry, name)
		}
		if _, dup := out[name]; dup {
			return nil, fmt.Errorf("entry %q: lane %q is asserted twice", entry, name)
		}
		n, err := strconv.Atoi(strings.TrimSpace(count))
		if err != nil {
			return nil, fmt.Errorf("entry %q: count: %w", entry, err)
		}
		if n < 0 {
			return nil, fmt.Errorf("entry %q: count %d is negative", entry, n)
		}
		out[name] = AssertRemovals(n)
	}
	return out, nil
}

// isLaneName reports whether name is one of the committed lanes.
func isLaneName(name string) bool {
	return slices.ContainsFunc(defaultLanes(), func(l lane) bool { return l.name == name })
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
	Name    string      `xml:"name,attr"`
	Version string      `xml:"version,attr"`
	Groups  []testGroup `xml:"testGroup"`
}

type testGroup struct {
	Name          string         `xml:"name,attr"`
	Version       string         `xml:"version,attr"`
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
	Version     string     `xml:"version,attr"`
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

// discovery is everything one pass over the suite catalog found: the cases to
// execute, and the IDs of the cases discovery deliberately WITHHELD because the
// suite's own applicability metadata scopes them away from this processor. The
// two are disjoint by construction — a withheld level yields no caseSpec — and
// Compare consumes the pair to tell a sanctioned applicability removal from a
// Vanished regression (issue #576). parseSuite returns both sorted by case ID.
type discovery struct {
	cases    []caseSpec
	withheld []string
}

// withholdTest records the ID of one schemaTest or instanceTest the suite scoped
// away from this processor. withholdGroup and withholdSet record every case the
// coarser levels would have produced, in catalog order; the caller stops
// descending at the level it withheld, so recording is not double-counted.
//
// All three build IDs through caseID — the same construction makeCase uses —
// because Compare matches a withheld ID against a committed expectation by exact
// string, so an ID assembled a second way would silently classify a sanctioned
// removal as a Vanished regression.
//
// The one reading that withholds anything is the suite's OR-connected `version`
// metadata (versionApplicable, issue #446), whose four filter sites in
// casesFromSet are these recorders' only callers. The landing order was
// deliberate — the ratchet had to know how to bank a sanctioned removal, under
// the arbiter's asserted count, before discovery was allowed to make one.
func (d *discovery) withholdTest(setName, groupName, kind, testName string) {
	d.withheld = append(d.withheld, caseID(setName, groupName, kind, testName))
}

func (d *discovery) withholdGroup(setName string, g testGroup) {
	for _, st := range g.SchemaTests {
		d.withholdTest(setName, g.Name, kindSchema, st.Name)
	}
	for _, it := range g.InstanceTests {
		d.withholdTest(setName, g.Name, kindInstance, it.Name)
	}
}

func (d *discovery) withholdSet(set testSet) {
	for _, g := range set.Groups {
		d.withholdGroup(set.Name, g)
	}
}

// absorb merges one nested pass's result into this one, preserving catalog order
// within each list; parseSuite sorts once at the end.
func (d *discovery) absorb(found discovery) {
	d.cases = append(d.cases, found.cases...)
	d.withheld = append(d.withheld, found.withheld...)
}

// parseSuite discovers every case reachable from the suite index (and its
// auxiliary extra-suite sibling), sorted by ID (STYLE D1), alongside the IDs
// discovery withheld as inapplicable. It errors on a malformed reference, an
// unreadable set, a case with no declared expectation, or a duplicate case ID.
func parseSuite(indexPath string) (discovery, error) {
	seen := map[string]struct{}{}
	seenSets := map[string]struct{}{}
	var d discovery
	for _, index := range suiteIndexPaths(indexPath) {
		found, err := casesFromIndex(index, seen, seenSets)
		if err != nil {
			return discovery{}, err
		}
		d.absorb(found)
	}
	slices.SortFunc(d.cases, func(a, b caseSpec) int { return strings.Compare(a.id, b.id) })
	slices.Sort(d.withheld)
	return d, nil
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
func casesFromIndex(indexPath string, seen, seenSets map[string]struct{}) (discovery, error) {
	idx, err := decodeSuiteIndex(indexPath)
	if err != nil {
		return discovery{}, err
	}
	baseDir := filepath.Dir(indexPath)
	var d discovery
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
			return discovery{}, fmt.Errorf("test set %s: %w", ref.Href, err)
		}
		found, err := casesFromSet(set, filepath.Dir(setPath), seen)
		if err != nil {
			return discovery{}, fmt.Errorf("test set %s: %w", ref.Href, err)
		}
		d.absorb(found)
	}
	return d, nil
}

// supportedVersionTokens is the ONE encoding (STYLE D3) of which xsts.xsd
// `version` tokens this processor claims support for. Every applicability
// decision reads it, so no "1.1" literal is repeated at the decode sites.
//
// It holds exactly "1.1". xmlschema11-1.md §4.2.2 fixes the decimal "representing
// the version of XSD supported by the processor" at 1.1 for a processor
// conforming to that specification, and this processor targets that version
// alone. §4.2.2 is borrowed for that ONE fact and nothing else: what §4.2.2
// itself governs is vc:minVersion/vc:maxVersion, the spec-normative conditional
// inclusion of schema-document CONTENT — an unrelated mechanism from the suite's
// `version` attribute, which is harness metadata defined solely by
// testdata/xsdtests/common/xsts.xsd. The two happen to need the same number; do
// not merge their readings.
//
// FEATURE tokens are deliberately NOT in the set. ts:version-info is an open
// list over ts:version-token, so a token need not be a version number at all:
// xsts.xsd:1854-1855 enumerates `restricted-xpath-in-CTA` and
// `full-xpath-in-CTA` as processor FEATURES, and the pinned suite uses
// `full-xpath-in-CTA` on 20 test groups (all in CTA.testSet) and `Unicode_4.0.0`
// on one instanceTest. THE RULING, stated rather than defaulted (issue #446):
// this processor's XPath engine is unlanded (M6/M7), so it supports neither full
// XPath in conditional type assignment nor any declared Unicode version, and
// those tokens are unsupported — the groups carrying only such a token are
// inapplicable and produce no cases. Scoring this processor against a feature it
// has never claimed is precisely the defect the XSD-1.0 groups exhibited, and
// declaring support here to keep the case count up would be the same mistake
// with the sign flipped. When the XPath engine lands, adding its token to this
// slice is the whole change.
var supportedVersionTokens = []string{"1.1"}

// versionApplicable reports whether a level of the suite catalog is applicable to
// this processor, given that level's `version` attribute value.
//
// The tokens are OR-connected: xsts.xsd:1449-1458 (the ts:version-info
// annotation) states that on testSuite, testSet, testGroup, schemaTest and
// instanceTest "the tokens have an implicit or connecting them: if a processor
// configuration supports any of them, the tests included are applicable". One
// supported token is therefore enough, so version="1.0 1.1" IS applicable here
// while version="1.0" is not.
//
// An ABSENT (or whitespace-only) value is applicable to everything, and that is
// its OWN case, not a consequence of the OR: an empty token list cannot satisfy
// "supports any of them", so a bare any-match loop would silently drop the
// overwhelming majority of the suite. `version` is use="optional" at every
// declaration site (xsts.xsd:228 testSuite, :319 testSet, :468 testGroup, :631
// schemaTest, :780 instanceTest, :956 expected) and ts:version-info declares no
// default, so absence carries no token list at all: the suite scopes nothing, and
// nothing is excluded.
//
// This is NOT resolveExpected's job and must not be folded into it. `expected`
// (xsts.xsd:956) is the one declaration site where the connector is an AND, and
// what it decides is WHICH declared outcome binds a processor that already runs
// the case. Different level, different connector, different question.
func versionApplicable(version string) bool {
	tokens := strings.Fields(version)
	if len(tokens) == 0 {
		return true
	}
	for _, tok := range tokens {
		if slices.Contains(supportedVersionTokens, tok) {
			return true
		}
	}
	return false
}

// casesFromSet flattens one testSet into cases, recording each ID in seen to
// enforce suite-wide uniqueness.
//
// A level the suite scopes away from this processor contributes NO CASE: not a
// declined case, not a scored one (issue #446). It is not silent either — every
// case the level would have produced is RECORDED as withheld through
// discovery.withholdSet/withholdGroup/withholdTest (issue #576), so an ID that
// already has a committed expectation classifies as a sanctioned Delta.Removed
// rather than a Vanished regression, and the arbiter banks it only against an
// asserted per-lane count. Withholding at the coarsest level that decided it is
// what keeps the two sets disjoint: the loop stops descending there, so no case
// is both produced and withheld.
//
// The filter runs at every level that carries an OR-connected `version` and that
// this decode shape already exposes — the set, each group, and each
// schemaTest/instanceTest (both are validityTest, so covering both is free).
// Measured at the current submodule pin, that drops 28 test groups in two
// separately-decided categories, 8 scoped to XSD 1.0 only (saxonMeta:
// Missing/missing001..006, VC/vc902, PDecimal/pdecimal001a) and 20 scoped to
// full-xpath-in-CTA only (CTA), plus one XSD-1.0-only testSet (saxonMeta/Missing,
// whose 6 groups are individually scoped the same way), 6 XSD-1.0-only
// schemaTests and 30 non-1.1 instanceTests inside otherwise-applicable groups.
//
// instanceTest is filtered NOW rather than deferred: the instance lane scores
// nothing yet, so this is the cheap moment, and one shared predicate at all four
// levels is less code than a documented exception at one of them.
//
// The testSuite root is deliberately NOT filtered. Neither suite.xml nor
// extra-suite.xml carries `version`, so the guard could never fire, and an
// applicability check able to empty the whole run silently is a hazard this
// harness gains nothing by holding. Re-pinning onto a versioned testSuite root
// is when to add it.
func casesFromSet(set testSet, setDir string, seen map[string]struct{}) (discovery, error) {
	var d discovery
	if !versionApplicable(set.Version) {
		d.withholdSet(set)
		return d, nil
	}
	for _, g := range set.Groups {
		if !versionApplicable(g.Version) {
			d.withholdGroup(set.Name, g)
			continue
		}
		for _, st := range g.SchemaTests {
			if !versionApplicable(st.Version) {
				d.withholdTest(set.Name, g.Name, kindSchema, st.Name)
				continue
			}
			c, err := makeCase(set.Name, g.Name, kindSchema, st, setDir, seen)
			if err != nil {
				return discovery{}, err
			}
			d.cases = append(d.cases, c)
		}
		for _, it := range g.InstanceTests {
			if !versionApplicable(it.Version) {
				d.withholdTest(set.Name, g.Name, kindInstance, it.Name)
				continue
			}
			c, err := makeCase(set.Name, g.Name, kindInstance, it, setDir, seen)
			if err != nil {
				return discovery{}, err
			}
			d.cases = append(d.cases, c)
		}
	}
	return d, nil
}

// makeCase builds one caseSpec, resolving its document path(s) relative to the
// set directory and its XSD 1.1 expected validity. A schemaTest's FIRST
// <schemaDocument> is the case's doc and the rest, in document order, are its
// extraDocs; an instanceTest has its one <instanceDocument> and no extras.
func makeCase(setName, groupName, kind string, t validityTest, setDir string, seen map[string]struct{}) (caseSpec, error) {
	id := caseID(setName, groupName, kind, t.Name)
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

// caseID renders the stable ID of one catalog entry,
// `<testSet>/<testGroup>/<kind>/<test-name>` (see "Case IDs" above). It is the
// ONE construction (STYLE D3): makeCase stamps a produced case with it and
// discovery.withholdTest stamps a withheld one, so the produced and withheld
// sets Compare partitions are comparable by exact string.
func caseID(setName, groupName, kind, testName string) string {
	return setName + "/" + groupName + "/" + kind + "/" + testName
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
