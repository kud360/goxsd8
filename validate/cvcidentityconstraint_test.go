package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These fixtures drive cvc-identity-constraint (§3.11.4) and the §3.11.5 node
// tables it rests on, over the schema shape icfixture_test.go builds.

// icItem is <item> with the given no-namespace attributes, one Loc per line.
func icItem(ns string, line int, attrs ...xsd.QName) func(...string) *testElement {
	return func(values ...string) *testElement {
		items := make([]Attribute, 0, len(attrs))
		for i, a := range attrs {
			items = append(items, icAttr(a, values[i], line))
		}
		return icElem(xsd.QName{Space: ns, Local: "item"}, line, items)
	}
}

// icIDed is <item id="..."> at line.
func icIDed(line int, id string) *testElement {
	return icItem("", line, xsd.QName{Local: "id"})(id)
}

// icRoot is <root> over the given children.
func icRoot(kids ...Element) *testElement {
	return icElem(xsd.QName{Local: "root"}, 1, nil, icKids(kids...)...)
}

// A key forbids two ·target nodes· to share a ·key-sequence· (clause 4.2.2),
// and the charge lands on the LATER of the two — the one a reader has to
// change.
func TestKeyChargesADuplicateKeySequence(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "@id")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil)

	icWantCharges(t, icAssess(t, schema, icRoot(icIDed(2, "a"), icIDed(3, "a"))),
		icCharge(ruleCvcIdentityConstraint, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(icIDed(2, "a"), icIDed(3, "b"))))
}

// Clause 4.2.1 makes a MISSING field an error for a key, and clause 4.1 leaves
// the same document alone for a unique — whose own Note says so: "a selected
// unique node may have fields that do not have corresponding [schema actual
// value]s".
func TestKeyChargesAnAbsentFieldAndUniqueDoesNot(t *testing.T) {
	bare := icElem(xsd.QName{Local: "item"}, 3, nil)

	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "@id")
	icWantCharges(t,
		icAssess(t, icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil), icRoot(icIDed(2, "a"), bare)),
		icCharge(ruleCvcIdentityConstraint, 3))

	unique := icDef(t, "U", xsd.IdentityConstraintUnique, ".//item", nil, "", "@id")
	icWantCharges(t,
		icAssess(t, icSchema(t, "", false, []xsd.IdentityConstraint{unique}, nil), icRoot(icIDed(2, "a"), bare)))
}

// Clause 3 admits at most one node with a non-absent [schema actual value] per
// field, so a field selecting two of them is charged at the second — the
// attribute information item that is one too many, not the ·target node·.
func TestKeyChargesAFieldSelectingTwoValuedNodes(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "@*")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil)

	two := icItem("", 2, xsd.QName{Local: "id"}, xsd.QName{Local: "k"})("a", "1")
	icWantCharges(t, icAssess(t, schema, icRoot(two)), icChargeAttr(ruleCvcIdentityConstraint, 2))
	icWantCharges(t, icAssess(t, schema, icRoot(icIDed(2, "a"))))
}

// A union is a sequence of distinct NODES, so an attribute two branches both
// select is ONE field node — not the clause 3 violation two would be.
func TestFieldUnionSelectsOneAttributeOnce(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "@id|@*")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil)

	icWantCharges(t, icAssess(t, schema, icRoot(icIDed(2, "a"))))
}

