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
// (key-cta-ta-select clause 1's Note), so an UNCAST attribute operand
// atomizes to xs:untypedAtomic and xpath20.md §3.5.2's untypedAtomic casting
// rules are the normal path of every comparison here rather than an edge
// case. The Note also names the remedy the grammar gives a test that needs
// type information — "explicit casts will sometimes be necessary" — which is
// [15]'s cast tail and [18]'s constructor function, both evaluated here.
//
// EVERY CAST IS A DATATYPE VALIDATION. xpath-functions.md §17.1.1 fixes that
// outright for an xs:string or xs:untypedAtomic operand: "Whitespace
// normalization is applied as indicated by the whiteSpace facet for the
// datatype. The resulting whitespace-normalized string must be a valid
// lexical form for the datatype. The semantics of casting are identical to
// XML Schema validation." So a cast is [value.ValidateLexical] — the whole
// facet pipeline, whiteSpace and pattern and value facets included — and
// never a lexical parser of this package's own (STYLE T4).

// AttributeValue reads one attribute of the element information item E whose
// ·selected type definition· is being decided: the [[normalized value]] of
// E.[[attributes]] member whose ·expanded name· is name, and whether E carries
// such an attribute at all.
//
// An ABSENT attribute — present false — atomizes to the empty sequence.
// xpath20.md §3.5.2 makes a general comparison true only when "there is a pair
// of atomic values, one in the first operand sequence and the other in the
// second", so a comparison naming an attribute E does not carry is false;
// §3.10.2 rule 3 makes the same absence a type error (err:XPTY0004) under a
// `cast as T` written without the `?` occurrence indicator. That is neither a
// decline nor an error to the caller in either case.
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
// types resolves every QName the expression names a datatype with — [15]
// ta-CastExpr's target, [18] ta-ConstructorFunction's function name, and the
// builtin types §3.5.2's casting rules answer with — and is read here and
// stored nowhere: the compiled tree holds the resolved *[xsd.SimpleType]
// components themselves, which is why [CTATest.Evaluate] takes a resolver
// again rather than reading one off the tree.
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
//     ctaparser.go that recognizes it;
//   - a comparison whose two operands need a type promotion this engine cannot
//     perform, which today is exactly B.1 rule 1.1's xs:float to xs:double
//     (ctaWider). Declining is what keeps the withhold and the DECIDED false
//     apart: a promotion that cannot be performed is not err:XPTY0004 and must
//     not become one, because that error is a decided false the caller reads as
//     "this alternative did not select";
//   - a cast target this engine does not cast to, which is TWO conditions
//     sharing one encoding deliberately. A target naming an in-scope atomic
//     type outside the XSD namespace is valid XPath outside §3.12.6's
//     required subset, which that section's own Note licenses a processor to
//     decline ("Conforming processors may but are not required to support
//     XPath expressions not belonging to the required subset of XPath"); a
//     target that resolves to nothing, to a complex type, to a non-atomic
//     type or to xs:anyAtomicType/xs:NOTATION is err:XPST0051/err:XPST0080, a
//     STATIC error and so an xpath-valid clause 2 failure. The encoding
//     tracks the CONSEQUENCE, and both consequences are the one withhold
//     validate/cta.go's conditionallySelected argues: the element's
//     ·governing type definition· is not determined, and no type is assessed
//     against in its place.
//
// It never returns an error: a decline is not a verdict about the schema.
//
// The static context is xpath-valid clause 2.2's: XPath 1.0 compatibility mode
// false (2.2.1), statically known namespaces from expr.NamespaceBindings()
// (2.2.2), the default element/type namespace from expr.DefaultNamespace()
// (2.2.3) and the default function namespace
// http://www.w3.org/2005/xpath-functions (2.2.4). The default element/type
// namespace is read only where [xsd.XPathExpression.DefaultNamespace] reports
// it present, because that accessor's own doc makes the first result "not
// meaningful" otherwise; an ABSENT one leaves ctaNames.defaultNamespace the
// empty string, which is the no-namespace answer §3.10.2 wants. It is
// consulted for an unprefixed cast TARGET and for nothing else (xpath20.md
// §3.10.2: "If the target type has no namespace prefix, it is considered to be
// in the default element/type namespace"): [17] ta-AttrName makes every
// NameTest in this grammar an attribute-axis one, whose principal node kind is
// never element, so an unprefixed NameTest is always in no namespace
// (xpath20.md §3.2.1.2, PRINCIPLES 15).
func CompileCTATest(expr xsd.XPathExpression, types xsd.TypeResolver) (CTATest, bool) {
	toks, ok := ctaTokenize(expr.Expression())
	if !ok {
		return CTATest{}, false
	}
	known, ok := ctaResolveTypes(types)
	if !ok {
		return CTATest{}, false
	}
	names := ctaNames{prefixes: make(map[string]string)}
	for _, b := range expr.NamespaceBindings() {
		names.prefixes[b.Prefix()] = b.Namespace()
	}
	if defaultNS, present := expr.DefaultNamespace(); present {
		names.defaultNamespace = defaultNS
	}
	p := ctaParser{toks: toks, names: names, types: known}
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
// subset — a cast that fails (err:FORG0001), an absent operand under a cast
// written without `?` or operand types the operator does not accept
// (err:XPTY0004) — is a dynamic or type error, which clause 2 makes a false
// rather than an error or a decline. The declines all happened at
// [CompileCTATest].
//
// Clause 2's subject is "the {test}", not the sub-expression that raised, so
// this is the ONE place the substitution happens: a raised error travels up
// the expression tree under XPath's own rules for each operator and becomes
// false here, at the root. An error under an fn:not is therefore a false
// {test} and never the inversion of one.
//
// b supplies the value spaces the comparisons and the casts run in and must be
// non-nil. types resolves the {base type definition} chain of every type the
// compiled tree holds — the same capability [CompileCTATest] classified those
// types with, threaded as a parameter and stored nowhere (ARCHITECTURE), so
// one compiled [CTATest] serves any resolver that answers for its components.
func (t CTATest) Evaluate(b value.Backend, types xsd.TypeResolver, attr AttributeValue) bool {
	return ctaEval(t.root, ctaEnv{backend: b, types: types, attr: attr}) == ctaTrue
}

// ctaEnv is the dynamic context of one [CTATest.Evaluate] call. cvc-xpath
// (§3.13.4.2) fixes the rest of it — context item E, context position and size
// 1, no variable values — and none of that is representable in this grammar,
// which reaches no context item and has no VarRef production, so the
// attributes, the value spaces and the type knowledge the casts need are the
// whole of what evaluation reads.
type ctaEnv struct {
	backend value.Backend
	types   xsd.TypeResolver
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
// a general comparison (xpath20.md §3.5.2) whose comparison type — the one
// type §3.5.2's casting rules convert BOTH operands into — was settled at
// compile time by ctaTypes.comparison.
type ctaCompare struct {
	op         ctaComparator
	comparison *xsd.SimpleType
	left       ctaValue
	right      ctaValue
}

// ctaEffectiveBoolean is [11] ta-BooleanExpr's third arm with its Comparator
// ABSENT — a bare ValueExpr standing in a boolean position, whose value is its
// ·effective boolean value· (xpath20.md §2.4.3, fn:boolean).
type ctaEffectiveBoolean struct{ operand ctaValue }

// ctaTypeError is a comparison whose operand types are not a valid combination
// for its operator under xpath20.md B.2 — no one type serves both operands,
// which ctaTypes.comparison decides. XPath raises err:XPTY0004 for it, so the
// node evaluates to ctaError: a raised error the enclosing operators carry,
// not a decline and not a node-local false.
type ctaTypeError struct{}

func (ctaOr) ctaExpr()               {}
func (ctaAnd) ctaExpr()              {}
func (ctaNot) ctaExpr()              {}
func (ctaCompare) ctaExpr()          {}
func (ctaEffectiveBoolean) ctaExpr() {}
func (ctaTypeError) ctaExpr()        {}

// ctaValue is the sealed sum of the ITEM-valued nodes: the two arms of [16]
// ta-SimpleValue, and the cast that [15] ta-CastExpr's tail and [18]
// ta-ConstructorFunction both build over one of them.
type ctaValue interface{ ctaValue() }

// ctaAttr is [17] ta-AttrName, with its NameTest ALREADY resolved to an
// ·expanded name·: a prefixed one against the {namespace bindings}, an
// unprefixed one to no namespace, because the attribute axis's principal node
// kind is never element (xpath20.md §3.2.1.2). That asymmetry is settled at
// compile time so evaluation is a comparison of expanded names carrying no
// axis of its own.
type ctaAttr struct{ name xsd.QName }

// ctaLiteral is the Literal arm of [16] ta-SimpleValue, carrying the builtin
// datatype its XPath literal kind fixes: a StringLiteral is xs:string, an
// IntegerLiteral or DecimalLiteral is xs:decimal (xs:integer's primitive base,
// which is the type two such literals are compared in), and a DoubleLiteral is
// xs:double.
type ctaLiteral struct {
	text string
	st   *xsd.SimpleType
}

// ctaCast is [15] ta-CastExpr's `cast as QName '?'?` tail and, equivalently,
// [18] ta-ConstructorFunction: xpath20.md §3.10.4 DEFINES the constructor
// function call T($arg) as (($arg) cast as T?), so one node serves both and a
// second implementation would be STYLE T4 duplication.
//
// target is a builtin datatype, which §3.12.6 clause 4 fixes for the cast
// spelling ("Any explicit casts ... are casts to built-in datatypes") and
// clause 3 for the constructor spelling ("Any strings matching the
// ConstructorFunction production are function calls to constructor functions
// for the built-in datatypes").
//
// allowsEmpty is the `?` occurrence indicator, and it is load-bearing rather
// than cosmetic: xpath20.md §3.10.2 rule 3 makes an empty operand the result
// of the cast when `?` is present and err:XPTY0004 when it is not. §3.10.4's
// equivalence carries the `?` — "This example is equivalent to ("2000-01-01"
// cast as xs:date?)" — so the constructor spelling always allows it, whatever
// [18]'s own production says.
type ctaCast struct {
	operand     ctaValue
	target      *xsd.SimpleType
	allowsEmpty bool
}

func (ctaAttr) ctaValue()    {}
func (ctaLiteral) ctaValue() {}
func (ctaCast) ctaValue()    {}

// ctaStatic is one operand's static type, which is what xpath20.md §3.5.2's
// casting rules dispatch on. It is a sealed sum of the two states this grammar
// can produce and not a datatype: an uncast attribute has no type ANNOTATION
// at all, because key-cta-ta-select clause 1 labels every node of the
// constructed instance untyped.
type ctaStatic interface{ ctaStatic() }

// ctaUntypedAtomic is an uncast attribute operand, which atomizes to a single
// xs:untypedAtomic value (§3.13.4.1's note on the same "labeled as untyped"
// condition: "its atomized value will be a single atomic value of type
// untypedAtomic").
type ctaUntypedAtomic struct{}

// ctaTyped is an operand carrying a datatype: a Literal, or the result of a
// cast or a constructor function. It carries the COMPONENT alone — st.Name()
// is the name, and storing both would be two encodings of one fact (STYLE D3).
type ctaTyped struct{ st *xsd.SimpleType }

func (ctaUntypedAtomic) ctaStatic() {}
func (ctaTyped) ctaStatic()         {}

// ctaStaticOf reports the static type of one [14] ta-ValueExpr. The default
// arm is unreachable: every branch of the ctaValue sum above is named.
func ctaStaticOf(v ctaValue) ctaStatic {
	switch n := v.(type) {
	case ctaLiteral:
		return ctaTyped{st: n.st}
	case ctaCast:
		return ctaTyped{st: n.target}
	default:
		return ctaUntypedAtomic{}
	}
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
// singletons or empty here — an attribute is at most one node, a literal is
// exactly one value, and a cast of either is one or neither — so the
// existential over pairs reduces to: no pair at all where either operand is
// absent, and one pair otherwise.
//
// A RAISED operand decides first, whatever the other operand is: §3.5.2 says
// a general comparison "may raise a dynamic error as soon as it encounters an
// error in evaluating either operand". An ABSENT operand is then a decided
// false and not an error, because forming no pair is what §3.5.2 asks of the
// empty sequence and nothing is raised.
//
// Both operands reach the comparison already converted into c.comparison,
// which is where §3.5.2 clause 2's casts happened, so what is left here is
// clause 3's value comparison — and B.1's URI promotion, which is applied
// HERE and not in the comparison type: an xs:anyURI comparison type is decided
// through the default collation exactly as an xs:string one is, because B.1
// rule 2 promotes xs:anyURI to xs:string and B.2 then gives it the same six
// fn:compare rows. Promoting the TYPE instead would change what clause 2.4
// casts an xs:untypedAtomic operand to and so change the answer
// (ctaTypes.comparison).
func (c ctaCompare) eval(env ctaEnv) ctaAnswer {
	left := ctaItemOf(c.left, c.comparison, env)
	right := ctaItemOf(c.right, c.comparison, env)
	if ctaRaisedItem(left) || ctaRaisedItem(right) {
		return ctaError
	}
	l, leftPresent := left.(ctaAtom)
	r, rightPresent := right.(ctaAtom)
	if !leftPresent || !rightPresent {
		return ctaFalse
	}
	if ctaStringLike(c.comparison) {
		return c.op.holdsCollated(l.v, r.v)
	}
	return c.op.holdsBetween(l.v, r.v)
}

// eval decides the ·effective boolean value· of a bare ValueExpr (xpath20.md
// §2.4.3, the fn:boolean rules quoted there).
//
// An AttrName evaluates to a sequence of attribute NODES rather than to an
// atomic value, so it takes rule 2 ("a sequence whose first item is a node")
// when the attribute is present and rule 1 (the empty sequence) when it is
// not, and no type of its own is involved. Every other operand is a singleton
// atomic value or the empty sequence, which ctaBoolean decides.
func (e ctaEffectiveBoolean) eval(env ctaEnv) ctaAnswer {
	switch n := e.operand.(type) {
	case ctaAttr:
		_, present := env.attr(n.name)
		return ctaAnswerOf(present)
	case ctaLiteral:
		return ctaBoolean(e.operand, n.st, env)
	case ctaCast:
		return ctaBoolean(e.operand, n.target, env)
	default:
		return ctaFalse
	}
}

// ctaBoolean is fn:boolean over a singleton atomic operand of type st
// (xpath20.md §2.4.3 rules 1 and 3 through 6), which it decides in st's
// {primitive type definition}: the rules quantify over "xs:string ... or a
// type derived from one of these", and the primitive is what answers that for
// every builtin at once instead of a name list per rule.
//
// Rule 6 — "in all other cases, fn:boolean raises a type error [err:FORG0006]"
// — is the fallthrough, and it is reachable: this grammar can cast to any
// builtin atomic datatype, and only the boolean, string, anyURI and numeric
// families have a rule.
//
// An unresolvable primitive is ctaError on the same terms as any other type
// fault (ctaValidate). It is unreachable for a tree this package compiled,
// which resolved the primitive of every literal and every cast target before
// admitting it.
func ctaBoolean(v ctaValue, st *xsd.SimpleType, env ctaEnv) ctaAnswer {
	p, err := st.Primitive(env.types)
	if err != nil || p == nil {
		return ctaError
	}
	switch item := ctaItemOf(v, p, env).(type) {
	case ctaAbsent:
		return ctaFalse
	case ctaAtom:
		return ctaBooleanOf(item.v, p, env)
	default:
		return ctaError
	}
}

// ctaBooleanOf applies fn:boolean's per-type rules to one atomic value of
// primitive type p, through the capability interfaces the value itself carries
// (STYLE T2) and never a type switch over a backend's concrete types.
//
//   - Rule 3, xs:boolean: the value unchanged, read as equality against the
//     value of the lexical "true".
//   - Rule 4, xs:string and xs:anyURI: false iff the value has zero length,
//     which is [value.Lengthed]'s answer and the same one the length facet
//     reads.
//   - Rule 5, the numeric types: false iff NaN or numerically zero. NaN is
//     detected as a value UNEQUAL TO ITSELF, which [value.Eq] states outright
//     for float and double, so no NaN literal or backend-specific probe is
//     needed.
//   - Rule 6, everything else: err:FORG0006, hence ctaError.
func ctaBooleanOf(v value.Value, p *xsd.SimpleType, env ctaEnv) ctaAnswer {
	eq, comparable := v.(value.Eq)
	switch p.Name() {
	case ctaBuiltin("boolean"):
		if !comparable {
			return ctaError
		}
		return ctaEqualsLexical(eq, p, "true", env)
	case ctaBuiltin("string"), ctaBuiltin("anyURI"):
		measured, lengthed := v.(value.Lengthed)
		if !lengthed {
			return ctaError
		}
		return ctaAnswerOf(measured.Len() != 0)
	case ctaBuiltin("decimal"), ctaBuiltin("float"), ctaBuiltin("double"):
		if !comparable {
			return ctaError
		}
		if !eq.Eq(v) {
			return ctaFalse
		}
		return ctaEqualsLexical(eq, p, "0", env).negated()
	}
	return ctaError
}

// ctaEqualsLexical reports whether eq equals the value of lexical in type p,
// which is how fn:boolean's rules 3 and 5 reach the two constants they name
// (true, and numeric zero) without any backend-specific value construction.
// A lexical outside p's value space is ctaError, and neither of the two is:
// "true" is in xs:boolean's lexical space and "0" in every numeric one.
func ctaEqualsLexical(eq value.Eq, p *xsd.SimpleType, lexical string, env ctaEnv) ctaAnswer {
	against, isAtom := ctaValidate(lexical, p, env).(ctaAtom)
	if !isAtom {
		return ctaError
	}
	return ctaAnswerOf(eq.Eq(against.v))
}

// ctaItem is what one [14] ta-ValueExpr contributes to the operator above it,
// and it has THREE states rather than a value and a presence flag: xpath20.md
// §3.10.2 rule 3 splits an absent operand into the empty SEQUENCE (with `?`)
// and a raised err:XPTY0004 (without it), and those two are the opposite of
// each other under fn:not. It is a sealed sum (STYLE T1/T2) so no fourth state
// and no illegal pairing of the three is representable.
type ctaItem interface{ ctaItem() }

// ctaAbsent is the empty sequence: an attribute the element does not carry, or
// a cast of one written with the `?` occurrence indicator.
type ctaAbsent struct{}

// ctaRaised is a dynamic or type error raised while producing the item —
// err:FORG0001 from a lexical or facet mismatch, err:XPTY0004 from an absent
// operand under a cast written without `?` or from a cast this processor does
// not support. Which of the two it was is not carried: key-cta-ta-select
// clause 2 (§3.12.4) gives both the same consequence, and the {test} is where
// that consequence is applied.
type ctaRaised struct{}

// ctaAtom is one atomic value, already in the type the enclosing operator
// asked for.
type ctaAtom struct{ v value.Value }

func (ctaAbsent) ctaItem() {}
func (ctaRaised) ctaItem() {}
func (ctaAtom) ctaItem()   {}

// ctaRaisedItem reports whether an item is a raised error, which several
// deciders test before reading the item itself.
func ctaRaisedItem(i ctaItem) bool {
	_, raised := i.(ctaRaised)
	return raised
}

// ctaItemOf evaluates one [14] ta-ValueExpr to the single item it yields,
// converted into type c — the type the enclosing operator compares or reads it
// in.
//
// The three arms are the three ways an item acquires a type:
//
//   - an ATTRIBUTE is xs:untypedAtomic, which §3.5.2's casting rules cast
//     STRAIGHT to c (clause 1's xs:string, or clause 2's type chosen from the
//     other operand). No intermediate type exists to cast through.
//   - a LITERAL carries its own type and is converted to c, which is a no-op
//     wherever the two coincide.
//   - a CAST evaluates its operand IN THE TARGET TYPE first, because that cast
//     is the expression the author wrote and its failure is the author's
//     err:FORG0001, and only then converts the result to c. Evaluating it
//     straight into c instead would let `@n cast as xs:integer` accept "3.5"
//     whenever the comparison happened to run in xs:double.
//
// The default arm is unreachable: every branch of the ctaValue sum is named.
func ctaItemOf(v ctaValue, c *xsd.SimpleType, env ctaEnv) ctaItem {
	switch n := v.(type) {
	case ctaAttr:
		lexical, present := env.attr(n.name)
		if !present {
			return ctaAbsent{}
		}
		return ctaValidate(lexical, c, env)
	case ctaLiteral:
		return ctaConvert(n.text, n.st, c, env)
	case ctaCast:
		return ctaCastItem(n, c, env)
	default:
		return ctaAbsent{}
	}
}

// ctaCastItem evaluates one cast and converts its result into c.
//
// The empty-sequence rules are xpath20.md §3.10.2's rule 3, both halves: with
// the `?` occurrence indicator the cast of an empty operand IS the empty
// sequence, and without it the cast raises err:XPTY0004. That difference is
// the whole reason the `?` is carried on the node.
func ctaCastItem(n ctaCast, c *xsd.SimpleType, env ctaEnv) ctaItem {
	switch inner := ctaItemOf(n.operand, n.target, env).(type) {
	case ctaAbsent:
		if n.allowsEmpty {
			return ctaAbsent{}
		}
		return ctaRaised{}
	case ctaAtom:
		return ctaPromote(inner, n.target, c, env)
	default:
		return ctaRaised{}
	}
}

// ctaConvert casts lexical, a value of type from, into type to.
func ctaConvert(lexical string, from, to *xsd.SimpleType, env ctaEnv) ctaItem {
	item := ctaValidate(lexical, from, env)
	atom, isAtom := item.(ctaAtom)
	if !isAtom {
		return item
	}
	return ctaPromote(atom, from, to, env)
}

// ctaPromote converts one atomic value of type from into type to, which is
// what §3.5.2 clause 2's cast and B.1's type promotions ask of an operand
// whose own type is not the type the comparison runs in.
//
// It goes through the value's ·canonical representation·, which is the one
// lexical the spec guarantees maps back to that same value (Datatypes
// §2.3.1), so a conversion is one more datatype validation and never a
// backend-specific value translation this package would have to know the
// representations for. A value carrying no [value.Canonical] cannot be
// converted at all, which is err:XPTY0004 — the "casting is not supported"
// arm of xpath-functions.md §17 — and so ctaRaised. That is unreachable for
// the target set [CompileCTATest] admits: the canonical mapping is absent
// only for xs:QName and xs:NOTATION (value.Mapping), and a cast whose target
// has either as its primitive is declined at compile time.
func ctaPromote(atom ctaAtom, from, to *xsd.SimpleType, env ctaEnv) ctaItem {
	if from.Name() == to.Name() {
		return atom
	}
	canonical, renders := atom.v.(value.Canonical)
	if !renders {
		return ctaRaised{}
	}
	return ctaValidate(canonical.Canonical(), to, env)
}

// ctaValidate casts lexical into st, which xpath-functions.md §17.1.1 makes
// one datatype validation: "The semantics of casting are identical to XML
// Schema validation", whiteSpace normalization included. [value.ValidateLexical]
// is that validation, so this package holds no lexical parser of its own
// (STYLE T4).
//
// The [value.Context] is nil, and there is nothing for it to carry: only the
// QName and NOTATION mappings are context-dependent (PRINCIPLES 19), and a
// cast whose target has either as its primitive is declined at compile time.
//
// A failure is ctaRaised either way, and the branch is what says WHICH error
// it is (STYLE E2). A [value.IsDatatypeVerdict] error is a verdict about the
// lexical — err:FORG0001, "it is not possible to cast the input value into
// the value space of the target type". Anything else is a fault of the type
// or of the backend and says nothing about the lexical, so it is not that
// verdict; it is the one xpath-functions.md §17 gives an ST/TT pair this
// processor cannot cast between at all, err:XPTY0004. key-cta-ta-select
// clause 2 makes the {test} false for both.
func ctaValidate(lexical string, st *xsd.SimpleType, env ctaEnv) ctaItem {
	v, err := value.ValidateLexical(env.backend, env.types, st, lexical, nil)
	if err == nil {
		return ctaAtom{v: v}
	}
	if value.IsDatatypeVerdict(err) {
		return ctaRaised{} // err:FORG0001
	}
	return ctaRaised{} // err:XPTY0004
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

// holdsCollated decides op between two xs:string values under the DEFAULT
// COLLATION, which xpath-valid clause 2.2.10 fixes as the Unicode codepoint
// collation — a static-context property no backend owns, which is why this
// one comparison is decided here rather than through a value capability.
// xpath20.md B.2 routes all six operators on xs:string (and, through B.1's
// promotion, on xs:anyURI) through fn:compare, so one sign decides them all.
//
// The compared string is the value's ·canonical representation·, which is the
// identity for xs:string and xs:anyURI (f-stringCanmap, f-anyURICanmap) and so
// is the string the cast produced — whiteSpace-collapsed where the operand's
// own type collapses, as `cast as xs:token` does. A value carrying no
// [value.Canonical] is err:XPTY0004 on ctaPromote's terms.
func (op ctaComparator) holdsCollated(l, r value.Value) ctaAnswer {
	ls, lRenders := l.(value.Canonical)
	rs, rRenders := r.(value.Canonical)
	if !lRenders || !rRenders {
		return ctaError
	}
	return ctaAnswerOf(op.holdsSign(strings.Compare(ls.Canonical(), rs.Canonical())))
}

// holdsBetween reports whether op holds between two values of one comparison
// type, through the capability interfaces the values themselves carry (STYLE
// T2) and never a type switch over a backend's concrete types.
//
// GAP(xpath): xpath20.md B.2's PER-OPERATOR rows are not enforced. This engine
// admits any two operands one type serves and lets the values answer which
// operators they support, so where B.2 gives a type eq/ne rows but no ordering
// rows — xs:duration, the Gregorian types, xs:hexBinary, xs:base64Binary — an
// ordering comparison decides from the value space instead of raising
// err:XPTY0004, and where B.2 DOES give ordering rows a value space cannot
// serve (op:boolean-less-than on xs:boolean, whose values are deliberately not
// [value.Ordered]) it decides ctaError instead of comparing. The direction is
// UNESTABLISHED (STYLE P3a): the single consumer is validate/cta.go's
// conditionallySelected, which reads a true {test} as ·successfully selects·
// and a false one as the next alternative's turn, so a wrong answer in either
// direction hands the element a type §3.12.4 did not select, and both a false
// accept and a false reject follow from that. (#887)
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
