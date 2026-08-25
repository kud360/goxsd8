package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// facetModelDoc builds a schema whose one <simpleType name="T"> holds lines as
// its subtree, each line on a source line of its own: <schema> is line 1, the
// <simpleType> line 2, and lines[i] line 3+i. Every rejection below names an
// exact line, so a check firing at the wrong child fails here rather than passing
// on the strength of rejecting something.
func facetModelDoc(lines ...string) string {
	return wrap("urn:x", "\n<xs:simpleType name=\"T\">\n"+strings.Join(lines, "\n")+"\n</xs:simpleType>")
}

// TestProduceSimpleTypeRestrictionOutOfModelChildRejected pins §4.1.2's content
// model for a <simpleType>'s <restriction> (xmlschema11-2.md:2748, the group
// xs:simpleRestrictionModel at :3929): an XSD-namespace child it has no position
// for is rejected rather than silently dropped, which is what let the producer
// build a component out of an s4s-invalid document and the schema lane report it
// valid (#972).
//
// The wildcard position is namespace="##other" and so excludes the XSD namespace
// itself: none of these names falls through it. Four of them — <attribute>,
// <attributeGroup>, <anyAttribute> and <assert> — are legal under the OTHER
// <restriction> this producer maps facets from, xs:simpleRestrictionType
// (xmlschema11-1.md:1692), and TestProduceSimpleContentRestrictionAdmitsTail is
// the other side of that seam.
//
// The fault carries NO rule ID: §5.1's first bullet (xmlschema11-1.md:4296) is
// what binds, and charging a src-*/cvc-*/cos-* over it would be a fabricated
// verdict (STYLE E2) — the footing checkS4SChildOrder and rejectProhibitedAttrs
// already stand on.
func TestProduceSimpleTypeRestrictionOutOfModelChildRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name      string
		lines     []string
		wantChild string // "<local> at <uri>:<line>:1"
	}{
		{
			// stF005's own fault: the legal facet is <whiteSpace>, and the lowercase
			// spelling names no element at all.
			name: "misspelled whitespace facet",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:whitespace value="collapse"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<whitespace> at " + produceURI + ":4:1",
		},
		{
			// #561's repro: §4.3.13 spells the FACET "assertions" and no element
			// bears that name — the over-admission s4sFacetElement carries and
			// facetKindOf does not.
			name: "plural assertions",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:assertions test="true()"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<assertions> at " + produceURI + ":4:1",
		},
		{
			// stC012's shape: a builtin type's name written where a facet belongs.
			name: "a type name in the facet position",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:duration value="P1Y"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<duration> at " + produceURI + ":4:1",
		},
		{
			name: "attribute",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:attribute name="a"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<attribute> at " + produceURI + ":4:1",
		},
		{
			name: "attributeGroup",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:attributeGroup ref="tns:AG"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<attributeGroup> at " + produceURI + ":4:1",
		},
		{
			name: "anyAttribute",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:anyAttribute namespace="##other"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<anyAttribute> at " + produceURI + ":4:1",
		},
		{
			// <assert> is the complex-type assertions group's element
			// (xmlschema11-1.md:4743), not the <assertion> facet this position takes.
			name: "assert rather than assertion",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:assert test="true()"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<assert> at " + produceURI + ":4:1",
		},
		{
			name: "a particle",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:sequence/>`,
				`</xs:restriction>`,
			},
			wantChild: "<sequence> at " + produceURI + ":4:1",
		},
		{
			// The FIRST offending child in document order is the reported one, whatever
			// legal facets surround it (STYLE D2).
			name: "reported at the first offender among legal facets",
			lines: []string{
				`<xs:restriction base="xs:string">`,
				`<xs:minLength value="1"/>`,
				`<xs:whitespace value="collapse"/>`,
				`<xs:maxLength value="8"/>`,
				`<xs:assert test="true()"/>`,
				`</xs:restriction>`,
			},
			wantChild: "<whitespace> at " + produceURI + ":5:1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, facetModelDoc(tc.lines...))
			if err == nil {
				t.Fatal("Produce accepted a <simpleType> <restriction> whose child §4.1.2's content model has no position for")
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
			}
			if !strings.Contains(err.Error(), tc.wantChild) {
				t.Errorf("error = %v, want it to name the offending %s", err, tc.wantChild)
			}
			if owner := "<restriction> at " + produceURI + ":3:1"; !strings.Contains(err.Error(), owner) {
				t.Errorf("error = %v, want it to name the owning %s", err, owner)
			}
		})
	}
}

// TestProduceSimpleTypeRestrictionModelAccepted is the other side of the check:
// every position §4.1.2's content model does admit, including the wildcard's own
// ##other child and the two precisionDecimal extension facets
// (xsd-precisionDecimal.md §4.2/§4.3) the 2012 summary predates and this producer
// folds. A membership predicate written off the fourteen names the summary lists
// would falsely reject the last row.
func TestProduceSimpleTypeRestrictionModelAccepted(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: "annotation, inline base and plain-lexical facets",
			body: `<xs:simpleType name="T"><xs:restriction>` +
				`<xs:annotation/><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>` +
				`<xs:minLength value="1"/><xs:maxLength value="8"/><xs:whiteSpace value="collapse"/>` +
				`</xs:restriction></xs:simpleType>`,
		},
		{
			name: "the two folded facets and the singular assertion",
			body: `<xs:simpleType name="T"><xs:restriction base="xs:string">` +
				`<xs:enumeration value="a"/><xs:enumeration value="b"/>` +
				`<xs:pattern value="a|b"/><xs:assertion test="true()"/>` +
				`</xs:restriction></xs:simpleType>`,
		},
		{
			// The wildcard position: namespace="##other" admits any namespace but the
			// XSD one, which is the whole reason an XSD-namespace child is decidable
			// here at all.
			name: "a child in another namespace",
			body: `<xs:simpleType name="T"><xs:restriction base="xs:string">` +
				`<xs:minLength value="1"/><foreign xmlns="urn:elsewhere"/>` +
				`</xs:restriction></xs:simpleType>`,
		},
		{
			name: "the precisionDecimal extension facets",
			body: `<xs:simpleType name="T"><xs:restriction base="xs:precisionDecimal">` +
				`<xs:maxScale value="2"/><xs:minScale value="0"/>` +
				`</xs:restriction></xs:simpleType>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:x", tc.body)); err != nil {
				t.Fatalf("Produce rejected a <restriction> §4.1.2's content model admits: %v", err)
			}
		})
	}
}

