package xsd

import "github.com/kud360/goxsd8/xsderr"

// componentWalk is THE descent over the component tree a Complex Type Definition
// roots, shared by every finalize phase that needs one (STYLE T4). A phase fills
// in only what it CHARGES at each kind of component the descent reaches; which
// slots exist, and which arm of each is followed, is decided here once.
//
// {base type definition} IS WALKED, and it is the slot a hand-written copy of
// this descent omits (#843). It is the only slot that reaches the {name}-·absent·
// original a <redefine> mints: src-expredef clause 1.1 gives that original an
// absent {name}, and clause 1.2 puts it in the redefining type's {base type
// definition} and nowhere else. A descent that skips the slot therefore sees
// neither that original's content model nor its attribute uses, and the rules
// quantified over them go uncharged — a fail-open no conformance case exhibits.
//
// A NIL FIELD IS "THIS PHASE OWES THAT KIND NOTHING", never "skip that part of
// the tree". The descent is the same for every phase; a component a phase
// charges nothing at is still walked THROUGH to the components below it.
//
// OWN SLOTS BEFORE NESTED COMPONENTS: a component's own charges are made before
// the descent enters anything it nests, so a schema wrong at two depths reports
// the outer failure (STYLE D1). Slices are walked in document order (STYLE D2).
//
// BY-NAME ARMS ARE CHARGED, NEVER FOLLOWED. A TypeDefinitionRef base, an
// <element ref>, a <group ref> and an <attribute ref> each name a TOP-LEVEL
// component, reached from a phase's ROOT LOOPS and never from a referring site,
// so following one would re-walk it once per site. A phase whose roots omit one
// of those tables owes nothing at what it holds: checkComponentValueConstraints
// and checkTypeTableSubstitutability enumerate no s.attributes, because a global
// attribute declaration nests only a simple {type definition} and neither phase
// charges a simple type — a-props-correct clause 2 against the declaration
// itself is checkAttributeDeclarationDefaults'. What the descent follows is
// OWNERSHIP — an inline type definition, an attribute use's local declaration, a
// particle's inline term — and an owned component must pre-exist the slot holding
// it, which is what makes the walked structure a finite tree.
//
// NO VISITED SET AND NO CYCLE GUARD (STYLE D4, PRINCIPLES 9). The edges that
// could close a cycle are exactly the by-name ones the descent does not follow.
// A circular {base type definition} chain is rejected by ct-props-correct clause
// 3 in Phase B (checkComplexBaseAcyclic); a seen-set here would mask that
// check's absence rather than defend against anything.
//
// {attribute wildcard}, {open content} and {assertions} carry no field. Each is
// a leaf of THIS tree — none nests a type definition, an attribute use or an
// element declaration — so a field for one would be parameterization no phase
// fills (STYLE T5). A phase that comes to charge a wildcard or an assertion adds
// the field here, and every phase gets the slot at once.
type componentWalk struct {
	// typeDefinitionSlot is charged on a {type definition}/{base type definition}
	// slot in every arm, including the inline one the descent then enters. loc
	// positions a rejection at the component holding the slot and ctx names the
	// slot for the message (the referrer-Loc convention, resolveReferences).
	typeDefinitionSlot func(ref TypeDefinitionOrRef, loc xsderr.Loc, ctx string) error
	// attributeUse is charged on an Attribute Use in both arms. loc is the
	// enclosing complex type's or attribute group definition's position and owner
	// names it, because an Attribute Use retains no position of its own.
	attributeUse func(u AttributeUse, loc xsderr.Loc, owner string) error
	// elementDeclaration is charged on an Element Declaration, global or local; it
	// retains its own Loc, so it takes neither loc nor owner.
	elementDeclaration func(e ElementDeclaration) error
	// simpleType is handed every *SimpleType the tree OWNS — an inline {type
	// definition}, an inline {base type definition}, SimpleContent's {simple type
	// definition}. The simple-type graph below it is a different tree with a
	// different set of followable arms, so this descent stops here and the phase's
	// own simple-type walk takes over.
	simpleType func(t *SimpleType) error
	// termRef is charged on a particle's BY-NAME {term} — an <element ref> or a
	// <group ref>. An inline term is descended instead and never reaches it.
	termRef func(t TermOrRef, loc xsderr.Loc) error
}

