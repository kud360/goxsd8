package jsonsrc

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// typeSchema declares base, an element-only type holding a; derived, an
// extension of base adding the child b and the attribute d; other, unrelated
// to base, holding c; item, of type base; anything, of xs:anyType; and names,
// whose sequence holds a repeating xs:QName q, then inner, which holds q
// again, then an item. Local declarations are qualified.
const typeSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"
    targetNamespace="urn:t" xmlns="urn:t" elementFormDefault="qualified">
  <xs:complexType name="base">
    <xs:sequence>
      <xs:element name="a" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:complexType name="derived">
    <xs:complexContent>
      <xs:extension base="base">
        <xs:sequence>
          <xs:element name="b" type="xs:string"/>
        </xs:sequence>
        <xs:attribute name="d" type="xs:integer"/>
      </xs:extension>
    </xs:complexContent>
  </xs:complexType>
  <xs:complexType name="other">
    <xs:sequence>
      <xs:element name="c" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
  <xs:element name="item" type="base"/>
  <xs:element name="anything"/>
  <xs:element name="names">
    <xs:complexType>
      <xs:sequence>
        <xs:element name="q" type="xs:QName" minOccurs="0" maxOccurs="unbounded"/>
        <xs:element name="inner" minOccurs="0">
          <xs:complexType>
            <xs:sequence>
              <xs:element name="q" type="xs:QName"/>
            </xs:sequence>
          </xs:complexType>
        </xs:element>
        <xs:element ref="item" minOccurs="0"/>
      </xs:sequence>
    </xs:complexType>
  </xs:element>
</xs:schema>`

// TestTypeMember pins the "$type" arm: a "$type" member is the element's
// xsi:type, resolved against the "$xmlns" bindings in scope, and the members
// after it are classified under the type it names wherever that resolves.
// Each row agrees with its XML twin.
func TestTypeMember(t *testing.T) {
	v := validatorOver(t, typeSchema)
	const x = `xmlns="urn:t" ` + xsiDecl
	for _, tc := range []struct {
		name, json, xml string
		want            []xsderr.Rule
	}{
		{"derived", `{"item":{"$xmlns":{"t":"urn:t"},"$type":"t:derived","a":"x","b":"y"}}`,
			`<item ` + x + ` xmlns:t="urn:t" xsi:type="t:derived"><a>x</a><b>y</b></item>`, none},
		{"derived's own attribute", `{"item":{"$xmlns":{"t":"urn:t"},"$type":"t:derived","d":"1","a":"x","b":"y"}}`,
			`<item ` + x + ` xmlns:t="urn:t" xsi:type="t:derived" d="1"><a>x</a><b>y</b></item>`, none},
		{"unprefixed, in the declaration's namespace", `{"item":{"$type":"derived","a":"x","b":"y"}}`,
			`<item ` + x + ` xsi:type="derived"><a>x</a><b>y</b></item>`, none},
		{"collapsed", `{"item":{"$xmlns":{"t":"urn:t"},"$type":" t:derived\n","a":"x","b":"y"}}`,
			`<item ` + x + ` xmlns:t="urn:t" xsi:type=" t:derived&#10;"><a>x</a><b>y</b></item>`, none},
		{"an empty-prefix binding is shadowed", `{"item":{"$xmlns":{"":"urn:other"},"$type":"derived","a":"x","b":"y"}}`,
			`<item ` + x + ` xsi:type="derived"><a>x</a><b>y</b></item>`, none},
		{"inherited binding", `{"names":{"$xmlns":{"t":"urn:t"},"item":{"$type":"t:derived","a":"x","b":"y"}}}`,
			`<names ` + x + ` xmlns:t="urn:t"><item xsi:type="t:derived"><a>x</a><b>y</b></item></names>`, none},
		{"innermost binding wins", `{"names":{"$xmlns":{"t":"urn:x"},"item":{"$xmlns":{"t":"urn:t"},"$type":"t:derived","a":"x","b":"y"}}}`,
			`<names ` + x + ` xmlns:t="urn:x"><item xmlns:t="urn:t" xsi:type="t:derived"><a>x</a><b>y</b></item></names>`, none},
		{"not overriding", `{"item":{"$xmlns":{"t":"urn:t"},"$type":"t:other","c":"z"}}`,
			`<item ` + x + ` xmlns:t="urn:t" xsi:type="t:other"><c>z</c></item>`, []xsderr.Rule{"cvc-elt", "cvc-complex-content"}},
		{"naming no type", `{"item":{"$xmlns":{"t":"urn:t"},"$type":"t:missing","a":"x"}}`,
			`<item ` + x + ` xmlns:t="urn:t" xsi:type="t:missing"><a>x</a></item>`, attr},
		{"unbound prefix", `{"item":{"$xmlns":{"u":"urn:t"},"$type":"t:derived","a":"x"}}`,
			`<item ` + x + ` xmlns:u="urn:t" xsi:type="t:derived"><a>x</a></item>`, attr},
		{"xs:error is not nilled", `{"anything":{"$xmlns":{"xs":"http://www.w3.org/2001/XMLSchema"},"$type":"xs:error"}}`,
			`<anything ` + x + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:error"/>`, typ},
	} {
		t.Run(tc.name, func(t *testing.T) { twin(t, v, tc.json, tc.xml, tc.want) })
	}
}

