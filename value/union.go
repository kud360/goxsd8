package value

import (
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file adds the union {variety} to the backend-generic pipeline: the member
// dispatch that decides a union-variety *xsd.SimpleType end to end
// (validateUnion/dispatchUnion) and the Mapping that exposes the same dispatch to
// the facet-{value} parsing seam (unionMapping, reached through
// governingMapping). Before this, governingMapping walked only the atomic base
// chain and a union resolved to no mapping at all, so ValidateLexical on any
// union-variety type returned "no backend mapping governs type"
// (a cvc-datatype-valid error) regardless of instance validity.
//
// Unlike list, union contributes NO value type. dv_list (§4.1.4 cl.2.2) builds a
// genuinely new value — "the sequence consisting of the values..." — which is why
// listValue exists; dv_union (cl.2.3) says only "let V be the value identified by
// Datatype Valid for L with respect to B", i.e. V IS the dispatched member's own
// value, in that member's own value space, with that member's own capabilities. A
// unionValue wrapper would be a second encoding of one fact (PRINCIPLES 5) and
// would hide the very capabilities (Identical/Eq/Ordered/…) the union's own
// enumeration facet has to compare through.
//
// Members are taken as DIRECT members, never a flattened membership: a member may
// itself be a union (§3.16.1 std-member_type_definitions admits any ordinary
// simple type; cos-st-restricts clause 3 forbids only special members and cyclic
// self-membership), and flattening would silently drop the intervening union's own
// pattern/enumeration facets, which clauses 1 and 3 apply at that nesting level
// (PRINCIPLES 11). dispatchUnion therefore recurses through validateLexical, which
// re-enters here for a nested union — the ·active member type· → ·active basic
// member· descent the Datatypes Terminology defines.

// validateUnion decides a union-variety st end to end per cvc-datatype-valid
// (§4.1.4): clause 2.3 (dv_union) dispatches the literal to st's {member type
// definitions}, clause 1 checks st's OWN ·lexical· facets (pattern) and clause 3
// its own ·value-based· facets (enumeration) — the only two constraining facets
// besides assertions that are applicable to a union at all (cos-applicable-facets
// §4.1.5), which is also why no whiteSpace stage runs here.
//
// The dispatch runs FIRST, before st's own pattern check, even though the pattern
// is clause 1 and the dispatch clause 2.3. That is not a reordering of the
// constraint — the three clauses are conjuncts of one "all of the following are
// true", so every one still has to hold — but a forced consequence of the
// dv_vfacets note: the ·pre-lexical· facets that produce the literal L clause 1
// tests are "those associated with B in clause 2.3 above", so the ·active basic
// member· B has to be known before there is a normalized literal to match at all
// (PRINCIPLES 11). Equally, st's own facets never influence WHICH member is
// active: clause 2.3 fixes B from member validity alone, so a literal that its
// active member accepts but st's own pattern or enumeration rejects is a
// rejection, never a retry against a later member.
func validateUnion(b Backend, st *xsd.SimpleType, u xsd.Union, rawLexical string, ctx Context) (Value, whiteSpace, error) {
	if !unionGoverned(b, u) {
		return nil, 0, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"value: no backend mapping governs type %s", st.Name())
	}
	lexFacets, valFacets, err := compile(b, st)
	if err != nil {
		return nil, 0, err
	}
	v, ws, err := dispatchUnion(b, u, rawLexical, ctx)
	if err != nil {
		return nil, 0, err
	}

	// clause 1 (cvc-pattern-valid, §4.3.4.4) on the literal as the active basic
	// member normalized it, NOT on the raw one and not on a union-level
	// normalization (a union has no whiteSpace facet to normalize with).
	lexical := normalizeWhiteSpace(rawLexical, ws)
	for _, lf := range lexFacets {
		if err := lf.CheckLexical(lexical); err != nil {
			return nil, 0, err
		}
	}

	// clause 3 (cvc-enumeration-valid, §4.3.5.4) on V, the dispatched member's own
	// value: enumMatch compares through V's own identity/equality relations
	// (§2.2.1/§2.2.2), which is what "equal or identical" means once the active
	// member has fixed the value space.
	for _, vf := range valFacets {
		if err := vf.CheckValue(v); err != nil {
			return nil, 0, err
		}
	}
	return v, ws, nil
}

