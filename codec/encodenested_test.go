package codec

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// The nested-encode schemas, each declaring s:r: enFlatPairSchema's
// sequence(c, sequence(a?, b?){1,2}), the nested sequence flattened, its
// emptiable term repeated; enPairFirstSchema's sequence(sequence(a?,
// b?){1,2}, c), the same flattened particle ahead of an element;
// enChoiceAnySchema's sequence(choice(b, c), any over ##other), a wildcard
// after a nested choice of two alternatives; enOptOccSchema's sequence(a,
// choice(b, sequence(c, d){0,2})), noOnceAltSchema's with the occurrence
// alternative's {min occurs} 0, which its branch reads as 1;
// enItemChoiceSchema's sequence(choice(b, c)){1,2}, an occurrence field
// whose item's one member is a once-choice none of whose alternatives is
// emptiable; enItemGroupSchema's sequence(choice(a, sequence(b,
// c))){1,2}, the same with a model-group alternative of {min occurs} 1; and
// enOccChoiceSchema's sequence(choice(a?, b){2,3}, c), a nested occurrence
// field of a choice one of whose alternatives is emptiable,
// enOccChoiceReqSchema's sequence(choice(a, b){2,3}, c), none of whose is, and
// enBranchOccSchema's choice(x, sequence(choice(a?, b){2,3}, c)), the first
// held in a content choice's spliced sequence alternative.
const (
	enFlatPairSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElC + `<xs:sequence maxOccurs="2"><xs:element name="a" type="xs:string" minOccurs="0"/>` +
		`<xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
	enPairFirstSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:sequence maxOccurs="2"><xs:element name="a" type="xs:string" minOccurs="0"/>` +
		`<xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence>` + cgElC + `</xs:sequence></xs:complexType></xs:element>`
	enChoiceAnySchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:choice>` + cgElB + cgElC + `</xs:choice>` +
		`<xs:any namespace="##other" processContents="lax"/></xs:sequence></xs:complexType></xs:element>`
	enOptOccSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + cgElA + `<xs:choice>` + cgElB + `<xs:sequence minOccurs="0" maxOccurs="2">` + cgElC +
		`<xs:element name="d" type="xs:string"/></xs:sequence></xs:choice></xs:sequence></xs:complexType></xs:element>`
	enItemChoiceSchema = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="2"><xs:choice>` + cgElB + cgElC + `</xs:choice></xs:sequence></xs:complexType></xs:element>`
	enItemGroupSchema  = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="2"><xs:choice>` + cgElA + `<xs:sequence>` + cgElB + cgElC +
		`</xs:sequence></xs:choice></xs:sequence></xs:complexType></xs:element>`
	enOccChoiceSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:choice minOccurs="2" maxOccurs="3"><xs:element name="a" type="xs:string" minOccurs="0"/>` +
		cgElB + `</xs:choice>` + cgElC + `</xs:sequence></xs:complexType></xs:element>`
	enOccChoiceReqSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:choice minOccurs="2" maxOccurs="3">` + cgElA + cgElB + `</xs:choice>` +
		cgElC + `</xs:sequence></xs:complexType></xs:element>`
	enBranchOccSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="x" type="xs:string"/><xs:sequence><xs:choice minOccurs="2" maxOccurs="3">` +
		`<xs:element name="a" type="xs:string" minOccurs="0"/>` + cgElB + `</xs:choice>` + cgElC + `</xs:sequence></xs:choice></xs:complexType></xs:element>`
)

