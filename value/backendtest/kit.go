package backendtest

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// typeVectors is one builtin type's spec-derived vectors (see gen_vectors.go,
// emitted by tools/backendtestgen). The schema is unexported: the kit's API is
// [Run]; the corpus is an internal implementation detail (STYLE T5).
type typeVectors struct {
	// typ is the builtin the vectors exercise.
	typ xsd.QName
	// valid are lexicals in the type's lexical space, each paired with the
	// canonical form its value must render to (the round-trip check).
	valid []roundtrip
	// invalid are lexicals outside the type's lexical space; Parse must reject
	// each with an *xsderr.Error (cvc-datatype-valid).
	invalid []string
	// narrowReject are lexicals a wider ancestor space accepts but a derived,
	// narrow representation cannot hold; Parse must surface a mapping error,
	// never a validity verdict (the widest-space discipline, value.Backend doc).
	// It is empty for primitives, whose representation is the widest — the slot
	// exists for derived-type backends and is exercised once one lands.
	narrowReject []string
	// applicableFacets are the type's applicable constraining facets in spec
	// order (cos-applicable-facets, §4.1.5). checkCapabilities asserts the value
	// the mapping produces carries the capability each facet's check requires.
	applicableFacets []string
}

// roundtrip is a valid lexical and the canonical form its value must render.
type roundtrip struct {
	// lexical is a member of the type's lexical space.
	lexical string
	// canonical is the canonical mapping of the value lexical parses to.
	canonical string
}

// Option configures a [Run]. Options are constructed with the exported option
// functions (e.g. [Absent]); the zero surface is "check every vector against
// the backend as given".
type Option func(*config)

// config accumulates the options a [Run] was given.
type config struct {
	// absent names the types the caller declared intentionally unmapped, so Run
	// skips them instead of reporting a missing mapping.
	absent map[xsd.QName]bool
}

// Absent declares that the backend intentionally does not map types — typically
// primitives it expects to be supplied by composition with [value.Override].
// Run then skips those types rather than reporting a missing mapping.
//
// Declaring a type Absent that the backend actually maps is a harmless no-op:
// Run only consults the declaration when the mapping is missing, so a mapped
// type is checked regardless — Absent can never mask a broken mapping.
//
// Full primitive-coverage checking (every builtin primitive mapped or declared
// Absent) arrives with the first concrete backend (see package doc); today Run
// checks coverage only over the types it has vectors for.
func Absent(types ...xsd.QName) Option {
	return func(c *config) {
		for _, t := range types {
			c.absent[t] = true
		}
	}
}

// reporter is the subset of *testing.T that the vector engine uses, so the
// engine can be tested against a recording fake. *testing.T satisfies it.
type reporter interface {
	Errorf(format string, args ...any)
	Helper()
}

// Run drives b through the spec-derived vectors, reporting a test failure for
// every deviation from the value contracts (see package doc). It is the
// conformance kit's entry point: a backend that passes Run for the types it
// covers implements those types' lexical↔value↔canonical mappings correctly.
//
// M3 exercises the lexical→value→canonical round-trips, invalid-lexical
// rejection, and capability coverage (each type's produced value implements the
// capability its applicable facets require, or Run fails loudly). Order/identity
// vectors and the widest-space discipline described in the package doc arrive
// with the derived-type facet model (#33) that can exercise them.
func Run(t *testing.T, b value.Backend, opts ...Option) {
	t.Helper()
	run(t, b, opts)
}

// run is Run's engine over the reporter interface (see [reporter]).
func run(r reporter, b value.Backend, opts []Option) {
	r.Helper()
	cfg := config{absent: map[xsd.QName]bool{}}
	for _, opt := range opts {
		opt(&cfg)
	}
	for _, tv := range vectors {
		m, ok := b.Mapping(tv.typ)
		if !ok {
			if cfg.absent[tv.typ] {
				continue
			}
			r.Errorf("backendtest: %s has vectors but the backend does not map it (declare it Absent if intended)", tv.typ)
			continue
		}
		checkType(r, tv, m)
	}
}

// checkType runs one type's vectors against its mapping.
func checkType(r reporter, tv typeVectors, m value.Mapping) {
	r.Helper()
	for _, rt := range tv.valid {
		checkRoundtrip(r, tv.typ, m, rt)
	}
	for _, lex := range tv.invalid {
		checkRejected(r, tv.typ, m, lex, "an invalid lexical (cvc-datatype-valid)")
	}
	for _, lex := range tv.narrowReject {
		checkRejected(r, tv.typ, m, lex, "a wide-valid lexical the narrow representation cannot hold")
	}
	checkCapabilities(r, tv, m)
}

// capability names the value-capability interface a facet's check requires. The
// zero value capNone means the facet needs no value capability.
type capability uint8

const (
	// capNone marks a facet whose check needs no value capability.
	capNone capability = iota
	capOrdered
	capDigitCounted
	capLengthed
	capScaled
	capEq
	capTimezoneAware
)

