package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// typeTableOf produces doc and returns the {type table} of its top-level
// element named name, reporting whether the property is present.
func typeTableOf(t *testing.T, doc string, name xsd.QName) (xsd.TypeTable, bool) {
	t.Helper()
	s, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(name)
	if !ok {
		t.Fatalf("element %s not found", name)
	}
	return ed.TypeTable()
}

// altTypeNames returns the {type definition} name of every {alternatives}
// member, in order, so a document-order assertion reads as one comparison. An
// inline-armed member has no name and reads as "«inline»".
func altTypeNames(alts []xsd.TypeAlternative) []string {
	names := make([]string, 0, len(alts))
	for _, a := range alts {
		names = append(names, altTypeName(a))
	}
	return names
}

// altTypeName is the local part of the type a Type Alternative NAMES, or
// "«inline»" for §3.12.2 declare-ta's inline arm, which names nothing.
func altTypeName(a xsd.TypeAlternative) string {
	ref, named := a.TypeDefinition().(xsd.TypeDefinitionRef)
	if !named {
		return "«inline»"
	}
	return ref.Name.Local
}

// altInlineComplexType unwraps the ANONYMOUS complex type an inline-armed Type
// Alternative owns, together with the identity its {context} carries.
func altInlineComplexType(t *testing.T, a xsd.TypeAlternative) (xsd.ComplexType, xsd.ComponentID) {
	t.Helper()
	inline, ok := a.TypeDefinition().(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("{type definition} = %#v, want the InlineTypeDefinition arm", a.TypeDefinition())
	}
	ct, ok := inline.Definition.(xsd.ComplexType)
	if !ok {
		t.Fatalf("InlineTypeDefinition wraps %T, want a ComplexType", inline.Definition)
	}
	context, ok := ct.Context()
	if !ok {
		t.Fatal("the alternative's anonymous complex type has an ·absent· {context}")
	}
	return ct, context.ID()
}

const typeTableTypes = `<xs:complexType name="B"><xs:sequence/></xs:complexType>
	<xs:complexType name="T"><xs:complexContent><xs:extension base="B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>
	<xs:complexType name="U"><xs:complexContent><xs:extension base="B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>
	<xs:complexType name="V"><xs:complexContent><xs:extension base="B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`

// A trailing <alternative> with no test attribute maps to {default type
// definition} and is NOT an {alternatives} member (§3.3.2.1, case 1).
func TestProduceTypeTableTrailingUntestedAlternativeIsTheDefault(t *testing.T) {
	tt, present := typeTableOf(t, wrap("", typeTableTypes+`
	<xs:element name="e" type="B">
	  <xs:alternative test="@k='t'" type="T"/>
	  <xs:alternative test="@k='u'" type="U"/>
	  <xs:alternative type="V"/>
	</xs:element>`), xsd.QName{Local: "e"})
	if !present {
		t.Fatal("{type table} is ·absent·, want present")
	}
	alts := tt.Alternatives()
	if got := altTypeNames(alts); len(got) != 2 || got[0] != "T" || got[1] != "U" {
		t.Fatalf("{alternatives} = %v, want [T U] in document order", got)
	}
	test, hasTest := alts[0].Test()
	if !hasTest || test.Expression() != "@k='t'" {
		t.Errorf("{alternatives}[0].{test} = %q/%t, want %q present", test.Expression(), hasTest, "@k='t'")
	}
	dflt := tt.DefaultTypeDefinition()
	if _, hasTest := dflt.Test(); hasTest {
		t.Error("{default type definition}.{test} is present, want ·absent·")
	}
	if got := altTypeName(dflt); got != "V" {
		t.Errorf("{default type definition}.{type definition} = %s, want V", got)
	}
}

// With every <alternative> tested, {default type definition} is SYNTHESIZED
// from the declaration's own {type definition} (§3.3.2.1, case 2).
func TestProduceTypeTableSynthesizesDefaultFromDeclaredType(t *testing.T) {
	tt, present := typeTableOf(t, wrap("", typeTableTypes+`
	<xs:element name="e" type="B">
	  <xs:alternative test="@k='t'" type="T"/>
	</xs:element>`), xsd.QName{Local: "e"})
	if !present {
		t.Fatal("{type table} is ·absent·, want present")
	}
	if got := altTypeNames(tt.Alternatives()); len(got) != 1 || got[0] != "T" {
		t.Fatalf("{alternatives} = %v, want [T]", got)
	}
	dflt := tt.DefaultTypeDefinition()
	if _, hasTest := dflt.Test(); hasTest {
		t.Error("{default type definition}.{test} is present, want ·absent·")
	}
	if got := altTypeName(dflt); got != "B" {
		t.Errorf("{default type definition}.{type definition} = %s, want the declaration's own B", got)
	}
}

