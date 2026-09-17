package icpath

import (
	"fmt"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleSelector is Selector Value OK (Structures §3.11.6.2, c-selector-xpath),
// the Schema Component Constraint on an identity-constraint definition's
// {selector}. The clause charged goes in the message: the catalog carries the
// bare name, so "c-selector-xpath.2.1" is not a valid [xsderr.Rule].
const ruleSelector xsderr.Rule = "c-selector-xpath"

// ruleFields is Fields Value OK (Structures §3.11.6.3, c-fields-xpaths), the
// same constraint over each member of an identity-constraint definition's
// {fields}.
const ruleFields xsderr.Rule = "c-fields-xpaths"

// ruleXPST0081 is the XPath static error an unbound namespace prefix is
// (xpath20.md Appendix G: "It is a static error if a QName used in an expression
// contains a namespace prefix that cannot be expanded into a namespace URI by
// using the statically known namespaces"), carried as the [xsderr.Rule] of the
// cause the two Violation façades wrap under the SCC.
const ruleXPST0081 xsderr.Rule = "err:XPST0081"

// SelectorViolation reports the c-selector-xpath (§3.11.6.2) violation the
// {selector} of an identity-constraint definition carries, positioned at loc,
// and nil for every other {expression} — including one this package merely
// cannot read.
//
// It is the assembler's entry point, answered before any instance exists: a
// Schema Component Constraint is decided when the component is assembled. loc is
// the position of the <selector> element the {expression} was written on, which
// this package cannot know and never reconstructs (STYLE E3).
//
// THE FOUR SHAPES IT CHARGES ARE THE ONLY ONES EITHER CLAUSE PROVES. Clause 2 is
// a disjunction — 2.1's literal BNF or 2.2's "XPath expression involving the
// child axis whose abbreviated form is as given above" — so an {expression} that
// fails 2.1 may still satisfy 2.2, and a recognizer for 2.1 alone cannot tell
// the two apart. What it CAN prove is a fault no spelling excuses: an unbound
// prefix, which is clause 1's xpath-valid (§3.13.6.2) static error under either
// arm; a predicate, which abbreviation neither introduces nor removes; and an
// attribute step, which clause 2.2 does not name for a selector at all and
// production [7] admits for a field only as a Path's final step. Everything else
// — above all an {expression} written in unabbreviated axis syntax, and a `.//`
// path whose Steps are all `.`, which production [3]'s bare `.` derives outright
// — is nil here and left to [CompileSelector] to decline at validate time.
//
// The result is an *[xsderr.Error] carrying the SCC as its rule. For the unbound
// prefix it wraps a cause carrying err:XPST0081, which a consumer reads with
// [xsderr.RuleOf] after one errors.Unwrap; the other three have no second
// vocabulary and wrap nothing.
func SelectorViolation(loc xsderr.Loc, x xsd.XPathExpression) error {
	return violationAt(loc, x, false)
}

// FieldViolation reports the c-fields-xpaths (§3.11.6.3) violation ONE member of
// an identity-constraint definition's {fields} carries, positioned at loc, and
// nil for every other {expression}.
//
// §3.11.6.3 clause 2 quantifies over the members one at a time ("For each member
// of the {fields}"), so this answers about one member and a caller charging a
// whole {fields} calls it per member, in document order. Everything
// [SelectorViolation] states about which shapes are provable holds here
// unchanged, with production [7] in production [2]'s place.
func FieldViolation(loc xsderr.Loc, x xsd.XPathExpression) error {
	return violationAt(loc, x, true)
}

// violationAt is the body both Violation façades are (STYLE T4). They are two
// named entry points rather than one taking a kind, because selector-or-field is
// a call-site choice and never a stored property: a kind parameter would add a
// representable-invalid zero value for nothing (STYLE T1).
func violationAt(loc xsderr.Loc, x xsd.XPathExpression, field bool) error {
	_, d := compile(x, field)
	if d.kind != defectViolation {
		// An untyped nil, never a nil *xsderr.Error in an error interface: a
		// caller's `!= nil` must mean what it says.
		return nil
	}
	charge := xsderr.New(sccOf(field), loc, "%s", d.why)
	if d.cause != nil {
		charge.Err = d.cause
	}
	return charge
}

// defect is what compile found wrong with an {expression}. The three states are
// sealed inside this package and none of them crosses the boundary: each
// exported entry point answers about ONE of them, so no caller ever classifies a
// defect and no caller can mistake a decline for a verdict.
//
// The zero value is "nothing wrong", which is the only state that yields a
// matchable [Expr].
type defect struct {
	kind defectKind
	// why is the whole message of the charge a defectViolation carries: the
	// {expression}, its fault, and the clause of the SCC that admits no such
	// thing (STYLE E4). The façade adds the rule and the position and nothing
	// else. The other kinds leave it empty because they have nothing to say: an
	// {expression} outside what this package reads is not a verdict about the
	// schema.
	why string
	// cause is the DELEGATED verdict under why, for the one shape with a
	// vocabulary of its own — see unboundViolation. The shapeFault violations
	// leave it nil.
	cause *xsderr.Error
}

// defectKind is the kind of a [defect].
type defectKind byte

const (
	// noDefect is a complete production of the subset, with every prefix
	// resolved.
	noDefect defectKind = iota
	// defectUnsupported is an {expression} this package cannot read — legal XPath
	// 2.0 outside the subset, an unabbreviated spelling clause 2.2 admits, or a
	// subset shape the matcher cannot represent. It is the fail-open direction
	// (PRINCIPLES 20) and never a schema fault.
	defectUnsupported
	// defectViolation is an {expression} that breaches c-selector-xpath or
	// c-fields-xpaths whichever arm of clause 2 it was written under.
	defectViolation
)

// sccOf is the Schema Component Constraint over one kind of {expression}:
// §3.11.6.2 governs the {selector}, §3.11.6.3 each member of {fields}.
func sccOf(field bool) xsderr.Rule {
	if field {
		return ruleFields
	}
	return ruleSelector
}

// subject names the property an {expression} belongs to, as the charge reports
// it.
func subject(field bool) string {
	if field {
		return "{fields} member"
	}
	return "{selector}"
}

// shapeViolation is the verdict a shapeFault proves. It carries no cause:
// productions [2], [3] and [7] are the SCC's own text and no other vocabulary
// states them, so a wrapped layer would put the same rule on both (STYLE E2).
func shapeViolation(x xsd.XPathExpression, field bool, fault string) defect {
	return defect{
		kind: defectViolation,
		why:  fmt.Sprintf("the %s %q %s", subject(field), x.Expression(), fault),
	}
}

// unboundViolation is the clause-1 verdict an unbound prefix is: clause 1 asks
// the {expression} to satisfy xpath-valid (§3.13.6.2), whose clause 2 is "X does
// not produce any static error", and a prefix the statically known namespaces —
// clause 2.2.2's, which are the record's {namespace bindings} — cannot expand is
// one.
//
// It is the ONE shape here with a vocabulary of its own, so it travels as two
// layers on parser/produce_typetable.go's terms: the XPath code is the wrapped
// cause, which errors.Unwrap reaches and [xsderr.RuleOf] then reads
// err:XPST0081 off, instead of a consumer scraping the message for it. It
// renders into the message as well, for a reader holding only the string. The
// cause's [xsderr.Loc] is the zero one, which renders as a `?`: this package
// reads no schema document and so has no position to report — the real one is
// the façade's parameter.
func unboundViolation(x xsd.XPathExpression, field bool, prefix string) defect {
	cause := xsderr.New(ruleXPST0081, xsderr.Loc{},
		"no statically-known namespace binding for prefix %q", prefix)
	return defect{
		kind: defectViolation,
		why: fmt.Sprintf("the %s %q has an XPath static error (%s), but %s clause 1 requires it to satisfy xpath-valid, whose clause 2 admits none",
			subject(field), x.Expression(), cause, sccOf(field)),
		cause: cause,
	}
}
