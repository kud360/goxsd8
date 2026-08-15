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

// An attribute of an UNFOLDED governing type declines even where a top-level
// declaration of its ·expanded name· exists: that declaration is a different
// component from the {attribute declaration} of the use this package cannot
// see, so its type is not the ·governing type definition· (walk.attributeType).
// Reading it would classify @aid by whichever type the schema happens to also
// declare at the top level, and charge clause 1 against a document whose own
// type declares the id.
func TestUnfoldedAttributeDeclinesOverACollidingTopLevelType(t *testing.T) {
	// @aid is xs:ID where <kid>'s own type governs it and xs:string at the
	// top level, so the top-level reading declares no id at all.
	schema := icUnfoldedSchema(t, "ID", "string", nil)

	icWantCharges(t, icAssess(t, schema, icRoot(icKid(2, "aid", "ref")("x1", "x1"))))
	// The decline holds in the other direction too: a reference this package
	// cannot type is not charged for the declaration it never read.
	icWantCharges(t, icAssess(t, schema, icRoot(icKid(2, "ref")("ghost"))))
}

// The same decline keeps clause 2 from charging a duplicate the document does
// not have: an item read under the WRONG type is not the unread item
// [idTable.charge]'s asymmetry reasons about, since it adds a binding member
// off a type that governs nothing here.
func TestUnfoldedAttributeDeclinesRatherThanManufacturingADuplicate(t *testing.T) {
	// @aid is xs:string where <kid>'s own type governs it and xs:ID at the
	// top level, so the top-level reading declares one id twice.
	schema := icUnfoldedSchema(t, "string", "ID", nil)

	icWantCharges(t, icAssess(t, schema,
		icRoot(icKid(2, "aid")("x1"), icKid(3, "aid")("x1"))))
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
