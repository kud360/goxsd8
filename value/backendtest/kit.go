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
	// Typ is the builtin the vectors exercise.
	Typ xsd.QName
	// Valid are lexicals in the type's lexical space, each paired with the
	// canonical form its value must render to (the round-trip check).
	Valid []roundtrip
	// Invalid are lexicals outside the type's lexical space; Parse must reject
	// each with an *xsderr.Error (cvc-datatype-valid).
	Invalid []string
	// NarrowReject are lexicals a wider ancestor space accepts but a derived,
	// narrow representation cannot hold; Parse must surface a mapping error,
	// never a validity verdict (the widest-space discipline, value.Backend doc).
	// It is empty for primitives, whose representation is the widest — the slot
	// exists for derived-type backends and is exercised once one lands.
	NarrowReject []string
}

// roundtrip is a valid lexical and the canonical form its value must render.
type roundtrip struct {
	// Lexical is a member of the type's lexical space.
	Lexical string
	// Canonical is the canonical mapping of the value Lexical parses to.
	Canonical string
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
// M3 exercises the lexical→value→canonical round-trips and invalid-lexical
// rejection. Order/identity, capability coverage, and the widest-space
// discipline described in the package doc arrive with the concrete backends
// that can exercise them.
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
		m, ok := b.Mapping(tv.Typ)
		if !ok {
			if cfg.absent[tv.Typ] {
				continue
			}
			r.Errorf("backendtest: %s has vectors but the backend does not map it (declare it Absent if intended)", tv.Typ)
			continue
		}
		checkType(r, tv, m)
	}
}

// checkType runs one type's vectors against its mapping.
func checkType(r reporter, tv typeVectors, m value.Mapping) {
	r.Helper()
	for _, rt := range tv.Valid {
		checkRoundtrip(r, tv.Typ, m, rt)
	}
	for _, lex := range tv.Invalid {
		checkRejected(r, tv.Typ, m, lex, "an invalid lexical (cvc-datatype-valid)")
	}
	for _, lex := range tv.NarrowReject {
		checkRejected(r, tv.Typ, m, lex, "a wide-valid lexical the narrow representation cannot hold")
	}
}

// checkRoundtrip asserts a valid lexical parses and canonicalizes as the spec
// requires.
func checkRoundtrip(r reporter, typ xsd.QName, m value.Mapping, rt roundtrip) {
	r.Helper()
	v, err := m.Parse(rt.Lexical, nil)
	if err != nil {
		r.Errorf("%s: Parse(%q) = error %v; want it to map", typ, rt.Lexical, err)
		return
	}
	if m.Canonical == nil {
		return
	}
	got, err := m.Canonical(v)
	if err != nil {
		r.Errorf("%s: Canonical(value of %q) = error %v", typ, rt.Lexical, err)
		return
	}
	if got != rt.Canonical {
		r.Errorf("%s: %q canonicalizes to %q; want %q", typ, rt.Lexical, got, rt.Canonical)
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
