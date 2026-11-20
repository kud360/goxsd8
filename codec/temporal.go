package codec

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

// convertTemporal returns the value of a's bare type, a date, time or duration
// datatype struct, that form denotes, or the error for a form that is not of
// a's lexical grammar (Datatypes §3.3.6.2, §3.3.7.2–§3.3.14.2, §3.4.26.2–
// §3.4.28.2, the fragments of §D.2.2) or that denotes no value of a's value
// space: a property outside its §D.2.1 range, as monthWithin and the checks
// beside it state the ranges, so an hour of 24, which the lexical mapping
// normalizes away (§E.3.1), included, dateTimeStamp's absent timezoneOffset
// (§3.4.28), and a yearMonthDuration's seconds or a dayTimeDuration's months
// other than 0 (§3.4.26, §3.4.27). Each property is the local value the form
// shows, never converted to UTC (§D.2.1), and a duration's Months and Seconds
// both take the form's one sign (§3.3.6.1). Any other a panics, since convert
// delegates to it every atomic convert does not name.
func convertTemporal(a atomic, form string) (reflect.Value, error) {
	c := &scanner{s: form}
	var v reflect.Value
	var err error
	switch a {
	case atomicDateTime, atomicDateTimeStamp:
		v, err = c.dateTime(a == atomicDateTimeStamp)
	case atomicTime:
		v, err = c.time()
	case atomicDate:
		v, err = c.date()
	case atomicGYearMonth:
		v, err = c.gYearMonth()
	case atomicGYear:
		v, err = c.gYear()
	case atomicGMonthDay:
		v, err = c.gMonthDay()
	case atomicGDay:
		v, err = c.gDay()
	case atomicGMonth:
		v, err = c.gMonth()
	case atomicDuration, atomicDayTimeDuration, atomicYearMonthDuration:
		v, err = c.duration(a)
	default:
		// convert delegates every atomic it does not convert itself; of those,
		// atomicString's value is the normalized lexical itself, and
		// atomicQName's expandedName's.
		panic(fmt.Sprintf("codec: atomic %s has no conversion", a.bare()))
	}
	if err != nil {
		return reflect.Value{}, err
	}
	if !c.done() {
		return reflect.Value{}, fmt.Errorf("%q follows the value at byte %d", c.s[c.i:], c.i)
	}
	return v, nil
}

// scanner reads a date, time or duration form s from byte offset i.
type scanner struct {
	s string
	i int
}

// done reports whether c has read the whole form.
func (c *scanner) done() bool { return c.i == len(c.s) }

// lit consumes b where it is the next byte, and reports whether it was.
func (c *scanner) lit(b byte) bool {
	if c.i < len(c.s) && c.s[c.i] == b {
		c.i++
		return true
	}
	return false
}

// expect consumes b, or returns the error naming what is there instead.
func (c *scanner) expect(b byte) error {
	if c.lit(b) {
		return nil
	}
	if c.done() {
		return fmt.Errorf("the form ends where %q is expected", b)
	}
	return fmt.Errorf("%q at byte %d where %q is expected", c.s[c.i], c.i, b)
}

// digits consumes the longest run of ASCII digits and returns it.
func (c *scanner) digits() string {
	start := c.i
	for c.i < len(c.s) && c.s[c.i] >= '0' && c.s[c.i] <= '9' {
		c.i++
	}
	return c.s[start:c.i]
}

// two consumes exactly two digits and returns their value as the property
// name, read by strconv at uint8's width; the caller checks its range.
func (c *scanner) two(name string) (uint8, error) {
	start := c.i
	d := c.digits()
	if len(d) != 2 {
		return 0, fmt.Errorf("the %s at byte %d is %q, not two digits", name, start, d)
	}
	n, err := strconv.ParseUint(d, 10, 8)
	if err != nil {
		return 0, err
	}
	return uint8(n), nil
}

// within returns nil where n, the property name, lies between lo and hi
// inclusive, and otherwise the error naming n and the bounds (§D.2.1).
func within(name string, n, lo, hi uint64) error {
	if n < lo || n > hi {
		return fmt.Errorf("the %s %d is outside %d to %d (§D.2.1)", name, n, lo, hi)
	}
	return nil
}

