package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// anyTypeName is the expanded name of xs:anyType, the ur-type (§3.4.7).
var anyTypeName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}

// seedAnyType builds the ur-type Complex Type Definition xs:anyType (§3.4.7): a
// mixed complex type whose {content type} is a 1..1 sequence wrapping a single
// 0..unbounded lax ##any element wildcard, with a lax ##any attribute wildcard
// and no attribute uses, and whose {base type definition} is itself (the sole
// permitted self-derivation, any-type-itself). checkComplexBaseAcyclic (#173)
// recognises that self-derivation by name, so the seeded value passes finalize.
func seedAnyType() (xsd.ComplexType, error) {
	anyNS, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	wildcard, err := xsd.NewWildcard(xsderr.Loc{}, anyNS, xsd.ProcessLax, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	inner, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, 0)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	wildcardParticle, err := xsd.NewParticle(xsderr.Loc{}, inner, xsd.ResolvedTerm{Term: wildcard}, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	seq, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{wildcardParticle}, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	oneOne, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	topParticle, err := xsd.NewParticle(xsderr.Loc{}, oneOne, xsd.ResolvedTerm{Term: seq}, nil)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	content := xsd.ElementContent{Mixed: true, Particle: topParticle}
	return xsd.NewComplexType(xsderr.Loc{}, anyTypeName, anyTypeName, nil,
		xsd.DerivationRestriction, false, nil, &wildcard, content, nil, nil, nil)
}

// produceComplexType maps a <complexType> element (§3.4.2) into a Complex Type
// Definition. Only the produce-time-decidable subset is built: implicit complex
// content (§3.4.2.3.2, restriction from xs:anyType) and explicit <complexContent>
// with <restriction>. <simpleContent> and <complexContent> with <extension> both
// need the resolved {base type definition} to compute their {content type}
// (§3.4.2.2 / §3.4.2.3.3 clause 4.2) — a finalize-time dependency (PRINCIPLES 9,
// phased construction) out of this slice's scope — so they are declined with a
// plain "not yet supported" error rather than a fabricated rule violation
// (mirroring Produce's non-schema-root precedent: no src-*/cos-* rule governs
// "this representation is not yet produced"). The conformance schema lane
// (conformance/schema.go) declines these shapes, so the decline never reaches a
// validity verdict.
func (p *producer) produceComplexType(name xsd.QName, el *Element) (xsd.ComplexType, error) {
	if childElement(el, xsd.XMLSchemaNS, "simpleContent") != nil {
		return xsd.ComplexType{}, fmt.Errorf("parser: <complexType> with <simpleContent> is not yet produced (its {simple type definition} needs the resolved base, §3.4.2.2)")
	}
	if cc := childElement(el, xsd.XMLSchemaNS, "complexContent"); cc != nil {
		return p.produceComplexContent(name, el, cc)
	}
	return p.produceImplicitContent(name, el)
}

// produceImplicitContent maps a <complexType> with neither <simpleContent> nor
// <complexContent> (§3.4.2.3.2): the {base type definition} is xs:anyType and
// {derivation method} is restriction. The explicit content, attribute uses, and
// attribute wildcard come directly from the <complexType>'s own children.
func (p *producer) produceImplicitContent(name xsd.QName, el *Element) (xsd.ComplexType, error) {
	mixed, _ := boolAttr(el, "mixed")
	abstract, _ := boolAttr(el, "abstract")
	content, err := p.buildComplexContentType(el, mixed)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	uses, wildcard, err := p.produceAttributeUses(el)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	return xsd.NewComplexType(el.Loc(), name, anyTypeName, nil,
		xsd.DerivationRestriction, abstract, uses, wildcard, content, nil, nil, nil)
}

// produceComplexContent maps a <complexType><complexContent> (§3.4.2.3). Only the
// <restriction> alternative is produced (its {content type} is purely structural,
// §3.4.2.3.3 clause 4.1); <extension> needs the resolved base's content type and
// is declined. It enforces src-ct clause 5 (§3.4.3): when mixed is present on both
// <complexType> and <complexContent>, the two actual values must agree.
func (p *producer) produceComplexContent(name xsd.QName, ctElem, cc *Element) (xsd.ComplexType, error) {
	restriction := childElement(cc, xsd.XMLSchemaNS, "restriction")
	if restriction == nil {
		return xsd.ComplexType{}, fmt.Errorf("parser: <complexContent> with <extension> is not yet produced (its {content type} needs the resolved base particle, §3.4.2.3.3 clause 4.2)")
	}
	ctMixed, ctHasMixed := boolAttr(ctElem, "mixed")
	ccMixed, ccHasMixed := boolAttr(cc, "mixed")
	if ctHasMixed && ccHasMixed && ctMixed != ccMixed {
		return xsd.ComplexType{}, xsderr.New(ruleSrcCT, cc.Loc(),
			"mixed is present on both <complexType> and <complexContent> with differing values, but src-ct clause 5 requires them to be the same")
	}
	// {effective mixed} (§3.4.2.3.3 clause 1): <complexContent>'s mixed if present,
	// else <complexType>'s, else false.
	mixed := ctMixed
	if ccHasMixed {
		mixed = ccMixed
	}
	abstract, _ := boolAttr(ctElem, "abstract")
	base, err := resolveQName(restriction, attrOr(restriction, "base"))
	if err != nil {
		return xsd.ComplexType{}, err
	}
	content, err := p.buildComplexContentType(restriction, mixed)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	uses, wildcard, err := p.produceAttributeUses(restriction)
	if err != nil {
		return xsd.ComplexType{}, err
	}
	return xsd.NewComplexType(ctElem.Loc(), name, base, nil,
		xsd.DerivationRestriction, abstract, uses, wildcard, content, nil, nil, nil)
}

// buildComplexContentType computes the {content type} of a restriction-derived
// (or implicit) complex content from parent's model-group child (§3.4.2.3.3
// clauses 2-4, restriction case). parent is the <complexType> (implicit) or the
// <restriction> (explicit complex content); effectiveMixed is clause 1's result.
//
// It declines <openContent>: computing {open content} needs <defaultOpenContent>
// fallback support (§3.4.2.3.3 clauses 5-6) not yet built, and silently mapping it
// to absent would be wrong (grounding: GAP/decline, never silently absent).
func (p *producer) buildComplexContentType(parent *Element, effectiveMixed bool) (xsd.ContentType, error) {
	if childElement(parent, xsd.XMLSchemaNS, "openContent") != nil {
		return nil, fmt.Errorf("parser: <openContent> is not yet produced (its {open content} needs <defaultOpenContent> fallback, §3.4.2.3.3)")
	}
	group := modelGroupChild(parent)
	explicit, err := p.explicitContent(group)
	if err != nil {
		return nil, err
	}
	// {effective content} (§3.4.2.3.3 clause 3): when explicit content is empty and
	// the type is mixed, an empty 1..1 sequence stands in so text is admitted.
	if explicit == nil {
		if !effectiveMixed {
			return xsd.EmptyContent{}, nil // clause 4.1.1 (restriction, empty)
		}
		seq, err := xsd.NewModelGroup(parent.Loc(), xsd.CompositorSequence, nil, nil)
		if err != nil {
			return nil, err
		}
		oneOne, err := xsd.NewOccurs(parent.Loc(), 1, 1)
		if err != nil {
			return nil, err
		}
		part, err := xsd.NewParticle(parent.Loc(), oneOne, xsd.ResolvedTerm{Term: seq}, nil)
		if err != nil {
			return nil, err
		}
		explicit = &part
	}
	// clause 4.1.2 (restriction, non-empty): mixed iff effectiveMixed.
	return xsd.ElementContent{Mixed: effectiveMixed, Particle: *explicit}, nil
}

// explicitContent maps the model-group child to the {explicit content} particle
// (§3.4.2.3.3 clause 2), returning nil for the ***empty*** cases: no group child
// (2.1.1), an empty <all>/<sequence> (2.1.2), a childless <choice minOccurs="0">
// (2.1.3), or a group child with maxOccurs="0" (2.1.4).
func (p *producer) explicitContent(group *Element) (*xsd.Particle, error) {
	if group == nil {
		return nil, nil // 2.1.1
	}
	local := group.Name().Local()
	if local == "group" {
		return nil, fmt.Errorf("parser: <group ref> content is not yet produced (needs a top-level model group definition, §3.7.2)")
	}
	hasChildren := hasParticleChild(group)
	if (local == "all" || local == "sequence") && !hasChildren {
		return nil, nil // 2.1.2
	}
	if local == "choice" && !hasChildren && minOccursZero(group) {
		return nil, nil // 2.1.3
	}
	if maxOccursZero(group) {
		return nil, nil // 2.1.4
	}
	return p.produceGroupParticle(group, true) // 2.2
}

// produceGroupParticle maps an <all>/<choice>/<sequence> element to a Particle
// wrapping a Model Group (§3.8.2), with {particles} in document order. top marks
// whether the group is the direct content particle of a complex type: an <all>
// may only appear there (cos-all-limited §3.8.6.2, clause 1), never nested in a
// <choice>/<sequence>. A minOccurs=maxOccurs=0 group maps to no component at all
// (§3.8.2) — produceGroupParticle returns (nil, nil) — so the caller omits it.
// The grammar's own {0,1} occurrence restriction on <all> is left to a later
// schema-for-schemas grammar check (per the #176 grounding), not charged here.
func (p *producer) produceGroupParticle(group *Element, top bool) (*xsd.Particle, error) {
	local := group.Name().Local()
	compositor, ok := compositorOf(local)
	if !ok {
		return nil, fmt.Errorf("parser: <group ref> content is not yet produced (needs a top-level model group definition, §3.7.2)")
	}
	if compositor == xsd.CompositorAll && !top {
		return nil, xsderr.New(ruleCosAllLimited, group.Loc(),
			"an <all> model group appears nested inside a <choice>/<sequence>, but cos-all-limited clause 1 permits it only as a complex type's content particle")
	}
	occ, elided, err := occursOf(group)
	if err != nil {
		return nil, err
	}
	if elided {
		return nil, nil
	}
	particles, err := p.groupParticles(group)
	if err != nil {
		return nil, err
	}
	mg, err := xsd.NewModelGroup(group.Loc(), compositor, particles, nil)
	if err != nil {
		return nil, err
	}
	part, err := xsd.NewParticle(group.Loc(), occ, xsd.ResolvedTerm{Term: mg}, nil)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

// groupParticles maps the particle children of a model group in document order,
// omitting each minOccurs=maxOccurs=0 child (which maps to no component, §3.9.2).
func (p *producer) groupParticles(group *Element) ([]xsd.Particle, error) {
	var particles []xsd.Particle
	for _, child := range group.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		var part *xsd.Particle
		var err error
		switch el.Name().Local() {
		case "annotation":
			continue
		case "element":
			part, err = p.produceElementParticle(el)
		case "any":
			part, err = p.produceAnyParticle(el)
		case "sequence", "choice", "all":
			part, err = p.produceGroupParticle(el, false)
		case "group":
			return nil, fmt.Errorf("parser: <group ref> content is not yet produced (needs a top-level model group definition, §3.7.2)")
		default:
			return nil, fmt.Errorf("parser: unexpected model group child <%s>", el.Name().Local())
		}
		if err != nil {
			return nil, err
		}
		if part != nil {
			particles = append(particles, *part)
		}
	}
	return particles, nil
}

// produceElementParticle maps a local <element> to a Particle (§3.3.2.3). A
// minOccurs=maxOccurs=0 element maps to no component at all (returns nil). An
// <element ref="..."> yields a deferred ElementDeclarationRef term (resolved at
// finalize, #173); otherwise a sibling local Element Declaration is built inline.
func (p *producer) produceElementParticle(el *Element) (*xsd.Particle, error) {
	occ, elided, err := occursOf(el)
	if err != nil {
		return nil, err
	}
	if elided {
		return nil, nil
	}
	if ref, hasRef := attrValue(el, "ref"); hasRef {
		qn, err := resolveQName(el, ref)
		if err != nil {
			return nil, err
		}
		part, err := xsd.NewParticle(el.Loc(), occ, xsd.ElementDeclarationRef{Name: qn}, nil)
		if err != nil {
			return nil, err
		}
		return &part, nil
	}
	decl, err := p.produceLocalElement(el)
	if err != nil {
		return nil, err
	}
	part, err := xsd.NewParticle(el.Loc(), occ, xsd.ResolvedTerm{Term: decl}, nil)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

// produceLocalElement maps a local inline <element name="..."> to a local
// Element Declaration (§3.3.2.3, dcl.elt.local, {scope} = local). type= form
// only: an inline <simpleType>/<complexType> child is declined (not yet
// produced). A type=-less element defaults its {type definition} to xs:anyType
// (§3.3.2.1 case 4), now resolvable.
func (p *producer) produceLocalElement(el *Element) (xsd.ElementDeclaration, error) {
	if childElement(el, xsd.XMLSchemaNS, "simpleType") != nil || childElement(el, xsd.XMLSchemaNS, "complexType") != nil {
		return xsd.ElementDeclaration{}, fmt.Errorf("parser: a local <element> with an inline <simpleType>/<complexType> is not yet produced (type= form only)")
	}
	name, _ := attrValue(el, "name")
	qname := xsd.QName{Space: p.localTargetNS(el, "elementFormDefault"), Local: name}
	vc, err := valueConstraintOf(el, ruleSrcElement)
	if err != nil {
		return xsd.ElementDeclaration{}, err
	}
	typeName := anyTypeName
	if typeLex, hasType := attrValue(el, "type"); hasType {
		typeName, err = resolveQName(el, typeLex)
		if err != nil {
			return xsd.ElementDeclaration{}, err
		}
	}
	nillable, _ := boolAttr(el, "nillable")
	return xsd.NewElementDeclaration(el.Loc(), qname, typeName, nil, xsd.ScopeLocal, vc,
		nillable, nil, nil, nil, false, nil, nil)
}

// produceAnyParticle maps an <any> to a Particle whose {term} is a Wildcard
// (§3.10.2.1). A minOccurs=maxOccurs=0 <any> maps to no component (returns nil).
func (p *producer) produceAnyParticle(el *Element) (*xsd.Particle, error) {
	occ, elided, err := occursOf(el)
	if err != nil {
		return nil, err
	}
	if elided {
		return nil, nil
	}
	wildcard, err := p.produceWildcard(el)
	if err != nil {
		return nil, err
	}
	part, err := xsd.NewParticle(el.Loc(), occ, xsd.ResolvedTerm{Term: wildcard}, nil)
	if err != nil {
		return nil, err
	}
	return &part, nil
}

// produceAttributeUses maps the attribute-bearing children of parent (a
// <complexType> or <restriction>) in document order into {attribute uses} plus an
// optional {attribute wildcard} from a single <anyAttribute> (§3.4.2.5, the
// inline case; the union-with-base/attributeGroup computation is finalize-time
// and out of scope). An <attributeGroup> reference is declined (not yet produced).
func (p *producer) produceAttributeUses(parent *Element) ([]xsd.AttributeUse, *xsd.Wildcard, error) {
	var uses []xsd.AttributeUse
	var wildcard *xsd.Wildcard
	for _, child := range parent.Children() {
		el, ok := child.(*Element)
		if !ok {
			continue
		}
		if el.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch el.Name().Local() {
		case "attribute":
			use, err := p.produceAttributeUse(el)
			if err != nil {
				return nil, nil, err
			}
			if use != nil {
				uses = append(uses, *use)
			}
		case "anyAttribute":
			wc, err := p.produceWildcard(el)
			if err != nil {
				return nil, nil, err
			}
			wildcard = &wc
		case "attributeGroup":
			return nil, nil, fmt.Errorf("parser: <attributeGroup ref> is not yet produced (needs a top-level attribute group definition, §3.6.2)")
		}
	}
	return uses, wildcard, nil
}

// produceAttributeUse maps a local <attribute> to an Attribute Use (§3.2.2.2,
// dcl.att.local). use="prohibited" maps to no Attribute Use (returns nil). An
// <attribute ref="..."> yields a deferred AttributeDeclarationRef; otherwise a
// sibling local Attribute Declaration is built inline. It enforces the structural
// src-attribute clauses (§3.2.3): 1 (default and fixed mutually exclusive) and 3
// (exactly one of ref/name; ref excludes type/form).
func (p *producer) produceAttributeUse(el *Element) (*xsd.AttributeUse, error) {
	use, _ := attrValue(el, "use") // default "optional"
	if use == "prohibited" {
		return nil, nil
	}
	_, hasDefault := attrValue(el, "default")
	_, hasFixed := attrValue(el, "fixed")
	if hasDefault && hasFixed {
		return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
			"attribute has both default and fixed, but src-attribute clause 1 forbids both")
	}
	required := use == "required"
	inheritable, _ := boolAttr(el, "inheritable")

	ref, hasRef := attrValue(el, "ref")
	_, hasName := attrValue(el, "name")
	if hasRef {
		if hasName {
			return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
				"attribute has both ref and name, but src-attribute clause 3 requires exactly one")
		}
		if childElement(el, xsd.XMLSchemaNS, "simpleType") != nil {
			return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
				"attribute has both ref and an inline <simpleType>, but src-attribute clause 3 forbids a type with ref")
		}
		if _, hasType := attrValue(el, "type"); hasType {
			return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
				"attribute has both ref and type, but src-attribute clause 3 forbids a type with ref")
		}
		qn, err := resolveQName(el, ref)
		if err != nil {
			return nil, err
		}
		au, err := xsd.NewAttributeUse(el.Loc(), required, xsd.AttributeDeclarationRef{Name: qn}, nil, inheritable, nil)
		if err != nil {
			return nil, err
		}
		return &au, nil
	}
	if !hasName {
		return nil, xsderr.New(ruleSrcAttribute, el.Loc(),
			"attribute has neither ref nor name, but src-attribute clause 3 requires exactly one")
	}
	decl, err := p.produceLocalAttribute(el)
	if err != nil {
		return nil, err
	}
	au, err := xsd.NewAttributeUse(el.Loc(), required, xsd.LocalAttributeDeclaration{Declaration: decl}, nil, inheritable, nil)
	if err != nil {
		return nil, err
	}
	return &au, nil
}

