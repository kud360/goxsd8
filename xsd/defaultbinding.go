package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file renders two §3.4.6.4 definitions statically over the component
// model: ·default binding· (key-dft-binding) and ·subsumes· (loc-testSubP).
//
// Both are worded over INFORMATION ITEMS at assessment time. Their ATTRIBUTE
// half is nonetheless decidable per (Complex Type Definition, expanded name)
// without any instance, and that static rendering is the whole of what
// derivation-ok-restriction clause 3 (c-ran) needs: the clause quantifies over
// every element information item whose attributes are valid against T, which for
// attributes reduces to "for each expanded name T admits, compare the bindings".
// The ELEMENT half (key-dft-binding case 1, loc-testSubP clause 4) is
// cos-content-act-restrict's business and belongs to #263; the sum below carries
// its variant so #263 adds a case, not a type.

// defaultBinding is the ·default binding· a Content Type or Complex Type
// Definition gives an information item (Structures §3.4.6.4, key-dft-binding):
// "an Element Declaration, an Attribute Use, or one of the keywords strict, lax,
// or skip". The spec's six cases land in exactly those three shapes, so the set
// is closed and this is a sealed sum (STYLE T2/T7, the PRINCIPLES 7 sealed-sum
// exception) mirroring ContentType and Term — never a kind tag beside optional
// payloads, which would make "an Element Declaration that is also skip"
// representable.
type defaultBinding interface{ defaultBinding() }

// elementDeclarationBinding is key-dft-binding case 1: the item has a ·governing
// element declaration·, and the binding is that Element Declaration. Nothing in
// this issue constructs one — attributeDefaultBinding decides the attribute half
// only — and loc-testSubP clause 4, the sub-test that reads it, is
// cos-content-act-restrict's (#263). The variant exists so that work adds a case
// to checkBindingSubsumes rather than reshaping the sum.
type elementDeclarationBinding struct{ decl ElementDeclaration }

// attributeUseBinding is key-dft-binding case 2 (the item has a ·governing
// attribute declaration· and is ·attributed· to an Attribute Use, so the binding
// is that use) and case 3 (it is ·attributed· to an attribute wildcard, so the
// binding is a SYNTHESIZED Attribute Use whose {attribute declaration} is the
// ·governing attribute declaration·, whose {value constraint} is ·absent·, and
// whose {inheritable} is that declaration's).
type attributeUseBinding struct{ use AttributeUse }

// wildcardKeywordBinding is key-dft-binding cases 4, 5 and 6: the item is
// ·attributed· to a strict or lax wildcard with NO ·governing· declaration
// (cases 4 and 5), or to a skip wildcard (case 6, which needs no such
// qualifier), so the binding is the keyword itself. The keyword is the already
// typed ProcessContents closed set (closedsets.go), never a string.
type wildcardKeywordBinding struct{ keyword ProcessContents }

func (elementDeclarationBinding) defaultBinding() {}
func (attributeUseBinding) defaultBinding()       {}
func (wildcardKeywordBinding) defaultBinding()    {}

// attributeDefaultBinding computes c's ·default binding· (key-dft-binding) for an
// attribute of expanded name n:
//
//   - case 2: an {attribute use} of c whose {attribute declaration} carries n,
//     looked up through foldedAttributeUse so that an INHERITED use counts.
//     cvc-complex-type clause 2.1 makes such a use the ·context-determined
//     declaration·, so it wins over any wildcard — including a wildcard on c
//     itself, since under §3.4.2.4 clause 3 the inherited use is a member of
//     c.{attribute uses} and case 2 is tested before cases 4/5/6.
//   - cases 4/5/6: otherwise, if c's {attribute wildcard} admits n, the wildcard's
//     {process contents} keyword. This one is read off c ALONE, which is exact:
//     §3.4.2.5 clause 2.1 makes a restriction's {attribute wildcard} the
//     ·complete wildcard· — its own <anyAttribute>, with nothing inherited — and
//     restriction is the only {derivation method} parser/produce_complex.go emits
//     for a complex type (its <extension> branches are declined). Walking the
//     chain for wildcards would be wrong, not merely lenient: every type reaches
//     ·xs:anyType·, whose lax ##any attribute wildcard would then admit every
//     name and make the caller's check vacuous. §3.4.2.5 clause 2.2's fold of the
//     ·base wildcard· into an EXTENSION's wildcard is therefore not reconstructed
//     here; it becomes reachable only when the producer emits extension.
//
// ok is false when c admits no attribute of that name at all — no matching use
// anywhere on the base chain and no admitting wildcard — in which case there is
// no binding to compare and the CALLER charges the failure. Case 1 (·governing
// element declaration·) is the element half and is not reachable here.
//
// GAP(xsd): case 3 — the item has a ·governing attribute declaration· and is
// ·attributed· to an attribute wildcard, so the binding is a SYNTHESIZED
// Attribute Use over that declaration — is not rendered, and the wildcard branch
// falls through to the keyword instead. Whether an attribute HAS a ·governing
// attribute declaration· is an assessment-episode fact (key-governing-ad clause
// 3 resolves it by expanded name at ·assessment· time, and clause 1 lets the
// processor stipulate one outright); a static schema check cannot fix it, and
// guessing "a top-level declaration of that name exists, so case 3 applies"
// makes an unrelated global declaration silently constrain a restriction's local
// attribute type. Falling through to the keyword is FAIL-OPEN — cases 4/5 are
// weaker tests than case 3's clause-5 comparison — and never a false reject
// (#265).
func (s *Schema) attributeDefaultBinding(c ComplexType, n QName) (defaultBinding, bool) {
	if u, ok := s.foldedAttributeUse(c, n); ok {
		return attributeUseBinding{use: u}, true // case 2
	}
	w, has := c.AttributeWildcard()
	if !has || !s.allowsAttributeWildcardName(w, n) {
		return nil, false
	}
	return wildcardKeywordBinding{keyword: w.ProcessContents()}, true // cases 4/5/6
}