// Node tables propagate UPWARD and never sideways (PRINCIPLES 15): a keyref on
// <box> resolves against the key sequences sourced inside that box, and a key
// sourced in a SIBLING box is not among them — §3.11.4 clause 4.3's own Note,
// "only element information items within the sub-tree rooted at the element
// information item being ·validated· can be referenced successfully".
func TestKeyrefResolvesInsideItsSubtreeAndNotOutsideIt(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "item", nil, "", "@id")
	keyref := icDef(t, "R", xsd.IdentityConstraintKeyref, "ref", nil, "K", "@r")
	schema := icSchema(t, "", false, nil, []xsd.IdentityConstraint{key, keyref})

	ref := func(line int, r string) *testElement {
		return icElem(xsd.QName{Local: "ref"}, line, []Attribute{icAttr(xsd.QName{Local: "r"}, r, line)})
	}
	box := func(line int, kids ...Element) *testElement {
		return icElem(xsd.QName{Local: "box"}, line, nil, icKids(kids...)...)
	}

	// One box holding both the key and the reference: the keyref resolves.
	inside := icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(box(2, icIDed(3, "a"), ref(4, "a"))))
	icWantCharges(t, icAssess(t, schema, inside))

	// The key one box over is invisible, however plainly it is in the
	// document: it never entered THIS box's node table.
	outside := icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(box(2, icIDed(3, "a"), ref(4, "a"))),
		ElementChild(box(5, icIDed(6, "b"), ref(7, "a"))))
	icWantCharges(t, icAssess(t, schema, outside), icCharge(ruleCvcIdentityConstraint, 7))
}

// Key-sequence members are compared in the VALUE space (Datatypes §2.2), so two
// xs:integer fields written "1" and "01" are ONE key sequence. A lexical
// comparison would find no duplicate here at all.
func TestKeySequencesCompareAsValuesAndNotLexicals(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "@k")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil)

	numbered := func(line int, k string) *testElement {
		return icItem("", line, xsd.QName{Local: "k"})(k)
	}
	icWantCharges(t, icAssess(t, schema, icRoot(numbered(2, "1"), numbered(3, "01"))),
		icCharge(ruleCvcIdentityConstraint, 3))
	icWantCharges(t, icAssess(t, schema, icRoot(numbered(2, "1"), numbered(3, "2"))))
}

// xpathDefaultNamespace supplies the default namespace of an unprefixed
// NameTest reached by an ELEMENT step and of no attribute step (PRINCIPLES 15).
// Both halves are pinned by one document: the selector "item" finds the
// namespaced elements only if the default reached it, and the field "@id" finds
// the NO-namespace attribute only if the default did not.
func TestSelectorAndFieldTreatTheDefaultNamespaceAsymmetrically(t *testing.T) {
	noNS, inNS := xsd.QName{Local: "id"}, xsd.QName{Space: icNS, Local: "id"}
	item := func(line int, bare, qualified string) *testElement {
		return icItem(icNS, line, noNS, inNS)(bare, qualified)
	}
	// The two items agree on the no-namespace id and differ on the namespaced
	// one, so which attribute the field selected decides whether they collide.
	doc := icRoot(item(2, "same", "1"), item(3, "same", "2"))
	def := icNS

	element := icDef(t, "K", xsd.IdentityConstraintKey, "item", &def, "", "@id")
	icWantCharges(t, icAssess(t, icSchema(t, icNS, false, []xsd.IdentityConstraint{element}, nil), doc),
		icCharge(ruleCvcIdentityConstraint, 3))

	qualified := icDef(t, "K", xsd.IdentityConstraintKey, "item", &def, "", "@p:id")
	icWantCharges(t, icAssess(t, icSchema(t, icNS, false, []xsd.IdentityConstraint{qualified}, nil), doc))
}

// A field node whose ·governing type definition· this package could not
// determine has no [schema actual value] to contribute (§3.3.5.4), so the
// constraint DECLINES rather than comparing lexicals — the same two documents
// that collide with a determinable type charge nothing with an xsi:type on the
// field node.
func TestFieldWithNoGoverningTypeDeclines(t *testing.T) {
	field := func(local string) *xsd.Schema {
		key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", local)
		return icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil)
	}
	named := func(line int, local, text string) *testElement {
		node := icElem(xsd.QName{Local: local}, line+1, nil,
			TextChild(&testText{data: text, loc: loc(line+1, 8)}))
		return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(node))
	}

	icWantCharges(t, icAssess(t, field("name"), icRoot(named(2, "name", "a"), named(4, "name", "a"))),
		icCharge(ruleCvcIdentityConstraint, 4))
	icWantCharges(t, icAssess(t, field("tabled"), icRoot(named(2, "tabled", "a"), named(4, "tabled", "a"))))
}

