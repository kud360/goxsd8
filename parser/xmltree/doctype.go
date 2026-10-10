package xmltree

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/kud360/goxsd8/internal/xmlchar"
	"github.com/kud360/goxsd8/internal/xmlname"
	"github.com/kud360/goxsd8/xsderr"
)

// entityDecl is one entity declaration of a DOCTYPE's internal subset, read
// by readEntityDecl: the entity's name, whether it is UNPARSED — an external
// general entity carrying an NDATA notation name, which is what an ·ENTITY
// value· must name (Structures §3.16.4 key-vde) — and, for an internal entity,
// its replacement text, which a reference to it includes (XML 1.0 §4.4.2,
// §4.4.5, §4.4.8). value is unreadable for an external entity, unparsed or
// not. inPE reports that the declaration stands in a parameter entity's
// replacement text, where XML 1.0 WFC: Entity Declared does not count it.
type entityDecl struct {
	name     string
	unparsed bool
	inPE     bool
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

// doctypeEntities reads the general entity declarations and the <!ATTLIST>
// attribute definitions of one directive's DOCTYPE, each in document order,
// and reports whether some declaration went unread: the inverse of the
// document's [all declarations processed] (XML Infoset §2.1). directive is
// the directive's source between its "<!" and its closing '>', comments
// included (see Reader.declareEntities). standalone is the XML declaration's
// standalone="yes" (XML 1.0 §2.9 SDDecl). A directive that is no doctypedecl
// — one whose keyword is not "DOCTYPE", in that case, or is run on into the
// text after it with no S between, `<!DOCTYPEr>` — and a DOCTYPE whose
// document type name is missing or is not a Name (XML 1.0 [22] prolog, [27]
// Misc, [28] doctypedecl, [5] Name) are not well-formed: a RuleXMLWellFormed
// fault at loc, the directive's start.
//
// The external DTD subset is never read, by design (XML 1.0 §5.1 and §5.2
// oblige a non-validating processor to read the document entity alone;
// ruled on #1668): an ExternalID in the DOCTYPE header reports unread and
// nothing more. The internal subset is read in full, and a parameter-entity
// reference between its declarations (DeclSep) is expanded in place when the
// entity is internal and already declared: its replacement text is the
// literal with each character reference replaced (§4.5), enlarged by one space
// either side (§4.4.8). A reference it does not read — to an external or
// undeclared parameter entity, or past maxPEDepth or maxPEExpansion, which a
// recursive reference (WFC No Recursion) always reaches — reports unread.
// Unless standalone, the rest of the internal subset is then only checked for
// well-formedness, which §5.1 requires of the entire internal subset: it binds
// no parameter entity, records no general entity or attribute definition and
// expands no parameter-entity reference, since §5.1 forbids processing an
// entity or attribute-list declaration that follows a reference to a parameter
// entity that is not read, except when standalone="yes". A conditional section
// in replacement text, which this scan does not read, is declined the same way
// and ends the text it appears in.
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
// (WFC: PEs in Internal Subset), and an internal subset or replacement text
// that ends inside a comment, processing instruction or markup declaration
// ([28b] intSubset, WFC: PE Between Declarations) are faults too. So is an
// entity declaration that is no [70] EntityDecl (see readEntityDecl): no S
// after its keyword, a declared name that is no Name ([71] GEDecl, [72]
// PEDecl, [5]), no definition, one that is neither an EntityValue nor an
// ExternalID ([73] EntityDef, [74] PEDef, [75] ExternalID, [11]
// SystemLiteral, [12] PubidLiteral), a token after an EntityValue, an
// NDataDecl in a parameter entity's PEDef ([74]), anything but one NDataDecl
// after a general entity's ExternalID ([76]), and an entity value literal
// with a '%' or '&' that opens no PEReference or Reference ([9] EntityValue,
// [66]–[69]) or a character reference naming no Char (WFC: Legal Character),
// whether or not the entity is ever referenced. So is an element type
// declaration that is no [45] elementdecl (see readElementDecl) — a missing
// S, a name that is no Name, a contentspec that is not 'EMPTY', 'ANY', a [51]
// Mixed or a [47] children model ([46]–[50]) — and an attribute-list
// declaration that is no [52] AttlistDecl (see readAttlistDecl) — a missing S,
// a name that is no Name, an AttType that is no [54]–[59] AttType, a missing
// [60] DefaultDecl, and a default value that is no [10] AttValue, a '<' in it
// among them. A validity constraint on either declaration is no fault. These
// checks run on after a declined reference, as §5.1 requires. Once the whole
// subset reads without one of them, an entity reference in a default value is
// checked against the entity it names, a fault too: WFC: Entity Declared,
// Parsed Entity, No Recursion, No External Entity References and No < in
// Attribute Values, and a '&' in replacement text that begins no Reference
// (see defaultsFault).
func doctypeEntities(directive string, standalone bool, loc xsderr.Loc) (decls []entityDecl, atts []attDecl, unread bool, err error) {
	rest, ok := strings.CutPrefix(directive, "DOCTYPE")
	if _, spaced := cutSpace(rest); !ok || !spaced && rest != "" {
		return nil, nil, false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "directive %q is no doctypedecl, '<!DOCTYPE' S Name, and no other directive may stand outside the document element (XML 1.0 [22] prolog, [27] Misc, [28] doctypedecl)", "<!"+excerpt(directive))
	}
	header := rest
	open := outsideQuotes(rest, "[<")
	if open >= 0 {
		header = rest[:open]
	}
	if name := doctypeName(header); !isName(name) {
		return nil, nil, false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "DOCTYPE document type name %q is not a Name (XML 1.0 [28] doctypedecl, [5] Name)", name)
	}
	if open >= 0 && rest[open] == '<' {
		return nil, nil, false, xsderr.New(xsderr.RuleXMLWellFormed, loc, "DOCTYPE holds %q before its internal subset, where only S, its name and an ExternalID may stand (XML 1.0 [28] doctypedecl)", excerpt(rest[open:]))
	}
	sc := subsetScan{standalone: standalone, loc: loc, unread: hasExternalID(header)}
	if open < 0 {
		return nil, nil, sc.unread, nil
	}
	if err := sc.scan(rest[open+1:]); err != nil {
		return nil, nil, false, err
	}
	if err := sc.defaultsFault(); err != nil {
		return nil, nil, false, err
	}
	return sc.decls, sc.atts, sc.unread, nil
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
// declarations read, atts the attribute definitions of the <!ATTLIST>
// declarations read, and unread records that some declaration was not.
// checkOnly records that a declined reference has ended processing, outside a
// standalone document (XML 1.0 §5.1): the scan reads on only to check
// well-formedness, recording no declaration in decls or atts. defaults
// collects the <!ATTLIST> default values read, in document order, for
// defaultsFault. loc is the directive's start, where every fault the scan
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
	atts       []attDecl
	defaults   []attDefault
}

