package validate

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kud360/goxsd8/regex"
	"github.com/kud360/goxsd8/xsd"
)

// This file evaluates the RESTRICTED path subset an identity constraint's
// {selector} and {fields} are written in — the ·selector subset· of §3.11.6.2
// (c-selector-xpath) and the ·field subset· of §3.11.6.3 (c-fields-xpaths) —
// and nothing wider. It is deliberately NOT a bridge to the XPath 2.0
// evaluator: the productions below are a path grammar over the child and
// attribute axes with no predicates, no functions and no other axis, so
// evaluating them directly is exact where a fail-open delegation to a general
// engine would be a guess.
//
//	[1] Selector ::= Path ( '|' Path )*
//	[2] Path     ::= ('.' '//')? Step ( '/' Step )*
//	[3] Step     ::= '.' | NameTest
//	[4] NameTest ::= QName | '*' | NCName ':*'
//	[7] Path     ::= ('.' '//')? ( Step '/' )* ( Step | '@' NameTest )   (fields only)
//
// Production [7] is the whole of the fields/selector difference: only a field's
// FINAL step may name an attribute.
//
// Evaluation is incremental, because the walk that drives it is streaming
// ([Children] is a pull cursor): an [icExpr] never sees a tree. [icExpr.start]
// opens an expression at its context node, [icExpr.advance] carries the live
// step indices one level down as the descent enters a child, and a path whose
// last step matches reports the selection at that child. The leading `.//` is
// the only unbounded construct, and it is handled by re-seeding step 0 at every
// level rather than by remembering a depth.
//
// GAP(xpath): an {expression} outside these two productions — a legal XPath 2.0
// path this subset does not admit, an unbound prefix, a `.//` with no element
// step left once the self steps are removed — is DECLINED by [icCompile], and
// the identity constraint carrying it charges nothing at all
// (cvcidentityconstraint.go's icCheck.open). Charging on a path this file
// cannot read would reject a document for a gap in the processor. The
// withheld value's whole consumer set is Result.violations and its one reader
// Result.Violations, both of which carry violations PRESENT, so withholding one
// can only cost a rejection and never manufacture one. c-selector-xpath and
// c-fields-xpaths are the schema-side rules that would reject such an
// {expression} at assembly, and neither is implemented
// (xsd/identityconstraint.go), so nothing upstream has narrowed what reaches
// here. #812 owns its retirement.

// icNameTest is one NameTest of production [4], with its prefix ALREADY
// resolved: the {namespace bindings} of the XPath Expression property record
// that carried the expression resolve a prefixed name, and an unprefixed one
// resolves per the AXIS it was reached by — the {default namespace} for an
// element step, no namespace at all for an attribute step (PRINCIPLES 15,
// XPath 2.0 §3.2.1.2: "otherwise, it has no namespace URI"). That asymmetry is
// settled here, at compile time, so [icNameTest.matches] is a comparison of
// ·expanded names· and carries no axis of its own.
type icNameTest struct {
	space    string
	local    string
	anyLocal bool // 'NCName:*'
	anySpace bool // bare '*', which is any namespace AND any local name
}

// matches reports whether the test admits the ·expanded name· n.
func (t icNameTest) matches(n xsd.QName) bool {
	if t.anySpace {
		return true
	}
	if t.space != n.Space {
		return false
	}
	return t.anyLocal || t.local == n.Local
}

// icPath is one Path of production [2] or [7], with the `.` self steps already
// removed: a self step selects the node the path is already at, so it changes
// nothing about which nodes the path selects, and keeping it would leave every
// step index off by an amount the matcher would have to re-derive.
//
// A path with no steps left and no attr selects its own CONTEXT node (`.`); one
// with no steps and an attr selects an attribute of the context node (`@id`).
// anyDepth with no steps is rejected at compile time rather than modeled:
// `.//.` is descendant-or-self over every node kind, which production [3] does
// not otherwise reach and which this file would have to guess at.
type icPath struct {
	anyDepth bool
	steps    []icNameTest
	attr     icNameTest
	hasAttr  bool
}

// icExpr is one compiled {selector} or {fields} member: the union of its Paths,
// in the order they were written, so what a level reports never varies between
// runs (STYLE D2).
type icExpr struct{ paths []icPath }

// icLive is one expression's live state at one level of the descent: live[i]
// holds the step indices path i is waiting to match against the NEXT element
// down, ascending and deduplicated. A path selecting its context node holds no
// live index — it has no step to advance — and [icExpr.self] reports it
// instead.
type icLive [][]int

// start opens x at its context node, with every path waiting on its first step.
//
// A `.//` path is seeded EMPTY, because icCandidates adds step 0 back at every
// level including this one. Seeding it here as well would put 0 in the
// candidate set twice, and every index derived from it twice again one level
// down — a live set doubling per level rather than one holding each step index
// once.
func (x icExpr) start() icLive {
	live := make(icLive, len(x.paths))
	for i := range x.paths {
		if len(x.paths[i].steps) > 0 && !x.paths[i].anyDepth {
			live[i] = []int{0}
		}
	}
	return live
}

