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
// nested choice; occFixedSchema's choice(a{2,2}, b) repeated; occTwoSchema's
// sequence(a, b) occurring 1 to 2 times; occOptTwoSchema's sequence(a, b)
// occurring 0 to 2 times; occSkipSchema's sequence(a, any*) occurring at
// least twice, its wildcard ##any with {process contents} skip; and the
// choices occurring 2 to 3 times, occTwoThreeSchema's choice(a?, b),
// occTwoThreeReqSchema's choice(a, b), occGroupAltSchema's choice(sequence(a)?,
// b), occWildAltSchema's choice(a?, any ##other lax) and occWildOptSchema's
// choice(a, any? ##other lax).
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
	occTwoSchema         = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="2"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	occOptTwoSchema      = `<xs:element name="r"><xs:complexType><xs:sequence minOccurs="0" maxOccurs="2"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	occSkipSchema        = `<xs:element name="r"><xs:complexType><xs:sequence minOccurs="2" maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:any minOccurs="0" maxOccurs="unbounded" processContents="skip"/></xs:sequence></xs:complexType></xs:element>`
	occTwoThreeSchema    = `<xs:element name="r"><xs:complexType><xs:choice minOccurs="2" maxOccurs="3"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	occTwoThreeReqSchema = `<xs:element name="r"><xs:complexType><xs:choice minOccurs="2" maxOccurs="3"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	occGroupAltSchema    = `<xs:element name="r"><xs:complexType><xs:choice minOccurs="2" maxOccurs="3"><xs:sequence minOccurs="0"><xs:element name="a" type="xs:string"/></xs:sequence><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	occWildAltSchema     = `<xs:element name="r"><xs:complexType><xs:choice minOccurs="2" maxOccurs="3"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:any namespace="##other" processContents="lax"/></xs:choice></xs:complexType></xs:element>`
	occWildOptSchema     = `<xs:element name="r"><xs:complexType><xs:choice minOccurs="2" maxOccurs="3"><xs:element name="a" type="xs:string"/><xs:any namespace="##other" processContents="lax" minOccurs="0"/></xs:choice></xs:complexType></xs:element>`
)

// The types codegen emits for the occurrence schemas' s:r, in shape, which
// TestOccurrenceTypesAreCodegens pins: occSeq for occSeqSchema, occRefSchema,
// occTwoSchema and occOptTwoSchema, occChoice for occChoiceSchema,
// occEmptyBranchSchema, occTwoThreeSchema and occTwoThreeReqSchema, occOpt
// for occOptSchema, over occChoice's interface, occWildMany for
// occWildManySchema and occSkipSchema, occPick for occPickSchema,
// occWildAltSchema and occWildOptSchema, and occWild, occPtr, occPart,
// occInner, occFixed and occGroupAlt for the schemas so named.
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

	occGroupAlt     struct{ Choice []occGroupAltItem }
	occGroupAltItem interface{ isOccGroupAltItem() }
	occGroupAltSeq  struct{ A string }
	occGroupAltB    struct{ B string }
)

func (occChoiceA) isOccChoiceItem()       {}
func (occChoiceB) isOccChoiceItem()       {}
func (occPickA) isOccPickItem()           {}
func (occPickAny) isOccPickItem()         {}
func (occInnerA) isOccInnerChoice()       {}
func (occInnerB) isOccInnerChoice()       {}
func (occFixedA) isOccFixedItem()         {}
func (occFixedB) isOccFixedItem()         {}
func (occGroupAltSeq) isOccGroupAltItem() {}
func (occGroupAltB) isOccGroupAltItem()   {}

