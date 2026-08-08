package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleCosCTExtends is Derivation Valid (Extension) (Structures §3.4.6.2,
// id="cos-ct-extends"): the constraint every complex type T whose {derivation
// method} is extension must satisfy against its {base type definition} B. Every
// clause charges this one rule and names its clause number (and, where the spec
// anchors one, its clause id — c-cte, c-vs-ctd-e) in the message text, matching
// the convention complexderivation.go:5-13 already states for
// derivation-ok-restriction. The clause anchors are NOT rule IDs — they carry no
// "Schema Component Constraint" label in the spec — so they are deliberately
// absent from xsderr's catalog.
//
// One clause is charged under its OWN rule instead: 1.4.3.2.2.2's delegate
// cos-particle-extend (see ruleCosParticleExtend).
//
// Appendix B (xmlschema11-1.md:5826) says a report "should use the name given
// below plus the part number, separated by a period", i.e. cos-ct-extends.1.2.
// This tree deliberately does not: one coarse rule ID plus the clause number in
// prose is the convention #262 set and this file mirrors. The catalog entry
// "cos-ct-extends.1.2" (xsderr/catalog.go:22) is an artifact of tools/rulecat
// matching that Appendix B sentence's own example prose and is never used as a
// Rule value.
const ruleCosCTExtends xsderr.Rule = "cos-ct-extends"

// ruleCosParticleExtend is Particle Valid (Extension) (Structures §3.9.6.2,
// id="cos-particle-extend"): whether a particle E is a ·valid extension· of a
// particle B. cos-ct-extends clause 1.4.3.2.2.2 is its only invocation.
//
// It is charged as its OWN error rather than folded into cos-ct-extends's
// message text — unlike cos-content-act-restrict, which contentrestricts.go
// answers as a bool because derivation-ok-restriction clause 2 is a disjunction
// whose failing branch is genuinely ambiguous. cos-ct-extends clause 1.4 IS a
// disjunction too ("One of the following is true"), but its three branches are
// discriminated by the {content type} varieties of T and B, which are total and
// mutually exclusive, so at most one branch is ever live: by the time
// 1.4.3.2.2.2 is reached, 1.4.3.1 and 1.4.3.2.2.1 have already pinned it as THE
// branch that must carry the derivation. That pinning — not any claim that
// clause 1.4 is a conjunction — is what makes the charge attributable. Every
// other clause-1.4 failure stays coarse cos-ct-extends.
//
// Inside §3.9.6.2 the three clauses ARE an unpinned disjunction ("one or more of
// the following is true"), so particleValidExtension answers a bool and the
// message names all three.
const ruleCosParticleExtend xsderr.Rule = "cos-particle-extend"

// extensionBlockingKeywords is the blocking-keyword set cos-ct-extends clause
// 1.6 (c-vs-ctd-e) works under: the EMPTY set. Clause 1.6 says the ·locally
// declared type· within T must be ·validly substitutable· for the one within B
// ·without limitation·, and §3.4.6.4 (key-val-sub-type-absolute,
// xmlschema11-1.md:1339) defines that term as exactly "the set of keywords
// controlling whether a type S is ·validly substitutable· for another type T is
// the empty set".
//
// It is a named value rather than a bare nil at the call site because an
// untagged nil there reads as an unfinished argument, and getting it wrong in
// the obvious way — reaching for restrictionBlockingKeywords, clause 1.6's
// nearest neighbour — is a FALSE REJECT rather than a missed rejection: an
// extension's ·locally declared type· is normally extension-DERIVED from the
// base's, and blocking extension rejects every valid case.
var extensionBlockingKeywords []DerivationMethod

// checkComplexTypeExtension runs cos-ct-extends (§3.4.6.2) for one complex type,
// after establishing the constraint's own precondition: {derivation method} =
// extension with a {base type definition} that resolves.
//
// Unlike checkComplexTypeRestriction, this does NOT return nil for a simple
// base. cos-ct-extends splits at the top on B's kind and has a live case 2 for a
// simple type definition (2.1 T's {content type} is simple over B itself, 2.2
// B.{final} does not contain extension) — the <simpleContent><extension> path —
// so the simple base is an explicit branch, not a skip. The restriction twin's
// skip is licensed by something with no analogue here: ct-props-correct clause 2
// forbids a simple base under restriction outright, so its whole case is already
// charged more precisely elsewhere.
//
// xs:anyType's self-derivation (§3.4.7, any-type-itself) is skipped, mirroring
// checkComplexTypeRestriction: it is the one type the spec permits to be its own
// base, and running the constraint on it would compare it with itself. The
// seeded xs:anyType derives by restriction, so this guard is a statement of the
// invariant rather than a live arm.
func (s *Schema) checkComplexTypeExtension(t ComplexType) error {
	if t.DerivationMethod() != DerivationExtension {
		return nil
	}
	baseName := t.BaseTypeDefinitionName()
	if baseName == (QName{}) {
		return nil
	}
	if t.Name() == anyTypeName && baseName == anyTypeName {
		return nil
	}
	base, ok := s.Type(baseName)
	if !ok {
		return nil // a dangling base was already charged src-resolve by Phase A
	}
	switch b := base.(type) {
	case ComplexType:
		return s.checkExtensionOfComplexBase(t, b) // case 1
	case *SimpleType:
		return checkExtensionOfSimpleBase(t, b) // case 2
	default:
		panic("xsd: checkComplexTypeExtension: non-exhaustive TypeDefinition switch")
	}
}

// checkExtensionOfComplexBase is cos-ct-extends case 1: the seven clauses that
// apply when B is a complex type definition. All seven are checked, in spec
// order, so the first reported failure is deterministic (STYLE D1). All three
// §3.4.2 base folds the case depends on are done: §3.4.2.4 clause 3's
// {attribute uses} (#401), §3.4.2.5 clause 2.2's {attribute wildcard} (#265) and
// §3.4.2.1 clause 1's {assertions} (#346).
func (s *Schema) checkExtensionOfComplexBase(t, b ComplexType) error {
	if err := checkExtensionBaseFinal(t, b); err != nil {
		return err
	}
	if err := s.checkExtensionAttributeUses(t, b); err != nil {
		return err
	}
	if err := checkExtensionAttributeWildcard(t, b); err != nil {
		return err
	}
	if err := s.checkExtensionContentType(t, b); err != nil {
		return err
	}
	if err := s.checkExtensionTwoStepDerivable(t); err != nil {
		return err
	}
	if err := s.checkExtensionLocallyDeclaredTypes(t, b); err != nil {
		return err
	}
	return checkExtensionAssertions(t, b)
}

// checkExtensionAssertions is clause 1.7: B.{assertions} is a prefix of
// T.{assertions}. The relation, and why it is charged rather than assumed even
// though §3.4.2.1 clause 1's fold makes it hold by construction for every
// produced type, live in assertionprefix.go — derivation-ok-restriction clause 5
// states the same test in the same words and shares the encoding.
func checkExtensionAssertions(t, b ComplexType) error {
	if assertionsPrefix(b.assertions, t.assertions) {
		return nil
	}
	return xsderr.New(ruleCosCTExtends, t.Loc(),
		"complex type %s extends %s, but %s's {assertions} (%d) are not a prefix of %s's (%d), which cos-ct-extends clause 1.7 requires: §3.4.2.1 clause 1 places the base's assertions, in order, ahead of the type's own <assert> children",
		t.Name(), b.Name(), b.Name(), len(b.assertions), t.Name(), len(t.assertions))
}

// checkExtensionBaseFinal is clause 1.1: B's {final} must not contain extension.
func checkExtensionBaseFinal(t, b ComplexType) error {
	if !finalContains(b.final, DerivationExtension) {
		return nil
	}
	return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
		"complex type %s extends %s, but %s has extension in its {final}, which cos-ct-extends clause 1.1 forbids", t.Name(), b.Name(), b.Name())
}

