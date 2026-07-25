package conformance

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
)

// schemaSrc wraps body in a <schema> with the xs prefix bound and, when tns is
// non-empty, a targetNamespace plus a tns prefix bound to it — the shape
// parser/parse_test.go's wrap produces, so the two test suites exercise the same
// document idiom.
func schemaSrc(tns, body string) string {
	head := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	if tns != "" {
		head += ` targetNamespace="` + tns + `" xmlns:tns="` + tns + `"`
	}
	return head + `>` + body + `</xs:schema>`
}

// include is the top-level <xs:include> naming location.
func include(location string) string {
	return `<xs:include schemaLocation="` + location + `"/>`
}

// closureDecidableIn runs the harness's own discovery walk over an in-memory
// document set (loader.Map is keyed by the exact location string, so it pins the
// resolution chain the walk computes) and reports its verdict for root.
func closureDecidableIn(t *testing.T, root string, docs map[string]string) bool {
	t.Helper()
	resolver := loader.Map(docs)
	rc, resolved, err := resolver.Resolve("", root)
	if err != nil {
		t.Fatalf("resolving root %q: %v", root, err)
	}
	defer func() { _ = rc.Close() }()
	doc, err := parser.ReadDocument(root, rc)
	if err != nil {
		t.Fatalf("ReadDocument(%q): %v", root, err)
	}
	return newClosureScan(resolver, doc, resolved).decidable(doc)
}

// undecidable is a top-level list-variety simpleType: a shape schemaShapeDecidable
// refuses (its absence of a <restriction> is an unsupported-variety rejection, not
// a decidable one).
const undecidable = `<xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`

// decidableType is a top-level restriction-only simpleType — squarely inside the
// producer's decidable subset.
const decidableType = `<xs:simpleType name="code">` +
	`<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction></xs:simpleType>`