// TestXmlnsMember pins the "$xmlns" arm for QName-valued content: a prefix a
// "$xmlns" binds resolves in its own element's content and its descendants',
// array items included, and nowhere above it. Each row agrees with its XML
// twin.
func TestXmlnsMember(t *testing.T) {
	v := validatorOver(t, typeSchema)
	for _, tc := range []struct {
		name, json, xml string
		want            []xsderr.Rule
	}{
		{"bound", `{"names":{"$xmlns":{"t":"urn:t"},"q":"t:foo"}}`,
			`<names xmlns="urn:t" xmlns:t="urn:t"><q>t:foo</q></names>`, none},
		{"unbound", `{"names":{"$xmlns":{"u":"urn:t"},"q":"t:foo"}}`,
			`<names xmlns="urn:t" xmlns:u="urn:t"><q>t:foo</q></names>`, typ},
		{"array items", `{"names":{"$xmlns":{"t":"urn:t"},"q":["t:a","t:b"]}}`,
			`<names xmlns="urn:t" xmlns:t="urn:t"><q>t:a</q><q>t:b</q></names>`, none},
		{"two levels down", `{"names":{"$xmlns":{"t":"urn:t"},"inner":{"q":"t:foo"}}}`,
			`<names xmlns="urn:t" xmlns:t="urn:t"><inner><q>t:foo</q></inner></names>`, none},
		{"bound in a sibling only", `{"names":{"inner":{"$xmlns":{"t":"urn:t"},"q":"t:foo"},"item":{"$type":"t:base","a":"x"}}}`,
			`<names xmlns="urn:t" ` + xsiDecl + `><inner xmlns:t="urn:t"><q>t:foo</q></inner><item xsi:type="t:base"><a>x</a></item></names>`,
			attr},
	} {
		t.Run(tc.name, func(t *testing.T) { twin(t, v, tc.json, tc.xml, tc.want) })
	}
}

// TestTypeAttribute pins what the engine reads off a "$type" member: the
// element's first attribute, xsi:type, its value the member's string
// unchanged, located at the "$type" key, before the attribute members.
func TestTypeAttribute(t *testing.T) {
	v := validatorOver(t, typeSchema)
	w := newWalker(jsonURI, strings.NewReader(`{"item":{"$type":" derived ","d":"1","a":"x","b":"y"}}`), v.Schema())
	root, err := w.root()
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	var got []string
	for _, a := range root.Attributes() {
		got = append(got, a.Name().String()+"="+a.Value()+"@"+a.Loc().String())
	}
	want := []string{"{http://www.w3.org/2001/XMLSchema-instance}type= derived @instance.json:1:10", "d=1@instance.json:1:30"}
	if !slices.Equal(got, want) {
		t.Fatalf("Attributes() = %q, want %q", got, want)
	}
	if err := w.end(root); err != nil {
		t.Fatalf("end: %v", err)
	}
}

