// Package parser compiles XSD schema documents into the xsd component
// model. It is deterministic, multi-phase, and cycle-check-free by
// construction.
//
// # Three phases (STYLE D4)
//
//	phase 1  parse    — each schema document → raw form via parser/xmltree;
//	                    every raw node keeps its Loc, so every later error
//	                    can cite file:line:column.
//	phase 2  resolve  — schema composition (include, import and override
//	                    ship; redefine does not yet) through ONE
//	                    loader.Resolver, with chameleon namespace
//	                    coercion on include and override only, override
//	                    pre-processing carried as data alongside the
//	                    effective namespace (parser/override.go), and
//	                    QName reference resolution through a symbol table
//	                    seeded once per assembly with the builtins
//	                    (builtin.Seed).
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
// composition index keyed by resolved location AND the namespace the
// document was reached under AND the override applied to it is DOCUMENT
// IDENTITY, not a cycle guard:
// §4.2.3 makes two xs:includes of the same resolved location the same
// schema document and declares include cycles legal, §4.2.6.2 says as
// much for repeated xs:imports, §4.2.5 says as much for equivalent
// xs:overrides (and requires the processor to recognize that closure has
// been reached, which is exactly what the index does), the namespace is
// part of the key because one document reached as a chameleon include
// and as an import yields two different component sets, and the override
// is part of it because one document overridden two different ways
// likewise does. Each distinct reading is loaded once and nothing is
// rejected.
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
// Parse assembles the <xs:include>, <xs:override> and <xs:import>
// closure of the root document (§4.2.3, §4.2.5, §4.2.6.2), including
// chameleon coercion of a
// no-targetNamespace <xs:include>d document into the including namespace
// (§F.1) — both the components it declares and the unqualified QName
// references inside it. An <xs:import>ed document is never coerced: it
// contributes its components in its OWN target namespace (absent
// included), which is what makes a cross-namespace reference resolve.
// An <xs:override>d document IS coerced, on the same terms as an
// <xs:include>d one (§4.2.5 clause 2.3): before the override is applied,
// the overridden document's identically-named source declarations are
// substituted by the override's children (§F.2), and the substitution
// cascades into every document that one <xs:include>s or <xs:override>s
// — §4.2.5's ·target set· — while stopping at <xs:import>. A substituted
// declaration is produced by the OVERRIDDEN document's producer, so the
// overridden document's target namespace and schema-level defaults apply
// to it (§4.2.5's document-level-defaults note, PRINCIPLES 16).
// Documents are loaded once, keyed by resolved location, the
// namespace they were reached under and the override applied to them, so
// a repeated include, import or equivalent override, a
// diamond, or a (spec-legal) cycle contributes its components once.
//
// [Produce] remains the single-document entry point: it maps one
// already-read document and follows no inter-document reference at all.
//
// # Planned composition (not yet implemented)
//
//   - GAP(xsd): xs:redefine is skipped, not followed: it is a top-level
//     representation this slice does not yet produce (§3.1.2), so a
//     schema needing it assembles short. Passing WithLogger is how that
//     is observed: every skipped child <xs:redefine> element is reported
//     at debug level with its location. That also empties §F.2 clause
//     1's "or <redefine>" scope: an <xs:override> substitutes only for
//     the <schema> children of the documents in its ·target set·, never
//     for a <redefine> child, because no <redefine> is read at all.
//   - GAP(xsd): src-resolve clause 4 (cl.qnr.nsdeclared, §3.17.6.2) is
//     not enforced: a QName reference into a namespace the containing
//     document never <xs:import>ed still resolves if some other document
//     of the assembly contributed that namespace. That direction
//     under-rejects (never false-rejects a valid schema), so it is a
//     recorded gap rather than a correctness hazard.
//   - GAP(xsd): §5.3 (Missing Sub-components) is never reported as such.
//     A namespace an <xs:import> declares but no document of the
//     assembly supplies — a bare import, or one whose schemaLocation
//     does not resolve — leaves that namespace's components genuinely
//     missing, and the only way that surfaces is an actual QName
//     reference into it failing src-resolve at finalize (which this
//     package hard-fails, see [xsd.SchemaBuilder.Finalize]); an assembly
//     that makes no such reference is accepted. Under-rejects in the
//     same direction as the clause 4 gap above.
//     That hard-fail is itself the deviation from §5.3, which makes an
//     unresolved reference an ·absent· value and defers the consequence
//     to ·assessment·, never rejecting the schema. One slot is already
//     aligned: a {substitution group affiliations} member naming no
//     declaration is retained as ·absent· rather than rejected (see
//     xsd/resolve.go's resolveElementDecl). The remaining slots — {type
//     definition}, <element ref>, <attribute ref>, <group ref>, keyref —
//     are #434, which must also supply the ·lax assessment· fallback
//     §5.3 requires on the validation side.
//   - GAP(xsd): two DISTINCT <xs:override> elements whose children are
//     textually equivalent are treated as two different overrides of the
//     same document, so overriding one document the same way down two
//     paths yields duplicate components and a sch-props-correct clause 2
//     rejection where §4.2.5's note ("multiple equivalent overrides of
//     the same schema document will not constitute a violation") wants
//     none. Override identity here is the ordered list of substituted
//     (element type, name, source location) triples, not the fn:deep-equal
//     comparison §4.2.5 offers as one way of detecting closure; the SAME
//     <xs:override> element reached twice is recognized, which is what
//     terminates every cycle. This over-rejects, so it can lose a valid
//     assembly, never accept an invalid one.
//   - GAP(xsd): two children of ONE <xs:override> with the same element
//     type and name are reported under src-override. §F.2's stylesheet is
//     the normative statement of the transformation (the prose beside it
//     is explicitly non-normative) and its clause 1 template selects
//     ($replacement, $original)[1], so first match wins and the case is
//     spec-defined; rejecting it is a deliberate choice not to discard one
//     of two conflicting declarations silently. Over-rejects in the same
//     direction as the entry above.
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
// cos-*/derivation-ok-* rules — plus cvc-datatype-valid where a schema
// document attribute is simply not valid against the type the schema for
// schema documents declares for it (e.g. an unrecognized ## token in
// notQName, §3.10.2), which no Schema Representation Constraint covers.
// PLANNED (not yet implemented): collecting
// them in document order rather than stopping at the first — [Parse] and
// [Produce] both return only the first error today.
package parser
