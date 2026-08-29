package xsd

import (
	"strconv"

	"github.com/kud360/goxsd8/xsderr"
)

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
// Clause 7 (type-table alternatives validly substitutable) is a cross-component
// finalize-phase constraint needing resolved type and element components; it is
// NOT enforced here but in Phase D (typetablesubstitutable.go's
// checkTypeTableSubstitutability). Clauses 2, 4 and 5 are likewise
// finalize-phase, and all three ARE enforced there rather than in this
// constructor, each needing more than the declaration in hand: clause 2 (Element
// Default Valid, §3.3.6.2 cos-valid-default) needs the resolved {type
// definition}'s {content type} and, for its clause 2.2, an ·emptiable· verdict
// over that content type's particle (resolve.go's Phase E,
// elementdefaultvalid.go's checkElementDefaultValid, #463); clause 4
// (validly-substitutable) needs the resolved {type definition} of the declaration
// AND of every head it is affiliated to (resolve.go's Phase D,
// substitutiongrouptypes.go's checkSubstitutionGroupTypes, #395), and clause 5
// (no circular substitution groups) needs the whole {substitution group
// affiliations} graph, which only exists once the schema set is assembled
// (resolve.go's checkSubstitutionGroupsAcyclic, #173).
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

// ElementScopeParent is the sealed sum of the ways an element declaration's
// {scope}.{parent} identifies its container (Structures §3.3.1, Scope record
// id="sc_e", {parent}: "Either a Complex Type Definition or a Model Group
// Definition"). The unexported elementScopeParent marker method seals it (STYLE
// T2), mirroring term.go's TermOrRef, so consumers exhaustively switch these
// variants and no fourth is representable.
//
// There are THREE arms for the spec's TWO component kinds, and the extra one is
// a REPRESENTATION split inside one kind, not a third kind. A Complex Type
// Definition container may be named (a top-level <complexType>) or ANONYMOUS (an
// inline <complexType> child of an <element>, §3.4.1 ctd-context), and the two
// are not identifiable the same way: a named one is followable by QName, an
// anonymous one has no name at all and is identified only by the ComponentID
// minted for the inline construct it belongs to. Encoding that as an optional
// second field on one arm would make "named AND anonymous" and "neither"
// representable; a separate arm makes both unrepresentable (STYLE T1), exactly
// as complextype.go's ComplexTypeContext partitions its own construction paths.
// A Model Group Definition needs no such split: §3.7.1 types its {name} as a
// Required xs:NCName, so a nameless one is not a legal component.
//
//   - ComplexTypeScopeParent — a NAMED containing complex type definition;
//   - AnonymousComplexTypeScopeParent — an ANONYMOUS containing complex type
//     definition, by owner identity;
//   - ModelGroupScopeParent — a containing model group definition.
//
// It is named for the ELEMENT declaration deliberately. The attribute
// declaration's own {scope}.{parent} (§3.2.1, record id="sc_a") is a DIFFERENT
// alternation — Complex Type Definition or Attribute Group Definition — and must
// get its own sealed sum rather than share this one: merging the two would make
// "an element scoped to an attribute group" representable.
//
// Every variant carries a REFERENCE to the container, not the container
// component itself. Embedding the value is impossible here: construction is
// bottom-up (a complex type's content model, which owns the local declaration,
// is built before the complex type is), so the parent does not exist when the
// child is made. A bare QName without the variant's kind discriminant would not
// do either: Complex Type Definitions and Model Group Definitions occupy
// INDEPENDENT symbol spaces on Schema (§3.17.1 {type definitions} versus {model
// group definitions}), so a CTD and an MGD may share one expanded name. The kind
// therefore lives in the variant type, and a consumer follows a by-NAME
// reference with a read-time lookup in the index that kind selects — the same
// pre-resolution-reference convention as TypeDefinitionRef, which Schema.Type
// serves. Both ComplexTypeScopeParent and ModelGroupScopeParent are followable
// today, by the same read-time lookup pattern: Schema.Type for the former,
// Schema.ModelGroup for the latter; an AnonymousComplexTypeScopeParent still
// waits on the ID→component resolver ComponentID's doc describes, which no
// consumer justifies yet either.
//
// Unlike those reference slots, this one is NOT checked by finalize: resolve.go
// adds no src-resolve (§3.17.6.2) clause for it. src-resolve governs QNames
// supplied by a schema document; {scope}.{parent} is synthesized by the producer
// from the ancestor axis of the very item it is producing, so its target exists
// by construction and there is nothing to dangle.
type ElementScopeParent interface{ elementScopeParent() }

