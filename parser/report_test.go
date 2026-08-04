package parser_test

import (
	"io"
	"slices"
	"testing"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// reportOf assembles root over an in-memory document set and returns the
// assembly report together with the parse outcome, so a test can assert on both
// — the report is populated whether or not the parse succeeded.
func reportOf(t *testing.T, root string, docs map[string]string) (*parser.AssemblyReport, error) {
	t.Helper()
	_, report, err := parser.ParseReport(root, parser.WithResolver(loader.Map(docs)))
	if report == nil {
		t.Fatal("ParseReport returned a nil *AssemblyReport")
	}
	return report, err
}

// locations returns the reported documents' resolved locations in order.
func locations(report *parser.AssemblyReport) []string {
	out := make([]string, 0, len(report.Documents()))
	for _, d := range report.Documents() {
		out = append(out, d.Location)
	}
	return out
}

// reasons returns the reported unfollowed directives' reasons in order.
func reasons(report *parser.AssemblyReport) []parser.UnfollowedReason {
	out := make([]parser.UnfollowedReason, 0, len(report.Unfollowed()))
	for _, u := range report.Unfollowed() {
		out = append(out, u.Reason)
	}
	return out
}

// TestParseReportDocumentsAreDiscoveryOrder pins the reported order against the
// order the assembly actually resolved documents in: ONE document-order pass,
// depth-first and pre-order over <include>, <override> and <import> alike
// (§4.2.1). The interleaved import/include/import shape is what makes this able
// to fail — a report rebuilt from the load-once map, or from a kind-segregated
// pass, would come back in another order (STYLE D1/D2).
func TestParseReportDocumentsAreDiscoveryOrder(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrapImporting("urn:a", "urn:b",
			`<xs:import namespace="urn:b" schemaLocation="b1.xsd"/>`+
				`<xs:include schemaLocation="lib.xsd"/>`+
				`<xs:import namespace="urn:b" schemaLocation="b2.xsd"/>`),
		"lib.xsd": wrapImporting("urn:a", "urn:b",
			`<xs:import namespace="urn:b" schemaLocation="b3.xsd"/>`),
		"b1.xsd": wrap("urn:b", `<xs:element name="e1" type="xs:string"/>`),
		"b2.xsd": wrap("urn:b", `<xs:element name="e2" type="xs:string"/>`),
		"b3.xsd": wrap("urn:b", `<xs:element name="e3" type="xs:string"/>`),
	}
	report, err := reportOf(t, "main.xsd", docs)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	want := []string{"main.xsd", "b1.xsd", "lib.xsd", "b3.xsd", "b2.xsd"}
	if got := locations(report); !slices.Equal(got, want) {
		t.Errorf("Documents() locations = %v, want %v", got, want)
	}
	for _, d := range report.Documents() {
		if d.Doc == nil {
			t.Fatalf("reported document %q carries a nil *Document", d.Location)
		}
		if !d.Doc.IsSchema() {
			t.Errorf("reported document %q is not a <schema>", d.Location)
		}
	}
	if got := report.Unfollowed(); len(got) != 0 {
		t.Errorf("Unfollowed() = %v, want none: every directive named a document", got)
	}
}