// monthWithin, dayWithin, hourWithin, minuteWithin, secondWithin and
// offsetWithin are the one check of each §D.2.1 bound of the seven-property
// model, made by the scanner on a form it reads and by (atomic).form on a
// datatype struct value it writes: month 1 to 12; day 1 to
// ·daysInMonth·(year, m), or to 31 where the month is absent, m 0 (gDay,
// §3.3.13.1); hour 0 to 23; minute 0 to 59; second 0 to 60, 60 excluded; and
// timezoneOffset −840 to 840. Each returns nil for a property in range, and
// otherwise the error naming it.
func monthWithin(m uint8) error  { return within("month", uint64(m), 1, 12) }
func hourWithin(h uint8) error   { return within("hour", uint64(h), 0, 23) }
func minuteWithin(m uint8) error { return within("minute", uint64(m), 0, 59) }

func dayWithin(year *big.Int, m, d uint8) error {
	return within("day", uint64(d), 1, uint64(daysInMonth(year, m)))
}

func secondWithin(s *big.Rat) error {
	if s.Sign() < 0 || s.Cmp(sixty) >= 0 {
		return fmt.Errorf("the second %s is outside 0 to 60, 60 excluded (§D.2.1)", s.RatString())
	}
	return nil
}

func offsetWithin(off int16) error {
	if off < -840 || off > 840 {
		return fmt.Errorf("the timezoneOffset %d is outside -840 to 840 minutes (§D.2.1)", off)
	}
	return nil
}

// sixty is the bound secondWithin compares with; nothing writes it.
var sixty = big.NewRat(60, 1)

// year consumes a yearFrag (§D.2.2): an optional '-' and four or more digits,
// a leading zero only in exactly four.
func (c *scanner) year() (big.Int, error) {
	start := c.i
	c.lit('-')
	d := c.digits()
	if len(d) < 4 || (len(d) > 4 && d[0] == '0') {
		return big.Int{}, fmt.Errorf("the year at byte %d is %q, no yearFrag", start, c.s[start:c.i])
	}
	var y big.Int
	if _, ok := y.SetString(c.s[start:c.i], 10); !ok {
		return big.Int{}, fmt.Errorf("big.Int does not read the year %q", c.s[start:c.i])
	}
	return y, nil
}

// second consumes a secondFrag (§D.2.2): two digits and an optional '.' with
// one or more digits, a value secondWithin holds.
func (c *scanner) second() (big.Rat, error) {
	start := c.i
	if len(c.digits()) != 2 {
		return big.Rat{}, fmt.Errorf("the second at byte %d is not two digits", start)
	}
	if c.lit('.') && c.digits() == "" {
		return big.Rat{}, fmt.Errorf("the second at byte %d has no digit after its '.'", start)
	}
	var s big.Rat
	if _, ok := s.SetString(c.s[start:c.i]); !ok {
		return big.Rat{}, fmt.Errorf("big.Rat does not read the second %q", c.s[start:c.i])
	}
	return s, secondWithin(&s)
}

// timezone consumes an optional timezoneFrag (§D.2.2) ending the form and
// returns its timezoneOffset in minutes, nil where the form ends without one:
// 'Z' is 0, and a sign, two-digit hours of at most 14 and two-digit minutes
// are an offset offsetWithin holds.
func (c *scanner) timezone() (*int16, error) {
	if c.done() {
		return nil, nil
	}
	if c.lit('Z') {
		return new(int16), nil
	}
	sign := int16(1)
	switch {
	case c.lit('-'):
		sign = -1
	case c.lit('+'):
	default:
		return nil, fmt.Errorf("%q at byte %d is no timezoneFrag", c.s[c.i], c.i)
	}
	h, err := c.two("timezone hour")
	if err != nil {
		return nil, err
	}
	if err := within("timezone hour", uint64(h), 0, 14); err != nil {
		return nil, err
	}
	if err := c.expect(':'); err != nil {
		return nil, err
	}
	m, err := c.two("timezone minute")
	if err != nil {
		return nil, err
	}
	if err := within("timezone minute", uint64(m), 0, 59); err != nil {
		return nil, err
	}
	off := sign * (int16(h)*60 + int16(m))
	return &off, offsetWithin(off)
}