// checkExtensionAttributeUses is clause 1.2 (c-cte): B.{attribute uses} is a
// SUBSET of T.{attribute uses} — "for every attribute use U in B.{attribute
// uses}, there is an attribute use in T.{attribute uses} whose properties,
// recursively, are identical to those of U". B's uses are walked in document
// order, so the first reported failure is deterministic (STYLE D2).
//
// Both sides are the MATERIALISED {attribute uses} (§3.4.2.4 clause 3,
// attributeusefold.go).
//
// The comparison is attributeUsesIdentical, NOT the ·subsumption· apparatus
// c-ran uses (attributeDefaultBinding/bindingSubsumes, defaultbinding.go).
// c-cte asks for property IDENTITY, which is strictly stronger than
// ·subsumption·; reusing the looser relation would decide a different constraint.
//
// NEITHER rejection below is reachable through Finalize, and the reason is the
// fold, so it is stated rather than assumed. Clause 3.1 makes T.{attribute uses}
// the concatenation of T's own uses with B's ENTIRE materialised set, member for
// member, so the clause's set-inclusion half holds BY CONSTRUCTION: the lookup
// cannot miss, and what it finds for a name T does not declare itself IS U, so
// the identity comparison cannot fail either. The remaining shape — T re-declares
// the name with different properties — leaves T holding two uses for it, which
// ct-props-correct clause 4 charges earlier in the same pass
// (checkCTPropsCorrectResolved runs before checkComplexTypeExtension for a given
// type, checkComplexDerivations).
//
// It is kept because it is the executable statement of clause 1.2 AND the guard
// on the fold's clause 3.1 arm: a fold that stopped copying B's uses forward
// would show up here as a verdict rather than as silence. That is the footing
// every fold-backed clause of this constraint stands on — 1.3
// (checkExtensionAttributeWildcard) and 1.7 (checkExtensionAssertions) alike.
// Retiring any of them belongs with a decision to trust the folds unchecked, not
// with #401.
func (s *Schema) checkExtensionAttributeUses(t, b ComplexType) error {
	for _, u := range b.attributeUses {
		name := attributeUseName(u)
		tu, ok := findAttributeUse(t.attributeUses, name)
		if !ok {
			return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
				"complex type %s extends %s, but the base's attribute use for %s has no counterpart in the extension's {attribute uses}, so B.{attribute uses} is not a subset of T.{attribute uses} (cos-ct-extends clause 1.2, c-cte)", t.Name(), b.Name(), name)
		}
		if s.attributeUsesIdentical(tu, u) {
			continue
		}
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s but re-declares attribute %s with properties that differ from the base's attribute use, which cos-ct-extends clause 1.2 (c-cte) requires to be identical", t.Name(), b.Name(), name)
	}
	return nil
}

// checkExtensionAttributeWildcard is clause 1.3: "If B has an {attribute
// wildcard}, then T also has one, and B.{attribute wildcard}.{namespace
// constraint} is a subset of T.{attribute wildcard}.{namespace constraint}, as
// defined by Wildcard Subset (§3.10.6.2)."
//
// The direction is the MIRROR of derivation-ok-restriction clause 3's wildcard
// half (checkRestrictionAttributeWildcard): an extension must admit at least
// everything its base admits, so B's constraint is sub and T's is super — the
// same one wildcardSubset (namespaceconstraint_subset.go), read the other way
// round (STYLE T4). A B with no {attribute wildcard} discharges the clause
// vacuously, which is why the absent case returns before T is read; charging on
// that half alone would be the false reject this seam was warned about.
//
// NEITHER rejection below is reachable through Finalize, and the reason is the
// fold, so it is stated rather than assumed — the same footing
// checkExtensionAttributeUses records for clause 1.2. §3.4.2.5 clause 2.2 makes
// T.{attribute wildcard} the cos-aw-union of T's own ·complete wildcard· with B's
// (attributewildcardfold.go), and a union is a superset of both operands under
// cos-ns-subset, so a folded extension satisfies 1.3 BY CONSTRUCTION: T cannot
// lack the property when B has it (clause 2.2.2.2 hands B's straight through),
// and the union cannot admit fewer names than B's operand did.
//
// It is kept, rather than left uncharged, because it is the executable statement
// of clause 1.3 AND the guard on the fold's clause 2.2 arm: a fold that stopped
// unioning the base's constraint in would show up here as a verdict rather than as
// silence. It also remains live for a Complex Type Definition assembled
// programmatically past the fold, which is the population the spec wrote the
// clause for.
func checkExtensionAttributeWildcard(t, b ComplexType) error {
	bw, has := b.AttributeWildcard()
	if !has {
		return nil
	}
	tw, extensionHas := t.AttributeWildcard()
	if !extensionHas {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s, which has an {attribute wildcard}, but %s has none, and cos-ct-extends clause 1.3 requires the extension to carry one too", t.Name(), b.Name(), t.Name())
	}
	if wildcardSubset(bw.NamespaceConstraint(), tw.NamespaceConstraint()) {
		return nil
	}
	return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
		"complex type %s extends %s but its {attribute wildcard} does not admit every expanded name the base's admits, which cos-ct-extends clause 1.3 requires as a Wildcard Subset (cos-ns-subset)", t.Name(), b.Name())
}

