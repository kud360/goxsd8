package parser_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// assertS4SIdentityFault checks err is the plain grammar fault checkS4SChildOrder
// charges (no rule ID), opening with wantPrefix — the offending child, its
// position and the fault kind, pinned as a prefix so the subject cannot move
// (#1048) — and naming wantOwner and the model quotation wantModel.
func assertS4SIdentityFault(t *testing.T, err error, wantPrefix, wantOwner, wantModel string) {
	t.Helper()
	if err == nil {
		t.Fatal("Produce accepted an identity constraint whose children its content model does not admit")
	}
	if rule, ok := xsderr.RuleOf(err); ok {
		t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
	}
	if !strings.HasPrefix(err.Error(), wantPrefix) {
		t.Errorf("error = %v, want it to open %q", err, wantPrefix)
	}
	if !strings.Contains(err.Error(), "the "+wantOwner) {
		t.Errorf("error = %v, want it to name the owning %s", err, wantOwner)
	}
	if !strings.Contains(err.Error(), wantModel) {
		t.Errorf("error = %v, want it to quote %q", err, wantModel)
	}
}

// keybaseModel is the quotation every <unique>/<key>/<keyref> fault carries.
const keybaseModel = "xs:keybase's content model (xmlschema11-1.md:2991) is (annotation?, (selector, field+)?)"

// TestProduceS4SIdentityConstraintChildOrderRejected pins xs:keybase
// (§3.11.2, xmlschema11-1.md:2991) on <unique>, <key> and <keyref>: the order
// and maxOccurs of its three positions, a name it has no position for, and the
// run order that puts this walk ahead of every src-identity-constraint clause —
// clause 1 at the fork, clause 4 on the ref= form, and clause 2 on a definition
// a forward ref= builds before its own position is reached (#1786). Each
// document here assembled clean before #1786 except the three run-order rows,
// which src-identity-constraint rejected under a clause instead.
//
// <schema> is line 1 and each row's lines start at line 2.
func TestProduceS4SIdentityConstraintChildOrderRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name       string
		lines      []string
		wantPrefix string
		wantOwner  string
		wantTail   string // the order fault's closing clause, article included (#1098)
	}{
		{
			// idA042's shape.
			name: "selector before annotation on a unique",
			lines: []string{
				`<xs:element name="e">`,
				`<xs:unique name="u">`,
				`<xs:selector xpath="a"/>`,
				`<xs:annotation/>`,
				`<xs:field xpath="@x"/>`,
				`</xs:unique>`,
				`</xs:element>`,
			},
			wantPrefix: "parser: <annotation> at " + produceURI + ":5:1 is out of the child order",
			wantOwner:  "<unique> at " + produceURI + ":3:1",
		},
		{
			// idA043's and annotation00101m2's shape. The article is #1098's: the
			// message reads "an <annotation>", by the table and not by the letter.
			name: "annotation after the fields on a keyref",
			lines: []string{
				`<xs:element name="e">`,
				`<xs:keyref name="r" refer="r">`,
				`<xs:selector xpath="a"/>`,
				`<xs:field xpath="@x"/>`,
				`<xs:annotation/>`,
				`</xs:keyref>`,
				`</xs:element>`,
			},
			wantPrefix: "parser: <annotation> at " + produceURI + ":6:1 is out of the child order",
			wantOwner:  "<keyref> at " + produceURI + ":3:1",
			wantTail:   ", and an <annotation> may not follow the children written before it here",
		},
		{
			name: "field before selector on a key",
			lines: []string{
				`<xs:element name="e">`,
				`<xs:key name="k">`,
				`<xs:field xpath="@x"/>`,
				`<xs:selector xpath="a"/>`,
				`</xs:key>`,
				`</xs:element>`,
			},
			wantPrefix: "parser: <selector> at " + produceURI + ":5:1 is out of the child order",
			wantOwner:  "<key> at " + produceURI + ":3:1",
			wantTail:   ", and a <selector> may not follow the children written before it here",
		},
		{
			name: "two selectors on a keyref",
			lines: []string{
				`<xs:element name="e">`,
				`<xs:keyref name="r" refer="r">`,
				`<xs:selector xpath="a"/>`,
				`<xs:selector xpath="b"/>`,
				`<xs:field xpath="@x"/>`,
				`</xs:keyref>`,
				`</xs:element>`,
			},
			wantPrefix: "parser: <selector> at " + produceURI + ":5:1 repeats a position",
			wantOwner:  "<keyref> at " + produceURI + ":3:1",
		},
		{
			name: "element child on a unique",
			lines: []string{
				`<xs:element name="e">`,
				`<xs:unique name="u">`,
				`<xs:selector xpath="a"/>`,
				`<xs:field xpath="@x"/>`,
				`<xs:element name="x"/>`,
				`</xs:unique>`,
				`</xs:element>`,
			},
			wantPrefix: "parser: <element> at " + produceURI + ":6:1 fills no position",
			wantOwner:  "<unique> at " + produceURI + ":3:1",
		},
		{
			// src-identity-constraint clause 4 rejects both children as well; the
			// walk runs ahead of the ref= arm and answers first.
			name: "field before selector on a ref= key",
			lines: []string{
				`<xs:element name="e">`,
				`<xs:key ref="tns:k">`,
				`<xs:field xpath="@x"/>`,
				`<xs:selector xpath="a"/>`,
				`</xs:key>`,
				`</xs:element>`,
				`<xs:element name="o"><xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:key></xs:element>`,
			},
			wantPrefix: "parser: <selector> at " + produceURI + ":5:1 is out of the child order",
			wantOwner:  "<key> at " + produceURI + ":3:1",
		},
		{
			// src-identity-constraint clause 1 rejects name and ref together; the
			// walk runs ahead of the fork that charges it.
			name: "field before selector on a unique carrying name and ref",
			lines: []string{
				`<xs:element name="e">`,
				`<xs:unique name="u" ref="u">`,
				`<xs:field xpath="@x"/>`,
				`<xs:selector xpath="a"/>`,
				`</xs:unique>`,
				`</xs:element>`,
			},
			wantPrefix: "parser: <selector> at " + produceURI + ":5:1 is out of the child order",
			wantOwner:  "<unique> at " + produceURI + ":3:1",
		},
		{
			// The ref= comes first, so it builds k through referencedIdentityConstraint
			// before k's own position is reached. k has no <selector>, which
			// src-identity-constraint clause 2 rejects, and an <element> child, which
			// the walk rejects; constructIdentityConstraint's own walk is what puts the
			// grammar fault first on this path.
			name: "element child on a key a forward ref= builds",
			lines: []string{
				`<xs:element name="e"><xs:key ref="tns:k"/></xs:element>`,
				`<xs:element name="o">`,
				`<xs:key name="k">`,
				`<xs:field xpath="@x"/>`,
				`<xs:element name="x"/>`,
				`</xs:key>`,
				`</xs:element>`,
			},
			wantPrefix: "parser: <element> at " + produceURI + ":6:1 fills no position",
			wantOwner:  "<key> at " + produceURI + ":4:1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, s4sTopLevelDoc(tc.lines...))
			assertS4SIdentityFault(t, err, tc.wantPrefix, tc.wantOwner, keybaseModel)
			if !strings.HasSuffix(err.Error(), tc.wantTail) {
				t.Errorf("error = %v, want it to close %q", err, tc.wantTail)
			}
		})
	}
}

