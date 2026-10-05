package validate

import (
	"fmt"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xpath"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// This file settles cvc-complex-type (§3.4.4.2) clause 6 — each assertion in
// the {assertions} of a complex ·governing type definition· — by EVALUATING its
// {test} wherever xpath compiles it, and records every assertions-facet site of
// a simple type as an [Unevaluated] without evaluating any. Both halves decline
// through [walk.decline], so every site this file does not decide is recorded
// and logged at the item it was reached at, and the two GAP markers below
// state, separately, what each of the two hooks still withholds.
//
// The two rules are DISTINCT and are never conflated. cvc-assertion
// (§3.13.4.1) is the complex-type variety, reached from cvc-complex-type
// (§3.4.4.2) clause 6 and from nowhere else. cvc-assertions-valid (Datatypes
// §4.3.13.3) is the simple-type one, reached from cvc-datatype-valid (§4.1.4)
// clause 3 (dv_vfacets) over the assertions facet of a simple type. One record
// type carries both, discriminated by [Unevaluated.Rule].
//
// The simple-type sites are collected STATICALLY off the type
// ([walk.assertionSites]): being unevaluated is a property of the compiled
// type, not of a value, and the cvc-datatype-valid variety recursion PRINCIPLES
// 12 quantifies over lives in package value behind a (Value, error) signature
// with no channel for a non-verdict. The collection therefore OVER-reports
// against what a real evaluator would have touched — every member of a union
// gets a record, not only the ·validating· one, and a list gets one record for
// its {item type definition} rather than one per item — and that is the safe
// direction: the added records' whole consumer set is Result.unevaluated and
// its one reader [Result.Unevaluated], which carry sites PRESENT, so an extra
// record costs a decline and manufactures no charge. A record count is not a
// claim about how many evaluations were skipped.
//
// The recording sites are the four places the assessment decides an instance
// lexical against a simple type — cvc-attribute clause 3 ([walk.declaredAttribute],
// for an attribute matched by an {attribute use} and for one ·attributed to· a
// strict or lax {attribute wildcard} alike), cvc-complex-type clause 4 over a
// ·defaulted attribute·'s {lexical form} ([walk.defaultedAttribute]), and cvc-type
// clause 3.1.3 / cvc-complex-type clause 1.2 over an element's ·initial value·
// ([contentCheck.stringValid]). cvcid.go and cvcidentityconstraint.go re-run the
// datatype pipeline over lexicals those sites already recorded, and record nothing
// of their own.

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
// misreport every simple-type assertion the day evaluation lands.
const ruleCvcAssertionsValid xsderr.Rule = "cvc-assertions-valid"

// elementAssertions settles cvc-complex-type (§3.4.4.2) clause 6 for e: "E is
// ·valid· with respect to each of the assertions in T.{assertions} as per
// Assertion Satisfied (§3.13.4.1)", T being e's ·governing type definition·. A
// governing type that is not a Complex Type Definition has no {assertions}
// property at all — a simple one's assertions are facets, and reach
// ruleCvcAssertionsValid instead.
//
// asserts is e's clause 6 state, opened when e was entered
// ([walk.compileAssertions], which compiles every {test}) and nil where T has
// no {assertions}. Each compiled test is evaluated with the binding
// [walk.assertionValue] gives `$value`: invalid says whether e is known to be
// invalid in the partial ·PSVI· by now (clause 2.3.1.1) — the caller's count of
// the violations recorded since e was entered — and content is e's own content
// check, exhausted, which holds the ·initial value· and the ·nilled· answer.
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
// residue (paths, the function library); and every assertion of an e one of
// whose attributes matching an {attribute use}, carried or ·defaulted·, has no
// ·actual value·, one of whose element [[children]] a {test} reads has no
// typed value this package reads ([walk.keepChild]), or whose `$value` is
// undecided ([walk.assertionValues]). Fail-open: the withheld value is clause
// 6's own verdict, whose whole consumer set inside this package is
// w.res.violations and its one reader [Result.Violations], which charge on a
// violation PRESENT, so a decline can only cost a rejection and can
// manufacture none. (#1042)
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
		if test.Evaluate(w.backend, w.schema, in.yield, asserts.yieldChildren, in.value) {
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
// child that has no typed value this package reads, nil while there is none.
type assertionCheck struct {
	ct       xsd.ComplexType
	tests    []assertionTest
	children []xpath.ChildElement
	lack     assertionLack
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

// lacking records l as c's lack where c has none yet, so the first lacking
// child in document order is the one a decline names.
func (c *assertionCheck) lacking(l assertionLack) {
	if c.lack == nil {
		c.lack = l
	}
}

// skipped records e, an element [[child]] ·skipped· by key-sva clause 3.2, as
// c's lack where some test of c reads e's name: a ·skipped· element is not
// ·assessed·, so the partial ·PSVI· gives it no type to read its value under.
// A nil c reads nothing.
func (c *assertionCheck) skipped(e Element) {
	if !c.reads(e.Name()) {
		return
	}
	c.lacking(lackingChild{name: e.Name(), loc: e.Loc(), why: "it is ·skipped·, so it is not ·assessed·"})
}

// yieldChildren is the kept children as an [xpath.ChildElements].
func (c *assertionCheck) yieldChildren(yield func(xpath.ChildElement) bool) {
	for _, child := range c.children {
		if !yield(child) {
			return
		}
	}
}

// assertionTest is one member of ct.{assertions} and its {test} as
// [xpath.CompileAssertionTest] compiled it, nil where that declined.
type assertionTest struct {
	a    xsd.Assertion
	test *xpath.AssertionTest
}

// compileAssertions compiles every assertion of g's complex ·governing type
// definition· when e is entered, and is nil where there is none to compile: g
// is not complex, or its {assertions} is empty. Every input a compile reads is
// STATIC — the {test}, T's {content type}, T's {attribute uses} and the
// ·locally declared types· within T ([walk.assertionElementTypes]) — so
// nothing the [[children]] carry can change a compiled tree, and compiling
// before they arrive is what lets the walk know which of them a {test} reads
// ([assertionCheck.reads]).
//
// {assertions} is read whole and its base chain is never walked: cos-ct-extends
// clause 1.7 and derivation-ok-restriction clause 5 both make B.{assertions} a
// prefix of T.{assertions}, so unioning the chain here would report every
// inherited assertion once per derivation step. Every one of them, inherited
// or not, is compiled against T's OWN {attribute uses} ([walk.assertionTypes]),
// because the instance it reads is e's as T types it: a restriction that
// narrows an attribute's type narrows it for the base's assertions too, and
// against T's {content type}, which fixes `$value`'s static type
// (cvc-assertion clause 2.3.1.3). Each is compiled per element and cached
// nowhere.
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
// assertions. recorded reports whether any violation or [Unevaluated] was
// recorded since e was entered — in e itself or anywhere below it.
//
// A child no {test} reads is kept nowhere, and a child that is read is kept as
// ONE value, never a subtree: no step xpath compiles reaches below a child, so
// the walk holds at most one value per element [[child]] of an element whose
// {assertions} read it, for the data model instance cvc-assertion clause 1
// builds from the parent.
func (w *walk) keepChild(parent *assertionCheck, e Element, g governance, content *contentCheck, recorded bool) {
	if !parent.reads(e.Name()) {
		return
	}
	child, lack := w.childValue(parent.ct, e, g, content, recorded)
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
// was compiled against. It reports a lackingChild instead where e's own
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
//     answered type.
//
// A ·nilled· e is [xpath.Child] of nil, the empty sequence. Otherwise the value
// is e's ·initial value· ([contentCheck.assessed], the {value constraint}'s
// {lexical form} where cvc-elt clause 5.1 substituted it), normalized under e's
// own type's whiteSpace (normalizedLexical) — an extension keeps it and a
// restriction never weakens it — and mapped under the answered type.
func (w *walk) childValue(ct xsd.ComplexType, e Element, g governance, content *contentCheck, recorded bool) (xpath.ChildElement, assertionLack) {
	lacking := func(why string) (xpath.ChildElement, assertionLack) {
		return xpath.ChildElement{}, lackingChild{name: e.Name(), loc: e.Loc(), why: why}
	}
	if recorded {
		return lacking("a violation or an unevaluated check was recorded for it or below it, so it is not known to be ·valid·")
	}
	ldt, local := w.schema.LocallyDeclaredElementType(ct, e.Name())
	answered := valueTypeOf(ldt)
	if !local || answered == nil {
		return lacking("its ·locally declared type· has no simple type its value is read under")
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
	own := g.valueType()
	if own == nil {
		return lacking("its ·governing type definition· has no simple type its value is read under")
	}
	normalized, ok := normalizedLexical(w.schema, own, content.assessed())
	if !ok {
		return lacking("its ·initial value· has no [schema normalized value] under its own type")
	}
	v, err := value.ValidateLexical(w.backend, w.schema, answered, normalized, elementContext{owner: e}, xpath.FacetAssertions())
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
// in document order, and the binding cvc-assertion clause 2.3 gives `$value`.
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

func (l lackingChild) declined(e xsd.QName) string {
	return fmt.Sprintf("the child element %s of the element %s at %s, which a {test} reads, has no typed value for the data model instance cvc-assertion clause 1 builds: %s", l.name, e, l.loc, l.why)
}

func (lackingValue) declined(e xsd.QName) string {
	return fmt.Sprintf("the element %s has simple content whose [schema actual value], which cvc-assertion clause 2.3.1 binds to $value, is undecided: String Valid over its ·initial value· was withheld", e)
}

// assertionValues is the input of e's assertions: the typed value
// ([walk.assertionTyped]) of each attribute of attrs that matches an {attribute
// use} of ct, in document order, then of each ·defaulted attribute· of e
// ([walk.defaultedConstraint], key-dflt-att) in the order of ct.{attribute
// uses}, each under the type [walk.assertionType] resolves for its use, and
// `$value`'s binding ([walk.assertionValue]).
//
// A ·defaulted attribute· is read as its use's ·effective value constraint·
// supplies it: its {lexical form} is the [schema normalized value], and the
// ·actual value· is mapped from it. The partial ·PSVI· cvc-assertion clause 1.2
// builds from holds it — clause 1.1 sets aside only cvc-complex-type clause 6,
// not the attribute defaulting key-dflt-att's PSVI contribution makes — so
// reading it as the empty sequence instead could fabricate a charge.
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
// typed value ([walk.keepChild]).
//
// An attribute matching no use is not read: no {test}
// [xpath.CompileAssertionTest] admits can name it ([walk.assertionTypes]), so
// its own ·actual value· decides nothing here.
//
// A lack declines every assertion of e, including one whose {test} never reads
// what is lacking: whether a {test} reads `$value` is not something the
// compiled test reports, and a child's lack is kept per element and not per
// {test}.
func (w *walk) assertionValues(e Element, attrs []Attribute, asserts *assertionCheck, content *contentCheck, invalid bool) (assertionInput, assertionLack) {
	ct := asserts.ct
	var in assertionInput
	for _, a := range attrs {
		u, matched := attributeUseNamed(ct.AttributeUses(), a.Name())
		if !matched {
			continue
		}
		v, read := w.assertionTyped(u, a.Value(), e, a.Loc())
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
		v, read := w.assertionTyped(u, vc.LexicalForm(), e, e.Loc())
		if !read {
			return assertionInput{}, lackingDefault{u: u}
		}
		in.attrs = append(in.attrs, assertionValue{name: u.DeclarationName(), v: v})
	}
	if asserts.lack != nil {
		return assertionInput{}, asserts.lack
	}
	bound, decided := w.assertionValue(e, ct, content, invalid)
	if !decided {
		return assertionInput{}, lackingValue{}
	}
	in.value = bound
	return in, nil
}

// assertionTyped is the typed value of an attribute of the use u whose
// [schema normalized value] is lexical, read at loc on e, reporting false
// where it has no ·actual value·: u's declaration or {type definition} does not
// resolve ([walk.assertionType]), String Valid ([walk.stringValid]) over
// lexical is rejected or withheld, or the mapping errors.
//
// The value is [xpath.Untyped] of lexical where the type is ·special·
// (isSpecial): xpath-datamodel §3.3.1.2 makes it the [schema normalized value]
// as xs:untypedAtomic. A carried attribute's [[normalized value]] is that
// value unchanged, because key-nv normalizes under xs:anySimpleType "as in the
// preserve case" and xs:anyAtomicType carries no whiteSpace facet either; a
// defaulted one's {lexical form} is it by key-dflt-att. The value is
// [xpath.Typed] of the ·actual value· mapped under the type otherwise.
func (w *walk) assertionTyped(u xsd.AttributeUse, lexical string, e Element, loc xsderr.Loc) (xpath.TypedValue, bool) {
	st, resolved := w.assertionType(u)
	if !resolved {
		return nil, false
	}
	decided, verdict := w.stringValid(st, lexical, e, loc)
	if !decided || verdict != nil {
		return nil, false
	}
	if isSpecial(st) {
		return xpath.Untyped(lexical), true
	}
	v, err := value.ValidateLexical(w.backend, w.schema, st, lexical, elementContext{owner: e}, xpath.FacetAssertions())
	if err != nil {
		return nil, false
	}
	return xpath.Typed(v), true
}

// assertionValue is the value cvc-assertion clause 2.3 binds to `$value` for e,
// reporting false where it is undecided.
//
// Clause 2.3.2's empty sequence — the zero [xpath.ValueBinding] — is decided
// from three facts this walk holds: ct's {content type} is not simple, e is
// ·nilled· (clause 2.3.1.2), or e is already known to be invalid (clause
// 2.3.1.1: the partial ·PSVI·'s [validity] "is given the value invalid if and
// only if the element is known to be invalid").
//
// Over a ·special· {simple type definition} (isSpecial) the value is
// [xpath.Untyped] of the ·initial value· [contentCheck.assessed] answers, which
// is e's [schema normalized value] unchanged — key-nv normalizes under
// xs:anySimpleType "as in the preserve case" — and whose XDM representation is
// that value as xs:untypedAtomic (Datatypes dt-xdmrep clause 1). String Valid
// accepts every lexical against either ·special· type, so it is not re-run.
//
// Otherwise the value is e's [schema actual value]: the ·initial value·, or
// the {value constraint}'s {lexical form} cvc-elt clause 5.1 substitutes for an
// empty e ([contentCheck.assessed]), mapped under the {simple type definition}
// by String Valid ([walk.stringValid]) re-run as [walk.assertionValues] re-runs
// it for an attribute. It is undecided where String Valid is withheld. A
// rejection would be clause 2.3.2 again, but cvc-complex-type clause 1.2 has
// charged it by now, so invalid is already true.
//
// A DECLINED check of e's own elsewhere leaves e's [validity] undecided
// between invalid and notKnown, and the actual value is bound all the same:
// were e invalid, e is rejected whatever its assertions answer, so no answer
// they give turns a valid document invalid or an invalid one valid.
func (w *walk) assertionValue(e Element, ct xsd.ComplexType, content *contentCheck, invalid bool) (xpath.ValueBinding, bool) {
	simple, isSimple := ct.ContentType().(xsd.SimpleContent)
	if !isSimple || content.nilled || invalid {
		return xpath.ValueBinding{}, true
	}
	lexical := content.assessed()
	if isSpecial(simple.SimpleType) {
		return xpath.BindValue(xpath.Untyped(lexical)), true
	}
	decided, verdict := w.stringValid(simple.SimpleType, lexical, e, e.Loc())
	if !decided {
		return xpath.ValueBinding{}, false
	}
	if verdict != nil {
		return xpath.ValueBinding{}, true
	}
	v, err := value.ValidateLexical(w.backend, w.schema, simple.SimpleType, lexical, elementContext{owner: e}, xpath.FacetAssertions())
	if err != nil {
		return xpath.ValueBinding{}, false
	}
	return xpath.BindValue(xpath.Typed(v)), true
}

// simpleAssertions records every assertions-facet site st carries through
// [walk.decline], under event for the item named name at loc — the attribute
// or element whose lexical is being decided against st, an assertion component
// carrying no Loc of its own (#35).
//
// GAP(validate): DIRECTION UNESTABLISHED. cvc-assertions-valid (§4.3.13.3) is
// not evaluated at all — a facet assertion reads its value through `$value`
// (clause 1.1), which xpath binds for an element's {assertions} and this
// package does not yet bind for a facet's (#2246, #1042) — so the assertions
// facet contributes nothing to the Datatype Valid (§4.1.4) verdict its clause
// 3 folds it into. The withheld value is a conjunct of datatype-validity, and
// its readers are NOT only w.res.violations and [Result.Violations] — which
// charge on a violation PRESENT and so lose a rejection.
// [walk.validatingType] and [walk.roleValues] (cvcid.go) classify a value by
// its ·validating type·, which cvc-datatype-valid clause 2.3 makes the FIRST
// member of a union the value is Datatype Valid against: an unchecked
// assertion can leave an earlier member ·validating· that the spec rejects,
// binding an ·ID value· the spec's §3.17.5.2 table has none of, which cvc-id
// clause 2 then charges as a duplicate, or an ·ENTITY value· String Valid
// clause 3 then charges as undeclared (cvcsimpletype.go). [walk.keyMember]
// (cvcidentityconstraint.go) reads a PRESENT [schema actual value] where the
// spec's is ·absent· for the same reason, lengthening a ·key-sequence· into
// the duplicate arm of cvc-identity-constraint clause 4. Both of those are
// FALSE REJECTS, so this hook is not fail-open, and the direction over the
// whole consumer set is not established here (STYLE P3a).
func (w *walk) simpleAssertions(st *xsd.SimpleType, event string, name xsd.QName, loc xsderr.Loc) {
	for _, s := range w.assertionSites(st) {
		w.decline(event, name, loc, ruleCvcAssertionsValid, "",
			"assertion %d of %d in the {value} of the assertions facet of the simple type %s, whose {test} is %q, was not evaluated, so the value at this location is not shown facet-valid with respect to that facet as cvc-assertions-valid requires (Datatypes §4.3.13.3, reached from cvc-datatype-valid clause 3)",
			s.i+1, s.n, typeName(s.st), s.test)
	}
}

// assertionSite is one member of the {value} of one assertions facet: the
// simple type the facet is effective on, the member's position i among n, and
// its {test}'s {expression}.
type assertionSite struct {
	st   *xsd.SimpleType
	i, n int
	test string
}

// assertionSites is the site set of one simple type, in cvc-datatype-valid
// (§4.1.4)'s own order: the constituents clause 2 recurses into first, then the
// type's own facets under clause 3. A list yields its {item type definition}'s
// sites then its own, a union yields each {member type definition}'s in
// document order then its own, and an atomic type yields its own alone — the
// three levels PRINCIPLES 12 names, all of which can carry assertions
// independently.
//
// It carries no visited set (STYLE D4). A union whose membership is circular is
// rejected at construction (xsd.Schema's checkUnionMembershipAcyclic) and
// §3.16.1 std-item_type_definition forbids a list of lists, which is why
// package value's own list and union recursions carry none either.
//
// An unresolvable hop — a {variety} whose base chain breaks, an itemType= or
// memberTypes= naming a type the schema cannot resolve — yields st's OWN
// assertions-facet sites rather than an error. Those sites survive the break:
// [walk.ownAssertionSites] reads [xsd.SimpleType.EffectiveFacets] and depends
// on none of the three hops, so a list whose itemType= resolves to nothing
// still has the list type's own assertions facet readable, while its
// constituents' sites are unreachable through a type nothing can map. There is
// no charge here to be wrong about: the same type fails value.ValidateLexical
// at every recording site, which declines there.
//
// No test drives those three arms and none can: each error is an unresolvable
// simple-type reference, unreachable for a finalized Schema for the reason
// xsd's validlyDerived states. They are the safe answer for a resolver that is
// not one, which [xsd.SimpleType.Variety] admits by taking a TypeResolver.
func (w *walk) assertionSites(st *xsd.SimpleType) []assertionSite {
	if st == nil {
		return nil
	}
	variety, err := st.Variety(w.schema)
	if err != nil {
		return w.ownAssertionSites(st)
	}
	var sites []assertionSite
	switch variety.(type) {
	case xsd.List:
		item, err := st.Item(w.schema)
		if err != nil {
			return w.ownAssertionSites(st)
		}
		sites = w.assertionSites(item)
	case xsd.Union:
		members, err := st.Members(w.schema)
		if err != nil {
			return w.ownAssertionSites(st)
		}
		for _, m := range members {
			sites = append(sites, w.assertionSites(m)...)
		}
	}
	return append(sites, w.ownAssertionSites(st)...)
}

// ownAssertionSites is one type's OWN assertions-facet sites, read off
// [xsd.SimpleType.EffectiveFacets] and never assembled by walking the base
// chain: §4.3.13.2 xr-assertions builds the facet's {value} as the base's
// Assertions followed by the restriction's own, and cos-assertions-restriction
// (§4.3.13.4) requires that prefix, so the effective facet already carries
// every inherited assertion exactly once.
//
// An EffectiveFacets error yields no sites, which is where assertionSites'
// "those sites survive the break" stops applying: the type's own facets are
// unreadable too. It is unreachable for a finalized Schema, whose src-resolve
// pass resolves every {base type definition} the effective facets fold over,
// on the terms assertionSites states for its three arms; and the type fails
// value.ValidateLexical at every recording site, which declines there, so the
// lost sites under-report a type that is declined anyway.
func (w *walk) ownAssertionSites(st *xsd.SimpleType) []assertionSite {
	facets, err := st.EffectiveFacets(w.schema)
	if err != nil {
		return nil
	}
	var sites []assertionSite
	for _, ef := range facets {
		assertions, isAssertions := ef.Facet().Assertions()
		if !isAssertions {
			continue
		}
		for i, a := range assertions {
			sites = append(sites, assertionSite{st: st, i: i, n: len(assertions), test: a.Test().Expression()})
		}
	}
	return sites
}
