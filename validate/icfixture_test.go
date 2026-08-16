package validate

import (
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The fixtures below build the two schema shapes cvcidentityconstraint_test.go
// and cvcid_test.go both drive. icSchema is the main one, with the identity
// constraints of <root> and of <box> as its only parameters:
//
//	root    RootType  sequence( item*, box*, ref*, tabled* )
//	box     BoxType   sequence( item*, ref* )          — a subtree of its own
//	item    ItemType  sequence( name?, tag?, utag?, tabled? ),
//	                  @id @k @p:id @xid @ref @refs,
//	                  @uid @usi @uref @upl @unest @lur  — icUnionTypes
//	ref     RefType   empty,                           @r
//	name    xs:string (or a nillable declaration of it)
//	tag     xs:ID     an ID-governed ELEMENT
//	utag    UID       a union-governed one
//	tabled  a declaration carrying a {type table}, so its ·governing type
//	        definition· is the one shape this package cannot determine at all
//
// Every complex type is NAMED, so its {attribute uses} are folded and an
// attribute is matched rather than declined (assess.go's
// attributePropertiesFolded), and every simple type is the SEEDED builtin, so
// a value comparison runs the real facets.
//
// icNS is the one namespace in play besides the absent one; item, box and ref
// are declared in it or not per the ns argument, and @p:id is in it always, so
// one shape can pin the element-step/attribute-step asymmetry of
// xpathDefaultNamespace.
//
// icUnfoldedSchema is the second shape, built precisely because the first one
// names every type: it holds an UNFOLDED governing type, over top-level
// attribute declarations colliding with that type's own uses.

const icNS = "urn:p"

// icSeeded returns the seeded builtin simple types, indexed by local name.
func icSeeded(t *testing.T) map[string]*xsd.SimpleType {
	t.Helper()
	seeded, err := builtin.Seed(testBackend())
	if err != nil {
		t.Fatalf("seeding the builtin types: %v", err)
	}
	byName := make(map[string]*xsd.SimpleType, len(seeded))
	for _, st := range seeded {
		byName[st.Name().Local] = st
	}
	return byName
}

// icBuiltin is the ·expanded name· of one builtin simple type.
func icBuiltin(local string) xsd.QName {
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: local}
}

// icUse builds an optional attribute use over a declaration of the named
// builtin type.
func icUse(t *testing.T, name xsd.QName, typ string) xsd.AttributeUse {
	t.Helper()
	return icUseOf(t, name, icBuiltin(typ))
}

// icUseOf is icUse over any type name, which is what a use of one of the
// fixture's own union and list types needs.
func icUseOf(t *testing.T, name, typ xsd.QName) xsd.AttributeUse {
	t.Helper()
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name,
		xsd.TypeDefinitionRef{Name: typ}, xsd.NewAttributeGlobalScope(), nil, false, nil)
	if err != nil {
		t.Fatalf("building the %s attribute declaration: %v", name, err)
	}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.LocalAttributeDeclaration{Declaration: d}, nil, false, nil)
	if err != nil {
		t.Fatalf("building the %s attribute use: %v", name, err)
	}
	return u
}

// icUnion builds a NAMED union simple type over by-name members, in declared
// order — the order dt-active-member selects in. Its {base type definition} is
// xs:anySimpleType, which is what §3.16.2.1 map.std.common case 2 gives a
// <union>.
func icUnion(t *testing.T, local string, members ...xsd.QName) *xsd.SimpleType {
	t.Helper()
	slots := make([]xsd.SimpleTypeOrRef, 0, len(members))
	for _, m := range members {
		slots = append(slots, xsd.SimpleTypeRef{Name: m})
	}
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Local: local},
		xsd.UnionDerivation{Members: slots},
		xsd.SimpleTypeRef{Name: icBuiltin("anySimpleType")}, nil, nil)
	if err != nil {
		t.Fatalf("building the %s union: %v", local, err)
	}
	return st
}

