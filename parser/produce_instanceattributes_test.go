package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// TestProduceSeedsInstanceAttributes pins that a compiled schema carries the
// four Attribute Declarations §3.2.7 states are "present in every schema by
// definition" (xmlschema11-1.md:991), each with the property table its section
// gives: {target namespace} the instance namespace, {scope} global with
// {parent} ·absent·, {value constraint} ·absent·, and the {type definition}
// §3.2.7.1–.4 name.
//
// The document declares NOTHING — the four are seeded, not mapped, so a schema
// document with an empty <schema> must report all four. Each is asserted
// through xsd.Schema.Attribute, the published by-name query, since resolving
// <xs:attribute ref="xsi:type"/> against src-resolve clause 1.2 is what the
// seeding exists for.
//
// xsi:schemaLocation is the one that cannot be asserted as a by-name reference:
// §3.2.7.3 fills its slot with an ANONYMOUS list-variety simple type over
// anyURI, not with anyURI itself, so its row asserts the inline arm, the list
// {variety} and the {item type definition} separately. A seeding that pointed
// it at anyURI directly would satisfy every other assertion here.
func TestProduceSeedsInstanceAttributes(t *testing.T) {
	s, err := produce(t, wrap("", ""))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}

	// A slice, not a map: subtest order is output (STYLE D2). wantType is the
	// by-name {type definition} each row expects, and is the zero QName for the
	// one row whose slot is an inline anonymous type instead.
	cases := []struct {
		local    string
		wantType xsd.QName
	}{
		{local: "type", wantType: xsd.QName{Space: xsdNS, Local: "QName"}},
		{local: "nil", wantType: xsd.QName{Space: xsdNS, Local: "boolean"}},
		{local: "schemaLocation"},
		{local: "noNamespaceSchemaLocation", wantType: xsd.QName{Space: xsdNS, Local: "anyURI"}},
	}
	for _, tc := range cases {
		t.Run(tc.local, func(t *testing.T) {
			name := xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: tc.local}
			d, found := s.Attribute(name)
			if !found {
				t.Fatalf("Schema.Attribute(%s) reports no declaration, want the §3.2.7 built-in one", name)
			}
			if got := d.Name(); got != name {
				t.Fatalf("{name} = %s, want %s", got, name)
			}
			if got := d.ScopeVariety(); got != xsd.ScopeGlobal {
				t.Fatalf("{scope}.{variety} = %v, want global", got)
			}
			if vc, has := d.ValueConstraint(); has {
				t.Fatalf("{value constraint} = %v, want ·absent·", vc)
			}
			if tc.wantType != (xsd.QName{}) {
				if got := declaredTypeName(t, d.TypeDefinition()); got != tc.wantType {
					t.Fatalf("{type definition} = %s, want %s", got, tc.wantType)
				}
				return
			}

			inline, ok := d.TypeDefinition().(xsd.InlineTypeDefinition)
			if !ok {
				t.Fatalf("{type definition} = %#v, want the anonymous list type §3.2.7.3 gives, not a by-name reference", d.TypeDefinition())
			}
			st, ok := inline.Definition.(*xsd.SimpleType)
			if !ok {
				t.Fatalf("{type definition} wraps %#v, want a *xsd.SimpleType", inline.Definition)
			}
			if got := st.Name(); got != (xsd.QName{}) {
				t.Fatalf("anonymous {type definition} {name} = %s, want ·absent· (the zero QName)", got)
			}
			variety, err := st.Variety(s)
			if err != nil {
				t.Fatalf("Variety: %v", err)
			}
			if _, isList := variety.(xsd.List); !isList {
				t.Fatalf("{variety} = %T, want list", variety)
			}
			item, err := st.Item(s)
			if err != nil {
				t.Fatalf("Item: %v", err)
			}
			if item == nil {
				t.Fatalf("{item type definition} is nil, want the built-in anyURI")
			}
			if got, want := item.Name(), (xsd.QName{Space: xsdNS, Local: "anyURI"}); got != want {
				t.Fatalf("{item type definition} = %s, want %s", got, want)
			}
		})
	}
}

// TestProduceInstanceAttributeRefResolves pins the defect the seeding closes:
// <xs:attribute ref="xsi:type"/> — and the same for the other three §3.2.7
// names — resolves at finalize instead of being charged src-resolve clause 1.2,
// which is the charge all seven Complex/complex003–complex010 schema cases sat
// on. The attribute use is asserted, not merely the absence of an error: a
// reference that resolved to nothing would still have to fail here.
func TestProduceInstanceAttributeRefResolves(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, local := range []string{"type", "nil", "schemaLocation", "noNamespaceSchemaLocation"} {
		t.Run(local, func(t *testing.T) {
			// Spelled out rather than built with wrap, which binds only xs and the
			// target namespace: the ref needs the instance namespace bound to have
			// an expanded name at all.
			doc := `<xs:schema xmlns:xs="` + xsdNS + `" xmlns:xsi="` + xsd.XMLSchemaInstanceNS +
				`" targetNamespace="urn:po">` + "\n" +
				`<xs:complexType name="ct">` + "\n" +
				`<xs:attribute ref="xsi:` + local + `"/>` + "\n" +
				`</xs:complexType>` + "\n" +
				`</xs:schema>`

			s, err := produce(t, doc)
			if err != nil {
				t.Fatalf("Produce: %v, want the §3.2.7 built-in declaration to resolve the ref (src-resolve clause 1.2)", err)
			}
			ct, found := s.Type(xsd.QName{Space: "urn:po", Local: "ct"})
			if !found {
				t.Fatalf("Schema.Type reports no {urn:po}ct")
			}
			complexType, ok := ct.(xsd.ComplexType)
			if !ok {
				t.Fatalf("{urn:po}ct = %T, want an xsd.ComplexType", ct)
			}
			uses := complexType.AttributeUses()
			if len(uses) != 1 {
				t.Fatalf("{attribute uses} = %d, want 1", len(uses))
			}
			want := xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: local}
			if got := uses[0].DeclarationName(); got != want {
				t.Fatalf("{attribute uses}[0] declaration name = %s, want %s", got, want)
			}
			if _, ok := s.ResolvedAttributeDeclaration(uses[0]); !ok {
				t.Fatalf("{attribute uses}[0] resolves to no declaration, want the §3.2.7 built-in %s", want)
			}
		})
	}
}
