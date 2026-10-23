package codegen_test

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/kud360/goxsd8/codegen"
	"github.com/kud360/goxsd8/parser"
)

// ExampleGenerate parses a schema, generates package order from it into a
// temporary directory, and lists the types the generated file declares.
func ExampleGenerate() {
	s, err := parser.Parse("testdata/flat.xsd")
	if err != nil {
		log.Fatal(err)
	}
	root, err := os.MkdirTemp("", "codegen-example")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(root)
	dir := filepath.Join(root, "gen", "order")

	if err := codegen.Generate([]codegen.Target{{Schema: s, Dir: dir, Package: "order"}}); err != nil {
		log.Fatal(err)
	}

	src, err := os.ReadFile(filepath.Join(dir, "xsd_gen.go"))
	if err != nil {
		log.Fatal(err)
	}
	lines := bufio.NewScanner(bytes.NewReader(src))
	for lines.Scan() {
		if name, ok := strings.CutPrefix(lines.Text(), "type "); ok {
			fmt.Println(strings.Fields(name)[0])
		}
	}
	if err := lines.Err(); err != nil {
		log.Fatal(err)
	}
	// Output:
	// Note
	// Record
	// Entry
}
