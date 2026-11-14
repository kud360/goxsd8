package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// The all schemas, each declaring s:r, whose content {particle} is an all
// group codegen splices (codegen/doc.go "# Particles"), so that its fields are
// those a sequence of the same particles gives: allABSchema's all(a, b);
// allOptSchema's all(a, b), b with {min occurs} 0; allRepSchema's all(a, b), a
// with {max occurs} unbounded (§3.8.6.2 cos-all-limited admits it in XSD 1.1);
// allRefSchema's a <group ref> to the all group g of a, b; allSplicedSchema's
// an all with {min occurs} 0 over an optional a and an optional b, emptiable
// (goshape.Splices); allWildSchema's all(a, any ##other); allNestedWildSchema's
// all(a, g), g a <group ref> to the all group of one ##other wildcard, nested
// with {min occurs} and {max occurs} 1 (cos-all-limited clause 1.3), so the
// wildcard's Matcher position is [1 0]; and allExtSchema's r is of t2, an
// extension of t, all(a), adding all(b), so t2's {particle} is all(a, b)
// (§3.4.2.3.3 dcl.ctd.ctcc.common clause 4.2.3.2).
const (
	allABSchema  = `<xs:element name="r"><xs:complexType><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:all></xs:complexType></xs:element>`
	allOptSchema = `<xs:element name="r"><xs:complexType><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int" minOccurs="0"/></xs:all></xs:complexType></xs:element>`
	allRepSchema = `<xs:element name="r"><xs:complexType><xs:all><xs:element name="a" type="xs:string" maxOccurs="unbounded"/><xs:element name="b" type="xs:int"/></xs:all></xs:complexType></xs:element>`
	allRefSchema = `<xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:all></xs:group>
  <xs:element name="r"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`
	allSplicedSchema    = `<xs:element name="r"><xs:complexType><xs:all minOccurs="0"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:int" minOccurs="0"/></xs:all></xs:complexType></xs:element>`
	allWildSchema       = `<xs:element name="r"><xs:complexType><xs:all><xs:element name="a" type="xs:string"/><xs:any namespace="##other" processContents="lax"/></xs:all></xs:complexType></xs:element>`
	allNestedWildSchema = `<xs:group name="g"><xs:all><xs:any namespace="##other" processContents="lax"/></xs:all></xs:group>
  <xs:element name="r"><xs:complexType><xs:all><xs:element name="a" type="xs:string"/><xs:group ref="s:g"/></xs:all></xs:complexType></xs:element>`
	allExtSchema = `<xs:complexType name="t"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>
  <xs:complexType name="t2"><xs:complexContent><xs:extension base="s:t"><xs:all><xs:element name="b" type="xs:int"/></xs:all></xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="r" type="s:t2"/>`
)

// The types codegen emits for the all schemas' s:r, in shape, which
// TestAllTypesAreCodegens pins.
type (
	allAB struct {
		A string
		B int32
	}
	allOpt struct {
		A string
		B *int32
	}
	allRep struct {
		A []string
		B int32
	}
	allSpliced struct {
		A *string
		B *int32
	}
	allWild struct {
		A   string
		Any anyElement
	}
)

// TestAllTypesAreCodegens pins the types above to the struct codegen emits
// for each all schema's s:r (codegen/doc.go "# Particles"): the all group's
// particles' fields, in particle order, as a sequence of them gives.
func TestAllTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls, want string
	}{
		{allABSchema, "type R struct {\n\tA string\n\tB int32\n}\n"},
		{allOptSchema, "type R struct {\n\tA string\n\tB *int32\n}\n"},
		{allRepSchema, "type R struct {\n\tA []string\n\tB int32\n}\n"},
		{allRefSchema, "type R struct {\n\tA string\n\tB int32\n}\n"},
		{allSplicedSchema, "type R struct {\n\tA *string\n\tB *int32\n}\n"},
		{allWildSchema, "type R struct {\n\tA   string\n\tAny AnyElement\n}\n"},
		{allNestedWildSchema, "type R struct {\n\tA   string\n\tAny AnyElement\n}\n"},
		{allExtSchema, "type R struct {\n\tA string\n\tB int32\n}\n"},
	}
	for _, row := range rows {
		gen := generated(t, schemaOf(t, row.decls))
		if !bytes.Contains(gen, []byte(row.want)) {
			t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, row.want)
		}
	}
}

