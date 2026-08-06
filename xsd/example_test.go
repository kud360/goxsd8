package xsd_test

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// exampleNS is the target namespace every name in these Examples lives in.
const exampleNS = "urn:example:po"

// anySimpleTypeName is xs:anySimpleType's expanded name. The Examples reference
// it as their one {type definition} so every declaration resolves at Finalize
// without seeding the whole builtin datatype set (parser.Produce's job).
var anySimpleTypeName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"}

// exampleQName expands local into exampleNS.
func exampleQName(local string) xsd.QName {
	return xsd.QName{Space: exampleNS, Local: local}
}

// exampleParticle wraps term in a {1,1} particle. An Example has no *testing.T,
// so a constructor rejection panics (the convention
// ExampleNamespaceConstraint_AllowsNamespace already uses).
func exampleParticle(term xsd.TermOrRef) xsd.Particle {
	occurs, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		panic(err)
	}
	p, err := xsd.NewParticle(xsderr.Loc{}, occurs, term, nil)
	if err != nil {
		panic(err)
	}
	return p
}

// exampleModelGroup builds a model group with the given compositor over
// particles in document order.
func exampleModelGroup(compositor xsd.Compositor, particles ...xsd.Particle) xsd.ModelGroup {
	g, err := xsd.NewModelGroup(xsderr.Loc{}, compositor, particles, nil)
	if err != nil {
		panic(err)
	}
	return g
}

// exampleAnyWildcard builds a lax ##any element wildcard.
func exampleAnyWildcard() xsd.Wildcard {
	nc, err := xsd.NewNamespaceConstraint(xsderr.Loc{}, xsd.NamespaceConstraintAny, nil, nil, nil)
	if err != nil {
		panic(err)
	}
	w, err := xsd.NewWildcard(xsderr.Loc{}, nc, xsd.ProcessLax, nil)
	if err != nil {
		panic(err)
	}
	return w
}

// exampleElement builds an element declaration of the given scope whose {type
// definition} is a by-name reference to typeName (the xsd.TypeDefinitionRef arm
// of the slot's sealed sum; the other arm owns an inline anonymous type).
func exampleElement(name, typeName xsd.QName, scope xsd.Scope) xsd.ElementDeclaration {
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: typeName}, nil, scope, nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		panic(err)
	}
	return e
}

// exampleLocalElement builds a local (in-content-model) element declaration of
// type xs:anySimpleType, scoped to the complex type container that holds it
// (§3.3.1 {scope}: a local declaration is available only within its {parent}).
func exampleLocalElement(container xsd.QName, local string) xsd.ElementDeclaration {
	scope, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: container})
	if err != nil {
		panic(err)
	}
	return exampleElement(exampleQName(local), anySimpleTypeName, scope)
}

// exampleElementOnlyType builds a named element-only complex type whose
// {content type} is a {1,1} particle over term.
func exampleElementOnlyType(name xsd.QName, term xsd.TermOrRef) xsd.ComplexType {
	ct, err := xsd.NewComplexType(xsderr.Loc{}, name, xsd.QName{}, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.ElementContent{Particle: exampleParticle(term)}, nil, nil, nil)
	if err != nil {
		panic(err)
	}
	return ct
}

