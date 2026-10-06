package xpath

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file evaluates the F&O functions the assertion and facet façades call
// beyond fn:not and fn:count (ctaFacade.callsLibrary), each of which a Type
// Alternative's {test} declines: fn:contains, fn:starts-with and fn:ends-with
// (xpath-functions.md §7.5.1–7.5.3, ctaMatch), fn:string-length and
// fn:normalize-space (§7.4.4, §7.4.5, ctaUnaryString), fn:empty and fn:exists
// (§15.1.4, §15.1.5, ctaPresence), and fn:string (§2.3, ctaStringFunction).
// fn:true and fn:false (§9.1.1, §9.1.2) are constants, which compile to the
// ctaLiteral of their xs:boolean (ctaParser.constantCall).
//
// An argument whose parameter is xs:string? is converted by xpath20.md
// §3.1.5's function conversion rules, as far as the static type settles them,
// at compile time (ctaTypes.stringArgument), and read here by ctaStringOf. The
// strings are compared under the default collation, which xpath-valid clause
// 2.2.10 fixes as the Unicode codepoint collation, so a match is a match of
// code points — which a match of UTF-8 bytes is.

// ctaMatchOp is one of the three functions xpath-functions.md §7.5 matches one
// xs:string against another with: fn:contains, fn:starts-with and
// fn:ends-with.
type ctaMatchOp byte

const (
	ctaContains ctaMatchOp = iota
	ctaStartsWith
	ctaEndsWith
)

// ctaUnaryStringOp is one of the two functions of one xs:string? argument this
// grammar calls: fn:string-length (§7.4.4) and fn:normalize-space (§7.4.5).
type ctaUnaryStringOp byte

const (
	ctaStringLength ctaUnaryStringOp = iota
	ctaNormalizeSpace
)

// ctaPresenceOp is one of the two functions of one `item()*` argument this
// grammar calls: fn:empty (§15.1.4) and fn:exists (§15.1.5).
type ctaPresenceOp byte

const (
	ctaEmptyTest ctaPresenceOp = iota
	ctaExistsTest
)

// ctaStringArgument is one argument of an F&O function whose parameter is
// xs:string?, as ctaTypes.stringArgument converted it at compile time: operand
// yields items of the xs:string family or none — an xs:untypedAtomic operand
// is held under its cast to xs:string — and mistyped marks an operand of any
// other type, whose every item is the err:XPTY0004 xpath20.md §3.1.5 raises.
// ctaStringOf reads it.
type ctaStringArgument struct {
	operand  ctaValue
	mistyped bool
}

// ctaMatch is a call to fn:contains, fn:starts-with or fn:ends-with (op) with
// two arguments, whose result is st, xs:boolean.
type ctaMatch struct {
	op          ctaMatchOp
	left, right ctaStringArgument
	st          *xsd.SimpleType
}

// ctaUnaryString is a call to fn:string-length or fn:normalize-space (op) with
// one argument, written or implicit (ctaParser.unaryStringCall), whose result
// is st: xs:integer for fn:string-length and xs:string for fn:normalize-space
// (resultType).
type ctaUnaryString struct {
	op  ctaUnaryStringOp
	arg ctaStringArgument
	st  *xsd.SimpleType
}

// ctaPresence is a call to fn:empty or fn:exists (op), whose result is st,
// xs:boolean. The operand is `item()*` and is not atomized: a step's nodes are
// counted whatever their values, a ·nilled· child among them
// (ctaSequenceLength).
type ctaPresence struct {
	op      ctaPresenceOp
	operand ctaValue
	st      *xsd.SimpleType
}

// ctaStringFunction is a call to fn:string, which is its argument cast to
// xs:string (xpath-functions.md §2.3, ctaParser.stringOf) with the empty
// sequence mapped to the zero-length string. Its static type is cast.target.
type ctaStringFunction struct{ cast ctaCast }

func (ctaMatch) ctaValue()          {}
func (ctaUnaryString) ctaValue()    {}
func (ctaPresence) ctaValue()       {}
func (ctaStringFunction) ctaValue() {}

