package codec

import (
	"bytes"
	"reflect"
	"testing"
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
// mgQ003's shape; snTwoHeadsSchema's sequence(<element ref> e0, <element ref>
// e1), e3 a member of both heads' groups through the abstract e2,
// s2_2_2v01's shape; snHeadTwiceSchema's sequence(<element ref> s,
// choice(sequence(any not ##definedSibling, a), sequence(<element ref> s,
// b))), s's group reached twice, the second ref in a choice alternative,
// snbranch's shape; and snGroupDeclGroupSchema's sequence(<element ref> g, m,
// <element ref> g2), m a member of both heads' groups and declared between
// them. Five hold the share at an occurrence field's item (bindItem):
// snItemTwoSchema's sequence(a, sequence(b, c, b)?) and snItemTwiceSchema's
// sequence(a, sequence(b, c, b){1,2}), both particles inside the item,
// particlesQ030's and particlesQ032's shape; snItemBeforeSchema's
// sequence(b, sequence(b, c){1,2}), the sibling before the item, and
// snItemAfterSchema's sequence(sequence(b, c){1,2}, d, b), the item before
// the sibling; and snItemRepSchema's sequence(b, sequence(b{1,2})?), the
// item's particle repeated, particlesQ024's shape.
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
	snTwoHeadsSchema     = `<xs:element name="e0" type="xs:string"/><xs:element name="e1" type="xs:string"/>
		<xs:element name="e2" type="xs:string" abstract="true" substitutionGroup="s:e0 s:e1"/><xs:element name="e3" type="xs:string" substitutionGroup="s:e2"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:e0"/><xs:element ref="s:e1"/></xs:sequence></xs:complexType></xs:element>`
	snHeadTwiceSchema = `<xs:element name="s" type="xs:string"/><xs:element name="s1" type="xs:string" substitutionGroup="s:s"/>` + snA + snB + `<xs:element name="e" type="xs:string"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:s"/><xs:choice><xs:sequence><xs:any notQName="##definedSibling"/><xs:element ref="s:a"/></xs:sequence>` +
		`<xs:sequence><xs:element ref="s:s"/><xs:element ref="s:b"/></xs:sequence></xs:choice></xs:sequence></xs:complexType></xs:element>`
	snGroupDeclGroupSchema = `<xs:element name="g" type="xs:string"/><xs:element name="g2" type="xs:string"/>
		<xs:element name="m" type="xs:string" substitutionGroup="s:g s:g2"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:g"/><xs:element name="m" type="xs:string"/><xs:element ref="s:g2"/></xs:sequence></xs:complexType></xs:element>`

	snItemTwoSchema    = `<xs:element name="r"><xs:complexType><xs:sequence>` + snA + `<xs:sequence minOccurs="0">` + snB + snC + snB + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	snItemTwiceSchema  = `<xs:element name="r"><xs:complexType><xs:sequence>` + snA + `<xs:sequence maxOccurs="2">` + snB + snC + snB + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	snItemBeforeSchema = `<xs:element name="r"><xs:complexType><xs:sequence>` + snB + `<xs:sequence maxOccurs="2">` + snB + snC + `</xs:sequence></xs:sequence></xs:complexType></xs:element>`
	snItemAfterSchema  = `<xs:element name="r"><xs:complexType><xs:sequence><xs:sequence maxOccurs="2">` + snB + snC + `</xs:sequence><xs:element name="d" type="xs:string"/>` + snB + `</xs:sequence></xs:complexType></xs:element>`
	snItemRepSchema    = `<xs:element name="r"><xs:complexType><xs:sequence>` + snB + `<xs:sequence minOccurs="0"><xs:element name="b" type="xs:string" maxOccurs="2"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`

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
// snAltsSchema, snAltSeq, its branches snAltSeqABA and snAltSeqB, for
// snAltSeqSchema, snTwoHeads, over the interfaces snE0Group and snE1Group,
// each with its own branch struct of e3, for snTwoHeadsSchema, snHeadTwice,
// over the interface snSGroup in both of its particles' fields and the
// choice snHeadTwiceChoice, for snHeadTwiceSchema, snGroupDeclGroup, over
// snGroup and snG2Group, for snGroupDeclGroupSchema, and, for the five
// item schemas, snItemTwo and snItemTwice over the item snItemTwoItem,
// snItemBefore and snItemAfter over snItemBC, and snItemRep over
// snItemRepItem.
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

	snE0        string
	snE1        string
	snE3        string
	snE0Group   interface{ isSnE0Group() }
	snE0GroupE0 struct{ E0 snE0 }
	snE0GroupE3 struct{ E3 snE3 }
	snE1Group   interface{ isSnE1Group() }
	snE1GroupE1 struct{ E1 snE1 }
	snE1GroupE3 struct{ E3 snE3 }
	snTwoHeads  struct {
		E0 snE0Group
		E1 snE1Group
	}

	snS               string
	snS1              string
	snRefA            string
	snRefB            string
	snSGroup          interface{ isSnSGroup() }
	snSGroupS         struct{ S snS }
	snSGroupS1        struct{ S1 snS1 }
	snHeadTwiceChoice interface{ isSnHeadTwiceChoice() }
	snHeadTwiceAnyA   struct {
		Any anyElement
		A   snRefA
	}
	snHeadTwiceSB struct {
		S snSGroup
		B snRefB
	}
	snHeadTwice struct {
		S      snSGroup
		Choice snHeadTwiceChoice
	}

	snG2             string
	snG2Group        interface{ isSnG2Group() }
	snG2GroupG2      struct{ G2 snG2 }
	snG2GroupM       struct{ M snM }
	snGroupDeclGroup struct {
		G  snGroup
		M  string
		G2 snG2Group
	}

	snItemTwoItem struct{ B, C, B2 string }
	snItemTwo     struct {
		A        string
		Sequence *snItemTwoItem
	}
	snItemTwice struct {
		A        string
		Sequence []snItemTwoItem
	}

	snItemBC     struct{ B, C string }
	snItemBefore struct {
		B        string
		Sequence []snItemBC
	}
	snItemAfter struct {
		Sequence []snItemBC
		D        string
		B        string
	}

	snItemRepItem struct{ B []string }
	snItemRep     struct {
		B        string
		Sequence *snItemRepItem
	}
)

