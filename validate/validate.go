package validate

import (
	"fmt"
	"log/slog"

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
// schema and the logger and nothing else, and every piece of walk state
// lives in a struct [Validator.Assess] creates and drops, so a second
// assessment cannot observe the first one's state.
type Validator struct {
	schema *xsd.Schema
	log    *slog.Logger
}

// New returns a [Validator] assessing instances against schema. It takes
// the one finalized [xsd.Schema] parser.Parse assembles — the §3.17.1
// Schema component is already multi-namespace-capable, so there is no
// separate schema-set type.
//
// It returns an error on exactly one condition, a nil schema, and that
// error is a plain one rather than an [xsderr.Error]: a missing schema is a
// caller's unchecked [xsd.SchemaBuilder.Finalize] result, not a validity
// verdict about any document.
func New(schema *xsd.Schema, opts ...Option) (*Validator, error) {
	if schema == nil {
		return nil, fmt.Errorf("validate: New: nil *xsd.Schema")
	}
	cfg := newConfig(opts)
	return &Validator{schema: schema, log: cfg.log.WithGroup("validate")}, nil
}

// Schema returns the read-only view of the compiled schema an adapter needs
// to resolve a root element declaration by expanded name, and nothing else
// (STYLE T3). Widening it is the job of the slice that needs the wider
// capability: [xsd.AttributeResolver] and [xsd.TypeResolver] are there when
// xsi:type resolution arrives.
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
