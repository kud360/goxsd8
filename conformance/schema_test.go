package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser"
)

// schemaDoc builds an in-memory schema document from body children wrapped in a
// <schema> with the xs prefix bound, mirroring parser/produce_test.go's wrap.
func schemaDoc(t *testing.T, body string) *parser.Document {
	t.Helper()
	src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xs:schema>`
	d, err := parser.ReadDocument("mem://schema.xsd", strings.NewReader(src))
	if err != nil {
		t.Fatalf("ReadDocument(%q): %v", body, err)
	}
	return d
}

// TestSchemaShapeDecidableAccepts proves schemaShapeDecidable admits exactly the
// producer's decidable subset: type=-form elements, bare-or-typed attributes,
// restriction-only simpleTypes (including a recursed anonymous inline base), and
// annotations — the shapes parser.Produce genuinely decides.
func TestSchemaShapeDecidableAccepts(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"typed element", `<xs:element name="e" type="xs:string"/>`},
		{"bare attribute (defaults to anySimpleType)", `<xs:attribute name="a"/>`},
		{"typed attribute", `<xs:attribute name="a" type="xs:string"/>`},
		{"restriction simpleType with pattern", `<xs:simpleType name="T"><xs:restriction base="xs:string"><xs:pattern value="1|2"/></xs:restriction></xs:simpleType>`},
		{"annotation", `<xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation>`},
		{"anonymous inline base (recursed)", `<xs:simpleType name="N"><xs:restriction><xs:simpleType><xs:restriction base="xs:string"><xs:pattern value="1*"/></xs:restriction></xs:simpleType><xs:minLength value="1"/></xs:restriction></xs:simpleType>`},
		{"all decidable kinds together", `<xs:element name="e" type="T"/><xs:attribute name="a"/><xs:simpleType name="T"><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`},
	}
	for _, tc := range cases {
		if !schemaShapeDecidable(schemaDoc(t, tc.body)) {
			t.Errorf("%s: schemaShapeDecidable = false, want true", tc.name)
		}
	}
}

// TestSchemaShapeDecidableDeclines proves schemaShapeDecidable declines every
// shape whose Produce verdict would be a limitation-in-disguise (a false reject or
// an unsupported-form rejection) or a vacuous pass over silently-skipped content.
func TestSchemaShapeDecidableDeclines(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"top-level complexType (silently skipped)", `<xs:complexType name="T"><xs:sequence/></xs:complexType>`},
		{"top-level group (silently skipped)", `<xs:group name="g"><xs:sequence/></xs:group>`},
		{"bare element (would false-reject at src-resolve)", `<xs:element name="e"/>`},
		{"element with inline anonymous type", `<xs:element name="e"><xs:complexType/></xs:element>`},
		{"element with both type= and inline type", `<xs:element name="e" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`},
		{"attribute with inline simpleType", `<xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`},
		{"list-variety simpleType", `<xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`},
		{"union-variety simpleType", `<xs:simpleType name="U"><xs:union memberTypes="xs:string"/></xs:simpleType>`},
		{"restriction with enumeration facet", `<xs:simpleType name="E"><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType>`},
		{"anonymous inline base with enumeration (recursed decline)", `<xs:simpleType name="N"><xs:restriction><xs:simpleType><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType></xs:restriction></xs:simpleType>`},
		{"one decidable + one undecidable child declines whole", `<xs:element name="e" type="xs:string"/><xs:complexType name="T"><xs:sequence/></xs:complexType>`},
	}
	for _, tc := range cases {
		if schemaShapeDecidable(schemaDoc(t, tc.body)) {
			t.Errorf("%s: schemaShapeDecidable = true, want false", tc.name)
		}
	}
}

// TestSchemaExecutorReadErrorDeclines proves a ReadDocument failure is DECLINED
// (Fail) for BOTH polarities, never turned into an observed-invalid verdict: the
// error cannot distinguish a genuine XML well-formedness fault from a parser
// encoding limitation (e.g. well-formed UTF-16 misread as invalid UTF-8), so
// claiming "invalid" would fabricate a verdict for a possibly-well-formed document.
func TestSchemaExecutorReadErrorDeclines(t *testing.T) {
	exec := newSchemaExec()
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.xsd")
	// Unclosed root element: a ReadDocument error (here an XML well-formedness fault).
	if err := os.WriteFile(malformed, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e"`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, ev := range []bool{true, false} {
		if exec(caseSpec{kind: kindSchema, doc: malformed, expectValid: ev}).IsPass() {
			t.Errorf("a ReadDocument error must Fail (decline) regardless of expectValid=%v", ev)
		}
	}
}

