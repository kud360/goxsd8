package codec

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsderr"
)

// nillable, nilled and content are codegen's Nillable and its arms Nilled and
// Content, as codegen/doc.go "# Element declarations" states them, which
// TestNillableTypesAreCodegens pins.
type (
	nillable[T, A any] interface{ isNillable(T, A) }
	nilled[T, A any]   struct{ Attributes A }
	content[T, A any]  struct{ Content T }
)

func (nilled[T, A]) isNillable(T, A)  {}
func (content[T, A]) isNillable(T, A) {}

// nilArms is the WithBranches of Nillable[T, A], one per instantiation T holds.
func nilArms[T, A any]() Option {
	return WithBranches[nillable[T, A]](nilled[T, A]{}, content[T, A]{})
}

// The nillable schemas, each declaring s:r. nilFieldsSchema's r holds a
// nillable field of each shape: id, a required xs:int; note, an optional
// xs:string; tag, a repeated one; price, of simple content with a defaulted
// and a required attribute, whose attributes struct A holds both; h, an
// <element ref> to a nillable top-level xs:string; next, of link, which
// reaches itself through an optional nillable next; size, with a default; o,
// optional, of opt, whose choice is emptiable; and c, nillable with a fixed
// value, which codegen emits unwrapped. nilFlatSchema's a is a nillable
// basic particle of a flattened choice, nilOccurSchema's a nillable particle
// of a repeated sequence held in an occurrence field, nilMixedSchema's em an
// optional nillable child of mixed content, and nilAltSchema's a a nillable
// choice alternative.
const (
	nilTypes = `<xs:complexType name="price"><xs:simpleContent><xs:extension base="xs:string">` +
		`<xs:attribute name="cur" type="xs:string" default="EUR"/><xs:attribute name="k" type="xs:string" use="required"/></xs:extension></xs:simpleContent></xs:complexType>` +
		`<xs:complexType name="link"><xs:sequence><xs:element name="next" type="s:link" nillable="true" minOccurs="0"/></xs:sequence></xs:complexType>` +
		`<xs:complexType name="opt"><xs:choice><xs:element name="x" type="xs:string" minOccurs="0"/><xs:element name="y" type="xs:int"/></xs:choice></xs:complexType>` +
		`<xs:element name="h" type="xs:string" nillable="true"/>`
	nilFieldsSchema = nilTypes + `<xs:element name="r"><xs:complexType><xs:sequence>` +
		`<xs:element name="id" type="xs:int" nillable="true"/>` +
		`<xs:element name="note" type="xs:string" nillable="true" minOccurs="0"/>` +
		`<xs:element name="tag" type="xs:string" nillable="true" maxOccurs="unbounded"/>` +
		`<xs:element name="price" type="s:price" nillable="true"/>` +
		`<xs:element ref="s:h"/>` +
		`<xs:element name="next" type="s:link" nillable="true"/>` +
		`<xs:element name="size" type="xs:int" nillable="true" default="1"/>` +
		`<xs:element name="o" type="s:opt" nillable="true" minOccurs="0"/>` +
		`<xs:element name="c" type="xs:string" nillable="true" fixed="A"/>` +
		`</xs:sequence></xs:complexType></xs:element>`
	nilFlatSchema  = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string" nillable="true"/><xs:element name="b" type="xs:int"/></xs:choice></xs:complexType></xs:element>`
	nilOccurSchema = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="2"><xs:element name="a" type="xs:string" nillable="true"/><xs:element name="b" type="xs:int"/></xs:sequence></xs:complexType></xs:element>`
	nilMixedSchema = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="em" type="xs:string" nillable="true" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	nilAltSchema   = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string" nillable="true" minOccurs="0"/><xs:element name="b" type="xs:int"/></xs:choice></xs:complexType></xs:element>`
)

