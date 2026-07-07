package xmltree

import (
	"encoding/xml"
	"errors"
	"io"
	"sort"

	"github.com/kud360/goxsd8/xsderr"
)

// Reader is a streaming, namespace-scoped XML token reader: the origin of
// every xsderr.Loc a schema or instance document produces. It wraps an
// io.Reader and yields resolved Nodes one at a time from Token, holding only
// bounded state — a namespace-scope chain, an open-element stack, and an
// index of newline offsets — never the whole document (STYLE P4).
//
// A Reader is single-use and not safe for concurrent use.
type Reader struct {
	uri string
	dec *xml.Decoder
	pos *posReader

	// stack holds one frame per currently-open element, so end tags match
	// their starts and nested elements resolve against the right scope.
	stack []frame
	// eof latches io.EOF so repeated Token calls keep returning it.
	eof bool
}

// frame is one open element: its resolved name (to match the end tag), the
// scope in force for its content, and the location of its start tag (so an
// unclosed-element error points at the tag left open, not at end-of-stream).
type frame struct {
	name  Name
	scope *scope
	loc   xsderr.Loc
}

// NewReader returns a Reader over r. uri names the document for locations
// (xsderr.Loc.URI); it is not opened or resolved here — it is only threaded
// into every Loc the reader emits.
func NewReader(uri string, r io.Reader) *Reader {
	pos := &posReader{r: r}
	return &Reader{
		uri: uri,
		dec: xml.NewDecoder(pos),
		pos: pos,
	}
}

// Token advances to the next element or character-data node and returns it.
// It returns io.EOF at the end of a well-formed document. Comments,
// processing instructions, and directives are skipped. Malformed input,
// unbound namespace prefixes, and mismatched or unclosed tags are returned
// as errors carrying an xsderr.Loc — never as a panic (see the fuzz target).
func (r *Reader) Token() (Node, error) {
	if r.eof {
		return nil, io.EOF
	}
	for {
		// InputOffset before RawToken is the offset of the token's first
		// byte; RawToken then advances the decoder past it.
		off := r.dec.InputOffset()
		tok, err := r.dec.RawToken()
		if err != nil {
			return r.handleReadErr(err)
		}
		node, emit, err := r.classify(tok, off)
		if err != nil {
			return nil, err
		}
		if emit {
			return node, nil
		}
	}
}

// handleReadErr maps a RawToken error to the reader's contract: io.EOF ends a
// well-formed document (unclosed elements are an error instead); any other
// error is a malformed-XML failure wrapped with the current location.
func (r *Reader) handleReadErr(err error) (Node, error) {
	if errors.Is(err, io.EOF) {
		r.eof = true
		if len(r.stack) > 0 {
			open := r.stack[len(r.stack)-1]
			return nil, xsderr.New(xsderr.RuleXMLWellFormed, open.loc, "unexpected end of document: element %s left unclosed", qname(open.name))
		}
		return nil, io.EOF
	}
	return nil, xsderr.Wrap(xsderr.RuleXMLWellFormed, r.locAt(r.dec.InputOffset()), err)
}

// classify resolves one raw token. It returns (node, true, nil) to emit a
// node, (nil, false, nil) to skip (comment/PI/directive/duplicate
// whitespace), or an error.
func (r *Reader) classify(tok xml.Token, off int64) (Node, bool, error) {
	loc := r.locAt(off)
	switch t := tok.(type) {
	case xml.StartElement:
		node, err := r.startElement(t, loc)
		if err != nil {
			return nil, false, err
		}
		return node, true, nil
	case xml.EndElement:
		node, err := r.endElement(t, loc)
		if err != nil {
			return nil, false, err
		}
		return node, true, nil
	case xml.CharData:
		return &CharData{data: string(t), offset: off, loc: loc}, true, nil
	default:
		// xml.Comment, xml.ProcInst, xml.Directive: not part of the
		// element/character-data stream the parser consumes.
		return nil, false, nil
	}
}

// startElement resolves an element's name and attributes against the scope
// its declarations establish, pushes an open-element frame, and returns the
// node.
func (r *Reader) startElement(t xml.StartElement, loc xsderr.Loc) (*StartElement, error) {
	parent := r.currentScope()
	child := parent.child(bindingsOf(t.Attr))

	if t.Name.Space == xmlnsPrefix {
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "%q is a reserved prefix and cannot name an element", xmlnsPrefix)
	}
	space, ok := child.lookup(t.Name.Space)
	if !ok {
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "unbound namespace prefix %q on element <%s>", t.Name.Space, rawName(t.Name))
	}
	name := Name{space: space, local: t.Name.Local}

	attrs, err := resolveAttrs(t.Attr, child, loc)
	if err != nil {
		return nil, err
	}

	r.stack = append(r.stack, frame{name: name, scope: child, loc: loc})
	return &StartElement{name: name, attrs: attrs, scope: child, loc: loc}, nil
}

