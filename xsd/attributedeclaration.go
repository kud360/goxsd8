package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleAPropsCorrect is Attribute Declaration Properties Correct (Structures
// §3.2.6.1, id="a-props-correct"): an attribute declaration's properties must
// match the §3.2.1 property tableau. This file enforces the clauses that are
// cheap, purely structural, and cross-reference-free at this layer, citing the
// specific clause number in each message (the rule ID is not sub-anchored per
// clause, matching elementdeclaration.go's single-rule-const convention):
//
//   - clause 1 (tableau shape): {name} is present, and a present {value
//     constraint} carries a legal {variety} (default or fixed). The {scope}
//     record's own shape is not checked here because it is unrepresentable
//     rather than checkable — see AttributeScope, whose {variety} is derived
//     from {parent}'s presence. What IS checkable there, a {parent} that
//     identifies no container, is charged this same rule by
//     NewAttributeLocalScope and checkAttributeScopeParent.
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

// AttributeScopeParent is the sealed sum of the ways an attribute declaration's
// {scope}.{parent} identifies its container (Structures §3.2.1, Scope record
// id="sc_a", {parent}: "Either a Complex Type Definition or a Attribute Group
// Definition"). The unexported attributeScopeParent marker method seals it
// (STYLE T2/T7), mirroring elementdeclaration.go's ElementScopeParent, so
// consumers exhaustively switch these variants and no fourth is representable.
//
// There are THREE arms for the spec's TWO component kinds, and the extra one is
// a REPRESENTATION split inside one kind, not a third kind. A Complex Type
// Definition container may be named (a top-level <complexType>) or ANONYMOUS (an
// inline <complexType> child of an <element>, §3.4.1 ctd-context), and the two
// are not identifiable the same way: a named one is followable by QName, an
// anonymous one has no name at all and is identified only by the ComponentID
// minted for the inline construct it belongs to. Encoding that as an optional
// second field on one arm would make "named AND anonymous" and "neither"
// representable; a separate arm makes both unrepresentable (STYLE T1). An
// Attribute Group Definition needs no such split: §3.6.1 types its {name} as a
// Required xs:NCName, and §3.6.2.1 makes a NESTED <attributeGroup> correspond to
// no component at all, so the only AGD that can ever be a {parent} is a
// top-level named one.
//
//   - AttributeComplexTypeScopeParent — a NAMED containing complex type
//     definition;
//   - AttributeAnonymousComplexTypeScopeParent — an ANONYMOUS containing complex
//     type definition, by owner identity;
//   - AttributeGroupScopeParent — a containing attribute group definition.
//
// It is a DIFFERENT type from elementdeclaration.go's ElementScopeParent, and
// deliberately so: sc_a's alternation is Complex Type Definition or Attribute
// Group Definition where §3.3.1 sc_e's is Complex Type Definition or Model Group
// Definition, and merging the two would make "an element scoped to an attribute
// group" — and "an attribute scoped to a model group" — representable. The arms
// carry the Attribute prefix for the same reason.
//
// Every variant carries a REFERENCE to the container, not the container
// component itself. Embedding the value is impossible here: construction is
// bottom-up (a complex type's or attribute group's {attribute uses}, which own
// the local declaration, is built before the container is), so the parent does
// not exist when the child is made. A bare QName without the variant's kind
// discriminant would not do either: Complex Type Definitions and Attribute Group
// Definitions occupy INDEPENDENT symbol spaces on Schema (§3.17.1 {type
// definitions} versus {attribute group definitions}), so a CTD and an AGD may
// share one expanded name. The kind therefore lives in the variant type, and a
// consumer follows a by-NAME reference with a read-time lookup in the index that
// kind selects — the same pre-resolution-reference convention as
// TypeDefinitionRef, which Schema.Type serves. An
// AttributeComplexTypeScopeParent is followable today; an
// AttributeGroupScopeParent waits on the Schema.AttributeGroup(QName) accessor
// this package does not export yet (STYLE T5 — no consumer justifies one), and
// an AttributeAnonymousComplexTypeScopeParent on the ID→component resolver
// ComponentID's doc describes, which no consumer justifies yet either.
//
// Unlike those reference slots, this one is NOT checked by finalize: resolve.go
// adds no src-resolve (§3.17.6.2) clause for it. src-resolve governs QNames
// supplied by a schema document; {scope}.{parent} is synthesized by the producer
// from the ancestor axis of the very item it is producing (§3.2.2.2
// dcl.att.local), so its target exists by construction and there is nothing to
// dangle.
type AttributeScopeParent interface{ attributeScopeParent() }

// AttributeComplexTypeScopeParent is the AttributeScopeParent variant naming the
// containing NAMED Complex Type Definition (§3.2.2.2 dcl.att.local: "If the
// <attribute> element information item has <complexType> as an ancestor, the
// Complex Type Definition corresponding to that item"). Name is that
// definition's expanded {name}; it is a PRESENT reference, never the absent
// (zero) QName — NewAttributeLocalScope rejects a zero Name (see there for why).
// An ANONYMOUS containing complex type is the
// AttributeAnonymousComplexTypeScopeParent variant instead. The field is
// read-only by convention; do not mutate it after construction.
type AttributeComplexTypeScopeParent struct{ Name QName }

// AttributeAnonymousComplexTypeScopeParent is the AttributeScopeParent variant
// identifying an ANONYMOUS containing Complex Type Definition — the same §3.2.2.2
// dcl.att.local case AttributeComplexTypeScopeParent covers, for a container that
// has no {name} to be referenced by (§3.4.1 ctd-context makes {name} and
// {context} a strict XOR, so an anonymous complex type carries a {context}
// instead).
//
// Owner is the identity carried by the anonymous container's OWN {context}
// (§3.4.1 ctd-context), whichever arm that is — an ElementDeclarationContext for
// an inline <complexType> child (§3.4.2.1 dcl.ctd.common), or a
// ComplexTypeDefinitionContext for the {name}-·absent· original of a redefinition
// (§4.2.4 src-expredef clause 1.1), where no element declaration is involved at
// all. It is not a second identity for the type itself. The invariant that holds
// in both cases is the 1:1 pairing: the container's {context} identity and this
// field hold the SAME ComponentID, one mint per pairing, one fact with one
// encoding (STYLE D3), exactly as AnonymousComplexTypeScopeParent carries it on
// the element side.
//
// Owner is a PRESENT identity, never the zero (unminted) ComponentID —
// NewAttributeLocalScope rejects an unminted one, and the parser cannot produce
// one: it mints the owning identity BEFORE produceComplexType is called on the
// anonymous type, so the identity every attribute inside it is scoped to is
// already minted when the scope is built. The field is read-only by convention;
// do not mutate it after construction.
type AttributeAnonymousComplexTypeScopeParent struct{ Owner ComponentID }

// AttributeGroupScopeParent is the AttributeScopeParent variant naming the
// containing Attribute Group Definition (§3.2.2.2 dcl.att.local: "otherwise (the
// <attribute> element information item is within an <attributeGroup> element
// information item), the Attribute Group Definition corresponding to that item").
// Name is that definition's expanded {name}, which §3.6.1 types as a Required
// xs:NCName, so it is always present; NewAttributeLocalScope rejects a zero Name.
//
// The parent an <attribute> child of a top-level <attributeGroup> reports is that
// group INVARIANTLY: the attribute has no <complexType> ancestor, so no complex
// type that happens to reference the group can become its {parent}, however many
// do. The field is read-only by convention; do not mutate it after construction.
type AttributeGroupScopeParent struct{ Name QName }

// attributeScopeParent marks AttributeComplexTypeScopeParent as an
// AttributeScopeParent (§3.2.1 sc_a-parent); see the AttributeScopeParent doc.
func (AttributeComplexTypeScopeParent) attributeScopeParent() {}

// attributeScopeParent marks AttributeAnonymousComplexTypeScopeParent as an
// AttributeScopeParent (§3.2.1 sc_a-parent); see the AttributeScopeParent doc.
func (AttributeAnonymousComplexTypeScopeParent) attributeScopeParent() {}

// attributeScopeParent marks AttributeGroupScopeParent as an
// AttributeScopeParent (§3.2.1 sc_a-parent); see the AttributeScopeParent doc.
func (AttributeGroupScopeParent) attributeScopeParent() {}

// checkAttributeScopeParent rejects a variant that identifies no container, per
// arm: the two by-NAME arms need a present name, the by-IDENTITY arm a minted
// ComponentID. All three are charged to ruleAPropsCorrect clause 1, the footing
// NewAttributeLocalScope's own tableau checks stand on.
//
// It deliberately does NOT share elementdeclaration.go's unnamedScopeParent
// helper: that one hardcodes e-props-correct and the words "element
// declaration", so reusing it here would put a wrong rule ID and a wrong
// component kind into a conformance-grade verdict against an attribute — worse
// than the small duplication STYLE T4 discourages.
//
// The default arm asserts the sealed-sum invariant and is unreachable for any
// value an outside package can produce: attributeScopeParent is unexported, so
// the three variants above are the only implementations that exist. A nil parent
// is caller-checked before this point, so it is not handled here.
func checkAttributeScopeParent(loc xsderr.Loc, parent AttributeScopeParent) error {
	switch p := parent.(type) {
	case AttributeComplexTypeScopeParent:
		if p.Name.Local == "" {
			return unnamedAttributeScopeParent(loc, parent)
		}
		return nil
	case AttributeGroupScopeParent:
		if p.Name.Local == "" {
			return unnamedAttributeScopeParent(loc, parent)
		}
		return nil
	case AttributeAnonymousComplexTypeScopeParent:
		if p.Owner == (ComponentID{}) {
			return xsderr.New(ruleAPropsCorrect, loc,
				"attribute declaration's {scope}.{parent} identifies no container: the %T variant carries an unminted identity, but this representation identifies an anonymous containing complex type by identity token; mint one with NewComponentID (a-props-correct clause 1)", parent)
		}
		return nil
	default:
		panic("xsd: checkAttributeScopeParent: non-exhaustive AttributeScopeParent switch")
	}
}

// unnamedAttributeScopeParent is the shared rejection of the two by-NAME
// AttributeScopeParent variants carrying an absent name (STYLE T4).
func unnamedAttributeScopeParent(loc xsderr.Loc, parent AttributeScopeParent) error {
	return xsderr.New(ruleAPropsCorrect, loc,
		"attribute declaration's {scope}.{parent} names no container: the %T variant carries an absent name, but this representation identifies the containing complex type or attribute group definition by name (a-props-correct clause 1)", parent)
}

// AttributeScope is the {scope} property record of an attribute declaration
// (Structures §3.2.1, id="sc_a"): a Required {variety} in {global, local} and a
// {parent} that is "Required if {variety} is local, otherwise must be ·absent·".
//
// The two properties are one fact, so only {parent} is stored and Variety() is
// DERIVED from its presence (STYLE D3), exactly as elementdeclaration.go's Scope
// does for sc_e. The tableau's correlation therefore needs no runtime check
// anywhere: a global scope carrying a parent, and a local scope missing one, are
// both unrepresentable.
//
// The zero AttributeScope is the global scope, matching NewAttributeGlobalScope;
// the two constructors are the only way to obtain a local one, since parent is
// unexported.
//
// It is a separate type from the element side's Scope for the reason
// AttributeScopeParent gives: the two records' {parent} alternations differ in
// their second member, and one shared record would make an element scoped to an
// attribute group representable.
//
// A local AttributeScope REFERENCES its {parent} rather than embedding it, and
// requires that reference to identify something — see AttributeScopeParent for
// why the reference is a discriminated QName (or, for an anonymous container, a
// minted identity) and NewAttributeLocalScope for why it may not point at
// nothing.
type AttributeScope struct {
	parent AttributeScopeParent // nil ⇔ {variety} = global
}

// NewAttributeGlobalScope returns the global {scope} of a top-level attribute
// declaration (§3.2.2.1 dcl.att.global: {variety} global, {parent} ·absent·). It
// cannot fail: the record has no other property to get wrong. The four builtin
// xsi: declarations of §3.2.7 are global on this same footing.
func NewAttributeGlobalScope() AttributeScope {
	return AttributeScope{}
}

// NewAttributeLocalScope returns the local {scope} of an attribute declaration
// nested in a <complexType> or a top-level <attributeGroup> (§3.2.2.2
// dcl.att.local), naming the container it is scoped to. It rejects the two states
// the §3.2.1 tableau and this representation forbid, citing a-props-correct
// clause 1 (§3.2.6.1):
//
//   - a nil parent: {parent} is Required when {variety} is local, and a local
//     scope with no container to be available within contradicts §3.2.1's
//     availability prose ("A is available for use only within ... A.{scope}.
//     {parent}").
//   - a parent variant that identifies no container: an absent (zero) Name on
//     either by-NAME variant, or an unminted (zero) Owner on
//     AttributeAnonymousComplexTypeScopeParent. Each variant carries the one
//     reference kind its container admits, so a reference of that kind which
//     points at nothing could never be followed. See checkAttributeScopeParent.
//
// The second rejection is unreachable from this module's parser; it guards scopes
// built programmatically (a direct caller, a test) instead. A top-level
// <complexType> or <attributeGroup> is rejected as a grammar fault at the top of
// the function that maps it — BEFORE any content is built — when its name
// attribute is absent or empty, and an inline anonymous <complexType> is produced
// only after its owning declaration's ComponentID has been minted.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built scope — may
// legitimately pass the zero xsderr.Loc{}.
func NewAttributeLocalScope(loc xsderr.Loc, parent AttributeScopeParent) (AttributeScope, error) {
	if parent == nil {
		return AttributeScope{}, xsderr.New(ruleAPropsCorrect, loc,
			"attribute declaration has {scope}.{variety} = local but an absent {scope}.{parent}, which the §3.2.1 tableau requires to be present when the variety is local (a-props-correct clause 1)")
	}
	if err := checkAttributeScopeParent(loc, parent); err != nil {
		return AttributeScope{}, err
	}
	return AttributeScope{parent: parent}, nil
}

// Variety returns the {variety} property (§3.2.1 sc_a-variety): ScopeLocal when a
// {parent} is present, ScopeGlobal otherwise. The token is derived from
// {parent}'s presence, never stored (STYLE D3).
func (s AttributeScope) Variety() ScopeVariety {
	if s.parent == nil {
		return ScopeGlobal
	}
	return ScopeLocal
}

// Parent returns the {parent} property (§3.2.1 sc_a-parent) as a discriminated
// QName reference to the containing Complex Type Definition or Attribute Group
// Definition; the second result is false when it is absent (a global scope), in
// which case the first result is nil.
//
// This is NOT the resolved container component: see AttributeScopeParent for why
// the reference is carried by name and how a consumer follows it.
func (s AttributeScope) Parent() (AttributeScopeParent, bool) {
	return s.parent, s.parent != nil
}

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
// The whole {scope} record is carried, {parent} included (an AttributeScope
// value, not a bare ScopeVariety): a local declaration names the Complex Type
// Definition or Attribute Group Definition it is scoped to, a global one names
// none, and the producer populates the reference from the ancestor axis
// (§3.2.2.2 dcl.att.local / §3.2.2.1 dcl.att.global). {parent} is a SECOND
// pre-resolution reference alongside {type definition}'s by-name arm, but unlike
// it, it is producer-synthesized rather than schema-document-supplied, so
// finalize adds no src-resolve check for it — see AttributeScopeParent. Only the
// {variety} closed set is shared with element declarations (ScopeVariety, per
// closedsets.go); the sc_a record itself is this file's own AttributeScope,
// because sc_a's and sc_e's {parent} alternations differ in their second member.
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
	scope              AttributeScope
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
//   - a present valueConstraint must carry a legal {variety} (ValueDefault or
//     ValueFixed) — this catches a caller passing the zero ValueConstraint{}
//     instead of a value built through NewValueConstraint.
//
// The {scope} record needs no clause-1 check of its own: an AttributeScope
// derives its {variety} from its {parent}'s presence, so the tableau's "Required
// if local, otherwise absent" correlation cannot be violated by any value that
// can be passed here, and the one thing that IS checkable — a {parent}
// identifying no container — was charged by NewAttributeLocalScope before this
// value existed.
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
func NewAttributeDeclaration(loc xsderr.Loc, name QName, typeDefinition TypeDefinitionOrRef, scope AttributeScope, valueConstraint *ValueConstraint, inheritable bool, annotations []Annotation) (AttributeDeclaration, error) {
	if name.Local == "" {
		return AttributeDeclaration{}, xsderr.New(ruleAPropsCorrect, loc,
			"attribute declaration has an absent {name}, but the §3.2.1 tableau types it as a Required xs:NCName, whose value space excludes the empty string (a-props-correct clause 1)")
	}
	if err := checkTypeDefinitionOrRef(loc, typeDefinition, "attribute declaration "+name.String()+" {type definition}"); err != nil {
		return AttributeDeclaration{}, err
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
		scope:          scope,
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

// Scope returns the whole {scope} property record (§3.2.1 sc_a): its {variety}
// and, for a local declaration, the {parent} naming the Complex Type Definition
// or Attribute Group Definition it is scoped to. Mirrors
// ElementDeclaration.Scope.
func (a AttributeDeclaration) Scope() AttributeScope {
	return a.scope
}

// ScopeVariety returns the {scope}.{variety} property (§3.2.1 sc_a-variety), the
// one member callers that do not care which container a local declaration lives
// in need. It is a delegate to Scope().Variety(), which derives the token from
// {parent}'s presence; Scope reports the container itself.
func (a AttributeDeclaration) ScopeVariety() ScopeVariety {
	return a.scope.Variety()
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