// yearMonthDay consumes year '-' month '-' day.
func (c *scanner) yearMonthDay() (big.Int, uint8, uint8, error) {
	y, err := c.year()
	if err != nil {
		return big.Int{}, 0, 0, err
	}
	m, d, err := c.monthDay(&y)
	return y, m, d, err
}

// monthDay consumes '-' month '-' day and returns the month and the day, the
// day at most ·daysInMonth· of year, nil for an absent one.
func (c *scanner) monthDay(year *big.Int) (uint8, uint8, error) {
	if err := c.expect('-'); err != nil {
		return 0, 0, err
	}
	m, err := c.two("month")
	if err != nil {
		return 0, 0, err
	}
	if err := monthWithin(m); err != nil {
		return 0, 0, err
	}
	if err := c.expect('-'); err != nil {
		return 0, 0, err
	}
	d, err := c.two("day")
	if err != nil {
		return 0, 0, err
	}
	return m, d, dayWithin(year, m, d)
}

// clock consumes hour ':' minute ':' second, the hour at most 23.
func (c *scanner) clock() (uint8, uint8, big.Rat, error) {
	h, err := c.two("hour")
	if err != nil {
		return 0, 0, big.Rat{}, err
	}
	if err := hourWithin(h); err != nil {
		return 0, 0, big.Rat{}, err
	}
	if err := c.expect(':'); err != nil {
		return 0, 0, big.Rat{}, err
	}
	m, err := c.two("minute")
	if err != nil {
		return 0, 0, big.Rat{}, err
	}
	if err := minuteWithin(m); err != nil {
		return 0, 0, big.Rat{}, err
	}
	if err := c.expect(':'); err != nil {
		return 0, 0, big.Rat{}, err
	}
	s, err := c.second()
	return h, m, s, err
}

// dateTime consumes a dateTime form (§3.3.7.2) and returns its value, of
// the atomic's bare type, or where stamp is true a dateTimeStamp's, whose
// timezoneOffset is required (§3.4.28).
func (c *scanner) dateTime(stamp bool) (reflect.Value, error) {
	y, mo, d, err := c.yearMonthDay()
	if err != nil {
		return reflect.Value{}, err
	}
	if err := c.expect('T'); err != nil {
		return reflect.Value{}, err
	}
	h, mi, s, err := c.clock()
	if err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	if err != nil {
		return reflect.Value{}, err
	}
	if !stamp {
		return atomicDateTime.build(prop{"Year", y}, prop{"Month", mo}, prop{"Day", d}, prop{"Hour", h}, prop{"Minute", mi}, prop{"Second", s}, prop{"TimezoneOffset", tz}), nil
	}
	if tz == nil {
		return reflect.Value{}, errors.New("a dateTimeStamp's timezoneOffset is absent (§3.4.28)")
	}
	return atomicDateTimeStamp.build(prop{"Year", y}, prop{"Month", mo}, prop{"Day", d}, prop{"Hour", h}, prop{"Minute", mi}, prop{"Second", s}, prop{"TimezoneOffset", *tz}), nil
}

// time consumes a time form (§3.3.8.2) and returns its value, of the atomic's bare type.
func (c *scanner) time() (reflect.Value, error) {
	h, mi, s, err := c.clock()
	if err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	return atomicTime.build(prop{"Hour", h}, prop{"Minute", mi}, prop{"Second", s}, prop{"TimezoneOffset", tz}), err
}

// date consumes a date form (§3.3.9.2) and returns its value, of the atomic's bare type.
func (c *scanner) date() (reflect.Value, error) {
	y, mo, d, err := c.yearMonthDay()
	if err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	return atomicDate.build(prop{"Year", y}, prop{"Month", mo}, prop{"Day", d}, prop{"TimezoneOffset", tz}), err
}

// gYearMonth consumes a gYearMonth form (§3.3.10.2) and returns its value, of the atomic's bare type.
func (c *scanner) gYearMonth() (reflect.Value, error) {
	y, err := c.year()
	if err != nil {
		return reflect.Value{}, err
	}
	if err := c.expect('-'); err != nil {
		return reflect.Value{}, err
	}
	mo, err := c.two("month")
	if err != nil {
		return reflect.Value{}, err
	}
	if err := monthWithin(mo); err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	return atomicGYearMonth.build(prop{"Year", y}, prop{"Month", mo}, prop{"TimezoneOffset", tz}), err
}

