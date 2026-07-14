package strict

import (
	"math/big"
	"regexp"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// durationLexical is the duration lexical space (§3.3.6.2, nt-durationRep). It
// is the anchored whole-string grammar equivalent to the intersection of the
// three constraining regular expressions the spec gives — fields in the order
// Y,M,D,T(H,M,S); at least one field present (so bare "P" is rejected); and 'T'
// never final (so "PT" is rejected). Transcribed from the spec's combined
// regular expression with the legibility whitespace removed (the spec states
// that removal explicitly). Anchored so the whole literal must match; whiteSpace
// (collapse, §3.3.6.3) is a pre-lexical pipeline stage, so Parse rejects stray
// whitespace here.
var durationLexical = regexp.MustCompile(`^-?P((([0-9]+Y([0-9]+M)?([0-9]+D)?|([0-9]+M)([0-9]+D)?|([0-9]+D))(T(([0-9]+H)([0-9]+M)?([0-9]+(\.[0-9]+)?S)?|([0-9]+M)([0-9]+(\.[0-9]+)?S)?|([0-9]+(\.[0-9]+)?S)))?)|(T(([0-9]+H)([0-9]+M)?([0-9]+(\.[0-9]+)?S)?|([0-9]+M)([0-9]+(\.[0-9]+)?S)?|([0-9]+(\.[0-9]+)?S))))$`)

// durationFields extracts the six numeric fragments of a durationLexicalRep in
// order: the year and month numerals of the year-month half, then after 'T' the
// hour, minute and second numerals of the day-time half (day precedes 'T'). It
// is applied ONLY after durationLexical has accepted the literal, so its
// all-optional structure — which alone would admit "P"/"PT" — is safe: those are
// already rejected. The two 'M' fragments are disambiguated by position (months
// precede 'T', minutes follow it).
var durationFields = regexp.MustCompile(`^(-)?P(?:([0-9]+)Y)?(?:([0-9]+)M)?(?:([0-9]+)D)?(?:T(?:([0-9]+)H)?(?:([0-9]+)M)?(?:([0-9]+(?:\.[0-9]+)?)S)?)?$`)

// durationVal is an xs:duration value (§3.3.6). The spec's value space
// (§3.3.6.1) is a two-property tuple — an integer ·months· and a decimal
// ·seconds· that share one sign — NOT six independent components. Encoding the
// sign once (negative) over nonnegative magnitudes makes a mixed-sign duration
// unrepresentable (STYLE T7; the spec's sign-coupling invariant, PRINCIPLES 4).
// big.Int/big.Rat because duration lexicals allow unbounded digit runs, and
// big.Rat normalizes decimal seconds structurally (1.0 == 1.00) so Eq/Identical
// stay structural.
type durationVal struct {
	negative bool     // shared sign of both halves; false for the zero duration
	months   *big.Int // magnitude ≥ 0, = 12·years + months
	seconds  *big.Rat // magnitude ≥ 0, = 86400·days + 3600·hours + 60·minutes + seconds
}

// parseDuration maps a duration lexical to its value (·durationMap·,
// f-durationMap, §3.3.6.2/§E.2): the year-month half maps to ·months· and the
// day-time half to ·seconds·, each independently, and the single leading '-'
// negates BOTH halves together — which is exactly the sign-coupling the value
// space requires.
func parseDuration(lexical string, _ value.Context) (value.Value, error) {
	if !durationLexical.MatchString(lexical) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"duration: %q is not in the lexical space (durationLexicalRep, §3.3.6.2)", lexical)
	}
	f := durationFields.FindStringSubmatch(lexical)
	// f[1]=sign, f[2]=years, f[3]=months, f[4]=days, f[5]=hours, f[6]=minutes,
	// f[7]=seconds.
	negative := f[1] == "-"

	// ·duYearMonthFragmentMap· (f-duYMMap): 12·y + m.
	months := new(big.Int)
	addMonths(months, f[2], 12)
	addMonths(months, f[3], 1)

	// ·duDayTimeFragmentMap· (f-duDTMap): 86400·d + 3600·h + 60·mi + s.
	seconds := new(big.Rat)
	addSeconds(seconds, f[4], 86400)
	addSeconds(seconds, f[5], 3600)
	addSeconds(seconds, f[6], 60)
	addSecondFraction(seconds, f[7])

	// The zero duration carries no sign (§3.3.6.1: its ·months· and ·seconds· are
	// both 0, so the sign is moot); normalize -P0M/-PT0S/… to the positive zero so
	// Eq/Identical and Canonical read one canonical fact (STYLE D3).
	if months.Sign() == 0 && seconds.Sign() == 0 {
		negative = false
	}
	return durationVal{negative: negative, months: months, seconds: seconds}, nil
}

