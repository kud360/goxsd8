package codec

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// The occurrence schemas, each declaring s:r, whose content {particle}
// occurs other than exactly once and which codegen neither splices nor
// flattens, so codegen gives it a field holding the type for one occurrence
// (codegen/doc.go "# Particles"): occSeqSchema's sequence(a, b) repeated;
// occChoiceSchema's choice(a, b) repeated; occOptSchema's optional choice(a,
// b); occRefSchema's <group ref> occurring twice to sequence(a, b);
// occWildSchema's sequence(a, any) repeated, its wildcard ##any;
// occPtrSchema's optional sequence(a); occPartSchema's sequence(a, b?)
// repeated; occEmptyBranchSchema's choice(a?, b) repeated; occWildManySchema's
// sequence(a, any*) repeated; occPickSchema's choice(a, any ##other) repeated;
// occInnerSchema's sequence(choice(a, b)) repeated, an occurrence holding a
// nested choice; and occFixedSchema's choice(a{2,2}, b) repeated.
const (
	occSeqSchema         = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	occChoiceSchema      = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	occOptSchema         = `<xs:element name="r"><xs:complexType><xs:choice minOccurs="0"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	occRefSchema         = `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>` + `<xs:element name="r"><xs:complexType><xs:group ref="s:g" maxOccurs="2"/></xs:complexType></xs:element>`
	occWildSchema        = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:any processContents="lax"/></xs:sequence></xs:complexType></xs:element>`
	occPtrSchema         = `<xs:element name="r"><xs:complexType><xs:sequence minOccurs="0"><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	occPartSchema        = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	occEmptyBranchSchema = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	occWildManySchema    = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:any minOccurs="0" maxOccurs="unbounded" processContents="lax"/></xs:sequence></xs:complexType></xs:element>`
	occPickSchema        = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:any namespace="##other" processContents="lax"/></xs:choice></xs:complexType></xs:element>`
	occInnerSchema       = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice></xs:sequence></xs:complexType></xs:element>`
	occFixedSchema       = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="2" maxOccurs="2"/><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
)

// The types codegen emits for the occurrence schemas' s:r, in shape, which
// TestOccurrenceTypesAreCodegens pins: occSeq for occSeqSchema and
// occRefSchema, occChoice for occChoiceSchema and occEmptyBranchSchema, occOpt
// for occOptSchema, over occChoice's interface, occWild, occPtr, occPart,
// occWildMany, occPick, occInner and occFixed for the schemas so named.
type (
	occSeq     struct{ Sequence []occSeqItem }
	occSeqItem struct{ A, B string }

	occChoice     struct{ Choice []occChoiceItem }
	occOpt        struct{ Choice occChoiceItem }
	occChoiceItem interface{ isOccChoiceItem() }
	occChoiceA    struct{ A string }
	occChoiceB    struct{ B string }

	occWild     struct{ Sequence []occWildItem }
	occWildItem struct {
		A   string
		Any anyElement
	}

	occPtr     struct{ Sequence *occPtrItem }
	occPtrItem struct{ A string }

	occPart     struct{ Sequence []occPartItem }
	occPartItem struct {
		A string
		B *string
	}

	occWildMany     struct{ Sequence []occWildManyItem }
	occWildManyItem struct {
		A   string
		Any []anyElement
	}

	occPick     struct{ Choice []occPickItem }
	occPickItem interface{ isOccPickItem() }
	occPickA    struct{ A string }
	occPickAny  struct{ Any anyElement }

	occInner       struct{ Sequence []occInnerItem }
	occInnerItem   struct{ Choice occInnerChoice }
	occInnerChoice interface{ isOccInnerChoice() }
	occInnerA      struct{ A string }
	occInnerB      struct{ B string }

	occFixed     struct{ Choice []occFixedItem }
	occFixedItem interface{ isOccFixedItem() }
	occFixedA    struct{ A []string }
	occFixedB    struct{ B string }
)

func (occChoiceA) isOccChoiceItem() {}
func (occChoiceB) isOccChoiceItem() {}
func (occPickA) isOccPickItem()     {}
func (occPickAny) isOccPickItem()   {}
func (occInnerA) isOccInnerChoice() {}
func (occInnerB) isOccInnerChoice() {}
func (occFixedA) isOccFixedItem()   {}
func (occFixedB) isOccFixedItem()   {}

