package parser

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

const simpleTypeBodyURI = "mem://simpletypebody.xsd"

// firstSimpleType reads a one-element <schema> and returns its <simpleType>
// child. It calls simpleTypeBody DIRECTLY, which is the only way to reach the
// MORE-THAN-ONE branch at all: constructSimpleType runs checkS4SChildOrder ahead
// of it and charges that shape over the repeated child first, so a fixture driven
// through Produce would pin the s4s walk's verdict and never this function's.
func firstSimpleType(t *testing.T, body string) *Element {
	t.Helper()
	doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xs:schema>`
	d, err := ReadDocument(simpleTypeBodyURI, strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	el := childElement(d.Root(), xsd.XMLSchemaNS, "simpleType")
	if el == nil {
		t.Fatalf("fixture %q has no <simpleType> child", body)
	}
	return el
}

// TestSimpleTypeBodyRejectionsCarryNoRule pins the footing of BOTH of
// simpleTypeBody's rejections: failing xs:simpleType's content model
// (xmlschema11-2.md:2743) by naming two of the three §3.16.2.1 alternatives or
// none is §5.1's first bullet (xmlschema11-1.md:4296), which carries no numbered
// rule ID. src-simple-type's four clauses (§3.16.3) each govern a condition
// within an alternative already chosen and none states how many are present, so
// charging it — as both branches did before #1097 — is a fabricated verdict
// (STYLE E2).
//
// The MORE-THAN-ONE case is pinned HERE rather than through Produce because
// checkS4SChildOrder intercepts that document; census (census.go) is the only
// caller that still reaches this branch, and it discards the value, so without
// this test the branch's footing is asserted nowhere.
func TestSimpleTypeBodyRejectionsCarryNoRule(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "more than one alternative",
			body: `<xs:simpleType name="B"><xs:restriction base="xs:string"/><xs:list itemType="xs:string"/></xs:simpleType>`,
			want: "has both a <restriction> and a <list> child",
		},
		{
			name: "no alternative",
			body: `<xs:simpleType name="B"><xs:annotation/></xs:simpleType>`,
			want: "has no <restriction>, <list> or <union> child",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := simpleTypeBody(firstSimpleType(t, c.body))
			if err == nil {
				t.Fatalf("simpleTypeBody accepted %s", c.name)
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %v, want it to state %q", err, c.want)
			}
			// STYLE E3: a plain error holds no xsderr.Loc, so the position rides in
			// the message text or it is lost.
			if !strings.Contains(err.Error(), "<simpleType> at "+simpleTypeBodyURI+":1:") {
				t.Errorf("error = %v, want it to position the offending <simpleType>", err)
			}
		})
	}
}

// TestSimpleTypeBodyReturnsTheChosenAlternative pins that the rejections above
// are not the whole function: each of the three §3.16.2.1 alternatives, written
// alone, is returned rather than refused.
func TestSimpleTypeBodyReturnsTheChosenAlternative(t *testing.T) {
	for _, want := range []string{"restriction", "list", "union"} {
		t.Run(want, func(t *testing.T) {
			body := `<xs:simpleType name="B"><xs:` + want + `/></xs:simpleType>`
			got, err := simpleTypeBody(firstSimpleType(t, body))
			if err != nil {
				t.Fatalf("simpleTypeBody on a lone <%s>: %v", want, err)
			}
			if got.Name().Local() != want {
				t.Errorf("simpleTypeBody = <%s>, want <%s>", got.Name().Local(), want)
			}
		})
	}
}
