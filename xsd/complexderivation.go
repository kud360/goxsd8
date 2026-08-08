package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleDerivationOKRestriction is Derivation Valid (Restriction, Complex)
// (Structures §3.4.6.3, id="derivation-ok-restriction"): the constraint every
// complex type T whose {derivation method} is restriction must satisfy against
// its {base type definition} B. Every clause charges this one rule and names its
// clause number (and, where the spec anchors one, its clause id — c-ran,
// c-vs-ctd-r) in the message text: the rule ID is not sub-anchored per clause,
// matching the convention complextype.go:5-12 already states. The clause anchors
// are NOT rule IDs — they carry no "Schema Component Constraint" label in the
// spec — so they are deliberately absent from xsderr's catalog.
const ruleDerivationOKRestriction xsderr.Rule = "derivation-ok-restriction"

// restrictionBlockingKeywords is the fixed blocking-keyword set
// derivation-ok-restriction clause 4 (c-vs-ctd-r) works under: {extension, list,
// union}. It reuses the shared DerivationMethod vocabulary (closedsets.go),
// which is documented as exactly key-val-sub-type's set K — the one place in
// this package where sharing a token type across properties is intended.
//
// Note what is NOT in it: restriction. That is why derivedOKSimple
// (derivation.go) is reusable unchanged for the simple half — cos-st-derived-ok
// reads its set S at exactly one place, clause 2.1 ("restriction is not in S, or
// in D.{base type definition}.{final}"), which is vacuously true for any set
// without restriction, i.e. identical to the empty-set behaviour derivedOKSimple
// already implements. The complex half is different: cos-ct-derived-ok's subset
// contains extension and IS load-bearing there, which is why derivedOKComplex
// takes the set as a parameter.
var restrictionBlockingKeywords = []DerivationMethod{DerivationExtension, DerivationList, DerivationUnion}

// checkComplexDerivations is Phase D of finalize: the complex-type derivation
// constraints that need the whole assembled set. It walks s.types in DOCUMENT
// ORDER (STYLE D2 — never typeIndex, so the first reported failure is
// deterministic) and, per Complex Type Definition, charges
//
//   - the ct-props-correct (§3.4.6.1) clauses that need a resolved
//     {base type definition} or a resolved {attribute declaration} — clause 2 and
//     clause 4 (checkCTPropsCorrectResolved); then
//   - derivation-ok-restriction (§3.4.6.3), for a restriction-derived type whose
//     base resolves to another complex type; then
//   - cos-ct-extends (§3.4.6.2), for an extension-derived type, over EITHER kind
//     of resolved base — its case 2 is the extension-of-a-simple-type path
//     (complexextension.go, #264).
//
// The last two are mutually exclusive by {derivation method}, so their order
// carries no verdict. ct-props-correct runs first for a given type so a simple
// base under {derivation method} = restriction is charged the precise clause 2
// rather than derivation-ok-restriction clause 1's coarser "B is a complex type
// definition" (STYLE E2, charge precision).
//
// PHASE ORDER IS LOAD-BEARING, three ways — this must run after Phases A, B and
// C, and the reasons are recorded here the way resolve.go:51-59 records Phase
// C's:
//
//   - after Phase A (existence): every {base type definition} and every
//     {attribute declaration} reference is known to resolve, so the lookups below
//     are hits rather than silent skips.
//   - after Phase B (circularity): derivedOKComplex and
//     locallyDeclaredAttributeType walk {base type definition} chains, and
//     effectivetotalrange.go follows <group ref> edges, with NO visited set. That
//     is licensed only because checkComplexBaseAcyclic and
//     checkModelGroupsAcyclic have already rejected a circular graph of each kind
//     (PRINCIPLES 9). The single spec-permitted self-derivation, xs:anyType
//     (§3.4.7), terminates every base walk here by an explicit anyTypeName test,
//     not by a guard.
//   - after Phase C (cos-element-consistent): clause 4 (c-vs-ctd-r) reads the
//     ·locally declared type· (key-ldtype) of an element expanded name within one
//     content model. Element Declarations Consistent is exactly what makes that a
//     FUNCTION rather than a relation — it forbids two same-named declarations in
//     one content model from carrying different {type definition}s — so without
//     it clause 4 would not be statable.
//
// ct-props-correct clause 5 ({content type}.{open content} non-absent ⇒
// {variety} element-only or mixed) gets no check here, deliberately: OpenContent
// is a field of ElementContent alone, whose Variety() is total over one bool and
// returns element-only or mixed and nothing else, so the forbidden state is
// unrepresentable and a runtime test would be dead code. See complextype.go's
// ruleCTPropsCorrect doc, which records the by-construction discharge.
//
// GAP(xsd): this walk quantifies over s.types, so an ANONYMOUS complex type —
// one reachable only through a slot that owns it — gets NO verdict from any
// constraint above. That covers the inline <complexType> of an element or
// attribute declaration (#438/#414) and, since #505, the src-expredef clause 1.1
// ORIGINAL a redefining complex type owns, which the spec makes a full component
// "as defined in Schema Component Details (§3)" and therefore subject to these
// same rules. The same absence leaves its own copy of the two folded properties
// at the producer's value (attributeusefold.go's storeFoldedAttributeUses). It
// UNDER-rejects in every case: a constraint that is not run refuses nothing.
// Closing it means a declaration-descending walk over the owning slots, which is
// one change for all three and is filed as #505's follow-up.
func (s *Schema) checkComplexDerivations() error {
	for _, t := range s.types {
		c, ok := t.(ComplexType)
		if !ok {
			continue // a *SimpleType derives under cos-st-restricts, checked at construction
		}
		if err := s.checkCTPropsCorrectResolved(c); err != nil {
			return err
		}
		if err := s.checkComplexTypeRestriction(c); err != nil {
			return err
		}
		if err := s.checkComplexTypeExtension(c); err != nil {
			return err
		}
	}
	return nil
}

