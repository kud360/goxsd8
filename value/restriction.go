package value

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file holds the SCHEMA-CONSTRUCTION half of the facet machinery: the
// §4.3 Schema Component Constraints whose operands live in a VALUE SPACE and so
// need a Backend, which is exactly the part of cos-st-restricts clause 1.3.2 /
// 2.2.2.5 / 3.2.2.5 that package xsd — a pure leaf that cannot import this
// package — has to leave undone (xsd/derivation.go's checkFacetRestrictions
// charges the count- and token-valued rules there).
//
// It is the construction-time complement of facets.go: the same four bound
// facets and the same enumeration facet, but comparing a RESTRICTION's facet
// {value}s against its {base type definition} ONCE, when the type is built,
// rather than comparing an instance's value against those facets on every
// validated literal.
//
// cos-pattern-restriction (§4.3.4.5, "it is an error if there is any member of
// the {value} of the pattern facet on the {base type definition} which is not
// also a member of the {value}") is deliberately NOT checked here: it is a
// structural invariant of xsd.SimpleType, not a runtime condition.
// SimpleType.EffectiveFacets computes {facets} by walking the WHOLE base chain
// and overlaying each level's own facets, and xsd's overlayFacet gives
// FacetPattern keep-both semantics (§4.3.4.2 xr-pattern: patterns at different
// derivation steps are ANDed, so the base's pattern facet survives as its own
// entry beside the derived one) — the base's pattern facet is retained verbatim
// regardless of what a caller supplies as own facets, so every member of its
// {value} is trivially still a member of {facets}. There is no reachable
// violating state to reject.

// The construction-time Schema Component Constraints this file charges — the
// §4.3 "valid restriction" siblings of facets.go's instance-time cvc-* rules.
// Each string is a live entry in xsderr's generated catalog.
const (
	// ruleEnumerationValidRestriction is enumeration valid restriction (§4.3.5.5,
	// id="enumeration-valid-restriction"): every member of a restriction's
	// enumeration facet {value} must be in the ·value space· of the {base type
	// definition}.
	ruleEnumerationValidRestriction xsderr.Rule = "enumeration-valid-restriction"
	// ruleMaxInclusiveValidRestriction is maxInclusive valid restriction
	// (§4.3.7.4, id="maxInclusive-valid-restriction").
	ruleMaxInclusiveValidRestriction xsderr.Rule = "maxInclusive-valid-restriction"
	// ruleMaxExclusiveValidRestriction is maxExclusive valid restriction
	// (§4.3.8.4, id="maxExclusive-valid-restriction").
	ruleMaxExclusiveValidRestriction xsderr.Rule = "maxExclusive-valid-restriction"
	// ruleMinExclusiveValidRestriction is minExclusive valid restriction
	// (§4.3.9.4, id="minExclusive-valid-restriction").
	ruleMinExclusiveValidRestriction xsderr.Rule = "minExclusive-valid-restriction"
	// ruleMinInclusiveValidRestriction is minInclusive valid restriction
	// (§4.3.10.4, id="minInclusive-valid-restriction").
	ruleMinInclusiveValidRestriction xsderr.Rule = "minInclusive-valid-restriction"
)

