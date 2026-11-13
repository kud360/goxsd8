package goshape

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// SubstitutionSum reports whether a field over the element declaration decl
// holds a sum, and returns its members: the declarations of decl's
// ·substitution group· (§3.3.6.4 key-eq, [xsd.Schema.SubstitutionGroup]) with
// {abstract} false (§3.3.4.3 cvc-elt clause 2), in group order. sum is true
// exactly when decl is top-level and members are not decl alone, so an
// abstract decl whose group holds no other declaration with {abstract} false
// answers a sum with no members, and a decl with {abstract} false whose other
// declarations are all abstract answers none. The group is the whole of
// [xsd.Schema.SubstitutionGroup]'s, whose doc states when decl is alone in it
// (cos-equiv-derived-ok-rec clauses 1, 2.1 and 2.3) and that a local
// declaration heads none (cvc-accept clause 2.3.2); codegen/doc.go
// "# Substitution groups" states the branches of the members it holds through
// an abstract declaration (cos-equiv-derived-ok-rec clause 2.2).
func SubstitutionSum(schema *xsd.Schema, decl xsd.ElementDeclaration) (members []xsd.ElementDeclaration, sum bool) {
	if decl.ScopeVariety() != xsd.ScopeGlobal {
		return nil, false
	}
	members = slices.DeleteFunc(schema.SubstitutionGroup(decl.Name()), xsd.ElementDeclaration.Abstract)
	if len(members) == 1 && members[0].Name() == decl.Name() {
		return nil, false
	}
	return members, true
}

// GoverningType is one of an element declaration's governing types
// (GoverningTypes): its type, resolved, and the Slot it is first held at. Only
// GoverningTypes makes one.
type GoverningType struct {
	typ  xsd.TypeDefinition
	slot Slot
}

// Type is g's type, resolved.
func (g GoverningType) Type() xsd.TypeDefinition { return g.typ }

// Slot is the slot of the element declaration g's type is first held at, which
// holds that type itself or, when it is one of the types GoverningTypes' members
// returns, the complex type with {abstract} true it stands for.
func (g GoverningType) Slot() Slot { return g.slot }

// Local is the local name the branch of g is named from (codegen/doc.go
// "# Naming"): its type's own where the type is named, and otherwise its slot's
// (Slot.Anonymous).
func (g GoverningType) Local() string {
	if g.typ.Name() != (xsd.QName{}) {
		return g.typ.Name().Local
	}
	return g.slot.Anonymous()
}

// Slot is the slot of an element declaration a governing type comes from
// (GoverningTypes): a {type table} alternative (AlternativeSlot), its {default
// type definition} (DefaultSlot), or, for a declaration with no {type table},
// its {type definition} (DeclaredSlot). §3.3.1 closes the set, so it is a
// sealed sum (STYLE T2's exception).
type Slot interface {
	// Name is how a decline or a header names the slot.
	Name() string
	// Anonymous is the local name the branch of an anonymous type held at the
	// slot is named from (codegen/doc.go "# Naming").
	Anonymous() string
	slot()
}

// AlternativeSlot is the {type table} alternative at this 1-based position in
// {alternatives}.
type AlternativeSlot int

// Name is "{type table} alternative N".
func (s AlternativeSlot) Name() string { return "{type table} alternative " + strconv.Itoa(int(s)) }

// Anonymous is "alternative" followed by s's position.
func (s AlternativeSlot) Anonymous() string { return "alternative" + strconv.Itoa(int(s)) }

func (AlternativeSlot) slot() {}

// DefaultSlot is a {type table}'s {default type definition}.
type DefaultSlot struct{}

// Name is "{default type definition}".
func (DefaultSlot) Name() string { return "{default type definition}" }

// Anonymous is "default".
func (DefaultSlot) Anonymous() string { return "default" }

func (DefaultSlot) slot() {}

// DeclaredSlot is the {type definition} of a declaration with no {type table}.
// No branch is named from its anonymous name: an anonymous {type definition} is
// a local <complexType> or <simpleType>, never abstract (the schema for
// schemas' localComplexType prohibits the abstract attribute), so it is its
// declaration's one governing type, which is no sum.
type DeclaredSlot struct{}

