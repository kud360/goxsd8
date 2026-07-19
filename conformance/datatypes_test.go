package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestDatatypesBackendSeeds guards the lane's precondition loudly: the strict
// backend must satisfy builtin.Seed (all primitives mapped), or the executor's
// symbol table is empty and every claimed case fails.
func TestDatatypesBackendSeeds(t *testing.T) {
	backend := strict.New()
	types, err := builtin.Seed(backend)
	if err != nil {
		t.Fatalf("Seed(strict) must succeed, got %v", err)
	}
	if got, want := len(types), len(builtin.Types)+1; got != want {
		t.Fatalf("Seed returned %d components, want %d", got, want)
	}
}

// TestDatatypesSelectorClaimsOnlyCohort proves the selector claims the lexical
// cohort's instance cases and nothing else.
func TestDatatypesSelectorClaimsOnlyCohort(t *testing.T) {
	cases := []struct {
		c    caseSpec
		want bool
	}{
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/decimal017.xml"}, true},
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/boolean001.xml"}, true},
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/string006.xml"}, true},
		// The context-dependent QName/NOTATION lexical cases are claimed (issue #131).
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/QName006.xml"}, true},
		// The NOTATION Facets-cohort cases are claimed by their own selector (issue
		// #153): datatypesCase/facetsCase do not match (NOTATION is not in
		// facetsBaseTypes), but notationFacetsCase does.
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/Facets/NOTATION/NOTATION_enumeration001.xml"}, true},
		// A schema case for the same file is not claimed (we cannot validate schemas).
		{caseSpec{kind: kindSchema, doc: "../testdata/xsdtests/msData/datatypes/decimal.xsd"}, false},
		// A facet-restricted NIST instance is out of the cohort.
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/nistData/atomic/decimal/Schema+Instance/NISTXML-SV-IV-atomic-decimal-minExclusive-1-1.xml"}, false},
		// The derived string family's Facets instances are claimed (issue #85).
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/Facets/token/token_length001.xml"}, true},
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/Facets/normalizedString/normalizedString_pattern001.xml"}, true},
		// The anyURI + binary primitives' Facets instances are claimed (issue #124).
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/Facets/anyURI/anyURI_length001.xml"}, true},
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/Facets/hexBinary/hexBinary_maxLength001.xml"}, true},
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/msData/datatypes/Facets/base64Binary/base64Binary_length002.xml"}, true},
		// The Saxon precisionDecimal instance cases are claimed (issue #135).
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/saxonData/PDecimal/pdecimal001.v1.xml"}, true},
		{caseSpec{kind: kindInstance, doc: "../testdata/xsdtests/saxonData/PDecimal/pdecimal010.n1.xml"}, true},
		// A precisionDecimal SCHEMA case is not claimed (we cannot validate schemas).
		{caseSpec{kind: kindSchema, doc: "../testdata/xsdtests/saxonData/PDecimal/pdecimal001.xsd"}, false},
	}
	for _, tc := range cases {
		if got := selectsDatatypes(tc.c); got != tc.want {
			t.Errorf("selectsDatatypes(%q) = %v, want %v", tc.c.doc, got, tc.want)
		}
	}
}

// TestDatatypesExecutorAgreesWithSuite drives the real executor over a handful
// of real cohort documents and asserts it agrees with the suite's declared
// validity for the right reason: Parse accepts in-lexical-space values and
// rejects out-of-space ones. Skips when the submodule is absent.
func TestDatatypesExecutorAgreesWithSuite(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	dir := filepath.Join(suiteRoot, "msData", "datatypes")
	cases := []struct {
		file        string
		expectValid bool // the suite's declared XSD 1.1 validity
	}{
		{"decimal010.xml", true},  // value "1"
		{"decimal002.xml", true},  // value "-3.14159"
		{"decimal017.xml", false}, // value "e"    (not a decimal lexical)
		{"decimal019.xml", false}, // value "-1E4" (decimal has no exponent)
		{"decimal020.xml", false}, // value "INF"
		{"decimal023.xml", false}, // value "ABCDEF"
		{"boolean002.xml", true},  // value "true"
		{"boolean005.xml", true},  // value "0"
		{"boolean001.xml", false}, // value ""     (empty)
		{"boolean011.xml", false}, // value "True" (case-sensitive)
		{"string006.xml", true},   // any string is in the lexical space
	}
	for _, tc := range cases {
		c := caseSpec{
			kind:        kindInstance,
			doc:         filepath.Join(dir, tc.file),
			expectValid: tc.expectValid,
		}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", tc.file, tc.expectValid)
		}
	}

	// A deliberately WRONG expectation must produce Fail — proving the executor
	// genuinely computes validity rather than always passing.
	wrong := caseSpec{kind: kindInstance, doc: filepath.Join(dir, "decimal017.xml"), expectValid: true}
	if exec(wrong).IsPass() {
		t.Errorf("executor must Fail when the declared expectation is wrong (decimal017 'e' is not valid)")
	}
}

