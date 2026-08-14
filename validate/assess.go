package validate

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCvcAssessElt is Schema-Validity Assessment (Element) (Structures
// §3.3.4.6, cvc-assess-elt). Its three clauses only dispatch — strictly
// assessed, not assessed, laxly assessed — so a ·validation root· that
// determines no declaration and no type definition violates no numbered
// clause of it. The charge stands instead on §5.2 strict wildcard
// validation, whose Note has the invoking process expecting the
// ·validation root· "to be declared and valid" and otherwise reporting "an
// error to its environment"; cvc-assess-elt is the rule that policy is
// stated against, and the catalog carries the bare name.
const ruleCvcAssessElt xsderr.Rule = "cvc-assess-elt"

// ruleCvcElt is Element Locally Valid (Element) (Structures §3.3.4.3,
// cvc-elt). The clause charged goes in the message, not the rule ID: unlike
// some rules whose dotted sub-ID the spec itself anchors (cos-ct-extends.1.2,
// src-simple-type.1), cvc-elt's catalog entry is the bare name only, so
// "cvc-elt.2" is not a valid [xsderr.Rule] (see [xsderr.IsValidRule]).
const ruleCvcElt xsderr.Rule = "cvc-elt"

// ruleCvcComplexType is Element Locally Valid (Complex Type) (Structures
// §3.4.4.2, cvc-complex-type). Clauses 2 and 3 — the attribute half — are
// charged here; the clause number goes in the message on ruleCvcElt's terms.
const ruleCvcComplexType xsderr.Rule = "cvc-complex-type"

// Assess walks root's subtree once — the element, then its [[attributes]],
// then its [[children]] in document order, recursively — and reports what
// the walk found. The walk topology is cvc-assess-elt's (§3.3.4.6, ·strictly
// assessed·).
//
// It decides the ·governing element declaration· of the validation root,
// which is the declaration root's ·expanded name· ·resolves· to among the
// schema's top-level element declarations (§3.3.4.6, ·governing element
// declaration· clause 4), and charges two rules over that dispatch:
//
//   - No such declaration and no xsi:type, so no ·governing type
//     definition· can exist either and cvc-assess-elt clause 1 cannot
//     apply: cvc-assess-elt is charged and the subtree is not walked.
//     Clause 3 would have the root ·laxly assessed· against xs:anyType,
//     which this package does not implement; charging instead is §5.2
//     strict wildcard validation.
//   - A declaration whose {abstract} is true: cvc-elt clause 2 is charged
//     and the walk still runs. ·Strictly assessed· clauses 2 and 3 assess
//     [[attributes]] and [[children]] whatever clause 1.1.2's evaluation
//     returned, so an abstract root does not silence its subtree.
//
// cvc-elt clause 1 (D ·non-absent·, E and D sharing an ·expanded name·) is
// satisfied by construction — D is the declaration found BY that expanded
// name — and so is never charged.
//
// Where the root's ·governing type definition· is determinable and complex
// (see rootComplexType), the root's own [[attributes]] are additionally
// assessed against that type's {attribute uses}: cvc-complex-type (§3.4.4.2)
// clauses 2 and 3, the half of that rule no datatype backend is needed for
// (see [walk.attributes]). Nothing below the root is: a DESCENDANT's
// governing type definition comes from matching it against a content model
// (cvc-complex-content, §3.4.4.3), which this package does not do, so its
// attributes are assessed against no type at all.
//
// Nothing else is decided: the remaining cvc-elt clauses, the rest of
// cvc-type (§3.3.4.4) and cvc-complex-type's own clauses 1 and 4-6 are not
// evaluated, so a [Result] carrying no violation says the root is declared,
// not abstract, and — where its type was determinable — carries no attribute
// clause 2 rejects and no required attribute clause 3 misses, and says
// nothing else about the document.
//
// It panics if root is nil, on the same grounds as [ElementChild].
func (v *Validator) Assess(root Element) *Result {
	if root == nil {
		panic("validate: Assess: nil root Element")
	}
	w := walk{log: v.log}
	// GAP(xsd): an xsi:type here is DETECTED, never ·resolved·, so the
	// charge below is withheld for a root carrying one that a full
	// cvc-resolve-instance (§3.17.6.3) would find unresolvable and charge
	// anyway. The direction is fail-open across the whole consumer set of
	// the withheld value, which is Result.violations and its one reader
	// Result.Violations: both carry violations PRESENT, so withholding one
	// can only cost a rejection and never manufacture one (#716).
	d, found := v.Schema().Element(root.Name())
	if !found && !hasInstanceType(root) {
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcAssessElt, root.Loc(),
			"the validation root %s has no top-level element declaration and no xsi:type, so it determines neither a ·governing element declaration· nor a ·governing type definition· and cannot be ·strictly assessed· (§5.2 strict wildcard validation)",
			root.Name()))
		return &w.res
	}
	if found && d.Abstract() {
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcElt, root.Loc(),
			"clause 2: the validation root %s is governed by an element declaration whose {abstract} is true, and an abstract declaration validates no element information item",
			root.Name()))
	}
	w.element(root, v.rootComplexType(root, d, found))
	return &w.res
}

