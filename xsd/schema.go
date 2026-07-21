package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleSchPropsCorrect is Schema Properties Correct (Structures §3.17.6.1,
// id="sch-props-correct"): a schema's properties must match the §3.17.1 Schema
// property tableau. Finalize enforces ONLY clause 2 — no two schema components
// of the same kind (the same {…definitions}/{…declarations} property) share an
// expanded name (target namespace + local name) — because that is locally
// decidable without any cross-reference resolution. Clause 1's
// cross-reference-dependent requirements, and all reference resolution
// (src-resolve, §4.2.4), are deferred to the finalize resolution pass (#173);
// the instance-time cvc-resolve-instance (§3.17.6.3) lookups the Query views
// serve are a still-later consumer's concern.
const ruleSchPropsCorrect xsderr.Rule = "sch-props-correct"

// SchemaBuilder accumulates the schema components discovered during the
// parse/resolve phases (Structures §4.2.3 src-include: an implementation "must
// retain QName values for such references … until an appropriately-named
// component becomes available"). It performs no cross-reference resolution and
// no sch-props-correct enforcement beyond the per-kind duplicate detection
// Finalize runs — construction order is deliberately unconstrained per §4.2.4's
// "lazy/just-in-time" note. Call Finalize to obtain the immutable compiled
// Schema.
//
// Each slice holds its kind's components in the document order they were added
// (STYLE D2/D3); that order is the source of truth the Finalize indexes are
// derived from, never the reverse.
type SchemaBuilder struct {
	types               []TypeDefinition
	elements            []ElementDeclaration
	attributes          []AttributeDeclaration
	attributeGroups     []AttributeGroupDefinition
	modelGroups         []ModelGroupDefinition
	notations           []Notation
	identityConstraints []IdentityConstraint
	annotations         []Annotation
}

// NewSchemaBuilder returns an empty accumulating builder.
func NewSchemaBuilder() *SchemaBuilder { return &SchemaBuilder{} }

// AddType appends a top-level type definition in document order.
//
// It panics if t is nil — either a nil TypeDefinition interface value, or a
// non-nil interface wrapping a nil *SimpleType. A nil type definition is a
// caller/producer bug (the wrong constructor or an unchecked error), not a
// schema-validity condition, so a panic — not an xsderr rejection — is the
// right guard, mirroring NewFacet's wrong-constructor panic in simpletype.go.
func (b *SchemaBuilder) AddType(t TypeDefinition) {
	if t == nil {
		panic("xsd: SchemaBuilder.AddType: nil TypeDefinition")
	}
	if st, ok := t.(*SimpleType); ok && st == nil {
		panic("xsd: SchemaBuilder.AddType: nil *SimpleType")
	}
	b.types = append(b.types, t)
}

// AddElement appends a top-level element declaration in document order.
func (b *SchemaBuilder) AddElement(e ElementDeclaration) {
	b.elements = append(b.elements, e)
}

// AddAttribute appends a top-level attribute declaration in document order.
func (b *SchemaBuilder) AddAttribute(a AttributeDeclaration) {
	b.attributes = append(b.attributes, a)
}

// AddAttributeGroup appends a top-level attribute group definition in document
// order.
func (b *SchemaBuilder) AddAttributeGroup(g AttributeGroupDefinition) {
	b.attributeGroups = append(b.attributeGroups, g)
}

// AddModelGroup appends a top-level model group definition in document order.
func (b *SchemaBuilder) AddModelGroup(d ModelGroupDefinition) {
	b.modelGroups = append(b.modelGroups, d)
}

// AddNotation appends a top-level notation declaration in document order.
func (b *SchemaBuilder) AddNotation(n Notation) {
	b.notations = append(b.notations, n)
}

// AddIdentityConstraint appends a top-level identity-constraint definition in
// document order.
func (b *SchemaBuilder) AddIdentityConstraint(c IdentityConstraint) {
	b.identityConstraints = append(b.identityConstraints, c)
}

// AddAnnotation appends a schema-level annotation in document order.
func (b *SchemaBuilder) AddAnnotation(a Annotation) {
	b.annotations = append(b.annotations, a)
}

// Schema is the finalized, immutable compiled schema set (Structures §3.17.1,
// assembled per §4.2.1's "schema(D)"). It is constructible ONLY via
// SchemaBuilder.Finalize: its fields are unexported and it has no other
// constructor, so a not-yet-finalized accumulator can never be handed off as a
// finalized Schema (STYLE T1/T7) — "not finalized" (SchemaBuilder) and
// "finalized" (Schema) are distinct Go types, not two states of one type.
//
// The document-order slices are the source of truth; the by-expanded-QName maps
// are indexes DERIVED from those slices at Finalize and exist only for O(1)
// lookup — they never determine iteration order (STYLE D2/D3; see xsd/doc.go's
// "Maps exist only as internal lookup indexes and never determine order").
//
// Cross-reference resolution (turning a retained QName reference into a resolved
// component pointer, src-resolve §4.2.4) and the remaining sch-props-correct
// clauses are the finalize resolution pass's responsibility (#173), not this
// component's; Finalize here enforces only sch-props-correct §3.17.6.1 clause 2,
// the one clause locally decidable without any cross-reference resolution.
type Schema struct {
	types               []TypeDefinition
	elements            []ElementDeclaration
	attributes          []AttributeDeclaration
	attributeGroups     []AttributeGroupDefinition
	modelGroups         []ModelGroupDefinition
	notations           []Notation
	identityConstraints []IdentityConstraint
	annotations         []Annotation

	typeIndex           map[QName]TypeDefinition
	elementIndex        map[QName]ElementDeclaration
	attributeIndex      map[QName]AttributeDeclaration
	attributeGroupIndex map[QName]AttributeGroupDefinition
	modelGroupIndex     map[QName]ModelGroupDefinition
	notationIndex       map[QName]Notation
	idcIndex            map[QName]IdentityConstraint
}

// Finalize builds the immutable Schema from the accumulated components. It
// copies each document-order slice onto fresh backing arrays — so the builder
// stays independently usable afterward, decoupled from the returned Schema —
// and builds each by-expanded-name index over the copy. The *SimpleType
// pointees a type-definition slice holds are shared, NOT deep-copied: pointer
// identity is load-bearing for SimpleType (see its doc), so the compiled Schema
// must reference the very same nodes.
//
// It rejects, charging Schema Properties Correct (§3.17.6.1, sch-props-correct)
// clause 2: two components of the same kind (the same §3.17.1 {…definitions}/
// {…declarations} property) sharing an expanded name — target namespace plus
// local name. The scan is deterministic (STYLE D2): each kind's slice is walked
// in document order and each name tested against a seen-set map, so the first
// duplicate by index is the one reported (the map is never ranged to produce
// the verdict). Because these top-level value components carry no source
// location accessor, a rejection is charged the zero xsderr.Loc{}, exactly as
// the synthesized-component convention throughout this package permits.
//
// Every OTHER sch-props-correct clause (in particular clause 1's
// cross-reference-dependent requirements) and all cross-reference resolution
// are the caller's responsibility through further passes (#173) — Finalize
// chases no reference.
func (b *SchemaBuilder) Finalize() (*Schema, error) {
	typeIndex, err := indexByName(b.types, TypeDefinition.Name, "type definitions")
	if err != nil {
		return nil, err
	}
	elementIndex, err := indexByName(b.elements, ElementDeclaration.Name, "element declarations")
	if err != nil {
		return nil, err
	}
	attributeIndex, err := indexByName(b.attributes, AttributeDeclaration.Name, "attribute declarations")
	if err != nil {
		return nil, err
	}
	attributeGroupIndex, err := indexByName(b.attributeGroups, AttributeGroupDefinition.Name, "attribute group definitions")
	if err != nil {
		return nil, err
	}
	modelGroupIndex, err := indexByName(b.modelGroups, ModelGroupDefinition.Name, "model group definitions")
	if err != nil {
		return nil, err
	}
	notationIndex, err := indexByName(b.notations, Notation.Name, "notation declarations")
	if err != nil {
		return nil, err
	}
	idcIndex, err := indexByName(b.identityConstraints, IdentityConstraint.Name, "identity-constraint definitions")
	if err != nil {
		return nil, err
	}
	return &Schema{
		types:               cloneSlice(b.types),
		elements:            cloneSlice(b.elements),
		attributes:          cloneSlice(b.attributes),
		attributeGroups:     cloneSlice(b.attributeGroups),
		modelGroups:         cloneSlice(b.modelGroups),
		notations:           cloneSlice(b.notations),
		identityConstraints: cloneSlice(b.identityConstraints),
		annotations:         cloneSlice(b.annotations),
		typeIndex:           typeIndex,
		elementIndex:        elementIndex,
		attributeIndex:      attributeIndex,
		attributeGroupIndex: attributeGroupIndex,
		modelGroupIndex:     modelGroupIndex,
		notationIndex:       notationIndex,
		idcIndex:            idcIndex,
	}, nil
}

// indexByName builds the by-expanded-name lookup index for one kind's
// document-order slice, rejecting sch-props-correct (§3.17.6.1) clause 2: two
// components of this kind sharing an expanded name. The slice is walked in
// document order (STYLE D2), so the first duplicate by index is the one
// reported; kind names the §3.17.1 property for the message. An empty slice
// yields a nil map (a nil map reads as a miss, which is the correct lookup
// behavior). name is the accessor promoting each component's {name}; passing a
// method expression (e.g. ElementDeclaration.Name) keeps every kind on one code
// path (STYLE T4).
func indexByName[T any](items []T, name func(T) QName, kind string) (map[QName]T, error) {
	if len(items) == 0 {
		return nil, nil
	}
	index := make(map[QName]T, len(items))
	for i, item := range items {
		n := name(item)
		if _, dup := index[n]; dup {
			return nil, xsderr.New(ruleSchPropsCorrect, xsderr.Loc{},
				"schema {%s}[%d] repeats the expanded name %s, but sch-props-correct clause 2 forbids two components of the same kind sharing an expanded name", kind, i, n)
		}
		index[n] = item
	}
	return index, nil
}

// cloneSlice returns a copy of s on a fresh backing array, holding an empty
// input as nil (this package's standing empty-is-nil convention). It is used by
// Finalize to decouple the compiled Schema's slices from the builder's.
func cloneSlice[T any](s []T) []T {
	if len(s) == 0 {
		return nil
	}
	return append([]T(nil), s...)
}

// ElementResolver is the minimal capability a consumer needs to look up a
// top-level element declaration by its expanded name (STYLE T3). Its intended
// consumer is the instance validator (cvc-resolve-instance §3.17.6.3, a future
// consumer) and any tooling that needs only element lookup, not the whole
// schema.
type ElementResolver interface {
	Element(name QName) (ElementDeclaration, bool)
}

// AttributeResolver is the minimal capability a consumer needs to look up a
// top-level attribute declaration by its expanded name (STYLE T3). Its intended
// consumer is the instance validator (a future consumer) and attribute-use
// resolution during finalize (#173).
type AttributeResolver interface {
	Attribute(name QName) (AttributeDeclaration, bool)
}

// TypeResolver is the minimal capability a consumer needs to look up a
// top-level type definition (simple or complex) by its expanded name (STYLE
// T3). Its intended consumer is finalize's {base type definition}/{type
// definition} cross-reference resolution (#173) and the instance validator's
// xsi:type resolution (a future consumer).
type TypeResolver interface {
	Type(name QName) (TypeDefinition, bool)
}

// *Schema is the sole implementation of each Query capability view; these
// assertions keep that promise checked at compile time.
var (
	_ ElementResolver   = (*Schema)(nil)
	_ AttributeResolver = (*Schema)(nil)
	_ TypeResolver      = (*Schema)(nil)
)

// Element returns the top-level element declaration with the given expanded
// name and true, or the zero ElementDeclaration and false when none is
// declared. It is a read-only window onto the compiled set (§3.17.1 {element
// declarations}); it copies nothing.
func (s *Schema) Element(name QName) (ElementDeclaration, bool) {
	d, ok := s.elementIndex[name]
	return d, ok
}

// Attribute returns the top-level attribute declaration with the given expanded
// name and true, or the zero AttributeDeclaration and false when none is
// declared. It is a read-only window onto the compiled set (§3.17.1 {attribute
// declarations}); it copies nothing.
func (s *Schema) Attribute(name QName) (AttributeDeclaration, bool) {
	d, ok := s.attributeIndex[name]
	return d, ok
}

// Type returns the top-level type definition (simple or complex) with the given
// expanded name and true, or nil and false when none is declared. It is a
// read-only window onto the compiled set (§3.17.1 {type definitions}); it
// copies nothing.
func (s *Schema) Type(name QName) (TypeDefinition, bool) {
	d, ok := s.typeIndex[name]
	return d, ok
}
