package validate

import (
	"context"
	"fmt"
	"log/slog"
)

// Assess walks root's subtree once — the element, then its [[attributes]],
// then its [[children]] in document order, recursively — and reports what
// the walk found. The walk topology is cvc-assess-elt's (§3.3.4.6, ·strictly
// assessed·); the decisions taken over it are not.
//
// No cvc- rule is decided yet, so the [Result] is empty. That is a scaffold,
// not a fail-open: nothing is being let through, because nothing is judged
// and the Result claims nothing about the document. It carries no GAP(
// marker for that reason (STYLE P3) — the marker tracks a construct whose
// verdict is withheld, and there are no verdicts here to withhold.
//
// It panics if root is nil, on the same grounds as [ElementChild].
func (v *Validator) Assess(root Element) *Result {
	if root == nil {
		panic("validate: Assess: nil root Element")
	}
	w := walk{log: v.log}
	w.element(root)
	return &w.res
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
