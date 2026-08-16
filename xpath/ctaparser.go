package xpath

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kud360/goxsd8/xsd"
)

// This file is the lexer and the recursive-descent parser for the §3.12.6
// required subset, one of each (STYLE T4, xpath/doc.go's "There is never a
// second, lenient parser"). Every method below is named for the production it
// parses, and the whole grammar is reached: [15] ta-CastExpr's `cast as` tail
// and [18] ta-ConstructorFunction, which this engine does not yet EVALUATE,
// are consumed by their own production and then declined there, so each
// decline is attributable to its construct rather than falling out of a
// generic syntax failure, and retiring either is a change to that production
// alone. The third declined construct, a wildcard NameTest, is NOT
// production-level: `*` is no token kind, so no wildcard spelling survives
// ctaTokenize to reach [17] ta-AttrName, and its marker sits on the tokenizer
// arm that declines it.

// ctaFunctionNS is the default function namespace of a {test}'s static context
// (xpath-valid clause 2.2.4, §3.13.6.2), which an unprefixed [12]
// ta-BooleanFunction name resolves in — so a bare not(...) is in the subset
// without any binding in the record's {namespace bindings}.
const ctaFunctionNS = "http://www.w3.org/2005/xpath-functions"

// ctaNotFunction is the ONE function name [12] ta-BooleanFunction may carry:
// §3.12.6 clause 3, "Any strings matching the BooleanFunction production are
// function calls to fn:not".
var ctaNotFunction = xsd.QName{Space: ctaFunctionNS, Local: "not"}

// ctaNames resolves the QNames of one XPath Expression property record against
// its {namespace bindings}. The map is internal and never iterated into output
// (STYLE D2) — it is read by prefix and nothing else.
//
// The record's {default namespace} is absent from the struct on purpose: it is
// the default ELEMENT/TYPE namespace, and this grammar reaches neither an
// element step nor (once the cast productions are declined) a type name.
type ctaNames struct{ prefixes map[string]string }

// attribute resolves a NameTest of [17] ta-AttrName to an ·expanded name·. An
// unprefixed one is in NO namespace: the attribute axis's principal node kind
// is attribute, never element, so xpath20.md §3.2.1.2's "otherwise, it has no
// namespace URI" applies and the {default namespace} is not consulted
// (PRINCIPLES 15). An unbound prefix is err:XPST0081 and declines the whole
// expression.
func (n ctaNames) attribute(text string) (xsd.QName, bool) {
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		return xsd.QName{Local: text}, true
	}
	space, bound := n.prefixes[prefix]
	if !bound {
		return xsd.QName{}, false
	}
	return xsd.QName{Space: space, Local: local}, true
}

// function resolves a function QName, whose unprefixed form takes the default
// FUNCTION namespace (xpath-valid clause 2.2.4) rather than the no-namespace
// answer an attribute NameTest gets.
func (n ctaNames) function(text string) (xsd.QName, bool) {
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		return xsd.QName{Space: ctaFunctionNS, Local: text}, true
	}
	space, bound := n.prefixes[prefix]
	if !bound {
		return xsd.QName{}, false
	}
	return xsd.QName{Space: space, Local: local}, true
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
			// GAP(xpath): a WILDCARD NameTest — `@*`, `@p:*`, `@*:n` — is
			// declined LEXICALLY, here: `*` is no token kind of this subset, so
			// `@*` and `@*:n` die on this arm and `@p:*` one token later on the
			// ':' arm. [17]'s NameTest is xpath20.md's [37], which does admit the
			// wildcards, but a wildcard selects a SEQUENCE of attributes and
			// [AttributeValue] answers one ·expanded name· at a time, so
			// evaluating one would need a wider read of E.[[attributes]] than the
			// surface exposes. Retiring it takes a token kind here AND a NameTest
			// production, not a change to attrName alone (#859). The decline
			// withholds the element's ·governing type definition· on the terms
			// validate/cta.go argues.
			j := ctaScanQName(s, i)
			if j == i {
				return nil, false
			}
			toks = append(toks, ctaToken{kind: ctaNameTok, text: s[i:j]})
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

// ctaScanQName reports the end of the QName starting at i, or i itself where
// none does: an NCName, optionally followed by ':' and a second NCName. The
// ':' is consumed only when a name really follows it, so 'attribute::x' leaves
// the '::' for the axis token.
func ctaScanQName(s string, i int) int {
	j := ctaScanNCName(s, i)
	if j == i || j >= len(s) || s[j] != ':' {
		return j
	}
	k := ctaScanNCName(s, j+1)
	if k == j+1 {
		return j
	}
	return k
}

