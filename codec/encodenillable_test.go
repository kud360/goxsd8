package codec

import (
	"errors"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/xsd"
)

// The encode-only nillable schemas, each declaring s:r: nilStringRootSchema's
// of xs:string; nilFixedRootSchema's of xs:string with a fixed value, which
// codegen emits unwrapped; nilFlatSeqSchema's a, a nillable basic particle of
// a flattened sequence; nilInnerSchema's a, of a local complex type holding
// the nillable n; nilSharedNameSchema's c, of the complex type r holding the
// nillable n; nilListSchema's of lst, which holds itself through the repeated
// nillable n; and nilRequiredUseSchema's of a simple content whose attribute
// use of xsi:nil is required.
const (
	nilStringRootSchema = `<xs:element name="r" type="xs:string" nillable="true"/>`
	nilFixedRootSchema  = `<xs:element name="r" type="xs:string" nillable="true" fixed="A"/>`
	nilTestedRootSchema = `<xs:element name="r" type="xs:string" nillable="true"><xs:alternative test="@xsi:nil" type="xs:string"/><xs:alternative test="count(@k) &gt; 0" type="xs:string"/></xs:element>`
	nilFlatSeqSchema    = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0" nillable="true"/></xs:sequence></xs:complexType></xs:element>`
	nilInnerSchema      = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a"><xs:complexType><xs:sequence><xs:element name="n" type="xs:string" nillable="true"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
	nilSharedNameSchema = `<xs:complexType name="r"><xs:sequence><xs:element name="n" type="xs:string" nillable="true"/></xs:sequence></xs:complexType>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="c" type="s:r"/></xs:sequence></xs:complexType></xs:element>`
	nilListSchema        = `<xs:complexType name="lst"><xs:sequence><xs:element name="n" type="s:lst" nillable="true" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType><xs:element name="r" type="s:lst"/>`
	nilRequiredUseSchema = `<xs:import namespace="http://www.w3.org/2001/XMLSchema-instance"/>` +
		`<xs:complexType name="ri"><xs:simpleContent><xs:extension base="xs:int"><xs:attribute ref="xsi:nil" use="required"/></xs:extension></xs:simpleContent></xs:complexType>` +
		`<xs:element name="r" type="s:ri" nillable="true"/>`
)

// The types codegen emits for those schemas' s:r, in shape, each named here
// by its schema.
type (
	nilStringRoot  string
	nilFlatSeq     struct{ Sequence []nilFlatSeqItem }
	nilFlatSeqItem interface{ isNilFlatSeqItem() }
	nilFlatSeqA    struct{ A nillable[string, struct{}] }
	nilInner       struct {
		A struct{ N nillable[string, struct{}] }
	}
	nilSharedName struct {
		C struct{ N nillable[string, struct{}] }
	}
	nilList struct{ N []nillable[nilList, struct{}] }
)

func (nilFlatSeqA) isNilFlatSeqItem() {}

// xsiNilTrue is the xsi:nil attribute of a nilled element as Append writes
// it, its prefix declared where the element is the first to need it.
const xsiNilTrue = ` xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:nil="true"`