// The types codegen emits for the nested-encode schemas' s:r, in shape,
// which TestEncodeNestedTypesAreCodegens pins: enFlatPair, enPairFirst over
// enFlatPair's interface, enChoiceAny over noBC, noOnceAlt for
// enOptOccSchema, enItemChoice over noBC, enItemGroup over cgOChoice, and
// enOccChoice over occChoiceItem for enOccChoiceSchema and
// enOccChoiceReqSchema, and enBranchOcc for enBranchOccSchema, its
// enBranchOccSeq branch over occChoiceItem.
type (
	enFlatPair struct {
		C        string
		Sequence []enFlatPairItem
	}
	enFlatPairItem interface{ isEnFlatPairItem() }
	enFlatPairA    struct{ A string }
	enFlatPairB    struct{ B string }

	enPairFirst struct {
		Sequence []enFlatPairItem
		C        string
	}

	enChoiceAny struct {
		Choice noBC
		Any    anyElement
	}

	enItemChoice     struct{ Sequence []enItemChoiceItem }
	enItemChoiceItem struct{ Choice noBC }

	enItemGroup     struct{ Sequence []enItemGroupItem }
	enItemGroupItem struct{ Choice cgOChoice }

	enOccChoice struct {
		Choice []occChoiceItem
		C      string
	}

	enBranchOcc    interface{ isEnBranchOcc() }
	enBranchOccX   struct{ X string }
	enBranchOccSeq struct {
		Choice []occChoiceItem
		C      string
	}
)

func (enFlatPairA) isEnFlatPairItem() {}
func (enFlatPairB) isEnFlatPairItem() {}
func (enBranchOccX) isEnBranchOcc()   {}
func (enBranchOccSeq) isEnBranchOcc() {}

// The branch sets of enFlatPairItem and enBranchOcc.
var (
	enFlatPairBranches  = WithBranches[enFlatPairItem](enFlatPairA{}, enFlatPairB{})
	enBranchOccBranches = WithBranches[enBranchOcc](enBranchOccX{}, enBranchOccSeq{})
)

