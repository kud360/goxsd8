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
// whose members are either all IntegerLiterals and such ranges
// (ctaIntegerRanges) or all StringLiterals (ctaStringSequence) — so `. = (1 to
// 10, 20, 30)`, `$value = 1 to 10` and `r = ('black wins', 'draw')` compile.
//
// An integer sequence is materialized when the comparison reads it
// (ctaAtoms), so its length is fixed at compile time from the literals alone,
// and bounded by ctaMaxSequenceLength; a string sequence holds one item per
// member token.

// ctaMaxSequenceLength is the most items an integer sequence may hold for
// ctaParser.integerSequence to compile it: every item is built when the
// comparison over it is evaluated, once per evaluation.
const ctaMaxSequenceLength = 1 << 12

// ctaIntegerRanges is a sequence of xs:integer values built by §3.3.1's
// operators from IntegerLiterals: the concatenation, in written order, of
// each range's integers from lo to hi inclusive — "a sequence containing two
// or more consecutive integers ... in increasing order", or one integer where
// lo is hi. Each range is non-empty: integerSequence stores none whose hi is
// less than its lo. st is xs:integer, the type a RangeExpr's items have and an
// IntegerLiteral's value is (xpath20.md §3.1.1), which is the operand's static
// type (ctaStaticOf).
//
// Its one constructor is ctaParser.integerSequence, which holds at least one
// item and at most ctaMaxSequenceLength: a sequence of none is the statically
// empty ctaEmptyValue instead.
type ctaIntegerRanges struct {
	ranges []ctaIntegerRange
	st     *xsd.SimpleType
}

// ctaIntegerRange is one non-empty member of a ctaIntegerRanges: `lo to hi`,
// lo at most hi, or one IntegerLiteral, where lo is hi.
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
// xpath20.md [10] ComparisonExpr makes [11] RangeExprs: a sequence where the
// tokens at the cursor spell one (sequenceLength), and an additiveExpr
// otherwise.
func (p *ctaParser) generalOperand() (ctaValue, bool) {
	if n := p.sequenceLength(0); n > 0 {
		return p.sequence(n)
	}
	return p.additiveExpr()
}

// sequenceLength is how many tokens, from offset at ahead of the cursor, spell
// a sequence a general comparison's operand may be: an integer sequence
// (integerSequenceLength) or a string sequence (stringSequenceLength), and 0
// where they spell neither. Nothing is consumed.
func (p *ctaParser) sequenceLength(at int) int {
	if n := p.integerSequenceLength(at); n > 0 {
		return n
	}
	return p.stringSequenceLength(at)
}

// sequence parses the n tokens sequenceLength measured at the cursor: a string
// sequence where its first member is a StringLiteral, which only a string
// sequence opens with, and an integer sequence otherwise.
func (p *ctaParser) sequence(n int) (ctaValue, bool) {
	if p.peek(1).kind == ctaStringTok {
		return p.stringSequence(n), true
	}
	return p.integerSequence(n)
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
// and `1 + 1 to 3`, a member that is none of an IntegerLiteral, such a range
// and a StringLiteral, a sequence mixing StringLiterals with the other two, as
// `('a', 1)` does — integerSequenceLength and stringSequenceLength both answer
// 0, and no other production takes that parenthesis and comma — a nested or
// empty parenthesis, a misplaced comma, as in `('a',)`, an IntegerLiteral
// beyond int64, and a sequence of more than ctaMaxSequenceLength items. The
// direction is the withhold [CompileAssertionTest] and [FacetAssertions]
// report. (#1042)
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
		if hi < lo {
			continue // an empty range, which contributes no item
		}
		// Neither literal is negative, so hi-lo cannot overflow.
		if hi-lo >= ctaMaxSequenceLength-total {
			return nil, false
		}
		total += hi - lo + 1
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

// ctaStringSequence is a sequence of xs:string values built by §3.3.1's comma
// operator from StringLiterals: one item per member, in written order, two
// equal members two items. st is xs:string, the type a StringLiteral's value
// is (xpath20.md §3.1.1), which is the operand's static type (ctaStaticOf).
//
// Its one constructor is ctaParser.stringSequence, which holds at least one
// member.
type ctaStringSequence struct {
	texts []string
	st    *xsd.SimpleType
}

func (ctaStringSequence) ctaValue() {}

// readsChild is false: the sequence reads nothing of the instance.
func (ctaStringSequence) readsChild(xsd.QName) bool { return false }

// counted appends nothing: the sequence is no fn:count call.
func (ctaStringSequence) counted(into []ctaTallied) []ctaTallied { return into }

// ctaStringSequenceItem is n's strings in written order, each converted into c
// as a ctaLiteral's text is (ctaConvert): the comparison type
// ctaTypes.comparison settled from xs:string, so the literal is never
// validated straight into the other operand's type.
func ctaStringSequenceItem(n ctaStringSequence, c *xsd.SimpleType, env ctaEnv) ctaItem {
	vs := make([]value.Value, 0, len(n.texts))
	for _, text := range n.texts {
		v, ok := ctaValidated(ctaConvert(text, n.st, c, env))
		if !ok {
			return ctaRaised{}
		}
		vs = append(vs, v)
	}
	return ctaAtoms{vs: vs}
}

// stringSequenceLength is how many tokens, from offset at ahead of the cursor,
// spell a string sequence — `'(' StringLiteral (',' StringLiteral)* ')'` —
// and 0 where they spell none or the façade constructs no sequence
// (ctaFacade.constructsSequences). Nothing is consumed.
func (p *ctaParser) stringSequenceLength(at int) int {
	if !p.facade.constructsSequences() || p.peek(at).kind != ctaLParen {
		return 0
	}
	n := 1
	for {
		if p.peek(at+n).kind != ctaStringTok {
			return 0
		}
		n++
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

// stringSequence parses the n tokens stringSequenceLength measured at the
// cursor into the ctaStringSequence they spell (xpath20.md §3.3.1: the comma
// operator concatenates its operands in order), typed by ctaTypes.str, the
// xs:string a ctaLiteral's StringLiteral has. It never declines;
// integerSequence's GAP(xpath) states every sequence that does.
func (p *ctaParser) stringSequence(n int) ctaStringSequence {
	end := p.pos + n
	var texts []string
	for p.pos < end {
		if p.at(ctaStringTok) {
			texts = append(texts, p.peek(0).text)
		}
		p.advance() // '(', a StringLiteral, ',' or ')'
	}
	return ctaStringSequence{texts: texts, st: p.types.str}
}
