package codegen

import (
	"bytes"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"testing"

	xsdparser "github.com/kud360/goxsd8/parser"
)

var update = flag.Bool("update", false, "rewrite codegen/testdata/*.golden from Generate's output")

// TestGenerateGolden pins Generate's output for each fixture. The fixtures are
// parsed by a relative location, so the Loc in every header is portable.
func TestGenerateGolden(t *testing.T) {
	for _, name := range []string{"flat", "nested", "collision", "named", "expanded", "symbolspaces", "choice", "attributes", "integers", "identity", "empty"} {
		t.Run(name, func(t *testing.T) {
			schema, err := xsdparser.Parse("testdata/" + name + ".xsd")
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			first := generateOne(t, Target{Schema: schema, Dir: filepath.Join(t.TempDir(), "a"), Package: name})
			second := generateOne(t, Target{Schema: schema, Dir: filepath.Join(t.TempDir(), "b"), Package: name})
			if !bytes.Equal(first, second) {
				t.Fatalf("two runs differ:\n%s\n---\n%s", first, second)
			}
			typeCheck(t, first)
			golden := filepath.Join("testdata", name+".golden")
			if *update {
				if err := os.WriteFile(golden, first, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first, want) {
				t.Errorf("output differs from %s (rerun with -update to accept):\n%s", golden, first)
			}
		})
	}
}

// generateOne runs Generate on the one target and returns the file it wrote.
func generateOne(t *testing.T, target Target) []byte {
	t.Helper()
	if err := Generate([]Target{target}); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	src, err := os.ReadFile(filepath.Join(target.Dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	return src
}

// typeCheck fails unless src is a package that compiles. The emitted package
// imports nothing, so no importer is configured: an import would fail here.
func typeCheck(t *testing.T, src []byte) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, fileName, src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing the emitted source: %v", err)
	}
	if _, err := new(types.Config).Check(file.Name.Name, fset, []*ast.File{file}, nil); err != nil {
		t.Fatalf("type-checking the emitted source: %v", err)
	}
}
