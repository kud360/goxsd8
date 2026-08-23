package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin §3.4.2.2's <restriction> half — cases 1 and 2, which
// SYNTHESIZE an anonymous simple type from the <restriction>'s own facet
// children rather than reusing one (#909) — together with the two case-5 arms
// that alternant can fall through to and the src-ct clause 2 representation
// check the source item carries. The <extension> half (cases 3-5) is pinned in
// produce_extension_test.go.

// tnsComplexType fetches a top-level complex type of the wrap("urn:x", …)
// documents these tests build, by local name. topComplexType
// (produce_groups_test.go) reads the no-namespace documents its own callers use.
func tnsComplexType(t *testing.T, s *xsd.Schema, local string) xsd.ComplexType {
	t.Helper()
	td, ok := s.Type(xq(local))
	if !ok {
		t.Fatalf("complex type %s not found", xq(local))
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type (%T)", xq(local), td)
	}
	return ct
}

// simpleContentTypeOf returns the {content type}.{simple type definition} of a
// top-level complex type, failing when the {variety} is not simple.
func simpleContentTypeOf(t *testing.T, s *xsd.Schema, local string) *xsd.SimpleType {
	t.Helper()
	ct := contentTypeOf(t, s, xq(local))
	sc, ok := ct.(xsd.SimpleContent)
	if !ok {
		t.Fatalf("%s {content type} is %T, want SimpleContent (§3.4.2.2 gives {variety} simple)", local, ct)
	}
	return sc.SimpleType
}

// TestProduceSimpleContentRestrictionCase1 pins §3.4.2.2 case 1 clause 1.2: a
// <restriction> under <simpleContent> whose base is a complex type with simple
// content and which carries NO inline <simpleType> synthesizes a new anonymous
// simple type whose {base type definition} B is the BASE's own
// {content type}.{simple type definition} — the very component, not a rebuilt
// twin — and whose {facets} are this <restriction>'s facet children.
func TestProduceSimpleContentRestrictionCase1(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string">
			<xs:attribute name="unit" type="xs:string"/>
		</xs:extension></xs:simpleContent></xs:complexType>
		<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
			<xs:maxLength value="4"/>
		</xs:restriction></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	synthesized := simpleContentTypeOf(t, s, "D")
	if synthesized == simpleContentTypeOf(t, s, "B") {
		t.Fatal("D's {simple type definition} is the base's own component, but case 1 synthesizes a NEW type restricting it")
	}
	if synthesized.Name() != (xsd.QName{}) {
		t.Errorf("synthesized {name} = %s, want ·absent· (the zero QName)", synthesized.Name())
	}
	if len(synthesized.Final()) != 0 {
		t.Errorf("synthesized {final} = %v, want the empty set", synthesized.Final())
	}
	base, err := synthesized.Base(s)
	if err != nil {
		t.Fatalf("{base type definition}: %v", err)
	}
	if base != simpleContentTypeOf(t, s, "B") {
		t.Errorf("synthesized {base type definition} = %s, want the very component B's {content type} holds (clause 1.2)", base.Name())
	}
	if v := facetValue(t, synthesized, xsd.FacetMaxLength); v != "4" {
		t.Errorf("synthesized {facets} maxLength = %q, want 4 from the <restriction>'s facet child (§3.16.6.4)", v)
	}
	d := tnsComplexType(t, s, "D")
	if m := d.DerivationMethod(); m != xsd.DerivationRestriction {
		t.Errorf("D {derivation method} = %s, want restriction", m)
	}
	// §3.4.2.4 clause 3.1 hands the base's attribute use through to a restriction
	// that does not restate it, so the produced type is usable as one.
	if !hasAttrUse(d.AttributeUses(), "unit") {
		t.Error("D is missing the base's attribute use 'unit'")
	}
}