// Example_termOrRefDiscrimination shows the TWO-LEVEL discrimination a
// particle's {term} slot needs (§3.9.1). Level 1 switches TermOrRef's own sealed
// variants — ResolvedTerm, ElementDeclarationRef, ModelGroupRef; level 2
// switches the Term sealed sum (§3.9 "Term": Element Declaration, Wildcard,
// Model Group) reached through ResolvedTerm.Term, and ONLY there. The two sums
// are disjoint: a Term variant does not satisfy TermOrRef, so a consumer that
// tries to skip level 1 gets no match at all.
func Example_termOrRefDiscrimination() {
	shipTo := exampleLocalElement(exampleQName("PurchaseOrderType"), "shipTo")
	slots := []xsd.TermOrRef{
		xsd.ElementDeclarationRef{Name: exampleQName("billTo")},
		xsd.ModelGroupRef{Name: exampleQName("addressGroup")},
		xsd.ResolvedTerm{Term: shipTo},
		xsd.ResolvedTerm{Term: exampleAnyWildcard()},
		xsd.ResolvedTerm{Term: exampleModelGroup(xsd.CompositorChoice, exampleParticle(xsd.ResolvedTerm{Term: shipTo}))},
	}
	for _, slot := range slots {
		// Level 1: the {term} slot itself. A ref variant carries only a QName;
		// Finalize proved it resolves (src-resolve) but did not rewrite the slot,
		// so following it is a read-time schema lookup, not a field access.
		switch v := slot.(type) {
		case xsd.ElementDeclarationRef:
			fmt.Println("ref to element declaration", v.Name)
		case xsd.ModelGroupRef:
			fmt.Println("ref to model group definition", v.Name)
		case xsd.ResolvedTerm:
			// Level 2: only a ResolvedTerm has a Term to discriminate.
			switch t := v.Term.(type) {
			case xsd.ElementDeclaration:
				fmt.Println("inline element declaration", t.Name())
			case xsd.Wildcard:
				fmt.Println("inline wildcard, processContents", t.ProcessContents())
			case xsd.ModelGroup:
				fmt.Println("inline model group, compositor", t.Compositor())
			}
		}
	}

	// The one-level shortcut is not merely a miss, it does not compile:
	//
	//	_, ok := slot.(xsd.ElementDeclaration) // impossible type assertion
	//
	// because ElementDeclaration implements Term, never TermOrRef.
	termOrRef := reflect.TypeOf((*xsd.TermOrRef)(nil)).Elem()
	fmt.Println("ElementDeclaration is a TermOrRef:", reflect.TypeOf(shipTo).Implements(termOrRef))
	fmt.Println("ResolvedTerm is a TermOrRef:", reflect.TypeOf(xsd.ResolvedTerm{}).Implements(termOrRef))

	// Output:
	// ref to element declaration {urn:example:po}billTo
	// ref to model group definition {urn:example:po}addressGroup
	// inline element declaration {urn:example:po}shipTo
	// inline wildcard, processContents lax
	// inline model group, compositor choice
	// ElementDeclaration is a TermOrRef: false
	// ResolvedTerm is a TermOrRef: true
}

// Example_buildFinalizeQuery goes end to end over the construct → Finalize →
// query sequence: NewSchemaBuilder accumulates components in document order,
// Finalize turns the accumulator into the immutable *Schema (rejecting
// sch-props-correct clause 2 duplicates and unresolvable references, and
// returning a NIL *Schema on any error), and the finalized Schema answers the
// three Query capability views — Type, Element, Attribute by expanded QName.
func Example_buildFinalizeQuery() {
	b := xsd.NewSchemaBuilder()
	// A real producer seeds the builtin datatypes (builtin.Seed, called from
	// parser.Produce); this hand-built schema needs only the one type its
	// declarations reference.
	b.AddType(xsd.AnySimpleType())
	addressType := exampleQName("AddressType")
	b.AddType(exampleElementOnlyType(addressType,
		xsd.ResolvedTerm{Term: exampleModelGroup(xsd.CompositorSequence,
			exampleParticle(xsd.ResolvedTerm{Term: exampleLocalElement(addressType, "street")}),
			exampleParticle(xsd.ResolvedTerm{Term: exampleLocalElement(addressType, "city")}),
		)}))
	b.AddElement(exampleElement(exampleQName("shipTo"), addressType, xsd.NewGlobalScope()))
	country, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, exampleQName("country"), xsd.TypeDefinitionRef{Name: anySimpleTypeName}, xsd.ScopeGlobal, nil, false, nil)
	if err != nil {
		panic(err)
	}
	b.AddAttribute(country)

	s, err := b.Finalize()
	if err != nil {
		// s is nil here — never dereference it on the error path.
		fmt.Println("finalize:", err)
		return
	}

	e, ok := s.Element(exampleQName("shipTo"))
	// {type definition} is a sealed sum: the by-name arm here, an owned inline
	// anonymous type for a declaration written with an inline <simpleType>.
	shipToType, byName := e.TypeDefinition().(xsd.TypeDefinitionRef)
	if !byName {
		panic("shipTo's type is not a by-name reference")
	}
	fmt.Println("element shipTo:", ok, "type", shipToType.Name)
	t, ok := s.Type(exampleQName("AddressType"))
	fmt.Println("type AddressType:", ok, t.Name())
	a, ok := s.Attribute(exampleQName("country"))
	fmt.Println("attribute country:", ok, a.Name())
	_, ok = s.Element(exampleQName("nosuch"))
	fmt.Println("element nosuch:", ok)

	// Output:
	// element shipTo: true type {urn:example:po}AddressType
	// type AddressType: true {urn:example:po}AddressType
	// attribute country: true {urn:example:po}country
	// element nosuch: false
}

// exampleAttribute builds a global attribute declaration of type
// xs:anySimpleType.
func exampleAttribute(name xsd.QName) xsd.AttributeDeclaration {
	a, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: anySimpleTypeName}, xsd.ScopeGlobal, nil, false, nil)
	if err != nil {
		panic(err)
	}
	return a
}