// endElement matches an end tag against the open element and pops it.
func (r *Reader) endElement(t xml.EndElement, loc xsderr.Loc) (*EndElement, error) {
	if len(r.stack) == 0 {
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "unexpected end tag </%s> with no open element", rawName(t.Name))
	}
	top := r.stack[len(r.stack)-1]
	space, ok := top.scope.lookup(t.Name.Space)
	if !ok {
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "unbound namespace prefix %q on end tag </%s>", t.Name.Space, rawName(t.Name))
	}
	got := Name{space: space, local: t.Name.Local}
	if got != top.name {
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "end tag </%s> does not match open element %s", rawName(t.Name), qname(top.name))
	}
	r.stack = r.stack[:len(r.stack)-1]
	return &EndElement{name: got, loc: loc}, nil
}

// currentScope is the scope in force for the innermost open element, or nil
// (the empty base scope) at the document level.
func (r *Reader) currentScope() *scope {
	if len(r.stack) == 0 {
		return nil
	}
	return r.stack[len(r.stack)-1].scope
}

// bindingsOf extracts the namespace declarations from an element's raw
// attributes, in document order. RawToken reports xmlns:p as {Space:"xmlns",
// Local:"p"} and the default xmlns as {Space:"", Local:"xmlns"}.
func bindingsOf(attrs []xml.Attr) []binding {
	var bs []binding
	for _, a := range attrs {
		if a.Name.Space == xmlnsPrefix {
			bs = append(bs, binding{prefix: a.Name.Local, uri: a.Value})
			continue
		}
		if a.Name.Space == "" && a.Name.Local == xmlnsPrefix {
			bs = append(bs, binding{prefix: "", uri: a.Value})
		}
	}
	return bs
}

// resolveAttrs resolves the non-declaration attributes of an element against
// scope s, in document order (STYLE D2: order is preserved, no map). An
// unprefixed attribute is in no namespace; a prefixed one whose prefix is
// unbound is an error.
func resolveAttrs(attrs []xml.Attr, s *scope, loc xsderr.Loc) ([]Attribute, error) {
	var out []Attribute
	for _, a := range attrs {
		if isDeclaration(a) {
			continue
		}
		if a.Name.Space == xmlnsPrefix {
			return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "%q is a reserved prefix and cannot name an attribute", xmlnsPrefix)
		}
		space := ""
		if a.Name.Space != "" {
			resolved, ok := s.lookup(a.Name.Space)
			if !ok {
				return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "unbound namespace prefix %q on attribute %s", a.Name.Space, rawName(a.Name))
			}
			space = resolved
		}
		out = append(out, Attribute{name: Name{space: space, local: a.Name.Local}, value: a.Value, loc: loc})
	}
	return out, nil
}

// isDeclaration reports whether a raw attribute is a namespace declaration
// (xmlns or xmlns:p) rather than a real attribute.
func isDeclaration(a xml.Attr) bool {
	if a.Name.Space == xmlnsPrefix {
		return true
	}
	return a.Name.Space == "" && a.Name.Local == xmlnsPrefix
}

// rawName renders a RawToken name (Space holds the raw prefix) as it appeared
// in the source, for error messages.
func rawName(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

// qname renders a resolved Name for error messages as "{uri}local", or just
// local for a name in no namespace.
func qname(n Name) string {
	if n.space == "" {
		return n.local
	}
	return "{" + n.space + "}" + n.local
}

// locAt maps a byte offset to a 1-based (line, column) using the newline
// index, sort-searched on demand (STYLE P4: no retained document content).
// Column counts bytes from the start of the line.
func (r *Reader) locAt(off int64) xsderr.Loc {
	if off < 0 {
		off = 0
	}
	nls := r.pos.newlines
	before := sort.Search(len(nls), func(i int) bool { return nls[i] >= off })
	lastNL := int64(-1)
	if before > 0 {
		lastNL = nls[before-1]
	}
	return xsderr.Loc{URI: r.uri, Line: before + 1, Col: int(off - lastNL)}
}

// posReader wraps the input, counting bytes and recording the offset of every
// newline so line/column can be derived without keeping the content. It grows
// with the number of lines, not the document size.
type posReader struct {
	r        io.Reader
	off      int64
	newlines []int64
}

// Read reads from the underlying reader, recording newline offsets as bytes
// pass through.
func (p *posReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	for i := 0; i < n; i++ {
		if b[i] == '\n' {
			p.newlines = append(p.newlines, p.off+int64(i))
		}
	}
	p.off += int64(n)
	return n, err
}
