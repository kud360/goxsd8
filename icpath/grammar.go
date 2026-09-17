package icpath

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kud360/goxsd8/regex"
	"github.com/kud360/goxsd8/xsd"
)

// compile is the ONE traversal the four exported entry points are façades over
// (STYLE T4): it tokenizes the {expression}, indexes the record's {namespace
// bindings} and its {default namespace}, parses production [1]'s union of Paths,
// and reports the tree together with what — if anything — was wrong with it.
// field selects production [7] over production [2]: only a field may end in an
// attribute step.
//
// The order the three states are decided in is the whole of "charge these four,
// decline everything else":
//
//   - UNSUPPORTED DOMINATES. A stream carrying a run this lexer cannot read is
//     declined whatever else it holds, before any shape is judged. That is what
//     keeps a quoted "[" inside an expression this package does not read from
//     being charged as a predicate, and it is why `child::p:a` with p unbound is
//     a decline and not a clause-1 charge: `::` does not lex, and reading its
//     prefix in isolation would reject a schema whose only fault is a spelling
//     this compiler does not read.
//   - A SHAPE VIOLATION IS INDEPENDENT OF THE PARSE. Over a fully lexed stream
//     shapeFault proves its three shapes from the tokens alone, so `a[b]` is
//     charged although it parses to nothing at all.
//   - AN UNBOUND PREFIX NEEDS A COMPLETE PARSE. Name resolution never fails a
//     parse — it records and carries on — so a parse that failed at all failed
//     for another reason, and is a decline with whatever an unbound prefix
//     recorded along the way DISCARDED.
func compile(x xsd.XPathExpression, field bool) (Expr, defect) {
	toks := tokenize(x.Expression())
	if !fullyLexed(toks) {
		return Expr{}, defect{kind: defectUnsupported}
	}
	if fault, bad := shapeFault(toks, field); bad {
		return Expr{}, shapeViolation(x, field, fault)
	}
	r := names{prefixes: make(map[string]string)}
	for _, b := range x.NamespaceBindings() {
		r.prefixes[b.Prefix()] = b.Namespace()
	}
	r.def, r.hasDef = x.DefaultNamespace()
	expr, ok := parse(toks, field, &r)
	if !ok {
		return Expr{}, defect{kind: defectUnsupported}
	}
	if r.hasUnbound {
		return Expr{}, unboundViolation(x, field, r.unbound)
	}
	return expr, defect{}
}

// names resolves the NameTests of one XPath Expression property record: its
// {namespace bindings} by prefix, and its {default namespace} for an unprefixed
// name on an ELEMENT step. The map is internal and never iterated into output
// (STYLE D2) — it is read by prefix and nothing else.
//
// unbound is the FIRST prefix with no binding, in expression order, which is the
// one the clause-1 charge names (STYLE D1). hasUnbound is not derivable from it:
// the empty prefix is not a prefix production [4] can carry, but nothing in this
// struct says so, and a present-but-empty answer would be indistinguishable from
// "none recorded".
type names struct {
	prefixes   map[string]string
	def        string
	hasDef     bool
	unbound    string
	hasUnbound bool
}

// test resolves one NameTest's text. attribute selects the axis, and with it
// what an unprefixed name means: no namespace on the attribute axis, the
// {default namespace} on the element axis.
//
// A prefix with no binding is RECORDED rather than failing the parse, because
// the two answers the caller has to tell apart — a complete production carrying
// an xpath-valid (§3.13.6.2) static error, and an {expression} this package
// simply cannot read — are distinguishable only once the rest of the stream has
// been parsed to its end. The NameTest it yields in the meantime matches
// nothing, and is never evaluated: compile discards the tree it sits in.
func (r *names) test(text string, attribute bool) nameTest {
	if text == "*" {
		return nameTest{anySpace: true, anyLocal: true}
	}
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		if attribute || !r.hasDef {
			return nameTest{local: text}
		}
		return nameTest{space: r.def, local: text}
	}
	space, bound := r.prefixes[prefix]
	if !bound {
		r.recordUnbound(prefix)
		return nameTest{unresolved: true}
	}
	if local == "*" {
		return nameTest{space: space, anyLocal: true}
	}
	return nameTest{space: space, local: local}
}

// recordUnbound records a prefix with no binding in the record's {namespace
// bindings} and lets the parse go on. The FIRST one decides the charge, so the
// answer is the one the walk reaches first in expression order and not a map's
// (STYLE D2).
//
// The record is PROVISIONAL: compile keeps it only where the parse then reached
// the end of a complete production, because an {expression} outside the subset
// is declined rather than charged, however its names resolve.
func (r *names) recordUnbound(prefix string) {
	if r.hasUnbound {
		return
	}
	r.unbound, r.hasUnbound = prefix, true
}

// token is one token of production [5], identified by the character that opens
// it — '.', '/', 'D' for '//', '|', '@', or 'n' for a NameTest, whose text is
// carried. Two kinds are NOT production [5]'s and exist so the lexer is total:
// '[' and ']' for a predicate's brackets, which the grammar admits nowhere, and
// '?' for one rune that opens no token of either sort.
type token struct {
	kind byte
	text string
}