// Clause 4.2.3 forbids a key's ·key-sequence· to take an ELEMENT member from a
// declaration whose {nillable} is true, whatever the element's content is. The
// charge carries the field node's own Loc.
func TestKeyChargesANillableElementMember(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "name")
	named := icElem(xsd.QName{Local: "item"}, 2, nil,
		ElementChild(icElem(xsd.QName{Local: "name"}, 3, nil,
			TextChild(&testText{data: "a", loc: loc(3, 8)}))))
	doc := icRoot(named)

	icWantCharges(t, icAssess(t, icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil), doc))
	icWantCharges(t, icAssess(t, icSchema(t, "", true, []xsd.IdentityConstraint{key}, nil), doc),
		icCharge(ruleCvcIdentityConstraint, 3))
}

// Only a ·nilled· field node withholds its ·key-sequence· member
// (elementKeyMember), and ·nilled· is key-nilled's conjunction — D.{nillable} =
// true AND an ·actual value· of true ([nilled]) — not the PRESENCE of an xsi:nil
// attribute. Both conjuncts are pinned by the duplicate a unique charges only
// where both <name> nodes supplied their member: reading presence alone would
// decline every slot below and charge nothing at all.
func TestOnlyANilledFieldNodeWithholdsItsKeySequenceMember(t *testing.T) {
	unique := icDef(t, "U", xsd.IdentityConstraintUnique, ".//item", nil, "", "name")
	doc := func(nil_ string) *testElement {
		named := func(line int) *testElement {
			name := icElem(xsd.QName{Local: "name"}, line+1,
				[]Attribute{icAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, nil_, line+1)},
				TextChild(&testText{data: "a", loc: loc(line+1, 8)}))
			return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(name))
		}
		return icRoot(named(2), named(4))
	}
	ics := []xsd.IdentityConstraint{unique}

	icWantCharges(t, icAssess(t, icSchema(t, "", true, ics, nil), doc("false")),
		icCharge(ruleCvcIdentityConstraint, 4))

	// Both nodes ·nilled·: §3.3.5.4 gives each an absent [schema actual value],
	// so neither ·key-sequence· has a member and clause 4.1 compares nothing.
	// Their character [[children]] are cvc-elt clause 3.2.3.1's to charge, at
	// the offending [[child]]'s own position.
	icWantCharges(t, icAssess(t, icSchema(t, "", true, ics, nil), doc("true")),
		icChargeAt(ruleCvcElt, loc(3, 8)), icChargeAt(ruleCvcElt, loc(5, 8)))

	// xsi:nil = true on a declaration whose {nillable} is false makes no
	// ·nilled· node either: cvc-elt clause 3.1 charges the attribute, and the
	// members are supplied and compared as before.
	icWantCharges(t, icAssess(t, icSchema(t, "", false, ics, nil), doc("true")),
		icCharge(ruleCvcElt, 3), icCharge(ruleCvcElt, 5),
		icCharge(ruleCvcIdentityConstraint, 4))
}

// An identity constraint whose {selector} or {fields} fall outside the
// §3.11.6.2/§3.11.6.3 subset charges nothing at all (icpath.go's GAP), and
// neither does a keyref referring to a key that declined.
func TestUnreadablePathDeclinesTheWholeConstraint(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "item[1]", nil, "", "@id")
	keyref := icDef(t, "R", xsd.IdentityConstraintKeyref, "ref", nil, "K", "@r")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{key, keyref}, nil)

	ref := icElem(xsd.QName{Local: "ref"}, 4, []Attribute{icAttr(xsd.QName{Local: "r"}, "zzz", 4)})
	icWantCharges(t, icAssess(t, schema, icRoot(icIDed(2, "a"), icIDed(3, "a"), ref)))
}

