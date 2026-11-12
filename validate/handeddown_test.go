package validate_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// handedXSD declares the top-level attributes g, inheritable, and n, not; Uses
// with a use of a, inheritable, b, not, g by reference with the use's
// inheritable false, n by reference with the use's inheritable true, d,
// inheritable and defaulting to "dv", and e, not inheritable and defaulting to
// "ev"; and Lax, Strict and Skip, each with only an {attribute wildcard} of that
// processContents.
const handedXSD = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:attribute name="g" type="xs:string" inheritable="true"/>
  <xs:attribute name="n" type="xs:string"/>
  <xs:complexType name="Uses">
    <xs:attribute name="a" type="xs:string" inheritable="true"/>
    <xs:attribute name="b" type="xs:string"/>
    <xs:attribute ref="g" inheritable="false"/>
    <xs:attribute ref="n" inheritable="true"/>
    <xs:attribute name="d" type="xs:string" default="dv" inheritable="true"/>
    <xs:attribute name="e" type="xs:string" default="ev"/>
  </xs:complexType>
  <xs:complexType name="Lax"><xs:anyAttribute processContents="lax"/></xs:complexType>
  <xs:complexType name="Strict"><xs:anyAttribute processContents="strict"/></xs:complexType>
  <xs:complexType name="Skip"><xs:anyAttribute processContents="skip"/></xs:complexType>
</xs:schema>`

// handedAttr is these rows' [validate.Attribute].
type handedAttr struct {
	name, value string
	loc         xsderr.Loc
}

func (a *handedAttr) Name() xsd.QName { return xsd.QName{Local: a.name} }
func (a *handedAttr) Value() string   { return a.value }
func (a *handedAttr) Loc() xsderr.Loc { return a.loc }

// hAt is the attribute name="value" at line 1, column col.
func hAt(name, value string, col int) validate.Attribute {
	return &handedAttr{name: name, value: value, loc: xsderr.Loc{URI: "doc.xml", Line: 1, Col: col}}
}

// handedE is the Loc HandedDown is handed as E's own.
var handedE = xsderr.Loc{URI: "doc.xml", Line: 7, Col: 3}

// handedRender is as, each member as name=value@line:col.
func handedRender(as []validate.Attribute) []string {
	out := []string{}
	for _, a := range as {
		out = append(out, fmt.Sprintf("%s=%s@%d:%d", a.Name().Local, a.Value(), a.Loc().Line, a.Loc().Col))
	}
	return out
}

// TestHandedDown pins [validate.Validator.HandedDown]'s composition, one row
// per arm of it: E's own ·potentially inherited· attributes in source order,
// then its ·defaulted attributes· whose use is inheritable, located at E, then
// each inherited member neither names.
func TestHandedDown(t *testing.T) {
	backend := strict.New()
	schema, err := parser.Parse("handed.xsd", parser.WithBackend(backend),
		parser.WithResolver(loader.Map(map[string]string{"handed.xsd": handedXSD})))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	v, err := validate.New(schema, backend)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	typ := func(ns, name string) xsd.TypeDefinition {
		td, ok := v.Schema().ResolvedType(xsd.TypeDefinitionRef{Name: xsd.QName{Space: ns, Local: name}})
		if !ok {
			t.Fatalf("no type %s", name)
		}
		return td
	}
	uses, lax, strictT, skip := typ("", "Uses"), typ("", "Lax"), typ("", "Strict"), typ("", "Skip")
	anyType := typ(xsd.XMLSchemaNS, "anyType")
	for _, tc := range []struct {
		name                  string
		governing             xsd.TypeDefinition
		attributes, inherited []validate.Attribute
		want                  []string
	}{
		{"a use's {inheritable} true and false", uses,
			[]validate.Attribute{hAt("b", "2", 5), hAt("a", "1", 9)}, nil,
			[]string{"a=1@1:9", "d=dv@7:3"}},
		{"a use's {inheritable} over its declaration's", uses,
			[]validate.Attribute{hAt("g", "1", 5), hAt("n", "2", 9)}, nil,
			[]string{"n=2@1:9", "d=dv@7:3"}},
		{"a specified attribute is not defaulted", uses,
			[]validate.Attribute{hAt("d", "x", 5)}, nil,
			[]string{"d=x@1:5"}},
		{"the top-level declaration through a lax wildcard", lax,
			[]validate.Attribute{hAt("g", "1", 5), hAt("n", "2", 9), hAt("x", "3", 13)}, nil,
			[]string{"g=1@1:5"}},
		{"the top-level declaration through a strict wildcard", strictT,
			[]validate.Attribute{hAt("g", "1", 5), hAt("n", "2", 9), hAt("x", "3", 13)}, nil,
			[]string{"g=1@1:5"}},
		{"a skip wildcard hands nothing down", skip,
			[]validate.Attribute{hAt("g", "1", 5)}, []validate.Attribute{hAt("q", "up", 2)},
			[]string{"q=up@1:2"}},
		{"the nearest ancestor's shadows", uses,
			[]validate.Attribute{hAt("a", "near", 5)}, []validate.Attribute{hAt("a", "far", 2), hAt("z", "up", 3)},
			[]string{"a=near@1:5", "d=dv@7:3", "z=up@1:3"}},
		{"a defaulted attribute shadows", uses,
			nil, []validate.Attribute{hAt("d", "far", 2)},
			[]string{"d=dv@7:3"}},
		{"a non-inheritable attribute shadows nothing", uses,
			[]validate.Attribute{hAt("b", "near", 5)}, []validate.Attribute{hAt("b", "far", 2)},
			[]string{"d=dv@7:3", "b=far@1:2"}},
		{"xs:anyType", anyType,
			[]validate.Attribute{hAt("g", "1", 5), hAt("n", "2", 9)}, []validate.Attribute{hAt("q", "up", 2)},
			[]string{"g=1@1:5", "q=up@1:2"}},
		{"nil governing is xs:anyType's answer", nil,
			[]validate.Attribute{hAt("g", "1", 5), hAt("n", "2", 9)}, []validate.Attribute{hAt("q", "up", 2)},
			[]string{"g=1@1:5", "q=up@1:2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attrs, inherited := slices.Clone(tc.attributes), slices.Clone(tc.inherited)
			got := v.HandedDown(tc.governing, handedE, tc.attributes, tc.inherited)
			if !slices.Equal(handedRender(got), tc.want) {
				t.Errorf("HandedDown = %v, want %v", handedRender(got), tc.want)
			}
			if !slices.Equal(tc.attributes, attrs) || !slices.Equal(tc.inherited, inherited) {
				t.Errorf("HandedDown modified its arguments")
			}
		})
	}

	// A specified member is the caller's own value, and E handing down
	// nothing of its own returns inherited itself.
	a := hAt("a", "1", 5)
	if got := v.HandedDown(uses, handedE, []validate.Attribute{a}, nil); len(got) == 0 || got[0] != a {
		t.Errorf("HandedDown(Uses, a) = %v, want a itself first", handedRender(got))
	}
	inherited := []validate.Attribute{hAt("q", "up", 2)}
	if got := v.HandedDown(skip, handedE, []validate.Attribute{hAt("g", "1", 5)}, inherited); len(got) != 1 || &got[0] != &inherited[0] {
		t.Errorf("HandedDown(Skip) = %v, want inherited itself", handedRender(got))
	}
}
