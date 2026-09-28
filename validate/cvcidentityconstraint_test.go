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

// Only a ·nilled· field node contributes no ·key-sequence· member
// (elementKeyMember), and ·nilled· is key-nilled's conjunction — D.{nillable} =
// true AND an ·actual value· of true ([nilled]) — not the PRESENCE of an xsi:nil
// attribute. Both conjuncts are pinned by the duplicate a unique charges only
// where both <name> nodes supplied their member: reading presence alone would
// leave every sequence below short and charge nothing at all.
func TestOnlyANilledFieldNodeContributesNoKeySequenceMember(t *testing.T) {
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
// §3.11.6.2/§3.11.6.3 subset charges nothing at all (icFrame.declined's GAP),
// and neither does a keyref referring to a key that declined.
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

// A `@NameTest` field over an ANONYMOUS governing type reads that type's own
// {attribute uses} for the same reason cvc-id does (icCheck.fieldAttributes
// over walk.attributeType): the top-level declaration its ·expanded name·
// resolves to is not the ·governing type definition·, and a key-sequence member
// compared under the wrong simple type is compared in the wrong ·value space·.
// The two documents below are one duplicate under the top level's xs:integer
// and two distinct ·key-sequences· under the xs:string <kid> declares, so the
// silence here is the field's decided verdict and the charge is what reading
// the wrong type would cost.
func TestFieldOverAnAnonymousTypeReadsItsOwnTypeAndNotTheTopLevelOne(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "kid", nil, "", "@aid")
	schema := icAnonymousSchema(t, "string", "integer", []xsd.IdentityConstraint{key})

	icWantCharges(t, icAssess(t, schema, icRoot(icKid(2, "aid")("1"), icKid(3, "aid")("01"))))
}

// A `@NameTest` field reads the same ·governing type definition· cvc-id does
// (icCheck.fieldAttributes over walk.attributeType), so an attribute a
// ***skip*** {attribute wildcard} admits lengthens no ·key-sequence· either
// (#1043): neither ·target node· below has a key-sequence to be charged clause
// 4.2.2 for.
//
// Each is charged clause 4.2.1 instead, and that is the whole difference #717
// made here. A ·target node· whose ·key-sequence· is short is out of the
// ·qualified node set·, which for a key is clause 4.2.1's charge (§3.11.4
// clause 3's Note names ·skipped· nodes as the case). The charge was withheld
// while ·skipped· was inferred from the wildcard's PRESENCE, since an attribute
// the wildcard did not admit arrived indistinguishably; cvc-wildcard now
// decides the ·attribution· ([walk.skippedAttribute]), so the absence read here
// is the spec's and the charge is made.
func TestSkipWildcardAttributeLengthensNoKeySequence(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "item", nil, "", "@wid")
	twice := icRoot(idItem(2, "wid", "a"), idItem(3, "wid", "a"))

	icWantCharges(t, icAssess(t, icWildcardSchema(t, xsd.ProcessSkip, []xsd.IdentityConstraint{key}), twice),
		icCharge(ruleCvcIdentityConstraint, 2), icCharge(ruleCvcIdentityConstraint, 3))

	// Under lax the same field DOES read the top-level declaration and the two
	// key-sequences collide, which is the charge the skip case withholds. The
	// declaration is xs:ID, so the id it also binds is charged alongside it.
	icWantCharges(t, icAssess(t, icWildcardSchema(t, xsd.ProcessLax, []xsd.IdentityConstraint{key}), twice),
		icCharge(ruleCvcIdentityConstraint, 3), icChargeAttr(ruleCvcID, 3))
}

// A field node that is an EMPTY element whose declaration carries a {value
// constraint} contributes the ·actual value· of that constraint's {lexical form}
// to the ·key-sequence·, which is exactly what §3.11.4's Note calls for: "default
// or fixed value constraints may play a part in ·key-sequences·" (cvc-elt clause
// 5.1, #853).
//
// Two such nodes therefore share ONE ·key-sequence· and a key charges clause
// 4.2.2 against the later — a charge the decline this replaced could not make,
// having contributed no member and left the sequence short. The three documents
// are the discriminating set: two substituted values collide, a substituted value
// and an owned one do not, and a MISSING <dtag> leaves the sequence short, which
// is clause 4.2.1's own charge and not this one.
func TestDefaultedEmptyElementFillsItsKeySequence(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "dtag")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil)

	icWantCharges(t, icAssess(t, schema, icRoot(idDefaulted(2), idDefaulted(4))),
		icCharge(ruleCvcIdentityConstraint, 4), icCharge(ruleCvcID, 5))

	own := icElem(xsd.QName{Local: "item"}, 4, nil, ElementChild(
		icElem(xsd.QName{Local: "dtag"}, 5, nil, TextChild(&testText{data: "d2", loc: loc(5, 8)}))))
	icWantCharges(t, icAssess(t, schema, icRoot(idDefaulted(2), own)))

	bare := icElem(xsd.QName{Local: "item"}, 4, nil)
	icWantCharges(t, icAssess(t, schema, icRoot(idDefaulted(2), bare)),
		icCharge(ruleCvcIdentityConstraint, 4))
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

// icDeclines is the cvc-identity-constraint records among undecided, in order.
func icDeclines(undecided []Unevaluated) []Unevaluated {
	var out []Unevaluated
	for _, u := range undecided {
		if u.Rule() == ruleCvcIdentityConstraint {
			out = append(out, u)
		}
	}
	return out
}