// checkCTPropsCorrectResolved charges the two Complex Type Definition Properties
// Correct (§3.4.6.1, ct-props-correct) clauses that need something resolved:
//
//   - clause 2: if the {base type definition} is a simple type definition, the
//     {derivation method} must be extension.
//   - clause 4: no two distinct members of {attribute uses} have {attribute
//     declaration}s with the same expanded name.
//
// Clause 3 (base-chain acyclicity) is Phase B's (checkComplexBaseAcyclic) and
// clause 1's tableau shape is NewComplexType's; see complextype.go.
func (s *Schema) checkCTPropsCorrectResolved(c ComplexType) error {
	if err := s.checkSimpleBaseIsExtension(c); err != nil {
		return err
	}
	return checkAttributeUseNamesUnique(c)
}

// checkSimpleBaseIsExtension is ct-props-correct clause 2. An absent or
// unresolvable base is skipped: the reference was already charged src-resolve by
// Phase A, and an absent one has no variety to read. The base is reached through
// typeOf, so an anonymous inline base is decided rather than skipped — skipping
// it would wave the clause through unchecked for every redefining complex type
// (STYLE T4, #505).
func (s *Schema) checkSimpleBaseIsExtension(c ComplexType) error {
	base, ok := s.typeOf(c.Base())
	if !ok {
		return nil
	}
	if _, isSimple := base.(*SimpleType); !isSimple {
		return nil
	}
	if c.DerivationMethod() == DerivationExtension {
		return nil
	}
	return xsderr.New(ruleCTPropsCorrect, xsderr.Loc{},
		"complex type %s has the simple type %s as its {base type definition} but {derivation method} = %s, and ct-props-correct clause 2 requires extension", c.Name(), typeDefinitionLabel(base), c.DerivationMethod())
}

// checkAttributeUseNamesUnique is ct-props-correct clause 4. The uses are walked
// in document order and each expanded name tested against a seen-set map, so the
// FIRST duplicate by position is the one reported and the map never determines
// the verdict (STYLE D2/D1).
//
// The set walked is the MATERIALISED one — the type's own uses with the base's
// folded in (§3.4.2.4 clause 3, attributeusefold.go) — so a collision between a
// local use and an INHERITED one is charged here, which is the whole substance
// this clause has. Clauses 3.2.1 and 3.2.2 remove the collision for a restriction,
// so what reaches this check is clause 3.1's: an extension that re-declares a name
// its base already carries, identically or not. An extension may add attributes;
// it may not restate the base's.
//
// Clause 3.2.2 is load-bearing for this check, not decoration. A name a
// restriction B prohibits must leave B's set entirely; if it lingered, an
// extension of B declaring that name itself would collide with it and be rejected
// here for a duplicate its source never wrote (#401).
func checkAttributeUseNamesUnique(c ComplexType) error {
	seen := map[QName]bool{}
	for _, u := range c.attributeUses {
		name := attributeUseName(u)
		if seen[name] {
			return xsderr.New(ruleCTPropsCorrect, xsderr.Loc{},
				"complex type %s has two {attribute uses} whose {attribute declaration}s share the expanded name %s, but ct-props-correct clause 4 forbids it", c.Name(), name)
		}
		seen[name] = true
	}
	return nil
}

// checkComplexTypeRestriction runs derivation-ok-restriction (§3.4.6.3) for one
// complex type, after establishing the constraint's own preconditions:
// {derivation method} = restriction, and a {base type definition} that resolves
// to another Complex Type Definition.
//
// xs:anyType's self-derivation (§3.4.7, any-type-itself) is skipped: it is the
// one type the spec permits to be its own base, and running the constraint on it
// would compare it with itself.
//
// A simple base returns nil here rather than charging clause 1's "B is a complex
// type definition": checkCTPropsCorrectResolved has already run for this type and
// charges the more precise ct-props-correct clause 2.
func (s *Schema) checkComplexTypeRestriction(t ComplexType) error {
	if t.DerivationMethod() != DerivationRestriction {
		return nil
	}
	base, ok := s.typeOf(t.Base())
	if !ok {
		return nil // an absent base, or a dangling one Phase A already charged src-resolve
	}
	b, ok := base.(ComplexType)
	if !ok {
		return nil
	}
	if t.Name() == anyTypeName && b.Name() == anyTypeName {
		return nil
	}
	return s.checkDerivationOKRestriction(t, b)
}

