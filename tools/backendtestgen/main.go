// Command backendtestgen emits value/backendtest/gen_vectors.go: the
// spec-derived conformance vectors that value/backendtest.Run drives every
// value.Backend through (PRINCIPLES 26/27). It parses the lexical/canonical
// mapping definitions out of the local Datatypes spec — nothing is
// hand-transcribed — and emits gofmt'd, deterministic Go, so running it twice
// is byte-identical (STYLE D1).
//
// M3 cohort: boolean (§3.3.2.2, f-booleanLexmap/f-booleanCanmap, nt-booleanRep),
// decimal (§3.3.3, decimal-lexical-representation, f-decimalLexmap/
// f-decimalCanmap) and string (§3.3.1.2, f-stringLexmap/f-stringCanmap, the
// identity). Each type also carries its applicable constraining facets in spec
// order (cos-applicable-facets, §4.1.5), read from the shared builtin spec
// parser (tools/hfnextract/builtins). Later types append rows to the same table
// as their mapping definitions are wired in.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kud360/goxsd8/tools/hfnextract/builtins"
)

func main() {
	structuresPath := flag.String("structures", "docs/specs/md/xmlschema11-2.md", "Datatypes spec Markdown")
	precisionPath := flag.String("precision", "docs/specs/md/xsd-precisionDecimal.md", "precisionDecimal Note Markdown (for the shared builtin parser)")
	outPath := flag.String("out", "value/backendtest/gen_vectors.go", "output Go file")
	flag.Parse()

	if err := run(*structuresPath, *precisionPath, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "backendtestgen: %v\n", err)
		os.Exit(1)
	}
}

func run(structuresPath, precisionPath, outPath string) error {
	src, err := build(structuresPath, precisionPath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, src, 0o644)
}

// build reads the spec, derives every cohort type's vectors and applicable
// facets, and returns the gofmt'd source. It is the pure core both run and the
// tests share, so a determinism test needs no filesystem write.
func build(structuresPath, precisionPath string) ([]byte, error) {
	content, err := os.ReadFile(structuresPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", structuresPath, err)
	}
	spec := string(content)

	boolean, err := parseBoolean(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: boolean: %w", structuresPath, err)
	}
	decimal, err := parseDecimal(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: decimal: %w", structuresPath, err)
	}
	str, err := parseString(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: string: %w", structuresPath, err)
	}

	facets, err := applicableFacets(structuresPath, precisionPath)
	if err != nil {
		return nil, err
	}
	boolean.ApplicableFacets = facets["boolean"]
	decimal.ApplicableFacets = facets["decimal"]
	str.ApplicableFacets = facets["string"]

	return emit([]typeVectors{boolean, decimal, str})
}

// applicableFacets reads each cohort type's applicable constraining facets in
// spec order (cos-applicable-facets, §4.1.5) from the shared builtin spec
// parser — the same source tools/typespecgen consumes, never a second hand-rolled
// parser (STYLE D1). The parser needs both spec files; precisionDecimal is not
// in this cohort, but Parse's signature requires the path.
func applicableFacets(structuresPath, precisionPath string) (map[string][]string, error) {
	types, err := builtins.Parse(structuresPath, precisionPath)
	if err != nil {
		return nil, fmt.Errorf("applicable facets: %w", err)
	}
	want := map[string]bool{"boolean": true, "decimal": true, "string": true}
	out := make(map[string][]string, len(want))
	for _, b := range types {
		if !want[b.Name] {
			continue
		}
		names := make([]string, 0, len(b.Facets))
		for _, f := range b.Facets {
			names = append(names, f.Name)
		}
		out[b.Name] = names
	}
	for name := range want {
		if _, ok := out[name]; !ok {
			return nil, fmt.Errorf("applicable facets: builtin %q not found in the spec parser output", name)
		}
	}
	return out, nil
}
