// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Forked from Go 1.26 src/encoding/xml/xml.go and read.go for goxsd8: the
// strict path only, restyled, with XML 1.0 5e name tables (doc.go).

package xmltok

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kud360/goxsd8/internal/xmlname"
)

// A Decoder reads XML tokens from one input stream, as encoding/xml's
// Decoder does in strict mode. The input is UTF-8 unless an XML declaration
// names another encoding, which CharsetReader then converts.
type Decoder struct {
	// Entity maps entity names other than the five predefined ones (lt, gt,
	// amp, apos, quot, which always resolve) to their replacement text, as
	// encoding/xml's Decoder.Entity does. It is a lookup index only.
	Entity map[string]string

	// CharsetReader, when non-nil, converts the input from the charset an
	// XML declaration names into UTF-8, as encoding/xml's
	// Decoder.CharsetReader does. A declared non-UTF-8 encoding with a nil
	// CharsetReader, or one that fails, stops the parse with an error; it
	// must not return a nil Reader with a nil error.
	CharsetReader func(charset string, input io.Reader) (io.Reader, error)

	r         byteReader
	buf       bytes.Buffer
	stk       *stack
	needClose bool
	toClose   xml.Name
	nextByte  int
	ns        map[string]string
	err       error
	line      int
	offset    int64
}

// byteReader is the input as the decoder reads it: byte at a time, and still
// an io.Reader for a CharsetReader to wrap.
type byteReader interface {
	io.Reader
	io.ByteReader
}

// NewDecoder returns a Decoder reading from r, buffering r itself unless r
// already implements io.ByteReader.
func NewDecoder(r io.Reader) *Decoder {
	d := &Decoder{
		ns:       make(map[string]string),
		nextByte: -1,
		line:     1,
	}
	d.switchToReader(r)
	return d
}

// Token returns the next XML token, or nil and io.EOF at the end of the
// input, as encoding/xml's Decoder.Token does: a self-closing element is a
// StartElement then an EndElement, start and end elements nest and match
// (Element Type Match, xml.md §3), names carry their namespace URL in Space
// where a binding is in scope and the bare prefix where none is, and the byte
// slices in a token are valid only until the next call.
func (d *Decoder) Token() (xml.Token, error) {
	t, err := d.RawToken()
	if err != nil {
		if errors.Is(err, io.EOF) && d.stk != nil {
			err = d.syntaxError("unexpected EOF")
		}
		return nil, err
	}
	switch t1 := t.(type) {
	case xml.StartElement:
		// The namespace declarations among the attributes bind the element
		// name and the other attribute names, so they are applied first.
		for _, a := range t1.Attr {
			if a.Name.Space == xmlnsPrefix {
				v, ok := d.ns[a.Name.Local]
				d.pushNs(a.Name.Local, v, ok)
				d.ns[a.Name.Local] = a.Value
			}
			if a.Name.Space == "" && a.Name.Local == xmlnsPrefix {
				v, ok := d.ns[""]
				d.pushNs("", v, ok)
				d.ns[""] = a.Value
			}
		}
		d.pushElement(t1.Name)
		d.translate(&t1.Name, true)
		for i := range t1.Attr {
			d.translate(&t1.Attr[i].Name, false)
		}
		return t1, nil
	case xml.EndElement:
		if !d.popElement(&t1) {
			return nil, d.err
		}
		return t1, nil
	}
	return t, nil
}

const (
	xmlURL      = "http://www.w3.org/XML/1998/namespace"
	xmlnsPrefix = "xmlns"
	xmlPrefix   = "xml"
)

// translate replaces n's prefix with the namespace URL bound to it. The
// default namespace applies to element names only.
func (d *Decoder) translate(n *xml.Name, isElementName bool) {
	switch {
	case n.Space == xmlnsPrefix:
		return
	case n.Space == "" && !isElementName:
		return
	case n.Space == xmlPrefix:
		n.Space = xmlURL
	case n.Space == "" && n.Local == xmlnsPrefix:
		return
	}
	if v, ok := d.ns[n.Space]; ok {
		n.Space = v
	}
}

