package codec

import (
	"bytes"
	"errors"
	"reflect"
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
// wildcard in the nested reach; and nfSpliceSchema's sequence(a, sequence(c,
// sequence(b?, any over ##other?) repeated)), the flattened particle inside a
// spliced sequence.
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
)

// The types codegen emits for the nested-flatten schemas' s:r, in shape,
// which TestNestedFlatTypesAreCodegens pins: nfSeq for nfSeqSchema and
// nfRefSchema, flatOpt (flatten_test.go) for nfOnceSchema, nfNested for
// nfNestedSchema, nfAfter for nfAfterSchema, nfWild for nfWildSchema and
// nfSplice, over nfWild's interface, for nfSpliceSchema.
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
)

func (nfSeqB) isNfSeqItem()       {}
func (nfNestedC) isNfNestedItem() {}
func (nfNestedB) isNfNestedItem() {}
func (nfWildB) isNfWildItem()     {}
func (nfWildAny) isNfWildItem()   {}

// The branch sets of the nested-flatten interfaces.
var (
	nfSeqBranches    = WithBranches[nfSeqItem](nfSeqB{})
	nfNestedBranches = WithBranches[nfNestedItem](nfNestedC{}, nfNestedB{})
	nfWildBranches   = WithBranches[nfWildItem](nfWildB{}, nfWildAny{})
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
// items are codegen's convention, a non-empty occurrence each. An element
// child of a nested reach is attributed by its expanded name, and a wildcard
// one by its Matcher position below the nested particle's, through a spliced
// sequence too. CheckDecode returns nil for each row's T.
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
// declining, Decode and CheckDecode alike: a repeated nested sequence codegen
// does not flatten, which takes an occurrence field (#3588); a nested particle
// codegen flattens inside a content particle held in an occurrence field,
// whose items' partition reads no repetition inside an item; and a nested
// reach's element particle sharing an expanded name with a sibling particle
// or with an earlier particle of the same reach, which the nested reach
// attributes by name, so it declines as constructSharedName where a content
// reach's declines as constructSharedBranch (TestDecodeFlattenSharedBranch).
func TestDecodeNestedFlattenedDeclines(t *testing.T) {
	rows := []struct {
		name, decls, body string
		construct         construct
	}{
		{"a repeated nested sequence codegen does not flatten", sequenceOf(`<xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="2"><xs:element name="b" type="xs:string"/></xs:sequence>`),
			"<s:a>1</s:a><s:b>2</s:b>", constructNestedGroup},
		{"a nested flattened sequence in an occurrence field's item", root(`<xs:sequence maxOccurs="2"><xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:sequence>`),
			"<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b>", constructNestedGroup},
		{"a nested reach's particle sharing a sibling's expanded name", sequenceOf(`<xs:element name="b" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence>`),
			"<s:b>1</s:b><s:b>2</s:b>", constructSharedName},
		{"a nested reach's two particles of one expanded name", sequenceOf(`<xs:element name="a" type="xs:string"/><xs:sequence maxOccurs="unbounded"><xs:sequence minOccurs="0"><xs:element name="c" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>` +
			`<xs:sequence minOccurs="0"><xs:element name="d" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:sequence>`),
			"<s:a>1</s:a><s:c>2</s:c><s:b>3</s:b>", constructSharedName},
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

// TestCheckDecodeUntrackedNestedFlattened pins the nested flattened arm's
// ContentMatcher decline: sequence(any?, (a{1,2}){1,21845}), the nested
// particle flattened by its repeating edge, is a content model
// xsd.Schema.ContentMatcher declines as too wide to track, so CheckDecode
// returns constructUntrackedModel, a *DeclineError and no plain error, though
// the content model's wildcard would have Decode build its Matcher through
// wildcardSlot.attributor. Validation leaves such an instance undecided, so
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