// Name is "{type definition}".
func (DeclaredSlot) Name() string { return "{type definition}" }

// Anonymous is "typeDefinition".
func (DeclaredSlot) Anonymous() string { return "typeDefinition" }

func (DeclaredSlot) slot() {}

// errorTypeName is the expanded name of xs:error (§3.16.7.3).
var errorTypeName = xsd.QName{Space: xsd.XMLSchemaNS, Local: "error"}

// GoverningTypes returns the governing types codegen/doc.go "# Element
// declarations" gives the element declaration decl, in its order and by its
// identity: the {type definition} of each of its {type table}'s alternatives
// in order, then that of its {default type definition} (§3.3.4.1
// key-selected-type, key-cta-select), or, with no {type table}, its {type
// definition} alone, each resolved through [xsd.Schema.ResolvedType]. A {type
// table} slot holding xs:error, matched by name, gives nothing. A slot holding
// a complex type with {abstract} true gives the types members returns for it
// in its place, its error returned as is. Each type is kept at its first slot
// only, two types being one exactly when both are named and the names are
// equal, so two anonymous types are never one. errorSlot reports whether a
// slot held xs:error. A slot that does not resolve is a plain error naming
// the slot, decl and decl's Loc, never a decline: Finalize resolves every
// {type definition} slot (src-resolve).
func GoverningTypes(schema *xsd.Schema, decl xsd.ElementDeclaration, members func(xsd.ComplexType) ([]xsd.TypeDefinition, error)) (types []GoverningType, errorSlot bool, err error) {
	type slotRef struct {
		slot Slot
		ref  xsd.TypeDefinitionOrRef
	}
	slots := []slotRef{{DeclaredSlot{}, decl.TypeDefinition()}}
	table, hasTable := decl.TypeTable()
	if hasTable {
		slots = nil
		for i, alt := range table.Alternatives() {
			slots = append(slots, slotRef{AlternativeSlot(i + 1), alt.TypeDefinition()})
		}
		slots = append(slots, slotRef{DefaultSlot{}, table.DefaultTypeDefinition().TypeDefinition()})
	}
	for _, s := range slots {
		td, ok := schema.ResolvedType(s.ref)
		if !ok {
			return nil, false, fmt.Errorf("%s of element %s at %s: its {type definition} does not resolve", s.slot.Name(), decl.Name(), decl.Loc())
		}
		if hasTable && td.Name() == errorTypeName {
			errorSlot = true
			continue
		}
		given := []xsd.TypeDefinition{td}
		if ct, isComplex := td.(xsd.ComplexType); isComplex && ct.Abstract() {
			given, err = members(ct)
			if err != nil {
				return nil, false, err
			}
		}
		for _, td := range given {
			if td.Name() != (xsd.QName{}) && slices.ContainsFunc(types, func(seen GoverningType) bool { return seen.typ.Name() == td.Name() }) {
				continue
			}
			types = append(types, GoverningType{typ: td, slot: s.slot})
		}
	}
	return types, errorSlot, nil
}

// BasicMembers returns the basic members of the union simple type st
// (Datatypes §2.4.1.3 dt-basicmember) in codegen's basic-member order
// (codegen/doc.go "# Simple types"), reading each union's {member type
// definitions} (§3.16.1) through [xsd.SimpleType.Members], which takes a
// restriction's from its base (§3.16.2.1), and keeping each member at its
// first position by identity. No union reaches itself through its members
// (cos-st-restricts clause 3.3 no-self-membership, which finalize checks), so
// the walk carries no visited set.
func BasicMembers(r xsd.TypeResolver, st *xsd.SimpleType) ([]*xsd.SimpleType, error) {
	members, err := st.Members(r)
	if err != nil {
		return nil, err
	}
	var basic []*xsd.SimpleType
	for _, m := range members {
		variety, err := m.Variety(r)
		if err != nil {
			return nil, err
		}
		reached := []*xsd.SimpleType{m}
		// {variety} is a closed sum, so the assertion is STYLE T2's sealed-sum
		// exception.
		if _, isUnion := variety.(xsd.Union); isUnion {
			reached, err = BasicMembers(r, m)
			if err != nil {
				return nil, err
			}
		}
		for _, b := range reached {
			if !slices.Contains(basic, b) {
				basic = append(basic, b)
			}
		}
	}
	return basic, nil
}

