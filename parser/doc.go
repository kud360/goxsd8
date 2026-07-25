// Package parser compiles XSD schema documents into the xsd component
// model. It is deterministic, multi-phase, and cycle-check-free by
// construction.
//
// # Three phases (STYLE D4)
//
//	phase 1  parse    — each schema document → raw form via parser/xmltree;
//	                    every raw node keeps its Loc, so every later error
//	                    can cite file:line:column.
//	phase 2  resolve  — schema composition (include ships; import /
//	                    redefine / override do not yet) through ONE
//	                    loader.Resolver, with chameleon namespace
//	                    coercion, and QName reference resolution through
//	                    a symbol table seeded once per assembly with the
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
// The phase structure makes cycle REJECTION unnecessary at traversal
// time — no traversal carries a `seen` set in order to detect a
// forbidden circularity. The one exception is where the spec itself
// declares the cycle legal: attribute-group references may form cycles
// (§3.6.2.1), and the transitive-closure fold of {attribute uses}
// carries a visited set purely to bound the walk and avoid re-descending
// a group already folded in — it rejects nothing. Likewise the
// composition index keyed by resolved location is DOCUMENT IDENTITY, not
// a cycle guard: §4.2.3 makes two xs:includes of the same resolved
// location the same schema document, and declares include cycles legal,
// so each distinct document is loaded once and nothing is rejected.
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
// The assembled schema is one [xsd.Schema] — the §3.17.1 Schema
// component, which is already multi-namespace-capable, since every
// component carries its own namespace in its expanded name. There is no
// separate "schema set" type.
//
// # Current coverage
//
//	func Parse(location string, opts ...Option) (*xsd.Schema, error)
//	    Options: WithResolver(loader.Resolver) (default loader.Dir(".")),
//	    WithBackend(value.Backend) (default builtin/strict),
//	    WithLogger(*slog.Logger) (default silent).
//
// Parse assembles the <xs:include> closure of the root document
// (§4.2.3), including chameleon coercion of a no-targetNamespace
// document into the including namespace (§F.1) — both the components it
// declares and the unqualified QName references inside it. Documents are
// loaded once, keyed by resolved location, so a repeated include, a
// diamond, or a (spec-legal) cycle contributes its components once.
//
// [Produce] remains the single-document entry point: it maps one
// already-read document and follows no inter-document reference at all.
//
// # Planned composition (not yet implemented)
//
//   - xs:import, xs:redefine and xs:override are skipped, not followed:
//     they are top-level representations this slice does not yet produce
//     (§3.1.2), so a schema needing them assembles short.
//   - xs:override will track its target document explicitly: components
//     declared inside an override belong to the OVERRIDDEN document
//     (its schema-level defaults apply), and suppression of replaced
//     components must never leak back into the overriding document under
//     mutual/circular overrides (PRINCIPLES 16).
//   - Assembling several root locations into one schema awaits a
//     consumer; nothing in the CLI or validator needs it yet, so no
//     multi-root entry point is exported (STYLE T5).
//
// # Planned instance-hint reader (not yet implemented)
//
//	func SchemaLocationHints(instance io.Reader) ([]Hint, error)
//	    The xsi:schemaLocation reader shared by the CLI, the validator,
//	    and the conformance harness — one implementation, resolved
//	    relative to the instance location, routed through the same
//	    Resolver as root schemas.
//
// Schema-validity violations are *xsderr.Error values carrying src-*/
// cos-*/derivation-ok-* rules. PLANNED (not yet implemented): collecting
// them in document order rather than stopping at the first — [Parse] and
// [Produce] both return only the first error today.
package parser
