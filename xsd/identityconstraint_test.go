package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func xp(expr string) xsd.XPathExpression {
	return xsd.NewXPathExpression(expr, nil, nil, nil)
}

func TestNewIdentityConstraintValid(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "k"}
	refer := xsd.QName{Space: "urn:ns", Local: "target"}
	cases := []struct {
		name        string
		category    xsd.IdentityConstraintCategory
		refer       *xsd.QName
		wantRefer   xsd.QName
		wantReferOK bool
	}{
		{"key", xsd.IdentityConstraintKey, nil, xsd.QName{}, false},
		{"unique", xsd.IdentityConstraintUnique, nil, xsd.QName{}, false},
		{"keyref", xsd.IdentityConstraintKeyref, &refer, refer, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			sel := xp(".//foo")
			fields := []xsd.XPathExpression{xp("@a"), xp("@b")}
			ic, err := xsd.NewIdentityConstraint(xsderr.Loc{}, name, c.category, sel, fields, c.refer, nil)
			if err != nil {
				t.Fatalf("NewIdentityConstraint unexpected error: %v", err)
			}
			if ic.Name() != name {
				t.Errorf("Name() = %v, want %v", ic.Name(), name)
			}
			if ic.Category() != c.category {
				t.Errorf("Category() = %v, want %v", ic.Category(), c.category)
			}
			if ic.Selector().Expression() != ".//foo" {
				t.Errorf("Selector().Expression() = %q, want %q", ic.Selector().Expression(), ".//foo")
			}
			gotFields := ic.Fields()
			if len(gotFields) != 2 || gotFields[0].Expression() != "@a" || gotFields[1].Expression() != "@b" {
				t.Errorf("Fields() = %+v, want [@a @b]", gotFields)
			}
			gotRefer, referOK := ic.ReferencedKeyName()
			if referOK != c.wantReferOK || gotRefer != c.wantRefer {
				t.Errorf("ReferencedKeyName() = (%v, %v), want (%v, %v)", gotRefer, referOK, c.wantRefer, c.wantReferOK)
			}
		})
	}
}

func TestNewIdentityConstraintRejectsUnknownCategory(t *testing.T) {
	_, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, 0, xp("."), []xsd.XPathExpression{xp("@a")}, nil, nil)
	if err == nil {
		t.Fatal("NewIdentityConstraint(category=0) succeeded, want c-props-correct error")
	}
	assertRule(t, err, "c-props-correct")
}

func TestNewIdentityConstraintRejectsEmptyFields(t *testing.T) {
	for _, fields := range [][]xsd.XPathExpression{nil, {}} {
		_, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, xsd.IdentityConstraintKey, xp("."), fields, nil, nil)
		if err == nil {
			t.Fatalf("NewIdentityConstraint(fields=%v) succeeded, want c-props-correct error", fields)
		}
		assertRule(t, err, "c-props-correct")
	}
}

func TestNewIdentityConstraintRejectsReferKeyMismatch(t *testing.T) {
	refer := xsd.QName{Local: "target"}
	fields := []xsd.XPathExpression{xp("@a")}

	// keyref WITHOUT a {referenced key}.
	_, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, xsd.IdentityConstraintKeyref, xp("."), fields, nil, nil)
	if err == nil {
		t.Fatal("keyref without {referenced key} succeeded, want c-props-correct error")
	}
	assertRule(t, err, "c-props-correct")

	// non-keyref WITH a {referenced key}.
	for _, cat := range []xsd.IdentityConstraintCategory{xsd.IdentityConstraintKey, xsd.IdentityConstraintUnique} {
		_, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, cat, xp("."), fields, &refer, nil)
		if err == nil {
			t.Fatalf("%v with {referenced key} succeeded, want c-props-correct error", cat)
		}
		assertRule(t, err, "c-props-correct")
	}
}

func TestIdentityConstraintAnnotationsRoundTrip(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "second")}, nil),
	}
	ic, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, xsd.IdentityConstraintKey, xp("."), []xsd.XPathExpression{xp("@a")}, nil, anns)
	if err != nil {
		t.Fatalf("NewIdentityConstraint: %v", err)
	}
	got := ic.Annotations()
	if len(got) != 2 {
		t.Fatalf("Annotations() len = %d, want 2", len(got))
	}
	if docs := got[0].Documentation(); len(docs) != 1 || docs[0].Content() != "first" {
		t.Errorf("Annotations()[0] documentation = %+v, want content %q", docs, "first")
	}
}

func TestIdentityConstraintAnnotationsNilWhenEmpty(t *testing.T) {
	ic, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, xsd.IdentityConstraintKey, xp("."), []xsd.XPathExpression{xp("@a")}, nil, nil)
	if err != nil {
		t.Fatalf("NewIdentityConstraint: %v", err)
	}
	if got := ic.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty {annotations}", got)
	}
}

func TestIdentityConstraintFieldsAccessorDoesNotAlias(t *testing.T) {
	ic, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, xsd.IdentityConstraintKey, xp("."), []xsd.XPathExpression{xp("@a")}, nil, nil)
	if err != nil {
		t.Fatalf("NewIdentityConstraint: %v", err)
	}
	first := ic.Fields()
	first[0] = xp("tampered")

	if second := ic.Fields(); second[0].Expression() != "@a" {
		t.Errorf("Fields() returned an aliased slice: got %q, want %q", second[0].Expression(), "@a")
	}
}

func TestIdentityConstraintDoesNotAliasConstructorFields(t *testing.T) {
	fields := []xsd.XPathExpression{xp("@a")}
	ic, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, xsd.IdentityConstraintKey, xp("."), fields, nil, nil)
	if err != nil {
		t.Fatalf("NewIdentityConstraint: %v", err)
	}
	// Mutate the ORIGINAL slice after construction.
	fields[0] = xp("tampered")

	if got := ic.Fields(); got[0].Expression() != "@a" {
		t.Errorf("IdentityConstraint aliased the constructor slice: got %q, want %q", got[0].Expression(), "@a")
	}
}

func TestIdentityConstraintCategoryString(t *testing.T) {
	cases := []struct {
		cat  xsd.IdentityConstraintCategory
		want string
	}{
		{xsd.IdentityConstraintKey, "key"},
		{xsd.IdentityConstraintKeyref, "keyref"},
		{xsd.IdentityConstraintUnique, "unique"},
		{xsd.IdentityConstraintCategory(0), "IdentityConstraintCategory(0)"},
		{xsd.IdentityConstraintCategory(99), "IdentityConstraintCategory(99)"},
	}
	for _, c := range cases {
		if got := c.cat.String(); got != c.want {
			t.Errorf("IdentityConstraintCategory(%d).String() = %q, want %q", c.cat, got, c.want)
		}
	}
}
