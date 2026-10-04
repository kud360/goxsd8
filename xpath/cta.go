package xpath

import (
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file evaluates the RESTRICTED expression subset a Type Alternative's
// {test} is written in — the "Test XPath expressions" grammar ta-props-correct
// clause 2.1 (§3.12.6) fixes, productions [8] ta-Test through [18]
// ta-ConstructorFunction — and nothing wider, but for the two productions the
// assertion façade adds and the Type Alternative one declines: [11]'s
// Comparator position also takes xpath20.md [23] ValueComp ('eq' | 'ne' | 'lt'
// | 'le' | 'gt' | 'ge'), evaluated as §3.5.1's value comparison
// (ctaValueCompare), and [16] SimpleValue also takes [44] VarRef ('$' VarName),
// of which only `$value` is in scope (cvc-assertion clause 2.2). It is not a
// stage of a general XPath 2.0 evaluator: the productions below reach no axis
// but attribute, no predicate, no variable but `$value` and no function but
// fn:not, so evaluating them directly is exact where a fail-open delegation to
// a general engine would be a guess.
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
// THREE ERROR DIRECTIONS MEET HERE AND NO TWO OF THEM ARE ONE.
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
// caller as a decline. An XPath STATIC error is neither: it is a
// ta-props-correct clause 2 (§3.12.6) violation the assembler charges before
// any instance exists, reported by [CTATestStaticError] and by nothing else in
// this package.
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

// Attributes yields the attributes of the element information item E whose
// ·selected type definition· is being decided — §3.12.4 clause 1.1.2's copy of
// E.[[attributes]], plus clause 1.1.3's non-clashing [[inherited attributes]] —
// each as its ·expanded name· and its [[normalized value]], in DOCUMENT ORDER.
// It must be non-nil, and a yield reporting false ends the walk.
//
// It is the ONE way this package reads an attribute, whatever NameTest [17]
// ta-AttrName carries: a QName NameTest filters this sequence to the one
// ·expanded name· it names, and each [37] Wildcard arm filters it its own way,
// so no second entry point encodes "read an attribute" a second time (STYLE
// D4).
//
// The sequence a NameTest matches is the operand xpath20.md §3.5.2 quantifies
// a general comparison over: true only when "there is a pair of atomic values,
// one in the first operand sequence and the other in the second", so a
// comparison matching NO attribute is false; §3.10.2 rule 3 makes that same
// empty sequence a type error (err:XPTY0004) under a `cast as T` written
// without the `?` occurrence indicator, and a matched sequence of two or more
// is err:XPTY0004 under any cast at all. That is neither a decline nor an error
// to the caller in any of those cases.
//
// The lexical is the attribute's [[normalized value]]: XML 1.0 §3.3.3
// attribute-value normalization applied, whiteSpace-facet normalization NOT —
// the latter belongs to the type a cast targets, and [CTATest.Evaluate]
// applies it per target type.
type Attributes func(yield func(name xsd.QName, lexical string) bool)

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
//     constraints charged at assembly by their owner, which reads this module
//     through [CTATestStaticError] and never through this entry point — so
//     nothing upstream has narrowed what arrives here and an error return would
//     be a false reject of every schema using full XPath in conditional type
//     assignment;
//   - a prefix with no binding in the record's {namespace bindings}
//     (err:XPST0081), which is a STATIC error and so not clause 2's false;
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
// It never returns an error: a decline is not a verdict about the schema. The
// verdict the STATIC errors two of those bullets name does carry is
// [CTATestStaticError]'s to report, over the same traversal and disagreeing
// with nothing here — this entry point still withholds them, because a tree
// built over an unresolved name or an unadmitted cast target is not evaluable
// whatever the schema's fate. Of the two, only the unbound prefix is reported
// today; the cast-target conditions are folded into this withhold and reach no
// charge, under the marker ctaTypes.castTarget carries.
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
	root, defect := compileCTATest(expr, types, ctaTypeAlternativeFacade{})
	if defect.kind != ctaNoDefect {
		return CTATest{}, false
	}
	return CTATest{root: root}, true
}

// ruleXPST0081 is the XPath static error an unbound namespace prefix is
// (xpath20.md Appendix G: "It is a static error if a QName used in an
// expression contains a namespace prefix that cannot be expanded into a
// namespace URI by using the statically known namespaces"), carried as the
// [xsderr.Rule] of the error [CTATestStaticError] returns.
//
// The CHARGE is the assembler's and the CODE is this package's (STYLE E2): a
// consumer reads this rule off the wrapped cause with [xsderr.RuleOf], never
// off a message it scraped, and reads the schema-side verdict —
// ta-props-correct — off the wrapper the assembler minted over it.
const ruleXPST0081 xsderr.Rule = "err:XPST0081"

