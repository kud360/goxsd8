package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleEPropsCorrect is Element Declaration Properties Correct (Structures
// §3.3.6.1, id="e-props-correct"): an element declaration's properties must
// match the §3.3.1 property tableau. This file enforces the clauses that are
// cheap, purely structural, and cross-reference-free at this layer, citing the
// specific clause number in each message (the rule ID is not sub-anchored per
// clause, matching identityconstraint.go's single-rule-const convention):
//
//   - clause 1 (tableau shape): {scope}.{variety} is one of the legal Scope
//     tokens; {substitution group exclusions} is a subset of {extension,
//     restriction}; {disallowed substitutions} is a subset of {substitution,
//     extension, restriction}. TypeTable's own tableau (clause 6) is enforced
//     in NewTypeTable.
//   - clause 3: a non-empty {substitution group affiliations} forces
//     {scope}.{variety} = global.
//
// Clauses 2 (Element Default Valid), 4/5 (validly-substitutable and circular
// substitution groups), and 7 (type-table alternatives validly substitutable)
// are cross-component / finalize-phase constraints needing resolved type and
// element components, which this package does not resolve yet; they are
// deferred to the schema-assembly issue that introduces phased construction
// (per doc.go's "parse → resolve → finalize") and are NOT enforced here.
const ruleEPropsCorrect xsderr.Rule = "e-props-correct"

// TypeTable is the {type table} property record of an element declaration
// (Structures §3.3.1, id="tt"): an ordered {alternatives} sequence of Type
// Alternative components and a Required {default type definition} (also a Type
// Alternative — the "otherwise" branch of §3.12.4's conditional type
// assignment). The record as a whole is Optional on the element declaration;
// when present, {default type definition} is Required, so a constructed
// TypeTable always carries one.
//
// Construct only through NewTypeTable, which enforces e-props-correct clause 6
// (§3.3.6.1): every {alternatives} member has a present {test}, and the
// {default type definition} is the test-absent "otherwise" alternative. This
// is a purely structural check over the Type Alternatives already in hand — no
// resolved-component cross-reference — so it is safely enforceable now.
// TypeTable is immutable after construction.
type TypeTable struct {
	alternatives          []TypeAlternative
	defaultTypeDefinition TypeAlternative
}

// NewTypeTable builds a TypeTable, rejecting the states e-props-correct clause 6
// (§3.3.6.1) forbids: an {alternatives} member whose {test} is absent, and a
// {default type definition} whose {test} is present (the default is the
// test-absent "otherwise" alternative). alternatives is copied; the caller's
// backing array is not aliased.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built table — may
// legitimately pass the zero xsderr.Loc{}.
func NewTypeTable(loc xsderr.Loc, alternatives []TypeAlternative, defaultTypeDefinition TypeAlternative) (TypeTable, error) {
	for i, alt := range alternatives {
		if _, hasTest := alt.Test(); !hasTest {
			return TypeTable{}, xsderr.New(ruleEPropsCorrect, loc,
				"type table {alternatives}[%d] has an absent {test}, but e-props-correct clause 6 requires every alternative's {test} to be present", i)
		}
	}
	if _, hasTest := defaultTypeDefinition.Test(); hasTest {
		return TypeTable{}, xsderr.New(ruleEPropsCorrect, loc,
			"type table {default type definition} has a present {test}, but it must be the test-absent \"otherwise\" alternative (e-props-correct clause 6)")
	}
	tt := TypeTable{defaultTypeDefinition: defaultTypeDefinition}
	if len(alternatives) > 0 {
		tt.alternatives = append([]TypeAlternative(nil), alternatives...)
	}
	return tt, nil
}

// Alternatives returns the {alternatives} property in document order. It
// returns a copy: mutating the result does not affect t. An empty
// {alternatives} yields nil.
func (t TypeTable) Alternatives() []TypeAlternative {
	if len(t.alternatives) == 0 {
		return nil
	}
	return append([]TypeAlternative(nil), t.alternatives...)
}

// DefaultTypeDefinition returns the {default type definition} property
// (Required): the test-absent "otherwise" Type Alternative selected when no
// {alternatives} member's {test} is true.
func (t TypeTable) DefaultTypeDefinition() TypeAlternative {
	return t.defaultTypeDefinition
}

// ElementDeclaration is the Element Declaration component (Structures §3.3.1,
// id="Element_Declaration_details"): a kind of Term with {name} (bundled with
// {target namespace} as an xsd.QName per this package's "Names are expanded
// QNames" convention — doc.go), {type definition}, {type table} (Optional),
// {scope}, {value constraint} (Optional), {nillable}, {identity-constraint
// definitions}, {substitution group affiliations}, {substitution group
// exclusions}, {disallowed substitutions}, {abstract}, and {annotations}.
//
// Like the other §3 component shapes in this package, ElementDeclaration is a
// STRUCTURAL holder built before resolution. Two properties are carried as
// pre-resolution QName REFERENCES, not resolved components: {type definition}
// (a single reference — the type/@type name of §3.3.2) and {substitution group
// affiliations} (a list of references — the substitutionGroup names). Their
// resolved-component accessors, and the cross-component clauses of
// e-props-correct that need them (clauses 2, 4, 5, 7), are deferred to the
// finalize-phase issue (#173) that first introduces phased construction; this
// package resolves neither yet.
//
// {scope}.{parent} (§3.3.1 sc_e-parent) is entirely UNMODELED by this issue.
// Only {scope}.{variety} is carried (as a ScopeVariety). A ScopeLocal element
// is therefore structurally incomplete: its containing Complex Type Definition
// or Model Group Definition (issue #171 and later) does not exist as a type
// yet, so there is nothing to point {parent} at. The gap is named here rather
// than buried; ScopeVariety() documents it too.
//
// Ratchet impact: unchanged. This is a leaf shape with no parser producer; the
// schema conformance lane moves only when the producer (#174/#175) wires it in.
//
// Construct only through NewElementDeclaration, which rejects the states
// e-props-correct (§3.3.6.1) clauses 1 and 3 forbid so they are unrepresentable
// (STYLE T1). ElementDeclaration is immutable after construction.
type ElementDeclaration struct {
	name                          QName
	typeDefinitionName            QName
	typeTable                     TypeTable
	hasTypeTable                  bool
	scopeVariety                  ScopeVariety
	valueConstraint               ValueConstraint
	hasValueConstraint            bool
	nillable                      bool
	identityConstraints           []IdentityConstraint
	substitutionGroupAffiliations []QName
	substitutionGroupExclusions   []DerivationMethod
	abstract                      bool
	disallowedSubstitutions       []DerivationMethod
	annotations                   []Annotation
}

// NewElementDeclaration builds an ElementDeclaration, rejecting the states
// Element Declaration Properties Correct (§3.3.6.1, e-props-correct) clauses 1
// and 3 forbid:
//
//   - clause 1: scopeVariety must be a legal Scope token (ScopeGlobal or
//     ScopeLocal); every substitutionGroupExclusions member must be extension
//     or restriction (the §3.3.1 {substitution group exclusions} subset); every
//     disallowedSubstitutions member must be substitution, extension, or
//     restriction (the §3.3.1 {disallowed substitutions} subset).
//   - clause 3: a non-empty substitutionGroupAffiliations forces
//     scopeVariety = ScopeGlobal.
//
// typeTable and valueConstraint are pointers so absence (nil) is distinct from
// a present zero record (mirroring identityconstraint.go's referencedKey); when
// non-nil the pointed-to value is COPIED into the struct and the corresponding
// has* flag is set — the pointer itself is never stored, so the caller's value
// is not aliased. Every slice parameter is copied; the caller's backing arrays
// are not aliased, and an empty input is held as nil.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built declaration — may
// legitimately pass the zero xsderr.Loc{}.
func NewElementDeclaration(loc xsderr.Loc, name QName, typeDefinitionName QName, typeTable *TypeTable, scopeVariety ScopeVariety, valueConstraint *ValueConstraint, nillable bool, identityConstraints []IdentityConstraint, substitutionGroupAffiliations []QName, substitutionGroupExclusions []DerivationMethod, abstract bool, disallowedSubstitutions []DerivationMethod, annotations []Annotation) (ElementDeclaration, error) {
	switch scopeVariety {
	case ScopeGlobal, ScopeLocal:
	default:
		return ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration has an unknown {scope}.{variety}: %s (e-props-correct clause 1)", scopeVariety)
	}
	for i, m := range substitutionGroupExclusions {
		switch m {
		case DerivationExtension, DerivationRestriction:
		default:
			return ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, loc,
				"element declaration {substitution group exclusions}[%d] is %s, but only extension or restriction are legal (e-props-correct clause 1)", i, m)
		}
	}
	for i, m := range disallowedSubstitutions {
		switch m {
		case DerivationSubstitution, DerivationExtension, DerivationRestriction:
		default:
			return ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, loc,
				"element declaration {disallowed substitutions}[%d] is %s, but only substitution, extension, or restriction are legal (e-props-correct clause 1)", i, m)
		}
	}
	if len(substitutionGroupAffiliations) > 0 && scopeVariety != ScopeGlobal {
		return ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration has a non-empty {substitution group affiliations} but its {scope}.{variety} is %s, not global (e-props-correct clause 3)", scopeVariety)
	}
	e := ElementDeclaration{
		name:               name,
		typeDefinitionName: typeDefinitionName,
		scopeVariety:       scopeVariety,
		nillable:           nillable,
		abstract:           abstract,
	}
	if typeTable != nil {
		e.typeTable, e.hasTypeTable = *typeTable, true
	}
	if valueConstraint != nil {
		e.valueConstraint, e.hasValueConstraint = *valueConstraint, true
	}
	if len(identityConstraints) > 0 {
		e.identityConstraints = append([]IdentityConstraint(nil), identityConstraints...)
	}
	if len(substitutionGroupAffiliations) > 0 {
		e.substitutionGroupAffiliations = append([]QName(nil), substitutionGroupAffiliations...)
	}
	if len(substitutionGroupExclusions) > 0 {
		e.substitutionGroupExclusions = append([]DerivationMethod(nil), substitutionGroupExclusions...)
	}
	if len(disallowedSubstitutions) > 0 {
		e.disallowedSubstitutions = append([]DerivationMethod(nil), disallowedSubstitutions...)
	}
	if len(annotations) > 0 {
		e.annotations = append([]Annotation(nil), annotations...)
	}
	return e, nil
}