// produceLocalAttribute maps the sibling local Attribute Declaration of a local
// <attribute> (§3.2.2.2, {scope} = local, {value constraint} always absent on the
// declaration — any default/fixed feeds the Attribute Use, #70). type= form only:
// an inline <simpleType> is declined (src-attribute clause 4 is the both-present
// rule; the inline-only form is simply not yet produced). A type=-less attribute
// defaults its {type definition} to xs:anySimpleType (§3.2.2.1).
func (p *producer) produceLocalAttribute(el *Element) (xsd.AttributeDeclaration, error) {
	if childElement(el, xsd.XMLSchemaNS, "simpleType") != nil {
		if _, hasType := attrValue(el, "type"); hasType {
			return xsd.AttributeDeclaration{}, xsderr.New(ruleSrcAttribute, el.Loc(),
				"attribute has both a type attribute and an inline <simpleType> child, but src-attribute clause 4 forbids both")
		}
		return xsd.AttributeDeclaration{}, fmt.Errorf("parser: a local <attribute> with an inline <simpleType> is not yet produced (type= form only)")
	}
	name, _ := attrValue(el, "name")
	qname := xsd.QName{Space: p.localTargetNS(el, "attributeFormDefault"), Local: name}
	typeName := xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"}
	if typeLex, hasType := attrValue(el, "type"); hasType {
		qn, err := resolveQName(el, typeLex)
		if err != nil {
			return xsd.AttributeDeclaration{}, err
		}
		typeName = qn
	}
	inheritable, _ := boolAttr(el, "inheritable")
	return xsd.NewAttributeDeclaration(el.Loc(), qname, typeName, xsd.ScopeLocal, nil, inheritable, nil)
}

