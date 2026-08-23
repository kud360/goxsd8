package xpath

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kud360/goxsd8/regex"
	"github.com/kud360/goxsd8/xsd"
)

// This file is the lexer and the recursive-descent parser for the §3.12.6
// required subset, one of each (STYLE T4, xpath/doc.go's "There is never a
// second, lenient parser"). Every method below is named for the production it
// parses, and the whole grammar is both reached and evaluated: no method here
// is a stub, because no compile-time decline is production-level. xpath/doc.go
// owns the enumeration of what declines; every decline reaching this file is
// ctaTypes answering ctaTypeDeclined for a comparison type or a cast target it
// will not serve, which the production that asked propagates unchanged.

// ctaFunctionNS is the default function namespace of a {test}'s static context
// (xpath-valid clause 2.2.4, §3.13.6.2), which an unprefixed [12]
// ta-BooleanFunction name resolves in — so a bare not(...) is in the subset
// without any binding in the record's {namespace bindings}.
const ctaFunctionNS = "http://www.w3.org/2005/xpath-functions"

// ctaNotFunction is the ONE function name [12] ta-BooleanFunction may carry:
// §3.12.6 clause 3, "Any strings matching the BooleanFunction production are
// function calls to fn:not".
var ctaNotFunction = xsd.QName{Space: ctaFunctionNS, Local: "not"}

// ctaNames holds the {namespace bindings} and the {default namespace} of one
// XPath Expression property record, the bindings indexed by prefix. The map is
// internal and never iterated into output (STYLE D2) — it is read by prefix and
// nothing else, and the diagnostic an unbound prefix produces names the prefix
// the parse walk reached, never one this map yielded.
//
// The record's {default namespace} is the default ELEMENT/TYPE namespace, so
// it answers for exactly one production here: [15] ta-CastExpr's target QName,
// which xpath20.md §3.10.2 puts in it ("if the target type has no namespace
// prefix, it is considered to be in the default element/type namespace"). An
// absent {default namespace} is the empty string, which is the no-namespace
// answer that case wants anyway. An unprefixed attribute NameTest and an
// unprefixed function name take their own answers, and neither is this one.
type ctaNames struct {
	prefixes         map[string]string
	defaultNamespace string
}

// ctaUnresolvedName is the ·expanded name· an UNBOUND prefix resolves to, so
// that the parse continues far enough to decide whether the rest of the
// expression is a complete [8] ta-Test — which is the whole of what tells a
// static error apart from an unsupported construct.
//
// It never escapes into an evaluable tree: recording the defect and building
// this name are one step, compileCTATest reports that defect, and
// [CompileCTATest] withholds on any defect at all. No QName the grammar can
// write equals it either, since ctaScanNameTest admits no empty local part.
// The wildcard NameTest has no such uninhabited value to take and uses a sum
// arm instead (ctaUnresolvedTest).
var ctaUnresolvedName = xsd.QName{}

// attributeName resolves the QName arm of [17] ta-AttrName's NameTest to an
// ·expanded name·. An unprefixed one is in NO namespace: the attribute axis's
// principal node kind is attribute, never element, so xpath20.md §3.2.1.2's
// "otherwise, it has no namespace URI" applies and the {default namespace} is
// not consulted (PRINCIPLES 15).
func (p *ctaParser) attributeName(text string) xsd.QName {
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		return xsd.QName{Local: text}
	}
	return p.prefixedName(prefix, local)
}

// wildcardTest resolves the WILDCARD arm of [17] ta-AttrName's NameTest —
// xpath20.md's [37], all three spellings — into the matcher it names.
//
// Only `NCName ':' '*'` resolves a prefix, and so only it can be err:XPST0081;
// `*` and `'*' ':' NCName` name no prefix at all and match across namespaces
// (§3.2.1.2).
func (p *ctaParser) wildcardTest(text string) ctaNameTest {
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		return ctaAnyName{}
	}
	if prefix == "*" {
		return ctaAnySpace{local: local}
	}
	space, bound := p.prefixedSpace(prefix)
	if !bound {
		return ctaUnresolvedTest{}
	}
	return ctaAnyLocal{space: space}
}

