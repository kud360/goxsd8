package validate

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xpath"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file settles cvc-complex-type (§3.4.4.2) clause 6 — each assertion in
// the {assertions} of a complex ·governing type definition· — by EVALUATING its
// {test} wherever xpath compiles it, and declines the assertions facets of a
// simple type whose {test} the value pipeline could not decide. Both halves
// decline through [walk.decline], so every site this file does not decide is
// recorded and logged at the item it was reached at, and the two
// GAP(validate) markers below state, separately, what each of the two still
// withholds; the GAP(xpath) markers on the string-value collector state which
// descendants of a child of mixed content leave its string-value undecided.
//
// The two rules are DISTINCT and are never conflated. cvc-assertion
// (§3.13.4.1) is the complex-type variety, reached from cvc-complex-type
// (§3.4.4.2) clause 6 and from nowhere else. cvc-assertions-valid (Datatypes
// §4.3.13.3) is the simple-type one, reached from cvc-datatype-valid (§4.1.4)
// clause 3 (dv_vfacets) over the assertions facet of a simple type. One record
// type carries both, discriminated by [Unevaluated.Rule].
//
// The simple-type facets are evaluated where Datatype Valid is decided, inside
// package value's pipeline, through [xpath.FacetAssertions] — on the type and
// on every list item and union member the cvc-datatype-valid recursion
// PRINCIPLES 12 quantifies over reaches, and on no other. A failed {test} is
// part of the verdict the caller charges under its own rule; a declined one is
// a non-verdict, which the recording sites decline under cvc-assertions-valid
// ([walk.declineAssertions]): cvc-attribute clause 3 ([walk.declaredAttribute],
// for an attribute matched by an {attribute use} and for one ·attributed to· a
// strict or lax {attribute wildcard} alike), and cvc-type clause 3.1.3 /
// cvc-complex-type clause 1.2 over an element's ·initial value·
// ([contentCheck.stringValid]), and cvc-complex-type clause 4 over a
// ·defaulted attribute·'s {lexical form} ([walk.defaultedAttribute]). cvcid.go
// and cvcidentityconstraint.go re-run the datatype pipeline over lexicals those
// sites already decided, and record only declines of their own.

// ruleCvcAssertion is Assertion Satisfied (Structures §3.13.4.1,
// cvc-assertion), whose single caller is cvc-complex-type clause 6. The clause
// charged goes in the message on ruleCvcElt's terms: the catalog carries the
// bare name.
const ruleCvcAssertion xsderr.Rule = "cvc-assertion"

// ruleCvcAssertionsValid is Assertions Valid (Datatypes §4.3.13.3,
// cvc-assertions-valid), the assertions facet's own per-facet specification
// under Facet Valid. A simple type's assertions answer to this rule and never
// to ruleCvcAssertion: the two differ in what they bind $value to and in what
// context item they evaluate against, so one ID standing for both would
// misreport every simple-type assertion.
const ruleCvcAssertionsValid xsderr.Rule = "cvc-assertions-valid"

// elementAssertions settles cvc-complex-type (§3.4.4.2) clause 6 for e: "E is
// ·valid· with respect to each of the assertions in T.{assertions} as per
// Assertion Satisfied (§3.13.4.1)", T being e's ·governing type definition·. A
// governing type that is not a Complex Type Definition has no {assertions}
// property at all — a simple one's assertions are facets, and reach
// ruleCvcAssertionsValid instead.
//
// asserts is e's clause 6 state, opened when e was entered ([walk.compileAssertions],
// which compiles every {test}) and nil where T has no {assertions}. Each compiled
// test is evaluated with the binding [walk.assertionValue] gives `.` and `$value`:
// invalid says whether e is known to be invalid in the partial ·PSVI· by now (clause
// 2.3.1.1) — the caller's count of the violations recorded since e was entered — and
// content is e's own content check, exhausted, which holds the ·initial value· and
// the ·nilled· answer.
//
// Each assertion takes exactly one of three outcomes: DECLINED, where
// [xpath.CompileAssertionTest] reported false or e's attributes, the children
// a {test} reads, or `$value` cannot be read; CHARGED under cvc-assertion, where
// [xpath.AssertionTest.Evaluate] reports false — the {test} was false or raised
// a dynamic or type error, which cvc-assertion's opening sentence treats alike
// ("evaluates to true ... without raising any dynamic error or type error");
// and SATISFIED otherwise. The log names no clause: cvc-assertion's verdict is
// that opening sentence, and its numbered clauses only build the evaluation's
// context.
//
// GAP(validate): an assertion this package does not evaluate is DECLINED —
// recorded as an [Unevaluated] under cvc-assertion at e through
// [walk.decline], never charged and never shown satisfied. The residue is: a
// {test} xpath declines, whose GAP(xpath) markers name the grammar and type
// residue (paths, the function library beyond its string and sequence core);
// and every assertion of an e one of whose attributes matching an {attribute
// use}, carried or ·defaulted·, has no ·actual value·, one of whose element
// [[children]] a {test} reads has no typed value this package reads
// ([walk.keepChild]) — a child of mixed content among them whose string-value
// spans a descendant whose contribution is undecided
// ([assertionAncestry.collectElement]) — whose `$value` is undecided
// ([walk.assertionValues]), whose string value was not gathered because it has
// simple content and element [[children]], or is ·nilled· with simple content
// and any [[children]] ([walk.assertionValue]), or one of whose {test}s counts
// the attribute nodes of, or filters children by the attribute names of, an
// element of its subtree whose ·governing type definition· this package could
// not determine, so that its ·defaulted attributes· are unknown
// ([walk.tallyElement]); a ·skipped· subtree is counted, by name
// ([assertionAncestry.tallySkipped]). Fail-open: the withheld value is clause
// 6's own verdict, whose whole consumer set inside this package is
// w.res.violations and its one reader [Result.Violations], which charge on a
// violation PRESENT, so a decline can only cost a rejection and can manufacture
// none. (#1042)
func (w *walk) elementAssertions(e Element, asserts *assertionCheck, content *contentCheck, invalid bool) {
	if asserts == nil {
		return
	}
	attrs := e.Attributes()
	in, lack := w.assertionValues(e, attrs, asserts, content, invalid)
	for i, c := range asserts.tests {
		site := fmt.Sprintf("assertion %d of %d in the {assertions} of the ·governing type definition· %s, whose {test} is %q,",
			i+1, len(asserts.tests), typeName(asserts.ct), c.a.Test().Expression())
		if lack != nil {
			w.decline("assessing element", e.Name(), e.Loc(), ruleCvcAssertion, "",
				"%s was not evaluated: %s, so whether the element is ·valid· with respect to it, as cvc-complex-type clause 6 requires, is undecided",
				site, lack.declined(e.Name()))
			continue
		}
		test := c.test
		if test == nil {
			w.decline("assessing element", e.Name(), e.Loc(), ruleCvcAssertion, "",
				"%s was not evaluated: this engine's XPath evaluator declined it, so whether the element %s is ·valid· with respect to it, as cvc-complex-type clause 6 requires (Assertion Satisfied, §3.13.4.1), is undecided",
				site, e.Name())
			continue
		}
		if test.Evaluate(w.backend, w.schema, in.yield, asserts.yieldChildren, c.tally, in.value) {
			w.logDecision("assessing element", e.Name(), e.Loc(), ruleCvcAssertion, "", "satisfied")
			continue
		}
		w.res.violations = append(w.res.violations, xsderr.New(ruleCvcAssertion, e.Loc(),
			"the element %s is not ·valid· with respect to %s which did not evaluate to true without raising a dynamic or type error, as Assertion Satisfied (§3.13.4.1) requires of each assertion cvc-complex-type clause 6 quantifies over",
			e.Name(), site))
		w.logDecision("assessing element", e.Name(), e.Loc(), ruleCvcAssertion, "", "charged")
	}
}