// TestDecodeAll pins the all lift: a content all group goshape.Splices
// answers true for, inline, through a <group ref>, with {min occurs} 0 over
// an emptiable group, or an extension's merged all, decodes into the fields
// its own particles give, each child into the field of the particle it is
// attributed to (§3.4.4.4 key-att-to) whatever the instance order
// (§3.8.4.1.3 all-mg), a repeated particle's children appended in instance
// order, and a wildcard particle's child, inline or inside a nested all, into
// its own field. CheckDecode returns nil for each row's T.
func TestDecodeAll(t *testing.T) {
	i := func(n int32) *int32 { return &n }
	w := anyElement{Space: "urn:o", Local: "w"}
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"all(a, b) in particle order", allABSchema, "<s:a>x</s:a><s:b>1</s:b>", into[allAB](), allAB{A: "x", B: 1}},
		{"all(a, b) permuted", allABSchema, "<s:b>1</s:b><s:a>x</s:a>", into[allAB](), allAB{A: "x", B: 1}},
		{"an optional particle absent", allOptSchema, "<s:a>x</s:a>", into[allOpt](), allOpt{A: "x"}},
		{"an optional particle present, permuted", allOptSchema, "<s:b>2</s:b><s:a>x</s:a>", into[allOpt](), allOpt{A: "x", B: i(2)}},
		{"a repeated particle interleaved with another's child", allRepSchema, "<s:a>p</s:a><s:b>3</s:b><s:a>q</s:a>", into[allRep](), allRep{A: []string{"p", "q"}, B: 3}},
		{"a <group ref> to an all group, permuted", allRefSchema, "<s:b>4</s:b><s:a>x</s:a>", into[allAB](), allAB{A: "x", B: 4}},
		{"a spliced optional all, no children", allSplicedSchema, "", into[allSpliced](), allSpliced{}},
		{"a spliced optional all, permuted", allSplicedSchema, "<s:b>5</s:b><s:a>x</s:a>", into[allSpliced](), allSpliced{A: strPtr("x"), B: i(5)}},
		{"a wildcard before the element", allWildSchema, `<o:w xmlns:o="urn:o"/><s:a>x</s:a>`, into[allWild](), allWild{A: "x", Any: w}},
		{"a wildcard after the element", allWildSchema, `<s:a>x</s:a><o:w xmlns:o="urn:o"/>`, into[allWild](), allWild{A: "x", Any: w}},
		{"a wildcard inside a nested all, first", allNestedWildSchema, `<o:w xmlns:o="urn:o"/><s:a>x</s:a>`, into[allWild](), allWild{A: "x", Any: w}},
		{"an extension merging two alls, permuted", allExtSchema, "<s:b>6</s:b><s:a>x</s:a>", into[allAB](), allAB{A: "x", B: 6}},
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

// strPtr returns a pointer to s.
func strPtr(s string) *string { return &s }

// TestDecodeAllRefusesWhatValidateCharges pins that a permuted instance of
// all(a, b) missing its required a, which validate charges (§3.8.4.1.3
// all-mg), returns validate's Charged error and no value, though every child
// it holds is attributed to a field.
func TestDecodeAllRefusesWhatValidateCharges(t *testing.T) {
	schema := schemaOf(t, allABSchema)
	got, err := Decode[allAB](schema, strict.New(), instanceOf("", "<s:b>1</s:b>"))
	if got != nil || err == nil {
		t.Fatalf("Decode = (%v, %v), want no value and an error", got, err)
	}
	if !strings.HasPrefix(err.Error(), "codec: the instance is invalid against the schema: ") {
		t.Errorf("error %q does not open as Decode's Charged error", err)
	}
	var de *DeclineError
	if errors.As(err, &de) {
		t.Errorf("error %v is a decline, want a violation", err)
	}
}