// attDecl is one attribute definition an <!ATTLIST> declaration gives, read by
// attDef: the element type and attribute names as the declaration spells them,
// prefix included, and whether its AttType is other than [55] CDATA — a [56]
// TokenizedType or a [57] EnumeratedType, which XML 1.0 §3.3.3 has the
// normalized value of trimmed and collapsed (see collapseSpace).
type attDecl struct {
	elem, name string
	tokenized  bool
}

// attDefault is one default value an <!ATTLIST> declaration gives, read by
// defaultDecl: about describes it for a fault message, lit is its text between
// the quotes, declared counts the general entity declarations recorded before
// it, and inPE reports that it stands in a parameter entity's replacement
// text.
type attDefault struct {
	about    string
	lit      string
	declared int
	inPE     bool
}

// entityValue is one entity's replacement text, and whether it has one the
// reader can read — false for an external entity.
type entityValue struct {
	text     string
	readable bool
}

// scan reads s — the internal subset, or a parameter entity's enlarged
// replacement text — and returns the fault that ends the whole read, if any
// (see stray, markup, unclosed and subsetEnd). Only the internal subset itself
// ends at a ']', and it must: text that runs out before one is no [28]
// doctypedecl.
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
		d, pe, err := sc.readEntityDecl(body)
		if err != nil {
			return "", true, err
		}
		if !sc.checkOnly {
			sc.declare(d, pe)
		}
		return after, true, nil
	case strings.HasPrefix(s, "<!NOTATION"):
		return sc.notation(s)
	case strings.HasPrefix(s, "<!ELEMENT"):
		body, after, closed, err := sc.markupDecl(s[len("<!ELEMENT"):])
		if err != nil || !closed {
			return "", closed, err
		}
		if err := sc.noPEReference(body); err != nil {
			return "", true, err
		}
		return after, true, sc.readElementDecl(body)
	case strings.HasPrefix(s, "<!ATTLIST"):
		body, after, closed, err := sc.markupDecl(s[len("<!ATTLIST"):])
		if err != nil || !closed {
			return "", closed, err
		}
		if err := sc.noPEReference(body); err != nil {
			return "", true, err
		}
		return after, true, sc.readAttlistDecl(body)
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

// readElementDecl reads body, an <!ELEMENT> declaration's text after its
// keyword, against XML 1.0 [45] elementdecl, S Name S contentspec S?,
// returning the fault of a body that does not match it. contentspec ([46]) is
// 'EMPTY', 'ANY', Mixed ([51], see mixed) or children ([47], see children),
// its keywords in upper case. The validity constraints on the declaration —
// one declaration per element type (VC: Unique Element Type Declaration), no
// name repeated in Mixed (VC: No Duplicate Types) and proper group/PE nesting
// — bind a validating processor only, and none is checked.
func (sc *subsetScan) readElementDecl(body string) error {
	rest, ok := cutSpace(body)
	if !ok {
		return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ELEMENT> declaration with no S after its keyword (XML 1.0 [45] elementdecl)", sc.where())
	}
	name, rest := tokenRun(rest)
	if !isName(name) {
		return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ELEMENT> declaration whose element type name %q is not a Name (XML 1.0 [45] elementdecl, [5] Name)", sc.where(), name)
	}
	spec, ok := cutSpace(rest)
	if !ok {
		return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ELEMENT> declaration of %q with %q where S and a contentspec must stand (XML 1.0 [45] elementdecl)", sc.where(), name, excerpt(rest))
	}
	spec = strings.TrimRight(spec, declSpace)
	if spec == "EMPTY" || spec == "ANY" {
		return nil
	}
	tail, ok, rule := "", false, "[46] contentspec"
	switch {
	case strings.HasPrefix(spec, "(") && strings.HasPrefix(strings.TrimLeft(spec[1:], declSpace), "#PCDATA"):
		tail, ok = mixed(spec)
		rule += ", [51] Mixed"
	case strings.HasPrefix(spec, "("):
		tail, ok = children(spec)
		rule += ", [47] children, [48] cp, [49] choice, [50] seq"
	}
	if ok && tail == "" {
		return nil
	}
	return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ELEMENT> declaration of %q whose content specification %q is not 'EMPTY', 'ANY', Mixed or children (XML 1.0 %s)", sc.where(), name, spec, rule)
}

// mixed reads spec, a content specification that opens with '(' S? '#PCDATA',
// against XML 1.0 [51] Mixed, returning what follows it, or reports false:
// '(' S? '#PCDATA' (S? '|' S? Name)* S? ')*', or '(' S? '#PCDATA' S? ')'. A
// Mixed naming an element type closes with ")*", no S inside it, and
// "(#PCDATA)" takes no '?' or '+'.
func mixed(spec string) (rest string, ok bool) {
	s := strings.TrimPrefix(strings.TrimLeft(spec[1:], declSpace), "#PCDATA")
	names := false
	for {
		after, bar := strings.CutPrefix(strings.TrimLeft(s, declSpace), "|")
		if !bar {
			break
		}
		name, after := tokenRun(strings.TrimLeft(after, declSpace))
		if !isName(name) {
			return "", false
		}
		s, names = after, true
	}
	s = strings.TrimLeft(s, declSpace)
	if rest, ok := strings.CutPrefix(s, ")*"); ok {
		return rest, true
	}
	if rest, ok := strings.CutPrefix(s, ")"); ok && !names {
		return rest, true
	}
	return "", false
}

