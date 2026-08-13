package value

import (
	"strconv"
	"strings"
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
	st, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		xsd.RestrictionDerivation{}, base, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%s): %v", local, err)
	}
	return st
}

func vsFixed(lexical string) xsd.ValueConstraint {
	return xsd.NewValueConstraint(xsd.ValueFixed, lexical, nil, nil)
}

// vsFixedIn builds a fixed value constraint carrying the namespace context its
// {lexical form} was written in (§3.3.18): defaultNS is nil when no default
// namespace is in scope, and bindings are the prefixed ones.
func vsFixedIn(lexical string, defaultNS *string, bindings ...xsd.NamespaceBinding) xsd.ValueConstraint {
	return xsd.NewValueConstraint(xsd.ValueFixed, lexical, bindings, defaultNS)
}

func binding(prefix, namespace string) xsd.NamespaceBinding {
	return xsd.NewNamespaceBinding(prefix, namespace)
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
			same, decided := vs.Identical(noSchema{}, tc.ta, vsFixed(tc.a), tc.tb, vsFixed(tc.b))
			if same != tc.wantSame || decided != tc.wantDecided {
				t.Errorf("Identical = (%t, %t), want (%t, %t)", same, decided, tc.wantSame, tc.wantDecided)
			}
			same, decided = vs.EqualOrIdentical(noSchema{}, tc.ta, vsFixed(tc.a), tc.tb, vsFixed(tc.b))
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
	if same, decided := vs.Identical(noSchema{}, intPrim, vsFixed("1"), otherPrim, vsFixed("1")); decided {
		t.Errorf("Identical across two governing mappings = (%t, %t), want undecided", same, decided)
	}
	if same, decided := vs.EqualOrIdentical(noSchema{}, intPrim, vsFixed("1"), otherPrim, vsFixed("1")); decided {
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
	lst, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "lst"},
		listOf(prim), xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list): %v", err)
	}
	uni, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "uni"},
		unionOf(prim), xsd.AnySimpleType(), nil, nil)
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
			if _, decided := vs.Identical(noSchema{}, tc.ta, vsFixed("1"), prim, vsFixed("1")); decided {
				t.Error("Identical decided, want undecided")
			}
			if _, decided := vs.EqualOrIdentical(noSchema{}, prim, vsFixed("1"), tc.ta, vsFixed("1")); decided {
				t.Error("EqualOrIdentical decided, want undecided")
			}
		})
	}
}

// TestValueSpaceResolvesQNameAndNOTATION pins the comparison a lexical test
// gets wrong in BOTH directions: each side's {lexical form} is resolved under
// the namespace bindings ITS OWN value constraint captured (§3.3.18, adopted
// verbatim by §3.3.19's "as given for QName"), so two prefixes naming one
// namespace are the SAME {value} and one prefix bound differently across two
// documents is NOT — the verdict au-props-correct clause 3 and loc-testSubP
// clauses 4.2/5.2.2 turn into a rejection.
//
// Every case runs for NOTATION exactly as for QName, and the unprefixed cases
// pin the ONE default-namespace rule §3.3.19 inherits: there is no
// QName/NOTATION split.
func TestValueSpaceResolvesQNameAndNOTATION(t *testing.T) {
	urnA, urnB := "urn:a", "urn:b"
	for _, local := range []string{"QName", "NOTATION"} {
		t.Run(local, func(t *testing.T) {
			prim := vsPrim(t, local)
			vs := NewValueSpace(qnameBackend{mapped: prim.Name()})
			derived := vsDerived(t, "d", prim)

			for _, tc := range []struct {
				name        string
				ta, tb      *xsd.SimpleType
				a, b        xsd.ValueConstraint
				wantSame    bool
				wantDecided bool
			}{
				{
					"different prefixes bound to one namespace are the same value",
					prim, prim,
					vsFixedIn("a:x", nil, binding("a", urnA)),
					vsFixedIn("b:x", nil, binding("b", urnA)),
					true, true,
				},
				{
					"one prefix bound differently in two documents is not",
					prim, prim,
					vsFixedIn("p:x", nil, binding("p", urnA)),
					vsFixedIn("p:x", nil, binding("p", urnB)),
					false, true,
				},
				{
					"the same binding and a different local part is not",
					prim, prim,
					vsFixedIn("p:x", nil, binding("p", urnA)),
					vsFixedIn("p:y", nil, binding("p", urnA)),
					false, true,
				},
				{
					"an unprefixed literal binds to the default namespace in scope",
					prim, prim,
					vsFixedIn("x", &urnA),
					vsFixedIn("a:x", nil, binding("a", urnA)),
					true, true,
				},
				{
					"an unprefixed literal with no default namespace binds to none",
					prim, prim,
					vsFixedIn("x", nil),
					vsFixedIn("x", &urnA),
					false, true,
				},
				{
					"a derived type resolves under its own constraint's bindings too",
					derived, prim,
					vsFixedIn("d:x", nil, binding("d", urnA)),
					vsFixedIn("a:x", nil, binding("a", urnA)),
					true, true,
				},
				{
					"an unresolvable prefix stays undecided, never a mismatch",
					prim, prim,
					vsFixedIn("gone:x", nil),
					vsFixedIn("a:x", nil, binding("a", urnA)),
					false, false,
				},
				{
					"an unresolvable prefix on the second side too",
					prim, prim,
					vsFixedIn("a:x", nil, binding("a", urnA)),
					vsFixedIn("gone:x", nil),
					false, false,
				},
			} {
				t.Run(tc.name, func(t *testing.T) {
					same, decided := vs.Identical(noSchema{}, tc.ta, tc.a, tc.tb, tc.b)
					if same != tc.wantSame || decided != tc.wantDecided {
						t.Errorf("Identical = (%t, %t), want (%t, %t)", same, decided, tc.wantSame, tc.wantDecided)
					}
					same, decided = vs.EqualOrIdentical(noSchema{}, tc.ta, tc.a, tc.tb, tc.b)
					if same != tc.wantSame || decided != tc.wantDecided {
						t.Errorf("EqualOrIdentical = (%t, %t), want (%t, %t)", same, decided, tc.wantSame, tc.wantDecided)
					}
				})
			}
		})
	}
}