// produceWildcard maps an <any>/<anyAttribute> to a Wildcard (§3.10.2.2). It
// enforces src-wildcard (§3.10.3): namespace and notNamespace must not both be
// present.
func (p *producer) produceWildcard(el *Element) (xsd.Wildcard, error) {
	nc, err := p.namespaceConstraint(el)
	if err != nil {
		return xsd.Wildcard{}, err
	}
	process := xsd.ProcessStrict
	if pc, ok := attrValue(el, "processContents"); ok {
		process, err = processContentsOf(pc, el.Loc())
		if err != nil {
			return xsd.Wildcard{}, err
		}
	}
	return xsd.NewWildcard(el.Loc(), nc, process, nil)
}

// namespaceConstraint maps the namespace/notNamespace/notQName attributes of an
// <any>/<anyAttribute> to a Namespace Constraint (§3.10.2.2).
func (p *producer) namespaceConstraint(el *Element) (xsd.NamespaceConstraint, error) {
	ns, hasNS := attrValue(el, "namespace")
	notNS, hasNotNS := attrValue(el, "notNamespace")
	if hasNS && hasNotNS {
		return xsd.NamespaceConstraint{}, xsderr.New(ruleSrcWildcard, el.Loc(),
			"wildcard has both namespace and notNamespace, but src-wildcard forbids both")
	}
	disallowed := p.disallowedNames(el)

	variety, namespaces := p.namespaceVarietyAndSet(ns, hasNS, notNS, hasNotNS)
	return xsd.NewNamespaceConstraint(el.Loc(), variety, namespaces, disallowed)
}

