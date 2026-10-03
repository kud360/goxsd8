package validate

import (
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive the children a parent DECIDEDLY ·attributes· to
// nothing (#1892): key-governing-ed clause 4 carries no attribution condition,
// so each is governed by its ·locally declared type· within the parent's
// complex type where that is non-·absent· (clause 4.3, key-governing-type-elem
// clause 7), else by the declaration its name ·resolves· to, else ·laxly
// assessed· — and none withholds cvc-id clause 1. Each row fails with
// walk.child treating that child as undecided, the pre-#1892 behaviour.

// decidedSchema declares "root" over the type named rootType with the given
// {nillable}, beside the seeded builtins, a top-level <kid> over KidType —
// whose sequence wants a <grand> its fixtures never carry — and the extra
// types the fixture adds.
func decidedSchema(t *testing.T, rootType xsd.QName, nillable bool, extra ...xsd.TypeDefinition) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	for _, st := range seeded {
		b.AddType(st)
	}
	for _, td := range extra {
		b.AddType(td)
	}
	b.AddType(dType(t, "KidType", "", xsd.DerivationRestriction, nil,
		cSequence(t, false, cParticle(t, "grand", 1, 1))))
	b.AddElement(dTopLevel(t, "kid", "KidType"))
	root, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: rootType}, nil, xsd.NewGlobalScope(),
		nil, nillable, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b.AddElement(root)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the decided-attribution schema: %v", err)
	}
	return schema
}

// decidedRootType is a RootType over content.
func decidedRootType(t *testing.T, content xsd.ContentType) xsd.ComplexType {
	t.Helper()
	return dType(t, "RootType", "", xsd.DerivationRestriction, nil, content)
}

// seededString is the seeded xs:string, the {simple type definition} a simple
// {content type} fixture carries.
func seededString(t *testing.T) *xsd.SimpleType {
	t.Helper()
	return icSeeded(t)["string"]
}

// rootNilled is <root xsi:nil="true"> over kids.
func rootNilled(kids ...Child) *testElement {
	e := dElem("root", 1, kids...)
	e.attrs = []Attribute{&testAttribute{
		name:  xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"},
		value: "true", loc: loc(1, 10)}}
	return e
}

// Each row's parent attributes the child at line 3 to nothing for a reason the
// spec decides, and the child is then governed per key-governing-ed clause 4:
// <kid> by the top-level declaration its name ·resolves· to, whose KidType it
// fails at its own position under cvc-complex-content clause 1, and <a> by its
// ·locally declared type· KidType within RootType, which it fails the same way
// although no top-level <a> exists. The parent's own charge comes first, at the
// child, or at the text run of line 2 for the charged-sibling rows. Every row
// fails with the fix reverted: the child is then assessed against nothing and
// its cvc-complex-content charge is absent.
func TestDecidedlyUnattributedChildIsGovernedPerClause4(t *testing.T) {
	elementOnly := decidedRootType(t, cSequence(t, false, cParticle(t, "x", 0, 1)))
	localA := decidedRootType(t, cSequence(t, false, dTyped(t, "RootType", "a", "KidType")))
	rootType := xsd.QName{Local: "RootType"}
	text := TextChild(&testText{data: "not white space", loc: loc(2, 1)})
	for _, tc := range []struct {
		why    string
		schema *xsd.Schema
		root   Element
		parent xsderr.Rule
		at     xsderr.Loc
	}{
		{"charged sibling", decidedSchema(t, rootType, false, elementOnly),
			dElem("root", 1, text, ElementChild(dElem("kid", 3))), ruleCvcComplexType, loc(2, 1)},
		{"charged sibling, locally declared type", decidedSchema(t, rootType, false, localA),
			dElem("root", 1, text, ElementChild(dElem("a", 3))), ruleCvcComplexType, loc(2, 1)},
		{"nilled parent", decidedSchema(t, rootType, true, elementOnly),
			rootNilled(ElementChild(dElem("kid", 3))), ruleCvcElt, loc(3, 1)},
		{"nilled parent, locally declared type", decidedSchema(t, rootType, true, localA),
			rootNilled(ElementChild(dElem("a", 3))), ruleCvcElt, loc(3, 1)},
		{"simple-typed parent", decidedSchema(t, icBuiltin("string"), false),
			dElem("root", 1, ElementChild(dElem("kid", 3))), ruleCvcType, loc(3, 1)},
		{"empty content", decidedSchema(t, rootType, false, decidedRootType(t, xsd.EmptyContent{})),
			dElem("root", 1, ElementChild(dElem("kid", 3))), ruleCvcComplexType, loc(3, 1)},
		{"simple content", decidedSchema(t, rootType, false,
			decidedRootType(t, xsd.SimpleContent{SimpleType: seededString(t)})),
			dElem("root", 1, ElementChild(dElem("kid", 3))), ruleCvcComplexType, loc(3, 1)},
		{"item no particle admits", decidedSchema(t, rootType, false, elementOnly),
			dElem("root", 1, ElementChild(dElem("kid", 3))), ruleCvcComplexContent, loc(3, 1)},
	} {
		t.Run(tc.why, func(t *testing.T) {
			got := cAssess(t, tc.schema, tc.root)
			icWantCharges(t, got, icChargeAt(tc.parent, tc.at), icCharge(ruleCvcComplexContent, 3))
			if len(got) == 2 && got[1].Rule == ruleCvcComplexContent {
				wantContentCharge(t, got[1:], ruleCvcComplexContent, "1", loc(3, 1))
			}
		})
	}
}