// rootComplexType is the ·governing type definition· (§3.3.4.6) of the
// validation root, narrowed to the Complex Type Definition cvc-type clause
// 3.2 dispatches to cvc-complex-type for; it is nil wherever this package
// cannot determine that type, and the root's attributes are then assessed
// against nothing.
//
// Determinable here means the whole of key-governing-type-elem's clause 4
// case: no processor-stipulated type (this package stipulates none), no
// ·instance-specified type definition· overriding it, and a ·selected type
// definition· (§3.3.4.1) that is D.{type definition} outright. Each of the
// three declines below withholds a type that could differ from the
// declaration's, and charging the root's attributes against the WRONG type
// is a false reject in both directions — an attribute the real governing
// type declares looks unmatched, one it forbids looks fine.
//
//   - GAP(xsd): an xsi:type makes the ·instance-specified type definition·
//     the governing one (clause 3) when it ·overrides· the selected type,
//     and ·resolving· it is cvc-resolve-instance (§3.17.6.3), unimplemented.
//     Presence alone is the decline (#716).
//   - GAP(xpath): a {type table} makes the selected type the one its
//     <alternative>s ·conditionally select· (§3.3.4.2), which means
//     evaluating each {test} as an XPath expression (#56).
//   - A {type definition} slot that resolves to nothing, or to a Simple Type
//     Definition. The simple case is not an omission: cvc-type dispatches to
//     cvc-complex-type only for a complex T, and a simple-typed element's own
//     attributes are governed by cvc-type clause 3.1.1 instead, which this
//     package does not decide either.
func (v *Validator) rootComplexType(root Element, d xsd.ElementDeclaration, found bool) *xsd.ComplexType {
	if !found || hasInstanceType(root) {
		return nil
	}
	if _, tabled := d.TypeTable(); tabled {
		return nil
	}
	t, ok := v.schema.ResolvedType(d.TypeDefinition())
	if !ok {
		return nil
	}
	ct, isComplex := t.(xsd.ComplexType)
	if !isComplex {
		return nil
	}
	return &ct
}

// hasInstanceType reports whether e carries an xsi:type attribute (§2.7.1).
// It is a presence test and not a ·resolution·: establishing the
// ·instance-specified type definition· that §5.2 counts as one of the
// ·validation root·'s three determinants needs cvc-resolve-instance
// (§3.17.6.3), and presence is the upper bound on it available short of
// that.
func hasInstanceType(e Element) bool {
	for _, a := range e.Attributes() {
		if a.Name() == (xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "type"}) {
			return true
		}
	}
	return false
}

// walk is the state of one [Validator.Assess] call, held here and not on the
// Validator so nothing survives the call that made it.
type walk struct {
	log *slog.Logger
	res Result
}

// element assesses one element information item: the item itself, then its
// [[attributes]], then its [[children]].
//
// governing is e's ·governing type definition· narrowed to a complex type,
// or nil where none was determined. It is NOT propagated to [walk.children]:
// a child's governing type follows from the particle it is ·attributed to·
// in its parent's {content type} (§3.4.4.4), which nothing here computes, so
// every element below the validation root is assessed against nil.
func (w *walk) element(e Element, governing *xsd.ComplexType) {
	if w.log.Enabled(context.Background(), slog.LevelDebug) {
		w.log.Debug("assessing element", slog.Any("name", e.Name()), slog.Any("loc", e.Loc()))
	}
	w.attributes(e, governing)
	w.children(e)
}

// attributes assesses E.[[attributes]] against governing, in the two
// directions cvc-complex-type (§3.4.4.2) quantifies in: clause 2 over the
// attribute information items PRESENT, in source order, then clause 3 over
// the {attribute uses} that must be present. Violations reach [Result] in
// that order, which is the order they were found in.
//
// A nil governing decides nothing at all: every attribute is walked (the
// log records the visit) and none is charged or passed.
func (w *walk) attributes(e Element, governing *xsd.ComplexType) {
	attrs := e.Attributes()
	for _, a := range attrs {
		w.attribute(a, governing)
	}
	if governing == nil {
		return
	}
	w.requiredAttributeUses(e, attrs, *governing)
}

