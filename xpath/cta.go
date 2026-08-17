package xpath

import (
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file evaluates the RESTRICTED expression subset a Type Alternative's
// {test} is written in — the "Test XPath expressions" grammar ta-props-correct
// clause 2.1 (§3.12.6) fixes, productions [8] ta-Test through [18]
// ta-ConstructorFunction — and nothing wider. It is not a stage of a general
// XPath 2.0 evaluator: the productions below reach no axis but attribute, no
// predicate, no variable and no function but fn:not, so evaluating them
// directly is exact where a fail-open delegation to a general engine would be
// a guess.
//
//	[8]  Test                ::= OrExpr
//	[9]  OrExpr              ::= AndExpr ( 'or' AndExpr )*
//	[10] AndExpr             ::= BooleanExpr ( 'and' BooleanExpr )*
//	[11] BooleanExpr         ::= '(' OrExpr ')' | BooleanFunction |
//	                             ValueExpr ( Comparator ValueExpr )?
//	[12] BooleanFunction     ::= QName '(' OrExpr ')'
//	[13] Comparator          ::= '=' | '!=' | '<' | '<=' | '>' | '>='
//	[14] ValueExpr           ::= CastExpr | ConstructorFunction
//	[15] CastExpr            ::= SimpleValue ( 'cast' 'as' QName '?'? )?
//	[16] SimpleValue         ::= AttrName | Literal
//	[17] AttrName            ::= '@' NameTest
//	[18] ConstructorFunction ::= QName '(' SimpleValue ')'
//
// TWO ERROR DIRECTIONS MEET HERE AND NEITHER IS THE OTHER.
//
// A test the engine cannot evaluate is declined at COMPILE time
// ([CompileCTATest] reporting false), and its consequence — the caller
// withholding the element's ·governing type definition· — is argued at that
// caller (validate/cta.go). A dynamic or type error inside a test the engine
// CAN evaluate is a decided false, because key-cta-ta-select clause 2
// (§3.12.4) says so: "If a dynamic error or a type error is raised during
// evaluation, then the {test} is treated as if it had evaluated (without
// error) to false." That is spec-normative behavior and not this module's
// fail-open contract, so it carries no GAP marker and never reaches the
// caller as a decline.
//
// The nodes of the XDM instance a {test} runs against are UNTYPED
// (key-cta-ta-select clause 1's Note), so every attribute operand atomizes to
// xs:untypedAtomic and xpath20.md §3.5.2's untypedAtomic casting rules are the
// normal path of every comparison here rather than an edge case.

// AttributeValue reads one attribute of the element information item E whose
// ·selected type definition· is being decided: the [[normalized value]] of
// E.[[attributes]] member whose ·expanded name· is name, and whether E carries
// such an attribute at all.
//
// An ABSENT attribute — present false — atomizes to the empty sequence, and
// xpath20.md §3.5.2 makes a general comparison true only when "there is a pair
// of atomic values, one in the first operand sequence and the other in the
// second". No pair can be formed against an empty sequence, so every
// comparison naming an attribute E does not carry is false. That is neither an
// error nor a decline, and an implementation must not report it as one.
//
// The lexical is the attribute's [[normalized value]]: XML 1.0 §3.3.3
// attribute-value normalization applied, whiteSpace-facet normalization NOT —
// the latter belongs to the type a cast targets, and [CTATest.Evaluate]
// applies it per target type.
type AttributeValue func(name xsd.QName) (lexical string, present bool)

// CTATest is a compiled Type Alternative {test}: the §3.12.6 required-subset
// expression tree [CompileCTATest] admitted, ready to be evaluated against any
// number of element information items.
type CTATest struct{ root ctaExpr }

// CompileCTATest compiles the {test} of a Type Alternative (§3.12.1, an
// [xsd.XPathExpression] property record) into an evaluable [CTATest],
// reporting ok false for an {expression} this engine cannot evaluate.
//
// ok false is the WITHHOLD direction and covers every such {expression} under
// one encoding, because they all have one consequence at the caller:
//
//   - text that is not a [8] ta-Test production at all. A legal full XPath 2.0
//     {test} looks exactly like this to a restricted-subset parser, and
//     ta-props-correct and xpath-valid (§3.13.6.2) are SCHEMA component
//     constraints charged at assembly by their owner, neither of which this
//     module implements — so nothing upstream has narrowed what arrives here
//     and an error return would be a false reject of every schema using full
//     XPath in conditional type assignment;
//   - a prefix with no binding in the record's {namespace bindings}
//     (err:XPST0081), which is a STATIC error and so not clause 2's false;
//   - a construct inside the required subset that this engine does not yet
//     evaluate, each carrying its own marker at the production in
//     ctaparser.go that recognizes it.
//
// It never returns an error: a decline is not a verdict about the schema.
//
// The static context is xpath-valid clause 2.2's: XPath 1.0 compatibility mode
// false (2.2.1), statically known namespaces from expr.NamespaceBindings()
// (2.2.2) and the default function namespace http://www.w3.org/2005/xpath-functions
// (2.2.4). expr.DefaultNamespace() — the default element/type namespace
// (2.2.3) — is deliberately NOT consulted: [17] ta-AttrName makes every
// NameTest in this grammar an attribute-axis one, whose principal node kind is
// never element, so an unprefixed NameTest here is always in no namespace
// (xpath20.md §3.2.1.2, PRINCIPLES 15).
func CompileCTATest(expr xsd.XPathExpression) (CTATest, bool) {
	toks, ok := ctaTokenize(expr.Expression())
	if !ok {
		return CTATest{}, false
	}
	names := ctaNames{prefixes: make(map[string]string)}
	for _, b := range expr.NamespaceBindings() {
		names.prefixes[b.Prefix()] = b.Namespace()
	}
	p := ctaParser{toks: toks, names: names}
	root, ok := p.test()
	if !ok {
		return CTATest{}, false
	}
	return CTATest{root: root}, true
}

// Evaluate reports whether the compiled {test} holds for the element
// information item whose attributes attr reads, which is the whole of what
// key-cta-ta-select (§3.12.4) asks of a {test}: "A.{test} evaluates to true".
//
// It always decides. Every way an evaluation can go wrong inside the compiled
// subset — a cast that fails (err:FORG0001), operand types the operator does
// not accept (err:XPTY0004) — is a dynamic or type error, which clause 2 makes
// a false rather than an error or a decline. The declines all happened at
// [CompileCTATest].
//
// Clause 2's subject is "the {test}", not the sub-expression that raised, so
// this is the ONE place the substitution happens: a raised error travels up
// the expression tree under XPath's own rules for each operator and becomes
// false here, at the root. An error under an fn:not is therefore a false
// {test} and never the inversion of one.
//
// b supplies the value spaces the numeric comparisons run in and must be
// non-nil; it is asked only for the primitives xs:decimal and xs:double, which
// every [value.Backend] covers by contract. Comparisons of xs:string operands
// do not reach it: xpath20.md B.2 defines all six operators on xs:string
// through fn:compare, hence through the default collation, which xpath-valid
// clause 2.2.10 fixes as the Unicode codepoint collation — a static-context
// property no backend owns.
func (t CTATest) Evaluate(b value.Backend, attr AttributeValue) bool {
	return ctaEval(t.root, ctaEnv{backend: b, attr: attr}) == ctaTrue
}

// ctaEnv is the dynamic context of one [CTATest.Evaluate] call. cvc-xpath
// (§3.13.4.2) fixes the rest of it — context item E, context position and size
// 1, no variable values — and none of that is representable in this grammar,
// which reaches no context item and has no VarRef production, so the
// attributes and the value spaces are the whole of what evaluation reads.
type ctaEnv struct {
	backend value.Backend
	attr    AttributeValue
}

// ctaExpr is the sealed sum of the BOOLEAN-valued nodes of the compiled tree.
// The grammar closes the set (STYLE T2's schema-closed-set exception), so
// consumers type-switch over the branches and no further branch is
// representable outside this package.
type ctaExpr interface{ ctaExpr() }

// ctaOr is [9] ta-OrExpr: existential over its operands, in written order.
// A one-operand OrExpr is never built — the parser returns the operand itself
// — so an ctaOr always holds two or more.
type ctaOr struct{ operands []ctaExpr }

// ctaAnd is [10] ta-AndExpr, universal over its operands on ctaOr's terms.
type ctaAnd struct{ operands []ctaExpr }

// ctaNot is the fn:not call [12] ta-BooleanFunction is fixed to by §3.12.6
// clause 3 ("Any strings matching the BooleanFunction production are function
// calls to fn:not").
type ctaNot struct{ operand ctaExpr }

// ctaCompare is [11] ta-BooleanExpr's third arm with its Comparator present:
// a general comparison (xpath20.md §3.5.2) whose operand value space was
// settled at compile time by ctaComparisonSpace.
type ctaCompare struct {
	op    ctaComparator
	space ctaSpace
	left  ctaValue
	right ctaValue
}

// ctaEffectiveBoolean is [11] ta-BooleanExpr's third arm with its Comparator
// ABSENT — a bare ValueExpr standing in a boolean position, whose value is its
// ·effective boolean value· (xpath20.md §2.4.3, fn:boolean).
type ctaEffectiveBoolean struct{ operand ctaValue }

// ctaTypeError is a comparison whose operand types are not a valid combination
// for its operator under xpath20.md B.2 — a string operand against a numeric
// one, the only such pair this grammar can build once the untypedAtomic
// casting rules have run. XPath raises err:XPTY0004 for it, so the node
// evaluates to ctaError: a raised error the enclosing operators carry, not a
// decline and not a node-local false.
type ctaTypeError struct{}

func (ctaOr) ctaExpr()               {}
func (ctaAnd) ctaExpr()              {}
func (ctaNot) ctaExpr()              {}
func (ctaCompare) ctaExpr()          {}
func (ctaEffectiveBoolean) ctaExpr() {}
func (ctaTypeError) ctaExpr()        {}

// ctaValue is the sealed sum of the ITEM-valued nodes: the two arms of [16]
// ta-SimpleValue that survive compilation. [15] ta-CastExpr's cast tail and
// [18] ta-ConstructorFunction have no arm here because [CompileCTATest]
// declines them (see the GAP markers in ctaparser.go).
type ctaValue interface{ ctaValue() }

// ctaAttr is [17] ta-AttrName, with its NameTest ALREADY resolved to an
// ·expanded name·: a prefixed one against the {namespace bindings}, an
// unprefixed one to no namespace, because the attribute axis's principal node
// kind is never element (xpath20.md §3.2.1.2). That asymmetry is settled at
// compile time so evaluation is a comparison of expanded names carrying no
// axis of its own.
type ctaAttr struct{ name xsd.QName }

// ctaLiteral is the Literal arm of [16] ta-SimpleValue, with the value space
// its XPath literal kind fixes: a StringLiteral is xs:string, an
// IntegerLiteral or DecimalLiteral is xs:decimal (xs:integer's primitive base,
// which is the space two such literals are compared in), and a DoubleLiteral
// is xs:double.
type ctaLiteral struct {
	text  string
	space ctaSpace
}

func (ctaAttr) ctaValue()    {}
func (ctaLiteral) ctaValue() {}

// ctaSpace is the value space a comparison's operands are compared in, which
// xpath20.md §3.5.2 derives from the operand types. Only three arise: this
// grammar's operands are xs:untypedAtomic attributes and string, decimal and
// double literals, and §3.5.2's casting rules map every admissible pair of
// those onto one of the three.
type ctaSpace byte

const (
	// ctaSpaceString is xs:string, and comparison in it is the default
	// (Unicode codepoint) collation rather than a backend value space.
	ctaSpaceString ctaSpace = iota
	// ctaSpaceDecimal is xs:decimal, the space two non-double numeric
	// literals are compared in.
	ctaSpaceDecimal
	// ctaSpaceDouble is xs:double, which §3.5.2 clause 2.1 forces whenever an
	// xs:untypedAtomic operand meets a numeric one.
	ctaSpaceDouble
)

// ctaSpaceName is the ·expanded name· of the builtin datatype a ctaSpace is,
// which is the key a [value.Backend] maps.
func ctaSpaceName(s ctaSpace) xsd.QName {
	switch s {
	case ctaSpaceDecimal:
		return xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"}
	case ctaSpaceDouble:
		return xsd.QName{Space: xsd.XMLSchemaNS, Local: "double"}
	default:
		return xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}
	}
}

