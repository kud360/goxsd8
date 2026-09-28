package validate

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The tests below pin that every content decline cvccomplexcontent.go makes is
// RECORDED as an [Unevaluated] and not only logged ([contentCheck.decline]): a
// decline that only logged would leave an empty Result reading as a walk that
// performed every check it reached. The undecided type is xs:decimal under
// gapBackend, a backend that does not map it, so String Valid over it is
// withheld rather than decided. xs:anySimpleType is not one: Datatype Valid
// holds for every literal against a ·special· datatype (Datatypes §4.1.4), so
// the walk decides it though no backend maps it (#1788).

// typeGap is the strict backend with one type unmapped: value.ValidateLexical
// then reports a fault of the TYPE for it and for every type derived from it,
// which is the decline the tests record. The schema is seeded under the strict
// backend, since builtin.Seed maps every primitive; only the assessment reads
// this one.
type typeGap struct {
	value.Backend
	gap xsd.QName
}

// gapBackend is the strict backend with gap unmapped.
func gapBackend(gap xsd.QName) value.Backend { return typeGap{Backend: testBackend(), gap: gap} }

func (b typeGap) Mapping(typ xsd.QName) (value.Mapping, bool) {
	if typ == b.gap {
		return value.Mapping{}, false
	}
	return b.Backend.Mapping(typ)
}

// assessRecorded assesses root against schema and returns what the walk charged
// and what it recorded as unevaluated.
func assessRecorded(t *testing.T, schema *xsd.Schema, root Element) ([]*xsderr.Error, []Unevaluated) {
	t.Helper()
	return assessRecordedWith(t, testBackend(), schema, root)
}

// assessRecordedWith is assessRecorded under backend.
func assessRecordedWith(t *testing.T, backend value.Backend, schema *xsd.Schema, root Element) ([]*xsderr.Error, []Unevaluated) {
	t.Helper()
	v, err := New(schema, backend)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	res := v.Assess(root)
	if res.Err() != nil {
		t.Fatalf("Err() = %v, want nil", res.Err())
	}
	return res.Violations(), res.Unevaluated()
}

// wantDeclines fails unless got holds exactly one record per want entry, in
// order, each carrying the entry's rule at its Loc and naming its clause.
func wantDeclines(t *testing.T, got []Unevaluated, want ...Unevaluated) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("Unevaluated() = %v, want %d record(s)", messages(got), len(want))
	}
	for i, w := range want {
		if got[i].Rule() != w.rule {
			t.Errorf("Unevaluated()[%d].Rule() = %q, want %q", i, got[i].Rule(), w.rule)
		}
		if got[i].Loc() != w.loc {
			t.Errorf("Unevaluated()[%d].Loc() = %s, want %s", i, got[i].Loc(), w.loc)
		}
		if !strings.Contains(got[i].Msg(), w.msg) {
			t.Errorf("Unevaluated()[%d].Msg() = %q, want it to name %q", i, got[i].Msg(), w.msg)
		}
	}
}

// cvc-type clause 3.1.3 withheld on a SIMPLE ·governing type definition· is
// recorded at the element, under cvc-type and not cvc-simple-type or
// cvc-datatype-valid: the record names the rule the element would have been
// charged under. The xs:integer control shows the record is the decline's and
// not every simple-typed element's.
func TestWithheldSimpleTypeValueIsRecorded(t *testing.T) {
	schema := simpleTypedSchema(t, icBuiltin("decimal"), nil, false)
	got, undecided := assessRecordedWith(t, gapBackend(icBuiltin("decimal")), schema, cRoot("#1.5"))
	wantSilence(t, got, "a withheld String Valid verdict charges nothing")
	wantDeclines(t, undecided, Unevaluated{rule: ruleCvcType, loc: loc(1, 1), msg: "cvc-type clause 3.1.3"})

	got, undecided = assessRecorded(t, schema, cRoot("#1.5"))
	wantSilence(t, got, "1.5 is an xs:decimal")
	wantDeclines(t, undecided)
}

// A ·special· ·governing type definition· is DECIDED: Datatype Valid holds for
// every literal against it (Datatypes §4.1.4, cvc-datatype-valid), so cvc-type
// clause 3.1.3 is satisfied with neither a charge nor a record, though no
// backend maps either type.
func TestSpecialSimpleTypeValueIsDecided(t *testing.T) {
	for _, typ := range []string{"anySimpleType", "anyAtomicType"} {
		got, undecided := assessRecorded(t, simpleTypedSchema(t, icBuiltin(typ), nil, false), cRoot("#x"))
		wantSilence(t, got, "Datatype Valid holds for every literal against xs:"+typ)
		wantDeclines(t, undecided)
	}
}