// assertionCheck is one element's cvc-complex-type clause 6 state, opened when
// the element is entered ([walk.compileAssertions]) and settled once its
// [[children]] are exhausted ([walk.elementAssertions]): the ·governing type
// definition· ct whose {assertions} are evaluated, and each of them with its
// compiled {test}, in {assertions} order.
//
// Between the two it collects, from the element's own [[children]] as each is
// exhausted ([walk.keepChild]), what the compiled tests read of them: children
// holds one [xpath.ChildElement] per element [[child]] some test reads, in
// arrival order, which is document order (STYLE D1); and lack is the first such
// child that has no typed value this package reads, nil while there is none —
// a read child of mixed content taking it as well from a descendant whose
// contribution to its string-value is undecided ([stringValueFrame.lacking]) —
// or the first element of the element's subtree a test's [xpath.Tally] cannot
// be told of exactly: one whose ·governing type definition· was not
// determined, at whose depth some test's count reads attribute names
// ([assertionCheck.countsAttributesAt], [walk.tallyElement]). The lack is the
// whole check's, never one test's. Each test's Tally collects the counts from
// the whole subtree as it is walked — a ·skipped· part of it by name
// ([assertionAncestry.tallySkipped]) — and no node of it is kept.
type assertionCheck struct {
	ct       xsd.ComplexType
	tests    []assertionTest
	children []xpath.ChildElement
	lack     assertionLack
}

// counts reports whether some compiled test of c counts nodes of the element's
// subtree, holding an [xpath.Tally]. A nil c — an element with no {assertions}
// — counts none.
func (c *assertionCheck) counts() bool {
	if c == nil {
		return false
	}
	for _, t := range c.tests {
		if t.tally != nil {
			return true
		}
	}
	return false
}

// countsAttributesAt reports whether the attribute names reported for an
// element depth levels below c's element, 0 being c's element itself, can
// change a count some compiled test of c holds
// ([xpath.Tally.CountsAttributesAt]). A test with no Tally counts none.
func (c *assertionCheck) countsAttributesAt(depth int) bool {
	for _, t := range c.tests {
		if t.tally.CountsAttributesAt(depth) {
			return true
		}
	}
	return false
}

// tally reports one element of c's element's subtree to each test's Tally, in
// one [xpath.Tally.Element] call: by path, the names of the elements from c's
// element's child down to it inclusive, with attrs, the names of its attribute
// nodes. The empty path is c's own element, whose element node selects
// nothing and whose attributes alone count. A test with no Tally takes nothing
// ([xpath.Tally]'s nil receiver).
func (c *assertionCheck) tally(path []xsd.QName, attrs []xsd.QName) {
	for _, t := range c.tests {
		t.tally.Element(path, attrs)
	}
}

// reads reports whether some compiled test of c reads a child element named
// name ([xpath.AssertionTest.ReadsChild]). A nil c — an element with no
// {assertions} — reads none.
func (c *assertionCheck) reads(name xsd.QName) bool {
	if c == nil {
		return false
	}
	for _, t := range c.tests {
		if t.test != nil && t.test.ReadsChild(name) {
			return true
		}
	}
	return false
}

// lacking records l as c's lack where c has none yet, so the first lack the
// walk meets is the one a decline names: a counted node's on entering it
// ([walk.tallyElement]), a read child's once its assessment is over
// ([walk.keepChild]), both in document order.
func (c *assertionCheck) lacking(l assertionLack) {
	if c.lack == nil {
		c.lack = l
	}
}

// assertionAncestry is what the {assertions} of an element's ancestors read of
// it, handed down the walk beside its identity-constraint state: parent, the
// clause 6 state of its parent, nil at the ·validation root· and wherever the
// parent has no {assertions}, which reads its typed value ([walk.keepChild]);
// counting, the nearest ancestor at any depth one of whose {test}s counts nodes
// of its subtree, which is told of the element ([walk.tallyElement]);
// collecting, the nearest ancestor at any depth whose string-value a {test} of
// its own parent reads, which the element's text runs and its own frame's
// text are appended to ([stringValueFrame]); and path, the ·expanded names· of
// the element's ancestors from the ·validation root· down, the root included,
// so len(path) is the element's depth below the root, 0 at the root.
//
// It holds O(depth) frames, one per counting ancestor and one per collecting
// one, O(depth) names, the text of each collecting ancestor's subtree so far,
// and no node.
//
// Every element's path shares its parent's backing array: below appends the
// element's own name in place, so a later sibling's name overwrites an earlier
// one's at the same index, and a descendant's those below it. That is safe
// because the walk is recursive — an element's subtree is done before its next
// sibling's name is written — and because path is read only during
// [walk.tallyElement]'s calls, which [xpath.Tally.Element] retains nothing of.
type assertionAncestry struct {
	parent     *assertionCheck
	counting   *tallyFrame
	collecting *stringValueFrame
	path       []xsd.QName
}

// tallyFrame is one counting ancestor: its clause 6 state, its depth below the
// ·validation root·, and the next counting ancestor above it, nil where there
// is none.
type tallyFrame struct {
	check *assertionCheck
	depth int
	outer *tallyFrame
}

// stringValueFrame is one element [[child]], named name at loc, whose
// string-value (xpath-datamodel §6.2.4) a {test} of its parent reads, whose
// clause 6 state is reader: a child whose ·locally declared type· and own
// ·governing type definition· both have mixed content, and which is not
// ·nilled· ([walk.stringValue]). text collects, bottom-up, the concatenation
// of every Text Node of its subtree in document order: a character run
// arrives in the innermost open frame ([assertionAncestry.collectText]), and a
// frame's text is appended to its outer one's when its element closes
// ([assertionAncestry.closeStringValue]). outer is the next such child
// enclosing it, nil where there is none.
//
// It holds the text of one subtree while that subtree is walked, the only
// data a string-value is made of, and nothing else of it.
type stringValueFrame struct {
	text   strings.Builder
	reader *assertionCheck
	name   xsd.QName
	loc    xsderr.Loc
	outer  *stringValueFrame
}

// lacking declines the reader of f and of every frame enclosing it, for the
// reason why: a descendant whose contribution to the innermost string-value is
// undecided leaves every string-value spanning it undecided too.
func (f *stringValueFrame) lacking(why string) {
	for ; f != nil; f = f.outer {
		f.reader.lacking(lackingChild{name: f.name, loc: f.loc, why: why})
	}
}