// ctaComparisonSpace settles which value space a comparison of l against r
// runs in, per xpath20.md §3.5.2's magnitude-relationship rules read against
// this grammar's operands (an attribute is xs:untypedAtomic, a literal is its
// own type). ok is false where B.2 admits no combination for the pair, which
// is err:XPTY0004 and so ctaTypeError.
//
//   - Two attributes are both xs:untypedAtomic, which rule 1 casts to
//     xs:string.
//   - An attribute against a string literal is rule 2.4's "primitive base type
//     of T", xs:string.
//   - An attribute against a numeric literal is rule 2.1's xs:double, whatever
//     the literal's own numeric type — after which rule 3's value comparison
//     promotes the literal to xs:double too (xpath20.md §B.1).
//   - Two literals are compared in their least common type: the shared one, or
//     xs:double where a double literal meets a decimal one.
//   - A string literal against a numeric literal has no B.2 row at all.
func ctaComparisonSpace(l, r ctaValue) (ctaSpace, bool) {
	ls, lLiteral := ctaOperandSpace(l)
	rs, rLiteral := ctaOperandSpace(r)
	if !lLiteral && !rLiteral {
		return ctaSpaceString, true
	}
	if !lLiteral || !rLiteral {
		literal := ls
		if !lLiteral {
			literal = rs
		}
		if literal == ctaSpaceString {
			return ctaSpaceString, true
		}
		return ctaSpaceDouble, true
	}
	if ls == rs {
		return ls, true
	}
	if ls == ctaSpaceString || rs == ctaSpaceString {
		return ctaSpaceString, false
	}
	return ctaSpaceDouble, true
}