// The optional all schemas, each declaring s:r, whose content {particle} is
// an all group with {min occurs} 0 over a model group that is not emptiable,
// which codegen does not splice (goshape.Splices) and holds in its occurrence
// field, All, a pointer to the struct for its one occurrence (codegen/doc.go
// "# Particles"): allOccASchema's all(a); allOccRefSchema's an optional
// <group ref> to the all group g of a; allOccABSchema's all(a, b);
// allOccRepSchema's all(a, b), a with {max occurs} unbounded;
// allOccNestedSchema's all(a, g), g a <group ref> to the all group of b,
// nested with {min occurs} and {max occurs} 1 (cos-all-limited clause 1.3),
// which codegen splices into the occurrence's struct; and allOccExtSchema's r
// is of t2, an extension of t, all(a) with {min occurs} 0, adding all(b) with
// {min occurs} 0, so t2's {particle} is one all(a, b) with {min occurs} 0
// (§3.4.2.3.3 dcl.ctd.ctcc.common clause 4.2.3.2).
const (
	allOccASchema   = `<xs:element name="r"><xs:complexType><xs:all minOccurs="0"><xs:element name="a" type="xs:string"/></xs:all></xs:complexType></xs:element>`
	allOccRefSchema = `<xs:group name="g"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:group>
  <xs:element name="r"><xs:complexType><xs:group ref="s:g" minOccurs="0"/></xs:complexType></xs:element>`
	allOccABSchema     = `<xs:element name="r"><xs:complexType><xs:all minOccurs="0"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:all></xs:complexType></xs:element>`
	allOccRepSchema    = `<xs:element name="r"><xs:complexType><xs:all minOccurs="0"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/><xs:element name="b" type="xs:int"/></xs:all></xs:complexType></xs:element>`
	allOccNestedSchema = `<xs:group name="g"><xs:all><xs:element name="b" type="xs:int"/></xs:all></xs:group>
  <xs:element name="r"><xs:complexType><xs:all minOccurs="0"><xs:element name="a" type="xs:string"/><xs:group ref="s:g"/></xs:all></xs:complexType></xs:element>`
	allOccExtSchema = `<xs:complexType name="t"><xs:all minOccurs="0"><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>
  <xs:complexType name="t2"><xs:complexContent><xs:extension base="s:t"><xs:all minOccurs="0"><xs:element name="b" type="xs:int"/></xs:all></xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="r" type="s:t2"/>`
)

// The types codegen emits for the optional all schemas' s:r, in shape, which
// TestAllOccurrenceTypesAreCodegens pins.
type (
	allOccA      struct{ All *allOccAItem }
	allOccAItem  struct{ A string }
	allOccAB     struct{ All *allOccABItem }
	allOccABItem struct {
		A string
		B int32
	}
	allOccRep     struct{ All *allOccRepItem }
	allOccRepItem struct {
		A []string
		B int32
	}
)

// TestAllOccurrenceTypesAreCodegens pins the types above to the struct
// codegen emits for each optional all schema's s:r (codegen/doc.go
// "# Particles"): the occurrence field All, a pointer to the struct of the
// all group's particles' fields: outer is R's declaration, in the
// occurrence struct's.
func TestAllOccurrenceTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls     string
		outer, in string
	}{
		{allOccASchema, "type R struct {\n\tAll *RAll\n}\n", "type RAll struct {\n\tA string\n}\n"},
		{allOccRefSchema, "type R struct {\n\tAll *RAll\n}\n", "type RAll struct {\n\tA string\n}\n"},
		{allOccABSchema, "type R struct {\n\tAll *RAll\n}\n", "type RAll struct {\n\tA string\n\tB int32\n}\n"},
		{allOccRepSchema, "type R struct {\n\tAll *RAll\n}\n", "type RAll struct {\n\tA []string\n\tB int32\n}\n"},
		{allOccNestedSchema, "type R struct {\n\tAll *RAll\n}\n", "type RAll struct {\n\tA string\n\tB int32\n}\n"},
		{allOccExtSchema, "type R struct {\n\tAll *T2All\n}\n", "type T2All struct {\n\tA string\n\tB int32\n}\n"},
	}
	for _, row := range rows {
		gen := generated(t, schemaOf(t, row.decls))
		for _, want := range []string{row.outer, row.in} {
			if !bytes.Contains(gen, []byte(want)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, want)
			}
		}
	}
}