// ComplexTypeScopeParent is the ElementScopeParent variant naming the containing
// NAMED Complex Type Definition (§3.3.2.3 dcl.elt.local: "If the <element>
// element information item has <complexType> as an ancestor, the Complex Type
// Definition corresponding to that item"). Name is that definition's expanded
// {name}; it is a PRESENT reference, never the absent (zero) QName —
// NewLocalScope rejects a zero Name (see there for why). An ANONYMOUS containing
// complex type is the AnonymousComplexTypeScopeParent variant instead. The field
// is read-only by convention; do not mutate it after construction.
type ComplexTypeScopeParent struct{ Name QName }

// AnonymousComplexTypeScopeParent is the ElementScopeParent variant identifying
// an ANONYMOUS containing Complex Type Definition — the same §3.3.2.3
// dcl.elt.local case ComplexTypeScopeParent covers, for a container that has no
// {name} to be referenced by (§3.4.1 ctd-context makes {name} and {context} a
// strict XOR, so an anonymous complex type carries a {context} instead).
//
// Owner is the identity carried by the anonymous container's OWN {context}
// (§3.4.1 ctd-context), whichever arm that is — not a second identity for the
// type itself. There are two such arms and both reach this field:
//
//   - an ElementDeclarationContext, when §3.4.2.1 dcl.ctd.common built the type
//     for an inline <complexType> child; Owner is then the owning ELEMENT
//     DECLARATION's minted identity;
//   - a ComplexTypeDefinitionContext, when §4.2.4 src-expredef clause 1.1 built
//     the type as the {name}-·absent· original of a redefinition; Owner is then
//     the REDEFINING COMPLEX TYPE's minted identity, and no element declaration
//     is involved at all.
//
// What holds in every case, and is the invariant to rely on, is that this field
// is the token of the OWNERSHIP EDGE reaching the container — one mint per edge,
// so two containers are never confused for one. It COINCIDES with the
// container's own {context} identity exactly where the owner reaches at most one
// anonymous type through that context, which is the ordinary case and the one
// NewElementDeclarationOwningTypes, NewComplexTypeOwningBase and
// NewAnonymousComplexTypeOwningBase check with ==. It does NOT coincide for a
// type owned by a Type Alternative (§3.12.2 declare-ta's inline arm): §3.4.2.1
// dcl.ctd.common walks past the <alternative> and makes the {context} the
// enclosing ELEMENT DECLARATION, shared with that element's own inline type and
// with every other alternative's, so each such edge carries its own mint here
// while all of them report one {context}. A chained <redefine> stacks two edges
// the same way — a redefining type owns an original which owns another — and
// mints a token for each (#585).
//
// Owner is a PRESENT identity, never the zero (unminted) ComponentID —
// NewLocalScope rejects an unminted one. The field is read-only by convention;
// do not mutate it after construction.
type AnonymousComplexTypeScopeParent struct{ Owner ComponentID }

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

// elementScopeParent marks AnonymousComplexTypeScopeParent as an
// ElementScopeParent (§3.3.1 sc_e-parent); see the ElementScopeParent doc.
func (AnonymousComplexTypeScopeParent) elementScopeParent() {}

// elementScopeParent marks ModelGroupScopeParent as an ElementScopeParent
// (§3.3.1 sc_e-parent); see the ElementScopeParent doc.
func (ModelGroupScopeParent) elementScopeParent() {}

