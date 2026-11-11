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

// ocAB is a sequence of the xs:string elements a and b.
const ocAB = `<xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence>`

// The {open content} schemas, each declaring s:r: ocInterleaveSchema's type is
// sequence(a, b) under an interleave ##other {open content},
// ocSuffixSchema's the same under suffix, and ocAnySchema's under an
// interleave ##any one; ocSuffixAnySchema's is sequence(a, b?) under a suffix
// ##any one; ocDefaultDoc is a whole schema document whose
// <defaultOpenContent> reaches r's type, which has no <openContent> of its
// own (dcl.ctd.ctcc.common clause 5.2.1); ocExtSchema's r is of ext, an
// extension of base, sequence(a) under {open content}, adding sequence(b) with
// neither element of its own, so it inherits base's (clause 4.2.2 and the
// {open content} row of the clause 4.2.3 tableau); ocNoneSchema's ext2 adds
// <openContent mode="none"> to base2, sequence(a) with none, so it has none;
// ocEmptySchema's type has empty explicit content, which clause 6.2 makes
// element-only; ocNestedSchema's r, of sequence(c*) under {open content},
// declares c of an anonymous type of sequence(a) under one of its own;
// ocChoiceSchema's type is choice(a, b) under {open content}, which codegen
// gives a struct rather than the choice's interface; ocWildSchema's is
// sequence(a, any?), the wildcard particle over urn:w, beside it;
// ocNameSchema's holds an element and an attribute named openContent, so the
// field is OpenContent2; ocFlatSchema's is flatSeqSchema's repeated
// sequence(a?, b?), which codegen flattens, and ocOccSchema's occSeqSchema's
// repeated sequence(a, b), held in an occurrence field, each under {open
// content}.
const (
	ocInterleaveSchema = `<xs:element name="r"><xs:complexType>` + `<xs:openContent mode="interleave"><xs:any namespace="##other" processContents="lax"/></xs:openContent>` + ocAB + `</xs:complexType></xs:element>`
	ocSuffixSchema     = `<xs:element name="r"><xs:complexType>` + `<xs:openContent mode="suffix"><xs:any namespace="##other" processContents="lax"/></xs:openContent>` + ocAB + `</xs:complexType></xs:element>`
	ocSuffixAnySchema  = `<xs:element name="r"><xs:complexType><xs:openContent mode="suffix"><xs:any processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	ocAnySchema        = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any processContents="lax"/></xs:openContent>` + ocAB + `</xs:complexType></xs:element>`
	ocDefaultDoc       = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="urn:s" targetNamespace="urn:s" elementFormDefault="qualified">
		<xs:defaultOpenContent><xs:any namespace="##other" processContents="lax"/></xs:defaultOpenContent>
		<xs:element name="r"><xs:complexType>` + ocAB + `</xs:complexType></xs:element></xs:schema>`
	ocExtSchema = `<xs:complexType name="base"><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:complexType name="ext"><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
		<xs:element name="r" type="s:ext"/>`
	ocNoneSchema = `<xs:complexType name="base2"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:complexType name="ext2"><xs:complexContent><xs:extension base="s:base2"><xs:openContent mode="none"/><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
		<xs:element name="r" type="s:ext2"/>`
	ocEmptySchema  = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent></xs:complexType></xs:element>`
	ocNestedSchema = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence>
		<xs:element name="c" maxOccurs="unbounded"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
		</xs:sequence></xs:complexType></xs:element>`
	ocChoiceSchema = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:choice></xs:complexType></xs:element>`
	ocWildSchema   = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="a" type="xs:string"/><xs:any namespace="urn:w" processContents="lax" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	ocNameSchema   = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence><xs:element name="openContent" type="xs:string"/></xs:sequence><xs:attribute name="k" type="xs:string"/></xs:complexType></xs:element>`
	ocFlatSchema   = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent><xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string" minOccurs="0"/><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
	ocOccSchema    = `<xs:element name="r"><xs:complexType><xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent>` +
		`<xs:sequence maxOccurs="unbounded"><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`
)

// The types codegen emits for those schemas' s:r, which
// TestOpenContentTypesAreCodegens pins, each open-content field a slice of
// anyElement, of AnyElement's shape: ocABType for ocInterleaveSchema's,
// ocSuffixSchema's, ocAnySchema's, ocDefaultDoc's and ocExtSchema's R;
// ocSuffixAny ocSuffixAnySchema's; ocNone ocNoneSchema's, with no such
// field; ocEmpty ocEmptySchema's, the field alone; ocNested and its
// occurrence's ocNestedC ocNestedSchema's; ocChoice and its branches
// ocChoiceSchema's; ocWild ocWildSchema's; ocName ocNameSchema's; ocFlat
// ocFlatSchema's, over flatSeqSchema's interface; and ocOcc ocOccSchema's,
// over occSeqSchema's item.
type (
	ocABType struct {
		A, B        string
		OpenContent []anyElement
	}
	ocSuffixAny struct {
		A           string
		B           *string
		OpenContent []anyElement
	}
	ocNone   struct{ A, B string }
	ocEmpty  struct{ OpenContent []anyElement }
	ocNested struct {
		C           []ocNestedC
		OpenContent []anyElement
	}
	ocNestedC struct {
		A           string
		OpenContent []anyElement
	}
	ocChoice struct {
		Choice      awChoiceI
		OpenContent []anyElement
	}
	ocWild struct {
		A           string
		Any         *anyElement
		OpenContent []anyElement
	}
	ocName struct {
		OpenContent  string
		OpenContent2 []anyElement
		K            *string
	}
	ocFlat struct {
		Sequence    []flatSeqItem
		OpenContent []anyElement
	}
	ocOcc struct {
		Sequence    []occSeqItem
		OpenContent []anyElement
	}
)

// ocSchema returns the schema of decls, a whole schema document where it
// opens with <xs:schema, as ocDefaultDoc does, and otherwise schemaOf's.
func ocSchema(t *testing.T, decls string) *xsd.Schema {
	t.Helper()
	if strings.HasPrefix(decls, "<xs:schema") {
		return parseString(t, decls)
	}
	return schemaOf(t, decls)
}

// TestOpenContentTypesAreCodegens pins the types above to those codegen emits
// for each {open content} schema, in shape (codegen/doc.go "# Open content").
func TestOpenContentTypesAreCodegens(t *testing.T) {
	ab := "type R struct {\n\tA           string\n\tB           string\n\tOpenContent []AnyElement\n}\n"
	rows := []struct {
		decls string
		want  []string
	}{
		{ocInterleaveSchema, []string{ab}},
		{ocSuffixSchema, []string{ab}},
		{ocAnySchema, []string{ab}},
		{ocDefaultDoc, []string{ab}},
		{ocExtSchema, []string{ab}},
		{ocSuffixAnySchema, []string{"type R struct {\n\tA           string\n\tB           *string\n\tOpenContent []AnyElement\n}\n"}},
		{ocNoneSchema, []string{"type R struct {\n\tA string\n\tB string\n}\n"}},
		{ocEmptySchema, []string{"type R struct {\n\tOpenContent []AnyElement\n}\n"}},
		{ocNestedSchema, []string{"type R struct {\n\tC           []C\n\tOpenContent []AnyElement\n}\n", "type C struct {\n\tA           string\n\tOpenContent []AnyElement\n}\n"}},
		{ocChoiceSchema, []string{
			"type R struct {\n\tChoice      RChoice\n\tOpenContent []AnyElement\n}\n",
			"type RChoiceA struct {\n\tA string\n}\n",
			"type RChoiceB struct {\n\tB int32\n}\n",
		}},
		{ocWildSchema, []string{"type R struct {\n\tA           string\n\tAny         *AnyElement\n\tOpenContent []AnyElement\n}\n"}},
		{ocNameSchema, []string{"type R struct {\n\tOpenContent  string\n\tOpenContent2 []AnyElement\n\tK            *string\n}\n"}},
		{ocFlatSchema, []string{"type R struct {\n\tSequence    []RSequence\n\tOpenContent []AnyElement\n}\n", "type RSequenceA struct {\n\tA string\n}\n", "type RSequenceB struct {\n\tB string\n}\n"}},
		{ocOccSchema, []string{"type R struct {\n\tSequence    []RSequence\n\tOpenContent []AnyElement\n}\n", "type RSequence struct {\n\tA string\n\tB string\n}\n"}},
	}
	for _, row := range rows {
		gen := generated(t, ocSchema(t, row.decls))
		for _, decl := range row.want {
			if !bytes.Contains(gen, []byte(decl)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
			}
		}
	}
}

// ocX is the raw element o:<local> in urn:o holding text.
func ocX(local, text string) anyElement {
	return anyElement{Space: "urn:o", Local: local, Content: anyType{Text: text}}
}

// TestDecodeOpenContent pins the {open content} lift: each child validation
// attributes to the {open content} (§3.4.4.4), one the {particle} could not
// take at that point (§3.4.4.3 cvc-complex-content clauses 2.3 and 3.3), is
// appended raw, in document order, to the open-content field codegen emits
// after the particle fields, and every other child fills its particle's
// field as without it: under {mode} interleave and suffix, where every child
// after the first it takes is its own (clause 2.1); an {open content}
// a <defaultOpenContent> brings, or an extension inherits; empty explicit
// content clause 6.2 makes element-only, the field alone; a nested type's own,
// in its occurrence's struct and never the parent's; a child bearing a
// particle's name the {particle} cannot take, under ##any; beside a choice,
// a wildcard particle, an element named openContent and an attribute, a
// flattened content model and an occurrence field; and a child read raw, its
// xsi:type among its attributes, no decline. An <openContent mode="none"> on
// an extension of a base with none gives no field.
func TestDecodeOpenContent(t *testing.T) {
	str := func(s string) *string { return &s }
	o := ` xmlns:o="urn:o"`
	a, b := "<s:a>1</s:a>", "<s:b>2</s:b>"
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"interleave", ocInterleaveSchema, o, a + `<o:x>x</o:x>` + b + `<o:y>y</o:y>`, into[ocABType](),
			ocABType{A: "1", B: "2", OpenContent: []anyElement{ocX("x", "x"), ocX("y", "y")}}},
		{"suffix", ocSuffixSchema, o, a + b + `<o:x>x</o:x>`, into[ocABType](),
			ocABType{A: "1", B: "2", OpenContent: []anyElement{ocX("x", "x")}}},
		{"suffix, a child after the first it takes, though the particle could take it", ocSuffixAnySchema, o, a + `<o:x>x</o:x>` + b, into[ocSuffixAny](),
			ocSuffixAny{A: "1", OpenContent: []anyElement{ocX("x", "x"), {Space: "urn:s", Local: "b", Content: anyType{Text: "2"}}}}},
		{"suffix, a child the particle takes before it", ocSuffixAnySchema, o, a + b + `<o:x>x</o:x>`, into[ocSuffixAny](),
			ocSuffixAny{A: "1", B: str("2"), OpenContent: []anyElement{ocX("x", "x")}}},
		{"no child the open content takes", ocInterleaveSchema, "", a + b, into[ocABType](), ocABType{A: "1", B: "2"}},
		{"a <defaultOpenContent>'s", ocDefaultDoc, o, `<o:x>x</o:x>` + a + b, into[ocABType](),
			ocABType{A: "1", B: "2", OpenContent: []anyElement{ocX("x", "x")}}},
		{"an extension inheriting its base's", ocExtSchema, o, a + `<o:x>x</o:x>` + b, into[ocABType](),
			ocABType{A: "1", B: "2", OpenContent: []anyElement{ocX("x", "x")}}},
		{"empty explicit content made element-only", ocEmptySchema, o, `<o:x>x</o:x><o:y>y</o:y>`, into[ocEmpty](),
			ocEmpty{OpenContent: []anyElement{ocX("x", "x"), ocX("y", "y")}}},
		{"a nested type's own, in its occurrence's struct", ocNestedSchema, o, `<s:c><s:a>1</s:a><o:x>x</o:x></s:c><o:z>z</o:z><s:c><o:y>y</o:y><s:a>2</s:a></s:c>`, into[ocNested](),
			ocNested{C: []ocNestedC{{A: "1", OpenContent: []anyElement{ocX("x", "x")}}, {A: "2", OpenContent: []anyElement{ocX("y", "y")}}}, OpenContent: []anyElement{ocX("z", "z")}}},
		{"under ##any, children of a particle's name the particle cannot take", ocAnySchema, "", a + `<s:a>3</s:a>` + b + `<s:b>4</s:b>`, into[ocABType](),
			ocABType{A: "1", B: "2", OpenContent: []anyElement{{Space: "urn:s", Local: "a", Content: anyType{Text: "3"}}, {Space: "urn:s", Local: "b", Content: anyType{Text: "4"}}}}},
		{"under ##any, children the particle can take", ocAnySchema, "", a + b, into[ocABType](), ocABType{A: "1", B: "2"}},
		{"beside a choice", ocChoiceSchema, o, `<o:x>x</o:x><s:b>2</s:b>`, into[ocChoice](WithBranches[awChoiceI](awChoiceA{}, awChoiceB{})),
			ocChoice{Choice: awChoiceB{B: 2}, OpenContent: []anyElement{ocX("x", "x")}}},
		{"beside a wildcard particle, which takes its child first", ocWildSchema, o + ` xmlns:w="urn:w"`, a + `<w:p>p</w:p><o:x>x</o:x>`, into[ocWild](),
			ocWild{A: "1", Any: &anyElement{Space: "urn:w", Local: "p", Content: anyType{Text: "p"}}, OpenContent: []anyElement{ocX("x", "x")}}},
		{"beside an element and an attribute named openContent", ocNameSchema, o + ` k="v"`, `<o:x>x</o:x><s:openContent>e</s:openContent>`, into[ocName](),
			ocName{OpenContent: "e", OpenContent2: []anyElement{ocX("x", "x")}, K: str("v")}},
		{"beside a flattened content model", ocFlatSchema, o, a + `<o:x>x</o:x>` + b + a, into[ocFlat](flatSeqBranches),
			ocFlat{Sequence: []flatSeqItem{flatSeqA{A: "1"}, flatSeqB{B: "2"}, flatSeqA{A: "1"}}, OpenContent: []anyElement{ocX("x", "x")}}},
		{"beside an occurrence field", ocOccSchema, o, a + `<o:x>x</o:x>` + b + a + `<o:y>y</o:y>` + b, into[ocOcc](),
			ocOcc{Sequence: []occSeqItem{{A: "1", B: "2"}, {A: "1", B: "2"}}, OpenContent: []anyElement{ocX("x", "x"), ocX("y", "y")}}},
		{"a child read raw, its xsi:type among its attributes", ocInterleaveSchema, o, a + `<o:x xsi:type="xs:string">x</o:x>` + b, into[ocABType](),
			ocABType{A: "1", B: "2", OpenContent: []anyElement{{Space: "urn:o", Local: "x", Content: anyType{Attributes: []anyAttribute{{Space: "http://www.w3.org/2001/XMLSchema-instance", Local: "type", Value: "xs:string"}}, Text: "x"}}}}},
		{"<openContent mode=\"none\"> on an extension of a base with none", ocNoneSchema, "", a + b, into[ocNone](), ocNone{A: "1", B: "2"}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := ocSchema(t, row.decls)
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

// TestDecodeOpenContentTypeMismatch pins item 6 of Decode's doc for the
// open-content field: a T without it, or with it misnamed, of another Go
// type or after the attribute fields, is ErrTypeMismatch, no decline.
func TestDecodeOpenContentTypeMismatch(t *testing.T) {
	type missing struct{ A, B string }
	type misnamed struct {
		A, B string
		Open []anyElement
	}
	type single struct {
		A, B        string
		OpenContent anyElement
	}
	type late struct {
		OpenContent  string
		K            *string
		OpenContent2 []anyElement
	}
	rows := []struct {
		name  string
		decls string
		check func(*xsd.Schema) error
	}{
		{"missing", ocInterleaveSchema, func(s *xsd.Schema) error { return CheckDecode[missing](s, xsd.QName{Space: "urn:s", Local: "r"}) }},
		{"misnamed", ocInterleaveSchema, func(s *xsd.Schema) error { return CheckDecode[misnamed](s, xsd.QName{Space: "urn:s", Local: "r"}) }},
		{"not a slice", ocInterleaveSchema, func(s *xsd.Schema) error { return CheckDecode[single](s, xsd.QName{Space: "urn:s", Local: "r"}) }},
		{"after the attribute fields", ocNameSchema, func(s *xsd.Schema) error { return CheckDecode[late](s, xsd.QName{Space: "urn:s", Local: "r"}) }},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			err := row.check(ocSchema(t, row.decls))
			if !errors.Is(err, ErrTypeMismatch) {
				t.Errorf("CheckDecode = %v, want ErrTypeMismatch", err)
			}
			var de *DeclineError
			if errors.As(err, &de) {
				t.Errorf("CheckDecode = %v, want no decline", err)
			}
		})
	}
	schema := ocSchema(t, ocInterleaveSchema)
	got, err := Decode[missing](schema, strict.New(), instanceOf("", "<s:a>1</s:a><s:b>2</s:b>"))
	if got != nil || !errors.Is(err, ErrTypeMismatch) {
		t.Errorf("Decode = %v, %v, want no value and ErrTypeMismatch", got, err)
	}
}
