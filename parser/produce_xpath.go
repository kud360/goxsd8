package parser

import (
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
// The ##defaultNamespace case follows clause 1 exactly: the [[namespace name]]
// of the host element's in-scope-namespaces entry whose [[prefix]] is absent
// (clause 1.1), and ·absent· when there is no such entry (clause 1.2). A
// declared default namespace always has a non-empty namespace name, and
// xmlns="" undeclares one (Namespaces in XML 1.1), so an empty LookupPrefix("")
// result means exactly "no absent-prefix entry in scope" — both clause 1.2
// shapes — and yields nil, never a present "".
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
		// ok is no signal here: the empty prefix always resolves (scope.lookup).
		// An empty URI is clause 1.2 — no default namespace bound, so ·absent·.
		uri, _ := hostElem.LookupPrefix("")
		if uri == "" {
			return nil
		}
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

// produceIdentityConstraint maps one <unique>/<key>/<keyref> element to the
// Identity-Constraint Definition it contributes to its host <element>'s
// {identity-constraint definitions}, over the two forms §3.11.2's XML Mapping
// Summary distinguishes. category comes from the element's local name.
//
// It enforces src-identity-constraint clause 1 (§3.11.3, "one of ref or name is
// present, but not both") here, at the fork, since it is what decides the form;
// the form-specific clauses live with their form (2 and 3 in
// constructIdentityConstraint, 4 and 5 in referencedIdentityConstraint).
//
//   - The name= form DEFINES a component. It is built (once per expanded name)
//     and registered as one of the schema's {identity-constraint definitions}
//     (§3.17.1) HERE, at the definition's own document-order position — never at
//     a demand-driven build site, so the registered order stays document order
//     (STYLE D2), exactly as buildComplexType leaves AddType to run.
//   - The ref= form defines NOTHING: "the corresponding schema component is the
//     identity-constraint definition ·resolved· to by the ·actual value· of the
//     ref [[attribute]]". It contributes that existing component and is
//     deliberately NOT registered — a second registration under the same name
//     would fabricate a sch-props-correct (§3.17.6.1) clause 2 collision against
//     the very definition it reuses.
func (p *producer) produceIdentityConstraint(el *Element, category xsd.IdentityConstraintCategory) (xsd.IdentityConstraint, error) {
	name, hasName := attrValue(el, "name")
	if _, hasRef := attrValue(el, "ref"); hasRef == hasName {
		return xsd.IdentityConstraint{}, xsderr.New(ruleSrcIdentityConstraint, el.Loc(),
			"<%s> must carry exactly one of name or ref, but src-identity-constraint clause 1 permits one and not both", el.Name().Local())
	}
	if !hasName {
		return p.referencedIdentityConstraint(el, category)
	}
	ic, err := p.buildIdentityConstraint(xsd.QName{Space: p.target, Local: name}, el, category)
	if err != nil {
		return xsd.IdentityConstraint{}, err
	}
	p.builder.AddIdentityConstraint(ic)
	return ic, nil
}

// referencedIdentityConstraint maps the ref= form of <unique>/<key>/<keyref> to
// the definition it names (§3.11.2's "reuse it directly via the ref attribute"),
// enforcing the two src-identity-constraint (§3.11.3) clauses that govern the
// form:
//
//   - clause 4, "if ref is present, then only id and <annotation> are allowed to
//     appear together with ref": every other schema-vocabulary attribute (refer
//     above all; name is already gone via clause 1) and every child element other
//     than <annotation> is rejected. Attributes from foreign namespaces are left
//     alone — the schema for schema documents admits them on every element, so
//     they are not "appearing together with ref" in the sense clause 4 restricts.
//   - clause 5, "the {identity-constraint category} of the identity-constraint
//     definition ·resolved· to by the ·actual value· of the ref attribute matches
//     the name of the element information item": EXACT category equality, since
//     category is a bijection with the element's local name. This is emphatically
//     not refer='s cross-category link (§3.11.1: a keyref's {referenced key} is a
//     key or unique) — a <keyref ref="…"> demands a keyref.
//
// A ref that names no definition of the assembly is charged src-resolve clause
// 1.7 (§3.17.6.2), positioned at the referring element. Clause 5 is decided
// against the pre-scan index BEFORE the target is built, so a reference that is
// both miscategorized and names a malformed definition reports its own local
// violation rather than the target's (one deterministic first failure, STYLE D1).
func (p *producer) referencedIdentityConstraint(el *Element, category xsd.IdentityConstraintCategory) (xsd.IdentityConstraint, error) {
	local := el.Name().Local()
	if err := checkIdentityConstraintRefBare(el, local); err != nil {
		return xsd.IdentityConstraint{}, err
	}
	refLex, _ := attrValue(el, "ref") // present: the caller took this arm on !hasName
	qn, err := p.resolveQName(el, refLex)
	if err != nil {
		return xsd.IdentityConstraint{}, err
	}
	src, ok := p.symbols.identityConstraints[qn]
	if !ok {
		return xsd.IdentityConstraint{}, xsderr.New(ruleSrcResolve, el.Loc(),
			"<%s ref=%q> resolves to no identity-constraint definition in the schema (src-resolve clause 1.7)", local, qn)
	}
	if src.category != category {
		return xsd.IdentityConstraint{}, xsderr.New(ruleSrcIdentityConstraint, el.Loc(),
			"<%s ref=%q> names a definition whose {identity-constraint category} is %s, but src-identity-constraint clause 5 requires it to match the referring element's name", local, qn, src.category)
	}
	return src.owner.buildIdentityConstraint(qn, src.elem, src.category)
}

// checkIdentityConstraintRefBare enforces src-identity-constraint clause 4
// (§3.11.3) on the ref= form of local: only id may accompany the ref attribute,
// and only <annotation> may appear among the children. Attributes are checked
// before children, each in source order, so the first reported violation is
// deterministic (STYLE D1).
func checkIdentityConstraintRefBare(el *Element, local string) error {
	for _, a := range el.Attributes() {
		if a.Name().Space() != "" {
			continue // a foreign-namespace attribute is outside clause 4's vocabulary
		}
		switch a.Name().Local() {
		case "ref", "id":
		default:
			return xsderr.New(ruleSrcIdentityConstraint, el.Loc(),
				"<%s ref=…> also carries %s, but src-identity-constraint clause 4 allows only id and <annotation> together with ref", local, a.Name().Local())
		}
	}
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok || isXSD(c, "annotation") {
			continue
		}
		return xsderr.New(ruleSrcIdentityConstraint, c.Loc(),
			"<%s ref=…> has a <%s> child, but src-identity-constraint clause 4 allows only id and <annotation> together with ref", local, c.Name().Local())
	}
	return nil
}