// checkElementScopeParent rejects a variant that identifies no container, per
// arm: the two by-NAME arms need a present name, the by-IDENTITY arm a minted
// ComponentID. All three are charged to ruleEPropsCorrect clause 1, the footing
// NewLocalScope's own tableau checks stand on.
//
// The default arm asserts the sealed-sum invariant and is unreachable for any
// value an outside package can produce: elementScopeParent is unexported, so the
// three variants above are the only implementations that exist (mirroring
// resolve.go's non-exhaustive-switch assertions). A nil parent is caller-checked
// before this point, so it is not handled here.
func checkElementScopeParent(loc xsderr.Loc, parent ElementScopeParent) error {
	switch p := parent.(type) {
	case ComplexTypeScopeParent:
		if p.Name.Local == "" {
			return unnamedScopeParent(loc, parent)
		}
		return nil
	case ModelGroupScopeParent:
		if p.Name.Local == "" {
			return unnamedScopeParent(loc, parent)
		}
		return nil
	case AnonymousComplexTypeScopeParent:
		if p.Owner == (ComponentID{}) {
			return xsderr.New(ruleEPropsCorrect, loc,
				"element declaration's {scope}.{parent} identifies no container: the %T variant carries an unminted identity, but this representation identifies an anonymous containing complex type by identity token; mint one with NewComponentID (e-props-correct clause 1)", parent)
		}
		return nil
	default:
		panic("xsd: checkElementScopeParent: non-exhaustive ElementScopeParent switch")
	}
}

