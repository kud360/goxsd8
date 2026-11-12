package codec

import (
	"bytes"
	"errors"
	"math/big"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// ttTypes declares base, a sequence of a, an xs:string, carrying an optional
// kind, and its extensions num, adding n, an xs:int, and flag, adding f and g,
// each an xs:boolean: three distinct names and three distinct shapes.
const ttTypes = `<xs:complexType name="base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="kind" type="xs:string"/></xs:complexType>
<xs:complexType name="num"><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="n" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>
<xs:complexType name="flag"><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="f" type="xs:boolean"/><xs:element name="g" type="xs:boolean"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>`

// ttWhen declares when, of ut, a union of xs:int and xs:boolean, whose {type
// table} selects xs:boolean where its type attribute is "bool" and xs:int
// where it is "int", an attribute when itself never carries, and ut by
// default, which decodes either value too.
const ttWhen = `<xs:element name="when" type="s:ut"><xs:alternative test="@type = 'bool'" type="xs:boolean"/><xs:alternative test="@type = 'int'" type="xs:int"/></xs:element>
<xs:simpleType name="ut"><xs:union memberTypes="xs:int xs:boolean"/></xs:simpleType>`

// The {type table} schemas, each declaring s:r. ttRootSchema's r, of base,
// selects num where its kind is "num", flag where it has a kind, which "num"
// satisfies too, and its explicit default base otherwise. ttLocalSchema's r
// holds a repeated local e of that table but flag's. ttLastTestSchema's r
// selects num and flag by test alone, so its default is its declared base,
// which no alternative names (§3.3.2.1). ttAnonSchema's r selects an
// anonymous extension of base where kind is "x", its one anonymous governing
// type, and base otherwise. ttErrorSchema's r of rt selects xs:error where it
// carries bad, so rt is its one governing type (codegen/doc.go "# Element
// declarations"). ttInheritSchema's r holds ev and evd, each holding when,
// and v, which selects xs:int where its type is "int" and its declared xs:long
// otherwise, defaulting to "5": r's, ev's and evd's type are inheritable, and
// evd's defaults to "int" (key-p-inherited, e-inherited_attributes).
// ttChoiceSchema's r holds when as a choice's alternative beside z, an
// xs:string, and ttGroupSchema's r a ref to h, an abstract head of type ut
// whose members are wh, selecting xs:boolean as when does, and n, an xs:int;
// r's type is inheritable in both.
const (
	ttRootSchema     = ttTypes + `<xs:element name="r" type="s:base"><xs:alternative test="@kind = 'num'" type="s:num"/><xs:alternative test="@kind" type="s:flag"/><xs:alternative type="s:base"/></xs:element>`
	ttLocalSchema    = ttTypes + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element name="e" type="s:base" maxOccurs="unbounded"><xs:alternative test="@kind = 'num'" type="s:num"/><xs:alternative type="s:base"/></xs:element></xs:sequence></xs:complexType></xs:element>`
	ttLastTestSchema = ttTypes + `<xs:element name="r" type="s:base"><xs:alternative test="@kind = 'num'" type="s:num"/><xs:alternative test="@kind = 'flag'" type="s:flag"/></xs:element>`
	ttAnonSchema     = ttTypes + `<xs:element name="r" type="s:base"><xs:alternative test="@kind = 'x'"><xs:complexType><xs:complexContent><xs:extension base="s:base"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType></xs:alternative></xs:element>`
	ttErrorSchema    = `<xs:complexType name="rt"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="bad" type="xs:string"/></xs:complexType>
<xs:element name="r" type="s:rt"><xs:alternative test="@bad" type="xs:error"/><xs:alternative type="s:rt"/></xs:element>`
	ttInheritSchema = ttWhen + `<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:ev" minOccurs="0" maxOccurs="unbounded"/><xs:element ref="s:evd" minOccurs="0"/>
<xs:element name="v" type="xs:long" default="5" minOccurs="0"><xs:alternative test="@type = 'int'" type="xs:int"/></xs:element></xs:sequence>
<xs:attribute name="type" type="xs:string" inheritable="true"/></xs:complexType></xs:element>
<xs:element name="ev"><xs:complexType><xs:sequence><xs:element ref="s:when"/></xs:sequence><xs:attribute name="type" type="xs:string" inheritable="true"/></xs:complexType></xs:element>
<xs:element name="evd"><xs:complexType><xs:sequence><xs:element ref="s:when"/></xs:sequence><xs:attribute name="type" type="xs:string" default="int" inheritable="true"/></xs:complexType></xs:element>`
	ttChoiceSchema = ttWhen + `<xs:element name="r"><xs:complexType><xs:choice><xs:element ref="s:when"/><xs:element name="z" type="xs:string"/></xs:choice><xs:attribute name="type" type="xs:string" inheritable="true"/></xs:complexType></xs:element>`
	ttGroupSchema  = ttWhen + `<xs:element name="h" type="s:ut" abstract="true"/><xs:element name="wh" type="s:ut" substitutionGroup="s:h"><xs:alternative test="@type = 'bool'" type="xs:boolean"/></xs:element><xs:element name="n" type="xs:int" substitutionGroup="s:h"/>
<xs:element name="r"><xs:complexType><xs:sequence><xs:element ref="s:h" maxOccurs="unbounded"/></xs:sequence><xs:attribute name="type" type="xs:string" inheritable="true"/></xs:complexType></xs:element>`
)

// ttRowsSchemas are the five {type table}s TestDecodeDeclines held as
// declines before tables decoded, each of whose governing types collapse to
// one: rt, a sequence of a, an xs:string, on the document element;
// xs:anyType on the document element, declared and implicit, the second
// nillable; and xs:anyType on a local a, the second nillable.
var ttRowsSchemas = [...]string{
	`<xs:complexType name="rt"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
<xs:element name="r" type="s:rt"><xs:alternative test="@k" type="s:rt"/></xs:element>`,
	`<xs:element name="r" type="xs:anyType"><xs:alternative test="@k" type="xs:anyType"/></xs:element>`,
	`<xs:element name="r" nillable="true"><xs:alternative test="@k" type="xs:anyType"/></xs:element>`,
	sequenceOf(`<xs:element name="a"><xs:alternative test="@k" type="xs:anyType"/></xs:element>`),
	sequenceOf(`<xs:element name="a" type="xs:anyType" nillable="true"><xs:alternative test="@k" type="xs:anyType"/></xs:element>`),
}

// The types codegen emits for the schemas above, which
// TestTypeTableTypesAreCodegens pins: ttBase, ttNum and ttFlag are Base, Num
// and Flag; ttRootSchema's and ttLastTestSchema's R, RNum, RFlag and RBase
// are ttR to ttRBase; ttLocalSchema's R, E, ENum and EBase are ttLocal to
// ttEBase; ttAnonSchema's R, RAlternative1, RAlternative1Type and RBase are
// ttA to ttABase; ttErrorSchema's R is ttErr; when's When, WhenBoolean,
// WhenInt and WhenUt, and ut's Ut, UtInt and UtBoolean, are ttWhenI to
// ttUtBoolean; ttInheritSchema's R, Ev, Evd, V, VInt and VLong are ttInh to
// ttVLong; ttChoiceSchema's R, RChoice, RChoiceWhen and RChoiceZ are ttCh to
// ttChZ; ttGroupSchema's R, HSubstitutionGroup, its branches, Wh, WhBoolean,
// WhUt and N are ttGr to ttN; and ttRowsSchemas[0]'s R is ttRow.
type (
	ttBase struct {
		A    string
		Kind *string
	}
	ttNum struct {
		A    string
		N    int32
		Kind *string
	}
	ttFlag struct {
		A    string
		F    bool
		G    bool
		Kind *string
	}
	ttR     interface{ isTtR() }
	ttRNum  struct{ Num ttNum }
	ttRFlag struct{ Flag ttFlag }
	ttRBase struct{ Base ttBase }

	ttLocal struct{ E []ttE }
	ttE     interface{ isTtE() }
	ttENum  struct{ Num ttNum }
	ttEBase struct{ Base ttBase }

	ttA         interface{ isTtA() }
	ttAAlt1     struct{ Alternative1 ttAAlt1Type }
	ttAAlt1Type struct {
		A    string
		X    string
		Kind *string
	}
	ttABase struct{ Base ttBase }

	ttErr struct {
		A   string
		Bad *string
	}

	ttWhenI       interface{ isTtWhen() }
	ttWhenBoolean struct{ Boolean bool }
	ttWhenInt     struct{ Int int32 }
	ttWhenUt      struct{ Ut ttUt }
	ttUt          interface{ isTtUt() }
	ttUtInt       int32
	ttUtBoolean   bool

	ttInh struct {
		Ev   []ttEv
		Evd  *ttEvd
		V    ttV
		Type *string
	}
	ttEv struct {
		When ttWhenI
		Type *string
	}
	ttEvd struct {
		When ttWhenI
		Type string
	}
	ttV     interface{ isTtV() }
	ttVInt  struct{ Int int32 }
	ttVLong struct{ Long int64 }

	ttCh struct {
		Choice ttChoice
		Type   *string
	}
	ttChoice interface{ isTtChoice() }
	ttChWhen struct{ When ttWhenI }
	ttChZ    struct{ Z string }

	ttGr struct {
		H    []ttGroup
		Type *string
	}
	ttGroup     interface{ isTtGroup() }
	ttGrWh      struct{ Wh ttWh }
	ttGrN       struct{ N ttN }
	ttWh        interface{ isTtWh() }
	ttWhBoolean struct{ Boolean bool }
	ttWhUt      struct{ Ut ttUt }
	ttN         int32

	ttRow struct{ A string }
)

func (ttRNum) isTtR()           {}
func (ttRFlag) isTtR()          {}
func (ttRBase) isTtR()          {}
func (ttENum) isTtE()           {}
func (ttEBase) isTtE()          {}
func (ttAAlt1) isTtA()          {}
func (ttABase) isTtA()          {}
func (ttWhenBoolean) isTtWhen() {}
func (ttWhenInt) isTtWhen()     {}
func (ttWhenUt) isTtWhen()      {}
func (ttUtInt) isTtUt()         {}
func (ttUtBoolean) isTtUt()     {}
func (ttVInt) isTtV()           {}
func (ttVLong) isTtV()          {}
func (ttChWhen) isTtChoice()    {}
func (ttChZ) isTtChoice()       {}
func (ttGrWh) isTtGroup()       {}
func (ttGrN) isTtGroup()        {}
func (ttWhBoolean) isTtWh()     {}
func (ttWhUt) isTtWh()          {}

// The branch sets of the interfaces above.
var (
	ttRBranches    = WithBranches[ttR](ttRNum{}, ttRFlag{}, ttRBase{})
	ttEBranches    = WithBranches[ttE](ttENum{}, ttEBase{})
	ttABranches    = WithBranches[ttA](ttAAlt1{}, ttABase{})
	ttWhenBranches = WithBranches[ttWhenI](ttWhenBoolean{}, ttWhenInt{}, ttWhenUt{})
	ttUtBranches   = WithBranches[ttUt](ttUtInt(0), ttUtBoolean(false))
	ttVBranches    = WithBranches[ttV](ttVInt{}, ttVLong{})
	ttChBranches   = WithBranches[ttChoice](ttChWhen{}, ttChZ{})
	ttGrBranches   = WithBranches[ttGroup](ttGrWh{}, ttGrN{})
	ttWhBranches   = WithBranches[ttWh](ttWhBoolean{}, ttWhUt{})
)

// TestTypeTableTypesAreCodegens pins the types above to those codegen emits
// for each schema, in shape, so that the rows decoding into them decode into
// codegen's types.
func TestTypeTableTypesAreCodegens(t *testing.T) {
	const (
		base   = "type Base struct {\n\tA    string\n\tKind *string\n}\n"
		num    = "type Num struct {\n\tA    string\n\tN    int32\n\tKind *string\n}\n"
		flag   = "type Flag struct {\n\tA    string\n\tF    bool\n\tG    bool\n\tKind *string\n}\n"
		r      = "type R interface {\n\tisR()\n}\n"
		rNum   = "type RNum struct {\n\tNum Num\n}\n\nfunc (RNum) isR() {}\n"
		rFlag  = "type RFlag struct {\n\tFlag Flag\n}\n\nfunc (RFlag) isR() {}\n"
		rBase  = "type RBase struct {\n\tBase Base\n}\n\nfunc (RBase) isR() {}\n"
		when   = "type When interface {\n\tisWhen()\n}\n"
		whenB  = "type WhenBoolean struct {\n\tBoolean bool\n}\n\nfunc (WhenBoolean) isWhen() {}\n"
		whenI  = "type WhenInt struct {\n\tInt int32\n}\n\nfunc (WhenInt) isWhen() {}\n"
		whenU  = "type WhenUt struct {\n\tUt Ut\n}\n\nfunc (WhenUt) isWhen() {}\n"
		ut     = "type Ut interface {\n\tisUt()\n}\n"
		utInt  = "type UtInt int32\n\nfunc (UtInt) isUt() {}\n"
		utBool = "type UtBoolean bool\n\nfunc (UtBoolean) isUt() {}\n"
	)
	rows := []struct {
		decls string
		want  []string
	}{
		{ttRootSchema, []string{r, rNum, num, rFlag, flag, rBase, base}},
		{ttLastTestSchema, []string{r, rNum, num, rFlag, flag, rBase, base}},
		{ttLocalSchema, []string{"type R struct {\n\tE []E\n}\n", "type E interface {\n\tisE()\n}\n",
			"type ENum struct {\n\tNum Num\n}\n\nfunc (ENum) isE() {}\n", "type EBase struct {\n\tBase Base\n}\n\nfunc (EBase) isE() {}\n", num, base}},
		{ttAnonSchema, []string{r, "type RAlternative1 struct {\n\tAlternative1 RAlternative1Type\n}\n\nfunc (RAlternative1) isR() {}\n",
			"type RAlternative1Type struct {\n\tA    string\n\tX    string\n\tKind *string\n}\n", rBase, base}},
		{ttErrorSchema, []string{"type R struct {\n\tA   string\n\tBad *string\n}\n"}},
		{ttInheritSchema, []string{"type R struct {\n\tEv   []Ev\n\tEvd  *Evd\n\tV    V\n\tType *string\n}\n",
			"type V interface {\n\tisV()\n}\n", "type VInt struct {\n\tInt int32\n}\n\nfunc (VInt) isV() {}\n", "type VLong struct {\n\tLong int64\n}\n\nfunc (VLong) isV() {}\n",
			"type Ev struct {\n\tWhen When\n\tType *string\n}\n", "type Evd struct {\n\tWhen When\n\tType string\n}\n",
			when, whenB, whenI, whenU, ut, utInt, utBool}},
		{ttChoiceSchema, []string{"type R struct {\n\tChoice RChoice\n\tType   *string\n}\n", "type RChoice interface {\n\tisRChoice()\n}\n",
			"type RChoiceWhen struct {\n\tWhen When\n}\n\nfunc (RChoiceWhen) isRChoice() {}\n", "type RChoiceZ struct {\n\tZ string\n}\n\nfunc (RChoiceZ) isRChoice() {}\n",
			when, whenB, whenI, whenU, ut, utInt, utBool}},
		{ttGroupSchema, []string{"type R struct {\n\tH    []HSubstitutionGroup\n\tType *string\n}\n",
			"type HSubstitutionGroup interface {\n\tisHSubstitutionGroup()\n}\n",
			"type HSubstitutionGroupWh struct {\n\tWh Wh\n}\n\nfunc (HSubstitutionGroupWh) isHSubstitutionGroup() {}\n",
			"type HSubstitutionGroupN struct {\n\tN N\n}\n\nfunc (HSubstitutionGroupN) isHSubstitutionGroup() {}\n",
			"type Wh interface {\n\tisWh()\n}\n", "type WhBoolean struct {\n\tBoolean bool\n}\n\nfunc (WhBoolean) isWh() {}\n",
			"type WhUt struct {\n\tUt Ut\n}\n\nfunc (WhUt) isWh() {}\n", "type N int32\n", ut, utInt, utBool}},
		{ttRowsSchemas[0], []string{"type R struct {\n\tA string\n}\n"}},
		{ttRowsSchemas[1], []string{"type R AnyType\n"}},
		{ttRowsSchemas[2], []string{"type R AnyType\n"}},
		{ttRowsSchemas[3], []string{"type R struct {\n\tA AnyType\n}\n"}},
		{ttRowsSchemas[4], []string{"type R struct {\n\tA AnyType\n}\n"}},
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

// TestDecodeTypeTable pins the {type table} lift: each element whose
// declaration has two or more governing types fills the branch of the type
// validate.Validator.ConditionallySelected selects (§3.3.4.1
// key-cta-select), the first alternative whose {test} is true, the {default
// type definition}'s where none is, at the document element, a local
// particle, an <element ref>, a choice's alternative and a substitution
// group's member; its {test}s read the element's [inherited attributes] as
// validate.Validator.HandedDown composes them (e-inherited_attributes): a
// parent's, a grandparent's through a parent that hands down nothing of its
// own, and a parent's ·defaulted attribute·, which shadows the grandparent's;
// a branch of simple type holds its declaration's default; and a table that
// collapses to one governing type decodes as that type, xs:error's slot
// giving none.
func TestDecodeTypeTable(t *testing.T) {
	str := func(s string) *string { return &s }
	inh := into[ttInh](ttWhenBranches, ttUtBranches, ttVBranches)
	rows := []struct {
		name  string
		decls string
		attrs string
		body  string
		into  target
		want  any
	}{
		{"document element, an instance satisfying two {test}s takes the first", ttRootSchema, ` kind="num"`, "<s:a>x</s:a><s:n>1</s:n>", into[ttR](ttRBranches),
			ttR(ttRNum{Num: ttNum{A: "x", N: 1, Kind: str("num")}})},
		{"document element, the second alternative", ttRootSchema, ` kind="y"`, "<s:a>x</s:a><s:f>true</s:f><s:g>0</s:g>", into[ttR](ttRBranches),
			ttR(ttRFlag{Flag: ttFlag{A: "x", F: true, Kind: str("y")}})},
		{"document element, the explicit default", ttRootSchema, "", "<s:a>x</s:a>", into[ttR](ttRBranches),
			ttR(ttRBase{Base: ttBase{A: "x"}})},
		{"a repeated local element", ttLocalSchema, "", `<s:e kind="num"><s:a>1</s:a><s:n>2</s:n></s:e><s:e><s:a>3</s:a></s:e>`, into[ttLocal](ttEBranches),
			ttLocal{E: []ttE{ttENum{Num: ttNum{A: "1", N: 2, Kind: str("num")}}, ttEBase{Base: ttBase{A: "3"}}}}},
		{"the declared type as the default no alternative names", ttLastTestSchema, "", "<s:a>x</s:a>", into[ttR](ttRBranches),
			ttR(ttRBase{Base: ttBase{A: "x"}})},
		{"a last alternative with a test", ttLastTestSchema, ` kind="flag"`, "<s:a>x</s:a><s:f>1</s:f><s:g>1</s:g>", into[ttR](ttRBranches),
			ttR(ttRFlag{Flag: ttFlag{A: "x", F: true, G: true, Kind: str("flag")}})},
		{"the one anonymous governing type", ttAnonSchema, ` kind="x"`, "<s:a>x</s:a><s:x>y</s:x>", into[ttA](ttABranches),
			ttA(ttAAlt1{Alternative1: ttAAlt1Type{A: "x", X: "y", Kind: str("x")}})},
		{"beside the one anonymous governing type", ttAnonSchema, ` kind="z"`, "<s:a>x</s:a>", into[ttA](ttABranches),
			ttA(ttABase{Base: ttBase{A: "x", Kind: str("z")}})},
		{"xs:error's slot gives no branch", ttErrorSchema, "", "<s:a>x</s:a>", into[ttErr](), ttErr{A: "x"}},
		{"a parent's inheritable attribute", ttInheritSchema, "", `<s:ev type="bool"><s:when>1</s:when></s:ev>`, inh,
			ttInh{Ev: []ttEv{{When: ttWhenBoolean{Boolean: true}, Type: str("bool")}}}},
		{"a grandparent's, through a parent handing down nothing of its own", ttInheritSchema, ` type="bool"`, `<s:ev><s:when>1</s:when></s:ev>`, inh,
			ttInh{Ev: []ttEv{{When: ttWhenBoolean{Boolean: true}}}, Type: str("bool")}},
		{"no inherited attribute, the default", ttInheritSchema, "", `<s:ev><s:when>1</s:when></s:ev>`, inh,
			ttInh{Ev: []ttEv{{When: ttWhenUt{Ut: ttUtInt(1)}}}}},
		{"a parent's defaulted attribute over a grandparent's", ttInheritSchema, ` type="bool"`, `<s:evd><s:when>1</s:when></s:evd>`, inh,
			ttInh{Evd: &ttEvd{When: ttWhenInt{Int: 1}, Type: "int"}, Type: str("bool")}},
		{"a branch of simple type holding its declaration's default", ttInheritSchema, ` type="int"`, `<s:v/>`, inh,
			ttInh{V: ttVInt{Int: 5}, Type: str("int")}},
		{"the declared branch holding its declaration's default", ttInheritSchema, "", `<s:v/>`, inh,
			ttInh{V: ttVLong{Long: 5}}},
		{"a choice's alternative", ttChoiceSchema, ` type="int"`, `<s:when>7</s:when>`, into[ttCh](ttChBranches, ttWhenBranches, ttUtBranches),
			ttCh{Choice: ttChWhen{When: ttWhenInt{Int: 7}}, Type: str("int")}},
		{"a substitution group's member", ttGroupSchema, ` type="bool"`, `<s:wh>1</s:wh><s:n>2</s:n>`, into[ttGr](ttGrBranches, ttWhBranches, ttUtBranches),
			ttGr{H: []ttGroup{ttGrWh{Wh: ttWhBoolean{Boolean: true}}, ttGrN{N: 2}}, Type: str("bool")}},
		{"a substitution group's member, its default", ttGroupSchema, "", `<s:wh>1</s:wh>`, into[ttGr](ttGrBranches, ttWhBranches, ttUtBranches),
			ttGr{H: []ttGroup{ttGrWh{Wh: ttWhUt{Ut: ttUtInt(1)}}}}},
		{"one governing type, a named complex type", ttRowsSchemas[0], "", "<s:a>x</s:a>", into[ttRow](), ttRow{A: "x"}},
		{"one governing type, xs:anyType on the document element", ttRowsSchemas[1], "", "t", into[rootAny](), rootAny{Text: "t"}},
		{"one governing type, xs:anyType on a nillable document element", ttRowsSchemas[2], "", "t", into[rootAny](), rootAny{Text: "t"}},
		{"one governing type, xs:anyType on a local element", ttRowsSchemas[3], "", "<s:a>t</s:a>", into[oneAny](), oneAny{A: anyType{Text: "t"}}},
		{"one governing type, xs:anyType on a nillable local element", ttRowsSchemas[4], "", "<s:a>t</s:a>", into[oneAny](), oneAny{A: anyType{Text: "t"}}},
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

// TestDecodeTypeTableRefuses pins what a {type table} does not decode: an
// instance selecting xs:error, which no instance is valid against (§2.2.1.1
// key-error), returns validate's violation and no value; a withheld selection
// returns Decode's item 4; and an xsi:type on an element whose declaration has
// two governing types declines as xsi:type, before any selection (item 7 of
// Decode's doc), while CheckDecode, reading no instance, returns nil.
func TestDecodeTypeTableRefuses(t *testing.T) {
	data := instanceOf(` bad="1"`, "<s:a>x</s:a>")
	if v, err := Decode[ttErr](schemaOf(t, ttErrorSchema), strict.New(), data); v != nil || ViolationsOf(err) == nil {
		t.Errorf("Decode(xs:error selected) = %v, %v; want validate's violation", v, err)
	}
	schema := schemaOf(t, ttRootSchema)
	data = instanceOf(` kind="num" xsi:type="s:num"`, "<s:a>x</s:a><s:n>1</s:n>")
	mustAccept(t, schema, data)
	_, err := Decode[ttR](schema, strict.New(), data, ttRBranches)
	assertDecline(t, err, constructXsiType, 1, 1)
	var de *DeclineError
	if !errors.As(err, &de) {
		t.Fatalf("Decode = %v, want a *DeclineError", err)
	}
	if err := CheckDecode[ttR](schema, rootName, ttRBranches); err != nil {
		t.Errorf("CheckDecode = %v, want nil: xsi:type is the instance's", err)
	}
	// A {test} the engine cannot evaluate, reached before any succeeds,
	// withholds the selection: Assess records it, and Decode returns item 4's
	// *UndecidedError before any branch is selected.
	schema = schemaOf(t, ttTypes+`<xs:element name="r" type="s:base"><xs:alternative test="count(@kind) > 0" type="s:num"/></xs:element>`)
	var ue *UndecidedError
	if _, err := Decode[ttR](schema, strict.New(), instanceOf("", "<s:a>x</s:a>"), ttRBranches); !errors.As(err, &ue) {
		t.Errorf("Decode(withheld selection) = %v, want an *UndecidedError", err)
	}
	// A branch of simple type reads no xsi:type of its own, so the decline is
	// the interface's: r selects xs:integer on its xsi:type, which names the
	// type selected and so is valid.
	schema = schemaOf(t, ttSimpleSchema)
	data = instanceOf(` xsi:type="xs:integer"`, "5")
	mustAccept(t, schema, data)
	_, err = Decode[ttSimple](schema, strict.New(), data, ttSimpleBranches)
	assertDecline(t, err, constructXsiType, 1, 1)
}

// ttSimpleSchema's r, an xs:decimal, selects xs:integer where it carries
// xsi:type; ttSimple, ttSimpleInteger and ttSimpleDecimal are its R, RInteger
// and RDecimal.
const ttSimpleSchema = `<xs:element name="r" type="xs:decimal"><xs:alternative test="@xsi:type" type="xs:integer"/></xs:element>`

type (
	ttSimple        interface{ isTtSimple() }
	ttSimpleInteger struct{ Integer big.Int }
	ttSimpleDecimal struct{ Decimal big.Rat }
)

func (ttSimpleInteger) isTtSimple() {}
func (ttSimpleDecimal) isTtSimple() {}

var ttSimpleBranches = WithBranches[ttSimple](ttSimpleInteger{}, ttSimpleDecimal{})

// TestTypeTableKeysItsOneType pins that a local declaration whose table
// collapses to one named type other than its declared one binds that type's
// struct, keyed by that type's name: e, declared base, selects num alone, and
// f, a local base beside it, keeps base's struct.
func TestTypeTableKeysItsOneType(t *testing.T) {
	schema := schemaOf(t, ttTypes+`<xs:element name="r"><xs:complexType><xs:sequence>
<xs:element name="e" type="s:base"><xs:alternative test="@kind" type="s:num"/><xs:alternative type="s:num"/></xs:element>
<xs:element name="f" type="s:base"/></xs:sequence></xs:complexType></xs:element>`)
	type pair struct {
		E ttNum
		F ttBase
	}
	if !bytes.Contains(generated(t, schema), []byte("type R struct {\n\tE Num\n\tF Base\n}\n")) {
		t.Fatalf("codegen emits\n%s\nwhich does not declare R{E Num; F Base}", generated(t, schema))
	}
	data := instanceOf("", "<s:e><s:a>1</s:a><s:n>2</s:n></s:e><s:f><s:a>3</s:a></s:f>")
	mustAccept(t, schema, data)
	got, err := Decode[pair](schema, strict.New(), data)
	if err != nil {
		t.Fatal(err)
	}
	if want := (pair{E: ttNum{A: "1", N: 2}, F: ttBase{A: "3"}}); !reflect.DeepEqual(*got, want) {
		t.Errorf("Decode = %#v, want %#v", *got, want)
	}
}