// TestClosureScanWalksIncludedDocuments proves the walk admits a closure whose
// every document is decidable, and — the point of the whole walk — DECLINES when
// the undecidable shape sits in an INCLUDED document rather than the root, which a
// root-only shape check would have vacuously admitted.
func TestClosureScanWalksIncludedDocuments(t *testing.T) {
	cases := []struct {
		name string
		docs map[string]string
		want bool
	}{
		{
			name: "whole closure decidable",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("lib.xsd")+`<xs:element name="root" type="tns:code"/>`),
				"lib.xsd":  schemaSrc("urn:a", decidableType),
			},
			want: true,
		},
		{
			name: "undecidable shape in the included document",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("lib.xsd")+`<xs:element name="root" type="xs:string"/>`),
				"lib.xsd":  schemaSrc("urn:a", undecidable),
			},
			want: false,
		},
		{
			name: "undecidable shape in a chameleon included document",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("cham.xsd")),
				"cham.xsd": schemaSrc("", `<xs:element name="e"><xs:complexType/></xs:element>`),
			},
			want: false,
		},
		{
			name: "unresolvable schemaLocation is not an error (§4.2.3 clause 2.4)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("missing.xsd")+`<xs:element name="root" type="xs:string"/>`),
			},
			want: true,
		},
		{
			name: "<include> without schemaLocation declines (target cannot be named)",
			docs: map[string]string{"main.xsd": schemaSrc("urn:a", `<xs:include/>`)},
			want: false,
		},
		{
			name: "non-schema included document is left to src-include clause 1",
			docs: map[string]string{
				"main.xsd":  schemaSrc("urn:a", include("html.xsd")),
				"html.xsd":  `<html/>`,
				"other.xsd": schemaSrc("urn:a", undecidable), // unreferenced: never walked
			},
			want: true,
		},
		{
			name: "malformed included document declines (could be an encoding limitation)",
			docs: map[string]string{
				"main.xsd":   schemaSrc("urn:a", include("broken.xsd")),
				"broken.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e"/>`,
			},
			want: false,
		},
		{
			name: "top-level <import> still declines (assembly does not follow it)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="lib.xsd"/>`),
				"lib.xsd":  schemaSrc("urn:b", decidableType),
			},
			want: false,
		},
	}
	for _, tc := range cases {
		if got := closureDecidableIn(t, "main.xsd", tc.docs); got != tc.want {
			t.Errorf("%s: closure decidable = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestClosureScanResolvesAgainstIncludingDocumentsBase pins the base-URI rule the
// whole guard rests on (§4.3.2 clause 4): a nested <include> resolves against the
// base URI of ITS OWN <include> element, so child.xsd (base "sub/child.xsd")
// naming a bare "grandchild.xsd" must reach "sub/grandchild.xsd".
//
// The decoy is what makes this test able to fail: a DECIDABLE "grandchild.xsd"
// sits at the resolver root beside an UNDECIDABLE "sub/grandchild.xsd". A walk
// that resolved the depth-2 hint against the root (or against main.xsd's base)
// would read the decoy and wrongly admit the case; only correct resolution reaches
// the undecidable one and declines.
func TestClosureScanResolvesAgainstIncludingDocumentsBase(t *testing.T) {
	tree := func(grandchildBody, decoyBody string) map[string]string {
		return map[string]string{
			"main.xsd":           schemaSrc("urn:a", include("sub/child.xsd")),
			"sub/child.xsd":      schemaSrc("urn:a", include("grandchild.xsd")),
			"sub/grandchild.xsd": schemaSrc("urn:a", grandchildBody),
			"grandchild.xsd":     schemaSrc("urn:a", decoyBody),
		}
	}
	// Depth 2 reached and decidable: the whole chain is admitted.
	if !closureDecidableIn(t, "main.xsd", tree(decidableType, undecidable)) {
		t.Error("a fully decidable relative include chain must be admitted")
	}
	// Depth 2 reached and undecidable, with a decidable decoy at the resolver root:
	// only base-relative resolution of the depth-2 hint declines here.
	if closureDecidableIn(t, "main.xsd", tree(undecidable, decidableType)) {
		t.Error("the walk admitted the case: it resolved sub/child.xsd's include against the wrong base and read the root-level decoy")
	}
}

// TestClosureScanTerminatesOnIncludeCycle proves the load-once index is doing
// spec-mandated work: <include> cycles are LEGAL (§4.2.3 — processors guard
// against infinite loops, they do not reject), so the walk must terminate and
// still decide. The undecidable variant proves termination is not achieved by
// giving up before the second document is shape-checked.
func TestClosureScanTerminatesOnIncludeCycle(t *testing.T) {
	cycle := func(libBody string) map[string]string {
		return map[string]string{
			"main.xsd": schemaSrc("urn:a", include("lib.xsd")),
			"lib.xsd":  schemaSrc("urn:a", include("main.xsd")+libBody),
		}
	}
	if !closureDecidableIn(t, "main.xsd", cycle(decidableType)) {
		t.Error("a decidable include cycle must terminate and be admitted")
	}
	if closureDecidableIn(t, "main.xsd", cycle(undecidable)) {
		t.Error("an include cycle whose second document is undecidable must decline")
	}
}

// writeSchemaTree materializes docs (relative slash-separated paths → source) under
// a fresh temp directory and returns the absolute path of root. It exercises
// execSchemaCase's real seam — a loader.Dir over the case document's own directory
// — rather than the in-memory resolver the walk-level tests use.
func writeSchemaTree(t *testing.T, root string, docs map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for _, rel := range slices.Sorted(maps.Keys(docs)) {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(docs[rel]), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, filepath.FromSlash(root))
}

// TestSchemaExecutorDecidesIncludeCases drives the real executor over multi-document
// cases on disk. Each case is decided for the right reason: the executor must agree
// with the stated validity AND disagree under the flipped expectation, which is
// what distinguishes a genuine decision from a vacuous one.
func TestSchemaExecutorDecidesIncludeCases(t *testing.T) {
	exec := newSchemaExec()
	cases := []struct {
		name        string
		docs        map[string]string
		expectValid bool
		why         string
	}{
		{
			name: "cross-document type reference assembles",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("lib.xsd")+`<xs:element name="root" type="tns:code"/>`),
				"lib.xsd":  schemaSrc("urn:a", decidableType),
			},
			expectValid: true,
			why:         "src-include clause 2.1: the includee's type resolves in the includer",
		},
		{
			name: "chameleon inclusion coerces the included namespace",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("cham.xsd")+`<xs:element name="root" type="tns:code"/>`),
				"cham.xsd": schemaSrc("", decidableType),
			},
			expectValid: true,
			why:         "src-include clause 2.3 / §F.1",
		},
		{
			name: "relative chain through a subdirectory",
			docs: map[string]string{
				"main.xsd":           schemaSrc("urn:a", include("sub/child.xsd")+`<xs:element name="root" type="tns:code"/>`),
				"sub/child.xsd":      schemaSrc("urn:a", include("grandchild.xsd")),
				"sub/grandchild.xsd": schemaSrc("urn:a", decidableType),
			},
			expectValid: true,
			why:         "§4.3.2 clause 4: depth-2 hint resolves against sub/child.xsd's base",
		},
		{
			name: "included document declares a conflicting targetNamespace",
			docs: map[string]string{
				"main.xsd":  schemaSrc("urn:a", include("other.xsd")),
				"other.xsd": schemaSrc("urn:b", decidableType),
			},
			expectValid: false,
			why:         "src-include clause 2: neither 2.1, 2.2, 2.3 nor 2.4 holds",
		},
		{
			name: "same name declared in two documents of one assembly",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("lib.xsd")+`<xs:element name="dup" type="xs:string"/>`),
				"lib.xsd":  schemaSrc("urn:a", `<xs:element name="dup" type="xs:string"/>`),
			},
			expectValid: false,
			why:         "sch-props-correct §3.17.6.1 clause 2, across documents",
		},
	}
	for _, tc := range cases {
		doc := writeSchemaTree(t, "main.xsd", tc.docs)
		if got := exec(caseSpec{kind: kindSchema, doc: doc, expectValid: tc.expectValid}); !got.IsPass() {
			t.Errorf("%s (%s): executor disagreed with expectValid=%v", tc.name, tc.why, tc.expectValid)
		}
		if exec(caseSpec{kind: kindSchema, doc: doc, expectValid: !tc.expectValid}).IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", tc.name)
		}
	}
}

