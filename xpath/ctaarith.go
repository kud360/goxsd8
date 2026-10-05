package xpath

import (
	"math"
	"math/big"
	"strconv"

	"github.com/kud360/goxsd8/xsd"
)

// This file evaluates xpath20.md §3.4's binary arithmetic operators over
// numeric operands, which the assertion and facet façades admit
// (ctaFacade.computes) and a Type Alternative's declines. The operator
// functions are F&O §6.2's: op:numeric-add, -subtract, -multiply, -divide,
// -integer-divide and -mod.
//
// value exposes no arithmetic capability, so each operand is read through its
// ·canonical representation· in the operation type — the round-trip ctaPromote
// converts through — computed here, and the result's lexical validated against
// the result type, which is a datatype validation as every value this package
// builds is.

// ctaArithOp is one of the six binary arithmetic operators of xpath20.md [13]
// AdditiveExpr and [14] MultiplicativeExpr.
type ctaArithOp byte

const (
	ctaAdd ctaArithOp = iota
	ctaSubtract
	ctaMultiply
	ctaDivide
	ctaIntegerDivide
	ctaModulus
)

// ctaArith is one binary arithmetic operator over two operands (xpath20.md
// §3.4), whose types were settled at compile time by ctaTypes.arithmetic:
// operation is the numeric primitive both atomized operands are converted into
// (B.1), and st the type of the result (B.2), which is why the operand's
// static type is st. The two differ for `idiv` (always xs:integer), for an
// xs:integer result computed in xs:decimal, and agree otherwise.
type ctaArith struct {
	op          ctaArithOp
	operation   *xsd.SimpleType
	st          *xsd.SimpleType
	left, right ctaValue
}

func (ctaArith) ctaValue() {}

// ctaDecimalDivisionDigits is the number of fractional digits an xs:decimal
// quotient that does not terminate is rounded to: F&O §6.2 makes "the number
// of digits of precision returned by the numeric operators" for xs:decimal
// ·implementation-defined·, and lets a result exceeding it be "truncated or
// rounded in an ·implementation-defined· manner". This engine rounds to
// nearest, halves away from zero ([big.Rat.FloatString]). Every other
// xs:decimal result, a terminating quotient included, is exact.
const ctaDecimalDivisionDigits = 18

// ctaArithItem evaluates n and converts its result into c on ctaTypedAttrItem's
// terms. xpath20.md §3.4's steps are applied to each operand in order, the left
// one first, on ctaValueCompare.eval's terms: an operand that raised is the
// error; an EMPTY one makes the result the empty sequence; one of more than one
// item is err:XPTY0004. A dynamic error the operator function raises —
// err:FOAR0001 for a division by zero, err:FOAR0002 for an `idiv` over NaN or
// an infinite dividend — is ctaRaised.
func ctaArithItem(n ctaArith, c *xsd.SimpleType, env ctaEnv) ctaItem {
	l, settled, single := ctaArithOperand(n.left, n.operation, env)
	if !single {
		return settled
	}
	r, settled, single := ctaArithOperand(n.right, n.operation, env)
	if !single {
		return settled
	}
	lexical, computed := n.compute(l, r)
	if !computed {
		return ctaRaised{}
	}
	v, validated := ctaValidated(ctaValidate(lexical, n.st, env))
	if !validated {
		return ctaRaised{}
	}
	return ctaPromote(v, n.st, c, env)
}

// ctaArithOperand evaluates one arithmetic operand into operation, reporting
// the ·canonical representation· of its one atomic value, or false with the
// item the whole expression takes without applying its operator: ctaRaised
// for a raised operand, one of two or more items (err:XPTY0004) or a value
// that does not render, and the empty sequence for an empty one.
func ctaArithOperand(v ctaValue, operation *xsd.SimpleType, env ctaEnv) (string, ctaItem, bool) {
	atoms, converted := ctaItemOf(v, operation, env).(ctaAtoms)
	if !converted || len(atoms.vs) > 1 {
		return "", ctaRaised{}, false
	}
	if len(atoms.vs) == 0 {
		return "", ctaAtoms{}, false
	}
	lexical, rendered := ctaCanonical(atoms.vs[0], operation, env)
	if !rendered {
		return "", ctaRaised{}, false
	}
	return lexical, nil, true
}