// below is the ancestry of each element [[child]] of the element named name
// whose ancestry is up and whose own clause 6 state is own: own is their
// parent, name ends their ancestors' path, the counting ancestors are up's,
// with own's element innermost where some test of own counts, and the
// innermost open string-value frame is frame, the element's own, where it
// opened one, and up's otherwise.
func (up assertionAncestry) below(name xsd.QName, own *assertionCheck, frame *stringValueFrame) assertionAncestry {
	next := assertionAncestry{parent: own, counting: up.counting, collecting: up.collecting, path: append(up.path, name)}
	if own.counts() {
		next.counting = &tallyFrame{check: own, depth: len(up.path), outer: up.counting}
	}
	if frame != nil {
		next.collecting = frame
	}
	return next
}

// stringValue opens the string-value frame of e, an element whose ancestry is
// up, governed by g, ·nilled· where nilled is true, and is nil where it opens
// none: e opens one exactly where some {test} of its parent reads its name, its
// ·locally declared type· within the parent's ·governing type definition· —
// the lookup [walk.assertionElementTypes] answers the compile — has mixed
// content, its own ·governing type definition· does too, and it is not
// ·nilled·. Every other read child's value is [walk.childValue]'s to decide,
// which lacks or reads it with no string-value.
func (w *walk) stringValue(e Element, g governance, nilled bool, up assertionAncestry) *stringValueFrame {
	if nilled || !mixedContent(g.typ) || !up.parent.reads(e.Name()) {
		return nil
	}
	ldt, local := w.schema.LocallyDeclaredElementType(up.parent.ct, e.Name())
	if !local || !mixedContent(ldt) {
		return nil
	}
	return &stringValueFrame{reader: up.parent, name: e.Name(), loc: e.Loc(), outer: up.collecting}
}

// mixedContent reports whether td is a complex type whose {content
// type}.{variety} is mixed — xs:anyType among them (§3.4.7) — whose element's
// typed value is its string-value as xs:untypedAtomic (xpath-datamodel
// §6.2.4).
func mixedContent(td xsd.TypeDefinition) bool {
	ct, isComplex := td.(xsd.ComplexType)
	return isComplex && ct.ContentType().Variety() == xsd.ContentMixed
}

// collectElement settles what e, an element whose ancestry is up, governed by g
// and ·nilled· where nilled is true, contributes to the string-value of every
// open frame enclosing it, before its own [[children]] arrive: a ·nilled· one
// contributes nothing and has no Text Node (xpath-datamodel §6.2.4), and an
// element-only, empty, mixed or ·laxly assessed· one contributes its
// descendants' text, which [assertionAncestry.collectText] reads off each run.
//
// GAP(xpath): an element of simple type or simple content, not ·nilled·,
// declines the reader of every frame enclosing it — xpath-datamodel §6.2.4
// lets its Text Node hold the raw characters or the [schema normalized value]
// (:1259, "may"), so the string-value spanning it is not fixed — and so does an
// element whose ·governing type definition· this package could not determine,
// which may be either. The direction is the withhold: the assertion is
// recorded as unevaluated, never charged and never shown satisfied. (#1042)
func (up assertionAncestry) collectElement(e Element, g governance, nilled bool) {
	if up.collecting == nil || nilled {
		return
	}
	if g.typeUndetermined() {
		up.collecting.lacking(fmt.Sprintf("the element %s at %s below it has a ·governing type definition· that was not determined, so what its text contributes to the string-value is undecided", e.Name(), e.Loc()))
		return
	}
	if g.valueType() != nil {
		up.collecting.lacking(fmt.Sprintf("the element %s at %s below it has simple content, whose Text Node the data model lets hold its raw characters or its [schema normalized value], so the string-value is undecided", e.Name(), e.Loc()))
	}
}

// collectText appends t, one character run [[child]] of the element whose
// content check is content, to the innermost open string-value frame of up,
// by that element's governance, on the terms xpath-datamodel §6.2.4 and §6.7.4
// build its Text Nodes: under mixed content, or ·laxly assessed· and so
// annotated xs:anyType, the raw run; under element-only or empty content
// nothing — a run that is white space alone is no Text Node there, and any
// other run is a violation recorded at that element, which declines its
// reader in [walk.childValue] — and under a ·nilled· element nothing, its run
// a violation too. An element of simple type or simple content contributes
// nothing here, having declined every frame on entry
// ([assertionAncestry.collectElement]).
//
// GAP(xpath): a run of an element whose ·governing type definition· this
// package could not determine declines the reader of every enclosing frame,
// fail-open on [assertionAncestry.collectElement]'s terms. (#1042)
func (up assertionAncestry) collectText(content *contentCheck, t Text) {
	if up.collecting == nil || content.nilled {
		return
	}
	if content.g.typeUndetermined() {
		up.collecting.lacking(fmt.Sprintf("a character run at %s below it is in an element whose ·governing type definition· was not determined", t.Loc()))
		return
	}
	if content.g.laxlyAssessed() || mixedContent(content.g.typ) {
		up.collecting.text.WriteString(t.Data())
	}
}

// closeStringValue settles e's own frame, frame, nil where e opened none, once
// e's [[children]] are exhausted, e being an element whose ancestry is up,
// governed by g, with the content check content: frame's text is appended to
// its outer frame's whatever e's reader goes on to decide, so a string-value
// spanning e holds e's text even where e's own value is lacked.
//
// GAP(xpath): an element of mixed content inside an open frame — e's own or
// one enclosing it — whose ·initial value· cvc-elt clause 5.1 substituted
// from its {value constraint} declines the reader of every enclosing frame:
// xpath-datamodel §6.2.4 (:1265) lets a processor add a Text Node for the
// default, not requiring it, so the string-value spanning it is not fixed.
// (#1042)
func (up assertionAncestry) closeStringValue(e Element, frame *stringValueFrame, g governance, content *contentCheck) {
	inner := up.collecting
	if frame != nil {
		inner = frame
	}
	if inner != nil && mixedContent(g.typ) && !content.nilled {
		if _, defaulted := content.defaulted(); defaulted {
			inner.lacking(fmt.Sprintf("the element %s at %s, of mixed content, took the default of its {value constraint}, which the data model may or may not make a Text Node", e.Name(), e.Loc()))
		}
	}
	if frame != nil && frame.outer != nil {
		frame.outer.text.WriteString(frame.text.String())
	}
}

// skipped hands e, an element [[child]] ·skipped· by key-sva clause 3.2 and
// whose ancestry is up, to the {assertions} of its ancestors: as a lack of its
// parent where some test of the parent reads e's name — a ·skipped· element is
// not ·assessed·, so the partial ·PSVI· gives it no type to read its value
// under — and, where some ancestor counts, as e's subtree told to every
// counting ancestor's Tallies ([assertionAncestry.tallySkipped]). It reports
// the fault in the source that stopped that subtree, and nil where none did;
// where no ancestor counts, nothing below e is read and it reports nil.
//
// GAP(xpath): a ·skipped· e inside an open string-value frame declines the
// reader of every enclosing frame ([stringValueFrame.lacking]) rather than
// reading its text into the string-value, which [assertionAncestry.tallySkipped]
// does not walk for. The direction is the withhold: the assertion is recorded
// as unevaluated, never charged and never shown satisfied. (#1042)
func (up assertionAncestry) skipped(e Element) error {
	if up.parent.reads(e.Name()) {
		up.parent.lacking(lackingChild{name: e.Name(), loc: e.Loc(), why: "it is ·skipped·, so it is not ·assessed·"})
	}
	if up.collecting != nil {
		up.collecting.lacking(fmt.Sprintf("the element %s at %s below it is ·skipped·, so its text is not read into the string-value", e.Name(), e.Loc()))
	}
	if up.counting == nil {
		return nil
	}
	return up.tallySkipped(e)
}

