package strict

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// floatingLexical is the lexical space shared by float and double (§3.3.4.2
// floatRep and §3.3.5.2 doubleRep give byte-for-byte the same production and
// regular expression): an optional-sign numeral with an optional fractional
// tail and optional [Ee] exponent, OR one of the special literals. The special
// sub-grammar (nt-numSpecReps) is stricter than a naive reading suggests — only
// bare NaN, and only +INF among the signed specials, are in the space: the NaN
// alternative carries NO sign, so "+NaN"/"-NaN" are rejected while "+INF" is
// accepted. Anchored whole-string; whiteSpace is a pre-lexical pipeline stage,
// so Parse rejects any stray whitespace here (mirrors decimalLexical).
var floatingLexical = regexp.MustCompile(`^((\+|-)?([0-9]+(\.[0-9]*)?|\.[0-9]+)([Ee](\+|-)?[0-9]+)?|(\+|-)?INF|NaN)$`)

// floatVal is an xs:float value (§3.3.4): IEEE 754 binary32, so a Go float32 is
// the exact underlying representation — it natively holds the whole value space
// including the special values (±0 via the sign bit, ±INF, NaN). It is
// value.Ordered with a PARTIAL order (§3.3.4.3, ordered=partial: NaN is
// incomparable with everything, itself included), value.Eq, value.Identical and
// value.Canonical — and deliberately none of Lengthed/DigitCounted/Scaled, which
// no float-applicable facet needs (cos-applicable-facets §4.1.5).
type floatVal float32

// doubleVal is an xs:double value (§3.3.5): IEEE 754 binary64, a Go float64. It
// carries the identical capability set to floatVal; the ONLY differences between
// the two datatypes are the three IEEE precision constants, which Go's float32
// vs float64 (and strconv's bitSize 32 vs 64) embody directly (§3.3.5 Note).
type doubleVal float64

// parseFloat maps a float lexical to its value (f-floatLexmap, §3.3.4.2):
// binary32 round-to-nearest-ties-to-even, which strconv.ParseFloat with bitSize
// 32 performs (it is IEEE 754-conformant, so an acceptable floatingPointRound).
func parseFloat(lexical string, _ value.Context) (value.Value, error) {
	f, err := parseFloating(lexical, 32, "float")
	if err != nil {
		return nil, err
	}
	return floatVal(float32(f)), nil
}

// parseDouble maps a double lexical to its value (f-doubleLexmap, §3.3.5.2):
// binary64 round-to-nearest-ties-to-even via strconv.ParseFloat bitSize 64.
func parseDouble(lexical string, _ value.Context) (value.Value, error) {
	f, err := parseFloating(lexical, 64, "double")
	if err != nil {
		return nil, err
	}
	return doubleVal(f), nil
}

// parseFloating is the shared lexical mapping for float (bitSize 32) and double
// (bitSize 64). It validates against the shared lexical space, resolves the
// special literals per specialRepValue (f-specRepVal), then rounds numerals to
// the target precision. It returns the value as a float64 (a binary32 result is
// a float64 that is exactly representable as a float32); the caller narrows.
func parseFloating(lexical string, bitSize int, name string) (float64, error) {
	if !floatingLexical.MatchString(lexical) {
		return 0, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"%s: %q is not in the lexical space (%s, §3.3.4.2/§3.3.5.2)", name, lexical, name+"Rep")
	}

	// specialRepValue (f-specRepVal): INF/+INF → +∞, -INF → −∞, NaN → NaN.
	// Handled before ParseFloat so its own lenient "Inf"/"NaN" spellings can
	// never widen the XSD lexical space.
	switch lexical {
	case "INF", "+INF":
		return math.Inf(1), nil
	case "-INF":
		return math.Inf(-1), nil
	case "NaN":
		return math.NaN(), nil
	}

	f, err := strconv.ParseFloat(lexical, bitSize)
	if err != nil {
		// An out-of-range numeral is VALID: floatingPointRound maps a magnitude
		// above the largest finite value to ±INF and one below the smallest
		// subnormal to a signed zero, and ParseFloat reports exactly those
		// results alongside ErrRange (the returned f is already ±Inf or ±0 with
		// the correct sign). Only a syntax error — which the regex already
		// excludes — is a real lexical-space rejection.
		if errors.Is(err, strconv.ErrRange) {
			return f, nil
		}
		return 0, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"%s: %q is not in the lexical space (%s, §3.3.4.2/§3.3.5.2)", name, lexical, name+"Rep")
	}
	return f, nil
}

// canonicalFloat is the Mapping.Canonical wrapper for float: it rejects a
// foreign value as an *xsderr.Error rather than panicking (warden guardrail).
func canonicalFloat(v value.Value) (string, error) {
	f, ok := v.(floatVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"float canonical: value of type %T is not a strict float", v)
	}
	return f.Canonical(), nil
}

// canonicalDouble is the Mapping.Canonical wrapper for double.
func canonicalDouble(v value.Value) (string, error) {
	d, ok := v.(doubleVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"double canonical: value of type %T is not a strict double", v)
	}
	return d.Canonical(), nil
}

// Canonical renders the float canonical lexical (f-floatCanmap, §3.3.4.2).
func (f floatVal) Canonical() string { return floatingCanonical(float64(f), 32) }

