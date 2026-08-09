package strict_test

import "github.com/kud360/goxsd8/xsd"

// noSchema is fixture_test.go's twin for the EXTERNAL test package: an
// [xsd.TypeResolver] that resolves nothing, which is total on the OWNED-arm
// chains these fixtures use. The duplication is Go's internal/external test
// split, not a second design.
type noSchema struct{}

func (noSchema) Type(xsd.QName) (xsd.TypeDefinition, bool) { return nil, false }