// TestAppendNillable pins the nillable lift (Structures §3.3.4.3 cvc-elt
// clause 3): a Nillable over a declaration Decode reads is written arm by arm,
// the Content arm as the element its T would write and the Nilled arm as an
// element carrying its Attributes and then xsi:nil true and no character or
// element child (clause 3.2.3.1), and each document validate accepts and
// Decode turns back into the value. A nilled element whose type requires
// content, the s:a of nilRootComplexSchema's r or an xs:int, is written empty
// (cvc-complex-type clause 1, cvc-type clause 3.1.3), and a nilled one keeps
// its attributes, a required one and a defaulted one written (cvc-complex-type
// clauses 2 to 4). A nillable declaration with a fixed value is written as
// plain T (cvc-elt clause 3.2.3.2), and one of type xs:anyType itself as a
// declaration that is not nillable is, its AnyType holding no xsi:nil.
func TestAppendNillable(t *testing.T) {
	simple := schemaOf(t, nilRootSimpleSchema)
	simpleArms := nilArms[nilRootSimple, struct{}]()
	t.Run("a simple root, Content", func(t *testing.T) {
		roundTrip(t, simple, rootName, nillable[nilRootSimple, struct{}](content[nilRootSimple, struct{}]{Content: 5}), `<r xmlns="urn:s">5</r>`, deepEqual, simpleArms)
	})
	t.Run("a simple root, Nilled", func(t *testing.T) {
		roundTrip(t, simple, rootName, nillable[nilRootSimple, struct{}](nilled[nilRootSimple, struct{}]{}), `<r xmlns="urn:s"`+xsiNilTrue+`/>`, deepEqual, simpleArms)
	})
	t.Run("an xs:string root, Content and Nilled", func(t *testing.T) {
		schema, arms := schemaOf(t, nilStringRootSchema), nilArms[nilStringRoot, struct{}]()
		roundTrip(t, schema, rootName, nillable[nilStringRoot, struct{}](content[nilStringRoot, struct{}]{Content: ""}), `<r xmlns="urn:s"/>`, deepEqual, arms)
		roundTrip(t, schema, rootName, nillable[nilStringRoot, struct{}](nilled[nilStringRoot, struct{}]{}), `<r xmlns="urn:s"`+xsiNilTrue+`/>`, deepEqual, arms)
	})
	complexRoot := schemaOf(t, nilRootComplexSchema)
	complexArms := nilArms[nilRootComplex, nilRootComplexAttributes]()
	t.Run("a complex root, Content", func(t *testing.T) {
		v := nillable[nilRootComplex, nilRootComplexAttributes](content[nilRootComplex, nilRootComplexAttributes]{Content: nilRootComplex{A: "x", K: "d"}})
		roundTrip(t, complexRoot, rootName, v, `<r xmlns="urn:s" k="d"><a>x</a></r>`, deepEqual, complexArms)
	})
	t.Run("a complex root whose type requires content, Nilled with its attribute", func(t *testing.T) {
		v := nillable[nilRootComplex, nilRootComplexAttributes](nilled[nilRootComplex, nilRootComplexAttributes]{Attributes: nilRootComplexAttributes{K: "v"}})
		roundTrip(t, complexRoot, rootName, v, `<r xmlns="urn:s" k="v"`+xsiNilTrue+`/>`, deepEqual, complexArms)
	})
	t.Run("a choice root, Content and Nilled", func(t *testing.T) {
		schema := schemaOf(t, nilRootPickSchema)
		opts := []Option{nilArms[nilRootPick, struct{}](), WithBranches[nilRootPick](nilRootPickX{}, nilRootPickY{})}
		roundTrip(t, schema, rootName, nillable[nilRootPick, struct{}](content[nilRootPick, struct{}]{Content: nilRootPickY{Y: 3}}), `<r xmlns="urn:s"><y>3</y></r>`, deepEqual, opts...)
		roundTrip(t, schema, rootName, nillable[nilRootPick, struct{}](nilled[nilRootPick, struct{}]{}), `<r xmlns="urn:s"`+xsiNilTrue+`/>`, deepEqual, opts...)
	})
	fields := schemaOf(t, nilFieldsSchema)
	t.Run("fields, a repeated one mixing both arms in order", func(t *testing.T) {
		v := nilFields{
			Id:    nilled[int32, struct{}]{},
			Note:  content[string, struct{}]{Content: "n"},
			Tag:   []nillable[string, struct{}]{content[string, struct{}]{Content: "a"}, nilled[string, struct{}]{}, content[string, struct{}]{Content: "b"}},
			Price: nilled[nilPrice, nilPriceAttributes]{Attributes: nilPriceAttributes{Cur: "GBP", K: "q"}},
			H:     content[nilH, struct{}]{Content: "h"},
			Next:  content[nilLink, struct{}]{Content: nilLink{Next: nilled[nilLink, struct{}]{}}},
			Size:  nilled[int32, struct{}]{},
			O:     content[nilOpt, struct{}]{Content: nilOpt{Choice: nilOptY{Y: 2}}},
			C:     "A",
		}
		roundTrip(t, fields, rootName, v, `<r xmlns="urn:s"><id`+xsiNilTrue+`/><note>n</note><tag>a</tag><tag xmlns:p1="http://www.w3.org/2001/XMLSchema-instance" p1:nil="true"/><tag>b</tag>`+
			`<price cur="GBP" k="q"`+xsiNilTrue+`/><h>h</h><next><next`+xsiNilTrue+`/></next><size`+xsiNilTrue+`/><o><y>2</y></o><c>A</c></r>`, deepEqual, nilFieldsOptions...)
	})
	t.Run("fields, the other arms", func(t *testing.T) {
		v := nilFields{
			Id:    content[int32, struct{}]{Content: 1},
			Tag:   []nillable[string, struct{}]{nilled[string, struct{}]{}},
			Price: content[nilPrice, nilPriceAttributes]{Content: nilPrice{Value: "p", Cur: "EUR", K: "q"}},
			H:     nilled[nilH, struct{}]{},
			Next:  nilled[nilLink, struct{}]{},
			Size:  content[int32, struct{}]{Content: 3},
			O:     nilled[nilOpt, struct{}]{},
			C:     "A",
		}
		roundTrip(t, fields, rootName, v, "", deepEqual, nilFieldsOptions...)
	})
	strArms := nilArms[string, struct{}]()
	t.Run("mixed content, Nilled and Content", func(t *testing.T) {
		schema := schemaOf(t, nilMixedSchema)
		roundTrip(t, schema, rootName, nilMixed{Text: "a", Em: &mixedChild[nillable[string, struct{}]]{Element: nilled[string, struct{}]{}, Tail: "b"}}, "", deepEqual, strArms)
		roundTrip(t, schema, rootName, nilMixed{Text: "a", Em: &mixedChild[nillable[string, struct{}]]{Element: content[string, struct{}]{Content: "e"}, Tail: "b"}}, "", deepEqual, strArms)
	})
	t.Run("a flattened choice's branch", func(t *testing.T) {
		v := nilFlat{Choice: []nilFlatChoice{nilFlatA{A: nilled[string, struct{}]{}}, nilFlatB{B: 1}, nilFlatA{A: content[string, struct{}]{Content: "x"}}}}
		roundTrip(t, schemaOf(t, nilFlatSchema), rootName, v, "", deepEqual, strArms, WithBranches[nilFlatChoice](nilFlatA{}, nilFlatB{}))
	})
	t.Run("a flattened sequence's basic particle", func(t *testing.T) {
		v := nilFlatSeq{Sequence: []nilFlatSeqItem{nilFlatSeqA{A: nilled[string, struct{}]{}}, nilFlatSeqA{A: content[string, struct{}]{Content: "x"}}}}
		roundTrip(t, schemaOf(t, nilFlatSeqSchema), rootName, v, "", deepEqual, strArms, WithBranches[nilFlatSeqItem](nilFlatSeqA{}))
	})
	t.Run("an occurrence struct's field", func(t *testing.T) {
		v := nilOccur{Sequence: []nilOccurItem{{A: nilled[string, struct{}]{}, B: 1}, {A: content[string, struct{}]{Content: "x"}, B: 2}}}
		roundTrip(t, schemaOf(t, nilOccurSchema), rootName, v, "", deepEqual, strArms)
	})
	t.Run("a choice's alternative", func(t *testing.T) {
		opts := []Option{strArms, WithBranches[nilPick](nilPickA{}, nilPickB{})}
		type alt struct{ Choice nilPick }
		roundTrip(t, schemaOf(t, nilAltSchema), rootName, alt{Choice: nilPickA{A: nilled[string, struct{}]{}}}, "", deepEqual, opts...)
		roundTrip(t, schemaOf(t, nilAltSchema), rootName, alt{Choice: nilPickA{A: content[string, struct{}]{Content: "v"}}}, "", deepEqual, opts...)
	})
	t.Run("a nested complex type's field", func(t *testing.T) {
		v := nilInner{}
		v.A.N = nilled[string, struct{}]{}
		roundTrip(t, schemaOf(t, nilInnerSchema), rootName, v, `<r xmlns="urn:s"><a><n`+xsiNilTrue+`/></a></r>`, deepEqual, strArms)
	})
	t.Run("a field of a type sharing its element's name", func(t *testing.T) {
		v := nilSharedName{}
		v.C.N = content[string, struct{}]{Content: "x"}
		roundTrip(t, schemaOf(t, nilSharedNameSchema), rootName, v, `<r xmlns="urn:s"><c><n>x</n></c></r>`, deepEqual, strArms)
	})
	// The Nilled element is taken by its particle's attribution slot, as the
	// Content arm's is: untaken, the wildcard's element is attributed where
	// the content model expects s:a, item 6.
	t.Run("a particle before a wildcard, Nilled", func(t *testing.T) {
		type wildAfter struct {
			A   nillable[string, struct{}]
			Any anyElement
		}
		schema := schemaOf(t, `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string" nillable="true"/><xs:any processContents="skip"/></xs:sequence></xs:complexType></xs:element>`)
		v := wildAfter{A: nilled[string, struct{}]{}, Any: anyElement{Space: "urn:o", Local: "w"}}
		roundTrip(t, schema, rootName, v, "", deepEqual, strArms)
	})
	// The Nilled arm's xsi:nil is in its {type table}'s selection input
	// (Structures §3.12.4 key-cta-ta-select): it decides the first {test}, so
	// the second, which package xpath leaves undecided, is never reached; the
	// Content arm, holding no xsi:nil, reaches it and is refused.
	t.Run("a {type table} whose first test reads xsi:nil", func(t *testing.T) {
		schema, arms := schemaOf(t, nilTestedRootSchema), nilArms[nilStringRoot, struct{}]()
		roundTrip(t, schema, rootName, nillable[nilStringRoot, struct{}](nilled[nilStringRoot, struct{}]{}), `<r xmlns="urn:s"`+xsiNilTrue+`/>`, deepEqual, arms)
		ttRefused(nillable[nilStringRoot, struct{}](content[nilStringRoot, struct{}]{Content: "x"}), "element {urn:s}r declared at s.xsd:1:181, at nillable[github.com/kud360/goxsd8/codec.nilStringRoot,struct {}].(content[github.com/kud360/goxsd8/codec.nilStringRoot,struct {}]).Content written as the type of its {type table} alternative 1: its {type table}'s selection over the attributes Append writes and its [inherited attributes] is undecided, a {test} package xpath does not evaluate", arms)(t, schema)
	})
	t.Run("a fixed value, plain T", func(t *testing.T) {
		roundTrip(t, schemaOf(t, nilFixedRootSchema), rootName, nilStringRoot("A"), `<r xmlns="urn:s">A</r>`, deepEqual)
	})
	t.Run("xs:anyType itself, holding no xsi:nil", func(t *testing.T) {
		roundTrip(t, schemaOf(t, nilRootAnySchema), rootName, rootAny{Text: "t"}, `<p1:r xmlns:p1="urn:s">t</p1:r>`, deepEqual)
		roundTrip(t, schemaOf(t, nilOneAnySchema), rootName, oneAny{A: anyType{Text: "t"}}, "", deepEqual)
		roundTrip(t, schemaOf(t, nilGroupAnySchema), rootName, groupAny{H: groupAnyM{M: member{Text: "t"}}}, "", deepEqual, groupAnyBranches)
	})
}

