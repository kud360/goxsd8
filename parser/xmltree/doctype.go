package xmltree

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/kud360/goxsd8/internal/xmlchar"
	"github.com/kud360/goxsd8/internal/xmlname"
	"github.com/kud360/goxsd8/xsderr"
)

// entityDecl is one general entity declaration of a DOCTYPE's internal
// subset: the entity's name, whether it is UNPARSED — an external entity
// carrying an NDATA notation name, which is what an ·ENTITY value· must name
// (Structures §3.16.4 key-vde) — and, for an internal entity, its replacement
// text, which a reference to it includes (XML 1.0 §4.4.2, §4.4.5). value is
// unreadable for every other entity: an external one, unparsed or not, and
// one whose definition is malformed.
type entityDecl struct {
	name     string
	unparsed bool
	value    entityValue
}

// Bounds on parameter-entity expansion. A reference past either one is
// declined, not expanded: it is XML 1.0 §5.1's "reference to a parameter
// entity that is not read", with the cost doctypeEntities states. maxPEDepth
// caps how deeply expansions nest; maxPEExpansion caps the bytes of
// replacement text scanned across one internal subset, so parameter entities
// that reference one another through a character-referenced '%' cannot blow
// the scan up.
const (
	maxPEDepth     = 64
	maxPEExpansion = 1 << 20
)

// doctypeEntities reads the general entity declarations of one directive's
// DOCTYPE, in document order, and reports whether some declaration went
// unread: the inverse of the document's [all declarations processed] (XML
// Infoset §2.1). directive is the directive's source between its "<!" and its
// closing '>', comments included (see Reader.declareEntities). A directive
// that is not a DOCTYPE declares none and leaves nothing unread. standalone
// is the XML declaration's standalone="yes" (XML 1.0 §2.9 SDDecl). A DOCTYPE
// whose document type name is missing or is not a Name (XML 1.0 [28]
// doctypedecl, [5] Name) is not well-formed: a RuleXMLWellFormed fault at
// loc, the directive's start.
//
// The external DTD subset is never read, by design (XML 1.0 §5.1 and §5.2
// oblige a non-validating processor to read the document entity alone;
// ruled on #1668): an ExternalID in the DOCTYPE header reports unread and
// nothing more. The internal subset is read in full, and a parameter-entity
// reference between its declarations (DeclSep) is expanded in place when the
// entity is internal and already declared: its replacement text is the
// literal with each character reference replaced (§4.5), enlarged by one space
// either side (§4.4.8). A reference it does not read — to an external or
// undeclared parameter entity, to one whose replacement text it cannot build,
// or past maxPEDepth or maxPEExpansion, which a recursive reference (WFC No
// Recursion) always reaches — reports unread. Unless standalone, the rest of
// the internal subset is then only checked for well-formedness, which §5.1
// requires of the entire internal subset: it binds no parameter entity,
// records no general entity and expands no parameter-entity reference, since
// §5.1 forbids processing an entity declaration that follows a reference to a
// parameter entity that is not read, except when standalone="yes". A
// conditional section in replacement text, which this scan does not read, is
// declined the same way and ends the text it appears in.
//
// Every fault below is a RuleXMLWellFormed fault at loc, which ends the read
// and declares nothing. Between declarations, in the internal subset and in an
// expanded parameter entity's replacement text alike, only S and PEReferences
// may stand (XML 1.0 [28b] intSubset, [28a] DeclSep, WFC: PE Between
// Declarations): any other text — a '%' run that is no PEReference ([69]), a
// '<!' opening no markup declaration, a conditional section in the internal
// subset itself, and a ']' in replacement text among it — is a fault, and so
// is a subset no ']' closes or text other than S between that ']' and the
// directive's '>' ([28] doctypedecl), and so is a '<' in the DOCTYPE header,
// before the subset ([28]). A comment holding "--" ([15] Comment), a
// processing instruction whose target is no Name or is "xml" in any case ([16]
// PI, [17] PITarget), a notation declaration that is no [82] NotationDecl, a
// '<' outside the literals of any markup declaration, a parameter-entity
// reference inside any markup declaration, an entity value literal included
// (WFC: PEs in Internal Subset), any other '%' in an entity value literal ([9]
// EntityValue) and replacement text that ends inside a comment, processing
// instruction or markup declaration (WFC: PE Between Declarations) are faults
// too. A declaration it cannot read declares no unparsed entity, which leaves
// an ·ENTITY value· naming that entity undeclared rather than declared.
//
// GAP(xml): the bodies of <!ELEMENT> and <!ATTLIST> declarations ([45]–[60])
// are checked for nothing but a parameter-entity reference, a '<' and the '>'
// that closes them, and an <!ENTITY> declaration ([70]–[76]) for nothing
// beyond those and what entityDeclOf and paramEntityOf read: a declaration
// that breaks its production there is stepped over, or declares nothing, where
// it is not well-formed. Tracked by #2225.
//
// GAP(xml): encoding/xml ends the directive at a '>' inside a processing
// instruction, so the subset text this scan reads may be cut short: a comment,
// processing instruction or markup declaration left open where the internal
// subset's own text ends is declined, not a fault, and the ']' missing behind
// it is no fault either. Tracked by #2226.
func doctypeEntities(directive string, standalone bool, loc xsderr.Loc) (decls []entityDecl, unread bool, err error) {
	rest, ok := strings.CutPrefix(directive, "DOCTYPE")
	if !ok {
		return nil, false, nil
	}
	header := rest
	open := outsideQuotes(rest, "[<")
	if open >= 0 {
		header = rest[:open]
	}
	if name := doctypeName(header); !isName(name) {
		return nil, false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "DOCTYPE document type name %q is not a Name (XML 1.0 [28] doctypedecl, [5] Name)", name)
	}
	if open >= 0 && rest[open] == '<' {
		return nil, false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "DOCTYPE holds %q before its internal subset, where only S, its name and an ExternalID may stand (XML 1.0 [28] doctypedecl)", excerpt(rest[open:]))
	}
	sc := subsetScan{standalone: standalone, loc: loc, unread: hasExternalID(header)}
	if open < 0 {
		return nil, sc.unread, nil
	}
	if err := sc.scan(rest[open+1:]); err != nil {
		return nil, false, err
	}
	return sc.decls, sc.unread, nil
}

