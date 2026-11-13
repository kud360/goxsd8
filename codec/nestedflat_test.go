package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// The nested-flatten schemas, each declaring s:r, whose content holds a model
// group particle codegen flattens other than as a repeated content {particle}
// (codegen/doc.go "# Particles"): nfSeqSchema's sequence(a, sequence(b?)
// repeated), a nested repeated emptiable sequence; nfOnceSchema's the content
// choice(a, sequence(b?)) occurring exactly once, flattened by its emptiable
// model-group alternative; nfNestedSchema's sequence(a, choice(c,
// sequence(b?))), that choice nested and occurring exactly once; nfRefSchema's
// sequence(a, <group ref> repeated), the group sequence(b?); nfAfterSchema's
// sequence(a, sequence(b?) repeated, c), a particle after the flattened one;
// nfWildSchema's sequence(a, sequence(b?, any over ##other?) repeated), a
// wildcard in the nested reach; nfSpliceSchema's sequence(a, sequence(c,
// sequence(b?, any over ##other?) repeated)), the flattened particle inside a
// spliced sequence. The rest each hold a reach particle sharing an expanded
// name with another particle: nfTwoSchema's sequence(sequence(a repeated)
// repeated, b, sequence(a repeated) repeated), particlesZ034_a's shape, two
// reaches; nfBeforeSchema's sequence(b, sequence(b?) repeated), a sibling
// before the reach; nfLaterSchema's sequence(a, sequence(b?) repeated, c, b),
// a sibling after it; nfShapesSchema's sequence(a, sequence(sequence(c, <element
// ref> s:b)?, sequence(d, b)?) repeated), one reach, its two b particles' branch
// structs of different shapes; nfOccBeforeSchema's sequence(sequence(b, c){1,2},
// a, sequence(b?) repeated) and nfOccAfterSchema's sequence(a, sequence(b?)
// repeated, c, sequence(b, d){1,2}), an occurrence field's particle before and
// after the reach; and nfChoiceSchema's sequence(sequence(b?) repeated, c,
// choice(b, d)), a choice bound in place after it.
const (
	nfSeqSchema    = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfOnceSchema   = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	nfNestedSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:choice><xs:element name="c" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:choice></xs:sequence></xs:complexType></xs:element>`
	nfRefSchema    = `<xs:group name="g"><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:group>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:group ref="s:g" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element>`
	nfAfterSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	nfWildSchema  = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/>` +
		`<xs:any namespace="##other" processContents="lax" minOccurs="0"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfSpliceSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="c" type="xs:string"/><xs:sequence maxOccurs="unbounded">` +
		`<xs:element name="b" type="xs:string" minOccurs="0"/><xs:any namespace="##other" processContents="lax" minOccurs="0"/></xs:sequence></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfTwoSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:sequence maxOccurs="100"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/></xs:sequence><xs:element name="b" type="xs:string"/>` +
		`<xs:sequence maxOccurs="100"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfBeforeSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="b" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfLaterSchema  = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence>` +
		`<xs:element name="c" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	nfShapesSchema = `<xs:element name="b" type="xs:string"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded">` +
		`<xs:sequence minOccurs="0"><xs:element name="c" type="xs:string"/><xs:element ref="s:b"/></xs:sequence><xs:sequence minOccurs="0"><xs:element name="d" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>` +
		`</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfOccBeforeSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:sequence maxOccurs="2"><xs:element name="b" type="xs:string"/><xs:element name="c" type="xs:string"/></xs:sequence>` +
		`<xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfOccAfterSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence>` +
		`<xs:element name="c" type="xs:string"/><xs:sequence maxOccurs="2"><xs:element name="b" type="xs:string"/><xs:element name="d" type="xs:string"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	nfChoiceSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence>` +
		`<xs:element name="c" type="xs:string"/><xs:choice><xs:element name="b" type="xs:string"/><xs:element name="d" type="xs:string"/></xs:choice></xs:sequence></xs:complexType></xs:element>`
)