// namespaceVarietyAndSet computes {variety} and {namespaces} (§3.10.2.2):
//   - neither namespace nor notNamespace present → any, empty set;
//   - namespace="##any" → any, empty set;
//   - namespace="##other" → not, {·absent·} plus the target namespace if present;
//   - otherwise (a namespace/notNamespace token list) → enumeration for namespace
//     or not for notNamespace, with ##targetNamespace/##local substituted.
func (p *producer) namespaceVarietyAndSet(ns string, hasNS bool, notNS string, hasNotNS bool) (xsd.NamespaceConstraintVariety, []xsd.Namespace) {
	if !hasNS && !hasNotNS {
		return xsd.NamespaceConstraintAny, nil
	}
	if hasNS && ns == "##any" {
		return xsd.NamespaceConstraintAny, nil
	}
	if hasNS && ns == "##other" {
		set := []xsd.Namespace{xsd.NamespaceName("")}
		if p.target != "" {
			set = append(set, xsd.NamespaceName(p.target))
		}
		return xsd.NamespaceConstraintNot, set
	}
	list := ns
	variety := xsd.NamespaceConstraintEnumeration
	if hasNotNS {
		list = notNS
		variety = xsd.NamespaceConstraintNot
	}
	var set []xsd.Namespace
	for _, tok := range strings.Fields(list) {
		switch tok {
		case "##targetNamespace":
			set = append(set, xsd.NamespaceName(p.target))
		case "##local":
			set = append(set, xsd.NamespaceName(""))
		default:
			set = append(set, xsd.NamespaceName(tok))
		}
	}
	return variety, set
}

