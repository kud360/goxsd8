package parser_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// attrUseLocal returns the local name of an attribute use's {attribute
// declaration}, without resolution (a local declaration's Name or a ref's Name).
func attrUseLocal(u xsd.AttributeUse) string {
	switch d := u.AttributeDeclaration().(type) {
	case xsd.LocalAttributeDeclaration:
		return d.Declaration.Name().Local
	case xsd.AttributeDeclarationRef:
		return d.Name.Local
	}
	return ""
}

// hasAttrUse reports whether uses contains one whose declaration local name is local.
func hasAttrUse(uses []xsd.AttributeUse, local string) bool {
	for _, u := range uses {
		if attrUseLocal(u) == local {
			return true
		}
	}
	return false
}

// topComplexType fetches a top-level complex type by local name.
func topComplexType(t *testing.T, s *xsd.Schema, local string) xsd.ComplexType {
	t.Helper()
	td, ok := s.Type(xsd.QName{Local: local})
	if !ok {
		t.Fatalf("complex type %q not found", local)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %q is not a complex type (%T)", local, td)
	}
	return ct
}

// topModelGroup returns the {model group} of a top-level complex type's element
// content particle.
func topModelGroup(t *testing.T, s *xsd.Schema, local string) xsd.ModelGroup {
	t.Helper()
	ct := topComplexType(t, s, local)
	ec, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("complex type %q content is %T, want ElementContent", local, ct.ContentType())
	}
	rt, ok := ec.Particle.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("complex type %q content particle term is %T, want ResolvedTerm", local, ec.Particle.Term())
	}
	mg, ok := rt.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("complex type %q content term is %T, want ModelGroup", local, rt.Term)
	}
	return mg
}