// doctypeName is a DOCTYPE header's first token, the document type name, or
// "" when the header holds none. The token runs on to the next white space, as
// entityDefTokens' do: XML 1.0 [28] doctypedecl admits only S, '[' or '>'
// after the Name, so a header with a quote run on into its name, `r"x"`, holds
// a token that is no Name.
func doctypeName(header string) string {
	toks := splitDecl(header, true)
	if len(toks) == 0 {
		return ""
	}
	return toks[0]
}

// hasExternalID reports whether a DOCTYPE header — the text after the DOCTYPE
// keyword, up to its internal subset — names an external subset: the
// ExternalID XML 1.0 doctypedecl admits after the document type name.
func hasExternalID(header string) bool {
	toks := declTokens(header)
	return len(toks) > 1 && (toks[1] == "SYSTEM" || toks[1] == "PUBLIC")
}

// subsetScan is one read of a DOCTYPE's internal subset. pes maps each
// parameter entity declared so far to its replacement text, keeping the first
// declaration of a name (XML 1.0 §4.2); it is a lookup index only, never
// iterated. depth counts the expansions in progress, and spent the bytes of
// replacement text scanned so far. decls collects the general entity
// declarations read, and unread records that some declaration was not.
// checkOnly records that a declined reference has ended processing, outside a
// standalone document (XML 1.0 §5.1): the scan reads on only to check
// well-formedness. loc is the directive's start, where every fault the scan
// finds is located.
type subsetScan struct {
	standalone bool
	loc        xsderr.Loc
	pes        map[string]entityValue
	depth      int
	spent      int
	decls      []entityDecl
	unread     bool
	checkOnly  bool
}

// entityValue is one entity's replacement text, and whether it has one the
// reader can read — false for an external entity, a malformed definition, or
// a literal whose replacement text cannot be built.
type entityValue struct {
	text     string
	readable bool
}

