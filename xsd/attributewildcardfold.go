package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file completes one mapping rule: Mapping Rule for Attribute Wildcard
// Property (Structures §3.4.2.5, dcl.ctd.anyatt), whose clause 2.2 no producer
// can apply.
//
// Clause 1 — the ·complete wildcard·, §3.6.2.2's intersection of the type's own
// <anyAttribute> with those of the attribute groups it references — is decidable
// from one <complexType> element, and parser/produce_complex.go applies it
// (combineAttributeWildcards). Clause 2 then splits on {derivation method}: for a
// restriction the value IS the ·complete wildcard· (clause 2.1) and the producer
// is already right, but for an EXTENSION the value folds in the ·base wildcard·
// (clause 2.2), which needs the base COMPONENT and so is reachable only once the
// whole schema set is assembled. So {attribute wildcard} arrives at finalize
// under-approximated for extensions, and is completed HERE, once, before any
// constraint reads it.
//
// This is attributeusefold.go's property-wise sibling and deliberately not merged
// with it: the two folds share nothing but the shape of the walk (base before
// derived), and §3.4.2.4 clause 3 and §3.4.2.5 clause 2 are different rules with
// different case lists and different combination operators. A shared "fold
// engine" would put two rules behind one abstraction for the sake of a loop.
//
// The direction of the omission is what made it urgent, and it is the OPPOSITE of
// most gaps in this package: an extension whose base carried the admitting
// <anyAttribute> read as admitting nothing, so a restriction of that extension was
// FALSELY REJECTED by derivation-ok-restriction clause 3 for an attribute the base
// really does allow (#265). Reading the property is now correct everywhere;
// defaultbinding.go's attributeDefaultBinding needs no chain walk of its own, and
// a naive chain walk there would have been wrong anyway — every type reaches
// ·xs:anyType·, whose lax ##any attribute wildcard would make every check vacuous.
// The fold does not have that defect: clause 2.1 stops it at the first
// restriction in the chain.

// attributeWildcardFold is one fold pass's working state: where each named
// Complex Type Definition sits in s.types, the folded {attribute wildcard}
// computed for each position, and which positions are done. It is a parameter
// object rather than three parallel arguments threaded through the recursion, and
// it is discarded when the pass returns — nothing derived from it outlives
// foldAttributeWildcards.
type attributeWildcardFold struct {
	// position maps the expanded {name} of a COMPLEX type definition to its
	// index in s.types. Simple types are deliberately absent: a base NAME that
	// misses here is §3.4.2.5 clause 2.2.1.2's "otherwise ·absent·", which is
	// exactly the answer a missing lookup gives. Names are unique across type
	// definitions (sch-props-correct clause 2, enforced at Finalize), so the map
	// is a function. It is a point-lookup index and never ranged (STYLE D2).
	//
	// It is consulted for the TypeDefinitionRef arm of a {base type definition}
	// ONLY. An anonymous inline base has no name and so no position, which is
	// not clause 2.2.1.2 but a base reached a different way; see
	// baseAttributeWildcard.
	position map[QName]int
	// wildcards holds each position's folded value, nil for the ·absent·
	// property — the same pointer-for-absence encoding NewComplexType's own
	// parameter uses, so no second "present" flag is carried beside it (STYLE
	// D3). folded is the memo mark, which nil cannot serve as: an absent
	// property is a legitimate folded result.
	wildcards []*Wildcard
	folded    []bool
}

// foldAttributeWildcards materialises §3.4.2.5 clause 2 into every Complex Type
// Definition's {attribute wildcard}, in base-before-derived dependency order.
//
// It is the second of the finalize pass's two mutations, beside foldAttributeUses
// (attributeusefold.go), and rests on the same footing: not a resolved-pointer
// cache — that would hold state derivable from the QName plus the index (STYLE
// D3) — but a property OVERWRITTEN with the value §3.4.2.5 defines, so the spec's
// value has exactly one encoding afterwards and the producer's partial one is gone
// rather than kept beside it.
//
// The two folds are independent — {attribute uses} and {attribute wildcard} are
// different properties, and neither's rule reads the other — so their relative
// order carries no verdict. Both must complete before Phase D's constraints, which
// read both.
//
// PHASE ORDER IS LOAD-BEARING, exactly as for foldAttributeUses. It must run after
// Phase A (existence), so a base name that misses in position is a simple base or
// an absent one and never a dangling reference, and after Phase B (circularity),
// which is what licenses the recursion below to carry no visited set: the {base
// type definition} graph is known acyclic apart from ·xs:anyType·'s
// self-derivation (§3.4.7, any-type-itself), the one edge foldTypeAttributeWildcard
// excludes by position rather than by a guard (PRINCIPLES 9, STYLE D4). It must run
// before Phase D, the first phase to read {attribute wildcard}.
//
// It returns an error where foldAttributeUses cannot, because clause 2.2.2.3
// CONSTRUCTS a component: NewNamespaceConstraint (through UnionNamespaceConstraint)
// and NewWildcard both re-check w-props-correct. Neither can fail for two
// validly-constructed operands — see UnionNamespaceConstraint's own note — but the
// error is plumbed out as a real return value rather than swallowed or panicked,
// so any future divergence fails closed as a w-props-correct rejection (STYLE P3).
func (s *Schema) foldAttributeWildcards() error {
	f := &attributeWildcardFold{
		position:  make(map[QName]int, len(s.types)),
		wildcards: make([]*Wildcard, len(s.types)),
		folded:    make([]bool, len(s.types)),
	}
	for i, t := range s.types {
		c, ok := t.(ComplexType)
		if !ok || c.Name() == (QName{}) {
			continue
		}
		f.position[c.Name()] = i
	}
	for i, t := range s.types {
		if _, ok := t.(ComplexType); !ok {
			continue // a simple type definition has no {attribute wildcard}
		}
		if _, err := s.foldTypeAttributeWildcard(f, i); err != nil {
			return err
		}
	}
	s.storeFoldedAttributeWildcards(f)
	return nil
}