// TestProduceAttributeGroupRefInlinesUses proves an <attributeGroup ref> inside a
// <complexType> splices in the referenced group's {attribute uses} (§3.6.2.1).
func TestProduceAttributeGroupRefInlinesUses(t *testing.T) {
	s, err := produce(t, wrap("", `
		<xs:attributeGroup name="ag"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>
		<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="ag"/></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	uses := topComplexType(t, s, "T").AttributeUses()
	if !hasAttrUse(uses, "a") {
		t.Fatalf("complex type T attribute uses = %d, want the inlined 'a' from ag", len(uses))
	}
}

// TestProduceAttributeWildcardIntersection proves the §3.6.2.2 combination folds a
// container's own <anyAttribute> (L) with a referenced group's wildcard (W) via
// INTERSECTION (cos-aw-intersect, §3.10.6.4): urn:b is common, urn:a/urn:c are not.
func TestProduceAttributeWildcardIntersection(t *testing.T) {
	s, err := produce(t, wrap("", `
		<xs:attributeGroup name="ag"><xs:anyAttribute namespace="urn:a urn:b"/></xs:attributeGroup>
		<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="ag"/><xs:anyAttribute namespace="urn:b urn:c"/></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	w, ok := topComplexType(t, s, "T").AttributeWildcard()
	if !ok {
		t.Fatalf("complex type T has no {attribute wildcard}, want the intersection")
	}
	if !w.AllowsName(xsd.QName{Space: "urn:b", Local: "z"}) {
		t.Error("intersection must admit urn:b (in both L and W)")
	}
	if w.AllowsName(xsd.QName{Space: "urn:a", Local: "z"}) {
		t.Error("intersection must reject urn:a (only in W)")
	}
	if w.AllowsName(xsd.QName{Space: "urn:c", Local: "z"}) {
		t.Error("intersection must reject urn:c (only in L)")
	}
}

// TestProduceDanglingAttributeGroupRef proves an <attributeGroup ref> to no
// top-level definition is rejected src-resolve (§3.17.6.2 clause 1.4).
func TestProduceDanglingAttributeGroupRef(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="missing"/></xs:complexType>`))
	if err == nil {
		t.Fatal("Produce accepted a dangling <attributeGroup ref>, want src-resolve error")
	}
	if !strings.Contains(err.Error(), "src-resolve") {
		t.Fatalf("error = %q, want it to cite src-resolve", err)
	}
}

// TestProduceCircularAttributeGroupLegal proves a circular <attributeGroup>
// reference chain is SPEC-LEGAL (§3.6.2.1, grounding Q3): the transitive closure
// of distinct attributes is taken, never rejected. T referencing one arm sees
// both attributes.
func TestProduceCircularAttributeGroupLegal(t *testing.T) {
	s, err := produce(t, wrap("", `
		<xs:attributeGroup name="a"><xs:attribute name="x"/><xs:attributeGroup ref="b"/></xs:attributeGroup>
		<xs:attributeGroup name="b"><xs:attribute name="y"/><xs:attributeGroup ref="a"/></xs:attributeGroup>
		<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="a"/></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce rejected a legal circular <attributeGroup> chain: %v", err)
	}
	uses := topComplexType(t, s, "T").AttributeUses()
	if !hasAttrUse(uses, "x") || !hasAttrUse(uses, "y") {
		t.Fatalf("complex type T attribute uses missing transitive members x,y (got %d)", len(uses))
	}
}

// TestProduceAttributeGroupDuplicateName proves ag-props-correct clause 2 fires on
// a genuine duplicate-name collision surfaced by the §3.6.2.1 union (two distinct
// <attribute> declarations sharing an expanded name across a reference chain).
func TestProduceAttributeGroupDuplicateName(t *testing.T) {
	_, err := produce(t, wrap("", `
		<xs:attributeGroup name="a"><xs:attribute name="dup"/><xs:attributeGroup ref="b"/></xs:attributeGroup>
		<xs:attributeGroup name="b"><xs:attribute name="dup"/></xs:attributeGroup>`))
	if err == nil {
		t.Fatal("Produce accepted two attribute uses sharing an expanded name, want ag-props-correct error")
	}
	if !strings.Contains(err.Error(), "ag-props-correct") {
		t.Fatalf("error = %q, want it to cite ag-props-correct", err)
	}
}

// TestProduceGroupRefResolves proves a <group ref> inside a complex type maps to a
// deferred ModelGroupRef (§3.7.2) that resolves at finalize against the top-level
// <group> definition.
func TestProduceGroupRefResolves(t *testing.T) {
	s, err := produce(t, wrap("", `
		<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
		<xs:complexType name="T"><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	mg := topModelGroup(t, s, "T")
	parts := mg.Particles()
	if len(parts) != 1 {
		t.Fatalf("T content model has %d particles, want 1 (the group ref)", len(parts))
	}
	ref, ok := parts[0].Term().(xsd.ModelGroupRef)
	if !ok {
		t.Fatalf("particle term is %T, want ModelGroupRef", parts[0].Term())
	}
	if ref.Name != (xsd.QName{Local: "g"}) {
		t.Fatalf("group ref name = %s, want {g}", ref.Name)
	}
}

// TestProduceDanglingGroupRef proves a <group ref> to no top-level model group
// definition is rejected src-resolve (§3.17.6.2 clause 1.5) at finalize — proving
// the ref is actually produced.
func TestProduceDanglingGroupRef(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:complexType name="T"><xs:sequence><xs:group ref="missing"/></xs:sequence></xs:complexType>`))
	if err == nil {
		t.Fatal("Produce accepted a dangling <group ref>, want src-resolve error")
	}
	if !strings.Contains(err.Error(), "src-resolve") {
		t.Fatalf("error = %q, want it to cite src-resolve", err)
	}
}

// TestProduceCircularGroupRef proves a circular <group ref> chain is rejected
// mg-props-correct clause 2 (no-circular-groups, §3.8.6.1) at finalize — the
// opposite of attribute groups, and proof the group refs are produced.
func TestProduceCircularGroupRef(t *testing.T) {
	_, err := produce(t, wrap("", `
		<xs:group name="a"><xs:sequence><xs:group ref="b"/></xs:sequence></xs:group>
		<xs:group name="b"><xs:sequence><xs:group ref="a"/></xs:sequence></xs:group>`))
	if err == nil {
		t.Fatal("Produce accepted a circular <group ref> chain, want mg-props-correct error")
	}
	if !strings.Contains(err.Error(), "mg-props-correct") {
		t.Fatalf("error = %q, want it to cite mg-props-correct", err)
	}
}

// TestProduceNamedGroupBodyRejected pins, END TO END from a schema DOCUMENT, that
// a named <group> whose body is not the one <all>/<choice>/<sequence>
// xs:namedGroup requires (xmlschema11-1.md:5187-:5216) is rejected as the
// content-model fault it is, AT the child that is not admitted — not one phase
// later at the definition element as mgd-props-correct's absent {model group}
// (the row TestProduceEmptyGroupDefinition covered before #884).
//
// The fault is plain, never a rule verdict: §3.7.3 (:2286) reads "None as such."
// in full, there is no src-mgd, and mgd-props-correct (§3.7.6, :2302) is a Schema
// Component Constraint over an already-built tableau — charging it for a document
// fault would be fabricated (STYLE E2). What binds is sd-valid (§2.4 clause 1,
// :615), the s4s-grammar class (xsderr/doc.go, #966), so every row asserts the
// error is NOT an *xsderr.Error and does NOT name mgd-props-correct.
//
// Every row DECLARES a legal <group name="G"> beside the malformed one, so no row
// can pass as a dangling reference, and each pins the LINE the fault is reported
// at (STYLE E3, carried in the message text since a plain error holds no
// xsderr.Loc): the malformed child's own line where one exists, the definition's
// where none does. Without the position assertion a guard that rejected at the
// definition for every shape would pass the table.
func TestProduceNamedGroupBodyRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). The legal <group> is
	// on line 2, the malformed one opens line 3 and its body is written on line 4,
	// so wantLine alone separates a fault reported at the child from one reported
	// at the definition.
	cases := []struct {
		name     string
		body     string
		wantName string
		wantLine int
	}{
		{
			// Also carries the name and maxOccurs xs:groupRef and xs:namedGroup
			// respectively prohibit (#876), and the content-model fault answers
			// first: the reference form is not admitted in this position under ANY
			// spelling, so its attributes are the consequence, not the mistake.
			name:     `nested <group ref= name= maxOccurs=0>`,
			body:     `<xs:group ref="tns:G" name="X" maxOccurs="0"/>`,
			wantName: "<group> at",
			wantLine: 4,
		},
		{
			name:     `nested <group ref=>`,
			body:     `<xs:group ref="tns:G"/>`,
			wantName: "<group> at",
			wantLine: 4,
		},
		{
			name:     `bare <element>`,
			body:     `<xs:element name="q" type="xs:string"/>`,
			wantName: "<element> at",
			wantLine: 4,
		},
		{
			// Nothing is written that the grammar declines, so the only thing left to
			// report is the unfilled required position, at the definition itself —
			// line 3, NOT the <annotation> on line 4.
			name:     `<annotation> only`,
			body:     `<xs:annotation/>`,
			wantName: "has no <all>, <choice> or <sequence> child",
			wantLine: 3,
		},
		{
			name:     `empty body`,
			body:     ``,
			wantName: "has no <all>, <choice> or <sequence> child",
			wantLine: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := wrap("urn:po", "\n"+
				`<xs:group name="G"><xs:sequence><xs:element name="p" type="xs:string"/></xs:sequence></xs:group>`+"\n"+
				`<xs:group name="G2">`+"\n"+tc.body+"\n"+`</xs:group>`)
			_, err := produce(t, doc)
			if err == nil {
				t.Fatalf("Produce succeeded, want a grammar fault for the malformed <group> body")
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if strings.Contains(err.Error(), "mgd-props-correct") {
				t.Fatalf("error = %v, want the document's content-model fault rather than the absent {model group} it causes", err)
			}
			if !strings.Contains(err.Error(), tc.wantName) {
				t.Fatalf("error = %v, want it to name %s as the fault", err, tc.wantName)
			}
			if !strings.Contains(err.Error(), "xs:namedGroup") {
				t.Fatalf("error = %v, want it to name the xs:namedGroup production it violates", err)
			}
			if at := fmt.Sprintf("%s:%d:", produceURI, tc.wantLine); !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at %s (E3)", err, at)
			}
		})
	}
}

