package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
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
