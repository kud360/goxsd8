package xsd_test

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This is simpletypefixture_test.go's twin for the EXTERNAL test package, over
// the exported API. The duplication is forced by Go's internal/external test
// split — package xsd_test cannot see package xsd's test files — and is kept to
// the two helpers the external fixtures actually use rather than the whole set.

// noSchema is the [xsd.TypeResolver] these fixtures resolve against: it resolves
// nothing, which is total on the owned-arm chains they build, because no by-name
// reference exists in one to be looked up.
type noSchema struct{}

func (noSchema) Type(xsd.QName) (xsd.TypeDefinition, bool) { return nil, false }

// newCheckedSimpleType is [xsd.NewSimpleType] followed by
// [xsd.SimpleType.CheckDerivation] over an OWNED base — the pairing that used to
// be NewSimpleType alone. A nil base still means "this type IS
// xs:anySimpleType", the nil slot.
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

// newCheckedPrimitiveType is the primitive-datatype twin of
// newCheckedSimpleType.
func newCheckedPrimitiveType(loc xsderr.Loc, name xsd.QName, ownFacets []xsd.Facet, final []xsd.DerivationMethod) (*xsd.SimpleType, error) {
	st, err := xsd.NewPrimitiveType(loc, name, ownFacets, final)
	if err != nil {
		return nil, err
	}
	if err := st.CheckDerivation(noSchema{}); err != nil {
		return nil, err
	}
	return st, nil
}