// TestEncodeNestedTypesAreCodegens pins the types above to those codegen
// emits for each nested-encode schema, in shape.
func TestEncodeNestedTypesAreCodegens(t *testing.T) {
	branch := func(iface, name, field string) string {
		return "type " + name + " struct {\n\t" + field + "\n}\n\nfunc (" + name + ") is" + iface + "() {}\n"
	}
	rows := []struct {
		decls string
		want  []string
	}{
		{enFlatPairSchema, []string{"type R struct {\n\tC        string\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceA", "A string"), branch("RSequence", "RSequenceB", "B string")}},
		{enPairFirstSchema, []string{"type R struct {\n\tSequence []RSequence\n\tC        string\n}\n", branch("RSequence", "RSequenceA", "A string"), branch("RSequence", "RSequenceB", "B string")}},
		{enChoiceAnySchema, []string{"type R struct {\n\tChoice RChoice\n\tAny    AnyElement\n}\n", branch("RChoice", "RChoiceB", "B string"), branch("RChoice", "RChoiceC", "C string")}},
		{enOptOccSchema, []string{"type R struct {\n\tA      string\n\tChoice RChoice\n}\n", branch("RChoice", "RChoiceB", "B string"),
			branch("RChoice", "RChoiceSequence", "Sequence []RChoiceSequence2"), "type RChoiceSequence2 struct {\n\tC string\n\tD string\n}\n"}},
		{enItemChoiceSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tChoice RSequenceChoice\n}\n",
			branch("RSequenceChoice", "RSequenceChoiceB", "B string"), branch("RSequenceChoice", "RSequenceChoiceC", "C string")}},
		{enItemGroupSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tChoice RSequenceChoice\n}\n",
			branch("RSequenceChoice", "RSequenceChoiceA", "A string"), "type RSequenceChoiceSequence struct {\n\tB string\n\tC string\n}\n\nfunc (RSequenceChoiceSequence) isRSequenceChoice() {}\n"}},
		{enOccChoiceSchema, []string{"type R struct {\n\tChoice []RChoice\n\tC      string\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}},
		{enOccChoiceReqSchema, []string{"type R struct {\n\tChoice []RChoice\n\tC      string\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}},
		{enBranchOccSchema, []string{"type R interface {\n\tisR()\n}\n", branch("R", "RX", "X string"), "type RSequence struct {\n\tChoice []RChoice\n\tC      string\n}\n\nfunc (RSequence) isR() {}\n",
			branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}},
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

// TestAppendNested pins the round trip of each nested model group particle
// hold "# What encodes" admits, the value written, accepted by validate and
// decoded back equal: a nested sequence codegen splices, inline or through
// <group ref>, in a content sequence and in an occurrence field's item, a
// wildcard among its particles; a nested choice occurring once, inline or
// through <group ref>, with a wildcard alternative, and its nil interface where
// an alternative is emptiable; a nested particle codegen flattens, inline or
// through <group ref>, a sequence's and a choice's, holding a wildcard,
// inside a spliced sequence, beside a later occurrence field and a later
// choice, and one of two alternatives ahead of an element, whose targets
// follow its two, as a wildcard's follow a nested choice's two alternatives'
// targets, both rows failing where writeMembers counts one target per
// member; a nested occurrence field, a sequence's and a choice's, repeated and
// optional, through <group ref>, holding a wildcard, a once-choice in its item,
// and two siblings; a choice's model-group alternative, spliced, through <group
// ref>, of the content choice, of a nested choice, of an occurrence choice's
// item, holding a nested choice, flattened and held in an occurrence field,
// the nil interface of a content choice whose model-group alternative has {min
// occurs} 0, and one item of a nested choice occurring 2 to 3 times, the
// occurrence it lacks matching the empty sequence through its alternative of
// {min occurs} 0 (§3.9.4.3 cvc-accept 3.1-3.3), in a content sequence and in
// a content choice's spliced sequence alternative. Each row fails with item 4's
// constructNestedGroup arm in particleEncodes put back
// (encodeWalk.groupEncodes), every row reaching a nested model group.
func TestAppendNested(t *testing.T) {
	const r = `<r xmlns="urn:s">`
	w := anyElement{Space: "urn:o", Local: "w", Content: anyType{Text: "w"}}
	const ow = `<p1:w xmlns="" xmlns:p1="urn:o">w</p1:w>`
	spliced := sequenceOf(`<xs:sequence>` + cgElA + `</xs:sequence>`)
	splicedRef := `<xs:group name="g"><xs:sequence>` + cgElA + `</xs:sequence></xs:group>` + sequenceOf(`<xs:group ref="s:g"/>`)
	inItem := root(`<xs:sequence maxOccurs="2">` + cgElA + `<xs:sequence>` + cgElB + `</xs:sequence></xs:sequence>`)
	inItemRef := `<xs:group name="g"><xs:sequence>` + cgElB + `</xs:sequence></xs:group>` + root(`<xs:sequence maxOccurs="2">`+cgElA+`<xs:group ref="s:g"/></xs:sequence>`)
	two := occSeq{Sequence: []occSeqItem{{A: "1", B: "2"}, {A: "3", B: "4"}}}
	for _, row := range []struct {
		name string
		run  func(t *testing.T)
	}{
		{"a spliced nested sequence", func(t *testing.T) {
			roundTrip(t, schemaOf(t, spliced), rootName, noSeqA{A: "1"}, r+`<a>1</a></r>`, deepEqual[noSeqA])
		}},
		{"a spliced nested <group ref>", func(t *testing.T) {
			roundTrip(t, schemaOf(t, splicedRef), rootName, noSeqA{A: "1"}, r+`<a>1</a></r>`, deepEqual[noSeqA])
		}},
		{"a spliced nested sequence in an occurrence field's item", func(t *testing.T) {
			roundTrip(t, schemaOf(t, inItem), rootName, two, r+`<a>1</a><b>2</b><a>3</a><b>4</b></r>`, deepEqual[occSeq])
		}},
		{"a spliced nested <group ref> in an occurrence field's item", func(t *testing.T) {
			roundTrip(t, schemaOf(t, inItemRef), rootName, two, r+`<a>1</a><b>2</b><a>3</a><b>4</b></r>`, deepEqual[occSeq])
		}},
		{"a wildcard in a spliced nested sequence", func(t *testing.T) {
			roundTrip(t, schemaOf(t, spliceWildInSchema), rootName, spliceWildIn{A: "1", Any: w, B: 2}, r+`<a>1</a>`+ow+`<b>2</b></r>`, deepEqual[spliceWildIn])
		}},
		{"a nested choice occurring once", func(t *testing.T) {
			roundTrip(t, schemaOf(t, ncInlineSchema), rootName, nc{A: "1", Choice: ncC{C: 2}}, r+`<a>1</a><c>2</c></r>`, deepEqual[nc], ncBranches)
		}},
		{"a nested <group ref> to a choice occurring once", func(t *testing.T) {
			roundTrip(t, schemaOf(t, ncRefSchema), rootName, nc{A: "1", Choice: ncB{B: "2"}}, r+`<a>1</a><b>2</b></r>`, deepEqual[nc], ncBranches)
		}},
		{"a nested choice's wildcard alternative", func(t *testing.T) {
			roundTrip(t, schemaOf(t, ncWildSchema), rootName, ncW{Any: anyElement{Space: "urn:o", Local: "w"}, Choice: ncWAny{Any: anyElement{Space: "urn:p", Local: "p"}}},
				r+`<p1:w xmlns="" xmlns:p1="urn:o"/><p1:p xmlns="" xmlns:p1="urn:p"/></r>`, deepEqual[ncW], ncWBranches)
		}},
		{"a nested emptiable choice's nil interface", func(t *testing.T) {
			roundTrip(t, schemaOf(t, ncEmptiableSchema), rootName, nc{A: "1"}, r+`<a>1</a></r>`, deepEqual[nc], ncBranches)
		}},
		{"a nested flattened sequence", func(t *testing.T) {
			roundTrip(t, schemaOf(t, nfSeqSchema), rootName, nfSeq{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}, nfSeqB{B: "3"}}}, r+`<a>1</a><b>2</b><b>3</b></r>`, deepEqual[nfSeq], nfSeqBranches)
		}},
		{"a nested flattened <group ref>", func(t *testing.T) {
			roundTrip(t, schemaOf(t, nfRefSchema), rootName, nfSeq{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}}}, r+`<a>1</a><b>2</b></r>`, deepEqual[nfSeq], nfSeqBranches)
		}},
		{"a nested flattened choice", func(t *testing.T) {
			roundTrip(t, schemaOf(t, nfNestedSchema), rootName, nfNested{A: "1", Choice: []nfNestedItem{nfNestedB{B: "2"}}}, r+`<a>1</a><b>2</b></r>`, deepEqual[nfNested], nfNestedBranches)
		}},
		{"a wildcard in a nested flattened sequence", func(t *testing.T) {
			v := nfWild{A: "1", Sequence: []nfWildItem{nfWildAny{Any: w}, nfWildB{B: "2"}}}
			roundTrip(t, schemaOf(t, nfWildSchema), rootName, v, r+`<a>1</a>`+ow+`<b>2</b></r>`, deepEqual[nfWild], nfWildBranches)
		}},
		{"a nested flattened sequence inside a spliced one", func(t *testing.T) {
			v := nfSplice{A: "1", C: "2", Sequence: []nfWildItem{nfWildB{B: "3"}, nfWildAny{Any: w}}}
			roundTrip(t, schemaOf(t, nfSpliceSchema), rootName, v, r+`<a>1</a><c>2</c><b>3</b>`+ow+`</r>`, deepEqual[nfSplice], nfWildBranches)
		}},
		{"a nested flattened sequence before an element and an occurrence field", func(t *testing.T) {
			v := nfOccAfter{A: "1", Sequence: []nfSeqItem{nfSeqB{B: "2"}}, C: "3", Sequence2: []nfOccBD{{B: "4", D: "5"}, {B: "6", D: "7"}}}
			roundTrip(t, schemaOf(t, nfOccAfterSchema), rootName, v, r+`<a>1</a><b>2</b><c>3</c><b>4</b><d>5</d><b>6</b><d>7</d></r>`, deepEqual[nfOccAfter], nfSeqBranches)
		}},
		{"a nested flattened sequence before a nested choice", func(t *testing.T) {
			v := nfChoice{Sequence: []nfSeqItem{nfSeqB{B: "1"}}, C: "2", Choice: nfChoiceB{B: "3"}}
			roundTrip(t, schemaOf(t, nfChoiceSchema), rootName, v, r+`<b>1</b><c>2</c><b>3</b></r>`, deepEqual[nfChoice], nfChoiceBranches...)
		}},
		{"a nested flattened particle of two alternatives before an element", func(t *testing.T) {
			v := enPairFirst{Sequence: []enFlatPairItem{enFlatPairA{A: "1"}, enFlatPairB{B: "2"}}, C: "3"}
			roundTrip(t, schemaOf(t, enPairFirstSchema), rootName, v, r+`<a>1</a><b>2</b><c>3</c></r>`, deepEqual[enPairFirst], enFlatPairBranches)
		}},
		{"a wildcard after a nested choice of two alternatives", func(t *testing.T) {
			roundTrip(t, schemaOf(t, enChoiceAnySchema), rootName, enChoiceAny{Choice: noB{B: "1"}, Any: w}, r+`<b>1</b>`+ow+`</r>`, deepEqual[enChoiceAny], noBCBranches)
		}},
		{"a nested repeated sequence", func(t *testing.T) {
			v := noRepSeq{A: "1", Sequence: []noSeqBC{{B: "2", C: "3"}, {B: "4", C: "5"}}}
			roundTrip(t, schemaOf(t, noRepSeqSchema), rootName, v, r+`<a>1</a><b>2</b><c>3</c><b>4</b><c>5</c></r>`, deepEqual[noRepSeq])
		}},
		{"a nested repeated choice", func(t *testing.T) {
			v := noRepChoice{A: "1", Choice: []noBC{noC{C: "2"}, noB{B: "3"}}}
			roundTrip(t, schemaOf(t, noRepChoiceSchema), rootName, v, r+`<a>1</a><c>2</c><b>3</b></r>`, deepEqual[noRepChoice], noBCBranches)
		}},
		{"a nested optional sequence, present", func(t *testing.T) {
			roundTrip(t, schemaOf(t, noOptSeqSchema), rootName, noOptSeq{A: "1", Sequence: &noSeqB{B: "2"}}, r+`<a>1</a><b>2</b></r>`, deepEqual[noOptSeq])
		}},
		{"a nested optional choice, absent", func(t *testing.T) {
			roundTrip(t, schemaOf(t, noOptChoiceSchema), rootName, noOptChoice{A: "1"}, r+`<a>1</a></r>`, deepEqual[noOptChoice], noBCBranches)
		}},
		{"a nested <group ref> occurring twice", func(t *testing.T) {
			v := noRepSeq{A: "1", Sequence: []noSeqBC{{B: "2", C: "3"}, {B: "4", C: "5"}}}
			roundTrip(t, schemaOf(t, noRefSchema), rootName, v, r+`<a>1</a><b>2</b><c>3</c><b>4</b><c>5</c></r>`, deepEqual[noRepSeq])
		}},
		{"a wildcard in a nested occurrence's item", func(t *testing.T) {
			v := noWild{A: "1", Sequence: []noWildItem{{B: "2", Any: w}, {B: "3", Any: w}}}
			roundTrip(t, schemaOf(t, noWildSchema), rootName, v, r+`<a>1</a><b>2</b>`+ow+`<b>3</b>`+ow+`</r>`, deepEqual[noWild])
		}},
		{"a once-choice in a nested occurrence's item", func(t *testing.T) {
			v := noInner{A: "1", Sequence: []noInnerItem{{Choice: noB{B: "2"}, D: "3"}, {Choice: noC{C: "4"}, D: "5"}}}
			roundTrip(t, schemaOf(t, noInnerSchema), rootName, v, r+`<a>1</a><b>2</b><d>3</d><c>4</c><d>5</d></r>`, deepEqual[noInner], noBCBranches)
		}},
		{"two sibling nested occurrences", func(t *testing.T) {
			v := noTwo{Sequence: []noSeqAB{{A: "1", B: "2"}, {A: "3", B: "4"}}, Sequence2: []noSeqCD{{C: "5", D: "6"}}}
			roundTrip(t, schemaOf(t, noTwoSchema), rootName, v, r+`<a>1</a><b>2</b><a>3</a><b>4</b><c>5</c><d>6</d></r>`, deepEqual[noTwo])
		}},
		{"a model-group alternative of the content choice", func(t *testing.T) {
			roundTrip(t, schemaOf(t, cgSeqSchema), rootName, cgR(cgSeq{B: "1", C: "2"}), r+`<b>1</b><c>2</c></r>`, deepEqual[cgR], cgBranches)
		}},
		{"a <group ref> alternative of the content choice", func(t *testing.T) {
			roundTrip(t, schemaOf(t, cgRefSchema), rootName, cgR(cgSeq{B: "1", C: "2"}), r+`<b>1</b><c>2</c></r>`, deepEqual[cgR], cgBranches)
		}},
		{"a model-group alternative of a nested choice", func(t *testing.T) {
			roundTrip(t, schemaOf(t, cgNestedSchema), rootName, cgN{X: "1", Choice: cgOSeq{B: "2", C: "3"}}, r+`<x>1</x><b>2</b><c>3</c></r>`, deepEqual[cgN], cgOBranches)
		}},
		{"a model-group alternative of an occurrence choice's item", func(t *testing.T) {
			v := cgRep{Choice: []cgOChoice{cgOSeq{B: "1", C: "2"}, cgOA{A: "3"}}}
			roundTrip(t, schemaOf(t, cgRepeatSchema), rootName, v, r+`<b>1</b><c>2</c><a>3</a></r>`, deepEqual[cgRep], cgOBranches)
		}},
		{"a model-group alternative holding a nested choice", func(t *testing.T) {
			roundTrip(t, schemaOf(t, cgInnerChoiceSchema), rootName, cgIC(cgICSeq{B: "1", Choice: cgICD{D: "2"}}), r+`<b>1</b><d>2</d></r>`, deepEqual[cgIC], cgICBranches...)
		}},
		{"a flattened model-group alternative", func(t *testing.T) {
			roundTrip(t, schemaOf(t, cgFlatSchema), rootName, cgFl(cgFlSeq{Sequence: []cgFlItem{cgFlB{B: "1"}, cgFlB{B: "2"}}}), r+`<b>1</b><b>2</b></r>`, deepEqual[cgFl], cgFlBranches...)
		}},
		{"an occurrence field as a once-choice's alternative", func(t *testing.T) {
			v := noOnceAlt{A: "1", Choice: noOnceAltSq{Sequence: []noSeqCD{{C: "2", D: "3"}, {C: "4", D: "5"}}}}
			roundTrip(t, schemaOf(t, noOnceAltSchema), rootName, v, r+`<a>1</a><c>2</c><d>3</d><c>4</c><d>5</d></r>`, deepEqual[noOnceAlt], noOnceAltBranches)
		}},
		{"an occurrence field as the content choice's alternative", func(t *testing.T) {
			v := noAlt(noAltSeq{Sequence: []noSeqBC{{B: "1", C: "2"}, {B: "3", C: "4"}}})
			roundTrip(t, schemaOf(t, noAltSchema), rootName, v, r+`<b>1</b><c>2</c><b>3</b><c>4</c></r>`, deepEqual[noAlt], noAltBranches)
		}},
		{"the nil interface of a content choice whose model-group alternative has {min occurs} 0", func(t *testing.T) {
			roundTrip(t, schemaOf(t, cgOptSchema), rootName, cgO{}, `<r xmlns="urn:s"/>`, deepEqual[cgO], cgOBranches)
		}},
		{"one item of a nested choice occurring 2 to 3 times, an element alternative emptiable", func(t *testing.T) {
			roundTrip(t, schemaOf(t, enOccChoiceSchema), rootName, enOccChoice{Choice: []occChoiceItem{occChoiceB{B: "1"}}, C: "2"}, r+`<b>1</b><c>2</c></r>`, deepEqual[enOccChoice], occChoiceBranches)
		}},
		{"one item of a nested choice occurring 2 to 3 times in a content choice's sequence alternative", func(t *testing.T) {
			v := enBranchOcc(enBranchOccSeq{Choice: []occChoiceItem{occChoiceB{B: "1"}}, C: "2"})
			roundTrip(t, schemaOf(t, enBranchOccSchema), rootName, v, r+`<b>1</b><c>2</c></r>`, deepEqual[enBranchOcc], enBranchOccBranches, occChoiceBranches)
		}},
	} {
		t.Run(row.name, row.run)
	}
}

