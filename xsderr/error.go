package xsderr

import (
	"errors"
	"fmt"
)

// Rule is a spec validation rule ID, such as "cvc-complex-type.2.1",
// "cos-st-restricts", or "derivation-ok-restriction". Exactly one Rule
// identifies each Error. Every Rule constructed in this module is expected to
// be present in the generated catalog (see IsValidRule).
type Rule string

// RuleXMLWellFormed is the sentinel Rule for XML well-formedness faults
// (unbound namespace prefix, mismatched or unclosed tag, malformed XML) that
// are not XSD schema- or instance-validity violations and so have no
// spec-defined cvc-*/src-*/cos-*/sic-* rule ID. It lets such an Error carry a
// recognizable, non-empty Rule instead of "" (which would be indistinguishable
// from a caller that simply forgot to set one). IsValidRule accepts it as a
// documented, non-spec exemption from the generated catalog.
const RuleXMLWellFormed Rule = "xml-wf"

// Loc identifies where an offending construct lives — the schema document or
// the instance document. Its fields are threaded from parser positions, never
// reconstructed. The zero Loc means the location is unknown.
type Loc struct {
	// URI is the document the construct was read from.
	URI string
	// Line is the 1-based line number, or 0 when unknown.
	Line int
	// Col is the 1-based column number, or 0 when unknown.
	Col int
}

// String renders a Loc as "uri:line:col". The zero Loc renders as "?", and an
// absent URI renders as "?".
func (l Loc) String() string {
	if l == (Loc{}) {
		return "?"
	}
	uri := l.URI
	if uri == "" {
		uri = "?"
	}
	return fmt.Sprintf("%s:%d:%d", uri, l.Line, l.Col)
}

// Error is the module's structured error currency: a validity verdict carrying
// the spec Rule it violates and the source Loc of the offending construct.
// Every schema- or instance-validity violation in the module is an *Error.
type Error struct {
	// Rule is the spec validation rule the construct violates.
	Rule Rule
	// Loc is where the offending construct lives; the zero Loc means unknown.
	Loc Loc
	// Msg is the human-readable explanation.
	Msg string
	// Err is an optional wrapped cause; Unwrap returns it.
	Err error
}

// Error renders the error as "loc: [rule] msg".
func (e *Error) Error() string {
	return fmt.Sprintf("%s: [%s] %s", e.Loc, e.Rule, e.Msg)
}

// Unwrap returns the wrapped cause so errors.Is and errors.As reach through the
// wrapping; a wrapper that hid its cause would break sentinel detection up the
// chain.
func (e *Error) Unwrap() error {
	return e.Err
}

// New builds an *Error attributing a formatted message to rule at loc, with no
// wrapped cause.
func New(rule Rule, loc Loc, format string, args ...any) *Error {
	return &Error{Rule: rule, Loc: loc, Msg: fmt.Sprintf(format, args...)}
}

// Wrap attaches rule and loc to a deeper plain error, preserving its message
// verbatim (Msg is err.Error()) and its identity (Unwrap returns err so
// errors.Is/As reach the wrapped cause). Wrap returns nil when err is nil.
func Wrap(rule Rule, loc Loc, err error) *Error {
	if err == nil {
		return nil
	}
	return &Error{Rule: rule, Loc: loc, Msg: err.Error(), Err: err}
}

// RuleOf reports the Rule of the first *Error in err's chain. The second result
// is false when the chain holds no *Error.
func RuleOf(err error) (Rule, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e.Rule, true
	}
	return "", false
}

// LocOf reports the Loc of the first *Error in err's chain. The second result
// is false when the chain holds no *Error.
func LocOf(err error) (Loc, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e.Loc, true
	}
	return Loc{}, false
}

// IsValidRule reports whether r is a Rule the module is allowed to construct:
// any rule ID present in the generated spec catalog (ruleCatalog, emitted by
// tools/rulecat into catalog.go), plus the hand-added RuleXMLWellFormed
// sentinel, which is deliberately outside the catalog because XML
// well-formedness faults have no spec-defined rule ID. It lives here rather
// than in the generated catalog.go so the sentinel exemption survives
// `go generate`.
func IsValidRule(r Rule) bool {
	if r == RuleXMLWellFormed {
		return true
	}
	_, ok := ruleCatalog[r]
	return ok
}
