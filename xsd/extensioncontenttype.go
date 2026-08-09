package xsd

import (
	"fmt"

	"github.com/kud360/goxsd8/xsderr"
)

// This file holds one mapping rule and its structural inverse: Mapping Rules for
// Complex Types with Complex Content (Structures §3.4.2.3.3) clause 4 — the
// ·explicit content type· — and the recovery of one extension step's own
// ·effective content· from the merged result.
//
// The rule lived in the PRODUCER until #392, as (*parser.producer).
// extensionContentType/extensionParticle/allGroupOf, because a <complexContent>
// <extension> is the only place a schema DOCUMENT asks for it. cos-ct-extends
// (§3.4.6.2) clause 1.5 asks for it again, over a chain re-ordered into an order
// no document expresses (collapsedintermediate.go), and xsd cannot import parser
// — parser imports xsd. So the rule moved down here, where both callers reach it,
// rather than being stated a second time (STYLE T4). ExtensionContentType is the
// producer's entry point and the ONE name this file exports; everything else,
// the inverse included, is in-package.
//
// The inverse sits in this file, immediately below the merge it inverts, and
// nowhere else: an inverter that drifts from its merge is the T4 failure mode
// this placement exists to make visible. Any edit to a clause 4.2 sub-case is an
// edit to recoverExtensionStepContent's matching row, and the round-trip test
// (extensioncontenttype_test.go) is what fails when only one of the two moves.

// ExtensionContentType is §3.4.2.3.3 clause 4.2 (ct-extension): the ·explicit
// content type· of an extension-derived complex content, computed from the
// RESOLVED {base type definition} and this derivation's own ·effective content·.
//
//   - 4.2.1 (c-ctes): a simple base, or a complex base whose {content type} has
//     {variety} empty or simple, contributes no particle — the result is clause
//     4.1.1/4.1.2's, delegated to explicitContentType;
//   - 4.2.2: a complex base with element-only or mixed content and an ***empty***
//     ·effective content· yields the base's ENTIRE Content Type record, {open
//     content} included — a different sharing rule from 4.2.3, which takes only
//     {open content} from the base;
//   - 4.2.3: otherwise {variety} is mixed iff ·effective mixed·, {particle} is
//     extensionParticle's merge of the ·base particle· with the ·effective
//     content·, {open content} is the base's, and {simple type definition} is
//     ·absent· — automatic here, since ElementContent has no such field.
//
// Nothing is deep-copied: the base particle and the effective content enter the
// synthesized structure as the very values the base and this derivation already
// hold, so "the same particles appear in both the base type definition and the
// extension" (xr.ctd.n4-bis). A Go copy of an immutable value component IS that
// component; no identity marker is added to carry the fact (STYLE D3).
//
// allGroup reads a <group ref> to the Model Group it denotes (§3.7.2, xr.mgd3),
// which clause 4.2.3's sub-case test needs — see allGroupOfParticle. It is a
// parameter and NOT a general extension point: this package's own caller has
// s.modelGroupIndex and passes Schema.modelGroupNamed, while the producer calls
// at produce time, before any *Schema exists, and must build a referenced
// definition on demand. Two call sites, two lookup regimes, one merge. An error
// it returns is wrapped at this boundary (STYLE E1) so a producer-built failure
// arrives naming the clause it was consulted for.
//
// loc is charged to every rejection this construction can raise; a caller with
// no real source position passes the zero xsderr.Loc{}.
func ExtensionContentType(loc xsderr.Loc, base TypeDefinition, effective *Particle, explicitEmpty, effectiveMixed bool, allGroup func(QName) (ModelGroup, bool, error)) (ContentType, error) {
	switch b := base.(type) {
	case *SimpleType:
		return explicitContentType(effective, effectiveMixed), nil // 4.2.1, simple base
	case ComplexType:
		return extensionContentTypeOver(loc, b.ContentType(), effective, explicitEmpty, effectiveMixed, allGroup)
	default:
		panic("xsd: ExtensionContentType: non-exhaustive TypeDefinition switch")
	}
}

