package codec

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// The encode choice schemas, each declaring s:r, whose content {particle} is
// a choice occurring once that codegen does not flatten, beside
// decode_test.go's pickSchema (the type itself, an alternative of {max
// occurs} 3), optSchema (emptiable, a struct holding it) and
// attributedSchema (a struct holding it beside attribute fields):
// ecRefSchema's choice(a, b) through a <group ref>, the type itself;
// ecEmptySchema's choice of no {particles}, the type itself, which no branch
// implements; ecEmptyAttrSchema's the same beside an attribute, a struct
// holding it; and ecSliceSchema's choice(a, b with {min occurs} 0 and {max
// occurs} 2), emptiable, a struct holding it, whose b branch holds a []V at
// {min occurs} 1 (goshape.BranchOccurs).
const (
	ecRefSchema = `<xs:group name="g"><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice></xs:group>` +
		`<xs:element name="r"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`
	ecEmptySchema     = `<xs:element name="r"><xs:complexType><xs:choice/></xs:complexType></xs:element>`
	ecEmptyAttrSchema = `<xs:element name="r"><xs:complexType><xs:choice/><xs:attribute name="tag" type="xs:string"/></xs:complexType></xs:element>`
	ecSliceSchema     = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string" minOccurs="0" maxOccurs="2"/></xs:choice></xs:complexType></xs:element>`
)

// The types codegen emits for the encode choice schemas' s:r, in shape, which
// TestEncodeChoiceTypesAreCodegens pins: ecRef, ecRefA and ecRefB for
// ecRefSchema; ecEmpty for ecEmptySchema; ecEmptyAttr and ecEmptyAttrChoice
// for ecEmptyAttrSchema; ecSlice, ecSliceChoice, ecSliceA and ecSliceB for
// ecSliceSchema.
type (
	ecRef  interface{ isEcRef() }
	ecRefA struct{ A string }
	ecRefB struct{ B string }

	ecEmpty interface{ isEcEmpty() }

	ecEmptyAttr struct {
		Choice ecEmptyAttrChoice
		Tag    *string
	}
	ecEmptyAttrChoice interface{ isEcEmptyAttrChoice() }

	ecSlice       struct{ Choice ecSliceChoice }
	ecSliceChoice interface{ isEcSliceChoice() }
	ecSliceA      struct{ A string }
	ecSliceB      struct{ B []string }
)

func (ecRefA) isEcRef()                  {}
func (ecRefB) isEcRef()                  {}
func (ecSliceA) isEcSliceChoice()        {}
func (ecSliceB) isEcSliceChoice()        {}
func (ecEmptyNone) isEcEmpty()           {}
func (ecEmptyNone) isEcEmptyAttrChoice() {}

// ecEmptyNone implements ecEmpty and ecEmptyAttrChoice, whose choice has no
// alternative, so codegen emits no branch of either: no document Decode reads
// holds it.
type ecEmptyNone struct{}

// The branch sets of the encode choice schemas' interfaces; ecEmpty's and
// ecEmptyAttrChoice's are empty.
var (
	ecRefBranches   = WithBranches[ecRef](ecRefA{}, ecRefB{})
	ecSliceBranches = WithBranches[ecSliceChoice](ecSliceA{}, ecSliceB{})
)

