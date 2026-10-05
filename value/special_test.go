package value_test

import (
	"errors"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// prefixes is a test [value.Context]: the instance side's namespace bindings.
type prefixes map[string]string

func (p prefixes) LookupNamespace(prefix string) (string, bool) {
	ns, ok := p[prefix]
	return ns, ok
}

// stringAs is a backend mapping only xs:string, layered over strict with
// [value.Override] to break one member of a ·special· type's mapping union.
type stringAs struct {
	parse func(string, value.Context) (value.Value, error)
}

func (s stringAs) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ != (xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}) {
		return value.Mapping{}, false
	}
	return value.Mapping{Parse: s.parse}, true
}

// declineEvery is a non-nil [value.AssertionEvaluator] that declines every
// {test}: ConstraintMatches requires one, and a ·special· type, which has no
// facets, never hands it any.
type declineEvery struct{}

func (declineEvery) Evaluate(value.Backend, xsd.TypeResolver, *xsd.SimpleType, xsd.XPathExpression, value.Value) value.AssertionOutcome {
	return value.AssertionDeclined
}

// TestConstraintMatchesSpecialTypesUnderStrict pins the ·special· branch
// against a backend that maps every primitive (#2040): NOT-same exactly where
// every member of the mapping union — each primitive, and for xs:anySimpleType
// each list type — is decided and none maps both literals to equal or identical
// values (Datatypes §3.2.1.2, §3.2.2.2; cvc-attribute clause 4, cvc-elt clause
// 5.2.2.2.2). Literals are tested raw (Structures §3.1).
func TestConstraintMatchesSpecialTypesUnderStrict(t *testing.T) {
	b := strict.New()
	ast, aat := xsd.AnySimpleType(), xsd.AnyAtomicType()
	inst := prefixes{"p": "urn:one"}
	fixed := func(lexical string) xsd.ValueConstraint {
		return xsd.NewValueConstraint(xsd.ValueFixed, lexical, []xsd.NamespaceBinding{xsd.NewNamespaceBinding("q", "urn:one")}, nil)
	}

	for _, tc := range []struct {
		name     string
		t        *xsd.SimpleType
		lexical  string
		ctx      value.Context
		fixed    string
		wantSame bool
		decided  bool
	}{
		{"no primitive maps 37 and 36 to one value", aat, "37", inst, "36", false, true},
		{"anySimpleType too", ast, "37", inst, "36", false, true},
		{"no QName context leaves the union undecided", aat, "37", nil, "36", false, false},
		{"decimal equates 1 and 1.0", aat, "1.0", inst, "1", true, true},
		{"boolean equates 1 and true", aat, "true", inst, "1", true, true},
		{"equal but not identical is the same under cvc-attribute clause 4", aat, "2000-01-01T13:00:00+01:00", inst, "2000-01-01T12:00:00Z", true, true},
		{"QName equates two prefixes of one namespace", aat, "p:x", inst, "q:x", true, true},
		{"a single token against a different literal", ast, "abc", inst, "123", false, true},
		{"the empty literal against one token", ast, "", inst, ".", false, true},
		{"padding is no list literal and no member strips it", ast, "\n\t\ttest information\n\t", inst, "test information", false, true},
		{"a list type equates two-item literals position by position", ast, "01 2.0", inst, "1 2", true, true},
		{"no list member under anyAtomicType", aat, "01 2.0", inst, "1 2", false, true},
		{"lists of different lengths", ast, "1 2", inst, "1 2 3", false, true},
		{"a doubled #x20 is no list literal", ast, "01  2", inst, "1  2", false, true},
	} {
		t.Run(tc.t.Name().Local+"/"+tc.name, func(t *testing.T) {
			same, decided := value.ConstraintMatches(b, nil, tc.t, tc.lexical, tc.ctx, fixed(tc.fixed), declineEvery{})
			if same != tc.wantSame || decided != tc.decided {
				t.Errorf("ConstraintMatches(%q, %q) = (%t, %t), want (%t, %t)",
					tc.lexical, tc.fixed, same, decided, tc.wantSame, tc.decided)
			}
		})
	}
}

// TestConstraintMatchesSpecialMemberFaultsAreUndecided pins the fail-open half:
// one member that cannot answer — a Parse error that is no *xsderr.Error, or
// values carrying neither value.Identical nor value.Eq — leaves a pair every
// other member decides not-same undecided.
func TestConstraintMatchesSpecialMemberFaultsAreUndecided(t *testing.T) {
	inst := prefixes{}
	for _, tc := range []struct {
		name  string
		parse func(string, value.Context) (value.Value, error)
	}{
		{"a backend fault", func(string, value.Context) (value.Value, error) { return nil, errors.New("backend fault") }},
		{"values with no sameness relation", func(lexical string, _ value.Context) (value.Value, error) { return struct{ s string }{lexical}, nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := value.Override(strict.New(), stringAs{parse: tc.parse})
			vc := xsd.NewValueConstraint(xsd.ValueFixed, "36", nil, nil)
			if same, decided := value.ConstraintMatches(b, nil, xsd.AnyAtomicType(), "37", inst, vc, declineEvery{}); decided {
				t.Errorf("ConstraintMatches = (%t, %t), want undecided", same, decided)
			}
		})
	}
}

