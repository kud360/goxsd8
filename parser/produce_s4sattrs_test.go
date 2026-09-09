package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestUndeclaredAttributeRejected pins the fault #1369 reported: an unprefixed
// attribute no Appendix A production of the element declares is REJECTED rather
// than dropped, on every one of the three positions that reproduced it — the
// <schema> element itself, a top-level declaration and a nested one.
//
// Each row asserts the message's opening "parser: <subject> at " as a PREFIX, so
// a row cannot pass on a rejection that names the same facts in another
// arrangement — two arguments swapped inside one Errorf changes no branch and
// leaves every asserted substring present (#1048) — and each asserts the Appendix
// A production the roster is transcribed from. Every row asserts a PLAIN error
// too: sd-valid is anchored but uncataloged, so charging any src-*/cvc-*/cos-* —
// or "sd-valid" itself in the Rule position — would be a fabricated verdict
// (STYLE E2, xsderr/doc.go).
func TestUndeclaredAttributeRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name        string
		doc         string
		wantSubject string // the element the message opens by charging
		wantCarries string // the offending attribute, in the message's own words
		wantGrammar string // the Appendix A production the message must name
	}{
		{
			name:        "on the <schema> element itself",
			doc:         `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" bogusRoot="1"/>`,
			wantSubject: "schema",
			wantCarries: "carries a bogusRoot attribute",
			wantGrammar: "xs:schema",
		},
		{
			name:        "on a top-level <element>",
			doc:         wrap("", `<xs:element name="foo" bogusAttr="1"/>`),
			wantSubject: "element",
			wantCarries: "carries a bogusAttr attribute",
			wantGrammar: "xs:element",
		},
		{
			// minOccur is minOccurs less its final letter, and the ONE reproduction
			// case with a compiled consequence rather than a missing diagnostic: the
			// declaration used to build with the DEFAULT occurrence constraint, so the
			// schema accepted instances its author wrote it to reject. The correct
			// spelling's own value is pinned below.
			name: "minOccur, the typo for minOccurs on a nested <element>",
			doc: wrap("", `<xs:complexType name="ct"><xs:sequence>`+
				`<xs:element name="bar" type="xs:string" minOccur="2"/>`+
				`</xs:sequence></xs:complexType>`),
			wantSubject: "element",
			wantCarries: "carries a minOccur attribute",
			wantGrammar: "xs:element",
		},
		{
			// The narrower half of the roster: name is declared on xs:group for
			// <group>'s sake, and every production <sequence> is written through —
			// xs:explicitGroup and xs:simpleExplicitGroup — sets it to
			// use="prohibited", so no production of <sequence> declares it at all.
			name: "name on a <sequence>, which no production of it declares",
			doc: wrap("", `<xs:complexType name="ct"><xs:sequence name="s">`+
				`<xs:element name="bar" type="xs:string"/></xs:sequence></xs:complexType>`),
			wantSubject: "sequence",
			wantCarries: "carries a name attribute",
			wantGrammar: "xs:explicitGroup",
		},
		{
			// A facet: xs:noFixedFacet sets fixed to use="prohibited" and <pattern>
			// restricts it further, so fixed reaches no production of <pattern>.
			name: "fixed on a <pattern>, which xs:noFixedFacet prohibits",
			doc: wrap("", `<xs:simpleType name="T"><xs:restriction base="xs:string">`+
				`<xs:pattern value="a" fixed="true"/></xs:restriction></xs:simpleType>`),
			wantSubject: "pattern",
			wantCarries: "carries a fixed attribute",
			wantGrammar: "xs:pattern",
		},
		{
			// <appinfo> is reached even though the walk descends no further into it,
			// its content being <xs:any processContents="lax"> content no guard there
			// governs while its own start tag is still its own production's.
			name:        "on an <appinfo>, whose content the walk does not descend into",
			doc:         wrap("", `<xs:annotation><xs:appinfo bogusInfo="1"/></xs:annotation>`),
			wantSubject: "appinfo",
			wantCarries: "carries a bogusInfo attribute",
			wantGrammar: "xs:appinfo",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			if err == nil {
				t.Fatalf("Produce succeeded, want the s4s-grammar fault %q", tc.wantCarries)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if !strings.HasPrefix(err.Error(), "parser: <"+tc.wantSubject+"> at ") {
				t.Fatalf("error = %v, want it to open by charging the <%s> at its location", err, tc.wantSubject)
			}
			if !strings.Contains(err.Error(), tc.wantCarries) {
				t.Fatalf("error = %v, want it to report %q", err, tc.wantCarries)
			}
			if !strings.Contains(err.Error(), "declares nowhere on <"+tc.wantSubject+">") {
				t.Fatalf("error = %v, want it to state that the grammar declares the name nowhere on the element", err)
			}
			if !strings.Contains(err.Error(), tc.wantGrammar+" (xmlschema11-") {
				t.Fatalf("error = %v, want it to name the Appendix A production %s and the line it is quoted from", err, tc.wantGrammar)
			}
		})
	}
}

