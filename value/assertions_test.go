package value

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// scriptedAssertions is an [AssertionEvaluator] answering each {test} by its
// {expression} text, and recording every call, so a test pins both the outcome
// the pipeline returns and what the evaluator was handed. A text it holds no
// answer for is declined, the safe default.
type scriptedAssertions struct {
	answers map[string]AssertionOutcome
	calls   *[]assertionCall
}

// assertionCall is one Evaluate call: the type the facet was effective on, the
// {test}'s {expression} and the value.
type assertionCall struct {
	st   *xsd.SimpleType
	test string
	v    Value
}

func (s scriptedAssertions) Evaluate(_ Backend, _ xsd.TypeResolver, st *xsd.SimpleType, test xsd.XPathExpression, v Value) AssertionOutcome {
	if s.calls != nil {
		*s.calls = append(*s.calls, assertionCall{st: st, test: test.Expression(), v: v})
	}
	return s.answers[test.Expression()]
}

// scripted is the evaluator answering hold, fail and decline as named.
func scripted(calls *[]assertionCall) scriptedAssertions {
	return scriptedAssertions{
		answers: map[string]AssertionOutcome{"hold": AssertionHolds, "fail": AssertionFails, "decline": AssertionDeclined},
		calls:   calls,
	}
}

// asserting builds a restriction of base named local whose own assertions
// facet carries one assertion per test, in order.
func asserting(t *testing.T, local string, base *xsd.SimpleType, tests ...string) *xsd.SimpleType {
	t.Helper()
	assertions := make([]xsd.Assertion, 0, len(tests))
	for _, test := range tests {
		assertions = append(assertions, xsd.NewAssertion(xsd.NewXPathExpression(test, nil, nil, nil)))
	}
	st, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: local},
		xsd.RestrictionDerivation{}, base, []xsd.Facet{xsd.NewAssertionsFacet(assertions)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(%q): %v", local, err)
	}
	return st
}

// TestAssertionsStageOutcomes pins the assertions stage's three outcomes
// (cvc-assertions-valid, Datatypes §4.3.13.3, folded into Datatype Valid by
// cvc-datatype-valid clause 3): a held {test} accepts, a failed one is a
// VERDICT under cvc-assertions-valid, and a declined one is the non-verdict
// IsAssertionDeclined reports. A failure anywhere in the {value} is the
// verdict even after a decline, since each assertion is a conjunct.
func TestAssertionsStageOutcomes(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	b := memberBackend{num.Name(): allDigits}
	for _, tc := range []struct {
		name               string
		tests              []string
		accept, verdict    bool
		declined           bool
		wantCalls          int
		wantRuleOnFailures xsderr.Rule
	}{
		{name: "held", tests: []string{"hold"}, accept: true, wantCalls: 1},
		{name: "failed", tests: []string{"fail"}, verdict: true, wantCalls: 1, wantRuleOnFailures: ruleCvcAssertionsValid},
		{name: "declined", tests: []string{"decline"}, declined: true, wantCalls: 1, wantRuleOnFailures: ruleCvcAssertionsValid},
		{name: "a failure after a decline", tests: []string{"decline", "fail"}, verdict: true, wantCalls: 2, wantRuleOnFailures: ruleCvcAssertionsValid},
		{name: "a decline after a hold", tests: []string{"hold", "decline"}, declined: true, wantCalls: 2, wantRuleOnFailures: ruleCvcAssertionsValid},
	} {
		var calls []assertionCall
		st := asserting(t, "asserted", num, tc.tests...)
		v, err := ValidateLexical(b, noSchema{}, st, " 7 ", nil, scripted(&calls))
		if tc.accept != (err == nil) {
			t.Errorf("%s: ValidateLexical err = %v, want accept %v", tc.name, err, tc.accept)
		}
		if got := IsDatatypeVerdict(err); got != tc.verdict {
			t.Errorf("%s: IsDatatypeVerdict = %v, want %v (err %v)", tc.name, got, tc.verdict, err)
		}
		if got := IsAssertionDeclined(err); got != tc.declined {
			t.Errorf("%s: IsAssertionDeclined = %v, want %v (err %v)", tc.name, got, tc.declined, err)
		}
		if err != nil && ruleOf(err) != tc.wantRuleOnFailures {
			t.Errorf("%s: rule = %s, want %s", tc.name, ruleOf(err), tc.wantRuleOnFailures)
		}
		if len(calls) != tc.wantCalls {
			t.Fatalf("%s: %d Evaluate calls, want %d", tc.name, len(calls), tc.wantCalls)
		}
		want := unionMemberVal{member: "numeric", lexical: "7"}
		if calls[0].st != st || calls[0].v != Value(want) {
			t.Errorf("%s: Evaluate got (%v, %#v), want the asserting type and the parsed, normalized value %#v", tc.name, calls[0].st.Name(), calls[0].v, want)
		}
		if tc.accept && v != Value(want) {
			t.Errorf("%s: value = %#v, want %#v", tc.name, v, want)
		}
	}
}

// TestAssertionsStageRunsLast pins the stage's position: a literal an earlier
// stage rejects never reaches the evaluator, so a decline there cannot mask the
// rejection.
func TestAssertionsStageRunsLast(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	b := memberBackend{num.Name(): allDigits}
	var calls []assertionCall
	_, err := ValidateLexical(b, noSchema{}, asserting(t, "asserted", num, "decline"), "abc", nil, scripted(&calls))
	if !IsDatatypeVerdict(err) {
		t.Fatalf("ValidateLexical(unmappable literal) = %v, want the mapping's verdict", err)
	}
	if len(calls) != 0 {
		t.Errorf("%d Evaluate calls for a literal the mapping rejected, want 0", len(calls))
	}
}

