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

// This file maps the seven remaining seven-property date/time primitives —
// time (§3.3.8), date (§3.3.9), gYearMonth (§3.3.10), gYear (§3.3.11),
// gMonthDay (§3.3.12), gDay (§3.3.13) and gMonth (§3.3.14) — as thin lexical
// "projections" (the spec's word) of dateTime over the shared seven-property
// value model, ·timeOnTimeline· order and timezone machinery datetime.go
// established (issue #103). Each type is the date/timeSevenPropertyModel
// (§D.2.1) with a per-type subset of properties forced ·absent·; only
// ·timezoneOffset· is optional-and-present-or-absent, carried as *int (nil ==
// absent) with no parallel presence flag, exactly as dateTimeVal does. The
// order, equality/identity divergence and imputation-incomparability all reuse
// datetime.go's cmpInstants / sevenProp.instant / imputedOrdering (PRINCIPLES 4);
// this file adds only the seven per-type lexical grammars and canonical maps.
// Every lexical/day-value rejection maps to cvc-datatype-valid (§4.1.4).

// timeLexical is the time lexical space (§3.3.8.2, nt-timeRep), the anchored
// whole-string form of the spec's "equivalent" regular expression with capture
// groups added. Groups: 1 hour, 2 minute, 3 second-integer, 4 second-fraction
// (with the leading '.'), 5 the whole endOfDayFrag (non-empty iff "24:00:00[.0+]"),
// 6 timezoneFrag. Unlike dateTime, endOfDayFrag has NO next day to carry into: the
// ·timeLexicalMap· (vp-timeLexRep, §E.3.5) maps both "00:00:00" and "24:00:00" to
// the same midnight value (hour=minute=second=0).
var timeLexical = regexp.MustCompile(
	`^(?:([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])(\.[0-9]+)?|(24:00:00(?:\.0+)?))` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// dateLexical is the date lexical space (§3.3.9.2, nt-dateRep). Groups: 1 year
// (optional leading '-'), 2 month, 3 day, 4 timezoneFrag. dateLexicalRep has NO
// endOfDayFrag alternative at all, so a "24:00:00"-shaped literal is not even a
// syntactic candidate. The day-of-month value constraint (con-date-dayValue,
// §3.3.9.1) is year-dependent and BEYOND this regex; parseDate checks it.
var dateLexical = regexp.MustCompile(
	`^(-?(?:[1-9][0-9]{3,}|0[0-9]{3}))-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// gYearMonthLexical is the gYearMonth lexical space (§3.3.10.2, nt-gYearMonthRep).
// Groups: 1 year, 2 month, 3 timezoneFrag.
var gYearMonthLexical = regexp.MustCompile(
	`^(-?(?:[1-9][0-9]{3,}|0[0-9]{3}))-(0[1-9]|1[0-2])` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// gYearLexical is the gYear lexical space (§3.3.11.2, nt-gYearRep). Groups:
// 1 year, 2 timezoneFrag. gYear permits a timezone (·timezoneOffset· stays
// optional, §3.3.11.1) — it is not timezone-forbidden.
var gYearLexical = regexp.MustCompile(
	`^(-?(?:[1-9][0-9]{3,}|0[0-9]{3}))` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// gMonthDayLexical is the gMonthDay lexical space (§3.3.12.2, nt-gMonthDayRep).
// Groups: 1 month, 2 day, 3 timezoneFrag. The day-of-month value constraint
// (con-gMonthDay-dayValue, §3.3.12.1) is year-INDEPENDENT (gMonthDay has no
// ·year·), so --02-29 is unconditionally valid; parseGMonthDay checks the flat
// per-month bound.
var gMonthDayLexical = regexp.MustCompile(
	`^--(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// gDayLexical is the gDay lexical space (§3.3.13.2, nt-gDayRep). Groups: 1 day,
// 2 timezoneFrag. gDay has NO day-of-month representation constraint: the flat
// 1-31 range the regex enforces is the whole rule (§3.3.13.1).
var gDayLexical = regexp.MustCompile(
	`^---(0[1-9]|[12][0-9]|3[01])` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// gMonthLexical is the gMonth lexical space (§3.3.14.2, nt-gMonthRep). Groups:
// 1 month, 2 timezoneFrag.
var gMonthLexical = regexp.MustCompile(
	`^--(0[1-9]|1[0-2])` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// matchTimezone maps an optional timezoneFrag capture (empty ⇒ absent) to a
// *int ·timezoneOffset· in minutes: nil when the fragment is empty, otherwise
// timezoneOffset's signed hh:mm (f-dt-tzMap, §E.3.5). Shared by every per-type
// parser so absence is encoded one way (nil, never a bool).
func matchTimezone(frag string) *int {
	if frag == "" {
		return nil
	}
	off := timezoneOffset(frag)
	return &off
}

// timeVal is an xs:time value (§3.3.8): the seven-property model with
// year/month/day ·absent·. ·second· is one decimal (*big.Rat, as dateTimeVal);
// hour is never 24 (endOfDayFrag maps to midnight at parse time).
type timeVal struct {
	hour, minute int      // 0-23, 0-59
	second       *big.Rat // 0 ≤ second < 60
	tzOffset     *int     // minutes, −840..840; nil == absent
}

// dateVal is an xs:date value (§3.3.9): year/month/day present, hour/minute/second
// ·absent·. The stored day satisfies con-date-dayValue for its month and year.
type dateVal struct {
	year       *big.Int // unbounded; 0 = 1 BCE (XSD 1.1 permits year 0)
	month, day int      // 1-12, 1-31 (day valid for month/year)
	tzOffset   *int     // nil == absent
}

// gYearMonthVal is an xs:gYearMonth value (§3.3.10): year and month present, all
// else ·absent·.
type gYearMonthVal struct {
	year     *big.Int
	month    int // 1-12
	tzOffset *int
}

// gYearVal is an xs:gYear value (§3.3.11): year present, all else ·absent·.
type gYearVal struct {
	year     *big.Int
	tzOffset *int
}

// gMonthDayVal is an xs:gMonthDay value (§3.3.12): month and day present, all
// else ·absent·. The stored day satisfies con-gMonthDay-dayValue (year-free).
type gMonthDayVal struct {
	month, day int // 1-12, 1-31 (day ≤ per-month bound)
	tzOffset   *int
}

// gDayVal is an xs:gDay value (§3.3.13): day present, all else ·absent·.
type gDayVal struct {
	day      int // 1-31
	tzOffset *int
}

// gMonthVal is an xs:gMonth value (§3.3.14): month present, all else ·absent·.
type gMonthVal struct {
	month    int // 1-12
	tzOffset *int
}

// gMonthDayMaxDay is the maximum ·day· con-gMonthDay-dayValue (§3.3.12.1) allows
// for month: 30 for April/June/September/November, 29 for February
// (UNCONDITIONAL — gMonthDay has no ·year·, so the leap-day --02-29 is always in
// the value space), 31 otherwise. gMonthDayLexical admits day 31 for every month,
// so a day beyond this bound is outside the lexical space (§4.1.4).
func gMonthDayMaxDay(month int) int {
	switch month {
	case 4, 6, 9, 11:
		return 30
	case 2:
		return 29
	}
	return 31
}

// parseTime maps a time lexical to its value (·timeLexicalMap·, vp-timeLexRep,
// §E.3.5): the endOfDayFrag "24:00:00" maps to midnight (not a day carry — time
// has no day), else hour/minute/second map directly.
func parseTime(lexical string, _ value.Context) (value.Value, error) {
	m := timeLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"time: %q is not in the lexical space (timeLexicalRep, §3.3.8.2)", lexical)
	}
	tz := matchTimezone(m[6])
	if m[5] != "" { // endOfDayFrag → midnight (§3.3.8.2 Note)
		return timeVal{hour: 0, minute: 0, second: new(big.Rat), tzOffset: tz}, nil
	}
	hour, _ := strconv.Atoi(m[1])
	minute, _ := strconv.Atoi(m[2])
	second, _ := new(big.Rat).SetString(m[3] + m[4]) // "SS" + ".fff" (or "")
	return timeVal{hour: hour, minute: minute, second: second, tzOffset: tz}, nil
}

// parseDate maps a date lexical to its value (·dateLexicalMap·, vp-dateLexRep,
// §E.3.5). The regex admits day 31 for any month, so con-date-dayValue (§3.3.9.1,
// year-dependent, leap-aware) is checked here; a day-of-month violation is
// outside the lexical space just like a grammar miss (§4.1.4).
func parseDate(lexical string, _ value.Context) (value.Value, error) {
	m := dateLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"date: %q is not in the lexical space (dateLexicalRep, §3.3.9.2)", lexical)
	}
	year, _ := new(big.Int).SetString(m[1], 10)
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	if day > daysInMonth(year, month) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"date: %q has day %d out of range for month %d of year %s (con-date-dayValue, §3.3.9.1)",
			lexical, day, month, year)
	}
	return dateVal{year: year, month: month, day: day, tzOffset: matchTimezone(m[4])}, nil
}