// The branch sets of the occurrence types' interfaces.
var (
	occChoiceBranches = WithBranches[occChoiceItem](occChoiceA{}, occChoiceB{})
	occPickBranches   = WithBranches[occPickItem](occPickA{}, occPickAny{})
	occInnerBranches  = WithBranches[occInnerChoice](occInnerA{}, occInnerB{})
	occFixedBranches  = WithBranches[occFixedItem](occFixedA{}, occFixedB{})
)

// TestOccurrenceTypesAreCodegens pins the types above to those codegen emits
// for each occurrence schema, in shape, so that the rows decoding into them
// decode into codegen's types.
func TestOccurrenceTypesAreCodegens(t *testing.T) {
	branch := func(iface, name, field string) string {
		return "type " + name + " struct {\n\t" + field + "\n}\n\nfunc (" + name + ") is" + iface + "() {}\n"
	}
	seqAB := []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA string\n\tB string\n}\n"}
	choiceAB := []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}
	rows := []struct {
		decls string
		want  []string
	}{
		{occSeqSchema, seqAB},
		{occRefSchema, seqAB},
		{occChoiceSchema, choiceAB},
		{occEmptyBranchSchema, choiceAB},
		{occOptSchema, []string{"type R struct {\n\tChoice RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}},
		{occWildSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA   string\n\tAny AnyElement\n}\n"}},
		{occPtrSchema, []string{"type R struct {\n\tSequence *RSequence\n}\n", "type RSequence struct {\n\tA string\n}\n"}},
		{occPartSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA string\n\tB *string\n}\n"}},
		{occWildManySchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA   string\n\tAny []AnyElement\n}\n"}},
		{occPickSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceAny", "Any AnyElement")}},
		{occInnerSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tChoice RSequenceChoice\n}\n",
			branch("RSequenceChoice", "RSequenceChoiceA", "A string"), branch("RSequenceChoice", "RSequenceChoiceB", "B string")}},
		{occFixedSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A []string"), branch("RChoice", "RChoiceB", "B string")}},
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

// TestDecodeOccurrence pins the occurrence lift: a content sequence or choice
// occurring other than exactly once that codegen neither splices nor
// flattens decodes into its occurrence field of codegen's shape, its items
// opened by the positional rule Decode's doc states, each child in the field
// or branch of the particle the Matcher attributes it to (§3.8.4.2
// key-vpath). The rows of a nested choice's second alternative and of a
// particle at its {max occurs} each open an item by a different clause of
// that rule. The guard rows decode as before: a content particle codegen
// flattens into its slice of a sealed interface, and one occurring exactly
// once into its own fields. CheckDecode returns nil for each row's T.
func TestDecodeOccurrence(t *testing.T) {
	str := func(s string) *string { return &s }
	raw := func(space, local, text string) anyElement {
		return anyElement{Space: space, Local: local, Content: anyType{Text: text}}
	}
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"a repeated sequence", occSeqSchema, "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a><s:b>4</s:b>", into[occSeq](),
			occSeq{Sequence: []occSeqItem{{A: "1", B: "2"}, {A: "3", B: "4"}}}},
		{"a repeated choice", occChoiceSchema, "<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b>", into[occChoice](occChoiceBranches),
			occChoice{Choice: []occChoiceItem{occChoiceA{A: "1"}, occChoiceB{B: "2"}, occChoiceB{B: "3"}}}},
		{"an optional choice, present", occOptSchema, "<s:b>1</s:b>", into[occOpt](occChoiceBranches), occOpt{Choice: occChoiceB{B: "1"}}},
		{"an optional choice, absent", occOptSchema, "", into[occOpt](occChoiceBranches), occOpt{}},
		{"a <group ref> occurring twice", occRefSchema, "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a><s:b>4</s:b>", into[occSeq](),
			occSeq{Sequence: []occSeqItem{{A: "1", B: "2"}, {A: "3", B: "4"}}}},
		{"a wildcard taking a child of an element particle's name", occWildSchema, "<s:a>1</s:a><s:z>2</s:z><s:a>3</s:a><s:a>4</s:a>", into[occWild](),
			occWild{Sequence: []occWildItem{{A: "1", Any: raw("urn:s", "z", "2")}, {A: "3", Any: raw("urn:s", "a", "4")}}}},
		{"an optional sequence, present", occPtrSchema, "<s:a>1</s:a>", into[occPtr](), occPtr{Sequence: &occPtrItem{A: "1"}}},
		{"an optional sequence, absent", occPtrSchema, "", into[occPtr](), occPtr{}},
		{"a partition by a required leading particle", occPartSchema, "<s:a>1</s:a><s:a>2</s:a><s:b>3</s:b>", into[occPart](),
			occPart{Sequence: []occPartItem{{A: "1"}, {A: "2", B: str("3")}}}},
		{"a choice counting only occurrences that match something", occEmptyBranchSchema, "<s:a>1</s:a><s:b>2</s:b>", into[occChoice](occChoiceBranches),
			occChoice{Choice: []occChoiceItem{occChoiceA{A: "1"}, occChoiceB{B: "2"}}}},
		{"an element particle taking a child before a repeated wildcard", occWildManySchema, "<s:a>1</s:a><s:a>2</s:a>", into[occWildMany](),
			occWildMany{Sequence: []occWildManyItem{{A: "1"}, {A: "2"}}}},
		{"a repeated choice with a wildcard alternative", occPickSchema, `<s:a>1</s:a><o:w xmlns:o="urn:o">2</o:w><s:a>3</s:a>`, into[occPick](occPickBranches),
			occPick{Choice: []occPickItem{occPickA{A: "1"}, occPickAny{Any: raw("urn:o", "w", "2")}, occPickA{A: "3"}}}},
		{"a second alternative of a nested choice opening the next item", occInnerSchema, "<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b>", into[occInner](occInnerBranches),
			occInner{Sequence: []occInnerItem{{Choice: occInnerA{A: "1"}}, {Choice: occInnerB{B: "2"}}, {Choice: occInnerB{B: "3"}}}}},
		{"a particle at its {max occurs} opening the next item", occFixedSchema, "<s:a>1</s:a><s:a>2</s:a><s:a>3</s:a><s:a>4</s:a><s:b>5</s:b>", into[occFixed](occFixedBranches),
			occFixed{Choice: []occFixedItem{occFixedA{A: []string{"1", "2"}}, occFixedA{A: []string{"3", "4"}}, occFixedB{B: "5"}}}},
		{"guard: a repeated emptiable sequence codegen flattens", flatSeqSchema, "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a>", into[flatSeq](flatSeqBranches),
			flatSeq{Sequence: []flatSeqItem{flatSeqA{A: "1"}, flatSeqB{B: "2"}, flatSeqA{A: "3"}}}},
		{"guard: an emptiable sequence occurring once", flatOnceSchema, "<s:a>1</s:a>", into[flatOnce](), flatOnce{A: str("1")}},
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

// TestDecodeOccurrenceItemDeclines pins item 5 inside an occurrence: a
// construct of one occurrence's type declines by its own construct, here a
// choice's sequence alternative occurring exactly twice, which codegen
// neither splices nor flattens and which leaves the choice unflattened, since
// it does not repeat, constructNestedGroup, never by a construct of the
// occurrence itself, at Decode and CheckDecode alike.
func TestDecodeOccurrenceItemDeclines(t *testing.T) {
	schema := schemaOf(t, root(`<xs:choice maxOccurs="unbounded"><xs:sequence minOccurs="2" maxOccurs="2"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence><xs:element name="c" type="xs:string"/></xs:choice>`))
	data := instanceOf("", "<s:a>1</s:a><s:a>2</s:a><s:c>3</s:c>")
	mustAccept(t, schema, data)
	_, err := Decode[struct{}](schema, strict.New(), data)
	assertDecline(t, err, constructNestedGroup, 1, 1)
	var de *DeclineError
	if err := CheckDecode[struct{}](schema, rootName); !errors.As(err, &de) || de.Construct() != (Construct{constructNestedGroup}) {
		t.Errorf("CheckDecode = %v, want a *DeclineError of %v", err, Construct{constructNestedGroup})
	}
}
