package value

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// lexicalValue is a value that is its own lexical, equal to another exactly
// where the two lexicals are.
type lexicalValue string

func (v lexicalValue) Eq(other Value) bool {
	o, ok := other.(lexicalValue)
	return ok && o == v
}

// lexicalBackend maps xs:NOTATION and xs:string each to a Parse accepting any
// lexical as itself, as a leaf mapping holding no schema accepts an undeclared
// NOTATION.
type lexicalBackend struct{}

func (lexicalBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if typ != notationName && typ != (xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}) {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, _ Context) (Value, error) { return lexicalValue(lexical), nil }}, true
}

// notationScope is a [Context] binding the empty prefix to no namespace, as an
// element with no default namespace declaration in scope does.
type notationScope struct{}

func (notationScope) LookupNamespace(prefix string) (string, bool) { return "", prefix == "" }

// declaringFoo is an assembled *xsd.Schema whose one notation declaration is
// the no-namespace foo.
func declaringFoo(t *testing.T) *xsd.Schema {
	t.Helper()
	public := "pubfoo"
	n, err := xsd.NewNotation(xsderr.Loc{}, xsd.QName{Local: "foo"}, nil, &public)
	if err != nil {
		t.Fatalf("NewNotation: %v", err)
	}
	sb := xsd.NewSchemaBuilder()
	sb.AddNotation(n)
	s, err := sb.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// wantUndeclaredBez fails unless err is a cvc-datatype-valid verdict carrying
// the one rejecting the undeclared NOTATION value bez: that verdict itself on
// the atomic and list routes, and folded into the union's on the union route.
func wantUndeclaredBez(t *testing.T, route string, err error) {
	t.Helper()
	const verdict = `the NOTATION value "bez" resolves to the QName bez, which names no notation declaration of the schema`
	rule, _ := xsderr.RuleOf(err)
	if !IsDatatypeVerdict(err) || rule != ruleCvcDatatypeValid || !strings.Contains(err.Error(), verdict) {
		t.Errorf("%s: err = %v, want a cvc-datatype-valid verdict carrying %q", route, err, verdict)
	}
}

// ValidateLexical and ValidatingType hold NOTATION to "the set of QNames of
// notations declared in the current schema" (Datatypes §3.3.19) wherever the
// resolver is an *xsd.Schema, on every route NOTATION decides a literal by:
// xs:NOTATION itself, a list item, and a union member. Against a resolver
// that declares nothing (noSchema), the same undeclared bez is accepted.
func TestValidateLexicalHoldsNotationToTheSchemasDeclarations(t *testing.T) {
	b := lexicalBackend{}
	schema := declaringFoo(t)
	notation := primType(t, "NOTATION", "collapse")
	str := primType(t, "string", "preserve")
	routes := []struct {
		name    string
		st      *xsd.SimpleType
		lexical string
	}{
		{"atomic", notation, "bez"},
		{"list item", listType(t, notation), "foo bez"},
		{"union member", unionType2(t, "nota", notation), "bez"},
	}
	for _, c := range routes {
		if _, err := ValidateLexical(b, schema, c.st, "foo", notationScope{}, assertionsUndecided{}); err != nil {
			t.Errorf("%s: ValidateLexical(foo) = %v, want the declared foo accepted", c.name, err)
		}
		_, err := ValidateLexical(b, schema, c.st, c.lexical, notationScope{}, assertionsUndecided{})
		wantUndeclaredBez(t, c.name+": ValidateLexical", err)
		_, _, err = ValidatingType(b, schema, c.st, c.lexical, notationScope{}, assertionsUndecided{})
		wantUndeclaredBez(t, c.name+": ValidatingType", err)

		if _, err := ValidateLexical(b, noSchema{}, c.st, c.lexical, notationScope{}, assertionsUndecided{}); err != nil {
			t.Errorf("%s: ValidateLexical against no notationDeclarer = %v, want %q accepted", c.name, err, c.lexical)
		}
		if _, _, err := ValidatingType(b, noSchema{}, c.st, c.lexical, notationScope{}, assertionsUndecided{}); err != nil {
			t.Errorf("%s: ValidatingType against no notationDeclarer = %v, want %q accepted", c.name, err, c.lexical)
		}
	}

	// A union member's undeclared NOTATION is that member's rejection, so the
	// dispatch falls through to the next member (Datatype Valid 2.3), which
	// becomes the ·validating type·.
	u := unionType2(t, "notaOrString", notation, str)
	got, _, err := ValidatingType(b, schema, u, "bez", notationScope{}, assertionsUndecided{})
	if err != nil || got != str {
		t.Errorf("ValidatingType(union(NOTATION, string), bez) = (%v, %v), want the string member", got, err)
	}
	got, _, err = ValidatingType(b, noSchema{}, u, "bez", notationScope{}, assertionsUndecided{})
	if err != nil || got != notation {
		t.Errorf("ValidatingType(union(NOTATION, string), bez) against no notationDeclarer = (%v, %v), want the NOTATION member", got, err)
	}

	// A nil Context binds no prefix, the empty one included, so even the
	// declared foo resolves to no QName under one, whatever b's mapping admits.
	_, err = ValidateLexical(b, schema, notation, "foo", nil, assertionsUndecided{})
	if rule, _ := xsderr.RuleOf(err); rule != ruleCvcDatatypeValid {
		t.Errorf("ValidateLexical(foo) under a nil Context = %v, want a cvc-datatype-valid verdict", err)
	}
}

// A fixed {value constraint} naming an undeclared notation, which ValidDefault
// leaves undecided at finalize (its gate 1 declines every NOTATION-governed
// type), is outside NOTATION's value space against the schema, so
// ConstraintMatches answers undecided where it used to compare the two QNames:
// a declared instance value against it is not reported NOT-same, and the
// undeclared value itself is not reported same.
func TestConstraintMatchesAnUndeclaredNotationFixedValueIsUndecided(t *testing.T) {
	b := lexicalBackend{}
	schema := declaringFoo(t)
	notation := primType(t, "NOTATION", "collapse")
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "bez", nil, nil)
	for _, lexical := range []string{"foo", "bez"} {
		if same, decided := ConstraintMatches(b, schema, notation, lexical, notationScope{}, fixed, assertionsUndecided{}); decided {
			t.Errorf("ConstraintMatches(%s, fixed bez) = (%t, decided), want undecided", lexical, same)
		}
	}
	if same, decided := ConstraintMatches(b, noSchema{}, notation, "bez", notationScope{}, fixed, assertionsUndecided{}); !same || !decided {
		t.Errorf("ConstraintMatches(bez, fixed bez) against no notationDeclarer = (%t, %t), want (true, true)", same, decided)
	}
}

// The declared-set decision runs before the assertions stage, so an
// undeclared NOTATION value is the cvc-datatype-valid verdict and never
// reaches an assertion: under an evaluator that fails every {test} the
// verdict is still NOTATION's, and no call is made. A declared foo reaches the
// assertion and is its verdict.
func TestValidateLexicalDecidesNotationBeforeAssertions(t *testing.T) {
	b := lexicalBackend{}
	schema := declaringFoo(t)
	st := asserting(t, "assertedNota", primType(t, "NOTATION", "collapse"), "fail")
	var calls []assertionCall
	_, err := ValidateLexical(b, schema, st, "bez", notationScope{}, scripted(&calls))
	wantUndeclaredBez(t, "asserted NOTATION", err)
	if len(calls) != 0 {
		t.Errorf("assertion calls for the undeclared bez = %d, want 0", len(calls))
	}
	_, err = ValidateLexical(b, schema, st, "foo", notationScope{}, scripted(&calls))
	if rule, _ := xsderr.RuleOf(err); rule != ruleCvcAssertionsValid || len(calls) != 1 {
		t.Errorf("ValidateLexical(foo) = %v after %d calls, want a cvc-assertions-valid verdict after 1", err, len(calls))
	}
}