// parseGYearMonth maps a gYearMonth lexical to its value (·gYearMonthLexicalMap·,
// vp-gYearMonthLexRep, §E.3.5).
func parseGYearMonth(lexical string, _ value.Context) (value.Value, error) {
	m := gYearMonthLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gYearMonth: %q is not in the lexical space (gYearMonthLexicalRep, §3.3.10.2)", lexical)
	}
	year, _ := new(big.Int).SetString(m[1], 10)
	month, _ := strconv.Atoi(m[2])
	return gYearMonthVal{year: year, month: month, tzOffset: matchTimezone(m[3])}, nil
}

// parseGYear maps a gYear lexical to its value (·gYearLexicalMap·,
// vp-gYearLexRep, §E.3.5).
func parseGYear(lexical string, _ value.Context) (value.Value, error) {
	m := gYearLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gYear: %q is not in the lexical space (gYearLexicalRep, §3.3.11.2)", lexical)
	}
	year, _ := new(big.Int).SetString(m[1], 10)
	return gYearVal{year: year, tzOffset: matchTimezone(m[2])}, nil
}

// parseGMonthDay maps a gMonthDay lexical to its value (·gMonthDayLexicalMap·,
// vp-gMonthDayLexRep, §E.3.5). con-gMonthDay-dayValue (§3.3.12.1) is checked here
// (year-free): the regex admits day 31 for every month.
func parseGMonthDay(lexical string, _ value.Context) (value.Value, error) {
	m := gMonthDayLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gMonthDay: %q is not in the lexical space (gMonthDayLexicalRep, §3.3.12.2)", lexical)
	}
	month, _ := strconv.Atoi(m[1])
	day, _ := strconv.Atoi(m[2])
	if day > gMonthDayMaxDay(month) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gMonthDay: %q has day %d out of range for month %d (con-gMonthDay-dayValue, §3.3.12.1)",
			lexical, day, month)
	}
	return gMonthDayVal{month: month, day: day, tzOffset: matchTimezone(m[3])}, nil
}