// TestEncodeChoiceTypesAreCodegens pins the types above to those codegen
// emits for each encode choice schema, in shape.
func TestEncodeChoiceTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls string
		want  []string
	}{
		{ecRefSchema, []string{
			"type R interface {\n\tisR()\n}\n",
			"type RA struct {\n\tA string\n}\n\nfunc (RA) isR() {}\n",
			"type RB struct {\n\tB string\n}\n\nfunc (RB) isR() {}\n",
		}},
		{ecEmptySchema, []string{"type R interface {\n\tisR()\n}\n"}},
		{ecEmptyAttrSchema, []string{
			"type R struct {\n\tChoice RChoice\n\tTag    *string\n}\n",
			"type RChoice interface {\n\tisRChoice()\n}\n",
		}},
		{ecSliceSchema, []string{
			"type R struct {\n\tChoice RChoice\n}\n",
			"type RChoiceA struct {\n\tA string\n}\n\nfunc (RChoiceA) isRChoice() {}\n",
			"type RChoiceB struct {\n\tB []string\n}\n\nfunc (RChoiceB) isRChoice() {}\n",
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

// TestAppendChoice pins the content choice occurring once that "# What
// encodes" admits, in each shape codegen gives it: T the sealed interface
// itself, inline or through a <group ref>, and a struct whose first field
// holds it, beside attribute fields or for an emptiable choice. Each value is
// written as the branch's alternative (cvc-model-group, §3.8.4.1.2), the
// branch placed by the WithBranches set Append is passed, validate accepts
// the document, and Decode under that set returns the value; the emptiable
// choice's nil interface, the empty match (§3.9.6.3 cos-group-emptiable),
// writes no child element. Each row fails with contentPlanOf's OnceChoice
// case declining as it did before, and the nil row of opt with the nil
// interface charged as item 6 wherever it is met.
func TestAppendChoice(t *testing.T) {
	str := func(s string) *string { return &s }
	t.Run("the type itself, an element alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, pickSchema), sname("r"), pick(pickA{A: " x "}), `<r xmlns="urn:s"><a> x </a></r>`, deepEqual, pickBranches)
	})
	t.Run("the type itself, a repeated alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, pickSchema), sname("r"), pick(pickB{B: []string{"x", "y"}}), `<r xmlns="urn:s"><b>x</b><b>y</b></r>`, deepEqual, pickBranches)
	})
	t.Run("the type itself through a <group ref>", func(t *testing.T) {
		roundTrip(t, schemaOf(t, ecRefSchema), sname("r"), ecRef(ecRefB{B: "y"}), `<r xmlns="urn:s"><b>y</b></r>`, deepEqual, ecRefBranches)
	})
	t.Run("a struct beside attribute fields", func(t *testing.T) {
		v := attributed{Choice: attributedB{B: "x"}, Choice2: str("c"), Tag: "t"}
		roundTrip(t, schemaOf(t, attributedSchema), sname("r"), v, `<r xmlns="urn:s" choice="c" tag="t"><b>x</b></r>`, deepEqual, attributedBranches)
	})
	t.Run("an emptiable choice's {min occurs} 0 alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, optSchema), sname("r"), opt{Choice: optB{B: "x"}}, `<r xmlns="urn:s"><b>x</b></r>`, deepEqual, optBranches)
	})
	t.Run("an emptiable choice's empty match", func(t *testing.T) {
		roundTrip(t, schemaOf(t, optSchema), sname("r"), opt{}, `<r xmlns="urn:s"/>`, deepEqual, optBranches)
	})
	t.Run("an emptiable choice's repeated alternative", func(t *testing.T) {
		roundTrip(t, schemaOf(t, ecSliceSchema), sname("r"), ecSlice{Choice: ecSliceB{B: []string{"x", "y"}}}, `<r xmlns="urn:s"><b>x</b><b>y</b></r>`, deepEqual, ecSliceBranches)
	})
}

// invalidChoice fails t unless err is item 6's error of Append's doc, with
// no xsderr rule, whose text opens with prefix and holds each of want.
func invalidChoice(t *testing.T, err error, prefix string, want ...string) {
	t.Helper()
	invalidValue(t, err, want...)
	noRule(t, err)
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Errorf("error %q does not open with %q", err, prefix)
	}
}

