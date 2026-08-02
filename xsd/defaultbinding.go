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
//
// The ELEMENT half (key-dft-binding case 1, loc-testSubP clause 4) is
// cos-content-act-restrict's business (derivation-ok-restriction clause 2.4.2)
// and lives here too, but is reached from contentrestricts.go rather than from
// this file: that constraint walks a content model to learn WHICH declaration or
// wildcard an item binds to, while the two halves share the ·subsumes· relation
// below. The two entry points differ only in what they return — the attribute
// half charges an error per sub-clause, the element half answers a bool, because
// it is one conjunct of derivation-ok-restriction clause 2's disjunction and so
// is not yet a verdict where it is evaluated. The wildcard-bucket lattice
// (clauses 1-3) has ONE encoding, keywordSubsumes, which both reach (STYLE T4).

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
// element declaration·, and the binding is that Element Declaration. It is
// constructed by contentrestricts.go's elementPositionBinding, for an item
// ·attributed· to an ·element particle· of a content model, and read by
// elementDeclarationSubsumes (loc-testSubP clause 4). attributeDefaultBinding
// never produces one: cases 2 and 3 are the attribute half.
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
//   - case 2: the member of c.{attribute uses} whose {attribute declaration}
//     carries n. The property is the MATERIALISED one (§3.4.2.4 clause 3,
//     attributeusefold.go), so an INHERITED use counts without any walk here.
//     cvc-complex-type clause 2.1 makes such a use the ·context-determined
//     declaration·, so it wins over any wildcard — including a wildcard on c
//     itself, since case 2 is tested before cases 4/5/6. The set is exact for
//     both derivation methods: clause 3.2.2's prohibited names are applied at the
//     fold too, so a name a restriction prohibits is absent here rather than
//     reported as the ancestor's use.
//   - cases 4/5/6: otherwise, if c's {attribute wildcard} admits n, the wildcard's
//     {process contents} keyword. This one is read off c ALONE, which is exact for
//     a restriction: §3.4.2.5 clause 2.1 makes its {attribute wildcard} the
//     ·complete wildcard· — its own <anyAttribute>, with nothing inherited.
//     Walking the chain for wildcards would be wrong, not merely lenient: every
//     type reaches ·xs:anyType·, whose lax ##any attribute wildcard would then
//     admit every name and make the caller's check vacuous.
//
// GAP(xsd): §3.4.2.5 clause 2.2's cos-aw-union of the ·base wildcard· into an
// EXTENSION's {attribute wildcard} is NOT folded — neither by the producer nor at
// finalize, unlike {attribute uses} — so an extension's wildcard is its own
// <anyAttribute> alone here. That is a FALSE-REJECT risk, not a fail-open one:
// the under-reported wildcard makes ok false for a name the extension genuinely
// admits, and checkRestrictionAttributes charges derivation-ok-restriction on
// exactly that !ok, so a restriction of an extension whose base carried the
// admitting wildcard is rejected for an attribute the base really does allow.
// Closing it is #265 section 3's (the extension wildcard fold); it is stated
// plainly here so no caller reads the gap as merely lenient.
//
// ok is false when c admits no attribute of that name at all — no member of
// {attribute uses} and no admitting wildcard — in which case there is no binding
// to compare and the CALLER charges the failure. Case 1 (·governing element
// declaration·) is the element half and is not reachable here.
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
	if u, ok := findAttributeUse(c.attributeUses, n); ok {
		return attributeUseBinding{use: u}, true // case 2
	}
	w, has := c.AttributeWildcard()
	if !has || !s.allowsAttributeWildcardName(w, n) {
		return nil, false
	}
	return wildcardKeywordBinding{keyword: w.ProcessContents()}, true // cases 4/5/6
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
// {attribute declarations}. Its one caller today is
// checkAttributeValueConstraintSubsumes, which is where loc-testSubP clause 5.2
// invokes ·effective value constraint· by name. It stays unexported: the eventual
// consumer is the instance validator's cvc-au, and that issue is the one that
// justifies exporting it (STYLE T5/T8).
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
//     binding, and decided by bindingSubsumes on the element side.
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
// binding G is one of the three keywords, and charges the one way they can fail.
// The predicate itself is keywordSubsumes, which the element half of the
// definition shares (STYLE T4); only the message is built here, and only the
// lax-versus-skip pairing of clause 2 can reach it.
func checkKeywordSubsumes(n QName, t, b ComplexType, general wildcardKeywordBinding, specific defaultBinding) error {
	if keywordSubsumes(general, specific) {
		return nil
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s, but %s binds attribute %s to a lax wildcard while the restriction binds it to a skip wildcard, and loc-testSubP clause 2 requires the specific binding not to be skip (derivation-ok-restriction clause 3, c-ran)", t.Name(), b.Name(), b.Name(), n)
}

// keywordSubsumes is loc-testSubP clauses 1-3, where the general binding G is
// one of the three keywords. It is the ONE encoding of the wildcard-bucket
// lattice (STYLE T4): both halves of ·subsumes· reach it — the attribute half
// through checkKeywordSubsumes (derivation-ok-restriction clause 3) and the
// element half through bindingSubsumes (clause 2.4.2's
// cos-content-act-restrict) — and neither re-derives it.
//
//   - clause 1: G is skip, which subsumes anything.
//   - clause 2: G is lax and S is not skip.
//   - clause 3: both G and S are strict; see the GAP below for the reading taken.
func keywordSubsumes(general wildcardKeywordBinding, specific defaultBinding) bool {
	switch general.keyword {
	case ProcessSkip:
		return true // clause 1
	case ProcessLax:
		k, ok := specific.(wildcardKeywordBinding)
		return !ok || k.keyword != ProcessSkip // clause 2
	case ProcessStrict:
		// GAP(xsd): loc-testSubP clause 3 says a strict G ·subsumes· only another
		// strict S, so a restriction that replaces a base's strict wildcard with a
		// named {attribute use} — or, on the element side, with a named element
		// particle — reads as a violation. That reading is not statically sound
		// and is not what conforming processors do: the keyword is reached only
		// through key-dft-binding cases 4/5, whose "does not have a ·governing
		// element declaration· or a ·governing attribute declaration·" qualifier
		// is an assessment-episode fact this check cannot settle (see
		// attributeDefaultBinding's GAP), and XSD 1.0's derivation-ok-restriction
		// decided this branch by namespace allowance alone. Rejecting would
		// decline the canonical valid pattern "base carries a ##any wildcard,
		// restriction names specific attributes or elements" — W3C suite
		// MS-ComplexType ctG007 and ctO003 declare exactly that VALID. Accepting
		// is FAIL-OPEN, never a false reject (#265).
		return true
	default:
		panic("xsd: keywordSubsumes: non-exhaustive ProcessContents switch")
	}
}

// bindingSubsumes is the bool-valued rendering of ·subsumes· (loc-testSubP) the
// ELEMENT half needs: cos-content-act-restrict clause 2
// (ctr-child-type-subsumption) is one conjunct of a DISJUNCTION —
// derivation-ok-restriction clause 2 — so a failure inside it is a verdict, not
// yet an error, and contentTypeRestricts (complexderivation.go) answers a bool
// all the way up to the one site that charges the rule.
//
//   - G a keyword: clauses 1-3, keywordSubsumes.
//   - both Element Declarations: clause 4, elementDeclarationSubsumes.
//   - anything else fails to subsume. An Element Declaration G against a keyword
//     S is the pairing loc-testSubP deliberately leaves uncovered — clause 3
//     requires BOTH sides strict — so a wildcard-matched item in the restriction
//     is never subsumed by an explicitly named element declaration in the base.
//     Attribute Uses cannot reach here: key-dft-binding cases 2 and 3 are
//     attribute-only, and the element sequences this predicate serves produce
//     only cases 1 and 4-6.
func (s *Schema) bindingSubsumes(general, specific defaultBinding) bool {
	if g, ok := general.(wildcardKeywordBinding); ok {
		return keywordSubsumes(g, specific) // clauses 1-3
	}
	g, gIsElement := general.(elementDeclarationBinding)
	sp, sIsElement := specific.(elementDeclarationBinding)
	if gIsElement && sIsElement {
		return s.elementDeclarationSubsumes(g.decl, sp.decl) // clause 4
	}
	return false
}

// elementDeclarationSubsumes is loc-testSubP clause 4, where both bindings are
// Element Declarations. All six sub-clauses must hold; they are tested in spec
// order so the verdict does not depend on evaluation order.
//
//   - 4.1: G.{nillable} = true or S.{nillable} = false.
//   - 4.2: G has no {value constraint}, or it is not fixed, or S has a fixed
//     {value constraint} with an equal or identical value.
//   - 4.3: S.{identity-constraint definitions} ⊇ G.{identity-constraint definitions}.
//   - 4.4: S disallows a superset of the substitutions G does.
//   - 4.5 (c-vs-ct): S's declared {type definition} is ·validly substitutable as
//     a restriction· for G's (key-val-sub-type-restricts: ·validly
//     substitutable· subject to the blocking keywords {extension, list, union},
//     which is exactly restrictionBlockingKeywords).
//   - 4.6 (c-tt-equiv): the two {type table}s are both ·absent· or both present
//     and ·equivalent· (key-equiv-tt, typeTablesEquivalent).
func (s *Schema) elementDeclarationSubsumes(general, specific ElementDeclaration) bool {
	if !general.Nillable() && specific.Nillable() {
		return false // clause 4.1
	}
	if !s.fixedValueConstraintSubsumes(general, specific) {
		return false // clause 4.2
	}
	if !identityConstraintsSuperset(specific, general) {
		return false // clause 4.3
	}
	if !disallowedSubstitutionsSuperset(specific, general) {
		return false // clause 4.4
	}
	if !s.declaredTypeRestricts(specific, general) {
		return false // clause 4.5
	}
	return typeTablesAgree(general, specific) // clause 4.6
}

// fixedValueConstraintSubsumes is loc-testSubP clause 4.2. An absent or
// non-fixed G {value constraint} discharges it outright; a fixed G against an
// absent or default S is an exact rejection, since no reading of "S has a fixed
// {value constraint}" can hold.
//
// The remaining outcome — both fixed — is 4.2's "with an equal or identical value",
// a VALUE-space test: "1" and "01" are the same xs:integer value, so the
// {lexical form}s ValueConstraint carries cannot decide it (valueconstraint.go).
// The schema's installed ValueSpace (valuespace.go) decides it instead, in the
// two declarations' own types; an undecided verdict accepts, so the comparison
// can only NARROW what this clause admits.
//
// GAP(xsd): what remains fail-open here is exactly what the ValueSpace declines
// to decide — a type no backend mapping governs, a {lexical form} that mapping
// cannot map, two types resolving to DIFFERENT governing mappings (an
// incommensurable cross-type comparison), and the context-dependent QName and
// NOTATION spaces, whose lexicals need the in-scope namespace bindings of the
// schema document that wrote them, which no ValueConstraint carries (see package
// value's own GAP(value) marker). A {type definition} that is absent,
// unresolvable, or COMPLEX (a simple-content complex type still bearing a value
// constraint) is skipped here for the same reason simpleTypeOf's other callers
// skip it: there is no simple type to name the value space. Every one of those
// accepts, so none is ever a false reject (#265).
func (s *Schema) fixedValueConstraintSubsumes(general, specific ElementDeclaration) bool {
	gvc, present := general.ValueConstraint()
	if !present || gvc.Kind() != ValueFixed {
		return true
	}
	svc, present := specific.ValueConstraint()
	if !present || svc.Kind() != ValueFixed {
		return false
	}
	gt, ok := s.simpleTypeOf(general.TypeDefinition())
	if !ok {
		return true
	}
	st, ok := s.simpleTypeOf(specific.TypeDefinition())
	if !ok {
		return true
	}
	same, decided := s.valueSpace.EqualOrIdentical(st, svc, gt, gvc)
	return same || !decided
}

// identityConstraintsSuperset is loc-testSubP clause 4.3: every member of
// general's {identity-constraint definitions} must also be a member of
// specific's. Component identity is the expanded {name}, the same reading
// sameTypeDefinition (complexderivation.go) takes for type definitions —
// sch-props-correct clause 2 keeps {identity-constraint definitions} unique by
// expanded name across the schema, so a name is a component key.
//
// Both sets are walked in document order; no map is consulted (STYLE D2).
func identityConstraintsSuperset(specific, general ElementDeclaration) bool {
	for _, g := range general.identityConstraints {
		if !hasIdentityConstraintNamed(specific, g.Name()) {
			return false
		}
	}
	return true
}

// hasIdentityConstraintNamed reports whether e's {identity-constraint
// definitions} contains one with the expanded name.
func hasIdentityConstraintNamed(e ElementDeclaration, name QName) bool {
	for _, c := range e.identityConstraints {
		if c.Name() == name {
			return true
		}
	}
	return false
}

// disallowedSubstitutionsSuperset is loc-testSubP clause 4.4, "S disallows a
// superset of the substitutions that G does": every member of general's
// {disallowed substitutions} must also be a member of specific's.
func disallowedSubstitutionsSuperset(specific, general ElementDeclaration) bool {
	for _, m := range general.disallowedSubstitutions {
		if !containsDerivationMethod(specific.disallowedSubstitutions, m) {
			return false
		}
	}
	return true
}

// declaredTypeRestricts is loc-testSubP clause 4.5 (c-vs-ct), delegating to
// validlySubstitutable (key-val-sub-type) under the {extension, list, union}
// blocking keywords that ·validly substitutable as a restriction·
// (key-val-sub-type-restricts) names — the same set derivation-ok-restriction
// clause 4 works under, and the same slice, so the two clauses cannot drift.
//
// An absent or unresolvable {type definition} on either side is SKIPPED rather
// than rejected, exactly as checkLocallyDeclaredElementTypes skips it: there is
// no component to compare, so the clause is not competent to charge a failure.
// Skipping is fail-open, never a false reject. An ANONYMOUS type is no longer
// among the skipped cases: typeOf hands back the inline component itself, so the
// comparison is made rather than waved through.
func (s *Schema) declaredTypeRestricts(specific, general ElementDeclaration) bool {
	sub, ok := s.typeOf(specific.TypeDefinition())
	if !ok {
		return true
	}
	super, ok := s.typeOf(general.TypeDefinition())
	if !ok {
		return true
	}
	return s.validlySubstitutable(sub, super, restrictionBlockingKeywords)
}

// typeOf is the one way this package turns an element or attribute declaration's
// {type definition} slot into a component, exhaustively over
// TypeDefinitionOrRef's two arms: an InlineTypeDefinition IS the component (it
// is in no by-name symbol table, so a lookup would miss it), while a
// TypeDefinitionRef is the by-name Schema.Type lookup. ok is false for an absent
// (nil) slot and for an unresolvable name — the cases every caller treats as
// "not decidable by this clause", never as a violation (a dangling name was
// already charged src-resolve by resolve.go's Phase A).
//
// Every {type definition} consumer goes through this helper or through its
// narrowed sibling simpleTypeOf; none re-derives a bare-name lookup of its own
// (STYLE T4).
func (s *Schema) typeOf(ref TypeDefinitionOrRef) (TypeDefinition, bool) {
	switch r := ref.(type) {
	case nil:
		return nil, false
	case TypeDefinitionRef:
		return s.Type(r.Name)
	case InlineTypeDefinition:
		return r.Definition, true
	default:
		panic("xsd: typeOf: non-exhaustive TypeDefinitionOrRef switch")
	}
}

// typeTablesAgree is loc-testSubP clause 4.6 (c-tt-equiv): the two {type table}s
// are both ·absent·, or both present and ·equivalent· per key-equiv-tt. The
// equivalence itself is elementconsistent.go's typeTablesEquivalent, the one
// canonical §3.8.6.3 implementation (STYLE T4).
func typeTablesAgree(general, specific ElementDeclaration) bool {
	gt, gPresent := general.TypeTable()
	st, sPresent := specific.TypeTable()
	if gPresent != sPresent {
		return false
	}
	if !gPresent {
		return true
	}
	return typeTablesEquivalent(st, gt)
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
// competent to charge. Skipping is fail-open, never a false reject. Both sides
// are resolved through attributeUseType, the one encoding of "the simple type
// governing this use" clause 5.2.2 also reads (STYLE T4).
func (s *Schema) checkAttributeTypeDerivedOK(n QName, t, b ComplexType, general, specific AttributeUse) error {
	gt, ok := s.attributeUseType(general)
	if !ok {
		return nil
	}
	st, ok := s.attributeUseType(specific)
	if !ok {
		return nil
	}
	if derivedOKSimple(st, gt) {
		return nil
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s but types attribute %s as %s, which is not validly derived from the base's %s (derivation-ok-restriction clause 3, c-ran, via loc-testSubP clause 5.1 and cos-st-derived-ok §3.16.6.3)", t.Name(), b.Name(), n, typeDefinitionLabel(st), typeDefinitionLabel(gt))
}

// checkAttributeValueConstraintSubsumes is loc-testSubP clause 5.2: with GVC and
// SVC the two ·effective value constraints· (key-evc), one or more of 5.2.1 (GVC
// ·absent· or {variety} default) and 5.2.2 (SVC.{variety} = fixed and SVC.{value}
// equal or identical to GVC.{value}) must hold.
//
// The first two outcomes are exact. 5.2.1 is read directly. When GVC is fixed
// and SVC is absent or default, 5.2.2 cannot hold under any reading, so the
// rejection is exact. When both are fixed, 5.2.2's "equal or identical" is a
// VALUE-space test — "1" and "01" are the same xs:integer value with different
// lexical forms — which the {lexical form}s ValueConstraint carries
// (valueconstraint.go) cannot decide; the schema's installed ValueSpace
// (valuespace.go) decides it, in each side's own attribute {type definition},
// and an undecided verdict accepts, so the comparison can only NARROW what this
// clause admits.
//
// GAP(xsd): what remains fail-open is exactly what the ValueSpace declines to
// decide — an ungoverned type, an unmappable {lexical form}, two types resolving
// to DIFFERENT governing mappings (an incommensurable cross-type comparison,
// which clause 5.1 permits: S's type need only be DERIVED from G's), and the
// context-dependent QName and NOTATION spaces, whose lexicals need the in-scope
// namespace bindings of the schema document that wrote them and which no
// ValueConstraint carries (see package value's own GAP(value) marker) — plus a
// {type definition} that is absent, unresolvable, or complex, skipped exactly as
// simpleTypeOf's other callers skip it. Every one of those accepts, so none is
// ever a false reject (#265).
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
	if s.attributeValueConstraintsAgree(general, specific, gvc, svc) {
		return nil // clause 5.2.2
	}
	return xsderr.New(ruleDerivationOKRestriction, xsderr.Loc{},
		"complex type %s restricts %s and fixes attribute %s to %q, but the base fixes it to %q, and loc-testSubP clause 5.2.2 requires the two {value}s to be equal or identical (derivation-ok-restriction clause 3, c-ran)", t.Name(), b.Name(), n, svc.LexicalForm(), gvc.LexicalForm())
}

// attributeValueConstraintsAgree decides loc-testSubP clause 5.2.2's "SVC.{value}
// is equal or identical to GVC.{value}" for two fixed ·effective value
// constraints·, asking the installed ValueSpace in each side's own attribute
// {type definition}. It reports true — accept — for every case the ValueSpace
// does not decide and for every side whose {type definition} names no simple
// type; see checkAttributeValueConstraintSubsumes' GAP for the residual.
func (s *Schema) attributeValueConstraintsAgree(general, specific AttributeUse, gvc, svc ValueConstraint) bool {
	gt, ok := s.attributeUseType(general)
	if !ok {
		return true
	}
	st, ok := s.attributeUseType(specific)
	if !ok {
		return true
	}
	same, decided := s.valueSpace.EqualOrIdentical(st, svc, gt, gvc)
	return same || !decided
}

// attributeUseType is the Simple Type Definition governing an attribute use's
// values: its {attribute declaration}.{type definition}, resolved through the
// same two helpers every other consumer here uses (STYLE T4). ok is false for a
// dangling Ref and for a {type definition} that is absent, unresolvable, or
// complex — the cases the value-constraint clauses treat as "not decidable",
// never as a violation.
func (s *Schema) attributeUseType(u AttributeUse) (*SimpleType, bool) {
	d, ok := s.attributeUseDeclaration(u)
	if !ok {
		return nil, false
	}
	return s.simpleTypeOf(d.TypeDefinition())
}

// simpleTypeOf narrows typeOf to a Simple Type Definition. ok is false for an
// absent slot, an unresolvable name, and a {type definition} that is a complex
// type — the three cases every caller here treats as "not decidable by this
// clause", never as a violation.
func (s *Schema) simpleTypeOf(ref TypeDefinitionOrRef) (*SimpleType, bool) {
	t, ok := s.typeOf(ref)
	if !ok {
		return nil, false
	}
	st, ok := t.(*SimpleType)
	return st, ok
}
