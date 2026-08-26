package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// unmappedNames renders one reported document's census as local names, so an
// assertion reads as the document's own top level does.
func unmappedNames(d parser.AssembledDocument) []string {
	out := make([]string, 0, len(d.Unmapped))
	for _, u := range d.Unmapped {
		out = append(out, u.Name.Local)
	}
	return out
}

// TestParseReportUnmappedNamesUndispatchedChildren pins the census to the
// children no dispatch of the producer maps, PER DISCOVERY: the two documents
// hold different undispatched children, so a census written back through a copy
// of the discovery — or written to one index for every document — reports the
// wrong document's top level, or none at all.
//
// Everything else on those top levels is in topLevelMapped's vocabulary and must
// stay unreported: the four §4.2.1 directives, the six named declaration kinds,
// <notation>, <defaultOpenContent> (mapped by a pass other than run's dispatch),
// and <annotation> (mapped by no pass, admitted because §3.15.1 puts annotations
// outside ·validation· altogether).
func TestParseReportUnmappedNamesUndispatchedChildren(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrap("urn:a",
			`<xs:annotation><xs:documentation>mapped</xs:documentation></xs:annotation>`+
				`<xs:defaultOpenContent mode="suffix"><xs:any namespace="urn:z"/></xs:defaultOpenContent>`+
				`<xs:include schemaLocation="lib.xsd"/>`+
				`<xs:field xpath="."/>`+
				`<xs:element name="e" type="xs:string"/>`+
				`<xs:notation name="n" public="image/jpeg"/>`),
		"lib.xsd": wrap("urn:a",
			`<xs:simpleType name="s"><xs:restriction base="xs:string"/></xs:simpleType>`+
				`<xs:selector xpath="."/>`+
				`<xs:attributeGroup name="ag"/>`),
	}
	report, err := reportOf(t, "main.xsd", docs)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	got := report.Documents()
	if len(got) != 2 {
		t.Fatalf("Documents() = %d documents, want 2", len(got))
	}
	want := map[string]string{"main.xsd": "field", "lib.xsd": "selector"}
	for _, d := range got {
		names := unmappedNames(d)
		if len(names) != 1 || names[0] != want[d.Location] {
			t.Errorf("%s: Unmapped = %v, want exactly [%s]", d.Location, names, want[d.Location])
			continue
		}
		u := d.Unmapped[0]
		if u.Name != (xsd.QName{Space: xsdNS, Local: want[d.Location]}) {
			t.Errorf("%s: Unmapped[0].Name = %s, want the expanded XSD-namespace name", d.Location, u.Name)
		}
		if u.Reason != parser.UnmappedNoDispatch {
			t.Errorf("%s: Unmapped[0].Reason = %s, want %s", d.Location, u.Reason, parser.UnmappedNoDispatch)
		}
		if u.At.URI != d.Location {
			t.Errorf("%s: Unmapped[0].At = %s, want a position in this document", d.Location, u.At)
		}
	}
}

// TestParseReportUnmappedIsDocumentOrder pins the census's order to the
// document's, which is what makes it readable against the source at all: the
// three undispatched children are interleaved with mapped ones, so a census
// gathered per KIND, or appended as build order reached each site, comes back
// permuted (STYLE D2).
func TestParseReportUnmappedIsDocumentOrder(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrap("urn:a",
			`<xs:sequence/>`+
				`<xs:element name="e" type="xs:string"/>`+
				`<xs:field xpath="."/>`+
				`<xs:complexType name="ct"/>`+
				`<xs:choice/>`),
	}
	report, err := reportOf(t, "main.xsd", docs)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if len(report.Documents()) != 1 {
		t.Fatalf("Documents() = %d documents, want 1", len(report.Documents()))
	}
	got := unmappedNames(report.Documents()[0])
	want := []string{"sequence", "field", "choice"}
	if len(got) != len(want) {
		t.Fatalf("Unmapped = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Unmapped = %v, want %v", got, want)
		}
	}
}

// TestParseReportUnmappedEmptyForMappedDocument pins the other half: a document
// whose whole top level the producer maps reports nothing. Without it the census
// could satisfy every other test here by flagging everything.
func TestParseReportUnmappedEmptyForMappedDocument(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrap("urn:a",
			`<xs:element name="e" type="xs:string"/>`+
				`<xs:attribute name="a" type="xs:string"/>`+
				`<xs:group name="g"><xs:sequence/></xs:group>`+
				`<xs:attributeGroup name="ag"/>`+
				`<xs:simpleType name="s"><xs:restriction base="xs:string"/></xs:simpleType>`+
				`<xs:complexType name="ct"/>`),
	}
	report, err := reportOf(t, "main.xsd", docs)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	for _, d := range report.Documents() {
		if len(d.Unmapped) != 0 {
			t.Errorf("%s: Unmapped = %v, want none: every top-level child is dispatched", d.Location, unmappedNames(d))
		}
	}
}

// TestUnmappedReasonString pins the reason's rendered form, the identifier a log
// line or a consumer's own report carries, and the diagnostic form of a value
// outside the closed set — which must not panic, being reached from formatting.
func TestUnmappedReasonString(t *testing.T) {
	if got := parser.UnmappedNoDispatch.String(); got != "no-dispatch" {
		t.Errorf("UnmappedNoDispatch.String() = %q, want %q", got, "no-dispatch")
	}
	var zero parser.UnmappedReason
	if got := zero.String(); got != "UnmappedReason(0)" {
		t.Errorf("UnmappedReason(0).String() = %q, want %q", got, "UnmappedReason(0)")
	}
}
