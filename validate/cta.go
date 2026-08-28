package validate

import (
	"github.com/kud360/goxsd8/xpath"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file is §3.3.4.1's ·selected type definition· — the half of the
// ·governing type definition· that a {type table} decides — and nothing else.
// The ·overriding· xsi:type read that sits above it stays in assess.go, where
// key-governing-type-elem's own clause order lives.

// ruleKeyCTATASelect is ·successfully selects· (§3.12.4, key-cta-ta-select),
// the check conditionallySelected reaches for each Type Alternative and does
// not perform when the alternative's {test} cannot be compiled. It is a
// [Definition:] anchor and NOT a Validation Rule, because §3.12.4 defines
// conditional type assignment through definitions and gives it no cvc-* rule of
// its own — so this is the only ID the spec text offers for the check, and an
// invented cvc-cta* one would name nothing (#56).
//
// It is a Rule only ever carried by an [Unevaluated], never by an
// [xsderr.Error]: a failed key-cta-ta-select charges no element. The
// alternative simply does not ·successfully select·, and key-cta-select's scan
// moves to the next one or to the {default type definition} — which is why the
// package charges nothing under this ID and must not start.
const ruleKeyCTATASelect xsderr.Rule = "key-cta-ta-select"

// selectedType is the ·selected type definition· S of an element information
// item E whose ·governing element declaration· is d (§3.3.4.1,
// key-selected-type); ok is false wherever this package cannot determine it,
// which is governingType's own decline and carries its consequences unchanged.
//
// The rule has exactly two cases and they are taken in its order: clause 1,
// a declaration WITH a {type table}, whose table ·conditionally selects· S;
// clause 2, one without, whose {type definition} is S outright.
func (w *walk) selectedType(e Element, d xsd.ElementDeclaration) (xsd.TypeDefinition, bool) {
	table, tabled := d.TypeTable()
	if !tabled {
		return w.schema.ResolvedType(d.TypeDefinition())
	}
	return w.conditionallySelected(e, table)
}

// conditionallySelected is the type a Type Table ·conditionally selects· for e
// (§3.3.4.1, key-cta-select): the {alternatives} are tried in document order
// until one ·successfully selects· a type definition (§3.12.4,
// key-cta-ta-select — its {test} evaluates to true), and if none does, the
// {default type definition}'s {type definition} is the answer.
//
// The scan is LAZY in both directions the rule quantifies in. It compiles and
// evaluates one alternative at a time, because "if any Type Alternative
// ·successfully selects· a type definition, none of the following Type
// Alternatives are tried" — so an alternative this engine cannot evaluate,
// sitting BEHIND one whose test is true, never costs e its type.
//
// An alternative this engine cannot evaluate, sitting BEFORE any that
// succeeds, withholds the whole element's ·governing type definition· and
// stops the scan. It does NOT fall through to the next alternative or to the
// {default type definition}: the undecided test might have been true, so
// continuing would select a type the rule may not have selected, and
// assessing an element against the wrong type manufactures a false reject in
// both directions (the charge governingType's own doc rules out). Withholding
// can only cost a rejection.
//
// That withhold is RECORDED, as one [Unevaluated] under ruleKeyCTATASelect at
// the element's own location, and it is the only thing here that is: nothing
// downstream charges the element, so without the record an element whose
// ·governing type definition· was withheld is byte-identical at the [Result]
// API to one that passed every rule (#56).
//
// A DYNAMIC OR TYPE ERROR INSIDE AN EVALUABLE {test} IS NOT RECORDED, because
// it is not a withhold: key-cta-ta-select clause 2 says such a {test} "is
// treated as if it had evaluated (without error) to false", so the alternative
// has been tried and did not select, and the scan continues to the next one or
// to the {default type definition} with a real type in hand. [xpath.CTATest]
// evaluation therefore reports a bool and no error at all, and the two
// outcomes stay apart at the type level rather than by a check here.
//
// w.schema is the xsd.TypeResolver both halves of the engine take, and it is
// passed rather than stored on either side: the compile reads the {type
// definitions} to classify the datatype a {test} casts to, and the evaluation
// reads them again to walk that type's {base type definition} chain.
//
// e-props-correct clause 6, which xsd.NewTypeTable enforces at construction,
// makes every {alternatives} member's {test} present, so the presence flag
// carries nothing here; a table that reached this package without it declines
// anyway, because a zero XPath Expression record has an empty {expression} and
// no {expression} is a Test production.
func (w *walk) conditionallySelected(e Element, table xsd.TypeTable) (xsd.TypeDefinition, bool) {
	attr := ctaAttributes(e)
	alts := table.Alternatives()
	for i, alt := range alts {
		test, _ := alt.Test()
		compiled, evaluable := xpath.CompileCTATest(test, w.schema)
		if !evaluable {
			w.res.unevaluated = append(w.res.unevaluated, newUnevaluated(ruleKeyCTATASelect, e.Loc(),
				"alternative %d of %d in the {alternatives} of the {type table} of %s's ·governing element declaration·, whose {test} is %q, was not evaluated: this engine's §3.12.6 required-subset compiler declined it, which ta-props-correct clause 2 licenses (a conforming processor may but is not required to support XPath outside that subset), so whether it ·successfully selects· its {type definition} is undecided and the type the table ·conditionally selects· (§3.3.4.1, key-cta-select) is withheld along with it, together with every alternative behind it",
				i+1, len(alts), e.Name(), test.Expression()))
			return nil, false
		}
		if !compiled.Evaluate(w.backend, w.schema, attr) {
			continue
		}
		return w.schema.ResolvedType(alt.TypeDefinition())
	}
	return w.schema.ResolvedType(table.DefaultTypeDefinition().TypeDefinition())
}

// ctaAttributes is the attribute sequence a {test} evaluates against: the
// [[attributes]] of e itself, in SOURCE order, which is what §3.12.4 clause
// 1.1.2 copies into the XDM instance the test runs over. The xsi: attributes
// are among them, because clause 1.1.2 copies E's [[attributes]] whole and
// names no exception; the namespace declarations are not, because [Element]'s
// Attributes excludes them — which is what keeps a wildcard NameTest from
// ranging over xmlns bindings as if they were attribute nodes.
//
// The slice is read once, not per walk: [Element]'s Attributes may build it,
// and a {test} naming three attributes would otherwise build it three times.
//
// GAP(xpath): E's {inherited attributes} (§3.12.4 clause 1.1.3 — those whose
// ·expanded names· no attribute of E already has) are NOT merged in, so a
// {test} reading an attribute e does not carry directly sees the empty
// sequence, and a WILDCARD NameTest (`@*`, `@p:*`, `@*:n`) ranges over e's own
// [[attributes]] alone. §3.3.5.6's inheritance mechanism is unimplemented, and
// {inheritable} is not read on global attribute declarations at all (#831),
// which is the precondition for merging correctly.
//
// The DIRECTION of this gap is UNESTABLISHED (STYLE P3a), not fail-open. The
// value withheld is one operand of a general comparison, and its readers are
// the {test}s of conditionallySelected's scan above: a missing operand makes
// that comparison false (xpath20.md §3.5.2 — no pair exists), which makes the
// alternative not ·successfully select·, which hands e the next alternative's
// type or the {default type definition}'s. Both of those are REAL selections,
// and assess.go, cvcelt.go, cvccomplexcontent.go and cvcattribute.go then
// charge e against whichever one arrived — including cvc-type clause 3.1's
// three sub-clauses, which an alternative naming xs:error or a bare simple type
// reaches. Charging an element against a type the rule did not select rejects a
// valid document as readily as it accepts an invalid one, so no direction is
// claimed here.
func ctaAttributes(e Element) xpath.Attributes {
	attrs := e.Attributes()
	return func(yield func(xsd.QName, string) bool) {
		for _, a := range attrs {
			if !yield(a.Name(), a.Value()) {
				return
			}
		}
	}
}
