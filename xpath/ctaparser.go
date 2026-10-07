package xpath

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kud360/goxsd8/regex"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is the lexer and the recursive-descent parser for the §3.12.6
// required subset, one of each (STYLE T4, xpath/doc.go's "There is never a
// second, lenient parser"), plus the productions beyond it an assertion's
// {test} reaches: xpath20.md [23] ValueComp, [44] VarRef, an abbreviated
// child-axis step whose NodeTest is a QName ([31] AbbrevForwardStep, §3.2.1.1),
// a "/" or "//" opening [25] PathExpr over one such step or one attribute step,
// a [26] RelativePathExpr of two or more such steps, or one element step `N`,
// `./N` or `.//N`, standing as the whole operand of fn:exists, fn:empty or an
// ·effective boolean value· (childPath, selectedElements), an fn:count call
// ([48] FunctionCall) over one counted path, a [40] Predicate on a child step
// in it (predicate) or a [21] UnionExpr of such paths (countArgument), or over
// an operand a library call takes as its argument (countCall), a call
// to one of the F&O string and sequence functions (libraryCall) whose arguments
// are additive expressions or `()`, the binary operators of [13] AdditiveExpr
// and [14] MultiplicativeExpr, and [47] ContextItemExpr `.`, which the facet
// façade admits, and the predicate façade reads as its candidate
// (valuePredicate) — each behind the façade (ctaFacade.comparesValues,
// ctaFacade.variable, ctaFacade.child, ctaFacade.childPath, ctaFacade.elements,
// ctaFacade.rooted, ctaFacade.count, ctaFacade.callsLibrary,
// ctaFacade.computes, ctaFacade.contextItem), so a Type Alternative's {test}
// reaches none of them. Every method below is named for the production it
// parses, and the whole grammar is both reached and evaluated: no method here
// is a stub, and the production-level declines are those ten façade methods'.
// xpath/doc.go owns the enumeration of what declines; every other decline
// reaching this file is ctaTypes answering ctaTypeDeclined for a comparison
// type, a cast target or a cast operand it will not serve, ctaTypes.arithmetic
// declining an operand pair, a library call of an arity its function does not
// have, a predicate or a union operand outside the shapes predicate,
// valuePredicate and ctaUnionOf admit, or the façade declining a NameTest, a
// variable's type or a settled comparison type, which the production that asked
// propagates unchanged.

// ctaFunctionNS is the default function namespace of a {test}'s static context
// (xpath-valid clause 2.2.4, §3.13.6.2), which an unprefixed [12]
// ta-BooleanFunction name resolves in — so a bare not(...) is in the subset
// without any binding in the record's {namespace bindings}.
const ctaFunctionNS = "http://www.w3.org/2005/xpath-functions"

// ctaNotFunction is the ONE function name [12] ta-BooleanFunction may carry:
// §3.12.6 clause 3, "Any strings matching the BooleanFunction production are
// function calls to fn:not".
var ctaNotFunction = xsd.QName{Space: ctaFunctionNS, Local: "not"}

// ctaCountFunction is fn:count (xpath-functions.md §15.4.1), which a [14]
// ta-ValueExpr calls on the assertion and facet façades (ctaParser.countCall);
// the other functions it calls but the constructors are
// ctaParser.libraryCall's.
var ctaCountFunction = xsd.QName{Space: ctaFunctionNS, Local: "count"}

// ctaNames holds the {namespace bindings} and the {default namespace} of one
// XPath Expression property record, the bindings indexed by prefix. The map is
// internal and never iterated into output (STYLE D2) — it is read by prefix and
// nothing else, and the diagnostic an unbound prefix produces names the prefix
// the parse walk reached, never one this map yielded.
//
// The record's {default namespace} is the default ELEMENT/TYPE namespace, so
// it answers for exactly two productions here: [15] ta-CastExpr's target QName,
// which xpath20.md §3.10.2 puts in it ("if the target type has no namespace
// prefix, it is considered to be in the default element/type namespace"), and
// the NameTest of a child-axis step (elementName). An absent {default
// namespace} is the empty string, which is the no-namespace answer both cases
// want then. An unprefixed attribute NameTest and an unprefixed function name
// take their own answers, and neither is this one.
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

