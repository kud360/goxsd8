package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin §3.3.2.1 dcl.elt.common clause 1 for the inline anonymous
// <complexType> child of an <element> (#340), on BOTH the global path
// (§3.3.2.2 dcl.elt.global) and the local one (§3.3.2.3 dcl.elt.local), plus the
// §3.4.2.1 dcl.ctd.common {context} identity that scopes the anonymous type's
// own nested local declarations.

// inlineComplexType asserts that a {type definition} slot is the
// InlineTypeDefinition arm owning an anonymous ComplexType, and returns it.
func inlineComplexType(t *testing.T, ref xsd.TypeDefinitionOrRef) xsd.ComplexType {
	t.Helper()
	inline, ok := ref.(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("{type definition} = %#v, want an xsd.InlineTypeDefinition", ref)
	}
	ct, ok := inline.Definition.(xsd.ComplexType)
	if !ok {
		t.Fatalf("InlineTypeDefinition wraps %T, want an xsd.ComplexType", inline.Definition)
	}
	if ct.Name() != (xsd.QName{}) {
		t.Fatalf("inline complex type has {name} %s, want the absent QName (§3.4.1 makes {name}/{context} a strict XOR)", ct.Name())
	}
	return ct
}

// ownerID reads the {context} identity of an anonymous complex type: the
// declaration §3.4.2.1 dcl.ctd.common makes its {context}. Compared with == and
// never reflect.DeepEqual, which is identity-blind on a ComponentID.
func ownerID(t *testing.T, ct xsd.ComplexType) xsd.ComponentID {
	t.Helper()
	context, ok := ct.Context()
	if !ok {
		t.Fatal("anonymous complex type has no {context}, which §3.4.1 makes Required when {name} is absent")
	}
	if _, isED := context.(xsd.ElementDeclarationContext); !isED {
		t.Fatalf("{context} = %T, want an xsd.ElementDeclarationContext (§3.4.2.1 has one case and it yields an Element Declaration)", context)
	}
	return context.ID()
}

// elementOf looks a global declaration up by local name in the empty namespace.
func elementOf(t *testing.T, s *xsd.Schema, local string) xsd.ElementDeclaration {
	t.Helper()
	ed, ok := s.Element(xsd.QName{Local: local})
	if !ok {
		t.Fatalf("element %s not found", local)
	}
	return ed
}

// particleTerms returns the element declarations a complex type's {content
// type} particle tree holds directly under its top model group.
func particleTerms(t *testing.T, ct xsd.ComplexType) []xsd.ElementDeclaration {
	t.Helper()
	content, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("{content type} = %T, want ElementContent", ct.ContentType())
	}
	rt, ok := content.Particle.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("top particle {term} = %T, want an inline ResolvedTerm", content.Particle.Term())
	}
	group, ok := rt.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("top {term} = %T, want a ModelGroup", rt.Term)
	}
	var decls []xsd.ElementDeclaration
	for _, p := range group.Particles() {
		inner, ok := p.Term().(xsd.ResolvedTerm)
		if !ok {
			continue
		}
		if ed, ok := inner.Term.(xsd.ElementDeclaration); ok {
			decls = append(decls, ed)
		}
	}
	return decls
}

