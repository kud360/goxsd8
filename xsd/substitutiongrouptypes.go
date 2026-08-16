package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file is the fourth step of Phase D of the finalize resolution pass
// (resolve.go): Element Declaration Properties Correct (§3.3.6.1,
// e-props-correct) clause 4, anchor c-vs-sg — "For each member M of
// E.{substitution group affiliations}, E.{type definition} is ·validly
// substitutable· for M.{type definition}, subject to the blocking keywords in
// M.{substitution group exclusions}."
//
// It is the FIRST consumer of {substitution group exclusions}: the property has
// exactly this one reader in the whole specification, and until #395 nothing
// blocked on it (#264's elementDeclarationsIdentical compares it for property
// identity, which is not a blocking read).
//
// It is also the rule that requires a member's ·derivation· from its head to
// EXIST. cos-equiv-derived-ok-rec clause 2.3 does not — it is a blocking clause,
// and a derivation that does not exist blocks nothing, which is why
// derivationAdmitsSubstitution (substitutiongroup.go) is deliberately silent
// about it. Clause 4 is the separate rule that supplies the requirement, at
// schema-construction time rather than inside the membership predicate.

// checkSubstitutionGroupTypes is Phase D's fourth step: e-props-correct clause 4
// over every element declaration that carries a {substitution group
// affiliations}.
//
// PHASE PLACEMENT. The check reads RESOLVED type definitions on both sides and
// walks their {base type definition} chains through validlyDerived, so it
// needs Phase A's resolvability (a dangling type name is charged src-resolve
// there, and the ResolvedType lookups here are hits rather than silent skips) and
// Phase B's checkComplexBaseAcyclic (the walk inside derivedOKComplex carries NO
// visited set — PRINCIPLES 9 — and terminates only because a circular base chain
// was already rejected). Those are the same two dependencies checkComplexDerivations
// records, which is why this runs as a step of Phase D rather than as a phase of
// its own: Phase D is where every ·validly derived·/·validly substitutable·
// verdict in this package is charged, over the one cos-ct-derived-ok /
// cos-st-derived-ok engine pair, and clause 4 is that same engine reached
// through a different quantifier — element declarations rather than complex
// types. One engine, one phase.
//
// It runs AFTER the complex-type walk, not inside it: the loop below quantifies
// over s.elements, and folding an element-declaration constraint into a
// per-ComplexType loop would decide it once per type that happens to be
// referenced rather than once per declaration. Running after also keeps the more
// structural failure first (STYLE D1) — a type whose own derivation is invalid
// is charged before a declaration that merely points at it.
//
// It does NOT need Phase C, and could equally have run right after Phase B. It
// is placed after Phase D anyway, on the same "most structural failure first"
// ordering the resolve.go doc records for Phase E: a broken complex-type
// derivation is the defect a reader must fix, and the clause 4 failure it
// induces at every declaration typed by it is downstream of it.
//
// Quantifying over s.elements — the TOP-LEVEL declarations — is complete, not an
// approximation: e-props-correct clause 3 forces a non-empty {substitution group
// affiliations} to a global {scope}, and NewElementDeclaration rejects the
// combination at construction, so no local declaration can have a member for
// this clause to range over.
//
// Declarations are walked, and each declaration's affiliations followed, in
// document order (STYLE D2); no map is ranged, so the first reported failure is
// deterministic.
func (s *Schema) checkSubstitutionGroupTypes() error {
	for _, e := range s.elements {
		if err := s.checkElementSubstitutableForHeads(e); err != nil {
			return err
		}
	}
	return nil
}

