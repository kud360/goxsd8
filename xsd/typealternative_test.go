package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// anonymousSimpleType is the ANONYMOUS simple type §3.12.2 declare-ta's inline
// arm yields for a <simpleType> child of an <alternative>: a restriction of a
// by-name base, with no {name} of its own.
func anonymousSimpleType(t *testing.T) *xsd.SimpleType {
	t.Helper()
	st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{}, xsd.RestrictionDerivation{},
		xsd.SimpleTypeRef{Name: xsd.QName{Local: "base"}}, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(anonymous): %v", err)
	}
	return st
}

// mustTypeAlternative builds a TypeAlternative whose {type definition} is the
// by-name reference to typeName, failing the test on any rejection.
func mustTypeAlternative(t *testing.T, test *xsd.XPathExpression, typeName xsd.QName, anns []xsd.Annotation) xsd.TypeAlternative {
	t.Helper()
	ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, test, xsd.TypeDefinitionRef{Name: typeName}, anns)
	if err != nil {
		t.Fatalf("NewTypeAlternative(%v): %v", typeName, err)
	}
	return ta
}

func TestNewTypeAlternativeTestPresent(t *testing.T) {
	test := xsd.NewXPathExpression("@a > 0", []xsd.NamespaceBinding{xsd.NewNamespaceBinding("p", "urn:ns")}, strptr("urn:dflt"), nil)
	ta := mustTypeAlternative(t, &test, xsd.QName{Space: "urn:t", Local: "Even"}, nil)

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
	ta := mustTypeAlternative(t, nil, xsd.QName{Space: "urn:t", Local: "Default"}, nil)

	if got, ok := ta.Test(); ok {
		t.Errorf("Test() = (%+v, true), want (_, false) for the absent \"otherwise\" alternative", got)
	}
}

func TestTypeAlternativeTypeDefinitionRoundTrip(t *testing.T) {
	name := xsd.QName{Space: "urn:t", Local: "Even"}
	ta := mustTypeAlternative(t, nil, name, nil)

	if got := ta.TypeDefinition(); got != (xsd.TypeDefinitionRef{Name: name}) {
		t.Errorf("TypeDefinition() = %+v, want TypeDefinitionRef{%+v}", got, name)
	}
}

// TestTypeAlternativeTypeDefinitionInlineRoundTrip pins §3.12.2 declare-ta's
// second arm: an <alternative> with a <complexType>/<simpleType> child OWNS the
// anonymous type, so the slot carries the component itself.
func TestTypeAlternativeTypeDefinitionInlineRoundTrip(t *testing.T) {
	st := anonymousSimpleType(t)
	ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, nil, xsd.InlineTypeDefinition{Definition: st}, nil)
	if err != nil {
		t.Fatalf("NewTypeAlternative with an inline type: %v", err)
	}

	inline, ok := ta.TypeDefinition().(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("TypeDefinition() = %T, want xsd.InlineTypeDefinition", ta.TypeDefinition())
	}
	if inline.Definition != xsd.TypeDefinition(st) {
		t.Errorf("TypeDefinition() wraps %+v, want the very component passed in", inline.Definition)
	}
}

// TestTypeAlternativeTypeDefinitionHeadInherited pins the arm §3.3.2.1's
// synthesized {default type definition} inherits verbatim from the declaring
// element (dcl.elt.common clause 3 through the {type table} row): the slot must
// admit it, or a substitutionGroup=-typed element with a tested final
// <alternative> would be unrepresentable.
func TestTypeAlternativeTypeDefinitionHeadInherited(t *testing.T) {
	head := xsd.QName{Space: "urn:t", Local: "base"}
	ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, nil, xsd.SubstitutionGroupHeadTypeRef{Head: head}, nil)
	if err != nil {
		t.Fatalf("NewTypeAlternative with a head-inherited type: %v", err)
	}
	if got := ta.TypeDefinition(); got != (xsd.SubstitutionGroupHeadTypeRef{Head: head}) {
		t.Errorf("TypeDefinition() = %+v, want SubstitutionGroupHeadTypeRef{%+v}", got, head)
	}
}

// TestNewTypeAlternativeRejectsAbsentType pins §3.12.1's Required {type
// definition}: nil is the sum's ABSENT encoding and has no place in this slot.
func TestNewTypeAlternativeRejectsAbsentType(t *testing.T) {
	if _, err := xsd.NewTypeAlternative(xsderr.Loc{}, nil, nil, nil); err == nil {
		t.Fatal("NewTypeAlternative(nil type) succeeded, want a component-invariant rejection")
	}
}

// TestNewTypeAlternativeRejectsIllegalArms pins that the slot delegates its
// arm-shape checks to checkTypeDefinitionOrRef rather than restating them.
func TestNewTypeAlternativeRejectsIllegalArms(t *testing.T) {
	for _, tc := range []struct {
		name string
		ref  xsd.TypeDefinitionOrRef
	}{
		{"zero-named reference", xsd.TypeDefinitionRef{}},
		{"empty inline", xsd.InlineTypeDefinition{}},
		{"zero-named head", xsd.SubstitutionGroupHeadTypeRef{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := xsd.NewTypeAlternative(xsderr.Loc{}, nil, tc.ref, nil); err == nil {
				t.Fatalf("NewTypeAlternative(%+v) succeeded, want a component-invariant rejection", tc.ref)
			}
		})
	}
}

func TestNewTypeAlternativeAnnotationsRoundTrip(t *testing.T) {
	anns := []xsd.Annotation{
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "first")}, nil),
		xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "second")}, nil),
	}
	ta := mustTypeAlternative(t, nil, xsd.QName{Local: "T"}, anns)

	got := ta.Annotations()
	if len(got) != 2 {
		t.Fatalf("Annotations() len = %d, want 2", len(got))
	}
	if docs := got[0].Documentation(); len(docs) != 1 || docs[0].Content() != "first" {
		t.Errorf("Annotations()[0] documentation = %+v, want content %q", docs, "first")
	}
}

func TestTypeAlternativeAnnotationsNilWhenEmpty(t *testing.T) {
	if got := mustTypeAlternative(t, nil, xsd.QName{Local: "T"}, nil).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for nil input", got)
	}
	if got := mustTypeAlternative(t, nil, xsd.QName{Local: "T"}, []xsd.Annotation{}).Annotations(); got != nil {
		t.Errorf("Annotations() = %v, want nil for empty-slice input", got)
	}
}

func TestTypeAlternativeDoesNotAliasConstructorAnnotations(t *testing.T) {
	anns := []xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)}
	ta := mustTypeAlternative(t, nil, xsd.QName{Local: "T"}, anns)

	// Mutate the ORIGINAL backing array.
	anns[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := ta.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("TypeAlternative aliased the constructor slice: got %q, want %q", docs[0].Content(), "keep")
	}
}

func TestTypeAlternativeAnnotationsAccessorDoesNotAlias(t *testing.T) {
	ta := mustTypeAlternative(t, nil, xsd.QName{Local: "T"},
		[]xsd.Annotation{xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "keep")}, nil)})

	// Mutate the RETURNED slice; a second call must be unaffected.
	ta.Annotations()[0] = xsd.NewAnnotation(nil, []xsd.Documentation{xsd.NewDocumentation(nil, nil, "tampered")}, nil)

	if docs := ta.Annotations()[0].Documentation(); docs[0].Content() != "keep" {
		t.Errorf("Annotations() returned an aliased slice: got %q, want %q", docs[0].Content(), "keep")
	}
}
