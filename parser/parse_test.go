package parser_test

import (
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// parseMap runs Parse over an in-memory document set, resolving locations by
// map key.
func parseMap(t *testing.T, root string, docs map[string]string) (*xsd.Schema, error) {
	t.Helper()
	return parser.Parse(root, parser.WithResolver(loader.Map(docs)))
}

// mustType fails unless the schema holds a type definition with the given
// expanded name.
func mustType(t *testing.T, s *xsd.Schema, name xsd.QName) {
	t.Helper()
	if _, ok := s.Type(name); !ok {
		t.Fatalf("type %s not found in assembled schema", name)
	}
}

// TestParseSingleDocument checks the degenerate assembly: a root with no
// <include> is schema(D) = immed(D) (§4.2.1).
func TestParseSingleDocument(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:element name="root" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "root"}); !ok {
		t.Fatalf("element {urn:a}root not found")
	}
}

// TestParseIncludeSameNamespace covers src-include clause 2.1 (c-normi): the
// included document's components join the including schema (clause 3.1.2,
// c-incl-incl), and a reference in the includer resolves to a definition
// contributed by the includee.
func TestParseIncludeSameNamespace(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:element name="root" type="tns:code"/>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction>`+
			`</xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
	ed, ok := s.Element(xsd.QName{Space: "urn:a", Local: "root"})
	if !ok {
		t.Fatalf("element {urn:a}root not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != (xsd.QName{Space: "urn:a", Local: "code"}) {
		t.Fatalf("root type = %s, want {urn:a}code", got)
	}
}

// TestParseIncludeCrossDocumentBase exercises the ASSEMBLY-WIDE pre-scan, which
// runs for every document before any document is produced: a base= is discharged
// eagerly at production time (xsd.NewSimpleType needs a live base pointer), so
// each direction of a cross-document base chain needs the other document already
// registered (§4.2.3 c-incl-incl). The forward direction — the INCLUDING
// document, produced first, deriving from a type the INCLUDED one defines — is
// the one a per-document pre-scan cannot serve.
func TestParseIncludeCrossDocumentBase(t *testing.T) {
	restriction := func(name, base string) string {
		return `<xs:simpleType name="` + name + `">` +
			`<xs:restriction base="` + base + `"><xs:minLength value="2"/></xs:restriction>` +
			`</xs:simpleType>`
	}
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			restriction("fromLib", "tns:libBase")+
			`<xs:simpleType name="mainBase">`+
			`<xs:restriction base="xs:string"><xs:maxLength value="8"/></xs:restriction>`+
			`</xs:simpleType>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="libBase">`+
			`<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction>`+
			`</xs:simpleType>`+
			restriction("fromMain", "tns:mainBase")),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "fromLib"})
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "fromMain"})
}

