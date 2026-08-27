package parser

import (
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/kud360/goxsd8/xsd"
)

// rawTreeDoc is this file's own fixture: a well-formed schema document carrying
// an xml:base on the root, an element that inherits it, an element that
// overrides it with a relative reference, and mixed content (documentation text
// inside an annotation). The black-box half declares its own; a fixture is not
// shared across the package boundary.
const rawTreeDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xml:base="http://example.org/base/">` +
	`<xs:element name="root"/>` +
	`<xs:annotation><xs:documentation>hello text</xs:documentation></xs:annotation>` +
	`<xs:element name="child" xml:base="sub/"/>` +
	`</xs:schema>`

const rawTreeDocURI = "http://host/dir/main.xsd"

// This test is package-internal because Element.attrs and Element.baseURI are
// unexported fields with no accessor (STYLE T5: nothing outside package parser
// reads a raw element's attribute list or its composed base URI). What it pins
// is what ReadDocument stores on the root: an attribute list that excludes
// namespace declarations and keeps the rest in document order, and a base URI
// that is the root's own xml:base resolved against the document URI.
func TestReadDocumentRootAttributesAndBaseURI(t *testing.T) {
	d, err := ReadDocument(rawTreeDocURI, strings.NewReader(rawTreeDoc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	root := d.Root()
	if root == nil {
		t.Fatal("Root() = nil")
	}

	// Attributes exclude the xmlns:xs declaration but keep targetNamespace and
	// xml:base, in document order.
	locals := attrLocals(root)
	if !slices.Contains(locals, "targetNamespace") {
		t.Errorf("attributes %v missing targetNamespace", locals)
	}
	if !slices.Contains(locals, "base") {
		t.Errorf("attributes %v missing xml:base", locals)
	}
	if slices.Contains(locals, "xs") {
		t.Errorf("attributes %v leaked the xmlns:xs declaration", locals)
	}

	if got := root.baseURI; got != "http://example.org/base/" {
		t.Errorf("root base URI = %q, want %q", got, "http://example.org/base/")
	}
}

// This test is package-internal because Element.baseURI is an unexported field
// with no accessor (STYLE T5). What it pins is the composition ReadDocument
// performs top-down as it builds the tree: an element declaring no xml:base
// inherits its parent's base unchanged, and one declaring a relative reference
// resolves it against that parent base.
func TestReadDocumentBaseURIInheritAndOverride(t *testing.T) {
	d, err := ReadDocument(rawTreeDocURI, strings.NewReader(rawTreeDoc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	root := d.Root()

	// The "root" element declares no xml:base: it inherits the parent's base
	// unchanged.
	rootEl := childElement(root, xsd.XMLSchemaNS, "element")
	if rootEl == nil {
		t.Fatal("no <xs:element> child found")
	}
	if got := rootEl.baseURI; got != "http://example.org/base/" {
		t.Errorf("inherited base URI = %q, want %q", got, "http://example.org/base/")
	}

	// The "child" element overrides with a relative reference, resolved against
	// its parent's base.
	var child *Element
	for _, n := range root.Children() {
		el, ok := n.(*Element)
		if ok && el.Name().Local() == "element" {
			if v, _ := el.Attr("name"); v == "child" {
				child = el
			}
		}
	}
	if child == nil {
		t.Fatal("no child <xs:element name=\"child\"> found")
	}
	if got := child.baseURI; got != "http://example.org/base/sub/" {
		t.Errorf("overridden base URI = %q, want %q", got, "http://example.org/base/sub/")
	}
}

// utf16LE encodes doc as little-endian UTF-16 behind the byte-order mark that
// XML 1.0 §4.3.3 requires of a UTF-16 entity.
func utf16LE(doc string) string {
	var b []byte
	for _, u := range utf16.Encode([]rune("\uFEFF" + doc)) {
		b = append(b, byte(u), byte(u>>8))
	}
	return string(b)
}

// This test is package-internal because sameTree compares Element.baseURI and
// Element.attrs, unexported fields with no accessor (STYLE T5). Those two
// comparisons are the load-bearing half here: attribute values and a composed
// base URI are decoded character data, exactly what a UTF-16 decode bug
// corrupts, so a thinner comparison would pass on a broken decoder.
func TestReadDocumentUTF16BOM(t *testing.T) {
	d, err := ReadDocument(rawTreeDocURI, strings.NewReader(utf16LE(rawTreeDoc)))
	if err != nil {
		t.Fatalf("ReadDocument on UTF-16 input: %v", err)
	}
	if !d.IsSchema() {
		t.Error("IsSchema() = false, want true")
	}
	want, err := ReadDocument(rawTreeDocURI, strings.NewReader(rawTreeDoc))
	if err != nil {
		t.Fatalf("UTF-8 baseline: %v", err)
	}
	sameTree(t, "/", d.Root(), want.Root())
}

// sameTree asserts that two element subtrees agree in name, location, base URI,
// attributes, and children. A UTF-16 document decodes to exactly the bytes of
// its UTF-8 spelling, so even locations must match.
func sameTree(t *testing.T, path string, got, want *Element) {
	t.Helper()
	if got.Name() != want.Name() {
		t.Fatalf("%s: name = %v, want %v", path, got.Name(), want.Name())
	}
	if got.Loc() != want.Loc() {
		t.Errorf("%s: loc = %v, want %v", path, got.Loc(), want.Loc())
	}
	if got.baseURI != want.baseURI {
		t.Errorf("%s: base URI = %q, want %q", path, got.baseURI, want.baseURI)
	}
	if len(got.attrs) != len(want.attrs) {
		t.Fatalf("%s: %d attributes, want %d", path, len(got.attrs), len(want.attrs))
	}
	for i, a := range want.attrs {
		g := got.attrs[i]
		if g.Name() != a.Name() || g.Value() != a.Value() {
			t.Errorf("%s: attribute %d = %v=%q, want %v=%q", path, i, g.Name(), g.Value(), a.Name(), a.Value())
		}
	}
	if len(got.Children()) != len(want.Children()) {
		t.Fatalf("%s: %d children, want %d", path, len(got.Children()), len(want.Children()))
	}
	for i, w := range want.Children() {
		child := path + want.Name().Local() + "/"
		switch w := w.(type) {
		case *Element:
			g, ok := got.Children()[i].(*Element)
			if !ok {
				t.Fatalf("%s: child %d = %T, want *Element", child, i, got.Children()[i])
			}
			sameTree(t, child, g, w)
		case *Text:
			g, ok := got.Children()[i].(*Text)
			if !ok {
				t.Fatalf("%s: child %d = %T, want *Text", child, i, got.Children()[i])
			}
			if g.Data() != w.Data() || g.Loc() != w.Loc() {
				t.Errorf("%s: child %d text = %q at %v, want %q at %v", child, i, g.Data(), g.Loc(), w.Data(), w.Loc())
			}
		}
	}
}

// attrLocals returns the local names of an element's attributes in order.
func attrLocals(e *Element) []string {
	var out []string
	for _, a := range e.attrs {
		out = append(out, a.Name().Local())
	}
	return out
}
