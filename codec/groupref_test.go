package codec

import (
	"bytes"
	"reflect"
	"testing"
)

// The group-ref schemas, each declaring s:r whose content {particle} is a
// <group ref>: gRefSeqSchema's to the sequence group g of a and an optional
// b; gRefChoiceSchema's to the choice group g of a or b; gRefShareSchema's
// r holds x of type t1 and y of type t2, whose contents are each a ref to
// the sequence group g of n, of an anonymous complex type; and
// gRefSelfSchema's r, warden F1's shape, refers to the sequence group g of
// an optional e, whose anonymous type's content refers to g again, so e's
// type reaches itself through the ref alone.
const (
	gRefSeqSchema = `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int" minOccurs="0"/></xs:sequence></xs:group>
  <xs:element name="r"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`
	gRefChoiceSchema = `<xs:group name="g"><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:choice></xs:group>
  <xs:element name="r"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`
	gRefShareSchema = `<xs:group name="g"><xs:sequence><xs:element name="n"><xs:complexType><xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:group>
  <xs:complexType name="t1"><xs:group ref="s:g"/></xs:complexType><xs:complexType name="t2"><xs:group ref="s:g"/></xs:complexType>
  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="x" type="s:t1"/><xs:element name="y" type="s:t2"/></xs:sequence></xs:complexType></xs:element>`
	gRefSelfSchema = `<xs:group name="g"><xs:sequence><xs:element name="e" minOccurs="0"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element></xs:sequence></xs:group>
  <xs:element name="r"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`
)

// The types codegen emits for the group-ref schemas, in shape.
type (
	groupSeqR struct {
		A string
		B *int32
	}
	groupChoiceR interface{ isGroupChoiceR() }
	groupChoiceA struct{ A string }
	groupChoiceB struct{ B int32 }
	groupShareR  struct {
		X groupShareT1
		Y groupShareT2
	}
	groupShareT1 struct{ N groupShareN }
	groupShareT2 struct{ N groupShareN }
	groupShareN  struct{ V string }
	groupSelfR   struct{ E *groupSelfE }
	groupSelfE   struct{ E *groupSelfE }
)

func (groupChoiceA) isGroupChoiceR() {}
func (groupChoiceB) isGroupChoiceR() {}

// TestGroupRefTypesAreCodegens pins the types above to those codegen emits
// for each group-ref schema, in shape (codegen/doc.go "# Particles"): the
// referenced group's fields or sealed interface in place of the ref, and,
// since item 6 compares T by shape alone, codegen's one declaration of a
// type nested in a model group definition, which every <group ref> to that
// group shares, gRefShareSchema's N held by T1's field and T2's alike and
// gRefSelfSchema's E by R's and its own.
func TestGroupRefTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls string
		want  []string
		once  []string
	}{
		{gRefSeqSchema, []string{"type R struct {\n\tA string\n\tB *int32\n}\n"}, nil},
		{gRefChoiceSchema, []string{
			"type R interface {\n\tisR()\n}\n",
			"type RA struct {\n\tA string\n}\n\nfunc (RA) isR() {}\n",
			"type RB struct {\n\tB int32\n}\n\nfunc (RB) isR() {}\n",
		}, nil},
		{gRefShareSchema, []string{
			"type R struct {\n\tX T1\n\tY T2\n}\n",
			"type T1 struct {\n\tN N\n}\n",
			"type T2 struct {\n\tN N\n}\n",
			"type N struct {\n\tV string\n}\n",
		}, []string{"type N struct"}},
		{gRefSelfSchema, []string{"type R struct {\n\tE *E\n}\n", "type E struct {\n\tE *E\n}\n"}, []string{"type E struct"}},
	}
	for _, row := range rows {
		gen := generated(t, schemaOf(t, row.decls))
		for _, decl := range row.want {
			if !bytes.Contains(gen, []byte(decl)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
			}
		}
		for _, decl := range row.once {
			if n := bytes.Count(gen, []byte(decl)); n != 1 {
				t.Errorf("codegen emits\n%s\nwhich declares %q %d times, want once", gen, decl, n)
			}
		}
	}
}

// TestDecodeGroupRefs pins the <group ref> content lift (§3.7.2
// declare-namedModelGroup: the ref's {term} is the definition's {model
// group}, the occurrence the ref's own; §3.4.2.3.3 dcl.ctd.ctcc.common
// clause 2.2): a ref to a sequence group decodes into the struct of the
// group's fields, a ref to a choice group into its sealed interface, two
// types referring to one group share the struct of the group's nested
// anonymous type, and a type that reaches itself through a ref alone decodes
// to the instance's depth. CheckDecode returns nil for each row's T. Each row
// declines as a <group ref> content particle with the lift removed.
func TestDecodeGroupRefs(t *testing.T) {
	i := func(n int32) *int32 { return &n }
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"a ref to a sequence group", gRefSeqSchema, "<s:a>x</s:a><s:b> 7 </s:b>", into[groupSeqR](), groupSeqR{A: "x", B: i(7)}},
		{"a ref to a sequence group, its optional particle absent", gRefSeqSchema, "<s:a>x</s:a>", into[groupSeqR](), groupSeqR{A: "x"}},
		{"a ref to a choice group", gRefChoiceSchema, "<s:b>3</s:b>", into[groupChoiceR](WithBranches[groupChoiceR](groupChoiceA{}, groupChoiceB{})),
			groupChoiceR(groupChoiceB{B: 3})},
		{"two types referring to one group, sharing its nested type", gRefShareSchema, "<s:x><s:n><s:v>1</s:v></s:n></s:x><s:y><s:n><s:v>2</s:v></s:n></s:y>",
			into[groupShareR](), groupShareR{X: groupShareT1{N: groupShareN{V: "1"}}, Y: groupShareT2{N: groupShareN{V: "2"}}}},
		{"a type reaching itself through a ref alone", gRefSelfSchema, "<s:e><s:e/></s:e>", into[groupSelfR](), groupSelfR{E: &groupSelfE{E: &groupSelfE{}}}},
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
