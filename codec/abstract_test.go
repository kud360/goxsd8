package codec

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsderr"
)

// The schemas of an element declaration whose complex {type definition} has
// {abstract} true, beside those nested_test.go declares. abstractKSchema
// declares ab, of empty content, and cc, its one member, adding k, and no
// element; each schema after it but the last four adds an s:r over it.
// abstractSeqSchema's r is a sequence of a, of ab, and an optional b, of cc;
// abstractMatcherSchema's a sequence of a, b, an xs:string, and a second
// optional a, which forces a Matcher (particleShape); abstractOccursSchema's a
// sequence of a and b occurring twice, an occurrence field; abstractFlatSchema's
// a repeated choice of a and b, a flattened particle. abstractSubstSchema's ab,
// of empty content, extends hb, of empty content too, and has cc, adding k,
// its one member, and r is a sequence of an <element ref> to h, of hb, the
// head of m, of ab. abstractChoiceSchema's ab has a choice of a and b, cc its one
// member adding nothing, and r is of ab. abstractLocalSchema's ab has a sequence of
// x, cc its one member adding nothing, and r a sequence of an optional a of ab.
// abstractAltSchema's r is of b0, which ab extends adding nothing, under a {type
// table} whose alternative selects ab, which has no member, where k is present;
// abstractAltMemberSchema adds cc, ab's one member, adding m. abstractRootSchema's
// ab, of empty content, has cc, adding nothing, its one member, and r is of ab;
// abstractParticleSchema's ab, of empty content, has no member, and r is a sequence
// of an optional a of ab. abstractTableSchema's r is of ab, of empty content,
// whose members are cc, adding k, and dd, adding m, under a {type table} whose
// alternative selects cc where k is present, its default ab.
const (
	abstractTableSchema = abstractKSchema + `<xs:complexType name="dd"><xs:complexContent><xs:extension base="s:ab"><xs:attribute name="m" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType>
		<xs:element name="r" type="s:ab"><xs:alternative test="@k" type="s:cc"/></xs:element>`
	abstractRootSchema     = `<xs:complexType name="ab" abstract="true"/><xs:complexType name="cc"><xs:complexContent><xs:extension base="s:ab"/></xs:complexContent></xs:complexType><xs:element name="r" type="s:ab"/>`
	abstractParticleSchema = `<xs:complexType name="ab" abstract="true"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="s:ab" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	abstractKSchema        = `<xs:complexType name="ab" abstract="true"/><xs:complexType name="cc"><xs:complexContent><xs:extension base="s:ab"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType>`
	abstractSeqSchema      = abstractKSchema + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="s:ab"/><xs:element name="b" type="s:cc" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	abstractMatcherSchema  = abstractKSchema + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="s:ab"/><xs:element name="b" type="xs:string"/><xs:element name="a" type="s:ab" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	abstractOccursSchema   = abstractKSchema + `<xs:element name="r"><xs:complexType><xs:sequence maxOccurs="2"><xs:element name="a" type="s:ab"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	abstractFlatSchema     = abstractKSchema + `<xs:element name="r"><xs:complexType><xs:choice maxOccurs="unbounded"><xs:element name="a" type="s:ab" maxOccurs="2"/><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	abstractSubstSchema    = `<xs:complexType name="hb"/><xs:complexType name="ab" abstract="true"><xs:complexContent><xs:extension base="s:hb"/></xs:complexContent></xs:complexType>
		<xs:complexType name="cc"><xs:complexContent><xs:extension base="s:ab"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType>
		<xs:element name="h" type="s:hb"/><xs:element name="m" type="s:ab" substitutionGroup="s:h"/>
		<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:h"/></xs:sequence></xs:complexType></xs:element>`
	abstractChoiceSchema = `<xs:complexType name="ab" abstract="true"><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:choice></xs:complexType>
		<xs:complexType name="cc"><xs:complexContent><xs:extension base="s:ab"/></xs:complexContent></xs:complexType><xs:element name="r" type="s:ab"/>`
	abstractLocalSchema = `<xs:complexType name="ab" abstract="true"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:complexType name="cc"><xs:complexContent><xs:extension base="s:ab"/></xs:complexContent></xs:complexType>
		<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="s:ab" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	abstractAltSchema = `<xs:complexType name="b0"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="k" type="xs:string"/></xs:complexType>
		<xs:complexType name="ab" abstract="true"><xs:complexContent><xs:extension base="s:b0"/></xs:complexContent></xs:complexType>
		<xs:element name="r" type="s:b0"><xs:alternative test="@k" type="s:ab"/></xs:element>`
	abstractAltMemberSchema = abstractAltSchema + `<xs:complexType name="cc"><xs:complexContent><xs:extension base="s:ab"><xs:attribute name="m" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType>`
)

// The types codegen emits for those schemas, each pinned by
// TestAbstractTypeShapesAreCodegens: abK is abstractKSchema's Cc; abSeqR,
// abMatchR, abOccR with abOccSeq, abFlatR with its interface and branches, and
// abSubR with its interface and branches are the R of the schema each is named
// for; abChR and its branches abstractChoiceSchema's R, RA and RB; abNoneR and
// abNone abstractParticleSchema's R and A, an interface with no branch;
// abLocalR and abLocalCc abstractLocalSchema's R and Cc; abAltR, abAltRCc,
// abAltCc, abAltRB0 and abB0 abstractAltMemberSchema's R, RCc, Cc, RB0 and B0;
// abTabR, abTabRCc and abTabRDd abstractTableSchema's R, RCc and RDd, abTabDd
// its Dd.
type (
	abK    struct{ K *string }
	abSeqR struct {
		A abK
		B *abK
	}
	abMatchR struct {
		A  abK
		B  string
		A2 *abK
	}
	abOccR   struct{ Sequence []abOccSeq }
	abOccSeq struct {
		A abK
		B string
	}
	abFlatR      struct{ Choice []abFlatChoice }
	abFlatChoice interface{ isAbFlatChoice() }
	abFlatA      struct{ A abK }
	abFlatB      struct{ B string }
	abSubR       struct{ H abSubGroup }
	abSubGroup   interface{ isAbSubGroup() }
	abSubH       struct{ H struct{} }
	abSubM       struct{ M abK }
	abChR        interface{ isAbChR() }
	abChRA       struct{ A string }
	abChRB       struct{ B int32 }
	abNoneR      struct{ A abNone }
	abNone       interface{ isAbNone() }
	abLocalR     struct{ A *abLocalCc }
	abLocalCc    struct{ X string }
	abAltR       interface{ isAbAltR() }
	abAltRCc     struct{ Cc abAltCc }
	abAltCc      struct {
		A string
		M *string
		K *string
	}
	abAltRB0 struct{ B0 abB0 }
	abTabR   interface{ isAbTabR() }
	abTabRCc struct{ Cc abK }
	abTabRDd struct{ Dd abTabDd }
	abTabDd  struct{ M *string }
	abB0     struct {
		A string
		K *string
	}
)

func (abFlatA) isAbFlatChoice() {}
func (abFlatB) isAbFlatChoice() {}
func (abSubH) isAbSubGroup()    {}
func (abSubM) isAbSubGroup()    {}
func (abChRA) isAbChR()         {}
func (abChRB) isAbChR()         {}
func (abAltRCc) isAbAltR()      {}
func (abAltRB0) isAbAltR()      {}
func (abTabRCc) isAbTabR()      {}
func (abTabRDd) isAbTabR()      {}

// TestAbstractTypeShapesAreCodegens pins the types above to those codegen
// emits, in shape (codegen/doc.go "An element declaration's governing
// types").
func TestAbstractTypeShapesAreCodegens(t *testing.T) {
	rows := []struct {
		decls string
		want  []string
	}{
		{abstractSeqSchema, []string{"type R struct {\n\tA Cc\n\tB *Cc\n}\n", "type Cc struct {\n\tK *string\n}\n"}},
		{abstractMatcherSchema, []string{"type R struct {\n\tA  Cc\n\tB  string\n\tA2 *Cc\n}\n"}},
		{abstractOccursSchema, []string{"type R struct {\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA Cc\n\tB string\n}\n"}},
		{abstractFlatSchema, []string{"type R struct {\n\tChoice []RChoice\n}\n", "type RChoiceA struct {\n\tA Cc\n}\n", "type RChoiceB struct {\n\tB string\n}\n"}},
		{abstractSubstSchema, []string{"type R struct {\n\tH HSubstitutionGroup\n}\n", "type H struct{}\n", "type M struct {\n\tK *string\n}\n",
			"type HSubstitutionGroupH struct {\n\tH H\n}\n", "type HSubstitutionGroupM struct {\n\tM M\n}\n"}},
		{abstractChoiceSchema, []string{"type R interface {\n\tisR()\n}\n", "type RA struct {\n\tA string\n}\n", "type RB struct {\n\tB int32\n}\n"}},
		{abstractKSchema + `<xs:element name="r" type="s:ab" nillable="true"/>`, []string{"type R struct {\n\tK *string\n}\n", "type RAttributes struct {\n\tK *string\n}\n"}},
		{abstractParticleSchema, []string{"type R struct {\n\tA A\n}\n", "type A interface {\n\tisA()\n}\n"}},
		{abstractRootSchema, []string{"type R struct{}\n"}},
		{abstractLocalSchema, []string{"type R struct {\n\tA *Cc\n}\n", "type Cc struct {\n\tX string\n}\n"}},
		{abstractAltSchema, []string{"type R struct {\n\tA string\n\tK *string\n}\n"}},
		{abstractAltMemberSchema, []string{"type R interface {\n\tisR()\n}\n", "type RCc struct {\n\tCc Cc\n}\n", "type Cc struct {\n\tA string\n\tM *string\n\tK *string\n}\n",
			"type RB0 struct {\n\tB0 B0\n}\n", "type B0 struct {\n\tA string\n\tK *string\n}\n"}},
		{abstractTableSchema, []string{"type R interface {\n\tisR()\n}\n", "type RCc struct {\n\tCc Cc\n}\n", "type RDd struct {\n\tDd Dd\n}\n",
			"type Cc struct {\n\tK *string\n}\n", "type Dd struct {\n\tM *string\n}\n"}},
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

// TestDecodeAbstractTypeChild pins an element of a declaration whose complex
// {type definition} has {abstract} true and no {type table} decoding, under an
// xsi:type naming a member, at each site that reads a child's xsi:type
// (checkChildXsiType): a content model attributed by name and by a Matcher
// (eachChild), an occurrence field's (boundOccurrence), a flattened particle's
// (boundFlattened), and a member of a substitution group whose head's
// particle it is attributed to, its governing declaration the member's own
// (cvc-accept clause 2.3.2). A declaration with no member gives the
// interface alone, whose one value is nil, and the {type table} alternative
// holding an abstract type with no member gives nothing, so its declaration's
// one governing type is its {default type definition}.
func TestDecodeAbstractTypeChild(t *testing.T) {
	rows := []struct {
		name    string
		decls   string
		content string
		attrs   string
		into    target
		want    any
	}{
		{"a content model attributed by name", abstractSeqSchema, `<s:a xsi:type="s:cc" k="v"/>`, "", into[abSeqR](), abSeqR{A: abK{K: strPtr("v")}}},
		{"a content model attributed by a Matcher", abstractMatcherSchema, `<s:a xsi:type="s:cc"/><s:b>x</s:b><s:a xsi:type="s:cc" k="v"/>`, "",
			into[abMatchR](), abMatchR{B: "x", A2: &abK{K: strPtr("v")}}},
		{"an occurrence field's item", abstractOccursSchema, `<s:a xsi:type="s:cc" k="v"/><s:b>x</s:b><s:a xsi:type="s:cc"/><s:b>y</s:b>`, "",
			into[abOccR](), abOccR{Sequence: []abOccSeq{{A: abK{K: strPtr("v")}, B: "x"}, {B: "y"}}}},
		{"a flattened particle", abstractFlatSchema, `<s:b>x</s:b><s:a xsi:type="s:cc" k="v"/>`, "",
			into[abFlatR](WithBranches[abFlatChoice](abFlatA{}, abFlatB{})), abFlatR{Choice: []abFlatChoice{abFlatB{B: "x"}, abFlatA{A: abK{K: strPtr("v")}}}}},
		{"a substitution group's member", abstractSubstSchema, `<s:m xsi:type="s:cc" k="v"/>`, "",
			into[abSubR](WithBranches[abSubGroup](abSubH{}, abSubM{})), abSubR{H: abSubM{M: abK{K: strPtr("v")}}}},
		{"an optional local element of an abstract type with one member, present", abstractLocalSchema, `<s:a xsi:type="s:cc"><s:x>v</s:x></s:a>`, "",
			into[abLocalR](), abLocalR{A: &abLocalCc{X: "v"}}},
		{"an optional local element of an abstract type with one member, absent", abstractLocalSchema, "", "", into[abLocalR](), abLocalR{}},
		{"an optional local element of an abstract type with no member, absent", abstractParticleSchema, "", "",
			into[abNoneR](WithBranches[abNone]()), abNoneR{}},
		{"a {type table} alternative of an abstract type with no member, the default selected", abstractAltSchema, `<s:a>x</s:a>`, "",
			into[struct {
				A string
				K *string
			}](), struct {
				A string
				K *string
			}{A: "x"}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf(row.attrs, row.content)
			mustAccept(t, schema, data)
			got, err := row.into.decode(schema, data)
			if err != nil {
				t.Fatalf("Decode = %v", err)
			}
			if !reflect.DeepEqual(got, row.want) {
				t.Errorf("Decode = %#v, want %#v", got, row.want)
			}
			if err := row.into.check(schema); err != nil {
				t.Errorf("CheckDecode = %v, want nil", err)
			}
		})
	}
}

// TestDecodeAbstractTypeGuards pins what an abstract complex type does not
// decode. An xsi:type is admitted by the element's governing declaration
// (admitsXsiType), never by its bound unit: a {type table} alternative
// holding an abstract type whose instance names its one member declines by
// xsi:type, as every {type table} declaration's does, an abstract declared
// type's included, and so does b, of cc, the member a's unit binds, beside a,
// of ab, and h, the head, of a concrete type, whose particle m's element, of
// ab, is attributed to too. An xsi:type naming the abstract type itself or no
// member, and an element of an abstract type with no xsi:type, are charged by
// validate, which Decode runs first, and decode into nothing.
func TestDecodeAbstractTypeGuards(t *testing.T) {
	declines := []struct {
		name    string
		decls   string
		attrs   string
		content string
		into    target
		col     func(data []byte) int
	}{
		{"a {type table} alternative of an abstract type, naming its member", abstractAltMemberSchema, ` k="x" xsi:type="s:cc"`, `<s:a>x</s:a>`,
			into[abAltR](WithBranches[abAltR](abAltRCc{}, abAltRB0{})), func([]byte) int { return 1 }},
		{"an abstract {type definition} under a {type table}, its default selected, naming a member", abstractTableSchema, ` xsi:type="s:dd" m="w"`, "",
			into[abTabR](WithBranches[abTabR](abTabRCc{}, abTabRDd{})), func([]byte) int { return 1 }},
		{"a child of the member's type beside one of the abstract type", abstractSeqSchema, "", `<s:a xsi:type="s:cc"/><s:b xsi:type="s:cc"/>`,
			into[abSeqR](), func(data []byte) int { return bytes.Index(data, []byte("<s:b")) + 1 }},
		{"a substitution group's concrete head, its member's type abstract", abstractSubstSchema, "", `<s:h xsi:type="s:cc" k="v"/>`,
			into[abSubR](WithBranches[abSubGroup](abSubH{}, abSubM{})), func(data []byte) int { return bytes.Index(data, []byte("<s:h")) + 1 }},
	}
	for _, row := range declines {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			data := instanceOf(row.attrs, row.content)
			mustAccept(t, schema, data)
			if err := row.into.check(schema); err != nil {
				t.Fatalf("CheckDecode = %v, want nil", err)
			}
			got, err := row.into.decode(schema, data)
			if got != nil {
				t.Fatalf("Decode = %v, want no value", got)
			}
			assertDecline(t, err, constructXsiType, 1, row.col(data))
		})
	}
	nonMember := abstractTypeOneSchema + `<xs:complexType name="o"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="k" type="xs:string"/></xs:complexType>`
	charged := []struct {
		name    string
		decls   string
		attrs   string
		content string
		into    target
		rule    string
	}{
		{"the document element naming the abstract type itself", abstractTypeOneSchema, ` xsi:type="s:ab"`, `<s:a>x</s:a>`, into[abOneR](), "cvc-type"},
		{"the document element naming a type that is no member", nonMember, ` xsi:type="s:o" k="v"`, `<s:a>x</s:a>`, into[abOneR](), "cvc-elt"},
		{"the document element with no xsi:type", abstractTypeOneSchema, "", `<s:a>x</s:a>`, into[abOneR](), "cvc-type"},
		{"a child of the abstract type with no xsi:type", abstractLocalSchema, "", `<s:a><s:x>v</s:x></s:a>`, into[abLocalR](), "cvc-type"},
		{"a {type table} alternative of an abstract type with no member, selected", abstractAltSchema, ` k="x"`, `<s:a>x</s:a>`, into[struct {
			A string
			K *string
		}](), "cvc-type"},
	}
	for _, row := range charged {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			got, err := row.into.decode(schema, instanceOf(row.attrs, row.content))
			if got != nil {
				t.Fatalf("Decode = %v, want no value", got)
			}
			var de *DeclineError
			if rule, _ := xsderr.RuleOf(err); string(rule) != row.rule || errors.As(err, &de) {
				t.Errorf("Decode error = %v, want validate's %s", err, row.rule)
			}
		})
	}
}

// TestInstanceSelectorUnmatched pins instanceSelector.selected where no
// instance Decode validated reaches it: an element of abstractTypeTwoSchema's
// r with no xsi:type, with one that resolves to no type definition, and with
// one naming ab itself, which is none of r's governing types. Each is a plain
// error naming the element's instance Loc and pointing at key-overrides.
func TestInstanceSelectorUnmatched(t *testing.T) {
	schema := schemaOf(t, abstractTypeTwoSchema)
	decl, ok := schema.Element(rootName)
	if !ok {
		t.Fatal("the schema declares no s:r")
	}
	types, _, err := governingTypes(schema, decl, true)
	if err != nil {
		t.Fatal(err)
	}
	tr := &tableRoot{selector: instanceSelector{}, types: types}
	for _, row := range []struct {
		attrs string
		want  string
	}{
		{"", "codec: ?:1:1: element {urn:s}r carries no xsi:type, but one overriding its {type definition} (key-overrides) is required"},
		{` xsi:type="s:zz"`, `codec: ?:1:1: element {urn:s}r: its xsi:type "s:zz" names no type definition, but one overriding its {type definition} (key-overrides) is required`},
		{` xsi:type="s:ab"`, `codec: ?:1:1: element {urn:s}r: its xsi:type "s:ab" names {urn:s}ab, none of its governing types, the types that override its {type definition} (key-overrides)`},
	} {
		start, err := documentElement(xmltree.NewReader("", bytes.NewReader(instanceOf(row.attrs, ""))))
		if err != nil {
			t.Fatal(err)
		}
		i, err := instanceSelector{}.selected(tr, &pipeline{schema: schema}, start)
		if err == nil || err.Error() != row.want {
			t.Errorf("selected(%q) = %d, %v; want the error %q", row.attrs, i, err, row.want)
		}
	}
}
