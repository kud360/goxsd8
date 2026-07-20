package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleAuPropsCorrect is Attribute Use Correct (Structures §3.5.6,
// id="au-props-correct"): an attribute use's properties must match the §3.5.1
// property tableau. This file enforces only clause 1's cheap structural part —
// the {attribute declaration} slot is present (a non-nil sum variant, never a
// nil interface):
//
//   - clause 1 (tableau shape): {attribute declaration} is Required, so the
//     AttributeDeclarationOrRef must be present.
//
// Clauses 2 and 3 (the {value constraint} default→fixed consistency and the
// §3.5.4 key-evc effective-value-constraint agreement with the declaration) are
// VACUOUS at this skeleton stage: the AttributeUse component carries no {value
// constraint} slot yet (deferred to #70), so there is nothing to check. They
// are deferred to the value-constraint layer (#70) and finalize (#173, which
// resolves the declaration a ref points at); NOT enforced here.
const ruleAuPropsCorrect xsderr.Rule = "au-props-correct"

// AttributeDeclarationOrRef is the {attribute declaration} slot of an Attribute
// Use (Structures §3.5.1): the Required Attribute Declaration a use references.
// It is a sealed sum (STYLE T2/T7, the PRINCIPLES 7 sealed-sum exception):
// LocalAttributeDeclaration and AttributeDeclarationRef are its only
// implementations, sealed by the unexported attributeDeclarationRef method, so
// consumers exhaustively switch the two branches and no third variant is
// representable. It mirrors simpletype.go's Variety sealed sum (exported
// variants, exported fields, unexported marker method).
//
// Two variants exist because the two XML mappings that produce an Attribute Use
// differ fundamentally in OWNERSHIP of the referenced declaration:
//
//   - LocalAttributeDeclaration (the dcl.att.local mapping, §3.2.2.2): the
//     sibling local declaration is built in the same producer call and nothing
//     else in the component model holds it — it is in no global symbol table, so
//     this slot is its SOLE owner. It is therefore carried BY VALUE. A bare
//     QName here would orphan the declaration (lose a component), and a QName is
//     not even a unique key for local declarations (many locals can share an
//     expanded name), so by-value ownership is the only faithful shape (STYLE
//     T1).
//   - AttributeDeclarationRef (the ref.att.local mapping, §3.2.2.3): the use
//     points at a top-level declaration that may be forward-referenced, so only
//     a deferred QName is available at parse time. It is resolved to the live
//     component at finalize (#173).
type AttributeDeclarationOrRef interface{ attributeDeclarationRef() }

// LocalAttributeDeclaration is the {attribute declaration} variant for the
// dcl.att.local mapping (§3.2.2.2): the Attribute Use is the SOLE owner of a
// sibling local Attribute Declaration, carried by value. The field is read-only
// by convention; do not mutate it after construction.
type LocalAttributeDeclaration struct{ Declaration AttributeDeclaration }

// AttributeDeclarationRef is the {attribute declaration} variant for the
// ref.att.local mapping (§3.2.2.3): a pre-resolution QName reference — the
// attribute/@ref name — to a possibly-forward-referenced top-level Attribute
// Declaration, resolved to the live component at finalize (#173). The field is
// read-only by convention; do not mutate it after construction.
type AttributeDeclarationRef struct{ Name QName }

func (LocalAttributeDeclaration) attributeDeclarationRef() {}
func (AttributeDeclarationRef) attributeDeclarationRef()   {}

// AttributeUse is the Attribute Use component (Structures §3.5.1, id="au"): a
// kind of Annotated Component with {required}, {attribute declaration}, {value
// constraint} (Optional), {inheritable}, and {annotations}.
//
// This is the shape-only slice. The {value constraint} property (§3.5.1 vc_au)
// is deliberately NOT modeled yet — it is deferred to the value-constraint layer
// (#70), along with au-props-correct clauses 2 and 3 (the §3.5.4 key-evc
// effective-value-constraint consistency with the {attribute declaration}) that
// depend on it.
//
// {required} is the derived boolean fact (true for a "required" use= token,
// §3.5.1). The §3.2.2 use= XML token itself (AttributeUseToken) is a parse-time
// INPUT, not a component property, and is deliberately NOT stored here alongside
// {required} — carrying both would be redundant state (STYLE D3).
//
// Ratchet impact: unchanged. This is a leaf shape with no parser producer; the
// schema conformance lane moves only when the producer (#174/#175) wires it in.
//
// Construct only through NewAttributeUse, which rejects the states
// au-props-correct (§3.5.6) clause 1 forbids so they are unrepresentable
// (STYLE T1). AttributeUse is immutable after construction.
type AttributeUse struct {
	required             bool
	attributeDeclaration AttributeDeclarationOrRef
	inheritable          bool
	annotations          []Annotation
}

// NewAttributeUse builds an AttributeUse, rejecting the state Attribute Use
// Correct (§3.5.6, au-props-correct) clause 1 forbids: an absent {attribute
// declaration} (a nil AttributeDeclarationOrRef). The property is Required, so a
// nil interface is illegal (STYLE T1).
//
// Clauses 2 and 3 (value-constraint consistency and §3.5.4 key-evc agreement)
// are vacuous at this skeleton stage — the component carries no {value
// constraint} slot yet (deferred to #70) — so they are not checked here.
//
// annotations is copied; the caller's backing array is not aliased, and an
// empty input is held as nil.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built use — may
// legitimately pass the zero xsderr.Loc{}.
func NewAttributeUse(loc xsderr.Loc, required bool, attributeDeclaration AttributeDeclarationOrRef, inheritable bool, annotations []Annotation) (AttributeUse, error) {
	if attributeDeclaration == nil {
		return AttributeUse{}, xsderr.New(ruleAuPropsCorrect, loc,
			"attribute use has an absent {attribute declaration}, but it is Required (au-props-correct clause 1)")
	}
	u := AttributeUse{
		required:             required,
		attributeDeclaration: attributeDeclaration,
		inheritable:          inheritable,
	}
	if len(annotations) > 0 {
		u.annotations = append([]Annotation(nil), annotations...)
	}
	return u, nil
}

// Required returns the {required} property (§3.5.1): whether the attribute must
// appear on a validated element. It is the derived boolean (true for a
// "required" use= token), never the AttributeUseToken itself (STYLE D3).
func (u AttributeUse) Required() bool {
	return u.required
}

// AttributeDeclaration returns the {attribute declaration} property (Required):
// the sealed sum identifying either a sibling local declaration
// (LocalAttributeDeclaration) or a pre-resolution reference to a top-level one
// (AttributeDeclarationRef). It is never nil on a value built through
// NewAttributeUse.
func (u AttributeUse) AttributeDeclaration() AttributeDeclarationOrRef {
	return u.attributeDeclaration
}

// Inheritable returns the {inheritable} property (Required).
func (u AttributeUse) Inheritable() bool {
	return u.inheritable
}

// Annotations returns the {annotations} property in document order. It returns
// a copy: mutating the result does not affect u. An empty {annotations} yields
// nil.
func (u AttributeUse) Annotations() []Annotation {
	if len(u.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), u.annotations...)
}