// checkDerivationOKRestriction is Derivation Valid (Restriction, Complex)
// (§3.4.6.3) for a complex type t restricting the complex type b. All five
// clauses are checked, in spec order, so the first reported failure is
// deterministic (STYLE D1).
func (s *Schema) checkDerivationOKRestriction(t, b ComplexType) error {
	if err := checkRestrictionBaseFinal(t, b); err != nil {
		return err
	}
	if err := s.checkRestrictionContentType(t, b); err != nil {
		return err
	}
	if err := s.checkRestrictionAttributes(t, b); err != nil {
		return err
	}
	if err := s.checkRestrictionLocallyDeclaredTypes(t, b); err != nil {
		return err
	}
	return checkRestrictionAssertions(t, b)
}

// checkRestrictionAssertions is clause 5: B.{assertions} is a prefix of
// T.{assertions}. The relation, and why it is charged rather than assumed even
// though §3.4.2.1 clause 1's fold makes it hold by construction for every
// produced type, live in assertionprefix.go — cos-ct-extends clause 1.7 states
// the same test in the same words and shares the encoding.
func checkRestrictionAssertions(t, b ComplexType) error {
	if assertionsPrefix(b.assertions, t.assertions) {
		return nil
	}
	return xsderr.New(ruleDerivationOKRestriction, t.Loc(),
		"complex type %s restricts %s, but %s's {assertions} (%d) are not a prefix of %s's (%d), which derivation-ok-restriction clause 5 requires: §3.4.2.1 clause 1 places the base's assertions, in order, ahead of the type's own <assert> children",
		t.Name(), b.Name(), b.Name(), len(b.assertions), t.Name(), len(t.assertions))
}

