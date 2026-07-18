package strict

import (
	"math/big"
	"regexp"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// dayTimeDurationLexical is the dayTimeDuration lexical space
// (§3.4.27.1, dayTimeDurationLexicalRep, nt-43): durationLexical with the pre-'T'
// year and month alternatives deleted, keeping the day alternative and the whole
// post-'T' time grammar (so "PT5M" — minutes after 'T' — stays accepted; only the
// pre-'T' month branch is dropped). Anchored so the whole literal must match; a
// year/month character (Y, or a pre-'T' M) is outside this space, the narrower
// gate Parse must apply itself (value/backend.go).
var dayTimeDurationLexical = regexp.MustCompile(`^-?P((([0-9]+D)(T(([0-9]+H)([0-9]+M)?([0-9]+(\.[0-9]+)?S)?|([0-9]+M)([0-9]+(\.[0-9]+)?S)?|([0-9]+(\.[0-9]+)?S)))?)|(T(([0-9]+H)([0-9]+M)?([0-9]+(\.[0-9]+)?S)?|([0-9]+M)([0-9]+(\.[0-9]+)?S)?|([0-9]+(\.[0-9]+)?S))))$`)

// parseDayTimeDuration maps a dayTimeDurationLexicalRep to its value
// (·dayTimeDurationMap·, f-dayTimeDurationMap, §3.4.27.1/§E.2): a restriction of
// ·durationMap· to the day-time half. It gates the narrower lexical space itself
// (the pattern facet may not have run — value/backend.go), then reuses duration's
// field extraction and ·duDayTimeFragmentMap· math verbatim.
func parseDayTimeDuration(lexical string, _ value.Context) (value.Value, error) {
	if !dayTimeDurationLexical.MatchString(lexical) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"dayTimeDuration: %q is not in the lexical space (dayTimeDurationLexicalRep, §3.4.27.1)", lexical)
	}
	f := durationFields.FindStringSubmatch(lexical)
	// f[1]=sign, f[4]=days, f[5]=hours, f[6]=minutes, f[7]=seconds; the
	// year/month groups are always empty here because the anchor above rejects
	// any Y or pre-'T' M character.
	negative := f[1] == "-"

	// ·months· is definitionally absent from dayTimeDuration's value space
	// (§3.4.27), not coincidentally zero: state it as the zero value directly
	// rather than derive it from empty year/month fields (STYLE D3).
	months := new(big.Int)

	seconds := new(big.Rat)
	addSeconds(seconds, f[4], 86400)
	addSeconds(seconds, f[5], 3600)
	addSeconds(seconds, f[6], 60)
	addSecondFraction(seconds, f[7])

	if seconds.Sign() == 0 {
		negative = false
	}
	return durationVal{negative: negative, months: months, seconds: seconds}, nil
}

// canonicalDayTimeDuration is the Mapping.Canonical wrapper
// (·dayTimeDurationCanonicalMap·, f-dayTimeDurationCanMap, §3.4.27.1/§E.2): a
// restriction of ·durationCanonicalMap· to the day-time half. It reuses
// durationVal.Canonical unchanged: for any value parseDayTimeDuration produces
// ·months· is 0, so the switch always takes its day-time (default) branch, whose
// output — "T0S" for the zero value included — is always inside dayTimeDuration's
// [^YM]*(T.*)? lexical space (§3.4.27.2), so the restriction is total (no
// no-canonical-form edge case, unlike yearMonthDuration). A foreign value is a
// caller-contract violation reported as *xsderr.Error, like canonicalDuration.
func canonicalDayTimeDuration(v value.Value) (string, error) {
	d, ok := v.(durationVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"dayTimeDuration canonical: value of type %T is not a strict dayTimeDuration", v)
	}
	return d.Canonical(), nil
}