// TestAppendNillableDeclines pins item 4 of Append's doc for a nillable
// declaration Append writes no value of: one of type xs:anyType itself still
// declines xsi:nil among its own Attributes, item 7 by constructXsiAttribute,
// as for one that is not nillable, since codegen wraps it in no Nillable; and
// one whose type's attribute use of xsi:nil has {required} true declines by
// constructNillable, since a Content arm holds no xsi:nil.
func TestAppendNillableDeclines(t *testing.T) {
	xsiNil := []anyAttribute{{xsd.XMLSchemaInstanceNS, "nil", "true"}}
	_, err := appendChecked(t, nil, schemaOf(t, nilRootAnySchema), strict.New(), rootName, rootAny{Attributes: xsiNil})
	if d := asDecline(t, err, "codec does not encode "+constructXsiAttribute.name()); d.Construct() != (Construct{c: constructXsiAttribute}) {
		t.Errorf("Construct = %v, want constructXsiAttribute", d.Construct())
	}
	out, err := appendChecked(t, nil, schemaOf(t, nilRequiredUseSchema), strict.New(), rootName, struct{}{})
	d := asDecline(t, err, "codec does not encode an element declaration with {nillable} true: element {urn:s}r declared at ")
	holds(t, err, ", whose type's attribute use of xsi:nil has {required} true, ")
	if d.Construct() != (Construct{c: constructNillable}) || out != nil {
		t.Errorf("Construct = %v, wrote %q, want constructNillable and nothing", d.Construct(), out)
	}
}

