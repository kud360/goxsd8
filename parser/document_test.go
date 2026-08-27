package parser_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// schemaDoc is a well-formed schema document carrying an xml:base on the root,
// an element that inherits it, an element that overrides it with a relative
// reference, and mixed content (documentation text inside an annotation).
const schemaDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t" xml:base="http://example.org/base/">` +
	`<xs:element name="root"/>` +
	`<xs:annotation><xs:documentation>hello text</xs:documentation></xs:annotation>` +
	`<xs:element name="child" xml:base="sub/"/>` +
	`</xs:schema>`

const docURI = "http://host/dir/main.xsd"

// childElement returns the first child Element of e whose local name is local,
// or nil. Text children are skipped.
func childElement(e *parser.Element, local string) *parser.Element {
	for _, n := range e.Children() {
		el, ok := n.(*parser.Element)
		if !ok {
			continue
		}
		if el.Name().Local() == local {
			return el
		}
	}
	return nil
}

func TestReadDocumentTree(t *testing.T) {
	d, err := parser.ReadDocument(docURI, strings.NewReader(schemaDoc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}

	if got := d.URI(); got != docURI {
		t.Errorf("URI() = %q, want %q", got, docURI)
	}

	root := d.Root()
	if root == nil {
		t.Fatal("Root() = nil")
	}
	if got := root.Name().Space(); got != xsd.XMLSchemaNS {
		t.Errorf("root name space = %q, want %q", got, xsd.XMLSchemaNS)
	}
	if got := root.Name().Local(); got != "schema" {
		t.Errorf("root local = %q, want %q", got, "schema")
	}
	if !d.IsSchema() {
		t.Error("IsSchema() = false, want true")
	}

	// Loc: the root's opening "<" is at line 1, column 1.
	if loc := root.Loc(); loc.Line != 1 || loc.Col != 1 || loc.URI != docURI {
		t.Errorf("root Loc() = %+v, want URI=%q line=1 col=1", loc, docURI)
	}
}

func TestReadDocumentMixedContentText(t *testing.T) {
	d, err := parser.ReadDocument(docURI, strings.NewReader(schemaDoc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	ann := childElement(d.Root(), "annotation")
	if ann == nil {
		t.Fatal("no <xs:annotation> found")
	}
	doc := childElement(ann, "documentation")
	if doc == nil {
		t.Fatal("no <xs:documentation> found")
	}
	// The documentation's sole child is a Text node round-tripping its content.
	kids := doc.Children()
	if len(kids) != 1 {
		t.Fatalf("documentation Children() len = %d, want 1", len(kids))
	}
	text, ok := kids[0].(*parser.Text)
	if !ok {
		t.Fatalf("documentation child = %T, want *parser.Text", kids[0])
	}
	if got := text.Data(); got != "hello text" {
		t.Errorf("text Data() = %q, want %q", got, "hello text")
	}
	if text.Loc().URI != docURI || text.Loc().Line != 1 {
		t.Errorf("text Loc() = %+v, want URI=%q line=1", text.Loc(), docURI)
	}
}

// TestReadDocumentPreservesWhitespaceText proves whitespace-only character data
// is retained as Text nodes (stripping is a later phase's decision).
func TestReadDocumentPreservesWhitespaceText(t *testing.T) {
	const doc = "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n  <xs:element name=\"a\"/>\n</xs:schema>"
	d, err := parser.ReadDocument(docURI, strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	var texts int
	for _, n := range d.Root().Children() {
		if _, ok := n.(*parser.Text); ok {
			texts++
		}
	}
	if texts == 0 {
		t.Error("whitespace text between elements was dropped; want it retained")
	}
}

// TestIsSchemaFalseNotError proves a non-schema root is NOT an error path: the
// document reads fine and IsSchema simply returns false.
func TestIsSchemaFalseNotError(t *testing.T) {
	const doc = `<root xmlns="urn:x"><a/></root>`
	d, err := parser.ReadDocument(docURI, strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument on non-schema root errored: %v", err)
	}
	if d.IsSchema() {
		t.Error("IsSchema() = true for <root>, want false")
	}
	if got := d.Root().Name().Local(); got != "root" {
		t.Errorf("root local = %q, want root", got)
	}
}

func TestReadDocumentMalformed(t *testing.T) {
	cases := map[string]string{
		"mismatched end tag": "<a></b>",
		"not xml":            "not xml at all",
		"unclosed element":   "<a><b>",
		"unbound prefix":     "<p:a/>",
		"empty document":     "",
		"only a comment":     "<!-- c -->",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			d, err := parser.ReadDocument(docURI, strings.NewReader(in))
			if err == nil {
				t.Fatalf("ReadDocument(%q) = nil error, want mapped xsderr", in)
			}
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("error %v is not an *xsderr.Error", err)
			}
			if xe.Loc.URI != docURI {
				t.Errorf("error Loc URI = %q, want %q", xe.Loc.URI, docURI)
			}
			if d != nil {
				t.Errorf("Document = %v on error, want nil", d)
			}
			// Rendering must not panic.
			_ = err.Error()
		})
	}
}

// failFirstReader fails on its very first Read and reports end-of-input on
// every one after — the source shape whose failure a dropped byte-order-mark
// peek error would replace with a clean end-of-document.
type failFirstReader struct {
	err  error
	done bool
}

func (f *failFirstReader) Read([]byte) (int, error) {
	if f.done {
		return 0, io.EOF
	}
	f.done = true
	return 0, f.err
}

// TestReadDocumentSourceFailureSurfacesItsCause is the end-to-end half of
// xmltree's TestSourceFailureSurfacesItsCause: an I/O failure must reach the
// caller as itself, not as the "document has no root element" diagnosis a
// swallowed error produces.
func TestReadDocumentSourceFailureSurfacesItsCause(t *testing.T) {
	cause := errors.New("boom: original cause")
	d, err := parser.ReadDocument(docURI, &failFirstReader{err: cause})
	if d != nil {
		t.Errorf("Document = %v on error, want nil", d)
	}
	if err == nil {
		t.Fatal("ReadDocument on a failing source = nil error")
	}
	if !errors.Is(err, cause) {
		t.Errorf("error %q does not unwrap to the original cause", err)
	}
	if strings.Contains(err.Error(), "no root element") {
		t.Errorf("error = %q, want the I/O cause, not a well-formedness diagnosis of it", err)
	}
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("error %v is not an *xsderr.Error", err)
	}
	if xe.Rule != xsderr.RuleXMLWellFormed || xe.Loc.URI != docURI {
		t.Errorf("error rule/loc = %q/%v, want %q at %q", xe.Rule, xe.Loc, xsderr.RuleXMLWellFormed, docURI)
	}
}
