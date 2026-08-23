package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// vcWrap wraps body in a <schema> with both the XSD and the versioning prefix
// bound, so a test body can write vc:minVersion without repeating the namespace.
// attrs goes on the <schema> element itself, which is how the root-stub case
// (§4.2.2's carve-out) is written.
func vcWrap(attrs, body string) string {
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` +
		` xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning" ` + attrs + `>` + body + `</xs:schema>`
}

// markedDeclaration writes one top-level <element name="e"> carrying marked,
// beside an unmarked sibling of another name so the document declares something
// whichever way the verdict goes.
func markedDeclaration(marked string) string {
	return vcWrap("", `<xs:element name="e" type="xs:string" `+marked+`/>`+
		`<xs:element name="unmarked" type="xs:int"/>`)
}

// declarationRetained produces doc and reports whether the declaration named e
// reached the schema — whether, that is, §4.2.2 retained it. The unmarked sibling
// is asserted too, so a fixture that produced nothing at all cannot read as an
// ignored element.
func declarationRetained(t *testing.T, doc string) bool {
	t.Helper()
	s, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "unmarked"}); !ok {
		t.Fatalf("the unmarked sibling declaration is missing, so nothing here reads as a §4.2.2 verdict")
	}
	_, ok := s.Element(xsd.QName{Local: "e"})
	return ok
}

// TestConditionalInclusionVersionBand pins §4.2.2's retained band,
// vc:minVersion ≤ V < vc:maxVersion with V = 1.1, one boundary at a time. The
// exclusive maxVersion end is the whole of the bug this mechanism was written
// for: an element marked vc:maxVersion="1.0" must go, or it collides with the
// sibling it was written to be replaced by.
func TestConditionalInclusionVersionBand(t *testing.T) {
	for _, tc := range []struct {
		name     string
		marked   string
		retained bool
	}{
		{"maxVersion below V ignores", `vc:minVersion="1.0" vc:maxVersion="1.0"`, false},
		{"maxVersion equal to V ignores, the end being exclusive", `vc:maxVersion="1.1"`, false},
		{"maxVersion above V retains", `vc:maxVersion="1.2"`, true},
		{"minVersion equal to V retains, the end being inclusive", `vc:minVersion="1.1"`, true},
		{"minVersion above V ignores", `vc:minVersion="1.2"`, false},
		{"minVersion below V retains", `vc:minVersion="1.0"`, true},
		{"a trailing zero is no different value", `vc:maxVersion="1.10"`, false},
		{"a leading zero is no different value", `vc:minVersion="01.2"`, false},
		{"a signed value compares numerically", `vc:minVersion="+2"`, false},
		{"a negative minVersion retains", `vc:minVersion="-1.5"`, true},
		{"a fraction-only lexical is a legal decimal", `vc:maxVersion=".9"`, false},
		{"an integral lexical is a legal decimal", `vc:minVersion="2"`, false},
		{"whitespace around the value is collapsed away", `vc:maxVersion="  1.0  "`, false},
		{"an unrecognized vc: attribute is not one of the six", `vc:minversion="9.9"`, true},
		{"an attribute of the same local name in no namespace is not one either", `minVersion="9.9"`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := declarationRetained(t, markedDeclaration(tc.marked)); got != tc.retained {
				t.Fatalf("declaration retained = %v, want %v", got, tc.retained)
			}
		})
	}
}

// TestConditionalInclusionAvailability pins §4.2.2's four availability clauses
// over the types and facets this processor knows, including the inversion its own
// note calls out: an empty vc:typeAvailable list ignores nothing, an empty
// vc:typeUnavailable list ignores the element.
func TestConditionalInclusionAvailability(t *testing.T) {
	for _, tc := range []struct {
		name     string
		marked   string
		retained bool
	}{
		{"typeAvailable naming a builtin retains", `vc:typeAvailable="xs:integer"`, true},
		{"typeAvailable naming xs:error retains", `vc:typeAvailable="xs:error"`, true},
		{"typeAvailable naming xs:anySimpleType retains", `vc:typeAvailable="xs:anySimpleType"`, true},
		{"typeAvailable naming xs:anyType retains", `vc:typeAvailable="xs:anyType"`, true},
		{"typeAvailable with one unknown item ignores", `vc:typeAvailable="xs:integer xs:bananaSkin"`, false},
		{"typeAvailable naming a type outside the XSD namespace ignores", `vc:typeAvailable="vc:myType"`, false},
		{"typeAvailable empty retains", `vc:typeAvailable=""`, true},
		{"typeUnavailable naming a builtin ignores", `vc:typeUnavailable="xs:integer"`, false},
		{"typeUnavailable with one unknown item retains", `vc:typeUnavailable="xs:integer vc:myType"`, true},
		{"typeUnavailable empty ignores", `vc:typeUnavailable=""`, false},
		{"facetAvailable naming a facet element retains", `vc:facetAvailable="xs:pattern"`, true},
		{"facetAvailable naming the assertion element retains", `vc:facetAvailable="xs:assertion"`, true},
		{"facetAvailable naming the enumeration element retains", `vc:facetAvailable="xs:enumeration"`, true},
		{"facetAvailable naming the facet's plural spelling ignores", `vc:facetAvailable="xs:assertions"`, false},
		{"facetAvailable with one unknown item ignores", `vc:facetAvailable="xs:pattern xs:bananaSkin"`, false},
		{"facetAvailable empty retains", `vc:facetAvailable=""`, true},
		{"facetUnavailable naming a facet element ignores", `vc:facetUnavailable="xs:minLength"`, false},
		{"facetUnavailable with one unknown item retains", `vc:facetUnavailable="xs:minLength xs:bananaSkin"`, true},
		{"facetUnavailable empty ignores", `vc:facetUnavailable=""`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := declarationRetained(t, markedDeclaration(tc.marked)); got != tc.retained {
				t.Fatalf("declaration retained = %v, want %v", got, tc.retained)
			}
		})
	}
}

