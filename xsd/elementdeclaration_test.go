package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// edLocalScope is a local {scope} whose {parent} names a containing complex type
// called container, for the tests that only need a non-global declaration.
func edLocalScope(t *testing.T) xsd.Scope {
	t.Helper()
	s, err := xsd.NewLocalScope(xsderr.Loc{}, xsd.ComplexTypeScopeParent{Name: xsd.QName{Local: "container"}})
	if err != nil {
		t.Fatalf("NewLocalScope: %v", err)
	}
	return s
}

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
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, name, xsd.TypeDefinitionRef{Name: typ}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration unexpected error: %v", err)
	}
	if e.Name() != name {
		t.Errorf("Name() = %v, want %v", e.Name(), name)
	}
	if got := e.TypeDefinition(); got != (xsd.TypeDefinitionOrRef)(xsd.TypeDefinitionRef{Name: typ}) {
		t.Errorf("TypeDefinition() = %v, want a TypeDefinitionRef naming %v", got, typ)
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
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "member"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, true, nil, []xsd.QName{head}, []xsd.DerivationMethod{xsd.DerivationExtension}, true, []xsd.DerivationMethod{xsd.DerivationSubstitution}, nil)
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
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, &tt, edLocalScope(t), &vc, false, nil, nil, nil, false, nil, nil)
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

// TestNewElementDeclarationRejectsAbsentName exercises e-props-correct clause 1
// for the {name} slot: the §3.3.1 tableau types it as a Required xs:NCName, and
// NCName's value space excludes the empty string, so a QName with an empty Local
// is not a legal {name} — with or without a namespace name. A present local name
// in no namespace stays legal (a zero Space is a present name, not an absent one).
func TestNewElementDeclarationRejectsAbsentName(t *testing.T) {
	tests := []struct {
		name    string
		qname   xsd.QName
		wantErr bool
	}{
		{"zero QName", xsd.QName{}, true},
		{"namespace with empty local", xsd.QName{Space: "urn:ns"}, true},
		{"no-namespace present local", xsd.QName{Local: "e"}, false},
		{"namespaced present local", xsd.QName{Space: "urn:ns", Local: "e"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewElementDeclaration(xsderr.Loc{}, tc.qname, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("NewElementDeclaration(%v) unexpected error: %v", tc.qname, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("NewElementDeclaration(%v) succeeded, want e-props-correct clause 1 error", tc.qname)
			}
			assertRule(t, err, "e-props-correct")
		})
	}
}

// TestNewGlobalScope pins §3.3.2.2 dcl.elt.global: {variety} global, {parent}
// ·absent·.
func TestNewGlobalScope(t *testing.T) {
	s := xsd.NewGlobalScope()
	if s.Variety() != xsd.ScopeGlobal {
		t.Errorf("Variety() = %v, want global", s.Variety())
	}
	parent, ok := s.Parent()
	if ok {
		t.Errorf("Parent() ok = true, want false for a global scope (got %#v)", parent)
	}
	if parent != nil {
		t.Errorf("Parent() = %#v, want nil for a global scope", parent)
	}
}

// TestNewLocalScopeCarriesParent pins §3.3.2.3 dcl.elt.local for both target
// kinds: {variety} is local (derived from {parent}'s presence, never stored) and
// {parent} reads back as the very variant it was built from — a Complex Type
// Definition for an <element> under a <complexType>, a Model Group Definition for
// one within a named <group>.
func TestNewLocalScopeCarriesParent(t *testing.T) {
	ct := xsd.QName{Space: "urn:ns", Local: "AddressType"}
	mgd := xsd.QName{Space: "urn:ns", Local: "addressGroup"}
	for _, tc := range []struct {
		name string
		want xsd.ElementScopeParent
	}{
		{"complex type definition", xsd.ComplexTypeScopeParent{Name: ct}},
		{"model group definition", xsd.ModelGroupScopeParent{Name: mgd}},
		{"anonymous complex type definition", xsd.AnonymousComplexTypeScopeParent{Owner: xsd.NewComponentID()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := xsd.NewLocalScope(xsderr.Loc{}, tc.want)
			if err != nil {
				t.Fatalf("NewLocalScope(%#v): %v", tc.want, err)
			}
			if s.Variety() != xsd.ScopeLocal {
				t.Errorf("Variety() = %v, want local", s.Variety())
			}
			got, ok := s.Parent()
			if !ok {
				t.Fatal("Parent() ok = false, want true for a local scope")
			}
			if got != tc.want {
				t.Errorf("Parent() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

// TestNewLocalScopeRejectsUnusableParent pins the two states NewLocalScope
// refuses (e-props-correct clause 1): an absent {parent}, which the §3.3.1
// tableau requires to be present when {variety} is local, and a variant that
// identifies nothing — an absent name on either by-NAME arm, or an unminted
// identity on the anonymous arm, neither of which could ever be followed.
func TestNewLocalScopeRejectsUnusableParent(t *testing.T) {
	for _, tc := range []struct {
		name   string
		parent xsd.ElementScopeParent
	}{
		{"absent parent", nil},
		{"unnamed complex type", xsd.ComplexTypeScopeParent{}},
		{"unnamed model group", xsd.ModelGroupScopeParent{}},
		{"unminted anonymous complex type", xsd.AnonymousComplexTypeScopeParent{}},
		{"namespace but no local name", xsd.ComplexTypeScopeParent{Name: xsd.QName{Space: "urn:ns"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewLocalScope(xsderr.Loc{}, tc.parent)
			if err == nil {
				t.Fatalf("NewLocalScope(%#v) succeeded, want an e-props-correct clause 1 rejection", tc.parent)
			}
			assertRule(t, err, "e-props-correct")
		})
	}
}

// TestElementDeclarationScopeRoundTrip is the containment round trip: a local
// element declaration nested in a named container names that container back, and
// the name is the one the container itself reports, so a consumer can go from the
// declaration to its {scope}.{parent} component with a schema lookup. The global
// declaration alongside it carries no {parent} at all (§3.3.2.2).
func TestElementDeclarationScopeRoundTrip(t *testing.T) {
	container, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "AddressType"},
		xsd.QName{Local: "anyType"}, nil, xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	group, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, nil, nil)
	if err != nil {
		t.Fatalf("NewModelGroup: %v", err)
	}
	groupDef, err := xsd.NewModelGroupDefinition(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "addressGroup"}, group, nil)
	if err != nil {
		t.Fatalf("NewModelGroupDefinition: %v", err)
	}

	for _, tc := range []struct {
		name   string
		parent xsd.ElementScopeParent
		want   xsd.QName
	}{
		{"in a complex type", xsd.ComplexTypeScopeParent{Name: container.Name()}, container.Name()},
		{"in a model group definition", xsd.ModelGroupScopeParent{Name: groupDef.Name()}, groupDef.Name()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scope, err := xsd.NewLocalScope(xsderr.Loc{}, tc.parent)
			if err != nil {
				t.Fatalf("NewLocalScope: %v", err)
			}
			e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "street"},
				xsd.TypeDefinitionRef{Name: xsd.QName{Local: "string"}}, nil, scope, nil, false, nil, nil, nil, false, nil, nil)
			if err != nil {
				t.Fatalf("NewElementDeclaration: %v", err)
			}
			if e.ScopeVariety() != xsd.ScopeLocal {
				t.Errorf("ScopeVariety() = %v, want local", e.ScopeVariety())
			}
			parent, ok := e.Scope().Parent()
			if !ok {
				t.Fatal("Scope().Parent() ok = false, want true for a local declaration")
			}
			var got xsd.QName
			switch p := parent.(type) {
			case xsd.ComplexTypeScopeParent:
				got = p.Name
			case xsd.ModelGroupScopeParent:
				got = p.Name
			default:
				t.Fatalf("Scope().Parent() = %T, want one of the by-NAME ElementScopeParent variants", parent)
			}
			if got != tc.want {
				t.Errorf("{scope}.{parent} names %s, want the container's own name %s", got, tc.want)
			}
			if parent != tc.parent {
				t.Errorf("Scope().Parent() = %#v, want %#v (the kind discriminant must survive too)", parent, tc.parent)
			}
		})
	}

	global, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Space: "urn:ns", Local: "shipTo"},
		xsd.TypeDefinitionRef{Name: xsd.QName{Local: "AddressType"}}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration(global): %v", err)
	}
	if _, ok := global.Scope().Parent(); ok {
		t.Error("a global declaration reports a {scope}.{parent}, want none (§3.3.2.2 dcl.elt.global)")
	}
}

