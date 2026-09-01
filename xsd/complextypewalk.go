package xsd

// This file supplies the ROOTS and the descent the READ-ONLY finalize passes
// quantified over "all complex type definitions" share, and charges nothing
// itself. §3.4.6's chapeau opens "All complex type definitions ... must satisfy
// the following constraints" and §3.8.6's opens "All model groups ... must
// satisfy the following constraints", neither with a carve-out for anonymity,
// nesting depth or the slot a type was reached through. An anonymous type enters
// no §3.17.2 {type definitions} slice — that property is scoped to the
// <complexType> children OF <schema> — so a pass ranging s.types alone reached
// none of them and produced no verdict at all for one (#438).
//
// The roots are s.types, s.elements and s.modelGroups — the three
// checkTypeTableSubstitutability (typetablesubstitutable.go) descends from, and
// THREE OF THE FOUR ownedTypeFold (ownedtypefold.go) descends from. Its fourth
// is s.modelGroupRedefinitions, the <redefine> ORIGINALS, "which are in no
// property and no index yet whose content models checkModelGroupRedefinitions
// charges clause 6.2.2 against" (ownedtypefold.go). This walk deliberately omits
// that root, which #438's Acceptance puts out of scope, and the residue the
// omission leaves is the GAP marker below. (checkComponentValueConstraints has a
// fourth of its own, s.attributeGroups, which holds attribute uses and a
// wildcard and so reaches no complex type definition at all.)
//
// s.modelGroups is a root in its OWN right rather than a convenience: §3.7.2
// gives <group name="..."> an (all|choice|sequence) child that can nest
// <element> children carrying inline <complexType>s, and where no <group ref>
// reachable from s.types names that Model Group Definition those types are
// reachable from nothing else.
//
// GAP(xsd): an anonymous complex type reachable ONLY through a <redefine>d
// <group>'s ORIGINAL definition (s.modelGroupRedefinitions[i].original,
// redefinition.go) gets no §3.4.6 or §3.8.6 verdict from any pass driven here.
// Its seating is the very one this walk otherwise covers — §3.7.2 lets that
// original nest <element> children whose LOCAL DECLARATIONS own inline
// <complexType>s — but the original itself sits in none of the three roots: it is
// in no property and no index, and s.modelGroups holds the REDEFINING definition
// under src-redefine clause 6.2, not the original. checkModelGroupRedefinitions
// (redefinition.go) charges src-redefine clause 6.2.2 language containment
// against it and nothing else, so no cos-nonambig, no cos-element-consistent, no
// ct-props-correct, no cos-ct-extends and no derivation-ok-restriction reaches
// those types. The gap is PRE-EXISTING — the s.types+s.modelGroups loops this
// walk replaces missed it too — and the direction is under-rejection, never a
// withheld property value: ownedTypeFold's fourth root folds them (#414). Adding
// the root is #584's, alongside the redefine original an owned {base type
// definition} seats.
//
// It is a second walk beside componentwalk.go's rather than a field on it
// because the two ENUMERATE DIFFERENTLY, not merely charge differently.
// componentWalk enters an owned {base type definition} as a component in its own
// right, which is the src-expredef clause 1.1 ORIGINAL a redefining complex type
// owns; that type is seated by a base slot rather than by a declaration's {type
// definition}, and giving it these three passes' verdicts is #584's change, not
// this one's. This descent therefore enters an owned base to reach the
// DECLARATIONS inside its content model and charges the base itself nothing —
// exactly the split ownedTypeFold.ownedBase/ownedTypeSlot already draws for the
// two mutating folds.
//
// It follows OWNERSHIP only, as both of those walks do and for the same reason:
// a by-name arm — a TypeDefinitionRef base, an <element ref>, a <group ref> —
// names a top-level component a root reaches in its own right, so following one
// would charge it once per referring site. That restriction is also what makes
// the walked structure a finite tree and lets it carry no visited set and no
// cycle guard (STYLE D4, PRINCIPLES 9): an owned component must pre-exist the
// slot holding it, so no ownership edge can close a cycle.
//
// A COMPONENT MAY BE CHARGED MORE THAN ONCE and every pass here is idempotent
// over one, so the repeats cost a verdict nothing. §3.3.2.1 dcl.elt.common case
// 2 can put a declaration's OWN {type definition} in its {type
// table}.{default type definition} too, and §3.4.2.3.2 builds an extension's
// {content type} particle AROUND its base's, so one anonymous type is reachable
// from more than one slot. Each of the three passes decides a property of the
// component alone at a Loc the component carries, so a second charge answers as
// the first did and reports at the same position.

