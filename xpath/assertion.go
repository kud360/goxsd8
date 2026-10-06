package xpath

import (
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
// The assertion façade widens the grammar by SEVEN productions the Type
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
// a [Tally] carries; and, in the same position, a call to one of the F&O
// string and sequence functions [CompileAssertionTest] lists
// (ctaFacade.callsLibrary, ctafunc.go), whose argument may also be the empty
// sequence `()`. It also admits xpath20.md §3.4's binary arithmetic operators
// over numeric operands (ctaFacade.computes), in each comparison operand's
// position.

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
// value is the empty sequence (xpath-datamodel §6.2.4): a node all the same,
// which a {test} naming it finds.
func Child(name xsd.QName, v TypedValue) ChildElement { return ChildElement{name: name, v: v} }

// ChildElements yields E's element [[children]] a compiled {test} reads, in
// DOCUMENT ORDER (STYLE D1), each as a [ChildElement]. It must be non-nil, and
// a yield reporting false ends the walk. A child no compiled step names need
// not be yielded: [AssertionTest.ReadsChild] reports which names one reads.
//
// Each value is [Typed] of a value of EXACTLY the simple type the compile read
// off the type [ElementTypes] answered for its name — that type itself, or the
// {simple type definition} of its simple {content type} — or nil for a
// ·nilled· child. That agreement is the caller's obligation, on the terms
// [TypedAttributes] states for an attribute's value: validate produces the
// value from the child's [schema normalized value] under the child's OWN
// ·governing type definition·, then maps it under that simple type. A value
// breaking it — the [Untyped] arm — is a dynamic error wherever the tree reads
// it, which [AssertionTest.Evaluate] answers false.
//
// It carries the children whose typed values a compiled step reads and nothing
// below them. A node a {test} only counts — a child, a descendant, an
// attribute — or only asks the existence of through a path, `a`, `.//a` or
// `a/b`, is reported to the [Tally] instead, and never yielded here for that
// sake.
type ChildElements func(yield func(ChildElement) bool)

// TypedValue is the typed value of one attribute node, of one element node of
// simple type or simple content (xpath-datamodel §6.2.4), or of `$value`, in
// the data model instance cvc-assertion clause 1 builds, as xpath-datamodel
// §3.3.1.2 (Typed Value Determination) computes it: an ·actual value· of a type
// fixed at compile time ([Typed]), or a [schema normalized value] "as an
// instance of xs:untypedAtomic" ([Untyped]), which is the typed value under
// xs:anySimpleType and xs:anyAtomicType there and in Datatypes dt-xdmrep
// clause 1. It is a sealed sum of exactly those two arms, so no value carries
// both a lexical and a typed value (STYLE T1).
type TypedValue interface{ typedValue() }

// tvTyped is [Typed]'s arm: an ·actual value· of the type the tree holds.
type tvTyped struct{ v value.Value }

// tvUntyped is [Untyped]'s arm: a [schema normalized value], xs:untypedAtomic.
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

// Untyped is lexical as an instance of xs:untypedAtomic, which lexical must be
// the [schema normalized value] of: xpath-datamodel §3.3.1.2 makes that the
// typed value of a node whose type is xs:anySimpleType or xs:anyAtomicType.
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

// ValueBinding is the value cvc-assertion clause 2.3 binds to `$value` for one
// evaluation. The zero ValueBinding is the empty sequence — clause 2.3.2's
// value, for an E whose [validity] in the partial ·PSVI· is invalid, whose
// [nil] is true, or whose ·governing type definition· has a {content
// type}.{variety} other than simple.
//
// It has no third state for an E whose value is UNDECIDED: [AssertionTest.Evaluate]
// always decides, so a caller that cannot tell which of clause 2.3's cases E is
// in declines the assertion itself, as it does an attribute with no ·actual
// value·.
type ValueBinding struct{ v TypedValue }

// BindValue binds `$value` to the typed value v of E's [schema actual value]
// (cvc-assertion clause 2.3.1), whose arm and type the {simple type definition}
// of the [xsd.SimpleContent] the [AssertionTest] was compiled for fixes, on the
// terms [TypedAttributes] states for an attribute's value: [Untyped] of E's
// [schema normalized value] where it is ·special·, and [Typed] of a value of
// exactly it otherwise. A list {simple type definition}'s value must carry
// [value.Listed], whose items are the flattened sequence Datatypes dt-xdmrep
// makes its XDM representation.
//
// BindValue(nil) is the zero ValueBinding, the empty sequence, so the empty
// sequence has one encoding.
func BindValue(v TypedValue) ValueBinding { return ValueBinding{v: v} }

// Tally is the node-count input of ONE evaluation of ONE [AssertionTest] over
// the element E: for each relative path the {test} counts over with fn:count
// (xpath-functions.md §15.4.1), or asks the existence of with fn:exists,
// fn:empty or an ·effective boolean value· (§15.1.4, §15.1.5, xpath20.md
// §2.4.3), how many nodes of E's subtree it selects. [AssertionTest.Tally]
// makes one, the caller reports E's subtree to it while that subtree streams
// past, and [AssertionTest.Evaluate] reads it. It keeps one counter per
// distinct path and no node, so what the caller holds for a count is one
// integer per path whatever the subtree's size.
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
	if key.selectsElement(path) {
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
// A nil content declines every {test} naming `$value`.
//
// The grammar is [CompileCTATest]'s with the value comparisons, `$value`, an
// abbreviated child-axis step, a "/" or "//" opening one child or attribute
// step, a relative path of two or more abbreviated child-axis steps with QName
// NameTests, `a/b`, or one element step, `a`, `./a` or `.//a`, standing as the
// whole operand of fn:exists, fn:empty or an ·effective boolean value·
// (xpath20.md §3.2, §3.2.4, §2.4.3), an fn:count call, the binary arithmetic
// operators `+`, `-`, `*`, `div`, `idiv` and `mod` (xpath20.md §3.4), and a
// call to one of the F&O string and sequence functions — fn:contains,
// fn:starts-with and fn:ends-with with two arguments, fn:string-length,
// fn:normalize-space and fn:string with one, fn:empty and fn:exists with one,
// and fn:true and fn:false with none (xpath-functions.md §7.5.1–7.5.3, §7.4.4,
// §7.4.5, §2.3, §15.1.4, §15.1.5, §9.1.1, §9.1.2), any argument of which may be
// the empty sequence `()` — added, and every decline [CompileCTATest] states is
// this one's too, under the same static context (xpath-valid clause 2.2)
// augmented with `$value` (cvc-assertion clause 2.2), plus these, each of which
// is the same withhold:
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
//   - a child read for its value whose type elems answers is not a simple type
//     the bullets above admit as an attribute's, nor a complex type whose
//     simple {content type} is one: a ·special· type declines, because an
//     xsi:type can give the child a typed value where the compile read an
//     xs:untypedAtomic one, and so does every mixed, element-only and empty
//     {content type};
//   - a path of more than one step anywhere but as the whole operand of
//     fn:exists, fn:empty or an ·effective boolean value·, and there any step
//     on another axis, with a wildcard, a predicate or a kind test, and any
//     "//" between steps — so `a/b = 1`, `count(a/b)`, `string(a/b)`, `a/@b`,
//     `a//b` and `a/*` decline — a rooted path of more than one step, and a
//     "/" with no step after it;
//   - an fn:count argument that is not one QName step — `N`, `@N`, either of
//     them behind `./` or `.//`, or a rooted one — so a wildcard, a longer
//     path, a predicate, a bare `.` and `$value` decline, and so does a step
//     behind `./` or `.//` anywhere but in an fn:count call or as the whole
//     operand of fn:exists, fn:empty or an ·effective boolean value·;
//   - a cast whose operand is a typed attribute, a typed child or `$value`
//     outside the xs:string family, to a target that operand's type is neither
//     nor derived from by restriction (F&O §17.4, §17.1, §17.5) — so
//     `xs:integer(@d)` over an xs:decimal @d declines and `xs:decimal(@i)`
//     over an xs:int @i, §17.3's cast, does not — and a cast of an xs:float
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
//   - a call to fn:contains, fn:starts-with or fn:ends-with with a third,
//     collation argument (§7.3.1), which is never read as the two-argument
//     form, and a call to any of the functions above with an arity it does not
//     have (err:XPST0017);
//   - fn:string-length, fn:normalize-space and fn:string with no argument,
//     whose implicit argument is E's string value, read through `.`, which this
//     engine builds no node for;
//   - fn:string over a typed attribute, a typed child, an fn:count call, an
//     arithmetic result, `$value`, a function result or a cast of one of them
//     outside the xs:string family, which is the cast to xs:string the bullet
//     above declines, and fn:string over any argument whose {primitive type
//     definition} is xs:float or xs:double, a literal or a cast included,
//     which is the cast to xs:string that bullet's floating clause declines.
//
// An xs:string? argument — of every function above but fn:empty, fn:exists and
// fn:string — of any type outside the xs:string and xs:anyURI families is not a
// decline: xpath20.md §3.1.5's function conversion raises err:XPTY0004 for each
// item it yields, an absent attribute's empty sequence being the zero-length
// string, and so does a `$value` of two or more items.
//
// A counted step consults neither attrs nor elems: fn:count does not atomize
// its argument (xpath-functions.md §15.4.1, `$arg as item()*`), so the step's
// type decides nothing and a node of any type, or matched by a wildcard,
// counts. Nor does a child path of two or more steps, or one element step, as
// the whole operand of fn:exists, fn:empty or an ·effective boolean value·, on
// the same terms: fn:exists and fn:empty take `item()*` (§15.1.4, §15.1.5) and
// an ·effective boolean value· asks only whether the first item is a node
// (xpath20.md §2.4.3 rule 2), so a node of any type, ·nilled· or not, is
// selected — `a` over an element-only a is decided where `a = 1` declines. What
// the {test} counts is read off the [Tally] its evaluation carries
// ([AssertionTest.Tally]).
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
// children read for their value whose type is not one simple type,
// arithmetic outside the numeric operands and the binary operators, the
// collation argument, and every F&O function but fn:count and those listed
// above among them. The direction is the withhold: the caller records the
// assertion as unevaluated and neither charges it nor shows it satisfied
// (PRINCIPLES 20). (#1042)
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
// yields, whose subtree counts has counted and whose `$value` is v, WITHOUT
// raising a dynamic or type error — the whole of what cvc-assertion
// (§3.13.4.1) asks: "An element information item E is locally ·valid· with
// respect to an assertion if and only if the {test} evaluates to true (see
// below) without raising any dynamic error or type error." Clause 3 converts
// the result "as if by a call to the XPath fn:boolean function", which a
// boolean-rooted tree already is.
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
// to v. A tree compiled for a {content type} that is not simple never reads v.
// b and types are read as [CTATest.Evaluate] reads them.
//
// counts is the [Tally] t.Tally() made for this evaluation, filled with E's
// subtree on the terms that type states, and nil exactly where t.Tally() is
// nil. Any other counts — a nil one where t counts, a non-nil one where it
// counts nothing, or one made by a test counting other paths — breaks that
// obligation, on the terms [TypedAttributes] states for an attribute's value,
// and Evaluate answers false for it whatever the tree holds. A Tally made by
// another test counting the same paths cannot be told from t's own and is read
// as one; handing over the one filled for this E is the caller's part.
func (t AssertionTest) Evaluate(b value.Backend, types xsd.TypeResolver, attrs TypedAttributes, children ChildElements, counts *Tally, v ValueBinding) bool {
	if !counts.fits(t.countedPaths()) {
		return false
	}
	return ctaEval(t.root, ctaEnv{backend: b, types: types, input: ctaTypedInput{attrs: attrs, children: children, counts: counts, value: v}}) == ctaTrue
}

// Tally is a fresh, empty [Tally] for one evaluation of t, holding one counter
// for each distinct relative path an fn:count call in t counts over and each
// distinct path — one element step or a child path of two or more steps —
// whose existence t asks (fn:exists, fn:empty, an ·effective boolean value·),
// one counter serving a path both counted and asked, or nil where t counts
// over none — a {test} with neither, or one whose every fn:count argument is
// rooted and raises. t itself is not changed, so one compiled test serves any
// number of evaluations, each with its own Tally.
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
// [AssertionTest.Evaluate]'s [ChildElements] must yield. Its consumer is
// validate's walk, which keeps a child's value only where some {test} of its
// parent reads it. An fn:count call reads no value and is no read here, nor is
// an element step or a child path of two or more steps whose existence alone
// is asked, whatever the type of the child it names: what each counts is
// [AssertionTest.Tally]'s. The answer is read off the tree itself.
func (t AssertionTest) ReadsChild(name xsd.QName) bool {
	if t.root == nil {
		// The zero AssertionTest, which no successful CompileAssertionTest
		// produces, holds nothing.
		return false
	}
	return t.root.readsChild(name)
}

// readsChild reports whether any of operands holds a ctaTypedChild naming
// name, at any depth.
func (n ctaOr) readsChild(name xsd.QName) bool { return ctaAnyReadsChild(n.operands, name) }

// readsChild is ctaOr.readsChild's.
func (n ctaAnd) readsChild(name xsd.QName) bool { return ctaAnyReadsChild(n.operands, name) }

// readsChild reports whether the operand holds a ctaTypedChild naming name.
func (n ctaNot) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild reports whether either operand is, or casts, a ctaTypedChild
// naming name.
func (n ctaCompare) readsChild(name xsd.QName) bool {
	return n.left.readsChild(name) || n.right.readsChild(name)
}

// readsChild is ctaCompare.readsChild's.
func (n ctaValueCompare) readsChild(name xsd.QName) bool {
	return n.left.readsChild(name) || n.right.readsChild(name)
}

// readsChild reports whether the operand is, or casts, a ctaTypedChild naming
// name.
func (n ctaEffectiveBoolean) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild is false: the node holds no operand.
func (ctaTypeError) readsChild(xsd.QName) bool { return false }

// ctaAnyReadsChild is readsChild over each of operands.
func ctaAnyReadsChild(operands []ctaExpr, name xsd.QName) bool {
	for _, o := range operands {
		if o.readsChild(name) {
			return true
		}
	}
	return false
}

// readsChild reports whether the step selects children named name.
func (n ctaTypedChild) readsChild(name xsd.QName) bool { return n.name == name }

// readsChild reports whether the cast's operand is a ctaTypedChild naming
// name.
func (n ctaCast) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild reports whether either operand of the arithmetic holds a
// ctaTypedChild naming name.
func (n ctaArith) readsChild(name xsd.QName) bool {
	return n.left.readsChild(name) || n.right.readsChild(name)
}

// readsChild reports whether either argument holds a ctaTypedChild naming
// name.
func (n ctaMatch) readsChild(name xsd.QName) bool {
	return n.left.operand.readsChild(name) || n.right.operand.readsChild(name)
}

// readsChild reports whether the argument holds a ctaTypedChild naming name.
func (n ctaUnaryString) readsChild(name xsd.QName) bool { return n.arg.operand.readsChild(name) }

// readsChild reports whether the operand holds a ctaTypedChild naming name:
// fn:exists over a cast of a child step reads its value off [ChildElements].
func (n ctaPresence) readsChild(name xsd.QName) bool { return n.operand.readsChild(name) }

// readsChild reports whether the cast fn:string is holds a ctaTypedChild naming
// name.
func (n ctaStringFunction) readsChild(name xsd.QName) bool { return n.cast.readsChild(name) }

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

// readsChild is false: an fn:count call reads no child's value, and what it
// counts is the [Tally]'s.
func (ctaCount) readsChild(xsd.QName) bool { return false }

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

// counted appends each path the cast's operand counts over.
func (n ctaCast) counted(into []ctaTallied) []ctaTallied { return n.operand.counted(into) }

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

// counted appends the call's path where it is relative; a rooted argument
// raises before it selects a node and counts nothing.
func (n ctaCount) counted(into []ctaTallied) []ctaTallied {
	path, relative := n.arg.(ctaCountPath)
	if !relative {
		return into
	}
	return append(into, path)
}

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
func (ctaNoDocumentRoot) counted(into []ctaTallied) []ctaTallied { return into }
func (ctaNoContextItem) counted(into []ctaTallied) []ctaTallied  { return into }
func (ctaLiteral) counted(into []ctaTallied) []ctaTallied        { return into }
func (ctaValueVar) counted(into []ctaTallied) []ctaTallied       { return into }
func (ctaEmptyValue) counted(into []ctaTallied) []ctaTallied     { return into }
func (ctaUntypedValue) counted(into []ctaTallied) []ctaTallied   { return into }

// ctaAssertionFacade is the assertion façade compileCTATest parses for: its
// attribute nodes are typed by attrs, its child element nodes by elems, its
// `$value` by content, and it declines the comparison types it cannot yet
// decide.
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
// normalized value] as xs:untypedAtomic (xpath-datamodel §3.3.1.2, Datatypes
// dt-xdmrep clause 1), which §3.5.2 casts as it casts an untyped one. A name
// attrs types with a type this engine reads as one atomic value
// (ctaTypes.typedAtomic) compiles to a ctaTypedAttr, and every other NameTest
// declines.
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
// must be one ctaTypes.typedAtomic reads as one atomic value; every other
// NameTest declines, and so does every other type:
//
//   - a ·special· simple type, or simple content over one: the compiled type is
//     static, and an xsi:type can make the child's own type a typed one, so
//     reading it as xs:untypedAtomic could charge a valid child;
//   - a complex type whose {content type} is mixed (its typed value is its
//     string value as xs:untypedAtomic), element-only (atomizing it is a type
//     error), or empty (its typed value is the empty sequence) — each read the
//     way xpath-datamodel §6.2.4 says, but by its own node this engine does not
//     build.
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

// rooted compiles a path opening with "/" or "//" to ctaNoDocumentRoot: E is
// the root of the instance cvc-assertion clause 1.3 builds, and no document
// node is above it, so the path raises err:XPDY0050 (xpath20.md §3.2).
func (ctaAssertionFacade) rooted() (ctaValue, bool) {
	return ctaNoDocumentRoot{}, true
}

// count compiles an fn:count call over arg to a ctaCount of xs:integer, the
// type xpath-functions.md §15.4.1 gives its result, and declines it where types
// resolves no xs:integer. arg is never typed against attrs or elems: fn:count
// does not atomize it ([CompileAssertionTest]).
func (ctaAssertionFacade) count(arg ctaCounted, types ctaTypes) (ctaValue, bool) {
	integer, resolved := types.simple(ctaBuiltin("integer"))
	if !resolved {
		return nil, false
	}
	return ctaCount{arg: arg, st: integer}, true
}

// contextItem declines `.`. The context item is E (cvc-xpath), an element node
// whose typed value this engine builds no node for, so the [47]
// ContextItemExpr is outside what [CompileAssertionTest] admits, on child's
// terms.
func (ctaAssertionFacade) contextItem() (ctaValue, bool) {
	return nil, false
}
