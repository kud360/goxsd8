package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// These tests are package-internal because the keyword half of {disallowed
// names} has no exported accessor this issue (STYLE T5: the readers are all
// in-package); reading the field is the only way to pin dedup and the
// cos-aw-intersect keyword formula.

func anyConstraint(t *testing.T, keywords ...DisallowedNameKeyword) NamespaceConstraint {
	t.Helper()
	c, err := NewNamespaceConstraint(xsderr.Loc{}, NamespaceConstraintAny, nil, nil, keywords)
	if err != nil {
		t.Fatalf("NewNamespaceConstraint(any, %v): %v", keywords, err)
	}
	return c
}

func TestNewNamespaceConstraintDedupsKeywords(t *testing.T) {
	// {disallowed names} is a SET: a repeated keyword collapses, in document
	// order of first occurrence.
	c := anyConstraint(t, DisallowedNameSibling, DisallowedNameDefined, DisallowedNameSibling)
	want := []DisallowedNameKeyword{DisallowedNameSibling, DisallowedNameDefined}
	if len(c.disallowedNameKeywords) != len(want) {
		t.Fatalf("keywords = %v, want %v", c.disallowedNameKeywords, want)
	}
	for i, k := range want {
		if c.disallowedNameKeywords[i] != k {
			t.Fatalf("keywords = %v, want %v", c.disallowedNameKeywords, want)
		}
	}
}

func TestNewNamespaceConstraintRejectsOutOfRangeKeyword(t *testing.T) {
	for _, k := range []DisallowedNameKeyword{0, 99} {
		_, err := NewNamespaceConstraint(xsderr.Loc{}, NamespaceConstraintAny, nil, nil, []DisallowedNameKeyword{k})
		if err == nil {
			t.Fatalf("NewNamespaceConstraint accepted keyword %d, want w-props-correct clause 1 rejection", uint8(k))
		}
		if got, ok := xsderr.RuleOf(err); !ok || got != ruleWildcardCorrect {
			t.Errorf("RuleOf = (%q, %v), want (%q, true)", got, ok, ruleWildcardCorrect)
		}
	}
}

func TestKeywordsDoNotAffectAllowsName(t *testing.T) {
	// cvc-wildcard-name (§3.10.4.2) clause 2 tests EXPANDED NAMES only, so a
	// keyword member never changes AllowsName's answer; cvc-wildcard clauses 2-3
	// are a different rule, resolved in wildcardadmit.go.
	c := anyConstraint(t, DisallowedNameDefined, DisallowedNameSibling)
	if !c.AllowsName(QName{Space: "urn:x", Local: "e"}) {
		t.Fatal("AllowsName rejected a name on the strength of a keyword member")
	}
}

func TestIntersectDisallowedNameKeywordsUnionsDefined(t *testing.T) {
	// cos-aw-intersect (§3.10.6.4) bullet 3: defined survives if it is a member
	// of EITHER operand — a union, the opposite of cos-aw-union's "both".
	cases := []struct {
		name string
		a, b []DisallowedNameKeyword
		want bool
	}{
		{"neither", nil, nil, false},
		{"left-only", []DisallowedNameKeyword{DisallowedNameDefined}, nil, true},
		{"right-only", nil, []DisallowedNameKeyword{DisallowedNameDefined}, true},
		{"both", []DisallowedNameKeyword{DisallowedNameDefined}, []DisallowedNameKeyword{DisallowedNameDefined}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := IntersectNamespaceConstraint(xsderr.Loc{}, anyConstraint(t, c.a...), anyConstraint(t, c.b...))
			if err != nil {
				t.Fatalf("IntersectNamespaceConstraint: %v", err)
			}
			if has := got.hasDisallowedNameKeyword(DisallowedNameDefined); has != c.want {
				t.Errorf("intersection contains defined = %v, want %v", has, c.want)
			}
			// Intersection is commutative in this half too.
			swapped, err := IntersectNamespaceConstraint(xsderr.Loc{}, anyConstraint(t, c.b...), anyConstraint(t, c.a...))
			if err != nil {
				t.Fatalf("IntersectNamespaceConstraint (swapped): %v", err)
			}
			if has := swapped.hasDisallowedNameKeyword(DisallowedNameDefined); has != c.want {
				t.Errorf("swapped intersection contains defined = %v, want %v", has, c.want)
			}
		})
	}
}
