package parser

import (
	"fmt"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file maps the XPath Expression property record (§3.13.1, id="x") and the
// two component families built on top of it: Identity-Constraint Definitions
// (§3.11.2, whose <selector>/<field> xpath attributes reuse the record verbatim)
// and Assertions (§3.13.2's <assert> on a complex type, Datatypes §4.3.13.2's
// <assertion> facet on a simple-type restriction — two different host elements,
// two different component targets, one construction algorithm).

// buildXPathExpression maps an XPath-bearing host element — <selector>, <field>,
// <assert>, or <assertion> — and the name of its expression attribute ("xpath"
// or "test") to an XPath Expression property record (§3.13.2):
//
//   - {expression} is the attribute's actual value, held verbatim.
//   - {namespace bindings} is one Namespace Binding per PREFIXED in-scope
//     namespace of the host element (§3.13.1 id="nb": {prefix} is a Required
//     xs:NCName, so the default-namespace entry cannot be one of them; it feeds
//     {default namespace} instead).
//   - {default namespace} follows the two-level xpathDefaultNamespace chain,
//     see xpathDefaultNamespace.
//   - {base URI} is the host element's [[base URI]] — genuinely served here,
//     xml:base and all, by parser.Element.BaseURI (see ReadDocument).
//
// An absent expression attribute yields an empty {expression}: the attribute's
// "Required" status is a schema-for-schemas grammar concern this producer does
// not validate, exactly as it does not validate a missing name= elsewhere. There
// is no rejectable state at this layer (xsd.NewXPathExpression's own doc), so
// there is no error to return.
func (p *producer) buildXPathExpression(hostElem *Element, exprAttr string) xsd.XPathExpression {
	expr, _ := attrValue(hostElem, exprAttr)
	var bindings []xsd.NamespaceBinding
	for _, ns := range hostElem.InScopePrefixes() {
		bindings = append(bindings, xsd.NewNamespaceBinding(ns.Prefix(), ns.URI()))
	}
	baseURI := hostElem.BaseURI()
	return xsd.NewXPathExpression(expr, bindings, p.xpathDefaultNamespace(hostElem), &baseURI)
}

// xpathDefaultNamespace computes an XPath Expression's {default namespace}
// (§3.13.2): let D be the xpathDefaultNamespace of the host element if present,
// else that of the <schema> ancestor, else ##local (<schema>'s own default,
// §3.17.2). Then ##local is absent (nil), ##defaultNamespace is the default
// namespace in scope at the host element, ##targetNamespace is the schema's
// target namespace (absent when <schema> carries no targetNamespace), and any
// other value is a literal xs:anyURI taken as-is.
//
// The ##defaultNamespace case returns a non-nil pointer to "" when no default
// namespace is in scope: "" is the no-namespace name, a legitimate present
// value, not an absence sentinel.
func (p *producer) xpathDefaultNamespace(hostElem *Element) *string {
	d, ok := attrValue(hostElem, "xpathDefaultNamespace")
	if !ok {
		d, ok = attrValue(p.schemaElem, "xpathDefaultNamespace")
	}
	if !ok {
		d = "##local"
	}
	switch d {
	case "##local":
		return nil
	case "##defaultNamespace":
		uri, _ := hostElem.LookupPrefix("") // the empty prefix always resolves
		return &uri
	case "##targetNamespace":
		if _, hasTarget := attrValue(p.schemaElem, "targetNamespace"); !hasTarget {
			return nil
		}
		target := p.target
		return &target
	default:
		return &d
	}
}

// identityConstraintsOf maps the <unique>/<key>/<keyref> children of an
// <element> to its {identity-constraint definitions} in document order
// (§3.3.2.1, dcl.elt.common — the mapping is shared by global and local element
// declarations). An <element> with no such child yields nil.
func (p *producer) identityConstraintsOf(hostElem *Element) ([]xsd.IdentityConstraint, error) {
	var constraints []xsd.IdentityConstraint
	for _, child := range hostElem.Children() {
		el, ok := child.(*Element)
		if !ok || el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		category, ok := identityConstraintCategoryOf(el.Name().Local())
		if !ok {
			continue
		}
		ic, err := p.produceIdentityConstraint(el, category)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, ic)
	}
	return constraints, nil
}

