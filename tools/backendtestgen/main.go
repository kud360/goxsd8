// Command backendtestgen emits value/backendtest/gen_vectors.go: the
// spec-derived conformance vectors that value/backendtest.Run drives every
// value.Backend through (PRINCIPLES 26/27). It parses the lexical/canonical
// mapping definitions out of the local Datatypes spec — nothing is
// hand-transcribed — and emits gofmt'd, deterministic Go, so running it twice
// is byte-identical (STYLE D1).
//
// M3 scope: boolean only (§3.3.2.2, f-booleanLexmap/f-booleanCanmap,
// nt-booleanRep). Later types append rows to the same table as their mapping
// definitions are wired in.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	structuresPath := flag.String("structures", "docs/specs/md/xmlschema11-2.md", "Datatypes spec Markdown")
	outPath := flag.String("out", "value/backendtest/gen_vectors.go", "output Go file")
	flag.Parse()

	if err := run(*structuresPath, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "backendtestgen: %v\n", err)
		os.Exit(1)
	}
}

func run(structuresPath, outPath string) error {
	content, err := os.ReadFile(structuresPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", structuresPath, err)
	}
	boolean, err := parseBoolean(string(content))
	if err != nil {
		return fmt.Errorf("parsing %s: %w", structuresPath, err)
	}
	src, err := emit([]typeVectors{boolean})
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, src, 0o644)
}
