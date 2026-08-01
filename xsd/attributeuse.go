package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleAuPropsCorrect is Attribute Use Correct (Structures §3.5.6,
// id="au-props-correct"): an attribute use's properties must match the §3.5.1
// property tableau. This file enforces the clauses that are cheap, structural,
// and cross-reference-free at this layer:
//
//   - clause 1 (tableau shape): {attribute declaration} is Required, so the
//     AttributeDeclarationOrRef must be present.
//   - clause 3 (variety half, Local case): if U.{attribute declaration} has
//     {value constraint}.{variety} = fixed and U itself has a {value
//     constraint}, then U.{value constraint}.{variety} must be fixed. Only the
//     variety-agreement half is enforced here, and only for the
//     LocalAttributeDeclaration variant, whose ValueConstraint() is available by
//     value at construction; the AttributeDeclarationRef variant is still an
//     unresolved QName. BOTH halves for BOTH variants are decided at finalize
//     instead, by Phase E (valueconstraintvalid.go): the {value}-identity half
//     needs a lexical→value mapping, which xsd.ValueConstraint cannot carry (it
//     holds {lexical form} only, see valueconstraint.go) and this package cannot
//     compute (see ValueSpace). This constructor check is therefore a strictly
//     earlier, strictly narrower rejection of the same states, kept so a use the
//     Local mapping builds is illegal-by-construction (STYLE T1).
//
// Clause 2 (Simple Default Valid — §3.2.6.2 cos-valid-simple-default) needs the
// resolved {attribute declaration}.{type definition} to validate the {value
// constraint}'s {lexical form}; it is not enforced anywhere yet and stays
// deferred to its own issue. The §3.5.4 key-evc effective value constraint
// needs the resolved declaration for the Ref variant, so it is NOT modeled on
// this component: it lives at finalize as (*Schema).effectiveValueConstraint
// (defaultbinding.go, #262), unexported until an external consumer justifies
// exporting it (STYLE T5).
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
//
// Name is a PRESENT reference, never the absent (zero) QName: this variant
// exists only for the mapping branch whose precondition is "the ref attribute is
// present". NewAttributeUse rejects a zero Name rather than deferring it (see
// there for why).
type AttributeDeclarationRef struct{ Name QName }

func (LocalAttributeDeclaration) attributeDeclarationRef() {}
func (AttributeDeclarationRef) attributeDeclarationRef()   {}

// AttributeUse is the Attribute Use component (Structures §3.5.1, id="au"): a
// kind of Annotated Component with {required}, {attribute declaration}, {value
// constraint} (Optional), {inheritable}, and {annotations}.
//
// The {value constraint} property (§3.5.1 vc_au) is modeled as an
// INDEPENDENTLY-optional slot (mirroring attributedeclaration.go): under the
// local mapping dcl.att.local (§3.2.2.2) an attribute's default/fixed feeds the
// Use's {value constraint}, so absence must be representable independently of
// the sibling declaration. au-props-correct clause 3's variety-agreement half is
// enforced now for the Local case (see ruleAuPropsCorrect); its {value}-identity
// half, the Ref case, and clause 2 (Simple Default Valid, needs a resolved {type
// definition}) are not. The whole of clause 3 — both halves, both variants — is
// decided at finalize by Phase E (valueconstraintvalid.go); clause 2 stays
// deferred. The §3.5.4 key-evc effective
// value constraint is computed at finalize instead of here, for the same reason
// (defaultbinding.go, #262).
//
// {required} is the derived boolean fact (true for a "required" use= token,
// §3.5.1). The §3.2.2 use= XML token itself (AttributeUseToken) is a parse-time
// INPUT, not a component property, and is deliberately NOT stored here alongside
// {required} — carrying both would be redundant state (STYLE D3).
//
// Ratchet impact: the producer wires real value constraints in as of #235,
// moving the schema conformance lane (derivation-ok-restriction clause 3 via
// key-evc, defaultbinding.go).
//
// Construct only through NewAttributeUse, which rejects the states
// au-props-correct (§3.5.6) clauses 1 and 3 (variety half, Local case) forbid so
// they are unrepresentable (STYLE T1). AttributeUse is immutable after
// construction.
type AttributeUse struct {
	required             bool
	attributeDeclaration AttributeDeclarationOrRef
	valueConstraint      ValueConstraint
	hasValueConstraint   bool
	inheritable          bool
	annotations          []Annotation
}

// NewAttributeUse builds an AttributeUse, rejecting the states Attribute Use
// Correct (§3.5.6, au-props-correct) forbids:
//
//   - clause 1: an absent {attribute declaration} (a nil
//     AttributeDeclarationOrRef). The property is Required, so a nil interface
//     is illegal (STYLE T1).
//   - clause 3 (variety half, Local case): if attributeDeclaration is a
//     LocalAttributeDeclaration whose own {value constraint} has {variety} =
//     fixed and valueConstraint is non-nil, then valueConstraint's {variety}
//     must be fixed too. The {value}-identity half and the Ref case are decided
//     at finalize by Phase E, valueconstraintvalid.go (see ruleAuPropsCorrect).
//
// It also rejects an AttributeDeclarationRef whose Name is the absent (zero)
// QName, charged to xsderr.RuleComponentInvariant: the Ref variant represents
// only the ref-present mapping branch (§3.2.2.3), so an absent name is an
// illegal representation this constructor makes unbuildable, not a resolution
// failure to defer to finalize. See NewParticle for the full reasoning — it
// rejects the two analogous ref {term} variants on the same footing.
//
// valueConstraint is a pointer so absence (nil) is distinct from a present zero
// record (mirroring attributedeclaration.go's *ValueConstraint handling); when
// non-nil the pointed-to value is COPIED into the struct and hasValueConstraint
// is set — the pointer itself is never stored, so the caller's value is not
// aliased. annotations is copied; the caller's backing array is not aliased, and
// an empty input is held as nil.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built use — may
// legitimately pass the zero xsderr.Loc{}.
func NewAttributeUse(loc xsderr.Loc, required bool, attributeDeclaration AttributeDeclarationOrRef, valueConstraint *ValueConstraint, inheritable bool, annotations []Annotation) (AttributeUse, error) {
	if attributeDeclaration == nil {
		return AttributeUse{}, xsderr.New(ruleAuPropsCorrect, loc,
			"attribute use has an absent {attribute declaration}, but it is Required (au-props-correct clause 1)")
	}
	if ref, ok := attributeDeclaration.(AttributeDeclarationRef); ok && ref.Name.Local == "" {
		return AttributeUse{}, xsderr.New(xsderr.RuleComponentInvariant, loc,
			"attribute use {attribute declaration} is an AttributeDeclarationRef carrying the absent (zero) QName, but that variant maps only the ref-present branch (§3.2.2.3 ref.att.local), whose attribute/@ref is an xs:QName with a non-empty local part")
	}
	// Clause 3 variety half fires only for the Local variant, whose declaration
	// (and its {value constraint}) is owned by value and readable now; the Ref
	// variant is unresolved (deferred to #173), so its declaration's {value
	// constraint} is not yet available and the check is skipped.
	if local, ok := attributeDeclaration.(LocalAttributeDeclaration); ok && valueConstraint != nil {
		declVC, hasDeclVC := local.Declaration.ValueConstraint()
		if hasDeclVC && declVC.Kind() == ValueFixed && valueConstraint.Kind() != ValueFixed {
			return AttributeUse{}, xsderr.New(ruleAuPropsCorrect, loc,
				"attribute use {value constraint}.{variety} is %s, but its {attribute declaration} has {value constraint}.{variety} = fixed, so the use's must also be fixed (au-props-correct clause 3)", valueConstraint.Kind())
		}
	}
	u := AttributeUse{
		required:             required,
		attributeDeclaration: attributeDeclaration,
		inheritable:          inheritable,
	}
	if valueConstraint != nil {
		u.valueConstraint, u.hasValueConstraint = *valueConstraint, true
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
// NewAttributeUse, and an AttributeDeclarationRef's Name is never the absent
// (zero) QName.
func (u AttributeUse) AttributeDeclaration() AttributeDeclarationOrRef {
	return u.attributeDeclaration
}

// ValueConstraint returns the {value constraint} property (§3.5.1 vc_au,
// Optional); the second result is false when it is absent, in which case the
// first result is not meaningful.
//
// This is the Use's OWN {value constraint}, not the §3.5.4 ·effective value
// constraint· (key-evc), which falls back to the {attribute declaration}'s and
// needs the resolved declaration for the Ref variant; that one is computed at
// finalize by (*Schema).effectiveValueConstraint (defaultbinding.go).
func (u AttributeUse) ValueConstraint() (ValueConstraint, bool) {
	return u.valueConstraint, u.hasValueConstraint
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
