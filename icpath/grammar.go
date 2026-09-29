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
// The order the three states are decided in is the whole of "charge what
// [SelectorViolation] enumerates, decline everything else":
//
//   - UNSUPPORTED DOMINATES. A stream carrying a run this lexer cannot read is
//     declined whatever else it holds, before any shape is judged. That is what
//     keeps a quoted "[" inside an expression this package does not read from
//     being charged as a predicate, and it is why
//     `p:a/processing-instruction('x')` with p unbound is a decline and not a
//     clause-1 charge: a KindTest with an argument does not lex, and reading its
//     prefix in isolation would reject a schema whose only fault is a spelling
//     this compiler does not read.
//   - A SHAPE VIOLATION IS INDEPENDENT OF THE PARSE. Over a fully lexed stream
//     shapeFault proves its shapes from the tokens alone, so `a[b]` is charged
//     although it parses to nothing at all.
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
// carried. The other kinds are NOT production [5]'s. '[' and ']' are a
// predicate's brackets, which the grammar admits nowhere. The three axis heads
// are an XPath 2.0 axis keyword and its '::' read as one token: 'C' for the
// child axis, 'A' for the attribute axis — the two clause 2.2 of
// c-fields-xpaths names — and 'X' for any of the other eleven; each carries
// its keyword. 'K' is an argument-free KindTest (xpath20 production [54]), such
// as `node()` or `text()`, carrying its spelling with the parentheses closed up.
// 'W' is a name split by white space around its ':' — `tid :*`, `tid : x`, or an
// axis keyword split from the second ':' of its '::' as in `child: :` — carrying
// its source spelling. It is no XPath 2.0 token at all and has no axis reading,
// whatever its NCName: scanSplitName documents the two runs it reads. 'F' is a
// whole FunctionCall (xpath20 production [48]) of the narrow form
// scanFunctionCall reads, carrying its source spelling. '=' is the GeneralComp
// operator (production [22]) and 's' a StringLiteral (production [74]) carrying
// its spelling, delimiters included; the lexer yields either only between a
// predicate's brackets. '?' is one rune or run that opens no token of any sort,
// and exists so the lexer is total.
type token struct {
	kind byte
	text string
}

// tokenize splits an {expression} into the tokens production [5] lists — "token
// ::= '.' | '/' | '//' | '|' | '@' | NameTest" — longest-token first, with
// production [6]'s white space allowed around tokens though not inside them —
// and into the axis heads and the argument-free KindTests scanAxisOrKind reads,
// which are what clause 2.2's unabbreviated spellings and the residual shapes
// charged beside them are written in, the FunctionCalls scanFunctionCall reads,
// and the white-space-split names scanSplitName reads, which are charged as no
// XPath 2.0 expression. A FunctionCall and a split name are tried before
// scanNameTest, which would otherwise read the NCName alone and leave the '(' or
// the ':' to open no token.
//
// '=' and a StringLiteral are read only between a '[' and its ']', where the
// predicate they sit in is charged whatever they hold. Outside a predicate
// neither completes any charged shape, and reading one there would hand a
// leading '//' or a selector's '@' a stream such as `//='x'` to charge under
// clause 2 although it is no XPath 2.0 expression; so there each opens no token,
// and the stream is declined as before.
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
	depth := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.IsSpace(r):
			i += size
		case r == '|', r == '@', r == '.', r == '[', r == ']':
			toks = append(toks, token{kind: byte(r)})
			i += size
			if r == '[' {
				depth++
			}
			if r == ']' && depth > 0 {
				depth--
			}
		case r == '=' && depth > 0:
			toks = append(toks, token{kind: '='})
			i += size
		case (r == '"' || r == '\'') && depth > 0:
			j, ok := scanStringLiteral(s, i)
			if !ok {
				toks = append(toks, token{kind: '?'})
				i += size
				continue
			}
			toks = append(toks, token{kind: 's', text: s[i:j]})
			i = j
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
			if t, j, ok := scanAxisOrKind(s, i); ok {
				toks = append(toks, t)
				i = j
				continue
			}
			if j, ok := scanFunctionCall(s, i); ok {
				toks = append(toks, token{kind: 'F', text: s[i:j]})
				i = j
				continue
			}
			if j, ok := scanSplitName(s, i); ok {
				toks = append(toks, token{kind: 'W', text: s[i:j]})
				i = j
				continue
			}
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