// parseGDay maps a gDay lexical to its value (·gDayLexicalMap·, vp-gDayLexRep,
// §E.3.5). gDay has no day-of-month representation constraint: the regex's flat
// 1-31 range is the whole rule (§3.3.13.1).
func parseGDay(lexical string, _ value.Context) (value.Value, error) {
	m := gDayLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gDay: %q is not in the lexical space (gDayLexicalRep, §3.3.13.2)", lexical)
	}
	day, _ := strconv.Atoi(m[1])
	return gDayVal{day: day, tzOffset: matchTimezone(m[2])}, nil
}

// parseGMonth maps a gMonth lexical to its value (·gMonthLexicalMap·,
// vp-gMonthLexRep, §E.3.5).
func parseGMonth(lexical string, _ value.Context) (value.Value, error) {
	m := gMonthLexical.FindStringSubmatch(lexical)
	if m == nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gMonth: %q is not in the lexical space (gMonthLexicalRep, §3.3.14.2)", lexical)
	}
	month, _ := strconv.Atoi(m[1])
	return gMonthVal{month: month, tzOffset: matchTimezone(m[2])}, nil
}

// canonicalTime is the Mapping.Canonical wrapper: a foreign value is an
// *xsderr.Error, not a panic (warden guardrail, canonicalDateTime precedent).
func canonicalTime(v value.Value) (string, error) {
	t, ok := v.(timeVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"time canonical: value of type %T is not a strict time", v)
	}
	return t.Canonical(), nil
}

