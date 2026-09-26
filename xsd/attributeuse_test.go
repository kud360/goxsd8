package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// localDecl builds a LocalAttributeDeclaration wrapping a local-scope
// declaration with the given name, for use in Attribute Use tests.
func localDecl(t *testing.T, name xsd.QName) xsd.LocalAttributeDeclaration {
	t.Helper()
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, adLocalScope(t), nil, false)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	return xsd.LocalAttributeDeclaration{Declaration: d}
}

// localDeclVC builds a LocalAttributeDeclaration whose declaration carries the
// given {value constraint} variety, for exercising au-props-correct clause 3.
func localDeclVC(t *testing.T, kind xsd.ValueConstraintKind) xsd.LocalAttributeDeclaration {
	t.Helper()
	vc := xsd.NewValueConstraint(kind, "v", nil, nil)
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, adLocalScope(t), &vc, false)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	return xsd.LocalAttributeDeclaration{Declaration: d}
}

func TestNewAttributeUseValidLocalDeclaration(t *testing.T) {
	decl := localDecl(t, xsd.QName{Space: "urn:ns", Local: "a"})
	yes := true
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, true, decl, nil, &yes)
	if err != nil {
		t.Fatalf("NewAttributeUse unexpected error: %v", err)
	}
	if !u.Required() {
		t.Error("Required() = false, want true")
	}
	if !inheritableSchema(t, nil).ResolvedInheritable(u) {
		t.Error("ResolvedInheritable = false, want true")
	}
	got, ok := u.AttributeDeclaration().(xsd.LocalAttributeDeclaration)
	if !ok {
		t.Fatalf("AttributeDeclaration() type = %T, want LocalAttributeDeclaration", u.AttributeDeclaration())
	}
	if got.Declaration.Name() != (xsd.QName{Space: "urn:ns", Local: "a"}) {
		t.Errorf("declaration name = %v, want {urn:ns}a", got.Declaration.Name())
	}
	if _, ok := u.ValueConstraint(); ok {
		t.Error("ValueConstraint() ok = true for a nil-valueConstraint use, want false")
	}
}

func TestNewAttributeUseValidRef(t *testing.T) {
	ref := xsd.AttributeDeclarationRef{Name: xsd.QName{Space: "urn:ns", Local: "b"}}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, ref, nil, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse unexpected error: %v", err)
	}
	if u.Required() {
		t.Error("Required() = true, want false")
	}
	got, ok := u.AttributeDeclaration().(xsd.AttributeDeclarationRef)
	if !ok {
		t.Fatalf("AttributeDeclaration() type = %T, want AttributeDeclarationRef", u.AttributeDeclaration())
	}
	if got.Name != (xsd.QName{Space: "urn:ns", Local: "b"}) {
		t.Errorf("ref name = %v, want {urn:ns}b", got.Name)
	}
}

func TestNewAttributeUseRejectsNilDeclaration(t *testing.T) {
	_, err := xsd.NewAttributeUse(xsderr.Loc{}, false, nil, nil, nil)
	if err == nil {
		t.Fatal("NewAttributeUse(nil declaration) succeeded, want au-props-correct error")
	}
	assertRule(t, err, "au-props-correct")
}

// TestNewAttributeUseRejectsAbsentRefName proves the representation invariant
// (STYLE T1) on AttributeDeclarationRef: the variant maps only the ref-present
// branch (§3.2.2.3 ref.att.local), so its Name is always a present QName. An
// empty local part — with or without a namespace name — is rejected at
// construction rather than deferred, so finalize's "a zero QName is absent" skip
// (resolve.go) can never swallow an unresolvable {attribute declaration}.
func TestNewAttributeUseRejectsAbsentRefName(t *testing.T) {
	tests := []struct {
		name    string
		ref     xsd.QName
		wantErr bool
	}{
		{"zero QName", xsd.QName{}, true},
		{"namespace with empty local", xsd.QName{Space: "urn:ns"}, true},
		{"no-namespace present local", xsd.QName{Local: "a"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: tc.ref}, nil, nil)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("NewAttributeUse(ref %v) unexpected error: %v", tc.ref, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewAttributeUse(ref %v) succeeded, want an absent-ref-name rejection", tc.ref)
			}
			assertRule(t, err, xsderr.RuleComponentInvariant)
			// Only that the message LEADS with the author-visible construct
			// (STYLE E1); the rest of the sentence is not a contract.
			assertMsgLeadsWith(t, err, "<attribute ref>")
		})
	}
}