// children reads spec, a content specification that opens with '(', against
// XML 1.0 [47] children, (choice | seq) ('?' | '*' | '+')?, returning what
// follows it, or reports false. A content particle ([48] cp) is a Name or a
// nested choice or seq, its occurrence indicator, if any, right after it with
// no S between. A group's first separator decides it: a choice ([49]) joins
// its cps with '|' alone, a seq ([50]) with ',' alone or holds a single cp, so
// `(a|b,c)` and `()` are neither. S may stand around each separator and
// parenthesis. seps keeps one entry per open group, the separator it uses or
// 0 before its first, so nesting depth costs no recursion.
func children(spec string) (rest string, ok bool) {
	seps := []byte{0}
	s := spec[1:]
	for {
		s = strings.TrimLeft(s, declSpace)
		if after, open := strings.CutPrefix(s, "("); open {
			seps = append(seps, 0)
			s = after
			continue
		}
		name, after := tokenRun(s)
		if !isName(name) {
			return "", false
		}
		s = occurrence(after)
		for {
			s = strings.TrimLeft(s, declSpace)
			if s == "" {
				return "", false
			}
			top := len(seps) - 1
			if c := s[0]; c == '|' || c == ',' {
				if seps[top] != 0 && seps[top] != c {
					return "", false
				}
				seps[top] = c
				s = s[1:]
				break
			}
			if s[0] != ')' {
				return "", false
			}
			seps = seps[:top]
			s = occurrence(s[1:])
			if len(seps) == 0 {
				return s, true
			}
		}
	}
}

// occurrence is s past the occurrence indicator, '?', '*' or '+', it opens
// with, if any (XML 1.0 [47] children, [48] cp).
func occurrence(s string) string {
	if s != "" && strings.IndexByte("?*+", s[0]) >= 0 {
		return s[1:]
	}
	return s
}

// readAttlistDecl reads body, an <!ATTLIST> declaration's text after its
// keyword, against XML 1.0 [52] AttlistDecl, S Name AttDef* S?, returning the
// fault of a body that does not match it; one with no AttDef matches. Each
// AttDef is attDef's. The validity constraints on the declaration — duplicate
// enumeration tokens (VC: No Duplicate Tokens), more than one ID attribute
// (VC: One ID per Element Type), an ID attribute's default (VC: ID Attribute
// Default), an undeclared notation (VC: Notation Attributes) — bind a
// validating processor only, and none is checked.
func (sc *subsetScan) readAttlistDecl(body string) error {
	rest, ok := cutSpace(body)
	if !ok {
		return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration with no S after its keyword (XML 1.0 [52] AttlistDecl)", sc.where())
	}
	elem, rest := tokenRun(rest)
	if !isName(elem) {
		return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration whose element type name %q is not a Name (XML 1.0 [52] AttlistDecl, [5] Name)", sc.where(), elem)
	}
	for {
		def, spaced := cutSpace(rest)
		if def == "" {
			return nil
		}
		if !spaced {
			return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration of %q with %q where S and an AttDef, or '>', must stand (XML 1.0 [52] AttlistDecl, [53] AttDef)", sc.where(), elem, excerpt(def))
		}
		var err error
		if rest, err = sc.attDef(elem, def); err != nil {
			return err
		}
	}
}

// attDef reads the attribute definition def opens, in the <!ATTLIST>
// declaration of elem, after the S before it: XML 1.0 [53] AttDef, Name S
// AttType S DefaultDecl (see defaultDecl), returning what follows it. AttType
// ([54]) is 'CDATA' ([55] StringType), a [56] TokenizedType keyword, or an
// EnumeratedType ([57]): 'NOTATION' S and a parenthesized list of Names ([58]
// NotationType) or a parenthesized list of Nmtokens ([59] Enumeration, [7]
// Nmtoken), see enumeration. Keywords are in upper case. Unless checkOnly, it
// records the definition on atts (XML 1.0 §5.1).
func (sc *subsetScan) attDef(elem, def string) (rest string, err error) {
	name, rest := tokenRun(def)
	if !isName(name) {
		return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration of %q whose attribute name %q is not a Name (XML 1.0 [53] AttDef, [5] Name)", sc.where(), elem, name)
	}
	typ, ok := cutSpace(rest)
	kw, rest := tokenRun(typ)
	switch {
	case !ok: // no S before the AttType
	case kw == "NOTATION":
		list, spaced := cutSpace(rest)
		after, listed := enumeration(list, isName)
		if !spaced || !listed {
			return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration of %q whose attribute %q has %q where S and a parenthesized list of Names must follow 'NOTATION' (XML 1.0 [58] NotationType, [5] Name)", sc.where(), elem, name, excerpt(list))
		}
		rest = after
	case kw == "" && strings.HasPrefix(typ, "("):
		after, listed := enumeration(typ, isNmtoken)
		if !listed {
			return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration of %q whose attribute %q has the enumerated type %q, which is no Enumeration (XML 1.0 [59] Enumeration, [7] Nmtoken)", sc.where(), elem, name, excerpt(typ))
		}
		rest = after
	default:
		ok = isTypeKeyword(kw)
	}
	if !ok {
		return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration of %q whose attribute %q has %q where S and an AttType must stand (XML 1.0 [53] AttDef, [54] AttType, [55] StringType, [56] TokenizedType, [57] EnumeratedType)", sc.where(), elem, name, excerpt(typ))
	}
	if rest, err = sc.defaultDecl(elem, name, rest); err != nil {
		return "", err
	}
	if !sc.checkOnly {
		sc.atts = append(sc.atts, attDecl{elem: elem, name: name, tokenized: kw != "CDATA"})
	}
	return rest, nil
}

