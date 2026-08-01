package parser

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// This test is package-internal because Element.lookupPrefix is unexported
// (STYLE T5: it has no consumer outside package parser). What it pins is the
// wiring, not the algorithm: that ReadDocument hands each Element the
// xmltree.StartElement whose scope the delegator reads, so a QName-valued
// lexical found in the tree resolves against the bindings actually in scope
// where it occurred (Datatypes §3.3.18). The scope-resolution algorithm itself
// is covered in parser/xmltree/scope_test.go.
func TestElementLookupPrefixDelegatesToStartTag(t *testing.T) {
	const doc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t"/>`

	d, err := ReadDocument("http://host/dir/main.xsd", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	root := d.Root()
	if root == nil {
		t.Fatal("Root() = nil")
	}

	uri, ok := root.lookupPrefix("xs")
	if !ok || uri != xsd.XMLSchemaNS {
		t.Errorf("lookupPrefix(xs) = %q,%v; want %q,true", uri, ok, xsd.XMLSchemaNS)
	}
}