// TestProduceNamedGroupBodyAccepted proves the bodies xs:namedGroup DOES admit —
// one <all>/<choice>/<sequence>, with and without a leading <annotation> — still
// produce a definition whose {model group} carries the body's compositor, so the
// sibling table's guard cannot be widened into them.
func TestProduceNamedGroupBodyAccepted(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name string
		body string
		want xsd.Compositor
	}{
		{
			name: `<sequence> alone`,
			body: `<xs:sequence><xs:element name="p" type="xs:string"/></xs:sequence>`,
			want: xsd.CompositorSequence,
		},
		{
			name: `<annotation> then <choice>`,
			body: `<xs:annotation/><xs:choice><xs:element name="p" type="xs:string"/></xs:choice>`,
			want: xsd.CompositorChoice,
		},
		{
			name: `<annotation> then <all>`,
			body: `<xs:annotation/><xs:all><xs:element name="p" type="xs:string"/></xs:all>`,
			want: xsd.CompositorAll,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:group name="g">`+tc.body+`</xs:group>`))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			d, ok := s.ModelGroup(xsd.QName{Local: "g"})
			if !ok {
				t.Fatal("schema has no model group definition g")
			}
			if got := d.ModelGroup().Compositor(); got != tc.want {
				t.Fatalf("{model group} {compositor} = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProduceGroupRefElided proves a <group ref> with minOccurs=maxOccurs=0 maps
// to no component at all (§3.7.2, xr.mgd3): the enclosing sequence gets no particle.
func TestProduceGroupRefElided(t *testing.T) {
	s, err := produce(t, wrap("", `
		<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>
		<xs:complexType name="T"><xs:sequence><xs:group ref="g" minOccurs="0" maxOccurs="0"/></xs:sequence></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if parts := topModelGroup(t, s, "T").Particles(); len(parts) != 0 {
		t.Fatalf("T content model has %d particles, want 0 (the group ref elided)", len(parts))
	}
}

// contentModelOf returns the {model group} of a top-level complex type's element
// content particle, addressing the type by its EXPANDED name (unlike
// topModelGroup, which assumes no target namespace).
func contentModelOf(t *testing.T, s *xsd.Schema, name xsd.QName) xsd.ModelGroup {
	t.Helper()
	td, ok := s.Type(name)
	if !ok {
		t.Fatalf("complex type %s not found", name)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type (%T)", name, td)
	}
	ec, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("complex type %s content is %T, want ElementContent", name, ct.ContentType())
	}
	return groupTermOf(t, ec.Particle)
}