// tallySkipped tells every counting ancestor in up of e, an element of a
// ·skipped· subtree whose ancestry is up, and then of each element below e, in
// document order: each ancestor's Tallies count an element by its chain of
// names below that ancestor, on [walk.tallyElement]'s terms, with the names of
// the attributes the element carries at that chain's length. A ·skipped·
// element and everything below it is governed by no type (key-skipped,
// key-governing-type-elem item 5), so it has no ·defaulted attribute·
// (key-dflt-att), and its attribute nodes are exactly the [[attributes]] it
// carries, xsi ones included ([Element.Attributes]); whether a {test} counts
// attribute nodes at its depth decides nothing here.
//
// It reads names and nothing else: no element below the skip is ·assessed·
// (cvc-assess-elt clause 2), so nothing is fed to an identity constraint or the
// ID table, no [inherited attributes] are handed down, no value is kept for a
// {test} ([walk.keepChild]), and nothing is logged. It opens e's [[children]]
// ([Element.Children]) while the source still stands at e — the caller's
// cursor has not advanced past it — and drains that cursor before it returns,
// as [Children] requires; a text child counts nothing and is passed over. It
// reports the fault in the source that stopped a cursor, wrapped once with the
// element whose [[children]] it was reading, and nil otherwise.
func (up assertionAncestry) tallySkipped(e Element) error {
	// chain is on [walk.tallyElement]'s shared-array terms: each child's call
	// writes the slot after it, and returns before the next child's does.
	chain := append(up.path, e.Name())
	var attrs []xsd.QName
	for _, a := range e.Attributes() {
		attrs = append(attrs, a.Name())
	}
	for f := up.counting; f != nil; f = f.outer {
		f.check.tally(chain[f.depth+1:], attrs)
	}
	below := assertionAncestry{counting: up.counting, path: chain}
	kids := e.Children()
	for {
		c, ok := kids.Next()
		if !ok {
			break
		}
		child, isElement := c.Element()
		if !isElement {
			continue
		}
		if err := below.tallySkipped(child); err != nil {
			return err
		}
	}
	if err := kids.Err(); err != nil {
		return fmt.Errorf("reading the children of %s at %s: %w", e.Name(), e.Loc(), err)
	}
	return nil
}

// tallyElement tells every counting ancestor in up, and e's own clause 6 state
// own, of e: each ancestor's Tallies count e by its chain of names below that
// ancestor — the names in up's path below the ancestor's depth, then e's own —
// with e's attribute nodes ([walk.attributeNodes]) at that chain's length, and
// own's count e's attribute nodes with the empty chain, depth 0. g is e's
// governance. Where e's attribute nodes are undecided — its ·governing type
// definition· was not determined, so its ·defaulted attributes· are unknown —
// an ancestor some test of which reads attribute names at e's depth below it
// ([assertionCheck.countsAttributesAt]) — counting attribute nodes, or
// filtering children by their attributes — takes the lack instead, which
// declines its assertions, since a count missing them could fabricate a
// charge; every other ancestor counts e's element node alone, which no
// attribute name it could miss changes.
//
// It runs on every element [walk.element] enters — invalid, ·nilled·, ·laxly
// assessed· or undecided alike — once, so each node is reported exactly once,
// as [xpath.Tally] obliges. A ·skipped· element is not entered, and
// [assertionAncestry.tallySkipped] reports it and its subtree in its place.
func (w *walk) tallyElement(e Element, g governance, up assertionAncestry, own *assertionCheck) {
	if up.counting == nil && !own.counts() {
		return
	}
	attrs, decided := w.attributeNodes(e, g)
	tell := func(c *assertionCheck, path []xsd.QName) {
		if decided {
			c.tally(path, attrs)
			return
		}
		if c.countsAttributesAt(len(path)) {
			c.lacking(lackingCount{name: e.Name(), loc: e.Loc(), why: "its ·governing type definition· was not determined, so its ·defaulted attributes·, whose names a {test} reads, are unknown"})
			return
		}
		c.tally(path, nil)
	}
	// chain is e's ancestors' names from the root down, then e's own, on
	// assertionAncestry's shared-array terms: e's own subtree, which would
	// overwrite the slot, has not been walked yet.
	chain := append(up.path, e.Name())
	for f := up.counting; f != nil; f = f.outer {
		tell(f.check, chain[f.depth+1:])
	}
	if own.counts() {
		tell(own, chain[len(chain):])
	}
}

// attributeNodes is the names of e's attribute nodes in the data model instance
// cvc-assertion clause 1 builds: each attribute e carries — xsi ones included,
// namespace declarations being none ([Element.Attributes]) — then each of its
// ·defaulted attributes· (key-dflt-att, [walk.defaultedConstraint]) under g's
// complex ·governing type definition·. decided is false where this package
// could not determine that type ([governance.typeUndetermined]), so which
// uses default is unknown. An element whose type is simple, or which is ·laxly
// assessed· and so ·governed by· no type, has no ·defaulted attribute·.
func (w *walk) attributeNodes(e Element, g governance) (names []xsd.QName, decided bool) {
	if g.typeUndetermined() {
		return nil, false
	}
	attrs := e.Attributes()
	for _, a := range attrs {
		names = append(names, a.Name())
	}
	ct := g.complexType()
	if ct == nil {
		return names, true
	}
	for _, u := range ct.AttributeUses() {
		if _, defaulted := w.defaultedConstraint(u, attrs); defaulted {
			names = append(names, u.DeclarationName())
		}
	}
	return names, true
}

// yieldChildren is the kept children as an [xpath.ChildElements].
func (c *assertionCheck) yieldChildren(yield func(xpath.ChildElement) bool) {
	for _, child := range c.children {
		if !yield(child) {
			return
		}
	}
}

// assertionTest is one member of ct.{assertions}, its {test} as
// [xpath.CompileAssertionTest] compiled it, nil where that declined, and the
// [xpath.Tally] that test's evaluation reads ([xpath.AssertionTest.Tally]),
// nil where it counts nothing.
type assertionTest struct {
	a     xsd.Assertion
	test  *xpath.AssertionTest
	tally *xpath.Tally
}

