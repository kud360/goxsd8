package xpath

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// FacetAssertions is the [value.AssertionEvaluator] for an assertions facet's
// {test}s (Datatypes §4.3.13.3, cvc-assertions-valid). Its consumer is
// validate, which passes it at every value.ValidateLexical and
// value.ValidatingType call.
func FacetAssertions() value.AssertionEvaluator { return facetAssertions{} }

// facetAssertions is [FacetAssertions]' implementation.
type facetAssertions struct{}

// Evaluate declines every {test}.
func (facetAssertions) Evaluate(value.Backend, xsd.TypeResolver, *xsd.SimpleType, xsd.XPathExpression, value.Value) value.AssertionOutcome {
	return value.AssertionDeclined
}
