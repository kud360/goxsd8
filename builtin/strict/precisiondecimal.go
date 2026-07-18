package strict

import (
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// precisionDecimalLexical is the precisionDecimal lexical space (xsd-precisionDecimal
// §3.2, pDecimalRep): an optional-sign numeral with an optional fractional tail and
// optional [Ee] exponent, OR one of exactly four special literals INF/+INF/-INF/NaN.
// The special sub-grammar is deliberately narrow: only 'INF' and 'NaN' are chosen,
// never the IEEE 'INFINITY' spelling or any case variant (§3.2 Note), and the NaN
// alternative carries no sign, so "+NaN"/"-NaN" are rejected while "+INF" is accepted.
// Anchored whole-string; whiteSpace=collapse is a pre-lexical pipeline stage, so
// Parse rejects any stray whitespace here. The production is byte-identical to
// float/double's regex because they share numericalSpecialRep, but it is kept
// separate: they are distinct lexical spaces (§3.2 vs §3.3.4.2/§3.3.5.2) whose spec
// citations must not be coupled through a shared variable.
var precisionDecimalLexical = regexp.MustCompile(`^((\+|-)?([0-9]+(\.[0-9]*)?|\.[0-9]+)([Ee](\+|-)?[0-9]+)?|(\+|-)?INF|NaN)$`)

// pdKind is the discriminated-union tag of a precisionDecimal value: the numeric
// arm carries a (coefficient, scale, sign) triple, while the three special values
// carry NONE of them — the spec's own iff-absence rules (·scale· absent iff
// numericalValue is special; ·sign· absent iff notANumber) become unrepresentable
// illegal states rather than runtime invariants (warden guardrail, STYLE T7).
type pdKind uint8

const (
	// pdNumeric is a numerical value: coefficient/scale/sign are meaningful.
	pdNumeric pdKind = iota
	// pdPosInf is positiveInfinity; scale and sign are absent (sign is implied).
	pdPosInf
	// pdNegInf is negativeInfinity; scale and sign are absent (sign is implied).
	pdNegInf
	// pdNaN is notANumber; scale and sign are both absent.
	pdNaN
)

// pdSign is the ·sign· property of a numeric precisionDecimal value. It is stored
// in its own field, NOT derived from the coefficient, because it is the only fact
// that distinguishes +0 from −0 — a distinction the spec keeps for canonical output
// and for Identical (§3.1: ·sign· is redundant "except when numericalValue is zero").
type pdSign uint8

const (
	// signPositive is ·sign· = positive.
	signPositive pdSign = iota
	// signNegative is ·sign· = negative.
	signNegative
)

// precisionDecimalVal is an xs:precisionDecimal value (xsd-precisionDecimal §3.1),
// the (·numericalValue·, ·scale·, ·sign·) triple. The numericalValue magnitude is
// coefficient × 10^(-scale) with coefficient an integer ≥ 0; scale is preserved
// VERBATIM from the lexical, so 3, 3.0 and 3.00 are distinct values (coefficient/
// scale (3,0), (30,1), (300,2)) that nonetheless compare numerically equal
// (PRINCIPLES 18). It is value.Ordered with a PARTIAL order (§3.1: NaN incomparable
// with everything including itself), value.Eq, value.Identical (scale-sensitive),
// value.Scaled and value.DigitCounted (totalDigits) — and value.Canonical. It is
// deliberately NOT value.Lengthed/TimezoneAware: no precisionDecimal-applicable
// facet needs them (§3.3).
//
// maxScale/minScale, precisionDecimal's two extension facets (§4.2/§4.3), are
// enforced at instance validation by value/facets.go's scaleFacet, which reads
// ·scale· through the value.Scaled capability this value model implements below
// (cvc-maxScale-valid, cvc-minScale-valid; #133). This mapping supplies the
// value; the facet check lives in the backend-generic pipeline.
type precisionDecimalVal struct {
	kind pdKind
	// coefficient is the integer significand magnitude (≥ 0); numeric arm only.
	coefficient *big.Int
	// scale is ·scale· (aP), kept verbatim from the lexical; numeric arm only.
	scale int
	// sign is ·sign·; numeric arm only. Stored, not derived (distinguishes ±0).
	sign pdSign
}

// parsePrecisionDecimal maps a pDecimalRep to its value (·precisionDecimalLexicalMap·,
// §3.2 steps 1–4): the special literals resolve to their kind, and a numeral splits
// into a magnitude coefficient (intPart+fracPart digits) and a ·scale· of
// len(fracPart) − exponent, so 3.0e2 → (coefficient 30, scale −1) and 3.00 →
// (coefficient 300, scale 2). Trailing/leading zeros are preserved into the scale,
// never stripped: the (coefficient, scale) pair IS the identity.
func parsePrecisionDecimal(lexical string, _ value.Context) (value.Value, error) {
	if !precisionDecimalLexical.MatchString(lexical) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"precisionDecimal: %q is not in the lexical space (pDecimalRep, §3.2)", lexical)
	}

	// specialRepValue (step 1, otherwise clause): only these four literals.
	switch lexical {
	case "INF", "+INF":
		return precisionDecimalVal{kind: pdPosInf}, nil
	case "-INF":
		return precisionDecimalVal{kind: pdNegInf}, nil
	case "NaN":
		return precisionDecimalVal{kind: pdNaN}, nil
	}

	sign := signPositive
	body := lexical
	switch body[0] {
	case '+':
		body = body[1:]
	case '-':
		sign = signNegative
		body = body[1:]
	}

	exp := 0
	if i := strings.IndexAny(body, "Ee"); i >= 0 {
		e, err := strconv.Atoi(body[i+1:])
		if err != nil {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
				"precisionDecimal: %q has an out-of-range exponent (pDecimalRep, §3.2)", lexical)
		}
		exp = e
		body = body[:i]
	}

	intPart, fracPart := body, ""
	if i := strings.IndexByte(body, '.'); i >= 0 {
		intPart, fracPart = body[:i], body[i+1:]
	}

	// The regex guarantees at least one digit across intPart+fracPart, so SetString
	// cannot fail; leading zeros are absorbed by base-10 parsing. The coefficient is
	// the magnitude — the sign is a separate stored fact.
	coeff, ok := new(big.Int).SetString(intPart+fracPart, 10)
	if !ok {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"precisionDecimal: %q has no digits (pDecimalRep, §3.2)", lexical)
	}

	// ·scale· (step 2): decimalPtPrecision (len fracPart) for a plain numeral,
	// scientificPrecision (that count minus the exponent) for scientific notation.
	return precisionDecimalVal{kind: pdNumeric, coefficient: coeff, scale: len(fracPart) - exp, sign: sign}, nil
}