// checkRestrictionBaseFinal is clause 1: B's {final} must not contain
// restriction.
func checkRestrictionBaseFinal(t, b ComplexType) error {
	if !finalContains(b.final, DerivationRestriction) {
		return nil
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s, but %s has restriction in its {final}, which derivation-ok-restriction clause 1 forbids", t.Name(), b.Name(), b.Name())
}

// checkRestrictionContentType is clause 2, a DISJUNCTION over four branches: the
// first satisfied one discharges the clause (happy path left, no else — STYLE
// S1/S2).
//
//   - 2.1: B is ·xs:anyType·, tested by expanded name against anyTypeName
//     (resolve.go), since this package models no anyType anchor to compare by
//     identity.
//   - 2.2: T's {content type} is simple and either its {simple type definition}
//     is validly derived from B's (cos-st-derived-ok) or B is mixed with an
//     ·emptiable· {particle}.
//   - 2.3: T's {content type} is empty and either B's is too, or B is
//     element-only/mixed with an ·emptiable· {particle}.
//   - 2.4: the variety/mixed match of 2.4.1, then 2.4.2's delegate.
//
// The message names every branch that could have carried the type, because a
// disjunction has no single failing clause to charge: 2.4.2's own two conditions
// (cos-content-act-restrict clause 1's language containment and clause 2's
// ctr-child-type-subsumption) are not reported apart, which is why
// contentTypeRestricts answers a bool rather than an error (contentrestricts.go).
func (s *Schema) checkRestrictionContentType(t, b ComplexType) error {
	if b.Name() == anyTypeName {
		return nil // clause 2.1
	}
	if s.restrictionSimpleContentOK(t, b) {
		return nil // clause 2.2
	}
	if s.restrictionEmptyContentOK(t, b) {
		return nil // clause 2.3
	}
	if s.restrictionComplexContentOK(t.ContentType(), b.ContentType()) {
		return nil // clause 2.4
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s, but its %s {content type} is not a valid restriction of the base's %s {content type} under any branch of derivation-ok-restriction clause 2 (2.1 base is xs:anyType, 2.2 simple content, 2.3 empty content, 2.4.1 element-only/mixed match, 2.4.2 the content model ·restricts· the base's per cos-content-act-restrict §3.4.6.4)", t.Name(), b.Name(), t.ContentType().Variety(), b.ContentType().Variety())
}

// restrictionSimpleContentOK is clause 2.2: 2.2.1 T's {content type}.{variety} is
// simple, AND one of 2.2.2.1 (ST validly derived from SB per cos-st-derived-ok,
// §3.16.6.3) or 2.2.2.2 (B is mixed and B's {particle} is ·emptiable·).
func (s *Schema) restrictionSimpleContentOK(t, b ComplexType) bool {
	tc, ok := t.ContentType().(SimpleContent)
	if !ok {
		return false // clause 2.2.1
	}
	if bc, ok := b.ContentType().(SimpleContent); ok && derivedOKSimple(tc.SimpleType, bc.SimpleType) {
		return true // clause 2.2.2.1
	}
	bc, ok := b.ContentType().(ElementContent)
	if !ok || !bc.Mixed {
		return false
	}
	return s.particleEmptiable(bc.Particle) // clause 2.2.2.2
}

// restrictionEmptyContentOK is clause 2.3: 2.3.1 T's {content type}.{variety} is
// empty, AND one of 2.3.2.1 (B's is empty) or 2.3.2.2 (B's is element-only or
// mixed and B's {particle} is ·emptiable· per cos-group-emptiable, §3.9.6.3).
func (s *Schema) restrictionEmptyContentOK(t, b ComplexType) bool {
	if _, ok := t.ContentType().(EmptyContent); !ok {
		return false // clause 2.3.1
	}
	if _, ok := b.ContentType().(EmptyContent); ok {
		return true // clause 2.3.2.1
	}
	bc, ok := b.ContentType().(ElementContent)
	if !ok {
		return false
	}
	return s.particleEmptiable(bc.Particle) // clause 2.3.2.2
}

// restrictionComplexContentOK is clause 2.4: the 2.4.1 variety match, decided
// STRICTLY, followed by 2.4.2's delegate.
func (s *Schema) restrictionComplexContentOK(tct, bct ContentType) bool {
	if !restrictionVarietyPairOK(tct, bct) {
		return false
	}
	return s.contentTypeRestricts(tct, bct)
}

// restrictionVarietyPairOK is clause 2.4.1: 2.4.1.1 T's {variety} is
// element-only and B's is element-only or mixed, or 2.4.1.2 both are mixed. No
// leniency applies here — §3.4.6.3's implementation-defined provisional
// acceptance is scoped to clause 2.4.2 alone.
func restrictionVarietyPairOK(tct, bct ContentType) bool {
	tc, ok := tct.(ElementContent)
	if !ok {
		return false
	}
	bc, ok := bct.(ElementContent)
	if !ok {
		return false
	}
	if !tc.Mixed {
		return true // clause 2.4.1.1: B is element-only or mixed, and it is one of them
	}
	return bc.Mixed // clause 2.4.1.2
}

// checkRestrictionAttributes is clause 3 (c-ran): for every element information
// item whose [attributes] satisfy cvc-complex-type (§3.4.4.2) clauses 2 and 3
// with respect to T, they must satisfy the same clauses with respect to B, and
// B's ·default binding· for each attribute must ·subsume· T's (loc-testSubP).
// Rendered statically over the component model, that is three obligations:
//
//   - every expanded name T admits through an {attribute use} must be admitted
//     by B at all (cvc-complex-type clause 2.1: otherwise an element carrying it
//     is valid against T and not against B), and B's binding for it must
//     ·subsume· T's, which for an {attribute use} is loc-testSubP clause 5;
//   - every expanded name T admits through its {attribute wildcard} must be
//     admitted by B too (cvc-complex-type clause 2.2, c-avaw —
//     checkRestrictionAttributeWildcard);
//   - every attribute B marks {required} must stay required in T
//     (cvc-complex-type clause 3, checkRestrictionRequiredAttributes).
//
// The three run in that order, which is cvc-complex-type's own: clause 2's two
// sub-cases before clause 3's. The uses are walked in document order, so the
// first reported failure is deterministic (STYLE D2).
//
// Both quantifications range over the MATERIALISED sets (§3.4.2.4 clause 3,
// attributeusefold.go), which is load-bearing in BOTH directions. In the reject
// direction the fold on B is what keeps a valid schema valid: in a chain
// A(@x) ← B(inheriting @x, re-declaring nothing) ← C(re-declaring @x), x is in
// B.{attribute uses} by inheritance, and charging C for it would be a FALSE
// REJECT. In the charge direction the fold on B is what makes
// checkRestrictionRequiredAttributes see a required attribute T inherits from
// higher up the chain rather than only the ones B declares itself.
func (s *Schema) checkRestrictionAttributes(t, b ComplexType) error {
	for _, u := range t.attributeUses {
		name := attributeUseName(u)
		general, ok := s.attributeDefaultBinding(b, name)
		if !ok {
			return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
				"complex type %s restricts %s but declares an attribute use for %s, which %s neither declares nor admits through an {attribute wildcard}, so an element valid against the restriction can carry an attribute the base rejects (derivation-ok-restriction clause 3, c-ran)", t.Name(), b.Name(), name, b.Name())
		}
		if err := s.checkBindingSubsumes(name, t, b, general, attributeUseBinding{use: u}); err != nil {
			return err
		}
	}
	if err := checkRestrictionAttributeWildcard(t, b); err != nil {
		return err
	}
	return checkRestrictionRequiredAttributes(t, b)
}

