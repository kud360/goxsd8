package xsd

import "slices"

// This file completes one mapping rule: Mapping Rule for Attribute Uses Property
// (Structures §3.4.2.4, dcl.ctd.attuses), whose clause 3 no producer can apply.
//
// Clauses 1 and 2 — the attribute uses corresponding to a source declaration's
// own <attribute> children, and the {attribute uses} of the attribute groups its
// <attributeGroup ref> children ·resolve· to — are decidable from one <complexType>
// element, and parser/produce_complex.go applies them. Clause 3 is not: the uses
// "inherited" from the {base type definition} need the base COMPONENT, which is
// reachable only once the whole schema set is assembled and its base references
// are known to resolve. So {attribute uses} arrives at finalize under-approximated,
// and is completed HERE, once, before any constraint reads it.
//
// The alternative — leaving the property partial and folding the base chain at
// each read site — is what this replaced (#262/#264's foldedAttributeUse). It put
// one relation in two encodings, one of them per-name and search-order-dependent,
// and it under-approximated for every consumer that quantifies over the whole set
// rather than looking up a single name (STYLE T4/D3, #401).

// attributeUseFold is one fold pass's working state: where each named Complex
// Type Definition sits in s.types, the folded set computed for each position,
// and which positions are done. It is a parameter object rather than three
// parallel arguments threaded through the recursion, and it is discarded when
// the pass returns — nothing derived from it outlives foldAttributeUses.
type attributeUseFold struct {
	// position maps the expanded {name} of a COMPLEX type definition to its
	// index in s.types. Simple types are deliberately absent: a base name that
	// misses here is §3.4.2.4 clause 3.3's "otherwise no attribute use is
	// inherited", which is exactly the answer a missing lookup gives. Names are
	// unique across type definitions (sch-props-correct clause 2, enforced at
	// Finalize), so the map is a function. It is a point-lookup index and never
	// ranged (STYLE D2).
	position map[QName]int
	uses     [][]AttributeUse
	folded   []bool
}

// foldAttributeUses materialises §3.4.2.4 clause 3 into every Complex Type
// Definition's {attribute uses}, in base-before-derived dependency order.
//
// It is the ONE mutation the finalize pass performs. That is a deliberate
// exception to resolve.go's "stores nothing" stance and not a resolved-pointer
// cache: a cache would hold state derivable from the QName plus the index (STYLE
// D3), whereas this OVERWRITES a property with its correct value. Afterwards the
// spec's set has exactly one encoding — the stored property — and the producer's
// partial value is gone rather than kept beside it.
//
// PHASE ORDER IS LOAD-BEARING. It must run after Phase A (existence), so a base
// name that misses in position is a simple base or an absent one and never a
// dangling reference, and after Phase B (circularity), which is what licenses the
// recursion below to carry no visited set: the {base type definition} graph is
// known acyclic apart from ·xs:anyType·'s self-derivation (§3.4.7,
// any-type-itself), the one edge foldTypeAttributeUses excludes by position
// rather than by a guard (PRINCIPLES 9, STYLE D4). It must run before Phase D,
// which is the first phase to read {attribute uses}.
//
// Clause 3.2.2 — the <attribute use="prohibited"> child, which BLOCKS the
// same-named inherited use — is applied here too, from the prohibited names the
// producer records on the type (complextype.go's prohibitedAttributeNames). It is
// not optional and its omission is not fail-open: because the block is what makes
// the name absent from a restriction B's set, skipping it leaves B carrying the
// base's use, and an EXTENSION of B that declares that name itself then holds two
// same-named members and is FALSELY rejected by ct-props-correct clause 4
// (checkAttributeUseNamesUnique) for a duplicate the source never wrote.
func (s *Schema) foldAttributeUses() {
	f := &attributeUseFold{
		position: make(map[QName]int, len(s.types)),
		uses:     make([][]AttributeUse, len(s.types)),
		folded:   make([]bool, len(s.types)),
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
			continue // a simple type definition has no {attribute uses}
		}
		s.foldTypeAttributeUses(f, i)
	}
	s.storeFoldedAttributeUses(f)
}

