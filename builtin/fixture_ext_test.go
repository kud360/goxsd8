package builtin_test

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is the FIXTURE SEAM for this package's external tests (STYLE T4).
// Every simple type it builds or reads is an OWNED-arm chain — a live
// *xsd.SimpleType in the {base type definition} slot, which is what
// [builtin.Seed] produces and what a Schema-less assembler builds — so noSchema
// is the right resolver for all of them.

// noSchema is the [xsd.TypeResolver] these fixtures resolve against: it resolves
// nothing, which is total on an owned-arm chain because no by-name reference
// exists in one to be looked up. It mirrors package builtin's own unexported
// twin, which Seed passes to CheckDerivation; the duplication is Go's
// internal/external test split, not a second design.
type noSchema struct{}

func (noSchema) Type(xsd.QName) (xsd.TypeDefinition, bool) { return nil, false }

// newCheckedSimpleType is [xsd.NewSimpleType] followed by
// [xsd.SimpleType.CheckDerivation] over an OWNED base — the pairing that used to
// be NewSimpleType alone, before the {base type definition} was deferred and the
// graph checks moved to a finalize-time entry point.
func newCheckedSimpleType(loc xsderr.Loc, name xsd.QName, derivation xsd.SimpleTypeDerivation, base *xsd.SimpleType, ownFacets []xsd.Facet, final []xsd.DerivationMethod) (*xsd.SimpleType, error) {
	var slot xsd.SimpleTypeOrRef
	if base != nil {
		slot = xsd.OwnedSimpleType{Definition: base}
	}
	st, err := xsd.NewSimpleType(loc, name, derivation, slot, ownFacets, final)
	if err != nil {
		return nil, err
	}
	if err := st.CheckDerivation(noSchema{}); err != nil {
		return nil, err
	}
	return st, nil
}

// The four readers below are the resolver-threaded accessors applied to an
// owned-arm chain, where they cannot fail. Each PANICS on an error rather than
// taking a *testing.T, so a fixture reads as an expression exactly where the
// pre-deferral accessor did.

func mustBase(t *xsd.SimpleType) *xsd.SimpleType {
	base, err := t.Base(noSchema{})
	if err != nil {
		panic("builtin: test fixture: Base: " + err.Error())
	}
	return base
}

func mustVariety(t *xsd.SimpleType) xsd.Variety {
	v, err := t.Variety(noSchema{})
	if err != nil {
		panic("builtin: test fixture: Variety: " + err.Error())
	}
	return v
}

func mustPrimitive(t *xsd.SimpleType) *xsd.SimpleType {
	p, err := t.Primitive(noSchema{})
	if err != nil {
		panic("builtin: test fixture: Primitive: " + err.Error())
	}
	return p
}

func mustItem(t *xsd.SimpleType) *xsd.SimpleType {
	i, err := t.Item(noSchema{})
	if err != nil {
		panic("builtin: test fixture: Item: " + err.Error())
	}
	return i
}
