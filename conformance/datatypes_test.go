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

// TestDatatypesBackendSeeds guards the lane's precondition loudly: the composed
// strict+fallback backend must satisfy builtin.Seed (all primitives mapped), or
// the executor's symbol table is empty and every claimed case fails.
func TestDatatypesBackendSeeds(t *testing.T) {
	backend := value.Override(fallbackPrimitives{}, strict.New())
	types, err := builtin.Seed(backend)
	if err != nil {
		t.Fatalf("Seed(strict+fallback) must succeed, got %v", err)
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
	strictBackend := strict.New()
	backend := value.Override(fallbackPrimitives{}, strictBackend)
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
	raw, base, children, ok := readFacetsCase(filepath.Join(anyURIDir, "anyURI_length001.xml"))
	if !ok {
		t.Fatal("readFacetsCase must accept the canonical single-<foo> anyURI_length001 shape")
	}
	if raw != "foofo" || base != "anyURI" || len(children) == 0 {
		t.Errorf("readFacetsCase(anyURI_length001) = raw=%q base=%q children=%d, want raw=foofo base=anyURI children>0", raw, base, len(children))
	}

	// The out-of-cohort shapes are declined: zero <foo> (b001) and multiple <foo>
	// (b006) both fail the exactly-one guard.
	for _, rel := range []string{"anyURI_b001.xml", "anyURI_b006.xml"} {
		if _, _, _, ok := readFacetsCase(filepath.Join(anyURIDir, rel)); ok {
			t.Errorf("readFacetsCase(%s) must decline the out-of-cohort shape (not exactly one <foo>)", rel)
		}
	}
}