// A keyref whose {referenced key} has no node table anywhere in the subtree is
// charged: "there is a node table associated with the {referenced key}" is the
// first half of clause 4.3, and a key whose own element never occurred fails it.
func TestKeyrefChargesWhenTheReferencedKeyNeverOccurred(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "item", nil, "", "@id")
	keyref := icDef(t, "R", xsd.IdentityConstraintKeyref, "ref", nil, "K", "@r")
	schema := icSchema(t, "", false, nil, []xsd.IdentityConstraint{key, keyref})

	ref := icElem(xsd.QName{Local: "ref"}, 3, []Attribute{icAttr(xsd.QName{Local: "r"}, "a", 3)})
	doc := icElem(xsd.QName{Local: "root"}, 1, nil,
		ElementChild(icElem(xsd.QName{Local: "box"}, 2, nil, ElementChild(ref))))
	icWantCharges(t, icAssess(t, schema, doc), icCharge(ruleCvcIdentityConstraint, 3))
}

// A `@NameTest` field over an ANONYMOUS governing type declines for the same
// reason cvc-id does: the top-level declaration its ·expanded name· resolves to
// is not the ·governing type definition·, and a key-sequence member compared
// under the wrong simple type is compared in the wrong ·value space·. The two
// documents below are one duplicate under xs:integer and two distinct values
// under the xs:string <kid> actually declares.
func TestFieldOverAnAnonymousTypeDeclinesRatherThanReadingTheTopLevelType(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "kid", nil, "", "@aid")
	schema := icAnonymousSchema(t, "string", "integer", []xsd.IdentityConstraint{key})

	icWantCharges(t, icAssess(t, schema, icRoot(icKid(2, "aid")("1"), icKid(3, "aid")("01"))))
}

// A `@NameTest` field reads the same ·governing type definition· cvc-id does
// (icCheck.fieldAttributes over walk.attributeType), so an attribute a
// ***skip*** {attribute wildcard} admits lengthens no ·key-sequence· either
// (#1043): the two ·target nodes· below share no key-sequence to be charged
// clause 4.2.2 for, since neither has one at all.
//
// The silence under skip is WIDER than the spec's, and deliberately: a
// ·target node· whose ·key-sequence· is short is out of the ·qualified node
// set·, which for a key is clause 4.2.1's charge (§3.11.4 clause 3's Note names
// ·skipped· nodes as the case), so the spec charges 4.2.1 twice on this
// document and the whole-frame decline withholds both. The GAP marker on
// icCheck.fieldAttributes states that, on the nilled sibling's terms.
func TestSkipWildcardAttributeLengthensNoKeySequence(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "item", nil, "", "@wid")
	twice := icRoot(idItem(2, "wid", "a"), idItem(3, "wid", "a"))

	icWantCharges(t, icAssess(t, icWildcardSchema(t, xsd.ProcessSkip, []xsd.IdentityConstraint{key}), twice))

	// Under lax the same field DOES read the top-level declaration and the two
	// key-sequences collide, which is the charge the skip case withholds. The
	// declaration is xs:ID, so the id it also binds is charged alongside it.
	icWantCharges(t, icAssess(t, icWildcardSchema(t, xsd.ProcessLax, []xsd.IdentityConstraint{key}), twice),
		icCharge(ruleCvcIdentityConstraint, 3), icChargeAttr(ruleCvcID, 3))
}

// The rule ID is the BARE catalog name, with the clause in the message text.
func TestIdentityConstraintRuleIsTheBareCatalogName(t *testing.T) {
	if !xsderr.IsValidRule(ruleCvcIdentityConstraint) {
		t.Errorf("%q is not a catalog rule", ruleCvcIdentityConstraint)
	}
	if xsderr.IsValidRule(xsderr.Rule("cvc-identity-constraint.4.2.1")) {
		t.Error("cvc-identity-constraint.4.2.1 is a catalog rule; the clause belongs in the message")
	}
}
