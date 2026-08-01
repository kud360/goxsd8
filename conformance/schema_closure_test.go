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

// override is the top-level <xs:override> naming location and carrying body as
// its substituting children.
func override(location, body string) string {
	return `<xs:override schemaLocation="` + location + `">` + body + `</xs:override>`
}

// closureDecidableIn runs the harness's own discovery walk over an in-memory
// document set (loader.Map is keyed by the exact location string, so it pins the
// resolution chain the walk computes) and reports its two verdicts for root: the
// decidability of every document it reached, and whether some directive in the
// closure named no document at all (closureScan.unresolved) — the fact
// execSchemaCase pairs with the parse outcome (#276).
func closureDecidableIn(t *testing.T, root string, docs map[string]string) (decidable, unresolved bool) {
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
	rootTNS, _ := elementAttr(doc.Root(), "targetNamespace")
	scan := newClosureScan(resolver, resolved, rootTNS)
	return scan.decidable(doc, rootTNS), scan.unresolved
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
//
// wantUnresolved pins the walk's second output (#276): a directive that named no
// document leaves the walk's decidability verdict alone — there is no shape to
// gate — and is recorded instead, for execSchemaCase to pair with the parse
// outcome. Every other case must leave that flag CLEAR, which is what keeps the
// recording specific to a genuinely absent document.
func TestClosureScanWalksIncludedDocuments(t *testing.T) {
	cases := []struct {
		name           string
		docs           map[string]string
		want           bool
		wantUnresolved bool
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
			name: "<include> whose schemaLocation does not resolve is recorded, not declined (§4.2.3 cl. 2.4, #276)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("missing.xsd")+`<xs:element name="root" type="xs:string"/>`),
			},
			want:           true,
			wantUnresolved: true,
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
			name: "<import> with a decidable D2 is walked and admitted (#182)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="lib.xsd"/>`),
				"lib.xsd":  schemaSrc("urn:b", decidableType),
			},
			want: true,
		},
		{
			name: "undecidable shape in the <import>ed document declines",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="lib.xsd"/>`),
				"lib.xsd":  schemaSrc("urn:b", undecidable),
			},
			want: false,
		},
		{
			name: "undecidable shape reached through an <import>ed document's own <include>",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="lib.xsd"/>`),
				"lib.xsd":  schemaSrc("urn:b", include("deep.xsd")),
				"deep.xsd": schemaSrc("urn:b", undecidable),
			},
			want: false,
		},
		{
			name: "bare <import> with no schemaLocation is recorded, not declined (§4.2.6.2, #276)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b"/>`),
			},
			want:           true,
			wantUnresolved: true,
		},
		{
			name: "<import> whose schemaLocation does not resolve is recorded, not declined (#276)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="missing.xsd"/>`),
			},
			want:           true,
			wantUnresolved: true,
		},
		{
			name: "malformed <import>ed document declines (could be an encoding limitation)",
			docs: map[string]string{
				"main.xsd":   schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="broken.xsd"/>`),
				"broken.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e"/>`,
			},
			want: false,
		},
		{
			name: "non-schema <import>ed document is left to src-import clause 2",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="html.xsd"/>`),
				"html.xsd": `<html/>`,
			},
			want: true,
		},
		{
			name: "<override> with a decidable Dold is walked and admitted (#183)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", override("lib.xsd", `<xs:element name="e" type="xs:string"/>`)),
				"lib.xsd":  schemaSrc("urn:a", `<xs:element name="e" type="xs:int"/>`),
			},
			want: true,
		},
		{
			name: "undecidable shape in the <override>n document declines",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", override("lib.xsd", `<xs:element name="e" type="xs:string"/>`)),
				"lib.xsd":  schemaSrc("urn:a", undecidable),
			},
			want: false,
		},
		{
			name: "undecidable shape reached through the ·target set· (§F.2 clause 3)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", override("mid.xsd", `<xs:element name="e" type="xs:string"/>`)),
				"mid.xsd":  schemaSrc("urn:a", include("deep.xsd")),
				"deep.xsd": schemaSrc("urn:a", undecidable),
			},
			want: false,
		},
		{
			name: "undecidable shape reached through a nested <override> (§F.2 clause 4)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", override("mid.xsd", `<xs:element name="e" type="xs:string"/>`)),
				"mid.xsd":  schemaSrc("urn:a", override("deep.xsd", `<xs:element name="f" type="xs:string"/>`)),
				"deep.xsd": schemaSrc("urn:a", undecidable),
			},
			want: false,
		},
		{
			name: "undecidable shape in a chameleon <override>n document",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", override("cham.xsd", `<xs:element name="e" type="xs:string"/>`)),
				"cham.xsd": schemaSrc("", undecidable),
			},
			want: false,
		},
		{
			name: "<override> whose schemaLocation does not resolve is recorded, not declined (§4.3.2, #276)",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", override("missing.xsd", `<xs:element name="e" type="xs:string"/>`)+
					`<xs:element name="root" type="xs:string"/>`),
			},
			want:           true,
			wantUnresolved: true,
		},
		{
			name: "<override> without schemaLocation declines (target cannot be named)",
			docs: map[string]string{"main.xsd": schemaSrc("urn:a", `<xs:override><xs:element name="e" type="xs:string"/></xs:override>`)},
			want: false,
		},
		{
			name: "non-schema <override>n document is left to src-override clause 1",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", override("html.xsd", `<xs:element name="e" type="xs:string"/>`)),
				"html.xsd": `<html/>`,
			},
			want: true,
		},
	}
	for _, tc := range cases {
		got, unresolved := closureDecidableIn(t, "main.xsd", tc.docs)
		if got != tc.want {
			t.Errorf("%s: closure decidable = %v, want %v", tc.name, got, tc.want)
		}
		if unresolved != tc.wantUnresolved {
			t.Errorf("%s: closure unresolved = %v, want %v", tc.name, unresolved, tc.wantUnresolved)
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
	if decidable, _ := closureDecidableIn(t, "main.xsd", tree(decidableType, undecidable)); !decidable {
		t.Error("a fully decidable relative include chain must be admitted")
	}
	// Depth 2 reached and undecidable, with a decidable decoy at the resolver root:
	// only base-relative resolution of the depth-2 hint declines here.
	if decidable, _ := closureDecidableIn(t, "main.xsd", tree(undecidable, decidableType)); decidable {
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
	if decidable, _ := closureDecidableIn(t, "main.xsd", cycle(decidableType)); !decidable {
		t.Error("a decidable include cycle must terminate and be admitted")
	}
	if decidable, _ := closureDecidableIn(t, "main.xsd", cycle(undecidable)); decidable {
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
		if got := exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(tc.expectValid)}); !got.IsPass() {
			t.Errorf("%s (%s): executor disagreed with expectValid=%v", tc.name, tc.why, tc.expectValid)
		}
		if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(!tc.expectValid)}).IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", tc.name)
		}
	}
}

// TestSchemaExecutorDecidesReachableExtraDocuments proves a multi-document
// schemaTest whose further documents are REACHED by the closure walk from the
// first is decided normally: the walk gated them, so parser.Parse assembles
// exactly the declared set. Each case must also Fail under the flipped
// expectation, which is what separates a real decision from a vacuous one.
func TestSchemaExecutorDecidesReachableExtraDocuments(t *testing.T) {
	exec := newSchemaExec()
	cases := []struct {
		name        string
		docs        map[string]string
		extra       []string
		expectValid bool
	}{
		{
			name: "second document reached by the first's <include>",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("lib.xsd")+`<xs:element name="root" type="tns:code"/>`),
				"lib.xsd":  schemaSrc("urn:a", decidableType),
			},
			extra:       []string{"lib.xsd"},
			expectValid: true,
		},
		{
			name: "second document reached by the first's <import>",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", `<xs:import namespace="urn:b" schemaLocation="lib.xsd"/>`+
					`<xs:element name="root" type="xs:string"/>`),
				"lib.xsd": schemaSrc("urn:b", decidableType),
			},
			extra:       []string{"lib.xsd"},
			expectValid: true,
		},
		{
			name: "second document reached two levels down, invalid across the set",
			docs: map[string]string{
				"main.xsd": schemaSrc("urn:a", include("mid.xsd")+`<xs:element name="dup" type="xs:string"/>`),
				"mid.xsd":  schemaSrc("urn:a", include("deep.xsd")),
				"deep.xsd": schemaSrc("urn:a", `<xs:element name="dup" type="xs:string"/>`),
			},
			extra:       []string{"mid.xsd", "deep.xsd"},
			expectValid: false,
		},
	}
	for _, tc := range cases {
		root := writeSchemaTree(t, "main.xsd", tc.docs)
		spec := caseSpec{
			kind:      kindSchema,
			doc:       root,
			extraDocs: extraPaths(root, tc.extra),
			expect:    expectValidity(tc.expectValid),
		}
		if !exec(spec).IsPass() {
			t.Errorf("%s: executor disagreed with expectValid=%v", tc.name, tc.expectValid)
		}
		flipped := spec
		flipped.expect = expectValidity(!tc.expectValid)
		if exec(flipped).IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", tc.name)
		}
	}
}

// TestSchemaExecutorDeclinesUnreachableExtraDocument proves the decline-not-guess
// rule for a multi-document schemaTest (issue #238): when a declared document is
// NOT reached by the closure walk from the first, parser.Parse — which takes one
// root — would assemble a schema the suite never declared, so the case must be
// DECLINED under BOTH polarities rather than decided against a subset.
//
// The decoy is what makes this able to fail: "other.xsd" declares the same name
// as the root, so a harness that merged the declared documents (or one that
// simply ignored the extra) would produce a decidable verdict either way —
// "invalid" if merged, "valid" if ignored. Only declining refuses both.
func TestSchemaExecutorDeclinesUnreachableExtraDocument(t *testing.T) {
	exec := newSchemaExec()
	trees := map[string][]string{
		"independent second document":            {"other.xsd"},
		"declared document absent from the tree": {"missing.xsd"},
	}
	docs := map[string]string{
		"main.xsd":  schemaSrc("urn:a", `<xs:element name="dup" type="xs:string"/>`),
		"other.xsd": schemaSrc("urn:a", `<xs:element name="dup" type="xs:string"/>`),
	}
	for _, name := range slices.Sorted(maps.Keys(trees)) {
		root := writeSchemaTree(t, "main.xsd", docs)
		for _, ev := range []bool{true, false} {
			spec := caseSpec{
				kind:      kindSchema,
				doc:       root,
				extraDocs: extraPaths(root, trees[name]),
				expect:    expectValidity(ev),
			}
			if exec(spec).IsPass() {
				t.Errorf("%s: must be DECLINED (Fail) regardless of expectValid=%v", name, ev)
			}
		}
	}
}

// extraPaths turns slash-separated names relative to root's directory into the
// resolved paths caseSpec.extraDocs carries, matching makeCase's own join.
func extraPaths(root string, names []string) []string {
	dir := filepath.Dir(root)
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, filepath.Join(dir, filepath.FromSlash(n)))
	}
	return out
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
			if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(ev)}).IsPass() {
				t.Errorf("%s: must be DECLINED (Fail) regardless of expectValid=%v", name, ev)
			}
		}
	}
}

// noD2Trees returns one single-document tree per composition directive that names
// no document at all: an <include>, an <override> and an <import> whose
// schemaLocation does not resolve, plus the bare <import> §4.2.6.2 calls legal.
// sameNSBody is the extra top-level content of the two same-namespace roots and
// foreignNSBody that of the two <import> roots (which bind the b: prefix to the
// imported namespace instead).
//
// The two tests below drive these SAME four shapes under the only variable that
// separates them (#276): whether the root refers to something only the missing
// document could have supplied — i.e. whether parser.Parse fails.
func noD2Trees(sameNSBody, foreignNSBody string) map[string]map[string]string {
	foreign := func(directive string) map[string]string {
		return map[string]string{"main.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` +
			` targetNamespace="urn:a" xmlns:b="urn:b">` + directive + foreignNSBody + `</xs:schema>`}
	}
	return map[string]map[string]string{
		"unresolved <include> target": {
			"main.xsd": schemaSrc("urn:a", include("missing.xsd")+sameNSBody),
		},
		"unresolved <override> target": {
			"main.xsd": schemaSrc("urn:a",
				override("missing.xsd", `<xs:element name="e" type="xs:string"/>`)+sameNSBody),
		},
		"unresolved <import> target":        foreign(`<xs:import namespace="urn:b" schemaLocation="missing.xsd"/>`),
		"bare <import> with no D2 to fetch": foreign(`<xs:import namespace="urn:b"/>`),
	}
}