// A decidedly unattributed child no longer withholds cvc-id clause 1: <junk>,
// the child of an empty-content <ref> (cvc-complex-type clause 1.1), is
// ·laxly assessed· and read into the ID/IDREF table like any other element, so
// the IDREF "ghost" nothing declares is charged at the validation root. It
// fails with walk.child setting ids.declined for that child, which suppresses
// the clause 1 arm.
func TestDecidedlyUnattributedChildWithholdsNoCvcIDClause1(t *testing.T) {
	junk := icElem(xsd.QName{Local: "junk"}, 4, nil)
	ref := icElem(xsd.QName{Local: "ref"}, 3, nil, ElementChild(junk))
	got := icAssess(t, idSchema(t), icRoot(idItem(2, "ref", "ghost"), ref))
	icWantCharges(t, got, icCharge(ruleCvcComplexType, 4), icChargeAttr(ruleCvcID, 2))
}

// An undecided child still withholds cvc-id clause 1: <a>, a child of a
// <root> whose clause 1.4 xsd.Schema.ContentMatcher declines, is walked
// against nothing, so an xs:ID anywhere beneath it is one the ID/IDREF table
// never saw, and the root's IDREF "ghost" is not charged. It fails with
// walk.child's undecided branch no longer setting ids.declined, the clause 1
// charge then reaching [Result].
func TestUndecidedChildWithholdsCvcIDClause1(t *testing.T) {
	ct := dType(t, "RootType", "", xsd.DerivationRestriction,
		[]xsd.AttributeUse{icUse(t, xsd.QName{Local: "ref"}, "IDREF")}, declinedContent(t))
	schema := cSchemaFrom(t, ct, func(b *xsd.SchemaBuilder) {
		seeded, err := builtin.Seed(testBackend())
		if err != nil {
			t.Fatalf("seeding the builtin types: %v", err)
		}
		for _, st := range seeded {
			b.AddType(st)
		}
	})
	root := icElem(xsd.QName{Local: "root"}, 1, []Attribute{icAttr(xsd.QName{Local: "ref"}, "ghost", 1)},
		ElementChild(dElem("a", 2)))
	got, unevaluated := assessRecorded(t, schema, root)
	wantSilence(t, got, "an undecided child withholds cvc-id clause 1")
	wantDeclines(t, unevaluated,
		Unevaluated{rule: ruleCvcComplexContent, loc: loc(2, 1), msg: "cvc-complex-content clause 1"})
}

// A decided child's xsi:nil IS charged: under a ·nilled· or simple-typed
// parent, <stray> ·resolves· to no declaration and is ·laxly assessed·, so
// cvc-attribute clause 3 charges its xsi:nil lexical outside xs:boolean
// against the built-in declaration that governs the attribute
// (key-governing-ad, §3.2.7.2). It fails with walk.child treating the child as
// undecided, whose guard in walk.instanceNilLexical then charges nothing.
func TestDecidedChildXSINilIsCharged(t *testing.T) {
	stray := func() *testElement { return icElem(xsd.QName{Local: "stray"}, 3, []Attribute{xsiNilAttr("maybe")}) }
	elementOnly := decidedRootType(t, cSequence(t, false, cParticle(t, "x", 0, 1)))
	for _, tc := range []struct {
		why    string
		schema *xsd.Schema
		root   Element
		parent xsderr.Rule
	}{
		{"nilled parent", decidedSchema(t, xsd.QName{Local: "RootType"}, true, elementOnly),
			rootNilled(ElementChild(stray())), ruleCvcElt},
		{"simple-typed parent", decidedSchema(t, icBuiltin("string"), false),
			dElem("root", 1, ElementChild(stray())), ruleCvcType},
	} {
		t.Run(tc.why, func(t *testing.T) {
			icWantCharges(t, cAssess(t, tc.schema, tc.root),
				icCharge(tc.parent, 3), icChargeAt(ruleCvcAttribute, loc(4, 20)))
		})
	}
}