// TestProduceS4SSelectorFieldChildRejected pins xs:selector (:3010) and xs:field
// (:3016), both "(annotation?)", against every XSD-namespace name the suite
// writes under one: idD018-idD033 under a <selector>, idE018-idE033 under a
// <field>. All of them assembled clean before #1786.
func TestProduceS4SSelectorFieldChildRejected(t *testing.T) {
	foreign := []string{
		"selector", "field", "attribute", "element", "simpleType", "complexType",
		"unique", "key", "keyref", "all", "choice", "sequence", "any",
	}
	for _, host := range []struct {
		local, model string
	}{
		{"selector", "xs:selector's content model (xmlschema11-1.md:3010) is (annotation?)"},
		{"field", "xs:field's content model (xmlschema11-1.md:3016) is (annotation?)"},
	} {
		for _, child := range foreign {
			t.Run(host.local+"/"+child, func(t *testing.T) {
				selector := []string{`<xs:selector xpath="a"/>`}
				field := []string{`<xs:field xpath="@x"/>`}
				hosted := []string{`<xs:` + host.local + ` xpath="a">`, `<xs:` + child + `/>`, `</xs:` + host.local + `>`}
				if host.local == "selector" {
					selector = hosted
				} else {
					field = hosted
				}
				lines := []string{`<xs:element name="e">`, `<xs:unique name="u">`}
				lines = append(lines, selector...)
				lines = append(lines, field...)
				lines = append(lines, `</xs:unique>`, `</xs:element>`)
				// The host opens on line 4 as the selector and on line 5 as the field,
				// and the child is the line after it.
				at := 4
				if host.local == "field" {
					at = 5
				}
				_, err := produce(t, s4sTopLevelDoc(lines...))
				assertS4SIdentityFault(t, err,
					fmt.Sprintf("parser: <%s> at %s:%d:1 fills no position", child, produceURI, at+1),
					fmt.Sprintf("<%s> at %s:%d:1", host.local, produceURI, at),
					host.model)
			})
		}
	}
}

