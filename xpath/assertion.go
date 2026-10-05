package xpath

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file is the ASSERTION façade over the §3.12.6 grammar compileCTATest
// parses (STYLE T4): an {assertions} member's {test} (§3.13.1) written in that
// grammar is compiled and evaluated against the element it guards, which
// cvc-assertion (§3.13.4.1) asks of it. It is the first slice of tier 2
// (doc.go) and nothing wider: the grammar is the Type Alternative one, and only
// the façade differs (ctaFacade) — an assertion's instance is TYPED.
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
// The assertion façade widens the grammar by TWO productions the Type
// Alternative façade declines: the eq/ne/lt/le/gt/ge value comparisons
// (xpath20.md §3.5.1, [23] ValueComp), in [11] ta-BooleanExpr's comparator
// position, and the variable reference `$value` (xpath20.md [44] VarRef), as
// one more arm of [16] ta-SimpleValue — cvc-assertion clause 2.2 augments an
// assertion's static context with that one variable, and ta-props-correct
// clause 2's grammar names none.

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

// TypedValue is the typed value of one attribute node, or of `$value`, in the
// data model instance cvc-assertion clause 1 builds, as xpath-datamodel
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

// AssertionTest is a compiled assertion {test}: the expression tree
// [CompileAssertionTest] admitted for one element's attribute types. It is a
// distinct type from [CTATest] so a tree typed for an assertion can never be
// evaluated over a Type Alternative's [Attributes], nor a Type Alternative
// tree over [TypedAttributes].
type AssertionTest struct{ root ctaExpr }

// CompileAssertionTest compiles an assertion's {test} (§3.13.1, an
// [xsd.XPathExpression] property record) for the element whose ·governing
// type definition· has the {content type} content and whose attribute types
// attrs answers, reporting ok false for a {test} this engine cannot evaluate.
// Its consumer is validate's cvc-assertion site (validate/cvcassertion.go),
// which declines the assertion on ok false.
//
// content fixes `$value`'s static type, which is what cvc-assertion clause
// 2.3.1.3 reads: under an [xsd.SimpleContent] it is the {simple type
// definition}, whose value [BindValue] binds at evaluation; under every other
// {content type} it is the empty sequence (clause 2.3.2), whatever the binding.
// A nil content declines every {test} naming `$value`.
//
// The grammar is [CompileCTATest]'s with the value comparisons and `$value`
// added, and every decline [CompileCTATest] states is this one's too, under the
// same static context (xpath-valid clause 2.2) augmented with `$value`
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
//   - a cast whose operand is a typed attribute or `$value` outside the
//     xs:string family;
//   - a general or value comparison whose comparison type's {primitive type
//     definition} is a date/time one, which without an implicit timezone this
//     engine cannot order (ctaAssertionFacade.admitsComparison).
//
// An XPath STATIC error is declined too and never reported: the
// static-error question about an assertion is the schema assembler's, and
// [CTATestStaticError] answers it for a Type Alternative only.
//
// GAP(xpath): unlike a Type Alternative's, an assertion's {test} has no
// required subset to stop at — §3.13 admits full XPath 2.0 — so every decline
// above is this engine's limit and not the spec's license: paths and axes
// beyond the attribute step, and the F&O function library among them. The
// direction is the withhold: the caller records the assertion as unevaluated
// and neither charges it nor shows it satisfied (PRINCIPLES 20). (#1042)
//
// types is read as [CompileCTATest] reads it and stored nowhere.
func CompileAssertionTest(expr xsd.XPathExpression, types xsd.TypeResolver, content xsd.ContentType, attrs AttributeTypes) (AssertionTest, bool) {
	root, defect := compileCTATest(expr, types, ctaAssertionFacade{content: content, attrs: attrs})
	if defect.kind != ctaNoDefect {
		return AssertionTest{}, false
	}
	return AssertionTest{root: root}, true
}