// CheckFacetRestriction charges the value-space Schema Component Constraints
// relating a Simple Type Definition's own Constraining Facets to its {base type
// definition} — the half of cos-st-restricts clause 1.3.2 / 2.2.2.5 / 3.2.2.5
// that needs a lexical→value mapping:
//
//   - maxInclusive / maxExclusive / minInclusive / minExclusive valid restriction
//     (§4.3.7.4–§4.3.10.4). Each is a FOUR-WAY cross-check: a derived bound is
//     compared against every bound facet in the base's {facets}, not only the
//     homonymous one, because a derived minInclusive can undercut the base's
//     minExclusive without ever touching the base's minInclusive.
//   - enumeration valid restriction (§4.3.5.5): every member of a derived
//     enumeration facet's {value} must be in the ·value space· of the {base type
//     definition}.
//
// It is the value-aware second half of a two-part seam whose entry point is
// builtin.CheckSimpleTypeRestriction; that function charges the atomic
// applicable-facet clause 1.3.1 first and then delegates here. A caller that
// builds an xsd.SimpleType through the xsd constructors directly, bypassing that
// entry point, gets neither check.
//
// b supplies the value space. When b has NO governing mapping for the base type,
// every comparison below is skipped and CheckFacetRestriction returns nil: an
// unmapped type is a gap in the BACKEND, not an invalidity in the schema, and
// turning it into a rejection would false-reject every schema written against a
// datatype the caller chose not to back. The same gap is still reported — as a
// real cvc-datatype-valid error, not silently — the moment an instance is
// validated against such a type (ValidateLexical).
//
// t's OWN facets are the derived side of every comparison, deliberately: a facet
// t merely INHERITS is the very same facet component the base carries, so
// comparing it against the base's {facets} is vacuous, and it was already charged
// against the base's own base when the base was constructed. This mirrors
// xsd/derivation.go's checkScaleValueRestriction.
func CheckFacetRestriction(b Backend, t *xsd.SimpleType) error {
	base := t.Base()
	if base == nil {
		// t IS xs:anySimpleType: no {base type definition} to restrict, and it
		// carries no facets of its own (§3.16.1).
		return nil
	}
	m, ok := governingMapping(b, base)
	if !ok {
		return nil
	}
	rc := restrictionCheck{mapping: m, whiteSpace: whiteSpaceInForce(base), owner: t}
	if err := rc.checkBoundRestrictions(); err != nil {
		return err
	}
	return rc.checkEnumerationRestriction(b)
}

// restrictionCheck is the resolved context one CheckFacetRestriction pass needs:
// the type under construction, the base's governing mapping, and the base's
// in-force whiteSpace mode. Resolving all three once and carrying them together
// keeps every BOUND facet {value} in this pass parsed the same way, which is what
// makes the resulting values comparable at all. (The enumeration check needs no
// mapping of its own: membership in the base's value space is decided by running
// each member through the base type definition itself, checkEnumerationRestriction.)
//
// The {base type definition} itself is deliberately NOT a field: it is exactly
// owner.Base(), so carrying it would be a second encoding of one fact (STYLE D3).
type restrictionCheck struct {
	mapping Mapping
	// whiteSpace is the mode in force on owner.Base() — the type whose value
	// space every facet {value} compared in this pass must be a member of — or the
	// zero mode when none is (a union {variety}, or a base carrying no usable
	// whiteSpace facet). It is resolved by whiteSpaceInForce and APPLIED by
	// facetValue, the same pair that parses facet {value}s at instance-pipeline
	// construction; this pass adds no normalization of its own.
	whiteSpace whiteSpace
	owner      *xsd.SimpleType
}

// checkBoundRestrictions charges the four bound-facet valid-restriction SCCs
// (§4.3.7.4–§4.3.10.4). BOTH operands of every comparison are parsed by ONE
// mapping — the BASE type's governing mapping — so the two values are members of
// the same space and their Cmp is meaningful; parsing the derived side in the
// derived type's own (possibly narrower) representation and the base side in the
// base's would compare across representations, which the widest-space rule
// (st-restrict-facets §3.16.6.4) exists to prevent.
//
// Both loops walk their operands in document order — t's own facets as declared,
// the base's {facets} as EffectiveFacets yields them — so which violation is
// reported first is deterministic (STYLE D2).
func (rc restrictionCheck) checkBoundRestrictions() error {
	for _, own := range rc.owner.OwnFacets() {
		kind := own.Kind()
		if !isBoundKind(kind) {
			continue
		}
		rule := boundRestrictionRule(kind)
		ownV, ordered, err := rc.boundLimit(own, rule)
		if err != nil {
			return err
		}
		if !ordered {
			continue
		}
		if err := rc.checkBoundAgainstBase(own, rule, ownV); err != nil {
			return err
		}
	}
	return nil
}

