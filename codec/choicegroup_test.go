package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// cgEl is a local element declaration named name of type xs:string.
func cgEl(name string) string { return `<xs:element name="` + name + `" type="xs:string"/>` }

// The choice-group schemas, each declaring s:r, whose content holds a choice
// codegen does not flatten with a model-group alternative (codegen/doc.go
// "# Complex types"): cgSeqSchema's content choice(a, sequence(b, c)), the
// type itself, its sequence alternative spliced; cgRefSchema's the same
// through a <group ref> to the sequence group g; cgChoiceSchema's choice(a,
// choice(b, c)); cgFlatSchema's choice(a, sequence(b repeated) repeated),
// whose alternative codegen flattens; cgOptSchema's choice(a, sequence(b, c)
// optional), emptiable, so a struct holding the choice, whose optional
// alternative is spliced at {min occurs} 1; cgNestedSchema's sequence(x,
// choice(a, sequence(b, c))), a nested choice; cgRepeatSchema's choice(a,
// sequence(b, c)) repeated, an occurrence field; cgOneSchema's choice(a,
// sequence(b)), a spliced alternative of one field; cgOptFieldSchema's
// choice(a, sequence(b, c optional)), one holding an optional field;
// cgInnerChoiceSchema's choice(a, sequence(b, choice(c, d))), a spliced
// alternative holding a nested choice; cgInnerSeqSchema's choice(a,
// sequence(b, sequence(c, d))), one holding a nested spliced sequence;
// cgInnerRefSchema's choice(a, <group ref>), the group sequence(b, choice(c,
// d)); cgWildSchema's choice(a, sequence(b, any over ##other)), a spliced
// alternative holding a wildcard; cgOptWildSchema's choice(a, any over
// ##other optional), an optional wildcard alternative, which leaves the
// choice unflattened and emptiable, so a struct holding it; and
// cgTwoChoiceSchema's choice(sequence(b, choice(c, d)), sequence(b
// unqualified of xs:int, choice(e, f))) and cgTwoFlatSchema's the same with
// sequence(c repeated) repeated and sequence(d repeated) repeated in place of
// the choices, flattened, two spliced alternatives whose fields are named
// alike, so that a branch is told from the other by its fields' Go types; and
// cgTwoNestedSchema's choice(choice(a, b), choice(c, d)), two nested-choice
// alternatives whose branch structs are each {Choice I}, so that a branch is
// told from the other by its interface's branch set; and cgTwoOneSchema's
// choice(choice(a, b), choice(a unqualified)), whose second interface's one
// branch the first choice's alternative a admits.
const (
	cgTwoOneSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice>` +
		`<xs:choice><xs:element name="a" form="unqualified" type="xs:string"/></xs:choice></xs:choice></xs:complexType></xs:element>`
	cgTwoNestedSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice>` +
		`<xs:choice><xs:element name="c" type="xs:string"/><xs:element name="d" type="xs:string"/></xs:choice></xs:choice></xs:complexType></xs:element>`
	cgSeqSchema         = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgRefSchema         = `<xs:group name="g"><xs:sequence><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:sequence></xs:group><xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:group ref="s:g"/></xs:choice></xs:complexType></xs:element>`
	cgChoiceSchema      = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:choice><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:choice></xs:choice></xs:complexType></xs:element>`
	cgFlatSchema        = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" maxOccurs="unbounded"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgOptSchema         = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence minOccurs="0"><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgNestedSchema      = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="x" type="xs:string"/><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:sequence></xs:choice></xs:sequence></xs:complexType></xs:element>`
	cgRepeatSchema      = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgOneSchema         = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgOptFieldSchema    = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string" minOccurs="0"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgInnerChoiceSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/>` +
		`<xs:choice><xs:element name="c" type="xs:string"/><xs:element name="d" type="xs:string"/></xs:choice></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgInnerSeqSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/>` +
		`<xs:sequence><xs:element name="c" type="xs:string"/><xs:element name="d" type="xs:string"/></xs:sequence></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgInnerRefSchema = `<xs:group name="g"><xs:sequence><xs:element name="b" type="xs:string"/><xs:choice><xs:element name="c" type="xs:string"/><xs:element name="d" type="xs:string"/></xs:choice></xs:sequence></xs:group>` +
		`<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:group ref="s:g"/></xs:choice></xs:complexType></xs:element>`
	cgWildSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string"/>` +
		`<xs:any namespace="##other" processContents="lax"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgOptWildSchema   = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:any namespace="##other" processContents="lax" minOccurs="0"/></xs:choice></xs:complexType></xs:element>`
	cgTwoChoiceSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:sequence><xs:element name="b" type="xs:string"/><xs:choice><xs:element name="c" type="xs:string"/><xs:element name="d" type="xs:string"/></xs:choice></xs:sequence>` +
		`<xs:sequence><xs:element name="b" form="unqualified" type="xs:int"/><xs:choice><xs:element name="e" type="xs:string"/><xs:element name="f" type="xs:string"/></xs:choice></xs:sequence></xs:choice></xs:complexType></xs:element>`
	cgTwoFlatSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:sequence><xs:element name="b" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="c" type="xs:string" maxOccurs="unbounded"/></xs:sequence></xs:sequence>` +
		`<xs:sequence><xs:element name="b" form="unqualified" type="xs:int"/><xs:sequence maxOccurs="unbounded"><xs:element name="d" type="xs:string" maxOccurs="unbounded"/></xs:sequence></xs:sequence></xs:choice></xs:complexType></xs:element>`
)

