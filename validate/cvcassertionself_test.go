package validate

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive an assertion {test} that reads `.` on an element
// whose {content type} is element-only: E is annotated xs:anyType in the
// partial ·PSVI· (cvc-assertion clause 1.2), so `.` atomizes to its string
// value, the concatenation of its Text Node descendants (xpath-datamodel
// §6.2.4). RootType's children are any number of d of xs:date, as in the
// suite's assert016; an n of xs:int; an eo of the element-only type EO,
// holding one d; an m of xs:anyType; a dflt of xs:string defaulting to "x";
// and a skip wildcard over other namespaces.

// selfSchema builds the schema whose RootType asserts test.
func selfSchema(t *testing.T, test string) *xsd.Schema {
	t.Helper()
	return parsedSchema(t, map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="EO">
    <xs:sequence><xs:element name="d" type="xs:date"/></xs:sequence>
  </xs:complexType>
  <xs:complexType name="RootType">
    <xs:sequence>
      <xs:element name="d" type="xs:date" minOccurs="0" maxOccurs="unbounded"/>
      <xs:element name="n" type="xs:int" minOccurs="0"/>
      <xs:element name="eo" type="EO" minOccurs="0"/>
      <xs:element name="m" minOccurs="0"/>
      <xs:element name="dflt" type="xs:string" default="x" minOccurs="0"/>
      <xs:any namespace="##other" processContents="skip" minOccurs="0"/>
    </xs:sequence>
    <xs:assert test="` + test + `"/>
  </xs:complexType>
  <xs:element name="root" type="RootType" nillable="true"/>
</xs:schema>`})
}

// selfCharged is the opening every cvc-assertion charge at <root> carries.
const selfCharged = "the element root is not ·valid· with respect to assertion 1 of 1 in the {assertions} of the ·governing type definition· RootType, whose {test} is "

// `.` on an element of element-only content is its string value: the
// suite's assert016.v1 and assert017.v1, `. castable as xs:date` and `data(.)
// instance of xs:untypedAtomic` over a temp whose one d is 2008-07-01 or
// 2010-11-17Z, hold, and `. castable as xs:date` over two d is charged, their
// text concatenated. A descendant of simple content whose raw characters are
// its [schema normalized value] contributes them, so `.` spans d and n; a
// descendant of mixed content contributes its own descendants' text, `. =
// 'ab'` over <m>a<b>b</b></m>. E's own runs are kept whole, white space
// included, E being annotated xs:anyType, while those of an element-only
// descendant that are white space alone are no Text Node (§6.7.4). A
// ·nilled· E with no [[children]] binds `.` to the zero-length string and
// `$value` to the empty sequence.
//
// With walk.assertionValue binding the zero ValueBinding under element-only
// content, every satisfied row is charged instead but assert017's, which
// holds over any string, and the nilled one; with the run of E's own frame
// owner dropped as an element-only one's, the own-white-space row is charged;
// with every run appended to every frame whatever its element's content, the
// element-only-descendant row is charged; and with a simple descendant's run
// dropped, every satisfied row whose `.` spans a d or an n is charged but
// assert017's.
func TestAssertionReadsElementOnlyStringValue(t *testing.T) {
	for _, tc := range []struct {
		why, test string
		root      *testElement
		charged   bool
	}{
		{"assert016's", ". castable as xs:date", acRoot(mxEl("d", 2, nil, mxText("2008-07-01", 2))), false},
		{"assert016's second temp", ". castable as xs:date", acRoot(mxEl("d", 2, nil, mxText("2010-11-17Z", 2))), false},
		{"assert017's", "data(.) instance of xs:untypedAtomic", acRoot(mxEl("d", 2, nil, mxText("2008-07-01", 2))), false},
		{"two dates", ". castable as xs:date", acRoot(mxEl("d", 2, nil, mxText("2008-07-01", 2)), mxEl("d", 3, nil, mxText("2008-07-02", 3))), true},
		{"simple descendants", ". = '2008-07-015'", acRoot(mxEl("d", 2, nil, mxText("2008-07-01", 2)), mxEl("n", 3, nil, mxText("5", 3))), false},
		{"a mixed descendant", ". = 'ab'", acRoot(mxEl("m", 2, nil, mxText("a", 2), mxEl("b", 3, nil, mxText("b", 3)))), false},
		{"its own white space", "string-length(.) = 14 and normalize-space(.) = '2008-07-01'",
			acRoot(mxText("\n  ", 1), mxEl("d", 2, nil, mxText("2008-07-01", 2)), mxText("\n", 2)), false},
		{"an element-only descendant's white space", ". = '2008-07-01'",
			acRoot(mxEl("eo", 2, nil, mxText("\n  ", 2), mxEl("d", 3, nil, mxText("2008-07-01", 3)), mxText("\n", 3))), false},
		{"a nilled E", ". = '' and empty($value)", &testElement{name: local("root"), attrs: []Attribute{mxNil}, loc: loc(1, 1)}, false},
	} {
		t.Run(tc.why, func(t *testing.T) {
			res := aAssess(t, selfSchema(t, tc.test), tc.root)
			if !tc.charged {
				wantSatisfied(t, res, tc.test)
				return
			}
			wantAssertionCharge(t, res, selfCharged+`"`+tc.test+`",`)
		})
	}
}

// An element-only E whose string value spans a descendant whose contribution
// is undecided DECLINES every assertion reading `.`, never charged and never
// satisfied, naming the descendant: a ·skipped· one, whose text is not read
// (assertionAncestry.skipped); an n whose raw characters " 5" are not its
// [schema normalized value] "5", or a dflt whose one run is white space alone,
// either of which its Text Node may hold (xpath-datamodel Appendix J.2)
// (collectEnd, descendantRun); and a dflt that took its {value constraint}
// default, which J.2 makes a Text Node the raw characters do not hold. A
// ·nilled· E with [[children]] has their text as its string value, which the
// walk does not gather (lackingNilledText). Each row is evaluated instead with
// its decline removed, and fails. The guard: a {test} that does not read `.`
// is evaluated over the ·skipped· descendant all the same — `count(d) = 0`,
// and `empty($value)`, which reads `$value` alone, the empty sequence under
// element-only content (cvc-assertion clause 2.3.2). Both are declined with
// assertionCheck.readsContextItem answering true for every compiled {test},
// and the `$value` row alone with ctaAssertionFacade.variable recording a
// read of `.` for `$value`.
func TestAssertionOverAnUndecidedOwnStringValueIsDeclined(t *testing.T) {
	skipped := ElementChild(&testElement{name: xsd.QName{Space: "urn:other", Local: "z"}, loc: loc(3, 3),
		kids: []Child{mxText("y", 3)}})
	const own = "the string value of the element root, which `.` reads in the data model instance cvc-assertion clause 1 builds, is undecided: "
	for _, tc := range []struct {
		why, test string
		root      *testElement
		want      string
	}{
		{"a skipped descendant", ". = 'y'", acRoot(skipped),
			own + "the element {urn:other}z at instance.xml:3:3 below it is ·skipped·"},
		{"a simple descendant", ". = '5'", acRoot(mxEl("n", 2, nil, mxText(" 5", 2))),
			own + `the element n at instance.xml:2:3 below it has simple content whose raw characters " 5" are not its [schema normalized value]`},
		{"a simple descendant's white space", ". = ''", acRoot(mxEl("dflt", 2, nil, mxText(" ", 2))),
			own + "a character run at instance.xml:2:9 below it is white space alone in an element of simple content"},
		{"a defaulted simple descendant", ". = ''", acRoot(mxEl("dflt", 2, nil)),
			own + `the element dflt at instance.xml:2:3 below it, of simple content, took the default "x" of its {value constraint}`},
		{"a nilled E with children", ". = ''", &testElement{name: local("root"), attrs: []Attribute{mxNil}, loc: loc(1, 1),
			kids: []Child{mxEl("d", 2, nil, mxText("2008-07-01", 2))}},
			"the element root has xsi:nil = true and [[children]]"},
	} {
		t.Run(tc.why, func(t *testing.T) {
			acDeclined(t, aAssess(t, selfSchema(t, tc.test), tc.root), tc.want)
		})
	}
	for _, test := range []string{"count(d) = 0", "empty($value)"} {
		wantSatisfied(t, aAssess(t, selfSchema(t, test), acRoot(skipped)), test+" over a skipped descendant")
	}
}