// scan reads s — the internal subset, or a parameter entity's enlarged
// replacement text — and returns the fault that ends the whole read, if any
// (see stray, markup, unclosed and subsetEnd). Only the internal subset itself
// ends at a ']', and it must: text that runs out before one is no [28]
// doctypedecl, unless a construct left open there declined it first (see
// unclosed).
func (sc *subsetScan) scan(s string) error {
	for s != "" {
		switch {
		case s[0] == ']' && sc.depth == 0:
			return sc.subsetEnd(s[1:])
		case s[0] == '<':
			after, closed, err := sc.markup(s)
			if err != nil {
				return err
			}
			if !closed {
				return sc.unclosed(s)
			}
			s = after
		case s[0] == '%':
			name, after, ok := peReference(s[1:])
			if !ok {
				return sc.stray(s)
			}
			if err := sc.expand(name); err != nil {
				return err
			}
			s = after
		case strings.ContainsRune(declSpace, rune(s[0])):
			s = s[1:]
		default:
			return sc.stray(s)
		}
	}
	if sc.depth > 0 {
		return nil
	}
	return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "DOCTYPE internal subset is closed by no ']' before the directive's '>' (XML 1.0 [28] doctypedecl)")
}

// markup reads the comment, processing instruction or markup declaration s
// opens with its '<', returning what follows it and whether it closes in s,
// or the fault that ends the whole read. A conditional section in replacement
// text is declined and ends that text, reported as closed with nothing after
// it; one in the internal subset itself, which [28b] intSubset does not admit,
// is stray, as is a '<' opening none of these.
func (sc *subsetScan) markup(s string) (after string, closed bool, err error) {
	switch {
	case strings.HasPrefix(s, "<!--"):
		return sc.comment(s)
	case strings.HasPrefix(s, "<?"):
		return sc.pi(s)
	case strings.HasPrefix(s, "<![") && sc.depth > 0:
		sc.decline()
		return "", true, nil
	case strings.HasPrefix(s, "<!ENTITY"):
		body, after, closed, err := sc.markupDecl(s[len("<!ENTITY"):])
		if err != nil || !closed {
			return "", closed, err
		}
		if err := sc.noPEReference(body); err != nil {
			return "", true, err
		}
		if err := sc.entityValuePercent(body); err != nil {
			return "", true, err
		}
		if !sc.checkOnly {
			sc.declare(body)
		}
		return after, true, nil
	case strings.HasPrefix(s, "<!NOTATION"):
		return sc.notation(s)
	case strings.HasPrefix(s, "<!ELEMENT"), strings.HasPrefix(s, "<!ATTLIST"):
		body, after, closed, err := sc.markupDecl(s[len("<!"):])
		if err != nil || !closed {
			return "", closed, err
		}
		return after, true, sc.noPEReference(body)
	}
	return "", true, sc.stray(s)
}

// comment reads the comment s opens: XML 1.0 [15] Comment, in which "--" may
// stand only as part of the closing "-->". Parameter-entity references are
// not recognized in it.
func (sc *subsetScan) comment(s string) (after string, closed bool, err error) {
	body := s[len("<!--"):]
	end := strings.Index(body, "--")
	if end < 0 || end+len("--") == len(body) {
		return "", false, nil
	}
	if body[end+len("--")] != '>' {
		return "", true, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds a comment with \"--\" inside it (XML 1.0 [15] Comment)", sc.where())
	}
	return body[end+len("-->"):], true, nil
}

// pi reads the processing instruction s opens: XML 1.0 [16] PI, whose target
// — the text up to its first S, or all of it — must be a Name other than
// "xml" in any case ([17] PITarget). Parameter-entity references are not
// recognized in it.
func (sc *subsetScan) pi(s string) (after string, closed bool, err error) {
	body := s[len("<?"):]
	end := strings.Index(body, "?>")
	if end < 0 {
		return "", false, nil
	}
	target := body[:end]
	if sp := strings.IndexAny(target, declSpace); sp >= 0 {
		target = target[:sp]
	}
	if !isName(target) || strings.EqualFold(target, "xml") {
		return "", true, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds a processing instruction whose target %q is no PITarget (XML 1.0 [16] PI, [17] PITarget)", sc.where(), target)
	}
	return body[end+len("?>"):], true, nil
}