// ctaOperandSpace reports the value space of one operand and whether it has
// one at all: a literal carries its own type, an attribute is
// xs:untypedAtomic, which is not a space this file compares in but the thing
// §3.5.2's casting rules consume.
func ctaOperandSpace(v ctaValue) (ctaSpace, bool) {
	lit, isLiteral := v.(ctaLiteral)
	if !isLiteral {
		return ctaSpaceString, false
	}
	return lit.space, true
}

// ctaComparator is one operator of [13] ta-Comparator. All six are GENERAL
// comparisons (xpath20.md §3.5.2), never the eq/ne/lt/le/gt/ge value
// comparisons, which this grammar has no production for.
type ctaComparator byte

const (
	ctaEqual ctaComparator = iota
	ctaNotEqual
	ctaLess
	ctaLessEqual
	ctaGreater
	ctaGreaterEqual
)

// ctaAnswer is what one node of a compiled {test} evaluates to: an XPath
// boolean, or ctaError — the state key-cta-ta-select clause 2 (§3.12.4)
// describes as "a dynamic error or a type error is raised during evaluation".
//
// The third state exists because clause 2's subject is "the {test}" and not
// the node that raised. A node that raises therefore reports it rather than
// deciding for the whole expression: fn:not propagates it and xpath20.md
// §3.6's truth tables say what and/or do with it, and only
// [CTATest.Evaluate] substitutes false for it.
//
// ctaFalse is the zero value, which is what the nil root of a zero CTATest
// answers.
type ctaAnswer byte