// requiredCapability classifies one applicable constraining facet by the value
// capability its check requires (dv_vfacets, §4.1.4). ok is false for a facet
// name the kit does not recognize — an unclassified facet is drift the kit must
// catch loudly, never silently skip.
//
// This table is fixed, hand-written and spec-cited — NOT generated data
// (PRINCIPLES 26, warden pre-flight): only each type's applicableFacets list is
// spec-derived. whiteSpace, pattern and assertions map to capNone: §4.1.4
// dv_vfacets is value-based and excludes them — whiteSpace and pattern act on
// the normalized lexical (pre-value stages) and assertions are an XPath
// mechanism, so none constrains a produced value's capabilities.
func requiredCapability(facet string) (capability, bool) {
	switch facet {
	case "minInclusive", "maxInclusive", "minExclusive", "maxExclusive": // §4.3.7–§4.3.10
		return capOrdered, true
	case "totalDigits", "fractionDigits": // §4.3.11–§4.3.12
		return capDigitCounted, true
	case "length", "minLength", "maxLength": // §4.3.1–§4.3.3
		return capLengthed, true
	case "maxScale", "minScale": // xsd-precisionDecimal (no cohort type uses these yet)
		return capScaled, true
	case "enumeration": // §4.3.5, cvc-enumeration-valid — "equal or identical"
		return capEq, true
	case "explicitTimezone": // §4.3.14, cvc-explicitTimezone-valid — reads HasTimezone
		return capTimezoneAware, true
	case "whiteSpace", "pattern", "assertions": // pre-lexical / lexical / assertion stages
		return capNone, true
	}
	return capNone, false
}

// checkCapabilities asserts the value the mapping produces implements every
// capability its applicable facets require (value.Backend doc: "a backend's
// values must implement the capabilities its types' applicable facets require").
// It parses the first valid lexical to obtain a representative value, then for
// each applicable facet looks up the required capability and type-asserts the
// value against it — failing loudly on a missing capability or an unclassified
// facet name.
func checkCapabilities(r reporter, tv typeVectors, m value.Mapping) {
	r.Helper()
	if len(tv.valid) == 0 {
		return
	}
	v, err := m.Parse(tv.valid[0].lexical, nil)
	if err != nil {
		return // the round-trip check already reported this parse failure
	}
	for _, facet := range tv.applicableFacets {
		cap, ok := requiredCapability(facet)
		if !ok {
			r.Errorf("%s: applicable facet %q is not classified by requiredCapability; an unclassified facet is drift the kit must catch", tv.typ, facet)
			continue
		}
		if hasCapability(v, cap) {
			continue
		}
		r.Errorf("%s: value of %q lacks %s, required by applicable facet %q", tv.typ, tv.valid[0].lexical, capabilityName(cap), facet)
	}
}

// hasCapability reports whether v implements the interface capability c names.
func hasCapability(v value.Value, c capability) bool {
	switch c {
	case capNone:
		return true
	case capOrdered:
		_, ok := v.(value.Ordered)
		return ok
	case capDigitCounted:
		_, ok := v.(value.DigitCounted)
		return ok
	case capLengthed:
		_, ok := v.(value.Lengthed)
		return ok
	case capScaled:
		_, ok := v.(value.Scaled)
		return ok
	case capEq:
		_, ok := v.(value.Eq)
		return ok
	case capTimezoneAware:
		_, ok := v.(value.TimezoneAware)
		return ok
	}
	return false
}

// capabilityName is the diagnostic name of a capability for failure messages.
func capabilityName(c capability) string {
	switch c {
	case capNone:
		return "no capability"
	case capOrdered:
		return "value.Ordered"
	case capDigitCounted:
		return "value.DigitCounted"
	case capLengthed:
		return "value.Lengthed"
	case capScaled:
		return "value.Scaled"
	case capEq:
		return "value.Eq"
	case capTimezoneAware:
		return "value.TimezoneAware"
	}
	return "no capability"
}

// checkRoundtrip asserts a valid lexical parses and canonicalizes as the spec
// requires.
func checkRoundtrip(r reporter, typ xsd.QName, m value.Mapping, rt roundtrip) {
	r.Helper()
	v, err := m.Parse(rt.lexical, nil)
	if err != nil {
		r.Errorf("%s: Parse(%q) = error %v; want it to map", typ, rt.lexical, err)
		return
	}
	if m.Canonical == nil {
		return
	}
	got, err := m.Canonical(v)
	if err != nil {
		r.Errorf("%s: Canonical(value of %q) = error %v", typ, rt.lexical, err)
		return
	}
	if got != rt.canonical {
		r.Errorf("%s: %q canonicalizes to %q; want %q", typ, rt.lexical, got, rt.canonical)
	}
}

// checkRejected asserts a lexical is rejected with a proper *xsderr.Error.
func checkRejected(r reporter, typ xsd.QName, m value.Mapping, lex, why string) {
	r.Helper()
	_, err := m.Parse(lex, nil)
	if err == nil {
		r.Errorf("%s: Parse(%q) accepted %s; want a mapping error", typ, lex, why)
		return
	}
	if _, ok := xsderr.RuleOf(err); !ok {
		r.Errorf("%s: Parse(%q) rejected with a non-xsderr error %T; want an *xsderr.Error", typ, lex, err)
	}
}