// checkBoundAgainstBase cross-checks ONE derived bound facet {value} against
// EVERY bound facet in the base's {facets}, per the four numbered conditions of
// that facet's valid-restriction SCC (boundRestrictionViolates).
//
// A malformed BASE-side operand is charged under the base facet's OWN
// valid-restriction rule (boundRestrictionRule(baseF.Kind())), not under rule —
// which names the DERIVED facet's SCC. A bad {value} on the base's minExclusive
// is a minExclusive-valid-restriction problem wherever it is noticed; reporting
// it as, say, maxInclusive-valid-restriction would name a constraint that has
// nothing to say about it (STYLE E2).
func (rc restrictionCheck) checkBoundAgainstBase(own xsd.Facet, rule xsderr.Rule, ownV Ordered) error {
	for _, ef := range rc.owner.Base().EffectiveFacets() {
		baseF := ef.Facet()
		if !isBoundKind(baseF.Kind()) {
			continue
		}
		baseV, ordered, err := rc.boundLimit(baseF, boundRestrictionRule(baseF.Kind()))
		if err != nil {
			return err
		}
		if !ordered {
			continue
		}
		if !boundRestrictionViolates(own.Kind(), baseF.Kind(), ownV.Cmp(baseV)) {
			continue
		}
		return xsderr.New(rule, rc.owner.Loc(),
			"simple type restriction's own %s {value} %q is not a valid restriction of the {base type definition}'s %s {value} %q (%s)",
			own.Kind(), boundLexical(own), baseF.Kind(), boundLexical(baseF), rule)
	}
	return nil
}

// boundLimit parses a bound facet's single {value} through the pass's mapping
// and returns it as an Ordered limit. ordered is false — with a nil error — when the parsed
// value does not implement Ordered, in which case the caller SKIPS this facet
// rather than rejecting: a bound facet on a non-ordered value space is an
// APPLICABILITY violation (cos-applicable-facets §4.1.5), already charged
// upstream by builtin.CheckSimpleTypeRestriction, and re-charging it here under
// a bound rule would name the wrong constraint (STYLE E2).
//
// A wrong {value} count, or a lexical the base type's mapping cannot parse, IS
// rejected under rule: every numbered condition of the SCC presupposes that the
// facet's {value} is a member of the base type definition's value space, so a
// {value} that is not one cannot satisfy the constraint.
func (rc restrictionCheck) boundLimit(f xsd.Facet, rule xsderr.Rule) (limit Ordered, ordered bool, err error) {
	values := f.Values()
	if len(values) != 1 {
		return nil, false, xsderr.New(rule, rc.owner.Loc(),
			"%s facet must carry exactly one value, has %d", f.Kind(), len(values))
	}
	v, err := facetValue(rc.mapping, rc.whiteSpace, values[0], nil)
	if err != nil {
		return nil, false, xsderr.Wrap(rule, rc.owner.Loc(), err)
	}
	ord, ok := v.(Ordered)
	if !ok {
		return nil, false, nil
	}
	return ord, true, nil
}

// boundRestrictionViolates reports whether a derived bound facet of kind derived
// violates its valid-restriction SCC against a base bound facet of kind base,
// given ord — the ·ordering· of the DERIVED {value} relative to the BASE {value}.
//
// Incomparable never violates: every numbered condition is a "greater than" /
// "less than" test, and on a partially ordered primitive (float/double) an
// incomparable pair satisfies none of them. That is the same reading facets.go's
// boundFacet.violates applies at instance time, where Incomparable is handled by
// its own separate clause rather than folded into the ordering tests.
func boundRestrictionViolates(derived, base xsd.FacetKind, ord Ordering) bool {
	switch derived {
	case xsd.FacetMaxInclusive:
		return maxInclusiveRestrictionViolates(base, ord)
	case xsd.FacetMaxExclusive:
		return maxExclusiveRestrictionViolates(base, ord)
	case xsd.FacetMinExclusive:
		return minExclusiveRestrictionViolates(base, ord)
	case xsd.FacetMinInclusive:
		return minInclusiveRestrictionViolates(base, ord)
	default:
		// Unreachable: every caller filters on isBoundKind first. Reported as
		// "no violation" rather than a panic because this is a predicate on
		// user-supplied schema data, not a capability assertion.
		return false
	}
}