// The types codegen emits for the nested-flatten schemas' s:r, in shape,
// which TestNestedFlatTypesAreCodegens pins: nfSeq for nfSeqSchema and
// nfRefSchema, flatOpt (flatten_test.go) for nfOnceSchema, nfNested for
// nfNestedSchema, nfAfter for nfAfterSchema, nfWild for nfWildSchema,
// nfSplice, over nfWild's interface, for nfSpliceSchema, and for each later
// schema the type of its name: nfTwo, nfBefore, nfLater and nfChoice over
// nfSeq's interface, nfShapes, nfOccBefore and nfOccAfter.
type (
	nfSeq struct {
		A        string
		Sequence []nfSeqItem
	}
	nfSeqItem interface{ isNfSeqItem() }
	nfSeqB    struct{ B string }

	nfNested struct {
		A      string
		Choice []nfNestedItem
	}
	nfNestedItem interface{ isNfNestedItem() }
	nfNestedC    struct{ C string }
	nfNestedB    struct{ B string }

	nfAfter struct {
		A        string
		Sequence []nfSeqItem
		C        string
	}

	nfWild struct {
		A        string
		Sequence []nfWildItem
	}
	nfWildItem interface{ isNfWildItem() }
	nfWildB    struct{ B string }
	nfWildAny  struct{ Any anyElement }

	nfSplice struct {
		A, C     string
		Sequence []nfWildItem
	}

	nfTwo struct {
		Sequence  []nfTwoItem
		B         string
		Sequence2 []nfTwo2Item
	}
	nfTwoItem  interface{ isNfTwoItem() }
	nfTwoA     struct{ A string }
	nfTwo2Item interface{ isNfTwo2Item() }
	nfTwo2A    struct{ A string }

	nfBefore struct {
		B        string
		Sequence []nfSeqItem
	}

	nfLater struct {
		A        string
		Sequence []nfSeqItem
		C, B     string
	}

	nfShapes struct {
		A        string
		Sequence []nfShapesItem
	}
	nfShapesItem interface{ isNfShapesItem() }
	nfShapesC    struct{ C string }
	nfShapesB    struct{ B nfB }
	nfShapesD    struct{ D string }
	nfShapesB2   struct{ B string }
	nfB          string

	nfOccBefore struct {
		Sequence  []nfOccBC
		A         string
		Sequence2 []nfSeqItem
	}
	nfOccBC struct{ B, C string }

	nfOccAfter struct {
		A         string
		Sequence  []nfSeqItem
		C         string
		Sequence2 []nfOccBD
	}
	nfOccBD struct{ B, D string }

	nfChoice struct {
		Sequence []nfSeqItem
		C        string
		Choice   nfChoiceItem
	}
	nfChoiceItem interface{ isNfChoiceItem() }
	nfChoiceB    struct{ B string }
	nfChoiceD    struct{ D string }
)

func (nfSeqB) isNfSeqItem()        {}
func (nfNestedC) isNfNestedItem()  {}
func (nfNestedB) isNfNestedItem()  {}
func (nfWildB) isNfWildItem()      {}
func (nfWildAny) isNfWildItem()    {}
func (nfTwoA) isNfTwoItem()        {}
func (nfTwo2A) isNfTwo2Item()      {}
func (nfShapesC) isNfShapesItem()  {}
func (nfShapesB) isNfShapesItem()  {}
func (nfShapesD) isNfShapesItem()  {}
func (nfShapesB2) isNfShapesItem() {}
func (nfChoiceB) isNfChoiceItem()  {}
func (nfChoiceD) isNfChoiceItem()  {}