func canonicalDate(v value.Value) (string, error) {
	d, ok := v.(dateVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"date canonical: value of type %T is not a strict date", v)
	}
	return d.Canonical(), nil
}

func canonicalGYearMonth(v value.Value) (string, error) {
	g, ok := v.(gYearMonthVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gYearMonth canonical: value of type %T is not a strict gYearMonth", v)
	}
	return g.Canonical(), nil
}

func canonicalGYear(v value.Value) (string, error) {
	g, ok := v.(gYearVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gYear canonical: value of type %T is not a strict gYear", v)
	}
	return g.Canonical(), nil
}

func canonicalGMonthDay(v value.Value) (string, error) {
	g, ok := v.(gMonthDayVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gMonthDay canonical: value of type %T is not a strict gMonthDay", v)
	}
	return g.Canonical(), nil
}

func canonicalGDay(v value.Value) (string, error) {
	g, ok := v.(gDayVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gDay canonical: value of type %T is not a strict gDay", v)
	}
	return g.Canonical(), nil
}

func canonicalGMonth(v value.Value) (string, error) {
	g, ok := v.(gMonthVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"gMonth canonical: value of type %T is not a strict gMonth", v)
	}
	return g.Canonical(), nil
}

// Canonical renders the canonical time lexical (·timeCanonicalMap·, vp-timeCanRep,
// §E.3.6): hour:minute:second, with the timezone fragment appended iff present.
func (t timeVal) Canonical() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%02d:%02d:", t.hour, t.minute)
	b.WriteString(dateTimeSecondFragment(t.second))
	if t.tzOffset != nil {
		b.WriteString(timezoneCanonicalFragment(*t.tzOffset))
	}
	return b.String()
}

// Canonical renders the canonical date lexical (·dateCanonicalMap·, vp-dateCanRep,
// §E.3.6): year-month-day, with the timezone fragment appended iff present.
func (d dateVal) Canonical() string {
	var b strings.Builder
	b.WriteString(yearCanonicalFragment(d.year))
	fmt.Fprintf(&b, "-%02d-%02d", d.month, d.day)
	if d.tzOffset != nil {
		b.WriteString(timezoneCanonicalFragment(*d.tzOffset))
	}
	return b.String()
}

// Canonical renders the canonical gYearMonth lexical (·gYearMonthCanonicalMap·,
// vp-gYearMonthCanRep, §E.3.6): year-month, timezone appended iff present.
func (g gYearMonthVal) Canonical() string {
	var b strings.Builder
	b.WriteString(yearCanonicalFragment(g.year))
	fmt.Fprintf(&b, "-%02d", g.month)
	if g.tzOffset != nil {
		b.WriteString(timezoneCanonicalFragment(*g.tzOffset))
	}
	return b.String()
}

// Canonical renders the canonical gYear lexical (·gYearCanonicalMap·,
// vp-gYearCanRep, §E.3.6): the yearFrag, timezone appended iff present.
func (g gYearVal) Canonical() string {
	s := yearCanonicalFragment(g.year)
	if g.tzOffset != nil {
		s += timezoneCanonicalFragment(*g.tzOffset)
	}
	return s
}

