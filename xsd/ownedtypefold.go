package xsd

// This file supplies the ROOTS the two finalize folds walk beyond the Schema's
// {type definitions}, and applies neither mapping rule itself. §3.4.2.4 opens
// "This mapping rule is the same for all complex type definitions" and §3.4.2.5
// says the same of its own, so a complex type reached only through a
// DECLARATION is owed clause 3 and clause 2 exactly as a named one is. An
// anonymous type enters no §3.17.1 symbol table and no {type definitions}
// slice, so the only route to one is the slot that owns it, and a fold walking
// s.types alone reached none (#414).
//
// The two folds stay two folds: what is shared here is the set of roots each
// outer walk starts from, never the mapping logic, whose case lists and
// combination operators differ (#265).
//
// The descent REWRITES rather than merely visits, which is what makes it a
// second walk and not a field on componentwalk.go's read-only one. A
// ComplexType is a value, so a folded component is observable only if every
// slot on the path from a root down to it is re-seated with the copy that holds
// it — the same argument baseAttributeUses makes for an owned inline {base type
// definition}, one level up.

// noTypePosition is the position argument for a complex type that HAS no index
// in s.types — every anonymous one. It matches no index, so the ·xs:anyType·
// self-edge exclusion (j == i) never fires for such a type. That is right
// rather than merely harmless: ·xs:anyType· is named, so no anonymous type is
// it, and none can reach it other than as a genuine base.
const noTypePosition = -1

// ownedTypeFold is the rewriting descent both finalize folds run over the
// components a Schema's roots OWN, carrying the one per-type fold to apply at
// every declaration-owned anonymous complex type.
//
// It follows OWNERSHIP only, exactly as componentwalk.go does and for the same
// reason: a by-name arm — a TypeDefinitionRef base, an <element ref>, a <group
// ref> — names a top-level component schema reaches in its own right, so
// following one would fold it once per referring site. That restriction is also
// what makes the walked structure a finite tree and lets it carry no visited set
// (STYLE D4, PRINCIPLES 9): an owned component must pre-exist the slot holding
// it, so no ownership edge can close a cycle.
type ownedTypeFold struct {
	// fold applies ONE mapping rule to one declaration-owned complex type and
	// returns the component with that property's folded value stored on it.
	// It is called once per owning SLOT, never once per component: two slots
	// holding structurally equal copies are two components as far as this
	// package can tell, and each gets its own fold with the same result.
	fold func(c ComplexType) (ComplexType, error)
}

// schema runs the descent from every root of the assembled Schema that can own
// an anonymous complex type, re-seating each root and the by-name index derived
// from it so a consumer reaching a component either way sees the folded value
// (storeFoldedAttributeUses makes the same argument for a named type).
//
// The roots are {type definitions}, {element declarations} and {model group
// definitions} — the three checkComponentValueConstraints and
// checkTypeTableSubstitutability descend from — plus the <redefine> ORIGINALS,
// which are in no property and no index yet whose content models
// checkModelGroupRedefinitions charges clause 6.2.2 against, reading the
// {attribute uses} of any type declared inside them.
//
// {attribute declarations} and {attribute group definitions} are absent because
// §3.2.1 types an attribute's {type definition} as a SIMPLE type definition, and
// an attribute group definition holds attribute uses and a wildcard and nothing
// else.
//
// GAP(xsd): this package does not enforce that §3.2.1 typing, so the three
// attribute-side slots stay unfolded. checkTypeDefinitionOrRef
// (typedefinition.go) decides the ARM of an attributeTypeSlot, never the
// VARIETY, so NewAttributeDeclaration accepts an InlineTypeDefinition wrapping a
// ComplexType and Finalize accepts the Schema. A caller assembling one through
// [SchemaBuilder] can therefore seat an anonymous complex type at a top-level
// attribute declaration's {type definition}, at a LocalAttributeDeclaration's
// inside a complex type's {attribute uses}, or at one inside an attribute group
// definition — all three reach that one constructor — and no root here walks any
// of them, so such a type reports its own {attribute uses} and its own
// <anyAttribute> alone. The PARSER cannot produce the shape: no production puts
// a <complexType> under an <attribute>. Nothing owns the retirement — enforcing
// the variety and widening these roots are two changes, and neither is #414's.
//
// Direction (STYLE P3a): the withheld members are UNDER-reported and no reader
// charges on their absence. In this package the one reader is
// componentWalk.walkComplexType, which ranges AttributeUses() to DESCEND, for
// resolveReferences and checkSimpleTypeDerivations (resolve.go),
// checkTypeTableSubstitutability (typetablesubstitutable.go) and
// checkComponentValueConstraints (valueconstraintvalid.go); every member clause
// 3 would have added comes from the {base type definition}, which that same walk
// enters one slot above, so no descent is lost. Outside it no reader obtains the
// component at all: every consumer of an attribute declaration's {type
// definition} narrows through [Schema.ResolvedSimpleType], which declines a
// complex type outright — attributeUseType (defaultbinding.go),
// checkAttributeDeclarationValueConstraint and checkAttributeUseValueConstraint
// (valueconstraintvalid.go) here, and validate's walk.matchedAttribute,
// walk.defaultedAttribute, walk.idDefaultedAttributes, walk.attributeType and
// walk.topLevelAttributeType (cvcattribute.go, cvcid.go).
//
// Roots are walked in document order and no index map is ranged (STYLE D2).
func (o ownedTypeFold) schema(s *Schema) error {
	for i, t := range s.types {
		c, isComplex := t.(ComplexType)
		if !isComplex {
			continue // a simple type definition owns no complex one
		}
		rewritten, err := o.complexType(c)
		if err != nil {
			return err
		}
		s.types[i] = rewritten
		if rewritten.Name() != (QName{}) {
			s.typeIndex[rewritten.Name()] = rewritten
		}
	}
	for i, e := range s.elements {
		rewritten, err := o.elementDeclaration(e)
		if err != nil {
			return err
		}
		s.elements[i] = rewritten
		s.elementIndex[rewritten.Name()] = rewritten
	}
	for i, d := range s.modelGroups {
		g, err := o.modelGroup(d.ModelGroup())
		if err != nil {
			return err
		}
		d.modelGroup = g
		s.modelGroups[i] = d
		s.modelGroupIndex[d.Name()] = d
	}
	for i, r := range s.modelGroupRedefinitions {
		g, err := o.modelGroup(r.original.ModelGroup())
		if err != nil {
			return err
		}
		r.original.modelGroup = g
		s.modelGroupRedefinitions[i] = r
	}
	return nil
}