// Evaluate reports whether the compiled {test} evaluates to true for the
// element whose attributes attrs yields and whose `$value` is v, WITHOUT
// raising a dynamic or type error — the whole of what cvc-assertion
// (§3.13.4.1) asks: "An element information item E is locally ·valid· with
// respect to an assertion if and only if the {test} evaluates to true (see
// below) without raising any dynamic error or type error." Clause 3 converts
// the result "as if by a call to the XPath fn:boolean function", which a
// boolean-rooted tree already is.
//
// So false is ONE answer for two outcomes the caller treats alike — the {test}
// was false, or it raised (err:FORG0001, err:XPTY0004, err:FORG0006) — and
// either way E is not ·valid· with respect to the assertion. A processor that
// raises a type error dynamically "will treat the expression as having
// evaluated to false" (cvc-xpath, §3.13.4.2). Every decline happened at
// [CompileAssertionTest].
//
// The dynamic context is cvc-xpath's — context item E, position and size 1 —
// with the one variable cvc-assertion clause 2.3 adds to it, `$value`, bound
// to v. A tree compiled for a {content type} that is not simple never reads v.
// b and types are read as [CTATest.Evaluate] reads them.
func (t AssertionTest) Evaluate(b value.Backend, types xsd.TypeResolver, attrs TypedAttributes, v ValueBinding) bool {
	return ctaEval(t.root, ctaEnv{backend: b, types: types, input: ctaTypedInput{attrs: attrs, value: v}}) == ctaTrue
}

// ctaAssertionFacade is the assertion façade compileCTATest parses for: its
// attribute nodes are typed by attrs, its `$value` by content, and it declines
// the comparison types it cannot yet decide.
type ctaAssertionFacade struct {
	content xsd.ContentType
	attrs   AttributeTypes
}

func (ctaAssertionFacade) ctaFacade() {}

// comparesValues is true: an assertion's {test} is full XPath 2.0 (§3.13), so
// [23] ValueComp is in its grammar, and the §3.13.2 example `@min le @max`
// writes one.
func (ctaAssertionFacade) comparesValues() bool { return true }

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
// type (ctaSpecial) to a ctaAttr, the untyped attribute node a Type
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
	if ctaSpecial(st) {
		return ctaAttr{test: exact}, true
	}
	if !types.typedAtomic(st) {
		return nil, false
	}
	return ctaTypedAttr{name: exact.name, st: st}, true
}

// admitsComparison declines a comparison type whose {primitive type
// definition} is in the date/time family — xs:dateTime (and so
// xs:dateTimeStamp), xs:time, xs:date, xs:gYearMonth, xs:gYear, xs:gMonthDay,
// xs:gDay, xs:gMonth — or cannot be resolved, and admits every other.
//
// GAP(xpath): F&O §10.4 (xpath-functions.md, "Comparison Operators on
// Duration, Date and Time Values") makes those comparisons a TOTAL order: "If
// either operand to a comparison function on date or time values does not have
// an (explicit) timezone then, for the purpose of the operation, an implicit
// timezone, provided by the dynamic context ..., is assumed to be present as
// part of the value." cvc-xpath clause 7 (§3.13.4.2) makes that implicit
// timezone implementation-defined but constant per ·assessment· episode. This
// engine has no implicit timezone, so ctaHoldsPair finds a timezoned operand
// and an untimezoned one value.Incomparable and unequal — false for every
// operator but != and ne, which it decides true — for a general and a value
// comparison alike, and an assertion would be charged (or satisfied) on that:
// `@d < @e or @d >= @e` over two xs:date attributes 2000-01-01 and 2000-01-01Z
// is a tautology this engine would answer false. The direction is the
// withhold: the {test} declines at [CompileAssertionTest], and the assertion
// is neither charged nor shown satisfied (PRINCIPLES 20). A Type Alternative's
// façade still evaluates them. (#1042)
func (ctaAssertionFacade) admitsComparison(types ctaTypes, c *xsd.SimpleType) bool {
	p, resolved := types.primitive(c)
	if !resolved {
		return false
	}
	switch p.Name() {
	case ctaBuiltin("dateTime"), ctaBuiltin("time"), ctaBuiltin("date"),
		ctaBuiltin("gYearMonth"), ctaBuiltin("gYear"), ctaBuiltin("gMonthDay"),
		ctaBuiltin("gDay"), ctaBuiltin("gMonth"):
		return false
	}
	return true
}
