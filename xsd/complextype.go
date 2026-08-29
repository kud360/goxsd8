package xsd

import "github.com/kud360/goxsd8/xsderr"

// ruleCTPropsCorrect is Complex Type Definition Properties Correct (Structures
// §3.4.6.1, id="ct-props-correct"): a complex type definition's properties must
// match the §3.4.1 property tableau. This file enforces ONLY the cheap,
// purely structural, cross-reference-free part of clause 1 (the property
// tableau shape) — the value spaces of the scalar/enum properties and the
// presence of the Required {content type} sub-parts — citing "clause 1" in each
// message (the rule ID is not sub-anchored per clause, matching
// elementdeclaration.go's single-rule-const convention):
//
//   - {derivation method} is one of {extension, restriction}; {final} and
//     {prohibited substitutions} are each a subset of {extension, restriction}.
//   - {content type} is present (a non-nil ContentType sum variant); a
//     SimpleContent carries a non-nil {simple type definition}; an
//     ElementContent carries a {particle} with a present {term}.
//   - {name} and {context} are a strict XOR ("{context} Required if {name} is
//     ·absent·, otherwise must be ·absent·"). This one is enforced by the
//     CONSTRUCTION-PATH PARTITION rather than by a check — NewComplexType takes
//     a name and no context, NewAnonymousComplexType the reverse — so the only
//     residue is each entry point rejecting its own half's absence (an empty
//     {name} local part, a nil {context}). See ComplexType's doc.
//
// The substantive, cross-component clauses are NOT enforced here — this
// constructor is deliberately not the full property-correctness check:
//
//   - clause 1's resolved-component parts and clause 2 ({base type definition} a
//     simple type forces {derivation method} = extension) need the {base type
//     definition} RESOLVED, which this constructor does not do. Clause 2 IS
//     enforced, but at finalize (Phase D, complexderivation.go's
//     checkSimpleBaseIsExtension, #262). Clause 3 (no circular {base type
//     definition} chain except xs:anyType) is likewise enforced at finalize
//     (Phase B, resolve.go's checkComplexBaseAcyclic, #173) — it needs the whole
//     base graph, which only exists once the schema set is assembled;
//   - clause 4 (no two {attribute uses} share an {attribute declaration}
//     expanded name) needs the Ref variant of {attribute declaration} resolved,
//     so it too is enforced at finalize (Phase D, checkAttributeUseNamesUnique,
//     #262) rather than at shape time (unlike AttributeGroupDefinition, whose
//     own ag-props-correct clause 2 is enforced at shape time over uses it
//     already owns);
//   - clause 5 ({content type}.{open content} non-absent ⇒ {variety} is
//     element-only or mixed) is satisfied BY CONSTRUCTION and gets no runtime
//     check anywhere, at shape time or at finalize: {open content} is a field
//     only of ElementContent, whose Variety() is a total function over one bool
//     returning element-only or mixed and nothing else, so the forbidden state is
//     unrepresentable and a test for it would be dead code.
//
// Of the derivation-validity rules, derivation-ok-restriction (§3.4.6.3) IS
// enforced, at finalize (Phase D, complexderivation.go, #262), clause 2.4.2's
// cos-content-act-restrict (§3.4.6.4) delegate included (contentrestricts.go,
// #263) — all five clauses, clause 5's {assertions} prefix included (#346); the
// one latitude left there is the content models §3.4.6.3's own all-group
// leniency licenses accepting provisionally. cos-ct-extends (§3.4.6.2) is
// enforced too, at the same phase (complexextension.go, #264), clause
// 1.4.3.2.2.2's cos-particle-extend (§3.9.6.2) delegate included — all seven
// clauses of case 1, since the three §3.4.2 base folds they read are done
// ({attribute uses} #401, {attribute wildcard} #265, {assertions} #346).
// Clause 1.5 is decided for BOTH chain shapes as of #392: the pure-extension
// chain by the identity re-ordering, and a chain mixing extension and
// restriction steps by synthesizing the §3.4.6.2 Note's collapsed intermediate
// (collapsedintermediate.go) and running derivation-ok-restriction against it.
// None of them is touched HERE — they are cross-component finalize-phase
// concerns, not tableau shape.
const ruleCTPropsCorrect xsderr.Rule = "ct-props-correct"

// ContentType is the sealed sum of the four Content Type varieties of a Complex
// Type Definition (Structures §3.4.1 "Content Type" property record, id="ct").
// The spec's {variety} property is closed to exactly {empty, simple,
// element-only, mixed}, so the set of variant shapes is closed. The unexported
// contentType marker method seals it (STYLE T2) — consumers exhaustively
// switch these variants and no further one is representable — mirroring
// term.go's Term sealed sum.
//
// The four spec varieties collapse into THREE variant types because
// element-only and mixed share an identical property shape (a Required
// {particle} plus an Optional {open content}); they are one struct
// (ElementContent) distinguished only by its Mixed bool, and Variety() derives
// the element-only/mixed token from that bool rather than storing it (STYLE D3,
// one fact one encoding).
//
// Variety reports the {variety} property (§3.4.1) without a stored field: each
// variant answers it from its own type (and, for ElementContent, its Mixed
// bool). It is part of the sealed capability so a consumer can read the variety
// without a type switch.
type ContentType interface {
	contentType()
	// Variety returns the {variety} property (§3.4.1): the empty/simple/
	// element-only/mixed token this Content Type denotes.
	Variety() ContentTypeVariety
}

// EmptyContent is the {variety} = empty Content Type (§3.4.1): a complex type
// whose {content type} carries no {particle} and no {simple type definition}.
// It has no properties of its own.
type EmptyContent struct{}

// contentType marks EmptyContent as a ContentType (§3.4.1); see the ContentType
// doc.
func (EmptyContent) contentType() {}

// Variety returns ContentEmpty (§3.4.1).
func (EmptyContent) Variety() ContentTypeVariety { return ContentEmpty }

// SimpleContent is the {variety} = simple Content Type (§3.4.1): a complex type
// whose content is text validated by a Simple Type Definition. SimpleType is the
// Required {simple type definition} property (§3.4.1 ct-simple_type_definition);
// NewComplexType rejects a nil SimpleType (ct-props-correct clause 1). The field
// is read-only by convention; do not mutate it after construction.
type SimpleContent struct{ SimpleType *SimpleType }

// contentType marks SimpleContent as a ContentType (§3.4.1); see the
// ContentType doc.
func (SimpleContent) contentType() {}

// Variety returns ContentSimple (§3.4.1).
func (SimpleContent) Variety() ContentTypeVariety { return ContentSimple }

// ElementContent is the {variety} = element-only or mixed Content Type
// (§3.4.1), collapsed into one struct because the two varieties share an
// identical property shape: a Required {particle} and an Optional {open
// content}. Mixed is the single distinguishing fact — false for element-only,
// true for mixed — and Variety() derives the token from it (STYLE D3), so the
// variety is never stored twice.
//
// Particle is the Required {particle} property (§3.4.1 ct-particle);
// NewComplexType rejects an ElementContent whose Particle has an absent {term}
// (a zero Particle{}), mirroring NewParticle's own nil-{term} rejection
// (p-props-correct clause 1). OpenContent is the Optional {open content}
// property (§3.4.1 ct-open_content): nil when absent, otherwise a Wildcard-
// carrying record built through NewOpenContent. The fields are read-only by
// convention; do not mutate them after construction.
type ElementContent struct {
	Mixed       bool
	Particle    Particle
	OpenContent *OpenContent
}

