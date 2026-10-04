package xpath

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file is the ASSERTION façade over the §3.12.6 grammar compileCTATest
// parses (STYLE T4): an {assertions} member's {test} (§3.13.1) written in that
// grammar is compiled and evaluated against the element it guards, which
// cvc-assertion (§3.13.4.1) asks of it. It is the first slice of tier 2
// (doc.go) and nothing wider: the grammar is the Type Alternative one, and only
// the attribute step differs — an assertion's instance is TYPED.
//
// cvc-assertion clause 1 builds the XDM instance from the partial ·PSVI· of E,
// so each attribute carries the type its ·governing attribute declaration·
// gave it, and §3.13.2's own example relies on that: in `@min le @max` over
// two xs:int attributes "the typed values of the attributes are available for
// comparison; it is not necessary to cast". A Type Alternative's instance is
// untyped instead (key-cta-ta-select clause 1's Note), which is why the two
// façades build different nodes for one [17] ta-AttrName and take different
// attribute inputs, and why neither tree can be fed the other's input.
//
// `$value` (cvc-assertion clause 2.2) and the eq/ne/lt/le/gt/ge value
// comparisons are outside the grammar, so a {test} using either is declined
// here as any other expression outside it is.

// AttributeTypes answers, for the element information item E whose assertions
// are being compiled, the {type definition} an attribute of E with the
// ·expanded name· name is typed by: the type of the {attribute declaration} of
// the {attribute use} of E's ·governing type definition· matching that name. ok
// false means no type is fixed for the name at compile time — no such use, a
// name only an {attribute wildcard} can admit, or a use the caller declines to
// type — and [CompileAssertionTest] declines a {test} naming it.
//
// It is STATIC in the sense the compile needs: the answer does not depend on
// whether E carries the attribute. Its one consumer is validate's
// cvc-assertion site (validate/cvcassertion.go), which builds it and
// [TypedAttributes] from one lookup so the two cannot disagree.
type AttributeTypes func(name xsd.QName) (*xsd.SimpleType, bool)

// TypedAttributes yields the attributes E carries that matched an {attribute
// use} of its ·governing type definition·, each as its ·expanded name· and its
// ·actual value·, in DOCUMENT ORDER (STYLE D1). It must be non-nil, and a
// yield reporting false ends the walk.
//
// Each value must be of EXACTLY the type [AttributeTypes] answered for its
// name when the [AssertionTest] being evaluated was compiled: the tree holds
// that type and converts the value from it. That agreement is the caller's
// obligation.
//
// Unlike [Attributes], it carries no [inherited attributes]: the XDM instance
// cvc-assertion clause 1.3 builds contains E's own [[attributes]] and nothing
// from outside E.
type TypedAttributes func(yield func(name xsd.QName, v value.Value) bool)

// AssertionTest is a compiled assertion {test}: the expression tree
// [CompileAssertionTest] admitted for one element's attribute types. It is a
// distinct type from [CTATest] so a tree typed for an assertion can never be
// evaluated over lexicals, nor a Type Alternative tree over typed values.
type AssertionTest struct{ root ctaExpr }

// CompileAssertionTest compiles an assertion's {test} (§3.13.1, an
// [xsd.XPathExpression] property record) for the element whose attribute
// types attrs answers, reporting ok false for a {test} this engine cannot
// evaluate. Its consumer is validate's cvc-assertion site
// (validate/cvcassertion.go), which declines the assertion on ok false.
//
// The grammar is [CompileCTATest]'s and so is every decline it states, under
// the same static context (xpath-valid clause 2.2), plus these, each of which
// is the same withhold:
//
//   - an attribute NameTest that is not a QName: a [37] Wildcard can match an
//     attribute ·attributed to· an {attribute wildcard}, whose type is not
//     fixed at compile time;
//   - a QName for which attrs reports false;
//   - an attribute whose type does not atomize to one atomic value of a type
//     known at compile time — a list or union {variety}, xs:anySimpleType,
//     xs:anyAtomicType — or whose {primitive type definition} is xs:QName or
//     xs:NOTATION, which carry no ·canonical representation· to convert
//     through. The variety is classified here, not trusted to attrs;
//   - a cast whose operand is a typed attribute outside the xs:string family.
//
// An XPath STATIC error is declined too and never reported: the
// static-error question about an assertion is the schema assembler's, and
// [CTATestStaticError] answers it for a Type Alternative only.
//
// GAP(xpath): unlike a Type Alternative's, an assertion's {test} has no
// required subset to stop at — §3.13 admits full XPath 2.0 — so every decline
// above is this engine's limit and not the spec's license: `$value`, the value
// comparisons, paths and axes beyond the attribute step, and the F&O function
// library among them. The direction is the withhold: the caller records the
// assertion as unevaluated and neither charges it nor shows it satisfied
// (PRINCIPLES 20). (#1042)
//
// types is read as [CompileCTATest] reads it and stored nowhere.
func CompileAssertionTest(expr xsd.XPathExpression, types xsd.TypeResolver, attrs AttributeTypes) (AssertionTest, bool) {
	root, defect := compileCTATest(expr, types, assertionStep(attrs))
	if defect.kind != ctaNoDefect {
		return AssertionTest{}, false
	}
	return AssertionTest{root: root}, true
}

// Evaluate reports whether the compiled {test} evaluates to true for the
// element whose attributes attrs yields, WITHOUT raising a dynamic or type
// error — the whole of what cvc-assertion (§3.13.4.1) asks: "An element
// information item E is locally ·valid· with respect to an assertion if and
// only if the {test} evaluates to true (see below) without raising any dynamic
// error or type error." Clause 3 converts the result "as if by a call to the
// XPath fn:boolean function", which a boolean-rooted tree already is.
//
// So false is ONE answer for two outcomes the caller treats alike — the {test}
// was false, or it raised (err:FORG0001, err:XPTY0004, err:FORG0006) — and
// either way E is not ·valid· with respect to the assertion. A processor that
// raises a type error dynamically "will treat the expression as having
// evaluated to false" (cvc-xpath, §3.13.4.2). Every decline happened at
// [CompileAssertionTest].
//
// The dynamic context is cvc-xpath's: context item E, position and size 1,
// and no variable this grammar can name. b and types are read as
// [CTATest.Evaluate] reads them.
func (t AssertionTest) Evaluate(b value.Backend, types xsd.TypeResolver, attrs TypedAttributes) bool {
	return ctaEval(t.root, ctaEnv{backend: b, types: types, input: ctaTypedInput{attrs: attrs}}) == ctaTrue
}

// assertionStep is the assertion façade's attribute step: a QName NameTest
// whose name attrs types with a type this engine reads as one atomic value
// (ctaTypes.typedAttribute) compiles to a ctaTypedAttr, and every other
// NameTest declines.
//
// An unbound prefix's ctaUnresolvedName reaches attrs like any other name; no
// attribute use can carry it, so the step declines and the parse ends
// unsupported, which an assertion withholds on exactly as it would on the
// static error.
func assertionStep(attrs AttributeTypes) ctaAttributeStep {
	return func(test ctaNameTest, types ctaTypes) (ctaValue, bool) {
		exact, isExact := test.(ctaExactName)
		if !isExact {
			return nil, false
		}
		st, typed := attrs(exact.name)
		if !typed || !types.typedAttribute(st) {
			return nil, false
		}
		return ctaTypedAttr{name: exact.name, st: st}, true
	}
}
