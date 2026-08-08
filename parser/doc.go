// Package parser compiles XSD schema documents into the xsd component
// model. It is deterministic, multi-phase, and cycle-check-free by
// construction.
//
// # Three phases (STYLE D4)
//
//	phase 1  parse    — each schema document → raw form via parser/xmltree;
//	                    every raw node keeps its Loc, so every later error
//	                    can cite file:line:column.
//	phase 2  resolve  — schema composition (include, import, override
//	                    and redefine) through ONE
//	                    loader.Resolver, with chameleon namespace
//	                    coercion on include, override and redefine only,
//	                    override pre-processing and redefinition both
//	                    carried as data alongside the effective namespace
//	                    (parser/override.go, parser/redefine.go), and
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
// document was reached under AND the override applied to it AND the
// redefinition applied to it is DOCUMENT
// IDENTITY, not a cycle guard:
// §4.2.3 makes two xs:includes of the same resolved location the same
// schema document and declares include cycles legal, §4.2.6.2 says as
// much for repeated xs:imports, §4.2.5 says as much for equivalent
// xs:overrides (and requires the processor to recognize that closure has
// been reached, which is exactly what the index does), §4.2.4's own note
// asks the same of "multiple equivalent xs:redefineing of the same
// schema document", the namespace is
// part of the key because one document reached as a chameleon include
// and as an import yields two different component sets, and the override
// and the redefinition are part of it because one document overridden —
// or redefined — two different ways likewise does. Each distinct reading
// is loaded once and nothing is rejected.
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
//	func ParseReport(location string, opts ...Option) (*xsd.Schema,
//	    *AssemblyReport, error)
//	    Options: WithResolver(loader.Resolver) (default loader.Dir(".")),
//	    WithBackend(value.Backend) (default builtin/strict),
//	    WithLogger(*slog.Logger) (default silent).
//
// ParseReport is the entry point and Parse the wrapper that drops its
// report. The [AssemblyReport] answers what the assembled
// [xsd.Schema] cannot — which schema documents went into it, in
// discovery order, and which ·inter-schema-document references· could
// not be followed to one — so a consumer reasoning about the DOCUMENT
// SET (§4.2.1's schema(D)) does not have to re-walk §4.2's composition
// edges itself. It is populated even when an error is returned.
//
// Parse assembles the <xs:include>, <xs:override>, <xs:redefine> and
// <xs:import>
// closure of the root document (§4.2.3, §4.2.5, §4.2.4, §4.2.6.2),
// including chameleon coercion of a
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
// An <xs:redefine>d document is coerced on the same terms too (§4.2.4
// clause 3.3), and is the one composition kind that BOTH includes and
// subtracts: the redefined document contributes every component it
// declares EXCEPT those the <redefine> names (clause 4.1.2), while the
// <redefine>'s own children contribute the replacements (clause 4.1.1)
// as definitions of the REDEFINING document. A redefining
// simpleType/complexType is paired with a hidden {name}-·absent· copy of
// the definition it replaces, which is what its own base= resolves to;
// a redefining group/attributeGroup self-reference resolves to the
// original the same way (src-expredef, parser/redefine.go). §4.2.4
// marks the whole mechanism ·deprecated·. A non-empty <xs:redefine>
// whose schemaLocation does not resolve is an ERROR (src-redefine clause
// 1) where an <xs:include>'s is explicitly not one (src-include clause
// 2.4). Documents are loaded once, keyed by resolved location, the
// namespace they were reached under, the override applied to them and
// the redefinition applied to them, so
// a repeated include, import, equivalent override or equivalent
// redefinition, a
// diamond, or a (spec-legal) cycle contributes its components once.
// Loading once suppresses the second COMPOSITION only, which is all
// §4.2.6.2's note asks for: a repeated <xs:import> is still judged
// against src-import clause 3, so one whose namespace disagrees with the
// already-loaded document's targetNamespace is rejected however that
// document was first reached.
//
// What the assembly contributes is not what a reference may reach for. A
// QName in a schema document resolves only into a namespace THAT document
// licenses (§3.17.6.2 src-resolve clause 4, cl.qnr.nsdeclared): its own
// effective (post-§F.1) target namespace, a namespace one of its OWN
// <xs:import> children names, or the XSD or XSI namespace — the last two
// with no <xs:import> at all. An unqualified reference left in the
// ·absent· namespace is licensed only by a document that declares no
// targetNamespace or carries a bare <xs:import> (clause 4.1). The license
// is per schema DOCUMENT, never assembly-wide: a namespace some sibling
// document of the assembly imported licenses nothing here, which is
// exactly what §4.2.6.1 states — "if references to components in a given
// namespace N appear in a schema document S, then S must contain an
// <import> element importing N". Such a reference is rejected as
// src-resolve at the reference itself, and deliberately not treated as a
// §5.3 missing sub-component, which §4.2.6.1 also says in as many words.
//
// [Produce] remains the single-document entry point: it maps one
// already-read document and follows no inter-document reference at all —
// it does READ that document's <xs:import> children, since clause 4
// licenses its references from them, but dereferences none of them.
//
// # Composition gaps
//
//   - GAP(xsd): a redefining <xs:complexType> is declined, not
//     produced. src-expredef clause 1.1 pairs it with a {name}-·absent·
//     copy of the type it replaces as its {base type definition}, and
//     [xsd.ComplexType] carries {base type definition} as a
//     pre-resolution QName reference, so the pairing has nowhere to
//     live: the only representable form would name the redefinition
//     itself as its own base, which Finalize (rightly, for what it would
//     be handed) rejects under ct-props-correct clause 3. The decline is
//     a plain "not yet produced" error, never a fabricated rule
//     violation, and it OVER-rejects — a valid redefining complex type
//     is refused, never accepted wrongly. Closing it needs an
//     xsd-package change (a complex type able to hold an anonymous
//     resolved base), which is library surface #286 did not open; it
//     needs a follow-up issue of its own. Redefining simpleType, group
//     and attributeGroup are produced in full.
//   - GAP(xsd): src-redefine clauses 6.2.2 and 7.2.2 — the
//     no-self-reference branches — are fail-open. 6.2.2 asks whether the
//     redefining group's {model group} accepts a SUBSET of the element
//     sequences the original accepts, a language-containment question
//     needing the content-model engine cos-content-act-restrict (#263)
//     needs; 7.2.2 asks whether the redefining attribute group satisfies
//     clause 3 of derivation-ok-restriction (§3.4.6.3) against the
//     original. Their 6.2.1/7.2.1 halves ARE charged, as src-expredef's
//     closing requirement. Both UNDER-reject: a redefinition that widens
//     rather than restricts is accepted, never wrongly refused. Owned by
//     #286.
//   - GAP(parser): a redefining declaration that an <xs:override>
//     substitutes for under §F.2 clause 1 loses its self-reference
//     resolution — see parser/override.go's GAP for the mechanism.
//     Over-rejects.
//   - GAP(xsd): §5.3 (Missing Sub-components) is never reported as such.
//     A namespace an <xs:import> declares but no document of the
//     assembly supplies — a bare import, or one whose schemaLocation
//     does not resolve — leaves that namespace's components genuinely
//     missing, and the only way that surfaces is an actual QName
//     reference into it failing src-resolve at finalize (which this
//     package hard-fails, see [xsd.SchemaBuilder.Finalize]); an assembly
//     that makes no such reference is accepted, which under-rejects.
//     The src-resolve clause 4 licensing above does NOT close this gap
//     and cannot: clause 4 judges whether the document ASKED for the
//     namespace, §5.3 what follows when a namespace it did ask for
//     supplies no such component — §4.2.6.1 holds the two apart in as
//     many words ("references … not imported by that schema document …
//     are not handled as if they referred to missing components").
//     That hard-fail is itself the deviation from §5.3, which makes an
//     unresolved reference an ·absent· value and defers the consequence
//     to ·assessment·, never rejecting the schema. One slot is already
//     aligned: a {substitution group affiliations} member naming no
//     declaration is retained as ·absent· rather than rejected (see
//     xsd/resolve.go's resolveElementDecl). The remaining slots — {type
//     definition}, <element ref>, <attribute ref>, <group ref>, keyref —
//     are #434, which must also supply the ·lax assessment· fallback
//     §5.3 requires on the validation side.
//   - GAP(xsd): two DISTINCT <xs:redefine> elements whose children are
//     textually equivalent are treated as two different redefinitions of
//     the same document, so redefining one document the same way down two
//     paths yields duplicate components and a sch-props-correct clause 2
//     rejection where §4.2.4's note ("multiple equivalent <redefine>ing
//     of the same schema document will not constitute a violation") wants
//     none. Redefinition identity is the ordered list of redefined
//     (element type, name, source location) triples, so the SAME
//     <xs:redefine> element reached twice is recognized — which is what
//     terminates every cycle — while two elements are not. This
//     over-rejects, so it can lose a valid assembly, never accept an
//     invalid one. <xs:override> no longer has this gap: its identity is
//     the substituted elements' ·canonical content· with the source
//     location left out (parser/override.go's writeCanonicalElement), so
//     two distinct but equivalent <xs:override> elements do reach one
//     document identity.
//   - Two children of ONE <xs:override> with the same element type and
//     name are reported under src-override, though §F.2's normative
//     stylesheet resolves the pair as first-match-wins and the published
//     REC therefore makes the case valid. This tracks the Working Group's
//     recorded intent instead (W3C Bugzilla 17617, 2012-06-29: "an
//     erratum is needed to make this situation an error"); the erratum
//     was never filed, and W3C Override/over021 is `accepted` on the
//     strength of it. Over-rejects in the same direction as the entry
//     above.
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