// TestParseChameleonCoercion covers src-include clause 2.3 (c-chami) and the
// §F.1 transformation's BOTH tasks: (a) the included no-namespace document's
// components are minted in the including namespace, and (b) its unqualified
// QName references — here an intra-document sibling reference — are coerced to
// that same namespace rather than left ·absent·.
func TestParseChameleonCoercion(t *testing.T) {
	chameleon := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
		`<xs:simpleType name="code">` +
		`<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction>` +
		`</xs:simpleType>` +
		`<xs:element name="local" type="code"/>` +
		`</xs:schema>`
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="cham.xsd"/>`+
			`<xs:element name="root" type="tns:code"/>`),
		"cham.xsd": chameleon,
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// Task (a): the coerced component carries the including namespace, and the
	// no-namespace name it would otherwise have carried is absent.
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
	if _, ok := s.Type(xsd.QName{Local: "code"}); ok {
		t.Fatalf("type {}code exists, but §F.1 task (a) coerces it to {urn:a}code")
	}
	// Task (b): the chameleon document's own unqualified type= reference resolves
	// to the coerced name, not to the absent namespace.
	ed, ok := s.Element(xsd.QName{Space: "urn:a", Local: "local"})
	if !ok {
		t.Fatalf("element {urn:a}local not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != (xsd.QName{Space: "urn:a", Local: "code"}) {
		t.Fatalf("chameleon type reference = %s, want {urn:a}code", got)
	}
}

// TestParseChameleonTransitive covers §4.2.3's recursion note: A includes B and
// B includes C, only A has a targetNamespace, so C's components are coerced too.
func TestParseChameleonTransitive(t *testing.T) {
	bare := func(body string) string {
		return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xs:schema>`
	}
	s, err := parseMap(t, "a.xsd", map[string]string{
		"a.xsd": wrap("urn:a", `<xs:include schemaLocation="b.xsd"/>`),
		"b.xsd": bare(`<xs:include schemaLocation="c.xsd"/><xs:element name="inB" type="deep"/>`),
		"c.xsd": bare(`<xs:simpleType name="deep">` +
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "deep"})
	ed, ok := s.Element(xsd.QName{Space: "urn:a", Local: "inB"})
	if !ok {
		t.Fatalf("element {urn:a}inB not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != (xsd.QName{Space: "urn:a", Local: "deep"}) {
		t.Fatalf("cross-document chameleon reference = %s, want {urn:a}deep", got)
	}
}

// TestParseChameleonIncludesCoercedNamespace pins §4.2.3's recursion note: A
// (targetNamespace urn:a) includes chameleon B, which itself includes C declaring
// targetNamespace urn:a. Clause 2 is evaluated against the COERCED namespace —
// the effect is as if A included B' and B' included C', with B' carrying A's
// targetNamespace — so this is legal, not a src-include violation.
func TestParseChameleonIncludesCoercedNamespace(t *testing.T) {
	s, err := parseMap(t, "a.xsd", map[string]string{
		"a.xsd": wrap("urn:a", `<xs:include schemaLocation="b.xsd"/>`),
		"b.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<xs:include schemaLocation="c.xsd"/></xs:schema>`,
		"c.xsd": wrap("urn:a", `<xs:element name="inC" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "inC"}); !ok {
		t.Fatalf("element {urn:a}inC not found")
	}
}

// TestParseChameleonKeepsQualifiedReferences guards the other half of §F.1 task
// (b): only UNQUALIFIED references are coerced. A reference qualified through an
// in-scope default namespace keeps that namespace.
func TestParseChameleonKeepsQualifiedReferences(t *testing.T) {
	// The chameleon document binds the XSD namespace as its DEFAULT namespace, so
	// its unprefixed base="string" is already qualified as {xs}string.
	cham := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="` + xsdNS + `">` +
		`<xs:simpleType name="code"><xs:restriction base="string"/></xs:simpleType>` +
		`</xs:schema>`
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="cham.xsd"/>`),
		"cham.xsd": cham,
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
}

// TestParseIncludeIdempotent covers §4.2.3's closing note: <include>ing the same
// schema document more than once — here both directly twice and through a
// cycle — must not violate sch-props-correct clause 2. Cycles are spec-legal.
func TestParseIncludeIdempotent(t *testing.T) {
	for _, tc := range []struct {
		name string
		docs map[string]string
	}{
		{
			name: "twice from one document",
			docs: map[string]string{
				"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
					`<xs:include schemaLocation="lib.xsd"/>`),
				"lib.xsd": wrap("urn:a", `<xs:element name="shared" type="xs:string"/>`),
			},
		},
		{
			name: "through a cycle",
			docs: map[string]string{
				"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`),
				"lib.xsd": wrap("urn:a", `<xs:include schemaLocation="main.xsd"/>`+
					`<xs:element name="shared" type="xs:string"/>`),
			},
		},
		{
			name: "diamond",
			docs: map[string]string{
				"main.xsd": wrap("urn:a", `<xs:include schemaLocation="left.xsd"/>`+
					`<xs:include schemaLocation="right.xsd"/>`),
				"left.xsd":  wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`),
				"right.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`),
				"lib.xsd":   wrap("urn:a", `<xs:element name="shared" type="xs:string"/>`),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := parseMap(t, "main.xsd", tc.docs)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "shared"}); !ok {
				t.Fatalf("element {urn:a}shared not found")
			}
		})
	}
}

