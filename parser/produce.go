package parser

import (
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The Schema Representation Constraints (§3.x.3) and other rules this producer
// charges. Each string is a live entry in xsderr's generated catalog.
const (
	ruleSrcElement    xsderr.Rule = "src-element"
	ruleSrcAttribute  xsderr.Rule = "src-attribute"
	ruleSrcSimpleType xsderr.Rule = "src-simple-type"
	ruleSrcResolve    xsderr.Rule = "src-resolve"
	ruleSTPropsCorr   xsderr.Rule = "st-props-correct"
)

// Produce maps the TOP-LEVEL <simpleType>, <element>, and <attribute>
// declarations of a single already-parsed schema document into xsd components,
// in document order, and returns the finalized [xsd.Schema]. It is the first
// end-to-end producer (M4): complex types, groups, attribute groups, notations,
// identity constraints, and multi-document composition are out of scope and
// their top-level elements are silently skipped (§3.1.2 permits ignoring
// not-yet-produced representations), not rejected.
//
// backend is passed explicitly rather than defaulted to a builtin/strict policy
// here: that default belongs to the eventual Parse wrapper (parser/doc.go's
// planned contract), keeping this leaf free of a builtin/strict edge. Produce
// seeds the builtin datatypes from backend ([builtin.Seed]) so a type="xs:…"
// reference resolves at finalize; the SAME *[xsd.SimpleType] pointer identity is
// both AddType'd into the builder and used as a simple-type base, as
// [xsd.SimpleType] requires.
//
// DEVIATION from parser/doc.go's "the parser collects them in document order
// rather than stopping at the first": Produce returns only the FIRST error. That
// promise is for the eventual full Parse; this first slice does not yet
// implement multi-error collection.
//
// A document whose root is not <schema> is a caller precondition fault, not a
// schema-validity verdict — §3.17.2 even allows <schema> not to be the document
// element — so it is reported as a plain Go error, deliberately NOT routed
// through xsderr (mirroring [builtin.MissingPrimitivesError]'s rationale): no
// src-*/cos-* rule governs "the document handed to a producer must be a schema
// document".
func Produce(doc *Document, backend value.Backend) (*xsd.Schema, error) {
	if !doc.IsSchema() {
		return nil, fmt.Errorf("parser: Produce requires a <schema> document root, got %s", doc.Root().Name().Local())
	}
	root := doc.Root()
	target, _ := attrValue(root, "targetNamespace")

	builder := xsd.NewSchemaBuilder()
	builtins, err := builtin.Seed(backend)
	if err != nil {
		return nil, err
	}
	built := make(map[xsd.QName]*xsd.SimpleType, len(builtins))
	for _, b := range builtins {
		builder.AddType(b)
		built[b.Name()] = b
	}

	p := &producer{
		schemaElem:       root,
		target:           target,
		builder:          builder,
		localSimpleTypes: make(map[xsd.QName]*Element),
		built:            built,
	}
	if err := p.run(); err != nil {
		return nil, err
	}
	return builder.Finalize()
}

// producer is the build context for one schema document. localSimpleTypes and
// built are pure lookup indexes, never ranged to produce user-visible order
// (STYLE D2); document order comes solely from walking schemaElem.Children().
type producer struct {
	schemaElem *Element
	target     string
	builder    *xsd.SchemaBuilder

	// localSimpleTypes maps each top-level named <simpleType>'s expanded name to
	// its raw element, filled by the pre-scan so forward base= references between
	// local simple types resolve (Structures §3.1.3).
	localSimpleTypes map[xsd.QName]*Element

	// built is the memo + cycle guard for simple-type construction, mirroring
	// xsd/resolve.go's color-map idiom collapsed into one map: an ABSENT key is
	// unstarted, a PRESENT-nil value is on the build stack (being built), and a
	// PRESENT-non-nil value is done. The pre-seeded builtins start out done.
	built map[xsd.QName]*xsd.SimpleType
}

// run walks the <schema> children once to register local simple types, then
// again in strict document order to produce each in-scope declaration.
func (p *producer) run() error {
	// Pre-scan: register every top-level named <simpleType> so forward base=
	// references resolve (§3.1.3). Build nothing yet.
	for _, child := range p.schemaElem.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if !isXSD(el, "simpleType") {
			continue
		}
		name, ok := attrValue(el, "name")
		if !ok {
			continue
		}
		p.localSimpleTypes[xsd.QName{Space: p.target, Local: name}] = el
	}

	// Main pass: dispatch by expanded element name in document order.
	for _, child := range p.schemaElem.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch el.Name().Local() {
		case "simpleType":
			name, _ := attrValue(el, "name")
			st, err := p.buildSimpleType(xsd.QName{Space: p.target, Local: name}, el)
			if err != nil {
				return err
			}
			p.builder.AddType(st)
		case "element":
			ed, err := p.produceElement(el)
			if err != nil {
				return err
			}
			p.builder.AddElement(ed)
		case "attribute":
			ad, err := p.produceAttribute(el)
			if err != nil {
				return err
			}
			p.builder.AddAttribute(ad)
		default:
			// annotation, complexType, group, import, include, … — not this
			// slice's scope (§3.1.2), skipped, not invalid.
		}
	}
	return nil
}

