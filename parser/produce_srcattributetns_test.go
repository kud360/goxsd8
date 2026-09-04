package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceLocalAttributeTargetNamespaceClause6 pins src-attribute clause 6
// (att-with-ns, §3.2.3, xmlschema11-1.md:868) on the local <attribute name="...">
// form: before this charge every one of these documents was ACCEPTED, the
// attribute being read only by localTargetNS, which mints the declaration in
// whatever namespace it names without asking whether the clause admits it.
//
// Each row carries a whole schema rather than a body fragment, because clause
// 6.3's antecedent reads the ancestor <schema>'s OWN targetNamespace and the rows
// differ in whether it has one — the fact wrap's shared body cannot vary.
//
// The rejections and the acceptances are one table on purpose: the clause's force
// is entirely in which side of it a document falls on, and a check that rejected
// everything writing targetNamespace would pass a rejection-only table.
func TestProduceLocalAttributeTargetNamespaceClause6(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		doc  string
		// wantMsg is empty for the rows clause 6 must ACCEPT.
		wantMsg string
		// wantLine is the 1-based line the offending <attribute> sits on, asserted so
		// a check firing at the wrong position fails here rather than passing on the
		// strength of rejecting something.
		wantLine int
	}{
		{
			// §3.2.2 admits <attributeGroup> as a local attribute's ancestor
			// (xmlschema11-1.md:841), and that ancestor reaches no <complexType> at all.
			name: "no complexType ancestor at all fails 6.3.1",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:attributeGroup name="ag">
<xs:attribute name="w" type="xs:integer" targetNamespace="b"/>
</xs:attributeGroup>
</xs:schema>`,
			wantMsg:  `has no <complexType> ancestor, which src-attribute clause 6.3.1 requires`,
			wantLine: 3,
		},
		{
			name: "complexType ancestor but no restriction fails 6.3.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="w" type="xs:integer" targetNamespace="b"/>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `no <restriction> stands between it and the <complexType>`,
			wantLine: 3,
		},
		{
			// The shape of two of the three suite fixtures this charge flips
			// (ibmData/schema_invalid/S3_2_3/s3_2_3si02, s3_2_3si09): an <extension> is
			// not a <restriction>, so 6.3.2 fails exactly as with no <restriction> at all.
			name: "extension rather than restriction fails 6.3.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:complexType name="base"><xs:sequence/></xs:complexType>
<xs:complexType name="ct"><xs:complexContent><xs:extension base="tns:base">
<xs:attribute name="w" type="xs:integer" targetNamespace="b"/>
</xs:extension></xs:complexContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `no <restriction> stands between it and the <complexType>`,
			wantLine: 4,
		},
		{
			// The <simpleContent> spelling of the same fault, which s3_2_3si02 writes:
			// the derivation alternant sits one level deeper and the walk must still
			// reach the <complexType>.
			name: "simpleContent extension fails 6.3.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct"><xs:simpleContent><xs:extension base="xs:integer">
<xs:attribute name="w" type="xs:integer" targetNamespace="b"/>
</xs:extension></xs:simpleContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `no <restriction> stands between it and the <complexType>`,
			wantLine: 3,
		},
		{
			// s3_2_3si05's shape, with the schema carrying no targetNamespace at all —
			// so 6.3's antecedent holds and 6.3.2 is read.
			name: "restriction whose base matches anyType fails 6.3.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xs:complexType name="ct"><xs:complexContent><xs:restriction base="xs:anyType">
<xs:attribute name="a1" type="xs:string" targetNamespace="b"/>
</xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `has base="xs:anyType", which ·matches· xs:anyType`,
			wantLine: 3,
		},
		{
			// base= is bound against the <restriction>'s OWN in-scope namespaces
			// (PRINCIPLES 19). The <attribute> REBINDS the same prefix to another
			// namespace, so a check reading the attribute's bindings instead would
			// resolve base to {urn:not-xsd}anyType and accept this document.
			name: "restriction base bound by the restriction's own prefix",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct"><xs:complexContent>
<xs:restriction xmlns:q="http://www.w3.org/2001/XMLSchema" base="q:anyType">
<xs:attribute xmlns:q="urn:not-xsd" name="w" type="xs:integer" targetNamespace="b"/>
</xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`,
			wantMsg:  `has base="q:anyType", which ·matches· xs:anyType`,
			wantLine: 4,
		},
		{
			// 6.2 is read ahead of 6.3, so it fires on a targetNamespace the ancestor
			// <schema> shares — the value 6.3's antecedent would have let through.
			name: "form present alongside targetNamespace fails 6.2",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="w" form="qualified" type="xs:integer" targetNamespace="a"/>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `src-attribute clause 6.2 admits no form attribute when targetNamespace is present`,
			wantLine: 3,
		},
		{
			// A schema with NO targetNamespace attribute satisfies 6.3's antecedent
			// however the attribute spells its own, so the ancestors are read even for a
			// targetNamespace="" that string-compares equal to the absent one.
			name: "empty targetNamespace under a schema having none is still read",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xs:complexType name="ct">
<xs:attribute name="w" type="xs:integer" targetNamespace=""/>
</xs:complexType>
</xs:schema>`,
			wantMsg:  `no <restriction> stands between it and the <complexType>`,
			wantLine: 3,
		},
		{
			// The easiest case to get wrong: 6.3's antecedent FAILS here, so 6.3.1 and
			// 6.3.2 are never read and the missing <complexType>/<restriction> ancestors
			// are no fault at all.
			name: "targetNamespace equal to the schema's is accepted with no ancestor",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:attributeGroup name="ag">
<xs:attribute name="w" type="xs:integer" targetNamespace="a"/>
</xs:attributeGroup>
</xs:schema>`,
		},
		{
			// The one shape clause 6 exists to admit, carried all the way through
			// finalization: the base's <anyAttribute> is what lets a {b}w use ·restrict·
			// an {a}-namespace base at all (derivation-ok-restriction clause 3, c-ran),
			// so the row asserts acceptance rather than merely the absence of a clause-6
			// verdict.
			name: "restriction whose base is not anyType is accepted",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a" xmlns:tns="a">
<xs:complexType name="base">
<xs:anyAttribute namespace="##other" processContents="lax"/>
</xs:complexType>
<xs:complexType name="ct"><xs:complexContent><xs:restriction base="tns:base">
<xs:attribute name="w" type="xs:integer" targetNamespace="b"/>
</xs:restriction></xs:complexContent></xs:complexType>
</xs:schema>`,
		},
		{
			name: "no targetNamespace at all is untouched by clause 6",
			doc: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="a">
<xs:complexType name="ct">
<xs:attribute name="w" type="xs:integer"/>
</xs:complexType>
</xs:schema>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			if tc.wantMsg == "" {
				if err != nil {
					t.Fatalf("Produce rejected a document src-attribute clause 6 admits: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Produce accepted the document, want the src-attribute clause 6 fault %q", tc.wantMsg)
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

// TestProduceRefAttributeTargetNamespaceStaysAccepted pins the BOUNDARY clause 6
// was deliberately not extended to: an <attribute ref="..."> carrying
// targetNamespace is accepted.
//
// It is a gap in the charge, not a reading of the spec, and the gap itself is
// tracked at rejectLocalAttributeTargetNamespace's ref= GAP(xsd) marker in
// parser/produce_complex.go — that marker, not this comment, is the greppable
// record and carries the account of why nothing charges the form. This test
// asserts only what the producer does today, not what the spec asks for.
func TestProduceRefAttributeTargetNamespaceStaysAccepted(t *testing.T) {
	if _, err := produce(t, wrap("a", `
<xs:attribute name="A" type="xs:string"/>
<xs:complexType name="ct">
<xs:attribute ref="tns:A" targetNamespace="b"/>
</xs:complexType>`)); err != nil {
		t.Fatalf("Produce rejected an <attribute ref> carrying targetNamespace: %v — the charge was scoped to the local name= form, so this document's acceptance is the pinned behavior", err)
	}
}

// TestProduceTopLevelAttributeTargetNamespaceKeepsGrammarFault pins that clause 6
// did not take the TOP-LEVEL form's targetNamespace away from
// rejectProhibitedAttrs, which charges it as the §5.1 grammar fault
// xs:topLevelAttribute's use="prohibited" makes it (xmlschema11-1.md:4713).
// Charging src-attribute there would be a fabricated verdict (STYLE E2), so the
// absence of a rule ID is the assertion.
func TestProduceTopLevelAttributeTargetNamespaceKeepsGrammarFault(t *testing.T) {
	_, err := produce(t, wrap("a", `<xs:attribute name="w" type="xs:string" targetNamespace="b"/>`))
	if err == nil {
		t.Fatal("Produce accepted a top-level <attribute> carrying targetNamespace, want the grammar fault")
	}
	if !strings.Contains(err.Error(), "carries a targetNamespace attribute, which the schema for schema documents prohibits") {
		t.Fatalf("error = %v, want the rejectProhibitedAttrs grammar fault", err)
	}
	if rule, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("error = %v, charged %s; want a plain grammar fault carrying no rule ID (STYLE E2)", err, rule)
	}
}