// CTATestStaticError reports the XPath STATIC error the {test} of a Type
// Alternative carries, and nil for every other {expression} — including one
// this engine merely cannot evaluate.
//
// It answers the one question ta-props-correct clause 2 (§3.12.6) asks of a
// {test}: whether it "satisfies the constraint XPath Valid (§3.13.6.2)", whose
// clause 2 is "X does not produce any static error". That is a Schema Component
// Constraint, decided when the component is assembled and independent of
// whether any instance is ever ·assessed·, so the charge belongs to the
// assembler: it mints ta-props-correct at the <alternative>'s location and
// wraps what this returns as the cause. The CODE is this package's, because
// err:XPST0081 is XPath's static error and not the assembler's rule, so the
// result is an *[xsderr.Error] carrying it (ruleXPST0081) and a consumer reads
// it with [xsderr.RuleOf] after one errors.Unwrap of the charge.
//
// UNSUPPORTED DOMINATES STATIC. A static error is reported only where the token
// stream parsed to its end as a complete [8] ta-Test production and name
// resolution was the sole defect. `count(//p:a) > 1` with p unbound is legal
// full XPath 2.0 that this engine declines under §3.12.6 clause 2's Note
// ("Conforming processors may but are not required to support XPath
// expressions not belonging to the required subset of XPath"), and reading its
// unbound prefix in isolation would turn that decline into the rejection of a
// conforming schema. Under-charging is a rejection this engine can take later;
// over-charging is a false reject now.
//
// types is the IN-SCOPE SCHEMA DEFINITIONS of the static context this question
// is asked in, which xpath-valid clause 2.2.5 fixes as "those components that
// are present in every schema by definition" — the built-ins of §3.2.7, §3.4.7
// and §3.16.7, and NOT the schema's own {type definitions}, which that clause
// does not put in scope. It is read exactly as [CompileCTATest] reads it and
// stored nowhere.
//
// The rule is the XPath error code and the message is the fact, so the result
// renders as
//
//	?: [err:XPST0081] no statically-known namespace binding for prefix "p"
//
// naming the FIRST unbound prefix in expression order (STYLE D2). The
// [xsderr.Loc] is the zero one, which renders as that `?`: this package reads
// no schema document and so has no position to report — the real one is the
// <alternative>'s, attached by the assembler that charges over this.
func CTATestStaticError(expr xsd.XPathExpression, types xsd.TypeResolver) error {
	_, defect := compileCTATest(expr, types, ctaTypeAlternativeFacade{})
	if defect.kind != ctaStaticError {
		// An untyped nil, never a nil *xsderr.Error in an error interface: a
		// caller's `!= nil` must mean what it says.
		return nil
	}
	return defect.static
}

// compileCTATest is the ONE traversal the entry points above and
// [CompileAssertionTest] are façades over (STYLE T4): it tokenizes the
// {expression}, resolves the builtin datatypes the compiler names, indexes the
// {namespace bindings} and parses [8] ta-Test, reporting the tree and what — if
// anything — was wrong with it.
//
// facade is the only thing the façades differ in: what one [17] ta-AttrName
// compiles to, and which settled comparison types they evaluate —
// ctaTypeAlternativeFacade for a Type Alternative, whose instance is untyped,
// and ctaAssertionFacade for an assertion, whose attributes are typed.
//
// Name resolution never fails a parse — it records and carries on — so a parse
// that failed at all failed for another reason, and is ctaUnsupported with
// whatever an unbound prefix recorded along the way DISCARDED. That is where
// "unsupported dominates static" is enforced, once, for every façade.
func compileCTATest(expr xsd.XPathExpression, types xsd.TypeResolver, facade ctaFacade) (ctaExpr, ctaDefect) {
	toks, ok := ctaTokenize(expr.Expression())
	if !ok {
		return nil, ctaDefect{kind: ctaUnsupported}
	}
	known, ok := ctaResolveTypes(types)
	if !ok {
		return nil, ctaDefect{kind: ctaUnsupported}
	}
	names := ctaNames{prefixes: make(map[string]string)}
	for _, b := range expr.NamespaceBindings() {
		names.prefixes[b.Prefix()] = b.Namespace()
	}
	if defaultNS, present := expr.DefaultNamespace(); present {
		names.defaultNamespace = defaultNS
	}
	p := ctaParser{toks: toks, names: names, types: known, facade: facade}
	root, ok := p.test()
	if !ok {
		return nil, ctaDefect{kind: ctaUnsupported}
	}
	return root, p.defect
}

// ctaDefect is what compileCTATest found wrong with an {expression}. The three
// states are sealed inside this package and none of them crosses the boundary:
// each exported entry point answers about ONE of them, so no caller ever
// classifies a defect and no caller can mistake a decline for a verdict.
//
// The zero value is "nothing wrong", which is the only state that yields an
// evaluable tree.
type ctaDefect struct {
	kind ctaDefectKind
	// static is the verdict of a ctaStaticError defect, carrying the XPath error
	// code it violates as its [xsderr.Rule] — the error [CTATestStaticError]
	// hands the assembler to wrap. The other kinds leave it nil because they
	// have nothing to say: an {expression} outside what this engine evaluates is
	// not a verdict about the schema, so there is no fact to report about it.
	static *xsderr.Error
}

// ctaDefectKind is the kind of a [ctaDefect].
type ctaDefectKind byte

const (
	// ctaNoDefect is a complete [8] ta-Test production this engine evaluates.
	ctaNoDefect ctaDefectKind = iota
	// ctaUnsupported is an {expression} this engine cannot evaluate — full XPath
	// 2.0 outside the required subset, or one of the subset constructs whose own
	// production declines it. It is the fail-open direction (PRINCIPLES 20) and
	// never a schema fault.
	ctaUnsupported
	// ctaStaticError is a complete ta-Test production carrying an XPath static
	// error, which xpath-valid clause 2 (§3.13.6.2) forbids outright.
	ctaStaticError
)

// Evaluate reports whether the compiled {test} holds for the element
// information item whose attributes attrs yields, which is the whole of what
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
func (t CTATest) Evaluate(b value.Backend, types xsd.TypeResolver, attrs Attributes) bool {
	return ctaEval(t.root, ctaEnv{backend: b, types: types, input: ctaLexicalInput{attrs: attrs}}) == ctaTrue
}

// ctaEnv is the dynamic context of one [CTATest.Evaluate] or
// [AssertionTest.Evaluate] call. cvc-xpath (§3.13.4.2) fixes the rest of it —
// context item E, context position and size 1, no variable values but the
// `$value` cvc-assertion clause 2.3 adds — and none of that but `$value` is
// representable in this grammar, which reaches no context item, so the
// attributes, `$value`'s binding, the value spaces and the type knowledge the
// casts need are the whole of what evaluation reads.
type ctaEnv struct {
	backend value.Backend
	types   xsd.TypeResolver
	input   ctaInput
}

// ctaInput is the sealed sum of the two attribute inputs an evaluation reads,
// one per façade: E's attributes as LEXICALS for a Type Alternative, whose
// instance is untyped (key-cta-ta-select clause 1's Note), and as TYPED values
// for an assertion (cvc-assertion clause 1, §3.13.4.1). The two are one field
// and not two nil-able ones, so no environment carries both.
//
// Each façade pairs its own tree with its own input: [CTATest.Evaluate] builds
// ctaLexicalInput over a tree of ctaAttr nodes, and [AssertionTest.Evaluate]
// builds ctaTypedInput over a tree of ctaTypedAttr nodes, so a node never meets
// the other input.
type ctaInput interface{ ctaInput() }

