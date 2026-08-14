package value

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestIsDatatypeVerdictSeparatesTypeFaultsFromRejections is the regression test
// the predicate exists for: every row below returns an error from
// ValidateLexical, and the rule ID alone cannot tell the two halves apart — an
// ungoverned type and a genuine rejection are both cvc-datatype-valid — so a
// caller reading the ID would false-reject every typeless attribute (§3.2.2.2's
// third tier types one as xs:anySimpleType, which no backend maps).
func TestIsDatatypeVerdictSeparatesTypeFaultsFromRejections(t *testing.T) {
	prim := vsPrim(t, "int")
	b := intBackend{mapped: prim.Name()}
	bad, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "badPattern"},
		xsd.RestrictionDerivation{}, prim,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetPattern, []string{`[a-`}, false)}, nil)
	if err != nil {
		t.Fatalf("building the bad-pattern type: %v", err)
	}
	noWS, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}, nil, nil)
	if err != nil {
		t.Fatalf("building the whiteSpace-less type: %v", err)
	}

	for _, tc := range []struct {
		name    string
		backend Backend
		t       *xsd.SimpleType
		lexical string
		want    bool
	}{
		{"a lexical the mapping rejects", b, prim, "zzz", true},
		{"an ungoverned type", emptyBackend{}, prim, "1", false},
		{"an ungoverned list item type", emptyBackend{}, vsList(t, "ints", prim), "1 2", false},
		{"an ungoverned union member", emptyBackend{}, vsUnion(t, "u", prim), "1", false},
		{"a pattern facet that will not compile", b, bad, "1", false},
		{"no whiteSpace mode in force", b, noWS, "1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateLexical(tc.backend, noSchema{}, tc.t, tc.lexical, nil)
			if err == nil {
				t.Fatal("ValidateLexical = nil, want an error to classify")
			}
			if got := IsDatatypeVerdict(err); got != tc.want {
				t.Errorf("IsDatatypeVerdict(%v) = %t, want %t", err, got, tc.want)
			}
		})
	}

	if IsDatatypeVerdict(nil) {
		t.Error("IsDatatypeVerdict(nil) = true, want false: no error is no verdict")
	}
	if _, err := ValidateLexical(b, noSchema{}, prim, "1", nil); err != nil {
		t.Fatalf("ValidateLexical(accepted) = %v, want nil", err)
	}
}

// TestTypeFaultKeepsTheRuleAndSubsumesPreconditions pins the two properties the
// marking must not cost: xsderr.RuleOf still reaches the *xsderr.Error under the
// wrapper, so a diagnostic still names the rule, and IsFacetPrecondition still
// answers the narrower "which fault" question for the cohort that wraps
// errFacetPrecondition — one chain, not two encodings.
func TestTypeFaultKeepsTheRuleAndSubsumesPreconditions(t *testing.T) {
	prim := vsPrim(t, "int")
	_, err := ValidateLexical(emptyBackend{}, noSchema{}, prim, "1", nil)
	if err == nil {
		t.Fatal("ValidateLexical(ungoverned) = nil, want the backend-gap error")
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != ruleCvcDatatypeValid {
		t.Errorf("RuleOf = (%q, %t), want (%q, true) through the marking", rule, ok, ruleCvcDatatypeValid)
	}
	if IsFacetPrecondition(err) {
		t.Error("IsFacetPrecondition(ungoverned) = true: a backend gap is not a precondition fault")
	}

	noWS, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "int"}, nil, nil)
	if err != nil {
		t.Fatalf("building the whiteSpace-less type: %v", err)
	}
	_, err = ValidateLexical(intBackend{mapped: noWS.Name()}, noSchema{}, noWS, "1", nil)
	if !IsFacetPrecondition(err) {
		t.Fatalf("IsFacetPrecondition(%v) = false, want true", err)
	}
	if IsDatatypeVerdict(err) {
		t.Error("IsDatatypeVerdict(precondition fault) = true: a precondition fault is one kind of type fault")
	}
}

// TestConstraintMatchesComparesInTheValueSpace pins cvc-attribute clause 4 /
// cvc-au's "equal or identical": the comparison is over ·actual values·, so "1"
// and "01" agree while a lexical comparison would report them different, and a
// genuinely different value is decided NOT the same — the verdict the charge
// stands on.
func TestConstraintMatchesComparesInTheValueSpace(t *testing.T) {
	prim := vsPrim(t, "int")
	b := intBackend{mapped: prim.Name()}

	for _, tc := range []struct {
		name    string
		lexical string
		fixed   string
		want    bool
	}{
		{"the same lexical", "1", "1", true},
		{"two lexicals of one value", "01", "1", true},
		{"whiteSpace normalization precedes the mapping", " 1 ", "1", true},
		{"different values", "2", "1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			same, decided := ConstraintMatches(b, noSchema{}, prim, tc.lexical, nil, vsFixed(tc.fixed))
			if !decided {
				t.Fatalf("ConstraintMatches = (%t, false), want a decided answer", same)
			}
			if same != tc.want {
				t.Errorf("ConstraintMatches = (%t, true), want (%t, true)", same, tc.want)
			}
		})
	}
}

// TestConstraintMatchesFailsOpen pins the fail-open contract: nothing a caller
// could charge a violation from comes out of a side that did not validate, and
// an ungoverned type answers undecided rather than "not the same".
func TestConstraintMatchesFailsOpen(t *testing.T) {
	prim := vsPrim(t, "int")
	b := intBackend{mapped: prim.Name()}

	for _, tc := range []struct {
		name    string
		backend Backend
		lexical string
		fixed   string
	}{
		{"an instance lexical outside the lexical space", b, "zzz", "1"},
		{"a fixed {lexical form} outside it", b, "1", "zzz"},
		{"an ungoverned type", emptyBackend{}, "1", "1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if same, decided := ConstraintMatches(tc.backend, noSchema{}, prim, tc.lexical, nil, vsFixed(tc.fixed)); decided {
				t.Errorf("ConstraintMatches = (%t, %t), want undecided (fail-open)", same, decided)
			}
		})
	}
}

// TestConstraintMatchesResolvesEachSideInItsOwnContext pins the asymmetry: the
// instance literal resolves against the namespace bindings in scope where the
// ATTRIBUTE was written and the fixed {lexical form} against those in scope where
// the SCHEMA wrote it (§3.3.18), so two different prefixes naming one namespace
// agree and one prefix bound differently on the two sides does not.
func TestConstraintMatchesResolvesEachSideInItsOwnContext(t *testing.T) {
	qname := vsPrim(t, "QName")
	b := qnameBackend{mapped: qname.Name()}
	instance := nsContext{bindings: map[string]string{"i": "urn:one"}}

	same, decided := ConstraintMatches(b, noSchema{}, qname, "i:x", instance,
		vsFixedIn("s:x", nil, binding("s", "urn:one")))
	if !decided || !same {
		t.Errorf("ConstraintMatches = (%t, %t), want (true, true): both prefixes name urn:one", same, decided)
	}

	same, decided = ConstraintMatches(b, noSchema{}, qname, "i:x", instance,
		vsFixedIn("s:x", nil, binding("s", "urn:two")))
	if !decided || same {
		t.Errorf("ConstraintMatches = (%t, %t), want (false, true): the two prefixes name different namespaces", same, decided)
	}
}
