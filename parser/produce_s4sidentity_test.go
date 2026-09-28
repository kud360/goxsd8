package parser_test

import (
	"fmt"
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
			// idA043's and annotation00101m2's shape.
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