// groupTermOf returns the Model Group a particle's {term} resolves to inline.
func groupTermOf(t *testing.T, p xsd.Particle) xsd.ModelGroup {
	t.Helper()
	rt, ok := p.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("particle term is %T, want an inline ResolvedTerm", p.Term())
	}
	mg, ok := rt.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("particle term is %T, want ModelGroup", rt.Term)
	}
	return mg
}

// elementTermOf returns the local Element Declaration a particle's {term} is.
func elementTermOf(t *testing.T, p xsd.Particle) xsd.ElementDeclaration {
	t.Helper()
	rt, ok := p.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("particle term is %T, want an inline ResolvedTerm", p.Term())
	}
	ed, ok := rt.Term.(xsd.ElementDeclaration)
	if !ok {
		t.Fatalf("particle term is %T, want ElementDeclaration", rt.Term)
	}
	return ed
}

// scopeParentOf returns a declaration's {scope}.{parent}, failing when absent.
func scopeParentOf(t *testing.T, ed xsd.ElementDeclaration) xsd.ElementScopeParent {
	t.Helper()
	parent, ok := ed.Scope().Parent()
	if !ok {
		t.Fatalf("element %s has no {scope}.{parent}, want its containing component", ed.Name())
	}
	return parent
}

// TestProduceLocalElementScopedToComplexType proves the §3.3.2.3 dcl.elt.local
// {parent} mapping for the <complexType>-ancestor case: a local <element>, at any
// nesting depth of the content model, is scoped to the ENCLOSING COMPLEX TYPE by
// expanded name — not to the nearest model group, which is not a scope boundary.
func TestProduceLocalElementScopedToComplexType(t *testing.T) {
	s, err := produce(t, wrap("urn:po", `
		<xs:complexType name="T">
			<xs:sequence>
				<xs:element name="a" type="xs:string"/>
				<xs:choice><xs:element name="b" type="xs:string"/></xs:choice>
			</xs:sequence>
		</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	want := xsd.ComplexTypeScopeParent{Name: xsd.QName{Space: "urn:po", Local: "T"}}
	parts := contentModelOf(t, s, xsd.QName{Space: "urn:po", Local: "T"}).Particles()
	if len(parts) != 2 {
		t.Fatalf("T content model has %d particles, want 2", len(parts))
	}

	direct := elementTermOf(t, parts[0])
	if direct.ScopeVariety() != xsd.ScopeLocal {
		t.Fatalf("element a scope = %s, want local", direct.ScopeVariety())
	}
	if got := scopeParentOf(t, direct); got != want {
		t.Fatalf("element a {scope}.{parent} = %#v, want %#v", got, want)
	}

	// The nested <choice> must not shift the parent: only <complexType> and a
	// named <group> are scope-determining ancestors.
	deep := elementTermOf(t, groupTermOf(t, parts[1]).Particles()[0])
	if got := scopeParentOf(t, deep); got != want {
		t.Fatalf("element b {scope}.{parent} = %#v, want %#v", got, want)
	}
}

// TestProduceTopLevelElementHasNoScopeParent proves the §3.3.2.2 dcl.elt.global
// half: a top-level <element> carries {parent} ·absent·.
func TestProduceTopLevelElementHasNoScopeParent(t *testing.T) {
	s, err := produce(t, wrap("urn:po", `<xs:element name="root" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Space: "urn:po", Local: "root"})
	if !ok {
		t.Fatal("element root not found")
	}
	if parent, ok := ed.Scope().Parent(); ok {
		t.Fatalf("top-level element root has {scope}.{parent} = %#v, want absent", parent)
	}
}