// TestConditionalInclusionRetiresOneOfTwoSiblings is the shape the mechanism
// exists for and the shape the reported bug had: two declarations of ONE expanded
// name, the first marked for a version this processor is not. Exactly one reaches
// S2, so sch-props-correct clause 2 sees no duplicate — and the survivor is the
// unmarked one, which a fixture that merely parsed would not show.
func TestConditionalInclusionRetiresOneOfTwoSiblings(t *testing.T) {
	s, err := produce(t, vcWrap("",
		`<xs:element name="e" type="xs:string" vc:minVersion="1.0" vc:maxVersion="1.0"/>`+
			`<xs:element name="e" type="xs:int" vc:minVersion="1.0"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Local: "e"})
	if !ok {
		t.Fatalf("element e not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got.Local != "int" {
		t.Fatalf("surviving element type = %s, want xs:int — the declaration inside the retained band", got)
	}
}

// TestConditionalInclusionIgnoresWholeSubtree pins the scope of removal: "the
// element on which the attribute appears is to be ignored, along with all its
// attributes and descendants". A DESCENDANT of the ignored element is what makes
// the cascade observable — a named <xs:key> registers a schema-level
// {identity-constraint definitions} member (§3.17.1) from inside the declaration,
// so it is visible in the finished schema wherever the subtree survived.
func TestConditionalInclusionIgnoresWholeSubtree(t *testing.T) {
	s, err := produce(t, vcWrap("", `<xs:element name="gone" vc:maxVersion="1.0">`+
		`<xs:complexType><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:complexType>`+
		`<xs:key name="k"><xs:selector xpath="c"/><xs:field xpath="."/></xs:key>`+
		`</xs:element>`+
		`<xs:element name="kept" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "gone"}); ok {
		t.Fatalf("element gone was declared, but §4.2.2 removes it with its whole subtree")
	}
	for _, ic := range s.IdentityConstraints() {
		if ic.Name().Local == "k" {
			t.Fatalf("identity constraint k reached the schema from inside an ignored element's subtree")
		}
	}
	if _, ok := s.Element(xsd.QName{Local: "kept"}); !ok {
		t.Fatalf("element kept not found")
	}
}

