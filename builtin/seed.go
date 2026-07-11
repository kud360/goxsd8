package builtin

import (
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// MissingPrimitivesError reports that a backend handed to [Seed] does not map
// every builtin primitive datatype Seed requires. It is a precondition
// violation on Seed's b argument — a library/API contract fault, NOT a spec
// validity verdict: no cvc-*/cos-* Schema Component Constraint governs "a
// processor must map primitive N" (that obligation is prose, xmlschema11-1.md
// §3.16.7.4), so this is a plain typed Go error and is deliberately NOT routed
// through xsderr. Detect it with errors.As.
//
// Missing lists every unmapped primitive, in Types order, and is never empty:
// Seed constructs this error only when at least one primitive is missing, so a
// MissingPrimitivesError with no primitives is unrepresentable. Treat Missing
// as read-only.
type MissingPrimitivesError struct {
	// Missing is the unmapped primitive datatypes by QName, in Types order.
	Missing []xsd.QName
}

// Error renders the missing primitives in Clark notation, in Types order.
func (e *MissingPrimitivesError) Error() string {
	names := make([]string, len(e.Missing))
	for i, q := range e.Missing {
		names[i] = q.String()
	}
	return "builtin: backend does not map required primitive datatype(s): " + strings.Join(names, ", ")
}

// Seed composes b with the generated builtin datatype table (Types) into ready
// Simple Type Definition components — the M3 composition step a consumer runs
// before it can validate anything against the builtins.
//
// # Returned slice
//
// The result is deterministic and holds exactly len(Types)+1 elements in this
// fixed order (STYLE D2): xs:anySimpleType first, then one component per row of
// Types in Types order. xs:anySimpleType has no row in Types (it has no facets
// and cannot be a restriction base, §3.2.1.3), so Seed prepends it. Its
// xs:anySimpleType and xs:anyAtomicType nodes are the canonical shared
// singletons from package xsd ([xsd.AnySimpleType]/[xsd.AnyAtomicType]), so
// every primitive's {base type definition} is the one xs:anyAtomicType identity
// and pointer identity (see [xsd.SimpleType]) holds across the whole graph.
// xs:anyType is a Complex Type Definition, outside [xsd.SimpleType]'s scope, and
// is NEVER in the returned slice; it remains a parser-level structural concern
// (M4).
//
// Each component carries its {name}, its {base type definition} as a fully
// linked pointer chain up to the shared xs:anySimpleType node, its {variety}
// (atomic or list, with list item pointers resolved), its own {facets} (the
// value-bearing spec defaults from the row, cos-applicable-facets §4.1.5), and
// an empty {final}. A derived atomic type's {primitive type definition} points
// at its primitive ancestor; a primitive datatype's own {primitive type
// definition} is itself, wired via [xsd.NewPrimitiveType]. Only xs:anyAtomicType
// carries an absent {primitive type definition} (Atomic{Primitive: nil}).
//
// # Precondition and error
//
// Seed requires b to map every builtin PRIMITIVE (a row whose {base type
// definition} is xs:anyAtomicType — the 19 spec-mandatory primitives of
// §3.3 plus precisionDecimal). precisionDecimal is required here by THIS
// project's own always-on policy, not by the XSD 1.1 spec, which lists it as an
// implementation-defined primitive (xsd-precisionDecimal.md). Derived builtins
// need no direct mapping: their governing mapping is their nearest mapped
// ancestor, ultimately a primitive.
//
// If b maps every required primitive, Seed returns (components, nil). Otherwise
// it returns (nil, err) where err is a *[MissingPrimitivesError] naming EVERY
// unmapped primitive (not just the first), in Types order — nil-on-any-error,
// no partial slice. Compose b with [value.Override] to fill gaps from another
// backend. Detect the error with errors.As:
//
//	types, err := builtin.Seed(b)
//	var missing *builtin.MissingPrimitivesError
//	if errors.As(err, &missing) {
//	    // missing.Missing lists the unmapped primitives
//	}
//
// # Purity
//
// Seed allocates a fresh, independent component for every named and derived
// type on each call, so repeated calls are cheap. The two structural anchors —
// xs:anySimpleType and xs:anyAtomicType — are the exception: they are the shared
// immutable singletons from package xsd, so every graph roots on one identity
// (required for [xsd.SimpleType.IsPrimitive] and for pointer identity to hold
// across graphs). Because all shared state is immutable, a consumer may Seed
// once and share the slice, or Seed per schema/rebuild, freely.
func Seed(b value.Backend) ([]*xsd.SimpleType, error) {
	// Precondition scan: every primitive row must be mapped. Collect ALL
	// missing primitives in Types order — a precondition scan, not error-
	// dropping in a loop (STYLE D2/S3). The error is built only when the scan
	// found at least one, so an empty MissingPrimitivesError never escapes.
	var missing []xsd.QName
	for i := range Types {
		if !Types[i].IsPrimitive() {
			continue
		}
		qn := qname(Types[i].Name)
		if _, ok := b.Mapping(qn); !ok {
			missing = append(missing, qn)
		}
	}
	if len(missing) > 0 {
		return nil, &MissingPrimitivesError{Missing: missing}
	}

	index := make(map[string]TypeSpec, len(Types))
	for _, t := range Types {
		index[t.Name] = t
	}

	// The two structural anchors are the canonical, shared xsd singletons — not
	// freshly synthesized here — so every graph roots on one xs:anySimpleType /
	// xs:anyAtomicType identity. That single identity is what makes IsPrimitive
	// (base == xsd.AnyAtomicType by pointer) hold on every Seed-produced
	// primitive; xs:anySimpleType has no Types row (§3.2.1.3) and is prepended.
	built := map[string]*xsd.SimpleType{
		"anySimpleType": xsd.AnySimpleType(),
		"anyAtomicType": xsd.AnyAtomicType(),
	}
	var build func(name string) (*xsd.SimpleType, error)
	build = func(name string) (*xsd.SimpleType, error) {
		if n, ok := built[name]; ok {
			return n, nil
		}
		spec, ok := index[name]
		if !ok {
			return nil, fmt.Errorf("builtin: type %q references unknown base or item %q", name, name)
		}
		facets, err := ownFacets(spec)
		if err != nil {
			return nil, err
		}
		// A primitive's {base type definition} is xs:anyAtomicType and its
		// {primitive type definition} is itself (§3.16.1); NewPrimitiveType wires
		// both — the canonical anchor base and the self-reference — so IsPrimitive
		// reports true.
		if spec.IsPrimitive() {
			node, err := xsd.NewPrimitiveType(xsderr.Loc{}, qname(spec.Name), facets, nil)
			if err != nil {
				return nil, fmt.Errorf("builtin: constructing xs:%s: %w", spec.Name, err)
			}
			built[name] = node
			return node, nil
		}
		base, err := build(spec.Base)
		if err != nil {
			return nil, err
		}
		variety, err := buildVariety(index, built, spec, build)
		if err != nil {
			return nil, err
		}
		node, err := xsd.NewSimpleType(xsderr.Loc{}, qname(spec.Name), variety, base, facets, nil)
		if err != nil {
			return nil, fmt.Errorf("builtin: constructing xs:%s: %w", spec.Name, err)
		}
		built[name] = node
		return node, nil
	}

	out := make([]*xsd.SimpleType, 0, len(Types)+1)
	out = append(out, xsd.AnySimpleType())
	for i := range Types {
		node, err := build(Types[i].Name)
		if err != nil {
			return nil, err
		}
		out = append(out, node)
	}
	return out, nil
}

// buildVariety translates a row's backend-neutral builtin.Variety into the
// resolved xsd.Variety, wiring live component pointers via build.
func buildVariety(index map[string]TypeSpec, built map[string]*xsd.SimpleType, spec TypeSpec, build func(string) (*xsd.SimpleType, error)) (xsd.Variety, error) {
	switch v := spec.Variety.(type) {
	case Atomic:
		// This path builds only DERIVED atomic types (primitives take the
		// NewPrimitiveType path in build, and xs:anyAtomicType is the pre-seeded
		// anchor). {primitive type definition} is the nearest primitive ancestor.
		if pn, ok := primitiveName(index, spec.Name); ok && pn != spec.Name {
			return xsd.Atomic{Primitive: built[pn]}, nil
		}
		return xsd.Atomic{}, nil
	case List:
		item, err := build(v.Item)
		if err != nil {
			return nil, err
		}
		return xsd.List{Item: item}, nil
	default:
		return nil, fmt.Errorf("builtin: type %q has no variety", spec.Name)
	}
}

// primitiveName returns the name of name's nearest primitive ancestor by
// walking the base chain in the data table. ok is false when the chain reaches
// xs:anySimpleType without passing a primitive (xs:anyAtomicType and the list
// types).
func primitiveName(index map[string]TypeSpec, name string) (string, bool) {
	for {
		t, ok := index[name]
		if !ok {
			return "", false
		}
		if t.IsPrimitive() {
			return t.Name, true
		}
		if t.Base == "anySimpleType" {
			return "", false
		}
		name = t.Base
	}
}

// ownFacets builds a type's own Constraining Facets from its row: one xsd.Facet
// per applicable facet that carries a spec value (Facet.Default non-empty),
// with {fixed} from Facet.Fixed (cos-applicable-facets, §4.1.5). An applicable
// facet with no spec value is not a Constraining Facet the type declares — it
// is expressible via TypeSpec.Applies — so it contributes no own facet.
func ownFacets(spec TypeSpec) ([]xsd.Facet, error) {
	var facets []xsd.Facet
	for _, f := range spec.Facets {
		if f.Default == "" {
			continue
		}
		kind, ok := facetKind(f.Name)
		if !ok {
			// A value-bearing facet with no xsd.FacetKind (maxScale/minScale
			// are precisionDecimal-only and carry no default, so they never
			// reach here) would silently drop a constraint: refuse instead.
			return nil, fmt.Errorf("builtin: type %q has value-bearing facet %q with no FacetKind", spec.Name, f.Name)
		}
		facets = append(facets, xsd.NewFacet(kind, []string{f.Default}, f.Fixed))
	}
	return facets, nil
}

// facetKind maps a builtin facet name to its xsd.FacetKind. ok is false for the
// precisionDecimal-only maxScale/minScale, which xsd.FacetKind excludes by
// design; those carry no spec default, so ownFacets never asks for their kind.
func facetKind(name FacetName) (xsd.FacetKind, bool) {
	switch name {
	case "length":
		return xsd.FacetLength, true
	case "minLength":
		return xsd.FacetMinLength, true
	case "maxLength":
		return xsd.FacetMaxLength, true
	case "pattern":
		return xsd.FacetPattern, true
	case "enumeration":
		return xsd.FacetEnumeration, true
	case "whiteSpace":
		return xsd.FacetWhiteSpace, true
	case "maxInclusive":
		return xsd.FacetMaxInclusive, true
	case "maxExclusive":
		return xsd.FacetMaxExclusive, true
	case "minExclusive":
		return xsd.FacetMinExclusive, true
	case "minInclusive":
		return xsd.FacetMinInclusive, true
	case "totalDigits":
		return xsd.FacetTotalDigits, true
	case "fractionDigits":
		return xsd.FacetFractionDigits, true
	case "assertions":
		return xsd.FacetAssertions, true
	case "explicitTimezone":
		return xsd.FacetExplicitTimezone, true
	}
	return 0, false
}

// qname bundles a builtin local name with the XSD namespace.
func qname(local string) xsd.QName { return xsd.QName{Space: xsd.XMLSchemaNS, Local: local} }