// notation reads the notation declaration s opens, which must match XML 1.0
// [82] NotationDecl: '<!NOTATION' S Name S, then an ExternalID ([75]) or a
// PublicID ([83]), then S? '>'.
func (sc *subsetScan) notation(s string) (after string, closed bool, err error) {
	body, after, closed, err := sc.markupDecl(s[len("<!NOTATION"):])
	if err != nil || !closed {
		return "", closed, err
	}
	if err := sc.noPEReference(body); err != nil {
		return "", true, err
	}
	if !notationDecl(body) {
		return "", true, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds a <!NOTATION> declaration that is no Name followed by an ExternalID or PublicID (XML 1.0 [82] NotationDecl, [75] ExternalID, [83] PublicID, [12] PubidLiteral)", sc.where())
	}
	return after, true, nil
}

// notationDecl reports whether body, a <!NOTATION> declaration's text after
// its keyword, reads by position as S Name S, then 'SYSTEM' S SystemLiteral,
// 'PUBLIC' S PubidLiteral S SystemLiteral or 'PUBLIC' S PubidLiteral, then S?
// (XML 1.0 [82], [75], [83], [11], [12]). Every token ends only at white space,
// so a literal run on into a keyword or another literal is no literal.
func notationDecl(body string) bool {
	if body == "" || !strings.ContainsRune(declSpace, rune(body[0])) {
		return false
	}
	toks := splitDecl(body, true)
	if len(toks) < 3 || !isName(toks[0]) {
		return false
	}
	switch toks[1] {
	case "SYSTEM":
		return len(toks) == 3 && isLiteral(toks[2])
	case "PUBLIC":
		return isPubidLiteral(toks[2]) && (len(toks) == 3 || len(toks) == 4 && isLiteral(toks[3]))
	}
	return false
}

// isPubidLiteral reports whether t, one of splitDecl's tokens, is a quoted
// literal holding PubidChars only (XML 1.0 [12] PubidLiteral, [13] PubidChar).
// isLiteral keeps a "'"-quoted literal free of "'".
func isPubidLiteral(t string) bool {
	if !isLiteral(t) {
		return false
	}
	for _, c := range []byte(t[1 : len(t)-1]) {
		alnum := 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9'
		if !alnum && strings.IndexByte(pubidOther, c) < 0 {
			return false
		}
	}
	return true
}

// pubidOther is every PubidChar that is not an ASCII letter or digit (XML 1.0
// [13] PubidChar).
const pubidOther = " \r\n-'()+,./:=?;!*#@$_%"

// noPEReference returns the fault of a parameter-entity reference standing
// outside the literals of body, a markup declaration's text: in the internal
// subset one must not occur within a markup declaration (XML 1.0 WFC: PEs in
// Internal Subset), and every parameter entity this scan reads is internal.
// The one literal that recognizes such a reference is entityValuePercent's.
func (sc *subsetScan) noPEReference(body string) error {
	for {
		pct := outsideQuotes(body, "%")
		if pct < 0 {
			return nil
		}
		body = body[pct+1:]
		if name, _, ok := peReference(body); ok {
			return sc.peInMarkup(name)
		}
	}
}

// entityValuePercent returns the fault of a '%' in the EntityValue literal of
// body, an <!ENTITY> declaration's text after its keyword: the literal its
// name is followed by. XML 1.0 §2.8 recognizes parameter-entity references in
// an entity value literal, alone among literals, so a PEReference there breaks
// WFC: PEs in Internal Subset, and any other '%' is no [9] EntityValue. An
// ExternalID's SystemLiteral and PubidLiteral recognize none and are not read.
func (sc *subsetScan) entityValuePercent(body string) error {
	toks := declTokens(body)
	if len(toks) > 0 && toks[0] == "%" {
		toks = toks[1:]
	}
	if len(toks) < 2 || toks[1][0] != '"' && toks[1][0] != '\'' {
		return nil
	}
	lit := toks[1][1:]
	pct := strings.IndexByte(lit, '%')
	if pct < 0 {
		return nil
	}
	if name, _, ok := peReference(lit[pct+1:]); ok {
		return sc.peInMarkup(name)
	}
	return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an entity value literal with a '%%' that opens no PEReference (XML 1.0 [9] EntityValue, [69] PEReference)", sc.where())
}

