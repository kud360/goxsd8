package builtin

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCosSTRestricts is Derivation Valid (Restriction, Simple) (Structures
// §3.16.6.2, id="cos-st-restricts"). This package charges exactly one of its
// sub-clauses — 1.3.1, "DF is applicable to D, as specified in [Applicable
// Facets] of [XML Schema: Datatypes]" — for the ATOMIC case, because that is the
// clause whose answer comes from the generated per-primitive applicability table
// (Types/TypeSpec.Applies). cos-applicable-facets (§4.1.5) is cited in the
// message rather than as the Rule: it DEFINES the applicable sets, while
// cos-st-restricts is the constraint that makes a non-applicable facet a
// rejection.
const ruleCosSTRestricts xsderr.Rule = "cos-st-restricts"

// NewRestrictionChecker returns the [xsd.SimpleTypeRestrictionChecker] a caller
// installs at [xsd.SchemaBuilder.FinalizeWith], so the finalize pass
// checkSimpleTypeDerivations can charge the facet-VALUE half of cos-st-restricts
// against every simple type a schema reaches — the anonymous inline ones no index
// holds included.
//
// It is the sole implementation of that interface, and it lives in this package
// because this package alone holds both edges the charge needs: the generated
// per-primitive applicability table (Types/TypeSpec.Applies) and an edge to
// package value for the value-space clauses. Package xsd, a pure leaf, may depend
// on neither (PRINCIPLES 1), which is exactly why the capability is taken as an
// input there rather than called directly.
//
// It charges cos-st-restricts clause 1.3.1 for an ATOMIC type — every facet in
// its {facets} must be applicable to its {primitive type definition}
// (cos-applicable-facets, Datatypes §4.1.5), answered against the generated
// table — and then delegates the value-space constraints of clauses 1.3.2 /
// 2.2.2.5 / 3.2.2.5 to [value.CheckFacetRestriction], for which b supplies the
// value space. Applicability runs FIRST so the delegate may assume a bound
// facet's value space really is ordered. See [xsd.SimpleTypeRestrictionChecker]
// for the reject-capable contract this satisfies.
func NewRestrictionChecker(b value.Backend) xsd.SimpleTypeRestrictionChecker {
	return restrictionChecker{backend: b}
}

// restrictionChecker is NewRestrictionChecker's return value: a value type
// carrying only the backend, since the check is a pure function of (backend,
// type) and holds no state between calls.
type restrictionChecker struct{ backend value.Backend }

// CheckRestriction implements [xsd.SimpleTypeRestrictionChecker] over
// checkSimpleTypeRestriction.
func (c restrictionChecker) CheckRestriction(t *xsd.SimpleType) error {
	return checkSimpleTypeRestriction(c.backend, t)
}

// checkSimpleTypeRestriction charges the facet-VALUE half of Derivation Valid
// (Restriction, Simple) (cos-st-restricts, Structures §3.16.6.2) on an
// already-constructed Simple Type Definition: the sub-clauses package xsd cannot
// reach from a pure leaf with no applicability table and no value spaces.
//
// It charges, in order:
//
//   - clause 1.3.1 for an ATOMIC t: every facet in t's {facets} must be
//     applicable to t's {primitive type definition} per cos-applicable-facets
//     (§4.1.5), answered against the generated table through TypeSpec.Applies —
//     never a second, hand-typed copy of that table (PRINCIPLES 26). The list and
//     union counterparts, clauses 2.2.2.4 and 3.2.2.4, are charged inside package
//     xsd instead: their applicable sets are fixed literals keyed off {variety}
//     alone, needing no table.
//   - the value-space constraints of clause 1.3.2 / 2.2.2.5 / 3.2.2.5 — the four
//     bound facets and enumeration — by delegating to value.CheckFacetRestriction.
//     Applicability runs FIRST so that delegate can assume a bound facet's value
//     space really is ordered.
//
// The count- and token-valued constraints of those same clauses (length,
// minLength, maxLength, totalDigits, fractionDigits, whiteSpace,
// explicitTimezone, and the same-type consistency SCCs) are already charged
// inside xsd.NewSimpleType, so t reaching this function has passed them.
//
// This is a SEPARATE, post-construction charge rather than a hook inside
// xsd.NewSimpleType because package xsd must not depend on this package or on
// package value (PRINCIPLES 1). It is unexported because the ONE way to reach it
// from outside is [NewRestrictionChecker]: a schema finalized without that
// capability installed keeps the old, weaker guarantee — value/facets.go's
// documented preconditions about facet applicability then remain the caller's to
// honor — and a second, direct entry point would be an exported identifier with
// no consumer (STYLE T5) growing callers against the function the capability seam
// is meant to own.
func checkSimpleTypeRestriction(b value.Backend, t *xsd.SimpleType) error {
	if err := checkAtomicApplicableFacets(t); err != nil {
		return err
	}
	return value.CheckFacetRestriction(b, t)
}

// checkAtomicApplicableFacets charges cos-st-restricts clause 1.3.1 for an
// atomic t. It walks t's {facets} in the document order EffectiveFacets yields
// (STYLE D2), so which violation is reported first is deterministic.
//
// A non-atomic {variety} returns nil: list and union applicability is
// cos-st-restricts clause 2.2.2.4 / 3.2.2.4, charged inside package xsd off the
// fixed literal sets cos-applicable-facets gives for those varieties.
//
// An ABSENT {primitive type definition} on an atomic type — only
// xs:anyAtomicType, §3.16.1 — makes NO facet applicable ("If {variety} is
// atomic, and {primitive type definition} is absent then no facets are
// applicable", §4.1.5), so any facet at all is rejected. A primitive that has no
// row in the generated table is not reachable today (every primitive component
// this module builds comes from Seed, hence from Types) and is skipped rather
// than rejected: the §4.1.5 footnote makes the applicable set of an
// implementation-defined primitive implementation-defined, so there is nothing
// to decide against.
func checkAtomicApplicableFacets(t *xsd.SimpleType) error {
	if _, ok := t.Variety().(xsd.Atomic); !ok {
		return nil
	}
	primitive := t.Primitive()
	if primitive == nil {
		return rejectAnyFacet(t, "an atomic simple type definition with an absent {primitive type definition}")
	}
	spec, ok := typeSpecOf(primitive.Name())
	if !ok {
		return nil
	}
	for _, ef := range t.EffectiveFacets() {
		kind := ef.Facet().Kind()
		name, known := facetName(kind)
		if known && spec.Applies(name) {
			continue
		}
		return xsderr.New(ruleCosSTRestricts, t.Loc(),
			"simple type {facets} carries %s, which is not applicable to the {primitive type definition} xs:%s (cos-st-restricts clause 1.3.1 via cos-applicable-facets §4.1.5)",
			kind, spec.Name)
	}
	return nil
}

// rejectAnyFacet rejects the first facet in t's {facets}, for the §4.1.5 case
// where the applicable set is empty. subject names that case in the message.
func rejectAnyFacet(t *xsd.SimpleType, subject string) error {
	eff := t.EffectiveFacets()
	if len(eff) == 0 {
		return nil
	}
	return xsderr.New(ruleCosSTRestricts, t.Loc(),
		"simple type {facets} carries %s, but no facet is applicable to %s (cos-st-restricts clause 1.3.1 via cos-applicable-facets §4.1.5)",
		eff[0].Facet().Kind(), subject)
}

// typeSpecOf returns the generated row for the builtin datatype named by qn, and
// whether qn names a builtin at all. It scans Types on demand rather than
// building an index: applicability is answered per simple type at schema
// construction, not per validated instance, so there is no measured hot path to
// justify a memoized index (STYLE D3).
func typeSpecOf(qn xsd.QName) (TypeSpec, bool) {
	if qn.Space != xsd.XMLSchemaNS {
		return TypeSpec{}, false
	}
	for i := range Types {
		if Types[i].Name == qn.Local {
			return Types[i], true
		}
	}
	return TypeSpec{}, false
}

// facetNames is THE bridge between the two spellings of one closed set: the
// FacetName strings the generated TypeSpec table is keyed by (§4.3, verbatim spec
// names) and the xsd.FacetKind enum the component model carries. Both lookup
// directions — facetName and seed.go's facetKind — scan this ONE ordered slice,
// so the bijection cannot drift the way two hand-typed inverse switches did (the
// old facetKind silently omitted maxScale/minScale, making the pair disagree on
// two of sixteen entries).
//
// It is a slice scanned linearly, never a map: sixteen entries answered at schema
// construction is no measured hot path (STYLE D3), and a slice keeps the ordering
// deterministic (STYLE D2). It is deliberately NOT a FacetKind.String()
// round-trip: String is a diagnostic rendering with a "FacetKind(n)" fallback for
// an invalid value, so routing table lookups through it would silently turn a
// future unmapped kind into a lookup miss instead of a visible gap in this table.
var facetNames = []struct {
	name FacetName
	kind xsd.FacetKind
}{
	{"length", xsd.FacetLength},
	{"minLength", xsd.FacetMinLength},
	{"maxLength", xsd.FacetMaxLength},
	{"pattern", xsd.FacetPattern},
	{"enumeration", xsd.FacetEnumeration},
	{"whiteSpace", xsd.FacetWhiteSpace},
	{"maxInclusive", xsd.FacetMaxInclusive},
	{"maxExclusive", xsd.FacetMaxExclusive},
	{"minExclusive", xsd.FacetMinExclusive},
	{"minInclusive", xsd.FacetMinInclusive},
	{"totalDigits", xsd.FacetTotalDigits},
	{"fractionDigits", xsd.FacetFractionDigits},
	{"assertions", xsd.FacetAssertions},
	{"explicitTimezone", xsd.FacetExplicitTimezone},
	{"maxScale", xsd.FacetMaxScale},
	{"minScale", xsd.FacetMinScale},
}

// facetName maps an xsd.FacetKind to the FacetName the generated table is keyed
// by, scanning facetNames. ok is false only for a FacetKind outside the closed
// set, which xsd.NewSimpleType already rejects (st-props-correct clause 5); a
// caller treats it as "not applicable".
func facetName(kind xsd.FacetKind) (FacetName, bool) {
	for _, e := range facetNames {
		if e.kind == kind {
			return e.name, true
		}
	}
	return "", false
}