// disallowedNames maps the literal QName items of a notQName attribute to
// {disallowed names} (§3.10.2.2). The ##defined/##definedSibling keyword tokens
// are not modelled by xsd.NamespaceConstraint (its documented GAP) and are
// skipped here rather than mis-mapped.
func (p *producer) disallowedNames(el *Element) []xsd.QName {
	notQName, ok := attrValue(el, "notQName")
	if !ok {
		return nil
	}
	var names []xsd.QName
	for _, tok := range strings.Fields(notQName) {
		if strings.HasPrefix(tok, "##") {
			// GAP: ##defined/##definedSibling need the live declaration graph; the
			// xsd package does not model the keywords, so they are not applied here.
			continue
		}
		qn, err := resolveQName(el, tok)
		if err != nil {
			continue // an unresolvable notQName member is dropped, not fatal
		}
		names = append(names, qn)
	}
	return names
}

// localTargetNS computes a local element/attribute declaration's {target
// namespace} (§3.3.2.3 / §3.2.2.2): an explicit targetNamespace attribute wins,
// else form= (qualified → the schema target, unqualified → absent), else the
// schema's *FormDefault (formDefaultAttr is "elementFormDefault" or
// "attributeFormDefault"), defaulting to absent (unqualified).
func (p *producer) localTargetNS(el *Element, formDefaultAttr string) string {
	if tns, ok := attrValue(el, "targetNamespace"); ok {
		return tns
	}
	if form, ok := attrValue(el, "form"); ok {
		if form == "qualified" {
			return p.target
		}
		return ""
	}
	if fd, ok := attrValue(p.schemaElem, formDefaultAttr); ok && fd == "qualified" {
		return p.target
	}
	return ""
}

