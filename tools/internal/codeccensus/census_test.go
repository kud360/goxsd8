package codeccensus

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/codegen"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/tools/internal/codeccensus/assess"
	"github.com/kud360/goxsd8/xsd"
)

// writeTree writes files, slash paths relative to a fresh directory, and
// returns that directory.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, body := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return root
}

// TestRankedOrdersByCountThenString pins Ranked's order on three tallies
// whose first-seen order, count order with a first-seen tie-break and count
// order with the String tie-break the tool owes are three different orders.
func TestRankedOrdersByCountThenString(t *testing.T) {
	x := assess.Tally{Construct: "x", Cases: []string{"1"}}
	m := assess.Tally{Construct: "m", Cases: []string{"1", "2"}}
	b := assess.Tally{Construct: "b", Cases: []string{"1", "2"}}
	tallies := []assess.Tally{x, m, b}
	want := []assess.Tally{b, m, x}
	if got := Ranked(tallies); !reflect.DeepEqual(got, want) {
		t.Errorf("Ranked = %v, want %v", got, want)
	}
}

// ownTypeSchema declares, beside a, an element A whose type's name collides
// with a's and takes A2; x, whose choice's branches XA and XH, each over an
// <element ref>, record a and h in their headers; h and its substitution
// group member m, whose interface's branches HSubstitutionGroupH and
// HSubstitutionGroupM record h and m; tt, a sealed interface over its {type
// table}; s, one over the members of its abstract {type definition}; and the
// abstract abs, which gets no type.
const ownTypeSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="a"><xs:complexType><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:complexType></xs:element>
  <xs:element name="A" type="xs:string"/>
  <xs:element name="x"><xs:complexType><xs:choice><xs:element ref="a"/><xs:element ref="h"/></xs:choice></xs:complexType></xs:element>
  <xs:element name="h" type="xs:string"/>
  <xs:element name="m" substitutionGroup="h" type="xs:string"/>
  <xs:element name="tt" type="xs:anyType"><xs:alternative test="@k" type="xs:string"/></xs:element>
  <xs:complexType name="shape" abstract="true"/>
  <xs:complexType name="circle"><xs:complexContent><xs:extension base="shape"/></xs:complexContent></xs:complexType>
  <xs:complexType name="square"><xs:complexContent><xs:extension base="shape"/></xs:complexContent></xs:complexType>
  <xs:element name="s" type="shape"/>
  <xs:element name="abs" abstract="true" type="xs:string"/>
</xs:schema>`

// TestOwnTypeFindsEachTopLevelElementsType generates ownTypeSchema and pins
// the type resolveCase finds for each top-level element. Without namedFrom's
// test, a, h and m each match a branch beside their own type and resolve to
// none; with ownProvenances' plain entry alone, tt and s resolve to none.
func TestOwnTypeFindsEachTopLevelElementsType(t *testing.T) {
	root := writeTree(t, map[string]string{"s.xsd": ownTypeSchema, "out/go.mod": "module census\n"})
	schema, err := parser.Parse("s.xsd", parser.WithResolver(loader.Dir(root)))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	dir := filepath.Join(root, "out", "p")
	if errs := codegen.Generate([]codegen.Target{{Schema: schema, Dir: dir, Package: "p"}}); errs != nil {
		t.Fatalf("Generate: %v", errs)
	}
	gen, err := parseGenerated(filepath.Join(dir, "xsd_gen.go"))
	if err != nil {
		t.Fatalf("parseGenerated: %v", err)
	}
	file := gen.file
	types := generatedTypes(file)
	cases := []struct {
		local, typ string
		decl, ok   bool
	}{
		{"a", "A", true, true},
		{"A", "A2", true, true},
		{"x", "X", true, true},
		{"h", "H", true, true},
		{"m", "M", true, true},
		{"tt", "Tt", true, true},
		{"s", "S", true, true},
		{"abs", "", false, false},
		{"undeclared", "", false, false},
	}
	for _, tc := range cases {
		typ, decl, ok := resolveCase(schema, types, xsd.QName{Local: tc.local}, dir)
		if typ != tc.typ || decl != tc.decl || ok != tc.ok {
			t.Errorf("resolveCase(%s) = %q, %v, %v, want %q, %v, %v; types %v", tc.local, typ, decl, ok, tc.typ, tc.decl, tc.ok, types)
		}
	}
}

// TestOwnTypeRendersAnAbsoluteLoc parses the schema under its absolute path,
// so that its declarations' Loc URIs are absolute: resolveCase finds a's type
// only through headerLoc's branch for one, and fails with that branch
// removed.
func TestOwnTypeRendersAnAbsoluteLoc(t *testing.T) {
	root := writeTree(t, map[string]string{"s.xsd": ownTypeSchema, "out/go.mod": "module census\n"})
	schema, err := parser.Parse(filepath.Join(root, "s.xsd"), parser.WithResolver(loader.Dir(string(filepath.Separator))))
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	decl, ok := schema.Element(xsd.QName{Local: "a"})
	if !ok || !filepath.IsAbs(decl.Loc().URI) {
		t.Fatalf("element a's Loc %v is not absolute", decl.Loc())
	}
	dir := filepath.Join(root, "out", "p")
	if errs := codegen.Generate([]codegen.Target{{Schema: schema, Dir: dir, Package: "p"}}); errs != nil {
		t.Fatalf("Generate: %v", errs)
	}
	gen, err := parseGenerated(filepath.Join(dir, "xsd_gen.go"))
	if err != nil {
		t.Fatalf("parseGenerated: %v", err)
	}
	file := gen.file
	types := generatedTypes(file)
	if typ, _, ok := resolveCase(schema, types, xsd.QName{Local: "a"}, dir); typ != "A" || !ok {
		t.Errorf("resolveCase(a) = %q, %v, want \"A\", true; types %v", typ, ok, types)
	}
}

// TestDriverSourceBindsBranches pins the census's branch-set supply: a
// generated file declaring three interfaces, one with two marker types and a
// header of another provenance, one with a union's non-struct variant, and
// PrecisionDecimal's Value interface, known by its header, with its four
// arms, beside a method that is no marker, a pointer-receiver method of a
// marker's name and a generic interface, yields sealedInterfaces' list, and a
// driver source declaring one WithBranches per other non-generic interface
// and one WithPrecisionDecimalArms of the arms in file order for the Value
// interface, in file order, bound to each codec of that package alone, and a
// main calling the entry driverSource is given with those codecs.
func TestDriverSourceBindsBranches(t *testing.T) {
	const src = `package p0

