// Package builtin defines the 49 builtin XSD datatypes — including
// precisionDecimal — as a generated, backend-neutral data table, and
// composes that table with a value.Backend into ready-to-use type
// definitions.
//
// # The generated table (M1)
//
//	type TypeSpec struct {
//	    Name         string   // spec name, verbatim (e.g. "nonNegativeInteger")
//	    Base         string   // base type name; primitives derive from anySimpleType
//	    Variety      Variety  // atomic | list
//	    Fundamental  Fundamental // ordered/bounded/cardinality/numeric
//	    Facets       []Facet  // constraining facets with spec defaults, in spec order
//	    Applicable   FacetSet // which constraining facets may be applied
//	}
//
//	var Types []TypeSpec   // all 49 builtins, spec order
//
// gen_typespec.go is emitted by the hfn generator (tools/hfnextract and
// its M1 generator) from the Appendix E function definitions and per-type
// property tables in docs/specs/md/xmlschema11-2.md and
// xsd-precisionDecimal.md. It contains DATA ONLY — no function values —
// and is byte-identical on regeneration. No row is ever hand-typed
// (PRINCIPLES 26).
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
