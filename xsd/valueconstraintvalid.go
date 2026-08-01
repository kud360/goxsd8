package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file is Phase E of the finalize resolution pass (resolve.go): Attribute
// Use Correct (Structures §3.5.6, au-props-correct) CLAUSE 3, the half of that
// constraint neither NewAttributeUse nor any earlier phase can decide.
//
// Clause 3, verbatim: "If U.{attribute declaration} has {value constraint}.
// {variety} = fixed and U itself has a {value constraint}, then U.{value
// constraint}.{variety} = fixed and U.{value constraint}.{value} is identical to
// U.{attribute declaration}.{value constraint}.{value}."
//
// Two things put it here rather than in the constructor:
//
//   - the {attribute declaration} half. For the AttributeDeclarationRef variant
//     the declaration is a deferred QName at construction time, so its {value
//     constraint} is unreadable there; only a finalized Schema can resolve it.
//     NewAttributeUse therefore decides the variety half for the Local variant
//     ONLY, and this phase decides it for both — the rule text draws no variant
//     distinction, so the Local case is re-tested here (vacuously, since the
//     constructor already rejected it) rather than special-cased away (STYLE T4).
//   - the {value} half. {value} is an ·actual value· (key-vv §3.2.1), a member of
//     the type's value space, not the {lexical form} ValueConstraint carries — so
//     deciding "identical" needs a lexical→value mapping, which this package
//     cannot have (PRINCIPLES 1). It asks the installed ValueSpace instead, and
//     accepts whenever the answer is undecided.
//
// Clause 2 (Simple Default Valid, §3.2.6.2 cos-valid-simple-default) is a
// separate obligation on the use's OWN {value constraint} against the
// declaration's {type definition}; it is not implemented here and stays deferred
// to its own issue.
//
// D4 (no traversal state): the walk below carries no visited set, exactly as
// Phase A's mirror walk does. It descends only BY-VALUE structure — a complex
// type's {attribute uses} and content-model particles, an element declaration's
// inline {type definition} — and never follows a by-name ref, which is what makes
// the structure a finite tree. It additionally inherits Phase B's acyclicity
// (checkComplexBaseAcyclic, checkModelGroupsAcyclic) for the by-name edges it
// deliberately does not take, so no cycle check is needed (PRINCIPLES 5).

