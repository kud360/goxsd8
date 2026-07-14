package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
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
