package xmltree_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsderr"
)

// collect drains a reader over doc into a slice of nodes, stopping at io.EOF
// and returning the first non-EOF error.
func collect(t *testing.T, uri, doc string) ([]xmltree.Node, error) {
	t.Helper()
	return collectFrom(t, uri, strings.NewReader(doc))
}

// collectFrom is collect over an arbitrary source, for inputs a string cannot
// stand in for: encoded byte streams and readers that fail on demand.
func collectFrom(t *testing.T, uri string, src io.Reader) ([]xmltree.Node, error) {
	t.Helper()
	r := xmltree.NewReader(uri, src)
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

// wantWellFormednessError asserts err is a located XML well-formedness fault,
// the one verdict every encoding-layer failure must reach.
func wantWellFormednessError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("want an error, got nil")
	}
	rule, ok := xsderr.RuleOf(err)
	if !ok || rule != xsderr.RuleXMLWellFormed {
		t.Errorf("error %v: rule = %q (ok=%v), want %q", err, rule, ok, xsderr.RuleXMLWellFormed)
	}
	if _, ok := xsderr.LocOf(err); !ok {
		t.Errorf("error %v carries no xsderr.Loc", err)
	}
}

// utf16Doc encodes doc as UTF-16 in the given byte order behind the mark XML
// 1.0 §4.3.3 requires of a UTF-16 entity.
func utf16Doc(doc string, bigEndian bool) string {
	var b []byte
	for _, u := range utf16.Encode([]rune("\uFEFF" + doc)) {
		if bigEndian {
			b = append(b, byte(u>>8), byte(u))
			continue
		}
		b = append(b, byte(u), byte(u>>8))
	}
	return string(b)
}

// utf16Units encodes explicit big-endian code units behind a mark, for input
// no Go string can express: lone surrogates and half a code unit.
func utf16Units(units ...uint16) string {
	b := []byte{0xFE, 0xFF}
	for _, u := range units {
		b = append(b, byte(u>>8), byte(u))
	}
	return string(b)
}

// sameNodes asserts two token streams agree in kind, name, data, and location.
// A UTF-16 document decodes to exactly the bytes of its UTF-8 spelling, so
// even locations must match.
func sameNodes(t *testing.T, got, want []xmltree.Node) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d nodes, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Loc() != w.Loc() {
			t.Errorf("node %d loc = %v, want %v", i, got[i].Loc(), w.Loc())
		}
		switch w := w.(type) {
		case *xmltree.StartElement:
			g, ok := got[i].(*xmltree.StartElement)
			if !ok {
				t.Fatalf("node %d = %T, want *StartElement", i, got[i])
			}
			sameStart(t, i, g, w)
		case *xmltree.EndElement:
			g, ok := got[i].(*xmltree.EndElement)
			if !ok {
				t.Fatalf("node %d = %T, want *EndElement", i, got[i])
			}
			if g.Name() != w.Name() {
				t.Errorf("node %d end name = %v, want %v", i, g.Name(), w.Name())
			}
		case *xmltree.CharData:
			g, ok := got[i].(*xmltree.CharData)
			if !ok {
				t.Fatalf("node %d = %T, want *CharData", i, got[i])
			}
			if g.Data() != w.Data() || g.Offset() != w.Offset() {
				t.Errorf("node %d chardata = %q at offset %d, want %q at %d", i, g.Data(), g.Offset(), w.Data(), w.Offset())
			}
		}
	}
}

// sameStart compares one start tag's name and attributes, in order.
func sameStart(t *testing.T, i int, got, want *xmltree.StartElement) {
	t.Helper()
	if got.Name() != want.Name() {
		t.Errorf("node %d start name = %v, want %v", i, got.Name(), want.Name())
	}
	if len(got.Attributes()) != len(want.Attributes()) {
		t.Fatalf("node %d has %d attributes, want %d", i, len(got.Attributes()), len(want.Attributes()))
	}
	for j, a := range want.Attributes() {
		g := got.Attributes()[j]
		if g.Name() != a.Name() || g.Value() != a.Value() {
			t.Errorf("node %d attribute %d = %v=%q, want %v=%q", i, j, g.Name(), g.Value(), a.Name(), a.Value())
		}
	}
}

