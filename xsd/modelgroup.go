package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleMgPropsCorrect is Model Group Correct (Structures §3.8.6.1,
// id="mg-props-correct"): a model group's properties must match the §3.8.1
// property tableau. This file enforces only clause 1's cheap structural part —
// {compositor} is one of {all, choice, sequence} (the zero Compositor is
// invalid). Clause 2 (no circular groups: no particle at any depth whose {term}
// is the group itself) needs the resolved <group ref> graph, which only exists
// once the schema set is assembled; per PRINCIPLES 5 it is NOT checked in this
// constructor (no seen-set traversal here) but at finalize (resolve.go's
// checkModelGroupsAcyclic, #173).
const ruleMgPropsCorrect xsderr.Rule = "mg-props-correct"

// ModelGroup is the Model Group component (Structures §3.8.1, id="mg"): a kind
// of Term with {annotations} (a sequence of Annotation), {compositor} (one of
// all/choice/sequence — the xsd.Compositor enum, closedsets.go), and {particles}
// (a sequence of Particle components).
//
// {particles} is spec-worded a SEQUENCE (§3.8.1 tableau), so document order is
// spec-significant, not merely a determinism convention: UPA (cos-nonambig,
// §3.8.6.4), particle restriction (§3.9.6.2/.3), and effective-total-range
// (§3.8.6.5/.6) all read the particles in order. It is held as a document-order
// slice, never de-duplicated or reordered.
//
// Ratchet impact: unchanged. This is a leaf shape with no parser producer; the
// schema conformance lane moves only when the producer (#176) wires it in.
//
// Construct only through NewModelGroup, which rejects an invalid {compositor} so
// an ill-formed record is unrepresentable (STYLE T1). ModelGroup is immutable
// after construction.
type ModelGroup struct {
	compositor  Compositor
	particles   []Particle
	annotations []Annotation
}

// NewModelGroup builds a ModelGroup, rejecting the state Model Group Correct
// (§3.8.6.1, mg-props-correct) clause 1 forbids: a {compositor} that is not one
// of CompositorAll/CompositorChoice/CompositorSequence (the invalid zero
// Compositor included), mirroring NewWildcard's {process contents} check.
//
// mg-props-correct clause 2 (no circular groups) is a finalize-phase concern
// (resolve.go, #173); it is deliberately NOT checked here — the constructor
// performs no traversal of nested particles.
//
// particles and annotations are copied; the caller's backing arrays are not
// aliased, and an empty input is held as nil.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built group — may
// legitimately pass the zero xsderr.Loc{}.
func NewModelGroup(loc xsderr.Loc, compositor Compositor, particles []Particle, annotations []Annotation) (ModelGroup, error) {
	switch compositor {
	case CompositorAll, CompositorChoice, CompositorSequence:
	default:
		return ModelGroup{}, xsderr.New(ruleMgPropsCorrect, loc,
			"model group {compositor} %s is not one of all/choice/sequence (mg-props-correct clause 1)", compositor)
	}
	g := ModelGroup{compositor: compositor}
	if len(particles) > 0 {
		g.particles = append([]Particle(nil), particles...)
	}
	if len(annotations) > 0 {
		g.annotations = append([]Annotation(nil), annotations...)
	}
	return g, nil
}

// term marks ModelGroup as a Term (§3.8.1: "a kind of Term"); see term.go.
func (ModelGroup) term() {}

// Compositor returns the {compositor} property (§3.8.1): one of all/choice/
// sequence.
func (g ModelGroup) Compositor() Compositor {
	return g.compositor
}

// Particles returns the {particles} property in document order. It returns a
// copy: mutating the result does not affect g. An empty {particles} yields nil.
// The order is spec-significant (§3.8.1 sequence), not merely a convention.
func (g ModelGroup) Particles() []Particle {
	if len(g.particles) == 0 {
		return nil
	}
	return append([]Particle(nil), g.particles...)
}

// Annotations returns the {annotations} property in document order. It returns a
// copy: mutating the result does not affect g. An empty {annotations} yields
// nil.
func (g ModelGroup) Annotations() []Annotation {
	if len(g.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), g.annotations...)
}