// TestProduceSimpleContentRestrictionCase1InlineSimpleType pins clause 1.1: the
// <simpleType> among the <restriction>'s [children] is B when there is one, in
// ADDITION to the required base= (which names the complex type being
// restricted), never instead of it — the schema for schema documents makes base
// use="required" on xs:simpleRestrictionType, unlike the datatype-level
// xs:restriction where base= and <simpleType> are alternatives.
func TestProduceSimpleContentRestrictionCase1InlineSimpleType(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
		<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
			<xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction></xs:simpleType>
		</xs:restriction></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	synthesized := simpleContentTypeOf(t, s, "D")
	b, err := synthesized.Base(s)
	if err != nil {
		t.Fatalf("{base type definition}: %v", err)
	}
	if b == simpleContentTypeOf(t, s, "B") {
		t.Fatal("clause 1.1's B is the base complex type's own {simple type definition}, want the inline <simpleType> the <restriction> carries")
	}
	if b.Name() != (xsd.QName{}) {
		t.Errorf("inline B {name} = %s, want ·absent·", b.Name())
	}
	if v := facetValue(t, b, xsd.FacetMaxLength); v != "4" {
		t.Errorf("inline B carries maxLength %q, want the inline <simpleType>'s own 4", v)
	}
	// The base= still decides {base type definition} of the COMPLEX type.
	if got := tnsComplexType(t, s, "D").Base(); got != (xsd.TypeDefinitionRef{Name: xq("B")}) {
		t.Errorf("D {base type definition} = %#v, want the base= QName tns:B", got)
	}
}

// TestProduceSimpleContentRestrictionCase2 pins case 2: the base is a complex
// type whose own {content type} has {variety} MIXED, so the tableau takes SB
// from the inline <simpleType> and the restriction narrows a mixed type down to
// simple content. The case is chosen by the BASE's content-type variety, never
// by the presence of that inline <simpleType> — the same child appears in case
// 1's clause 1.1.
//
// derivation-ok-restriction clause 2.2.2.2 is what makes this schema valid
// rather than a mis-derivation: B's {particle} is the empty 1..1 sequence
// §3.4.2.3.3 clause 3.1.1 substitutes for a mixed type with no model group, and
// that is ·emptiable·. This is the shape ibmData/valid/S3_12/s3_12v02.xsd uses.
func TestProduceSimpleContentRestrictionCase2(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B" mixed="true"><xs:attribute name="kind" type="xs:string"/></xs:complexType>
		<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
			<xs:simpleType><xs:restriction base="xs:integer"/></xs:simpleType>
		</xs:restriction></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	synthesized := simpleContentTypeOf(t, s, "D")
	if synthesized.IsAnySimpleType() {
		t.Fatal("D's {simple type definition} is ·xs:anySimpleType· itself, which is case 5's result: a mixed base with an ·emptiable· particle is case 2, which SYNTHESIZES a restriction of SB")
	}
	sb, err := synthesized.Base(s)
	if err != nil {
		t.Fatalf("{base type definition}: %v", err)
	}
	sbBase, err := sb.Base(s)
	if err != nil {
		t.Fatalf("SB's own {base type definition}: %v", err)
	}
	if sbBase.Name() != (xsd.QName{Space: xsdNS, Local: "integer"}) {
		t.Fatalf("case 2's SB restricts %s, want the inline <simpleType>'s xs:integer", sbBase.Name())
	}
}