// attribute assesses one attribute information item against clause 2, whose
// two arms are the whole rule: an attribute matching an attribute use is
// judged by cvc-au (clause 2.1), one matching none needs an {attribute
// wildcard} that admits it (clause 2.2). There is no third arm, so an
// attribute that matches neither violates clause 2 outright — and that is
// decidable with no datatype backend wherever no {attribute wildcard} is
// present at all.
func (w *walk) attribute(a Attribute, governing *xsd.ComplexType) {
	if governing == nil {
		w.logAttribute(a, "", "", "")
		return
	}
	if isInstanceAttribute(a.Name()) {
		// Clause 2 excepts xsi:type, xsi:nil, xsi:schemaLocation and
		// xsi:noNamespaceSchemaLocation (§3.2.7) from its quantifier by name,
		// so no arm of it applies to one: it is neither charged nor passed.
		w.logAttribute(a, ruleCvcComplexType, "2", "exempt")
		return
	}
	u, matched := attributeUseNamed(governing.AttributeUses(), a.Name())
	if !matched {
		w.unmatchedAttribute(a, *governing)
		return
	}
	if vc, has := u.ValueConstraint(); has && vc.Kind() == xsd.ValueFixed {
		// GAP(validate): clause 2.1 sends a matched attribute to cvc-au
		// (§3.5.4), which for a use carrying a fixed {value constraint}
		// compares ·actual values· — the value space, where "1" and "01" are
		// one xs:integer. That needs the attribute's ·normalized value· mapped
		// through its {type definition}, which this package cannot do, and a
		// lexical comparison in its place would reject documents the spec
		// accepts. The attribute is left undecided (#766).
		w.logAttribute(a, ruleCvcComplexType, "2.1", "declined")
		return
	}
	// cvc-au is vacuously true for a use with no fixed {value constraint} —
	// it constrains nothing else — so clause 2.1 is satisfied outright.
	w.logAttribute(a, ruleCvcComplexType, "2.1", "satisfied")
}

// unmatchedAttribute settles clause 2 for an attribute matching no attribute
// use, where only clause 2.2 is left to satisfy.
//
// GAP(xsd): this is the one charge in the package that a property REPORTED
// TOO SMALL could fabricate, so it is the one that has to name the exposure.
// [xsd.ComplexType.AttributeUses] and [xsd.ComplexType.AttributeWildcard]
// both run their §3.4.2.4/§3.4.2.5 fold over a finalized schema's TYPE
// DEFINITIONS only, so an ANONYMOUS type owned by an element declaration
// reports its own uses and its own <anyAttribute> alone (#414). Under-report
// the uses and an inherited attribute looks unmatched; under-report the
// wildcard and the decline above does not fire — together they would charge
// an attribute the base admits. It does not happen for a schema this module
// parses: the producer's inline form is the IMPLICIT-CONTENT one, a
// restriction of xs:anyType, whose {attribute uses} §3.4.7 makes empty and
// whose {attribute wildcard} clause 2.1 takes from the type itself, so the
// unrun fold is the identity there. A caller who assembles an inline
// EXTENSION through [xsd.SchemaBuilder] is outside that bound until #414
// widens the fold.
func (w *walk) unmatchedAttribute(a Attribute, governing xsd.ComplexType) {
	if _, wild := governing.AttributeWildcard(); wild {
		// GAP(validate): clause 2.2.1 holds, but clause 2.2 is a conjunction
		// and 2.2.2 sends the attribute to cvc-wildcard (§3.10.4.1), which
		// this package does not evaluate. Charging on 2.2.1 alone would reject
		// every attribute a wildcard admits, so clause 2 is left undecided for
		// an element whose type carries one (#717).
		w.logAttribute(a, ruleCvcComplexType, "2.2", "declined")
		return
	}
	w.res.violations = append(w.res.violations, xsderr.New(ruleCvcComplexType, a.Loc(),
		"clause 2: the attribute %s matches no attribute use among the {attribute uses} of its element's ·governing type definition· (clause 2.1), which has no {attribute wildcard} for it to be ·valid· with respect to either (clause 2.2.1)",
		a.Name()))
	w.logAttribute(a, ruleCvcComplexType, "2", "charged")
}