const (
	ctaFalse ctaAnswer = iota
	ctaTrue
	ctaError
)

// ctaAnswerOf lifts a decided boolean into a ctaAnswer.
func ctaAnswerOf(b bool) ctaAnswer {
	if b {
		return ctaTrue
	}
	return ctaFalse
}

// negated is fn:not (xpath20.md §3.6): it inverts a boolean and propagates a
// raised error unchanged — "If an error is encountered in finding the
// effective boolean value of its operand, fn:not raises the same error."
// fn:not is a function and not a logical operator, so no truth table lets it
// absorb its operand's error into a boolean; only the {test} as a whole
// absorbs it, at [CTATest.Evaluate].
func (a ctaAnswer) negated() ctaAnswer {
	if a == ctaError {
		return ctaError
	}
	return ctaAnswerOf(a == ctaFalse)
}

// ctaEval evaluates one boolean-valued node.
//
// The default arm covers exactly one shape: the nil root of a zero CTATest,
// which no successful [CompileCTATest] produces. Every branch of the sum above
// is named.
func ctaEval(x ctaExpr, env ctaEnv) ctaAnswer {
	switch n := x.(type) {
	case ctaOr:
		return n.eval(env)
	case ctaAnd:
		return n.eval(env)
	case ctaNot:
		return ctaEval(n.operand, env).negated()
	case ctaCompare:
		return n.eval(env)
	case ctaEffectiveBoolean:
		return n.eval(env)
	case ctaTypeError:
		return ctaError
	default:
		return ctaFalse
	}
}

