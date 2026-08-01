package xsd

// ValueSpace is the lexical→value mapping plus the two value-space comparisons
// this package needs but, as a pure leaf (PRINCIPLES 1, doc.go), cannot
// implement: they require package value, which is layered ABOVE xsd. A
// ValueConstraint carries {lexical form} only, never {value} (valueconstraint.go
// says why), so every "is this {value} the same as that one?" question the §3.5.6
// and §3.4.6.4 constraints ask is answered here or not at all.
//
// Both methods take BOTH sides' governing Simple Type Definition because the two
// {value}s compared need not belong to one type: loc-testSubP (§3.4.6.4) clauses
// 4.2 and 5.2.2 compare a general declaration's {value} against a specific
// (restricting) one's, and only the implementation can decide whether the two
// types share a value space.
//
// FAIL-OPEN CONTRACT, binding on every implementation: decided=false means "this
// comparison was not made" and is the answer for an ungoverned type, a lexical
// the governing mapping cannot map, and two incommensurable value spaces (ta and
// tb resolving to different governing mappings). A caller treats undecided as
// "the clause is not competent to charge a failure" and accepts; an
// implementation must never use undecided as licence to reject, and must never
// report a false NOT-same, because every caller turns a decided not-same into a
// schema rejection (PRINCIPLES 9's direction, applied to value spaces).
//
// Install one with [SchemaBuilder.FinalizeWith]. A Schema finalized through plain
// [SchemaBuilder.Finalize] has no value space and every comparison is undecided,
// which is exactly the pre-existing fail-open behavior.
type ValueSpace interface {
	// Identical is the identity relation (Datatypes §2.2.1) — what
	// au-props-correct (§3.5.6) clause 3 compares two {value}s under. It is
	// NOT the equal-or-identical union: clause 3 says "identical", and for the
	// datatypes whose equality differs from their identity (float/double ±0,
	// dateTime across timezone offsets, §2.2.2) the two verdicts differ.
	Identical(ta *SimpleType, a ValueConstraint, tb *SimpleType, b ValueConstraint) (identical, decided bool)

	// EqualOrIdentical is the equal-or-identical union (Datatypes §2.2.1/§2.2.2,
	// "All comparisons for 'sameness' prescribed by this specification test for
	// either equality or identity, not for identity alone") — what loc-testSubP
	// (§3.4.6.4) clauses 4.2 and 5.2.2 compare two {value}s under.
	EqualOrIdentical(ta *SimpleType, a ValueConstraint, tb *SimpleType, b ValueConstraint) (same, decided bool)
}

// undecidedValueSpace is the ValueSpace a Schema finalized without one carries:
// every comparison is undecided, so each consumer takes its documented fail-open
// branch. It exists so no consumer needs a nil check — one code path, not two
// (STYLE S1) — and stays unexported: "no value space installed" is not a
// capability a caller supplies, it is the absence of one, reached only by not
// calling FinalizeWith (STYLE T5).
type undecidedValueSpace struct{}

func (undecidedValueSpace) Identical(*SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	return false, false
}

func (undecidedValueSpace) EqualOrIdentical(*SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	return false, false
}

var _ ValueSpace = undecidedValueSpace{}