// term marks ElementDeclaration as a Term (§3.3.1: "a kind of Term"); see
// term.go.
func (ElementDeclaration) term() {}

// Name returns the {name} property, bundled with {target namespace} as a QName.
func (e ElementDeclaration) Name() QName {
	return e.name
}

// TypeDefinitionName returns the {type definition} property (Required) as a
// pre-resolution QName reference — the type/@type name of §3.3.2.
//
// This is NOT the resolved {type definition} component (§3.3.1). The resolved
// component accessor, and its resolution, are deferred to the future
// finalize-phase issue that first introduces phased construction (#173, per
// doc.go's "parse → resolve → finalize"); nothing in this package resolves it
// yet.
func (e ElementDeclaration) TypeDefinitionName() QName {
	return e.typeDefinitionName
}

// TypeTable returns the {type table} property (Optional); the second result is
// false when it is absent, in which case the first result is not meaningful.
func (e ElementDeclaration) TypeTable() (TypeTable, bool) {
	return e.typeTable, e.hasTypeTable
}

// ScopeVariety returns the {scope}.{variety} property (§3.3.1 sc_e).
//
// It does NOT expose {scope}.{parent} (§3.3.1 sc_e-parent), which is entirely
// unmodeled by this issue: a ScopeLocal element is structurally incomplete
// until the Complex Type Definition / Model Group Definition components exist
// to be its {parent} (issue #171 and later). Until then a local element
// declaration carries only its variety, not the container it is scoped to.
func (e ElementDeclaration) ScopeVariety() ScopeVariety {
	return e.scopeVariety
}