// checkExtensionContentType is clause 1.4. The spec words it as a disjunction
// ("One of the following is true"), but T's own {content type}.{variety}
// discriminates the three branches totally and exclusively — 1.4.1 needs T
// simple, 1.4.2 needs T empty, 1.4.3.1 needs T element-only or mixed — so the
// switch below selects the ONE branch that can carry the derivation and the
// failure it reports is the failure of that branch, not a coarse "no branch
// applied" (contrast checkRestrictionContentType, whose clause-2 disjunction has
// no such discriminant and must name all four).
func (s *Schema) checkExtensionContentType(t, b ComplexType) error {
	switch tc := t.ContentType().(type) {
	case SimpleContent:
		return checkExtensionSimpleContent(t, b, tc) // clause 1.4.1
	case EmptyContent:
		if _, ok := b.ContentType().(EmptyContent); ok {
			return nil // clause 1.4.2
		}
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s with an empty {content type}, but the base's is %s, and cos-ct-extends clause 1.4.2 requires both to be empty", t.Name(), b.Name(), b.ContentType().Variety())
	case ElementContent:
		return s.checkExtensionElementContent(t, b, tc) // clause 1.4.3
	default:
		panic("xsd: checkExtensionContentType: non-exhaustive ContentType switch")
	}
}

// checkExtensionSimpleContent is clause 1.4.1: B and T both have {content
// type}.{variety} = simple and both have THE SAME {content type}.{simple type
// definition}.
//
// "The same" is component identity. Pointer identity is tested first because it
// is what the producer actually yields — simpleContentSimpleType
// (parser/produce_complex.go) returns an EXISTING *xsd.SimpleType and rebuilds
// nothing — and sameTypeDefinition's expanded-name reading catches the
// programmatically assembled case. Two ANONYMOUS simple types that are not the
// same pointer are reported as different, the licence §3.4.6.5's no-identity
// Note grants.
func checkExtensionSimpleContent(t, b ComplexType, tc SimpleContent) error {
	bc, ok := b.ContentType().(SimpleContent)
	if !ok {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s with a simple {content type}, but the base's is %s, and cos-ct-extends clause 1.4.1 requires both to be simple", t.Name(), b.Name(), b.ContentType().Variety())
	}
	if tc.SimpleType == bc.SimpleType || sameTypeDefinition(tc.SimpleType, bc.SimpleType) {
		return nil
	}
	return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
		"complex type %s extends %s but its {content type}.{simple type definition} is %s where the base's is %s, and cos-ct-extends clause 1.4.1 requires the same one", t.Name(), b.Name(), typeDefinitionLabel(tc.SimpleType), typeDefinitionLabel(bc.SimpleType))
}

// checkExtensionElementContent is clause 1.4.3, entered once 1.4.3.1 (T's
// {variety} is element-only or mixed) holds by the caller's switch. It decides
// 1.4.3.2's inner disjunction: 1.4.3.2.1 (B empty) discharges the clause
// outright, otherwise all four conditions of 1.4.3.2.2 must hold.
//
// No xs:anyType exemption appears here, deliberately. derivation-ok-restriction
// clause 2.1 exempts an xs:anyType base outright; cos-ct-extends states no such
// case, and the asymmetry is intended — restricting AWAY from the ur-type must
// always be permitted, whereas extending a mixed type by adding element-only
// content is a narrowing in disguise, uniformly for every base. The shape that
// looks like a false reject (mixed="false" extending anyType) never reaches here
// with a mismatched variety: §3.4.2.3.3 clause 4.2.2 copies the base's whole
// {content type}, {variety} included, whenever the ·effective content· is empty,
// which is base-agnostic and makes the mixed attribute inert there.
func (s *Schema) checkExtensionElementContent(t, b ComplexType, tc ElementContent) error {
	if _, ok := b.ContentType().(EmptyContent); ok {
		return nil // clause 1.4.3.2.1
	}
	bc, ok := b.ContentType().(ElementContent)
	if !ok {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s with an %s {content type}, but the base's is %s, which satisfies neither cos-ct-extends clause 1.4.3.2.1 (base empty) nor clause 1.4.3.2.2.1 (both mixed or both element-only)", t.Name(), b.Name(), tc.Variety(), b.ContentType().Variety())
	}
	if tc.Mixed != bc.Mixed {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s with a %s {content type} where the base's is %s, but cos-ct-extends clause 1.4.3.2.2.1 requires both to be mixed or both to be element-only", t.Name(), b.Name(), tc.Variety(), bc.Variety())
	}
	if !s.particleValidExtension(tc.Particle, bc.Particle) {
		return xsderr.New(ruleCosParticleExtend, xsderr.Loc{},
			"complex type %s extends %s, but its {content type}.{particle} is not a ·valid extension· of the base's under any clause of cos-particle-extend (§3.9.6.2: clause 1 the same particle, clause 2 a 1..1 sequence group whose first member is identical to the base's particle, clause 3 both ·all· groups with equal {min occurs} and the base's {particles} a prefix of the extension's), which cos-ct-extends clause 1.4.3.2.2.2 requires", t.Name(), b.Name())
	}
	if !extensionOpenContentModesOK(bc.OpenContent, tc.OpenContent) {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s, but the base's {open content} has {mode} %s and the extension's has {mode} %s, satisfying none of cos-ct-extends clause 1.4.3.2.2.3's branches (.1 the base's is absent, .2 the extension's is interleave, .3 both are suffix)", t.Name(), b.Name(), bc.OpenContent.Mode(), openContentModeLabel(tc.OpenContent))
	}
	if !extensionOpenContentWildcardOK(bc.OpenContent, tc.OpenContent) {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends %s, but the base's {open content}.{wildcard}.{namespace constraint} is not a subset of the extension's per cos-ns-subset (§3.10.6.2), which cos-ct-extends clause 1.4.3.2.2.4 requires", t.Name(), b.Name())
	}
	return nil
}