// foldedAttributeUse is the member of c.{attribute uses} whose {attribute
// declaration} carries the expanded name n, reading {attribute uses} as
// §3.4.2.4 clause 3 defines it: the type's OWN uses (clauses 1 and 2) together
// with those "inherited" from the {base type definition}, under both clause 3.1
// (extension) and clause 3.2 (restriction). parser/produce_complex.go maps each
// <restriction>'s own <attribute> children and no more, so the fold is
// reconstructed here by walking the base chain — without it a use B genuinely
// has by inheritance reads as absent, and charging its absence FALSE-REJECTS a
// valid schema.
//
// The type's own uses win: the walk stops at the FIRST level carrying the name,
// which is clause 3.2.1's "already included in the set" exception rendered by
// search order rather than by building the set.
//
// GAP(xsd): clause 3.2.2's other exception — an <attribute use="prohibited">
// child, which removes an inherited use — corresponds to no component at all and
// so leaves nothing on the chain to stop the walk. A prohibited name therefore
// still reports the ancestor's use. That direction is FAIL-OPEN (a binding is
// found where the spec has none, so a rejection is missed) and never a false
// reject (#265).
//
// The chain walk carries no visited set (Phase B licenses it — see
// checkComplexDerivations) and terminates on ·xs:anyType·, the one type §3.4.7
// lets be its own base. The name is tested against anyType's uses first anyway:
// §3.4.7 gives it an empty {attribute uses}, so the order is immaterial and only
// the termination matters.
func (s *Schema) foldedAttributeUse(c ComplexType, n QName) (AttributeUse, bool) {
	for {
		if u, ok := findAttributeUse(c, n); ok {
			return u, true
		}
		if c.Name() == anyTypeName {
			return AttributeUse{}, false
		}
		next, ok := s.baseComplexType(c)
		if !ok {
			return AttributeUse{}, false
		}
		c = next
	}
}

// attributeUseName is the expanded name of an attribute use's {attribute
// declaration}: the sibling local declaration's {name} for the Local variant,
// the deferred reference's QName for the Ref variant. The two agree by
// construction — a ref names the declaration it resolves to — so no lookup is
// needed and no name is stored twice (STYLE D3).
func attributeUseName(u AttributeUse) QName {
	switch d := u.attributeDeclaration.(type) {
	case LocalAttributeDeclaration:
		return d.Declaration.Name()
	case AttributeDeclarationRef:
		return d.Name
	default:
		panic("xsd: attributeUseName: non-exhaustive AttributeDeclarationOrRef switch")
	}
}

// attributeUseDeclaration resolves the Attribute Declaration behind an attribute
// use for both variants. ok is false only for a dangling Ref, which Phase A
// already rejected (src-resolve clause 1.2), so it is unreachable on a *Schema
// that exists.
func (s *Schema) attributeUseDeclaration(u AttributeUse) (AttributeDeclaration, bool) {
	switch d := u.attributeDeclaration.(type) {
	case LocalAttributeDeclaration:
		return d.Declaration, true
	case AttributeDeclarationRef:
		return s.Attribute(d.Name)
	default:
		panic("xsd: attributeUseDeclaration: non-exhaustive AttributeDeclarationOrRef switch")
	}
}

