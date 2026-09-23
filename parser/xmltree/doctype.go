package xmltree

import (
	"slices"
	"strings"
)

// entityDecl is one general entity declaration of a DOCTYPE's internal
// subset: the entity's name, and whether it is UNPARSED — an external entity
// carrying an NDATA notation name, which is what an ·ENTITY value· must name
// (Structures §3.16.4 key-vde).
type entityDecl struct {
	name     string
	unparsed bool
}

// doctypeEntities reads the general entity declarations of one directive's
// internal subset, in document order. A directive that is not a DOCTYPE, or a
// DOCTYPE with no internal subset, declares none.
//
// It reads markup and nothing else: comments are already gone (encoding/xml
// replaces each one inside a directive with a space), a processing
// instruction or any other markup declaration is stepped over whole, and a
// parameter-entity reference is stepped over unexpanded, so an entity
// declared in its replacement text is not seen. A declaration it cannot read
// declares nothing, which leaves an ·ENTITY value· naming that entity
// undeclared rather than declared.
func doctypeEntities(directive string) []entityDecl {
	rest, ok := strings.CutPrefix(directive, "DOCTYPE")
	if !ok {
		return nil
	}
	open := outsideQuotes(rest, '[')
	if open < 0 {
		return nil
	}
	var decls []entityDecl
	for s := rest[open+1:]; s != ""; {
		switch {
		case s[0] == ']':
			return decls
		case strings.HasPrefix(s, "<?"):
			end := strings.Index(s, "?>")
			if end < 0 {
				return decls
			}
			s = s[end+len("?>"):]
		case strings.HasPrefix(s, "<!ENTITY"):
			body, after := markupDecl(s[len("<!ENTITY"):])
			if d, general := entityDeclOf(body); general {
				decls = append(decls, d)
			}
			s = after
		case strings.HasPrefix(s, "<!"):
			_, s = markupDecl(s[len("<!"):])
		default:
			s = s[1:]
		}
	}
	return decls
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
// of, returning the declaration's body and what follows it. A declaration
// with no closing '>' runs to the end of s.
func markupDecl(s string) (body, after string) {
	end := outsideQuotes(s, '>')
	if end < 0 {
		return s, ""
	}
	return s[:end], s[end+1:]
}

// entityDeclOf reads one <!ENTITY ...> body. It reports false for a parameter
// entity declaration (<!ENTITY % name ...>), which declares no general entity
// at all, for a keyword run on into the name with no white space between, and
// for a body too short to name one. A general entity is unparsed when its
// definition is an external identifier (SYSTEM or PUBLIC) followed by an
// NDATA notation name; a literal definition is a parsed entity whatever it
// spells, since a quoted token never equals the bare keyword.
func entityDeclOf(body string) (entityDecl, bool) {
	if body == "" || !strings.ContainsRune(declSpace, rune(body[0])) {
		return entityDecl{}, false
	}
	toks := declTokens(body)
	if len(toks) < 2 || strings.HasPrefix(toks[0], "%") {
		return entityDecl{}, false
	}
	external := toks[1] == "SYSTEM" || toks[1] == "PUBLIC"
	return entityDecl{name: toks[0], unparsed: external && slices.Contains(toks[2:], "NDATA")}, true
}

// declSpace is the white space that separates the tokens of a markup
// declaration: XML 1.0's S production.
const declSpace = " \t\r\n"

// declTokens splits a markup declaration body on white space, keeping each
// quoted literal whole and WITH its quotes, so a literal never compares equal
// to a keyword.
func declTokens(s string) []string {
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
			toks = append(toks, s[:end+2])
			s = s[end+2:]
			continue
		}
		end := strings.IndexAny(s, declSpace+`"'`)
		if end < 0 {
			return append(toks, s)
		}
		toks = append(toks, s[:end])
		s = s[end:]
	}
}