// TestDatatypesFacetsStringFamily drives the executor over real derived
// string-family Facets cases (issue #85): normalizedString and token restrictions
// resolve to their xs:string primitive ancestor (strictGoverns/primitiveOfType,
// reused from #81) and the seeded type's inherited whiteSpace (replace/collapse)
// normalizes the value before the string length/pattern checks. It asserts
// agreement for both polarities and that a wrong expectation yields Fail, so the
// test can actually fail if the cohort is mis-decided. Skips when the submodule
// is absent.
func TestDatatypesFacetsStringFamily(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	facetsDir := filepath.Join(suiteRoot, "msData", "datatypes", "Facets")
	cases := []struct {
		rel         string
		expectValid bool // the suite's declared XSD 1.1 validity
	}{
		{"token/token_length001.xml", false},                         // length=4, value "foofo" (5)
		{"token/token_length002.xml", true},                          // length=5, value "foofo" (5)
		{"token/token_pattern001.xml", true},                         // [a-z]{3}, value "abc"
		{"token/token_minLength001.xml", true},                       // minLength=4, value "foofo"
		{"normalizedString/normalizedString_length001.xml", false},   // length=4, value "foofo"
		{"normalizedString/normalizedString_minLength001.xml", true}, // minLength=4, value "foofo"
		{"normalizedString/normalizedString_pattern001.xml", true},   // [a-z]{3}, value "abc"
	}
	for _, tc := range cases {
		c := caseSpec{
			kind:        kindInstance,
			doc:         filepath.Join(facetsDir, filepath.FromSlash(tc.rel)),
			expectValid: tc.expectValid,
		}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", tc.rel, tc.expectValid)
		}
	}

	// A deliberately WRONG expectation must Fail: token_length001 ("foofo", 5) is
	// invalid under length=4, so claiming it valid must not pass.
	wrong := caseSpec{
		kind:        kindInstance,
		doc:         filepath.Join(facetsDir, "token", "token_length001.xml"),
		expectValid: true,
	}
	if exec(wrong).IsPass() {
		t.Errorf("executor must Fail when the declared expectation is wrong (token_length001 'foofo' is not length 4)")
	}
}

// TestDatatypesFacetsWideStringFamily drives the executor over the wider
// string-family Facets cohort (issue #106): language/Name/NCName/NMTOKEN
// restrictions resolve to their xs:string primitive ancestor through the token
// chain (strictGoverns/primitiveOfType, reused from #81), and the seeded type's
// intrinsic pattern + whiteSpace=collapse apply before the own facets. It also
// asserts the NCName cross-step pattern AND directly (§4.3.4.2 xr-pattern): a
// colon-bearing value passes Name's \i\c* but must be rejected by NCName's own
// [\i-[:]][\c-[:]]* via cvc-pattern-valid — no corpus case carries a colon, so
// this exercises the composition through the real value pipeline. Both
// polarities are asserted, and a wrong expectation must yield Fail. The NMTOKEN
// cases carry the tested value in an attribute, exercising the reader path.
func TestDatatypesFacetsWideStringFamily(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	facetsDir := filepath.Join(suiteRoot, "msData", "datatypes", "Facets")
	cases := []struct {
		rel         string
		expectValid bool // the suite's declared XSD 1.1 validity
	}{
		{"language/language_pattern001.xml", true},
		{"language/language_enumeration001.xml", false},
		{"Name/Name_pattern001.xml", true},
		{"Name/Name_length002.xml", true},  // length=5, value "foofo"
		{"Name/Name_length001.xml", false}, // length=4, value "foofo"
		{"NCName/NCName_pattern001.xml", true},
		{"NCName/NCName_length001.xml", false},
		{"NCName/NCName_enumeration001.xml", false},
		{"NMTOKEN/NMTOKEN_pattern001.xml", true}, // value in attrTest attribute
		{"NMTOKEN/NMTOKEN_length002.xml", true},  // length=5, value "foofo" (attribute)
		{"NMTOKEN/NMTOKEN_length001.xml", false}, // length=4, value "foofo" (attribute)
	}
	for _, tc := range cases {
		c := caseSpec{
			kind:        kindInstance,
			doc:         filepath.Join(facetsDir, filepath.FromSlash(tc.rel)),
			expectValid: tc.expectValid,
		}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", tc.rel, tc.expectValid)
		}
	}

	// A deliberately WRONG expectation must Fail: NMTOKEN_length001 ("foofo", 5)
	// is invalid under length=4, so claiming it valid must not pass — this also
	// proves the executor reads the ATTRIBUTE value, not the <foo> element text.
	wrong := caseSpec{
		kind:        kindInstance,
		doc:         filepath.Join(facetsDir, "NMTOKEN", "NMTOKEN_length001.xml"),
		expectValid: true,
	}
	if exec(wrong).IsPass() {
		t.Errorf("executor must Fail when the declared expectation is wrong (NMTOKEN_length001 'foofo' is not length 4)")
	}

	// NCName cross-step pattern AND, verified through the real value pipeline.
	backend := strict.New()
	types, err := builtin.Seed(backend)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	var ncname *xsd.SimpleType
	want := xsd.QName{Space: xsd.XMLSchemaNS, Local: "NCName"}
	for _, ty := range types {
		if ty.Name() == want {
			ncname = ty
			break
		}
	}
	if ncname == nil {
		t.Fatal("xs:NCName not seeded")
	}
	if _, verr := value.ValidateLexical(backend, ncname, "abc", nil); verr != nil {
		t.Errorf("NCName should accept %q: %v", "abc", verr)
	}
	_, verr := value.ValidateLexical(backend, ncname, "a:b", nil)
	if verr == nil {
		t.Fatal("NCName must reject a colon-bearing value via its intrinsic pattern")
	}
	rule, ok := xsderr.RuleOf(verr)
	if !ok || rule != "cvc-pattern-valid" {
		t.Errorf("NCName colon rejection rule = %q (ok=%v), want cvc-pattern-valid", rule, ok)
	}
}