// An <element> with no <alternative> child keeps {type table} ·absent·.
func TestProduceTypeTableAbsentWithoutAlternatives(t *testing.T) {
	if _, present := typeTableOf(t, wrap("", typeTableTypes+`
	<xs:element name="e" type="B"/>`), xsd.QName{Local: "e"}); present {
		t.Error("{type table} is present, want ·absent· for an <element> with no <alternative>")
	}
}

// §3.12.2 declare-ta's INLINE arm maps to an InlineTypeDefinition owning the
// anonymous type, whose {context} is the enclosing element declaration —
// §3.4.2.1 dcl.ctd.common climbs to the nearest ancestor <element> whatever the
// child position, so it is the SAME identity the element's own inline type
// carries.
func TestProduceTypeTableInlineAlternativeArm(t *testing.T) {
	tt, present := typeTableOf(t, wrap("", typeTableTypes+`
	<xs:element name="e">
	  <xs:complexType><xs:sequence/></xs:complexType>
	  <xs:alternative test="@k='t'" type="T"/>
	  <xs:alternative test="@k='i'"><xs:complexType><xs:sequence/></xs:complexType></xs:alternative>
	</xs:element>`), xsd.QName{Local: "e"})
	if !present {
		t.Fatal("{type table} is ·absent·, want present where one <alternative> takes the inline arm")
	}
	alts := tt.Alternatives()
	if got := altTypeNames(alts); len(got) != 2 || got[0] != "T" || got[1] != "«inline»" {
		t.Fatalf("{alternatives} = %v, want [T «inline»] in document order", got)
	}
	_, altContext := altInlineComplexType(t, alts[1])

	// The synthesized {default type definition} carries the declaration's own
	// inline type verbatim (§3.3.2.1 case 2), which is the shape #800 could not
	// express at all.
	_, ownContext := altInlineComplexType(t, tt.DefaultTypeDefinition())
	if altContext != ownContext {
		t.Error("the alternative's anonymous type and the element's own do not share one {context}, but §3.4.2.1 gives both the same element declaration")
	}
}

// An <alternative> taking the inline SIMPLE-type arm owns an anonymous simple
// type on the same footing.
func TestProduceTypeTableInlineSimpleAlternativeArm(t *testing.T) {
	tt, present := typeTableOf(t, wrap("", typeTableTypes+`
	<xs:element name="e" type="B">
	  <xs:alternative test="@k='s'"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:alternative>
	</xs:element>`), xsd.QName{Local: "e"})
	if !present {
		t.Fatal("{type table} is ·absent·, want present for an inline <simpleType> arm")
	}
	inline, ok := tt.Alternatives()[0].TypeDefinition().(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("{alternatives}[0].{type definition} = %#v, want the InlineTypeDefinition arm", tt.Alternatives()[0].TypeDefinition())
	}
	if _, ok := inline.Definition.(*xsd.SimpleType); !ok {
		t.Errorf("InlineTypeDefinition wraps %T, want a *SimpleType", inline.Definition)
	}
}

// Two anonymous complex types under ONE <element> — the element's own and an
// <alternative>'s — share a {context} but must NOT share the container token
// their nested local declarations report as {scope}.{parent}: one mint per
// OWNERSHIP EDGE, or the two structurally distinct containers become
// indistinguishable (#822).
func TestProduceTypeTableInlineAlternativeGetsItsOwnContainerToken(t *testing.T) {
	tt, present := typeTableOf(t, wrap("", typeTableTypes+`
	<xs:element name="e">
	  <xs:complexType><xs:sequence><xs:element name="own" type="B"/></xs:sequence></xs:complexType>
	  <xs:alternative test="@k='i'"><xs:complexType><xs:sequence><xs:element name="alt" type="B"/></xs:sequence></xs:complexType></xs:alternative>
	</xs:element>`), xsd.QName{Local: "e"})
	if !present {
		t.Fatal("{type table} is ·absent·, want present")
	}
	altType, _ := altInlineComplexType(t, tt.Alternatives()[0])
	ownType, _ := altInlineComplexType(t, tt.DefaultTypeDefinition())
	altOwner := nestedScopeOwner(t, altType, "alt")
	ownOwner := nestedScopeOwner(t, ownType, "own")
	if altOwner == ownOwner {
		t.Error("the alternative's anonymous type and the element's own report the SAME {scope}.{parent} token to their nested declarations, but they are different containers")
	}
}