// compute applies n's operator to two canonical lexicals of n.operation,
// reporting the lexical of the result in n.st, or false where the operator
// function raises. The default arm is unreachable: ctaTypes.arithmetic admits
// the three numeric primitives alone.
func (n ctaArith) compute(l, r string) (string, bool) {
	switch n.operation.Name() {
	case ctaBuiltin("decimal"):
		return n.op.decimal(l, r)
	case ctaBuiltin("float"):
		return n.op.floating(l, r, 32)
	case ctaBuiltin("double"):
		return n.op.floating(l, r, 64)
	}
	return "", false
}

// decimal applies op to two xs:decimal lexicals (F&O §6.2), exactly, but for
// a non-terminating quotient (ctaDecimalDivisionDigits). A zero divisor of
// `div`, `idiv` or `mod` raises err:FOAR0001 (§6.2.4, §6.2.5, §6.2.6), which is
// false here. No xs:decimal operation overflows: the values are unbounded.
func (op ctaArithOp) decimal(l, r string) (string, bool) {
	a, parsed := new(big.Rat).SetString(l)
	if !parsed {
		return "", false
	}
	b, parsed := new(big.Rat).SetString(r)
	if !parsed {
		return "", false
	}
	switch op {
	case ctaAdd:
		return ctaDecimalLexical(new(big.Rat).Add(a, b)), true
	case ctaSubtract:
		return ctaDecimalLexical(new(big.Rat).Sub(a, b)), true
	case ctaMultiply:
		return ctaDecimalLexical(new(big.Rat).Mul(a, b)), true
	default:
		return op.decimalDivision(a, b)
	}
}

// decimalDivision is decimal's arm for the three division operators, `div`,
// `idiv` and `mod`, each of which raises err:FOAR0001 for a zero divisor.
func (op ctaArithOp) decimalDivision(a, b *big.Rat) (string, bool) {
	if b.Sign() == 0 {
		return "", false // err:FOAR0001
	}
	switch op {
	case ctaIntegerDivide:
		return ctaTruncatedQuotient(a, b).String(), true
	case ctaModulus:
		// §6.2.6: (a idiv b)*b + (a mod b) = a, so the sign follows the
		// dividend.
		t := new(big.Rat).SetInt(ctaTruncatedQuotient(a, b))
		return ctaDecimalLexical(new(big.Rat).Sub(a, t.Mul(t, b))), true
	default:
		return ctaDecimalLexical(new(big.Rat).Quo(a, b)), true
	}
}

// ctaTruncatedQuotient is a / b truncated toward zero, which is the xs:integer
// op:numeric-integer-divide answers (F&O §6.2.5: "the largest (furthest from
// zero) xs:integer value $N such that fn:abs($N * $arg2) le fn:abs($arg1)"
// with the sign of the exact quotient). b is non-zero.
func ctaTruncatedQuotient(a, b *big.Rat) *big.Int {
	q := new(big.Rat).Quo(a, b)
	return new(big.Int).Quo(q.Num(), q.Denom())
}

// ctaDecimalLexical renders q as an xs:decimal lexical: an integer with no
// fractional part, a terminating fraction with exactly the digits it needs,
// and any other fraction rounded to ctaDecimalDivisionDigits.
func ctaDecimalLexical(q *big.Rat) string {
	if q.IsInt() {
		return q.Num().String()
	}
	rest := new(big.Int).Set(q.Denom())
	twos := ctaStripFactor(rest, 2)
	fives := ctaStripFactor(rest, 5)
	if rest.Cmp(big.NewInt(1)) != 0 {
		return q.FloatString(ctaDecimalDivisionDigits)
	}
	return q.FloatString(max(twos, fives))
}