// TestAppendNestedInvalid pins item 6 of Append's doc for a nested model
// group particle, none carrying an xsderr rule: the nil interface of a nested
// choice occurring once none of whose alternatives is emptiable, its
// model-group alternative's {min occurs} 1 among them, and so inside an
// occurrence field's item whose one member it is, an element alternative's
// and a model-group alternative's of {min occurs} 1, an item that would
// otherwise write nothing, and a dynamic type its WithBranches set places at
// no alternative; a nil item of a nested flattened particle's slice, and
// items whose order the content model rejects, the third a of sequence(a?,
// b?){1,2}, which no Matcher check but the content's catches, since the
// schema has no identity constraint and Append assesses nothing; a nested
// occurrence field's length outside its occurrence, and outside the
// occurrence its branch reads a model-group alternative's {min occurs} 0 as,
// the error naming both, and one item of a nested choice occurring 2 to 3
// times none of whose alternatives is emptiable; and a nil item of a nested
// occurrence's choice. Each row's prefix names the value's Go path first.
func TestAppendNestedInvalid(t *testing.T) {
	for _, row := range []struct {
		name   string
		run    func(t *testing.T) error
		prefix string
		want   []string
	}{
		{"a nested choice's nil interface", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, cgNestedSchema), strict.New(), rootName, cgN{X: "1"}, cgOBranches)
			return err
		}, "codec: Append: cgN.Choice is nil, the empty match, which a nested choice particle of the {type definition} of element {urn:s}r declared at ", []string{"none of its 2 alternatives is emptiable"}},
		{"an occurrence item's nil once-choice", func(t *testing.T) error {
			v := enItemChoice{Sequence: []enItemChoiceItem{{Choice: noB{B: "1"}}, {}}}
			_, err := appendChecked(t, nil, schemaOf(t, enItemChoiceSchema), strict.New(), rootName, v, noBCBranches)
			return err
		}, "codec: Append: enItemChoice.Sequence[1].Choice is nil, the empty match, which a nested choice particle of ", []string{"none of its 2 alternatives is emptiable"}},
		{"an occurrence item's nil once-choice of a model-group alternative", func(t *testing.T) error {
			v := enItemGroup{Sequence: []enItemGroupItem{{Choice: cgOA{A: "1"}}, {}}}
			_, err := appendChecked(t, nil, schemaOf(t, enItemGroupSchema), strict.New(), rootName, v, cgOBranches)
			return err
		}, "codec: Append: enItemGroup.Sequence[1].Choice is nil, the empty match, which a nested choice particle of ", []string{"none of its 2 alternatives is emptiable"}},
		{"a nested choice's branch pointer", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, ncInlineSchema), strict.New(), rootName, nc{A: "1", Choice: &ncB{B: "2"}}, WithBranches[ncChoice](ncB{}, ncC{}))
			return err
		}, "codec: Append: nc.Choice holds a *codec.ncB, which the WithBranches set of codec.ncChoice places at no alternative of a nested choice particle of ", nil},
		{"a nil item of a nested flattened particle", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, nfSeqSchema), strict.New(), rootName, nfSeq{A: "1", Sequence: []nfSeqItem{nil}}, nfSeqBranches)
			return err
		}, "codec: Append: nfSeq.Sequence[0] is nil, an item of the nested flattened particle of the {type definition} of element {urn:s}r declared at ", nil},
		{"a nested flattened particle's items in an order its content model rejects", func(t *testing.T) error {
			v := enFlatPair{C: "1", Sequence: []enFlatPairItem{enFlatPairA{A: "2"}, enFlatPairA{A: "3"}, enFlatPairA{A: "4"}}}
			_, err := appendChecked(t, nil, schemaOf(t, enFlatPairSchema), strict.New(), rootName, v, enFlatPairBranches)
			return err
		}, "codec: Append: enFlatPair.Sequence[2].(enFlatPairA).A holds element {urn:s}a, which no particle of the content model of the {type definition} of element {urn:s}r declared at ", nil},
		{"a nested occurrence field's length", func(t *testing.T) error {
			v := noRepSeq{A: "1", Sequence: make([]noSeqBC, 3)}
			_, err := appendChecked(t, nil, schemaOf(t, noRepSeqSchema), strict.New(), rootName, v)
			return err
		}, "codec: Append: noRepSeq.Sequence holds 3 items, outside the occurrence 1..2 of a nested sequence model group particle of the {type definition} of element {urn:s}r declared at ", nil},
		{"a nested occurrence alternative's length, its {min occurs} 0 read as 1", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, enOptOccSchema), strict.New(), rootName, noOnceAlt{A: "1", Choice: noOnceAltSq{}}, noOnceAltBranches)
			return err
		}, "codec: Append: noOnceAlt.Choice.(noOnceAltSq).Sequence holds 0 items, outside the occurrence 1..2, the occurrence 0..2 with {min occurs} 0 read as 1 in its branch, of a nested sequence model group particle of ", nil},
		{"a nested choice occurrence field's length, none of its alternatives emptiable", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, enOccChoiceReqSchema), strict.New(), rootName, enOccChoice{Choice: []occChoiceItem{occChoiceB{B: "1"}}, C: "2"}, occChoiceBranches)
			return err
		}, "codec: Append: enOccChoice.Choice holds 1 items, outside the occurrence 2..3 of a nested choice model group particle of the {type definition} of element {urn:s}r declared at ", nil},
		{"a nil item of a nested occurrence's choice", func(t *testing.T) error {
			_, err := appendChecked(t, nil, schemaOf(t, noRepChoiceSchema), strict.New(), rootName, noRepChoice{A: "1", Choice: []noBC{noB{B: "2"}, nil}}, noBCBranches)
			return err
		}, "codec: Append: noRepChoice.Choice[1] is nil, an item holding no child element, which Decode never returns in the occurrence field of a nested choice model group particle of ", nil},
		{"a nil item of an occurrence choice whose model-group alternative has {min occurs} 0", func(t *testing.T) error {
			schema := schemaOf(t, strings.Replace(cgRepeatSchema, `<xs:sequence>`, `<xs:sequence minOccurs="0">`, 1))
			_, err := appendChecked(t, nil, schema, strict.New(), rootName, cgRep{Choice: []cgOChoice{cgOA{A: "1"}, nil}}, cgOBranches)
			return err
		}, "codec: Append: cgRep.Choice[1] is nil, an item holding no child element, which Decode never returns in the occurrence field of the content choice of ", nil},
	} {
		t.Run(row.name, func(t *testing.T) {
			invalidChoice(t, row.run(t), row.prefix, row.want...)
		})
	}
}

