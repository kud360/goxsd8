package strict

import "testing"

// TestNormalizeWhiteSpaceModes exercises the three §4.3.6 modes directly on raw
// text carrying tabs, newlines, carriage returns and space runs.
func TestNormalizeWhiteSpaceModes(t *testing.T) {
	const raw = "  a\tb\r\nc  d  "
	cases := []struct {
		mode whiteSpace
		want string
	}{
		{preserveWS, "  a\tb\r\nc  d  "}, // identity
		{replaceWS, "  a b  c  d  "},     // #x9/#xA/#xD → #x20 (\r\n → two spaces), no collapse/trim
		{collapseWS, "a b c d"},          // replace, then collapse runs and trim ends
	}
	for _, c := range cases {
		if got := normalizeWhiteSpace(raw, c.mode); got != c.want {
			t.Errorf("normalizeWhiteSpace(%q, %d) = %q, want %q", raw, c.mode, got, c.want)
		}
	}
}

// TestCollapseKeepsNonAsciiSpace confirms collapse touches only #x20:
// a non-breaking space (U+00A0) is not collapsed or trimmed.
func TestCollapseKeepsNonAsciiSpace(t *testing.T) {
	const nbsp = " "
	got := normalizeWhiteSpace(nbsp+"a"+nbsp, collapseWS)
	if got != nbsp+"a"+nbsp {
		t.Errorf("collapse altered non-#x20 whitespace: got %q", got)
	}
}

// TestWhiteSpaceOf pins the per-type modes resolved from builtin.Types
// (§4.3.6): decimal/boolean collapse, string preserve.
func TestWhiteSpaceOf(t *testing.T) {
	cases := map[string]whiteSpace{
		"decimal": collapseWS,
		"boolean": collapseWS,
		"string":  preserveWS,
	}
	for local, want := range cases {
		if got := whiteSpaceOf(local); got != want {
			t.Errorf("whiteSpaceOf(%q) = %d, want %d", local, got, want)
		}
	}
}

// TestNormalizeForParseThenParse demonstrates the whiteSpace stage end-to-end on
// genuinely un-normalized raw literals for all three cohort types: normalize per
// §4.3.6, then feed the result to the existing Parse (which expects normalized
// input). collapse makes "  42  " and " true " parseable; preserve leaves a
// padded string exactly as written.
func TestNormalizeForParseThenParse(t *testing.T) {
	// decimal: collapse (fixed). Raw "  42  " normalizes to "42", which parses.
	if got := normalizeForParse("decimal", "  42  "); got != "42" {
		t.Fatalf("normalizeForParse(decimal, %q) = %q, want %q", "  42  ", got, "42")
	}
	if _, err := parseDecimal(normalizeForParse("decimal", "  42  "), nil); err != nil {
		t.Errorf("parseDecimal after normalize: unexpected error %v", err)
	}
	// Raw decimal with an interior tab collapses to a single run then the point.
	if got := normalizeForParse("decimal", "\t-1.5\n"); got != "-1.5" {
		t.Errorf("normalizeForParse(decimal, %q) = %q, want %q", "\t-1.5\n", got, "-1.5")
	}

	// boolean: collapse (fixed). Raw " true " normalizes to "true", which parses.
	if got := normalizeForParse("boolean", " true "); got != "true" {
		t.Fatalf("normalizeForParse(boolean, %q) = %q, want %q", " true ", got, "true")
	}
	if _, err := parseBoolean(normalizeForParse("boolean", " true "), nil); err != nil {
		t.Errorf("parseBoolean after normalize: unexpected error %v", err)
	}

	// string: preserve (not fixed). Whitespace is significant, so a padded
	// literal is returned unchanged and parses to itself.
	if got := normalizeForParse("string", " padded "); got != " padded " {
		t.Fatalf("normalizeForParse(string, %q) = %q, want identity", " padded ", got)
	}
	v, err := parseString(normalizeForParse("string", " padded "), nil)
	if err != nil {
		t.Fatalf("parseString after normalize: unexpected error %v", err)
	}
	if s, err := canonicalString(v); err != nil || s != " padded " {
		t.Errorf("string round-trip after normalize = (%q,%v), want (%q,nil)", s, err, " padded ")
	}
}

// TestNormalizeWhiteSpaceInvalidModePanics confirms the zero-value sentinel is a
// caught bug, not a silent identity.
func TestNormalizeWhiteSpaceInvalidModePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("normalizeWhiteSpace(zero mode): want panic, got none")
		}
	}()
	_ = normalizeWhiteSpace("x", whiteSpace(0))
}