// self reports what x selects at its own CONTEXT node: whether some path
// selects the node itself (`.`), and the attribute NameTests of the paths that
// select one of its attributes (`@id`). Both are the zero-step case, which
// advance never reaches — it only ever looks one level down.
func (x icExpr) self() (element bool, attrs []icNameTest) {
	for _, p := range x.paths {
		if len(p.steps) > 0 {
			continue
		}
		if p.hasAttr {
			attrs = append(attrs, p.attr)
			continue
		}
		element = true
	}
	return element, attrs
}

// advance carries live one level down, into a child element whose ·expanded
// name· is n, and reports what that child is selected as: the selected node
// itself for a path whose last step matched, or the attribute NameTests to
// apply to the child's [[attributes]] for a field path whose element steps all
// matched.
//
// A path whose anyDepth is set re-seeds step 0 at EVERY level, which is the
// whole of `('.' '//')?`: descendant-or-self::node()/child::Step matches the
// first step at any depth at or below the context node's children, and every
// later step relative to wherever that match landed.
func (x icExpr) advance(live icLive, n xsd.QName) (next icLive, element bool, attrs []icNameTest) {
	next = make(icLive, len(x.paths))
	for i, p := range x.paths {
		if len(p.steps) == 0 {
			continue
		}
		for _, j := range icCandidates(live[i], p.anyDepth) {
			if !p.steps[j].matches(n) {
				continue
			}
			if j+1 == len(p.steps) {
				if p.hasAttr {
					attrs = append(attrs, p.attr)
					continue
				}
				element = true
				continue
			}
			next[i] = append(next[i], j+1)
		}
	}
	return next, element, attrs
}

// icCandidates is the step indices one path tries against the next element
// down: the ones carried from the level above, plus step 0 again for a `.//`
// path. The seed is prepended rather than appended so the result stays
// ascending, which is what keeps next ascending and duplicate-free without a
// sort — a carried index is always at least 1, so live never holds 0.
func icCandidates(live []int, anyDepth bool) []int {
	if !anyDepth {
		return live
	}
	return append([]int{0}, live...)
}

// icNames resolves the NameTests of one XPath Expression property record: its
// {namespace bindings} by prefix, and its {default namespace} for an unprefixed
// name on an ELEMENT step. The map is internal and never iterated into output
// (STYLE D2) — it is read by prefix and nothing else.
type icNames struct {
	prefixes map[string]string
	def      string
	hasDef   bool
}

// test resolves one NameTest's text. attribute selects the axis, and with it
// what an unprefixed name means: no namespace on the attribute axis, the
// {default namespace} on the element axis. A prefix with no binding is not a
// name this file can resolve, and the whole expression declines.
func (r icNames) test(text string, attribute bool) (icNameTest, bool) {
	if text == "*" {
		return icNameTest{anySpace: true, anyLocal: true}, true
	}
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		if attribute || !r.hasDef {
			return icNameTest{local: text}, true
		}
		return icNameTest{space: r.def, local: text}, true
	}
	space, bound := r.prefixes[prefix]
	if !bound {
		return icNameTest{}, false
	}
	if local == "*" {
		return icNameTest{space: space, anyLocal: true}, true
	}
	return icNameTest{space: space, local: local}, true
}

// icCompile compiles one XPath Expression property record into the subset
// matcher, reporting false for an {expression} the subset does not admit. field
// selects production [7] over production [2]: only a field may end in an
// attribute step.
func icCompile(x xsd.XPathExpression, field bool) (icExpr, bool) {
	toks, ok := icTokenize(x.Expression())
	if !ok {
		return icExpr{}, false
	}
	names := icNames{prefixes: make(map[string]string)}
	for _, b := range x.NamespaceBindings() {
		names.prefixes[b.Prefix()] = b.Namespace()
	}
	names.def, names.hasDef = x.DefaultNamespace()
	return icParse(toks, field, names)
}

// icToken is one token of §3.11.6.2's lexical production, identified by the
// character that opens it — '.', '/', 'D' for '//', '|', '@', or 'n' for a
// NameTest, whose text is carried.
type icToken struct {
	kind byte
	text string
}

// icTokenize splits an {expression} into the tokens §3.11.6.2 lists — "token
// ::= '.' | '/' | '//' | '|' | '@' | NameTest", longest-token first, with white
// space allowed around tokens though not inside them.
//
// Longest-token is load-bearing in one place a shorter rule gets wrong: '.' is
// a legal NCName character after the first, so "a.b" is ONE NameTest and not a
// name, a self step and a second name. The scan reaches that by trying a
// NameTest first wherever one can start, which '.' cannot: an NCName opens with
// \i (Datatypes §3.4.7.1's pattern facet), and \i is XML's NameStartChar
// (§G.4.2.5), which does not admit '.'.
func icTokenize(s string) ([]icToken, bool) {
	var toks []icToken
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.IsSpace(r):
			i += size
		case r == '|', r == '@', r == '.':
			toks = append(toks, icToken{kind: byte(r)})
			i += size
		case r == '/':
			if strings.HasPrefix(s[i:], "//") {
				toks = append(toks, icToken{kind: 'D'})
				i += 2
				continue
			}
			toks = append(toks, icToken{kind: '/'})
			i++
		case r == '*':
			toks = append(toks, icToken{kind: 'n', text: "*"})
			i += size
		default:
			j := icScanNameTest(s, i)
			if j == i {
				return nil, false
			}
			toks = append(toks, icToken{kind: 'n', text: s[i:j]})
			i = j
		}
	}
	return toks, true
}

