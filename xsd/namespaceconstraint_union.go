package xsd

import "github.com/kud360/goxsd8/xsderr"

// UnionNamespaceConstraint returns the Attribute Wildcard Union (Structures
// §3.10.6.3, id="cos-aw-union") of a and b: the Namespace Constraint that admits
// a namespace name iff a or b admits it. It is the mirror of
// namespaceconstraint_intersect.go's IntersectNamespaceConstraint, over the same
// record and the same set helpers (namespaceconstraint_sets.go), and the two must
// be read together — several of their clauses differ only by a swapped operator.
//
// Its out-of-package consumer is the parser's Open Content producer: §3.4.2.3.3
// (dcl.ctd.ctcc.common) clause 6.2 defines a derived complex type's {open
// content}.{wildcard} as this union of the wildcard corresponding to the
// ·wildcard element·'s <any> child and the ·explicit content type·'s own
// {open content}.{wildcard}, which the producer reaches when an <extension>
// carrying an <openContent> derives from a base that already has one
// (parser/produce_complex.go, openContentWildcard). In-package,
// contentrestricts.go's coveringWildcardUnion folds live element wildcards
// through it. The §3.6.2.2 <extension> attribute-wildcard combination (§3.4.2.5
// dcl.ctd.anyatt clause 2.2.2.3 — NOT §3.4.2.2, which is the simple-content
// mapping) is the third spec use and lands with #265.
//
// The {variety}/{namespaces} result is the §3.10.6.3 five-case table:
//
//  1. a and b identical in variety+namespaces -> that same pair;
//  2. either is any                           -> any;
//  3. both enumeration                        -> variety enumeration,
//     {namespaces} = set UNION;
//  4. both not                                -> the set INTERSECTION, and then
//     4.1 empty     -> variety any;
//     4.2 non-empty -> variety not with that intersection;
//  5. one not (set S1), one enumeration (S2)  -> the set difference S1 minus S2,
//     and then
//     5.1 empty     -> variety any;
//     5.2 non-empty -> variety not with that difference.
//
// Case 1 needs no special arm: two identical enumeration sets union to that set
// (case 3), and two identical non-empty not sets intersect to that set (case
// 4.2, whose non-emptiness w-props-correct clause 2 guarantees).
//
// Cases 4 and 5 are the two places variety is NOT a function of the operand
// varieties alone, which is why unionVarietyAndSet computes the set FIRST and
// only then reads its emptiness to choose between any and not. Choosing not up
// front and repairing it afterwards would momentarily name a record
// w-props-correct clause 2 forbids.
//
// The result's {disallowed names} is built from the §3.10.6.3 three-bullet list.
// Bullets 1-2 (the QName half): a's members NOT ALLOWED BY b, followed by b's
// members not allowed by a — the cvc-wildcard-name (§3.10.4.2) expanded-name
// test, which is a strictly different predicate from cos-aw-intersect's
// namespace-name test (cvc-wildcard-namespace, §3.10.4.3); the two filters are
// not interchangeable and neither file may borrow the other's.
// Bullet 3 (the keyword half): "the keyword defined if it is contained in BOTH
// {disallowed names}" — a set INTERSECTION. Do NOT transpose this with
// cos-aw-intersect (§3.10.6.4), whose corresponding clause is an OR ("a member of
// EITHER"): wildcard union is AND for defined, wildcard intersection is OR. The
// spec's own Note licenses the resulting over-approximation — dropping defined
// "may allow QNames that are not allowed by either wildcard[;] this is to ensure
// that all unions are expressible" — so the asymmetry is deliberate, not a
// simplification taken here. The keyword sibling has NO bullet in this formula at
// all, so an operand carrying it has it dropped from the result — silently, with
// no defensive branch, because that is exactly what §3.10.6.3 defines. Operands do
// reach here carrying it: coveringWildcardUnion folds the {namespace constraint}s
// of ELEMENT wildcards, on which sibling is legal (parser/produce_complex.go maps
// ##definedSibling to it in the non-attribute-wildcard case). Dropping it only
// ever WIDENS the union — one fewer excluded name — so the loss is fail-open at
// every call site, attribute or element, and can never cause a false reject.
//
// The result is built through NewNamespaceConstraint, so w-props-correct clauses
// 1-4 (§3.10.6.1) are re-checked and {namespaces}/{disallowed names} are
// deduplicated and copied by the one canonical path (STYLE T4) rather than
// duplicated here. A correct union always satisfies those clauses — the union
// admits every namespace name either operand admits, so every retained
// {disallowed names} member's namespace, already admitted by the operand that
// carried it (that operand's own clause 4), is admitted by the union too, making
// clause 4 a confirming assertion rather than the filter — so the error is
// unreachable for two validly-constructed operands; it is returned rather than
// swallowed so any future divergence fails closed as a w-props-correct
// *xsderr.Error at loc instead of a silently ill-formed wildcard (STYLE T1/P3).
//
// Union is commutative: (loc, a, b) and (loc, b, a) yield equal results. loc
// charges the (defensive) rejection position; a synthesized caller may pass the
// zero xsderr.Loc{}.
func UnionNamespaceConstraint(loc xsderr.Loc, a, b NamespaceConstraint) (NamespaceConstraint, error) {
	variety, namespaces := unionVarietyAndSet(a, b)
	disallowed := unionDisallowedNames(a, b)
	return NewNamespaceConstraint(loc, variety, namespaces, disallowed, unionDisallowedNameKeywords(a, b))
}

