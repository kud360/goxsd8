package icpath

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kud360/goxsd8/regex"
	"github.com/kud360/goxsd8/xsd"
)

// compile is the ONE traversal the two exported compile entry points are façades
// over (STYLE T4): it tokenizes the {expression}, indexes the record's
// {namespace bindings} and its {default namespace}, and parses production [1]'s
// union of Paths. field selects production [7] over production [2]: only a field
// may end in an attribute step.
func compile(x xsd.XPathExpression, field bool) (Expr, bool) {
	toks, ok := tokenize(x.Expression())
	if !ok {
		return Expr{}, false
	}
	r := names{prefixes: make(map[string]string)}
	for _, b := range x.NamespaceBindings() {
		r.prefixes[b.Prefix()] = b.Namespace()
	}
	r.def, r.hasDef = x.DefaultNamespace()
	return parse(toks, field, r)
}

// names resolves the NameTests of one XPath Expression property record: its
// {namespace bindings} by prefix, and its {default namespace} for an unprefixed
// name on an ELEMENT step. The map is internal and never iterated into output
// (STYLE D2) — it is read by prefix and nothing else.
type names struct {
	prefixes map[string]string
	def      string
	hasDef   bool
}

// test resolves one NameTest's text. attribute selects the axis, and with it
// what an unprefixed name means: no namespace on the attribute axis, the
// {default namespace} on the element axis. A prefix with no binding is not a
// name this package can resolve, and the whole expression declines.
func (r names) test(text string, attribute bool) (nameTest, bool) {
	if text == "*" {
		return nameTest{anySpace: true, anyLocal: true}, true
	}
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		if attribute || !r.hasDef {
			return nameTest{local: text}, true
		}
		return nameTest{space: r.def, local: text}, true
	}
	space, bound := r.prefixes[prefix]
	if !bound {
		return nameTest{}, false
	}
	if local == "*" {
		return nameTest{space: space, anyLocal: true}, true
	}
	return nameTest{space: space, local: local}, true
}

// token is one token of production [5], identified by the character that opens
// it — '.', '/', 'D' for '//', '|', '@', or 'n' for a NameTest, whose text is
// carried.
type token struct {
	kind byte
	text string
}

// tokenize splits an {expression} into the tokens production [5] lists — "token
// ::= '.' | '/' | '//' | '|' | '@' | NameTest" — longest-token first, with
// production [6]'s white space allowed around tokens though not inside them.
//
// Longest-token is load-bearing in one place a shorter rule gets wrong: '.' is a
// legal NCName character after the first, so "a.b" is ONE NameTest and not a
// name, a self step and a second name. The scan reaches that by trying a
// NameTest first wherever one can start, which '.' cannot: an NCName opens with
// \i (Datatypes §3.4.7.1's pattern facet), and \i is XML's NameStartChar
// (§G.4.2.5), which does not admit '.'.
func tokenize(s string) ([]token, bool) {
	var toks []token
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.IsSpace(r):
			i += size
		case r == '|', r == '@', r == '.':
			toks = append(toks, token{kind: byte(r)})
			i += size
		case r == '/':
			if strings.HasPrefix(s[i:], "//") {
				toks = append(toks, token{kind: 'D'})
				i += 2
				continue
			}
			toks = append(toks, token{kind: '/'})
			i++
		case r == '*':
			toks = append(toks, token{kind: 'n', text: "*"})
			i += size
		default:
			j := scanNameTest(s, i)
			if j == i {
				return nil, false
			}
			toks = append(toks, token{kind: 'n', text: s[i:j]})
			i = j
		}
	}
	return toks, true
}

// scanNameTest reports the end of the NameTest starting at i, or i itself where
// none does. It admits the shapes of production [4] that can start with a name
// character — QName, prefixed or not, and 'NCName:*' — the bare '*' being
// handled by the tokenizer before it gets here.
func scanNameTest(s string, i int) int {
	j := scanNCName(s, i)
	if j == i || j >= len(s) || s[j] != ':' {
		return j
	}
	if j+1 < len(s) && s[j+1] == '*' {
		return j + 2
	}
	k := scanNCName(s, j+1)
	if k == j+1 {
		return i // 'p:' with no local part is no NameTest at all
	}
	return k
}

