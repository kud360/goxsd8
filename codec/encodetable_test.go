package codec

import (
	"bytes"
	"errors"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// The {type table} schemas the Append rows below read beside typetable_test.go's,
// each declaring s:r. ttZzSchema is cta0042's shape: r, of zz, simple content of
// xs:string with an xs:int attribute type, selects zz where type is 1, zzd, zz
// extended by an optional note, where it is 2, and xs:error otherwise, so that a
// zzd holding type 1 is written as a document valid against zz. ttLangSchema is
// cta0010's: r, whose lang is inheritable, holds repeated c, of no type, which
// selects xs:string where lang is "en", fr, a sequence of t, where it has
// another lang, and xs:error where it has none, read from r's. ttAnySchema's r
// of xs:anyType selects xs:string where its k is "s", and ttAnyChildSchema's r,
// carrying a k of its own, holds c of that table. ttAssertSchema's r of
// base selects asserted, base extended by z and asserting a is "ok", where it
// has a kind. ttWildSchema's r, of wb, a sequence of a with a skip {attribute
// wildcard}, selects wn, wb extended by n, where it carries w, an attribute
// only the wildcard admits. ttUndecidedSchema's r selects num by a {test} the CTA evaluator
// declines. ttFixedSchema's r, an xs:string, nillable and fixed "a", which
// goshape.Nillable reads as not nillable, selects xs:token where it carries x.
const (
	ttZzSchema = `<xs:complexType name="zz"><xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="type" type="xs:int"/></xs:extension></xs:simpleContent></xs:complexType>
<xs:complexType name="zzd"><xs:simpleContent><xs:extension base="s:zz"><xs:attribute name="note" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType>
<xs:element name="r" type="s:zz"><xs:alternative test="@type = 1" type="s:zz"/><xs:alternative test="@type = 2" type="s:zzd"/><xs:alternative type="xs:error"/></xs:element>`
	ttLangSchema = `<xs:complexType name="fr"><xs:sequence><xs:element name="t" type="xs:string"/></xs:sequence></xs:complexType>
<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="c" maxOccurs="unbounded"><xs:alternative test="@lang = 'en'" type="xs:string"/><xs:alternative test="@lang" type="s:fr"/><xs:alternative type="xs:error"/></xs:element></xs:sequence>
<xs:attribute name="lang" type="xs:string" inheritable="true"/></xs:complexType></xs:element>`
	ttAnySchema      = `<xs:element name="r"><xs:alternative test="@k = 's'" type="xs:string"/></xs:element>`
	ttAnyChildSchema = `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="c"><xs:alternative test="@k = 's'" type="xs:string"/></xs:element></xs:sequence><xs:attribute name="k" type="xs:string"/></xs:complexType></xs:element>`
	ttAssertSchema   = ttTypes + `<xs:complexType name="asserted"><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="z" type="xs:string"/></xs:sequence><xs:assert test="s:a = 'ok'"/></xs:extension></xs:complexContent></xs:complexType>
<xs:element name="r" type="s:base"><xs:alternative test="@kind" type="s:asserted"/></xs:element>`
	ttWildSchema = `<xs:complexType name="wb"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:anyAttribute namespace="##local" processContents="skip"/></xs:complexType>
<xs:complexType name="wn"><xs:complexContent><xs:extension base="s:wb"><xs:sequence><xs:element name="n" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
<xs:element name="r" type="s:wb"><xs:alternative test="@w" type="s:wn"/></xs:element>`
	ttUndecidedSchema = ttTypes + `<xs:element name="r" type="s:base"><xs:alternative test="count(@kind) > 0" type="s:num"/></xs:element>`
	ttFixedSchema     = `<xs:element name="r" type="xs:string" nillable="true" fixed="a"><xs:alternative test="@x" type="xs:token"/></xs:element>`
)

// The types codegen emits for the schemas above, which
// TestAppendTypeTableTypesAreCodegens pins: ttZzSchema's R, RZz, Zz, RZzd and
// Zzd are ttZz to ttZzdType; ttLangSchema's R, C, CString, CFr and Fr are
// ttLang to ttFr; ttAnySchema's R, RString and RAnyType are ttAny to
// ttAnyAnyType; ttAssertSchema's R, RAsserted, Asserted and RBase are ttAs to
// ttAsBase; ttWildSchema's R, RWn, Wn, RWb and Wb are ttWd to ttWb; and
// ttFixedSchema's R, RToken and RString are ttFx to ttFxString.
type (
	ttZz     interface{ isTtZz() }
	ttZzZz   struct{ Zz ttZzType }
	ttZzType struct {
		Value string
		Type  *int32
	}
	ttZzZzd   struct{ Zzd ttZzdType }
	ttZzdType struct {
		Value string
		Note  *string
		Type  *int32
	}

	ttLang struct {
		C    []ttC
		Lang *string
	}
	ttC       interface{ isTtC() }
	ttCString struct{ String string }
	ttCFr     struct{ Fr ttFr }
	ttFr      struct{ T string }

	ttAny        interface{ isTtAny() }
	ttAnyString  struct{ String string }
	ttAnyAnyType struct{ AnyType anyType }

	ttAnyCh struct {
		C ttAnyC
		K *string
	}
	ttAnyC        interface{ isTtAnyC() }
	ttAnyCString  struct{ String string }
	ttAnyCAnyType struct{ AnyType anyType }

	ttAs         interface{ isTtAs() }
	ttAsAsserted struct{ Asserted ttAsserted }
	ttAsserted   struct {
		A    string
		Z    string
		Kind *string
	}
	ttAsBase struct{ Base ttBase }

	ttWd   interface{ isTtWd() }
	ttWdWn struct{ Wn ttWn }
	ttWn   struct {
		A            string
		N            int32
		AnyAttribute []anyAttribute
	}
	ttWdWb struct{ Wb ttWb }
	ttWb   struct {
		A            string
		AnyAttribute []anyAttribute
	}

	ttFx       interface{ isTtFx() }
	ttFxToken  struct{ Token string }
	ttFxString struct{ String string }
)

func (ttZzZz) isTtZz()          {}
func (ttZzZzd) isTtZz()         {}
func (ttCString) isTtC()        {}
func (ttCFr) isTtC()            {}
func (ttAnyString) isTtAny()    {}
func (ttAnyAnyType) isTtAny()   {}
func (ttAnyCString) isTtAnyC()  {}
func (ttAnyCAnyType) isTtAnyC() {}
func (ttAsAsserted) isTtAs()    {}
func (ttAsBase) isTtAs()        {}
func (ttWdWn) isTtWd()          {}
func (ttWdWb) isTtWd()          {}
func (ttFxToken) isTtFx()       {}
func (ttFxString) isTtFx()      {}

// The branch sets of the interfaces above.
var (
	ttZzBranches   = WithBranches[ttZz](ttZzZz{}, ttZzZzd{})
	ttCBranches    = WithBranches[ttC](ttCString{}, ttCFr{})
	ttAnyBranches  = WithBranches[ttAny](ttAnyString{}, ttAnyAnyType{})
	ttAnyCBranches = WithBranches[ttAnyC](ttAnyCString{}, ttAnyCAnyType{})
	ttAsBranches   = WithBranches[ttAs](ttAsAsserted{}, ttAsBase{})
	ttFxBranches   = WithBranches[ttFx](ttFxToken{}, ttFxString{})
	ttWdBranches   = WithBranches[ttWd](ttWdWn{}, ttWdWb{})
	// ttNumBaseBranches are ttR's branches for a table of num and base alone.
	ttNumBaseBranches = WithBranches[ttR](ttRNum{}, ttRBase{})
)

// TestAppendTypeTableTypesAreCodegens pins the types above to those codegen
// emits for each schema, in shape.
func TestAppendTypeTableTypesAreCodegens(t *testing.T) {
	rows := []struct {
		decls string
		want  []string
	}{
		{ttZzSchema, []string{"type R interface {\n\tisR()\n}\n", "type RZz struct {\n\tZz Zz\n}\n", "type Zz struct {\n\tValue string\n\tType  *int32\n}\n",
			"type RZzd struct {\n\tZzd Zzd\n}\n", "type Zzd struct {\n\tValue string\n\tNote  *string\n\tType  *int32\n}\n"}},
		{ttLangSchema, []string{"type R struct {\n\tC    []C\n\tLang *string\n}\n", "type C interface {\n\tisC()\n}\n",
			"type CString struct {\n\tString string\n}\n", "type CFr struct {\n\tFr Fr\n}\n", "type Fr struct {\n\tT string\n}\n"}},
		{ttAnySchema, []string{"type R interface {\n\tisR()\n}\n", "type RString struct {\n\tString string\n}\n", "type RAnyType struct {\n\tAnyType AnyType\n}\n"}},
		{ttAnyChildSchema, []string{"type R struct {\n\tC C\n\tK *string\n}\n", "type C interface {\n\tisC()\n}\n", "type CString struct {\n\tString string\n}\n", "type CAnyType struct {\n\tAnyType AnyType\n}\n"}},
		{ttAssertSchema, []string{"type RAsserted struct {\n\tAsserted Asserted\n}\n", "type Asserted struct {\n\tA    string\n\tZ    string\n\tKind *string\n}\n", "type RBase struct {\n\tBase Base\n}\n"}},
		{ttWildSchema, []string{"type RWn struct {\n\tWn Wn\n}\n", "type Wn struct {\n\tA            string\n\tN            int32\n\tAnyAttribute []AnyAttribute\n}\n",
			"type RWb struct {\n\tWb Wb\n}\n", "type Wb struct {\n\tA            string\n\tAnyAttribute []AnyAttribute\n}\n"}},
		{ttFixedSchema, []string{"type R interface {\n\tisR()\n}\n", "type RToken struct {\n\tToken string\n}\n", "type RString struct {\n\tString string\n}\n"}},
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

// ttRound returns a row that round-trips v through Append and Decode under
// opts (roundTrip), the bytes want where want is not empty.
func ttRound[T any](v T, want string, opts ...Option) func(*testing.T, *xsd.Schema) {
	return func(t *testing.T, schema *xsd.Schema) {
		t.Helper()
		roundTrip(t, schema, rootName, v, want, deepEqual[T], opts...)
	}
}

// TestAppendTypeTable pins the {type table} lift of Append: each element
// whose declaration has a {type table} is written as the governing type of the
// branch its value holds, or as its one governing type, and validate accepts
// the document and Decode returns the value, at the document element, a local
// particle, an <element ref>, a choice's alternative and a substitution group's
// member; of a named, an anonymous, a simple and an xs:anyType arm; the first
// alternative whose {test} is true, a later one, and the {default type
// definition}, explicit and the declared type; its {test}s reading the
// attributes Append writes, an xs:anyType arm's raw ones among them, and the
// [inherited attributes] it composes as Decode does: a parent's, a
// grandparent's through a parent that hands down nothing of its own, and a
// parent's defaulted attribute, which Append writes, over a grandparent's.
func TestAppendTypeTable(t *testing.T) {
	str := func(s string) *string { return &s }
	one := int32(1)
	inh := []Option{ttWhenBranches, ttUtBranches, ttVBranches}
	rows := []struct {
		name  string
		decls string
		run   func(*testing.T, *xsd.Schema)
	}{
		{"document element, the first alternative", ttRootSchema, ttRound(ttR(ttRNum{Num: ttNum{A: "x", N: 1, Kind: str("num")}}), `<r xmlns="urn:s" kind="num"><a>x</a><n>1</n></r>`, ttRBranches)},
		{"document element, the second alternative", ttRootSchema, ttRound(ttR(ttRFlag{Flag: ttFlag{A: "x", F: true, Kind: str("y")}}), "", ttRBranches)},
		{"document element, the explicit default", ttRootSchema, ttRound(ttR(ttRBase{Base: ttBase{A: "x"}}), `<r xmlns="urn:s"><a>x</a></r>`, ttRBranches)},
		{"a repeated local element", ttLocalSchema, ttRound(ttLocal{E: []ttE{ttENum{Num: ttNum{A: "1", N: 2, Kind: str("num")}}, ttEBase{Base: ttBase{A: "3"}}}}, "", ttEBranches)},
		{"the declared type as the default no alternative names", ttLastTestSchema, ttRound(ttR(ttRBase{Base: ttBase{A: "x"}}), "", ttRBranches)},
		{"a last alternative with a test", ttLastTestSchema, ttRound(ttR(ttRFlag{Flag: ttFlag{A: "x", F: true, G: true, Kind: str("flag")}}), "", ttRBranches)},
		{"the one anonymous governing type", ttAnonSchema, ttRound(ttA(ttAAlt1{Alternative1: ttAAlt1Type{A: "x", X: "y", Kind: str("x")}}), "", ttABranches)},
		{"beside the one anonymous governing type", ttAnonSchema, ttRound(ttA(ttABase{Base: ttBase{A: "x", Kind: str("z")}}), "", ttABranches)},
		{"one governing type beside xs:error's slot", ttErrorSchema, ttRound(ttErr{A: "x"}, `<r xmlns="urn:s"><a>x</a></r>`)},
		{"a parent's inheritable attribute", ttInheritSchema, ttRound(ttInh{Ev: []ttEv{{When: ttWhenBoolean{Boolean: true}, Type: str("bool")}}}, "", inh...)},
		{"a grandparent's, through a parent handing down nothing of its own", ttInheritSchema, ttRound(ttInh{Ev: []ttEv{{When: ttWhenBoolean{Boolean: true}}}, Type: str("bool")}, "", inh...)},
		{"a sibling's own attribute, not handed to the next sibling", ttInheritSchema,
			ttRound(ttInh{Ev: []ttEv{{When: ttWhenInt{Int: 1}, Type: str("int")}, {When: ttWhenBoolean{Boolean: true}}}, Type: str("bool")}, "", inh...)},
		{"no inherited attribute, the default", ttInheritSchema, ttRound(ttInh{Ev: []ttEv{{When: ttWhenUt{Ut: ttUtInt(1)}}}}, "", inh...)},
		{"a parent's defaulted attribute, written, over a grandparent's", ttInheritSchema, ttRound(ttInh{Evd: &ttEvd{When: ttWhenInt{Int: 1}, Type: "int"}, Type: str("bool")}, "", inh...)},
		{"a branch of simple type", ttInheritSchema, ttRound(ttInh{V: ttVInt{Int: 5}, Type: str("int")}, "", inh...)},
		{"the declared branch of simple type", ttInheritSchema, ttRound(ttInh{V: ttVLong{Long: 5}}, "", inh...)},
		{"a choice's alternative", ttChoiceSchema, ttRound(ttCh{Choice: ttChWhen{When: ttWhenInt{Int: 7}}, Type: str("int")}, "", ttChBranches, ttWhenBranches, ttUtBranches)},
		{"a substitution group's member", ttGroupSchema, ttRound(ttGr{H: []ttGroup{ttGrWh{Wh: ttWhBoolean{Boolean: true}}, ttGrN{N: 2}}, Type: str("bool")}, "", ttGrBranches, ttWhBranches, ttUtBranches)},
		{"a substitution group's member, its default", ttGroupSchema, ttRound(ttGr{H: []ttGroup{ttGrWh{Wh: ttWhUt{Ut: ttUtInt(1)}}}}, "", ttGrBranches, ttWhBranches, ttUtBranches)},
		{"document element, a governing type sharing its name", ttNamesakeSchema, ttRound(ttNs(ttNsR{R: ttNum{A: "x", N: 1, Kind: str("x")}}), "", ttNsBranches)},
		{"an <element ref>, a governing type sharing its name", ttRefNamesakeSchema, ttRound(ttRefNs{E: ttRefNsEE{E: ttNum{A: "x", N: 1, Kind: str("x")}}}, "", ttRefNsBranches)},
		{"one governing type, xs:anyType on the document element", ttRowsSchemas[1], ttRound(rootAny{Text: "t"}, "")},
		{"one governing type, xs:anyType on a local element", ttRowsSchemas[3], ttRound(oneAny{A: anyType{Text: "t"}}, "")},
		{"an attribute of the arm's own selects it", ttZzSchema, ttRound(ttZz(ttZzZz{Zz: ttZzType{Value: "42", Type: &one}}), `<r xmlns="urn:s" type="1">42</r>`, ttZzBranches)},
		{"an inherited attribute selects a simple arm", ttLangSchema, ttRound(ttLang{C: []ttC{ttCString{String: "x"}}, Lang: str("en")}, "", ttCBranches)},
		{"an inherited attribute selects a complex arm", ttLangSchema, ttRound(ttLang{C: []ttC{ttCFr{Fr: ttFr{T: "y"}}}, Lang: str("de")}, "", ttCBranches)},
		{"an xs:anyType arm's raw attribute", ttAnySchema, ttRound(ttAny(ttAnyAnyType{AnyType: anyType{Attributes: []anyAttribute{{Local: "k", Value: "t"}}, Text: "u"}}), "", ttAnyBranches)},
		{"an xs:anyType arm's own attributes, not its parent's", ttAnyChildSchema, ttRound(ttAnyCh{C: ttAnyCAnyType{AnyType: anyType{Text: "u"}}, K: str("s")}, "", ttAnyCBranches)},
		{"an attribute an {attribute wildcard}'s field holds", ttWildSchema, ttRound(ttWd(ttWdWn{Wn: ttWn{A: "x", N: 1, AnyAttribute: []anyAttribute{{Local: "w", Value: "1"}}}}), `<r xmlns="urn:s" w="1"><a>x</a><n>1</n></r>`, ttWdBranches)},
		{"a fixed, nillable declaration goshape.Nillable reads as not nillable", ttFixedSchema, ttRound(ttFx(ttFxString{String: "a"}), `<r xmlns="urn:s">a</r>`, ttFxBranches)},
		{"an alternative type's {assertions}, kept", ttAssertSchema, ttRound(ttAs(ttAsAsserted{Asserted: ttAsserted{A: "ok", Z: "z", Kind: str("k")}}), "", ttAsBranches)},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			row.run(t, schemaOf(t, row.decls))
		})
	}
}

// ttRefused returns a row whose v Append refuses by item 7 of its doc,
// constructTypeTable, after CheckAppend's nil, its detail opening with want,
// and writes nothing.
func ttRefused[T any](v T, want string, opts ...Option) func(*testing.T, *xsd.Schema) {
	return func(t *testing.T, schema *xsd.Schema) {
		t.Helper()
		if err := CheckAppend[T](schema, rootName, opts...); err != nil {
			t.Fatalf("CheckAppend = %v, want nil: the refusal is the value's", err)
		}
		got, err := appendChecked(t, []byte("pre"), schema, strict.New(), rootName, v, opts...)
		d := asDecline(t, err, "codec does not encode "+constructTypeTable.name()+": "+want)
		if d.Construct() != (Construct{constructTypeTable}) {
			t.Errorf("Construct %v, want constructTypeTable's", d.Construct())
		}
		if string(got) != "pre" {
			t.Errorf("Append wrote %q", got)
		}
	}
}

// TestAppendTypeTableRefuses pins item 7 of Append's doc for a {type table}:
// a value whose branch's type the table does not select, over the attributes
// Append writes and the [inherited attributes] it composes, is refused, never
// written: another alternative's type, where the written document is valid
// against the type selected and would decode into its branch (cta0042's
// shape), and where the arm's own attributes select the other branch, an
// xs:anyType arm's raw ones among them; xs:error, by the arm's own attribute,
// of one governing type and of two, and by the absence of an inherited one
// (cta0010's shape); a branch that needs xsi:type, which Append never writes;
// and a selection the CTA evaluator leaves undecided, whatever the value.
func TestAppendTypeTableRefuses(t *testing.T) {
	str := func(s string) *string { return &s }
	one := int32(1)
	const r = "element {urn:s}r declared at "
	rows := []struct {
		name  string
		decls string
		run   func(*testing.T, *xsd.Schema)
	}{
		{"an alternative's branch its attributes select the other alternative of", ttZzSchema,
			ttRefused(ttZz(ttZzZzd{Zzd: ttZzdType{Value: "42", Type: &one}}), r, ttZzBranches)},
		{"the explicit default's branch carrying the first alternative's attribute", ttRootSchema,
			ttRefused(ttR(ttRBase{Base: ttBase{A: "x", Kind: str("num")}}), r, ttRBranches)},
		{"xs:error, one governing type", ttErrorSchema, ttRefused(ttErr{A: "x", Bad: str("1")}, r)},
		{"xs:error, two governing types", ttTypes + `<xs:element name="r" type="s:base"><xs:alternative test="@kind = 'e'" type="xs:error"/><xs:alternative test="@kind" type="s:num"/></xs:element>`,
			ttRefused(ttR(ttRBase{Base: ttBase{A: "x", Kind: str("e")}}), r, ttNumBaseBranches)},
		{"xs:error, no inherited attribute", ttLangSchema, ttRefused(ttLang{C: []ttC{ttCString{String: "x"}}}, "element {urn:s}c declared at ", ttCBranches)},
		{"another arm, an inherited attribute", ttLangSchema, ttRefused(ttLang{C: []ttC{ttCString{String: "x"}}, Lang: str("de")}, "element {urn:s}c declared at ", ttCBranches)},
		{"an xs:anyType arm's raw attribute selecting the simple one", ttAnySchema,
			ttRefused(ttAny(ttAnyAnyType{AnyType: anyType{Attributes: []anyAttribute{{Local: "k", Value: "s"}}}}), r, ttAnyBranches)},
		{"a simple arm whose {test} reads an attribute it cannot carry", ttAnySchema, ttRefused(ttAny(ttAnyString{String: "u"}), r, ttAnyBranches)},
		{"an {attribute wildcard}'s attribute selecting the other branch", ttWildSchema, ttRefused(ttWd(ttWdWb{Wb: ttWb{A: "x", AnyAttribute: []anyAttribute{{Local: "w", Value: "1"}}}}), r, ttWdBranches)},
		{"a branch only xsi:type selects", ttSimpleSchema, ttRefused(ttSimple(ttSimpleInteger{}), r, ttSimpleBranches)},
		{"a fixed, nillable declaration's arm its attributes do not select", ttFixedSchema, ttRefused(ttFx(ttFxToken{Token: "a"}), r, ttFxBranches)},
		{"an undecided selection", ttUndecidedSchema, ttRefused(ttR(ttRBase{Base: ttBase{A: "x"}}), r, ttNumBaseBranches)},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			row.run(t, schemaOf(t, row.decls))
		})
	}
	// The detail names the slot written and the slot selected.
	_, err := Append(nil, schemaOf(t, ttZzSchema), strict.New(), rootName, ttZz(ttZzZzd{Zzd: ttZzdType{Value: "42", Type: &one}}), ttZzBranches)
	const want = "at ttZz.(ttZzZzd).Zzd written as the type of its {type table} alternative 2: over the attributes Append writes and its [inherited attributes], its {type table} selects the type of its {type table} alternative 1, the governing type of another branch (Structures §3.3.4.1 key-cta-select)"
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte(want)) {
		t.Errorf("Append = %v, want a detail holding %q", err, want)
	}
}