// unnamedScopeParent is the shared rejection of the two by-NAME
// ElementScopeParent variants carrying an absent name (STYLE T4).
func unnamedScopeParent(loc xsderr.Loc, parent ElementScopeParent) error {
	return xsderr.New(ruleEPropsCorrect, loc,
		"element declaration's {scope}.{parent} names no container: the %T variant carries an absent name, but this representation identifies the containing complex type or model group definition by name (e-props-correct clause 1)", parent)
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
// A local Scope REFERENCES its {parent} rather than embedding it, and requires
// that reference to identify something — see ElementScopeParent for why the
// reference is a discriminated QName (or, for an anonymous container, a minted
// identity) and NewLocalScope for why it may not point at nothing.
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
//   - a parent variant that identifies no container: an absent (zero) Name on
//     either by-NAME variant, or an unminted (zero) Owner on
//     AnonymousComplexTypeScopeParent. Each variant carries the one reference
//     kind its container admits, so a reference of that kind which points at
//     nothing could never be followed. See checkElementScopeParent.
//
// The second rejection is unreachable from this module's parser; it guards
// scopes built programmatically (a direct caller, a test) instead. A top-level
// <complexType> or a top-level named <group> is rejected as a grammar fault at
// the top of the function that maps it — BEFORE any content is built — when its
// name attribute is absent or empty, and an inline anonymous <complexType> is
// produced only after its owning declaration's ComponentID has been minted
// (parser's produceElement/produceLocalElement, #340), so no variant pointing at
// nothing is ever constructed.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built scope — may
// legitimately pass the zero xsderr.Loc{}.
func NewLocalScope(loc xsderr.Loc, parent ElementScopeParent) (Scope, error) {
	if parent == nil {
		return Scope{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration has {scope}.{variety} = local but an absent {scope}.{parent}, which the §3.3.1 tableau requires to be present when the variety is local (e-props-correct clause 1)")
	}
	if err := checkElementScopeParent(loc, parent); err != nil {
		return Scope{}, err
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
// STRUCTURAL holder built before resolution. {substitution group affiliations}
// is carried as a list of pre-resolution QName REFERENCES (the substitutionGroup
// names), and {type definition} as a TypeDefinitionOrRef — a by-name reference
// for the type/@type, NAMED-head substitution-group and xs:anyType tiers of
// §3.3.2.1 dcl.elt.common, the owned anonymous component itself for clause 1's
// inline <simpleType>/<complexType> child, or a reference to the OWNING HEAD for
// clause 3's anonymous-head case (SubstitutionGroupHeadTypeRef, #342, the one
// arm whose name lives in the ELEMENT symbol space rather than the type one).
// Finalize (resolve.go, #173) VALIDATES that every by-name reference resolves
// against the schema indexes (src-resolve clauses 1.1 and 1.3) and that the
// substitution-group graph is acyclic (e-props-correct clause 5), but does NOT
// rewrite the references into resolved components: the QNames are retained, and
// a consumer follows them by read-time schema.Type/schema.Element lookups. Of
// the cross-component clauses that need resolved components, clauses 2 (#463)
// and 4 (#395) are charged by finalize's later phases; clause 7 stays deferred.
//
// The whole {scope} record is carried, {parent} included (a Scope value, not a
// bare ScopeVariety): a local declaration names the Complex Type Definition or
// Model Group Definition it is scoped to, a global one names none, and the
// producer populates the reference from the ancestor axis (§3.3.2.3
// dcl.elt.local / §3.3.2.2 dcl.elt.global). {parent} is a THIRD pre-resolution
// reference alongside {type definition}'s by-name arm and {substitution group
// affiliations}, but unlike them it is producer-synthesized rather than
// schema-document-supplied, so finalize adds no src-resolve check for it — see
// ElementScopeParent.
//
// Ratchet impact: the schema lane widens whenever the producer starts mapping a
// {type definition} shape it used to decline — the inline anonymous <simpleType>
// of a local declaration (#229) and then the inline anonymous <complexType> of a
// local OR global one (#340), both of which the InlineTypeDefinition arm of the
// slot exists to hold.
//
// Construct only through NewElementDeclaration, or through
// NewElementDeclarationOwningTypes when the declaration owns the anonymous
// complex type of an inline <complexType> child, its own or an <alternative>'s;
// both reject the states e-props-correct (§3.3.6.1) clauses 1 and 3 forbid so
// they are unrepresentable (STYLE T1). ElementDeclaration is immutable after
// construction.
type ElementDeclaration struct {
	loc                           xsderr.Loc // source position; provenance, not a §3.3.1 property
	name                          QName
	typeDefinition                TypeDefinitionOrRef
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
// It also rejects the two illegal encodings of the typeDefinition slot — a
// zero-named TypeDefinitionRef and an InlineTypeDefinition that is empty or
// wraps a NAMED type — charged to xsderr.RuleComponentInvariant; see
// TypeDefinitionOrRef and checkTypeDefinitionOrRef. A nil slot is the legal
// encoding of an absent {type definition}, which a programmatically built
// declaration is in before the §3.3.2.1 defaulting tiers are applied.
//
// An InlineTypeDefinition wrapping a ComplexType is rejected here too, on the
// same footing, and in EVERY slot ownedTypeSlots enumerates — the {type
// definition} itself and the {type table}'s alternatives alike. That shape is
// §3.3.2.1 dcl.elt.common clause 1's inline <complexType> child or §3.12.2
// declare-ta's, whose anonymous type carries a {context} naming THIS declaration
// (§3.4.2.1 dcl.ctd.common), and this entry point takes no identity to check
// that back-pointer against. Build it through NewElementDeclarationOwningTypes,
// which does.
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
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the declaration's provenance. Pass the position of this
// declaration's own declaring element, never a convenient nearby one (a parent
// element's, say) — it is observable, not merely an error-charging convenience.
// A caller with no real parser position — a synthesized or programmatically
// built declaration — passes the zero xsderr.Loc{}, which reads as "unknown".
func NewElementDeclaration(loc xsderr.Loc, name QName, typeDefinition TypeDefinitionOrRef, typeTable *TypeTable, scope Scope, valueConstraint *ValueConstraint, nillable bool, identityConstraints []IdentityConstraint, substitutionGroupAffiliations []QName, substitutionGroupExclusions []DerivationMethod, abstract bool, disallowedSubstitutions []DerivationMethod, annotations []Annotation) (ElementDeclaration, error) {
	for _, slot := range ownedTypeSlots(typeDefinition, typeTable) {
		if _, isComplex := ownedComplexType(slot.ref); isComplex {
			return ElementDeclaration{}, xsderr.New(xsderr.RuleComponentInvariant, loc,
				"element declaration %s %s is an InlineTypeDefinition wrapping a ComplexType, but a declaration that OWNS an anonymous complex type must be built through NewElementDeclarationOwningTypes, the only entry point that can check the type's {context} back-pointer (§3.4.2.1 dcl.ctd.common) against the declaration's own identity", name, slot.property)
		}
	}
	return newElementDeclaration(loc, name, typeDefinition, typeTable, scope, valueConstraint, nillable, identityConstraints, substitutionGroupAffiliations, substitutionGroupExclusions, abstract, disallowedSubstitutions, annotations)
}

// NewElementDeclarationOwningTypes builds an ElementDeclaration that OWNS one or
// more ANONYMOUS Complex Type Definitions: its own inline <complexType> child
// (§3.3.2.1 dcl.elt.common clause 1), the inline <complexType> child of any
// <alternative> in its {type table} (§3.12.2 declare-ta's second arm), or both.
// It is NewElementDeclaration's parameter list preceded by the declaration's own
// minted identity; every other parameter, check, copy, and rejection is
// identical, because both entry points run one shared core.
//
// It is the ONLY construction path for a declaration owning an anonymous complex
// type — NewElementDeclaration rejects that shape in every slot ownedTypeSlots
// enumerates — and it exists to make ONE state unrepresentable that no shape
// check could reach: an inline anonymous complex type whose {context} names some
// OTHER component. §3.4.2.1 dcl.ctd.common makes that {context} "the Element
// Declaration corresponding to the nearest <element> information item among the
// ancestor element information items", which for an inline child of this
// <element> OR of one of its <alternative> children is this very declaration
// (§3.16.2.1 map.std.common case 2.4 says the same of a simple type, naming
// <alternative> explicitly), so the identity the caller minted for it and the
// identity each type's own {context} carries must be the SAME ComponentID
// (compared with ==; see ComponentID for why reflect.DeepEqual cannot see it).
// It adds three rejections, all charged to xsderr.RuleComponentInvariant because
// a ComponentID is this package's representation rather than a spec-visible name
// — the footing checkComplexTypeContext already stands on:
//
//   - an unminted (zero) id: there would be nothing to compare the {context}
//     against, and the anonymous type's {scope}.{parent} references
//     (AnonymousComplexTypeScopeParent) would identify nothing either;
//   - a {context} that is an ElementDeclarationContext naming a DIFFERENT
//     identity;
//   - a {context} that is a ComplexTypeDefinitionContext: §3.4.2.1 has exactly
//     one case and it yields an Element Declaration, so a <complexType> reached
//     from an <element> is never contexted in a complex type.
//
// One id serves EVERY owned type, because all of them share one {context}: the
// owning declaration. What is NOT shared is the token their nested local
// declarations report as {scope}.{parent}, which is one mint per ownership EDGE
// — see AnonymousComplexTypeScopeParent.
//
// A definition that is NAMED (its {context} therefore ·absent· per §3.4.1's XOR)
// is rejected by the shared core's checkTypeDefinitionOrRef, which admits only
// an anonymous type in the InlineTypeDefinition arm.
//
// id is NOT stored on the built declaration. Its whole role is this
// construction-time comparison, and a field written but never read is dead state
// (STYLE D3); the landing that adds an ID→component resolver adds the field
// together with its reader (see ComponentID).
func NewElementDeclarationOwningTypes(loc xsderr.Loc, id ComponentID, name QName, typeDefinition TypeDefinitionOrRef, typeTable *TypeTable, scope Scope, valueConstraint *ValueConstraint, nillable bool, identityConstraints []IdentityConstraint, substitutionGroupAffiliations []QName, substitutionGroupExclusions []DerivationMethod, abstract bool, disallowedSubstitutions []DerivationMethod, annotations []Annotation) (ElementDeclaration, error) {
	if id == (ComponentID{}) {
		return ElementDeclaration{}, xsderr.New(xsderr.RuleComponentInvariant, loc,
			"element declaration %s owns an anonymous complex type but carries an unminted identity, which that type's {context} back-pointer could not name; mint one with NewComponentID", name)
	}
	if err := checkOwnedTypeContexts(loc, id, name, typeDefinition, typeTable); err != nil {
		return ElementDeclaration{}, err
	}
	return newElementDeclaration(loc, name, typeDefinition, typeTable, scope, valueConstraint, nillable, identityConstraints, substitutionGroupAffiliations, substitutionGroupExclusions, abstract, disallowedSubstitutions, annotations)
}

// ownedTypeSlot is one TypeDefinitionOrRef slot an element declaration may own
// an anonymous type through, paired with the §3 property label that names it in
// a rejection message.
type ownedTypeSlot struct {
	property string
	ref      TypeDefinitionOrRef
}

// ownedTypeSlots enumerates every slot of an element declaration that can hold
// an anonymous type the declaration OWNS, in a fixed order: its own {type
// definition}, then each {type table}.{alternatives} member's, then the {type
// table}.{default type definition}'s. It has TWO readers that must agree
// exactly — checkOwnedTypeContexts, which checks each owned type's {context}
// back-pointer, and NewElementDeclaration's symmetry rejection, which refuses an
// owned type reached through any of them — so it is one producer rather than two
// hand-written enumerations that would drift apart (STYLE T4).
//
// The finalize folds enumerate the SAME three slots in the same order and must
// stay in step with this list, in ownedTypeFold.elementDeclaration and
// ownedTypeFold.typeTable (ownedtypefold.go). They cannot take this producer:
// they WRITE each slot back with the folded type, which a flattened list of refs
// and labels cannot express.
//
// The {default type definition} slot is enumerated UNCONDITIONALLY, and where
// §3.3.2.1's case 2 synthesized it from the declaring element's own {type
// definition} that means the same component is visited twice. Both readers are
// idempotent over one component — a {context} check and a shape rejection each
// answer the same way for the second visit as for the first — and the two
// visits are indistinguishable anyway: a ComplexType is a value holding slices,
// so two slots holding one component cannot be told from two slots holding
// structurally equal copies. Skipping the slot is not an option either: a
// TRAILING untested <alternative> feeds this slot and NO other, so its owned
// type appears here alone.
func ownedTypeSlots(typeDefinition TypeDefinitionOrRef, typeTable *TypeTable) []ownedTypeSlot {
	slots := []ownedTypeSlot{{property: "{type definition}", ref: typeDefinition}}
	if typeTable == nil {
		return slots
	}
	for i, alt := range typeTable.alternatives {
		slots = append(slots, ownedTypeSlot{
			property: "{type table}.{alternatives}[" + strconv.Itoa(i) + "].{type definition}",
			ref:      alt.TypeDefinition(),
		})
	}
	return append(slots, ownedTypeSlot{
		property: "{type table}.{default type definition}.{type definition}",
		ref:      typeTable.defaultTypeDefinition.TypeDefinition(),
	})
}

// ownedComplexType reports the ANONYMOUS complex type a slot owns, and false for
// every other shape of slot — a by-name reference, a head-inherited reference, an
// absent slot, and an owned SIMPLE type, whose own {context} (§3.16.1
// std-context) this package does not model.
func ownedComplexType(ref TypeDefinitionOrRef) (ComplexType, bool) {
	inline, owns := ref.(InlineTypeDefinition)
	if !owns {
		return ComplexType{}, false
	}
	ct, isComplex := inline.Definition.(ComplexType)
	return ct, isComplex
}

// checkOwnedTypeContexts charges checkOwnedTypeContext over every slot
// ownedTypeSlots enumerates, reporting the first offender in that order.
func checkOwnedTypeContexts(loc xsderr.Loc, id ComponentID, name QName, typeDefinition TypeDefinitionOrRef, typeTable *TypeTable) error {
	for _, slot := range ownedTypeSlots(typeDefinition, typeTable) {
		if err := checkOwnedTypeContext(loc, id, name, slot.property, slot.ref); err != nil {
			return err
		}
	}
	return nil
}

// checkOwnedTypeContext rejects an owned anonymous complex type whose {context}
// (§3.4.1 ctd-context) does not name the owning declaration's identity id. A slot
// owning no complex type passes, and so does an ABSENT {context}: that means the
// definition is NAMED, which the shared core's checkTypeDefinitionOrRef rejects
// with the message that fits it rather than this one (STYLE T4). property names
// the offending slot for the message.
//
// The switch is exhaustive over ComplexTypeContext's sealed sum; the default arm
// asserts the invariant and is unreachable for any value an outside package can
// produce, since complexTypeContext is unexported.
func checkOwnedTypeContext(loc xsderr.Loc, id ComponentID, name QName, property string, ref TypeDefinitionOrRef) error {
	definition, owns := ownedComplexType(ref)
	if !owns {
		return nil
	}
	context, present := definition.Context()
	if !present {
		return nil
	}
	switch c := context.(type) {
	case ElementDeclarationContext:
		if c.ID() != id {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"element declaration %s owns through its %s an anonymous complex type whose {context} names a DIFFERENT element declaration, but §3.4.2.1 dcl.ctd.common makes an inline <complexType>'s {context} the nearest enclosing declaration; mint one identity per owning declaration and pass it to every type it owns", name, property)
		}
		return nil
	case ComplexTypeDefinitionContext:
		return xsderr.New(xsderr.RuleComponentInvariant, loc,
			"element declaration %s owns through its %s an anonymous complex type whose {context} is a ComplexTypeDefinitionContext, but §3.4.2.1 dcl.ctd.common has exactly one case and it yields an Element Declaration, so a <complexType> reached from an <element> is never contexted in a complex type definition", name, property)
	default:
		panic("xsd: checkOwnedTypeContext: non-exhaustive ComplexTypeContext switch")
	}
}

// newElementDeclaration is the shared core of NewElementDeclaration and
// NewElementDeclarationOwningTypes: every check and copy that does not concern
// the ownership of an anonymous complex type lives here exactly once (STYLE T4).
//
// PRECONDITION, enforced by its TWO CALLERS and not by itself: an
// InlineTypeDefinition wrapping a ComplexType, in the typeDefinition slot or in
// any {type table} slot ownedTypeSlots enumerates, arrived through
// NewElementDeclarationOwningTypes, so its {context} has been checked against the
// owner's identity. This layer cannot express that check — it takes no identity
// — and does not attempt one. Any third caller added here must re-establish it.
func newElementDeclaration(loc xsderr.Loc, name QName, typeDefinition TypeDefinitionOrRef, typeTable *TypeTable, scope Scope, valueConstraint *ValueConstraint, nillable bool, identityConstraints []IdentityConstraint, substitutionGroupAffiliations []QName, substitutionGroupExclusions []DerivationMethod, abstract bool, disallowedSubstitutions []DerivationMethod, annotations []Annotation) (ElementDeclaration, error) {
	if name.Local == "" {
		return ElementDeclaration{}, xsderr.New(ruleEPropsCorrect, loc,
			"element declaration has an absent {name}, but the §3.3.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (e-props-correct clause 1)")
	}
	if err := checkTypeDefinitionOrRef(loc, typeDefinition, elementTypeSlot, "element declaration "+name.String()); err != nil {
		return ElementDeclaration{}, err
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
		loc:            loc,
		name:           name,
		typeDefinition: typeDefinition,
		scope:          scope,
		nillable:       nillable,
		abstract:       abstract,
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

// Loc reports the source position of the declaring element — provenance, not a
// §3.3.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown.
func (e ElementDeclaration) Loc() xsderr.Loc {
	return e.loc
}

// TypeDefinition returns the {type definition} property (Required) as the
// TypeDefinitionOrRef sealed sum: a TypeDefinitionRef naming a top-level type
// (§3.3.2.1 dcl.elt.common clauses 2 and 4, and clause 3 with a NAMED-typed
// head), an InlineTypeDefinition owning the anonymous type of an inline
// <simpleType>/<complexType> child (clause 1), or a SubstitutionGroupHeadTypeRef
// naming the head that OWNS the anonymous type this declaration inherits (clause
// 3, anonymous-head case). It is nil only for a declaration built with an absent
// {type definition}.
//
// Neither reference arm is resolved into a component here. Finalize (#173)
// validates that a TypeDefinitionRef's name resolves to a type definition
// (src-resolve clause 1.1) but adds no resolved-component accessor: the QName is
// retained, and a consumer obtains the component by a read-time
// schema.Type(name) lookup. A SubstitutionGroupHeadTypeRef is followed instead
// through schema.Element(head) — a different symbol space — and then one read of
// that head's own {type definition}; a dangling head is an ·absent· member under
// §5.3 and is not a finalize failure. The inline arm needs no lookup at all — it
// carries the component.
func (e ElementDeclaration) TypeDefinition() TypeDefinitionOrRef {
	return e.typeDefinition
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
// Declaration components (§3.3.1). Finalize validates that the affiliation graph
// is acyclic (e-props-correct clause 5, #173) and that each name it CAN resolve
// heads a type this declaration's own is ·validly substitutable· for (clause 4,
// #395), but adds no resolved-component accessor: the QNames are retained,
// followed by read-time schema.Element lookups. A name that resolves to no
// declaration at all is not rejected — it is an ·absent· member under §5.3
// (Missing Sub-components), which resolveElementDecl records as the one
// reference slot deliberately exempt from src-resolve.
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
