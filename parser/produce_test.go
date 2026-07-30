package parser_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/value"
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
	if at.Primitive() == nil || at.Primitive().Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("Foo {primitive} = %v, want {xs}string", at.Primitive())
	}
	if fs := st.OwnFacets(); len(fs) != 1 || fs[0].Kind() != xsd.FacetMinLength {
		t.Fatalf("Foo own facets = %v, want one minLength", fs)
	}
}

// simpleTypeOf produces doc's body and returns the named simple type.
func simpleTypeOf(t *testing.T, name, body string) *xsd.SimpleType {
	t.Helper()
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: name})
	if !ok {
		t.Fatalf("type %s not found", name)
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("type %s = %T, want *xsd.SimpleType", name, td)
	}
	return st
}

// TestProduceSiblingPatternsFoldIntoOneFacet pins xr-pattern (§4.3.4.2): the
// <pattern> children of one <restriction> are branches of a SINGLE pattern facet
// (same-step OR), not one facet each — one-per-sibling both misreads them as
// separate ANDed steps (cvc-pattern-valid §4.3.4.4) and, being two same-kind
// ownFacets, is rejected by st-props-correct clause 4 before that.
func TestProduceSiblingPatternsFoldIntoOneFacet(t *testing.T) {
	st := simpleTypeOf(t, "st", `<xs:simpleType name="st">
	  <xs:restriction base="xs:string">
	    <xs:minLength value="1"/>
	    <xs:pattern value="[a-z]+"/>
	    <xs:pattern value="[0-9]+"/>
	    <xs:maxLength value="8"/>
	  </xs:restriction>
	</xs:simpleType>`)

	facets := st.OwnFacets()
	kinds := make([]xsd.FacetKind, 0, len(facets))
	for _, f := range facets {
		kinds = append(kinds, f.Kind())
	}
	want := []xsd.FacetKind{xsd.FacetMinLength, xsd.FacetPattern, xsd.FacetMaxLength}
	if !slices.Equal(kinds, want) {
		t.Fatalf("facet kinds = %v, want %v (one folded pattern facet, in document order)", kinds, want)
	}
	if got := facets[1].Values(); !slices.Equal(got, []string{"[a-z]+", "[0-9]+"}) {
		t.Fatalf("pattern {value} = %q, want both branches in document order", got)
	}

	for _, lex := range []string{"abc", "123"} {
		if _, err := value.ValidateLexical(strict.New(), st, lex, nil); err != nil {
			t.Errorf("ValidateLexical(%q) = %v, want accept (matches one same-step pattern)", lex, err)
		}
	}
	if _, err := value.ValidateLexical(strict.New(), st, "a1", nil); err == nil {
		t.Error(`ValidateLexical("a1") = nil, want a cvc-pattern-valid rejection (matches neither branch)`)
	}
}

// TestProduceSinglePatternUnchanged keeps the one-<pattern> case at exactly one
// {value} member (xr-pattern case 1: R is that value).
func TestProduceSinglePatternUnchanged(t *testing.T) {
	st := simpleTypeOf(t, "st", `<xs:simpleType name="st">
	  <xs:restriction base="xs:string"><xs:pattern value="[a-z]+"/></xs:restriction>
	</xs:simpleType>`)

	facets := st.OwnFacets()
	if len(facets) != 1 || facets[0].Kind() != xsd.FacetPattern {
		t.Fatalf("own facets = %v, want one pattern facet", facets)
	}
	if got := facets[0].Values(); !slices.Equal(got, []string{"[a-z]+"}) {
		t.Fatalf("pattern {value} = %q, want [[a-z]+]", got)
	}
	if _, err := value.ValidateLexical(strict.New(), st, "abc", nil); err != nil {
		t.Errorf(`ValidateLexical("abc") = %v, want accept`, err)
	}
	if _, err := value.ValidateLexical(strict.New(), st, "ABC", nil); err == nil {
		t.Error(`ValidateLexical("ABC") = nil, want a cvc-pattern-valid rejection`)
	}
}