// A key whose field selects only ·nilled· nodes leaves every ·key-sequence·
// short, which §3.11.4 clause 3's Note names outright, so each ·target node·
// is out of the ·qualified node set· and clause 4.2.1 charges it. Nothing is
// withheld, so nothing is recorded.
func TestANilledFieldNodeLeavesAKeySequenceShort(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "name")
	schema := icSchema(t, "", true, []xsd.IdentityConstraint{key}, nil)
	doc := icRoot(idNilled(2, "true"), idNilled(4, "true"))

	got, undecided := assessRecorded(t, schema, doc)
	icWantCharges(t, got, icCharge(ruleCvcIdentityConstraint, 2), icCharge(ruleCvcIdentityConstraint, 4))
	wantDeclines(t, undecided)
}

// Every identity-constraint decline is RECORDED at its origin: a path outside
// the subset at the element declaring the constraint (clause 4), a keyref whose
// {referenced key}'s node table was not assembled (clause 4.3), and a field node
// with no determinable ·governing type definition· (clause 3).
func TestIdentityConstraintDeclinesAreRecorded(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, "item[1]", nil, "", "@id")
	keyref := icDef(t, "R", xsd.IdentityConstraintKeyref, "ref", nil, "K", "@r")
	ref := icElem(xsd.QName{Local: "ref"}, 4, []Attribute{icAttr(xsd.QName{Local: "r"}, "zzz", 4)})
	_, undecided := assessRecorded(t, icSchema(t, "", false, []xsd.IdentityConstraint{key, keyref}, nil),
		icRoot(icIDed(2, "a"), icIDed(3, "a"), ref))
	wantDeclines(t, icDeclines(undecided),
		Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(1, 1), msg: "clauses 3 and 4 are undecided"},
		Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(1, 1), msg: "clause 4.3 is undecided"})

	tabled := icDef(t, "T", xsd.IdentityConstraintKey, ".//item", nil, "", "tabled")
	named := func(line int) *testElement {
		node := icElem(xsd.QName{Local: "tabled"}, line+1, nil, TextChild(&testText{data: "a", loc: loc(line+1, 8)}))
		return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(node))
	}
	_, undecided = assessRecorded(t, icSchema(t, "", false, []xsd.IdentityConstraint{tabled}, nil), icRoot(named(2), named(4)))
	wantDeclines(t, icDeclines(undecided),
		Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(3, 1), msg: "clauses 3 and 4 are undecided"},
		Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(5, 1), msg: "clauses 3 and 4 are undecided"})
}

// A ·key-sequence· comparison sameKeyMember cannot make — two members validated
// against different simple types, here xs:string and xs:ID — withholds the
// clause that reads it, and is recorded: clause 4.2.2 against the later ·target
// node· of a key, and clause 4.3 against a keyref member. The xs:string/xs:string
// control is the decided answer the first record stands in for.
func TestUndecidedKeySequenceComparisonsAreRecorded(t *testing.T) {
	key := icDef(t, "K", xsd.IdentityConstraintKey, ".//item", nil, "", "@id|@xid")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{key}, nil)
	icWantCharges(t, icAssess(t, schema, icRoot(icIDed(2, "a"), icIDed(3, "a"))), icCharge(ruleCvcIdentityConstraint, 3))
	got, undecided := assessRecorded(t, schema, icRoot(icIDed(2, "a"), idItem(3, "xid", "a")))
	wantSilence(t, got, "an undecided comparison charges nothing")
	wantDeclines(t, icDeclines(undecided), Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(3, 1), msg: "clause 4.2.2 is undecided"})

	unique := icDef(t, "U", xsd.IdentityConstraintUnique, "item", nil, "", "@xid")
	keyref := icDef(t, "R", xsd.IdentityConstraintKeyref, "item", nil, "U", "@ref")
	got, undecided = assessRecorded(t, icSchema(t, "", false, []xsd.IdentityConstraint{unique, keyref}, nil),
		icRoot(idItem(2, "xid", "a"), idItem(3, "ref", "a")))
	wantSilence(t, got, "an undecided lookup charges nothing")
	wantDeclines(t, icDeclines(undecided), Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(3, 1), msg: "clause 4.3 is undecided"})
}

// §3.11.5's conflict resolution drops two child entries sharing a
// ·key-sequence·, so a keyref member matching that sequence is charged under
// clause 4.3. Where the comparison between the two could not be made, both
// entries are kept CONTESTED, and a member matching only a contested entry is
// recorded as undecided rather than passing on an entry the proviso may have
// removed. Without the contested mark, the second document walks clean.
func TestKeyrefMatchingOnlyAContestedEntryIsRecorded(t *testing.T) {
	unique := icDef(t, "U", xsd.IdentityConstraintUnique, "item", nil, "", "@id|@xid")
	keyref := icDef(t, "R", xsd.IdentityConstraintKeyref, "item", nil, "U", "@id")
	schema := icSchema(t, "", false, []xsd.IdentityConstraint{keyref}, []xsd.IdentityConstraint{unique})
	box := func(line int, item *testElement) *testElement {
		return icElem(xsd.QName{Local: "box"}, line, nil, ElementChild(item))
	}

	got, undecided := assessRecorded(t, schema, icRoot(icIDed(2, "a"), box(3, icIDed(4, "a")), box(5, icIDed(6, "a"))))
	icWantCharges(t, got, icCharge(ruleCvcIdentityConstraint, 2))
	wantDeclines(t, icDeclines(undecided))

	got, undecided = assessRecorded(t, schema, icRoot(icIDed(2, "a"), box(3, icIDed(4, "a")), box(5, idItem(6, "xid", "a"))))
	wantSilence(t, got, "a match on a contested entry charges nothing")
	wantDeclines(t, icDeclines(undecided), Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(2, 1), msg: "clause 4.3 is undecided"})
}
