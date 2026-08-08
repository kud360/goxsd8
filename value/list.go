package value

import (
	"strings"

	"github.com/kud360/goxsd8/xsd"
)

// This file adds the list {variety} to the backend-generic pipeline: the
// lexical mapping a list-variety *xsd.SimpleType resolves to (listMapping) and
// the value it produces (listValue). Before this, governingMapping walked only
// the atomic base chain, so ValidateLexical on any list-variety type returned
// "no backend mapping governs type" unconditionally (a cvc-datatype-valid
// error) regardless of instance validity. The single governingMapping list
// branch (facets.go) now wraps the item TYPE in a listMapping, so BOTH the
// candidate mapping in ValidateLexical AND declaringFacetSpace's enum/bound
// facet-{value} parsing resolve for a list-variety type.
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
// TYPE, implementing cvc-datatype-valid clause dv_list (§4.1.4 cl.2.2): "each
// space-delimited substring of L is Datatype Valid with respect to the {item
// type definition}", and V is the ordered sequence of the values so identified.
// Parse splits the ALREADY whiteSpace-normalized lexical (list's whiteSpace is
// fixed collapse, §4.3.6.1 f-w-fixed, applied upstream by ValidateLexical's
// whiteSpace stage before Parse runs) on whitespace via strings.Fields, then
// decides each token against the item type.
//
// Each token recurses through validateLexical — the FULL cvc-datatype-valid
// rule against the item type — not through the item type's governing Mapping
// alone. dv_list says "Datatype Valid with respect to the {item type
// definition}", which is the whole rule: clause 1 (dv_pattern, the item type's
// own ·lexical· facets), clause 2.1 (dv_atomic, the lexical mapping of the item
// type's {primitive type definition}) AND clause 3 (dv_vfacets, the item type's
// OWN ·value-based· facets). For an item type that IS its own {primitive type
// definition} (boolean, float, anyURI, …) those coincide, so the item mapping
// alone sufficed; for a DERIVED item type they do not. xs:byte's {primitive
// type definition} is xs:decimal, so the mapping accepts every decimal literal
// while byte's own effective facets — pattern [\-+]?[0-9]+, fractionDigits=0,
// minInclusive=-128 (byte.minInclusive), maxInclusive=127 (byte.maxInclusive) —
// are what reject "128", "-129" and "1.5". Parsing a token through the item
// mapping only would FALSE-ACCEPT all three (issue #224); recursing runs the
// same facet pipeline a standalone xs:byte value takes, so the two paths cannot
// disagree. The same holds one sub-clause over for a UNION item type (cl.2.3,
// dv_union): the recursion layers the union's OWN pattern (clause 1) and
// enumeration (clause 3) — with assertions the only facets §4.1.5
// cos-applicable-facets admits on a union — around the member dispatch, where
// parsing a token through the item's mapping alone (unionMapping, which is that
// dispatch and nothing else) would apply neither (issue #326). The recursion
// terminates structurally: neither the item type nor any
// basic member of it is itself a list (§3.16.1, std-item_type_definition), so
// validateLexical never re-enters this Parse.
//
// A token's rejection — whether from its Parse or from one of the item type's
// facets — is already the right cvc-datatype-valid-family error (cvc-pattern-valid,
// cvc-minInclusive-valid, cvc-maxInclusive-valid, cvc-fractionDigits-valid, …),
// so it propagates unchanged with no rewrap. That unchanged propagation is also
// what carries a facet-pipeline PRECONDITION fault in the ITEM type out to the
// caller still discriminable (IsFacetPrecondition, ValidateLexical): no rewrap means
// no rule re-attribution, and the caller deciding validity — which is not this
// mapping — is the one that must tell the fault from a rejection.
//
// Canonical is deliberately nil: no current cohort needs a canonical list form,
// and per the Mapping doc a nil Canonical means "this whole type has no
// canonical form", which callers must treat as such rather than an error.
func listMapping(b Backend, item *xsd.SimpleType) Mapping {
	return Mapping{
		Parse: func(lexical string, ctx Context) (Value, error) {
			// This split is the only point at which the ITEM type's whiteSpace mode
			// could have mattered, and no mode can change a token's verdict. The
			// recursion below does re-enter validateLexical's whiteSpace stage with
			// the item type's own mode, but whiteSpace is a ·pre-lexical· facet and
			// the dv_vfacets note (§4.1.4) says outright that "whiteSpace facets and
			// other ·pre-lexical· facets do not take part in checking Datatype
			// Valid": pre-lexical facets normalize the infoset value BEFORE datatype
			// validity is checked, and the one doing so here is the LIST's own
			// whiteSpace — fixed to collapse (§4.3.6.1 f-w-fixed, §3.16.2.1 case 3)
			// and already applied upstream by validateLexical before this Parse. So
			// no test pins a preserve/replace/collapse item type (issue #326): the
			// exclusion is structural, not merely an artifact of Fields happening to
			// leave a token with no whitespace for a mode to act on.
			tokens := strings.Fields(lexical)
			items := make([]Value, 0, len(tokens))
			for _, tok := range tokens {
				v, _, err := validateLexical(b, item, tok, ctx)
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