// numericalValue returns the exact ·numericalValue· as a reduced rational (numeric
// arm only): ± coefficient × 10^(-scale). big.Rat reduction makes the value
// quantum-blind — 3.00 (300/100) and 3 (3/1) reduce to the same 3 — which is what
// Eq/Cmp compare (§3.1: "ordered … as their numericalValue values are ordered").
func (p precisionDecimalVal) numericalValue() *big.Rat {
	r := new(big.Rat).SetInt(p.coefficient)
	pow := new(big.Int).Exp(bigTen, big.NewInt(int64(abs(p.scale))), nil)
	switch {
	case p.scale > 0:
		r.Quo(r, new(big.Rat).SetInt(pow))
	case p.scale < 0:
		r.Mul(r, new(big.Rat).SetInt(pow))
	}
	if p.sign == signNegative {
		r.Neg(r)
	}
	return r
}

// abs returns the absolute value of n.
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Cmp is the PARTIAL order on precisionDecimal (§3.1): −INF < every numeric value <
// +INF, numeric values order by ·numericalValue· (scale-blind), and NaN is
// Incomparable with everything, itself included. A non-precisionDecimal argument is
// Incomparable rather than a spurious order (rf-ordered).
func (p precisionDecimalVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(precisionDecimalVal)
	if !ok {
		return value.Incomparable
	}
	if p.kind == pdNaN || o.kind == pdNaN {
		return value.Incomparable
	}
	switch p.kind {
	case pdPosInf:
		if o.kind == pdPosInf {
			return value.Equal
		}
		return value.Greater
	case pdNegInf:
		if o.kind == pdNegInf {
			return value.Equal
		}
		return value.Less
	case pdNumeric, pdNaN:
		// p is numeric (NaN returned above); fall through to the numeric branch.
	}
	switch o.kind {
	case pdPosInf:
		return value.Less
	case pdNegInf:
		return value.Greater
	case pdNumeric, pdNaN:
		// both numeric (o NaN returned above); compare numericalValue below.
	}
	switch p.numericalValue().Cmp(o.numericalValue()) {
	case -1:
		return value.Less
	case 1:
		return value.Greater
	}
	return value.Equal
}

