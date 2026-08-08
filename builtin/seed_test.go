package builtin_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// mapBackend is a value.Backend backed by a set of primitive local names it
// claims to map; the mapping itself is a trivial identity Parse (Seed only
// checks presence, never calls Parse).
type mapBackend map[string]struct{}

func (b mapBackend) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ.Space != xsd.XMLSchemaNS {
		return value.Mapping{}, false
	}
	if _, ok := b[typ.Local]; !ok {
		return value.Mapping{}, false
	}
	return value.Mapping{
		Parse: func(lexical string, _ value.Context) (value.Value, error) { return lexical, nil },
	}, true
}

// allPrimitives is a backend that maps exactly the primitive rows of Types —
// the minimal backend Seed accepts.
func allPrimitives() mapBackend {
	b := mapBackend{}
	for _, t := range builtin.Types {
		if t.IsPrimitive() {
			b[t.Name] = struct{}{}
		}
	}
	return b
}

// byName indexes a Seed result by local name for lookups in assertions.
func byName(types []*xsd.SimpleType) map[string]*xsd.SimpleType {
	m := make(map[string]*xsd.SimpleType, len(types))
	for _, t := range types {
		m[t.Name().Local] = t
	}
	return m
}

func TestSeedSuccessShapeAndOrder(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed with all primitives mapped: unexpected error: %v", err)
	}
	if got, want := len(types), len(builtin.Types)+1; got != want {
		t.Fatalf("len(types) = %d, want len(Types)+1 = %d", got, want)
	}
	if types[0].Name() != (xsd.QName{Space: xsd.XMLSchemaNS, Local: "anySimpleType"}) {
		t.Fatalf("types[0] = %v, want xs:anySimpleType prepended", types[0].Name())
	}
	if !types[0].IsAnySimpleType() {
		t.Errorf("prepended node must report IsAnySimpleType")
	}
	// The remaining len(Types) elements are Types in order.
	for i := range builtin.Types {
		if got, want := types[i+1].Name().Local, builtin.Types[i].Name; got != want {
			t.Fatalf("types[%d] = %q, want %q (Types order)", i+1, got, want)
		}
	}
}

func TestSeedNoAnyType(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	for _, ty := range types {
		if ty.Name().Local == "anyType" {
			t.Fatalf("xs:anyType (a complex type) must never appear in Seed's result")
		}
	}
}

func TestSeedMissingPrimitivesCollectsAllInOrder(t *testing.T) {
	// Map every primitive EXCEPT decimal and boolean.
	b := allPrimitives()
	delete(b, "decimal")
	delete(b, "boolean")

	types, err := builtin.Seed(b)
	if types != nil {
		t.Errorf("Seed must return a nil slice on error, got %d components", len(types))
	}
	var missing *builtin.MissingPrimitivesError
	if !errors.As(err, &missing) {
		t.Fatalf("error %v is not errors.As-detectable as *MissingPrimitivesError", err)
	}
	if len(missing.Missing) == 0 {
		t.Fatal("MissingPrimitivesError.Missing must never be empty")
	}
	// boolean precedes decimal? No: Types order is string, boolean, decimal,...
	// so the collected order is boolean then decimal.
	want := []xsd.QName{
		{Space: xsd.XMLSchemaNS, Local: "boolean"},
		{Space: xsd.XMLSchemaNS, Local: "decimal"},
	}
	if len(missing.Missing) != len(want) {
		t.Fatalf("Missing = %v, want %v", missing.Missing, want)
	}
	for i := range want {
		if missing.Missing[i] != want[i] {
			t.Fatalf("Missing[%d] = %v, want %v (Types order)", i, missing.Missing[i], want[i])
		}
	}
}

func TestSeedMissingErrorMessageNamesQNames(t *testing.T) {
	b := allPrimitives()
	delete(b, "decimal")
	_, err := builtin.Seed(b)
	if err == nil {
		t.Fatal("expected an error")
	}
	if got := err.Error(); !strings.Contains(got, "{http://www.w3.org/2001/XMLSchema}decimal") {
		t.Errorf("error %q must name the missing primitive in Clark notation", got)
	}
}

