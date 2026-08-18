package xpath

import "github.com/kud360/goxsd8/xsd"

// This file is the COMPILE-TIME type knowledge of the §3.12.6 subset: which
// builtin datatype a Literal carries, and which type a comparison's two
// operands are converted into (xpath20.md §3.5.2's casting rules and B.1's
// type promotions). Nothing here runs at evaluation time — the compiled tree
// holds the resolved *[xsd.SimpleType] components and no resolver
// (ARCHITECTURE), so [CTATest.Evaluate] takes one again.

// ctaBuiltin is the ·expanded name· of a builtin datatype, which is the key an
// [xsd.TypeResolver] holds it under. The XSD namespace is closed to user
// declarations, so a type resolvable in it is one of the 49 builtins by
// construction and no separate builtin table is needed here (STYLE T4).
func ctaBuiltin(local string) xsd.QName {
	return xsd.QName{Space: xsd.XMLSchemaNS, Local: local}
}

// ctaTypes is the type knowledge one [CompileCTATest] call reads: the resolver
// itself, plus the three builtin datatypes the compiler names outright — the
// two Literal kinds' types, and the xs:double §3.5.2 clause 2.1 answers with.
//
// It holds the resolver for the duration of the compile and puts it on no
// compiled node, which is ARCHITECTURE's rule for every reader above xsd that
// walks a {base type definition} chain.
type ctaTypes struct {
	resolver xsd.TypeResolver
	str      *xsd.SimpleType
	decimal  *xsd.SimpleType
	double   *xsd.SimpleType
}

// ctaResolveTypes reads the three builtin datatypes the compiler names, or
// reports false where the resolver answers for none of them — a schema whose
// {type definitions} were assembled without the builtins seeded. That is a
// whole-expression decline and not an error: no comparison in this grammar can
// be evaluated without xs:string, so the caller withholds on
// [CompileCTATest]'s own terms.
func ctaResolveTypes(r xsd.TypeResolver) (ctaTypes, bool) {
	t := ctaTypes{resolver: r}
	var resolved bool
	if t.str, resolved = t.simple(ctaBuiltin("string")); !resolved {
		return ctaTypes{}, false
	}
	if t.decimal, resolved = t.simple(ctaBuiltin("decimal")); !resolved {
		return ctaTypes{}, false
	}
	if t.double, resolved = t.simple(ctaBuiltin("double")); !resolved {
		return ctaTypes{}, false
	}
	return t, true
}

// simple resolves name to a simple type definition, reporting false where the
// {type definitions} hold none under that name or hold a COMPLEX one.
//
// The default arm is unreachable: [xsd.TypeDefinition] is a sealed sum of
// exactly the two variants named (STYLE T2's schema-closed-set exception), and
// the two are matched with the receiver kinds that sum's doc fixes — by value
// for a complex type, by pointer for a simple one.
func (t ctaTypes) simple(name xsd.QName) (*xsd.SimpleType, bool) {
	td, declared := t.resolver.Type(name)
	if !declared {
		return nil, false
	}
	switch d := td.(type) {
	case *xsd.SimpleType:
		return d, true
	case xsd.ComplexType:
		return nil, false
	default:
		return nil, false
	}
}

// literal is the builtin datatype a [16] ta-SimpleValue Literal carries: a
// DoubleLiteral (the one with an exponent, xpath20.md [73]) is xs:double, and
// an IntegerLiteral or DecimalLiteral is xs:decimal — xs:integer's primitive
// base, and so the type any two of them are compared in. A StringLiteral does
// not reach here; its type is xs:string with no text to inspect.
func (t ctaTypes) literal(text string) *xsd.SimpleType {
	for _, r := range text {
		if r == 'e' || r == 'E' {
			return t.double
		}
	}
	return t.decimal
}

// primitive resolves st's {primitive type definition}, reporting false where
// the chain cannot be walked or the property is ·absent· — xs:anySimpleType,
// xs:anyAtomicType, and the list and union varieties, none of which any
// operand of this grammar can carry once the cast targets are classified.
func (t ctaTypes) primitive(st *xsd.SimpleType) (*xsd.SimpleType, bool) {
	p, err := st.Primitive(t.resolver)
	if err != nil || p == nil {
		return nil, false
	}
	return p, true
}