// addMonths adds weight·field to acc when field is a numeral (empty ⇒ absent ⇒
// 0). field matches unsignedNoDecimalPtNumeral, so SetString base 10 succeeds.
func addMonths(acc *big.Int, field string, weight int64) {
	if field == "" {
		return
	}
	n, _ := new(big.Int).SetString(field, 10)
	acc.Add(acc, n.Mul(n, big.NewInt(weight)))
}

// addSeconds adds weight·field seconds to acc for an integer numeral field.
func addSeconds(acc *big.Rat, field string, weight int64) {
	if field == "" {
		return
	}
	n, _ := new(big.Int).SetString(field, 10)
	term := new(big.Rat).SetInt(n)
	acc.Add(acc, term.Mul(term, new(big.Rat).SetInt64(weight)))
}

// addSecondFraction adds the (possibly fractional) duSecondFrag numeral to acc.
// big.Rat.SetString parses a decimal-point numeral exactly (e.g. "1.5" ⇒ 3/2).
func addSecondFraction(acc *big.Rat, field string) {
	if field == "" {
		return
	}
	s, _ := new(big.Rat).SetString(field)
	acc.Add(acc, s)
}

// canonicalDuration is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail).
func canonicalDuration(v value.Value) (string, error) {
	d, ok := v.(durationVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"duration canonical: value of type %T is not a strict duration", v)
	}
	return d.Canonical(), nil
}

// Canonical renders the canonical duration lexical (·durationCanonicalMap·,
// f-durationCanMap, §E.2): a single leading '-' when negative, then 'P', then
// the year-month half and/or the day-time half per the three branches. When
// ·months· is zero the day-time half is emitted regardless of ·seconds·, so the
// zero duration renders "PT0S" (its ·duDayTimeCanonicalFragmentMap· zero case).
func (d durationVal) Canonical() string {
	sgn := ""
	if d.negative {
		sgn = "-"
	}
	monthsZero := d.months.Sign() == 0
	secondsZero := d.seconds.Sign() == 0
	switch {
	case !monthsZero && !secondsZero:
		return sgn + "P" + yearMonthCanonicalFragment(d.months) + dayTimeCanonicalFragment(d.seconds)
	case !monthsZero: // ·seconds· is zero
		return sgn + "P" + yearMonthCanonicalFragment(d.months)
	default: // ·months· is zero
		return sgn + "P" + dayTimeCanonicalFragment(d.seconds)
	}
}

// yearMonthCanonicalFragment maps a nonnegative ·months· magnitude to a
// duYearMonthFrag (·duYearMonthCanonicalFragmentMap·, f-duYMCan): "yYmM", "yY",
// or "mM", each zero sub-component dropped. It is invoked only when ·months· ≠ 0,
// so it never yields a bare "0M".
func yearMonthCanonicalFragment(months *big.Int) string {
	y, m := new(big.Int), new(big.Int)
	y.DivMod(months, big.NewInt(12), m)
	yZero := y.Sign() == 0
	mZero := m.Sign() == 0
	switch {
	case !yZero && !mZero:
		return y.String() + "Y" + m.String() + "M"
	case !yZero:
		return y.String() + "Y"
	default:
		return m.String() + "M"
	}
}

// dayTimeCanonicalFragment maps a nonnegative ·seconds· magnitude to a
// duDayTimeFrag (·duDayTimeCanonicalFragmentMap·, f-duDTCan): decompose into
// days/hours/minutes/seconds, drop each zero sub-component, and render "T0S"
// when the whole ·seconds· value is zero.
func dayTimeCanonicalFragment(seconds *big.Rat) string {
	if seconds.Sign() == 0 {
		return "T0S"
	}
	day, rem := ratDivMod(seconds, 86400)
	hour, rem := ratDivMod(rem, 3600)
	minute, second := ratDivMod(rem, 60)
	dayFrag := ""
	if day.Sign() != 0 {
		dayFrag = day.String() + "D"
	}
	return dayFrag + timeCanonicalFragment(hour, minute, second)
}