// eval decides an or-expression against xpath20.md §3.6's or-table, read with
// XPath 1.0 compatibility mode false (xpath-valid clause 2.2.1).
//
// A true operand determines the whole expression and stops the walk, whether
// or not an earlier operand raised: the table's two cells pairing an error
// with a true permit either true or the error, and true is the answer an
// or-expression whose determining operand succeeded has always had. Every
// other cell holding an error is the error, so an error survives to the end of
// the loop when nothing decides.
func (n ctaOr) eval(env ctaEnv) ctaAnswer {
	answer := ctaFalse
	for _, o := range n.operands {
		got := ctaEval(o, env)
		if got == ctaTrue {
			return ctaTrue
		}
		if got == ctaError {
			answer = ctaError
		}
	}
	return answer
}

// eval decides an and-expression on ctaOr.eval's terms, against §3.6's
// and-table: a false operand determines the expression and stops the walk, and
// an error otherwise survives to the end.
func (n ctaAnd) eval(env ctaEnv) ctaAnswer {
	answer := ctaTrue
	for _, o := range n.operands {
		got := ctaEval(o, env)
		if got == ctaFalse {
			return ctaFalse
		}
		if got == ctaError {
			answer = ctaError
		}
	}
	return answer
}

// eval decides one general comparison (xpath20.md §3.5.2). Both operands are
// singletons or empty here — an attribute is at most one node and a literal is
// exactly one value — so the existential over pairs reduces to: no pair at all
// where either operand is absent, and one pair otherwise.
//
// An ABSENT operand is a decided false and not an error: forming no pair is
// what §3.5.2 asks of the empty sequence, and nothing is raised. A failed cast
// of a PRESENT operand is err:FORG0001, so it is ctaError and the enclosing
// operators decide what becomes of it.
func (c ctaCompare) eval(env ctaEnv) ctaAnswer {
	left, leftPresent := ctaLexical(c.left, env)
	right, rightPresent := ctaLexical(c.right, env)
	if !leftPresent || !rightPresent {
		return ctaFalse
	}
	if c.space == ctaSpaceString {
		return ctaAnswerOf(c.op.holdsSign(strings.Compare(left, right)))
	}
	lv, lok := ctaParseIn(env.backend, c.space, left)
	rv, rok := ctaParseIn(env.backend, c.space, right)
	if !lok || !rok {
		return ctaError
	}
	return c.op.holdsBetween(lv, rv)
}

// eval decides the ·effective boolean value· of a bare ValueExpr (xpath20.md
// §2.4.3, the fn:boolean rules quoted there).
//
// The two arms are rules 2 and 4/5. An AttrName evaluates to a sequence of
// attribute NODES, so a present attribute is rule 2's "sequence whose first
// item is a node" and an absent one is rule 1's empty sequence. A literal is a
// singleton atomic value: rule 4 for xs:string (false iff zero length), rule 5
// for the numeric types (false iff numerically equal to zero — NaN, rule 5's
// other false, has no literal in XPath's grammar to reach this by).
func (e ctaEffectiveBoolean) eval(env ctaEnv) ctaAnswer {
	switch v := e.operand.(type) {
	case ctaAttr:
		_, present := env.attr(v.name)
		return ctaAnswerOf(present)
	case ctaLiteral:
		if v.space == ctaSpaceString {
			return ctaAnswerOf(v.text != "")
		}
		return ctaNonZero(env.backend, v)
	default:
		return ctaFalse
	}
}

// ctaNonZero reports whether a numeric literal is numerically unequal to zero,
// which is its ·effective boolean value·. It asks the backend rather than
// reading the lexical, so "0.0", "0" and "0E0" are one answer decided in the
// value space and not three lexical special cases.
//
// A cast that fails is err:FORG0001 like any other, hence ctaError. It is
// unreachable for a backend meeting [value.Backend]'s contract: the lexical is
// one the tokenizer already admitted as an XPath numeric literal, and every
// such literal is in the lexical space of the ctaSpace its kind fixes.
func ctaNonZero(b value.Backend, lit ctaLiteral) ctaAnswer {
	v, ok := ctaParseIn(b, lit.space, lit.text)
	if !ok {
		return ctaError
	}
	zero, ok := ctaParseIn(b, lit.space, "0")
	if !ok {
		return ctaError
	}
	return ctaNotEqual.holdsBetween(v, zero)
}