// ancestor reports the ancestor of st named name — st ITSELF included — on the
// {base type definition} chain, or nil where the chain holds no such type. An
// unwalkable chain is the error, never a nil answer, so a caller cannot read
// "could not resolve the base" as "is not derived from it" (STYLE S3).
//
// TERMINATION: the walk carries no visited set (STYLE D4), on
// [xsd.SimpleType.Variety]'s terms — every base chain a finalized Schema holds
// is acyclic, established in Phase B before any pass that walks one runs.
func (t ctaTypes) ancestor(st *xsd.SimpleType, name xsd.QName) (*xsd.SimpleType, error) {
	for at := st; at != nil; {
		if at.Name() == name {
			return at, nil
		}
		base, err := at.Base(t.resolver)
		if err != nil {
			return nil, err
		}
		at = base
	}
	return nil, nil
}

// castTarget is the datatype [15] ta-CastExpr's tail and [18]
// ta-ConstructorFunction cast to, or false where this engine declines the
// whole expression rather than casting to it.
//
// The three conditions §3.12.6 and xpath20.md §3.10.2 separate are separated
// here, and two of them share the false because they share their consequence
// at the caller ([CompileCTATest]):
//
//   - a BUILTIN atomic datatype is the required subset's own case, which
//     §3.12.6 clause 4 fixes for the cast spelling and clause 3 for the
//     constructor spelling, and it is admitted.
//   - any OTHER in-scope atomic type — a user-defined one — is valid XPath
//     that does not belong to the required subset, and §3.12.6's Note licenses
//     declining exactly that: "Conforming processors may but are not required
//     to support XPath expressions not belonging to the required subset of
//     XPath."
//   - a name resolving to nothing, to a complex type, to a non-atomic type
//     (xs:anySimpleType, and the list builtins xs:IDREFS, xs:NMTOKENS,
//     xs:ENTITIES), or to xs:anyAtomicType or xs:NOTATION is a STATIC error:
//     err:XPST0051 for "the target type must be an atomic type that is in the
//     in-scope schema types", and err:XPST0080 for the two named exclusions.
//
// GAP(xpath): a cast target that is XPST0051/XPST0080 is folded into the
// compile-time withhold; xpath-valid cl. 2 is charged by its owner (#886).
//
// GAP(xpath): a target whose {primitive type definition} is xs:QName is
// declined too, though §3.10.2 excludes only xs:NOTATION and xs:anyAtomicType
// by name. Casting to xs:QName is context-dependent — the lexical's prefix
// resolves against the static context's namespaces (xpath-functions.md §5.3),
// which is the [value.Context] this engine has no value for (PRINCIPLES 19) —
// and F&O's casting table supports no dynamically-supplied operand for it at
// all. BOTH spellings decline, because the target is what is classified here;
// only the string-LITERAL one is a LOSS, an attribute operand having no
// defined result to withhold in the first place. It takes [CompileCTATest]'s
// own withhold direction, argued there, rather than deciding. (#888)
func (t ctaTypes) castTarget(name xsd.QName) (*xsd.SimpleType, bool) {
	st, declared := t.simple(name)
	if !declared {
		return nil, false
	}
	if name.Space != xsd.XMLSchemaNS {
		return nil, false
	}
	if name == ctaBuiltin("anyAtomicType") || name == ctaBuiltin("NOTATION") {
		return nil, false
	}
	// §3.10.2's "the target type must be an atomic type" is decided on the
	// {primitive type definition} and not on the {variety}, because the
	// property is ·absent· for EXACTLY the targets that rule excludes: the
	// list and union varieties, xs:anySimpleType's ·absent· {variety}, and
	// xs:anyAtomicType, which the line above already excluded by name.
	p, resolved := t.primitive(st)
	if !resolved {
		return nil, false
	}
	if p.Name() == ctaBuiltin("QName") {
		return nil, false
	}
	return st, true
}