func TestNewElementDeclarationRejectsLocalScopeWithAffiliations(t *testing.T) {
	head := xsd.QName{Local: "head"}
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, edLocalScope(t), nil, false, nil, []xsd.QName{head}, nil, false, nil, nil)
	if err == nil {
		t.Fatal("NewElementDeclaration(local scope + affiliations) succeeded, want e-props-correct clause 3 error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestNewElementDeclarationRejectsIllegalExclusion(t *testing.T) {
	// substitution is not a legal {substitution group exclusions} token.
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, []xsd.DerivationMethod{xsd.DerivationSubstitution}, false, nil, nil)
	if err == nil {
		t.Fatal("NewElementDeclaration(exclusion=substitution) succeeded, want e-props-correct error")
	}
	assertRule(t, err, "e-props-correct")
}

func TestNewElementDeclarationRejectsIllegalDisallowedSubstitution(t *testing.T) {
	// list is not a legal {disallowed substitutions} token.
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, []xsd.DerivationMethod{xsd.DerivationList}, nil)
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
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, []xsd.IdentityConstraint{ic}, nil, nil, false, nil, nil)
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
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, nil, []xsd.QName{head}, []xsd.DerivationMethod{xsd.DerivationExtension}, false, []xsd.DerivationMethod{xsd.DerivationRestriction}, nil)
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
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, nil, affs, nil, false, nil, nil)
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
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, anns)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	if got := e.Annotations(); len(got) != 1 || got[0].Documentation()[0].Content() != "first" {
		t.Errorf("Annotations() = %+v, want one with content first", got)
	}

	bare, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "e"}, xsd.TypeDefinitionRef{Name: xsd.QName{Local: "T"}}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclaration: %v", err)
	}
	if got := bare.Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty {annotations}", got)
	}
}