// icList builds a NAMED list simple type over a by-name {item type definition},
// on icUnion's terms. The fixed collapse whiteSpace is the {facets} set
// §3.16.2.1 manufactures for every <list>, and the only one cos-st-restricts
// clause 2.2.1.2 admits on a list whose base is xs:anySimpleType.
func icList(t *testing.T, local string, item xsd.QName) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Local: local},
		xsd.ListDerivation{Item: xsd.SimpleTypeRef{Name: item}},
		xsd.SimpleTypeRef{Name: icBuiltin("anySimpleType")},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("building the %s list: %v", local, err)
	}
	return st
}

// icLocal builds a local element declaration of the named type, carrying ics
// and nillable.
func icLocal(t *testing.T, parent string, name xsd.QName, typ xsd.QName, nillable bool, ics []xsd.IdentityConstraint) xsd.ElementDeclaration {
	t.Helper()
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: xsd.QName{Local: parent}})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: typ}, nil, scope,
		nil, nillable, ics, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the %s element declaration: %v", name, err)
	}
	return d
}

// icTabled builds a local element declaration carrying a {type table} whose
// one <alternative> has a {test} OUTSIDE the §3.12.6 required subset, so the
// engine cannot decide which type the table ·conditionally selects· and
// determines no ·governing type definition· for such an element at all — the
// one shape left that types an element against nothing. A {test} the engine
// CAN evaluate would type it, which is what validate/cta_test.go drives.
func icTabled(t *testing.T, parent string, name xsd.QName) xsd.ElementDeclaration {
	t.Helper()
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: xsd.QName{Local: parent}})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	test := xsd.NewXPathExpression("count(@k) > 0", nil, nil, nil)
	alt := xsd.NewTypeAlternative(&test, icBuiltin("string"), nil)
	table, err := xsd.NewTypeTable(xsderr.Loc{}, []xsd.TypeAlternative{alt},
		xsd.NewTypeAlternative(nil, icBuiltin("string"), nil))
	if err != nil {
		t.Fatalf("building the type table: %v", err)
	}
	d, err := xsd.NewElementDeclaration(xsderr.Loc{}, name,
		xsd.TypeDefinitionRef{Name: icBuiltin("string")}, &table, scope,
		nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the %s element declaration: %v", name, err)
	}
	return d
}

// icRepeated wraps a declaration in a 0..unbounded particle.
func icRepeated(t *testing.T, d xsd.ElementDeclaration) xsd.Particle {
	t.Helper()
	o, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, 0)
	if err != nil {
		t.Fatalf("NewUnboundedOccurs: %v", err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, o, xsd.ResolvedTerm{Term: d}, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

// icOptional wraps a declaration in a 0..1 particle.
func icOptional(t *testing.T, d xsd.ElementDeclaration) xsd.Particle {
	t.Helper()
	o, err := xsd.NewOccurs(xsderr.Loc{}, 0, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, o, xsd.ResolvedTerm{Term: d}, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return p
}

// icContent is an element-only {content type} over a <sequence>.
func icContent(t *testing.T, particles ...xsd.Particle) xsd.ContentType {
	t.Helper()
	g, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, particles, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	o, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		t.Fatalf("NewOccurs: %v", err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, o, xsd.ResolvedTerm{Term: g}, nil)
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return xsd.ElementContent{Particle: p}
}

// icComplex builds a NAMED complex type.
func icComplex(t *testing.T, name string, uses []xsd.AttributeUse, content xsd.ContentType) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: name}, xsd.QName{}, nil,
		xsd.DerivationRestriction, false, uses, nil, nil, content, nil, nil, nil)
	if err != nil {
		t.Fatalf("building %s: %v", name, err)
	}
	return ct
}

