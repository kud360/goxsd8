package value

import (
	"errors"
	"fmt"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is the LAST stage of the facet pipeline (doc.go "The facet
// pipeline"): the assertions facet, Assertions Valid (Datatypes §4.3.13.3,
// cvc-assertions-valid), which cvc-datatype-valid clause 3 (dv_vfacets, §4.1.4)
// folds into Datatype Valid with the other ·value-based· facets. Its {test}s
// are XPath 2.0, which this package cannot evaluate — package xpath imports
// this one — so the stage asks an [AssertionEvaluator] the caller injects, and
// that capability's three answers are the stage's three outcomes.

// ruleCvcAssertionsValid is Assertions Valid (Datatypes §4.3.13.3,
// id="cvc-assertions-valid"): a value is facet-valid with respect to an
// assertions facet iff each {test} in its {value} evaluates to true without
// raising a dynamic or type error.
const ruleCvcAssertionsValid xsderr.Rule = "cvc-assertions-valid"

// AssertionOutcome is what an [AssertionEvaluator] decided of one {test} of an
// assertions facet: a sum of three states, not a (holds, decided) pair, so no
// answer can be "decided" without saying which way (STYLE T1).
type AssertionOutcome byte

const (
	// AssertionDeclined is the zero value: the evaluator did not decide the
	// {test}, which the pipeline returns as a non-verdict [IsAssertionDeclined]
	// reports true for — never as a rejection and never as a pass.
	AssertionDeclined AssertionOutcome = iota
	// AssertionHolds is a {test} that evaluated to true without raising a
	// dynamic or type error (cvc-assertions-valid).
	AssertionHolds
	// AssertionFails is a {test} that evaluated to false or raised a dynamic or
	// type error, which cvc-assertions-valid treats alike: the value is not
	// facet-valid, so it is not Datatype Valid (cvc-datatype-valid clause 3).
	AssertionFails
)

// AssertionEvaluator decides one {test} of an assertions facet against a value,
// on the terms of Assertions Valid (Datatypes §4.3.13.3, cvc-assertions-valid):
// no context item (clause 1.2), and `$value` bound to the XDM representation of
// v under st (clauses 1.4 and 1.5, dt-xdmrep). st is the type under which
// `$value` is v's XDM representation (dt-xdmrep): the type the facet is
// effective on, or — where that type is a union — the ·active basic member·
// that identified v (dt-xdmrep clause 4), never a union. An implementation is
// never called for a value with no active basic member (cvc-assertions-valid
// clause 1.5): the union dispatch rejects such a literal before the assertions
// stage. b and r are the ones [ValidateLexical] was handed, passed per call and
// stored nowhere, so one evaluator serves every schema.
//
// Its one consumer is the pipeline's assertions stage, which runs after every
// other value facet has accepted v; validate injects package xpath's
// implementation (xpath.FacetAssertions) at every [ValidateLexical],
// [ValidatingType] and [ConstraintMatches] call. An implementation answers
// [AssertionDeclined] for a {test} it cannot evaluate, never a guess.
type AssertionEvaluator interface {
	Evaluate(b Backend, r xsd.TypeResolver, st *xsd.SimpleType, test xsd.XPathExpression, v Value) AssertionOutcome
}

// errAssertionDeclined is the sentinel every assertions-stage DECLINE wraps: an
// [AssertionEvaluator] answered [AssertionDeclined], so the value's Datatype
// Valid verdict is undecided. It is the third non-verdict class beside
// errTypeFault and errFacetPrecondition, and it wraps NEITHER: a decline is not a
// fault of the type or of the backend, it is a {test} this caller's evaluator
// could not decide. assertionDeclined is its one construction site and
// [IsAssertionDeclined] its one test.
var errAssertionDeclined = errors.New("value: an assertions-facet {test} was not evaluated")

// assertionDeclined builds the error for the i-th (0-based) of the n assertions
// in the {value} of the assertions facet effective on st, whose {test} is test,
// that the evaluator declined. It is the ONE construction site of the class
// (STYLE T4): `grep assertionDeclined(` enumerates every decline.
func assertionDeclined(st *xsd.SimpleType, i, n int, test xsd.XPathExpression) error {
	return xsderr.Wrap(ruleCvcAssertionsValid, xsderr.Loc{}, fmt.Errorf(
		"%w: assertion %d of %d in the {value} of the assertions facet of %s, whose {test} is %q: it is undecided whether the value is facet-valid with respect to it, which cvc-assertions-valid requires",
		errAssertionDeclined, i+1, n, simpleTypeLabel(st.Name()), test.Expression()))
}

// simpleTypeLabel names the simple type whose {name} is name for a facet-stage
// message, the assertions stage's and the pattern stage's: "the simple type" and
// its ·expanded name·, or "an anonymous simple type" for the zero QName an inline
// definition carries, which would otherwise render as nothing.
func simpleTypeLabel(name xsd.QName) string {
	if name == (xsd.QName{}) {
		return "an anonymous simple type"
	}
	return fmt.Sprintf("the simple type %s", name)
}

// IsAssertionDeclined reports whether err — an error [ValidateLexical] or
// [ValidatingType] returned — is an assertions-facet DECLINE: the
// [AssertionEvaluator] answered [AssertionDeclined] for a {test} of an
// assertions facet the pipeline reached, so whether the literal is Datatype
// Valid is undecided. [IsDatatypeVerdict] is false for it. The message names
// the assertion, its type and its {test}, and [xsderr.RuleOf] reads
// cvc-assertions-valid off it.
//
// A caller deciding validity declines on it under cvc-assertions-valid, where
// the other non-verdicts decline under the caller's own rule.
func IsAssertionDeclined(err error) bool {
	return errors.Is(err, errAssertionDeclined)
}

// assertionsUndecided is the [AssertionEvaluator] this package's own
// schema-time callers pass — [CheckFacetRestriction] and the [xsd.ValueSpace]
// [NewValueSpace] returns — neither of which is handed an evaluator by its
// caller. It declines every {test}: this package has no XPath engine of its
// own, and answering [AssertionHolds] would accept a value an assertion
// rejects. GAP(value): each of the two therefore leaves undecided what an
// assertions facet in a type's closure would decide; each states its own
// consumers and direction at its marker. (#1042)
type assertionsUndecided struct{}

// Evaluate declines every {test}.
func (assertionsUndecided) Evaluate(Backend, xsd.TypeResolver, *xsd.SimpleType, xsd.XPathExpression, Value) AssertionOutcome {
	return AssertionDeclined
}

// checkAssertions is the assertions stage over v, already accepted by every
// other stage against st: each {test} in the {value} of each assertions facet
// in facets — st's effective ones, which compile collected in effective-facet
// order — is handed to a, in that order, with basic as the type under which
// `$value` is v's XDM representation (dt-xdmrep): st itself, or where st is a
// union the ·active basic member· that identified v (dt-xdmrep clause 4). st
// names the facet in every message.
//
// The first [AssertionFails] is the verdict, a cvc-assertions-valid rejection,
// and it stops the scan. A decline does NOT stop it: every assertion is a
// conjunct of one facet-valid condition, so a later failure decides the value
// invalid whatever an earlier declined {test} would have said. Only where no
// {test} fails is the first decline returned, as assertionDeclined's
// non-verdict.
func checkAssertions(b Backend, r xsd.TypeResolver, st, basic *xsd.SimpleType, v Value, facets []xsd.Facet, a AssertionEvaluator) error {
	var declined error
	for _, f := range facets {
		assertions, _ := f.Assertions()
		for i, as := range assertions {
			outcome := a.Evaluate(b, r, basic, as.Test(), v)
			if outcome == AssertionFails {
				return xsderr.New(ruleCvcAssertionsValid, xsderr.Loc{},
					"the value is not facet-valid with respect to assertion %d of %d in the {value} of the assertions facet of %s, whose {test} %q did not evaluate to true without raising a dynamic or type error, but cvc-assertions-valid requires every {test} to",
					i+1, len(assertions), simpleTypeLabel(st.Name()), as.Test().Expression())
			}
			if outcome == AssertionDeclined && declined == nil {
				declined = assertionDeclined(st, i, len(assertions), as.Test())
			}
		}
	}
	return declined
}