// peInMarkup is the fault of the parameter-entity reference to name standing
// inside a markup declaration (XML 1.0 WFC: PEs in Internal Subset).
func (sc *subsetScan) peInMarkup(name string) error {
	return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds the parameter-entity reference %%%s; inside a markup declaration (XML 1.0 WFC: PEs in Internal Subset)", sc.where(), name)
}

// where names the text the scan is reading, for a fault message.
func (sc *subsetScan) where() string {
	if sc.depth > 0 {
		return "replacement text of a parameter entity referenced between DOCTYPE declarations"
	}
	return "DOCTYPE internal subset"
}

// stray is the fault of s, text standing between declarations that opens no
// markup declaration, comment, processing instruction or PEReference and is
// not S. In the internal subset it matches no alternative of XML 1.0 [28b]
// intSubset or [28a] DeclSep; in a parameter entity's replacement text it
// breaks WFC: PE Between Declarations, which requires that text to match [31]
// extSubsetDecl. A '%' run that is no '%' Name ';' ([69] PEReference) is
// stray, and so is a ']' in replacement text, and so is a '<' opening no
// comment, processing instruction or markup declaration, a conditional
// section in the internal subset itself among them. Either way the document is
// not well-formed, a fatal error (XML 1.0 §1.2), so the fault ends the whole
// read where a decline would only end processing.
func (sc *subsetScan) stray(s string) error {
	rule := "XML 1.0 [28b] intSubset, [28a] DeclSep"
	if sc.depth > 0 {
		rule = "XML 1.0 WFC: PE Between Declarations, [31] extSubsetDecl"
	}
	if s[0] == '%' {
		rule += ", [69] PEReference"
	}
	return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds %q between declarations, which is no markup declaration, PEReference or S (%s)", sc.where(), excerpt(s), rule)
}

// subsetEnd checks what follows the ']' closing the internal subset, up to the
// directive's closing '>': S alone may stand there (XML 1.0 [28]
// doctypedecl), and any other text is a fault.
func (sc *subsetScan) subsetEnd(after string) error {
	rest := strings.TrimLeft(after, declSpace)
	if rest == "" {
		return nil
	}
	return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "DOCTYPE holds %q between its internal subset's ']' and '>', where only S may stand (XML 1.0 [28] doctypedecl)", excerpt(rest))
}

// excerpt is the run non-empty s opens, up to the next S, '<' or ']' after its
// first byte and at most maxExcerpt runes, for a fault message to quote.
func excerpt(s string) string {
	if end := strings.IndexAny(s[1:], declSpace+"<]"); end >= 0 {
		s = s[:end+1]
	}
	runes := 0
	for i := range s {
		if runes == maxExcerpt {
			return s[:i]
		}
		runes++
	}
	return s
}

// maxExcerpt bounds the runes of stray text a fault message quotes.
const maxExcerpt = 32

// peReference reads the Name and ';' of a PEReference whose '%' has just been
// consumed, returning what follows it. It reports false unless a ';' closes a
// run that is a Name (XML 1.0 [69] PEReference, [5] Name).
func peReference(s string) (name, after string, ok bool) {
	end := strings.IndexByte(s, ';')
	if end < 0 || !isName(s[:end]) {
		return "", "", false
	}
	return s[:end], s[end+1:], true
}