// extensionContentTypeOver is clause 4.2 over a base CONTENT TYPE rather than a
// base COMPONENT. It is the whole of the clause bar its simple-base arm, and it
// is what the clause-1.5 collapse needs: there the "base" is the partially
// collapsed intermediate's {content type}, a value that belongs to no component
// (collapsedintermediate.go).
func extensionContentTypeOver(loc xsderr.Loc, base ContentType, effective *Particle, explicitEmpty, effectiveMixed bool, allGroup func(QName) (ModelGroup, bool, error)) (ContentType, error) {
	switch bc := base.(type) {
	case EmptyContent, SimpleContent:
		return explicitContentType(effective, effectiveMixed), nil // 4.2.1
	case ElementContent:
		if effective == nil {
			return bc, nil // 4.2.2: the base's whole Content Type record
		}
		particle, err := extensionParticle(loc, bc.Particle, *effective, explicitEmpty, allGroup)
		if err != nil {
			return nil, err
		}
		// 4.2.3: {open content} comes from the base, {particle} is the merge.
		return ElementContent{Mixed: effectiveMixed, Particle: particle, OpenContent: bc.OpenContent}, nil
	default:
		panic("xsd: extensionContentTypeOver: non-exhaustive ContentType switch")
	}
}

// explicitContentType is §3.4.2.3.3 clause 4.1 — the restriction branch's
// ·explicit content type· — as a total function of the already-computed
// ·effective content·: clause 4.1.1 (empty effective content ⇒ {variety} empty,
// which admits NO character content at all, unlike element-only) and clause 4.1.2
// (otherwise mixed iff ·effective mixed·). Clause 4.2.1 routes the extension
// cases with a simple or empty/simple-content base through this same function,
// which is what "a Content Type as per clause 4.1.1 and clause 4.1.2 above"
// means.
//
// The producer states clause 4.1 a second time, for the RESTRICTION branch it
// alone can reach (parser's explicitContentType). That is deliberate and was
// ruled at #392's warden pre-flight: the alternative was a second exported name
// on this package for a two-line total function whose one out-of-package caller
// is the producer's restriction arm (STYLE T5). The two encodings are pinned
// against each other by parser's own content-type tests; a change to either
// clause 4.1.1 or 4.1.2 is a change to both sites.
func explicitContentType(effective *Particle, effectiveMixed bool) ContentType {
	if effective == nil {
		return EmptyContent{} // clause 4.1.1
	}
	return ElementContent{Mixed: effectiveMixed, Particle: *effective} // clause 4.1.2
}