// TestValueSpaceResolvesReservedPrefixes pins the two reserved-prefix rules the
// shared nsContext holds (Namespaces in XML §3): "xml" resolves with no
// declaration anywhere in the captured context, and "xmlns" never resolves, so
// a constraint using it stays undecided rather than comparing as a mismatch.
func TestValueSpaceResolvesReservedPrefixes(t *testing.T) {
	prim := vsPrim(t, "QName")
	vs := NewValueSpace(qnameBackend{mapped: prim.Name()})
	declared := "http://www.w3.org/XML/1998/namespace"

	same, decided := vs.Identical(noSchema{}, prim, vsFixedIn("xml:lang", nil), prim,
		vsFixedIn("d:lang", nil, binding("d", declared)))
	if !same || !decided {
		t.Errorf("Identical(xml: against its declared namespace) = (%t, %t), want (true, true)", same, decided)
	}
	if same, decided := vs.Identical(noSchema{}, prim, vsFixedIn("xmlns:x", nil), prim, vsFixedIn("xmlns:x", nil)); decided {
		t.Errorf("Identical(xmlns: prefix) = (%t, %t), want undecided", same, decided)
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

	same, decided := vs.Identical(noSchema{}, prim, vsFixed("1"), prim, vsFixed("2"))
	if same || !decided {
		t.Errorf("Identical(equal-but-not-identical) = (%t, %t), want (false, true)", same, decided)
	}
	same, decided = vs.EqualOrIdentical(noSchema{}, prim, vsFixed("1"), prim, vsFixed("2"))
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

// qnameValue is the {namespace name, local part} tuple a QName or NOTATION
// lexical denotes once its prefix is resolved (§3.3.18). §2.2.2 names no
// equality exception for either type, so equality IS identity here and Eq alone
// answers both relations.
type qnameValue struct{ space, local string }

func (q qnameValue) Eq(other Value) bool {
	o, ok := other.(qnameValue)
	return ok && o == q
}

// qnameBackend maps one name to a context-dependent mapping: it splits the
// lexical at the colon and resolves the prefix through the [Context] it is
// handed, rejecting an unbound prefix as an *xsderr.Error exactly as a real
// backend would. A nil Context rejects everything, which is what makes a
// dropped context visible as undecided rather than as a silent verdict.
type qnameBackend struct{ mapped xsd.QName }

func (b qnameBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if typ != b.mapped {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, ctx Context) (Value, error) {
		if ctx == nil {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
				"test backend: a QName lexical needs a context")
		}
		prefix, local := "", lexical
		if i := strings.IndexByte(lexical, ':'); i >= 0 {
			prefix, local = lexical[:i], lexical[i+1:]
		}
		ns, ok := ctx.LookupNamespace(prefix)
		if !ok {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
				"test backend: prefix %q is not bound", prefix)
		}
		return qnameValue{space: ns, local: local}, nil
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
	st, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		listOf(item), xsd.AnySimpleType(),
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
	st, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		unionOf(members...), xsd.AnySimpleType(), nil, nil)
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
			valid, decided := vs.ValidDefault(noSchema{}, tc.t, vsFixed(tc.lexical))
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
			if valid, decided := vs.ValidDefault(noSchema{}, tc.t, vsFixed("zzz not a value of anything")); decided {
				t.Errorf("ValidDefault = (%t, %t), want undecided (fail-open)", valid, decided)
			}
		})
	}
}

// TestValidDefaultAcceptsContextDependentTypes pins needsContext, the recursive
// form of contextDependent: a QName or NOTATION lexical resolves a prefix against
// the bindings in scope where it was written (§3.3.18/§3.3.19, PRINCIPLES 19),
// which this one-sided check does not route to the mapping — the comparisons do,
// the list/union dispatch does not — so no verdict is available, for a list OF
// QName and a union WITH a QName member as much as for the primitive.
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
					if valid, decided := vs.ValidDefault(noSchema{}, tc.t, vsFixed("p:local")); decided {
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
		if valid, decided := vs.ValidDefault(noSchema{}, bad, vsFixed(lexical)); decided {
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
		if valid, decided := vs.ValidDefault(noSchema{}, st, vsFixed(lexical)); decided {
			t.Errorf("ValidDefault(%q) = (%t, %t), want undecided (gate 4, fail-open)", lexical, valid, decided)
		}
	}
}