// TestProduceGlobalElementInlineComplexType pins tier 1 on the GLOBAL path: the
// anonymous type is built, wired as {type definition}, and — because it has no
// {name} — never enters the schema's {type definitions} (§3.17.2 scopes that
// property to the <complexType> children OF <schema>).
func TestProduceGlobalElementInlineComplexType(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:element name="doc">
		<xs:complexType><xs:sequence/><xs:attribute name="a" type="xs:string"/></xs:complexType>
	</xs:element>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed := elementOf(t, s, "doc")
	if ed.ScopeVariety() != xsd.ScopeGlobal {
		t.Fatalf("scope = %s, want global (§3.3.2.2)", ed.ScopeVariety())
	}
	ct := inlineComplexType(t, ed.TypeDefinition())
	if ct.BaseTypeDefinitionName() != (xsd.QName{Space: xsdNS, Local: "anyType"}) {
		t.Fatalf("{base type definition} = %s, want xs:anyType (§3.4.2.3.2)", ct.BaseTypeDefinitionName())
	}
	if len(ct.AttributeUses()) != 1 {
		t.Fatalf("{attribute uses} has %d members, want the one declared <attribute>", len(ct.AttributeUses()))
	}
	// The anonymous container has no {name} for the attribute's {scope}.{parent}
	// (§3.2.1 sc_a) to reference, so it is carried by the identity the type's own
	// {context} holds — the same one mint per inline construct the nested element
	// declarations use. == and never reflect.DeepEqual, which cannot see a
	// ComponentID.
	attr := ct.AttributeUses()[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration
	parent, ok := attr.Scope().Parent()
	if !ok {
		t.Fatal("the attribute of the inline anonymous complex type has no {scope}.{parent}")
	}
	anon, ok := parent.(xsd.AttributeAnonymousComplexTypeScopeParent)
	if !ok {
		t.Fatalf("{scope}.{parent} = %T, want an xsd.AttributeAnonymousComplexTypeScopeParent", parent)
	}
	if anon.Owner != ownerID(t, ct) {
		t.Error("the attribute's {scope}.{parent}.Owner is not the anonymous type's own {context} identity")
	}
	for _, td := range s.Types() {
		if td.Name() == (xsd.QName{}) {
			t.Fatalf("an anonymous type reached {type definitions}; it must stay reachable only through the owning declaration")
		}
	}
}

// TestProduceLocalElementInlineComplexType pins tier 1 on the LOCAL path, which
// §3.3.2.3 supplements only for {scope} and {target namespace} — never for
// {type definition}, so the mapping is the global one.
func TestProduceLocalElementInlineComplexType(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:complexType name="CT"><xs:sequence>
		<xs:element name="inner"><xs:complexType><xs:sequence/></xs:complexType></xs:element>
	</xs:sequence></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: "CT"})
	if !ok {
		t.Fatal("type CT not found")
	}
	decls := particleTerms(t, td.(xsd.ComplexType))
	if len(decls) != 1 {
		t.Fatalf("CT holds %d local element declarations, want 1", len(decls))
	}
	if decls[0].ScopeVariety() != xsd.ScopeLocal {
		t.Fatalf("scope = %s, want local (§3.3.2.3)", decls[0].ScopeVariety())
	}
	inlineComplexType(t, decls[0].TypeDefinition())
}

// TestProduceInlineComplexTypeScopesNestedLocals is the identity round trip: a
// local <element> two levels inside an inline anonymous <complexType> reports
// an AnonymousComplexTypeScopeParent whose Owner is the SAME ComponentID the
// anonymous type's own {context} carries (§3.4.2.1 dcl.ctd.common). The
// comparison is ==, never reflect.DeepEqual, which cannot see a ComponentID.
func TestProduceInlineComplexTypeScopesNestedLocals(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:element name="doc">
		<xs:complexType><xs:sequence>
			<xs:choice><xs:element name="deep" type="xs:string"/></xs:choice>
		</xs:sequence></xs:complexType>
	</xs:element>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ct := inlineComplexType(t, elementOf(t, s, "doc").TypeDefinition())
	id := ownerID(t, ct)

	content := ct.ContentType().(xsd.ElementContent)
	seq := content.Particle.Term().(xsd.ResolvedTerm).Term.(xsd.ModelGroup)
	choice := seq.Particles()[0].Term().(xsd.ResolvedTerm).Term.(xsd.ModelGroup)
	deep := choice.Particles()[0].Term().(xsd.ResolvedTerm).Term.(xsd.ElementDeclaration)

	parent, ok := deep.Scope().Parent()
	if !ok {
		t.Fatal("the nested declaration has no {scope}.{parent}")
	}
	anon, ok := parent.(xsd.AnonymousComplexTypeScopeParent)
	if !ok {
		t.Fatalf("{scope}.{parent} = %T, want an xsd.AnonymousComplexTypeScopeParent", parent)
	}
	if anon.Owner != id {
		t.Fatal("the nested declaration's {scope}.{parent}.Owner is not the anonymous type's own {context} identity")
	}
}