// exampleAttributeGroup builds an empty named attribute group definition.
func exampleAttributeGroup(name xsd.QName) xsd.AttributeGroupDefinition {
	g, err := xsd.NewAttributeGroupDefinition(xsderr.Loc{}, name, nil, nil, nil)
	if err != nil {
		panic(err)
	}
	return g
}

// exampleModelGroupDefinition builds a named group over an empty sequence.
func exampleModelGroupDefinition(name xsd.QName) xsd.ModelGroupDefinition {
	d, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, name, exampleModelGroup(xsd.CompositorSequence), nil)
	if err != nil {
		panic(err)
	}
	return d
}

// exampleNotation builds a notation declaration with a system identifier.
func exampleNotation(name xsd.QName, systemID string) xsd.Notation {
	n, err := xsd.NewNotation(xsderr.Loc{}, name, &systemID, nil, nil)
	if err != nil {
		panic(err)
	}
	return n
}

// exampleIdentityConstraint builds a keyref-free unique constraint over one
// field.
func exampleIdentityConstraint(name xsd.QName) xsd.IdentityConstraint {
	c, err := xsd.NewIdentityConstraint(xsderr.Loc{}, name, xsd.IdentityConstraintUnique,
		xsd.NewXPathExpression(".", nil, nil, nil),
		[]xsd.XPathExpression{xsd.NewXPathExpression("@sku", nil, nil, nil)}, nil, nil)
	if err != nil {
		panic(err)
	}
	return c
}

// Example_schemaEnumeration lists a finalized schema's §3.17.1 properties. Next
// to the by-QName Query views, *Schema enumerates all eight properties in
// document order — the order components were added, which is a guarantee even
// though §3.17.1 words seven of the eight as unordered sets ({annotations},
// alone, is a sequence there too.)
//
// Each enumerator returns a COPY of its slice, so writing through the result
// cannot reach the compiled set; the components inside are shared and
// immutable. Note Types() holds only TOP-LEVEL type definitions: the local
// element's type is reached by walking the content model, not from this list.
func Example_schemaEnumeration() {
	addressType := exampleQName("AddressType")
	b := xsd.NewSchemaBuilder()
	b.AddType(xsd.AnySimpleType())
	b.AddType(exampleElementOnlyType(addressType,
		xsd.ResolvedTerm{Term: exampleModelGroup(xsd.CompositorSequence,
			exampleParticle(xsd.ResolvedTerm{Term: exampleLocalElement(addressType, "street")}),
		)}))
	b.AddElement(exampleElement(exampleQName("shipTo"), addressType, xsd.NewGlobalScope()))
	b.AddAttribute(exampleAttribute(exampleQName("country")))
	b.AddAttributeGroup(exampleAttributeGroup(exampleQName("common")))
	b.AddModelGroup(exampleModelGroupDefinition(exampleQName("nameGroup")))
	b.AddNotation(exampleNotation(exampleQName("jpeg"), "image/jpeg"))
	b.AddIdentityConstraint(exampleIdentityConstraint(exampleQName("skuKey")))
	// parser.Parse wires no schema-level annotation today; a producer calling
	// AddAnnotation itself is what Annotations() reports.
	b.AddAnnotation(xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "purchase orders")}, nil))

	s, err := b.Finalize()
	if err != nil {
		panic(err)
	}

	for _, d := range s.Types() {
		fmt.Println("type:", d.Name())
	}
	for _, d := range s.Elements() {
		fmt.Println("element:", d.Name().Local)
	}
	for _, d := range s.Attributes() {
		fmt.Println("attribute:", d.Name().Local)
	}
	for _, d := range s.AttributeGroups() {
		fmt.Println("attribute group:", d.Name().Local)
	}
	for _, d := range s.ModelGroups() {
		fmt.Println("model group:", d.Name().Local)
	}
	for _, d := range s.Notations() {
		fmt.Println("notation:", d.Name().Local)
	}
	for _, d := range s.IdentityConstraints() {
		fmt.Println("identity constraint:", d.Name().Local)
	}
	for _, a := range s.Annotations() {
		fmt.Println("annotation:", a.Documentation()[0].Content())
	}

	elements := s.Elements()
	elements[0] = xsd.ElementDeclaration{}
	fmt.Println("after clearing the returned slice, element:", s.Elements()[0].Name().Local)

	// Output:
	// type: {http://www.w3.org/2001/XMLSchema}anySimpleType
	// type: {urn:example:po}AddressType
	// element: shipTo
	// attribute: country
	// attribute group: common
	// model group: nameGroup
	// notation: jpeg
	// identity constraint: skuKey
	// annotation: purchase orders
	// after clearing the returned slice, element: shipTo
}

