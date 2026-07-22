package value

import "strings"

// This file adds the list {variety} to the backend-generic pipeline: the
// lexical mapping a list-variety *xsd.SimpleType resolves to (listMapping) and
// the value it produces (listValue). Before this, governingMapping walked only
// the atomic base chain, so ValidateLexical on any list-variety type returned
// "no backend mapping governs type" unconditionally (a cvc-datatype-valid
// error) regardless of instance validity. The single governingMapping list
// branch (facets.go) now wraps the item type's own governing mapping in a
// listMapping, so BOTH the candidate mapping in ValidateLexical AND
// declaringMapping's enum/bound facet-{value} parsing resolve for a
// list-variety type through the same widest-space rule the atomic cohort uses.
//
// The list item type is always atomic or a union-of-atomics — never itself a
// list (Structures §3.16.1, std-item_type_definition: the {item type
// definition} "must not itself be a list type ... or have any basic members
// which are list types") — so listMapping's recursion structurally bottoms out
// at one level.

// Compile-time assertions that listValue satisfies the capability interfaces
// the list pipeline relies on: Lengthed realizes cvc-length-valid's "measured
// in number of list items" unit (§4.3.1.3), and Identical/Eq realize
// cvc-enumeration-valid's "equal or identical" over a list value space
// (§4.3.5.4 + §2.2.1/§2.2.2). enumMatch (facets.go) and lengthFacet.CheckValue
// discover them via these interfaces, never a concrete type, so the assertions
// have real call sites.
var (
	_ Lengthed  = listValue{}
	_ Identical = listValue{}
	_ Eq        = listValue{}
)

// listMapping builds the lexical mapping for a list-variety type from its item
// type's governing mapping, implementing cvc-datatype-valid clause dv_list
// (§4.1.4 cl.2.2): "each space-delimited substring of L is Datatype Valid with
// respect to the {item type definition}", and V is the ordered sequence of the
// values so identified. Parse splits the ALREADY whiteSpace-normalized lexical
// (list's whiteSpace is fixed collapse, §4.3.6.1 f-w-fixed, applied upstream by
// ValidateLexical's whiteSpace stage before Parse runs) on whitespace via
// strings.Fields, then parses each token against the item mapping. A token's
// own Parse failure — the item is itself Datatype-Valid against the item type
// (dv_list clause 2.2) — is already the right cvc-datatype-valid-family error,
// so it propagates unchanged with no rewrap.
//
// Canonical is deliberately nil: no current cohort needs a canonical list form,
// and per the Mapping doc a nil Canonical means "this whole type has no
// canonical form", which callers must treat as such rather than an error.
func listMapping(item Mapping) Mapping {
	return Mapping{
		Parse: func(lexical string, ctx Context) (Value, error) {
			tokens := strings.Fields(lexical)
			items := make([]Value, 0, len(tokens))
			for _, tok := range tokens {
				v, err := item.Parse(tok, ctx)
				if err != nil {
					return nil, err
				}
				items = append(items, v)
			}
			return listValue{items: items}, nil
		},
	}
}

// listValue is a list-variety value: the ordered sequence of item values
// produced by listMapping.Parse (§4.1.4 cl.2.2). Its capabilities realize the
// list-applicable facets — length in items (§4.3.1.3) and enumeration by
// value-space "equal or identical" (§4.3.5.4) — over that sequence.
type listValue struct {
	items []Value
}

// Len returns the number of list items, the unit cvc-length-valid (§4.3.1.3),
// cvc-minLength-valid (§4.3.2.3) and cvc-maxLength-valid (§4.3.3.3) measure for
// a datatype ·constructed· by ·list· (dt-length: "length is measured in number
// of list items"). lengthFacet.CheckValue reads it through the Lengthed
// capability, so the list case needs no length-facet code of its own.
func (l listValue) Len() int { return len(l.items) }

// Identical reports the §2.2.1 identity relation over list values: two lists are
// identical iff they have equal length and every item is pairwise Identical to
// its counterpart, using each item's OWN identity relation. cvc-enumeration-valid
// (§4.3.5.4) accepts a candidate "equal or identical" to a member, and enumMatch
// (facets.go) prefers this identity relation. An item lacking the Identical
// capability yields false for that comparison — a normal "no match" outcome,
// mirroring enumMatch's "a candidate with neither capability matches nothing"
// convention, never a panic (that would be a schema-construction claim, which
// this is not).
//
// SCOPE: the §2.2.2 wrinkle that a length-1 list is equal to its bare atomic
// member is deliberately NOT implemented — no current fixture needs it, and
// PRINCIPLES 26 forbids speculative code. A cross-type argument (not a
// listValue) is not identical.
func (l listValue) Identical(other Value) bool {
	o, ok := other.(listValue)
	if !ok || len(l.items) != len(o.items) {
		return false
	}
	for i, item := range l.items {
		id, ok := item.(Identical)
		if !ok || !id.Identical(o.items[i]) {
			return false
		}
	}
	return true
}

// Eq reports the §2.2.2 equality relation over list values: two lists are equal
// iff they have equal length and every item is pairwise Eq to its counterpart,
// using each item's OWN equality relation. It unions with Identical in enumMatch
// so an equal-but-not-identical member still matches (§4.3.5.4). An item lacking
// the Eq capability yields false for that comparison (the Identical convention),
// never a panic. The §2.2.2 length-1/bare-atomic wrinkle is out of scope here
// too (see Identical). A cross-type argument is not equal.
func (l listValue) Eq(other Value) bool {
	o, ok := other.(listValue)
	if !ok || len(l.items) != len(o.items) {
		return false
	}
	for i, item := range l.items {
		eq, ok := item.(Eq)
		if !ok || !eq.Eq(o.items[i]) {
			return false
		}
	}
	return true
}