// The types codegen emits for those schemas' s:r, each named here by its
// schema, which TestNillableTypesAreCodegens pins.
type (
	nilFields struct {
		Id    nillable[int32, struct{}]
		Note  nillable[string, struct{}]
		Tag   []nillable[string, struct{}]
		Price nillable[nilPrice, nilPriceAttributes]
		H     nillable[nilH, struct{}]
		Next  nillable[nilLink, struct{}]
		Size  nillable[int32, struct{}]
		O     nillable[nilOpt, struct{}]
		C     string
	}
	nilPrice struct {
		Value string
		Cur   string
		K     string
	}
	nilPriceAttributes struct {
		Cur string
		K   string
	}
	nilH    string
	nilLink struct{ Next nillable[nilLink, struct{}] }
	nilOpt  struct{ Choice nilOptChoice }

	nilOptChoice interface{ isNilOptChoice() }
	nilOptX      struct{ X string }
	nilOptY      struct{ Y int32 }

	nilMixed struct {
		Text string
		Em   *mixedChild[nillable[string, struct{}]]
	}
	nilFlat       struct{ Choice []nilFlatChoice }
	nilFlatChoice interface{ isNilFlatChoice() }
	nilFlatA      struct{ A nillable[string, struct{}] }
	nilFlatB      struct{ B int32 }
	nilOccur      struct{ Sequence []nilOccurItem }
	nilOccurItem  struct {
		A nillable[string, struct{}]
		B int32
	}
	nilPick  interface{ isNilPick() }
	nilPickA struct{ A nillable[string, struct{}] }
	nilPickB struct{ B int32 }
)

func (nilOptX) isNilOptChoice()   {}
func (nilOptY) isNilOptChoice()   {}
func (nilFlatA) isNilFlatChoice() {}
func (nilFlatB) isNilFlatChoice() {}
func (nilPickA) isNilPick()       {}
func (nilPickB) isNilPick()       {}

// nilFieldsOptions is the option list of nilFields: one set per Nillable
// instantiation and opt's choice's.
var nilFieldsOptions = []Option{
	nilArms[int32, struct{}](), nilArms[string, struct{}](), nilArms[nilPrice, nilPriceAttributes](),
	nilArms[nilH, struct{}](), nilArms[nilLink, struct{}](), nilArms[nilOpt, struct{}](),
	WithBranches[nilOptChoice](nilOptX{}, nilOptY{}),
}

// TestNillableTypesAreCodegens pins the types above to those codegen emits
// for each schema, in shape, so that the rows decoding into them decode into
// codegen's types.
func TestNillableTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls string
		want  []string
	}{
		{nilFieldsSchema, []string{
			"type R struct {\n\tId    Nillable[int32, struct{}]\n\tNote  Nillable[string, struct{}]\n\tTag   []Nillable[string, struct{}]\n\tPrice Nillable[Price, PriceAttributes]\n" +
				"\tH     Nillable[H, struct{}]\n\tNext  Nillable[Link, struct{}]\n\tSize  Nillable[int32, struct{}]\n\tO     Nillable[Opt, struct{}]\n\tC     string\n}\n",
			"type Price struct {\n\tValue string\n\tCur   string\n\tK     string\n}\n",
			"type PriceAttributes struct {\n\tCur string\n\tK   string\n}\n",
			"type H string\n",
			"type Link struct {\n\tNext Nillable[Link, struct{}]\n}\n",
			"type Opt struct {\n\tChoice OptChoice\n}\n",
			"type Nillable[T, A any] interface {\n\tisNillable(T, A)\n}\n",
			"type Nilled[T, A any] struct {\n\tAttributes A\n}\n",
			"type Content[T, A any] struct {\n\tContent T\n}\n",
		}},
		{nilFlatSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", "type RChoiceA struct {\n\tA Nillable[string, struct{}]\n}\n"}},
		{nilOccurSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA Nillable[string, struct{}]\n\tB int32\n}\n"}},
		{nilMixedSchema, []string{"type R struct {\n\tText string\n\tEm   *MixedChild[Nillable[string, struct{}]]\n}\n"}},
		{nilAltSchema, []string{"type R struct {\n\tChoice RChoice\n}\n", "type RChoiceA struct {\n\tA Nillable[string, struct{}]\n}\n"}},
		{nilRootSimpleSchema, []string{"type R int32\n"}},
		{nilRootComplexSchema, []string{"type R struct {\n\tA string\n\tK string\n}\n", "type RAttributes struct {\n\tK string\n}\n"}},
		{nilRootPickSchema, []string{"type R interface {\n\tisR()\n}\n"}},
	}
	for _, row := range rows {
		gen := generated(t, schemaOf(t, row.decls))
		for _, decl := range row.want {
			if !bytes.Contains(gen, []byte(decl)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
			}
		}
	}
}

