package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleMgdPropsCorrect is Model Group Definition Properties Correct (Structures
// §3.7.6, id="mgd-props-correct"): a model group definition's properties must
// match the §3.7.1 property tableau. This file enforces the two cheap,
// cross-reference-free parts of that tableau: {name} is a Required xs:NCName, so
// a QName with an empty local part is rejected; and {model group} is Required,
// so a zero (never-constructed) ModelGroup — whose {compositor} is the invalid
// zero — is rejected too, mirroring NewWildcard's zero-NamespaceConstraint guard.
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
// Construct only through NewModelGroupDefinition, which rejects an absent {name}
// and an absent (zero) {model group} so both Required-property violations are
// unrepresentable (STYLE T1). ModelGroupDefinition is immutable after
// construction.
type ModelGroupDefinition struct {
	loc         xsderr.Loc // source position; provenance, not a §3.7.1 property
	name        QName
	modelGroup  ModelGroup
	annotations []Annotation
}

// NewModelGroupDefinition builds a ModelGroupDefinition, rejecting the two
// states Model Group Definition Properties Correct (§3.7.6, mgd-props-correct)
// forbids at this layer:
//
//   - an absent {name}: its local part may not be empty. The §3.7.1 tableau
//     types {name} as a Required xs:NCName, and NCName's value space (Datatypes
//     §3.4.7, pattern \i\c*) excludes the empty string, so a zero-Local QName is
//     categorically not a legal {name}. The §5.3 Missing Sub-components escape
//     hatch does not cover it: §5.3 is scoped to properties whose value is
//     another component reached by QName ·resolution·, and {name} is the
//     identity other components resolve AGAINST. The guard is unconditional
//     because a model group definition has NO anonymous form: per §3.7.2 only a
//     <group> child of <schema>/<redefine> — always carrying name — maps to this
//     component, while a <group ref> maps to a Particle instead. That reasoning
//     is deliberately NOT generalized to NewComplexType / NewSimpleType, whose
//     components have a genuine anonymous form ({name} Optional). Testing the
//     local part, not name == QName{}, is deliberate: the latter would admit
//     QName{Space: "urn:x", Local: ""} as a named definition. Same idiom as
//     NewElementDeclaration's e-props-correct clause 1 check.
//   - an absent {model group}: the property is Required (§3.7.1), so a zero
//     ModelGroup — one never built through NewModelGroup, carrying the invalid
//     zero {compositor} — is illegal, mirroring NewWildcard's rejection of a
//     zero NamespaceConstraint.
//
// annotations is copied; the caller's backing array is not aliased, and an empty
// input is held as nil.
//
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the definition's provenance. Pass the position of this
// definition's own declaring element, never a convenient nearby one (a parent
// element's, say) — it is observable, not merely an error-charging convenience.
// A caller with no real parser position — a synthesized or programmatically
// built definition — passes the zero xsderr.Loc{}, which reads as "unknown".
func NewModelGroupDefinition(loc xsderr.Loc, name QName, modelGroup ModelGroup, annotations []Annotation) (ModelGroupDefinition, error) {
	if name.Local == "" {
		return ModelGroupDefinition{}, xsderr.New(ruleMgdPropsCorrect, loc,
			"model group definition has an absent {name}, but the §3.7.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (mgd-props-correct)")
	}
	switch modelGroup.Compositor() {
	case CompositorAll, CompositorChoice, CompositorSequence:
	default:
		return ModelGroupDefinition{}, xsderr.New(ruleMgdPropsCorrect, loc,
			"model group definition has an absent {model group} (a zero ModelGroup not built through NewModelGroup), but it is Required (mgd-props-correct)")
	}
	d := ModelGroupDefinition{loc: loc, name: name, modelGroup: modelGroup}
	if len(annotations) > 0 {
		d.annotations = append([]Annotation(nil), annotations...)
	}
	return d, nil
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
func (d ModelGroupDefinition) Name() QName {
	return d.name
}

// Loc reports the source position of the declaring element — provenance, not a
// §3.7.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown.
func (d ModelGroupDefinition) Loc() xsderr.Loc {
	return d.loc
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
