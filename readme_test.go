package goxsd8

import (
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/codegen"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
)

// TestReadmeFixtures holds each file of testdata/readme that README.md shows
// to the text README shows for it: order.xsd and order.json byte for byte as
// a fence's body, and the one-line twice.json, its newline dropped, as an
// inline code span. README's examples run against the committed files, so a
// fence edited without its file, or a file without its fence, would show the
// reader one document and run another.
//
// It holds README's code-generation snippet to readmeCodegen's body, which
// compiles with this file, and the xsd_gen.go README shows to the file that
// running readmeCodegen writes for order.xsd, both byte for byte as a fence's
// body.
func TestReadmeFixtures(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		file string
		want func(body string) string
	}{
		{"order.xsd", func(body string) string { return "```xml\n" + body + "```\n" }},
		{"order.json", func(body string) string { return "```json\n" + body + "```\n" }},
		{"twice.json", func(body string) string { return "`" + strings.TrimSuffix(body, "\n") + "`" }},
	} {
		body, err := os.ReadFile(filepath.Join("testdata", "readme", c.file))
		if err != nil {
			t.Errorf("%s: %v", c.file, err)
			continue
		}
		if want := c.want(string(body)); !strings.Contains(string(readme), want) {
			t.Errorf("README.md does not show testdata/readme/%s as\n%s", c.file, want)
		}
	}

	snippet, err := funcBody("readme_test.go", "readmeCodegen")
	if err != nil {
		t.Fatal(err)
	}
	if want := "```go\n" + snippet + "```\n"; !strings.Contains(string(readme), want) {
		t.Errorf("README.md does not show readmeCodegen's body as\n%s", want)
	}

	// The program parses order.xsd with testdata/readme as its working
	// directory, so its Loc reads order.xsd:3:3.
	schema, err := parser.Parse("order.xsd", parser.WithResolver(loader.Dir(filepath.Join("testdata", "readme"))))
	if err != nil {
		t.Fatal(err)
	}
	// The program runs in its module's root, as README states.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(root)
	readmeCodegen(schema)
	gen, err := os.ReadFile(filepath.Join("gen", "order", "xsd_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "```go\n" + string(gen) + "```\n"; !strings.Contains(string(readme), want) {
		t.Errorf("README.md does not show the xsd_gen.go Generate writes for testdata/readme/order.xsd as\n%s", want)
	}
}

// readmeCodegen is README's code-generation snippet: README shows its body
// byte for byte, each line less one leading tab. schema stands for the
// program's variable of that name.
func readmeCodegen(schema *xsd.Schema) {
	// Writes package order into gen/order.
	if err := codegen.Generate([]codegen.Target{{Schema: schema, Dir: "gen/order", Package: "order"}}); err != nil {
		// Every non-nil error is an errors.Join: print each of its elements, a
		// decline by its Loc and Construct, any other failure as it reads.
		for _, e := range err.(interface{ Unwrap() []error }).Unwrap() {
			var de *codegen.DeclineError
			if errors.As(e, &de) {
				fmt.Printf("%s: codegen declined %s\n", de.Loc(), de.Construct())
				continue
			}
			fmt.Println(e)
		}
		os.Exit(1)
	}
}

// funcBody returns the statements of the function name declares in file, the
// lines between its braces with one leading tab dropped from each.
func funcBody(file, name string) (string, error) {
	src, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, file, src, goparser.ParseComments)
	if err != nil {
		return "", err
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Name.Name != name {
			continue
		}
		body := string(src[fset.Position(fn.Body.Lbrace).Offset+2 : fset.Position(fn.Body.Rbrace).Offset])
		return strings.ReplaceAll("\n"+body, "\n\t", "\n")[1:], nil
	}
	return "", fmt.Errorf("%s declares no func %s", file, name)
}
