package xsderr

import (
	"errors"
	"fmt"
)

// Rule is a spec rule ID, such as "cvc-complex-type", "cos-st-restricts", or
// "derivation-ok-restriction". Exactly one Rule identifies each Error. It is
// usually a VALIDATION rule; where the spec names a check but states no rule
// for it, the definition anchor naming that check is the ID instead
// (key-cta-ta-select, validate/cta.go). Every Rule constructed in this module
// is expected to be present in the generated catalog (see IsValidRule).
type Rule string

// RuleXMLWellFormed is the sentinel Rule for XML well-formedness faults
// (unbound namespace prefix, mismatched or unclosed tag, malformed XML) that
// are not XSD schema- or instance-validity violations and so have no
// spec-defined cvc-*/src-*/cos-*/sic-* rule ID. It lets such an Error carry a
// recognizable, non-empty Rule instead of "" (which would be indistinguishable
// from a caller that simply forgot to set one). IsValidRule accepts it as a
// documented, non-spec exemption from the generated catalog.
const RuleXMLWellFormed Rule = "xml-wf"

// RuleComponentInvariant is the sentinel Rule for a component constructor
// rejecting a state the spec never names as a numbered clause because the XML
// mapping layer above it makes the state unrepresentable — an absent (zero)
// QName in a slot that exists only to carry a present reference, for example.
// Such a rejection is a representation-invariant fault of the caller, not a
// schema- or instance-validity verdict, so citing a cvc-*/src-*/*-props-correct
// rule would misclassify it for RuleOf/errors.As consumers. Like
// RuleXMLWellFormed it lets the Error carry a recognizable, non-empty Rule
// instead of "", and IsValidRule accepts it as a documented, non-spec exemption
// from the generated catalog.
//
// UNREACHABILITY IS THE PRECONDITION, and it is the standing convention for
// every future guard (#343): a rejection may cite RuleComponentInvariant only
// when the state it rejects is genuinely unreachable from a schema document —
// when the XML mapping layer above the constructor cannot produce it. If a human
// could type the schema document that produces the state, the producer owes it a
// real spec rule first, charged at the construct the author wrote, and the
// constructor guard is then a BACKSTOP for the programmatic caller rather than
// the primary charge. Reaching this sentinel from a document is therefore a
// defect in the producer, not a verdict about the document: it tells a
// RuleOf/errors.As consumer that the library built something illegal, which is
// the wrong thing to say about an author's mistake.
const RuleComponentInvariant Rule = "component-invariant"

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
// tools/rulecat into catalog.go), plus the two hand-added non-spec sentinels,
// RuleXMLWellFormed and RuleComponentInvariant, which are deliberately outside
// the catalog because the faults they name have no spec-defined rule ID. It
// lives here rather than in the generated catalog.go so the sentinel exemptions
// survive `go generate`.
//
// Membership is exact: no prefix-trimming, no clause-suffix (dotted) leniency.
// The generated catalog is keyed on the base rule IDs tools/rulecat extracts
// from the specs ("cvc-complex-type", "cvc-elt"), plus the two hand-listed
// sentinels above — not on clause-level spellings. So both
// IsValidRule("cvc-complex-type.2.1") and IsValidRule("cvc-elt.1") are false,
// and stay false. That costs nothing: no non-test construction site in the
// module cites a dotted rule ID, because the convention here is one coarse rule
// ID plus the clause number in prose (see xsd/complexextension.go). And the
// "cvc-elt.1" fixtures in error_test.go are incidental, not a deliberate claim
// of validity — those tests exercise Error()/Wrap() rendering and nil handling,
// where any string would serve identically. Should a call site ever genuinely
// need to cite a clause-level rule, extend tools/rulecat to emit it rather than
// weakening this check.
func IsValidRule(r Rule) bool {
	if r == RuleXMLWellFormed || r == RuleComponentInvariant {
		return true
	}
	_, ok := ruleCatalog[r]
	return ok
}
