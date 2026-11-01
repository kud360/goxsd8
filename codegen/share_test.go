package codegen

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/loader"
	xsdparser "github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// schemaDoc is one schema document of a sharing fixture: its location and its
// content inside <xs:schema>, targetNamespace being the root's attribute or "".
type schemaDoc struct {
	location, targetNamespace, body string
}

// parseDocs parses the document at root among docs, each resolved by its
// location from docs alone, so that every Loc is that relative location.
func parseDocs(t *testing.T, root string, docs []schemaDoc) *xsd.Schema {
	t.Helper()
	resolve := loader.ResolverFunc(func(_, location string) (io.ReadCloser, string, error) {
		i := slices.IndexFunc(docs, func(d schemaDoc) bool { return d.location == location })
		if i < 0 {
			return nil, "", fmt.Errorf("no document %q", location)
		}
		tns := ""
		if docs[i].targetNamespace != "" {
			tns = ` targetNamespace="` + docs[i].targetNamespace + `" xmlns="` + docs[i].targetNamespace + `"`
		}
		doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + tns + ">\n" + docs[i].body + "\n</xs:schema>\n"
		return io.NopCloser(strings.NewReader(doc)), location, nil
	})
	schema, err := xsdparser.Parse(root, xsdparser.WithResolver(resolve))
	if err != nil {
		t.Fatalf("Parse(%s): %v", root, err)
	}
	return schema
}

// generateAll runs Generate on targets, each Dir under the module root a
// moduleTempDir made, and returns each file it wrote, in targets' order.
func generateAll(t *testing.T, targets []Target) []string {
	t.Helper()
	if err := Generate(targets); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	srcs := make([]string, len(targets))
	for i, target := range targets {
		srcs[i] = read(t, filepath.Join(target.Dir, fileName))
	}
	return srcs
}

// checkTogether type-checks srcs in order as the packages of paths, each
// import of an earlier one resolving to that checked package and every other
// import through the default importer, and returns the checked packages.
func checkTogether(t *testing.T, paths, srcs []string) []*types.Package {
	t.Helper()
	pkgs := make([]*types.Package, len(srcs))
	fallback := importer.Default()
	imp := importerFunc(func(path string) (*types.Package, error) {
		if i := slices.Index(paths, path); i >= 0 && pkgs[i] != nil {
			return pkgs[i], nil
		}
		return fallback.Import(path)
	})
	for i, src := range srcs {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, fileName, src, parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v\n%s", paths[i], err, src)
		}
		pkg, err := (&types.Config{Importer: imp}).Check(paths[i], fset, []*ast.File{file}, nil)
		if err != nil {
			t.Fatalf("type-checking %s: %v\n%s", paths[i], err, src)
		}
		pkgs[i] = pkg
	}
	return pkgs
}

// importerFunc is a types.Importer of one function.
type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// assertLines fails unless src holds each of want as a whole line run, and
// none of absent.
func assertLines(t *testing.T, label, src string, want, absent []string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(src, "\n"+w+"\n") {
			t.Errorf("%s lacks %q:\n%s", label, w, src)
		}
	}
	for _, a := range absent {
		if strings.Contains(src, a) {
			t.Errorf("%s holds %q:\n%s", label, a, src)
		}
	}
}

// sharedItem is a document whose one element's type names a math/big type
// and a datatype struct, which a.xsd and b.xsd below both include.
var sharedItem = schemaDoc{location: "shared.xsd", body: `<xs:element name="item"><xs:complexType><xs:sequence>` +
	`<xs:element name="qty" type="xs:integer"/><xs:element name="when" type="xs:date"/>` +
	`</xs:sequence></xs:complexType></xs:element>`}