// typeName resolves the QName naming a datatype on attributeName's terms,
// except that its unprefixed form takes the default ELEMENT/TYPE namespace
// (xpath20.md §3.10.2) rather than the no-namespace answer an attribute
// NameTest gets or the function-namespace one a function name gets.
//
// The err:XPST0081 an unbound prefix records here is always DISCARDED, because
// ctaTypes.castTarget resolves no type under ctaUnresolvedName and so declines
// the whole expression before the parse can reach the end of a [8] ta-Test.
// That is one more cast-target static condition this engine does not charge,
// under the marker ctaTypes.castTarget carries.
func (p *ctaParser) typeName(text string) xsd.QName {
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		return xsd.QName{Space: p.names.defaultNamespace, Local: text}
	}
	return p.prefixedName(prefix, local)
}

// functionName resolves a function QName on attributeName's terms, except that
// its unprefixed form takes the default FUNCTION namespace (xpath-valid clause
// 2.2.4) rather than the no-namespace answer an attribute NameTest gets.
func (p *ctaParser) functionName(text string) xsd.QName {
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		return xsd.QName{Space: ctaFunctionNS, Local: text}
	}
	return p.prefixedName(prefix, local)
}

// prefixedName resolves a PREFIXED QName against the {namespace bindings},
// recording err:XPST0081 for a prefix with no binding among them and yielding
// ctaUnresolvedName in its place.
func (p *ctaParser) prefixedName(prefix, local string) xsd.QName {
	space, bound := p.prefixedSpace(prefix)
	if !bound {
		return ctaUnresolvedName
	}
	return xsd.QName{Space: space, Local: local}
}

// prefixedSpace resolves a prefix to the NAMESPACE it is bound to, recording
// err:XPST0081 for one with no binding on prefixedName's terms and reporting ok
// false — [37]'s `NCName ':' '*'` carries no local part to pair it with, and no
// namespace URI is uninhabited, so the answer is the reported false and never a
// sentinel URI a real attribute could match (ctaUnresolvedTest).
func (p *ctaParser) prefixedSpace(prefix string) (string, bool) {
	space, bound := p.names.prefixes[prefix]
	if !bound {
		p.recordUnbound(prefix)
		return "", false
	}
	return space, true
}

// recordUnbound records the err:XPST0081 an unbound prefix is (xpath20.md
// Appendix G: "a static error if a QName used in an expression contains a
// namespace prefix that cannot be expanded into a namespace URI") and lets the
// parse go on.
//
// The FIRST unbound prefix decides the message, so the answer is the one the
// walk reaches first in expression order and not a map's (STYLE D2).
//
// The record is PROVISIONAL: compileCTATest keeps it only where the parse then
// reached the end of a complete [8] ta-Test, because an expression outside the
// required subset is declined rather than charged, however its names resolve.
func (p *ctaParser) recordUnbound(prefix string) {
	if p.defect.kind != ctaNoDefect {
		return
	}
	p.defect = ctaDefect{
		kind:   ctaStaticError,
		detail: fmt.Sprintf("err:XPST0081: no in-scope namespace binding for prefix %q", prefix),
	}
}

// ctaKind identifies one token of the subset's lexical structure.
type ctaKind byte

