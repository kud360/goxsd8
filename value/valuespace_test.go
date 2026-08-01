package value

import (
	"strconv"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// vsPrim builds a primitive named local in the XML Schema namespace, carrying
// the collapse whiteSpace facet so a value constraint's {lexical form} is
// normalized before it is parsed (key-nv §3.1.4).
func vsPrim(t *testing.T, local string) *xsd.SimpleType {
	t.Helper()
	p, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: local},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType(%s): %v", local, err)
	}
	return p
}

// vsDerived builds an unmapped restriction of base — the widest-space case: its
// values live in base's space, governed by base's mapping.
func vsDerived(t *testing.T, local string, base *xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	prim, _ := base.Variety().(xsd.Atomic)
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		xsd.NewAtomic(prim.Primitive()), base, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%s): %v", local, err)
	}
	return st
}

func vsFixed(lexical string) xsd.ValueConstraint {
	return xsd.NewValueConstraint(xsd.ValueFixed, lexical)
}

// TestValueSpaceDecidesInOneSpace pins the whole point of the adapter: two
// {lexical form}s that differ are compared by VALUE, not by string — "1" and
// "01" are one xs:integer value (Datatypes §2.2.1) — while genuinely different
// values are decided NOT the same, which is the verdict au-props-correct clause
// 3 and loc-testSubP clauses 4.2/5.2.2 turn into a rejection.
func TestValueSpaceDecidesInOneSpace(t *testing.T) {
	prim := vsPrim(t, "int")
	vs := NewValueSpace(intBackend{mapped: prim.Name()})
	derived := vsDerived(t, "narrow", prim)

	for _, tc := range []struct {
		name        string
		ta, tb      *xsd.SimpleType
		a, b        string
		wantSame    bool
		wantDecided bool
	}{
		{"the same lexical form", prim, prim, "1", "1", true, true},
		{"two lexical forms of one value", prim, prim, "1", "01", true, true},
		{"whiteSpace normalization precedes the mapping", prim, prim, " 1 ", "1", true, true},
		{"genuinely different values", prim, prim, "1", "2", false, true},
		{"a derived type shares its base's governing mapping", derived, prim, "01", "1", true, true},
		{"an unmappable lexical is undecided, never a mismatch", prim, prim, "zzz", "1", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			same, decided := vs.Identical(tc.ta, vsFixed(tc.a), tc.tb, vsFixed(tc.b))
			if same != tc.wantSame || decided != tc.wantDecided {
				t.Errorf("Identical = (%t, %t), want (%t, %t)", same, decided, tc.wantSame, tc.wantDecided)
			}
			same, decided = vs.EqualOrIdentical(tc.ta, vsFixed(tc.a), tc.tb, vsFixed(tc.b))
			if same != tc.wantSame || decided != tc.wantDecided {
				t.Errorf("EqualOrIdentical = (%t, %t), want (%t, %t)", same, decided, tc.wantSame, tc.wantDecided)
			}
		})
	}
}

// TestValueSpaceRefusesIncommensurableSpaces pins the correctness-critical
// refusal: two types governed by DIFFERENT mappings hold values §2.2.1 makes
// "artificially distinct", so no comparison is made at all. Deciding here would
// risk the one outcome the fail-open contract forbids — a spurious NOT-same,
// which becomes a false schema rejection.
func TestValueSpaceRefusesIncommensurableSpaces(t *testing.T) {
	intPrim := vsPrim(t, "int")
	otherPrim := vsPrim(t, "other")
	b := twoTypeBackend{a: intPrim.Name(), b: otherPrim.Name()}
	vs := NewValueSpace(b)

	// Both mappings parse "1" to the same Go value, so a naive comparison would
	// happily report "the same" — the refusal must be structural, not value-based.
	if same, decided := vs.Identical(intPrim, vsFixed("1"), otherPrim, vsFixed("1")); decided {
		t.Errorf("Identical across two governing mappings = (%t, %t), want undecided", same, decided)
	}
	if same, decided := vs.EqualOrIdentical(intPrim, vsFixed("1"), otherPrim, vsFixed("1")); decided {
		t.Errorf("EqualOrIdentical across two governing mappings = (%t, %t), want undecided", same, decided)
	}
}

// TestValueSpaceRefusesUngovernedAndNonAtomic pins the remaining fail-open
// gates: a type no mapping governs, and the list and union varieties, whose
// governing mappings are synthesized per type.
func TestValueSpaceRefusesUngovernedAndNonAtomic(t *testing.T) {
	prim := vsPrim(t, "int")
	vs := NewValueSpace(intBackend{mapped: prim.Name()})

	unmapped := vsPrim(t, "unmapped")
	lst, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "lst"},
		xsd.NewList(prim), xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list): %v", err)
	}
	uni, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "uni"},
		xsd.NewUnion(prim), xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union): %v", err)
	}

	for _, tc := range []struct {
		name string
		ta   *xsd.SimpleType
	}{
		{"a type no backend mapping governs", unmapped},
		{"the list variety", lst},
		{"the union variety", uni},
		{"xs:anySimpleType, which has no variety at all", xsd.AnySimpleType()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, decided := vs.Identical(tc.ta, vsFixed("1"), prim, vsFixed("1")); decided {
				t.Error("Identical decided, want undecided")
			}
			if _, decided := vs.EqualOrIdentical(prim, vsFixed("1"), tc.ta, vsFixed("1")); decided {
				t.Error("EqualOrIdentical decided, want undecided")
			}
		})
	}
}

