package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These fixtures drive cvc-id (§3.3.4.5) over the [ID/IDREF table] of
// §3.17.5.2, assembled across the schema shape icfixture_test.go builds: @xid
// is xs:ID, @ref is xs:IDREF, @refs is xs:IDREFS, and <tag> is an ID-governed
// ELEMENT, whose value binds its PARENT.

// idSchema is the fixture schema with no identity constraints on it: cvc-id
// needs none.
func idSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	return icSchema(t, "", false, nil, nil)
}

// idItem is <item> carrying one attribute of the given local name.
func idItem(line int, attr, value string) *testElement {
	return icElem(xsd.QName{Local: "item"}, line,
		[]Attribute{icAttr(xsd.QName{Local: attr}, value, line)})
}

// idTagged is <item><tag>value</tag></item>, whose ID-governed element content
// binds the ITEM.
func idTagged(line int, value string) *testElement {
	tag := icElem(xsd.QName{Local: "tag"}, line+1, nil,
		TextChild(&testText{data: value, loc: loc(line+1, 8)}))
	return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(tag))
}

// idUTagged is <item><utag>value</utag></item>, whose UNION-governed element
// content binds the ITEM.
func idUTagged(line int, value string) *testElement {
	utag := icElem(xsd.QName{Local: "utag"}, line+1, nil,
		TextChild(&testText{data: value, loc: loc(line+1, 8)}))
	return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(utag))
}

// An id two elements both claim gives its ID/IDREF binding two members, which
// cvc-id clause 2 forbids. The charge cites the SECOND claim.
func TestDuplicateIDIsChargedAtTheValidationRoot(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idItem(3, "xid", "a"))),
		icChargeAttr(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idItem(3, "xid", "b"))))
}

// An IDREF naming an id nothing declares gives its binding an empty set, which
// cvc-id clause 1 forbids. The charge cites where the string was first met.
func TestUnresolvedIDREFIsChargedAtTheValidationRoot(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "ref", "ghost"))),
		icChargeAttr(ruleCvcID, 2))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idItem(3, "ref", "a"))))
}

// An IDREFS value holds one ·IDREF value· per list item, so each is looked up
// on its own.
func TestIDREFSReferencesEachListItem(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idItem(3, "refs", "a b"))),
		icChargeAttr(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, schema,
		icRoot(idItem(2, "xid", "a"), idItem(3, "xid", "b"), idItem(4, "refs", "a b"))))
}

// An ID-governed ELEMENT binds its PARENT, not itself (§3.17.5.2's [binding]),
// so an element whose content is an id and an attribute of another element with
// the same id are two members of one binding.
func TestAnIDGovernedElementBindsItsParent(t *testing.T) {
	schema := idSchema(t)

	// The charge cites where the second claim was WRITTEN — the <tag> element
	// itself — while the binding member it added is that element's parent.
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idTagged(3, "a"))),
		icChargeAt(ruleCvcID, loc(4, 1)))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idTagged(3, "b"))))
	// One element claiming an id twice is still one member of the binding.
	both := icElem(xsd.QName{Local: "item"}, 2,
		[]Attribute{icAttr(xsd.QName{Local: "xid"}, "a", 2)},
		ElementChild(icElem(xsd.QName{Local: "tag"}, 3, nil,
			TextChild(&testText{data: "a", loc: loc(3, 8)}))))
	icWantCharges(t, icAssess(t, schema, icRoot(both)))
}

// cvc-id fires at the ·validation root· and nowhere else (cvc-elt clause 7):
// a duplicate two levels down is charged ONCE, by the root, and not again by
// every ancestor whose subtree also holds it.
func TestCvcIDIsChargedOnceHoweverDeepTheDuplicateIs(t *testing.T) {
	nested := icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(icElem(xsd.QName{Local: "box"}, 2, nil,
			ElementChild(idItem(3, "xid", "a")),
			ElementChild(idItem(4, "xid", "a")))))
	icWantCharges(t, icAssess(t, idSchema(t), nested), icChargeAttr(ruleCvcID, 4))
}