// TestParseReportLocationIsResolvedNotRequested pins the documented distinction
// between AssembledDocument.Location (what the Resolver reported, the assembly's
// own dedup identity) and Document.URI (what was requested). The resolver here
// canonicalizes every location the way loader.Dir canonicalizes on-disk case, so
// a report that echoed the requested location instead would come back without
// the prefix.
func TestParseReportLocationIsResolvedNotRequested(t *testing.T) {
	inner := loader.Map(map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`),
		"lib.xsd":  wrap("urn:a", `<xs:element name="e" type="xs:string"/>`),
	})
	canonical := loader.ResolverFunc(func(namespace, location string) (io.ReadCloser, string, error) {
		rc, resolved, err := inner.Resolve(namespace, location)
		if err != nil {
			return nil, "", err
		}
		return rc, "canonical/" + resolved, nil
	})
	_, report, err := parser.ParseReport("main.xsd", parser.WithResolver(canonical))
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	want := []string{"canonical/main.xsd", "canonical/lib.xsd"}
	if got := locations(report); !slices.Equal(got, want) {
		t.Fatalf("Documents() locations = %v, want %v (the RESOLVED locations)", got, want)
	}
	for _, d := range report.Documents() {
		if d.Doc.URI() == d.Location {
			t.Errorf("document URI %q equals its reported Location: the requested location leaked into the report", d.Doc.URI())
		}
	}
}

// TestParseReportDedupHitIsNotUnfollowed pins the distinction the whole report
// rests on: fetch returns no document BOTH when a location does not resolve and
// when the document was already loaded, and only the first is an unfollowed
// reference. Reporting the dedup hit would mark almost every multi-document
// assembly as short of a document it in fact holds.
func TestParseReportDedupHitIsNotUnfollowed(t *testing.T) {
	report, err := reportOf(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:include schemaLocation="lib.xsd"/>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="e" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if got := locations(report); !slices.Equal(got, []string{"main.xsd", "lib.xsd"}) {
		t.Errorf("Documents() locations = %v, want [main.xsd lib.xsd]: the repeat must load once (§4.2.3)", got)
	}
	if got := report.Unfollowed(); len(got) != 0 {
		t.Errorf("Unfollowed() = %v, want none: an already-loaded document WAS followed", got)
	}
}

// TestParseReportSameDocumentTwoReadings pins that Documents is a list of
// READINGS, not a set of files: one document reached as a chameleon <include>
// (coerced into the includer's namespace) and again as a bare <import> (staying
// in no namespace) contributes two distinct component sets, so it is two
// distinct discoveries.
func TestParseReportSameDocumentTwoReadings(t *testing.T) {
	report, err := reportOf(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:import schemaLocation="lib.xsd"/>`),
		"lib.xsd": wrap("", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if got := locations(report); !slices.Equal(got, []string{"main.xsd", "lib.xsd", "lib.xsd"}) {
		t.Errorf("Documents() locations = %v, want lib.xsd twice (two readings, §4.2.5's note)", got)
	}
}

// TestParseReportUnfollowedReasons drives one shape per UnfollowedReason and
// pins both the reason and whether the assembly ALSO failed — the two facts a
// consumer combines. The reasons are what a document set that came back short
// can be blamed on, and each shape here would be indistinguishable from the
// others under a single "something was unresolved" flag.
func TestParseReportUnfollowedReasons(t *testing.T) {
	cases := []struct {
		name    string
		docs    map[string]string
		want    []parser.UnfollowedReason
		wantErr bool
	}{
		{
			// §4.2.3 clause 2.4: not an error, so the assembly completes.
			name: "<include> location that does not resolve",
			docs: map[string]string{"main.xsd": wrap("urn:a", `<xs:include schemaLocation="missing.xsd"/>`)},
			want: []parser.UnfollowedReason{parser.UnfollowedLocationUnresolved},
		},
		{
			// §4.2.6.2: the hint is optional, so this too is no error.
			name: "<import> location that does not resolve",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:import namespace="urn:b" schemaLocation="missing.xsd"/>`)},
			want: []parser.UnfollowedReason{parser.UnfollowedLocationUnresolved},
		},
		{
			// A bare <import> is explicitly legal (§4.2.6.2) and names no document.
			name: "bare <import> with no schemaLocation",
			docs: map[string]string{"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b"/>`)},
			want: []parser.UnfollowedReason{parser.UnfollowedNoLocation},
		},
		{
			// §4.2.1: an <include>'s schemaLocation is mandatory, "not a hint".
			name:    "<include> with no schemaLocation at all",
			docs:    map[string]string{"main.xsd": wrap("urn:a", `<xs:include/>`)},
			want:    []parser.UnfollowedReason{parser.UnfollowedNoSchemaLocation},
			wantErr: true,
		},
		{
			name: "<override> with no schemaLocation at all",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:override><xs:element name="e" type="xs:string"/></xs:override>`)},
			want:    []parser.UnfollowedReason{parser.UnfollowedNoSchemaLocation},
			wantErr: true,
		},
		{
			// src-include clause 1.1 wants a well-formed information set, so this is
			// a verdict AND a document the assembly never took in.
			name: "included document that resolves but is not well-formed",
			docs: map[string]string{
				"main.xsd":   wrap("urn:a", `<xs:include schemaLocation="broken.xsd"/>`),
				"broken.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e"/>`,
			},
			want:    []parser.UnfollowedReason{parser.UnfollowedUnreadable},
			wantErr: true,
		},
		{
			// Two directives, two records, in document order.
			name: "an unresolved include and a bare import",
			docs: map[string]string{"main.xsd": wrap("urn:a",
				`<xs:include schemaLocation="missing.xsd"/>`+
					`<xs:import namespace="urn:b"/>`)},
			want: []parser.UnfollowedReason{parser.UnfollowedLocationUnresolved, parser.UnfollowedNoLocation},
		},
	}
	for _, tc := range cases {
		report, err := reportOf(t, "main.xsd", tc.docs)
		if got := reasons(report); !slices.Equal(got, tc.want) {
			t.Errorf("%s: Unfollowed() reasons = %v, want %v", tc.name, got, tc.want)
		}
		if (err != nil) != tc.wantErr {
			t.Errorf("%s: ParseReport error = %v, want error: %v", tc.name, err, tc.wantErr)
		}
		// Whatever else happened, the root itself was assembled and reported.
		if got := locations(report); len(got) == 0 || got[0] != "main.xsd" {
			t.Errorf("%s: Documents() = %v, want the root first", tc.name, got)
		}
	}
}