// extensionOpenContentModesOK is clause 1.4.3.2.2.3, a disjunction over the two
// Open Content records the clause names BOT (the base's) and EOT (the
// extension's): .1 BOT is ·absent·; .2 EOT has {mode} interleave; .3 both have
// {mode} suffix. An absent record is a nil *OpenContent (complextype.go).
func extensionOpenContentModesOK(bot, eot *OpenContent) bool {
	if bot == nil {
		return true // clause 1.4.3.2.2.3.1
	}
	if eot == nil {
		return false // neither .2 nor .3 can hold without EOT
	}
	if eot.Mode() == OpenContentInterleave {
		return true // clause 1.4.3.2.2.3.2
	}
	return bot.Mode() == OpenContentSuffix && eot.Mode() == OpenContentSuffix // clause 1.4.3.2.2.3.3
}

// extensionOpenContentWildcardOK is clause 1.4.3.2.2.4: IF neither BOT nor EOT
// is ·absent·, BOT.{wildcard}.{namespace constraint} must be a subset of EOT's
// per Wildcard Subset (§3.10.6.2). The guarded "if" makes an absent record on
// either side vacuously satisfying. The subset relation is
// namespaceconstraint_subset.go's wildcardSubset, the one implementation of
// cos-ns-subset in this tree (STYLE T4).
func extensionOpenContentWildcardOK(bot, eot *OpenContent) bool {
	if bot == nil || eot == nil {
		return true
	}
	return wildcardSubset(bot.Wildcard().NamespaceConstraint(), eot.Wildcard().NamespaceConstraint())
}

// openContentModeLabel renders an Open Content's {mode} inside an error message,
// naming an absent record rather than leaving a hole (STYLE E1).
func openContentModeLabel(oc *OpenContent) string {
	if oc == nil {
		return "absent"
	}
	return oc.Mode().String()
}

// checkExtensionTwoStepDerivable is clause 1.5: it must be possible in principle
// to ·derive· T in two steps — first an extension, then a possibly vacuous
// restriction — from the ancestor whose {base type definition} is ·xs:anyType·.
//
// The PURE-EXTENSION chain is decided here, and it is decided by a proof rather
// than by construction. Walk T's {base type definition} chain up to the ancestor
// whose base is ·xs:anyType·. If every step in that chain has {derivation
// method} = extension, then the re-ordering the clause's Note prescribes ("put
// all the extension steps first, then collapse them into a single extension") is
// the IDENTITY: there is nothing to move, the collapse of the whole chain is T
// itself, and the second step — the restriction from that collapsed intermediate
// to T — is the vacuous restriction of T to T, which is valid. So clause 1.5
// holds for every pure-extension chain, with no intermediate type synthesized
// and nothing to compare.
//
// The walk carries no visited set, licensed by Phase B exactly as the other base
// walks in this package are (see checkComplexDerivations), and terminates on
// ·xs:anyType·, the one type §3.4.7 lets be its own base.
//
// A chain that MIXES extension and restriction steps is decided by CONSTRUCTION,
// which is what the Note prescribes for it: the collapsed intermediate M is
// synthesized (collapsedintermediate.go) and T is asked to be a valid restriction
// of it. Two things are charged against M, in this order:
//
//   - ct-props-correct clause 4 on M itself, through the same one scan the
//     clause's own check uses (duplicateAttributeUseName, complexderivation.go).
//     M is duplicate-free BY CONSTRUCTION — collapsedAttributeUses drops an
//     extension step's re-declaration of a name the collapse already carries,
//     because an intermediate holding both is unrepresentable rather than merely
//     invalid — so this is the guard on that construction and not the substance
//     of the clause. It is charged as cos-ct-extends all the same: a collapse
//     that yielded a duplicate would mean NO legal intermediate exists, which is
//     a clause 1.5 verdict, never a ct-props-correct one against a type the user
//     did not write.
//   - derivation-ok-restriction (§3.4.6.3) whole, T against M. That is the
//     Note's "if the resulting definition can be the basis for a valid
//     restriction to the desired definition", and it is the ONE encoding of that
//     relation in this tree (complexderivation.go, STYLE T4) — no second
//     restriction engine, and in particular no second containment engine beside
//     cos-content-act-restrict's. It is also where the Note's own motivating
//     example lands: something a restriction step removed and an extension step
//     added back "with a conflicting type assignment or value constraint" is a
//     T whose use or declaration fails clause 3's ·subsumption· or clause 4's
//     ·validly substitutable· against the inherited one M still carries.
//
// Neither verdict is surfaced as the rule it was decided by. M is synthetic and
// carries A's {name} so the base walks off it terminate on real components, so an
// error naming it would name a type the user did not write and charge a rule
// (ct-props-correct, derivation-ok-restriction) the user's schema does not
// violate (STYLE E1/E2). Both are re-charged as cos-ct-extends at T's own
// position, with the re-ordering explained in the message.
//
// GAP(xsd): the CONTENT half of the second charge is undecided for a chain
// carrying {open content} — owned by #413, which owns the same skip for real
// restrictions. checkDerivationOKRestriction reaches the content models through
// clause 2.4.2's contentTypeRestricts, which early-accepts the moment either side
// carries an {open content} record (contentrestricts.go), and M inherits whatever
// the §3.4.2.3.3 clause 4.2 merge propagated. The direction is fail-open — clause
// 1.5's content half is skipped, never falsely charged — and the clause's
// attribute halves (both charges above) still decide such a chain.
func (s *Schema) checkExtensionTwoStepDerivable(t ComplexType) error {
	if s.pureExtensionChain(t) {
		return nil // clause 1.5 holds; see the proof above
	}
	m, ok, err := s.collapsedExtension(t)
	if err != nil {
		return xsderr.New(ruleCosCTExtends, t.Loc(),
			"complex type %s cannot be shown ·derivable· in two steps — an extension followed by a possibly vacuous restriction — from the ancestor whose {base type definition} is xs:anyType, because re-ordering its derivation chain as cos-ct-extends clause 1.5's Note prescribes does not yield a legal Complex Type Definition", t.Name())
	}
	if !ok {
		// GAP(xsd): the chain leaves the complex-type graph before reaching an
		// ancestor whose base is ·xs:anyType· (an unresolvable or simple base),
		// or one step's own contribution is not recoverable from the folded
		// properties — so clause 1.5 is ACCEPTED undecided, as the whole mixed
		// chain was before #392. #586 owns the remainder. The direction is
		// fail-open against the clause's one reader: this function's error is
		// consumed only by checkExtensionOfComplexBase → checkComplexTypeExtension
		// → checkComplexDerivations → Schema.resolve, each of which propagates a
		// non-nil error as a rejection and reads nothing else out of it, so a nil
		// here is a missed rejection and can never fabricate one.
		return nil
	}
	if name, duplicate := duplicateAttributeUseName(m.attributeUses); duplicate {
		return xsderr.New(ruleCosCTExtends, t.Loc(),
			"complex type %s is not ·derivable· in two steps — an extension followed by a possibly vacuous restriction — from the ancestor whose {base type definition} is xs:anyType, as cos-ct-extends clause 1.5 requires: re-ordering its derivation chain to put every extension step first collapses two attribute uses for %s into one type, which ct-props-correct clause 4 forbids, so an extension in the chain adds back an attribute a restriction in the chain removed", t.Name(), name)
	}
	if s.restrictionFromCollapseIsVacuous(t, m) {
		return nil // clause 1.5's own "(possibly vacuous)" arm; see below
	}
	if err := s.checkDerivationOKRestriction(t, m); err != nil {
		return xsderr.New(ruleCosCTExtends, t.Loc(),
			"complex type %s is not ·derivable· in two steps — an extension followed by a possibly vacuous restriction — from the ancestor whose {base type definition} is xs:anyType, as cos-ct-extends clause 1.5 requires: re-ordering its derivation chain to put every extension step first, then collapsing them into a single extension, yields an intermediate type that %s does not validly restrict (derivation-ok-restriction, §3.4.6.3), so something a restriction step removed is added back incompatibly by an extension step", t.Name(), t.Name())
	}
	return nil
}