// gYear consumes a gYear form (§3.3.11.2) and returns its value, of the atomic's bare type.
func (c *scanner) gYear() (reflect.Value, error) {
	y, err := c.year()
	if err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	return atomicGYear.build(prop{"Year", y}, prop{"TimezoneOffset", tz}), err
}

// gMonthDay consumes a gMonthDay form (§3.3.12.2), whose day is at most
// ·daysInMonth· of an absent year, and returns its value, of the atomic's bare type.
func (c *scanner) gMonthDay() (reflect.Value, error) {
	if err := c.expect('-'); err != nil {
		return reflect.Value{}, err
	}
	mo, d, err := c.monthDay(nil)
	if err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	return atomicGMonthDay.build(prop{"Month", mo}, prop{"Day", d}, prop{"TimezoneOffset", tz}), err
}

// gDay consumes a gDay form (§3.3.13.2) and returns its value, of the atomic's bare type.
func (c *scanner) gDay() (reflect.Value, error) {
	for range 3 {
		if err := c.expect('-'); err != nil {
			return reflect.Value{}, err
		}
	}
	d, err := c.two("day")
	if err != nil {
		return reflect.Value{}, err
	}
	if err := dayWithin(nil, 0, d); err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	return atomicGDay.build(prop{"Day", d}, prop{"TimezoneOffset", tz}), err
}

// gMonth consumes a gMonth form (§3.3.14.2) and returns its value, of the atomic's bare type.
func (c *scanner) gMonth() (reflect.Value, error) {
	for range 2 {
		if err := c.expect('-'); err != nil {
			return reflect.Value{}, err
		}
	}
	mo, err := c.two("month")
	if err != nil {
		return reflect.Value{}, err
	}
	if err := monthWithin(mo); err != nil {
		return reflect.Value{}, err
	}
	tz, err := c.timezone()
	return atomicGMonth.build(prop{"Month", mo}, prop{"TimezoneOffset", tz}), err
}

// durationPart is one designator of a durationLexicalRep (§3.3.6.2) and what
// its unit weighs in ·months· and in ·seconds· (§E.2 ·durationMap·).
type durationPart struct {
	designator byte
	months     int64
	seconds    int64
}

// The designators of a durationLexicalRep in order, before its 'T' and after
// it.
var (
	dateParts = []durationPart{{'Y', 12, 0}, {'M', 1, 0}, {'D', 0, 86400}}
	timeParts = []durationPart{{'H', 0, 3600}, {'M', 0, 60}, {'S', 0, 1}}
)

// duration consumes a durationLexicalRep (§3.3.6.2) and returns the value
// of the bare type of a, duration, dayTimeDuration or yearMonthDuration:
// ·months· and ·seconds· as ·durationMap· sums them (§E.2), both negated by
// a leading '-'.
func (c *scanner) duration(a atomic) (reflect.Value, error) {
	negative := c.lit('-')
	if err := c.expect('P'); err != nil {
		return reflect.Value{}, err
	}
	var months big.Int
	var seconds big.Rat
	n, err := c.durationHalf(dateParts, &months, &seconds)
	if err != nil {
		return reflect.Value{}, err
	}
	if c.lit('T') {
		t, err := c.durationHalf(timeParts, &months, &seconds)
		if err != nil {
			return reflect.Value{}, err
		}
		if t == 0 {
			return reflect.Value{}, errors.New("a duration's 'T' is followed by no hour, minute or second")
		}
		n += t
	}
	if n == 0 {
		return reflect.Value{}, errors.New("a duration has no year, month, day, hour, minute or second")
	}
	if negative {
		months.Neg(&months)
		seconds.Neg(&seconds)
	}
	switch a {
	case atomicDayTimeDuration:
		if months.Sign() != 0 {
			return reflect.Value{}, errors.New("a dayTimeDuration's months are not 0 (§3.4.27)")
		}
		return atomicDayTimeDuration.build(prop{"Seconds", seconds}), nil
	case atomicYearMonthDuration:
		if seconds.Sign() != 0 {
			return reflect.Value{}, errors.New("a yearMonthDuration's seconds are not 0 (§3.4.26)")
		}
		return atomicYearMonthDuration.build(prop{"Months", months}), nil
	default:
		return atomicDuration.build(prop{"Months", months}, prop{"Seconds", seconds}), nil
	}
}

