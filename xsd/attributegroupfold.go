package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file completes the two mapping rules an <attributeGroup ref> takes part
// in, for every container that can hold one: the {attribute uses} union of
// §3.6.2.1 and the {attribute wildcard} intersection of §3.6.2.2
// (declare-attributeGroup-wildcard). For an Attribute Group Definition those
// ARE its two attribute properties. For a Complex Type Definition they are
// §3.4.2.4 clause 2 — "the {attribute uses} of the attribute groups ·resolved·
// to by the ·actual value·s of the ref [attribute] of the <attributeGroup>
// [children]" — and §3.4.2.5 clause 1's ·complete wildcard·, which that clause
// defines by §3.6.2.2's rules.
//
// Both need the REFERENCED definition, which is reachable only once the schema
// set is assembled, and §3.6.2.1 takes the TRANSITIVE closure: "An
// <attributeGroup> element involved in such a reference cycle maps to a component
// whose {attribute uses} and {attribute wildcard} properties reflect all the
// <attribute> and <any> elements contained in, or referred to (directly or
// indirectly) by elements in the cycle." So a container arrives here carrying its
// own content only — attributeContent, and its own <anyAttribute> as its
// wildcard — and leaves carrying both properties' §3.6.2 values, with
// attributeContent set to nil.
//
// Both properties are folded in ONE pass because both are defined over the same
// closure: the set of attribute groups a container reaches, each contributing its
// own <attribute>s and its own <anyAttribute>. Folding them apart would compute
// that closure twice and give it two encodings (STYLE T4).

// foldAttributeGroupReferences materialises §3.6.2.1 and §3.6.2.2 into every
// Attribute Group Definition and every Complex Type Definition the assembled
// Schema holds or owns.
//
// Its routes are every place a container can sit, and missing one leaves that
// container's refs unfolded and its properties under-approximated — an attribute
// its groups declare is then REJECTED on an instance, a pass→fail flip:
//
//   - {attribute group definitions}, re-seating attributeGroupIndex;
//   - the <redefine> ORIGINALS of attribute groups (attributeGroupRedefinitions),
//     in no property and no index, yet the B side of src-redefine clause 7.2.2
//     (checkAttributeGroupRedefinitions);
//   - {type definitions}, re-seating typeIndex, and each one's OWNED inline
//     {base type definition} — the src-expredef clause 1.1 original — at the
//     slot that owns it, through foldTypeAttributeContent;
//   - every complex type a DECLARATION owns, through ownedtypefold.go's
//     descent, which also reaches the <redefine>d <group> originals;
//   - the original a ResolvedAttributeGroup holds, which the closure expands in
//     place at the slot owning it, since no index reaches it.
//
// Every closure is computed from UNFOLDED definitions: the attribute group
// definitions are folded into pass-local slices and stored only after every
// complex type has been folded, so no closure reads another's half-written
// output (attributeUseFold does the same with its types slice). A complex type's
// fold reads no other complex type, so types are stored as they are folded.
//
// It is the first of the finalize pass's three mutations, beside
// foldAttributeUses (attributeusefold.go) and foldAttributeWildcards
// (attributewildcardfold.go), and rests on their footing: not a
// resolved-pointer cache but a property OVERWRITTEN with the value its mapping
// rule defines, and the AttributeGroupRefs that value was computed from are
// consumed rather than kept beside it — §3.6.2.1 says such a reference "does not
// correspond to any component as such".
//
// PHASE ORDER IS LOAD-BEARING. It must run after Phase A, which is what makes
// every AttributeGroupRef resolvable (resolveAttributeGroupRef); an
// unresolvable one is a src-resolve rejection there and never reaches this fold.
// It must run IMMEDIATELY BEFORE foldAttributeUses and foldAttributeWildcards:
// §3.4.2.4 clause 3 and §3.4.2.5 clause 2 both start from the clause 1 and 2
// value this fold produces, and foldAttributeUses retains exactly that value as
// ownAttributeUses for cos-ct-extends clause 1.5. The readers of attribute
// content ahead of it are the shared descent's two pre-fold phases, Phase A
// (resolveReferences) and checkSimpleTypeDerivations, and both walk the
// UNFOLDED content through attributeMembers — every reference and every owned
// original — so neither needs this fold's output. Phase B and Phase C read no
// attribute property.
//
// The fold runs AT MOST ONCE per container in effect: a container it has folded
// holds a nil attributeContent, which foldTypeAttributeContent and
// foldGroupAttributeContent answer by returning it unchanged, so the one
// component ownedtypefold.go reaches through two slots folds to one value.
// foldAttributeUses' "at most once" invariant is unaffected, since this fold
// writes no ownAttributeUses.
//
// Its rejections are §3.6.2.2's: the intersection CONSTRUCTS a wildcard, and
// intersectNamespaceConstraint and NewWildcard re-check w-props-correct. Neither
// can fail for validly-constructed operands (intersectNamespaceConstraint's own
// note), but the error is returned rather than swallowed, so a future divergence
// fails closed (STYLE P3).
func (s *Schema) foldAttributeGroupReferences() error {
	groups := make([]AttributeGroupDefinition, len(s.attributeGroups))
	for i, g := range s.attributeGroups {
		folded, err := s.foldGroupAttributeContent(g)
		if err != nil {
			return err
		}
		groups[i] = folded
	}
	originals := make([]AttributeGroupDefinition, len(s.attributeGroupRedefinitions))
	for i, r := range s.attributeGroupRedefinitions {
		folded, err := s.foldGroupAttributeContent(r.original)
		if err != nil {
			return err
		}
		originals[i] = folded
	}
	for i, t := range s.types {
		c, ok := t.(ComplexType)
		if !ok {
			continue // a simple type definition has no attribute content
		}
		folded, err := s.foldTypeAttributeContent(c)
		if err != nil {
			return err
		}
		s.types[i] = folded
		if folded.Name() != (QName{}) {
			s.typeIndex[folded.Name()] = folded
		}
	}
	owned := ownedTypeFold{fold: s.foldTypeAttributeContent}
	if err := owned.schema(s); err != nil {
		return err
	}
	for i, g := range groups {
		s.attributeGroups[i] = g
		s.attributeGroupIndex[g.Name()] = g
	}
	for i, g := range originals {
		s.attributeGroupRedefinitions[i].original = g
	}
	return nil
}

