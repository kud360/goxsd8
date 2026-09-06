package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceLocalAttributeTypeAndSimpleTypeClause4 pins src-attribute clause 4
// (§3.2.3, xmlschema11-1.md:868 — "The type attribute and a simpleType child
// element must not both be present") across every use= a local <attribute> can
// spell. The use="prohibited" row was ACCEPTED before this table, produceAttributeUse
// having returned on the prohibited token ahead of produceLocalAttribute, which
// held the clause's only charge (#1270); the other three are the no-regression
// half, rejecting today and pinned so a misplaced charge cannot move them.
//
// The gap was exactly one use= value wide, so the four spellings are one table:
// clause 4's antecedent names neither use= nor the component the element maps to,
// and a table holding only the prohibited row could not tell a charge that
// reaches every form from one that reaches only the form that used to escape.
func TestProduceLocalAttributeTypeAndSimpleTypeClause4(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		doc  string
		// wantMsg is empty for the rows clause 4 must ACCEPT.
		wantMsg string
		// wantLine is the 1-based line the offending <attribute> opens on, asserted so
		// a charge firing at the wrong position fails here rather than passing on the
		// strength of rejecting something.
		wantLine int
	}{
		{
			// The gap this table closes: mapping to no Attribute Use (§3.2.2) skips the
			// component-building half of the mapping, never §5.1's validation of the
			// element information item.
			name: "prohibited with type and an inline simpleType fails clause 4",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="x" use="prohibited" type="xs:string">
<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
</xs:attribute>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both`,
			wantLine: 3,
		},
		{
			name: "use omitted with type and an inline simpleType fails clause 4",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="x" type="xs:string">
<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
</xs:attribute>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both`,
			wantLine: 3,
		},
		{
			name: "optional with type and an inline simpleType fails clause 4",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="x" use="optional" type="xs:string">
<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
</xs:attribute>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both`,
			wantLine: 3,
		},
		{
			name: "required with type and an inline simpleType fails clause 4",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="x" use="required" type="xs:string">
<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
</xs:attribute>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both`,
			wantLine: 3,
		},
		{
			// The placement constraint the whole table exists to hold, and the only row that
			// falsifies a charge hoisted to the top of produceAttributeUse: this document
			// satisfies clause 4's antecedent AND clause 3.2's, and clause 3.2 owns it,
			// being charged in the hasRef arm ahead of the prohibited return. A hoisted
			// clause-4 charge accepts every other row in this table while silently changing
			// which fault class THIS one reports.
			name: "ref with type and an inline simpleType keeps clause 3's verdict",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:attribute name="A" type="xs:string"/>
<xs:complexType name="ct">
<xs:attribute ref="tns:A" use="prohibited" type="xs:string">
<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
</xs:attribute>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `attribute has both ref and an inline <simpleType>, but src-attribute clause 3 forbids a simpleType with ref`,
			wantLine: 4,
		},
		{
			// The issue's own reproduction row: clause 4's antecedent is unmet (no
			// <simpleType> child), and the type=-with-ref conjunct of clause 3.2 stands
			// unmoved by the new charge.
			name: "ref with type and use=prohibited keeps clause 3's verdict",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:attribute name="A" type="xs:string"/>
<xs:complexType name="ct">
<xs:attribute ref="tns:A" use="prohibited" type="xs:string"/>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `attribute has both ref and type, but src-attribute clause 3 forbids a type with ref`,
			wantLine: 4,
		},
		{
			// One half of the antecedent alone is no fault under any use=, prohibited
			// included: a charge keyed off the <simpleType> child or the type= attribute
			// alone would take these down with it.
			name: "prohibited with an inline simpleType and no type is accepted",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="x" use="prohibited">
<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
</xs:attribute>
</xs:complexType>
</xs:schema>`,
		},
		{
			name: "prohibited with type and no inline simpleType is accepted",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="x" use="prohibited" type="xs:string"/>
</xs:complexType>
</xs:schema>`,
		},
		{
			name: "optional with an inline simpleType and no type is accepted",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="x" use="optional">
<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>
</xs:attribute>
</xs:complexType>
</xs:schema>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			if tc.wantMsg == "" {
				if err != nil {
					t.Fatalf("Produce rejected a document src-attribute clause 4 admits: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Produce accepted the document, want the src-attribute fault %q", tc.wantMsg)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("error = %v, want it to state %q", err, tc.wantMsg)
			}
			assertRule(t, err, "src-attribute")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want the <attribute>'s (E3)", err)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the offending <attribute> at %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}