// contentType marks ElementContent as a ContentType (§3.4.1); see the
// ContentType doc.
func (ElementContent) contentType() {}

// Variety returns ContentMixed when Mixed is true, otherwise ContentElementOnly
// (§3.4.1). The token is derived from Mixed, never stored (STYLE D3).
func (e ElementContent) Variety() ContentTypeVariety {
	if e.Mixed {
		return ContentMixed
	}
	return ContentElementOnly
}

// OpenContent is the Open Content property record of a Content Type (Structures
// §3.4.1, id="oc"): {mode} (Required, one of interleave/suffix — closedsets.go)
// and {wildcard} (Required, a Wildcard — §3.10.1). It appears only on an
// element-only or mixed Content Type (as ElementContent.OpenContent), and its
// absence is modeled by a nil *OpenContent there, not by a "none" mode (see
// OpenContentMode's doc for why there is no "none" member).
//
// The zero value is NOT a valid OpenContent (its {mode} is the invalid zero
// OpenContentMode); construct only through NewOpenContent, which rejects an
// out-of-range {mode}, so an ill-formed record is unrepresentable (STYLE T1).
// OpenContent is immutable after construction.
type OpenContent struct {
	mode     OpenContentMode
	wildcard Wildcard
}

// NewOpenContent builds an OpenContent, rejecting a {mode} that is not one of
// OpenContentInterleave or OpenContentSuffix (the §3.4.1 {mode} value space;
// the "none" case is the ABSENT record, a nil *OpenContent, never a third
// mode). The mode rejection is charged to ct-props-correct clause 1 (§3.4.6.1),
// the §3.4.1 property tableau this record is part of.
//
// wildcard is the Required {wildcard} property; it must be a Wildcard built
// through NewWildcard (its zero value is invalid — see Wildcard's doc). The
// record trusts it, mirroring how NewParticle trusts an Occurs already
// validated by its own constructor.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built record — may
// legitimately pass the zero xsderr.Loc{}.
func NewOpenContent(loc xsderr.Loc, mode OpenContentMode, wildcard Wildcard) (OpenContent, error) {
	switch mode {
	case OpenContentInterleave, OpenContentSuffix:
	default:
		return OpenContent{}, xsderr.New(ruleCTPropsCorrect, loc,
			"open content {mode} %s is not one of interleave/suffix (ct-props-correct clause 1)", mode)
	}
	return OpenContent{mode: mode, wildcard: wildcard}, nil
}

// Mode returns the {mode} property (Required): interleave or suffix (§3.4.1).
func (o OpenContent) Mode() OpenContentMode {
	return o.mode
}

// Wildcard returns the {wildcard} property (Required): the Wildcard whose
// {namespace constraint} governs which open-content elements are admitted
// (§3.4.1).
func (o OpenContent) Wildcard() Wildcard {
	return o.wildcard
}

// ComplexTypeContext is the sealed sum of the two component kinds an anonymous
// complex type definition's {context} property may name (Structures §3.4.1,
// id="ctd-context": "Either an Element Declaration or a Complex Type
// Definition"). The unexported complexTypeContext marker method seals it (STYLE
// T2), mirroring this file's ContentType and elementdeclaration.go's
// ElementScopeParent.
//
// Each arm carries a ComponentID, not a QName: §3.4.2.1 dcl.ctd.common makes
// the target the nearest ancestor <element>, frequently a LOCAL element
// declaration, which is not name-unique — see ComponentID for why a name cannot
// identify it. ID is promoted into the sum so a consumer reads the identity
// without a type switch, exactly as ContentType promotes Variety and
// TypeDefinition promotes Name and Loc. The arms ALSO keep an exported field,
// for the literal-construction symmetry ElementScopeParent's variants have; Go
// forbids a field and a method sharing a name, which is the only reason the
// field is Component and the method is ID.
//
// This sum is deliberately NOT shared with ElementScopeParent, the {scope}.
// {parent} sum: §J Summary of Changes makes {context} (on type definitions) and
// {scope} (on declarations) separate, differently-shaped properties — §3.3.1
// sc_e types {parent} as Complex Type Definition | Model Group Definition,
// while §3.4.1 types {context} as Element Declaration | Complex Type
// Definition. Merging them would make "a complex type contexted in a model
// group definition" representable. Separate sums, one shared primitive.
//
// The ElementDeclarationContext arm — the only arm any mapping rule reaches — is
// populated by the parser as of #340, for the inline anonymous <complexType> of
// a local or a global <element> alike. ComplexTypeDefinitionContext stays
// producer-unreachable by construction, not by omission; see its own doc.
type ComplexTypeContext interface {
	complexTypeContext()
	// ID returns the identity of the component this {context} names. It is
	// present (non-zero) on any arm NewAnonymousComplexType accepted.
	ID() ComponentID
}

// ElementDeclarationContext is the {context} arm naming the containing Element
// Declaration (§3.4.2.1 dcl.ctd.common: "the Element Declaration corresponding
// to the nearest <element> information item among the ancestor element
// information items"). It is the ONLY arm any mapping rule in the spec
// produces.
//
// Component is that declaration's minted identity; it is a PRESENT identity,
// never the zero ComponentID — NewAnonymousComplexType rejects an unminted one.
// The field is read-only by convention; do not mutate it after construction.
//
// A mis-pointing {context} — an InlineTypeDefinition wrapping an anonymous
// ComplexType whose {context} names some OTHER component — is unrepresentable as
// of #340: NewElementDeclarationOwningTypes is the only construction path for a
// declaration that owns an anonymous complex type, and it rejects a Component
// that differs from the identity the caller minted for the declaration itself.
//
// That same identity is what the anonymous type's own nested local element
// declarations report as their {scope}.{parent}, through
// AnonymousComplexTypeScopeParent.Owner (elementdeclaration.go): the {context}
// back-pointer here and that forward reference are ONE fact with one encoding
// (STYLE D3), one mint per inline construct. See that type for the other half.
type ElementDeclarationContext struct{ Component ComponentID }