// TestDecodeNillable pins the nillable lift for a field (cvc-elt clause 3):
// an element not nilled, with no xsi:nil or xsi:nil false, its lexical form 0
// included, decodes into the Content arm holding what the unwrapped field
// would; a nilled one, xsi:nil true, 1 or " true " read as an xs:boolean,
// into the Nilled arm holding its attributes, a default included
// (cvc-complex-type clauses 2 to 4), and none of the element's own value
// constraint (cvc-elt clause 5.1); an absent optional one is the nil
// interface; and a repeated one a slice of the interface. A declaration with
// a fixed value decodes unwrapped (cvc-elt clause 3.2.3.2).
func TestDecodeNillable(t *testing.T) {
	rows := []struct {
		name  string
		decls string
		body  string
		into  target
		want  any
	}{
		{"not nilled", nilFieldsSchema,
			`<s:id>1</s:id><s:note>n</s:note><s:tag>a</s:tag><s:tag xsi:nil="true"/><s:price k="q">p</s:price><s:h>h</s:h><s:next><s:next xsi:nil="1"/></s:next><s:size/><s:o><s:y>2</s:y></s:o><s:c>A</s:c>`,
			into[nilFields](nilFieldsOptions...), nilFields{
				Id:    content[int32, struct{}]{Content: 1},
				Note:  content[string, struct{}]{Content: "n"},
				Tag:   []nillable[string, struct{}]{content[string, struct{}]{Content: "a"}, nilled[string, struct{}]{}},
				Price: content[nilPrice, nilPriceAttributes]{Content: nilPrice{Value: "p", Cur: "EUR", K: "q"}},
				H:     content[nilH, struct{}]{Content: "h"},
				Next:  content[nilLink, struct{}]{Content: nilLink{Next: nilled[nilLink, struct{}]{}}},
				Size:  content[int32, struct{}]{Content: 1},
				O:     content[nilOpt, struct{}]{Content: nilOpt{Choice: nilOptY{Y: 2}}},
				C:     "A",
			}},
		{"nilled", nilFieldsSchema,
			`<s:id xsi:nil="true"/><s:tag xsi:nil=" true "/><s:price xsi:nil="1" k="q"/><s:h xsi:nil="true"/><s:next xsi:nil="true"/><s:size xsi:nil="true"/><s:o xsi:nil="true"/><s:c/>`,
			into[nilFields](nilFieldsOptions...), nilFields{
				Id:    nilled[int32, struct{}]{},
				Tag:   []nillable[string, struct{}]{nilled[string, struct{}]{}},
				Price: nilled[nilPrice, nilPriceAttributes]{Attributes: nilPriceAttributes{Cur: "EUR", K: "q"}},
				H:     nilled[nilH, struct{}]{},
				Next:  nilled[nilLink, struct{}]{},
				Size:  nilled[int32, struct{}]{},
				O:     nilled[nilOpt, struct{}]{},
				C:     "A",
			}},
		{"xsi:nil false and 0", nilFieldsSchema,
			`<s:id xsi:nil="0">2</s:id><s:tag xsi:nil="false">t</s:tag><s:price k="q" cur="USD" xsi:nil="false">p</s:price><s:h>h</s:h><s:next/><s:size>3</s:size><s:o/><s:c>A</s:c>`,
			into[nilFields](nilFieldsOptions...), nilFields{
				Id:    content[int32, struct{}]{Content: 2},
				Tag:   []nillable[string, struct{}]{content[string, struct{}]{Content: "t"}},
				Price: content[nilPrice, nilPriceAttributes]{Content: nilPrice{Value: "p", Cur: "USD", K: "q"}},
				H:     content[nilH, struct{}]{Content: "h"},
				Next:  content[nilLink, struct{}]{Content: nilLink{}},
				Size:  content[int32, struct{}]{Content: 3},
				O:     content[nilOpt, struct{}]{Content: nilOpt{}},
				C:     "A",
			}},
		{"a nilled element keeps its attributes", nilFieldsSchema,
			`<s:id>1</s:id><s:tag>a</s:tag><s:price xsi:nil="true" cur="GBP" k="q"/><s:h>h</s:h><s:next/><s:size>1</s:size><s:c>A</s:c>`,
			into[nilFields](nilFieldsOptions...), nilFields{
				Id:    content[int32, struct{}]{Content: 1},
				Tag:   []nillable[string, struct{}]{content[string, struct{}]{Content: "a"}},
				Price: nilled[nilPrice, nilPriceAttributes]{Attributes: nilPriceAttributes{Cur: "GBP", K: "q"}},
				H:     content[nilH, struct{}]{Content: "h"},
				Next:  content[nilLink, struct{}]{Content: nilLink{}},
				Size:  content[int32, struct{}]{Content: 1},
				C:     "A",
			}},
		{"mixed content, nilled", nilMixedSchema, `a<s:em xsi:nil="true"/>b`, into[nilMixed](nilArms[string, struct{}]()),
			nilMixed{Text: "a", Em: &mixedChild[nillable[string, struct{}]]{Element: nilled[string, struct{}]{}, Tail: "b"}}},
		{"mixed content, not nilled", nilMixedSchema, `a<s:em>e</s:em>b`, into[nilMixed](nilArms[string, struct{}]()),
			nilMixed{Text: "a", Em: &mixedChild[nillable[string, struct{}]]{Element: content[string, struct{}]{Content: "e"}, Tail: "b"}}},
		{"mixed content, absent", nilMixedSchema, `a`, into[nilMixed](nilArms[string, struct{}]()), nilMixed{Text: "a"}},
		{"a flattened slice's branch, nilled and not", nilFlatSchema, `<s:a xsi:nil="true"/><s:b>1</s:b><s:a>x</s:a>`,
			into[nilFlat](nilArms[string, struct{}](), WithBranches[nilFlatChoice](nilFlatA{}, nilFlatB{})),
			nilFlat{Choice: []nilFlatChoice{nilFlatA{A: nilled[string, struct{}]{}}, nilFlatB{B: 1}, nilFlatA{A: content[string, struct{}]{Content: "x"}}}}},
		{"an occurrence struct's field, nilled and not", nilOccurSchema, `<s:a xsi:nil="true"/><s:b>1</s:b><s:a>x</s:a><s:b>2</s:b>`,
			into[nilOccur](nilArms[string, struct{}]()),
			nilOccur{Sequence: []nilOccurItem{{A: nilled[string, struct{}]{}, B: 1}, {A: content[string, struct{}]{Content: "x"}, B: 2}}}},
		{"a choice's alternative, nilled", nilAltSchema, `<s:a xsi:nil="true"/>`,
			into[struct{ Choice nilPick }](nilArms[string, struct{}](), WithBranches[nilPick](nilPickA{}, nilPickB{})),
			struct{ Choice nilPick }{Choice: nilPickA{A: nilled[string, struct{}]{}}}},
		{"a choice's alternative, not nilled", nilAltSchema, `<s:a>v</s:a>`,
			into[struct{ Choice nilPick }](nilArms[string, struct{}](), WithBranches[nilPick](nilPickA{}, nilPickB{})),
			struct{ Choice nilPick }{Choice: nilPickA{A: content[string, struct{}]{Content: "v"}}}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf("", row.body)
			mustAccept(t, schema, data)
			got, err := row.into.decode(schema, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, row.want) {
				t.Errorf("Decode = %#v, want %#v", got, row.want)
			}
			if err := row.into.check(schema); err != nil {
				t.Errorf("CheckDecode = %v, want nil for a row Decode decodes", err)
			}
		})
	}
}