// foldTypeAttributeWildcard computes and memoises the {attribute wildcard} of the
// Complex Type Definition at position i, recursing into its {base type definition}
// first so the base's own value is already complete when clause 2.2 folds it in
// (base-before-derived, STYLE D4). A nil result is the ·absent· property.
//
// The recursion terminates for the same two reasons every other base walk in this
// package does: Phase B has rejected every circular chain — including one running
// through an anonymous inline base, which Phase B descends for exactly this
// reason (resolve.go's checkComplexBaseAcyclic) — and ·xs:anyType·'s permitted
// self-derivation is excluded by j != i, the one self-edge §3.4.7 allows, which
// no acyclicity check can remove. ·xs:anyType· derives by restriction, so clause
// 2.1 would return its own wildcard unchanged in any case.
func (s *Schema) foldTypeAttributeWildcard(f *attributeWildcardFold, i int) (*Wildcard, error) {
	if f.folded[i] {
		return f.wildcards[i], nil
	}
	c, ok := s.types[i].(ComplexType)
	if !ok {
		panic("xsd: foldTypeAttributeWildcard: position of a non-complex type definition")
	}
	w, err := s.clause2AttributeWildcard(f, c, i)
	if err != nil {
		return nil, err
	}
	f.wildcards[i], f.folded[i] = w, true
	return w, nil
}

// clause2AttributeWildcard decides §3.4.2.5 clause 2 for one Complex Type
// Definition: which of the ·complete wildcard· and the ·base wildcard· the
// property takes, or how the two combine.
//
//   - clause 2.1 (restriction): the ·complete wildcard· alone, nothing inherited.
//     This is what keeps the fold from collapsing every type onto ·xs:anyType·'s
//     ##any wildcard — a restriction anywhere in the chain stops it.
//   - clause 2.2.1 (extension): the ·base wildcard· is the {base type
//     definition}'s {attribute wildcard} when that base is a complex type
//     definition (2.2.1.1), otherwise ·absent· (2.2.1.2) — which is what a miss in
//     position means: a simple base, or none at all.
//   - clause 2.2.2: unionExtensionAttributeWildcard.
//
// The base's value is the base's own FOLDED one, so an extension of an extension
// carries the whole inherited union: clause 2.2.1.1 names the base's {attribute
// wildcard} property, which is itself this rule's output, not the base's
// <anyAttribute>.
//
// It takes the component rather than a position because not every type it must
// answer for HAS one — see foldComponentAttributeUses (attributeusefold.go) for
// the same argument on the sibling property; i is the folding type's own
// position, used only to exclude the ·xs:anyType· self-edge.
func (s *Schema) clause2AttributeWildcard(f *attributeWildcardFold, c ComplexType, i int) (*Wildcard, error) {
	own := completeAttributeWildcard(c)
	if c.DerivationMethod() != DerivationExtension {
		return own, nil // clause 2.1
	}
	base, err := s.baseAttributeWildcard(f, c, i)
	if err != nil {
		return nil, err
	}
	return unionExtensionAttributeWildcard(c.loc, own, base)
}