const (
	// ctaEOF is the sentinel past the last token; the token slice never holds
	// one, and ctaParser.peek synthesizes it.
	ctaEOF ctaKind = iota
	// ctaNameTok is an NCName or a prefixed QName, whose text is as written.
	// The keywords 'or', 'and', 'cast', 'as' and the 'attribute' axis name are
	// this kind too: XPath has no reserved words, so what a name means is the
	// parser's to decide from position.
	ctaNameTok
	// ctaWildcardTok is one [37] Wildcard — `*`, `NCName ':' '*'` or
	// `'*' ':' NCName` — whose text is as written. It is its own kind and not a
	// ctaNameTok carrying a `*`, because ctaNameTok also carries the keywords and
	// the axis name and a wildcard reaches none of those positions: attrName
	// accepts this kind and no other production does, so `1 * 2` stays a decline
	// and no multiplicative production is implied.
	ctaWildcardTok
	// ctaStringTok is a StringLiteral, whose text is its VALUE — quotes
	// stripped, doubled quotes folded to one.
	ctaStringTok
	// ctaNumberTok is a NumericLiteral, whose text is as written.
	ctaNumberTok
	ctaLParen
	ctaRParen
	// ctaAtTok is '@', the abbreviation ta-props-correct clause 2.1 spells
	// [17] ta-AttrName with.
	ctaAtTok
	// ctaAxisTok is '::', which only the unabbreviated attribute-axis form
	// clause 2.2 admits reaches.
	ctaAxisTok
	// ctaQuestionTok is the '?' occurrence indicator of [15] ta-CastExpr.
	ctaQuestionTok
	// ctaCompTok is one operator of [13] ta-Comparator, whose text is its
	// spelling.
	ctaCompTok
)

// ctaToken is one token, identified by kind. text carries the source spelling
// for the three kinds that have one (name, number) or the decoded value
// (string), and is empty for the punctuation kinds.
type ctaToken struct {
	kind ctaKind
	text string
}

// ctaTokenize splits an {expression} into tokens, reporting false for text
// whose lexical structure the subset does not admit at all.
//
// White space between tokens is skipped per §3.12.6's Note ("[XPath 2.0]
// allows whitespace to be used between tokens ... even though this is not
// explicitly shown in the grammar"), and so are XPath comments, which nest
// (xpath20.md §A.2.4.1) and are legal wherever white space is.
func ctaTokenize(s string) ([]ctaToken, bool) {
	var toks []ctaToken
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case unicode.IsSpace(r):
			i += size
		case strings.HasPrefix(s[i:], "(:"):
			j, ok := ctaSkipComment(s, i)
			if !ok {
				return nil, false
			}
			i = j
		case r == '(':
			toks = append(toks, ctaToken{kind: ctaLParen})
			i++
		case r == ')':
			toks = append(toks, ctaToken{kind: ctaRParen})
			i++
		case r == '@':
			toks = append(toks, ctaToken{kind: ctaAtTok})
			i++
		case r == '?':
			toks = append(toks, ctaToken{kind: ctaQuestionTok})
			i++
		case r == ':':
			if !strings.HasPrefix(s[i:], "::") {
				return nil, false
			}
			toks = append(toks, ctaToken{kind: ctaAxisTok})
			i += 2
		case r == '=', r == '!', r == '<', r == '>':
			tok, j, ok := ctaScanComparator(s, i)
			if !ok {
				return nil, false
			}
			toks = append(toks, tok)
			i = j
		case r == '\'', r == '"':
			text, j, ok := ctaScanString(s, i)
			if !ok {
				return nil, false
			}
			toks = append(toks, ctaToken{kind: ctaStringTok, text: text})
			i = j
		case r >= '0' && r <= '9', r == '.':
			j := ctaScanNumber(s, i)
			if j == i {
				return nil, false
			}
			toks = append(toks, ctaToken{kind: ctaNumberTok, text: s[i:j]})
			i = j
		default:
			kind, j := ctaScanNameTest(s, i)
			if j == i {
				return nil, false
			}
			toks = append(toks, ctaToken{kind: kind, text: s[i:j]})
			i = j
		}
	}
	return toks, true
}

// ctaSkipComment reports the index past the comment opening at i, or false for
// one left unclosed. Comments nest, so the scan counts depth rather than
// stopping at the first ":)".
func ctaSkipComment(s string, i int) (int, bool) {
	depth := 0
	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "(:"):
			depth++
			i += 2
		case strings.HasPrefix(s[i:], ":)"):
			depth--
			i += 2
			if depth == 0 {
				return i, true
			}
		default:
			_, size := utf8.DecodeRuneInString(s[i:])
			i += size
		}
	}
	return 0, false
}

