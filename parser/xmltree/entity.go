package xmltree

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/xsderr"
)

// Bounds on general-entity inclusion. A reference past either one is refused,
// not read: maxGEDepth caps how deeply inclusions nest, and maxGEExpansion
// caps the bytes of replacement text included across one document, so
// entities that each reference another several times cannot blow the read up
// (the "billion laughs" document). WFC No Recursion keeps a well-formed
// document's nesting below the number of entities it declares.
const (
	maxGEDepth     = 64
	maxGEExpansion = 1 << 20
)

// predefined maps the names of XML 1.0 §4.6's predefined entities to the
// character each one escapes. A reference to one is that character whether or
// not the document declares it, and a declaration of one changes nothing
// (§4.6 requires it to declare the same character). It is a lookup index only,
// never iterated.
var predefined = map[string]string{"lt": "<", "gt": ">", "amp": "&", "apos": "'", "quot": `"`}

// cdataOpen opens a CDATA section, inside which '&' begins no reference.
const cdataOpen = "<![CDATA["

// refersToEntity reports whether raw, the source of one character-data token,
// references a general entity other than a predefined one. A CDATA section
// references none.
func refersToEntity(raw string) bool {
	if strings.HasPrefix(raw, cdataOpen) {
		return false
	}
	for {
		amp := strings.IndexByte(raw, '&')
		if amp < 0 {
			return false
		}
		raw = raw[amp+1:]
		name, _, _ := strings.Cut(raw, ";")
		if _, builtin := predefined[name]; !builtin && !strings.HasPrefix(name, "#") {
			return true
		}
	}
}

// source returns the source of the token just read, from its first byte at
// offset off.
func (r *Reader) source(off int64) string {
	return r.pos.span(off, r.dec.InputOffset())
}

// included reads raw, the source of one character-data token that references
// a general entity, starting at offset off: in place of each reference it
// includes the entity's replacement text, parsed as content (XML 1.0 §4.4.2).
// It returns the first node that produces and queues the rest on pending.
// Character data reads as one CharData up to each element the inclusion
// produces, located where its first character is, or, for a character inside
// replacement text, where the outermost reference to it is.
//
// The decoder has already read raw, and charged every reference in it that
// names no internal entity this reader read the declaration of: one declared
// after a parameter-entity reference the reader did not read, outside a
// standalone document, among them. That refusal is the reader's policy and no
// well-formedness verdict, since an entity the reader did not read may be
// declared (XML 1.0 §5.1, WFC Entity Declared). A reference outside the
// document element is no content at all (XML 1.0 [1] document).
func (r *Reader) included(raw string, off int64, loc xsderr.Loc) (Node, bool, error) {
	if len(r.stack) == 0 {
		return nil, false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "entity reference outside the document element: only element content may reference a general entity (XML 1.0 [1] document, [43] content)")
	}
	c := content{r: r}
	if err := c.chars(raw, func(i int) int64 { return off + int64(i) }, true, nil); err != nil {
		return nil, false, err
	}
	c.flush()
	if len(c.nodes) == 0 {
		return nil, false, nil
	}
	r.pending = c.nodes[1:]
	return c.nodes[0], true, nil
}

// content collects the nodes an inclusion in content produces: the nodes so
// far, and the character data read since the last of them, which began at
// offset at.
type content struct {
	r     *Reader
	nodes []Node
	text  strings.Builder
	at    int64
}

// char appends s, located at offset at, to the character data being read.
func (c *content) char(s string, at int64) {
	if s == "" {
		return
	}
	if c.text.Len() == 0 {
		c.at = at
	}
	c.text.WriteString(s)
}

// flush emits the character data read so far as one CharData.
func (c *content) flush() {
	if c.text.Len() == 0 {
		return
	}
	c.nodes = append(c.nodes, &CharData{data: c.text.String(), offset: c.at, loc: c.r.locAt(c.at)})
	c.text.Reset()
}

// node emits n after the character data read so far.
func (c *content) node(n Node) {
	c.flush()
	c.nodes = append(c.nodes, n)
}

