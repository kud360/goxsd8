package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal: the effective-total-range machinery is
// unexported (STYLE T5) and is reached only through finalize, so the assertions
// are made directly on the helpers. The component builders come from
// particleattribution_test.go — one set of helpers, not two (STYLE T4).

// rSchema finalizes a schema carrying only the named types and model group
// definitions build adds, so that effectiveTotalRange has a *Schema to resolve
// <group ref>s against.
func rSchema(t *testing.T, build func(*SchemaBuilder)) *Schema {
	t.Helper()
	b := NewSchemaBuilder()
	b.AddType(uNamedType(t, uq("T")))
	if build != nil {
		build(b)
	}
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// rElem is a particle over a local element declaration with the given occurrence
// bounds — one of the "wildcard or element declaration particles" both
// cos-seq-range and cos-choice-range aggregate by their OWN {min occurs}/{max
// occurs}.
func rElem(t *testing.T, minOccurs, maxOccurs int) Particle {
	t.Helper()
	return rNamedElem(t, "e", minOccurs, maxOccurs)
}

// rNamedElem is rElem with an explicit declaration name, for the model groups
// that need two DISTINCT members (two same-named element particles in one group
// would trip cos-nonambig before any range is computed).
func rNamedElem(t *testing.T, local string, minOccurs, maxOccurs int) Particle {
	t.Helper()
	return uParticle(t, uOccurs(t, minOccurs, maxOccurs), ResolvedTerm{Term: uLocal(t, uq(local), uq("T"))})
}

// TestEffectiveTotalRangeSumVersusMinOf is the trap cos-seq-range and
// cos-choice-range set: the two constraints read almost identically but the
// sequence/all minimum is a SUM across {particles} while the choice minimum is a
// MINIMUM-OF, and the maxima are correspondingly a sum and a maximum-of. An
// implementation that transposes them fails every row here.
func TestEffectiveTotalRangeSumVersusMinOf(t *testing.T) {
	s := rSchema(t, nil)
	for _, tc := range []struct {
		name       string
		compositor Compositor
		members    []Particle
		want       effectiveRange
	}{
		{"sequence sums both bounds", CompositorSequence,
			[]Particle{rElem(t, 1, 1), rElem(t, 1, 1)}, effectiveRange{min: 2, max: 2}},
		{"choice takes the minimum and the maximum", CompositorChoice,
			[]Particle{rElem(t, 1, 1), rElem(t, 1, 1)}, effectiveRange{min: 1, max: 1}},
		{"sequence of 0..1 and 2..3", CompositorSequence,
			[]Particle{rElem(t, 0, 1), rElem(t, 2, 3)}, effectiveRange{min: 2, max: 4}},
		{"choice of 0..1 and 2..3", CompositorChoice,
			[]Particle{rElem(t, 0, 1), rElem(t, 2, 3)}, effectiveRange{min: 0, max: 3}},
		{"all sums like sequence", CompositorAll,
			[]Particle{rElem(t, 1, 1), rElem(t, 1, 1)}, effectiveRange{min: 2, max: 2}},
		{"empty sequence is 0..0", CompositorSequence, nil, effectiveRange{min: 0, max: 0}},
		{"empty choice is 0..0", CompositorChoice, nil, effectiveRange{min: 0, max: 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := uOne(t, ResolvedTerm{Term: uGroup(t, tc.compositor, tc.members...)})
			if got := s.effectiveTotalRange(p); got != tc.want {
				t.Fatalf("effectiveTotalRange = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestEffectiveTotalRangeParentFactor pins that the enclosing particle's bounds
// multiply the aggregate, per both constraints' "the product of P.{min occurs}
// and ...".
func TestEffectiveTotalRangeParentFactor(t *testing.T) {
	s := rSchema(t, nil)
	g := uGroup(t, CompositorSequence, rElem(t, 2, 3))
	p := uParticle(t, uOccurs(t, 4, 5), ResolvedTerm{Term: g})
	if got := (effectiveRange{min: 8, max: 15}); s.effectiveTotalRange(p) != got {
		t.Fatalf("effectiveTotalRange = %+v, want %+v", s.effectiveTotalRange(p), got)
	}
}

// TestEffectiveTotalRangeUnbounded covers the maximum rule's three outcomes,
// including the one warden singled out: an unbounded PARENT over a body whose
// every member maximum is 0 yields 0, NOT unbounded ("if any of those is
// non-zero and P.{max occurs} = unbounded").
func TestEffectiveTotalRangeUnbounded(t *testing.T) {
	s := rSchema(t, nil)
	for _, tc := range []struct {
		name   string
		parent Occurs
		member Particle
		want   effectiveRange
	}{
		{"unbounded member propagates", uOccurs(t, 1, 1), uParticle(t, uUnbounded(t, 1),
			ResolvedTerm{Term: uLocal(t, uq("e"), uq("T"))}), effectiveRange{min: 1, max: unboundedMax}},
		{"unbounded parent over a non-zero body", uUnbounded(t, 1), rElem(t, 1, 1),
			effectiveRange{min: 1, max: unboundedMax}},
		{"unbounded parent over an all-zero body is 0, not unbounded", uUnbounded(t, 1), rElem(t, 0, 0),
			effectiveRange{min: 0, max: 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := uParticle(t, tc.parent, ResolvedTerm{Term: uGroup(t, CompositorSequence, tc.member)})
			if got := s.effectiveTotalRange(p); got != tc.want {
				t.Fatalf("effectiveTotalRange = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestEffectiveTotalRangeThroughGroupRef pins that a <group ref> {term} is
// followed into the referenced definition's {model group} (§3.7.2), so a
// referenced group contributes its own effective total range rather than being
// treated as a leaf.
func TestEffectiveTotalRangeThroughGroupRef(t *testing.T) {
	s := rSchema(t, func(b *SchemaBuilder) {
		inner := uGroup(t, CompositorSequence, rElem(t, 1, 1), rElem(t, 1, 1))
		mgd, err := NewModelGroupDefinition(xsderr.Loc{}, uq("g"), inner, nil)
		if err != nil {
			t.Fatalf("NewModelGroupDefinition: %v", err)
		}
		b.AddModelGroup(mgd)
	})
	outer := uGroup(t, CompositorSequence, uOne(t, ModelGroupRef{Name: uq("g")}), rElem(t, 1, 1))
	got := s.effectiveTotalRange(uOne(t, ResolvedTerm{Term: outer}))
	if want := (effectiveRange{min: 3, max: 3}); got != want {
		t.Fatalf("effectiveTotalRange through <group ref> = %+v, want %+v", got, want)
	}
}

// TestEffectiveTotalRangeSaturates pins that a range large enough to overflow
// int saturates rather than wrapping. Wrapping is the one failure that matters:
// a huge positive minimum turning into 0 would make a non-emptiable particle
// look ·emptiable· and ACCEPT a restriction the spec rejects.
func TestEffectiveTotalRangeSaturates(t *testing.T) {
	s := rSchema(t, nil)
	members := make([]Particle, 0, 20)
	for range 20 {
		members = append(members, rElem(t, 100000, 100000))
	}
	inner := uGroup(t, CompositorSequence, members...)
	p := uParticle(t, uOccurs(t, 100000, 100000), ResolvedTerm{Term: inner})
	got := s.effectiveTotalRange(p)
	if got.min != maxFiniteRange || got.max != maxFiniteRange {
		t.Fatalf("effectiveTotalRange = %+v, want both bounds saturated at %d", got, maxFiniteRange)
	}
}

// TestParticleEmptiable covers both cos-group-emptiable clauses and the two
// negatives.
func TestParticleEmptiable(t *testing.T) {
	s := rSchema(t, nil)
	elem := ResolvedTerm{Term: uLocal(t, uq("e"), uq("T"))}
	for _, tc := range []struct {
		name string
		p    Particle
		want bool
	}{
		{"clause 1: {min occurs} = 0", uParticle(t, uOccurs(t, 0, 1), elem), true},
		{"a 1..1 element particle is not emptiable", uOne(t, elem), false},
		{"clause 2: sequence whose members are all optional",
			uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence, rElem(t, 0, 1))}), true},
		{"clause 2: sequence with a mandatory member is not emptiable",
			uOne(t, ResolvedTerm{Term: uGroup(t, CompositorSequence, rElem(t, 1, 1))}), false},
		{"clause 2: choice with one optional branch is emptiable",
			uOne(t, ResolvedTerm{Term: uGroup(t, CompositorChoice, rElem(t, 1, 1), rElem(t, 0, 1))}), true},
		{"clause 2: choice with only mandatory branches is not emptiable",
			uOne(t, ResolvedTerm{Term: uGroup(t, CompositorChoice, rElem(t, 1, 1), rElem(t, 2, 2))}), false},
		{"clause 2: an EMPTY choice is emptiable — the divergence from UPA's own notion",
			uOne(t, ResolvedTerm{Term: uGroup(t, CompositorChoice)}), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.particleEmptiable(tc.p); got != tc.want {
				t.Fatalf("particleEmptiable = %t, want %t", got, tc.want)
			}
		})
	}
}