// checkElementSubstitutableForHeads charges e-props-correct clause 4 for one
// element declaration E, against each DIRECT member of its {substitution group
// affiliations}.
//
// DIRECT, not transitive: the clause says "for each member M of E.{substitution
// group affiliations}", and the transitive closure belongs to
// cos-equiv-derived-ok-rec clause 2.2 (affiliationChainReaches,
// substitutiongroup.go), a different rule read by different consumers. Walking
// the chain here would be a stricter rule than the spec states — and an
// unnecessary one, since every declaration on a chain is itself an entry of
// s.elements and so has its own edge checked here.
//
// The blocking keyword set is M's {substitution group exclusions}, never E's,
// and never either declaration's {disallowed substitutions} — that property
// feeds cos-equiv-derived-ok-rec clause 2.1, a distinct rule over a distinct
// attribute — and never M.{type definition}.{prohibited substitutions} either.
// That last exclusion is a deliberate reading of clause 4 AGAINST its own words:
// the clause says ·validly substitutable·, whose complex/complex arm unions the
// target type's {prohibited substitutions} into the blocking set
// (key-val-sub-type), so a literal composition would reject every affiliation
// whose head TYPE blocks the member's derivation. Three passages say it must
// not, and the suite agrees:
//
//   - §3.3.3: "An empty {substitution group exclusions} allows a declaration to
//     be named in the {substitution group affiliations} of other element
//     declarations having the same declared {type definition} or some type
//     ·derived· therefrom" — naming the head is governed by the exclusions
//     ALONE, and an empty set excludes nothing;
//   - §3.4.1's prose on {prohibited substitutions} lists what that property
//     governs — an xsi:type override, ·substitution group· MEMBERSHIP inside a
//     model group, and a local element's type in a restriction — and declaring
//     an affiliation is not among them. Membership is cos-equiv-derived-ok-rec
//     clause 2.3, which unions the head type's set itself (substitutiongroup.go);
//   - XSD 1.0's clause 4 read "validly derived ... given the value of the
//     {substitution group exclusions}", with no union, and §3.3.3's prose
//     survived the 1.1 rewording unchanged.
//
// W3C sunData ElemDecl disallowedSubst00503m3/m4/m5 pin it end to end: a head
// typed by a complexType carrying block="restriction" (respectively
// "restriction extension"), with a member typed by a RESTRICTION of that type,
// is a VALID schema in which the member is merely not ·substitutable· for the
// head. The union reading rejects all three. So this charges ·validly derived·
// (validlyDerived, complexderivation.go) over the exclusions, which is clause 4
// with its one over-broad term removed and nothing else changed.
//
// The set is read off the unexported field rather than through
// SubstitutionGroupExclusions(), whose defensive copy would be allocated only to
// be discarded: validlyDerived never retains or mutates the slice
// (unionDerivationMethods and complexBlockingSubset both build fresh ones). Same
// reason resolveKeyref reads ic.fields directly.
//
// Two kinds of ·absent· component are SKIPPED rather than charged, both because
// there is no component to read a {type definition} from:
//
//   - an affiliation naming no declaration in the schema. §5.3 (Missing
//     Sub-components) makes it an ·absent· member and resolveElementDecl
//     deliberately does not reject it — W3C saxonData/Missing missing002 pins
//     that a substitutionGroup naming nothing is a VALID schema — and clause 1's
//     own "modulo the impact of Missing Sub-components (§5.3)" carries the same
//     licence into this tableau. Charging it here would reject the schema §5.3
//     says stands, and would double-report what Phase A already decided to allow;
//   - an absent or unresolvable {type definition} on either side. A dangling type
//     name was already charged src-resolve by Phase A, so reaching this point
//     with one is possible only for a genuinely ABSENT slot, which
//     derivationAdmitsSubstitution and declaredTypeRestricts (defaultbinding.go)
//     skip identically.
//
// Both skips are fail-open — they withhold a rejection, never invent one — and
// neither can mask a failure the spec states, since clause 4 quantifies over
// members and predicates over type definitions that must be there to be read.
//
// A THIRD case is not a skip but a clause-4 PASS decided before ·validly
// derived· is consulted: the two sides are the SAME component. Clause 2.1 of
// cos-ct-derived-ok is that identity test and it precedes the blocking-keyword
// test (derivedOKComplex charges 2.1 before clause 1), so answering it early
// changes no verdict — it only reaches an identity sameTypeDefinition cannot
// see. sameTypeDefinition compares expanded names and reports two ANONYMOUS
// types as different, which is §3.4.6.5's no-identity licence read correctly for
// the general case and WRONGLY here: §3.3.2.1 clause 3 makes a member's {type
// definition} the head's own component, and §3.4.6.5's no-identity Note names
// that very case — "when an element's type definition defaults to being the same
// type definition as that of its substitution-group head" — among the ones where
// component identity IS determined. Without the shortcut, a member sharing its
// head's inline anonymous type falls past clause 2.1, walks the base chain, and
// is FALSE-REJECTED under a rule the schema does not violate (#342).
func (s *Schema) checkElementSubstitutableForHeads(e ElementDeclaration) error {
	if len(e.substitutionGroupAffiliations) == 0 {
		return nil
	}
	memberType, ok := s.ResolvedType(e.TypeDefinition())
	if !ok {
		return nil
	}
	for _, aff := range e.substitutionGroupAffiliations {
		head, ok := s.Element(aff)
		if !ok {
			continue // an ·absent· member (§5.3): no component to be substitutable for
		}
		headType, ok := s.ResolvedType(head.TypeDefinition())
		if !ok {
			continue
		}
		if sameAnonymousTypeByConstruction(memberType, headType) {
			continue // cos-ct-derived-ok clause 2.1: one component, reached twice
		}
		derived, err := s.validlyDerived(memberType, headType, head.substitutionGroupExclusions)
		if err != nil {
			return err
		}
		if derived {
			continue
		}
		return xsderr.New(ruleEPropsCorrect, e.Loc(),
			"element declaration %s is typed %s, which is not validly ·derived· from %s, the {type definition} of the substitution group head %s it is affiliated to, subject to that head's {substitution group exclusions} %v; e-props-correct clause 4 (c-vs-sg) requires it to be",
			e.Name(), typeDefinitionLabel(memberType), typeDefinitionLabel(headType), aff, head.substitutionGroupExclusions)
	}
	return nil
}

