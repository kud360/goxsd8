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
		xsd.NewList(prim), xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
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

// vsList builds a list-variety type over item, carrying the collapse whiteSpace
// facet the list stage needs in force (§4.3.6, effectiveWhiteSpace).
func vsList(t *testing.T, local string, item *xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		xsd.NewList(item), xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list %s): %v", local, err)
	}
	return st
}

// vsUnion builds a union-variety type over members. A union carries no whiteSpace
// facet: it is categorically not applicable (cos-applicable-facets §4.1.5).
func vsUnion(t *testing.T, local string, members ...*xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		xsd.NewUnion(members...), xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union %s): %v", local, err)
	}
	return st
}

// TestValidDefaultDecidesGovernedTypes pins the verdict half of
// cos-valid-simple-default (§3.2.6.2): for a type the backend governs, a lexical
// the mapping accepts is a valid default and one it rejects is NOT — the only
// outcome that may answer decided=true.
//
// The list and union rows are the difference from Identical/EqualOrIdentical,
// which refuse both varieties outright: this predicate needs ONE type's mapping,
// not a shared one, so Datatype Valid clauses 2.2 and 2.3 are decided here.
func TestValidDefaultDecidesGovernedTypes(t *testing.T) {
	prim := vsPrim(t, "int")
	vs := NewValueSpace(intBackend{mapped: prim.Name()})
	derived := vsDerived(t, "narrow", prim)

	for _, tc := range []struct {
		name    string
		t       *xsd.SimpleType
		lexical string
		want    bool
	}{
		{"an atomic lexical the mapping accepts", prim, "1", true},
		{"an atomic lexical the mapping rejects", prim, "zzz", false},
		{"whiteSpace normalization precedes the mapping", prim, " 1 ", true},
		{"a derived type is decided by its base's mapping", derived, "01", true},
		{"a derived type rejects too", derived, "zzz", false},
		{"a list whose every item maps", vsList(t, "ints", prim), "1 2 3", true},
		{"a list with one item that does not", vsList(t, "ints", prim), "1 zzz", false},
		{"a union some member accepts", vsUnion(t, "u", prim), "1", true},
		{"a union no member accepts", vsUnion(t, "u", prim), "zzz", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			valid, decided := vs.ValidDefault(tc.t, vsFixed(tc.lexical))
			if !decided {
				t.Fatalf("ValidDefault = (%t, false), want a decided verdict", valid)
			}
			if valid != tc.want {
				t.Errorf("ValidDefault = (%t, true), want (%t, true)", valid, tc.want)
			}
		})
	}
}

// TestValidDefaultAcceptsUngovernedTypes is the regression test the whole gate
// exists for: xs:anySimpleType and xs:anyAtomicType are the ·special· datatypes
// (Datatypes §4.1), for which Datatype Valid is UNCONDITIONALLY true, and
// §3.2.2.2's third tier types every <xs:attribute> with no @type as
// xs:anySimpleType. No backend maps them, so ValidateLexical reports "no backend
// mapping governs type" — a BACKEND gap, not a verdict. Reading that as a
// rejection would false-reject every typeless attribute default there is.
//
// The lexicals are deliberately ones the governed mapping would reject, so a row
// cannot pass by the literal happening to be valid.
func TestValidDefaultAcceptsUngovernedTypes(t *testing.T) {
	prim := vsPrim(t, "int")
	vs := NewValueSpace(intBackend{mapped: prim.Name()})
	unmapped := vsPrim(t, "unmapped")

	for _, tc := range []struct {
		name string
		t    *xsd.SimpleType
	}{
		{"xs:anySimpleType, the ·special· datatype a typeless attribute gets", xsd.AnySimpleType()},
		{"xs:anyAtomicType, the other ·special· datatype", xsd.AnyAtomicType()},
		{"a named type no backend mapping governs", unmapped},
		{"a list whose ITEM type is ungoverned", vsList(t, "l", unmapped)},
		{"a union with one ungoverned MEMBER", vsUnion(t, "u", prim, unmapped)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if valid, decided := vs.ValidDefault(tc.t, vsFixed("zzz not a value of anything")); decided {
				t.Errorf("ValidDefault = (%t, %t), want undecided (fail-open)", valid, decided)
			}
		})
	}
}