// expand reads the replacement text of the parameter entity a reference
// names, or declines the reference when that text cannot be read within
// bounds. Once checkOnly, it expands nothing: the reference follows one that
// was not read (XML 1.0 §5.1).
func (sc *subsetScan) expand(name string) error {
	if sc.checkOnly {
		return nil
	}
	pe, declared := sc.pes[name]
	cost := len(pe.text) + len("  ")
	if !declared || !pe.readable || sc.depth == maxPEDepth || sc.spent+cost > maxPEExpansion {
		sc.decline()
		return nil
	}
	sc.spent += cost
	sc.depth++
	err := sc.scan(" " + pe.text + " ")
	sc.depth--
	return err
}

// unclosed ends the read of s at the comment, processing instruction or markup
// declaration s opens and never closes. In replacement text that is a fault: a
// parameter entity referenced between declarations must expand to complete
// markup declarations (XML 1.0 WFC: PE Between Declarations, [31]
// extSubsetDecl).
//
// GAP(xml): in the internal subset itself the construct is declined, and the
// ']' the subset then never reaches is no fault: encoding/xml ends the
// directive at a '>' inside a processing instruction, so the open construct
// may be the decoder's cut rather than the document's. Tracked by #2226.
func (sc *subsetScan) unclosed(s string) error {
	if sc.depth > 0 {
		return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s leaves %q open where it ends (XML 1.0 WFC: PE Between Declarations, [31] extSubsetDecl)", sc.where(), excerpt(s))
	}
	sc.decline()
	return nil
}

// decline records a reference or construct the scan does not read. Unless
// standalone="yes", the scan reads on only to check well-formedness: XML 1.0
// §5.1 forbids processing the entity declarations after it.
func (sc *subsetScan) decline() {
	sc.unread = true
	if !sc.standalone {
		sc.checkOnly = true
	}
}

// declare records one <!ENTITY ...> body: a parameter entity into pes, a
// general one into decls. The first declaration of a name binds (XML 1.0
// §4.2); for a general entity that is the Reader's to apply.
func (sc *subsetScan) declare(body string) {
	name, pe, isPE := paramEntityOf(body)
	if !isPE {
		if d, general := entityDeclOf(body); general {
			sc.decls = append(sc.decls, d)
		}
		return
	}
	if _, bound := sc.pes[name]; bound {
		return
	}
	if sc.pes == nil {
		sc.pes = make(map[string]entityValue)
	}
	sc.pes[name] = pe
}

// paramEntityOf reads one <!ENTITY ...> body as a parameter entity
// declaration, <!ENTITY % name PEDef>, reporting false for any other body —
// one whose name is not a Name among them, which is no PEDecl (XML 1.0 [72],
// [5]) and declares nothing. The entity is readable only when its PEDef is
// one EntityValue literal whose replacement text can be built (see
// replacementText): an ExternalID, or anything malformed, declares an entity
// this scan never reads.
func paramEntityOf(body string) (string, entityValue, bool) {
	if body == "" || !strings.ContainsRune(declSpace, rune(body[0])) {
		return "", entityValue{}, false
	}
	toks := declTokens(body)
	if len(toks) < 3 || toks[0] != "%" || !isName(toks[1]) {
		return "", entityValue{}, false
	}
	if len(toks) != 3 {
		return toks[1], entityValue{}, true
	}
	return toks[1], literalValue(toks[2]), true
}

// literalValue is the value a definition token gives an entity: the
// replacement text it builds when it is one EntityValue literal (see
// replacementText), and unreadable when it is any other token.
func literalValue(tok string) entityValue {
	if !isLiteral(tok) {
		return entityValue{}
	}
	text, ok := replacementText(tok[1 : len(tok)-1])
	return entityValue{text: text, readable: ok}
}