// ctaScanComparator reads one [13] ta-Comparator at i, longest first so '<='
// is never read as '<' followed by '='. '!' opens only '!='.
func ctaScanComparator(s string, i int) (ctaToken, int, bool) {
	for _, op := range []string{"!=", "<=", ">=", "=", "<", ">"} {
		if strings.HasPrefix(s[i:], op) {
			return ctaToken{kind: ctaCompTok, text: op}, i + len(op), true
		}
	}
	return ctaToken{}, 0, false
}

// ctaScanString reads the StringLiteral opening at i (xpath20.md [74]),
// returning its VALUE: the quote character may appear inside the literal only
// doubled, and a doubled pair denotes one such character.
func ctaScanString(s string, i int) (string, int, bool) {
	quote := s[i]
	var b strings.Builder
	for j := i + 1; j < len(s); {
		if s[j] != quote {
			_, size := utf8.DecodeRuneInString(s[j:])
			b.WriteString(s[j : j+size])
			j += size
			continue
		}
		if j+1 < len(s) && s[j+1] == quote {
			b.WriteByte(quote)
			j += 2
			continue
		}
		return b.String(), j + 1, true
	}
	return "", 0, false
}

// ctaScanNumber reports the end of the NumericLiteral starting at i, or i
// itself where none does. It admits xpath20.md's [71] IntegerLiteral, [72]
// DecimalLiteral and [73] DoubleLiteral, which share one scan: digits, an
// optional fractional part, an optional exponent — with at least one digit in
// the mantissa.
//
// No sign is admitted, and that is the grammar's doing rather than an
// omission: [16] ta-SimpleValue reaches a Literal directly, with no unary
// operator production between them, so "-1" is two tokens the subset has no
// rule for.
func ctaScanNumber(s string, i int) int {
	j := ctaScanDigits(s, i)
	if j < len(s) && s[j] == '.' {
		j = ctaScanDigits(s, j+1)
	}
	if j == i || (j == i+1 && s[i] == '.') {
		return i
	}
	if j >= len(s) || (s[j] != 'e' && s[j] != 'E') {
		return j
	}
	k := j + 1
	if k < len(s) && (s[k] == '+' || s[k] == '-') {
		k++
	}
	end := ctaScanDigits(s, k)
	if end == k {
		return j // 'e' with no exponent digits is not part of the literal
	}
	return end
}

// ctaScanDigits reports the end of the run of ASCII digits at i.
func ctaScanDigits(s string, i int) int {
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	return i
}

// ctaScanNameTest reports the kind and the end of the [36] NameTest starting at
// i, or i itself where none does. One scanner reads both arms (STYLE T4),
// because they share their first NCName and differ only past the ':':
//
//   - `NCName` and `NCName ':' NCName` are ctaNameTok, the QName arm — and the
//     spelling every keyword and the axis name arrives under.
//   - `*`, `NCName ':' '*'` and `'*' ':' NCName` are ctaWildcardTok, [37]'s
//     three arms.
//
// The ':' is consumed only when a name or a `*` really follows it, so
// 'attribute::x' leaves the '::' for the axis token and 'attribute::*' leaves
// it for the axis token followed by a bare wildcard. No white space may sit
// inside a NameTest, which is what keeps `@p : name` out of the grammar.
func ctaScanNameTest(s string, i int) (ctaKind, int) {
	if s[i] == '*' {
		return ctaWildcardTok, ctaScanWildcardTail(s, i+1)
	}
	j := ctaScanNCName(s, i)
	if j == i || j >= len(s) || s[j] != ':' {
		return ctaNameTok, j
	}
	if j+1 < len(s) && s[j+1] == '*' {
		return ctaWildcardTok, j + 2
	}
	k := ctaScanNCName(s, j+1)
	if k == j+1 {
		return ctaNameTok, j
	}
	return ctaNameTok, k
}

// ctaScanWildcardTail reports the end of the optional `':' NCName` following a
// leading '*', or i itself where none does.
func ctaScanWildcardTail(s string, i int) int {
	if i >= len(s) || s[i] != ':' {
		return i
	}
	j := ctaScanNCName(s, i+1)
	if j == i+1 {
		return i
	}
	return j
}