// TestDatatypesFacetsBinaryAndURI drives the executor over real anyURI, hexBinary
// and base64Binary Facets cases (issue #124). All three are strict-mapped
// primitives (#82/#83), so their restrictions resolve to their own mapping
// (strictGoverns at the first step) and validate through the generic pipeline. The
// cohort is length-facet-carrying with a UNIT split (§4.3.1.3 clauses 1.1/1.2):
// anyURI measures length in characters (like string), the binary types in decoded
// OCTETS — a distinction the value.Lengthed Len() dispatch already realizes. Two
// cases are deliberately chosen so that a regression to lexical-character counting
// would flip them: hexBinary_maxLength001 (maxLength=4, "abcdef" = 6 hex chars but
// 3 octets, valid only under octet counting) and base64Binary_length002 (length=5,
// "MS0yLTM=" = 8 base64 chars but 5 octets, valid only under octet counting). Both
// polarities are asserted and a wrong expectation must yield Fail. Skips when the
// submodule is absent.
func TestDatatypesFacetsBinaryAndURI(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	facetsDir := filepath.Join(suiteRoot, "msData", "datatypes", "Facets")
	cases := []struct {
		rel         string
		expectValid bool // the suite's declared XSD 1.1 validity
	}{
		// anyURI: length in CHARACTERS (like string).
		{"anyURI/anyURI_length001.xml", false},   // length=4, "foofo" (5 chars)
		{"anyURI/anyURI_length002.xml", true},    // length=5, "foofo" (5 chars)
		{"anyURI/anyURI_minLength001.xml", true}, // minLength=4, "foofo"
		{"anyURI/anyURI_enumeration002.xml", true},
		// hexBinary: length in decoded OCTETS, not hex characters.
		{"hexBinary/hexBinary_maxLength001.xml", true}, // maxLength=4, "abcdef" = 3 octets
		{"hexBinary/hexBinary_minLength001.xml", true}, // minLength=4, "abcdefab" = 4 octets
		{"hexBinary/hexBinary_length001.xml", false},   // length=4, "abcde" not even-hex (lexical-invalid)
		{"hexBinary/hexBinary_enumeration002.xml", true},
		// base64Binary: length in decoded OCTETS, not base64 characters.
		{"base64Binary/base64Binary_length002.xml", true},    // length=5, "MS0yLTM=" = 5 octets
		{"base64Binary/base64Binary_minLength001.xml", true}, // minLength=4, "MS0yLTM=" = 5 octets
		{"base64Binary/base64Binary_length001.xml", false},   // length=4, "abcde" not multiple-of-4 (lexical-invalid)
	}
	for _, tc := range cases {
		c := caseSpec{
			kind:        kindInstance,
			doc:         filepath.Join(facetsDir, filepath.FromSlash(tc.rel)),
			expectValid: tc.expectValid,
		}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", tc.rel, tc.expectValid)
		}
	}

	// A deliberately WRONG expectation must Fail, and it is octet-unit-load-bearing:
	// hexBinary_maxLength001 ("abcdef", 3 octets under maxLength=4) is VALID, so
	// claiming it invalid must not pass. A regression to hex-character counting (6
	// chars > 4) would make the executor compute invalid and this wrong claim would
	// spuriously pass.
	wrong := caseSpec{
		kind:        kindInstance,
		doc:         filepath.Join(facetsDir, "hexBinary", "hexBinary_maxLength001.xml"),
		expectValid: false,
	}
	if exec(wrong).IsPass() {
		t.Errorf("executor must Fail when the declared expectation is wrong (hexBinary_maxLength001 'abcdef' is 3 octets, valid under maxLength=4)")
	}
}

// TestDatatypesFacetsShapeGuard proves readFacetsCase decides only the canonical
// single-<foo> instance shape and honestly declines the anyURI out-of-cohort
// shapes (issue #124): anyURI_b001.xml carries its values in repeated <bar>
// children (zero <foo>) and anyURI_b006.xml repeats many <foo> values against one
// enumeration (a list-style shape), neither of which is a single tested value. A
// mis-read there would coincidentally pass or fail for the wrong reason, inflating
// the ratchet; the exactly-one-<foo> guard declines both. Skips when the submodule
// is absent.
func TestDatatypesFacetsShapeGuard(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	anyURIDir := filepath.Join(suiteRoot, "msData", "datatypes", "Facets", "anyURI")

	// The canonical single-<foo> shape is read: value and base recovered.
	raw, base, children, _, ok := readFacetsCase(filepath.Join(anyURIDir, "anyURI_length001.xml"))
	if !ok {
		t.Fatal("readFacetsCase must accept the canonical single-<foo> anyURI_length001 shape")
	}
	if raw != "foofo" || base != "anyURI" || len(children) == 0 {
		t.Errorf("readFacetsCase(anyURI_length001) = raw=%q base=%q children=%d, want raw=foofo base=anyURI children>0", raw, base, len(children))
	}

	// The out-of-cohort shapes are declined: zero <foo> (b001) and multiple <foo>
	// (b006) both fail the exactly-one guard.
	for _, rel := range []string{"anyURI_b001.xml", "anyURI_b006.xml"} {
		if _, _, _, _, ok := readFacetsCase(filepath.Join(anyURIDir, rel)); ok {
			t.Errorf("readFacetsCase(%s) must decline the out-of-cohort shape (not exactly one <foo>)", rel)
		}
	}
}

