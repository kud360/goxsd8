package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin resolveAttributeDecl: an attribute declaration's by-name
// {type definition} must resolve to a SIMPLE type definition (§3.2.2.1
// dcl.att.global, §3.2.2.2 dcl.att.local), so a type= naming a complex type is
// charged src-resolve at the declaration's own position, on every route Phase A
// reaches an attribute declaration by. They are package-internal for the
// redefinition route, whose original is reachable only through unexported
// state. The component builders come from attributegroupfold_test.go and
// complexderivation_test.go (STYLE T4).

// kindAttrDecl builds an attribute declaration named local at loc whose type=
// names typeName, global when local is false.
func kindAttrDecl(t *testing.T, loc xsderr.Loc, local bool, name, typeName QName) AttributeDeclaration {
	t.Helper()
	scope := NewAttributeGlobalScope()
	if local {
		scope = aLocalScope(t)
	}
	d, err := NewAttributeDeclaration(loc, name, TypeDefinitionRef{Name: typeName}, scope, nil, false)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration(%s): %v", name, err)
	}
	return d
}

// kindAttrUse wraps kindAttrDecl's local declaration in an attribute use, as
// one member of a container's attribute content.
func kindAttrUse(t *testing.T, loc xsderr.Loc, typeName QName) AttributeUseOrGroupRef {
	t.Helper()
	u, err := NewAttributeUse(xsderr.Loc{}, false, LocalAttributeDeclaration{Declaration: kindAttrDecl(t, loc, true, uq("a"), typeName)}, nil, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	return ResolvedAttributeUse{Use: u}
}

// TestAttributeTypeNamingComplexTypeRejected pins the charge on each route: a
// global declaration, a local one in a complex type, a local one in an
// attribute group, the S2 original of an attribute group redefinition (in no
// index, reached only through the redefinition loop), and xs:anyType as the
// complex type named. The prefix pins the position, the rule and the subject,
// so a rejection charged at the container rather than at the <attribute>, or a
// message naming the wrong side, fails it. Each case's simple twin — the same
// shape naming the str primitive — must finalize, so the charge is the kind
// and not the shape.
func TestAttributeTypeNamingComplexTypeRejected(t *testing.T) {
	at := resolveLocAt(7)
	for _, tc := range []struct {
		name   string
		decl   string // the rejected declaration's name, as the message renders it
		target QName  // the complex type named
		build  func(b *SchemaBuilder, typeName QName)
	}{
		{"global declaration", uq("g").String(), uq("C"), func(b *SchemaBuilder, typeName QName) {
			b.AddAttribute(kindAttrDecl(t, at, false, uq("g"), typeName))
		}},
		{"local declaration in a complex type", uq("a").String(), uq("C"), func(b *SchemaBuilder, typeName QName) {
			b.AddType(gTypeAt(t, resolveLocAt(5), "D", nil, kindAttrUse(t, at, typeName)))
		}},
		{"local declaration in an attribute group", uq("a").String(), uq("C"), func(b *SchemaBuilder, typeName QName) {
			b.AddAttributeGroup(gGroupAt(t, resolveLocAt(5), "G", nil, kindAttrUse(t, at, typeName)))
		}},
		{"local declaration in a redefined attribute group's original", uq("a").String(), uq("C"), func(b *SchemaBuilder, typeName QName) {
			b.AddRedefiningAttributeGroup(gGroup(t, "G", nil),
				gGroupAt(t, resolveLocAt(5), "G", nil, kindAttrUse(t, at, typeName)))
		}},
		{"global declaration naming xs:anyType", uq("g").String(), anyTypeName, func(b *SchemaBuilder, typeName QName) {
			b.AddAttribute(kindAttrDecl(t, at, false, uq("g"), typeName))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := gFinalize(t, func(b *SchemaBuilder) {
				b.AddType(gType(t, "C", nil))
				tc.build(b, tc.target)
			})
			if err == nil {
				t.Fatal("Finalize(attribute type= naming a complex type) = nil, want a src-resolve rejection")
			}
			want := at.String() + ": [src-resolve] attribute declaration " + tc.decl +
				" {type definition} references simple type " + tc.target.String() +
				", but that expanded name is a complex type definition"
			if !strings.HasPrefix(err.Error(), want) {
				t.Fatalf("Finalize error = %q, want prefix %q", err, want)
			}
			gSchema(t, func(b *SchemaBuilder) {
				b.AddType(gType(t, "C", nil))
				tc.build(b, uq("str"))
			})
		})
	}
}
