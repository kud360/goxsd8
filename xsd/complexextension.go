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
// apply when B is a complex type definition. They are checked in spec order, so
// the first reported failure is deterministic (STYLE D1).
//
// GAP(xsd): clauses 1.3 and 1.7 are NOT charged, on exactly the footing
// checkDerivationOKRestriction records for derivation-ok-restriction clause 5.
//
//   - 1.3 — if B has an {attribute wildcard}, T must have one whose {namespace
//     constraint} is a superset of B's per cos-ns-subset. §3.4.2.5 clause 2.2
//     makes an extension's {attribute wildcard} the ·attribute wildcard union·
//     of its own with the base's, which satisfies 1.3 UNCONDITIONALLY for a
//     faithfully mapped type; the clause exists to constrain components
//     assembled by other means. No producer in this repo performs that union
//     (parser/produce_complex.go maps the type's OWN <anyAttribute> only), so a
//     produced extension that declares a narrower own wildcard — or, far more
//     commonly, none at all — reads as a 1.3 violation and is not one. The
//     excuse is the missing fold, NOT a missing subset relation: wildcardSubset
//     (namespaceconstraint_subset.go) already implements cos-ns-subset and is
//     directly usable the day the fold lands. Charging the cheap half alone
//     ("B has a wildcard, T has none") would be precisely the false reject.
//   - 1.7 — B.{assertions} is a prefix of T.{assertions}. §3.4.2.1 clause 1
//     makes a complex type's {assertions} the base's followed by its own
//     <assert> children; the producer maps the type's own children only, so on
//     every produced extension of a base carrying an assertion the prefix fails
//     and charging it would be a FALSE REJECT of a valid schema.
//
// Both skips are FAIL-OPEN — a missing rejection, never a false one — and both
// land with the fold (#265).
func (s *Schema) checkExtensionOfComplexBase(t, b ComplexType) error {
	if err := checkExtensionBaseFinal(t, b); err != nil {
		return err
	}
	if err := s.checkExtensionAttributeUses(t, b); err != nil {
		return err
	}
	if err := s.checkExtensionContentType(t, b); err != nil {
		return err
	}
	if err := s.checkExtensionTwoStepDerivable(t); err != nil {
		return err
	}
	return s.checkExtensionLocallyDeclaredTypes(t, b)
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
// The lookup in T is foldedAttributeUse — T's own uses with the base chain
// folded in, as §3.4.2.4 clause 3 defines {attribute uses} — and that is
// load-bearing in the REJECT direction, not leniency. The producer maps a type's
// OWN <attribute> children and no more (#265), so a use T holds by clause 3.1's
// inheritance reads as absent on the component; charging that absence would
// false-reject every ordinary extension whose base declares an attribute. What
// the clause then actually charges is the unambiguous shape: T re-declares the
// name itself with different properties.
//
// The comparison is attributeUsesIdentical, NOT the ·subsumption· apparatus
// c-ran uses (attributeDefaultBinding/bindingSubsumes, defaultbinding.go).
// c-cte asks for property IDENTITY, which is strictly stronger than
// ·subsumption·; reusing the looser relation would decide a different constraint.
//
// The "no counterpart at all" rejection below states the clause's set-inclusion
// half and is not reachable while the fold is reconstructed by a chain walk:
// T.{base type definition} IS B, so foldedAttributeUse always reaches B's own
// uses and finds the name. It is written rather than dropped because the
// lookup's second result must decide something (STYLE S3), and because it is the
// rejection that becomes live the day #265 makes {attribute uses} a stored,
// genuinely folded property that a caller could assemble incompletely.
func (s *Schema) checkExtensionAttributeUses(t, b ComplexType) error {
	for _, u := range b.attributeUses {
		name := attributeUseName(u)
		tu, ok := s.foldedAttributeUse(t, name)
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
// GAP(xsd): a chain that MIXES extension and restriction steps is not decided —
// it is accepted. Deciding it needs the intermediate type the Note describes,
// and synthesizing that means merging a base particle with an ·effective
// content· the way §3.4.2.3.3 clause 4.2 does, repeatedly, over a chain that has
// been re-ordered into an order no schema document expresses. That merge exists
// in this repo only as the finalize-phase-eligible construction the producer
// performs for one real <extension>; it is not built for arbitrary reordered
// chains, and re-deriving it here would be the same rule in a second encoding
// (STYLE T4). The direction is FAIL-OPEN — a mixed chain that violates clause
// 1.5 is accepted, never a false reject — and it should be filed as a follow-up
// xsd issue. Note for whoever picks it up: the natural final test is
// s.contentTypeRestricts(T's {content type}, the collapsed one), and that
// function early-accepts whenever either side carries {open content}
// (contentrestricts.go:219-231), so an open-content extension would inherit that
// skip into clause 1.5 as well.
func (s *Schema) checkExtensionTwoStepDerivable(t ComplexType) error {
	if s.pureExtensionChain(t) {
		return nil // clause 1.5 holds; see the proof above
	}
	return nil // GAP(xsd): the mixed chain is not decided; see the doc above
}

// pureExtensionChain reports whether every step of t's {base type definition}
// chain, up to the ancestor whose base is ·xs:anyType·, has {derivation method}
// = extension. A chain that leaves the complex-type graph before reaching
// xs:anyType — an unresolvable base, or a simple one — is not one of the pure
// chains clause 1.5's identity argument covers, so it answers false and the
// caller's GAP applies.
func (s *Schema) pureExtensionChain(t ComplexType) bool {
	c := t
	for {
		if c.Name() == anyTypeName {
			return true // the ur-type itself terminates the walk
		}
		if c.DerivationMethod() != DerivationExtension {
			return false
		}
		baseName := c.BaseTypeDefinitionName()
		if baseName == anyTypeName {
			return true // c IS the ancestor whose {base type definition} is ·xs:anyType·
		}
		next, ok := s.baseComplexType(c)
		if !ok {
			return false
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
// D4 / PRINCIPLES 5): every component-valued slot that could close a cycle — a
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
