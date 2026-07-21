package xsd

import "github.com/kud360/goxsd8/xsderr"

// Particle is the Particle component (Structures §3.9.1, id="p"): {min occurs},
// {max occurs}, {term}, and {annotations}. The {min occurs}/{max occurs} pair is
// composed directly as an xsd.Occurs (occurs.go), which already enforces Particle
// Correct (§3.9.6.1, p-props-correct) clause 2.1 (min ≤ a numeric max) — so this
// file does not restate the occurrence-range invariants, one fact in one
// encoding (STYLE D3).
//
// {term} is a TermOrRef (term.go), NOT a resolved Term: it is either an inline
// component (ResolvedTerm) or a deferred <element ref>/<group ref> QName
// (ElementDeclarationRef/ModelGroupRef). Finalize (#173) VALIDATES that each
// such ref resolves against the schema indexes — rejecting an unresolvable or
// circular target with src-resolve (or a named-circularity rule) — but it does
// not rewrite the slot: Term() keeps returning the ref, and a consumer follows
// it by a read-time lookup through the schema (e.g. schema.Element(ref.Name)).
//
// A max occurs of 0 is representable through Occurs{0,0} and NOT rejected here:
// occurs.go (#29) documents {0,0} as a legal vacuous range and enforces
// p-props-correct clause 2.1 for it. The §3.7.2/§3.8.2/§3.9.2 XML-mapping rules
// that say minOccurs=maxOccurs=0 "maps to no component at all" are a PRODUCER
// normalization (a min=max=0 source item yields no Particle, never a Particle
// with {max occurs} 0) enforced upstream in the producer (#176), not an
// invariant of the abstract Particle component; rejecting Occurs{0,0} here would
// contradict landed occurs.go and the plain-nonNegativeInteger §3.9.1 tableau.
//
// Ratchet impact: unchanged. This is a leaf shape with no parser producer; the
// schema conformance lane moves only when the producer (#176) wires it in.
//
// Construct only through NewParticle, which rejects an absent {term} so the
// Required-property violation is unrepresentable (STYLE T1). Particle is
// immutable after construction.
type Particle struct {
	occurs      Occurs
	term        TermOrRef
	annotations []Annotation
}

// NewParticle builds a Particle, rejecting the state Particle Correct (§3.9.6.1,
// p-props-correct) clause 1 forbids at this layer: an absent {term}. That
// covers two representable absences — a nil TermOrRef interface, and a
// ResolvedTerm wrapping a nil Term (the one sealed-sum variant that wraps an
// interface, so ResolvedTerm{Term: nil} slips past the outer nil check yet is an
// equally absent {term}). The property is Required, so either is illegal (STYLE
// T1), mirroring NewAttributeUse's clause-1 nil check.
//
// The occurrence-range invariants (p-props-correct clauses 1 and 2.1) are
// already enforced by the Occurs constructors, so occurs is trusted here; a
// vacuous Occurs{0,0} is accepted (see the type doc comment).
//
// annotations is copied; the caller's backing array is not aliased, and an empty
// input is held as nil.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built particle — may
// legitimately pass the zero xsderr.Loc{}.
func NewParticle(loc xsderr.Loc, occurs Occurs, term TermOrRef, annotations []Annotation) (Particle, error) {
	if term == nil {
		return Particle{}, xsderr.New(ruleParticleCorrect, loc,
			"particle has an absent {term}, but it is Required (p-props-correct clause 1)")
	}
	if rt, ok := term.(ResolvedTerm); ok && rt.Term == nil {
		return Particle{}, xsderr.New(ruleParticleCorrect, loc,
			"particle {term} is a ResolvedTerm wrapping a nil Term, but {term} is Required (p-props-correct clause 1)")
	}
	p := Particle{occurs: occurs, term: term}
	if len(annotations) > 0 {
		p.annotations = append([]Annotation(nil), annotations...)
	}
	return p, nil
}

// Occurs returns the {min occurs}/{max occurs} pair (§3.9.1) as an Occurs.
func (p Particle) Occurs() Occurs {
	return p.occurs
}

// Term returns the {term} property (Required): the TermOrRef identifying either
// an inline Term (ResolvedTerm) or a pre-resolution <element ref>/<group ref>
// reference (ElementDeclarationRef/ModelGroupRef). It is never nil on a value
// built through NewParticle.
func (p Particle) Term() TermOrRef {
	return p.term
}

// Annotations returns the {annotations} property in document order. It returns a
// copy: mutating the result does not affect p. An empty {annotations} yields
// nil.
func (p Particle) Annotations() []Annotation {
	if len(p.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), p.annotations...)
}
