// Package xsd is the XSD 1.1 component model: the immutable, in-memory
// representation of a compiled schema set.
//
// It is a pure leaf — it imports only xsderr and the stdlib. Parsing
// lives in package parser; values in package value; this package is the
// shape those operate on.
//
// # Components
//
// The model covers the component kinds of Structures §2.2: simple and
// complex type definitions, element and attribute declarations, attribute
// and model groups, particles, wildcards, identity constraints, type
// alternatives (CTA), assertions, notations, and annotations.
//
// Design rules (see docs/STYLE.md, docs/PRINCIPLES.md):
//
//   - Immutable after construction. Construction happens in phases
//     (parse → resolve → finalize) so no traversal ever needs a cycle
//     check (D4); spec-forbidden circularities are rejected at finalize
//     with their named src-/cos- rule.
//
//   - Every child collection is a slice in document order. Maps exist
//     only as internal lookup indexes and never determine order (D2).
//
//   - One fact, one encoding (D3): nothing derivable is stored. There is
//     no Primitive bool — a type that defines its own fundamental facets
//     is a primitive, answered by IsPrimitive(); effective facets are
//     computed on demand by merging the base chain, never cached.
//
//   - Closed sets (variety, derivation method, process-contents, use)
//     are typed constants with unexported tags, never strings (T1).
//
// Names are expanded QNames:
//
//	type QName struct{ Space, Local string }
//
// The zero value means absent/anonymous; String() renders Clark notation
// ("{ns}local").
//
// A name identifies a component that HAS one. An anonymous component has
// none, so where one component must point at another that may be
// anonymous — or at a LOCAL declaration, which is not name-unique — the
// reference carries a ComponentID instead: an opaque identity minted by
// the producer with NewComponentID before either endpoint is built, and
// threaded into both. It is compared with ==, never rendered (its
// underlying value is an address, so any textual or sorted form would be
// nondeterministic, D1/D2), and never derived from position — Loc is
// provenance, not identity. ComplexType's {context} (§3.4.1) is the first
// slot to use it, and the reciprocal second is the anonymous-container arm
// of each scope-parent sum — ElementScopeParent's
// AnonymousComplexTypeScopeParent (§3.3.1 sc_e) and AttributeScopeParent's
// AttributeAnonymousComplexTypeScopeParent (§3.2.1 sc_a). All three hold
// the SAME token: one mint per inline construct serves the back-pointers
// and the forward reference alike.
//
// The eight component kinds a schema's §3.17.1 properties hold — element
// and attribute declarations, complex and simple type definitions, model
// group and attribute group definitions, notations, and identity
// constraints — additionally report a source position through Loc. That
// position is PROVENANCE, not a component property at all — no kind's
// §3.x.1 property list has it. It is where the declaring element sits in
// the schema document, retained from the constructor so a finalize-time
// rejection can cite file:line:column instead of "?" — sch-props-correct
// clause 2 (schema.go) was the first such consumer, and resolve.go's Phase
// A/B rejections (src-resolve, c-props-correct, ct-props-correct,
// mg-props-correct, e-props-correct) are the second: each is charged to
// the REFERRING or offending component's own Loc, per the referrer-Loc
// convention documented on resolveReferences. The zero xsderr.Loc means
// the position is unknown, and is the correct value for a component with
// no schema document behind it — parser.Produce's synthesized xs:anyType
// and package builtin's seeded built-in datatypes are the legitimate
// zero-Loc producers. Other constructors take a loc to charge their own
// rejections but do not retain it: nothing consumes those positions yet,
// so no accessor is exported for them (T5).
//
// # Query API
//
// Direct lookups over a compiled schema set — element, attribute, and
// type definitions by QName — exposed as minimal capability views (T3):
// a consumer that needs only element lookup receives an interface with
// exactly that method, not the whole schema. Those by-QName views are
// read-only windows onto the compiled set; they never copy it.
//
// The views are ElementResolver, AttributeResolver, and TypeResolver;
// *Schema satisfies all three, and SchemaBuilder.Finalize (or its sibling
// SchemaBuilder.FinalizeWith) is the only way to obtain one.
//
// Alongside them *Schema enumerates each of the eight §3.17.1 properties
// in document order — Types, Elements, Attributes, AttributeGroups,
// ModelGroups, Notations, IdentityConstraints, Annotations. Unlike the
// by-QName views these DO copy: each returns a fresh slice (the
// components in it are shared and immutable), so a caller cannot mutate
// through the result and desync a source-of-truth slice from the index
// derived from it. They are methods on *Schema and nothing more; no
// capability view bundles them until a second consumer needs one (T5).
//
// This section is shipped surface — see the package Examples for the
// construct → Finalize → query sequence.
//
// # Mapping rules
//
// One §3.4.2 mapping rule is exported, and only because it has two
// callers on opposite sides of the parse/finalize line:
// ExtensionContentType is Structures §3.4.2.3.3 clause 4.2, the merge of
// a base's {content type} with an extension's ·effective content·. The
// producer applies it to one <extension> element at parse time;
// cos-ct-extends clause 1.5 applies it repeatedly at finalize, over a
// derivation chain re-ordered into an order no schema document
// expresses. It takes its <group ref> lookup as a parameter because
// those two callers resolve one differently, not because it is an
// extension point. Every other mapping rule this package completes —
// the §3.4.2.4 clause 3 and §3.4.2.5 clause 2 folds, and this one's
// structural inverse — stays in-package (T5).
//
// # Value spaces
//
// A few finalize-time constraints reach into a value space. Two COMPARE
// {value}s — au-props-correct (§3.5.6) clause 3, loc-testSubP (§3.4.6.4)
// clauses 4.2 and 5.2.2 — and two ask whether one {lexical form} denotes a
// value of its type at all: a-props-correct (§3.2.6.1) clause 2 and
// au-props-correct clause 2, both charging Simple Default Valid (§3.2.6.2).
// A {value} is an ·actual value·, not a lexical string, and mapping a
// lexical to a value needs package value, which is layered above this pure
// leaf — so the whole capability is taken as an INPUT: the ValueSpace
// interface, passed to SchemaBuilder.FinalizeWith. Every ValueSpace method
// may answer "undecided", and undecided always accepts, so plain Finalize
// (which installs none) is the fully fail-open configuration.
// value.NewValueSpace is the implementation the parser installs.
//
// FinalizeWith installs a SECOND capability at the same seam, and it is
// deliberately a second interface rather than a fourth ValueSpace method:
// SimpleTypeRestrictionChecker charges the facet-VALUE half of
// cos-st-restricts (§3.16.6.2) — the atomic applicability clause, whose
// per-primitive applicable sets come from a generated table, and the bound
// and enumeration constraints, which need a value space. That one is
// REJECT-CAPABLE: it answers with an error carrying a per-facet rule ID,
// which finalize returns as the schema's rejection, so bundling it with the
// fail-open comparisons would make an incoherent capability mix
// representable (T1). checkSimpleTypeDerivations puts every simple type
// the assembled schema reaches to it, anonymous inline ones included;
// builtin.NewRestrictionChecker is the implementation the parser installs.
//
// # Walk API
//
// Traversal of a type's effective content model. The reusable core is an
// algebra — type-derivation validity, substitution-group acceptance,
// wildcard admission (one canonical implementation, context supplied by
// the caller: Wildcard.AllowsName answers cvc-wildcard clause 1 from the
// {namespace constraint} alone, while clauses 2-3 resolve the
// defined/sibling keywords against the declaration graph and the
// containing complex type the caller supplies), attribute-use lookup.
// Of that algebra Wildcard.AllowsName is the one canonical admission entry
// point, and the only one a caller admitting a name should reach for;
// NamespaceConstraint.AllowsName and NamespaceConstraint.AllowsNamespace
// are exported beneath it as the {namespace constraint} property's own
// lower-level accessors. The rest is in-package machinery finalize drives.
//
// Until a driver ships, a consumer traverses a content model by hand,
// switching Particle.Term over the TermOrRef sealed sum and then, for the
// ResolvedTerm branch, over the Term sealed sum; the package Examples walk
// one end to end.
//
// # Planned contract (M5/M9 — not yet implemented)
//
// Two drivers are designed on the algebra above. Neither exists: the
// package declares no Walker and no Matcher.
//
//	type Walker   // M9, codegen's push driver
//	    Exhaustive and schema-only: visits every particle reachable
//	    through sequences, choices, all-groups, and named group
//	    references, in document order.
//
//	type Matcher  // M5, validation's pull driver
//	    Instance-guided: advances the content model one child at a time,
//	    greedily and deterministically.
//
// Substitution groups will not be expanded at walk time — instance-time
// concern. Recursive named-group references terminate by construction in
// the finalized model; the Walker will need no visited set beyond the
// path-scoped guard the spec's one legal nesting requires.
package xsd