// Assessing a SUBTREE as its own validation root reads only that subtree's
// table, which is what makes an interior element's ids invisible to a sibling's
// assessment — the same subtree bound the §3.11.5 node tables to.
func TestCvcIDReadsOnlyTheValidationRootsOwnSubtree(t *testing.T) {
	// The reference and the declaration are in different boxes, so a
	// whole-document assessment resolves it and a box-rooted one does not.
	// <box> is not a top-level declaration, so assessing one directly is
	// cvc-assess-elt's business; the reference is charged against <root>.
	doc := icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(icElem(xsd.QName{Local: "box"}, 2, nil, ElementChild(idItem(3, "xid", "a")))),
		ElementChild(icElem(xsd.QName{Local: "box"}, 4, nil, ElementChild(idItem(5, "ref", "a")))))
	icWantCharges(t, icAssess(t, idSchema(t), doc))
}

// An item of the subtree this package could not type could have been the
// DECLARATION an IDREF names, so clause 1 declines for the whole document once
// one appears. Clause 2 keeps charging: an unread item can only ADD members to
// a binding, never take one away.
func TestAnUntypedItemDeclinesClauseOneAndNotClauseTwo(t *testing.T) {
	schema := idSchema(t)
	// <tabled> carries a {type table}, so its ·governing type definition· is not
	// determinable and anything beneath it is unread. It comes LAST, the
	// {content type} being a sequence.
	untyped := icElem(xsd.QName{Local: "tabled"}, 5, nil)

	// Without the untyped item, the dangling reference is charged.
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(3, "ref", "ghost"))),
		icChargeAttr(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(3, "ref", "ghost"), untyped)))

	// The duplicate is charged either way.
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(3, "xid", "a"), idItem(4, "xid", "a"), untyped)),
		icChargeAttr(ruleCvcID, 4))
}

// An attribute of an element whose ·governing type definition· was not
// determined binds no ·ID value·: key-governing-ad (§3.2.4.2) clause 2's
// ·context-determined declaration·, from a use of that type, comes before
// clause 3's resolution by ·expanded name·, so the top-level xs:ID declaration
// of @wid is not the attribute's, and the true one — T's xs:string use, which
// every alternative of <e>'s undecided {type table} names — makes the two
// "a"s no duplicate. cvc-id clause 2 is therefore not charged, and clause 1
// stays withheld by <e>'s own record (#2192).
//
// The children of <e> are the UNDECIDED shape ([governance]), whose governance
// this package could not decide at all, and their @wid binds nothing either:
// whatever attributes them, a use or a skip {attribute wildcard} of their true
// type may come before the top-level declaration.
func TestAnUndeterminedTypesAttributeBindsNoIDByItsExpandedName(t *testing.T) {
	got, undecided := assessRecorded(t, icTabledAttrSchema(t, nil), icRoot(icTabledE(2, "a"), icTabledE(3, "a")))
	wantSilence(t, got, "@wid is xs:string under T whichever alternative selects")
	var ids []Unevaluated
	for _, u := range undecided {
		if u.Rule() == ruleCvcID {
			ids = append(ids, u)
		}
	}
	wantDeclines(t, ids,
		Unevaluated{rule: ruleCvcID, loc: loc(2, 1), msg: "cvc-id clause 1 is undecided"},
		Unevaluated{rule: ruleCvcID, loc: loc(3, 1), msg: "cvc-id clause 1 is undecided"})

	kid := func(line int) *testElement {
		return icElem(xsd.QName{Local: "kid"}, line, []Attribute{icAttr(xsd.QName{Local: "wid"}, "a", line)})
	}
	parent := icElem(xsd.QName{Local: "e"}, 2, nil, ElementChild(kid(3)), ElementChild(kid(4)))
	wantSilence(t, icAssess(t, icTabledAttrSchema(t, nil), icRoot(parent)),
		"an undecided <kid>'s @wid is not known to be the top-level xs:ID")
}