// ctaTyping is which of the three outcomes settling a comparison's type
// reached, and the three are three DIFFERENT directions rather than degrees of
// one (STYLE P3):
//
//   - ctaTypeSettled carries the type both operands are converted into.
//   - ctaTypeErrored is err:XPTY0004 — no one type serves both operands, which
//     is a raised type error and so a ctaTypeError node, decided false for the
//     whole {test} by key-cta-ta-select clause 2.
//   - ctaTypeDeclined is [CompileCTATest]'s WITHHOLD: this engine will not
//     decide the pair at all, and the caller's ·governing type definition· is
//     left undetermined rather than settled on a wrong answer.
//
// ctaTypeErrored is the zero value, so a fault that returns no type reports
// the error direction and never the withhold by accident.
type ctaTyping byte

const (
	ctaTypeErrored ctaTyping = iota
	ctaTypeDeclined
	ctaTypeSettled
)

// comparison settles the type a general comparison of l against r converts
// BOTH its operands into, per xpath20.md §3.5.2 clause 2's
// magnitude-relationship rules and B.1's type promotions.
//
// Two rules cover the three operand shapes this grammar builds, because an
// operand is either xs:untypedAtomic (an uncast attribute) or typed (a
// Literal, a cast, a constructor function):
//
//   - BOTH xs:untypedAtomic: clause 1, "the values are cast to the type
//     xs:string".
//   - EXACTLY ONE xs:untypedAtomic: clause 2 casts it to a type read off the
//     other operand's type T — untypedAgainst.
//   - NEITHER: one type must serve both, which is shared.
//
// B.1's URI promotion is NOT applied here. Answering xs:string for an
// xs:anyURI pair would change the type clause 2.4 casts the xs:untypedAtomic
// operand to, from xs:anyURI (whiteSpace collapse) to xs:string (preserve),
// and so change the answer: with @u=" http://a " and @v="http://a",
// `@u = @v cast as xs:anyURI` is true only if @u was collapsed. The promotion
// belongs where the comparison happens, and ctaCompare.eval applies it there
// by routing an xs:anyURI comparison type through the default collation
// exactly as it routes xs:string.
func (t ctaTypes) comparison(l, r ctaValue) (*xsd.SimpleType, ctaTyping) {
	lt, lTyped := ctaStaticOf(l).(ctaTyped)
	rt, rTyped := ctaStaticOf(r).(ctaTyped)
	if !lTyped && !rTyped {
		return t.str, ctaTypeSettled
	}
	if !lTyped {
		return t.untypedAgainst(rt.st)
	}
	if !rTyped {
		return t.untypedAgainst(lt.st)
	}
	return t.shared(lt.st, rt.st)
}

// untypedAgainst is xpath20.md §3.5.2 clause 2: the type an xs:untypedAtomic
// operand is cast to, chosen from the other operand's type T. All four
// sub-clauses are here, in their order.
//
//   - 2.1, "if T is a numeric type or is derived from a numeric type, then V
//     is cast to xs:double" — whatever the numeric type actually is, so a
//     comparison against an xs:integer cast runs in xs:double and not in
//     xs:decimal.
//   - 2.2 and 2.3, the two duration types, which are named rather than reduced
//     to their xs:duration primitive. The Note attached to them says why: "the
//     special treatment of the duration types is required to avoid errors that
//     may arise when comparing the primitive type xs:duration with any
//     duration type."
//   - 2.4, "in all other cases, V is cast to the primitive base type of T".
func (t ctaTypes) untypedAgainst(st *xsd.SimpleType) (*xsd.SimpleType, ctaTyping) {
	p, resolved := t.primitive(st)
	if !resolved {
		return nil, ctaTypeErrored
	}
	if ctaNumeric(p) {
		// GAP(xpath): clause 2.1 answers xs:double for an xs:float-primitive
		// T too, and B.1 rule 1.1 must then promote the T operand ITSELF to
		// xs:double — the promotion ctaWider declines, for the reason argued
		// there and under the same withhold. (#889)
		if p.Name() == ctaBuiltin("float") {
			return nil, ctaTypeDeclined
		}
		return t.double, ctaTypeSettled
	}
	dayTime, err := t.ancestor(st, ctaBuiltin("dayTimeDuration"))
	if err != nil {
		return nil, ctaTypeErrored
	}
	if dayTime != nil {
		return dayTime, ctaTypeSettled
	}
	yearMonth, err := t.ancestor(st, ctaBuiltin("yearMonthDuration"))
	if err != nil {
		return nil, ctaTypeErrored
	}
	if yearMonth != nil {
		return yearMonth, ctaTypeSettled
	}
	return p, ctaTypeSettled
}

