package xsd

// This file renders Wildcard Subset (Structures §3.10.6.2, cos-ns-subset) — the
// "is every expanded name sub admits also admitted by super?" relation between
// two Namespace Constraints. It is the SUBSET counterpart of
// namespaceconstraint_intersect.go's cos-aw-intersect: both are §3.10.6
// wildcard-set operations over the same record, and neither re-derives the
// other's algebra.
//
// It stays UNEXPORTED. The only consumer is contentrestricts.go's
// wildcard-versus-wildcard transition test, which is in-package (STYLE T5:
// export nothing without a consumer). #52 owns the exported wildcard-set
// surface and will generalize this relation there when a library consumer for
// it exists; until then one unexported implementation is the whole of it, so
// there is no second encoding to drift (STYLE T4).

// wildcardSubset is cos-ns-subset: "Given two Namespace Constraints sub and
// super, sub is a wildcard subset of super if and only if ONE of the following
// is true" (the four variety/namespaces cases, wildcardVarietySubset) "AND ALL
// of the following are true" (the three {disallowed names} conditions,
// disallowedNamesSubset). The second group is a CONJUNCTIVE addendum to the
// first, not an alternative to it, so both halves must hold.
//
// The relation is asymmetric and the parameter names carry the direction: sub
// is the narrower constraint, super the wider one.
func wildcardSubset(sub, super NamespaceConstraint) bool {
	if !wildcardVarietySubset(sub, super) {
		return false
	}
	return disallowedNamesSubset(sub, super)
}

// wildcardVarietySubset decides the four-case disjunction of cos-ns-subset:
//
//   - clause 1: super.{variety} = any;
//   - clause 2: both enumeration, and super.{namespaces} ⊇ sub.{namespaces};
//   - clause 3: sub enumeration, super not, and the two {namespaces} disjoint;
//   - clause 4: both not, and super.{namespaces} ⊆ sub.{namespaces}.
//
// The switch is on SUPER's variety because the four clauses partition on it:
// clause 1 is super = any, clause 2 is super = enumeration, and clauses 3 and 4
// are the two sub-varieties admissible under super = not. No other pairing
// appears in the constraint, so anything the switch does not reach is not a
// wildcard subset — notably an "any" sub under an "enumeration" or "not" super,
// which genuinely admits names super does not.
//
// Membership is the same == identity test AllowsNamespace uses (Namespace is
// comparable), read off the document-order {namespaces} slices; no map is
// consulted, so no iteration order reaches the verdict (STYLE D2).
func wildcardVarietySubset(sub, super NamespaceConstraint) bool {
	switch super.variety {
	case NamespaceConstraintAny:
		return true // clause 1
	case NamespaceConstraintEnumeration:
		if sub.variety != NamespaceConstraintEnumeration {
			return false
		}
		return everyNamespaceIn(sub.namespaces, super) // clause 2
	case NamespaceConstraintNot:
		if sub.variety == NamespaceConstraintEnumeration {
			return noNamespaceIn(sub.namespaces, super) // clause 3, disjoint
		}
		if sub.variety != NamespaceConstraintNot {
			return false
		}
		return everyNamespaceIn(super.namespaces, sub) // clause 4
	default:
		return false // an unconstructed (zero) record admits nothing and is nothing's superset
	}
}

// everyNamespaceIn reports whether every member of names is a member of c's
// {namespaces}.
func everyNamespaceIn(names []Namespace, c NamespaceConstraint) bool {
	for _, n := range names {
		if !c.hasNamespace(n) {
			return false
		}
	}
	return true
}

// noNamespaceIn reports whether no member of names is a member of c's
// {namespaces} — the disjointness cos-ns-subset clause 3 asks for.
func noNamespaceIn(names []Namespace, c NamespaceConstraint) bool {
	for _, n := range names {
		if c.hasNamespace(n) {
			return false
		}
	}
	return true
}

// disallowedNamesSubset decides the conjunctive {disallowed names} tail of
// cos-ns-subset:
//
//  1. each QName member of super.{disallowed names} is not allowed by sub, as
//     defined in Wildcard allows Expanded Name (§3.10.4.2, cvc-wildcard-name);
//  2. if super.{disallowed names} contains defined, so does sub's;
//  3. if super.{disallowed names} contains sibling, so does sub's.
//
// Condition 1 goes through AllowsName, the canonical cvc-wildcard-name entry
// point, rather than re-deriving the allowance algorithm (namespaceconstraint.go's
// standing instruction to callers).
//
// The keyword conditions read the keyword half of the ONE {disallowed names}
// property through hasDisallowedNameKeyword; the QName half is walked in
// document order (STYLE D2).
func disallowedNamesSubset(sub, super NamespaceConstraint) bool {
	for _, name := range super.disallowedNames {
		if sub.AllowsName(name) {
			return false // condition 1
		}
	}
	if super.hasDisallowedNameKeyword(DisallowedNameDefined) && !sub.hasDisallowedNameKeyword(DisallowedNameDefined) {
		return false // condition 2
	}
	if super.hasDisallowedNameKeyword(DisallowedNameSibling) && !sub.hasDisallowedNameKeyword(DisallowedNameSibling) {
		return false // condition 3
	}
	return true
}