// TestValueSpaceComparesSpecialPairsOverTheUnion pins valueSpace.compare's
// ·special· branch (#1379, taking #2040's RULING to the schema seam): two
// ·special· types are decided over their mapping union, under identity for
// Identical (au-props-correct clause 3) and under equal-or-identical for
// EqualOrIdentical (loc-testSubP clauses 4.2 and 5.2.2). Each side's {lexical
// form} is parsed under the bindings its OWN value constraint captured
// (§3.3.18), and the list members are scanned only when both types are
// xs:anySimpleType (Datatypes §3.2.2.2).
func TestValueSpaceComparesSpecialPairsOverTheUnion(t *testing.T) {
	vs := value.NewValueSpace(strict.New())
	ast, aat := xsd.AnySimpleType(), xsd.AnyAtomicType()
	decimal, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType(decimal): %v", err)
	}
	in := func(lexical, prefix, namespace string) xsd.ValueConstraint {
		return xsd.NewValueConstraint(xsd.ValueFixed, lexical, []xsd.NamespaceBinding{xsd.NewNamespaceBinding(prefix, namespace)}, nil)
	}
	plain := func(lexical string) xsd.ValueConstraint { return in(lexical, "z", "urn:z") }

	type verdict struct{ same, decided bool }
	for _, tc := range []struct {
		name             string
		ta               *xsd.SimpleType
		a                xsd.ValueConstraint
		tb               *xsd.SimpleType
		b                xsd.ValueConstraint
		identical, eqOrI verdict
	}{
		{"no member equates 123 and abc (addB108, attO025)", aat, plain("123"), ast, plain("abc"),
			verdict{false, true}, verdict{false, true}},
		{"decimal makes 1 and 1.0 identical", ast, plain("1"), ast, plain("1.0"),
			verdict{true, true}, verdict{true, true}},
		{"one instant at two offsets is equal but not identical", ast, plain("2000-01-01T12:00:00Z"), aat, plain("2000-01-01T13:00:00+01:00"),
			verdict{false, true}, verdict{true, true}},
		{"decimal makes float's 0 and -0 identical", aat, plain("0"), aat, plain("-0"),
			verdict{true, true}, verdict{true, true}},
		{"a special side against an ordinary one stays undecided", ast, plain("1"), decimal, plain("1"),
			verdict{false, false}, verdict{false, false}},
		{"the ordinary side first, too", decimal, plain("1"), aat, plain("1"),
			verdict{false, false}, verdict{false, false}},
		{"anyAtomicType against anySimpleType scans no list member", aat, plain("01 2.0"), ast, plain("1 2"),
			verdict{false, true}, verdict{false, true}},
		{"anySimpleType against anyAtomicType scans none either", ast, plain("01 2.0"), aat, plain("1 2"),
			verdict{false, true}, verdict{false, true}},
		{"two anySimpleTypes scan the list members", ast, plain("01 2.0"), ast, plain("1 2"),
			verdict{true, true}, verdict{true, true}},
		{"a list member compares each position under the same relation", ast, plain("2000-01-01T12:00:00Z 1"), ast, plain("2000-01-01T13:00:00+01:00 1"),
			verdict{false, true}, verdict{true, true}},
		{"two prefixes naming one namespace, each under its own bindings", aat, in("p:x", "p", "urn:one"), aat, in("q:x", "q", "urn:one"),
			verdict{true, true}, verdict{true, true}},
		{"two prefixes naming two namespaces", aat, in("p:x", "p", "urn:one"), aat, in("q:x", "q", "urn:two"),
			verdict{false, true}, verdict{false, true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			same, decided := vs.Identical(nil, tc.ta, tc.a, tc.tb, tc.b)
			if (verdict{same, decided}) != tc.identical {
				t.Errorf("Identical(%q, %q) = (%t, %t), want %+v", tc.a.LexicalForm(), tc.b.LexicalForm(), same, decided, tc.identical)
			}
			same, decided = vs.EqualOrIdentical(nil, tc.ta, tc.a, tc.tb, tc.b)
			if (verdict{same, decided}) != tc.eqOrI {
				t.Errorf("EqualOrIdentical(%q, %q) = (%t, %t), want %+v", tc.a.LexicalForm(), tc.b.LexicalForm(), same, decided, tc.eqOrI)
			}
		})
	}
}