// The branch sets of the nested-flatten interfaces.
var (
	nfSeqBranches    = WithBranches[nfSeqItem](nfSeqB{})
	nfNestedBranches = WithBranches[nfNestedItem](nfNestedC{}, nfNestedB{})
	nfWildBranches   = WithBranches[nfWildItem](nfWildB{}, nfWildAny{})
	nfTwoBranches    = []Option{WithBranches[nfTwoItem](nfTwoA{}), WithBranches[nfTwo2Item](nfTwo2A{})}
	nfShapesBranches = WithBranches[nfShapesItem](nfShapesC{}, nfShapesB{}, nfShapesD{}, nfShapesB2{})
	nfChoiceBranches = []Option{nfSeqBranches, WithBranches[nfChoiceItem](nfChoiceB{}, nfChoiceD{})}
)

// TestNestedFlatTypesAreCodegens pins the types above to those codegen emits
// for each nested-flatten schema's s:r, in shape, so the rows decoding into
// them decode into codegen's types.
func TestNestedFlatTypesAreCodegens(t *testing.T) {
	branch := func(iface, name, field string) string {
		return "type " + name + " struct {\n\t" + field + "\n}\n\nfunc (" + name + ") is" + iface + "() {}\n"
	}
	seq := []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceB", "B string")}
	wild := []string{branch("RSequence", "RSequenceB", "B string"), branch("RSequence", "RSequenceAny", "Any AnyElement")}
	rows := []struct {
		decls string
		want  []string
	}{
		{nfSeqSchema, seq},
		{nfRefSchema, seq},
		{nfOnceSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}},
		{nfNestedSchema, []string{"type R struct {\n\tA      string\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceC", "C string"), branch("RChoice", "RChoiceB", "B string")}},
		{nfAfterSchema, []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n\tC        string\n}\n", branch("RSequence", "RSequenceB", "B string")}},
		{nfWildSchema, append([]string{"type R struct {\n\tA        string\n\tSequence []RSequence\n}\n"}, wild...)},
		{nfSpliceSchema, append([]string{"type R struct {\n\tA        string\n\tC        string\n\tSequence []RSequence\n}\n"}, wild...)},
		{nfTwoSchema, []string{"type R struct {\n\tSequence  []RSequence\n\tB         string\n\tSequence2 []RSequence2\n}\n", branch("RSequence", "RSequenceA", "A string"), branch("RSequence2", "RSequence2A", "A string")}},
		{nfBeforeSchema, []string{"type R struct {\n\tB        string\n\tSequence []RSequence\n}\n", seq[1]}},
		{nfLaterSchema, []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n\tC        string\n\tB        string\n}\n", seq[1]}},
		{nfShapesSchema, []string{"type B string\n", "type R struct {\n\tA        string\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceC", "C string"), branch("RSequence", "RSequenceB", "B B"),
			branch("RSequence", "RSequenceD", "D string"), branch("RSequence", "RSequenceB2", "B string")}},
		{nfOccBeforeSchema, []string{"type R struct {\n\tSequence  []RSequence\n\tA         string\n\tSequence2 []RSequence2\n}\n", "type RSequence struct {\n\tB string\n\tC string\n}\n", branch("RSequence2", "RSequence2B", "B string")}},
		{nfOccAfterSchema, []string{"type R struct {\n\tA         string\n\tSequence  []RSequence\n\tC         string\n\tSequence2 []RSequence2\n}\n", seq[1], "type RSequence2 struct {\n\tB string\n\tD string\n}\n"}},
		{nfChoiceSchema, []string{"type R struct {\n\tSequence []RSequence\n\tC        string\n\tChoice   RChoice\n}\n", seq[1], branch("RChoice", "RChoiceB", "B string"), branch("RChoice", "RChoiceD", "D string")}},
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

