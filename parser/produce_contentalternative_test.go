package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ctWithAlternatives builds a schema whose one <complexType> carries first and
// second as consecutive children, each on a line of its own, so a rejection can
// be asserted to name the SECOND one: the <complexType> opens at line 2 column 1,
// first at line 3 column 1 and second at line 4 column 1.
func ctWithAlternatives(first, second string) string {
	return wrap("urn:x", "\n<xs:complexType name=\"D\">\n"+first+"\n"+second+"\n</xs:complexType>")
}

const (
	simpleContentExt  = `<xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent>`
	complexContentExt = `<xs:complexContent><xs:extension base="xs:anyType"><xs:sequence/></xs:extension></xs:complexContent>`
)

// TestProduceRepeatedContentAlternativeRejected pins the xs:complexTypeModel
// (§3.4.2) cardinality: the group is a plain xs:choice, so a <complexType>
// carrying more than one of <simpleContent>/<complexContent> is a grammar fault
// in all four orderings. Before this check produceComplexType dispatched on a
// first-match childElement read and silently produced from whichever wrapper came
// first, accepting every one of these documents.
//
// The fault carries no rule ID (§5.1's first bullet, STYLE E2), and it names both
// positions: the second alternative, which is the item to delete, and the
// <complexType> whose content model bounds it.
func TestProduceRepeatedContentAlternativeRejected(t *testing.T) {
	for _, tc := range []struct {
		name          string
		first, second string
		wantLocal     string
	}{
		{"two simpleContent", simpleContentExt, simpleContentExt, "simpleContent"},
		{"simpleContent then complexContent", simpleContentExt, complexContentExt, "complexContent"},
		{"complexContent then simpleContent", complexContentExt, simpleContentExt, "simpleContent"},
		{"two complexContent", complexContentExt, complexContentExt, "complexContent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, ctWithAlternatives(tc.first, tc.second))
			if err == nil {
				t.Fatal("Produce accepted a <complexType> carrying two content alternatives")
			}
			if _, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, want a plain grammar fault rather than a rule verdict", err)
			}
			if !strings.Contains(err.Error(), "<"+tc.wantLocal+"> at "+produceURI+":4:1") {
				t.Errorf("error = %v, want it to name the second <%s> at line 4", err, tc.wantLocal)
			}
			if !strings.Contains(err.Error(), "<complexType> at "+produceURI+":2:1") {
				t.Errorf("error = %v, want it to name the enclosing <complexType> at line 2", err)
			}
		})
	}
}

// TestProduceRepeatedContentAlternativeBeatsMissingAlternant pins ctB006's shape,
// where the two faults compete: a <simpleContent> <extension> followed by a
// <complexContent> carrying neither <restriction> nor <extension>. The
// cardinality check runs ahead of the dispatch, so the repeated alternative is
// what is reported — the missing-alternant charge (#868) never sees the second
// wrapper, because under the old first-match dispatch the <simpleContent> won and
// the document was accepted outright.
func TestProduceRepeatedContentAlternativeBeatsMissingAlternant(t *testing.T) {
	_, err := produce(t, ctWithAlternatives(simpleContentExt, `<xs:complexContent><xs:sequence/></xs:complexContent>`))
	if err == nil {
		t.Fatal("Produce accepted ctB006's shape")
	}
	if !strings.Contains(err.Error(), "is a second content alternative") {
		t.Fatalf("error = %v, want the repeated-alternative fault", err)
	}
	if strings.Contains(err.Error(), "has neither a <restriction> nor an <extension> child") {
		t.Errorf("error = %v, want the repeated alternative charged rather than the missing alternant inside it", err)
	}
}

// TestProduceSingleContentAlternativeAccepted is the regression guard on the
// check's other side: one wrapper, or none, is the well-formed case and still
// produces.
func TestProduceSingleContentAlternativeAccepted(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"simpleContent alone", `<xs:complexType name="D">` + simpleContentExt + `</xs:complexType>`},
		{"complexContent alone", `<xs:complexType name="D">` + complexContentExt + `</xs:complexType>`},
		{"implicit content", `<xs:complexType name="D"><xs:sequence/></xs:complexType>`},
		// An <annotation> beside the one wrapper is not a second alternative.
		{"annotation then complexContent", `<xs:complexType name="D"><xs:annotation/>` + complexContentExt + `</xs:complexType>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:x", tc.body)); err != nil {
				t.Fatalf("Produce: %v", err)
			}
		})
	}
}

// TestProduceContentAlternativeWithBothDerivationsUnchanged pins the behavior
// this check deliberately leaves alone: a SINGLE <simpleContent>/<complexContent>
// carrying BOTH <restriction> and <extension> is still accepted, mapped by one
// alternant with the other silently dropped. That is a distinct grammar fault —
// xs:simpleContent and xs:complexContent each hold a (restriction | extension)
// choice of their own — and no document in the W3C suite exercises it, so it is
// left to an issue of its own rather than absorbed here.
//
// The two halves drop opposite children, which is why each is asserted: a
// <simpleContent> maps by its <extension> (produceSimpleContent short-circuits on
// it), a <complexContent> by its <restriction> (complexContentDerivation looks
// for that one first).
func TestProduceContentAlternativeWithBothDerivationsUnchanged(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   string
		method xsd.DerivationMethod
	}{
		{
			"simpleContent maps by its extension",
			`<xs:complexType name="D"><xs:simpleContent>` +
				`<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction>` +
				`<xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`,
			xsd.DerivationExtension,
		},
		{
			"complexContent maps by its restriction",
			`<xs:complexType name="D"><xs:complexContent>` +
				`<xs:restriction base="xs:anyType"><xs:sequence/></xs:restriction>` +
				`<xs:extension base="xs:anyType"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`,
			xsd.DerivationRestriction,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, wrap("urn:x", tc.body))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			td, ok := s.Type(xq("D"))
			if !ok {
				t.Fatal("complex type D not found")
			}
			ct, ok := td.(xsd.ComplexType)
			if !ok {
				t.Fatalf("type D is not a complex type (%T)", td)
			}
			if ct.DerivationMethod() != tc.method {
				t.Errorf("{derivation method} = %v, want %v", ct.DerivationMethod(), tc.method)
			}
		})
	}
}
