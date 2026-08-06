package parser_test

import (
	"errors"
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

// TestProduceEmptyGroupDefinition proves a top-level <group> with no model-group
// body is rejected mgd-props-correct (§3.7.6, {model group} Required).
func TestProduceEmptyGroupDefinition(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:group name="g"/>`))
	if err == nil {
		t.Fatal("Produce accepted a <group> with no model-group body, want mgd-props-correct error")
	}
	if !strings.Contains(err.Error(), "mgd-props-correct") {
		t.Fatalf("error = %q, want it to cite mgd-props-correct", err)
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

// TestProduceNamelessTopLevelRejected pins that a top-level <complexType>,
// <group>, <attributeGroup>, <element> or <attribute> with no usable name is
// rejected the SAME way — one fault shape for all five kinds — whether or not
// its content holds anything. name is use="required" with type xs:NCName in the
// schema for schema documents (xs:topLevelComplexType, xs:namedGroup,
// xs:namedAttributeGroup, xs:topLevelElement, xs:topLevelAttribute), so an
// absent and an empty attribute are equally unusable, and no Schema
// Representation Constraint states a clause of its own for any of them (§3.4.3
// src-ct incorporates the schema for schema documents by reference; §3.6.3
// src-attribute_group is "None as such") — hence a plain grammar fault, not a
// rule verdict. The two DECLARATION kinds are charged the same way here even
// though xsd.NewElementDeclaration/xsd.NewAttributeDeclaration would later
// charge e-props-correct/a-props-correct clause 1 for the same empty {name}: at
// top level the fault belongs to the schema document's grammar and is reported
// before anything is built. Those two constructor verdicts stay pinned on the
// LOCAL paths by TestProduceAbsentNameAndEmptyRefRejected.
//
// The content-bearing rows are the regression guard, and they are why reverting
// the topLevelName call in run's switch fails this test rather than merely
// changing a message: before #206 the empty <complexType>/<group> bodies
// produced silently while the bodies holding a local <element> failed with a
// bogus e-props-correct, and before #305 a nameless <attributeGroup> was minted
// under QName{tns, ""} and carried on.
func TestProduceNamelessTopLevelRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		decl string
	}{
		{"complexType empty content", `<xs:complexType><xs:sequence/></xs:complexType>`},
		{"complexType with local element", `<xs:complexType><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`},
		{"complexType with complexContent", `<xs:complexType><xs:complexContent>` +
			`<xs:restriction base="xs:anyType"><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction>` +
			`</xs:complexContent></xs:complexType>`},
		{"complexType with empty name", `<xs:complexType name=""><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`},
		{"group empty content", `<xs:group><xs:sequence/></xs:group>`},
		{"group with local element", `<xs:group><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`},
		{"group with empty name", `<xs:group name=""><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`},
		{"attributeGroup empty content", `<xs:attributeGroup/>`},
		{"attributeGroup with attribute use", `<xs:attributeGroup>` +
			`<xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`},
		{"attributeGroup with empty name", `<xs:attributeGroup name="">` +
			`<xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`},
		{"element with type", `<xs:element type="xs:string"/>`},
		{"element with inline complexType", `<xs:element><xs:complexType><xs:sequence>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`},
		{"element with empty name", `<xs:element name="" type="xs:string"/>`},
		{"attribute with type", `<xs:attribute type="xs:string"/>`},
		{"attribute with no type", `<xs:attribute/>`},
		{"attribute with empty name", `<xs:attribute name="" type="xs:string"/>`},
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
		})
	}
}

// TestProduceNamelessBaseTypeRejectedOffDispatch reaches produceComplexType's own
// nameless-{name} guard, which since #305 no longer lies on run's document-order
// dispatch path (that fault is topLevelName's a step earlier). The remaining live
// entry is resolveBaseType's ON-DEMAND build: prescan indexes a top-level
// <complexType name=""> under QName{target, ""} — an empty local part nothing
// filters — and a base="" lexical binds to exactly that name, so building the
// derived type pulls the nameless one in before run ever dispatches on it.
//
// DOCUMENT ORDER IS LOAD-BEARING: the deriving <complexType> must come FIRST, or
// run reaches the nameless declaration itself and topLevelName raises the fault,
// which would leave the guard untested. Deleting the guard does not merely
// change this message — production walks on into the nameless type's content and
// complexTypeIdentity.scopeParent's zero-identity assertion panics, which is
// exactly why the fault is charged here, before anything is built.
func TestProduceNamelessBaseTypeRejectedOffDispatch(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:complexType name="d"><xs:complexContent>`+
		`<xs:restriction base=""><xs:sequence>`+
		`<xs:element name="b" type="xs:string"/></xs:sequence></xs:restriction>`+
		`</xs:complexContent></xs:complexType>`+
		`<xs:complexType name=""><xs:sequence>`+
		`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`))
	if err == nil {
		t.Fatalf("Produce succeeded, want a grammar fault for the base= reference's nameless <complexType>")
	}
	var xe *xsderr.Error
	if errors.As(err, &xe) {
		t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
	}
	if !strings.Contains(err.Error(), "top-level <complexType>") || !strings.Contains(err.Error(), "no usable name") {
		t.Fatalf("error = %v, want the <complexType> grammar fault reporting the unusable name", err)
	}
}