// TestProduceCrossStepPatternsStillANDed guards the other half of xr-pattern:
// folding same-step siblings must not fold ACROSS derivation steps, which stay
// separate effective facets and are ANDed.
func TestProduceCrossStepPatternsStillANDed(t *testing.T) {
	st := simpleTypeOf(t, "derived", `<xs:simpleType name="base">
	  <xs:restriction base="xs:string">
	    <xs:pattern value="[a-z]+"/>
	    <xs:pattern value="[0-9]+"/>
	  </xs:restriction>
	</xs:simpleType>
	<xs:simpleType name="derived">
	  <xs:restriction base="base"><xs:pattern value=".{3}"/></xs:restriction>
	</xs:simpleType>`)

	patterns := 0
	for _, ef := range st.EffectiveFacets() {
		if ef.Facet().Kind() == xsd.FacetPattern {
			patterns++
		}
	}
	if patterns != 2 {
		t.Fatalf("effective pattern facets = %d, want 2 (one per derivation step)", patterns)
	}
	if _, err := value.ValidateLexical(strict.New(), st, "abc", nil); err != nil {
		t.Errorf(`ValidateLexical("abc") = %v, want accept (matches both steps)`, err)
	}
	// "ab" satisfies the base step's OR-set but not the derived step's .{3}.
	if _, err := value.ValidateLexical(strict.New(), st, "ab", nil); err == nil {
		t.Error(`ValidateLexical("ab") = nil, want rejection: cross-step patterns are ANDed`)
	}
	// "A1c" satisfies the derived step but neither base branch.
	if _, err := value.ValidateLexical(strict.New(), st, "A1c", nil); err == nil {
		t.Error(`ValidateLexical("A1c") = nil, want rejection: the base step's patterns still apply`)
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
	// case 4). xs:anyType is now seeded as a Complex Type Definition (§3.4.7), so
	// the deferred reference discharges at finalize and the schema is accepted;
	// the element's {type definition} resolves to the seeded xs:anyType.
	s, err := produce(t, wrap("", `<xs:element name="e"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Local: "e"})
	if !ok {
		t.Fatalf("element e not found")
	}
	if got := ed.TypeDefinitionName(); got != anyTypeQN {
		t.Fatalf("type = %s, want {xs}anyType", got)
	}
	td, ok := s.Type(anyTypeQN)
	if !ok {
		t.Fatalf("xs:anyType not present in {type definitions}")
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("xs:anyType is %T, want xsd.ComplexType", td)
	}
	if ct.ContentType().Variety() != xsd.ContentMixed {
		t.Fatalf("xs:anyType {content type} variety = %s, want mixed", ct.ContentType().Variety())
	}
}

var anyTypeQN = xsd.QName{Space: xsdNS, Local: "anyType"}

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
	// annotation, group and friends are skipped, not rejected. (complexType is no
	// longer out of scope — it is produced; see the complex-type tests below.)
	body := `<xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation>` +
		`<xs:group name="g"><xs:sequence/></xs:group>` +
		`<xs:element name="e" type="xs:string"/>`
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "e"}); !ok {
		t.Fatalf("element e not produced alongside skipped out-of-scope elements")
	}
	if _, ok := s.Type(xsd.QName{Local: "g"}); ok {
		t.Fatalf("group g should have been skipped, not produced")
	}
}

// complexType reads the produced complex type named local (no namespace) from a
// schema built from body, failing on any Produce error or a missing/ wrong-kind
// type.
func complexType(t *testing.T, body, local string) xsd.ComplexType {
	t.Helper()
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: local})
	if !ok {
		t.Fatalf("complexType %s not found", local)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("%s is %T, want xsd.ComplexType", local, td)
	}
	return ct
}

// topGroup extracts the top model group of an element-content complex type.
func topGroup(t *testing.T, ct xsd.ComplexType) xsd.ModelGroup {
	t.Helper()
	ec, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("content type = %T, want ElementContent", ct.ContentType())
	}
	rt, ok := ec.Particle.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("top term = %T, want ResolvedTerm", ec.Particle.Term())
	}
	mg, ok := rt.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("top term inner = %T, want ModelGroup", rt.Term)
	}
	return mg
}

func TestProduceComplexTypeEmpty(t *testing.T) {
	ct := complexType(t, `<xs:complexType name="CT"/>`, "CT")
	if ct.ContentType().Variety() != xsd.ContentEmpty {
		t.Fatalf("variety = %s, want empty", ct.ContentType().Variety())
	}
	if ct.DerivationMethod() != xsd.DerivationRestriction {
		t.Fatalf("derivation = %s, want restriction", ct.DerivationMethod())
	}
	if ct.BaseTypeDefinitionName() != anyTypeQN {
		t.Fatalf("base = %s, want xs:anyType", ct.BaseTypeDefinitionName())
	}
}

func TestProduceComplexTypeSequence(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:element name="a" type="xs:string"/>` +
		`<xs:element name="b" type="xs:int" minOccurs="0" maxOccurs="unbounded"/>` +
		`</xs:sequence></xs:complexType>`
	ct := complexType(t, body, "CT")
	if ct.ContentType().Variety() != xsd.ContentElementOnly {
		t.Fatalf("variety = %s, want element-only", ct.ContentType().Variety())
	}
	mg := topGroup(t, ct)
	if mg.Compositor() != xsd.CompositorSequence {
		t.Fatalf("compositor = %s, want sequence", mg.Compositor())
	}
	ps := mg.Particles()
	if len(ps) != 2 {
		t.Fatalf("particles = %d, want 2", len(ps))
	}
	// Second particle b: 0..unbounded, local element decl.
	if ps[1].Occurs().Min() != 0 || !ps[1].Occurs().IsUnbounded() {
		t.Fatalf("b occurs = %s, want 0..unbounded", ps[1].Occurs())
	}
	rt := ps[0].Term().(xsd.ResolvedTerm)
	ed := rt.Term.(xsd.ElementDeclaration)
	if ed.Name() != (xsd.QName{Local: "a"}) || ed.ScopeVariety() != xsd.ScopeLocal {
		t.Fatalf("a decl = %s / %s, want {}a / local", ed.Name(), ed.ScopeVariety())
	}
}

func TestProduceComplexTypeChoiceAndAll(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		compositor xsd.Compositor
	}{
		{"choice", `<xs:complexType name="CT"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice></xs:complexType>`, xsd.CompositorChoice},
		{"all", `<xs:complexType name="CT"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`, xsd.CompositorAll},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mg := topGroup(t, complexType(t, tc.body, "CT"))
			if mg.Compositor() != tc.compositor {
				t.Fatalf("compositor = %s, want %s", mg.Compositor(), tc.compositor)
			}
		})
	}
}

