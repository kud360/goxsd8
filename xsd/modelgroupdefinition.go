package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleMgdPropsCorrect is Model Group Definition Properties Correct (Structures
// §3.7.6, id="mgd-props-correct"): a model group definition's properties must
// match the §3.7.1 property tableau. This file enforces the one cheap,
// cross-reference-free part of that tableau: {model group} is Required, so a
// zero (never-constructed) ModelGroup — whose {compositor} is the invalid zero —
// is rejected, mirroring NewWildcard's zero-NamespaceConstraint guard.
const ruleMgdPropsCorrect xsderr.Rule = "mgd-props-correct"

// ModelGroupDefinition is the Model Group Definition component (Structures
// §3.7.1, id="mgd"): a kind of Annotated Component with {annotations}, {name}
// (bundled with {target namespace} as an xsd.QName per this package's "Names are
// expanded QNames" convention — doc.go), and {model group} (a Required Model
// Group component).
//
// It is NOT a Term: only its {model group} is (see term.go). A <group ref>
// resolves to that shared {model group}, never to the definition itself (§3.7.2).
// The definition carries no occurrence range: {min occurs}/{max occurs} live
// solely on the particles that refer to it (§3.7.2 note), so there is nothing to
// store here for occurrence.
//
// Ratchet impact: unchanged. This is a leaf shape with no parser producer; the
// schema conformance lane moves only when the producer (#176) wires it in.
//
// Construct only through NewModelGroupDefinition, which rejects an absent (zero)
// {model group} so the Required-property violation is unrepresentable (STYLE T1).
// ModelGroupDefinition is immutable after construction.
type ModelGroupDefinition struct {
	name        QName
	modelGroup  ModelGroup
	annotations []Annotation
}

// NewModelGroupDefinition builds a ModelGroupDefinition, rejecting the state
// Model Group Definition Properties Correct (§3.7.6, mgd-props-correct) forbids
// at this layer: an absent {model group}. The property is Required (§3.7.1), so a
// zero ModelGroup — one never built through NewModelGroup, carrying the invalid
// zero {compositor} — is illegal, mirroring NewWildcard's rejection of a zero
// NamespaceConstraint.
//
// annotations is copied; the caller's backing array is not aliased, and an empty
// input is held as nil.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built definition — may
// legitimately pass the zero xsderr.Loc{}.
func NewModelGroupDefinition(loc xsderr.Loc, name QName, modelGroup ModelGroup, annotations []Annotation) (ModelGroupDefinition, error) {
	switch modelGroup.Compositor() {
	case CompositorAll, CompositorChoice, CompositorSequence:
	default:
		return ModelGroupDefinition{}, xsderr.New(ruleMgdPropsCorrect, loc,
			"model group definition has an absent {model group} (a zero ModelGroup not built through NewModelGroup), but it is Required (mgd-props-correct)")
	}
	d := ModelGroupDefinition{name: name, modelGroup: modelGroup}
	if len(annotations) > 0 {
		d.annotations = append([]Annotation(nil), annotations...)
	}
	return d, nil
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
func (d ModelGroupDefinition) Name() QName {
	return d.name
}

// ModelGroup returns the {model group} property (Required): the Model Group a
// <group ref> to this definition resolves its particle's {term} to (§3.7.2).
func (d ModelGroupDefinition) ModelGroup() ModelGroup {
	return d.modelGroup
}

// Annotations returns the {annotations} property in document order. It returns a
// copy: mutating the result does not affect d. An empty {annotations} yields
// nil.
func (d ModelGroupDefinition) Annotations() []Annotation {
	if len(d.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), d.annotations...)
}
