package value

import (
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is the shared FIXTURE SEAM for this package's tests (STYLE T4).
// Every simple type built here is an OWNED-arm chain — a live *xsd.SimpleType in
// the {base type definition} slot — which is what a programmatic assembler
// builds and is why noSchema is the right resolver for all of them.

// noSchema is the [xsd.TypeResolver] these fixtures pass to every entry point
// that takes one. It resolves nothing, which is total on an owned-arm chain
// because no by-name reference exists in one to be looked up.
type noSchema struct{}

func (noSchema) Type(xsd.QName) (xsd.TypeDefinition, bool) { return nil, false }

// newCheckedSimpleType is [xsd.NewSimpleType] followed by
// [xsd.SimpleType.CheckDerivation] over an OWNED base — the pairing that used to
// be NewSimpleType alone, before the {base type definition} was deferred and the
// graph checks moved to a finalize-time entry point. A nil base still means
// "this type IS xs:anySimpleType", the nil slot.
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
