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
//
// The substantive, cross-component clauses are NOT enforced here — this
// constructor is deliberately not the full property-correctness check:
//
//   - clause 1's resolved-component parts and clause 2 ({base type definition} a
//     simple type forces {derivation method} = extension) need the {base type
//     definition} RESOLVED, which this constructor does not do. Clause 3 (no
//     circular {base type definition} chain except xs:anyType) IS enforced, but
//     at finalize (resolve.go's checkComplexBaseAcyclic, #173) — it needs the
//     whole base graph, which only exists once the schema set is assembled;
//   - clause 4 (no two {attribute uses} share an {attribute declaration}
//     expanded name) is left to the producer/finalize layer for this component
//     (unlike AttributeGroupDefinition, whose own ag-props-correct clause 2 is
//     enforced at shape time);
//   - clause 5 ({content type}.{open content} non-absent ⇒ {variety} is
//     element-only or mixed) is satisfied BY CONSTRUCTION: {open content} is a
//     field only of ElementContent, whose {variety} is always element-only or
//     mixed, so the forbidden state is unrepresentable.
//
// The derivation-validity rules cos-ct-extends (§3.4.6.2), derivation-ok-restriction
// (§3.4.6.3), and cos-content-act-restrict (§3.4.6.4) are cross-component
// finalize-phase concerns and are likewise NOT touched here; they are deferred
// to the producer (#176) and finalize (#173/#181).
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

// ComplexType is the Complex Type Definition component (Structures §3.4.1,
// id="Complex_Type_Definition_details"): a kind of Type Definition with
// {annotations}, {name} (bundled with {target namespace} as an xsd.QName per
// this package's "Names are expanded QNames" convention — doc.go; the zero
// QName is an anonymous complex type), {base type definition}, {final},
// {derivation method}, {abstract}, {attribute uses}, {attribute wildcard}
// (Optional), {content type}, {prohibited substitutions}, and {assertions}.
//
// Like the other §3 component shapes in this package, ComplexType is a
// STRUCTURAL holder built before resolution. {base type definition} is carried
// as a pre-resolution QName REFERENCE (baseTypeDefinitionName), not a resolved
// simple-or-complex type. Finalize (resolve.go, #173) VALIDATES that the
// reference resolves to a type definition (src-resolve clause 1.1) and that the
// complex-type base chain is acyclic except xs:anyType's self-derivation
// (ct-props-correct clause 3), but does NOT rewrite it into a resolved
// component: the QName is retained, and a consumer follows it by a read-time
// schema.Type(name) lookup. Clause 2 and clause 1's resolved parts, plus the
// derivation-validity rules (cos-ct-extends §3.4.6.2, derivation-ok-restriction
// §3.4.6.3), stay deferred to the producer (#176) and later finalize work.
//
// {context} (§3.4.1 ctd-context — the component an anonymous type appears in) is
// entirely UNMODELED by this issue, exactly as ElementDeclaration leaves
// {scope}.{parent} unmodeled: the containing declaration/type that would be the
// {context} is not wired to this component yet, so an anonymous ComplexType is
// structurally incomplete in that one respect. The gap is named here rather than
// buried.
//
// Ratchet impact: unchanged. This is a leaf shape with no parser producer; the
// schema conformance lane moves only when the producer (#176) wires it in.
//
// Construct only through NewComplexType, which rejects the tableau-shape states
// ct-props-correct (§3.4.6.1) clause 1 forbids so they are unrepresentable
// (STYLE T1). It is NOT the full property-correctness check (see
// ruleCTPropsCorrect's doc for exactly which clauses are deferred). ComplexType
// is immutable after construction.
type ComplexType struct {
	name                    QName
	baseTypeDefinitionName  QName
	derivationMethod        DerivationMethod
	final                   []DerivationMethod
	abstract                bool
	attributeUses           []AttributeUse
	attributeWildcard       Wildcard
	hasAttributeWildcard    bool
	contentType             ContentType
	prohibitedSubstitutions []DerivationMethod
	assertions              []Assertion
	annotations             []Annotation
}

// NewComplexType builds a ComplexType, rejecting the tableau-shape states
// Complex Type Definition Properties Correct (§3.4.6.1, ct-props-correct)
// clause 1 forbids:
//
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
// The substantive cross-component clauses (base-type resolution, circularity,
// attribute-use expanded-name uniqueness) and the derivation-validity rules are
// NOT checked here — see ruleCTPropsCorrect's doc.
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
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built definition — may
// legitimately pass the zero xsderr.Loc{}.
func NewComplexType(loc xsderr.Loc, name QName, baseTypeDefinitionName QName, final []DerivationMethod, derivationMethod DerivationMethod, abstract bool, attributeUses []AttributeUse, attributeWildcard *Wildcard, contentType ContentType, prohibitedSubstitutions []DerivationMethod, assertions []Assertion, annotations []Annotation) (ComplexType, error) {
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
	c := ComplexType{
		name:                   name,
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
// The zero QName denotes an anonymous complex type (§3.4.1).
func (c ComplexType) Name() QName {
	return c.name
}

// BaseTypeDefinitionName returns the {base type definition} property (Required)
// as a pre-resolution QName reference.
//
// This is NOT the resolved {base type definition} component (§3.4.1). Finalize
// (#173) validates the name resolves to a type definition (src-resolve clause
// 1.1) and that the base chain is acyclic (ct-props-correct clause 3), but adds
// no resolved-component accessor: the QName is retained, and a consumer obtains
// the component by a read-time schema.Type(name) lookup, mirroring
// ElementDeclaration.TypeDefinitionName.
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
func (c ComplexType) AttributeUses() []AttributeUse {
	if len(c.attributeUses) == 0 {
		return nil
	}
	return append([]AttributeUse(nil), c.attributeUses...)
}

// AttributeWildcard returns the {attribute wildcard} property (Optional); the
// second result is false when it is absent, in which case the first result is
// not meaningful.
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