// sameAnonymousTypeByConstruction reports whether a and b are ONE anonymous
// complex type definition reached through two slots — the identity §3.3.2.1
// dcl.elt.common clause 3 constructs when a member inherits its head's inline
// <complexType>, and the one §3.4.6.5's no-identity Note singles out as
// determined by the specification.
//
// It is decided on the {context} ComponentID (§3.4.1 ctd-context), because that
// is the identity mechanism this package already has for a component with no
// {name} to be compared by (componentid.go; Appendix G.1.11 says the property
// exists "to simplify testing for the identity of anonymous type definitions").
// Three properties of that choice are load-bearing:
//
//   - it is compared with ==, never reflect.DeepEqual, which is IDENTITY-BLIND
//     for a ComponentID and would report two distinct mints as equal (see
//     ComponentID);
//   - it survives the by-value copy a ComplexType undergoes on every read, so
//     the head's own slot and the member's inherited read compare equal;
//   - it is slot-shape independent, so ONE test covers both the direct edge (a
//     member affiliated to the owning head) and the chain edge (a member and an
//     intermediate head that both name the same terminal owner), with no arm
//     pattern-matching and no walk.
//
// A NAMED type has an ·absent· {context} per §3.4.1's XOR and answers false
// here; sameTypeDefinition already decides those by expanded name, correctly. A
// present {context} always carries a minted identity — checkComplexTypeContext
// rejects the zero one at construction — so equality here is never two unminted
// components accidentally matching.
//
// It deliberately does NOT generalize into sameTypeDefinition: comparing ALL
// anonymous types by {context} would change cos-ct-derived-ok clause 2.1
// everywhere it is charged, is unmodeled for *SimpleType (whose §3.16.1
// {context} this package does not carry, #206), and carries its own ratchet
// exposure. That widening is a separate landing.
func sameAnonymousTypeByConstruction(a, b TypeDefinition) bool {
	ida, aOK := anonymousComplexTypeIdentity(a)
	if !aOK {
		return false
	}
	idb, bOK := anonymousComplexTypeIdentity(b)
	if !bOK {
		return false
	}
	return ida == idb
}

// anonymousComplexTypeIdentity reads the {context} identity of an anonymous
// Complex Type Definition. ok is false for a *SimpleType (no modeled {context},
// #206) and for a NAMED complex type, whose {context} is ·absent· by §3.4.1's
// XOR and which is compared by expanded name instead.
func anonymousComplexTypeIdentity(t TypeDefinition) (ComponentID, bool) {
	c, isComplex := t.(ComplexType)
	if !isComplex {
		return ComponentID{}, false
	}
	context, present := c.Context()
	if !present {
		return ComponentID{}, false
	}
	return context.ID(), true
}