// TestGenerateImportsAgreeingDocument pins Generate's import rule for two
// targets whose emitted declarations of shared.xsd agree: the first defines
// Item and the per-package Date its field reaches; the second imports the
// first's package by the import path its go.mod derives, aliases Item under
// the header Item carries, defines its own Note, and writes neither Date,
// which only the aliased Item reached, nor the math/big import only Item's
// field needed. Both packages type-check together, and the alias is the
// owner's type. Without the import rule the second file defines Item and
// Date itself and the alias assertions fail.
func TestGenerateImportsAgreeingDocument(t *testing.T) {
	docs := []schemaDoc{
		sharedItem,
		{location: "a.xsd", body: `<xs:include schemaLocation="shared.xsd"/>` +
			`<xs:element name="order"><xs:complexType><xs:sequence><xs:element ref="item"/></xs:sequence></xs:complexType></xs:element>`},
		{location: "b.xsd", body: `<xs:include schemaLocation="shared.xsd"/><xs:element name="note" type="xs:string"/>`},
	}
	root := moduleTempDir(t)
	srcs := generateAll(t, []Target{
		{Schema: parseDocs(t, "a.xsd", docs), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "b"), Package: "b"},
	})
	assertLines(t, "a's file", srcs[0], []string{"type Item struct {", "type Date struct {", `import "math/big"`}, []string{"import a", " = "})
	assertLines(t, "b's file", srcs[1], []string{
		`import a "example.com/m/a"`,
		"// Item is generated from element item declared at shared.xsd:2:1.",
		"type Item = a.Item",
		"type Note string",
	}, []string{"type Item struct", "Date", "math/big"})
	pkgs := checkTogether(t, []string{"example.com/m/a", "example.com/m/b"}, srcs)
	owned, aliased := pkgs[0].Scope().Lookup("Item"), pkgs[1].Scope().Lookup("Item")
	if owned == nil || aliased == nil || !types.Identical(owned.Type(), aliased.Type()) {
		t.Errorf("b.Item = %v, a.Item = %v, want one type", aliased, owned)
	}
}

