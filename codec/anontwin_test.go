package codec

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// atTwinSchema returns the schema of r, carrying kind, inheritable, and
// holding e, an xs:string whose {type table} selects one anonymous
// restriction of xs:string where the inherited kind is "x", another where it
// is "y", and xs:string otherwise. The two are built by program, as
// xsd.NewSimpleType builds them: no facet, the zero Loc, equal in every
// property, so no comparison by value tells them apart. A pair the parser
// builds differs by the Loc of each <simpleType>. atTwin, atTwinE, atTwinAlt1,
// atTwinAlt2 and atTwinString are codegen's R, E, EAlternative1, EAlternative2
// and EString.
func atTwinSchema(t *testing.T) *xsd.Schema {
	t.Helper()
	parsed := schemaOf(t, `<xs:element name="e" type="xs:string"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:e"/></xs:sequence><xs:attribute name="kind" type="xs:string" inheritable="true"/></xs:complexType></xs:element>`)
	xsString := xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"}
	alternative := func(test string) xsd.TypeAlternative {
		st, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{}, xsd.RestrictionDerivation{}, xsd.SimpleTypeRef{Name: xsString}, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		x := xsd.NewXPathExpression(test, nil, nil, nil)
		ta, err := xsd.NewTypeAlternative(xsderr.Loc{}, &x, xsd.InlineTypeDefinition{Definition: st})
		if err != nil {
			t.Fatal(err)
		}
		return ta
	}
	dflt, err := xsd.NewTypeAlternative(xsderr.Loc{}, nil, xsd.TypeDefinitionRef{Name: xsString})
	if err != nil {
		t.Fatal(err)
	}
	table, err := xsd.NewTypeTable(xsderr.Loc{}, []xsd.TypeAlternative{alternative("@kind = 'x'"), alternative("@kind = 'y'")}, dflt)
	if err != nil {
		t.Fatal(err)
	}
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Space: "urn:s", Local: "e"}, xsd.TypeDefinitionRef{Name: xsString}, &table,
		xsd.NewGlobalScope(), nil, false, nil, nil, nil, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	b := xsd.NewSchemaBuilder()
	for _, td := range parsed.Types() {
		b.AddType(td)
	}
	for _, d := range parsed.Elements() {
		if d.Name() == e.Name() {
			d = e
		}
		b.AddElement(d)
	}
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("Finalize: %v", err)
	}
	return schema
}

type (
	atTwin struct {
		E    atTwinE
		Kind *string
	}
	atTwinE      interface{ isAtTwinE() }
	atTwinAlt1   struct{ Alternative1 string }
	atTwinAlt2   struct{ Alternative2 string }
	atTwinString struct{ String string }
)

func (atTwinAlt1) isAtTwinE()   {}
func (atTwinAlt2) isAtTwinE()   {}
func (atTwinString) isAtTwinE() {}

// TestDecodeAnonymousTwins pins that atTwinSchema's two anonymous types,
// equal in every property, each fill the branch of its own slot, selected by
// its own {test}, and that CheckDecode returns nil for codegen's T.
func TestDecodeAnonymousTwins(t *testing.T) {
	schema := atTwinSchema(t)
	branches := WithBranches[atTwinE](atTwinAlt1{}, atTwinAlt2{}, atTwinString{})
	str := func(s string) *string { return &s }
	for _, row := range []struct {
		kind string
		want atTwin
	}{
		{"x", atTwin{E: atTwinAlt1{Alternative1: "v"}, Kind: str("x")}},
		{"y", atTwin{E: atTwinAlt2{Alternative2: "v"}, Kind: str("y")}},
		{"z", atTwin{E: atTwinString{String: "v"}, Kind: str("z")}},
	} {
		data := instanceOf(` kind="`+row.kind+`"`, "<s:e>v</s:e>")
		mustAccept(t, schema, data)
		got, err := Decode[atTwin](schema, strict.New(), data, branches)
		if err != nil {
			t.Fatalf("Decode(kind %s) = %v", row.kind, err)
		}
		if !reflect.DeepEqual(*got, row.want) {
			t.Errorf("Decode(kind %s) = %#v, want %#v", row.kind, *got, row.want)
		}
	}
	if err := CheckDecode[atTwin](schema, rootName, branches); err != nil {
		t.Errorf("CheckDecode = %v, want nil", err)
	}
}

// TestTableMatchedUnmatched pins tableRoot.matched where the selected type is
// none of the table's governing types, which no instance Decode validated
// reaches: xs:error at its slot, which goshape.GoverningTypes drops, and an
// anonymous type at a slot no governing type holds. Each is a plain error
// naming the element's instance Loc and pointing at key-cta-ta-select, never
// a branch.
func TestTableMatchedUnmatched(t *testing.T) {
	schema := schemaOf(t, ttTypes+`<xs:element name="r" type="s:base"><xs:alternative test="@kind = 'e'" type="xs:error"/><xs:alternative test="@kind" type="s:num"/></xs:element>`)
	decl, ok := schema.Element(rootName)
	if !ok {
		t.Fatal("the schema declares no s:r")
	}
	types, _, err := governingTypes(schema, decl, true)
	if err != nil {
		t.Fatal(err)
	}
	table, _ := decl.TypeTable()
	tr := &tableRoot{table: table, types: types}
	start, err := documentElement(xmltree.NewReader("", bytes.NewReader(instanceOf("", ""))))
	if err != nil {
		t.Fatal(err)
	}
	xsError, ok := schema.ResolvedType(xsd.TypeDefinitionRef{Name: xsd.QName{Space: xsd.XMLSchemaNS, Local: "error"}})
	if !ok {
		t.Fatal("xs:error does not resolve")
	}
	two := schemaOf(t, atTwoSchema)
	twoDecl, _ := two.Element(rootName)
	twoTable, _ := twoDecl.TypeTable()
	anonymous, ok := two.ResolvedType(twoTable.Alternatives()[0].TypeDefinition())
	if !ok || anonymous.Name() != (xsd.QName{}) {
		t.Fatal("atTwoSchema's first alternative holds no anonymous type")
	}
	for _, row := range []struct {
		slot goshape.Slot
		td   xsd.TypeDefinition
		want string
	}{
		{goshape.AlternativeSlot(1), xsError, "codec: ?:1:1: element {urn:s}r: the type its {type table} selects (key-cta-ta-select) at its {type table} alternative 1 is none of its governing types"},
		{goshape.AlternativeSlot(1), anonymous, "codec: ?:1:1: element {urn:s}r: the type its {type table} selects (key-cta-ta-select) at its {type table} alternative 1 is none of its governing types"},
	} {
		i, err := tr.matched(start, row.slot, row.td)
		if err == nil || err.Error() != row.want {
			t.Errorf("matched(%s, %s) = %d, %v; want the error %q", row.slot.Name(), row.td.Name(), i, err, row.want)
		}
	}
}