// scanAxisOrKind reads, at i, the two runs of XPath 2.0 this lexer reads beyond
// production [5]: an axis head, `NCName S? '::'`, and an argument-free KindTest,
// `KindKeyword S? "(" S? ")"` over kindKeyword's vocabulary. It reports ok false
// where neither starts at i, and the caller scans a NameTest there instead.
//
// Axis-ness is decided by the '::' and never by the name, and it is tested
// first, so `child` alone is still an element NameTest and `attribute::` an axis
// head while `attribute()` is a KindTest; and white space is admitted on both
// sides of the '::', because in XPath 2.0 an axis keyword and '::' are two
// tokens (production [30]'s `("child" "::")`). A KindTest with an argument, whose
// QNames, TypeNames and StringLiterals this reader does not read, is not read
// here, and scanFunctionCall declines its reserved keyword too: its '(' opens no
// token.
func scanAxisOrKind(s string, i int) (token, int, bool) {
	j := scanNCName(s, i)
	if j == i {
		return token{}, i, false
	}
	k := skipSpace(s, j)
	if strings.HasPrefix(s[k:], "::") {
		return axisHead(s[i:j]), k + 2, true
	}
	if !kindKeyword(s[i:j]) || !strings.HasPrefix(s[k:], "(") {
		return token{}, i, false
	}
	k = skipSpace(s, k+1)
	if !strings.HasPrefix(s[k:], ")") {
		return token{}, i, false
	}
	return token{kind: 'K', text: s[i:j] + "()"}, k + 1, true
}

// scanSplitName reports the end of the white-space-split name starting at i,
// and ok false where none does. It reads exactly two runs, each holding white
// space no XPath 2.0 token admits:
//
//   - `NCName S? ':' S? (NCName | '*')` with at least one S non-empty — a QName
//     (xpath20 production [78]) or a Wildcard (production [37], ws: explicit)
//     split around its ':', as in `tid :*`, `tid: *` and `tid : x`;
//   - `NCName S? ':' S ':'` — an axis keyword whose '::', one terminal of
//     production [30] and [33], is split in two, as in `child: :`. What follows
//     it lexes as usual.
//
// Both runs need white space beside a lone ':', so neither is ever an unsplit
// `a:b` or `tid:*`, nor an axis head, whose '::' is unsplit, nor a KindTest.
// Any other colon residue — `a:b :c`, `p:`, `:a` — is left to open no token.
func scanSplitName(s string, i int) (int, bool) {
	j := scanNCName(s, i)
	if j == i {
		return i, false
	}
	k := skipSpace(s, j)
	if !strings.HasPrefix(s[k:], ":") {
		return i, false
	}
	m := skipSpace(s, k+1)
	if strings.HasPrefix(s[m:], ":") && m > k+1 {
		return m + 1, true
	}
	if k == j && m == k+1 {
		return i, false
	}
	if strings.HasPrefix(s[m:], "*") {
		return m + 1, true
	}
	if n := scanNCName(s, m); n > m {
		return n, true
	}
	return i, false
}

// scanFunctionCall reports the end of the FunctionCall starting at i, and ok
// false where none does. It reads xpath20 production [48], `QName "("
// (ExprSingle ("," ExprSingle)*)? ")"`, narrowed to what is XPath 2.0 whatever
// the static context holds:
//
//   - the name is an NCName, never a prefixed QName, because a prefix this lexer
//     read would be resolved nowhere, and an unbound one is clause 1's
//     err:XPST0081 and not the clause-2 charge shapeFault makes;
//   - the name is none of A.3's reserved function names, which before a '(' are a
//     KindTest or a keyword and never a FunctionCall — so `element(a)` and
//     `processing-instruction('x')` are not read here;
//   - every argument is a StringLiteral (production [74]), the one ExprSingle
//     read without a parser. A comment `(:` after the '(' is none, so `f (: c :)`
//     is not read as a call either (xpath20 A.1.3, gn: parens).
func scanFunctionCall(s string, i int) (int, bool) {
	j := scanNCName(s, i)
	if j == i || reservedFunctionName(s[i:j]) {
		return i, false
	}
	k := skipSpace(s, j)
	if !strings.HasPrefix(s[k:], "(") {
		return i, false
	}
	k = skipSpace(s, k+1)
	if strings.HasPrefix(s[k:], ")") {
		return k + 1, true
	}
	for {
		m, ok := scanStringLiteral(s, k)
		if !ok {
			return i, false
		}
		k = skipSpace(s, m)
		if strings.HasPrefix(s[k:], ")") {
			return k + 1, true
		}
		if !strings.HasPrefix(s[k:], ",") {
			return i, false
		}
		k = skipSpace(s, k+1)
	}
}

