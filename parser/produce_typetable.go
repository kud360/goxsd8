package parser

import (
	"fmt"

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
// for the ·absent· property. edID is that declaration's minted identity, the
// {context} every anonymous type an <alternative> owns points back at, and
// declaredType is its own {type definition}, which §3.3.2.1 makes the {default
// type definition}'s {type definition} whenever the final <alternative> carries
// a test.
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
//     definition}, and an empty {annotations}. §3.3.2.1 makes that synthesized
//     slot "the {type definition} property of the parent Element Declaration",
//     so declaredType is passed through WHOLE — the same component, not a copy,
//     and whichever arm it holds, including the SubstitutionGroupHeadTypeRef a
//     substitutionGroup=-typed element carries.
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
		ta, terr := xsd.NewTypeAlternative(alt.Loc(), &test, types[i], nil)
		if terr != nil {
			return nil, terr
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
// ·resolved· to by the ·actual value· of the type attribute, if one is present,
// otherwise the type definition corresponding to the complexType or simpleType
// among the children of the <alternative> element". checkSrcTA has already passed
// over every child, so exactly one of the three forms is present; the last arm
// reports the mismatch as a plain producer fault rather than a fabricated rule
// verdict, on the same footing newComplexType's redefining arm does.
//
// The inline arm OWNS the anonymous type it yields, so it is an
// xsd.InlineTypeDefinition, and the anonymous COMPLEX type carries two identities
// (typeAlternativeOwnedComplexType): edID, the enclosing element declaration
// §3.4.2.1 dcl.ctd.common makes its {context} — the <alternative> is not an
// <element>, so the ancestor walk passes straight over it — and a container token
// minted per <alternative>, which its own nested local declarations report as
// {scope}.{parent}. An anonymous SIMPLE type goes through constructSimpleType,
// the same entry point an inline <simpleType> child of an <element> or
// <attribute> takes, and its {context} (§3.16.1 std-context) is unmodeled here as
// it is everywhere else in this producer (#206).
//
// The anonymous complex type this builds is the FOURTH owning slot that receives
// no Phase D verdict and neither §3.4.2.4 nor §3.4.2.5 attribute fold; the
// direction argument and the trackers are stated once, at the GAP marker in
// xsd/complexderivation.go's checkComplexDerivations.
func (p *producer) alternativeTypes(alternatives []*Element, edID xsd.ComponentID) ([]xsd.TypeDefinitionOrRef, error) {
	types := make([]xsd.TypeDefinitionOrRef, len(alternatives))
	for i, alt := range alternatives {
		typeLex, hasType := alt.Attr("type")
		if hasType {
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
		inlineSimple := childElement(alt, xsd.XMLSchemaNS, "simpleType")
		if inlineSimple == nil {
			return nil, fmt.Errorf("parser: the <alternative> at %s carries none of a type attribute, a <complexType> child and a <simpleType> child, but src-ta requires exactly one and checkSrcTA charges it before this mapping runs", alt.Loc())
		}
		st, err := p.constructSimpleType(xsd.QName{}, inlineSimple)
		if err != nil {
			return nil, err
		}
		types[i] = xsd.InlineTypeDefinition{Definition: st}
	}
	return types, nil
}

// checkSrcTA charges Type Alternative Representation OK (§3.12.3) for an
// <alternative> carrying none, or more than one, of a type attribute, a
// <complexType> child and a <simpleType> child.
//
// It runs BEFORE alternativeTypes maps any child, so the "exactly one form" it
// establishes is what that mapping's arm order relies on: src-ta is a constraint
// on the XML representation, and an <alternative> holding exactly one inline type
// child satisfies it outright.
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