// icScanNameTest reports the end of the NameTest starting at i, or i itself
// where none does. It admits the shapes of production [4] that can start with a
// name character — QName, prefixed or not, and 'NCName:*' — the bare '*' being
// handled by the tokenizer before it gets here.
func icScanNameTest(s string, i int) int {
	j := icScanNCName(s, i)
	if j == i || j >= len(s) || s[j] != ':' {
		return j
	}
	if j+1 < len(s) && s[j+1] == '*' {
		return j + 2
	}
	k := icScanNCName(s, j+1)
	if k == j+1 {
		return i // 'p:' with no local part is no NameTest at all
	}
	return k
}

// icNCNameRE matches the longest NCName at the START of the string it is
// applied to — [Namespaces in XML] production [4], spelled as the pattern
// Datatypes §3.4.7.1 fixes for xs:NCName, "[\i-[:]][\c-[:]]*". It is
// translated and compiled once here through [regex.Translate], so the code
// points behind \i and \c are the ones the regex package owns and not a second
// table (PRINCIPLES 26/27; regex/class.go records which edition of XML
// supplies them).
//
// The flavor is FO because only FO's '^' is a real anchor: FlavorXSD anchors
// the WHOLE string, which cannot express a prefix scan. This pattern carries
// no construct the two flavors read differently.
var icNCNameRE = func() *regexp.Regexp {
	goRE, err := regex.Translate(`^[\i-[:]][\c-[:]]*`, regex.FlavorFO, "")
	if err != nil {
		panic("validate: translating the NCName pattern: " + err.Error())
	}
	return regexp.MustCompile(goRE)
}()

// icScanNCName reports the end of the NCName starting at i, or i where none
// does. Datatypes §G.4.2.5 defines \i and \c by direct reference to XML's
// NameStartChar and NameChar, so the class here is the grammar's own and not an
// approximation of it — which is what a PREFIX has to have. The name this
// bounds is resolved against the {namespace bindings} in scope (PRINCIPLES 15),
// and a scan admitting one character too many would carry into the prefix a
// character that ends the name and opens the next token.
//
// The match is anchored at i, so its length IS the end it reports. A miss and
// an empty match are the same "" here, and mean the same thing: the pattern
// requires a NameStartChar, so it never matches empty.
func icScanNCName(s string, i int) int {
	return i + len(icNCNameRE.FindString(s[i:]))
}

// icParse parses the token stream as production [1]'s union of Paths, declining
// an empty union and any Path the grammar does not admit.
func icParse(toks []icToken, field bool, names icNames) (icExpr, bool) {
	var x icExpr
	start := 0
	for i := 0; i <= len(toks); i++ {
		if i < len(toks) && toks[i].kind != '|' {
			continue
		}
		p, ok := icParsePath(toks[start:i], field, names)
		if !ok {
			return icExpr{}, false
		}
		x.paths = append(x.paths, p)
		start = i + 1
	}
	if len(x.paths) == 0 {
		return icExpr{}, false
	}
	return x, true
}

// icParsePath parses one Path: the optional `.//` prefix, then Steps separated
// by '/', with an '@' NameTest admitted as the final step for a field alone.
//
// A `.` Step is dropped as it is read (see [icPath]). The one shape dropping
// cannot handle is a `.//` path whose every Step was a `.`, which leaves no
// element step to match at any depth; it is declined rather than guessed at.
func icParsePath(toks []icToken, field bool, names icNames) (icPath, bool) {
	var p icPath
	if len(toks) >= 2 && toks[0].kind == '.' && toks[1].kind == 'D' {
		p.anyDepth = true
		toks = toks[2:]
	}
	if len(toks) == 0 {
		return icPath{}, false
	}
	for len(toks) > 0 {
		switch toks[0].kind {
		case '@':
			if !field || len(toks) != 2 || toks[1].kind != 'n' {
				return icPath{}, false
			}
			t, ok := names.test(toks[1].text, true)
			if !ok {
				return icPath{}, false
			}
			p.attr, p.hasAttr = t, true
			toks = nil
		case '.':
			toks = toks[1:]
		case 'n':
			t, ok := names.test(toks[0].text, false)
			if !ok {
				return icPath{}, false
			}
			p.steps = append(p.steps, t)
			toks = toks[1:]
		default:
			return icPath{}, false
		}
		if len(toks) == 0 {
			break
		}
		if toks[0].kind != '/' {
			return icPath{}, false
		}
		toks = toks[1:]
		if len(toks) == 0 {
			return icPath{}, false // a trailing '/' has no Step after it
		}
	}
	if p.anyDepth && len(p.steps) == 0 {
		return icPath{}, false
	}
	return p, true
}
