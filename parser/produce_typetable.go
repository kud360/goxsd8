package parser

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file maps the <alternative> children of an <element> into the element
// declaration's {type table} (§3.3.2.1 dcl.elt.common over §3.12.2 declare-ta)
// and charges §3.12.3 src-ta over the same children. It serves the GLOBAL
// <element> path (produceElement) and both LOCAL ones (produceLocalElement)
// from one implementation (STYLE T4): §3.3.2.1's {type table} row is a COMMON
// mapping rule and §3.3.2.2 dcl.elt.global supplements {target namespace} and
// {scope} alone, so a top-level <element> has nothing to map differently.

// ruleSrcTA is Type Alternative Representation OK (Structures §3.12.3,
// id="src-ta"): "each <alternative> element must have one (and only one) of the
// following: a type attribute, or a complexType child element, or a simpleType
// child element". The rule is a single unnumbered sentence, so messages cite no
// clause.
const ruleSrcTA xsderr.Rule = "src-ta"

// typeTableOf maps the <alternative> children of el into the {type table} of
// the element declaration el produces (§3.3.2.1 dcl.elt.common), returning nil
// for the ·absent· property. declaredType is that declaration's own {type
// definition} SLOT, which §3.3.2.1 makes the {default type definition}'s {type
// definition} — verbatim, whichever arm it is — whenever the final
// <alternative> carries a test.
//
// edID is the enclosing element declaration's minted identity, threaded in
// because §3.4.2.1 dcl.ctd.common makes it the {context} of every anonymous
// <complexType> an <alternative> here owns (typeAlternativeOwnedComplexType);
// the caller mints it before either component exists and passes the same value
// to xsd.NewElementDeclarationOwningTypes.
//
// The mapping, once src-ta has passed over every <alternative>:
//
//   - {alternatives} takes the <alternative> children WITH a test attribute, in
//     document order, each through declare-ta. A TRAILING untested one is not
//     among them — it feeds {default type definition} instead — and an untested
//     one anywhere else maps to nothing at all, which is the mapping as written
//     rather than an omission: {alternatives} is defined over the tested
//     children and {default type definition} over the final child, so a
//     non-final untested <alternative> is named by neither. src-element clause
//     5 (§3.3.3) forbids that document, and this producer does not charge it.
//   - {default type definition} is the final <alternative> through declare-ta
//     when it has no test attribute, and otherwise a Type Alternative
//     SYNTHESIZED with an ·absent· {test}, declaredType as its {type
//     definition}, and an empty {annotations}.
//
// {annotations} is the empty sequence on every Type Alternative built here,
// matching every other component this producer emits: no <annotation> is mapped
// anywhere in this package yet.
func (p *producer) typeTableOf(el *Element, edID xsd.ComponentID, declaredType xsd.TypeDefinitionOrRef) (*xsd.TypeTable, error) {
	alternatives := childElements(el, xsd.XMLSchemaNS, "alternative")
	if len(alternatives) == 0 {
		return nil, nil // §3.3.2.1: "otherwise ·absent·"
	}
	for _, alt := range alternatives {
		if err := checkSrcTA(alt); err != nil {
			return nil, err
		}
	}
	types, err := p.alternativeTypes(alternatives, edID)
	if err != nil {
		return nil, err
	}
	last := len(alternatives) - 1
	defaultType := types[last] // §3.3.2.1 case 1
	if _, lastTested := alternatives[last].Attr("test"); lastTested {
		defaultType = declaredType // §3.3.2.1 case 2
	}
	dflt, err := xsd.NewTypeAlternative(el.Loc(), nil, defaultType, nil)
	if err != nil {
		return nil, err
	}
	var alts []xsd.TypeAlternative
	for i, alt := range alternatives {
		// The test attribute is the whole membership rule, so a TRAILING untested
		// <alternative> drops out here exactly as a non-final one does — it is
		// already the {default type definition} above.
		if _, hasTest := alt.Attr("test"); !hasTest {
			continue
		}
		// declare-ta: {test} is the XPath Expression property record §3.13.2
		// builds for the <alternative> host element and its test attribute — the
		// same record and the same xpathDefaultNamespace chain an <assert> gets,
		// which is why this reuses buildXPathExpression rather than restating it.
		test := p.buildXPathExpression(alt, "test")
		ta, err := xsd.NewTypeAlternative(alt.Loc(), &test, types[i], nil)
		if err != nil {
			return nil, err
		}
		alts = append(alts, ta)
	}
	tt, err := xsd.NewTypeTable(el.Loc(), alts, dflt)
	if err != nil {
		return nil, err
	}
	return &tt, nil
}

// alternativeTypes maps the {type definition} of each <alternative> in document
// order, over both arms §3.12.2 declare-ta states: "the type definition
// ·resolved· to by the ·actual value· of the type attribute", or "the type
// definition corresponding to the complexType or simpleType child".
//
// src-ta has already passed, so exactly one arm is present on each child and the
// returned slot is never nil.
//
// The inline <complexType> arm mints a container token per <alternative>
// (newTypeAlternativeOwned) and takes edID as the type's {context}; the inline
// <simpleType> arm goes through the one anonymous-simple-type entry point
// constructSimpleType, whose §3.16.1 {context} (std-context clause 2.4, the
// <alternative> parent) is unmodeled here exactly as it is for an inline
// <simpleType> child of an <element> (#206).
//
// An anonymous complex type owned here inherits the withheld finalize verdicts
// every owner-reachable anonymous type has — the whole of Phase D and the two
// attribute folds — with the direction argument and the issues that own it in
// xsd/complexderivation.go's GAP marker (#584/#414/#438).
func (p *producer) alternativeTypes(alternatives []*Element, edID xsd.ComponentID) ([]xsd.TypeDefinitionOrRef, error) {
	types := make([]xsd.TypeDefinitionOrRef, len(alternatives))
	for i, alt := range alternatives {
		if typeLex, hasType := alt.Attr("type"); hasType {
			qn, err := p.resolveQName(alt, typeLex, "type")
			if err != nil {
				return nil, err
			}
			types[i] = xsd.TypeDefinitionRef{Name: qn}
			continue
		}
		if inlineComplex := childElement(alt, xsd.XMLSchemaNS, "complexType"); inlineComplex != nil {
			ct, err := p.produceComplexType(newTypeAlternativeOwned(edID), inlineComplex)
			if err != nil {
				return nil, err
			}
			types[i] = xsd.InlineTypeDefinition{Definition: ct}
			continue
		}
		st, err := p.constructSimpleType(xsd.QName{}, childElement(alt, xsd.XMLSchemaNS, "simpleType"))
		if err != nil {
			return nil, err
		}
		types[i] = xsd.InlineTypeDefinition{Definition: st}
	}
	return types, nil
}

// checkSrcTA charges Type Alternative Representation OK (§3.12.3) for an
// <alternative> carrying none, or more than one, of a type attribute, a
// <complexType> child and a <simpleType> child. It runs over every
// <alternative> before any of them is mapped, because alternativeTypes reads the
// exactly-one shape it establishes.
func checkSrcTA(alt *Element) error {
	forms := 0
	if _, hasType := alt.Attr("type"); hasType {
		forms++
	}
	if childElement(alt, xsd.XMLSchemaNS, "complexType") != nil {
		forms++
	}
	if childElement(alt, xsd.XMLSchemaNS, "simpleType") != nil {
		forms++
	}
	if forms == 1 {
		return nil
	}
	return xsderr.New(ruleSrcTA, alt.Loc(),
		"<alternative> has %d of a type attribute, a <complexType> child and a <simpleType> child, but src-ta requires exactly one", forms)
}