// restrictionFromCollapseIsVacuous reports whether the second of clause 1.5's two
// steps is the VACUOUS restriction the clause explicitly permits — "the first an
// extension and the second a restriction (possibly vacuous)" — by testing the
// three properties derivation-ok-restriction compares for property identity:
// {content type}, {attribute uses} member for member, and {attribute wildcard}.
// When they agree, the second step changes nothing and is valid by the identity
// argument, which is precisely the argument #264 landed for the pure-extension
// chain (see this file's clause-1.5 doc). {final} and {assertions} are not
// compared: M's {final} is empty, so clause 1 cannot fail, and A's {assertions}
// are a prefix of T's along the real chain, so clause 5 cannot either
// (newCollapsedExtension).
//
// It is not an optimisation, and skipping it would be a FALSE REJECT. Running
// derivation-ok-restriction on a T and a structurally identical M charges clause
// 2.4.2's ctr-child-type-subsumption and clause 4's ·validly substitutable· for
// every element or attribute whose {type definition} is ANONYMOUS, because
// sameTypeDefinition reports two anonymous types as different (§3.4.6.5's
// no-identity Note) — so an anonymous type is not substitutable even for ITSELF,
// and a type whose content model holds one would fail to restrict its own copy.
// That shape is ordinary: a chain whose only restriction step is above an
// extension of an empty-content base collapses to an M identical to T, and any
// inline <complexType> in its content model would sink it (the W3C suite's
// MS-Element elemZ015 is exactly that schema).
//
// The three predicates are this package's existing identity relations, the same
// ones cos-ct-extends clause 1.2 and cos-particle-extend read (STYLE T4).
func (s *Schema) restrictionFromCollapseIsVacuous(t, m ComplexType) bool {
	if !s.contentTypesIdentical(t.ContentType(), m.ContentType()) {
		return false
	}
	if len(t.attributeUses) != len(m.attributeUses) {
		return false
	}
	for i, u := range t.attributeUses {
		if !s.attributeUsesIdentical(u, m.attributeUses[i]) {
			return false
		}
	}
	tw, tHas := t.AttributeWildcard()
	mw, mHas := m.AttributeWildcard()
	if tHas != mHas {
		return false
	}
	return !tHas || wildcardsIdentical(tw, mw)
}

// contentTypesIdentical decides property identity between two Content Types
// (§3.4.1), exhaustively over the sealed ContentType sum: two records are
// identical when they have the same variety and their variety's own properties
// are identical — the {simple type definition} at component identity, the
// {particle} under particlesIdentical, {mixed} and {open content} field for
// field. Two Content Types of different varieties are never identical.
//
// It answers ONE question, clause 1.5's: whether the residual restriction from
// the collapsed intermediate to T is vacuous. It is not a restriction test and
// not ·equivalent· (§3.8.6.3); see restrictionFromCollapseIsVacuous.
func (s *Schema) contentTypesIdentical(a, b ContentType) bool {
	switch ac := a.(type) {
	case EmptyContent:
		_, ok := b.(EmptyContent)
		return ok
	case SimpleContent:
		bc, ok := b.(SimpleContent)
		if !ok {
			return false
		}
		return ac.SimpleType == bc.SimpleType || sameTypeDefinition(ac.SimpleType, bc.SimpleType)
	case ElementContent:
		bc, ok := b.(ElementContent)
		if !ok || ac.Mixed != bc.Mixed || !openContentsIdentical(ac.OpenContent, bc.OpenContent) {
			return false
		}
		return s.particlesIdentical(ac.Particle, bc.Particle)
	default:
		panic("xsd: contentTypesIdentical: non-exhaustive ContentType switch")
	}
}

// openContentsIdentical decides identity between two Optional {open content}
// records (§3.4.1): both ·absent·, or both present with the same {mode} and
// property-identical {wildcard}s.
func openContentsIdentical(a, b *OpenContent) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Mode() == b.Mode() && wildcardsIdentical(a.Wildcard(), b.Wildcard())
}

// pureExtensionChain reports whether every step of t's {base type definition}
// chain, up to and including the ancestor whose base is ·xs:anyType·, has
// {derivation method} = extension. A chain that leaves the complex-type graph
// before reaching xs:anyType — an unresolvable base, or a simple one — is not one
// of the pure chains clause 1.5's identity argument covers, so it answers false
// and the caller synthesizes the collapsed intermediate instead.
//
// The ancestor is tested too, and deliberately: it is a step of the chain like
// any other, and the identity argument covers the whole chain or none of it. Only
// ·xs:anyType· itself, which baseChainToAnyType reports as its own root, has no
// such step to test.
func (s *Schema) pureExtensionChain(t ComplexType) bool {
	steps, a, ok := s.baseChainToAnyType(t)
	if !ok {
		return false
	}
	for _, c := range steps {
		if c.DerivationMethod() != DerivationExtension {
			return false
		}
	}
	return a.Name() == anyTypeName || a.DerivationMethod() == DerivationExtension
}