// occursOf maps the minOccurs/maxOccurs attributes to an Occurs (§3.9.2), each
// defaulting to 1. elided is true for the minOccurs=maxOccurs=0 case, which the
// XML mapping rules say "maps to no component at all" (§3.7.2/§3.8.2/§3.9.2) — the
// caller omits the particle entirely rather than building a vacuous Occurs{0,0}.
func occursOf(el *Element) (occ xsd.Occurs, elided bool, err error) {
	min := 1
	if minS, ok := attrValue(el, "minOccurs"); ok {
		min, err = nonNegativeInt(minS, el.Loc(), "minOccurs")
		if err != nil {
			return xsd.Occurs{}, false, err
		}
	}
	unbounded := false
	max := 1
	if maxS, ok := attrValue(el, "maxOccurs"); ok {
		if strings.TrimSpace(maxS) == "unbounded" {
			unbounded = true
		} else {
			max, err = nonNegativeInt(maxS, el.Loc(), "maxOccurs")
			if err != nil {
				return xsd.Occurs{}, false, err
			}
		}
	}
	if !unbounded && min == 0 && max == 0 {
		return xsd.Occurs{}, true, nil
	}
	if unbounded {
		occ, err = xsd.NewUnboundedOccurs(el.Loc(), min)
		return occ, false, err
	}
	occ, err = xsd.NewOccurs(el.Loc(), min, max)
	return occ, false, err
}