// complexTypeWalk is the descent over every Complex Type Definition a Schema's
// roots hold or own, carrying the charges one read-only finalize pass makes at
// each. A phase fills in only what it CHARGES; which slots exist and which arm
// of each is followed is decided here once (STYLE T4).
//
// A NIL FIELD IS "THIS PHASE OWES THAT KIND NOTHING", never "skip that part of
// the tree" — componentwalk.go's convention, and for its reason: the descent is
// the same for every phase.
type complexTypeWalk struct {
	// complexType is charged on every Complex Type Definition the walk reaches: a
	// member of s.types, and every anonymous one a declaration owns through a
	// slot at any depth. A ComplexType retains its own Loc (doc.go), so it takes
	// neither a position nor an owner phrase.
	complexType func(c ComplexType) error
	// modelGroupDefinition is charged on each member of s.modelGroups. §3.7.1
	// makes a Model Group Definition's {model group} a Model Group component the
	// moment <group name="..."> is processed, so §3.8.6 binds it whether or not
	// any <group ref> points at it; a pass that constrains complex types alone
	// leaves the field nil.
	modelGroupDefinition func(d ModelGroupDefinition) error
}

// schema walks the three roots in document order (STYLE D1/D2 — no index map is
// ranged), so the first reported failure is deterministic. Each root's own
// charge is made before the descent enters anything it nests, so a schema wrong
// at two depths reports the outer failure.
func (w complexTypeWalk) schema(s *Schema) error {
	for _, t := range s.types {
		c, isComplex := t.(ComplexType)
		if !isComplex {
			continue // a simple type definition is no complex one and owns none
		}
		if err := w.walkComplexType(c); err != nil {
			return err
		}
	}
	for _, e := range s.elements {
		if err := w.walkElementDeclaration(e); err != nil {
			return err
		}
	}
	for _, d := range s.modelGroups {
		if w.modelGroupDefinition != nil {
			if err := w.modelGroupDefinition(d); err != nil {
				return err
			}
		}
		if err := w.walkModelGroup(d.ModelGroup()); err != nil {
			return err
		}
	}
	return nil
}

// walkComplexType charges c and then descends what c owns.
func (w complexTypeWalk) walkComplexType(c ComplexType) error {
	if w.complexType != nil {
		if err := w.complexType(c); err != nil {
			return err
		}
	}
	return w.walkOwned(c)
}

// walkOwned descends what c owns WITHOUT charging c: its owned {base type
// definition}, then its {content type}'s particle tree.
//
// The base is entered but never charged, for the reason this file's head gives —
// an owned base holds the <redefine> original, which is #584's to give a verdict
// to, while the local element declarations inside its content model own
// anonymous types that are this issue's. A by-name base names a top-level type
// s.types reaches; an owned SIMPLE base owns no complex type. EmptyContent nests
// nothing and SimpleContent's {simple type definition} is a simple type, whose
// own graph reaches no complex type definition.
func (w complexTypeWalk) walkOwned(c ComplexType) error {
	if base, owns := ownedComplexType(c.Base()); owns {
		if err := w.walkOwned(base); err != nil {
			return err
		}
	}
	switch ct := c.ContentType().(type) {
	case EmptyContent:
		return nil
	case SimpleContent:
		return nil
	case ElementContent:
		return w.walkParticle(ct.Particle)
	default:
		panic("xsd: complexTypeWalk.walkOwned: non-exhaustive ContentType switch")
	}
}

// walkParticle descends one particle's {term}. A by-name {term} is left alone:
// it names a top-level element declaration or model group definition a root
// reaches.
func (w complexTypeWalk) walkParticle(p Particle) error {
	t, resolved := p.Term().(ResolvedTerm)
	if !resolved {
		return nil
	}
	switch inner := t.Term.(type) {
	case ElementDeclaration:
		return w.walkElementDeclaration(inner)
	case ModelGroup:
		return w.walkModelGroup(inner)
	case Wildcard:
		return nil // a wildcard owns no component this tree reaches
	default:
		panic("xsd: complexTypeWalk.walkParticle: non-exhaustive Term switch")
	}
}

// walkModelGroup descends every particle of a model group in document order. It
// reads the unexported {particles} rather than Particles(), whose defensive copy
// would be allocated only to be discarded — this walk neither retains nor
// mutates the slice (same reason checkTypeTableAlternatives reads
// disallowedSubstitutions directly).
func (w complexTypeWalk) walkModelGroup(g ModelGroup) error {
	for _, p := range g.particles {
		if err := w.walkParticle(p); err != nil {
			return err
		}
	}
	return nil
}

// walkElementDeclaration enters every slot of an element declaration that can
// own an anonymous complex type, charging and descending each type it finds.
//
// The slots come from ownedTypeSlots (elementdeclaration.go), the one producer
// of that enumeration, so this walk cannot drift out of step with the {context}
// check and the constructor's symmetry rejection that read it too. The two
// mutating folds hand-roll the same three slots because they WRITE each one back
// with the folded type, which a flattened list of refs cannot express; a
// read-only walk has no such need.
//
// An element declaration nests nothing else this tree reaches: its {identity
// constraint definitions} hold selectors and fields, and its {substitution group
// affiliations} name top-level declarations s.elements holds.
func (w complexTypeWalk) walkElementDeclaration(e ElementDeclaration) error {
	var owned *TypeTable
	if table, hasTable := e.TypeTable(); hasTable {
		owned = &table
	}
	for _, slot := range ownedTypeSlots(e.TypeDefinition(), owned) {
		c, owns := ownedComplexType(slot.ref)
		if !owns {
			continue
		}
		if err := w.walkComplexType(c); err != nil {
			return err
		}
	}
	return nil
}
