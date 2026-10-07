package xpath

import (
	"testing"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// The fixtures below drive fn:string and the cast to xs:string over a typed
// xs:date, xs:dateTime or xs:time operand, which xpath-functions.md §17.1.2
// renders as its local value (ctaTypes.localValueSource).

// dsUses is the attribute types the rows read: d an xs:date, dt an
// xs:dateTime, tm an xs:time, ts an xs:dateTimeStamp, g an xs:gYear, f an
// xs:double and n an xs:decimal.
func dsUses(t *testing.T) AttributeTypes {
	t.Helper()
	return asUses(t, map[string]string{
		"d": "date", "dt": "dateTime", "tm": "time", "ts": "dateTimeStamp",
		"g": "gYear", "f": "double", "n": "decimal",
	})
}

// dsCompile compiles expr for an E with empty content over dsUses, reporting
// whether it compiled.
func dsCompile(t *testing.T, expr string) (AssertionTest, bool) {
	t.Helper()
	return CompileAssertionTest(asRecord(expr), seededTypes, xsd.EmptyContent{}, dsUses(t), asNoElems)
}

// assert-simple006's {test} over a restriction of union(xs:date,
// xs:dateTime): `$value` is typed by its ·active basic member· (dt-xdmrep
// clause 4), and fn:string over it is §17.1.2's cast to xs:string, so the
// facet holds for 2008-01-01 and 2008-01-01T00:00:00 and fails for
// 2009-01-01. Every row declines with castsFrom's localValueSource arm
// removed.
func TestFacetAssertionsStringOfADateOrDateTime(t *testing.T) {
	date, dateTime := ctaBuiltin("date"), ctaBuiltin("dateTime")
	union := fcUnion(t, "DateOrDateTime", date, dateTime)
	test := "starts-with(string($value), '2008')"
	st := fcAssertedUnion(t, union, test, date, dateTime)
	for _, tc := range []struct {
		lexical string
		want    value.AssertionOutcome
	}{
		{"2008-01-01", value.AssertionHolds},
		{"2008-01-01T00:00:00", value.AssertionHolds},
		{"2009-01-01", value.AssertionFails},
	} {
		_, err := value.ValidateLexical(backend(), asTypesWith(union), st, tc.lexical, nil, FacetAssertions())
		if got := fcOutcome(t, err); got != tc.want {
			t.Errorf("%s over %q = %d, want %d (err %v)", test, tc.lexical, got, tc.want, err)
		}
	}
}

// fn:string over a typed date/time attribute, and both cast spellings to
// xs:string, render its local value: `string(@d) = '2008-01-01Z'` over an
// xs:date 2008-01-01Z holds, the constructor `xs:string(@d)` and `@d cast as
// xs:string` are the same cast, a comparison over it parenthesised whole
// included — a parenthesised comparison OPERAND declines whatever it holds,
// `(@s) = 'a'` too — an xs:dateTimeStamp is admitted through its xs:dateTime
// primitive, and fn:string over the identity cast `xs:date(@d)` or over
// fn:distinct-values of @d is the same string. Every row declines with
// castsFrom's localValueSource arm removed.
func TestStringOfATypedDateAttribute(t *testing.T) {
	d := []asTyped{{uq("d"), "date", "2008-01-01Z"}}
	for _, tc := range []struct {
		expr  string
		attrs []asTyped
		want  bool
	}{
		{"string(@d) = '2008-01-01Z'", d, true},
		{"string(@d) = '2008-01-01'", d, false},
		{"string(@d) eq '2008-01-01Z'", d, true},
		{"xs:string(@d) = '2008-01-01Z'", d, true},
		{"@d cast as xs:string = '2008-01-01Z'", d, true},
		{"(@d cast as xs:string = '2008-01-01Z')", d, true},
		{"string(xs:date(@d)) = '2008-01-01Z'", d, true},
		{"string(distinct-values(@d)) = '2008-01-01Z'", d, true},
		{"string(@d) = ''", nil, true},
		{"string(@ts) = '2008-01-01T00:00:00Z'", []asTyped{{uq("ts"), "dateTimeStamp", "2008-01-01T00:00:00Z"}}, true},
		{"string(@tm) = '00:00:00'", []asTyped{{uq("tm"), "time", "24:00:00"}}, true},
	} {
		t.Run(tc.expr, func(t *testing.T) {
			test, ok := dsCompile(t, tc.expr)
			if !ok {
				t.Fatalf("CompileAssertionTest(%q): declined, want compiled", tc.expr)
			}
			got := test.Evaluate(backend(), seededTypes, asValues(t, tc.attrs...), asNoChildren, nil, ValueBinding{})
			if got != tc.want {
				t.Errorf("Evaluate(%q) over %v = %v, want %v", tc.expr, tc.attrs, got, tc.want)
			}
		})
	}
}

// §17.1.2 renders the VALUE, never the lexical it was read from, so the
// string is the ·canonical representation· (dateCanonicalMap and its
// siblings): a -00:30 offset stays -00:30, where eg:convertTZtoString's
// `$tzh >= 0` would render +00:30; +00:00 and -00:00 render Z; a .000
// fraction is dropped; xs:time 24:00:00 is 00:00:00 ("the hours component of
// the resulting string will never be "24""), and an xs:dateTime at 24:00:00 is
// the next day's midnight. Each row is fn:string over `$value` under the
// type, and declines with castsFrom's localValueSource arm removed.
func TestStringOfADateTimeRendersTheLocalValue(t *testing.T) {
	for _, tc := range []struct {
		typ, lexical, want string
	}{
		{"date", "2008-01-01-00:30", "2008-01-01-00:30"},
		{"date", "2008-01-01+00:00", "2008-01-01Z"},
		{"date", "2008-01-01-00:00", "2008-01-01Z"},
		{"date", "2008-01-01", "2008-01-01"},
		{"time", "12:00:00.000", "12:00:00"},
		{"time", "12:00:00.500", "12:00:00.5"},
		{"time", "24:00:00", "00:00:00"},
		{"dateTime", "2008-01-01T24:00:00", "2008-01-02T00:00:00"},
		{"dateTime", "2008-01-01T10:00:00+00:00", "2008-01-01T10:00:00Z"},
	} {
		st := asBuiltin(t, tc.typ)
		expr := "string($value) eq '" + tc.want + "'"
		if !afEvalValue(t, expr, st, tc.lexical) {
			t.Errorf("Evaluate(%s) over xs:%s %q = false, want true", expr, tc.typ, tc.lexical)
		}
	}
	if afEvalValue(t, "string($value) eq '2008-01-01+00:30'", asBuiltin(t, "date"), "2008-01-01-00:30") {
		t.Error("Evaluate(string($value) eq '2008-01-01+00:30') over xs:date 2008-01-01-00:30 = true, want false: the offset is negative")
	}
}

// What stays declined (castsFrom's GAP(xpath), #1042): the arm keys on the
// date/time primitives and on xs:string itself, so fn:string over an
// xs:double (#2339), an xs:decimal or an xs:gYear operand declines, and so
// does a cast of an xs:date operand to xs:token, a target derived from
// xs:string (§17.5). With localValueSource answering true for every primitive
// the first three rows compile, and with the arm keyed on the target's
// primitive in place of xs:string itself the xs:token rows do.
func TestStringOfAnotherTypedOperandDeclines(t *testing.T) {
	for _, expr := range []string{
		"string(@f) = '1.5'",
		"string(@n) = '1.5'",
		"string(@g) = '2008'",
		"xs:token(@d) = '2008-01-01'",
		"@d cast as xs:token = '2008-01-01'",
	} {
		if _, ok := dsCompile(t, expr); ok {
			t.Errorf("CompileAssertionTest(%q): compiled, want declined", expr)
		}
	}
}