// TestAppendChoiceInvalid pins item 6 of Append's doc for a content choice
// occurring once: the nil interface where no alternative is emptiable, T
// itself or a struct's first field, a choice of no {particles} among them,
// whose language is empty (§3.8.4.3's note); a value of a dynamic type the
// WithBranches set placed at no alternative, a pointer to a branch among
// them, which implements the interface since codegen's marker method has a
// value receiver; and a branch slice whose length lies outside its
// alternative's occurrence, {min occurs} 0 read as 1. None carries an xsderr
// rule, and an attribute's error comes before the content's.
func TestAppendChoiceInvalid(t *testing.T) {
	pickS, attributedS := schemaOf(t, pickSchema), schemaOf(t, attributedSchema)
	t.Run("nil, the type itself", func(t *testing.T) {
		_, err := appendChecked(t, nil, pickS, strict.New(), sname("r"), pick(nil), pickBranches)
		invalidChoice(t, err, "codec: Append: pick is nil, the empty match, which the content choice of the {type definition} of element {urn:s}r declared at ", "does not accept: none of its 2 alternatives is emptiable", "cos-group-emptiable")
	})
	t.Run("nil, a struct's first field", func(t *testing.T) {
		_, err := appendChecked(t, nil, attributedS, strict.New(), sname("r"), attributed{Tag: "t"}, attributedBranches)
		invalidChoice(t, err, "codec: Append: attributed.Choice is nil, the empty match, which the content choice of the {type definition} of element {urn:s}r declared at ", "does not accept: none of its 2 alternatives is emptiable")
	})
	t.Run("nil, a choice of no {particles}", func(t *testing.T) {
		_, err := appendChecked(t, nil, schemaOf(t, ecEmptySchema), strict.New(), sname("r"), ecEmpty(nil))
		invalidChoice(t, err, "codec: Append: ecEmpty is nil, the empty match, which the content choice of the {type definition} of element {urn:s}r declared at ", "does not accept: the choice has no alternative, so that it accepts no sequence (§3.8.4.3 cvc-model-group)")
	})
	t.Run("nil, a struct's choice of no {particles}", func(t *testing.T) {
		_, err := appendChecked(t, nil, schemaOf(t, ecEmptyAttrSchema), strict.New(), sname("r"), ecEmptyAttr{})
		invalidChoice(t, err, "codec: Append: ecEmptyAttr.Choice is nil, the empty match, ", "the choice has no alternative")
	})
	t.Run("an implementer of a choice of no {particles}", func(t *testing.T) {
		_, err := appendChecked(t, nil, schemaOf(t, ecEmptySchema), strict.New(), sname("r"), ecEmpty(ecEmptyNone{}))
		invalidChoice(t, err, "codec: Append: ecEmpty holds a codec.ecEmptyNone, which the WithBranches set of codec.ecEmpty places at no alternative of the content choice of ")
	})
	t.Run("a pointer to a branch", func(t *testing.T) {
		_, err := appendChecked(t, nil, pickS, strict.New(), sname("r"), pick(&pickA{A: "x"}), pickBranches)
		invalidChoice(t, err, "codec: Append: pick holds a *codec.pickA, which the WithBranches set of codec.pick places at no alternative of the content choice of ", "so that Decode returns no such value")
	})
	t.Run("an implementer the set does not hold", func(t *testing.T) {
		_, err := appendChecked(t, nil, pickS, strict.New(), sname("r"), pick(pickC{C: "x"}), pickBranches)
		invalidChoice(t, err, "codec: Append: pick holds a codec.pickC, which the WithBranches set of codec.pick places at no alternative")
	})
	t.Run("a branch slice above {max occurs}", func(t *testing.T) {
		_, err := appendChecked(t, nil, pickS, strict.New(), sname("r"), pick(pickB{B: []string{"1", "2", "3", "4"}}), pickBranches)
		invalidChoice(t, err, "codec: Append: pick.(pickB).B holds 4 items, outside the occurrence 1..3 ")
	})
	t.Run("an empty branch slice of a {min occurs} 0 alternative", func(t *testing.T) {
		_, err := appendChecked(t, nil, schemaOf(t, ecSliceSchema), strict.New(), sname("r"), ecSlice{Choice: ecSliceB{B: []string{}}}, ecSliceBranches)
		invalidChoice(t, err, "codec: Append: ecSlice.Choice.(ecSliceB).B holds 0 items, outside the occurrence 1..2, the occurrence 0..2 with {min occurs} 0 read as 1 in its branch, of the particle of element {urn:s}b declared at ")
	})
	t.Run("an attribute before the choice", func(t *testing.T) {
		_, err := appendChecked(t, nil, attributedS, strict.New(), sname("r"), attributed{Tag: "\x00"}, attributedBranches)
		invalidChoice(t, err, "codec: Append: attributed.Tag holds ")
	})
}

// TestAppendChoiceBranches pins item 5 of Append's doc for a content choice
// occurring once: with no WithBranches set for its interface, Append and
// CheckAppend return Decode's ErrTypeMismatch, whatever the value, a branch
// the set would place and nil included, so a reading that takes the branch
// from the value alone fails it; under the set, CheckAppend returns nil, and
// for a choice of no {particles} it returns nil with no set. A set placing an
// implementer other than codegen's, of a branch's shape, is placed as Decode
// places it.
func TestAppendChoiceBranches(t *testing.T) {
	schema := schemaOf(t, pickSchema)
	for _, v := range []pick{pickA{A: "x"}, nil} {
		_, err := appendChecked(t, nil, schema, strict.New(), sname("r"), v)
		if !errors.Is(err, ErrTypeMismatch) || !strings.Contains(err.Error(), "no WithBranches supplies the branches of codec.pick") {
			t.Errorf("Append(%#v) with no WithBranches = %v, want ErrTypeMismatch", v, err)
		}
	}
	if err := CheckAppend[pick](schema, sname("r"), pickBranches); err != nil {
		t.Errorf("CheckAppend[pick] with its set = %v, want nil", err)
	}
	if err := CheckAppend[ecEmpty](schemaOf(t, ecEmptySchema), sname("r")); err != nil {
		t.Errorf("CheckAppend[ecEmpty] = %v, want nil", err)
	}
	other := WithBranches[pick](pickA2{}, pickB{})
	roundTrip(t, schema, sname("r"), pick(pickA2{A: "x"}), `<r xmlns="urn:s"><a>x</a></r>`, deepEqual, other)
}