// The branch sets of the occurrence types' interfaces.
var (
	occChoiceBranches   = WithBranches[occChoiceItem](occChoiceA{}, occChoiceB{})
	occPickBranches     = WithBranches[occPickItem](occPickA{}, occPickAny{})
	occInnerBranches    = WithBranches[occInnerChoice](occInnerA{}, occInnerB{})
	occFixedBranches    = WithBranches[occFixedItem](occFixedA{}, occFixedB{})
	occGroupAltBranches = WithBranches[occGroupAltItem](occGroupAltSeq{}, occGroupAltB{})
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
		{occTwoSchema, seqAB},
		{occOptTwoSchema, seqAB},
		{occChoiceSchema, choiceAB},
		{occEmptyBranchSchema, choiceAB},
		{occTwoThreeSchema, choiceAB},
		{occTwoThreeReqSchema, choiceAB},
		{occGroupAltSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceSequence", "A string"), branch("RChoice", "RChoiceB", "B string")}},
		{occWildAltSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceAny", "Any AnyElement")}},
		{occWildOptSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceAny", "Any AnyElement")}},
		{occOptSchema, []string{"type R struct {\n\tChoice RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}},
		{occWildSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA   string\n\tAny AnyElement\n}\n"}},
		{occSkipSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA   string\n\tAny []AnyElement\n}\n"}},
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
			if !reflect.DeepEqual(unscoped(got), row.want) {
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

// TestAppendOccurrence pins the occurrence field "# What encodes" admits: a
// content sequence or choice codegen holds in an occurrence field writes each
// item the field holds, in slice order, as one occurrence of its model group,
// its particles in order (§3.9.4.3 cvc-accept clause 3): a []S of one and of
// two items, the nil []S of a sequence with {min occurs} 0, a []I of an
// unbounded choice and of one whose alternative occurs exactly twice, so that
// two items of it write four elements of one name, a *S and a bare I, nil
// meaning absent, a <group ref>, and two items of a sequence with {min
// occurs} 2, a skip wildcard's element in the first, under one attribution
// check across both. Below its {min occurs}, the []I of a choice an
// element, sequence or wildcard alternative of which has {min occurs} 0
// (Append's doc, item 6): one item of each such choice occurring 2 to 3
// times, the two over a wildcard under the content's attribution check, and
// none of one occurring 1 to unbounded times, each the slice Decode returns.
// Validate accepts each document and Decode returns the value.
func TestAppendOccurrence(t *testing.T) {
	const r = `<r xmlns="urn:s">`
	for _, row := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"a sequence occurring 1 to 2 times, one occurrence", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occTwoSchema), rootName, occSeq{Sequence: []occSeqItem{{A: "1", B: "2"}}}, r+`<a>1</a><b>2</b></r>`, deepEqual[occSeq])
		}},
		{"a sequence occurring 1 to 2 times, two occurrences", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occTwoSchema), rootName, occSeq{Sequence: []occSeqItem{{A: "1", B: "2"}, {A: "3", B: "4"}}}, r+`<a>1</a><b>2</b><a>3</a><b>4</b></r>`, deepEqual[occSeq])
		}},
		{"a sequence occurring 0 to 2 times, zero occurrences", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occOptTwoSchema), rootName, occSeq{}, `<r xmlns="urn:s"/>`, deepEqual[occSeq])
		}},
		{"an unbounded choice", func(t *testing.T) {
			v := occChoice{Choice: []occChoiceItem{occChoiceA{A: "1"}, occChoiceB{B: "2"}, occChoiceB{B: "3"}}}
			roundTrip(t, schemaOf(t, occChoiceSchema), rootName, v, r+`<a>1</a><b>2</b><b>3</b></r>`, deepEqual[occChoice], occChoiceBranches)
		}},
		{"an unbounded choice of a{2,2} and b", func(t *testing.T) {
			v := occFixed{Choice: []occFixedItem{occFixedA{A: []string{"1", "2"}}, occFixedA{A: []string{"3", "4"}}}}
			roundTrip(t, schemaOf(t, occFixedSchema), rootName, v, r+`<a>1</a><a>2</a><a>3</a><a>4</a></r>`, deepEqual[occFixed], occFixedBranches)
		}},
		{"an optional choice, absent", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occOptSchema), rootName, occOpt{}, `<r xmlns="urn:s"/>`, deepEqual[occOpt], occChoiceBranches)
		}},
		{"an optional choice, present", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occOptSchema), rootName, occOpt{Choice: occChoiceB{B: "1"}}, r+`<b>1</b></r>`, deepEqual[occOpt], occChoiceBranches)
		}},
		{"an optional sequence, absent", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occPtrSchema), rootName, occPtr{}, `<r xmlns="urn:s"/>`, deepEqual[occPtr])
		}},
		{"an optional sequence, present", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occPtrSchema), rootName, occPtr{Sequence: &occPtrItem{A: "1"}}, r+`<a>1</a></r>`, deepEqual[occPtr])
		}},
		{"a <group ref> occurring twice", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occRefSchema), rootName, occSeq{Sequence: []occSeqItem{{A: "1", B: "2"}, {A: "3", B: "4"}}}, r+`<a>1</a><b>2</b><a>3</a><b>4</b></r>`, deepEqual[occSeq])
		}},
		{"a skip wildcard in a sequence occurring at least twice", func(t *testing.T) {
			v := occWildMany{Sequence: []occWildManyItem{{A: "1", Any: []anyElement{{Space: "urn:o", Local: "w"}}}, {A: "2"}}}
			roundTrip(t, schemaOf(t, occSkipSchema), rootName, v, r+`<a>1</a><p1:w xmlns="" xmlns:p1="urn:o"/><a>2</a></r>`, deepEqual[occWildMany])
		}},
		{"one item of a choice occurring 2 to 3 times, an element alternative emptiable", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occTwoThreeSchema), rootName, occChoice{Choice: []occChoiceItem{occChoiceB{B: "1"}}}, r+`<b>1</b></r>`, deepEqual[occChoice], occChoiceBranches)
		}},
		{"zero items of a choice occurring 1 to unbounded times, an element alternative emptiable", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occEmptyBranchSchema), rootName, occChoice{}, `<r xmlns="urn:s"/>`, deepEqual[occChoice], occChoiceBranches)
		}},
		{"one item of a choice occurring 2 to 3 times, a sequence alternative emptiable", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occGroupAltSchema), rootName, occGroupAlt{Choice: []occGroupAltItem{occGroupAltB{B: "1"}}}, r+`<b>1</b></r>`, deepEqual[occGroupAlt], occGroupAltBranches)
		}},
		{"one item of a choice occurring 2 to 3 times, an element alternative emptiable beside a wildcard", func(t *testing.T) {
			v := occPick{Choice: []occPickItem{occPickAny{Any: anyElement{Space: "urn:o", Local: "w"}}}}
			roundTrip(t, schemaOf(t, occWildAltSchema), rootName, v, r+`<p1:w xmlns="" xmlns:p1="urn:o"/></r>`, deepEqual[occPick], occPickBranches)
		}},
		{"one item of a choice occurring 2 to 3 times, its wildcard alternative emptiable", func(t *testing.T) {
			roundTrip(t, schemaOf(t, occWildOptSchema), rootName, occPick{Choice: []occPickItem{occPickA{A: "1"}}}, r+`<a>1</a></r>`, deepEqual[occPick], occPickBranches)
		}},
	} {
		t.Run(row.name, row.run)
	}
}

