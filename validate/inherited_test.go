package validate

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below drive §3.3.5.6's [inherited attributes] into the one
// reader it has, §3.12.4 key-cta-ta-select clause 1.1.3, over the three-level
// tree <root><mid><leaf/></mid></root>. <leaf> is the top-level declaration
// carrying a {type table} whose one alternative, `@lang = 'de'`, selects First;
// its {default type definition} is Fallback. First requires needFirst and <leaf>
// never carries it, so a cvc-complex-type clause 3 charge naming needFirst is
// how a test sees that the {test} read lang as 'de', and its absence that it
// did not.

// inhLeaf is the top-level <leaf> declaration.
func inhLeaf(t *testing.T) xsd.ElementDeclaration {
	t.Helper()
	test := xsd.NewXPathExpression("@lang = 'de'", nil, nil, nil)
	table, err := xsd.NewTypeTable(xsderr.Loc{},
		[]xsd.TypeAlternative{namedTypeAlternative(t, &test, local("First"))},
		namedTypeAlternative(t, nil, local("Fallback")))
	if err != nil {
		t.Fatalf("building the type table: %v", err)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("leaf"),
		xsd.TypeDefinitionRef{Name: local("Fallback")}, &table, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the leaf element declaration: %v", err)
	}
	return d
}

// inhUse is an optional use of lang over a local declaration, with the use's
// own {inheritable} and {value constraint}; the declaration's {inheritable} is
// false, so only key-p-inherited clause 3.1 can make it inheritable.
func inhUse(t *testing.T, inheritable bool, vc *xsd.ValueConstraint) xsd.AttributeUse {
	t.Helper()
	return inhNamedUse(t, "lang", inheritable, vc)
}

// inhNamedUse is inhUse for an attribute named name.
func inhNamedUse(t *testing.T, name string, inheritable bool, vc *xsd.ValueConstraint) xsd.AttributeUse {
	t.Helper()
	decl, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, local(name), nil,
		xsd.NewAttributeGlobalScope(), nil, false)
	if err != nil {
		t.Fatalf("building the %s attribute declaration: %v", name, err)
	}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false,
		xsd.LocalAttributeDeclaration{Declaration: decl}, vc, &inheritable)
	if err != nil {
		t.Fatalf("building the %s attribute use: %v", name, err)
	}
	return u
}

// inhRefUse is an optional use of lang referring to the top-level declaration
// (inhGlobal), with the use's own inheritable attribute (nil where it writes
// none) and {value constraint}.
func inhRefUse(t *testing.T, inheritable *bool, vc *xsd.ValueConstraint) xsd.AttributeUse {
	t.Helper()
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false,
		xsd.AttributeDeclarationRef{Name: local("lang")}, vc, inheritable)
	if err != nil {
		t.Fatalf("building the lang reference use: %v", err)
	}
	return u
}

// inhGlobal is the top-level lang declaration with the given {inheritable}.
func inhGlobal(t *testing.T, inheritable bool) xsd.AttributeDeclaration {
	t.Helper()
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, local("lang"), nil,
		xsd.NewAttributeGlobalScope(), nil, inheritable)
	if err != nil {
		t.Fatalf("building the top-level lang declaration: %v", err)
	}
	return d
}

// inhFixture is one schema's variable parts: RootType's and MidType's
// {attribute uses}, RootType's {attribute wildcard}, the particle RootType's
// sequence holds (a local <mid> typed MidType where nil), and the top-level
// declarations the schema adds besides <leaf>.
type inhFixture struct {
	rootUses, midUses []xsd.AttributeUse
	rootWild          *xsd.Wildcard
	midParticle       *xsd.Particle
	global            func(*xsd.SchemaBuilder)
}

// schema builds f. MidType's sequence holds <leaf> by reference, so <leaf> is
// governed by the tabled top-level declaration.
func (f inhFixture) schema(t *testing.T) *xsd.Schema {
	t.Helper()
	mid := dTyped(t, "RootType", "mid", "MidType")
	if f.midParticle != nil {
		mid = *f.midParticle
	}
	root, err := xsd.NewComplexType(xsderr.Loc{}, local("RootType"), xsd.QName{}, nil,
		xsd.DerivationRestriction, false, attrContent(f.rootUses), nil, f.rootWild,
		cSequence(t, false, mid), nil, nil)
	if err != nil {
		t.Fatalf("building RootType: %v", err)
	}
	return cSchemaFrom(t, root, func(b *xsd.SchemaBuilder) {
		for _, st := range ctaBuiltins(t) {
			b.AddType(st)
		}
		b.AddType(ctaFallbackType(t))
		b.AddType(ctaCandidateType(t, "First"))
		b.AddType(dType(t, "MidType", "", xsd.DerivationRestriction, f.midUses,
			cSequence(t, false, dParticleOver(t, xsd.ElementDeclarationRef{Name: local("leaf")}))))
		b.AddElement(inhLeaf(t))
		if f.global != nil {
			f.global(b)
		}
	})
}

