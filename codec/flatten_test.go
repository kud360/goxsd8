package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// The flatten schemas, each declaring s:r, whose content {particle} codegen
// flattens (codegen/doc.go "# Particles"): flatSeqSchema's is
// sequence(a?, b?) repeated, a repeated emptiable sequence; flatChoiceSchema's
// choice(a+, b) repeated, a repeated model group with a repeating edge, beside
// attributes named sequence and choice; flatOptSchema's the optional
// choice(a, sequence(b?)), whose model-group alternative is emptiable;
// flatRefSchema's a <group ref> occurring twice to sequence(a?, b?);
// flatWildSchema's sequence(a+, any?) repeated; flatPickSchema's choice(a+,
// any) repeated, its wildcard ##any; flatHeadSchema's sequence(h?, b?)
// repeated, h an <element ref> to the head of m's substitution group; and
// flatSharedSchema's sequence(sequence(c, a)?, sequence(d, a)?) repeated, two
// local declarations of a in one reach. flatOnceSchema's sequence(a?, b?)
// occurs exactly once, so codegen gives it its own fields and no slice,
// though flatSeqSchema's repeated one is flattened.
const (
	flatOnceSchema   = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	flatSeqSchema    = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	flatChoiceSchema = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/><xs:element name="b" type="xs:int"/></xs:choice>
  <xs:attribute name="sequence" type="xs:string"/><xs:attribute name="choice" type="xs:string"/></xs:complexType></xs:element>`
	flatOptSchema  = `<xs:element name="r"><xs:complexType><xs:choice minOccurs="0"><xs:element name="a" type="xs:string"/><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:choice></xs:complexType></xs:element>`
	flatRefSchema  = `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:group>` + `<xs:element name="r"><xs:complexType><xs:group ref="s:g" maxOccurs="2"/></xs:complexType></xs:element>`
	flatWildSchema = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/><xs:any minOccurs="0" processContents="lax"/></xs:sequence></xs:complexType></xs:element>`
	flatPickSchema = `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string" maxOccurs="unbounded"/><xs:any processContents="lax"/></xs:choice></xs:complexType></xs:element>`
	flatHeadSchema = `<xs:element name="h" type="xs:string"/><xs:element name="m" type="xs:string" substitutionGroup="s:h"/>` +
		`<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:element ref="s:h" minOccurs="0"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	flatSharedSchema = `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="unbounded"><xs:sequence minOccurs="0"><xs:element name="c" type="xs:string"/><xs:element name="a" type="xs:string"/></xs:sequence>` +
		`<xs:sequence minOccurs="0"><xs:element name="d" type="xs:string"/><xs:element name="a" type="xs:string"/></xs:sequence></xs:sequence></xs:complexType></xs:element>`
)

// The types codegen emits for the flatten schemas' s:r, in shape, which
// TestFlattenTypesAreCodegens pins: flatSeq for flatSeqSchema and
// flatRefSchema, flatChoice for flatChoiceSchema, flatOpt for flatOptSchema,
// flatWild for flatWildSchema, flatPick for flatPickSchema and flatHead, with
// its head's interface flatGroup, for flatHeadSchema, and flatOnce for
// flatOnceSchema.
type (
	flatOnce struct{ A, B *string }

	flatSeq     struct{ Sequence []flatSeqItem }
	flatSeqItem interface{ isFlatSeqItem() }
	flatSeqA    struct{ A string }
	flatSeqB    struct{ B string }

	flatChoice struct {
		Choice   []flatChoiceItem
		Sequence *string
		Choice2  *string
	}
	flatChoiceItem interface{ isFlatChoiceItem() }
	flatChoiceA    struct{ A string }
	flatChoiceB    struct{ B int32 }

	flatOpt     struct{ Choice []flatOptItem }
	flatOptItem interface{ isFlatOptItem() }
	flatOptA    struct{ A string }
	flatOptB    struct{ B string }

	flatWild     struct{ Sequence []flatWildItem }
	flatWildItem interface{ isFlatWildItem() }
	flatWildA    struct{ A string }
	flatWildAny  struct{ Any anyElement }

	flatPick     struct{ Choice []flatPickItem }
	flatPickItem interface{ isFlatPickItem() }
	flatPickA    struct{ A string }
	flatPickAny  struct{ Any anyElement }

	flatHead       struct{ Sequence []flatHeadItem }
	flatHeadItem   interface{ isFlatHeadItem() }
	flatHeadH      struct{ H flatGroup }
	flatHeadB      struct{ B string }
	flatGroup      interface{ isFlatGroup() }
	flatGroupH     struct{ H flatGroupHName }
	flatGroupM     struct{ M flatGroupMName }
	flatGroupHName string
	flatGroupMName string
)

func (flatSeqA) isFlatSeqItem()       {}
func (flatSeqB) isFlatSeqItem()       {}
func (flatChoiceA) isFlatChoiceItem() {}
func (flatChoiceB) isFlatChoiceItem() {}
func (flatOptA) isFlatOptItem()       {}
func (flatOptB) isFlatOptItem()       {}
func (flatWildA) isFlatWildItem()     {}
func (flatWildAny) isFlatWildItem()   {}
func (flatPickA) isFlatPickItem()     {}
func (flatPickAny) isFlatPickItem()   {}
func (flatHeadH) isFlatHeadItem()     {}
func (flatHeadB) isFlatHeadItem()     {}
func (flatGroupH) isFlatGroup()       {}
func (flatGroupM) isFlatGroup()       {}

// The branch sets of the flatten types' interfaces.
var (
	flatSeqBranches    = WithBranches[flatSeqItem](flatSeqA{}, flatSeqB{})
	flatChoiceBranches = WithBranches[flatChoiceItem](flatChoiceA{}, flatChoiceB{})
	flatOptBranches    = WithBranches[flatOptItem](flatOptA{}, flatOptB{})
	flatWildBranches   = WithBranches[flatWildItem](flatWildA{}, flatWildAny{})
	flatPickBranches   = WithBranches[flatPickItem](flatPickA{}, flatPickAny{})
	flatHeadBranches   = WithBranches[flatHeadItem](flatHeadH{}, flatHeadB{})
	flatGroupBranches  = WithBranches[flatGroup](flatGroupH{}, flatGroupM{})
)

// TestFlattenTypesAreCodegens pins the types above to those codegen emits for
// each flatten schema, in shape, so that the rows decoding into them decode
// into codegen's types; flatSharedSchema's two branches of field A, of one
// shape, are the decline TestDecodeFlattenSharedBranch pins.
func TestFlattenTypesAreCodegens(t *testing.T) {
	branch := func(iface, name, field string) string {
		return "type " + name + " struct {\n\t" + field + "\n}\n\nfunc (" + name + ") is" + iface + "() {}\n"
	}
	rows := []struct {
		decls string
		want  []string
	}{
		{flatOnceSchema, []string{"type R struct {\n\tA *string\n\tB *string\n}\n"}},
		{flatSeqSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceA", "A string"), branch("RSequence", "RSequenceB", "B string")}},
		{flatRefSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceA", "A string"), branch("RSequence", "RSequenceB", "B string")}},
		{flatChoiceSchema, []string{"type R struct {\n\tChoice   []RChoice\n\tSequence *string\n\tChoice2  *string\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B int32")}},
		{flatOptSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceB", "B string")}},
		{flatWildSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceA", "A string"), branch("RSequence", "RSequenceAny", "Any AnyElement")}},
		{flatPickSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", branch("RChoice", "RChoiceA", "A string"), branch("RChoice", "RChoiceAny", "Any AnyElement")}},
		{flatHeadSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceH", "H HSubstitutionGroup"), branch("RSequence", "RSequenceB", "B string"),
			branch("HSubstitutionGroup", "HSubstitutionGroupH", "H H"), branch("HSubstitutionGroup", "HSubstitutionGroupM", "M M")}},
		{flatSharedSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", branch("RSequence", "RSequenceA", "A string"), branch("RSequence", "RSequenceA2", "A string")}},
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

// TestDecodeFlattened pins the flatten lift: a content sequence or choice
// codegen flattens decodes into its slice field, one item per child element
// in document order, each the branch of the basic particle the Matcher
// attributes the child to, an element particle before a wildcard
// (§3.8.4.2 key-vpath), a substitution group's member to its head's particle
// (cvc-accept clause 2.3.2), and no grouping into occurrences kept
// (cvc-accept clause 3). An absent optional choice is the nil slice. The
// guard row's sequence(a?, b?) occurring exactly once decodes into its own
// fields. CheckDecode returns nil for each row's T.
func TestDecodeFlattened(t *testing.T) {
	str := func(s string) *string { return &s }
	w := func(space string) anyElement { return anyElement{Space: space, Local: "w"} }
	rows := []struct {
		name, decls, attrs, body string
		into                     target
		want                     any
	}{
		{"guard: an emptiable sequence occurring once", flatOnceSchema, "", "<s:b>2</s:b>", into[flatOnce](), flatOnce{B: str("2")}},
		{"a repeated emptiable sequence", flatSeqSchema, "", "<s:a>1</s:a><s:b>2</s:b><s:b>3</s:b><s:a>4</s:a>", into[flatSeq](flatSeqBranches),
			flatSeq{Sequence: []flatSeqItem{flatSeqA{A: "1"}, flatSeqB{B: "2"}, flatSeqB{B: "3"}, flatSeqA{A: "4"}}}},
		{"a repeated choice with a repeating edge, beside attributes", flatChoiceSchema, ` choice="c" sequence="q"`, "<s:a>x</s:a><s:a>y</s:a><s:b>1</s:b><s:a>z</s:a><s:b>2</s:b>", into[flatChoice](flatChoiceBranches),
			flatChoice{Choice: []flatChoiceItem{flatChoiceA{A: "x"}, flatChoiceA{A: "y"}, flatChoiceB{B: 1}, flatChoiceA{A: "z"}, flatChoiceB{B: 2}}, Sequence: str("q"), Choice2: str("c")}},
		{"an optional choice with an emptiable model-group alternative, absent", flatOptSchema, "", "", into[flatOpt](flatOptBranches), flatOpt{}},
		{"an optional choice with an emptiable model-group alternative, its group's", flatOptSchema, "", "<s:b>v</s:b>", into[flatOpt](flatOptBranches),
			flatOpt{Choice: []flatOptItem{flatOptB{B: "v"}}}},
		{"an optional choice with an emptiable model-group alternative, its element's", flatOptSchema, "", "<s:a>v</s:a>", into[flatOpt](flatOptBranches),
			flatOpt{Choice: []flatOptItem{flatOptA{A: "v"}}}},
		{"a <group ref> occurring twice", flatRefSchema, "", "<s:a>1</s:a><s:b>2</s:b><s:a>3</s:a>", into[flatSeq](flatSeqBranches),
			flatSeq{Sequence: []flatSeqItem{flatSeqA{A: "1"}, flatSeqB{B: "2"}, flatSeqA{A: "3"}}}},
		{"a wildcard in the reach", flatWildSchema, "", `<s:a>1</s:a><s:a>2</s:a><o:w xmlns:o="urn:o"/><s:a>3</s:a>`, into[flatWild](flatWildBranches),
			flatWild{Sequence: []flatWildItem{flatWildA{A: "1"}, flatWildA{A: "2"}, flatWildAny{Any: w("urn:o")}, flatWildA{A: "3"}}}},
		{"an element particle taking a child before a wildcard", flatPickSchema, "", `<s:a>1</s:a><o:w xmlns:o="urn:o"/><s:a>2</s:a>`, into[flatPick](flatPickBranches),
			flatPick{Choice: []flatPickItem{flatPickA{A: "1"}, flatPickAny{Any: w("urn:o")}, flatPickA{A: "2"}}}},
		{"a substitution group's member in its head's branch", flatHeadSchema, "", "<s:m>1</s:m><s:b>2</s:b><s:h>3</s:h>", into[flatHead](flatHeadBranches, flatGroupBranches),
			flatHead{Sequence: []flatHeadItem{flatHeadH{H: flatGroupM{M: "1"}}, flatHeadB{B: "2"}, flatHeadH{H: flatGroupH{H: "3"}}}}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf(row.attrs, row.body)
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

// TestDecodeFlattenedXsiType pins item 7 inside a flattened content model:
// a child attributed to an element particle carrying xsi:type declines at its
// own Loc, and one attributed to a wildcard particle is read raw, its
// xsi:type one of its raw attributes.
func TestDecodeFlattenedXsiType(t *testing.T) {
	schema := schemaOf(t, flatWildSchema)
	declined := instanceOf("", `<s:a xsi:type="xs:string">1</s:a>`)
	mustAccept(t, schema, declined)
	_, err := Decode[flatWild](schema, strict.New(), declined, flatWildBranches)
	var de *DeclineError
	if !errors.As(err, &de) || de.Construct() != (Construct{constructXsiType}) {
		t.Fatalf("Decode = %v, want the xsi:type decline", err)
	}
	if loc, _ := de.Loc(); loc.Line != 1 || loc.Col != 120 {
		t.Errorf("decline Loc = %s, want the child's 1:120", loc)
	}
	raw := instanceOf("", `<s:a>1</s:a><o:w xmlns:o="urn:o" xsi:type="xs:string"/>`)
	mustAccept(t, schema, raw)
	got, err := Decode[flatWild](schema, strict.New(), raw, flatWildBranches)
	if err != nil {
		t.Fatal(err)
	}
	want := flatWild{Sequence: []flatWildItem{flatWildA{A: "1"}, flatWildAny{Any: anyElement{Space: "urn:o", Local: "w",
		Content: anyType{Attributes: []anyAttribute{{Space: "http://www.w3.org/2001/XMLSchema-instance", Local: "type", Value: "xs:string"}}}}}}}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("Decode = %#v, want %#v", *got, want)
	}
}

// TestDecodeFlattenSharedBranch pins item 5 for a flattened reach holding two
// local declarations of one expanded name: their {type definition}s are one
// (Structures §3.8.6.3 cos-element-consistent), so their branch structs have
// one shape and decline as constructSharedBranch, and no expanded name an
// earlier particle takes declines as constructSharedName, since position,
// not name, attributes a child. Decode and CheckDecode decline alike.
func TestDecodeFlattenSharedBranch(t *testing.T) {
	schema := schemaOf(t, flatSharedSchema)
	data := instanceOf("", "<s:c>1</s:c><s:a>2</s:a><s:d>3</s:d><s:a>4</s:a>")
	mustAccept(t, schema, data)
	_, decoded := Decode[struct{}](schema, strict.New(), data)
	for _, got := range []struct {
		name string
		err  error
	}{{"Decode", decoded}, {"CheckDecode", CheckDecode[struct{}](schema, rootName)}} {
		var de *DeclineError
		if !errors.As(got.err, &de) || de.Construct() != (Construct{constructSharedBranch}) {
			t.Errorf("%s = %v, want a *DeclineError of %v", got.name, got.err, Construct{constructSharedBranch})
		}
	}
}

// TestCheckDecodeUntrackedFlattened pins the flattened arm's ContentMatcher
// decline: (a{1,2}){1,21845}, flattened by its repeating edge, is a content
// model xsd.Schema.ContentMatcher declines as too wide to track, so
// CheckDecode returns constructUntrackedModel, a *DeclineError and no
// plain error. Validation leaves such an instance undecided (item 4), so no
// Decode row reaches it.
func TestCheckDecodeUntrackedFlattened(t *testing.T) {
	schema := schemaOf(t, `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="21845"><xs:element name="a" type="xs:string" maxOccurs="2"/></xs:sequence></xs:complexType></xs:element>`)
	err := CheckDecode[struct{}](schema, rootName)
	var de *DeclineError
	if !errors.As(err, &de) || de.Construct() != (Construct{constructUntrackedModel}) {
		t.Fatalf("CheckDecode = %v, want a *DeclineError of %v", err, Construct{constructUntrackedModel})
	}
	const prefix = "codec does not decode a content model the content matcher does not track: the complex type defined at "
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Errorf("CheckDecode = %q, want the prefix %q", err, prefix)
	}
}

// TestCheckDecodeFlattenedMaxOccursZero pins item 5's plain error for a
// particle with {max occurs} 0 inside a flattened reach: type v's content is
// sequence(a?, any) repeated, the wildcard rebuilt through xsd.NewParticle
// with the vacuous Occurs{0,0} (no schema document yields one, §3.10.2), so
// CheckDecode's error names it wildcard particle 1, and is no decline.
func TestCheckDecodeFlattenedMaxOccursZero(t *testing.T) {
	parsed := schemaOf(t, `<xs:complexType name="v"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:any namespace="##other" processContents="lax"/></xs:sequence></xs:complexType><xs:element name="r" type="s:v"/>`)
	v := xsd.QName{Space: "urn:s", Local: "v"}
	declared, ok := parsed.Type(v)
	if !ok {
		t.Fatalf("the parsed schema declares no type %s", v)
	}
	content := declared.(xsd.ComplexType).ContentType().(xsd.ElementContent)
	particles := content.Particle.Term().(xsd.ResolvedTerm).Term.(xsd.ModelGroup).Particles()
	none, err := xsd.NewOccurs(xsderr.Loc{}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	vacuous, err := xsd.NewParticle(xsderr.Loc{}, none, particles[1].Term())
	if err != nil {
		t.Fatal(err)
	}
	group, err := xsd.NewModelGroup(xsderr.Loc{}, xsd.CompositorSequence, []xsd.Particle{particles[0], vacuous})
	if err != nil {
		t.Fatal(err)
	}
	particle, err := xsd.NewParticle(xsderr.Loc{}, content.Particle.Occurs(), xsd.ResolvedTerm{Term: group})
	if err != nil {
		t.Fatal(err)
	}
	ct, err := xsd.NewComplexType(declared.Loc(), v, xsd.QName{Space: xsd.XMLSchemaNS, Local: "anyType"}, nil, xsd.DerivationRestriction, false,
		nil, nil, nil, xsd.ElementContent{Particle: particle}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = CheckDecode[struct{}](replacedSchema(t, parsed, ct), rootName)
	const prefix, suffix = "codec: wildcard particle 1 of the {type definition} of element {urn:s}r", " has {max occurs} 0, for which codegen emits no type"
	var de *DeclineError
	if err == nil || errors.As(err, &de) || !strings.HasPrefix(err.Error(), prefix) || !strings.HasSuffix(err.Error(), suffix) {
		t.Errorf("CheckDecode = %v, want the plain error %q ... %q", err, prefix, suffix)
	}
}
