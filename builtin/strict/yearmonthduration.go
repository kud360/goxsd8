package strict

import (
	"errors"
	"fmt"
	"math/big"
	"regexp"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// yearMonthDurationLexical is the yearMonthDuration lexical space
// (§3.4.26.1, yearMonthDurationLexicalRep, nt-42): duration's grammar with only
// the year-month half retained, transcribed verbatim from the two alternate
// regexes §3.4.26.1 gives. Anchored so the whole literal must match; a day/time
// character (D, T, H, S) is outside this space, which is the narrower gate Parse
// must apply independently of the generic pattern facet (value/backend.go).
var yearMonthDurationLexical = regexp.MustCompile(`^-?P(([0-9]+Y([0-9]+M)?)|([0-9]+M))$`)

// errNoYearMonthCanonical marks the one yearMonthDuration value (·months·=0) that
// has no canonical representation (§3.4.26.1 Note). It is unexported: no consumer
// distinguishes it yet (STYLE T5), but wrapping it keeps the case errors.Is-
// identifiable inside the package.
var errNoYearMonthCanonical = errors.New("yearMonthDuration: zero value has no canonical representation")

// parseYearMonthDuration maps a yearMonthDurationLexicalRep to its value
// (·yearMonthDurationMap·, f-yearMonthDurationMap, §3.4.26.1/§E.2): a restriction
// of ·durationMap· to the year-month half. It gates the narrower lexical space
// itself (the pattern facet may not have run — value/backend.go), then reuses
// duration's field extraction and ·duYearMonthFragmentMap· math verbatim.
func parseYearMonthDuration(lexical string, _ value.Context) (value.Value, error) {
	if !yearMonthDurationLexical.MatchString(lexical) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"yearMonthDuration: %q is not in the lexical space (yearMonthDurationLexicalRep, §3.4.26.1)", lexical)
	}
	f := durationFields.FindStringSubmatch(lexical)
	// f[1]=sign, f[2]=years, f[3]=months; the day/time groups are always empty
	// here because the anchor above rejects any D/T character.
	negative := f[1] == "-"

	months := new(big.Int)
	addMonths(months, f[2], 12)
	addMonths(months, f[3], 1)

	// ·seconds· is definitionally absent from yearMonthDuration's value space
	// (§3.4.26), not coincidentally zero: state it as the zero value directly
	// rather than derive it from empty day/time fields (STYLE D3).
	seconds := new(big.Rat)

	if months.Sign() == 0 {
		negative = false
	}
	return durationVal{negative: negative, months: months, seconds: seconds}, nil
}

// canonicalYearMonthDuration is the Mapping.Canonical wrapper
// (·yearMonthDurationCanonicalMap·, f-yearMonthDurationCanMap, §3.4.26.1/§E.2): a
// restriction of ·durationCanonicalMap· to the year-month half. It cannot reuse
// durationVal.Canonical, whose ·months·=0 branch emits "PT0S" — a 'T'-bearing
// string outside yearMonthDuration's [^DT]* lexical space (§3.4.26.2). A foreign
// value is a caller-contract violation (*xsderr.Error, like canonicalDuration);
// the zero value is a legally VALID value that simply has no canonical form
// (§3.4.26.1 Note, licensed by §2.3.1's "(where possible)"), so it is a plain
// error, NOT a validity verdict (xsderr/error.go: an *Error means a validity
// violation, which this is not).
func canonicalYearMonthDuration(v value.Value) (string, error) {
	d, ok := v.(durationVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"yearMonthDuration canonical: value of type %T is not a strict yearMonthDuration", v)
	}
	if d.months.Sign() == 0 {
		return "", fmt.Errorf("%w (·months·=0; duration's 'PT0S' is outside [^DT]*, §3.4.26.1 Note, dt-canonical-mapping)", errNoYearMonthCanonical)
	}
	sgn := ""
	if d.negative {
		sgn = "-"
	}
	return sgn + "P" + yearMonthCanonicalFragment(d.months), nil
}
