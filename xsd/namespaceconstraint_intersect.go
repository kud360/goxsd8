package xsd

import "github.com/kud360/goxsd8/xsderr"

// IntersectNamespaceConstraint returns the Attribute Wildcard Intersection
// (Structures §3.10.6.4, id="cos-aw-intersect") of a and b: the Namespace
// Constraint that admits a namespace name (and expanded name) iff BOTH a and b
// admit it. It is the binary primitive of the §3.6.2.2 "Common Rules for
// Attribute Wildcards" combination (declare-attributeGroup-wildcard) applied by
// the parser's <attributeGroup>/<complexType> attribute-wildcard producers; the
// spec's "more than two" case (§3.10.6.4 final paragraph) is a plain left fold of
// this primitive done by the caller, not by this function.
//
// The {variety}/{namespaces} result is the §3.10.6.4 five-case table:
//
//  1. a and b identical in variety+namespaces -> that same pair;
//  2. either is any                           -> the other's pair;
//  3. both enumeration                        -> variety enumeration,
//     {namespaces} = set intersection;
//  4. both not                                -> variety not,
//     {namespaces} = set UNION;
//  5. one not (set S1), one enumeration (S2)  -> variety enumeration,
//     {namespaces} = S2 minus S1.
//
// Case 1 needs no special arm: two identical enumeration sets intersect to that
// set (case 3) and two identical not sets union to that set (case 4).
//
// The result's {disallowed names} is the union of a's members whose namespace
// name b admits with b's members whose namespace name a admits (§3.10.6.4). This
// filter is applied EXPLICITLY here — never by handing the unfiltered union to
// NewNamespaceConstraint, whose w-props-correct clause-4 check REJECTS a
// disallowed name the spec would silently drop rather than dropping it. The
// ##defined keyword clause of that union is inapplicable: this package does not
// represent the defined/sibling keywords at all (the GAP marker on
// NewNamespaceConstraint), so it contributes nothing.
//
// The result is built through NewNamespaceConstraint, so w-props-correct clauses
// 1-4 (§3.10.6.1) are re-checked and {namespaces}/{disallowed names} are
// deduplicated and copied by the one canonical path (STYLE T4) rather than
// duplicated here. A correct intersection always satisfies those clauses — every
// retained {disallowed names} member's namespace is admitted by both operands
// hence by the intersection (clause 4 is a confirming assertion, not the filter),
// and every case yields a representable record — so the error is unreachable for
// two validly-constructed operands; it is returned rather than swallowed so any
// future divergence fails closed as a w-props-correct *xsderr.Error at loc
// instead of a silently ill-formed wildcard (STYLE T1/P3).
//
// Intersection is commutative: (loc, a, b) and (loc, b, a) yield equal results.
// loc charges the (defensive) rejection position; a synthesized caller may pass
// the zero xsderr.Loc{}.
func IntersectNamespaceConstraint(loc xsderr.Loc, a, b NamespaceConstraint) (NamespaceConstraint, error) {
	variety, namespaces := intersectVarietyAndSet(a, b)
	disallowed := intersectDisallowedNames(a, b)
	return NewNamespaceConstraint(loc, variety, namespaces, disallowed)
}

// intersectVarietyAndSet computes the {variety}/{namespaces} of the intersection
// per the §3.10.6.4 five-case table (see IntersectNamespaceConstraint). It reads
// the operands' sealed {variety} within its defining package, which is not a
// forbidden type switch (STYLE T3 governs concrete switches outside the package).
func intersectVarietyAndSet(a, b NamespaceConstraint) (NamespaceConstraintVariety, []Namespace) {
	if a.variety == NamespaceConstraintAny {
		return b.variety, b.namespaces // case 2: any ∩ X = X
	}
	if b.variety == NamespaceConstraintAny {
		return a.variety, a.namespaces // case 2 (symmetric)
	}
	if a.variety == NamespaceConstraintEnumeration && b.variety == NamespaceConstraintEnumeration {
		return NamespaceConstraintEnumeration, intersectNamespaces(a.namespaces, b.namespaces) // case 3
	}
	if a.variety == NamespaceConstraintNot && b.variety == NamespaceConstraintNot {
		return NamespaceConstraintNot, unionNamespaces(a.namespaces, b.namespaces) // case 4
	}
	// case 5: one not (S1), one enumeration (S2) -> enumeration, S2 minus S1.
	if a.variety == NamespaceConstraintEnumeration {
		return NamespaceConstraintEnumeration, differenceNamespaces(a.namespaces, b.namespaces)
	}
	return NamespaceConstraintEnumeration, differenceNamespaces(b.namespaces, a.namespaces)
}

// intersectDisallowedNames computes the {disallowed names} of the intersection
// (§3.10.6.4): a's QName members whose namespace name b admits, followed by b's
// members whose namespace name a admits, in document order (a first, then b).
// A member the OTHER operand's namespace test rejects is silently DROPPED (the
// spec's filter) rather than carried into NewNamespaceConstraint, whose clause-4
// check would reject it. The seen-free append may leave a cross-operand duplicate;
// NewNamespaceConstraint's dedupQNames removes it (STYLE T4).
func intersectDisallowedNames(a, b NamespaceConstraint) []QName {
	var out []QName
	for _, name := range a.disallowedNames {
		if b.AllowsNamespace(NamespaceName(name.Space)) {
			out = append(out, name)
		}
	}
	for _, name := range b.disallowedNames {
		if a.AllowsNamespace(NamespaceName(name.Space)) {
			out = append(out, name)
		}
	}
	return out
}

// intersectNamespaces returns the members of a that are also members of b, in a's
// document order. The inB map is a membership set only; output order is a's,
// never a map iteration order (STYLE D2/D3). Inputs come from validly-constructed
// NamespaceConstraints (already deduplicated), so the result carries no duplicate.
func intersectNamespaces(a, b []Namespace) []Namespace {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	inB := make(map[Namespace]struct{}, len(b))
	for _, n := range b {
		inB[n] = struct{}{}
	}
	var out []Namespace
	for _, n := range a {
		if _, ok := inB[n]; ok {
			out = append(out, n)
		}
	}
	return out
}

// unionNamespaces returns the members of a followed by the members of b not
// already contributed by a, in document order (a's order, then b's new members).
// The seen map is a membership set only; output order is never a map iteration
// order (STYLE D2/D3).
func unionNamespaces(a, b []Namespace) []Namespace {
	seen := make(map[Namespace]struct{}, len(a)+len(b))
	out := make([]Namespace, 0, len(a)+len(b))
	for _, n := range a {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	for _, n := range b {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}

// differenceNamespaces returns the members of a that are NOT members of b, in a's
// document order. The inB map is a membership set only; output order is a's,
// never a map iteration order (STYLE D2/D3).
func differenceNamespaces(a, b []Namespace) []Namespace {
	if len(a) == 0 {
		return nil
	}
	inB := make(map[Namespace]struct{}, len(b))
	for _, n := range b {
		inB[n] = struct{}{}
	}
	var out []Namespace
	for _, n := range a {
		if _, ok := inB[n]; !ok {
			out = append(out, n)
		}
	}
	return out
}
