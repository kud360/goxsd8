package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// icLaxFieldSchema is a no-namespace schema declaring <root> over the item
// declaration the markup item spells, with one unique U on <root> over the
// selector "item" and field. T is a named complex type whose element and
// attribute wildcards are both ***lax***, for item to name.
func icLaxFieldSchema(t *testing.T, item, field string) *xsd.Schema {
	t.Helper()
	return parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="T">
    <xs:sequence><xs:any processContents="lax" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>
    <xs:anyAttribute processContents="lax"/>
  </xs:complexType>
  <xs:element name="root">
    <xs:complexType><xs:sequence>` + item + `</xs:sequence></xs:complexType>
    <xs:unique name="U"><xs:selector xpath="item"/><xs:field xpath="` + field + `"/></xs:unique>
  </xs:element>
</xs:schema>`})
}

// The item declarations icLaxFieldSchema is built over: of xs:anyType; of T
// behind a {type table} whose {test} the §3.12.6 evaluator declines
// (key-cta-ta-select), so no ·governing type definition· is determined; of T
// alone; and of an anonymous type whose element and attribute wildcards are
// both ***skip***.
const (
	icAnyItem    = `<xs:element name="item" type="xs:anyType" maxOccurs="unbounded"/>`
	icTabledItem = `<xs:element name="item" type="T" maxOccurs="unbounded"><xs:alternative test="child::x" type="T"/><xs:alternative type="T"/></xs:element>`
	icLaxItem    = `<xs:element name="item" type="T" maxOccurs="unbounded"/>`
	icSkipItem   = `<xs:element name="item" maxOccurs="unbounded"><xs:complexType>
      <xs:sequence><xs:any processContents="skip" minOccurs="0"/></xs:sequence>
      <xs:anyAttribute processContents="skip"/>
    </xs:complexType></xs:element>`
)

// icBItem is <item> at line holding one <b> at line+1 whose content is "x",
// the <b> carrying attrs, with xs bound to the XSD namespace for an xsi:type.
func icBItem(line int, attrs ...Attribute) *testElement {
	b := icElem(xsd.QName{Local: "b"}, line+1, attrs, TextChild(&testText{data: "x", loc: loc(line+1, 4)}))
	b.bindings = map[string]string{"xs": xsd.XMLSchemaNS}
	return icElem(xsd.QName{Local: "item"}, line, nil, ElementChild(b))
}

// icAbsentOpening is the opening of the clause 3 charge of a field "b" or "@b"
// of U selecting a node whose ·governing type definition· is ·absent·, at at.
func icAbsentOpening(field string, at xsderr.Loc) string {
	return at.String() + `: [cvc-identity-constraint] element item, a ·target node· of the identity constraint U declared on root, has its field "` + field + `" select a node whose ·governing type definition· is ·absent·, which cvc-identity-constraint clause 3 admits as none of its nodes`
}

// icWantAbsentCharge fails unless got is the one clause 3 charge
// icAbsentOpening states, and undecided holds no cvc-identity-constraint
// record.
func icWantAbsentCharge(t *testing.T, got []*xsderr.Error, undecided []Unevaluated, field string, at xsderr.Loc) {
	t.Helper()
	icWantCharges(t, got, icChargeAt(ruleCvcIdentityConstraint, at))
	if want := icAbsentOpening(field, at); got[0].Error() != want {
		t.Errorf("Violations()[0] = %q, want %q", got[0].Error(), want)
	}
	wantDeclines(t, icDeclines(undecided))
}

// A field node ·laxly assessed· with no ·instance-specified type definition·
// has an ·absent· ·governing type definition· (key-lva clause 1,
// key-governing-type-elem's closing sentence) and is not ·skipped·
// (key-skipped), so cvc-identity-constraint clause 3 admits it as none of its
// nodes: the <b> an xs:anyType <item> attributes to its lax wildcard is charged
// at its own Loc, and nothing is recorded undecided. An xsi:type that does not
// ·resolve· leaves it laxly assessed too (key-itd), so it is charged the same,
// beside cvc-attribute's clause 5 charge of the xsi:type itself.
func TestALaxlyAssessedFieldNodeIsChargedUnderClause3(t *testing.T) {
	got, undecided := assessRecorded(t, icLaxFieldSchema(t, icAnyItem, "b"), icRoot(icBItem(2)))
	icWantAbsentCharge(t, got, undecided, "b", loc(3, 1))

	unresolved := icAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}, "xs:nosuch", 3)
	got, undecided = assessRecorded(t, icLaxFieldSchema(t, icAnyItem, "b"), icRoot(icBItem(2, unresolved)))
	icWantCharges(t, got, icChargeAttr(ruleCvcAttribute, 3), icCharge(ruleCvcIdentityConstraint, 3))
	wantDeclines(t, icDeclines(undecided))
}

// A field node whose ·governing type definition· this package could not
// determine is no ·laxly assessed· one and stays declined: the <b> beneath an
// <item> whose {type table} selection the evaluator declines is walked
// undecided, so U charges nothing and records the clause 3 decline at <b>.
func TestAnUndeterminedFieldNodeBesideTheLaxOneDeclines(t *testing.T) {
	got, undecided := assessRecorded(t, icLaxFieldSchema(t, icTabledItem, "b"), icRoot(icBItem(2)))
	wantSilence(t, got, "<b>'s governance was not decided")
	wantDeclines(t, icDeclines(undecided),
		Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(3, 1), msg: "its ·governing type definition· could not be determined"})
}

// A ·skipped· <b>, which a ***skip*** wildcard attributes, is one of the
// "zero or more ·skipped· nodes" clause 3 admits: it never reaches a field
// slot, so a unique is satisfied by an <item> holding one and nothing is
// recorded.
func TestASkippedFieldNodeIsNoneOfClause3sOtherNodes(t *testing.T) {
	got, undecided := assessRecorded(t, icLaxFieldSchema(t, icSkipItem, "b"), icRoot(icBItem(2)))
	wantSilence(t, got, "a ·skipped· <b> leaves U's ·key-sequence· short")
	wantDeclines(t, icDeclines(undecided))
}

// A <b> with no declaration whose xsi:type ·resolves· to xs:string is governed
// by that type (key-governing-type-elem clause 8) and ·strictly assessed·
// (key-sva clause 1.2), not ·laxly assessed·: it fills the one simple-valued
// slot clause 3 admits, so one <item> is valid and a second sharing its value is
// charged clause 4.1.
func TestAnUndeclaredFieldNodeTypedByXSITypeFillsTheSlot(t *testing.T) {
	str := func(line int) Attribute {
		return icAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}, "xs:string", line)
	}
	schema := icLaxFieldSchema(t, icAnyItem, "b")
	got, undecided := assessRecorded(t, schema, icRoot(icBItem(2, str(3))))
	wantSilence(t, got, "<b> is governed by xs:string")
	wantDeclines(t, icDeclines(undecided))

	got, undecided = assessRecorded(t, schema, icRoot(icBItem(2, str(3)), icBItem(4, str(5))))
	icWantCharges(t, got, icCharge(ruleCvcIdentityConstraint, 4))
	if len(got) == 1 && !strings.Contains(got[0].Error(), "clause 4.1 ") {
		t.Errorf("Violations()[0] = %q, want it charged under clause 4.1", got[0].Error())
	}
	wantDeclines(t, icDeclines(undecided))
}

// An attribute a ***lax*** {attribute wildcard} attributes, whose ·expanded
// name· ·resolves· to no top-level declaration, has no ·governing attribute
// declaration· (key-governing-ad clause 3) and so an ·absent· ·governing type
// definition·: a field `@b` selecting it is charged under clause 3 at the
// attribute, with nothing recorded undecided. A ***skip*** wildcard's @b is
// ·skipped· instead, leaving the unique's ·key-sequence· short and the document
// valid; and an <item> whose {type table} selection is declined declines the
// slot at the attribute, its type being undetermined.
func TestAnAttributeWithNoGoverningDeclarationIsChargedUnderClause3(t *testing.T) {
	b := func() []Attribute { return []Attribute{icAttr(xsd.QName{Local: "b"}, "x", 2)} }
	item := func() *testElement { return icElem(xsd.QName{Local: "item"}, 2, b()) }

	got, undecided := assessRecorded(t, icLaxFieldSchema(t, icLaxItem, "@b"), icRoot(item()))
	icWantAbsentCharge(t, got, undecided, "@b", loc(2, 2))

	got, undecided = assessRecorded(t, icLaxFieldSchema(t, icSkipItem, "@b"), icRoot(item()))
	wantSilence(t, got, "a ·skipped· @b leaves U's ·key-sequence· short")
	wantDeclines(t, icDeclines(undecided))

	got, undecided = assessRecorded(t, icLaxFieldSchema(t, icTabledItem, "@b"), icRoot(item()))
	wantSilence(t, got, "<item>'s type was not determined")
	wantDeclines(t, icDeclines(undecided),
		Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(2, 2), msg: "its ·governing type definition· could not be determined"})
}

// An attribute named by one of the four §3.2.7 Built-in Attribute Declarations
// is always governed by it (key-governing-ad's Note), so a schema assembled
// through [xsd.SchemaBuilder.Finalize], which seeds none of them, leaves its
// type undetermined and not ·absent·: `@*` over an <item> carrying
// xsi:noNamespaceSchemaLocation beside a lax wildcard declines the slot at the
// attribute and charges nothing (attributeTyping's GAP).
func TestABuiltInAttributeWithNoDeclarationDeclines(t *testing.T) {
	unique := icDef(t, "U", xsd.IdentityConstraintUnique, "item", nil, "", "@*")
	nnsl := icAttr(xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "noNamespaceSchemaLocation"}, "x.xsd", 2)
	got, undecided := assessRecorded(t, icWildcardSchema(t, xsd.ProcessLax, []xsd.IdentityConstraint{unique}),
		icRoot(icElem(xsd.QName{Local: "item"}, 2, []Attribute{nnsl})))
	wantSilence(t, got, "the built-in declaration is not one this schema holds")
	wantDeclines(t, icDeclines(undecided),
		Unevaluated{rule: ruleCvcIdentityConstraint, loc: loc(2, 2), msg: "its ·governing type definition· could not be determined"})
}

// icAbsentHeadSchema declares <root> over c*, c's {type definition} naming the
// ·absent· substitution group head absentHead, with ics on <root>.
func icAbsentHeadSchema(t *testing.T, ics []xsd.IdentityConstraint) *xsd.Schema {
	t.Helper()
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: local("RootType")})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	c, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("c"), absentHead, nil, scope,
		nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the c element declaration: %v", err)
	}
	root, err := xsd.NewElementDeclaration(xsderr.Loc{}, local("root"), xsd.TypeDefinitionRef{Name: local("RootType")},
		nil, xsd.NewGlobalScope(), nil, false, ics, nil, nil, false, nil)
	if err != nil {
		t.Fatalf("building the root element declaration: %v", err)
	}
	b := xsd.NewSchemaBuilder()
	for _, st := range ctaBuiltins(t) {
		b.AddType(st)
	}
	b.AddType(icComplex(t, "RootType", nil, icContent(t, icRepeated(t, c))))
	b.AddElement(root)
	for _, ic := range ics {
		b.AddIdentityConstraint(ic)
	}
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the absent-head schema: %v", err)
	}
	return schema
}

// A declared field node whose ·selected type definition· is ·absent· is
// charged under cvc-elt clause 1 and ·laxly assessed· (§5.3), so it too has an
// ·absent· ·governing type definition·: U is charged clause 3 at <c> beside
// that charge, and nothing is recorded undecided.
func TestAFieldNodeLaxlyAssessedUnderSection53IsChargedUnderClause3(t *testing.T) {
	unique := icDef(t, "U", xsd.IdentityConstraintUnique, ".", nil, "", "c")
	c := icElem(local("c"), 2, nil, TextChild(&testText{data: "x", loc: loc(2, 4)}))
	got, undecided := assessRecorded(t, icAbsentHeadSchema(t, []xsd.IdentityConstraint{unique}), icRoot(c))
	icWantCharges(t, got, icCharge(ruleCvcElt, 2), icCharge(ruleCvcIdentityConstraint, 2))
	wantDeclines(t, icDeclines(undecided))
}