// buildIdentityConstraint returns the component the name= form declared by el
// maps to, building it at most once per expanded name. The memo is what makes a
// <key ref="…"> contribute the VERY component its definition contributed rather
// than a rebuilt twin, as §3.11.2's mapping summary requires; it needs no
// on-stack sentinel, since constructing a definition never reaches another one
// (see symbols.builtIC).
//
// It registers nothing with the builder: the schema's {identity-constraint
// definitions} is populated at the definition's own document-order position by
// produceIdentityConstraint, not here, where a forward reference may have pulled
// the build early.
//
// A SECOND definition of an already-built name takes the memo hit rather than
// being mapped itself, exactly as buildComplexType's does. That is a deliberate
// consequence, not an oversight: such a document is invalid either way, and the
// duplicate registration it still performs is what reports it — as
// sch-props-correct (§3.17.6.1) clause 2 rather than as whatever the second
// declaration's own body would have been charged.
func (p *producer) buildIdentityConstraint(name xsd.QName, el *Element, category xsd.IdentityConstraintCategory) (xsd.IdentityConstraint, error) {
	if ic, done := p.symbols.builtIC[name]; done {
		return ic, nil
	}
	ic, err := p.constructIdentityConstraint(name, el, category)
	if err != nil {
		return xsd.IdentityConstraint{}, err
	}
	p.symbols.builtIC[name] = ic
	return ic, nil
}

// constructIdentityConstraint maps the name= form of one <unique>/<key>/<keyref>
// element to an Identity-Constraint Definition (§3.11.2, declare-key). name is
// bundled with the declaring document's effective target namespace (§3.11.2: an
// identity constraint is always named in the target namespace).
//
// It enforces the two src-identity-constraint (§3.11.3) clauses that govern the
// definition form: 2 (a <selector> child is present) and 3 (a <keyref> carries a
// refer attribute). It does NOT memoize — that bookkeeping lives in
// buildIdentityConstraint.
func (p *producer) constructIdentityConstraint(name xsd.QName, el *Element, category xsd.IdentityConstraintCategory) (xsd.IdentityConstraint, error) {
	local := el.Name().Local()
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
		qn, err := p.resolveQName(el, referLex)
		if err != nil {
			return xsd.IdentityConstraint{}, err
		}
		referencedKey = &qn
	}
	return xsd.NewIdentityConstraint(el.Loc(), name, category, selector, fields, referencedKey, nil)
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
