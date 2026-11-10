package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// The nested-choice schemas, each declaring s:r, whose content sequence
// holds a choice particle occurring exactly once that codegen neither
// splices nor flattens (codegen/doc.go "# Particles"), so codegen gives it a
// field of its bare sealed interface named as if from local name choice:
// ncInlineSchema's is sequence(a, choice(b, c)); ncRefSchema's the same
// through a <group ref> to the choice group g; ncComplexSchema's
// sequence(a, choice(b, c)) with b of an anonymous complex type, vStruct's;
// ncSplicedSchema's sequence(a, sequence(choice(b, c))), the inner sequence
// spliced; ncExtSchema's r is of t2, an extension of t, choice(a), adding
// sequence(b), so t2's {particle} is sequence(choice(a), sequence(b))
// (§3.4.2.3.3 dcl.ctd.ctcc.common clause 4.2.3.3); ncTwoSchema's
// sequence(choice(a, b), choice(c repeated, d)), whose second field is
// Choice2; ncWildSchema's sequence(any over urn:o, choice(b, any over
// urn:p)), a sequence wildcard and an alternative's; and ncEmptiableSchema's
// sequence(a, choice(b optional, c)), an emptiable choice.
const (
	ncInlineSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/>
  <xs:choice><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:int"/></xs:choice></xs:sequence></xs:complexType></xs:element>`
	ncRefSchema = `<xs:group name="g"><xs:choice><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:int"/></xs:choice></xs:group>
  <xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:group ref="s:g"/></xs:sequence></xs:complexType></xs:element>`
	ncComplexSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/>
  <xs:choice>` + `<xs:element name="b"><xs:complexType><xs:sequence><xs:element name="v" type="xs:string"/></xs:sequence></xs:complexType></xs:element>` + `<xs:element name="c" type="xs:int"/></xs:choice></xs:sequence></xs:complexType></xs:element>`
	ncSplicedSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/>
  <xs:sequence><xs:choice><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:int"/></xs:choice></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	ncExtSchema = `<xs:complexType name="t"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice></xs:complexType>
  <xs:complexType name="t2"><xs:complexContent><xs:extension base="s:t"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="r" type="s:t2"/>`
	ncTwoSchema = `<xs:element name="r"><xs:complexType><xs:sequence>
  <xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:choice>
  <xs:choice><xs:element name="c" type="xs:string" maxOccurs="2"/><xs:element name="d" type="xs:int"/></xs:choice></xs:sequence></xs:complexType></xs:element>`
	ncWildSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:any namespace="urn:o" processContents="lax"/>
  <xs:choice><xs:element name="b" type="xs:string"/><xs:any namespace="urn:p" processContents="lax"/></xs:choice></xs:sequence></xs:complexType></xs:element>`
	ncEmptiableSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/>
  <xs:choice><xs:element name="b" type="xs:string" minOccurs="0"/><xs:element name="c" type="xs:int"/></xs:choice></xs:sequence></xs:complexType></xs:element>`
)

// The types codegen emits for the nested-choice schemas' s:r, in shape,
// which TestNestedChoiceTypesAreCodegens pins: nc and its branches
// ncInlineSchema's, ncRefSchema's, ncSplicedSchema's and
// ncEmptiableSchema's R, RChoice, RChoiceB and RChoiceC; ncx
// ncComplexSchema's, ncxV its RB; ncE ncExtSchema's; ncT ncTwoSchema's; and
// ncW ncWildSchema's.
type (
	nc struct {
		A      string
		Choice ncChoice
	}
	ncChoice interface{ isNcChoice() }
	ncB      struct{ B string }
	ncC      struct{ C int32 }

	ncx struct {
		A      string
		Choice ncxChoice
	}
	ncxChoice interface{ isNcxChoice() }
	ncxB      struct{ B ncxV }
	ncxV      struct{ V string }
	ncxC      struct{ C int32 }

	ncE struct {
		Choice ncEChoice
		B      string
	}
	ncEChoice interface{ isNcEChoice() }
	ncEA      struct{ A string }

	ncT struct {
		Choice  ncTChoice
		Choice2 ncTChoice2
	}
	ncTChoice  interface{ isNcTChoice() }
	ncTA       struct{ A string }
	ncTB       struct{ B int32 }
	ncTChoice2 interface{ isNcTChoice2() }
	ncT2C      struct{ C []string }
	ncT2D      struct{ D int32 }

	ncW struct {
		Any    anyElement
		Choice ncWChoice
	}
	ncWChoice interface{ isNcWChoice() }
	ncWB      struct{ B string }
	ncWAny    struct{ Any anyElement }
)

func (ncB) isNcChoice()     {}
func (ncC) isNcChoice()     {}
func (ncxB) isNcxChoice()   {}
func (ncxC) isNcxChoice()   {}
func (ncEA) isNcEChoice()   {}
func (ncTA) isNcTChoice()   {}
func (ncTB) isNcTChoice()   {}
func (ncT2C) isNcTChoice2() {}
func (ncT2D) isNcTChoice2() {}
func (ncWB) isNcWChoice()   {}
func (ncWAny) isNcWChoice() {}

// The branch sets of the nested-choice interfaces.
var (
	ncBranches  = WithBranches[ncChoice](ncB{}, ncC{})
	ncxBranches = WithBranches[ncxChoice](ncxB{}, ncxC{})
	ncEBranches = WithBranches[ncEChoice](ncEA{})
	ncTBranches = []Option{WithBranches[ncTChoice](ncTA{}, ncTB{}), WithBranches[ncTChoice2](ncT2C{}, ncT2D{})}
	ncWBranches = WithBranches[ncWChoice](ncWB{}, ncWAny{})
)

// TestNestedChoiceTypesAreCodegens pins the types above to those codegen
// emits for each nested-choice schema's s:r, in shape (codegen/doc.go
// "# Particles"), so the rows decoding into them decode into codegen's
// types.
func TestNestedChoiceTypesAreCodegens(t *testing.T) {
	ncDecls := []string{
		"type R struct {\n\tA      string\n\tChoice RChoice\n}\n",
		"type RChoice interface {\n\tisRChoice()\n}\n",
		"type RChoiceB struct {\n\tB string\n}\n\nfunc (RChoiceB) isRChoice() {}\n",
		"type RChoiceC struct {\n\tC int32\n}\n\nfunc (RChoiceC) isRChoice() {}\n",
	}
	rows := []struct {
		decls string
		want  []string
	}{
		{ncInlineSchema, ncDecls},
		{ncRefSchema, ncDecls},
		{ncSplicedSchema, ncDecls},
		{ncEmptiableSchema, ncDecls},
		{ncComplexSchema, []string{
			"type R struct {\n\tA      string\n\tChoice RChoice\n}\n",
			"type RChoice interface {\n\tisRChoice()\n}\n",
			"type RChoiceB struct {\n\tB RB\n}\n\nfunc (RChoiceB) isRChoice() {}\n",
			"type RB struct {\n\tV string\n}\n",
			"type RChoiceC struct {\n\tC int32\n}\n\nfunc (RChoiceC) isRChoice() {}\n",
		}},
		{ncExtSchema, []string{
			"type R struct {\n\tChoice T2Choice\n\tB      string\n}\n",
			"type T2Choice interface {\n\tisT2Choice()\n}\n",
			"type T2ChoiceA struct {\n\tA string\n}\n\nfunc (T2ChoiceA) isT2Choice() {}\n",
		}},
		{ncTwoSchema, []string{
			"type R struct {\n\tChoice  RChoice\n\tChoice2 RChoice2\n}\n",
			"type RChoiceA struct {\n\tA string\n}\n\nfunc (RChoiceA) isRChoice() {}\n",
			"type RChoiceB struct {\n\tB int32\n}\n\nfunc (RChoiceB) isRChoice() {}\n",
			"type RChoice2 interface {\n\tisRChoice2()\n}\n",
			"type RChoice2C struct {\n\tC []string\n}\n\nfunc (RChoice2C) isRChoice2() {}\n",
			"type RChoice2D struct {\n\tD int32\n}\n\nfunc (RChoice2D) isRChoice2() {}\n",
		}},
		{ncWildSchema, []string{
			"type R struct {\n\tAny    AnyElement\n\tChoice RChoice\n}\n",
			"type RChoiceB struct {\n\tB string\n}\n\nfunc (RChoiceB) isRChoice() {}\n",
			"type RChoiceAny struct {\n\tAny AnyElement\n}\n\nfunc (RChoiceAny) isRChoice() {}\n",
		}},
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

// TestDecodeNestedChoice pins the nested-choice lift (cvc-model-group: one
// alternative matches): a choice particle of a content sequence, occurring
// exactly once, inline, through a <group ref>, inside a spliced sequence, or
// an extension's base particle, decodes into its field at its place, the
// branch of the alternative its child matches, an alternative of complex
// type into its struct; two such choices fill Choice and Choice2, a repeated
// alternative appending into its one branch; a sequence's wildcard and an
// alternative's each fill their own; and an emptiable choice's empty match
// is the nil interface. CheckDecode returns nil for each row's T.
func TestDecodeNestedChoice(t *testing.T) {
	w := func(space string) anyElement { return anyElement{Space: space, Local: "w"} }
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"inline, its first alternative", ncInlineSchema, "<s:a>x</s:a><s:b>y</s:b>", into[nc](ncBranches), nc{A: "x", Choice: ncB{B: "y"}}},
		{"inline, its second alternative", ncInlineSchema, "<s:a>x</s:a><s:c>1</s:c>", into[nc](ncBranches), nc{A: "x", Choice: ncC{C: 1}}},
		{"through a <group ref>", ncRefSchema, "<s:a>x</s:a><s:c>2</s:c>", into[nc](ncBranches), nc{A: "x", Choice: ncC{C: 2}}},
		{"an alternative of complex type", ncComplexSchema, "<s:a>x</s:a><s:b><s:v>z</s:v></s:b>", into[ncx](ncxBranches), ncx{A: "x", Choice: ncxB{B: ncxV{V: "z"}}}},
		{"inside a spliced sequence", ncSplicedSchema, "<s:a>x</s:a><s:b>y</s:b>", into[nc](ncBranches), nc{A: "x", Choice: ncB{B: "y"}}},
		{"an extension of a choice adding a sequence", ncExtSchema, "<s:a>x</s:a><s:b>y</s:b>", into[ncE](ncEBranches), ncE{Choice: ncEA{A: "x"}, B: "y"}},
		{"two nested choices", ncTwoSchema, "<s:b>3</s:b><s:c>p</s:c><s:c>q</s:c>", into[ncT](ncTBranches...), ncT{Choice: ncTB{B: 3}, Choice2: ncT2C{C: []string{"p", "q"}}}},
		{"a sequence's wildcard and an alternative's", ncWildSchema, `<o:w xmlns:o="urn:o"/><p:w xmlns:p="urn:p"/>`, into[ncW](ncWBranches),
			ncW{Any: w("urn:o"), Choice: ncWAny{Any: w("urn:p")}}},
		{"a sequence's wildcard and an element alternative", ncWildSchema, `<o:w xmlns:o="urn:o"/><s:b>y</s:b>`, into[ncW](ncWBranches),
			ncW{Any: w("urn:o"), Choice: ncWB{B: "y"}}},
		{"an emptiable choice, empty match", ncEmptiableSchema, "<s:a>x</s:a>", into[nc](ncBranches), nc{A: "x"}},
		{"an emptiable choice, its optional alternative present", ncEmptiableSchema, "<s:a>x</s:a><s:b>y</s:b>", into[nc](ncBranches), nc{A: "x", Choice: ncB{B: "y"}}},
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

// TestDecodeNestedChoiceDeclines pins the nested choice's declines, each
// charged to the document element at 1:1: an alternative's expanded name an
// earlier particle of the content sequence takes, and a later particle's an
// alternative takes, which decline as shared names, the sequence and the
// choice indexing one children map; and two wildcard alternatives sharing a
// branch shape after a sequence wildcard, which declines naming the second
// alternative wildcard particle 3 of the type, its ordinal counted across
// the sequence's wildcard and the choice's in one wildcard slot.
func TestDecodeNestedChoiceDeclines(t *testing.T) {
	const wildPrefix = "?:1:1: codec does not decode two branches of a sealed interface whose structs have one shape: wildcard particle 3 of the {type definition} of element {urn:s}r declared at"
	rows := []struct {
		name, decls, body string
		construct         construct
		prefix            string
	}{
		{"an alternative sharing an earlier particle's name", sequenceOf(`<xs:element name="a" type="xs:string"/><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice>`),
			"<s:a>x</s:a><s:b>y</s:b>", constructSharedName, ""},
		{"a later particle sharing an alternative's name", sequenceOf(`<xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice><xs:element name="a" type="xs:string"/>`),
			"<s:b>y</s:b><s:a>x</s:a>", constructSharedName, ""},
		{"two wildcard alternatives sharing a branch shape after a sequence wildcard", sequenceOf(`<xs:any namespace="urn:o" processContents="lax"/><xs:choice><xs:any namespace="urn:p" processContents="lax"/><xs:any namespace="urn:q" processContents="lax"/></xs:choice>`),
			`<o:w xmlns:o="urn:o"/><p:w xmlns:p="urn:p"/>`, constructSharedBranch, wildPrefix},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf("", row.body)
			mustAccept(t, schema, data)
			got, err := Decode[struct{}](schema, strict.New(), data)
			if got != nil {
				t.Fatalf("Decode = %v, want no value", *got)
			}
			assertDecline(t, err, row.construct, 1, 1)
			if !strings.HasPrefix(err.Error(), row.prefix) {
				t.Errorf("Decode error = %v, want the prefix %q", err, row.prefix)
			}
		})
	}
}

// TestDecodeNestedChoiceTypeMismatch pins item 6 for a nested choice's
// field: a T whose field at the choice's place is not named Choice is
// ErrTypeMismatch naming that field by its index, CheckDecode's error
// Decode's.
func TestDecodeNestedChoiceTypeMismatch(t *testing.T) {
	type misnamed struct {
		A string
		C ncChoice
	}
	schema := schemaOf(t, ncInlineSchema)
	data := instanceOf("", "<s:a>x</s:a><s:b>y</s:b>")
	mustAccept(t, schema, data)
	got, err := Decode[misnamed](schema, strict.New(), data, ncBranches)
	if got != nil {
		t.Fatalf("Decode = %v, want no value", *got)
	}
	const want = "codec: T is not the type codegen emits for the document element: field 1 of codec.misnamed is C codec.ncChoice where codegen emits Choice, a sealed interface"
	if err == nil || !errors.Is(err, ErrTypeMismatch) || err.Error() != want {
		t.Errorf("Decode error = %v, want %q", err, want)
	}
	assertProbeMismatch(t, CheckDecode[misnamed](schema, rootName, ncBranches), err)
}
