package xsd

// This file resolves the defined/sibling keyword half of a wildcard's
// {disallowed names} (§3.10.1) against the live declaration graph, implementing
// Item Valid (Wildcard) (Structures §3.10.4.1, cvc-wildcard) clauses 2-3.
//
// It lives on *Schema, not on Wildcard or NamespaceConstraint, because the
// declaration graph is the thing being consulted — the same placement
// resolve.go uses for every graph-consulting operation. Being a *Schema method
// also makes it unreachable before SchemaBuilder.Finalize, which is what
// licenses the walks below to carry NO cycle guard: Finalize has already
// rejected a circular <group ref> graph (mg-props-correct clause 2), a circular
// {substitution group affiliations} graph (e-props-correct clause 5) and a
// circular complex-type base chain (ct-props-correct clause 3 — reached through
// substitutiongroup.go's clause 2.3 walk), so every graph these walks follow is
// acyclic by construction on any *Schema that exists (PRINCIPLES 5; xsd/doc.go's
// "no visited set beyond the path-scoped guard").
//
// Nothing here is memoized. The sibling name set is per-(complex type,
// occurrence), not per-Wildcard-component: the same Wildcard value can be
// reached from two different complex types' content models, and caching the
// resolved set on the Wildcard would answer the second query with the first's
// content model. The containing type is therefore always a parameter.
//
// The two entry points return bool, not error, because cvc-wildcard is a LOCAL
// validity rule: in a content-model matcher "this wildcard does not admit this
// name" is a non-match (try the next particle), not a reportable fault — the
// reported fault is the enclosing cvc-complex-type clause. The rule ID each
// eventual M5 caller must cite is named in the doc comments below so the
// mapping from bool to error is not lost.

// allowsElementWildcardName reports whether the expanded name is admitted by the
// ELEMENT wildcard w occurring in containing's {content type} particle tree,
// implementing Item Valid (Wildcard) (§3.10.4.1, cvc-wildcard) in full for the
// element-wildcard case:
//
//   - clause 1: the name is ·valid· with respect to w.{namespace constraint},
//     delegated to Wildcard.AllowsName — the one canonical implementation of
//     cvc-wildcard-name (§3.10.4.2), never re-derived here;
//   - clause 2.1: if {disallowed names} contains defined, the name must not
//     ·resolve· to a top-level element declaration (§3.17.6.3,
//     cvc-resolve-instance, over §3.17.1's {element declarations}, which is
//     top-level by construction — no extra scope filter is needed);
//   - clause 3: if {disallowed names} contains sibling, the name must not
//     ·match· any element declaration ·contained· — directly, indirectly, or
//     implicitly — in containing's content model.
//
// A caller reporting a rejection cites cvc-wildcard, clause 2 or clause 3 as
// appropriate — NOT cvc-wildcard-name, which is clause 1's delegate and a
// different rule.
//
// Clause 3's remaining preconditions (3.3-3.6: the item has an element parent
// with the same validation context whose ·governing type definition· is the
// complex type containing w) are the CALLER's to establish: passing containing
// is the assertion that 3.6 holds. This method does not and cannot check them —
// it sees no instance item.
func (s *Schema) allowsElementWildcardName(w Wildcard, containing ComplexType, name QName) bool {
	if !w.AllowsName(name) {
		return false
	}
	c := w.namespaceConstraint
	if c.hasDisallowedNameKeyword(DisallowedNameDefined) {
		if _, declared := s.Element(name); declared {
			return false
		}
	}
	if c.hasDisallowedNameKeyword(DisallowedNameSibling) {
		return !s.contentModelContainsName(containing, name)
	}
	return true
}

// allowsAttributeWildcardName reports whether the expanded name is admitted by
// the ATTRIBUTE wildcard w, implementing Item Valid (Wildcard) (§3.10.4.1,
// cvc-wildcard) in full for the attribute-wildcard case: clause 1 (delegated to
// Wildcard.AllowsName, §3.10.4.2) and clause 2.2 (if {disallowed names} contains
// defined, the name must not ·resolve· to a top-level attribute declaration).
//
// A caller reporting a rejection cites cvc-wildcard clause 2 — not
// cvc-wildcard-name, which is clause 1's delegate and a different rule.
//
// There is deliberately no containing-type parameter: clause 3.2 restricts
// sibling to element wildcards and w-props-correct clause 5 forbids an attribute
// wildcard from carrying it at all (rejectSiblingOnAttributeWildcard enforces
// that at construction), so the sibling path is not merely unreached here — it
// is unrepresentable in this signature (STYLE T1). Nothing below re-checks it.
func (s *Schema) allowsAttributeWildcardName(w Wildcard, name QName) bool {
	if !w.AllowsName(name) {
		return false
	}
	if !w.namespaceConstraint.hasDisallowedNameKeyword(DisallowedNameDefined) {
		return true
	}
	_, declared := s.Attribute(name)
	return !declared
}