// TestDatatypesQNameFacets proves the QName carve-outs of issue #125 as widened
// by issue #152: the length-family cases decide as vacuous passes through a real
// (non-nil) instance context (a nil context would fail parseQName even for the
// unprefixed literal "foofo"), and buildOwnFacets now ADMITS an enumeration facet
// on QName, building xsd.EnumerationMembers that carry the declaring schema's
// namespace context so a prefixed member resolves (§3.3.18). The four
// QName_enumeration001-004 fixtures are driven end-to-end and must decide
// correctly in both polarities. Skips when the submodule is absent.
func TestDatatypesQNameFacets(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	qnameDir := filepath.Join(suiteRoot, "msData", "datatypes", "Facets", "QName")

	// A real root-level context is built and the length case is read whole.
	raw, base, children, ctx, ok := readFacetsCase(filepath.Join(qnameDir, "QName_length001.xml"))
	if !ok || base != "QName" || raw != "foofo" || len(children) == 0 {
		t.Fatalf("readFacetsCase(QName_length001) = raw=%q base=%q children=%d ok=%v, want raw=foofo base=QName children>0 ok=true", raw, base, len(children), ok)
	}
	if ctx == nil {
		t.Fatal("readFacetsCase must thread a non-nil context for the QName cohort")
	}

	// buildOwnFacets admits both the length facet (vacuous per clause 1.3) and,
	// now, an enumeration facet on QName — the member carries the declaring
	// schema's context (issue #152), so it is no longer declined.
	if _, ok := buildOwnFacets("QName", []facetChild{{name: "length", value: "4"}}); !ok {
		t.Error("buildOwnFacets(QName, length) must be admitted (vacuous per clause 1.3)")
	}
	enumChild := facetChild{name: "enumeration", value: "foo:fo", bindings: map[string]string{"foo": "foobar"}}
	facets, ok := buildOwnFacets("QName", []facetChild{enumChild})
	if !ok || len(facets) != 1 {
		t.Fatalf("buildOwnFacets(QName, enumeration) must now be admitted (issue #152), got ok=%v facets=%d", ok, len(facets))
	}
	members, isEnum := facets[0].EnumerationMembers()
	if !isEnum || len(members) != 1 || members[0].Lexical() != "foo:fo" {
		t.Fatalf("enumeration facet members = %v (isEnum=%v), want one member foo:fo", members, isEnum)
	}
	if binds := members[0].NamespaceBindings(); len(binds) != 1 || binds[0].Prefix() != "foo" || binds[0].Namespace() != "foobar" {
		t.Errorf("enumeration member bindings = %v, want one foo=foobar binding", binds)
	}

	// End-to-end: the four QName_enumeration fixtures decide correctly. 001/003
	// carry an empty instance value (invalid); 002/004 carry "foo:fo" resolving to
	// the same expanded QName as the schema's member (valid).
	exec := newDatatypesExec()
	enumCases := []struct {
		file        string
		expectValid bool
	}{
		{"QName_enumeration001.xml", false},
		{"QName_enumeration002.xml", true},
		{"QName_enumeration003.xml", false},
		{"QName_enumeration004.xml", true},
	}
	for _, ec := range enumCases {
		c := caseSpec{kind: kindInstance, doc: filepath.Join(qnameDir, ec.file), expectValid: ec.expectValid}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", ec.file, ec.expectValid)
		}
		// A flipped expectation must yield Fail, proving the executor really decides.
		flipped := caseSpec{kind: kindInstance, doc: filepath.Join(qnameDir, ec.file), expectValid: !ec.expectValid}
		if got := exec(flipped); got.IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", ec.file)
		}
	}
}

// TestDatatypesNotationFacets drives the executor over the real NOTATION
// Facets-cohort fixtures (issue #153), whose two-step restriction shape (a named
// simpleType restricting xsd:NOTATION with jpeg/mpeg/g, then an attribute
// simpleType restricting THAT with one tested facet, tested value in <foo>'s
// attrTest attribute) is decoded by decodeNotationRestriction and decided by
// execNotationFacetsCase. Skips when the submodule is absent.
func TestDatatypesNotationFacets(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	dir := filepath.Join(suiteRoot, "msData", "datatypes", "Facets", "NOTATION")

	// The decoder must find both restriction steps: the outer base step restricting
	// NOTATION with the three enumerations, and the inner attrTest leaf step.
	baseStep, leafStep, ok := decodeNotationRestriction(filepath.Join(dir, "NOTATION_enumeration004.xsd"))
	if !ok || baseStep.base != "NOTATION" || len(baseStep.children) != 3 {
		t.Fatalf("decodeNotationRestriction base step = base=%q children=%d ok=%v, want base=NOTATION children=3 ok=true", baseStep.base, len(baseStep.children), ok)
	}
	if leafStep.attrName != "attrTest" || len(leafStep.children) != 2 {
		t.Fatalf("decodeNotationRestriction leaf step = attrName=%q children=%d, want attrTest children=2 (mpeg,g)", leafStep.attrName, len(leafStep.children))
	}
	raw, baseChildren, leafChildren, ctx, ok := readNotationFacetsCase(filepath.Join(dir, "NOTATION_enumeration004.xml"))
	if !ok || raw != "g" || len(baseChildren) != 3 || len(leafChildren) != 2 || ctx == nil {
		t.Fatalf("readNotationFacetsCase(enumeration004) = raw=%q base=%d leaf=%d ok=%v ctx=%v, want raw=g base=3 leaf=2 ok=true ctx!=nil", raw, len(baseChildren), len(leafChildren), ok, ctx)
	}

	// End-to-end over all 15 fixtures. Per the catalog, enumeration001/003 carry an
	// empty attrTest against a non-empty leaf enumeration (invalid); every other
	// case is valid — the length family is vacuous over NOTATION (§4.3.1.3 clause
	// 1.3), pattern001 matches, and the enumeration cases resolve their bare notation
	// name against the effective (superseded) enumeration set.
	exec := newDatatypesExec()
	cases := []struct {
		file        string
		expectValid bool
	}{
		{"NOTATION_length001.xml", true}, {"NOTATION_length002.xml", true}, {"NOTATION_length003.xml", true},
		{"NOTATION_minLength001.xml", true}, {"NOTATION_minLength002.xml", true},
		{"NOTATION_minLength003.xml", true}, {"NOTATION_minLength004.xml", true},
		{"NOTATION_maxLength001.xml", true}, {"NOTATION_maxLength002.xml", true}, {"NOTATION_maxLength003.xml", true},
		{"NOTATION_pattern001.xml", true},
		{"NOTATION_enumeration001.xml", false}, {"NOTATION_enumeration002.xml", true},
		{"NOTATION_enumeration003.xml", false}, {"NOTATION_enumeration004.xml", true},
	}
	for _, tc := range cases {
		c := caseSpec{kind: kindInstance, doc: filepath.Join(dir, tc.file), expectValid: tc.expectValid}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", tc.file, tc.expectValid)
		}
		// A flipped expectation must yield Fail, proving the executor really decides.
		flipped := caseSpec{kind: kindInstance, doc: filepath.Join(dir, tc.file), expectValid: !tc.expectValid}
		if got := exec(flipped); got.IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", tc.file)
		}
	}
}