// ctaLexical reads one operand's lexical form, reporting false where the
// operand is the empty sequence — which only an absent attribute is.
func ctaLexical(v ctaValue, env ctaEnv) (string, bool) {
	switch n := v.(type) {
	case ctaAttr:
		return env.attr(n.name)
	case ctaLiteral:
		return n.text, true
	default:
		return "", false
	}
}

// ctaXMLSpace is the whiteSpace characters XML recognizes, which the collapse
// the numeric datatypes' whiteSpace facet prescribes removes from the ends of
// a lexical before the cast reads it (F&O §17.1.1: "Whitespace normalization
// is applied as indicated by the whiteSpace facet for the datatype").
//
// TRIMMING is the whole of collapse for these two targets, not an
// approximation of it: the lexical spaces of xs:decimal and xs:double contain
// no space character at all, so a lexical whose interior white space collapse
// would rewrite is outside both spaces before and after the rewrite, and the
// cast fails either way with the same err:FORG0001. value's own
// normalizeWhiteSpace is not reachable for this — it is unexported and keyed
// off a resolved xs:SimpleType's facet chain, and no simple type is in hand
// here.
const ctaXMLSpace = " \t\r\n"

// ctaParseIn casts a lexical into space, reporting false where the cast fails.
// A failed cast is err:FORG0001 — a dynamic error, hence key-cta-ta-select
// clause 2's false at the caller and never a decline.
//
// The [value.Context] is nil because it cannot be consulted: only the QName
// and NOTATION mappings are context-dependent (PRINCIPLES 19), and neither is
// a ctaSpace.
func ctaParseIn(b value.Backend, space ctaSpace, lexical string) (value.Value, bool) {
	m, mapped := b.Mapping(ctaSpaceName(space))
	if !mapped {
		return nil, false
	}
	v, err := m.Parse(strings.Trim(lexical, ctaXMLSpace), nil)
	if err != nil {
		return nil, false
	}
	return v, true
}

// holdsSign reports whether op holds for two operands whose collation
// comparison yielded sign, which is how xpath20.md B.2 defines every one of
// the six operators over xs:string: through fn:compare against 0.
func (op ctaComparator) holdsSign(sign int) bool {
	switch op {
	case ctaEqual:
		return sign == 0
	case ctaNotEqual:
		return sign != 0
	case ctaLess:
		return sign < 0
	case ctaLessEqual:
		return sign <= 0
	case ctaGreater:
		return sign > 0
	case ctaGreaterEqual:
		return sign >= 0
	}
	return false
}

// holdsBetween reports whether op holds between two values of one numeric
// space, through the capability interfaces the values themselves carry (STYLE
// T2) and never a type switch over a backend's concrete types.
//
// A value carrying neither capability cannot be compared at all, which is
// err:XPTY0004 and so ctaError. It is unreachable for a backend meeting
// [value.Backend]'s contract, which value/backendtest checks mechanically:
// xs:decimal and xs:double are bounded, so their values are [value.Ordered]
// and therefore [value.Eq] too.
func (op ctaComparator) holdsBetween(l, r value.Value) ctaAnswer {
	if op == ctaEqual || op == ctaNotEqual {
		eq, comparable := l.(value.Eq)
		if !comparable {
			return ctaError
		}
		return ctaAnswerOf(eq.Eq(r) == (op == ctaEqual))
	}
	ord, ordered := l.(value.Ordered)
	if !ordered {
		return ctaError
	}
	return ctaAnswerOf(op.holdsOrdering(ord.Cmp(r)))
}

// holdsOrdering reports whether op holds for two values that compared as o.
// [value.Incomparable] is false for all four ordering operators: no magnitude
// relationship holds between values no order relates, which is §3.5.2's
// "otherwise the result of the comparison is false".
func (op ctaComparator) holdsOrdering(o value.Ordering) bool {
	switch o {
	case value.Less:
		return op == ctaLess || op == ctaLessEqual
	case value.Equal:
		return op == ctaLessEqual || op == ctaGreaterEqual
	case value.Greater:
		return op == ctaGreater || op == ctaGreaterEqual
	case value.Incomparable:
		return false
	}
	return false
}