// extensionParticle merges the ·base particle· with the ·effective content· per
// §3.4.2.3.3 clause 4.2.3's three sub-cases:
//
//   - 4.2.3.1: the base particle's {term} is an all group and the ·explicit
//     content· is empty ⇒ the base particle itself, unchanged;
//   - 4.2.3.2: both terms are all groups ⇒ a particle whose {min occurs} is the
//     effective content's, {max occurs} 1, and {term} an all group holding the
//     base group's {particles} followed by the effective group's;
//   - 4.2.3.3 (c-suffix-extension): otherwise a 1..1 particle over a SEQUENCE of
//     the base particle followed by the effective content.
//
// Both operands are spliced in as-is (xr.ctd.n4-bis: particles are reused, not
// copied), which is also what lets a wildcard's ##definedSibling see
// base-declared element names as siblings — the derived {content type} genuinely
// contains the base's particle, so this package's content-model walk needs no
// {base type definition} edge.
//
// The sub-case test reads THROUGH a <group ref>: a {term} that is a ModelGroupRef
// to an all-bodied model group definition selects 4.2.3.1/4.2.3.2 exactly as an
// inline <all> does (allGroupOfParticle), because §3.7.2 makes the {term} of such
// a particle the definition's {model group} — the reference is a source spelling,
// not a different component. Reading only the inline spelling would send an
// all-bodied base down 4.2.3.3, whose synthesized sequence wrapping two all
// groups is a shape cos-all-limited (§3.8.6.2) clause 1 forbids outright.
func extensionParticle(loc xsderr.Loc, baseParticle, effective Particle, explicitEmpty bool, allGroup func(QName) (ModelGroup, bool, error)) (Particle, error) {
	baseGroup, baseIsAll, err := allGroupOfParticle(baseParticle, allGroup)
	if err != nil {
		return Particle{}, err
	}
	if baseIsAll && explicitEmpty {
		return baseParticle, nil // 4.2.3.1
	}
	effectiveGroup, effectiveIsAll, err := allGroupOfParticle(effective, allGroup)
	if err != nil {
		return Particle{}, err
	}
	if baseIsAll && effectiveIsAll {
		// 4.2.3.2: one all group over both {particles} lists, base's first.
		merged := append(baseGroup.Particles(), effectiveGroup.Particles()...)
		mg, err := NewModelGroup(loc, CompositorAll, merged, nil)
		if err != nil {
			return Particle{}, err
		}
		occ, err := NewOccurs(loc, effective.Occurs().Min(), 1)
		if err != nil {
			return Particle{}, err
		}
		return NewParticle(loc, occ, ResolvedTerm{Term: mg}, nil)
	}
	// 4.2.3.3: a 1..1 sequence, base particle then effective content.
	seq, err := NewModelGroup(loc, CompositorSequence, []Particle{baseParticle, effective}, nil)
	if err != nil {
		return Particle{}, err
	}
	oneOne, err := NewOccurs(loc, 1, 1)
	if err != nil {
		return Particle{}, err
	}
	return NewParticle(loc, oneOne, ResolvedTerm{Term: seq}, nil)
}

// allGroupOfParticle returns the Model Group a particle's {term} is when that
// group's {compositor} is all (§3.8.1), reporting false for every other {term}:
// an element declaration, a wildcard, a choice/sequence group, or an <element
// ref>. A <group ref> {term} is followed through the injected lookup (see
// ExtensionContentType); a ref that resolves to nothing simply reports false,
// leaving src-resolve clause 1.5 and mg-props-correct clause 2 to their own
// phases.
//
// It is the error-carrying twin of resolveTermGroup (effectivetotalrange.go),
// which answers the same question for an already-finalized *Schema, where a
// lookup cannot fail. The two are not merged because this one exists precisely
// to serve a caller whose lookup CAN fail — the producer's on-demand build of a
// referenced model group definition — and folding that error channel into
// resolveTermGroup would put it on every in-package reader that has no use for
// it.
func allGroupOfParticle(part Particle, allGroup func(QName) (ModelGroup, bool, error)) (ModelGroup, bool, error) {
	switch t := part.Term().(type) {
	case ResolvedTerm:
		mg, ok := t.Term.(ModelGroup)
		if !ok || mg.Compositor() != CompositorAll {
			return ModelGroup{}, false, nil
		}
		return mg, true, nil
	case ModelGroupRef:
		mg, ok, err := allGroup(t.Name)
		if err != nil {
			// The lookup is the caller's, so its failure arrives with the
			// caller's own rule and position already attached; this adds the
			// context of what THIS package was asking it for and no second rule
			// (STYLE E1/E2 — %w keeps xsderr.RuleOf reaching the real verdict).
			return ModelGroup{}, false, fmt.Errorf("xsd: reading the <group ref> to %s for the §3.4.2.3.3 clause 4.2.3 sub-case test: %w", t.Name, err)
		}
		if !ok || mg.Compositor() != CompositorAll {
			return ModelGroup{}, false, nil
		}
		return mg, true, nil
	case ElementDeclarationRef:
		return ModelGroup{}, false, nil
	default:
		panic("xsd: allGroupOfParticle: non-exhaustive TermOrRef switch")
	}
}