// inhLang is a lang attribute carrying value, or none where value is empty.
func inhLang(value string, line int) []Attribute {
	if value == "" {
		return nil
	}
	return []Attribute{&testAttribute{name: local("lang"), value: value, loc: loc(line, 5)}}
}

// inhDoc is <root lang=root><mid lang=mid><leaf lang=leaf/></mid></root>, an
// empty value leaving that element without lang.
func inhDoc(root, mid, leaf string) *testElement {
	l := dElem("leaf", 3)
	l.attrs = inhLang(leaf, 3)
	m := dElem("mid", 2, ElementChild(l))
	m.attrs = inhLang(mid, 2)
	r := dElem("root", 1, ElementChild(m))
	r.attrs = inhLang(root, 1)
	return r
}

// inhWantSelected fails unless <leaf>'s {test} selected First exactly where
// want says so: a needFirst charge at <leaf>'s location, or none anywhere.
func inhWantSelected(t *testing.T, got []*xsderr.Error, want bool, why string) {
	t.Helper()
	selected := false
	for _, v := range got {
		if strings.Contains(v.Msg, "needFirst") && v.Loc == loc(3, 1) {
			selected = true
		}
	}
	if selected != want {
		t.Errorf("%s: <leaf> selected First = %v, want %v; violations %v", why, selected, want, got)
	}
}

// key-p-inherited clause 1 is "one of E's ancestors", at any depth: <mid>
// carries no lang, so the one <leaf> reads is <root>'s, two levels up — which
// only a composition that hands <mid>'s own [inherited attributes] down, and
// not just <mid>'s own [[attributes]], can deliver. The second row is the
// discriminating one for that: <mid> carries an inheritable attribute of its
// OWN, under another name, so the set it hands down is a merge of both levels
// and not <root>'s passed through. The same lang under a use whose
// {inheritable} is false is not ·potentially inherited· (clause 3.1).
func TestGrandparentAttributeIsInherited(t *testing.T) {
	inhWantSelected(t, cAssess(t,
		inhFixture{rootUses: []xsd.AttributeUse{inhUse(t, true, nil)}}.schema(t),
		inhDoc("de", "", "")), true, "an inheritable lang two levels up")
	doc := inhDoc("de", "", "")
	mid, _ := doc.kids[0].Element()
	mid.(*testElement).attrs = []Attribute{&testAttribute{name: local("region"), value: "eu", loc: loc(2, 5)}}
	inhWantSelected(t, cAssess(t,
		inhFixture{
			rootUses: []xsd.AttributeUse{inhUse(t, true, nil)},
			midUses:  []xsd.AttributeUse{inhNamedUse(t, "region", true, nil)},
		}.schema(t), doc), true, "an inheritable lang two levels up, past <mid>'s own inheritable region")
	inhWantSelected(t, cAssess(t,
		inhFixture{rootUses: []xsd.AttributeUse{inhUse(t, false, nil)}}.schema(t),
		inhDoc("de", "", "")), false, "a non-inheritable lang two levels up")
}

// e-inherited_attributes clause 2: of two same-named ·potentially inherited·
// candidates, the one whose owner is the descendant of the other's wins. The
// second row is the discriminating one — a set yielding both would find the
// root's 'de' and select.
func TestNearerInheritableAttributeShadowsFartherOne(t *testing.T) {
	f := inhFixture{
		rootUses: []xsd.AttributeUse{inhUse(t, true, nil)},
		midUses:  []xsd.AttributeUse{inhUse(t, true, nil)},
	}
	inhWantSelected(t, cAssess(t, f.schema(t), inhDoc("fr", "de", "")), true, "<mid>'s 'de' over <root>'s 'fr'")
	inhWantSelected(t, cAssess(t, f.schema(t), inhDoc("de", "fr", "")), false, "<mid>'s 'fr' over <root>'s 'de'")
}