// Splices reports whether a model group particle of group occurring as occurs,
// inline or a <group ref> to group's definition (§3.7.2
// declare-namedModelGroup), the content {particle} or nested, is spliced,
// giving in its place the fields group's own particles give, as
// codegen/doc.go "# Particles" states. Each caller follows a <group ref> to
// group itself. Whether one occurrence of group can match nothing is
// TermEmptiable's answer (§3.8.6.5 cos-seq-range). The error is
// TermEmptiable's.
func Splices(schema *xsd.Schema, group xsd.ModelGroup, occurs xsd.Occurs) (bool, error) {
	if group.Compositor() == xsd.CompositorChoice {
		return false, nil
	}
	upper, bounded := occurs.Max()
	if !bounded || upper != 1 {
		return false, nil
	}
	if occurs.Min() == 1 {
		return true, nil
	}
	return TermEmptiable(schema, xsd.ResolvedTerm{Term: group})
}

// Flattens reports whether a model group particle of group occurring as
// occurs, inline or a <group ref> to group's definition (§3.7.2
// declare-namedModelGroup), the content {particle} or nested, meets one of
// the three gates codegen/doc.go "# Particles" states for a flattened field,
// which that section owns. Each caller follows a <group ref> to group itself,
// as for Splices; Flattens follows the <group ref>s inside group itself. It
// does not decide which particle on the walk inward is flattened: a caller
// asks it of a particle only outside a flattened particle's reach. The error
// is TermEmptiable's or ModelGroupTerm's, unwrapped.
func Flattens(schema *xsd.Schema, group xsd.ModelGroup, occurs xsd.Occurs) (bool, error) {
	if group.Compositor() == xsd.CompositorChoice {
		for _, p := range group.Particles() {
			emptiable, err := TermEmptiable(schema, p.Term())
			if err != nil {
				return false, err
			}
			if emptiable {
				return true, nil
			}
		}
	}
	if upper, bounded := occurs.Max(); bounded && upper <= 1 {
		return false, nil
	}
	if group.Compositor() == xsd.CompositorSequence {
		emptiable, err := TermEmptiable(schema, xsd.ResolvedTerm{Term: group})
		if err != nil {
			return false, err
		}
		if emptiable {
			return true, nil
		}
	}
	return repeatingEdge(schema, group)
}

// Reach walks the reach of group, the model group of a particle Flattens
// answers true for, as codegen/doc.go "# Particles" states it: depth-first
// through group's {particles} in order, calling visit for each basic particle
// (Structures §2.2.3.2 key-basic-particle), an element particle, local or an
// <element ref>, or a wildcard particle, whatever its {max occurs}, which
// visit reads, and for a model group particle with {max occurs} 0 in place of
// its group's reach, and replacing each other model group particle, inline or
// a <group ref>, by its own model group's. It never enters an element's type.
// position is the particle's indices into each enclosing model group's
// {particles} from group down, outermost first, group itself not counted, as
// [xsd.Matcher.NextParticle] reports a position; each call passes a fresh
// slice the visitor may keep. The first error visit returns ends the walk and
// is Reach's. An all group particle inside group is an error, raised before
// visit sees that particle: cos-all-limited (§3.8.6.2), checked at Finalize,
// admits one only as a content {particle} with {max occurs} 1 or among an all
// group's {particles}, and group is neither an all group nor inside one. A
// <group ref> that does not resolve is an error too: Finalize resolves every
// one. Both are unwrapped. The <group ref> graph is acyclic after Finalize,
// so the walk carries no visited set.
func Reach(schema *xsd.Schema, group xsd.ModelGroup, visit func(p xsd.Particle, position []int) error) error {
	return reach(schema, group, nil, visit)
}

