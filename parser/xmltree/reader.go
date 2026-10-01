package xmltree

import (
	"encoding/xml"
	"errors"
	"io"
	"sort"

	"github.com/kud360/goxsd8/internal/xmldecl"
	"github.com/kud360/goxsd8/internal/xmlenc"
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

	// bom is what the document's byte-order mark said its encoding is — the
	// evidence an encoding declaration must agree with (XML 1.0 §4.3.3).
	bom xmlenc.Mark

	// stack holds one frame per currently-open element, so end tags match
	// their starts and nested elements resolve against the right scope.
	stack []frame
	// eof latches io.EOF so repeated Token calls keep returning it.
	eof bool

	// entities maps each general entity name the DOCTYPE's internal subset
	// declares to whether that entity is unparsed. It holds the parsed ones
	// too because the FIRST declaration of a name binds (XML 1.0 §4.2), so a
	// later NDATA declaration of a name already declared parsed declares no
	// unparsed entity. It is a lookup index only, never iterated.
	entities map[string]bool
	// declsUnread is the inverse of [all declarations processed]: the DOCTYPE
	// named an external subset, or its internal subset referenced a parameter
	// entity that was not read (see doctypeEntities). Inverted so that the
	// zero value — no DOCTYPE at all — is the right answer.
	declsUnread bool
	// standalone records the XML declaration's standalone="yes" (XML 1.0
	// §2.9), which the DOCTYPE after it is read under.
	standalone bool
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
//
// A document whose XML declaration specifies a 1.x version number other than
// 1.0 is read as a 1.0 document (XML 1.0 §2.8 Note), through
// xmldecl.As10's same-length rewrite of that number, so locations are
// unchanged by it.
//
// A leading byte-order mark is honoured per XML 1.0 §4.3.3: a UTF-16 document
// is decoded to UTF-8 before the XML decoder sees it, and a UTF-8 mark is
// dropped as the encoding signature it is (internal/xmlenc). Locations are
// therefore offsets into the decoded UTF-8 stream, not into the source bytes.
func NewReader(uri string, r io.Reader) *Reader {
	body, bom := xmlenc.Decode(r)
	pos := &posReader{r: xmldecl.As10(body)}
	dec := xml.NewDecoder(pos)
	dec.CharsetReader = bom.CharsetReader
	return &Reader{
		uri: uri,
		dec: dec,
		pos: pos,
		bom: bom,
	}
}

// Token advances to the next element or character-data node and returns it.
// It returns io.EOF at the end of a well-formed document. Comments, processing
// instructions, and directives are skipped; a DOCTYPE directive's entity declarations
// are read on the way past (see HasUnparsedEntity). Malformed input, unbound namespace
// prefixes, and mismatched or unclosed tags are returned as errors carrying an
// xsderr.Loc — never as a panic (see the fuzz target).
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
	case xml.ProcInst:
		if t.Target == "xml" {
			r.standalone = pseudoAttr(string(t.Inst), "standalone") == "yes"
		}
		return nil, false, r.checkDeclaration(t, loc)
	case xml.Directive:
		r.declareEntities(t)
		return nil, false, nil
	default:
		// xml.Comment: not part of the element/character-data stream the
		// parser consumes.
		return nil, false, nil
	}
}

// declareEntities records the general entity declarations of a DOCTYPE
// directive at the document level, keeping the first declaration of each name,
// and whether any declaration went unread. A directive inside an element is no
// DOCTYPE and declares nothing.
func (r *Reader) declareEntities(d xml.Directive) {
	if len(r.stack) > 0 {
		return
	}
	decls, unread := doctypeEntities(string(d), r.standalone)
	if unread {
		r.declsUnread = true
	}
	for _, decl := range decls {
		if _, bound := r.entities[decl.name]; bound {
			continue
		}
		if r.entities == nil {
			r.entities = make(map[string]bool)
		}
		r.entities[decl.name] = decl.unparsed
	}
}

// HasUnparsedEntity reports whether name is the name of an unparsed entity —
// one declared <!ENTITY name SYSTEM|PUBLIC ... NDATA notation> — in the
// document's DOCTYPE internal subset: the [unparsed entities] property of the
// document information item, which an ·ENTITY value· must name (Structures
// §3.16.4 key-vde, Appendix D).
//
// The answer is final once Token has returned the document element's
// StartElement: XML 1.0 places the doctypedecl before the document element,
// so no declaration can arrive later. Asked earlier, it reports only what has
// been read so far.
//
// The internal subset is read with its internal parameter entities expanded;
// the external DTD subset is never read, by design (#1668), nor is an external
// parameter entity, and a declaration after a parameter-entity reference that
// was not read is not processed unless the document is standalone="yes" (XML
// 1.0 §5.1). An unparsed entity declared only where the reader did not read
// is reported false, and AllDeclarationsProcessed then reports false too.
func (r *Reader) HasUnparsedEntity(name string) bool {
	return r.entities[name]
}

// AllDeclarationsProcessed reports the document information item's [all
// declarations processed] (XML Infoset §2.1): whether the reader read the
// complete DTD. It is false when the DOCTYPE names an external subset, which
// is never read, or when the internal subset references a parameter entity
// the reader did not read (see HasUnparsedEntity), whatever the document's
// standalone declaration says.
//
// The answer is final once Token has returned the document element's
// StartElement, on HasUnparsedEntity's terms.
func (r *Reader) AllDeclarationsProcessed() bool {
	return !r.declsUnread
}

// checkDeclaration enforces XML 1.0 §4.3.3's fatal error: "it is a fatal error
// for an entity including an encoding declaration to be presented to the XML
// processor in an encoding other than that named in the declaration". The
// byte-order mark is the evidence of the encoding the entity was presented in.
//
// It catches the direction the XML decoder cannot: a declaration naming UTF-8
// is the decoder's default and never reaches the mark's CharsetReader, so a
// UTF-16 mark contradicting it would otherwise pass unnoticed.
func (r *Reader) checkDeclaration(pi xml.ProcInst, loc xsderr.Loc) error {
	if pi.Target != "xml" {
		return nil
	}
	name := pseudoAttr(string(pi.Inst), "encoding")
	if name == "" || r.bom.AgreesWith(name) {
		return nil
	}
	// xmlenc.Mark.String names the mark itself, so the message states the
	// entity's actual encoding rather than repeating "byte-order mark".
	return xsderr.New(xsderr.RuleXMLWellFormed, loc, "encoding declaration %q disagrees with the entity's actual encoding: %s", name, r.bom)
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
