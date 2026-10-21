// Package codegen emits Go source from a compiled schema set:
// deterministic, type-narrowed (planned, see below), and
// allocation-conscious.
//
// # Entry point
//
//	type Target struct { Schema *xsd.Schema; Dir, Package string }
//	func Generate(targets []Target) error
//
// Each target carries one finalized [xsd.Schema] — what parser.Parse
// assembles from a root location and its composition closure; the
// §3.17.1 Schema component is already multi-namespace-capable, so there
// is no separate schema-set type. Each target emits one package into its
// own directory. [Generate] states the phases, the decline and destination
// rules, and the file it writes.
//
// # Coverage
//
// Generate emits, in document order, one Go type per top-level element
// declaration and one struct per anonymous complex type nested in its type
// definition, and declines everything outside this slice:
//
//   - An element whose {type definition} is an atomic simple type gets a
//     named type over a Go builtin, chosen by the type's {primitive type
//     definition} (§3.16.1): string and anyURI give string, boolean gives
//     bool. Facets are not represented, and a named user simple type gets no
//     Go type of its own.
//   - An element whose {type definition} is a complex type, named or
//     anonymous, gets a struct of its own when the type is a restriction of
//     xs:anyType, implicit (§3.4.2.3.2 dcl.ctd.ctcc.implicit) or explicit
//     (<complexContent><restriction base="xs:anyType">, §3.4.2.3.1
//     dcl.ctd.ctcc.explicit), with element-only content (§3.4.2.3.3
//     dcl.ctd.ctcc.common) and no {open content}, {attribute uses},
//     {attribute wildcard}, {assertions} or {abstract}, whose {particle}
//     occurs exactly once and holds a sequence of local element
//     declarations.
//   - Each such particle gives one field, in particle order: a local element
//     with an atomic simple type gets its builtin, one with an anonymous
//     complex type of the same shape gets a struct of its own, emitted after
//     its parent. A struct nested in a named complex type is emitted once,
//     after the first top-level element of that type, and every element of
//     that type shares it. {max occurs} above 1 or unbounded gives a slice
//     whatever {min occurs} is; {min occurs} 0 with {max occurs} 1 gives a
//     pointer.
//   - Every element declaration emitted from has {nillable} and {abstract}
//     false and no {value constraint}, {type table}, {substitution group
//     affiliations} or {identity-constraint definitions}.
//
// GAP(codegen): every other construct declines: choice and all compositors,
// nested model groups, <group ref> and <element ref> particles, wildcards, a
// particle with {max occurs} 0, attributes, mixed, simple and empty content,
// extension, restriction of any base but xs:anyType, a local element with a
// named complex type, xs:anyType, the special simple types, list and union
// types, and every primitive but the three above. No tracking issue owns
// lifting a decline yet; decline in emit.go carries the same marker.
//
// # Determinism (STYLE D1/D2)
//
// Identical schema input produces byte-identical output. No map
// iteration anywhere near the emitter; emission order is document order.
// Golden-file tests pin the output.
//
// # Naming
//
// A single namer component owns every XSD-name → Go-identifier decision.
// Each XSD local name becomes an exported identifier: '_' and every rune Go
// does not admit in an identifier break words, every word starts upper-case,
// and a result with no upper-case first letter is prefixed with
// X. That upper-case start is also what keeps every identifier clear of the
// Go keywords. A top-level element's type is named from the element. An
// anonymous type nested in it is named by the path of names from its nearest
// named ancestor: the top-level element when the element's own type is
// anonymous (element shipTo under element purchaseOrder → PurchaseOrderShipTo),
// the complex type when that type is named (element x in complex type ct → CtX,
// whichever elements have type ct). A residual collision — two
// names folding to one identifier, among a package's types or one struct's
// fields — is disambiguated deterministically by document order: the later
// name takes the smallest free numeric suffix from 2. Every generated
// type's header comment records the element declaration's QName and its
// schema Loc.
//
// # Planned (M9 — not yet implemented)
//
// Shortening an anonymous type's path to only as much of it as uniqueness
// requires. Cross-target type references import across the generated
// packages. The CLI maps its repeated -schema/-out flag pairs onto targets.
//
// Type narrowing in interfaces, the generated-code idiom:
//
//   - An xs:choice becomes a SEALED INTERFACE: an interface with an
//     unexported marker method, one concrete branch type per
//     alternative; consumers type-switch over the branches. This is the
//     closed-sum exception to STYLE T2 — "N pointer fields, exactly one
//     non-nil" never appears in generated code.
//   - Generated readers/views expose the narrowest interface a consumer
//     needs; optionality and nillability are modeled in the types, not
//     in comments.
//   - Simple-content fields are typed by the chosen backend's
//     representation (strict or native, per generation option).
//
// # The Emitter seam (value.Emitter; API frozen in M9; not yet declared here)
//
// Backends contribute specialized decode/encode source for their value
// representations — parsing straight from the reader's byte window into
// the target field, facet checks inlined, no intermediate string, no
// boxed value. A backend without an Emitter falls back to emitting calls
// into codec's runtime path. Emitted fast paths carry the same pipeline
// stage and rule-ID metadata as the runtime path; codec's differential
// tests hold the two to identical behavior.
package codegen
