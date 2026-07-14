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
	datatypesPath := flag.String("datatypes", "docs/specs/md/xmlschema11-2.md", "Datatypes spec Markdown")
	precisionPath := flag.String("precision", "docs/specs/md/xsd-precisionDecimal.md", "precisionDecimal Note Markdown (for the shared builtin parser)")
	outPath := flag.String("out", "value/backendtest/gen_vectors.go", "output Go file")
	flag.Parse()

	if err := run(*datatypesPath, *precisionPath, *outPath); err != nil {
		fmt.Fprintf(os.Stderr, "backendtestgen: %v\n", err)
		os.Exit(1)
	}
}

func run(datatypesPath, precisionPath, outPath string) error {
	src, err := build(datatypesPath, precisionPath)
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, src, 0o644)
}

// build reads the spec, derives every cohort type's vectors and applicable
// facets, and returns the gofmt'd source. It is the pure core both run and the
// tests share, so a determinism test needs no filesystem write.
func build(datatypesPath, precisionPath string) ([]byte, error) {
	content, err := os.ReadFile(datatypesPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", datatypesPath, err)
	}
	spec := string(content)

	boolean, err := parseBoolean(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: boolean: %w", datatypesPath, err)
	}
	decimal, err := parseDecimal(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: decimal: %w", datatypesPath, err)
	}
	str, err := parseString(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: string: %w", datatypesPath, err)
	}
	flt, err := parseFloating(spec, "float", 32)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: float: %w", datatypesPath, err)
	}
	dbl, err := parseFloating(spec, "double", 64)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: double: %w", datatypesPath, err)
	}
	hexBin, err := parseHexBinary(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: hexBinary: %w", datatypesPath, err)
	}
	b64Bin, err := parseBase64Binary(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: base64Binary: %w", datatypesPath, err)
	}
	dur, err := parseDuration(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: duration: %w", datatypesPath, err)
	}
	dt, err := parseDateTime(spec)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: dateTime: %w", datatypesPath, err)
	}

	facets, err := applicableFacets(datatypesPath, precisionPath)
	if err != nil {
		return nil, err
	}
	boolean.ApplicableFacets = facets["boolean"]
	decimal.ApplicableFacets = facets["decimal"]
	str.ApplicableFacets = facets["string"]
	flt.ApplicableFacets = facets["float"]
	dbl.ApplicableFacets = facets["double"]
	hexBin.ApplicableFacets = facets["hexBinary"]
	b64Bin.ApplicableFacets = facets["base64Binary"]
	dur.ApplicableFacets = facets["duration"]
	dt.ApplicableFacets = facets["dateTime"]

	return emit([]typeVectors{boolean, decimal, str, flt, dbl, hexBin, b64Bin, dur, dt})
}

// applicableFacets reads each cohort type's applicable constraining facets in
// spec order (cos-applicable-facets, §4.1.5) from the shared builtin spec
// parser — the same source tools/typespecgen consumes, never a second hand-rolled
// parser (STYLE D1). The parser needs both spec files; precisionDecimal is not
// in this cohort, but Parse's signature requires the path.
func applicableFacets(datatypesPath, precisionPath string) (map[string][]string, error) {
	types, err := builtins.Parse(datatypesPath, precisionPath)
	if err != nil {
		return nil, fmt.Errorf("applicable facets: %w", err)
	}
	want := map[string]bool{"boolean": true, "decimal": true, "string": true, "float": true, "double": true, "hexBinary": true, "base64Binary": true, "duration": true, "dateTime": true}
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