// TestParseIncludeNamespaceMismatch covers the one src-include clause 2 failure:
// the included document resolves but declares a DIFFERENT non-absent
// targetNamespace, so none of clauses 2.1-2.4 holds.
func TestParseIncludeNamespaceMismatch(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="other.xsd"/>`),
		"other.xsd": wrap("urn:b", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("Parse error = %v, want an *xsderr.Error", err)
	}
	if xe.Rule != "src-include" {
		t.Fatalf("rule = %q, want src-include", xe.Rule)
	}
	if xe.Loc.URI != "main.xsd" {
		t.Fatalf("loc = %s, want the <include> element in main.xsd", xe.Loc)
	}
}

// TestParseIncludeNoNamespaceIncluderMismatch is the mirror case: the includer
// has no targetNamespace and the includee does, so clause 2.1 fails for want of
// a targetNamespace on D1.
func TestParseIncludeNoNamespaceIncluderMismatch(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd":  wrap("", `<xs:include schemaLocation="other.xsd"/>`),
		"other.xsd": wrap("urn:b", `<xs:element name="e" type="xs:string"/>`),
	})
	var xe *xsderr.Error
	if !errors.As(err, &xe) || xe.Rule != "src-include" {
		t.Fatalf("Parse error = %v, want an *xsderr.Error with rule src-include", err)
	}
}

// TestParseIncludeNeitherNamespace covers src-include clause 2.2 (c-normi2):
// neither document has a targetNamespace, so the components merge unqualified
// and NO chameleon coercion happens.
func TestParseIncludeNeitherNamespace(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("", `<xs:include schemaLocation="lib.xsd"/>`),
		"lib.xsd": wrap("", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`+
			`<xs:element name="e" type="code"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Local: "code"})
	ed, ok := s.Element(xsd.QName{Local: "e"})
	if !ok {
		t.Fatalf("element {}e not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != (xsd.QName{Local: "code"}) {
		t.Fatalf("type = %s, want the no-namespace {}code", got)
	}
}

// TestParseIncludeUnresolvable covers src-include clause 2.4: a schemaLocation
// that does not resolve is NOT an error; the inclusion is simply not performed.
func TestParseIncludeUnresolvable(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="missing.xsd"/>`+
			`<xs:element name="root" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v, want the unresolvable inclusion to be skipped silently", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "root"}); !ok {
		t.Fatalf("element {urn:a}root not found")
	}
}

// TestParseIncludeNotWellFormed covers src-include clause 1.1: a location that
// DOES resolve but yields a document that is not well-formed is an error, unlike
// a location that fails to resolve.
func TestParseIncludeNotWellFormed(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="broken.xsd"/>`),
		"broken.xsd": `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
			`<xs:element name="e"/>`, // unclosed <xs:schema>
	})
	var xe *xsderr.Error
	if !errors.As(err, &xe) || xe.Rule != "src-include" {
		t.Fatalf("Parse error = %v, want an *xsderr.Error with rule src-include", err)
	}
}

// TestParseIncludeNotASchema covers src-include clause 1: the resolved document
// element must be <schema>.
func TestParseIncludeNotASchema(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd":       wrap("urn:a", `<xs:include schemaLocation="notaschema.xsd"/>`),
		"notaschema.xsd": `<html/>`,
	})
	var xe *xsderr.Error
	if !errors.As(err, &xe) || xe.Rule != "src-include" {
		t.Fatalf("Parse error = %v, want an *xsderr.Error with rule src-include", err)
	}
}

// TestParseIncludeMissingSchemaLocation checks the grammar-level case: the
// attribute the schema for schema documents requires is absent. It is distinct
// from an unresolvable location (which is not an error at all).
func TestParseIncludeMissingSchemaLocation(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include/>`),
	})
	if err == nil {
		t.Fatalf("Parse succeeded, want an error for <include> without schemaLocation")
	}
	var xe *xsderr.Error
	if errors.As(err, &xe) {
		t.Fatalf("error = %v, want a plain error: no Schema Representation Constraint governs a missing required attribute", err)
	}
}

