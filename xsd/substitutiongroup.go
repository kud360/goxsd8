package xsd

// This file answers ·substitution group· membership (§3.3.6.4, key-eq) — "is the
// element declaration named member ·substitutable· for the one named head?" — for
// the three finalize-phase consumers that ask it: cvc-wildcard clause 3's
// ·implicit· containment (wildcardadmit.go), cos-nonambig's ·overlap· relation
// (particleattribution.go), and cos-element-consistent's ·implicitly contains·
// (elementconsistent.go). It was lifted out of wildcardadmit.go, whose subject is
// wildcards, once the second and third consumers appeared (STYLE T4/T5).
//
// It exposes TWO predicates rather than one because Substitution Group OK
// (Transitive) (§3.3.6.3, cos-equiv-derived-ok-rec) clause 2.3 is not implemented
// here — it needs a ·derivation· walk over RESOLVED type definitions, which an
// anonymous {type definition} (a zero QName) cannot supply — and clause 2.3 is a
// BLOCKING clause, so omitting it over-approximates membership while refusing
// membership whenever it might block under-approximates it. The two consumer
// families need opposite directions:
//
//   - mayBeInSubstitutionGroupOf OVER-approximates (omits clause 2.3 entirely).
//     Its consumer, cvc-wildcard clause 3, uses membership to DISALLOW a name, so
//     an over-broad group can only refuse a name the spec would admit — the
//     fail-CLOSED direction #51's acceptance criteria demand there.
//   - certainlyInSubstitutionGroupOf UNDER-approximates: it answers true only
//     when clause 2.3 is decidably vacuous. Its consumers, cos-nonambig and
//     cos-element-consistent, use membership to FIRE a schema-component
//     constraint, so an over-broad group would manufacture a rejection of a valid
//     schema.
//
// The two directions are not symmetric in cost. A false ACCEPT of a
// UPA-violating content model is graceful: §3.8.4.2's ·validation-path· machinery
// defines V(M) even for a non-deterministic model group, so instance validation
// still proceeds by a spec-defined rule. A false REJECT makes an entire schema
// unusable. An unimplemented sub-clause of a blocking condition therefore
// resolves in whichever polarity makes the enclosing constraint NOT fire.
//
// When clause 2.3 is implemented (it needs {type definition}s resolvable for
// anonymous types, which the component model does not yet give them) the two
// predicates collapse back into one exact one.
//
// Neither walk carries a cycle guard. That is licensed by Finalize's Phase B:
// checkSubstitutionGroupsAcyclic (e-props-correct clause 5) has already rejected
// a circular {substitution group affiliations} graph, so the only edge kind
// followed here is acyclic by construction on any *Schema that exists
// (PRINCIPLES 5). No {base type definition} edge is followed at all — §3.4.7 lets
// xs:anyType be its own base, which would reintroduce the self-loop this design
// relies on Phase B having ruled out.

// substitutionFacts are the two schema-global facts the membership predicates
// need. Both are computed ONCE per resolve() call and threaded as a parameter,
// never stored on *Schema: they are derivable from the compiled component set
// (STYLE D3), and a field on *Schema would be a second encoding of that set.
//
//   - anyAffiliation is false when no element declaration in the schema carries a
//     {substitution group affiliations} member. Every substitution group is then
//     the singleton {HEAD}, so both predicates answer false for member ≠ head and
//     their callers can skip the walk entirely.
//   - anyProhibitedSubstitutions is true when SOME complex type definition in the
//     schema has a non-empty {prohibited substitutions}. It is the cheap, exact
//     precondition for cos-equiv-derived-ok-rec clause 2.3 being vacuous: clause
//     2.3's blocking set is the union of H.{disallowed substitutions},
//     H.{type definition}.{prohibited substitutions}, and every intermediate
//     declared type's {prohibited substitutions}. If no complex type anywhere
//     carries a {prohibited substitutions} member, the second and third parts of
//     that union are empty whatever the ·derivation· turns out to be, so with an
//     empty H.{disallowed substitutions} the whole union is empty and clause 2.3
//     cannot block. Membership is then EXACT, not approximate.
type substitutionFacts struct {
	anyAffiliation             bool
	anyProhibitedSubstitutions bool
}

// substitutionFacts computes the schema-global facts above, reading the
// document-order slices (STYLE D2) rather than the by-name indexes: a map is
// never ranged even for a boolean, so no iteration order can reach a result.
func (s *Schema) substitutionFacts() substitutionFacts {
	var facts substitutionFacts
	for _, e := range s.elements {
		if len(e.substitutionGroupAffiliations) > 0 {
			facts.anyAffiliation = true
			break
		}
	}
	for _, t := range s.types {
		ct, ok := t.(ComplexType)
		if !ok {
			continue // a *SimpleType has no {prohibited substitutions}
		}
		if len(ct.prohibitedSubstitutions) > 0 {
			facts.anyProhibitedSubstitutions = true
			break
		}
	}
	return facts
}

