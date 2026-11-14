package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// snGroups declares s:g of xs:string heading the substitution group of s:m,
// also of xs:string, which codegen gives the sealed interface
// GSubstitutionGroup, branches G and M.
const snGroups = `<xs:element name="g" type="xs:string"/><xs:element name="m" type="xs:string" substitutionGroup="s:g"/>`

// The shared-name schemas, each declaring s:r, whose content model holds two
// element particles taking one expanded name that item 5 of Decode's doc lets
// decode: snFlatSchema's sequence(a, b, a); snSpliceSchema's sequence(a,
// sequence(b, a)), the inner sequence spliced; snSiblingSchema's sequence(b,
// sequence(b, c)), likewise; snAroundSchema's sequence(b, sequence(c?)
// repeated, b), the two b siblings around a nested reach; snWildSchema's
// sequence(a, a, any not s:a), wild029's shape; snGroupDeclSchema's
// sequence(<element ref> g, m), the head's group holding m before m's own
// particle; snDeclGroupSchema's sequence(m, <element ref> g), the reverse;
// snGroupRefSchema's sequence(<element ref> g, <element ref> m);
// snLocalRefSchema's sequence(a, <element ref> a), a local declaration and a
// top-level one; snChoiceAfterSchema's sequence(a, choice(a, b)) and
// snChoiceBeforeSchema's sequence(choice(a, b), a), the choice bound in
// place, mgO006's shape; snAltsSchema's content choice(sequence(a, b),
// sequence(c, a)), two spliced alternatives; and snAltSeqSchema's content
// choice(sequence(a, b, a), b), two particles of one spliced alternative,
// mgQ003's shape.
const (
	snFlatSchema         = `<xs:element name="r"><xs:complexType><xs:sequence>` + snA + snB + snA + `</xs:sequence></xs:complexType></xs:element>`
	snSpliceSchema       = `<xs:element name="r"><xs:complexType><xs:sequence>` + snA + `<xs:sequence>` + snB + snA + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	snSiblingSchema      = `<xs:element name="r"><xs:complexType><xs:sequence>` + snB + `<xs:sequence>` + snB + snC + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	snAroundSchema       = `<xs:element name="r"><xs:complexType><xs:sequence>` + snB + `<xs:sequence maxOccurs="unbounded"><xs:element name="c" type="xs:string" minOccurs="0"/></xs:sequence>` + snB + `</xs:sequence></xs:complexType></xs:element>`
	snWildSchema         = `<xs:element name="r"><xs:complexType><xs:sequence>` + snA + snA + `<xs:any notQName="s:a" processContents="skip"/></xs:sequence></xs:complexType></xs:element>`
	snGroupDeclSchema    = snGroups + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:g"/><xs:element name="m" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	snDeclGroupSchema    = snGroups + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="m" type="xs:string"/><xs:element ref="s:g"/></xs:sequence></xs:complexType></xs:element>`
	snGroupRefSchema     = snGroups + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:g"/><xs:element ref="s:m"/></xs:sequence></xs:complexType></xs:element>`
	snLocalRefSchema     = `<xs:element name="a" type="xs:string"/><xs:element name="r"><xs:complexType><xs:sequence>` + snA + `<xs:element ref="s:a"/></xs:sequence></xs:complexType></xs:element>`
	snChoiceAfterSchema  = `<xs:element name="r"><xs:complexType><xs:sequence>` + snA + `<xs:choice>` + snA + snB + `</xs:choice></xs:sequence></xs:complexType></xs:element>`
	snChoiceBeforeSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:choice>` + snA + snB + `</xs:choice>` + snA + `</xs:sequence></xs:complexType></xs:element>`
	snAltsSchema         = `<xs:element name="r"><xs:complexType><xs:choice><xs:sequence>` + snA + snB + `</xs:sequence><xs:sequence>` + snC + snA + `</xs:sequence></xs:choice></xs:complexType></xs:element>`
	snAltSeqSchema       = `<xs:element name="r"><xs:complexType><xs:choice><xs:sequence>` + snA + snB + snA + `</xs:sequence>` + snB + `</xs:choice></xs:complexType></xs:element>`

	snA = `<xs:element name="a" type="xs:string"/>`
	snB = `<xs:element name="b" type="xs:string"/>`
	snC = `<xs:element name="c" type="xs:string"/>`
)

// The types codegen emits for the shared-name schemas' s:r, in shape, which
// TestSharedNameTypesAreCodegens pins: snABA for snFlatSchema and
// snSpliceSchema, snBBC for snSiblingSchema, snAround, its item snAroundC,
// for snAroundSchema, snWild for snWildSchema, snGroupDecl, snDeclGroup and
// snGroupRef, over the interface snGroup, for the schemas of their names,
// snLocalRef for snLocalRefSchema, snChoiceAfter and snChoiceBefore, over
// snChoice, for theirs, snAlts, its branches snAltsAB and snAltsCA, for
// snAltsSchema, and snAltSeq, its branches snAltSeqABA and snAltSeqB, for
// snAltSeqSchema.
type (
	snABA struct{ A, B, A2 string }
	snBBC struct{ B, B2, C string }

	snAround struct {
		B        string
		Sequence []snAroundItem
		B2       string
	}
	snAroundItem interface{ isSnAroundItem() }
	snAroundC    struct{ C string }

	snWild struct {
		A, A2 string
		Any   anyElement
	}

	snG      string
	snM      string
	snGroup  interface{ isSnGroup() }
	snGroupG struct{ G snG }
	snGroupM struct{ M snM }

	snGroupDecl struct {
		G snGroup
		M string
	}
	snDeclGroup struct {
		M string
		G snGroup
	}
	snGroupRef struct {
		G snGroup
		M snM
	}

	snA2       string
	snLocalRef struct {
		A  string
		A2 snA2
	}

	snChoice  interface{ isSnChoice() }
	snChoiceA struct{ A string }
	snChoiceB struct{ B string }

	snChoiceAfter struct {
		A      string
		Choice snChoice
	}
	snChoiceBefore struct {
		Choice snChoice
		A      string
	}

	snAlts   interface{ isSnAlts() }
	snAltsAB struct{ A, B string }
	snAltsCA struct{ C, A string }

	snAltSeq    interface{ isSnAltSeq() }
	snAltSeqABA struct{ A, B, A2 string }
	snAltSeqB   struct{ B string }
)

func (snAroundC) isSnAroundItem() {}
func (snGroupG) isSnGroup()       {}
func (snGroupM) isSnGroup()       {}
func (snChoiceA) isSnChoice()     {}
func (snChoiceB) isSnChoice()     {}
func (snAltsAB) isSnAlts()        {}
func (snAltsCA) isSnAlts()        {}
func (snAltSeqABA) isSnAltSeq()   {}
func (snAltSeqB) isSnAltSeq()     {}

// The branch sets of the shared-name interfaces.
var (
	snAroundBranches = WithBranches[snAroundItem](snAroundC{})
	snGroupBranches  = WithBranches[snGroup](snGroupG{}, snGroupM{})
	snChoiceBranches = WithBranches[snChoice](snChoiceA{}, snChoiceB{})
	snAltsBranches   = WithBranches[snAlts](snAltsAB{}, snAltsCA{})
	snAltSeqBranches = WithBranches[snAltSeq](snAltSeqABA{}, snAltSeqB{})
)

// TestSharedNameTypesAreCodegens pins the types above to those codegen emits
// for each shared-name schema's s:r, in shape, so the rows decoding into them
// decode into codegen's types.
func TestSharedNameTypesAreCodegens(t *testing.T) {
	branch := func(iface, name, field string) string {
		return "type " + name + " struct {\n\t" + field + "\n}\n\nfunc (" + name + ") is" + iface + "() {}\n"
	}
	aba := "type R struct {\n\tA  string\n\tB  string\n\tA2 string\n}\n"
	group := []string{"type G string\n", "type M string\n", branch("GSubstitutionGroup", "GSubstitutionGroupG", "G G"), branch("GSubstitutionGroup", "GSubstitutionGroupM", "M M")}
	choice := []string{branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}
	rows := []struct {
		decls string
		want  []string
	}{
		{snFlatSchema, []string{aba}},
		{snSpliceSchema, []string{aba}},
		{snSiblingSchema, []string{"type R struct {\n\tB  string\n\tB2 string\n\tC  string\n}\n"}},
		{snAroundSchema, []string{"type R struct {\n\tB        string\n\tSequence []RSequence\n\tB2       string\n}\n", branch("RSequence", "RSequenceC", "C string")}},
		{snWildSchema, []string{"type R struct {\n\tA   string\n\tA2  string\n\tAny AnyElement\n}\n"}},
		{snGroupDeclSchema, append([]string{"type R struct {\n\tG GSubstitutionGroup\n\tM string\n}\n"}, group...)},
		{snDeclGroupSchema, append([]string{"type R struct {\n\tM string\n\tG GSubstitutionGroup\n}\n"}, group...)},
		{snGroupRefSchema, append([]string{"type R struct {\n\tG GSubstitutionGroup\n\tM M\n}\n"}, group...)},
		{snLocalRefSchema, []string{"type A string\n", "type R struct {\n\tA  string\n\tA2 A\n}\n"}},
		{snChoiceAfterSchema, append([]string{"type R struct {\n\tA      string\n\tChoice RChoice\n}\n"}, choice...)},
		{snChoiceBeforeSchema, append([]string{"type R struct {\n\tChoice RChoice\n\tA      string\n}\n"}, choice...)},
		{snAltsSchema, []string{"type RSequence struct {\n\tA string\n\tB string\n}\n\nfunc (RSequence) isR() {}\n", "type RSequence2 struct {\n\tC string\n\tA string\n}\n\nfunc (RSequence2) isR() {}\n"}},
		{snAltSeqSchema, []string{"type RSequence struct {\n\tA  string\n\tB  string\n\tA2 string\n}\n\nfunc (RSequence) isR() {}\n", branch("R", "RB", "B string")}},
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

// TestDecodeSharedName pins the shared-name lift: two element particles of
// one content model taking one expanded name that item 5 of Decode's doc
// lets decode, a declaration's field beside another's or beside a group's,
// flat, spliced, around a nested reach, beside a wildcard or a choice bound
// in place, or across a content choice's alternatives, put each child to the
// field of the particle the Matcher attributes it to by position (§3.8.4.1
// cvc-accept; §3.8.6.4 cos-nonambig), never by its name, a member of g's
// group to g's particle's field where the Matcher attributes it to that
// particle's position (cvc-accept clause 2.3.2). CheckDecode returns nil for
// each row's T.
func TestDecodeSharedName(t *testing.T) {
	w := anyElement{Space: "urn:o", Local: "w"}
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"a flat sequence's two particles", snFlatSchema, "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a>", into[snABA](), snABA{A: "1", B: "2", A2: "3"}},
		{"a spliced sequence's particle and an earlier sibling", snSpliceSchema, "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a>", into[snABA](), snABA{A: "1", B: "2", A2: "3"}},
		{"a spliced sequence's particle and the sibling before it", snSiblingSchema, "<s:b>1</s:b><s:b>2</s:b><s:c>3</s:c>", into[snBBC](), snBBC{B: "1", B2: "2", C: "3"}},
		{"two siblings around a nested reach", snAroundSchema, "<s:b>1</s:b><s:c>2</s:c><s:b>3</s:b>", into[snAround](snAroundBranches),
			snAround{B: "1", Sequence: []snAroundItem{snAroundC{C: "2"}}, B2: "3"}},
		{"two siblings around an empty nested reach", snAroundSchema, "<s:b>1</s:b><s:b>3</s:b>", into[snAround](snAroundBranches), snAround{B: "1", B2: "3"}},
		{"two particles beside a wildcard, wild029's shape", snWildSchema, `<s:a>1</s:a><s:a>2</s:a><o:w xmlns:o="urn:o"/>`, into[snWild](), snWild{A: "1", A2: "2", Any: w}},
		{"a group's head, then a particle of its member's name", snGroupDeclSchema, "<s:g>1</s:g><s:m>2</s:m>", into[snGroupDecl](snGroupBranches),
			snGroupDecl{G: snGroupG{G: "1"}, M: "2"}},
		{"a group's member, then a particle of its name", snGroupDeclSchema, "<s:m>1</s:m><s:m>2</s:m>", into[snGroupDecl](snGroupBranches),
			snGroupDecl{G: snGroupM{M: "1"}, M: "2"}},
		{"a particle of a member's name, then the member in its group", snDeclGroupSchema, "<s:m>1</s:m><s:m>2</s:m>", into[snDeclGroup](snGroupBranches),
			snDeclGroup{M: "1", G: snGroupM{M: "2"}}},
		{"a group's member, then a ref to the member", snGroupRefSchema, "<s:m>1</s:m><s:m>2</s:m>", into[snGroupRef](snGroupBranches),
			snGroupRef{G: snGroupM{M: "1"}, M: "2"}},
		{"a local declaration, then a ref to a top-level one of its name", snLocalRefSchema, "<s:a>1</s:a><s:a>2</s:a>", into[snLocalRef](), snLocalRef{A: "1", A2: "2"}},
		{"a choice alternative sharing an earlier particle's name", snChoiceAfterSchema, "<s:a>1</s:a><s:a>2</s:a>", into[snChoiceAfter](snChoiceBranches),
			snChoiceAfter{A: "1", Choice: snChoiceA{A: "2"}}},
		{"a later particle sharing a choice alternative's name", snChoiceBeforeSchema, "<s:a>1</s:a><s:a>2</s:a>", into[snChoiceBefore](snChoiceBranches),
			snChoiceBefore{Choice: snChoiceA{A: "1"}, A: "2"}},
		{"a later particle sharing a choice alternative's name, the other alternative", snChoiceBeforeSchema, "<s:b>1</s:b><s:a>2</s:a>", into[snChoiceBefore](snChoiceBranches),
			snChoiceBefore{Choice: snChoiceB{B: "1"}, A: "2"}},
		{"a content choice's alternatives sharing a name, the first", snAltsSchema, "<s:a>1</s:a><s:b>2</s:b>", into[snAlts](snAltsBranches), snAlts(snAltsAB{A: "1", B: "2"})},
		{"a content choice's alternatives sharing a name, the second", snAltsSchema, "<s:c>1</s:c><s:a>2</s:a>", into[snAlts](snAltsBranches), snAlts(snAltsCA{C: "1", A: "2"})},
		{"two particles of one spliced choice alternative, mgQ003's shape", snAltSeqSchema, "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a>", into[snAltSeq](snAltSeqBranches),
			snAltSeq(snAltSeqABA{A: "1", B: "2", A2: "3"})},
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

// TestDecodeSharedNameDeclines pins what the shared-name lift leaves
// declining that no other table pins, Decode and CheckDecode alike, each
// row's prefix the decline's subject: a second group's interface over a
// member whose name an earlier group's interface took before a declaration's
// particle took it too, the one row that fails where childIndex.take
// overwrites a name's taker; and a later sibling sharing the name of an
// earlier particle bound in an occurrence field's item, the one row that
// fails where particleShape reads no earlier taker's inItem.
// TestDecodeDeclines pins two groups' interfaces sharing a member, and
// TestDecodeNestedOccurrenceDeclines an item's particle sharing an earlier
// sibling's name.
func TestDecodeSharedNameDeclines(t *testing.T) {
	const prefix = "?:1:1: codec does not decode two particles taking one expanded name, each by its own element declaration or a member of its substitution group: element {urn:s}"
	rows := []struct {
		name, decls, body, subject string
	}{
		{"a group's interface, a declaration and a second group's interface over one member", `<xs:element name="g" type="xs:string"/><xs:element name="g2" type="xs:string"/>
			<xs:element name="m" type="xs:string" substitutionGroup="s:g s:g2"/>` + sequenceOf(`<xs:element ref="s:g"/><xs:element name="m" type="xs:string"/><xs:element ref="s:g2"/>`),
			`<s:g>x</s:g><s:m>y</s:m><s:g2>z</s:g2>`, "m declared at"},
		{"a sibling sharing an earlier occurrence field's item particle's name", sequenceOf(`<xs:sequence maxOccurs="2">` + snB + snC + `</xs:sequence>` + `<xs:element name="d" type="xs:string"/>` + snB),
			"<s:b>1</s:b><s:c>2</s:c><s:d>3</s:d><s:b>4</s:b>", "b declared at"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf("", row.body)
			mustAccept(t, schema, data)
			_, decoded := Decode[struct{}](schema, strict.New(), data)
			assertDecline(t, decoded, constructSharedName, 1, 1)
			if !strings.HasPrefix(decoded.Error(), prefix+row.subject) {
				t.Errorf("Decode error = %v, want the prefix %q", decoded, prefix+row.subject)
			}
			var de *DeclineError
			if err := CheckDecode[struct{}](schema, rootName); !errors.As(err, &de) || de.Construct() != (Construct{constructSharedName}) {
				t.Errorf("CheckDecode = %v, want a *DeclineError of %v", err, Construct{constructSharedName})
			}
		})
	}
}