// walkTypeRoot enters one member of a schema's {type definitions}, which is a
// TypeDefinition and not a slot: it is held by no referring component, so there
// is nothing to charge typeDefinitionSlot for.
func (w componentWalk) walkTypeRoot(t TypeDefinition) error {
	switch t := t.(type) {
	case ComplexType:
		return w.walkComplexType(t)
	case *SimpleType:
		return w.walkSimpleType(t)
	default:
		panic("xsd: componentWalk.walkTypeRoot: non-exhaustive TypeDefinition switch")
	}
}

// walkComplexType descends one Complex Type Definition: its {base type
// definition}, then each of its {attribute uses}, then its {content type} —
// SimpleContent's {simple type definition} or ElementContent's particle tree,
// EmptyContent nesting nothing.
//
// The base slot comes first because a fault in a base is a fault the type
// inherits, so a reader meeting two is sent to the one the other is built on.
//
// This is where the referrer-Loc convention re-roots: everything below is
// positioned at c.Loc() until a nested declaration that retains its own position
// takes over. For an anonymous complex type that Loc is its own <complexType>
// element, which is nearer than the owning declaration's.
func (w componentWalk) walkComplexType(c ComplexType) error {
	owner := complexTypeOwner(c)
	if err := w.walkTypeDefinition(c.Base(), c.Loc(), owner+" {base type definition}"); err != nil {
		return err
	}
	for _, u := range c.AttributeUses() {
		if err := w.walkAttributeUse(u, c.Loc(), owner); err != nil {
			return err
		}
	}
	switch ct := c.ContentType().(type) {
	case EmptyContent:
		return nil
	case SimpleContent:
		return w.walkSimpleType(ct.SimpleType)
	case ElementContent:
		return w.walkParticle(ct.Particle, c.Loc())
	default:
		panic("xsd: componentWalk.walkComplexType: non-exhaustive ContentType switch")
	}
}

// complexTypeOwner renders c as the owner phrase of a rejection message. An
// inline <xs:complexType> has no {name} (the zero QName, whose String is ""), so
// naming it would leave a hole in the message; it is described by what it is
// instead.
func complexTypeOwner(c ComplexType) string {
	n := c.Name()
	if n == (QName{}) {
		return "anonymous complex type"
	}
	return "complex type " + n.String()
}

// walkTypeDefinition charges one {type definition}/{base type definition} slot
// and then enters the component it OWNS. Only the InlineTypeDefinition arm is
// entered: a TypeDefinitionRef names a top-level type the phase's root loop
// reaches in its own right, and a SubstitutionGroupHeadTypeRef names the HEAD
// declaration that owns the anonymous type, which is itself a member of the
// schema's {element declarations} (entering from the member too would charge the
// same components twice and report a failure at the wrong declaration).
func (w componentWalk) walkTypeDefinition(ref TypeDefinitionOrRef, loc xsderr.Loc, ctx string) error {
	if w.typeDefinitionSlot != nil {
		if err := w.typeDefinitionSlot(ref, loc, ctx); err != nil {
			return err
		}
	}
	return w.enterTypeDefinition(ref)
}

// enterTypeDefinition is walkTypeDefinition's descent half, split out for the
// one site that must charge something between the slot's own verdict and the
// component below it (walkElementDeclaration).
func (w componentWalk) enterTypeDefinition(ref TypeDefinitionOrRef) error {
	inline, ok := ref.(InlineTypeDefinition)
	if !ok {
		return nil
	}
	switch d := inline.Definition.(type) {
	case *SimpleType:
		return w.walkSimpleType(d)
	case ComplexType:
		return w.walkComplexType(d)
	default:
		panic("xsd: componentWalk.enterTypeDefinition: non-exhaustive TypeDefinition switch")
	}
}