// mayBeInSubstitutionGroupOf reports whether the top-level element declaration
// named member MAY be ·substitutable· for the declaration named head, i.e.
// whether member may be in head's ·substitution group· (§3.3.6.4, key-eq), per
// Substitution Group OK (Transitive) (§3.3.6.3, cos-equiv-derived-ok-rec):
//
//   - clause 1 (same declaration) is the caller's == test;
//   - clause 2.1: head's {disallowed substitutions} must not contain
//     substitution. The rule tests HEAD's property only — intermediate
//     declarations on the affiliation chain are not consulted, and the spec says
//     "intermediate" only in clause 2.3, about type definitions;
//   - clause 2.2: there is a chain of {substitution group affiliations} from
//     member to head.
//
// {abstract} plays NO part: it appears nowhere in cos-equiv-derived-ok-rec, so an
// abstract declaration genuinely in the substitution group still counts.
//
// Membership is decided by walking the affiliation chain UPWARD from member (one
// index point lookup per hop), not by scanning {element declarations} for every
// member of head's group: the question here is membership of ONE name, and the
// upward walk answers exactly that. No map is ever ranged, and no iteration order
// reaches the result (STYLE D2).
//
// The name records the direction of the approximation at every call site (STYLE
// T1): clause 2.3 is omitted, so the group computed here is a SUPERSET of the
// true one. Its sole consumer is cvc-wildcard clause 3, for which that is the
// fail-closed direction; a consumer that fires a constraint on membership must
// call certainlyInSubstitutionGroupOf instead.
func (s *Schema) mayBeInSubstitutionGroupOf(member, head QName) bool {
	h, ok := s.Element(head)
	if !ok {
		return false // a local declaration heads no substitution group
	}
	for _, m := range h.disallowedSubstitutions {
		if m == DerivationSubstitution {
			return false // clause 2.1
		}
	}
	return s.affiliationChainReaches(member, head)
}

// certainlyInSubstitutionGroupOf reports whether the top-level element
// declaration named member is ·substitutable· for the one named head with NO
// approximation in the accepting direction: it answers true only when
// cos-equiv-derived-ok-rec clauses 2.1 and 2.2 hold AND clause 2.3 is decidably
// vacuous, so a true answer is exactly the spec's answer.
//
// Clause 1 (same declaration) is not folded in here, matching
// mayBeInSubstitutionGroupOf; inSubstitutionGroup adds it for callers that need
// the reflexive relation.
//
// Clause 2.3 is vacuous exactly when its blocking union is empty: head's own
// {disallowed substitutions} empty (which also discharges clause 2.1, a stronger
// test than clause 2.1's "does not contain substitution"), and no complex type
// definition in the schema carrying a {prohibited substitutions} member — the
// schema-global fact facts.anyProhibitedSubstitutions, which covers head's own
// {type definition} and every intermediate declared type in the ·derivation·
// without walking it. That covers essentially every real schema.
func (s *Schema) certainlyInSubstitutionGroupOf(member, head QName, facts substitutionFacts) bool {
	if !facts.anyAffiliation {
		return false // no affiliation edge exists, so no chain can reach head
	}
	h, ok := s.Element(head)
	if !ok {
		return false // a local declaration heads no substitution group
	}
	if len(h.disallowedSubstitutions) > 0 {
		// GAP(subst): cos-equiv-derived-ok-rec clause 2.3 undecided without the
		// ·derivation· walk; reported as non-membership so the SCC never
		// manufactures a rejection.
		return false
	}
	if facts.anyProhibitedSubstitutions {
		// GAP(subst): cos-equiv-derived-ok-rec clause 2.3 undecided without the
		// ·derivation· walk; reported as non-membership so the SCC never
		// manufactures a rejection.
		return false
	}
	return s.affiliationChainReaches(member, head)
}

// inSubstitutionGroup is certainlyInSubstitutionGroupOf with
// cos-equiv-derived-ok-rec clause 1 (M and H are the same element declaration)
// folded in, for the callers that need the reflexive relation — Appendix J's
// bullet 3, which asks whether two heads' ·substitution groups· share a member
// and must count each head as a member of its own group.
//
// Name equality is component identity here: both operands name entries of
// {element declarations}, which sch-props-correct clause 2 keeps unique by
// expanded name.
func (s *Schema) inSubstitutionGroup(member, head QName, facts substitutionFacts) bool {
	if member == head {
		return true // clause 1
	}
	return s.certainlyInSubstitutionGroupOf(member, head, facts)
}

// affiliationChainReaches reports whether a chain of {substitution group
// affiliations} runs from the top-level element declaration named member to head
// (cos-equiv-derived-ok-rec clause 2.2). Affiliations are followed in document
// order (STYLE D2); the walk terminates without a visited set because Finalize
// rejected a circular affiliation graph (e-props-correct clause 5).
func (s *Schema) affiliationChainReaches(member, head QName) bool {
	m, ok := s.Element(member)
	if !ok {
		return false
	}
	for _, aff := range m.substitutionGroupAffiliations {
		if aff == head {
			return true
		}
		if s.affiliationChainReaches(aff, head) {
			return true
		}
	}
	return false
}