// checkAttributeUseValueConstraints is Phase E: it charges au-props-correct
// clause 3 against every Attribute Use the compiled schema holds, walking in
// document order so the first reported failure is deterministic (STYLE D1/D2 —
// no index map is ranged).
//
// The walk mirrors Phase A's descent site for site, because the two must reach
// the same attribute uses: top-level type definitions, then top-level element
// declarations (whose inline anonymous complex types carry attribute uses of
// their own, reached through resolveTypeDefinition's InlineTypeDefinition arm),
// then top-level model group definitions (whose particles can carry element
// declarations with inline complex types), and finally — where Phase A stops —
// top-level attribute group definitions.
//
// Phase A does NOT walk {attribute group definitions} (see resolve.go's
// FOLLOW-COST ASYMMETRY note): every <attributeGroup ref> is inlined at producer
// mapping time, so a group's uses are already folded into each complex type that
// references it and walking types alone would suffice for the parser path. This
// phase walks them anyway, because an UNREFERENCED group — and any group a
// SchemaBuilder caller adds directly — holds Attribute Use components the spec
// constrains all the same, and a folded use is simply re-tested with the same
// verdict. The price of going where Phase A did not is that a group's <attribute
// ref> was never vetted for resolvability: an unresolvable one is SKIPPED here
// (attributeUseDeclaration reports no declaration), not charged src-resolve,
// which is fail-open and never a false reject.
func (s *Schema) checkAttributeUseValueConstraints() error {
	for _, t := range s.types {
		c, ok := t.(ComplexType)
		if !ok {
			continue // a simple type has no {attribute uses}
		}
		if err := s.checkComplexTypeAttributeUses(c); err != nil {
			return err
		}
	}
	for _, e := range s.elements {
		if err := s.checkElementAttributeUses(e); err != nil {
			return err
		}
	}
	for _, mgd := range s.modelGroups {
		if err := s.checkModelGroupAttributeUses(mgd.ModelGroup()); err != nil {
			return err
		}
	}
	for _, g := range s.attributeGroups {
		for _, u := range g.AttributeUses() {
			if err := s.checkAttributeUseValueConstraint(u, g.Loc(), "attribute group definition "+g.Name().String()); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkComplexTypeAttributeUses charges clause 3 for c's own {attribute uses} and
// then descends c's {content type} particle tree, where a nested element
// declaration may carry an inline complex type with attribute uses of its own.
// The descent mirrors resolveComplexType's.
func (s *Schema) checkComplexTypeAttributeUses(c ComplexType) error {
	for _, u := range c.AttributeUses() {
		if err := s.checkAttributeUseValueConstraint(u, c.Loc(), "complex type "+c.Name().String()); err != nil {
			return err
		}
	}
	ct, ok := c.ContentType().(ElementContent)
	if !ok {
		return nil // Empty and Simple content carry no particle tree
	}
	return s.checkParticleAttributeUses(ct.Particle)
}

// checkElementAttributeUses descends an element declaration's inline {type
// definition}, mirroring resolveElementDecl/resolveTypeDefinition: a
// TypeDefinitionRef names a top-level type this phase already walked in its own
// right, so only the InlineTypeDefinition arm is descended.
func (s *Schema) checkElementAttributeUses(e ElementDeclaration) error {
	inline, ok := e.TypeDefinition().(InlineTypeDefinition)
	if !ok {
		return nil
	}
	c, ok := inline.Definition.(ComplexType)
	if !ok {
		return nil // an inline *SimpleType has no {attribute uses}
	}
	return s.checkComplexTypeAttributeUses(c)
}

// checkParticleAttributeUses descends one particle's {term}, mirroring
// resolveTerm: an <element ref>/<group ref> is a by-name leaf owned by the
// component it names, never descended here.
func (s *Schema) checkParticleAttributeUses(p Particle) error {
	t, ok := p.Term().(ResolvedTerm)
	if !ok {
		return nil
	}
	switch inner := t.Term.(type) {
	case ElementDeclaration:
		return s.checkElementAttributeUses(inner)
	case ModelGroup:
		return s.checkModelGroupAttributeUses(inner)
	case Wildcard:
		return nil // a wildcard carries no declaration
	default:
		panic("xsd: checkParticleAttributeUses: non-exhaustive Term switch")
	}
}

// checkModelGroupAttributeUses descends every particle of a model group in
// document order.
func (s *Schema) checkModelGroupAttributeUses(g ModelGroup) error {
	for _, p := range g.Particles() {
		if err := s.checkParticleAttributeUses(p); err != nil {
			return err
		}
	}
	return nil
}

// checkAttributeUseValueConstraint charges au-props-correct (§3.5.6) clause 3
// against one Attribute Use. owner names the enclosing component and loc is its
// position: an Attribute Use is not one of the eight kinds that retain a Loc
// (doc.go), so the rejection is charged where a reader must edit — the complex
// type or attribute group that holds the use — and names the attribute in the
// message.
//
// The antecedent is read in the rule's own order. Both conjuncts must hold for
// the clause to bite: U must have its OWN {value constraint} (never the §3.5.4
// ·effective value constraint·, which would make the test vacuously self-
// comparing), and U.{attribute declaration}.{value constraint} must be present
// with {variety} = fixed.
//
// Two consequents follow, in spec order so the first reported failure is
// deterministic (STYLE D1): the use's {variety} must be fixed, and its {value}
// must be identical to the declaration's. Both {value}s belong to ONE value
// space — the declaration's {type definition} governs each, since the use's
// {value constraint} constrains that very declaration — so the same type is
// handed to ValueSpace.Identical on both sides.
//
// Every non-decision is fail-open and never a false reject: an unresolvable
// declaration (a dangling Ref, already charged src-resolve by Phase A on the
// paths Phase A walks), a {type definition} that is absent, unresolvable, or
// complex, and an undecided ValueSpace verdict all accept.
func (s *Schema) checkAttributeUseValueConstraint(u AttributeUse, loc xsderr.Loc, owner string) error {
	uvc, own := u.ValueConstraint()
	if !own {
		return nil // "U itself has a {value constraint}" fails: clause 3 is discharged
	}
	d, ok := s.attributeUseDeclaration(u)
	if !ok {
		return nil
	}
	dvc, present := d.ValueConstraint()
	if !present || dvc.Kind() != ValueFixed {
		return nil // the declaration is not fixed: clause 3 is discharged
	}
	n := attributeUseName(u)
	if uvc.Kind() != ValueFixed {
		return xsderr.New(ruleAuPropsCorrect, loc,
			"%s gives attribute %s a %s value constraint, but the attribute declaration fixes it to %q, and au-props-correct clause 3 requires the use's {value constraint}.{variety} to be fixed too", owner, n, uvc.Kind(), dvc.LexicalForm())
	}
	t, ok := s.simpleTypeOf(d.TypeDefinition())
	if !ok {
		return nil
	}
	identical, decided := s.valueSpace.Identical(t, uvc, t, dvc)
	if decided && !identical {
		return xsderr.New(ruleAuPropsCorrect, loc,
			"%s fixes attribute %s to %q, but the attribute declaration fixes it to %q, and au-props-correct clause 3 requires the two {value}s to be identical", owner, n, uvc.LexicalForm(), dvc.LexicalForm())
	}
	return nil
}
