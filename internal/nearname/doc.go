// Package nearname picks the declared name nearest a name that resolves to
// nothing, once for every library package that charges src-resolve
// (§3.17.6.2) clause 1 for a QName naming no component: xsd's finalize
// charges, and parser's two eager ones for a complex type's base= and a <key
// ref=>. Each such charge names the candidate this package picks, so a reader
// who mistyped a name is shown the one they could have meant.
//
// # Contract
//
//	type Name struct{ Space, Local string }
//	func Hint(missing Name, candidates []Name, render func(Name) string) string
//
// Hint is the clause a src-resolve message appends to its "no component with
// that expanded name is present" fact: "; did you mean " + render(c) + "?" for
// the candidate c nearest missing, or "" when none qualifies. render spells c
// as the message spells missing, so the two names read alike.
//
// The caller owns two parts of the rule, through what it passes: candidates
// holds only components of the kind its lookup seeks — those that would have
// satisfied it without its own wrong-kind charge — in that population's slice
// order, never in an order read off a map (STYLE D2). Hint owns the rest:
//
//   - Namespace: a candidate qualifies only when its Space is identical to
//     missing's, the empty (absent) namespace equalling only itself. The zero
//     Name is an anonymous component and never qualifies.
//   - Distance: the Levenshtein distance between the two Locals over runes,
//     case-sensitive, insertion, deletion and substitution each costing 1. A
//     candidate qualifies when that distance d satisfies d <= min(2, max(n,
//     3)/3) under integer division, n being the rune count of missing's
//     Local: d <= 1 for a Local of up to five runes, d <= 2 from six.
//   - Choice: the least distance wins, a tie goes to the earliest candidate,
//     and at most one name is offered.
//
// It depends on nothing but the standard library.
package nearname