// TestDecodeNestedFlattened pins the nested flatten lift: a nested model group
// particle codegen flattens, and a choice occurring exactly once that codegen
// flattens, content or nested, decode into the flattened slice field, one item
// per child element in document order, each the branch of the basic particle
// the child is attributed to (§3.8.4.2 key-vpath), no grouping into
// occurrences kept: cvc-accept clause 3's Note fixes no partition, so the
// items are codegen's convention, a non-empty occurrence each. A child of a
// content model holding a nested reach is attributed by its particle's Matcher
// position, an element's and a wildcard's alike, through a spliced sequence
// too, so a reach particle sharing an expanded name with a sibling before or
// after the reach, with another reach's particle, with a particle of an
// occurrence field or a choice bound in place, or with a particle of its own
// reach whose branch struct differs in shape, puts each child to its own
// particle's field. CheckDecode returns nil for each row's T.
func TestDecodeNestedFlattened(t *testing.T) {
	w := anyElement{Space: "urn:o", Local: "w"}
	const ow = `<o:w xmlns:o="urn:o"/>`
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"a nested repeated emptiable sequence", nfSeqSchema, "<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b>", into[nfSeq](nfSeqBranches),
			nfSeq{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}, nfSeqB{B: "3"}}}},
		{"a nested repeated emptiable sequence, absent", nfSeqSchema, "<s:a>1</s:a>", into[nfSeq](nfSeqBranches), nfSeq{A: "1"}},
		{"a content choice occurring once with an emptiable model-group alternative, its group's", nfOnceSchema, "<s:b>2</s:b>", into[flatOpt](flatOptBranches),
			flatOpt{Choice: []flatOptItem{flatOptB{B: "2"}}}},
		{"a content choice occurring once with an emptiable model-group alternative, its element's", nfOnceSchema, "<s:a>1</s:a>", into[flatOpt](flatOptBranches),
			flatOpt{Choice: []flatOptItem{flatOptA{A: "1"}}}},
		{"a nested choice occurring once with an emptiable model-group alternative", nfNestedSchema, "<s:a>1</s:a><s:b>2</s:b>", into[nfNested](nfNestedBranches),
			nfNested{A: "1", Choice: []nfNestedItem{nfNestedB{B: "2"}}}},
		{"a nested choice occurring once with an emptiable model-group alternative, its element's", nfNestedSchema, "<s:a>1</s:a><s:c>3</s:c>", into[nfNested](nfNestedBranches),
			nfNested{A: "1", Choice: []nfNestedItem{nfNestedC{C: "3"}}}},
		{"a nested <group ref> repeated to an emptiable sequence", nfRefSchema, "<s:a>1</s:a><s:b>2</s:b>", into[nfSeq](nfSeqBranches),
			nfSeq{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}}}},
		{"a particle after the nested flattened one", nfAfterSchema, "<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b><s:c>4</s:c>", into[nfAfter](nfSeqBranches),
			nfAfter{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}, nfSeqB{B: "3"}}, C: "4"}},
		{"a wildcard in the nested reach", nfWildSchema, "<s:a>1</s:a><s:b>2</s:b>" + ow + "<s:b>3</s:b>" + ow, into[nfWild](nfWildBranches),
			nfWild{A: "1", Sequence: []nfWildItem{nfWildB{B: "2"}, nfWildAny{Any: w}, nfWildB{B: "3"}, nfWildAny{Any: w}}}},
		{"a wildcard in a nested reach inside a spliced sequence", nfSpliceSchema, "<s:a>1</s:a><s:c>2</s:c>" + ow + "<s:b>3</s:b>", into[nfSplice](nfWildBranches),
			nfSplice{A: "1", C: "2", Sequence: []nfWildItem{nfWildAny{Any: w}, nfWildB{B: "3"}}}},
		{"two reaches of one expanded name, particlesZ034_a's shape", nfTwoSchema, "<s:a>1</s:a><s:a>2</s:a><s:b>3</s:b><s:a>4</s:a>", into[nfTwo](nfTwoBranches...),
			nfTwo{Sequence: []nfTwoItem{nfTwoA{A: "1"}, nfTwoA{A: "2"}}, B: "3", Sequence2: []nfTwo2Item{nfTwo2A{A: "4"}}}},
		{"a nested reach's particle sharing a sibling's expanded name", nfBeforeSchema, "<s:b>1</s:b><s:b>2</s:b>", into[nfBefore](nfSeqBranches),
			nfBefore{B: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}}}},
		{"a sibling after a nested reach sharing its particle's expanded name", nfLaterSchema, "<s:a>1</s:a><s:b>2</s:b><s:c>3</s:c><s:b>4</s:b>", into[nfLater](nfSeqBranches),
			nfLater{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}}, C: "3", B: "4"}},
		{"a nested reach's two particles of one expanded name, branch structs of two shapes, the <element ref>'s", nfShapesSchema, "<s:a>1</s:a><s:c>2</s:c><s:b>3</s:b>", into[nfShapes](nfShapesBranches),
			nfShapes{A: "1", Sequence: []nfShapesItem{nfShapesC{C: "2"}, nfShapesB{B: "3"}}}},
		{"a nested reach's two particles of one expanded name, branch structs of two shapes, the local one's", nfShapesSchema, "<s:a>1</s:a><s:d>2</s:d><s:b>3</s:b><s:c>4</s:c><s:b>5</s:b>", into[nfShapes](nfShapesBranches),
			nfShapes{A: "1", Sequence: []nfShapesItem{nfShapesD{D: "2"}, nfShapesB2{B: "3"}, nfShapesC{C: "4"}, nfShapesB{B: "5"}}}},
		{"an occurrence field's particle before a nested reach sharing its expanded name", nfOccBeforeSchema, "<s:b>1</s:b><s:c>2</s:c><s:b>3</s:b><s:c>4</s:c><s:a>5</s:a><s:b>6</s:b>", into[nfOccBefore](nfSeqBranches),
			nfOccBefore{Sequence: []nfOccBC{{B: "1", C: "2"}, {B: "3", C: "4"}}, A: "5", Sequence2: []nfSeqItem{nfSeqB{B: "6"}}}},
		{"an occurrence field's particle after a nested reach sharing its expanded name", nfOccAfterSchema, "<s:a>1</s:a><s:b>2</s:b><s:c>3</s:c><s:b>4</s:b><s:d>5</s:d><s:b>6</s:b><s:d>7</s:d>", into[nfOccAfter](nfSeqBranches),
			nfOccAfter{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}}, C: "3", Sequence2: []nfOccBD{{B: "4", D: "5"}, {B: "6", D: "7"}}}},
		{"a choice bound in place after a nested reach sharing its particle's expanded name", nfChoiceSchema, "<s:b>1</s:b><s:b>2</s:b><s:c>3</s:c><s:b>4</s:b>", into[nfChoice](nfChoiceBranches...),
			nfChoice{Sequence: []nfSeqItem{nfSeqB{B: "1"}, nfSeqB{B: "2"}}, C: "3", Choice: nfChoiceB{B: "4"}}},
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