// ComplexTypeDefinitionContext is the {context} arm naming a containing Complex
// Type Definition. §3.4.1's tableau declares it, and no mapping rule in
// Structures §3.4.2 reaches it: §3.4.2.1 dcl.ctd.common has exactly one case,
// which yields an Element Declaration, and the §3.4.1 <complexType> content
// model has no slot for a nested unnamed <complexType> child.
//
// ONE rule outside §3.4.2 does reach it: §4.2.4 src-expredef clause 1.1, whose
// {name}-·absent· ORIGINAL has "its {context} … the redefining component" —
// which for a redefining <complexType> is a Complex Type Definition. That is the
// arm's producer-reachable case, minted and checked by NewComplexTypeOwningBase,
// and by NewAnonymousComplexTypeOwningBase when the redefining component is
// itself such an original (a chained <redefine>, #585). Before that landing the
// arm existed only so the §3.4.1 value space stayed whole for ct-props-correct
// (§3.4.6.1) clause 1 bookkeeping and so a caller assembling a schema
// programmatically was not rejected for building a legal component state.
//
// Component is the containing definition's minted identity, present under the
// same rule as ElementDeclarationContext's. The field is read-only by
// convention; do not mutate it after construction.
type ComplexTypeDefinitionContext struct{ Component ComponentID }

// complexTypeContext marks ElementDeclarationContext as a ComplexTypeContext
// (§3.4.1 ctd-context); see the ComplexTypeContext doc.
func (ElementDeclarationContext) complexTypeContext() {}

// complexTypeContext marks ComplexTypeDefinitionContext as a ComplexTypeContext
// (§3.4.1 ctd-context); see the ComplexTypeContext doc.
func (ComplexTypeDefinitionContext) complexTypeContext() {}

// ID returns the identity of the containing Element Declaration.
func (e ElementDeclarationContext) ID() ComponentID { return e.Component }

// ID returns the identity of the containing Complex Type Definition.
func (c ComplexTypeDefinitionContext) ID() ComponentID { return c.Component }

// checkComplexTypeContext rejects a {context} arm carrying the zero (unminted)
// ComponentID, charged to xsderr.RuleComponentInvariant rather than to
// ct-props-correct: the tableau's own Required-ness is what the nil-context
// rejection in NewAnonymousComplexType cites, whereas a ComponentID is THIS
// package's invented representation, so a malformed one is a representation
// invariant we own — the footing checkTypeDefinitionOrRef already stands on.
//
// The switch is exhaustive over the sealed sum; the default arm asserts the
// invariant and is unreachable for any value an outside package can produce,
// since complexTypeContext is unexported (mirroring elementScopeParentName).
// A nil context is caller-checked before this point, so it is not handled here.
func checkComplexTypeContext(loc xsderr.Loc, context ComplexTypeContext) error {
	switch context.(type) {
	case ElementDeclarationContext, ComplexTypeDefinitionContext:
		if context.ID() == (ComponentID{}) {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"anonymous complex type definition {context} is a %T carrying an unminted identity, but this representation identifies the containing component by identity token; mint one with NewComponentID", context)
		}
		return nil
	default:
		panic("xsd: checkComplexTypeContext: non-exhaustive ComplexTypeContext switch")
	}
}

// ComplexType is the Complex Type Definition component (Structures §3.4.1,
// id="Complex_Type_Definition_details"): a kind of Type Definition with
// {annotations}, {name} (bundled with {target namespace} as an xsd.QName per
// this package's "Names are expanded QNames" convention — doc.go; the zero
// QName is an anonymous complex type), {context}, {base type definition},
// {final}, {derivation method}, {abstract}, {attribute uses}, {attribute
// wildcard} (Optional), {content type}, {prohibited substitutions}, and
// {assertions}.
//
// Like the other §3 component shapes in this package, ComplexType is a
// STRUCTURAL holder built before resolution. {base type definition} is carried
// as a TypeDefinitionOrRef (typedefinition.go): ordinarily a pre-resolution
// TypeDefinitionRef naming a top-level type, and for the one mapping rule that
// needs an already-resolved anonymous base — §4.2.4 src-expredef clause 1.1's
// redefine pairing — an InlineTypeDefinition owning it outright. Finalize
// (resolve.go, #173) VALIDATES that a named reference resolves to a type
// definition (src-resolve clause 1.1) and that the complex-type base chain is
// acyclic except xs:anyType's self-derivation (ct-props-correct clause 3), but
// does NOT rewrite a name into a resolved component: the QName is retained, and
// a consumer follows the slot with Base. Finalize also charges ct-props-correct
// clauses 2 and 4 and, against the resolved base, derivation-ok-restriction
// (§3.4.6.3) for a restriction and cos-ct-extends (§3.4.6.2) for an extension
// (Phase D, complexderivation.go and complexextension.go, #262/#264). Clause 1's
// remaining resolved parts stay deferred.
//
// {attribute uses} and {attribute wildcard} are the TWO properties Finalize
// completes rather than merely checks, because each has a mapping clause that
// needs the resolved base. §3.4.2.4 clause 3 folds the {base type definition}'s
// uses into the type's own, so the value a producer supplies to NewComplexType is
// clauses 1 and 2 alone and Phase D overwrites it with the full set
// (attributeusefold.go, #401). §3.4.2.5 clause 2.2 unions an EXTENSION's own
// ·complete wildcard· with the base's, so for an extension the supplied value is
// clause 1 alone and Phase D overwrites it with the union
// (attributewildcardfold.go, #265); for a restriction clause 2.1 makes the
// supplied value already final. See AttributeUses and AttributeWildcard.
//
// prohibitedAttributeNames is the one field here that is NOT a §3.4.1 property.
// It is a retained MAPPING INPUT: clause 3.2.2 excludes from that fold the
// expanded name of "what would have been an attribute use corresponding to an
// <attribute> child, if the <attribute> had not had use = prohibited", and
// §3.4.2.4's Note makes such an <attribute> correspond to no component at all —
// so nothing among the properties records it and absence from {attribute uses}
// cannot distinguish "never declared" from "explicitly prohibited". The fold
// runs after the producer is gone, so the fact travels on the component. It is
// about ONE type's own source declaration, consulted once at that type's own
// fold step, and is never walked up a base chain.
//
// {context} (§3.4.1 ctd-context) is the component an ANONYMOUS type appears in,
// and the §3.4.1 tableau makes it and {name} a strict XOR: "Required if {name}
// is ·absent·, otherwise must be ·absent·". It is carried as a
// ComplexTypeContext — a discriminated reference to the containing Element
// Declaration or Complex Type Definition by minted ComponentID, not by name,
// because §3.4.2.1 dcl.ctd.common makes the target the nearest ancestor
// <element>, frequently a LOCAL element declaration, which is not name-unique.
// See ComponentID for the identity scheme and ComplexTypeContext for the sum;
// see Context for the accessor.
//
// The XOR needs no runtime check, because it is enforced by a CONSTRUCTION-PATH
// PARTITION rather than by re-validating two independently optional fields:
// NewComplexType and NewComplexTypeOwningBase build the named variety and take
// no context, and NewAnonymousComplexType and NewAnonymousComplexTypeOwningBase
// build the anonymous one and take no name. It is the same move
// NewGlobalScope/NewLocalScope make for {scope}'s tableau correlation, and it
// banks the same payoff: a named type carrying a context, and an anonymous one
// missing one, are both unrepresentable.
//
// loc and {context} are two different NON-property facts about a component and
// are easy to conflate: loc is PROVENANCE, where the declaring element sits in
// a schema document, and two distinct components may share one; a ComponentID
// is IDENTITY, minted, opaque, and deliberately not derived from position. See
// Loc, whose meaning this adds nothing to.
//
// Ratchet impact: the schema conformance lane widened when the producer started
// building this shape (#176) and again when it started building the ANONYMOUS
// one, {context} and all, for an inline <complexType> child of an <element>
// (#340).
//
// Construct only through NewComplexType (named), NewComplexTypeOwningBase (named,
// owning its base), NewAnonymousComplexType (anonymous) or
// NewAnonymousComplexTypeOwningBase (anonymous, owning its base), which reject the
// tableau-shape states ct-props-correct (§3.4.6.1) clause 1 forbids so they are
// unrepresentable (STYLE T1). None is the full property-correctness check (see
// ruleCTPropsCorrect's doc for exactly which clauses are deferred). ComplexType
// is immutable after construction, and remains so with a {context}: the cell a
// ComponentID points at is opaque and holds no mutable state, so copying a
// ComplexType is still a complete copy — the copies share one {context} identity,
// which is the point.
type ComplexType struct {
	loc  xsderr.Loc // source position; provenance, not a §3.4.1 property
	name QName
	// context is the {context} property: nil ⇔ absent, which the §3.4.1
	// tableau makes equivalent to "name is present". Presence is the nil
	// check, never a companion bool (STYLE D3). The name/context XOR holds
	// because only NewComplexType and NewAnonymousComplexType write this
	// pair; newComplexType itself can express both-present and
	// neither-present and does not check.
	context ComplexTypeContext
	// base is the {base type definition} property. nil ⇔ ·absent·, the one
	// encoding of absence this sum has in every slot it serves; see Base for
	// why a state ct-props-correct clause 1 makes Required is representable.
	base             TypeDefinitionOrRef
	derivationMethod DerivationMethod
	final            []DerivationMethod
	abstract         bool
	attributeUses    []AttributeUse
	// prohibitedAttributeNames is a mapping input, not a §3.4.1 property: the
	// expanded names this type's OWN source declaration gave use="prohibited"
	// (§3.4.2.4 clause 3.2.2). See the type doc.
	prohibitedAttributeNames []QName
	attributeWildcard        Wildcard
	hasAttributeWildcard     bool
	contentType              ContentType
	prohibitedSubstitutions  []DerivationMethod
	assertions               []Assertion
	annotations              []Annotation
}