// An attribute of an ANONYMOUS governing type is read off THAT type's own
// {attribute uses}, not off a top-level declaration of the same ·expanded
// name·: the two are different components, and only the use's {attribute
// declaration} carries the ·governing type definition· §3.17.5.2 clause 3 asks
// for (walk.attributeType). The <kid> type's uses are read because they are the
// spec's — §3.4.2.4 clause 3 folds a declaration-owned type through its owning
// slot (#414) — and the top-level declarations here differ in type from them,
// so a lookup that reached one would misclassify the value.
func TestAnonymousTypeAttributeReadsItsOwnUseOverACollidingTopLevelType(t *testing.T) {
	// @aid is xs:ID where <kid>'s own type governs it and xs:string at the
	// top level, so the top-level reading declares no id at all.
	schema := icAnonymousSchema(t, "ID", "string", nil)

	// The id IS declared, by <kid>'s own xs:ID use, so the reference to it
	// binds: reading @aid as the top-level xs:string would leave the binding
	// empty and charge clause 1 here.
	icWantCharges(t, icAssess(t, schema, icRoot(icKid(2, "aid", "ref")("x1", "x1"))))
	// The reference is read through the same type's own xs:IDREF use, so a
	// dangling one leaves an empty binding, which clause 1 charges.
	icWantCharges(t, icAssess(t, schema, icRoot(icKid(2, "ref")("ghost"))),
		icChargeAttr(ruleCvcID, 2))
}

// The same reading keeps clause 2 from charging a duplicate the document does
// not have: <kid>'s own use makes @aid xs:string, and a value no ·validating
// type· reads as an ·ID value· puts no member in a binding at all, so the two
// items below collide only under the type the schema declares at the TOP level
// and never under the one that governs them.
func TestAnonymousTypeAttributeReadsItsOwnUseRatherThanManufacturingADuplicate(t *testing.T) {
	// @aid is xs:string where <kid>'s own type governs it and xs:ID at the
	// top level, so the top-level reading declares one id twice.
	schema := icAnonymousSchema(t, "string", "ID", nil)

	icWantCharges(t, icAssess(t, schema,
		icRoot(icKid(2, "aid")("x1"), icKid(3, "aid")("x1"))))
}

// A union is ·constructed· from its members, so clause 3 of the ·eligible item
// set· admits an item it governs; WHICH of ID, IDREF or neither the value is
// then read as is its ·validating type·'s to say (§3.16.4 key-vtype clause 1),
// and @uid's first member takes "a".
func TestAUnionMemberValidatingAsIDDeclaresTheID(t *testing.T) {
	schema := idSchema(t)

	// The id is declared through the union member, so the reference resolves.
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "uid", "a"), idItem(3, "ref", "a"))))
	// @upl is a union of integer and string: nothing it validates is an ·ID
	// value·, so the same reference has an empty binding (clause 1).
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "upl", "a"), idItem(3, "ref", "a"))),
		icChargeAttr(ruleCvcID, 3))
}

// The IDREF half: a value the union's IDREF member validated is an ·IDREF
// value·, so it puts its string in the table whether or not anything declares
// it.
func TestAUnionMemberValidatingAsIDREFReferences(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "uref", "ghost"))),
		icChargeAttr(ruleCvcID, 2))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idItem(3, "uref", "a"))))
}

// A union whose ·validating type· is neither ID nor IDREF contributes NOTHING —
// not an entry whose binding is empty, which clause 1 would charge.
func TestAUnionValidatingAsNeitherRecordsNothing(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "upl", "7"))))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "upl", "ghost"))))
}

// An id declared through a union member is a member of the SAME binding an
// xs:ID-governed attribute declares, so the two together are the duplicate
// clause 2 forbids.
func TestADuplicateThroughAUnionMemberIsChargedClauseTwo(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "uid", "a"), idItem(3, "xid", "a"))),
		icChargeAttr(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "uid", "a"), idItem(3, "uid", "a"))),
		icChargeAttr(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "uid", "a"), idItem(3, "uid", "b"))))
}

// The ·active member type· is "the first of its members in order which accepts
// the instance as valid" (dt-active-member), so @usi — the same two members as
// @uid in the other order — reads "a" as an xs:string and declares no id at all.
func TestTheFirstMemberInOrderDecidesTheValidatingType(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "usi", "a"), idItem(3, "ref", "a"))),
		icChargeAttr(ruleCvcID, 3))
}

// The descent to the ·active basic member· does not stop at a member that is
// itself a union (dt-active-basic-member): @unest's second member is @uid's
// union, whose own first member is ID.
func TestANestedUnionDescendsToTheBasicMember(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "unest", "a"), idItem(3, "ref", "a"))))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "unest", "a"), idItem(3, "xid", "a"))),
		icChargeAttr(ruleCvcID, 3))
}