// TestParseIncludeRelativeToBase checks that a nested <include> resolves its
// location against the base URI in scope at the <include> element, not against
// the root document's location (§4.3.2 clause 4).
func TestParseIncludeRelativeToBase(t *testing.T) {
	s, err := parseMap(t, "top/main.xsd", map[string]string{
		"top/main.xsd":    wrap("urn:a", `<xs:include schemaLocation="sub/mid.xsd"/>`),
		"top/sub/mid.xsd": wrap("urn:a", `<xs:include schemaLocation="../leaf.xsd"/>`),
		"top/leaf.xsd":    wrap("urn:a", `<xs:element name="leaf" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "leaf"}); !ok {
		t.Fatalf("element {urn:a}leaf not found: nested include did not resolve against its own base")
	}
}

// TestParseDuplicateAcrossDocuments checks that assembly feeds sch-props-correct
// clause 2 correctly: two documents declaring the same expanded name collide,
// while the once-seeded builtins do not collide with themselves.
func TestParseDuplicateAcrossDocuments(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:element name="dup" type="xs:string"/>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="dup" type="xs:string"/>`),
	})
	var xe *xsderr.Error
	if !errors.As(err, &xe) || xe.Rule != "sch-props-correct" {
		t.Fatalf("Parse error = %v, want an *xsderr.Error with rule sch-props-correct", err)
	}
}

// TestParseRootUnresolvable checks that a root location that does not resolve is
// a plain error: unlike an <include>, the caller named a document that must
// exist, so §4.2.3's "not an error to fail to resolve" does not apply.
func TestParseRootUnresolvable(t *testing.T) {
	_, err := parseMap(t, "missing.xsd", map[string]string{})
	if !errors.Is(err, loader.ErrNotFound) {
		t.Fatalf("Parse error = %v, want it to wrap loader.ErrNotFound", err)
	}
}

// TestParseRootNotASchema checks the caller-precondition fault: a root whose
// document element is not <schema> is a plain error, not a validity verdict
// (§3.17.2 allows <schema> not to be the document element).
func TestParseRootNotASchema(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": `<html/>`})
	if err == nil {
		t.Fatalf("Parse succeeded on a non-schema root")
	}
	var xe *xsderr.Error
	if errors.As(err, &xe) {
		t.Fatalf("error = %v, want a plain error, not a validity verdict", err)
	}
}

// TestWithResolverNilPanics pins the nil-argument convention: a nil resolver is
// a caller bug, guarded like xsd.SchemaBuilder.AddType's nil component.
func TestWithResolverNilPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("WithResolver(nil) did not panic")
		}
	}()
	parser.WithResolver(nil)
}

// TestWithBackendNilPanics pins the same convention for the backend.
func TestWithBackendNilPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("WithBackend(nil) did not panic")
		}
	}()
	parser.WithBackend(nil)
}

// TestWithLoggerNilIsSilent checks that a nil logger selects the silent default
// rather than panicking or dereferencing nil (STYLE L1).
func TestWithLoggerNilIsSilent(t *testing.T) {
	_, err := parser.Parse("main.xsd",
		parser.WithResolver(loader.Map(map[string]string{"main.xsd": wrap("urn:a", "")})),
		parser.WithLogger(nil))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
}