// The types codegen emits for the choice-group schemas' s:r, in shape, which
// TestChoiceGroupTypesAreCodegens pins: cgR, its branches cgA and cgSeq, for
// cgSeqSchema and cgRefSchema; cgCh for cgChoiceSchema; cgFl for
// cgFlatSchema; cgO for cgOptSchema; cgN for cgNestedSchema; cgRep for
// cgRepeatSchema; cgOne for cgOneSchema; cgOF for cgOptFieldSchema; cgIC for
// cgInnerChoiceSchema and cgInnerRefSchema; cgIS for cgInnerSeqSchema; cgW
// for cgWildSchema; cgTC for cgTwoChoiceSchema; cgTF for cgTwoFlatSchema;
// cgOW for cgOptWildSchema; cgTN for cgTwoNestedSchema; and cgTO, its first
// branch's interface cgTNIn1's, for cgTwoOneSchema.
type (
	cgTO    interface{ isCgTO() }
	cgTO1   struct{ Choice cgTNIn1 }
	cgTO2   struct{ Choice cgTOIn2 }
	cgTOIn2 interface{ isCgTOIn2() }
	cgTOA   struct{ A string }

	cgTN    interface{ isCgTN() }
	cgTN1   struct{ Choice cgTNIn1 }
	cgTNIn1 interface{ isCgTNIn1() }
	cgTNA   struct{ A string }
	cgTNB   struct{ B string }
	cgTN2   struct{ Choice cgTNIn2 }
	cgTNIn2 interface{ isCgTNIn2() }
	cgTNC   struct{ C string }
	cgTND   struct{ D string }

	cgR   interface{ isCgR() }
	cgA   struct{ A string }
	cgSeq struct{ B, C string }

	cgCh      interface{ isCgCh() }
	cgChA     struct{ A string }
	cgChC     struct{ Choice cgChInner }
	cgChInner interface{ isCgChInner() }
	cgChB2    struct{ B string }
	cgChC2    struct{ C string }

	cgFl     interface{ isCgFl() }
	cgFlA    struct{ A string }
	cgFlSeq  struct{ Sequence []cgFlItem }
	cgFlItem interface{ isCgFlItem() }
	cgFlB    struct{ B string }

	cgO       struct{ Choice cgOChoice }
	cgOChoice interface{ isCgOChoice() }
	cgOA      struct{ A string }
	cgOSeq    struct{ B, C string }

	cgN struct {
		X      string
		Choice cgOChoice
	}

	cgRep struct{ Choice []cgOChoice }

	cgOne    interface{ isCgOne() }
	cgOneA   struct{ A string }
	cgOneSeq struct{ B string }

	cgOF    interface{ isCgOF() }
	cgOFA   struct{ A string }
	cgOFSeq struct {
		B string
		C *string
	}

	cgIC    interface{ isCgIC() }
	cgICA   struct{ A string }
	cgICSeq struct {
		B      string
		Choice cgICInner
	}
	cgICInner interface{ isCgICInner() }
	cgICC     struct{ C string }
	cgICD     struct{ D string }

	cgIS    interface{ isCgIS() }
	cgISA   struct{ A string }
	cgISSeq struct{ B, C, D string }

	cgTC  interface{ isCgTC() }
	cgTC1 struct {
		B      string
		Choice cgICInner
	}
	cgTC2 struct {
		B      int32
		Choice cgTCInner
	}
	cgTCInner interface{ isCgTCInner() }
	cgTCE     struct{ E string }
	cgTCF     struct{ F string }

	cgTF  interface{ isCgTF() }
	cgTF1 struct {
		B        string
		Sequence []cgTFItem
	}
	cgTF2 struct {
		B        int32
		Sequence []cgTFItem2
	}
	cgTFItem  interface{ isCgTFItem() }
	cgTFC     struct{ C string }
	cgTFItem2 interface{ isCgTFItem2() }
	cgTFD     struct{ D string }

	cgOW       struct{ Choice cgOWChoice }
	cgOWChoice interface{ isCgOWChoice() }
	cgOWA      struct{ A string }
	cgOWAny    struct{ Any anyElement }

	// cgSwapped and cgRetyped are cgSeq's fields in another order and of
	// another Go type, branches of cgR codegen does not emit.
	cgSwapped struct{ C, B string }
	cgRetyped struct {
		B string
		C int32
	}

	cgW    interface{ isCgW() }
	cgWA   struct{ A string }
	cgWSeq struct {
		B   string
		Any anyElement
	}
)

