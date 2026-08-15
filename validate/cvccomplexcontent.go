package validate

import (
	"context"
	"log/slog"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// ruleCvcComplexContent is Element Sequence Locally Valid (Complex Content)
// (Structures §3.4.4.3, cvc-complex-content). Its clause 1 — the whole rule
// where {open content} is ·absent· — sends the sequence to cvc-particle
// (§3.9.4.2), which [xsd.Matcher] decides; the clause charged goes in the
// message on ruleCvcElt's terms, since the catalog carries the bare name.
const ruleCvcComplexContent xsderr.Rule = "cvc-complex-content"

// This file decides cvc-complex-type (§3.4.4.2) clause 1 over one element's
// [[children]] — the CONTENT half of the rule whose attribute half (clauses 2
// to 4) is assess.go's. Its four sub-clauses are one dispatch on {content
// type}.{variety}, and this file is that dispatch:
//
//   - 1.1, empty: no character AND no element information item [[children]],
//     white space included. Empty is STRICTER than element-only (PRINCIPLES
//     13), and clause 1.3's white-space allowance is the whole of the
//     difference — 1.1 states no exception of any kind.
//   - 1.2, simple: no element information item [[children]], AND the ·initial
//     value· — the [[character code]] of each character information item
//     [[child]], concatenated in order (Glossary) — ·valid· with respect to the
//     {simple type definition} per String Valid (§3.16.4). The first half is
//     settled per child, the second only once the [[children]] are exhausted.
//
//     GAP(validate): String Valid clause 3, "every ·ENTITY value· in V is a
//     ·declared entity name·", is not checked for an ·initial value· any more
//     than it is for an attribute's; cvcattribute.go's file comment states the
//     withheld property and the direction of the fail-open for both (#773).
//   - 1.3, element-only: no character information item [[children]] other than
//     XML 1.1 white space.
//   - 1.4, element-only or mixed: the sequence of element information items,
//     taken in order, is ·valid· per cvc-complex-content (§3.4.4.3), which
//     [xsd.Schema.ContentMatcher] decides one child at a time. Mixed restricts
//     character content not at all: clause 1.4 is the only clause it reaches.
//
// One element carries AT MOST ONE content charge. Every clause above says the
// same thing about the element once charged — it is not locally ·valid· with
// respect to its ·governing type definition·, so cvc-type clause 3.2 fails and
// its [validity] is invalid — and the first charge is the one at the offending
// position, which is what a reader needs. Continuing to match after a rejected
// child would report the rest of the sequence against a position the document
// never reached.
//
// Two cvc-elt (§3.3.4.3) clauses read the same [[children]] and are settled
// here rather than in cvcelt.go, because arriving children are what decide
// them:
//
//   - 3.2.3.1, for a ·nilled· element: no character or element information item
//     [[children]] at all. It REPLACES clause 1 rather than joining it — clause
//     1 applies "if E is not ·nilled·" — so the two are never both live, and it
//     shares the one-charge rule above for the same reason.
//   - 5.2.2, for an element whose ·governing element declaration· carries a
//     fixed {value constraint}: no element information item [[children]]
//     (5.2.2.1) and an ·initial value· agreeing with the constraint (5.2.2.2).
//     It joins clause 1 rather than replacing it, and is settled once the
//     [[children]] are exhausted (fixedValue).

// contentCheck is the state of one element's [[children]] assessment against
// its ·governing type definition·'s {content type}. It is built per element by
// [walk.contentCheck] and dropped when that element's [[children]] are
// exhausted.
//
// A nil governing decides no clause 1 at all: every child is walked and none is
// charged or passed by it, and none is ·attributed to· anything either, so the
// whole subtree below such an element is walked against no type in its turn
// ([walk.childGoverning]). The two cvc-elt clauses below still apply — they read
// the DECLARATION and not the type. A nil matcher beside a non-nil governing is
// neither state: it is clause 1.4 alone declining — an {open content} or a shape
// [xsd.Schema.ContentMatcher] does not decide — while clauses 1.1 to 1.3 still
// hold, since they read the {variety} and not the particle.
//
// nilled is whether E is ·nilled· (§3.3.4.3, key-nilled), decided before any
// child arrives ([walk.nilCheck]). It turns clause 1 off wholesale and clause
// 3.2.3.1 on, and it is the state that makes governing nil-or-not insufficient
// on its own: a nilled element with a determinable governing type still runs no
// clause 1.
//
// initial gathers the ·initial value· clauses 1.2 and 5.2.2.2 test, and is
// written only where one of them reads it (gathers): no other clause reads a
// character information item [[child]] beyond the one it is charging, so
// gathering elsewhere would hold the whole of an element's text for nothing.
//
// sawElement and sawText record which kinds of [[children]] arrived, which is
// what cvc-elt clause 5's case split and clause 5.2.2.1 quantify over. They are
// not derivable from initial, which is gathered conditionally and holds nothing
// for the elements whose text no clause reads.
type contentCheck struct {
	e          Element
	g          governance
	nilled     bool
	governing  *xsd.ComplexType
	matcher    *xsd.Matcher
	initial    strings.Builder
	sawElement bool
	sawText    bool
	charged    bool
}

// contentCheck builds the check for one element's [[children]]. governing is
// non-nil only where cvc-complex-type clause 1 is live: a determinable complex
// ·governing type definition· AND an element that is not ·nilled·, clause 1
// applying only "if E is not ·nilled·".
func (w *walk) contentCheck(e Element, g governance, nilled bool) *contentCheck {
	c := &contentCheck{e: e, g: g, nilled: nilled}
	if nilled {
		return c
	}
	governing := g.complexType()
	if governing == nil {
		return c
	}
	c.governing = governing
	if m, ok := w.schema.ContentMatcher(*governing); ok {
		c.matcher = m
	}
	return c
}

// gathers reports whether any charge reads this element's ·initial value·:
// cvc-complex-type clause 1.2's String Valid, over a simple {content type}, or
// cvc-elt clause 5.2.2.2's agreement with a fixed {value constraint}. A ·nilled·
// element gathers nothing — clause 1 is off and clause 5.2.2 is gated on "E is
// not ·nilled·" — and neither does one whose text no clause would read.
func (c *contentCheck) gathers() bool {
	if c.nilled {
		return false
	}
	if _, fixed := elementFixed(c.g); fixed {
		return true
	}
	return c.governing != nil && c.governing.ContentType().Variety() == xsd.ContentSimple
}

// text settles clauses 1.1 and 1.3 for one run of character information items —
// or cvc-elt clause 3.2.3.1, for a ·nilled· element, which admits no character
// [[child]] at all — and gathers the run into the ·initial value· clauses 1.2
// and 5.2.2.2 test at the end.
//
// A run of NO characters is no character information item at all, so it
// reaches neither clause: both quantify over the items in [[children]], and an
// adapter that reports an empty run for <e></e> has reported a run, not an
// item. It contributes nothing to the ·initial value· either, that being a
// concatenation, and it is not the character [[child]] clause 5's case split
// turns on.
func (c *contentCheck) text(w *walk, t Text) {
	if c.charged || t.Data() == "" {
		return
	}
	if c.nilled {
		c.charge(w, ruleCvcElt, "3.2.3.1", t.Loc(),
			"the element %s has xsi:nil = true, so it is ·nilled·, but it has a character information item [[child]], and cvc-elt clause 3.2.3.1 admits no character or element information item [[children]] on a ·nilled· element",
			c.e.Name())
		return
	}
	c.sawText = true
	if c.gathers() {
		c.initial.WriteString(t.Data())
	}
	if c.governing == nil {
		return
	}
	switch c.governing.ContentType().Variety() {
	case xsd.ContentEmpty:
		c.charge(w, ruleCvcComplexType, "1.1", t.Loc(),
			"the element %s has a character information item [[child]], but cvc-complex-type clause 1.1 admits no character or element information item [[children]] at all where the {content type}.{variety} of the ·governing type definition· is empty — white space included, which clause 1.3 allows for element-only and clause 1.1 allows for nothing",
			c.e.Name())
	case xsd.ContentElementOnly:
		if isXMLWhitespace(t.Data()) {
			return
		}
		c.charge(w, ruleCvcComplexType, "1.3", t.Loc(),
			"the element %s has a character information item [[child]] that is not white space, but cvc-complex-type clause 1.3 admits white space alone where the {content type}.{variety} of the ·governing type definition· is element-only",
			c.e.Name())
	case xsd.ContentSimple:
		// Clause 1.2 tests the runs CONCATENATED, so no run decides anything on
		// its own and none is charged here; gathers has already retained it.
	case xsd.ContentMixed:
		// A mixed {content type} restricts character content in no clause at
		// all.
	}
}

// element settles clauses 1.1, 1.2 and 1.4 for one element information item of
// the sequence — or cvc-elt clause 3.2.3.1, for a ·nilled· element — and reports
// what clause 1.4 ·attributed· the item to (§3.4.4.4) for the walk's own descent
// into it (cvc-assess-elt clause 3.1, [walk.childGoverning]).
//
// The attribution is nil wherever nothing attributed the item: a check with no
// ·governing type definition·, a ·nilled· element, an element already charged, a
// {variety} that admits no element information item [[child]] at all, and a
// clause 1.4 that declined or charged. A nilled element's child is still WALKED
// — the charge is against the parent, not the child, and the child's own
// subtree is assessed against nothing rather than skipped.
func (c *contentCheck) element(w *walk, child Element) xsd.Attribution {
	if c.charged {
		return nil
	}
	if c.nilled {
		c.charge(w, ruleCvcElt, "3.2.3.1", child.Loc(),
			"the element %s has xsi:nil = true, so it is ·nilled·, but it has the element information item %s among its [[children]], and cvc-elt clause 3.2.3.1 admits no character or element information item [[children]] on a ·nilled· element",
			c.e.Name(), child.Name())
		return nil
	}
	c.sawElement = true
	if c.governing == nil {
		return nil
	}
	switch c.governing.ContentType().Variety() {
	case xsd.ContentEmpty:
		c.charge(w, ruleCvcComplexType, "1.1", child.Loc(),
			"the element %s has the element information item %s among its [[children]], but cvc-complex-type clause 1.1 admits no character or element information item [[children]] where the {content type}.{variety} of the ·governing type definition· is empty",
			c.e.Name(), child.Name())
	case xsd.ContentSimple:
		c.charge(w, ruleCvcComplexType, "1.2", child.Loc(),
			"the element %s has the element information item %s among its [[children]], but cvc-complex-type clause 1.2 admits no element information item [[children]] where the {content type}.{variety} of the ·governing type definition· is simple",
			c.e.Name(), child.Name())
	case xsd.ContentElementOnly, xsd.ContentMixed:
		return c.match(w, child)
	}
	return nil
}

// match advances the content model over one element information item (clause
// 1.4), reporting what the item is ·attributed to· and charging
// cvc-complex-content against the item's OWN location where no particle live at
// that position admits it.
func (c *contentCheck) match(w *walk, child Element) xsd.Attribution {
	if c.matcher == nil {
		c.log(w, child.Name(), child.Loc(), ruleCvcComplexContent, "1", "declined")
		return nil
	}
	if a, ok := c.matcher.Next(child.Name()); ok {
		if w.log.Enabled(context.Background(), slog.LevelDebug) {
			c.log(w, child.Name(), child.Loc(), ruleCvcComplexContent, "1", "attributed to "+attributedTo(a))
		}
		return a
	}
	c.charge(w, ruleCvcComplexContent, "1", child.Loc(),
		"the element information item %s is ·attributed to· no particle of the {content type} of %s at its position in the [[children]], so the sequence is not ·valid· with respect to that {content type} as cvc-complex-content clause 1 requires (Element Sequence Accepted (Particle), §3.9.4.3)",
		child.Name(), c.e.Name())
	return nil
}

// end settles the two clauses that only exhausted [[children]] can settle,
// dispatching on the {content type} itself rather than on its {variety} (STYLE
// T2's closed-sum exception) because one of them reads a property inside it:
//
//   - simple: clause 1.2's ·initial value· half, over the runs text gathered.
//   - element-only and mixed: clause 1.4, for a sequence that ran out with a
//     particle its {min occurs} left open.
//
// An empty {content type} settles nothing here. Clause 1.1 quantifies over the
// [[children]] PRESENT, every one of which was already decided as it arrived.
//
// cvc-elt clause 5.2.2 is settled after them, over the same exhausted
// [[children]], and for an element with no ·governing type definition· too — it
// reads the DECLARATION's {value constraint}, which a type this package could
// not determine does not withhold.
func (c *contentCheck) end(w *walk) {
	if c.charged {
		return
	}
	if c.governing != nil {
		switch ct := c.governing.ContentType().(type) {
		case xsd.SimpleContent:
			c.initialValue(w, ct.SimpleType)
		case xsd.ElementContent:
			c.sequenceEnd(w)
		case xsd.EmptyContent:
			// Settled child by child; see above.
		}
	}
	c.fixedValue(w)
}

// fixedValue settles cvc-elt clause 5.2.2: where D.{value constraint}.{variety}
// is fixed and E is not ·nilled·, E has no element information item [[children]]
// (5.2.2.1) and its ·initial value· agrees with the constraint (5.2.2.2).
//
// The rule is only reached through clause 5's case split, and this function
// applies that split rather than clause 5.2.2 alone: clause 5.2 holds when D has
// no {value constraint}, or E has element or character [[children]], or E is
// ·nilled·. With a fixed constraint present and E not nilled, the first and
// third disjuncts are false, so clause 5.2 — and with it 5.2.2 — is live for
// exactly the elements that HAVE [[children]]. An empty element with a fixed
// constraint takes clause 5.1's arm instead, which is the decline initialValue
// states.
//
// Clause 5.2.2.2's two cases are read off the ·governing type definition·: a
// mixed complex type compares LEXICALS (5.2.2.2.1, "matches"), and a simple type
// or simple {content type} compares ·actual values· (5.2.2.2.2, "equal or
// identical"), which is value.ConstraintMatches over the same pipeline the
// attribute charges use. A governing type of any other shape — element-only,
// empty, or none determined — matches no case and clause 5.2.2.2 is vacuous.
//
// A content charge already made silences this one: clause 5.2.2 asks whether the
// ·initial value· AGREES with the fixed value, and an ·initial value· cvc-type
// has just rejected has no agreement to report that its own charge does not
// already say.
func (c *contentCheck) fixedValue(w *walk) {
	if c.charged {
		return
	}
	f, fixed := elementFixed(c.g)
	if !fixed || c.nilled {
		return
	}
	if !c.sawElement && !c.sawText {
		return // clause 5.1's arm; see initialValue
	}
	if c.sawElement {
		c.charge(w, ruleCvcElt, "5.2.2.1", c.e.Loc(),
			"the element %s has element information item [[children]], but its ·governing element declaration· carries the fixed {value constraint} %q, and cvc-elt clause 5.2.2.1 admits no element information item [[children]] on an element so constrained",
			c.e.Name(), f.LexicalForm())
		return
	}
	if ct := c.governing; ct != nil && ct.ContentType().Variety() == xsd.ContentMixed {
		c.fixedLexical(w, f)
		return
	}
	c.fixedActualValue(w, f)
}

// fixedLexical settles clause 5.2.2.2.1: for a mixed {content type}, the
// ·initial value· of E matches D.{value constraint}.{lexical form}. The match is
// over LEXICALS and not values — a mixed type has no {simple type definition} to
// map either side through — so it is string equality and nothing else.
func (c *contentCheck) fixedLexical(w *walk, f xsd.ValueConstraint) {
	if c.initial.String() == f.LexicalForm() {
		c.log(w, c.e.Name(), c.e.Loc(), ruleCvcElt, "5.2.2.2.1", "satisfied")
		return
	}
	c.charge(w, ruleCvcElt, "5.2.2.2.1", c.e.Loc(),
		"the ·initial value· of the element %s does not match the {lexical form} %q of the fixed {value constraint} of its ·governing element declaration·, which cvc-elt clause 5.2.2.2.1 requires of an element whose ·governing type definition· has {content type}.{variety} = mixed",
		c.e.Name(), f.LexicalForm())
}

// fixedActualValue settles clause 5.2.2.2.2: for a simple ·governing type
// definition· — a Simple Type Definition, or a complex one with a simple
// {content type} — the ·actual value· of E is equal or identical to
// D.{value constraint}.{value}.
//
// A governing type that is neither leaves clause 5.2.2.2 with no applicable case
// and charges nothing. An undecided comparison charges nothing either, on
// [walk.fixedAgreement]'s terms and for the same reasons: an ungoverned type or a
// {lexical form} outside its own type's lexical space is a gap in this processor
// or a schema fault cos-valid-default charges at assembly, not the instance's.
func (c *contentCheck) fixedActualValue(w *walk, f xsd.ValueConstraint) {
	st := c.g.simpleType()
	if st == nil {
		return
	}
	same, decided := value.ConstraintMatches(w.backend, w.schema, st, c.initial.String(), elementContext{owner: c.e}, f)
	if !decided {
		c.log(w, c.e.Name(), c.e.Loc(), ruleCvcElt, "5.2.2.2.2", "declined")
		return
	}
	if same {
		c.log(w, c.e.Name(), c.e.Loc(), ruleCvcElt, "5.2.2.2.2", "satisfied")
		return
	}
	c.charge(w, ruleCvcElt, "5.2.2.2.2", c.e.Loc(),
		"the ·actual value· of the element %s is neither equal nor identical to the {value} of the fixed {value constraint} %q of its ·governing element declaration·, which cvc-elt clause 5.2.2.2.2 requires",
		c.e.Name(), f.LexicalForm())
}

// initialValue settles the ·initial value· half of clause 1.2: the string
// composed, in order, of the [[character code]] of each character information
// item in E.[[children]] (Glossary, ·initial value·) is ·valid· with respect to
// T.{content type}.{simple type definition} as String Valid (§3.16.4) defines
// it. The charge carries the CONTAINING element's location: the value is
// assembled from every run and belongs to none of them.
//
// st needs no resolution, unlike the {type definition} the attribute charges
// reach through — [xsd.SimpleContent] carries the component itself and
// [xsd.NewComplexType] rejects a nil one (ct-props-correct clause 1) — so the
// two declines below are the whole of what this charge withholds.
//
// GAP(validate): an element with NO character information item [[child]] is
// declined, because cvc-elt clause 5 dispatches BEFORE cvc-type reaches this
// rule and its clause 5.1 may replace the item validated. Where the
// ·governing element declaration· has a {value constraint} and the element is
// not ·nilled·, what is assessed is "the element information item with
// D.{value constraint}.{lexical form} used as its ·normalized value·", whose
// ·initial value· is that lexical and never the empty string; only clause 5.2
// assesses the element itself. Clause 5.1's own two conjuncts are unimplemented
// — 5.1.1 is Element Default Valid (Immediate) (§3.3.6.2) over an
// ·instance-specified· governing type, and 5.1.2 re-enters cvc-type with a
// substituted ·normalized value· — so charging the empty ·initial value· here
// would reject every empty element its declaration supplies a default for.
// fixedValue below declines the same shape for clause 5.2.2, which clause 5.1
// takes the arm of.
//
// GAP(validate): a ValidateLexical error that is not a VERDICT is the same
// fail-open cvcattribute.go's matchedAttribute states in full, over the same
// [value.IsDatatypeVerdict] classification: an ungoverned {simple type
// definition} reports under cvc-datatype-valid exactly as a genuine rejection
// does, and charging it would reject every element whose simple content this
// backend cannot read (#774).
func (c *contentCheck) initialValue(w *walk, st *xsd.SimpleType) {
	if c.initial.Len() == 0 {
		c.log(w, c.e.Name(), c.e.Loc(), ruleCvcComplexType, "1.2", "declined")
		return
	}
	if _, err := value.ValidateLexical(w.backend, w.schema, st, c.initial.String(), elementContext{owner: c.e}); err != nil {
		if !value.IsDatatypeVerdict(err) {
			c.log(w, c.e.Name(), c.e.Loc(), ruleCvcComplexType, "1.2", "declined")
			return
		}
		c.charge(w, ruleCvcComplexType, "1.2", c.e.Loc(),
			"the ·initial value· of the element %s is not ·valid· with respect to the {simple type definition} %s of its ·governing type definition·'s {content type}, which cvc-complex-type clause 1.2 requires as per String Valid (§3.16.4): %v",
			c.e.Name(), st.Name(), err)
		return
	}
	c.log(w, c.e.Name(), c.e.Loc(), ruleCvcComplexType, "1.2", "satisfied")
}

// sequenceEnd settles clause 1.4 for a sequence that ran out: every item was
// ·attributed to· a particle, but the particles left open cannot all be closed.
// The charge carries the CONTAINING element's location, there being no child at
// the offending position to carry one.
func (c *contentCheck) sequenceEnd(w *walk) {
	if c.matcher == nil {
		return
	}
	if c.matcher.Accepting() {
		c.log(w, c.e.Name(), c.e.Loc(), ruleCvcComplexContent, "1", "accepted")
		return
	}
	c.charge(w, ruleCvcComplexContent, "1", c.e.Loc(),
		"the [[children]] of %s end before every particle of the {content type} of its ·governing type definition· has taken the occurrences its {min occurs} requires, so the sequence is not ·valid· with respect to that {content type} as cvc-complex-content clause 1 requires (Element Sequence Accepted (Particle), §3.9.4.3)",
		c.e.Name())
}

// charge records one violation and closes the element to any further content
// charge (see the file comment).
func (c *contentCheck) charge(w *walk, rule xsderr.Rule, clause string, loc xsderr.Loc, format string, args ...any) {
	w.res.violations = append(w.res.violations, xsderr.New(rule, loc, format, args...))
	c.charged = true
	c.log(w, c.e.Name(), loc, rule, clause, "charged")
}

// log records one content decision: which rule and clause settled it, and how
// (STYLE L1). A check that settles nothing logs nothing — the walk's own
// "assessing element"/"assessing text" lines already record the visit.
func (c *contentCheck) log(w *walk, name xsd.QName, loc xsderr.Loc, rule xsderr.Rule, clause, outcome string) {
	if !w.log.Enabled(context.Background(), slog.LevelDebug) {
		return
	}
	w.log.LogAttrs(context.Background(), slog.LevelDebug, "assessing content",
		slog.Any("name", name), slog.Any("loc", loc), slog.String("rule", string(rule)),
		slog.String("clause", clause), slog.String("outcome", outcome))
}

// attributedTo names the particle {term} an item was ·attributed to·
// (§3.4.4.4) for the log, switching over the two variants [xsd.Attribution]
// seals (STYLE T2's closed-sum exception). What the descent reads off the same
// Attribution — the child's ·governing element declaration·, including the
// ·substituting declaration· a substituting item carries rather than the
// particle's own — is [walk.childGoverning]'s, and the two are deliberately
// separate: a log line names the particle that consumed the item, which is a
// fact about the PARENT's content model.
func attributedTo(a xsd.Attribution) string {
	switch t := a.(type) {
	case xsd.ElementDeclaration:
		return "element declaration " + t.Name().String()
	case xsd.Wildcard:
		return "wildcard " + t.NamespaceConstraint().Variety().String()
	default:
		return "particle"
	}
}

// xmlWhitespace is white space as XML 1.1 defines it (S ::= (#x20 | #x9 | #xD
// | #xA)+), the set clause 1.3 allows an element-only {content type} to carry
// and the one whiteSpace = collapse removes (collapseXMLWhitespace,
// cvcelt.go). One encoding serves both (STYLE T4).
const xmlWhitespace = " \t\r\n"

// isXMLWhitespace reports whether every character of s is xmlWhitespace.
func isXMLWhitespace(s string) bool {
	return strings.TrimLeft(s, xmlWhitespace) == ""
}