// cvc-complex-type clause 1.2's ·initial value· half, withheld over a simple
// {content type}, is recorded at the CONTAINING element.
func TestWithheldSimpleContentValueIsRecorded(t *testing.T) {
	got, undecided := assessRecordedWith(t, gapBackend(icBuiltin("decimal")), simpleContentSchema(t, icBuiltin("decimal")), cRoot("#1.5"))
	wantSilence(t, got, "a withheld String Valid verdict charges nothing")
	wantDeclines(t, undecided, Unevaluated{rule: ruleCvcComplexType, loc: loc(1, 1), msg: "cvc-complex-type clause 1.2"})
}

// An undecided comparison against a fixed {value constraint} (cvc-elt clause
// 5.2.2.2.2) is recorded after the cvc-type clause 3.1.3 record the same
// undecided type makes: a decline sets no charge, so it silences no later
// clause, and each withheld clause is its own record. Over xs:anySimpleType the
// comparison alone is undecided — its value space has no lexical mapping that
// is a function (Datatypes §3.2.1.2) — while clause 3.1.3 is decided.
func TestUndecidedFixedValueComparisonIsRecorded(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "1.5", nil, nil)
	got, undecided := assessRecordedWith(t, gapBackend(icBuiltin("decimal")),
		simpleTypedSchema(t, icBuiltin("decimal"), &fixed, false), cRoot("#1.5"))
	wantSilence(t, got, "an undecided comparison charges nothing")
	wantDeclines(t, undecided,
		Unevaluated{rule: ruleCvcType, loc: loc(1, 1), msg: "cvc-type clause 3.1.3"},
		Unevaluated{rule: ruleCvcElt, loc: loc(1, 1), msg: "cvc-elt clause 5.2.2.2.2"})

	fixed = xsd.NewValueConstraint(xsd.ValueFixed, "x", nil, nil)
	got, undecided = assessRecorded(t, simpleTypedSchema(t, icBuiltin("anySimpleType"), &fixed, false), cRoot("#x"))
	wantSilence(t, got, "an undecided comparison charges nothing")
	wantDeclines(t, undecided, Unevaluated{rule: ruleCvcElt, loc: loc(1, 1), msg: "cvc-elt clause 5.2.2.2.2"})
}

// A {content type} xsd.Schema.ContentMatcher declines leaves each element
// [[child]] unmatched, and each is recorded under cvc-complex-content at the
// CHILD's own location — where a charge against it would have sat. The model
// is two repeating sibling groups past the matcher's region ceiling
// (xsd.maxPartitionStates), the shape its GAP(xsd) decline names.
func TestUndecidedContentModelIsRecordedPerChild(t *testing.T) {
	group := func(max int, leaf string) xsd.Particle {
		t.Helper()
		unbounded, err := xsd.NewUnboundedOccurs(xsderr.Loc{}, 1)
		if err != nil {
			t.Fatalf("NewUnboundedOccurs: %v", err)
		}
		p, err := xsd.NewParticle(xsderr.Loc{}, unbounded, xsd.ResolvedTerm{Term: cLocal(t, leaf)})
		if err != nil {
			t.Fatalf("NewParticle: %v", err)
		}
		g, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{p})
		if err != nil {
			t.Fatalf("NewModelGroup: %v", err)
		}
		o, err := xsd.NewOccurs(xsderr.Loc{}, 1, max)
		if err != nil {
			t.Fatalf("NewOccurs: %v", err)
		}
		gp, err := xsd.NewParticle(xsderr.Loc{}, o, xsd.ResolvedTerm{Term: g})
		if err != nil {
			t.Fatalf("NewParticle: %v", err)
		}
		return gp
	}
	schema := cSchema(t, cSequence(t, false, group(1000, "a"), cParticle(t, "b", 1, 1), group(1000, "c")))

	got, undecided := assessRecorded(t, schema, cRoot("a", "b"))
	wantSilence(t, got, "an unmatched child charges nothing")
	wantDeclines(t, undecided,
		Unevaluated{rule: ruleCvcComplexContent, loc: loc(2, 1), msg: "cvc-complex-content clause 1"},
		Unevaluated{rule: ruleCvcComplexContent, loc: loc(3, 1), msg: "cvc-complex-content clause 1"})
}