// TestValueSpaceRefusesQNameAndNOTATION pins the GAP(value) marker in
// sharedMapping: QName and NOTATION lexicals resolve a prefix against the
// bindings in scope where the literal was written (§3.3.18/§3.3.19), and
// xsd.ValueConstraint carries no such context, so no comparison is attempted —
// for a derivation of either primitive as much as for the primitive itself.
func TestValueSpaceRefusesQNameAndNOTATION(t *testing.T) {
	for _, local := range []string{"QName", "NOTATION"} {
		t.Run(local, func(t *testing.T) {
			prim := vsPrim(t, local)
			vs := NewValueSpace(intBackend{mapped: prim.Name()})
			derived := vsDerived(t, "d", prim)
			if _, decided := vs.Identical(prim, vsFixed("1"), prim, vsFixed("1")); decided {
				t.Errorf("Identical on xs:%s decided, want undecided", local)
			}
			if _, decided := vs.EqualOrIdentical(derived, vsFixed("1"), prim, vsFixed("1")); decided {
				t.Errorf("EqualOrIdentical on a derivation of xs:%s decided, want undecided", local)
			}
		})
	}
}

// TestIdentityIsNotTheEqualOrIdenticalUnion pins that the two relations are
// genuinely different where the datatype distinguishes them (§2.2.2's float ±0
// and dateTime timezone-offset cases): a value that is EQUAL but NOT IDENTICAL
// answers au-props-correct clause 3 with "not identical" and loc-testSubP
// clauses 4.2/5.2.2 with "the same".
func TestIdentityIsNotTheEqualOrIdenticalUnion(t *testing.T) {
	prim := vsPrim(t, "loose")
	vs := NewValueSpace(looseBackend{mapped: prim.Name()})

	same, decided := vs.Identical(prim, vsFixed("1"), prim, vsFixed("2"))
	if same || !decided {
		t.Errorf("Identical(equal-but-not-identical) = (%t, %t), want (false, true)", same, decided)
	}
	same, decided = vs.EqualOrIdentical(prim, vsFixed("1"), prim, vsFixed("2"))
	if !same || !decided {
		t.Errorf("EqualOrIdentical(equal-but-not-identical) = (%t, %t), want (true, true)", same, decided)
	}
}

// TestEqualOrIdenticalUndecidedWithoutCapabilities pins the decided=false arm
// both predicates share: a value carrying NEITHER relation cannot be compared,
// which is distinct from "compared and found different". enumMatch collapses the
// two deliberately (a facet's verdict is "no match" either way); the adapter must
// not.
func TestEqualOrIdenticalUndecidedWithoutCapabilities(t *testing.T) {
	type opaque struct{}
	if same, decided := equalOrIdentical(opaque{}, opaque{}); same || decided {
		t.Errorf("equalOrIdentical(no capabilities) = (%t, %t), want (false, false)", same, decided)
	}
	if same, decided := identical(opaque{}, opaque{}); same || decided {
		t.Errorf("identical(no capabilities) = (%t, %t), want (false, false)", same, decided)
	}
	if enumMatch(opaque{}, opaque{}) {
		t.Error("enumMatch(no capabilities) = true, want false (unchanged by the refactor)")
	}
}

// TestNewValueSpaceNilBackendPanics pins the nil guard, matching
// parser.WithBackend's.
func TestNewValueSpaceNilBackendPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("NewValueSpace(nil) must panic")
		}
	}()
	NewValueSpace(nil)
}

// twoTypeBackend maps two distinct type names to two DISTINCT Mappings that
// nonetheless produce comparable values — the shape that makes an
// incommensurable-space comparison look deceptively decidable.
type twoTypeBackend struct{ a, b xsd.QName }

func (t twoTypeBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if typ != t.a && typ != t.b {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, _ Context) (Value, error) {
		n, err := strconv.Atoi(lexical)
		if err != nil {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{}, "test backend: %q is not an integer", lexical)
		}
		return intValue(n), nil
	}}, true
}

// looseValue models the §2.2.2 datatypes whose equality is WIDER than their
// identity (float's ±0, dateTime across timezone offsets): every value is equal
// to every other, but identity is exact.
type looseValue int

func (l looseValue) Eq(Value) bool { return true }

func (l looseValue) Identical(other Value) bool {
	o, ok := other.(looseValue)
	return ok && o == l
}

// looseBackend maps one name to a mapping producing looseValues.
type looseBackend struct{ mapped xsd.QName }

func (b looseBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if typ != b.mapped {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, _ Context) (Value, error) {
		n, err := strconv.Atoi(lexical)
		if err != nil {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{}, "test backend: %q is not an integer", lexical)
		}
		return looseValue(n), nil
	}}, true
}