// baseAttributeWildcard returns the ·base wildcard· of §3.4.2.5 clause 2.2.1,
// exhaustively over c's {base type definition} slot: 2.2.1.1's base {attribute
// wildcard} when the base is a Complex Type Definition, and 2.2.1.2's ·absent·
// (a nil result) otherwise — a base that is absent, simple, unresolvable, or
// ·xs:anyType· reached by its own self-edge.
//
// The InlineTypeDefinition arm is not a miss: the src-expredef clause 1.1
// original a redefining complex type owns is in no position map, and reading that
// as ·absent· would drop the wildcard a redefining EXTENSION inherits, so an
// attribute its original admits would be rejected on the redefinition (#505).
func (s *Schema) baseAttributeWildcard(f *attributeWildcardFold, c ComplexType, i int) (*Wildcard, error) {
	switch b := c.Base().(type) {
	case nil:
		return nil, nil
	case TypeDefinitionRef:
		j, ok := f.position[b.Name]
		if !ok || j == i {
			return nil, nil
		}
		return s.foldTypeAttributeWildcard(f, j)
	case InlineTypeDefinition:
		bc, isComplex := b.Definition.(ComplexType)
		if !isComplex {
			return nil, nil
		}
		return s.clause2AttributeWildcard(f, bc, i)
	default:
		panic("xsd: baseAttributeWildcard: non-exhaustive TypeDefinitionOrRef switch")
	}
}

// completeAttributeWildcard is §3.4.2.5 clause 1's ·complete wildcard· for a type
// the producer has already mapped: §3.6.2.2's combination of the type's own
// <anyAttribute> with the referenced attribute groups' is exactly what
// NewComplexType was given, so it is read back off the component rather than
// recomputed (the producer's combineAttributeWildcards is its one encoding). nil is
// the ·absent· property.
func completeAttributeWildcard(c ComplexType) *Wildcard {
	w, ok := c.AttributeWildcard()
	if !ok {
		return nil
	}
	return &w
}

// unionExtensionAttributeWildcard is §3.4.2.5 clause 2.2.2, whose three cases are
// tested in spec order because the constraint says "the FIRST case among the
// following which applies":
//
//   - 2.2.2.1: the ·base wildcard· is ·absent· -> the ·complete wildcard·;
//   - 2.2.2.2: the ·complete wildcard· is ·absent· -> the ·base wildcard·;
//   - 2.2.2.3: otherwise a NEW Wildcard whose {process contents} and {annotations}
//     are the ·complete wildcard·'s — the extension's OWN declarations, never the
//     base's — and whose {namespace constraint} is the wildcard union of the two,
//     as defined in Attribute Wildcard Union (§3.10.6.3, cos-aw-union). Only
//     {namespace constraint} is combined; the other two properties are copied.
//
// Cases 2.2.2.1 and 2.2.2.2 hand back an operand unchanged, which is what the
// spec says: the value IS that wildcard, not a copy of it, and no union is formed.
//
// loc is the extension's own source position, charged to the defensive rejection
// either constructor can return. Both are unreachable for two validly-constructed
// operands (UnionNamespaceConstraint's doc argues the namespace-constraint half;
// the Wildcard half re-checks a {process contents} that already passed NewWildcard
// on the operand), so the errors are returned rather than swallowed purely so a
// future divergence fails closed (STYLE P3).
func unionExtensionAttributeWildcard(loc xsderr.Loc, own, base *Wildcard) (*Wildcard, error) {
	if base == nil {
		return own, nil // clause 2.2.2.1
	}
	if own == nil {
		return base, nil // clause 2.2.2.2
	}
	nc, err := UnionNamespaceConstraint(loc, own.NamespaceConstraint(), base.NamespaceConstraint())
	if err != nil {
		return nil, err
	}
	w, err := NewWildcard(loc, nc, own.ProcessContents(), own.Annotations())
	if err != nil {
		return nil, err
	}
	return &w, nil // clause 2.2.2.3
}

// storeFoldedAttributeWildcards writes each folded value back onto its Complex
// Type Definition. ComplexType is a VALUE type held in two places — the
// document-order s.types slice and the by-name s.typeIndex — so both are
// re-seated; a consumer that reaches a type either way sees the same folded
// property. The index entry exists only for a named type (an anonymous one is in
// no §3.17.1 symbol table).
//
// An ANONYMOUS base is in neither place, so its own stored {attribute wildcard}
// is left at the producer's clause-1 value; see storeFoldedAttributeUses
// (attributeusefold.go) for the shared argument and the recorded gap.
//
// A nil folded value needs no store: the fold never turns a present property
// absent — clause 2.1 keeps the ·complete wildcard·, and every clause 2.2 case
// yields ·absent· only when BOTH operands are — so nil means the producer already
// left the property absent and there is nothing to overwrite.
func (s *Schema) storeFoldedAttributeWildcards(f *attributeWildcardFold) {
	for i, w := range f.wildcards {
		if w == nil {
			continue
		}
		c, ok := s.types[i].(ComplexType)
		if !ok {
			continue
		}
		c.attributeWildcard, c.hasAttributeWildcard = *w, true
		s.types[i] = c
		if c.Name() != (QName{}) {
			s.typeIndex[c.Name()] = c
		}
	}
}