// TestDecodeNillableThreeValues pins codegen/doc.go's three values of an
// optional nillable field whose type's choice is emptiable: absent, the nil
// interface; nilled, the Nilled arm; and empty, the Content arm holding the
// choice's empty match, a nil interface inside it. No two are equal.
func TestDecodeNillableThreeValues(t *testing.T) {
	schema := schemaOf(t, nilFieldsSchema)
	const lead = `<s:id>1</s:id><s:tag>a</s:tag><s:price k="q">p</s:price><s:h>h</s:h><s:next/><s:size>1</s:size>`
	bodies := []string{lead + `<s:c>A</s:c>`, lead + `<s:o xsi:nil="true"/><s:c>A</s:c>`, lead + `<s:o/><s:c>A</s:c>`}
	wants := []nillable[nilOpt, struct{}]{nil, nilled[nilOpt, struct{}]{}, content[nilOpt, struct{}]{}}
	for i, body := range bodies {
		data := instanceOf("", body)
		mustAccept(t, schema, data)
		got, err := Decode[nilFields](schema, strict.New(), data, nilFieldsOptions...)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got.O, wants[i]) {
			t.Errorf("%s: O = %#v, want %#v", body, got.O, wants[i])
		}
		for j := range i {
			if reflect.DeepEqual(wants[i], wants[j]) {
				t.Errorf("values %d and %d are equal", i, j)
			}
		}
	}
}