func (d *Decoder) switchToReader(r io.Reader) {
	if rb, ok := r.(byteReader); ok {
		d.r = rb
		return
	}
	d.r = bufio.NewReader(r)
}

// stack holds the open elements and the namespace bindings each one
// shadowed. The bindings to restore when an element ends sit below its
// entry, pushed before it.
type stack struct {
	next *stack
	kind int
	name xml.Name
	ok   bool
}

const (
	stkStart = iota
	stkNs
)

func (d *Decoder) push(kind int) *stack {
	s := &stack{next: d.stk, kind: kind}
	d.stk = s
	return s
}

func (d *Decoder) pop() *stack {
	s := d.stk
	if s != nil {
		d.stk = s.next
	}
	return s
}

func (d *Decoder) pushElement(name xml.Name) {
	s := d.push(stkStart)
	s.name = name
}

// pushNs records that ns[local] is being rebound; url and ok are its previous
// value and presence.
func (d *Decoder) pushNs(local string, url string, ok bool) {
	s := d.push(stkNs)
	s.name.Local = local
	s.name.Space = url
	s.ok = ok
}

func (d *Decoder) syntaxError(msg string) error {
	return &xml.SyntaxError{Msg: msg, Line: d.line}
}

// popElement ends the element on top of the stack, which t must match by
// prefix and local name (Element Type Match, xml.md §3), then restores the
// bindings that element shadowed. It reports false with d.err set on a
// mismatch.
func (d *Decoder) popElement(t *xml.EndElement) bool {
	s := d.pop()
	name := t.Name
	switch {
	case s == nil || s.kind != stkStart:
		d.err = d.syntaxError("unexpected end element </" + name.Local + ">")
		return false
	case s.name.Local != name.Local:
		d.err = d.syntaxError("element <" + s.name.Local + "> closed by </" + name.Local + ">")
		return false
	case s.name.Space != name.Space:
		ns := name.Space
		if name.Space == "" {
			ns = `""`
		}
		d.err = d.syntaxError("element <" + s.name.Local + "> in space " + s.name.Space +
			" closed by </" + name.Local + "> in space " + ns)
		return false
	}

	d.translate(&t.Name, true)

	for d.stk != nil && d.stk.kind != stkStart {
		s := d.pop()
		if s.ok {
			d.ns[s.name.Local] = s.name.Space
			continue
		}
		delete(d.ns, s.name.Local)
	}
	return true
}

// RawToken is Token without the end-element match check and without
// namespace translation: names carry their prefix in Space.
func (d *Decoder) RawToken() (xml.Token, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.needClose {
		// The EndElement half of a self-closing element.
		d.needClose = false
		return xml.EndElement{Name: d.toClose}, nil
	}

	b, ok := d.getc()
	if !ok {
		return nil, d.err
	}
	if b != '<' {
		d.ungetc(b)
		data := d.text(-1, false)
		if data == nil {
			return nil, d.err
		}
		return xml.CharData(data), nil
	}

	if b, ok = d.mustgetc(); !ok {
		return nil, d.err
	}
	switch b {
	case '/':
		return d.endElement()
	case '?':
		return d.procInst()
	case '!':
		return d.markup()
	}
	d.ungetc(b)
	return d.startElement()
}

// endElement reads an end tag after its "</": [42] ETag ::= '</' Name S? '>'.
func (d *Decoder) endElement() (xml.Token, error) {
	name, ok := d.nsname()
	if !ok {
		if d.err == nil {
			d.err = d.syntaxError("expected element name after </")
		}
		return nil, d.err
	}
	d.space()
	b, ok := d.mustgetc()
	if !ok {
		return nil, d.err
	}
	if b != '>' {
		d.err = d.syntaxError("invalid characters between </" + name.Local + " and >")
		return nil, d.err
	}
	return xml.EndElement{Name: name}, nil
}

