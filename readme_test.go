package goxsd8

import (
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/codec"
	"github.com/kud360/goxsd8/codec/testdata/readme/order"
	"github.com/kud360/goxsd8/codegen"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestReadmeFixtures holds each file of testdata/readme that README.md's CLI
// section shows to the text README shows for it (TestReadmeSharedDocument
// holds the files of the two-target gen example): qty.xsd, qty.xml and
// zero.xml byte for byte as the body of a quoted heredoc (heredoc), which the
// shell writes unaltered; order.xsd and order.json byte for byte as a fence's
// body; and the one-line twice.json, its newline dropped, as an inline code
// span. README's examples run against the committed files, and its first
// example against the files its heredocs write, so a fence edited without its
// file, or a file without its fence, would show the reader one document and
// run another.
//
// It holds README's code-generation snippet to readmeCodegen's body, which
// compiles with this file, and the xsd_gen.go README shows to the file that
// running readmeCodegen writes for order.xsd, both byte for byte as a fence's
// body. It holds README's decode and encode snippets to readmeDecode's and
// readmeAppend's bodies the same way, runs readmeDecode on order.xml and
// readmeAppend on the Order it decodes, and holds what each prints to the
// output its snippet's last comment states.
func TestReadmeFixtures(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		file string
		want func(body string) string
	}{
		{"qty.xsd", func(body string) string { return heredoc("qty.xsd", body) }},
		{"qty.xml", func(body string) string { return heredoc("qty.xml", body) }},
		{"zero.xml", func(body string) string { return heredoc("zero.xml", body) }},
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
	for _, name := range []string{"readmeDecode", "readmeAppend"} {
		snippet, err = funcBody("readme_test.go", name)
		if err != nil {
			t.Fatal(err)
		}
		if want := "```go\n" + snippet + "```\n"; !strings.Contains(string(readme), want) {
			t.Errorf("README.md does not show %s's body as\n%s", name, want)
		}
	}
	// Read before the test leaves the checkout's root.
	decodePrints, appendPrints := lastComment(t, "readmeDecode"), lastComment(t, "readmeAppend")

	// The program parses order.xsd with testdata/readme as its working
	// directory, so its Loc reads order.xsd:3:3.
	schema, err := parser.Parse("order.xsd", parser.WithResolver(loader.Dir(filepath.Join("testdata", "readme"))))
	if err != nil {
		t.Fatal(err)
	}
	// The program runs in its module's root, one of the directories README
	// states it may run in.
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/app\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
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

	// The program reads order.xml from its working directory; package order
	// is the golden copy of the file readmeCodegen writes, which
	// codec.TestDecodeReadmeOrder holds to codegen's output.
	t.Chdir(filepath.Join(wd, "testdata", "readme"))
	if got := stdout(t, func() { readmeDecode(schema, strict.New()) }); got != decodePrints {
		t.Errorf("readmeDecode prints %q, its last comment states %q", got, decodePrints)
	}
	data, err := os.ReadFile("order.xml")
	if err != nil {
		t.Fatal(err)
	}
	o, err := codec.Decode[order.Order](schema, strict.New(), data, codec.WithURI("order.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := stdout(t, func() { readmeAppend(schema, strict.New(), o) }); got != appendPrints {
		t.Errorf("readmeAppend prints %q, its last comment states %q", got, appendPrints)
	}
}

// heredoc returns the shell lines README's first CLI example writes file
// with: `cat > <file> <<'EOF'`, body, then `EOF`. The delimiter is quoted, so
// the shell expands nothing in body and writes it byte for byte.
func heredoc(file, body string) string {
	return "cat > " + file + " <<'EOF'\n" + body + "EOF\n"
}

// lastComment returns the text after "// " on the last line of the function
// name declares in readme_test.go, a newline appended: the output README's
// snippet states that line prints.
func lastComment(t *testing.T, name string) string {
	t.Helper()
	body, err := funcBody("readme_test.go", name)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	_, comment, ok := strings.Cut(lines[len(lines)-1], "// ")
	if !ok {
		t.Fatalf("%s's last line carries no comment", name)
	}
	return comment + "\n"
}

// stdout returns what f writes to os.Stdout.
func stdout(t *testing.T, f func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = saved }()
	f()
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// TestReadmeProgram holds testdata/readme/main.go, the file README's Library
// section names as its program, to the program readmeProgram joins from
// README's blocks. Each row then edits the file or README in one place and
// must make readmeProgram report the mismatch.
func TestReadmeProgram(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	file, err := os.ReadFile(filepath.Join("testdata", "readme", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if err := readmeProgram(string(readme), string(file)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, old, new string
		inReadme       bool
	}{
		{name: "first block", old: `"builtin types:"`, new: `"builtin type:"`},
		{name: "validate block", old: "fmt.Println(res.Standing())", new: "fmt.Print(res.Standing())"},
		{name: "jsonsrc block", old: `jsonsrc.WithURI("order.json")`, new: `jsonsrc.WithURI("order.jsn")`},
		{name: "codegen block", old: `Dir: "gen/order"`, new: `Dir: "gen/ordr"`},
		{name: "closing brace", old: "\tos.Exit(1)\n\t}\n}\n", new: "\tos.Exit(1)\n\t}\n"},
		{name: "README fence", old: "fmt.Println(res.Standing())", new: "fmt.Print(res.Standing())", inReadme: true},
	} {
		r, f := string(readme), string(file)
		target := &f
		if c.inReadme {
			target = &r
		}
		if n := strings.Count(*target, c.old); n != 1 {
			t.Fatalf("%s: %q occurs %d times, want once", c.name, c.old, n)
		}
		*target = strings.Replace(*target, c.old, c.new, 1)
		if readmeProgram(r, f) == nil {
			t.Errorf("%s: readmeProgram reports no mismatch after %q became %q", c.name, c.old, c.new)
		}
	}
}

// TestReadmeProgramCompiles holds testdata/readme/main.go to compiling, which
// checks the one span TestReadmeProgram leaves to it, the file's import
// declaration. No pattern of gate part 1 compiles the program, since ./...
// skips testdata. go vet writes nothing into the tree, so the test needs no
// gen/ and leaves none.
func TestReadmeProgramCompiles(t *testing.T) {
	out, err := exec.Command("go", "vet", "./testdata/readme").CombinedOutput()
	if err != nil {
		t.Fatalf("go vet ./testdata/readme: %v: %s", err, out)
	}
}

// readmeProgram reports where file departs from the program README's blocks
// give, or nil when it reads as they do. The blocks are the go fence that
// opens "package main" and the three go fences after it, in README order. The
// program is the first block with its closing "}" line moved to the end and
// its import declaration replaced by file's, then each later block, preceded
// by a blank line and with one tab before each non-empty line. file's import
// declaration is the one span it does not hold: README adds those imports in
// prose, and TestReadmeProgramCompiles checks them.
func readmeProgram(readme, file string) error {
	var fences []string
	var body strings.Builder
	open := false
	for line := range strings.Lines(readme) {
		switch {
		case !open && line == "```go\n":
			open = true
			body.Reset()
		case open && line == "```\n":
			open = false
			fences = append(fences, body.String())
		case open:
			body.WriteString(line)
		}
	}
	first := -1
	for i, f := range fences {
		if !strings.HasPrefix(f, "package main\n") {
			continue
		}
		if first >= 0 {
			return errors.New("README.md has two go fences that open package main")
		}
		first = i
	}
	if first < 0 || first+4 > len(fences) {
		return errors.New("README.md has no go fence that opens package main with three go fences after it")
	}
	head, err := withImports(fences[first], file)
	if err != nil {
		return err
	}
	head, ok := strings.CutSuffix(head, "}\n")
	if !ok {
		return errors.New(`README.md's go fence that opens package main does not end in "}"`)
	}
	var want strings.Builder
	want.WriteString(head)
	for _, block := range fences[first+1 : first+4] {
		want.WriteString("\n")
		for line := range strings.Lines(block) {
			if line != "\n" {
				want.WriteString("\t")
			}
			want.WriteString(line)
		}
	}
	want.WriteString("}\n")
	return firstDifference(file, want.String())
}

// withImports returns src with its one import declaration replaced by file's.
func withImports(src, file string) (string, error) {
	start, end, err := importDecl(src)
	if err != nil {
		return "", fmt.Errorf("README.md's go fence that opens package main: %w", err)
	}
	fstart, fend, err := importDecl(file)
	if err != nil {
		return "", fmt.Errorf("testdata/readme/main.go: %w", err)
	}
	return src[:start] + file[fstart:fend] + src[end:], nil
}

// importDecl returns the byte offsets of src's one import declaration.
func importDecl(src string) (start, end int, err error) {
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, "", src, goparser.ImportsOnly)
	if err != nil {
		return 0, 0, err
	}
	var decls []ast.Decl
	for _, d := range f.Decls {
		if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
			decls = append(decls, g)
		}
	}
	if len(decls) != 1 {
		return 0, 0, fmt.Errorf("%d import declarations, want 1", len(decls))
	}
	return fset.Position(decls[0].Pos()).Offset, fset.Position(decls[0].End()).Offset, nil
}

// firstDifference reports the first line at which testdata/readme/main.go's
// text got departs from want, or nil when they are equal.
func firstDifference(got, want string) error {
	if got == want {
		return nil
	}
	g, w := strings.SplitAfter(got, "\n"), strings.SplitAfter(want, "\n")
	for i := range max(len(g), len(w)) {
		var gl, wl string
		if i < len(g) {
			gl = g[i]
		}
		if i < len(w) {
			wl = w[i]
		}
		if gl != wl {
			return fmt.Errorf("testdata/readme/main.go:%d reads %q, README.md's blocks give %q", i+1, gl, wl)
		}
	}
	return errors.New("testdata/readme/main.go departs from README.md's blocks")
}

// TestReadmeSharedDocument holds README's two-target gen example: note.xsd,
// ship.xsd and bill.xsd byte for byte as fences' bodies, and the
// gen/bill/xsd_gen.go README shows to the file Generate writes for the run
// README shows, byte for byte as a fence's body. It takes gen's route: each
// document is parsed by its absolute path under a resolver rooted at the
// filesystem root, and Generate runs in testdata/readme beside the checkout's
// go.mod, each in a copy under a temp directory, so the headers and the import
// path read as gen writes them in the checkout.
func TestReadmeSharedDocument(t *testing.T) {
	readme, err := os.ReadFile("README.md")
	if err != nil {
		t.Fatal(err)
	}
	gomod, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), gomod, 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "testdata", "readme")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	docs := map[string]string{}
	for _, file := range []string{"note.xsd", "ship.xsd", "bill.xsd"} {
		body, err := os.ReadFile(filepath.Join("testdata", "readme", file))
		if err != nil {
			t.Fatal(err)
		}
		docs[file] = string(body)
		if err := os.WriteFile(filepath.Join(dir, file), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, file := range []string{"note.xsd", "ship.xsd", "bill.xsd"} {
		if want := "```xml\n" + docs[file] + "```\n"; !strings.Contains(string(readme), want) {
			t.Errorf("README.md does not show testdata/readme/%s as\n%s", file, want)
		}
	}

	var targets []codegen.Target
	for _, file := range []string{"ship.xsd", "bill.xsd"} {
		abs := filepath.Join(dir, file)
		schema, err := parser.Parse(abs, parser.WithResolver(loader.Dir(filepath.VolumeName(abs)+string(filepath.Separator))))
		if err != nil {
			t.Fatal(err)
		}
		pkg := strings.TrimSuffix(file, ".xsd")
		targets = append(targets, codegen.Target{Schema: schema, Dir: "./gen/" + pkg, Package: pkg})
	}
	t.Chdir(dir)
	if err := codegen.Generate(targets); err != nil {
		t.Fatal(err)
	}
	ship, err := os.ReadFile(filepath.Join("gen", "ship", "xsd_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ship), "\ntype Note string\n") {
		t.Errorf("gen/ship/xsd_gen.go does not declare type Note string, as README states:\n%s", ship)
	}
	bill, err := os.ReadFile(filepath.Join("gen", "bill", "xsd_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "```go\n" + string(bill) + "```\n"; !strings.Contains(string(readme), want) {
		t.Errorf("README.md does not show the gen/bill/xsd_gen.go Generate writes for testdata/readme's ship.xsd and bill.xsd as\n%s", want)
	}
}

// readmeCodegen is README's code-generation snippet: README shows its body
// byte for byte, each line less one leading tab. schema stands for the
// program's variable of that name.
func readmeCodegen(schema *xsd.Schema) {
	// Writes package order into gen/order.
	if errs := codegen.Generate([]codegen.Target{{Schema: schema, Dir: "gen/order", Package: "order"}}); errs != nil {
		// Generate returns one *codegen.TargetError per failure: print each, a
		// decline by its Loc and Construct, any other failure as it reads.
		for _, e := range errs {
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

// readmeDecode is README's decode snippet: README shows its body byte for
// byte, each line less one leading tab. schema and backend stand for the
// program's variables of those names, and package order for the package
// readmeCodegen writes.
func readmeDecode(schema *xsd.Schema, backend value.Backend) {
	data, err := os.ReadFile("order.xml")
	if err != nil {
		log.Fatal(err)
	}
	// order.Order is the generated type for order.xml's document element.
	o, err := codec.Decode[order.Order](schema, backend, data, codec.WithURI("order.xml"))
	if err != nil {
		var ue *codec.UndecidedError
		var de *codec.DeclineError
		rule, _ := xsderr.RuleOf(err)
		switch vs := codec.ViolationsOf(err); {
		case vs != nil:
			// A charged instance: every violation, ViolationsOf's.
			for _, v := range vs {
				fmt.Println(v.Loc, v.Rule)
			}
		case rule == xsderr.RuleXMLWellFormed:
			// The instance's source fault, an *xsderr.Error.
			fmt.Println(err)
		case errors.As(err, &ue):
			// An *UndecidedError: the check the validator left undecided.
			fmt.Println(ue.Unevaluated().Loc(), ue.Unevaluated().Rule())
		case errors.As(err, &de):
			// A *DeclineError: the construct codec declined, charged to an
			// instance Loc.
			loc, _ := de.Loc()
			fmt.Println(loc, de.Construct())
		case errors.Is(err, codec.ErrTypeMismatch):
			// ErrTypeMismatch: T does not match the declaration.
			fmt.Println(err)
		default:
			// A plain error: a fault of the arguments, of the schema or of
			// the backend.
			fmt.Println(err)
		}
		log.Fatal(err)
	}
	fmt.Println(o.Item, *o.Kind) // [a b] x
}

// readmeAppend is README's encode snippet: README shows its body byte for
// byte, each line less one leading tab. schema and backend stand for the
// program's variables of those names, and o for the *order.Order
// readmeDecode's snippet decodes.
func readmeAppend(schema *xsd.Schema, backend value.Backend, o *order.Order) {
	// Append takes Decode's *o dereferenced and order.xml's document element
	// {urn:o}order.
	out, err := codec.Append(nil, schema, backend, xsd.QName{Space: "urn:o", Local: "order"}, *o)
	if err != nil {
		var de *codec.DeclineError
		switch {
		case errors.Is(err, codec.ErrInvalidValue):
			// ErrInvalidValue: a value no valid document holds, or a
			// violation of the document Append wrote.
			fmt.Println(err)
		case errors.As(err, &de):
			// A *DeclineError: a construct, or a value, that codec declines.
			fmt.Println(de.Construct())
		case errors.Is(err, codec.ErrTypeMismatch):
			// ErrTypeMismatch: T does not match the declaration.
			fmt.Println(err)
		case errors.Is(err, codec.ErrUndeclaredElement), errors.Is(err, codec.ErrAbstractElement):
			// ErrUndeclaredElement or ErrAbstractElement: the schema declares
			// no top-level element of that name, or declares it abstract.
			fmt.Println(err)
		default:
			// Any other error: a plain error, a fault of the arguments or of
			// the backend, or the *UndecidedError of an undecided document.
			fmt.Println(err)
		}
		log.Fatal(err)
	}
	fmt.Println(string(out)) // <order xmlns="urn:o" kind="x"><item>a</item><item>b</item></order>
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