// Canonical renders the canonical gMonthDay lexical (·gMonthDayCanonicalMap·,
// vp-gMonthDayCanRep, §E.3.6): "--" month "-" day, timezone appended iff present.
func (g gMonthDayVal) Canonical() string {
	var b strings.Builder
	fmt.Fprintf(&b, "--%02d-%02d", g.month, g.day)
	if g.tzOffset != nil {
		b.WriteString(timezoneCanonicalFragment(*g.tzOffset))
	}
	return b.String()
}

// Canonical renders the canonical gDay lexical (·gDayCanonicalMap·, vp-gDayCanRep,
// §E.3.6): "---" day, timezone appended iff present.
func (g gDayVal) Canonical() string {
	s := fmt.Sprintf("---%02d", g.day)
	if g.tzOffset != nil {
		s += timezoneCanonicalFragment(*g.tzOffset)
	}
	return s
}

// Canonical renders the canonical gMonth lexical (·gMonthCanonicalMap·,
// vp-gMonthCanRep, §E.3.6): "--" month, timezone appended iff present. The spec's
// literal text applies ·monthCanonicalFragmentMap· to "gM's ·day·", but gMonth's
// ·day· is forced absent (§3.3.14.1) — a transcription artifact; the
// semantically-forced reading is the ·month· property (PRINCIPLES 25).
func (g gMonthVal) Canonical() string {
	s := fmt.Sprintf("--%02d", g.month)
	if g.tzOffset != nil {
		s += timezoneCanonicalFragment(*g.tzOffset)
	}
	return s
}

// instant is each type's ·timeOnTimeline· (§E.3.4), built from a sevenProp whose
// absent properties nil-fill to the shared 1972-12-31T00:00:00 civil filler.

func (t timeVal) instant() *big.Rat {
	return sevenProp{hour: &t.hour, minute: &t.minute, second: t.second, tz: t.tzOffset}.instant()
}

func (d dateVal) instant() *big.Rat {
	return sevenProp{year: d.year, month: &d.month, day: &d.day, tz: d.tzOffset}.instant()
}

func (g gYearMonthVal) instant() *big.Rat {
	return sevenProp{year: g.year, month: &g.month, tz: g.tzOffset}.instant()
}

func (g gYearVal) instant() *big.Rat {
	return sevenProp{year: g.year, tz: g.tzOffset}.instant()
}

func (g gMonthDayVal) instant() *big.Rat {
	return sevenProp{month: &g.month, day: &g.day, tz: g.tzOffset}.instant()
}

func (g gDayVal) instant() *big.Rat {
	return sevenProp{day: &g.day, tz: g.tzOffset}.instant()
}

func (g gMonthVal) instant() *big.Rat {
	return sevenProp{month: &g.month, tz: g.tzOffset}.instant()
}

// Cmp is each type's PARTIAL order (§D.2.1) over ·timeOnTimeline· via the shared
// cmpInstants (the ±840-minute dual imputation). A foreign argument (including a
// different seven-property type, e.g. gYear vs dateTime) is Incomparable — a
// definite verdict, not a fail-open. Eq is the derived equality (equal iff the
// order is Equal). Identical is the structural §2.2.2 relation over the stored
// properties INCLUDING the exact ·timezoneOffset·, so a timezone-shifted pair
// denoting the same instant is Eq but not Identical.

func (t timeVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(timeVal)
	if !ok {
		return value.Incomparable
	}
	return cmpInstants(t.instant(), o.instant(), t.tzOffset != nil, o.tzOffset != nil)
}

func (t timeVal) Eq(other value.Value) bool { return t.Cmp(other) == value.Equal }

func (t timeVal) Identical(other value.Value) bool {
	o, ok := other.(timeVal)
	if !ok {
		return false
	}
	return t.hour == o.hour && t.minute == o.minute &&
		t.second.Cmp(o.second) == 0 && tzOffsetEqual(t.tzOffset, o.tzOffset)
}

func (t timeVal) HasTimezone() bool { return t.tzOffset != nil }