// TestSchemaExecutorDeclinesUndecidableInclusion proves the false-accept guard
// end-to-end: the root alone is decidable, so a root-only shape check would decide
// the case, but the INCLUDED document carries a shape the producer does not build.
// The executor must decline for BOTH polarities — never observed-valid, never
// observed-invalid.
func TestSchemaExecutorDeclinesUndecidableInclusion(t *testing.T) {
	exec := newSchemaExec()
	trees := map[string]map[string]string{
		"undecidable directly included": {
			"main.xsd": schemaSrc("urn:a", include("lib.xsd")+`<xs:element name="root" type="xs:string"/>`),
			"lib.xsd":  schemaSrc("urn:a", undecidable),
		},
		"undecidable two levels down a relative chain": {
			"main.xsd":           schemaSrc("urn:a", include("sub/child.xsd")),
			"sub/child.xsd":      schemaSrc("urn:a", include("grandchild.xsd")),
			"sub/grandchild.xsd": schemaSrc("urn:a", undecidable),
		},
		"undecidable across a legal include cycle": {
			"main.xsd": schemaSrc("urn:a", include("lib.xsd")),
			"lib.xsd":  schemaSrc("urn:a", include("main.xsd")+undecidable),
		},
	}
	for _, name := range slices.Sorted(maps.Keys(trees)) {
		doc := writeSchemaTree(t, "main.xsd", trees[name])
		for _, ev := range []bool{true, false} {
			if exec(caseSpec{kind: kindSchema, doc: doc, expectValid: ev}).IsPass() {
				t.Errorf("%s: must be DECLINED (Fail) regardless of expectValid=%v", name, ev)
			}
		}
	}
}