// contentModelContainsName reports whether name ·matches· an element declaration
// ·contained· in c's content model, whether ·directly·, ·indirectly·, or
// ·implicitly· (§3.10.4.1 clause 3; the containment definitions are §3.8.2/§3.9.1
// and the glossary "directly/indirectly/implicitly contains").
//
// It reads ONLY c.ContentType() — the effective {content type} particle tree —
// and never follows {base type definition}. That is not an optimization: §3.4.7
// (any-type-itself) permits xs:anyType to be its own base, so a base-chain walk
// would reintroduce exactly the self-loop this file's no-cycle-guard design
// relies on Finalize's acyclicity passes to have ruled out. A derived type's
// effective content model already folds in whatever the derivation contributes
// (§3.4.2.3.3), so the base chain has nothing further to add.
//
// An empty or simple {content type} contains no element declaration, and so does
// an {open content} wildcard: a Wildcard is not an element declaration.
func (s *Schema) contentModelContainsName(c ComplexType, name QName) bool {
	ct, ok := c.ContentType().(ElementContent)
	if !ok {
		return false
	}
	return s.particleContainsName(ct.Particle, name)
}

// particleContainsName reports whether name matches a declaration the particle
// ·contains· — its own {term} (·directly contains·) or anything reachable
// through it (·indirectly contains·).
func (s *Schema) particleContainsName(p Particle, name QName) bool {
	return s.termContainsName(p.Term(), name)
}

// termContainsName descends a particle's {term}. A <group ref> is followed
// through modelGroupIndex to the referenced definition's {model group} (§3.7.2,
// declare-namedModelGroup) — the index is read directly, as resolve.go's
// resolveModelGroupName and checkModelGroupsAcyclic do, so no
// Schema.ModelGroup accessor is minted for an in-package reader (STYLE T5).
// A ref that resolves to nothing contributes nothing: Finalize already rejected
// a dangling ModelGroupRef (src-resolve clause 1.5), so this is unreachable on
// a *Schema that exists, not a silent skip.
func (s *Schema) termContainsName(t TermOrRef, name QName) bool {
	switch t := t.(type) {
	case ResolvedTerm:
		return s.resolvedTermContainsName(t.Term, name)
	case ElementDeclarationRef:
		// An <element ref> always names a TOP-LEVEL declaration (§3.3.2.4,
		// ref.elt.global), so it heads a substitution group.
		return s.topLevelDeclarationMatchesName(t.Name, name)
	case ModelGroupRef:
		mgd, ok := s.modelGroupIndex[t.Name]
		if !ok {
			return false
		}
		return s.modelGroupContainsName(mgd.ModelGroup(), name)
	default:
		panic("xsd: termContainsName: non-exhaustive TermOrRef switch")
	}
}

// resolvedTermContainsName descends an inline {term}, exhaustively over the
// sealed Term sum. A Wildcard contributes nothing — clause 3 speaks of element
// declarations contained in the content model, and a wildcard is not one.
func (s *Schema) resolvedTermContainsName(t Term, name QName) bool {
	switch t := t.(type) {
	case ElementDeclaration:
		return s.inlineDeclarationMatchesName(t, name)
	case ModelGroup:
		return s.modelGroupContainsName(t, name)
	case Wildcard:
		return false
	default:
		panic("xsd: resolvedTermContainsName: non-exhaustive Term switch")
	}
}

// modelGroupContainsName descends every particle of a model group in document
// order (STYLE D2), so which declaration answers a query is deterministic.
func (s *Schema) modelGroupContainsName(g ModelGroup, name QName) bool {
	for _, p := range g.Particles() {
		if s.particleContainsName(p, name) {
			return true
		}
	}
	return false
}

// topLevelDeclarationMatchesName reports whether name is disallowed by the
// presence, in the content model, of the TOP-LEVEL element declaration named
// declared: either name ·matches· it (·key-en-match·, plain expanded-name
// equality — the ·directly·/·indirectly· contained case) or name is a member of
// its ·substitution group· (§3.3.6.4 — the ·implicitly· contained case, "a list
// of particles implicitly contains an element declaration if and only if a
// member of the list contains that element declaration in its ·substitution
// group·").
//
// ·match· here is key-en-match, NOT the substitution-aware key-e-d-match:
// substitution-group awareness enters through implicit containment expanding the
// candidate set, never through the match relation itself.
func (s *Schema) topLevelDeclarationMatchesName(declared, name QName) bool {
	if declared == name {
		return true
	}
	return s.inSubstitutionGroupOf(name, declared)
}

// inlineDeclarationMatchesName is topLevelDeclarationMatchesName for a
// declaration written INLINE in the content model, where the {scope} is known
// from the component itself. A local declaration heads no substitution group —
// §3.3.6.4 defines one for each declaration "in the {element declarations} of a
// schema", which §3.17.1 restricts to top-level declarations — so only the
// expanded-name ·match· applies to it, even when a same-named top-level
// declaration exists with members of its own.
func (s *Schema) inlineDeclarationMatchesName(d ElementDeclaration, name QName) bool {
	if d.Name() == name {
		return true
	}
	if d.ScopeVariety() != ScopeGlobal {
		return false
	}
	return s.inSubstitutionGroupOf(name, d.Name())
}