// modelGroupNamed is this package's own group lookup for the injected parameter
// ExtensionContentType and recoverExtensionStepContent take: the §3.17.1 model
// group definition symbol table, read by expanded name. It cannot fail — the
// definitions are already built — so the error result is always nil, and exists
// only because the producer's regime, which builds on demand, can fail.
func (s *Schema) modelGroupNamed(name QName) (ModelGroup, bool, error) {
	mgd, ok := s.modelGroupIndex[name]
	if !ok {
		return ModelGroup{}, false, nil
	}
	return mgd.ModelGroup(), true, nil
}

// recoveredContent is one extension step's own contribution to §3.4.2.3.3 clause
// 4.2, recovered from the merged result: the three arguments the merge consumed
// beside the base. It is a value object holding exactly what
// extensionContentTypeOver takes, so a caller re-applies it by passing the
// fields straight back (collapsedintermediate.go).
type recoveredContent struct {
	effective      *Particle
	explicitEmpty  bool
	effectiveMixed bool
}

// recoverExtensionStepContent inverts §3.4.2.3.3 clause 4.2 structurally: given
// an extension step's merged {content type} c and the {content type} b of the
// base it was merged against, it recovers the ·effective content· that step
// contributed. cos-ct-extends clause 1.5 needs it because the re-ordering the
// §3.4.6.2 Note prescribes re-applies each extension step against a DIFFERENT
// base — the collapsed intermediate so far — and the component model stores only
// the merged result, never the step's own contribution.
//
// Every clause 4.2 sub-case is inverted by the row that built it, and the rows
// are tested most-specific first so a shape that two rows could explain is read
// as the one the merge would have produced:
//
//   - 4.2.1 (b empty or simple): c IS clause 4.1's output — an EmptyContent means
//     an ***empty*** ·effective content·, an ElementContent means the effective
//     content is its {particle} and ·effective mixed· its Mixed.
//   - 4.2.2 (b element-only/mixed, ·effective content· ***empty***): c is b's
//     whole record, so c.{particle} is property-identical to b's and b's {term}
//     is NOT an all group — recovered as a nil ·effective content·.
//   - 4.2.3.1 (b's {term} an all group, ·explicit content· empty): c.{particle}
//     is likewise b's, but the base being an all group is what selects this row
//     over 4.2.2. The merge reaches it only with a non-nil ·effective content·
//     whose ·explicit content· was empty, and §3.4.2.3.3 clause 3.1.1 makes that
//     ONE value — a 1..1 particle over an empty sequence — so the recovery is
//     exact rather than a guess.
//   - 4.2.3.2 (both all groups): c.{particle} is an all group whose {particles}
//     open with b's, so the effective content is an all group over the SUFFIX,
//     with c.{particle}'s {min occurs}. See below on {max occurs}.
//   - 4.2.3.3 (otherwise): c.{particle} is a 1..1 sequence of exactly two
//     members whose first is property-identical to b's — the effective content is
//     the second. explicitEmpty is recovered as false: the merge consults it only
//     when the base particle is an all group (4.2.3.1's guard), which this row
//     has already excluded, so the merge is constant in it here.
//   - anything else: DECLINED (ok false). The rows above are the exact inverse of
//     the only construction the producer performs, so the decline is unreachable
//     for a parser-produced component and live only for one assembled through
//     [SchemaBuilder] past the merge. See checkExtensionTwoStepDerivable for the
//     direction it fails in and the issue that owns it.
//
// 4.2.3.2 is the one lossy row: the merge forces {max occurs} = 1 on its result,
// so the effective content's own {max occurs} is not readable back. It is lossy
// only for a component no schema document can express — cos-all-limited
// (§3.8.6.2) clause 1.2 requires {max occurs} = 1 of any particle over an all
// group that is a complex type's content particle, and the schema for schema
// documents admits only maxOccurs ∈ {0,1} on <all> (a 0 makes the ·explicit
// content· empty by clause 2.1.4, which is 4.2.3.1's row, not this one). So for
// every source-derived component the recovered 1 IS the original, and the
// residue is a programmatically assembled all-group particle that violates
// cos-all-limited to begin with.
//
// The all-group reading goes through modelGroupNamed rather than the injected
// lookup ExtensionContentType takes: the inverse has exactly one caller, in this
// package, on a *Schema whose model group definitions are all built.
//
// loc is charged to the two particles this recovery CONSTRUCTS (4.2.3.1's
// stand-in and 4.2.3.2's suffix group) and to their rejections. A Particle
// carries no provenance of its own, so the caller passes the position of the
// type whose derivation is being decided.
func (s *Schema) recoverExtensionStepContent(loc xsderr.Loc, c, b ContentType) (recoveredContent, bool, error) {
	bc, baseIsElement := b.(ElementContent)
	if !baseIsElement {
		return recoverExplicitContent(c) // 4.2.1
	}
	cc, ok := c.(ElementContent)
	if !ok {
		// A merge over an element-only/mixed base yields element content on
		// every row of 4.2.2/4.2.3; empty or simple content here is a shape the
		// merge does not build.
		return recoveredContent{}, false, nil
	}
	baseGroup, baseIsAll, err := allGroupOfParticle(bc.Particle, s.modelGroupNamed)
	if err != nil {
		return recoveredContent{}, false, err
	}
	if s.particlesIdentical(cc.Particle, bc.Particle) {
		return recoverUnchangedBaseParticle(loc, cc.Mixed, baseIsAll) // 4.2.2, 4.2.3.1
	}
	if baseIsAll {
		suffix, ok, err := s.recoverAllGroupSuffix(loc, cc.Particle, baseGroup)
		if err != nil || ok {
			return recoveredContent{effective: suffix, effectiveMixed: cc.Mixed}, ok, err // 4.2.3.2
		}
	}
	tail, ok := s.recoverSequenceTail(cc.Particle, bc.Particle)
	if !ok {
		return recoveredContent{}, false, nil
	}
	return recoveredContent{effective: tail, effectiveMixed: cc.Mixed}, true, nil // 4.2.3.3
}