// TestDecodeAllOccurrence pins the optional all lift: a content all group
// with {min occurs} 0 that goshape.Splices answers false for, inline, through
// an optional <group ref>, or an extension's merged all, decodes into its
// occurrence field: nil for an element with no children, the empty sequence
// the particle's {min occurs} 0 admits (§3.9.4.3 cvc-accept clause 3), and
// otherwise a pointer to one item holding every child, each in the field of
// the particle it is attributed to (§3.4.4.4 key-att-to) whatever the
// instance order (§3.8.4.1.3 all-mg), a repeated particle's children
// appended in instance order. CheckDecode returns nil for each row's T.
func TestDecodeAllOccurrence(t *testing.T) {
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"one required particle", allOccASchema, "<s:a>x</s:a>", into[allOccA](), allOccA{All: &allOccAItem{A: "x"}}},
		{"no children", allOccASchema, "", into[allOccA](), allOccA{}},
		{"an optional <group ref>", allOccRefSchema, "<s:a>x</s:a>", into[allOccA](), allOccA{All: &allOccAItem{A: "x"}}},
		{"an optional <group ref>, no children", allOccRefSchema, "", into[allOccA](), allOccA{}},
		{"two particles in particle order", allOccABSchema, "<s:a>x</s:a><s:b>1</s:b>", into[allOccAB](), allOccAB{All: &allOccABItem{A: "x", B: 1}}},
		{"two particles permuted", allOccABSchema, "<s:b>1</s:b><s:a>x</s:a>", into[allOccAB](), allOccAB{All: &allOccABItem{A: "x", B: 1}}},
		{"a repeated particle interleaved with another's child", allOccRepSchema, "<s:a>p</s:a><s:b>3</s:b><s:a>q</s:a>", into[allOccRep](), allOccRep{All: &allOccRepItem{A: []string{"p", "q"}, B: 3}}},
		{"a nested all's particle first", allOccNestedSchema, "<s:b>4</s:b><s:a>x</s:a>", into[allOccAB](), allOccAB{All: &allOccABItem{A: "x", B: 4}}},
		{"an extension merging two alls", allOccExtSchema, "<s:a>x</s:a><s:b>6</s:b>", into[allOccAB](), allOccAB{All: &allOccABItem{A: "x", B: 6}}},
		{"an extension merging two alls, permuted", allOccExtSchema, "<s:b>6</s:b><s:a>x</s:a>", into[allOccAB](), allOccAB{All: &allOccABItem{A: "x", B: 6}}},
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

// TestCheckDecodeAllOccurrenceValue pins that a T whose occurrence field
// holds the struct for one occurrence as a value, where codegen emits a
// pointer to it for an all group with {min occurs} 0 (allOccASchema), is
// ErrTypeMismatch alone, no decline, naming the field and the all group.
func TestCheckDecodeAllOccurrenceValue(t *testing.T) {
	type valued struct{ All allOccAItem }
	err := CheckDecode[valued](schemaOf(t, allOccASchema), rootName)
	const want = "field 0 of codec.valued is All codec.allOccAItem where codegen emits All, *I for the type I of one occurrence of the all"
	if err == nil || err.Error() != ErrTypeMismatch.Error()+": "+want {
		t.Errorf("CheckDecode = %v, want %q", err, want)
	}
	var de *DeclineError
	if !errors.Is(err, ErrTypeMismatch) || errors.As(err, &de) {
		t.Errorf("error %v: want ErrTypeMismatch alone", err)
	}
}