// timeCanonicalFragment maps the normalized hour, minute and second values to a
// duTimeFrag (·duTimeCanonicalFragmentMap·, f-duTCan): 'T' then each nonzero
// sub-component, or the empty string when all three are zero — so 'T' is
// suppressed unless a time field follows it (a day may still stand alone).
func timeCanonicalFragment(hour, minute *big.Int, second *big.Rat) string {
	if hour.Sign() == 0 && minute.Sign() == 0 && second.Sign() == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("T")
	if hour.Sign() != 0 {
		b.WriteString(hour.String() + "H")
	}
	if minute.Sign() != 0 {
		b.WriteString(minute.String() + "M")
	}
	if second.Sign() != 0 {
		b.WriteString(secondCanonicalFragment(second) + "S")
	}
	return b.String()
}

// secondCanonicalFragment renders a nonnegative second sub-value as a
// duSecondFrag numeral (·duSecondCanonicalFragmentMap·, f-duSCan, without the
// trailing 'S'): a bare integer when integral (·unsignedNoDecimalPtCanonicalMap·),
// else an unsignedDecimalPtNumeral with a mandatory point
// (·unsignedDecimalPtCanonicalMap·). The value derives from a decimal lexical, so
// its expansion terminates.
func secondCanonicalFragment(second *big.Rat) string {
	if second.IsInt() {
		return second.Num().String()
	}
	return terminatingDecimal(second)
}

// ratDivMod splits a nonnegative rational r as q·w + rem with q a nonnegative
// integer and 0 ≤ rem < w — the spec's ·div·/·mod· on decimals used to peel
// days, hours and minutes off a ·seconds· magnitude.
func ratDivMod(r *big.Rat, w int64) (*big.Int, *big.Rat) {
	weight := big.NewInt(w)
	// r ≥ 0 and Denom > 0, so truncated division equals floor.
	q := new(big.Int).Quo(r.Num(), new(big.Int).Mul(r.Denom(), weight))
	rem := new(big.Rat).Sub(r, new(big.Rat).SetInt(new(big.Int).Mul(q, weight)))
	return q, rem
}

// terminatingDecimal renders a nonnegative rational with a terminating decimal
// expansion as "intPart.fracDigits" (·unsignedDecimalPtCanonicalMap·,
// f-unsDecCanFragMap): at least one integer digit, a mandatory point, and the
// exact fractional digits with no trailing zeros. Every duration seconds value
// derives from a decimal numeral, so its denominator is a product of 2s and 5s
// and the loop terminates.
func terminatingDecimal(r *big.Rat) string {
	intPart, rem := new(big.Int), new(big.Int)
	intPart.QuoRem(new(big.Int).Set(r.Num()), r.Denom(), rem)
	var frac []byte
	for rem.Sign() != 0 {
		rem.Mul(rem, bigTen)
		digit, mod := new(big.Int), new(big.Int)
		digit.QuoRem(rem, r.Denom(), mod)
		frac = append(frac, byte('0'+digit.Int64()))
		rem = mod
	}
	return intPart.String() + "." + string(frac)
}

// durationEqual is the structural (·months·, ·seconds·) equality the value space
// defines (§3.3.6.1: "two duration values are equal if and only if they are
// identical"). Magnitudes are normalized (nonnegative, decimal-scale-collapsed
// by big.Rat, zero-signless), so tuple equality is exactly value equality. Both
// Eq and Identical read it: duration has no Eq/Identical divergence (unlike
// float's NaN/±0), but Identical is implemented so enumeration matching
// (cvc-enumeration-valid, §4.3.5.4) never falls back to the partial order's Cmp.
func durationEqual(a, b durationVal) bool {
	return a.negative == b.negative &&
		a.months.Cmp(b.months) == 0 &&
		a.seconds.Cmp(b.seconds) == 0
}

// Eq is duration equality (§3.3.6.1), structural over the (·months·, ·seconds·)
// tuple; a non-duration argument is unequal.
func (d durationVal) Eq(other value.Value) bool {
	o, ok := other.(durationVal)
	if !ok {
		return false
	}
	return durationEqual(d, o)
}

// Identical is the duration identity relation (§2.2.2), the same structural
// comparison as Eq; a non-duration argument is not identical.
func (d durationVal) Identical(other value.Value) bool {
	o, ok := other.(durationVal)
	if !ok {
		return false
	}
	return durationEqual(d, o)
}

// durationRef is one of the four fixed reference dateTimes the duration order is
// defined against (§3.3.6.1). Each has day 1 and midnight UTC, so only (year,
// month) is needed: ·dateTimePlusDuration· pins the day to min(1, daysInMonth) =
// 1, and the shared 00:00:00 offset cancels in every pairwise comparison.
type durationRef struct{ year, month int }