// procInst reads a processing instruction after its "<?", [16] PI with a
// [17] PITarget Name, and switches to CharsetReader's reader when it is an XML
// declaration naming another encoding.
func (d *Decoder) procInst() (xml.Token, error) {
	target, ok := d.name()
	if !ok {
		if d.err == nil {
			d.err = d.syntaxError("expected target name after <?")
		}
		return nil, d.err
	}
	d.space()
	d.buf.Reset()
	var b0 byte
	for {
		b, ok := d.mustgetc()
		if !ok {
			return nil, d.err
		}
		d.buf.WriteByte(b)
		if b0 == '?' && b == '>' {
			break
		}
		b0 = b
	}
	data := d.buf.Bytes()
	data = data[0 : len(data)-2] // chop ?>
	if target != "xml" {
		return xml.ProcInst{Target: target, Inst: data}, nil
	}
	if err := d.xmlDecl(string(data)); err != nil {
		return nil, err
	}
	return xml.ProcInst{Target: target, Inst: data}, nil
}

// xmlDecl applies an XML declaration's version and encoding, setting and
// returning d.err when it cannot.
func (d *Decoder) xmlDecl(content string) error {
	ver := procInstParam("version", content)
	if ver != "" && ver != "1.0" {
		d.err = fmt.Errorf("xml: unsupported version %q; only version 1.0 is supported", ver)
		return d.err
	}
	enc := procInstParam("encoding", content)
	if enc == "" || strings.EqualFold(enc, "utf-8") {
		return nil
	}
	if d.CharsetReader == nil {
		d.err = fmt.Errorf("xml: encoding %q declared but Decoder.CharsetReader is nil", enc)
		return d.err
	}
	newr, err := d.CharsetReader(enc, d.r)
	if err != nil {
		d.err = fmt.Errorf("xml: opening charset %q: %w", enc, err)
		return d.err
	}
	if newr == nil {
		panic("CharsetReader returned a nil Reader for charset " + enc)
	}
	d.switchToReader(newr)
	return nil
}

// markup reads what follows "<!": a comment, a CDATA section or a directive.
func (d *Decoder) markup() (xml.Token, error) {
	b, ok := d.mustgetc()
	if !ok {
		return nil, d.err
	}
	switch b {
	case '-':
		return d.comment()
	case '[':
		return d.cdata()
	}
	return d.directive(b)
}

// comment reads a comment after its "<!-": [15] Comment, in which "--" may
// appear only as part of the closing "-->".
func (d *Decoder) comment() (xml.Token, error) {
	b, ok := d.mustgetc()
	if !ok {
		return nil, d.err
	}
	if b != '-' {
		d.err = d.syntaxError("invalid sequence <!- not part of <!--")
		return nil, d.err
	}
	d.buf.Reset()
	var b0, b1 byte
	for {
		if b, ok = d.mustgetc(); !ok {
			return nil, d.err
		}
		d.buf.WriteByte(b)
		if b0 == '-' && b1 == '-' {
			if b != '>' {
				d.err = d.syntaxError(`invalid sequence "--" not allowed in comments`)
				return nil, d.err
			}
			break
		}
		b0, b1 = b1, b
	}
	data := d.buf.Bytes()
	data = data[0 : len(data)-3] // chop -->
	return xml.Comment(data), nil
}

// cdata reads a CDATA section after its "<![": [18] CDSect.
func (d *Decoder) cdata() (xml.Token, error) {
	const open = "CDATA["
	for i := range len(open) {
		b, ok := d.mustgetc()
		if !ok {
			return nil, d.err
		}
		if b != open[i] {
			d.err = d.syntaxError("invalid <![ sequence")
			return nil, d.err
		}
	}
	data := d.text(-1, true)
	if data == nil {
		return nil, d.err
	}
	return xml.CharData(data), nil
}

