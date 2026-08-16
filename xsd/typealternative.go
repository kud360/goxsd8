package xsd

import "github.com/kud360/goxsd8/xsderr"

// TypeAlternative is the Type Alternative component (Structures §3.12.1,
// id="tac"): a kind of Annotated Component with {annotations} (a sequence of
// Annotation), {test} (an XPathExpression property record, Optional — unlike
// Assertion's {test}, which is Required) and {type definition} (Required).
//
// A Type Alternative is one entry of the ordered {alternatives} list on an
// element declaration's {type table} (§3.3.2.1): §3.12.4's conditional type
// assignment picks, per instance element, the first alternative whose {test}
// is true, and the alternative's {type definition} governs that element.
//
// Like Assertion, TypeAlternative is a STRUCTURAL, opaque holder: {test} is
// preserved verbatim by the embedded XPathExpression (see its doc), never
// compiled or evaluated here. §3.12.2 delegates {test}'s XML mapping verbatim
// to §3.13.2 (the same XPath Expression property record Assertion uses), so the
// two components reuse xsd.XPathExpression. Evaluation (cvc-type-alternative,
// cvc-cta-ta-select) is deferred to the M6/M7 XPath engine and is out of scope
// here.
//
// {type definition} is the TypeDefinitionOrRef sum, the same slot an element or
// attribute declaration carries, because §3.12.2 declare-ta has the same two
// arms an element declaration's own {type definition} does: the type= attribute
// names a top-level definition (TypeDefinitionRef) and an inline <simpleType>/
// <complexType> child yields an anonymous one this slot OWNS
// (InlineTypeDefinition). The third arm reaches it through the SYNTHESIZED
// {default type definition}, which §3.3.2.1 copies verbatim from the declaring
// element's own slot; see typeAlternativeTypeSlot.
//
// Construct only through NewTypeAlternative. TypeAlternative is immutable
// after construction.
type TypeAlternative struct {
	test           XPathExpression
	hasTest        bool
	typeDefinition TypeDefinitionOrRef
	annotations    []Annotation
}

// NewTypeAlternative builds a TypeAlternative. test == nil means {test} is
// absent (the default/"otherwise" alternative — legal only as the last
// element of the containing element declaration's ordered alternatives list;
// src-element clause 5 (§3.3.3) is not charged anywhere in this processor yet
// — parser/produce_typetable.go's typeTableOf documents the gap). A pointer
// to a (possibly empty) XPathExpression means {test} is present, because an
// empty {expression} is a legal present value (see NewXPathExpression's doc) —
// so absence cannot collapse into a zero record and needs its own flag,
// mirroring hasDefaultNamespace/hasBaseURI. annotations is copied; the
// caller's backing array is not aliased.
//
// It rejects an ABSENT (nil) typeDefinition, charged to
// xsderr.RuleComponentInvariant: §3.12.1's tableau types {type definition} as
// Required, and nil is TypeDefinitionOrRef's one encoding of an absent slot, so
// a nil here is an illegal representation rather than a deferred resolution
// (the footing NewAnonymousComplexType rejects an absent {context} on;
// ta-props-correct §3.12.6 is about the XPath subset and states nothing here).
// Every other illegal encoding of the slot is rejected by
// checkTypeDefinitionOrRef, which decides the arm × slot table in one place.
//
// loc is the source position charged to any rejection. A caller with no real
// parser position — a synthesized or programmatically built alternative — may
// legitimately pass the zero xsderr.Loc{}; the position is not retained, since
// §3.12.1 has no provenance property and the owning <element>'s position is
// what resolveTypeTable charges.
func NewTypeAlternative(loc xsderr.Loc, test *XPathExpression, typeDefinition TypeDefinitionOrRef, annotations []Annotation) (TypeAlternative, error) {
	if typeDefinition == nil {
		return TypeAlternative{}, xsderr.New(xsderr.RuleComponentInvariant, loc,
			"type alternative has an absent {type definition}, but the §3.12.1 tableau types it as Required")
	}
	if err := checkTypeDefinitionOrRef(loc, typeDefinition, typeAlternativeTypeSlot, "type alternative"); err != nil {
		return TypeAlternative{}, err
	}
	t := TypeAlternative{typeDefinition: typeDefinition}
	if test != nil {
		t.test, t.hasTest = *test, true
	}
	if len(annotations) > 0 {
		t.annotations = append([]Annotation(nil), annotations...)
	}
	return t, nil
}

// Test returns the {test} property (Optional): an XPath Expression property
// record (Structures §3.12.2, delegating to §3.13.2's mapping — the same
// shape Assertion.Test() uses). The second result is false when {test} is
// absent: this is the default/"otherwise" alternative, legal only as the last
// element of the containing ordered list (src-element clause 5 is unenforced,
// not enforced elsewhere — see NewTypeAlternative's doc), in which case the
// first result is not meaningful.
func (t TypeAlternative) Test() (XPathExpression, bool) {
	return t.test, t.hasTest
}

// TypeDefinition returns the {type definition} property (Required) as the
// TypeDefinitionOrRef slot §3.12.2 declare-ta populates: a TypeDefinitionRef
// for the type= arm, an InlineTypeDefinition owning the anonymous type of a
// <simpleType>/<complexType> child, or — reached only through a synthesized
// {default type definition} — a SubstitutionGroupHeadTypeRef.
//
// Resolve it through Schema.ResolvedType, the one reader that answers for all
// three arms; it may name either a simple or a complex type definition
// (§3.12.1).
func (t TypeAlternative) TypeDefinition() TypeDefinitionOrRef {
	return t.typeDefinition
}

// Annotations returns the {annotations} property in document order. It returns
// a copy: mutating the result does not affect t. An empty {annotations} yields
// nil.
func (t TypeAlternative) Annotations() []Annotation {
	if len(t.annotations) == 0 {
		return nil
	}
	return append([]Annotation(nil), t.annotations...)
}