// Eq is precisionDecimal equality (§3.1): it coincides with the order's Equal, so it
// is scale-blind (1.0 = 1.00), NaN ≠ NaN (Incomparable ⇒ not Equal) and each infinity
// equals only itself. Scale-sensitive distinctness lives on Identical, not here.
func (p precisionDecimalVal) Eq(other value.Value) bool { return p.Cmp(other) == value.Equal }

// Identical is the precisionDecimal identity relation (PRINCIPLES 18, §2.2.2),
// DISTINCT from Eq: it folds in ·scale· and ·sign·, so 3, 3.0 and 3.00 are three
// distinct values and +0 is not identical to −0, while NaN IS identical to itself
// (the single notANumber value). Enumeration matching (cvc-enumeration-valid,
// §4.3.5.4) reads this, not Eq.
func (p precisionDecimalVal) Identical(other value.Value) bool {
	o, ok := other.(precisionDecimalVal)
	if !ok {
		return false
	}
	if p.kind != o.kind {
		return false
	}
	if p.kind != pdNumeric {
		return true // NaN ≡ NaN, INF ≡ INF, −INF ≡ −INF
	}
	return p.sign == o.sign && p.scale == o.scale && p.coefficient.Cmp(o.coefficient) == 0
}

// Scale returns ·scale· (§3.1); ok is false for the special values, whose ·scale·
// is absent, encoding the spec's "absent iff numericalValue is a special value".
func (p precisionDecimalVal) Scale() (int, bool) {
	if p.kind != pdNumeric {
		return 0, false
	}
	return p.scale, true
}

// TotalDigits is the value cvc-totalDigits-valid reads (§4.1): for a nonzero numeric
// value with ·numericalValue· nV and ·scale· aP the rule requires
// (aP + 1 + log10(|nV|) div 1) ≤ t. With nV = coefficient × 10^(-aP) and a
// D-digit coefficient, floor(log10(|nV|)) = (D−1) − aP, so the whole expression
// collapses to D — the coefficient's digit count (trailing zeros INCLUDED, unlike
// decimal, which counts the trailing-zero-stripped minimal form). Zero and the
// specials are unconditionally facet-valid (§4.1 clause 1, avoiding log10(0)); they
// report 1 so the pipeline's ≤ t check passes for every t ≥ 1.
func (p precisionDecimalVal) TotalDigits() int {
	if p.kind != pdNumeric || p.coefficient.Sign() == 0 {
		return 1
	}
	return len(p.coefficient.String())
}

// FractionDigits satisfies the value.DigitCounted interface but is inert for
// precisionDecimal: the fractionDigits facet is NOT applicable to this type (§3.3),
// so the facet pipeline never invokes it. It reports ·scale· (0 for the specials) as
// a spec-consistent value rather than a placeholder.
func (p precisionDecimalVal) FractionDigits() int {
	if p.kind != pdNumeric {
		return 0
	}
	return p.scale
}

// canonicalPrecisionDecimal is the Mapping.Canonical wrapper: it rejects a foreign
// value as an *xsderr.Error rather than panicking (warden guardrail).
func canonicalPrecisionDecimal(v value.Value) (string, error) {
	p, ok := v.(precisionDecimalVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"precisionDecimal canonical: value of type %T is not a strict precisionDecimal", v)
	}
	return p.Canonical(), nil
}