// TestProduceNamelessTopLevelRejected pins that a top-level <simpleType>,
// <complexType>, <group>, <attributeGroup>, <element> or <attribute> with no
// usable name is rejected the SAME way — one fault shape for all six kinds —
// whether or not its content holds anything. name is use="required" with type
// xs:NCName in the schema for schema documents (xs:topLevelSimpleType,
// xs:topLevelComplexType, xs:namedGroup, xs:namedAttributeGroup,
// xs:topLevelElement, xs:topLevelAttribute), so an absent and an empty attribute
// are equally unusable, and no Schema Representation Constraint states a clause
// of its own for any of them (§3.4.3 src-ct and §3.16.3 src-simple-type
// incorporate the schema for schema documents by reference; §3.6.3
// src-attribute_group is "None as such") — hence a plain grammar fault, not a
// rule verdict. Each row also pins WHICH of the six its own diagnostic names,
// which is topLevelGrammar's whole mapping: an s4s-grammar rejection carries no
// Rule, so the production is the only citation it has (#975).
//
// The two DECLARATION kinds are charged the same way here even
// though xsd.NewElementDeclaration/xsd.NewAttributeDeclaration would later
// charge e-props-correct/a-props-correct clause 1 for the same empty {name}: at
// top level the fault belongs to the schema document's grammar and is reported
// before anything is built. Those two constructor verdicts stay pinned on the
// LOCAL paths by TestProduceAbsentNameAndEmptyRefRejected.
//
// <simpleType> has NO constructor verdict behind it to fall back on and never
// will: §3.16.1 types Simple Type Definition's {name} "Optional" so that
// anonymous simple types stay constructible, which is why its rows here are the
// only enforcement of the top-level form's required name (#523). That the
// rejection did not also outlaw the anonymous form is pinned by
// TestProduceAnonymousInlineBaseRestriction,
// TestProduceLocalElementInlineSimpleType and
// TestProduceLocalAttributeInlineSimpleType, each of which builds a nameless
// <simpleType> that must still produce.
//
// The content-bearing rows are the regression guard, and they are why reverting
// the topLevelName call in run's switch fails this test rather than merely
// changing a message: before #206 the empty <complexType>/<group> bodies
// produced silently while the bodies holding a local <element> failed with a
// bogus e-props-correct, before #305 a nameless <attributeGroup> was minted
// under QName{tns, ""} and carried on, and before #523 a nameless <simpleType>
// was minted the same way.
func TestProduceNamelessTopLevelRejected(t *testing.T) {
	for _, tc := range []struct {
		name        string
		decl        string
		wantGrammar string // the Appendix A production the message must name (#975)
	}{
		// Each <simpleType> row carries a WELL-FORMED body, so the "no usable
		// name" assertion cannot pass on a src-simple-type body fault standing in
		// for the name fault. The facet-bearing row is the suite shape
		// (MS-SimpleType2006-07-15/stA015).
		{"simpleType absent name", `<xs:simpleType>` +
			`<xs:restriction base="xs:string"/></xs:simpleType>`, "xs:topLevelSimpleType"},
		{"simpleType with a facet", `<xs:simpleType><xs:restriction base="xs:string">` +
			`<xs:length value="3"/></xs:restriction></xs:simpleType>`, "xs:topLevelSimpleType"},
		{"simpleType with empty name", `<xs:simpleType name="">` +
			`<xs:restriction base="xs:string"/></xs:simpleType>`, "xs:topLevelSimpleType"},
		{"complexType empty content", `<xs:complexType><xs:sequence/></xs:complexType>`,
			"xs:topLevelComplexType"},
		{"complexType with local element", `<xs:complexType><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			"xs:topLevelComplexType"},
		{"complexType with complexContent", `<xs:complexType><xs:complexContent>` +
			`<xs:restriction base="xs:anyType"><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>` +
			`</xs:complexContent></xs:complexType>`, "xs:topLevelComplexType"},
		{"complexType with empty name", `<xs:complexType name=""><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
			"xs:topLevelComplexType"},
		{"group empty content", `<xs:group><xs:sequence/></xs:group>`, "xs:namedGroup"},
		{"group with local element", `<xs:group><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`, "xs:namedGroup"},
		{"group with empty name", `<xs:group name=""><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`, "xs:namedGroup"},
		{"attributeGroup empty content", `<xs:attributeGroup/>`, "xs:namedAttributeGroup"},
		{"attributeGroup with attribute use", `<xs:attributeGroup>` +
			`<xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`, "xs:namedAttributeGroup"},
		{"attributeGroup with empty name", `<xs:attributeGroup name="">` +
			`<xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`, "xs:namedAttributeGroup"},
		{"element with type", `<xs:element type="xs:string"/>`, "xs:topLevelElement"},
		{"element with inline complexType", `<xs:element><xs:complexType><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`,
			"xs:topLevelElement"},
		{"element with empty name", `<xs:element name="" type="xs:string"/>`, "xs:topLevelElement"},
		{"attribute with type", `<xs:attribute type="xs:string"/>`, "xs:topLevelAttribute"},
		{"attribute with no type", `<xs:attribute/>`, "xs:topLevelAttribute"},
		{"attribute with empty name", `<xs:attribute name="" type="xs:string"/>`, "xs:topLevelAttribute"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.decl))
			if err == nil {
				t.Fatalf("Produce succeeded, want a grammar fault for the missing name")
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if !strings.Contains(err.Error(), "no usable name") {
				t.Fatalf("error = %v, want it to report the unusable name", err)
			}
			if !strings.Contains(err.Error(), tc.wantGrammar) {
				t.Fatalf("error = %v, want it to name %s, the production whose required name is missing", err, tc.wantGrammar)
			}
		})
	}
}