// baseChainToAnyType walks t's {base type definition} chain to the ancestor A
// clause 1.5 names — "that type definition among its ancestors whose {base type
// definition} is ·xs:anyType·" — and returns the steps from t down to but
// EXCLUDING A, in that order, together with A itself. A is excluded because
// clause 1.5 makes it the STARTING POINT of the two-step derivation: A's own
// derivation from the ur-type is never re-ordered, and the collapse begins from
// A's properties whatever method produced them.
//
// ok is false when the chain leaves the complex-type graph first — a base that is
// absent, unresolvable, or a simple type definition, none of which is an ancestor
// the clause can start from.
//
// It is the ONE base walk this constraint makes: pureExtensionChain reads the
// same result for its proof and collapsedExtension for its construction, rather
// than each walking the chain itself (STYLE T4). Like every other base walk in
// this package it carries NO visited set, licensed by Phase B having already
// rejected every circular chain (STYLE D4, on the licence checkComplexDerivations
// states in full); ·xs:anyType·'s self-derivation, the one §3.4.7 permits and no
// acyclicity check can remove, terminates it by the explicit name test rather
// than by a guard.
func (s *Schema) baseChainToAnyType(t ComplexType) (steps []ComplexType, a ComplexType, ok bool) {
	c := t
	for {
		if c.Name() == anyTypeName {
			return steps, c, true // the ur-type itself is its own root (§3.4.7)
		}
		if c.BaseTypeDefinitionName() == anyTypeName {
			return steps, c, true // c IS the ancestor whose {base type definition} is ·xs:anyType·
		}
		steps = append(steps, c)
		next, ok := s.baseComplexType(c)
		if !ok {
			return nil, ComplexType{}, false
		}
		c = next
	}
}

// checkExtensionLocallyDeclaredTypes is clause 1.6 (c-vs-ctd-e): for any element
// or attribute information item, its ·locally declared type· (key-ldtype) within
// T must be ·validly substitutable· for its ·locally declared type· within B
// ·without limitation·, IF neither is ·absent·.
//
// It is the same quantification derivation-ok-restriction clause 4 makes, over
// the same walk, so it runs through the one shared encoding
// (checkLocallyDeclaredTypes, complexderivation.go) rather than a mirror copy.
// The ONE load-bearing difference is the blocking-keyword set: ·without
// limitation· is the empty set (extensionBlockingKeywords), never
// restrictionBlockingKeywords.
func (s *Schema) checkExtensionLocallyDeclaredTypes(t, b ComplexType) error {
	return s.checkLocallyDeclaredTypes(t, b, locallyDeclaredTypeCheck{
		rule:       ruleCosCTExtends,
		blocked:    extensionBlockingKeywords,
		verb:       "extends",
		relation:   "extension",
		limitation: "·without limitation·",
		clause:     "cos-ct-extends clause 1.6, c-vs-ctd-e",
	})
}

// checkExtensionOfSimpleBase is cos-ct-extends case 2: B is a simple type
// definition — the <simpleContent><extension> path. Both clauses are cheap.
//
//   - 2.1: T.{content type}.{variety} = simple and its {simple type definition}
//     IS B. Component identity, decided the same two ways clause 1.4.1 decides
//     it (see checkExtensionSimpleContent).
//   - 2.2: B.{final} does not contain extension.
func checkExtensionOfSimpleBase(t ComplexType, b *SimpleType) error {
	tc, ok := t.ContentType().(SimpleContent)
	if !ok {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends the simple type %s but its {content type} is %s, and cos-ct-extends clause 2.1 requires simple", t.Name(), b.Name(), t.ContentType().Variety())
	}
	if tc.SimpleType != b && !sameTypeDefinition(tc.SimpleType, b) {
		return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
			"complex type %s extends the simple type %s but its {content type}.{simple type definition} is %s, and cos-ct-extends clause 2.1 requires it to be the base itself", t.Name(), b.Name(), typeDefinitionLabel(tc.SimpleType))
	}
	if !finalContains(b.final, DerivationExtension) {
		return nil
	}
	return xsderr.New(ruleCosCTExtends, xsderr.Loc{},
		"complex type %s extends the simple type %s, but %s has extension in its {final}, which cos-ct-extends clause 2.2 forbids", t.Name(), b.Name(), b.Name())
}

// particleValidExtension is ·valid extension· (Particle Valid (Extension),
// §3.9.6.2, cos-particle-extend): whether the particle e extends the particle b.
// The three clauses are a disjunction ("one or more of the following is true"),
// so the first satisfied one discharges the relation and the answer is a bool —
// there is no single failing clause to charge from inside.
//
//   - clause 1: they are the SAME particle. This package gives Particle no
//     component identity — it holds slices, so == is unavailable — so the test
//     is particlesIdentical, recursive property identity. That is a SUPERSET of
//     "the same particle" (two distinct but property-identical particles answer
//     true), which is the accepting, fail-open direction.
//   - clause 2: e is a 1..1 particle whose {term} is a SEQUENCE group whose
//     {particles}' first member is property-identical to b. This is exactly the
//     shape §3.4.2.3.3 clause 4.2.3.3 builds — sequence[base particle, effective
//     content] — so every faithfully produced <complexContent><extension>
//     satisfies the relation through this clause.
//   - clause 3: both {term}s are ·all· groups, {min occurs} agree, and b's
//     {particles} are a positional PREFIX of e's — the shape clause 4.2.3.2
//     builds for an all group.
func (s *Schema) particleValidExtension(e, b Particle) bool {
	if s.particlesIdentical(e, b) {
		return true // clause 1
	}
	if s.extensionPrependsToSequence(e, b) {
		return true // clause 2
	}
	return s.extensionAllGroupPrefix(e, b) // clause 3
}

// extensionPrependsToSequence is cos-particle-extend clause 2: E.{min occurs} =
// E.{max occurs} = 1, E.{term} is a sequence group, and the first member of its
// {particles} has properties recursively identical to B's.
func (s *Schema) extensionPrependsToSequence(e, b Particle) bool {
	maxOccurs, bounded := e.Occurs().Max()
	if e.Occurs().Min() != 1 || !bounded || maxOccurs != 1 {
		return false
	}
	g, ok := s.resolveTermGroup(e.Term())
	if !ok || g.Compositor() != CompositorSequence || len(g.particles) == 0 {
		return false
	}
	return s.particlesIdentical(g.particles[0], b)
}