// durationHalf consumes the components of one half of a durationLexicalRep up
// to its 'T' or its end, each a numeral and one of parts' designators in
// parts' order, adds each one's weight to months and seconds, and returns how
// many it read. Only a second carries a fractional part (duSecondFrag).
func (c *scanner) durationHalf(parts []durationPart, months *big.Int, seconds *big.Rat) (int, error) {
	n := 0
	for !c.done() && c.s[c.i] != 'T' {
		start := c.i
		whole := c.digits()
		point := c.lit('.')
		frac := ""
		if point {
			frac = c.digits()
		}
		if whole == "" && frac == "" {
			return 0, fmt.Errorf("%q at byte %d is no duration component", c.s[start:], start)
		}
		if c.done() {
			return 0, fmt.Errorf("the duration component %q has no designator", c.s[start:])
		}
		designator := c.s[c.i]
		c.i++
		k := slices.IndexFunc(parts, func(p durationPart) bool { return p.designator == designator })
		if k < 0 {
			return 0, fmt.Errorf("the designator %q at byte %d is out of order or of the other half", designator, c.i-1)
		}
		part := parts[k]
		parts = parts[k+1:]
		numeral := c.s[start : c.i-1]
		if point && designator != 'S' {
			return 0, fmt.Errorf("the duration component %q carries a fraction", c.s[start:c.i])
		}
		var r big.Rat
		if _, ok := r.SetString(numeral); !ok {
			return 0, fmt.Errorf("big.Rat does not read %q", numeral)
		}
		if part.months != 0 {
			months.Add(months, new(big.Int).Mul(r.Num(), big.NewInt(part.months)))
		}
		seconds.Add(seconds, r.Mul(&r, new(big.Rat).SetInt64(part.seconds)))
		n++
	}
	return n, nil
}