// buildSimpleType returns the compiled simple type named name, building it (and
// its base chain) on demand with memoization and a cycle guard. name is the zero
// QName only via constructSimpleType for anonymous inline types, which never
// enter this memoized path.
func (p *producer) buildSimpleType(name xsd.QName, elem *Element) (*xsd.SimpleType, error) {
	if st, started := p.built[name]; started {
		if st != nil {
			return st, nil
		}
		// PRESENT-nil: name is on the current build stack — a circular base chain.
		return nil, xsderr.New(ruleSTPropsCorr, elem.Loc(),
			"circular simple type definition: %s derives ultimately from itself, but st-props-correct clause 2 requires every simple type derive from xs:anySimpleType", name)
	}
	p.built[name] = nil // mark on-stack

	st, err := p.constructSimpleType(name, elem)
	if err != nil {
		return nil, err
	}
	p.built[name] = st // replace the on-stack sentinel with the finished node
	return st, nil
}

// constructSimpleType maps one <simpleType> element (named or anonymous) into a
// component: it reads the single <restriction> child, resolves the base, maps
// the own facets, and constructs. It does NOT memoize — the memo/cycle bookkeeping
// lives in buildSimpleType; an anonymous inline type has no name to key on and is
// unreferenceable, so it is built here directly, once.
func (p *producer) constructSimpleType(name xsd.QName, elem *Element) (*xsd.SimpleType, error) {
	restriction, err := restrictionOf(elem)
	if err != nil {
		return nil, err
	}
	base, err := p.resolveBase(restriction)
	if err != nil {
		return nil, err
	}
	facets, err := restrictionFacets(restriction)
	if err != nil {
		return nil, err
	}
	// {variety} of a restriction is the {variety} of its base (§3.16.2.1). Reusing
	// base.Variety() propagates the base's own {primitive type definition} pointer
	// for an atomic base (warden finding #4), and the item/member pointers for a
	// list/union base.
	return xsd.NewSimpleType(elem.Loc(), name, base.Variety(), base, facets, nil)
}

// restrictionOf returns the single <restriction> child of a <simpleType>. A
// <simpleType> using <list> or <union> instead has no <restriction> child; that
// is rejected explicitly (never silently skipped), since this slice only
// implements the restriction case (§3.16.3 src-simple-type governs the required
// <restriction>|<list>|<union> shape).
func restrictionOf(elem *Element) (*Element, error) {
	if r := childElement(elem, xsd.XMLSchemaNS, "restriction"); r != nil {
		return r, nil
	}
	return nil, xsderr.New(ruleSrcSimpleType, elem.Loc(),
		"simpleType has no <restriction> child; this producer does not yet support <list> or <union> simple types")
}

// resolveBase resolves a <restriction>'s {base type definition} to a live
// *SimpleType. It enforces src-simple-type clause 2 (§3.16.3): exactly one of a
// base= attribute or an inline <simpleType> child, never both, never neither. A
// base= is discharged EARLY here — unlike element/attribute type=, which defers
// to finalize — because NewSimpleType demands a live base pointer at construction.
func (p *producer) resolveBase(restriction *Element) (*xsd.SimpleType, error) {
	baseLex, hasBase := attrValue(restriction, "base")
	inline := childElement(restriction, xsd.XMLSchemaNS, "simpleType")

	if hasBase && inline != nil {
		return nil, xsderr.New(ruleSrcSimpleType, restriction.Loc(),
			"restriction has both a base attribute and an inline <simpleType> child, but src-simple-type clause 2 allows only one")
	}
	if !hasBase && inline == nil {
		return nil, xsderr.New(ruleSrcSimpleType, restriction.Loc(),
			"restriction has neither a base attribute nor an inline <simpleType> child, but src-simple-type clause 2 requires exactly one")
	}

	if inline != nil {
		// Anonymous base: built inline, once, with an absent {name} (zero QName).
		return p.constructSimpleType(xsd.QName{}, inline)
	}

	qn, err := resolveQName(restriction, baseLex)
	if err != nil {
		return nil, err
	}
	// Pre-seeded builtins and already-finished locals resolve directly.
	if st, ok := p.built[qn]; ok && st != nil {
		return st, nil
	}
	// A local (unbuilt or on-stack) recurses; buildSimpleType handles memo hit
	// and cycle rejection.
	if localElem, ok := p.localSimpleTypes[qn]; ok {
		return p.buildSimpleType(qn, localElem)
	}
	return nil, xsderr.New(ruleSrcResolve, restriction.Loc(),
		"base type %s does not resolve to any simple type in scope (src-resolve clause 1.1)", qn)
}