// compileAssertions compiles every assertion of g's complex ·governing type
// definition· when e is entered, and is nil where there is none to compile: g
// is not complex, or its {assertions} is empty. Every input a compile reads is
// STATIC — the {test}, T's {content type}, T's {attribute uses} and the
// ·locally declared types· within T ([walk.assertionElementTypes]) — so
// nothing the [[children]] carry can change a compiled tree, and compiling
// before they arrive is what lets the walk know which of them a {test} reads
// ([assertionCheck.reads]) and which {test}s count nodes of e's subtree, each
// of which gets its Tally here, fresh for e ([assertionCheck.counts]).
//
// {assertions} is read whole and its base chain is never walked: cos-ct-extends
// clause 1.7 and derivation-ok-restriction clause 5 both make B.{assertions} a
// prefix of T.{assertions}, so unioning the chain here would report every
// inherited assertion once per derivation step. Every one of them, inherited
// or not, is compiled against T's OWN {attribute uses} ([walk.assertionTypes]),
// because the instance it reads is e's as T types it: a restriction that
// narrows an attribute's type narrows it for the base's assertions too, and
// against T's {content type}, which fixes `$value`'s static type
// (cvc-assertion clause 2.3.1.3) and whether `.` is read. Each is compiled per
// element and cached nowhere.
func (w *walk) compileAssertions(g governance) *assertionCheck {
	ct := g.complexType()
	if ct == nil || len(ct.Assertions()) == 0 {
		return nil
	}
	check := &assertionCheck{ct: *ct}
	for _, a := range ct.Assertions() {
		c := assertionTest{a: a}
		if test, compiled := xpath.CompileAssertionTest(a.Test(), w.schema, ct.ContentType(), w.assertionTypes(*ct), w.assertionElementTypes(*ct)); compiled {
			c.test = &test
			c.tally = test.Tally()
		}
		check.tests = append(check.tests, c)
	}
	return check
}

// assertionType is the {type definition} of the {attribute declaration} of
// the use u, which an assertion {test} reads that use's attribute as. resolved
// is false where the declaration or the type does not resolve to a simple
// type. It is the ONE lookup [walk.assertionTypes] and [walk.assertionValues]
// share, so the type a {test} is compiled against and the arm and type its
// values are yielded under cannot disagree, which [xpath.TypedAttributes] makes
// the caller's obligation.
func (w *walk) assertionType(u xsd.AttributeUse) (st *xsd.SimpleType, resolved bool) {
	d, resolved := w.schema.ResolvedAttributeDeclaration(u)
	if !resolved {
		return nil, false
	}
	st, simple := w.schema.ResolvedSimpleType(d.TypeDefinition())
	if !simple {
		return nil, false
	}
	return st, true
}

// assertionTypes is the [xpath.AttributeTypes] of an element whose
// ·governing type definition· is ct: the type of the {attribute use} of ct
// matching the name (cvc-complex-type clause 2.1's match, attributeUseNamed),
// on [walk.assertionType]'s terms. A name no use matches has no type fixed at
// compile time — only an {attribute wildcard} can admit it — and answers false,
// as does a use whose declaration or type does not resolve.
//
// The answer does not depend on whether the element carries the attribute: a
// carried one and a ·defaulted attribute· (key-dflt-att) are both in the
// partial ·PSVI· cvc-assertion clause 1.2 builds from, and both are yielded
// ([walk.assertionValues]); a use neither carried nor defaulted is absent, and
// the empty sequence is what the {test} reads.
func (w *walk) assertionTypes(ct xsd.ComplexType) xpath.AttributeTypes {
	return func(name xsd.QName) (*xsd.SimpleType, bool) {
		u, matched := attributeUseNamed(ct.AttributeUses(), name)
		if !matched {
			return nil, false
		}
		return w.assertionType(u)
	}
}

// assertionElementTypes is the [xpath.ElementTypes] of an element whose
// ·governing type definition· is ct: the ·locally declared type· within ct of
// a child element with the name ([xsd.Schema.LocallyDeclaredElementType],
// key-ldt-elem), false where it is ·absent·. [walk.keepChild] reads the same
// lookup, so the type a {test} is compiled against and the one a child's value
// is mapped under cannot disagree, which [xpath.ChildElements] makes the
// caller's obligation.
func (w *walk) assertionElementTypes(ct xsd.ComplexType) xpath.ElementTypes {
	return func(name xsd.QName) (xsd.TypeDefinition, bool) {
		return w.schema.LocallyDeclaredElementType(ct, name)
	}
}

// keepChild hands e, an element [[child]] of the element whose clause 6 state
// is parent, to that state once e's own assessment is over, where some {test}
// of the parent reads e's name: one [xpath.ChildElement] carrying e's typed
// value ([walk.childValue]), or the lack that declines the parent's
// assertions. frame is e's string-value frame, nil where e opened none
// ([walk.stringValue]). recorded reports whether any violation or
// [Unevaluated] was recorded since e was entered — in e itself or anywhere
// below it.
//
// A child no {test} reads is kept nowhere, and a child that is read is kept as
// ONE value, never a subtree: no value step xpath compiles reaches below a
// child but through the string-value of a child of mixed content, which its
// frame collected as the subtree streamed past and which is one string here,
// so the walk holds at most one value per element [[child]] of an element
// whose {assertions} read it, for the data model instance cvc-assertion clause
// 1 builds from the parent; an fn:count over a child step filtered by a
// predicate over its value reads it here too, and counts the kept values. A
// {test} that only counts a child, filtered by its attributes or not, or only
// asks its existence or that of a child path through it, reads its
// [xpath.Tally] instead ([walk.tallyElement]), and the child never comes
// through here for it.
func (w *walk) keepChild(parent *assertionCheck, e Element, g governance, content *contentCheck, frame *stringValueFrame, recorded bool) {
	if !parent.reads(e.Name()) {
		return
	}
	child, lack := w.childValue(parent.ct, e, g, content, frame, recorded)
	if lack != nil {
		parent.lacking(lack)
		return
	}
	parent.children = append(parent.children, child)
}