// resultType is the ·expanded name· of the type op returns.
func (op ctaUnaryStringOp) resultType() xsd.QName {
	if op == ctaStringLength {
		return ctaBuiltin("integer")
	}
	return ctaBuiltin("string")
}

// holds reports whether s matches sub as op asks, code point by code point:
// anywhere for fn:contains, at the start for fn:starts-with and at the end for
// fn:ends-with. A zero-length sub matches every s, and no other sub matches a
// zero-length s, which are the two empty-string rules §7.5.1–7.5.3 state, in
// the order they state them.
func (op ctaMatchOp) holds(s, sub string) bool {
	switch op {
	case ctaContains:
		return strings.Contains(s, sub)
	case ctaStartsWith:
		return strings.HasPrefix(s, sub)
	case ctaEndsWith:
		return strings.HasSuffix(s, sub)
	}
	return false // op is one of the three constants; never reached
}

// ctaStringOf is the one xs:string a evaluates to, reporting false where it
// raises: the empty sequence is the zero-length string, which is how every
// xs:string? parameter of this grammar reads it (§7.4.4, §7.4.5, §7.5.1–7.5.3);
// two or more items are err:XPTY0004, and so is any item of a mistyped
// argument (xpath20.md §3.1.5); an operand that raised stays raised. The string
// is the item's ·canonical representation·, read through [value.Canonical] on
// holdsCollated's terms — the identity for the xs:string family and for
// xs:anyURI, which §3.1.5's URI promotion makes an xs:string.
//
// The operand is read in its own static type, so no item is converted here:
// stringArgument cast an xs:untypedAtomic one to xs:string at compile time. The
// statically empty operand holds no item. An xs:untypedAtomic static type is
// raised, since no argument stringArgument built has one.
func ctaStringOf(a ctaStringArgument, env ctaEnv) (string, bool) {
	switch s := ctaStaticOf(a.operand).(type) {
	case ctaEmptySequence:
		return "", true
	case ctaTyped:
		atoms, converted := ctaItemOf(a.operand, s.st, env).(ctaAtoms)
		if !converted {
			return "", false
		}
		if len(atoms.vs) == 0 {
			return "", true
		}
		if len(atoms.vs) > 1 || a.mistyped {
			return "", false // err:XPTY0004
		}
		str, renders := atoms.vs[0].(value.Canonical)
		if !renders {
			return "", false
		}
		return str.Canonical(), true
	case ctaUntypedAtomic:
		return "", false
	}
	return "", false // ctaStatic has the three arms above; never reached
}

// ctaMatchItem evaluates n, its left argument first, and converts the
// xs:boolean it returns into c: an argument that raises is the error, and the
// result is validated against n.st and promoted, on ctaArithItem's terms.
func ctaMatchItem(n ctaMatch, c *xsd.SimpleType, env ctaEnv) ctaItem {
	s, ok := ctaStringOf(n.left, env)
	if !ok {
		return ctaRaised{}
	}
	sub, ok := ctaStringOf(n.right, env)
	if !ok {
		return ctaRaised{}
	}
	return ctaConvert(strconv.FormatBool(n.op.holds(s, sub)), n.st, c, env)
}

// ctaUnaryStringItem evaluates n and converts its result into c on
// ctaMatchItem's terms: fn:string-length counts CODE POINTS ("the length in
// characters", §7.4.4), and fn:normalize-space strips leading and trailing
// whitespace and folds each inner run into one #x20, the whitespace being XML's
// S production alone (§7.4.5, ctaNormalizedSpace).
func ctaUnaryStringItem(n ctaUnaryString, c *xsd.SimpleType, env ctaEnv) ctaItem {
	s, ok := ctaStringOf(n.arg, env)
	if !ok {
		return ctaRaised{}
	}
	if n.op == ctaStringLength {
		return ctaConvert(strconv.Itoa(utf8.RuneCountInString(s)), n.st, c, env)
	}
	return ctaConvert(ctaNormalizedSpace(s), n.st, c, env)
}