// restrictionFacets maps the plain-lexical constraining-facet children of a
// <restriction> in document order. enumeration and assertion facets need richer
// sub-shapes and are not yet produced: rather than silently dropping a constraint
// (a false-accept), an actual <enumeration>/<assertion> child is rejected. The
// non-facet children <annotation> and the inline base <simpleType> are skipped.
func restrictionFacets(restriction *Element) ([]xsd.Facet, error) {
	var facets []xsd.Facet
	for _, child := range restriction.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		local := el.Name().Local()
		if local == "annotation" || local == "simpleType" {
			continue
		}
		if local == "enumeration" || local == "assertion" {
			return nil, xsderr.New(ruleSrcSimpleType, el.Loc(),
				"restriction has a <%s> facet, which this producer does not yet support; refusing to silently drop it", local)
		}
		kind, ok := facetKindOf(local)
		if !ok {
			continue
		}
		val, _ := attrValue(el, "value")
		fixed := xsdBool(el)
		facets = append(facets, xsd.NewFacet(kind, []string{val}, fixed))
	}
	return facets, nil
}

// produceElement maps a top-level <element> into a global Element Declaration
// (§3.3.2.2 dcl.elt.global). type= form only: an inline <simpleType>/<complexType>
// child is not wired in this slice.
func (p *producer) produceElement(elem *Element) (xsd.ElementDeclaration, error) {
	name, _ := attrValue(elem, "name")
	qname := xsd.QName{Space: p.target, Local: name}

	typeLex, hasType := attrValue(elem, "type")
	inline := childElement(elem, xsd.XMLSchemaNS, "simpleType") != nil ||
		childElement(elem, xsd.XMLSchemaNS, "complexType") != nil

	if hasType && inline {
		return xsd.ElementDeclaration{}, xsderr.New(ruleSrcElement, elem.Loc(),
			"element has both a type attribute and an inline <simpleType>/<complexType> child, but src-element clause 3 forbids both")
	}
	if inline {
		return xsd.ElementDeclaration{}, xsderr.New(ruleSrcElement, elem.Loc(),
			"element has an inline <simpleType>/<complexType> child, which this producer does not yet support (only the type attribute form); src-element clause 3")
	}

	vc, err := valueConstraintOf(elem, ruleSrcElement)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}

	typeName := xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"} // §3.3.2.1 case 4
	if hasType {
		typeName, err = resolveQName(elem, typeLex)
		if err != nil {
			return xsd.ElementDeclaration{}, err
		}
	}
	return xsd.NewElementDeclaration(elem.Loc(), qname, typeName, nil, xsd.ScopeGlobal, vc,
		false, nil, nil, nil, false, nil, nil)
}

// produceAttribute maps a top-level <attribute> into a global Attribute
// Declaration (§3.2.2.1 dcl.att.global). type= form only.
func (p *producer) produceAttribute(elem *Element) (xsd.AttributeDeclaration, error) {
	name, _ := attrValue(elem, "name")
	qname := xsd.QName{Space: p.target, Local: name}

	typeLex, hasType := attrValue(elem, "type")
	inline := childElement(elem, xsd.XMLSchemaNS, "simpleType") != nil

	if hasType && inline {
		return xsd.AttributeDeclaration{}, xsderr.New(ruleSrcAttribute, elem.Loc(),
			"attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both")
	}
	if inline {
		return xsd.AttributeDeclaration{}, xsderr.New(ruleSrcAttribute, elem.Loc(),
			"attribute has an inline <simpleType> child, which this producer does not yet support (only the type attribute form); src-attribute clause 4")
	}

	vc, err := valueConstraintOf(elem, ruleSrcAttribute)
	if err != nil {
		return xsd.AttributeDeclaration{}, err
	}

	typeName := xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"} // §3.2.2.1
	if hasType {
		typeName, err = resolveQName(elem, typeLex)
		if err != nil {
			return xsd.AttributeDeclaration{}, err
		}
	}
	return xsd.NewAttributeDeclaration(elem.Loc(), qname, typeName, xsd.ScopeGlobal, vc, false, nil)
}