// ctaLexicalInput is a Type Alternative's attribute input.
type ctaLexicalInput struct{ attrs Attributes }

// ctaTypedInput is an assertion's input: its typed attributes, and the value
// cvc-assertion clause 2.3 binds to `$value`. The binding lives here and on no
// other arm, so a Type Alternative's evaluation cannot carry one.
type ctaTypedInput struct {
	attrs TypedAttributes
	value ValueBinding
}

func (ctaLexicalInput) ctaInput() {}
func (ctaTypedInput) ctaInput()   {}

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

// ctaValueCompare is a value comparison (xpath20.md §3.5.1), [10]
// ComparisonExpr's ValueComp arm, which only the assertion façade admits
// (ctaFacade.comparesValues). Its comparison type — the one type §3.5.1 converts
// both atomized operands into — was settled at compile time by
// ctaTypes.valueComparison, so the node is B.2-legal by construction as a
// ctaCompare is.
//
// It is a node of its own and not a ctaCompare with a flag, because the two
// quantify differently over the same operand sequences: a general comparison is
// existential over them, and a value comparison admits at most one item per
// operand and answers the EMPTY SEQUENCE for an empty one (ctaValueCompare.eval).
type ctaValueCompare struct {
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
// for its operator under xpath20.md B.2 — no one type serves both operands, or
// B.2 holds no row for that operator over the type that does, both of which
// ctaTypes.comparison decides. XPath raises err:XPTY0004 for it, so the node
// evaluates to ctaError: a raised error the enclosing operators carry, not a
// decline and not a node-local false.
type ctaTypeError struct{}

func (ctaOr) ctaExpr()               {}
func (ctaAnd) ctaExpr()              {}
func (ctaNot) ctaExpr()              {}
func (ctaCompare) ctaExpr()          {}
func (ctaValueCompare) ctaExpr()     {}
func (ctaEffectiveBoolean) ctaExpr() {}
func (ctaTypeError) ctaExpr()        {}

// ctaValue is the sealed sum of the ITEM-valued nodes: the arms of [16]
// ta-SimpleValue — its AttrName arm in the untyped and the typed form, one per
// façade (ctaFacade.attribute), its Literal arm, and the assertion façade's
// `$value` in its two static forms (ctaFacade.variable) — and the cast that [15]
// ta-CastExpr's tail and [18] ta-ConstructorFunction both build over one of
// them.
type ctaValue interface{ ctaValue() }

// ctaAttr is [17] ta-AttrName over an UNTYPED instance: the attribute step
// whose NameTest selects a SEQUENCE of E's attributes, in document order, out
// of what [Attributes] yields.
//
// The NameTest is settled at compile time, so evaluation carries no axis and no
// prefix of its own — every name it could resolve is already an ·expanded name·
// (ctaNameTest).
type ctaAttr struct{ test ctaNameTest }

// ctaTypedAttr is [17] ta-AttrName over a TYPED instance, which is an
// assertion's (ctaAssertionFacade): the attribute E carries under the ·expanded
// name· name, at most one, whose typed value is of type st — the {type
// definition} [AttributeTypes] answered for that name at compile time, which is
// why the node carries it and the operand's static type is st rather than
// xs:untypedAtomic.
//
// Only a QName NameTest builds one: a [37] Wildcard arm can match an attribute
// ·attributed to· an {attribute wildcard}, whose type is not fixed at compile
// time, so ctaAssertionFacade declines it.
type ctaTypedAttr struct {
	name xsd.QName
	st   *xsd.SimpleType
}

// ctaValueVar is `$value` over a simple {content type} (cvc-assertion clause
// 2.3.1): the XDM representation of E's [schema actual value], read from the
// [ValueBinding] the evaluation carries. atom is the type of each item — the
// {simple type definition} itself, or its {item type definition} where listed
// is true and the value is a list, whose items Datatypes dt-xdmrep flattens
// into the sequence. Both were settled at compile time (ctaTypes.valueVariable),
// which is why the operand's static type is atom.
type ctaValueVar struct {
	atom   *xsd.SimpleType
	listed bool
}

// ctaEmptyValue is `$value` under any {content type} that is not simple
// (cvc-assertion clause 2.3.2): the empty sequence, decided at compile time,
// so the evaluation's [ValueBinding] is never read.
type ctaEmptyValue struct{}

// ctaFacade is the sealed sum of the façades compileCTATest parses for — one
// value, so the attribute node a façade builds and the comparisons it admits
// can never come from two different façades (STYLE T1). The grammar's two
// consumers close the set (STYLE T2's schema-closed-set exception).
type ctaFacade interface {
	ctaFacade()
	// attribute compiles one [17] ta-AttrName whose NameTest resolved to test
	// into its node, reporting false where the façade declines it — which
	// declines the whole expression on [CompileCTATest]'s withhold terms. The
	// types are the compile's own, for a façade that classifies the type it
	// reads.
	attribute(test ctaNameTest, types ctaTypes) (ctaValue, bool)
	// admitsComparison reports whether the façade evaluates a comparison —
	// general or value — whose operands ctaTypes.comparison or
	// ctaTypes.valueComparison settled into c, reporting false where it declines
	// the whole expression on the same withhold terms.
	admitsComparison(types ctaTypes, c *xsd.SimpleType) bool
	// comparesValues reports whether the façade admits xpath20.md [23]
	// ValueComp at all, which §3.12.6's grammar has no production for.
	comparesValues() bool
	// variable compiles one [44] VarRef naming name into its node, reporting
	// false where the façade declines it — every name not in its static
	// context, which is a static error (err:XPST0008) withheld on the same
	// terms as attribute's decline.
	variable(name xsd.QName, types ctaTypes) (ctaValue, bool)
}

// ctaTypeAlternativeFacade is a Type Alternative's façade: every NameTest is
// admitted and reads E's attributes untyped, and every settled comparison type
// is evaluated.
type ctaTypeAlternativeFacade struct{}

func (ctaTypeAlternativeFacade) ctaFacade() {}

func (ctaTypeAlternativeFacade) attribute(test ctaNameTest, _ ctaTypes) (ctaValue, bool) {
	return ctaAttr{test: test}, true
}

func (ctaTypeAlternativeFacade) admitsComparison(ctaTypes, *xsd.SimpleType) bool {
	return true
}

// comparesValues is false: [13] ta-Comparator spells the general comparators
// alone, and a value comparison is outside the required subset §3.12.6's Note
// licenses a processor to decline.
func (ctaTypeAlternativeFacade) comparesValues() bool { return false }

// variable declines every name: ta-props-correct clause 2's grammar has no
// VarRef, and xpath-valid clause 2.2.6 leaves a Type Alternative's in-scope
// variables empty, so `$value` there is err:XPST0008 — a static error this
// engine declines rather than charges, on the "unsupported dominates static"
// terms [CTATestStaticError] states.
func (ctaTypeAlternativeFacade) variable(xsd.QName, ctaTypes) (ctaValue, bool) {
	return nil, false
}

// ctaNameTest is the sealed sum of [36] NameTest's arms as [17] ta-AttrName
// reaches them, matching one ·expanded name· at a time on the ATTRIBUTE axis,
// whose principal node kind is attribute and never element (xpath20.md
// §3.2.1.2). The grammar closes the set (STYLE T2's schema-closed-set
// exception), so no further arm is representable outside this package.
type ctaNameTest interface {
	ctaNameTest()
	// matches reports whether an attribute with this ·expanded name· is
	// selected by the test.
	matches(name xsd.QName) bool
}

// ctaExactName is [36]'s QName arm, resolved: a prefixed NameTest against the
// {namespace bindings}, an unprefixed one to NO namespace, because the
// {default namespace} is the default ELEMENT/type namespace and this axis's
// principal node kind is never element (PRINCIPLES 15).
//
// A NameTest whose prefix has no binding holds ctaUnresolvedName instead, and
// no such node is ever evaluated: it comes with a ctaStaticError defect, on
// which [CompileCTATest] withholds.
type ctaExactName struct{ name xsd.QName }

// ctaAnyName is [37] Wildcard's `*` arm: "a node test * is true for any node of
// the principal node kind of the step axis" (xpath20.md §3.2.1.2), which on the
// attribute axis is every attribute of E.
type ctaAnyName struct{}

// ctaAnyLocal is [37]'s `NCName ':' '*'` arm, whose prefix is ALREADY resolved
// against the {namespace bindings}: "true for any node ... whose expanded QName
// has the namespace URI to which the prefix is bound, regardless of the local
// part of the name".
//
// The empty space is a real matcher — every attribute in no namespace — and not
// a sentinel, which is why an unbound prefix takes ctaUnresolvedTest instead.
type ctaAnyLocal struct{ space string }

// ctaAnySpace is [37]'s `'*' ':' NCName` arm: "true for any node ... whose local
// name matches the given NCName, regardless of its namespace or lack of a
// namespace". No prefix is resolved, so this arm has no err:XPST0081 exposure.
type ctaAnySpace struct{ local string }

// ctaUnresolvedTest is the NameTest an UNBOUND prefix in [37]'s `NCName ':' '*'`
// resolves to. It matches nothing and is never evaluated: it comes with a
// ctaStaticError defect, on which [CompileCTATest] withholds — ctaUnresolvedName's
// role for a QName NameTest, as a TYPE rather than as a value, because no
// namespace URI is uninhabited and so none can stand in for one that resolved
// to nothing.
type ctaUnresolvedTest struct{}

func (ctaExactName) ctaNameTest()      {}
func (ctaAnyName) ctaNameTest()        {}
func (ctaAnyLocal) ctaNameTest()       {}
func (ctaAnySpace) ctaNameTest()       {}
func (ctaUnresolvedTest) ctaNameTest() {}

func (t ctaExactName) matches(name xsd.QName) bool { return name == t.name }

func (ctaAnyName) matches(xsd.QName) bool { return true }

func (t ctaAnyLocal) matches(name xsd.QName) bool { return name.Space == t.space }

func (t ctaAnySpace) matches(name xsd.QName) bool { return name.Local == t.local }

func (ctaUnresolvedTest) matches(xsd.QName) bool { return false }

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

func (ctaAttr) ctaValue()       {}
func (ctaTypedAttr) ctaValue()  {}
func (ctaLiteral) ctaValue()    {}
func (ctaCast) ctaValue()       {}
func (ctaValueVar) ctaValue()   {}
func (ctaEmptyValue) ctaValue() {}

// ctaStatic is one operand's static type, which is what xpath20.md §3.5.2's
// casting rules dispatch on. It is a sealed sum of the three states this
// grammar can produce and not a datatype: an uncast UNTYPED attribute has no
// type ANNOTATION at all, because key-cta-ta-select clause 1 labels every node
// of the constructed instance untyped, and the statically empty `$value` has
// no item to carry one.
type ctaStatic interface{ ctaStatic() }

// ctaUntypedAtomic is an uncast attribute operand, which atomizes to a single
// xs:untypedAtomic value (§3.13.4.1's note on the same "labeled as untyped"
// condition: "its atomized value will be a single atomic value of type
// untypedAtomic").
type ctaUntypedAtomic struct{}

// ctaTyped is an operand carrying a datatype: a Literal, the result of a cast
// or a constructor function, a typed attribute, or each item of `$value`. It
// carries the COMPONENT alone — st.Name() is the name, and storing both would
// be two encodings of one fact (STYLE D3).
type ctaTyped struct{ st *xsd.SimpleType }

// ctaEmptySequence is the statically empty operand, ctaEmptyValue: it yields
// no item, so no operator is applied to it and no type is involved
// (ctaTypes.againstEmpty).
type ctaEmptySequence struct{}

func (ctaUntypedAtomic) ctaStatic() {}
func (ctaTyped) ctaStatic()         {}
func (ctaEmptySequence) ctaStatic() {}

// ctaStaticOf reports the static type of one [14] ta-ValueExpr. ctaAttr is the
// one untyped arm.
func ctaStaticOf(v ctaValue) ctaStatic {
	switch n := v.(type) {
	case ctaLiteral:
		return ctaTyped{st: n.st}
	case ctaCast:
		return ctaTyped{st: n.target}
	case ctaTypedAttr:
		return ctaTyped{st: n.st}
	case ctaValueVar:
		return ctaTyped{st: n.atom}
	case ctaEmptyValue:
		return ctaEmptySequence{}
	default:
		return ctaUntypedAtomic{}
	}
}

// ctaComparator is one of the six comparison operators: a [13] ta-Comparator
// spelling in a general comparison (xpath20.md §3.5.2, ctaCompare), or a [23]
// ValueComp spelling in a value comparison (§3.5.1, ctaValueCompare). Each is
// named for the B.2 rows both spellings of it read — `=` and `eq` the
// `A eq B` row — and the node it sits on is what says how it quantifies.
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
// [CTATest.Evaluate] and [AssertionTest.Evaluate] substitute false for it —
// the second because cvc-assertion's subject is the {test} too.
//
// ctaFalse is the zero value, which is what the nil root of a zero CTATest or
// AssertionTest answers.
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
// absorbs it, at [CTATest.Evaluate] or [AssertionTest.Evaluate].
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
	case ctaValueCompare:
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

// eval decides one general comparison (xpath20.md §3.5.2), which is
// EXISTENTIAL over the two operand sequences: "the result of the comparison is
// true if and only if there is a pair of atomic values, one in the first
// operand sequence and the other in the second operand sequence, that have the
// required magnitude relationship. Otherwise the result of the comparison is
// false." An operand matching no attribute forms no pair and so decides false,
// which is not an error.
//
// A pair that raises decides for the whole comparison, on the same latitude the
// operands take below; the pairs are walked in document order, so which one
// that is does not vary between runs (STYLE D1).
//
// A RAISED operand decides first, whatever the other operand is: §3.5.2 says
// a general comparison "may raise a dynamic error as soon as it encounters an
// error in evaluating either operand" — which is also why an operand is
// converted whole, so one item of a matched sequence failing §3.5.2 clause 2's
// cast raises for the operand rather than being dropped from it.
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
//
// xs:boolean takes a third route for the same kind of reason: the operator
// functions B.2 names for it are defined over the two values themselves and
// not over an order the value space carries (holdsBoolean).
func (c ctaCompare) eval(env ctaEnv) ctaAnswer {
	l, leftAtoms := ctaItemOf(c.left, c.comparison, env).(ctaAtoms)
	r, rightAtoms := ctaItemOf(c.right, c.comparison, env).(ctaAtoms)
	if !leftAtoms || !rightAtoms {
		return ctaError // ctaRaised, the sum's only other arm
	}
	for _, lv := range l.vs {
		for _, rv := range r.vs {
			if got := ctaHoldsPair(c.op, c.comparison, lv, rv, env); got != ctaFalse {
				return got
			}
		}
	}
	return ctaFalse
}

// ctaHoldsPair decides op between two values both already converted into the
// comparison type c. It is the one decision both comparison nodes reach: one
// PAIR of a general comparison's existential is §3.5.2 clause 3's value
// comparison, and a value comparison (§3.5.1) is that same application of the
// operator to its two singletons.
func ctaHoldsPair(op ctaComparator, c *xsd.SimpleType, l, r value.Value, env ctaEnv) ctaAnswer {
	if ctaStringLike(c) {
		return op.holdsCollated(l, r)
	}
	if c.Name() == ctaBuiltin("boolean") {
		return op.holdsBoolean(l, r, c, env)
	}
	return op.holdsBetween(l, r)
}

// eval decides one value comparison (xpath20.md §3.5.1), whose steps are
// applied to each operand in order, the left one first — the order is
// implementation-dependent, and fixing it fixes which answer a pair of faulty
// operands gets (STYLE D1):
//
//   - an operand that raised is the error;
//   - step 2, an EMPTY atomized operand: "the result of the value comparison is
//     an empty sequence";
//   - step 3, an operand of more than one item: err:XPTY0004.
//
// Step 4's xs:untypedAtomic cast and the conversion to the least common type
// both happened in converting each operand into c.comparison
// (ctaTypes.valueComparison), so what remains is the operator, on the one pair.
//
// The empty sequence is answered as ctaFalse, and that is exact rather than an
// approximation: every consumer of an [11] ta-BooleanExpr takes its ·effective
// boolean value· — §3.6's and/or and fn:not over their operands, and clause 3
// of cvc-assertion over the {test} — and the effective boolean value of the
// empty sequence is false (§2.4.3 rule 1). So `not(@x eq 1)` over an E without
// @x is true, where an err:XPTY0004 under the same fn:not would stay an error.
func (c ctaValueCompare) eval(env ctaEnv) ctaAnswer {
	l, settled, single := ctaSingletonOperand(c.left, c.comparison, env)
	if !single {
		return settled
	}
	r, settled, single := ctaSingletonOperand(c.right, c.comparison, env)
	if !single {
		return settled
	}
	return ctaHoldsPair(c.op, c.comparison, l, r, env)
}

// ctaSingletonOperand evaluates one value-comparison operand into c, reporting
// its one atomic value, or false with the answer the whole comparison takes
// without applying its operator: ctaError for a raised operand or one of two
// or more items (§3.5.1 step 3, err:XPTY0004), and ctaFalse — the empty
// sequence, read as ctaValueCompare.eval states — for an empty one (step 2).
func ctaSingletonOperand(v ctaValue, c *xsd.SimpleType, env ctaEnv) (value.Value, ctaAnswer, bool) {
	atoms, converted := ctaItemOf(v, c, env).(ctaAtoms)
	if !converted {
		return nil, ctaError, false
	}
	if len(atoms.vs) == 0 {
		return nil, ctaFalse, false
	}
	if len(atoms.vs) > 1 {
		return nil, ctaError, false // err:XPTY0004
	}
	return atoms.vs[0], ctaFalse, true
}

// eval decides the ·effective boolean value· of a bare ValueExpr (xpath20.md
// §2.4.3, the fn:boolean rules quoted there).
//
// An AttrName, untyped or typed, evaluates to a sequence of attribute NODES
// rather than to atomic values, so it takes rule 2 ("a sequence whose first
// item is a node") whenever its NameTest matches at all and rule 1 (the empty
// sequence) when it matches nothing, and no type of its own is involved. Rule
// 2 holds whatever the sequence's LENGTH, which is what a wildcard NameTest
// makes observable. `$value` is atomic values and no node: the statically empty
// one is rule 1's false, and the bound one is decided by ctaBoolean, a list of
// two or more items included. Every other operand is a singleton atomic value
// or the empty sequence, which ctaBoolean decides.
func (e ctaEffectiveBoolean) eval(env ctaEnv) ctaAnswer {
	switch n := e.operand.(type) {
	case ctaAttr:
		return ctaAnswerOf(len(ctaMatchedAttributes(n, env)) != 0)
	case ctaTypedAttr:
		return ctaAnswerOf(len(ctaMatchedTyped(n, env)) != 0)
	case ctaLiteral:
		return ctaBoolean(e.operand, n.st, env)
	case ctaCast:
		return ctaBoolean(e.operand, n.target, env)
	case ctaValueVar:
		return ctaBoolean(e.operand, n.atom, env)
	case ctaEmptyValue:
		return ctaFalse
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
//
// A sequence of TWO OR MORE items is rule 6 as well — fn:boolean is defined
// over "a sequence whose first item is not a node" of length one, and raises
// err:FORG0006 otherwise. A `$value` over a list {simple type definition}
// reaches it with its items; a cast can put a wildcard-matched sequence here
// too, but raises err:XPTY0004 for that length before this is reached.
func ctaBoolean(v ctaValue, st *xsd.SimpleType, env ctaEnv) ctaAnswer {
	p, err := st.Primitive(env.types)
	if err != nil || p == nil {
		return ctaError
	}
	atoms, converted := ctaItemOf(v, p, env).(ctaAtoms)
	if !converted {
		return ctaError
	}
	if len(atoms.vs) == 0 {
		return ctaFalse
	}
	if len(atoms.vs) > 1 {
		return ctaError // err:FORG0006
	}
	return ctaBooleanOf(atoms.vs[0], p, env)
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
		return ctaTruth(v, p, env)
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
	against, validated := ctaValidated(ctaValidate(lexical, p, env))
	if !validated {
		return ctaError
	}
	return ctaAnswerOf(eq.Eq(against))
}

// ctaItem is what one [14] ta-ValueExpr contributes to the operator above it,
// and it is a sequence or a raised error rather than a value and a presence
// flag: xpath20.md §3.10.2 rule 3 splits an EMPTY operand into the empty
// sequence (with `?`) and a raised err:XPTY0004 (without it), and those two are
// the opposite of each other under fn:not. It is a sealed sum (STYLE T1/T2) so
// no third state is representable.
type ctaItem interface{ ctaItem() }

// ctaAtoms is the sequence of atomic values an operand yields, in document
// order, already in the type the enclosing operator asked for. A zero-length
// slice IS the empty sequence — the XDM's own encoding, which a [17]
// ta-AttrName whose NameTest matches nothing produces and which a cast written
// with `?` produces over it — so no separate absence marker exists to disagree
// with it (STYLE D3).
type ctaAtoms struct{ vs []value.Value }

// ctaRaised is a dynamic or type error raised while producing the item —
// err:FORG0001 from a lexical or facet mismatch, err:XPTY0004 from an empty
// operand under a cast written without `?`, from a cast over a sequence of two
// or more items, or from a cast this processor does not support. Which of them
// it was is not carried: key-cta-ta-select clause 2 (§3.12.4) gives them all
// the same consequence, as cvc-assertion (§3.13.4.1) does for an assertion,
// and the {test} is where that consequence is applied.
type ctaRaised struct{}

func (ctaAtoms) ctaItem()  {}
func (ctaRaised) ctaItem() {}

// ctaSingleton is the one-item sequence a datatype validation yields.
func ctaSingleton(v value.Value) ctaItem { return ctaAtoms{vs: []value.Value{v}} }

// ctaValidated is the single value a ctaValidate or ctaPromote result carries,
// reporting false where it raised: both answer a SINGLETON sequence or
// ctaRaised, and their callers want the value itself.
func ctaValidated(i ctaItem) (value.Value, bool) {
	atoms, converted := i.(ctaAtoms)
	if !converted || len(atoms.vs) != 1 {
		return nil, false
	}
	return atoms.vs[0], true
}

// ctaItemOf evaluates one [14] ta-ValueExpr to the sequence it yields,
// converted into type c — the type the enclosing operator compares or reads it
// in.
//
// The arms are the ways an item acquires a type:
//
//   - an UNTYPED attribute is xs:untypedAtomic, which §3.5.2's casting rules
//     cast STRAIGHT to c (clause 1's xs:string, or clause 2's type chosen from
//     the other operand). No intermediate type exists to cast through.
//   - a TYPED attribute, a LITERAL and each item of `$value` carry their own
//     type and are converted to c, which is a no-op wherever the two coincide;
//     the statically empty `$value` yields nothing to convert.
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
		return ctaAttrItem(n, c, env)
	case ctaTypedAttr:
		return ctaTypedAttrItem(n, c, env)
	case ctaLiteral:
		return ctaConvert(n.text, n.st, c, env)
	case ctaCast:
		return ctaCastItem(n, c, env)
	case ctaValueVar:
		return ctaValueItem(n, c, env)
	case ctaEmptyValue:
		return ctaAtoms{}
	default:
		return ctaAtoms{}
	}
}

// ctaValueItem converts `$value`'s binding into c on ctaTypedAttrItem's terms
// (ctaPromote). The zero [ValueBinding] is the empty sequence (cvc-assertion
// clause 2.3.2). A listed n ranges the bound value's [value.Listed] items in
// order, each of type n.atom — the flattened sequence Datatypes dt-xdmrep makes
// a list value's XDM representation, so an empty list is the empty sequence and
// a list of two or more items is a sequence of that length.
//
// The input is ctaTypedInput by construction (ctaInput); the other arm binds
// nothing and is unreachable. A listed binding whose value does not carry
// [value.Listed] breaks the obligation [BindValue] states and is ctaRaised,
// unreachable for a caller that keeps it.
func ctaValueItem(n ctaValueVar, c *xsd.SimpleType, env ctaEnv) ctaItem {
	in, typed := env.input.(ctaTypedInput)
	if !typed || in.value.v == nil {
		return ctaAtoms{}
	}
	if !n.listed {
		return ctaPromote(in.value.v, n.atom, c, env)
	}
	list, isList := in.value.v.(value.Listed)
	if !isList {
		return ctaRaised{}
	}
	var vs []value.Value
	for item := range list.Items() {
		converted, ok := ctaValidated(ctaPromote(item, n.atom, c, env))
		if !ok {
			return ctaRaised{}
		}
		vs = append(vs, converted)
	}
	return ctaAtoms{vs: vs}
}

// ctaMatchedAttributes is the [[normalized value]]s of the attributes n's
// NameTest selects, in the DOCUMENT ORDER [Attributes] yields them in — which
// is the order of the operand sequence itself and so of every answer decided
// over it (STYLE D1).
//
// The whole sequence is walked: a QName NameTest matches at most one attribute
// because no element carries two of one ·expanded name·, and a [37] Wildcard
// arm has no such bound.
//
// The input is ctaLexicalInput by construction (ctaInput); the other arm
// matches nothing and is unreachable.
func ctaMatchedAttributes(n ctaAttr, env ctaEnv) []string {
	in, lexical := env.input.(ctaLexicalInput)
	if !lexical {
		return nil
	}
	var matched []string
	in.attrs(func(name xsd.QName, lexical string) bool {
		if n.test.matches(name) {
			matched = append(matched, lexical)
		}
		return true
	})
	return matched
}

// ctaMatchedTyped is ctaMatchedAttributes for a typed attribute: the typed
// values [TypedAttributes] yields under n's ·expanded name·, at most one, each
// of type n.st by the caller's obligation that type states.
//
// The input is ctaTypedInput by construction (ctaInput); the other arm matches
// nothing and is unreachable.
func ctaMatchedTyped(n ctaTypedAttr, env ctaEnv) []value.Value {
	in, typed := env.input.(ctaTypedInput)
	if !typed {
		return nil
	}
	var matched []value.Value
	in.attrs(func(name xsd.QName, v value.Value) bool {
		if name == n.name {
			matched = append(matched, v)
		}
		return true
	})
	return matched
}

// ctaTypedAttrItem converts the matched typed value into c on ctaPromote's
// terms, which is B.1's promotion or §3.5.2's conversion into the comparison
// type — never a re-validation of the attribute's lexical, which has none here.
func ctaTypedAttrItem(n ctaTypedAttr, c *xsd.SimpleType, env ctaEnv) ctaItem {
	matched := ctaMatchedTyped(n, env)
	vs := make([]value.Value, 0, len(matched))
	for _, v := range matched {
		converted, ok := ctaValidated(ctaPromote(v, n.st, c, env))
		if !ok {
			return ctaRaised{}
		}
		vs = append(vs, converted)
	}
	return ctaAtoms{vs: vs}
}

// ctaAttrItem casts the matched attributes into c, which §3.5.2's casting rules
// do to an xs:untypedAtomic operand.
//
// The conversion is EAGER over the whole sequence: one item that does not cast
// raises for the OPERAND, on §3.5.2's own latitude ("may raise a dynamic error
// as soon as it encounters an error in evaluating either operand"), rather than
// dropping out of it. So `@* = 3` over an element carrying n="3" and s="abc"
// raises and is false, where a per-pair conversion would answer true.
func ctaAttrItem(n ctaAttr, c *xsd.SimpleType, env ctaEnv) ctaItem {
	matched := ctaMatchedAttributes(n, env)
	vs := make([]value.Value, 0, len(matched))
	for _, lexical := range matched {
		v, validated := ctaValidated(ctaValidate(lexical, c, env))
		if !validated {
			return ctaRaised{}
		}
		vs = append(vs, v)
	}
	return ctaAtoms{vs: vs}
}

// ctaCastItem evaluates one cast and converts its result into c.
//
// The empty-sequence rules are xpath20.md §3.10.2's rule 3, both halves: with
// the `?` occurrence indicator the cast of an empty operand IS the empty
// sequence, and without it the cast raises err:XPTY0004. That difference is
// the whole reason the `?` is carried on the node.
//
// TWO OR MORE items raise err:XPTY0004 whether or not the `?` is written: rule
// 3 admits "exactly one atomic value" and `?` widens that to AT MOST one, never
// to more, so a cast over a wildcard NameTest matching several attributes is a
// type error and not the first of them.
func ctaCastItem(n ctaCast, c *xsd.SimpleType, env ctaEnv) ctaItem {
	inner, converted := ctaItemOf(n.operand, n.target, env).(ctaAtoms)
	if !converted {
		return ctaRaised{}
	}
	if len(inner.vs) == 0 {
		if n.allowsEmpty {
			return ctaAtoms{}
		}
		return ctaRaised{}
	}
	if len(inner.vs) > 1 {
		return ctaRaised{}
	}
	return ctaPromote(inner.vs[0], n.target, c, env)
}

// ctaConvert casts lexical, a value of type from, into type to.
func ctaConvert(lexical string, from, to *xsd.SimpleType, env ctaEnv) ctaItem {
	v, validated := ctaValidated(ctaValidate(lexical, from, env))
	if !validated {
		return ctaRaised{}
	}
	return ctaPromote(v, from, to, env)
}

// ctaPromote converts one atomic value of type from into type to, which is
// what §3.5.2 clause 2's cast and B.1's type promotions ask of an operand
// whose own type is not the type the comparison runs in.
//
// It goes through the value's ·canonical representation·, which is the one
// lexical the spec guarantees maps back to that same value (Datatypes
// §2.3.1), so a conversion is one more datatype validation and never a
// backend-specific value translation this package would have to know the
// representations for. ctaCanonical renders it. A value it cannot render
// cannot be converted, which is ctaRaised on either of two terms:
//
//   - no canonical mapping on from's chain, which is err:XPTY0004 — the
//     "casting is not supported" arm of xpath-functions.md §17. That is
//     unreachable for the target set [CompileCTATest] admits: the canonical
//     mapping is absent only for xs:QName and xs:NOTATION (value.Mapping),
//     and a cast whose target has either as its primitive is declined at
//     compile time.
//   - a canonical form beyond the backend's capacity (xmlschema11-2 §5.4,
//     the strict backend's precisionDecimal zero of a huge ·scale·), a
//     dynamic error raised as an implementation limit — indicated, never
//     rendered as a padded or substitute lexical. No compiled test converts
//     out of xs:precisionDecimal: it has no xpath20.md B.2 row, and castsFrom
//     declines a cast from a typed attribute that is not xs:string.
func ctaPromote(v value.Value, from, to *xsd.SimpleType, env ctaEnv) ctaItem {
	if from.Name() == to.Name() {
		return ctaSingleton(v)
	}
	lexical, rendered := ctaCanonical(v, from, env)
	if !rendered {
		return ctaRaised{}
	}
	return ctaValidate(lexical, to, env)
}

// ctaCanonical renders v, a value of type from, through the backend's
// [value.Mapping.Canonical] — the (string, error) channel, because
// [value.Canonical]'s string-only one has no outcome for a form beyond the
// backend's capacity, and a value whose type can reach one does not carry it.
//
// The mapping is the nearest one on from's {base type definition} chain, from
// itself included, which is the mapping that produced v (value.Backend's
// nearest-mapped-ancestor rule, which value.ValidateLexical applies).
// [value.Mapping] is keyed by name, so an anonymous or user-derived from
// reaches its builtin ancestor's. A mapping whose Canonical is nil, or errors
// on v, does not end the walk: its error is per-value (value.Mapping), and a
// partial-domain one says only that v has no form in THAT type's lexical
// space, so the next mapped ancestor's wider canonical mapping is asked —
// xs:yearMonthDuration's P0M has none of its own (§3.4.26.1 Note) and renders
// through xs:duration's as PT0S. Reporting false when no mapping on the chain
// renders v, or the chain cannot be walked, is the caller's ctaRaised.
func ctaCanonical(v value.Value, from *xsd.SimpleType, env ctaEnv) (string, bool) {
	for at := from; at != nil; {
		m, mapped := env.backend.Mapping(at.Name())
		if mapped && m.Canonical != nil {
			lexical, err := m.Canonical(v)
			if err == nil {
				return lexical, true
			}
		}
		base, err := at.Base(env.types)
		if err != nil {
			return "", false
		}
		at = base
	}
	return "", false
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
		return ctaSingleton(v)
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
// own type collapses, as `cast as xs:token` does. It is read through
// [value.Canonical], not ctaCanonical's mapping walk: an identity canonical
// form has no beyond-capacity case, and the string-only capability needs no
// type to find a mapping by. A value carrying none is err:XPTY0004, the
// "casting is not supported" arm ctaPromote also raises on.
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
// A value carrying neither capability is ctaError, and that is a fault of the
// BACKEND rather than a judgment about the operator: which operators an
// operand type admits is xpath20.md B.2's answer, settled when the node was
// built (ctaTypes.admitsComparison), so every pair reaching here is one B.2
// holds a row for.
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

// holdsBoolean decides op between two xs:boolean values, which B.2 gives all
// six rows: eq and ne through op:boolean-equal, and the four ordering
// operators through op:boolean-less-than and op:boolean-greater-than.
//
// Those two functions are defined over the VALUES and not over an order —
// "Returns true if $arg1 is false and $arg2 is true. Otherwise, returns false"
// (xpath-functions.md §9.2.2) — so the ordering is decided through
// [value.Eq] against the value of the lexical "true", exactly as fn:boolean
// reads an xs:boolean operand (ctaTruth). It is deliberately NOT decided
// through [value.Ordered]: that capability is the datatype's own order, which
// xs:boolean does not have ("The XML Schema datatype xs:boolean is not
// ordered", xpath-functions.md §9.2), and B.2's rows are about which operators
// an operand type admits rather than about the value space.
func (op ctaComparator) holdsBoolean(l, r value.Value, st *xsd.SimpleType, env ctaEnv) ctaAnswer {
	if op == ctaEqual || op == ctaNotEqual {
		return op.holdsBetween(l, r)
	}
	left := ctaTruth(l, st, env)
	right := ctaTruth(r, st, env)
	if left == ctaError || right == ctaError {
		return ctaError
	}
	return ctaAnswerOf(op.holdsOrdering(ctaBooleanOrdering(left == ctaTrue, right == ctaTrue)))
}

// ctaTruth reads one xs:boolean value of type st as an answer, which is
// equality against the value of the lexical "true" — fn:boolean's rule 3 for
// an xs:boolean operand (xpath20.md §2.4.3), and the reading
// op:boolean-less-than and op:boolean-greater-than are written in terms of.
func ctaTruth(v value.Value, st *xsd.SimpleType, env ctaEnv) ctaAnswer {
	eq, comparable := v.(value.Eq)
	if !comparable {
		return ctaError
	}
	return ctaEqualsLexical(eq, st, "true", env)
}

// ctaBooleanOrdering orders two xs:boolean values as op:boolean-less-than and
// op:boolean-greater-than order them: "false is less than true" and nothing
// else is related, which over two values is a total order and so never
// [value.Incomparable].
func ctaBooleanOrdering(l, r bool) value.Ordering {
	if l == r {
		return value.Equal
	}
	if !l {
		return value.Less
	}
	return value.Greater
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