// daysInMonth is ·daysInMonth·(y, m) (Datatypes §E.3.2 f-daysInMonth): 28 for
// February of a year not divisible by 4, or divisible by 100 and not by 400,
// 29 for February of any other year or of an absent one, nil, 30 for April,
// June, September and November, and 31 otherwise, an absent month, 0, among
// them.
func daysInMonth(year *big.Int, m uint8) uint8 {
	switch m {
	case 2:
		if year == nil || divisible(year, 4) && (!divisible(year, 100) || divisible(year, 400)) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

// divisible reports whether n is divisible by d.
func divisible(n *big.Int, d int64) bool {
	return new(big.Int).Rem(n, big.NewInt(d)).Sign() == 0
}

// temporalForm is form's for a, a date, time or duration atomic, and v, a
// value of its bare type: the canonical mapping of a's datatype, a
// dateTimeStamp's being dateTime's (§3.4.28), of Datatypes §E.3.5
// (vp-dateTimeCanRep, vp-timeCanRep, vp-dateCanRep, vp-gYearMonthCanRep,
// vp-gYearCanRep, vp-gMonthDayCanRep, vp-gDayCanRep, vp-gMonthCanRep) and §E.2
// (f-durationCanMap, of which f-dayTimeDurationCanMap is a restriction, and
// f-yearMonthDurationCanMap, which spells the zero yearMonthDuration P0M and
// never PT0S, §3.4.26.1's note saying it has none). Its error is that of the
// first property, in field order, that lies outside its §D.2.1 range
// (monthWithin and the checks beside it), a Second or Seconds outside the
// decimal value space, or a duration's Months and Seconds of opposite signs
// (§3.3.6.1), checked before its Seconds. Any other a is an internal bug,
// which panics.
func temporalForm(a atomic, v reflect.Value) (string, error) {
	switch a {
	case atomicDateTime, atomicDateTimeStamp:
		date, err := dateFrag(v)
		if err != nil {
			return "", err
		}
		clock, err := clockFrag(v)
		if err != nil {
			return "", err
		}
		return withOffset(date+"T"+clock, v)
	case atomicTime:
		clock, err := clockFrag(v)
		if err != nil {
			return "", err
		}
		return withOffset(clock, v)
	case atomicDate:
		date, err := dateFrag(v)
		if err != nil {
			return "", err
		}
		return withOffset(date, v)
	case atomicGYearMonth:
		m := uintOf(v, "Month")
		if err := monthWithin(m); err != nil {
			return "", err
		}
		return withOffset(yearFrag(intOf(v, "Year"))+"-"+twoDigits(m), v)
	case atomicGYear:
		return withOffset(yearFrag(intOf(v, "Year")), v)
	case atomicGMonthDay:
		m, d := uintOf(v, "Month"), uintOf(v, "Day")
		if err := monthWithin(m); err != nil {
			return "", err
		}
		if err := dayWithin(nil, m, d); err != nil {
			return "", err
		}
		return withOffset("--"+twoDigits(m)+"-"+twoDigits(d), v)
	case atomicGDay:
		d := uintOf(v, "Day")
		if err := dayWithin(nil, 0, d); err != nil {
			return "", err
		}
		return withOffset("---"+twoDigits(d), v)
	case atomicGMonth:
		m := uintOf(v, "Month")
		if err := monthWithin(m); err != nil {
			return "", err
		}
		return withOffset("--"+twoDigits(m), v)
	case atomicDuration:
		return durationForm(intOf(v, "Months"), ratOf(v, "Seconds"))
	case atomicDayTimeDuration:
		return durationForm(new(big.Int), ratOf(v, "Seconds"))
	case atomicYearMonthDuration:
		m := intOf(v, "Months")
		return durationSign(m.Sign() < 0) + "P" + yearMonthFrag(new(big.Int).Abs(m)), nil
	default:
		panic(fmt.Sprintf("codec: no canonical form of atomic %d", a))
	}
}

// uintOf, intOf and ratOf read v's field name, of type uint8, big.Int and
// big.Rat; the big.Int and big.Rat are copies the caller only reads.
func uintOf(v reflect.Value, name string) uint8 { return uint8(v.FieldByName(name).Uint()) }

func intOf(v reflect.Value, name string) *big.Int {
	i := v.FieldByName(name).Interface().(big.Int)
	return &i
}

func ratOf(v reflect.Value, name string) *big.Rat {
	r := v.FieldByName(name).Interface().(big.Rat)
	return &r
}

// twoDigits is ·unsTwoDigitCanonicalFragmentMap·(n) (§E.3.5
// f-unsTwoDigCanFragMap), n below 100.
func twoDigits(n uint8) string { return fmt.Sprintf("%02d", n) }

// yearFrag is ·yearCanonicalFragmentMap·(y) (f-yrCanFragMap): y in base 10
// where |y| exceeds 9999, and otherwise four digits, zero-padded, after a '-'
// for a negative y.
func yearFrag(y *big.Int) string {
	if y.CmpAbs(big.NewInt(9999)) > 0 {
		return y.String()
	}
	if y.Sign() < 0 {
		return fmt.Sprintf("-%04d", -y.Int64())
	}
	return fmt.Sprintf("%04d", y.Int64())
}

// dateFrag is the yearFrag '-' monthFrag '-' dayFrag of v's Year, Month and
// Day, or the error of the first of Month and Day outside its range.
func dateFrag(v reflect.Value) (string, error) {
	y, m, d := intOf(v, "Year"), uintOf(v, "Month"), uintOf(v, "Day")
	if err := monthWithin(m); err != nil {
		return "", err
	}
	if err := dayWithin(y, m, d); err != nil {
		return "", err
	}
	return yearFrag(y) + "-" + twoDigits(m) + "-" + twoDigits(d), nil
}

// clockFrag is the hourFrag ':' minuteFrag ':' secondFrag of v's Hour, Minute
// and Second, or the error of the first of them outside its range.
func clockFrag(v reflect.Value) (string, error) {
	h, m, s := uintOf(v, "Hour"), uintOf(v, "Minute"), ratOf(v, "Second")
	if err := hourWithin(h); err != nil {
		return "", err
	}
	if err := minuteWithin(m); err != nil {
		return "", err
	}
	if err := secondWithin(s); err != nil {
		return "", err
	}
	c, err := decimalForm(s)
	if err != nil {
		return "", fmt.Errorf("the second %s: %w", s.RatString(), err)
	}
	// ·secondCanonicalFragmentMap· (f-seCanFragMap): two digits before any
	// point, and the fewest after it.
	if strings.IndexByte(c+".", '.') == 1 {
		c = "0" + c
	}
	return twoDigits(h) + ":" + twoDigits(m) + ":" + c, nil
}

// withOffset returns lead followed by ·timezoneCanonicalFragmentMap· of v's
// TimezoneOffset (f-tzCanFragMap), an int16 or a *int16, nothing where that
// is nil: Z for 0, and otherwise its sign, two-digit hours, ':' and two-digit
// minutes; or the error of an offset outside its range.
func withOffset(lead string, v reflect.Value) (string, error) {
	var off int16
	switch x := v.FieldByName("TimezoneOffset").Interface().(type) {
	case int16:
		off = x
	case *int16:
		if x == nil {
			return lead, nil
		}
		off = *x
	}
	if err := offsetWithin(off); err != nil {
		return "", err
	}
	if off == 0 {
		return lead + "Z", nil
	}
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	return fmt.Sprintf("%s%s%02d:%02d", lead, sign, off/60, off%60), nil
}

// durationSign is the sgn of f-durationCanMap and f-yearMonthDurationCanMap:
// '-' for a negative value, and otherwise nothing.
func durationSign(negative bool) string {
	if negative {
		return "-"
	}
	return ""
}

// durationForm is ·durationCanonicalMap·(m, s) (§E.2 f-durationCanMap) for
// the ·months· m and ·seconds· s, or the error for m and s of opposite signs
// (§3.3.6.1) or an s outside the decimal value space.
func durationForm(m *big.Int, s *big.Rat) (string, error) {
	if m.Sign()*s.Sign() < 0 {
		return "", fmt.Errorf("the Months %s and the Seconds %s are of opposite signs (§3.3.6.1)", m, s.RatString())
	}
	lead := durationSign(m.Sign() < 0 || s.Sign() < 0) + "P"
	if s.Sign() == 0 && m.Sign() != 0 {
		return lead + yearMonthFrag(new(big.Int).Abs(m)), nil
	}
	dt, err := dayTimeFrag(new(big.Rat).Abs(s))
	if err != nil {
		return "", fmt.Errorf("the Seconds %s: %w", s.RatString(), err)
	}
	if m.Sign() == 0 {
		return lead + dt, nil
	}
	return lead + yearMonthFrag(new(big.Int).Abs(m)) + dt, nil
}

// yearMonthFrag is ·duYearMonthCanonicalFragmentMap·(ym) (f-duYMCan), ym not
// negative: its years and its months beyond them, each with its designator
// and omitted where 0 but for months where both are, 0M.
func yearMonthFrag(ym *big.Int) string {
	y, m := new(big.Int).QuoRem(ym, big.NewInt(12), new(big.Int))
	switch {
	case y.Sign() == 0:
		return m.String() + "M"
	case m.Sign() == 0:
		return y.String() + "Y"
	}
	return y.String() + "Y" + m.String() + "M"
}

// dayTimeFrag is ·duDayTimeCanonicalFragmentMap·(ss) (f-duDTCan), ss not
// negative: T0S for 0, and otherwise its days, then 'T' and its hours,
// minutes and seconds where any is not 0, each with its designator and
// omitted where 0, the seconds the shortest exact decimal
// (f-duSCan); or decimalForm's error for an ss outside the decimal value
// space.
func dayTimeFrag(ss *big.Rat) (string, error) {
	if ss.Sign() == 0 {
		return "T0S", nil
	}
	whole := new(big.Int).Quo(ss.Num(), ss.Denom())
	frac := new(big.Rat).Sub(ss, new(big.Rat).SetInt(whole))
	d, rest := new(big.Int).QuoRem(whole, big.NewInt(86400), new(big.Int))
	h, rest := new(big.Int).QuoRem(rest, big.NewInt(3600), new(big.Int))
	mi, rest := new(big.Int).QuoRem(rest, big.NewInt(60), new(big.Int))
	sec := new(big.Rat).Add(new(big.Rat).SetInt(rest), frac)
	out := ""
	if d.Sign() != 0 {
		out = d.String() + "D"
	}
	t := ""
	if h.Sign() != 0 {
		t += h.String() + "H"
	}
	if mi.Sign() != 0 {
		t += mi.String() + "M"
	}
	if sec.Sign() != 0 {
		c, err := decimalForm(sec)
		if err != nil {
			return "", err
		}
		t += c + "S"
	}
	if t != "" {
		out += "T" + t
	}
	return out, nil
}