// checkRestrictionAttributeWildcard is the cvc-complex-type clause 2.2 (c-avaw)
// half of c-ran: the names an element valid against T may carry WITHOUT a
// matching {attribute use} are exactly those T's {attribute wildcard} admits, and
// B must admit each of them too — through its own {attribute wildcard}, since no
// finite {attribute uses} set covers the open name set a wildcard admits.
//
// So the obligation is a relation between the two wildcards, and it is decided by
// wildcardSubset — Wildcard Subset (§3.10.6.2, cos-ns-subset), the ONE encoding of
// that relation in this package (namespaceconstraint_subset.go, shared with the
// element-side transition test in contentrestricts.go; STYLE T4). The direction
// is T's constraint as sub and B's as super: T may admit fewer names than B, never
// more. A T with no {attribute wildcard} admits nothing this way and discharges
// the clause vacuously, which is why the absent case returns before B is read.
//
// Both sides are read off the type directly and neither walks a base chain,
// because §3.4.2.5 makes that exact. A restriction's {attribute wildcard} is its
// ·complete wildcard· — its own <anyAttribute>, clause 2.1 — and an extension's is
// already the union of its own with its base's, materialised at finalize
// (attributewildcardfold.go, clause 2.2). Before that fold, a B that inherited its
// wildcard read as having none and this check FALSELY rejected its restrictions;
// the fold is what makes the comparison sound, not merely more complete.
//
// GAP(xsd): the comparison is an over-approximation in the FAIL-CLOSED
// direction, owned by #430. cvc-complex-type clause 2.2 is reached only
// "otherwise", i.e. for a name with no matching {attribute use} in T, so a name
// T holds a use for is outside the quantification even when T's wildcard also
// admits it. wildcardSubset decides cos-ns-subset over {namespace constraint}s
// alone and knows nothing of {attribute uses}, so a subset failure caused SOLELY
// by a QName in B.{attribute wildcard}.{namespace constraint}.{disallowed names}
// that B (and therefore T, by §3.4.2.4 clause 3.2) declares an {attribute use}
// for is charged here though clause 3 does not require it.
//
// The error returned below has exactly one consumer chain, and every link
// propagates a non-nil error as a rejection rather than reading it for anything
// else: checkRestrictionAttributes → Schema.checkComplexTypeRestriction →
// Schema.checkComplexDerivations → Schema.resolve → SchemaBuilder.Finalize (and
// FinalizeWith), which returns it to the caller in place of a *Schema. So the
// over-charge is a FALSE REJECT of a VALID schema, not a missed rejection: a
// restriction whose base disallows BY NAME an attribute both types govern with an
// {attribute use} — B declaring <xs:attribute name="foo"/> alongside
// <anyAttribute namespace="##any" notQName="foo"/>, T restricting B with
// <anyAttribute namespace="##any"/> and inheriting the foo use — is rejected by
// Finalize today. Reachable from plain source syntax; no W3C suite case has the
// shape, which is why neither the gate nor the ratchet measures it.
//
// Narrowing it here would mean re-deriving cos-ns-subset with a name-set
// parameter — the same relation in two encodings, which is what T4 forbids and
// what #262 declined to build. #430 owns the choice between that and a
// pre-filter of B's {disallowed names} against T's {attribute uses} at this call
// site; nothing is narrowed yet, so the false reject above stands as described.
func checkRestrictionAttributeWildcard(t, b ComplexType) error {
	tw, has := t.AttributeWildcard()
	if !has {
		return nil
	}
	bw, baseHas := b.AttributeWildcard()
	if !baseHas {
		return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
			"complex type %s restricts %s and declares an {attribute wildcard}, but %s has none, so an element valid against the restriction can carry a wildcard-admitted attribute the base rejects (derivation-ok-restriction clause 3, c-ran, via cvc-complex-type clause 2.2, c-avaw)", t.Name(), b.Name(), b.Name())
	}
	if wildcardSubset(tw.NamespaceConstraint(), bw.NamespaceConstraint()) {
		return nil
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s but its {attribute wildcard} admits expanded names %s's does not, so an element valid against the restriction can carry a wildcard-admitted attribute the base rejects (derivation-ok-restriction clause 3, c-ran, via cvc-complex-type clause 2.2, c-avaw, and cos-ns-subset)", t.Name(), b.Name(), b.Name())
}

// checkRestrictionRequiredAttributes is the cvc-complex-type clause 3 half of
// c-ran: an attribute B marks {required} must also be required in T, or an
// attribute set that satisfies clause 3 with respect to T by omitting it fails
// with respect to B.
//
// Both sides are the MATERIALISED {attribute uses} (§3.4.2.4 clause 3), so the
// quantification is exact rather than an under-approximation. What the fold buys
// is reach on the B side: a name B holds by inheritance from higher up the chain
// is now a member of B.{attribute uses} and is compared, where before only the
// uses the producer mapped onto B itself were.
//
// The two ways the clause can fail are both charged. A name T relaxes to optional
// is one. A base-required name with NO member in T at all is the other, and it is
// live: clause 3.2 inherits every base use T does not declare itself, so the only
// way a required name goes missing is clause 3.2.2's <attribute use="prohibited">
// blocking it (attributeusefold.go) — which is exactly the shape this half is
// meant to charge, a restriction that prohibits an attribute its base requires.
func checkRestrictionRequiredAttributes(t, b ComplexType) error {
	for _, bu := range b.attributeUses {
		if !bu.Required() {
			continue
		}
		name := attributeUseName(bu)
		tu, ok := findAttributeUse(t.attributeUses, name)
		if !ok {
			return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
				"complex type %s restricts %s but its {attribute uses} carry no use for attribute %s, which the base requires, so an element omitting it is valid against the restriction and not against the base (derivation-ok-restriction clause 3, c-ran, via cvc-complex-type clause 3)", t.Name(), b.Name(), name)
		}
		if tu.Required() {
			continue
		}
		return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
			"complex type %s restricts %s but declares attribute %s as optional where the base requires it, so an element omitting it is valid against the restriction and not against the base (derivation-ok-restriction clause 3, c-ran, via cvc-complex-type clause 3)", t.Name(), b.Name(), name)
	}
	return nil
}