// ValueConstraint returns the {value constraint} property (Optional); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
func (e ElementDeclaration) ValueConstraint() (ValueConstraint, bool) {
	return e.valueConstraint, e.hasValueConstraint
}

// Nillable returns the {nillable} property.
func (e ElementDeclaration) Nillable() bool {
	return e.nillable
}

// IdentityConstraints returns the {identity-constraint definitions} property in
// document order. It returns a copy: mutating the result does not affect e. An
// empty set yields nil.
func (e ElementDeclaration) IdentityConstraints() []IdentityConstraint {
	if len(e.identityConstraints) == 0 {
		return nil
	}
	return append([]IdentityConstraint(nil), e.identityConstraints...)
}

// SubstitutionGroupAffiliationNames returns the {substitution group
// affiliations} property as pre-resolution QName references — the
// substitutionGroup names of §3.3.2 — in document order. It returns a copy:
// mutating the result does not affect e. An empty set yields nil.
//
// These are NOT the resolved {substitution group affiliations} Element
// Declaration components (§3.3.1). The resolved-component accessor, and
// e-props-correct clauses 4/5 (validly-substitutable and circular substitution
// groups), are deferred to the future finalize-phase issue that first
// introduces phased construction (#173, per doc.go's "parse → resolve →
// finalize"); nothing in this package resolves them yet.
func (e ElementDeclaration) SubstitutionGroupAffiliationNames() []QName {
	if len(e.substitutionGroupAffiliations) == 0 {
		return nil
	}
	return append([]QName(nil), e.substitutionGroupAffiliations...)
}

// SubstitutionGroupExclusions returns the {substitution group exclusions}
// property (a subset of {extension, restriction}) in document order. It returns
// a copy: mutating the result does not affect e. An empty subset yields nil.
func (e ElementDeclaration) SubstitutionGroupExclusions() []DerivationMethod {
	if len(e.substitutionGroupExclusions) == 0 {
		return nil
	}
	return append([]DerivationMethod(nil), e.substitutionGroupExclusions...)
}

// Abstract returns the {abstract} property.
func (e ElementDeclaration) Abstract() bool {
	return e.abstract
}

// DisallowedSubstitutions returns the {disallowed substitutions} property (a
// subset of {substitution, extension, restriction}) in document order. It
// returns a copy: mutating the result does not affect e. An empty subset yields
// nil.
func (e ElementDeclaration) DisallowedSubstitutions() []DerivationMethod {
	if len(e.disallowedSubstitutions) == 0 {
		return nil
	}
	return append([]DerivationMethod(nil), e.disallowedSubstitutions...)
}

// Annotations returns the {annotations} property in document order. It returns
// a copy: mutating the result does not affect e. An empty {annotations} yields
// nil.
func (e ElementDeclaration) Annotations() []Annotation {
	if len(e.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), e.annotations...)
}
