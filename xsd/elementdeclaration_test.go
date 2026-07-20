package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// typeAlt is a small helper: a Type Alternative with the given test expression
// (empty string means the test-absent "otherwise" alternative).
func typeAlt(test string, typeName xsd.QName) xsd.TypeAlternative {
	if test == "" {
		return xsd.NewTypeAlternative(nil, typeName, nil)
	}
	x := xp(test)
	return xsd.NewTypeAlternative(&x, typeName, nil)
}

func TestNewTypeTableValid(t *testing.T) {
	even := xsd.QName{Space: "urn:t", Local: "Even"}
	dflt := xsd.QName{Space: "urn:t", Local: "Default"}
	alts := []xsd.TypeAlternative{typeAlt("@a > 0", even)}
	tt, err := xsd.NewTypeTable(xsderr.Loc{}, alts, typeAlt("", dflt))
	if err != nil {
		t.Fatalf("NewTypeTable unexpected error: %v", err)
	}
	got := tt.Alternatives()
	if len(got) != 1 || got[0].TypeDefinitionName() != even {
		t.Errorf("Alternatives() = %+v, want one alternative for %v", got, even)
	}
	if def := tt.DefaultTypeDefinition(); def.TypeDefinitionName() != dflt {
		t.Errorf("DefaultTypeDefinition() type = %v, want %v", def.TypeDefinitionName(), dflt)
	}
	if _, ok := tt.DefaultTypeDefinition().Test(); ok {
		t.Error("DefaultTypeDefinition().Test() ok = true, want false for the otherwise alternative")
	}
}