// Clause 2 compares against another candidate that is ITSELF ·potentially
// inherited·, so a nearer attribute that is not inheritable shadows nothing:
// <root>'s 'de' reaches <leaf> past <mid>'s non-inheritable 'fr'. The second
// row gives <mid> an inheritable region besides, so <mid>'s set is a merge in
// which its non-inheritable lang must still shadow nothing.
func TestNonInheritableAttributeShadowsNothing(t *testing.T) {
	f := inhFixture{
		rootUses: []xsd.AttributeUse{inhUse(t, true, nil)},
		midUses:  []xsd.AttributeUse{inhUse(t, false, nil), inhNamedUse(t, "region", true, nil)},
	}
	inhWantSelected(t, cAssess(t, f.schema(t), inhDoc("de", "fr", "")), true, "a non-inheritable 'fr' between")
	doc := inhDoc("de", "fr", "")
	mid, _ := doc.kids[0].Element()
	m := mid.(*testElement)
	m.attrs = append(m.attrs, &testAttribute{name: local("region"), value: "eu", loc: loc(2, 9)})
	inhWantSelected(t, cAssess(t, f.schema(t), doc), true, "a non-inheritable 'fr' beside an inheritable region")
}

// §3.12.4 clause 1.1.3 copies only the [inherited attributes] whose ·expanded
// names· none of E's own [[attributes]] has: <leaf>'s own lang wins outright.
// The first row is the discriminating one — a sequence yielding both would find
// the inherited 'de' and select.
func TestOwnAttributeWinsOverAnInheritedOne(t *testing.T) {
	f := inhFixture{rootUses: []xsd.AttributeUse{inhUse(t, true, nil)}}
	inhWantSelected(t, cAssess(t, f.schema(t), inhDoc("de", "", "fr")), false, "<leaf>'s own 'fr' over an inherited 'de'")
	inhWantSelected(t, cAssess(t, f.schema(t), inhDoc("fr", "", "de")), true, "<leaf>'s own 'de' over an inherited 'fr'")
}

// key-p-inherited clause 3.2: an attribute ·attributed to· no use is inheritable
// through its ·governing attribute declaration·'s {inheritable} — the top-level
// declaration a lax or strict {attribute wildcard} resolves it to. A skip
// wildcard leaves it ·skipped· and with no governing declaration at all
// (key-governing-ad clause 3), so it is not inheritable whatever the top-level
// declaration says.
func TestWildcardAttributeInheritsThroughItsDeclaration(t *testing.T) {
	for _, tc := range []struct {
		pc          xsd.ProcessContents
		inheritable bool
		want        bool
	}{
		{xsd.ProcessLax, true, true},
		{xsd.ProcessStrict, true, true},
		{xsd.ProcessLax, false, false},
		{xsd.ProcessSkip, true, false},
	} {
		f := inhFixture{
			rootWild: anyWildcard(t, tc.pc),
			global:   func(b *xsd.SchemaBuilder) { b.AddAttribute(inhGlobal(t, tc.inheritable)) },
		}
		inhWantSelected(t, cAssess(t, f.schema(t), inhDoc("de", "", "")), tc.want,
			tc.pc.String()+" wildcard, declaration {inheritable} "+strconv.FormatBool(tc.inheritable))
	}
}

// Clause 3.1 reads the USE's {inheritable}, and clause 3.2 the declaration's
// only for an attribute attributed to no use: a use referring to the top-level
// declaration decides by its own property in both directions.
func TestAttributeUseInheritableWinsOverItsDeclarations(t *testing.T) {
	for _, tc := range []struct {
		use, decl bool
	}{{use: false, decl: true}, {use: true, decl: false}} {
		f := inhFixture{
			rootUses: []xsd.AttributeUse{inhRefUse(t, &tc.use, nil)},
			global:   func(b *xsd.SchemaBuilder) { b.AddAttribute(inhGlobal(t, tc.decl)) },
		}
		inhWantSelected(t, cAssess(t, f.schema(t), inhDoc("de", "", "")), tc.use,
			"the use's {inheritable} against the declaration's")
	}
}