// chars reads s, the source of one run of character data, located through at
// (the offset of s's i-th byte). In document source (src), a line end reads as
// #xA (XML 1.0 §2.11); in replacement text, whose line ends the declaration
// normalized, every character is data. A character reference and a predefined
// entity read as the character they name, and a reference to a general entity
// includes it (include). open names the entities whose inclusion s is part of,
// outermost first.
func (c *content) chars(s string, at func(int) int64, src bool, open []string) error {
	for i := 0; i < len(s); {
		amp := strings.IndexByte(s[i:], '&')
		if amp < 0 {
			amp = len(s) - i
		}
		lit := s[i : i+amp]
		if src {
			lit = lineEnds.Replace(lit)
		}
		c.char(lit, at(i))
		i += amp
		if i == len(s) {
			return nil
		}
		loc := c.r.locAt(at(i))
		end := strings.IndexByte(s[i:], ';')
		if end < 0 {
			return unterminated(loc)
		}
		name := s[i+1 : i+end]
		text, entity, err := c.r.reference(name, loc)
		if err != nil {
			return err
		}
		if !entity {
			c.char(text, at(i))
			i += end + 1
			continue
		}
		if err := c.include(name, text, at(i), open); err != nil {
			return err
		}
		i += end + 1
	}
	return nil
}

// include parses text, the replacement text of the entity name referenced at
// offset ref, as content (XML 1.0 §4.4.2, §4.3.2 well-formed parsed entity):
// an element it opens must close in it, and it closes none it did not open.
// Every node it produces is located at the reference.
func (c *content) include(name, text string, ref int64, open []string) error {
	loc := c.r.locAt(ref)
	open, err := c.r.charge(name, text, loc, open)
	if err != nil {
		return err
	}
	dec := xml.NewDecoder(strings.NewReader(text))
	dec.Entity = c.r.dec.Entity
	depth := len(c.r.stack)
	at := func(int) int64 { return ref }
	for {
		from := dec.InputOffset()
		tok, err := dec.RawToken()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return xsderr.Wrap(xsderr.RuleXMLWellFormed, loc, fmt.Errorf("in the replacement text of entity %s: %w", name, err))
		}
		raw := text[from:dec.InputOffset()]
		if err := c.token(tok, raw, at, depth, loc, open); err != nil {
			return err
		}
	}
	if len(c.r.stack) > depth {
		return xsderr.New(xsderr.RuleXMLWellFormed, loc, "element %s opened in the replacement text of entity %s does not close in it (XML 1.0 §4.3.2, [43] content)", qname(c.r.stack[len(c.r.stack)-1].name), name)
	}
	return nil
}

// token reads one token of replacement text, whose source is raw, into c.
// depth is the number of elements open where the inclusion began.
func (c *content) token(tok xml.Token, raw string, at func(int) int64, depth int, loc xsderr.Loc, open []string) error {
	switch t := tok.(type) {
	case xml.StartElement:
		attrs, err := c.r.expandAttrs(t.Attr, raw, false, loc, open)
		if err != nil {
			return err
		}
		t.Attr = attrs
		node, err := c.r.startElement(t, loc)
		if err != nil {
			return err
		}
		c.node(node)
	case xml.EndElement:
		if len(c.r.stack) == depth {
			return xsderr.New(xsderr.RuleXMLWellFormed, loc, "end tag </%s> in the replacement text of entity %s closes an element the entity did not open (XML 1.0 §4.3.2, [43] content)", rawName(t.Name), open[len(open)-1])
		}
		node, err := c.r.endElement(t, loc)
		if err != nil {
			return err
		}
		c.node(node)
	case xml.CharData:
		if body, ok := strings.CutPrefix(raw, cdataOpen); ok {
			c.char(strings.TrimSuffix(body, "]]>"), at(0))
			return nil
		}
		return c.chars(raw, at, false, open)
	}
	return nil
}

// charge admits the inclusion of entity name, whose replacement text is text,
// inside the inclusions open names, and returns open with name appended. It
// refuses a reference to an entity already being included (WFC No Recursion),
// and one past maxGEDepth or maxGEExpansion, which is no verdict on the
// document and so wraps a cause.
func (r *Reader) charge(name, text string, loc xsderr.Loc, open []string) ([]string, error) {
	if slices.Contains(open, name) {
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "entity %s references itself (XML 1.0 WFC No Recursion)", name)
	}
	if len(open) == maxGEDepth || r.spent+len(text) > maxGEExpansion {
		return nil, xsderr.Wrap(xsderr.RuleXMLWellFormed, loc, fmt.Errorf("%w: including entity %s nests deeper than %d entities or includes more than %d bytes of replacement text in the document", errExpansionBound, name, maxGEDepth, maxGEExpansion))
	}
	r.spent += len(text)
	return append(slices.Clip(open), name), nil
}

// unterminated charges a '&' that no ';' closes, which begins no Reference
// (XML 1.0 [67]) and may not stand as data ([10] AttValue, [14] CharData).
func unterminated(loc xsderr.Loc) error {
	return xsderr.New(xsderr.RuleXMLWellFormed, loc, "'&' begins no reference: no ';' closes it (XML 1.0 [67] Reference)")
}

