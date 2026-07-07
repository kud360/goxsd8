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
// # Only primitives carry code
//
// Derived builtins are restrictions of a primitive plus facets from the
// table; they inherit the primitive's operations. A backend therefore
// implements mappings for the ~25 primitives only (several share a value
// space — the Gregorian types ride one temporal model), and every list
// builtin (NMTOKENS, IDREFS, ENTITIES) is handled generically by the
// engine via its item type.
//
// # Seeding
//
//	func Seed(b value.Backend) ([]*xsd.SimpleType, error)
//
// Seed walks Types in order, builds each builtin type definition, and
// attaches b's Mapping at the primitives. It errors if b lacks a mapping
// for a primitive the table needs (compose with value.Override to fill
// gaps from another backend). The parser seeds its symbol table from the
// result; xs:anyType and xs:anySimpleType are structural and always
// present.
//
// precisionDecimal is registered always-on: its applicable facet set
// (totalDigits, maxScale, minScale — NOT fractionDigits or the length
// facets) comes from the precisionDecimal spec's applicability list, so
// cos-applicable-facets fires correctly on misuse.
package builtin