// TestProduceSimpleContentRestrictionAdmitsTail is the seam #972 turns on: the
// SAME facet mapping runs under a <simpleContent>'s <restriction>, whose model is
// xs:simpleRestrictionType (xmlschema11-1.md:1692) and ends "((attribute |
// attributeGroup)*, anyAttribute?), assert*". All four names the sibling test
// rejects are legal here — produceAttributeUses and assertionsOf map them off
// this very element — so a membership check inside restrictionFacets rather than
// at the §4.1.2 caller would false-reject this document.
func TestProduceSimpleContentRestrictionAdmitsTail(t *testing.T) {
	_, err := produce(t, wrap("urn:x",
		`<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string">`+
			`<xs:anyAttribute namespace="##any"/></xs:extension></xs:simpleContent></xs:complexType>`+
			`<xs:attributeGroup name="AG"><xs:attribute name="g"/></xs:attributeGroup>`+
			`<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">`+
			`<xs:maxLength value="8"/>`+
			`<xs:attribute name="a"/><xs:attributeGroup ref="tns:AG"/>`+
			`<xs:anyAttribute namespace="##other"/><xs:assert test="true()"/>`+
			`</xs:restriction></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce rejected the attribute tail xs:simpleRestrictionType admits: %v", err)
	}
}