// childValue is e's typed value (xpath-datamodel §6.2.4) as a {test} of its
// parent, whose ·governing type definition· is ct, reads it: under the simple
// type the ·locally declared type· of e within ct has its value in
// ([walk.assertionElementTypes], valueTypeOf), which is the type the {test}
// was compiled against — or, where that type has mixed content, as
// [xpath.Untyped] of e's string-value, which frame collected
// ([walk.stringValue]). It reports a lackingChild instead where e's own
// assessment leaves that value unread:
//
//   - recorded: a violation makes e invalid in the partial ·PSVI·, and an
//     [Unevaluated] — String Valid over its ·initial value· withheld among
//     them — leaves its validity undecided;
//   - e has no ·governing type definition· — ·laxly assessed·, or one this
//     package could not determine — or one that is neither the same as nor
//     ·validly substitutable· ·without limitation· for the ·locally declared
//     type· ([xsd.Schema.ValidlySubstitutable] with no blocking keyword). That
//     is cvc-complex-type clause 5's own test, repeated here because
//     [walk.locallyDeclaredType] decides that clause in [walk.child], before
//     e's frame opens, so its charge is not among those recorded covers. Once
//     it holds, the simple type g.valueType() answers is the answered type
//     or validly derived from it (cos-st-derived-ok), whose lexical mapping is
//     a subset of the answered type's: an extension keeps its base's {simple
//     type definition} (cos-ct-extends clauses 1.4.1 and 2.1), and a
//     restriction's is validly derived from its base's (derivation-ok-restriction
//     clause 2.2.2.1; its clause 2.2.2.2 needs a mixed base, which answers no
//     simple type);
//   - the [schema normalized value] under e's own type does not map under the
//     answered type;
//   - the ·locally declared type· has mixed content and e's own ·governing
//     type definition· has none — an xsi:type naming a simple or an
//     element-only type — so e's typed value is not the string-value the
//     {test} was compiled to read.
//
// A ·nilled· e is [xpath.Child] of nil, the empty sequence, whatever its type
// (xpath20.md §2.5.2 item 4.1). A mixed one is [xpath.Untyped] of frame's text,
// in which a descendant whose contribution is undecided has already lacked the
// parent ([stringValueFrame.lacking]). Otherwise the value is e's ·initial
// value· ([contentCheck.assessed], the {value constraint}'s {lexical form}
// where cvc-elt clause 5.1 substituted it), normalized under e's own type's
// whiteSpace (normalizedLexical) — an extension keeps it and a restriction
// never weakens it — and mapped under the answered type, in the namespace
// context assessed pairs it with.
func (w *walk) childValue(ct xsd.ComplexType, e Element, g governance, content *contentCheck, frame *stringValueFrame, recorded bool) (xpath.ChildElement, assertionLack) {
	lacking := func(why string) (xpath.ChildElement, assertionLack) {
		return xpath.ChildElement{}, lackingChild{name: e.Name(), loc: e.Loc(), why: why}
	}
	if recorded {
		return lacking("a violation or an unevaluated check was recorded for it or below it, so it is not known to be ·valid·")
	}
	ldt, local := w.schema.LocallyDeclaredElementType(ct, e.Name())
	mixed := local && mixedContent(ldt)
	answered := valueTypeOf(ldt)
	if !local || !mixed && answered == nil {
		return lacking("its ·locally declared type· has neither a simple type its value is read under nor mixed content")
	}
	if g.typ == nil {
		return lacking("it has no ·governing type definition·")
	}
	if !sameType(g.typ, ldt) {
		substitutable, err := w.schema.ValidlySubstitutable(g.typ, ldt, nil)
		if err != nil || !substitutable {
			return lacking(fmt.Sprintf("its ·governing type definition· %s is neither its ·locally declared type· %s nor ·validly substitutable· for it", typeName(g.typ), typeName(ldt)))
		}
	}
	if content.nilled {
		return xpath.Child(e.Name(), nil), nil
	}
	if mixed {
		if !mixedContent(g.typ) {
			return lacking(fmt.Sprintf("its ·governing type definition· %s has no mixed content, so its typed value is not the string-value its ·locally declared type· %s is read as", typeName(g.typ), typeName(ldt)))
		}
		if frame == nil {
			// Unreachable: walk.element opens a frame for every child a {test}
			// reads whose ·locally declared type· and ·governing type definition·
			// both have mixed content and which is not ·nilled·
			// ([walk.stringValue]).
			return lacking("its string-value was not collected")
		}
		return xpath.Child(e.Name(), xpath.Untyped(frame.text.String())), nil
	}
	own := g.valueType()
	if own == nil {
		return lacking("its ·governing type definition· has no simple type its value is read under")
	}
	lexical, ctx := content.assessed()
	normalized, ok := normalizedLexical(w.schema, own, lexical)
	if !ok {
		return lacking("its ·initial value· has no [schema normalized value] under its own type")
	}
	v, err := value.ValidateLexical(w.backend, w.schema, answered, normalized, ctx, xpath.FacetAssertions())
	if err != nil {
		return lacking(fmt.Sprintf("its [schema normalized value] has no ·actual value· under %s", typeName(answered)))
	}
	return xpath.Child(e.Name(), xpath.Typed(v)), nil
}

// assertionValue is one attribute's typed value as an assertion {test} reads
// it.
type assertionValue struct {
	name xsd.QName
	v    xpath.TypedValue
}

// assertionInput is what one element's assertions read: its typed attributes,
// in document order, and its string value and the binding cvc-assertion clause
// 2.3 gives `$value` ([xpath.ValueBinding]).
type assertionInput struct {
	attrs []assertionValue
	value xpath.ValueBinding
}

// yield is the attributes as an [xpath.TypedAttributes].
func (in assertionInput) yield(yield func(xsd.QName, xpath.TypedValue) bool) {
	for _, a := range in.attrs {
		if !yield(a.name, a.v) {
			return
		}
	}
}

// assertionLack is why an element's assertions cannot be evaluated at all: an
// input every {test} of the element would be evaluated over is undecided, and
// [xpath.AssertionTest.Evaluate] has no undecided answer to give. The arms are
// this file's, so the decline text is a capability of the lack and the one
// decline site never switches over them (STYLE T2).
type assertionLack interface {
	// declined states what is missing for the element named e, as the clause
	// the decline message completes.
	declined(e xsd.QName) string
}

// lackingAttribute is an attribute of the instance, matching an {attribute
// use}, that has no ·actual value· ([walk.assertionValues]).
type lackingAttribute struct{ a Attribute }

// lackingDefault is a ·defaulted attribute·, which has no attribute of the
// instance to carry, whose use's ·effective value constraint· has no ·actual
// value· ([walk.assertionValues]). The use is the whole of it: the name is its
// declaration's, and the {lexical form} its ·effective value constraint·'s.
type lackingDefault struct{ u xsd.AttributeUse }

// lackingValue is a simple {content type} whose `$value` is undecided
// ([walk.assertionValue]).
type lackingValue struct{}

// lackingText is a simple {content type} over an element with element
// [[children]], whose string value, which `.` reads, this walk did not gather
// ([walk.assertionValue]).
type lackingText struct{}

// lackingNilledText is a simple {content type} over a ·nilled· element with
// [[children]], whose string value, which `.` reads, this walk did not gather
// ([walk.assertionValue]).
type lackingNilledText struct{}

// lackingChild is an element [[child]], named name at loc, that a {test} reads
// and that has no typed value this package reads, for the reason why
// ([walk.childValue]).
type lackingChild struct {
	name xsd.QName
	loc  xsderr.Loc
	why  string
}

func (l lackingAttribute) declined(e xsd.QName) string {
	return fmt.Sprintf("the attribute %s of the element %s has no ·actual value· for the data model instance cvc-assertion clause 1 builds", l.a.Name(), e)
}

func (l lackingDefault) declined(e xsd.QName) string {
	return fmt.Sprintf("the ·defaulted attribute· %s of the element %s has no ·actual value· for the data model instance cvc-assertion clause 1 builds", l.u.DeclarationName(), e)
}

// lackingCount is an element, named name at loc, of the subtree of an element
// one of whose {test}s counts its nodes, which the counting [xpath.Tally]
// cannot be told of exactly, for the reason why: its ·governing type
// definition· was not determined, and a {test} counts its attribute nodes
// ([walk.tallyElement]).
type lackingCount struct {
	name xsd.QName
	loc  xsderr.Loc
	why  string
}

func (l lackingCount) declined(e xsd.QName) string {
	return fmt.Sprintf("the element %s at %s, in the subtree of the element %s whose nodes a {test} counts, cannot be counted exactly for the data model instance cvc-assertion clause 1 builds: %s", l.name, l.loc, e, l.why)
}

