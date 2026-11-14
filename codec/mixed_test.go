package codec

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/xsd"
)

// mixedChild is of the shape of codegen's MixedChild[T], which
// TestMixedTypesAreCodegens pins; its name differs from codegen's, which
// item 6 never compares.
type mixedChild[T any] struct {
	Element T
	Tail    string
}

// The mixed-content schemas, each declaring s:r of a type whose content is
// mixed (§3.4.1) unless named otherwise: mxSeqSchema's is sequence(a);
// mxEmptySchema's has an empty {particle} and an attribute k
// (dcl.ctd.ctcc.common clause 3.1.1); mxOptSchema's is sequence(a?);
// mxWildSchema's is sequence(any); mxExtSchema's r is of an extension of a
// mixed base, sequence(a), declaring mixed="true" and adding sequence(b),
// and mxExtAttrSchema's of one adding only an attribute, which is mixed by
// clause 4.2.2 without mixed="true"; mxOpenSchema's is sequence(a) under an
// {open content}; mxPairSchema's is sequence(a, b); mxPickSchema's is
// choice(a, b) occurring once, which codegen gives a struct rather than the
// choice's interface; mxAnyTypeSchema's is sequence(a), a of type
// xs:anyType; mxRepChoiceSchema's is that choice repeated, held in an
// occurrence field of the choice's interface; mxFlatSchema's is sequence(a?,
// b?) repeated, which codegen flattens; mxSealedOptSchema's is sequence(a?),
// a of a type whose content is a choice, which codegen gives a bare sealed
// interface; mxOccSchema's is sequence(a, b) repeated, held in an occurrence
// field; mxDefaultSchema's is sequence(a?) under a default on r; and
// mxAnyRestrictSchema's r is of a mixed restriction of xs:anyType,
// sequence(b?); and mxNameAnySchema's is a choice of a wildcard and an
// element any of xs:string, whose branches hold AnyChild and
// MixedChild[string], two Go types. Of the guards, mxStrictSchema's r is of
// an element-only restriction of a mixed base, and elementOnlyPairSchema's
// is mxPairSchema's sequence with element-only content.
const (
	mxSeqSchema           = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	mxEmptySchema         = `<xs:element name="r"><xs:complexType mixed="true"><xs:attribute name="k" type="xs:string"/></xs:complexType></xs:element>`
	mxOptSchema           = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	mxWildSchema          = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:any processContents="lax"/></xs:sequence></xs:complexType></xs:element>`
	mxMixedBase           = `<xs:complexType name="base" mixed="true"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`
	mxExtSchema           = mxMixedBase + `<xs:complexType name="ext" mixed="true"><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="b" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType><xs:element name="r" type="s:ext"/>`
	mxExtAttrSchema       = mxMixedBase + `<xs:complexType name="ext"><xs:complexContent><xs:extension base="s:base"><xs:attribute name="k" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType><xs:element name="r" type="s:ext"/>`
	mxOpenSchema          = `<xs:element name="r"><xs:complexType mixed="true"><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
	mxPairSchema          = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:sequence></xs:complexType></xs:element>`
	mxPickSchema          = `<xs:element name="r"><xs:complexType mixed="true"><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:choice></xs:complexType></xs:element>`
	mxAnyTypeSchema       = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" type="xs:anyType"/></xs:sequence></xs:complexType></xs:element>`
	mxRepChoiceSchema     = `<xs:element name="r"><xs:complexType mixed="true"><xs:choice maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:choice></xs:complexType></xs:element>`
	mxFlatSchema          = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:int" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	mxSealedOptSchema     = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" minOccurs="0"><xs:complexType><xs:choice><xs:element name="x" type="xs:string"/><xs:element name="y" type="xs:int"/></xs:choice></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
	mxOccSchema           = `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:sequence></xs:complexType></xs:element>`
	mxDefaultSchema       = `<xs:element name="r" default="d"><xs:complexType mixed="true"><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	mxAnyRestrict         = `<xs:complexType name="dt" mixed="true"><xs:complexContent><xs:restriction base="xs:anyType"><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`
	mxAnyRestrictSchema   = mxAnyRestrict + `<xs:element name="r" type="s:dt"/>`
	mxNameAnySchema       = `<xs:element name="r"><xs:complexType mixed="true"><xs:choice><xs:any namespace="##other" processContents="lax"/><xs:element name="any" type="xs:string"/></xs:choice></xs:complexType></xs:element>`
	mxStrictSchema        = mxMixedBase + `<xs:complexType name="strict"><xs:complexContent><xs:restriction base="s:base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType><xs:element name="r" type="s:strict"/>`
	elementOnlyPairSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:sequence></xs:complexType></xs:element>`
)

// The types codegen emits for those schemas' s:r, which TestMixedTypesAreCodegens
// pins: mxSeq mxSeqSchema's; mxEmpty mxEmptySchema's; mxOpt mxOptSchema's and
// mxDefaultSchema's; mxWild mxWildSchema's; mxPair mxExtSchema's and mxPairSchema's;
// mxExtAttr mxExtAttrSchema's; mxOpen mxOpenSchema's; mxPick, mxPickA and mxPickB
// mxPickSchema's, and mxRepChoice mxRepChoiceSchema's over the same interface; mxAny
// mxAnyTypeSchema's; mxFlat and its branches mxFlatA and mxFlatB mxFlatSchema's;
// mxSealedOpt and mxAI's branches mxAX and mxAY mxSealedOptSchema's; mxOcc and its
// item mxOccItem mxOccSchema's; mxB mxAnyRestrictSchema's; mxNameAny and its
// branches mxNameAnyAny and mxNameAnyAny2 mxNameAnySchema's; and of the guards,
// mxStrict mxStrictSchema's and elementOnlyPair elementOnlyPairSchema's.
type (
	mxSeq struct {
		Text string
		A    mixedChild[string]
	}
	mxEmpty struct {
		Text string
		K    *string
	}
	mxOpt struct {
		Text string
		A    *mixedChild[string]
	}
	mxWild struct {
		Text string
		Any  anyChild
	}
	mxPair struct {
		Text string
		A    mixedChild[string]
		B    mixedChild[int32]
	}
	mxExtAttr struct {
		Text string
		A    mixedChild[string]
		K    *string
	}
	mxOpen struct {
		Text        string
		A           mixedChild[string]
		OpenContent []anyChild
	}
	mxPickI interface{ isMxPick() }
	mxPickA struct{ A mixedChild[string] }
	mxPickB struct{ B mixedChild[int32] }
	mxPick  struct {
		Text   string
		Choice mxPickI
	}
	mxFlatI interface{ isMxFlat() }
	mxFlatA struct{ A mixedChild[string] }
	mxFlatB struct{ B mixedChild[int32] }
	mxFlat  struct {
		Text     string
		Sequence []mxFlatI
	}
	mxAI        interface{ isMxA() }
	mxAX        struct{ X string }
	mxAY        struct{ Y int32 }
	mxSealedOpt struct {
		Text string
		A    *mixedChild[mxAI]
	}
	mxRepChoice struct {
		Text   string
		Choice []mxPickI
	}
	mxAny struct {
		Text string
		A    mixedChild[anyType]
	}
	mxOccItem struct {
		A mixedChild[string]
		B mixedChild[int32]
	}
	mxOcc struct {
		Text     string
		Sequence []mxOccItem
	}
	mxB struct {
		Text string
		B    *mixedChild[string]
	}
	mxNameAnyI    interface{ isMxNameAny() }
	mxNameAnyAny  struct{ Any anyChild }
	mxNameAnyAny2 struct{ Any mixedChild[string] }
	mxNameAny     struct {
		Text   string
		Choice mxNameAnyI
	}
	mxStrict        struct{ A string }
	elementOnlyPair struct {
		A string
		B int32
	}
)

func (mxPickA) isMxPick() {}
func (mxPickB) isMxPick() {}
func (mxFlatA) isMxFlat() {}
func (mxFlatB) isMxFlat() {}
func (mxAX) isMxA()       {}
func (mxAY) isMxA()       {}

func (mxNameAnyAny) isMxNameAny()  {}
func (mxNameAnyAny2) isMxNameAny() {}

// mxPickBranches supplies mxPickI's branches.
var mxPickBranches = WithBranches[mxPickI](mxPickA{}, mxPickB{})

// TestMixedTypesAreCodegens pins the types above to those codegen emits for
// each mixed-content schema, in shape (codegen/doc.go "# Complex types").
func TestMixedTypesAreCodegens(t *testing.T) {
	mixedChildDecl := "type MixedChild[T any] struct {\n\tElement T\n\tTail    string\n}\n"
	pair := "type R struct {\n\tText string\n\tA    MixedChild[string]\n\tB    MixedChild[int32]\n}\n"
	opt := "type R struct {\n\tText string\n\tA    *MixedChild[string]\n}\n"
	pickA, pickB := "type RChoiceA struct {\n\tA MixedChild[string]\n}\n", "type RChoiceB struct {\n\tB MixedChild[int32]\n}\n"
	rows := []struct {
		decls string
		want  []string
	}{
		{mxSeqSchema, []string{"type R struct {\n\tText string\n\tA    MixedChild[string]\n}\n", mixedChildDecl}},
		{mxEmptySchema, []string{"type R struct {\n\tText string\n\tK    *string\n}\n"}},
		{mxOptSchema, []string{opt}},
		{mxDefaultSchema, []string{opt}},
		{mxWildSchema, []string{"type R struct {\n\tText string\n\tAny  AnyChild\n}\n", "type AnyChild struct {\n\tElement AnyElement\n\tTail    string\n}\n"}},
		{mxExtSchema, []string{pair}},
		{mxPairSchema, []string{pair}},
		{mxExtAttrSchema, []string{"type R struct {\n\tText string\n\tA    MixedChild[string]\n\tK    *string\n}\n"}},
		{mxOpenSchema, []string{"type R struct {\n\tText        string\n\tA           MixedChild[string]\n\tOpenContent []AnyChild\n}\n"}},
		{mxPickSchema, []string{"type R struct {\n\tText   string\n\tChoice RChoice\n}\n", pickA, pickB}},
		{mxRepChoiceSchema, []string{"type R struct {\n\tText   string\n\tChoice []RChoice\n}\n", pickA, pickB}},
		{mxAnyTypeSchema, []string{"type R struct {\n\tText string\n\tA    MixedChild[AnyType]\n}\n"}},
		{mxFlatSchema, []string{"type R struct {\n\tText     string\n\tSequence []RSequence\n}\n", "type RSequence interface {\n\tisRSequence()\n}\n", "type RSequenceA struct {\n\tA MixedChild[string]\n}\n", "type RSequenceB struct {\n\tB MixedChild[int32]\n}\n"}},
		{mxSealedOptSchema, []string{"type R struct {\n\tText string\n\tA    *MixedChild[A]\n}\n", "type A interface {\n\tisA()\n}\n", "type AX struct {\n\tX string\n}\n"}},
		{mxOccSchema, []string{"type R struct {\n\tText     string\n\tSequence []RSequence\n}\n", "type RSequence struct {\n\tA MixedChild[string]\n\tB MixedChild[int32]\n}\n"}},
		{mxAnyRestrictSchema, []string{"type R struct {\n\tText string\n\tB    *MixedChild[string]\n}\n"}},
		{mxNameAnySchema, []string{"type R struct {\n\tText   string\n\tChoice RChoice\n}\n", "type RChoiceAny struct {\n\tAny AnyChild\n}\n", "type RChoiceAny2 struct {\n\tAny MixedChild[string]\n}\n"}},
		{mxStrictSchema, []string{"type R struct {\n\tA string\n}\n"}},
		{elementOnlyPairSchema, []string{"type R struct {\n\tA string\n\tB int32\n}\n"}},
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

// TestDecodeMixed pins the mixed-content lift (§3.4.1 {variety} mixed), the
// fields codegen/doc.go "# Complex types" gives filled as item 6 of Decode's
// doc states: sequence(a), each character run raw; an empty {particle}, Text
// and the attribute alone; a 0..1 particle, absent and present; a wildcard
// particle; an extension of a mixed base, adding a particle under
// mixed="true" and adding only an attribute (dcl.ctd.ctcc.common clause 4.2,
// cos-ct-extends clause 1.4.3.2.2.1); an {open content}, before and after
// the particle's child; white space between and around children kept
// verbatim, a run broken by a comment or CDATA concatenated; a choice
// occurring once, which stays a struct field; an element of type xs:anyType;
// a repeated choice and a repeated sequence, each held in an occurrence
// field, and a flattened content model; a 0..1 element of a bare sealed
// interface, *MixedChild[I], absent and present; a {value constraint}, which
// supplies no Text (§3.3.5.4 sic-eltType); a mixed restriction of
// xs:anyType, the document element's and a local element's; and a choice of
// a wildcard and an element any of xs:string, whose branch fields, AnyChild
// and MixedChild[string], share no Go type.
func TestDecodeMixed(t *testing.T) {
	str := func(s string) *string { return &s }
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"sequence(a), its characters around a", mxSeqSchema, "", "x<s:a>v</s:a>y", into[mxSeq](),
			mxSeq{Text: "x", A: mixedChild[string]{Element: "v", Tail: "y"}}},
		{"an empty {particle}", mxEmptySchema, ` k="kv"`, "hello", into[mxEmpty](),
			mxEmpty{Text: "hello", K: str("kv")}},
		{"a 0..1 particle, absent", mxOptSchema, "", "just text", into[mxOpt](),
			mxOpt{Text: "just text"}},
		{"a 0..1 particle, present", mxOptSchema, "", "x<s:a>v</s:a>y", into[mxOpt](),
			mxOpt{Text: "x", A: &mixedChild[string]{Element: "v", Tail: "y"}}},
		{"a wildcard particle", mxWildSchema, "", "t<s:a>x</s:a>u", into[mxWild](),
			mxWild{Text: "t", Any: anyChild{Element: anyElement{Space: "urn:s", Local: "a", Content: anyType{Text: "x"}}, Tail: "u"}}},
		{"an extension of a mixed base adding a particle", mxExtSchema, "", "x<s:a>v</s:a>y<s:b>1</s:b>z", into[mxPair](),
			mxPair{Text: "x", A: mixedChild[string]{Element: "v", Tail: "y"}, B: mixedChild[int32]{Element: 1, Tail: "z"}}},
		{"an extension of a mixed base adding an attribute", mxExtAttrSchema, ` k="kv"`, "x<s:a>v</s:a>y", into[mxExtAttr](),
			mxExtAttr{Text: "x", A: mixedChild[string]{Element: "v", Tail: "y"}, K: str("kv")}},
		{"an {open content}", mxOpenSchema, ` xmlns:o="urn:o"`, "<o:q>q</o:q>w<s:a>v</s:a>y<o:p>p</o:p>z", into[mxOpen](),
			mxOpen{Text: "", A: mixedChild[string]{Element: "v", Tail: "y"}, OpenContent: []anyChild{{Element: ocX("q", "q"), Tail: "w"}, {Element: ocX("p", "p"), Tail: "z"}}}},
		{"white space between and around children, verbatim", mxPairSchema, "", "\n  <s:a>1</s:a>\n  <s:b>2</s:b>\n", into[mxPair](),
			mxPair{Text: "\n  ", A: mixedChild[string]{Element: "1", Tail: "\n  "}, B: mixedChild[int32]{Element: 2, Tail: "\n"}}},
		{"runs broken by a comment, a processing instruction and CDATA", mxPairSchema, "", "x<!--c-->y<s:a>1</s:a>p<?pi d?>q<![CDATA[<r>]]><s:b>2</s:b>", into[mxPair](),
			mxPair{Text: "xy", A: mixedChild[string]{Element: "1", Tail: "pq<r>"}, B: mixedChild[int32]{Element: 2}}},
		{"a choice occurring once, a struct field", mxPickSchema, "", "x<s:b>2</s:b>y", into[mxPick](mxPickBranches),
			mxPick{Text: "x", Choice: mxPickB{B: mixedChild[int32]{Element: 2, Tail: "y"}}}},
		{"an element of type xs:anyType", mxAnyTypeSchema, "", `x<s:a k="v">in<s:c/>out</s:a>y`, into[mxAny](),
			mxAny{Text: "x", A: mixedChild[anyType]{Element: anyType{Attributes: []anyAttribute{{Local: "k", Value: "v"}}, Text: "in", Children: []anyChild{{Element: anyElement{Space: "urn:s", Local: "c"}, Tail: "out"}}}, Tail: "y"}}},
		{"a flattened content model", mxFlatSchema, "", "a<s:a>1</s:a>b<s:b>2</s:b>c<s:a>3</s:a>d", into[mxFlat](WithBranches[mxFlatI](mxFlatA{}, mxFlatB{})),
			mxFlat{Text: "a", Sequence: []mxFlatI{mxFlatA{A: mixedChild[string]{Element: "1", Tail: "b"}}, mxFlatB{B: mixedChild[int32]{Element: 2, Tail: "c"}}, mxFlatA{A: mixedChild[string]{Element: "3", Tail: "d"}}}}},
		{"a 0..1 element of a sealed interface, absent", mxSealedOptSchema, "", "t", into[mxSealedOpt](WithBranches[mxAI](mxAX{}, mxAY{})), mxSealedOpt{Text: "t"}},
		{"a 0..1 element of a sealed interface, present", mxSealedOptSchema, "", "t<s:a><s:y>2</s:y></s:a>u", into[mxSealedOpt](WithBranches[mxAI](mxAX{}, mxAY{})),
			mxSealedOpt{Text: "t", A: &mixedChild[mxAI]{Element: mxAY{Y: 2}, Tail: "u"}}},
		{"a repeated choice, held in an occurrence field", mxRepChoiceSchema, "", "a<s:a>1</s:a>b<s:b>2</s:b>c<s:a>3</s:a>d", into[mxRepChoice](mxPickBranches),
			mxRepChoice{Text: "a", Choice: []mxPickI{mxPickA{A: mixedChild[string]{Element: "1", Tail: "b"}}, mxPickB{B: mixedChild[int32]{Element: 2, Tail: "c"}}, mxPickA{A: mixedChild[string]{Element: "3", Tail: "d"}}}}},
		{"an occurrence content model", mxOccSchema, "", "w<s:a>1</s:a>x<s:b>2</s:b>y<s:a>3</s:a><s:b>4</s:b>z", into[mxOcc](),
			mxOcc{Text: "w", Sequence: []mxOccItem{{A: mixedChild[string]{Element: "1", Tail: "x"}, B: mixedChild[int32]{Element: 2, Tail: "y"}}, {A: mixedChild[string]{Element: "3"}, B: mixedChild[int32]{Element: 4, Tail: "z"}}}}},
		{"an empty element under a default, no Text supplied", mxDefaultSchema, "", "", into[mxOpt](), mxOpt{}},
		{"a local empty element under a default, no Text supplied", sequenceOf(`<xs:element name="a" default="d"><xs:complexType mixed="true"><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`),
			"", "<s:a/>", into[struct{ A mxB }](), struct{ A mxB }{}},
		{"a mixed restriction of xs:anyType", mxAnyRestrictSchema, "", "t", into[mxB](), mxB{Text: "t"}},
		{"a local element of a mixed restriction of xs:anyType", mxAnyRestrict + sequenceOf(`<xs:element name="a" type="s:dt"/>`), "", "<s:a>t<s:b>v</s:b></s:a>", into[struct{ A mxB }](),
			struct{ A mxB }{A: mxB{Text: "t", B: &mixedChild[string]{Element: "v"}}}},
		{"a choice of a wildcard and an element any of xs:string", mxNameAnySchema, ` xmlns:o="urn:o"`, "x<s:any>v</s:any>y", into[mxNameAny](WithBranches[mxNameAnyI](mxNameAnyAny{}, mxNameAnyAny2{})),
			mxNameAny{Text: "x", Choice: mxNameAnyAny2{Any: mixedChild[string]{Element: "v", Tail: "y"}}}},
		{"guard: an element-only restriction of a mixed base", mxStrictSchema, "", "\n<s:a>v</s:a>\n", into[mxStrict](), mxStrict{A: "v"}},
		{"guard: white space around element-only children", elementOnlyPairSchema, "", "\n  <s:a>1</s:a>\n  <s:b>2</s:b>\n", into[elementOnlyPair](), elementOnlyPair{A: "1", B: 2}},
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

// rootCharacters returns the character data of data's document element's
// own character children, every run concatenated, as the reader reports it.
func rootCharacters(t *testing.T, data []byte) string {
	t.Helper()
	r := xmltree.NewReader("", bytes.NewReader(data))
	var text strings.Builder
	depth := 0
	for {
		tok, err := r.Token()
		if err != nil {
			t.Fatal(err)
		}
		switch tok := tok.(type) {
		case *xmltree.StartElement:
			depth++
		case *xmltree.EndElement:
			depth--
			if depth == 0 {
				return text.String()
			}
		case *xmltree.CharData:
			if depth == 1 {
				text.WriteString(tok.Data())
			}
		}
	}
}

// TestDecodeMixedRoundTrip pins codegen's round trip for mixed content
// without an encoder: Text and then each child's Tail, in document order,
// concatenate to the document element's own character data (codegen/doc.go
// "# Complex types"), white space included.
func TestDecodeMixedRoundTrip(t *testing.T) {
	schema := schemaOf(t, mxRepChoiceSchema)
	data := instanceOf(` xmlns:o="urn:o"`, "\n <s:a>1</s:a> two \t<s:b>2</s:b><s:a>3</s:a>\n<!--c-->end\n")
	mustAccept(t, schema, data)
	got, err := Decode[mxRepChoice](schema, strict.New(), data, mxPickBranches)
	if err != nil {
		t.Fatal(err)
	}
	chars := got.Text
	for _, item := range got.Choice {
		// mxPickI's branches are a closed set, so the switch is STYLE T2's
		// sealed-sum exception.
		switch item := item.(type) {
		case mxPickA:
			chars += item.A.Tail
		case mxPickB:
			chars += item.B.Tail
		}
	}
	if want := rootCharacters(t, data); chars != want {
		t.Errorf("Text and the Tails = %q, want the document element's characters %q", chars, want)
	}
}

// The types codegen emits for codegen/testdata/mixed.xsd's para, which
// TestDecodeMixedPara pins against testdata/mixed.golden, with the types its
// fields reach: every field shape mixed content gives.
type (
	mxPara struct {
		Text     string
		Text2    mixedChild[int32]
		Em       *mixedChild[string]
		B        []mixedChild[string]
		Mark     *mixedChild[mxMark]
		Head     *mixedChild[mxHeadI]
		Link     mixedChild[mxLink]
		Any      []anyChild
		Choice   mxParaChoiceI
		Sequence []mxParaSeq
	}
	mxMark       string
	mxHead       string
	mxMember     string
	mxHeadI      interface{ isMxHead() }
	mxHeadHead   struct{ Head mxHead }
	mxHeadMember struct{ Member mxMember }
	mxLink       struct{ Sequence []mxLinkSeq }
	mxLinkSeq    struct{ Href string }

	mxParaChoiceI   interface{ isMxParaChoice() }
	mxParaChoiceX   struct{ X mixedChild[string] }
	mxParaChoiceSeq struct {
		Y mixedChild[string]
		Z []mixedChild[string]
	}
	mxParaSeq struct{ Href mixedChild[string] }
)

func (mxHeadHead) isMxHead()            {}
func (mxHeadMember) isMxHead()          {}
func (mxParaChoiceX) isMxParaChoice()   {}
func (mxParaChoiceSeq) isMxParaChoice() {}

// TestDecodeMixedPara decodes codegen/testdata/mixed.xsd's para, whose
// struct holds every shape a mixed-content field takes: a local V wrapped,
// an element named text taking Text2, *MixedChild[V], []MixedChild[V], a
// ref's named type wrapped, a substitution group's interface wrapped, a
// complex type's struct wrapped, its own element-only fields not, a
// wildcard's []AnyChild, a choice's interface unwrapped with its branch
// fields wrapped, and an occurrence field unwrapped with its item's fields
// wrapped (codegen/doc.go "# Complex types").
func TestDecodeMixedPara(t *testing.T) {
	golden, err := os.ReadFile(filepath.Join("..", "codegen", "testdata", "mixed.golden"))
	if err != nil {
		t.Fatal(err)
	}
	const paraDecl = "type Para struct {\n\tText     string\n\tText2    MixedChild[int32]\n\tEm       *MixedChild[string]\n\tB        []MixedChild[string]\n\tMark     *MixedChild[Mark]\n\tHead     *MixedChild[HeadSubstitutionGroup]\n\tLink     MixedChild[Link]\n\tAny      []AnyChild\n\tChoice   ParaChoice\n\tSequence []ParaSequence\n}\n"
	if !bytes.Contains(golden, []byte(paraDecl)) {
		t.Fatalf("testdata/mixed.golden does not declare\n%s", paraDecl)
	}
	doc, err := os.ReadFile(filepath.Join("..", "codegen", "testdata", "mixed.xsd"))
	if err != nil {
		t.Fatal(err)
	}
	schema := parseString(t, string(doc))
	data := []byte(`<para>t0<text>7</text>t1<em>e</em>t2<b>b1</b>t3<b>b2</b>t4<mark>m</mark>t5<member>s</member>t6<link><href>h</href></link>t7<w:p xmlns:w="urn:w">w</w:p>t8<y>y</y>t9<z>z1</z>t10<href>h2</href>t11</para>`)
	mustAccept(t, schema, data)
	opts := []Option{WithBranches[mxHeadI](mxHeadHead{}, mxHeadMember{}), WithBranches[mxParaChoiceI](mxParaChoiceX{}, mxParaChoiceSeq{})}
	got, err := Decode[mxPara](schema, strict.New(), data, opts...)
	if err != nil {
		t.Fatal(err)
	}
	var member mxHeadI = mxHeadMember{Member: "s"}
	want := mxPara{
		Text:     "t0",
		Text2:    mixedChild[int32]{Element: 7, Tail: "t1"},
		Em:       &mixedChild[string]{Element: "e", Tail: "t2"},
		B:        []mixedChild[string]{{Element: "b1", Tail: "t3"}, {Element: "b2", Tail: "t4"}},
		Mark:     &mixedChild[mxMark]{Element: "m", Tail: "t5"},
		Head:     &mixedChild[mxHeadI]{Element: member, Tail: "t6"},
		Link:     mixedChild[mxLink]{Element: mxLink{Sequence: []mxLinkSeq{{Href: "h"}}}, Tail: "t7"},
		Any:      []anyChild{{Element: anyElement{Space: "urn:w", Local: "p", Content: anyType{Text: "w"}}, Tail: "t8"}},
		Choice:   mxParaChoiceSeq{Y: mixedChild[string]{Element: "y", Tail: "t9"}, Z: []mixedChild[string]{{Element: "z1", Tail: "t10"}}},
		Sequence: []mxParaSeq{{Href: mixedChild[string]{Element: "h2", Tail: "t11"}}},
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("Decode = %#v, want %#v", *got, want)
	}
	if err := CheckDecode[mxPara](schema, xsd.QName{Local: "para"}, opts...); err != nil {
		t.Errorf("CheckDecode = %v, want nil", err)
	}
}

// otherChild is of AnyChild's shape but is not the struct anyElement's
// Content's Children hold, so it is no wildcard's field in mixed content.
type otherChild struct {
	Element anyElement
	Tail    string
}

// TestDecodeMixedTypeMismatch pins item 6 of Decode's doc for mixed content:
// a T without Text, with Text misnamed or ahead of nothing, a child's field
// holding V unwrapped or a struct of Tail before Element, or a wildcard's
// field holding AnyElement or a struct of AnyChild's shape other than
// AnyElement's own, is ErrTypeMismatch, no decline; and an element-only
// type's T of the mixed shape is too.
func TestDecodeMixedTypeMismatch(t *testing.T) {
	type noText struct{ A mixedChild[string] }
	type unwrapped struct {
		Text string
		A    string
	}
	type swapped struct {
		Text string
		A    struct {
			Tail    string
			Element string
		}
	}
	type tailInt struct {
		Text string
		A    struct {
			Element string
			Tail    int
		}
	}
	type wildElement struct {
		Text string
		Any  anyElement
	}
	type wildOther struct {
		Text string
		Any  otherChild
	}
	rows := []struct {
		name  string
		decls string
		check func(*xsd.Schema) error
	}{
		{"no Text", mxSeqSchema, func(s *xsd.Schema) error { return CheckDecode[noText](s, rootName) }},
		{"a child's V unwrapped", mxSeqSchema, func(s *xsd.Schema) error { return CheckDecode[unwrapped](s, rootName) }},
		{"Tail before Element", mxSeqSchema, func(s *xsd.Schema) error { return CheckDecode[swapped](s, rootName) }},
		{"a Tail of another type", mxSeqSchema, func(s *xsd.Schema) error { return CheckDecode[tailInt](s, rootName) }},
		{"a wildcard's AnyElement", mxWildSchema, func(s *xsd.Schema) error { return CheckDecode[wildElement](s, rootName) }},
		{"a wildcard's struct of AnyChild's shape, not AnyElement's own", mxWildSchema, func(s *xsd.Schema) error { return CheckDecode[wildOther](s, rootName) }},
		{"guard: an element-only restriction of a mixed base, of the mixed shape", mxStrictSchema, func(s *xsd.Schema) error { return CheckDecode[mxSeq](s, rootName) }},
		{"guard: element-only content, of the mixed shape", elementOnlyPairSchema, func(s *xsd.Schema) error { return CheckDecode[mxPair](s, rootName) }},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			err := row.check(schemaOf(t, row.decls))
			if !errors.Is(err, ErrTypeMismatch) {
				t.Errorf("CheckDecode = %v, want ErrTypeMismatch", err)
			}
			var de *DeclineError
			if errors.As(err, &de) {
				t.Errorf("CheckDecode = %v, want no decline", err)
			}
		})
	}
	type txt struct {
		Txt string
		A   mixedChild[string]
	}
	const want = "codec: T is not the type codegen emits for the document element: field 0 of codec.txt is Txt string where codegen emits Text string, the characters of mixed content before its first child"
	if err := CheckDecode[txt](schemaOf(t, mxSeqSchema), rootName); err == nil || err.Error() != want {
		t.Errorf("CheckDecode = %v, want %q", err, want)
	}
	const wantWrap = "codec: T is not the type codegen emits for the document element: field 1 of codec.unwrapped is A string where codegen emits A M for a struct M of codegen's MixedChild shape whose Element is string"
	if err := CheckDecode[unwrapped](schemaOf(t, mxSeqSchema), rootName); err == nil || err.Error() != wantWrap {
		t.Errorf("CheckDecode = %v, want %q", err, wantWrap)
	}
	// A branch of an unwrapped field is held to no alternative, as a scalar
	// one-field branch matching none is, rather than to the alternative it
	// names (field.namesBranch reads the wrapped arm's Element).
	const wantBranch = "codec: T is not the type codegen emits for the document element: branch codec.mxBareA of codec.mxPickI has the shape of no branch struct codegen emits that an earlier branch did not match"
	err := CheckDecode[mxPick](schemaOf(t, mxPickSchema), rootName, WithBranches[mxPickI](mxBareA{}, mxBareB{}))
	if err == nil || err.Error() != wantBranch {
		t.Errorf("CheckDecode = %v, want %q", err, wantBranch)
	}
}

// mxBareA and mxBareB are mxPickI's branches with their fields unwrapped,
// which mixed content's choice does not emit.
type (
	mxBareA struct{ A string }
	mxBareB struct{ B int32 }
)

func (mxBareA) isMxPick() {}
func (mxBareB) isMxPick() {}

// TestDecodeElementOnlyCharacters pins that a non-white-space character
// child of element-only content stays validate's violation (§3.4.4.2
// cvc-complex-type clause 1.3): Decode returns the charged error, no value.
func TestDecodeElementOnlyCharacters(t *testing.T) {
	schema := schemaOf(t, elementOnlyPairSchema)
	got, err := Decode[elementOnlyPair](schema, strict.New(), instanceOf("", "x<s:a>1</s:a><s:b>2</s:b>"))
	if got != nil || len(ViolationsOf(err)) == 0 {
		t.Errorf("Decode = %v, %v, want no value and validate's violation", got, err)
	}
}

// The mixed {type table} schemas, each declaring s:r. mxTableSchema's r, of
// base, a mixed sequence(a?) carrying an optional kind, selects num, a mixed
// extension adding n, an xs:int, where kind is "num", and plain, an
// element-only restriction of base, where it is "p": each governing type is
// mixed by its own {content type} (dcl.ctd.ctcc.common clause 4.2).
// mxLocalTableSchema's r, mixed, holds e, a repeated element of ttTypes'
// element-only base selecting num where kind is "num", and v, an optional
// xs:long selecting xs:int where it carries xsi:type.
const (
	mxTableSchema = `<xs:complexType name="base" mixed="true"><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence><xs:attribute name="kind" type="xs:string"/></xs:complexType>
<xs:complexType name="num" mixed="true"><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="n" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
<xs:complexType name="plain"><xs:complexContent><xs:restriction base="s:base"><xs:sequence><xs:element name="a" type="xs:string" minOccurs="0"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>
<xs:element name="r" type="s:base"><xs:alternative test="@kind = 'num'" type="s:num"/><xs:alternative test="@kind = 'p'" type="s:plain"/></xs:element>`
	mxLocalTableSchema = ttTypes + `<xs:element name="r"><xs:complexType mixed="true"><xs:sequence><xs:element name="e" type="s:base" maxOccurs="unbounded"><xs:alternative test="@kind = 'num'" type="s:num"/></xs:element>
<xs:element name="v" type="xs:long" minOccurs="0"><xs:alternative test="@xsi:type" type="xs:int"/></xs:element></xs:sequence></xs:complexType></xs:element>`
)

// The types codegen emits for those schemas, which TestDecodeMixedTypeTable
// pins: mxTableSchema's R, RNum, Num, RPlain, Plain, RBase and Base are mxtR
// to mxtBase; mxLocalTableSchema's R, E, ENum, EBase, V, VInt and VLong are
// mxtLocal to mxtVLong, its Num and Base ttTypes' ttNum and ttBase.
type (
	mxtR    interface{ isMxtR() }
	mxtRNum struct{ Num mxtNum }
	mxtNum  struct {
		Text string
		A    *mixedChild[string]
		N    mixedChild[int32]
		Kind *string
	}
	mxtRPlain struct{ Plain mxtPlain }
	mxtPlain  struct {
		A    *string
		Kind *string
	}
	mxtRBase struct{ Base mxtBase }
	mxtBase  struct {
		Text string
		A    *mixedChild[string]
		Kind *string
	}

	mxtLocal struct {
		Text string
		E    []mixedChild[mxtE]
		V    *mixedChild[mxtV]
	}
	mxtE     interface{ isMxtE() }
	mxtENum  struct{ Num ttNum }
	mxtEBase struct{ Base ttBase }
	mxtV     interface{ isMxtV() }
	mxtVInt  struct{ Int int32 }
	mxtVLong struct{ Long int64 }
)

func (mxtRNum) isMxtR()   {}
func (mxtRPlain) isMxtR() {}
func (mxtRBase) isMxtR()  {}
func (mxtENum) isMxtE()   {}
func (mxtEBase) isMxtE()  {}
func (mxtVInt) isMxtV()   {}
func (mxtVLong) isMxtV()  {}

// TestDecodeMixedTypeTable pins mixed content under a {type table} (§3.3.4.1
// key-cta-select), each row decoding into the types codegen emits, which the
// test first checks against its output: at the document element, the
// selected governing type's branch holds that type's own struct, with Text
// and MixedChild where its own {content type} is mixed and neither where it
// is element-only, the branch's one field never wrapped; and in mixed
// content, a particle whose declaration has two governing types holds
// MixedChild over their interface, its Tail the characters after it.
func TestDecodeMixedTypeTable(t *testing.T) {
	str := func(s string) *string { return &s }
	gens := []struct {
		decls string
		want  []string
	}{
		{mxTableSchema, []string{"type RNum struct {\n\tNum Num\n}\n",
			"type Num struct {\n\tText string\n\tA    *MixedChild[string]\n\tN    MixedChild[int32]\n\tKind *string\n}\n",
			"type RPlain struct {\n\tPlain Plain\n}\n", "type Plain struct {\n\tA    *string\n\tKind *string\n}\n",
			"type RBase struct {\n\tBase Base\n}\n", "type Base struct {\n\tText string\n\tA    *MixedChild[string]\n\tKind *string\n}\n"}},
		{mxLocalTableSchema, []string{"type R struct {\n\tText string\n\tE    []MixedChild[E]\n\tV    *MixedChild[V]\n}\n",
			"type ENum struct {\n\tNum Num\n}\n", "type EBase struct {\n\tBase Base\n}\n",
			"type VInt struct {\n\tInt int32\n}\n", "type VLong struct {\n\tLong int64\n}\n"}},
	}
	for _, g := range gens {
		gen := generated(t, schemaOf(t, g.decls))
		for _, decl := range g.want {
			if !bytes.Contains(gen, []byte(decl)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
			}
		}
	}
	rootInto := into[mxtR](WithBranches[mxtR](mxtRNum{}, mxtRPlain{}, mxtRBase{}))
	localInto := into[mxtLocal](WithBranches[mxtE](mxtENum{}, mxtEBase{}), WithBranches[mxtV](mxtVInt{}, mxtVLong{}))
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"a mixed extension selected", mxTableSchema, ` kind="num"`, "x<s:a>v</s:a>y<s:n>1</s:n>z", rootInto,
			mxtR(mxtRNum{Num: mxtNum{Text: "x", A: &mixedChild[string]{Element: "v", Tail: "y"}, N: mixedChild[int32]{Element: 1, Tail: "z"}, Kind: str("num")}})},
		{"the mixed declared type by default", mxTableSchema, "", "x<s:a>v</s:a>y", rootInto,
			mxtR(mxtRBase{Base: mxtBase{Text: "x", A: &mixedChild[string]{Element: "v", Tail: "y"}}})},
		{"guard: an element-only restriction selected", mxTableSchema, ` kind="p"`, "\n<s:a>v</s:a>\n", rootInto,
			mxtR(mxtRPlain{Plain: mxtPlain{A: str("v"), Kind: str("p")}})},
		{"a particle of two governing types in mixed content", mxLocalTableSchema, "", `t<s:e kind="num"><s:a>1</s:a><s:n>2</s:n></s:e>u<s:e><s:a>3</s:a></s:e>w<s:v>4</s:v>z`, localInto,
			mxtLocal{Text: "t", E: []mixedChild[mxtE]{{Element: mxtENum{Num: ttNum{A: "1", N: 2, Kind: str("num")}}, Tail: "u"}, {Element: mxtEBase{Base: ttBase{A: "3"}}, Tail: "w"}},
				V: &mixedChild[mxtV]{Element: mxtVLong{Long: 4}, Tail: "z"}}},
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

// TestMixedCharAfterChildIsDefect pins fillState.char: a content loop that
// reads character data after a child no tailReader read returns, for mixed
// content, the plain defect error at that data's Loc, and drops it for
// element-only content. The child is read to its end tag alone, as an
// unwrapped field's reader reads it; for mixed content the Text phase
// boundBinding.fill runs (charRun) comes first.
func TestMixedCharAfterChildIsDefect(t *testing.T) {
	const doc = "<r>\n<a>v</a>y</r>"
	x := childIndex{children: map[xsd.QName]taker{{Local: "a"}: {}}}
	for _, mixed := range []bool{false, true} {
		r := xmltree.NewReader("", bytes.NewReader([]byte(doc)))
		if _, err := documentElement(r); err != nil {
			t.Fatal(err)
		}
		st := &fillState{mixed: mixed}
		if mixed {
			_, next, err := charRun(r)
			if err != nil {
				t.Fatal(err)
			}
			st.pending = next
		}
		err := eachChild(r, nil, x, st, func(*xmltree.StartElement, int) error {
			return skipToEnd(r)
		})
		if !mixed {
			if err != nil {
				t.Errorf("element-only: eachChild = %v, want nil", err)
			}
			continue
		}
		const want = "codec: ?:2:9: character data after a child of mixed content no tailReader read"
		if err == nil || err.Error() != want {
			t.Errorf("mixed: eachChild = %v, want %q", err, want)
		}
	}
}

// skipToEnd reads r up to the end tag of an element whose start tag r has
// just returned and whose content holds no element.
func skipToEnd(r *xmltree.Reader) error {
	for {
		tok, err := r.Token()
		if err != nil {
			return err
		}
		if _, end := tok.(*xmltree.EndElement); end {
			return nil
		}
	}
}

// TestDecodeMixedUnwrapped pins that Decode answers a T whose child field
// holds V unwrapped in mixed content ErrTypeMismatch, as CheckDecode does
// (TestDecodeMixedTypeMismatch), and decodes no value into it.
func TestDecodeMixedUnwrapped(t *testing.T) {
	type unwrapped struct {
		Text string
		A    string
	}
	schema := schemaOf(t, mxSeqSchema)
	got, err := Decode[unwrapped](schema, strict.New(), instanceOf("", "x<s:a>v</s:a>y"))
	if !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Decode = %#v, %v; want ErrTypeMismatch", got, err)
	}
}
