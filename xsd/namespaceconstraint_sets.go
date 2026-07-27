package xsd

// This file holds the plain set operations over a Namespace Constraint's
// {namespaces} slice — intersection, union, and difference of two
// document-ordered, already-deduplicated []Namespace. They belong to no single
// §3.10.6 operator: cos-aw-intersect (§3.10.6.4,
// namespaceconstraint_intersect.go) and cos-aw-union (§3.10.6.3,
// namespaceconstraint_union.go) each reach for all three, in different
// combinations, so neither operator's file owns them (STYLE T4 — one encoding of
// "set intersection", not one per caller).
//
// None of the three interprets {variety}: they are set algebra over the member
// slices only, and the variety/namespaces tables that call them are what give a
// result its meaning. Each uses a map purely as a membership test and takes its
// output order from an operand's document order, so no map iteration order ever
// reaches a result (STYLE D2/D3).

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
