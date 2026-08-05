package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleAPropsCorrect is Attribute Declaration Properties Correct (Structures
// §3.2.6.1, id="a-props-correct"): an attribute declaration's properties must
// match the §3.2.1 property tableau. This file enforces the clauses that are
// cheap, purely structural, and cross-reference-free at this layer, citing the
// specific clause number in each message (the rule ID is not sub-anchored per
// clause, matching elementdeclaration.go's single-rule-const convention):
//
//   - clause 1 (tableau shape): {name} is present, {scope}.{variety} is one of
//     the legal Scope tokens (global or local), and a present {value
//     constraint} carries a legal {variety} (default or fixed).
//
// Clause 2 (Simple Default Valid — §3.2.6.2 cos-valid-simple-default) is a
// cross-component constraint: it needs the resolved {type definition} to validate
// the {value constraint}'s {lexical form} against it, and a value space to do the
// validating, neither of which this constructor has. It IS enforced, but at
// FINALIZE rather than here — Phase E of resolve (valueconstraintvalid.go's
// checkAttributeDeclarationValueConstraint, #371, on the phased-construction seam
// #173 introduced) — so a rejection carries this rule and this declaration's Loc
// while being raised by SchemaBuilder.Finalize, not by NewAttributeDeclaration.
const ruleAPropsCorrect xsderr.Rule = "a-props-correct"

// AttributeDeclaration is the Attribute Declaration component (Structures
// §3.2.1, id="Attribute_Declaration_details"): a kind of Annotated Component
// with {name} (bundled with {target namespace} as an xsd.QName per this
// package's "Names are expanded QNames" convention — doc.go), {type
// definition}, {scope}, {value constraint} (Optional), {inheritable}, and
// {annotations}.
//
// Like the other §3 component shapes in this package, AttributeDeclaration is a
// STRUCTURAL holder built before resolution. Its {type definition} is carried as
// a TypeDefinitionOrRef: a pre-resolution QName REFERENCE for the type/@type and
// xs:anySimpleType tiers of §3.2.2.2 dcl.att.local, or the owned anonymous
// component itself for the inline <simpleType> tier. Note that §3.2.2.2's chain
// has three tiers, not the four of an element's §3.3.2.1 dcl.elt.common: an
// attribute has no substitution-group analog. A resolved-component accessor for
// the by-name arm is still deferred; a-props-correct clause 2 (Simple Default
// Valid) is decided at FINALIZE, over the resolved {type definition} and the
// installed ValueSpace (valueconstraintvalid.go, #371), not on this component.
//
// {scope}.{parent} (§3.2.1 sc_a — a Complex Type Definition or Attribute Group
// Definition) is entirely UNMODELED, tracked as #169. Only {scope}.{variety} is
// carried (as a ScopeVariety, shared with element declarations per closedsets.go
// — only the variety closed set is shared, the sc_a Scope record itself is
// not). A ScopeLocal attribute is therefore structurally incomplete: it does not
// name the Complex Type Definition or Attribute Group Definition it is scoped
// to. The element-declaration precedent is now wired (elementdeclaration.go's
// Scope / ElementScopeParent) and is the shape to follow here, with sc_a's own
// distinct two-member sum — the two alternations differ in their second member,
// so they must not share one type. The gap is named here rather than buried;
// ScopeVariety() documents it too.
//
// The {value constraint} is deliberately an INDEPENDENTLY-optional slot: under
// the local mapping dcl.att.local (§3.2.2.2) a locally-declared attribute's own
// {value constraint} is always absent (any default/fixed feeds the sibling
// Attribute Use's {value constraint} instead), while the global mapping
// dcl.att.global (§3.2.2.1) populates it here — so absence must be
// representable on the declaration independently of the use.
//
// Ratchet impact: the schema lane widens whenever the producer starts mapping a
// {type definition} shape it used to decline — most recently the inline
// anonymous <simpleType> of a local declaration (#229), which the
// InlineTypeDefinition arm of the slot exists to hold.
//
// Construct only through NewAttributeDeclaration, which rejects the states
// a-props-correct (§3.2.6.1) clause 1 forbids so they are unrepresentable
// (STYLE T1). AttributeDeclaration is immutable after construction.
type AttributeDeclaration struct {
	loc                xsderr.Loc // source position; provenance, not a §3.2.1 property
	name               QName
	typeDefinition     TypeDefinitionOrRef
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
//   - name must be present: its local part may not be empty. The §3.2.1
//     tableau types {name} as a Required xs:NCName, and NCName's value space
//     (Datatypes §3.4.7, pattern \i\c*) excludes the empty string, so a
//     zero-Local QName is categorically not a legal {name}. The §5.3 Missing
//     Sub-components escape hatch does not cover it: §5.3 is scoped to
//     properties whose value is another component reached by QName
//     ·resolution·, and {name} is the identity other components resolve
//     AGAINST — unlike the deferred QName REFERENCE this component carries in
//     its {type definition} slot. The guard is unconditional because an
//     attribute declaration has NO anonymous form: §3.2.2.1/§3.2.2.2 make the
//     name attribute present for every declaration-producing case, and the ref
//     form resolves to an existing declaration rather than minting an unnamed
//     one. That reasoning is deliberately NOT generalized to NewComplexType /
//     NewSimpleType, whose components have a genuine anonymous form ({name}
//     Optional, {context} Required instead). Testing the local part, not
//     name == QName{}, is deliberate: the latter would admit
//     QName{Space: "urn:x", Local: ""} as a named declaration. Same idiom as
//     NewElementDeclaration's e-props-correct clause 1 check;
//   - scopeVariety must be a legal Scope token (ScopeGlobal or ScopeLocal);
//   - a present valueConstraint must carry a legal {variety} (ValueDefault or
//     ValueFixed) — this catches a caller passing the zero ValueConstraint{}
//     instead of a value built through NewValueConstraint.
//
// Clause 2 (Simple Default Valid, §3.2.6.2) needs the resolved {type definition}
// and a value space, so it is decided at FINALIZE instead (Phase E,
// valueconstraintvalid.go, #371) — enforced, but not by this constructor.
//
// It also rejects the two illegal encodings of the typeDefinition slot — a
// zero-named TypeDefinitionRef and an InlineTypeDefinition that is empty or
// wraps a NAMED type — charged to xsderr.RuleComponentInvariant; see
// TypeDefinitionOrRef and checkTypeDefinitionOrRef. A nil slot is the legal
// encoding of an absent {type definition}, which a programmatically built
// declaration is in before the §3.2.2.2 defaulting tiers are applied.
//
// valueConstraint is a pointer so absence (nil) is distinct from a present zero
// record (mirroring elementdeclaration.go's *ValueConstraint handling); when
// non-nil the pointed-to value is COPIED into the struct and hasValueConstraint
// is set — the pointer itself is never stored, so the caller's value is not
// aliased. annotations is copied; the caller's backing array is not aliased, and
// an empty input is held as nil.
//
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the declaration's provenance. Pass the position of this
// declaration's own declaring element, never a convenient nearby one (a parent
// element's, say) — it is observable, not merely an error-charging convenience.
// A caller with no real parser position — a synthesized or programmatically
// built declaration — passes the zero xsderr.Loc{}, which reads as "unknown".
func NewAttributeDeclaration(loc xsderr.Loc, name QName, typeDefinition TypeDefinitionOrRef, scopeVariety ScopeVariety, valueConstraint *ValueConstraint, inheritable bool, annotations []Annotation) (AttributeDeclaration, error) {
	if name.Local == "" {
		return AttributeDeclaration{}, xsderr.New(ruleAPropsCorrect, loc,
			"attribute declaration has an absent {name}, but the §3.2.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (a-props-correct clause 1)")
	}
	if err := checkTypeDefinitionOrRef(loc, typeDefinition, "attribute declaration "+name.String()); err != nil {
		return AttributeDeclaration{}, err
	}
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
		loc:            loc,
		name:           name,
		typeDefinition: typeDefinition,
		scopeVariety:   scopeVariety,
		inheritable:    inheritable,
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

// Loc reports the source position of the declaring element — provenance, not a
// §3.2.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown.
func (a AttributeDeclaration) Loc() xsderr.Loc {
	return a.loc
}

// TypeDefinition returns the {type definition} property (Required) as the
// TypeDefinitionOrRef sealed sum: a TypeDefinitionRef naming a top-level simple
// type (the type/@type or xs:anySimpleType tiers of §3.2.2.2 dcl.att.local), or
// an InlineTypeDefinition owning the anonymous type of an inline <simpleType>
// child (its first tier). It is nil only for a declaration built with an absent
// {type definition}.
//
// The by-name arm is NOT resolved into a component here; a consumer obtains the
// component by a read-time schema.Type(name) lookup, exactly as for an element
// declaration. The inline arm needs no lookup — it carries the component.
func (a AttributeDeclaration) TypeDefinition() TypeDefinitionOrRef {
	return a.typeDefinition
}

// ScopeVariety returns the {scope}.{variety} property (§3.2.1 sc_a).
//
// It does NOT expose {scope}.{parent} (§3.2.1 sc_a — a Complex Type Definition
// or Attribute Group Definition), which is entirely unmodeled (#169): a
// ScopeLocal attribute carries only its variety, not the container it is scoped
// to. The element declaration's parallel {parent} IS wired
// (ElementDeclaration.Scope), and is the precedent to follow here.
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
