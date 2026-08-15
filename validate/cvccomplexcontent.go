package validate

import (
	"context"
	"log/slog"
	"strings"

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
//   - 1.2, simple: no element information item [[children]]. Its other half,
//     the ·initial value· against the {simple type definition}, is not decided
//     here (cvc-simple-type, §3.16.4, over an assembled ·initial value·).
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
type contentCheck struct {
	e         Element
	governing *xsd.ComplexType
	matcher   *xsd.Matcher
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

// text settles clauses 1.1 and 1.3 for one run of character information items.
//
// A run of NO characters is no character information item at all, so it
// reaches neither clause: both quantify over the items in [[children]], and an
// adapter that reports an empty run for <e></e> has reported a run, not an
// item.
func (c *contentCheck) text(w *walk, t Text) {
	if c.governing == nil || c.charged || t.Data() == "" {
		return
	}
	switch c.governing.ContentType().Variety() {
	case xsd.ContentEmpty:
		c.charge(w, ruleCvcComplexType, "1.1", t.Loc(),
			"clause 1.1: the element %s has a character information item [[child]], but the {content type}.{variety} of its ·governing type definition· is empty, which admits no character or element information item [[children]] at all — white space included, which clause 1.3 allows for element-only and clause 1.1 allows for nothing",
			c.e.Name())
	case xsd.ContentElementOnly:
		if isXMLWhitespace(t.Data()) {
			return
		}
		c.charge(w, ruleCvcComplexType, "1.3", t.Loc(),
			"clause 1.3: the element %s has a character information item [[child]] that is not white space, but the {content type}.{variety} of its ·governing type definition· is element-only",
			c.e.Name())
	case xsd.ContentSimple, xsd.ContentMixed:
		// A mixed {content type} restricts character content in no clause at
		// all, and clause 1.2's character half is the ·initial value· test,
		// which this package does not assemble.
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
			"clause 1.1: the element %s has the element information item %s among its [[children]], but the {content type}.{variety} of its ·governing type definition· is empty, which admits no character or element information item [[children]]",
			c.e.Name(), child.Name())
	case xsd.ContentSimple:
		c.charge(w, ruleCvcComplexType, "1.2", child.Loc(),
			"clause 1.2: the element %s has the element information item %s among its [[children]], but the {content type}.{variety} of its ·governing type definition· is simple, which admits no element information item [[children]]",
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
		"clause 1: the element information item %s is ·attributed to· no particle of the {content type} of %s at its position in the [[children]], so the sequence is not ·valid· with respect to that {content type} (Element Sequence Accepted (Particle), §3.9.4.3)",
		child.Name(), c.e.Name())
}

// end settles clause 1.4 for a sequence that ran out: every item was
// ·attributed to· a particle, but the particles left open cannot all be closed.
// The charge carries the CONTAINING element's location, there being no child at
// the offending position to carry one.
func (c *contentCheck) end(w *walk) {
	if c.governing == nil || c.charged || c.matcher == nil {
		return
	}
	if c.matcher.Accepting() {
		c.log(w, c.e.Name(), c.e.Loc(), ruleCvcComplexContent, "1", "accepted")
		return
	}
	c.charge(w, ruleCvcComplexContent, "1", c.e.Loc(),
		"clause 1: the [[children]] of %s end before every particle of the {content type} of its ·governing type definition· has taken the occurrences its {min occurs} requires, so the sequence is not ·valid· with respect to that {content type} (Element Sequence Accepted (Particle), §3.9.4.3)",
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