func (cgA) isCgR()            {}
func (cgSeq) isCgR()          {}
func (cgChA) isCgCh()         {}
func (cgChC) isCgCh()         {}
func (cgChB2) isCgChInner()   {}
func (cgChC2) isCgChInner()   {}
func (cgFlA) isCgFl()         {}
func (cgFlSeq) isCgFl()       {}
func (cgFlB) isCgFlItem()     {}
func (cgOA) isCgOChoice()     {}
func (cgOSeq) isCgOChoice()   {}
func (cgOneA) isCgOne()       {}
func (cgOneSeq) isCgOne()     {}
func (cgOFA) isCgOF()         {}
func (cgOFSeq) isCgOF()       {}
func (cgICA) isCgIC()         {}
func (cgICSeq) isCgIC()       {}
func (cgICC) isCgICInner()    {}
func (cgICD) isCgICInner()    {}
func (cgISA) isCgIS()         {}
func (cgISSeq) isCgIS()       {}
func (cgWA) isCgW()           {}
func (cgTC1) isCgTC()         {}
func (cgTC2) isCgTC()         {}
func (cgTCE) isCgTCInner()    {}
func (cgTCF) isCgTCInner()    {}
func (cgTF1) isCgTF()         {}
func (cgTF2) isCgTF()         {}
func (cgTFC) isCgTFItem()     {}
func (cgTFD) isCgTFItem2()    {}
func (cgOWA) isCgOWChoice()   {}
func (cgOWAny) isCgOWChoice() {}
func (cgSwapped) isCgR()      {}
func (cgRetyped) isCgR()      {}
func (cgWSeq) isCgW()         {}
func (cgTN1) isCgTN()         {}
func (cgTN2) isCgTN()         {}
func (cgTNA) isCgTNIn1()      {}
func (cgTNB) isCgTNIn1()      {}
func (cgTNC) isCgTNIn2()      {}
func (cgTND) isCgTNIn2()      {}
func (cgTO1) isCgTO()         {}
func (cgTO2) isCgTO()         {}
func (cgTOA) isCgTOIn2()      {}

// The branch sets of the choice-group interfaces.
var (
	cgBranches   = WithBranches[cgR](cgA{}, cgSeq{})
	cgChBranches = []Option{WithBranches[cgCh](cgChA{}, cgChC{}), WithBranches[cgChInner](cgChB2{}, cgChC2{})}
	cgFlBranches = []Option{WithBranches[cgFl](cgFlA{}, cgFlSeq{}), WithBranches[cgFlItem](cgFlB{})}
	cgOBranches  = WithBranches[cgOChoice](cgOA{}, cgOSeq{})
	cgOneBranch  = WithBranches[cgOne](cgOneA{}, cgOneSeq{})
	cgOFBranches = WithBranches[cgOF](cgOFA{}, cgOFSeq{})
	cgICBranches = []Option{WithBranches[cgIC](cgICA{}, cgICSeq{}), WithBranches[cgICInner](cgICC{}, cgICD{})}
	cgISBranches = WithBranches[cgIS](cgISA{}, cgISSeq{})
	cgWBranches  = WithBranches[cgW](cgWA{}, cgWSeq{})
	cgTCBranches = []Option{WithBranches[cgTC](cgTC1{}, cgTC2{}), WithBranches[cgICInner](cgICC{}, cgICD{}), WithBranches[cgTCInner](cgTCE{}, cgTCF{})}
	cgOWBranches = WithBranches[cgOWChoice](cgOWA{}, cgOWAny{})
	cgTFBranches = []Option{WithBranches[cgTF](cgTF1{}, cgTF2{}), WithBranches[cgTFItem](cgTFC{}), WithBranches[cgTFItem2](cgTFD{})}
	// cgTNBranches supplies cgTN's branches out of alternative order, so that
	// cgTN2 is matched while the first alternative is still unmatched.
	cgTNBranches = []Option{WithBranches[cgTN](cgTN2{}, cgTN1{}), WithBranches[cgTNIn1](cgTNA{}, cgTNB{}), WithBranches[cgTNIn2](cgTNC{}, cgTND{})}
	cgTOBranches = []Option{WithBranches[cgTO](cgTO1{}, cgTO2{}), WithBranches[cgTNIn1](cgTNA{}, cgTNB{}), WithBranches[cgTOIn2](cgTOA{})}
)

