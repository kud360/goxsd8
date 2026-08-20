package validate

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// Option configures [New]. The zero set of options is a complete, usable
// configuration: options only replace defaults, never combine into an
// invalid one (STYLE T1).
type Option func(*config)

// config is the resolved [New] configuration. Every field is non-nil from
// construction onward, since each Option installs a usable replacement.
type config struct {
	log *slog.Logger
}

// newConfig applies opts over the defaults: nothing is logged.
func newConfig(opts []Option) config {
	cfg := config{log: slog.New(slog.DiscardHandler)}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithLogger sets the logger the validator reports assessment progress on,
// at debug level, under the "validate" group [New] installs (STYLE L1). A
// nil logger selects the silent default ([slog.DiscardHandler]), so
// assessment is quiet unless a logger is asked for.
func WithLogger(l *slog.Logger) Option {
	if l == nil {
		l = slog.New(slog.DiscardHandler)
	}
	return func(c *config) { c.log = l }
}

// Validator assesses instance documents against one compiled schema.
//
// It is immutable and reusable by layout, not by promise: it holds the
// schema, the backend and the logger and nothing else — no cache of a
// governing mapping, no compiled facets (STYLE D3; package value recomputes
// both by design) — and every piece of walk state lives in a struct
// [Validator.Assess] creates and drops, so a second assessment cannot
// observe the first one's state.
type Validator struct {
	schema  *xsd.Schema
	backend value.Backend
	log     *slog.Logger
}

// New returns a [Validator] assessing instances against schema, reading
// instance lexicals in backend's value space. It takes the one finalized
// [xsd.Schema] parser.Parse assembles — the §3.17.1 Schema component is
// already multi-namespace-capable, so there is no separate schema-set type.
//
// backend MUST be the backend schema was compiled with. Its facets were
// checked in the value space [xsd.SchemaBuilder.FinalizeWith] installed at
// assembly, so assessing with a different one parses instance lexicals in a
// space no facet on the schema was ever checked against: a bound facet
// admitting a value the instance mapping cannot represent, or representing
// it differently, turns into a rejection of a document the schema admits.
// Nothing here can detect that — the schema does not carry the capability —
// so it is the caller's to honor. parser.Parse's own backend option is the
// one to pass.
//
// A nil backend panics, on [value.NewValueSpace]'s and parser.WithBackend's
// grounds: it is a caller bug, not a validity verdict about any document,
// and a parameter cannot be forgotten the way an option can. There is no
// default — installing one would either assess a schema compiled elsewhere
// under the wrong value space with no error, or make "a Validator that
// charges no datatype violation" a representable state whose empty [Result]
// is indistinguishable from a valid document.
//
// It returns an error on exactly one condition, a nil schema, and that
// error is a plain one rather than an [xsderr.Error]: a missing schema is a
// caller's unchecked [xsd.SchemaBuilder.Finalize] result, not a validity
// verdict about any document.
func New(schema *xsd.Schema, backend value.Backend, opts ...Option) (*Validator, error) {
	if backend == nil {
		panic("validate: New: nil value.Backend")
	}
	if schema == nil {
		return nil, fmt.Errorf("validate: New: nil *xsd.Schema")
	}
	cfg := newConfig(opts)
	return &Validator{schema: schema, backend: backend, log: cfg.log.WithGroup("validate")}, nil
}

// Schema returns the read-only view of the compiled schema an adapter needs
// to resolve a root element declaration by expanded name, and nothing else
// (STYLE T3). The walk reads the *xsd.Schema itself and not this narrowing, so
// widening the accessor is the job of a CALLER that needs more, never of a rule
// inside the package.
func (v *Validator) Schema() xsd.ElementResolver { return v.schema }

// Result is the outcome of one [Validator.Assess] call.
type Result struct {
	violations []*xsderr.Error
	err        error
}

// Violations returns the violations the assessment charged, in document
// order. The slice is copied: mutating the result does not affect r. No
// violation yields nil.
func (r *Result) Violations() []*xsderr.Error {
	if len(r.violations) == 0 {
		return nil
	}
	return append([]*xsderr.Error(nil), r.violations...)
}

// Err reports the fault in the source that stopped the walk, or nil when
// the walk reached the end of the document. A non-nil Err means the
// assessment is INCOMPLETE, not that the document is invalid: an invalid
// document is reported through [Result.Violations].
func (r *Result) Err() error { return r.err }

// causedBy builds the [xsderr.Error] for one violation, carrying cause — the
// verdict of the rule this one DELEGATES to — in both places a consumer looks
// for it, from ONE argument (STYLE D3): appended to the formatted Msg after
// ": ", and set as the wrapped cause so errors.Unwrap, errors.Is and errors.As
// reach the inner *xsderr.Error and the rule ID it carries. A reader holding
// only the rendered string has no chain to walk; a reader holding the error
// should not have to scrape a message for a rule ID; and taking one argument
// is what stops the two from naming different errors.
//
// format therefore stops at the delegating rule's own sentence — the ": " and
// the cause's own rendering are this function's to append. A nil cause yields
// the plain [xsderr.New] result, Msg unadorned and Unwrap nil, and reaches
// here only through [contentCheck.charge]: every other non-delegating charge
// site in the package calls [xsderr.New] itself.
func causedBy(rule xsderr.Rule, loc xsderr.Loc, cause error, format string, args ...any) *xsderr.Error {
	if cause == nil {
		return xsderr.New(rule, loc, format, args...)
	}
	v := xsderr.New(rule, loc, format+": %v", append(slices.Clone(args), cause)...)
	v.Err = cause
	return v
}
