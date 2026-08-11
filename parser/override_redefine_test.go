package parser_test

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// This file is the seam where §F.2 clause 1 meets src-expredef: an <override>
// substituting for a <redefine> child that references ITSELF. Each test pins one
// of the four element types a <redefine> admits, because each reaches
// src-expredef's self-reference resolution through a different site — base= for
// the two type kinds, ref= for the two group kinds.
//
// Every one of them is a false-circularity trap: if the substitute is not
// recognized as the redefining declaration it stands in for, its self-reference
// resolves to the visible redefinition and the assembly is rejected as a
// circular derivation or a circular group.

// TestParseOverrideRedefinedSimpleTypeResolvesToOriginal is src-expredef clause
// 1.1 under substitution: main.xsd's <override> replaces mid.xsd's redefining
// <simpleType>, whose <restriction> names its own expanded name, and that base
// must still be the ORIGINAL in lib.xsd (S2) rather than the visible
// redefinition.
func TestParseOverrideRedefinedSimpleTypeResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code">`+
			`<xs:maxLength value="3"/></xs:restriction></xs:simpleType>`+
			`</xs:override>`),
		"mid.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code">`+
			`<xs:maxLength value="2"/></xs:restriction></xs:simpleType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"><xs:maxLength value="8"/></xs:restriction>`+
			`</xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	code := mustSimpleType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
	if got := facetValue(t, code, xsd.FacetMaxLength); got != "3" {
		t.Fatalf("{urn:a}code own maxLength = %q, want the OVERRIDING declaration's 3 (§F.2 clause 1)", got)
	}
	base := mustBase(t, s, code)
	if base == nil {
		t.Fatal("the substituted redefinition has no {base type definition}")
	}
	if got := base.Name(); got != (xsd.QName{}) {
		t.Fatalf("hidden original's {name} = %s, want ·absent· (src-expredef clause 1.1)", got)
	}
	if got := facetValue(t, base, xsd.FacetMaxLength); got != "8" {
		t.Fatalf("hidden original maxLength = %q, want lib.xsd's 8 — the substitute's self-reference must resolve into S2", got)
	}
}

// TestParseOverrideRedefinedComplexTypeResolvesToOriginal is the same seam for
// src-expredef clause 1.2's pairing: the substituting <complexType>'s
// <extension> names itself, and the {base type definition} must be the anonymous
// original built from lib.xsd's declaration.
func TestParseOverrideRedefinedComplexTypeResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:complexType name="ct"><xs:complexContent><xs:extension base="tns:ct">`+
			`<xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence>`+
			`</xs:extension></xs:complexContent></xs:complexType>`+
			`</xs:override>`),
		"mid.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:complexType name="ct"><xs:complexContent><xs:extension base="tns:ct">`+
			`<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>`+
			`</xs:extension></xs:complexContent></xs:complexType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:complexType name="ct">`+
			`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ct := mustComplexType(t, s, xsd.QName{Space: "urn:a", Local: "ct"})
	base := mustOwnedBase(t, ct)
	if got := elementNamesOf(t, base); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("the clause-1.1 original declares %v, want lib.xsd's [a]", got)
	}
	// The substitute's own content, extended over the original's: mid.xsd's
	// redefining declaration is gone entirely (§F.2 clause 1), so "b" appears
	// nowhere.
	if got := elementNamesOf(t, ct); !slices.Equal(got, []string{"a", "c"}) {
		t.Fatalf("the substituted redefinition declares %v, want [a c]", got)
	}
}

// TestParseOverrideRedefinedGroupResolvesToOriginal is src-expredef clause 2 for
// a model group definition under substitution: the substituting <group>'s
// <group ref> names its own expanded name and must splice in lib.xsd's
// particles, not read as a circular <group ref> graph (mg-props-correct clause
// 2).
func TestParseOverrideRedefinedGroupResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`<xs:element name="c" type="xs:string"/>`+
			`</xs:sequence></xs:group>`+
			`</xs:override>`),
		"mid.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`<xs:element name="b" type="xs:string"/>`+
			`</xs:sequence></xs:group>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mgd := mustModelGroup(t, s, xsd.QName{Space: "urn:a", Local: "g"})
	particles := mgd.ModelGroup().Particles()
	if len(particles) != 2 {
		t.Fatalf("redefined {urn:a}g has %d particles, want 2", len(particles))
	}
	resolved, ok := particles[0].Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("self-reference {term} = %#v, want the original's resolved inline model group", particles[0].Term())
	}
	inner, ok := resolved.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("self-reference {term} = %#v, want an xsd.ModelGroup", resolved.Term)
	}
	if got := len(inner.Particles()); got != 1 {
		t.Fatalf("inlined original has %d particles, want lib.xsd's 1", got)
	}
}

// TestParseOverrideRedefinedAttributeGroupResolvesToOriginal is src-expredef
// clause 2's attributeGroup half under substitution: the substituting
// <attributeGroup>'s self-reference splices in lib.xsd's {attribute uses}.
func TestParseOverrideRedefinedAttributeGroupResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:attributeGroup name="ag">`+
			`<xs:attributeGroup ref="tns:ag"/>`+
			`<xs:attribute name="c" type="xs:string"/>`+
			`</xs:attributeGroup>`+
			`</xs:override>`),
		"mid.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:attributeGroup name="ag">`+
			`<xs:attributeGroup ref="tns:ag"/>`+
			`<xs:attribute name="b" type="xs:string"/>`+
			`</xs:attributeGroup>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:attributeGroup name="ag">`+
			`<xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ag := mustAttributeGroup(t, s, xsd.QName{Space: "urn:a", Local: "ag"})
	if got := len(ag.AttributeUses()); got != 2 {
		t.Fatalf("redefined {urn:a}ag has %d attribute uses, want 2 (lib.xsd's a plus the substitute's c)", got)
	}
}

// TestParseNestedOverrideRedefinedGroupResolvesToOriginal is the same resolution
// two <override> levels down, where the substitute reaching the <redefine> child
// is the one §F.2 clause 4.1 kept when the inner override was composed under the
// outer. The membership index is built from the COMPOSED override in force over
// the redefining document, so nesting depth changes nothing (PRINCIPLES 16).
func TestParseNestedOverrideRedefinedGroupResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="outer.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`<xs:element name="d" type="xs:string"/>`+
			`</xs:sequence></xs:group>`+
			`</xs:override>`),
		"outer.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`<xs:element name="c" type="xs:string"/>`+
			`</xs:sequence></xs:group>`+
			`</xs:override>`),
		"mid.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`<xs:element name="b" type="xs:string"/>`+
			`</xs:sequence></xs:group>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mgd := mustModelGroup(t, s, xsd.QName{Space: "urn:a", Local: "g"})
	particles := mgd.ModelGroup().Particles()
	if len(particles) != 2 {
		t.Fatalf("redefined {urn:a}g has %d particles, want 2", len(particles))
	}
	// The OUTERMOST override wins under §F.2 clause 4.1, so the second particle
	// is main.xsd's "d".
	second, ok := particles[1].Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("second particle {term} = %#v, want a resolved element declaration", particles[1].Term())
	}
	ed, ok := second.Term.(xsd.ElementDeclaration)
	if !ok {
		t.Fatalf("second particle {term} = %#v, want an xsd.ElementDeclaration", second.Term)
	}
	if got := ed.Name().Local; got != "d" {
		t.Fatalf("second particle declares %q, want main.xsd's \"d\" (§F.2 clause 4.1)", got)
	}
}