// TestChoiceGroupTypesAreCodegens pins the types above to those codegen
// emits for each choice-group schema's s:r, in shape (codegen/doc.go
// "# Complex types"), so the rows decoding into them decode into codegen's
// types.
func TestChoiceGroupTypesAreCodegens(t *testing.T) {
	seq := []string{
		"type R interface {\n\tisR()\n}\n",
		"type RA struct {\n\tA string\n}\n\nfunc (RA) isR() {}\n",
		"type RSequence struct {\n\tB string\n\tC string\n}\n\nfunc (RSequence) isR() {}\n",
	}
	held := []string{
		"type RChoice interface {\n\tisRChoice()\n}\n",
		"type RChoiceA struct {\n\tA string\n}\n\nfunc (RChoiceA) isRChoice() {}\n",
		"type RChoiceSequence struct {\n\tB string\n\tC string\n}\n\nfunc (RChoiceSequence) isRChoice() {}\n",
	}
	innerChoice := []string{
		"type RA struct {\n\tA string\n}\n\nfunc (RA) isR() {}\n",
		"type RSequence struct {\n\tB      string\n\tChoice RChoice\n}\n\nfunc (RSequence) isR() {}\n",
		"type RChoiceC struct {\n\tC string\n}\n\nfunc (RChoiceC) isRChoice() {}\n",
		"type RChoiceD struct {\n\tD string\n}\n\nfunc (RChoiceD) isRChoice() {}\n",
	}
	rows := []struct {
		decls string
		want  []string
	}{
		{cgSeqSchema, seq},
		{cgRefSchema, seq},
		{cgChoiceSchema, []string{
			"type RChoice struct {\n\tChoice RChoice2\n}\n\nfunc (RChoice) isR() {}\n",
			"type RChoice2B struct {\n\tB string\n}\n\nfunc (RChoice2B) isRChoice2() {}\n",
			"type RChoice2C struct {\n\tC string\n}\n\nfunc (RChoice2C) isRChoice2() {}\n",
		}},
		{cgFlatSchema, []string{
			"type RSequence struct {\n\tSequence []RSequence2\n}\n\nfunc (RSequence) isR() {}\n",
			"type RSequence2 interface {\n\tisRSequence2()\n}\n",
			"type RSequence2B struct {\n\tB string\n}\n\nfunc (RSequence2B) isRSequence2() {}\n",
		}},
		{cgOptSchema, append([]string{"type R struct {\n\tChoice RChoice\n}\n"}, held...)},
		{cgNestedSchema, append([]string{"type R struct {\n\tX      string\n\tChoice RChoice\n}\n"}, held...)},
		{cgRepeatSchema, append([]string{"type R struct {\n\tChoice []RChoice\n}\n"}, held...)},
		{cgOneSchema, []string{
			"type RA struct {\n\tA string\n}\n\nfunc (RA) isR() {}\n",
			"type RSequence struct {\n\tB string\n}\n\nfunc (RSequence) isR() {}\n",
		}},
		{cgOptFieldSchema, []string{
			"type RSequence struct {\n\tB string\n\tC *string\n}\n\nfunc (RSequence) isR() {}\n",
		}},
		{cgInnerChoiceSchema, innerChoice},
		{cgInnerRefSchema, innerChoice},
		{cgInnerSeqSchema, []string{
			"type RSequence struct {\n\tB string\n\tC string\n\tD string\n}\n\nfunc (RSequence) isR() {}\n",
		}},
		{cgWildSchema, []string{
			"type RSequence struct {\n\tB   string\n\tAny AnyElement\n}\n\nfunc (RSequence) isR() {}\n",
		}},
		{cgOptWildSchema, []string{
			"type R struct {\n\tChoice RChoice\n}\n",
			"type RChoiceAny struct {\n\tAny AnyElement\n}\n\nfunc (RChoiceAny) isRChoice() {}\n",
		}},
		{cgTwoChoiceSchema, []string{
			"type RSequence struct {\n\tB      string\n\tChoice RChoice\n}\n\nfunc (RSequence) isR() {}\n",
			"type RSequence2 struct {\n\tB      int32\n\tChoice RChoice2\n}\n\nfunc (RSequence2) isR() {}\n",
			"type RChoice2E struct {\n\tE string\n}\n\nfunc (RChoice2E) isRChoice2() {}\n",
		}},
		{cgTwoFlatSchema, []string{
			"type RSequence struct {\n\tB        string\n\tSequence []RSequence2\n}\n\nfunc (RSequence) isR() {}\n",
			"type RSequence3 struct {\n\tB        int32\n\tSequence []RSequence4\n}\n\nfunc (RSequence3) isR() {}\n",
			"type RSequence4D struct {\n\tD string\n}\n\nfunc (RSequence4D) isRSequence4() {}\n",
		}},
		{cgTwoNestedSchema, []string{
			"type R interface {\n\tisR()\n}\n",
			"type RChoice struct {\n\tChoice RChoice2\n}\n\nfunc (RChoice) isR() {}\n",
			"type RChoice2 interface {\n\tisRChoice2()\n}\n",
			"type RChoice2A struct {\n\tA string\n}\n\nfunc (RChoice2A) isRChoice2() {}\n",
			"type RChoice2B struct {\n\tB string\n}\n\nfunc (RChoice2B) isRChoice2() {}\n",
			"type RChoice3 struct {\n\tChoice RChoice4\n}\n\nfunc (RChoice3) isR() {}\n",
			"type RChoice4 interface {\n\tisRChoice4()\n}\n",
			"type RChoice4C struct {\n\tC string\n}\n\nfunc (RChoice4C) isRChoice4() {}\n",
			"type RChoice4D struct {\n\tD string\n}\n\nfunc (RChoice4D) isRChoice4() {}\n",
		}},
		{cgTwoOneSchema, []string{
			"type R interface {\n\tisR()\n}\n",
			"type RChoice struct {\n\tChoice RChoice2\n}\n\nfunc (RChoice) isR() {}\n",
			"type RChoice2A struct {\n\tA string\n}\n\nfunc (RChoice2A) isRChoice2() {}\n",
			"type RChoice2B struct {\n\tB string\n}\n\nfunc (RChoice2B) isRChoice2() {}\n",
			"type RChoice3 struct {\n\tChoice RChoice4\n}\n\nfunc (RChoice3) isR() {}\n",
			"type RChoice4 interface {\n\tisRChoice4()\n}\n",
			"type RChoice4A struct {\n\tA string\n}\n\nfunc (RChoice4A) isRChoice4() {}\n",
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

// TestDecodeChoiceGroup pins the model-group alternative lift
// (cvc-model-group: one alternative matches; §3.8.4.1.2): a choice's
// sequence alternative codegen splices, inline or through a <group ref>,
// optional or not, decodes into its branch's fields, each child filling its
// particle's, a nested choice and a nested spliced sequence of the branch
// included, a wildcard child the branch's raw field; a choice alternative
// into its branch's one field holding the nested interface, its branch told
// from a second choice alternative's by the branch set WithBranches supplies
// for that interface; and one codegen flattens into its branch's one slice,
// one item per child. The choice is the type itself, a struct's field, a
// nested choice's field or an occurrence field's item, where each occurrence
// of the choice holds one branch (cvc-accept clause 3). CheckDecode returns
// nil for each row's T.
func TestDecodeChoiceGroup(t *testing.T) {
	two := "2"
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"a spliced sequence alternative", cgSeqSchema, "<s:b>1</s:b><s:c>2</s:c>", into[cgR](cgBranches), cgR(cgSeq{B: "1", C: "2"})},
		{"the element alternative beside a spliced one", cgSeqSchema, "<s:a>1</s:a>", into[cgR](cgBranches), cgR(cgA{A: "1"})},
		{"a spliced alternative through a <group ref>", cgRefSchema, "<s:b>1</s:b><s:c>2</s:c>", into[cgR](cgBranches), cgR(cgSeq{B: "1", C: "2"})},
		{"a choice alternative", cgChoiceSchema, "<s:c>1</s:c>", into[cgCh](cgChBranches...), cgCh(cgChC{Choice: cgChC2{C: "1"}})},
		{"an alternative codegen flattens", cgFlatSchema, "<s:b>1</s:b><s:b>2</s:b>", into[cgFl](cgFlBranches...),
			cgFl(cgFlSeq{Sequence: []cgFlItem{cgFlB{B: "1"}, cgFlB{B: "2"}}})},
		{"an optional sequence alternative spliced at {min occurs} 1", cgOptSchema, "<s:b>1</s:b><s:c>2</s:c>", into[cgO](cgOBranches), cgO{Choice: cgOSeq{B: "1", C: "2"}}},
		{"an emptiable choice's empty match", cgOptSchema, "", into[cgO](cgOBranches), cgO{}},
		{"a nested choice's spliced alternative", cgNestedSchema, "<s:x>0</s:x><s:b>1</s:b><s:c>2</s:c>", into[cgN](cgOBranches), cgN{X: "0", Choice: cgOSeq{B: "1", C: "2"}}},
		{"a repeated choice's spliced alternative, one branch per occurrence", cgRepeatSchema, "<s:b>1</s:b><s:c>2</s:c><s:a>3</s:a><s:b>4</s:b><s:c>5</s:c>", into[cgRep](cgOBranches),
			cgRep{Choice: []cgOChoice{cgOSeq{B: "1", C: "2"}, cgOA{A: "3"}, cgOSeq{B: "4", C: "5"}}}},
		{"a spliced alternative of one field", cgOneSchema, "<s:b>1</s:b>", into[cgOne](cgOneBranch), cgOne(cgOneSeq{B: "1"})},
		{"a spliced alternative's optional field, absent", cgOptFieldSchema, "<s:b>1</s:b>", into[cgOF](cgOFBranches), cgOF(cgOFSeq{B: "1"})},
		{"a spliced alternative's optional field, present", cgOptFieldSchema, "<s:b>1</s:b><s:c>2</s:c>", into[cgOF](cgOFBranches), cgOF(cgOFSeq{B: "1", C: &two})},
		{"a spliced alternative holding a nested choice", cgInnerChoiceSchema, "<s:b>1</s:b><s:d>2</s:d>", into[cgIC](cgICBranches...), cgIC(cgICSeq{B: "1", Choice: cgICD{D: "2"}})},
		{"a <group ref> alternative holding a nested choice", cgInnerRefSchema, "<s:b>1</s:b><s:c>2</s:c>", into[cgIC](cgICBranches...), cgIC(cgICSeq{B: "1", Choice: cgICC{C: "2"}})},
		{"a spliced alternative holding a nested spliced sequence", cgInnerSeqSchema, "<s:b>1</s:b><s:c>2</s:c><s:d>3</s:d>", into[cgIS](cgISBranches), cgIS(cgISSeq{B: "1", C: "2", D: "3"})},
		{"a spliced alternative holding a wildcard", cgWildSchema, `<s:b>1</s:b><o:w xmlns:o="urn:o">2</o:w>`, into[cgW](cgWBranches),
			cgW(cgWSeq{B: "1", Any: anyElement{Space: "urn:o", Local: "w", Content: anyType{Text: "2"}}})},
		{"an optional wildcard alternative, its {min occurs} 0 read as 1", cgOptWildSchema, `<o:w xmlns:o="urn:o">2</o:w>`, into[cgOW](cgOWBranches),
			cgOW{Choice: cgOWAny{Any: anyElement{Space: "urn:o", Local: "w", Content: anyType{Text: "2"}}}}},
		{"the second of two spliced alternatives holding a nested choice, told apart by Go type", cgTwoChoiceSchema, "<b>1</b><s:e>2</s:e>", into[cgTC](cgTCBranches...),
			cgTC(cgTC2{B: 1, Choice: cgTCE{E: "2"}})},
		{"the second of two spliced alternatives holding a flattened particle, told apart by Go type", cgTwoFlatSchema, "<b>1</b><s:d>2</s:d><s:d>3</s:d>", into[cgTF](cgTFBranches...),
			cgTF(cgTF2{B: 1, Sequence: []cgTFItem2{cgTFD{D: "2"}, cgTFD{D: "3"}}})},
		{"the second of two nested choice alternatives, told apart by branch set", cgTwoNestedSchema, "<s:c>1</s:c>", into[cgTN](cgTNBranches...), cgTN(cgTN2{Choice: cgTNC{C: "1"}})},
		{"the second of two nested choice alternatives, its one branch one the first's alternatives admit", cgTwoOneSchema, "<a>1</a>", into[cgTO](cgTOBranches...), cgTO(cgTO2{Choice: cgTOA{A: "1"}})},
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

// TestDecodeChoiceGroupDeclines pins the model-group alternative's declines,
// each charged to the document element at 1:1: two branch structs of one
// shape, two nested choices' {Choice I} whose choices one branch set could
// hold, as many alternatives each of one shape as one of the other's, two
// wildcard alternatives' {Any E}, two spliced sequences' fields of one name
// and Go type, and a spliced sequence of one field beside an element
// alternative of its name and Go type; an expanded name an earlier
// alternative's spliced sequence takes; and, in an occurrence field's item, a
// sequence alternative holding a particle codegen flattens, whose prefix is
// the decline's subject. A repeated sequence alternative, which codegen
// neither splices nor flattens, decodes into its occurrence field
// (TestDecodeNestedOccurrence).
func TestDecodeChoiceGroupDeclines(t *testing.T) {
	unq := func(name string) string {
		return `<xs:element name="` + name + `" form="unqualified" type="xs:string"/>`
	}
	rows := []struct {
		name, decls, body string
		construct         construct
		prefix            string
	}{
		{"two nested choice alternatives one branch set could hold", root(`<xs:choice><xs:choice>` + cgEl("x") + `</xs:choice><xs:choice>` + unq("x") + `</xs:choice></xs:choice>`),
			"<s:x>1</s:x>", constructSharedBranch,
			"?:1:1: codec does not decode two branches of a sealed interface whose structs have one shape: a choice model group particle of the {type definition} of element {urn:s}r declared at"},
		{"two nested choice alternatives of one shape in another order", root(`<xs:choice><xs:choice>` + cgEl("a") + cgEl("b") + `</xs:choice><xs:choice>` + unq("b") + unq("a") + `</xs:choice></xs:choice>`),
			"<s:a>1</s:a>", constructSharedBranch,
			"?:1:1: codec does not decode two branches of a sealed interface whose structs have one shape: a choice model group particle of the {type definition} of element {urn:s}r declared at"},
		{"two wildcard alternatives", root(`<xs:choice><xs:any namespace="urn:n1" processContents="skip"/><xs:any namespace="urn:n2" processContents="skip"/></xs:choice>`),
			`<n:w xmlns:n="urn:n2"/>`, constructSharedBranch,
			"?:1:1: codec does not decode two branches of a sealed interface whose structs have one shape: wildcard particle 2 of the {type definition} of element {urn:s}r declared at"},
		{"two spliced sequences of one shape", root(`<xs:choice><xs:sequence>` + cgEl("x") + cgEl("y") + `</xs:sequence><xs:sequence>` + unq("x") + unq("y") + `</xs:sequence></xs:choice>`),
			"<s:x>1</s:x><s:y>2</s:y>", constructSharedBranch,
			"?:1:1: codec does not decode two branches of a sealed interface whose structs have one shape: a sequence model group particle of the {type definition} of element {urn:s}r declared at"},
		{"a spliced sequence of one field beside an element of its shape", root(`<xs:choice>` + cgEl("x") + `<xs:sequence>` + unq("x") + `</xs:sequence></xs:choice>`),
			"<s:x>1</s:x>", constructSharedBranch, ""},
		{"an element alternative after a spliced sequence of its shape", root(`<xs:choice><xs:sequence>` + unq("x") + `</xs:sequence>` + cgEl("x") + `</xs:choice>`),
			"<s:x>1</s:x>", constructSharedBranch, "?:1:1: codec does not decode two branches of a sealed interface whose structs have one shape: element {urn:s}x declared at"},
		{"a name an earlier alternative's spliced sequence takes", root(`<xs:choice><xs:sequence>` + cgEl("a") + cgEl("b") + `</xs:sequence><xs:sequence>` + cgEl("c") + cgEl("a") + `</xs:sequence></xs:choice>`),
			"<s:a>1</s:a><s:b>2</s:b>", constructSharedName, ""},
		{"a flattened particle in an occurrence item's spliced alternative", root(`<xs:choice maxOccurs="unbounded"><xs:sequence>` + cgEl("a") + `<xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" maxOccurs="unbounded"/></xs:sequence></xs:sequence>` + cgEl("c") + `</xs:choice>`),
			"<s:a>1</s:a><s:b>2</s:b><s:c>3</s:c>", constructNestedGroup,
			"?:1:1: codec does not decode a nested model group particle: a particle of the {type definition} of element {urn:s}r declared at"},
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
			var de *DeclineError
			if err := CheckDecode[struct{}](schema, rootName); !errors.As(err, &de) || de.Construct() != (Construct{row.construct}) {
				t.Errorf("CheckDecode = %v, want a *DeclineError of %v", err, Construct{row.construct})
			}
		})
	}
}

// TestDecodeNestedChoiceAlternativesBind pins that two nested-choice
// alternatives no one branch set could hold, as many alternatives each of
// one shape as one of the other's, bind with no decline, beside
// TestDecodeChoiceGroup's choice(choice(a, b), choice(c, d)): where the
// counts differ, the earlier choice's one alternative sharing a shape with
// one of the later's two, and where they agree but one alternative shares no
// shape with the other's. CheckDecode of struct{}, no interface, is then the
// mismatch of T.
func TestDecodeNestedChoiceAlternativesBind(t *testing.T) {
	unqA := `<xs:element name="a" form="unqualified" type="xs:string"/>`
	rows := []struct{ name, decls string }{
		{"one alternative before two", root(`<xs:choice><xs:choice>` + unqA + `</xs:choice><xs:choice>` + cgEl("a") + cgEl("b") + `</xs:choice></xs:choice>`)},
		{"one shape shared, one not", root(`<xs:choice><xs:choice>` + cgEl("a") + cgEl("b") + `</xs:choice><xs:choice>` + unqA + cgEl("c") + `</xs:choice></xs:choice>`)},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			err := CheckDecode[struct{}](schemaOf(t, row.decls), rootName)
			const want = "codec: T is not the type codegen emits for the document element: T is struct {}, no interface, where codegen emits a sealed interface"
			if err == nil || !errors.Is(err, ErrTypeMismatch) || err.Error() != want {
				t.Errorf("CheckDecode = %v, want %q", err, want)
			}
		})
	}
}

