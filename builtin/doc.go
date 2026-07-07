// Package builtin defines the 49 builtin XSD datatypes — including
// precisionDecimal — as a generated, backend-neutral data table, and
// composes that table with a value.Backend into ready-to-use type
// definitions.
//
// # The generated table (M1)
//
//	type TypeSpec struct {
//	    Name        string       // spec name, verbatim (e.g. "nonNegativeInteger")
//	    Base        string       // base type name (see below)
//	    Variety     Variety      // Atomic{} | List{Item: ...}
//	    Fundamental *Fundamental // ordered/bounded/cardinality/numeric; nil for anyAtomicType
//	    Facets      []Facet      // applicable constraining facets, spec order, with defaults
//	}
//
//	var Types []TypeSpec   // all 49 builtins, spec order
//
// Variety is a sealed sum: Atomic{} or List{Item: ...}, so a list's item
// type cannot exist without list-ness (STYLE T1/T2). Fundamental is a
// pointer: every datatype has all four fundamental facets, except
// anyAtomicType, whose empty {fundamental facets} (§4.1.6) is the nil case —
// no partial mix is representable. The applicable-facet set is exactly the
// names in Facets, so it is read off Facets via TypeSpec.Applies rather than
// stored twice (STYLE D3). Base follows the spec hierarchy (§4.1.6): the 19
// primitives and precisionDecimal derive from anyAtomicType, anyAtomicType
// from anySimpleType, and each list restricts an anonymous list rooted at
// anySimpleType. The closed value types (Variety, Fundamental, Facet) are
// hand-written in typespec.go; the type IsPrimitive helper reports
// Base == "anyAtomicType".
//
// gen_typespec.go is emitted by tools/typespecgen from the per-type
// property subsections in docs/specs/md/xmlschema11-2.md (§3.3/§3.4,
// cross-checked against Appendix F.1) and xsd-precisionDecimal.md §3,
// parsed by tools/hfnextract/builtins. It contains DATA ONLY — no function
// values — and is byte-identical on regeneration. No row is ever
// hand-typed (PRINCIPLES 26).
//
// # Mapping resolution: nearest mapped ancestor
//
// Derived builtins are DATA BY DEFAULT — restrictions of a primitive
// plus facets from the table, inheriting operations — so a minimal
// backend implements only the ~25 primitives (several share a value
// space — the Gregorian types ride one temporal model), and every list
// builtin (NMTOKENS, IDREFS, ENTITIES) is handled generically by the
// engine via its item type.
//
// A backend MAY additionally map derived builtins to give them their
// own, typically narrower, representation. Each builtin's governing
// mapping is resolved by walking UP the base chain to the nearest type
// the backend maps; the primitives are the mandatory floor of that
// walk. A derived mapping governs only the value the application
// receives — inherited facet checks (enumeration, bounds) and
// restriction-validity checks still run in the declaring/base type's
// wider space per the widest-space rule in package value. A lexical
// that passes those checks but cannot be represented by the narrow
// derived mapping is a mapping error on that type, never a false
// validity verdict.
//
// # Seeding
//
//	func Seed(b value.Backend) ([]*xsd.SimpleType, error)
//
// Seed walks Types in order, builds each builtin type definition, and
// attaches each type's governing mapping by the nearest-mapped-ancestor
// resolution above. It errors if b lacks a mapping for a primitive
// (compose with value.Override to fill gaps from another backend). The
// parser seeds its symbol table from the result; xs:anyType and
// xs:anySimpleType are structural and always present.
//
// precisionDecimal is registered always-on: its applicable facet set
// (totalDigits, maxScale, minScale — NOT fractionDigits or the length
// facets) comes from the precisionDecimal spec's applicability list, so
// cos-applicable-facets fires correctly on misuse.
package builtin
