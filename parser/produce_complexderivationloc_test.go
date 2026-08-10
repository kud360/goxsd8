package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// derivationDocLine is the 1-based line the complex type UNDER TEST opens on in
// the documents derivationDoc builds. Every base type opens on an earlier line,
// so an assertion on the reported line distinguishes the spec's T from its B.
const derivationDocLine = 5

// derivationDoc builds a document holding up to three <complexType> children:
// base on line 2, intermediate (pass "" for the two-type shapes) on line 3, and
// under — the type whose derivation or properties are being rejected — on
// derivationDocLine.
func derivationDoc(base, intermediate, under string) string {
	return wrap("urn:x", strings.Join([]string{"", base, intermediate, "", under}, "\n"))
}

// TestComplexDerivationRejectionsCiteTheOffendingType pins the POSITION of every
// derivation-ok-restriction (§3.4.6.3) and ct-props-correct (§3.4.6.1) rejection
// complexderivation.go raises, end to end through Parse: each is charged to the
// complex type the constraint is stated against — the spec's T for
// derivation-ok-restriction, the type whose properties are being checked for
// ct-props-correct — and not to the zero Loc, which renders as "?" and tells a
// schema author nothing (#662).
//
// One case per CHARGE SITE, not per rule: nine of these sites share the
// derivation-ok-restriction rule ID, so each case also asserts a message
// fragment unique to its site, and a fixture that drifted onto a sibling site
// would fail rather than pass on the shared rule ID.
//
// Asserting the exact line, rather than merely a non-zero Loc, is what makes a
// case fail if its site is charged the BASE type's position instead: every
// fixture's base types sit on lines other than derivationDocLine.
func TestComplexDerivationRejectionsCiteTheOffendingType(t *testing.T) {
	for _, tc := range []struct {
		site         string
		rule         xsderr.Rule
		base         string
		intermediate string
		under        string
		want         string // a message fragment unique to this charge site
	}{
		{site: "checkSimpleBaseIsExtension: ct-props-correct clause 2",
			rule: "ct-props-correct",
			base: `<xs:complexType name="B"/>`,
			// A simple {base type definition} under restriction is reachable only
			// through <complexContent>: src-ct (§3.4.3) states no clause against a
			// simple base there, so ct-props-correct clause 2 is the constraint that
			// rejects it. The <simpleContent><restriction> spelling of the same shape
			// is declined earlier, by produce_complex.go's §3.4.2.2 cases 1-2.
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="xs:string"/></xs:complexContent></xs:complexType>`,
			want:  "ct-props-correct clause 2 requires extension"},

		{site: "checkAttributeUseNamesUnique: ct-props-correct clause 4",
			rule:  "ct-props-correct",
			base:  `<xs:complexType name="B"><xs:attribute name="a" type="xs:string"/></xs:complexType>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:extension base="tns:B"><xs:attribute name="a" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType>`,
			want:  `share the expanded name a, but ct-props-correct clause 4 forbids it`},

		{site: "checkRestrictionBaseFinal: derivation-ok-restriction clause 1",
			rule:  "derivation-ok-restriction",
			base:  `<xs:complexType name="B" final="restriction"/>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"/></xs:complexContent></xs:complexType>`,
			want:  "derivation-ok-restriction clause 1 forbids"},

		{site: "checkRestrictionContentType: derivation-ok-restriction clause 2",
			rule:  "derivation-ok-restriction",
			base:  `<xs:complexType name="B"/>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`,
			want:  "under any branch of derivation-ok-restriction clause 2"},

		{site: "checkRestrictionAttributes: clause 3, c-ran",
			rule:  "derivation-ok-restriction",
			base:  `<xs:complexType name="B"/>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:attribute name="a" type="xs:string"/></xs:restriction></xs:complexContent></xs:complexType>`,
			want:  "neither declares nor admits through an {attribute wildcard}"},

		{site: "checkRestrictionAttributeWildcard: clause 3 via c-avaw, the base has no wildcard",
			rule:  "derivation-ok-restriction",
			base:  `<xs:complexType name="B"/>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:anyAttribute namespace="##any"/></xs:restriction></xs:complexContent></xs:complexType>`,
			want:  "declares an {attribute wildcard}, but"},

		{site: "checkRestrictionAttributeWildcard: clause 3 via c-avaw and cos-ns-subset",
			rule:  "derivation-ok-restriction",
			base:  `<xs:complexType name="B"><xs:anyAttribute namespace="urn:y"/></xs:complexType>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:anyAttribute namespace="##any"/></xs:restriction></xs:complexContent></xs:complexType>`,
			want:  "{attribute wildcard} admits expanded names"},

		{site: "checkRestrictionRequiredAttributes: clause 3 via cvc-complex-type clause 3, the use is prohibited",
			rule:  "derivation-ok-restriction",
			base:  `<xs:complexType name="B"><xs:attribute name="a" type="xs:string" use="required"/></xs:complexType>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:attribute name="a" use="prohibited"/></xs:restriction></xs:complexContent></xs:complexType>`,
			want:  "carry no use for attribute a, which the base requires"},

		{site: "checkRestrictionRequiredAttributes: clause 3 via cvc-complex-type clause 3, the use is relaxed",
			rule:  "derivation-ok-restriction",
			base:  `<xs:complexType name="B"><xs:attribute name="a" type="xs:string" use="required"/></xs:complexType>`,
			under: `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:attribute name="a" type="xs:string" use="optional"/></xs:restriction></xs:complexContent></xs:complexType>`,
			want:  "as optional where the base requires it"},

		// The two clause 4 sites need a base whose ·locally declared type· for the
		// name is NOT the one clause 3 already compared, or clause 3 charges first:
		// B prohibits the attribute and drops the element BB declares, so
		// key-ldtype's case-3 recursion finds BB's type while B's own binding for
		// the name is its wildcard.
		{site: "checkLocallyDeclaredAttributeTypes: clause 4, c-vs-ctd-r",
			rule:         "derivation-ok-restriction",
			base:         `<xs:complexType name="BB"><xs:attribute name="a" type="xs:string"/><xs:anyAttribute namespace="##any" processContents="lax"/></xs:complexType>`,
			intermediate: `<xs:complexType name="B"><xs:complexContent><xs:restriction base="tns:BB"><xs:attribute name="a" use="prohibited"/><xs:anyAttribute namespace="##any" processContents="lax"/></xs:restriction></xs:complexContent></xs:complexType>`,
			under:        `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:attribute name="a" type="xs:int"/><xs:anyAttribute namespace="##any" processContents="lax"/></xs:restriction></xs:complexContent></xs:complexType>`,
			want:         "of attribute a within the restriction is not ·validly substitutable·"},

		{site: "checkLocallyDeclaredElementTypes: clause 4, c-vs-ctd-r",
			rule:         "derivation-ok-restriction",
			base:         `<xs:complexType name="BB"><xs:sequence><xs:element name="e" type="xs:string" minOccurs="0"/><xs:any namespace="##any" minOccurs="0" processContents="lax"/></xs:sequence></xs:complexType>`,
			intermediate: `<xs:complexType name="B"><xs:complexContent><xs:restriction base="tns:BB"><xs:sequence><xs:any namespace="##any" minOccurs="0" processContents="lax"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`,
			under:        `<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B"><xs:sequence><xs:element name="e" type="xs:int" minOccurs="0"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`,
			want:         "of element e within the restriction is not ·validly substitutable·"},
	} {
		t.Run(tc.site, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", map[string]string{
				"main.xsd": derivationDoc(tc.base, tc.intermediate, tc.under),
			})
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("Parse error = %v, want an *xsderr.Error", err)
			}
			if xe.Rule != tc.rule {
				t.Fatalf("rule = %q, want %q (%v)", xe.Rule, tc.rule, err)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want the %s message, which contains %q", err, tc.site, tc.want)
			}
			if xe.Loc == (xsderr.Loc{}) {
				t.Fatalf("loc is the zero Loc, rendering as %q: %v", xe.Loc, err)
			}
			if xe.Loc.URI != "main.xsd" || xe.Loc.Line != derivationDocLine {
				t.Fatalf("loc = %s, want main.xsd:%d — the <complexType name=\"R\"> the constraint is stated against", xe.Loc, derivationDocLine)
			}
		})
	}
}