// key-vtype clause 2 decides a list whose {item type definition} is a union PER
// ITEM: @lur is a list of union(IDREF, string), so "a" is an ·IDREF value· and
// "1" — no NCName, so no IDREF — is not one at all.
func TestAListOfUnionIsDecidedPerItem(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idItem(3, "lur", "a 1"))))
	// One charge and not two: "1" contributes no entry to charge for.
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "lur", "a 1"))),
		icChargeAttr(ruleCvcID, 2))
}

// The element half of the ·eligible item set· reads a union-governed ELEMENT
// the same way, whose ·initial value· binds its PARENT.
func TestAUnionGovernedElementDeclaresTheIDOfItsParent(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idUTagged(2, "a"), idItem(4, "ref", "a"))))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "a"), idUTagged(3, "a"))),
		icChargeAt(ruleCvcID, loc(4, 1)))
}

// idNilled is <item><name xsi:nil="..."/></item>, the shape whose <name> is
// ·nilled· or not according to the declaration icSchema's nillable argument
// builds and the ·actual value· given here.
func idNilled(line int, nil_ string) *testElement {
	name := icElem(xsd.QName{Local: "name"}, line+1,
		[]Attribute{icAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, nil_, line+1)})
	return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(name))
}

// A ·nilled· element is outside the ·eligible item set· and withholds nothing:
// §3.3.5.4 gives it an absent [schema actual value], and §3.17.5.2's clause 2
// excludes every such item. The dangling reference is charged under cvc-id
// clause 1 whether <name> is ·nilled· (key-nilled: D.{nillable} = true AND an
// ·actual value· of true), not ·nilled·, or carries an xsi:nil clause 3.1
// charges, and nothing is recorded as unevaluated in any of the three.
func TestANilledElementWithholdsNoIDCheck(t *testing.T) {
	ghost := icRoot(idItem(2, "ref", "ghost"), idNilled(3, "false"))
	icWantCharges(t, icAssess(t, icSchema(t, "", true, nil, nil), ghost),
		icChargeAttr(ruleCvcID, 2))

	nilled := icRoot(idItem(2, "ref", "ghost"), idNilled(3, "true"))
	icWantCharges(t, icAssess(t, icSchema(t, "", true, nil, nil), nilled),
		icChargeAttr(ruleCvcID, 2))
	_, undecided := assessRecorded(t, icSchema(t, "", true, nil, nil), nilled)
	wantDeclines(t, undecided)

	icWantCharges(t, icAssess(t, icSchema(t, "", false, nil, nil), nilled),
		icCharge(ruleCvcElt, 4), icChargeAttr(ruleCvcID, 2))
}

// An attribute a ***skip*** {attribute wildcard} admits has no ·governing·
// declaration (§3.10.4.1's Note), so key-sva (§3.3.4.6) clause 2.2 leaves it
// unassessed and §3.17.5.2 clause 3 keeps it out of the ·eligible item set·:
// two of them sharing a lexical bind no ·ID value· and cvc-id clause 2 charges
// nothing (#1043).
func TestSkipWildcardAttributeBindsNoID(t *testing.T) {
	twice := icRoot(idItem(2, "wid", "a"), idItem(3, "wid", "a"))

	icWantCharges(t, icAssess(t, icWildcardSchema(t, xsd.ProcessSkip, nil), twice))

	// Strict and lax ·resolve· the same ·expanded name· to the same top-level
	// xs:ID declaration and DO bind it, which is what makes the silence above a
	// withheld charge rather than a fixture that declares no id at all.
	for _, pc := range []xsd.ProcessContents{xsd.ProcessStrict, xsd.ProcessLax} {
		icWantCharges(t, icAssess(t, icWildcardSchema(t, pc, nil), twice),
			icChargeAttr(ruleCvcID, 3))
	}
}