// tokenize splits an {expression} into the tokens production [5] lists — "token
// ::= '.' | '/' | '//' | '|' | '@' | NameTest" — longest-token first, with
// production [6]'s white space allowed around tokens though not inside them.
//
// It is TOTAL: every {expression} yields a stream, and a rune that opens no
// token becomes one '?' token rather than ending the scan. A lexer that stopped
// at the first such rune would report an unbound prefix and a predicate as the
// same undifferentiated failure, and the two have opposite consequences — one is
// a Schema Component Constraint violation, the other is what this package does
// not read.
//
// Longest-token is load-bearing in one place a shorter rule gets wrong: '.' is a
// legal NCName character after the first, so "a.b" is ONE NameTest and not a
// name, a self step and a second name. The scan reaches that by trying a
// NameTest first wherever one can start, which '.' cannot: an NCName opens with
// \i (Datatypes §3.4.7.1's pattern facet), and \i is XML's NameStartChar
// (§G.4.2.5), which does not admit '.'.
func tokenize(s string) []token {
	var toks []token
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.IsSpace(r):
			i += size
		case r == '|', r == '@', r == '.', r == '[', r == ']':
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
				toks = append(toks, token{kind: '?'})
				i += size
				continue
			}
			toks = append(toks, token{kind: 'n', text: s[i:j]})
			i = j
		}
	}
	return toks
}

// fullyLexed reports whether every rune of the {expression} the stream came from
// fell inside a token this package reads — production [5]'s, or a predicate
// bracket. Nothing is charged over a stream that fails this: see compile.
func fullyLexed(toks []token) bool {
	for _, t := range toks {
		if t.kind == '?' {
			return false
		}
	}
	return true
}

// shapeFault reports the production violation a FULLY LEXED token stream proves,
// in the words the charge names it with, and ok false where it proves none. The
// three shapes it decides are exactly the ones no spelling of the {expression}
// can excuse — an abbreviated XPath and the unabbreviated form clause 2.2 admits
// as its equivalent both carry them — so each is a violation whichever of clause
// 2's two arms the author was writing under.
//
// The scan is per union member, because production [1] is a union of Paths and a
// member's final step is its own: `a/@b|c/@d` is two legal field Paths and reads
// as one illegal one if the '|' is ignored.
func shapeFault(toks []token, field bool) (string, bool) {
	scc := sccOf(field)
	for _, t := range toks {
		// A predicate is outside production [3] Step, which is '.' or a NameTest
		// and nothing else; and abbreviating an unabbreviated XPath never drops
		// one, so clause 2.2 does not admit it either.
		if t.kind == '[' || t.kind == ']' {
			return fmt.Sprintf("carries a predicate, but %s clause 2 admits none — production [3] Step is '.' or a NameTest, and abbreviating an unabbreviated XPath does not drop a predicate", scc), true
		}
	}
	for _, m := range unionMembers(toks) {
		for j, t := range m {
			if t.kind != '@' {
				continue
			}
			// Production [2] has no '@' at all and clause 2.2 names the child axis
			// alone, so an attribute anywhere in a SELECTOR is outside both arms.
			if !field {
				return fmt.Sprintf("names an attribute, but %s clause 2 admits only the child axis — production [2] has no '@' and clause 2.2 names no other axis", scc), true
			}
			// The selector arm above needs no NameTest after the '@' — the
			// attribute AXIS is what clause 2.2 withholds from a selector, whatever
			// names it — but the field arm below is a claim about POSITION, and an
			// '@' with no NameTest after it holds no position in production [7].
			//
			// GAP(xpath): an '@' with no NameTest after it, in a FIELD. It is
			// provable — no remaining token of production [5] is a NodeTest, so
			// nothing in a fully lexed stream completes the abbreviated attribute
			// step either arm of clause 2 reads — but what it breaks is
			// xpath-valid's clause 1, "The {expression} of X is a valid XPath
			// expression", and this package reads clause 2, "X does not produce
			// any static error", and nothing else. Charging it under a clause-2
			// message would cite the wrong clause, so it is DECLINED until clause
			// 1 has a grounding of its own. The withheld value reaches only
			// [FieldViolation]'s caller, parser's constructIdentityConstraint,
			// which rejects the schema on a NON-nil error and does nothing at all
			// on nil, so withholding one can only let a schema through and never
			// reject a conforming one. Nothing owns its retirement.
			if j+1 >= len(m) || m[j+1].kind != 'n' {
				continue
			}
			// Production [7] admits `'@' NameTest` as the whole of a Path's final
			// step and nowhere else, and clause 2.2's "child and/or attribute axes
			// whose abbreviated form is as given above" reaches no further.
			if j != len(m)-2 {
				return fmt.Sprintf("names an attribute before its final step, but %s clause 2 admits one only as a Path's final step (production [7])", scc), true
			}
		}
	}
	return "", false
}

// unionMembers splits a token stream on production [1]'s '|' into its Paths. An
// empty member is yielded like any other: it is no Path, which is the parse's
// decline and not this scan's business.
func unionMembers(toks []token) [][]token {
	var members [][]token
	start := 0
	for i := 0; i <= len(toks); i++ {
		if i < len(toks) && toks[i].kind != '|' {
			continue
		}
		members = append(members, toks[start:i])
		start = i + 1
	}
	return members
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
func parse(toks []token, field bool, r *names) (Expr, bool) {
	var x Expr
	for _, m := range unionMembers(toks) {
		p, ok := parsePath(m, field, r)
		if !ok {
			return Expr{}, false
		}
		x.paths = append(x.paths, p)
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
func parsePath(toks []token, field bool, r *names) (path, bool) {
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
			p.attr, p.hasAttr = r.test(toks[1].text, true), true
			toks = nil
		case '.':
			toks = toks[1:]
		case 'n':
			p.steps = append(p.steps, r.test(toks[0].text, false))
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
