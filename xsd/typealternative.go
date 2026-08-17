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
// {type definition} is the TypeDefinitionOrRef sum an element or attribute
// declaration's own {type definition} carries, because §3.12.2 declare-ta
// populates it from the same two kinds of source: the type attribute, whose
// ·resolved· component is reachable by name, and the <complexType>/<simpleType>
// child, whose anonymous component this slot OWNS. §3.3.2.1's synthesized
// {default type definition} adds the third arm — it is "the {type definition}
// property of the parent Element Declaration", copied verbatim, so a member of a
// substitution group inherits its head's type through this slot too.
//
// {test} by contrast is STRUCTURAL and opaque: it is preserved verbatim by the
// embedded XPathExpression (see its doc) and never compiled here. §3.12.2
// delegates {test}'s XML mapping verbatim to §3.13.2 (the same XPath Expression
// property record Assertion uses), so the two components reuse
// xsd.XPathExpression. Evaluation (cvc-type-alternative, cvc-cta-ta-select) is
// validate/cta.go's, which reads the slot through Schema.ResolvedType.
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
// typeDefinition is Required by the §3.12.1 tableau, so a nil slot — the sum's
// encoding of an ABSENT property — is rejected, charged to
// xsderr.RuleComponentInvariant on the footing NewAnonymousComplexType rejects
// an absent {context} on: a Required property left empty is this package's
// representation invariant, not a clause a schema author can violate. The
// illegal encodings WITHIN the sum are rejected by checkTypeDefinitionOrRef,
// which admits the SubstitutionGroupHeadTypeRef arm here because §3.3.2.1's
// synthesized {default type definition} copies the declaring element's own slot
// whatever arm it holds.
//
// loc is the source position charged to a rejection. A caller with no real
// parser position — a synthesized or programmatically built alternative — may
// legitimately pass the zero xsderr.Loc{}.
func NewTypeAlternative(loc xsderr.Loc, test *XPathExpression, typeDefinition TypeDefinitionOrRef, annotations []Annotation) (TypeAlternative, error) {
	if typeDefinition == nil {
		return TypeAlternative{}, xsderr.New(xsderr.RuleComponentInvariant, loc,
			"type alternative {type definition} is absent, but the §3.12.1 tableau types it as Required and §3.12.2 declare-ta populates it from the type attribute or from the <complexType>/<simpleType> child")
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
// TypeDefinitionOrRef slot §3.12.2 declare-ta populated: a TypeDefinitionRef for
// the type attribute's ·resolved· component, an InlineTypeDefinition for the
// <complexType>/<simpleType> child this alternative owns, or — on a synthesized
// {default type definition} alone — the SubstitutionGroupHeadTypeRef the
// declaring element's own slot held. Follow it with Schema.ResolvedType, which
// answers for all three arms; never re-derive a by-name lookup of your own.
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