// An attribute a ***skip*** {attribute wildcard} does NOT admit is not
// ·skipped·: §3.4.4.4 ·attributes· an item to the wildcard only where the
// wildcard MATCHES it, so this one is ·attributed to· nothing and
// key-governing-ad (§3.2.4.2) clause 3 ·resolves· it by name after all. It
// therefore binds its ·ID value· exactly as the strict and lax arms above do,
// and the duplicate is charged cvc-id clause 2 (#717).
//
// The same non-match is a cvc-complex-type clause 2.2.2 charge at each <item>,
// which is what an attribute matching neither a use nor the wildcard costs; the
// id charge lands after both, at the ·validation root·.
func TestSkipWildcardBindsAnIDItDoesNotAdmit(t *testing.T) {
	twice := icRoot(idItem(2, "wid", "a"), idItem(3, "wid", "a"))
	elsewhere := icWildcardSchemaWith(t, nsWildcard(t, xsd.ProcessSkip, "urn:elsewhere"), nil, nil)

	icWantCharges(t, icAssess(t, elsewhere, twice),
		icChargeAttr(ruleCvcComplexType, 2), icChargeAttr(ruleCvcComplexType, 3),
		icChargeAttr(ruleCvcID, 3))
}

// idDefaulted is <item><dtag/></item>, whose EMPTY ID-governed element takes its
// ·initial value· from cvc-elt clause 5.1's substitution rather than from its own
// (absent) character content, so the id it declares is the {lexical form} of its
// declaration's {value constraint} and it binds the ITEM (#853).
func idDefaulted(line int) *testElement {
	dtag := icElem(xsd.QName{Local: "dtag"}, line+1, nil)
	return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(dtag))
}

// An EMPTY element whose declaration carries a {value constraint} declares the id
// that constraint supplies: §3.17.5.2's Note says "default or fixed value
// constraints may play a part", and clause 5.1 is what puts the substituted item
// in the ·eligible item set·. Two of them claim ONE id, which cvc-id clause 2
// charges — a charge the decline this replaced could not make, having recorded no
// member at all.
func TestDefaultedEmptyElementDeclaresItsID(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idDefaulted(2), idDefaulted(4))),
		icCharge(ruleCvcID, 5))

	// The same document with one of them carrying its own character content
	// takes clause 5.2's arm instead, so the two ids differ and clause 2 is
	// satisfied — which is what makes the charge above the substitution's and
	// not a fixture that charges whatever it is given.
	own := icElem(xsd.QName{Local: "item"}, 4, nil, ElementChild(
		icElem(xsd.QName{Local: "dtag"}, 5, nil, TextChild(&testText{data: "d2", loc: loc(5, 8)}))))
	icWantCharges(t, icAssess(t, schema, icRoot(idDefaulted(2), own)))
}

// An IDREF naming the id a defaulted empty element declares RESOLVES, which is
// the other direction of the same substitution: clause 1 charges an empty
// binding, and the reference finds its declaration only because the default was
// read.
func TestDefaultedEmptyElementSatisfiesAnIDREF(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "ref", "d1"), idDefaulted(3))))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "ref", "d2"), idDefaulted(3))),
		icChargeAttr(ruleCvcID, 2))
}

// idDefaultedAttrSchema builds the attribute-side counterpart of idDefaulted's
// shape, which icSchema's shared ItemType cannot carry without every <item> of
// every other fixture declaring the same id:
//
//	root   RootType   sequence( item*, ditem*, dref* )
//	item   ItemType   empty, @xid xs:ID, @ref xs:IDREF
//	ditem  DItemType  empty, @did xs:ID    default "d1"
//	dref   DRefType   empty, @dref xs:IDREF default "ghost"
func idDefaultedAttrSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	defaulted := func(local, typ, lexical string) xsd.AttributeUse {
		vc := xsd.NewValueConstraint(xsd.ValueDefault, lexical, nil, nil)
		return typedUse(t, local, icBuiltin(typ), false, &vc, nil)
	}
	itemType := icComplex(t, "ItemType", []xsd.AttributeUse{
		icUse(t, xsd.QName{Local: "xid"}, "ID"),
		icUse(t, xsd.QName{Local: "ref"}, "IDREF"),
	}, xsd.EmptyContent{})
	dItemType := icComplex(t, "DItemType", []xsd.AttributeUse{defaulted("did", "ID", "d1")}, xsd.EmptyContent{})
	dRefType := icComplex(t, "DRefType", []xsd.AttributeUse{defaulted("dref", "IDREF", "ghost")}, xsd.EmptyContent{})
	local := func(name, typ string) xsd.Particle {
		return icRepeated(t, icLocal(t, "RootType", xsd.QName{Local: name}, xsd.QName{Local: typ}, false, nil))
	}
	rootType := icComplex(t, "RootType", nil,
		icContent(t, local("item", "ItemType"), local("ditem", "DItemType"), local("dref", "DRefType")))
	root, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "root"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "RootType"}}, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	for _, st := range icSeeded(t) {
		b.AddType(st)
	}
	b.AddType(itemType)
	b.AddType(dItemType)
	b.AddType(dRefType)
	b.AddType(rootType)
	b.AddElement(root)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the defaulted-attribute schema: %v", err)
	}
	return schema
}

