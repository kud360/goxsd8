package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleSrcResolve is Schema Representation Constraint: QName resolution (Schema
// Document) (Structures §3.17.6.2, id="src-resolve"): for a QName to resolve to
// a schema component of a specified kind, that component must be a member of the
// appropriate {…definitions}/{…declarations} property (clause 1, kind-specific:
// 1.1 type, 1.2 attribute decl, 1.3 element decl, 1.4 attribute group, 1.5 model
// group, 1.7 identity constraint) with a matching {name}/{target namespace}
// (clauses 2–3). A dangling reference (no such component) and a wrong-kind
// reference (the name exists only in another kind's table) are the SAME failure
// — the kind-specific lookup simply misses — so both are charged this rule,
// differing only in message. Clause 4 (namespace reachability from the referring
// document) is a distinct precondition that needs the schema-document import
// graph, which the compiled component model does not carry; it is out of #173's
// scope and left to the producer (#176).
const ruleSrcResolve xsderr.Rule = "src-resolve"

// anyTypeName is the expanded name of xs:anyType, the one Complex Type
// Definition permitted to be its own {base type definition} (§3.4.7,
// any-type-itself). This package models no built-in anyType anchor (it is
// outside the simple-type graph, per simpletype.go), so checkComplexBaseAcyclic
// detects the permitted self-derivation by this name rather than by pointer
// identity.
var anyTypeName = QName{Space: XMLSchemaNS, Local: "anyType"}

// resolve is the finalize resolution pass (Structures §3.17.3 assembly, §3.17.6.2
// src-resolve). It runs in two phases, both driven from the compiled Schema's
// document-order slices (STYLE D2 — the by-name indexes are used only for point
// lookups, never ranged to walk or to pick which failure to report, so the first
// reported failure is deterministic):
//
//   - Phase A (existence): walk every in-scope QName reference site and reject an
//     unresolvable target with src-resolve (or, for a keyref pointing at another
//     keyref, c-props-correct clause 1).
//   - Phase B (circularity): reject the spec-forbidden named circularities that
//     become representable only across the assembled set — the complex-type base
//     chain (ct-props-correct clause 3), <group ref> graph (mg-props-correct
//     clause 2), and substitution-group affiliation graph (e-props-correct clause
//     5).
//
// resolve stores nothing: it returns no value and mutates no component. A
// consumer that later wants the component behind a reference obtains it by a
// read-time index lookup (schema.Type/Element/Attribute), never from a resolved
// pointer this pass produced — that pointer would be state derivable from the
// QName plus the index (STYLE D3).
//
// An absent (zero) QName reference is skipped, not treated as dangling: the zero
// QName means "no reference", which src-resolve has nothing to resolve. Only a
// present-but-unresolvable QName is a failure.
//
// FOLLOW-COST ASYMMETRY (recorded deliberately, not silently): Phase A wires
// present-tense readers for the three Query views (Type/Element/Attribute
// Resolvers) and for modelGroupIndex + idcIndex. It reads NEITHER
// attributeGroupIndex NOR notationIndex — no in-scope reference resolves into
// them yet (an <attributeGroup ref> is inlined at producer mapping time with no
// persistent ref component, §3.6.2.1; nothing carries a NOTATION reference). And
// because resolution is validation-only, this package exposes no
// Schema.ModelGroup(name)/Schema.IdentityConstraint(name) accessor (STYLE 8 —
// export nothing without a consumer): the cost of following a ModelGroupRef or a
// keyref at read time is shifted onto the future Walker/Matcher and instance
// validator, which will need exactly those accessors. That temporary asymmetry
// with the three existing views is intentional, discharged by the consumer issue
// that adds the two accessors.
func (s *Schema) resolve() error {
	if err := s.resolveReferences(); err != nil {
		return err
	}
	if err := s.checkComplexBaseAcyclic(); err != nil {
		return err
	}
	if err := s.checkModelGroupsAcyclic(); err != nil {
		return err
	}
	return s.checkSubstitutionGroupsAcyclic()
}

// resolveReferences is Phase A: it walks every reference site in document order,
// rejecting the first unresolvable one. The three narrow-view resolvers below
// take a Resolver interface (STYLE T3) so they are testable against a fake; the
// model-group and keyref resolvers read the internal indexes directly, since no
// external consumer justifies minting a capability interface for them (STYLE 8).
func (s *Schema) resolveReferences() error {
	for _, t := range s.types {
		switch t := t.(type) {
		case ComplexType:
			if err := s.resolveComplexType(t); err != nil {
				return err
			}
		case *SimpleType:
			// A simple type's {base}/{item}/{member} slots are live pointers set
			// once at construction with no setter (simpletype.go), so they are
			// resolved and acyclic by construction — there is no QName-based
			// simple-type base reference in this package for a producer to even
			// misuse into a dangling ref or a cycle. Hence st-props-correct clause
			// 2 / cos-st-restricts clause 3.3 need no check here, and none of the
			// complex-type asymmetry applies: nothing to resolve.
		default:
			panic("xsd: resolveReferences: non-exhaustive TypeDefinition switch")
		}
	}
	for _, e := range s.elements {
		if err := s.resolveElementDecl(e); err != nil {
			return err
		}
	}
	for _, a := range s.attributes {
		if err := s.resolveAttributeDecl(a); err != nil {
			return err
		}
	}
	for _, mgd := range s.modelGroups {
		if err := s.resolveModelGroup(mgd.ModelGroup()); err != nil {
			return err
		}
	}
	for _, ic := range s.identityConstraints {
		if err := s.resolveKeyref(ic); err != nil {
			return err
		}
	}
	return nil
}

// resolveTypeName resolves a {type definition}/{base type definition} reference
// (src-resolve clause 1.1). A zero ref is absent and resolves to (nil, nil); a
// present-but-missing ref is rejected. ctx names the referring site for the
// message.
func resolveTypeName(r TypeResolver, ref QName, ctx string) (TypeDefinition, error) {
	if ref == (QName{}) {
		return nil, nil
	}
	t, ok := r.Type(ref)
	if !ok {
		return nil, xsderr.New(ruleSrcResolve, xsderr.Loc{},
			"%s references type %s, but no type definition with that expanded name is present in the schema (src-resolve clause 1.1)", ctx, ref)
	}
	return t, nil
}

// resolveElementName resolves an element-declaration reference (src-resolve
// clause 1.3): an <element ref> {term} or a {substitution group affiliations}
// member. A zero ref is absent and skipped.
func resolveElementName(r ElementResolver, ref QName, ctx string) error {
	if ref == (QName{}) {
		return nil
	}
	if _, ok := r.Element(ref); !ok {
		return xsderr.New(ruleSrcResolve, xsderr.Loc{},
			"%s references element declaration %s, but no element declaration with that expanded name is present in the schema (src-resolve clause 1.3)", ctx, ref)
	}
	return nil
}

// resolveAttributeName resolves an <attribute ref> {attribute declaration}
// reference (src-resolve clause 1.2). A zero ref is absent and skipped.
func resolveAttributeName(r AttributeResolver, ref QName, ctx string) error {
	if ref == (QName{}) {
		return nil
	}
	if _, ok := r.Attribute(ref); !ok {
		return xsderr.New(ruleSrcResolve, xsderr.Loc{},
			"%s references attribute declaration %s, but no attribute declaration with that expanded name is present in the schema (src-resolve clause 1.2)", ctx, ref)
	}
	return nil
}

// resolveModelGroupName resolves a <group ref> {term} reference (src-resolve
// clause 1.5) against modelGroupIndex directly. A zero ref is absent and skipped.
func (s *Schema) resolveModelGroupName(ref QName, ctx string) error {
	if ref == (QName{}) {
		return nil
	}
	if _, ok := s.modelGroupIndex[ref]; !ok {
		return xsderr.New(ruleSrcResolve, xsderr.Loc{},
			"%s references model group definition %s, but no model group definition with that expanded name is present in the schema (src-resolve clause 1.5)", ctx, ref)
	}
	return nil
}

// resolveKeyref resolves an identity constraint's {referenced key} (src-resolve
// clause 1.7) against idcIndex directly, but only for a keyref (a key/unique
// carries no reference). Beyond existence, it enforces the c-props-correct clause
// 1 tableau requirement that the referenced constraint be a key or unique, NOT
// another keyref: a same-kind lookup passes src-resolve (both are IDCs), so the
// keyref→keyref category mismatch is charged c-props-correct, not src-resolve.
func (s *Schema) resolveKeyref(ic IdentityConstraint) error {
	ref, isKeyref := ic.ReferencedKeyName()
	if !isKeyref || ref == (QName{}) {
		return nil
	}
	target, ok := s.idcIndex[ref]
	if !ok {
		return xsderr.New(ruleSrcResolve, xsderr.Loc{},
			"keyref %s references identity constraint %s, but no identity-constraint definition with that expanded name is present in the schema (src-resolve clause 1.7)", ic.Name(), ref)
	}
	if target.Category() == IdentityConstraintKeyref {
		return xsderr.New(ruleICProps, xsderr.Loc{},
			"keyref %s references %s, which is itself a keyref, but c-props-correct clause 1 requires a keyref's {referenced key} to be a key or unique", ic.Name(), ref)
	}
	return nil
}

// resolveComplexType descends a complex type's reference sites: its {base type
// definition} (clause 1.1), each {attribute use}, and its {content type}
// particle tree.
func (s *Schema) resolveComplexType(c ComplexType) error {
	if _, err := resolveTypeName(s, c.BaseTypeDefinitionName(), "complex type "+c.Name().String()+" {base type definition}"); err != nil {
		return err
	}
	for _, u := range c.AttributeUses() {
		if err := s.resolveAttributeUse(u); err != nil {
			return err
		}
	}
	switch ct := c.ContentType().(type) {
	case EmptyContent, SimpleContent:
		// Empty carries no reference. Simple carries a *SimpleType {simple type
		// definition}, a live pointer resolved by construction (not a QName ref).
	case ElementContent:
		return s.resolveParticle(ct.Particle)
	default:
		panic("xsd: resolveComplexType: non-exhaustive ContentType switch")
	}
	return nil
}

// resolveParticle descends a particle's {term}.
func (s *Schema) resolveParticle(p Particle) error {
	return s.resolveTerm(p.Term())
}

// resolveTerm resolves a particle's {term}: a <element ref> or <group ref> is a
// leaf resolved by a single lookup (never descended — that would cross into
// another component's own resolution), while an inline ResolvedTerm is descended.
func (s *Schema) resolveTerm(t TermOrRef) error {
	switch t := t.(type) {
	case ResolvedTerm:
		return s.resolveResolvedTerm(t.Term)
	case ElementDeclarationRef:
		return resolveElementName(s, t.Name, "particle {term} <element ref>")
	case ModelGroupRef:
		return s.resolveModelGroupName(t.Name, "particle {term} <group ref>")
	default:
		panic("xsd: resolveTerm: non-exhaustive TermOrRef switch")
	}
}

// resolveResolvedTerm descends an inline Term. A nil Term is unreachable on a
// value built through NewParticle (which rejects ResolvedTerm{Term: nil}); the
// default arm asserts the sealed-sum invariant.
func (s *Schema) resolveResolvedTerm(t Term) error {
	switch t := t.(type) {
	case ElementDeclaration:
		return s.resolveElementDecl(t)
	case ModelGroup:
		return s.resolveModelGroup(t)
	case Wildcard:
		return nil // a wildcard carries no QName reference
	default:
		panic("xsd: resolveResolvedTerm: non-exhaustive Term switch")
	}
}

// resolveModelGroup descends every particle of a model group in document order.
func (s *Schema) resolveModelGroup(g ModelGroup) error {
	for _, p := range g.Particles() {
		if err := s.resolveParticle(p); err != nil {
			return err
		}
	}
	return nil
}

// resolveAttributeUse resolves an attribute use's {attribute declaration}: an
// <attribute ref> is resolved by lookup (clause 1.2); a sibling local
// declaration is descended so its own {type definition} reference resolves.
func (s *Schema) resolveAttributeUse(u AttributeUse) error {
	switch d := u.AttributeDeclaration().(type) {
	case LocalAttributeDeclaration:
		return s.resolveAttributeDecl(d.Declaration)
	case AttributeDeclarationRef:
		return resolveAttributeName(s, d.Name, "attribute use <attribute ref>")
	default:
		panic("xsd: resolveAttributeUse: non-exhaustive AttributeDeclarationOrRef switch")
	}
}

// resolveElementDecl resolves an element declaration's reference sites: its
// {type definition} (clause 1.1), each {substitution group affiliations} member
// (clause 1.3), each type-table alternative's {type definition} (clause 1.1),
// and each nested {identity-constraint definitions} keyref (clause 1.7).
func (s *Schema) resolveElementDecl(e ElementDeclaration) error {
	if _, err := resolveTypeName(s, e.TypeDefinitionName(), "element declaration "+e.Name().String()+" {type definition}"); err != nil {
		return err
	}
	for _, aff := range e.SubstitutionGroupAffiliationNames() {
		if err := resolveElementName(s, aff, "element declaration "+e.Name().String()+" {substitution group affiliations}"); err != nil {
			return err
		}
	}
	if tt, ok := e.TypeTable(); ok {
		if err := s.resolveTypeTable(tt); err != nil {
			return err
		}
	}
	for _, ic := range e.IdentityConstraints() {
		if err := s.resolveKeyref(ic); err != nil {
			return err
		}
	}
	return nil
}

// resolveTypeTable resolves each Type Alternative's {type definition} reference
// (src-resolve clause 1.1; §3.12.3 maps the type/@type of a <alternative> via
// [·resolved·]). Both the {alternatives} members and the {default type
// definition} carry the same QName reference slot.
func (s *Schema) resolveTypeTable(tt TypeTable) error {
	for _, alt := range tt.Alternatives() {
		if _, err := resolveTypeName(s, alt.TypeDefinitionName(), "type alternative {type definition}"); err != nil {
			return err
		}
	}
	if _, err := resolveTypeName(s, tt.DefaultTypeDefinition().TypeDefinitionName(), "type table {default type definition}"); err != nil {
		return err
	}
	return nil
}

// resolveAttributeDecl resolves an attribute declaration's {type definition}
// reference (src-resolve clause 1.1). An attribute's type is always a simple
// type; the kind-specific lookup rejects a same-name non-type as dangling.
func (s *Schema) resolveAttributeDecl(a AttributeDeclaration) error {
	_, err := resolveTypeName(s, a.TypeDefinitionName(), "attribute declaration "+a.Name().String()+" {type definition}")
	return err
}

// checkComplexBaseAcyclic is Phase B for the complex-type base chain
// (ct-props-correct §3.4.6.1 clause 3): a complex type's {base type definition}
// chain must terminate, the sole permitted self-derivation being xs:anyType
// (§3.4.7). Because each type has at most one base, the "graph" is functional
// (out-degree ≤ 1), so a cycle is a repeated name on a single chain walk.
//
// The path map is a per-walk, finalize-scoped cycle guard: it lives entirely
// inside this function and is discarded when resolve returns (PRINCIPLES 5). It
// is NEVER threaded into any later traversal — doc.go promises the Walker needs
// "no visited set beyond the path-scoped guard" — so no runtime traversal
// inherits it.
//
// Roots are iterated in document order (STYLE D2); an anonymous root (zero name)
// is walked but never recorded (it can be no base's target, having no name to be
// referenced by), so the first reported cycle is deterministic.
func (s *Schema) checkComplexBaseAcyclic() error {
	for _, t := range s.types {
		ct, ok := t.(ComplexType)
		if !ok {
			continue // *SimpleType base chains are acyclic by construction
		}
		path := map[QName]bool{}
		cur := ct
		for {
			name := cur.Name()
			base := cur.BaseTypeDefinitionName()
			if base == (QName{}) {
				break // absent base ends the chain
			}
			if name == anyTypeName && base == anyTypeName {
				break // §3.4.7: xs:anyType is the one permitted self-based type
			}
			if name != (QName{}) {
				path[name] = true
			}
			next, ok := s.Type(base)
			if !ok {
				break // dangling base already reported by Phase A
			}
			nextCT, ok := next.(ComplexType)
			if !ok {
				break // base is a simple type: chain terminates
			}
			if path[base] {
				return xsderr.New(ruleCTPropsCorrect, xsderr.Loc{},
					"complex type %s participates in a circular {base type definition} chain, but ct-props-correct clause 3 forbids it (only xs:anyType may be its own base)", base)
			}
			cur = nextCT
		}
	}
	return nil
}

// checkModelGroupsAcyclic is Phase B for <group ref> circularity
// (mg-props-correct §3.8.6.1 clause 2, no-circular-groups): within the
// {particles} of a group there is no particle at any depth whose {term} is the
// group itself. Nodes are the top-level model group definitions; an edge M→N
// exists for each ModelGroupRef to N reachable through M's particle tree
// (inline model groups are descended; a ModelGroupRef is a leaf edge, resolved
// by visiting its target definition, not descended in place).
//
// The color map is a finalize-scoped cycle guard (0 unvisited, 1 on the current
// DFS stack, 2 finished): it lives only in this function and is discarded when
// resolve returns (PRINCIPLES 5), never threaded into any later traversal.
// Definitions are iterated, and each definition's out-refs collected, in
// document order (STYLE D2), so the first reported cycle is deterministic.
func (s *Schema) checkModelGroupsAcyclic() error {
	// The map's zero value is the implicit "unvisited" state; onStack/done are
	// the two recorded states.
	const (
		onStack = 1
		done    = 2
	)
	color := map[QName]int{}
	var visit func(name QName) error
	visit = func(name QName) error {
		switch color[name] {
		case done:
			return nil
		case onStack:
			return xsderr.New(ruleMgPropsCorrect, xsderr.Loc{},
				"model group definition %s participates in a circular <group ref> chain, but mg-props-correct clause 2 forbids circular groups", name)
		}
		color[name] = onStack
		if mgd, ok := s.modelGroupIndex[name]; ok {
			for _, ref := range groupRefsIn(mgd.ModelGroup()) {
				if err := visit(ref); err != nil {
					return err
				}
			}
		}
		color[name] = done
		return nil
	}
	for _, mgd := range s.modelGroups {
		if err := visit(mgd.Name()); err != nil {
			return err
		}
	}
	return nil
}

// groupRefsIn returns, in document order, the name of every <group ref>
// (ModelGroupRef) reachable through g's particle tree without descending into a
// referenced definition (that edge is followed by the DFS, not inlined here).
func groupRefsIn(g ModelGroup) []QName {
	var refs []QName
	for _, p := range g.Particles() {
		collectGroupRefs(p.Term(), &refs)
	}
	return refs
}

// collectGroupRefs appends the ModelGroupRef names in t's subtree (document
// order). Inline model groups are descended; an element (declaration or ref) is
// not — a nested element's own references belong to that element's resolution,
// not this group's reference graph.
func collectGroupRefs(t TermOrRef, refs *[]QName) {
	switch t := t.(type) {
	case ResolvedTerm:
		switch inner := t.Term.(type) {
		case ModelGroup:
			for _, p := range inner.Particles() {
				collectGroupRefs(p.Term(), refs)
			}
		case ElementDeclaration, Wildcard:
			// not a group reference
		default:
			panic("xsd: collectGroupRefs: non-exhaustive Term switch")
		}
	case ModelGroupRef:
		*refs = append(*refs, t.Name)
	case ElementDeclarationRef:
		// not a group reference
	default:
		panic("xsd: collectGroupRefs: non-exhaustive TermOrRef switch")
	}
}

// checkSubstitutionGroupsAcyclic is Phase B for substitution-group circularity
// (e-props-correct §3.3.6.1 clause 5): it must not be possible to return to an
// element E by repeatedly following any member of its {substitution group
// affiliations}. Nodes are the top-level element declarations; an edge E→F
// exists for each affiliation name F of E.
//
// The color map is a finalize-scoped cycle guard (same 0/1/2 scheme as
// checkModelGroupsAcyclic): it lives only in this function and is discarded when
// resolve returns (PRINCIPLES 5), never threaded into a later traversal.
// Elements are iterated, and each element's affiliations followed, in document
// order (STYLE D2), so the first reported cycle is deterministic.
func (s *Schema) checkSubstitutionGroupsAcyclic() error {
	// The map's zero value is the implicit "unvisited" state; onStack/done are
	// the two recorded states.
	const (
		onStack = 1
		done    = 2
	)
	color := map[QName]int{}
	var visit func(name QName) error
	visit = func(name QName) error {
		switch color[name] {
		case done:
			return nil
		case onStack:
			return xsderr.New(ruleEPropsCorrect, xsderr.Loc{},
				"element declaration %s participates in a circular {substitution group affiliations} chain, but e-props-correct clause 5 forbids circular substitution groups", name)
		}
		color[name] = onStack
		if e, ok := s.elementIndex[name]; ok {
			for _, aff := range e.SubstitutionGroupAffiliationNames() {
				if err := visit(aff); err != nil {
					return err
				}
			}
		}
		color[name] = done
		return nil
	}
	for _, e := range s.elements {
		if err := visit(e.Name()); err != nil {
			return err
		}
	}
	return nil
}