// foldGroupAttributeContent returns g with §3.6.2.1's {attribute uses} and
// §3.6.2.2's {attribute wildcard} stored and its content consumed. g's own name
// seeds the closure's reached set, so a reference chain leading back to g adds
// nothing g does not already hold — §3.6.2.1's cycle, taken as a closure.
func (s *Schema) foldGroupAttributeContent(g AttributeGroupDefinition) (AttributeGroupDefinition, error) {
	if g.attributeContent == nil {
		return g, nil // already folded, or never had content: attributeUses is complete
	}
	cl := newAttributeClosure(g.name)
	s.closeAttributeContent(cl, g.wildcard, g.hasWildcard, g.attributeContent)
	w, err := cl.wildcard(g.loc)
	if err != nil {
		return AttributeGroupDefinition{}, err
	}
	g.attributeUses, g.attributeContent = cl.uses, nil
	g.wildcard, g.hasWildcard = Wildcard{}, false
	if w != nil {
		g.wildcard, g.hasWildcard = *w, true
	}
	return g, nil
}

// foldTypeAttributeContent returns c with §3.4.2.4 clause 2's {attribute uses}
// and §3.4.2.5 clause 1's ·complete wildcard· stored and its content consumed,
// and with its OWNED inline {base type definition} folded the same way and
// re-seated — the src-expredef clause 1.1 original, which no index reaches and
// whose own references nothing else would fold (the #505 lesson).
//
// A type's closure starts from an empty reached set: a complex type has no name
// in the attribute group symbol space, so no reference can lead back to it.
func (s *Schema) foldTypeAttributeContent(c ComplexType) (ComplexType, error) {
	if base, owns := ownedComplexType(c.base); owns {
		folded, err := s.foldTypeAttributeContent(base)
		if err != nil {
			return ComplexType{}, err
		}
		c.base = InlineTypeDefinition{Definition: folded}
	}
	if c.attributeContent == nil {
		return c, nil // already folded, or never had content: attributeUses is complete
	}
	cl := newAttributeClosure(QName{})
	s.closeAttributeContent(cl, c.attributeWildcard, c.hasAttributeWildcard, c.attributeContent)
	w, err := cl.wildcard(c.loc)
	if err != nil {
		return ComplexType{}, err
	}
	c.attributeUses, c.attributeContent = cl.uses, nil
	c.attributeWildcard, c.hasAttributeWildcard = Wildcard{}, false
	if w != nil {
		c.attributeWildcard, c.hasAttributeWildcard = *w, true
	}
	return c, nil
}

// attributeClosure is one container's §3.6.2.1 closure under construction: the
// attribute uses it has collected, the wildcards in §3.6.2.2 pre-order, and the
// attribute group definitions already reached. It is pass-local and scoped to ONE
// container's closure — a fresh one per container — and is discarded when that
// container is folded.
type attributeClosure struct {
	// reached is the visited set §3.6.2.1 calls for, and the only one in this
	// package that guards a by-name edge: "Circular reference is not
	// disallowed", and ag-props-correct (§3.6.6) has no circularity clause, so a
	// reference cycle is a closure to take and never a failure to charge
	// (PRINCIPLES 9 has no rule ID to cite). It is a set union's membership test
	// — a group reached twice, round a cycle, down a diamond, or both named
	// explicitly and synthesized from <schema defaultAttributes>, contributes
	// once — and is read only by key, never ranged (STYLE D2).
	reached   map[QName]struct{}
	uses      []AttributeUse
	wildcards []Wildcard
}