// cgTNStr is a branch of cgTNIn2 that is no struct.
type cgTNStr string

func (cgTNStr) isCgTNIn2() {}

// TestDecodeNestedChoiceAlternativesTypeMismatch pins item 6 for a branch
// set of a nested choice whose branch struct is one of two {Choice I}: a set
// holding a nil value, a type that is no struct, or a type twice is held to
// neither alternative, so the outer branch is held to the first alternative
// that names it, the error that alternative's mismatch; CheckDecode's error
// is Decode's.
func TestDecodeNestedChoiceAlternativesTypeMismatch(t *testing.T) {
	const prefix = "codec: T is not the type codegen emits for the document element: "
	rows := []struct {
		name  string
		inner Option
		want  string
	}{
		{"a nil branch", WithBranches[cgTNIn2](nil, cgTND{}), prefix + "WithBranches supplies a nil branch of codec.cgTNIn2"},
		{"a branch that is no struct", WithBranches[cgTNIn2](cgTNStr(""), cgTND{}), prefix + "WithBranches supplies codec.cgTNStr, no struct, as a branch of codec.cgTNIn2"},
		{"a branch twice", WithBranches[cgTNIn2](cgTNC{}, cgTNC{}), prefix + "branch codec.cgTNC of codec.cgTNIn2 has the shape of no branch struct codegen emits that an earlier branch did not match"},
	}
	schema := schemaOf(t, cgTwoNestedSchema)
	data := instanceOf("", "<s:c>1</s:c>")
	mustAccept(t, schema, data)
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			opts := []Option{WithBranches[cgTN](cgTN2{}, cgTN1{}), WithBranches[cgTNIn1](cgTNA{}, cgTNB{}), row.inner}
			got, err := Decode[cgTN](schema, strict.New(), data, opts...)
			if got != nil {
				t.Fatalf("Decode = %v, want no value", *got)
			}
			if err == nil || !errors.Is(err, ErrTypeMismatch) || err.Error() != row.want {
				t.Errorf("Decode error = %v, want %q", err, row.want)
			}
			assertProbeMismatch(t, CheckDecode[cgTN](schema, rootName, opts...), err)
		})
	}
}