// replacementText builds an internal entity's replacement text from its
// literal entity value (XML 1.0 §4.5): each line end the literal spells is
// normalized to #xA (§2.11), each character reference is replaced by the
// character it names, and a general entity reference is left as it stands
// (§4.4.7, bypassed), for the reader to expand where the entity is included.
// It reports false for a character reference that is malformed or names no
// Char (WFC Legal Character). The literal holds no '%': markup faults the
// declaration before it is read (see entityValuePercent).
func replacementText(lit string) (string, bool) {
	var b strings.Builder
	for {
		amp := strings.IndexByte(lit, '&')
		if amp < 0 {
			b.WriteString(lineEnds.Replace(lit))
			return b.String(), true
		}
		b.WriteString(lineEnds.Replace(lit[:amp]))
		lit = lit[amp:]
		if !strings.HasPrefix(lit, "&#") {
			b.WriteByte('&')
			lit = lit[1:]
			continue
		}
		end := strings.IndexByte(lit, ';')
		if end < 0 {
			return "", false
		}
		r, ok := charRef(lit[len("&#"):end])
		if !ok {
			return "", false
		}
		b.WriteRune(r)
		lit = lit[end+1:]
	}
}

// lineEnds normalizes XML 1.0 §2.11's line ends, #xD#xA and a lone #xD, to
// #xA.
var lineEnds = strings.NewReplacer("\r\n", "\n", "\r", "\n")

// charRef reads the digits of a character reference — what XML 1.0 CharRef
// holds between "&#" and ";" — reporting false unless they name a Char (XML
// 1.0 §2.2).
func charRef(digits string) (rune, bool) {
	base := 10
	if hex, ok := strings.CutPrefix(digits, "x"); ok {
		digits, base = hex, 16
	}
	n, err := strconv.ParseUint(digits, base, 32)
	if err != nil {
		return 0, false
	}
	r := rune(n)
	return r, xmlchar.IsChar(r)
}

// outsideQuotes reports the index of the first byte of set in s that is not
// inside a quoted literal, or -1.
func outsideQuotes(s, set string) int {
	var quote byte
	for i := 0; i < len(s); i++ {
		switch {
		case quote != 0:
			if s[i] == quote {
				quote = 0
			}
		case s[i] == '"' || s[i] == '\'':
			quote = s[i]
		case strings.IndexByte(set, s[i]) >= 0:
			return i
		}
	}
	return -1
}

// markupDecl splits s at the '>' closing the markup declaration it is inside
// of, returning the declaration's body and what follows it, and reports
// whether that '>' is in s. A declaration with no closing '>' runs to the end
// of s. A '<' outside the declaration's literals is a fault: no markup
// declaration admits one (XML 1.0 [45] elementdecl, [52] AttlistDecl, [70]
// EntityDecl, [82] NotationDecl). A comment there is one, and is not skipped:
// a quote inside it would open a literal the declaration never closes.
func (sc *subsetScan) markupDecl(s string) (body, after string, closed bool, err error) {
	end := outsideQuotes(s, "<>")
	if end < 0 {
		return s, "", false, nil
	}
	if s[end] == '<' {
		return "", "", true, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds %q inside a markup declaration, outside its literals, where no markup declaration admits a '<' (XML 1.0 [45] elementdecl, [52] AttlistDecl, [70] EntityDecl, [82] NotationDecl)", sc.where(), excerpt(s[end:]))
	}
	return s[:end], s[end+1:], true, nil
}

// entityDeclOf reads one <!ENTITY ...> body. It reports false for a parameter
// entity declaration (<!ENTITY % name ...>), which declares no general entity
// at all, for a keyword run on into the name with no white space between, for
// a body too short to name one, and for a name that is not a Name — `1x`,
// `a&b`, or `a"b"` with a literal run on into it — which is no GEDecl (XML 1.0
// [71], [5]) and declares nothing, internal or unparsed. A general entity is
// unparsed only when its definition is exactly an ExternalID followed by an
// NDataDecl (XML 1.0 EntityDef); any other definition, a malformed one
// included, is a parsed entity. A parsed entity is internal, with a readable
// value, only when its definition is exactly one EntityValue literal whose
// replacement text builds.
func entityDeclOf(body string) (entityDecl, bool) {
	if body == "" || !strings.ContainsRune(declSpace, rune(body[0])) {
		return entityDecl{}, false
	}
	toks := entityDefTokens(body)
	if len(toks) < 2 || !isName(toks[0]) {
		return entityDecl{}, false
	}
	d := entityDecl{name: toks[0], unparsed: unparsedDef(toks[1:])}
	if len(toks) == 2 {
		d.value = literalValue(toks[1])
	}
	return d, true
}

