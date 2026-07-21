package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

const produceURI = "mem://produce.xsd"

// produce reads doc and runs Produce with the strict backend, returning the
// finalized schema or the first error.
func produce(t *testing.T, doc string) (*xsd.Schema, error) {
	t.Helper()
	d, err := parser.ReadDocument(produceURI, strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	return parser.Produce(d, strict.New())
}

// wrap wraps body children inside a <schema> with the xs prefix bound and an
// optional targetNamespace.
func wrap(target, body string) string {
	tns := ""
	if target != "" {
		tns = ` targetNamespace="` + target + `" xmlns:tns="` + target + `"`
	}
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + tns + `>` + body + `</xs:schema>`
}

const xsdNS = "http://www.w3.org/2001/XMLSchema"

func TestProduceTopLevelElement(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:element name="root" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Local: "root"})
	if !ok {
		t.Fatalf("element root not found")
	}
	if got := ed.TypeDefinitionName(); got != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("type = %s, want {xs}string", got)
	}
	if ed.ScopeVariety() != xsd.ScopeGlobal {
		t.Fatalf("scope = %s, want global", ed.ScopeVariety())
	}
}

func TestProduceTopLevelAttribute(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:attribute name="count" type="xs:int"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ad, ok := s.Attribute(xsd.QName{Local: "count"})
	if !ok {
		t.Fatalf("attribute count not found")
	}
	if got := ad.TypeDefinitionName(); got != (xsd.QName{Space: xsdNS, Local: "int"}) {
		t.Fatalf("type = %s, want {xs}int", got)
	}
}

func TestProduceElementTargetNamespace(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `<xs:element name="root" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:x", Local: "root"}); !ok {
		t.Fatalf("element {urn:x}root not found; targetNamespace not applied to {name}")
	}
}

func TestProduceSimpleTypeWithFacetAndBackReference(t *testing.T) {
	// A named simpleType, then an element referencing it (backward reference),
	// proving resolution through finalize.
	body := `<xs:simpleType name="Foo"><xs:restriction base="xs:string"><xs:minLength value="1"/></xs:restriction></xs:simpleType>` +
		`<xs:element name="e" type="tns:Foo"/>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Space: "urn:x", Local: "Foo"})
	if !ok {
		t.Fatalf("type {urn:x}Foo not found")
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("Foo is not a *SimpleType")
	}
	if base := st.Base(); base == nil || base.Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("Foo base = %v, want {xs}string", base)
	}
	// The base's own primitive pointer must be propagated (warden finding #4).
	at, ok := st.Variety().(xsd.Atomic)
	if !ok {
		t.Fatalf("Foo variety = %T, want Atomic", st.Variety())
	}
	if at.Primitive == nil || at.Primitive.Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("Foo {primitive} = %v, want {xs}string", at.Primitive)
	}
	if fs := st.OwnFacets(); len(fs) != 1 || fs[0].Kind() != xsd.FacetMinLength {
		t.Fatalf("Foo own facets = %v, want one minLength", fs)
	}
}