// A use referring to the top-level declaration with no inheritable attribute
// of its own has the declaration's {inheritable} (§3.2.2.3 ref.att.local), and
// clause 3.1 reads that: <root>'s lang reaches <leaf> exactly where the
// declaration is inheritable, for a specified lang (walk.inheritable) and for
// one the use supplies as a ·defaulted attribute· (walk.handedDown) (#1682).
func TestRefUseWithoutInheritableTakesItsDeclarations(t *testing.T) {
	de := xsd.NewValueConstraint(xsd.ValueDefault, "de", nil, nil)
	for _, tc := range []struct {
		decl bool
		vc   *xsd.ValueConstraint
		doc  *testElement
		why  string
	}{
		{true, nil, inhDoc("de", "", ""), "a specified lang, declaration {inheritable} true"},
		{false, nil, inhDoc("de", "", ""), "a specified lang, declaration {inheritable} false"},
		{true, &de, inhDoc("", "", ""), "a defaulted lang, declaration {inheritable} true"},
		{false, &de, inhDoc("", "", ""), "a defaulted lang, declaration {inheritable} false"},
	} {
		f := inhFixture{
			rootUses: []xsd.AttributeUse{inhRefUse(t, nil, tc.vc)},
			global:   func(b *xsd.SchemaBuilder) { b.AddAttribute(inhGlobal(t, tc.decl)) },
		}
		inhWantSelected(t, cAssess(t, f.schema(t), tc.doc), tc.decl, tc.why)
	}
}

// key-p-inherited counts an attribute "defaulted as described in Attribute
// Default Value (§3.4.5.1)" as much as a specified one: <root> carries no lang,
// and its inheritable use supplies 'de'.
func TestDefaultedAttributeIsInherited(t *testing.T) {
	de := xsd.NewValueConstraint(xsd.ValueDefault, "de", nil, nil)
	inhWantSelected(t, cAssess(t,
		inhFixture{rootUses: []xsd.AttributeUse{inhUse(t, true, &de)}}.schema(t),
		inhDoc("", "", "")), true, "a defaulted inheritable lang")
	inhWantSelected(t, cAssess(t,
		inhFixture{rootUses: []xsd.AttributeUse{inhUse(t, false, &de)}}.schema(t),
		inhDoc("", "", "")), false, "a defaulted non-inheritable lang")
}

// e-inherited_attributes gives the property to no child attributed to a skip
// Wildcard, and [walk.child] never walks one: under a lax wildcard <mid>
// resolves, is assessed, and hands <root>'s lang to <leaf>; under skip the same
// document charges nothing, <leaf> included.
func TestSkippedSubtreeInheritsNothing(t *testing.T) {
	for _, tc := range []struct {
		pc   xsd.ProcessContents
		want bool
	}{{xsd.ProcessLax, true}, {xsd.ProcessSkip, false}} {
		wild := dWildcard(t, tc.pc)
		f := inhFixture{
			rootUses:    []xsd.AttributeUse{inhUse(t, true, nil)},
			midParticle: &wild,
			global: func(b *xsd.SchemaBuilder) {
				b.AddElement(dTopLevel(t, "mid", "MidType"))
			},
		}
		got := cAssess(t, f.schema(t), inhDoc("de", "", ""))
		inhWantSelected(t, got, tc.want, "<mid> under a "+tc.pc.String()+" wildcard")
		if !tc.want {
			wantSilence(t, got, "a ·skipped· subtree is not assessed")
		}
	}
}

// nilFixture is a schema whose validation root <p> is {nillable}, typed PType,
// with a {type table} whose one alternative and {default type definition} both
// name PType, under the {test} test: "count(@wid) > 0" is one the §3.12.6
// evaluator declines (key-cta-ta-select), leaving <p>'s ·governing type
// definition· undetermined, and "@lang = 'de'" one it decides, the control.
// PType's one attribute use refers to the top-level lang with the use's own
// {inheritable} use, the top-level declaration's being decl. PType's sequence
// holds a lax wildcard, so a <leaf> child of a ·nilled· <p> has no ·locally
// declared type· within PType and ·resolves· to the tabled top-level <leaf>
// (inhLeaf); localLeaf gives it a LOCAL <leaf> typed Fallback instead, which is
// that ·locally declared type· (key-ldt-elem).
func nilFixture(t *testing.T, test string, use, decl, localLeaf bool) *xsd.Schema {
	t.Helper()
	leaf := dWildcard(t, xsd.ProcessLax)
	if localLeaf {
		leaf = dTyped(t, "PType", "leaf", "Fallback")
	}
	x := xsd.NewXPathExpression(test, nil, nil, nil)
	table, err := xsd.NewTypeTable(xsderr.Loc{},
		[]xsd.TypeAlternative{namedTypeAlternative(t, &x, local("PType"))},
		namedTypeAlternative(t, nil, local("PType")))
	if err != nil {
		t.Fatalf("building the type table: %v", err)
	}
	p, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("p"),
		xsd.TypeDefinitionRef{Name: local("PType")}, &table, xsd.NewGlobalScope(),
		nil, true, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the p element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	for _, st := range ctaBuiltins(t) {
		b.AddType(st)
	}
	b.AddType(ctaFallbackType(t))
	b.AddType(ctaCandidateType(t, "First"))
	b.AddType(dType(t, "PType", "", xsd.DerivationRestriction,
		[]xsd.AttributeUse{inhRefUse(t, &use, nil)}, cSequence(t, false, leaf)))
	b.AddElement(inhLeaf(t))
	b.AddElement(p)
	b.AddAttribute(inhGlobal(t, decl))
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the nilled-parent schema: %v", err)
	}
	return schema
}

