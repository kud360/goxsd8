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
// model group under a simple-content one — so what declines them is each
// container's own vocabulary and not some cruder namespace test. Since #1047
// checkS4SChildOrder rejects the document over each of them; the census is taken
// before the first pass that can fail, so what it holds is unchanged (censusOf).
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

// TestCensusNamedGroupBody pins the named <group> region (§3.7.2):
// buildDefinitionModelGroup reads the compositor child and nothing else, so a
// name xs:namedGroup does not admit is a silence — but only where a compositor
// is there to be read. With none, rejectNamedGroupBody charges that very child,
// and a census claiming it would call a named fault a silent skip.
//
// A body carrying TWO compositors is refused outright since #1048, so the walk
// stops there as well: reporting the <attribute> beside them would call a
// construct unmapped in a document the producer refuses to map at all.
func TestCensusNamedGroupBody(t *testing.T) {
	withBody := censusOf(t, `<xs:group name="g">`+
		`<xs:annotation><xs:documentation>d</xs:documentation></xs:annotation>`+
		`<xs:sequence/>`+
		`<xs:attribute name="a" type="xs:string"/>`+
		`</xs:group>`)
	assertCensus(t, withBody, []string{"attribute"})

	twoCompositors := censusOf(t, `<xs:group name="g">`+
		`<xs:sequence/>`+
		`<xs:choice/>`+
		`<xs:attribute name="a" type="xs:string"/>`+
		`</xs:group>`)
	assertCensus(t, twoCompositors, nil)

	noBody := censusOf(t, `<xs:group name="g"><xs:attribute name="a" type="xs:string"/></xs:group>`)
	assertCensus(t, noBody, nil)
}

// TestCensusAttributeGroupBody pins the top-level <attributeGroup> region
// (§3.6.2). Its three attribute names go through the same
// collectAttributeContent a complex type's tail does, but xs:namedAttributeGroup
// has no xs:assertions position and assertionsOf is reached from a complex type
// alone, so an <assert> child of a group maps to nothing — the one place the
// tail vocabulary may not be reused whole.
func TestCensusAttributeGroupBody(t *testing.T) {
	got := censusOf(t, `<xs:attributeGroup name="ag">`+
		`<xs:annotation><xs:documentation>d</xs:documentation></xs:annotation>`+
		`<xs:attribute name="a" type="xs:string"/>`+
		`<xs:attributeGroup ref="other"/>`+
		`<xs:anyAttribute/>`+
		`<xs:assert test="true()"/>`+
		`<xs:element name="e" type="xs:string"/>`+
		`</xs:attributeGroup>`+
		`<xs:attributeGroup name="other"/>`)
	assertCensus(t, got, []string{"assert", "element"})
}

// TestCensusModelGroupReportsNothingItselfAndDescends pins both halves of the
// model-group region: groupParticles REJECTS every name it has no arm for, so no
// child of an <all>/<choice>/<sequence> is ever a silence, while the anonymous
// complex type a local <element> owns is censused at whatever depth it sits.
func TestCensusModelGroupReportsNothingItselfAndDescends(t *testing.T) {
	got := censusOf(t, `<xs:complexType name="ct"><xs:sequence>`+
		`<xs:any/>`+
		`<xs:choice>`+
		`<xs:element name="e"><xs:complexType><xs:key name="k"/></xs:complexType></xs:element>`+
		`</xs:choice>`+
		`</xs:sequence></xs:complexType>`)
	assertCensus(t, got, []string{"key"})
}

// TestCensusSimpleTypeAlternatives pins the <list> and <union> vocabularies
// (§3.16.2.1): each carries the one inline <simpleType> position listItem and
// unionMembers read, and any other name is dropped in silence.
func TestCensusSimpleTypeAlternatives(t *testing.T) {
	got := censusOf(t, `<xs:simpleType name="l"><xs:list>`+
		`<xs:annotation><xs:documentation>d</xs:documentation></xs:annotation>`+
		`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`+
		`<xs:length value="3"/>`+
		`</xs:list></xs:simpleType>`+
		`<xs:simpleType name="u"><xs:union>`+
		`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`+
		`<xs:pattern value="."/>`+
		`</xs:union></xs:simpleType>`)
	assertCensus(t, got, []string{"length", "pattern"})
}

// TestCensusSimpleTypeRestrictionSilent pins the <restriction> alternative
// reporting nothing of its own. rejectOutOfModelFacetChildren (#972) refuses
// every XSD-namespace child Datatypes §4.1.2 has no position for, so the spec's
// own silent-non-mapping carve-out — map.std.common case 2, an unrecognized
// facet making the whole <simpleType> map to no component without being in error
// — is unreachable here: the document is rejected instead. The census must not
// claim what a verdict already names.
//
// A <simpleType> naming none of the three alternatives, or two, stops the walk
// for the same reason: simpleTypeBody answers neither, and the producer refuses
// the document either way — none under src-simple-type (§3.16.3), two as a repeat
// of the single alternative position s4sSimpleType gives them (#1076).
func TestCensusSimpleTypeRestrictionSilent(t *testing.T) {
	outOfModel := censusOf(t, `<xs:simpleType name="s"><xs:restriction base="xs:string">`+
		`<xs:period/>`+
		`</xs:restriction></xs:simpleType>`)
	assertCensus(t, outOfModel, nil)

	noAlternative := censusOf(t, `<xs:simpleType name="s"><xs:attribute name="a"/></xs:simpleType>`)
	assertCensus(t, noAlternative, nil)

	twoAlternatives := censusOf(t, `<xs:simpleType name="s">`+
		`<xs:restriction base="xs:string"/>`+
		`<xs:list itemType="xs:string"><xs:length value="3"/></xs:list>`+
		`</xs:simpleType>`)
	assertCensus(t, twoAlternatives, nil)
}

// TestCensusReachesInlineSimpleTypeAtEveryOwner pins the descent into the simple
// types the census must reach to censusing them at all: the one a local
// <attribute> in a derivation tail owns, the one an <element>'s <alternative>
// owns, and the base a <simpleContent> <restriction> synthesizes from. Each
// carries a <list> holding a name nothing maps, so a missed descent shows up as
// a missing report rather than as a shape nobody notices.
func TestCensusReachesInlineSimpleTypeAtEveryOwner(t *testing.T) {
	inlineList := `<xs:simpleType><xs:list itemType="xs:string"><xs:field xpath="."/></xs:list></xs:simpleType>`
	got := censusOf(t, `<xs:complexType name="ct">`+
		`<xs:sequence><xs:element name="e"><xs:complexType>`+
		`<xs:attribute name="deep">`+inlineList+`</xs:attribute>`+
		`</xs:complexType></xs:element></xs:sequence>`+
		`<xs:attribute name="a">`+inlineList+`</xs:attribute>`+
		`</xs:complexType>`+
		`<xs:complexType name="sc"><xs:simpleContent><xs:restriction base="xs:string">`+
		inlineList+
		`</xs:restriction></xs:simpleContent></xs:complexType>`+
		`<xs:element name="top"><xs:alternative test="true()">`+inlineList+`</xs:alternative></xs:element>`+
		`<xs:attribute name="ta">`+inlineList+`</xs:attribute>`)
	assertCensus(t, got, []string{"field", "field", "field", "field", "field"})
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