// requiredAttributeUses charges clause 3 for each {required} attribute use
// of governing that no attribute information item of e carries the
// ·expanded name· of. It is an existence test over names and nothing more:
// whether a matching item is itself ·valid· is clause 2's question, and an
// OPTIONAL use with no matching item is silent here whatever its ·effective
// value constraint· says.
//
// attrs is e.[[attributes]] as [walk.attributes] read it, so the two clauses
// quantify over one reading of the source. The scan is NOT filtered by
// isInstanceAttribute: clause 2's exception is clause 2's alone, and its own
// Note keeps a use of xsi:type satisfying clause 3 while being ·attributed
// to· nothing.
func (w *walk) requiredAttributeUses(e Element, attrs []Attribute, governing xsd.ComplexType) {
	for _, u := range governing.AttributeUses() {
		if !u.Required() {
			continue
		}
		name := u.DeclarationName()
		if hasAttributeNamed(attrs, name) {
			continue
		}
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcComplexType, e.Loc(),
			"clause 3: the element %s carries no attribute information item named %s, but its ·governing type definition· has an attribute use for that name whose {required} is true",
			e.Name(), name))
		if w.log.Enabled(context.Background(), slog.LevelDebug) {
			w.log.Debug("assessing attribute use", slog.Any("name", name), slog.Any("loc", e.Loc()),
				slog.String("rule", string(ruleCvcComplexType)), slog.String("clause", "3"),
				slog.String("outcome", "charged"))
		}
	}
}

// logAttribute records one attribute information item's assessment: its
// ·expanded name· and location always, and the rule and clause that settled
// it wherever clause 2 settled anything (STYLE L1). An attribute assessed
// against no ·governing type definition· has no rule to name and its line
// carries none — the walk visited it and decided nothing.
func (w *walk) logAttribute(a Attribute, rule xsderr.Rule, clause, outcome string) {
	if !w.log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	attrs := []slog.Attr{slog.Any("name", a.Name()), slog.Any("loc", a.Loc())}
	if rule != "" {
		attrs = append(attrs, slog.String("rule", string(rule)),
			slog.String("clause", clause), slog.String("outcome", outcome))
	}
	w.log.LogAttrs(context.Background(), slog.LevelDebug, "assessing attribute", attrs...)
}

// attributeUseNamed reports the attribute use among uses whose {attribute
// declaration} has the ·expanded name· n, as clause 2.1 matches. The name is
// read off the use itself ([xsd.AttributeUse.DeclarationName]) and never off
// a resolved declaration: the match is by name, and a resolution that failed
// for some unrelated reason must not turn a declared attribute into an
// unmatched one.
func attributeUseNamed(uses []xsd.AttributeUse, n xsd.QName) (xsd.AttributeUse, bool) {
	for _, u := range uses {
		if u.DeclarationName() == n {
			return u, true
		}
	}
	return xsd.AttributeUse{}, false
}

// hasAttributeNamed reports whether attrs holds an attribute information
// item whose ·expanded name· is n, as clause 3 asks.
func hasAttributeNamed(attrs []Attribute, n xsd.QName) bool {
	for _, a := range attrs {
		if a.Name() == n {
			return true
		}
	}
	return false
}

// isInstanceAttribute reports whether n is one of the four attributes
// cvc-complex-type clause 2 excepts by name — xsi:type, xsi:nil,
// xsi:schemaLocation, xsi:noNamespaceSchemaLocation, the Built-in Attribute
// Declarations of §3.2.7, which §3.2.6 a-props-correct forbids a schema to
// redeclare.
func isInstanceAttribute(n xsd.QName) bool {
	if n.Space != xsd.XMLSchemaInstanceNS {
		return false
	}
	switch n.Local {
	case "type", "nil", "schemaLocation", "noNamespaceSchemaLocation":
		return true
	}
	return false
}

// text assesses one run of character information items. It reports the run's
// length rather than its content: instance data does not belong in a log.
// Under the silent default the guard leaves it with no body, on the same
// terms as attribute — the ·initial value· this run contributes to is
// assembled here once cvc-type clause 3.1.3 arrives.
func (w *walk) text(t Text) {
	if !w.log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	w.log.Debug("assessing text", slog.Int("chars", len(t.Data())), slog.Any("loc", t.Loc()))
}

// children pulls e's [[children]] in document order and assesses each. It
// stops at the first fault — one the cursor reports, or one raised deeper in
// the subtree — and keeps it: a later child assessed after a fault would be
// assessed out of a context the source never finished delivering.
func (w *walk) children(e Element) {
	kids := e.Children()
	for {
		c, ok := kids.Next()
		if !ok {
			break
		}
		w.child(c)
		if w.res.err != nil {
			return
		}
	}
	if err := kids.Err(); err != nil {
		w.res.err = fmt.Errorf("reading the children of %s at %s: %w", e.Name(), e.Loc(), err)
	}
}

// child assesses one child, whichever arm it holds. A Child holding neither
// is an adapter bug, not a fault in the source, so it panics rather than
// reaching [Result.Err] — that field means the walk stopped on a source
// fault, and a CLI would otherwise report a bug in an adapter to a user as
// a broken document.
func (w *walk) child(c Child) {
	if e, ok := c.Element(); ok {
		w.element(e, nil)
		return
	}
	t, ok := c.Text()
	if !ok {
		panic("validate: walk.child: Child holds neither arm; build one with ElementChild or TextChild")
	}
	w.text(t)
}