// effectiveValueConstraint is the ·effective value constraint· of an attribute
// use (Structures §3.5.4, key-evc): U.{value constraint} if present, otherwise
// U.{attribute declaration}.{value constraint} if present, otherwise ·absent·.
//
// It is a finalize-phase helper on *Schema rather than a method on AttributeUse
// because the Ref variant's declaration is reachable only through the schema's
// {attribute declarations}. It stays unexported: the eventual consumer is the
// instance validator's cvc-au, and that issue is the one that justifies
// exporting it (STYLE T5/T8).
func (s *Schema) effectiveValueConstraint(u AttributeUse) (ValueConstraint, bool) {
	if vc, ok := u.ValueConstraint(); ok {
		return vc, true
	}
	d, ok := s.attributeUseDeclaration(u)
	if !ok {
		return ValueConstraint{}, false
	}
	return d.ValueConstraint()
}

// checkBindingSubsumes charges derivation-ok-restriction clause 3 when general
// does not ·subsume· specific (Structures §3.4.6.4, loc-testSubP). general is
// the BASE type's binding (the spec's G) and specific the restriction's (S); n
// names the attribute both are for, and t/b name the two complex types, for the
// message.
//
//   - clause 1: G is skip — subsumes anything.
//   - clause 2: G is lax and S is not skip.
//   - clause 3: both are strict.
//   - clause 4: both are Element Declarations — unreachable from an attribute
//     binding; cos-content-act-restrict (#263) fills it in.
//   - clause 5: both are Attribute Uses (checkAttributeUseSubsumes).
//
// Anything else fails to subsume: G strict against an Attribute Use S, for
// instance, is a real violation — under B the attribute would have to ·resolve·
// to a declaration and be assessed strictly, which S's own use does not
// guarantee.
func (s *Schema) checkBindingSubsumes(n QName, t, b ComplexType, general, specific defaultBinding) error {
	if g, ok := general.(wildcardKeywordBinding); ok {
		return checkKeywordSubsumes(n, t, b, g, specific)
	}
	g, gIsUse := general.(attributeUseBinding)
	sp, sIsUse := specific.(attributeUseBinding)
	if gIsUse && sIsUse {
		return s.checkAttributeUseSubsumes(n, t, b, g.use, sp.use) // clause 5
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s, but %s's ·default binding· for attribute %s does not ·subsume· the restriction's (derivation-ok-restriction clause 3, c-ran, via loc-testSubP)", t.Name(), b.Name(), b.Name(), n)
}

// checkKeywordSubsumes decides loc-testSubP clauses 1-3, where the base's
// binding G is one of the three keywords.
func checkKeywordSubsumes(n QName, t, b ComplexType, general wildcardKeywordBinding, specific defaultBinding) error {
	switch general.keyword {
	case ProcessSkip:
		return nil // clause 1: skip subsumes anything
	case ProcessLax:
		if k, ok := specific.(wildcardKeywordBinding); ok && k.keyword == ProcessSkip {
			return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
				"complex type %s restricts %s, but %s binds attribute %s to a lax wildcard while the restriction binds it to a skip wildcard, and loc-testSubP clause 2 requires the specific binding not to be skip (derivation-ok-restriction clause 3, c-ran)", t.Name(), b.Name(), b.Name(), n)
		}
		return nil // clause 2
	case ProcessStrict:
		// GAP(xsd): loc-testSubP clause 3 says a strict G ·subsumes· only another
		// strict S, so a restriction that replaces a base's strict attribute
		// wildcard with a named {attribute use} reads as a violation. That reading
		// is not statically sound and is not what conforming processors do: the
		// keyword is reached only through key-dft-binding case 4, whose "does not
		// have a ·governing attribute declaration·" qualifier is an
		// assessment-episode fact this check cannot fix (see
		// attributeDefaultBinding's GAP), and XSD 1.0's derivation-ok-restriction
		// decided this branch by namespace allowance alone. Rejecting would decline
		// the canonical valid pattern "base carries <anyAttribute namespace='##any'/>,
		// restriction names specific attributes" — W3C suite MS-ComplexType ctG007
		// and ctO003 declare exactly that VALID. Accepting is FAIL-OPEN, never a
		// false reject (#265).
		return nil
	default:
		panic("xsd: checkKeywordSubsumes: non-exhaustive ProcessContents switch")
	}
}

// checkAttributeUseSubsumes decides loc-testSubP clause 5, where both bindings
// are Attribute Uses: 5.1 type derivation, 5.2 effective value constraint, 5.3
// {inheritable} equality. The sub-clauses are checked in spec order so the first
// reported failure is deterministic (STYLE D1).
func (s *Schema) checkAttributeUseSubsumes(n QName, t, b ComplexType, general, specific AttributeUse) error {
	if err := s.checkAttributeTypeDerivedOK(n, t, b, general, specific); err != nil {
		return err
	}
	if err := s.checkAttributeValueConstraintSubsumes(n, t, b, general, specific); err != nil {
		return err
	}
	if general.Inheritable() != specific.Inheritable() {
		return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
			"complex type %s restricts %s, but attribute %s has {inheritable} = %t there and %t in the base, and loc-testSubP clause 5.3 requires them to be equal (derivation-ok-restriction clause 3, c-ran)", t.Name(), b.Name(), n, specific.Inheritable(), general.Inheritable())
	}
	return nil
}