// complexType descends what c owns and returns c re-seated with the result. It
// does NOT fold c: a type reached here is either a root of s.types, which the
// fold's own position walk has already folded, or an owned inline {base type
// definition}, which that walk folded through baseAttributeUses. Folding
// happens at ownedTypeSlot alone, which is the one place a component arrives
// unfolded.
func (o ownedTypeFold) complexType(c ComplexType) (ComplexType, error) {
	base, err := o.ownedBase(c.base)
	if err != nil {
		return ComplexType{}, err
	}
	c.base = base
	content, err := o.contentType(c.contentType)
	if err != nil {
		return ComplexType{}, err
	}
	c.contentType = content
	return c, nil
}

// ownedBase descends the anonymous complex type a {base type definition} slot
// OWNS — the src-expredef clause 1.1 original a redefining complex type holds —
// without folding it, and returns the slot to store back. A by-name base names a
// top-level type the root loop reaches; an owned SIMPLE base owns nothing this
// tree reaches.
func (o ownedTypeFold) ownedBase(ref TypeDefinitionOrRef) (TypeDefinitionOrRef, error) {
	c, owns := ownedComplexType(ref)
	if !owns {
		return ref, nil
	}
	rewritten, err := o.complexType(c)
	if err != nil {
		return nil, err
	}
	return InlineTypeDefinition{Definition: rewritten}, nil
}

// contentType descends the one {content type} variety that nests a particle
// tree. EmptyContent nests nothing and SimpleContent's {simple type definition}
// is a simple type, whose own graph reaches no complex type definition.
func (o ownedTypeFold) contentType(ct ContentType) (ContentType, error) {
	switch t := ct.(type) {
	case EmptyContent:
		return t, nil
	case SimpleContent:
		return t, nil
	case ElementContent:
		p, err := o.particle(t.Particle)
		if err != nil {
			return nil, err
		}
		t.Particle = p
		return t, nil
	default:
		panic("xsd: ownedTypeFold.contentType: non-exhaustive ContentType switch")
	}
}

// particle descends one particle's {term} and returns the particle re-seated
// with the result. A by-name {term} is left alone: it names a top-level element
// declaration or model group definition the root loop reaches.
func (o ownedTypeFold) particle(p Particle) (Particle, error) {
	t, resolved := p.term.(ResolvedTerm)
	if !resolved {
		return p, nil
	}
	switch inner := t.Term.(type) {
	case ElementDeclaration:
		e, err := o.elementDeclaration(inner)
		if err != nil {
			return Particle{}, err
		}
		p.term = ResolvedTerm{Term: e}
		return p, nil
	case ModelGroup:
		g, err := o.modelGroup(inner)
		if err != nil {
			return Particle{}, err
		}
		p.term = ResolvedTerm{Term: g}
		return p, nil
	case Wildcard:
		return p, nil // a wildcard owns no component this tree reaches
	default:
		panic("xsd: ownedTypeFold.particle: non-exhaustive Term switch")
	}
}

