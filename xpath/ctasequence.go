package xpath

import (
	"strconv"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// This file parses and evaluates the sequence expressions of xpath20.md §3.3.1
// the assertion and facet façades admit (ctaFacade.constructsSequences) as an
// operand of a general comparison, and nowhere else: [11] RangeExpr,
// `IntegerLiteral 'to' IntegerLiteral`, written bare or parenthesized, and the
// comma operator of [2] Expr inside a [46] ParenthesizedExpr, `(a, b, …)`,
// whose members are IntegerLiterals and such ranges — so `. = (1 to 10, 20,
// 30)` and `$value = 1 to 10` compile.
//
// The sequence is materialized when the comparison reads it (ctaAtoms), so its
// length is fixed at compile time from the literals alone, and bounded by
// ctaMaxSequenceLength.

// ctaMaxSequenceLength is the most items an integer sequence may hold for
// ctaParser.integerSequence to compile it: every item is built when the
// comparison over it is evaluated, once per evaluation.
const ctaMaxSequenceLength = 1 << 12

// ctaIntegerRanges is a sequence of xs:integer values built by §3.3.1's
// operators from IntegerLiterals: the concatenation, in written order, of
// each range's integers from lo to hi inclusive — "a sequence containing two
// or more consecutive integers ... in increasing order", one integer where lo
// is hi, and none where hi is less than lo. st is xs:integer, the type a
// RangeExpr's items have and an IntegerLiteral's value is (xpath20.md §3.1.1),
// which is the operand's static type (ctaStaticOf).
//
// Its one constructor is ctaParser.integerSequence, which holds at least one
// item and at most ctaMaxSequenceLength: a sequence of none is the statically
// empty ctaEmptyValue instead.
type ctaIntegerRanges struct {
	ranges []ctaIntegerRange
	st     *xsd.SimpleType
}

// ctaIntegerRange is one member of a ctaIntegerRanges: `lo to hi`, or one
// IntegerLiteral, where lo is hi.
type ctaIntegerRange struct{ lo, hi int64 }

func (ctaIntegerRanges) ctaValue() {}

// readsChild is false: the sequence reads nothing of the instance.
func (ctaIntegerRanges) readsChild(xsd.QName) bool { return false }

// counted appends nothing: the sequence is no fn:count call.
func (ctaIntegerRanges) counted(into []ctaTallied) []ctaTallied { return into }

// ctaIntegerRangesItem is n's integers in written order, each converted into
// c on ctaTypedAttrItem's terms through the lexical of that integer, which is a
// datatype validation as every value this package builds is.
func ctaIntegerRangesItem(n ctaIntegerRanges, c *xsd.SimpleType, env ctaEnv) ctaItem {
	var vs []value.Value
	for _, r := range n.ranges {
		for k := int64(0); k <= r.hi-r.lo; k++ {
			v, ok := ctaValidated(ctaConvert(strconv.FormatInt(r.lo+k, 10), n.st, c, env))
			if !ok {
				return ctaRaised{}
			}
			vs = append(vs, v)
		}
	}
	return ctaAtoms{vs: vs}
}

// generalOperand parses one operand of a general comparison, whose operands
// xpath20.md [10] ComparisonExpr makes [11] RangeExprs: an integer sequence
// where the tokens at the cursor spell one (integerSequenceLength), and an
// additiveExpr otherwise.
func (p *ctaParser) generalOperand() (ctaValue, bool) {
	if n := p.integerSequenceLength(0); n > 0 {
		return p.integerSequence(n)
	}
	return p.additiveExpr()
}

// integerSequenceLength is how many tokens, from offset at ahead of the
// cursor, spell an integer sequence — `IntegerLiteral 'to' IntegerLiteral`,
// or `'(' member (',' member)* ')'`, each member an IntegerLiteral or such a
// range — and 0 where they spell none or the façade constructs no sequence
// (ctaFacade.constructsSequences). A bare IntegerLiteral is additiveExpr's.
// Nothing is consumed.
func (p *ctaParser) integerSequenceLength(at int) int {
	if !p.facade.constructsSequences() {
		return 0
	}
	if p.peek(at).kind != ctaLParen {
		if p.rangeMemberLength(at) != 3 {
			return 0
		}
		return 3
	}
	n := 1
	for {
		m := p.rangeMemberLength(at + n)
		if m == 0 {
			return 0
		}
		n += m
		switch p.peek(at + n).kind {
		case ctaCommaTok:
			n++
		case ctaRParen:
			return n + 1
		default:
			return 0
		}
	}
}

// rangeMemberLength is how many tokens at offset at spell one member of an
// integer sequence: 3 for `IntegerLiteral 'to' IntegerLiteral`, 1 for an
// IntegerLiteral alone, and 0 for anything else.
func (p *ctaParser) rangeMemberLength(at int) int {
	if !ctaIntegerLiteral(p.peek(at)) {
		return 0
	}
	if to := p.peek(at + 1); to.kind == ctaNameTok && to.text == "to" && ctaIntegerLiteral(p.peek(at+2)) {
		return 3
	}
	return 1
}

// ctaIntegerLiteral reports whether tok is xpath20.md [71] IntegerLiteral, a
// NumericLiteral of digits alone.
func ctaIntegerLiteral(tok ctaToken) bool {
	if tok.kind != ctaNumberTok {
		return false
	}
	for _, r := range tok.text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// integerSequence parses the n tokens integerSequenceLength measured at the
// cursor into the ctaIntegerRanges they spell (xpath20.md §3.3.1: "If either
// operand is an empty sequence, or if the integer derived from the first
// operand is greater than the integer derived from the second operand, the
// result of the range expression is an empty sequence"; the comma operator
// concatenates its operands in order), or into ctaEmptyValue where it holds no
// item, as `(1 to 0)` does. It declines where xs:integer does not resolve.
//
// GAP(xpath): every other sequence expression declines: a RangeExpr or a comma
// anywhere but as a general comparison's operand — a value comparison's, an
// arithmetic or function operand, a cast's, an ·effective boolean value· — a
// range operand that is not an IntegerLiteral, as in `1 to .`, `1 to $value`
// and `1 + 1 to 3`, a member that is neither, a nested or empty parenthesis,
// an IntegerLiteral beyond int64, and a sequence of more than
// ctaMaxSequenceLength items. The direction is the withhold
// [CompileAssertionTest] and [FacetAssertions] report. (#1042)
func (p *ctaParser) integerSequence(n int) (ctaValue, bool) {
	end := p.pos + n
	var ranges []ctaIntegerRange
	total := int64(0)
	for p.pos < end {
		if !ctaIntegerLiteral(p.peek(0)) {
			p.advance() // '(', ',' or ')'
			continue
		}
		lo, err := strconv.ParseInt(p.peek(0).text, 10, 64)
		if err != nil {
			return nil, false
		}
		hi := lo
		if p.rangeMemberLength(0) == 3 {
			if hi, err = strconv.ParseInt(p.peek(2).text, 10, 64); err != nil {
				return nil, false
			}
			p.advance() // the first IntegerLiteral
			p.advance() // 'to'
		}
		p.advance() // the last IntegerLiteral
		if hi >= lo {
			// Neither literal is negative, so hi-lo cannot overflow.
			if hi-lo >= ctaMaxSequenceLength-total {
				return nil, false
			}
			total += hi - lo + 1
		}
		ranges = append(ranges, ctaIntegerRange{lo: lo, hi: hi})
	}
	if total == 0 {
		return ctaEmptyValue{}, true
	}
	integer, resolved := p.types.simple(ctaBuiltin("integer"))
	if !resolved {
		return nil, false
	}
	return ctaIntegerRanges{ranges: ranges, st: integer}, true
}
