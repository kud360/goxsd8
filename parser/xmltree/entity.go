package xmltree

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/kud360/goxsd8/internal/xmltok"
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
// The decoder has already read raw, and refused every reference in it that
// names no entity this reader read a declaration of — one declared after a
// parameter-entity reference the reader did not read, outside a standalone
// document, among them; Reader.reference charges or refuses a reference to any
// other entity that is not internal (see Reader.entities). The decoder's
// refusal is the reader's policy and no well-formedness verdict, since an
// entity the reader did not read may be declared (XML 1.0 §5.1, WFC Entity
// Declared). A reference outside the document element is no content at all
// (XML 1.0 [1] document).
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
		if end < 0 || !isReference(s[i+1:i+end]) {
			return noReference(loc, open)
		}
		name := s[i+1 : i+end]
		text, entity, err := c.r.reference(name, loc, open, false)
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
// Every node it produces is located at the reference. A syntax error the
// decoder finds in a token holding a '&' that begins no Reference is that
// fault, charged wrapping no cause (noReference, strayInToken); any other is
// wrapped as the cause.
func (c *content) include(name, text string, ref int64, open []string) error {
	loc := c.r.locAt(ref)
	open, err := c.r.charge(name, text, loc, open)
	if err != nil {
		return err
	}
	dec := xmltok.NewDecoder(strings.NewReader(text))
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
			if strayInToken(text[from:]) {
				return noReference(loc, open)
			}
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

// strayInToken reports whether src, replacement text from the first byte of
// the token the decoder failed to read, holds a '&' that begins no Reference
// (strayAmp) in that token: in character data, which runs to the next '<', or
// in an attribute value of a start tag, which runs to the '>' outside its
// literals that closes it. A comment, CDATA section, processing instruction,
// end tag or unclosed start tag holds none it checks.
func strayInToken(src string) bool {
	if !strings.HasPrefix(src, "<") {
		data, _, _ := strings.Cut(src, "<")
		return strayAmp(data)
	}
	end := outsideQuotes(src, ">")
	if end < 0 || strings.HasPrefix(src, "<!") || strings.HasPrefix(src, "<?") || strings.HasPrefix(src, "</") {
		return false
	}
	return slices.ContainsFunc(attrSources(src[:end]), strayAmp)
}

// token reads one token of replacement text, whose source is raw, into c.
// depth is the number of elements open where the inclusion began. A comment
// and a processing instruction add nothing; one targeting "xml" in any case
// ([17] PITarget) and any directive are RuleXMLWellFormed faults at loc, the
// reference, since replacement text included in content must match [43]
// content (XML 1.0 §4.3.2).
func (c *content) token(tok xml.Token, raw string, at func(int) int64, depth int, loc xsderr.Loc, open []string) error {
	switch t := tok.(type) {
	case xml.StartElement:
		attrs, err := c.r.expandAttrs(t, raw, false, loc, open)
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
	case xml.ProcInst:
		if strings.EqualFold(t.Target, "xml") {
			return xsderr.New(xsderr.RuleXMLWellFormed, loc, "processing instruction target %q in the replacement text of entity %s, a name XML 1.0 [17] PITarget excludes", t.Target, open[len(open)-1])
		}
	case xml.Directive:
		return xsderr.New(xsderr.RuleXMLWellFormed, loc, "directive %q in the replacement text of entity %s, where XML 1.0 [43] content admits no directive", directiveName(raw), open[len(open)-1])
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

// noReference charges a '&' that begins no Reference (XML 1.0 [67]): no ';'
// closes it, or what stands before the ';' is neither an EntityRef's Name
// ([68]) nor a CharRef's digits ([66]) (see isReference). It may stand as data
// in neither an attribute value ([10] AttValue) nor content ([14] CharData),
// and replacement text is reparsed where it is included (§4.4.2, §4.4.5), so a
// '&' a literal spelled `&#38;` or `&#x26;` is this fault wherever the entity
// is referenced, unless a Reference follows it (§4.5, Appendix D). open names
// the inclusions the '&' is part of, outermost first.
func noReference(loc xsderr.Loc, open []string) error {
	if len(open) == 0 {
		return xsderr.New(xsderr.RuleXMLWellFormed, loc, "'&' begins no Reference, '&' Name ';' or a character reference (XML 1.0 [67] Reference, [68] EntityRef, [66] CharRef)")
	}
	return xsderr.New(xsderr.RuleXMLWellFormed, loc, "the replacement text of entity %s holds a '&' that begins no Reference, '&' Name ';' or a character reference (XML 1.0 §4.4.2, [67] Reference, [68] EntityRef, [66] CharRef)", open[len(open)-1])
}

// errExpansionBound is the cause of a reference refused at maxGEDepth or
// maxGEExpansion: a bound of this reader, not a fault of the document.
var errExpansionBound = errors.New("general-entity expansion bound reached")

// reference resolves the name a reference spells between '&' and ';', read at
// loc inside the inclusions open, outermost first. A character reference and a
// predefined entity resolve to the character they name; a general entity to
// its replacement text, with entity set. A malformed character reference is
// not well-formed (WFC Legal Character), and so, in a standalone="yes"
// document, is a reference that does not occur within a parameter entity
// (withinPE) to an entity declared only in a parameter entity's replacement
// text, which WFC Entity Declared does not count (XML 1.0 wf-entdeclared): the
// clause keys on where the declaration stands, not on what it declares, so it
// is charged before readability is asked, for an external entity, an unparsed
// one among them, as for an internal one. Outside a standalone document an
// internal subset that declares one has referenced a parameter entity, which
// lifts that constraint. Next, in an attribute value (attr), a reference to an
// unparsed or external entity whose declaration the reader read is charged
// (attrEntityFault). Any other name that resolves to no replacement text — an
// undeclared entity, an external or unparsed one referenced in content outside
// the Entity Declared charge, one declared where the reader did not read — is
// refused, wrapping a cause, as the decoder refuses a name no recorded
// declaration names (see Reader.entities).
func (r *Reader) reference(name string, loc xsderr.Loc, open []string, attr bool) (text string, entity bool, err error) {
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
	bound, declared := r.entities[name]
	if r.standalone && bound.onlyInPE && !r.withinPE(open) {
		return "", false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "reference to entity &%s; in a standalone=\"yes\" document, where no general entity declaration outside every parameter entity declares it (XML 1.0 WFC Entity Declared)", name)
	}
	if attr && declared {
		if err := attrEntityFault(bound.binding, loc); err != nil {
			return "", false, err
		}
	}
	if !bound.binding.value.readable {
		return "", false, xsderr.Wrap(xsderr.RuleXMLWellFormed, loc, fmt.Errorf("reference to entity &%s;, which is not an internal entity the reader read the declaration of", name))
	}
	return bound.binding.value.text, true, nil
}

// attrEntityFault charges a reference in an attribute value, directly or
// through replacement text, to the entity whose binding declaration (XML 1.0
// §4.2) is d, located at loc, if an attribute value may not reference it, in
// the order entityGraph.enter checks the two: an unparsed entity (WFC: Parsed
// Entity) or an external one (WFC: No External Entity References). Neither
// constraint keys on standalone or on where d stands, but d must be a
// declaration the reader read: a name declared only where it did not read has
// no binding to decide them (XML 1.0 §5.1).
func attrEntityFault(d entityDecl, loc xsderr.Loc) error {
	switch {
	case d.unparsed:
		return xsderr.New(xsderr.RuleXMLWellFormed, loc, "attribute value that references, directly or indirectly, the unparsed entity %s (XML 1.0 WFC: Parsed Entity)", d.name)
	case !d.value.readable:
		return xsderr.New(xsderr.RuleXMLWellFormed, loc, "attribute value that references, directly or indirectly, the external entity %s (XML 1.0 WFC: No External Entity References)", d.name)
	}
	return nil
}

// withinPE reports whether a reference inside the inclusions open, outermost
// first, occurs within a parameter entity, where WFC Entity Declared does not
// bind it (XML 1.0 wf-entdeclared): whether the binding declaration (§4.2) of
// the innermost entity, whose replacement text holds the reference, stands in
// a parameter entity's replacement text. XML 1.0 does not say where a
// reference nested in replacement text occurs; this reading (#2365) declines
// to charge as a definite fault what the text leaves open. Only the innermost
// entity counts: a reference in the text of an entity declared outside every
// parameter entity occurs outside them, though that entity is itself
// referenced in the text of one declared in a parameter entity. A reference in
// document source is inside no inclusion and within none.
func (r *Reader) withinPE(open []string) bool {
	return len(open) > 0 && r.entities[open[len(open)-1]].binding.inPE
}

// expandAttrs returns the attributes of t, the start tag whose source is raw,
// with each value normalized per XML 1.0 §3.3.3: steps 1–3 (attrValue), then,
// for an attribute the internal subset defines on t's element type with an
// AttType other than CDATA (Reader.tokenized), the paragraph after step 3
// (collapseSpace). An attribute it defines as CDATA, or not at all, is read as
// CDATA (§3.3.3's last paragraph). The decoder's value is already steps 1–3's
// for a value whose source holds neither a reference to a general entity nor a
// literal #x9, #xA or #xD (attrSpace): the decoder resolves character
// references and predefined entities and applies §2.11, but not step 3. Every
// other value is re-read from raw. src reports that raw is document source
// rather than replacement text; open names the inclusions raw is part of.
func (r *Reader) expandAttrs(t xml.StartElement, raw string, src bool, loc xsderr.Loc, open []string) ([]xml.Attr, error) {
	reread := strings.ContainsAny(raw, "&"+attrSpace)
	if !reread && r.tokenized == nil {
		return t.Attr, nil
	}
	var vals []string
	if reread {
		vals = attrSources(raw)
		if len(vals) != len(t.Attr) {
			return nil, xsderr.New(xsderr.RuleXMLWellFormed, loc, "start tag re-read found %d attribute values where the decoder read %d", len(vals), len(t.Attr))
		}
	}
	out := slices.Clone(t.Attr)
	for i := range out {
		if reread && (refersToEntity(vals[i]) || strings.ContainsAny(vals[i], attrSpace)) {
			var b strings.Builder
			if err := r.attrValue(&b, vals[i], src, loc, open); err != nil {
				return nil, err
			}
			out[i].Value = b.String()
		}
		if r.tokenized[attName{elem: rawName(t.Name), name: rawName(out[i].Name)}] {
			out[i].Value = collapseSpace(out[i].Value)
		}
	}
	return out, nil
}

// collapseSpace returns v, an attribute's normalized value after XML 1.0
// §3.3.3's step 3, with leading and trailing #x20 discarded and each run of
// #x20 replaced by one: the paragraph after step 3, for an attribute whose
// type is not CDATA. It touches #x20 alone, so a #x9, #xA or #xD a character
// reference produced stays as it is.
func collapseSpace(v string) string {
	return strings.Join(strings.FieldsFunc(v, func(c rune) bool { return c == ' ' }), " ")
}

// attrSpace is the white space §3.3.3's step 3 maps to #x20 and the decoder
// leaves in an attribute value as it is: declSpace less #x20 itself.
const attrSpace = "\t\n\r"

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
			if end < 0 || !isReference(s[i+1:i+end]) {
				return noReference(loc, open)
			}
			name := s[i+1 : i+end]
			i += end
			text, entity, err := r.reference(name, loc, open, true)
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