// ctaNCNameRE matches the longest NCName at the START of the string it is
// applied to — [Namespaces in XML] production [4], spelled as the pattern
// Datatypes §3.4.7.1 fixes for xs:NCName, "[\i-[:]][\c-[:]]*". It is
// translated and compiled once here through [regex.Translate], so the code
// points behind \i and \c are the ones the regex package owns and not a second
// table (PRINCIPLES 26/27; parser and builtin/strict reach the same class the
// same way, and regex/class.go records which edition of XML supplies it).
//
// The flavor is FO because only FO's '^' is a real anchor: FlavorXSD anchors
// the WHOLE string, which cannot express a prefix scan. This pattern carries
// no construct the two flavors read differently.
var ctaNCNameRE = func() *regexp.Regexp {
	goRE, err := regex.Translate(`^[\i-[:]][\c-[:]]*`, regex.FlavorFO, "")
	if err != nil {
		panic("xpath: translating the NCName pattern: " + err.Error())
	}
	return regexp.MustCompile(goRE)
}()

// ctaScanNCName reports the end of the NCName starting at i, or i where none
// does. Its character classes are EXACT and not an approximation of the
// grammar: Datatypes §G.4.2.5 defines \i and \c by direct reference to XML's
// NameStartChar and NameChar, so the boundary reported here is the boundary
// XML draws — no character that terminates a name is read into one, and none
// that continues a name terminates it.
//
// The match is anchored at i, so its length IS the end it reports. A miss and
// an empty match are the same "" here, and mean the same thing: the pattern
// requires a NameStartChar, so it never matches empty.
func ctaScanNCName(s string, i int) int {
	return i + len(ctaNCNameRE.FindString(s[i:]))
}

// ctaParser is the recursive-descent parser over the token stream. It carries
// no visited set and no depth guard (STYLE D4): the grammar has no back edge,
// so a descent over a finite token slice terminates by construction.
type ctaParser struct {
	toks  []ctaToken
	pos   int
	names ctaNames
	types ctaTypes
	// defect is what name resolution found wrong, which is the one defect kind
	// the walk itself has to carry: every other one is the false a production
	// returns. It lives here rather than on ctaNames because a value receiver
	// cannot keep it.
	defect ctaDefect
}

// peek reports the token at offset ahead of the cursor, or the EOF sentinel.
func (p *ctaParser) peek(ahead int) ctaToken {
	if p.pos+ahead >= len(p.toks) {
		return ctaToken{kind: ctaEOF}
	}
	return p.toks[p.pos+ahead]
}

// at reports whether the cursor sits on a token of kind k.
func (p *ctaParser) at(k ctaKind) bool { return p.peek(0).kind == k }

// atName reports whether the cursor sits on the unprefixed name text, which is
// how the keywords 'or', 'and', 'cast', 'as' and the 'attribute' axis are
// recognized.
func (p *ctaParser) atName(text string) bool {
	return p.peek(0).kind == ctaNameTok && p.peek(0).text == text
}

// advance moves the cursor past one token.
func (p *ctaParser) advance() { p.pos++ }

// test parses [8] ta-Test, which is one OrExpr and then the end of the
// expression: trailing tokens are not a Test, however well the prefix parsed.
func (p *ctaParser) test() (ctaExpr, bool) {
	x, ok := p.orExpr()
	if !ok {
		return nil, false
	}
	if !p.at(ctaEOF) {
		return nil, false
	}
	return x, true
}

// orExpr parses [9] ta-OrExpr. A single operand yields that operand rather
// than a one-armed ctaOr, so the tree holds no node that decides nothing.
func (p *ctaParser) orExpr() (ctaExpr, bool) {
	first, ok := p.andExpr()
	if !ok {
		return nil, false
	}
	operands := []ctaExpr{first}
	for p.atName("or") {
		p.advance()
		next, ok := p.andExpr()
		if !ok {
			return nil, false
		}
		operands = append(operands, next)
	}
	if len(operands) == 1 {
		return first, true
	}
	return ctaOr{operands: operands}, true
}