// TestSchemaExecutorDeclinesUnresolvedDirectiveTarget pins the whole hazard #276
// is about, symmetrically across all three directives: a directive that names no
// document contributes NO components, so the root's reference into what that
// document would have defined fails src-resolve clauses 1-3 at finalize. The
// "invalid" that comes back is fabricated — src-include clause 2.4 and
// src-import's parallel "not an error" text both say the failure to resolve is
// itself no error — so the case must be DECLINED under BOTH polarities.
//
// The reference is what makes this able to fail: each root names a type only the
// missing document could have defined, so a harness that decided the case anyway
// observes "invalid" and PASSES the expectValid=false polarity for the wrong
// reason. It is also exactly what TestSchemaExecutorDecidesUnbrokenUnresolvedDirective
// removes, and the two together pin the CONJUNCTION rather than either half.
func TestSchemaExecutorDeclinesUnresolvedDirectiveTarget(t *testing.T) {
	exec := newSchemaExec()
	trees := noD2Trees(`<xs:element name="root" type="tns:code"/>`, `<xs:element name="root" type="b:code"/>`)
	for _, name := range slices.Sorted(maps.Keys(trees)) {
		doc := writeSchemaTree(t, "main.xsd", trees[name])
		for _, ev := range []bool{true, false} {
			if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(ev)}).IsPass() {
				t.Errorf("%s: must be DECLINED (Fail) regardless of expectValid=%v", name, ev)
			}
		}
	}
}