func TestProduceSimpleTypeForwardReferenceChain(t *testing.T) {
	// B is declared before A in document order, but A restricts B and B restricts
	// xs:string. Additionally a C forward-references A. Proves the topological
	// build resolves both directions.
	body := `<xs:simpleType name="B"><xs:restriction base="tns:A"><xs:maxLength value="9"/></xs:restriction></xs:simpleType>` +
		`<xs:simpleType name="A"><xs:restriction base="xs:string"><xs:minLength value="1"/></xs:restriction></xs:simpleType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	bTD, _ := s.Type(xsd.QName{Space: "urn:x", Local: "B"})
	bST := bTD.(*xsd.SimpleType)
	aTD, _ := s.Type(xsd.QName{Space: "urn:x", Local: "A"})
	aST := aTD.(*xsd.SimpleType)
	if bST.Base() != aST {
		t.Fatalf("B.Base() is not the same *SimpleType as A (pointer identity broken)")
	}
	if aST.Base() == nil || aST.Base().Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("A base = %v, want {xs}string", aST.Base())
	}
}

func TestProduceSimpleTypeCircularRejected(t *testing.T) {
	// A restricts B, B restricts A: a circular base chain.
	body := `<xs:simpleType name="A"><xs:restriction base="tns:B"/></xs:simpleType>` +
		`<xs:simpleType name="B"><xs:restriction base="tns:A"/></xs:simpleType>`
	_, err := produce(t, wrap("urn:x", body))
	assertRule(t, err, "st-props-correct")
}

func TestProduceRestrictionBaseAndInlineRejected(t *testing.T) {
	body := `<xs:simpleType name="Bad"><xs:restriction base="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:restriction></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceRestrictionNeitherBaseNorInlineRejected(t *testing.T) {
	body := `<xs:simpleType name="Bad"><xs:restriction/></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceSimpleTypeListRejected(t *testing.T) {
	body := `<xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceEnumerationFacetRejected(t *testing.T) {
	body := `<xs:simpleType name="E"><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceElementTypeAndInlineRejected(t *testing.T) {
	body := `<xs:element name="e" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-element")
}

func TestProduceElementInlineOnlyRejected(t *testing.T) {
	body := `<xs:element name="e"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-element")
}

func TestProduceElementDefaultAndFixedRejected(t *testing.T) {
	body := `<xs:element name="e" type="xs:string" default="a" fixed="b"/>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-element")
}

func TestProduceAttributeTypeAndInlineRejected(t *testing.T) {
	body := `<xs:attribute name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-attribute")
}

func TestProduceBadPrefixRejected(t *testing.T) {
	body := `<xs:element name="e" type="nope:string"/>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-resolve")
}

func TestProduceUnresolvableBaseRejected(t *testing.T) {
	body := `<xs:simpleType name="S"><xs:restriction base="tns:Missing"/></xs:simpleType>`
	_, err := produce(t, wrap("urn:x", body))
	assertRule(t, err, "src-resolve")
}

func TestProduceElementNoTypeDefaultsAnyType(t *testing.T) {
	// A bare <element> defaults its {type definition} to xs:anyType (§3.3.2.1
	// case 4). xs:anyType is a Complex Type Definition and is NOT produced by this
	// slice (builtin.Seed yields only simple types), so the deferred reference does
	// not discharge at finalize: the schema is rejected with src-resolve whose
	// message names anyType, proving the default mapping fired. This limitation
	// lifts once complex types (and xs:anyType) are produced.
	_, err := produce(t, wrap("", `<xs:element name="e"/>`))
	assertRule(t, err, "src-resolve")
	if !strings.Contains(err.Error(), "anyType") {
		t.Fatalf("error %v does not name anyType; default mapping may not have fired", err)
	}
}

func TestProduceAttributeNoTypeDefaultsAnySimpleType(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:attribute name="a"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ad, _ := s.Attribute(xsd.QName{Local: "a"})
	if got := ad.TypeDefinitionName(); got != (xsd.QName{Space: xsdNS, Local: "anySimpleType"}) {
		t.Fatalf("type = %s, want {xs}anySimpleType", got)
	}
}

func TestProduceAnonymousInlineBaseRestriction(t *testing.T) {
	// A restriction whose base is an inline anonymous <simpleType>.
	body := `<xs:simpleType name="Wrap"><xs:restriction><xs:simpleType><xs:restriction base="xs:string"><xs:minLength value="2"/></xs:restriction></xs:simpleType><xs:maxLength value="5"/></xs:restriction></xs:simpleType>`
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, _ := s.Type(xsd.QName{Local: "Wrap"})
	st := td.(*xsd.SimpleType)
	anon := st.Base()
	if anon == nil || anon.Name() != (xsd.QName{}) {
		t.Fatalf("Wrap base is not an anonymous (zero-QName) simple type: %v", anon)
	}
	if anon.Base() == nil || anon.Base().Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("anonymous base's base = %v, want {xs}string", anon.Base())
	}
}

func TestProduceNonSchemaRootRejected(t *testing.T) {
	d, err := parser.ReadDocument(produceURI, strings.NewReader(`<notschema/>`))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	if _, err := parser.Produce(d, strict.New()); err == nil {
		t.Fatalf("Produce accepted a non-<schema> root")
	}
}

func TestProduceSkipsOutOfScope(t *testing.T) {
	// annotation, complexType, group and friends are skipped, not rejected.
	body := `<xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation>` +
		`<xs:complexType name="CT"/>` +
		`<xs:element name="e" type="xs:string"/>`
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "e"}); !ok {
		t.Fatalf("element e not produced alongside skipped out-of-scope elements")
	}
	if _, ok := s.Type(xsd.QName{Local: "CT"}); ok {
		t.Fatalf("complexType CT should have been skipped, not produced")
	}
}

// assertRule fails unless err is non-nil and its first *xsderr.Error carries the
// expected rule.
func assertRule(t *testing.T, err error, want xsderr.Rule) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error charged %s, got nil", want)
	}
	got, ok := xsderr.RuleOf(err)
	if !ok {
		t.Fatalf("error %v carries no xsderr rule, want %s", err, want)
	}
	if got != want {
		t.Fatalf("error charged %s, want %s (%v)", got, want, err)
	}
}