func (l lackingChild) declined(e xsd.QName) string {
	return fmt.Sprintf("the child element %s of the element %s at %s, which a {test} reads, has no typed value for the data model instance cvc-assertion clause 1 builds: %s", l.name, e, l.loc, l.why)
}

func (lackingText) declined(e xsd.QName) string {
	return fmt.Sprintf("the element %s has simple content and element [[children]], which cvc-complex-type clause 1.2 charged, so its string value, the text of all its descendants, was not gathered", e)
}

func (lackingNilledText) declined(e xsd.QName) string {
	return fmt.Sprintf("the element %s has xsi:nil = true and [[children]], which cvc-elt clause 3.2.3.1 charged, so its string value, the text of its descendants and never the zero-length string while dm:nilled is false in the partial ·PSVI·, was not gathered", e)
}

func (lackingValue) declined(e xsd.QName) string {
	return fmt.Sprintf("the element %s has simple content whose [schema actual value], which cvc-assertion clause 2.3.1 binds to $value, is undecided: String Valid over its ·initial value· was withheld", e)
}

// assertionValues is the input of e's assertions: the typed value
// ([walk.assertionTyped]) of each attribute of attrs that matches an {attribute
// use} of ct, in document order, then of each ·defaulted attribute· of e
// ([walk.defaultedConstraint], key-dflt-att) in the order of ct.{attribute
// uses}, each under the type [walk.assertionType] resolves for its use, and
// the binding of `.` and `$value` ([walk.assertionValue]).
//
// A ·defaulted attribute· is read as its use's ·effective value constraint·
// supplies it: its {lexical form} is the [schema normalized value], and the
// ·actual value· is mapped from it under [value.ConstraintContext], the bindings
// in scope where the schema document wrote it (Datatypes §3.3.18), never e's.
// The partial ·PSVI· cvc-assertion clause 1.2 builds from holds it — clause 1.1
// sets aside only cvc-complex-type clause 6, not the attribute defaulting
// key-dflt-att's PSVI contribution makes — so reading it as the empty sequence
// instead could fabricate a charge.
//
// It reports the lack, carrying the first attribute lacking one, where any
// such attribute has no ·actual value·: its declaration or {type definition}
// does not resolve, or String Valid ([walk.stringValid]) over its lexical is
// rejected or withheld — charged or declined by cvc-attribute clause 3 for a
// carried attribute and by cvc-complex-type clause 4 for a defaulted one
// ([walk.defaultedAttribute]), the same check re-run here because the walk
// keeps no ·actual values·. Omitting such an attribute instead would make `@a`
// the empty sequence and could fabricate a charge. After the attributes it
// reports asserts' own lack: an element [[child]] a {test} reads that has no
// typed value ([walk.keepChild]), or a node of e's subtree a {test} counts that
// its Tally could not be told of ([walk.tallyElement]).
//
// An attribute matching no use is not read: no {test}
// [xpath.CompileAssertionTest] admits can name it ([walk.assertionTypes]), so
// its own ·actual value· decides nothing here.
//
// A lack declines every assertion of e, including one whose {test} never reads
// what is lacking: whether a {test} reads `$value` is not something the
// compiled test reports, and a child's lack, like a count's, is kept per
// element and not per {test}.
func (w *walk) assertionValues(e Element, attrs []Attribute, asserts *assertionCheck, content *contentCheck, invalid bool) (assertionInput, assertionLack) {
	ct := asserts.ct
	var in assertionInput
	for _, a := range attrs {
		u, matched := attributeUseNamed(ct.AttributeUses(), a.Name())
		if !matched {
			continue
		}
		v, read := w.assertionTyped(u, a.Value(), elementContext{owner: e}, a.Loc())
		if !read {
			return assertionInput{}, lackingAttribute{a: a}
		}
		in.attrs = append(in.attrs, assertionValue{name: a.Name(), v: v})
	}
	for _, u := range ct.AttributeUses() {
		vc, defaulted := w.defaultedConstraint(u, attrs)
		if !defaulted {
			continue
		}
		v, read := w.assertionTyped(u, vc.LexicalForm(), value.ConstraintContext(vc), e.Loc())
		if !read {
			return assertionInput{}, lackingDefault{u: u}
		}
		in.attrs = append(in.attrs, assertionValue{name: u.DeclarationName(), v: v})
	}
	if asserts.lack != nil {
		return assertionInput{}, asserts.lack
	}
	bound, lack := w.assertionValue(e, ct, content, invalid)
	if lack != nil {
		return assertionInput{}, lack
	}
	in.value = bound
	return in, nil
}

// assertionTyped is the typed value of an attribute of the use u whose
// [schema normalized value] is lexical, read at loc and mapped under ctx,
// reporting false where it has no ·actual value·: u's declaration or {type
// definition} does not resolve ([walk.assertionType]), String Valid
// ([walk.stringValid]) over lexical is rejected or withheld, or the mapping
// errors.
//
// The value is [xpath.Untyped] of lexical where the type is ·special·
// ([xsd.SimpleType.IsSpecial]): xpath-datamodel §3.3.1.2 makes it the [schema
// normalized value] as xs:untypedAtomic. A carried attribute's [[normalized
// value]] is that value unchanged, because key-nv normalizes under
// xs:anySimpleType "as in the preserve case" and xs:anyAtomicType carries no
// whiteSpace facet either; a defaulted one's {lexical form} is it by
// key-dflt-att. The value is [xpath.Typed] of the ·actual value· mapped under
// the type otherwise.
func (w *walk) assertionTyped(u xsd.AttributeUse, lexical string, ctx value.Context, loc xsderr.Loc) (xpath.TypedValue, bool) {
	st, resolved := w.assertionType(u)
	if !resolved {
		return nil, false
	}
	decided, verdict := w.stringValid(st, lexical, ctx, loc)
	if !decided || verdict != nil {
		return nil, false
	}
	if st.IsSpecial() {
		return xpath.Untyped(lexical), true
	}
	v, err := value.ValidateLexical(w.backend, w.schema, st, lexical, ctx, xpath.FacetAssertions())
	if err != nil {
		return nil, false
	}
	return xpath.Typed(v), true
}