// scanStringLiteral reports the end of the StringLiteral starting at i, and ok
// false where none does: xpath20 production [74], a run delimited by a quotation
// mark or an apostrophe, in which the delimiter doubled is the escape of
// productions [75] and [76] and not the end. An unterminated literal is none.
func scanStringLiteral(s string, i int) (int, bool) {
	if i >= len(s) || (s[i] != '"' && s[i] != '\'') {
		return i, false
	}
	q := s[i]
	for j := i + 1; j < len(s); j++ {
		if s[j] != q {
			continue
		}
		if j+1 < len(s) && s[j+1] == q {
			j++
			continue
		}
		return j + 1, true
	}
	return i, false
}

// reservedFunctionName reports whether name is one of xpath20 A.3's thirteen
// Reserved Function Names, which "are not allowed as function names in an
// unprefixed form because expression syntax takes precedence".
func reservedFunctionName(name string) bool {
	switch name {
	case "attribute", "comment", "document-node", "element", "empty-sequence", "if", "item",
		"node", "processing-instruction", "schema-attribute", "schema-element", "text", "typeswitch":
		return true
	}
	return false
}

// kindKeyword reports whether name opens a KindTest that admits empty
// parentheses: the switch is the whole of the xpath20 production [54] KindTest
// alternatives whose argument is optional or absent — productions [55] to [60]
// and [64] — a closed vocabulary like axisHead's. `schema-element` and
// `schema-attribute` (productions [66] and [62]) require an argument and are not
// in it. xpath20 A.3 reserves all seven as function names, so none of them
// before `()` is ever a FunctionCall.
func kindKeyword(name string) bool {
	switch name {
	case "node", "text", "comment", "processing-instruction", "document-node", "element", "attribute":
		return true
	}
	return false
}

// axisHead classifies the name read before an axis head's '::'. The switch is
// the whole of xpath20 production [30] ForwardAxis and production [33]
// ReverseAxis, thirteen keywords in a closed vocabulary, so every axis reaches
// shapeFault classified. A name that is none of them is one '?' token: no axis
// of XPath 2.0 spells it.
func axisHead(name string) token {
	switch name {
	case "child":
		return token{kind: 'C', text: name}
	case "attribute":
		return token{kind: 'A', text: name}
	case "descendant", "self", "descendant-or-self", "following-sibling", "following", "namespace", // [30]
		"parent", "ancestor", "preceding-sibling", "preceding", "ancestor-or-self": // [33]
		return token{kind: 'X', text: name}
	}
	return token{kind: '?'}
}

// skipSpace reports the end of the white space starting at i, by the one
// predicate tokenize skips white space with.
func skipSpace(s string, i int) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsSpace(r) {
			return i
		}
		i += size
	}
	return i
}

// fullyLexed reports whether every rune of the {expression} the stream came from
// fell inside a token this package reads — production [5]'s, a predicate
// bracket, an axis head, an argument-free KindTest, a FunctionCall, a
// white-space-split name, or a '=' or StringLiteral inside a predicate.
// Nothing is charged over a stream that fails this: see compile.
func fullyLexed(toks []token) bool {
	for _, t := range toks {
		if t.kind == '?' {
			return false
		}
	}
	return true
}