// TestAppendNillableInvalid pins item 6 of Append's doc at a Nillable: a nil
// one where nil does not mean absent, a pointer to an arm, and a value that
// holds itself through a []Nillable field, its Content reaching its own item
// again, each writing nothing.
func TestAppendNillableInvalid(t *testing.T) {
	fields := schemaOf(t, nilFieldsSchema)
	valid := nilFields{
		Id: content[int32, struct{}]{Content: 1}, Tag: []nillable[string, struct{}]{}, Price: nilled[nilPrice, nilPriceAttributes]{Attributes: nilPriceAttributes{K: "q"}},
		H: nilled[nilH, struct{}]{}, Next: nilled[nilLink, struct{}]{}, Size: nilled[int32, struct{}]{}, C: "A",
	}
	t.Run("a nil Nillable", func(t *testing.T) {
		v := valid
		v.Id = nil
		_, err := appendChecked(t, nil, fields, strict.New(), rootName, v, nilFieldsOptions...)
		invalidValue(t, err, "codec: Append: nilFields.Id is nil, codegen's Nillable over element {urn:s}id declared at ")
	})
	t.Run("a pointer to an arm", func(t *testing.T) {
		v := valid
		v.Id = &content[int32, struct{}]{Content: 1}
		_, err := appendChecked(t, nil, fields, strict.New(), rootName, v, nilFieldsOptions...)
		invalidValue(t, err, "codec: Append: nilFields.Id holds a *codec.content[int32,struct {}], which the WithBranches set of codec.nillable[int32,struct {}] places at neither Content nor Nilled")
	})
	t.Run("a value that holds itself", func(t *testing.T) {
		items := make([]nillable[nilList, struct{}], 1)
		items[0] = content[nilList, struct{}]{Content: nilList{N: items}}
		_, err := appendChecked(t, nil, schemaOf(t, nilListSchema), strict.New(), rootName, nilList{N: items}, nilArms[nilList, struct{}]())
		invalidValue(t, err, "codec: Append: nilList.N[0].(content[github.com/kud360/goxsd8/codec.nilList,struct {}]).Content.N[0] is the value at nilList.N[0], whose element is still being written")
	})
}

