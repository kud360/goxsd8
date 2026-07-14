package strict

import (
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// dateTimeLexical is the dateTime lexical space (§3.3.7.2, nt-dateTimeRep). It is
// the anchored whole-string grammar equivalent to the regular expression the spec
// gives verbatim ("The dateTimeLexicalRep production is equivalent to this regular
// expression once whitespace is removed", §3.3.7.2), transcribed with the
// legibility whitespace removed as the spec instructs and with capture groups
// added to extract each fragment. The spec is explicit that this regex alone does
// NOT enforce the day-of-month constraint (con-dateTime-day/con-dateTime-dayValue);
// parseDateTime checks that separately. whiteSpace (collapse, §3.3.7.3) is a
// pre-lexical pipeline stage, so Parse rejects stray surrounding whitespace here.
//
// Capture groups: 1 year (with optional leading '-'), 2 month, 3 day, 4 hour,
// 5 minute, 6 second-integer, 7 second-fraction (with the leading '.'), 8 the
// whole endOfDayFrag (non-empty iff "24:00:00[.0+]" was matched), 9 timezoneFrag.
var dateTimeLexical = regexp.MustCompile(
	`^(-?(?:[1-9][0-9]{3,}|0[0-9]{3}))` + // yearFrag [56]
		`-(0[1-9]|1[0-2])` + // monthFrag [57]
		`-(0[1-9]|[12][0-9]|3[01])` + // dayFrag [58]
		`T(?:([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])(\.[0-9]+)?|(24:00:00(?:\.0+)?))` + // hourFrag:minuteFrag:secondFrag | endOfDayFrag [59-62]
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`) // timezoneFrag [63]

// dateTimeVal is an xs:dateTime value (§3.3.7). The spec's value space
// (§3.3.7.1) is the seven-property date/timeSevenPropertyModel (§D.2.1); dateTime
// permits no property except ·timezoneOffset· to be absent, so year/month/day/
// hour/minute/second are always present. ·second· is ONE decimal number (a
// *big.Rat, mirroring duration's ·seconds·), not a separate int+fraction pair
// (PRINCIPLES 4/5). ·year· is unbounded (yearFrag admits arbitrary-length digit
// runs), so it is a *big.Int and reuses duration.go's daysFromCivil directly. The
// endOfDayFrag hour-24 case is normalized/carried at parse time, so ·hour· is
// NEVER 24 in a stored value, and no illegal day/month/year combination is ever
// stored (con-dateTime-dayValue is checked at construction).
type dateTimeVal struct {
	year         *big.Int // unbounded; 0 = 1 BCE, −1 = 2 BCE (XSD 1.1 permits year 0)
	month, day   int      // 1-12, 1-31 (day valid for month/year per con-dateTime-dayValue)
	hour, minute int      // 0-23, 0-59 (hour never 24: endOfDayFrag is carried at parse time)
	second       *big.Rat // 0 ≤ second < 60, one decimal value
	tzOffset     *int     // minutes, −840..840; nil == absent (no separate hasTimezone flag)
}

// parseDateTime maps a dateTime lexical to its value (·dateTimeLexicalMap·,
// vp-dateTimeLexRep, §E.3.5): each fragment maps to its property, then
// endOfDayFrag (hour 24) is normalized and carried into the next calendar day
// (·newDateTime· via ·normalizeSecond·, p-setDTFromRaw/§E.3.1), so hour 24 never
// survives as a stored value. The regex accepts day up to 31 for any month, so
// the day-of-month constraint (con-dateTime-day/con-dateTime-dayValue, §3.3.7.1)
// is checked here on the as-written day; both a regex mismatch and a day-of-month
// violation are "not in the lexical space" (§4.1.4) and map to cvc-datatype-valid.
func parseDateTime(lexical string, _ value.Context) (value.Value, error) {
	m := dateTimeLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"dateTime: %q is not in the lexical space (dateTimeLexicalRep, §3.3.7.2)", lexical)
	}
	year, _ := new(big.Int).SetString(m[1], 10) // regex guarantees a valid integer numeral
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])

	// con-dateTime-day/con-dateTime-dayValue (§3.3.7.1): the regex admits day 31
	// for every month, so reject a day beyond the month's length (leap year aware)
	// as outside the lexical space.
	if day > daysInMonth(year, month) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"dateTime: %q has day %d out of range for month %d of year %s (con-dateTime-dayValue, §3.3.7.1)",
			lexical, day, month, year)
	}

	var tz *int
	if m[9] != "" {
		off := timezoneOffset(m[9])
		tz = &off
	}

	if m[8] != "" { // endOfDayFrag: hour 24, carry into the next calendar day
		year, month, day = nextCalendarDay(year, month, day)
		return dateTimeVal{year: year, month: month, day: day, hour: 0, minute: 0, second: new(big.Rat), tzOffset: tz}, nil
	}

	hour, _ := strconv.Atoi(m[4])
	minute, _ := strconv.Atoi(m[5])
	second, _ := new(big.Rat).SetString(m[6] + m[7]) // "SS" + ".fff" (or ""); regex guarantees a decimal
	return dateTimeVal{year: year, month: month, day: day, hour: hour, minute: minute, second: second, tzOffset: tz}, nil
}

// timezoneOffset maps a timezoneFrag to its ·timezoneOffset· minutes
// (·timezoneFragValue·, f-dt-tzMap): 'Z' is 0, otherwise the signed hh:mm. The
// grammar (production [63]) already bounds the result to [−840, 840], so no
// range check is needed here.
func timezoneOffset(frag string) int {
	if frag == "Z" {
		return 0
	}
	hh, _ := strconv.Atoi(frag[1:3])
	mm, _ := strconv.Atoi(frag[4:6])
	off := hh*60 + mm
	if frag[0] == '-' {
		return -off
	}
	return off
}

// nextCalendarDay returns the calendar day after (year, month, day), rolling the
// month and year forward when the day overflows the month length — the carry the
// endOfDayFrag (midnight of "the first moment of the next day", §3.3.7.2) needs.
// day is already a valid day-of-month, so the incremented day exceeds the month
// length only when it was the last day.
func nextCalendarDay(year *big.Int, month, day int) (*big.Int, int, int) {
	if day < daysInMonth(year, month) {
		return year, month, day + 1
	}
	if month < 12 {
		return year, month + 1, 1
	}
	return new(big.Int).Add(year, big.NewInt(1)), 1, 1
}

// daysInMonth returns the length of month in year (con-dateTime-dayValue,
// §3.3.7.1): 30 for April/June/September/November, 28 or 29 for February by the
// proleptic-Gregorian leap rule, 31 otherwise. year is unbounded, so divisibility
// is tested on the *big.Int.
func daysInMonth(year *big.Int, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	}
	if isLeapYear(year) {
		return 29
	}
	return 28
}

// isLeapYear applies the proleptic-Gregorian leap rule (con-dateTime-dayValue,
// §3.3.7.1): divisible by 4, except centuries not divisible by 400. Divisibility
// is sign-independent (rem == 0), so it holds for the negative years XSD 1.1
// permits.
func isLeapYear(year *big.Int) bool {
	divisibleBy := func(n int64) bool {
		return new(big.Int).Rem(year, big.NewInt(n)).Sign() == 0
	}
	if !divisibleBy(4) {
		return false
	}
	if !divisibleBy(100) {
		return true
	}
	return divisibleBy(400)
}

// canonicalDateTime is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail), mirroring
// canonicalDuration.
func canonicalDateTime(v value.Value) (string, error) {
	d, ok := v.(dateTimeVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"dateTime canonical: value of type %T is not a strict dateTime", v)
	}
	return d.Canonical(), nil
}

// Canonical renders the canonical dateTime lexical (·dateTimeCanonicalMap·,
// vp-dateTimeCanRep, §E.3.6): year-month-day 'T' hour:minute:second, with the
// timezone fragment appended iff ·timezoneOffset· is present. Because the stored
// value already normalizes hour 24, the endOfDayFrag input "…T24:00:00" renders
// as the carried next-day "…T00:00:00".
func (d dateTimeVal) Canonical() string {
	var b strings.Builder
	b.WriteString(yearCanonicalFragment(d.year))
	fmt.Fprintf(&b, "-%02d-%02dT%02d:%02d:", d.month, d.day, d.hour, d.minute)
	b.WriteString(dateTimeSecondFragment(d.second))
	if d.tzOffset != nil {
		b.WriteString(timezoneCanonicalFragment(*d.tzOffset))
	}
	return b.String()
}

// yearCanonicalFragment maps a ·year· to a yearFrag (·yearCanonicalFragmentMap·,
// f-yrCanFragMap, §E.3.6): a plain signed numeral when |year| > 9999
// (·noDecimalPtCanonicalMap·), else an always-four-digit numeral with the sign
// preserved (·fourDigitCanonicalFragmentMap·), e.g. year 5 → "0005", −1 → "-0001".
func yearCanonicalFragment(year *big.Int) string {
	abs := new(big.Int).Abs(year)
	if abs.Cmp(big.NewInt(9999)) > 0 {
		return year.String()
	}
	if year.Sign() < 0 {
		return fmt.Sprintf("-%04d", abs.Int64())
	}
	return fmt.Sprintf("%04d", abs.Int64())
}

// secondCanonicalFragment maps a ·second· decimal to a secondFrag
// (·secondCanonicalFragmentMap·, f-seCanFragMap, §E.3.6): an always-two-digit
// integer part, followed by '.' and the exact fractional digits (with no trailing
// zeros, ·fractionDigitsCanonicalFragmentMap·) when the value is not integral.
// ·second· is in [0, 60) and derives from a decimal numeral, so the integer part
// fits an int64 and the fractional expansion terminates.
func dateTimeSecondFragment(second *big.Rat) string {
	intPart, rem := new(big.Int), new(big.Int)
	intPart.QuoRem(second.Num(), second.Denom(), rem)
	whole := fmt.Sprintf("%02d", intPart.Int64())
	if rem.Sign() == 0 {
		return whole
	}
	den := second.Denom()
	var frac []byte
	for rem.Sign() != 0 {
		rem.Mul(rem, bigTen)
		digit, mod := new(big.Int), new(big.Int)
		digit.QuoRem(rem, den, mod)
		frac = append(frac, byte('0'+digit.Int64()))
		rem = mod
	}
	return whole + "." + string(frac)
}

// timezoneCanonicalFragment maps a ·timezoneOffset· to a timezoneFrag
// (·timezoneCanonicalFragmentMap·, f-tzCanFragMap, §E.3.6): 'Z' for offset 0, else
// the signed hh:mm.
func timezoneCanonicalFragment(offset int) string {
	if offset == 0 {
		return "Z"
	}
	if offset < 0 {
		return fmt.Sprintf("-%02d:%02d", -offset/60, -offset%60)
	}
	return fmt.Sprintf("+%02d:%02d", offset/60, offset%60)
}

// Eq is dateTime equality (§D.2.1), derived from the order: two values are equal
// iff they compare Equal on the ·timeOnTimeline·. A non-dateTime argument (Cmp
// yields Incomparable) is unequal. Eq and Identical genuinely diverge here: a
// timezone-shifted pair denoting the same instant is Eq but not Identical.
func (d dateTimeVal) Eq(other value.Value) bool {
	return d.Cmp(other) == value.Equal
}

// Identical is the dateTime identity relation (§2.2.2): a STRUCTURAL comparison of
// every stored property including the exact ·timezoneOffset·. So
// 2002-10-10T12:00:00−05:00 and 2002-10-10T17:00:00Z are Eq (same instant) but NOT
// identical (different stored offset). A non-dateTime argument is not identical.
func (d dateTimeVal) Identical(other value.Value) bool {
	o, ok := other.(dateTimeVal)
	if !ok {
		return false
	}
	return d.year.Cmp(o.year) == 0 &&
		d.month == o.month && d.day == o.day &&
		d.hour == o.hour && d.minute == o.minute &&
		d.second.Cmp(o.second) == 0 &&
		tzOffsetEqual(d.tzOffset, o.tzOffset)
}

// tzOffsetEqual is exact ·timezoneOffset· equality including absence: two absent
// offsets are equal, an absent and a present offset are not.
func tzOffsetEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// Cmp is the PARTIAL order on dateTime (§D.2.1), computed over ·timeOnTimeline·
// (vp-dt-timeOnTimeline, §E.3.4). When both operands carry a timezone (or both
// lack one) their instants compare directly. When exactly one lacks a timezone,
// the spec imputes the maximum (+840) and minimum (−840) offsets to the absent
// operand: if both imputations yield the same strict inequality that is the
// order, otherwise the pair is Incomparable. A non-dateTime argument is
// Incomparable (rf-ordered). Incomparable is a definite verdict here, not a
// fail-open.
func (d dateTimeVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(dateTimeVal)
	if !ok {
		return value.Incomparable
	}
	dHas := d.tzOffset != nil
	oHas := o.tzOffset != nil
	if dHas == oHas {
		return ratOrdering(d.instant(), o.instant())
	}
	if !dHas { // d absent, o timezoned: order the absent operand against o
		return imputedOrdering(d.instant(), o.instant())
	}
	// d timezoned, o absent: order o against d, then flip to d-relative.
	return flipOrdering(imputedOrdering(o.instant(), d.instant()))
}

// instant is the value's ·timeOnTimeline· (vp-dt-timeOnTimeline, §E.3.4) as exact
// seconds: the proleptic-Gregorian day ordinal (daysFromCivil, reused from
// duration.go per PRINCIPLES 4) times 86400, plus hour/minute/second, minus
// ·timezoneOffset· seconds when present (the spec subtracts the offset from
// minutes). An absent offset leaves the raw local instant, which is exactly the
// quantity the ±840 imputations then shift.
func (d dateTimeVal) instant() *big.Rat {
	ordinal := daysFromCivil(d.year, d.month, d.day)
	secs := new(big.Rat).SetInt(new(big.Int).Mul(ordinal, big.NewInt(86400)))
	secs.Add(secs, new(big.Rat).SetInt64(int64(d.hour)*3600+int64(d.minute)*60))
	secs.Add(secs, d.second)
	if d.tzOffset != nil {
		secs.Sub(secs, new(big.Rat).SetInt64(int64(*d.tzOffset)*60))
	}
	return secs
}

// tzImputationSeconds is 840 minutes (14 hours) in seconds — the maximum
// ·timezoneOffset· magnitude the incomparability rule imputes to a timezone-less
// operand.
var tzImputationSeconds = big.NewRat(840*60, 1)

// imputedOrdering orders a timezone-less instant (raw) against a timezoned
// instant per §D.2.1: impute offset +840 (subtract the imputation) and −840 (add
// it); if both comparisons agree that is the order, otherwise Incomparable.
func imputedOrdering(raw, timezoned *big.Rat) value.Ordering {
	plus := new(big.Rat).Sub(raw, tzImputationSeconds)  // offset +840
	minus := new(big.Rat).Add(raw, tzImputationSeconds) // offset −840
	hi := ratOrdering(plus, timezoned)
	lo := ratOrdering(minus, timezoned)
	if hi == lo {
		return hi
	}
	return value.Incomparable
}

// flipOrdering reverses an ordering so an a-relative-to-b verdict becomes
// b-relative-to-a. Equal and Incomparable are symmetric and unchanged.
func flipOrdering(o value.Ordering) value.Ordering {
	switch o {
	case value.Less:
		return value.Greater
	case value.Greater:
		return value.Less
	default: // Equal and Incomparable are symmetric
		return o
	}
}

// HasTimezone reports whether the value carries an explicit ·timezoneOffset·
// (value.TimezoneAware; the explicitTimezone facet, §4.3.15, reads it).
func (d dateTimeVal) HasTimezone() bool {
	return d.tzOffset != nil
}