// shapeFault reports the production violation a FULLY LEXED token stream proves,
// in the words the charge names it with, and ok false where it proves none.
//
// Every shape it decides fails clause 2 under both of its arms, or fails clause
// 1 outright. A predicate is one abbreviation neither introduces nor removes. An
// attribute step is named for a selector by neither arm, and admitted for a
// field only as a Path's final step. An axis head other than child — or, for a
// field, attribute — is one clause 2.2 does not name, and clause 2.1 spells no
// axis at all. A KindTest step is no NameTest (xpath20 production [35]), and
// production [3] Step spells none, nor a FunctionCall, which xpath20 production
// [27] makes a FilterExpr and no AxisStep. A root-relative path, opened by '/' or '//',
// is one production [2] and [7]'s context-relative Path never is. A '//'
// anywhere but the leading './/' pair is §3.2.4 rule 3's descendant-or-self
// step, which neither arm admits. Five shapes are no XPath 2.0 expression under
// any spelling, which fails clause 1: an '@' or an axis head with no NodeTest
// after it; an empty union member, which is the empty {expression} and an
// operand missing on either side of '|'; a '/' or '//' with no Step after it,
// save a '/' that is the whole member; and a white-space-split name.
//
// Where one member holds several of these faults, the LEFTMOST in token order is
// the one charged: `self::node()` is charged for its axis and not its KindTest,
// `/@a` in a selector for its opening '/' and not its attribute, and `/ /a` for
// its first '/', which no Step follows, and not as root-relative.
//
// What it does NOT decide is a child-axis or attribute-axis head clause 2.2
// admits before a NameTest, which parse compiles onto the arm its abbreviated
// spelling takes. Nor does it charge two Steps with no separator between them
// (`. .`, `a b`): the lexer reads the legal parent step `..` as the same two
// '.' tokens, so the parse declines all three.
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
		// An empty member is no XPath 2.0 expression at all: production [2]
		// Expr is one ExprSingle or more, and production [14] UnionExpr needs
		// an operand on both sides of '|'. That covers the empty or white-space
		// {expression} as well as `| a` and `a||b`.
		if len(m) == 0 {
			return fmt.Sprintf("has an empty operand where a Path is required, but %s clause 1 requires it to satisfy xpath-valid, whose clause 1 requires a valid XPath 2.0 expression — xpath20 production [2] Expr is never empty, and production [14] UnionExpr needs an operand on both sides of '|'", scc), true
		}
		for j, t := range m {
			// A name split by white space around its ':' is no XPath 2.0
			// expression at all: a QName (xpath20 production [78]) admits no
			// white space, a Wildcard (production [37]) is ws: explicit, and '::'
			// is one terminal of production [30] and [33]. It is charged here,
			// before parse resolves any prefix, so its NCName is never read as a
			// prefix and charged unbound.
			if t.kind == 'W' {
				return fmt.Sprintf("has a name %q split by white space, but %s clause 1 requires it to satisfy xpath-valid, whose clause 1 requires a valid XPath 2.0 expression — xpath20 production [78] QName admits no white space, production [37] Wildcard is ws: explicit, and '::' is one terminal", t.text, scc), true
			}
			// A '/' or '//' with no Step after it — the end of the member, or
			// another '/' or '//' — is no XPath 2.0 expression at all: production
			// [26] RelativePathExpr puts a StepExpr after every '/' and '//', and
			// production [25] PathExpr admits a leading '//' only before one and a
			// leading '/' alone only as the whole path. That lone '/' is legal, and
			// is charged below as root-relative. Deciding this first at every
			// token is what leaves the two arms below only separators a Step
			// follows.
			if (t.kind == '/' || t.kind == 'D') && !stepAfter(m, j) && !loneRoot(m) {
				return fmt.Sprintf("has a %q with no Step after it, but %s clause 1 requires it to satisfy xpath-valid, whose clause 1 requires a valid XPath 2.0 expression — xpath20 production [26] RelativePathExpr puts a StepExpr after every '/' and '//', and production [25] PathExpr admits a lone '/' only as the whole path", spelling(t), scc), true
			}
			// A leading '/' or '//' makes the path root-relative (xpath20
			// production [25] PathExpr and the expansions under it), and
			// production [2] and [7]'s Path is context-relative, abbreviated or
			// not: the only '//' either opens with is the one in the './/' pair.
			if j == 0 && (t.kind == '/' || t.kind == 'D') {
				return fmt.Sprintf("is root-relative (it opens with %q), but %s clause 2 admits it under neither arm — xpath20 production [25] expands a leading %q to a step from the root, and production %s's Path is context-relative, its only leading '//' the './/' pair", spelling(t), scc, spelling(t), pathProduction(field)), true
			}
			// Every '//' but the leading './/' pair is non-initial, which xpath20
			// §3.2.4 rule 3 expands to /descendant-or-self::node()/: an axis clause
			// 2.2 does not name, in a place production [2] and [7] admit no '//'.
			if j > 0 && t.kind == 'D' && (j != 1 || m[0].kind != '.') {
				return fmt.Sprintf("has a non-initial '//', but %s clause 2 admits it under neither arm — xpath20 §3.2.4 rule 3 expands it to /descendant-or-self::node()/, and production %s admits '//' only in its leading './/' pair", scc, pathProduction(field)), true
			}
			// A KindTest is no NameTest (xpath20 production [35] NodeTest is
			// `KindTest | NameTest`), production [3] Step is '.' or a NameTest, and
			// no §3.2.4 abbreviation turns the one into the other — whatever axis
			// head, '@' or '/' precedes it.
			if t.kind == 'K' {
				return fmt.Sprintf("has a KindTest step %q, but %s clause 2 admits it under neither arm — %s, and xpath20 production [35] makes a KindTest no NameTest", t.text, scc, stepProductions(field)), true
			}
			// A FunctionCall is a PrimaryExpr (xpath20 production [41]), a
			// FilterExpr's and never an AxisStep's (production [27]), so it is no
			// Step of production [3] and no §3.2.4 abbreviation of one.
			if t.kind == 'F' {
				return fmt.Sprintf("has a FunctionCall %q, but %s clause 2 admits it under neither arm — %s, and xpath20 production [27] makes a FunctionCall a FilterExpr and no AxisStep", t.text, scc, stepProductions(field)), true
			}
			head := t.kind == 'C' || t.kind == 'A' || t.kind == 'X'
			// An axis head with no NodeTest after it is no XPath 2.0 expression at
			// all: production [29] ForwardStep is `ForwardAxis NodeTest` and
			// production [32] ReverseStep `ReverseAxis NodeTest`, and the NodeTest
			// is mandatory on both. That fails xpath-valid clause 1, and with it
			// clause 1 of the SCC, before clause 2's arms are reached.
			if head && !nodeTestAfter(m, j) {
				return fmt.Sprintf("has an axis step %q with no NodeTest after it, but %s clause 1 requires it to satisfy xpath-valid, whose clause 1 requires a valid XPath 2.0 expression — production [29] ForwardStep and production [32] ReverseStep are an axis then a NodeTest, and the NodeTest is mandatory", t.text+"::", scc), true
			}
			// Clause 2.2 names the child axis for a selector and the child and
			// attribute axes for a field, and nothing else; clause 2.1's tokens
			// (production [5]) spell no axis at all. So every other axis of
			// productions [30] and [33] fails both arms, whichever NodeTest follows.
			if t.kind == 'X' {
				return fmt.Sprintf("steps along the %s axis, but %s clause 2 admits it under neither arm — production [5] spells no axis, and clause 2.2 names %s", t.text, scc, namedAxes(field)), true
			}
			if t.kind != '@' && t.kind != 'A' {
				continue
			}
			// Production [2] has no '@' at all and clause 2.2 names the child axis
			// alone, so an attribute anywhere in a SELECTOR is outside both arms,
			// under either spelling.
			if !field {
				return fmt.Sprintf("names an attribute, but %s clause 2 admits only the child axis — production [2] has no '@' and clause 2.2 names no other axis", scc), true
			}
			// An '@' with no NodeTest after it is no XPath 2.0 expression at all:
			// production [31] AbbrevForwardStep is `"@"? NodeTest`, and its
			// NodeTest is mandatory. That fails xpath-valid clause 1, and with it
			// c-fields-xpaths clause 1. The spelled-out attribute head was charged
			// above, on production [29]'s terms.
			if !nodeTestAfter(m, j) {
				return fmt.Sprintf("has an '@' with no NodeTest after it, but %s clause 1 requires it to satisfy xpath-valid, whose clause 1 requires a valid XPath 2.0 expression — production [31] AbbrevForwardStep is \"@\"? NodeTest, and its NodeTest is mandatory", scc), true
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

// nodeTestAfter reports whether the token after m[j] is a NodeTest (xpath20
// production [35], `KindTest | NameTest`): a NameTest, or an argument-free
// KindTest.
func nodeTestAfter(m []token, j int) bool {
	return j+1 < len(m) && (m[j+1].kind == 'n' || m[j+1].kind == 'K')
}

// stepAfter reports whether a token opening a Step follows m[j]: anything but
// the end of the member, a '/' or a '//'.
func stepAfter(m []token, j int) bool {
	return j+1 < len(m) && m[j+1].kind != '/' && m[j+1].kind != 'D'
}

// loneRoot reports whether the member is `/` alone: the root, and legal XPath
// 2.0 (xpath20 production [25]) although no Step follows its '/'.
func loneRoot(m []token) bool {
	return len(m) == 1 && m[0].kind == '/'
}

// spelling is the text of a '/' or '//' token, as a charge names it.
func spelling(t token) string {
	if t.kind == 'D' {
		return "//"
	}
	return "/"
}

// pathProduction is the Path production governing one kind of {expression}.
func pathProduction(field bool) string {
	if field {
		return "[7]"
	}
	return "[2]"
}

// stepProductions is what production [3] and, for a field, production [7]'s
// final step admit, as the KindTest charge reports it.
func stepProductions(field bool) string {
	if field {
		return "production [3] Step is '.' or a NameTest and production [7]'s final step is '@' NameTest"
	}
	return "production [3] Step is '.' or a NameTest"
}

// namedAxes is what clause 2.2 of the SCC over one kind of {expression} names,
// as the clause-2 charge on any other axis reports it.
func namedAxes(field bool) string {
	if field {
		return "the child and attribute axes alone"
	}
	return "the child axis alone"
}

// unionMembers splits a token stream on production [1]'s '|' into its Paths. An
// empty member is yielded like any other, and an empty stream is one empty
// member: shapeFault charges it, and parse declines it.
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

// ncNamePattern is the pattern Datatypes §3.4.7.1 fixes for xs:NCName,
// "[\i-[:]][\c-[:]]*", behind the '^' a prefix scan needs. It is a local copy
// of the generated builtin row's value rather than a production import of
// builtin, and TestNCNamePatternPinned fails if the two diverge (PRINCIPLES
// 26/27).
const ncNamePattern = `^[\i-[:]][\c-[:]]*`

// ncNameRE matches the longest NCName at the START of the string it is applied
// to — [Namespaces in XML] production [4], spelled as ncNamePattern. It is
// translated and compiled once here through [regex.Translate], so the code
// points behind \i and \c are the ones the regex package owns and not a second
// table (PRINCIPLES 26/27; regex/class.go records which edition of XML supplies
// them).
//
// The flavor is FO because only FO's '^' is a real anchor: FlavorXSD anchors the
// WHOLE string, which cannot express a prefix scan. This pattern carries no
// construct the two flavors read differently.
var ncNameRE = func() *regexp.Regexp {
	goRE, err := regex.Translate(ncNamePattern, regex.FlavorFO, "")
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
// empty union and any Path the grammar does not admit. It is TOTAL over any
// stream: compile reaches it only after shapeFault has charged the empty member
// and the separator with no Step after it, and it still declines both rather
// than rely on that order.
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
// The unabbreviated spellings clause 2.2 admits take the arms their abbreviated
// twins take — `child::` NameTest the NameTest arm, `attribute::` NameTest the
// '@' arm — so each compiles to the tree its abbreviated spelling compiles to,
// by construction. Every other axis head, and every KindTest or FunctionCall
// after an admitted one, is charged before the parse is reached (shapeFault).
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
		return path{}, false // an empty member or `.//`: shapeFault charged both first
	}
	for len(toks) > 0 {
		// A child-axis head before a NameTest is the unabbreviated spelling of
		// that NameTest alone (xpath20 §3.2.4 rule 2: "If the axis name is
		// omitted from an axis step, the default axis is child"), so dropping
		// the head leaves the step its abbreviated twin reads, on the 'n' arm
		// below. Before anything else it stays, and declines at the default arm.
		if len(toks) >= 2 && toks[0].kind == 'C' && toks[1].kind == 'n' {
			toks = toks[1:]
		}
		switch toks[0].kind {
		// An attribute-axis head is the unabbreviated spelling of '@' (xpath20
		// §3.2.4 rule 1: "The attribute axis attribute:: can be abbreviated by
		// @"), and takes its arm.
		case '@', 'A':
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
			return path{}, false // shapeFault charged it first; kept so parse stays total
		}
	}
	if p.anyDepth && len(p.steps) == 0 {
		return path{}, false
	}
	return p, true
}