// isTypeKeyword reports whether kw is an AttType keyword that stands alone: a
// [55] StringType or a [56] TokenizedType.
func isTypeKeyword(kw string) bool {
	switch kw {
	case "CDATA", "ID", "IDREF", "IDREFS", "ENTITY", "ENTITIES", "NMTOKEN", "NMTOKENS":
		return true
	}
	return false
}

// enumeration reads the parenthesized list s opens, '(' S? tok (S? '|' S?
// tok)* S? ')', each tok a run valid accepts — a Name in a [58] NotationType,
// an Nmtoken in a [59] Enumeration — returning what follows it, or reports
// false.
func enumeration(s string, valid func(string) bool) (rest string, ok bool) {
	s, ok = strings.CutPrefix(s, "(")
	if !ok {
		return "", false
	}
	for {
		tok, after := tokenRun(strings.TrimLeft(s, declSpace))
		if !valid(tok) {
			return "", false
		}
		s = strings.TrimLeft(after, declSpace)
		if rest, ok := strings.CutPrefix(s, ")"); ok {
			return rest, true
		}
		if s, ok = strings.CutPrefix(s, "|"); !ok {
			return "", false
		}
	}
}

// defaultDecl reads the S and DefaultDecl that s, the text after attribute
// name's AttType in the <!ATTLIST> declaration of elem, opens with: XML 1.0
// [53] AttDef, [60] DefaultDecl, '#REQUIRED', '#IMPLIED', or an AttValue
// literal with '#FIXED' S before it or not, its text attValueFault's and,
// once the subset is read, defaultsFault's. It returns what follows the
// DefaultDecl.
func (sc *subsetScan) defaultDecl(elem, name, s string) (rest string, err error) {
	at, ok := cutSpace(s)
	kw, lit := tokenRun(at)
	switch kw {
	case "#REQUIRED", "#IMPLIED":
		if ok {
			return lit, nil
		}
	case "#FIXED":
		var spaced bool
		lit, spaced = cutSpace(lit)
		ok = ok && spaced
	case "": // an AttValue with no keyword before it
	default:
		ok = false
	}
	end := -1
	if ok && lit != "" && (lit[0] == '"' || lit[0] == '\'') {
		end = strings.IndexByte(lit[1:], lit[0])
	}
	if end < 0 {
		return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ATTLIST> declaration of %q whose attribute %q has %q where S and a DefaultDecl, '#REQUIRED', '#IMPLIED' or an AttValue with ('#FIXED' S)? before it, must stand (XML 1.0 [53] AttDef, [60] DefaultDecl)", sc.where(), elem, name, excerpt(at))
	}
	what := "an <!ATTLIST> declaration of " + strconv.Quote(elem) + " whose attribute " + strconv.Quote(name) + " has a default value"
	if err := sc.attValueFault(what, lit[1:1+end]); err != nil {
		return "", err
	}
	sc.defaults = append(sc.defaults, attDefault{about: sc.where() + " holds " + what, lit: lit[1 : 1+end], declared: len(sc.decls), inPE: sc.depth > 0})
	return lit[end+2:], nil
}

// attValueFault returns the fault of lit, the text between the quotes of the
// default value what describes, when it is no XML 1.0 [10] AttValue: it holds
// no '<', and every '&' in it opens a Reference, an EntityRef or a CharRef
// naming a Char (see reference). A '>' and a '%' are data in it: an AttValue
// recognizes no parameter-entity reference. The entity an EntityRef names is
// defaultsFault's to check, once every declaration is read.
func (sc *subsetScan) attValueFault(what, lit string) error {
	for {
		i := strings.IndexAny(lit, "<&")
		if i < 0 {
			return nil
		}
		if lit[i] == '<' {
			return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds %s with a '<' in it (XML 1.0 [10] AttValue)", sc.where(), what)
		}
		after, err := sc.reference(lit[i+1:], what, "[10] AttValue")
		if err != nil {
			return err
		}
		lit = after
	}
}

// defaultsFault returns the fault of the first entity reference, in document
// order, by which a default value of the subset's <!ATTLIST> declarations
// breaks a well-formedness constraint on the entity it names, or nil. These
// bind the declaration, whether or not its default is ever applied, and are
// checked once the whole subset is read, so a fault the scan finds anywhere in
// the subset is reported in place of one of these.
//
// WFC: Entity Declared binds a default value outside a parameter entity's
// replacement text, of a document that is standalone="yes" or whose DOCTYPE
// names no external subset and whose internal subset references no parameter
// entity. There each EntityRef ([68]) the default value holds directly must
// name a general entity declared before the <!ATTLIST>, and each one the
// replacement text of an entity it reaches holds must name one declared before
// the <!ATTLIST> or after it, an indirect reference to an entity declared after
// it being VC: Entity Declared's alone — unless the binding declaration (§4.2)
// of the entity whose replacement text holds it stands in a parameter entity's
// replacement text, where the reference occurs within that parameter entity
// and the constraint does not bind it, as Reader.withinPE reads a reference
// in included replacement text (#2365). Either way the constraint counts only
// a declaration outside every parameter entity's replacement text
// (entityDecl.inPE), and never asks a predefined name (amp, lt, gt, apos,
// quot; §4.6) to be declared. So under standalone="yes" an entity declared
// only in a parameter entity is declared for none of these references, and an
// entity declared outside every parameter entity whose replacement text
// references a name declared nowhere breaks the constraint once a default value
// references it, though its declaration alone does not. Where the constraint
// does not bind, a reference to a name declared nowhere is passed over.
//
// The entity a reference names, by its first declaration, which binds (§4.2),
// and every entity its replacement text references in turn, at any depth, must
// be parsed (WFC: Parsed Entity) and internal (WFC: No External Entity
// References), its replacement text must hold no '<' (WFC: No < in Attribute
// Values), and it must not reach itself (WFC: No Recursion; see entityGraph).
// A CharRef ([66]) names no entity: `&#60;` breaks none of these, and neither
// does an entity whose replacement text is `&#60;`. A '&' that replacement
// text holds where its literal spelled `&#38;` must begin a Reference there
// too (§4.4.5, [67]; see strayAmp): `&#38;#60;` is clean and `&#38;b` a fault,
// while `&#38;b;` references b. An entity declaration after a declined
// reference, outside a standalone document, is not recorded (§5.1), so a name
// first declared there is none this check knows: Entity Declared does not bind
// there, and the others are not checked.
func (sc *subsetScan) defaultsFault() error {
	if len(sc.defaults) == 0 {
		return nil
	}
	first, outside := make(map[string]int), make(map[string]int)
	for i, d := range sc.decls {
		if _, bound := first[d.name]; !bound {
			first[d.name] = i
		}
		if _, bound := outside[d.name]; !bound && !d.inPE {
			outside[d.name] = i
		}
	}
	// Every parameter-entity reference between declarations was declined,
	// setting unread, expanded, charging spent, or read once checkOnly, after
	// a decline; an external subset sets unread from the header.
	binds := sc.standalone || !sc.unread && sc.spent == 0
	graphs := [2]entityGraph{
		{loc: sc.loc, decls: sc.decls, first: first, outside: outside, state: make(map[string]walkState)},
		{loc: sc.loc, decls: sc.decls, first: first, outside: outside, declared: true, state: make(map[string]walkState)},
	}
	for _, d := range sc.defaults {
		g := &graphs[0]
		if binds && !d.inPE {
			g = &graphs[1]
		}
		if err := g.defaultFault(d); err != nil {
			return err
		}
	}
	return nil
}

