// Package native is the Go-friendly value backend: familiar Go types
// with documented, deliberate deviations from the spec value spaces.
//
// # Planned representations (M12 — not yet implemented; contract fixed now)
//
//   - integer family — int64; lexicals outside int64 range are errors
//     (a deviation: the spec spaces are unbounded).
//   - decimal, float, double — float64 (a deviation: decimal loses
//     precision beyond float64).
//   - precisionDecimal — float64 with scale facets admitted but inert
//     (documented deviation; use strict where scale semantics matter).
//   - date/time family — time.Time with documented timezone folding;
//     duration — time.Duration (a deviation: no month/year components).
//   - string family — string; binaries — []byte; QName — a small
//     namespace/local struct.
//
// Every deviation is documented on the mapping and exercised by
// value/backendtest options that relax only the deviating vectors —
// nothing deviates silently.
//
// Use native for ergonomic data binding (codegen fields people want to
// touch); use strict when verdict-grade fidelity matters. Mix per type
// with value.Override.
//
// The package is designed to pass value/backendtest (with its declared
// deviations) and to implement value.Emitter for zero-allocation scalar
// decode paths.
package native