// TestParseReportPopulatedOnCompositionError pins that a report survives a
// genuine composition rejection: the documents read BEFORE src-include clause 2
// rejected the assembly are exactly what makes the failure attributable, and a
// consumer gating on the report must see them.
func TestParseReportPopulatedOnCompositionError(t *testing.T) {
	report, err := reportOf(t, "main.xsd", map[string]string{
		"main.xsd":  wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`),
		"lib.xsd":   wrap("urn:a", `<xs:include schemaLocation="other.xsd"/>`),
		"other.xsd": wrap("urn:b", `<xs:element name="e" type="xs:string"/>`),
	})
	if err == nil {
		t.Fatal("ParseReport: want a src-include clause 2 rejection, got nil")
	}
	if got := locations(report); !slices.Equal(got, []string{"main.xsd", "lib.xsd"}) {
		t.Errorf("Documents() locations = %v, want the two documents read before the rejection", got)
	}
	if got := report.Unfollowed(); len(got) != 0 {
		t.Errorf("Unfollowed() = %v, want none: the rejected document resolved and was read", got)
	}
}

// TestParseIsParseReportWithoutTheReport pins the wrapper: Parse must return
// exactly what ParseReport does, so no caller sees two different assemblies.
func TestParseIsParseReportWithoutTheReport(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:element name="root" type="tns:code"/>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	}
	schema, err := parser.Parse("main.xsd", parser.WithResolver(loader.Map(docs)))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	reported, _, err := parser.ParseReport("main.xsd", parser.WithResolver(loader.Map(docs)))
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if (schema == nil) != (reported == nil) {
		t.Fatalf("Parse schema nil = %v, ParseReport schema nil = %v", schema == nil, reported == nil)
	}
	if _, ok := schema.Type(xsd.QName{Space: "urn:a", Local: "code"}); !ok {
		t.Error("Parse lost the included type its report-carrying twin assembles")
	}
}

// TestParseReportOnUnusableRoot pins that the report is never nil, even when
// nothing was assembled at all: a caller reads the same shape on every path.
func TestParseReportOnUnusableRoot(t *testing.T) {
	cases := map[string]map[string]string{
		"root does not resolve": {},
		"root is not a <schema>": {
			"main.xsd": `<html/>`,
		},
	}
	for name, docs := range cases {
		report, err := reportOf(t, "main.xsd", docs)
		if err == nil {
			t.Errorf("%s: ParseReport error = nil, want one", name)
		}
		if len(report.Documents()) != 0 || len(report.Unfollowed()) != 0 {
			t.Errorf("%s: report = %v/%v, want empty", name, report.Documents(), report.Unfollowed())
		}
	}
}

// TestUnfollowedReasonString pins the closed set's rendering, including the
// diagnostic form for the invalid zero value — String is reached from logging
// and error formatting, so it must never panic.
func TestUnfollowedReasonString(t *testing.T) {
	want := map[parser.UnfollowedReason]string{
		parser.UnfollowedLocationUnresolved: "location-unresolved",
		parser.UnfollowedNoLocation:         "no-location",
		parser.UnfollowedNoSchemaLocation:   "no-schema-location",
		parser.UnfollowedUnreadable:         "unreadable",
		parser.UnfollowedReason(0):          "UnfollowedReason(0)",
		parser.UnfollowedReason(99):         "UnfollowedReason(99)",
	}
	for reason, s := range want {
		if got := reason.String(); got != s {
			t.Errorf("UnfollowedReason(%d).String() = %q, want %q", uint8(reason), got, s)
		}
	}
}
