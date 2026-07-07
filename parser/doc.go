// Package parser compiles XSD schema documents into the xsd component
// model. It is deterministic, multi-phase, and cycle-check-free by
// construction.
//
// # Three phases (STYLE D4)
//
//	phase 1  parse    — each schema document → raw form via parser/xmltree;
//	                    every raw node keeps its Loc, so every later error
//	                    can cite file:line:column.
//	phase 2  resolve  — schema composition (include / import / redefine /
//	                    override, chameleon namespace coercion) through
//	                    ONE loader.Resolver, and QName reference
//	                    resolution through a symbol table seeded with the
//	                    builtins (builtin.Seed).
//	phase 3  finalize — components completed in dependency order: a
//	                    component's base/item/member types are finished
//	                    before it is. Spec-forbidden circularities
//	                    (circular unions, circular groups, circular
//	                    substitution groups) are rejected HERE, once, with
//	                    their named src-/cos- rule and location. UPA,
//	                    particle-restriction, and EDC checks run against
//	                    the finalized shape.
//
// No traversal anywhere in the parser carries a `seen` set; the phase
// structure makes cycles impossible at traversal time.
//
// # Determinism
//
// All child collections are built as slices in document order; symbol
// tables are internal indexes only (STYLE D2). Parsing the same schema
// set produces an identical model, identical error list, identical
// order.
//
// # Composition
//
//   - Multiple root schemas compile into one set; the loader dedupes by
//     resolved location.
//   - xs:override tracks its target document explicitly: components
//     declared inside an override belong to the OVERRIDDEN document
//     (its schema-level defaults apply), and suppression of replaced
//     components never leaks back into the overriding document under
//     mutual/circular overrides (PRINCIPLES 16).
//
// # Contract (implemented across M4)
//
//	func Parse(location string, opts ...Option) (*xsd.SchemaSet, error)
//	func ParseMultiple(locations []string, opts ...Option) (*xsd.SchemaSet, error)
//	    Options: WithResolver(loader.Resolver), WithBackend(value.Backend)
//	    (default strict), WithLogger(*slog.Logger).
//
//	func SchemaLocationHints(instance io.Reader) ([]Hint, error)
//	    The xsi:schemaLocation reader shared by the CLI, the validator,
//	    and the conformance harness — one implementation, resolved
//	    relative to the instance location, routed through the same
//	    Resolver as root schemas.
//
// Schema-validity violations are *xsderr.Error values carrying src-*/
// cos-*/derivation-ok-* rules; the parser collects them in document
// order rather than stopping at the first.
package parser