// TestDecodeChoiceGroupTypeMismatch pins item 6 for a model-group
// alternative's branch: a branch struct whose fields are its alternative's
// names in another order matches no alternative, and one of its
// alternative's names whose field is of another Go type is held to that
// alternative, the error that field's mismatch; CheckDecode's error is
// Decode's.
func TestDecodeChoiceGroupTypeMismatch(t *testing.T) {
	rows := []struct {
		name   string
		branch cgR
		want   string
	}{
		{"fields out of order", cgSwapped{}, "codec: T is not the type codegen emits for the document element: branch codec.cgSwapped of codec.cgR has the shape of no branch struct codegen emits that an earlier branch did not match"},
		{"a field of another Go type", cgRetyped{}, "codec: T is not the type codegen emits for the document element: field 1 of codec.cgRetyped is C int32 where codegen emits C string"},
	}
	schema := schemaOf(t, cgSeqSchema)
	data := instanceOf("", "<s:b>1</s:b><s:c>2</s:c>")
	mustAccept(t, schema, data)
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			opt := WithBranches[cgR](cgA{}, row.branch)
			got, err := Decode[cgR](schema, strict.New(), data, opt)
			if got != nil {
				t.Fatalf("Decode = %v, want no value", *got)
			}
			if err == nil || !errors.Is(err, ErrTypeMismatch) || err.Error() != row.want {
				t.Errorf("Decode error = %v, want %q", err, row.want)
			}
			assertProbeMismatch(t, CheckDecode[cgR](schema, rootName, opt), err)
		})
	}
}