func TestNewTypeTableRejectsAlternativeWithoutTest(t *testing.T) {
	// An {alternatives} member with an absent {test} violates clause 6.
	alts := []xsd.TypeAlternative{typeAlt("", xsd.QName{Local: "T"})}
	_, err := xsd.NewTypeTable(xsderr.Loc{}, alts, typeAlt("", xsd.QName{Local: "D"}))
	if err == nil {
		t.Fatal("NewTypeTable(alternative without test) succeeded, want e-props-correct error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestNewTypeTableRejectsDefaultWithTest(t *testing.T) {
	// The {default type definition} must be the test-absent alternative.
	alts := []xsd.TypeAlternative{typeAlt("@a", xsd.QName{Local: "T"})}
	_, err := xsd.NewTypeTable(xsderr.Loc{}, alts, typeAlt("@fallback", xsd.QName{Local: "D"}))
	if err == nil {
		t.Fatal("NewTypeTable(default with test) succeeded, want e-props-correct error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestTypeTableAlternativesAccessorDoesNotAlias(t *testing.T) {
	alts := []xsd.TypeAlternative{typeAlt("@a", xsd.QName{Local: "T"})}
	tt, err := xsd.NewTypeTable(xsderr.Loc{}, alts, typeAlt("", xsd.QName{Local: "D"}))
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	first := tt.Alternatives()
	first[0] = typeAlt("@z", xsd.QName{Local: "Tampered"})
	if second := tt.Alternatives(); second[0].TypeDefinitionName() != (xsd.QName{Local: "T"}) {
		t.Errorf("Alternatives() returned an aliased slice: got %v", second[0].TypeDefinitionName())
	}
}

func TestTypeTableDoesNotAliasConstructorAlternatives(t *testing.T) {
	alts := []xsd.TypeAlternative{typeAlt("@a", xsd.QName{Local: "T"})}
	tt, err := xsd.NewTypeTable(xsderr.Loc{}, alts, typeAlt("", xsd.QName{Local: "D"}))
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	alts[0] = typeAlt("@z", xsd.QName{Local: "Tampered"})
	if got := tt.Alternatives(); got[0].TypeDefinitionName() != (xsd.QName{Local: "T"}) {
		t.Errorf("TypeTable aliased the constructor slice: got %v", got[0].TypeDefinitionName())
	}
}

func TestNewElementDeclarationValidGlobalNoAffiliations(t *testing.T) {
	name := xsd.QName{Space: "urn:ns", Local: "root"}
	typ := xsd.QName{Space: "urn:t", Local: "RootType"}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, name, typ, nil, xsd.ScopeGlobal, nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration unexpected error: %v", err)
	}
	if e.Name() != name {
		t.Errorf("Name() = %v, want %v", e.Name(), name)
	}
	if e.TypeDefinitionName() != typ {
		t.Errorf("TypeDefinitionName() = %v, want %v", e.TypeDefinitionName(), typ)
	}
	if e.ScopeVariety() != xsd.ScopeGlobal {
		t.Errorf("ScopeVariety() = %v, want global", e.ScopeVariety())
	}
	if e.Nillable() {
		t.Error("Nillable() = true, want false")
	}
	if e.Abstract() {
		t.Error("Abstract() = true, want false")
	}
	if _, ok := e.TypeTable(); ok {
		t.Error("TypeTable() ok = true, want false for absent type table")
	}
	if _, ok := e.ValueConstraint(); ok {
		t.Error("ValueConstraint() ok = true, want false for absent value constraint")
	}
	if got := e.SubstitutionGroupAffiliationNames(); got != nil {
		t.Errorf("SubstitutionGroupAffiliationNames() = %v, want nil", got)
	}
}

func TestNewElementDeclarationValidWithAffiliations(t *testing.T) {
	head := xsd.QName{Space: "urn:ns", Local: "head"}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "member"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, true, nil, []xsd.QName{head}, []xsd.DerivationMethod{xsd.DerivationExtension}, true, []xsd.DerivationMethod{xsd.DerivationSubstitution}, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration unexpected error: %v", err)
	}
	if !e.Nillable() {
		t.Error("Nillable() = false, want true")
	}
	if !e.Abstract() {
		t.Error("Abstract() = false, want true")
	}
	if got := e.SubstitutionGroupAffiliationNames(); len(got) != 1 || got[0] != head {
		t.Errorf("SubstitutionGroupAffiliationNames() = %v, want [%v]", got, head)
	}
	if got := e.SubstitutionGroupExclusions(); len(got) != 1 || got[0] != xsd.DerivationExtension {
		t.Errorf("SubstitutionGroupExclusions() = %v, want [extension]", got)
	}
	if got := e.DisallowedSubstitutions(); len(got) != 1 || got[0] != xsd.DerivationSubstitution {
		t.Errorf("DisallowedSubstitutions() = %v, want [substitution]", got)
	}
}

func TestNewElementDeclarationTypeTableAndValueConstraintPresent(t *testing.T) {
	tt, err := xsd.NewTypeTable(xsderr.Loc{}, []xsd.TypeAlternative{typeAlt("@a", xsd.QName{Local: "T"})}, typeAlt("", xsd.QName{Local: "D"}))
	if err != nil {
		t.Fatalf("NewTypeTable: %v", err)
	}
	vc := xsd.NewValueConstraint(xsd.ValueFixed, "42")
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, &tt, xsd.ScopeLocal, &vc, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	gotTT, ok := e.TypeTable()
	if !ok {
		t.Fatal("TypeTable() ok = false, want true")
	}
	if gotTT.DefaultTypeDefinition().TypeDefinitionName() != (xsd.QName{Local: "D"}) {
		t.Errorf("TypeTable default = %v, want D", gotTT.DefaultTypeDefinition().TypeDefinitionName())
	}
	gotVC, ok := e.ValueConstraint()
	if !ok {
		t.Fatal("ValueConstraint() ok = false, want true")
	}
	if gotVC.Kind() != xsd.ValueFixed || gotVC.LexicalForm() != "42" {
		t.Errorf("ValueConstraint() = (%v, %q), want (fixed, 42)", gotVC.Kind(), gotVC.LexicalForm())
	}
}

func TestNewElementDeclarationRejectsUnknownScope(t *testing.T) {
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeVariety(0), nil, false, nil, nil, nil, false, nil, nil)
	if err == nil {
		t.Fatal("NewElementDeclaration(scope=0) succeeded, want e-props-correct error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestNewElementDeclarationRejectsLocalScopeWithAffiliations(t *testing.T) {
	head := xsd.QName{Local: "head"}
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeLocal, nil, false, nil, []xsd.QName{head}, nil, false, nil, nil)
	if err == nil {
		t.Fatal("NewElementDeclaration(local scope + affiliations) succeeded, want e-props-correct clause 3 error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestNewElementDeclarationRejectsIllegalExclusion(t *testing.T) {
	// substitution is not a legal {substitution group exclusions} token.
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, false, nil, nil, []xsd.DerivationMethod{xsd.DerivationSubstitution}, false, nil, nil)
	if err == nil {
		t.Fatal("NewElementDeclaration(exclusion=substitution) succeeded, want e-props-correct error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestNewElementDeclarationRejectsIllegalDisallowedSubstitution(t *testing.T) {
	// list is not a legal {disallowed substitutions} token.
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, false, nil, nil, nil, false, []xsd.DerivationMethod{xsd.DerivationList}, nil)
	if err == nil {
		t.Fatal("NewElementDeclaration(disallowed=list) succeeded, want e-props-correct error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestElementDeclarationIdentityConstraintsAccessorDoesNotAlias(t *testing.T) {
	ic, err := xsd.NewIdentityConstraint(xsderr.Loc{}, xsd.QName{Local: "k"}, xsd.IdentityConstraintKey, xp("."), []xsd.XPathExpression{xp("@a")}, nil, nil)
	if err != nil {
		t.Fatalf("NewIdentityConstraint: %v", err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, false, []xsd.IdentityConstraint{ic}, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	first := e.IdentityConstraints()
	first[0] = xsd.IdentityConstraint{}
	if second := e.IdentityConstraints(); second[0].Name() != (xsd.QName{Local: "k"}) {
		t.Errorf("IdentityConstraints() returned an aliased slice: got %v", second[0].Name())
	}
}

func TestElementDeclarationSliceAccessorsDoNotAlias(t *testing.T) {
	head := xsd.QName{Local: "head"}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, false, nil, []xsd.QName{head}, []xsd.DerivationMethod{xsd.DerivationExtension}, false, []xsd.DerivationMethod{xsd.DerivationRestriction}, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	e.SubstitutionGroupAffiliationNames()[0] = xsd.QName{Local: "tampered"}
	if got := e.SubstitutionGroupAffiliationNames(); got[0] != head {
		t.Errorf("SubstitutionGroupAffiliationNames() aliased: got %v", got[0])
	}
	e.SubstitutionGroupExclusions()[0] = xsd.DerivationRestriction
	if got := e.SubstitutionGroupExclusions(); got[0] != xsd.DerivationExtension {
		t.Errorf("SubstitutionGroupExclusions() aliased: got %v", got[0])
	}
	e.DisallowedSubstitutions()[0] = xsd.DerivationExtension
	if got := e.DisallowedSubstitutions(); got[0] != xsd.DerivationRestriction {
		t.Errorf("DisallowedSubstitutions() aliased: got %v", got[0])
	}
}

func TestElementDeclarationDoesNotAliasConstructorSlices(t *testing.T) {
	affs := []xsd.QName{{Local: "head"}}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, false, nil, affs, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	affs[0] = xsd.QName{Local: "tampered"}
	if got := e.SubstitutionGroupAffiliationNames(); got[0] != (xsd.QName{Local: "head"}) {
		t.Errorf("ElementDeclaration aliased the constructor slice: got %v", got[0])
	}
}

func TestElementDeclarationAnnotationsRoundTripAndNil(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, false, nil, nil, nil, false, nil, anns)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	if got := e.Annotations(); len(got) != 1 || got[0].Documentation()[0].Content() != "first" {
		t.Errorf("Annotations() = %+v, want one with content first", got)
	}

	bare, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.QName{Local: "T"}, nil, xsd.ScopeGlobal, nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	if got := bare.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty {annotations}", got)
	}
}
