package codec

import (
	"reflect"
	"testing"
)

// The schemas whose {type table}s hold two or more anonymous governing types,
// each declaring s:r. atTwoSchema's r, of ttTypes' base, selects an anonymous
// extension of base adding x, an xs:string, where its kind is "x", another
// adding y, an xs:int, where it is "y", and its declared base otherwise.
// atUnionSchema's r, of an anonymous union of xs:int and xs:boolean, selects
// an anonymous complex type of simple content extending xs:int, a member of
// that union, with an attribute k, where it carries k, and its declared union
// otherwise. atInheritSchema's r carries type, inheritable, and holds e, of
// base, which selects atTwoSchema's two anonymous extensions on its inherited
// type alone, e itself declaring no such attribute.
var (
	atTwoSchema     = ttTypes + `<xs:element name="r" type="s:base">` + atTwoAlternatives(`@kind = 'x'`, `@kind = 'y'`) + `</xs:element>`
	atUnionSchema   = `<xs:element name="r"><xs:simpleType><xs:union memberTypes="xs:int xs:boolean"/></xs:simpleType><xs:alternative test="@k"><xs:complexType><xs:simpleContent><xs:extension base="xs:int"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType></xs:alternative></xs:element>`
	atInheritSchema = ttTypes + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="s:base">` + atTwoAlternatives(`@type = 'x'`, `@type = 'y'`) + `</xs:element></xs:sequence><xs:attribute name="type" type="xs:string" inheritable="true"/></xs:complexType></xs:element>`
)

// atTwoAlternatives is two <alternative>s, of tests x and y, each holding an
// anonymous extension of base: x's adding x, an xs:string, and y's adding y,
// an xs:int.
func atTwoAlternatives(x, y string) string {
	return `<xs:alternative test="` + x + `"><xs:complexType><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType></xs:alternative>` +
		`<xs:alternative test="` + y + `"><xs:complexType><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="y" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType></xs:alternative>`
}

// The types codegen emits for the schemas above, which each row's CheckDecode
// pins: atTwoSchema's R, RAlternative1, RAlternative1Type, RAlternative2,
// RAlternative2Type and RBase are atR to atRBase; atUnionSchema's R,
// RAlternative1, RAlternative1Type, RDefault and RDefaultTypeUnion with its
// branches are atU to atUBoolean; atInheritSchema's R, E, EAlternative1,
// EAlternative2 and EBase are atInh to atEBase, the alternatives' types
// atTwoSchema's.
type (
	atR        interface{ isAtR() }
	atRAlt1    struct{ Alternative1 atAlt1Type }
	atAlt1Type struct {
		A    string
		X    string
		Kind *string
	}
	atRAlt2    struct{ Alternative2 atAlt2Type }
	atAlt2Type struct {
		A    string
		Y    int32
		Kind *string
	}
	atRBase struct{ Base ttBase }

	atU         interface{ isAtU() }
	atUAlt1     struct{ Alternative1 atUAlt1Type }
	atUAlt1Type struct {
		Value int32
		K     *string
	}
	atUDefault struct{ Default atUnion }
	atUnion    interface{ isAtUnion() }
	atUInt     int32
	atUBoolean bool

	atInh struct {
		E    atE
		Type *string
	}
	atE     interface{ isAtE() }
	atEAlt1 struct{ Alternative1 atAlt1Type }
	atEAlt2 struct{ Alternative2 atAlt2Type }
	atEBase struct{ Base ttBase }
)

func (atRAlt1) isAtR()        {}
func (atRAlt2) isAtR()        {}
func (atRBase) isAtR()        {}
func (atUAlt1) isAtU()        {}
func (atUDefault) isAtU()     {}
func (atUInt) isAtUnion()     {}
func (atUBoolean) isAtUnion() {}
func (atEAlt1) isAtE()        {}
func (atEAlt2) isAtE()        {}
func (atEBase) isAtE()        {}

// TestDecodeAnonymousTypeTable pins the lift of two or more anonymous
// governing types: each element fills the branch of the slot
// validate.Validator.ConditionallySelectedIndex selects, an anonymous type's
// branch found by its slot and a named one's by its name: two anonymous
// alternatives beside a named default, an anonymous alternative beside an
// anonymous declared type, the default slot's, and two anonymous alternatives
// a {test} selects between on an inherited attribute alone.
func TestDecodeAnonymousTypeTable(t *testing.T) {
	str := func(s string) *string { return &s }
	two := into[atR](WithBranches[atR](atRAlt1{}, atRAlt2{}, atRBase{}))
	union := into[atU](WithBranches[atU](atUAlt1{}, atUDefault{}), WithBranches[atUnion](atUInt(0), atUBoolean(false)))
	inh := into[atInh](WithBranches[atE](atEAlt1{}, atEAlt2{}, atEBase{}))
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"the first anonymous alternative", atTwoSchema, ` kind="x"`, "<s:a>a</s:a><s:x>b</s:x>", two,
			atR(atRAlt1{Alternative1: atAlt1Type{A: "a", X: "b", Kind: str("x")}})},
		{"the second anonymous alternative", atTwoSchema, ` kind="y"`, "<s:a>a</s:a><s:y>3</s:y>", two,
			atR(atRAlt2{Alternative2: atAlt2Type{A: "a", Y: 3, Kind: str("y")}})},
		{"the named default beside two anonymous alternatives", atTwoSchema, ` kind="z"`, "<s:a>a</s:a>", two,
			atR(atRBase{Base: ttBase{A: "a", Kind: str("z")}})},
		{"an anonymous alternative beside an anonymous declared union", atUnionSchema, ` k="z"`, "5", union,
			atU(atUAlt1{Alternative1: atUAlt1Type{Value: 5, K: str("z")}})},
		{"the anonymous declared union, its boolean member", atUnionSchema, "", "true", union,
			atU(atUDefault{Default: atUBoolean(true)})},
		{"the anonymous declared union, its int member", atUnionSchema, "", "7", union,
			atU(atUDefault{Default: atUInt(7)})},
		{"an inherited attribute selecting the second anonymous alternative", atInheritSchema, ` type="y"`, "<s:e><s:a>a</s:a><s:y>3</s:y></s:e>", inh,
			atInh{E: atEAlt2{Alternative2: atAlt2Type{A: "a", Y: 3}}, Type: str("y")}},
		{"an inherited attribute selecting the first anonymous alternative", atInheritSchema, ` type="x"`, "<s:e><s:a>a</s:a><s:x>b</s:x></s:e>", inh,
			atInh{E: atEAlt1{Alternative1: atAlt1Type{A: "a", X: "b"}}, Type: str("x")}},
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
