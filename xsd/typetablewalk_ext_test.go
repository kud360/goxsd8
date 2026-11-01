package xsd_test

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsderr"
)

// parseTypeTableSchema parses body as line 2 of the one schema document s.xsd,
// whose xs prefix is bound to the XSD namespace, and returns parser.Parse's
// error.
func parseTypeTableSchema(t *testing.T, body string) error {
	t.Helper()
	doc := "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n" + body + "\n</xs:schema>\n"
	resolve := loader.ResolverFunc(func(_, location string) (io.ReadCloser, string, error) {
		if location != "s.xsd" {
			return nil, "", fmt.Errorf("no document %q", location)
		}
		return io.NopCloser(strings.NewReader(doc)), location, nil
	})
	_, err := parser.Parse("s.xsd", parser.WithResolver(resolve))
	return err
}

// TestParseChargesTypeTableOwnedComponents pins that parser.Parse charges a
// component a {type table} owns, or one nested under such a component, under
// the rule it breaks there, as it does in every other slot: Phase D's
// st-props-correct clause 1 for a restriction of xs:anySimpleType (absent
// {variety}) or of xs:anyAtomicType (absent {primitive type definition}),
// Phase E's au-props-correct clause 2 and e-props-correct clause 2, and
// e-props-correct clause 7 for a type table nested in an alternative's type.
// Each row but the src-resolve ones was accepted before componentWalk's
// walkElementDeclaration entered the table, and msg pins which charge fired.
// The src-resolve rows pin the context the shared descent now gives a by-name
// Type Alternative slot, which Phase A charged before the descent entered the
// table too.
func TestParseChargesTypeTableOwnedComponents(t *testing.T) {
	const (
		absentVariety = "simple type restricting {http://www.w3.org/2001/XMLSchema}anySimpleType has an absent {variety}"
		noPrimitive   = "atomic simple type may not name {http://www.w3.org/2001/XMLSchema}anyAtomicType as its {base type definition} unless it is a primitive datatype"
		restrictAny   = `<xs:simpleType><xs:restriction base="xs:anySimpleType"/></xs:simpleType>`
		restrictAtom  = `<xs:simpleType><xs:restriction base="xs:anyAtomicType"/></xs:simpleType>`
	)
	tests := []struct {
		name string
		body string
		rule xsderr.Rule
		msg  string
	}{
		// P5
		{"alternative restricts anySimpleType",
			`<xs:element name="e" type="xs:anySimpleType"><xs:alternative test="@k">` + restrictAny + `</xs:alternative></xs:element>`,
			"st-props-correct", absentVariety},
		// P6
		{"alternative restricts anyAtomicType",
			`<xs:element name="e" type="xs:anySimpleType"><xs:alternative test="@k">` + restrictAtom + `</xs:alternative></xs:element>`,
			"st-props-correct", noPrimitive},
		// P7
		{"default restricts anyAtomicType",
			`<xs:element name="e" type="xs:anySimpleType"><xs:alternative>` + restrictAtom + `</xs:alternative></xs:element>`,
			"st-props-correct", noPrimitive},
		// P7b
		{"default after a tested alternative restricts anyAtomicType",
			`<xs:element name="e" type="xs:anySimpleType"><xs:alternative test="@k" type="xs:string"/><xs:alternative>` + restrictAtom + `</xs:alternative></xs:element>`,
			"st-props-correct", noPrimitive},
		// P10
		{"two-step chain under an alternative",
			`<xs:element name="e" type="xs:anySimpleType"><xs:alternative test="@k"><xs:simpleType><xs:restriction>` + restrictAtom + `</xs:restriction></xs:simpleType></xs:alternative></xs:element>`,
			"st-props-correct", noPrimitive},
		// P11
		{"local element in an alternative's complex type",
			`<xs:element name="e"><xs:alternative test="@k"><xs:complexType><xs:sequence><xs:element name="l">` + restrictAny + `</xs:element></xs:sequence></xs:complexType></xs:alternative></xs:element>`,
			"st-props-correct", absentVariety},
		// P11b
		{"attribute in an alternative's complex type",
			`<xs:element name="e"><xs:alternative test="@k"><xs:complexType><xs:attribute name="a">` + restrictAny + `</xs:attribute></xs:complexType></xs:alternative></xs:element>`,
			"st-props-correct", absentVariety},
		// P11d
		{"type table nested in an alternative's complex type",
			`<xs:element name="e"><xs:alternative test="@k"><xs:complexType><xs:sequence><xs:element name="l" type="xs:anySimpleType"><xs:alternative test="@j">` + restrictAtom + `</xs:alternative></xs:element></xs:sequence></xs:complexType></xs:alternative></xs:element>`,
			"st-props-correct", noPrimitive},
		// P19
		{"attribute default invalid in an alternative's complex type",
			`<xs:element name="e"><xs:alternative test="@k"><xs:complexType><xs:attribute name="a" type="xs:int" default="abc"/></xs:complexType></xs:alternative></xs:element>`,
			"au-props-correct", "au-props-correct clause 2"},
		// P20
		{"element default invalid in an alternative's complex type",
			`<xs:element name="e"><xs:alternative test="@k"><xs:complexType><xs:sequence><xs:element name="l" type="xs:int" default="abc"/></xs:sequence></xs:complexType></xs:alternative></xs:element>`,
			"e-props-correct", "e-props-correct clause 2"},
		// P22
		{"nested alternative not substitutable in an alternative's complex type",
			`<xs:element name="e"><xs:alternative test="@k"><xs:complexType><xs:sequence><xs:element name="l" type="xs:int"><xs:alternative test="@j" type="xs:string"/></xs:element></xs:sequence></xs:complexType></xs:alternative></xs:element>`,
			"e-props-correct", "e-props-correct clause 7"},
		{"alternative names no type",
			`<xs:element name="e" type="xs:anySimpleType"><xs:alternative test="@k" type="Missing"/></xs:element>`,
			"src-resolve", "element declaration e {type table}.{alternatives}[0].{type definition} references type Missing"},
		{"default names no type",
			`<xs:element name="e" type="xs:anySimpleType"><xs:alternative test="@k" type="xs:string"/><xs:alternative type="Missing"/></xs:element>`,
			"src-resolve", "element declaration e {type table}.{default type definition}.{type definition} references type Missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseTypeTableSchema(t, tt.body)
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("Parse = %v, want an *xsderr.Error charged %s", err, tt.rule)
			}
			if xe.Rule != tt.rule {
				t.Fatalf("Parse rule = %s, want %s: %v", xe.Rule, tt.rule, err)
			}
			if !strings.Contains(xe.Msg, tt.msg) {
				t.Errorf("Parse message %q does not contain %q", xe.Msg, tt.msg)
			}
		})
	}
}