// TestNillableDeclinesOnError pins the one domain of nillableDeclines no
// parsed schema reaches, Finalize resolving every slot: a nillable
// declaration whose goshape.GoverningTypes errs declines, for Decode and
// Append alike, and the same declaration with its one governing type does not.
func TestNillableDeclinesOnError(t *testing.T) {
	schema := schemaOf(t, nilRootSimpleSchema)
	decl, _ := schema.Element(rootName)
	types, errorSlot, err := goshape.GoverningTypes(schema, decl)
	if err != nil {
		t.Fatalf("GoverningTypes: %v", err)
	}
	if nillableDeclines(decl, true, types, errorSlot, nil) {
		t.Fatalf("nillableDeclines = true for GoverningTypes %v, want false for one governing type", types)
	}
	if !nillableDeclines(decl, true, types, errorSlot, errors.New("x")) {
		t.Errorf("nillableDeclines = false, want true where GoverningTypes errs")
	}
}

// TestAppendNillableKeyMember pins item 8 of Append's doc for a key whose
// field selects an element of a nillable declaration: validate charges
// cvc-identity-constraint clause 4.2.3, one of the clauses validate's package
// doc states it charges, to which item 8 points.
func TestAppendNillableKeyMember(t *testing.T) {
	type keyed struct{ A nillable[string, struct{}] }
	schema := schemaOf(t, `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string" nillable="true"/></xs:sequence></xs:complexType>`+
		`<xs:key name="k"><xs:selector xpath="."/><xs:field xpath="s:a"/></xs:key></xs:element>`)
	_, err := appendChecked(t, nil, schema, strict.New(), rootName, keyed{A: content[string, struct{}]{Content: "x"}}, nilArms[string, struct{}]())
	invalidValue(t, err, "[cvc-identity-constraint]", "which cvc-identity-constraint clause 4.2.3 forbids for a key")
}
