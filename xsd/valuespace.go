package xsd

// ValueSpace is the lexical→value bridge this package needs but, as a pure leaf
// (PRINCIPLES 1, doc.go), cannot implement: it requires package value, which is
// layered ABOVE xsd. A ValueConstraint carries {lexical form} and the namespace
// context that lexical needs, never {value} (valueconstraint.go says why), so
// every question the §3.2.6.2, §3.5.6 and §3.4.6.4 constraints ask ABOUT a
// {value} — "is this one the same as that one?", "does this {lexical form} denote
// one at all?" — is answered here or not at all.
//
// Two shapes of question sit behind the one interface. They share it because they
// share an installation seam ([SchemaBuilder.FinalizeWith]) and one capability
// bundle: a value space that can compare but not validate — or the reverse — is
// not a state worth making representable (STYLE T1/T2).
//
//   - the COMPARISONS, Identical and EqualOrIdentical. Each takes BOTH sides'
//     governing Simple Type Definition because the two {value}s compared need not
//     belong to one type: loc-testSubP (§3.4.6.4) clauses 4.2 and 5.2.2 compare a
//     general declaration's {value} against a specific (restricting) one's, and
//     only the implementation can decide whether the two types share a value
//     space.
//   - the one-sided VALIDITY question, ValidDefault, which asks whether a single
//     {lexical form} denotes a value of a single type at all (§3.2.6.2). It takes
//     one type, and its fail-open scope differs from the comparisons' in
//     consequence: see its own doc.
//
// FAIL-OPEN CONTRACT, binding on every implementation: decided=false means "this
// question was not answered". It is the answer for an ungoverned type, a lexical
// the governing mapping cannot map, two incommensurable value spaces (ta and tb
// resolving to different governing mappings), and a QName or NOTATION literal
// whose prefix its ValueConstraint's captured context does not bind. A caller
// treats undecided as "the clause is not competent to charge a failure" and
// accepts; an implementation must never use undecided as licence to reject, must
// never report a false NOT-same, and must never report a false NOT-valid: every
// reader turns a decided negative into a refusal, but not always the same kind of
// refusal. checkSimpleDefault (valueconstraintvalid.go) reads ValidDefault for
// a-props-correct cl.2, au-props-correct cl.2 and e-props-correct;
// checkAttributeUseValueConstraint (valueconstraintvalid.go) reads Identical for
// au-props-correct cl.3; and defaultbinding.go reads EqualOrIdentical for
// loc-testSubP cl.4.3 and cl.5.2.2 — all four charge a SCHEMA rejection. But
// validate/cvcattribute.go's defaultedAttribute also reads ValidDefault, for
// cvc-complex-type cl.4, and charges an INSTANCE violation during document
// assessment instead. Either way a decided negative is refused, never let
// through (PRINCIPLES 20's direction, applied to value spaces).
//
// That contract is the deliberate OPPOSITE of the other capability installed at
// the same seam, [SimpleTypeRestrictionChecker], and the two must not be
// implemented against each other's. That one is REJECT-CAPABLE: it answers with
// an error, and a non-nil answer is a real *xsderr.Error carrying a per-facet
// rule ID that finalize returns as the schema's rejection. This one can never
// produce a rejection of its own — it hands back a verdict a caller may charge —
// and its undecided answer is an instruction to accept. Two capabilities rather
// than one bundle precisely so that a reject-capable method is not representable
// inside a fail-open interface (STYLE T1).
//
// EVERY METHOD TAKES A [TypeResolver], AND NO IMPLEMENTATION MAY STORE ONE. A
// Simple Type Definition's {base type definition} may be a deferred reference by
// name (simpletyperef.go), and answering any question below means walking that
// chain — {variety}, the governing mapping, the facets in force are all read
// through it. The resolver is therefore a per-call parameter, exactly as it is
// on [SimpleTypeRestrictionChecker.CheckRestriction]: an implementation is built
// once and installed into whichever schema finalize hands it, so a stored
// resolver would silently tie one implementation to one schema.
//
// Install one with [SchemaBuilder.FinalizeWith]. A Schema finalized through plain
// [SchemaBuilder.Finalize] has no value space and every question is undecided,
// which is exactly the pre-existing fail-open behavior.
type ValueSpace interface {
	// Identical is the identity relation (Datatypes §2.2.1) — what
	// au-props-correct (§3.5.6) clause 3 compares two {value}s under. It is
	// NOT the equal-or-identical union: clause 3 says "identical", and for the
	// datatypes whose equality differs from their identity (float/double ±0,
	// dateTime across timezone offsets, §2.2.2) the two verdicts differ.
	Identical(r TypeResolver, ta *SimpleType, a ValueConstraint, tb *SimpleType, b ValueConstraint) (identical, decided bool)

	// EqualOrIdentical is the equal-or-identical union (Datatypes §2.2.1/§2.2.2,
	// "All comparisons for 'sameness' prescribed by this specification test for
	// either equality or identity, not for identity alone") — what loc-testSubP
	// (§3.4.6.4) clauses 4.2 and 5.2.2 compare two {value}s under.
	EqualOrIdentical(r TypeResolver, ta *SimpleType, a ValueConstraint, tb *SimpleType, b ValueConstraint) (same, decided bool)

	// ValidDefault is Simple Default Valid (§3.2.6.2, cos-valid-simple-default)
	// — what a-props-correct (§3.2.6.1) clause 2 and au-props-correct (§3.5.6)
	// clause 2 both charge: vc.{lexical form} must be ·valid· with respect to t
	// as Datatype Valid defines it (Datatypes §4.1.4, cvc-datatype-valid — the
	// lexical facets, the lexical→value mapping and the value facets in one
	// verdict), and it must map to vc.{value}.
	//
	// That second clause needs no separate test here. A ValueConstraint carries
	// no independently stored {value} (valueconstraint.go): {value} is always
	// DERIVED from {lexical form} through this very mapping, so "maps to
	// vc.{value}" collapses to "maps to a value", which the mapping stage of the
	// first clause already decides.
	//
	// Unlike the comparisons this takes ONE type, because the constraint relates
	// a lexical form to the single type that constrains it. An implementation
	// therefore needs no SHARED mapping across two types and must decide the
	// list and union varieties (Datatype Valid clauses 2.2 and 2.3 define both)
	// rather than refusing them the way Identical/EqualOrIdentical do.
	//
	// decided=false — accept, never reject — is required for at least:
	//
	//   - an UNGOVERNED type, xs:anySimpleType and xs:anyAtomicType included.
	//     Those two are the ·special· datatypes (Datatypes §4.1), for which
	//     Datatype Valid is unconditionally true, and they are exactly what
	//     §3.2.2.2's third tier gives every attribute declaration with no @type
	//     — so answering "not valid" merely because no implementation maps them
	//     would false-reject every typeless attribute default there is.
	//   - a QName- or NOTATION-governed value space ANYWHERE in t's {item type
	//     definition}/{member type definitions} closure, unless the
	//     implementation routes the namespace context vc captured
	//     (§3.3.18/§3.3.19, PRINCIPLES 19) all the way to the literal's own
	//     mapping — through the list and union dispatch included. Resolving a
	//     prefix without that context is a guess, not a verdict.
	//   - a construction-stage failure in T'S OWN facets: a pattern the
	//     implementation cannot compile, an enumeration or bound facet whose
	//     declaring type it cannot map. That is a statement about the TYPE, not
	//     a verdict about vc.{lexical form}, and charging it as clause 2 would
	//     blame the wrong component under the wrong rule ID.
	//   - a facet on t that is NOT APPLICABLE to t (cos-applicable-facets,
	//     Datatypes §4.1.5), or a t carrying no usable whiteSpace mode where
	//     §3.16.7.4 and §4.3.6.1 guarantee one. Both are reachable only for a
	//     *SimpleType assembled by calling this package's constructors directly,
	//     since the construction seam every parsed type goes through discharges
	//     applicability. Each is a fault of T, not a verdict about vc.{lexical
	//     form}, so an implementation that meets one answers undecided and never
	//     decided-invalid. [SchemaBuilder.FinalizeWith] and the clause-2 check
	//     itself document that to their own callers as a guarantee, which is why
	//     it binds every implementation here rather than describing one.
	//
	// Only a genuine verdict-stage failure — the lexical form itself failing a
	// facet or the mapping — may answer decided with a non-nil cause, and cause
	// is that verdict itself, passed out unmodified: cvc-datatype-valid
	// (Datatypes §4.1.4) for the mapping, one of the facet rules under it
	// otherwise. No implementation re-tags it, and a caller charging under its
	// own rule ID wraps it rather than replacing it.
	//
	// cause is non-nil IF AND ONLY IF decided and vc.{lexical form} is invalid:
	// an undecided answer is (nil, false) and a valid one is (nil, true). The
	// cause's presence IS the verdict, so neither valid-with-a-cause nor
	// invalid-without-one is representable (STYLE D3).
	ValidDefault(r TypeResolver, t *SimpleType, vc ValueConstraint) (cause error, decided bool)
}

// undecidedValueSpace is the ValueSpace a Schema finalized without one carries:
// every question is undecided, so each consumer takes its documented fail-open
// branch. It exists so no consumer needs a nil check — one code path, not two
// (STYLE S1) — and stays unexported: "no value space installed" is not a
// capability a caller supplies, it is the absence of one, reached only by not
// calling FinalizeWith (STYLE T5).
type undecidedValueSpace struct{}

func (undecidedValueSpace) Identical(TypeResolver, *SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	return false, false
}

func (undecidedValueSpace) EqualOrIdentical(TypeResolver, *SimpleType, ValueConstraint, *SimpleType, ValueConstraint) (bool, bool) {
	return false, false
}

func (undecidedValueSpace) ValidDefault(TypeResolver, *SimpleType, ValueConstraint) (error, bool) {
	return nil, false
}

var _ ValueSpace = undecidedValueSpace{}