// andExpr parses [10] ta-AndExpr, on orExpr's terms.
func (p *ctaParser) andExpr() (ctaExpr, bool) {
	first, ok := p.booleanExpr()
	if !ok {
		return nil, false
	}
	operands := []ctaExpr{first}
	for p.atName("and") {
		p.advance()
		next, ok := p.booleanExpr()
		if !ok {
			return nil, false
		}
		operands = append(operands, next)
	}
	if len(operands) == 1 {
		return first, true
	}
	return ctaAnd{operands: operands}, true
}

// booleanExpr parses [11] ta-BooleanExpr's three arms.
//
// The second and third arms both admit `QName '('` — [12] ta-BooleanFunction
// and [18] ta-ConstructorFunction — and §3.12.6 clause 3's Note resolves that
// ambiguity by FUNCTION NAME, not by argument shape: a name that is fn:not
// takes the BooleanFunction arm, and every other name is a constructor call
// and therefore a ValueExpr of the third arm. There is no fourth reading in
// which an unknown name is its own error.
func (p *ctaParser) booleanExpr() (ctaExpr, bool) {
	if p.at(ctaLParen) {
		p.advance()
		x, ok := p.orExpr()
		if !ok {
			return nil, false
		}
		if !p.at(ctaRParen) {
			return nil, false
		}
		p.advance()
		return x, true
	}
	if p.at(ctaNameTok) && p.peek(1).kind == ctaLParen && p.functionName(p.peek(0).text) == ctaNotFunction {
		return p.booleanFunction()
	}
	left, ok := p.valueExpr()
	if !ok {
		return nil, false
	}
	op, compared := p.comparator()
	if !compared {
		return ctaEffectiveBoolean{operand: left}, true
	}
	right, ok := p.valueExpr()
	if !ok {
		return nil, false
	}
	comparison, typing := p.types.comparison(op, left, right)
	if typing == ctaTypeDeclined {
		return nil, false
	}
	if typing == ctaTypeErrored {
		return ctaTypeError{}, true
	}
	return ctaCompare{op: op, comparison: comparison, left: left, right: right}, true
}

// booleanFunction parses [12] ta-BooleanFunction, whose name the caller has
// already resolved to fn:not.
func (p *ctaParser) booleanFunction() (ctaExpr, bool) {
	p.advance() // the function name
	p.advance() // '('
	arg, ok := p.orExpr()
	if !ok {
		return nil, false
	}
	if !p.at(ctaRParen) {
		return nil, false
	}
	p.advance()
	return ctaNot{operand: arg}, true
}

// comparator reads one [13] ta-Comparator, reporting false where the cursor is
// on something else — which is [11]'s optional-comparator arm, not an error.
func (p *ctaParser) comparator() (ctaComparator, bool) {
	if !p.at(ctaCompTok) {
		return ctaEqual, false
	}
	text := p.peek(0).text
	p.advance()
	switch text {
	case "=":
		return ctaEqual, true
	case "!=":
		return ctaNotEqual, true
	case "<":
		return ctaLess, true
	case "<=":
		return ctaLessEqual, true
	case ">":
		return ctaGreater, true
	case ">=":
		return ctaGreaterEqual, true
	}
	return ctaEqual, false
}

// valueExpr parses [14] ta-ValueExpr, dispatching on whether a function call
// opens it.
func (p *ctaParser) valueExpr() (ctaValue, bool) {
	if p.at(ctaNameTok) && p.peek(1).kind == ctaLParen {
		return p.constructorFunction()
	}
	return p.castExpr()
}