// TestDatatypesLexicalItemShape drives the executor over the real
// <data><item SOMITEM_DATATYPE_X="value"/></data> fixtures (issue #146) that carry
// their tested value in an attribute and declare their schema out-of-band (no
// noNamespaceSchemaLocation, resolved to the sibling datatypes.xsd). Every one is
// suite-invalid because its tested value is out of its primitive's lexical space —
// duration "P"/"P1Y2M3DT"/"P1" (durationLexicalRep §3.3.6.2), gMonthDay
// "--02-30"/"--02-31" (con-gMonthDay-dayValue §3.3.12.1) and dateTime/date with a
// leading '+' before the year (§3.3.7.2/§3.3.9.2). The executor must agree (Pass),
// and a wrong expectation must yield Fail, so the test can actually fail if the
// item shape is mis-read. Skips when the submodule is absent.
func TestDatatypesLexicalItemShape(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	dir := filepath.Join(suiteRoot, "msData", "datatypes")
	// All five suite-declared invalid: at least one tested value is out-of-space.
	files := []string{
		"dateTime013.xml",  // +2001-07-11T12:23:45 (dateTime) and +2001-07-11 (date)
		"duration028.xml",  // "P"        (no field)
		"duration029.xml",  // "P1Y2M3DT" (dangling T)
		"duration030.xml",  // "P1"       (bare numeral)
		"gMonthDay006.xml", // --02-30 and --02-31 (day > 29 for month 2)
	}
	for _, f := range files {
		c := caseSpec{kind: kindInstance, doc: filepath.Join(dir, f), expectValid: false}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=false)", f)
		}
	}

	// A deliberately WRONG expectation must Fail: gMonthDay006 ("--02-30"/"--02-31")
	// is invalid, so claiming it valid must not pass — proving the executor actually
	// reads and parses the <item> attribute values rather than always passing.
	wrong := caseSpec{kind: kindInstance, doc: filepath.Join(dir, "gMonthDay006.xml"), expectValid: true}
	if exec(wrong).IsPass() {
		t.Errorf("executor must Fail when the declared expectation is wrong (gMonthDay006 --02-30/--02-31 are invalid)")
	}

	// readItemCase resolves each attribute to its primitive (from the sibling
	// datatypes.xsd) in document order, and declines a shape with no <item> children.
	lits, ok := readItemCase(filepath.Join(dir, "dateTime013.xml"))
	if !ok {
		t.Fatal("readItemCase must accept the two-item dateTime013 shape")
	}
	if len(lits) != 2 ||
		lits[0].prim != "dateTime" || lits[0].value != "+2001-07-11T12:23:45" ||
		lits[1].prim != "date" || lits[1].value != "+2001-07-11" {
		t.Errorf("readItemCase(dateTime013) = %+v, want [{dateTime +2001-07-11T12:23:45} {date +2001-07-11}]", lits)
	}
	// A comp_foo/simpleTest lexical case has no <item> children, so readItemCase
	// declines it (the comp_foo path owns it) rather than mis-reading it.
	if _, ok := readItemCase(filepath.Join(dir, "decimal010.xml")); ok {
		t.Error("readItemCase(decimal010) must decline the non-<item> comp_foo shape")
	}
}

// TestDatatypesLexicalDateTimeStampTimezone proves issue #140: a lexical-cohort
// dateTimeStamp literal is decided by the VALUE-based explicitTimezone facet
// (cvc-explicitTimezone-valid §4.3.14.3, checked at cvc-datatype-valid §4.1.4
// clause 3), not by lexical-space membership alone. The current W3C checkout has
// ZERO msData/datatypes/dateTimeStampNNN.xml cases, so this drives SYNTHETIC
// fixtures (a facet-free xs:dateTimeStamp restriction, the lexical cohort's exact
// comp_foo/simpleTest shape) through the real executor: a tz-bearing literal is
// accepted, a tz-ABSENT one is REJECTED. The tz-absent-claimed-valid assertion is
// load-bearing — the pre-#140 Parse-only path would false-ACCEPT it (making that
// wrong claim spuriously Pass); the fix routes dateTimeStamp through the facet
// pipeline so it Fails. It also pins the routing decision itself (fixesTimezone)
// and the rejection rule.
func TestDatatypesLexicalDateTimeStampTimezone(t *testing.T) {
	exec := newDatatypesExec()
	dir := t.TempDir()

	const schema = `<?xml version='1.0'?>
<xsd:schema xmlns:xsd='http://www.w3.org/2001/XMLSchema'>
  <xsd:element name='complexTest' type='complexfooType'/>
  <xsd:element name='simpleTest' type='simplefooType'/>
  <xsd:complexType name='complexfooType'>
    <xsd:sequence>
      <xsd:element name='comp_foo' type='xsd:dateTimeStamp'/>
    </xsd:sequence>
  </xsd:complexType>
  <xsd:simpleType name='simplefooType'>
    <xsd:restriction base='xsd:dateTimeStamp'/>
  </xsd:simpleType>
</xsd:schema>`
	if err := os.WriteFile(filepath.Join(dir, "dateTimeStamp.xsd"), []byte(schema), 0o600); err != nil {
		t.Fatal(err)
	}
	instancePath := func(name, lexical string) string {
		doc := `<?xml version='1.0'?>
<root xmlns:xsi='http://www.w3.org/2001/XMLSchema-instance' xsi:noNamespaceSchemaLocation='dateTimeStamp.xsd'>
  <complexTest><comp_foo>` + lexical + `</comp_foo></complexTest>
  <simpleTest>` + lexical + `</simpleTest>
</root>`
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}

	// A tz-bearing literal is in dateTimeStamp's value space AND satisfies the
	// required-timezone facet, so it is VALID; a tz-absent one is lexically a
	// dateTime but violates cvc-explicitTimezone-valid, so it is INVALID.
	bearing := instancePath("bearing.xml", "2002-10-10T12:00:00Z")
	absent := instancePath("absent.xml", "2002-10-10T12:00:00")

	cases := []struct {
		name       string
		doc        string
		suiteValid bool // the spec-correct validity of the literal
	}{
		{"tz-bearing", bearing, true},
		{"tz-absent", absent, false},
	}
	for _, tc := range cases {
		// The executor agrees with the spec-correct validity...
		right := caseSpec{kind: kindInstance, doc: tc.doc, expectValid: tc.suiteValid}
		if got := exec(right); !got.IsPass() {
			t.Errorf("%s: executor disagreed with spec-correct validity (expectValid=%v)", tc.name, tc.suiteValid)
		}
		// ...and DISAGREES with the opposite claim, so the test can actually fail.
		// For tz-absent this is the #140 anti-regression: the Parse-only path would
		// false-accept it and this wrong "valid" claim would spuriously Pass.
		wrong := caseSpec{kind: kindInstance, doc: tc.doc, expectValid: !tc.suiteValid}
		if exec(wrong).IsPass() {
			t.Errorf("%s: executor must Fail against the wrong expectation (expectValid=%v)", tc.name, !tc.suiteValid)
		}
	}

	// Pin the routing decision itself: fixesTimezone is true for the seeded
	// dateTimeStamp (explicitTimezone=required) and false for plain dateTime
	// (explicitTimezone=optional), so only the former leaves the parseOK path.
	backend := strict.New()
	types, err := builtin.Seed(backend)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	var dts, dt *xsd.SimpleType
	for _, ty := range types {
		switch ty.Name() {
		case xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTimeStamp"}:
			dts = ty
		case xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTime"}:
			dt = ty
		}
	}
	if dts == nil || dt == nil {
		t.Fatal("Seed did not return xs:dateTimeStamp and xs:dateTime")
	}
	if !fixesTimezone(dts) {
		t.Error("fixesTimezone(dateTimeStamp) = false, want true (explicitTimezone=required)")
	}
	if fixesTimezone(dt) {
		t.Error("fixesTimezone(dateTime) = true, want false (explicitTimezone=optional)")
	}

	// The rejection reason is cvc-explicitTimezone-valid, not a lexical failure —
	// proving the value-based facet, not Parse, decides the tz-absent literal.
	_, verr := value.ValidateLexical(backend, dts, "2002-10-10T12:00:00", nil)
	if verr == nil {
		t.Fatal("tz-absent dateTimeStamp must be rejected via value.ValidateLexical, got nil")
	}
	if rule, ok := xsderr.RuleOf(verr); !ok || rule != "cvc-explicitTimezone-valid" {
		t.Errorf("tz-absent dateTimeStamp rejection rule = %q (ok=%v), want cvc-explicitTimezone-valid", rule, ok)
	}
}

