package strict

import "github.com/kud360/goxsd8/xsd"

// noSchema is the [xsd.TypeResolver] this package's fixtures resolve against: it
// resolves nothing, which is total on the OWNED-arm chains they build (a live
// *xsd.SimpleType in the {base type definition} slot), because no by-name
// reference exists in one to be looked up. Its twins live in packages builtin,
// value, xsd and conformance for the same reason; each is local to the
// Schema-less lane it serves.
type noSchema struct{}

func (noSchema) Type(xsd.QName) (xsd.TypeDefinition, bool) { return nil, false }
