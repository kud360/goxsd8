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
	ws, wsKnown := baseWhiteSpace(base)
	rc := restrictionCheck{mapping: m, whiteSpace: ws, whiteSpaceKnown: wsKnown, owner: t, base: base}
	if err := rc.checkBoundRestrictions(); err != nil {
		return err
	}
	return rc.checkEnumerationRestriction()
}

// restrictionCheck is the resolved context one CheckFacetRestriction pass needs:
// the type under construction, its {base type definition}, the base's governing
// mapping, and the base's in-force whiteSpace mode. Resolving all four once and
// carrying them together keeps every facet {value} in this pass parsed the same
// way, which is what makes the resulting values comparable at all.
type restrictionCheck struct {
	mapping    Mapping
	whiteSpace whiteSpace
	// whiteSpaceKnown is false when no whiteSpace facet is in force on the base
	// (a union {variety}, or a base carrying none) or its {value} is outside the
	// §4.3.6.1 domain — see baseWhiteSpace. A facet lexical is then passed to
	// Parse unchanged.
	whiteSpaceKnown bool
	owner           *xsd.SimpleType
	base            *xsd.SimpleType
}

// baseWhiteSpace resolves the whiteSpace mode in force on base, used to
// normalize a facet's lexical {value} before it is parsed in base's value space.
// The XML mapping of a facet's value [attribute] interprets it through the base
// type's lexical mapping, whose first stage is that normalization (§4.3.6, key-nv
// §3.1.4), so a maxInclusive written as "2002-10-10T12:00:00-05:00 " on a
// collapse-normalized base denotes the same {value} as the untrailed spelling.
//
// known is false when no whiteSpace facet is in force or its {value} is outside
// the three-token domain; the caller then leaves the lexical unchanged. It
// deliberately does not reuse effectiveWhiteSpace, which PANICS on both of those
// states: there they are instance-validation invariants, here they are ordinary
// facts about a caller-supplied type being constructed (xs:anyAtomicType, for
// one, carries no whiteSpace facet at all). Only the token table is shared, via
// whiteSpaceOf.
func baseWhiteSpace(base *xsd.SimpleType) (ws whiteSpace, known bool) {
	for _, ef := range base.EffectiveFacets() {
		if ef.Facet().Kind() != xsd.FacetWhiteSpace {
			continue
		}
		values := ef.Facet().Values()
		if len(values) != 1 {
			return 0, false
		}
		return whiteSpaceOf(values[0])
	}
	return 0, false
}

// normalize applies the base type's whiteSpace normalization to a facet's
// lexical {value}, or returns it unchanged when no mode is in force.
func (rc restrictionCheck) normalize(lexical string) string {
	if !rc.whiteSpaceKnown {
		return lexical
	}
	return normalizeWhiteSpace(lexical, rc.whiteSpace)
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
func (rc restrictionCheck) checkBoundAgainstBase(own xsd.Facet, rule xsderr.Rule, ownV Ordered) error {
	for _, ef := range rc.base.EffectiveFacets() {
		baseF := ef.Facet()
		if !isBoundKind(baseF.Kind()) {
			continue
		}
		baseV, ordered, err := rc.boundLimit(baseF, rule)
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
	v, err := rc.mapping.Parse(rc.normalize(values[0]), nil)
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
// type definition}". Each member is parsed through the BASE type's governing
// mapping — NOT, as newEnumFacet does for the instance-time check, through the
// mapping of the type that DECLARED the facet. The two differ exactly where this
// SCC bites: a facet declared on the derived type D resolves to D's own governing
// mapping, which may be narrower than the base's, so a member outside the base's
// value space could still parse there.
//
// Each member carries its own namespace context — the bindings in scope where its
// <enumeration> was written (§3.3.18) — threaded through the same memberContext
// newEnumFacet uses, so a QName/NOTATION member resolves its prefix against the
// declaring schema's scope rather than against nothing.
func (rc restrictionCheck) checkEnumerationRestriction() error {
	for _, own := range rc.owner.OwnFacets() {
		if own.Kind() != xsd.FacetEnumeration {
			continue
		}
		// Kind() is FacetEnumeration here, so EnumerationMembers always reports
		// ok=true; the second result is discarded deliberately.
		members, _ := own.EnumerationMembers()
		for _, em := range members {
			if _, err := rc.mapping.Parse(rc.normalize(em.Lexical()), newMemberContext(em)); err != nil {
				return xsderr.Wrap("enumeration-valid-restriction", rc.owner.Loc(), err)
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
		return "maxInclusive-valid-restriction"
	case xsd.FacetMaxExclusive:
		return "maxExclusive-valid-restriction"
	case xsd.FacetMinInclusive:
		return "minInclusive-valid-restriction"
	case xsd.FacetMinExclusive:
		return "minExclusive-valid-restriction"
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