func (d dateVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(dateVal)
	if !ok {
		return value.Incomparable
	}
	return cmpInstants(d.instant(), o.instant(), d.tzOffset != nil, o.tzOffset != nil)
}

func (d dateVal) Eq(other value.Value) bool { return d.Cmp(other) == value.Equal }

func (d dateVal) Identical(other value.Value) bool {
	o, ok := other.(dateVal)
	if !ok {
		return false
	}
	return d.year.Cmp(o.year) == 0 && d.month == o.month && d.day == o.day &&
		tzOffsetEqual(d.tzOffset, o.tzOffset)
}

func (d dateVal) HasTimezone() bool { return d.tzOffset != nil }

func (g gYearMonthVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(gYearMonthVal)
	if !ok {
		return value.Incomparable
	}
	return cmpInstants(g.instant(), o.instant(), g.tzOffset != nil, o.tzOffset != nil)
}

func (g gYearMonthVal) Eq(other value.Value) bool { return g.Cmp(other) == value.Equal }

func (g gYearMonthVal) Identical(other value.Value) bool {
	o, ok := other.(gYearMonthVal)
	if !ok {
		return false
	}
	return g.year.Cmp(o.year) == 0 && g.month == o.month && tzOffsetEqual(g.tzOffset, o.tzOffset)
}

func (g gYearMonthVal) HasTimezone() bool { return g.tzOffset != nil }

func (g gYearVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(gYearVal)
	if !ok {
		return value.Incomparable
	}
	return cmpInstants(g.instant(), o.instant(), g.tzOffset != nil, o.tzOffset != nil)
}

func (g gYearVal) Eq(other value.Value) bool { return g.Cmp(other) == value.Equal }

func (g gYearVal) Identical(other value.Value) bool {
	o, ok := other.(gYearVal)
	if !ok {
		return false
	}
	return g.year.Cmp(o.year) == 0 && tzOffsetEqual(g.tzOffset, o.tzOffset)
}

func (g gYearVal) HasTimezone() bool { return g.tzOffset != nil }

func (g gMonthDayVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(gMonthDayVal)
	if !ok {
		return value.Incomparable
	}
	return cmpInstants(g.instant(), o.instant(), g.tzOffset != nil, o.tzOffset != nil)
}

func (g gMonthDayVal) Eq(other value.Value) bool { return g.Cmp(other) == value.Equal }

func (g gMonthDayVal) Identical(other value.Value) bool {
	o, ok := other.(gMonthDayVal)
	if !ok {
		return false
	}
	return g.month == o.month && g.day == o.day && tzOffsetEqual(g.tzOffset, o.tzOffset)
}

func (g gMonthDayVal) HasTimezone() bool { return g.tzOffset != nil }

func (g gDayVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(gDayVal)
	if !ok {
		return value.Incomparable
	}
	return cmpInstants(g.instant(), o.instant(), g.tzOffset != nil, o.tzOffset != nil)
}

func (g gDayVal) Eq(other value.Value) bool { return g.Cmp(other) == value.Equal }

func (g gDayVal) Identical(other value.Value) bool {
	o, ok := other.(gDayVal)
	if !ok {
		return false
	}
	return g.day == o.day && tzOffsetEqual(g.tzOffset, o.tzOffset)
}

func (g gDayVal) HasTimezone() bool { return g.tzOffset != nil }

func (g gMonthVal) Cmp(other value.Value) value.Ordering {
	o, ok := other.(gMonthVal)
	if !ok {
		return value.Incomparable
	}
	return cmpInstants(g.instant(), o.instant(), g.tzOffset != nil, o.tzOffset != nil)
}

func (g gMonthVal) Eq(other value.Value) bool { return g.Cmp(other) == value.Equal }

func (g gMonthVal) Identical(other value.Value) bool {
	o, ok := other.(gMonthVal)
	if !ok {
		return false
	}
	return g.month == o.month && tzOffsetEqual(g.tzOffset, o.tzOffset)
}

func (g gMonthVal) HasTimezone() bool { return g.tzOffset != nil }