// durationRefs are the four reference dateTimes from §3.3.6.1, whose disagreeing
// month lengths and leap years are what make the order partial.
var durationRefs = [...]durationRef{{1696, 9}, {1697, 2}, {1903, 3}, {1903, 7}}

// Cmp is the PARTIAL order on duration (§3.3.6.1): add each duration to the four
// reference dateTimes and compare the resulting instants; when all four agree
// the pair is ordered that way, otherwise it is Incomparable. A non-duration
// argument is Incomparable (rf-ordered). This yields the spec's canonical
// incomparable pairs (P1M vs P30D, P1M vs P31D). Incomparable is a definite
// verdict here, not a fail-open: a bound facet against an incomparable value
// genuinely fails (§2.2.3), it is not an unsupported construct.
func (d durationVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(durationVal)
	if !ok {
		return value.Incomparable
	}
	var agreed value.Ordering
	for i, ref := range durationRefs {
		got := ratOrdering(d.instant(ref), o.instant(ref))
		if i == 0 {
			agreed = got
			continue
		}
		if got != agreed {
			return value.Incomparable
		}
	}
	return agreed
}

// instant returns d added to reference ref as a proleptic-Gregorian instant,
// measured in exact seconds from the 1970-01-01 epoch (·dateTimePlusDuration·,
// vp-dt-dateTimePlusDuration): add ·months· to the year-month (carrying
// overflow), pin the day (a no-op at day 1), convert to a day ordinal, then add
// ·seconds· as a flat offset. big.Rat throughout, so sub-nanosecond fractional
// seconds and unbounded magnitudes both compare exactly — unlike time.Duration.
func (d durationVal) instant(ref durationRef) *big.Rat {
	monthIndex := big.NewInt(int64(ref.year)*12 + int64(ref.month-1))
	monthIndex.Add(monthIndex, d.signedMonths())
	year, monthZeroBased := new(big.Int), new(big.Int)
	year.DivMod(monthIndex, big.NewInt(12), monthZeroBased) // Euclidean: [0,11]
	ordinal := daysFromCivil(year, int(monthZeroBased.Int64())+1, 1)
	secs := new(big.Rat).SetInt(new(big.Int).Mul(ordinal, big.NewInt(86400)))
	secs.Add(secs, d.signedSeconds())
	return secs
}

// signedMonths applies the shared sign to the ·months· magnitude.
func (d durationVal) signedMonths() *big.Int {
	if d.negative {
		return new(big.Int).Neg(d.months)
	}
	return new(big.Int).Set(d.months)
}

// signedSeconds applies the shared sign to the ·seconds· magnitude.
func (d durationVal) signedSeconds() *big.Rat {
	if d.negative {
		return new(big.Rat).Neg(d.seconds)
	}
	return new(big.Rat).Set(d.seconds)
}

// daysFromCivil returns the proleptic-Gregorian day number of (year, month, day)
// relative to 1970-01-01, exact for any year magnitude (Howard Hinnant's
// days_from_civil). month ∈ [1,12] and day ∈ [1,31] stay in int64; only year is
// unbounded, so it is a big.Int. The single negative-year floor adjustment keeps
// year-of-era in [0,399] so the remaining divisions are on nonnegative operands.
func daysFromCivil(year *big.Int, month, day int) *big.Int {
	y := new(big.Int).Set(year)
	if month <= 2 {
		y.Sub(y, big.NewInt(1))
	}
	era, rem := new(big.Int), new(big.Int)
	era.DivMod(y, big.NewInt(400), rem) // Euclidean floor for the positive divisor
	yoe := rem.Int64()                  // year of era ∈ [0,399]
	mp := int64(month)
	if month > 2 {
		mp -= 3
	} else {
		mp += 9
	}
	doy := (153*mp+2)/5 + int64(day) - 1   // day of year ∈ [0,365]
	doe := yoe*365 + yoe/4 - yoe/100 + doy // day of era ∈ [0,146096]
	days := new(big.Int).Mul(era, big.NewInt(146097))
	return days.Add(days, big.NewInt(doe-719468))
}

// ratOrdering maps a big.Rat comparison to a value.Ordering (never Incomparable:
// two rationals always relate — the partial order lives in Cmp's four-way vote).
func ratOrdering(a, b *big.Rat) value.Ordering {
	switch a.Cmp(b) {
	case -1:
		return value.Less
	case 1:
		return value.Greater
	}
	return value.Equal
}