func (snAroundC) isSnAroundItem()            {}
func (snGroupG) isSnGroup()                  {}
func (snGroupM) isSnGroup()                  {}
func (snChoiceA) isSnChoice()                {}
func (snChoiceB) isSnChoice()                {}
func (snAltsAB) isSnAlts()                   {}
func (snAltsCA) isSnAlts()                   {}
func (snAltSeqABA) isSnAltSeq()              {}
func (snAltSeqB) isSnAltSeq()                {}
func (snE0GroupE0) isSnE0Group()             {}
func (snE0GroupE3) isSnE0Group()             {}
func (snE1GroupE1) isSnE1Group()             {}
func (snE1GroupE3) isSnE1Group()             {}
func (snSGroupS) isSnSGroup()                {}
func (snSGroupS1) isSnSGroup()               {}
func (snHeadTwiceAnyA) isSnHeadTwiceChoice() {}
func (snHeadTwiceSB) isSnHeadTwiceChoice()   {}
func (snG2GroupG2) isSnG2Group()             {}
func (snG2GroupM) isSnG2Group()              {}

// The branch sets of the shared-name interfaces.
var (
	snAroundBranches    = WithBranches[snAroundItem](snAroundC{})
	snGroupBranches     = WithBranches[snGroup](snGroupG{}, snGroupM{})
	snChoiceBranches    = WithBranches[snChoice](snChoiceA{}, snChoiceB{})
	snAltsBranches      = WithBranches[snAlts](snAltsAB{}, snAltsCA{})
	snAltSeqBranches    = WithBranches[snAltSeq](snAltSeqABA{}, snAltSeqB{})
	snE0Branches        = WithBranches[snE0Group](snE0GroupE0{}, snE0GroupE3{})
	snE1Branches        = WithBranches[snE1Group](snE1GroupE1{}, snE1GroupE3{})
	snSBranches         = WithBranches[snSGroup](snSGroupS{}, snSGroupS1{})
	snHeadTwiceBranches = WithBranches[snHeadTwiceChoice](snHeadTwiceAnyA{}, snHeadTwiceSB{})
	snG2Branches        = WithBranches[snG2Group](snG2GroupG2{}, snG2GroupM{})
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
		{snTwoHeadsSchema, []string{"type E0 string\n", "type E1 string\n", "type E3 string\n", "type R struct {\n\tE0 E0SubstitutionGroup\n\tE1 E1SubstitutionGroup\n}\n",
			branch("E0SubstitutionGroup", "E0SubstitutionGroupE0", "E0 E0"), branch("E0SubstitutionGroup", "E0SubstitutionGroupE3", "E3 E3"),
			branch("E1SubstitutionGroup", "E1SubstitutionGroupE1", "E1 E1"), branch("E1SubstitutionGroup", "E1SubstitutionGroupE3", "E3 E3")}},
		{snHeadTwiceSchema, []string{"type S string\n", "type S1 string\n", "type A string\n", "type B string\n", "type R struct {\n\tS      SSubstitutionGroup\n\tChoice RChoice\n}\n",
			branch("SSubstitutionGroup", "SSubstitutionGroupS", "S S"), branch("SSubstitutionGroup", "SSubstitutionGroupS1", "S1 S1"),
			"type RChoiceSequence struct {\n\tAny AnyElement\n\tA   A\n}\n\nfunc (RChoiceSequence) isRChoice() {}\n",
			"type RChoiceSequence2 struct {\n\tS SSubstitutionGroup\n\tB B\n}\n\nfunc (RChoiceSequence2) isRChoice() {}\n"}},
		{snGroupDeclGroupSchema, append([]string{"type G2 string\n", "type R struct {\n\tG  GSubstitutionGroup\n\tM  string\n\tG2 G2SubstitutionGroup\n}\n",
			branch("G2SubstitutionGroup", "G2SubstitutionGroupG2", "G2 G2"), branch("G2SubstitutionGroup", "G2SubstitutionGroupM", "M M")}, group...)},
		{snItemTwoSchema, []string{"type R struct {\n\tA        string\n\tSequence *RSequence\n}\n", "type RSequence struct {\n\tB  string\n\tC  string\n\tB2 string\n}\n"}},
		{snItemTwiceSchema, []string{"type R struct {\n\tA        string\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tB  string\n\tC  string\n\tB2 string\n}\n"}},
		{snItemBeforeSchema, []string{"type R struct {\n\tB        string\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tB string\n\tC string\n}\n"}},
		{snItemAfterSchema, []string{"type R struct {\n\tSequence []RSequence\n\tD        string\n\tB        string\n}\n", "type RSequence struct {\n\tB string\n\tC string\n}\n"}},
		{snItemRepSchema, []string{"type R struct {\n\tB        string\n\tSequence *RSequence\n}\n", "type RSequence struct {\n\tB []string\n}\n"}},
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
// in place, or across a content choice's alternatives, or two groups'
// interfaces over one member, of two heads or of one head reached twice,
// put each child to the field of the particle the Matcher attributes it to
// by position (§3.9.4.3 cvc-accept; §3.8.6.4 cos-nonambig), never by its
// name, a member of g's group to g's particle's field where the Matcher
// attributes it to that particle's position (cvc-accept clause 2.3.2).
// CheckDecode returns nil for each row's T.
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
		{"two heads' groups sharing a member, the member twice, s2_2_2v01's shape", snTwoHeadsSchema, "<s:e3>1</s:e3><s:e3>2</s:e3>", into[snTwoHeads](snE0Branches, snE1Branches),
			snTwoHeads{E0: snE0GroupE3{E3: "1"}, E1: snE1GroupE3{E3: "2"}}},
		{"two heads' groups sharing a member, the first head, then the member", snTwoHeadsSchema, "<s:e0>1</s:e0><s:e3>2</s:e3>", into[snTwoHeads](snE0Branches, snE1Branches),
			snTwoHeads{E0: snE0GroupE0{E0: "1"}, E1: snE1GroupE3{E3: "2"}}},
		{"two heads' groups sharing a member, the member, then the second head", snTwoHeadsSchema, "<s:e3>1</s:e3><s:e1>2</s:e1>", into[snTwoHeads](snE0Branches, snE1Branches),
			snTwoHeads{E0: snE0GroupE3{E3: "1"}, E1: snE1GroupE1{E1: "2"}}},
		{"one head reached twice, the second in a choice alternative, sns1b's shape", snHeadTwiceSchema, "<s:s>1</s:s><s:s1>2</s:s1><s:b>3</s:b>", into[snHeadTwice](snSBranches, snHeadTwiceBranches),
			snHeadTwice{S: snSGroupS{S: "1"}, Choice: snHeadTwiceSB{S: snSGroupS1{S1: "2"}, B: "3"}}},
		{"one head reached twice, the member first, then the head", snHeadTwiceSchema, "<s:s1>1</s:s1><s:s>2</s:s><s:b>3</s:b>", into[snHeadTwice](snSBranches, snHeadTwiceBranches),
			snHeadTwice{S: snSGroupS1{S1: "1"}, Choice: snHeadTwiceSB{S: snSGroupS{S: "2"}, B: "3"}}},
		{"one head reached twice, the wildcard's alternative, snea's shape", snHeadTwiceSchema, "<s:s>1</s:s><s:e>2</s:e><s:a>3</s:a>", into[snHeadTwice](snSBranches, snHeadTwiceBranches),
			snHeadTwice{S: snSGroupS{S: "1"}, Choice: snHeadTwiceAnyA{Any: anyElement{Space: "urn:s", Local: "e", Content: anyType{Text: "2"}}, A: "3"}}},
		{"a group's interface, a declaration and a second group's interface over one member", snGroupDeclGroupSchema, "<s:m>1</s:m><s:m>2</s:m><s:m>3</s:m>",
			into[snGroupDeclGroup](snGroupBranches, snG2Branches), snGroupDeclGroup{G: snGroupM{M: "1"}, M: "2", G2: snG2GroupM{M: "3"}}},
		{"a group's interface, a declaration and a second group's interface, each head", snGroupDeclGroupSchema, "<s:g>1</s:g><s:m>2</s:m><s:g2>3</s:g2>",
			into[snGroupDeclGroup](snGroupBranches, snG2Branches), snGroupDeclGroup{G: snGroupG{G: "1"}, M: "2", G2: snG2GroupG2{G2: "3"}}},
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

// TestDecodeSharedNameInItem pins the share at an occurrence field's item,
// which item 5 of Decode's doc lets decode: two element particles of one
// content model taking one expanded name, both inside the item or one inside
// it and one across its boundary, put each child to the field, and the item,
// of the particle the Matcher attributes it to by position (§3.9.4.3
// cvc-accept clause 3), never by its name — the two particles of one name
// carry distinct texts in every row, so a row whose second child goes to the
// first particle's field fails. Every enclosing nested group here occurs at
// most once, so cvc-accept admits one partition and its Note's "no fixed
// rule" does not arise. CheckDecode returns nil for each row's T.
func TestDecodeSharedNameInItem(t *testing.T) {
	rows := []struct {
		name, decls, body string
		into              target
		want              any
	}{
		{"two of an item's particles sharing a name, particlesQ030's shape", snItemTwoSchema, "<s:a>1</s:a><s:b>2</s:b><s:c>3</s:c><s:b>4</s:b>", into[snItemTwo](),
			snItemTwo{A: "1", Sequence: &snItemTwoItem{B: "2", C: "3", B2: "4"}}},
		{"two of a repeated item's particles sharing a name, each of two items", snItemTwiceSchema,
			"<s:a>1</s:a><s:b>2</s:b><s:c>3</s:c><s:b>4</s:b><s:b>5</s:b><s:c>6</s:c><s:b>7</s:b>", into[snItemTwice](),
			snItemTwice{A: "1", Sequence: []snItemTwoItem{{B: "2", C: "3", B2: "4"}, {B: "5", C: "6", B2: "7"}}}},
		{"an item's particle sharing an earlier sibling's name", snItemBeforeSchema, "<s:b>1</s:b><s:b>2</s:b><s:c>3</s:c>", into[snItemBefore](),
			snItemBefore{B: "1", Sequence: []snItemBC{{B: "2", C: "3"}}}},
		{"a sibling sharing an earlier item particle's name", snItemAfterSchema, "<s:b>1</s:b><s:c>2</s:c><s:d>3</s:d><s:b>4</s:b>", into[snItemAfter](),
			snItemAfter{Sequence: []snItemBC{{B: "1", C: "2"}}, D: "3", B: "4"}},
		{"a repeated item particle sharing an earlier sibling's name, particlesQ024's shape", snItemRepSchema, "<s:b>1</s:b><s:b>2</s:b><s:b>3</s:b>", into[snItemRep](),
			snItemRep{B: "1", Sequence: &snItemRepItem{B: []string{"2", "3"}}}},
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