func TestProduceComplexTypeMixed(t *testing.T) {
	body := `<xs:complexType name="CT" mixed="true"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`
	ct := complexType(t, body, "CT")
	if ct.ContentType().Variety() != xsd.ContentMixed {
		t.Fatalf("variety = %s, want mixed", ct.ContentType().Variety())
	}
}

func TestProduceComplexTypeMixedEmptySynthesizesSequence(t *testing.T) {
	// mixed with no content model → an empty 1..1 sequence stands in (§3.4.2.3.3
	// clause 3.1.1), and the variety is mixed, not empty.
	ct := complexType(t, `<xs:complexType name="CT" mixed="true"/>`, "CT")
	if ct.ContentType().Variety() != xsd.ContentMixed {
		t.Fatalf("variety = %s, want mixed", ct.ContentType().Variety())
	}
	mg := topGroup(t, ct)
	if mg.Compositor() != xsd.CompositorSequence || len(mg.Particles()) != 0 {
		t.Fatalf("mixed-empty group = %s/%d, want empty sequence", mg.Compositor(), len(mg.Particles()))
	}
}

func TestProduceComplexContentRestriction(t *testing.T) {
	body := `<xs:complexType name="Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>` +
		`<xs:complexType name="CT"><xs:complexContent><xs:restriction base="tns:Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, _ := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	ct := td.(xsd.ComplexType)
	if ct.BaseTypeDefinitionName() != (xsd.QName{Space: "urn:x", Local: "Base"}) {
		t.Fatalf("base = %s, want {urn:x}Base", ct.BaseTypeDefinitionName())
	}
	if ct.ContentType().Variety() != xsd.ContentElementOnly {
		t.Fatalf("variety = %s, want element-only", ct.ContentType().Variety())
	}
}

func TestProduceElementZeroOccursElided(t *testing.T) {
	// An element and a nested group each with minOccurs=maxOccurs=0 map to no
	// component at all (§3.9.2/§3.8.2): they must not appear in {particles}.
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:element name="keep" type="xs:string"/>` +
		`<xs:element name="drop" type="xs:string" minOccurs="0" maxOccurs="0"/>` +
		`<xs:choice minOccurs="0" maxOccurs="0"><xs:element name="x" type="xs:string"/></xs:choice>` +
		`</xs:sequence></xs:complexType>`
	mg := topGroup(t, complexType(t, body, "CT"))
	ps := mg.Particles()
	if len(ps) != 1 {
		t.Fatalf("particles = %d, want 1 (zero-occurs element and group elided)", len(ps))
	}
	ed := ps[0].Term().(xsd.ResolvedTerm).Term.(xsd.ElementDeclaration)
	if ed.Name() != (xsd.QName{Local: "keep"}) {
		t.Fatalf("surviving particle = %s, want keep", ed.Name())
	}
}