// NewComplexType builds a NAMED ComplexType — one whose {context} is ·absent·
// per the §3.4.1 tableau — rejecting the tableau-shape states Complex Type
// Definition Properties Correct (§3.4.6.1, ct-props-correct) clause 1 forbids:
//
//   - name must be present: its local part may not be empty. §3.4.1 types
//     {name} as "an xs:NCName, or ·absent·", so an empty local part is a third
//     state with no tableau meaning, and an ABSENT {name} makes {context}
//     Required — which this entry point cannot supply. Build an anonymous
//     complex type through NewAnonymousComplexType instead. (Testing the local
//     part, not name == QName{}, is deliberate: the latter would admit
//     QName{Space: "urn:x", Local: ""} as a named type. Same idiom as
//     NewLocalScope's parent-name check.)
//   - derivationMethod must be one of DerivationExtension or DerivationRestriction
//     (the §3.4.1 {derivation method} value space).
//   - every final member and every prohibitedSubstitutions member must be
//     DerivationExtension or DerivationRestriction (the §3.4.1 {final} and
//     {prohibited substitutions} subsets).
//   - contentType is Required: a nil ContentType is rejected. A SimpleContent
//     must carry a non-nil {simple type definition}; an ElementContent must
//     carry a {particle} with a present {term} (a zero Particle{} is rejected,
//     mirroring NewParticle's own p-props-correct clause 1 nil-{term} check).
//
// It also rejects one state Wildcard Properties Correct (§3.10.6.1,
// w-props-correct) clause 5 forbids, charged to that rule rather than to
// ct-props-correct: an {attribute wildcard} whose {namespace constraint}
// carries the sibling keyword. This slot is one of the two places an attribute
// wildcard is identifiable as such; see rejectSiblingOnAttributeWildcard.
//
// The substantive cross-component clauses (base-type resolution, circularity,
// attribute-use expanded-name uniqueness) and the derivation-validity rules are
// NOT checked here — see ruleCTPropsCorrect's doc.
//
// prohibitedAttributeNames is not a §3.4.1 property and is not validated here:
// it is the mapping input §3.4.2.4 clause 3.2.2 needs and any set of expanded
// names is legal (the Note makes a prohibited <attribute> "pointless, though not
// an error" outside a restriction). It arrives through this constructor rather
// than through a setter because ComplexType is immutable after construction and
// this is its single construction path (STYLE T1); the alternative — an
// exported With… copier — would add a second way to build the value for the sake
// of one field that has nothing to check.
//
// baseTypeDefinitionName is a pre-resolution QName reference, not a resolved
// component; nothing resolves it yet (#173). It is lifted into the {base type
// definition} slot by baseTypeDefinitionRef, so the zero QName is stored as the
// nil (·absent·) slot rather than as a nameless reference — see Base. A base
// this entry point cannot express, the anonymous already-resolved one of
// src-expredef clause 1.1, has its own entry point in NewComplexTypeOwningBase.
// attributeWildcard is a pointer so absence (nil) is distinct from a present
// zero record (mirroring attributegroupdefinition.go's *Wildcard slot); when
// non-nil the pointed-to value is COPIED into the struct and
// hasAttributeWildcard is set — the pointer itself is never stored, so the
// caller's value is not aliased. Every slice parameter is copied; the caller's
// backing arrays are not aliased, and an empty input is held as nil.
//
// loc is the source position charged to any rejection AND retained: Loc reports
// it back as the definition's provenance. Pass the position of this
// definition's own declaring element, never a convenient nearby one (a parent
// element's, say) — it is observable, not merely an error-charging convenience.
// A caller with no real parser position — a synthesized or programmatically
// built definition — passes the zero xsderr.Loc{}, which reads as "unknown".
func NewComplexType(loc xsderr.Loc, name QName, baseTypeDefinitionName QName, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, prohibitedAttributeNames []QName, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
	if name.Local == "" {
		return ComplexType{}, xsderr.New(ruleCTPropsCorrect, loc,
			"complex type definition has an absent {name}, but the §3.4.1 tableau makes {context} Required when {name} is absent; build an anonymous complex type through NewAnonymousComplexType (ct-props-correct clause 1)")
	}
	return newComplexType(loc, name, nil, baseTypeDefinitionRef(baseTypeDefinitionName), final, derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
}

// baseTypeDefinitionRef lifts a pre-resolution base QName into the {base type
// definition} slot: the zero QName is the ·absent· base, which the sum encodes
// as nil and never as a nameless TypeDefinitionRef (which
// checkTypeDefinitionOrRef rejects as an illegal representation). It is the ONE
// place the QName-shaped constructor parameters meet the sum, so the two
// encodings of absence cannot drift apart (STYLE D3).
func baseTypeDefinitionRef(name QName) TypeDefinitionOrRef {
	if name == (QName{}) {
		return nil
	}
	return TypeDefinitionRef{Name: name}
}

// NewComplexTypeOwningBase builds a NAMED ComplexType whose {base type
// definition} is the ANONYMOUS, already-resolved component §4.2.4 src-expredef
// clause 1.1 pairs a redefining <complexType> with: "one component which
// corresponds to the top-level definition item with the same name in the
// <redefine>d schema document … except that its {name} is ·absent· and its
// {context} is the redefining component". It is NewComplexType's parameter list
// with the base-QName slot replaced, in place, by the owned definition and
// preceded by the redefining type's own minted identity; every other parameter,
// check, copy and rejection is identical, because all three entry points run one
// shared core. base is wrapped in an InlineTypeDefinition here, so a caller never
// spells that arm.
//
// It is the ONLY construction path for a complex type that owns its base, and it
// exists to make ONE state unrepresentable that no shape check could reach: an
// owned original whose {context} names some OTHER component, which is the
// redefine-side form of the context-tracking hazard PRINCIPLES 16 names for
// <override>. It adds three rejections beyond NewComplexType's, all charged to
// xsderr.RuleComponentInvariant because a ComponentID is this package's
// representation rather than a spec-visible name — the footing
// checkComplexTypeContext already stands on:
//
//   - an unminted (zero) id: there would be nothing to compare the {context}
//     against;
//   - a {context} that is a ComplexTypeDefinitionContext naming a DIFFERENT
//     identity;
//   - a {context} that is an ElementDeclarationContext: clause 1.1 makes the
//     original's {context} "the redefining component", which for a redefining
//     <complexType> is a Complex Type Definition and never a declaration.
//
// A base that is NAMED (its {context} therefore ·absent· per §3.4.1's XOR) is
// rejected by the shared core's checkTypeDefinitionOrRef, which admits only an
// anonymous type in the InlineTypeDefinition arm.
//
// # Why an identity, and why it is not stored
//
// id is NOT stored on the built type. Its whole role is this construction-time
// comparison, and a field written but never read is dead state (STYLE D3); the
// landing that adds an ID→component resolver adds the field together with its
// reader (see ComponentID), exactly as NewElementDeclarationOwningTypes's doc
// already commits this repo to. That makes the back-pointer's target
// unobservable, which is admissible only BECAUSE the token is never stored: for
// a NAMED, top-level component like this one, identity normally IS its expanded
// name, so a stored token would be a second encoding of that identity (STYLE
// D3).
//
// The name cannot serve in the token's place. §3.4.1's {context} is carried as a
// ComplexTypeContext, whose arms hold a ComponentID and not a QName (see
// ComplexTypeContext for why: §3.4.2.1's other case targets a frequently LOCAL
// element declaration, which is not name-unique). Adding a name-carrying arm to
// make this one case expressible would fork the §3.4.1 sum in two for a single
// mapping rule, and every exhaustive switch over it would grow a case that means
// the same thing as an existing one (STYLE T4).
func NewComplexTypeOwningBase(loc xsderr.Loc, id ComponentID, name QName, base ComplexType, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, prohibitedAttributeNames []QName, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
	if name.Local == "" {
		return ComplexType{}, xsderr.New(ruleCTPropsCorrect, loc,
			"complex type definition has an absent {name}, but the §3.4.1 tableau makes {context} Required when {name} is absent; build an anonymous complex type through NewAnonymousComplexType (ct-props-correct clause 1)")
	}
	if err := checkOwnedBaseContext(loc, id, complexTypeLabel(name), base); err != nil {
		return ComplexType{}, err
	}
	return newComplexType(loc, name, nil, InlineTypeDefinition{Definition: base}, final, derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
}

// NewAnonymousComplexTypeOwningBase builds an ANONYMOUS ComplexType that itself
// OWNS the §4.2.4 src-expredef clause 1.1 original of a further redefinition —
// the shape a CHAINED <redefine> produces, where D1 redefines D2 and D2 redefines
// D3 for one (kind, name). Clause 1.1 makes D1's original "the top-level
// definition item with the same name in the <redefine>d schema document, as
// defined in Schema Component Details (§3), except that its {name} is ·absent·
// and its {context} is the redefining component" — and when that item is D2's own
// redefining <complexType>, "as defined in §3" carries clause 1.2 with it, so the
// original is anonymous AND owns an original of its own (D3's).
//
// It is NewComplexTypeOwningBase's parameter list with the name argument replaced,
// in place, by the context, and it applies both entry points' checks: the
// nil-context and unminted-context rejections NewAnonymousComplexType charges
// (checkAnonymousComplexTypeContext), and the unminted-id and
// {context}-must-name-the-owner rejections NewComplexTypeOwningBase charges
// (checkOwnedBaseContext).
//
// id is this type's identity AS AN OWNER, and is DISTINCT from the identity its
// own {context} carries: one ComponentID is minted per ownership edge, so a chain
// C1 owns O2 owns O3 mints one token for C1→O2 (which is O2's {context}, and what
// O2's local declarations report as {scope}.{parent}) and another for O2→O3. A
// single token for both edges would make O2 and O3 indistinguishable containers,
// so this entry point REJECTS the collapse — the rejection is its own, because it
// is the only constructor holding both edges of a chain at once: the owner-side
// token and the {context} the same component takes from the level above it.
// It is not stored, for the reason NewComplexTypeOwningBase's doc gives.
func NewAnonymousComplexTypeOwningBase(loc xsderr.Loc, id ComponentID, context ComplexTypeContext, base ComplexType, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, prohibitedAttributeNames []QName, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
	if err := checkAnonymousComplexTypeContext(loc, context); err != nil {
		return ComplexType{}, err
	}
	if context.ID() == id {
		return ComplexType{}, xsderr.New(xsderr.RuleComponentInvariant, loc,
			"anonymous complex type definition owns its {base type definition} under the SAME identity its own {context} carries, but one identity is minted per ownership edge; the two containers would be indistinguishable, and the owned original's {context} would name this type's own container instead of this type; mint a separate identity with NewComponentID for each edge")
	}
	if err := checkOwnedBaseContext(loc, id, complexTypeLabel(QName{}), base); err != nil {
		return ComplexType{}, err
	}
	return newComplexType(loc, QName{}, context, InlineTypeDefinition{Definition: base}, final, derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
}

// checkOwnedBaseContext rejects an owning type whose identity id is unminted, and
// an owned anonymous base whose {context} (§3.4.1 ctd-context) does not name that
// id. An ABSENT {context} means base is NAMED, which the shared core's
// checkTypeDefinitionOrRef rejects with the message that fits it; this checker
// passes it through rather than duplicating that verdict (STYLE T4).
//
// The unminted-id rejection precedes the ABSENT-{context} pass-through, so it is
// charged for a named base too: it is a fault in what the CALLER minted, and an
// owning entry point that reached the core with the zero token would have written
// a back-pointer target nothing can name.
//
// label names the owning type in both verdicts, from complexTypeLabel, because
// both owning entry points reach here and only one of them has a {name} to print.
//
// The switch is exhaustive over ComplexTypeContext's sealed sum; the default arm
// asserts the invariant and is unreachable for any value an outside package can
// produce, since complexTypeContext is unexported. It is checkOwnedTypeContext's
// twin (elementdeclaration.go) with the two arms' verdicts swapped, because the
// two ownership edges point at different kinds of container.
func checkOwnedBaseContext(loc xsderr.Loc, id ComponentID, label string, base ComplexType) error {
	if id == (ComponentID{}) {
		return xsderr.New(xsderr.RuleComponentInvariant, loc,
			"%s owns its {base type definition} but carries an unminted identity, which that base's {context} back-pointer could not name; mint one with NewComponentID", label)
	}
	context, present := base.Context()
	if !present {
		return nil
	}
	switch c := context.(type) {
	case ComplexTypeDefinitionContext:
		if c.ID() != id {
			return xsderr.New(xsderr.RuleComponentInvariant, loc,
				"%s owns an anonymous {base type definition} whose {context} names a DIFFERENT complex type definition, but src-expredef clause 1.1 makes that {context} the redefining component itself; mint one identity per ownership edge and pass it to both ends", label)
		}
		return nil
	case ElementDeclarationContext:
		return xsderr.New(xsderr.RuleComponentInvariant, loc,
			"%s owns an anonymous {base type definition} whose {context} is an ElementDeclarationContext, but src-expredef clause 1.1 makes the redefined original's {context} the redefining component, which here is a Complex Type Definition and never an Element Declaration", label)
	default:
		panic("xsd: checkOwnedBaseContext: non-exhaustive ComplexTypeContext switch")
	}
}

// NewAnonymousComplexType builds an ANONYMOUS ComplexType — one whose {name} is
// ·absent· and whose {context} the §3.4.1 tableau therefore makes Required. Its
// parameter list is NewComplexType's with the name argument replaced, in place,
// by the context; every other parameter, check, copy, and rejection is
// identical, because both entry points run one shared core.
//
// It adds two rejections of its own:
//
//   - a nil context: {context} is Required when {name} is absent (§3.4.1's
//     tableau), charged to ct-props-correct clause 1 — the tableau's own
//     Required-ness is a spec fact about the property.
//   - a context arm carrying an unminted (zero) ComponentID, charged to
//     xsderr.RuleComponentInvariant instead: a ComponentID is this package's
//     representation, not a spec-visible name, so a malformed one is an
//     invariant we own. See checkComplexTypeContext.
//
// There is no third check for "named AND contexted": that state does not
// type-check at this layer, because this constructor accepts no name and neither
// NewComplexType nor NewComplexTypeOwningBase accepts a context.
//
// The parser calls this for the inline anonymous <complexType> of a local or a
// global <element> (#340), always with an ElementDeclarationContext naming the
// declaration it is building; see ComplexTypeContext.
func NewAnonymousComplexType(loc xsderr.Loc, context ComplexTypeContext, baseTypeDefinitionName QName, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, prohibitedAttributeNames []QName, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
	if err := checkAnonymousComplexTypeContext(loc, context); err != nil {
		return ComplexType{}, err
	}
	return newComplexType(loc, QName{}, context, baseTypeDefinitionRef(baseTypeDefinitionName), final, derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
}

// checkAnonymousComplexTypeContext charges the two rejections every anonymous
// entry point owes its {context}: a nil one, which §3.4.1's tableau makes
// Required when {name} is absent, and an arm carrying an unminted ComponentID.
// Both anonymous constructors call it, so neither can drop one half (STYLE T4).
// See NewAnonymousComplexType's doc for why the two rejections cite different
// rules, and checkComplexTypeContext for the second.
func checkAnonymousComplexTypeContext(loc xsderr.Loc, context ComplexTypeContext) error {
	if context == nil {
		return xsderr.New(ruleCTPropsCorrect, loc,
			"anonymous complex type definition has an absent {context}, but the §3.4.1 tableau requires it to be present when {name} is absent (ct-props-correct clause 1)")
	}
	return checkComplexTypeContext(loc, context)
}

// newCollapsedExtension builds the collapsed intermediate M that cos-ct-extends
// (§3.4.6.2) clause 1.5's Note describes: the single extension of a, the ancestor
// whose {base type definition} is ·xs:anyType·, that the Note's re-ordering
// yields (collapsedintermediate.go computes the properties it is handed).
//
// It is newComplexType's FOURTH caller, and it re-establishes that function's
// {name}/{context} XOR precondition by borrowing A's identity pair WHOLE: M takes
// a's {name} AND a's {context}. A is itself a validly constructed component, so
// exactly one of that pair is present and the XOR holds however A was built —
// which is the only formulation that survives A being the ANONYMOUS src-expredef
// clause 1.1 original (#505). Passing a's {name} with a hard-coded nil {context}
// held only while every A was named; for an anonymous A it would build a type
// that is neither named nor contexted, the one state §3.4.1's tableau forbids
// outright. Building M through the validating core rather than as a bare
// ComplexType literal is what keeps every ct-props-correct clause-1 shape check on
// it — a literal inside this package would bypass all of them, which is the one
// construction hole this type's whole design exists to close (STYLE T1).
//
// M is a VALUE with a lifetime of one constraint check. It is never added to a
// SchemaBuilder, never written into s.types or s.typeIndex, and never reachable
// from a resolver: it corresponds to no <complexType> element, so a schema that
// could see it would report a component no document declares. Its identity is a's
// only so that the base walks derivation-ok-restriction performs off it terminate
// on real components; every error the check produces is re-charged against T
// before it reaches a caller (checkExtensionTwoStepDerivable).
//
// Two properties are FIXED here rather than collapsed, and both are load-bearing:
//
//   - {final} is EMPTY. The Note's intermediate has no source and hence no
//     final=; copying a's would make derivation-ok-restriction clause 1
//     (checkRestrictionBaseFinal) charge T for a constraint stated by a type that
//     does not exist, false-rejecting every chain rooted at a final="restriction"
//     ancestor. A's own {final} is already charged where it belongs, against A's
//     real derivations.
//   - {assertions} is A's. Along the REAL chain §3.4.2.1 clause 1 makes A's
//     assertions a prefix of T's, so derivation-ok-restriction clause 5
//     (checkRestrictionAssertions) is satisfied by construction.
//
// GAP(xsd): that {assertions} choice discharges clause 5 VACUOUSLY rather than
// deciding it — owned by #586. The true collapsed intermediate would also carry
// each extension step's own assertions, but under the re-ordering that sequence
// is in general NOT a prefix of T's (a restriction step's assertions sit between
// them), so charging it would be a false reject. The one reader neutralised is
// checkRestrictionAssertions (complexderivation.go), which this makes answer
// "prefix" for every chain: the direction is fail-open — clause 1.5 never rejects
// on assertions — and no other reader of M.{assertions} exists.
func newCollapsedExtension(loc xsderr.Loc, a ComplexType, p collapsedProperties) (ComplexType, error) {
	context, _ := a.Context()
	return newComplexType(loc, a.Name(), context, collapsedExtensionBase(a), nil, DerivationExtension, false,
		p.uses, nil, p.wildcard, p.content, nil, a.assertions, nil)
}

// collapsedExtensionBase is M's {base type definition}: a reference to A, in
// whichever arm of TypeDefinitionOrRef can express A (#505). It is what makes the
// base walks derivation-ok-restriction performs off M — key-ldtype case 3's
// recursion in locallyDeclaredAttributeType and locallyDeclaredElementType
// (complexderivation.go) — reach A and then A's own ancestors, which is the
// whole reason M carries A's identity at all.
//
// The arm follows from A's {name}, which is the ONE fact that distinguishes them
// and is never separately stored (see TypeDefinitionOrRef):
//
//   - A is NAMED: a TypeDefinitionRef, resolved by the ordinary Schema.Type
//     lookup, exactly as every base= produced from a document is.
//   - A is ANONYMOUS: A is the src-expredef clause 1.1 original a redefining
//     <complexType> owns, which is in no by-name symbol table, so a ref could not
//     name it and the InlineTypeDefinition arm carries the component itself.
//
// The InlineTypeDefinition arm's OWNERSHIP invariant is not violated by the
// second holder this creates. The arm's owner is the redefining type's own
// {base type definition} slot; M is a throwaway VALUE outside the component model
// altogether — never registered, never reachable from a resolver, discarded when
// the clause-1.5 check returns — so the copy it holds can neither be observed
// beside the original nor diverge from it (the divergence that doc warns of needs
// a fold writing through one holder, and no fold runs over M).
func collapsedExtensionBase(a ComplexType) TypeDefinitionOrRef {
	if a.Name() == (QName{}) {
		return InlineTypeDefinition{Definition: a}
	}
	return TypeDefinitionRef{Name: a.Name()}
}

// newComplexType is the shared core of NewComplexType,
// NewComplexTypeOwningBase and NewAnonymousComplexType: every check and copy
// that does not concern the {name}/{context} pair lives here exactly once
// (STYLE T4).
//
// PRECONDITION, enforced by its FOUR CALLERS and not by itself: exactly one of
// name and context is present, per the §3.4.1 tableau's XOR. This layer can
// express both-present and neither-present and does not reject either; the
// partition's guarantee comes from the entry points, each of which accepts only
// one half of the pair — the three exported ones by their parameter lists, and
// newCollapsedExtension by copying the already-validated pair off the ancestor A
// it collapses onto. Any further caller added here must re-establish the XOR
// itself.
//
// A SECOND precondition belongs to NewComplexTypeOwningBase alone: a base
// holding an InlineTypeDefinition arrived through it, so the owned component's
// {context} has been checked against the owner's identity. This layer cannot
// express that check — it takes no identity — and does not attempt one.
// newCollapsedExtension is outside that precondition and needs no such check: the
// anonymous base it may pass is A itself, already constructed and already
// context-checked against its real owner, and M is never a component a consumer
// can reach.
func newComplexType(loc xsderr.Loc, name QName, context ComplexTypeContext, base TypeDefinitionOrRef, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, prohibitedAttributeNames []QName, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
	if err := checkTypeDefinitionOrRef(loc, base, baseTypeSlot, complexTypeLabel(name)); err != nil {
		return ComplexType{}, err
	}
	switch derivationMethod {
	case DerivationExtension, DerivationRestriction:
	default:
		return ComplexType{}, xsderr.New(ruleCTPropsCorrect, loc,
			"complex type definition has an unknown {derivation method}: %s, but only extension or restriction are legal (ct-props-correct clause 1)", derivationMethod)
	}
	for i, m := range final {
		switch m {
		case DerivationExtension, DerivationRestriction:
		default:
			return ComplexType{}, xsderr.New(ruleCTPropsCorrect, loc,
				"complex type definition {final}[%d] is %s, but only extension or restriction are legal (ct-props-correct clause 1)", i, m)
		}
	}
	for i, m := range prohibitedSubstitutions {
		switch m {
		case DerivationExtension, DerivationRestriction:
		default:
			return ComplexType{}, xsderr.New(ruleCTPropsCorrect, loc,
				"complex type definition {prohibited substitutions}[%d] is %s, but only extension or restriction are legal (ct-props-correct clause 1)", i, m)
		}
	}
	if err := checkContentType(loc, contentType); err != nil {
		return ComplexType{}, err
	}
	if err := rejectSiblingOnAttributeWildcard(loc, attributeWildcard); err != nil {
		return ComplexType{}, err
	}
	c := ComplexType{
		loc:              loc,
		name:             name,
		context:          context,
		base:             base,
		derivationMethod: derivationMethod,
		abstract:         abstract,
		contentType:      contentType,
	}
	if len(final) > 0 {
		c.final = append([]DerivationMethod(nil), final...)
	}
	if len(attributeUses) > 0 {
		c.attributeUses = append([]AttributeUse(nil), attributeUses...)
	}
	if len(prohibitedAttributeNames) > 0 {
		c.prohibitedAttributeNames = append([]QName(nil), prohibitedAttributeNames...)
	}
	if attributeWildcard != nil {
		c.attributeWildcard, c.hasAttributeWildcard = *attributeWildcard, true
	}
	if len(prohibitedSubstitutions) > 0 {
		c.prohibitedSubstitutions = append([]DerivationMethod(nil), prohibitedSubstitutions...)
	}
	if len(assertions) > 0 {
		c.assertions = append([]Assertion(nil), assertions...)
	}
	if len(annotations) > 0 {
		c.annotations = append([]Annotation(nil), annotations...)
	}
	return c, nil
}

// checkContentType enforces the Required {content type} tableau-shape parts of
// ct-props-correct clause 1: {content type} present, a SimpleContent's {simple
// type definition} present, and an ElementContent's {particle} carrying a
// present {term}. The type switch is exhaustive over the sealed ContentType sum
// (EmptyContent needs no check); a nil interface is caught first.
func checkContentType(loc xsderr.Loc, contentType ContentType) error {
	if contentType == nil {
		return xsderr.New(ruleCTPropsCorrect, loc,
			"complex type definition has an absent {content type}, but it is Required (ct-props-correct clause 1)")
	}
	switch ct := contentType.(type) {
	case SimpleContent:
		if ct.SimpleType == nil {
			return xsderr.New(ruleCTPropsCorrect, loc,
				"complex type definition {content type} is simple but has a nil {simple type definition}, which is Required (ct-props-correct clause 1)")
		}
	case ElementContent:
		if ct.Particle.Term() == nil {
			return xsderr.New(ruleCTPropsCorrect, loc,
				"complex type definition {content type} is %s but its {particle} has an absent {term}, which is Required (ct-props-correct clause 1; cf. p-props-correct clause 1)", ct.Variety())
		}
	}
	return nil
}

// Name returns the {name} property, bundled with {target namespace} as a QName.
// The zero QName denotes an anonymous complex type (§3.4.1) — which by the same
// tableau row carries a Required {context} naming the component it appears in;
// see Context. A value built through NewComplexType always has a present name
// and an absent context, and one built through NewAnonymousComplexType always
// has the reverse.
func (c ComplexType) Name() QName {
	return c.name
}

// Context returns the {context} property (§3.4.1 ctd-context) as a
// discriminated identity reference to the containing Element Declaration or
// Complex Type Definition; the second result is false when it is absent (a
// NAMED type), in which case the first result is nil.
//
// This is NOT the resolved container component: see ComplexTypeContext for why
// the reference is carried by identity token rather than by name, and
// ComponentID for how a consumer compares one (with ==, never
// reflect.DeepEqual).
func (c ComplexType) Context() (ComplexTypeContext, bool) {
	return c.context, c.context != nil
}

// Loc reports the source position of the declaring element — provenance, not a
// §3.4.1 component property (see the package doc's Components section). The
// zero xsderr.Loc means the position is unknown, as it is for the synthesized
// xs:anyType.
func (c ComplexType) Loc() xsderr.Loc {
	return c.loc
}

// Base returns the {base type definition} property (§3.4.1) as the
// TypeDefinitionOrRef sum, which a consumer switches exhaustively over:
//
//   - TypeDefinitionRef is the ordinary case, a PRE-RESOLUTION reference by
//     name. Finalize (#173) validates that it resolves to a type definition
//     (src-resolve clause 1.1) and that the base chain is acyclic
//     (ct-props-correct clause 3), but does not rewrite it: the QName is
//     retained, and a consumer obtains the component by a read-time
//     schema.Type(name) lookup, exactly as for ElementDeclaration.TypeDefinition.
//   - InlineTypeDefinition is the ALREADY-RESOLVED anonymous component §4.2.4
//     src-expredef clause 1.1 pairs a redefining <complexType> with; it is in no
//     symbol table, so it is reachable only through this slot, and no lookup
//     will find it. See NewComplexTypeOwningBase.
//   - nil is an ·absent· base. ct-props-correct (§3.4.6.1) clause 1 makes {base
//     type definition} Required, so nil is a state the spec's tableau does not
//     describe — but it IS representable, deliberately: the constructors do not
//     reject it, and finalize reads it as "the chain ends here" rather than as a
//     violation. Clause 1's Required-ness is not enforced at construction, and
//     closing that state is a behaviour change with its own ratchet risk, kept
//     apart from the slot's re-encoding (#505).
func (c ComplexType) Base() TypeDefinitionOrRef {
	return c.base
}

// complexTypeLabel renders a complex type being CONSTRUCTED as the owner phrase
// of a rejection message, from its {name} alone — the component does not exist
// yet, so componentwalk.go's complexTypeOwner, which takes one, cannot
// serve. An absent name (the anonymous variety) has no QName to print, so it is
// described by what it is instead (STYLE E1).
func complexTypeLabel(name QName) string {
	if name == (QName{}) {
		return "anonymous complex type"
	}
	return "complex type " + name.String()
}

// DerivationMethod returns the {derivation method} property: DerivationExtension
// or DerivationRestriction (§3.4.1).
func (c ComplexType) DerivationMethod() DerivationMethod {
	return c.derivationMethod
}

// Final returns the {final} property (a subset of {extension, restriction}) in
// document order. It returns a copy: mutating the result does not affect c. An
// empty subset yields nil.
func (c ComplexType) Final() []DerivationMethod {
	if len(c.final) == 0 {
		return nil
	}
	return append([]DerivationMethod(nil), c.final...)
}

// Abstract returns the {abstract} property (§3.4.1).
func (c ComplexType) Abstract() bool {
	return c.abstract
}

// AttributeUses returns the {attribute uses} property in document order. It
// returns a copy: mutating the result does not affect c. An empty {attribute
// uses} yields nil.
//
// The spec property is a set (§3.4.1); the document order here is an
// implementation choice for determinism and carries no spec significance.
//
// On a type reached through a finalized [Schema] this is the §3.4.2.4 clause 3
// property — the type's own uses followed by those inherited from its {base type
// definition}, less what clauses 3.2.1 and 3.2.2 exclude — because Finalize
// materialises the fold (attributeusefold.go, #401). That holds for an ANONYMOUS
// type a declaration owns as well as for a named one: §3.4.2.4 makes the mapping
// rule "the same for all complex type definitions", and the fold reaches an
// inline <complexType> nested in a particle tree or declared under an
// <alternative> through the slot that owns it (ownedtypefold.go, #414).
//
// On a ComplexType a caller built with [NewComplexType] and has not yet
// finalized, it is only what that caller passed in: clause 3 needs the base
// COMPONENT, which a standalone value has no way to reach.
func (c ComplexType) AttributeUses() []AttributeUse {
	if len(c.attributeUses) == 0 {
		return nil
	}
	return append([]AttributeUse(nil), c.attributeUses...)
}

// AttributeWildcard returns the {attribute wildcard} property (Optional); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
//
// On a type reached through a finalized [Schema] this is the §3.4.2.5 clause 2
// property. For a restriction that is the ·complete wildcard· the caller supplied
// (clause 2.1); for an EXTENSION it is that wildcard's {namespace constraint}
// unioned with the {base type definition}'s per cos-aw-union, under the
// extension's own {process contents} and {annotations} (clause 2.2), because
// Finalize materialises the fold (attributewildcardfold.go, #265). As for
// AttributeUses, that holds for an ANONYMOUS type a declaration owns too: the
// §3.4.2.5 mapping rule is "the same for all complex type definitions", and the
// fold reaches such a type through the slot that owns it (ownedtypefold.go,
// #414).
//
// On a ComplexType a caller built with [NewComplexType] and has not yet
// finalized, it is only what that caller passed in: clause 2.2 needs the base
// COMPONENT, which a standalone value has no way to reach.
func (c ComplexType) AttributeWildcard() (Wildcard, bool) {
	return c.attributeWildcard, c.hasAttributeWildcard
}

// ContentType returns the {content type} property (Required): the sealed sum
// identifying the empty/simple/element-only/mixed content variety. It is never
// nil on a value built through NewComplexType.
func (c ComplexType) ContentType() ContentType {
	return c.contentType
}

// ProhibitedSubstitutions returns the {prohibited substitutions} property (a
// subset of {extension, restriction}) in document order. It returns a copy:
// mutating the result does not affect c. An empty subset yields nil.
func (c ComplexType) ProhibitedSubstitutions() []DerivationMethod {
	if len(c.prohibitedSubstitutions) == 0 {
		return nil
	}
	return append([]DerivationMethod(nil), c.prohibitedSubstitutions...)
}

// Assertions returns the {assertions} property in document order. It returns a
// copy: mutating the result does not affect c. An empty {assertions} yields nil.
func (c ComplexType) Assertions() []Assertion {
	if len(c.assertions) == 0 {
		return nil
	}
	return append([]Assertion(nil), c.assertions...)
}

// Annotations returns the {annotations} property in document order. It returns a
// copy: mutating the result does not affect c. An empty {annotations} yields
// nil.
func (c ComplexType) Annotations() []Annotation {
	if len(c.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), c.annotations...)
}