// ctaNormalizedSpace is fn:normalize-space over s: its runs of XML S — #x20,
// #x9, #xD and #xA, and nothing else, so neither U+0085 nor U+00A0 is
// whitespace here — dropped at either end and folded to one #x20 between.
func ctaNormalizedSpace(s string) string {
	return strings.Join(strings.FieldsFunc(s, ctaXMLSpace), " ")
}

// ctaXMLSpace reports whether r is one of the four characters of XML 1.0's S
// production.
func ctaXMLSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n'
}

// ctaPresenceItem evaluates n and converts the xs:boolean it returns into c on
// ctaMatchItem's terms: fn:empty is true and fn:exists false exactly where the
// operand is the empty sequence. An operand that raises is the error.
func ctaPresenceItem(n ctaPresence, c *xsd.SimpleType, env ctaEnv) ctaItem {
	length, ok := ctaSequenceLength(n.operand, env)
	if !ok {
		return ctaRaised{}
	}
	return ctaConvert(strconv.FormatBool((length == 0) == (n.op == ctaEmptyTest)), n.st, c, env)
}

// ctaStringFunctionItem evaluates fn:string's cast and converts the xs:string
// it returns into c on ctaMatchItem's terms: the cast of the empty sequence is
// the empty sequence, which fn:string returns as the zero-length string
// (§2.3), and every other outcome — one value, or the err:XPTY0004 of two or
// more items (ctaCastItem) — is the cast's.
func ctaStringFunctionItem(n ctaStringFunction, c *xsd.SimpleType, env ctaEnv) ctaItem {
	cast, converted := ctaCastItem(n.cast, n.cast.target, env).(ctaAtoms)
	if !converted {
		return ctaRaised{}
	}
	if len(cast.vs) == 0 {
		return ctaConvert("", n.cast.target, c, env)
	}
	return ctaPromote(cast.vs[0], n.cast.target, c, env)
}

// ctaSequenceLength is how many items v evaluates to, without atomizing it —
// the question fn:empty and fn:exists ask of their `item()*` argument —
// reporting false where it raises. A step is the number of nodes it selects,
// which ctaStep.nodes answers for both this and the ·effective boolean value·;
// `$value` over ·special· content is one xs:untypedAtomic value or, unbound,
// none (ctaUntypedBinding); the statically empty sequence is none; and every
// other operand is the length of its items in its own static type, which
// converts none of them.
func ctaSequenceLength(v ctaValue, env ctaEnv) (int, bool) {
	if step, isStep := v.(ctaStep); isStep {
		return step.nodes(env)
	}
	if _, untyped := v.(ctaUntypedValue); untyped {
		_, bound, ok := ctaUntypedBinding(env)
		if !bound {
			return 0, ok
		}
		return 1, ok
	}
	typed, isTyped := ctaStaticOf(v).(ctaTyped)
	if !isTyped {
		return 0, true // ctaEmptyValue, the statically empty sequence
	}
	atoms, ok := ctaItemOf(v, typed.st, env).(ctaAtoms)
	return len(atoms.vs), ok
}

// ctaStep is a ctaValue that evaluates to a sequence of NODES rather than of
// atomic values: an attribute step, untyped (ctaAttr) or typed (ctaTypedAttr),
// a child-axis step (ctaTypedChild), an element step whose existence is asked
// (ctaSelectedElements), a path of child steps (ctaChildPath), the candidate
// `.` inside a value predicate (ctaCandidate), and the two steps that raise
// before they select a node, a rooted path (ctaNoDocumentRoot) and a read of
// an absent context item (ctaNoContextItem). Its nodes method is the ONE
// reading of node existence, which the ·effective boolean value· of a step
// (ctaEffectiveBoolean.eval) and fn:empty and fn:exists (ctaSequenceLength)
// both take; every other operand's items are atomic values the caller reads
// its own way.
type ctaStep interface {
	ctaValue
	// nodes is how many nodes the step selects, a ·nilled· child counting as a
	// node, reporting false where it raises: a rooted path (err:XPDY0050), a
	// read of an absent context item (err:XPDY0002), a matched value breaking
	// the caller's obligation ([TypedAttributes], [ChildElements]), and a
	// [Tally] holding no counter for an element step or a child path, which
	// [AssertionTest.Evaluate] refuses before the tree is read.
	nodes(env ctaEnv) (int, bool)
}