// findAttributeUse returns the member of a {attribute uses} set whose {attribute
// declaration} carries the expanded name, scanning in document order so the first
// match is deterministic (STYLE D1). It takes the set rather than the Complex
// Type Definition because the §3.4.2.4 clause 3 fold needs the same scan over a
// set that is still being built (attributeusefold.go) — one encoding, not two
// (STYLE T4).
func findAttributeUse(uses []AttributeUse, name QName) (AttributeUse, bool) {
	for _, u := range uses {
		if attributeUseName(u) == name {
			return u, true
		}
	}
	return AttributeUse{}, false
}

// locallyDeclaredTypeCheck bundles everything that differs between the two
// constraints quantifying over ·locally declared types· (key-ldtype, §3.4.6.4):
// derivation-ok-restriction clause 4 (c-vs-ctd-r) and cos-ct-extends clause 1.6
// (c-vs-ctd-e). The two clauses state the SAME relation over the same walk and
// differ only in the blocking-keyword set they pass to ·validly substitutable·
// and in the prose that names them, so they share one encoding of the walk
// rather than a fifth and sixth near-copy of it (STYLE T4).
//
// blocked is the load-bearing field: restriction passes
// restrictionBlockingKeywords, extension passes the EMPTY
// extensionBlockingKeywords (complexextension.go). limitation renders that same
// fact in the message, so a reader can see which set was in force without
// finding the call site.
type locallyDeclaredTypeCheck struct {
	rule       xsderr.Rule        // the rule the failure is charged to
	blocked    []DerivationMethod // the keywords ·validly substitutable· is subject to
	verb       string             // "restricts" | "extends"
	relation   string             // "restriction" | "extension"
	limitation string             // how the message names the blocking set
	clause     string             // the clause and clause anchor named in the message
}

// checkRestrictionLocallyDeclaredTypes is clause 4 (c-vs-ctd-r): for any element
// or attribute information item, its ·locally declared type· (key-ldtype) within
// T must be ·validly substitutable· (key-val-sub-type) for its ·locally declared
// type· within B, subject to the blocking keywords {extension, list, union}, IF
// the item has one in both.
//
// Both halves are quantified over what T declares — the attribute half over
// T.{attribute uses}, the element half over the declarations T's content model
// ·contains· — because a name with no ·locally declared type· in T fails the
// clause's "in both" precondition and is vacuous.
func (s *Schema) checkRestrictionLocallyDeclaredTypes(t, b ComplexType) error {
	return s.checkLocallyDeclaredTypes(t, b, locallyDeclaredTypeCheck{
		rule:       ruleDerivationOKRestriction,
		blocked:    restrictionBlockingKeywords,
		verb:       "restricts",
		relation:   "restriction",
		limitation: "subject to {extension, list, union}",
		clause:     "derivation-ok-restriction clause 4, c-vs-ctd-r",
	})
}

// checkLocallyDeclaredTypes runs one key-ldtype quantification — the attribute
// half then the element half — under the parameters k names.
func (s *Schema) checkLocallyDeclaredTypes(t, b ComplexType, k locallyDeclaredTypeCheck) error {
	if err := s.checkLocallyDeclaredAttributeTypes(t, b, k); err != nil {
		return err
	}
	return s.checkLocallyDeclaredElementTypes(t, b, k)
}

// checkLocallyDeclaredAttributeTypes is the attribute half (key-ldt-att).
func (s *Schema) checkLocallyDeclaredAttributeTypes(t, b ComplexType, k locallyDeclaredTypeCheck) error {
	for _, u := range t.attributeUses {
		name := attributeUseName(u)
		within, ok := s.locallyDeclaredAttributeType(t, name)
		if !ok {
			continue
		}
		base, ok := s.locallyDeclaredAttributeType(b, name)
		if !ok {
			continue // no ·locally declared type· in B: the clause's precondition fails
		}
		if s.validlySubstitutable(within, base, k.blocked) {
			continue
		}
		return xsderr.New(k.rule, xsderr.Loc{},
			"complex type %s %s %s, but the ·locally declared type· %s of attribute %s within the %s is not ·validly substitutable· for the base's %s %s (%s)", t.Name(), k.verb, b.Name(), typeDefinitionLabel(within), name, k.relation, typeDefinitionLabel(base), k.limitation, k.clause)
	}
	return nil
}

// checkLocallyDeclaredElementTypes is the element half (key-ldt-elem).
// Declarations are visited in the document order the content-model gatherer
// yields (STYLE D2).
func (s *Schema) checkLocallyDeclaredElementTypes(t, b ComplexType, k locallyDeclaredTypeCheck) error {
	for _, e := range s.contentModelDeclarations(t) {
		name := e.decl.Name()
		within, ok := s.typeOf(e.decl.TypeDefinition())
		if !ok {
			continue // absent or unresolvable: not decidable by this clause
		}
		base, ok := s.locallyDeclaredElementType(b, name)
		if !ok {
			continue // no ·locally declared type· in B: the clause's precondition fails
		}
		if s.validlySubstitutable(within, base, k.blocked) {
			continue
		}
		return xsderr.New(k.rule, xsderr.Loc{},
			"complex type %s %s %s, but the ·locally declared type· %s of element %s within the %s is not ·validly substitutable· for the base's %s %s (%s)", t.Name(), k.verb, b.Name(), typeDefinitionLabel(within), name, k.relation, typeDefinitionLabel(base), k.limitation, k.clause)
	}
	return nil
}

