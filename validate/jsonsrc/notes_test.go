package jsonsrc

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// notedSchema adds, beside a root r, the shapes a note must not answer: a
// child of an abstract complex type, charged cvc-type clause 2 at its own key
// under a complex parent, and a declared child repeated past its maxOccurs,
// charged cvc-complex-content at a key its parent's type does name.
const notedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="abs" abstract="true">
    <xs:sequence><xs:element name="z" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="n" type="xs:int" minOccurs="0"/>
        <xs:element name="a" type="abs" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// TestNoteHints pins the clause Validate appends to a charge the engine makes
// at a member the mapping notes, and that it appends none where the note does
// not answer the charge's rule. Each row's charge is the engine's, rule and
// location unchanged; hint is the whole of what follows the engine's message,
// "" where nothing does.
func TestNoteHints(t *testing.T) {
	for _, c := range []struct {
		name, schema, doc string
		rule              xsderr.Rule
		loc               string
		hint              string
	}{
		{"attribute key spelled @kind", readmeSchema, `{"o:order":{"$xmlns":{"o":"urn:o"},"@kind":"x","item":["a","b"]}}`,
			ruleCvcComplexContent, jsonURI + ":1:36",
			`; JSON member "@kind" names no child element declaration and no attribute use of the type of "o:order", whose members are keyed "kind", "item"; an attribute member is keyed by the attribute's own name, "kind", not "@kind"`},
		{"unknown key", readmeSchema, `{"o:order":{"$xmlns":{"o":"urn:o"},"kind":"x","colour":"red","item":["a","b"]}}`,
			ruleCvcComplexContent, jsonURI + ":1:47",
			`; JSON member "colour" names no child element declaration and no attribute use of the type of "o:order", whose members are keyed "kind", "item"`},
		{"scalar where an object is expected", readmeSchema, `{"order":"just text"}`,
			ruleCvcComplexType, jsonURI + ":1:10",
			`; JSON member "order" has the scalar value "just text", where the type of its element admits no character content: object expected, {"order":{...}}`},
		{"object where a scalar is expected", readmeSchema, `{"order":{"item":"a","item":{"y":"b"}}}`,
			ruleCvcType, jsonURI + ":1:30",
			`; JSON member "y" is a member of the object of "item", whose type is a simple type: scalar expected, {"item":<string, number or boolean>}`},
		{"object where a scalar is expected, as an array item", readmeSchema, `{"order":{"item":["a",{"y":"b"}]}}`,
			ruleCvcType, jsonURI + ":1:24",
			`; JSON member "y" is a member of the object of "item", whose type is a simple type: scalar expected, {"item":<string, number or boolean>}`},
		{"abstract type under a complex parent", notedSchema, `{"r":{"a":{"z":"1"}}}`,
			ruleCvcType, jsonURI + ":1:7", ""},
		{"declared child repeated", notedSchema, `{"r":{"n":1,"n":2}}`,
			ruleCvcComplexContent, jsonURI + ":1:13", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, err := Validate(validatorOver(t, c.schema), strings.NewReader(c.doc), WithURI(jsonURI))
			if err != nil {
				t.Fatalf("Validate(%s): %v", c.doc, err)
			}
			v := res.Violations()
			if len(v) != 1 || v[0].Rule != c.rule || v[0].Loc.String() != c.loc {
				t.Fatalf("Validate(%s) violations = %v, want one %s charge at %s", c.doc, v, c.rule, c.loc)
			}
			_, hint, found := strings.Cut(v[0].Msg, "; JSON member ")
			if found {
				hint = "; JSON member " + hint
			}
			if hint != c.hint {
				t.Errorf("Validate(%s) message %q\nends with %q, want %q", c.doc, v[0].Msg, hint, c.hint)
			}
		})
	}
}
