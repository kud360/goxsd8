package parser

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This test is package-internal because the token→keyword pairing it pins is not
// observable from outside: xsd.NamespaceConstraint exposes no accessor for the
// keyword half of {disallowed names} (STYLE T5, until M5 supplies a caller). The
// pairing is exactly where a silent conformance bug would hide — §3.10.2.2 maps
// ##definedSibling to the keyword sibling ALONE, never to both keywords.
func TestDisallowedNameKeywordOf(t *testing.T) {
	cases := []struct {
		name              string
		tok               string
		attributeWildcard bool
		want              xsd.DisallowedNameKeyword
		wantErr           bool
	}{
		{name: "defined-on-any", tok: "##defined", want: xsd.DisallowedNameDefined},
		{name: "defined-on-anyAttribute", tok: "##defined", attributeWildcard: true, want: xsd.DisallowedNameDefined},
		{name: "definedSibling-on-any", tok: "##definedSibling", want: xsd.DisallowedNameSibling},
		// xs:qnameListA (the <anyAttribute> notQName type) enumerates only
		// ##defined — the machine-checkable form of w-props-correct clause 5.
		{name: "definedSibling-on-anyAttribute", tok: "##definedSibling", attributeWildcard: true, wantErr: true},
		{name: "unknown-on-any", tok: "##foo", wantErr: true},
		{name: "unknown-on-anyAttribute", tok: "##foo", attributeWildcard: true, wantErr: true},
		{name: "case-sensitive", tok: "##Defined", wantErr: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := disallowedNameKeywordOf(c.tok, c.attributeWildcard, xsderr.Loc{})
			if c.wantErr {
				if err == nil {
					t.Fatalf("disallowedNameKeywordOf(%q, attribute=%v) accepted the token, want a datatype-validity rejection", c.tok, c.attributeWildcard)
				}
				if rule, ok := xsderr.RuleOf(err); !ok || rule != ruleDatatypeValid {
					t.Errorf("RuleOf = (%q, %v), want (%q, true)", rule, ok, ruleDatatypeValid)
				}
				return
			}
			if err != nil {
				t.Fatalf("disallowedNameKeywordOf(%q, attribute=%v): %v", c.tok, c.attributeWildcard, err)
			}
			if got != c.want {
				t.Errorf("disallowedNameKeywordOf(%q) = %s, want %s", c.tok, got, c.want)
			}
		})
	}
}
