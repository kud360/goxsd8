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
		{"bare element (defaults to anyType, now seeded)", `<xs:element name="e"/>`},
		{"empty complexType", `<xs:complexType name="CT"/>`},
		{"complexType with sequence + local element + attribute", `<xs:complexType name="CT"><xs:sequence><xs:element name="a" type="xs:string"/><xs:any/></xs:sequence><xs:attribute name="at" type="xs:int"/></xs:complexType>`},
		{"complexType with choice + anyAttribute", `<xs:complexType name="CT"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice><xs:anyAttribute namespace="##other"/></xs:complexType>`},
		{"complexContent restriction", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="CT"><xs:complexContent><xs:restriction base="B"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"top-level group definition (§3.7.2)", `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`},
		{"top-level group with choice + any", `<xs:group name="g"><xs:choice><xs:element name="a" type="xs:string"/><xs:any/></xs:choice></xs:group>`},
		{"top-level attributeGroup definition (§3.6.2)", `<xs:attributeGroup name="ag"><xs:attribute name="a" type="xs:string"/><xs:anyAttribute namespace="##other"/></xs:attributeGroup>`},
		{"attributeGroup referencing another group", `<xs:attributeGroup name="ag"><xs:attributeGroup ref="base"/><xs:attribute name="a"/></xs:attributeGroup>`},
		{"complexType with group ref content", `<xs:complexType name="T"><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType>`},
		{"complexType with top-level group ref as content", `<xs:complexType name="T"><xs:group ref="g"/></xs:complexType>`},
		{"complexType with attributeGroup ref", `<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="ag"/></xs:complexType>`},
		{"all decidable kinds together", `<xs:element name="e" type="T"/><xs:attribute name="a"/><xs:simpleType name="T"><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`},
		{"top-level notation (§3.14.2)", `<xs:notation name="n" public="-//x//y" system="x.dtd"/>`},
		{"element with name= identity constraint", `<xs:element name="e"><xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key></xs:element>`},
		// #229: an inline anonymous <simpleType> on a LOCAL element or attribute is
		// produced (§3.3.2.1 dcl.elt.common clause 1, §3.2.2.2 dcl.att.local), so the
		// gates admit it — provided the inline type's own shape is produced too.
		{"local element with inline anonymous simpleType", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		{"local attribute with inline anonymous simpleType", `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:attribute></xs:complexType>`},
		{"attributeGroup attribute with inline anonymous simpleType", `<xs:attributeGroup name="ag"><xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute></xs:attributeGroup>`},
		// Both type= and an inline <simpleType> on a LOCAL declaration is now a
		// genuine src-element/src-attribute rejection, not a limitation, so the case
		// is decided rather than declined.
		{"local element with both type= and an inline simpleType (src-element clause 3)", `<xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		{"local attribute with both type= and an inline simpleType (src-attribute clause 4)", `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute></xs:complexType>`},
		{"local element with name= identity constraint", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:unique name="u"><xs:selector xpath="b"/><xs:field xpath="@x"/></xs:unique></xs:element></xs:sequence></xs:complexType>`},
		// #240 produced the ref= form too — it maps to the definition it names
		// (§3.11.2), so src-identity-constraint clauses 1/4/5 and src-resolve on it
		// are genuine verdicts, not limitations.
		{"element with ref= identity constraint", `<xs:element name="e"><xs:key ref="k"/></xs:element><xs:element name="d"><xs:key name="k"><xs:selector xpath="b"/><xs:field xpath="@x"/></xs:key></xs:element>`},
		{"local element with ref= identity constraint", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:keyref ref="kr"/></xs:element></xs:sequence></xs:complexType>`},
		{"complexType with assert", `<xs:complexType name="T"><xs:sequence/><xs:assert test="true()"/></xs:complexType>`},
		{"complexContent restriction with assert", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:complexContent><xs:restriction base="B"><xs:sequence/><xs:assert test="true()"/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"restriction with assertion facet", `<xs:simpleType name="A"><xs:restriction base="xs:int"><xs:assertion test="$value > 0"/></xs:restriction></xs:simpleType>`},
		// #242: <include> contributes no component of its own, so it is admitted
		// here; the decidability of what it points at is the closure walk's job
		// (closureScan.decidable), not this allowlist's.
		{"top-level include", `<xs:include schemaLocation="lib.xsd"/>`},
		{"include beside decidable kinds", `<xs:include schemaLocation="lib.xsd"/><xs:element name="e" type="xs:string"/>`},
		// #182: <import> likewise contributes no component of its own. The stricter
		// no-D2 rule that governs it lives in closureScan.importDirective, not in
		// this shape allowlist, so a bare <import> is admitted HERE and declined
		// THERE.
		{"top-level import", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`},
		{"import beside include and decidable kinds", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/><xs:include schemaLocation="lib.xsd"/><xs:element name="e" type="xs:string"/>`},
		// #183: an <override> is admitted when each of its children is a decidable
		// source declaration, since §F.2 clause 1 makes those children top-level
		// declarations of the OVERRIDDEN document. What it points at is the closure
		// walk's job (closureScan.compose), not this allowlist's.
		{"override with decidable children", `<xs:override schemaLocation="b.xsd"><xs:element name="e" type="xs:string"/><xs:simpleType name="T"><xs:restriction base="xs:string"/></xs:simpleType></xs:override>`},
		{"override with only an annotation", `<xs:override schemaLocation="b.xsd"><xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation></xs:override>`},
		{"empty override", `<xs:override schemaLocation="b.xsd"/>`},
		{"override beside decidable kinds", `<xs:override schemaLocation="b.xsd"><xs:notation name="n" public="p"/></xs:override><xs:element name="e" type="xs:string"/>`},
		// #230: <openContent> maps to {open content} (§3.4.2.3.3 clauses 5-6) in
		// every position the schema for schema documents allows one, and the
		// schema-level <defaultOpenContent> it falls back to is read rather than
		// skipped — so all four shapes are decided rather than declined.
		{"complexType with openContent", `<xs:complexType name="T"><xs:openContent mode="interleave"><xs:any/></xs:openContent><xs:sequence/></xs:complexType>`},
		{"complexType with openContent mode=none", `<xs:complexType name="T"><xs:openContent mode="none"/><xs:sequence/></xs:complexType>`},
		{"complexContent restriction with openContent", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:complexContent><xs:restriction base="B"><xs:openContent mode="suffix"><xs:any/></xs:openContent><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"top-level defaultOpenContent with any", `<xs:defaultOpenContent><xs:any/></xs:defaultOpenContent><xs:complexType name="T"><xs:sequence/></xs:complexType>`},
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
		{"top-level group without name (reference form is malformed)", `<xs:group ref="g"/>`},
		{"top-level attributeGroup with an inline attribute type outside the produced simple-type subset", `<xs:attributeGroup name="ag"><xs:attribute name="a"><xs:simpleType><xs:list itemType="xs:string"/></xs:simpleType></xs:attribute></xs:attributeGroup>`},
		{"complexType with bare nested group (no ref)", `<xs:complexType name="T"><xs:sequence><xs:group name="inner"><xs:sequence/></xs:group></xs:sequence></xs:complexType>`},
		{"complexType with simpleContent extension (produced, but cos-ct-extends unjudged)", `<xs:complexType name="T"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`},
		{"complexType with simpleContent restriction (synthesizes an anonymous simple type)", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction></xs:simpleContent></xs:complexType>`},
		{"complexType with complexContent extension (produced, but cos-ct-extends unjudged)", `<xs:complexType name="T"><xs:complexContent><xs:extension base="xs:anyType"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`},
		{"complexType with inline anonymous element type", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:complexType/></xs:element></xs:sequence></xs:complexType>`},
		{"element with inline anonymous type", `<xs:element name="e"><xs:complexType/></xs:element>`},
		{"element with both type= and inline type", `<xs:element name="e" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`},
		{"attribute with inline simpleType", `<xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`},
		// #229's deliberate asymmetries: the inline <complexType> form is unproduced
		// (#340), the GLOBAL inline <simpleType> mapping is unproduced, and an inline
		// simple type outside the produced subset moves the decline inward.
		{"local element with an inline simpleType outside the produced subset", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:simpleType><xs:union memberTypes="xs:string"/></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		{"local attribute with an inline simpleType carrying an enumeration facet", `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:string"><xs:enumeration value="x"/></xs:restriction></xs:simpleType></xs:attribute></xs:complexType>`},
		{"local element with both type= and an inline complexType (the complexType half is unproduced)", `<xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"><xs:complexType/></xs:element></xs:sequence></xs:complexType>`},
		{"list-variety simpleType", `<xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`},
		{"union-variety simpleType", `<xs:simpleType name="U"><xs:union memberTypes="xs:string"/></xs:simpleType>`},
		{"restriction with enumeration facet", `<xs:simpleType name="E"><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType>`},
		{"anonymous inline base with enumeration (recursed decline)", `<xs:simpleType name="N"><xs:restriction><xs:simpleType><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType></xs:restriction></xs:simpleType>`},
		{"one decidable + one undecidable child declines whole", `<xs:element name="e" type="xs:string"/><xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`},
		// The one composition kind parser.Parse does not follow (the redefine half
		// of #183 is unlanded): a document carrying it assembles SHORT, so admitting
		// it would be the vacuous pass the allowlist exists to refuse. Unlike
		// <include> (#242), <import> (#182) and <override> (#183).
		{"top-level redefine (not followed by the assembly)", `<xs:redefine schemaLocation="b.xsd"><xs:simpleType name="T"><xs:restriction base="xs:string"/></xs:simpleType></xs:redefine>`},
		// An <override> child the parser can only ignore, or one whose own shape is
		// outside the decidable subset, declines the whole case: after substitution
		// it would be an unmapped or undecidable TOP-LEVEL declaration.
		{"override child with no name (matches nothing, silently ignored)", `<xs:override schemaLocation="b.xsd"><xs:element type="xs:string"/></xs:override>`},
		{"override child with an inline anonymous type", `<xs:override schemaLocation="b.xsd"><xs:element name="e"><xs:complexType/></xs:element></xs:override>`},
		{"override child that is a list-variety simpleType", `<xs:override schemaLocation="b.xsd"><xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType></xs:override>`},
		{"override child of an out-of-model kind", `<xs:override schemaLocation="b.xsd"><xs:include schemaLocation="c.xsd"/></xs:override>`},
		{"include beside an undecidable kind still declines", `<xs:include schemaLocation="lib.xsd"/><xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`},
		// #230 admits <defaultOpenContent> only WITH the <any> child its content
		// model makes mandatory: the producer rejects the childless form, but only
		// once some complex type of the document reaches clause 5.2, so admitting it
		// would make the verdict depend on unrelated content.
		{"top-level defaultOpenContent with no any child", `<xs:defaultOpenContent/>`},
		// Same lazy shape, same decline: mode="none" is legal on a type's own
		// <openContent> but out of <defaultOpenContent>'s (interleave|suffix)
		// enumeration, and so is any other token — both are rejected only once some
		// complex type reaches clause 5.2.
		{"top-level defaultOpenContent with mode=none", `<xs:defaultOpenContent mode="none"><xs:any/></xs:defaultOpenContent>`},
		{"top-level defaultOpenContent with an out-of-enumeration mode", `<xs:defaultOpenContent mode="bogus"><xs:any/></xs:defaultOpenContent>`},
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
		if exec(caseSpec{kind: kindSchema, doc: malformed, expect: expectValidity(ev)}).IsPass() {
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
		if exec(caseSpec{kind: kindSchema, doc: nonSchema, expect: expectValidity(ev)}).IsPass() {
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
		c := caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(tc.expectValid)}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s (%s): executor disagreed with suite (expectValid=%v)", tc.rel, tc.why, tc.expectValid)
		}
		// A flipped expectation must Fail, proving the executor really decides.
		flipped := caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(!tc.expectValid)}
		if exec(flipped).IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", tc.rel)
		}
	}
}

// TestSchemaExecutorDeclinesUndecidableSuiteCase proves the false-accept guard on a
// real fixture: abstract00101m.xsd is suite-VALID but its <element name="root">
// carries an inline anonymous <complexType>, a form the producer does not yet
// build, so the executor must DECLINE (Fail) rather than vacuously pass — a
// valid-declared case the executor refuses to claim, recording an honest gap.
// Skips when the submodule is absent.
func TestSchemaExecutorDeclinesUndecidableSuiteCase(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newSchemaExec()
	doc := filepath.Join(suiteRoot, "sunData", "ElemDecl", "abstract", "abstract00101m", "abstract00101m.xsd")
	// Suite-valid, but undecidable (an element with an inline anonymous complexType):
	// the executor must not claim it — Fail against the true valid expectation is
	// the honest gap.
	if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValid()}).IsPass() {
		t.Error("a suite-valid case with a skipped top-level complexType must be DECLINED (Fail), never vacuously passed")
	}
}
