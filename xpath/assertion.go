package xpath

import (
	"time"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file is the ASSERTION façade over the §3.12.6 grammar compileCTATest
// parses (STYLE T4): an {assertions} member's {test} (§3.13.1) written in that
// grammar is compiled and evaluated against the element it guards, which
// cvc-assertion (§3.13.4.1) asks of it. It holds the first slices of tier 2
// (doc.go) and nothing wider: the grammar is the Type Alternative one, widened
// only where the façade admits a production (ctaFacade) — an assertion's
// instance is TYPED.
//
// cvc-assertion clause 1 builds the XDM instance from the partial ·PSVI· of E,
// so each attribute carries the type its ·governing attribute declaration·
// gave it, and §3.13.2's own example relies on that: in `@min le @max` over
// two xs:int attributes "the typed values of the attributes are available for
// comparison; it is not necessary to cast". A Type Alternative's instance is
// untyped instead (key-cta-ta-select clause 1's Note), which is why the two
// façades build different nodes for one [17] ta-AttrName over a typed
// attribute and take different attribute inputs, and why neither tree can be
// fed the other's input. An attribute whose type is ·special· is the one an
// assertion reads untyped too: its typed value is xs:untypedAtomic
// (xpath-datamodel §3.3.1.2), so both façades build the same node for it.
//
// The assertion façade widens the grammar by NINE productions the Type
// Alternative façade declines: the eq/ne/lt/le/gt/ge value comparisons
// (xpath20.md §3.5.1, [23] ValueComp), in [11] ta-BooleanExpr's comparator
// position; three more arms of [16] ta-SimpleValue — the variable reference
// `$value` (xpath20.md [44] VarRef), which cvc-assertion clause 2.2 adds to an
// assertion's static context where ta-props-correct clause 2's grammar names
// no variable; an abbreviated child-axis step with a QName NameTest
// (§3.2.1.1), which reads E's element [[children]], whose typed values
// cvc-assertion clause 1.2's partial ·PSVI· holds; and a path opening with "/"
// or "//" over one child or attribute step, which raises err:XPDY0050 over
// that instance (§3.2); a relative path of two or more such child steps (§3.2,
// [26] RelativePathExpr), or one element step `N`, `./N` or `.//N` (§3.2.4),
// as the whole operand of fn:exists, fn:empty or an ·effective boolean value·,
// whether it selects a node of E's subtree being what a [Tally] carries
// whatever the selected elements' types (ctaFacade.childPath,
// ctaFacade.elements); an fn:count call (xpath-functions.md §15.4.1) in [14]
// ta-ValueExpr's position, over one counted path of E's subtree, whose counts
// a [Tally] carries, or over an operand that is no path, such as `$value`,
// whose items it counts; inside that call's argument, xpath20.md [40] Predicate,
// one on a child step, testing the existence of the child's attributes, which
// the Tally reads too, or comparing the child's own value, read off
// [ChildElements] (§3.2.2); and [21] UnionExpr over such operands (§3.3.3),
// each node counted once; and, in [14]'s position, a call to one of the F&O
// string and sequence functions [CompileAssertionTest] lists
// (ctaFacade.callsLibrary, ctafunc.go), whose argument may also be the empty
// sequence `()`. It also admits xpath20.md §3.4's binary arithmetic operators
// over numeric operands (ctaFacade.computes), in each comparison operand's
// position; §3.8's [7] IfExpr wherever an ExprSingle stands whole in a boolean
// position (ctaFacade.conditional); [47] ContextItemExpr `.` over simple
// content, as E's string value (ctaAssertionFacade.contextItem); and, as a
// general comparison's operand, §3.3.1's integer sequences
// (ctaFacade.constructsSequences).

// AttributeTypes answers, for the element information item E whose assertions
// are being compiled, the {type definition} an attribute of E with the
// ·expanded name· name is typed by: the type of the {attribute declaration} of
// the {attribute use} of E's ·governing type definition· matching that name. ok
// false means no type is fixed for the name at compile time — no such use, a
// name only an {attribute wildcard} can admit, or a use whose declaration or
// type does not resolve — and [CompileAssertionTest] declines a {test} naming
// it.
//
// It is STATIC in the sense the compile needs: the answer does not depend on
// whether E carries the attribute, or a ·defaulted attribute· (key-dflt-att)
// stands in for it, both of which [TypedAttributes] yields. Its one consumer
// is validate's cvc-assertion site (validate/cvcassertion.go), which builds
// it and [TypedAttributes] from one lookup so the two cannot disagree.
type AttributeTypes func(name xsd.QName) (*xsd.SimpleType, bool)

// ElementTypes answers, for the element information item E whose assertions are
// being compiled, the {type definition} a child element of E with the ·expanded
// name· name is read under: its ·locally declared type· within E's ·governing
// type definition· (key-ldt-elem), the type cvc-complex-type clause 5 requires
// that child's own ·governing type definition· to be the same as or ·validly
// substitutable· for. ok false means no type is fixed for the name at compile
// time — the type is ·absent· — and [CompileAssertionTest] declines a {test}
// naming it.
//
// It is STATIC, on [AttributeTypes]' terms: the answer does not depend on
// which children E carries, or on the type an xsi:type gives one, which is why
// the type the compile reads and the one a child is governed by can differ and
// [ChildElements] states which values agree with it. Its one consumer is
// validate's cvc-assertion site (validate/cvcassertion.go).
type ElementTypes func(name xsd.QName) (xsd.TypeDefinition, bool)

// ChildElement is one element [[child]] of E as an assertion {test} reads it:
// its ·expanded name· and its typed value (xpath-datamodel §6.2.4). It is
// opaque, so its one constructor is [Child].
type ChildElement struct {
	name xsd.QName
	v    TypedValue
}

// Child is the child element named name whose typed value is v, on the terms
// [ChildElements] states. Child(name, nil) is a ·nilled· child, whose typed
// value is the empty sequence whatever its type (xpath-datamodel §3.3.1.2;
// xpath20.md §2.5.2 item 4.1 for mixed content): a node all the same, which a
// {test} naming it finds.
func Child(name xsd.QName, v TypedValue) ChildElement { return ChildElement{name: name, v: v} }

// ChildElements yields E's element [[children]] a compiled {test} reads, in
// DOCUMENT ORDER (STYLE D1), each as a [ChildElement]. It must be non-nil, and
// a yield reporting false ends the walk. A child no compiled step names need
// not be yielded: [AssertionTest.ReadsChild] reports which names one reads.
//
// Each value's arm is fixed by the type [ElementTypes] answered for its name
// when the [AssertionTest] being evaluated was compiled; a ·nilled· child's is
// nil whatever that type. A complex type whose {content type}.{variety} is
// mixed — xs:anyType among them — gives [Untyped] of the child's string-value
// (xpath-datamodel §6.2.4), which spans the text of all its descendants and
// not only its own. Any other type gives [Typed] of a value of EXACTLY the
// simple type the compile read off it — that type itself, or the {simple type
// definition} of its simple {content type}. That agreement is the caller's
// obligation, on the terms [TypedAttributes] states for an attribute's value:
// validate maps the child's [schema normalized value] under its OWN
// ·governing type definition· into that simple type, takes a string-value only
// where that governing type has mixed content too, and withholds the
// assertion where the child's own type cannot give the arm the compile read.
// A value breaking it — the other arm — is a dynamic error wherever the tree
// reads it, which [AssertionTest.Evaluate] answers false.
//
// It carries the children whose typed values a compiled step reads and nothing
// below them: those a value step names, and those an fn:count argument filters
// by a predicate over their value, `count(N[. = 'x'])`, which is counted over
// them here. A node a {test} only counts — a child, a descendant, an
// attribute, a child filtered by its attributes' existence — or only asks the
// existence of through a path, `a`, `.//a` or `a/b`, is reported to the
// [Tally] instead, and never yielded here for that sake.
type ChildElements func(yield func(ChildElement) bool)

// TypedValue is the typed value of one attribute node, of one element node of
// simple type, simple content or mixed content (xpath-datamodel §6.2.4), or of
// `$value`, in the data model instance cvc-assertion clause 1 builds: an
// ·actual value· of a type fixed at compile time ([Typed]), or an
// xs:untypedAtomic value ([Untyped]) — the [schema normalized value] under
// xs:anySimpleType or xs:anyAtomicType (xpath-datamodel §3.3.1.2), or an
// element's string-value under mixed content (§6.2.4). Datatypes dt-xdmrep
// clause 1 gives a ·special· type's value that dynamic type. It is a sealed
// sum of exactly those two arms, so no value carries both a lexical and a
// typed value (STYLE T1).
type TypedValue interface{ typedValue() }

// tvTyped is [Typed]'s arm: an ·actual value· of the type the tree holds.
type tvTyped struct{ v value.Value }

// tvUntyped is [Untyped]'s arm: an xs:untypedAtomic value, by its lexical.
type tvUntyped struct{ lexical string }

func (tvTyped) typedValue()   {}
func (tvUntyped) typedValue() {}

// Typed is the typed value v, an ·actual value· mapped under a type whose
// typed value is not xs:untypedAtomic. Typed(nil) is nil, so a nil value has
// one encoding.
func Typed(v value.Value) TypedValue {
	if v == nil {
		return nil
	}
	return tvTyped{v: v}
}

// Untyped is lexical as an instance of xs:untypedAtomic. lexical must be what
// the data model makes that typed value of: the [schema normalized value] of
// an attribute, an element or `$value` whose type is xs:anySimpleType or
// xs:anyAtomicType (xpath-datamodel §3.3.1.2), or the string-value of an
// element whose type is a complex type with mixed content (§6.2.4).
func Untyped(lexical string) TypedValue { return tvUntyped{lexical: lexical} }

// TypedAttributes yields E's attributes that matched an {attribute use} of its
// ·governing type definition·, each as its ·expanded name· and its typed
// value: those E carries, in DOCUMENT ORDER (STYLE D1), then its ·defaulted
// attributes· (key-dflt-att), which the partial ·PSVI· cvc-assertion clause
// 1.2 builds from holds too, in an order the caller fixes. It must be non-nil,
// and a yield reporting false ends the walk.
//
// Each value's arm is fixed by the type [AttributeTypes] answered for its name
// when the [AssertionTest] being evaluated was compiled: [Untyped] of the
// attribute's [schema normalized value] exactly when that type is ·special· —
// xs:anySimpleType or xs:anyAtomicType — and [Typed] of a value of EXACTLY that
// type otherwise, which the tree holds and converts the value from. That
// agreement is the caller's obligation, and it is the same one [BindValue]
// places on `$value`'s value against the {simple type definition} of the
// content type the test was compiled for. A value breaking it — nil, or the
// other arm — is a dynamic error wherever the tree reads it, which
// [AssertionTest.Evaluate] answers false.
//
// Unlike [Attributes], it carries no [inherited attributes]: the XDM instance
// cvc-assertion clause 1.3 builds contains E's own [[attributes]] and nothing
// from outside E.
type TypedAttributes func(yield func(name xsd.QName, v TypedValue) bool)

// ValueBinding is what one evaluation reads of E itself, two independent facts
// [BindValue] binds: E's string value, which the [47] ContextItemExpr `.`
// atomizes to under an [xsd.SimpleContent], and the value cvc-assertion clause
// 2.3 binds to `$value`.
//
// The string value is always one xs:untypedAtomic value: E's annotation in the
// partial ·PSVI· cvc-assertion clause 1.2 builds is xs:anyType
// (xpath-datamodel §3.3.1.1, [validation attempted] partial), a complex type
// with mixed content, whose typed value is its string value as xs:untypedAtomic
// (xpath20.md §2.5.2) — "its atomized value will be a single atomic value of
// type untypedAtomic", as clause 2.3.1's Note says, whatever the {simple type
// definition}'s variety. It has no absent state: an empty E's string value is
// the zero-length string (xpath-datamodel §6.2.4), which is xs:untypedAtomic ""
// and never the empty sequence. A ·nilled· E is no exception either way:
// dm:nilled is false in the partial ·PSVI·, whose [validity] is never valid, so
// a ·nilled· E with [[children]] has their text as its string value, not "".
//
// A nil `$value` is clause 2.3.2's empty sequence — for an E that is invalid
// in the partial ·PSVI·, ·nilled·, or under a {content type} that is not
// simple. The zero ValueBinding is BindValue("", nil), what a ·nilled· E with
// no [[children]] binds; an invalid E binds its string value beside a nil
// `$value`.
//
// It has no third state for an E whose value is UNDECIDED: [AssertionTest.Evaluate]
// always decides, so a caller that cannot tell which of clause 2.3's cases E is
// in declines the assertion itself, as it does an attribute with no ·actual
// value·.
type ValueBinding struct {
	text string
	v    TypedValue
}

// BindValue binds E's string value text, which `.` reads, and `$value`'s
// value v, on the terms [ValueBinding] states. Each fact is the caller's to
// supply, and neither is derived from the other.
//
// text is E's string value as the data model instance holds it: the
// concatenation of its text node children (xpath-datamodel §6.2.4), which
// Appendix J.2's children rule builds from the character [[children]] of E in
// document order — its ·initial value· — or, where cvc-elt clause 5.1
// supplied a default because E has neither element nor character
// [[children]], the {value constraint}'s {lexical form}, the one text node
// J.2's "may" lets the data model build from the [schema normalized value] of
// a defaulted element. Each is a processor's choice J.2 leaves open, and
// validate's caller makes it (the GAP(xpath) at its cvc-assertion site,
// validate/cvcassertion.go). It is never re-normalized under the {simple type
// definition}'s whiteSpace: `.` over xs:integer content "0030" is "0030",
// where `$value` is 30. An invalid E has a string value all the same, so text
// is bound where v is nil.
//
// v is the typed value of E's [schema actual value] (cvc-assertion clause
// 2.3.1), whose arm and type the {simple type definition} of the
// [xsd.SimpleContent] the [AssertionTest] was compiled for fixes, on the terms
// [TypedAttributes] states for an attribute's value: [Untyped] of E's [schema
// normalized value] where it is ·special· — the representation of that value
// xpath-datamodel §3.3.1.2 fixes, a ·special· type's lexical mapping not being
// a function (Datatypes §3.2.1.2) — and [Typed] of a value of exactly it
// otherwise. A list {simple type definition}'s value must carry
// [value.Listed], whose items are the flattened sequence Datatypes dt-xdmrep
// makes its XDM representation. A nil v is the empty sequence of clause 2.3.2,
// so the empty sequence has one encoding.
func BindValue(text string, v TypedValue) ValueBinding { return ValueBinding{text: text, v: v} }

// Tally is the node-count input of ONE evaluation of ONE [AssertionTest] over
// the element E: for each relative path the {test} counts over with fn:count
// (xpath-functions.md §15.4.1) — a step, a child step filtered by the
// existence of its attributes, or a union of those (xpath20.md §3.2.2,
// §3.3.3) — or asks the existence of with fn:exists, fn:empty or an ·effective
// boolean value· (§15.1.4, §15.1.5, xpath20.md §2.4.3), how many nodes of E's
// subtree it selects. A child step filtered by its VALUE is not among them:
// [ChildElements] answers it. [AssertionTest.Tally] makes one, the caller
// reports E's subtree to it while that subtree streams past, and
// [AssertionTest.Evaluate] reads it. It keeps one counter per distinct path
// and no node, so what the caller holds for a count is one integer per path
// whatever the subtree's size.
//
// The caller's obligation is to report EVERY node of E's subtree in the data
// model instance cvc-assertion clause 1 builds, exactly once, in any order, by
// one [Tally.Element] call per element, E itself included: the element, by the
// chain of ·expanded names· from E's child down to it — the empty chain for E —
// whatever its validity, whether it is ·nilled·, and whether it was ·strictly·
// or ·laxly assessed· or ·skipped·, with the names of ALL its attribute nodes:
// those it carries, xsi:type and the other xsi attributes among them, and its
// ·defaulted attributes· (key-dflt-att), which the partial ·PSVI· clause 1.2
// builds from holds too; never a namespace declaration, which is not an
// attribute node.
//
// A ·skipped· element and everything below it is governed by no type
// (key-skipped, key-governing-type-elem item 5), so it has no ·defaulted
// attribute·: its attribute nodes are exactly those it carries, xsi ones
// included. An element whose ·governing type definition· the caller could not
// determine has ·defaulted attributes· it cannot know, and may be reported
// with no attribute names only at a depth where [Tally.CountsAttributesAt] is
// false; a caller that cannot report a subtree exactly otherwise declines the
// assertion itself, on the terms [ValueBinding] states for an undecided
// `$value`. Under-reporting would make a count too small and could fabricate a
// charge.
//
// Its consumer is validate's cvc-assertion site (validate/cvcassertion.go),
// which reports each element it walks, and each element of a ·skipped· subtree
// by name, to the Tally of every enclosing element's {assertions} that has one.
type Tally struct{ counters []ctaCounter }

// ctaCounter is one counter of a [Tally]: the path it counts and how many of
// the nodes reported so far that path selects.
type ctaCounter struct {
	path ctaTallied
	n    int
}

// Element reports one element of E's subtree, by path, and its attribute
// nodes, by attrs. path is the ·expanded names· of the elements from E's child
// down to the reported one inclusive, so len(path) is its depth below E — 0
// for E itself, 1 for a child, 2 for a grandchild, and so on. attrs is the
// names of ALL its attribute nodes, carried or ·defaulted·, on the terms
// [Tally] states. Neither slice is retained: both are read during the call.
//
// Every path the {test} counts selecting the element counts it — `N` where
// path is [N], `.//N` where path ends in N at any depth, since xpath20.md
// §3.2.4 makes `.//N` `./descendant-or-self::node()/child::N`, which never
// selects E itself, and `N1/N2/…` where path is exactly its steps (§3.2) — and
// every path selecting one of its attribute nodes counts that node: `@N` at
// depth 0 only, and `.//@N` at every depth, E's own included, since §3.2.4
// makes `.//@N` `./descendant-or-self::node()/attribute::N`. The empty path
// is E, whose element node no path selects. Every report to a nil Tally
// selects nothing.
func (c *Tally) Element(path []xsd.QName, attrs []xsd.QName) {
	if c == nil {
		return
	}
	for i := range c.counters {
		c.counters[i].n += ctaSelected(c.counters[i].path, path, attrs)
	}
}

// ctaSelected is how many of the nodes one [Tally.Element] report carries — the
// element at path, and its attribute nodes attrs — key selects, each node
// counted once.
func ctaSelected(key ctaTallied, path, attrs []xsd.QName) int {
	n := 0
	if key.selectsElement(path, attrs) {
		n++
	}
	for _, a := range attrs {
		if key.selectsAttribute(len(path), a) {
			n++
		}
	}
	return n
}

// CountsAttributesAt reports whether the attribute names reported for an
// element depth levels below E, 0 being E, can change any count c holds: `@N`
// selects attribute nodes at depth 0 only, and `.//@N` at every depth from 0.
// It is false for a nil Tally and for a depth below 0, where no [Tally.Element]
// report stands.
//
// Its consumer is validate's walk (validate/cvcassertion.go), which asks it of
// an element whose ·defaulted attributes· it cannot know: where it is false,
// no attribute name reported for that element could change any count, and the
// element is reported with none.
func (c *Tally) CountsAttributesAt(depth int) bool {
	if c == nil {
		return false
	}
	for _, counter := range c.counters {
		if counter.path.selectsAttributesAt(depth) {
			return true
		}
	}
	return false
}

// count is the counter c holds for path, false where it holds none.
func (c *Tally) count(path ctaTallied) (int, bool) {
	if c == nil {
		return 0, false
	}
	for _, counter := range c.counters {
		if counter.path.same(path) {
			return counter.n, true
		}
	}
	return 0, false
}

// fits reports whether c holds exactly one counter for each of paths, in that
// order — which a Tally [AssertionTest.Tally] made from the test paths was
// read off holds — and is nil where paths is empty, since a test that counts
// nothing has no Tally and any non-nil c given to it is a breach.
func (c *Tally) fits(paths []ctaTallied) bool {
	if len(paths) == 0 {
		return c == nil
	}
	var held []ctaCounter
	if c != nil {
		held = c.counters
	}
	if len(held) != len(paths) {
		return false
	}
	for i, counter := range held {
		if !counter.path.same(paths[i]) {
			return false
		}
	}
	return true
}

// AssertionTest is a compiled assertion {test}: the expression tree
// [CompileAssertionTest] admitted for one element's attribute and child
// element types. It is a distinct type from [CTATest] so a tree typed for an
// assertion can never be evaluated over a Type Alternative's [Attributes],
// nor a Type Alternative tree over [TypedAttributes].
type AssertionTest struct{ root ctaExpr }

// CompileAssertionTest compiles an assertion's {test} (§3.13.1, an
// [xsd.XPathExpression] property record) for the element whose ·governing
// type definition· has the {content type} content, whose attribute types
// attrs answers and whose child element types elems answers, reporting ok
// false for a {test} this engine cannot evaluate. Its consumer is
// validate's cvc-assertion site (validate/cvcassertion.go), which declines
// the assertion on ok false.
//
// content fixes `$value`'s static type, which is what cvc-assertion clause
// 2.3.1.3 reads: under an [xsd.SimpleContent] it is the {simple type
// definition}, whose value [BindValue] binds at evaluation; under every other
// {content type} it is the empty sequence (clause 2.3.2), whatever the binding.
// A nil content declines every {test} naming `$value`. content also decides
// `.`: under an [xsd.SimpleContent] it is E's string value, one xs:untypedAtomic
// value [BindValue] binds beside `$value`, whatever the {simple type
// definition}'s variety, and under every other {content type}, and a nil one,
// it declines.
//
// The grammar is [CompileCTATest]'s with the value comparisons, `$value`, an
// abbreviated child-axis step, a "/" or "//" opening one child or attribute
// step, a relative path of two or more abbreviated child-axis steps with QName
// NameTests, `a/b`, or one element step, `a`, `./a` or `.//a`, standing as the
// whole operand of fn:exists, fn:empty or an ·effective boolean value·
// (xpath20.md §3.2, §3.2.4, §2.4.3), an fn:count call, whose argument may
// filter a child step by one predicate or join operands with `|` or `union`
// (§3.2.2, §3.3.3; the bullets below bound both), or be an operand that is no
// path — `$value`, `()`, a literal or a function call, read as any argument of
// the functions below is — the binary arithmetic operators `+`, `-`, `*`,
// `div`, `idiv` and `mod` (xpath20.md §3.4), and a call to one of the F&O
// string and sequence functions — fn:contains, fn:starts-with and fn:ends-with
// with two arguments, fn:string-length, fn:normalize-space and fn:string with
// one or none, the implicit argument being `.`, fn:empty, fn:exists and
// fn:distinct-values with one, and fn:true, fn:false and fn:current-date with
// none (xpath-functions.md §7.5.1–7.5.3, §7.4.4, §7.4.5, §2.3, §15.1.4,
// §15.1.5, §15.1.6, §9.1.1, §9.1.2, §16.4), any argument of which may be the
// empty sequence `()` — the conditional `if (Expr) then ExprSingle else
// ExprSingle` (xpath20.md §3.8) as the whole {test}, inside parentheses, as
// fn:not's argument or as an operand of another, whose test's ·effective
// boolean value· selects the one branch evaluated, so a dynamic error in the
// other is never raised, and both of whose branches are compiled, so a decline
// in either declines the {test}; the [47] ContextItemExpr `.` over simple
// content, atomized (§3.1.4, §2.4.2); and, as an operand of a general
// comparison, an integer sequence: [11] RangeExpr `I to J` over two
// IntegerLiterals, bare or parenthesized, or a parenthesized comma sequence of
// IntegerLiterals and such ranges, `(1 to 10, 20, 30)` (§3.3.1) — added, and
// every decline [CompileCTATest] states is this one's too, under the same
// static context (xpath-valid clause 2.2) augmented with `$value`
// (cvc-assertion clause 2.2), plus these, each of which is the same withhold:
//
//   - an attribute NameTest that is not a QName: a [37] Wildcard can match an
//     attribute ·attributed to· an {attribute wildcard}, whose type is not
//     fixed at compile time;
//   - a QName for which attrs reports false;
//   - an attribute whose type is not ·special· and does not atomize to one
//     atomic value of a type known at compile time — a list or union {variety}
//     — or whose {primitive type definition} is xs:QName or xs:NOTATION, which
//     carry no ·canonical representation· to convert through. The variety is
//     classified here, not trusted to attrs. An attribute whose type is
//     ·special· is read as xs:untypedAtomic and is admitted;
//   - a `$value` over an [xsd.SimpleContent] whose {simple type definition} the
//     bullet above declines as an attribute's type, unless it is a list whose
//     {item type definition} that bullet admits — so a union, a list of a
//     union, and an xs:QName or xs:NOTATION primitive or item type decline. A
//     ·special· one is read as xs:untypedAtomic and is admitted;
//   - any variable but `$value`, which is not in the static context at all
//     (err:XPST0008);
//   - a child-axis step whose NameTest is not a QName, or spells its axis out,
//     and, where the step's value is read — anywhere but as the whole operand
//     of fn:exists, fn:empty or an ·effective boolean value· — one naming a
//     child for which elems reports false;
//   - a child read for its value whose type elems answers is none of: a simple
//     type the bullets above admit as an attribute's, a complex type whose
//     simple {content type} is one, or a complex type whose {content
//     type}.{variety} is mixed — xs:anyType among them — which is read as
//     xs:untypedAtomic. A ·special· simple type declines, because an xsi:type
//     can give the child a typed value where the compile read an
//     xs:untypedAtomic one; a mixed one does not, because the caller withholds
//     a child whose own type has no mixed content ([ChildElements]). Every
//     element-only and empty {content type} declines;
//   - a path of more than one step anywhere but as the whole operand of
//     fn:exists, fn:empty or an ·effective boolean value·, and there any step
//     on another axis, with a wildcard, a predicate or a kind test, and any
//     "//" between steps — so `a/b = 1`, `count(a/b)`, `string(a/b)`, `a/@b`,
//     `a//b` and `a/*` decline — a rooted path of more than one step, and a
//     "/" with no step after it;
//   - an fn:count argument that is not one QName step — `N`, `@N`, either of
//     them behind `./` or `.//`, or a rooted one — nor such a child step `N`
//     or `./N` filtered by one predicate the next two bullets admit, nor a `|`
//     or `union` of relative such operands, nor an operand that opens as no
//     path does — `$`, `(`, a literal, or a name followed by `(` — and that a
//     function argument admits — so a wildcard, a longer path, a bare `.`, a
//     union with `$value`, a predicate on any other step, a second predicate,
//     and a union operand that is rooted or filtered by its value decline, and
//     so does a step behind `./` or `.//` anywhere but in an fn:count call or
//     as the whole operand of fn:exists, fn:empty or an ·effective boolean
//     value·, and a predicate or a union anywhere but in an fn:count argument;
//   - a predicate that is not a conjunction of attribute-existence tests with
//     QName NameTests, `N[@A]` or `N[@A1 and @A2 …]`, nor a comparison, or
//     and, or and fn:not over comparisons, of the candidate `.`, literals,
//     casts and arithmetic — so `N[@a or @b]`, `N[not(@a)]`, `N[@*]`, an
//     attribute atomized or beside `.`, any other node, `$value`, fn:count and
//     every other function call decline — and a predicate whose root is a bare
//     value, `N[1]`, `N[1 + 0]` or `N[.]`, which a numeric value would make
//     positional (§3.2.2); position() and last() are no library function here;
//   - a predicate over a child whose type elems answers is not one a child
//     read for its value is admitted under as a TYPED value, on that bullet's
//     terms — so a predicate over a mixed child, `count(body[. = 'x'])`,
//     declines — unless the predicate tests attribute existence alone;
//   - a cast whose operand is a typed attribute, a typed child or `$value`
//     outside the xs:string family, to a target that operand's type is neither
//     nor derived from by restriction (F&O §17.4, §17.1, §17.5) — so
//     `xs:integer(@d)` over an xs:decimal @d declines and `xs:decimal(@i)`
//     over an xs:int @i, §17.3's cast, does not, nor does a cast of an
//     xs:date, xs:dateTime or xs:time operand to xs:string itself, §17.1.2's
//     local value, while one to xs:token declines — and a cast of an xs:float
//     or xs:double operand, a DoubleLiteral or (as fn:string's argument) a
//     cast to either, to any target but its own type (F&O §17.1.2, §17.1.3
//     and §17.1.6 over the value where §17.1's casting table marks the target
//     Y or M, §17.1's err:XPTY0004 where it marks N), so `xs:string(1.5e0)`
//     and `xs:date(1.5e0)` decline and `xs:double(1.5e0)` does not — and a
//     cast of an IntegerLiteral or a DecimalLiteral to any target but one
//     whose primitive is xs:string, xs:float or xs:double, xs:decimal, and for
//     an IntegerLiteral xs:integer's branch (F&O §17.1.3.4 and §17.1.6 over
//     the value, §17.1's err:XPTY0004), so `xs:integer(1.5)`,
//     `xs:boolean(2)` and `xs:anyURI(1.5)` decline and `xs:integer(2)` and
//     `xs:string(1.5)` do not;
//   - an arithmetic operand whose {primitive type definition} is not
//     xs:decimal, xs:float or xs:double — so the duration and date/time
//     arithmetic B.2 defines declines, and so does the err:XPTY0004 of any
//     other type — and an xs:float operand against an xs:double or
//     xs:untypedAtomic one, which needs B.1 rule 1.1 (#889); a unary `+` or
//     `-`, and a parenthesized operand, which [11]'s `(` arm reads as a
//     boolean expression;
//   - a conditional anywhere but in the positions listed above — so a
//     function's argument, `contains(if (@a) then 'x' else 'y', 'x')`, and
//     a parenthesized comparison or arithmetic operand,
//     `(if (@a) then 1 else 2) = 1`, decline — and a conditional inside a
//     predicate;
//   - a call to fn:contains, fn:starts-with or fn:ends-with with a third,
//     collation argument, or to fn:distinct-values with a second (§7.3.1),
//     which is never read as the form without it, and a call to any of the
//     functions above with an arity it does not have (err:XPST0017);
//   - `.` under a {content type} that is not simple, whose string value is the
//     text of E's descendants, and `.` as the whole operand of an ·effective
//     boolean value·, fn:not over one, fn:exists, fn:empty or fn:count, where it
//     is a node and not the atom it is read as elsewhere — so `.`, `not(.)`,
//     `exists(.)` and `count(.)` decline — and fn:string-length,
//     fn:normalize-space and fn:string with no argument wherever `.` declines;
//   - an integer sequence anywhere but as a general comparison's operand —
//     `. eq (1 to 3)`, `(1 to 3) + 1` — a range operand that is not an
//     IntegerLiteral, `1 to .`, a member that is neither, a nested or empty
//     parenthesis, an IntegerLiteral beyond int64, and a sequence of more than
//     ctaMaxSequenceLength items;
//   - fn:string over a typed attribute, a typed child, an fn:count call, an
//     arithmetic result, `$value`, a function result or a cast of one of them
//     outside the xs:string family and the xs:date, xs:dateTime and xs:time
//     primitives, which is the cast to xs:string the bullet above declines,
//     and fn:string over any argument whose {primitive type
//     definition} is xs:float or xs:double, a literal or a cast included,
//     which is the cast to xs:string that bullet's floating clause declines.
//
// An xs:string? argument — of every function above but fn:empty, fn:exists,
// fn:distinct-values and fn:string — of any type outside the xs:string and
// xs:anyURI families is not a decline: xpath20.md §3.1.5's function conversion
// raises err:XPTY0004 for each item it yields, an absent attribute's empty
// sequence being the zero-length string, and so does a `$value` of two or more
// items.
//
// A counted step consults neither attrs nor elems: fn:count does not atomize
// its argument (xpath-functions.md §15.4.1, `$arg as item()*`), so the step's
// type decides nothing and a node of any type, or matched by a wildcard,
// counts. Nor does a child path of two or more steps, or one element step, as
// the whole operand of fn:exists, fn:empty or an ·effective boolean value·, on
// the same terms: fn:exists and fn:empty take `item()*` (§15.1.4, §15.1.5) and
// an ·effective boolean value· asks only whether the first item is a node
// (xpath20.md §2.4.3 rule 2), so a node of any type, ·nilled· or not, is
// selected — `a` over an element-only a is decided where `a = 1` declines. Nor
// does an attribute-existence predicate, whose ·effective boolean value· asks
// only whether the attribute node exists (§2.4.3 rule 2), so `c[@a]` counts a
// c carrying an empty a. What the {test} counts is read off the [Tally] its
// evaluation carries ([AssertionTest.Tally]). A predicate reading `.` is the
// one counted argument elems types: it atomizes each candidate child, typed as
// a child step naming it reads a TYPED value, and counts over [ChildElements];
// a dynamic or type error over any candidate raises for the whole count. An
// argument that is no path is counted off its own items, never the [Tally], on
// fn:exists' terms: `count($value)` is the number of items of `$value` — one
// for an atomic or ·special· value, each item of a list (Datatypes dt-xdmrep
// clause 3), and none for an empty list or the empty sequence clause 2.3.2
// binds.
//
// fn:distinct-values atomizes its argument and drops each item eq to an
// earlier one (xpath-functions.md §15.1.6), compared in the items' {primitive
// type definition}: under the codepoint collation, so "a" and "A" are two; at
// the implicit timezone, Z, for a date or time without one; with 0 and -0 one
// value and NaN one item although `NaN eq NaN` is false; an xs:untypedAtomic
// item as xs:string; and two items eq does not relate as two, never an error.
// Which of two equal items survives is the first.
//
// An XPath STATIC error is declined too and never reported: the
// static-error question about an assertion is the schema assembler's, and
// [CTATestStaticError] answers it for a Type Alternative only.
//
// A leading "/" or "//" followed by one child or attribute step is not a
// decline, inside an fn:count call or outside one: E is the root of the data
// model instance, with no document node above it (cvc-assertion clause 1.3),
// so the path raises err:XPDY0050 (xpath20.md §3.2) before it selects a node,
// and the {test} is false whatever E is named and whatever its subtree holds.
//
// GAP(xpath): unlike a Type Alternative's, an assertion's {test} has no
// required subset to stop at — §3.13 admits full XPath 2.0 — so every decline
// above is this engine's limit and not the spec's license: paths of more than
// one step outside fn:exists, fn:empty and an ·effective boolean value·, axes
// beyond the attribute step, one child step, the child steps of such a path,
// the one element step whose existence is asked and the one counted step,
// predicates beyond the two kinds on a counted child step, positional ones
// among them, unions outside fn:count or over other operands, children read
// for their value whose type is element-only, empty or ·special·, a value
// predicate over a mixed child, arithmetic outside the numeric operands and
// the binary operators, conditionals whose value is read as an item rather
// than for its ·effective boolean value·, `.` where E's string value is not an
// input or `.` is a node, sequence expressions beyond the integer sequences of
// a general comparison's operand, the collation argument, and every F&O
// function but fn:count and those listed above among them. The direction is
// the withhold: the caller records the assertion as unevaluated and neither
// charges it nor shows it satisfied (PRINCIPLES 20). (#1042)
//
// types is read as [CompileCTATest] reads it and stored nowhere.
func CompileAssertionTest(expr xsd.XPathExpression, types xsd.TypeResolver, content xsd.ContentType, attrs AttributeTypes, elems ElementTypes) (AssertionTest, bool) {
	root, defect := compileCTATest(expr, types, ctaAssertionFacade{content: content, attrs: attrs, elems: elems})
	if defect.kind != ctaNoDefect {
		return AssertionTest{}, false
	}
	return AssertionTest{root: root}, true
}

// Evaluate reports whether the compiled {test} evaluates to true for the
// element whose attributes attrs yields, whose element [[children]] children
// yields, whose subtree counts has counted and whose string value and `$value`
// v binds ([ValueBinding]), WITHOUT raising a dynamic or type error — the
// whole of what cvc-assertion (§3.13.4.1) asks: "An element information item E
// is locally ·valid· with respect to an assertion if and only if the {test}
// evaluates to true (see below) without raising any dynamic error or type
// error." Clause 3 converts the result "as if by a call to the XPath
// fn:boolean function", which a boolean-rooted tree already is.
//
// So false is ONE answer for two outcomes the caller treats alike — the {test}
// was false, or it raised (err:FORG0001, err:XPTY0004, which a function
// argument's conversion or cardinality raises too, err:FORG0006,
// err:XPDY0050, err:FOAR0001 for a `div`, `idiv` or `mod` by zero over
// xs:decimal or xs:integer operands and for any `idiv` by zero, err:FOAR0002
// for an `idiv` over a NaN operand or an infinite dividend or whose quotient
// overflows to an infinity) — and either way E is not ·valid· with respect to the
// assertion. A processor that raises a type error dynamically "will treat the
// expression as having evaluated to false" (cvc-xpath, §3.13.4.2). Every
// decline happened at [CompileAssertionTest].
//
// The dynamic context is cvc-xpath's — context item E, position and size 1 —
// with the one variable cvc-assertion clause 2.3 adds to it, `$value`, bound
// to v's typed value; `.` atomizes to v's string value. A tree compiled for a
// {content type} that is not simple never reads v. b and types are read as
// [CTATest.Evaluate] reads them.
//
// now is the dynamic context's current dateTime (xpath20.md §2.1.2), which
// fn:current-date reads as the xs:date of now in now's own UTC offset
// (xpath-functions.md §16.4, §17.1.5); the implicit timezone stays Z whatever
// now's offset is. An offset no timezoneFrag can spell — not a whole number of
// minutes, or outside -14:00 to +14:00 (xmlschema11-2 timezoneFrag) — is
// replaced by UTC: the date is that of now.UTC(), with timezone Z. cvc-xpath
// clause 6 makes the current dateTime constant during an assessment episode,
// so the caller hands every evaluation of one episode the same instant. Its
// consumer is validate, which reads the clock once per Validator.Assess and
// passes that instant here and to [FacetAssertions].
//
// counts is the [Tally] t.Tally() made for this evaluation, filled with E's
// subtree on the terms that type states, and nil exactly where t.Tally() is
// nil. Any other counts — a nil one where t counts, a non-nil one where it
// counts nothing, or one made by a test counting other paths — breaks that
// obligation, on the terms [TypedAttributes] states for an attribute's value,
// and Evaluate answers false for it whatever the tree holds. A Tally made by
// another test counting the same paths cannot be told from t's own and is read
// as one; handing over the one filled for this E is the caller's part.
func (t AssertionTest) Evaluate(b value.Backend, types xsd.TypeResolver, attrs TypedAttributes, children ChildElements, counts *Tally, v ValueBinding, now time.Time) bool {
	if !counts.fits(t.countedPaths()) {
		return false
	}
	return ctaEval(t.root, ctaEnv{backend: b, types: types, input: ctaTypedInput{attrs: attrs, children: children, counts: counts, value: v, now: now}}) == ctaTrue
}

// Tally is a fresh, empty [Tally] for one evaluation of t, holding one counter
// for each distinct relative path an fn:count call in t counts over — a step,
// a child step filtered by attribute existence, or a union, one counter
// serving `c[@a and @b]` and `c[@b and @a]`, and `e | e` and `e` — and each
// distinct path — one element step or a child path of two or more steps —
// whose existence t asks (fn:exists, fn:empty, an ·effective boolean value·),
// one counter serving a path both counted and asked, or nil where t counts
// over none — a {test} with neither, or one whose every fn:count argument is
// rooted and raises, is a child step filtered by its value, which
// [ChildElements] answers, or is no path and counts none, as `count($value)`
// counts the items of `$value`. t itself is not changed, so one compiled test
// serves any number of evaluations, each with its own Tally.
//
// Its consumer is validate's walk (validate/cvcassertion.go), which reads the
// nil as its gate: only an element one of whose {test}s has a Tally reports its
// subtree to one, and which children a value step reads stays
// [AssertionTest.ReadsChild]'s answer alone.
func (t AssertionTest) Tally() *Tally {
	paths := t.countedPaths()
	if len(paths) == 0 {
		return nil
	}
	c := &Tally{counters: make([]ctaCounter, len(paths))}
	for i, path := range paths {
		c.counters[i].path = path
	}
	return c
}

// countedPaths is each distinct path t counts over — an fn:count argument, or
// an element step or child path whose existence it asks — in written order,
// read off the tree itself.
func (t AssertionTest) countedPaths() []ctaTallied {
	if t.root == nil {
		// The zero AssertionTest, which no successful CompileAssertionTest
		// produces, holds nothing.
		return nil
	}
	var distinct []ctaTallied
	for _, path := range t.root.counted(nil) {
		if !ctaHoldsPath(distinct, path) {
			distinct = append(distinct, path)
		}
	}
	return distinct
}

// ctaHoldsPath reports whether paths holds a key the same as path.
func ctaHoldsPath(paths []ctaTallied, path ctaTallied) bool {
	for _, p := range paths {
		if p.same(path) {
			return true
		}
	}
	return false
}

// ReadsChild reports whether the compiled {test} holds a child-axis step that
// selects a child element named name for its VALUE, which is what
// [AssertionTest.Evaluate]'s [ChildElements] must yield: a value step, or a
// child step an fn:count argument filters by a predicate over its value,
// `count(N[. = 'x'])`, which is counted over the children yielded. Its
// consumer is validate's walk, which keeps a child's value only where some
// {test} of its parent reads it. Any other fn:count path reads no value and is
// no read here — an argument that is no path reads what its operand reads,
// `count($value)` nothing — nor is an element step or a child path of two or more
// steps whose existence alone is asked, whatever the type of the child it
// names: what each counts is [AssertionTest.Tally]'s. The answer is read off
// the tree itself.
func (t AssertionTest) ReadsChild(name xsd.QName) bool {
	if t.root == nil {
		// The zero AssertionTest, which no successful CompileAssertionTest
		// produces, holds nothing.
		return false
	}
	return t.root.readsChild(name)
}

// readsChild reports whether any of operands holds a child value step naming
// name, at any depth.
func (n ctaOr) readsChild(name xsd.QName) bool { return ctaAnyReadsChild(n.operands, name) }

// readsChild is ctaOr.readsChild's.
func (n ctaAnd) readsChild(name xsd.QName) bool { return ctaAnyReadsChild(n.operands, name) }

// readsChild reports whether the operand holds a child value step naming name.
func (n ctaNot) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild reports whether either operand is, or casts, a child value step
// naming name.
func (n ctaCompare) readsChild(name xsd.QName) bool {
	return n.left.readsChild(name) || n.right.readsChild(name)
}

// readsChild is ctaCompare.readsChild's.
func (n ctaValueCompare) readsChild(name xsd.QName) bool {
	return n.left.readsChild(name) || n.right.readsChild(name)
}

// readsChild reports whether the operand is, or casts, a child value step
// naming name.
func (n ctaEffectiveBoolean) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild is false: the node holds no operand.
func (ctaTypeError) readsChild(xsd.QName) bool { return false }

// readsChild reports whether the test or either branch holds a child value
// step naming name: which branch is evaluated is not known until E is.
func (n ctaIf) readsChild(name xsd.QName) bool {
	return n.test.readsChild(name) || n.then.readsChild(name) || n.otherwise.readsChild(name)
}

// ctaAnyReadsChild is readsChild over each of operands.
func ctaAnyReadsChild(operands []ctaExpr, name xsd.QName) bool {
	for _, o := range operands {
		if o.readsChild(name) {
			return true
		}
	}
	return false
}

// readsChild reports whether the step selects children named name. It and
// ctaUntypedChild are the child value steps every readsChild here looks for.
func (n ctaTypedChild) readsChild(name xsd.QName) bool { return n.name == name }

// readsChild is ctaTypedChild.readsChild's.
func (n ctaUntypedChild) readsChild(name xsd.QName) bool { return n.name == name }

// readsChild reports whether the cast's operand is a child value step naming
// name.
func (n ctaCast) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild is its cast's.
func (n ctaCastable) readsChild(name xsd.QName) bool { return n.cast.readsChild(name) }

// readsChild reports whether either operand of the arithmetic holds a
// child value step naming name.
func (n ctaArith) readsChild(name xsd.QName) bool {
	return n.left.readsChild(name) || n.right.readsChild(name)
}

// readsChild reports whether either argument holds a child value step naming
// name.
func (n ctaMatch) readsChild(name xsd.QName) bool {
	return n.left.operand.readsChild(name) || n.right.operand.readsChild(name)
}

// readsChild reports whether the argument holds a child value step naming name.
func (n ctaUnaryString) readsChild(name xsd.QName) bool { return n.arg.operand.readsChild(name) }

// readsChild reports whether the operand holds a child value step naming name:
// fn:exists over a cast of a child step reads its value off [ChildElements].
func (n ctaPresence) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild reports whether the cast fn:string is holds a child value step
// naming name.
func (n ctaStringFunction) readsChild(name xsd.QName) bool { return n.cast.readsChild(name) }

// readsChild reports whether the argument holds a child value step naming name.
func (n ctaDistinctValues) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild is false for each of these: none is a child-axis step or holds an
// operand.
func (ctaAttr) readsChild(xsd.QName) bool           { return false }
func (ctaTypedAttr) readsChild(xsd.QName) bool      { return false }
func (ctaNoDocumentRoot) readsChild(xsd.QName) bool { return false }
func (ctaNoContextItem) readsChild(xsd.QName) bool  { return false }
func (ctaLiteral) readsChild(xsd.QName) bool        { return false }
func (ctaValueVar) readsChild(xsd.QName) bool       { return false }
func (ctaEmptyValue) readsChild(xsd.QName) bool     { return false }
func (ctaUntypedValue) readsChild(xsd.QName) bool   { return false }
func (ctaCurrentDate) readsChild(xsd.QName) bool    { return false }
func (ctaContextAtom) readsChild(xsd.QName) bool    { return false }

// readsChild reports whether the call's argument reads the value of a child
// named name.
func (n ctaCount) readsChild(name xsd.QName) bool { return n.arg.readsChild(name) }

// readsChild is false for each of these: a counted path reads no child's
// value, and what it counts is the [Tally]'s.
func (ctaCountPath) readsChild(xsd.QName) bool        { return false }
func (ctaFilteredChildren) readsChild(xsd.QName) bool { return false }
func (ctaUnion) readsChild(xsd.QName) bool            { return false }

// readsChild reports whether the counted operand holds a child value step
// naming name: `count($value)` reads none.
func (c ctaCountedItems) readsChild(name xsd.QName) bool { return c.operand.readsChild(name) }

// readsChild reports whether m filters children named name, whose values its
// predicate reads: they are counted over [ChildElements], never the [Tally].
func (m ctaMatchingChildren) readsChild(name xsd.QName) bool { return m.name == name }

// readsChild is false: the candidate's value is the enclosing
// ctaMatchingChildren's read, which reports its name.
func (ctaCandidate) readsChild(xsd.QName) bool { return false }

// readsChild is false, on ctaCount's terms: a child path reads no node's value,
// and how many nodes it selects is the [Tally]'s.
func (ctaChildPath) readsChild(xsd.QName) bool { return false }

// readsChild is false, on ctaChildPath's terms.
func (ctaSelectedElements) readsChild(xsd.QName) bool { return false }

// counted appends each path any of operands counts over.
func (n ctaOr) counted(into []ctaTallied) []ctaTallied { return ctaAnyCounted(n.operands, into) }

// counted is ctaOr.counted's.
func (n ctaAnd) counted(into []ctaTallied) []ctaTallied { return ctaAnyCounted(n.operands, into) }

// counted appends each path the operand counts over.
func (n ctaNot) counted(into []ctaTallied) []ctaTallied { return n.operand.counted(into) }

// counted appends each path either operand counts over, the left one first.
func (n ctaCompare) counted(into []ctaTallied) []ctaTallied {
	return n.right.counted(n.left.counted(into))
}

// counted is ctaCompare.counted's.
func (n ctaValueCompare) counted(into []ctaTallied) []ctaTallied {
	return n.right.counted(n.left.counted(into))
}

// counted appends each path the operand counts over.
func (n ctaEffectiveBoolean) counted(into []ctaTallied) []ctaTallied {
	return n.operand.counted(into)
}

// counted appends each path the test, then either branch, counts over, on
// readsChild's terms.
func (n ctaIf) counted(into []ctaTallied) []ctaTallied {
	return n.otherwise.counted(n.then.counted(n.test.counted(into)))
}

// counted appends each path the cast's operand counts over.
func (n ctaCast) counted(into []ctaTallied) []ctaTallied { return n.operand.counted(into) }

// counted is its cast's.
func (n ctaCastable) counted(into []ctaTallied) []ctaTallied { return n.cast.counted(into) }

// counted appends each path either operand of the arithmetic counts over, the
// left one first.
func (n ctaArith) counted(into []ctaTallied) []ctaTallied {
	return n.right.counted(n.left.counted(into))
}

// counted appends each path either argument counts over, the left one first.
func (n ctaMatch) counted(into []ctaTallied) []ctaTallied {
	return n.right.operand.counted(n.left.operand.counted(into))
}

// counted appends each path the argument counts over.
func (n ctaUnaryString) counted(into []ctaTallied) []ctaTallied {
	return n.arg.operand.counted(into)
}

// counted appends each path the operand counts over.
func (n ctaPresence) counted(into []ctaTallied) []ctaTallied { return n.operand.counted(into) }

// counted appends each path the cast fn:string is counts over.
func (n ctaStringFunction) counted(into []ctaTallied) []ctaTallied {
	return n.cast.counted(into)
}

// counted appends each path the argument counts over.
func (n ctaDistinctValues) counted(into []ctaTallied) []ctaTallied {
	return n.operand.counted(into)
}

// counted appends the path the call's argument counts over, if any.
func (n ctaCount) counted(into []ctaTallied) []ctaTallied { return n.arg.counted(into) }

// counted appends the key itself: the [Tally] counts the nodes it selects.
func (p ctaCountPath) counted(into []ctaTallied) []ctaTallied        { return append(into, p) }
func (f ctaFilteredChildren) counted(into []ctaTallied) []ctaTallied { return append(into, f) }
func (u ctaUnion) counted(into []ctaTallied) []ctaTallied            { return append(into, u) }

// counted appends each path the counted operand counts over, and never c
// itself: its items are read off the operand, so `count($value)` keys nothing
// in the [Tally].
func (c ctaCountedItems) counted(into []ctaTallied) []ctaTallied { return c.operand.counted(into) }

// counted appends nothing: m is counted over [ChildElements], never keyed in
// the [Tally], and its predicate counts nothing (ctaPredicateFacade.count).
func (ctaMatchingChildren) counted(into []ctaTallied) []ctaTallied { return into }

// counted appends nothing: the candidate is no fn:count call.
func (ctaCandidate) counted(into []ctaTallied) []ctaTallied { return into }

// counted appends the path itself: the [Tally] counts the nodes it selects.
func (n ctaChildPath) counted(into []ctaTallied) []ctaTallied { return append(into, n) }

// counted appends the step's path, the key an fn:count over the same step
// counts under too.
func (n ctaSelectedElements) counted(into []ctaTallied) []ctaTallied {
	return append(into, n.path)
}

// ctaAnyCounted is counted over each of operands, in written order.
func ctaAnyCounted(operands []ctaExpr, into []ctaTallied) []ctaTallied {
	for _, o := range operands {
		into = o.counted(into)
	}
	return into
}

// counted appends nothing for each of these: none is an fn:count call or holds
// an operand.
func (ctaTypeError) counted(into []ctaTallied) []ctaTallied      { return into }
func (ctaAttr) counted(into []ctaTallied) []ctaTallied           { return into }
func (ctaTypedAttr) counted(into []ctaTallied) []ctaTallied      { return into }
func (ctaTypedChild) counted(into []ctaTallied) []ctaTallied     { return into }
func (ctaUntypedChild) counted(into []ctaTallied) []ctaTallied   { return into }
func (ctaNoDocumentRoot) counted(into []ctaTallied) []ctaTallied { return into }
func (ctaNoContextItem) counted(into []ctaTallied) []ctaTallied  { return into }
func (ctaLiteral) counted(into []ctaTallied) []ctaTallied        { return into }
func (ctaValueVar) counted(into []ctaTallied) []ctaTallied       { return into }
func (ctaEmptyValue) counted(into []ctaTallied) []ctaTallied     { return into }
func (ctaUntypedValue) counted(into []ctaTallied) []ctaTallied   { return into }
func (ctaContextAtom) counted(into []ctaTallied) []ctaTallied    { return into }
func (ctaCurrentDate) counted(into []ctaTallied) []ctaTallied    { return into }

// ctaAssertionFacade is the assertion façade compileCTATest parses for: its
// attribute nodes are typed by attrs, its child element nodes by elems, its
// `$value` by content, which also decides whether `.` is read, and it declines
// the comparison types it cannot yet decide.
type ctaAssertionFacade struct {
	content xsd.ContentType
	attrs   AttributeTypes
	elems   ElementTypes
}

func (ctaAssertionFacade) ctaFacade() {}

// comparesValues is true: an assertion's {test} is full XPath 2.0 (§3.13), so
// [23] ValueComp is in its grammar, and the §3.13.2 example `@min le @max`
// writes one.
func (ctaAssertionFacade) comparesValues() bool { return true }

// computes is true, on comparesValues' terms: §3.4's arithmetic is in full
// XPath 2.0.
func (ctaAssertionFacade) computes() bool { return true }

// callsLibrary is true, on comparesValues' terms: the F&O function library is
// in full XPath 2.0.
func (ctaAssertionFacade) callsLibrary() bool { return true }

// conditional is true, on comparesValues' terms: §3.8's IfExpr is in full
// XPath 2.0.
func (ctaAssertionFacade) conditional() bool { return true }

// constructsSequences is true, on comparesValues' terms: §3.3.1's sequence
// expressions are in full XPath 2.0.
func (ctaAssertionFacade) constructsSequences() bool { return true }

// castable is true, on comparesValues' terms: §3.10.3's CastableExpr is in
// full XPath 2.0.
func (ctaAssertionFacade) castable() bool { return true }

// ctaValueName is the ·expanded name· of the one variable an assertion's
// static context holds (cvc-assertion clause 2.3): "no namespace URI and ...
// "value" as the local name".
var ctaValueName = xsd.QName{Local: "value"}

// variable compiles the [44] VarRef `$value` against content (cvc-assertion
// clause 2.3), and declines every other name, which is not in the static
// context (err:XPST0008) — a static error, withheld on [CompileAssertionTest]'s
// terms.
//
// Clause 2.3 is decided on content's {variety} first: anything but an
// [xsd.SimpleContent] binds the empty sequence (2.3.2), statically, which is
// ctaEmptyValue. A simple one binds the XDM representation of E's [schema
// actual value] under its {simple type definition} (2.3.1), whose static type
// ctaTypes.valueVariable classifies. A nil content declines.
func (f ctaAssertionFacade) variable(name xsd.QName, types ctaTypes) (ctaValue, bool) {
	if name != ctaValueName || f.content == nil {
		return nil, false
	}
	simple, isSimple := f.content.(xsd.SimpleContent)
	if !isSimple {
		return ctaEmptyValue{}, true
	}
	return types.valueVariable(simple.SimpleType)
}

// attribute compiles a QName NameTest whose name attrs types with a ·special·
// type ([xsd.SimpleType.IsSpecial]) to a ctaAttr, the untyped attribute node a Type
// Alternative builds — the typed value of such an attribute is its [schema
// normalized value] as xs:untypedAtomic (xpath-datamodel §3.3.1.2; Datatypes
// dt-xdmrep clause 1 for that dynamic type), which §3.5.2 casts as it casts an
// untyped one. A name attrs types with a type this engine reads as one atomic
// value (ctaTypes.typedAtomic) compiles to a ctaTypedAttr, and every other
// NameTest declines.
//
// An unbound prefix's ctaUnresolvedName reaches attrs like any other name; no
// attribute use can carry it, so the façade declines it and the parse ends
// unsupported, which an assertion withholds on exactly as it would on the
// static error.
func (f ctaAssertionFacade) attribute(test ctaNameTest, types ctaTypes) (ctaValue, bool) {
	exact, isExact := test.(ctaExactName)
	if !isExact {
		return nil, false
	}
	st, typed := f.attrs(exact.name)
	if !typed {
		return nil, false
	}
	if st.IsSpecial() {
		return ctaAttr{test: exact}, true
	}
	if !types.typedAtomic(st) {
		return nil, false
	}
	return ctaTypedAttr{name: exact.name, st: st}, true
}

// child compiles a QName NameTest on the child axis whose name elems types to
// a ctaTypedChild, over the simple type a child of that type has its typed
// value in (ctaChildValueType): xpath-datamodel §6.2.4 makes the typed value
// of an element whose type is a simple type, or a complex type with simple
// content, the one §3.3.1.2 computes for that simple type. The simple type
// must be one ctaTypes.typedAtomic reads as one atomic value. A name elems
// types with a complex type whose {content type}.{variety} is mixed —
// xs:anyType among them, never singled out by name — compiles to a
// ctaUntypedChild instead: §6.2.4 makes such an element's typed value its
// string-value as xs:untypedAtomic. Every other NameTest declines, and so does
// every other type:
//
//   - a ·special· simple type, or simple content over one: the compiled type is
//     static, and an xsi:type can make the child's own type a typed one, so
//     reading it as xs:untypedAtomic could charge a valid child. A mixed type
//     carries no such hazard: the caller withholds a child whose own type has
//     no mixed content ([ChildElements]);
//   - a complex type whose {content type} is element-only (atomizing it is a
//     type error) or empty (its typed value is the empty sequence) — each read
//     the way xpath-datamodel §6.2.4 says, but by its own node this engine does
//     not build.
//
// A step standing as the whole operand of fn:exists, fn:empty or an ·effective
// boolean value· never reaches here: its value is not read, and elements
// builds its node whatever the type.
func (f ctaAssertionFacade) child(test ctaNameTest, types ctaTypes) (ctaValue, bool) {
	exact, isExact := test.(ctaExactName)
	if !isExact {
		return nil, false
	}
	td, typed := f.elems(exact.name)
	if !typed {
		return nil, false
	}
	if ctaMixedType(td) {
		return ctaUntypedChild(exact), true
	}
	st, simple := ctaChildValueType(td)
	if !simple || st.IsSpecial() || !types.typedAtomic(st) {
		return nil, false
	}
	return ctaTypedChild{name: exact.name, st: st}, true
}

// childPath compiles a relative path of two or more child-axis steps to a
// ctaChildPath over steps, consulting neither attrs nor elems: the path stands
// only where fn:exists, fn:empty or an ·effective boolean value· asks whether
// it selects a node, none of which atomizes it (xpath-functions.md §15.1.4,
// §15.1.5, `$arg as item()*`; xpath20.md §2.4.3 rule 2), so no step's type
// decides anything and a node of any type, ·nilled· or not, is selected — on
// count's terms. What the path selects is read off the [Tally].
func (ctaAssertionFacade) childPath(steps []xsd.QName) (ctaValue, bool) {
	path, admitted := ctaChildPathOf(steps)
	if !admitted {
		return nil, false
	}
	return path, true
}

// elements compiles one element step, `N`, `./N` or `.//N`, to a
// ctaSelectedElements over path, consulting neither attrs nor elems, on
// childPath's terms: the step stands only where its existence is asked, so N's
// type decides nothing — an element-only, mixed, empty or ·special· one is
// selected as a simple one is — and what it selects is read off the [Tally].
// An attribute axis declines; the parser builds none here.
func (ctaAssertionFacade) elements(path ctaCountPath) (ctaValue, bool) {
	selected, admitted := ctaSelectedElementsOf(path)
	if !admitted {
		return nil, false
	}
	return selected, true
}

// ctaChildValueType is the simple type the typed value of an element of type
// td is computed in (xpath-datamodel §6.2.4): td itself where it is simple,
// its {simple type definition} where it is complex with a simple {content
// type}, and false otherwise. The default arm is unreachable:
// [xsd.TypeDefinition] is a sealed sum of the two variants named (STYLE T2's
// schema-closed-set exception).
func ctaChildValueType(td xsd.TypeDefinition) (*xsd.SimpleType, bool) {
	switch t := td.(type) {
	case *xsd.SimpleType:
		return t, true
	case xsd.ComplexType:
		sc, simple := t.ContentType().(xsd.SimpleContent)
		return sc.SimpleType, simple
	}
	return nil, false
}

// ctaMixedType reports whether td is a complex type whose {content
// type}.{variety} is mixed, whose element's typed value is its string-value as
// xs:untypedAtomic (xpath-datamodel §6.2.4) — xs:anyType among them, whose
// {variety} is mixed (§3.4.7), and which is not told apart by its name.
func ctaMixedType(td xsd.TypeDefinition) bool {
	ct, isComplex := td.(xsd.ComplexType)
	return isComplex && ct.ContentType().Variety() == xsd.ContentMixed
}

// rooted compiles a path opening with "/" or "//" to ctaNoDocumentRoot: E is
// the root of the instance cvc-assertion clause 1.3 builds, and no document
// node is above it, so the path raises err:XPDY0050 (xpath20.md §3.2).
func (ctaAssertionFacade) rooted() (ctaValue, bool) {
	return ctaNoDocumentRoot{}, true
}

// count compiles an fn:count call over the path arg to its ctaCount (ctaCountOf). arg
// is never typed against attrs or elems: fn:count does not atomize it
// ([CompileAssertionTest]).
func (ctaAssertionFacade) count(arg ctaCounted, types ctaTypes) (ctaValue, bool) {
	return ctaCountOf(arg, types)
}

// ctaCountOf is the ctaCount of xs:integer over arg, the type
// xpath-functions.md §15.4.1 gives fn:count's result, false where types
// resolves no xs:integer.
func ctaCountOf(arg ctaCounted, types ctaTypes) (ctaValue, bool) {
	integer, resolved := types.simple(ctaBuiltin("integer"))
	if !resolved {
		return nil, false
	}
	return ctaCount{arg: arg, st: integer}, true
}

// contextItem compiles the [47] ContextItemExpr `.` to ctaContextAtom where
// content is an [xsd.SimpleContent]. The context item is E (cvc-xpath clause
// 1), annotated xs:anyType in the partial ·PSVI· (cvc-assertion clause 1.2),
// so `.` atomizes to one xs:untypedAtomic value, E's string value — under a
// list or union {simple type definition} as under an atomic one, the variety
// deciding `$value`'s type alone.
//
// GAP(xpath): under every other {content type}, and a nil one, `.` declines:
// E's string value is then the text of its descendants (xpath-datamodel
// §6.2.4), which is not an [AssertionTest.Evaluate] input. The direction is
// the withhold [CompileAssertionTest] reports. (#1042)
func (f ctaAssertionFacade) contextItem() (ctaValue, bool) {
	if _, simple := f.content.(xsd.SimpleContent); !simple {
		return nil, false
	}
	return ctaContextAtom{}, true
}
