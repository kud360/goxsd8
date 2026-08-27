package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestS4SGrammarFaultsNameTheirProduction pins the ONE thing every s4s-grammar
// rejection owes its reader beyond the offending item and its location: the
// Appendix A production the schema document violates (xsderr/doc.go, "The
// s4s-grammar class"). The class carries no Rule — §2.4 clause 1 (sd-valid,
// xmlschema11-1.md:615) is anchored but uncataloged — so the production name is
// the only citation these messages can carry, and a message without one leaves
// the author nothing to look up (#975).
//
// Every row asserts BOTH the production and a fragment identifying the fault, so
// a row cannot pass on some other document-level rejection that happens to name
// a production of its own, and every row asserts a PLAIN error: charging any
// src-*/cvc-*/cos-* over this class would be a fabricated verdict (STYLE E2).
//
// The six-kind top-level table lives in TestProduceNamelessTopLevelRejected
// (produce_groups_test.go), which pins topLevelGrammar's whole mapping row by
// row, and produceComplexType's nameless backstop — the one message of the class
// no schema document can draw — in produce_s4sgrammar_internal_test.go.
func TestS4SGrammarFaultsNameTheirProduction(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name        string
		docs        map[string]string
		wantGrammar string // the Appendix A production the message must name
		wantFault   string // a fragment of the fault itself, so no row passes on another
	}{
		{
			name:        "<include> with no schemaLocation",
			docs:        map[string]string{"main.xsd": wrap("urn:a", `<xs:include/>`)},
			wantGrammar: "xs:include",
			wantFault:   "has no schemaLocation attribute",
		},
		{
			name:        "<redefine> with no schemaLocation",
			docs:        map[string]string{"main.xsd": wrap("urn:a", `<xs:redefine/>`)},
			wantGrammar: "xs:redefine",
			wantFault:   "has no schemaLocation attribute",
		},
		{
			name: "<override> with no schemaLocation",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:override><xs:element name="e" type="xs:string"/></xs:override>`)},
			wantGrammar: "xs:override",
			wantFault:   "has no schemaLocation attribute",
		},
		{
			// The four redefinable kinds take the TOP LEVEL's own productions:
			// §4.2.4's content model reaches the same global element declarations
			// through xs:redefinable (xmlschema11-1.md:4465).
			name:        "redefining <simpleType> with no name",
			docs:        redefining(`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`),
			wantGrammar: "xs:topLevelSimpleType",
			wantFault:   "has no name attribute",
		},
		{
			name:        "redefining <complexType> with no name",
			docs:        redefining(`<xs:complexType><xs:sequence/></xs:complexType>`),
			wantGrammar: "xs:topLevelComplexType",
			wantFault:   "has no name attribute",
		},
		{
			name: "redefining <group> with no name",
			docs: redefining(`<xs:group><xs:sequence>` +
				`<xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`),
			wantGrammar: "xs:namedGroup",
			wantFault:   "has no name attribute",
		},
		{
			name: "redefining <attributeGroup> with no name",
			docs: redefining(`<xs:attributeGroup>` +
				`<xs:attribute name="b" type="xs:string"/></xs:attributeGroup>`),
			wantGrammar: "xs:namedAttributeGroup",
			wantFault:   "has no name attribute",
		},
		{
			name: "<annotation> nested in <annotation>",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:annotation><xs:annotation><xs:appinfo>a</xs:appinfo></xs:annotation></xs:annotation>`)},
			wantGrammar: "xs:annotation's content model",
			wantFault:   "<annotation> child of <annotation>",
		},
		{
			name: "<openContent> beside <simpleContent>",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:complexType name="T"><xs:openContent><xs:any/></xs:openContent>`+
					`<xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`)},
			wantGrammar: "xs:openContent",
			wantFault:   "is in a position the schema for schema documents does not allow",
		},
		{
			name:        "<defaultOpenContent> with no <any>",
			docs:        map[string]string{"main.xsd": wrap("urn:a", `<xs:defaultOpenContent/>`)},
			wantGrammar: "xs:defaultOpenContent",
			wantFault:   "has no <any> child",
		},
		{
			name: "<defaultOpenContent mode=none>",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:defaultOpenContent mode="none"><xs:any/></xs:defaultOpenContent>`)},
			wantGrammar: "xs:defaultOpenContent",
			wantFault:   `mode="none"`,
		},
		{
			// The facet position's own model, whose ##other wildcard excludes the
			// XSD namespace — so a misspelled facet has no position to fall through
			// to (#972).
			name: "misspelled facet under a <simpleType>'s <restriction>",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:simpleType name="T"><xs:restriction base="xs:string">`+
					`<xs:whitespace value="collapse"/></xs:restriction></xs:simpleType>`)},
			wantGrammar: "xs:simpleRestrictionModel",
			wantFault:   "does not admit under the <restriction>",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", tc.docs)
			if err == nil {
				t.Fatalf("Parse succeeded, want the s4s-grammar fault %q", tc.wantFault)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if !strings.Contains(err.Error(), tc.wantFault) {
				t.Fatalf("error = %v, want it to report %q", err, tc.wantFault)
			}
			if !strings.Contains(err.Error(), tc.wantGrammar) {
				t.Fatalf("error = %v, want it to name the Appendix A production %s it violates", err, tc.wantGrammar)
			}
		})
	}
}
