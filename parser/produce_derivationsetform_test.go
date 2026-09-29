package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceDerivationSetProhibitedFormCharged pins that a block=/final=
// family attribute its element's form prohibits is charged as the prohibited
// attribute, never as a value outside a derivation-set type: use="prohibited"
// removes the declaration, so no type assesses the value and "foo" is no
// datatype fault. xs:localElement prohibits final (xmlschema11-1.md:5125), whose
// ref= form draws src-element clause 2.2; xs:localSimpleType prohibits final
// (xmlschema11-2.md:3908); xs:localComplexType prohibits abstract, final and
// block (xmlschema11-1.md:4824-4826), at every nested position. The plain rows
// carry no rule ID (§5.1 alone, STYLE E2) and pin the opening "parser: <subject>
// at <loc>" as well as the attribute named. The offending element sits on line 3
// in every row.
func TestProduceDerivationSetProhibitedFormCharged(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name string
		body string
		// wantPrefix is a plain error's opening, up to the column; empty for the
		// src-element row, whose message wantMsg pins as a prefix instead.
		wantPrefix string
		wantMsg    string
	}{
		{
			name: `local <element> final="foo"`,
			body: `<xs:complexType name="ct"><xs:sequence>` + "\n" +
				`<xs:element name="l" type="xs:string" final="foo"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantPrefix: "parser: local <element> at " + produceURI + ":3:",
			wantMsg:    "carries a final attribute",
		},
		{
			name: `<element ref> final="foo"`,
			body: `<xs:element name="e" type="xs:string"/><xs:complexType name="ct"><xs:sequence>` + "\n" +
				`<xs:element ref="e" final="foo"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantMsg: `the <element ref="..."> carries a final attribute`,
		},
		{
			name: `nested <simpleType> final="foo"`,
			body: `<xs:element name="e">` + "\n" +
				`<xs:simpleType final="foo"><xs:restriction base="xs:string"/></xs:simpleType>` + "\n" +
				`</xs:element>`,
			wantPrefix: "parser: nested <simpleType> at " + produceURI + ":3:",
			wantMsg:    "carries a final attribute",
		},
		{
			name: `nested <complexType> block="foo"`,
			body: `<xs:element name="e">` + "\n" +
				`<xs:complexType block="foo"><xs:sequence/></xs:complexType>` + "\n" +
				`</xs:element>`,
			wantPrefix: "parser: nested <complexType> at " + produceURI + ":3:",
			wantMsg:    "carries a block attribute",
		},
		{
			name: `nested <complexType> block="extension"`,
			body: `<xs:element name="e">` + "\n" +
				`<xs:complexType block="extension"><xs:sequence/></xs:complexType>` + "\n" +
				`</xs:element>`,
			wantPrefix: "parser: nested <complexType> at " + produceURI + ":3:",
			wantMsg:    "carries a block attribute",
		},
		{
			name: `nested <complexType> final="restriction"`,
			body: `<xs:element name="e">` + "\n" +
				`<xs:complexType final="restriction"><xs:sequence/></xs:complexType>` + "\n" +
				`</xs:element>`,
			wantPrefix: "parser: nested <complexType> at " + produceURI + ":3:",
			wantMsg:    "carries a final attribute",
		},
		{
			name: `nested <complexType> abstract="true" final="#all"`,
			body: `<xs:element name="e">` + "\n" +
				`<xs:complexType abstract="true" final="#all"><xs:sequence/></xs:complexType>` + "\n" +
				`</xs:element>`,
			wantPrefix: "parser: nested <complexType> at " + produceURI + ":3:",
			wantMsg:    "carries a abstract attribute",
		},
		{
			name: `<alternative> <complexType> block="#all"`,
			body: `<xs:element name="e" type="xs:anyType"><xs:alternative test="true()">` + "\n" +
				`<xs:complexType block="#all"><xs:sequence/></xs:complexType>` + "\n" +
				`</xs:alternative></xs:element>`,
			wantPrefix: "parser: nested <complexType> at " + produceURI + ":3:",
			wantMsg:    "carries a block attribute",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, finalSchema("", "\n"+tc.body))
			if tc.wantPrefix == "" {
				assertRule(t, err, "src-element")
				assertIDFault(t, err, tc.wantMsg, 3)
				return
			}
			if err == nil {
				t.Fatalf("Produce accepted the document, want a plain error %q...%q", tc.wantPrefix, tc.wantMsg)
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Fatalf("error charged %s, want a plain error with no rule ID: %v", rule, err)
			}
			msg := err.Error()
			if !strings.HasPrefix(msg, tc.wantPrefix) || !strings.Contains(msg, tc.wantMsg) {
				t.Fatalf("error = %q, want prefix %q and %q", msg, tc.wantPrefix, tc.wantMsg)
			}
		})
	}
}

// TestProduceDerivationSetDeclaredFormNamed pins that the diagnostic names the
// production that declares the attribute on the element's own form: a local
// <element>'s block is xs:localElement's, a top-level <element>'s final is
// xs:topLevelElement's, and a <redefine> child <complexType> is the top-level
// form, so its final is checked, never passed over as a nested form's.
func TestProduceDerivationSetDeclaredFormNamed(t *testing.T) {
	for _, tc := range []struct {
		name    string
		body    string
		wantMsg string
	}{
		{
			name: `local <element> block="list"`,
			body: `<xs:complexType name="ct"><xs:sequence>` + "\n" +
				`<xs:element name="l" type="xs:string" block="list"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantMsg: `<element> block "list" is not in the ·lexical space· of xs:blockSet, the type xs:localElement declares block with`,
		},
		{
			name:    `top-level <element> final="list"`,
			body:    "\n" + `<xs:element name="e" type="xs:string" final="list"/>`,
			wantMsg: `<element> final "list" is not in the ·lexical space· of xs:derivationSet, the type xs:topLevelElement declares final with`,
		},
		{
			name: `<redefine> <complexType> final="list"`,
			body: `<xs:redefine schemaLocation="nowhere.xsd">` + "\n" +
				`<xs:complexType name="ct" final="list"><xs:sequence/></xs:complexType>` + "\n" +
				`</xs:redefine>`,
			wantMsg: `<complexType> final "list" is not in the ·lexical space· of xs:derivationSet, the type xs:topLevelComplexType declares final with`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, finalSchema("", "\n"+tc.body))
			assertRule(t, err, "cvc-datatype-valid")
			assertIDFault(t, err, tc.wantMsg, 3)
		})
	}
}
