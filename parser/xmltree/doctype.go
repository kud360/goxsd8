package xmltree

import (
	"strconv"
	"strings"
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
// Infoset §2.1). A directive that is not a DOCTYPE declares none and leaves
// nothing unread. standalone is the XML declaration's standalone="yes" (XML
// 1.0 §2.9 SDDecl).
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
// Recursion) always reaches — reports unread and, unless standalone, ends the scan:
// §5.1 forbids processing an entity declaration that follows a reference to a
// parameter entity that is not read, except when standalone="yes". A
// conditional section, which the internal subset cannot hold and this scan
// does not read, is declined the same way and ends the text it appears in.
//
// It reads markup and nothing else: a comment, a processing instruction and
// any markup declaration other than <!ENTITY> are stepped over whole. A
// declaration it cannot read declares no unparsed entity, which leaves an
// ·ENTITY value· naming that entity undeclared rather than declared. One
// not-well-formed declaration is read all the same: a notation name that
// breaks the Name production only in a non-ASCII character, the GAP(xml)
// isNotationName carries.
func doctypeEntities(directive string, standalone bool) (decls []entityDecl, unread bool) {
	rest, ok := strings.CutPrefix(directive, "DOCTYPE")
	if !ok {
		return nil, false
	}
	header := rest
	open := outsideQuotes(rest, '[')
	if open >= 0 {
		header = rest[:open]
	}
	sc := subsetScan{standalone: standalone, unread: hasExternalID(header)}
	if open >= 0 {
		sc.scan(rest[open+1:])
	}
	return sc.decls, sc.unread
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
type subsetScan struct {
	standalone bool
	pes        map[string]entityValue
	depth      int
	spent      int
	decls      []entityDecl
	unread     bool
}

// entityValue is one entity's replacement text, and whether it has one the
// reader can read — false for an external entity, a malformed definition, or
// a literal whose replacement text cannot be built.
type entityValue struct {
	text     string
	readable bool
}

// scan reads s — the internal subset, or a parameter entity's enlarged
// replacement text — and reports whether the whole scan stops, which a
// declined reference makes it do unless standalone. Only the internal subset
// itself ends at a ']'. A comment, processing instruction or markup
// declaration left open where replacement text ends is declined (see
// unclosed).
func (sc *subsetScan) scan(s string) (stop bool) {
	for s != "" {
		switch {
		case s[0] == ']' && sc.depth == 0:
			return false
		case strings.HasPrefix(s, "<!--"):
			end := strings.Index(s, "-->")
			if end < 0 {
				return sc.unclosed()
			}
			s = s[end+len("-->"):]
		case strings.HasPrefix(s, "<?"):
			end := strings.Index(s, "?>")
			if end < 0 {
				return sc.unclosed()
			}
			s = s[end+len("?>"):]
		case strings.HasPrefix(s, "<!["):
			return sc.decline()
		case strings.HasPrefix(s, "<!ENTITY"):
			body, after, closed := markupDecl(s[len("<!ENTITY"):])
			if !closed && sc.depth > 0 {
				return sc.decline()
			}
			sc.declare(body)
			s = after
		case strings.HasPrefix(s, "<!"):
			_, after, closed := markupDecl(s[len("<!"):])
			if !closed {
				return sc.unclosed()
			}
			s = after
		case s[0] == '%':
			name, after, ok := peReference(s[1:])
			if !ok {
				return sc.decline()
			}
			if sc.expand(name) {
				return true
			}
			s = after
		default:
			s = s[1:]
		}
	}
	return false
}

// peReference reads the Name and ';' of a PEReference whose '%' has just been
// consumed, returning what follows it. It reports false when no ';' closes a
// run that could be a Name.
func peReference(s string) (name, after string, ok bool) {
	end := strings.IndexByte(s, ';')
	if end < 1 || !isDeclName(s[:end]) || strings.ContainsAny(s[:end], declSpace+"<>&;") {
		return "", "", false
	}
	return s[:end], s[end+1:], true
}

// expand reads the replacement text of the parameter entity a reference
// names, or declines the reference when that text cannot be read within
// bounds.
func (sc *subsetScan) expand(name string) (stop bool) {
	pe, declared := sc.pes[name]
	cost := len(pe.text) + len("  ")
	if !declared || !pe.readable || sc.depth == maxPEDepth || sc.spent+cost > maxPEExpansion {
		return sc.decline()
	}
	sc.spent += cost
	sc.depth++
	stop = sc.scan(" " + pe.text + " ")
	sc.depth--
	return stop
}

// unclosed ends the read of s at a construct s opens and never closes. In
// replacement text that construct is declined: a parameter entity referenced
// between declarations must expand to complete markup declarations (XML 1.0
// WFC PE Between Declarations), so the text is not well-formed and the scan
// must not read on past it. In the internal subset itself only the scan ends.
func (sc *subsetScan) unclosed() (stop bool) {
	if sc.depth == 0 {
		return false
	}
	return sc.decline()
}

// decline records a reference or construct the scan does not read, and
// reports whether the scan stops there: XML 1.0 §5.1 forbids processing the
// entity declarations after it unless standalone="yes".
func (sc *subsetScan) decline() (stop bool) {
	sc.unread = true
	return !sc.standalone
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
// declaration, <!ENTITY % name PEDef>, reporting false for any other body.
// The entity is readable only when its PEDef is one EntityValue literal whose
// replacement text can be built (see replacementText): an ExternalID, or
// anything malformed, declares an entity this scan never reads.
func paramEntityOf(body string) (string, entityValue, bool) {
	if body == "" || !strings.ContainsRune(declSpace, rune(body[0])) {
		return "", entityValue{}, false
	}
	toks := declTokens(body)
	if len(toks) < 3 || toks[0] != "%" || !isDeclName(toks[1]) {
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
// It reports false for a literal holding a parameter-entity reference, which
// the internal subset forbids inside a markup declaration (WFC PEs in Internal
// Subset), or a character reference that is malformed or names no Char (WFC
// Legal Character).
func replacementText(lit string) (string, bool) {
	if strings.ContainsRune(lit, '%') {
		return "", false
	}
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
	legal := r == 0x9 || r == 0xA || r == 0xD ||
		(r >= 0x20 && r <= 0xD7FF) || (r >= 0xE000 && r <= 0xFFFD) || (r >= 0x10000 && r <= 0x10FFFF)
	return r, legal
}

// outsideQuotes reports the index of the first c in s that is not inside a
// quoted literal, or -1.
func outsideQuotes(s string, c byte) int {
	var quote byte
	for i := 0; i < len(s); i++ {
		switch {
		case quote != 0:
			if s[i] == quote {
				quote = 0
			}
		case s[i] == '"' || s[i] == '\'':
			quote = s[i]
		case s[i] == c:
			return i
		}
	}
	return -1
}

// markupDecl splits s at the '>' closing the markup declaration it is inside
// of, returning the declaration's body and what follows it, and reports
// whether that '>' is in s. A declaration with no closing '>' runs to the end
// of s.
func markupDecl(s string) (body, after string, closed bool) {
	end := outsideQuotes(s, '>')
	if end < 0 {
		return s, "", false
	}
	return s[:end], s[end+1:], true
}

// entityDeclOf reads one <!ENTITY ...> body. It reports false for a parameter
// entity declaration (<!ENTITY % name ...>), which declares no general entity
// at all, for a keyword run on into the name with no white space between, and
// for a body too short to name one. A general entity is unparsed only when its
// definition is exactly an ExternalID followed by an NDataDecl (XML 1.0
// EntityDef); any other definition, a malformed one included, is a parsed
// entity. A parsed entity is internal, with a readable value, only when its
// definition is exactly one EntityValue literal whose replacement text builds.
func entityDeclOf(body string) (entityDecl, bool) {
	if body == "" || !strings.ContainsRune(declSpace, rune(body[0])) {
		return entityDecl{}, false
	}
	toks := entityDefTokens(body)
	if len(toks) < 2 || strings.HasPrefix(toks[0], "%") {
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
	return def[1+lits] == "NDATA" && isNotationName(def[2+lits])
}

// isNotationName reports whether t, the token an NDataDecl closes on, is an
// XML 1.0 Name (production [5]) in its ASCII characters: the first is a
// NameStartChar ([4]: ':', A-Z, '_', a-z) and every later one a NameChar ([4a]:
// those, '-', '.', 0-9). It rejects the notation names `g&h` and `1gif`.
//
// GAP(xml): a non-ASCII character is admitted without the range check of
// NameStartChar [4] and NameChar [4a], so a notation name breaking Name only
// in one — '×' (#xD7), say — reads as a Name and its entity as unparsed. The
// direction is fail-OPEN against the set's one reader: validate's
// (*walk).entitiesDeclared, reached through xmltree.Reader.HasUnparsedEntity
// and validate/xmlsrc's element.HasUnparsedEntity, raises no cvc-simple-type
// clause 3 error for an ·ENTITY value· naming that entity. Owned by #1745.
func isNotationName(t string) bool {
	for i := 0; i < len(t); i++ {
		c := t[i]
		start := c == ':' || c == '_' || ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z')
		later := i > 0 && (c == '-' || c == '.' || ('0' <= c && c <= '9'))
		if c < 0x80 && !start && !later {
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

// isDeclName reports whether t, one of declTokens' tokens, can be a Name: it
// holds no quote, no subset bracket and no parameter-entity reference.
func isDeclName(t string) bool {
	return !strings.ContainsAny(t, `"'[]%`)
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
// header and a parameter entity's body keep declTokens' split, where every
// token ends at a quote as well as at white space.
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
