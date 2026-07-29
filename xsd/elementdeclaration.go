package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleEPropsCorrect is Element Declaration Properties Correct (Structures
// §3.3.6.1, id="e-props-correct"): an element declaration's properties must
// match the §3.3.1 property tableau. This file enforces the clauses that are
// cheap, purely structural, and cross-reference-free at this layer, citing the
// specific clause number in each message (the rule ID is not sub-anchored per
// clause, matching identityconstraint.go's single-rule-const convention):
//
//   - clause 1 (tableau shape): {name} is present; {substitution group
//     exclusions} is a subset of {extension, restriction}; {disallowed
//     substitutions} is a subset of {substitution, extension, restriction}.
//     TypeTable's own tableau (clause 6) is enforced in NewTypeTable, and the
//     {scope} record's own shape — {variety} a legal token, {parent} present iff
//     local — in NewGlobalScope/NewLocalScope (see Scope).
//   - clause 3: a non-empty {substitution group affiliations} forces
//     {scope}.{variety} = global.
//
// Clauses 2 (Element Default Valid), 4 (validly-substitutable), and 7
// (type-table alternatives validly substitutable) are cross-component
// finalize-phase constraints needing resolved type and element components; they
// are NOT enforced here. Clause 5 (no circular substitution groups) IS enforced,
// but at finalize (resolve.go's checkSubstitutionGroupsAcyclic, #173), not in
// this constructor — it needs the whole {substitution group affiliations} graph,
// which only exists once the schema set is assembled.
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

// ElementScopeParent is the sealed sum of the two component kinds an element
// declaration's {scope}.{parent} may name (Structures §3.3.1, Scope record
// id="sc_e", {parent}: "Either a Complex Type Definition or a Model Group
// Definition"). The spec's alternation is closed at two, so the set of variant
// shapes is closed; the unexported elementScopeParent marker method seals it
// (STYLE T2/T7, the PRINCIPLES 7 sealed-sum exception), mirroring term.go's
// TermOrRef, so consumers exhaustively switch these two variants and no third is
// representable.
//
// It is named for the ELEMENT declaration deliberately. The attribute
// declaration's own {scope}.{parent} (§3.2.1, record id="sc_a") is a DIFFERENT
// two-member alternation — Complex Type Definition or Attribute Group Definition
// — and must get its own sealed sum rather than share this one: merging the two
// into a single three-variant sum would make "an element scoped to an attribute
// group" representable.
//
// Both variants carry a QName REFERENCE to the container, not the container
// component itself. Embedding the value is impossible here: construction is
// bottom-up (a complex type's content model, which owns the local declaration,
// is built before the complex type is), so the parent does not exist when the
// child is made. A bare QName without the variant's kind discriminant would not
// do either: Complex Type Definitions and Model Group Definitions occupy
// INDEPENDENT symbol spaces on Schema (§3.17.1 {type definitions} versus {model
// group definitions}), so a CTD and an MGD may share one expanded name. The kind
// therefore lives in the variant type, and a consumer follows the reference with
// a read-time lookup in the index that kind selects — the same
// pre-resolution-reference convention as TypeDefinitionName, which Schema.Type
// serves. A ComplexTypeScopeParent is followable today; a ModelGroupScopeParent
// waits on the Schema.ModelGroup(QName) accessor this package does not export
// yet (STYLE T5 — no consumer justifies one, the same follow-cost asymmetry
// resolve.go already records for ModelGroupRef).
//
// Unlike those reference slots, this one is NOT checked by finalize: resolve.go
// adds no src-resolve (§3.17.6.2) clause for it. src-resolve governs QNames
// supplied by a schema document; {scope}.{parent} is synthesized by the producer
// from the ancestor axis of the very item it is producing, so its target exists
// by construction and there is nothing to dangle.
type ElementScopeParent interface{ elementScopeParent() }

// ComplexTypeScopeParent is the ElementScopeParent variant naming the containing
// Complex Type Definition (§3.3.2.3 dcl.elt.local: "If the <element> element
// information item has <complexType> as an ancestor, the Complex Type Definition
// corresponding to that item"). Name is that definition's expanded {name}; it is
// a PRESENT reference, never the absent (zero) QName — NewLocalScope rejects a
// zero Name (see there for why). The field is read-only by convention; do not
// mutate it after construction.
type ComplexTypeScopeParent struct{ Name QName }

// ModelGroupScopeParent is the ElementScopeParent variant naming the containing
// Model Group Definition (§3.3.2.3 dcl.elt.local: "otherwise (the <element>
// element information item is within a named <group> element information item),
// the Model Group Definition corresponding to that item"). Name is that
// definition's expanded {name}, which §3.7.1 types as a Required xs:NCName, so it
// is always present; NewLocalScope rejects a zero Name. The field is read-only by
// convention; do not mutate it after construction.
type ModelGroupScopeParent struct{ Name QName }

// elementScopeParent marks ComplexTypeScopeParent as an ElementScopeParent
// (§3.3.1 sc_e-parent); see the ElementScopeParent doc.
func (ComplexTypeScopeParent) elementScopeParent() {}

// elementScopeParent marks ModelGroupScopeParent as an ElementScopeParent
// (§3.3.1 sc_e-parent); see the ElementScopeParent doc.
func (ModelGroupScopeParent) elementScopeParent() {}

// elementScopeParentName returns the expanded name a variant references. The
// default arm asserts the sealed-sum invariant and is unreachable for any value
// an outside package can produce: elementScopeParent is unexported, so the two
// variants above are the only implementations that exist (mirroring resolve.go's
// non-exhaustive-switch assertions).
func elementScopeParentName(parent ElementScopeParent) QName {
	switch p := parent.(type) {
	case ComplexTypeScopeParent:
		return p.Name
	case ModelGroupScopeParent:
		return p.Name
	default:
		panic("xsd: elementScopeParentName: non-exhaustive ElementScopeParent switch")
	}
}

// Scope is the {scope} property record of an element declaration (Structures
// §3.3.1, id="sc_e"): a Required {variety} in {global, local} and a {parent}
// that is "Required if {variety} is local, otherwise must be ·absent·".
//
// The two properties are one fact, so only {parent} is stored and Variety() is
// DERIVED from its presence (STYLE D3), exactly as complextype.go's
// ElementContent derives element-only-versus-mixed from its Mixed bool. The
// tableau's correlation therefore needs no runtime check anywhere: a global scope
// carrying a parent, and a local scope missing one, are both unrepresentable.
//
// The zero Scope is the global scope, matching NewGlobalScope; the two
// constructors are the only way to obtain a local one, since parent is
// unexported.
//
// A local Scope names its {parent} rather than embedding it, and requires that
// name to be present — see ElementScopeParent for why the reference is a
// discriminated QName and NewLocalScope for why the name may not be absent.
type Scope struct {
	parent ElementScopeParent // nil ⇔ {variety} = global
}

// NewGlobalScope returns the global {scope} of a top-level element declaration
// (§3.3.2.2 dcl.elt.global: {variety} global, {parent} ·absent·). It cannot fail:
// the record has no other property to get wrong.
func NewGlobalScope() Scope {
	return Scope{}
}

// NewLocalScope returns the local {scope} of an element declaration nested in a
// <complexType> or a named <group> (§3.3.2.3 dcl.elt.local), naming the container
// it is scoped to. It rejects the two states the §3.3.1 tableau and this
// representation forbid, citing e-props-correct clause 1 (§3.3.6.1):
//
//   - a nil parent: {parent} is Required when {variety} is local, and a local
//     scope with no container to be available within contradicts §3.3.1's
//     availability prose ("E is available for use only within ... E.{scope}.
//     {parent}").
//   - a parent variant whose Name is the absent (zero) QName: this
//     representation identifies the container BY NAME (ElementScopeParent), so an
//     unnamed container could not be found again.
//
// The second rejection is unreachable from this module's parser; it guards
// scopes built programmatically (a direct caller, a test) instead. Only a
// top-level <complexType> or a top-level named <group> ever mints an
// ElementScopeParent, and the producer rejects each as a grammar fault at the
// top of the function that maps it — BEFORE any content is built — when its
// name attribute is absent or empty, so no variant carrying an absent name is
// ever constructed. The remaining source of an anonymous Complex Type
// Definition, an inline <complexType>, is declined by every producer path that
// could reach one. Should inline anonymous complex types land, this rejection is
// the compile-of-record that the representation must be revisited (a component
// handle, not a name) rather than silently mis-scoping.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built scope — may
// legitimately pass the zero xsderr.Loc{}.
func NewLocalScope(loc xsderr.Loc, parent ElementScopeParent) (Scope, error) {
	if parent == nil {
		return Scope{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration has {scope}.{variety} = local but an absent {scope}.{parent}, which the §3.3.1 tableau requires to be present when the variety is local (e-props-correct clause 1)")
	}
	if name := elementScopeParentName(parent); name.Local == "" {
		return Scope{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration's {scope}.{parent} names no container: the %T variant carries an absent name, but this representation identifies the containing complex type or model group definition by name (e-props-correct clause 1)", parent)
	}
	return Scope{parent: parent}, nil
}

// Variety returns the {variety} property (§3.3.1 sc_e-variety): ScopeLocal when
// a {parent} is present, ScopeGlobal otherwise. The token is derived from
// {parent}'s presence, never stored (STYLE D3).
func (s Scope) Variety() ScopeVariety {
	if s.parent == nil {
		return ScopeGlobal
	}
	return ScopeLocal
}

// Parent returns the {parent} property (§3.3.1 sc_e-parent) as a discriminated
// QName reference to the containing Complex Type Definition or Model Group
// Definition; the second result is false when it is absent (a global scope), in
// which case the first result is nil.
//
// This is NOT the resolved container component: see ElementScopeParent for why
// the reference is carried by name and how a consumer follows it.
func (s Scope) Parent() (ElementScopeParent, bool) {
	return s.parent, s.parent != nil
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
// affiliations} (a list of references — the substitutionGroup names). Finalize
// (resolve.go, #173) VALIDATES that both resolve against the schema indexes
// (src-resolve clauses 1.1 and 1.3) and that the substitution-group graph is
// acyclic (e-props-correct clause 5), but does NOT rewrite them into resolved
// components: the QNames are retained, and a consumer follows them by read-time
// schema.Type/schema.Element lookups. The remaining cross-component clauses that
// need resolved components (clauses 2, 4, 7) stay deferred.
//
// The whole {scope} record is carried, {parent} included (a Scope value, not a
// bare ScopeVariety): a local declaration names the Complex Type Definition or
// Model Group Definition it is scoped to, a global one names none, and the
// producer populates the reference from the ancestor axis (§3.3.2.3
// dcl.elt.local / §3.3.2.2 dcl.elt.global). {parent} is a THIRD pre-resolution
// reference alongside {type definition} and {substitution group affiliations},
// but unlike them it is producer-synthesized rather than schema-document-
// supplied, so finalize adds no src-resolve check for it — see
// ElementScopeParent.
//
// Ratchet impact: unchanged. Wiring {scope}.{parent} adds a component property
// no validation rule reads yet, so no conformance lane moves.
//
// Construct only through NewElementDeclaration, which rejects the states
// e-props-correct (§3.3.6.1) clauses 1 and 3 forbid so they are unrepresentable
// (STYLE T1). ElementDeclaration is immutable after construction.
type ElementDeclaration struct {
	name                          QName
	typeDefinitionName            QName
	typeTable                     TypeTable
	hasTypeTable                  bool
	scope                         Scope
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
//   - clause 1: name must be present — the §3.3.1 tableau types {name} as a
//     Required xs:NCName, and NCName's value space (Datatypes §3.4.7, pattern
//     \i\c*) excludes the empty string, so a zero-Local QName is categorically
//     not a legal {name}. The §5.3 Missing Sub-components escape hatch does not
//     cover it: §5.3 is scoped to properties whose value is another component
//     reached by QName ·resolution·, and {name} is the identity other components
//     resolve AGAINST — unlike the deferred QName REFERENCES this component
//     carries ({type definition}, {substitution group affiliations}), which
//     finalize (#173) resolves.
//   - clause 1: every substitutionGroupExclusions member must be extension or
//     restriction (the §3.3.1 {substitution group exclusions} subset); every
//     disallowedSubstitutions member must be substitution, extension, or
//     restriction (the §3.3.1 {disallowed substitutions} subset).
//   - clause 3: a non-empty substitutionGroupAffiliations forces
//     scope.Variety() = ScopeGlobal.
//
// The rest of clause 1's {scope} shape is not checked here because it is
// unrepresentable: scope is a Scope, obtainable only from NewGlobalScope or
// NewLocalScope, so its {variety} is always a legal token and its {parent} is
// present exactly when the variety is local (see Scope).
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
func NewElementDeclaration(loc xsderr.Loc, name QName, typeDefinitionName QName, typeTable *TypeTable, scope Scope, valueConstraint *ValueConstraint, nillable bool, identityConstraints []IdentityConstraint, substitutionGroupAffiliations []QName, substitutionGroupExclusions []DerivationMethod, abstract bool, disallowedSubstitutions []DerivationMethod, annotations []Annotation) (ElementDeclaration, error) {
	if name.Local == "" {
		return ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration has an absent {name}, but the §3.3.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (e-props-correct clause 1)")
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
	if len(substitutionGroupAffiliations) > 0 && scope.Variety() != ScopeGlobal {
		return ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration has a non-empty {substitution group affiliations} but its {scope}.{variety} is %s, not global (e-props-correct clause 3)", scope.Variety())
	}
	e := ElementDeclaration{
		name:               name,
		typeDefinitionName: typeDefinitionName,
		scope:              scope,
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
// Its Local is never empty on a value built through NewElementDeclaration, which
// rejects an absent {name} (e-props-correct clause 1).
func (e ElementDeclaration) Name() QName {
	return e.name
}

// TypeDefinitionName returns the {type definition} property (Required) as a
// pre-resolution QName reference — the type/@type name of §3.3.2.
//
// This is NOT the resolved {type definition} component (§3.3.1). Finalize (#173)
// validates the name resolves to a type definition (src-resolve clause 1.1) but
// adds no resolved-component accessor: the QName is retained, and a consumer
// obtains the component by a read-time schema.Type(name) lookup.
func (e ElementDeclaration) TypeDefinitionName() QName {
	return e.typeDefinitionName
}

// TypeTable returns the {type table} property (Optional); the second result is
// false when it is absent, in which case the first result is not meaningful.
func (e ElementDeclaration) TypeTable() (TypeTable, bool) {
	return e.typeTable, e.hasTypeTable
}

// Scope returns the {scope} property record (§3.3.1 sc_e): its {variety} and,
// for a local declaration, the {parent} naming the Complex Type Definition or
// Model Group Definition the declaration is scoped to.
func (e ElementDeclaration) Scope() Scope {
	return e.scope
}

// ScopeVariety returns the {scope}.{variety} property (§3.3.1 sc_e-variety), a
// shorthand for e.Scope().Variety() kept for the many callers that only ask
// global-versus-local. Read {scope}.{parent} through Scope.
func (e ElementDeclaration) ScopeVariety() ScopeVariety {
	return e.scope.Variety()
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
// Declaration components (§3.3.1). Finalize (#173) validates each name resolves
// to an element declaration (src-resolve clause 1.3) and that the affiliation
// graph is acyclic (e-props-correct clause 5), but adds no resolved-component
// accessor: the QNames are retained, followed by read-time schema.Element
// lookups. Clause 4 (validly-substitutable) stays deferred.
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