// elementName resolves the QName NameTest of a child-axis step to an ·expanded
// name·. An unprefixed one takes the {default namespace}: the child axis's
// principal node kind is element, so xpath20.md §3.2.1.2 puts an unprefixed
// name test in "the default element/type namespace in the expression context",
// which xpath-valid clause 2.2.3 fixes as the {default namespace} — an
// assertion's xpathDefaultNamespace, already resolved (PRINCIPLES 15).
func (p *ctaParser) elementName(text string) xsd.QName {
	prefix, local, prefixed := strings.Cut(text, ":")
	if !prefixed {
		return xsd.QName{Space: p.names.defaultNamespace, Local: text}
	}
	return p.prefixedName(prefix, local)
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
// parse go on. The code is the recorded verdict's own rule (ruleXPST0081) and
// its [xsderr.Loc] is the zero one, this package holding no schema position.
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
		kind: ctaStaticError,
		static: xsderr.New(ruleXPST0081, xsderr.Loc{},
			"no statically-known namespace binding for prefix %q", prefix),
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
	// the axis name and a wildcard reaches none of those positions. Two
	// productions read it: attrName as a NameTest, and multiplicativeOperator
	// as the `*` operator where a bare `*` follows a complete operand, which is
	// a position no NameTest can take.
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
	// ctaDollarTok is the '$' opening xpath20.md [44] VarRef, which only the
	// assertion and facet façades' `$value` reaches (ctaFacade.variable), and
	// which opens an fn:count argument that is no path (ctaParser.countsItems).
	ctaDollarTok
	// ctaSlashTok is '/' and ctaSlashSlashTok is '//'. Each is read only where
	// it opens a [25] PathExpr (ctaParser.rootedPath) or follows the `.`
	// opening an fn:count argument or an element step whose existence is
	// asked (ctaParser.countPath), and a '/' also between the steps of a
	// child path (ctaParser.childPath); anywhere else it is a token no
	// production takes, so `a//b`, and `a/b` outside a child path's three
	// positions, are not expressions here.
	ctaSlashTok
	ctaSlashSlashTok
	// ctaDotTok is a '.' that opens no NumericLiteral: the [47]
	// ContextItemExpr, which the facet and predicate façades admit
	// (ctaFacade.contextItem), or the context item an fn:count argument's or
	// an element step's `./` or `.//` opens with (ctaParser.countPath).
	// '..', the abbreviated parent step, is not tokenized at all.
	ctaDotTok
	// ctaPlusTok is '+' and ctaMinusTok is '-', the two operators of xpath20.md
	// [13] AdditiveExpr (ctaParser.additiveOperator). Neither can open a name or
	// a number, so no other token is read across one; a '-' INSIDE a name is an
	// NCName character and never this token, so `a-1` is one name.
	ctaPlusTok
	ctaMinusTok
	// ctaCommaTok is ',', which separates the arguments of a call to one of the
	// F&O functions the façade admits (ctaParser.arguments) and is read nowhere
	// else, so a comma in any other position is a token no production takes.
	ctaCommaTok
	// ctaLBracketTok is '[' and ctaRBracketTok is ']', which open and close
	// xpath20.md [40] Predicate, read only after a child step of an fn:count
	// argument (ctaParser.predicate).
	ctaLBracketTok
	ctaRBracketTok
	// ctaBarTok is '|', xpath20.md [21] UnionExpr's operator, read only between
	// the operands of an fn:count argument (ctaParser.countArgument).
	ctaBarTok
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
		case r == '$':
			toks = append(toks, ctaToken{kind: ctaDollarTok})
			i++
		case r == '+':
			toks = append(toks, ctaToken{kind: ctaPlusTok})
			i++
		case r == '-':
			toks = append(toks, ctaToken{kind: ctaMinusTok})
			i++
		case r == ',':
			toks = append(toks, ctaToken{kind: ctaCommaTok})
			i++
		case r == '[':
			toks = append(toks, ctaToken{kind: ctaLBracketTok})
			i++
		case r == ']':
			toks = append(toks, ctaToken{kind: ctaRBracketTok})
			i++
		case r == '|':
			toks = append(toks, ctaToken{kind: ctaBarTok})
			i++
		case strings.HasPrefix(s[i:], "//"):
			toks = append(toks, ctaToken{kind: ctaSlashSlashTok})
			i += 2
		case r == '/':
			toks = append(toks, ctaToken{kind: ctaSlashTok})
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
			if j > i {
				toks = append(toks, ctaToken{kind: ctaNumberTok, text: s[i:j]})
				i = j
				continue
			}
			if r != '.' || strings.HasPrefix(s[i:], "..") {
				return nil, false
			}
			toks = append(toks, ctaToken{kind: ctaDotTok})
			i++
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
// operator production between them, so "-1" is a ctaMinusTok and a number,
// and no production takes a ctaMinusTok where an operand opens.
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

// ctaNCNamePattern is the pattern Datatypes §3.4.7.1 fixes for xs:NCName,
// "[\i-[:]][\c-[:]]*", behind the '^' a prefix scan needs. It is a local copy
// of the generated builtin row's value rather than a production import of
// builtin, and TestCTANCNamePatternPinned fails if the two diverge (PRINCIPLES
// 26/27).
const ctaNCNamePattern = `^[\i-[:]][\c-[:]]*`

// ctaNCNameRE matches the longest NCName at the START of the string it is
// applied to — [Namespaces in XML] production [4], spelled as
// ctaNCNamePattern. It is translated and compiled once here through
// [regex.Translate], so the code points behind \i and \c are the ones the
// regex package owns and not a second table (PRINCIPLES 26/27; parser and
// builtin/strict reach the same class the same way, and regex/class.go records
// which edition of XML supplies it).
//
// The flavor is FO because only FO's '^' is a real anchor: FlavorXSD anchors
// the WHOLE string, which cannot express a prefix scan. This pattern carries
// no construct the two flavors read differently.
var ctaNCNameRE = func() *regexp.Regexp {
	goRE, err := regex.Translate(ctaNCNamePattern, regex.FlavorFO, "")
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
	// facade compiles each [17] ta-AttrName, which is the one production whose
	// node differs between the façades, and admits each settled comparison type
	// (compileCTATest).
	facade ctaFacade
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
//
// The third arm with its Comparator absent also takes a child path
// (childPath) or one element step (selectedElements), whose ·effective boolean
// value· is whether it selects a node, where the path is the WHOLE ValueExpr:
// the token after it ends the BooleanExpr (closesBoolean). Followed by anything
// else — a comparator, an operator, a cast — it is left to additiveExpr, whose
// one child step leaves a '/' after it a token no production takes, and
// declines, and so does a `.` before a '/' or '//'; one step is childStep's,
// whose value is read.
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
	if n := p.childPathLength(0); n > 0 && p.closesBoolean(n) {
		path, ok := p.childPath(n)
		if !ok {
			return nil, false
		}
		return ctaEffectiveBoolean{operand: path}, true
	}
	if n := p.selectedStepLength(0); n > 0 && p.closesBoolean(n) {
		step, ok := p.selectedElements()
		if !ok {
			return nil, false
		}
		return ctaEffectiveBoolean{operand: step}, true
	}
	left, ok := p.additiveExpr()
	if !ok {
		return nil, false
	}
	if op, isValue := p.valueComparator(); isValue {
		return p.valueComparison(op, left)
	}
	op, compared := p.comparator()
	if !compared {
		return ctaEffectiveBoolean{operand: left}, true
	}
	right, ok := p.additiveExpr()
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

// valueComparator reads one xpath20.md [23] ValueComp, reporting false where
// the cursor is on anything else. Each spelling is the operator whose B.2 rows
// it reads — `eq` the row `A eq B`, and so on for the other five — which are
// the rows the [13] ta-Comparator spellings reach through §3.5.2's definition
// of a general comparison, so one operator type serves both.
//
// The six spellings are NCNames, and XPath has no reserved words, so what makes
// one an operator is its position: right after a [14] ta-ValueExpr no other
// production opens with a name but 'cast', 'and' and 'or', none of which is
// spelled like one.
func (p *ctaParser) valueComparator() (ctaComparator, bool) {
	if !p.at(ctaNameTok) {
		return ctaEqual, false
	}
	var op ctaComparator
	switch p.peek(0).text {
	case "eq":
		op = ctaEqual
	case "ne":
		op = ctaNotEqual
	case "lt":
		op = ctaLess
	case "le":
		op = ctaLessEqual
	case "gt":
		op = ctaGreater
	case "ge":
		op = ctaGreaterEqual
	default:
		return ctaEqual, false
	}
	p.advance()
	return op, true
}

// valueComparison parses the right operand of a value comparison whose left
// operand and operator are already read, and builds its node: xpath20.md [10]
// ComparisonExpr with its ValueComp arm, which §3.12.6's grammar does not
// have. The façade decides whether it is admitted at all
// (ctaFacade.comparesValues), so a Type Alternative's {test} declines it as
// outside its required subset.
//
// [10] admits ONE comparison per ComparisonExpr, so `@a eq @b eq @c` is not an
// expression; here the trailing `eq` is a token no production takes, and the
// parse ends unsupported as any other unparsed tail does.
//
// The type both operands are compared in is settled here, on §3.5.1's terms
// (ctaTypes.valueComparison), and a type it cannot be compared in is the
// err:XPTY0004 node a general comparison builds too, and a settled type is
// evaluated exactly as a general comparison's is (ctaHoldsPair).
func (p *ctaParser) valueComparison(op ctaComparator, left ctaValue) (ctaExpr, bool) {
	if !p.facade.comparesValues() {
		return nil, false
	}
	right, ok := p.additiveExpr()
	if !ok {
		return nil, false
	}
	comparison, typing := p.types.valueComparison(op, left, right)
	if typing == ctaTypeDeclined {
		return nil, false
	}
	if typing == ctaTypeErrored {
		return ctaTypeError{}, true
	}
	return ctaValueCompare{op: op, comparison: comparison, left: left, right: right}, true
}

// additiveExpr parses xpath20.md [13] AdditiveExpr, `MultiplicativeExpr ( ("+"
// | "-") MultiplicativeExpr )*`, left-associatively, in each position [11]
// ta-BooleanExpr and a value comparison read an operand in. A single operand
// yields that operand rather than an arithmetic node, so a {test} with no
// operator compiles to the tree it always did.
func (p *ctaParser) additiveExpr() (ctaValue, bool) {
	left, ok := p.multiplicativeExpr()
	if !ok {
		return nil, false
	}
	for {
		op, isOp := p.additiveOperator()
		if !isOp {
			return left, true
		}
		right, ok := p.multiplicativeExpr()
		if !ok {
			return nil, false
		}
		if left, ok = p.arithmetic(op, left, right); !ok {
			return nil, false
		}
	}
}

// additiveOperator reads one [13] AdditiveExpr operator, reporting false where
// the cursor is on anything else.
func (p *ctaParser) additiveOperator() (ctaArithOp, bool) {
	switch p.peek(0).kind {
	case ctaPlusTok:
		p.advance()
		return ctaAdd, true
	case ctaMinusTok:
		p.advance()
		return ctaSubtract, true
	default:
		return ctaAdd, false
	}
}

// multiplicativeExpr parses xpath20.md [14] MultiplicativeExpr, `UnionExpr (
// ("*" | "div" | "idiv" | "mod") UnionExpr )*`, on additiveExpr's terms. Each
// operand is a [14] ta-ValueExpr: the productions between UnionExpr and
// ValueExpr are reached only through their one-operand arms, so a union — but
// in an fn:count argument (countArgument) — a `treat`, an `instance of` and a
// unary sign leave a token no production takes and decline.
func (p *ctaParser) multiplicativeExpr() (ctaValue, bool) {
	left, ok := p.valueExpr()
	if !ok {
		return nil, false
	}
	for {
		op, isOp := p.multiplicativeOperator()
		if !isOp {
			return left, true
		}
		right, ok := p.valueExpr()
		if !ok {
			return nil, false
		}
		if left, ok = p.arithmetic(op, left, right); !ok {
			return nil, false
		}
	}
}

// multiplicativeOperator reads one [14] MultiplicativeExpr operator, reporting
// false where the cursor is on anything else. `*` is a bare ctaWildcardTok,
// which right after a complete operand no NameTest can be; `div`, `idiv` and
// `mod` are NCNames, operators by position on valueComparator's terms.
func (p *ctaParser) multiplicativeOperator() (ctaArithOp, bool) {
	tok := p.peek(0)
	var op ctaArithOp
	switch {
	case tok.kind == ctaWildcardTok && tok.text == "*":
		op = ctaMultiply
	case tok.kind == ctaNameTok && tok.text == "div":
		op = ctaDivide
	case tok.kind == ctaNameTok && tok.text == "idiv":
		op = ctaIntegerDivide
	case tok.kind == ctaNameTok && tok.text == "mod":
		op = ctaModulus
	default:
		return ctaAdd, false
	}
	p.advance()
	return op, true
}

// arithmetic builds the node of one binary arithmetic operator over two
// operands already parsed (xpath20.md §3.4), declining where the façade
// computes nothing (ctaFacade.computes) or ctaTypes.arithmetic will not type
// the pair.
func (p *ctaParser) arithmetic(op ctaArithOp, left, right ctaValue) (ctaValue, bool) {
	if !p.facade.computes() {
		return nil, false
	}
	return p.types.arithmetic(op, left, right)
}

// valueExpr parses [14] ta-ValueExpr, dispatching on whether a function call
// opens it, and on which function it calls: fn:count, which the assertion
// and facet façades call (countCall), one of the F&O functions a façade that calls the
// library admits (ctaFacade.callsLibrary, libraryCall), or a constructor.
//
// The gate is asked before the name is: where the façade calls no library
// function, every name but fn:count is a constructor call, so a Type
// Alternative's {test} reaches castTarget with it and declines, and its grammar
// stays at fn:not (§3.12.6 clause 3).
func (p *ctaParser) valueExpr() (ctaValue, bool) {
	if !p.at(ctaNameTok) || p.peek(1).kind != ctaLParen {
		return p.castExpr()
	}
	name := p.functionName(p.peek(0).text)
	if name == ctaCountFunction {
		return p.countCall()
	}
	if p.facade.callsLibrary() && name.Space == ctaFunctionNS {
		return p.libraryCall(name.Local)
	}
	return p.constructorFunction()
}

// libraryCall parses a call, xpath20.md [48] FunctionCall, to the function
// named local in the default function namespace, where it is one of the F&O
// functions the assertion and facet façades admit (ctafunc.go), and is a
// constructorFunction for every other local, where castTarget declines it.
// The name is matched here and nowhere else, and no node stores it.
func (p *ctaParser) libraryCall(local string) (ctaValue, bool) {
	switch local {
	case "contains":
		return p.matchCall(ctaContains)
	case "starts-with":
		return p.matchCall(ctaStartsWith)
	case "ends-with":
		return p.matchCall(ctaEndsWith)
	case "string-length":
		return p.unaryStringCall(ctaStringLength)
	case "normalize-space":
		return p.unaryStringCall(ctaNormalizeSpace)
	case "string":
		return p.stringCall()
	case "empty":
		return p.presenceCall(ctaEmptyTest)
	case "exists":
		return p.presenceCall(ctaExistsTest)
	case "distinct-values":
		return p.distinctValuesCall()
	case "true", "false":
		return p.constantCall(local)
	}
	return p.constructorFunction()
}

// arguments parses the parenthesized argument list of a call whose name the
// cursor is on, xpath20.md [48] FunctionCall's `"(" (ExprSingle ("," ExprSingle)*)?
// ")"`, each argument as argument parses it. The arity is the caller's to
// check, before it builds a node.
func (p *ctaParser) arguments() ([]ctaValue, bool) {
	p.advance() // the function name
	p.advance() // '('
	if p.at(ctaRParen) {
		p.advance()
		return nil, true
	}
	var args []ctaValue
	for {
		arg, ok := p.argument()
		if !ok {
			return nil, false
		}
		args = append(args, arg)
		if p.at(ctaRParen) {
			p.advance()
			return args, true
		}
		if !p.at(ctaCommaTok) {
			return nil, false
		}
		p.advance()
	}
}

// argument parses one argument of a library call: an additiveExpr operand, or
// the empty sequence `()` (xpath20.md [46] ParenthesizedExpr with no Expr),
// which is admitted in argument position only and compiles to the statically
// empty ctaEmptyValue. Every other parenthesized expression declines here, as
// it does in an arithmetic operand's position.
func (p *ctaParser) argument() (ctaValue, bool) {
	if p.at(ctaLParen) && p.peek(1).kind == ctaRParen {
		p.advance()
		p.advance()
		return ctaEmptyValue{}, true
	}
	return p.additiveExpr()
}

// matchCall parses a call to fn:contains, fn:starts-with or fn:ends-with (op)
// with its two xs:string? arguments (xpath-functions.md §7.5.1–7.5.3), whose
// result is xs:boolean. Every other arity declines (err:XPST0017, withheld on
// [CompileAssertionTest]'s terms), and so does a boolean that does not
// resolve.
//
// GAP(xpath): the three-argument form, whose third argument names a collation
// (§7.3.1), declines rather than being evaluated — and is never evaluated as
// the two-argument form, which compares under the default collation alone. The
// direction is the withhold [CompileAssertionTest] reports. (#1042)
func (p *ctaParser) matchCall(op ctaMatchOp) (ctaValue, bool) {
	args, ok := p.arguments()
	if !ok || len(args) != 2 {
		return nil, false
	}
	boolean, resolved := p.types.simple(ctaBuiltin("boolean"))
	if !resolved {
		return nil, false
	}
	return ctaMatch{op: op, left: p.types.stringArgument(args[0]), right: p.types.stringArgument(args[1]), st: boolean}, true
}

// unaryStringCall parses a call to fn:string-length or fn:normalize-space (op)
// with its one xs:string? argument, or with none, which F&O makes the string
// value of the context item: `f(fn:string(.))` (xpath-functions.md §7.4.4,
// §7.4.5), whose `.` is argumentOrDot's and whose fn:string is stringOf's. Two
// or more arguments decline, and so does a result type that does not resolve.
func (p *ctaParser) unaryStringCall(op ctaUnaryStringOp) (ctaValue, bool) {
	args, ok := p.arguments()
	if !ok || len(args) > 1 {
		return nil, false
	}
	arg, ok := p.argumentOrDot(args)
	if !ok {
		return nil, false
	}
	if len(args) == 0 {
		if arg, ok = p.stringOf(arg); !ok {
			return nil, false
		}
	}
	st, resolved := p.types.simple(op.resultType())
	if !resolved {
		return nil, false
	}
	return ctaUnaryString{op: op, arg: p.types.stringArgument(arg), st: st}, true
}

// argumentOrDot is the one argument args holds, or, where it holds none, the
// context item `.`, which is p.facade's (ctaFacade.contextItem): the assertion
// façade declines it, and the facet façade's raises err:XPDY0002.
func (p *ctaParser) argumentOrDot(args []ctaValue) (ctaValue, bool) {
	if len(args) == 1 {
		return args[0], true
	}
	return p.facade.contextItem()
}

// stringCall parses a call to fn:string with its one `item()?` argument, or
// with none, which is the context item `.` (xpath-functions.md §2.3). Two or
// more arguments decline.
func (p *ctaParser) stringCall() (ctaValue, bool) {
	args, ok := p.arguments()
	if !ok || len(args) > 1 {
		return nil, false
	}
	arg, ok := p.argumentOrDot(args)
	if !ok {
		return nil, false
	}
	return p.stringOf(arg)
}

// stringOf builds fn:string over arg: "the same string as is returned by the
// expression "$arg cast as xs:string"" for an atomic value (xpath-functions.md
// §2.3), which is a ctaCast to xs:string admitted on castsFrom's terms and
// allowing the empty sequence, which ctaStringFunction maps to the zero-length
// string. A node's string-value is the same string wherever castsFrom admits
// the node: the string-value of an attribute or of an element of simple type is
// its [schema normalized value] (xpath-datamodel :1562, :1307), which for the
// xs:string family is its typed value. For an xs:date, xs:dateTime or xs:time
// node this engine takes it to be the ·canonical representation· of the typed
// value, which castsFrom's third rule renders: an implementation storing
// only the typed value "may use any valid lexical representation of the typed
// value for the string-value property" (xpath-datamodel :1567, :1309). Every
// other typed node, and every xs:float or xs:double argument, a literal
// included, declines under castsFrom's GAP(xpath).
func (p *ctaParser) stringOf(arg ctaValue) (ctaValue, bool) {
	if !p.types.castsFrom(arg, p.types.str) {
		return nil, false
	}
	return ctaStringFunction{cast: ctaCast{operand: arg, target: p.types.str, allowsEmpty: true}}, true
}

// presenceCall parses a call to fn:empty or fn:exists (op) with its one
// `item()*` argument (xpath-functions.md §15.1.4, §15.1.5), which is not
// atomized (presenceArgument), and whose result is xs:boolean. A boolean that
// does not resolve declines.
func (p *ctaParser) presenceCall(op ctaPresenceOp) (ctaValue, bool) {
	operand, ok := p.presenceArgument()
	if !ok {
		return nil, false
	}
	boolean, resolved := p.types.simple(ctaBuiltin("boolean"))
	if !resolved {
		return nil, false
	}
	return ctaPresence{op: op, operand: operand, st: boolean}, true
}

// presenceArgument parses the parenthesized argument list of the fn:empty or
// fn:exists call whose name the cursor is on: a child path (childPath) or one
// element step (selectedElements) where it is the whole list, closed by the
// call's ')', and otherwise the list arguments parses, of which exactly one
// argument is admitted — every other arity declines (err:XPST0017).
func (p *ctaParser) presenceArgument() (ctaValue, bool) {
	if n := p.childPathLength(2); n > 0 && p.peek(2+n).kind == ctaRParen {
		p.advance() // the function name
		p.advance() // '('
		path, ok := p.childPath(n)
		p.advance() // ')'
		return path, ok
	}
	if n := p.selectedStepLength(2); n > 0 && p.peek(2+n).kind == ctaRParen {
		p.advance() // the function name
		p.advance() // '('
		step, ok := p.selectedElements()
		p.advance() // ')'
		return step, ok
	}
	args, ok := p.arguments()
	if !ok || len(args) != 1 {
		return nil, false
	}
	return args[0], true
}

// distinctValuesCall parses a call to fn:distinct-values with its one
// `xs:anyAtomicType*` argument (xpath-functions.md §15.1.6), which is atomized
// (ctaDistinctValues), and whose result is that argument's distinct items. A
// statically empty argument compiles to ctaEmptyValue, which is the empty
// sequence the call returns over it. The node's st is the type its items are
// compared in: the argument's own static type, or xs:string for an
// xs:untypedAtomic one, which declines where p.types resolves no xs:string.
// No argument declines (err:XPST0017).
//
// GAP(xpath): the two-argument form, whose second argument names a collation
// (§7.3.1), declines rather than being evaluated — and is never evaluated as
// the one-argument form, which compares under the default collation alone. The
// direction is the withhold [CompileAssertionTest] reports. (#1042)
func (p *ctaParser) distinctValuesCall() (ctaValue, bool) {
	args, ok := p.arguments()
	if !ok || len(args) != 1 {
		return nil, false
	}
	if ctaIsEmpty(args[0]) {
		return ctaEmptyValue{}, true
	}
	if typed, isTyped := ctaStaticOf(args[0]).(ctaTyped); isTyped {
		return ctaDistinctValues{operand: args[0], st: typed.st}, true
	}
	str, resolved := p.types.simple(ctaBuiltin("string"))
	if !resolved {
		return nil, false
	}
	return ctaDistinctValues{operand: args[0], st: str}, true
}

// constantCall parses a call to fn:true or fn:false, named local, with no
// argument (xpath-functions.md §9.1.1, §9.1.2), into the ctaLiteral of the
// xs:boolean it returns, whose lexical local is. An argument declines, and so
// does a boolean that does not resolve.
func (p *ctaParser) constantCall(local string) (ctaValue, bool) {
	args, ok := p.arguments()
	if !ok || len(args) != 0 {
		return nil, false
	}
	boolean, resolved := p.types.simple(ctaBuiltin("boolean"))
	if !resolved {
		return nil, false
	}
	return ctaLiteral{text: local, st: boolean}, true
}

// countCall parses an fn:count call, xpath20.md [48] FunctionCall with one
// argument, whose name the caller has already resolved to fn:count. Where the
// façade calls the library (ctaFacade.callsLibrary) and the argument opens as
// no path can (countsItems), the argument is one a library call takes
// (argument) — `$value`, `()`, a function call, a literal — whose items are
// counted, and the node is their ctaCount (ctaCountedItems, ctaCountOf) on
// every such façade. Every other argument is a path countArgument parses,
// whose node is p.facade's (ctaFacade.count), which may decline it, as is the
// node of a rooted argument; an argument outside those shapes declines.
func (p *ctaParser) countCall() (ctaValue, bool) {
	p.advance() // the function name
	p.advance() // '('
	if p.facade.callsLibrary() && p.countsItems() {
		operand, ok := p.argument()
		if !ok || !p.at(ctaRParen) {
			return nil, false
		}
		p.advance()
		return ctaCountOf(ctaCountedItems{operand: operand}, p.types)
	}
	arg, ok := p.countArgument()
	if !ok || !p.at(ctaRParen) {
		return nil, false
	}
	p.advance()
	return p.facade.count(arg, p.types)
}

// countsItems reports whether the cursor opens an fn:count argument no path
// opens with: a VarRef's '$', a '(' — `()` among them — a literal, or a
// function call, a name followed by '('. A path opens with "/", "//", ".",
// '@', a wildcard, an axis or a name standing alone. Nothing is consumed.
func (p *ctaParser) countsItems() bool {
	if p.at(ctaNameTok) {
		return p.peek(1).kind == ctaLParen
	}
	return p.at(ctaDollarTok) || p.at(ctaLParen) || p.at(ctaStringTok) || p.at(ctaNumberTok)
}

// countArgument parses fn:count's argument: one operand (countOperand), or
// xpath20.md [21] UnionExpr over two or more, joined by `|` or `union`
// (§3.3.3), whose node ctaUnionOf builds and which declines a rooted operand
// and one filtered by a predicate that reads the child's value.
func (p *ctaParser) countArgument() (ctaCounted, bool) {
	first, ok := p.countOperand()
	if !ok {
		return nil, false
	}
	if !p.atUnion() {
		return first, true
	}
	operands := []ctaCounted{first}
	for p.atUnion() {
		p.advance()
		next, ok := p.countOperand()
		if !ok {
			return nil, false
		}
		operands = append(operands, next)
	}
	return ctaUnionOf(operands)
}

// atUnion reports whether the cursor sits on a [21] UnionExpr operator, `|` or
// `union`, which right after a complete operand no other production opens
// with.
func (p *ctaParser) atUnion() bool { return p.at(ctaBarTok) || p.atName("union") }

// countOperand parses one operand of fn:count's argument: a path countPath
// parses, and, where a '[' follows a child step `N` or `./N`, the one
// predicate filtering it (predicate). A predicate after any other step — `.//N`,
// `@N`, a rooted step — and a second predicate leave a token no production
// takes, and decline.
func (p *ctaParser) countOperand() (ctaCounted, bool) {
	arg, ok := p.countPath()
	if !ok || !p.at(ctaLBracketTok) {
		return arg, ok
	}
	step, isPath := arg.(ctaCountPath)
	if !isPath || step.axis != ctaCountChildren {
		return nil, false
	}
	return p.predicate(step.name)
}

// predicate parses xpath20.md [40] Predicate, `"[" Expr "]"`, filtering the
// child step that selects E's children named name, as far as an fn:count
// argument admits it: a conjunction of attribute-existence tests with QName
// NameTests, `@A` or `@A1 and @A2 …`, resolved on attributeName's terms, which
// is ctaFilteredChildren, and otherwise a predicate over the child's value
// (valuePredicate).
//
// GAP(xpath): every other predicate declines — an attribute test in a
// disjunction, under fn:not, with a wildcard or atomized (`c[@a = 1]`), a
// numeric one (`c[1]`), and any predicate outside an fn:count argument. The
// direction is the withhold [CompileAssertionTest] reports. (#1042)
func (p *ctaParser) predicate(name xsd.QName) (ctaCounted, bool) {
	p.advance() // '['
	n := p.existenceLength()
	if n == 0 || p.peek(n).kind != ctaRBracketTok {
		return p.valuePredicate(name)
	}
	var required []xsd.QName
	for end := p.pos + n; p.pos < end; p.advance() {
		if p.at(ctaAtTok) {
			p.advance()
			required = append(required, p.attributeName(p.peek(0).text))
		}
	}
	p.advance() // ']'
	return ctaFilteredChildrenOf(name, required)
}

// valuePredicate parses the predicate whose '[' the cursor has passed, over
// the children named name, as a predicate over each child's value
// (ctaMatchingChildren): an [8] ta-Test production read with `.` the child
// (ctaCandidate), typed as p.facade types a child step naming name, so a child
// that step would decline — the Type Alternative's and the facet's every one,
// and an assertion's of no single simple typed value — declines here too, and
// so does one that step reads untyped, a child of mixed content
// (ctaUntypedChild): a candidate is typed, and an untyped one would need an arm
// of its own. Every other production reads ctaPredicateFacade, which declines
// each read of a node — an attribute, a child or element step, a path, `$value`
// — an fn:count call and every F&O function. The root must be a comparison, or
// and, or and fn:not over comparisons (ctaComparisonRooted): a bare value may
// be numeric, which selects by position.
func (p *ctaParser) valuePredicate(name xsd.QName) (ctaCounted, bool) {
	step, typed := p.facade.child(ctaExactName{name: name}, p.types)
	child, isChild := step.(ctaTypedChild)
	if !typed || !isChild {
		return nil, false
	}
	outer := p.facade
	p.facade = ctaPredicateFacade{candidate: ctaCandidate{st: child.st}}
	pred, parsed := p.orExpr()
	p.facade = outer
	if !parsed || !p.at(ctaRBracketTok) || !ctaComparisonRooted(pred) {
		return nil, false
	}
	p.advance() // ']'
	return ctaMatchingChildren{name: name, pred: pred}, true
}

// ctaPredicateFacade is the façade a value predicate's own expression parses
// under (ctaParser.valuePredicate): its context item is candidate, the child
// the predicate filters, so every production that reads another node
// declines — an attribute of the candidate would need its own type, a step
// below it a subtree this engine does not keep — and so does `$value`, an
// fn:count call, and every F&O function. Comparisons and arithmetic are
// admitted over what remains, the candidate, literals and casts.
type ctaPredicateFacade struct{ candidate ctaCandidate }

func (ctaPredicateFacade) ctaFacade() {}

func (ctaPredicateFacade) attribute(ctaNameTest, ctaTypes) (ctaValue, bool) { return nil, false }

// comparesValues is true, on ctaAssertionFacade.comparesValues' terms.
func (ctaPredicateFacade) comparesValues() bool { return true }

func (ctaPredicateFacade) variable(xsd.QName, ctaTypes) (ctaValue, bool) { return nil, false }

func (ctaPredicateFacade) child(ctaNameTest, ctaTypes) (ctaValue, bool) { return nil, false }

func (ctaPredicateFacade) childPath([]xsd.QName) (ctaValue, bool) { return nil, false }

func (ctaPredicateFacade) elements(ctaCountPath) (ctaValue, bool) { return nil, false }

func (ctaPredicateFacade) rooted() (ctaValue, bool) { return nil, false }

// contextItem compiles `.` to the candidate child (ctaCandidate).
func (f ctaPredicateFacade) contextItem() (ctaValue, bool) { return f.candidate, true }

func (ctaPredicateFacade) count(ctaCounted, ctaTypes) (ctaValue, bool) { return nil, false }

// computes is true, on ctaAssertionFacade.computes' terms.
func (ctaPredicateFacade) computes() bool { return true }

// callsLibrary is false: a library call in a predicate declines at
// constructorFunction, so `string-length(.)` and position() and last(), which
// the library holds none of, never reach a node.
func (ctaPredicateFacade) callsLibrary() bool { return false }

// existenceLength is how many tokens at the cursor spell a conjunction of
// attribute-existence tests, `'@' QName ('and' '@' QName)*`, and 0 where they
// spell none. Nothing is consumed.
func (p *ctaParser) existenceLength() int {
	n := 0
	for {
		if p.peek(n).kind != ctaAtTok || p.peek(n+1).kind != ctaNameTok {
			return 0
		}
		n += 2
		if tok := p.peek(n); tok.kind != ctaNameTok || tok.text != "and" {
			return n
		}
		n++
	}
}

// countPath parses one counted path: a rooted path (rootedPath), whose node
// p.facade builds and which is an argument only where it is ctaCounted —
// ctaNoDocumentRoot, which raises before it selects a node — or a relative
// path of one step with a QName NameTest, `N`, `@N`, or either behind `./` or
// `.//` (countStep). A longer path, a wildcard and a bare `.` leave a token no
// production takes, or none at all, and decline. selectedElements parses its
// element steps too.
func (p *ctaParser) countPath() (ctaCounted, bool) {
	if p.at(ctaSlashTok) || p.at(ctaSlashSlashTok) {
		rooted, ok := p.rootedPath()
		if !ok {
			return nil, false
		}
		counted, isCounted := rooted.(ctaCounted)
		return counted, isCounted
	}
	if !p.at(ctaDotTok) {
		return p.countStep(ctaCountChildren, ctaCountOwnAttributes)
	}
	p.advance()
	if p.at(ctaSlashSlashTok) {
		p.advance()
		return p.countStep(ctaCountDescendants, ctaCountSubtreeAttributes)
	}
	if !p.at(ctaSlashTok) {
		return nil, false
	}
	p.advance()
	return p.countStep(ctaCountChildren, ctaCountOwnAttributes)
}

// countStep parses the one step of a counted relative path: `'@' QName`,
// resolved on attributeName's terms and counted on the axis attributes, or a
// QName, resolved on elementName's terms and counted on the axis elements. The
// step is never typed: fn:count does not atomize it (ctaCount).
func (p *ctaParser) countStep(elements, attributes ctaCountAxis) (ctaCounted, bool) {
	if p.at(ctaAtTok) {
		p.advance()
		if !p.at(ctaNameTok) {
			return nil, false
		}
		name := p.attributeName(p.peek(0).text)
		p.advance()
		return ctaCountPath{axis: attributes, name: name}, true
	}
	if !p.at(ctaNameTok) {
		return nil, false
	}
	name := p.elementName(p.peek(0).text)
	p.advance()
	return ctaCountPath{axis: elements, name: name}, true
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
	if !admitted || !p.types.castsFrom(v, target) {
		return nil, false
	}
	return ctaCast{operand: v, target: target, allowsEmpty: allowsEmpty}, true
}

// constructorFunction parses [18] ta-ConstructorFunction.
//
// §3.12.6 clause 3 makes every QName '(' SimpleValue ')' whose name is not
// fn:not a constructor call for a built-in datatype, so this one production
// covers both the constructor spelling of a cast and the "unknown boolean
// function" reading — there is no third case to distinguish. The calls beyond
// §3.12.6 are fn:count and, on a façade that calls the library, the F&O
// functions libraryCall names, which valueExpr hands to countCall and
// libraryCall instead. The name is a FUNCTION name, resolved as one, and it
// names the datatype at the same time: an unprefixed int(...) is fn:int,
// which declares no constructor, and never xs:int.
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
	if !admitted || !p.types.castsFrom(arg, target) {
		return nil, false
	}
	return ctaCast{operand: arg, target: target, allowsEmpty: true}, true
}

// simpleValue parses [16] ta-SimpleValue's two arms, the arms the assertion
// façade adds — [44] VarRef (varRef), a child-axis step (childStep), and a
// rooted path (rootedPath) — and [47] ContextItemExpr, which the facet and
// predicate façades add. A name opens the unabbreviated attribute axis only
// where `::` follows the name `attribute`, and a child-axis step otherwise.
func (p *ctaParser) simpleValue() (ctaValue, bool) {
	switch p.peek(0).kind {
	case ctaAtTok:
		return p.attrName()
	case ctaNameTok:
		if p.atName("attribute") && p.peek(1).kind == ctaAxisTok {
			return p.attrName()
		}
		return p.childStep()
	case ctaSlashTok, ctaSlashSlashTok:
		return p.rootedPath()
	case ctaDollarTok:
		return p.varRef()
	case ctaDotTok:
		p.advance()
		return p.facade.contextItem()
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

// varRef parses xpath20.md [44] VarRef, `'$' VarName`, whose [45] VarName is a
// QName. An unprefixed one is in NO namespace, which is where cvc-assertion
// clause 2.3 puts `$value` ("no namespace URI and ... "value" as the local
// name"); a prefixed one resolves against the {namespace bindings} on
// attributeName's terms. The node is p.facade's, which declines every name but
// the one variable its static context holds.
func (p *ctaParser) varRef() (ctaValue, bool) {
	p.advance() // '$'
	if !p.at(ctaNameTok) {
		return nil, false
	}
	text := p.peek(0).text
	p.advance()
	return p.facade.variable(p.attributeName(text), p.types)
}

// childStep parses one abbreviated child-axis step, xpath20.md [31]
// AbbrevForwardStep without its '@' — "If the axis name is omitted from an
// axis step, the default axis is child" (§3.2.4) — whose NodeTest is a QName
// NameTest, resolved on elementName's terms. The node is p.facade's, which
// may decline it. A [37] Wildcard NameTest is a ctaWildcardTok and never
// reaches here, and `child::` spelled out is a name followed by a `::` no
// production takes: both decline.
func (p *ctaParser) childStep() (ctaValue, bool) {
	text := p.peek(0).text
	p.advance()
	return p.facade.child(ctaExactName{name: p.elementName(text)}, p.types)
}

// childPathLength is how many tokens, from offset at ahead of the cursor,
// spell a relative path of two or more abbreviated child-axis steps with QName
// NameTests, `QName ('/' QName)+`, and 0 where they spell none. A name
// followed by '(' or '::' is a function call or an axis spelled out, never
// such a step, and so is a '/' followed by anything but a name: each answers
// 0, as does one step alone, which is childStep's. Nothing is consumed.
func (p *ctaParser) childPathLength(at int) int {
	n := 0
	for {
		if p.peek(at+n).kind != ctaNameTok {
			return 0
		}
		next := p.peek(at + n + 1).kind
		if next == ctaLParen || next == ctaAxisTok {
			return 0
		}
		n++
		if next != ctaSlashTok {
			break
		}
		n++
	}
	if n < 3 {
		return 0
	}
	return n
}

// closesBoolean reports whether the token at offset at ahead of the cursor ends
// a [11] ta-BooleanExpr: the end of the expression, the ')' of a parenthesized
// OrExpr or of fn:not, or the 'and' or 'or' of the expression enclosing it.
func (p *ctaParser) closesBoolean(at int) bool {
	tok := p.peek(at)
	if tok.kind == ctaEOF || tok.kind == ctaRParen {
		return true
	}
	return tok.kind == ctaNameTok && (tok.text == "and" || tok.text == "or")
}

// childPath parses the n tokens childPathLength measured at the cursor as
// xpath20.md [26] RelativePathExpr, `StepExpr ("/" StepExpr)+`, each step an
// abbreviated child-axis step (§3.2.1.1, §3.2.4) whose QName NameTest is
// resolved on elementName's terms, so an unprefixed one takes the {default
// namespace}. It is the ONE place a ctaChildPath is built, and it is reached
// from two positions alone: the ·effective boolean value· arm of booleanExpr
// and fn:exists or fn:empty's argument (presenceArgument), each where the path
// is the whole operand — the three positions that ask only whether the path
// selects a node, and never atomize it. simpleValue and additiveExpr never
// reach it, so no comparison, operator, cast or other call takes one, and
// childStep stays one step; one step in the same positions is
// selectedElements'. The node is p.facade's, which may decline it.
//
// GAP(xpath): a path in any other position, or with a wildcard, a predicate,
// an axis spelled out, a '//' between steps or an attribute step, declines;
// the direction is the withhold [CompileAssertionTest] reports. (#1042)
func (p *ctaParser) childPath(n int) (ctaValue, bool) {
	var steps []xsd.QName
	for end := p.pos + n; p.pos < end; p.advance() {
		if p.at(ctaSlashTok) {
			continue
		}
		steps = append(steps, p.elementName(p.peek(0).text))
	}
	return p.facade.childPath(steps)
}

// selectedStepLength is how many tokens, from offset at ahead of the cursor,
// spell one of countPath's element steps — a QName, `N`, alone or behind
// `./` or `.//` — and 0 where they spell none. Nothing is consumed, and nothing
// after the name is read: a name followed by '(' or '::' is a function call or
// an axis spelled out, which the caller's check that the token after the step
// closes the operand turns away.
func (p *ctaParser) selectedStepLength(at int) int {
	n := 0
	if p.peek(at).kind == ctaDotTok {
		if next := p.peek(at + 1).kind; next != ctaSlashTok && next != ctaSlashSlashTok {
			return 0
		}
		n = 2
	}
	if p.peek(at+n).kind != ctaNameTok {
		return 0
	}
	return n + 1
}

// selectedElements parses the step selectedStepLength measured at the cursor
// on countPath's grammar — `N` and `./N` the element children of E named
// N, `.//N` every element below E so named (xpath20.md §3.2.4) — into the node
// p.facade builds for it (ctaFacade.elements), which may decline it. It is the
// ONE place a ctaSelectedElements is built, reached from childPath's two
// positions alone and on its terms: the ·effective boolean value· arm of
// booleanExpr and fn:exists or fn:empty's argument (presenceArgument), where
// the step is the whole operand, whatever type the step's name has. Everywhere
// else a step is childStep's, whose value is read, and `./N` and `.//N`
// decline.
func (p *ctaParser) selectedElements() (ctaValue, bool) {
	arg, ok := p.countPath()
	if !ok {
		return nil, false
	}
	path, relative := arg.(ctaCountPath)
	if !relative {
		return nil, false
	}
	return p.facade.elements(path)
}

// rootedPath parses xpath20.md [25] PathExpr's two rooted arms, "/"
// RelativePathExpr and "//" RelativePathExpr, as far as ONE step with a QName
// NameTest: a child-axis one, or an abbreviated attribute one, `'@' QName`; a
// longer path, a wildcard, and a bare "/" leave a token no production takes.
// The step is resolved — an unbound prefix in it is err:XPST0081 like any other,
// on elementName's or attributeName's terms — and never typed: the node is
// p.facade's, and each façade that admits it builds a node that raises before
// any step is taken (ctaNoDocumentRoot, ctaNoContextItem), so what the step
// would select is never asked.
func (p *ctaParser) rootedPath() (ctaValue, bool) {
	p.advance() // '/' or '//'
	attribute := p.at(ctaAtTok)
	if attribute {
		p.advance()
	}
	if !p.at(ctaNameTok) {
		return nil, false
	}
	p.rootedStepName(attribute, p.peek(0).text)
	p.advance()
	return p.facade.rooted()
}

// rootedStepName resolves the NameTest text of rootedPath's one step, on
// attributeName's terms where attribute is true and elementName's otherwise.
// The name is discarded: resolving it is what records an unbound prefix.
func (p *ctaParser) rootedStepName(attribute bool, text string) {
	if attribute {
		p.attributeName(text)
		return
	}
	p.elementName(text)
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
//
// The node the resolved NameTest becomes is p.facade's, which may decline it.
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
		return p.facade.attribute(ctaExactName{name: p.attributeName(text)}, p.types)
	case ctaWildcardTok:
		p.advance()
		return p.facade.attribute(p.wildcardTest(text), p.types)
	default:
		return nil, false
	}
}