// reach is Reach's walk of group, whose own position is at, nil for the
// walk's root.
func reach(schema *xsd.Schema, group xsd.ModelGroup, at []int, visit func(p xsd.Particle, position []int) error) error {
	for i, p := range group.Particles() {
		inner, isGroup, err := ModelGroupTerm(schema, p.Term())
		if err != nil {
			return err
		}
		if isGroup && inner.Compositor() == xsd.CompositorAll {
			return errors.New("an all model group particle inside a flattened model group reached Reach")
		}
		position := append(slices.Clone(at), i)
		if upper, bounded := p.Occurs().Max(); !isGroup || bounded && upper == 0 {
			if err := visit(p, position); err != nil {
				return err
			}
			continue
		}
		if err := reach(schema, inner, position, visit); err != nil {
			return err
		}
	}
	return nil
}

// repeatingEdge reports whether group, the term of a model group particle
// with {max occurs} above 1 or unbounded, holds a particle that repeats
// (repeats) at both its edges, as codegen/doc.go "# Particles" states
// (othersEmptiable reads a sequence's edges). It reads model group terms
// through ModelGroupTerm, never enters an element's type, and stops at the
// first repeating edge particle; the <group ref> graph is acyclic after
// Finalize, so the walk ends. No all group is reached: cos-all-limited
// (§3.8.6.2 clauses 1.2 and 1.3) admits a particle whose term is one only as
// a content particle with {max occurs} 1 or as an all group's member, neither
// of which is group's particle or inside it.
func repeatingEdge(schema *xsd.Schema, group xsd.ModelGroup) (bool, error) {
	particles := group.Particles()
	for i, p := range particles {
		if group.Compositor() != xsd.CompositorChoice && !othersEmptiable(schema, particles, i) {
			continue
		}
		if repeats(p.Occurs()) {
			return true, nil
		}
		inner, isGroup, err := ModelGroupTerm(schema, p.Term())
		if err != nil {
			return false, err
		}
		if !isGroup {
			continue
		}
		edge, err := repeatingEdge(schema, inner)
		if err != nil {
			return false, err
		}
		if edge {
			return true, nil
		}
	}
	return false, nil
}

// othersEmptiable reports whether every particle of particles but the one at
// index i is emptiable ([xsd.Schema.ParticleEmptiable]).
func othersEmptiable(schema *xsd.Schema, particles []xsd.Particle, i int) bool {
	for j, p := range particles {
		if j != i && !schema.ParticleEmptiable(p) {
			return false
		}
	}
	return true
}

// repeats reports whether a particle of occurs repeats, as codegen/doc.go
// "# Particles" defines the word.
func repeats(occurs xsd.Occurs) bool {
	upper, bounded := occurs.Max()
	return !bounded || upper > 1 && upper != occurs.Min()
}

// TermEmptiable reports whether one occurrence of a particle's {term} can
// match nothing: [xsd.Schema.ParticleEmptiable] (§3.9.6.3
// cos-group-emptiable) over a particle of term occurring exactly once
// (ExactlyOnce), so that its clause 2 alone decides, never read off a
// particle's own range. It is false for an element declaration or a wildcard
// and true for a model group with no {particles}. The error is
// xsd.NewParticle's, unwrapped; it does not reject a resolved term.
func TermEmptiable(schema *xsd.Schema, term xsd.TermOrRef) (bool, error) {
	p, err := xsd.NewParticle(xsderr.Loc{}, ExactlyOnce(), term)
	if err != nil {
		return false, err
	}
	return schema.ParticleEmptiable(p), nil
}

// Nillable reports whether the element declaration decl is nillable as
// codegen/doc.go "# Element declarations" states it: {nillable} true (§3.3.1
// ed-nillable) and no {value constraint} of {variety} fixed, which makes
// every nilled instance invalid (§3.3.4.3 cvc-elt clause 3.2.3.2), so that a
// decl with one emits as one with {nillable} false. That section states which
// fields over a nillable decl hold Nillable. It is the one test of it: every
// codegen site that wraps a type in Nillable, or passes a field over decl as
// no struct value, reads it, and so does codec where it binds such a field.
func Nillable(decl xsd.ElementDeclaration) bool {
	if !decl.Nillable() {
		return false
	}
	vc, ok := decl.ValueConstraint()
	return !ok || vc.Kind() != xsd.ValueFixed
}