// entityGraph walks the general entities a subset's default values reference,
// directly or through one another's replacement text, for defaultsFault. decls
// are the subset's general entity declarations, in document order, first the
// index of each name's first declaration among them, and outside the index of
// each name's first declaration outside every parameter entity's replacement
// text. declared reports that WFC: Entity Declared binds the default values
// this graph walks from; state records how far the walk from each entity has
// got, which a walk where the constraint binds may not take from one where it
// does not, so each graph keeps its own. The maps are lookup indexes only,
// never iterated. loc is the directive's start, where every fault is located.
type entityGraph struct {
	loc      xsderr.Loc
	decls    []entityDecl
	first    map[string]int
	outside  map[string]int
	declared bool
	state    map[string]walkState
}

// walkState is how far an entityGraph's walk from one entity has got.
type walkState uint8

const (
	// unwalked: the walk has not reached the entity.
	unwalked walkState = iota
	// walking: the entity is on the path from the default value being walked.
	walking
	// walked: the entity and every entity it reaches break none of the
	// constraints, so a later reference to it needs no walk.
	walked
)

// defaultFault returns the fault of the first entity reference d's default
// value holds directly that breaks a constraint defaultsFault names, or nil.
func (g *entityGraph) defaultFault(d attDefault) error {
	lit := d.lit
	for {
		name, after, ok := nextEntityRef(lit)
		if !ok {
			return nil
		}
		lit = after
		if _, builtin := predefined[name]; builtin {
			continue
		}
		i, counted := g.outside[name]
		if g.declared && (!counted || i >= d.declared) {
			return xsderr.New(xsderr.RuleXMLWellFormed, g.loc, "%s that references entity %s, which no general entity declaration before it, outside every parameter entity, declares (XML 1.0 WFC: Entity Declared)", d.about, name)
		}
		if _, known := g.first[name]; !known {
			continue
		}
		if err := g.fault(d.about, name); err != nil {
			return err
		}
	}
}

// walkFrame is one entity on the path of an entityGraph's walk: its name and
// the part of its replacement text the walk has yet to read.
type walkFrame struct {
	name string
	rest string
}

// fault walks from name, a declared general entity the default value about
// describes references, depth first through the replacement text of every
// entity it reaches, and returns the fault of the first it finds: an entity
// enter refuses, a reference to one already on the path (XML 1.0 WFC: No
// Recursion), or, where declared, a reference to a name no declaration outside
// every parameter entity declares, in the replacement text of an entity whose
// binding declaration stands outside every parameter entity too (WFC: Entity
// Declared; see defaultsFault). The path is a slice, not the call stack, so
// entities nested as deeply as the subset can declare them cost no recursion
// and no bound. A predefined name is passed over, and so, unless declared, is
// one declared nowhere.
func (g *entityGraph) fault(about, name string) error {
	if g.state[name] == walked {
		return nil
	}
	path, err := g.enter(nil, about, name)
	if err != nil {
		return err
	}
	for len(path) > 0 {
		top := &path[len(path)-1]
		ref, after, ok := nextEntityRef(top.rest)
		if !ok {
			g.state[top.name] = walked
			path = path[:len(path)-1]
			continue
		}
		top.rest = after
		if _, builtin := predefined[ref]; builtin {
			continue
		}
		if _, counted := g.outside[ref]; g.declared && !counted && !g.decls[g.first[top.name]].inPE {
			return xsderr.New(xsderr.RuleXMLWellFormed, g.loc, "%s that references, directly or indirectly, entity %s, whose replacement text references entity %s, which no general entity declaration outside every parameter entity declares (XML 1.0 WFC: Entity Declared)", about, top.name, ref)
		}
		if _, known := g.first[ref]; !known {
			continue
		}
		switch g.state[ref] {
		case unwalked:
			if path, err = g.enter(path, about, ref); err != nil {
				return err
			}
		case walking:
			return xsderr.New(xsderr.RuleXMLWellFormed, g.loc, "%s that references, directly or indirectly, entity %s, which references itself (XML 1.0 WFC: No Recursion)", about, ref)
		case walked:
		}
	}
	return nil
}