// wrapImporting wraps body inside a <schema> with targetNamespace target — the
// xs and tns prefixes bound exactly as wrap binds them — plus the prefix "imp"
// bound to imported, so a cross-namespace QName reference can be written.
func wrapImporting(target, imported, body string) string {
	return `<xs:schema xmlns:xs="` + xsdNS + `" targetNamespace="` + target +
		`" xmlns:tns="` + target + `" xmlns:imp="` + imported + `">` + body + `</xs:schema>`
}

// mustXSDRule fails unless err is an *xsderr.Error charging rule and positioned
// in the document wantURI — the offending <import> is always in a NAMED document,
// so pinning it keeps a test from passing on a verdict reached elsewhere in the
// assembly.
func mustXSDRule(t *testing.T, err error, rule xsderr.Rule, wantURI string) {
	t.Helper()
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("Parse error = %v, want an *xsderr.Error charging %s", err, rule)
	}
	if xe.Rule != rule {
		t.Fatalf("rule = %q, want %s", xe.Rule, rule)
	}
	if xe.Loc.URI != wantURI {
		t.Fatalf("loc = %s, want the offending element in %s", xe.Loc, wantURI)
	}
}

// TestParseImportCrossNamespace is src-import's whole point (§4.2.6.2): the
// imported document's components enter the assembly IN THEIR OWN namespace, so a
// cross-namespace type= reference in the importing document resolves at finalize.
func TestParseImportCrossNamespace(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrapImporting("urn:a", "urn:b",
			`<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`+
				`<xs:element name="root" type="imp:code"/>`),
		"b.xsd": wrap("urn:b", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction>`+
			`</xs:simpleType>`+
			`<xs:element name="inB" type="tns:code"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:b", Local: "code"})
	if _, ok := s.Type(xsd.QName{Space: "urn:a", Local: "code"}); ok {
		t.Fatalf("type {urn:a}code exists: an <import> must not mint D2's components in the importer's namespace")
	}
	ed, ok := s.Element(xsd.QName{Space: "urn:a", Local: "root"})
	if !ok {
		t.Fatalf("element {urn:a}root not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != (xsd.QName{Space: "urn:b", Local: "code"}) {
		t.Fatalf("root type = %s, want {urn:b}code", got)
	}
	// D2's own components and its own intra-namespace reference are unaffected.
	if _, ok := s.Element(xsd.QName{Space: "urn:b", Local: "inB"}); !ok {
		t.Fatalf("element {urn:b}inB not found")
	}
}

// TestParseImportNoChameleonCoercion pins the difference between <import> and
// <include>: §F.1's coercion is src-include clause 2.3's alone. A bare <import>
// (no namespace attribute, src-import clause 1.2 satisfied by the importer's own
// targetNamespace) of a no-targetNamespace document leaves that document's
// components — and its unqualified QName references — in NO namespace, even
// though the importing document has one.
func TestParseImportNoChameleonCoercion(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:import schemaLocation="lib.xsd"/>`+
			`<xs:element name="root" type="xs:string"/>`),
		"lib.xsd": wrap("", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`+
			`<xs:element name="e" type="code"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Local: "code"})
	if _, ok := s.Type(xsd.QName{Space: "urn:a", Local: "code"}); ok {
		t.Fatalf("type {urn:a}code exists: <import> must never apply §F.1 chameleon coercion")
	}
	ed, ok := s.Element(xsd.QName{Local: "e"})
	if !ok {
		t.Fatalf("element {}e not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != (xsd.QName{Local: "code"}) {
		t.Fatalf("imported unqualified reference = %s, want the no-namespace {}code", got)
	}
}

// TestParseImportSelfNamespace covers src-import clause 1.1
// (src-import-noselfimport): the namespace attribute must not name the importing
// schema's own target namespace.
func TestParseImportSelfNamespace(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:import namespace="urn:a" schemaLocation="lib.xsd"/>`),
		"lib.xsd":  wrap("urn:a", `<xs:element name="e" type="xs:string"/>`),
	})
	mustXSDRule(t, err, "src-import-noselfimport", "main.xsd")
}

