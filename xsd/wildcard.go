package xsd

import "github.com/kud360/goxsd8/xsderr"

// Wildcard is the Wildcard component (Structures §3.10.1, id="w"): a kind of
// Term with {annotations} (a sequence of Annotation), {namespace constraint}
// (a NamespaceConstraint, §3.10.1 "nc" — see namespaceconstraint.go), and
// {process contents} (one of skip/strict/lax — see closedsets.go). It is the
// thin composition wiring an element/attribute wildcard's admission to the
// §3.10.4 allowance algorithm: AllowsName is the one canonical entry point
// (xsd/doc.go's "wildcard admission ... one canonical implementation").
//
// The zero value is NOT a valid Wildcard (its NamespaceConstraint and
// ProcessContents are both the invalid zero); construct only through
// NewWildcard, which rejects every state Wildcard Properties Correct
// (§3.10.6.1, w-props-correct) clause 1 forbids, so an ill-formed record is
// unrepresentable (STYLE T1). Wildcard is immutable after construction.
//
// {process contents} controls what the VALIDATOR does with an item AFTER
// AllowsName admits it (skip/lax/strict, §3.10.1); it plays no role in
// admission itself — cvc-wildcard (§3.10.4.1) does not test it. Do not fold
// ProcessContents into AllowsName.
type Wildcard struct {
	namespaceConstraint NamespaceConstraint
	processContents     ProcessContents
	annotations         []Annotation
}

// NewWildcard builds a Wildcard, rejecting the states Wildcard Properties
// Correct (§3.10.6.1, w-props-correct) clause 1 forbids for the Wildcard's
// own two scalar properties:
//
//   - processContents not one of ProcessSkip/ProcessStrict/ProcessLax;
//   - namespaceConstraint not validly constructed (its {variety} is the
//     invalid zero NamespaceConstraintVariety) — this catches a caller
//     passing the zero NamespaceConstraint{} instead of a value built
//     through NewNamespaceConstraint, which would otherwise make an illegal
//     Wildcard representable.
//
// w-props-correct clauses 2-4 are the NamespaceConstraint sub-record's own
// invariants, already enforced by NewNamespaceConstraint; clause 5
// (attribute wildcards must not carry the sibling keyword) is vacuously
// satisfied because NamespaceConstraint does not represent the
// defined/sibling keywords at all (see the GAP marker on
// NewNamespaceConstraint).
//
// annotations is copied; the caller's backing array is not aliased.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built wildcard — may
// legitimately pass the zero xsderr.Loc{}.
func NewWildcard(loc xsderr.Loc, namespaceConstraint NamespaceConstraint, processContents ProcessContents, annotations []Annotation) (Wildcard, error) {
	switch processContents {
	case ProcessSkip, ProcessStrict, ProcessLax:
	default:
		return Wildcard{}, xsderr.New(ruleWildcardCorrect, loc,
			"wildcard {process contents} %s is not one of skip/strict/lax (w-props-correct clause 1)", processContents)
	}
	switch namespaceConstraint.Variety() {
	case NamespaceConstraintAny, NamespaceConstraintEnumeration, NamespaceConstraintNot:
	default:
		return Wildcard{}, xsderr.New(ruleWildcardCorrect, loc,
			"wildcard {namespace constraint} is not a validly constructed NamespaceConstraint (w-props-correct clause 1)")
	}
	w := Wildcard{namespaceConstraint: namespaceConstraint, processContents: processContents}
	if len(annotations) > 0 {
		w.annotations = append([]Annotation(nil), annotations...)
	}
	return w, nil
}

// term marks Wildcard as a Term (§3.10.1: "a kind of Term"); see term.go.
func (Wildcard) term() {}

// ProcessContents returns the {process contents} property.
func (w Wildcard) ProcessContents() ProcessContents {
	return w.processContents
}

// Annotations returns the {annotations} property in document order. It
// returns a copy: mutating the result does not affect w. An empty
// {annotations} yields nil.
func (w Wildcard) Annotations() []Annotation {
	if len(w.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), w.annotations...)
}

// AllowsName reports whether the expanded name is admitted by w's {namespace
// constraint}, delegating to NamespaceConstraint.AllowsName — this is the ONE
// canonical wildcard-admission entry point (xsd/doc.go); callers must never
// re-derive the allowance algorithm by reaching past this method.
//
// This implements cvc-wildcard (§3.10.4.1) clause 1 ONLY (expanded name
// valid per cvc-wildcard-name, §3.10.4.2). Clauses 2-3 (the defined/sibling
// keyword exclusions against the live declaration graph) are OUT of scope
// for this pure-leaf package — see the identical GAP(xsd) marker on
// NamespaceConstraint.AllowsName.
//
// {process contents} plays no part in this decision: even a skip wildcard
// admits everything {namespace constraint} admits; ProcessContents tells the
// validator what to do with an admitted item, not whether to admit it.
func (w Wildcard) AllowsName(name QName) bool {
	return w.namespaceConstraint.AllowsName(name)
}