func (s ctaAttr) nodes(env ctaEnv) (int, bool) {
	matched, ok := ctaMatchedAttributes(s, env)
	return len(matched), ok
}

func (s ctaTypedAttr) nodes(env ctaEnv) (int, bool) {
	matched, ok := ctaMatchedTyped(s, env)
	return len(matched), ok
}

func (s ctaTypedChild) nodes(env ctaEnv) (int, bool) {
	_, nodes, ok := ctaMatchedChildren(s, env)
	return nodes, ok
}

// nodes is the counter the evaluation's [Tally] holds for s (ctaTalliedNodes).
func (s ctaChildPath) nodes(env ctaEnv) (int, bool) { return ctaTalliedNodes(s, env) }

// nodes is the counter the evaluation's [Tally] holds for s's path
// (ctaTalliedNodes).
func (s ctaSelectedElements) nodes(env ctaEnv) (int, bool) { return ctaTalliedNodes(s.path, env) }

// nodes is the counter the evaluation's [Tally] holds for p (ctaTalliedNodes),
// which is how many nodes fn:count over p counts.
func (p ctaCountPath) nodes(env ctaEnv) (int, bool) { return ctaTalliedNodes(p, env) }

// nodes is ctaCountPath.nodes'.
func (f ctaFilteredChildren) nodes(env ctaEnv) (int, bool) { return ctaTalliedNodes(f, env) }

// nodes is ctaCountPath.nodes'.
func (u ctaUnion) nodes(env ctaEnv) (int, bool) { return ctaTalliedNodes(u, env) }

// nodes is how many of E's children named m.name its predicate is true for,
// each child in document order the context item (ctaEnv.candidate): its typed
// value, or none for a ·nilled· one. A predicate that raises over any child
// raises for the whole count, and so does a yielded value breaking the
// obligation [ChildElements] states. The input is ctaTypedInput by
// construction (ctaInput); the other arm carries no children and raises,
// unreachably.
func (m ctaMatchingChildren) nodes(env ctaEnv) (int, bool) {
	in, typed := env.input.(ctaTypedInput)
	if !typed {
		return 0, false
	}
	n, ok := 0, true
	in.children(func(c ChildElement) bool {
		if c.name != m.name {
			return true
		}
		inner := env
		inner.candidate = nil
		if c.v != nil {
			tv, isTyped := c.v.(tvTyped)
			if !isTyped {
				ok = false
				return false
			}
			inner.candidate = []value.Value{tv.v}
		}
		switch ctaEval(m.pred, inner) {
		case ctaTrue:
			n++
		case ctaError:
			ok = false
			return false
		case ctaFalse:
		}
		return true
	})
	return n, ok
}

// nodes is 1: the candidate is one node, ·nilled· or not.
func (ctaCandidate) nodes(ctaEnv) (int, bool) { return 1, true }

// ctaTalliedNodes is the counter the evaluation's [Tally] holds for key, which
// the caller filled with E's subtree. The input is ctaTypedInput by
// construction (ctaInput); the other arm holds no Tally and raises,
// unreachably.
func ctaTalliedNodes(key ctaTallied, env ctaEnv) (int, bool) {
	in, typed := env.input.(ctaTypedInput)
	if !typed {
		return 0, false
	}
	return in.counts.count(key)
}

func (ctaNoDocumentRoot) nodes(ctaEnv) (int, bool) { return 0, false } // err:XPDY0050
func (ctaNoContextItem) nodes(ctaEnv) (int, bool)  { return 0, false } // err:XPDY0002