// TestParseImportSelfNamespaceUnderChameleon pins that clause 1.1 is judged
// against the EFFECTIVE (post-§F.1) target namespace: a chameleon-included
// document is produced as if it declared the includer's targetNamespace, so its
// <import> of that namespace is a self-import (§4.2.3's note on the coercion).
func TestParseImportSelfNamespaceUnderChameleon(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd":  wrap("urn:a", `<xs:include schemaLocation="cham.xsd"/>`),
		"cham.xsd":  wrap("", `<xs:import namespace="urn:a" schemaLocation="other.xsd"/>`),
		"other.xsd": wrap("urn:a", `<xs:element name="e" type="xs:string"/>`),
	})
	mustXSDRule(t, err, "src-import-noselfimport", "cham.xsd")
}

// TestParseImportBareFromNoNamespaceDocument covers src-import clause 1.2: with
// no namespace attribute the enclosing <schema> must have a targetNamespace,
// since a bare <import> from a no-namespace document would import the
// no-namespace it is already in.
func TestParseImportBareFromNoNamespaceDocument(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("", `<xs:import schemaLocation="lib.xsd"/>`),
		"lib.xsd":  wrap("", `<xs:element name="e" type="xs:string"/>`),
	})
	mustXSDRule(t, err, "src-import-noselfimport", "main.xsd")
}

// TestParseImportNamespaceMismatch covers src-import clause 3.1: the namespace
// attribute's value must be identical to the resolved document's own
// targetNamespace.
func TestParseImportNamespaceMismatch(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="c.xsd"/>`),
		"c.xsd":    wrap("urn:c", `<xs:element name="e" type="xs:string"/>`),
	})
	mustXSDRule(t, err, "src-import", "main.xsd")
}

// TestParseImportNamespaceAbsentButD2HasOne covers src-import clause 3.2: with
// no namespace attribute the resolved document must have no targetNamespace.
// Clause 3 branches on the PRESENCE of namespace, not on its value.
func TestParseImportNamespaceAbsentButD2HasOne(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:import schemaLocation="b.xsd"/>`),
		"b.xsd":    wrap("urn:b", `<xs:element name="e" type="xs:string"/>`),
	})
	mustXSDRule(t, err, "src-import", "main.xsd")
}

// TestParseImportWithoutSchemaLocation covers the bare <import> §4.2.6.2 calls
// out as legal: with no schemaLocation there is nothing for the reference
// strategy to succeed at, so clauses 2 and 3 do not apply and no document is
// read — in particular the importing document must NOT be re-read as its own D2.
func TestParseImportWithoutSchemaLocation(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b"/>`+
			`<xs:element name="root" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "root"}); !ok {
		t.Fatalf("element {urn:a}root not found")
	}
}

// TestParseImportUnresolvable covers §4.2.6.2's "It is not an error for the
// application schema component reference strategy to fail": a schemaLocation
// that does not resolve leaves clauses 2 and 3 vacuous, so the import is simply
// not performed.
func TestParseImportUnresolvable(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="missing.xsd"/>`+
			`<xs:element name="root" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v, want the unresolvable import to be skipped silently", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "root"}); !ok {
		t.Fatalf("element {urn:a}root not found")
	}
}

// TestParseImportNotWellFormed covers the other half of src-import clause 2: a
// location that DOES resolve must yield a <schema> in a well-formed information
// set, so a well-formedness fault there is a real violation — unlike a location
// that fails to resolve.
func TestParseImportNotWellFormed(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="broken.xsd"/>`),
		"broken.xsd": `<xs:schema xmlns:xs="` + xsdNS + `" targetNamespace="urn:b">` +
			`<xs:element name="e"/>`, // unclosed <xs:schema>
	})
	mustXSDRule(t, err, "src-import", "main.xsd")
}