// TestProduceS4SIdentityConstraintChildOrderAccepted is the other side: every
// position of xs:keybase, xs:selector and xs:field filled, in order, on all three
// elements, with foreign-namespace children wherever they stand, and the ref=
// form carrying the <annotation> clause 4 admits.
func TestProduceS4SIdentityConstraintChildOrderAccepted(t *testing.T) {
	doc := wrap("", `<xs:element name="e" xmlns:o="urn:other">`+
		`<xs:unique name="u"><xs:annotation/><o:hint/>`+
		`<xs:selector xpath="a"><xs:annotation/><o:hint/></xs:selector>`+
		`<xs:field xpath="@x"><o:hint/><xs:annotation/></xs:field>`+
		`<xs:field xpath="@y"><xs:annotation/></xs:field></xs:unique>`+
		`<xs:key name="k"><xs:annotation/><xs:selector xpath="b"/><xs:field xpath="@x"/></xs:key>`+
		`<xs:keyref name="r" refer="k"><xs:annotation/><xs:selector xpath="c"/>`+
		`<xs:field xpath="@x"/><o:hint/></xs:keyref>`+
		`<xs:key ref="k"><xs:annotation/></xs:key>`+
		`</xs:element>`)
	if _, err := produce(t, doc); err != nil {
		t.Fatalf("Produce rejected identity constraints written in order: %v", err)
	}
}

// TestProduceIdentityConstraintMisplacedRejected pins §5.1's first bullet
// (xmlschema11-1.md:4296) on a <unique>, <key> or <keyref> written where
// Appendix A admits none: under a top-level <attributeGroup>
// (xs:namedAttributeGroup, :5502) and under an <all>, <choice> or <sequence>
// (xs:allModel, :5259; xs:nestedParticle, :4657), local or a named group's body —
// the shapes of idA/B/C024 and idA/B/C026-028 (#1817). The fault is positioned
// at the constraint, names its owner, and carries no rule ID (STYLE E2).
//
// <schema> is line 1, each owner opens on line 3 and the constraint is line 4.
func TestProduceIdentityConstraintMisplacedRejected(t *testing.T) {
	constraints := []struct{ local, line string }{
		{"unique", `<xs:unique name="c"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>`},
		{"key", `<xs:key name="c"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:key>`},
		{"keyref", `<xs:keyref name="c" refer="tns:k"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:keyref>`},
	}
	// The <key> every <keyref> row refers to, written after the host so the host's
	// line numbers hold.
	const referenced = `<xs:element name="o"><xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:key></xs:element>`
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, host := range []struct {
		name  string
		open  []string // lines 2 and 3; line 3 is the constraint's owner
		close []string
		fault string // the diagnostic's opening: %[1]s the constraint, %[2]s the document
		model string
	}{
		{
			name:  "top-level attributeGroup",
			open:  []string{`<xs:attribute name="a"/>`, `<xs:attributeGroup name="g">`},
			close: []string{`</xs:attributeGroup>`},
			fault: "parser: <%[1]s> at %[2]s:4:1 is not admitted among the children of the <attributeGroup> at %[2]s:3:1",
			model: "xs:namedAttributeGroup's content model (xmlschema11-1.md:5502)",
		},
		{
			name:  "local sequence",
			open:  []string{`<xs:complexType name="t">`, `<xs:sequence>`},
			close: []string{`</xs:sequence>`, `</xs:complexType>`},
			fault: "parser: unexpected model group child <%[1]s> at %[2]s:4:1 under the <sequence> at %[2]s:3:1",
			model: "xs:nestedParticle (xmlschema11-1.md:4657)",
		},
		{
			name:  "choice body of a named group",
			open:  []string{`<xs:group name="g">`, `<xs:choice>`},
			close: []string{`</xs:choice>`, `</xs:group>`},
			fault: "parser: unexpected model group child <%[1]s> at %[2]s:4:1 under the <choice> at %[2]s:3:1",
			model: "xs:nestedParticle (xmlschema11-1.md:4657)",
		},
		{
			name:  "all body of a named group",
			open:  []string{`<xs:group name="g">`, `<xs:all>`},
			close: []string{`</xs:all>`, `</xs:group>`},
			fault: "parser: unexpected model group child <%[1]s> at %[2]s:4:1 under the <all> at %[2]s:3:1",
			model: "xs:allModel (xmlschema11-1.md:5259)",
		},
	} {
		for _, c := range constraints {
			t.Run(host.name+"/"+c.local, func(t *testing.T) {
				lines := slices.Concat(host.open, []string{c.line}, host.close, []string{referenced})
				_, err := produce(t, s4sTopLevelDoc(lines...))
				if err == nil {
					t.Fatalf("Produce accepted a <%s> under a %s", c.local, host.name)
				}
				if rule, ok := xsderr.RuleOf(err); ok {
					t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
				}
				if want := fmt.Sprintf(host.fault, c.local, produceURI); !strings.HasPrefix(err.Error(), want) {
					t.Errorf("error = %v, want it to open %q", err, want)
				}
				if !strings.Contains(err.Error(), host.model) {
					t.Errorf("error = %v, want it to cite %q", err, host.model)
				}
			})
		}
	}
}