// nestedScopeOwner returns the AnonymousComplexTypeScopeParent.Owner token the
// local element declaration named local, nested in ct's content model, reports.
func nestedScopeOwner(t *testing.T, ct xsd.ComplexType, local string) xsd.ComponentID {
	t.Helper()
	content, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("the anonymous type holding %q has no element-only content", local)
	}
	group, ok := content.Particle.Term().(xsd.ResolvedTerm).Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("the anonymous type holding %q has no model group", local)
	}
	for _, part := range group.Particles() {
		term, ok := part.Term().(xsd.ResolvedTerm)
		if !ok {
			continue
		}
		decl, ok := term.Term.(xsd.ElementDeclaration)
		if !ok || decl.Name().Local != local {
			continue
		}
		parent, ok := decl.Scope().Parent()
		if !ok {
			t.Fatalf("local element %q has no {scope}.{parent}", local)
		}
		anon, ok := parent.(xsd.AnonymousComplexTypeScopeParent)
		if !ok {
			t.Fatalf("local element %q reports %T, want an AnonymousComplexTypeScopeParent", local, parent)
		}
		return anon.Owner
	}
	t.Fatalf("no local element %q in the anonymous type", local)
	return xsd.ComponentID{}
}

// ·xs:error· (§3.16.7.3) is present in every schema by definition (builtin.Seed
// prepends it), so an <alternative> naming it maps like any other named type and
// finalize resolves the reference instead of charging src-resolve clause 1.1.
func TestProduceTypeTableForXSError(t *testing.T) {
	tt, present := typeTableOf(t, wrap("", typeTableTypes+`
	<xs:element name="e" type="B">
	  <xs:alternative test="@k='t'" type="T"/>
	  <xs:alternative type="xs:error"/>
	</xs:element>`), xsd.QName{Local: "e"})
	if !present {
		t.Fatal("{type table} is ·absent·, want present where an <alternative> names xs:error")
	}
	dflt := tt.DefaultTypeDefinition().TypeDefinition()
	if want := (xsd.TypeDefinitionOrRef)(xsd.TypeDefinitionRef{Name: xsd.QName{Space: xsd.XMLSchemaNS, Local: "error"}}); dflt != want {
		t.Errorf("{default type definition}.{type definition} = %v, want %v", dflt, want)
	}
}

