package builtin

// This file holds the hand-written, closed value types that gen_typespec.go
// (generated, data-only) refers to. Keeping the types here and the data there
// lets the generated file stay pure data — no logic, no function values — per
// PRINCIPLES 26 and builtin/doc.go.

// Variety is a builtin datatype's {variety} (§4.1.1). Only atomic and list
// occur among the builtins; no builtin has union variety. The zero value is
// invalid so an unset Variety is caught rather than silently treated as
// atomic.
type Variety uint8

// The builtin varieties. anySimpleType's absent variety has no row (it is
// excluded from the table), so no "absent" member is needed here.
const (
	VarietyAtomic Variety = iota + 1
	VarietyList
)

// Ordered is the value of the {ordered} fundamental facet (§4.2.1). The zero
// value OrderedAbsent is what anyAtomicType carries, whose {fundamental
// facets} is the empty set (§4.1.6 anyAtomicType-def).
type Ordered uint8

// The {ordered} values.
const (
	OrderedAbsent Ordered = iota
	OrderedFalse
	OrderedPartial
	OrderedTotal
)

// Bounded is the value of the {bounded} fundamental facet (§4.2.2). Zero is
// absent (anyAtomicType).
type Bounded uint8

// The {bounded} values.
const (
	BoundedAbsent Bounded = iota
	BoundedFalse
	BoundedTrue
)

// Cardinality is the value of the {cardinality} fundamental facet (§4.2.3).
// Zero is absent (anyAtomicType).
type Cardinality uint8

// The {cardinality} values.
const (
	CardinalityAbsent Cardinality = iota
	CardinalityFinite
	CardinalityCountablyInfinite
)

// Numeric is the value of the {numeric} fundamental facet (§4.2.4). Zero is
// absent (anyAtomicType).
type Numeric uint8

// The {numeric} values.
const (
	NumericAbsent Numeric = iota
	NumericFalse
	NumericTrue
)

// Fundamental groups a datatype's four fundamental facets (§4.2). For
// anyAtomicType every field is its Absent zero value.
type Fundamental struct {
	Ordered     Ordered
	Bounded     Bounded
	Cardinality Cardinality
	Numeric     Numeric
}

// FacetName is a constraining-facet name (§4.3), spelled verbatim as in the
// spec (e.g. "minInclusive", "explicitTimezone", "maxScale").
type FacetName string

// Facet is one constraining facet that applies to a datatype, together with
// the spec-given default the datatype fixes or seeds it with.
type Facet struct {
	// Name is the facet name (§4.3).
	Name FacetName
	// Default is the value the spec gives the facet on this datatype, or ""
	// if the spec lists the facet as applicable without a value.
	Default string
	// Fixed reports that Default must not be changed by a restricting
	// derivation ("fixed" in the spec's Facets subsections).
	Fixed bool
}

// TypeSpec is the backend-neutral description of one builtin datatype: its
// name and base, variety (and item type for lists), fundamental facets, and
// the constraining facets that apply to it with their spec defaults. It is
// data only; all rows live in gen_typespec.go.
type TypeSpec struct {
	// Name is the spec datatype name, verbatim (e.g. "nonNegativeInteger").
	Name string
	// Base is the name of the {base type definition}. Primitives derive from
	// anyAtomicType (§4.1.6 dummy-def); anyAtomicType derives from
	// anySimpleType; lists restrict an anonymous list rooted at anySimpleType.
	Base string
	// Variety is atomic or list.
	Variety Variety
	// Item is the list item type name; "" unless Variety is VarietyList.
	Item string
	// Fundamental holds the four fundamental facets (§4.2).
	Fundamental Fundamental
	// Facets are the applicable constraining facets in spec order, each with
	// its spec default. The applicable-facet set is exactly the names here;
	// it is not stored separately (STYLE D3).
	Facets []Facet
}

// IsPrimitive reports whether t is a primitive datatype — one whose {base
// type definition} is anyAtomicType (§4.1.6 dummy-def). The 19 §3.3
// primitives and precisionDecimal are primitive; anyAtomicType and the §3.4
// ordinary datatypes are not.
func (t TypeSpec) IsPrimitive() bool { return t.Base == "anyAtomicType" }

// Applies reports whether constraining facet f may be applied to t
// (cos-applicable-facets). The applicable set is exactly the facets listed
// for t, so this reads it off t.Facets rather than a redundant stored set.
func (t TypeSpec) Applies(f FacetName) bool {
	for i := range t.Facets {
		if t.Facets[i].Name == f {
			return true
		}
	}
	return false
}