// TestDatatypesLexicalQNameCohort drives the executor over the context-dependent
// QName lexical cohort (issue #131): each comp_foo/simpleTest literal resolves
// its prefix against the in-scope XML namespace bindings the harness decodes from
// the instance (readQNameContexts/nsContext), so strict's parseQName decides
// lexical-space membership under a real value.Context (§3.3.18). Both polarities
// are asserted against the suite's declared validity, and a wrong expectation
// must yield Fail, so the test can actually fail if the adapter mis-resolves.
// NOTATION carries no plain lexical case in the checkout (its cases are all
// facet-cohort under Facets/NOTATION), so this cohort is QName-only today. Skips
// when the submodule is absent.
func TestDatatypesLexicalQNameCohort(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	dir := filepath.Join(suiteRoot, "msData", "datatypes")
	cases := []struct {
		file        string
		expectValid bool // the suite's declared XSD 1.1 validity (.v/.i case suffix)
	}{
		{"QName001.xml", false}, // ""         empty, not an NCName
		{"QName002.xml", true},  // "_foo"     unprefixed, binds to the default namespace
		{"QName003.xml", true},  // "fo124"    unprefixed NCName
		{"QName004.xml", false}, // "1fo"      not an NCName (leading digit)
		{"QName005.xml", false}, // "-foo"     not an NCName (leading hyphen)
		{"QName006.xml", true},  // "fo:foo"   prefix fo bound to "myNamespace" on root
		{"QName007.xml", false}, // ":foo"     empty prefix part
		{"QName008.xml", false}, // "fo:1fo"   local part not an NCName
		{"QName009.xml", false}, // "xmlns:xsi" prefix xmlns is not bindable, so unbound (bugzilla 4053)
		{"QName010.xml", false}, // "@test"    not an NCName
		{"QName011.xml", false}, // "//foo"    not an NCName
	}
	for _, tc := range cases {
		c := caseSpec{
			kind:        kindInstance,
			doc:         filepath.Join(dir, tc.file),
			expectValid: tc.expectValid,
		}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", tc.file, tc.expectValid)
		}
	}

	// A deliberately WRONG expectation must Fail: QName006 ("fo:foo", a bound
	// prefix) is valid, so claiming it invalid must not pass — proving the executor
	// actually resolves the prefix rather than always passing.
	wrong := caseSpec{kind: kindInstance, doc: filepath.Join(dir, "QName006.xml"), expectValid: false}
	if exec(wrong).IsPass() {
		t.Errorf("executor must Fail when the declared expectation is wrong (QName006 'fo:foo' is valid)")
	}

	// readQNameContexts must capture the fo binding declared on <root> and read
	// both comp_foo and simpleTest as "fo:foo" with a context that resolves fo.
	lits, ok := readQNameContexts(filepath.Join(dir, "QName006.xml"))
	if !ok {
		t.Fatal("readQNameContexts must accept the QName006 comp_foo/simpleTest shape")
	}
	if len(lits) != 2 || lits[0].value != "fo:foo" || lits[1].value != "fo:foo" {
		t.Fatalf("readQNameContexts(QName006) values = %+v, want two \"fo:foo\"", lits)
	}
	if ns, bound := lits[0].ctx.LookupNamespace("fo"); !bound || ns != "myNamespace" {
		t.Errorf("comp_foo context LookupNamespace(fo) = (%q,%v), want (\"myNamespace\",true)", ns, bound)
	}
}