// TestValidDefaultAcceptsContextDependentTypes pins needsContext, the recursive
// form of contextDependent: a QName or NOTATION lexical resolves a prefix against
// the bindings in scope where it was written (§3.3.18/§3.3.19, PRINCIPLES 19), and
// xsd.ValueConstraint carries no such context, so no verdict is available — for a
// list OF QName and a union WITH a QName member as much as for the primitive.
//
// The backend here DOES map QName, so the ungoverned gate cannot be what accepts
// these: only needsContext can. Without it every row would be a false reject,
// since the mapping is handed a nil Context.
func TestValidDefaultAcceptsContextDependentTypes(t *testing.T) {
	for _, local := range []string{"QName", "NOTATION"} {
		t.Run(local, func(t *testing.T) {
			prim := vsPrim(t, local)
			vs := NewValueSpace(intBackend{mapped: prim.Name()})
			derived := vsDerived(t, "d", prim)

			for _, tc := range []struct {
				name string
				t    *xsd.SimpleType
			}{
				{"the primitive itself", prim},
				{"a restriction of it", derived},
				{"a list of it", vsList(t, "l", prim)},
				{"a union with it as a member", vsUnion(t, "u", derived)},
				{"a list of a union of it", vsList(t, "lu", vsUnion(t, "u2", prim))},
			} {
				t.Run(tc.name, func(t *testing.T) {
					if valid, decided := vs.ValidDefault(tc.t, vsFixed("p:local")); decided {
						t.Errorf("ValidDefault = (%t, %t), want undecided (fail-open)", valid, decided)
					}
				})
			}
		})
	}
}

// TestValidDefaultAcceptsConstructionStageFailures pins the compile gate: a
// pattern facet the regex translator cannot express (src-pattern-value) is a
// statement about the TYPE, not about the value constraint's lexical form.
// Charging it as a-props-correct / au-props-correct clause 2 would reject a
// schema for an unrelated facet, under the wrong rule and against the wrong
// component — so it is undecided, and a lexical the mapping would happily accept
// stays undecided too, which is what distinguishes this from a verdict.
func TestValidDefaultAcceptsConstructionStageFailures(t *testing.T) {
	qn := xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}
	bad, err := xsd.NewPrimitiveType(xsderr.Loc{}, qn, []xsd.Facet{
		xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true),
		xsd.NewFacet(xsd.FacetPattern, []string{"["}, false), // unclosed character class
	}, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	vs := NewValueSpace(intBackend{mapped: qn})

	for _, lexical := range []string{"1", "zzz"} {
		if valid, decided := vs.ValidDefault(bad, vsFixed(lexical)); decided {
			t.Errorf("ValidDefault(%q) = (%t, %t), want undecided (fail-open)", lexical, valid, decided)
		}
	}
}

// TestValidDefaultAcceptsFacetPreconditionFaults pins GATE 4: a type carrying a facet
// that is not applicable to its value space (cos-applicable-facets §4.1.5) is a fault
// in the TYPE, so every literal is undecided — never decided-INVALID.
//
// Charging it would be a false schema rejection with teeth: checkSimpleDefault
// (xsd/valueconstraintvalid.go) turns decided-invalid into an a-props-correct /
// au-props-correct clause 2 rejection, and since the fault repeats for every literal,
// no default whatsoever could have satisfied the clause. Gate 4 rather than gates 1–3
// is what answers, because the pairing only becomes observable once a facet meets a
// parsed value — which is why preconditionType uses a length facet (whose {value} is a
// plain count, so compile() and gate 3 both pass) rather than a bound facet.
func TestValidDefaultAcceptsFacetPreconditionFaults(t *testing.T) {
	st := preconditionType(t, "unmeasurable")
	vs := NewValueSpace(plainBackend{st.Name(): true})

	// "ab" has length 2 and so would SATISFY the length facet if the value space were
	// Lengthed at all: the undecided answer is the fault's, not a smuggled rejection.
	for _, lexical := range []string{"ab", "abcd"} {
		if valid, decided := vs.ValidDefault(st, vsFixed(lexical)); decided {
			t.Errorf("ValidDefault(%q) = (%t, %t), want undecided (gate 4, fail-open)", lexical, valid, decided)
		}
	}
}