// Canonical renders the canonical pDecimalRep (·precisionDecimalCanonicalMap·, §6):
// the specials map to their fixed literal (step 2); an integer with ·scale· 0 in
// [1E−6, 1E6] renders as a bare numeral (step 3); a positive ·scale· in range renders
// with the decimal point placed and trailing zeros PADDED to the scale (step 4, so 3
// with scale 2 → "3.00"); everything else — a negative scale, or a magnitude outside
// the range — renders in scientific notation, likewise padded to preserve the scale
// (step 5). Trailing zeros are canonical, never stripped: that is how the canonical
// form is quantum-preserving even though Eq is quantum-blind.
func (p precisionDecimalVal) Canonical() string {
	switch p.kind {
	case pdNaN:
		return "NaN"
	case pdPosInf:
		return "INF"
	case pdNegInf:
		return "-INF"
	case pdNumeric:
		// Rendered by the numeric algorithm below.
	}

	sign := ""
	if p.sign == signNegative {
		sign = "-"
	}

	// Zero has no log10 (§4.1) and is excluded from steps 3/4 by the 1E−6 lower
	// bound, so it takes step 5 with scientificCanonicalMap(0) = "0.0E0": the
	// mantissa "0.0" (f = 1) padded with aP − 1 trailing zeros to preserve ·scale·.
	if p.coefficient.Sign() == 0 {
		pad := p.scale - 1
		if pad < 0 {
			pad = 0
		}
		return sign + "0.0" + strings.Repeat("0", pad) + "E0"
	}

	digits := p.coefficient.String()
	if p.scale >= 0 && p.inCanonicalRange() {
		if p.scale == 0 {
			return sign + digits // step 3: bare numeral
		}
		for len(digits) <= p.scale { // step 4: decimal point, trailing zeros preserved
			digits = "0" + digits
		}
		return sign + digits[:len(digits)-p.scale] + "." + digits[len(digits)-p.scale:]
	}
	return sign + p.scientificCanonical(digits)
}

// inCanonicalRange reports whether |numericalValue| lies in [1E−6, 1E6], the range in
// which precisionDecimalCanonicalMap steps 3 and 4 use plain (non-scientific) forms.
// Called only for a nonzero numeric value with scale ≥ 0. With |nV| = C × 10^(-aP):
// |nV| ≤ 1E6 ⟺ C ≤ 10^(aP+6), and |nV| ≥ 1E−6 ⟺ C ≥ 10^(aP−6) (automatic for aP ≤ 6).
func (p precisionDecimalVal) inCanonicalRange() bool {
	upper := new(big.Int).Exp(bigTen, big.NewInt(int64(p.scale+6)), nil)
	if p.coefficient.Cmp(upper) > 0 {
		return false
	}
	if p.scale <= 6 {
		return true
	}
	lower := new(big.Int).Exp(bigTen, big.NewInt(int64(p.scale-6)), nil)
	return p.coefficient.Cmp(lower) >= 0
}

// scientificCanonical renders precisionDecimalCanonicalMap step 5 for a nonzero
// magnitude: strip the coefficient's trailing zeros to the minimal significand C'
// (a factor of 10^k), form the one-leading-digit mantissa m with exponent
// exp = (len(C')−1) + k − scale, then append aP + exp − f trailing zeros (f = m's
// fractional-digit count) so the printed scale matches ·scale·. digits is the
// coefficient magnitude string. The sign is prepended by the caller.
func (p precisionDecimalVal) scientificCanonical(digits string) string {
	cPrime := digits
	k := 0
	for len(cPrime) > 1 && cPrime[len(cPrime)-1] == '0' {
		cPrime = cPrime[:len(cPrime)-1]
		k++
	}
	exp := len(cPrime) - 1 + k - p.scale

	mantissa, frac := cPrime, 0
	if len(cPrime) == 1 {
		mantissa, frac = cPrime+".0", 1
	}
	if len(cPrime) > 1 {
		mantissa, frac = cPrime[:1]+"."+cPrime[1:], len(cPrime)-1
	}

	pad := p.scale + exp - frac
	if pad < 0 {
		pad = 0
	}
	return mantissa + strings.Repeat("0", pad) + "E" + strconv.Itoa(exp)
}
