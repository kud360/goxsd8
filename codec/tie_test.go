package codec

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// The tie schemas, each declaring s:r. tieGroupSchema's r is a sequence of an
// <element ref> to C, then one to h, an abstract head whose members are c and
// C, each of xs:string, in that order: the walk reaches C's declaration first,
// so codegen names C's type C and c's C2, while the branches follow group
// order, HSubstitutionGroupC for c and HSubstitutionGroupC2 for C, so the
// member's field type orders the tie against its branches. tieChoiceSchema's
// r is a choice of a_a, a.a and a-a, each of xs:string, beside a top-level
// rAA that claims RAA first, so codegen names the branches RAA2, RAA3 and
// RAA4, never RAA.
const (
	tieGroupSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:C"/><xs:element ref="s:h"/></xs:sequence></xs:complexType></xs:element>` +
		`<xs:element name="h" abstract="true" type="xs:string"/><xs:element name="c" type="xs:string" substitutionGroup="s:h"/><xs:element name="C" type="xs:string" substitutionGroup="s:h"/>`
	tieChoiceSchema = `<xs:element name="rAA" type="xs:string"/><xs:element name="r"><xs:complexType><xs:choice>` +
		`<xs:element name="a_a" type="xs:string"/><xs:element name="a.a" type="xs:string"/><xs:element name="a-a" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
)

// tieGR to TieHC2 are tieGroupSchema's R, C, HSubstitutionGroup,
// HSubstitutionGroupC, C2 and HSubstitutionGroupC2, and tieR to TieRAA4
// tieChoiceSchema's R, RAA2, RAA3 and RAA4, which TestTieTypesAreCodegens
// pins. Each branch carries the name codegen gives it over its interface's,
// since Decode places a tie's branches by name (item 6 of Decode's doc).
type (
	tieGR struct {
		C tieC
		H tieH
	}
	tieC   string
	tieH   interface{ isTieH() }
	TieHC  struct{ C tieC2 }
	tieC2  string
	TieHC2 struct{ C tieC }

	tieR    interface{ isTieR() }
	TieRAA2 struct{ AA string }
	TieRAA3 struct{ AA string }
	TieRAA4 struct{ AA string }
)

func (TieHC) isTieH()   {}
func (TieHC2) isTieH()  {}
func (TieRAA2) isTieR() {}
func (TieRAA3) isTieR() {}
func (TieRAA4) isTieR() {}

// Alpha and Beta are struct{ AA string } implementers of tieR that codegen
// names no branch of it, and tieInt one of a shape no alternative of it
// admits.
type (
	Alpha  struct{ AA string }
	Beta   struct{ AA string }
	tieInt struct{ AA int32 }
)

func (Alpha) isTieR()  {}
func (Beta) isTieR()   {}
func (tieInt) isTieR() {}

// The tie schemas' branch sets, each supplied against alternative order.
var (
	tieGroupBranches  = WithBranches[tieH](TieHC2{}, TieHC{})
	tieChoiceBranches = WithBranches[tieR](TieRAA4{}, TieRAA2{}, TieRAA3{})
)

// TestTieTypesAreCodegens pins that the tie schemas' types are codegen's.
func TestTieTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls string
		want  []string
	}{
		{tieGroupSchema, []string{
			"type R struct {\n\tC C\n\tH HSubstitutionGroup\n}\n",
			"type HSubstitutionGroup interface {\n\tisHSubstitutionGroup()\n}\n",
			"type HSubstitutionGroupC struct {\n\tC C2\n}\n\nfunc (HSubstitutionGroupC) isHSubstitutionGroup() {}\n",
			"type HSubstitutionGroupC2 struct {\n\tC C\n}\n\nfunc (HSubstitutionGroupC2) isHSubstitutionGroup() {}\n",
			"type C2 string\n",
			"type C string\n",
		}},
		{tieChoiceSchema, []string{
			"type RAA string\n",
			"type R interface {\n\tisR()\n}\n",
			"type RAA2 struct {\n\tAA string\n}\n\nfunc (RAA2) isR() {}\n",
			"type RAA3 struct {\n\tAA string\n}\n\nfunc (RAA3) isR() {}\n",
			"type RAA4 struct {\n\tAA string\n}\n\nfunc (RAA4) isR() {}\n",
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

// TestDecodeTie pins item 6's tie: the branches of substitution group
// members, and of choice alternatives, whose local names render one Go
// identifier, each set supplied against alternative order, go to their
// alternatives by name, so each child decodes into its own declaration's
// branch; CheckDecode returns nil for each row's T.
func TestDecodeTie(t *testing.T) {
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"a member c, its type named C2", tieGroupSchema, "<s:C>x</s:C><s:c>v</s:c>", into[tieGR](tieGroupBranches), tieGR{C: "x", H: TieHC{C: "v"}}},
		{"a member C, its type named C", tieGroupSchema, "<s:C>x</s:C><s:C>v</s:C>", into[tieGR](tieGroupBranches), tieGR{C: "x", H: TieHC2{C: "v"}}},
		{"a choice's alternative a_a", tieChoiceSchema, "<s:a_a>v</s:a_a>", into[tieR](tieChoiceBranches), tieR(TieRAA2{AA: "v"})},
		{"a choice's alternative a.a", tieChoiceSchema, "<s:a.a>v</s:a.a>", into[tieR](tieChoiceBranches), tieR(TieRAA3{AA: "v"})},
		{"a choice's alternative a-a", tieChoiceSchema, "<s:a-a>v</s:a-a>", into[tieR](tieChoiceBranches), tieR(TieRAA4{AA: "v"})},
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

// TestDecodeTieMismatch pins item 6's tie for a branch set it cannot place:
// branches of the tie's shape not named from their interface's name, or
// fewer of the tie's shape than the tie has alternatives, are ErrTypeMismatch, Decode
// returning no value and CheckDecode Decode's error, never a decode into any
// branch, though each branch's shape is one an alternative admits.
func TestDecodeTieMismatch(t *testing.T) {
	const prefix = "codec: T is not the type codegen emits for the document element: "
	rows := []struct {
		name   string
		branch Option
		want   string
	}{
		{"branches not named from the interface's name", WithBranches[tieR](Alpha{}, Beta{}, TieRAA4{}),
			prefix + "branch codec.Alpha of codec.tieR is named neither TieRAA nor TieRAA followed by a decimal from 2, as codegen names the branch structs of its 3 alternatives whose local names render one identifier"},
		{"two of three branches named from the interface's name", WithBranches[tieR](TieRAA2{}, TieRAA3{}, Alpha{}),
			prefix + "branch codec.Alpha of codec.tieR is named neither TieRAA nor TieRAA followed by a decimal from 2, as codegen names the branch structs of its 3 alternatives whose local names render one identifier"},
		{"a branch of no alternative's shape", WithBranches[tieR](TieRAA2{}, TieRAA3{}, tieInt{}),
			prefix + "WithBranches supplies 2 branches of codec.tieR of the shape of the 3 alternatives codegen names TieRAA and TieRAA followed by a decimal from 2"},
	}
	schema := schemaOf(t, tieChoiceSchema)
	data := instanceOf("", "<s:a_a>v</s:a_a>")
	mustAccept(t, schema, data)
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			got, err := Decode[tieR](schema, strict.New(), data, row.branch)
			if got != nil {
				t.Fatalf("Decode = %#v, want no value", *got)
			}
			if err == nil || !errors.Is(err, ErrTypeMismatch) || err.Error() != row.want {
				t.Fatalf("Decode error = %v, want %q", err, row.want)
			}
			assertProbeMismatch(t, CheckDecode[tieR](schema, rootName, row.branch), err)
		})
	}
}

// TestDecodeTieDeclines pins the shared-branch declines a tie leaves: two
// basic particles of one flattened reach whose local names render one
// identifier, which no name places, and two element alternatives of one
// rendered identifier that a wildcard alternative's word any renders too,
// which is no tie; each declines constructSharedBranch from Decode and
// CheckDecode alike. Two wildcard alternatives are
// TestDecodeChoiceGroupDeclines', and two particles of one expanded name in a
// flattened reach TestDecodeFlattenSharedBranch's.
func TestDecodeTieDeclines(t *testing.T) {
	rows := []struct {
		name, decls, body string
	}{
		{"a flattened reach of c and C", root(`<xs:choice maxOccurs="unbounded"><xs:element name="c" type="xs:string" maxOccurs="unbounded"/><xs:element name="C" type="xs:string"/></xs:choice>`),
			"<s:c>v</s:c><s:C>w</s:C>"},
		{"alternatives any and Any beside a wildcard", root(`<xs:choice><xs:element name="any" type="xs:string"/><xs:element name="Any" type="xs:string"/><xs:any namespace="##other" processContents="skip"/></xs:choice>`),
			"<s:any>v</s:any>"},
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
			assertDecline(t, err, constructSharedBranch, 1, 1)
			assertProbeDecline(t, CheckDecode[struct{}](schema, rootName), err)
		})
	}
}

// TestCheckDecodeNestedChoicesShare pins two conjuncts of choicesShare
// (#3692): r's choice holds two spliced sequences, each holding a nested
// choice's {Choice I}, the first's of element any, of type xs:anyType, and a
// wildcard, whose branches share no shape. Beside one of Any alone, of
// codegen's struct, the counts differ; beside one of Any and z, an xs:string,
// z shares no branch with either of the first's. Neither pair declines, so
// Decode and CheckDecode into struct{} both return item 6's ErrTypeMismatch.
func TestCheckDecodeNestedChoicesShare(t *testing.T) {
	const first = `<xs:sequence><xs:choice><xs:element name="any"/><xs:any namespace="##other" processContents="skip"/></xs:choice></xs:sequence>`
	rows := []struct{ name, second string }{
		{"differing alternative counts", `<xs:sequence><xs:choice>` + vStruct("Any", "") + `</xs:choice></xs:sequence>`},
		{"the second's z sharing no branch with the first's", `<xs:sequence><xs:choice>` + vStruct("Any", "") + `<xs:element name="z" type="xs:string"/></xs:choice></xs:sequence>`},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, root(`<xs:sequence><xs:element name="h" type="xs:string"/><xs:choice>`+first+row.second+`</xs:choice></xs:sequence>`))
			data := instanceOf("", "<s:h>x</s:h><s:any/>")
			mustAccept(t, schema, data)
			_, err := Decode[struct{}](schema, strict.New(), data)
			var de *DeclineError
			if errors.As(err, &de) || !errors.Is(err, ErrTypeMismatch) {
				t.Fatalf("Decode = %v, want ErrTypeMismatch and no decline", err)
			}
			assertProbeMismatch(t, CheckDecode[struct{}](schema, rootName), err)
		})
	}
}