// directive reads a directive (<!DOCTYPE ...>, <!ENTITY ...>, ...) after its
// "<!", whose first byte is b, through the '>' that closes it. Quoted '<' and
// '>' do not nest; a comment inside it becomes one space.
func (d *Decoder) directive(b byte) (xml.Token, error) {
	d.buf.Reset()
	d.buf.WriteByte(b)
	var st directiveState
	for {
		b, ok := d.mustgetc()
		if !ok {
			return nil, d.err
		}
		if st.inquote == 0 && b == '>' && st.depth == 0 {
			return xml.Directive(d.buf.Bytes()), nil
		}
		for {
			d.buf.WriteByte(b)
			if !st.step(b) {
				break
			}
			next, isComment, ok := d.directiveComment()
			if !ok {
				return nil, d.err
			}
			if isComment {
				break
			}
			// A '<' opening no comment nests, and the byte that broke the
			// "!--" match is handled next. The close test above cannot
			// apply to it: depth is now positive.
			st.depth++
			b = next
		}
	}
}

// directiveState is a directive's quoting and nesting while it is read.
type directiveState struct {
	inquote byte
	depth   int
}

// step applies one byte of a directive and reports whether it is an unquoted
// '<', which may open a comment.
func (st *directiveState) step(b byte) bool {
	switch {
	case b == st.inquote:
		st.inquote = 0
	case st.inquote != 0:
		// Quoted: no nesting.
	case b == '\'' || b == '"':
		st.inquote = b
	case b == '>':
		st.depth--
	case b == '<':
		return true
	}
	return false
}

// directiveComment follows an unquoted '<' in a directive. On "!--" it reads
// the comment through "-->" and replaces the '<' and the comment with one
// space, so the markup on either side is not joined. Otherwise it writes the
// matched part of "!--" and returns the byte that broke the match.
func (d *Decoder) directiveComment() (next byte, isComment, ok bool) {
	const open = "!--"
	for i := range len(open) {
		b, ok := d.mustgetc()
		if !ok {
			return 0, false, false
		}
		if b != open[i] {
			d.buf.WriteString(open[:i])
			return b, false, true
		}
	}
	d.buf.Truncate(d.buf.Len() - 1)
	var b0, b1 byte
	for {
		b, ok := d.mustgetc()
		if !ok {
			return 0, false, false
		}
		if b0 == '-' && b1 == '-' && b == '>' {
			break
		}
		b0, b1 = b1, b
	}
	d.buf.WriteByte(' ')
	return 0, true, true
}

// startElement reads a start tag after its '<': [40] STag ::= '<' Name (S
// Attribute)* S? '>' or [44] EmptyElemTag ::= '<' Name (S Attribute)* S?
// '/>'.
func (d *Decoder) startElement() (xml.Token, error) {
	name, ok := d.nsname()
	if !ok {
		if d.err == nil {
			d.err = d.syntaxError("expected element name after <")
		}
		return nil, d.err
	}
	attr := []xml.Attr{}
	for {
		d.space()
		b, ok := d.mustgetc()
		if !ok {
			return nil, d.err
		}
		if b == '/' {
			if b, ok = d.mustgetc(); !ok {
				return nil, d.err
			}
			if b != '>' {
				d.err = d.syntaxError("expected /> in element")
				return nil, d.err
			}
			d.needClose = true
			d.toClose = name
			return xml.StartElement{Name: name, Attr: attr}, nil
		}
		if b == '>' {
			return xml.StartElement{Name: name, Attr: attr}, nil
		}
		d.ungetc(b)
		a, ok := d.attribute()
		if !ok {
			return nil, d.err
		}
		attr = append(attr, a)
	}
}

// attribute reads [41] Attribute ::= Name Eq AttValue, reporting false with
// d.err set when it cannot.
func (d *Decoder) attribute() (xml.Attr, bool) {
	var a xml.Attr
	var ok bool
	if a.Name, ok = d.nsname(); !ok {
		if d.err == nil {
			d.err = d.syntaxError("expected attribute name in element")
		}
		return a, false
	}
	d.space()
	b, ok := d.mustgetc()
	if !ok {
		return a, false
	}
	if b != '=' {
		d.err = d.syntaxError("attribute name without = in element")
		return a, false
	}
	d.space()
	data := d.attrval()
	if data == nil {
		return a, false
	}
	a.Value = string(data)
	return a, true
}