// TestProduceNestedInlineComplexTypesMintDistinctIdentities pins that each
// inline construct gets its OWN mint: an anonymous complex type inside an
// anonymous complex type must not share the outer one's identity, or every
// nested scope would collapse onto the outer container.
func TestProduceNestedInlineComplexTypesMintDistinctIdentities(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:element name="doc">
		<xs:complexType><xs:sequence>
			<xs:element name="inner"><xs:complexType><xs:sequence>
				<xs:element name="leaf" type="xs:string"/>
			</xs:sequence></xs:complexType></xs:element>
		</xs:sequence></xs:complexType>
	</xs:element>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	outer := inlineComplexType(t, elementOf(t, s, "doc").TypeDefinition())
	outerID := ownerID(t, outer)

	inner := particleTerms(t, outer)
	if len(inner) != 1 {
		t.Fatalf("the outer anonymous type holds %d local declarations, want 1", len(inner))
	}
	innerCT := inlineComplexType(t, inner[0].TypeDefinition())
	innerID := ownerID(t, innerCT)
	if innerID == outerID {
		t.Fatal("the nested anonymous complex type shares the outer one's identity; one mint per inline construct is required")
	}
	// The inner declaration is scoped to the OUTER container, its own leaf to
	// the inner one — the threading, not merely the mint, must differ.
	if parent, _ := inner[0].Scope().Parent(); parent != (xsd.AnonymousComplexTypeScopeParent{Owner: outerID}) {
		t.Fatalf("inner declaration {scope}.{parent} = %#v, want the OUTER container's identity", parent)
	}
	leaf := particleTerms(t, innerCT)
	if len(leaf) != 1 {
		t.Fatalf("the inner anonymous type holds %d local declarations, want 1", len(leaf))
	}
	if parent, _ := leaf[0].Scope().Parent(); parent != (xsd.AnonymousComplexTypeScopeParent{Owner: innerID}) {
		t.Fatalf("leaf declaration {scope}.{parent} = %#v, want the INNER container's identity", parent)
	}
}

// TestProduceSiblingInlineComplexTypesCarryDistinctOwners is the reverse of the
// DeepEqual trap: two anonymous complex types under DIFFERENT declarations must
// carry different identities even though their shapes are identical.
func TestProduceSiblingInlineComplexTypesCarryDistinctOwners(t *testing.T) {
	s, err := produce(t, wrap("", `
		<xs:element name="a"><xs:complexType><xs:sequence>
			<xs:element name="x" type="xs:string"/>
		</xs:sequence></xs:complexType></xs:element>
		<xs:element name="b"><xs:complexType><xs:sequence>
			<xs:element name="x" type="xs:string"/>
		</xs:sequence></xs:complexType></xs:element>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	first := inlineComplexType(t, elementOf(t, s, "a").TypeDefinition())
	second := inlineComplexType(t, elementOf(t, s, "b").TypeDefinition())
	if ownerID(t, first) == ownerID(t, second) {
		t.Fatal("two sibling anonymous complex types share one identity")
	}
	firstParent, _ := particleTerms(t, first)[0].Scope().Parent()
	secondParent, _ := particleTerms(t, second)[0].Scope().Parent()
	if firstParent == secondParent {
		t.Fatal("the two containers' nested declarations report the same {scope}.{parent}, so the scopes are indistinguishable")
	}
}

// TestProduceElementInlineComplexTypeWithTypeAttr pins src-element clause 3
// (§3.3.3) for the inline-<complexType> arm specifically, on BOTH paths — the
// clause names "<simpleType> or <complexType> child and a type attribute", and
// the widening this landing makes would otherwise let type= silently lose to an
// inline child (or win over it) rather than being rejected.
func TestProduceElementInlineComplexTypeWithTypeAttr(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"global", `<xs:element name="e" type="xs:string">
			<xs:complexType><xs:sequence/></xs:complexType></xs:element>`},
		{"local", `<xs:complexType name="CT"><xs:sequence>
			<xs:element name="e" type="xs:string">
				<xs:complexType><xs:sequence/></xs:complexType></xs:element>
		</xs:sequence></xs:complexType>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			assertRule(t, err, "src-element")
		})
	}
}