// errExpansionBound is the cause of a reference refused at maxGEDepth or
// maxGEExpansion: a bound of this reader, not a fault of the document.
var errExpansionBound = errors.New("general-entity expansion bound reached")

// reference resolves the name a reference spells between '&' and ';', read at
// loc. A character reference and a predefined entity resolve to the character
// they name; a general entity to its replacement text, with entity set. A
// name that resolves to neither — an undeclared entity, an external or
// unparsed one, one declared where the reader did not read — is refused, as
// the decoder refuses it; a malformed character reference is not well-formed
// (WFC Legal Character).
func (r *Reader) reference(name string, loc xsderr.Loc) (text string, entity bool, err error) {
	if digits, ok := strings.CutPrefix(name, "#"); ok {
		c, legal := charRef(digits)
		if !legal {
			return "", false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "character reference &%s; names no XML character (XML 1.0 WFC Legal Character)", name)
		}
		return string(c), false, nil
	}
	if c, ok := predefined[name]; ok {
		return c, false, nil
	}
	decl := r.entities[name]
	if !decl.value.readable {
		return "", false, xsderr.Wrap(xsderr.RuleXMLWellFormed, loc, fmt.Errorf("reference to entity &%s;, which is not an internal entity the reader read the declaration of", name))
	}
	return decl.value.text, true, nil
}

// expandAttrs returns attrs, the attributes of the start tag whose source is
// raw, with the value of each that references a general entity re-read from
// raw and normalized (attrValue). src reports that raw is document source
// rather than replacement text; open names the inclusions raw is part of. An
// attribute referencing no general entity keeps the decoder's value.
func (r *Reader) expandAttrs(attrs []xml.Attr, raw string, src bool, loc xsderr.Loc, open []string) ([]xml.Attr, error) {
	if !strings.Contains(raw, "&") {
		return attrs, nil
	}
	vals := attrSources(raw)
	if len(vals) != len(attrs) {
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "start tag re-read found %d attribute values where the decoder read %d", len(vals), len(attrs))
	}
	out := slices.Clone(attrs)
	for i, v := range vals {
		if !refersToEntity(v) {
			continue
		}
		var b strings.Builder
		if err := r.attrValue(&b, v, src, loc, open); err != nil {
			return nil, err
		}
		out[i].Value = b.String()
	}
	return out, nil
}

// attrSources returns the source of each attribute value in raw, a start
// tag's source, in document order: the text between the value's quotes. The
// decoder has read raw as a start tag, and a quote can begin nothing else in
// one.
func attrSources(raw string) []string {
	var vals []string
	for {
		open := strings.IndexAny(raw, `"'`)
		if open < 0 {
			return vals
		}
		end := strings.IndexByte(raw[open+1:], raw[open])
		if end < 0 {
			return vals
		}
		vals = append(vals, raw[open+1:open+1+end])
		raw = raw[open+1+end+1:]
	}
}

// attrValue appends to b the normalized value of s, an attribute value's
// source or replacement text it includes, per XML 1.0 §3.3.3: a character
// reference appends the character it names, a white-space character appends
// #x20, and a reference to a general entity appends the normalized value of
// its replacement text, which must hold no '<' (WFC No < in Attribute Values),
// whatever depth the reference is at. In document source (src) a line end is
// one character (§2.11).
func (r *Reader) attrValue(b *strings.Builder, s string, src bool, loc xsderr.Loc, open []string) error {
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '&':
			end := strings.IndexByte(s[i:], ';')
			if end < 0 {
				return unterminated(loc)
			}
			name := s[i+1 : i+end]
			i += end
			text, entity, err := r.reference(name, loc)
			if err != nil {
				return err
			}
			if !entity {
				b.WriteString(text)
				continue
			}
			if strings.ContainsRune(text, '<') {
				return xsderr.New(xsderr.RuleXMLWellFormed, loc, "the replacement text of entity %s, referenced in an attribute value, holds '<' (XML 1.0 WFC No < in Attribute Values)", name)
			}
			inner, err := r.charge(name, text, loc, open)
			if err != nil {
				return err
			}
			if err := r.attrValue(b, text, false, loc, inner); err != nil {
				return err
			}
		case src && c == '\r' && strings.HasPrefix(s[i+1:], "\n"):
			b.WriteByte(' ')
			i++
		case strings.IndexByte(declSpace, c) >= 0:
			b.WriteByte(' ')
		default:
			b.WriteByte(c)
		}
	}
	return nil
}
