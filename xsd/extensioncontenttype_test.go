package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: the §3.4.2.3.3 clause 4 merge is exported
// (xsd.ExtensionContentType, for the producer) but its inverse is not, and the
// round trip is the whole subject here.

// eStandIn is §3.4.2.3.3 clause 3.1.1's ·effective content· for a mixed content
// with no model-group child: a 1..1 particle over an EMPTY sequence. It is the
// one value clause 4.2.3.1 is ever reached with, which is what makes that
// sub-case invertible exactly.
func eStandIn(t *testing.T) Particle {
	t.Helper()
	p, err := emptySequenceParticle(xsderr.Loc{})
	if err != nil {
		t.Fatalf("emptySequenceParticle: %v", err)
	}
	return p
}

// eAllParticle wraps element declarations in a particle over an ·all· group with
// the given {min occurs} and {max occurs} = 1 — the only shape cos-all-limited
// (§3.8.6.2) clause 1.2 permits as a complex type's content particle, and the one
// clause 4.2.3.2's inverse relies on to recover the {max occurs} it cannot read
// back.
func eAllParticle(t *testing.T, minOccurs int, names ...QName) Particle {
	t.Helper()
	members := make([]Particle, 0, len(names))
	for _, n := range names {
		members = append(members, uOne(t, ResolvedTerm{Term: uLocal(t, n, uq("T"))}))
	}
	return uParticle(t, uOccurs(t, minOccurs, 1), ResolvedTerm{Term: uGroup(t, CompositorAll, members...)})
}

// eSeqParticle wraps element declarations in a 1..1 particle over a SEQUENCE
// group — a base particle that is NOT an all group, which is what sends the merge
// down clause 4.2.2 rather than 4.2.3.1 when the ·effective content· is empty.
func eSeqParticle(t *testing.T, names ...QName) Particle {
	t.Helper()
	members := make([]Particle, 0, len(names))
	for _, n := range names {
		members = append(members, uOne(t, ResolvedTerm{Term: uLocal(t, n, uq("T"))}))
	}
	return uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence, members...)})
}

// TestExtensionContentTypeRoundTrip is the coupling guard between §3.4.2.3.3
// clause 4's merge and the structural inverse cos-ct-extends clause 1.5 recovers
// each extension step's own ·effective content· with: for every shape the merge
// builds, inverting its output must give back the ·effective content· that went
// in. An inverter that drifts from its merge is the failure mode that placement
// beside each other cannot prevent on its own; this test is what fails when only
// one of the two moves.
//
// All six shapes are covered. Each row states the sub-case it exercises and what
// discriminates it from its neighbour:
//
//   - 4.2.1 twice, over an empty base — with and without an ·effective content·.
//     The recovered explicitEmpty is the merge's own constant on this arm (it
//     consults the flag only for an all-group base), so the rows pass the value
//     the merge is insensitive to and the round trip stays exact.
//   - 4.2.2 and 4.2.3.1 both leave the base particle untouched, and the base's
//     {compositor} is what tells them apart: 4.2.2's base here is a SEQUENCE,
//     4.2.3.1's an ·all· group. Their {variety}s must agree with the base's for
//     4.2.2, which the merge drops — cos-ct-extends clause 1.4.3.2.2.1 forces the
//     same agreement on any real chain.
//   - 4.2.3.2 is the one lossy shape, and the row pins why it is lossless in
//     practice: the effective content's {max occurs} is forced to 1 by the merge,
//     and cos-all-limited clause 1.2 already requires 1 of it.
//   - 4.2.3.3 is the residual sequence-wrapping case.
func TestExtensionContentTypeRoundTrip(t *testing.T) {
	s := xSchema(t, func(*SchemaBuilder) {})
	seqBase := ElementContent{Particle: eSeqParticle(t, uq("b1"))}
	allBase := ElementContent{Particle: eAllParticle(t, 1, uq("b1"))}
	own := eSeqParticle(t, uq("e1"))
	for _, tc := range []struct {
		name string
		base ContentType
		in   recoveredContent
	}{
		{"4.2.1 over an empty base, ·effective content· ***empty***", EmptyContent{},
			recoveredContent{explicitEmpty: true}},
		{"4.2.1 over a simple base, with an ·effective content·", SimpleContent{SimpleType: AnyAtomicType()},
			recoveredContent{effective: &own, effectiveMixed: true}},
		{"4.2.2: element base, ·effective content· ***empty***", seqBase,
			recoveredContent{explicitEmpty: true}},
		{"4.2.3.1: all-group base, ·explicit content· empty", allBase,
			recoveredContent{effective: ptr(eStandIn(t)), explicitEmpty: true, effectiveMixed: true}},
		{"4.2.3.2: all-group base, all-group ·effective content·", allBase,
			recoveredContent{effective: ptr(eAllParticle(t, 0, uq("e1"), uq("e2")))}},
		{"4.2.3.3: sequence base, sequence ·effective content·", seqBase,
			recoveredContent{effective: &own}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := extensionContentTypeOver(xsderr.Loc{}, tc.base, tc.in.effective, tc.in.explicitEmpty, tc.in.effectiveMixed, s.modelGroupNamed)
			if err != nil {
				t.Fatalf("the §3.4.2.3.3 clause 4.2 merge failed: %v", err)
			}
			got, ok, err := s.recoverExtensionStepContent(xsderr.Loc{}, merged, tc.base)
			if err != nil {
				t.Fatalf("recovering the ·effective content· failed: %v", err)
			}
			if !ok {
				t.Fatalf("the inverse DECLINED a shape the merge built, so clause 1.5 would give up on an ordinary chain")
			}
			if got.explicitEmpty != tc.in.explicitEmpty || got.effectiveMixed != tc.in.effectiveMixed {
				t.Fatalf("recovered (explicitEmpty=%t, effectiveMixed=%t), want (%t, %t)",
					got.explicitEmpty, got.effectiveMixed, tc.in.explicitEmpty, tc.in.effectiveMixed)
			}
			if (got.effective == nil) != (tc.in.effective == nil) {
				t.Fatalf("recovered ·effective content· presence = %t, want %t", got.effective != nil, tc.in.effective != nil)
			}
			if got.effective != nil && !s.particlesIdentical(*got.effective, *tc.in.effective) {
				t.Fatalf("recovered a ·effective content· particle that is not property-identical to the one merged in")
			}
		})
	}
}