// byteOrders names the two UTF-16 serializations every decode test runs under.
var byteOrders = []struct {
	name   string
	bigEnd bool
}{
	{"big-endian", true},
	{"little-endian", false},
}

func TestUTF16BOMRoundTrips(t *testing.T) {
	docs := []struct{ name, doc string }{
		// An astral character exercises surrogate-pair decoding.
		{"namespaced multi-line", "<a xmlns=\"urn:D\" x=\"1\">\n  <b:c xmlns:b=\"urn:B\">clef \U0001D11E</b:c>\n</a>"},
		{"xml declaration without encoding", `<?xml version="1.0"?><a><b/></a>`},
	}
	for _, tc := range docs {
		for _, order := range byteOrders {
			t.Run(tc.name+"/"+order.name, func(t *testing.T) {
				want, err := collect(t, "t.xml", tc.doc)
				if err != nil {
					t.Fatalf("UTF-8 baseline: %v", err)
				}
				got, err := collectFrom(t, "t.xml", strings.NewReader(utf16Doc(tc.doc, order.bigEnd)))
				if err != nil {
					t.Fatalf("UTF-16 decode: %v", err)
				}
				sameNodes(t, got, want)
			})
		}
	}
}

// dribbleReader hands out one byte per Read, so every code unit and every
// surrogate pair straddles a fill boundary at some point.
type dribbleReader struct{ s string }

func (d *dribbleReader) Read(p []byte) (int, error) {
	if d.s == "" {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = d.s[0]
	d.s = d.s[1:]
	return 1, nil
}

func TestUTF16StreamsAcrossFillBoundaries(t *testing.T) {
	doc := "<a>" + strings.Repeat("x", 2000) + "\U0001D11E" + "</a>"
	want, err := collect(t, "t.xml", doc)
	if err != nil {
		t.Fatalf("UTF-8 baseline: %v", err)
	}
	sources := []struct {
		name string
		open func(string) io.Reader
	}{
		{"bulk reads", func(s string) io.Reader { return strings.NewReader(s) }},
		{"one byte per read", func(s string) io.Reader { return &dribbleReader{s: s} }},
	}
	for _, src := range sources {
		for _, order := range byteOrders {
			t.Run(src.name+"/"+order.name, func(t *testing.T) {
				got, err := collectFrom(t, "t.xml", src.open(utf16Doc(doc, order.bigEnd)))
				if err != nil {
					t.Fatalf("UTF-16 decode: %v", err)
				}
				sameNodes(t, got, want)
			})
		}
	}
}

func TestUTF16BOMAgreesWithDeclaration(t *testing.T) {
	cases := []struct {
		name     string
		declared string
		bigEnd   bool
	}{
		{"big-endian declares UTF-16", "UTF-16", true},
		{"little-endian declares UTF-16", "UTF-16", false},
		{"big-endian declares UTF-16BE", "UTF-16BE", true},
		{"little-endian declares lowercase utf-16le", "utf-16le", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := `<?xml version="1.0" encoding="` + tc.declared + `"?><a>x</a>`
			nodes, err := collectFrom(t, "t.xml", strings.NewReader(utf16Doc(doc, tc.bigEnd)))
			if err != nil {
				t.Fatalf("decode with encoding=%q: %v", tc.declared, err)
			}
			if len(nodes) != 3 {
				t.Fatalf("got %d nodes, want 3 (start, text, end)", len(nodes))
			}
		})
	}
}

// TestUTF8BOMIsStripped pins XML 1.0 §4.3.3's "encoding signature, not part of
// either the markup or the character data": the root still starts at column 1.
func TestUTF8BOMIsStripped(t *testing.T) {
	nodes, err := collectFrom(t, "t.xml", strings.NewReader("\xEF\xBB\xBF<a>x</a>"))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3", len(nodes))
	}
	wantLoc(t, nodes[0], 1, 1)
}