// TestAppendTypeTableInvalid pins item 6 of Append's doc for a {type table}'s
// interface: nil, where nil does not mean absent, and a dynamic type the
// WithBranches set places at no branch, a pointer to a branch, each of which
// Decode never returns, and nothing is written.
func TestAppendTypeTableInvalid(t *testing.T) {
	schema := schemaOf(t, ttRootSchema)
	got, err := appendChecked(t, nil, schema, strict.New(), rootName, ttR(nil), ttRBranches)
	invalidValue(t, err, "codec: Append: ttR is nil, the sealed interface over the governing types of element {urn:s}r declared at ")
	if got != nil {
		t.Errorf("Append wrote %q", got)
	}
	_, err = appendChecked(t, nil, schema, strict.New(), rootName, ttR(&ttRNum{}), ttRBranches)
	invalidValue(t, err, "codec: Append: ttR holds a *codec.ttRNum, which the WithBranches set of codec.ttR places at no branch of the governing types of element {urn:s}r declared at ")
}

// TestAppendTypeTableAssertions pins item 4's walk of every governing type:
// an alternative's type with {assertions} makes Append assess what it writes,
// so that a value breaking them is item 8, ErrInvalidValue, where a walk of
// the declaration's {type definition} alone, base, would write it unassessed.
func TestAppendTypeTableAssertions(t *testing.T) {
	str := func(s string) *string { return &s }
	_, err := appendChecked(t, nil, schemaOf(t, ttAssertSchema), strict.New(), rootName, ttAs(ttAsAsserted{Asserted: ttAsserted{A: "bad", Z: "z", Kind: str("k")}}), ttAsBranches)
	if !errors.Is(err, ErrInvalidValue) || ViolationsOf(err) == nil {
		t.Errorf("Append = %v, want item 8's ErrInvalidValue", err)
	}
}