// idDItem is <ditem/>, leaving its defaulted @did absent unless did is given.
func idDItem(line int, did ...string) *testElement {
	attrs := make([]Attribute, 0, len(did))
	for _, v := range did {
		attrs = append(attrs, icAttr(xsd.QName{Local: "did"}, v, line))
	}
	return icElem(xsd.QName{Local: "ditem"}, line, attrs)
}

// idDRef is <dref/>, whose defaulted @dref is absent.
func idDRef(line int) *testElement {
	return icElem(xsd.QName{Local: "dref"}, line, nil)
}

// An IDREF naming the id an absent ·defaulted attribute· supplies RESOLVES:
// §3.17.5.2's Note puts the item Attribute Default Value adds in the ·eligible
// item set·, and it binds its OWNER element.
func TestDefaultedAttributeSatisfiesAnIDREF(t *testing.T) {
	schema := idDefaultedAttrSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "ref", "d1"), idDItem(3))))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "ref", "d2"), idDItem(3))),
		icChargeAttr(ruleCvcID, 2))
}

// A dangling IDREF beside an element whose ID-governed ·defaulted attribute· is
// absent is still charged under cvc-id clause 1: the default is read, not
// declined, so it does not withhold the clause for the whole ·validation root·
// (#1676).
func TestDefaultedAttributeLeavesClauseOneCharged(t *testing.T) {
	schema := idDefaultedAttrSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "ref", "nowhere"), idDItem(3))),
		icChargeAttr(ruleCvcID, 2))
}

// One defaulted ID supplied to two elements gives its binding two members, which
// cvc-id clause 2 charges at the SECOND owner — the synthesized item has no
// position of its own. A defaulted id matching an explicit one is the same
// duplicate, and an explicit value in place of the default is none.
func TestDefaultedAttributeDeclaresItsID(t *testing.T) {
	schema := idDefaultedAttrSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idDItem(2), idDItem(3))),
		icCharge(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "d1"), idDItem(3))),
		icCharge(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(idDItem(2), idDItem(3, "d2"))))
}

// An absent IDREF-governed ·defaulted attribute· whose default names nothing is
// a dangling reference, which cvc-id clause 1 charges at its owner element.
func TestDefaultedAttributeIDREFIsChargedWhenItNamesNothing(t *testing.T) {
	schema := idDefaultedAttrSchema(t)

	icWantCharges(t, icAssess(t, schema, icRoot(idDRef(2))),
		icCharge(ruleCvcID, 2))
	icWantCharges(t, icAssess(t, schema, icRoot(idItem(2, "xid", "ghost"), idDRef(3))))
}

// The rule ID is the BARE catalog name, with the clause in the message text.
func TestCvcIDRuleIsTheBareCatalogName(t *testing.T) {
	if !xsderr.IsValidRule(ruleCvcID) {
		t.Errorf("%q is not a catalog rule", ruleCvcID)
	}
	if xsderr.IsValidRule(xsderr.Rule("cvc-id.2")) {
		t.Error("cvc-id.2 is a catalog rule; the clause belongs in the message")
	}
}

