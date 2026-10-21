package codegen

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/loader"
	xsdparser "github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// parseString parses doc as the one schema document at location s.xsd.
func parseString(t *testing.T, doc string) *xsd.Schema {
	t.Helper()
	resolve := loader.ResolverFunc(func(_, location string) (io.ReadCloser, string, error) {
		if location != "s.xsd" {
			return nil, "", fmt.Errorf("no document %q", location)
		}
		return io.NopCloser(strings.NewReader(doc)), location, nil
	})
	schema, err := xsdparser.Parse("s.xsd", xsdparser.WithResolver(resolve))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return schema
}

// TestGenerateDeclines pins each construct outside the slice: Generate
// returns a plain error wrapping errors.ErrUnsupported that names the
// construct and its Loc, and writes nothing. Each body is one line, so every
// Loc is on line 2.
func TestGenerateDeclines(t *testing.T) {
	const str = `type="xs:string"`
	tests := []struct {
		name, body, construct string
	}{
		{"choice", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element>`,
			"with a choice compositor"},
		{"all", `<xs:element name="e"><xs:complexType><xs:all><xs:element name="a" ` + str + `/></xs:all></xs:complexType></xs:element>`,
			"with a all compositor"},
		{"nested group", `<xs:element name="e"><xs:complexType><xs:sequence><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:sequence></xs:complexType></xs:element>`,
			"a nested sequence model group particle"},
		{"group ref particle", `<xs:group name="g"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:group><xs:element name="e"><xs:complexType><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element>`,
			"a <group ref> particle to g"},
		{"group ref content", `<xs:group name="g"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:group><xs:element name="e"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>`,
			"whose content particle is a <group ref>"},
		{"element ref", `<xs:element name="h" ` + str + `/><xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>`,
			"an <element ref> particle to h"},
		{"wildcard", `<xs:element name="e"><xs:complexType><xs:sequence><xs:any/></xs:sequence></xs:complexType></xs:element>`,
			"a wildcard particle"},
		{"attribute", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:attribute name="t"/></xs:complexType></xs:element>`,
			"with {attribute uses}"},
		{"attribute wildcard", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:anyAttribute/></xs:complexType></xs:element>`,
			"with an {attribute wildcard}"},
		{"mixed", `<xs:element name="e"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with mixed content"},
		{"simple content", `<xs:element name="e"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType></xs:element>`,
			"derived by extension"},
		{"simple content restriction", `<xs:complexType name="b"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType><xs:element name="e"><xs:complexType><xs:simpleContent><xs:restriction base="b"/></xs:simpleContent></xs:complexType></xs:element>`,
			"a restriction of b rather than of xs:anyType"},
		{"empty sequence", `<xs:element name="e"><xs:complexType><xs:sequence/></xs:complexType></xs:element>`,
			"with empty content"},
		{"maxOccurs 0 sequence", `<xs:element name="e"><xs:complexType><xs:sequence minOccurs="0" maxOccurs="0"><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with empty content"},
		{"open content", `<xs:element name="e"><xs:complexType><xs:openContent><xs:any/></xs:openContent><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with {open content}"},
		{"default open content", `<xs:defaultOpenContent><xs:any/></xs:defaultOpenContent><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"with {open content}"},
		{"extension", `<xs:complexType name="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType></xs:element>`,
			"derived by extension"},
		{"restriction of another base", `<xs:complexType name="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:element>`,
			"a restriction of b rather than of xs:anyType"},
		{"outer occurrence", `<xs:element name="e"><xs:complexType><xs:sequence minOccurs="0"><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"whose content particle occurs 0..1 times"},
		{"nillable", `<xs:element name="e" nillable="true" ` + str + `/>`,
			"element e with {nillable} true"},
		{"abstract element", `<xs:element name="e" abstract="true" ` + str + `/>`,
			"element e with {abstract} true"},
		{"value constraint", `<xs:element name="e" default="d" ` + str + `/>`,
			"element e with a {value constraint}"},
		{"type table", `<xs:element name="e" ` + str + `><xs:alternative test="true()" type="xs:token"/></xs:element>`,
			"element e with a {type table}"},
		{"substitution group", `<xs:element name="h" ` + str + `/><xs:element name="e" substitutionGroup="h" ` + str + `/>`,
			"element e with {substitution group affiliations}"},
		{"identity constraint", `<xs:element name="e" ` + str + `><xs:key name="k"><xs:selector xpath="."/><xs:field xpath="."/></xs:key></xs:element>`,
			"element e with {identity-constraint definitions}"},
		{"assertion", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:assert test="true()"/></xs:complexType></xs:element>`,
			"with {assertions}"},
		{"abstract complex type", `<xs:complexType name="c" abstract="true"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e" type="c"/>`,
			"complex type c of element e with {abstract} true"},
		{"anyType", `<xs:element name="e"/>`,
			"element e with {type definition} xs:anyType"},
		{"anySimpleType", `<xs:element name="e" type="xs:anySimpleType"/>`,
			"{http://www.w3.org/2001/XMLSchema}anySimpleType, a special type"},
		{"anyAtomicType", `<xs:element name="e" type="xs:anyAtomicType"/>`,
			"{http://www.w3.org/2001/XMLSchema}anyAtomicType, a special type"},
		{"list", `<xs:element name="e" type="xs:NMTOKENS"/>`,
			"{http://www.w3.org/2001/XMLSchema}NMTOKENS, a list type"},
		{"union", `<xs:element name="e"><xs:simpleType><xs:union memberTypes="xs:string xs:int"/></xs:simpleType></xs:element>`,
			"element e with {type definition} (anonymous), a union type"},
		{"uncovered primitive", `<xs:element name="e" type="xs:int"/>`,
			"whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}decimal"},
		{"named complex local", `<xs:complexType name="c"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l" type="c"/></xs:sequence></xs:complexType></xs:element>`,
			"local element l with named complex {type definition} c"},
		{"local element property", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l" nillable="true" ` + str + `/></xs:sequence></xs:complexType></xs:element>`,
			"element l with {nillable} true"},
		{"nested struct", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="l"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/></xs:choice></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			"complex type (anonymous) of element l with a choice compositor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseString(t, "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n"+tt.body+"\n</xs:schema>\n")
			out := filepath.Join(t.TempDir(), "out")
			err := Generate([]Target{{Schema: schema, Dir: out, Package: "p"}})
			if err == nil {
				t.Fatal("Generate succeeded, want a decline")
			}
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Errorf("error %v does not wrap errors.ErrUnsupported", err)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Errorf("error %v is an *xsderr.Error, want a plain error", err)
			}
			for _, want := range []string{"codegen: target 0: ", tt.construct, " at s.xsd:2:", "a generator limitation, not a schema fault"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("Dir %s exists after a decline (stat: %v)", out, err)
			}
		})
	}
}