// assertionValue is what e's assertions read of e itself — its string value,
// which `.` atomizes to, and the value cvc-assertion clause 2.3 binds to
// `$value` ([xpath.BindValue]) — or the lack that leaves either undecided.
//
// The string value is the ·initial value· [contentCheck.assessed] answers, or
// the {value constraint}'s {lexical form} cvc-elt clause 5.1 substitutes for an
// empty e, unnormalized. It is bound whatever `$value` is, an invalid e's
// included, but for the e below: `.` is E's string value whatever its
// [validity]. An e whose {content type} is not simple binds the zero
// [xpath.ValueBinding], a string value no {test} compiled for that content
// reads, `.` declining there ([xpath.CompileAssertionTest]).
//
// GAP(xpath): E's string value under simple content is a processor's choice
// the data model leaves open, and this walk makes one reading of it.
// xpath-datamodel Appendix J.2's children rule builds a Text Node from the
// character [[children]], and also says a processor "may" build instead one
// Text Node from the [schema normalized value]; §6.2.4's string-value gives an
// EMPTY element the zero-length string. This walk binds the raw ·initial value·
// for an e with character [[children]], J.2's first reading, which the
// xs:anyType annotation of the partial ·PSVI· points at, and the {value
// constraint}'s {lexical form} for a defaulted e with none — the [schema
// normalized value] cvc-elt clause 5.1 supplies, J.2's "may" — and not the
// zero-length string §6.2.4 gives an empty element. The direction is
// unestablished: a {test} reading `.` can be charged or satisfied under either
// reading.
//
// A ·nilled· e binds the zero [xpath.ValueBinding] where it has no [[children]]
// ([contentCheck.empty]): the zero-length string, an empty element's string
// value (xpath-datamodel §6.2.4), and the empty `$value` of clause 2.3.1.2.
// dm:nilled is false in the partial ·PSVI· — [validity] is never valid there —
// so a ·nilled· e WITH [[children]], which cvc-elt clause 3.2.3.1 charged as
// the first arrived, has the text of its descendants as its string value, and
// this walk gathers none of it ([contentCheck.gathers]). Its assertions are
// declined ([lackingNilledText]) on the terms [walk.elementAssertions] states,
// a {test} reading only `$value` among them, since the lack is kept per element
// and not per {test}.
//
// An e under simple content that has element [[children]] — which
// cvc-complex-type clause 1.2 charged as the first arrived, so invalid is true
// — has a string value this walk never gathered: the text of its descendants,
// and every run after that charge ([contentCheck.text]). Its assertions are
// declined ([lackingText]) on the terms [walk.elementAssertions] states, never
// evaluated over the partial text.
//
// Clause 2.3.2's empty sequence — a nil `$value` — is decided from three facts
// this walk holds: ct's {content type} is not simple, e is ·nilled· (clause
// 2.3.1.2), or e is already known to be invalid (clause 2.3.1.1: the partial
// ·PSVI·'s [validity] "is given the value invalid if and only if the element is
// known to be invalid").
//
// Over a ·special· {simple type definition} ([xsd.SimpleType.IsSpecial])
// `$value` is [xpath.Untyped] of the ·initial value· [contentCheck.assessed]
// answers, which is e's [schema normalized value] unchanged — key-nv
// normalizes under xs:anySimpleType "as in the preserve case" — and whose XDM
// representation is that value as xs:untypedAtomic (Datatypes dt-xdmrep clause
// 1). String Valid accepts every lexical against either ·special· type, so it
// is not re-run.
//
// Otherwise `$value` is e's [schema actual value]: the ·initial value·, or the
// {value constraint}'s {lexical form} cvc-elt clause 5.1 substitutes for an
// empty e ([contentCheck.assessed]), mapped under the {simple type definition},
// in the namespace context assessed pairs it with, by String Valid
// ([walk.stringValid]) re-run as [walk.assertionValues] re-runs it for an
// attribute. It is undecided where String Valid is withheld. A rejection is
// clause 2.3.2 again, which cvc-complex-type clause 1.2 has charged by now.
//
// A DECLINED check of e's own elsewhere leaves e's [validity] undecided
// between invalid and notKnown, and the actual value is bound all the same:
// were e invalid, e is rejected whatever its assertions answer, so no answer
// they give turns a valid document invalid or an invalid one valid.
func (w *walk) assertionValue(e Element, ct xsd.ComplexType, content *contentCheck, invalid bool) (xpath.ValueBinding, assertionLack) {
	simple, isSimple := ct.ContentType().(xsd.SimpleContent)
	if !isSimple {
		return xpath.ValueBinding{}, nil
	}
	if content.nilled && !content.empty() {
		return xpath.ValueBinding{}, lackingNilledText{}
	}
	if content.nilled {
		return xpath.ValueBinding{}, nil
	}
	if content.sawElement {
		return xpath.ValueBinding{}, lackingText{}
	}
	lexical, ctx := content.assessed()
	if invalid {
		return xpath.BindValue(lexical, nil), nil
	}
	if simple.SimpleType.IsSpecial() {
		return xpath.BindValue(lexical, xpath.Untyped(lexical)), nil
	}
	decided, verdict := w.stringValid(simple.SimpleType, lexical, ctx, e.Loc())
	if !decided {
		return xpath.ValueBinding{}, lackingValue{}
	}
	if verdict != nil {
		return xpath.BindValue(lexical, nil), nil
	}
	v, err := value.ValidateLexical(w.backend, w.schema, simple.SimpleType, lexical, ctx, xpath.FacetAssertions())
	if err != nil {
		return xpath.ValueBinding{}, lackingValue{}
	}
	return xpath.BindValue(lexical, xpath.Typed(v)), nil
}

// declineAssertions records err, the non-verdict [walk.stringValid] withheld a
// verdict on, as a decline under cvc-assertions-valid at the item named name at
// loc, under event, where it is an assertions-facet decline
// ([value.IsAssertionDeclined]), and reports whether it was one. withheld names
// the caller's clause the decline leaves undecided. The message is the one
// value's decline carries, naming the assertion, the simple type its facet is
// effective on and its {test}: an assertion component carries no Loc of its
// own (#35).
//
// cvc-assertions-valid (§4.3.13.3) is evaluated inside String Valid: the value
// pipeline runs each assertions facet it reaches through
// [xpath.FacetAssertions], so a failed {test} is part of the Datatype Valid
// verdict (cvc-datatype-valid clause 3) the caller charges under its own rule,
// and a union's ·validating type· — which [walk.validatingType],
// [walk.roleValues] and [walk.keyMember] read — is the first member whose
// facets, assertions included, accept the value (dt-active-member).
//
// GAP(validate): a {test} [xpath.FacetAssertions] declines — one outside the
// grammar [xpath.CompileAssertionTest] admits, over the facet's type or, for a
// union's own assertions facet, over its ·active basic member· — leaves the
// value's Datatype Valid verdict undecided, and is DECLINED here. The cvcid.go
// and cvcidentityconstraint.go re-runs reach the same non-verdict and decline
// on it, and the union dispatch stops at a member that declined rather than
// handing the value to a later one, so no ·validating type· is chosen past a
// {test} nobody decided. Fail-open: the withheld verdict's whole consumer set
// is w.res.violations and its one reader [Result.Violations], which charge on a
// violation PRESENT, so a decline can only cost a rejection and can manufacture
// none. (#1042)
func (w *walk) declineAssertions(err error, event string, name xsd.QName, loc xsderr.Loc, withheld string) bool {
	if !value.IsAssertionDeclined(err) {
		return false
	}
	w.decline(event, name, loc, ruleCvcAssertionsValid, "",
		"%s; that facet is reached from cvc-datatype-valid clause 3, so %s is undecided", declinedAssertion(err), withheld)
	return true
}

// declinedAssertion is the message of the *xsderr.Error value's decline err
// carries, which names the assertion it declined, or err's own rendering where
// it carries none.
func declinedAssertion(err error) string {
	var xe *xsderr.Error
	if errors.As(err, &xe) {
		return xe.Msg
	}
	return err.Error()
}