// xsiNil is the expanded name of the built-in attribute declaration xsi:nil
// (§3.2.7.2).
var xsiNil = xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}

// FieldAttributeUses returns t's {attribute uses} but a use of xsi:nil, the
// uses codegen gives a field each (codegen/doc.go "# Attributes"): a nillable
// element's nilled-ness is Nillable's arm (cvc-elt clause 3.2.3), and a
// declaration that is not nillable admits no xsi:nil (cvc-elt clause 3.1). It
// matches the use's {attribute declaration} by expanded name, which stands
// for the built-in's identity because no-xsi (§3.2.6.4) keeps a schema
// document out of the xsi namespace.
//
// GAP(xsd): a programmatically built schema is not kept out (#1462), so a
// declaration it builds named {xsi}nil matches too: codegen then emits no
// field for that use and HasAttributeFields does not count it, so the emitted
// type omits that attribute and nothing declines.
func FieldAttributeUses(t xsd.ComplexType) []xsd.AttributeUse {
	return slices.DeleteFunc(t.AttributeUses(), func(u xsd.AttributeUse) bool { return u.DeclarationName() == xsiNil })
}

// HasAttributeFields reports whether the complex type t has an attribute use
// FieldAttributeUses keeps or an {attribute wildcard}, read from the
// finalized component, so inherited ones count: exactly when the struct
// codegen emits for t holds attribute fields (codegen/doc.go "# Attributes"),
// and so when codegen never emits the choice t's content holds as t itself
// (codegen/doc.go "# Complex types").
func HasAttributeFields(t xsd.ComplexType) bool {
	_, hasWildcard := t.AttributeWildcard()
	return hasWildcard || len(FieldAttributeUses(t)) > 0
}

// Mixed reports whether the complex type t's {content type} has {variety}
// mixed (§3.4.1): element content whose effective mixed value (§3.4.2.3.3
// dcl.ctd.ctcc.common clause 1) is true, read from t's resolved {content type}
// alone, never its base's, so that an extension's (clause 4.2) and a
// restriction's are each its own.
func Mixed(t xsd.ComplexType) bool {
	content, isElement := t.ContentType().(xsd.ElementContent)
	return isElement && content.Mixed
}

// Shape is the occurrence in which a field holds its value V: V itself, *V or
// []V. The zero value is invalid (STYLE T1).
type Shape uint8

// The Shapes.
const (
	Bare    Shape = iota + 1 // V
	Pointer                  // *V
	Slice                    // []V
)

// ParticleShape returns the Shape codegen/doc.go "# Particles" gives the
// field of a particle occurring as occurs, its V a sealed interface exactly
// when sealed. ok is false exactly when {max occurs} is 0, for which codegen
// emits no field.
func ParticleShape(occurs xsd.Occurs, sealed bool) (s Shape, ok bool) {
	upper, bounded := occurs.Max()
	switch {
	case !bounded || upper > 1:
		return Slice, true
	case upper == 0:
		return 0, false
	case occurs.Min() == 0 && !sealed:
		return Pointer, true
	}
	return Bare, true
}

