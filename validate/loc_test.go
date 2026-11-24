package validate_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// locNested declares r in urn:t over unbounded w children, each holding an
// xs:int e and carrying an optional xs:int attribute a.
const locNested = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:t="urn:t" targetNamespace="urn:t" elementFormDefault="qualified">
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="w" maxOccurs="unbounded">
          <xs:complexType>
            <xs:sequence><xs:element name="e" type="xs:int"/></xs:sequence>
            <xs:attribute name="a" type="xs:int"/>
          </xs:complexType>
        </xs:element>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// The xsderr package doc's "# Using xsderr" says how a violation locates and
// names its item; these two rows pin that for a nested element and an
// attribute. The cvc-attribute charge on the second w's a sits at the
// attribute name's first byte, not at w's own <, and the cvc-type charge on
// that w's e at e's own <, not at w's nor at the valid e on line 2. Both rows
// are guards: they pin behaviour that doc describes, not behaviour it changes.
func TestViolationLocIsTheChargedItems(t *testing.T) {
	const doc = "<t:r xmlns:t=\"urn:t\">\n <t:w><t:e>1</t:e></t:w>\n <t:w a=\"zz\">\n  <t:e>notint</t:e>\n </t:w>\n</t:r>"
	want := []struct {
		rule xsderr.Rule
		loc  xsderr.Loc
		name string
	}{
		{"cvc-attribute", xsderr.Loc{URI: "p.xml", Line: 3, Col: 7}, "attribute a"},
		{"cvc-type", xsderr.Loc{URI: "p.xml", Line: 4, Col: 3}, "element {urn:t}e"},
	}
	got := orderAssess(t, "p.xml", locNested, doc).Violations()
	if len(got) != len(want) {
		t.Fatalf("Violations() = %v, want %d", got, len(want))
	}
	for i, w := range want {
		if got[i].Rule != w.rule || got[i].Loc != w.loc {
			t.Errorf("violation %d = %s at %+v, want %s at %+v", i, got[i].Rule, got[i].Loc, w.rule, w.loc)
		}
		if !strings.Contains(got[i].Msg, w.name) {
			t.Errorf("violation %d Msg = %q, want it to name %s", i, got[i].Msg, w.name)
		}
	}
}