// ctaScanNCName reports the end of the NCName starting at i, or i where none
// does. The character classes are Datatypes §3.4.7's \i and \c narrowed to
// what Unicode's own categories decide: a name character this admits that XML
// would not is never the difference between two documents' verdicts, because
// the name still has to MATCH an ·expanded name· the source produced.
func ctaScanNCName(s string, i int) int {
	r, size := utf8.DecodeRuneInString(s[i:])
	if !unicode.IsLetter(r) && r != '_' {
		return i
	}
	j := i + size
	for j < len(s) {
		r, size = utf8.DecodeRuneInString(s[j:])
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '-' && r != '_' {
			break
		}
		j += size
	}
	return j
}

// ctaParser is the recursive-descent parser over the token stream. It carries
// no visited set and no depth guard (STYLE D4): the grammar has no back edge,
// so a descent over a finite token slice terminates by construction.
type ctaParser struct {
	toks  []ctaToken
	pos   int
	names ctaNames
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
	if p.at(ctaNameTok) && p.peek(1).kind == ctaLParen {
		name, resolved := p.names.function(p.peek(0).text)
		if !resolved {
			return nil, false
		}
		if name == ctaNotFunction {
			return p.booleanFunction()
		}
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
	space, admitted := ctaComparisonSpace(left, right)
	if !admitted {
		return ctaTypeError{}, true
	}
	return ctaCompare{op: op, space: space, left: left, right: right}, true
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
// GAP(xpath): a `cast as QName ?` tail is recognized and DECLINED (#858). The
// cast is to a built-in datatype (§3.12.6 clause 4) and wants value's lexical
// mapping over the untypedAtomic operand; until #858 supplies it, the whole
// expression is unevaluable and CompileCTATest withholds. The tail is parsed
// out in full rather than left to fail as a syntax error, so #858 replaces the
// decline below with the cast and touches nothing else.
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
	p.advance()
	if p.at(ctaQuestionTok) {
		p.advance()
	}
	return nil, false
}

// constructorFunction parses [18] ta-ConstructorFunction.
//
// GAP(xpath): a constructor function is recognized and DECLINED (#858).
// §3.12.6 clause 3 makes every QName '(' SimpleValue ')' whose name is not
// fn:not a constructor call for a built-in datatype, so this one production
// covers both the constructor spelling of a cast and the "unknown boolean
// function" reading — there is no third case to distinguish. xpath20.md
// §3.10.4 defines T($arg) as (($arg) cast as T?), which is why #858 retires
// this decline and castExpr's together.
func (p *ctaParser) constructorFunction() (ctaValue, bool) {
	p.advance() // the function name
	p.advance() // '('
	if _, ok := p.simpleValue(); !ok {
		return nil, false
	}
	if !p.at(ctaRParen) {
		return nil, false
	}
	p.advance()
	return nil, false
}

// simpleValue parses [16] ta-SimpleValue's two arms.
func (p *ctaParser) simpleValue() (ctaValue, bool) {
	switch p.peek(0).kind {
	case ctaAtTok, ctaNameTok:
		return p.attrName()
	case ctaStringTok:
		text := p.peek(0).text
		p.advance()
		return ctaLiteral{text: text, space: ctaSpaceString}, true
	case ctaNumberTok:
		text := p.peek(0).text
		p.advance()
		return ctaLiteral{text: text, space: ctaNumericSpace(text)}, true
	default:
		return nil, false
	}
}

// ctaNumericSpace is the value space a NumericLiteral's own type puts it in: a
// DoubleLiteral (the one with an exponent, xpath20.md [73]) is xs:double, and
// an IntegerLiteral or DecimalLiteral is xs:decimal — xs:integer's primitive
// base, and so the least common type of any two of them.
func ctaNumericSpace(text string) ctaSpace {
	if strings.ContainsAny(text, "eE") {
		return ctaSpaceDouble
	}
	return ctaSpaceDecimal
}

// attrName parses [17] ta-AttrName in BOTH spellings ta-props-correct clause 2
// admits, which is disjunctive: clause 2.1's abbreviated `'@' NameTest`, and
// clause 2.2's "XPath expression involving the attribute axis whose
// abbreviated form is as given above", i.e. `attribute::NameTest`.
//
// The NameTest admitted here is a QName and nothing else. A wildcard one never
// arrives to be declined: ctaTokenize has no token kind for `*` and rejects
// every wildcard spelling lexically, which is where that gap is marked.
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
	if !p.at(ctaNameTok) {
		return nil, false
	}
	text := p.peek(0).text
	p.advance()
	name, resolved := p.names.attribute(text)
	if !resolved {
		return nil, false
	}
	return ctaAttr{name: name}, true
}
