// Package codegen emits Go source from a compiled schema set:
// deterministic, type-narrowed, and allocation-conscious.
//
// # Determinism (STYLE D1/D2)
//
// Identical schema input produces byte-identical output. No map
// iteration anywhere near the emitter; emission order is document order.
// Golden-file tests pin the output.
//
// # Type narrowing in interfaces (the generated-code idiom)
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
// # Naming
//
// A single namer component owns every XSD-name → Go-identifier decision.
// Anonymous types are named from the nearest named ancestor (element
// shipTo under element purchaseOrder → PurchaseOrderShipTo), extending
// the path only as far as uniqueness requires; residual collisions
// (case folding, Go keywords, XML-legal-but-Go-illegal characters) are
// disambiguated deterministically by document order. Every generated
// type's header comment records its schema Loc and original QName.
//
// # Multiple schemas, multiple output dirs
//
//	type Target struct { Set *xsd.SchemaSet; Dir, Package string }
//	func Generate(targets []Target, opts ...Option) error
//
// Each target emits one package into its own directory; cross-target
// type references import across the generated packages. The CLI maps
// its repeated -schema/-out flag pairs onto targets.
//
// # The Emitter seam (value.Emitter; API frozen in M9)
//
// Backends contribute specialized decode/encode source for their value
// representations — parsing straight from the reader's byte window into
// the target field, facet checks inlined, no intermediate string, no
// boxed value. A backend without an Emitter falls back to emitting calls
// into codec's runtime path. Emitted fast paths carry the same pipeline
// stage and rule-ID metadata as the runtime path; codec's differential
// tests hold the two to identical behavior.
package codegen