// castExpr parses [15] ta-CastExpr: a SimpleValue and an optional cast tail.
//
// The tail's QName is resolved as a TYPE name and classified before the node
// is built, so a target this engine does not cast to declines the whole
// expression here rather than deciding anything at evaluation time
// (ctaTypes.castTarget). §3.12.6 clause 4 fixes what an admitted one is: "Any
// explicit casts (i.e. any strings which match the optional "cast as" QName in
// the CastExpr production) are casts to built-in datatypes."
func (p *ctaParser) castExpr() (ctaValue, bool) {
	v, ok := p.simpleValue()
	if !ok {
		return nil, false
	}
	if !p.atName("cast") {
		return v, true
	}
	p.advance()
	if !p.atName("as") {
		return nil, false
	}
	p.advance()
	if !p.at(ctaNameTok) {
		return nil, false
	}
	text := p.peek(0).text
	p.advance()
	allowsEmpty := false
	if p.at(ctaQuestionTok) {
		p.advance()
		allowsEmpty = true
	}
	target, admitted := p.types.castTarget(p.typeName(text))
	if !admitted {
		return nil, false
	}
	return ctaCast{operand: v, target: target, allowsEmpty: allowsEmpty}, true
}

// constructorFunction parses [18] ta-ConstructorFunction.
//
// §3.12.6 clause 3 makes every QName '(' SimpleValue ')' whose name is not
// fn:not a constructor call for a built-in datatype, so this one production
// covers both the constructor spelling of a cast and the "unknown boolean
// function" reading — there is no third case to distinguish. The name is a
// FUNCTION name, resolved as one, and it names the datatype at the same time:
// an unprefixed int(...) is fn:int, which declares no constructor, and never
// xs:int.
//
// The node is castExpr's, with allowsEmpty TRUE unconditionally: xpath20.md
// §3.10.4 defines T($arg) as (($arg) cast as T?), and the `?` is part of that
// equivalence however [18]'s own production is written — so a constructor call
// over an absent attribute is the empty sequence where the same cast written
// without `?` would be err:XPTY0004.
func (p *ctaParser) constructorFunction() (ctaValue, bool) {
	name := p.functionName(p.peek(0).text)
	p.advance() // the function name
	p.advance() // '('
	arg, ok := p.simpleValue()
	if !ok {
		return nil, false
	}
	if !p.at(ctaRParen) {
		return nil, false
	}
	p.advance()
	target, admitted := p.types.castTarget(name)
	if !admitted {
		return nil, false
	}
	return ctaCast{operand: arg, target: target, allowsEmpty: true}, true
}

// simpleValue parses [16] ta-SimpleValue's two arms.
func (p *ctaParser) simpleValue() (ctaValue, bool) {
	switch p.peek(0).kind {
	case ctaAtTok, ctaNameTok:
		return p.attrName()
	case ctaStringTok:
		text := p.peek(0).text
		p.advance()
		return ctaLiteral{text: text, st: p.types.str}, true
	case ctaNumberTok:
		text := p.peek(0).text
		p.advance()
		return ctaLiteral{text: text, st: p.types.literal(text)}, true
	default:
		return nil, false
	}
}

// attrName parses [17] ta-AttrName in BOTH spellings ta-props-correct clause 2
// admits, which is disjunctive: clause 2.1's abbreviated `'@' NameTest`, and
// clause 2.2's "XPath expression involving the attribute axis whose
// abbreviated form is as given above", i.e. `attribute::NameTest`.
//
// BOTH arms of [36] NameTest are admitted, in both spellings: the QName one,
// and xpath20.md's [37] Wildcard, which [17] reaches because it names NameTest
// rather than a QName. ctaWildcardTok is accepted HERE and by no other
// production, so no other position in this grammar admits a `*`.
func (p *ctaParser) attrName() (ctaValue, bool) {
	switch {
	case p.at(ctaAtTok):
		p.advance()
	case p.atName("attribute") && p.peek(1).kind == ctaAxisTok:
		p.advance()
		p.advance()
	default:
		return nil, false
	}
	text := p.peek(0).text
	switch p.peek(0).kind {
	case ctaNameTok:
		p.advance()
		return ctaAttr{test: ctaExactName{name: p.attributeName(text)}}, true
	case ctaWildcardTok:
		p.advance()
		return ctaAttr{test: p.wildcardTest(text)}, true
	default:
		return nil, false
	}
}