// R is generated from element r declared at s.xsd:1:1.
type R interface {
	isR()
}

type RA struct{ A string }

func (RA) isR() {}

type RB struct{ B string }

func (RB) isR() {}

func (RB) String() string { return "" }

type U interface {
	isU()
}

type UInt int32

func (UInt) isU() {}

func (*RA) isU() {}

type Nillable[T, A any] interface {
	isNillable(T, A)
}

// D is generated from datatype {http://www.w3.org/2001/XMLSchema}precisionDecimal.
type D interface {
	isD()
}

type DN struct{ Coefficient, Scale int }

func (DN) isD() {}

type DP struct{}

func (DP) isD() {}

type DM struct{}

func (DM) isD() {}

type DQ struct{}

func (DQ) isD() {}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "xsd_gen.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	gen, err := parseGenerated(path)
	if err != nil {
		t.Fatal(err)
	}
	sealed := sealedInterfaces(gen.file)
	want := []sealedDecl{{name: "R", branches: []string{"RA", "RB"}}, {name: "U", branches: []string{"UInt"}}, {name: "D", branches: []string{"DN", "DP", "DM", "DQ"}, decimal: true}}
	if !reflect.DeepEqual(sealed, want) {
		t.Errorf("sealedInterfaces = %+v, want %+v", sealed, want)
	}
	got := string(driverSource(batch{packages: []genPackage{{name: "p0", sealed: sealed}, {name: "p1"}}, codecs: []string{"p0.R", "p1.S", "p0.X"}}, "DriveEncode"))
	for _, line := range []string{
		"\t\"github.com/kud360/goxsd8/codec\"\n",
		"var p0Branches = []codec.Option{\n\tcodec.WithBranches[p0.R](*new(p0.RA), *new(p0.RB)),\n\tcodec.WithBranches[p0.U](*new(p0.UInt)),\n" +
			"\tcodec.WithPrecisionDecimalArms[p0.D](*new(p0.DN), *new(p0.DP), *new(p0.DM), *new(p0.DQ)),\n}\n",
		"\tassess.CodecFor[p0.R](p0Branches...),\n\tassess.CodecFor[p1.S](),\n\tassess.CodecFor[p0.X](p0Branches...),\n",
		"\tif err := assess.DriveEncode(os.Stdin, os.Stdout, codecs); err != nil {\n",
	} {
		if !strings.Contains(got, line) {
			t.Errorf("driver source\n%s\nlacks\n%s", got, line)
		}
	}
	if strings.Contains(got, "p1Branches") {
		t.Errorf("driver source\n%s\ndeclares branches for p1, which declares no interface", got)
	}
}