// TestDecodeNestedFlattenedDeclines pins what the nested flatten lift leaves
// declining, Decode and CheckDecode alike: a nested particle codegen
// flattens inside a content particle held in an occurrence field,
// whose items' partition reads no repetition inside an item; a nested reach's
// two particles of one expanded name whose branch structs have one shape, as
// constructSharedBranch, as a content reach's do
// (TestDecodeFlattenSharedBranch).
func TestDecodeNestedFlattenedDeclines(t *testing.T) {
	rows := []struct {
		name, decls, body string
		construct         construct
	}{
		{"a nested flattened sequence in an occurrence field's item", root(`<xs:sequence maxOccurs="2"><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence>`),
			"<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b>", constructNestedGroup},
		{"a nested reach's two particles of one expanded name", sequenceOf(`<xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:sequence minOccurs="0"><xs:element name="c" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>` +
			`<xs:sequence minOccurs="0"><xs:element name="d" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:sequence>`),
			"<s:a>1</s:a><s:c>2</s:c><s:b>3</s:b>", constructSharedBranch},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf("", row.body)
			mustAccept(t, schema, data)
			_, decoded := Decode[struct{}](schema, strict.New(), data)
			for _, got := range []struct {
				name string
				err  error
			}{{"Decode", decoded}, {"CheckDecode", CheckDecode[struct{}](schema, rootName)}} {
				var de *DeclineError
				if !errors.As(got.err, &de) || de.Construct() != (Construct{row.construct}) {
					t.Errorf("%s = %v, want a *DeclineError of %v", got.name, got.err, Construct{row.construct})
				}
			}
		})
	}
}