// enter returns path with the declared general entity name pushed on it, or
// the fault of an entity the default value about describes may not reference,
// directly or indirectly: an unparsed one (XML 1.0 WFC: Parsed Entity), an
// external one (WFC: No External Entity References), one whose replacement
// text holds '<' (WFC: No < in Attribute Values), or one whose replacement
// text holds a '&' that begins no Reference (strayAmp), which is no [10]
// AttValue where the text is included (§4.4.5, [67] Reference).
func (g *entityGraph) enter(path []walkFrame, about, name string) ([]walkFrame, error) {
	d := g.decls[g.first[name]]
	switch {
	case d.unparsed:
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, g.loc, "%s that references, directly or indirectly, the unparsed entity %s (XML 1.0 WFC: Parsed Entity)", about, name)
	case !d.value.readable:
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, g.loc, "%s that references, directly or indirectly, the external entity %s (XML 1.0 WFC: No External Entity References)", about, name)
	case strings.ContainsRune(d.value.text, '<'):
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, g.loc, "%s that references, directly or indirectly, entity %s, whose replacement text holds '<' (XML 1.0 WFC: No < in Attribute Values)", about, name)
	case strayAmp(d.value.text):
		return nil, xsderr.New(xsderr.RuleXMLWellFormed, g.loc, "%s that references, directly or indirectly, entity %s, whose replacement text holds a '&' that begins no Reference, '&' Name ';' or a character reference (XML 1.0 §4.4.5, [10] AttValue, [67] Reference)", about, name)
	}
	g.state[name] = walking
	return append(path, walkFrame{name: name, rest: d.value.text}), nil
}

// nextEntityRef returns the Name of the first EntityRef, '&' Name ';' (XML 1.0
// [68]), in s — a default value's text, or replacement text it includes — and
// what follows it, or reports false when s holds none. A CharRef ([66]) is no
// EntityRef and is passed over; every other '&' in s begins an EntityRef,
// attValueFault having charged one in a default value that does not, and enter
// one in replacement text.
func nextEntityRef(s string) (name, after string, ok bool) {
	for {
		amp := strings.IndexByte(s, '&')
		if amp < 0 {
			return "", "", false
		}
		s = s[amp+1:]
		if end := strings.IndexByte(s, ';'); end >= 0 && isName(s[:end]) {
			return s[:end], s[end+1:], true
		}
	}
}

// cutSpace reports whether s opens with S (XML 1.0 [3]), returning s with its
// leading S removed.
func cutSpace(s string) (rest string, spaced bool) {
	rest = strings.TrimLeft(s, declSpace)
	return rest, len(rest) < len(s)
}

// tokenRun splits s before its first S, quote, parenthesis or content-model
// punctuation ('|', ',', '?', '*', '+'), returning the run before it — in a
// well-formed <!ELEMENT> or <!ATTLIST> declaration a Name, an Nmtoken or a
// keyword — and what follows it.
func tokenRun(s string) (tok, rest string) {
	end := strings.IndexAny(s, declSpace+`"'()|,?*+`)
	if end < 0 {
		return s, ""
	}
	return s[:end], s[end:]
}

// noPEReference returns the fault of a parameter-entity reference standing
// outside the literals of body, a markup declaration's text: in the internal
// subset one must not occur within a markup declaration (XML 1.0 WFC: PEs in
// Internal Subset), and every parameter entity this scan reads is internal.
// The one literal that recognizes such a reference is an entity value's (see
// entityValueFault).
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

// readEntityDecl reads body, an <!ENTITY> declaration's text after its
// keyword, against XML 1.0 [70] EntityDecl, returning the declaration and
// whether it declares a parameter entity, or the fault of a body that matches
// neither [71] GEDecl, S Name S EntityDef S?, nor [72] PEDecl, S '%' S Name S
// PEDef S?. Its tokens are entityDefTokens', so a missing S runs two tokens
// into one that matches nothing: `a"x"`, `SYSTEM"x"` and `"x"NDATA` are
// faults. EntityDef ([73]) is one EntityValue literal ([9], see
// readEntityValue) or an ExternalID ([75]): 'SYSTEM' S SystemLiteral ([11]),
// or 'PUBLIC' S PubidLiteral ([12], [13]) S SystemLiteral, its keywords in
// upper case. An EntityValue ends the definition; an ExternalID may be
// followed by an NDataDecl ([76]), 'NDATA' S Name, in a general entity's
// EntityDef alone, never in a PEDef ([74]). The notation the NDataDecl names
// need not be declared: VC: Notation Declared binds a validating processor
// only. A SystemLiteral and a PubidLiteral recognize no reference.
func (sc *subsetScan) readEntityDecl(body string) (entityDecl, bool, error) {
	if body == "" || !strings.ContainsRune(declSpace, rune(body[0])) {
		return entityDecl{}, false, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ENTITY> declaration with no S after its keyword (XML 1.0 [71] GEDecl, [72] PEDecl)", sc.where())
	}
	toks := entityDefTokens(body)
	pe := len(toks) > 0 && toks[0] == "%"
	decl, defn := "[71] GEDecl", "[73] EntityDef"
	if pe {
		toks = toks[1:]
		decl, defn = "[72] PEDecl", "[74] PEDef"
	}
	if len(toks) == 0 || !isName(toks[0]) {
		name := ""
		if len(toks) > 0 {
			name = toks[0]
		}
		return entityDecl{}, false, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ENTITY> declaration whose name %q is not a Name (XML 1.0 %s, [5] Name)", sc.where(), name, decl)
	}
	d := entityDecl{name: toks[0], inPE: sc.depth > 0}
	def := toks[1:]
	if len(def) == 0 {
		return entityDecl{}, false, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ENTITY> declaration of %q with no definition (XML 1.0 %s, %s)", sc.where(), d.name, decl, defn)
	}
	if q := def[0][0]; q == '"' || q == '\'' {
		text, err := sc.readEntityValue(def)
		if err != nil {
			return entityDecl{}, false, err
		}
		d.value = entityValue{text: text, readable: true}
		return d, pe, nil
	}
	n, err := sc.externalID(def)
	if err != nil {
		return entityDecl{}, false, err
	}
	ndata := def[n:]
	if len(ndata) == 0 {
		return d, pe, nil
	}
	if pe {
		return entityDecl{}, false, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds a parameter entity declaration with %q after its ExternalID, where a PEDef admits no NDataDecl (XML 1.0 [72] PEDecl, [74] PEDef)", sc.where(), ndata[0])
	}
	if len(ndata) != 2 || ndata[0] != "NDATA" || !isName(ndata[1]) {
		return entityDecl{}, false, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ENTITY> declaration with %q after its ExternalID, where only an NDataDecl, 'NDATA' S Name, may stand (XML 1.0 [71] GEDecl, [73] EntityDef, [76] NDataDecl)", sc.where(), strings.Join(ndata, " "))
	}
	d.unparsed = true
	return d, false, nil
}