// Example_contentModelTraversal walks a global element's content model down to
// its leaf element declarations, the five-type chain a consumer needs first:
// Schema.Element → ElementDeclaration.TypeDefinition → Schema.Type →
// ComplexType.ContentType → ElementContent.Particle → Particle.Term (a
// TermOrRef) → Term → ModelGroup.Particles, recursing on each nested particle.
//
// The model mixes inline groups with one <element ref> so both TermOrRef shapes
// appear; a ModelGroupRef is deliberately absent, because following one needs a
// Schema.ModelGroup(QName) accessor this package does not export yet (see
// resolve.go's recorded follow-cost asymmetry).
func Example_contentModelTraversal() {
	// PurchaseOrderType: sequence( shipTo, choice( comment, note ), ref billTo )
	poType := exampleQName("PurchaseOrderType")
	b := xsd.NewSchemaBuilder()
	b.AddType(xsd.AnySimpleType())
	b.AddType(exampleElementOnlyType(poType,
		xsd.ResolvedTerm{Term: exampleModelGroup(xsd.CompositorSequence,
			exampleParticle(xsd.ResolvedTerm{Term: exampleLocalElement(poType, "shipTo")}),
			exampleParticle(xsd.ResolvedTerm{Term: exampleModelGroup(xsd.CompositorChoice,
				exampleParticle(xsd.ResolvedTerm{Term: exampleLocalElement(poType, "comment")}),
				exampleParticle(xsd.ResolvedTerm{Term: exampleLocalElement(poType, "note")}),
			)}),
			exampleParticle(xsd.ElementDeclarationRef{Name: exampleQName("billTo")}),
		)}))
	b.AddElement(exampleElement(exampleQName("purchaseOrder"), poType, xsd.NewGlobalScope()))
	b.AddElement(exampleElement(exampleQName("billTo"), anySimpleTypeName, xsd.NewGlobalScope()))
	s, err := b.Finalize()
	if err != nil {
		panic(err)
	}

	// Step 1: the global element declaration, and the type its {type definition}
	// QName names. The reference is a read-time lookup: Finalize validated it
	// (src-resolve clause 1.1) but stored no resolved pointer.
	root, ok := s.Element(exampleQName("purchaseOrder"))
	if !ok {
		panic("purchaseOrder not declared")
	}
	rootType, byName := root.TypeDefinition().(xsd.TypeDefinitionRef)
	if !byName {
		panic("purchaseOrder's type is not a by-name reference")
	}
	def, ok := s.Type(rootType.Name)
	if !ok {
		panic("purchaseOrder type not declared")
	}

	// Step 2: only a ComplexType has a content model, and only the element-only
	// and mixed varieties (ElementContent) carry a {particle}.
	ct, ok := def.(xsd.ComplexType)
	if !ok {
		panic("purchaseOrder is not complex-typed")
	}
	content, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		fmt.Println(ct.Name(), "has", ct.ContentType().Variety(), "content, no particle")
		return
	}
	fmt.Println(ct.Name(), "content type:", content.Variety())

	// Step 3: recurse through the particle tree. Nesting terminates by
	// construction — Finalize rejected the circular group graphs (PRINCIPLES 9),
	// so no visited set is needed.
	var walk func(p xsd.Particle, depth int)
	walk = func(p xsd.Particle, depth int) {
		indent := strings.Repeat("  ", depth)
		switch v := p.Term().(type) {
		case xsd.ResolvedTerm:
			switch t := v.Term.(type) {
			case xsd.ElementDeclaration:
				fmt.Printf("%selement %s\n", indent, t.Name().Local)
			case xsd.Wildcard:
				fmt.Printf("%swildcard\n", indent)
			case xsd.ModelGroup:
				fmt.Printf("%s%s\n", indent, t.Compositor())
				for _, child := range t.Particles() {
					walk(child, depth+1)
				}
			}
		case xsd.ElementDeclarationRef:
			target, found := s.Element(v.Name)
			if !found {
				panic("Finalize admitted an unresolvable element ref")
			}
			fmt.Printf("%selement %s (via ref)\n", indent, target.Name().Local)
		case xsd.ModelGroupRef:
			fmt.Printf("%sgroup ref %s (not followable: no Schema.ModelGroup accessor)\n", indent, v.Name)
		}
	}
	walk(content.Particle, 1)

	// Output:
	// {urn:example:po}PurchaseOrderType content type: element-only
	//   sequence
	//     element shipTo
	//     choice
	//       element comment
	//       element note
	//     element billTo (via ref)
}
