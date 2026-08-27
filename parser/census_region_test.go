package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/parser"
)

// censusOf assembles a one-document schema and returns its census as local
// names. The parse OUTCOME is deliberately not asserted: the census is taken
// before the first pass that can fail, so it is complete for a single document
// whether or not the document goes on to be rejected, and pinning a vocabulary
// must not turn on whether the fixture happens to assemble.
func censusOf(t *testing.T, body string) []string {
	t.Helper()
	report, _ := reportOf(t, "main.xsd", map[string]string{"main.xsd": wrap("urn:a", body)})
	if len(report.Documents()) != 1 {
		t.Fatalf("Documents() = %d documents, want 1", len(report.Documents()))
	}
	return unmappedNames(report.Documents()[0])
}

// assertCensus compares a census against the exact ordered names expected.
func assertCensus(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("census = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("census = %v, want %v", got, want)
		}
	}
}

// TestCensusDerivationTail pins the vocabulary every complex-type derivation
// alternant ends with (§3.4.2): each of <attribute>, <attributeGroup>,
// <anyAttribute> and <assert> is mapped at all five containers, and a name
// outside the container's own vocabulary is reported at each of them.
//
// The four unmapped children are chosen to be names the schema for schema
// documents admits SOMEWHERE — a facet under a complex-content alternant, a
// model group under a simple-content one — because those are the ones
// checkS4SChildOrder passes over rather than rejecting, and so the ones that
// reach a component-less silence.
func TestCensusDerivationTail(t *testing.T) {
	got := censusOf(t, `<xs:complexType name="implicit">`+
		`<xs:sequence/>`+
		`<xs:attribute name="a" type="xs:string"/>`+
		`<xs:anyAttribute/>`+
		`<xs:key name="k"/>`+
		`</xs:complexType>`+
		`<xs:complexType name="sext"><xs:simpleContent><xs:extension base="xs:string">`+
		`<xs:choice/>`+
		`<xs:attribute name="b" type="xs:string"/>`+
		`</xs:extension></xs:simpleContent></xs:complexType>`+
		`<xs:complexType name="srest"><xs:simpleContent><xs:restriction base="sext">`+
		`<xs:length value="3"/>`+
		`<xs:period/>`+
		`<xs:attribute name="c" type="xs:string"/>`+
		`</xs:restriction></xs:simpleContent></xs:complexType>`+
		`<xs:complexType name="cext"><xs:complexContent><xs:extension base="implicit">`+
		`<xs:sequence/>`+
		`<xs:maxLength value="3"/>`+
		`<xs:attributeGroup ref="ag"/>`+
		`<xs:assert test="true()"/>`+
		`</xs:extension></xs:complexContent></xs:complexType>`+
		`<xs:complexType name="crest"><xs:complexContent><xs:restriction base="implicit">`+
		`<xs:field xpath="."/>`+
		`</xs:restriction></xs:complexContent></xs:complexType>`+
		`<xs:attributeGroup name="ag"/>`)
	assertCensus(t, got, []string{"key", "choice", "period", "maxLength", "field"})
}

// TestCensusDerivationAlternantMappedChildrenSilent pins the other half: a
// complex type whose every child some pass reads reports nothing. Without it the
// vocabularies could satisfy the test above by declining everything.
//
// <openContent> and the model group are the two structural positions only the
// complex-content containers carry, and <simpleType> and the facets are the two
// only the simple-content <restriction> does.
func TestCensusDerivationAlternantMappedChildrenSilent(t *testing.T) {
	got := censusOf(t, `<xs:complexType name="implicit">`+
		`<xs:annotation><xs:documentation>d</xs:documentation></xs:annotation>`+
		`<xs:openContent mode="suffix"><xs:any namespace="urn:z"/></xs:openContent>`+
		`<xs:sequence/>`+
		`<xs:attribute name="a" type="xs:string"/>`+
		`<xs:attributeGroup ref="ag"/>`+
		`<xs:anyAttribute/>`+
		`<xs:assert test="true()"/>`+
		`</xs:complexType>`+
		`<xs:complexType name="srest"><xs:simpleContent><xs:restriction base="xs:string">`+
		`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`+
		`<xs:enumeration value="x"/>`+
		`<xs:assertion test="true()"/>`+
		`<xs:attribute name="c" type="xs:string"/>`+
		`<xs:assert test="true()"/>`+
		`</xs:restriction></xs:simpleContent></xs:complexType>`+
		`<xs:attributeGroup name="ag"/>`)
	assertCensus(t, got, nil)
}

// TestCensusInlineComplexTypeReached pins the descent: an anonymous complex type
// an <element> owns (§3.3.2.1 dcl.elt.common clause 1) is censused exactly as a
// top-level one is, at global scope and under an <alternative> alike (§3.12.2
// declare-ta).
func TestCensusInlineComplexTypeReached(t *testing.T) {
	got := censusOf(t, `<xs:element name="e">`+
		`<xs:complexType><xs:sequence/><xs:key name="k"/></xs:complexType>`+
		`<xs:alternative test="true()"><xs:complexType><xs:unique name="u"/></xs:complexType></xs:alternative>`+
		`</xs:element>`)
	assertCensus(t, got, []string{"key", "unique"})
}

// TestCensusStopsAtRepeatedAlternative pins the two shapes the walk stops at,
// both rejections rather than silences: a <complexType> carrying two content
// alternatives (repeatedContentAlternative) and a wrapper carrying two
// derivation alternants (repeatedDerivationAlternant). Reporting under either
// would call a construct unmapped in a document the producer refuses to map at
// all.
func TestCensusStopsAtRepeatedAlternative(t *testing.T) {
	got := censusOf(t, `<xs:complexType name="two">`+
		`<xs:simpleContent><xs:extension base="xs:string"><xs:choice/></xs:extension></xs:simpleContent>`+
		`<xs:complexContent><xs:restriction base="xs:anyType"><xs:field xpath="."/></xs:restriction></xs:complexContent>`+
		`</xs:complexType>`+
		`<xs:complexType name="bothAlternants"><xs:complexContent>`+
		`<xs:restriction base="xs:anyType"><xs:field xpath="."/></xs:restriction>`+
		`<xs:extension base="xs:anyType"><xs:field xpath="."/></xs:extension>`+
		`</xs:complexContent></xs:complexType>`)
	assertCensus(t, got, nil)
}

// TestCensusReasonIsNoDispatch pins that a construct found below the top level
// carries the same closed-set reason a top-level one does: the position it stood
// at has no arm for its name, which is what UnmappedNoDispatch states.
func TestCensusReasonIsNoDispatch(t *testing.T) {
	report, _ := reportOf(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:complexType name="ct"><xs:key name="k"/></xs:complexType>`),
	})
	d := report.Documents()[0]
	if len(d.Unmapped) != 1 {
		t.Fatalf("Unmapped = %v, want exactly one construct", unmappedNames(d))
	}
	if d.Unmapped[0].Reason != parser.UnmappedNoDispatch {
		t.Errorf("Reason = %s, want %s", d.Unmapped[0].Reason, parser.UnmappedNoDispatch)
	}
	if d.Unmapped[0].At.URI != "main.xsd" {
		t.Errorf("At = %s, want a position in main.xsd", d.Unmapped[0].At)
	}
}
