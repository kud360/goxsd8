package backendtest

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

func booleanQName() xsd.QName { return xsd.QName{Space: xsd.XMLSchemaNS, Local: "boolean"} }
func decimalQName() xsd.QName { return xsd.QName{Space: xsd.XMLSchemaNS, Local: "decimal"} }
func stringQName() xsd.QName  { return xsd.QName{Space: xsd.XMLSchemaNS, Local: "string"} }
func floatQName() xsd.QName   { return xsd.QName{Space: xsd.XMLSchemaNS, Local: "float"} }
func doubleQName() xsd.QName  { return xsd.QName{Space: xsd.XMLSchemaNS, Local: "double"} }
func hexBinaryQName() xsd.QName {
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: "hexBinary"}
}
func base64BinaryQName() xsd.QName {
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: "base64Binary"}
}
func durationQName() xsd.QName {
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: "duration"}
}
func dateTimeQName() xsd.QName {
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: "dateTime"}
}

// othersAbsent declares the cohort types other than boolean absent, so the
// boolean-only test backends below are checked purely on boolean without Run
// reporting the (intentionally) unmapped
// decimal/string/float/double/hexBinary/base64Binary/duration/dateTime vectors.
func othersAbsent() []Option {
	return []Option{Absent(decimalQName(), stringQName(), floatQName(), doubleQName(), hexBinaryQName(), base64BinaryQName(), durationQName(), dateTimeQName())}
}

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
	run(&rg, good, othersAbsent())
	if rg.errs != 0 {
		t.Fatalf("correct backend: run reported %d failures, want 0", rg.errs)
	}

	bad := mapBackend{booleanQName(): buggyBoolean()}
	var rb recordT
	run(&rb, bad, othersAbsent())
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
	run(&r2, empty, []Option{Absent(booleanQName(), decimalQName(), stringQName(), floatQName(), doubleQName(), hexBinaryQName(), base64BinaryQName(), durationQName(), dateTimeQName())})
	if r2.errs != 0 {
		t.Fatalf("Absent(all cohort): run reported %d failures, want 0", r2.errs)
	}
}

// TestRunPublic exercises the exported Run against a *testing.T with a correct
// backend: the public entry point must run the generated vectors and pass.
func TestRunPublic(t *testing.T) {
	Run(t, mapBackend{booleanQName(): correctBoolean()}, othersAbsent()...)
}

// orderedVal implements value.Ordered (hence value.Eq) so a synthetic mapping
// can produce a capability-carrying value.
type orderedVal struct{}

func (orderedVal) Eq(value.Value) bool            { return true }
func (orderedVal) Cmp(value.Value) value.Ordering { return value.Equal }

// TestRequiredCapabilityClassification pins the fixed facet→capability table,
// including that an unrecognized facet name is reported (ok=false).
func TestRequiredCapabilityClassification(t *testing.T) {
	cases := map[string]struct {
		cap capability
		ok  bool
	}{
		"minInclusive":     {capOrdered, true},
		"maxExclusive":     {capOrdered, true},
		"totalDigits":      {capDigitCounted, true},
		"fractionDigits":   {capDigitCounted, true},
		"length":           {capLengthed, true},
		"maxLength":        {capLengthed, true},
		"maxScale":         {capScaled, true},
		"enumeration":      {capEq, true},
		"explicitTimezone": {capTimezoneAware, true},
		"whiteSpace":       {capNone, true},
		"pattern":          {capNone, true},
		"assertions":       {capNone, true},
		"bogusFacet":       {capNone, false},
	}
	for facet, want := range cases {
		gotCap, gotOK := requiredCapability(facet)
		if gotCap != want.cap || gotOK != want.ok {
			t.Errorf("requiredCapability(%q) = (%v,%v), want (%v,%v)", facet, gotCap, gotOK, want.cap, want.ok)
		}
	}
}

// TestCheckCapabilitiesCatchesMissing proves checkCapabilities fails loudly when
// a produced value misses a required capability and passes when it carries it.
func TestCheckCapabilitiesCatchesMissing(t *testing.T) {
	// A mapping whose value carries no capabilities at all.
	bare := value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return struct{}{}, nil }}
	ordered := value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return orderedVal{}, nil }}

	tv := typeVectors{
		typ:              xsd.QName{Space: xsd.XMLSchemaNS, Local: "synthetic"},
		valid:            []roundtrip{{lexical: "x", canonical: "x"}},
		applicableFacets: []string{"minInclusive"}, // requires value.Ordered
	}

	var miss recordT
	checkCapabilities(&miss, tv, bare)
	if miss.errs == 0 {
		t.Fatal("bare value missing value.Ordered: checkCapabilities reported 0 failures, want > 0")
	}

	var ok recordT
	checkCapabilities(&ok, tv, ordered)
	if ok.errs != 0 {
		t.Fatalf("ordered value: checkCapabilities reported %d failures, want 0", ok.errs)
	}
}

// TestCheckCapabilitiesUnclassifiedFacet proves an applicable facet the table
// does not recognize is reported, never silently skipped.
func TestCheckCapabilitiesUnclassifiedFacet(t *testing.T) {
	tv := typeVectors{
		typ:              xsd.QName{Space: xsd.XMLSchemaNS, Local: "synthetic"},
		valid:            []roundtrip{{lexical: "x", canonical: "x"}},
		applicableFacets: []string{"bogusFacet"},
	}
	pass := value.Mapping{Parse: func(string, value.Context) (value.Value, error) { return orderedVal{}, nil }}
	var r recordT
	checkCapabilities(&r, tv, pass)
	if r.errs == 0 {
		t.Fatal("unclassified facet: checkCapabilities reported 0 failures, want > 0")
	}
}