// readEntityValue reads def, an entity definition's tokens opening with a quote,
// as one EntityValue literal and nothing after it (XML 1.0 [73] EntityDef,
// [74] PEDef), returning its replacement text, or the fault of a definition
// that is not: a literal with text run on after it, a token after it, or a
// literal entityValueFault refuses.
func (sc *subsetScan) readEntityValue(def []string) (string, error) {
	if !isLiteral(def[0]) {
		return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ENTITY> declaration whose entity value %q is no quoted literal followed by S or '>' (XML 1.0 [9] EntityValue, [71] GEDecl, [72] PEDecl)", sc.where(), def[0])
	}
	lit := def[0][1 : len(def[0])-1]
	if err := sc.entityValueFault(lit); err != nil {
		return "", err
	}
	if len(def) > 1 {
		return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ENTITY> declaration with %q after its entity value, which ends the definition (XML 1.0 [73] EntityDef, [74] PEDef)", sc.where(), def[1])
	}
	return replacementText(lit), nil
}

// entityValueFault returns the fault of lit, the text between an entity value
// literal's quotes, when it is no [9] EntityValue: every '%' and '&' in it must
// open a PEReference ([69]) or a Reference ([67]). XML 1.0 §2.8 recognizes
// parameter-entity references in an entity value literal, alone among
// literals, so a PEReference there breaks WFC: PEs in Internal Subset, and any
// other '%' is no EntityValue. A '&' must open an EntityRef, '&' Name ';'
// ([68]), whether or not the entity is declared (§4.4.7 bypasses it), or a
// CharRef ([66]), '&#' [0-9]+ ';' or '&#x' [0-9a-fA-F]+ ';', naming a Char
// (WFC: Legal Character), which binds at the declaration: the replacement text
// holds the character (§4.5), whether or not the entity is ever referenced.
func (sc *subsetScan) entityValueFault(lit string) error {
	for {
		i := strings.IndexAny(lit, "%&")
		if i < 0 {
			return nil
		}
		mark, rest := lit[i], lit[i+1:]
		if mark == '%' {
			if name, _, ok := peReference(rest); ok {
				return sc.peInMarkup(name)
			}
			return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an entity value literal with a '%%' that opens no PEReference (XML 1.0 [9] EntityValue, [69] PEReference)", sc.where())
		}
		after, err := sc.reference(rest, "an entity value literal", "[9] EntityValue")
		if err != nil {
			return err
		}
		lit = after
	}
}

// reference reads the Reference whose '&' has just been consumed from rest, in
// the literal what describes, whose production is rule, returning what follows
// its ';', or the fault of a '&' that opens no Reference: an EntityRef, '&'
// Name ';' ([68]), or a CharRef ([66]), '&#' [0-9]+ ';' or '&#x' [0-9a-fA-F]+
// ';', naming a Char (WFC: Legal Character).
func (sc *subsetScan) reference(rest, what, rule string) (after string, err error) {
	end := strings.IndexByte(rest, ';')
	if end < 0 || !isReference(rest[:end]) {
		return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds %s with a '&' that opens no Reference (XML 1.0 %s, [67] Reference, [68] EntityRef, [66] CharRef)", sc.where(), what, rule)
	}
	if digits, ok := strings.CutPrefix(rest[:end], "#"); ok {
		if _, legal := charRef(digits); !legal {
			return "", xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds %s whose character reference &#%s; names no XML character (XML 1.0 [66] CharRef, WFC: Legal Character)", sc.where(), what, digits)
		}
	}
	return rest[end+1:], nil
}

// isReference reports whether ref, the text between a Reference's '&' and
// ';', reads as an EntityRef's Name ([68], [5]) or a CharRef's '#' and decimal
// digits, or '#x' and hexadecimal ones ([66]), whatever character they name.
func isReference(ref string) bool {
	digits, ok := strings.CutPrefix(ref, "#")
	if !ok {
		return isName(ref)
	}
	set := "0123456789"
	if hex, ok := strings.CutPrefix(digits, "x"); ok {
		digits, set = hex, "0123456789abcdefABCDEF"
	}
	return digits != "" && strings.Trim(digits, set) == ""
}

// strayAmp reports whether s, replacement text or a run of it, holds a '&'
// that begins no Reference ([67]): no ';' follows it, or what stands between
// them is no Reference (isReference). A literal's `&#38;` or `&#x26;` puts a
// bare '&' in replacement text (XML 1.0 §4.5), and the text is reparsed where
// it is included (§4.4.2, §4.4.5), so such a '&' is a fault there unless a
// Reference follows it: `&#38;#60;` is clean, `&#38;b` is not.
func strayAmp(s string) bool {
	for {
		amp := strings.IndexByte(s, '&')
		if amp < 0 {
			return false
		}
		s = s[amp+1:]
		end := strings.IndexByte(s, ';')
		if end < 0 || !isReference(s[:end]) {
			return true
		}
		s = s[end+1:]
	}
}

