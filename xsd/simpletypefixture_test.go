package xsd

import "github.com/kud360/goxsd8/xsderr"

// This file is the shared FIXTURE SEAM for every simple-type test in this
// package (STYLE T4 — one set of helpers, not nine). It exists because #636
// split what used to be one call into two: NewSimpleType now builds a WELL
// FORMED component, and SimpleType.CheckDerivation — run by finalize's Phase D
// pass — decides whether it is well DERIVED. A test whose subject is a
// cross-reference clause (cos-st-restricts, st-props-correct clauses 1/3/5) is
// about the second, and a test whose subject is a derived property reads it
// through a resolver-threaded accessor.
//
// Every fixture here builds an OWNED-arm chain — a live *SimpleType in the {base
// type definition} slot — which is what a programmatic assembler does and is why
// noSchema is the right resolver for all of them. The by-NAME arm has its own
// tests, which resolve against a real Schema (simpletyperef_test.go).

// noSchema is the TypeResolver these fixtures resolve against: it resolves
// nothing, which is total on an owned-arm chain because no by-name reference
// exists in one to be looked up.
type noSchema struct{}

func (noSchema) Type(QName) (TypeDefinition, bool) { return nil, false }

// newCheckedSimpleType is NewSimpleType followed by CheckDerivation — the
// pairing that used to be NewSimpleType alone — over an OWNED base. It takes the
// base as a live *SimpleType, so a fixture reads exactly as it did before the
// split, and a nil base still means "this type IS xs:anySimpleType" (the nil
// slot, never an OwnedSimpleType wrapping nil, which NewSimpleType rejects).
//
// Either half's error is returned verbatim, so a test asserting a rule ID gets
// the same one whichever half charges it.
func newCheckedSimpleType(loc xsderr.Loc, name QName, derivation SimpleTypeDerivation, base *SimpleType, ownFacets []Facet, final []DerivationMethod) (*SimpleType, error) {
	st, err := NewSimpleType(loc, name, derivation, ownedBase(base), ownFacets, final)
	if err != nil {
		return nil, err
	}
	if err := st.CheckDerivation(noSchema{}); err != nil {
		return nil, err
	}
	return st, nil
}

// newCheckedPrimitiveType is NewPrimitiveType followed by CheckDerivation, the
// primitive-datatype twin of newCheckedSimpleType.
func newCheckedPrimitiveType(loc xsderr.Loc, name QName, ownFacets []Facet, final []DerivationMethod) (*SimpleType, error) {
	st, err := NewPrimitiveType(loc, name, ownFacets, final)
	if err != nil {
		return nil, err
	}
	if err := st.CheckDerivation(noSchema{}); err != nil {
		return nil, err
	}
	return st, nil
}

// ownedBase maps a live base pointer to the {base type definition} slot value it
// belongs in: nil stays the nil slot (·absent·, i.e. xs:anySimpleType), and
// anything else is the owned arm.
func ownedBase(base *SimpleType) SimpleTypeOrRef {
	if base == nil {
		return nil
	}
	return OwnedSimpleType{Definition: base}
}

// listOf and unionOf are the same mapping for the {item type definition} and
// {member type definitions} slots, which take no nil: a fixture names live item
// and member pointers and gets the owned arm for each. A fixture whose subject
// IS the slot encoding — a by-name item, an absent one — writes the arm out
// instead of calling these.

func listOf(item *SimpleType) ListDerivation {
	return ListDerivation{Item: OwnedSimpleType{Definition: item}}
}

func unionOf(members ...*SimpleType) UnionDerivation {
	if len(members) == 0 {
		return UnionDerivation{}
	}
	slots := make([]SimpleTypeOrRef, 0, len(members))
	for _, m := range members {
		slots = append(slots, OwnedSimpleType{Definition: m})
	}
	return UnionDerivation{Members: slots}
}

// The six readers below are the resolver-threaded accessors applied to an
// owned-arm chain, where they cannot fail. Each PANICS on an error rather than
// taking a *testing.T, so a fixture reads as an expression exactly where the
// pre-#636 accessor did; a panic here means a fixture built a by-name base,
// which these helpers are not for.

func mustBase(t *SimpleType) *SimpleType {
	base, err := t.Base(noSchema{})
	if err != nil {
		panic("xsd: test fixture: Base: " + err.Error())
	}
	return base
}

func mustVariety(t *SimpleType) Variety {
	v, err := t.Variety(noSchema{})
	if err != nil {
		panic("xsd: test fixture: Variety: " + err.Error())
	}
	return v
}

func mustPrimitive(t *SimpleType) *SimpleType {
	p, err := t.Primitive(noSchema{})
	if err != nil {
		panic("xsd: test fixture: Primitive: " + err.Error())
	}
	return p
}

func mustItem(t *SimpleType) *SimpleType {
	i, err := t.Item(noSchema{})
	if err != nil {
		panic("xsd: test fixture: Item: " + err.Error())
	}
	return i
}

func mustMembers(t *SimpleType) []*SimpleType {
	m, err := t.Members(noSchema{})
	if err != nil {
		panic("xsd: test fixture: Members: " + err.Error())
	}
	return m
}

func mustEffectiveFacets(t *SimpleType) []EffectiveFacet {
	eff, err := t.EffectiveFacets(noSchema{})
	if err != nil {
		panic("xsd: test fixture: EffectiveFacets: " + err.Error())
	}
	return eff
}
