package parser_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceNestedSimpleTypeAttrsProhibited pins, END TO END from a schema
// DOCUMENT, that a <simpleType> written anywhere but as a child of <schema>,
// <redefine> or <override> is REJECTED for a name or final attribute:
// xs:localSimpleType restricts both to use="prohibited" (xmlschema11-2.md:3901,
// :3908). Without the guard every row PRODUCES — the five nested construction
// sites hand constructSimpleType the zero QName and simply discard whatever name
// was written — so a row that merely asserted "rejected" would be pinning
// nothing.
//
// The fault is a plain grammar fault, never a rule verdict: src-simple-type
// (§3.16.3) states no clause for either attribute and no s4s-* identifier exists
// in the spec, so charging src-simple-type or st-props-correct would be
// fabricated (STYLE E2). Each row asserts the diagnostic names ITS OWN attribute
// and is positioned at the offending element's own line (STYLE E3, carried in
// the message text since a plain error holds no xsderr.Loc), so a guard that
// checked one attribute and reported the other, or reported the enclosing
// declaration's position, would not pass.
//
// The rows walk every nested position the producer builds a <simpleType> from:
// the three §3.16.2.1 bodies, the top-level and local <element> and <attribute>
// forms, plus final and the both-written ordering case.
func TestProduceNestedSimpleTypeAttrsProhibited(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts
	// on line 2 of the wrapped document.
	const nested = "\n" + `<xs:simpleType %s>` + "\n" +
		`<xs:restriction base="xs:string"><xs:length value="4"/></xs:restriction>` + "\n" +
		`</xs:simpleType>` + "\n"
	cases := []struct {
		name     string
		body     string
		wantAttr string
		wantLine int
	}{
		{
			name: `<restriction> body`,
			body: "\n" + `<xs:simpleType name="parent"><xs:restriction>` +
				fmt.Sprintf(nested, `name="fooType"`) +
				`</xs:restriction></xs:simpleType>`,
			wantAttr: "name",
			wantLine: 3,
		},
		{
			name: `<list> body`,
			body: "\n" + `<xs:simpleType name="parent"><xs:list>` +
				fmt.Sprintf(nested, `name="fooType"`) +
				`</xs:list></xs:simpleType>`,
			wantAttr: "name",
			wantLine: 3,
		},
		{
			name: `<union> body`,
			body: "\n" + `<xs:simpleType name="parent"><xs:union>` +
				fmt.Sprintf(nested, `name="fooType"`) +
				`</xs:union></xs:simpleType>`,
			wantAttr: "name",
			wantLine: 3,
		},
		{
			name: `top-level <attribute>`,
			body: "\n" + `<xs:attribute name="parent">` +
				fmt.Sprintf(nested, `name="fooType"`) +
				`</xs:attribute>`,
			wantAttr: "name",
			wantLine: 3,
		},
		{
			name: `top-level <element>`,
			body: "\n" + `<xs:element name="parent">` +
				fmt.Sprintf(nested, `name="fooType"`) +
				`</xs:element>`,
			wantAttr: "name",
			wantLine: 3,
		},
		{
			name: `local <element>`,
			body: "\n" + `<xs:complexType name="ct"><xs:sequence><xs:element name="child">` +
				fmt.Sprintf(nested, `name="fooType"`) +
				`</xs:element></xs:sequence></xs:complexType>`,
			wantAttr: "name",
			wantLine: 3,
		},
		{
			name: `local <attribute>`,
			body: "\n" + `<xs:complexType name="ct"><xs:attribute name="a">` +
				fmt.Sprintf(nested, `name="fooType"`) +
				`</xs:attribute></xs:complexType>`,
			wantAttr: "name",
			wantLine: 3,
		},
		{
			name: `final on a nested <simpleType>`,
			body: "\n" + `<xs:element name="parent">` +
				fmt.Sprintf(nested, `final="restriction"`) +
				`</xs:element>`,
			wantAttr: "final",
			wantLine: 3,
		},
		{
			name: `name and final together report name`,
			body: "\n" + `<xs:element name="parent">` +
				fmt.Sprintf(nested, `name="fooType" final="restriction"`) +
				`</xs:element>`,
			wantAttr: "name",
			wantLine: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.body))
			if err == nil {
				t.Fatalf("Produce succeeded, want a grammar fault for the prohibited %s", tc.wantAttr)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if want := fmt.Sprintf("carries a %s attribute", tc.wantAttr); !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to name %s as the prohibited attribute", err, tc.wantAttr)
			}
			if at := fmt.Sprintf("%s:%d:", produceURI, tc.wantLine); !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at %s (E3)", err, at)
			}
		})
	}
}

// TestProduceTopLevelSimpleTypeKeepsNameAndFinal pins the other side of the
// position split the guard keys on: a <simpleType> that IS a <schema> child
// takes xs:topLevelSimpleType, where name is REQUIRED (xmlschema11-2.md:3883)
// and final stays the optional attribute the abstract xs:simpleType declares
// (:3865), so the very attributes rejected one level down must still produce a
// named component with its {final} mapped. A guard keyed on the attribute alone
// rather than on the position would fail here.
func TestProduceTopLevelSimpleTypeKeepsNameAndFinal(t *testing.T) {
	st := producedSimpleType(t, wrap("", `<xs:simpleType name="code" final="restriction">`+
		`<xs:restriction base="xs:string"><xs:length value="4"/></xs:restriction>`+
		`</xs:simpleType>`), "code")
	if got := st.Name(); got.Local != "code" {
		t.Fatalf("{name} = %v, want code", got)
	}
	if got := st.Final(); !slices.Contains(got, xsd.DerivationRestriction) {
		t.Fatalf("{final} = %v, want it to carry restriction", got)
	}
}