// attrval reads a quoted [10] AttValue.
func (d *Decoder) attrval() []byte {
	b, ok := d.mustgetc()
	if !ok {
		return nil
	}
	if b != '"' && b != '\'' {
		d.err = d.syntaxError("unquoted or missing attribute value in element")
		return nil
	}
	return d.text(int(b), false)
}

// space skips [3] S.
func (d *Decoder) space() {
	for {
		b, ok := d.getc()
		if !ok {
			return
		}
		if b != ' ' && b != '\r' && b != '\n' && b != '\t' {
			d.ungetc(b)
			return
		}
	}
}

// getc reads one byte, maintaining the line and offset. It reports false
// with the error left in d.err when there is none.
func (d *Decoder) getc() (byte, bool) {
	if d.err != nil {
		return 0, false
	}
	b := byte(d.nextByte)
	if d.nextByte < 0 {
		var err error
		if b, err = d.r.ReadByte(); err != nil {
			d.err = err
			return 0, false
		}
	}
	d.nextByte = -1
	if b == '\n' {
		d.line++
	}
	d.offset++
	return b, true
}

// InputOffset returns the input byte offset of the decoder's position: the
// end of the token most recently returned and the start of the next, as
// encoding/xml's Decoder.InputOffset does. After a syntax error it is the
// offset the error was found at, which for a name fault is just after the
// name.
func (d *Decoder) InputOffset() int64 {
	return d.offset
}

// mustgetc is getc for a byte the grammar requires: the end of input is the
// syntax error "unexpected EOF".
func (d *Decoder) mustgetc() (byte, bool) {
	b, ok := d.getc()
	if !ok && errors.Is(d.err, io.EOF) {
		d.err = d.syntaxError("unexpected EOF")
	}
	return b, ok
}

// ungetc unreads b, which getc returned last.
func (d *Decoder) ungetc(b byte) {
	if b == '\n' {
		d.line--
	}
	d.nextByte = int(b)
	d.offset--
}

// predefined resolves the five predefined entities (xml.md §4.6), which a
// processor recognises whether or not they are declared.
func predefined(name string) (rune, bool) {
	switch name {
	case "lt":
		return '<', true
	case "gt":
		return '>', true
	case "amp":
		return '&', true
	case "apos":
		return '\'', true
	case "quot":
		return '"', true
	}
	return 0, false
}

// text reads character data, with references resolved and line ends
// normalised (xml.md §2.11). With quote >= 0 it reads an attribute value
// through that closing quote; with cdata it reads a CDATA section through
// "]]>". It returns nil with the error left in d.err on failure.
func (d *Decoder) text(quote int, cdata bool) []byte {
	var b0, b1 byte
	var trunc int
	d.buf.Reset()
	for {
		b, ok := d.getc()
		if !ok {
			if !cdata {
				break
			}
			if errors.Is(d.err, io.EOF) {
				d.err = d.syntaxError("unexpected EOF in CDATA section")
			}
			return nil
		}

		// "]]>" ends a CDATA section and may not appear in character data
		// ([14] CharData); a quoted value may hold it.
		if quote < 0 && b0 == ']' && b1 == ']' && b == '>' {
			if !cdata {
				d.err = d.syntaxError("unescaped ]]> not in CDATA section")
				return nil
			}
			trunc = 2
			break
		}

		if b == '<' && !cdata {
			if quote >= 0 {
				d.err = d.syntaxError("unescaped < inside quoted string")
				return nil
			}
			d.ungetc('<')
			break
		}
		if quote >= 0 && b == byte(quote) {
			break
		}
		if b == '&' && !cdata {
			before := d.buf.Len()
			d.buf.WriteByte('&')
			text, have, ok := d.reference(before)
			if !ok {
				return nil
			}
			if !have {
				ent := string(d.buf.Bytes()[before:])
				if ent[len(ent)-1] != ';' {
					ent += " (no semicolon)"
				}
				d.err = d.syntaxError("invalid character entity " + ent)
				return nil
			}
			d.buf.Truncate(before)
			d.buf.WriteString(text)
			b0, b1 = 0, 0
			continue
		}

		switch {
		case b == '\r':
			d.buf.WriteByte('\n')
		case b1 == '\r' && b == '\n':
			// The '\n' was written for the '\r'.
		default:
			d.buf.WriteByte(b)
		}
		b0, b1 = b1, b
	}
	data := d.buf.Bytes()
	data = data[0 : len(data)-trunc]

	for buf := data; len(buf) > 0; {
		r, size := utf8.DecodeRune(buf)
		if r == utf8.RuneError && size == 1 {
			d.err = d.syntaxError("invalid UTF-8")
			return nil
		}
		buf = buf[size:]
		if !isInCharacterRange(r) {
			d.err = d.syntaxError(fmt.Sprintf("illegal character code %U", r))
			return nil
		}
	}
	return data
}