func TestSeedAnySimpleTypeSingleIdentity(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	idx := byName(types)
	anySimple := types[0]
	anyAtomic := idx["anyAtomicType"]
	if anyAtomic == nil {
		t.Fatal("anyAtomicType missing from result")
	}
	if anyAtomic.Base() != anySimple {
		t.Errorf("anyAtomicType.Base() must be the one prepended anySimpleType node (pointer identity)")
	}
	// Every primitive's base chain must reach that same anySimpleType node.
	for _, t2 := range builtin.Types {
		if !t2.IsPrimitive() {
			continue
		}
		node := idx[t2.Name]
		if node.Base() != anyAtomic {
			t.Errorf("primitive %q base is not the shared anyAtomicType node", t2.Name)
		}
	}
}

func TestSeedBaseChainAndPrimitivePointer(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	idx := byName(types)
	decimal := idx["decimal"]
	integer := idx["integer"]
	if integer.Base() != decimal {
		t.Errorf("integer.Base() must be the shared decimal node")
	}
	// A derived atomic type points {primitive type definition} at its primitive
	// ancestor (decimal).
	at, ok := integer.Variety().(xsd.Atomic)
	if !ok {
		t.Fatalf("integer variety = %T, want xsd.Atomic", integer.Variety())
	}
	if at.Primitive() != decimal {
		t.Errorf("integer {primitive type definition} = %v, want the decimal node", at.Primitive())
	}
	// A primitive's own {primitive type definition} is itself (§3.16.1).
	dat, ok := decimal.Variety().(xsd.Atomic)
	if !ok {
		t.Fatalf("decimal variety = %T, want xsd.Atomic", decimal.Variety())
	}
	if dat.Primitive() != decimal {
		t.Errorf("decimal {primitive type definition} must point at itself, got %v", dat.Primitive())
	}
}

// TestSeedPrimitivesReportIsPrimitive proves the warden-flagged defect is fixed:
// every Seed-produced primitive's {base type definition} is the canonical
// xs:anyAtomicType anchor, so IsPrimitive reports true, and its own {primitive
// type definition} self-references. The two special types are not primitive.
func TestSeedPrimitivesReportIsPrimitive(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	idx := byName(types)

	for _, spec := range builtin.Types {
		if !spec.IsPrimitive() {
			continue
		}
		node := idx[spec.Name]
		if node == nil {
			t.Fatalf("primitive %q missing from Seed result", spec.Name)
		}
		if !node.IsPrimitive() {
			t.Errorf("%q.IsPrimitive() = false, want true (base must be the canonical xs:anyAtomicType)", spec.Name)
		}
		if node.Base() != xsd.AnyAtomicType() {
			t.Errorf("%q.Base() is not the canonical xsd.AnyAtomicType() anchor", spec.Name)
		}
		at, ok := node.Variety().(xsd.Atomic)
		if !ok {
			t.Fatalf("%q variety = %T, want xsd.Atomic", spec.Name, node.Variety())
		}
		if at.Primitive() != node {
			t.Errorf("%q {primitive type definition} must self-reference, got %v", spec.Name, at.Primitive())
		}
	}

	// The two special types are not primitive.
	if idx["anyAtomicType"].IsPrimitive() {
		t.Error("xs:anyAtomicType must not report IsPrimitive")
	}
	if types[0].IsPrimitive() {
		t.Error("xs:anySimpleType must not report IsPrimitive")
	}
}

func TestSeedListVarietyWired(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	idx := byName(types)
	nmtokens := idx["NMTOKENS"]
	lst, ok := nmtokens.Variety().(xsd.List)
	if !ok {
		t.Fatalf("NMTOKENS variety = %T, want xsd.List", nmtokens.Variety())
	}
	if lst.Item() != idx["NMTOKEN"] {
		t.Errorf("NMTOKENS list item must be the shared NMTOKEN node")
	}
}

