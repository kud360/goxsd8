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
// IT CHARGES FOURTEEN SHAPES, each of which fails clause 1 or both arms of clause 2;
// they are not every shape clause 2 proves. Clause 2 is a disjunction — 2.1's
// literal BNF or 2.2's "XPath expression involving the child axis whose
// abbreviated form is as given above" — and 2.2 is read syntactically: the
// unabbreviated spellings it admits are the ones XPath 2.0 §3.2.4's
// abbreviations reduce to 2.1's BNF, which for a selector is a `child::` head
// before a NameTest and for a field also an `attribute::` head before its final
// NameTest. Those compile as their abbreviated twins do. What is charged is a
// fault no spelling excuses:
//
//   - an unbound prefix, which is clause 1's xpath-valid (§3.13.6.2) static
//     error under either arm;
//   - a field's '@' with no NodeTest after it, and an axis head with none
//     after it (`child::`, `attribute::`), each no XPath 2.0 expression at all
//     and so failing xpath-valid's own clause 1;
//   - on the same clause-1 terms, an empty union member — the empty
//     {expression}, or an operand missing beside a `|` as in `| a` — which
//     XPath 2.0's Expr and UnionExpr never derive;
//   - a `/` or `//` with no Step after it — at the end of a Path, or before
//     another `/` or `//` as in `a//`, `.//`, `./ /.` and `a////b` — save a `/`
//     that is the whole Path, which XPath 2.0's RelativePathExpr never derives
//     and so fails clause 1 too;
//   - a name split by white space around its `:` (`tid :*`, `tid : x`, or
//     `child: :` for `child::`), which no QName, Wildcard or axis spells and so
//     fails clause 1 too, whatever its prefix would resolve to;
//   - a predicate, which abbreviation neither introduces nor removes;
//   - an attribute named anywhere in a selector, under either spelling, which
//     clause 2.2 does not name for a selector at all;
//   - an attribute step before a field's final step, which production [7]
//     admits only as a Path's final step;
//   - an axis head clause 2.2 does not name — any of XPath 2.0's thirteen but
//     child, and for a field attribute — which clause 2.1's tokens do not spell
//     either, `self::node()` included: XPath 2.0 states no abbreviation of it
//     to `.`;
//   - a root-relative path, opened by `/` or `//`, which production [2] and
//     [7]'s context-relative Path never is;
//   - a `//` anywhere but the leading `.//` pair, which §3.2.4 rule 3 expands
//     to `descendant-or-self::node()`, an axis clause 2.2 does not name;
//   - a KindTest step, `child::node()` and `attribute::node()` included, which
//     production [3]'s Step (`'.' | NameTest`) does not spell and no
//     abbreviation turns into a NameTest; and
//   - a FunctionCall such as `document("")`, which XPath 2.0 makes a
//     FilterExpr and no axis step, so no Step of production [3] either. It is
//     read only with an unprefixed name that is no reserved function name and
//     StringLiteral arguments alone, which is XPath 2.0 whatever function
//     signatures the static context holds.
//
// Everything else is nil here and left to [CompileSelector] to decline at
// validate time. That covers, above all, an {expression} this package cannot
// read whole: a KindTest with an argument (`element(a)`), a FunctionCall with a
// prefixed name or an argument other than a StringLiteral (`p:f('x')`, `f(a)`),
// a predicate holding any other token (`a[1]`), or a Wildcard `*:a`, whatever
// clause 1 or 2 says of it. It also covers two Steps with no separator between
// them, which this package cannot tell from the parent step `..`, and a `.//`
// path whose Steps are all `.`, which production [3]'s bare `.` derives
// outright.
//
// The result is an *[xsderr.Error] carrying the SCC as its rule, with the
// clause it breaks in the message. For the unbound prefix it wraps a cause
// carrying err:XPST0081, which a consumer reads with [xsderr.RuleOf] after one
// errors.Unwrap; the other shapes wrap nothing.
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
// [SelectorViolation] states about which shapes are charged holds here
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
	// 2.0 outside the subset that no charged shape covers, or a subset shape the
	// matcher cannot represent. It is the fail-open direction (PRINCIPLES 20) and
	// never a schema fault.
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
// The clause-1 shapes — a field's bare '@', an axis head with no NodeTest, an
// empty union member, a separator with no Step after it and a split name —
// break an XPath 2.0 production instead. Their messages name the production, and
// they wrap no err:XPST0003, because clause 1 delegates to the grammar —
// xpath-valid's clause 1 is "a valid XPath 2.0 expression" — while err:XPST0003
// is the vocabulary of the static errors its clause 2 excludes.
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