// reference reads a [67] Reference after its '&', which d.buf holds at
// before, and returns its replacement text when it resolves. ok is false when
// the read failed with d.err set.
func (d *Decoder) reference(before int) (text string, have, ok bool) {
	b, ok := d.mustgetc()
	if !ok {
		return "", false, false
	}
	if b == '#' {
		d.buf.WriteByte(b)
		return d.charRef()
	}
	d.ungetc(b)
	return d.entityRef(before)
}

// charRef reads a [66] CharRef after its "&#".
func (d *Decoder) charRef() (text string, have, ok bool) {
	b, ok := d.mustgetc()
	if !ok {
		return "", false, false
	}
	base := 10
	if b == 'x' {
		base = 16
		d.buf.WriteByte(b)
		if b, ok = d.mustgetc(); !ok {
			return "", false, false
		}
	}
	start := d.buf.Len()
	for '0' <= b && b <= '9' ||
		base == 16 && 'a' <= b && b <= 'f' ||
		base == 16 && 'A' <= b && b <= 'F' {
		d.buf.WriteByte(b)
		if b, ok = d.mustgetc(); !ok {
			return "", false, false
		}
	}
	if b != ';' {
		d.ungetc(b)
		return "", false, true
	}
	s := string(d.buf.Bytes()[start:])
	d.buf.WriteByte(';')
	n, err := strconv.ParseUint(s, base, 64)
	if err != nil || n > unicode.MaxRune {
		// An unparsable reference is reported by the caller as an invalid
		// character entity, so the parse error is not carried.
		return "", false, true
	}
	return string(rune(n)), true, true
}

// entityRef reads a [68] EntityRef after its '&', which d.buf holds at
// before. Its Name is checked as 5e [5] Name; a predefined entity or one in
// d.Entity resolves.
func (d *Decoder) entityRef(before int) (text string, have, ok bool) {
	if !d.readName() && d.err != nil {
		return "", false, false
	}
	b, ok := d.mustgetc()
	if !ok {
		return "", false, false
	}
	if b != ';' {
		d.ungetc(b)
		return "", false, true
	}
	name := d.buf.Bytes()[before+1:]
	d.buf.WriteByte(';')
	if !isName(name) {
		return "", false, true
	}
	s := string(name)
	if r, ok := predefined(s); ok {
		return string(r), true, true
	}
	text, have = d.Entity[s]
	return text, have, true
}