// TestProduceEmptyBaseClosesTheOnDemandNamelessBuild pins the document that used
// to be the last schema-writable way into produceComplexType's nameless-{name}
// guard, and pins that #343 closed it one step earlier.
//
// The shape: prescan indexes a top-level <complexType name=""> under
// QName{target, ""} — an empty local part nothing filters — and a base=""
// lexical used to bind to exactly that name, so resolveBaseType's ON-DEMAND
// build pulled the nameless type in before run ever dispatched on it (run's own
// dispatch path raises the fault from topLevelName a step earlier, #305).
// DOCUMENT ORDER IS STILL LOAD-BEARING for that: the deriving <complexType> comes
// FIRST, or run reaches the nameless declaration itself and topLevelName answers.
//
// Since #343 the base="" lexical never binds to anything: bindQName rejects an
// empty QName local part at the attribute, charging cvc-datatype-valid (Datatypes
// §4.1.4) against the xs:QName the schema for schema documents declares for
// base. So the verdict this document earns is that lexical one, positioned on the
// <restriction>, and the nameless type is never built — which is also why the
// panic the guard's doc describes (production walking into the nameless type
// until complexTypeIdentity.scopeParent's zero-identity assertion fires) stays
// out of reach. The guard itself is deliberately kept as an in-package backstop
// and is NOT asserted here any more; nothing a schema author can write reaches
// it.
func TestProduceEmptyBaseClosesTheOnDemandNamelessBuild(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:complexType name="d"><xs:complexContent>`+
		`<xs:restriction base=""><xs:sequence>`+
		`<xs:element name="b" type="xs:string"/></xs:sequence></xs:restriction>`+
		`</xs:complexContent></xs:complexType>`+
		`<xs:complexType name=""><xs:sequence>`+
		`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`))
	assertRule(t, err, "cvc-datatype-valid")
	loc, ok := xsderr.LocOf(err)
	if !ok || loc.URI != produceURI || loc.Line != 1 || loc.Col == 0 {
		t.Fatalf("position = %v (found %t), want the <restriction> at %s:1 with a column", loc, ok, produceURI)
	}
}