// produceIdentityConstraint maps one <unique>/<key>/<keyref> element to an
// Identity-Constraint Definition (§3.11.2, declare-key). category comes from the
// element's local name; {name} is bundled with the schema's target namespace
// (§3.11.2: an identity constraint is always named in the target namespace).
//
// It enforces the structural src-identity-constraint clauses (§3.11.3): 1
// (exactly one of ref or name), 2 (a name= form has a <selector> child), and 3
// (a name= <keyref> has a refer attribute). The ref= form corresponds to no new
// component — it names an existing definition, resolved at finalize — and is
// declined as not yet produced (a plain Go error, never a fabricated rule
// violation), mirroring the producer's other not-yet-produced declines.
func (p *producer) produceIdentityConstraint(el *Element, category xsd.IdentityConstraintCategory) (xsd.IdentityConstraint, error) {
	local := el.Name().Local()
	name, hasName := attrValue(el, "name")
	if _, hasRef := attrValue(el, "ref"); hasRef == hasName {
		return xsd.IdentityConstraint{}, xsderr.New(ruleSrcIdentityConstraint, el.Loc(),
			"<%s> must carry exactly one of name or ref, but src-identity-constraint clause 1 permits one and not both", local)
	}
	if !hasName {
		return xsd.IdentityConstraint{}, fmt.Errorf("parser: an identity constraint in the ref= form is not yet produced (§3.11.2: it names an existing definition, resolved at finalize)")
	}

	selectorEl := childElement(el, xsd.XMLSchemaNS, "selector")
	if selectorEl == nil {
		return xsd.IdentityConstraint{}, xsderr.New(ruleSrcIdentityConstraint, el.Loc(),
			"<%s> has no <selector> child, but src-identity-constraint clause 2 requires one when name is present", local)
	}
	selector := p.buildXPathExpression(selectorEl, "xpath")

	// {fields} in document order; an empty sequence is rejected by
	// NewIdentityConstraint (c-props-correct clause 1), not pre-checked here.
	var fields []xsd.XPathExpression
	for _, child := range el.Children() {
		fieldEl, ok := child.(*Element)
		if !ok || !isXSD(fieldEl, "field") {
			continue
		}
		fields = append(fields, p.buildXPathExpression(fieldEl, "xpath"))
	}

	var referencedKey *xsd.QName
	if category == xsd.IdentityConstraintKeyref {
		referLex, ok := attrValue(el, "refer")
		if !ok {
			return xsd.IdentityConstraint{}, xsderr.New(ruleSrcIdentityConstraint, el.Loc(),
				"<keyref> has no refer attribute, but src-identity-constraint clause 3 requires one when name is present")
		}
		// Only the QName is retained: the link to the referenced definition is a
		// finalize-phase resolution (src-resolve clause 1.7), never made here.
		qn, err := resolveQName(el, referLex)
		if err != nil {
			return xsd.IdentityConstraint{}, err
		}
		referencedKey = &qn
	}
	return xsd.NewIdentityConstraint(el.Loc(), xsd.QName{Space: p.target, Local: name},
		category, selector, fields, referencedKey, nil)
}

// identityConstraintCategoryOf maps an identity-constraint element's local name
// to its {identity-constraint category} (§3.11.2: "one of key, keyref or unique,
// depending on the item"). ok is false for any other name.
func identityConstraintCategoryOf(local string) (xsd.IdentityConstraintCategory, bool) {
	switch local {
	case "unique":
		return xsd.IdentityConstraintUnique, true
	case "key":
		return xsd.IdentityConstraintKey, true
	case "keyref":
		return xsd.IdentityConstraintKeyref, true
	}
	return 0, false
}

// assertionsOf maps the <assert> children of a <complexType> or its
// <complexContent> <restriction> to {assertions} in document order (§3.4.2.1,
// dcl.ctd.common clause 2). The base type's own {assertions} (clause 1) are not
// folded in: that needs the resolved base, a finalize-phase concern.
//
// An Assertion has no rejectable state (its {test} is an opaque XPath Expression
// record), so there is nothing to return an error for.
func (p *producer) assertionsOf(parent *Element) []xsd.Assertion {
	var assertions []xsd.Assertion
	for _, child := range parent.Children() {
		el, ok := child.(*Element)
		if !ok || !isXSD(el, "assert") {
			continue
		}
		assertions = append(assertions, xsd.NewAssertion(p.buildXPathExpression(el, "test"), nil))
	}
	return assertions
}