// maxInclusiveRestrictionViolates is maxInclusive valid restriction (§4.3.7.4),
// clause by clause: 1 {value} greater than the base's maxInclusive; 2 {value}
// greater than or equal to the base's maxExclusive; 3 {value} less than the
// base's minInclusive; 4 {value} less than or equal to the base's minExclusive.
func maxInclusiveRestrictionViolates(base xsd.FacetKind, ord Ordering) bool {
	switch base {
	case xsd.FacetMaxInclusive:
		return ord == Greater
	case xsd.FacetMaxExclusive:
		return ord == Greater || ord == Equal
	case xsd.FacetMinInclusive:
		return ord == Less
	case xsd.FacetMinExclusive:
		return ord == Less || ord == Equal
	default:
		return false
	}
}

// maxExclusiveRestrictionViolates is maxExclusive valid restriction (§4.3.8.4),
// clause by clause: 1 {value} greater than the base's maxExclusive; 2 {value}
// greater than the base's maxInclusive; 3 {value} less than or equal to the
// base's minInclusive; 4 {value} less than or equal to the base's minExclusive.
// Clause 3 is the asymmetry a shared "interval" abstraction would erase: an
// exclusive upper bound EQUAL to the base's inclusive lower bound leaves an
// empty space and is an error, whereas the inclusive/inclusive pairing at
// maxInclusive clause 3 is not.
func maxExclusiveRestrictionViolates(base xsd.FacetKind, ord Ordering) bool {
	switch base {
	case xsd.FacetMaxExclusive:
		return ord == Greater
	case xsd.FacetMaxInclusive:
		return ord == Greater
	case xsd.FacetMinInclusive:
		return ord == Less || ord == Equal
	case xsd.FacetMinExclusive:
		return ord == Less || ord == Equal
	default:
		return false
	}
}

// minExclusiveRestrictionViolates is minExclusive valid restriction (§4.3.9.4),
// clause by clause: 1 {value} less than the base's minExclusive; 2 {value} less
// than the base's minInclusive; 3 {value} greater than or equal to the base's
// maxInclusive; 4 {value} greater than or equal to the base's maxExclusive.
func minExclusiveRestrictionViolates(base xsd.FacetKind, ord Ordering) bool {
	switch base {
	case xsd.FacetMinExclusive:
		return ord == Less
	case xsd.FacetMinInclusive:
		return ord == Less
	case xsd.FacetMaxInclusive:
		return ord == Greater || ord == Equal
	case xsd.FacetMaxExclusive:
		return ord == Greater || ord == Equal
	default:
		return false
	}
}

// minInclusiveRestrictionViolates is minInclusive valid restriction (§4.3.10.4),
// clause by clause: 1 {value} less than the base's minInclusive; 2 {value}
// greater than the base's maxInclusive (the spec's "is greater the {value} of
// that maxInclusive" is missing a "than"); 3 {value} less than or equal to the
// base's minExclusive; 4 {value} greater than or equal to the base's
// maxExclusive.
func minInclusiveRestrictionViolates(base xsd.FacetKind, ord Ordering) bool {
	switch base {
	case xsd.FacetMinInclusive:
		return ord == Less
	case xsd.FacetMaxInclusive:
		return ord == Greater
	case xsd.FacetMinExclusive:
		return ord == Less || ord == Equal
	case xsd.FacetMaxExclusive:
		return ord == Greater || ord == Equal
	default:
		return false
	}
}