// recoverUnchangedBaseParticle splits the two rows that leave the base particle
// untouched, on the fact that discriminates them in the merge: a base whose
// {term} is an all group took 4.2.3.1 (a non-empty ·effective content· whose
// ·explicit content· was empty — clause 3.1.1's stand-in, the one value that
// reaches it), and any other base took 4.2.2 (an ***empty*** ·effective
// content·). The merge tests 4.2.2 first, so an all-group base with a genuinely
// empty effective content is read back as 4.2.3.1's stand-in; the two re-merge
// to the same particle and differ only in a {variety} that cos-ct-extends clause
// 1.4.3.2.2.1 already forces to agree. The recovered explicitEmpty is TRUE on
// both rows, and on 4.2.2's it is exact rather than conventional: the merge
// takes that row only for an ***empty*** ·effective content·, which §3.4.2.3.3
// clause 3.1.2 produces only from an empty ·explicit content·.
func recoverUnchangedBaseParticle(loc xsderr.Loc, mixed, baseIsAll bool) (recoveredContent, bool, error) {
	if !baseIsAll {
		return recoveredContent{explicitEmpty: true, effectiveMixed: mixed}, true, nil // 4.2.2
	}
	stand, err := emptySequenceParticle(loc)
	if err != nil {
		return recoveredContent{}, false, err
	}
	return recoveredContent{effective: &stand, explicitEmpty: true, effectiveMixed: mixed}, true, nil // 4.2.3.1
}