// locallyDeclaredAttributeType is the ·locally declared type· of an attribute
// within a Complex Type Definition (§3.4.6.4, key-ldt-att): case 1, ·absent· when
// the type is ·xs:anyType·; case 2, the {type definition} of the {attribute
// declaration} of a matching {attribute use}; case 3, otherwise the same question
// asked of the {base type definition}.
//
// The chain walk carries no visited set (Phase B licenses it) and the xs:anyType
// case-1 test terminates the one self-derivation §3.4.7 permits.
func (s *Schema) locallyDeclaredAttributeType(c ComplexType, name QName) (TypeDefinition, bool) {
	for {
		if c.Name() == anyTypeName {
			return nil, false // case 1
		}
		if u, ok := findAttributeUse(c.attributeUses, name); ok {
			d, ok := s.attributeUseDeclaration(u)
			if !ok {
				return nil, false
			}
			return s.typeOf(d.TypeDefinition()) // case 2
		}
		next, ok := s.baseComplexType(c) // case 3
		if !ok {
			return nil, false
		}
		c = next
	}
}

// locallyDeclaredElementType is the ·locally declared type· of an element within
// a Complex Type Definition (§3.4.6.4, key-ldt-elem), with the same three cases
// as key-ldt-att over the ·content model· rather than {attribute uses}.
//
// It deliberately does NOT fold in ·implicitly contained· substitution-group
// members. Including them would only ever add pairs to compare and so could only
// ever cause a false reject; excluding them under-approximates the element set,
// which is fail-open.
func (s *Schema) locallyDeclaredElementType(c ComplexType, name QName) (TypeDefinition, bool) {
	for {
		if c.Name() == anyTypeName {
			return nil, false // case 1
		}
		for _, e := range s.contentModelDeclarations(c) {
			if e.decl.Name() == name {
				return s.typeOf(e.decl.TypeDefinition()) // case 2
			}
		}
		next, ok := s.baseComplexType(c) // case 3
		if !ok {
			return nil, false
		}
		c = next
	}
}

// contentModelDeclarations returns, in document order, the element declarations
// c's ·content model· ·contains· directly or indirectly. It reuses
// elementconsistent.go's gatherer rather than writing a second content-model walk
// (STYLE T4); the component keys that gatherer assigns de-duplicate a declaration
// reached by several paths, which is exactly the identity key-ldtype needs.
func (s *Schema) contentModelDeclarations(c ComplexType) []containedElement {
	ec, ok := c.ContentType().(ElementContent)
	if !ok {
		return nil
	}
	var contents groupContents
	s.gatherTermContents(ec.Particle.Term(), c.Name(), "", &contents)
	return contents.elements
}

// baseComplexType is typeOf (typedefinition.go) narrowed to a COMPLEX base: it
// resolves c's {base type definition} when it is another Complex Type
// Definition, and is false for an absent, unresolvable or simple base — the
// three ways key-ldtype's case-3 recursion terminates without an answer. It
// re-derives no lookup of its own, so an anonymous inline base (src-expredef
// clause 1.1) is followed here exactly as a named one is (STYLE T4).
func (s *Schema) baseComplexType(c ComplexType) (ComplexType, bool) {
	base, ok := s.typeOf(c.Base())
	if !ok {
		return ComplexType{}, false
	}
	b, ok := base.(ComplexType)
	return b, ok
}

// validlySubstitutable is ·validly substitutable· (§3.4.6.4, key-val-sub-type):
// a type definition sub is validly substitutable for super subject to a set of
// blocking keywords, by one of three cases —
//
//   - both complex: sub is validly ·derived· from super subject to the UNION of
//     blocked and super.{prohibited substitutions}, per cos-ct-derived-ok;
//   - sub complex, super simple: sub is validly derived from super subject to
//     blocked, per cos-ct-derived-ok;
//   - sub simple: sub is validly derived from super subject to blocked, per
//     cos-st-derived-ok.
//
// A simple sub against the complex super ·xs:anyType· is answered true up front.
// Every simple type IS derived from xs:anyType — xs:anySimpleType's {base type
// definition} is xs:anyType — but this package models no anyType node inside the
// simple-type graph (simpletype.go), so derivedOKSimple cannot see the last hop.
// Answering false there would false-reject the extremely common shape of a
// restriction that types an element the base left untyped (a bare <element>
// defaults to xs:anyType, §3.3.2.1 case 4).
func (s *Schema) validlySubstitutable(sub, super TypeDefinition, blocked []DerivationMethod) bool {
	switch sup := super.(type) {
	case ComplexType:
		if sup.Name() == anyTypeName {
			return true
		}
		sc, ok := sub.(ComplexType)
		if !ok {
			return false // a simple type is derived from no complex type but xs:anyType
		}
		return s.derivedOKComplex(sc, super, unionDerivationMethods(blocked, sup.prohibitedSubstitutions))
	case *SimpleType:
		if sc, ok := sub.(ComplexType); ok {
			return s.derivedOKComplex(sc, super, blocked)
		}
		ss, ok := sub.(*SimpleType)
		return ok && derivedOKSimple(ss, sup)
	default:
		panic("xsd: validlySubstitutable: non-exhaustive TypeDefinition switch")
	}
}

