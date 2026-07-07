package xmltree_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsderr"
)

// collect drains a reader into a slice of nodes, stopping at io.EOF and
// returning the first non-EOF error.
func collect(t *testing.T, uri, doc string) ([]xmltree.Node, error) {
	t.Helper()
	r := xmltree.NewReader(uri, strings.NewReader(doc))
	var nodes []xmltree.Node
	for {
		n, err := r.Token()
		if errors.Is(err, io.EOF) {
			return nodes, nil
		}
		if err != nil {
			return nodes, err
		}
		nodes = append(nodes, n)
	}
}

func wantLoc(t *testing.T, n xmltree.Node, line, col int) {
	t.Helper()
	loc := n.Loc()
	if loc.Line != line || loc.Col != col {
		t.Errorf("loc = %d:%d, want %d:%d (node %T)", loc.Line, loc.Col, line, col, n)
	}
}

func TestPositionsMultiLine(t *testing.T) {
	nodes, err := collect(t, "t.xml", "<a>\n  <b>x</b>\n</a>")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	// START a, CharData "\n  ", START b, CharData "x", END b, CharData "\n", END a
	if len(nodes) != 7 {
		t.Fatalf("got %d nodes, want 7", len(nodes))
	}
	if nodes[0].Loc().URI != "t.xml" {
		t.Errorf("URI = %q, want t.xml", nodes[0].Loc().URI)
	}
	wantLoc(t, nodes[0], 1, 1) // <a>
	wantLoc(t, nodes[1], 1, 4) // "\n  " starts right after '>'
	wantLoc(t, nodes[2], 2, 3) // <b> after two spaces
	wantLoc(t, nodes[3], 2, 6) // 'x'
	wantLoc(t, nodes[4], 2, 7) // </b>
	wantLoc(t, nodes[6], 3, 1) // </a>

	cd, ok := nodes[3].(*xmltree.CharData)
	if !ok {
		t.Fatalf("nodes[3] = %T, want *CharData", nodes[3])
	}
	if cd.Data() != "x" {
		t.Errorf("chardata = %q, want x", cd.Data())
	}
	if cd.Offset() != 9 {
		t.Errorf("offset = %d, want 9", cd.Offset())
	}
}

func TestPositionsCRLF(t *testing.T) {
	// Columns count raw bytes, so the CR contributes a column even though
	// encoding/xml normalizes CRLF to LF in the character data value.
	nodes, err := collect(t, "t.xml", "<a>\r\n<b/>\r\n</a>")
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	wantLoc(t, nodes[0], 1, 1) // <a>
	wantLoc(t, nodes[1], 1, 4) // CharData at the CR, still line 1
	wantLoc(t, nodes[2], 2, 1) // <b/> on line 2
}

func TestNamespaceDefaultAndShadowing(t *testing.T) {
	// Outer default urn:D, inner element rebinds default to urn:E for <c>.
	nodes, err := collect(t, "t.xml", `<a xmlns="urn:D"><b><c xmlns="urn:E"/></b></a>`)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	a := nodes[0].(*xmltree.StartElement)
	if a.Name().Space() != "urn:D" || a.Name().Local() != "a" {
		t.Errorf("a name = %v", a.Name())
	}
	c := nodes[2].(*xmltree.StartElement)
	if c.Name().Space() != "urn:E" {
		t.Errorf("c space = %q, want urn:E (default shadowed)", c.Name().Space())
	}
	// After </c></b>, the closing </a> must still resolve under urn:D.
	last := nodes[len(nodes)-1].(*xmltree.EndElement)
	if last.Name().Space() != "urn:D" || last.Name().Local() != "a" {
		t.Errorf("closing a = %v, want {urn:D}a", last.Name())
	}
}

func TestPrefixShadowing(t *testing.T) {
	// p bound to urn:P at <a>, rebound to urn:Q at <p:b>; <p:c> sees urn:Q.
	nodes, err := collect(t, "t.xml", `<a xmlns:p="urn:P"><p:b xmlns:p="urn:Q"><p:c/></p:b></a>`)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	b := nodes[1].(*xmltree.StartElement)
	if b.Name().Space() != "urn:Q" {
		t.Errorf("b space = %q, want urn:Q", b.Name().Space())
	}
	// LookupPrefix on the inner element yields the shadowing binding.
	if uri, ok := b.LookupPrefix("p"); !ok || uri != "urn:Q" {
		t.Errorf("LookupPrefix(p) on b = (%q,%v), want (urn:Q,true)", uri, ok)
	}
	c := nodes[2].(*xmltree.StartElement)
	if c.Name().Space() != "urn:Q" {
		t.Errorf("c space = %q, want urn:Q", c.Name().Space())
	}
}