// TestConditionalInclusionRootStub pins §4.2.2's carve-out for an ignored
// <schema> element: S2 keeps the root with an empty [children], so the document
// declares nothing and follows none of the directives it was written with.
func TestConditionalInclusionRootStub(t *testing.T) {
	s, err := produce(t, vcWrap(`targetNamespace="urn:a" vc:maxVersion="0.9"`,
		`<xs:include schemaLocation="nowhere.xsd"/>`+
			`<xs:element name="e" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "e"}); ok {
		t.Fatalf("element {urn:a}e was declared, but its <schema> is ignored and S2's [children] is empty")
	}
	if _, ok := s.Element(xsd.QName{Local: "e"}); ok {
		t.Fatalf("element e was declared in no namespace, but S2's [children] is empty")
	}
}

// TestConditionalInclusionInComposedDocument is the composed half of the
// mechanism, which no single-document test reaches: pre-processing runs on EVERY
// document of the assembly as it is read, before its own directives are followed
// (§4.2.2's "always performed first"). Four documents in one closure, each
// carrying a different half of it, all declaring the same expanded name.
func TestConditionalInclusionInComposedDocument(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		// The root declares e itself and includes documents that must contribute
		// nothing.
		"main.xsd": vcWrap(`targetNamespace="urn:a"`,
			`<xs:include schemaLocation="stub.xsd"/>`+
				`<xs:include schemaLocation="marked.xsd"/>`+
				`<xs:include schemaLocation="third.xsd" vc:maxVersion="0.9"/>`+
				`<xs:element name="e" type="xs:int"/>`),
		// Its own <schema> is ignored, so it neither declares e nor follows the
		// <include> that would bring a further declaration of it in.
		"stub.xsd": vcWrap(`targetNamespace="urn:a" vc:maxVersion="0.9"`,
			`<xs:include schemaLocation="third.xsd"/>`+
				`<xs:element name="e" type="xs:string"/>`),
		// A declaration marked for a version this processor is not.
		"marked.xsd": vcWrap(`targetNamespace="urn:a"`,
			`<xs:element name="e" type="xs:string" vc:minVersion="2.0"/>`),
		"third.xsd": vcWrap(`targetNamespace="urn:a"`,
			`<xs:element name="e" type="xs:boolean"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Space: "urn:a", Local: "e"})
	if !ok {
		t.Fatalf("element {urn:a}e not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got.Local != "int" {
		t.Fatalf("surviving element type = %s, want xs:int — the root's own declaration", got)
	}
}

// TestConditionalInclusionIgnoredRedefineIsNotFollowed pins the directive half of
// the composed case from the other side: a <redefine> the transform removes is
// not a <redefine> of S2 at all, so neither its own children nor the document it
// names contributes anything — and src-redefine clause 1, which would reject this
// unresolvable location outright, is never charged.
func TestConditionalInclusionIgnoredRedefineIsNotFollowed(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": vcWrap(`targetNamespace="urn:a"`,
			`<xs:redefine schemaLocation="nowhere.xsd" vc:maxVersion="1.1">`+
				`<xs:simpleType name="s"><xs:restriction base="xs:string"/></xs:simpleType>`+
				`</xs:redefine>`+
				`<xs:element name="e" type="xs:int"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := s.Type(xsd.QName{Space: "urn:a", Local: "s"}); ok {
		t.Fatalf("the redefining <simpleType> contributed a type, but its <redefine> is absent from S2")
	}
}

// TestConditionalInclusionSrcCIP pins src-cip: an ill-formed value is a
// schema-document error, never a silent prune or a silent keep. Each case would
// otherwise be a legal-looking marking that quietly decides which declaration
// survives.
func TestConditionalInclusionSrcCIP(t *testing.T) {
	for _, tc := range []struct {
		name   string
		marked string
		want   string
	}{
		{"minVersion outside xs:decimal", `vc:minVersion="1.1.3"`, "xs:decimal"},
		{"minVersion with a trailing letter", `vc:minVersion="10g"`, "xs:decimal"},
		{"maxVersion empty", `vc:maxVersion=""`, "xs:decimal"},
		{"maxVersion in exponent notation", `vc:maxVersion="1e1"`, "xs:decimal"},
		{"maxVersion a bare sign", `vc:maxVersion="-"`, "xs:decimal"},
		{"maxVersion a bare point", `vc:maxVersion="."`, "xs:decimal"},
		{"typeAvailable item is no NCName", `vc:typeAvailable="xs:integer 23"`, "xs:QName"},
		{"typeAvailable item has an empty local part", `vc:typeAvailable="xs:"`, "xs:QName"},
		{"typeUnavailable item has an unbound prefix", `vc:typeUnavailable="vx:t"`, "in-scope namespace"},
		{"facetAvailable item is no NCName", `vc:facetAvailable="-nope"`, "xs:QName"},
		{"facetUnavailable item has two colons", `vc:facetUnavailable="a:b:c"`, "xs:QName"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, markedDeclaration(tc.marked))
			var xerr *xsderr.Error
			if !errors.As(err, &xerr) {
				t.Fatalf("Produce error = %v, want an *xsderr.Error", err)
			}
			if xerr.Rule != "src-cip" {
				t.Fatalf("rule = %s, want src-cip", xerr.Rule)
			}
			if !strings.Contains(xerr.Error(), tc.want) {
				t.Fatalf("message %q does not mention %q", xerr.Error(), tc.want)
			}
		})
	}
}

// TestConditionalInclusionIgnoredSubtreeEscapesSrcCIP pins the ordering §4.2.1
// states — Schema Representation Constraints are "enforced after, not before, the
// ·conditional-inclusion pre-processing·" — over src-cip itself: an ill-formed
// vc: value inside a subtree the transform removes is not in S2 and is charged
// nothing.
func TestConditionalInclusionIgnoredSubtreeEscapesSrcCIP(t *testing.T) {
	if _, err := produce(t, vcWrap("", `<xs:element name="gone" vc:maxVersion="1.0">`+
		`<xs:complexType><xs:sequence vc:minVersion="not-a-decimal"/></xs:complexType>`+
		`</xs:element>`+
		`<xs:element name="kept" type="xs:string"/>`)); err != nil {
		t.Fatalf("Produce: %v, but the ill-formed value sits inside a subtree S2 does not contain", err)
	}
}

// TestConditionalInclusionLeavesAnUnmarkedDocumentAlone pins §4.2.2's own no-op
// note, "if S1 contains no elements or attributes to be ignored, then S1 and S2
// are identical", at the level a consumer can see: a document with no versioning
// attribute keeps every child, mixed content included.
func TestConditionalInclusionLeavesAnUnmarkedDocumentAlone(t *testing.T) {
	doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
		`<xs:annotation><xs:documentation>kept verbatim</xs:documentation></xs:annotation>` +
		`<xs:element name="e" type="xs:string"/></xs:schema>`
	s, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "e"}); !ok {
		t.Fatalf("element e not found")
	}
}
