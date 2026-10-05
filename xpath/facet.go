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
// attribute or child-axis step, a "/" or "//" opening a path — raises
// err:XPDY0002 (ctaNoContextItem), and `$value`, bound to the XDM
// representation of the value under the facet's type (clause 1.4, dt-xdmrep),
// is the whole of what a {test} can read.

// FacetAssertions is the [value.AssertionEvaluator] for an assertions facet's
// {test}s: it compiles the {test} under the facet's static context and
// evaluates it with no context item, `$value` bound to v ([value.AssertionEvaluator]
// states the contract). Its consumer is validate, which passes it at every
// value.ValidateLexical and value.ValidatingType call, so a value's Datatype
// Valid verdict — and a union's ·validating type· with it — includes its
// assertions facets.
//
// Each {test} answers one of three outcomes:
//
//   - [value.AssertionHolds], where it evaluates to true;
//   - [value.AssertionFails], where it evaluates to false or raises a dynamic or
//     type error, which cvc-assertions-valid treats alike — err:XPDY0002 for a
//     read of the absent context item among them;
//   - [value.AssertionDeclined], where this engine does not evaluate it: a
//     {test} [CompileAssertionTest] would decline over a simple {content type}
//     of the same type, on that function's terms — the grammar is the same and
//     so is every decline it states — and every {test} of a union's own
//     assertions facet.
//
// GAP(xpath): a union's own assertions facet is declined whatever its {test}:
// `$value` is the XDM representation of the value under the union's ·active
// basic member· (dt-xdmrep clause 4), which only the dispatch knows, and this
// evaluator is handed the union. Every other decline is [CompileAssertionTest]'s,
// under its GAP(xpath). The direction is the withhold: the caller declines
// the value's Datatype Valid verdict, never charging it and never showing it
// satisfied. (#1042)
//
// Nothing is cached: each call compiles its {test} afresh (STYLE D3), and b and
// r are read as [AssertionTest.Evaluate] reads them and stored nowhere.
func FacetAssertions() value.AssertionEvaluator { return facetAssertions{} }

// facetAssertions is [FacetAssertions]' implementation, fusing the compile and
// the evaluation so no compiled facet tree crosses the package boundary.
type facetAssertions struct{}

// Evaluate decides test against v, a value of st, on [FacetAssertions]' terms.
func (facetAssertions) Evaluate(b value.Backend, r xsd.TypeResolver, st *xsd.SimpleType, test xsd.XPathExpression, v value.Value) value.AssertionOutcome {
	variety, err := st.Variety(r)
	if err != nil {
		return value.AssertionDeclined
	}
	if _, isUnion := variety.(xsd.Union); isUnion {
		return value.AssertionDeclined
	}
	root, defect := compileCTATest(test, r, ctaFacetFacade{st: st})
	if defect.kind != ctaNoDefect {
		return value.AssertionDeclined
	}
	in := ctaTypedInput{attrs: noTypedAttributes, children: noChildElements, value: BindValue(Typed(v))}
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
// typed by st, the simple type the assertions facet is effective on, and every
// read of the context item compiles to ctaNoContextItem.
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
