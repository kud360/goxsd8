package xsd

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
// compiled or evaluated here, and {type definition} is carried as a QName
// REFERENCE, not the resolved component. §3.12.2 delegates {test}'s XML
// mapping verbatim to §3.13.2 (the same XPath Expression property record
// Assertion uses), so the two components reuse xsd.XPathExpression. Evaluation
// (cvc-type-alternative, cvc-cta-ta-select) is deferred to the M6/M7 XPath
// engine and is out of scope here.
//
// Construct only through NewTypeAlternative. TypeAlternative is immutable
// after construction.
type TypeAlternative struct {
	test               XPathExpression
	hasTest            bool
	typeDefinitionName QName
	annotations        []Annotation
}

// NewTypeAlternative builds a TypeAlternative. test == nil means {test} is
// absent (the default/"otherwise" alternative — legal only as the last
// element of the containing element declaration's ordered alternatives list;
// enforced by src-element clause 5 (§3.3.3), not by this component). A pointer
// to a (possibly empty) XPathExpression means {test} is present, because an
// empty {expression} is a legal present value (see NewXPathExpression's doc) —
// so absence cannot collapse into a zero record and needs its own flag,
// mirroring hasDefaultNamespace/hasBaseURI. annotations is copied; the
// caller's backing array is not aliased. There is no rejectable state at this
// structural layer, mirroring NewAssertion — hence no loc/error.
func NewTypeAlternative(test *XPathExpression, typeDefinitionName QName, annotations []Annotation) TypeAlternative {
	t := TypeAlternative{typeDefinitionName: typeDefinitionName}
	if test != nil {
		t.test, t.hasTest = *test, true
	}
	if len(annotations) > 0 {
		t.annotations = append([]Annotation(nil), annotations...)
	}
	return t
}

// Test returns the {test} property (Optional): an XPath Expression property
// record (Structures §3.12.2, delegating to §3.13.2's mapping — the same
// shape Assertion.Test() uses). The second result is false when {test} is
// absent: this is the default/"otherwise" alternative, legal only as the last
// element of the containing ordered list (enforced upstream at src-element,
// not here), in which case the first result is not meaningful.
func (t TypeAlternative) Test() (XPathExpression, bool) {
	return t.test, t.hasTest
}

// TypeDefinitionName returns the {type definition} property (Required) as a
// QName reference — the pre-resolution name from §3.12.3's type/complexType/
// simpleType alternatives.
//
// This is NOT the resolved {type definition} component (§3.12.1), which may
// name either a simple or a complex type definition. The resolved component
// accessor, and its resolution, are deferred to the future finalize-phase
// issue that first introduces phased construction (per doc.go's "parse →
// resolve → finalize"); nothing in this package resolves it yet.
func (t TypeAlternative) TypeDefinitionName() QName {
	return t.typeDefinitionName
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