// TestParseAcceptsWellFormedTypeTableOwnedComponents is the guard over
// TestParseChargesTypeTableOwnedComponents: well-formed simple and complex types
// an <alternative> owns, and a well-formed type table nested in one, still
// parse, so the descent into a {type table} charges nothing a valid schema
// holds.
func TestParseAcceptsWellFormedTypeTableOwnedComponents(t *testing.T) {
	const restrictString = `<xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`
	tests := []struct{ name, body string }{
		{"alternative restricts string",
			`<xs:element name="e" type="xs:string"><xs:alternative test="@k">` + restrictString + `</xs:alternative></xs:element>`},
		{"default restricts string",
			`<xs:element name="e" type="xs:string"><xs:alternative test="@k" type="xs:string"/><xs:alternative>` + restrictString + `</xs:alternative></xs:element>`},
		{"alternative's complex type with well-formed members",
			`<xs:element name="e"><xs:alternative test="@k"><xs:complexType><xs:sequence><xs:element name="l" default="ab">` + restrictString +
				`</xs:element><xs:element name="m" type="xs:int"><xs:alternative test="@j" type="xs:short"/></xs:element></xs:sequence><xs:attribute name="a" type="xs:int" default="7"/></xs:complexType></xs:alternative></xs:element>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := parseTypeTableSchema(t, tt.body); err != nil {
				t.Fatalf("Parse = %v, want success", err)
			}
		})
	}
}