// The LOCAL <element> path maps the table on the same terms as the global one.
func TestProduceTypeTableOnLocalElement(t *testing.T) {
	s, err := produce(t, wrap("", typeTableTypes+`
	<xs:complexType name="ct">
	  <xs:sequence>
	    <xs:element name="local" type="B">
	      <xs:alternative test="@k='t'" type="T"/>
	    </xs:element>
	  </xs:sequence>
	</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	tt, present := localElementTypeTable(t, s, xsd.QName{Local: "ct"}, "local")
	if !present {
		t.Fatal("local declaration's {type table} is ·absent·, want present")
	}
	if got := altTypeNames(tt.Alternatives()); len(got) != 1 || got[0] != "T" {
		t.Fatalf("{alternatives} = %v, want [T]", got)
	}
	if got := altTypeName(tt.DefaultTypeDefinition()); got != "B" {
		t.Errorf("{default type definition}.{type definition} = %s, want B", got)
	}
}

// localElementTypeTable returns the {type table} of the local element
// declaration named local in the content model of the complex type typeName.
func localElementTypeTable(t *testing.T, s *xsd.Schema, typeName xsd.QName, local string) (xsd.TypeTable, bool) {
	t.Helper()
	def, ok := s.Type(typeName)
	if !ok {
		t.Fatalf("type %s not found", typeName)
	}
	ct, ok := def.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type", typeName)
	}
	content, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("type %s has no element-only content", typeName)
	}
	group, ok := content.Particle.Term().(xsd.ResolvedTerm).Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("type %s's {content type} particle is not a model group", typeName)
	}
	for _, part := range group.Particles() {
		term, ok := part.Term().(xsd.ResolvedTerm)
		if !ok {
			continue
		}
		decl, ok := term.Term.(xsd.ElementDeclaration)
		if !ok || decl.Name().Local != local {
			continue
		}
		return decl.TypeTable()
	}
	t.Fatalf("no local element %q in type %s", local, typeName)
	return xsd.TypeTable{}, false
}

// {test} is the §3.13.2 XPath Expression property record, so the <alternative>
// host element's own xpathDefaultNamespace and namespace bindings govern it —
// the same construction an <assert> gets, not a bespoke one.
func TestProduceTypeTableTestIsAnXPathExpressionRecord(t *testing.T) {
	tt, present := typeTableOf(t, wrap("urn:t", `
	<xs:complexType name="B"><xs:sequence/></xs:complexType>
	<xs:complexType name="T"><xs:complexContent><xs:extension base="tns:B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>
	<xs:element name="e" type="tns:B">
	  <xs:alternative test="@k='t'" type="tns:T" xpathDefaultNamespace="##targetNamespace" xmlns:p="urn:p"/>
	</xs:element>`), xsd.QName{Space: "urn:t", Local: "e"})
	if !present {
		t.Fatal("{type table} is ·absent·, want present")
	}
	test, hasTest := tt.Alternatives()[0].Test()
	if !hasTest {
		t.Fatal("{alternatives}[0].{test} is ·absent·, want present")
	}
	ns, nsPresent := test.DefaultNamespace()
	if !nsPresent || ns != "urn:t" {
		t.Errorf("{default namespace} = %q/%t, want urn:t from ##targetNamespace", ns, nsPresent)
	}
	var bound bool
	for _, b := range test.NamespaceBindings() {
		if b.Prefix() == "p" && b.Namespace() == "urn:p" {
			bound = true
		}
	}
	if !bound {
		t.Errorf("{namespace bindings} = %v, want the host element's p→urn:p binding", test.NamespaceBindings())
	}
}

// src-ta (§3.12.3) counts the two INLINE forms as present, so an <alternative>
// with exactly one type child is fine and only zero or two-or-more is charged.
func TestProduceSrcTA(t *testing.T) {
	cases := []struct {
		name       string
		alternates string
		wantRule   bool
	}{
		{name: "type attribute alone", alternates: `<xs:alternative test="@k='t'" type="T"/>`},
		{name: "complexType child alone", alternates: `<xs:alternative test="@k='t'"><xs:complexType><xs:sequence/></xs:complexType></xs:alternative>`},
		{name: "simpleType child alone", alternates: `<xs:alternative test="@k='t'"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:alternative>`},
		{name: "no form at all", alternates: `<xs:alternative test="@k='t'"/>`, wantRule: true},
		{name: "type attribute and complexType child", alternates: `<xs:alternative test="@k='t'" type="T"><xs:complexType><xs:sequence/></xs:complexType></xs:alternative>`, wantRule: true},
		{name: "type attribute and simpleType child", alternates: `<xs:alternative test="@k='t'" type="T"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:alternative>`, wantRule: true},
		{name: "both inline children", alternates: `<xs:alternative test="@k='t'"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType><xs:complexType><xs:sequence/></xs:complexType></xs:alternative>`, wantRule: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", typeTableTypes+`
			<xs:element name="e" type="B">`+tc.alternates+`</xs:element>`))
			if !tc.wantRule {
				if err != nil {
					t.Fatalf("Produce: %v, want no src-ta charge", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Produce succeeded, want a src-ta charge")
			}
			assertRule(t, err, "src-ta")
		})
	}
}

// A dangling alternative type name is charged src-resolve at finalize, which is
// only reachable once the table is actually built.
func TestProduceTypeTableAlternativeTypeResolves(t *testing.T) {
	_, err := produce(t, wrap("", typeTableTypes+`
	<xs:element name="e" type="B">
	  <xs:alternative test="@k='t'" type="Missing"/>
	</xs:element>`))
	if err == nil {
		t.Fatal("Produce succeeded, want src-resolve for an alternative naming no type")
	}
	assertRule(t, err, "src-resolve")
}