// extensionAllGroupPrefix is cos-particle-extend clause 3: 3.1 E.{min occurs} =
// B.{min occurs}; 3.2 both E and B have ·all· groups as their {term}s; 3.3 the
// {particles} of B's all group is a PREFIX of the {particles} of E's — position
// by position, under the same recursive property identity clauses 1 and 2 use.
func (s *Schema) extensionAllGroupPrefix(e, b Particle) bool {
	if e.Occurs().Min() != b.Occurs().Min() {
		return false // clause 3.1
	}
	eg, ok := s.resolveTermGroup(e.Term())
	if !ok || eg.Compositor() != CompositorAll {
		return false // clause 3.2
	}
	bg, ok := s.resolveTermGroup(b.Term())
	if !ok || bg.Compositor() != CompositorAll {
		return false // clause 3.2
	}
	if len(bg.particles) > len(eg.particles) {
		return false // clause 3.3: a longer list is no prefix
	}
	for i, bp := range bg.particles {
		if !s.particlesIdentical(eg.particles[i], bp) {
			return false // clause 3.3
		}
	}
	return true
}

// The predicates below decide the "properties, recursively, are identical"
// relation cos-ct-extends clause 1.2 (c-cte) and cos-particle-extend clauses 1,
// 2 and 3.3 all appeal to. They are NOT ·equivalent· (§3.8.6.3, the
// …Equivalent family in elementconsistent.go), which is a different, weaker
// spec relation, and they are not sameTypeDefinition either, which is component
// identity for one property kind — they are named for the relation they decide.
//
// The recursion is finite BY CONSTRUCTION, with no visited set anywhere (STYLE
// D4 / PRINCIPLES 9): every component-valued slot that could close a cycle — a
// declaration's {type definition}, a <group ref>, an <element ref> — bottoms out
// at COMPONENT IDENTITY (an expanded name, sameTypeDefinition's reading), never
// at a structural walk into the referenced component. What the predicates do
// descend into is the inline value tree of a model group's {particles}, which is
// a finite tree by construction because it follows no reference.
//
// Properties NOT compared are listed at each predicate. Every omission makes the
// relation LOOSER, i.e. reports MORE pairs identical, and since both consumers
// read "identical" as "the clause is satisfied", every omission is FAIL-OPEN.

// attributeUsesIdentical decides property identity between two Attribute Uses
// (§3.5.1): {required}, {value constraint} (presence and both its fields), and
// {inheritable}, plus the {attribute declaration} slot. {annotations} is not
// compared — it carries no schema-significant content.
func (s *Schema) attributeUsesIdentical(a, b AttributeUse) bool {
	if a.required != b.required || a.inheritable != b.inheritable {
		return false
	}
	if a.hasValueConstraint != b.hasValueConstraint || a.valueConstraint != b.valueConstraint {
		return false
	}
	return s.attributeDeclarationsIdentical(a.attributeDeclaration, b.attributeDeclaration)
}

// attributeDeclarationsIdentical decides identity between two {attribute
// declaration} slots. Two AttributeDeclarationRefs naming the same expanded name
// denote the SAME top-level component, which is identity outright. Two
// LocalAttributeDeclarations own distinct components with no name to be resolved
// by, so their properties are compared: {name}, {scope}.{variety}, {value
// constraint}, {inheritable}, and the {type definition} at component identity.
// {annotations} is not compared. The two variants are never identical to each
// other: one names a global declaration, the other owns a local one.
func (s *Schema) attributeDeclarationsIdentical(a, b AttributeDeclarationOrRef) bool {
	switch ad := a.(type) {
	case AttributeDeclarationRef:
		bd, ok := b.(AttributeDeclarationRef)
		return ok && ad.Name == bd.Name
	case LocalAttributeDeclaration:
		bd, ok := b.(LocalAttributeDeclaration)
		if !ok {
			return false
		}
		return s.localAttributeDeclarationsIdentical(ad.Declaration, bd.Declaration)
	default:
		panic("xsd: attributeDeclarationsIdentical: non-exhaustive AttributeDeclarationOrRef switch")
	}
}

// localAttributeDeclarationsIdentical compares two sibling-owned Attribute
// Declarations property by property; see attributeDeclarationsIdentical.
//
// {scope} is compared at its {variety} only, deliberately excluding {parent}
// (§3.2.1 sc_a-parent, wired in #306). The two declarations reaching here are a
// base type's attribute use and the structurally identical use on a type
// extending it, which §3.4.2.4 clause 3's fold gives DIFFERENT parents by
// construction — each names its own containing complex type. Comparing {parent}
// would therefore make every legal extension's copy non-identical and fail
// cos-ct-extends clause 1.2 on it: a false reject, not a tightening.
func (s *Schema) localAttributeDeclarationsIdentical(a, b AttributeDeclaration) bool {
	if a.Name() != b.Name() || a.ScopeVariety() != b.ScopeVariety() || a.Inheritable() != b.Inheritable() {
		return false
	}
	if a.hasValueConstraint != b.hasValueConstraint || a.valueConstraint != b.valueConstraint {
		return false
	}
	return s.typeDefinitionSlotsIdentical(a.TypeDefinition(), b.TypeDefinition())
}

// typeDefinitionSlotsIdentical bottoms the recursion out at COMPONENT IDENTITY
// for a {type definition} slot: two slots denote identical types when they
// resolve to type definitions sameTypeDefinition calls the same, i.e. when they
// carry the same expanded name.
//
// Two cases answer TRUE without deciding anything, and both are deliberate
// fail-open:
//
//   - either side is absent or unresolvable. Nothing is decidable, and Phase A
//     already charged a dangling reference src-resolve.
//   - either side is ANONYMOUS. sameTypeDefinition reports two anonymous types
//     as different (§3.4.6.5's no-identity Note), which is the right reading for
//     cos-ct-derived-ok but the WRONG direction here: c-cte routinely compares a
//     base's own attribute use with itself — foldedAttributeUse returns exactly
//     that whenever T does not re-declare the name — and an inline anonymous
//     attribute type would then make a type identical to itself report as
//     different, false-rejecting an entirely ordinary schema.
func (s *Schema) typeDefinitionSlotsIdentical(a, b TypeDefinitionOrRef) bool {
	ta, aOK := s.typeOf(a)
	tb, bOK := s.typeOf(b)
	if !aOK || !bOK {
		return true
	}
	if typeDefinitionName(ta) == (QName{}) || typeDefinitionName(tb) == (QName{}) {
		return true
	}
	return sameTypeDefinition(ta, tb)
}