// The nillable document elements: nilRootSimpleSchema's s:r of xs:int,
// nilRootComplexSchema's of a sequence and a defaulted attribute beside an
// xsi:nil use, which gives no field, and nilRootPickSchema's of pick, whose
// choice codegen emits as the type itself.
const (
	nilRootSimpleSchema  = `<xs:element name="r" type="xs:int" nillable="true"/>`
	nilRootComplexSchema = `<xs:import namespace="http://www.w3.org/2001/XMLSchema-instance"/>` +
		`<xs:element name="r" nillable="true"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>` +
		`<xs:attribute name="k" type="xs:string" default="d"/><xs:attribute ref="xsi:nil"/></xs:complexType></xs:element>`
	nilRootPickSchema = `<xs:complexType name="pick"><xs:choice><xs:element name="x" type="xs:string"/><xs:element name="y" type="xs:int"/></xs:choice></xs:complexType>` +
		`<xs:element name="r" type="s:pick" nillable="true"/>`
)

// The types codegen emits for those schemas' s:r: R, and Nillable over R a
// nilled root's value.
type (
	nilRootSimple  int32
	nilRootComplex struct {
		A string
		K string
	}
	nilRootComplexAttributes struct{ K string }
	nilRootPick              interface{ isNilRootPick() }
	nilRootPickX             struct{ X string }
	nilRootPickY             struct{ Y int32 }
)

func (nilRootPickX) isNilRootPick() {}
func (nilRootPickY) isNilRootPick() {}