// TestSeedListTwoStepDerivation pins the two-step derivation Datatypes §3.4.5/
// §3.4.10/§3.4.12 define for the three list builtins: an ANONYMOUS list over the
// singular item type is defined first, and the named plural type restricts THAT
// — it is not a one-step list off xs:anySimpleType. The anonymous node's whole
// {facets} is whiteSpace = collapse with {fixed} = true (§3.16.2.1 map.std.common
// case 3, the only set cos-st-restricts clause 2.2.1.2 admits) and the named
// type's own {facets} is minLength = 1, UNFIXED (§3.4.5.1 permits a further
// restriction to raise it).
func TestSeedListTwoStepDerivation(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	idx := byName(types)
	for _, tc := range []struct{ plural, item string }{
		{"NMTOKENS", "NMTOKEN"},
		{"IDREFS", "IDREF"},
		{"ENTITIES", "ENTITY"},
	} {
		t.Run(tc.plural, func(t *testing.T) {
			named := idx[tc.plural]
			if named == nil {
				t.Fatalf("xs:%s is not seeded", tc.plural)
			}
			anon := named.Base()
			if anon == nil || anon == xsd.AnySimpleType() {
				t.Fatalf("xs:%s {base type definition} = %v, want the anonymous intermediate list", tc.plural, anon)
			}
			if got := anon.Name(); got != (xsd.QName{}) {
				t.Errorf("intermediate list {name} = %v, want absent (the zero QName)", got)
			}
			if anon.Base() != xsd.AnySimpleType() {
				t.Errorf("intermediate list {base type definition} = %v, want xs:anySimpleType", anon.Base())
			}
			lst, ok := anon.Variety().(xsd.List)
			if !ok {
				t.Fatalf("intermediate {variety} = %T, want xsd.List", anon.Variety())
			}
			if lst.Item() != idx[tc.item] {
				t.Errorf("intermediate {item type definition} = %v, want the shared xs:%s node", lst.Item(), tc.item)
			}
			own := anon.OwnFacets()
			if len(own) != 1 || own[0].Kind() != xsd.FacetWhiteSpace {
				t.Fatalf("intermediate own facets = %v, want exactly one whiteSpace facet (cos-st-restricts 2.2.1.2)", own)
			}
			if vals := own[0].Values(); len(vals) != 1 || vals[0] != "collapse" {
				t.Errorf("intermediate whiteSpace values = %v, want [collapse]", vals)
			}
			if fixed, ok := own[0].Fixed(); !ok || !fixed {
				t.Errorf("intermediate whiteSpace must be fixed")
			}

			namedOwn := named.OwnFacets()
			if len(namedOwn) != 1 || namedOwn[0].Kind() != xsd.FacetMinLength {
				t.Fatalf("xs:%s own facets = %v, want exactly one minLength facet", tc.plural, namedOwn)
			}
			if vals := namedOwn[0].Values(); len(vals) != 1 || vals[0] != "1" {
				t.Errorf("xs:%s minLength values = %v, want [1]", tc.plural, vals)
			}
			if fixed, ok := namedOwn[0].Fixed(); !ok || fixed {
				t.Errorf("xs:%s minLength must NOT be fixed (§3.4.5.1)", tc.plural)
			}
		})
	}
}

// TestSeedExcludesAnonymousIntermediates pins the returned slice's contract
// against the graph change: the anonymous intermediate lists are components of
// the seeded graph but NOT members of the result, which stays exactly
// len(Types)+1 with every member named.
func TestSeedExcludesAnonymousIntermediates(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if len(types) != len(builtin.Types)+1 {
		t.Fatalf("Seed returned %d components, want len(Types)+1 = %d", len(types), len(builtin.Types)+1)
	}
	for i, st := range types {
		if st.Name() == (xsd.QName{}) {
			t.Errorf("Seed result[%d] has an absent {name}; anonymous intermediates must not be returned", i)
		}
	}
}

