package xsd

// Assertion is the Assertion component (Structures §3.13.1, id="as"):
// a kind of Annotated Component with {annotations} (a sequence of
// Annotation) and {test} (an XPathExpression property record, Required —
// unlike TypeAlternative's {test}, which is Optional).
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
	test        XPathExpression
	annotations []Annotation
}

// NewAssertion builds an Assertion. annotations is copied; the caller's
// backing array is not aliased. There is no rejectable state at this
// structural layer (see XPathExpression's NewXPathExpression doc) — hence
// no loc/error, mirroring NewAnnotation.
func NewAssertion(test XPathExpression, annotations []Annotation) Assertion {
	a := Assertion{test: test}
	if len(annotations) > 0 {
		a.annotations = append([]Annotation(nil), annotations...)
	}
	return a
}

// Test returns the {test} property: the Required XPathExpression this
// assertion evaluates (once the M6/M7 engine exists).
func (a Assertion) Test() XPathExpression {
	return a.test
}

// Annotations returns the {annotations} property in document order. It
// returns a copy: mutating the result does not affect a. An empty
// {annotations} yields nil.
func (a Assertion) Annotations() []Annotation {
	if len(a.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), a.annotations...)
}