// externalID reads the ExternalID def, an entity definition's tokens, opens
// with — 'SYSTEM' S SystemLiteral or 'PUBLIC' S PubidLiteral S SystemLiteral
// (XML 1.0 [75], [11], [12], [13]) — returning how many tokens it spans, or the
// fault of a definition that opens no ExternalID: a keyword in another case
// among them, or one run on into its literal.
func (sc *subsetScan) externalID(def []string) (int, error) {
	switch {
	case def[0] == "SYSTEM" && len(def) > 1 && isLiteral(def[1]):
		return 2, nil
	case def[0] == "PUBLIC" && len(def) > 2 && isPubidLiteral(def[1]) && isLiteral(def[2]):
		return 3, nil
	}
	return 0, xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s holds an <!ENTITY> declaration whose definition %q is neither an entity value literal nor an ExternalID, 'SYSTEM' S SystemLiteral or 'PUBLIC' S PubidLiteral S SystemLiteral (XML 1.0 [73] EntityDef, [74] PEDef, [75] ExternalID, [11] SystemLiteral, [12] PubidLiteral)", sc.where(), strings.Join(def, " "))
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

// excerpt is the run s opens, up to the next S, '<' or ']' after its first
// byte and at most maxExcerpt runes, for a fault message to quote; "" when s
// is.
func excerpt(s string) string {
	if s == "" {
		return ""
	}
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

// unclosed is the fault of the comment, processing instruction or markup
// declaration s opens and never closes. In replacement text it breaks XML 1.0
// WFC: PE Between Declarations: a parameter entity referenced between
// declarations must expand to complete markup declarations ([31]
// extSubsetDecl). In the internal subset itself it is no [28b] intSubset,
// whose every markupdecl ([29]) closes before the subset's ']'. The decoder
// ends a DOCTYPE directive only outside every such construct (internal/xmltok
// reads a processing instruction there through its "?>"), so through the
// Reader the internal subset never ends inside one, and that fault is reached
// from doctypeEntities alone.
func (sc *subsetScan) unclosed(s string) error {
	rule := "XML 1.0 [28b] intSubset, [29] markupdecl"
	if sc.depth > 0 {
		rule = "XML 1.0 WFC: PE Between Declarations, [31] extSubsetDecl"
	}
	return xsderr.New(xsderr.RuleXMLWellFormed, sc.loc, "%s leaves %q open where it ends (%s)", sc.where(), excerpt(s), rule)
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

// declare records one <!ENTITY> declaration d, read by readEntityDecl: a
// parameter entity's value into pes, when pe, and a general entity into decls.
// The first declaration of a name binds (XML 1.0 §4.2); for a general entity
// that is the Reader's to apply.
func (sc *subsetScan) declare(d entityDecl, pe bool) {
	if !pe {
		sc.decls = append(sc.decls, d)
		return
	}
	if _, bound := sc.pes[d.name]; bound {
		return
	}
	if sc.pes == nil {
		sc.pes = make(map[string]entityValue)
	}
	sc.pes[d.name] = d.value
}

// replacementText builds an internal entity's replacement text from its
// literal entity value (XML 1.0 §4.5): each line end the literal spells is
// normalized to #xA (§2.11), each character reference is replaced by the
// character it names, and a general entity reference is left as it stands
// (§4.4.7, bypassed), for the reader to expand where the entity is included.
// The literal is one entityValueFault passed: it holds no '%', and every '&'
// in it opens a Reference whose character reference names a Char.
func replacementText(lit string) string {
	var b strings.Builder
	for {
		amp := strings.IndexByte(lit, '&')
		if amp < 0 {
			b.WriteString(lineEnds.Replace(lit))
			return b.String()
		}
		b.WriteString(lineEnds.Replace(lit[:amp]))
		lit = lit[amp:]
		if !strings.HasPrefix(lit, "&#") {
			b.WriteByte('&')
			lit = lit[1:]
			continue
		}
		end := strings.IndexByte(lit, ';')
		r, _ := charRef(lit[len("&#"):end])
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

// isName reports whether t, a declaration token, is an XML 1.0 Name
// (production [5]): it is not empty, its first character is a NameStartChar
// ([4]) and every later one a NameChar ([4a]), both internal/xmlname's tables.
// It checks the DOCTYPE's document type name, an entity's declared name, the
// name a PEReference carries, between declarations or in an entity value, the
// name an EntityRef in an entity value or an attribute default carries, the
// notation name an NDataDecl closes on, and the element type, attribute and
// notation names of <!ELEMENT> and <!ATTLIST> declarations, rejecting `1x`,
// `g&h` and `a×b` (U+00D7 is in neither production).
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

// isNmtoken reports whether t is an XML 1.0 Nmtoken (production [7]): one or
// more NameChars ([4a]), a NameStartChar or one of internal/xmlname's
// NameCharExtra, in any position, so `1x` is one and `a×b` is not.
func isNmtoken(t string) bool {
	if t == "" {
		return false
	}
	for _, r := range t {
		if !unicode.In(r, xmlname.NameStartChar, xmlname.NameCharExtra) {
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
// tokens of a markup declaration, and the only character data that may precede
// or follow the document element (see outsideRootFault).
const declSpace = " \t\r\n"

// declTokens splits a markup declaration body on white space, keeping each
// quoted literal whole and WITH its quotes, so a literal never compares equal
// to a keyword.
func declTokens(s string) []string {
	return splitDecl(s, false)
}

// entityDefTokens is declTokens for an <!ENTITY> body, general or parameter,
// except that every token ends only at white space. Text run on after a
// literal's closing quote with no S between — `"x"NDATA`, which XML 1.0
// NDataDecl [76] forbids — stays in the literal's token, which isLiteral then
// refuses; a literal run on after a keyword — `SYSTEM"x"`, which ExternalID
// [75] forbids — stays in the keyword's token, which readEntityDecl then
// refuses. The DOCTYPE header's ExternalID (hasExternalID) keeps declTokens'
// split, where every token ends at a quote as well as at white space; the
// header's name does not (see doctypeName).
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