// TestUnionMemberFailingItsAssertionsYields pins dt-active-member over the
// assertions stage: a member whose assertions facet FAILS is not Datatype Valid,
// so the next member that accepts is the ·active member type· and supplies the
// value — through ValidateLexical and ValidatingType alike — while a member
// whose {test} is DECLINED stops the dispatch, so no later member is chosen on
// a verdict nobody decided.
func TestUnionMemberFailingItsAssertionsYields(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text", "preserve")
	b := memberBackend{num.Name(): allDigits, text.Name(): func(string) bool { return true }}
	for _, tc := range []struct {
		test      string
		wantFirst bool
		declined  bool
	}{
		{test: "hold", wantFirst: true},
		{test: "fail", wantFirst: false},
		{test: "decline", declined: true},
	} {
		first := asserting(t, "first", num, tc.test)
		u := unionType2(t, "u", first, text)
		v, err := ValidateLexical(b, noSchema{}, u, "7", nil, scripted(nil))
		typ, _, terr := ValidatingType(b, noSchema{}, u, "7", nil, scripted(nil))
		if tc.declined {
			if !IsAssertionDeclined(err) || !IsAssertionDeclined(terr) {
				t.Errorf("%s: ValidateLexical err %v, ValidatingType err %v, want both the assertions decline", tc.test, err, terr)
			}
			continue
		}
		if err != nil || terr != nil {
			t.Fatalf("%s: ValidateLexical err %v, ValidatingType err %v, want accept", tc.test, err, terr)
		}
		wantType, wantTag := text, "text"
		if tc.wantFirst {
			wantType, wantTag = first, "numeric"
		}
		if typ != wantType {
			t.Errorf("%s: ValidatingType = %s, want %s", tc.test, typ.Name(), wantType.Name())
		}
		if got := v.(unionMemberVal).member; got != wantTag {
			t.Errorf("%s: value mapped by %q, want %q", tc.test, got, wantTag)
		}
	}
}

// TestUnionOwnAssertionsGetTheActiveBasicMember pins what a union's OWN
// assertions facet hands the evaluator as st: the ·active basic member· that
// identified v, under which `$value` is v's XDM representation (dt-xdmrep
// clause 4, cvc-assertions-valid clause 1.4), never the union — through a
// nested union down to its basic member (dt-active-basic-member). Every row
// fails with validateUnion handing checkAssertions st in place of member.
func TestUnionOwnAssertionsGetTheActiveBasicMember(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text", "preserve")
	b := memberBackend{num.Name(): allDigits, text.Name(): func(string) bool { return true }}
	hold := []xsd.Facet{xsd.NewAssertionsFacet([]xsd.Assertion{xsd.NewAssertion(xsd.NewXPathExpression("hold", nil, nil, nil))})}
	flat := unionRestriction(t, "flat", unionType2(t, "u", num, text), hold)
	nested := unionRestriction(t, "nested", unionType2(t, "outer", unionType2(t, "inner", num, text)), hold)
	for _, tc := range []struct {
		name    string
		st      *xsd.SimpleType
		lexical string
		want    *xsd.SimpleType
	}{
		{"the first member", flat, "7", num},
		{"a later member", flat, "x", text},
		{"a nested union's first member", nested, "7", num},
		{"a nested union's later member", nested, "x", text},
	} {
		var calls []assertionCall
		if _, err := ValidateLexical(b, noSchema{}, tc.st, tc.lexical, nil, scripted(&calls)); err != nil {
			t.Fatalf("%s: ValidateLexical(%q) = %v, want accept", tc.name, tc.lexical, err)
		}
		if len(calls) != 1 || calls[0].st != tc.want {
			t.Errorf("%s: Evaluate calls %v, want one handed %s", tc.name, calls, tc.want.Name())
		}
	}
}

// ruleOf is the rule err carries, empty where it carries none.
func ruleOf(err error) xsderr.Rule {
	rule, _ := xsderr.RuleOf(err)
	return rule
}

// TestListItemAssertionsRunPerItem pins the list recursion (dv_list): each item
// is Datatype Valid against the {item type definition} by the whole rule, its
// assertions facet included, so an item whose {test} fails rejects the list.
func TestListItemAssertionsRunPerItem(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	b := memberBackend{num.Name(): allDigits}
	item := asserting(t, "item", num, "fail")
	var calls []assertionCall
	_, err := ValidateLexical(b, noSchema{}, listType(t, item), "7 8", nil, scripted(&calls))
	if ruleOf(err) != ruleCvcAssertionsValid || !IsDatatypeVerdict(err) {
		t.Fatalf("ValidateLexical(list of a failing item) = %v, want the item's cvc-assertions-valid verdict", err)
	}
	if len(calls) != 1 || calls[0].st != item {
		t.Errorf("Evaluate calls = %d, want 1 against the item type", len(calls))
	}
}

// TestEnumerationRestrictionSkipsAnAssertionsDecline pins
// checkEnumerationRestriction's skip: CheckFacetRestriction holds no XPath
// engine, so a member every other facet of the base accepts reaches the base's
// assertions undecided, and charging that under §4.3.5.5 would reject a valid
// schema.
func TestEnumerationRestrictionSkipsAnAssertionsDecline(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	b := memberBackend{num.Name(): allDigits}
	base := asserting(t, "base", num, "hold")
	enum := xsd.NewEnumerationFacet([]xsd.EnumerationMember{xsd.NewEnumerationMember("7", nil, nil)})
	derived, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "derived"},
		xsd.RestrictionDerivation{}, base, []xsd.Facet{enum}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(derived): %v", err)
	}
	if err := CheckFacetRestriction(b, noSchema{}, derived); err != nil {
		t.Errorf("CheckFacetRestriction(enumeration over an asserting base) = %v, want nil", err)
	}
}