// valueConstraintOf maps the default/fixed attributes of an <element>/<attribute>
// to a *ValueConstraint, rejecting the both-present case (src-element clause 1 /
// src-attribute clause 1). rule selects which of the two constraints is charged.
func valueConstraintOf(elem *Element, rule xsderr.Rule) (*xsd.ValueConstraint, error) {
	defLex, hasDef := attrValue(elem, "default")
	fixLex, hasFix := attrValue(elem, "fixed")
	if hasDef && hasFix {
		return nil, xsderr.New(rule, elem.Loc(),
			"declaration has both default and fixed, but %s clause 1 forbids both", rule)
	}
	if hasDef {
		vc := xsd.NewValueConstraint(xsd.ValueDefault, defLex)
		return &vc, nil
	}
	if hasFix {
		vc := xsd.NewValueConstraint(xsd.ValueFixed, fixLex)
		return &vc, nil
	}
	return nil, nil
}

// resolveQName resolves a QName-valued lexical (a type=/base= value) against the
// namespace bindings in scope at elem (§3.17.6.2 src-resolve clause 4). A
// prefixed name whose prefix is unbound is rejected. An unprefixed name binds to
// the in-scope default namespace, or — when none is declared — to the
// no-namespace name (clause 4.1.1), deliberately NOT the schema's own
// targetNamespace.
func resolveQName(elem *Element, lexical string) (xsd.QName, error) {
	before, after, found := strings.Cut(lexical, ":")
	prefix, local := "", before
	if found {
		prefix, local = before, after
	}

	if prefix == "" {
		uri, ok := elem.LookupPrefix("")
		if !ok {
			return xsd.QName{Space: "", Local: local}, nil
		}
		return xsd.QName{Space: uri, Local: local}, nil
	}

	uri, ok := elem.LookupPrefix(prefix)
	if !ok {
		return xsd.QName{}, xsderr.New(ruleSrcResolve, elem.Loc(),
			"the QName prefix %q of %q does not resolve to an in-scope namespace (src-resolve)", prefix, lexical)
	}
	return xsd.QName{Space: uri, Local: local}, nil
}

// facetKindOf maps a plain-lexical constraining-facet element's local name to its
// [xsd.FacetKind]. enumeration and assertion are deliberately absent — they need
// richer sub-shapes and are handled (rejected) by restrictionFacets.
func facetKindOf(local string) (xsd.FacetKind, bool) {
	switch local {
	case "length":
		return xsd.FacetLength, true
	case "minLength":
		return xsd.FacetMinLength, true
	case "maxLength":
		return xsd.FacetMaxLength, true
	case "pattern":
		return xsd.FacetPattern, true
	case "whiteSpace":
		return xsd.FacetWhiteSpace, true
	case "maxInclusive":
		return xsd.FacetMaxInclusive, true
	case "maxExclusive":
		return xsd.FacetMaxExclusive, true
	case "minInclusive":
		return xsd.FacetMinInclusive, true
	case "minExclusive":
		return xsd.FacetMinExclusive, true
	case "totalDigits":
		return xsd.FacetTotalDigits, true
	case "fractionDigits":
		return xsd.FacetFractionDigits, true
	case "explicitTimezone":
		return xsd.FacetExplicitTimezone, true
	}
	return 0, false
}

// isXSD reports whether el's expanded name is {XMLSchemaNS}local.
func isXSD(el *Element, local string) bool {
	return el.Name().Space() == xsd.XMLSchemaNS && el.Name().Local() == local
}

// attrValue returns the value of el's unprefixed (no-namespace) attribute local,
// as XSD schema-element attributes carry no namespace. ok is false when absent.
func attrValue(el *Element, local string) (string, bool) {
	for _, a := range el.Attributes() {
		if a.Name().Space() == "" && a.Name().Local() == local {
			return a.Value(), true
		}
	}
	return "", false
}

// childElement returns el's first child element with the expanded name
// {space}local, or nil.
func childElement(el *Element, space, local string) *Element {
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok {
			continue
		}
		if name := c.Name(); name.Space() == space && name.Local() == local {
			return c
		}
	}
	return nil
}

// xsdBool reads a facet element's {fixed} from its fixed attribute, per the
// xs:boolean lexical space (true/1). An absent attribute is false.
func xsdBool(el *Element) bool {
	v, ok := attrValue(el, "fixed")
	if !ok {
		return false
	}
	return v == "true" || v == "1"
}