// unionVarietyAndSet computes the {variety}/{namespaces} of the union per the
// §3.10.6.3 five-case table (see UnionNamespaceConstraint). It reads the
// operands' sealed {variety} within its defining package, which is not a
// forbidden type switch (STYLE T3 governs concrete switches outside the package).
func unionVarietyAndSet(a, b NamespaceConstraint) (NamespaceConstraintVariety, []Namespace) {
	if a.variety == NamespaceConstraintAny || b.variety == NamespaceConstraintAny {
		return NamespaceConstraintAny, nil // case 2: any ∪ X = any
	}
	if a.variety == NamespaceConstraintEnumeration && b.variety == NamespaceConstraintEnumeration {
		return NamespaceConstraintEnumeration, unionNamespaces(a.namespaces, b.namespaces) // case 3
	}
	if a.variety == NamespaceConstraintNot && b.variety == NamespaceConstraintNot {
		return notOrAny(intersectNamespaces(a.namespaces, b.namespaces)) // case 4
	}
	// case 5: one not (S1), one enumeration (S2) -> S1 minus S2.
	if a.variety == NamespaceConstraintNot {
		return notOrAny(differenceNamespaces(a.namespaces, b.namespaces))
	}
	return notOrAny(differenceNamespaces(b.namespaces, a.namespaces))
}

// notOrAny names the set computed by §3.10.6.3 clause 4 or clause 5: an empty
// set excludes nothing, so the union is any (clauses 4.1/5.1) and its
// {namespaces} must be empty (w-props-correct clause 3); a non-empty one is the
// exclusion list of a not constraint (clauses 4.2/5.2, whose non-emptiness is
// what w-props-correct clause 2 demands).
func notOrAny(namespaces []Namespace) (NamespaceConstraintVariety, []Namespace) {
	if len(namespaces) == 0 {
		return NamespaceConstraintAny, nil
	}
	return NamespaceConstraintNot, namespaces
}

// unionDisallowedNames computes the {disallowed names} of the union (§3.10.6.3
// bullets 1-2): a's QName members that b does not allow, followed by b's members
// that a does not allow, in document order (a first, then b). A member the OTHER
// operand ALLOWS is silently DROPPED — the union admits that name through that
// operand, so continuing to disallow it would name a set the union is not.
//
// The test is AllowsName, the canonical cvc-wildcard-name (§3.10.4.2) entry
// point, on the whole expanded name. That is NOT the test cos-aw-intersect's
// corresponding bullets use: they ask AllowsNamespace about the member's
// namespace name alone (§3.10.4.3). The predicates differ in both axis and
// polarity, so intersectDisallowedNames is not reusable here and vice versa.
//
// The seen-free append may leave a cross-operand duplicate — a QName disallowed
// by BOTH operands is not allowed by either, so both loops keep it;
// NewNamespaceConstraint's dedupSlice removes it (STYLE T4).
func unionDisallowedNames(a, b NamespaceConstraint) []QName {
	var out []QName
	for _, name := range a.disallowedNames {
		if !b.AllowsName(name) {
			out = append(out, name)
		}
	}
	for _, name := range b.disallowedNames {
		if !a.AllowsName(name) {
			out = append(out, name)
		}
	}
	return out
}

// unionDisallowedNameKeywords computes the keyword half of the union's
// {disallowed names} (§3.10.6.3 bullet 3): the keyword defined if it is contained
// in BOTH operands' {disallowed names} — a set intersection, NOT the union the
// surrounding operation's name suggests, and NOT the "member of either" that
// cos-aw-intersect (§3.10.6.4) specifies for the intersection operation.
//
// sibling is never propagated because §3.10.6.3's bullet list has no sibling
// bullet — not because no operand can carry one. An element wildcard's constraint
// legally carries sibling and reaches here through coveringWildcardUnion, and the
// keyword is then dropped. Nothing reads the operands' sibling membership; the
// drop widens the union, so it is fail-open wherever it happens.
func unionDisallowedNameKeywords(a, b NamespaceConstraint) []DisallowedNameKeyword {
	if !a.hasDisallowedNameKeyword(DisallowedNameDefined) || !b.hasDisallowedNameKeyword(DisallowedNameDefined) {
		return nil
	}
	return []DisallowedNameKeyword{DisallowedNameDefined}
}