// TestSchemaExecutorDeclinesNonSchemaRoot proves a well-formed document whose root
// is not <schema> is DECLINED unconditionally (§3.17.2 does not require a <schema>
// root, so it is not decidable for this lane) — Fail for both polarities.
func TestSchemaExecutorDeclinesNonSchemaRoot(t *testing.T) {
	exec := newSchemaExec()
	dir := t.TempDir()
	nonSchema := filepath.Join(dir, "notschema.xml")
	if err := os.WriteFile(nonSchema, []byte(`<root/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, ev := range []bool{true, false} {
		if exec(caseSpec{kind: kindSchema, doc: nonSchema, expectValid: ev}).IsPass() {
			t.Errorf("non-schema root must Fail (decline) regardless of expectValid=%v", ev)
		}
	}
}

// TestSchemaExecutorAgreesWithSuite drives the real executor over real suite
// schemaTest fixtures and asserts it agrees with the suite's declared validity for
// the right reason: a decidable valid schema Produces cleanly, a duplicate
// top-level simpleType name is rejected (sch-props-correct §3.17.6.1 clause 2), and
// a wrong expectation yields Fail so the test can actually fail. Skips when the
// submodule is absent.
func TestSchemaExecutorAgreesWithSuite(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newSchemaExec()

	sunSType := filepath.Join(suiteRoot, "sunData", "SType")
	cases := []struct {
		rel         string
		expectValid bool
		why         string
	}{
		// Decidable VALID: top-level element type="Test" + restriction-only simpleType.
		{"ST_baseTD/ST_baseTD00101m/ST_baseTD00101m.xsd", true, "element type= + restriction simpleType (pattern)"},
		// Decidable VALID: anonymous inline base reached through the restriction chain.
		{"ST_facets/ST_facets00101m/ST_facets00101m.xsd", true, "restriction over an inline anonymous simpleType base"},
		// Decidable INVALID: two top-level simpleTypes named "Test" collide per kind.
		{"ST_name/ST_name00301m/ST_name00301m.xsd", false, "duplicate top-level simpleType name (sch-props-correct clause 2)"},
	}
	for _, tc := range cases {
		doc := filepath.Join(sunSType, filepath.FromSlash(tc.rel))
		c := caseSpec{kind: kindSchema, doc: doc, expectValid: tc.expectValid}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s (%s): executor disagreed with suite (expectValid=%v)", tc.rel, tc.why, tc.expectValid)
		}
		// A flipped expectation must Fail, proving the executor really decides.
		flipped := caseSpec{kind: kindSchema, doc: doc, expectValid: !tc.expectValid}
		if exec(flipped).IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", tc.rel)
		}
	}
}

// TestSchemaExecutorDeclinesUndecidableSuiteCase proves the false-accept guard on a
// real fixture: abstract00101m.xsd is suite-VALID but carries a top-level
// <complexType> Produce silently skips, so the executor must DECLINE (Fail) rather
// than vacuously pass — a valid-declared case the executor refuses to claim,
// recording an honest gap. Skips when the submodule is absent.
func TestSchemaExecutorDeclinesUndecidableSuiteCase(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newSchemaExec()
	doc := filepath.Join(suiteRoot, "sunData", "ElemDecl", "abstract", "abstract00101m", "abstract00101m.xsd")
	// Suite-valid, but undecidable (contains complexType): the executor must not
	// claim it — Fail against the true valid expectation is the honest gap.
	if exec(caseSpec{kind: kindSchema, doc: doc, expectValid: true}).IsPass() {
		t.Error("a suite-valid case with a skipped top-level complexType must be DECLINED (Fail), never vacuously passed")
	}
}
