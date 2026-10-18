package jsonsrc

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/validate"
)

// ncnameSchema declares, in no namespace, s, whose sequence is a skip ##local
// element wildcard of any number; a, whose {attribute wildcard} is a skip
// ##local one; and es and ea, the same under ##any wildcards.
const ncnameSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="s"><xs:complexType><xs:sequence>
    <xs:any namespace="##local" processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence></xs:complexType></xs:element>
  <xs:element name="a"><xs:complexType>
    <xs:anyAttribute namespace="##local" processContents="skip"/>
  </xs:complexType></xs:element>
  <xs:element name="es"><xs:complexType><xs:sequence>
    <xs:any processContents="skip" minOccurs="0" maxOccurs="unbounded"/>
  </xs:sequence></xs:complexType></xs:element>
  <xs:element name="ea"><xs:complexType>
    <xs:anyAttribute processContents="skip"/>
  </xs:complexType></xs:element>
</xs:schema>`

// notNCName is the opening of the decline of an unprefixed key that is no
// NCName, after its location (walker.notNCName).
func notNCName(key string) string {
	return `member "` + key + `" is no NCName; a key with no colon is the local name of the element or attribute it names, an NCName (XML 1.0 §2.3, Namespaces in XML §7)`
}

// TestKeyNotNCNameDeclines pins an unprefixed key that is no NCName declined
// (walker.keyName): it has no XML twin, and the mapping would otherwise name
// an element or attribute no XML document can spell. Each decline is the
// whole message at the key; one whose key is "@" or "-" before the local name
// of an attribute use of the object it is classified against names the key
// that use is named by (walker.notNCName), and every other names none.
func TestKeyNotNCNameDeclines(t *testing.T) {
	readme := validatorOver(t, readmeSchema)
	wild := validatorOver(t, ncnameSchema)
	run := validatorOver(t, runSchema)
	const order = `{"o:order":{"$xmlns":{"o":"urn:o"},`
	named := func(key, use, owner, as string) string {
		return notNCName(key) + `, and an attribute member's key takes no "@" or "-": the attribute use ` + use + ` of the type of ` + owner + ` is named by the key "` + as + `"`
	}
	for _, tc := range []struct {
		name string
		v    *validate.Validator
		json string
		want string
	}{
		{"an attribute key with \"@\"", readme, order + `"@kind":"x","item":["a"]}}`,
			`jsonsrc: instance.json:1:36: ` + named("@kind", "kind", "{urn:o}order", "kind")},
		{"an attribute key with \"-\"", readme, order + `"-kind":"x","item":["a"]}}`,
			`jsonsrc: instance.json:1:36: ` + named("-kind", "kind", "{urn:o}order", "kind")},
		{"an \"@\" key no use of the owner carries", readme, order + `"@other":"x","item":["a"]}}`,
			`jsonsrc: instance.json:1:36: ` + notNCName("@other")},
		{"an \"@\" key after a child member", readme, order + `"item":["a"],"@kind":"x"}}`,
			`jsonsrc: instance.json:1:49: ` + named("@kind", "kind", "{urn:o}order", "kind")},
		{"an \"@\" key with an array value", readme, order + `"@kind":["x"],"item":["a"]}}`,
			`jsonsrc: instance.json:1:36: ` + named("@kind", "kind", "{urn:o}order", "kind")},
		{"a key opening with a digit", readme, order + `"1bad":"x","item":["a"]}}`,
			`jsonsrc: instance.json:1:36: ` + notNCName("1bad")},
		{"a key with a later character no NameChar", readme, order + `"item":["a"],"a b":"x"}}`,
			`jsonsrc: instance.json:1:49: ` + notNCName("a b")},
		{"the empty key", readme, order + `"":"x","item":["a"]}}`,
			`jsonsrc: instance.json:1:36: ` + notNCName("")},
		{"an \"@\" root key", readme, `{"@root":{}}`,
			`jsonsrc: instance.json:1:2: ` + notNCName("@root")},
		{"an \"@\" root key a use's name follows", readme, `{"@kind":{}}`,
			`jsonsrc: instance.json:1:2: ` + notNCName("@kind")},
		{"a root key opening with a digit", readme, `{"1bad":"x"}`,
			`jsonsrc: instance.json:1:2: ` + notNCName("1bad")},
		{"the empty root key", readme, `{"":null}`,
			`jsonsrc: instance.json:1:2: ` + notNCName("")},
		{"an \"@\" key under a skip ##local element wildcard", wild, `{"s":{"@kind":"x","1bad":{}}}`,
			`jsonsrc: instance.json:1:7: ` + notNCName("@kind")},
		{"a key opening with a digit under a skip ##local element wildcard", wild, `{"s":{"1bad":{}}}`,
			`jsonsrc: instance.json:1:7: ` + notNCName("1bad")},
		{"a key opening with a digit after a child under a skip ##local element wildcard", wild, `{"s":{"ok":{},"1bad":{}}}`,
			`jsonsrc: instance.json:1:15: ` + notNCName("1bad")},
		{"an \"@\" key under a skip ##local {attribute wildcard}", wild, `{"a":{"@kind":"x"}}`,
			`jsonsrc: instance.json:1:7: ` + notNCName("@kind")},
		{"a \"-\" key under a skip ##local {attribute wildcard}", wild, `{"a":{"-kind":"x"}}`,
			`jsonsrc: instance.json:1:7: ` + notNCName("-kind")},
		{"an \"@\" key under a ##any element wildcard", wild, `{"es":{"@kind":{}}}`,
			`jsonsrc: instance.json:1:8: ` + notNCName("@kind")},
		{"an \"@\" key under a ##any {attribute wildcard}", wild, `{"ea":{"@kind":"x"}}`,
			`jsonsrc: instance.json:1:8: ` + notNCName("@kind")},
		{"an \"@\" key opening the attribute run of an element with a {type table}", run, `{"tu":{"@a":"u"}}`,
			`jsonsrc: instance.json:1:8: ` + named("@a", "a", "{urn:s}tu", "a")},
		{"a \"-\" key the attribute run reads ahead to", run, `{"tu":{"a":"z","m":"x","-a":"1"}}`,
			`jsonsrc: instance.json:1:24: ` + named("-a", "a", "{urn:s}tu", "a")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, err := Validate(tc.v, strings.NewReader(tc.json), WithURI(jsonURI))
			var d *DeclineError
			if !errors.As(err, &d) {
				t.Fatalf("Validate(%s) = %v, %v; want a *DeclineError", tc.json, res, err)
			}
			if res != nil {
				t.Errorf("Validate(%s) returned a result beside its decline", tc.json)
			}
			if err.Error() != tc.want {
				t.Errorf("Validate(%s) =\n%q\nwant\n%q", tc.json, err, tc.want)
			}
		})
	}
}