// newAttributeClosure starts a closure for a container whose own name in the
// attribute group symbol space is self — an Attribute Group Definition's
// {name}, or the zero QName for a complex type, which has none.
func newAttributeClosure(self QName) *attributeClosure {
	cl := &attributeClosure{reached: map[QName]struct{}{}}
	if self != (QName{}) {
		cl.reached[self] = struct{}{}
	}
	return cl
}

// closeAttributeContent adds one container's own contribution to cl — its own
// <anyAttribute>, then its members in document order — descending each attribute
// group it references that cl has not reached yet. The order is the one the
// producer-time splice this replaced collected in (#479), which fixes two
// observable things: the index ag-props-correct clause 2 reports a duplicate at,
// and the wildcard whose {process contents} §3.6.2.2 takes — the container's own
// when present, otherwise the first referenced group's in document order.
//
//   - ResolvedAttributeUse contributes its use.
//   - AttributeGroupRef is looked up in attributeGroupIndex, never ranged; a
//     group already reached contributes nothing more.
//   - ResolvedAttributeGroup is expanded IN PLACE, with no reached test and
//     without marking its name: its definition is a <redefine>d original that
//     the index does not hold, so a reference by that name elsewhere means the
//     REDEFINITION, a different component. It is owned by the slot, so its
//     nesting is finite by construction and needs no guard.
//
// It reads each referenced definition's CURRENT content through
// attributeMembers, which is its unfolded content during this pass and its
// folded {attribute uses} for a definition that arrived already folded — whose
// wildcard is then its folded one too, so either way the definition contributes
// its whole closure.
func (s *Schema) closeAttributeContent(cl *attributeClosure, wildcard Wildcard, hasWildcard bool, content []AttributeUseOrGroupRef) {
	if hasWildcard {
		cl.wildcards = append(cl.wildcards, wildcard)
	}
	for _, m := range content {
		switch m := m.(type) {
		case ResolvedAttributeUse:
			cl.uses = append(cl.uses, m.Use)
		case AttributeGroupRef:
			if _, seen := cl.reached[m.Name]; seen {
				continue
			}
			cl.reached[m.Name] = struct{}{}
			g, ok := s.attributeGroupIndex[m.Name]
			if !ok {
				panic("xsd: closeAttributeContent: no attribute group definition named " + m.Name.String() +
					": Phase A's resolveAttributeGroupRef rejects every unresolvable <attributeGroup ref> before this fold runs")
			}
			s.closeAttributeContent(cl, g.wildcard, g.hasWildcard, attributeMembers(g.attributeContent, g.attributeUses))
		case ResolvedAttributeGroup:
			d := m.Definition
			s.closeAttributeContent(cl, d.wildcard, d.hasWildcard, attributeMembers(d.attributeContent, d.attributeUses))
		default:
			panic("xsd: closeAttributeContent: non-exhaustive AttributeUseOrGroupRef switch")
		}
	}
}

// wildcard is §3.6.2.2's {attribute wildcard} over the closure's collected
// wildcards W, in pre-order: ·absent· (nil) when W is empty; otherwise a wildcard
// whose {namespace constraint} is the Attribute Wildcard Intersection
// (cos-aw-intersect, §3.10.6.4) of every member — the spec's "more than two"
// case, a left fold of the binary primitive — and whose {process contents} is
// the first member's. That first member is the container's own <anyAttribute>
// when it has one (§3.6.2.2's ·local wildcard· L), and otherwise the first
// referenced group's, which is exactly §3.6.2.2's two non-empty cases.
//
// A single member is returned as itself: the intersection of one wildcard is
// that wildcard, and no new component is constructed. loc is the container's
// position, charged to the defensive rejection either constructor can return.
func (cl *attributeClosure) wildcard(loc xsderr.Loc) (*Wildcard, error) {
	if len(cl.wildcards) == 0 {
		return nil, nil
	}
	first := cl.wildcards[0]
	if len(cl.wildcards) == 1 {
		return &first, nil
	}
	nc := first.NamespaceConstraint()
	for _, w := range cl.wildcards[1:] {
		intersected, err := intersectNamespaceConstraint(loc, nc, w.NamespaceConstraint())
		if err != nil {
			return nil, err
		}
		nc = intersected
	}
	w, err := NewWildcard(loc, nc, first.ProcessContents())
	if err != nil {
		return nil, err
	}
	return &w, nil
}
