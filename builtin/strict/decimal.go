package strict

import (
	"math/big"
	"regexp"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// decimalLexical is the decimal lexical space (Datatypes §3.3.3.1,
// decimal-lexical-representation): an optional sign then either a digit run
// with an optional fractional tail, or a bare fractional tail. There is NO
// exponent production — "1E2" is a decimal lexical-space violation (unlike
// float/double). Anchored so the whole literal must match; whiteSpace handling
// is a pre-lexical pipeline stage, so Parse rejects any stray whitespace here.
var decimalLexical = regexp.MustCompile(`^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)$`)

// decimalVal is a decimal value kept as unscaled × 10^-scale, normalized to
// MINIMAL form at parse time (no trailing fractional zeros, scale ≥ 0). big.Int
// (not big.Rat, which would reduce 1020/100 to 51/5 and destroy the scale that
// fractionDigits needs). Because the value space collapses precision — 2.0 is
// not distinct from 2.00 (dt-decimal-datatype) — the minimal form is the value:
// two decimals are equal iff their minimal (unscaled, scale) pairs agree.
type decimalVal struct {
	unscaled *big.Int
	scale    int
}

var bigTen = big.NewInt(10)

// parseDecimal maps a decimal lexical to its value (f-decimalLexmap,
// §3.3.3.2), normalizing to minimal form immediately so equality, canonical
// form and digit counts all read off one stored fact (STYLE D3).
func parseDecimal(lexical string, _ value.Context) (value.Value, error) {
	if !decimalLexical.MatchString(lexical) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"decimal: %q is not in the lexical space (decimal-lexical-representation, §3.3.3.1)", lexical)
	}

	neg := false
	body := lexical
	switch body[0] {
	case '+':
		body = body[1:]
	case '-':
		neg = true
		body = body[1:]
	}

	intPart, fracPart := body, ""
	if i := strings.IndexByte(body, '.'); i >= 0 {
		intPart, fracPart = body[:i], body[i+1:]
	}

	// The regex guarantees at least one digit across intPart+fracPart, so
	// SetString cannot fail; leading zeros are absorbed by base-10 parsing.
	unscaled, ok := new(big.Int).SetString(intPart+fracPart, 10)
	if !ok {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"decimal: %q has no digits (decimal-lexical-representation, §3.3.3.1)", lexical)
	}
	if neg {
		unscaled.Neg(unscaled)
	}
	scale := len(fracPart)

	// Strip trailing fractional zeros so 1.20 → (12, 1) and 2.00 → (2, 0):
	// the value collapses precision, so the minimal pair is the identity.
	q, mod := new(big.Int), new(big.Int)
	for scale > 0 {
		q.QuoRem(unscaled, bigTen, mod)
		if mod.Sign() != 0 {
			break
		}
		unscaled.Set(q)
		scale--
	}

	return decimalVal{unscaled: unscaled, scale: scale}, nil
}

// canonicalDecimal is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail).
func canonicalDecimal(v value.Value) (string, error) {
	d, ok := v.(decimalVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"decimal canonical: value of type %T is not a strict decimal", v)
	}
	return d.Canonical(), nil
}

// Canonical renders the canonical decimal lexical (f-decimalCanmap, §3.3.3.2):
// an integer value gets NO decimal point ("+1.0" → "1"); otherwise a mandatory
// point with at least one digit on each side and no superfluous zeros
// ("010.20" → "10.2"). No leading '+'.
func (d decimalVal) Canonical() string {
	if d.scale == 0 {
		return d.unscaled.String()
	}
	abs := new(big.Int).Abs(d.unscaled).String()
	for len(abs) <= d.scale {
		abs = "0" + abs
	}
	intPart := abs[:len(abs)-d.scale]
	fracPart := abs[len(abs)-d.scale:]
	sign := ""
	if d.unscaled.Sign() < 0 {
		sign = "-"
	}
	return sign + intPart + "." + fracPart
}

// Cmp is the total order on decimal (§3.3.3.2, ordered=total). A non-decimal
// argument is Incomparable rather than a spurious order (rf-ordered).
func (d decimalVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(decimalVal)
	if !ok {
		return value.Incomparable
	}
	l, r := alignDecimals(d, o)
	switch l.Cmp(r) {
	case -1:
		return value.Less
	case 1:
		return value.Greater
	}
	return value.Equal
}

// Eq is decimal equality, which is identity in a precision-collapsed space
// (§2.2.2, no decimal carve-out): it coincides with the total order's Equal.
func (d decimalVal) Eq(other value.Value) bool { return d.Cmp(other) == value.Equal }

// FractionDigits is the count of fractional digits of the minimal value
// (rf-fractionDigits) — the normalized scale, so "010.20" reports 1, not 2.
func (d decimalVal) FractionDigits() int { return d.scale }

// TotalDigits is the count of significant decimal digits of the minimal value
// (rf-totalDigits) — the digit count of the minimal unscaled coefficient, so
// "010.20" reports 3, not 5. The zero value has no nonzero digits; XSD offers
// no explicit rule for it, so we report 1 ("0" is one digit) to keep
// totalDigits ≥ 1 for every value.
func (d decimalVal) TotalDigits() int {
	if d.unscaled.Sign() == 0 {
		return 1
	}
	return len(new(big.Int).Abs(d.unscaled).String())
}

// alignDecimals rescales a and b to a common scale and returns the two
// integers to compare. New big.Ints throughout: it never mutates a receiver.
func alignDecimals(a, b decimalVal) (*big.Int, *big.Int) {
	if a.scale == b.scale {
		return a.unscaled, b.unscaled
	}
	common := a.scale
	if b.scale > common {
		common = b.scale
	}
	return scaleTo(a.unscaled, common-a.scale), scaleTo(b.unscaled, common-b.scale)
}

// scaleTo returns x × 10^n (n ≥ 0) without mutating x.
func scaleTo(x *big.Int, n int) *big.Int {
	if n == 0 {
		return x
	}
	pow := new(big.Int).Exp(bigTen, big.NewInt(int64(n)), nil)
	return new(big.Int).Mul(x, pow)
}