// icDef builds one identity-constraint definition. selector and fields are the
// {expression}s; def is the {default namespace} the element steps resolve an
// unprefixed NameTest under, absent when nil; refer names the {referenced key}
// of a keyref and is unread otherwise. The prefix "p" is bound to [icNS] on
// every expression, so a fixture can spell either half of PRINCIPLES 15.
func icDef(t *testing.T, name string, cat xsd.IdentityConstraintCategory, selector string, def *string, refer string, fields ...string) xsd.IdentityConstraint {
	t.Helper()
	bindings := []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", icNS)}
	exprs := make([]xsd.XPathExpression, 0, len(fields))
	for _, f := range fields {
		exprs = append(exprs, xsd.NewXPathExpression(f, bindings, def, nil))
	}
	var key *xsd.QName
	if cat == xsd.IdentityConstraintKeyref {
		key = &xsd.QName{Local: refer}
	}
	ic, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: name}, cat,
		xsd.NewXPathExpression(selector, bindings, def, nil), exprs, key, nil)
	if err != nil {
		t.Fatalf("building the %s identity constraint: %v", name, err)
	}
	return ic
}

// icUnionTypes builds the ·constructed· types <item> carries, the shapes
// key-vtype has to resolve per value rather than per declaration:
//
//	UID     union( ID,      string )   an ID member, reached first
//	USI     union( string,  ID     )   the same members in the other order
//	URef    union( IDREF,   string )   an IDREF member
//	UPlain  union( integer, string )   neither, so nothing to record
//	UNest   union( integer, UID    )   a union member of a union
//	LURef   list of URef              key-vtype clause 2, per item
//
// The pairing with xs:string is what makes the member scan observable: every
// literal is valid against the union whichever member takes it, so a difference
// in the [ID/IDREF table] can only come from WHICH member did.
func icUnionTypes(t *testing.T) []*xsd.SimpleType {
	t.Helper()
	named := func(local string) xsd.QName { return xsd.QName{Local: local} }
	return []*xsd.SimpleType{
		icUnion(t, "UID", icBuiltin("ID"), icBuiltin("string")),
		icUnion(t, "USI", icBuiltin("string"), icBuiltin("ID")),
		icUnion(t, "URef", icBuiltin("IDREF"), icBuiltin("string")),
		icUnion(t, "UPlain", icBuiltin("integer"), icBuiltin("string")),
		icUnion(t, "UNest", icBuiltin("integer"), named("UID")),
		icList(t, "LURef", named("URef")),
	}
}

