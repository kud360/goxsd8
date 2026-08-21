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

// This test is package-internal because Element.parent is an unexported field
// with no accessor (STYLE T5: nothing outside package parser navigates upward).
// What it pins is what ReadDocument builds: the root's parent is nil and its
// child's parent is the root, the structural edge a later src-resolve phase
// (§3.17.6.2) walks up to reach a <schema> ancestor.
func TestReadDocumentLinksEachChildToItsParent(t *testing.T) {
	const doc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:t">` +
		`<xs:element name="root"/>` +
		`</xs:schema>`

	d, err := ReadDocument("http://host/dir/main.xsd", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	root := d.Root()
	if root == nil {
		t.Fatal("Root() = nil")
	}
	if root.parent != nil {
		t.Errorf("root parent = %v, want nil", root.parent)
	}

	var child *Element
	for _, n := range root.Children() {
		el, ok := n.(*Element)
		if !ok {
			continue
		}
		child = el
		break
	}
	if child == nil {
		t.Fatal("no <xs:element> child found")
	}
	if child.parent != root {
		t.Errorf("child parent = %v, want the schema root %v", child.parent, root)
	}
}
