package xsd

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are about the BY-NAME arm of {base type definition}, which is why
// they resolve against a real finalized Schema rather than against
// simpletypefixture_test.go's noSchema: a SimpleTypeRef is exactly the arm a
// resolves-nothing resolver cannot follow.

// refType builds a named simple type whose base is a by-name SimpleTypeRef. It
// deliberately does NOT run CheckDerivation — the whole point of the deferral is
// that a component naming a base it cannot yet see is well FORMED, and only
// finalize decides whether it is well DERIVED.
func refType(t *testing.T, local string, base QName) *SimpleType {
	t.Helper()
	st, err := NewSimpleType(xsderr.Loc{}, QName{Local: local}, RestrictionDerivation{}, SimpleTypeRef{Name: base}, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%s over a SimpleTypeRef): %v", local, err)
	}
	return st
}

// TestSimpleBaseCycleRejected is st-props-correct clause 2's pin: TWO named
// simple types whose base= references close a cycle. Neither could exist while
// the slot held a live pointer — a base had to pre-exist the type naming it — so
// this is the rule the deferral makes checkable and checkSimpleBaseAcyclic
// exists to charge.
func TestSimpleBaseCycleRejected(t *testing.T) {
	b := NewSchemaBuilder()
	b.AddType(refType(t, "A", QName{Local: "B"}))
	b.AddType(refType(t, "B", QName{Local: "A"}))

	_, err := b.Finalize()
	if err == nil {
		t.Fatal("Finalize(A base=B, B base=A) = nil, want a circular-base rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != ruleSTPropsCorrect {
		t.Fatalf("charged %s, want %s", r, ruleSTPropsCorrect)
	}
	if !strings.Contains(err.Error(), "st-props-correct clause 2") {
		t.Fatalf("message does not name clause 2, the clause this check owns: %v", err)
	}
	if !strings.Contains(err.Error(), "circular {base type definition} chain") {
		t.Fatalf("message does not name the circularity it found: %v", err)
	}
}

// TestSimpleBaseCycleThroughOwnedArmRejected is ruling 4(c)'s pin: the walk must
// traverse BOTH arms. An owned arm alone cannot close a cycle — an owned
// component must pre-exist the slot holding it — but a MIXED chain can, and it
// is exactly the shape a redefining <simpleType> produces: a named type whose
// base is an anonymous OWNED original whose own base= names a type again. A walk
// that stopped at the owned hop would miss every cycle running through a
// redefinition.
func TestSimpleBaseCycleThroughOwnedArmRejected(t *testing.T) {
	// anon's base= names A; A's base is anon, owned. A → anon → A.
	anon, err := NewSimpleType(xsderr.Loc{}, QName{}, RestrictionDerivation{}, SimpleTypeRef{Name: QName{Local: "A"}}, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(anonymous original): %v", err)
	}
	a, err := NewSimpleType(xsderr.Loc{}, QName{Local: "A"}, RestrictionDerivation{}, OwnedSimpleType{Definition: anon}, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(A over the owned original): %v", err)
	}

	b := NewSchemaBuilder()
	b.AddType(a)
	_, err = b.Finalize()
	if err == nil {
		t.Fatal("Finalize(A base=<owned anon base=A>) = nil, want a circular-base rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != ruleSTPropsCorrect {
		t.Fatalf("charged %s, want %s", r, ruleSTPropsCorrect)
	}
}

// TestSimpleBaseAcyclicAcceptsSharedAncestor guards the other polarity: a
// diamond over one shared ancestor is NOT a cycle, and the per-root path map
// must not report one just because a node is reached twice across roots.
func TestSimpleBaseAcyclicAcceptsSharedAncestor(t *testing.T) {
	root, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "string"}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(root)
	b.AddType(refType(t, "L", root.Name()))
	b.AddType(refType(t, "R", root.Name()))
	if _, err := b.Finalize(); err != nil {
		t.Fatalf("Finalize(two restrictions of one primitive) = %v, want acceptance", err)
	}
}

// TestSimpleBaseDanglingRejected pins the OTHER half of the deferral: a base=
// naming nothing is charged src-resolve clause 1.1 at finalize, from the sum's
// one resolution helper — no longer from the producer.
func TestSimpleBaseDanglingRejected(t *testing.T) {
	b := NewSchemaBuilder()
	b.AddType(refType(t, "A", QName{Local: "missing"}))

	_, err := b.Finalize()
	if err == nil {
		t.Fatal("Finalize(base= naming nothing) = nil, want a src-resolve rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != ruleSrcResolve {
		t.Fatalf("charged %s, want %s", r, ruleSrcResolve)
	}
	if !strings.Contains(err.Error(), "src-resolve clause 1.1") {
		t.Fatalf("message does not name clause 1.1: %v", err)
	}
}

// TestSimpleBaseWrongKindRejected pins simpleTypeOfRef's second charge: a base=
// whose name resolves to a COMPLEX type is the same failure as a miss — the
// kind-specific lookup finds nothing — and is charged the same rule.
func TestSimpleBaseWrongKindRejected(t *testing.T) {
	// xs:anyType is its own base (§3.4.7), the one self-derivation the complex
	// side permits; C restricts it so the schema resolves on the complex side too.
	anyType, err := NewComplexType(xsderr.Loc{}, anyTypeName, anyTypeName, nil,
		DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType(xs:anyType): %v", err)
	}
	ct, err := NewComplexType(xsderr.Loc{}, QName{Local: "C"}, anyTypeName, nil,
		DerivationRestriction, false, nil, nil, nil, EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	b := NewSchemaBuilder()
	b.AddType(anyType)
	b.AddType(ct)
	b.AddType(refType(t, "A", QName{Local: "C"}))

	_, err = b.Finalize()
	if err == nil {
		t.Fatal("Finalize(simple base= naming a complex type) = nil, want a src-resolve rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != ruleSrcResolve {
		t.Fatalf("charged %s, want %s", r, ruleSrcResolve)
	}
	if !strings.Contains(err.Error(), "complex type definition") {
		t.Fatalf("message does not say the name resolved to a complex type: %v", err)
	}
}

// TestSimpleTypeOrRefRejectsIllegalEncodings pins the two constructor
// rejections SimpleTypeOrRef's doc declares: a reference that names nothing, and
// an owned arm holding nothing. Both are representation invariants, so both are
// charged xsderr.RuleComponentInvariant rather than a spec rule.
func TestSimpleTypeOrRefRejectsIllegalEncodings(t *testing.T) {
	for _, tc := range []struct {
		name string
		base SimpleTypeOrRef
		want string
	}{
		{"a ref naming nothing", SimpleTypeRef{}, "absent (zero) QName"},
		{"an owned arm holding nothing", OwnedSimpleType{}, "no definition"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSimpleType(xsderr.Loc{}, QName{Local: "D"}, RestrictionDerivation{}, tc.base, nil, nil)
			if err == nil {
				t.Fatal("NewSimpleType = nil, want a component-invariant rejection")
			}
			if r, _ := xsderr.RuleOf(err); r != xsderr.RuleComponentInvariant {
				t.Fatalf("charged %s, want %s", r, xsderr.RuleComponentInvariant)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("message does not name the illegal encoding (%q): %v", tc.want, err)
			}
		})
	}
}

// TestDeferredBaseReadersResolveThroughSchema pins B4's whole point: the derived
// readers follow a by-name base when handed a resolver, and report the
// src-resolve error — never a silently short chain — when the name resolves to
// nothing. A short chain would answer "absent {variety}", the shape
// st-props-correct clause 1 reserves for xs:anySimpleType alone, so the two
// polarities are asserted together.
func TestDeferredBaseReadersResolveThroughSchema(t *testing.T) {
	prim, err := newCheckedPrimitiveType(xsderr.Loc{}, QName{Space: XMLSchemaNS, Local: "string"}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	derived := refType(t, "D", prim.Name())

	b := NewSchemaBuilder()
	b.AddType(prim)
	b.AddType(derived)
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	base, err := derived.Base(s)
	if err != nil {
		t.Fatalf("D.Base(schema): %v", err)
	}
	if base != prim {
		t.Fatalf("D.Base(schema) = %v, want the very component {type definitions} holds", base)
	}
	variety, err := derived.Variety(s)
	if err != nil {
		t.Fatalf("D.Variety(schema): %v", err)
	}
	if _, ok := variety.(Atomic); !ok {
		t.Fatalf("D.Variety(schema) = %T, want Atomic derived through the reference", variety)
	}
	primitive, err := derived.Primitive(s)
	if err != nil {
		t.Fatalf("D.Primitive(schema): %v", err)
	}
	if primitive != prim {
		t.Fatalf("D.Primitive(schema) = %v, want the primitive reached through the reference", primitive)
	}

	// The same reads against a resolver that resolves nothing must ERROR, not
	// answer a truncated chain.
	for _, tc := range []struct {
		name string
		read func() error
	}{
		{"Base", func() error { _, err := derived.Base(noSchema{}); return err }},
		{"Variety", func() error { _, err := derived.Variety(noSchema{}); return err }},
		{"Primitive", func() error { _, err := derived.Primitive(noSchema{}); return err }},
		{"Item", func() error { _, err := derived.Item(noSchema{}); return err }},
		{"Members", func() error { _, err := derived.Members(noSchema{}); return err }},
		{"EffectiveFacets", func() error { _, err := derived.EffectiveFacets(noSchema{}); return err }},
	} {
		t.Run(tc.name+" surfaces an unresolvable base", func(t *testing.T) {
			err := tc.read()
			if err == nil {
				t.Fatalf("D.%s(resolver that resolves nothing) = nil error, want the src-resolve failure rather than a short chain", tc.name)
			}
			if r, _ := xsderr.RuleOf(err); r != ruleSrcResolve {
				t.Fatalf("charged %s, want %s", r, ruleSrcResolve)
			}
		})
	}
}
