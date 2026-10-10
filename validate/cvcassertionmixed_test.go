package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive an assertion {test} that reads a child of mixed
// content for its value: xpath-datamodel §6.2.4 makes its typed value its
// string-value, the concatenation of the text of all its descendants, as
// xs:untypedAtomic. RootType's children are body, of xs:anyType (no type
// attribute) and {nillable}; um, of the user-defined mixed type UserMixed;
// and dm, of UserMixed with a {value constraint} default. UserMixed holds
// optional i children of xs:anyType, an s of xs:int, an eo of the element-only
// type EO and a skip wildcard over other namespaces, and asserts
// `not(i = 'never')`, which reads its own i children's string-values.

// mxSchema builds the schema whose RootType asserts test.
func mxSchema(t *testing.T, test string) *xsd.Schema {
	t.Helper()
	return parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="EO">
    <xs:sequence><xs:element name="i" minOccurs="0"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="UserMixed" mixed="true">
    <xs:sequence>
      <xs:element name="i" minOccurs="0" maxOccurs="unbounded"/>
      <xs:element name="s" type="xs:int" minOccurs="0"/>
      <xs:element name="eo" type="EO" minOccurs="0"/>
      <xs:any namespace="##other" processContents="skip" minOccurs="0"/>
    </xs:sequence>
    <xs:assert test="not(i = 'never')"/>
  </xs:complexType>
  <xs:complexType name="RootType">
    <xs:sequence>
      <xs:element name="body" nillable="true" minOccurs="0" maxOccurs="unbounded"/>
      <xs:element name="um" type="UserMixed" minOccurs="0"/>
      <xs:element name="dm" type="UserMixed" default="dflt" minOccurs="0"/>
    </xs:sequence>
    <xs:assert test="` + test + `"/>
  </xs:complexType>
  <xs:element name="root" type="RootType"/>
</xs:schema>`})
}

// mxEl is the element named name at line over kids, carrying attrs, with the xs
// prefix bound for an xsi:type.
func mxEl(name string, line int, attrs []Attribute, kids ...Child) Child {
	return ElementChild(&testElement{name: local(name), attrs: attrs, kids: kids, loc: loc(line, 3),
		bindings: map[string]string{"xs": xsd.XMLSchemaNS}})
}

// mxText is a character run at line.
func mxText(data string, line int) Child { return TextChild(&testText{data: data, loc: loc(line, 9)}) }

// mxNil is xsi:nil="true".
var mxNil = &testAttribute{name: xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}, value: "true", loc: loc(2, 6)}

// mxCharged is the opening every cvc-assertion charge at <root> carries.
const mxCharged = "the element root is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is "

// A child of mixed content is read as its string-value, xs:untypedAtomic
// (xpath-datamodel §6.2.4): `string-length(body)` counts the characters of
// one body, is 0 for none, and is charged over two, err:XPTY0004 against
// fn:string-length's xs:string? (xpath20.md §3.1.5). The string-value spans
// the text of every DESCENDANT, at any depth, not only body's own runs:
// <body>he<b>l<c>l</c></b>o</body> is "hello". um's type is a user-defined
// mixed type, read the same way: the compile admits on the {variety}, never
// the name xs:anyType. Inside um, eo's element-only content contributes its
// descendants' text and drops its white-space runs (§6.7.4), and an i, which
// UserMixed's own assertion reads and so opens a frame of its own, hands its
// text up to um's. A general comparison, a cast and fn:string read it
// untyped. A ·nilled· body is the empty sequence (xpath20.md §2.5.2 item
// 4.1): `string-length(body) eq 0` holds and `body = ”` is charged. With
// walk.childValue reading no type as mixed, every row but `no body` declines
// instead, and fails; with assertionAncestry.below handing a frame to the read
// child's own runs alone, the descendant, element-only and cast rows fail;
// with assertionAncestry.collectText appending a run to the innermost open
// frame alone, the user-mixed row, whose i's frame is innermost, fails; with
// it appending every run whatever its element's content, the element-only row
// fails; and with a ·nilled· mixed child bound as Untyped(""), the `body = ”`
// row over one is satisfied instead. An s of simple type whose raw characters
// are its [schema normalized value] contributes them (collectEnd): with that
// run dropped, the simple-descendant row is charged.
func TestAssertionReadsMixedChildStringValue(t *testing.T) {
	for _, tc := range []struct {
		why, test string
		kids      []Child
		charged   bool
	}{
		{"one body", "string-length(body) eq 5", []Child{mxEl("body", 2, nil, mxText("hello", 2))}, false},
		{"test6's", "string-length(body) gt 0", []Child{mxEl("body", 2, nil, mxText("test message..", 2))}, false},
		{"an empty body", "string-length(body) gt 0", []Child{mxEl("body", 2, nil)}, true},
		{"d4_3_15v06's", "string-length(body) eq 0", []Child{mxEl("body", 2, nil)}, false},
		{"no body", "string-length(body) eq 0", nil, false},
		{"two bodies", "string-length(body) ge 0", []Child{mxEl("body", 2, nil, mxText("a", 2)), mxEl("body", 3, nil, mxText("b", 3))}, true},
		{"descendants", "body = 'hello'", []Child{mxEl("body", 2, nil, mxText("he", 2),
			mxEl("b", 3, nil, mxText("l", 3), mxEl("c", 4, nil, mxText("l", 4))), mxText("o", 5))}, false},
		{"own runs alone are not it", "body = 'heo'", []Child{mxEl("body", 2, nil, mxText("he", 2),
			mxEl("b", 3, nil, mxText("l", 3), mxEl("c", 4, nil, mxText("l", 4))), mxText("o", 5))}, true},
		{"a user mixed type", "um = 'abcd'", []Child{mxEl("um", 2, nil, mxText("ab", 2), mxEl("i", 3, nil, mxText("cd", 3)))}, false},
		{"element-only content inside", "um = 'abc'", []Child{mxEl("um", 2, nil, mxText("a", 2),
			mxEl("eo", 3, nil, mxText("\n  ", 3), mxEl("i", 4, nil, mxText("b", 4)), mxText("\n", 5)), mxText("c", 6))}, false},
		{"a simple descendant", "um = 'x5'", []Child{mxEl("um", 2, nil, mxText("x", 2), mxEl("s", 3, nil, mxText("5", 3)))}, false},
		{"a cast", "xs:integer(body) eq 42", []Child{mxEl("body", 2, nil, mxText(" 4", 2), mxEl("b", 3, nil, mxText("2 ", 3)))}, false},
		{"fn:string", "string(body) eq 'hello'", []Child{mxEl("body", 2, nil, mxText("hello", 2))}, false},
		{"a nilled body's length", "string-length(body) eq 0", []Child{mxEl("body", 2, []Attribute{mxNil})}, false},
		{"a nilled body is no zero-length string", "body = ''", []Child{mxEl("body", 2, []Attribute{mxNil})}, true},
		{"an empty body is one", "body = ''", []Child{mxEl("body", 2, nil)}, false},
	} {
		t.Run(tc.why, func(t *testing.T) {
			res := aAssess(t, mxSchema(t, tc.test), acRoot(tc.kids...))
			if !tc.charged {
				wantSatisfied(t, res, tc.test)
				return
			}
			wantAssertionCharge(t, res, mxCharged+`"`+tc.test+`",`)
		})
	}
}

// A mixed child whose string-value this package does not read DECLINES every
// assertion of <root>, never charged and never satisfied: a body an xsi:type
// gives a type with no mixed content — xs:int, or the element-only EO — whose
// typed value is not the string-value the compile read (walk.childValue); a um
// holding a ·skipped· element (assertionAncestry.skipped), an s of simple type
// whose raw characters " 5" are not its [schema normalized value] "5", or whose
// one run is white space alone, either of which its Text Node may hold
// (xpath-datamodel Appendix J.2) (collectEnd, descendantRun), or a dm that took
// its {value constraint} default (cvc-elt clause 5.1), which the data model
// may or may not make a Text Node. Each row's {test} is one the value would
// satisfy were it read as the text alone. The xsi:type rows are guards: with
// walk.childValue's no-mixed-content check removed they still decline, through
// its nil-frame branch, and fail on the message alone. The other four are
// satisfied instead with their decline removed.
func TestAssertionOverAnUnreadableMixedChildIsDeclined(t *testing.T) {
	skipped := ElementChild(&testElement{name: xsd.QName{Space: "urn:other", Local: "z"}, loc: loc(3, 3),
		kids: []Child{mxText("y", 3)}})
	for _, tc := range []struct {
		why, test string
		kids      []Child
		want      string
	}{
		{"an xsi:type to xs:int", "string-length(body) eq 1", []Child{mxEl("body", 2, []Attribute{xsiTypeAttr("xs:int")}, mxText("5", 2))},
			"the child element body of the element root at instance.xml:2:3, which a {test} reads, has no typed value for the data model instance cvc-assertion clause 1 builds: its ·governing type definition· {http://www.w3.org/2001/XMLSchema}int has no mixed content"},
		{"an xsi:type to an element-only type", "string-length(body) eq 0", []Child{mxEl("body", 2, []Attribute{xsiTypeAttr("EO")})},
			"its ·governing type definition· EO has no mixed content"},
		{"a skipped descendant", "um = 'x'", []Child{mxEl("um", 2, nil, mxText("x", 2), skipped)},
			"the element {urn:other}z at instance.xml:3:3 below it is ·skipped·"},
		{"a simple descendant", "um = 'x'", []Child{mxEl("um", 2, nil, mxText("x", 2), mxEl("s", 3, nil, mxText(" 5", 3)))},
			`the element s at instance.xml:3:3 below it has simple content whose raw characters " 5" are not its [schema normalized value]`},
		{"a simple descendant's white space", "um = 'x'", []Child{mxEl("um", 2, nil, mxText("x", 2), mxEl("s", 3, nil, mxText(" ", 3)))},
			"a character run at instance.xml:3:9 below it is white space alone in an element of simple content"},
		{"a defaulted mixed child", "dm = ''", []Child{mxEl("dm", 2, nil)},
			"the element dm at instance.xml:2:3, of mixed content, took the default of its {value constraint}"},
	} {
		t.Run(tc.why, func(t *testing.T) {
			acDeclined(t, aAssess(t, mxSchema(t, tc.test), acRoot(tc.kids...)), tc.want)
		})
	}
}