// TestKeyNCNameMaps pins the keys walker.keyName's no-NCName decline leaves
// mapped: an NCName whose first character is no letter, "_kind", and ones
// holding "." and "-", as elements under a skip ##local element wildcard and
// as attributes under a skip ##local {attribute wildcard}, each beside its XML
// twin; and the colon-form declines, whose messages it leaves unchanged.
func TestKeyNCNameMaps(t *testing.T) {
	wild := validatorOver(t, ncnameSchema)
	twin(t, wild, `{"s":{"_kind":{},"a.b":"x","a-b":null}}`,
		`<s><_kind/><a.b>x</a.b><a-b xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:nil="true"/></s>`, none)
	twin(t, wild, `{"a":{"_kind":"x","a.b":"y","a-b":"z"}}`, `<a _kind="x" a.b="y" a-b="z"/>`, none)
	readme := validatorOver(t, readmeSchema)
	twin(t, readme, `{"o:order":{"$xmlns":{"o":"urn:o"},"kind":"x","item":["a"]}}`,
		`<o:order xmlns:o="urn:o" kind="x"><o:item>a</o:item></o:order>`, none)
	for _, key := range []string{"o:1bad", "o:@kind", "@o:kind"} {
		doc := `{"o:order":{"$xmlns":{"o":"urn:o"},"` + key + `":"x","item":["a"]}}`
		want := `jsonsrc: instance.json:1:36: member "` + key + `" is no QName; a key with a colon is a prefix and a local name, each an NCName`
		_, err := Validate(readme, strings.NewReader(doc), WithURI(jsonURI))
		if err == nil || err.Error() != want {
			t.Errorf("Validate(%s) = %v, want %q", doc, err, want)
		}
	}
}