func TestNewAttributeUseValueConstraintRoundTrip(t *testing.T) {
	vc := xsd.NewValueConstraint(xsd.ValueDefault, "d", nil, nil)
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, localDecl(t, xsd.QName{Local: "a"}), &vc, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	got, ok := u.ValueConstraint()
	if !ok {
		t.Fatal("ValueConstraint() ok = false, want true")
	}
	if got.Kind() != xsd.ValueDefault || got.LexicalForm() != "d" {
		t.Errorf("ValueConstraint() = {%s %q}, want {default \"d\"}", got.Kind(), got.LexicalForm())
	}
}

// TestNewAttributeUseClause3 exercises au-props-correct clause 3's variety half
// for the Local case: a fixed declaration constrains the use's own variety.
func TestNewAttributeUseClause3(t *testing.T) {
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "v", nil, nil)
	deflt := xsd.NewValueConstraint(xsd.ValueDefault, "v", nil, nil)
	tests := []struct {
		name    string
		decl    xsd.AttributeDeclarationOrRef
		useVC   *xsd.ValueConstraint
		wantErr bool
	}{
		{"local fixed decl, fixed use", localDeclVC(t, xsd.ValueFixed), &fixed, false},
		{"local fixed decl, default use", localDeclVC(t, xsd.ValueFixed), &deflt, true},
		{"local fixed decl, no use vc", localDeclVC(t, xsd.ValueFixed), nil, false},
		{"local default decl, default use", localDeclVC(t, xsd.ValueDefault), &deflt, false},
		{"ref decl, default use", xsd.AttributeDeclarationRef{Name: xsd.QName{Local: "a"}}, &deflt, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewAttributeUse(xsderr.Loc{}, false, tc.decl, tc.useVC, nil)
			if tc.wantErr {
				if err == nil {
					t.Fatal("NewAttributeUse succeeded, want au-props-correct clause 3 error")
				}
				assertRule(t, err, "au-props-correct")
				return
			}
			if err != nil {
				t.Fatalf("NewAttributeUse unexpected error: %v", err)
			}
		})
	}
}

// inheritableSchema finalizes a schema whose {attribute declarations} hold at
// most a top-level "a" of the given {inheritable}; a nil global declares none.
func inheritableSchema(t *testing.T, global *bool) *xsd.Schema {
	t.Helper()
	b := xsd.NewSchemaBuilder()
	if global != nil {
		d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, nil, xsd.NewAttributeGlobalScope(), nil, *global)
		if err != nil {
			t.Fatalf("NewAttributeDeclaration(a): %v", err)
		}
		b.AddAttribute(d)
	}
	s, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return s
}

// TestResolvedInheritable pins an Attribute Use's {inheritable} for every shape
// of its own inheritable attribute (absent, false, true) on both mappings. The
// local form falls back to false (§3.2.2.2 dcl.att.local); the ref form falls
// back to the referenced declaration's {inheritable} (§3.2.2.3 ref.att.local),
// and a dangling ref answers false (#1682).
func TestResolvedInheritable(t *testing.T) {
	yes, no := true, false
	ref := xsd.AttributeDeclarationRef{Name: xsd.QName{Local: "a"}}
	for _, tc := range []struct {
		name   string
		decl   xsd.AttributeDeclarationOrRef
		own    *bool
		global *bool
		want   bool
	}{
		{"local, absent", localDecl(t, xsd.QName{Local: "a"}), nil, nil, false},
		{"local, absent, over a true global of the same name", localDecl(t, xsd.QName{Local: "a"}), nil, &yes, false},
		{"local, true", localDecl(t, xsd.QName{Local: "a"}), &yes, nil, true},
		{"ref, absent, over a true global", ref, nil, &yes, true},
		{"ref, absent, over a false global", ref, nil, &no, false},
		{"ref, false, over a true global (cta0012.xsd)", ref, &no, &yes, false},
		{"ref, true, over a false global (cta0011.xsd)", ref, &yes, &no, true},
		{"ref, absent, dangling", ref, nil, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, tc.decl, nil, tc.own)
			if err != nil {
				t.Fatalf("NewAttributeUse: %v", err)
			}
			if got := inheritableSchema(t, tc.global).ResolvedInheritable(u); got != tc.want {
				t.Errorf("ResolvedInheritable = %t, want %t", got, tc.want)
			}
		})
	}
}