// Canonical renders the double canonical lexical (f-doubleCanmap, §3.3.5.2).
func (d doubleVal) Canonical() string { return floatingCanonical(float64(d), 64) }

// floatingCanonical implements floatCanonicalMap/doubleCanonicalMap: the special
// forms INF/-INF/NaN (specialRepCanonicalMap, f-specValCanMap) and 0.0E0/-0.0E0
// for the signed zeros, else scientificCanonicalMap (f-sciCanFragMap) — the
// shortest decimal that round-trips, one leading mantissa digit, a mandatory
// decimal point, an uppercase E and a minimal exponent. strconv.FormatFloat with
// precision -1 gives the shortest round-tripping digits at the target bitSize;
// reshapeScientific reshapes its C-style output into the spec's numeral shape.
func floatingCanonical(f float64, bitSize int) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "INF"
	case math.IsInf(f, -1):
		return "-INF"
	case f == 0:
		if math.Signbit(f) {
			return "-0.0E0"
		}
		return "0.0E0"
	}
	return reshapeScientific(strconv.FormatFloat(f, 'e', -1, bitSize))
}

// reshapeScientific rewrites strconv's 'e'-format output (e.g. "-1.5e+01",
// "1e+00", "1.2e-05") into scientificCanonicalMap's numeral shape: a mantissa
// that always carries a decimal point (unsignedDecimalPtCanonicalMap emits at
// least one fractional digit, so "1e…" becomes "1.0E…"), an uppercase 'E', and
// an exponent with no '+' sign and no leading zeros (noDecimalPtCanonicalMap).
func reshapeScientific(s string) string {
	i := strings.IndexByte(s, 'e')
	mantissa, exp := s[:i], s[i+1:]
	if !strings.ContainsRune(mantissa, '.') {
		mantissa += ".0"
	}
	// strconv always emits a signed, ≥2-digit exponent (e.g. "+01", "-05").
	neg := exp[0] == '-'
	exp = strings.TrimLeft(exp[1:], "0")
	if exp == "" {
		exp = "0"
	}
	if neg && exp != "0" {
		exp = "-" + exp
	}
	return mantissa + "E" + exp
}

// Cmp is the PARTIAL order on float (§3.3.4.3): a non-float argument (a
// different value space, e.g. double) and any pair involving NaN are
// Incomparable (rf-ordered); ±0 compare Equal.
func (f floatVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(floatVal)
	if !ok {
		return value.Incomparable
	}
	return floatingCmp(float64(f), float64(o))
}

// Cmp is the partial order on double (§3.3.5.3); a non-double argument is
// Incomparable.
func (d doubleVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(doubleVal)
	if !ok {
		return value.Incomparable
	}
	return floatingCmp(float64(d), float64(o))
}

// floatingCmp is the shared partial order. Go's < and > already treat +0 and −0
// as equal and order ±INF correctly; only NaN needs an explicit guard, since
// every IEEE comparison against NaN is false and would otherwise fall through to
// a bogus Equal.
func floatingCmp(a, b float64) value.Ordering {
	if math.IsNaN(a) || math.IsNaN(b) {
		return value.Incomparable
	}
	switch {
	case a < b:
		return value.Less
	case a > b:
		return value.Greater
	}
	return value.Equal
}

// Eq is float equality (§3.3.4.1): NaN ≠ NaN and +0 = −0. Go's IEEE == encodes
// exactly this, so Eq reads it DIRECTLY — it is not derived from Identical,
// because for this cohort equality and identity genuinely disagree on both NaN
// and signed zero (the rare §2.2.2 carve-out).
func (f floatVal) Eq(other value.Value) bool {
	o, ok := other.(floatVal)
	if !ok {
		return false
	}
	return float32(f) == float32(o)
}

// Eq is double equality (§3.3.5.1).
func (d doubleVal) Eq(other value.Value) bool {
	o, ok := other.(doubleVal)
	if !ok {
		return false
	}
	return float64(d) == float64(o)
}

// Identical is the float identity relation (§3.3.4.1, §2.2.2), DISTINCT from
// Eq: NaN is identical to itself (every NaN is the one notANumber value) while
// +0 and −0 are NOT identical (they differ by sign bit). Enumeration and
// identity-constraint matching read this, not Eq (value/doc.go, enumMatch).
func (f floatVal) Identical(other value.Value) bool {
	o, ok := other.(floatVal)
	if !ok {
		return false
	}
	return floatingIdentical(float64(float32(f)), float64(float32(o)))
}

// Identical is the double identity relation (§3.3.5.1, §2.2.2).
func (d doubleVal) Identical(other value.Value) bool {
	o, ok := other.(doubleVal)
	if !ok {
		return false
	}
	return floatingIdentical(float64(d), float64(o))
}

// floatingIdentical is the shared identity relation: all NaNs are the single
// notANumber value, so any NaN pair is identical; otherwise identity is bit
// equality, which distinguishes +0 from −0 (differing sign bits) yet coincides
// with numeric equality on every ordinary value.
func floatingIdentical(a, b float64) bool {
	if math.IsNaN(a) && math.IsNaN(b) {
		return true
	}
	return math.Float64bits(a) == math.Float64bits(b)
}
