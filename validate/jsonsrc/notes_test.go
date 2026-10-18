package jsonsrc

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// notedSchema holds, beside readmeSchema's shapes, the ones a note must not
// answer or must answer differently: under a root r, a declared child n,
// which a repeat charges cvc-complex-content at a key r's type does name, and
// no declaration of t, a top-level element of an abstract complex type, which
// a member of r's object keyed t charges cvc-type clause 2 at its own key
// under a complex parent, beside the cvc-complex-content charge there; and a
// root w whose one particle is a lax wildcard, which names an unprefixed key
// and rejects a second member.
const notedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="abs" abstract="true">
    <xs:sequence><xs:element name="z" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="t" type="abs"/>
  <xs:element name="r">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="n" type="xs:int" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="w">
    <xs:complexType>
      <xs:sequence>
        <xs:any namespace="##local" processContents="lax"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// charge is one violation a TestNoteHints row expects: its rule, its location
// and the clause after the engine's message, "" where none follows it.
type charge struct {
	rule      xsderr.Rule
	loc, hint string
}

// TestNoteHints pins the clause Validate appends to a charge the engine makes
// at a member the mapping notes, and that it appends none where no note
// answers the charge's rule. Each charge's rule and location are the
// engine's.
func TestNoteHints(t *testing.T) {
	for _, c := range []struct {
		name, schema, doc string
		want              []charge
	}{
		{"attribute key spelled @kind", readmeSchema, `{"o:order":{"$xmlns":{"o":"urn:o"},"@kind":"x","item":["a","b"]}}`, []charge{
			{ruleCvcComplexContent, ":1:36", `; JSON member "@kind" names no child element declaration and no attribute use of the type of "o:order", whose members are keyed "kind", "item"; an attribute member is keyed by the attribute's own name, "kind", not "@kind"`},
		}},
		{"unknown key", readmeSchema, `{"o:order":{"$xmlns":{"o":"urn:o"},"kind":"x","colour":"red","item":["a","b"]}}`, []charge{
			{ruleCvcComplexContent, ":1:47", `; JSON member "colour" names no child element declaration and no attribute use of the type of "o:order", whose members are keyed "kind", "item"`},
		}},
		{"@ before a name no use carries", readmeSchema, `{"order":{"@colour":"red","item":"a"}}`, []charge{
			{ruleCvcComplexContent, ":1:11", `; JSON member "@colour" names no child element declaration and no attribute use of the type of "order", whose members are keyed "kind", "item"`},
		}},
		{"scalar where an object is expected", readmeSchema, `{"order":"just text"}`, []charge{
			{ruleCvcComplexType, ":1:10", `; JSON member "order" has the scalar value "just text", where the type of its element admits no character content: object expected, {"order":{...}}`},
		}},
		{"object where a scalar is expected", readmeSchema, `{"order":{"item":"a","item":{"y":"b"}}}`, []charge{
			{ruleCvcType, ":1:30", `; JSON member "y" is a member of the object of "item", whose type is a simple type: scalar expected, {"item":<string, number or boolean>}`},
		}},
		{"object where a scalar is expected, as an array item", readmeSchema, `{"order":{"item":["a",{"y":"b"}]}}`, []charge{
			{ruleCvcType, ":1:24", `; JSON member "y" is a member of the object of "item", whose type is a simple type: scalar expected, {"item":<string, number or boolean>}`},
		}},
		{"cvc-type at an unnamed member of a complex parent", notedSchema, `{"r":{"t":{"z":"1"}}}`, []charge{
			{ruleCvcComplexContent, ":1:7", `; JSON member "t" names no child element declaration and no attribute use of the type of "r", whose members are keyed "n"`},
			{ruleCvcType, ":1:7", ""},
		}},
		{"declared child repeated", notedSchema, `{"r":{"n":1,"n":2}}`, []charge{
			{ruleCvcComplexContent, ":1:13", ""},
		}},
		{"wildcard-named child rejected", notedSchema, `{"w":{"a":"1","b":"2"}}`, []charge{
			{ruleCvcComplexContent, ":1:15", ""},
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			res, err := Validate(validatorOver(t, c.schema), strings.NewReader(c.doc), WithURI(jsonURI))
			if err != nil {
				t.Fatalf("Validate(%s): %v", c.doc, err)
			}
			v := res.Violations()
			if len(v) != len(c.want) {
				t.Fatalf("Validate(%s) violations = %v, want %d", c.doc, v, len(c.want))
			}
			for i, w := range c.want {
				if v[i].Rule != w.rule || v[i].Loc.String() != jsonURI+w.loc {
					t.Fatalf("Validate(%s) violation %d = %v, want a %s charge at %s", c.doc, i+1, v[i], w.rule, jsonURI+w.loc)
				}
				_, hint, found := strings.Cut(v[i].Msg, "; JSON member ")
				if found {
					hint = "; JSON member " + hint
				}
				if hint != w.hint {
					t.Errorf("Validate(%s) violation %d message %q\nends with %q, want %q", c.doc, i+1, v[i].Msg, hint, w.hint)
				}
			}
		})
	}
}