// TestDatatypesPDecimalCohort drives the executor over the real Saxon PDecimal
// precisionDecimal cohort (issue #135): each <doc> carries repeated
// <e value="…"/> literals validated against ONE synthesized leaf (precisionDecimal
// restricted by the sibling schema's facets), through the real facet pipeline.
// Both polarities are asserted for the right reason — a wrong expectation must
// yield Fail, exercised on the NaN-vs-bound case so a regression to accepting NaN
// under a bound facet (violating the partial order) would surface. The two-step
// chain / list / union shapes (pdecimal016/019/020) are declined by readPDecimalCase,
// and the pdecimal006.n2 suite quirk (NaN matching a NaN enumeration member) is
// pinned as spec-correct-VALID. Skips when the submodule is absent.
func TestDatatypesPDecimalCohort(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	dir := filepath.Join(suiteRoot, "saxonData", "PDecimal")
	cases := []struct {
		file        string
		expectValid bool // the suite's declared XSD 1.1 validity
	}{
		{"pdecimal001.v1.xml", true},  // unrestricted precisionDecimal: numerals, specials, -0
		{"pdecimal001.v2.xml", true},  // same values with leading/trailing whitespace (collapse)
		{"pdecimal001.n1.xml", false}, // " fried chicken " not in the lexical space
		{"pdecimal001.n2.xml", false}, // "Infinity" (only INF is a special literal)
		{"pdecimal002.v1.xml", true},  // minInclusive=0
		{"pdecimal002.n1.xml", false}, // "-12" < 0
		{"pdecimal002.n2.xml", false}, // "-INF" < 0
		{"pdecimal002.n3.xml", false}, // "NaN" incomparable with the bound ⇒ excluded
		{"pdecimal006.v1.xml", true},  // enumeration {-INF,+INF,0.0,1.0,NaN}, value-space match
		{"pdecimal006.n1.xml", false}, // "17.3" not an enumeration member
		{"pdecimal007.v1.xml", true},  // pattern "NaN"
		{"pdecimal007.n1.xml", false}, // "13" fails the pattern
		{"pdecimal008.v1.xml", true},  // totalDigits=4, zero + specials vacuously pass
		{"pdecimal008.n1.xml", false}, // "12345" has 5 total digits
		{"pdecimal010.v1.xml", true},  // minScale=4,maxScale=8
		{"pdecimal010.n1.xml", false}, // "0.003" scale 3 < minScale 4
	}
	for _, tc := range cases {
		c := caseSpec{kind: kindInstance, doc: filepath.Join(dir, tc.file), expectValid: tc.expectValid}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=%v)", tc.file, tc.expectValid)
		}
	}

	// A deliberately WRONG expectation must Fail, and it is partial-order-load-bearing:
	// pdecimal002.n3 ("NaN" under minInclusive=0) is INVALID because NaN is
	// incomparable with every bound (§3.1), so claiming it valid must not pass. A
	// regression to treating NaN as satisfying (or vacuously passing) a bound facet
	// would compute valid and this wrong claim would spuriously pass.
	wrong := caseSpec{kind: kindInstance, doc: filepath.Join(dir, "pdecimal002.n3.xml"), expectValid: true}
	if exec(wrong).IsPass() {
		t.Errorf("executor must Fail when the declared expectation is wrong (pdecimal002.n3 'NaN' fails minInclusive=0)")
	}

	// readPDecimalCase reads the direct-primitive shape whole and resolves the base.
	children, values, ok := readPDecimalCase(filepath.Join(dir, "pdecimal001.v1.xml"))
	if !ok {
		t.Fatal("readPDecimalCase must accept the direct-precisionDecimal pdecimal001.v1 shape")
	}
	if len(children) != 0 || len(values) != 18 {
		t.Errorf("readPDecimalCase(pdecimal001.v1) = children=%d values=%d, want children=0 values=18", len(children), len(values))
	}

	// The multi-step chain (016), list (019) and union (020) varieties are declined:
	// this single-synthesized-leaf model cannot decide them, so they are honest gaps
	// rather than mis-decided cases.
	for _, rel := range []string{"pdecimal016.v1.xml", "pdecimal019.v1.xml", "pdecimal020.v1.xml"} {
		if _, _, ok := readPDecimalCase(filepath.Join(dir, rel)); ok {
			t.Errorf("readPDecimalCase(%s) must decline the multi-step/list/union shape", rel)
		}
	}

	// pdecimal006.n2 is a KNOWN suite quirk: "NaN" against an enumeration containing
	// a "NaN" member is VALID per cvc-enumeration-valid's identity branch (§4.3.5.4;
	// NaN is identical to itself). The executor keeps the spec-correct verdict, so it
	// computes VALID: a claim of valid Passes (proving the identity match), and the
	// suite's own invalid expectation yields a Fail (the honest recorded gap).
	n2 := filepath.Join(dir, "pdecimal006.n2.xml")
	if !exec(caseSpec{kind: kindInstance, doc: n2, expectValid: true}).IsPass() {
		t.Error("pdecimal006.n2 'NaN' must be computed VALID (matches the NaN enumeration member by identity, §4.3.5.4)")
	}
	if exec(caseSpec{kind: kindInstance, doc: n2, expectValid: false}).IsPass() {
		t.Error("pdecimal006.n2 must Fail against the suite's invalid expectation (spec-correct disagreement, not a false pass)")
	}
}

