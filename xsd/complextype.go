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
//     so it too is enforced at finalize (Phase D,
//     checkAttributeUseNamesUnique, #262) rather than at shape time (unlike
//     AttributeGroupDefinition, whose own ag-props-correct clause 2 is enforced
//     at shape time over uses it already owns);
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
// #263) — the exceptions there are clause 5's {assertions} prefix, and the
// content models §3.4.6.3's own all-group leniency licenses accepting
// provisionally. cos-ct-extends (§3.4.6.2) is enforced too, at the same phase
// (complexextension.go, #264), clause 1.4.3.2.2.2's cos-particle-extend
// (§3.9.6.2) delegate included — the exceptions there are clauses 1.3 and 1.7,
// which need the §3.4.2.4/§3.4.2.5 base folds no producer performs yet (#265),
// and clause 1.5 for a derivation chain mixing extension and restriction steps.
// None of them is touched HERE — they are cross-component finalize-phase
// concerns, not tableau shape.
const ruleCTPropsCorrect xsderr.Rule = "ct-props-correct"

// ContentType is the sealed sum of the four Content Type varieties of a Complex
// Type Definition (Structures §3.4.1 "Content Type" property record, id="ct").
// The spec's {variety} property is closed to exactly {empty, simple,
// element-only, mixed}, so the set of variant shapes is closed. The unexported
// contentType marker method seals it (STYLE T2/T7, the PRINCIPLES 7 sealed-sum
// exception) — consumers exhaustively switch these variants and no further one
// is representable — mirroring term.go's Term sealed sum.
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
// T2/T7, the PRINCIPLES 7 sealed-sum exception), mirroring this file's
// ContentType and elementdeclaration.go's ElementScopeParent.
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
// of #340: NewElementDeclarationOwningType is the only construction path for a
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
// Type Definition. §3.4.1's tableau declares it, and NO mapping rule in
// Structures reaches it: §3.4.2.1 dcl.ctd.common has exactly one case, which
// yields an Element Declaration, and the §3.4.1 <complexType> content model has
// no slot for a nested unnamed <complexType> child. That is a spec-text
// asymmetry, not an implementation gap — the arm exists so the §3.4.1 value
// space stays whole for ct-props-correct (§3.4.6.1) clause 1 bookkeeping and so
// a caller assembling a schema programmatically is not rejected for building a
// legal component state. Its sibling std-context (§3.16.1) is a four-member sum
// with every arm reachable, so an unreachable arm here is shape, not an edge
// case. It carries no GAP marker because there is nothing to complete: the
// absence is permanent and correct.
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
// as a pre-resolution QName REFERENCE (baseTypeDefinitionName), not a resolved
// simple-or-complex type. Finalize (resolve.go, #173) VALIDATES that the
// reference resolves to a type definition (src-resolve clause 1.1) and that the
// complex-type base chain is acyclic except xs:anyType's self-derivation
// (ct-props-correct clause 3), but does NOT rewrite it into a resolved
// component: the QName is retained, and a consumer follows it by a read-time
// schema.Type(name) lookup. Finalize also charges ct-props-correct clauses 2 and
// 4 and, against the resolved base, derivation-ok-restriction (§3.4.6.3) for a
// restriction and cos-ct-extends (§3.4.6.2) for an extension (Phase D,
// complexderivation.go and complexextension.go, #262/#264). Clause 1's remaining
// resolved parts stay deferred.
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
// NewComplexType builds the named variety and takes no context, and
// NewAnonymousComplexType builds the anonymous one and takes no name. It is the
// same move NewGlobalScope/NewLocalScope make for {scope}'s tableau
// correlation, and it banks the same payoff: a named type carrying a context,
// and an anonymous one missing one, are both unrepresentable.
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
// Construct only through NewComplexType (named) or NewAnonymousComplexType
// (anonymous), which reject the tableau-shape states ct-props-correct (§3.4.6.1)
// clause 1 forbids so they are unrepresentable (STYLE T1). Neither is the full
// property-correctness check (see ruleCTPropsCorrect's doc for exactly which
// clauses are deferred). ComplexType is immutable after construction, and
// remains so with a {context}: the cell a ComponentID points at is opaque and
// holds no mutable state, so copying a ComplexType is still a complete copy —
// the copies share one {context} identity, which is the point.
type ComplexType struct {
	loc  xsderr.Loc // source position; provenance, not a §3.4.1 property
	name QName
	// context is the {context} property: nil ⇔ absent, which the §3.4.1
	// tableau makes equivalent to "name is present". Presence is the nil
	// check, never a companion bool (STYLE D3). The name/context XOR holds
	// because only NewComplexType and NewAnonymousComplexType write this
	// pair; newComplexType itself can express both-present and
	// neither-present and does not check.
	context                ComplexTypeContext
	baseTypeDefinitionName QName
	derivationMethod       DerivationMethod
	final                  []DerivationMethod
	abstract               bool
	attributeUses          []AttributeUse
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
// this is its single construction path (STYLE T1/T7); the alternative — an
// exported With… copier — would add a second way to build the value for the sake
// of one field that has nothing to check.
//
// baseTypeDefinitionName is a pre-resolution QName reference, not a resolved
// component; nothing resolves it yet (#173). attributeWildcard is a pointer so
// absence (nil) is distinct from a present zero record (mirroring
// attributegroupdefinition.go's *Wildcard slot); when non-nil the pointed-to
// value is COPIED into the struct and hasAttributeWildcard is set — the pointer
// itself is never stored, so the caller's value is not aliased. Every slice
// parameter is copied; the caller's backing arrays are not aliased, and an empty
// input is held as nil.
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
	return newComplexType(loc, name, nil, baseTypeDefinitionName, final, derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
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
// type-check at this layer, because this constructor accepts no name and
// NewComplexType accepts no context.
//
// The parser calls this for the inline anonymous <complexType> of a local or a
// global <element> (#340), always with an ElementDeclarationContext naming the
// declaration it is building; see ComplexTypeContext.
func NewAnonymousComplexType(loc xsderr.Loc, context ComplexTypeContext, baseTypeDefinitionName QName, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, prohibitedAttributeNames []QName, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
	if context == nil {
		return ComplexType{}, xsderr.New(ruleCTPropsCorrect, loc,
			"anonymous complex type definition has an absent {context}, but the §3.4.1 tableau requires it to be present when {name} is absent (ct-props-correct clause 1)")
	}
	if err := checkComplexTypeContext(loc, context); err != nil {
		return ComplexType{}, err
	}
	return newComplexType(loc, QName{}, context, baseTypeDefinitionName, final, derivationMethod, abstract, attributeUses, prohibitedAttributeNames, attributeWildcard, contentType, prohibitedSubstitutions, assertions, annotations)
}

// newComplexType is the shared core of NewComplexType and
// NewAnonymousComplexType: every check and copy that does not concern the
// {name}/{context} pair lives here exactly once (STYLE T4).
//
// PRECONDITION, enforced by its TWO CALLERS and not by itself: exactly one of
// name and context is present, per the §3.4.1 tableau's XOR. This layer can
// express both-present and neither-present and does not reject either; the
// partition's guarantee comes from the exported entry points, each of which
// accepts only one half of the pair. Any third caller added here must
// re-establish the XOR itself.
func newComplexType(loc xsderr.Loc, name QName, context ComplexTypeContext, baseTypeDefinitionName QName, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, prohibitedAttributeNames []QName, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
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
		loc:                    loc,
		name:                   name,
		context:                context,
		baseTypeDefinitionName: baseTypeDefinitionName,
		derivationMethod:       derivationMethod,
		abstract:               abstract,
		contentType:            contentType,
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

// BaseTypeDefinitionName returns the {base type definition} property (Required)
// as a pre-resolution QName reference.
//
// This is NOT the resolved {base type definition} component (§3.4.1). Finalize
// (#173) validates the name resolves to a type definition (src-resolve clause
// 1.1) and that the base chain is acyclic (ct-props-correct clause 3), but adds
// no resolved-component accessor: the QName is retained, and a consumer obtains
// the component by a read-time schema.Type(name) lookup, mirroring the
// TypeDefinitionRef arm of ElementDeclaration.TypeDefinition. A base is always
// reachable by name here — a complex type's {base type definition} is never the
// inline anonymous kind this producer builds — so this slot stays a bare QName
// rather than a TypeDefinitionOrRef.
func (c ComplexType) BaseTypeDefinitionName() QName {
	return c.baseTypeDefinitionName
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
// On a NAMED type reached through a finalized [Schema] this is the §3.4.2.4
// clause 3 property — the type's own uses followed by those inherited from its
// {base type definition}, less what clauses 3.2.1 and 3.2.2 exclude — because
// Finalize materialises the fold (attributeusefold.go, #401). On a ComplexType a
// caller built with [NewComplexType] and has not yet finalized, it is only what
// that caller passed in: clause 3 needs the base COMPONENT, which a standalone
// value has no way to reach.
//
// GAP(xsd): the fold walks the finalized Schema's TYPE DEFINITIONS only, so an
// anonymous complex type owned by a declaration (an InlineTypeDefinition, at top
// level or nested in a particle tree) is not folded and reports its own uses
// alone. The parser reaches that shape as of #340, for the inline <complexType>
// child of a local or a global <element>; a caller assembling a Schema through
// [SchemaBuilder] reaches it too. The parser's own shape is provably unaffected
// — conformance/schema.go admits only the IMPLICIT-CONTENT form, whose base is
// xs:anyType, whose {attribute uses} §3.4.7 makes empty, so the unrun fold is
// the identity — but the widening itself is #414's, one shape with two sites.
// Under-reporting the set is the direction the whole fold's absence had before
// #401: it can withhold a member, never fabricate one.
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
// Finalize materialises the fold (attributewildcardfold.go, #265). On a
// ComplexType a caller built with [NewComplexType] and has not yet finalized, it
// is only what that caller passed in: clause 2.2 needs the base COMPONENT, which a
// standalone value has no way to reach.
//
// GAP(xsd): the {attribute wildcard} half of the seam AttributeUses records for
// {attribute uses} — ONE unfolded shape with two sites, not two gaps, tracked as
// #414. Both folds walk the finalized Schema's TYPE DEFINITIONS only, so an
// anonymous complex type owned by a declaration (an InlineTypeDefinition) is
// folded for neither property and reports its own <anyAttribute> alone. The
// parser reaches that shape as of #340, in the implicit-content form alone — see
// AttributeUses for why that form makes the unrun fold the identity — and a
// caller assembling a Schema through [SchemaBuilder] can nest an inline complex
// type that extends a base carrying a wildcard. Phase D quantifies over
// the same type definitions, so no constraint in this package reads the unfolded
// value; the exposure is a READING consumer's, and under-reporting is the
// direction the whole fold's absence had before #265 — it can withhold admitted
// names, never fabricate them.
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