// TestCheckDecodeNestedFlattenedWildcardOrdinal pins item 5's wildcard
// ordinal inside a nested reach: of sequence(any over urn:o, sequence(any over
// urn:p?, any over urn:q?) repeated), the reach's two wildcard alternatives
// share a branch shape, and the second's subject is wildcard particle 3, the
// sibling counted first.
func TestCheckDecodeNestedFlattenedWildcardOrdinal(t *testing.T) {
	schema := schemaOf(t, sequenceOf(`<xs:any namespace="urn:o" processContents="lax"/><xs:sequence maxOccurs="unbounded">`+
		`<xs:any namespace="urn:p" processContents="lax" minOccurs="0"/><xs:any namespace="urn:q" processContents="lax" minOccurs="0"/></xs:sequence>`))
	err := CheckDecode[struct{}](schema, rootName)
	const want = "codec does not decode two branches of a sealed interface whose structs have one shape: wildcard particle 3 of the {type definition} of element {urn:s}r"
	var de *DeclineError
	if !errors.As(err, &de) || de.Construct() != (Construct{constructSharedBranch}) || !strings.HasPrefix(err.Error(), want) {
		t.Errorf("CheckDecode = %v, want a *DeclineError of %v with the prefix %q", err, Construct{constructSharedBranch}, want)
	}
}

// TestCheckDecodeNestedFlattenedMismatch pins item 6 at a nested flattened
// field: a T whose second field is not named Sequence is no type codegen
// emits for nfSeqSchema's s:r, and the error names that field by its index.
func TestCheckDecodeNestedFlattenedMismatch(t *testing.T) {
	type nfMisnamed struct {
		A   string
		Seq []nfSeqItem
	}
	err := CheckDecode[nfMisnamed](schemaOf(t, nfSeqSchema), rootName, nfSeqBranches)
	const want = "codec: T is not the type codegen emits for the document element: field 1 of codec.nfMisnamed is Seq []codec.nfSeqItem where codegen emits Sequence, a slice of a sealed interface"
	if !errors.Is(err, ErrTypeMismatch) || err.Error() != want {
		t.Errorf("CheckDecode = %v, want %q", err, want)
	}
}

// TestCheckDecodeUntrackedNestedFlattened pins the nested flattened arm's
// ContentMatcher decline: sequence(any?, (a{1,2}){1,21845}), the nested
// particle flattened by its repeating edge, is a content model
// xsd.Schema.ContentMatcher declines as too wide to track, so CheckDecode
// returns constructUntrackedModel, a *DeclineError and no plain error, though
// the content model's matcherSlot, which its wildcard and its nested reach
// each give it, would have Decode build its Matcher through
// matcherSlot.attributor. Validation leaves such an instance undecided, so
// Decode returns an *UndecidedError, item 4, before it binds.
func TestCheckDecodeUntrackedNestedFlattened(t *testing.T) {
	schema := schemaOf(t, sequenceOf(`<xs:any namespace="##other" processContents="lax" minOccurs="0"/><xs:sequence maxOccurs="21845"><xs:element name="a" type="xs:string" maxOccurs="2"/></xs:sequence>`))
	var de *DeclineError
	if err := CheckDecode[struct{}](schema, rootName); !errors.As(err, &de) || de.Construct() != (Construct{constructUntrackedModel}) {
		t.Errorf("CheckDecode = %v, want a *DeclineError of %v", err, Construct{constructUntrackedModel})
	}
	var ue *UndecidedError
	if _, err := Decode[struct{}](schema, strict.New(), instanceOf("", "<s:a>1</s:a>")); !errors.As(err, &ue) {
		t.Errorf("Decode = %v, want an *UndecidedError", err)
	}
}
