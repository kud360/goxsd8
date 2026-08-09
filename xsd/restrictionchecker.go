package xsd

// SimpleTypeRestrictionChecker is the facet-VALUE half of Derivation Valid
// (Restriction, Simple) (Structures §3.16.6.2, cos-st-restricts) as a finalize
// capability: the sub-clauses this package, a pure leaf (PRINCIPLES 1, doc.go),
// cannot decide about a Simple Type Definition it has already constructed.
//
// Two families sit behind the one method, and they share it because they share
// one installation seam ([SchemaBuilder.FinalizeWith]) and one subject — an
// already-constructed *[SimpleType]:
//
//   - clause 1.3.1 for an ATOMIC type: every facet in its {facets} must be
//     applicable to its {primitive type definition} (cos-applicable-facets,
//     Datatypes §4.1.5). The applicable set is per-primitive and comes from a
//     GENERATED table (PRINCIPLES 26), which lives above this leaf.
//   - the value-space constraints of clauses 1.3.2 / 2.2.2.5 / 3.2.2.5 — the
//     four bound facets and enumeration — which need a lexical→value mapping,
//     also above this leaf.
//
// The list and union applicability counterparts, clauses 2.2.2.4 and 3.2.2.4,
// are NOT this interface's: their applicable sets are fixed literals keyed off
// {variety} alone, so checkVarietyApplicableFacets (derivation.go) charges them
// here with no table at all.
//
// REJECT-CAPABLE CONTRACT, binding on every implementation, and it is the
// deliberate OPPOSITE of [ValueSpace]'s fail-open one. A ValueSpace answers
// (verdict, decided) and its undecided answer is an instruction to ACCEPT — "an
// implementation must never use undecided as licence to reject" (valuespace.go).
// This interface has no undecided state to abuse: it answers with an error, a
// nil error means the clauses above HOLD for t, and a non-nil one is a real
// *[github.com/kud360/goxsd8/xsderr.Error] carrying the specific per-facet rule
// ID — cos-st-restricts for applicability, the per-facet Schema Component
// Constraint (minInclusive valid restriction §4.3.10.4, enumeration valid
// restriction §4.3.5.5, …) for the value-space clauses — which finalize returns
// verbatim as the schema's rejection. Nobody should implement either interface
// against the other's contract; the two agree on one point only, that a fault
// the implementation cannot judge is never a rejection, and here that is spelled
// as a nil error rather than as a third answer.
//
// Install one with [SchemaBuilder.FinalizeWith]. A Schema finalized through
// plain [SchemaBuilder.Finalize] carries no checker and these clauses go
// unchecked, which is exactly the guarantee a programmatically assembled schema
// has always had (see undecidedRestrictionChecker).
type SimpleTypeRestrictionChecker interface {
	// CheckRestriction charges the clauses above against one already-constructed
	// Simple Type Definition. r resolves t's {base type definition} chain, which
	// may be deferred by name (simpletyperef.go) — every reader an
	// implementation needs (Variety, Primitive, EffectiveFacets, Base) takes one.
	// It is a PARAMETER and must be stored nowhere: an implementation is built
	// once, by [NewValueSpace]'s sibling constructor, and serves whichever schema
	// finalize hands it, so a stored resolver would tie it to one.
	CheckRestriction(r TypeResolver, t *SimpleType) error
}

// undecidedRestrictionChecker is the SimpleTypeRestrictionChecker a Schema
// finalized without one carries: every type passes, so checkSimpleTypeDerivations
// still walks every slot of the inventory and charges nothing. It exists so no
// consumer needs a nil check — one code path, not two (STYLE S1) — and stays
// unexported: "no checker installed" is not a capability a caller supplies, it
// is the absence of one, reached only by not calling FinalizeWith (STYLE T5).
//
// ITS NIL RETURN MEANS "NO CHECKER INSTALLED", NOT "UNDECIDED" in
// undecidedValueSpace's sense (valuespace.go). A reject-capable interface has no
// undecided state, so there is no third answer for this type to give; it is
// named for the role it plays at the shared seam, not for a shared contract. The
// two types look alike and their contracts are not alike — see
// [SimpleTypeRestrictionChecker].
type undecidedRestrictionChecker struct{}

func (undecidedRestrictionChecker) CheckRestriction(TypeResolver, *SimpleType) error { return nil }

var _ SimpleTypeRestrictionChecker = undecidedRestrictionChecker{}