// TestParseImportNotASchema covers src-import clause 2 as well: the resolved
// document element must be <schema>.
func TestParseImportNotASchema(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd":       wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="notaschema.xsd"/>`),
		"notaschema.xsd": `<html/>`,
	})
	mustXSDRule(t, err, "src-import", "main.xsd")
}

// TestParseImportIdempotent covers §4.2.6.2's note that the component-inclusion
// wording is "carefully worded so that multiple <import>ing of the same schema
// document will not constitute a violation of clause 2 [c-nmd] of Schema
// Properties Correct": the same document reached twice contributes its
// components ONCE, whether from one document, from two, or around a cycle.
func TestParseImportIdempotent(t *testing.T) {
	bImportingA := wrapImporting("urn:b", "urn:a",
		`<xs:import namespace="urn:a" schemaLocation="main.xsd"/>`+
			`<xs:element name="shared" type="xs:string"/>`)
	for _, tc := range []struct {
		name string
		docs map[string]string
	}{
		{
			name: "twice from one document",
			docs: map[string]string{
				"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`+
					`<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`),
				"b.xsd": wrap("urn:b", `<xs:element name="shared" type="xs:string"/>`),
			},
		},
		{
			name: "through a cycle",
			docs: map[string]string{
				"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`),
				"b.xsd":    bImportingA,
			},
		},
		{
			name: "diamond through an include",
			docs: map[string]string{
				"main.xsd": wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`+
					`<xs:include schemaLocation="lib.xsd"/>`),
				"lib.xsd": wrap("urn:a", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`),
				"b.xsd":   wrap("urn:b", `<xs:element name="shared" type="xs:string"/>`),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := parseMap(t, "main.xsd", tc.docs)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if _, ok := s.Element(xsd.QName{Space: "urn:b", Local: "shared"}); !ok {
				t.Fatalf("element {urn:b}shared not found")
			}
		})
	}
}

// TestParseSameDocumentIncludedAndImported is why the load-once key is a
// (resolved location, namespace) pair and not a location alone: ONE
// no-targetNamespace document reached both as a chameleon <include> (coerced into
// the includer's namespace, §F.1) and as a bare <import> (staying in no
// namespace) is two different component sets, and both are part of the assembly.
func TestParseSameDocumentIncludedAndImported(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:import schemaLocation="lib.xsd"/>`),
		"lib.xsd": wrap("", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
	mustType(t, s, xsd.QName{Local: "code"})
}

// TestParseDirectiveDocumentOrder pins that <include> and <import> are followed
// in ONE document-order pass, depth-first and pre-order, rather than in two
// kind-segregated passes. The order documents enter the assembly is the order
// their components enter the builder, so it is user-visible in sch-props-correct
// duplicate reports and must not depend on which KIND of directive named a
// document (STYLE D1/D2).
func TestParseDirectiveDocumentOrder(t *testing.T) {
	docs := map[string]string{
		// import, include, import — interleaved, so a kind-segregated pass would
		// reorder them.
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
	var requested []string
	inner := loader.Map(docs)
	recording := loader.ResolverFunc(func(namespace, location string) (io.ReadCloser, string, error) {
		requested = append(requested, location)
		return inner.Resolve(namespace, location)
	})
	s, err := parser.Parse("main.xsd", parser.WithResolver(recording))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := []string{"main.xsd", "b1.xsd", "lib.xsd", "b3.xsd", "b2.xsd"}
	if !slices.Equal(requested, want) {
		t.Fatalf("resolution order = %v, want %v (single document-order pass, depth-first)", requested, want)
	}
	for _, local := range []string{"e1", "e2", "e3"} {
		if _, ok := s.Element(xsd.QName{Space: "urn:b", Local: local}); !ok {
			t.Fatalf("element {urn:b}%s not found", local)
		}
	}
}