// edAnonType builds an ANONYMOUS complex type whose {context} names id — the
// shape §3.4.2.1 dcl.ctd.common gives an inline <complexType> child of an
// <element>.
func edAnonType(t *testing.T, context xsd.ComplexTypeContext) xsd.ComplexType {
	t.Helper()
	ct, err := xsd.NewAnonymousComplexType(xsderr.Loc{}, context, xsd.QName{Local: "anyType"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewAnonymousComplexType: %v", err)
	}
	return ct
}

// TestNewElementDeclarationOwningTypeMatchingIdentity pins the accepting case of
// the §3.4.2.1 dcl.ctd.common round trip: one identity minted for the inline
// construct, threaded into the type's {context} and into the declaration that
// owns it, so the two compare == and the slot reads back as the InlineTypeDefinition
// arm holding that very type.
func TestNewElementDeclarationOwningTypeMatchingIdentity(t *testing.T) {
	id := xsd.NewComponentID()
	ct := edAnonType(t, xsd.ElementDeclarationContext{Component: id})
	e, err := xsd.NewElementDeclarationOwningType(xsderr.Loc{}, id, xsd.QName{Local: "doc"}, ct,
		nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err != nil {
		t.Fatalf("NewElementDeclarationOwningType: %v", err)
	}
	inline, ok := e.TypeDefinition().(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("{type definition} = %#v, want the InlineTypeDefinition arm", e.TypeDefinition())
	}
	got, ok := inline.Definition.(xsd.ComplexType)
	if !ok {
		t.Fatalf("InlineTypeDefinition wraps %T, want the ComplexType passed in", inline.Definition)
	}
	context, ok := got.Context()
	if !ok {
		t.Fatal("the owned type lost its {context}")
	}
	// == and never reflect.DeepEqual: componentid_test.go pins that DeepEqual is
	// identity-blind, so a DeepEqual assertion here would accept a wrong context.
	if context.ID() != id {
		t.Error("the owned type's {context} is not the declaration's own identity")
	}
}

// TestNewElementDeclarationOwningTypeRejectsBadIdentity pins the three states the
// owning constructor makes unrepresentable, all charged component-invariant: an
// unminted owner identity, a {context} naming a DIFFERENT element declaration,
// and a ComplexTypeDefinitionContext in a slot §3.4.2.1 gives exactly one case
// for, which yields an Element Declaration.
func TestNewElementDeclarationOwningTypeRejectsBadIdentity(t *testing.T) {
	id := xsd.NewComponentID()
	for _, tc := range []struct {
		name string
		id   xsd.ComponentID
		ct   xsd.ComplexType
	}{
		{"unminted owner identity", xsd.ComponentID{}, edAnonType(t, xsd.ElementDeclarationContext{Component: id})},
		{"context names another declaration", id, edAnonType(t, xsd.ElementDeclarationContext{Component: xsd.NewComponentID()})},
		{"context is a complex type definition", id, edAnonType(t, xsd.ComplexTypeDefinitionContext{Component: id})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewElementDeclarationOwningType(xsderr.Loc{}, tc.id, xsd.QName{Local: "doc"}, tc.ct,
				nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
			if err == nil {
				t.Fatal("NewElementDeclarationOwningType succeeded, want a component-invariant rejection")
			}
			assertRule(t, err, xsderr.RuleComponentInvariant)
		})
	}
}

// TestNewElementDeclarationOwningTypeRejectsNamedType pins that the owning
// constructor is for ANONYMOUS types only: a NAMED complex type is reachable by
// name and so is always the TypeDefinitionRef arm. The verdict comes from the
// shared core's InlineTypeDefinition shape check, not from a duplicate of it.
func TestNewElementDeclarationOwningTypeRejectsNamedType(t *testing.T) {
	named, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "T"}, xsd.QName{Local: "anyType"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	_, err = xsd.NewElementDeclarationOwningType(xsderr.Loc{}, xsd.NewComponentID(), xsd.QName{Local: "doc"}, named,
		nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err == nil {
		t.Fatal("NewElementDeclarationOwningType accepted a NAMED complex type")
	}
	assertRule(t, err, xsderr.RuleComponentInvariant)
}

// TestNewElementDeclarationRejectsOwnedComplexType pins the other half of the
// partition: the plain constructor takes no identity, so it cannot check a
// {context} back-pointer and must refuse the shape outright rather than admit a
// possibly mis-pointing one.
func TestNewElementDeclarationRejectsOwnedComplexType(t *testing.T) {
	ct := edAnonType(t, xsd.ElementDeclarationContext{Component: xsd.NewComponentID()})
	_, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "doc"},
		xsd.InlineTypeDefinition{Definition: ct}, nil, xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil, nil)
	if err == nil {
		t.Fatal("NewElementDeclaration accepted an InlineTypeDefinition wrapping a ComplexType")
	}
	assertRule(t, err, xsderr.RuleComponentInvariant)
}
