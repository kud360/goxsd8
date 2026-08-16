package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// taRef is the by-name {type definition} slot most of these tests carry.
func taRef(local string) xsd.TypeDefinitionOrRef {
	return xsd.TypeDefinitionRef{Name: xsd.QName{Local: local}}
}

// taOf builds a Type Alternative, failing the test on a rejection.
func taOf(t *testing.T, test *xsd.XPathExpression, typeDefinition xsd.TypeDefinitionOrRef, annotations []xsd.Annotation) xsd.TypeAlternative {
	t.Helper()
	ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, test, typeDefinition, annotations)
	if err != nil {
		t.Fatalf("NewTypeAlternative: %v", err)
	}
	return ta
}

func TestNewTypeAlternativeTestPresent(t *testing.T) {
	test := xsd.NewXPathExpression("@a > 0", []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:ns")}, strptr("urn:dflt"), nil)
	ta := taOf(t, &test, xsd.TypeDefinitionRef{Name: xsd.QName{Space: "urn:t", Local: "Even"}}, nil)

	got, ok := ta.Test()
	if !ok {
		t.Fatal("Test() ok = false, want true for a present {test}")
	}
	if got.Expression() != "@a > 0" {
		t.Errorf("Test().Expression() = %q, want %q", got.Expression(), "@a > 0")
	}
	binds := got.NamespaceBindings()
	if len(binds) != 1 || binds[0].Prefix() != "p" || binds[0].Namespace() != "urn:ns" {
		t.Errorf("Test().NamespaceBindings() = %+v, want [p=urn:ns]", binds)
	}
	if ns, ok := got.DefaultNamespace(); !ok || ns != "urn:dflt" {
		t.Errorf("Test().DefaultNamespace() = (%q, %v), want (%q, true)", ns, ok, "urn:dflt")
	}
}

func TestNewTypeAlternativeTestAbsent(t *testing.T) {
	ta := taOf(t, nil, taRef("Default"), nil)

	if got, ok := ta.Test(); ok {
		t.Errorf("Test() = (%+v, true), want (_, false) for the absent \"otherwise\" alternative", got)
	}
}

// TestTypeAlternativeTypeDefinitionRoundTrip pins that the {type definition}
// slot reads back as the very arm it was built with — §3.12.2 declare-ta's two
// arms, plus the SubstitutionGroupHeadTypeRef a synthesized {default type
// definition} copies out of the declaring element's own slot (§3.3.2.1).
func TestTypeAlternativeTypeDefinitionRoundTrip(t *testing.T) {
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{}, nil, xsd.SimpleTypeRef{Name: xsd.QName{Local: "string"}}, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	for _, tc := range []struct {
		name string
		slot xsd.TypeDefinitionOrRef
	}{
		{"by name", xsd.TypeDefinitionRef{Name: xsd.QName{Space: "urn:t", Local: "Even"}}},
		{"inline anonymous simple type", xsd.InlineTypeDefinition{Definition: st}},
		{"substitution group head", xsd.SubstitutionGroupHeadTypeRef{Head: xsd.QName{Local: "head"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := taOf(t, nil, tc.slot, nil).TypeDefinition(); got != tc.slot {
				t.Errorf("TypeDefinition() = %#v, want %#v", got, tc.slot)
			}
		})
	}
}

// TestNewTypeAlternativeRejectsIllegalSlots pins the states §3.12.1's Required
// {type definition} and TypeDefinitionOrRef's own invariants forbid. The nil
// case is this constructor's own rejection; the rest are
// checkTypeDefinitionOrRef's, reached through the type-alternative slot.
func TestNewTypeAlternativeRejectsIllegalSlots(t *testing.T) {
	named, err := xsd.NewComplexType(xsderr.Loc{}, xsd.QName{Local: "T"}, xsd.QName{Local: "anyType"}, nil,
		xsd.DerivationRestriction, false, nil, nil, nil, xsd.EmptyContent{}, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewComplexType: %v", err)
	}
	for _, tc := range []struct {
		name string
		slot xsd.TypeDefinitionOrRef
	}{
		{"absent slot", nil},
		{"zero-named reference", xsd.TypeDefinitionRef{}},
		{"empty inline definition", xsd.InlineTypeDefinition{}},
		{"inline NAMED definition", xsd.InlineTypeDefinition{Definition: named}},
		{"zero-named head", xsd.SubstitutionGroupHeadTypeRef{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := xsd.NewTypeAlternative(xsderr.Loc{}, nil, tc.slot, nil)
			if err == nil {
				t.Fatal("NewTypeAlternative succeeded, want a component-invariant rejection")
			}
			assertRule(t, err, xsderr.RuleComponentInvariant)
		})
	}
}

func TestNewTypeAlternativeAnnotationsRoundTrip(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "second")}, nil),
	}
	ta := taOf(t, nil, taRef("T"), anns)

	got := ta.Annotations()
	if len(got) != 2 {
		t.Fatalf("Annotations() len = %d, want 2", len(got))
	}
	if docs := got[0].Documentation(); len(docs) != 1 || docs[0].Content() != "first" {
		t.Errorf("Annotations()[0] documentation = %+v, want content %q", docs, "first")
	}
}

func TestTypeAlternativeAnnotationsNilWhenEmpty(t *testing.T) {
	if got := taOf(t, nil, taRef("T"), nil).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for nil input", got)
	}
	if got := taOf(t, nil, taRef("T"), []xsd.Annotation{}).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty-slice input", got)
	}
}

func TestTypeAlternativeDoesNotAliasConstructorAnnotations(t *testing.T) {
	anns := []xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)}
	ta := taOf(t, nil, taRef("T"), anns)

	// Mutate the ORIGINAL backing array.
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := ta.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("TypeAlternative aliased the constructor slice: got %q, want %q", docs[0].Content(), "keep")
	}
}

func TestTypeAlternativeAnnotationsAccessorDoesNotAlias(t *testing.T) {
	ta := taOf(t, nil, taRef("T"),
		[]xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)})

	// Mutate the RETURNED slice; a second call must be unaffected.
	ta.Annotations()[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := ta.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("Annotations() returned an aliased slice: got %q, want %q", docs[0].Content(), "keep")
	}
}