// modelGroup descends every particle of a model group in document order,
// returning the group re-seated with a FRESH {particles} slice.
//
// The copy keeps Finalize from rewriting a slice a CALLER still holds.
// NewModelGroup copies its input, so the array walked here is the one the
// ModelGroup value the caller built points at; writing a folded particle into it
// would fold that unfinalized component too, against what AttributeUses
// documents ("only what that caller passed in"). It is not a defence against
// double folding: §3.4.2.3.2 builds an extension's {content type} particle
// around its BASE's, so one backing array is reachable from more than one root,
// but both mapping rules are idempotent — clause 3.2.1 excludes an inherited use
// whose name an own use already carries, cos-aw-union re-unions to the same
// constraint — so folding a component twice yields what folding it once does.
// TestOwnedFoldLeavesTheCallersSlicesAlone pins what the copy actually decides.
func (o ownedTypeFold) modelGroup(g ModelGroup) (ModelGroup, error) {
	particles := append([]Particle(nil), g.particles...)
	for i, p := range particles {
		rewritten, err := o.particle(p)
		if err != nil {
			return ModelGroup{}, err
		}
		particles[i] = rewritten
	}
	g.particles = particles
	return g, nil
}

// elementDeclaration folds and descends every slot of an element declaration
// that can own an anonymous complex type, and returns the declaration re-seated
// with the results.
//
// The slots are ownedTypeSlots' (elementdeclaration.go), in its order: the
// declaration's own {type definition}, then each {type table}.{alternatives}
// member's, then the {type table}.{default type definition}'s. The two
// enumerations MUST agree — this one writes each slot back where that one only
// reads it for a rejection message, which is why the read-only producer cannot
// serve both.
func (o ownedTypeFold) elementDeclaration(e ElementDeclaration) (ElementDeclaration, error) {
	ref, err := o.ownedTypeSlot(e.typeDefinition)
	if err != nil {
		return ElementDeclaration{}, err
	}
	e.typeDefinition = ref
	if !e.hasTypeTable {
		return e, nil
	}
	tt, err := o.typeTable(e.typeTable)
	if err != nil {
		return ElementDeclaration{}, err
	}
	e.typeTable = tt
	return e, nil
}

// typeTable folds and descends each {alternatives} member's owned type in
// document order and then the {default type definition}'s, returning the table
// re-seated with a FRESH {alternatives} slice for the reason modelGroup copies
// its particles.
//
// §3.3.2.1 dcl.elt.common case 2 can make the {default type definition} hold the
// declaration's OWN {type definition}, in which case one component is folded at
// two slots. Both answer alike whichever way that lands, for the idempotence
// modelGroup's comment states; ownedTypeSlots enumerates that slot
// unconditionally for the same reason.
func (o ownedTypeFold) typeTable(tt TypeTable) (TypeTable, error) {
	alternatives := append([]TypeAlternative(nil), tt.alternatives...)
	for i, alt := range alternatives {
		rewritten, err := o.typeAlternative(alt)
		if err != nil {
			return TypeTable{}, err
		}
		alternatives[i] = rewritten
	}
	tt.alternatives = alternatives
	dflt, err := o.typeAlternative(tt.defaultTypeDefinition)
	if err != nil {
		return TypeTable{}, err
	}
	tt.defaultTypeDefinition = dflt
	return tt, nil
}

// typeAlternative folds and descends the anonymous complex type a Type
// Alternative's {type definition} owns — §3.12.2 declare-ta's inline arm, the
// <alternative> element's own <complexType> child.
func (o ownedTypeFold) typeAlternative(a TypeAlternative) (TypeAlternative, error) {
	ref, err := o.ownedTypeSlot(a.typeDefinition)
	if err != nil {
		return TypeAlternative{}, err
	}
	a.typeDefinition = ref
	return a, nil
}

// ownedTypeSlot is where a component arrives UNFOLDED: it applies the fold to
// the anonymous complex type a declaration's slot owns, then descends what that
// type owns in turn, and returns the slot to store back. Every other arm of the
// sum — a by-name reference, a substitution-group-head reference, an absent
// slot, an owned simple type — is returned untouched.
//
// The fold runs BEFORE the descent so a nested declaration inside this type's
// content model is reached through the folded component, which is the order the
// rest of the pass reads in (base before derived, STYLE D4). The two are
// independent all the same: neither mapping rule reads a type declared inside
// the content model it is folding.
func (o ownedTypeFold) ownedTypeSlot(ref TypeDefinitionOrRef) (TypeDefinitionOrRef, error) {
	c, owns := ownedComplexType(ref)
	if !owns {
		return ref, nil
	}
	folded, err := o.fold(c)
	if err != nil {
		return nil, err
	}
	rewritten, err := o.complexType(folded)
	if err != nil {
		return nil, err
	}
	return InlineTypeDefinition{Definition: rewritten}, nil
}