func TestProduceLocalAttributeUses(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/>` +
		`<xs:attribute name="a" type="xs:string" use="required"/>` +
		`<xs:attribute name="b" type="xs:int"/>` +
		`<xs:attribute name="gone" type="xs:string" use="prohibited"/>` +
		`</xs:complexType>`
	ct := complexType(t, body, "CT")
	uses := ct.AttributeUses()
	if len(uses) != 2 {
		t.Fatalf("attribute uses = %d, want 2 (prohibited elided)", len(uses))
	}
	if !uses[0].Required() {
		t.Fatalf("use a should be required")
	}
	if uses[1].Required() {
		t.Fatalf("use b should be optional")
	}
	decl := uses[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration
	if decl.ScopeVariety() != xsd.ScopeLocal || decl.Name() != (xsd.QName{Local: "a"}) {
		t.Fatalf("a decl = %s / %s, want {}a / local", decl.Name(), decl.ScopeVariety())
	}
}

func TestProduceAttributeRefUse(t *testing.T) {
	body := `<xs:attribute name="g" type="xs:string"/>` +
		`<xs:complexType name="CT"><xs:sequence/><xs:attribute ref="tns:g"/></xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, _ := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	uses := td.(xsd.ComplexType).AttributeUses()
	if len(uses) != 1 {
		t.Fatalf("uses = %d, want 1", len(uses))
	}
	ref, ok := uses[0].AttributeDeclaration().(xsd.AttributeDeclarationRef)
	if !ok || ref.Name != (xsd.QName{Space: "urn:x", Local: "g"}) {
		t.Fatalf("attr use decl = %v, want ref {urn:x}g", uses[0].AttributeDeclaration())
	}
}