// TestMinOccursCorrectlySpelledStillReachesTheParticle is the typo case's second
// half, and the reason #1369 called it the costly one: minOccur="2" used to
// compile to the DEFAULT {min occurs} of 1 with no diagnostic. The rejection above
// is worth nothing unless the correct spelling still reaches 2, which is the fact
// that makes the typo a silent wrong answer rather than a refused attribute.
func TestMinOccursCorrectlySpelledStillReachesTheParticle(t *testing.T) {
	ct := complexType(t, `<xs:complexType name="CT"><xs:sequence>`+
		`<xs:element name="bar" type="xs:string" minOccurs="2" maxOccurs="3"/>`+
		`</xs:sequence></xs:complexType>`, "CT")
	ps := topGroup(t, ct).Particles()
	if len(ps) != 1 {
		t.Fatalf("particles = %d, want 1", len(ps))
	}
	if ps[0].Occurs().Min() != 2 {
		t.Fatalf("bar occurs = %s, want a {min occurs} of 2", ps[0].Occurs())
	}
}

// TestOtherNamespaceAttributeAccepted pins both directions of xs:openAttrs on ONE
// element: its wildcard is namespace="##other" (xmlschema11-1.md:4412-:4425), so
// an attribute in ANOTHER namespace is admitted whatever its local name, and the
// SAME local name unprefixed is admitted by nothing.
//
// processContents="lax" on that wildcard governs how deeply an admitted
// other-namespace attribute is then validated, and has no part in either
// direction here.
func TestOtherNamespaceAttributeAccepted(t *testing.T) {
	const prefixed = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` +
		` xmlns:app="urn:app"><xs:element name="foo" app:bogusAttr="1"/></xs:schema>`
	s, err := produce(t, prefixed)
	if err != nil {
		t.Fatalf("Produce: %v, want an attribute in another namespace to be admitted", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "foo"}); !ok {
		t.Fatal("element foo is missing, so nothing here reads as an admission")
	}

	const unprefixed = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` +
		` xmlns:app="urn:app"><xs:element name="foo" bogusAttr="1"/></xs:schema>`
	if _, err := produce(t, unprefixed); err == nil {
		t.Fatal("the same local name unprefixed was admitted, but ##other reaches no attribute in no namespace")
	}
}

// TestProhibitedAttributeStillChargedAsProhibited pins the seam between the two
// attribute checks. rejectProhibitedAttrs asks whether a name the grammar admits
// SOMEWHERE on the element is forbidden on the FORM in hand, and
// rejectUndeclaredAttrs asks whether the name is admitted on the element at all.
// Every name the first prohibits is in the second's roster, so a prohibited
// attribute must still draw the prohibition verdict — the one that says the name
// is legal on the local form — and never the roster one.
func TestProhibitedAttributeStillChargedAsProhibited(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  string
	}{
		{"ref on a top-level <element>", wrap("", `<xs:element ref="xs:string"/>`)},
		{"minOccurs on a top-level <element>", wrap("", `<xs:element name="e" type="xs:string" minOccurs="2"/>`)},
		{"use on a top-level <attribute>", wrap("", `<xs:attribute name="a" type="xs:string" use="required"/>`)},
		{"minOccurs on a top-level <group>", wrap("", `<xs:group name="g" minOccurs="2"><xs:sequence/></xs:group>`)},
		{"ref on a top-level <attributeGroup>", wrap("", `<xs:attributeGroup name="ag" ref="ag"/>`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			if err == nil {
				t.Fatal("Produce succeeded, want the prohibited-attribute fault")
			}
			if !strings.Contains(err.Error(), `use="prohibited"`) {
				t.Fatalf("error = %v, want the prohibition verdict rather than the undeclared-name one", err)
			}
			if strings.Contains(err.Error(), "declares nowhere on") {
				t.Fatalf("error = %v, want the prohibition verdict; the name is in the element's roster", err)
			}
		})
	}
}