// shared is the type serving two TYPED operands: their {primitive type
// definition} where they share one, and otherwise B.1's two promotions — the
// wider of two numeric primitives, and xs:string for an xs:anyURI met by an
// xs:string. Those two are the only ways two DIFFERENT primitives are ever
// admitted, which is the reason a date cast compared against a string literal
// is err:XPTY0004 (the shape §3.5.2's untypedAtomic rules never reach, because
// a StringLiteral is xs:string and so no operand is untyped).
//
// The comparison is by ·expanded name· rather than by component identity: a
// primitive is always named, and a resolver is free to answer with a component
// this compile did not itself resolve.
func (t ctaTypes) shared(a, b *xsd.SimpleType) (*xsd.SimpleType, ctaTyping) {
	pa, resolved := t.primitive(a)
	if !resolved {
		return nil, ctaTypeErrored
	}
	pb, resolved := t.primitive(b)
	if !resolved {
		return nil, ctaTypeErrored
	}
	if pa.Name() == pb.Name() {
		return pa, ctaTypeSettled
	}
	if ctaNumeric(pa) && ctaNumeric(pb) {
		return ctaWider(pa, pb)
	}
	if ctaStringLike(pa) && ctaStringLike(pb) {
		return t.str, ctaTypeSettled
	}
	return nil, ctaTypeErrored
}

// ctaNumeric reports whether p is one of the three primitives xpath20.md calls
// numeric: xs:decimal, xs:float and xs:double (§B.1's numeric promotions are
// written over exactly these three).
func ctaNumeric(p *xsd.SimpleType) bool {
	return ctaRank(p) >= 0
}

// ctaStringLike reports whether p is xs:anyURI or xs:string, the two
// primitives B.1's URI promotion relates. p need not be primitive:
// ctaCompare.eval asks it of a comparison type, which untypedAgainst may have
// answered as a named duration type, and the name comparison answers false for
// that as for anything else.
func ctaStringLike(p *xsd.SimpleType) bool {
	return p.Name() == ctaBuiltin("string") || p.Name() == ctaBuiltin("anyURI")
}

// ctaWider is the target of B.1's numeric promotions between two different
// numeric primitives: xs:float promotes to xs:double, and xs:decimal to either
// of them, so the wider of the pair is the one both reach.
//
// GAP(xpath): the xs:float against xs:double pair is DECLINED rather than
// compared, because reaching it needs B.1 rule 1.1 and this engine cannot
// perform rule 1.1. That rule promotes the xs:float operand to "the xs:double
// value that is the same as the original value" — the same point of the real
// line, not a re-parse of any lexical — and value exposes no such widening:
// over the strict backend an xs:float value and an xs:double value answer Eq
// false and Cmp value.Incomparable, correctly, because they are values of
// different types. ctaPromote's ·canonical representation· round-trip is not
// that widening either: xs:float's canonical is the shortest decimal that
// round-trips to the FLOAT, so reparsing it as an xs:double lands on a
// DIFFERENT xs:double. (Rule 1.2's xs:decimal promotion IS "created by
// casting", so the same round-trip is exactly right for the decimal pairs and
// they stay admitted.) The direction is [CompileCTATest]'s withhold: the
// {test} decides nothing rather than deciding it wrongly. (#889)
func ctaWider(a, b *xsd.SimpleType) (*xsd.SimpleType, ctaTyping) {
	wider, narrower := a, b
	if ctaRank(b) > ctaRank(a) {
		wider, narrower = b, a
	}
	if wider.Name() == ctaBuiltin("double") && narrower.Name() == ctaBuiltin("float") {
		return nil, ctaTypeDeclined
	}
	return wider, ctaTypeSettled
}

// ctaRank orders the three numeric primitives by which promotes to which, and
// reports -1 for a primitive that is not numeric at all.
func ctaRank(p *xsd.SimpleType) int {
	switch p.Name() {
	case ctaBuiltin("decimal"):
		return 0
	case ctaBuiltin("float"):
		return 1
	case ctaBuiltin("double"):
		return 2
	}
	return -1
}