// TestReservedMemberDeclines pins every "$type" and "$xmlns" the arm does not
// map: each is (nil, err), err opening with "jsonsrc: <uri>:<line>:<col>: " at
// the construct it names.
func TestReservedMemberDeclines(t *testing.T) {
	v := validatorOver(t, typeSchema)
	for _, tc := range []struct {
		name, json string
		prefix     string
		says       string
	}{
		{"$type after a member", `{"item":{"a":"x","$type":"derived"}}`,
			"jsonsrc: instance.json:1:18: ", `reserved member "$type" is out of place; "$xmlns" and then "$type" lead an element's object`},
		{"$type before $xmlns", `{"item":{"$type":"t:derived","$xmlns":{"t":"urn:t"}}}`,
			"jsonsrc: instance.json:1:30: ", `reserved member "$xmlns" is out of place`},
		{"$type repeated", `{"item":{"$type":"base","$type":"base"}}`,
			"jsonsrc: instance.json:1:25: ", `reserved member "$type" is out of place`},
		{"$xmlns repeated", `{"item":{"$xmlns":{},"$xmlns":{}}}`,
			"jsonsrc: instance.json:1:22: ", `reserved member "$xmlns" is out of place`},
		{"$xmlns after a member", `{"names":{"q":"x","$xmlns":{}}}`,
			"jsonsrc: instance.json:1:19: ", `reserved member "$xmlns" is out of place`},
		{"$type not a string", `{"item":{"$type":1}}`,
			"jsonsrc: instance.json:1:18: ", `the "$type" member of {urn:t}item is not a string`},
		{"$xmlns not an object", `{"item":{"$xmlns":"urn:t"}}`,
			"jsonsrc: instance.json:1:19: ", `the "$xmlns" member of {urn:t}item is not an object`},
		{"$xmlns an array", `{"item":{"$xmlns":[]}}`,
			"jsonsrc: instance.json:1:19: ", `the "$xmlns" member of {urn:t}item is not an object`},
		{"$xmlns binding not a string", `{"item":{"$xmlns":{"t":null}}}`,
			"jsonsrc: instance.json:1:24: ", `the "$xmlns" member of {urn:t}item binds "t" to a value that is not a string`},
		{"$xmlns binding a prefix twice", `{"item":{"$xmlns":{"t":"urn:t","t":"urn:t"}}}`,
			"jsonsrc: instance.json:1:32: ", `the "$xmlns" member of {urn:t}item binds "t" twice`},
		{"$xmlns binding the empty prefix twice", `{"item":{"$xmlns":{"":"urn:t","":"urn:t"}}}`,
			"jsonsrc: instance.json:1:31: ", `binds "" twice`},
		{"$xmlns binding a prefix no NCName starts", `{"item":{"$xmlns":{"1x":"urn:t"},"$type":"1x:derived","a":"x","b":"y"}}`,
			"jsonsrc: instance.json:1:20: ", `the "$xmlns" member of {urn:t}item binds "1x", which is not an NCName`},
		{"$xmlns binding a prefix with a space", `{"item":{"$xmlns":{"a b":"urn:t"},"$type":"a b:derived","a":"x","b":"y"}}`,
			"jsonsrc: instance.json:1:20: ", `binds "a b", which is not an NCName`},
		{"$xmlns binding a prefix with a colon", `{"item":{"$xmlns":{"a:b":"urn:t"}}}`,
			"jsonsrc: instance.json:1:20: ", `binds "a:b", which is not an NCName`},
		{"$xmlns binding xml", `{"item":{"$xmlns":{"xml":"http://www.w3.org/XML/1998/namespace"}}}`,
			"jsonsrc: instance.json:1:20: ", `the "$xmlns" member of {urn:t}item binds the reserved prefix "xml"`},
		{"$xmlns binding xmlns", `{"item":{"$xmlns":{"xmlns":"urn:t"}}}`,
			"jsonsrc: instance.json:1:20: ", `binds the reserved prefix "xmlns"`},
		{"$xmlns undeclaring a prefix", `{"item":{"$xmlns":{"t":""}}}`,
			"jsonsrc: instance.json:1:24: ", `binds "t" to the empty string; a prefix cannot be undeclared`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Validate(v, strings.NewReader(tc.json), WithURI(jsonURI))
			if err == nil {
				t.Fatalf("Validate(%s) = %v violations, nil error; want the decline", tc.json, rules(res))
			}
			if res != nil {
				t.Errorf("Validate(%s) returned a result beside its error", tc.json)
			}
			if !strings.HasPrefix(err.Error(), tc.prefix) || !strings.Contains(err.Error(), tc.says) {
				t.Errorf("Validate(%s) = %q, want it to open %q and say %q", tc.json, err, tc.prefix, tc.says)
			}
		})
	}
}
