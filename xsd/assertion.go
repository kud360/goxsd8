package xsd

// Assertion is the Assertion component (Structures §3.13.1, id="as"):
// a kind of Annotated Component with {test} (an XPathExpression property
// record, Required — unlike TypeAlternative's {test}, which is Optional).
//
// Assertion is a STRUCTURAL, opaque holder: {test} is preserved verbatim by
// the embedded XPathExpression (see its doc), never compiled or evaluated
// here. This lets complex/simple types carry their §3.13 assertions (and,
// via the Datatypes §4.3.13 assertions facet, simple-type assertions — the
// same Assertion type serves both hosts, per §4.3.13's mapping to §3.13.2)
// before the XPath engine (M6/M7) exists to evaluate cvc-assertion /
// as-props-correct.
//
// Construct only through NewAssertion. Assertion is immutable after
// construction.
type Assertion struct {
	test XPathExpression
}

// NewAssertion builds an Assertion. There is no rejectable state at this
// structural layer (see XPathExpression's NewXPathExpression doc) — hence
// no loc/error.
func NewAssertion(test XPathExpression) Assertion {
	a := Assertion{test: test}
	return a
}

// Test returns the {test} property: the Required XPathExpression this
// assertion evaluates (once the M6/M7 engine exists).
func (a Assertion) Test() XPathExpression {
	return a.test
}