// isInCharacterRange reports whether r is an XML 1.0 [2] Char.
func isInCharacterRange(r rune) bool {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

// nsname reads a Name and splits it at its one colon into prefix (Space)
// and local part, as encoding/xml does: a name with no colon, or an empty
// prefix or local part, is all Local, and one with two colons fails with
// d.err unset, for the caller to report.
func (d *Decoder) nsname() (xml.Name, bool) {
	s, ok := d.name()
	if !ok {
		return xml.Name{}, false
	}
	if strings.Count(s, ":") > 1 {
		return xml.Name{}, false
	}
	space, local, ok := strings.Cut(s, ":")
	if !ok || space == "" || local == "" {
		return xml.Name{Local: s}, true
	}
	return xml.Name{Space: space, Local: local}, true
}

// name reads a [5] Name. A missing name fails with d.err unset (unless the
// input ended), so the caller reports it in context; a name outside 5e [4]
// NameStartChar / [4a] NameChar is the [xml-wf] fault "invalid XML name",
// raised at the offset after the name.
func (d *Decoder) name() (string, bool) {
	d.buf.Reset()
	if !d.readName() {
		return "", false
	}
	b := d.buf.Bytes()
	if !isName(b) {
		d.err = d.syntaxError("invalid XML name: " + string(b))
		return "", false
	}
	return string(b), true
}

// readName appends a name's bytes to d.buf. The name ends at the first
// single-byte character outside isNameByte; every multi-byte character is
// taken, for the caller to check with isName.
func (d *Decoder) readName() bool {
	b, ok := d.mustgetc()
	if !ok {
		return false
	}
	if b < utf8.RuneSelf && !isNameByte(b) {
		d.ungetc(b)
		return false
	}
	d.buf.WriteByte(b)
	for {
		if b, ok = d.mustgetc(); !ok {
			return false
		}
		if b < utf8.RuneSelf && !isNameByte(b) {
			d.ungetc(b)
			return true
		}
		d.buf.WriteByte(b)
	}
}

// isNameByte reports whether ASCII c is a [4a] NameChar: 5e's ASCII
// NameChars are exactly these, so the delimiting is the edition's own.
func isNameByte(c byte) bool {
	return 'A' <= c && c <= 'Z' ||
		'a' <= c && c <= 'z' ||
		'0' <= c && c <= '9' ||
		c == '_' || c == ':' || c == '.' || c == '-'
}

// isName reports whether s is an XML 1.0 5e [5] Name ::= NameStartChar
// (NameChar)*, over internal/xmlname's [4] and [4a] tables.
func isName(s []byte) bool {
	if len(s) == 0 {
		return false
	}
	r, n := utf8.DecodeRune(s)
	if r == utf8.RuneError && n == 1 {
		return false
	}
	if !unicode.Is(xmlname.NameStartChar, r) {
		return false
	}
	for s = s[n:]; len(s) > 0; s = s[n:] {
		r, n = utf8.DecodeRune(s)
		if r == utf8.RuneError && n == 1 {
			return false
		}
		if !unicode.In(r, xmlname.NameStartChar, xmlname.NameCharExtra) {
			return false
		}
	}
	return true
}

// procInstParam returns the value of pseudo-attribute param in an XML
// declaration's content s, or "" when it has none. Like encoding/xml's
// procInst it matches "param=" anywhere and does not parse [23] XMLDecl.
func procInstParam(param, s string) string {
	param += "="
	lenp := len(param)
	i := 0
	var sep byte
	for i < len(s) {
		sub := s[i:]
		k := strings.Index(sub, param)
		if k < 0 || lenp+k >= len(sub) {
			return ""
		}
		i += lenp + k + 1
		if c := sub[lenp+k]; c == '\'' || c == '"' {
			sep = c
			break
		}
	}
	if sep == 0 {
		return ""
	}
	j := strings.IndexByte(s[i:], sep)
	if j < 0 {
		return ""
	}
	return s[i : i+j]
}

// Skip reads tokens through the end element matching the start element most
// recently consumed, skipping nested elements, as encoding/xml's
// Decoder.Skip does. It returns nil on that end element and the error
// otherwise.
func (d *Decoder) Skip() error {
	var depth int64
	for {
		tok, err := d.Token()
		if err != nil {
			return err
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			if depth == 0 {
				return nil
			}
			depth--
		}
	}
}