// SuppliedValueConstraint returns the effective value constraint of the
// attribute use u (§3.5.4 key-evc, [xsd.Schema.EffectiveValueConstraint])
// where it supplies the value the field of an absent attribute holds, as
// codegen/doc.go "# Attributes" states, and ok false elsewhere. It reads u's
// {attribute declaration} as a built-in xsi: one, whose constraint supplies
// none, exactly when it is in the xsi namespace (§3.2.6.4 no-xsi), and does
// not read u's {required}.
//
// GAP(xsd): a programmatically built schema is not kept out of the xsi
// namespace (#1462), so for an optional use of a built {xsi}-named
// declaration with an effective value constraint ok is false, where
// "# Attributes" gives a bare field holding that value: codegen gives the
// field the Pointer shape, a union's interface aside (AttributeShape), and
// codec supplies the field no value when the attribute is absent.
func SuppliedValueConstraint(schema *xsd.Schema, u xsd.AttributeUse) (vc xsd.ValueConstraint, ok bool) {
	vc, ok = schema.EffectiveValueConstraint(u)
	if !ok || u.DeclarationName().Space == xsd.XMLSchemaInstanceNS {
		return xsd.ValueConstraint{}, false
	}
	return vc, true
}

// AttributeShape returns the Shape codegen/doc.go "# Attributes" gives the
// field of the attribute use u, its V a sealed interface exactly when sealed,
// reading whether an absent attribute's field holds a value from
// SuppliedValueConstraint.
func AttributeShape(schema *xsd.Schema, u xsd.AttributeUse, sealed bool) Shape {
	_, supplied := SuppliedValueConstraint(schema, u)
	if u.Required() || supplied || sealed {
		return Bare
	}
	return Pointer
}

// ValueLocal and ValueShape are the local name the value field of a complex
// type with simple content is named from and its Shape (codegen/doc.go
// "# Complex types").
const (
	ValueLocal = "value"
	ValueShape = Bare
)

// ExactlyOnce returns the {min occurs} = {max occurs} = 1 range, under which,
// among others, an element particle gives a value field and a flattened
// field's branch holds its particle's term (codegen/doc.go "# Particles").
// xsd.NewOccurs cannot reject the constant range, so ExactlyOnce panics if it
// does.
func ExactlyOnce() xsd.Occurs {
	once, err := xsd.NewOccurs(xsderr.Loc{}, 1, 1)
	if err != nil {
		panic(err)
	}
	return once
}

// BranchOccurs returns the occurrence a choice's branch struct holds its
// alternative particle at, the alternative occurring as occurs: occurs with
// {min occurs} 0 read as 1 where {max occurs} is not 0, {max occurs} kept,
// and occurs itself otherwise (codegen/doc.go "# Complex types"). A branch is
// the alternative chosen, so it never holds the empty match the nil
// interface stands for (§3.9.6.3 cos-group-emptiable). xsd.NewOccurs and
// xsd.NewUnboundedOccurs cannot reject a {min occurs} of 1 under a {max
// occurs} of at least 1, so BranchOccurs panics if they do.
func BranchOccurs(occurs xsd.Occurs) xsd.Occurs {
	upper, bounded := occurs.Max()
	if occurs.Min() > 0 || bounded && upper == 0 {
		return occurs
	}
	raised, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, 1)
	if bounded {
		raised, err = xsd.NewOccurs(xsderr.Loc{}, 1, upper)
	}
	if err != nil {
		panic(err)
	}
	return raised
}

// ModelGroupTerm returns the model group term holds and true: an inline
// sequence, choice or all group, or the {model group} of the top-level model
// group definition a <group ref> names (§3.7.2 declare-namedModelGroup),
// followed through [xsd.Schema.ModelGroup]. It returns false for an element
// declaration, an <element ref> or a wildcard. A <group ref> naming no
// definition is an error whose "its" stands for the complex type holding the
// particle, which the caller names. The term is a closed sum, so the switch
// is STYLE T2's sealed-sum exception.
func ModelGroupTerm(schema *xsd.Schema, term xsd.TermOrRef) (xsd.ModelGroup, bool, error) {
	switch term := term.(type) {
	case xsd.ModelGroupRef:
		def, ok := schema.ModelGroup(term.Name)
		if !ok {
			return xsd.ModelGroup{}, false, fmt.Errorf("its <group ref> particle to %s does not resolve", term.Name)
		}
		return def.ModelGroup(), true, nil
	case xsd.ResolvedTerm:
		group, isGroup := term.Term.(xsd.ModelGroup)
		return group, isGroup, nil
	}
	return xsd.ModelGroup{}, false, nil
}