// TestProduceElementBothInlineTypesRejected pins the explicit rejection of an
// <element> carrying BOTH inline type children. The schema for schema documents
// allows one, and no src-element clause states the condition, so the rejection
// carries NO rule — a fabricated verdict would be worse than the silent
// complexType-wins the widening would otherwise introduce.
func TestProduceElementBothInlineTypesRejected(t *testing.T) {
	const both = `<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>` +
		`<xs:complexType><xs:sequence/></xs:complexType>`
	for _, tc := range []struct{ name, body string }{
		{"global", `<xs:element name="e">` + both + `</xs:element>`},
		{"local", `<xs:complexType name="CT"><xs:sequence>` +
			`<xs:element name="e">` + both + `</xs:element>` +
			`</xs:sequence></xs:complexType>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			if err == nil {
				t.Fatal("Produce accepted an <element> with both inline type children")
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Fatalf("error carries rule %s, want a plain grammar-fault error", rule)
			}
			if !strings.Contains(err.Error(), "both an inline <simpleType> and an inline <complexType>") {
				t.Fatalf("error = %v, want the both-children grammar fault", err)
			}
		})
	}
}

// TestResolveInlineComplexTypeReferences pins xsd/resolve.go's
// resolveTypeDefinition ComplexType inner arm: finalize must descend an
// anonymous complex type owned by a declaration and charge src-resolve for its
// dangling references, exactly as it does for a top-level one.
//
// Only the PARTICLE TREE (clause 1.3) is covered from here. The arm's other
// reference site, the {base type definition} (clause 1.1), is no longer reachable
// through the producer: resolveBaseType resolves every base= — on the restriction
// alternant too, since #346's §3.4.2.1 clause 1 {assertions} fold reads the base
// component on both — and charges the same src-resolve clause 1.1 first, and
// POSITIONED, which is what TestProduceComplexContentDanglingBase pins. Finalize
// keeps that arm for the SchemaBuilder, which has no producer to defend it
// (xsd/resolve_test.go's TestResolveAnonymousComplexTypeDanglingBase), the same
// two-entry-points-one-rule shape buildComplexType's cycle rejection has.
func TestResolveInlineComplexTypeReferences(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"dangling <element ref> in the particle tree", `<xs:element name="doc">
			<xs:complexType><xs:sequence>
				<xs:element ref="tns:missing"/>
			</xs:sequence></xs:complexType></xs:element>`},
		{"dangling ref nested in a local element's own inline type", `<xs:element name="doc">
			<xs:complexType><xs:sequence>
				<xs:element name="inner"><xs:complexType><xs:sequence>
					<xs:element ref="tns:missing"/>
				</xs:sequence></xs:complexType></xs:element>
			</xs:sequence></xs:complexType></xs:element>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", tc.body))
			assertRule(t, err, "src-resolve")
		})
	}
}

// TestProduceComplexContentDanglingBase pins the producer's own src-resolve
// clause 1.1 verdict on a base= that names nothing, on BOTH <complexContent>
// alternants and for a named type as well as an inline anonymous one. The
// restriction alternant reaches it because #346's {assertions} fold resolves the
// base component there too; the message is POSITIONED at the derivation element,
// which finalize's counterpart (xsderr.Loc{}) cannot be.
func TestProduceComplexContentDanglingBase(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"named restriction", `<xs:complexType name="ct"><xs:complexContent>
			<xs:restriction base="tns:missing"><xs:sequence/></xs:restriction>
		</xs:complexContent></xs:complexType>`},
		{"named extension", `<xs:complexType name="ct"><xs:complexContent>
			<xs:extension base="tns:missing"><xs:sequence/></xs:extension>
		</xs:complexContent></xs:complexType>`},
		{"inline anonymous restriction", `<xs:element name="doc">
			<xs:complexType><xs:complexContent>
				<xs:restriction base="tns:missing"><xs:sequence/></xs:restriction>
			</xs:complexContent></xs:complexType></xs:element>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", tc.body))
			assertRule(t, err, "src-resolve")
			if !strings.Contains(err.Error(), "mem://produce.xsd:") {
				t.Fatalf("error = %v, want a positioned producer diagnostic", err)
			}
		})
	}
}