// unparsedDef reports whether def, the tokens of an entity definition, reads
// by position as `SYSTEM lit NDATA Name` or `PUBLIC lit lit NDATA Name` and
// nothing more: XML 1.0's ExternalID followed by an NDataDecl.
func unparsedDef(def []string) bool {
	var lits int
	switch def[0] {
	case "SYSTEM":
		lits = 1
	case "PUBLIC":
		lits = 2
	default:
		return false
	}
	if len(def) != 1+lits+2 {
		return false
	}
	for _, t := range def[1 : 1+lits] {
		if !isLiteral(t) {
			return false
		}
	}
	return def[1+lits] == "NDATA" && isName(def[2+lits])
}

// isName reports whether t, a declaration token, is an XML 1.0 Name
// (production [5]): it is not empty, its first character is a NameStartChar
// ([4]) and every later one a NameChar ([4a]), both internal/xmlname's tables.
// It checks the DOCTYPE's document type name, an entity's declared name, the
// name a PEReference between declarations carries and the notation name an
// NDataDecl closes on, rejecting `1x`, `g&h` and `a×b` (U+00D7 is in neither
// production).
func isName(t string) bool {
	if t == "" {
		return false
	}
	for i, r := range t {
		if !unicode.Is(xmlname.NameStartChar, r) && (i == 0 || !unicode.Is(xmlname.NameCharExtra, r)) {
			return false
		}
	}
	return true
}

// isLiteral reports whether t, one of splitDecl's tokens, is a closed quoted
// literal and nothing more: its quote recurs only as its last character, so a
// literal with text run on after it — `"x"y""`, an entityDefTokens token — is
// none.
func isLiteral(t string) bool {
	return len(t) >= 2 && (t[0] == '"' || t[0] == '\'') && strings.IndexByte(t[1:], t[0]) == len(t)-2
}

// declSpace is XML 1.0's S production: the white space that separates the
// tokens of a markup declaration, and the only character data that may follow
// the document element (see trailerFault).
const declSpace = " \t\r\n"

// declTokens splits a markup declaration body on white space, keeping each
// quoted literal whole and WITH its quotes, so a literal never compares equal
// to a keyword.
func declTokens(s string) []string {
	return splitDecl(s, false)
}

// entityDefTokens is declTokens for a general entity's <!ENTITY> body, except
// that every token ends only at white space. Text run on after a literal's
// closing quote with no S between — `"x"NDATA`, which XML 1.0 NDataDecl [76]
// forbids — stays in the literal's token, which isLiteral then refuses; a
// literal run on after a keyword — `SYSTEM"x"`, which ExternalID [75] forbids
// — stays in the keyword's token, which unparsedDef then refuses. The DOCTYPE
// header's ExternalID (hasExternalID) and a parameter entity's body keep
// declTokens' split, where every token ends at a quote as well as at white
// space; the header's name does not (see doctypeName).
func entityDefTokens(s string) []string {
	return splitDecl(s, true)
}

// splitDecl splits s as declTokens does, every token running on to the next
// white space when runOn is set.
func splitDecl(s string, runOn bool) []string {
	var toks []string
	for {
		s = strings.TrimLeft(s, declSpace)
		if s == "" {
			return toks
		}
		if q := s[0]; q == '"' || q == '\'' {
			end := strings.IndexByte(s[1:], q)
			if end < 0 {
				return append(toks, s)
			}
			end += 2
			if runOn {
				end = runEnd(s, end)
			}
			toks = append(toks, s[:end])
			s = s[end:]
			continue
		}
		stops := declSpace + `"'`
		if runOn {
			stops = declSpace
		}
		end := strings.IndexAny(s, stops)
		if end < 0 {
			return append(toks, s)
		}
		toks = append(toks, s[:end])
		s = s[end:]
	}
}

// runEnd reports the index of the first white space in s at or after from, or
// len(s).
func runEnd(s string, from int) int {
	end := strings.IndexAny(s[from:], declSpace)
	if end < 0 {
		return len(s)
	}
	return from + end
}
