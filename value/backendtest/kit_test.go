package backendtest

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

const xsdNS = "http://www.w3.org/2001/XMLSchema"

func booleanQName() xsd.QName { return xsd.QName{Space: xsdNS, Local: "boolean"} }

// mapBackend is a test value.Backend over an explicit table.
type mapBackend map[xsd.QName]value.Mapping

func (b mapBackend) Mapping(typ xsd.QName) (value.Mapping, bool) {
	m, ok := b[typ]
	return m, ok
}

// correctBoolean maps boolean per §3.3.2.2: the four literals parse, anything
// else is a cvc-datatype-valid error, and the value canonicalizes to true/false.
func correctBoolean() value.Mapping {
	return value.Mapping{
		Parse: func(lexical string, _ value.Context) (value.Value, error) {
			switch lexical {
			case "true", "1":
				return true, nil
			case "false", "0":
				return false, nil
			}
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{}, "boolean: %q not in lexical space", lexical)
		},
		Canonical: func(v value.Value) (string, error) {
			if v.(bool) {
				return "true", nil
			}
			return "false", nil
		},
	}
}

// buggyBoolean accepts every lexical (never rejecting the invalids) and always
// canonicalizes to "true" — two contract violations Run must catch.
func buggyBoolean() value.Mapping {
	return value.Mapping{
		Parse: func(_ string, _ value.Context) (value.Value, error) {
			return true, nil
		},
		Canonical: func(value.Value) (string, error) {
			return "true", nil
		},
	}
}

// recordT is a recording reporter so tests can assert whether run reported any
// failure without failing the enclosing test.
type recordT struct{ errs int }

func (r *recordT) Errorf(string, ...any) { r.errs++ }
func (r *recordT) Helper()               {}

func TestRunCatchesBackends(t *testing.T) {
	good := mapBackend{booleanQName(): correctBoolean()}
	var rg recordT
	run(&rg, good, nil)
	if rg.errs != 0 {
		t.Fatalf("correct backend: run reported %d failures, want 0", rg.errs)
	}

	bad := mapBackend{booleanQName(): buggyBoolean()}
	var rb recordT
	run(&rb, bad, nil)
	if rb.errs == 0 {
		t.Fatal("buggy backend: run reported 0 failures, want > 0")
	}
}

func TestRunAbsentSkipsUnmapped(t *testing.T) {
	empty := mapBackend{}

	var r1 recordT
	run(&r1, empty, nil)
	if r1.errs == 0 {
		t.Fatal("unmapped boolean without Absent: run reported 0 failures, want > 0")
	}

	var r2 recordT
	run(&r2, empty, []Option{Absent(booleanQName())})
	if r2.errs != 0 {
		t.Fatalf("Absent(boolean): run reported %d failures, want 0", r2.errs)
	}
}

// TestRunPublic exercises the exported Run against a *testing.T with a correct
// backend: the public entry point must run the generated vectors and pass.
func TestRunPublic(t *testing.T) {
	Run(t, mapBackend{booleanQName(): correctBoolean()})
}