func TestSeedFacetsAttached(t *testing.T) {
	types, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	idx := byName(types)
	// decimal fixes whiteSpace=collapse (cos-applicable-facets); it must be an
	// own facet, and applicable-but-unset facets (e.g. maxInclusive) must not.
	var sawWhiteSpace, sawMaxInclusive bool
	for _, f := range idx["decimal"].OwnFacets() {
		switch f.Kind() {
		case xsd.FacetWhiteSpace:
			sawWhiteSpace = true
			if vals := f.Values(); len(vals) != 1 || vals[0] != "collapse" {
				t.Errorf("decimal whiteSpace values = %v, want [collapse]", vals)
			}
			if fixed, ok := f.Fixed(); !ok || !fixed {
				t.Errorf("decimal whiteSpace must be fixed")
			}
		case xsd.FacetMaxInclusive:
			sawMaxInclusive = true
		default:
			// other facet kinds are not asserted here
		}
	}
	if !sawWhiteSpace {
		t.Errorf("decimal must carry an own whiteSpace facet")
	}
	if sawMaxInclusive {
		t.Errorf("decimal must not carry an own maxInclusive facet (applicable but unset)")
	}
}

func TestSeedIdempotentIndependentGraphs(t *testing.T) {
	a, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed a: %v", err)
	}
	b, err := builtin.Seed(allPrimitives())
	if err != nil {
		t.Fatalf("Seed b: %v", err)
	}
	if len(a) != len(b) {
		t.Fatalf("two Seed calls produced different lengths %d vs %d", len(a), len(b))
	}
	// The two structural anchors are the shared xsd singletons and MUST be
	// identical across calls (single identity is what makes IsPrimitive work);
	// every other node is freshly allocated and MUST differ between calls.
	shared := map[*xsd.SimpleType]bool{
		xsd.AnySimpleType(): true,
		xsd.AnyAtomicType(): true,
	}
	for i := range a {
		if a[i].Name() != b[i].Name() {
			t.Fatalf("node %d names differ across calls: %v vs %v", i, a[i].Name(), b[i].Name())
		}
		if shared[a[i]] {
			if a[i] != b[i] {
				t.Fatalf("shared anchor %v must be the same node across calls", a[i].Name())
			}
			continue
		}
		if a[i] == b[i] {
			t.Fatalf("Seed must allocate independent graphs: node %d (%v) shared between calls", i, a[i].Name())
		}
	}
}

// TestFacetKindByNameCoversTheGeneratedTable pins the EXPORTED half of the
// name↔kind bijection parser.facetKindOf now delegates to (#323), reached only
// through the public surface: it walks every FacetName the generated Types table
// is keyed by, requires each to resolve, and then requires the resolved kinds to
// cover the WHOLE closed xsd.FacetKind range with no two names colliding on one
// kind. A FacetKind added without a bridge-table row — or a row misspelled
// against the generated table — fails here instead of becoming a silent lookup
// miss in the parser, which is exactly how maxScale/minScale were dropped once
// already (#219).
func TestFacetKindByNameCoversTheGeneratedTable(t *testing.T) {
	seen := make(map[xsd.FacetKind]builtin.FacetName)
	for _, spec := range builtin.Types {
		for _, f := range spec.Facets {
			kind, ok := builtin.FacetKindByName(f.Name)
			if !ok {
				t.Errorf("FacetKindByName(%q) = (_, false), want a kind: %s lists it as applicable", f.Name, spec.Name)
				continue
			}
			if prior, dup := seen[kind]; dup && prior != f.Name {
				t.Errorf("FacetKindByName maps both %q and %q to %s; the bijection is not injective", prior, f.Name, kind)
			}
			seen[kind] = f.Name
		}
	}
	for k := xsd.FacetLength; k <= xsd.FacetMinScale; k++ {
		if _, ok := seen[k]; !ok {
			t.Errorf("no builtin facet name resolves to FacetKind %s", k)
		}
	}
	// §4.3.13 spells the facet "assertions"; "assertion" is the ELEMENT name a
	// schema writes. parser.facetKindOf leans on the two being distinct.
	if _, ok := builtin.FacetKindByName("assertion"); ok {
		t.Error(`FacetKindByName("assertion") = (_, true), want false: the facet is spelled "assertions"`)
	}
}