// ncNameRE matches the longest NCName at the START of the string it is applied
// to — [Namespaces in XML] production [4], spelled as the pattern Datatypes
// §3.4.7.1 fixes for xs:NCName, "[\i-[:]][\c-[:]]*". It is translated and
// compiled once here through [regex.Translate], so the code points behind \i and
// \c are the ones the regex package owns and not a second table (PRINCIPLES
// 26/27; regex/class.go records which edition of XML supplies them).
//
// The flavor is FO because only FO's '^' is a real anchor: FlavorXSD anchors the
// WHOLE string, which cannot express a prefix scan. This pattern carries no
// construct the two flavors read differently.
var ncNameRE = func() *regexp.Regexp {
	goRE, err := regex.Translate(`^[\i-[:]][\c-[:]]*`, regex.FlavorFO, "")
	if err != nil {
		panic("icpath: translating the NCName pattern: " + err.Error())
	}
	return regexp.MustCompile(goRE)
}()

// scanNCName reports the end of the NCName starting at i, or i where none does.
// Datatypes §G.4.2.5 defines \i and \c by direct reference to XML's
// NameStartChar and NameChar, so the class here is the grammar's own and not an
// approximation of it — which is what a PREFIX has to have. The name this bounds
// is resolved against the {namespace bindings} in scope (PRINCIPLES 15), and a
// scan admitting one character too many would carry into the prefix a character
// that ends the name and opens the next token.
//
// The match is anchored at i, so its length IS the end it reports. A miss and an
// empty match are the same "" here, and mean the same thing: the pattern
// requires a NameStartChar, so it never matches empty.
func scanNCName(s string, i int) int {
	return i + len(ncNameRE.FindString(s[i:]))
}

// parse parses the token stream as production [1]'s union of Paths, declining an
// empty union and any Path the grammar does not admit.
func parse(toks []token, field bool, r names) (Expr, bool) {
	var x Expr
	start := 0
	for i := 0; i <= len(toks); i++ {
		if i < len(toks) && toks[i].kind != '|' {
			continue
		}
		p, ok := parsePath(toks[start:i], field, r)
		if !ok {
			return Expr{}, false
		}
		x.paths = append(x.paths, p)
		start = i + 1
	}
	if len(x.paths) == 0 {
		return Expr{}, false
	}
	return x, true
}

// parsePath parses one Path: the optional `.//` prefix, then Steps separated by
// '/', with an '@' NameTest admitted as the final step for a field alone.
//
// A `.` Step is dropped as it is read (see path). The one shape dropping cannot
// handle is a `.//` path whose every Step was a `.`, which leaves no element
// step to match at any depth; it is declined rather than guessed at.
func parsePath(toks []token, field bool, r names) (path, bool) {
	var p path
	if len(toks) >= 2 && toks[0].kind == '.' && toks[1].kind == 'D' {
		p.anyDepth = true
		toks = toks[2:]
	}
	if len(toks) == 0 {
		return path{}, false
	}
	for len(toks) > 0 {
		switch toks[0].kind {
		case '@':
			if !field || len(toks) != 2 || toks[1].kind != 'n' {
				return path{}, false
			}
			t, ok := r.test(toks[1].text, true)
			if !ok {
				return path{}, false
			}
			p.attr, p.hasAttr = t, true
			toks = nil
		case '.':
			toks = toks[1:]
		case 'n':
			t, ok := r.test(toks[0].text, false)
			if !ok {
				return path{}, false
			}
			p.steps = append(p.steps, t)
			toks = toks[1:]
		default:
			return path{}, false
		}
		if len(toks) == 0 {
			break
		}
		if toks[0].kind != '/' {
			return path{}, false
		}
		toks = toks[1:]
		if len(toks) == 0 {
			return path{}, false // a trailing '/' has no Step after it
		}
	}
	if p.anyDepth && len(p.steps) == 0 {
		return path{}, false
	}
	return p, true
}