// TestDecodeNillableRoot pins Decode's T for a nillable document element:
// Nillable over R, the Content arm for one not nilled and the Nilled arm,
// holding its attributes, for a nilled one.
func TestDecodeNillableRoot(t *testing.T) {
	pickOpts := []Option{nilArms[nilRootPick, struct{}](), WithBranches[nilRootPick](nilRootPickX{}, nilRootPickY{})}
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"simple, not nilled", nilRootSimpleSchema, "", "5", into[nillable[nilRootSimple, struct{}]](nilArms[nilRootSimple, struct{}]()),
			nillable[nilRootSimple, struct{}](content[nilRootSimple, struct{}]{Content: 5})},
		{"simple, nilled", nilRootSimpleSchema, ` xsi:nil="true"`, "", into[nillable[nilRootSimple, struct{}]](nilArms[nilRootSimple, struct{}]()),
			nillable[nilRootSimple, struct{}](nilled[nilRootSimple, struct{}]{})},
		{"complex, not nilled", nilRootComplexSchema, ` xsi:nil="false"`, "<s:a>x</s:a>",
			into[nillable[nilRootComplex, nilRootComplexAttributes]](nilArms[nilRootComplex, nilRootComplexAttributes]()),
			nillable[nilRootComplex, nilRootComplexAttributes](content[nilRootComplex, nilRootComplexAttributes]{Content: nilRootComplex{A: "x", K: "d"}})},
		{"complex, nilled, its default attribute supplied", nilRootComplexSchema, ` xsi:nil="true"`, "",
			into[nillable[nilRootComplex, nilRootComplexAttributes]](nilArms[nilRootComplex, nilRootComplexAttributes]()),
			nillable[nilRootComplex, nilRootComplexAttributes](nilled[nilRootComplex, nilRootComplexAttributes]{Attributes: nilRootComplexAttributes{K: "d"}})},
		{"complex, nilled, its attribute present", nilRootComplexSchema, ` k="v" xsi:nil="1"`, "",
			into[nillable[nilRootComplex, nilRootComplexAttributes]](nilArms[nilRootComplex, nilRootComplexAttributes]()),
			nillable[nilRootComplex, nilRootComplexAttributes](nilled[nilRootComplex, nilRootComplexAttributes]{Attributes: nilRootComplexAttributes{K: "v"}})},
		{"a choice's interface, not nilled", nilRootPickSchema, "", "<s:y>3</s:y>", into[nillable[nilRootPick, struct{}]](pickOpts...),
			nillable[nilRootPick, struct{}](content[nilRootPick, struct{}]{Content: nilRootPickY{Y: 3}})},
		{"a choice's interface, nilled", nilRootPickSchema, ` xsi:nil="true"`, "", into[nillable[nilRootPick, struct{}]](pickOpts...),
			nillable[nilRootPick, struct{}](nilled[nilRootPick, struct{}]{})},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf(row.attrs, row.body)
			mustAccept(t, schema, data)
			got, err := row.into.decode(schema, data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, row.want) {
				t.Errorf("Decode = %#v, want %#v", got, row.want)
			}
			if err := row.into.check(schema); err != nil {
				t.Errorf("CheckDecode = %v, want nil for a row Decode decodes", err)
			}
		})
	}
}