// nilDoc is <p lang="de" xsi:nil="true"> over kids.
func nilDoc(kids ...Child) *testElement {
	p := dElem("p", 1, kids...)
	p.attrs = []Attribute{
		&testAttribute{name: local("lang"), value: "de", loc: loc(1, 5)},
		&testAttribute{name: xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, value: "true", loc: loc(1, 15)},
	}
	return p
}

// A ·nilled· <p> whose ·governing type definition· this package could not
// determine walks its <leaf> undecided (contentCheck.element, #2211): cvc-elt
// clause 3.2.3.1 is charged at the child as for any ·nilled· parent, and
// <p>'s key-cta-ta-select and cvc-id [Unevaluated] records stand for the type
// it declined. The control rows decide <p>'s {test} and show what each fixture
// discriminates; each declined row fails with contentCheck.element reporting
// the child decided, as it did before #2211.
//
//   - Inheritable by the declaration only: the true type's use hands lang down
//     to nobody, but a read by ·expanded name· would, and <leaf> would select
//     First and be charged needFirst at line 3 — an over-reject.
//   - Inheritable by the use only: the true type hands lang down and <leaf>
//     selects First, but a read by ·expanded name· would not, and <leaf> would
//     be governed by its Fallback default instead — the lost charge. Fallback,
//     like First, admits no element [[child]], so <leaf>'s <x> is charged
//     cvc-complex-type clause 1.1 under either, and its absence is what shows
//     <leaf> was governed by neither.
//   - A local <leaf>: the true type governs it by its ·locally declared type·
//     Fallback (key-governing-ed clause 4.3), which has no {type table}; read
//     without the parent's type it would resolve to the tabled top-level <leaf>
//     and select First on an inherited lang.
//   - A text [[child]] before <leaf>: the parent is already charged when
//     <leaf> arrives, the second arm contentCheck.element reports through.
func TestNilledUndeterminedParentsChildIsUndecided(t *testing.T) {
	const declined, decided = "count(@wid) > 0", "@lang = 'de'"
	leaf := func(kids ...Child) Child { return ElementChild(dElem("leaf", 3, kids...)) }
	x := ElementChild(dElem("x", 4))
	text := TextChild(&testText{data: "t", loc: loc(2, 1)})
	type charge = struct {
		rule xsderr.Rule
		loc  xsderr.Loc
	}
	nilAt := func(line int) charge { return icCharge(ruleCvcElt, line) }
	for _, tc := range []struct {
		why                  string
		test                 string
		use, decl, localLeaf bool
		doc                  *testElement
		want                 []charge
	}{
		{"declaration only, decided", decided, false, true, false, nilDoc(leaf()),
			[]charge{nilAt(3)}},
		{"declaration only, declined", declined, false, true, false, nilDoc(leaf()),
			[]charge{nilAt(3)}},
		{"use only, decided", decided, true, false, false, nilDoc(leaf(x)),
			[]charge{nilAt(3), icCharge(ruleCvcComplexType, 3), icCharge(ruleCvcComplexType, 4)}},
		{"use only, declined", declined, true, false, false, nilDoc(leaf(x)),
			[]charge{nilAt(3)}},
		{"local leaf, decided", decided, true, true, true, nilDoc(leaf()),
			[]charge{nilAt(3)}},
		{"local leaf, declined", declined, true, true, true, nilDoc(leaf()),
			[]charge{nilAt(3)}},
		{"charged before the child, declined", declined, false, true, false, nilDoc(text, leaf()),
			[]charge{nilAt(2)}},
	} {
		t.Run(tc.why, func(t *testing.T) {
			got, unevaluated := assessRecorded(t, nilFixture(t, tc.test, tc.use, tc.decl, tc.localLeaf), tc.doc)
			icWantCharges(t, got, tc.want...)
			if tc.test == decided {
				return
			}
			wantDeclines(t, unevaluated,
				Unevaluated{rule: ruleKeyCTATASelect, loc: loc(1, 1), msg: "count(@wid) > 0"},
				Unevaluated{rule: ruleCvcID, loc: loc(1, 1), msg: ""})
		})
	}
}
