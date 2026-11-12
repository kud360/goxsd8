package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// The nested-occurrence schemas, each declaring s:r, whose content holds a
// model group particle nested in a sequence or as a choice's alternative that
// codegen neither splices nor flattens and that is no choice occurring
// exactly once, so codegen gives it an occurrence field at its particle's
// place (codegen/doc.go "# Particles"): noRepChoiceSchema's sequence(a,
// choice(b, c) unbounded); noOptChoiceSchema's sequence(a, choice(b, c)?);
// noOptSeqSchema's sequence(a, sequence(b)?), b not emptiable;
// noRepSeqSchema's sequence(a, sequence(b, c){1,2}); noAltSchema's the
// content choice(a, sequence(b, c){1,2}); noRefSchema's sequence(a, <group
// ref>{1,2}), the group sequence(b, c); noWildSchema's sequence(a, sequence(b,
// any over ##other){1,2}); noInnerSchema's sequence(a, sequence(choice(b, c),
// d){1,2}), a nested choice in the item; noOnceAltSchema's sequence(a,
// choice(b, sequence(c, d){1,2})), the occurrence a once-choice's
// alternative; noTwoSchema's sequence(sequence(a, b){1,2}, sequence(c,
// d){1,2}); noRecSchema's s:r of type t, sequence(sequence(b?, c){1,2}), b of
// type t; noOnlySchema's sequence(choice(a){1,2}); noExtSchema's an extension
// of sequence(a){1,2} adding sequence(b); noAltFirstSchema's the content
// choice(sequence(a){1,2}, b); noRefChoiceSchema's sequence(<group ref>{1,2}),
// the group choice(a); and noSeqOneSchema's sequence(a, sequence(b){1,2}).
const (
	noRepChoiceSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:choice maxOccurs="unbounded">` + cgElB + cgElC + `</xs:choice></xs:sequence></xs:complexType></xs:element>`
	noOptChoiceSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:choice minOccurs="0">` + cgElB + cgElC + `</xs:choice></xs:sequence></xs:complexType></xs:element>`
	noOptSeqSchema    = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:sequence minOccurs="0">` + cgElB + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	noRepSeqSchema    = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:sequence maxOccurs="2">` + cgElB + cgElC + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	noAltSchema       = `<xs:element name="r"><xs:complexType><xs:choice>` + cgElA + `<xs:sequence maxOccurs="2">` + cgElB + cgElC + `</xs:sequence></xs:choice></xs:complexType></xs:element>`
	noRefSchema       = `<xs:group name="g"><xs:sequence>` + cgElB + cgElC + `</xs:sequence></xs:group>` +
		`<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:group ref="s:g" maxOccurs="2"/></xs:sequence></xs:complexType></xs:element>`
	noWildSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:sequence maxOccurs="2">` + cgElB +
		`<xs:any namespace="##other" processContents="lax"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	noInnerSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:sequence maxOccurs="2"><xs:choice>` + cgElB + cgElC + `</xs:choice>` +
		`<xs:element name="d" type="xs:string"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	noOnceAltSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:choice>` + cgElB + `<xs:sequence maxOccurs="2">` + cgElC +
		`<xs:element name="d" type="xs:string"/></xs:sequence></xs:choice></xs:sequence></xs:complexType></xs:element>`
	noTwoSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:sequence maxOccurs="2">` + cgElA + cgElB + `</xs:sequence><xs:sequence maxOccurs="2">` + cgElC +
		`<xs:element name="d" type="xs:string"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	noRecSchema = `<xs:complexType name="t"><xs:sequence><xs:sequence maxOccurs="2"><xs:element name="b" type="s:t" minOccurs="0"/>` + cgElC +
		`</xs:sequence></xs:sequence></xs:complexType><xs:element name="r" type="s:t"/>`
	noOnlySchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:choice maxOccurs="2">` + cgElA + `</xs:choice></xs:sequence></xs:complexType></xs:element>`
	noExtSchema  = `<xs:complexType name="t"><xs:sequence maxOccurs="2">` + cgElA + `</xs:sequence></xs:complexType>` +
		`<xs:complexType name="t2"><xs:complexContent><xs:extension base="s:t"><xs:sequence>` + cgElB + `</xs:sequence></xs:extension></xs:complexContent></xs:complexType><xs:element name="r" type="s:t2"/>`
	noAltFirstSchema  = `<xs:element name="r"><xs:complexType><xs:choice><xs:sequence maxOccurs="2">` + cgElA + `</xs:sequence>` + cgElB + `</xs:choice></xs:complexType></xs:element>`
	noRefChoiceSchema = `<xs:group name="g"><xs:choice>` + cgElA + `</xs:choice></xs:group>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:group ref="s:g" maxOccurs="2"/></xs:sequence></xs:complexType></xs:element>`
	noSeqOneSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:sequence maxOccurs="2">` + cgElB + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
)

// cgElA, cgElB and cgElC are cgEl's local elements a, b and c, as constants.
const (
	cgElA = `<xs:element name="a" type="xs:string"/>`
	cgElB = `<xs:element name="b" type="xs:string"/>`
	cgElC = `<xs:element name="c" type="xs:string"/>`
)

// The types codegen emits for the nested-occurrence schemas' s:r, in shape,
// which TestNestedOccurrenceTypesAreCodegens pins: noRepChoice, noOptChoice,
// noOptSeq, noRepSeq (noRefSchema's too), noAlt, noWild, noInner, noOnceAlt,
// noTwo, noRec, noOnly (noRefChoiceSchema's too), noExt, noAltFirst and
// noSeqOne for the schemas so named.
type (
	noBC interface{ isNoBC() }
	noB  struct{ B string }
	noC  struct{ C string }

	noRepChoice struct {
		A      string
		Choice []noBC
	}
	noOptChoice struct {
		A      string
		Choice noBC
	}

	noSeqB   struct{ B string }
	noOptSeq struct {
		A        string
		Sequence *noSeqB
	}

	noSeqBC  struct{ B, C string }
	noRepSeq struct {
		A        string
		Sequence []noSeqBC
	}

	noAlt    interface{ isNoAlt() }
	noAltA   struct{ A string }
	noAltSeq struct{ Sequence []noSeqBC }

	noWildItem struct {
		B   string
		Any anyElement
	}
	noWild struct {
		A        string
		Sequence []noWildItem
	}

	noInnerItem struct {
		Choice noBC
		D      string
	}
	noInner struct {
		A        string
		Sequence []noInnerItem
	}

	noSeqCD     struct{ C, D string }
	noOnceAltI  interface{ isNoOnceAlt() }
	noOnceAltB  struct{ B string }
	noOnceAltSq struct{ Sequence []noSeqCD }
	noOnceAlt   struct {
		A      string
		Choice noOnceAltI
	}

	noSeqAB struct{ A, B string }
	noTwo   struct {
		Sequence  []noSeqAB
		Sequence2 []noSeqCD
	}

	noRec     struct{ Sequence []noRecItem }
	noRecItem struct {
		B *noRec
		C string
	}

	noAI   interface{ isNoA() }
	noA    struct{ A string }
	noOnly struct{ Choice []noAI }

	noSeqA struct{ A string }
	noExt  struct {
		Sequence []noSeqA
		B        string
	}

	noAltFirst    interface{ isNoAltFirst() }
	noAltFirstSeq struct{ Sequence []noSeqA }
	noAltFirstB   struct{ B string }

	noSeqOne struct {
		A        string
		Sequence []noSeqB
	}
)

func (noB) isNoBC()                 {}
func (noC) isNoBC()                 {}
func (noAltA) isNoAlt()             {}
func (noAltSeq) isNoAlt()           {}
func (noOnceAltB) isNoOnceAlt()     {}
func (noOnceAltSq) isNoOnceAlt()    {}
func (noA) isNoA()                  {}
func (noAltFirstSeq) isNoAltFirst() {}
func (noAltFirstB) isNoAltFirst()   {}

// The branch sets of the nested-occurrence interfaces.
var (
	noBCBranches       = WithBranches[noBC](noB{}, noC{})
	noAltBranches      = WithBranches[noAlt](noAltA{}, noAltSeq{})
	noOnceAltBranches  = WithBranches[noOnceAltI](noOnceAltB{}, noOnceAltSq{})
	noABranches        = WithBranches[noAI](noA{})
	noAltFirstBranches = WithBranches[noAltFirst](noAltFirstSeq{}, noAltFirstB{})
)

// TestNestedOccurrenceTypesAreCodegens pins the types above to those codegen
// emits for each nested-occurrence schema, in shape, so that the rows decoding
// into them decode into codegen's types.
func TestNestedOccurrenceTypesAreCodegens(t *testing.T) {
	branch := func(iface, name, field string) string {
		return "type " + name + " struct {\n\t" + field + "\n}\n\nfunc (" + name + ") is" + iface + "() {}\n"
	}
	bc := []string{branch("RChoice", "RChoiceB", "B string"), branch("RChoice", "RChoiceC", "C string")}
	repSeq := []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tB string\n\tC string\n}\n"}
	only := []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string")}
	rows := []struct {
		decls string
		want  []string
	}{
		{noRepChoiceSchema, append([]string{"type R struct {\n\tA      string\n\tChoice []RChoice\n}\n"}, bc...)},
		{noOptChoiceSchema, append([]string{"type R struct {\n\tA      string\n\tChoice RChoice\n}\n"}, bc...)},
		{noOptSeqSchema, []string{"type R struct {\n\tA        string\n\tSequence *RSequence\n}\n", "type RSequence struct {\n\tB string\n}\n"}},
		{noRepSeqSchema, repSeq},
		{noRefSchema, repSeq},
		{noAltSchema, []string{"type R interface {\n\tisR()\n}\n", branch("R", "RA", "A string"), branch("R", "RSequence", "Sequence []RSequence2"), "type RSequence2 struct {\n\tB string\n\tC string\n}\n"}},
		{noWildSchema, []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tB   string\n\tAny AnyElement\n}\n"}},
		{noInnerSchema, []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tChoice RSequenceChoice\n\tD      string\n}\n",
			branch("RSequenceChoice", "RSequenceChoiceB", "B string"), branch("RSequenceChoice", "RSequenceChoiceC", "C string")}},
		{noOnceAltSchema, []string{"type R struct {\n\tA      string\n\tChoice RChoice\n}\n", branch("RChoice", "RChoiceB", "B string"),
			branch("RChoice", "RChoiceSequence", "Sequence []RChoiceSequence2"), "type RChoiceSequence2 struct {\n\tC string\n\tD string\n}\n"}},
		{noTwoSchema, []string{"type R struct {\n\tSequence  []RSequence\n\tSequence2 []RSequence2\n}\n", "type RSequence struct {\n\tA string\n\tB string\n}\n", "type RSequence2 struct {\n\tC string\n\tD string\n}\n"}},
		{noRecSchema, []string{"type R struct {\n\tSequence []TSequence\n}\n", "type TSequence struct {\n\tB *T\n\tC string\n}\n", "type T struct {\n\tSequence []TSequence\n}\n"}},
		{noOnlySchema, only},
		{noRefChoiceSchema, only},
		{noExtSchema, []string{"type R struct {\n\tSequence []T2Sequence\n\tB        string\n}\n", "type T2Sequence struct {\n\tA string\n}\n"}},
		{noAltFirstSchema, []string{"type R interface {\n\tisR()\n}\n", branch("R", "RSequence", "Sequence []RSequence2"), "type RSequence2 struct {\n\tA string\n}\n", branch("R", "RB", "B string")}},
		{noSeqOneSchema, []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tB string\n}\n"}},
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

// TestDecodeNestedOccurrence pins the nested occurrence lift: a model group
// particle nested in a sequence or as a choice's alternative that codegen
// neither splices nor flattens and that is no choice occurring exactly once
// decodes into its occurrence field at its particle's place, a slice for
// {max occurs} above 1, a pointer for a sequence occurring 0..1 and the bare
// interface for a choice occurring 0..1, its items opened by the positional
// rule Decode's doc states, each child an element particle's by its expanded
// name or a wildcard's by its Matcher position, the partition read below the
// nested particle's position. The recursive row's inner s:b fills its own
// occurrence field between two children of the outer one's first item, so it
// fails where the partition is kept anywhere but on the fill's own stack.
// CheckDecode returns nil for each row's T.
func TestDecodeNestedOccurrence(t *testing.T) {
	w := anyElement{Space: "urn:o", Local: "w", Content: anyType{Text: "w"}}
	const ow = `<o:w xmlns:o="urn:o">w</o:w>`
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"a nested repeated choice", noRepChoiceSchema, "<s:a>1</s:a><s:c>2</s:c><s:b>3</s:b>", into[noRepChoice](noBCBranches),
			noRepChoice{A: "1", Choice: []noBC{noC{C: "2"}, noB{B: "3"}}}},
		{"a nested optional choice, present", noOptChoiceSchema, "<s:a>1</s:a><s:b>2</s:b>", into[noOptChoice](noBCBranches), noOptChoice{A: "1", Choice: noB{B: "2"}}},
		{"a nested optional choice, absent", noOptChoiceSchema, "<s:a>1</s:a>", into[noOptChoice](noBCBranches), noOptChoice{A: "1"}},
		{"a nested optional sequence over a group that cannot match nothing, present", noOptSeqSchema, "<s:a>1</s:a><s:b>2</s:b>", into[noOptSeq](),
			noOptSeq{A: "1", Sequence: &noSeqB{B: "2"}}},
		{"a nested optional sequence over a group that cannot match nothing, absent", noOptSeqSchema, "<s:a>1</s:a>", into[noOptSeq](), noOptSeq{A: "1"}},
		{"a nested repeated sequence", noRepSeqSchema, "<s:a>1</s:a><s:b>2</s:b><s:c>3</s:c><s:b>4</s:b><s:c>5</s:c>", into[noRepSeq](),
			noRepSeq{A: "1", Sequence: []noSeqBC{{B: "2", C: "3"}, {B: "4", C: "5"}}}},
		{"a repeated sequence alternative of the content choice", noAltSchema, "<s:b>1</s:b><s:c>2</s:c><s:b>3</s:b><s:c>4</s:c>", into[noAlt](noAltBranches),
			noAltSeq{Sequence: []noSeqBC{{B: "1", C: "2"}, {B: "3", C: "4"}}}},
		{"the content choice's element alternative beside a repeated sequence alternative", noAltSchema, "<s:a>1</s:a>", into[noAlt](noAltBranches), noAltA{A: "1"}},
		{"a nested <group ref> occurring twice", noRefSchema, "<s:a>1</s:a><s:b>2</s:b><s:c>3</s:c><s:b>4</s:b><s:c>5</s:c>", into[noRepSeq](),
			noRepSeq{A: "1", Sequence: []noSeqBC{{B: "2", C: "3"}, {B: "4", C: "5"}}}},
		{"a wildcard in a nested occurrence's item", noWildSchema, "<s:a>1</s:a><s:b>2</s:b>" + ow + "<s:b>3</s:b>" + ow, into[noWild](),
			noWild{A: "1", Sequence: []noWildItem{{B: "2", Any: w}, {B: "3", Any: w}}}},
		{"a nested choice in a nested occurrence's item", noInnerSchema, "<s:a>1</s:a><s:b>2</s:b><s:d>3</s:d><s:c>4</s:c><s:d>5</s:d>", into[noInner](noBCBranches),
			noInner{A: "1", Sequence: []noInnerItem{{Choice: noB{B: "2"}, D: "3"}, {Choice: noC{C: "4"}, D: "5"}}}},
		{"a nested occurrence as a once-choice's alternative", noOnceAltSchema, "<s:a>1</s:a><s:c>2</s:c><s:d>3</s:d><s:c>4</s:c><s:d>5</s:d>", into[noOnceAlt](noOnceAltBranches),
			noOnceAlt{A: "1", Choice: noOnceAltSq{Sequence: []noSeqCD{{C: "2", D: "3"}, {C: "4", D: "5"}}}}},
		{"two sibling nested occurrences", noTwoSchema, "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a><s:b>4</s:b><s:c>5</s:c><s:d>6</s:d>", into[noTwo](),
			noTwo{Sequence: []noSeqAB{{A: "1", B: "2"}, {A: "3", B: "4"}}, Sequence2: []noSeqCD{{C: "5", D: "6"}}}},
		{"a recursive type filling its own nested occurrence inside an item", noRecSchema, "<s:b><s:c>1</s:c></s:b><s:c>2</s:c><s:c>3</s:c>", into[noRec](),
			noRec{Sequence: []noRecItem{{B: &noRec{Sequence: []noRecItem{{C: "1"}}}, C: "2"}, {C: "3"}}}},
		{"a nested choice occurring twice", noOnlySchema, "<s:a>1</s:a><s:a>2</s:a>", into[noOnly](noABranches), noOnly{Choice: []noAI{noA{A: "1"}, noA{A: "2"}}}},
		{"a nested <group ref> to a choice group, occurring twice", noRefChoiceSchema, "<s:a>1</s:a><s:a>2</s:a>", into[noOnly](noABranches), noOnly{Choice: []noAI{noA{A: "1"}, noA{A: "2"}}}},
		{"an extension of a repeated sequence adding a sequence", noExtSchema, "<s:a>1</s:a><s:a>2</s:a><s:b>3</s:b>", into[noExt](),
			noExt{Sequence: []noSeqA{{A: "1"}, {A: "2"}}, B: "3"}},
		{"a choice's first alternative a repeated sequence", noAltFirstSchema, "<s:a>1</s:a><s:a>2</s:a>", into[noAltFirst](noAltFirstBranches),
			noAltFirstSeq{Sequence: []noSeqA{{A: "1"}, {A: "2"}}}},
		{"a repeated nested sequence codegen does not flatten", noSeqOneSchema, "<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b>", into[noSeqOne](),
			noSeqOne{A: "1", Sequence: []noSeqB{{B: "2"}, {B: "3"}}}},
		{"guard: a nested repeated emptiable sequence codegen flattens", nfSeqSchema, "<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b>", into[nfSeq](nfSeqBranches),
			nfSeq{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}, nfSeqB{B: "3"}}}},
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

// The group-path schema: model group definition g, sequence(a, sequence(b){1,2}),
// b of an anonymous complex type, sequence(x), and top-level s:p and s:q,
// each of an anonymous type whose content is a <group ref> to g. Codegen
// emits P and Q, each with its own occurrence struct, sharing one B.
const noGroupPathSchema = `<xs:group name="g"><xs:sequence>` + cgElA + `<xs:sequence maxOccurs="2"><xs:element name="b"><xs:complexType><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:sequence></xs:group>` +
	`<xs:element name="p"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element><xs:element name="q"><xs:complexType><xs:group ref="s:g"/></xs:complexType></xs:element>`

// The types codegen emits for noGroupPathSchema's s:p and s:q, in shape:
// noGP serves either.
type (
	noGP struct {
		A        string
		Sequence []noGPItem
	}
	noGPItem struct{ B noGPB }
	noGPB    struct{ X string }
)

// TestDecodeNestedOccurrenceGroupPath pins groupPath against codegen's key
// for a type nested in a model group definition: the anonymous type of a
// local element inside a nested occurrence's item, in a group two
// declarations reference, is the one component codegen emits once, and
// Decode and CheckDecode hold it under groupPath, unchanged, for each.
func TestDecodeNestedOccurrenceGroupPath(t *testing.T) {
	schema := schemaOf(t, noGroupPathSchema)
	gen := generated(t, schema)
	for _, decl := range []string{"type P struct {\n\tA        string\n\tSequence []PSequence\n}\n", "type PSequence struct {\n\tB B\n}\n",
		"type Q struct {\n\tA        string\n\tSequence []QSequence\n}\n", "type QSequence struct {\n\tB B\n}\n", "type B struct {\n\tX string\n}\n"} {
		if !bytes.Contains(gen, []byte(decl)) {
			t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
		}
	}
	want := noGP{A: "1", Sequence: []noGPItem{{B: noGPB{X: "2"}}, {B: noGPB{X: "3"}}}}
	for _, local := range []string{"p", "q"} {
		data := []byte(`<s:` + local + ` xmlns:s="urn:s"><s:a>1</s:a><s:b><s:x>2</s:x></s:b><s:b><s:x>3</s:x></s:b></s:` + local + `>`)
		mustAccept(t, schema, data)
		got, err := Decode[noGP](schema, strict.New(), data)
		if err != nil {
			t.Fatalf("Decode(s:%s) = %v", local, err)
		}
		if !reflect.DeepEqual(*got, want) {
			t.Errorf("Decode(s:%s) = %#v, want %#v", local, *got, want)
		}
		if err := CheckDecode[noGP](schema, xsd.QName{Space: "urn:s", Local: local}); err != nil {
			t.Errorf("CheckDecode(s:%s) = %v, want nil", local, err)
		}
	}
}

// TestDecodeNestedOccurrenceDeclines pins what the nested occurrence lift
// leaves declining, Decode and CheckDecode alike, each row's prefix the
// decline's subject: a nested particle that would take an occurrence field
// inside an occurrence field's item, with {max occurs} above 1 and 0..1 alike,
// since the item's partition reads no repetition of a model group inside the
// item, as constructNestedGroup; and an element particle of a nested
// occurrence's item sharing an expanded name with a sibling particle, which
// the content model attributes by name, as constructSharedName.
func TestDecodeNestedOccurrenceDeclines(t *testing.T) {
	rows := []struct {
		name, decls, body string
		construct         construct
		prefix            string
	}{
		{"a repeated nested choice in an occurrence field's item", root(`<xs:sequence maxOccurs="2">` + cgElA + `<xs:choice maxOccurs="2">` + cgElB + `</xs:choice></xs:sequence>`),
			"<s:a>1</s:a><s:b>2</s:b>", constructNestedGroup,
			"?:1:1: codec does not decode a nested model group particle: a particle of the {type definition} of element {urn:s}r declared at"},
		{"an optional nested choice in an occurrence field's item", root(`<xs:sequence maxOccurs="2">` + cgElA + `<xs:choice minOccurs="0">` + cgElB + cgElC + `</xs:choice></xs:sequence>`),
			"<s:a>1</s:a><s:b>2</s:b>", constructNestedGroup,
			"?:1:1: codec does not decode a nested model group particle: a particle of the {type definition} of element {urn:s}r declared at"},
		{"a nested occurrence's particle sharing a sibling's expanded name", sequenceOf(cgElB + `<xs:sequence maxOccurs="2">` + cgElB + cgElC + `</xs:sequence>`),
			"<s:b>1</s:b><s:b>2</s:b><s:c>3</s:c>", constructSharedName,
			"?:1:1: codec does not decode two particles taking one expanded name, each by its own element declaration or a member of its substitution group: element {urn:s}b declared at s.xsd:1:297"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf("", row.body)
			mustAccept(t, schema, data)
			_, err := Decode[struct{}](schema, strict.New(), data)
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

// TestCheckDecodeUntrackedNestedOccurrence pins the nested occurrence's
// ContentMatcher decline and its place in item 5's order: of sequence(sequence(a,
// b){1,2}, n nillable, sequence(c{1,2}){1,21845}), the third particle,
// flattened by its repeating edge, makes the content model one
// xsd.Schema.ContentMatcher declines as too wide to track, and the nested
// occurrence declines constructUntrackedModel after its own last particle,
// before n's constructNillable.
func TestCheckDecodeUntrackedNestedOccurrence(t *testing.T) {
	schema := schemaOf(t, sequenceOf(`<xs:sequence maxOccurs="2">`+cgElA+cgElB+`</xs:sequence><xs:element name="n" type="xs:string" nillable="true"/>`+
		`<xs:sequence maxOccurs="21845"><xs:element name="c" type="xs:string" maxOccurs="2"/></xs:sequence>`))
	var de *DeclineError
	if err := CheckDecode[struct{}](schema, rootName); !errors.As(err, &de) || de.Construct() != (Construct{constructUntrackedModel}) {
		t.Errorf("CheckDecode = %v, want a *DeclineError of %v", err, Construct{constructUntrackedModel})
	}
}

// TestCheckDecodeNestedOccurrenceMismatch pins item 6 at a nested occurrence
// field: a T whose second field holds noRepSeqSchema's occurrence struct bare,
// no slice, is no type codegen emits, and the error names that field by its
// index and the shape codegen gives it.
func TestCheckDecodeNestedOccurrenceMismatch(t *testing.T) {
	type noBare struct {
		A        string
		Sequence noSeqBC
	}
	err := CheckDecode[noBare](schemaOf(t, noRepSeqSchema), rootName)
	const want = "codec: T is not the type codegen emits for the document element: field 1 of codec.noBare is Sequence codec.noSeqBC where codegen emits Sequence, []I for the type I of one occurrence of the sequence"
	if !errors.Is(err, ErrTypeMismatch) || err.Error() != want {
		t.Errorf("CheckDecode = %v, want %q", err, want)
	}
}

// TestCheckDecodeNestedMaxOccursZero pins item 5's plain error for a nested
// model group particle with {max occurs} 0, a sequence's and a choice's, which
// codegen gives no field: no schema document yields one, so the schema is
// built as TestCheckDecodeMaxOccursZero's is.
func TestCheckDecodeNestedMaxOccursZero(t *testing.T) {
	rows := []struct{ name, particle, want string }{
		{"sequence", `<xs:sequence>` + cgElA + `</xs:sequence>`,
			"codec: a sequence model group particle of the {type definition} of element {urn:s}r declared at s.xsd:1:316, a type defined at s.xsd:1:181 has {max occurs} 0, for which codegen emits no type"},
		{"choice", `<xs:choice>` + cgElA + `</xs:choice>`,
			"codec: a choice model group particle of the {type definition} of element {urn:s}r declared at s.xsd:1:312, a type defined at s.xsd:1:181 has {max occurs} 0, for which codegen emits no type"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) { checkMaxOccursZero(t, row.particle, row.want) })
	}
}