// checkEnumerationRestriction charges enumeration valid restriction (§4.3.5.5):
// "it is an error if any member of {value} is not in the ·value space· of {base
// type definition}". MEMBERSHIP in that value space is the whole test, and it is
// strictly narrower than "the base's governing mapping can parse this lexical":
// the base type definition's own facets carve its value space out of its
// primitive's, so `200` is a perfectly well-formed integer lexical that is NOT in
// xs:byte's value space (bounded to [-128,127] by inherited maxInclusive /
// minInclusive), and `-5` is not in xs:positiveInteger's. Parsing alone accepts
// both. Each member is therefore run through the ordinary lexical→value pipeline
// against the BASE TYPE DEFINITION itself (ValidateLexical), which applies the
// base's whiteSpace, its pattern facets, its governing mapping and its value
// facets — the base's full facet stack, which is exactly what "value space of
// {base type definition}" denotes.
//
// The base — not, as newEnumFacet does for the instance-time check, the type that
// DECLARED the facet. The two differ exactly where this SCC bites: a facet
// declared on the derived type D resolves to D's own type, whose facets may be
// narrower than the base's, so a member outside the base's value space could
// still validate there.
//
// Each member carries its own namespace context — the bindings in scope where its
// <enumeration> was written (§3.3.18) — threaded through the same memberContext
// newEnumFacet uses, so a QName/NOTATION member resolves its prefix against the
// declaring schema's scope rather than against nothing.
//
// b is the same Backend CheckFacetRestriction was handed; it is a parameter
// rather than a restrictionCheck field because only this check needs it, and its
// caller has already established that b governs the base (an unmapped base
// returns early there), so ValidateLexical's no-mapping error is unreachable from
// here.
//
// A facet-pipeline PRECONDITION fault in the BASE type's own facets is SKIPPED, not
// charged (IsFacetPrecondition, ValidateLexical) — the same "skip, don't
// mis-attribute" answer boundLimit gives an unordered bound {value} above, and for
// the same reason (STYLE E2). §4.3.5.5 asks whether a member is in the base's value
// space; a base carrying a facet not applicable to it has no well-defined value
// space to be in, so the fault is an APPLICABILITY violation of the base —
// builtin.CheckSimpleTypeRestriction's to charge, against the base, under §4.1.5 —
// and re-charging it here as enumeration-valid-restriction against the DERIVED type
// would name a constraint with nothing to say about it and reject a schema whose
// enumeration may be perfectly valid.
func (rc restrictionCheck) checkEnumerationRestriction(b Backend) error {
	for _, own := range rc.owner.OwnFacets() {
		if own.Kind() != xsd.FacetEnumeration {
			continue
		}
		// Kind() is FacetEnumeration here, so EnumerationMembers always reports
		// ok=true; the second result is discarded deliberately.
		members, _ := own.EnumerationMembers()
		for _, em := range members {
			_, err := ValidateLexical(b, rc.owner.Base(), em.Lexical(), newMemberContext(em))
			if IsFacetPrecondition(err) {
				continue
			}
			if err != nil {
				return xsderr.Wrap(ruleEnumerationValidRestriction, rc.owner.Loc(), err)
			}
		}
	}
	return nil
}

// isBoundKind reports whether kind is one of the four bound Constraining Facets
// (§4.3.7–§4.3.10) — the kinds whose {value} is a member of the type's value
// space and is compared through the ·ordering· relation.
func isBoundKind(kind xsd.FacetKind) bool {
	switch kind {
	case xsd.FacetMaxInclusive, xsd.FacetMaxExclusive, xsd.FacetMinInclusive, xsd.FacetMinExclusive:
		return true
	default:
		return false
	}
}

// boundRestrictionRule maps a bound facet kind to its construction-time
// valid-restriction rule ID (§4.3.7.4–§4.3.10.4) — the schema-construction
// sibling of boundRule's instance-time cvc-* IDs, and it panics on a non-bound
// kind for the same reason: every caller filters on isBoundKind first, so
// reaching the default is a package-internal bug, not schema data.
func boundRestrictionRule(k xsd.FacetKind) xsderr.Rule {
	switch k {
	case xsd.FacetMaxInclusive:
		return ruleMaxInclusiveValidRestriction
	case xsd.FacetMaxExclusive:
		return ruleMaxExclusiveValidRestriction
	case xsd.FacetMinInclusive:
		return ruleMinInclusiveValidRestriction
	case xsd.FacetMinExclusive:
		return ruleMinExclusiveValidRestriction
	default:
		panic("value: boundRestrictionRule: " + k.String() + " is not a bound facet")
	}
}

// boundLexical renders a bound facet's lexical {value} for an error message.
// boundLimit has already rejected any facet that does not carry exactly one
// value, so the empty fallback is message rendering only, never a validity
// decision.
func boundLexical(f xsd.Facet) string {
	values := f.Values()
	if len(values) != 1 {
		return ""
	}
	return values[0]
}