// TestDecodeNillableMismatch pins item 6 at Nillable: the bare R of a
// nillable document element, nilled or not, is no type codegen emits for it,
// so no nilled root decodes into a value of R indistinguishable from an empty
// one; nor is a Nillable whose arms no WithBranches supplies, or whose
// Attributes is no struct of the attribute fields. Each is ErrTypeMismatch
// from Decode and CheckDecode alike, and nothing is filled.
func TestDecodeNillableMismatch(t *testing.T) {
	type wrongAttributes struct{ J string }
	rows := []struct {
		name   string
		decls  string
		attrs  string
		body   string
		into   target
		prefix string
	}{
		{"bare R of a simple root, not nilled", nilRootSimpleSchema, "", "5", into[nilRootSimple](),
			"codec: T is not the type codegen emits for the document element: T is codec.nilRootSimple, which is not an interface of one unexported method, codegen's Nillable"},
		{"bare R of a simple root, nilled", nilRootSimpleSchema, ` xsi:nil="true"`, "", into[nilRootSimple](),
			"codec: T is not the type codegen emits for the document element: T is codec.nilRootSimple, which is not an interface of one unexported method, codegen's Nillable"},
		{"bare R of a complex root", nilRootComplexSchema, "", "<s:a>x</s:a>", into[nilRootComplex](),
			"codec: T is not the type codegen emits for the document element: T is codec.nilRootComplex, which is not an interface of one unexported method, codegen's Nillable"},
		{"no arms supplied", nilRootSimpleSchema, "", "5", into[nillable[nilRootSimple, struct{}]](),
			"codec: T is not the type codegen emits for the document element: no WithBranches supplies the arms of "},
		{"Attributes of no attribute field", nilRootComplexSchema, "", "<s:a>x</s:a>", into[nillable[nilRootComplex, wrongAttributes]](nilArms[nilRootComplex, wrongAttributes]()),
			"codec: T is not the type codegen emits for the document element: field 0 of codec.wrongAttributes is J string where codegen emits K string"},
		{"a field's Content of the wrong Go type", nilAltSchema, "", `<s:a>v</s:a>`,
			into[struct{ Choice nilPick }](nilArms[int32, struct{}](), WithBranches[nilPick](nilPickA{}, nilPickB{})),
			"codec: T is not the type codegen emits for the document element: "},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf(row.attrs, row.body)
			mustAccept(t, schema, data)
			got, err := row.into.decode(schema, data)
			if got != nil || !errors.Is(err, ErrTypeMismatch) || len(err.Error()) < len(row.prefix) || err.Error()[:len(row.prefix)] != row.prefix {
				t.Fatalf("Decode = %v, %v, want no value and an ErrTypeMismatch opening %q", got, err, row.prefix)
			}
			assertProbeMismatch(t, row.into.check(schema), err)
		})
	}
}

// TestDecodeNillableInvalid pins that each invalid nil form is validate's
// charge, cvc-elt, never a Nilled value: an xsi:nil on a declaration that is
// not nillable, false included (clause 3.1); a nilled element with a
// character or element child (clause 3.2.3.1); and a nilled element whose
// declaration has a fixed value (clause 3.2.3.2).
func TestDecodeNillableInvalid(t *testing.T) {
	const plain = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	rows := []struct {
		name, decls, body string
		into              target
	}{
		{"xsi:nil false on a declaration that is not nillable", plain, `<s:a xsi:nil="false">x</s:a>`, into[struct{ A string }]()},
		{"xsi:nil true on a declaration that is not nillable", plain, `<s:a xsi:nil="true"/>`, into[struct{ A string }]()},
		{"a nilled element with a character child", nilRootSimpleSchema, ``, into[nillable[nilRootSimple, struct{}]](nilArms[nilRootSimple, struct{}]())},
		{"a nilled element with an element child", nilAltSchema, `<s:a xsi:nil="true"><s:b>1</s:b></s:a>`,
			into[struct{ Choice nilPick }](nilArms[string, struct{}](), WithBranches[nilPick](nilPickA{}, nilPickB{}))},
		{"a nilled element of a fixed value", nilFieldsSchema, `<s:id>1</s:id><s:tag>a</s:tag><s:price k="q">p</s:price><s:h>h</s:h><s:next/><s:size>1</s:size><s:c xsi:nil="true"/>`,
			into[nilFields](nilFieldsOptions...)},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf("", row.body)
			if row.decls == nilRootSimpleSchema {
				data = instanceOf(` xsi:nil="true"`, "5")
			}
			got, err := row.into.decode(schema, data)
			if got != nil || len(ViolationsOf(err)) == 0 {
				t.Fatalf("Decode = %v, %v, want no value and validate's charge", got, err)
			}
			if rule, _ := xsderr.RuleOf(err); rule != xsderr.Rule("cvc-elt") {
				t.Errorf("rule = %q, want cvc-elt", rule)
			}
		})
	}
}