// ptr is the address of a freshly built particle, for the table rows above; the
// recoveredContent field is a pointer because ·absent· is a real value there.
func ptr(p Particle) *Particle { return &p }

// TestExtensionContentTypeInverseDeclines pins the decline the whole recovery
// rests on: a shape the merge never builds is answered "not recoverable" rather
// than guessed at, so clause 1.5 gives up (fail-open) instead of synthesizing an
// intermediate that corresponds to no derivation. Here the extension step's
// {content type} is empty where its base's is element-only, which no clause 4.2
// row produces.
func TestExtensionContentTypeInverseDeclines(t *testing.T) {
	s := xSchema(t, func(*SchemaBuilder) {})
	base := ElementContent{Particle: eSeqParticle(t, uq("b1"))}
	if _, ok, err := s.recoverExtensionStepContent(xsderr.Loc{}, EmptyContent{}, base); err != nil || ok {
		t.Fatalf("recovering an unbuildable shape returned (ok=%t, err=%v), want (false, nil)", ok, err)
	}
}

// TestExtensionContentTypeIsTheProducersMerge pins that the exported entry point
// dispatches on the base COMPONENT the way the producer needs it to: a simple
// base takes clause 4.2.1 whatever the effective content is, and a complex base
// is decided by its {content type}. The producer's own content-type tests cover
// the source-level behaviour; this one guards the relocation (#392) at the
// package boundary.
func TestExtensionContentTypeIsTheProducersMerge(t *testing.T) {
	s := xSchema(t, func(*SchemaBuilder) {})
	own := eSeqParticle(t, uq("e1"))
	simple, err := ExtensionContentType(xsderr.Loc{}, AnyAtomicType(), &own, false, true, s.modelGroupNamed)
	if err != nil {
		t.Fatalf("ExtensionContentType over a simple base: %v", err)
	}
	ec, ok := simple.(ElementContent)
	if !ok || !ec.Mixed || !s.particlesIdentical(ec.Particle, own) {
		t.Fatalf("clause 4.2.1 over a simple base gave %#v, want the ·effective content· as mixed element content", simple)
	}
	baseCT := uCT(t, uq("mergeBase"), eSeqParticle(t, uq("b1")))
	merged, err := ExtensionContentType(xsderr.Loc{}, baseCT, &own, false, false, s.modelGroupNamed)
	if err != nil {
		t.Fatalf("ExtensionContentType over a complex base: %v", err)
	}
	mc, ok := merged.(ElementContent)
	if !ok {
		t.Fatalf("clause 4.2.3 gave %#v, want element content", merged)
	}
	if !s.extensionPrependsToSequence(mc.Particle, eSeqParticle(t, uq("b1"))) {
		t.Fatalf("clause 4.2.3.3 did not build the 1..1 sequence cos-particle-extend clause 2 recognises")
	}
}