// TestProduceSimpleContentRestrictionCase2WithoutInlineSimpleType pins the
// pathology §3.4.2.2 case 2 documents in a Note of its own: with no inline
// <simpleType>, SB is ·xs:anySimpleType·, and "the result will be a simple type
// definition component which fails to obey the constraints on simple type
// definitions, including for example clause 1.1 of Derivation Valid
// (Restriction, Simple)". The producer must BUILD that component anyway —
// §3.4.2.2 is an unconditional mapping — and let the ordinary simple-type
// derivation checks reject it, rather than refusing the representation and
// charging a rule the source does not violate.
//
// The verdict is st-props-correct clause 1, which the Note's "including for
// example" leaves room for and which CheckDerivation charges first: restricting
// ·xs:anySimpleType· leaves the synthesized type's {variety} ·absent·, and only
// xs:anySimpleType itself may have that. What the test pins is that a genuine
// simple-type constraint rejects the component — not which of the several it
// breaks is reported.
func TestProduceSimpleContentRestrictionCase2WithoutInlineSimpleType(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B" mixed="true"><xs:attribute name="kind" type="xs:string"/></xs:complexType>
		<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
			<xs:maxLength value="4"/>
		</xs:restriction></xs:simpleContent></xs:complexType>`))
	assertRule(t, err, "st-props-correct")
	if !strings.Contains(err.Error(), "anySimpleType") {
		t.Fatalf("error = %v, want it to name the ·xs:anySimpleType· the missing inline <simpleType> defaulted SB to", err)
	}
}

// TestProduceSimpleContentRestrictionCase5 pins the two bases the <restriction>
// alternant falls through to case 5 ("otherwise ·xs:anySimpleType·") on: a base
// that is a simple type definition, and a base complex type whose own {content
// type} is element-only. The tableau MAPS both — it states a result for every
// base — and each is rejected downstream by the rule that really governs it, so
// the mapping fabricates no representation error of its own.
//
// The third arm carries the inline <simpleType> case 5 DROPS: restrictedSimpleBase
// discriminates the case off the base alone and returns before that child is
// read, so it is never built and its own errors never surface. The arm's inline
// type WOULD be rejected if it were built (st-props-correct clause 1, restricting
// ·xs:anySimpleType· — the verdict
// TestProduceSimpleContentRestrictionCase2WithoutInlineSimpleType pins), and the
// arm asserts case 5's own governing rule lands instead, which is what makes the
// drop safe rather than merely unobserved. It claims nothing for case 2 above,
// which READS the same child.
func TestProduceSimpleContentRestrictionCase5(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		rule       xsderr.Rule
		clause     string
	}{
		{
			// ct-props-correct clause 2: a simple {base type definition} demands
			// {derivation method} extension.
			"simple type base", `
			<xs:complexType name="D"><xs:simpleContent><xs:restriction base="xs:string">
				<xs:maxLength value="4"/>
			</xs:restriction></xs:simpleContent></xs:complexType>`,
			"ct-props-correct", "clause 2",
		},
		{
			// derivation-ok-restriction clause 2: T is simple content, and no branch
			// admits an element-only B — 2.2.2.1 wants a B with simple content and
			// 2.2.2.2 wants a mixed one.
			"element-only base", `
			<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
			<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
				<xs:maxLength value="4"/>
			</xs:restriction></xs:simpleContent></xs:complexType>`,
			"derivation-ok-restriction", "clause 2",
		},
		{
			// ct-props-correct clause 2 again, over a <restriction> whose inline
			// <simpleType> would be rejected by st-props-correct clause 1 if case 5
			// built it.
			"simple type base with a dropped inline simpleType", `
			<xs:complexType name="D"><xs:simpleContent><xs:restriction base="xs:string">
				<xs:simpleType><xs:restriction base="xs:anySimpleType"/></xs:simpleType>
				<xs:maxLength value="4"/>
			</xs:restriction></xs:simpleContent></xs:complexType>`,
			"ct-props-correct", "clause 2",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", tc.body))
			assertRule(t, err, tc.rule)
			if !strings.Contains(err.Error(), tc.clause) {
				t.Fatalf("error = %v, want it to cite %s %s", err, tc.rule, tc.clause)
			}
		})
	}
}

// TestProduceSimpleContentRestrictionDuplicateFacet pins src-ct clause 2
// (§3.4.3): under a <simpleContent> <restriction>, no facet-specifying element
// other than xs:enumeration, xs:pattern or xs:assertion may appear more than
// once. It is a Schema Representation Constraint on the source item, so the
// verdict must be src-ct — not the st-props-correct clause 4 that
// xsd.NewSimpleType would charge for the same document one step later, about the
// component rather than the representation.
func TestProduceSimpleContentRestrictionDuplicateFacet(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
		<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
			<xs:maxLength value="4"/><xs:maxLength value="5"/>
		</xs:restriction></xs:simpleContent></xs:complexType>`))
	assertRule(t, err, "src-ct")
	if !strings.Contains(err.Error(), "clause 2") {
		t.Fatalf("error = %v, want it to cite src-ct clause 2", err)
	}
}