// dispatchUnion is cvc-datatype-valid clause dv_union (§4.1.4 cl.2.3): a literal
// is Datatype Valid with respect to a union iff it is Datatype Valid with respect
// to at least one member of the {member type definitions}, and V is the value
// that member identifies. Members are tried in {member type definitions} order and
// the FIRST that accepts wins — the ·active member type· is "the first of its
// members in order which accepts the instance as valid" (Datatypes Terminology),
// so order is load-bearing and the scan short-circuits rather than looking for a
// best match. A member that is itself a union recurses through validateLexical,
// so the whiteSpace returned is always the ·active basic member·'s: the non-union
// type at the bottom of that chain.
//
// Each member is handed the RAW literal. A union carries no whiteSpace facet of
// its own (cos-applicable-facets §4.1.5, §4.3.6: "for all datatypes ·constructed·
// by ·union· whiteSpace does not apply directly"), so every candidate normalizes
// the literal through its OWN whiteSpace before deciding — which is exactly how
// one literal can be rejected by an early member and accepted by a later one on
// normalization grounds alone.
//
// A member's rejection is not an error to propagate: it IS the dispatch
// mechanism. So the rejections are collected rather than dropped (STYLE S3) and,
// when no member accepts, folded into the one cvc-datatype-valid rejection
// returned here, which names every member's reason in membership order
// (deterministic, STYLE D2). A union whose {member type definitions} is EMPTY —
// xs:error (Structures §3.16.7.3), whose value and lexical spaces are both empty —
// falls out of the loop with zero candidates and so rejects every literal
// including "", with no special case.
func dispatchUnion(b Backend, u xsd.Union, rawLexical string, ctx Context) (Value, whiteSpace, error) {
	members := u.Members()
	// Left nil so the common case — an early member accepts — allocates nothing;
	// the slice only materializes on the path that actually reports rejections.
	var rejections []string
	for i, m := range members {
		v, ws, err := validateLexical(b, m, rawLexical, ctx)
		if err == nil {
			return v, ws, nil
		}
		rejections = append(rejections, fmt.Sprintf("member %d (%s): %v", i, m.Name(), err))
	}
	return nil, 0, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
		"value %q is Datatype Valid with respect to no member of the union's %d {member type definitions} (cvc-datatype-valid clause 2.3, §4.1.4): %s",
		rawLexical, len(members), strings.Join(rejections, "; "))
}

// unionMapping builds the lexical mapping for a union-variety type out of its
// {member type definitions}, so a union resolves through governingMapping exactly
// as a list does: Parse IS the dv_union dispatch (§4.1.4 cl.2.3) and returns the
// ·active member type·'s own value verbatim.
//
// Its live call site is facet-{value} parsing, not instance validation. An
// enumeration facet DECLARED on a union type has each of its {value} members
// parsed through this mapping (newEnumFacet → declaringFacetSpace → facetValue),
// which is precisely what src-enumeration-value (§4.3.5.3) requires — a member
// literal must itself be Datatype Valid with respect to the type declaring the
// facet, i.e. dispatched to some member of that union. Instance validation takes
// validateUnion instead, because a Mapping cannot report WHICH basic member won
// and the union's own pattern facet needs that member's whiteSpace normalization
// (the dv_vfacets note).
//
// Canonical is deliberately nil, as in listMapping: a union value's canonical form
// is the active member's, which this mapping cannot name having dropped the member
// it dispatched to. Per the Mapping doc a nil Canonical means "this type has no
// canonical form", which callers must treat as such rather than as an error.
func unionMapping(b Backend, u xsd.Union) Mapping {
	return Mapping{
		Parse: func(lexical string, ctx Context) (Value, error) {
			v, _, err := dispatchUnion(b, u, lexical, ctx)
			return v, err
		},
	}
}

// unionGoverned reports whether b governs EVERY member of u, the union's analogue
// of listMapping's "an ungoverned item type leaves the list ungoverned".
//
// EVERY member, not merely one: dv_union takes the FIRST member that accepts
// (§4.1.4 cl.2.3), and an unmapped member is indistinguishable from one that
// rejects — so a partially mapped union would skip past the member that should
// have been active and hand back a LATER member's value (a wrong V for a verdict
// that still says "valid"), or reject outright when no other member accepts.
// Reporting the whole union ungoverned instead keeps an unmapped type a BACKEND
// gap rather than a validity verdict about instance data: it surfaces as
// ValidateLexical's "no backend mapping governs" cvc-datatype-valid error and as a
// skipped CheckFacetRestriction, the same way an ungoverned atomic type does.
//
// An EMPTY membership is vacuously governed, which is right for xs:error
// (§3.16.7.3): its value space is empty, so its mapping's whole job is to reject
// every literal — what dispatchUnion does with zero candidates.
func unionGoverned(b Backend, u xsd.Union) bool {
	for _, m := range u.Members() {
		if _, ok := governingMapping(b, m); !ok {
			return false
		}
	}
	return true
}