func TestProduceAnyAttributeWildcard(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute namespace="##other" processContents="lax"/></xs:complexType>`
	ct := complexType(t, body, "CT")
	wc, ok := ct.AttributeWildcard()
	if !ok {
		t.Fatalf("attribute wildcard absent, want present")
	}
	if wc.ProcessContents() != xsd.ProcessLax {
		t.Fatalf("processContents = %s, want lax", wc.ProcessContents())
	}
	// ##other in a no-target-namespace schema admits any present namespace but not
	// ·absent· (unqualified) names.
	if wc.AllowsName(xsd.QName{Local: "x"}) {
		t.Fatalf("##other should reject an unqualified (absent-namespace) name")
	}
	if !wc.AllowsName(xsd.QName{Space: "urn:z", Local: "x"}) {
		t.Fatalf("##other should admit a foreign-namespace name")
	}
}

func TestProduceAnyElementWildcardParticle(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence><xs:any namespace="##any" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType>`
	mg := topGroup(t, complexType(t, body, "CT"))
	ps := mg.Particles()
	if len(ps) != 1 {
		t.Fatalf("particles = %d, want 1", len(ps))
	}
	wc, ok := ps[0].Term().(xsd.ResolvedTerm).Term.(xsd.Wildcard)
	if !ok {
		t.Fatalf("term = %T, want Wildcard", ps[0].Term().(xsd.ResolvedTerm).Term)
	}
	if !wc.AllowsName(xsd.QName{Space: "urn:z", Local: "x"}) {
		t.Fatalf("##any wildcard should admit any name")
	}
}

func TestProduceComplexContentMixedMismatchRejected(t *testing.T) {
	// src-ct clause 5: mixed on both <complexType> and <complexContent> must agree.
	body := `<xs:complexType name="Base"><xs:sequence/></xs:complexType>` +
		`<xs:complexType name="CT" mixed="true"><xs:complexContent mixed="false"><xs:restriction base="tns:Base"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`
	_, err := produce(t, wrap("urn:x", body))
	assertRule(t, err, "src-ct")
}

func TestProduceAllNestedRejected(t *testing.T) {
	// cos-all-limited: an <all> may not be nested inside a <sequence>/<choice>.
	body := `<xs:complexType name="CT"><xs:sequence><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:sequence></xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "cos-all-limited")
}

func TestProduceWildcardBothNamespaceFormsRejected(t *testing.T) {
	// src-wildcard: namespace and notNamespace must not both be present.
	body := `<xs:complexType name="CT"><xs:sequence><xs:any namespace="##any" notNamespace="urn:z"/></xs:sequence></xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-wildcard")
}

func TestProduceSimpleContentDeclined(t *testing.T) {
	// <simpleContent> needs the resolved base for its {simple type definition}
	// (§3.4.2.2) — not yet produced. Declined with a non-xsderr limitation error.
	body := `<xs:complexType name="CT"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`
	_, err := produce(t, wrap("", body))
	if err == nil {
		t.Fatalf("expected a decline error for <simpleContent>, got nil")
	}
	if _, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("simpleContent decline should be a plain limitation error, not an xsderr rule: %v", err)
	}
}