// TestUTF16IllFormedIsError pins the transcoder's central decision: ill-formed
// UTF-16 is a terminal error, never a U+FFFD substitution. Each case names the
// message it wants, so a substitution mutant cannot hide behind an unrelated
// structural failure — the unpaired surrogate is followed by ordinary
// character data rather than by '<' for exactly that reason.
func TestUTF16IllFormedIsError(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"unpaired surrogate", utf16Units('<', 'a', '>', 0xD834, 'x', '<', '/', 'a', '>'), "unpaired surrogate"},
		{"truncated code unit", utf16Units('<', 'a', '/', '>') + "\x00", "do not form a code unit"},
		{"mismatched end tag", utf16Units('<', 'a', '>', '<', 'b', '>', '<', '/', 'a', '>'), "does not match open element b"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := collectFrom(t, "t.xml", strings.NewReader(tc.src))
			wantWellFormednessError(t, err)
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to mention %q", err, tc.want)
			}
		})
	}
}

// TestBOMContradictsEncodingDeclaration pins XML 1.0 §4.3.3's fatal error for
// an entity presented in an encoding other than the one it declares.
func TestBOMContradictsEncodingDeclaration(t *testing.T) {
	decl := func(enc string) string {
		return `<?xml version="1.0" encoding="` + enc + `"?><a/>`
	}
	cases := []struct{ name, src string }{
		{"big-endian mark declares UTF-8", utf16Doc(decl("UTF-8"), true)},
		{"little-endian mark declares UTF-16BE", utf16Doc(decl("UTF-16BE"), false)},
		{"big-endian mark declares UTF-16LE", utf16Doc(decl("UTF-16LE"), true)},
		{"no mark declares UTF-16", decl("UTF-16")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := collectFrom(t, "t.xml", strings.NewReader(tc.src))
			wantWellFormednessError(t, err)
			if !strings.Contains(err.Error(), "disagrees") {
				t.Errorf("error = %q, want a disagreement message", err)
			}
		})
	}
}

// failFirstReader fails on its very first Read and reports end-of-input on
// every one after: the shape that exposes a dropped Peek error, since the
// retry answers with io.EOF and nothing carries the original cause any more.
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

// prefixThenFailReader yields prefix, then fails: enough bytes for a mark to
// be ruled out without the peek itself failing, so no error is latched.
type prefixThenFailReader struct {
	prefix string
	err    error
}

func (p *prefixThenFailReader) Read(b []byte) (int, error) {
	if p.prefix == "" {
		return 0, p.err
	}
	n := copy(b, p.prefix)
	p.prefix = p.prefix[n:]
	return n, nil
}

// TestSourceFailureSurfacesItsCause guards the byte-order-mark layer against
// swallowing a read failure. A source that fails before a mark can be read is
// the one bufio.Reader.Peek hands its error to exactly once, so a dropped
// error there turns a broken source into a bare io.EOF — which Token
// documents as the end of a well-formed document.
func TestSourceFailureSurfacesItsCause(t *testing.T) {
	cause := errors.New("boom: original cause")
	cases := []struct {
		name      string
		open      func() io.Reader
		wantNodes int
	}{
		{"fails before a mark can be read", func() io.Reader { return &failFirstReader{err: cause} }, 0},
		{"fails after a mark is ruled out", func() io.Reader { return &prefixThenFailReader{prefix: "<a>", err: cause} }, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := collectFrom(t, "t.xml", tc.open())
			if err == nil || errors.Is(err, io.EOF) {
				t.Fatalf("err = %v, want the source's failure, not end-of-document", err)
			}
			wantWellFormednessError(t, err)
			if !errors.Is(err, cause) {
				t.Errorf("error %q does not unwrap to the original cause", err)
			}
			if !strings.Contains(err.Error(), "boom") {
				t.Errorf("error = %q, want it to name the original cause", err)
			}
			if len(nodes) != tc.wantNodes {
				t.Errorf("got %d nodes, want %d", len(nodes), tc.wantNodes)
			}
		})
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