// foldTypeAttributeUses computes and memoises the folded {attribute uses} of the
// Complex Type Definition at position i, recursing into its {base type
// definition} first so the base's own set is already complete when clause 3 folds
// it in (base-before-derived, STYLE D4).
//
// The recursion terminates for the same two reasons every other base walk in this
// package does: Phase B has rejected every circular chain, and ·xs:anyType·'s
// permitted self-derivation is excluded by j != i — the one self-edge §3.4.7
// allows, which no acyclicity check can remove. §3.4.7 gives ·xs:anyType· an
// empty {attribute uses}, so excluding it loses nothing.
func (s *Schema) foldTypeAttributeUses(f *attributeUseFold, i int) []AttributeUse {
	if f.folded[i] {
		return f.uses[i]
	}
	c, ok := s.types[i].(ComplexType)
	if !ok {
		panic("xsd: foldTypeAttributeUses: position of a non-complex type definition")
	}
	uses := c.attributeUses // clauses 1 and 2, as the producer mapped them
	if j, ok := f.position[c.BaseTypeDefinitionName()]; ok && j != i {
		uses = inheritAttributeUses(uses, s.foldTypeAttributeUses(f, j), c.DerivationMethod(), c.prohibitedAttributeNames)
	}
	f.uses[i], f.folded[i] = uses, true
	return uses
}

// inheritAttributeUses is §3.4.2.4 clause 3's inherited half: own is the set
// clauses 1 and 2 built, base is the {base type definition}'s already-folded
// {attribute uses}, method selects the case, and prohibited is the expanded names
// the deriving type's own source gave use="prohibited".
//
//   - clause 3.1 (extension): every member of base is inherited, unconditionally.
//     A name the extension also declares itself therefore appears TWICE, which is
//     precisely what ct-props-correct clause 4 forbids and charges
//     (checkAttributeUseNamesUnique) — an extension may add attributes, never
//     re-declare the base's. prohibited is ignored on this branch, exactly as
//     §3.4.2.4's Note directs: use="prohibited" outside a restriction is
//     "pointless, though not an error", and the <attribute> "is simply ignored".
//   - clause 3.2 (restriction): every member of base is inherited EXCEPT one whose
//     {attribute declaration}'s expanded name is either already in own — clause
//     3.2.1's "already been included in the set, following the rules in clause 1
//     or clause 2 above" — or among prohibited, which is clause 3.2.2.
//   - clause 3.3 (no complex base): unreachable here — a base that is absent,
//     simple, or unresolvable never yields a position, so this function is not
//     called at all.
//
// The result is a slice in document order — own uses first, then the base's, each
// in its own document order — and no map takes part (STYLE D2). own is copied
// rather than appended to, so the component's backing array is never aliased into
// a longer slice.
func inheritAttributeUses(own, base []AttributeUse, method DerivationMethod, prohibited []QName) []AttributeUse {
	folded := append(make([]AttributeUse, 0, len(own)+len(base)), own...)
	for _, u := range base {
		name := attributeUseName(u)
		if method == DerivationRestriction && (hasAttributeUseNamed(own, name) || slices.Contains(prohibited, name)) {
			continue // clause 3.2.1, clause 3.2.2
		}
		folded = append(folded, u) // clause 3.1, and clause 3.2 otherwise
	}
	return folded
}

// hasAttributeUseNamed reports whether the set already carries an attribute use
// for the expanded name, reusing the one document-order scan every other consumer
// takes (STYLE T4).
func hasAttributeUseNamed(uses []AttributeUse, name QName) bool {
	_, ok := findAttributeUse(uses, name)
	return ok
}

// storeFoldedAttributeUses writes each folded set back onto its Complex Type
// Definition. ComplexType is a VALUE type held in two places — the document-order
// s.types slice and the by-name s.typeIndex — so both are re-seated; a consumer
// that reaches a type either way sees the same folded property. The index entry
// exists only for a named type (an anonymous one is in no §3.17.1 symbol table),
// and no anonymous type can be a {base type definition}, since a base is named by
// QName.
func (s *Schema) storeFoldedAttributeUses(f *attributeUseFold) {
	for i, uses := range f.uses {
		c, ok := s.types[i].(ComplexType)
		if !ok {
			continue
		}
		c.attributeUses = uses
		s.types[i] = c
		if c.Name() != (QName{}) {
			s.typeIndex[c.Name()] = c
		}
	}
}