// TestGenerateSharesAcrossDirDepths pins that agreement is measured with each
// header's Loc as the Loc carries it, independent of Dir: shared.xsd is read
// by an absolute location, so a's and b's headers, rendered against Dirs of
// different depths, climb differently, and b still imports Item, under the
// header rendered against its own Dir. Comparing the rendered headers instead
// makes b define Item and fails the test.
func TestGenerateSharesAcrossDirDepths(t *testing.T) {
	root := moduleTempDir(t)
	schemas := filepath.Join(root, "schemas")
	if err := os.Mkdir(schemas, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(schemas, "shared.xsd"), `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+sharedItem.body+"</xs:schema>\n")
	write(t, filepath.Join(schemas, "a.xsd"), `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:include schemaLocation="shared.xsd"/></xs:schema>`+"\n")
	parse := func(name string) *xsd.Schema {
		schema, err := xsdparser.Parse(filepath.Join(schemas, name), xsdparser.WithResolver(loader.Dir(string(filepath.Separator))))
		if err != nil {
			t.Fatalf("Parse(%s): %v", name, err)
		}
		return schema
	}
	srcs := generateAll(t, []Target{
		{Schema: parse("a.xsd"), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parse("shared.xsd"), Dir: filepath.Join(root, "deep", "er", "b"), Package: "b"},
	})
	assertLines(t, "a's file", srcs[0], []string{"// Item is generated from element item declared at ../schemas/shared.xsd:1:56."}, nil)
	assertLines(t, "b's file", srcs[1], []string{
		"// Item is generated from element item declared at ../../../schemas/shared.xsd:1:56.",
		"type Item = a.Item",
	}, nil)
}

// TestGenerateDefinesDifferingDocument pins Generate's rule for a shared
// document whose emitted declarations differ (the GROUNDING's probe 2):
// a.xsd adds a member to the substitution group of shared.xsd's h, so a's Po
// holds the group's interface while b's holds H, and each target defines
// shared.xsd's types itself, importing nothing.
func TestGenerateDefinesDifferingDocument(t *testing.T) {
	docs := []schemaDoc{
		{location: "shared.xsd", body: `<xs:element name="h" type="xs:string"/>` +
			`<xs:element name="po"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>`},
		{location: "a.xsd", body: `<xs:include schemaLocation="shared.xsd"/><xs:element name="m1" substitutionGroup="h" type="xs:string"/>`},
		{location: "b.xsd", body: `<xs:include schemaLocation="shared.xsd"/>`},
	}
	root := moduleTempDir(t)
	srcs := generateAll(t, []Target{
		{Schema: parseDocs(t, "a.xsd", docs), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "b"), Package: "b"},
	})
	assertLines(t, "a's file", srcs[0], []string{"\tH HSubstitutionGroup"}, []string{"import"})
	assertLines(t, "b's file", srcs[1], []string{"type Po struct {", "\tH H", "type H string"}, []string{"import", " = "})
	checkTogether(t, []string{"example.com/m/a", "example.com/m/b"}, srcs)
}

// TestGenerateChameleonNotShared pins that one URI composed into different
// components is never shared (the GROUNDING's probe 3): common.xsd, with no
// targetNamespace, is included under urn:a and under urn:b, so its item
// renders {urn:a}item and {urn:b}item, and each target defines Item.
func TestGenerateChameleonNotShared(t *testing.T) {
	docs := []schemaDoc{
		{location: "common.xsd", body: `<xs:element name="item" type="xs:string"/>`},
		{location: "a.xsd", targetNamespace: "urn:a", body: `<xs:include schemaLocation="common.xsd"/>`},
		{location: "b.xsd", targetNamespace: "urn:b", body: `<xs:include schemaLocation="common.xsd"/>`},
	}
	root := moduleTempDir(t)
	srcs := generateAll(t, []Target{
		{Schema: parseDocs(t, "a.xsd", docs), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "b"), Package: "b"},
	})
	assertLines(t, "a's file", srcs[0], []string{"// Item is generated from element {urn:a}item declared at common.xsd:2:1.", "type Item string"}, nil)
	assertLines(t, "b's file", srcs[1], []string{"// Item is generated from element {urn:b}item declared at common.xsd:2:1.", "type Item string"}, []string{"import"})
}

// TestGenerateClosureCondition pins the closure condition: d.xsd's
// declarations, and e.xsd's E1 that its Box reaches, are the same text in
// both targets, but e.xsd as a whole differs, since a.xsd adds a member to
// the substitution group of e.xsd's h, so b defines e.xsd's types itself, and
// importing d.xsd would give b a Box whose field is a's E1, not b's: b
// defines Box too and imports nothing. Without the closure condition b
// aliases Box and the assertions on b fail.
func TestGenerateClosureCondition(t *testing.T) {
	docs := []schemaDoc{
		{location: "d.xsd", body: `<xs:include schemaLocation="e.xsd"/>` +
			`<xs:element name="box"><xs:complexType><xs:sequence><xs:element ref="e1"/></xs:sequence></xs:complexType></xs:element>`},
		{location: "e.xsd", body: `<xs:element name="e1" type="xs:string"/><xs:element name="h" type="xs:string"/>` +
			`<xs:element name="e2"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>`},
		{location: "a.xsd", body: `<xs:include schemaLocation="d.xsd"/><xs:element name="m1" substitutionGroup="h" type="xs:string"/>`},
		{location: "b.xsd", body: `<xs:include schemaLocation="d.xsd"/>`},
	}
	root := moduleTempDir(t)
	srcs := generateAll(t, []Target{
		{Schema: parseDocs(t, "a.xsd", docs), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "b"), Package: "b"},
	})
	assertLines(t, "a's file", srcs[0], []string{"type Box struct {", "\tH HSubstitutionGroup"}, nil)
	assertLines(t, "b's file", srcs[1], []string{"type Box struct {", "type E1 string", "\tH H"}, []string{"import", " = "})
	checkTogether(t, []string{"example.com/m/a", "example.com/m/b"}, srcs)
}

// TestGenerateSealedClosure pins that a sealed interface's closure holds every
// declaration carrying its marker method: shared.xsd's declarations are the
// same text in both targets, but a.xsd and b.xsd each add a member m1 to the
// substitution group of shared.xsd's h, so the interface HSubstitutionGroup
// reaches a branch HSubstitutionGroupM1 attributed to a.xsd in one target and
// b.xsd in the other, and b defines the interface itself, as its own branch,
// whose marker method is unexported, could implement no other package's.
// Without the marker relation in the closure b aliases the interface and the
// assertions on b fail.
func TestGenerateSealedClosure(t *testing.T) {
	member := `<xs:include schemaLocation="shared.xsd"/><xs:element name="m1" substitutionGroup="h" type="xs:string"/>`
	docs := []schemaDoc{
		{location: "shared.xsd", body: `<xs:element name="h" type="xs:string"/>` +
			`<xs:element name="po"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>`},
		{location: "a.xsd", body: member},
		{location: "b.xsd", body: member},
	}
	root := moduleTempDir(t)
	srcs := generateAll(t, []Target{
		{Schema: parseDocs(t, "a.xsd", docs), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "b"), Package: "b"},
	})
	assertLines(t, "b's file", srcs[1], []string{"type HSubstitutionGroup interface {", "func (HSubstitutionGroupM1) isHSubstitutionGroup() {}"}, []string{"import", " = "})
}

// TestGenerateUnattributedClosure pins that the closure compared holds the
// declarations that belong to no document: shared.xsd's V and U are the same
// text in both targets, but b's own element uString claims UString first, so
// the variant of U's builtin member xs:string, which renders no Loc, is
// UString in a and UString2 in b, and b defines U itself. Comparing D's own
// declarations alone makes b alias U, whose variants b's file still declares
// as arms of no interface of its own, and fails the test.
func TestGenerateUnattributedClosure(t *testing.T) {
	docs := []schemaDoc{
		{location: "shared.xsd", body: `<xs:simpleType name="u"><xs:union memberTypes="xs:string xs:int"/></xs:simpleType><xs:element name="v" type="u"/>`},
		{location: "a.xsd", body: `<xs:include schemaLocation="shared.xsd"/>`},
		{location: "b.xsd", body: `<xs:include schemaLocation="shared.xsd"/><xs:element name="uString" type="xs:boolean"/>`},
	}
	root := moduleTempDir(t)
	srcs := generateAll(t, []Target{
		{Schema: parseDocs(t, "a.xsd", docs), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "b"), Package: "b"},
	})
	assertLines(t, "a's file", srcs[0], []string{"type UString string"}, nil)
	assertLines(t, "b's file", srcs[1], []string{"type U interface {", "type UString2 string", "type V U"}, []string{"import", " = "})
}

// TestGenerateNillableRoot pins that the type of a nillable top-level
// declaration reaches Nillable and its arms though its code names none of
// them: a and b share shared.xsd, b also declares the nillable note, and c
// compiles b's schema. b defines Note, NoteAttributes and the Nillable family
// that Note's nilled value needs; c aliases Note and NoteAttributes from b and
// declares no family, Note's nilled value being b's Nillable; all three
// packages type-check together.
func TestGenerateNillableRoot(t *testing.T) {
	docs := []schemaDoc{
		sharedItem,
		{location: "a.xsd", body: `<xs:include schemaLocation="shared.xsd"/>`},
		{location: "b.xsd", body: `<xs:include schemaLocation="shared.xsd"/>` +
			`<xs:element name="note" nillable="true"><xs:complexType><xs:attribute name="lang" type="xs:string"/></xs:complexType></xs:element>`},
	}
	root := moduleTempDir(t)
	srcs := generateAll(t, []Target{
		{Schema: parseDocs(t, "a.xsd", docs), Dir: filepath.Join(root, "a"), Package: "a"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "b"), Package: "b"},
		{Schema: parseDocs(t, "b.xsd", docs), Dir: filepath.Join(root, "c"), Package: "c"},
	})
	assertLines(t, "a's file", srcs[0], nil, []string{"Nillable", "Nilled", "Content"})
	assertLines(t, "b's file", srcs[1], []string{
		"type Item = a.Item",
		"type Note struct {",
		"type Nillable[T, A any] interface {",
		"type Nilled[T, A any] struct {",
		"type Content[T, A any] struct {",
	}, nil)
	assertLines(t, "c's file", srcs[2], []string{
		"type Item = a.Item",
		"type Note = b.Note",
		"type NoteAttributes = b.NoteAttributes",
	}, []string{"Nillable", "Nilled", "Content"})
	checkTogether(t, []string{"example.com/m/a", "example.com/m/b", "example.com/m/c"}, srcs)
}

// TestGenerateImportNames pins the import name rule: the third target imports
// p.xsd from the first and q.xsd from the second, both of Package Item, and
// declares its own type Item, so the first owner takes Item2 and the second,
// that name taken by the first, Item3; owners of Package big and len, a
// predeclared identifier, take big2 and len2.
func TestGenerateImportNames(t *testing.T) {
	docs := []schemaDoc{
		{location: "p.xsd", body: `<xs:element name="pe" type="xs:string"/>`},
		{location: "q.xsd", body: `<xs:element name="qe" type="xs:boolean"/>`},
		{location: "z.xsd", body: `<xs:include schemaLocation="p.xsd"/><xs:include schemaLocation="q.xsd"/><xs:element name="item" type="xs:string"/>`},
	}
	for _, tt := range []struct {
		first, second, wantFirst, wantSecond string
	}{
		{"Item", "Item", "Item2", "Item3"},
		{"big", "len", "big2", "len2"},
	} {
		t.Run(tt.first+" "+tt.second, func(t *testing.T) {
			root := moduleTempDir(t)
			srcs := generateAll(t, []Target{
				{Schema: parseDocs(t, "p.xsd", docs), Dir: filepath.Join(root, "one", "x"), Package: tt.first},
				{Schema: parseDocs(t, "q.xsd", docs), Dir: filepath.Join(root, "two", "x"), Package: tt.second},
				{Schema: parseDocs(t, "z.xsd", docs), Dir: filepath.Join(root, "z"), Package: "z"},
			})
			assertLines(t, "z's file", srcs[2], []string{
				"\t" + tt.wantFirst + ` "example.com/m/one/x"`,
				"\t" + tt.wantSecond + ` "example.com/m/two/x"`,
				"type Pe = " + tt.wantFirst + ".Pe",
				"type Qe = " + tt.wantSecond + ".Qe",
				"type Item string",
			}, nil)
			checkTogether(t, []string{"example.com/m/one/x", "example.com/m/two/x", "example.com/m/z"}, srcs)
		})
	}
}

// moduleDocs are the sharing fixtures of TestGenerateSharesWithinModule: a.xsd
// declares A, and b.xsd and c.xsd each import common.xsd, which declares
// Common.
var moduleDocs = []schemaDoc{
	{location: "a.xsd", body: `<xs:element name="a" type="xs:string"/>`},
	{location: "common.xsd", targetNamespace: "urn:c", body: `<xs:element name="common" type="xs:string"/>`},
	{location: "b.xsd", body: `<xs:import namespace="urn:c" schemaLocation="common.xsd"/><xs:element name="b" type="xs:string"/>`},
	{location: "c.xsd", body: `<xs:import namespace="urn:c" schemaLocation="common.xsd"/><xs:element name="c" type="xs:string"/>`},
}

// subModule makes root's directory sub hold a go.mod of its own declaring
// module example.com/sub.
func subModule(t *testing.T, root string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "sub", "go.mod"), "module example.com/sub\n")
}

// TestGenerateSharesWithinModule pins Generate's module condition on sharing,
// in a module example.com/m whose directory sub holds a go.mod of its own
// declaring example.com/sub. A later target whose nearest go.mod is not its
// agreeing earlier target's defines the shared document itself and imports
// nothing, with the earlier target outside sub or inside it, for one schema
// generated twice (A) and for two schemas importing one document (Common).
// Among three targets of one schema, o1 outside sub and o2 and t inside it, t
// imports A from o2, the earliest agreeing target under its own go.mod, and
// not from o1.
func TestGenerateSharesWithinModule(t *testing.T) {
	for _, tt := range []struct {
		name, first, second, decl string
		ownerInSub                bool
	}{
		{"one schema, owner outside sub", "a.xsd", "a.xsd", "type A string", false},
		{"one schema, owner in sub", "a.xsd", "a.xsd", "type A string", true},
		{"shared import, owner outside sub", "b.xsd", "c.xsd", "type Common string", false},
		{"shared import, owner in sub", "b.xsd", "c.xsd", "type Common string", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := moduleTempDir(t)
			subModule(t, root)
			ownerDir, laterDir, ownerPath := filepath.Join(root, "o", "x"), filepath.Join(root, "sub", "y"), "example.com/m/o/x"
			if tt.ownerInSub {
				ownerDir, laterDir, ownerPath = filepath.Join(root, "sub", "o", "x"), filepath.Join(root, "y"), "example.com/sub/o/x"
			}
			srcs := generateAll(t, []Target{
				{Schema: parseDocs(t, tt.first, moduleDocs), Dir: ownerDir, Package: "x"},
				{Schema: parseDocs(t, tt.second, moduleDocs), Dir: laterDir, Package: "y"},
			})
			assertLines(t, "the later target's file", srcs[1], []string{tt.decl}, []string{ownerPath, "import", " = "})
		})
	}
	t.Run("owner chosen under the target's go.mod", func(t *testing.T) {
		root := moduleTempDir(t)
		subModule(t, root)
		srcs := generateAll(t, []Target{
			{Schema: parseDocs(t, "a.xsd", moduleDocs), Dir: filepath.Join(root, "o1"), Package: "o1"},
			{Schema: parseDocs(t, "a.xsd", moduleDocs), Dir: filepath.Join(root, "sub", "o2"), Package: "o2"},
			{Schema: parseDocs(t, "a.xsd", moduleDocs), Dir: filepath.Join(root, "sub", "t"), Package: "t"},
		})
		assertLines(t, "o2's file", srcs[1], []string{"type A string"}, []string{"import"})
		assertLines(t, "t's file", srcs[2], []string{`import o2 "example.com/sub/o2"`, "type A = o2.A"}, []string{"example.com/m/o1", "type A string"})
		checkTogether(t, []string{"example.com/m/o1", "example.com/sub/o2", "example.com/sub/t"}, srcs)
	})
}
