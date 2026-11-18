package xsd

import "github.com/kud360/goxsd8/internal/nearname"

// nameHint is the clause a src-resolve clause 1 charge appends for missing, a
// QName naming no component of the kind its lookup seeks: nearname.Hint's, over
// the components of that kind. Each *Schema method below is one kind's hint,
// drawing from that §3.17.1 property's slice in builder order; noHint is the
// one for a lookup with no population to draw from.
type nameHint func(missing QName) string

// noHint adds nothing: SimpleType.Base, Item and Members resolve through an
// arbitrary TypeResolver, which offers no population to search.
func noHint(QName) string { return "" }

// hintAmong is nearname.Hint over candidates, spelling the one it names as
// every src-resolve message spells the missing name, by QName.Display.
func hintAmong(missing QName, candidates []nearname.Name) string {
	return nearname.Hint(nearname.Name(missing), candidates, func(n nearname.Name) string { return QName(n).Display() })
}

// typeHint is the hint for a {type definition} slot that takes either variety
// (src-resolve clause 1.1): every named type definition.
func (s *Schema) typeHint(missing QName) string {
	return s.typeHintWhere(missing, func(TypeDefinition) bool { return true })
}

// simpleTypeHint is the hint for a slot that must resolve to a simple type —
// an attribute declaration's type=, and a simple type's base=, itemType= and
// memberTypes= — whose lookup charges a complex one as the wrong kind.
func (s *Schema) simpleTypeHint(missing QName) string {
	return s.typeHintWhere(missing, func(t TypeDefinition) bool {
		_, simple := t.(*SimpleType)
		return simple
	})
}

// typeHintWhere draws from {type definitions} the entries keep admits.
func (s *Schema) typeHintWhere(missing QName, keep func(TypeDefinition) bool) string {
	var names []nearname.Name
	for _, t := range s.types {
		if keep(t) {
			names = append(names, nearname.Name(t.Name()))
		}
	}
	return hintAmong(missing, names)
}

// elementHint is the hint for an <element ref> (src-resolve clause 1.3).
func (s *Schema) elementHint(missing QName) string {
	names := make([]nearname.Name, 0, len(s.elements))
	for _, e := range s.elements {
		names = append(names, nearname.Name(e.Name()))
	}
	return hintAmong(missing, names)
}

// attributeHint is the hint for an <attribute ref> (src-resolve clause 1.2),
// the seeded xsi: and xml: declarations among its candidates.
func (s *Schema) attributeHint(missing QName) string {
	names := make([]nearname.Name, 0, len(s.attributes))
	for _, a := range s.attributes {
		names = append(names, nearname.Name(a.Name()))
	}
	return hintAmong(missing, names)
}

// modelGroupHint is the hint for a <group ref> (src-resolve clause 1.5).
func (s *Schema) modelGroupHint(missing QName) string {
	names := make([]nearname.Name, 0, len(s.modelGroups))
	for _, g := range s.modelGroups {
		names = append(names, nearname.Name(g.Name()))
	}
	return hintAmong(missing, names)
}

// attributeGroupHint is the hint for an <attributeGroup ref> (src-resolve
// clause 1.4).
func (s *Schema) attributeGroupHint(missing QName) string {
	names := make([]nearname.Name, 0, len(s.attributeGroups))
	for _, g := range s.attributeGroups {
		names = append(names, nearname.Name(g.Name()))
	}
	return hintAmong(missing, names)
}

// referencedKeyHint is the hint for a keyref's refer= (src-resolve clause
// 1.7): the key and unique definitions only, since resolveKeyref charges a
// keyref target c-props-correct clause 1.
func (s *Schema) referencedKeyHint(missing QName) string {
	var names []nearname.Name
	for _, ic := range s.identityConstraints {
		if ic.Category() != IdentityConstraintKeyref {
			names = append(names, nearname.Name(ic.Name()))
		}
	}
	return hintAmong(missing, names)
}
