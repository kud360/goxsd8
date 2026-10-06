package xpath

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file is the FACET façade over the grammar compileCTATest parses (STYLE
// T4): an assertions facet's {test} (Datatypes §4.3.13, an Assertion's {test})
// written in the assertion façade's grammar is compiled and evaluated against
// one value, which Assertions Valid (§4.3.13.3, cvc-assertions-valid) asks of
// it. It differs from the assertion façade in the one way the rule's
// conditions differ from cvc-assertion's: there is no context item (clause
// 1.2), so every expression that reads one — [47] ContextItemExpr `.`, an
// attribute or child-axis step, a "/" or "//" opening a path, and the implicit
// argument of fn:string, fn:string-length and fn:normalize-space called with
// none — raises err:XPDY0002 (ctaNoContextItem), and `$value`, bound to the XDM
// representation of the value under the facet's type, or under its ·active
// basic member· where that type is a union (clause 1.4, dt-xdmrep clause 4),
// is the whole of what a {test} can read, arithmetic and the F&O string and
// sequence functions over it included (ctaFacetFacade.computes,
// ctaFacetFacade.callsLibrary), and fn:count over it or over any other operand
// that is no path. An fn:count call over a path declines
// (ctaFacetFacade.count).

// FacetAssertions is the [value.AssertionEvaluator] for an assertions facet's
// {test}s: it compiles the {test} under the facet's static context and
// evaluates it with no context item, `$value` bound to v ([value.AssertionEvaluator]
// states the contract). Its consumer is validate, which passes it at every
// value.ValidateLexical, value.ValidatingType and value.ConstraintMatches call,
// so a value's Datatype Valid verdict — and a union's ·validating type· and a
// fixed-value comparison with it — includes its assertions facets.
//
// Each {test} answers one of three outcomes:
//
//   - [value.AssertionHolds], where it evaluates to true;
//   - [value.AssertionFails], where it evaluates to false or raises a dynamic or
//     type error, which cvc-assertions-valid treats alike — err:XPDY0002 for a
//     read of the absent context item among them, arithmetic over one and
//     fn:string, fn:string-length and fn:normalize-space with no argument
//     included, and the err:FOAR0001 and err:FOAR0002 of arithmetic
//     ([AssertionTest.Evaluate] lists them);
//   - [value.AssertionDeclined], where this engine does not evaluate it: a
//     {test} [CompileAssertionTest] would decline over a simple {content type}
//     of the same type, on that function's terms — the grammar is the same and
//     so is every decline it states — and a {test} calling fn:count over a
//     path, `count(@a)`, which `count($value)` is not.
//
// A union's own assertions facet is evaluated like any other: the pipeline
// hands this evaluator the union's ·active basic member· as st, the type under
// which `$value` is the value's XDM representation (dt-xdmrep clause 4,
// cvc-assertions-valid clause 1.4), so a {test} is compiled against that
// member and declines only where it would over the member itself.
//
// GAP(xpath): an fn:count call over a path is declined, whose argument would
// raise err:XPDY0002 over the absent context item (ctaFacetFacade.count). Every other
// decline is [CompileAssertionTest]'s, under its GAP(xpath). The direction is
// the withhold: the caller declines the value's Datatype Valid verdict, never
// charging it and never showing it satisfied. (#1042)
//
// Nothing is cached: each call compiles its {test} afresh (STYLE D3), and b and
// r are read as [AssertionTest.Evaluate] reads them and stored nowhere.
func FacetAssertions() value.AssertionEvaluator { return facetAssertions{} }

// facetAssertions is [FacetAssertions]' implementation, fusing the compile and
// the evaluation so no compiled facet tree crosses the package boundary.
type facetAssertions struct{}

// Evaluate decides test against v, a value of st, on [FacetAssertions]' terms.
func (facetAssertions) Evaluate(b value.Backend, r xsd.TypeResolver, st *xsd.SimpleType, test xsd.XPathExpression, v value.Value) value.AssertionOutcome {
	root, defect := compileCTATest(test, r, ctaFacetFacade{st: st})
	if defect.kind != ctaNoDefect {
		return value.AssertionDeclined
	}
	// No string value is bound: the facet tree never reads one, since `.`
	// compiles to ctaNoContextItem (cvc-assertions-valid clause 1.2).
	in := ctaTypedInput{attrs: noTypedAttributes, children: noChildElements, value: BindValue("", Typed(v))}
	if ctaEval(root, ctaEnv{backend: b, types: r, input: in}) != ctaTrue {
		return value.AssertionFails
	}
	return value.AssertionHolds
}

// noTypedAttributes and noChildElements are the empty inputs a facet
// evaluation carries: a facet tree holds no attribute or child step to read
// either, since ctaFacetFacade compiles each to ctaNoContextItem.
func noTypedAttributes(func(xsd.QName, TypedValue) bool) {}

func noChildElements(func(ChildElement) bool) {}

// ctaFacetFacade is the facet façade compileCTATest parses for: `$value` is
// typed by st, the type [value.AssertionEvaluator] hands over — the simple type
// the assertions facet is effective on, or the ·active basic member· where that
// type is a union (dt-xdmrep clause 4) — and every read of the context item
// compiles to ctaNoContextItem.
type ctaFacetFacade struct{ st *xsd.SimpleType }

func (ctaFacetFacade) ctaFacade() {}

// attribute compiles any attribute step to ctaNoContextItem: the step's
// context node is the context item (xpath20.md §3.2.1), which is absent.
func (ctaFacetFacade) attribute(ctaNameTest, ctaTypes) (ctaValue, bool) {
	return ctaNoContextItem{}, true
}

// comparesValues is true: the {test} is full XPath 2.0, on
// ctaAssertionFacade.comparesValues' terms.
func (ctaFacetFacade) comparesValues() bool { return true }

// computes is true, on ctaAssertionFacade.computes' terms. An operand that
// reads the absent context item is ctaNoContextItem, whose err:XPDY0002 the
// arithmetic over it raises, so `. + 1` fails the facet rather than declining
// (cvc-assertions-valid clause 1.2's Note).
func (ctaFacetFacade) computes() bool { return true }

// callsLibrary is true, on ctaAssertionFacade.callsLibrary's terms. A call
// whose argument, written or implicit, reads the absent context item raises
// the err:XPDY0002 that argument's ctaNoContextItem does, so
// `string-length() > 0` fails the facet rather than declining.
func (ctaFacetFacade) callsLibrary() bool { return true }

// constructsSequences is true, on ctaAssertionFacade.constructsSequences'
// terms.
func (ctaFacetFacade) constructsSequences() bool { return true }

// variable compiles `$value` (clause 1.1: "no namespace URI and ... 'value' as
// the local name") against st as ctaTypes.valueVariable classifies it, the XDM
// representation of a value of st (dt-xdmrep), and declines every other name,
// which is not in the static context (err:XPST0008), on
// ctaAssertionFacade.variable's terms.
func (f ctaFacetFacade) variable(name xsd.QName, types ctaTypes) (ctaValue, bool) {
	if name != ctaValueName {
		return nil, false
	}
	return types.valueVariable(f.st)
}

// child compiles any child-axis step to ctaNoContextItem, on attribute's
// terms.
func (ctaFacetFacade) child(ctaNameTest, ctaTypes) (ctaValue, bool) {
	return ctaNoContextItem{}, true
}

// childPath compiles any path of child steps to ctaNoContextItem, on
// attribute's terms: its first step's context node is the absent context item.
func (ctaFacetFacade) childPath([]xsd.QName) (ctaValue, bool) {
	return ctaNoContextItem{}, true
}

// elements compiles any element step, `N`, `./N` or `.//N`, to
// ctaNoContextItem, on childPath's terms: `N` is a child step of the absent
// context item, and `./N` and `.//N` open with a read of it, `.`.
func (ctaFacetFacade) elements(ctaCountPath) (ctaValue, bool) {
	return ctaNoContextItem{}, true
}

// rooted compiles a path opening with "/" or "//" to ctaNoContextItem: the
// path begins at `fn:root(self::node())` (xpath20.md §3.2), a read of the
// context item that raises err:XPDY0002 before the `treat as` whose
// err:XPDY0050 ctaNoDocumentRoot is.
func (ctaFacetFacade) rooted() (ctaValue, bool) {
	return ctaNoContextItem{}, true
}

// contextItem compiles `.` to ctaNoContextItem (xpath20.md §3.1.4).
func (ctaFacetFacade) contextItem() (ctaValue, bool) {
	return ctaNoContextItem{}, true
}

// count declines every fn:count call over a path, which reads the absent
// context item and would raise err:XPDY0002 but is counted off a [Tally] the
// facet evaluation has none of; that decline is [FacetAssertions]' withhold,
// under its GAP(xpath). An fn:count call over an operand that is no path,
// `count($value)`, never reaches it (ctaParser.countCall).
func (ctaFacetFacade) count(ctaCounted, ctaTypes) (ctaValue, bool) {
	return nil, false
}