// Every item the ID/IDREF table could not read is RECORDED as an [Unevaluated]
// under cvc-id at the item's own location, and not only folded into the flag
// that suppresses clause 1: under a backend that maps no xs:string, neither the
// IDREF nor the ID attribute below has a readable value, so the dangling
// reference is not charged and each item says why.
func TestUnreadableIDItemsAreRecorded(t *testing.T) {
	doc := icRoot(idItem(2, "ref", "ghost"), idItem(3, "xid", "a"))
	icWantCharges(t, icAssess(t, idSchema(t), doc), icChargeAttr(ruleCvcID, 2))

	got, undecided := assessRecordedWith(t, gapBackend(icBuiltin("string")), idSchema(t), doc)
	wantSilence(t, got, "an unread item suppresses cvc-id clause 1")
	var ids []Unevaluated
	for _, u := range undecided {
		if u.Rule() == ruleCvcID {
			ids = append(ids, u)
		}
	}
	wantDeclines(t, ids,
		Unevaluated{rule: ruleCvcID, loc: loc(2, 2), msg: "cvc-id clause 1 is undecided"},
		Unevaluated{rule: ruleCvcID, loc: loc(3, 2), msg: "cvc-id clause 1 is undecided"})
}

// idLaxSchema is the fixture for an element ·laxly assessed· below the
// validation root: <root>'s {content type} is a sequence of one lax wildcard,
// 0..unbounded, three top-level element declarations a name below it can
// resolve to, and one top-level attribute declaration.
//
//	root  RootType (named)  sequence( any lax * )
//	thing xs:ID    ref xs:IDREF    num xs:int
//	top-level               @wid xs:ID
func idLaxSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	o, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, 0)
	if err != nil {
		t.Fatalf("NewUnboundedOccurs: %v", err)
	}
	wild, err := xsd.NewParticle(xsderr.Loc{}, o, xsd.ResolvedTerm{Term: *anyWildcard(t, xsd.ProcessLax)})
	if err != nil {
		t.Fatalf("NewParticle: %v", err)
	}
	return cSchemaFrom(t, dType(t, "RootType", "", xsd.DerivationRestriction, nil, cSequence(t, false, wild)),
		func(b *xsd.SchemaBuilder) {
			for _, st := range icSeeded(t) {
				b.AddType(st)
			}
			for _, d := range [][2]string{{"thing", "ID"}, {"ref", "IDREF"}, {"num", "int"}} {
				decl, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: d[0]},
					xsd.TypeDefinitionRef{Name: icBuiltin(d[1])}, nil, xsd.NewGlobalScope(),
					nil, false, nil, nil, nil, false, nil)
				if err != nil {
					t.Fatalf("building the top-level %s element declaration: %v", d[0], err)
				}
				b.AddElement(decl)
			}
			b.AddAttribute(icTopAttribute(t, "wid", "ID"))
		})
}

// idText is an element named local at line holding one run of text.
func idText(local string, line int, text string) *testElement {
	return icElem(xsd.QName{Local: local}, line, nil,
		TextChild(&testText{data: text, loc: loc(line, 8)}))
}

// An element the lax wildcard admitted and no declaration governs is ·laxly
// assessed· against xs:anyType (cvc-assess-elt clause 3, key-lva), whose own
// lax wildcard attributes each of ITS [[children]] in turn, so a <thing> below
// <unknown> resolves to its top-level declaration and its xs:ID value enters
// the ID/IDREF table (§3.17.5.2) like any other. The dangling <ref> beside it
// is then cvc-id clause 1's to charge: nothing below <unknown> was left unread,
// so nothing suppresses it and nothing is recorded as withheld. Handing
// <unknown>'s children no attribution instead fails the second assessment
// (#1823).
func TestAnIDBelowALaxlyAssessedElementIsRead(t *testing.T) {
	schema := idLaxSchema(t)
	unknown := func(id string) *testElement {
		return icElem(xsd.QName{Local: "unknown"}, 2, nil, ElementChild(idText("thing", 3, id)))
	}

	icWantCharges(t, icAssess(t, schema, icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(idText("ref", 4, "dangling")))),
		icCharge(ruleCvcID, 4))

	got, undecided := assessRecordedWith(t, testBackend(), schema, icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(unknown("a")), ElementChild(idText("ref", 4, "dangling"))))
	icWantCharges(t, got, icCharge(ruleCvcID, 4))
	if len(undecided) != 0 {
		t.Errorf("Unevaluated() = %v, want none: a ·laxly assessed· element withholds nothing", undecided)
	}

	icWantCharges(t, icAssess(t, schema, icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(unknown("dangling")), ElementChild(idText("ref", 4, "dangling")))))
}