func TestProduceComplexContentExtensionDeclined(t *testing.T) {
	// <complexContent><extension> needs the resolved base particle (§3.4.2.3.3
	// clause 4.2) — not yet produced. Declined with a non-xsderr limitation error.
	body := `<xs:complexType name="Base"><xs:sequence/></xs:complexType>` +
		`<xs:complexType name="CT"><xs:complexContent><xs:extension base="tns:Base"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`
	_, err := produce(t, wrap("urn:x", body))
	if err == nil {
		t.Fatalf("expected a decline error for <complexContent><extension>, got nil")
	}
	if _, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("extension decline should be a plain limitation error: %v", err)
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

// TestProduceNotQNameKeywords pins the §3.10.2.2 notQName mapping: the ##defined
// and ##definedSibling keyword tokens are mapped (no longer skipped) alongside
// the literal QName members, which keep their cvc-wildcard-name clause-2 effect.
func TestProduceNotQNameKeywords(t *testing.T) {
	// A target namespace is needed so the tns: prefix of the literal member
	// resolves (an unresolvable prefix is dropped — #232, out of scope here).
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:any notQName="##defined ##definedSibling tns:foo"/>` +
		`</xs:sequence></xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	if !ok {
		t.Fatalf("complex type CT not found")
	}
	mg := topGroup(t, td.(xsd.ComplexType))
	wc, ok := mg.Particles()[0].Term().(xsd.ResolvedTerm).Term.(xsd.Wildcard)
	if !ok {
		t.Fatalf("term = %T, want Wildcard", mg.Particles()[0].Term())
	}
	// The literal member still governs cvc-wildcard-name (§3.10.4.2) clause 2;
	// the keywords are resolved by cvc-wildcard (§3.10.4.1) clauses 2-3, a
	// different rule at the declaration-graph layer, so they leave AllowsName
	// alone.
	if wc.AllowsName(xsd.QName{Space: "urn:x", Local: "foo"}) {
		t.Errorf("AllowsName admitted the literal notQName member {urn:x}foo")
	}
	if !wc.AllowsName(xsd.QName{Space: "urn:x", Local: "bar"}) {
		t.Errorf("AllowsName rejected a name in no half of {disallowed names}")
	}
}

// TestProduceNotQNameRejections pins that an unrecognized ## token — including
// ##definedSibling on an <anyAttribute>, whose notQName is typed xs:qnameListA
// (§3.10.2, the machine-checkable form of w-props-correct clause 5) — is a
// datatype-validity rejection, never a silent skip.
func TestProduceNotQNameRejections(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"unknown-token-on-any", `<xs:complexType name="CT"><xs:sequence><xs:any notQName="##foo"/></xs:sequence></xs:complexType>`},
		{"unknown-token-on-anyAttribute", `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute notQName="##foo"/></xs:complexType>`},
		{"definedSibling-on-anyAttribute", `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute notQName="##definedSibling"/></xs:complexType>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", c.body))
			if err == nil {
				t.Fatalf("Produce accepted %s, want a cvc-datatype-valid rejection", c.body)
			}
			if rule, ok := xsderr.RuleOf(err); !ok || rule != xsderr.Rule("cvc-datatype-valid") {
				t.Errorf("RuleOf = (%q, %v), want (%q, true)", rule, ok, "cvc-datatype-valid")
			}
		})
	}
	// ##defined stays legal on an attribute wildcard (cvc-wildcard clause 2.2).
	if _, err := produce(t, wrap("urn:x", `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute notQName="##defined"/></xs:complexType>`)); err != nil {
		t.Errorf("Produce rejected notQName=\"##defined\" on <anyAttribute>: %v", err)
	}
}

// TestProduceChargesFacetValueRestriction proves the producer's sole
// xsd.NewSimpleType call site is followed by builtin.CheckSimpleTypeRestriction:
// a widening bound (cos-st-restricts clause 1.3.2 via
// minInclusive-valid-restriction §4.3.10.4) and an inapplicable facet (clause
// 1.3.1 via cos-applicable-facets §4.1.5) are BOTH rejected here, neither of
// which xsd.NewSimpleType can decide on its own.
func TestProduceChargesFacetValueRestriction(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantRule xsderr.Rule
	}{
		{
			"bound widens the base value space",
			`<xs:simpleType name="wide">
			   <xs:restriction base="xs:byte">
			     <xs:minInclusive value="-200"/>
			   </xs:restriction>
			 </xs:simpleType>`,
			"minInclusive-valid-restriction",
		},
		{
			"facet inapplicable to the primitive",
			`<xs:simpleType name="bad">
			   <xs:restriction base="xs:string">
			     <xs:maxInclusive value="5"/>
			   </xs:restriction>
			 </xs:simpleType>`,
			"cos-st-restricts",
		},
		{
			"length may not move across a restriction",
			`<xs:simpleType name="lenbase">
			   <xs:restriction base="xs:string">
			     <xs:length value="5"/>
			   </xs:restriction>
			 </xs:simpleType>
			 <xs:simpleType name="lenderived">
			   <xs:restriction base="tns:lenbase">
			     <xs:length value="4"/>
			   </xs:restriction>
			 </xs:simpleType>`,
			"length-valid-restriction",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:facets", c.body))
			if err == nil {
				t.Fatalf("Produce accepted an invalid restriction, want %s", c.wantRule)
			}
			got, ok := xsderr.RuleOf(err)
			if !ok || got != c.wantRule {
				t.Fatalf("rule = %q (ok=%v), want %q; err=%v", got, ok, c.wantRule, err)
			}
		})
	}
}

// TestProduceAcceptsNarrowingRestriction is the false-reject guard for the same
// seam: a legitimately narrowing restriction of a builtin must still produce.
func TestProduceAcceptsNarrowingRestriction(t *testing.T) {
	_, err := produce(t, wrap("urn:facets", `<xs:simpleType name="small">
	   <xs:restriction base="xs:byte">
	     <xs:minInclusive value="0"/>
	     <xs:maxInclusive value="100"/>
	   </xs:restriction>
	 </xs:simpleType>`))
	if err != nil {
		t.Fatalf("narrowing restriction rejected: %v", err)
	}
}

// TestProduceScaleFacetsReachTheFacetSet pins xr-maxScale/xr-minScale
// (xsd-precisionDecimal.md §4.2.2/§4.3.2): the two precisionDecimal extension
// facets are plain-lexical value=/fixed= facets, so <maxScale>/<minScale>
// children must land in {facets} carrying both properties. Before this, they hit
// facetKindOf's unknown-name branch and were silently dropped, which also left
// every scale Schema Component Constraint unreachable from a parsed schema.
func TestProduceScaleFacetsReachTheFacetSet(t *testing.T) {
	st := simpleTypeOf(t, "Scaled", `<xs:simpleType name="Scaled">
	   <xs:restriction base="xs:precisionDecimal">
	     <xs:maxScale value="4" fixed="true"/>
	     <xs:minScale value="2"/>
	   </xs:restriction>
	 </xs:simpleType>`)
	want := []struct {
		kind  xsd.FacetKind
		value string
		fixed bool
	}{
		{xsd.FacetMaxScale, "4", true},
		{xsd.FacetMinScale, "2", false},
	}
	facets := st.OwnFacets()
	if len(facets) != len(want) {
		t.Fatalf("own facets = %v, want %d (maxScale, minScale)", facets, len(want))
	}
	for i, w := range want {
		got := facets[i]
		if got.Kind() != w.kind {
			t.Errorf("facet %d kind = %v, want %v", i, got.Kind(), w.kind)
		}
		if vs := got.Values(); len(vs) != 1 || vs[0] != w.value {
			t.Errorf("facet %d {value} = %v, want [%q]", i, vs, w.value)
		}
		fixed, ok := got.Fixed()
		if !ok {
			t.Errorf("facet %d reports no {fixed}, but %v carries one", i, w.kind)
		}
		if fixed != w.fixed {
			t.Errorf("facet %d {fixed} = %v, want %v", i, fixed, w.fixed)
		}
	}
}

// TestProduceChargesScaleRestriction is the end-to-end proof that the
// construction-time scale SCCs (xsd/derivation.go, issue #157) now see parsed
// input: a derived maxScale WIDER than the base's violates
// maxScale-valid-restriction (xsd-precisionDecimal.md §4.2.4) and must be
// rejected at produce time. While the facet was dropped in the producer this
// schema was silently accepted.
func TestProduceChargesScaleRestriction(t *testing.T) {
	_, err := produce(t, wrap("urn:facets", `<xs:simpleType name="tight">
	   <xs:restriction base="xs:precisionDecimal">
	     <xs:maxScale value="2"/>
	   </xs:restriction>
	 </xs:simpleType>
	 <xs:simpleType name="wider">
	   <xs:restriction base="tns:tight">
	     <xs:maxScale value="5"/>
	   </xs:restriction>
	 </xs:simpleType>`))
	if err == nil {
		t.Fatalf("Produce accepted a widening maxScale, want maxScale-valid-restriction")
	}
	got, ok := xsderr.RuleOf(err)
	if !ok || got != xsderr.Rule("maxScale-valid-restriction") {
		t.Fatalf("rule = %q (ok=%v), want %q; err=%v", got, ok, "maxScale-valid-restriction", err)
	}
}
