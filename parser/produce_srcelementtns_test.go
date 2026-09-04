package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceLocalElementTargetNamespaceClause4 pins src-element clause 4
// (ed-with-ns, §3.3.3, xmlschema11-1.md:1321) on the inline local <element
// name="..."> form: before this charge every one of these documents was
// ACCEPTED, the attribute being read only by localTargetNS, which mints the
// declaration in whatever namespace it names without asking whether the clause
// admits it.
//
// Each row carries a whole schema rather than a body fragment, because clause
// 4.3's antecedent reads the ancestor <schema>'s OWN targetNamespace and the
// rows differ in whether it has one — the fact wrap's shared body cannot vary.
//
// The rejections and the acceptances are one table on purpose: the clause's
// force is entirely in which side of it a document falls on, and a check that
// rejected everything writing targetNamespace would pass a rejection-only table.
func TestProduceLocalElementTargetNamespaceClause4(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		doc  string
		// wantMsg is empty for the rows clause 4 must ACCEPT.
		wantMsg string
		// wantLine is the 1-based line the offending <element> sits on, asserted so
		// a check firing at the wrong position fails here rather than passing on the
		// strength of rejecting something.
		wantLine int
	}{
		{
			name: "no complexType ancestor at all fails 4.3.1",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:group name="g"><xs:sequence>
<xs:element name="e" type="xs:integer" targetNamespace="b"/>
</xs:sequence></xs:group>
</xs:schema>`,
			wantMsg:  `has no <complexType> ancestor, which src-element clause 4.3.1 requires`,
			wantLine: 3,
		},
		{
			name: "complexType ancestor but no restriction fails 4.3.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct"><xs:sequence>
<xs:element name="e" type="xs:integer" targetNamespace="b"/>
</xs:sequence></xs:complexType>
</xs:schema>`,
			wantMsg:  `no <restriction> stands between it and the <complexType>`,
			wantLine: 3,
		},
		{
			name: "extension rather than restriction fails 4.3.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:complexType name="base"><xs:sequence/></xs:complexType>
<xs:complexType name="ct"><xs:complexContent><xs:extension base="tns:base"><xs:sequence>
<xs:element name="e" type="xs:integer" targetNamespace="b"/>
</xs:sequence></xs:extension></xs:complexContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `no <restriction> stands between it and the <complexType>`,
			wantLine: 4,
		},
		{
			name: "restriction whose base matches anyType fails 4.3.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct"><xs:complexContent><xs:restriction base="xs:anyType"><xs:sequence>
<xs:element name="e" type="xs:integer" targetNamespace="b"/>
</xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `has base="xs:anyType", which ·matches· xs:anyType`,
			wantLine: 3,
		},
		{
			name: "form present alongside targetNamespace fails 4.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:complexType name="base"><xs:sequence>
<xs:element name="e" type="xs:integer" minOccurs="0"/>
</xs:sequence></xs:complexType>
<xs:complexType name="ct"><xs:complexContent><xs:restriction base="tns:base"><xs:sequence>
<xs:element name="e" form="qualified" type="xs:integer" targetNamespace="b"/>
</xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `src-element clause 4.2 admits no form attribute when targetNamespace is present`,
			wantLine: 6,
		},
		{
			name: "name absent alongside targetNamespace fails 4.1",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct"><xs:sequence>
<xs:element type="xs:integer" targetNamespace="a"/>
</xs:sequence></xs:complexType>
</xs:schema>`,
			wantMsg:  `src-element clause 4.1 requires`,
			wantLine: 3,
		},
		{
			// The easiest case to get wrong: 4.3's antecedent FAILS here, so 4.3.1 and
			// 4.3.2 are never read and the missing <complexType>/<restriction>
			// ancestors are no fault at all.
			name: "targetNamespace equal to the schema's is accepted with no ancestor",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:group name="g"><xs:sequence>
<xs:element name="e" type="xs:integer" targetNamespace="a"/>
</xs:sequence></xs:group>
<xs:complexType name="ct"><xs:group ref="tns:g"/></xs:complexType>
<xs:element name="root" type="tns:ct"/>
</xs:schema>`,
		},
		{
			// The one shape clause 4 exists to admit, carried all the way through
			// finalization: the base's wildcard is what lets a {b}e particle ·restrict·
			// an {a}-namespace base at all, so the row asserts acceptance rather than
			// merely the absence of a clause-4 verdict.
			name: "restriction whose base is not anyType is accepted",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:complexType name="base"><xs:sequence>
<xs:any namespace="##any" processContents="lax" minOccurs="0"/>
</xs:sequence></xs:complexType>
<xs:complexType name="ct"><xs:complexContent><xs:restriction base="tns:base"><xs:sequence>
<xs:element name="e" type="xs:integer" targetNamespace="b" minOccurs="0"/>
</xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`,
		},
		{
			// base= is bound against the <restriction>'s OWN in-scope namespaces
			// (PRINCIPLES 19). The <element> REBINDS the same prefix to another
			// namespace, so a check reading the element's bindings instead would
			// resolve base to {urn:not-xsd}anyType and accept this document.
			name: "restriction base bound by the restriction's own prefix",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct"><xs:complexContent>
<xs:restriction xmlns:q="http://www.w3.org/2001/XMLSchema" base="q:anyType"><xs:sequence>
<xs:element xmlns:q="urn:not-xsd" name="e" type="xs:integer" targetNamespace="b"/>
</xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `has base="q:anyType", which ·matches· xs:anyType`,
			wantLine: 4,
		},
		{
			// A schema with NO targetNamespace attribute satisfies 4.3's antecedent
			// however the element spells its own, so the ancestors are read even for a
			// targetNamespace="" that string-compares equal to the absent one.
			name: "empty targetNamespace under a schema having none is still read",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xs:complexType name="ct"><xs:sequence>
<xs:element name="e" type="xs:integer" targetNamespace=""/>
</xs:sequence></xs:complexType>
</xs:schema>`,
			wantMsg:  `no <restriction> stands between it and the <complexType>`,
			wantLine: 3,
		},
		{
			name: "no targetNamespace at all is untouched by clause 4",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct"><xs:sequence>
<xs:element name="e" type="xs:integer"/>
</xs:sequence></xs:complexType>
</xs:schema>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			if tc.wantMsg == "" {
				if err != nil {
					t.Fatalf("Produce rejected a document src-element clause 4 admits: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Produce accepted the document, want the src-element clause 4 fault %q", tc.wantMsg)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("error = %v, want it to state %q", err, tc.wantMsg)
			}
			assertRule(t, err, "src-element")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want the <element>'s (E3)", err)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the offending <element> at %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}

// TestProduceRefElementTargetNamespaceKeepsClause22 pins that clause 4 did not
// take the ref= form's targetNamespace away from src-element clause 2.2, which
// rejectRefElementDeclarationAttrs charges and whose doc records that clause 4.1
// fires with it and loses. The ref= form never reaches produceLocalElement, so
// the message — not merely the rule ID, which both charges share — is the
// assertion that separates them.
func TestProduceRefElementTargetNamespaceKeepsClause22(t *testing.T) {
	_, err := produce(t, wrap("a", `
<xs:element name="E" type="xs:string"/>
<xs:complexType name="ct"><xs:sequence>
<xs:element ref="tns:E" targetNamespace="b"/>
</xs:sequence></xs:complexType>`))
	if err == nil {
		t.Fatal(`Produce accepted an <element ref> carrying targetNamespace, want the src-element clause 2.2 fault`)
	}
	const want = `the <element ref="..."> carries a targetNamespace attribute, but src-element clause 2.2 admits no unqualified attribute other than minOccurs, maxOccurs and id`
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want it to state %q", err, want)
	}
	assertRule(t, err, "src-element")
}

// TestProduceTopLevelElementTargetNamespaceKeepsGrammarFault pins that clause 4
// did not take the TOP-LEVEL form's targetNamespace away from
// rejectProhibitedAttrs, which charges it as the §5.1 grammar fault
// xs:topLevelElement's use="prohibited" makes it (xmlschema11-1.md:5086-:5107).
// Charging src-element there would be a fabricated verdict (STYLE E2), so the
// absence of a rule ID is the assertion.
func TestProduceTopLevelElementTargetNamespaceKeepsGrammarFault(t *testing.T) {
	_, err := produce(t, wrap("a", `<xs:element name="root" type="xs:string" targetNamespace="b"/>`))
	if err == nil {
		t.Fatal("Produce accepted a top-level <element> carrying targetNamespace, want the grammar fault")
	}
	if !strings.Contains(err.Error(), "carries a targetNamespace attribute, which the schema for schema documents prohibits") {
		t.Fatalf("error = %v, want the rejectProhibitedAttrs grammar fault", err)
	}
	if rule, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("error = %v, charged %s; want a plain grammar fault carrying no rule ID (STYLE E2)", err, rule)
	}
}
