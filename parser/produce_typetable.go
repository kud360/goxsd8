package parser

import (
	"slices"

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
// definition}, which §3.3.2.1 makes the {default type definition}'s {type
// definition} whenever the final <alternative> carries a test.
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
func (p *producer) typeTableOf(el *Element, declaredType xsd.TypeDefinitionOrRef) (*xsd.TypeTable, error) {
	alternatives := childElements(el, xsd.XMLSchemaNS, "alternative")
	if len(alternatives) == 0 {
		return nil, nil // §3.3.2.1: "otherwise ·absent·"
	}
	for _, alt := range alternatives {
		if err := checkSrcTA(alt); err != nil {
			return nil, err
		}
	}
	names, err := p.alternativeTypeNames(alternatives)
	if err != nil {
		return nil, err
	}
	last := len(alternatives) - 1
	_, lastTested := alternatives[last].Attr("test")
	if !typeTableRepresentable(names, declaredType, lastTested) {
		return nil, nil
	}
	dflt := xsd.NewTypeAlternative(nil, names[last], nil) // §3.3.2.1 case 1
	if lastTested {
		declaredName, _ := typeDefinitionRefName(declaredType) // named: typeTableRepresentable said so
		dflt = xsd.NewTypeAlternative(nil, declaredName, nil)  // §3.3.2.1 case 2
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
		alts = append(alts, xsd.NewTypeAlternative(&test, names[i], nil))
	}
	tt, err := xsd.NewTypeTable(el.Loc(), alts, dflt)
	if err != nil {
		return nil, err
	}
	return &tt, nil
}

// alternativeTypeNames resolves the type attribute of each <alternative> in
// document order (declare-ta, §3.12.2: "the type definition ·resolved· to by
// the ·actual value· of the type attribute"). An <alternative> taking the
// inline arm has no such attribute and gets the zero QName, which
// typeTableRepresentable reads as the withheld arm.
func (p *producer) alternativeTypeNames(alternatives []*Element) ([]xsd.QName, error) {
	names := make([]xsd.QName, len(alternatives))
	for i, alt := range alternatives {
		typeLex, hasType := alt.Attr("type")
		if !hasType {
			continue
		}
		qn, err := p.resolveQName(alt, typeLex, "type")
		if err != nil {
			return nil, err
		}
		names[i] = qn
	}
	return names, nil
}

// typeTableRepresentable reports whether every Type Alternative the table needs
// can be expressed by the xsd.TypeAlternative this producer has to hand. names
// are the resolved alternative type names in document order, zero where the
// <alternative> took the inline arm; lastTested says whether the final
// <alternative> carries a test attribute and so leaves the {default type
// definition} to be synthesized from declaredType.
//
// GAP(xsd): two shapes are not mapped, and each withholds the WHOLE {type
// table} of the declaration rather than one entry of it, because
// xsd.TypeAlternative carries its {type definition} as a QName REFERENCE
// (typealternative.go) and a table short of an entry is worse than none:
// §3.8.6.3 key-equiv-tt pairs {alternatives} by POSITION
// (xsd/elementconsistent.go typeTablesEquivalent), so a short list mis-pairs two
// declarations the schema document makes agree.
//
//   - an <alternative> taking §3.12.2's INLINE arm, a <simpleType> or
//     <complexType> child in place of a type attribute: an anonymous type has no
//     name to carry;
//   - a synthesized {default type definition} whose declaring <element> has an
//     inline type of its own, for the same reason.
//
// Both shapes are fail-CLOSED over their four readers, which do not all charge
// in the same direction:
//
//   - validate/assess.go walk.governingType returns nil for a declaration that
//     HAS a table and D.{type definition} for one that has none, so a withheld
//     table assesses the element against its DECLARED type where an
//     <alternative> would ·conditionally select· another — a FALSE REJECT of
//     content the selected type admits, at the ·validation root· and at every
//     descendant. This reader alone makes the whole set fail-CLOSED.
//   - xsd/elementconsistent.go checkTypeTablesAgree and xsd/defaultbinding.go
//     typeTablesAgree both read "all ·absent· or all present and ·equivalent·".
//     Two withheld tables read as agreeing, which is an UNMADE
//     cos-element-consistent / loc-testSubP clause 4.6 rejection; but a withheld
//     table beside a mapped one reads as DISAGREEING, which is a FALSE REJECT of
//     two declarations that agree in the schema document — reachable only where
//     one of a same-named pair takes a withheld shape and the other does not.
//   - xsd/resolve.go resolveTypeTable has nothing to resolve for a withheld
//     table, so a dangling alternative type name goes uncharged (src-resolve
//     clause 1.1): an unmade rejection.
//
// Closing both means growing xsd.TypeAlternative the xsd.TypeDefinitionOrRef
// sum AttributeDeclaration and ElementDeclaration already carry, which first
// needs §3.4.2.1's ownership invariant settled for an <element> that has both a
// type attribute and an inline alternative: the anonymous type's {context} is
// the enclosing element declaration, a shape NewElementDeclarationOwningType's
// identity check does not cover (#822).
func typeTableRepresentable(names []xsd.QName, declaredType xsd.TypeDefinitionOrRef, lastTested bool) bool {
	if slices.Contains(names, xsd.QName{}) {
		return false
	}
	if !lastTested {
		return true
	}
	_, named := typeDefinitionRefName(declaredType)
	return named
}

// typeDefinitionRefName reports the QName an element declaration's {type
// definition} references, and false when the declaration owns an ANONYMOUS type
// instead — an inline <simpleType> or <complexType> child, which has no name a
// Type Alternative could carry.
func typeDefinitionRefName(declaredType xsd.TypeDefinitionOrRef) (xsd.QName, bool) {
	ref, named := declaredType.(xsd.TypeDefinitionRef)
	if !named {
		return xsd.QName{}, false
	}
	return ref.Name, true
}

// checkSrcTA charges Type Alternative Representation OK (§3.12.3) for an
// <alternative> carrying none, or more than one, of a type attribute, a
// <complexType> child and a <simpleType> child.
//
// It counts the two INLINE forms as present even though typeTableRepresentable
// declines to map them: src-ta is a constraint on the XML representation, not on
// what this producer builds from it, so an <alternative> holding exactly one
// inline type child satisfies it outright. Reading "no type attribute" as "no
// form present" would reject every conforming inline-arm schema document.
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
