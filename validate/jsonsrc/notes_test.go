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
// and rejects a second member. The top-level elements a member of another's
// object names by key charge their own content at that member's location
// beside the parent's charge there: item of a simple type, num of xs:int, eo
// with element-only content, e0 with empty content, and ra with element-only
// content and a required attribute, an array item of which lst's content
// model takes. tt's {type definition} is ·xs:anyType·, whose content is
// mixed, and its {type table} selects eoT's element-only content.
const notedSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="abs" abstract="true">
    <xs:sequence><xs:element name="z" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="t" type="abs"/>
  <xs:element name="item" type="xs:string"/>
  <xs:element name="num" type="xs:int"/>
  <xs:element name="eo">
    <xs:complexType>
      <xs:sequence><xs:element name="q" type="xs:string"/></xs:sequence>
    </xs:complexType>
  </xs:element>
  <xs:element name="e0"><xs:complexType/></xs:element>
  <xs:element name="ra">
    <xs:complexType>
      <xs:sequence><xs:element name="q" type="xs:string"/></xs:sequence>
      <xs:attribute name="req" use="required"/>
    </xs:complexType>
  </xs:element>
  <xs:complexType name="eoT">
    <xs:sequence><xs:element name="q" type="xs:string"/></xs:sequence>
  </xs:complexType>
  <xs:element name="tt"><xs:alternative type="eoT"/></xs:element>
  <xs:element name="lst">
    <xs:complexType>
      <xs:sequence><xs:element ref="ra" maxOccurs="unbounded"/></xs:sequence>
    </xs:complexType>
  </xs:element>
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

// scalarExpectedHint is the clause a scalarExpected note appends for the member
// key of the object of owner.
func scalarExpectedHint(key, owner string) string {
	return `; JSON member "` + key + `" is a member of the object of "` + owner + `", whose type is a simple type: scalar expected, {"` + owner + `":<string, number or boolean>}`
}

// unnamedRHint is the clause an unnamed note appends for the member key of the
// object of notedSchema's r.
func unnamedRHint(key string) string {
	return `; JSON member "` + key + `" names no child element declaration and no attribute use of the type of "r", whose members are keyed "n"`
}

// charactersHint is the clause a characters note appends for the member key
// whose scalar's lexical form is lexical.
func charactersHint(key, lexical string) string {
	return `; JSON member "` + key + `" has the scalar value "` + lexical + `", where the type of its element admits no character content: object expected, {"` + key + `":{...}}`
}

// TestNoteHints pins the clause Validate appends to a charge the engine makes
// at a member the mapping notes, and that it appends none to a charge no note
// answers, the parent's or the element's own at a rule and location a noted
// charge shares included. Each charge's rule and location are the
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
		{"own cvc-type at a second member of a simple parent", notedSchema, `{"item":{"y":"b","num":"abc"}}`, []charge{
			{ruleCvcType, ":1:10", scalarExpectedHint("y", "item")},
			{ruleCvcType, ":1:18", ""},
		}},
		{"own cvc-type clause 3.1.3 at the first member of a simple parent", notedSchema, `{"item":{"num":"abc"}}`, []charge{
			{ruleCvcType, ":1:10", scalarExpectedHint("num", "item")},
			{ruleCvcType, ":1:10", ""},
		}},
		{"own cvc-type clause 2 at the first member of a simple parent", notedSchema, `{"item":{"t":{"z":"1"}}}`, []charge{
			{ruleCvcType, ":1:10", scalarExpectedHint("t", "item")},
			{ruleCvcType, ":1:10", ""},
		}},
		{"own cvc-complex-content at an unnamed member", notedSchema, `{"r":{"n":1,"eo":{}}}`, []charge{
			{ruleCvcComplexContent, ":1:13", unnamedRHint("eo")},
			{ruleCvcComplexContent, ":1:13", ""},
		}},
		{"own cvc-type and cvc-complex-content at an unnamed member", notedSchema, `{"r":{"n":1,"t":{}}}`, []charge{
			{ruleCvcComplexContent, ":1:13", unnamedRHint("t")},
			{ruleCvcType, ":1:13", ""},
			{ruleCvcComplexContent, ":1:13", ""},
		}},
		{"unnamed member after the parent's content is charged", notedSchema, `{"r":{"n":1,"n":2,"eo":{}}}`, []charge{
			{ruleCvcComplexContent, ":1:13", ""},
			{ruleCvcComplexContent, ":1:19", ""},
		}},
		{"unnamed array item with a scalar", notedSchema, `{"r":{"n":1,"eo":["x"]}}`, []charge{
			{ruleCvcComplexContent, ":1:19", unnamedRHint("eo")},
			{ruleCvcComplexType, ":1:19", charactersHint("eo", "x")},
		}},
		{"array item with a scalar under a simple parent", notedSchema, `{"item":{"eo":["x"]}}`, []charge{
			{ruleCvcType, ":1:16", scalarExpectedHint("eo", "item")},
			{ruleCvcComplexType, ":1:16", charactersHint("eo", "x")},
		}},
		{"array item with a scalar under an empty parent", notedSchema, `{"e0":{"eo":["x"]}}`, []charge{
			{ruleCvcComplexType, ":1:14", ""},
			{ruleCvcComplexType, ":1:14", charactersHint("eo", "x")},
		}},
		{"scalar root whose {type table} selects element-only content", notedSchema, `{"tt":"x"}`, []charge{
			{ruleCvcComplexType, ":1:7", charactersHint("tt", "x")},
		}},
		{"array item with a scalar missing a required attribute", notedSchema, `{"lst":{"ra":["x"]}}`, []charge{
			{ruleCvcComplexType, ":1:15", ""},
			{ruleCvcComplexType, ":1:15", charactersHint("ra", "x")},
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
