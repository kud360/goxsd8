package codec

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// The {attribute wildcard} schemas, each declaring s:r. awUseSchema's type
// has the use a and a lax wildcard; awExtSchema's r is of ext, an extension
// adding the use b to base, whose wildcard ext inherits (dcl.ctd.anyatt
// 2.2.2.2); awUnionSchema's ext2 adds its own wildcard over urn:x to base2's
// over urn:b, the union of both (2.2.2.3, cos-aw-union); awRestrictSchema's
// r is of res, a restriction of base, which inherits no wildcard (2.1);
// awGroupSchema's wildcard is brought by <xs:attributeGroup ref>
// (declare-attributeGroup-wildcard); awChoiceSchema's type is a choice with
// no attribute use; awNameSchema's has an attribute named anyAttribute and
// awElemNameSchema's an element so named, so codegen names the wildcard's
// field AnyAttribute2; awParticleSchema's sequence holds a wildcard particle;
// awSimpleSchema's a is of simple content with a wildcard, and
// awNestedSchema's a of a nested anonymous type with one; and
// awStrictSchema's wildcard is strict over the target namespace, whose
// top-level attribute g it declares; awEmptySchema's type has empty content
// and a wildcard alone. awDefaultDoc is a whole schema document
// whose defaultAttributes names an attribute group holding the wildcard.
const (
	awUseSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence>
		<xs:attribute name="a" type="xs:int"/><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`
	awBase      = `<xs:complexType name="base"><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:attribute name="a" type="xs:int"/><xs:anyAttribute processContents="lax"/></xs:complexType>`
	awExtSchema = awBase + `<xs:complexType name="ext"><xs:complexContent><xs:extension base="s:base"><xs:attribute name="b" type="xs:string"/></xs:extension></xs:complexContent></xs:complexType>
		<xs:element name="r" type="s:ext"/>`
	awUnionSchema = `<xs:complexType name="base2"><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:anyAttribute namespace="urn:b" processContents="skip"/></xs:complexType>
		<xs:complexType name="ext2"><xs:complexContent><xs:extension base="s:base2"><xs:anyAttribute namespace="urn:x" processContents="skip"/></xs:extension></xs:complexContent></xs:complexType>
		<xs:element name="r" type="s:ext2"/>`
	awRestrictSchema = awBase + `<xs:complexType name="res"><xs:complexContent><xs:restriction base="s:base"><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:attribute name="a" type="xs:int"/></xs:restriction></xs:complexContent></xs:complexType>
		<xs:element name="r" type="s:res"/>`
	awGroupSchema = `<xs:attributeGroup name="g"><xs:attribute name="a" type="xs:int"/><xs:anyAttribute processContents="lax"/></xs:attributeGroup>
		<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:attributeGroup ref="s:g"/></xs:complexType></xs:element>`
	awChoiceSchema = `<xs:element name="r"><xs:complexType><xs:choice><xs:element name="a" type="xs:string"/><xs:element name="b" type="xs:int"/></xs:choice><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`
	awNameSchema   = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence>
		<xs:attribute name="anyAttribute" type="xs:string"/><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`
	awElemNameSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="anyAttribute" type="xs:string"/></xs:sequence><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`
	awParticleSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:any processContents="lax"/></xs:sequence><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`
	awSimpleSchema   = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"><xs:anyAttribute processContents="lax"/></xs:extension></xs:simpleContent></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
	awNestedSchema   = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="a"><xs:complexType><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence><xs:anyAttribute processContents="lax"/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
	awStrictSchema   = `<xs:attribute name="g" type="xs:int"/><xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:anyAttribute namespace="##targetNamespace"/></xs:complexType></xs:element>`
	awEmptySchema    = `<xs:element name="r"><xs:complexType><xs:anyAttribute processContents="skip"/></xs:complexType></xs:element>`
	awDefaultDoc     = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:s="urn:s" targetNamespace="urn:s" elementFormDefault="qualified" defaultAttributes="s:dg">
		<xs:attributeGroup name="dg"><xs:anyAttribute processContents="lax"/></xs:attributeGroup>
		<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence></xs:complexType></xs:element></xs:schema>`
)

// The types codegen emits for those schemas' s:r, which
// TestAnyAttributeTypesAreCodegens pins, each wildcard field a slice of
// anyAttribute, of AnyAttribute's shape: awUse awUseSchema's R, which
// awGroupSchema's shares; awExt awExtSchema's, its own use b ahead of base's
// a as [xsd.ComplexType.AttributeUses] orders them; awOne awUnionSchema's,
// which awStrictSchema's and awDefaultDoc's share; awRestrict
// awRestrictSchema's, with no wildcard field; awChoice and its branches
// awChoiceSchema's R, RChoice, RChoiceA and RChoiceB; awName awNameSchema's
// and awElemName awElemNameSchema's, each wildcard field AnyAttribute2;
// awParticle awParticleSchema's; and awSimple and awNested awSimpleSchema's
// and awNestedSchema's, each A codegen's A; and awEmpty awEmptySchema's.
type (
	awUse struct {
		E            string
		A            *int32
		AnyAttribute []anyAttribute
	}
	awExt struct {
		E            string
		B            *string
		A            *int32
		AnyAttribute []anyAttribute
	}
	awOne struct {
		E            string
		AnyAttribute []anyAttribute
	}
	awRestrict struct {
		E string
		A *int32
	}
	awChoice struct {
		Choice       awChoiceI
		AnyAttribute []anyAttribute
	}
	awChoiceI interface{ isAwChoice() }
	awChoiceA struct{ A string }
	awChoiceB struct{ B int32 }
	awName    struct {
		E             string
		AnyAttribute  *string
		AnyAttribute2 []anyAttribute
	}
	awElemName struct {
		AnyAttribute  string
		AnyAttribute2 []anyAttribute
	}
	awParticle struct {
		Any          anyElement
		AnyAttribute []anyAttribute
	}
	awSimple struct {
		A struct {
			Value        string
			AnyAttribute []anyAttribute
		}
	}
	awEmpty  struct{ AnyAttribute []anyAttribute }
	awNested struct {
		A struct {
			B            string
			AnyAttribute []anyAttribute
		}
	}
)

func (awChoiceA) isAwChoice() {}
func (awChoiceB) isAwChoice() {}

// awSchema returns the schema of decls, a whole schema document where it
// opens with <xs:schema, as awDefaultDoc does, and otherwise schemaOf's.
func awSchema(t *testing.T, decls string) *xsd.Schema {
	t.Helper()
	if strings.HasPrefix(decls, "<xs:schema") {
		return parseString(t, decls)
	}
	return schemaOf(t, decls)
}

// TestAnyAttributeTypesAreCodegens pins the types above to those codegen
// emits for each {attribute wildcard} schema, in shape (codegen/doc.go
// "# Attributes").
func TestAnyAttributeTypesAreCodegens(t *testing.T) {
	use := "struct {\n\tE            string\n\tA            *int32\n\tAnyAttribute []AnyAttribute\n}\n"
	one := "type R struct {\n\tE            string\n\tAnyAttribute []AnyAttribute\n}\n"
	rows := []struct {
		decls string
		want  []string
	}{
		{awUseSchema, []string{"type R " + use, "type AnyAttribute struct {\n\tSpace, Local, Value string\n}\n"}},
		{awExtSchema, []string{"type R struct {\n\tE            string\n\tB            *string\n\tA            *int32\n\tAnyAttribute []AnyAttribute\n}\n"}},
		{awUnionSchema, []string{one}},
		{awRestrictSchema, []string{"type R struct {\n\tE string\n\tA *int32\n}\n"}},
		{awGroupSchema, []string{"type R " + use}},
		{awChoiceSchema, []string{
			"type R struct {\n\tChoice       RChoice\n\tAnyAttribute []AnyAttribute\n}\n",
			"type RChoiceA struct {\n\tA string\n}\n\nfunc (RChoiceA) isRChoice() {}\n",
			"type RChoiceB struct {\n\tB int32\n}\n\nfunc (RChoiceB) isRChoice() {}\n",
		}},
		{awNameSchema, []string{"type R struct {\n\tE             string\n\tAnyAttribute  *string\n\tAnyAttribute2 []AnyAttribute\n}\n"}},
		{awElemNameSchema, []string{"type R struct {\n\tAnyAttribute  string\n\tAnyAttribute2 []AnyAttribute\n}\n"}},
		{awParticleSchema, []string{"type R struct {\n\tAny          AnyElement\n\tAnyAttribute []AnyAttribute\n}\n"}},
		{awSimpleSchema, []string{"type R struct {\n\tA A\n}\n", "type A struct {\n\tValue        string\n\tAnyAttribute []AnyAttribute\n}\n"}},
		{awNestedSchema, []string{"type R struct {\n\tA A\n}\n", "type A struct {\n\tB            string\n\tAnyAttribute []AnyAttribute\n}\n"}},
		{awStrictSchema, []string{one}},
		{awDefaultDoc, []string{one}},
		{awEmptySchema, []string{"type R struct {\n\tAnyAttribute []AnyAttribute\n}\n"}},
	}
	for _, row := range rows {
		gen := generated(t, awSchema(t, row.decls))
		for _, decl := range row.want {
			if !bytes.Contains(gen, []byte(decl)) {
				t.Errorf("codegen emits\n%s\nwhich does not declare\n%s", gen, decl)
			}
		}
	}
}

// TestDecodeAnyAttribute pins the {attribute wildcard} lift: an element of a
// type with one, its own, inherited by an extension (dcl.ctd.anyatt 2.2.2),
// the union of an extension's and its base's (cos-aw-union), brought by
// <xs:attributeGroup ref> or by the schema's defaultAttributes
// (declare-attributeGroup-wildcard), decodes into codegen's struct, whose
// last field holds, in start-tag order, each present attribute no attribute
// use's declaration names (cvc-complex-type clause 2, a use taking precedence
// by clause 2.1), raw, whatever the wildcard's {process contents}, an xsi:
// attribute among them but for xsi:type, xsi:nil, xsi:schemaLocation and
// xsi:noNamespaceSchemaLocation, and is nil where none is present; a
// restriction inherits no wildcard (2.1), so its struct has no such field;
// the field is the struct's last after a choice's field, a wildcard
// particle's field and a simple content's value field, alone for empty
// content, and in a nested type's struct, and takes the name AnyAttribute2
// where an attribute or an element took AnyAttribute.
func TestDecodeAnyAttribute(t *testing.T) {
	xsi := "http://www.w3.org/2001/XMLSchema-instance"
	i32 := func(v int32) *int32 { return &v }
	str := func(s string) *string { return &s }
	e := "<s:e>v</s:e>"
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"a use beside the wildcard", awUseSchema, ` a="1" xmlns:p="urn:p" p:x="2" y="3"`, e, into[awUse](),
			awUse{E: "v", A: i32(1), AnyAttribute: []anyAttribute{{Space: "urn:p", Local: "x", Value: "2"}, {Local: "y", Value: "3"}}}},
		{"xsi:schemaLocation and xsi:noNamespaceSchemaLocation, which the field excludes", awUseSchema,
			` xsi:schemaLocation="urn:x x.xsd" y="3" xsi:noNamespaceSchemaLocation="n.xsd"`, e, into[awUse](),
			awUse{E: "v", AnyAttribute: []anyAttribute{{Local: "y", Value: "3"}}}},
		{"an xsi: attribute other than the four", awUseSchema, ` xsi:other="o"`, e, into[awUse](),
			awUse{E: "v", AnyAttribute: []anyAttribute{{Space: xsi, Local: "other", Value: "o"}}}},
		{"no attribute the wildcard takes", awUseSchema, ` a="1"`, e, into[awUse](), awUse{E: "v", A: i32(1)}},
		{"a value held raw, its prefix unresolved", awUseSchema, ` y=" p:q  r "`, e, into[awUse](),
			awUse{E: "v", AnyAttribute: []anyAttribute{{Local: "y", Value: " p:q  r "}}}},
		{"an extension inheriting its base's wildcard", awExtSchema, ` z="9" b="t" a="2"`, e, into[awExt](),
			awExt{E: "v", B: str("t"), A: i32(2), AnyAttribute: []anyAttribute{{Local: "z", Value: "9"}}}},
		{"an extension's wildcard united with its base's", awUnionSchema, ` xmlns:b="urn:b" xmlns:x="urn:x" x:q="1" b:p="2"`, e, into[awOne](),
			awOne{E: "v", AnyAttribute: []anyAttribute{{Space: "urn:x", Local: "q", Value: "1"}, {Space: "urn:b", Local: "p", Value: "2"}}}},
		{"a restriction, which inherits no wildcard", awRestrictSchema, ` a="3"`, e, into[awRestrict](), awRestrict{E: "v", A: i32(3)}},
		{"a wildcard an attribute group brings", awGroupSchema, ` w="5" a="4"`, e, into[awUse](),
			awUse{E: "v", A: i32(4), AnyAttribute: []anyAttribute{{Local: "w", Value: "5"}}}},
		{"a wildcard defaultAttributes brings", awDefaultDoc, ` w="5"`, e, into[awOne](),
			awOne{E: "v", AnyAttribute: []anyAttribute{{Local: "w", Value: "5"}}}},
		{"a strict wildcard's attribute, held raw", awStrictSchema, ` s:g=" 5"`, e, into[awOne](),
			awOne{E: "v", AnyAttribute: []anyAttribute{{Space: "urn:s", Local: "g", Value: " 5"}}}},
		{"a choice with no attribute use", awChoiceSchema, ` w="1"`, "<s:b>2</s:b>", into[awChoice](WithBranches[awChoiceI](awChoiceA{}, awChoiceB{})),
			awChoice{Choice: awChoiceB{B: 2}, AnyAttribute: []anyAttribute{{Local: "w", Value: "1"}}}},
		{"an attribute named anyAttribute", awNameSchema, ` w="1" anyAttribute="u"`, e, into[awName](),
			awName{E: "v", AnyAttribute: str("u"), AnyAttribute2: []anyAttribute{{Local: "w", Value: "1"}}}},
		{"an element named anyAttribute", awElemNameSchema, ` w="1"`, "<s:anyAttribute>x</s:anyAttribute>", into[awElemName](),
			awElemName{AnyAttribute: "x", AnyAttribute2: []anyAttribute{{Local: "w", Value: "1"}}}},
		{"beside a wildcard particle", awParticleSchema, ` w="1"`, "<s:p>x</s:p>", into[awParticle](),
			awParticle{Any: anyElement{Space: "urn:s", Local: "p", Content: anyType{Text: "x"}}, AnyAttribute: []anyAttribute{{Local: "w", Value: "1"}}}},
		{"simple content's", awSimpleSchema, "", `<s:a w="1">v</s:a>`, into[awSimple](),
			awSimple{A: struct {
				Value        string
				AnyAttribute []anyAttribute
			}{Value: "v", AnyAttribute: []anyAttribute{{Local: "w", Value: "1"}}}}},
		{"empty content's", awEmptySchema, ` w="1"`, "", into[awEmpty](), awEmpty{AnyAttribute: []anyAttribute{{Local: "w", Value: "1"}}}},
		{"a nested type's", awNestedSchema, "", `<s:a w="1"><s:b>x</s:b></s:a>`, into[awNested](),
			awNested{A: struct {
				B            string
				AnyAttribute []anyAttribute
			}{B: "x", AnyAttribute: []anyAttribute{{Local: "w", Value: "1"}}}}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := awSchema(t, row.decls)
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

// TestDecodeAnyAttributeTypeMismatch pins item 6 of Decode's doc for the
// {attribute wildcard}'s field: a T whose last field is not it, by count,
// name or Go type, is ErrTypeMismatch, no decline, and nothing is filled;
// and a type with no {attribute wildcard}, a restriction's of a base with
// one included, has no such field, so codegen's T passes CheckDecode and the
// same T with a trailing AnyAttribute field does not.
func TestDecodeAnyAttributeTypeMismatch(t *testing.T) {
	type short struct {
		E string
		A *int32
	}
	type misnamed struct {
		E   string
		A   *int32
		Any []anyAttribute
	}
	type ofString struct {
		E            string
		A            *int32
		AnyAttribute []string
	}
	type single struct {
		E            string
		A            *int32
		AnyAttribute anyAttribute
	}
	type extra struct {
		E            string
		A            *int32
		AnyAttribute []anyAttribute
	}
	plain := `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="xs:string"/></xs:sequence><xs:attribute name="a" type="xs:int"/></xs:complexType></xs:element>`
	data := instanceOf(` a="1"`, "<s:e>v</s:e>")
	rows := []struct {
		name  string
		decls string
		into  target
		want  string
	}{
		{"no wildcard field", awUseSchema, into[short](), "codec.short has 2 fields where codegen emits 3"},
		{"the wildcard field misnamed", awUseSchema, into[misnamed](),
			"field 2 of codec.misnamed is Any []codec.anyAttribute where codegen emits AnyAttribute, a slice of a struct of codegen's AnyAttribute shape"},
		{"the wildcard field a slice of no AnyAttribute shape", awUseSchema, into[ofString](),
			"field 2 of codec.ofString is AnyAttribute []string where codegen emits AnyAttribute, a slice of a struct of codegen's AnyAttribute shape"},
		{"the wildcard field no slice", awUseSchema, into[single](),
			"field 2 of codec.single is AnyAttribute codec.anyAttribute where codegen emits AnyAttribute, a slice of a struct of codegen's AnyAttribute shape"},
		{"a wildcard field where the type has no wildcard", plain, into[extra](), "codec.extra has 3 fields where codegen emits 2"},
		{"a wildcard field where a restriction inherits none", awRestrictSchema, into[extra](), "codec.extra has 3 fields where codegen emits 2"},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			mustAccept(t, schema, data)
			got, err := row.into.decode(schema, data)
			if got != nil || err == nil {
				t.Fatalf("Decode = (%v, %v), want no value and an error", got, err)
			}
			if want := ErrTypeMismatch.Error() + ": " + row.want; err.Error() != want {
				t.Errorf("error = %q, want %q", err, want)
			}
			var de *DeclineError
			if !errors.Is(err, ErrTypeMismatch) || errors.As(err, &de) {
				t.Errorf("error %v: want ErrTypeMismatch alone", err)
			}
			assertProbeMismatch(t, row.into.check(schema), err)
		})
	}
	if err := CheckDecode[short](schemaOf(t, plain), rootName); err != nil {
		t.Errorf("CheckDecode for a type with no wildcard = %v, want nil", err)
	}
	if err := CheckDecode[awRestrict](schemaOf(t, awRestrictSchema), rootName); err != nil {
		t.Errorf("CheckDecode for a restriction = %v, want nil", err)
	}
}