// recoverExplicitContent inverts clause 4.1 (reached through 4.2.1, for a base
// whose {content type} is empty or simple): the ·effective content· is the whole
// of c's {particle}, or ***empty*** when c is EmptyContent. Simple content is
// declined — clause 4.2.1 never yields it, and a <simpleContent><extension> step
// is §3.4.2.2's tableau, which the collapse handles without asking this.
func recoverExplicitContent(c ContentType) (recoveredContent, bool, error) {
	switch cc := c.(type) {
	case EmptyContent:
		return recoveredContent{explicitEmpty: true}, true, nil // clause 4.1.1
	case ElementContent:
		p := cc.Particle
		return recoveredContent{effective: &p, effectiveMixed: cc.Mixed}, true, nil // clause 4.1.2
	case SimpleContent:
		return recoveredContent{}, false, nil
	default:
		panic("xsd: recoverExplicitContent: non-exhaustive ContentType switch")
	}
}

// recoverAllGroupSuffix inverts clause 4.2.3.2: merged is an all-group particle
// whose group's {particles} open with the base group's, so the ·effective
// content· is a particle over an all group holding the remaining members, with
// merged's own {min occurs} and {max occurs} = 1 (see
// recoverExtensionStepContent on why the 1 is exact for any source-derived
// component). It answers false when the shape is not 4.2.3.2's.
func (s *Schema) recoverAllGroupSuffix(loc xsderr.Loc, merged Particle, baseGroup ModelGroup) (*Particle, bool, error) {
	mergedGroup, mergedIsAll, err := allGroupOfParticle(merged, s.modelGroupNamed)
	if err != nil {
		return nil, false, err
	}
	if !mergedIsAll || len(mergedGroup.particles) < len(baseGroup.particles) {
		return nil, false, nil
	}
	for i, bp := range baseGroup.particles {
		if !s.particlesIdentical(mergedGroup.particles[i], bp) {
			return nil, false, nil
		}
	}
	g, err := NewModelGroup(loc, CompositorAll, mergedGroup.particles[len(baseGroup.particles):], nil)
	if err != nil {
		return nil, false, err
	}
	occ, err := NewOccurs(loc, merged.Occurs().Min(), 1)
	if err != nil {
		return nil, false, err
	}
	p, err := NewParticle(loc, occ, ResolvedTerm{Term: g}, nil)
	if err != nil {
		return nil, false, err
	}
	return &p, true, nil
}

// recoverSequenceTail inverts clause 4.2.3.3: merged is a 1..1 particle over a
// SEQUENCE of exactly two members whose first is property-identical to the base
// particle, so the ·effective content· is the second. The two-member shape is
// what the merge builds and nothing else; a longer sequence is declined rather
// than read as "base plus everything after it", which would invent a grouping the
// merge never produced.
func (s *Schema) recoverSequenceTail(merged, baseParticle Particle) (*Particle, bool) {
	maxOccurs, bounded := merged.Occurs().Max()
	if merged.Occurs().Min() != 1 || !bounded || maxOccurs != 1 {
		return nil, false
	}
	g, ok := s.resolveTermGroup(merged.Term())
	if !ok || g.Compositor() != CompositorSequence || len(g.particles) != 2 {
		return nil, false
	}
	if !s.particlesIdentical(g.particles[0], baseParticle) {
		return nil, false
	}
	tail := g.particles[1]
	return &tail, true
}

// emptySequenceParticle is the ·effective content· §3.4.2.3.3 clause 3.1.1
// substitutes for a mixed content with no model-group child: a 1..1 particle over
// an empty sequence, which admits character content and no elements. The producer
// builds the same value at produce time; this one is built at finalize, by
// clause 4.2.3.1's inverse, for a step whose source is long gone.
func emptySequenceParticle(loc xsderr.Loc) (Particle, error) {
	seq, err := NewModelGroup(loc, CompositorSequence, nil, nil)
	if err != nil {
		return Particle{}, err
	}
	oneOne, err := NewOccurs(loc, 1, 1)
	if err != nil {
		return Particle{}, err
	}
	return NewParticle(loc, oneOne, ResolvedTerm{Term: seq}, nil)
}
