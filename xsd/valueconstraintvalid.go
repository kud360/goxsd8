package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file is Phase E of the finalize resolution pass (resolve.go): the
// value-constraint validity constraints neither NewAttributeDeclaration,
// NewAttributeUse, nor any earlier phase can decide, because each needs a
// RESOLVED {type definition} or {attribute declaration}. Three clauses land here,
// all reading the same two components:
//
//   - a-props-correct (§3.2.6.1) clause 2: "if there is a {value constraint},
//     then it is a valid default with respect to the {type definition} as
//     defined in Simple Default Valid (§3.2.6.2)."
//   - au-props-correct (§3.5.6) clause 2: "If U.{value constraint} is not
//     ·absent·, then it is a valid default with respect to U.{attribute
//     declaration}.{type definition} as defined in Simple Default Valid."
//   - au-props-correct clause 3, below.
//
// The two clause 2s are ONE predicate — cos-valid-simple-default over a (type,
// value constraint) pair — differing only in which type and which constraint they
// pair. They therefore share one implementation, checkSimpleDefault (STYLE T4,
// #371); a second, parallel one for either call site would be a design failure.
// Both are gated on PRESENCE: with no {value constraint} the clause is not
// reached at all, which is not the same as reached-and-vacuously-satisfied.
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
// The two walks. A GLOBAL Attribute Declaration is charged a-props-correct clause
// 2 by its own walk over the schema's {attribute declarations}
// (checkAttributeDeclarationDefaults), never again per use that references it: one
// charge per component keeps the first reported failure deterministic and does not
// re-run the same verdict once per referencing site. A LOCAL declaration is not in
// that table at all — its sole owner is the AttributeUse holding it — so it is
// charged from the use side instead. Together the two reach every declaration
// exactly once; neither alone does.
//
// D4 (no traversal state): the walk below carries no visited set, exactly as
// Phase A's mirror walk does. It descends only BY-VALUE structure — a complex
// type's {attribute uses} and content-model particles, an element declaration's
// inline {type definition} — and never follows a by-name ref, which is what makes
// the structure a finite tree. It additionally inherits Phase B's acyclicity
// (checkComplexBaseAcyclic, checkModelGroupsAcyclic) for the by-name edges it
// deliberately does not take, so no cycle check is needed (PRINCIPLES 9).

// checkAttributeUseValueConstraints is Phase E's use-side walk: it charges
// au-props-correct clauses 2 and 3 — and, for a use owning a LOCAL declaration,
// that declaration's own a-props-correct clause 2 — against every Attribute Use
// the compiled schema holds, walking in document order so the first reported
// failure is deterministic (STYLE D1/D2 — no index map is ranged).
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
//
// Since #401 materialised §3.4.2.4 clause 3, an INHERITED use is a member here
// too, so it is re-checked at every type that inherits it and charged against
// THAT type's Loc rather than the ancestor's — deliberate, because clause 3 makes
// the use genuinely a property of the derived type and that is the position a
// reader is looking at. The extra passes cannot change the verdict: the walk is
// over a set, and a use that passed once passes again.
func (s *Schema) checkComplexTypeAttributeUses(c ComplexType) error {
	for _, u := range c.AttributeUses() {
		if err := s.checkAttributeUseValueConstraint(u, c.Loc(), complexTypeOwner(c)); err != nil {
			return err
		}
	}
	ct, ok := c.ContentType().(ElementContent)
	if !ok {
		return nil // Empty and Simple content carry no particle tree
	}
	return s.checkParticleAttributeUses(ct.Particle)
}