// nonNegativeInt parses an xs:nonNegativeInteger-valued occurrence attribute,
// charging p-props-correct (§3.9.6.1) on a malformed or negative value.
func nonNegativeInt(lexical string, loc xsderr.Loc, attr string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(lexical))
	if err != nil || n < 0 {
		return 0, xsderr.New(ruleParticleCorr, loc,
			"%s value %q is not a nonNegativeInteger (p-props-correct)", attr, lexical)
	}
	return n, nil
}

// processContentsOf maps a processContents lexical to a ProcessContents token,
// charging w-props-correct (§3.10.6.1) on an out-of-range value.
func processContentsOf(lexical string, loc xsderr.Loc) (xsd.ProcessContents, error) {
	switch strings.TrimSpace(lexical) {
	case "skip":
		return xsd.ProcessSkip, nil
	case "strict":
		return xsd.ProcessStrict, nil
	case "lax":
		return xsd.ProcessLax, nil
	}
	return 0, xsderr.New(ruleWildcardCorr, loc,
		"wildcard processContents %q is not one of skip/strict/lax", lexical)
}

// compositorOf maps an <all>/<choice>/<sequence> local name to its Compositor.
// ok is false for <group> (a reference, out of scope) and any other name.
func compositorOf(local string) (xsd.Compositor, bool) {
	switch local {
	case "all":
		return xsd.CompositorAll, true
	case "choice":
		return xsd.CompositorChoice, true
	case "sequence":
		return xsd.CompositorSequence, true
	}
	return 0, false
}

// modelGroupChild returns el's first <all>/<choice>/<sequence>/<group> child (the
// model-group child of a <complexType>/<restriction>/<extension>), or nil.
func modelGroupChild(el *Element) *Element {
	for _, child := range el.Children() {
		c, ok := child.(*Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		switch c.Name().Local() {
		case "all", "choice", "sequence", "group":
			return c
		}
	}
	return nil
}

// hasParticleChild reports whether group has any non-<annotation> element child
// (an element/any/group/choice/sequence), i.e. whether it is non-empty for the
// §3.4.2.3.3 clause 2 empty-content tests.
func hasParticleChild(group *Element) bool {
	for _, child := range group.Children() {
		c, ok := child.(*Element)
		if !ok || c.Name().Space() != xsd.XMLSchemaNS {
			continue
		}
		if c.Name().Local() != "annotation" {
			return true
		}
	}
	return false
}

// minOccursZero reports whether el's minOccurs actual value is 0.
func minOccursZero(el *Element) bool {
	v, ok := attrValue(el, "minOccurs")
	return ok && strings.TrimSpace(v) == "0"
}

// maxOccursZero reports whether el's maxOccurs actual value is 0.
func maxOccursZero(el *Element) bool {
	v, ok := attrValue(el, "maxOccurs")
	return ok && strings.TrimSpace(v) == "0"
}

// attrOr returns el's attribute local value, or the empty string when absent.
func attrOr(el *Element, local string) string {
	v, _ := attrValue(el, local)
	return v
}

// boolAttr reads an xs:boolean-valued attribute (true/1 → true), reporting
// presence. An absent attribute is (false, false).
func boolAttr(el *Element, local string) (val bool, present bool) {
	v, ok := attrValue(el, local)
	if !ok {
		return false, false
	}
	return v == "true" || v == "1", true
}
