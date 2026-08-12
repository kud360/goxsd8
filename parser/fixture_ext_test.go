package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// This file is the FIXTURE SEAM for reading a PRODUCED simple type's derived
// properties (STYLE T4). Unlike the other packages' fixtures, these resolve
// against the finalized [xsd.Schema] rather than against a resolves-nothing
// stub, and that is the point: the producer emits an [xsd.SimpleTypeRef] for
// every by-name base=, so following a produced base chain IS a schema lookup.
// A test that asserted the chain any other way would stop testing what the
// producer actually built.

// mustBase resolves st's {base type definition} through s. It fails the test on
// an unresolvable base rather than returning one: every reference in a schema
// that survived Produce has already been charged src-resolve at finalize, so an
// error here means the test built something Produce would have rejected.
func mustBase(t *testing.T, s *xsd.Schema, st *xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	base, err := st.Base(s)
	if err != nil {
		t.Fatalf("%s {base type definition}: %v", st.Name(), err)
	}
	return base
}

// mustVariety resolves st's {variety} through s.
func mustVariety(t *testing.T, s *xsd.Schema, st *xsd.SimpleType) xsd.Variety {
	t.Helper()
	v, err := st.Variety(s)
	if err != nil {
		t.Fatalf("%s {variety}: %v", st.Name(), err)
	}
	return v
}

// mustPrimitive resolves st's {primitive type definition} through s.
func mustPrimitive(t *testing.T, s *xsd.Schema, st *xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	p, err := st.Primitive(s)
	if err != nil {
		t.Fatalf("%s {primitive type definition}: %v", st.Name(), err)
	}
	return p
}

// mustItem resolves st's {item type definition} through s. A produced itemType=
// is an [xsd.SimpleTypeRef], so this read is a schema lookup for exactly the
// reason mustBase's is.
func mustItem(t *testing.T, s *xsd.Schema, st *xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	item, err := st.Item(s)
	if err != nil {
		t.Fatalf("%s {item type definition}: %v", st.Name(), err)
	}
	return item
}

// mustEffectiveFacets resolves st's {facets} overlay through s.
func mustEffectiveFacets(t *testing.T, s *xsd.Schema, st *xsd.SimpleType) []xsd.EffectiveFacet {
	t.Helper()
	eff, err := st.EffectiveFacets(s)
	if err != nil {
		t.Fatalf("%s {facets}: %v", st.Name(), err)
	}
	return eff
}