// walkSimpleType hands a *SimpleType to the phase, which owns the descent below
// it. A nil t — an owned arm that was absent — is passed through unexamined: only
// the phase knows whether absence is a fault under the rule it charges, so
// deciding it here would name a rule this descent does not own (STYLE E2).
func (w componentWalk) walkSimpleType(t *SimpleType) error {
	if w.simpleType == nil {
		return nil
	}
	return w.simpleType(t)
}

// walkAttributeUse charges one Attribute Use and then enters the LOCAL
// declaration it owns; an <attribute ref> names a global declaration the phase's
// own root loop reaches instead.
func (w componentWalk) walkAttributeUse(u AttributeUse, loc xsderr.Loc, owner string) error {
	if w.attributeUse != nil {
		if err := w.attributeUse(u, loc, owner); err != nil {
			return err
		}
	}
	d, ok := u.AttributeDeclaration().(LocalAttributeDeclaration)
	if !ok {
		return nil
	}
	return w.walkAttributeDeclaration(d.Declaration)
}

// walkAttributeDeclaration descends an Attribute Declaration's one nesting slot,
// its {type definition}. The declaration retains its own Loc, so the slot is
// charged there rather than at whatever reached it. An attribute's type is always
// a simple type (§3.2.1), so the slot's inline arm reaches simpleType and its
// by-name arm is a kind-specific lookup that rejects a same-name non-type as
// dangling.
func (w componentWalk) walkAttributeDeclaration(a AttributeDeclaration) error {
	return w.walkTypeDefinition(a.TypeDefinition(), a.Loc(),
		"attribute declaration "+a.Name().String()+" {type definition}")
}

// walkElementDeclaration charges one Element Declaration and then enters its
// inline {type definition}. The slot's own verdict comes FIRST, before the
// declaration's, because a declaration whose type= names nothing has no type for
// any later clause to predicate over.
func (w componentWalk) walkElementDeclaration(e ElementDeclaration) error {
	ref := e.TypeDefinition()
	if w.typeDefinitionSlot != nil {
		if err := w.typeDefinitionSlot(ref, e.Loc(),
			"element declaration "+e.Name().String()+" {type definition}"); err != nil {
			return err
		}
	}
	if w.elementDeclaration != nil {
		if err := w.elementDeclaration(e); err != nil {
			return err
		}
	}
	return w.enterTypeDefinition(ref)
}

// walkParticle descends one particle's {term}, carrying loc — the enclosing
// component's position, since a Particle retains none of its own.
func (w componentWalk) walkParticle(p Particle, loc xsderr.Loc) error {
	t, ok := p.Term().(ResolvedTerm)
	if !ok {
		if w.termRef == nil {
			return nil
		}
		return w.termRef(p.Term(), loc)
	}
	switch inner := t.Term.(type) {
	case ElementDeclaration:
		return w.walkElementDeclaration(inner)
	case ModelGroup:
		return w.walkModelGroup(inner, loc)
	case Wildcard:
		return nil // a wildcard nests no component this tree reaches
	default:
		panic("xsd: componentWalk.walkParticle: non-exhaustive Term switch")
	}
}

// walkModelGroup descends every particle of a model group in document order,
// carrying loc: a ModelGroup retains no position of its own, so a top-level
// group is walked under its ModelGroupDefinition's Loc and an inline one under
// whatever component encloses it.
func (w componentWalk) walkModelGroup(g ModelGroup, loc xsderr.Loc) error {
	for _, p := range g.Particles() {
		if err := w.walkParticle(p, loc); err != nil {
			return err
		}
	}
	return nil
}

// attributeGroupOwner renders an Attribute Group Definition as the owner phrase
// of a rejection message charged against one of its {attribute uses}. Unlike a
// complex type's, its {name} is never absent (§3.6.1 requires it), so there is
// no anonymous case to describe.
func attributeGroupOwner(g AttributeGroupDefinition) string {
	return "attribute group definition " + g.Name().String()
}