// TestSchemaExecutorDecidesUnbrokenUnresolvedDirective pins the other half of the
// same conjunction (#276): when nothing in the root depended on the missing
// document, parser.Parse succeeds and §4.2.3 clause 2.4 — "it is not an error for
// the ·actual value· of the schemaLocation [attribute] to fail to resolve at all,
// in which case the corresponding inclusion must not be performed" — is simply in
// force, §4.2.6.2 saying the same of <import>. Nothing was fabricated, so the case
// must still be DECIDED: valid, and Fail under the flipped expectation.
//
// The suite depends on this directly — MS-Schema schD8 includes a document named
// "must%20not%20resolve.xyzzy" precisely to test clause 2.4 tolerance — so a
// harness that declined every unresolved directive would refuse the very cases the
// rule is about. The flipped-expectation half is what makes the test able to fail:
// a decline would satisfy the first assertion vacuously.
func TestSchemaExecutorDecidesUnbrokenUnresolvedDirective(t *testing.T) {
	exec := newSchemaExec()
	unrelated := `<xs:element name="root" type="xs:string"/>`
	trees := noD2Trees(unrelated, unrelated)
	for _, name := range slices.Sorted(maps.Keys(trees)) {
		doc := writeSchemaTree(t, "main.xsd", trees[name])
		if !exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(true)}).IsPass() {
			t.Errorf("%s: an unresolved directive nothing depended on must still be DECIDED valid", name)
		}
		if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(false)}).IsPass() {
			t.Errorf("%s: must Fail under a flipped expectation (decides for real)", name)
		}
	}
}