// TestProduceSimpleContentRestrictionRepeatedExceptedFacets pins the other half
// of src-ct clause 2: its three EXCEPTED names may repeat, and each folds into
// the single facet its own §4.3 rule describes rather than into two components
// st-props-correct clause 4 would reject.
func TestProduceSimpleContentRestrictionRepeatedExceptedFacets(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
		<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
			<xs:pattern value="a*"/><xs:pattern value="b*"/>
			<xs:enumeration value="aa"/><xs:enumeration value="bb"/>
			<xs:assertion test="true()"/><xs:assertion test="true()"/>
		</xs:restriction></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	synthesized := simpleContentTypeOf(t, s, "D")
	counts := map[xsd.FacetKind]int{}
	for _, f := range synthesized.OwnFacets() {
		counts[f.Kind()]++
	}
	for _, kind := range []xsd.FacetKind{xsd.FacetPattern, xsd.FacetEnumeration, xsd.FacetAssertions} {
		if counts[kind] != 1 {
			t.Errorf("synthesized type has %d %s facets, want the one folded component", counts[kind], kind)
		}
	}
}

// TestProduceSimpleContentRestrictionRepeatedAlternant pins that #904's
// repeated-alternant guard still fires on this arm now that <restriction> is
// produced: xs:simpleContent holds a plain xs:choice, so a wrapper carrying both
// alternants is a grammar fault charged AHEAD of the alternant read — without
// it, derivationAlternant's <restriction>-first read would map from one and drop
// the other in silence.
func TestProduceSimpleContentRestrictionRepeatedAlternant(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="D"><xs:simpleContent>
			<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction>
			<xs:extension base="xs:string"/>
		</xs:simpleContent></xs:complexType>`))
	if err == nil {
		t.Fatal("Produce accepted a <simpleContent> carrying both alternants")
	}
	if _, ok := xsderr.RuleOf(err); ok {
		t.Errorf("error = %v, want a plain grammar fault rather than a rule verdict", err)
	}
	if !strings.Contains(err.Error(), "second derivation alternant") {
		t.Errorf("error = %v, want it to name the second alternant", err)
	}
}

// TestProduceSimpleContentRestrictionAssertVersusAssertion pins the two
// same-sounding element names apart, which no error would catch if they were
// conflated: the <assertion> inside the facet choice is a FACET of the
// synthesized simple type (Datatypes §4.3.13), while the trailing <assert> of
// the same content model is the CTD's own {assertions} (§3.4.2.1 clause 2).
func TestProduceSimpleContentRestrictionAssertVersusAssertion(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string">
			<xs:attribute name="unit" type="xs:string"/>
		</xs:extension></xs:simpleContent></xs:complexType>
		<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">
			<xs:assertion test="true()"/>
			<xs:attribute name="unit" type="xs:string"/>
			<xs:assert test="true()"/>
		</xs:restriction></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	synthesized := simpleContentTypeOf(t, s, "D")
	facetAssertions := 0
	for _, f := range synthesized.OwnFacets() {
		if a, ok := f.Assertions(); ok {
			facetAssertions = len(a)
		}
	}
	if facetAssertions != 1 {
		t.Errorf("synthesized type carries %d assertion facet members, want the <assertion> child's one", facetAssertions)
	}
	d := tnsComplexType(t, s, "D")
	if len(d.Assertions()) != 1 {
		t.Errorf("D has %d {assertions}, want the <assert> child's one — the <assertion> facet must not land here too", len(d.Assertions()))
	}
	if !hasAttrUse(d.AttributeUses(), "unit") {
		t.Error("D is missing the <restriction>'s own attribute use 'unit'")
	}
}