// complexTypeOwner renders c as the owner phrase of a rejection message. An
// inline <xs:complexType> has no {name} (the zero QName, whose String is ""),
// so naming it would leave a hole in the message; it is described by what it is
// instead.
func complexTypeOwner(c ComplexType) string {
	n := c.Name()
	if n == (QName{}) {
		return "anonymous complex type"
	}
	return "complex type " + n.String()
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

// checkAttributeUseValueConstraint charges au-props-correct (§3.5.6) clauses 2 and
// 3 against one Attribute Use, plus a-props-correct clause 2 against the LOCAL
// declaration a use owns. owner names the enclosing component and loc is its
// position: an Attribute Use is not one of the eight kinds that retain a Loc
// (doc.go), so the rejection is charged where a reader must edit — the complex
// type or attribute group that holds the use — and names the attribute in the
// message. The owned local declaration is charged at its OWN Loc, which it does
// retain.
//
// Clause 2 is decided first, in spec order, so a use violating both clauses
// reports the more basic failure: its {value constraint} is not a valid default
// at all, which makes "is it identical to the declaration's" moot. It is checked
// against the RESOLVED {attribute declaration}.{type definition} — never a type
// on the Use, which has none — and only when the use has its OWN {value
// constraint}, never the §3.5.4 ·effective value constraint· (that one is the
// declaration's, already charged a-props-correct clause 2 in its own right).
//
// Clause 3's antecedent is then read in the rule's own order. Both conjuncts must
// hold for it to bite: U must have its OWN {value constraint} (never the
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
	d, ok := s.attributeUseDeclaration(u)
	if !ok {
		return nil
	}
	// The Local variant's declaration is in no global table, so this use is the
	// only site that can charge it (see the two walks, above). The Ref variant's
	// target IS in that table and checkAttributeDeclarationDefaults charges it
	// there, exactly once, however many uses reference it.
	if local, isLocal := u.AttributeDeclaration().(LocalAttributeDeclaration); isLocal {
		if err := s.checkAttributeDeclarationValueConstraint(local.Declaration); err != nil {
			return err
		}
	}
	uvc, own := u.ValueConstraint()
	if !own {
		// Both clauses are gated on U having its OWN {value constraint} —
		// clause 2's "U.{value constraint} is not ·absent·" and clause 3's "U
		// itself has a {value constraint}" — so neither is reached.
		return nil
	}
	n := attributeUseName(u)
	t, hasType := s.simpleTypeOf(d.TypeDefinition())
	if hasType {
		if err := s.checkSimpleDefault(ruleAuPropsCorrect, loc, owner+" attribute "+n.String(), t, uvc); err != nil {
			return err
		}
	}
	dvc, present := d.ValueConstraint()
	if !present || dvc.Kind() != ValueFixed {
		return nil // the declaration is not fixed: clause 3 is discharged
	}
	if uvc.Kind() != ValueFixed {
		return xsderr.New(ruleAuPropsCorrect, loc,
			"%s gives attribute %s a %s value constraint, but the attribute declaration fixes it to %q, and au-props-correct clause 3 requires the use's {value constraint}.{variety} to be fixed too", owner, n, uvc.Kind(), dvc.LexicalForm())
	}
	if !hasType {
		return nil
	}
	identical, decided := s.valueSpace.Identical(t, uvc, t, dvc)
	if decided && !identical {
		return xsderr.New(ruleAuPropsCorrect, loc,
			"%s fixes attribute %s to %q, but the attribute declaration fixes it to %q, and au-props-correct clause 3 requires the two {value}s to be identical", owner, n, uvc.LexicalForm(), dvc.LexicalForm())
	}
	return nil
}

// checkAttributeDeclarationDefaults is Phase E's declaration-side walk: it charges
// a-props-correct (§3.2.6.1) clause 2 against every GLOBAL Attribute Declaration,
// in document order so the first reported failure is deterministic (STYLE D1/D2 —
// s.attributes is the document-ordered slice, not the by-name index).
//
// It walks the GLOBAL table only. A LocalAttributeDeclaration is never a member of
// it — the dcl.att.local mapping (§3.2.2.2) hands the declaration to its sibling
// Attribute Use, which is its sole owner (attributeuse.go) — so it is unreachable
// from here and is charged the same clause from the use side instead, by
// checkAttributeUseValueConstraint. Neither walk alone reaches every declaration:
// a global one may be referenced by no use at all, and a local one appears in no
// table.
func (s *Schema) checkAttributeDeclarationDefaults() error {
	for _, d := range s.attributes {
		if err := s.checkAttributeDeclarationValueConstraint(d); err != nil {
			return err
		}
	}
	return nil
}