// particlesIdentical decides property identity between two Particles (§3.9.1):
// {min occurs}, {max occurs}, and {term}. {annotations} is not compared.
func (s *Schema) particlesIdentical(a, b Particle) bool {
	if a.Occurs() != b.Occurs() {
		return false
	}
	return s.termsIdentical(a.Term(), b.Term())
}

// termsIdentical decides identity between two {term} slots. The two reference
// variants bottom the recursion out at COMPONENT IDENTITY — an <element ref> or
// a <group ref> carrying the same expanded name denotes the same top-level
// component, and the referenced component is NOT walked into, which is what
// keeps this relation finite without a visited set. An inline ResolvedTerm owns
// its component, so its properties are compared.
func (s *Schema) termsIdentical(a, b TermOrRef) bool {
	switch at := a.(type) {
	case ElementDeclarationRef:
		bt, ok := b.(ElementDeclarationRef)
		return ok && at.Name == bt.Name
	case ModelGroupRef:
		bt, ok := b.(ModelGroupRef)
		return ok && at.Name == bt.Name
	case ResolvedTerm:
		bt, ok := b.(ResolvedTerm)
		return ok && s.resolvedTermsIdentical(at.Term, bt.Term)
	default:
		panic("xsd: termsIdentical: non-exhaustive TermOrRef switch")
	}
}

// resolvedTermsIdentical decides identity between two inline {term} components,
// exhaustively over the sealed Term sum. Two terms of different kinds are never
// identical.
func (s *Schema) resolvedTermsIdentical(a, b Term) bool {
	switch at := a.(type) {
	case ElementDeclaration:
		bt, ok := b.(ElementDeclaration)
		return ok && s.elementDeclarationsIdentical(at, bt)
	case Wildcard:
		bt, ok := b.(Wildcard)
		return ok && wildcardsIdentical(at, bt)
	case ModelGroup:
		bt, ok := b.(ModelGroup)
		return ok && s.modelGroupsIdentical(at, bt)
	default:
		panic("xsd: resolvedTermsIdentical: non-exhaustive Term switch")
	}
}

// modelGroupsIdentical decides property identity between two Model Groups
// (§3.8.1): {compositor} and {particles}, position by position — the property is
// spec-worded a SEQUENCE, so order is significant (modelgroup.go). This is the
// one descending recursion, and it descends only into an inline value tree that
// follows no reference, so it terminates without a visited set. {annotations} is
// not compared.
func (s *Schema) modelGroupsIdentical(a, b ModelGroup) bool {
	if a.Compositor() != b.Compositor() || len(a.particles) != len(b.particles) {
		return false
	}
	for i, ap := range a.particles {
		if !s.particlesIdentical(ap, b.particles[i]) {
			return false
		}
	}
	return true
}

// elementDeclarationsIdentical decides property identity between two inline
// Element Declarations (§3.3.1): {name}, {nillable}, {abstract}, {value
// constraint}, {disallowed substitutions}, {substitution group exclusions},
// {substitution group affiliations}, and the {type definition} at component
// identity.
//
// NOT compared, each omission making the relation looser and so fail-open:
// {scope} (a copied particle keeps the {parent} of the type it was copied FROM,
// so comparing it would reject the very shape §3.4.2.3.3 clause 4.2.3.3 builds),
// {identity-constraint definitions}, {type table}, and {annotations}.
func (s *Schema) elementDeclarationsIdentical(a, b ElementDeclaration) bool {
	if a.Name() != b.Name() || a.Nillable() != b.Nillable() || a.Abstract() != b.Abstract() {
		return false
	}
	if a.hasValueConstraint != b.hasValueConstraint || a.valueConstraint != b.valueConstraint {
		return false
	}
	if !derivationMethodsIdentical(a.disallowedSubstitutions, b.disallowedSubstitutions) {
		return false
	}
	if !derivationMethodsIdentical(a.substitutionGroupExclusions, b.substitutionGroupExclusions) {
		return false
	}
	if !qnamesIdentical(a.substitutionGroupAffiliations, b.substitutionGroupAffiliations) {
		return false
	}
	return s.typeDefinitionSlotsIdentical(a.TypeDefinition(), b.TypeDefinition())
}

// wildcardsIdentical decides property identity between two Wildcards (§3.10.1):
// {namespace constraint} and {process contents}. {annotations} is not compared.
func wildcardsIdentical(a, b Wildcard) bool {
	if a.ProcessContents() != b.ProcessContents() {
		return false
	}
	return namespaceConstraintsIdentical(a.NamespaceConstraint(), b.NamespaceConstraint())
}

// namespaceConstraintsIdentical decides property identity between two Namespace
// Constraints (§3.10.1): {variety}, {namespaces}, and both halves of the
// partitioned {disallowed names} (namespaceconstraint.go). Each set is held as a
// document-order slice and compared position by position, so no map iteration
// reaches the verdict (STYLE D2). Two constraints that hold the same members in
// a different order are reported as different — the strict direction, but the
// producer emits one document order per source wildcard, so it is not reachable
// from a schema document.
func namespaceConstraintsIdentical(a, b NamespaceConstraint) bool {
	if a.variety != b.variety || len(a.namespaces) != len(b.namespaces) {
		return false
	}
	for i, ns := range a.namespaces {
		if ns != b.namespaces[i] {
			return false
		}
	}
	if !qnamesIdentical(a.disallowedNames, b.disallowedNames) {
		return false
	}
	return disallowedNameKeywordsIdentical(a.disallowedNameKeywords, b.disallowedNameKeywords)
}

// derivationMethodsIdentical compares two {final}/{prohibited substitutions}-
// style keyword slices position by position.
func derivationMethodsIdentical(a, b []DerivationMethod) bool {
	if len(a) != len(b) {
		return false
	}
	for i, m := range a {
		if m != b[i] {
			return false
		}
	}
	return true
}

// qnamesIdentical compares two document-order expanded-name slices position by
// position.
func qnamesIdentical(a, b []QName) bool {
	if len(a) != len(b) {
		return false
	}
	for i, n := range a {
		if n != b[i] {
			return false
		}
	}
	return true
}

// disallowedNameKeywordsIdentical compares the keyword half of a partitioned
// {disallowed names} property position by position.
func disallowedNameKeywordsIdentical(a, b []DisallowedNameKeyword) bool {
	if len(a) != len(b) {
		return false
	}
	for i, k := range a {
		if k != b[i] {
			return false
		}
	}
	return true
}