// icSchema assembles the fixture schema. ns is the namespace item, box and ref
// are declared in — "" for the absent one — and nillable makes <name>'s
// declaration {nillable} true, which is the only property clause 4.2.3 reads.
func icSchema(t *testing.T, ns string, nillable bool, rootICs, boxICs []xsd.IdentityConstraint) *xsd.Schema {
	t.Helper()
	seeded := icSeeded(t)
	in := func(local string) xsd.QName { return xsd.QName{Space: ns, Local: local} }

	unions := icUnionTypes(t)
	named := func(local string) xsd.QName { return xsd.QName{Local: local} }

	nameDecl := icLocal(t, "ItemType", in("name"), icBuiltin("string"), nillable, nil)
	tagDecl := icLocal(t, "ItemType", in("tag"), icBuiltin("ID"), false, nil)
	utagDecl := icLocal(t, "ItemType", in("utag"), named("UID"), false, nil)
	itemType := icComplex(t, "ItemType", []xsd.AttributeUse{
		icUse(t, xsd.QName{Local: "id"}, "string"),
		icUse(t, xsd.QName{Local: "k"}, "integer"),
		icUse(t, xsd.QName{Space: icNS, Local: "id"}, "string"),
		icUse(t, xsd.QName{Local: "xid"}, "ID"),
		icUse(t, xsd.QName{Local: "ref"}, "IDREF"),
		icUse(t, xsd.QName{Local: "refs"}, "IDREFS"),
		icUseOf(t, named("uid"), named("UID")),
		icUseOf(t, named("usi"), named("USI")),
		icUseOf(t, named("uref"), named("URef")),
		icUseOf(t, named("upl"), named("UPlain")),
		icUseOf(t, named("unest"), named("UNest")),
		icUseOf(t, named("lur"), named("LURef")),
	}, icContent(t, icOptional(t, nameDecl), icOptional(t, tagDecl), icOptional(t, utagDecl),
		icOptional(t, icTabled(t, "ItemType", in("tabled")))))
	refType := icComplex(t, "RefType", []xsd.AttributeUse{
		icUse(t, xsd.QName{Local: "r"}, "string"),
	}, xsd.EmptyContent{})

	boxItem := icLocal(t, "BoxType", in("item"), xsd.QName{Local: "ItemType"}, false, nil)
	boxRef := icLocal(t, "BoxType", in("ref"), xsd.QName{Local: "RefType"}, false, nil)
	boxType := icComplex(t, "BoxType", nil, icContent(t, icRepeated(t, boxItem), icRepeated(t, boxRef)))

	rootItem := icLocal(t, "RootType", in("item"), xsd.QName{Local: "ItemType"}, false, nil)
	rootBox := icLocal(t, "RootType", in("box"), xsd.QName{Local: "BoxType"}, false, boxICs)
	rootRef := icLocal(t, "RootType", in("ref"), xsd.QName{Local: "RefType"}, false, nil)
	rootType := icComplex(t, "RootType", nil,
		icContent(t, icRepeated(t, rootItem), icRepeated(t, rootBox), icRepeated(t, rootRef),
			icRepeated(t, icTabled(t, "RootType", in("tabled")))))

	root, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, rootICs, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}

	b := xsd.NewSchemaBuilder()
	for _, st := range seeded {
		b.AddType(st)
	}
	for _, st := range unions {
		b.AddType(st)
	}
	b.AddType(itemType)
	b.AddType(refType)
	b.AddType(boxType)
	b.AddType(rootType)
	b.AddElement(root)
	for _, ic := range rootICs {
		b.AddIdentityConstraint(ic)
	}
	for _, ic := range boxICs {
		b.AddIdentityConstraint(ic)
	}
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the identity-constraint schema: %v", err)
	}
	return schema
}

// icTopAttribute builds a TOP-LEVEL attribute declaration of the named
// builtin type, which is what an ·expanded name· ·resolves· to under §3.10.4.1.
func icTopAttribute(t *testing.T, local, typ string) xsd.AttributeDeclaration {
	t.Helper()
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: local},
		xsd.TypeDefinitionRef{Name: icBuiltin(typ)}, xsd.NewAttributeGlobalScope(), nil, false, nil)
	if err != nil {
		t.Fatalf("building the top-level %s attribute declaration: %v", local, err)
	}
	return d
}

// icUnfoldedSchema builds the shape icSchema's every-type-named convention
// cannot reach: <kid> OWNS an ANONYMOUS complex type restricting a named base,
// so no finalize pass folds its {attribute uses} (#414,
// attributePropertiesFolded), and the schema ALSO declares top-level attributes
// of the same ·expanded names· carrying OTHER types.
//
//	root  RootType (named)              sequence( kid* )
//	kid   (anonymous, restricts Base)   empty, @aid <kidAid>, @ref xs:IDREF
//	Base  (named)                       empty
//	top-level                           @aid <topAid>, @ref xs:IDREF
//
// kidAid is <kid>'s own governing type for @aid and topAid is the colliding
// top-level one, so a fixture spells both halves of the misreading a top-level
// lookup would be: the two differ in exactly what the rule under test reads.
func icUnfoldedSchema(t *testing.T, kidAid, topAid string, rootICs []xsd.IdentityConstraint) *xsd.Schema {
	t.Helper()
	seeded := icSeeded(t)

	kidID := xsd.NewComponentID()
	kidType, err := xsd.NewAnonymousComplexType(xsderr.Loc{},
		xsd.ElementDeclarationContext{Component: kidID}, xsd.QName{Local: "Base"}, nil,
		xsd.DerivationRestriction, false, []xsd.AttributeUse{
			icUse(t, xsd.QName{Local: "aid"}, kidAid),
			icUse(t, xsd.QName{Local: "ref"}, "IDREF"),
		}, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("building the anonymous kid type: %v", err)
	}
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: xsd.QName{Local: "RootType"}})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	kid, err := xsd.NewElementDeclarationOwningType(xsderr.Loc{}, kidID, xsd.QName{Local: "kid"}, kidType,
		nil, scope, nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the kid element declaration: %v", err)
	}
	rootType := icComplex(t, "RootType", nil, icContent(t, icRepeated(t, kid)))
	root, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, rootICs, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}

	b := xsd.NewSchemaBuilder()
	for _, st := range seeded {
		b.AddType(st)
	}
	b.AddType(icComplex(t, "Base", nil, xsd.EmptyContent{}))
	b.AddType(rootType)
	b.AddElement(root)
	b.AddAttribute(icTopAttribute(t, "aid", topAid))
	b.AddAttribute(icTopAttribute(t, "ref", "IDREF"))
	for _, ic := range rootICs {
		b.AddIdentityConstraint(ic)
	}
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the unfolded-type schema: %v", err)
	}
	return schema
}