// TestAppendNestedDeclineSubject pins the subject item 4 of Append's doc
// names for a particle inside a nested model group particle, here a wildcard
// particle rebuilt vacuous (vacatedSchema): a spliced sequence's particle and
// a nested choice's alternative by their place inside their group's, a
// nested flattened particle's basic particle by its reach ordinal, and a
// nested occurrence field's particle by its place.
func TestAppendNestedDeclineSubject(t *testing.T) {
	const wild = "codec does not encode a particle with {max occurs} 0: "
	const any = `<xs:any namespace="##other"/>`
	for _, row := range []struct{ name, content, subject string }{
		{"spliced", `<xs:sequence>` + cgElA + `<xs:sequence>` + cgElB + any + `</xs:sequence></xs:sequence>`, "particle 2 of particle 2 of the content {particle} of "},
		{"once-choice", `<xs:sequence>` + cgElA + `<xs:choice>` + cgElB + any + `</xs:choice></xs:sequence>`, "alternative 2 of particle 2 of the content {particle} of "},
		{"flattened", `<xs:sequence>` + cgElA + `<xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/><xs:any namespace="##other" minOccurs="0"/></xs:sequence></xs:sequence>`,
			"alternative 2 of the flattened particle 2 of the content {particle} of "},
		{"occurrence", `<xs:sequence>` + cgElA + `<xs:sequence maxOccurs="2">` + cgElB + any + `</xs:sequence></xs:sequence>`, "particle 2 of particle 2 of the content {particle} of "},
	} {
		t.Run(row.name, func(t *testing.T) {
			_, err := appendChecked(t, nil, vacatedSchema(t, row.content), strict.New(), rootName, struct{}{})
			_ = asDecline(t, err, wild+row.subject+"the {type definition} of element {urn:s}r declared at ")
		})
	}
}
