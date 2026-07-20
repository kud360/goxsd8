package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleAPropsCorrect is Attribute Declaration Properties Correct (Structures
// §3.2.6.1, id="a-props-correct"): an attribute declaration's properties must
// match the §3.2.1 property tableau. This file enforces the clauses that are
// cheap, purely structural, and cross-reference-free at this layer, citing the
// specific clause number in each message (the rule ID is not sub-anchored per
// clause, matching elementdeclaration.go's single-rule-const convention):
//
//   - clause 1 (tableau shape): {scope}.{variety} is one of the legal Scope
//     tokens (global or local), and a present {value constraint} carries a
//     legal {variety} (default or fixed).
//
// Clause 2 (Simple Default Valid — §3.2.6.2 cos-valid-simple-default) is a
// cross-component / finalize-phase constraint: it needs the resolved {type
// definition} to validate the {value constraint}'s {lexical form} against it,
// which this package does not resolve yet. It is deferred to the finalize-phase
// issue (#173) that first introduces phased construction (per doc.go's
// "parse → resolve → finalize") and is NOT enforced here.
const ruleAPropsCorrect xsderr.Rule = "a-props-correct"

// AttributeDeclaration is the Attribute Declaration component (Structures
// §3.2.1, id="Attribute_Declaration_details"): a kind of Annotated Component
// with {name} (bundled with {target namespace} as an xsd.QName per this
// package's "Names are expanded QNames" convention — doc.go), {type
// definition}, {scope}, {value constraint} (Optional), {inheritable}, and
// {annotations}.
//
// Like the other §3 component shapes in this package, AttributeDeclaration is a
// STRUCTURAL holder built before resolution. Its {type definition} is carried
// as a pre-resolution QName REFERENCE, not a resolved component (the type/@type
// name of §3.2.2). Its resolved-component accessor, and a-props-correct clause 2
// (Simple Default Valid), are deferred to the finalize-phase issue (#173) that
// first introduces phased construction; this package resolves neither yet.
//
// {scope}.{parent} (§3.2.1 sc_a — a Complex Type Definition or Attribute Group
// Definition) is entirely UNMODELED by this issue. Only {scope}.{variety} is
// carried (as a ScopeVariety, shared with element declarations per closedsets.go
// — only the variety closed set is shared, the sc_a Scope record itself is
// not). A ScopeLocal attribute is therefore structurally incomplete: its
// containing Complex Type Definition or Attribute Group Definition does not
// exist as a resolved back-reference yet, so there is nothing to point {parent}
// at (a decl↔container back-reference deferred to finalize per STYLE 5). The gap
// is named here rather than buried; ScopeVariety() documents it too.
//
// The {value constraint} is deliberately an INDEPENDENTLY-optional slot: under
// the local mapping dcl.att.local (§3.2.2.2) a locally-declared attribute's own
// {value constraint} is always absent (any default/fixed feeds the sibling
// Attribute Use's {value constraint} instead), while the global mapping
// dcl.att.global (§3.2.2.1) populates it here — so absence must be
// representable on the declaration independently of the use.
//
// Ratchet impact: unchanged. This is a leaf shape with no parser producer; the
// schema conformance lane moves only when the producer (#174/#175) wires it in.
//
// Construct only through NewAttributeDeclaration, which rejects the states
// a-props-correct (§3.2.6.1) clause 1 forbids so they are unrepresentable
// (STYLE T1). AttributeDeclaration is immutable after construction.
type AttributeDeclaration struct {
	name               QName
	typeDefinitionName QName
	scopeVariety       ScopeVariety
	valueConstraint    ValueConstraint
	hasValueConstraint bool
	inheritable        bool
	annotations        []Annotation
}

// NewAttributeDeclaration builds an AttributeDeclaration, rejecting the states
// Attribute Declaration Properties Correct (§3.2.6.1, a-props-correct) clause 1
// forbids:
//
//   - scopeVariety must be a legal Scope token (ScopeGlobal or ScopeLocal);
//   - a present valueConstraint must carry a legal {variety} (ValueDefault or
//     ValueFixed) — this catches a caller passing the zero ValueConstraint{}
//     instead of a value built through NewValueConstraint.
//
// Clause 2 (Simple Default Valid, §3.2.6.2) needs the resolved {type
// definition} and is deferred to finalize (#173); it is NOT enforced here.
//
// valueConstraint is a pointer so absence (nil) is distinct from a present zero
// record (mirroring elementdeclaration.go's *ValueConstraint handling); when
// non-nil the pointed-to value is COPIED into the struct and hasValueConstraint
// is set — the pointer itself is never stored, so the caller's value is not
// aliased. annotations is copied; the caller's backing array is not aliased, and
// an empty input is held as nil.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built declaration — may
// legitimately pass the zero xsderr.Loc{}.
func NewAttributeDeclaration(loc xsderr.Loc, name QName, typeDefinitionName QName, scopeVariety ScopeVariety, valueConstraint *ValueConstraint, inheritable bool, annotations []Annotation) (AttributeDeclaration, error) {
	switch scopeVariety {
	case ScopeGlobal, ScopeLocal:
	default:
		return AttributeDeclaration{}, xsderr.New(ruleAPropsCorrect, loc,
			"attribute declaration has an unknown {scope}.{variety}: %s (a-props-correct clause 1)", scopeVariety)
	}
	if valueConstraint != nil {
		switch valueConstraint.Kind() {
		case ValueDefault, ValueFixed:
		default:
			return AttributeDeclaration{}, xsderr.New(ruleAPropsCorrect, loc,
				"attribute declaration {value constraint} has an unknown {variety}: %s (a-props-correct clause 1)", valueConstraint.Kind())
		}
	}
	a := AttributeDeclaration{
		name:               name,
		typeDefinitionName: typeDefinitionName,
		scopeVariety:       scopeVariety,
		inheritable:        inheritable,
	}
	if valueConstraint != nil {
		a.valueConstraint, a.hasValueConstraint = *valueConstraint, true
	}
	if len(annotations) > 0 {
		a.annotations = append([]Annotation(nil), annotations...)
	}
	return a, nil
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
func (a AttributeDeclaration) Name() QName {
	return a.name
}

// TypeDefinitionName returns the {type definition} property (Required) as a
// pre-resolution QName reference — the type/@type name of §3.2.2.
//
// This is NOT the resolved {type definition} component (§3.2.1). The resolved
// component accessor, and its resolution, are deferred to the future
// finalize-phase issue that first introduces phased construction (#173, per
// doc.go's "parse → resolve → finalize"); nothing in this package resolves it
// yet.
func (a AttributeDeclaration) TypeDefinitionName() QName {
	return a.typeDefinitionName
}

// ScopeVariety returns the {scope}.{variety} property (§3.2.1 sc_a).
//
// It does NOT expose {scope}.{parent} (§3.2.1 sc_a — a Complex Type Definition
// or Attribute Group Definition), which is entirely unmodeled by this issue: a
// ScopeLocal attribute is structurally incomplete until that containing
// component exists as a resolved back-reference (a decl↔container link deferred
// to finalize, #173). Until then a local attribute declaration carries only its
// variety, not the container it is scoped to.
func (a AttributeDeclaration) ScopeVariety() ScopeVariety {
	return a.scopeVariety
}

// ValueConstraint returns the {value constraint} property (Optional); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
//
// Absence is meaningful and independent of any sibling Attribute Use: a
// locally-declared attribute (dcl.att.local, §3.2.2.2) always has an absent
// {value constraint} here, while a global declaration (dcl.att.global,
// §3.2.2.1) may carry one.
func (a AttributeDeclaration) ValueConstraint() (ValueConstraint, bool) {
	return a.valueConstraint, a.hasValueConstraint
}

// Inheritable returns the {inheritable} property (Required).
func (a AttributeDeclaration) Inheritable() bool {
	return a.inheritable
}

// Annotations returns the {annotations} property in document order. It returns
// a copy: mutating the result does not affect a. An empty {annotations} yields
// nil.
func (a AttributeDeclaration) Annotations() []Annotation {
	if len(a.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), a.annotations...)
}