// icKid is <kid> at line, carrying the named no-namespace attributes, in
// icItem's own curried shape: names first, values applied.
func icKid(line int, names ...string) func(...string) *testElement {
	return func(values ...string) *testElement {
		attrs := make([]Attribute, 0, len(names))
		for i, n := range names {
			attrs = append(attrs, icAttr(xsd.QName{Local: n}, values[i], line))
		}
		return icElem(xsd.QName{Local: "kid"}, line, attrs)
	}
}

// icAssess assesses root against schema and returns the violations charged.
func icAssess(t *testing.T, schema *xsd.Schema, root Element) []*xsderr.Error {
	t.Helper()
	v, err := New(schema, testBackend())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res.Violations()
}

// icAttr is one attribute information item.
func icAttr(name xsd.QName, value string, line int) Attribute {
	return &testAttribute{name: name, value: value, loc: loc(line, 2)}
}

// icElem is one element information item at line, with its own attributes and
// children.
func icElem(name xsd.QName, line int, attrs []Attribute, kids ...Child) *testElement {
	return &testElement{name: name, attrs: attrs, kids: kids, loc: loc(line, 1)}
}

// icKids wraps elements as [[children]].
func icKids(elems ...Element) []Child {
	kids := make([]Child, 0, len(elems))
	for _, e := range elems {
		kids = append(kids, ElementChild(e))
	}
	return kids
}

// icWantCharges fails unless the violations charged are exactly the given
// (rule, Loc) pairs, in order.
func icWantCharges(t *testing.T, got []*xsderr.Error, want ...struct {
	rule xsderr.Rule
	loc  xsderr.Loc
},
) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Violations() = %v, want %d charge(s)", got, len(want))
	}
	for i, w := range want {
		if got[i].Rule != w.rule || got[i].Loc != w.loc {
			t.Errorf("violation %d = (%s at %s), want (%s at %s): %v",
				i, got[i].Rule, got[i].Loc, w.rule, w.loc, got[i])
		}
	}
}

// icCharge names one expected violation for icWantCharges, at the Loc icElem
// gives an element.
func icCharge(rule xsderr.Rule, line int) struct {
	rule xsderr.Rule
	loc  xsderr.Loc
} {
	return icChargeAt(rule, loc(line, 1))
}

// icChargeAttr names one expected violation at the Loc icAttr gives an
// attribute, which is where a charge against an ATTRIBUTE field node lands.
func icChargeAttr(rule xsderr.Rule, line int) struct {
	rule xsderr.Rule
	loc  xsderr.Loc
} {
	return icChargeAt(rule, loc(line, 2))
}

// icChargeAt names one expected violation at an arbitrary Loc.
func icChargeAt(rule xsderr.Rule, at xsderr.Loc) struct {
	rule xsderr.Rule
	loc  xsderr.Loc
} {
	return struct {
		rule xsderr.Rule
		loc  xsderr.Loc
	}{rule: rule, loc: at}
}