// checkAttributeDeclarationValueConstraint charges a-props-correct (§3.2.6.1)
// clause 2 against one Attribute Declaration: "if there is a {value constraint},
// then it is a valid default with respect to the {type definition}". The
// declaration is charged at its own Loc, which an AttributeDeclaration retains
// (doc.go), so a reader is sent to the <xs:attribute> that wrote the default.
//
// Both gates accept rather than reject: no {value constraint} means the clause is
// not reached at all (never reached-and-satisfied), and a {type definition} that
// is absent, unresolvable, or complex is simpleTypeOf's documented "not decidable
// by this clause".
func (s *Schema) checkAttributeDeclarationValueConstraint(d AttributeDeclaration) error {
	dvc, present := d.ValueConstraint()
	if !present {
		return nil // "if there is a {value constraint}" fails: clause 2 is not reached
	}
	t, ok := s.simpleTypeOf(d.TypeDefinition())
	if !ok {
		return nil
	}
	return s.checkSimpleDefault(ruleAPropsCorrect, d.Loc(), "attribute declaration "+d.Name().String(), t, dvc)
}

// checkSimpleDefault is Simple Default Valid (§3.2.6.2, cos-valid-simple-default)
// — the ONE implementation of it in this package (STYLE T4), shared by
// a-props-correct clause 2 and au-props-correct clause 2, which differ only in
// which (type, value constraint) pair they hand it and which rule the failure is
// charged to. The clause phrase in the message is DERIVED from rule rather than
// passed alongside it: there are exactly two callers and rule already tells them
// apart, so a second parameter would make "ruleAPropsCorrect + au-props-correct
// clause 2" a representable, wrong state (STYLE D3).
//
// The verdict is the installed ValueSpace's, and an UNDECIDED verdict ACCEPTS.
// That is not laxity, it is the fail-open contract (ValueSpace, PRINCIPLES 20):
// undecided covers a type no backend mapping governs — xs:anySimpleType and
// xs:anyAtomicType included, which is what §3.2.2.2's third tier gives every
// typeless <xs:attribute default="…">, and for which Datatype Valid is
// unconditionally true — a QName/NOTATION-governed value space, whose lexical
// mapping needs namespace bindings a ValueConstraint does not carry (PRINCIPLES
// 19), and a construction-stage failure in the TYPE's facets, which is not a
// verdict about this value constraint at all.
//
// This is the first finalize-phase check that enters the installed value space's
// FACET pipeline rather than its Mapping alone, so it is also the first that can
// meet a type carrying a facet not applicable to it (cos-applicable-facets §4.1.5) —
// possible only for a *SimpleType assembled by calling this package's constructors
// directly, since builtin.CheckSimpleTypeRestriction discharges applicability for
// every type the PARSER builds. That fault is a fault of the TYPE, and the value
// space reports it undecided rather than as a verdict, so it lands in the accepting
// branch below: this clause never rejects a schema for it, and never crashes on it.
func (s *Schema) checkSimpleDefault(rule xsderr.Rule, loc xsderr.Loc, owner string, t *SimpleType, vc ValueConstraint) error {
	valid, decided := s.valueSpace.ValidDefault(t, vc)
	if !decided || valid {
		return nil
	}
	clause := "a-props-correct clause 2"
	if rule == ruleAuPropsCorrect {
		clause = "au-props-correct clause 2"
	}
	return xsderr.New(rule, loc,
		"%s has a {value constraint} of %q, which is not Datatype Valid with respect to its {type definition} (Datatypes §4.1.4 cvc-datatype-valid), so it is not a valid default (%s, cos-valid-simple-default §3.2.6.2)", owner, vc.LexicalForm(), clause)
}