func TestAttributesNoDefaultNamespace(t *testing.T) {
	// Default namespace applies to element names, never to attribute names.
	nodes, err := collect(t, "t.xml", `<a xmlns="urn:D" x="1"/>`)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	a := nodes[0].(*xmltree.StartElement)
	attrs := a.Attributes()
	if len(attrs) != 1 {
		t.Fatalf("got %d attrs, want 1 (xmlns is not an attribute)", len(attrs))
	}
	if attrs[0].Name().Space() != "" || attrs[0].Name().Local() != "x" {
		t.Errorf("attr name = %v, want unqualified x", attrs[0].Name())
	}
	if attrs[0].Value() != "1" {
		t.Errorf("attr value = %q, want 1", attrs[0].Value())
	}
}

func TestPrefixedAttributeResolves(t *testing.T) {
	nodes, err := collect(t, "t.xml", `<a xmlns:p="urn:P" p:x="1"/>`)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	a := nodes[0].(*xmltree.StartElement)
	if got := a.Attributes()[0].Name(); got.Space() != "urn:P" || got.Local() != "x" {
		t.Errorf("attr name = %v, want {urn:P}x", got)
	}
}

func TestXMLPrefixImplicit(t *testing.T) {
	// The xml: prefix resolves with no xmlns:xml declaration.
	nodes, err := collect(t, "t.xml", `<r><x xml:lang="en"/></r>`)
	if err != nil {
		t.Fatalf("collect: %v", err)
	}
	x := nodes[1].(*xmltree.StartElement)
	a := x.Attributes()[0]
	if a.Name().Space() != "http://www.w3.org/XML/1998/namespace" || a.Name().Local() != "lang" {
		t.Errorf("attr name = %v, want the XML namespace lang", a.Name())
	}
	if uri, ok := x.LookupPrefix("xml"); !ok || uri != "http://www.w3.org/XML/1998/namespace" {
		t.Errorf("LookupPrefix(xml) = (%q,%v)", uri, ok)
	}
}

func TestUnboundPrefixIsError(t *testing.T) {
	cases := map[string]string{
		"element":   `<a><u:b/></a>`,
		"attribute": `<a u:x="1"/>`,
		"end tag":   `<a><b></p:b></a>`,
	}
	// Deterministic iteration is irrelevant here (independent subtests).
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := collect(t, "t.xml", doc)
			if err == nil {
				t.Fatalf("want error for %q", doc)
			}
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no xsderr.Loc", err)
			}
			if loc.Line == 0 || loc.URI != "t.xml" {
				t.Errorf("error loc = %v, want a located t.xml error", loc)
			}
			if !strings.Contains(err.Error(), "unbound namespace prefix") {
				t.Errorf("error = %q, want an unbound-prefix message", err)
			}
		})
	}
}

func TestMismatchedEndTagIsError(t *testing.T) {
	_, err := collect(t, "t.xml", `<a></b>`)
	if err == nil {
		t.Fatal("want error for mismatched end tag")
	}
	if _, ok := xsderr.LocOf(err); !ok {
		t.Fatalf("error %v carries no location", err)
	}
}

func TestUnclosedElementIsError(t *testing.T) {
	_, err := collect(t, "t.xml", `<a><b></b>`)
	if err == nil {
		t.Fatal("want error for unclosed root element")
	}
	if !strings.Contains(err.Error(), "unclosed") {
		t.Errorf("error = %q, want an unclosed-element message", err)
	}
}

func TestMalformedXMLIsErrorNotPanic(t *testing.T) {
	_, err := collect(t, "t.xml", "<a>\n<b>\x00</b></a>")
	if err == nil {
		t.Fatal("want error for control character in content")
	}
	loc, ok := xsderr.LocOf(err)
	if !ok || loc.URI != "t.xml" {
		t.Errorf("malformed-XML error missing located wrap: %v", err)
	}
}

func TestEOFIsIdempotent(t *testing.T) {
	r := xmltree.NewReader("t.xml", strings.NewReader("<a/>"))
	for {
		_, err := r.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if _, err := r.Token(); !errors.Is(err, io.EOF) {
		t.Errorf("second post-EOF Token = %v, want io.EOF", err)
	}
}
