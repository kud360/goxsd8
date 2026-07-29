package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleNotationCorrect is Notation Declaration Correct (Structures §3.14.6,
// id="n-props-correct"): a notation declaration's properties must match the
// §3.14.1 tableau. §3.14.3 (Constraints on XML Representations) and §3.14.4
// (Validation Rules) are both "None as such," so the tableau's requirement
// that {system identifier} be present when {public identifier} is absent (and
// vice versa) — i.e. at least one of the two must be present — is enforced
// through this generic constraint. There is no dedicated src-notation-* id.
const ruleNotationCorrect xsderr.Rule = "n-props-correct"

// Notation is the Notation Declaration component (Structures §3.14.1): {name}
// (bundled with {target namespace} as an xsd.QName, per this package's "Names
// are expanded QNames" convention — doc.go), {system identifier} (an anyURI,
// optional), {public identifier} (a publicID, optional), and {annotations}.
// At least one of {system identifier}/{public identifier} must be present
// (§3.14.1 tableau, enforced via the generic Notation Declaration Correct
// constraint, §3.14.6 — §3.14.3/§3.14.4 define no dedicated SCC or validation
// rule).
//
// Construct only through NewNotation, which rejects the state the tableau
// forbids (both identifiers absent) so it is unrepresentable (STYLE T1).
// Notation is immutable after construction.
type Notation struct {
	loc         xsderr.Loc // source position; provenance, not a §3.14.1 property
	name        QName
	systemID    string
	hasSystem   bool
	publicID    string
	hasPublic   bool
	annotations []Annotation
}

// NewNotation builds a Notation, rejecting the state Notation Declaration
// Correct (§3.14.6, n-props-correct) forbids: both {system identifier} and
// {public identifier} absent (nil). Either identifier, or both, may be
// present; an empty string is a legal (present) anyURI/publicID value, which
// is why presence is signalled by a non-nil pointer rather than a "" sentinel.
// annotations is copied; the caller's slice is not aliased.
//
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the declaration's provenance. Pass the position of this
// declaration's own declaring element, never a convenient nearby one (a parent
// element's, say) — it is observable, not merely an error-charging convenience.
// A caller with no real parser position — a synthesized or programmatically
// built declaration — passes the zero xsderr.Loc{}, which reads as "unknown".
func NewNotation(loc xsderr.Loc, name QName, systemID, publicID *string, annotations []Annotation) (Notation, error) {
	if systemID == nil && publicID == nil {
		return Notation{}, xsderr.New(ruleNotationCorrect, loc,
			"notation declaration must have a {system identifier} or a {public identifier}, or both")
	}
	n := Notation{loc: loc, name: name}
	if systemID != nil {
		n.systemID, n.hasSystem = *systemID, true
	}
	if publicID != nil {
		n.publicID, n.hasPublic = *publicID, true
	}
	if len(annotations) > 0 {
		n.annotations = append([]Annotation(nil), annotations...)
	}
	return n, nil
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
func (n Notation) Name() QName {
	return n.name
}

// Loc reports the source position of the declaring element — provenance, not a
// §3.14.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown.
func (n Notation) Loc() xsderr.Loc {
	return n.loc
}

// SystemIdentifier returns the {system identifier} property (an anyURI); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
func (n Notation) SystemIdentifier() (string, bool) {
	return n.systemID, n.hasSystem
}

// PublicIdentifier returns the {public identifier} property (a publicID); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
func (n Notation) PublicIdentifier() (string, bool) {
	return n.publicID, n.hasPublic
}

// Annotations returns the {annotations} property in document order. It returns
// a copy: mutating the result does not affect n. An empty {annotations}
// yields nil.
func (n Notation) Annotations() []Annotation {
	if len(n.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), n.annotations...)
}