// An element whose governance was DECIDED keeps key-governing-ad (§3.2.4.2)
// clause 3 for an attribute no use governs, so walk.attributeType's decline for
// an undetermined ·governing type definition· reaches neither shape below
// (#2192). A ·laxly assessed· <unknown> is assessed against xs:anyType, whose
// {attribute wildcard} is lax (§3.4.7); a <num> its top-level declaration
// governs has the SIMPLE type xs:int, with no {attribute wildcard} to be
// ·skipped· by. Either way @wid ·resolves· by ·expanded name· to the top-level
// xs:ID declaration, the two "a"s bind one id twice, and cvc-id clause 2
// charges it. The <num>s are also charged cvc-type clause 3.1.1, which admits
// no attribute but the xsi ones on a simple-typed element.
func TestADecidedElementsAttributeResolvesByItsExpandedName(t *testing.T) {
	withWid := func(local string, line int, text string) *testElement {
		return icElem(xsd.QName{Local: local}, line, []Attribute{icAttr(xsd.QName{Local: "wid"}, "a", line)},
			TextChild(&testText{data: text, loc: loc(line, 8)}))
	}
	doc := func(local, text string) *testElement {
		return icElem(xsd.QName{Local: "root"}, 1, nil,
			ElementChild(withWid(local, 2, text)), ElementChild(withWid(local, 3, text)))
	}

	icWantCharges(t, icAssess(t, idLaxSchema(t), doc("unknown", "")),
		icChargeAttr(ruleCvcID, 3))
	icWantCharges(t, icAssess(t, idLaxSchema(t), doc("num", "12")),
		icChargeAttr(ruleCvcType, 2), icChargeAttr(ruleCvcType, 3), icChargeAttr(ruleCvcID, 3))
}

// The same attribution makes a resolving descendant ·strictly assessed·
// (key-governing-ed clause 3 over a lax wildcard): <num> below <unknown> is
// charged cvc-type clause 3.1.3 for a value its xs:int declaration rejects, and
// so is one two ·laxly assessed· levels down, a name that resolves nothing being
// laxly assessed again and never invalid.
func TestAResolvingDescendantOfALaxlyAssessedElementIsStrictlyAssessed(t *testing.T) {
	schema := idLaxSchema(t)

	wantSilence(t, icAssess(t, schema, icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(icElem(xsd.QName{Local: "unknown"}, 2, nil, ElementChild(idText("num", 3, "12")))))),
		"12 is an xs:int")
	got := icAssess(t, schema, icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(icElem(xsd.QName{Local: "unknown"}, 2, nil, ElementChild(idText("num", 3, "abc"))))))
	wantContentCharge(t, got, "cvc-type", "3.1.3", loc(3, 1))
	got = icAssess(t, schema, icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(icElem(xsd.QName{Local: "unknown"}, 2, nil,
			ElementChild(icElem(xsd.QName{Local: "other"}, 3, nil, ElementChild(idText("num", 4, "abc"))))))))
	wantContentCharge(t, got, "cvc-type", "3.1.3", loc(4, 1))
}

// IDREFS items are delimited on XML white space alone (cvc-datatype-valid,
// Datatypes §4.1.4 clause 2.2). U+1680 is an NCName character, so the IDREFS
// value a<U+1680>b is ONE ·IDREF value·, naming the ID a<U+1680>b; split at
// the U+1680 it would name the undeclared a and b, charged cvc-id clause 1.
func TestIDREFSItemsSplitOnXMLSpaceAlone(t *testing.T) {
	schema := idSchema(t)

	icWantCharges(t, icAssess(t, schema,
		icRoot(idItem(2, "xid", "a b"), idItem(3, "refs", "a b"))))
}

// valueTokens splits on XML white space alone. A U+00A0 or U+2028 inside an
// item cannot reach it end to end — xs:NCName's String Valid check rejects
// the item first — so these rows sit here.
func TestValueTokensSplitOnXMLSpaceAlone(t *testing.T) {
	for _, lexical := range []string{"a b", "a b", "a b"} {
		if got := valueTokens(lexical, true); len(got) != 1 || got[0] != lexical {
			t.Errorf("valueTokens(%q, true) = %q, want the one token %q", lexical, got, lexical)
		}
	}
	if got := valueTokens("a\tb\nc\rd e", true); len(got) != 5 {
		t.Errorf("valueTokens over the four S characters = %q, want 5 tokens", got)
	}
}