// TestNSContextLookup proves nsContext resolves prefixes exactly as §3.3.18
// requires: a declared prefix resolves to its binding; the reserved xml prefix is
// bound without a declaration (Namespaces in XML §3) while xmlns is not a bindable
// prefix (WG ruling bugzilla 4053, unbound → ok=false); the empty
// prefix binds to the default namespace when declared and to no namespace
// otherwise (ok=true, element-name semantics); a never-declared non-empty prefix
// is genuinely unbound (ok=false), which strict's Parse turns into a rejection.
func TestNSContextLookup(t *testing.T) {
	c := nsContext{bindings: map[string]string{"fo": "myNamespace", "": "defaultNS"}}
	cases := []struct {
		prefix   string
		wantNS   string
		wantBnd  bool
		describe string
	}{
		{"fo", "myNamespace", true, "declared prefix"},
		{"", "defaultNS", true, "empty prefix with a declared default"},
		{"xml", xmlPrefixNS, true, "reserved xml prefix"},
		{"xmlns", "", false, "xmlns is not a resolvable prefix"},
		{"zzz", "", false, "undeclared non-empty prefix is unbound"},
	}
	for _, tc := range cases {
		ns, bound := c.LookupNamespace(tc.prefix)
		if ns != tc.wantNS || bound != tc.wantBnd {
			t.Errorf("%s: LookupNamespace(%q) = (%q,%v), want (%q,%v)",
				tc.describe, tc.prefix, ns, bound, tc.wantNS, tc.wantBnd)
		}
	}

	// With no default declared, an unprefixed name still resolves — to no
	// namespace — so an unprefixed QName is never rejected as unbound.
	empty := nsContext{bindings: map[string]string{}}
	if ns, bound := empty.LookupNamespace(""); !bound || ns != "" {
		t.Errorf("empty prefix with no default: LookupNamespace(\"\") = (%q,%v), want (\"\",true)", ns, bound)
	}
}

// TestDatatypesD34Cohort drives the executor over the real IBM D3_3_4
// precisionDecimal cohort (issue #162): one schema declares SEVERAL named
// simpleTypes (each a direct single-step precisionDecimal restriction) and the
// instance's <root> carries MULTIPLE children, each dispatched to its own named
// type by the child's declared type= attribute. Both polarities are asserted for
// the decidable cases (v14/v23/v24, ii01[,a-f], ii02); the structurally
// out-of-reach shapes (v15 type-reference root, v16 list, v17 union, v18 ref=
// children, v19-v22 multi-step chains) must be DECLINED by readD34Case (ok=false)
// so they remain honest gaps rather than mis-decided cases. Skips when the
// submodule is absent.
func TestDatatypesD34Cohort(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newDatatypesExec()

	validDir := filepath.Join(suiteRoot, "ibmData", "valid", "D3_3_4")
	invalidDir := filepath.Join(suiteRoot, "ibmData", "instance_invalid", "D3_3_4")

	// The valid cases are decided VALID and Pass against their true expectation.
	for _, f := range []string{"d3_3_4v14.xml", "d3_3_4v23.xml", "d3_3_4v24.xml"} {
		c := caseSpec{kind: kindInstance, doc: filepath.Join(validDir, f), expectValid: true}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=true)", f)
		}
		// A wrong (invalid) expectation must Fail, proving the executor really
		// computed VALID rather than declining.
		wrong := caseSpec{kind: kindInstance, doc: filepath.Join(validDir, f), expectValid: false}
		if exec(wrong).IsPass() {
			t.Errorf("%s: a wrong invalid expectation must Fail (executor must compute VALID)", f)
		}
	}

	// The instance_invalid cases are decided INVALID and Pass against their true
	// (invalid) expectation — a shared-schema instance (ii01a..f reuse ii01.xsd)
	// resolves its schema from xsi:schemaLocation, not a filename-derived path.
	for _, f := range []string{
		"d3_3_4ii01.xml", "d3_3_4ii01a.xml", "d3_3_4ii01b.xml", "d3_3_4ii01c.xml",
		"d3_3_4ii01d.xml", "d3_3_4ii01e.xml", "d3_3_4ii01f.xml", "d3_3_4ii02.xml",
	} {
		c := caseSpec{kind: kindInstance, doc: filepath.Join(invalidDir, f), expectValid: false}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s: executor disagreed with suite (expectValid=false)", f)
		}
		// A wrong (valid) expectation must Fail, proving a genuine INVALID verdict
		// (some child's value violates its type's facet), not a decline.
		wrong := caseSpec{kind: kindInstance, doc: filepath.Join(invalidDir, f), expectValid: true}
		if exec(wrong).IsPass() {
			t.Errorf("%s: a wrong valid expectation must Fail (executor must compute INVALID)", f)
		}
	}

	// readD34Case accepts the decidable multi-type shape whole: v14 binds seven
	// named types via its inline sequence and its instance children all resolve.
	typeFacets, elems, ok := readD34Case(filepath.Join(validDir, "d3_3_4v14.xml"))
	if !ok {
		t.Fatal("readD34Case must accept the decidable d3_3_4v14 multi-type shape")
	}
	if len(elems) == 0 || len(typeFacets) == 0 {
		t.Fatalf("readD34Case(v14) = typeFacets=%d elems=%d, want both non-empty", len(typeFacets), len(elems))
	}
	// Every instance child resolves to an indexed named type carried in typeFacets.
	for _, e := range elems {
		if _, indexed := typeFacets[e.typeName]; !indexed {
			t.Errorf("readD34Case(v14): child bound to un-indexed type %q", e.typeName)
		}
	}

	// The structurally out-of-reach shapes are DECLINED (ok=false), each for its
	// own reason, so they remain honest gaps rather than mis-decided cases:
	// v15 type-reference root, v16 list, v17 union, v18 ref= children, v19-v22
	// multi-step restriction chains.
	for _, f := range []string{
		"d3_3_4v15.xml", "d3_3_4v16.xml", "d3_3_4v17.xml", "d3_3_4v18.xml",
		"d3_3_4v19.xml", "d3_3_4v20.xml", "d3_3_4v21.xml", "d3_3_4v22.xml",
	} {
		if _, _, ok := readD34Case(filepath.Join(validDir, f)); ok {
			t.Errorf("readD34Case(%s) must decline the out-of-reach shape (honest gap)", f)
		}
	}
}