// checkAttributeTypeDerivedOK is loc-testSubP clause 5.1: S.{attribute
// declaration}.{type definition} must be validly ·derived· from G's, as defined
// in Type Derivation OK (Simple) (§3.16.6.3, cos-st-derived-ok).
//
// The blocking-keyword set the enclosing clause 3 works under is empty here —
// clause 5.1 names cos-st-derived-ok with no set — so derivedOKSimple
// (derivation.go) is used unchanged.
//
// An unresolvable or non-simple {type definition} on either side is SKIPPED
// rather than rejected: a dangling type name was already charged src-resolve by
// Phase A, and a name resolving to a complex type is not a fact this clause is
// competent to charge. Skipping is fail-open, never a false reject.
func (s *Schema) checkAttributeTypeDerivedOK(n QName, t, b ComplexType, general, specific AttributeUse) error {
	gd, ok := s.attributeUseDeclaration(general)
	if !ok {
		return nil
	}
	sd, ok := s.attributeUseDeclaration(specific)
	if !ok {
		return nil
	}
	gt, ok := s.simpleTypeNamed(gd.TypeDefinitionName())
	if !ok {
		return nil
	}
	st, ok := s.simpleTypeNamed(sd.TypeDefinitionName())
	if !ok {
		return nil
	}
	if derivedOKSimple(st, gt) {
		return nil
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s but types attribute %s as %s, which is not validly derived from the base's %s (derivation-ok-restriction clause 3, c-ran, via loc-testSubP clause 5.1 and cos-st-derived-ok §3.16.6.3)", t.Name(), b.Name(), n, sd.TypeDefinitionName(), gd.TypeDefinitionName())
}

// checkAttributeValueConstraintSubsumes is loc-testSubP clause 5.2: with GVC and
// SVC the two ·effective value constraints· (key-evc), one or more of 5.2.1 (GVC
// ·absent· or {variety} default) and 5.2.2 (SVC.{variety} = fixed and SVC.{value}
// equal or identical to GVC.{value}) must hold.
//
// Three of the four outcomes are exact. 5.2.1 is read directly. When GVC is
// fixed and SVC is absent or default, 5.2.2 cannot hold under any reading, so
// the rejection is exact. When both are fixed with equal {lexical form}s, 5.2.2
// holds.
//
// GAP(xsd): the fourth outcome — both fixed, {lexical form}s DIFFERENT — is
// accepted. 5.2.2 compares {value}s, a value-space test: "1" and "01" are the
// same xs:integer value with different lexical forms. ValueConstraint carries
// only {lexical form} (valueconstraint.go) and this package must not depend on
// package value, so a lexical mismatch is not evidence of a value mismatch.
// Accepting is FAIL-OPEN — a schema that should fail may pass — and never a
// false reject. Closing it needs the lexical mapping of the attribute's
// {type definition}, which belongs with the instance validator's key-evc
// consumer.
func (s *Schema) checkAttributeValueConstraintSubsumes(n QName, t, b ComplexType, general, specific AttributeUse) error {
	gvc, present := s.effectiveValueConstraint(general)
	if !present || gvc.Kind() != ValueFixed {
		return nil // clause 5.2.1
	}
	svc, present := s.effectiveValueConstraint(specific)
	if !present || svc.Kind() != ValueFixed {
		return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
			"complex type %s restricts %s, but the base fixes attribute %s to %q while the restriction leaves it unfixed, and loc-testSubP clause 5.2 requires a fixed ·effective value constraint· with the same value (derivation-ok-restriction clause 3, c-ran)", t.Name(), b.Name(), n, gvc.LexicalForm())
	}
	return nil // clause 5.2.2, exactly when the lexical forms agree; see the GAP above
}

// simpleTypeNamed resolves a {type definition} reference to a Simple Type
// Definition. ok is false for an absent (zero) name, an unresolvable one, and
// one naming a complex type — the three cases every caller here treats as "not
// decidable by this clause", never as a violation.
func (s *Schema) simpleTypeNamed(name QName) (*SimpleType, bool) {
	if name == (QName{}) {
		return nil, false
	}
	t, ok := s.Type(name)
	if !ok {
		return nil, false
	}
	st, ok := t.(*SimpleType)
	return st, ok
}