// derivedOKComplex is Type Derivation OK (Complex) (§3.4.6.5,
// cos-ct-derived-ok): whether the complex type d is validly ·derived· from the
// type definition b subject to the blocking keywords in blocked. blocked is
// narrowed to the {extension, restriction} subset the constraint is defined over
// before it is read, per DerivationMethod's own doc (each consuming context
// admits its own subset).
//
// The clauses are applied on each step of the {base type definition} walk, which
// is what clause 2.3.2.1's "as defined by this constraint" recursion amounts to:
// clause 2.1 (B = D) first, then clause 1 (D's {derivation method} not in the
// subset), then clause 2.2 (B = D.{base type definition}), then clause 2.3
// (D.{base type definition} is not xs:anyType and is itself validly derived).
// The walk carries no visited set; see checkComplexDerivations for the licence.
//
// Clause 2.1's component identity is decided by expanded name for named types.
// Two ANONYMOUS types both present as the zero QName and are reported as NOT
// identical, exactly the licence §3.4.6.5's no-identity Note grants — the same
// call typeAlternativesEquivalent already makes for key-equiv-ta clause 5.
//
// The walk follows the {base type definition} SLOT through typeOf, both arms.
// Terminating at an InlineTypeDefinition instead would answer FALSE for every
// type derived through a redefining complex type — an over-REJECT, not a
// conservative answer, because a false here is what makes an instance
// validation fail (#505).
func (s *Schema) derivedOKComplex(d ComplexType, b TypeDefinition, blocked []DerivationMethod) bool {
	subset := complexBlockingSubset(blocked)
	for {
		if sameTypeDefinition(d, b) {
			return true // clause 2.1
		}
		if containsDerivationMethod(subset, d.DerivationMethod()) {
			return false // clause 1
		}
		base, ok := s.typeOf(d.Base())
		if !ok {
			return false // an absent base, or a dangling one Phase A already charged
		}
		if sameTypeDefinition(base, b) {
			return true // clause 2.2
		}
		if typeDefinitionName(base) == anyTypeName {
			return false // clause 2.3.1
		}
		next, ok := base.(ComplexType)
		if !ok {
			// clause 2.3.2.2: D's base is simple, so cos-st-derived-ok decides.
			ds, dOK := base.(*SimpleType)
			bs, bOK := b.(*SimpleType)
			return dOK && bOK && derivedOKSimple(ds, bs)
		}
		d = next // clause 2.3.2.1
	}
}

// complexBlockingSubset narrows a blocking-keyword set to the {extension,
// restriction} subset cos-ct-derived-ok is defined over, preserving order so the
// result is deterministic (STYLE D2).
func complexBlockingSubset(blocked []DerivationMethod) []DerivationMethod {
	var subset []DerivationMethod
	for _, m := range blocked {
		if m == DerivationExtension || m == DerivationRestriction {
			subset = append(subset, m)
		}
	}
	return subset
}

// unionDerivationMethods returns a ∪ b with a's order first and no duplicate,
// the set union key-val-sub-type's first bullet takes between the caller's
// blocking keywords and the super type's {prohibited substitutions}.
func unionDerivationMethods(a, b []DerivationMethod) []DerivationMethod {
	union := append([]DerivationMethod(nil), a...)
	for _, m := range b {
		if !containsDerivationMethod(union, m) {
			union = append(union, m)
		}
	}
	return union
}

// containsDerivationMethod reports whether the set contains m.
func containsDerivationMethod(set []DerivationMethod, m DerivationMethod) bool {
	for _, v := range set {
		if v == m {
			return true
		}
	}
	return false
}

// typeDefinitionName is the expanded {name} of a type definition, or the zero
// QName for an anonymous one and for a nil interface.
func typeDefinitionName(t TypeDefinition) QName {
	if t == nil {
		return QName{}
	}
	return t.Name()
}

// typeDefinitionLabel renders a type definition inside an error message. An
// anonymous one — what an inline <simpleType> yields — has the zero QName, which
// String()s to the empty string, so it is named as anonymous instead of leaving a
// hole in the message (STYLE E1: the reader must be able to find the construct).
func typeDefinitionLabel(t TypeDefinition) string {
	name := typeDefinitionName(t)
	if name == (QName{}) {
		return "an anonymous type definition"
	}
	return name.String()
}

// sameTypeDefinition decides the component identity cos-ct-derived-ok clause 2.1
// and cos-st-derived-ok clause 1 appeal to: two named top-level components with
// the same expanded name are the same definition. Two anonymous types are
// reported as different — the no-identity Note's licence — as is any pair where
// either side is absent.
func sameTypeDefinition(a, b TypeDefinition) bool {
	na, nb := typeDefinitionName(a), typeDefinitionName(b)
	if na == (QName{}) || nb == (QName{}) {
		return false
	}
	return na == nb
}
