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
// cvc-elt). The clause charged goes in the message, not the rule ID: the
// generated catalog is keyed on base rule IDs and admits no dotted
// spelling (see [xsderr.IsValidRule]).
const ruleCvcElt xsderr.Rule = "cvc-elt"

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
// Nothing else is decided: the remaining cvc-elt clauses and cvc-type
// (§3.3.4.4) are not evaluated, so a [Result] carrying no violation says
// the root is declared and not abstract, and says nothing else about the
// document.
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
	// GAP(xpath): where a declaration was found, its ·selected type
	// definition· (§3.3.4.1/.2) is never computed, because selecting among
	// a {type table}'s <alternative>s means evaluating each one's test as
	// an XPath expression. The ·governing type definition· it feeds is
	// left undetermined, so cvc-elt clause 5 and cvc-type (§3.3.4.4) are
	// not reached and no verdict about the root's type is withheld or
	// claimed here (#56).
	w.element(root)
	return &w.res
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
func (w *walk) element(e Element) {
	if w.log.Enabled(context.Background(), slog.LevelDebug) {
		w.log.Debug("assessing element", slog.Any("name", e.Name()), slog.Any("loc", e.Loc()))
	}
	for _, a := range e.Attributes() {
		w.attribute(a)
	}
	w.children(e)
}

// attribute assesses one attribute information item. Under the silent
// default the guard leaves it with no body, which is what a walk that
// decides no cvc- rule yet amounts to at an attribute — the cvc-attribute
// (§3.2.4.1) decisions land here, so do not inline it away.
func (w *walk) attribute(a Attribute) {
	if !w.log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	w.log.Debug("assessing attribute", slog.Any("name", a.Name()), slog.Any("loc", a.Loc()))
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
		w.element(e)
		return
	}
	t, ok := c.Text()
	if !ok {
		panic("validate: walk.child: Child holds neither arm; build one with ElementChild or TextChild")
	}
	w.text(t)
}