// TestAppendOccurrenceDeclineSubject pins the subject item 4 of Append's doc
// names for a particle of a model group held in an occurrence field, here a
// wildcard particle rebuilt vacuous (vacatedSchema), in occWildSchema's and
// occPickSchema's content: a sequence's particle by its place, a choice's by
// its alternative, as a content choice occurring once names its own.
func TestAppendOccurrenceDeclineSubject(t *testing.T) {
	const wild = "codec does not encode a particle with {max occurs} 0: "
	_, err := appendChecked(t, nil, vacatedSchema(t, `<xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:any processContents="lax"/></xs:sequence>`), strict.New(), rootName, occWild{})
	_ = asDecline(t, err, wild+"particle 2 of the content {particle} of the {type definition} of element {urn:s}r declared at ")
	_, err = appendChecked(t, nil, vacatedSchema(t, `<xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:any namespace="##other" processContents="lax"/></xs:choice>`), strict.New(), rootName, occPick{}, occPickBranches)
	_ = asDecline(t, err, wild+"alternative 2 of the content {particle} of the {type definition} of element {urn:s}r declared at ")
}

// TestAppendOccurrenceInvalid pins item 6 of Append's doc for an occurrence
// field: a slice whose length lies outside the content particle's occurrence,
// zero items of a sequence with {min occurs} 1 among them (cvc-accept clause
// 3.1), one item and none of a choice occurring 2 to 3 times none of whose
// alternatives is emptiable, and four items of one occurring 2 to 3 times one
// of whose alternatives is, the bound it waives being the lower alone; a nil
// item of a choice's slice, whether or not one of the choice's alternatives
// is emptiable, since Decode returns no item that holds no child; and a skip
// wildcard's element of an element particle's name, which the content
// model's Matcher attributes to that particle, opening the next occurrence.
// None carries an xsderr rule.
func TestAppendOccurrenceInvalid(t *testing.T) {
	wild := occWildMany{Sequence: []occWildManyItem{{A: "1", Any: []anyElement{{Space: "urn:s", Local: "a"}}}, {A: "2"}}}
	for _, row := range []struct {
		name   string
		run    func(t *testing.T) error
		prefix string
		want   []string
	}{
		{"zero occurrences of a sequence with {min occurs} 1", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, occTwoSchema), strict.New(), rootName, occSeq{})
			return err
		}, "codec: Append: occSeq.Sequence holds 0 items, outside the occurrence 1..2 of the sequence content particle of the {type definition} of element {urn:s}r declared at ", nil},
		{"three occurrences of a sequence with {max occurs} 2", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, occTwoSchema), strict.New(), rootName, occSeq{Sequence: make([]occSeqItem, 3)})
			return err
		}, "codec: Append: occSeq.Sequence holds 3 items, outside the occurrence 1..2 of the sequence content particle of ", nil},
		{"one item of a choice occurring 2 to 3 times none of whose alternatives is emptiable", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, occTwoThreeReqSchema), strict.New(), rootName, occChoice{Choice: []occChoiceItem{occChoiceB{B: "1"}}}, occChoiceBranches)
			return err
		}, "codec: Append: occChoice.Choice holds 1 items, outside the occurrence 2..3 of the choice content particle of the {type definition} of element {urn:s}r declared at ", nil},
		{"zero items of a choice occurring 2 to 3 times none of whose alternatives is emptiable", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, occTwoThreeReqSchema), strict.New(), rootName, occChoice{}, occChoiceBranches)
			return err
		}, "codec: Append: occChoice.Choice holds 0 items, outside the occurrence 2..3 of the choice content particle of ", nil},
		{"four items of a choice occurring 2 to 3 times an alternative of which is emptiable", func(t *testing.T) error {
			v := occChoice{Choice: []occChoiceItem{occChoiceB{B: "1"}, occChoiceB{B: "2"}, occChoiceB{B: "3"}, occChoiceB{B: "4"}}}
			_, err := appendChecked(t, nil, schemaOf(t, occTwoThreeSchema), strict.New(), rootName, v, occChoiceBranches)
			return err
		}, "codec: Append: occChoice.Choice holds 4 items, outside the occurrence 2..3 of the choice content particle of the {type definition} of element {urn:s}r declared at ", nil},
		{"a nil item of a choice's slice", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, occChoiceSchema), strict.New(), rootName, occChoice{Choice: []occChoiceItem{occChoiceA{A: "1"}, nil}}, occChoiceBranches)
			return err
		}, "codec: Append: occChoice.Choice[1] is nil, an item holding no child element, which Decode never returns in the occurrence field of the content choice of the {type definition} of element {urn:s}r declared at ", nil},
		{"a nil item of a choice's slice one of whose element alternatives is emptiable", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, occEmptyBranchSchema), strict.New(), rootName, occChoice{Choice: []occChoiceItem{occChoiceB{B: "1"}, nil}}, occChoiceBranches)
			return err
		}, "codec: Append: occChoice.Choice[1] is nil, an item holding no child element, which Decode never returns in the occurrence field of the content choice of the {type definition} of element {urn:s}r declared at ", nil},
		{"a wildcard's element attributed to an element particle", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, occSkipSchema), strict.New(), rootName, wild)
			return err
		}, "codec: Append: occWildMany.Sequence[0].Any[0] holds element {urn:s}a, which the content model of the {type definition} of element {urn:s}r declared at ", []string{"attributes to a particle other than the one whose field holds it"}},
	} {
		t.Run(row.name, func(t *testing.T) {
			invalidChoice(t, row.run(t), row.prefix, row.want...)
		})
	}
}
