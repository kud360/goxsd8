package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// refInheritableDoc is a schema with a top-level attribute a carrying global
// (the whole inheritable="…" attribute, or "" for none) and complex type T whose
// one attribute use is <attribute ref="tns:a"> carrying own likewise.
func refInheritableDoc(global, own string) string {
	return wrap("urn:x", `<xs:attribute name="a" type="xs:string"`+global+`/>`+
		`<xs:complexType name="T"><xs:sequence/><xs:attribute ref="tns:a"`+own+`/></xs:complexType>`)
}

// TestRefAttributeUseInheritable pins the {inheritable} of an Attribute Use the
// ref.att.local mapping (§3.2.2.3) builds: the ·actual value· of the use's own
// inheritable attribute if present, otherwise the referenced declaration's
// {inheritable}. The first row is the defect: the producer read an absent
// attribute as false whatever the declaration said (#1682).
func TestRefAttributeUseInheritable(t *testing.T) {
	for _, tc := range []struct {
		name        string
		global, own string
		want        bool
	}{
		{"absent over a true declaration", ` inheritable="true"`, "", true},
		{"absent over a declaration with none", "", "", false},
		{"false over a true declaration (cta0012.xsd)", ` inheritable="true"`, ` inheritable="false"`, false},
		{"true over a declaration with none (cta0011.xsd)", "", ` inheritable="true"`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, refInheritableDoc(tc.global, tc.own))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			uses := topComplexTypeIn(t, s, xq("T")).AttributeUses()
			if len(uses) != 1 {
				t.Fatalf("{attribute uses} = %d, want 1", len(uses))
			}
			if got := s.ResolvedInheritable(uses[0]); got != tc.want {
				t.Errorf("Attribute Use {inheritable} = %t, want %t (§3.2.2.3 ref.att.local)", got, tc.want)
			}
		})
	}
}

// TestRestrictionOfRefUseComparesResolvedInheritable is derivation-ok-restriction
// clause 3 reaching loc-testSubP clause 5.3, G.{inheritable} = S.{inheritable},
// where the base's use is a ref with no inheritable over a declaration whose
// {inheritable} is true: G's is true (§3.2.2.3 ref.att.local), so a restriction
// writing inheritable="true" is accepted and one writing "false" is rejected.
func TestRestrictionOfRefUseComparesResolvedInheritable(t *testing.T) {
	doc := func(restr string) string {
		return wrap("urn:x", `<xs:attribute name="a" type="xs:string" inheritable="true"/>`+
			`<xs:complexType name="B"><xs:attribute ref="tns:a"/></xs:complexType>`+
			`<xs:complexType name="R"><xs:complexContent><xs:restriction base="tns:B">`+
			`<xs:attribute ref="tns:a" inheritable="`+restr+`"/></xs:restriction></xs:complexContent></xs:complexType>`)
	}
	if _, err := produce(t, doc("true")); err != nil {
		t.Errorf(`restriction writing inheritable="true": %v, want accepted (loc-testSubP clause 5.3)`, err)
	}
	_, err := produce(t, doc("false"))
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf(`restriction writing inheritable="false": error = %v, want an *xsderr.Error`, err)
	}
	if xe.Rule != "derivation-ok-restriction" {
		t.Fatalf("rule = %q, want derivation-ok-restriction (%v)", xe.Rule, err)
	}
	const want = "has {inheritable} = false there and true in the base, and loc-testSubP clause 5.3"
	if !strings.Contains(xe.Msg, want) {
		t.Errorf("message = %q, want it to contain %q", xe.Msg, want)
	}
}
