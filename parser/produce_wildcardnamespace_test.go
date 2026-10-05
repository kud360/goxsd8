package parser_test

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// TestProduceWildcardNamespaceActualValue pins the {variety} and {namespaces}
// §3.10.2.2 maps a wildcard's namespace attribute to, read through its ·actual
// value·. The attribute is xs:namespaceList (Appendix A), whose
// specialNamespaceList member restricts xs:token (whiteSpace collapse), so a
// padded " ##any " is the keyword ##any and " ##other " is ##other. Every
// padded-keyword row fails when namespaceVarietyAndSet compares the raw
// attribute string: the padded value falls through to the token split and maps
// to enumeration {"##any"} or {"##other"}.
//
// The tab/newline/CR spellings use character references because §3.3.3
// attribute-value normalization has already turned a literal tab or newline into
// #x20 by the time the producer reads the attribute; only a character reference
// survives it as #x9/#xA/#xD.
//
// The remaining rows pin what the actual value does NOT make a keyword: the test
// is case-sensitive, an empty or whitespace-only value is an enumeration of the
// empty set, and an end-padded multi-token list keeps its enumeration.
func TestProduceWildcardNamespaceActualValue(t *testing.T) {
	const tns = "urn:t"
	absent := xsd.NamespaceName("")
	target := xsd.NamespaceName(tns)
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name    string
		target  string
		ns      string
		variety xsd.NamespaceConstraintVariety
		set     []xsd.Namespace
	}{
		{name: "space-padded ##any", target: tns, ns: " ##any ",
			variety: xsd.NamespaceConstraintAny},
		{name: "tab/newline-padded ##any", target: tns, ns: "&#9;##any&#10;",
			variety: xsd.NamespaceConstraintAny},
		{name: "space-padded ##other", target: tns, ns: " ##other ",
			variety: xsd.NamespaceConstraintNot, set: []xsd.Namespace{absent, target}},
		{name: "tab/CR-padded ##other", target: tns, ns: "&#9;##other&#13;",
			variety: xsd.NamespaceConstraintNot, set: []xsd.Namespace{absent, target}},
		// {namespaces} clause 3: with no targetNamespace the set is {·absent·}
		// alone.
		{name: "padded ##other, no targetNamespace", ns: " ##other ",
			variety: xsd.NamespaceConstraintNot, set: []xsd.Namespace{absent}},
		{name: "case differs from ##any", target: tns, ns: "##Any",
			variety: xsd.NamespaceConstraintEnumeration, set: []xsd.Namespace{xsd.NamespaceName("##Any")}},
		{name: "empty", target: tns, ns: "",
			variety: xsd.NamespaceConstraintEnumeration},
		{name: "whitespace only", target: tns, ns: " &#9; ",
			variety: xsd.NamespaceConstraintEnumeration},
		{name: "end-padded list", target: tns, ns: "##targetNamespace ##local ",
			variety: xsd.NamespaceConstraintEnumeration, set: []xsd.Namespace{target, absent}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := `<xs:complexType name="CT"><xs:sequence>` +
				`<xs:any namespace="` + tc.ns + `" processContents="skip"/>` +
				`</xs:sequence><xs:anyAttribute namespace="` + tc.ns + `" processContents="skip"/></xs:complexType>`
			s, err := produce(t, wrap(tc.target, body))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			td, ok := s.Type(xsd.QName{Space: tc.target, Local: "CT"})
			if !ok {
				t.Fatalf("complexType CT not found")
			}
			ct, ok := td.(xsd.ComplexType)
			if !ok {
				t.Fatalf("CT is %T, want xsd.ComplexType", td)
			}
			ps := topGroup(t, ct).Particles()
			if len(ps) != 1 {
				t.Fatalf("particles = %d, want 1", len(ps))
			}
			elementWC, ok := ps[0].Term().(xsd.ResolvedTerm).Term.(xsd.Wildcard)
			if !ok {
				t.Fatalf("term = %T, want Wildcard", ps[0].Term().(xsd.ResolvedTerm).Term)
			}
			attributeWC, ok := ct.AttributeWildcard()
			if !ok {
				t.Fatalf("attribute wildcard absent, want present")
			}
			for _, w := range []struct {
				site string
				wc   xsd.Wildcard
			}{{"<xs:any>", elementWC}, {"<xs:anyAttribute>", attributeWC}} {
				nc := w.wc.NamespaceConstraint()
				if got := nc.Variety(); got != tc.variety {
					t.Errorf("%s {variety} = %s, want %s", w.site, got, tc.variety)
				}
				if got := nc.Namespaces(); !slices.Equal(got, tc.set) {
					t.Errorf("%s {namespaces} = %v, want %v", w.site, got, tc.set)
				}
			}
		})
	}
}
