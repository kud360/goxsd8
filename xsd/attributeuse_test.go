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
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, adLocalScope(t), nil, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	return xsd.LocalAttributeDeclaration{Declaration: d}
}

// localDeclVC builds a LocalAttributeDeclaration whose declaration carries the
// given {value constraint} variety, for exercising au-props-correct clause 3.
func localDeclVC(t *testing.T, kind xsd.ValueConstraintKind) xsd.LocalAttributeDeclaration {
	t.Helper()
	vc := xsd.NewValueConstraint(kind, "v")
	d, err := xsd.NewAttributeDeclaration(xsderr.Loc{}, xsd.QName{Local: "a"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, adLocalScope(t), &vc, false, nil)
	if err != nil {
		t.Fatalf("NewAttributeDeclaration: %v", err)
	}
	return xsd.LocalAttributeDeclaration{Declaration: d}
}

func TestNewAttributeUseValidLocalDeclaration(t *testing.T) {
	decl := localDecl(t, xsd.QName{Space: "urn:ns", Local: "a"})
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, true, decl, nil, true, nil)
	if err != nil {
		t.Fatalf("NewAttributeUse unexpected error: %v", err)
	}
	if !u.Required() {
		t.Error("Required() = false, want true")
	}
	if !u.Inheritable() {
		t.Error("Inheritable() = false, want true")
	}
	got, ok := u.AttributeDeclaration().(xsd.LocalAttributeDeclaration)
	if !ok {
		t.Fatalf("AttributeDeclaration() type = %T, want LocalAttributeDeclaration", u.AttributeDeclaration())
	}
	if got.Declaration.Name() != (xsd.QName{Space: "urn:ns", Local: "a"}) {
		t.Errorf("declaration name = %v, want {urn:ns}a", got.Declaration.Name())
	}
	if u.Annotations() != nil {
		t.Errorf("Annotations() = %v, want nil", u.Annotations())
	}
	if _, ok := u.ValueConstraint(); ok {
		t.Error("ValueConstraint() ok = true for a nil-valueConstraint use, want false")
	}
}

func TestNewAttributeUseValidRef(t *testing.T) {
	ref := xsd.AttributeDeclarationRef{Name: xsd.QName{Space: "urn:ns", Local: "b"}}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, ref, nil, false, nil)
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
	_, err := xsd.NewAttributeUse(xsderr.Loc{}, false, nil, nil, false, nil)
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
			_, err := xsd.NewAttributeUse(xsderr.Loc{}, false, xsd.AttributeDeclarationRef{Name: tc.ref}, nil, false, nil)
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
	vc := xsd.NewValueConstraint(xsd.ValueDefault, "d")
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, localDecl(t, xsd.QName{Local: "a"}), &vc, false, nil)
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
	fixed := xsd.NewValueConstraint(xsd.ValueFixed, "v")
	deflt := xsd.NewValueConstraint(xsd.ValueDefault, "v")
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
			_, err := xsd.NewAttributeUse(xsderr.Loc{}, false, tc.decl, tc.useVC, false, nil)
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

func TestAttributeUseAnnotationsRoundTripAndAlias(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "u")}, nil),
	}
	u, err := xsd.NewAttributeUse(xsderr.Loc{}, false, localDecl(t, xsd.QName{Local: "a"}), nil, false, anns)
	if err != nil {
		t.Fatalf("NewAttributeUse: %v", err)
	}
	if got := u.Annotations(); len(got) != 1 || got[0].Documentation()[0].Content() != "u" {
		t.Errorf("Annotations() = %+v, want one with content u", got)
	}
	anns[0] = xsd.NewAnnotation(nil, nil, nil)
	if got := u.Annotations(); got[0].Documentation()[0].Content() != "u" {
		t.Error("AttributeUse aliased the constructor annotations slice")
	}
	first := u.Annotations()
	first[0] = xsd.NewAnnotation(nil, nil, nil)
	if got := u.Annotations(); got[0].Documentation()[0].Content() != "u" {
		t.Error("Annotations() returned an aliased slice")
	}
}
