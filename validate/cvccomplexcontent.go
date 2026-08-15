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

// contentCheck is the state of one element's [[children]] assessment against
// its ·governing type definition·'s {content type}. It is built per element by
// [walk.contentCheck] and dropped when that element's [[children]] are
// exhausted.
//
// A nil governing decides NOTHING: every child is walked and none is charged or
// passed, which is the state every element below the validation root is in
// (§3.3.4.6 threads a ·context-determined declaration· into the descent and
// this package does not). A nil matcher beside a non-nil governing is not that
// state: it is clause 1.4 alone declining — an {open content} or a shape
// [xsd.Schema.ContentMatcher] does not decide — while clauses 1.1 to 1.3 still
// hold, since they read the {variety} and not the particle.
//
// initial gathers the ·initial value· clause 1.2 tests, and is written for a
// simple {content type} alone: no other clause reads a character information
// item [[child]] beyond the one it is charging, so gathering under any other
// {variety} would hold the whole of an element's text for nothing.
type contentCheck struct {
	e         Element
	governing *xsd.ComplexType
	matcher   *xsd.Matcher
	initial   strings.Builder
	charged   bool
}

// contentCheck builds the check for one element's [[children]].
//
// GAP(xsd): an element carrying xsi:nil decides nothing at all. cvc-complex-type
// clause 1 applies only where E is not ·nilled·, and ·nilled· (§3.3.4.3
// cvc-elt clause 3.2) needs the {nillable} of the ·governing element
// declaration· and the ·actual value· of the attribute, neither of which this
// dispatch has; presence of the attribute is the upper bound on it, exactly as
// hasInstanceType is for xsi:type. The withheld value's consumer set is
// Result.violations and its one reader Result.Violations, both of which carry
// violations PRESENT, so withholding costs a rejection and manufactures none
// (#716).
func (w *walk) contentCheck(e Element, governing *xsd.ComplexType) *contentCheck {
	if governing == nil || hasInstanceNil(e) {
		return &contentCheck{e: e}
	}
	c := &contentCheck{e: e, governing: governing}
	if m, ok := w.schema.ContentMatcher(*governing); ok {
		c.matcher = m
	}
	return c
}

// text settles clauses 1.1 and 1.3 for one run of character information items,
// and gathers the run into the ·initial value· clause 1.2 tests at the end.
//
// A run of NO characters is no character information item at all, so it
// reaches neither clause: both quantify over the items in [[children]], and an
// adapter that reports an empty run for <e></e> has reported a run, not an
// item. It contributes nothing to the ·initial value· either, that being a
// concatenation.
func (c *contentCheck) text(w *walk, t Text) {
	if c.governing == nil || c.charged || t.Data() == "" {
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
		// its own and none is charged here.
		c.initial.WriteString(t.Data())
	case xsd.ContentMixed:
		// A mixed {content type} restricts character content in no clause at
		// all.
	}
}

// element settles clauses 1.1, 1.2 and 1.4 for one element information item of
// the sequence.
func (c *contentCheck) element(w *walk, child Element) {
	if c.governing == nil || c.charged {
		return
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
		c.match(w, child)
	}
}

// match advances the content model over one element information item (clause
// 1.4), charging cvc-complex-content against the item's OWN location where no
// particle live at that position admits it.
func (c *contentCheck) match(w *walk, child Element) {
	if c.matcher == nil {
		c.log(w, child.Name(), child.Loc(), ruleCvcComplexContent, "1", "declined")
		return
	}
	if a, ok := c.matcher.Next(child.Name()); ok {
		if w.log.Enabled(context.Background(), slog.LevelDebug) {
			c.log(w, child.Name(), child.Loc(), ruleCvcComplexContent, "1", "attributed to "+attributedTo(a))
		}
		return
	}
	c.charge(w, ruleCvcComplexContent, "1", child.Loc(),
		"the element information item %s is ·attributed to· no particle of the {content type} of %s at its position in the [[children]], so the sequence is not ·valid· with respect to that {content type} as cvc-complex-content clause 1 requires (Element Sequence Accepted (Particle), §3.9.4.3)",
		child.Name(), c.e.Name())
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
func (c *contentCheck) end(w *walk) {
	if c.governing == nil || c.charged {
		return
	}
	switch ct := c.governing.ContentType().(type) {
	case xsd.SimpleContent:
		c.initialValue(w, ct.SimpleType)
	case xsd.ElementContent:
		c.sequenceEnd(w)
	case xsd.EmptyContent:
		// Settled child by child; see above.
	}
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
// assesses the element itself. The declaration is not among what a content
// check is given, so charging the empty ·initial value· here would reject every
// empty element its declaration supplies a default for (#716).
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
// (STYLE L1). A check with no ·governing type definition· settles nothing and
// logs nothing — the walk's own "assessing element"/"assessing text" lines
// already record the visit.
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
// seals (STYLE T2's closed-sum exception). It is the whole of what this
// package reads off an Attribution today; the ·context-determined declaration·
// a substituting item carries is §3.3.4.6's, and arrives with the descent that
// threads it (#716).
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

// hasInstanceNil reports whether e carries an xsi:nil attribute (§2.7.1). Like
// hasInstanceType it is a presence test and not an evaluation: ·nilled·
// (§3.3.4.3) additionally needs the declaration's {nillable} and the
// attribute's ·actual value·.
func hasInstanceNil(e Element) bool {
	for _, a := range e.Attributes() {
		if a.Name() == (xsd.QName{Space: xsd.XMLSchemaInstanceNS, Local: "nil"}) {
			return true
		}
	}
	return false
}

// isXMLWhitespace reports whether every character of s is white space as XML
// 1.1 defines it (S ::= (#x20 | #x9 | #xD | #xA)+), which is the set clause 1.3
// allows an element-only {content type} to carry.
func isXMLWhitespace(s string) bool {
	return strings.TrimLeft(s, " \t\r\n") == ""
}