// ctaStripFactor divides every factor f out of n, in place, and reports how
// many there were.
func ctaStripFactor(n *big.Int, f int64) int {
	divisor := big.NewInt(f)
	quotient, remainder := new(big.Int), new(big.Int)
	count := 0
	for {
		quotient.QuoRem(n, divisor, remainder)
		if remainder.Sign() != 0 {
			return count
		}
		n.Set(quotient)
		count++
	}
}

// floating applies op to two lexicals of the IEEE 754 type of bits bits —
// xs:float at 32, xs:double at 64 — whose special values F&O §6.2 fixes:
// `div` by zero is an infinity or NaN and raises nothing (§6.2.4), `mod` is
// NaN for an infinite dividend or a zero divisor and the dividend for an
// infinite divisor (§6.2.6), all of which Go's IEEE arithmetic and [math.Mod]
// answer as written. Each result is rounded to the type: computing an xs:float
// operation in float64 and rounding once is exact, float64 carrying more than
// twice float32's precision. `idiv` is integerDivide.
func (op ctaArithOp) floating(l, r string, bits int) (string, bool) {
	a, err := strconv.ParseFloat(l, bits)
	if err != nil {
		return "", false
	}
	b, err := strconv.ParseFloat(r, bits)
	if err != nil {
		return "", false
	}
	var x float64
	switch op {
	case ctaAdd:
		x = a + b
	case ctaSubtract:
		x = a - b
	case ctaMultiply:
		x = a * b
	case ctaDivide:
		x = a / b
	case ctaModulus:
		x = math.Mod(a, b)
	case ctaIntegerDivide:
		return ctaFloatIntegerDivide(a, b, bits)
	}
	return ctaFloatLexical(ctaRounded(x, bits), bits), true
}

// ctaFloatIntegerDivide is op:numeric-integer-divide over two IEEE 754 values
// (F&O §6.2.5): a zero divisor raises err:FOAR0001, a NaN operand or an
// infinite dividend err:FOAR0002, an infinite divisor of a finite dividend is
// 0, and otherwise the quotient is `($a div $b) cast as xs:integer` — the `div`
// in the operands' own type, then truncated. A quotient that overflows to an
// infinity is err:FOAR0002 too, "subject to limits of precision and
// overflow/underflow conditions". Every error is false here.
func ctaFloatIntegerDivide(a, b float64, bits int) (string, bool) {
	if b == 0 {
		return "", false // err:FOAR0001
	}
	if math.IsNaN(a) || math.IsNaN(b) || math.IsInf(a, 0) {
		return "", false // err:FOAR0002
	}
	q := ctaRounded(a/b, bits)
	if math.IsInf(q, 0) {
		return "", false // err:FOAR0002
	}
	// Int's Accuracy is not an error: a truncated finite value is exact.
	truncated, _ := big.NewFloat(math.Trunc(q)).Int(nil)
	return truncated.String(), true
}

// ctaRounded is x rounded to the IEEE 754 type of bits bits.
func ctaRounded(x float64, bits int) float64 {
	if bits == 32 {
		return float64(float32(x))
	}
	return x
}

// ctaFloatLexical renders x, a value of the IEEE 754 type of bits bits, as a
// lexical of that type (Datatypes §3.3.4, §3.3.5): the special values spelled
// INF, -INF and NaN, and every other value in scientific notation with the
// fewest digits that map back to it.
func ctaFloatLexical(x float64, bits int) string {
	switch {
	case math.IsNaN(x):
		return "NaN"
	case math.IsInf(x, 1):
		return "INF"
	case math.IsInf(x, -1):
		return "-INF"
	}
	return strconv.FormatFloat(x, 'E', -1, bits)
}
