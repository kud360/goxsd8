package regex

import (
	"regexp"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// mustTranslate fails the test if translation errors; it returns the RE2 source.
func mustTranslate(t *testing.T, pat string, fl Flavor, flags string) string {
	t.Helper()
	out, err := Translate(pat, fl, flags)
	if err != nil {
		t.Fatalf("Translate(%q, %v, %q) unexpected error: %v", pat, fl, flags, err)
	}
	return out
}

// mustCompile compiles a translated pattern, failing on a bad RE2 result — the
// translator must never emit source Go's regexp rejects.
func mustCompile(t *testing.T, src string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(src)
	if err != nil {
		t.Fatalf("regexp.Compile(%q) failed: %v", src, err)
	}
	return re
}

func TestAnchoringDiffersByFlavor(t *testing.T) {
	xsd := mustTranslate(t, "abc", FlavorXSD, "")
	if !strings.HasPrefix(xsd, `\A(?:`) || !strings.HasSuffix(xsd, `)\z`) {
		t.Fatalf("XSD pattern not implicitly anchored: %q", xsd)
	}
	// XSD is whole-value anchored: a substring must not match.
	if mustCompile(t, xsd).MatchString("xabcx") {
		t.Fatalf("XSD %q matched a superstring; anchoring lost", xsd)
	}
	if !mustCompile(t, xsd).MatchString("abc") {
		t.Fatalf("XSD %q failed to match the exact value", xsd)
	}

	fo := mustTranslate(t, "abc", FlavorFO, "")
	if strings.Contains(fo, `\A`) || strings.Contains(fo, `\z`) {
		t.Fatalf("FO pattern was anchored: %q", fo)
	}
	// FO is unanchored: a substring match succeeds.
	if !mustCompile(t, fo).MatchString("xabcx") {
		t.Fatalf("FO %q did not match a substring; should be unanchored", fo)
	}
}

func TestCaretDollarLiteralInXSDAnchorInFO(t *testing.T) {
	// XSD: ^ and $ are literal characters.
	xsd := mustCompile(t, mustTranslate(t, "^a$", FlavorXSD, ""))
	if !xsd.MatchString("^a$") {
		t.Fatalf("XSD ^a$ should match the literal string \"^a$\"")
	}
	if xsd.MatchString("a") {
		t.Fatalf("XSD ^a$ should not match \"a\" (^/$ are literal, not anchors)")
	}

	// FO: ^ and $ are real anchors around the whole string by default.
	fo := mustCompile(t, mustTranslate(t, "^a$", FlavorFO, ""))
	if !fo.MatchString("a") {
		t.Fatalf("FO ^a$ should anchor-match \"a\"")
	}
	if fo.MatchString("ba") {
		t.Fatalf("FO ^a$ should not match \"ba\"")
	}
}

func TestGroupsCaptureOnlyInFO(t *testing.T) {
	xsd := mustCompile(t, mustTranslate(t, "(a)(b)", FlavorXSD, ""))
	if got := xsd.NumSubexp(); got != 0 {
		t.Fatalf("XSD groups should be non-capturing; NumSubexp = %d, want 0", got)
	}
	fo := mustCompile(t, mustTranslate(t, "(a)(b)", FlavorFO, ""))
	if got := fo.NumSubexp(); got != 2 {
		t.Fatalf("FO groups should capture in paren order; NumSubexp = %d, want 2", got)
	}
	m := fo.FindStringSubmatch("ab")
	if len(m) != 3 || m[1] != "a" || m[2] != "b" {
		t.Fatalf("FO capture order wrong: %#v", m)
	}
}

func TestCharClassSubtraction(t *testing.T) {
	re := mustCompile(t, mustTranslate(t, "[a-z-[m]]", FlavorXSD, ""))
	for _, c := range []string{"a", "l", "n", "z"} {
		if !re.MatchString(c) {
			t.Fatalf("[a-z-[m]] should match %q", c)
		}
	}
	if re.MatchString("m") {
		t.Fatalf("[a-z-[m]] must not match the subtracted %q", "m")
	}
	// Nested subtraction: [a-z-[a-c-[b]]] == {a-z} minus ({a,c}) == keeps b.
	re2 := mustCompile(t, mustTranslate(t, "[a-z-[a-c-[b]]]", FlavorXSD, ""))
	if !re2.MatchString("b") || re2.MatchString("a") || re2.MatchString("c") {
		t.Fatalf("nested subtraction wrong: b=%v a=%v c=%v", re2.MatchString("b"), re2.MatchString("a"), re2.MatchString("c"))
	}
}

func TestUnicodeBlockEscapes(t *testing.T) {
	basic := mustCompile(t, mustTranslate(t, `\p{IsBasicLatin}+`, FlavorXSD, ""))
	if !basic.MatchString("Az09") {
		t.Fatalf("IsBasicLatin should match ASCII")
	}
	if basic.MatchString("é") {
		t.Fatalf("IsBasicLatin should not match a Latin-1 Supplement character")
	}
	sup := mustCompile(t, mustTranslate(t, `\p{IsLatin-1Supplement}`, FlavorXSD, ""))
	if !sup.MatchString("é") {
		t.Fatalf("IsLatin-1Supplement should match é")
	}
	if sup.MatchString("A") {
		t.Fatalf("IsLatin-1Supplement should not match ASCII 'A'")
	}
	// Complement block escape.
	notBasic := mustCompile(t, mustTranslate(t, `\P{IsBasicLatin}`, FlavorXSD, ""))
	if notBasic.MatchString("A") || !notBasic.MatchString("é") {
		t.Fatalf("\\P{IsBasicLatin} complement wrong")
	}
	// Whitespace/underbar normalization of the block name (Datatypes G.4.2.3).
	if _, err := Translate(`\p{IsBasic_Latin}`, FlavorXSD, ""); err != nil {
		t.Fatalf("normalized block name should be recognized: %v", err)
	}
}

// TestNameCharEscapes covers the XML name-character multi-character escapes
// \i (NameStartChar), \c (NameChar) and their complements \I/\C (Datatypes
// §G.4.2.5), both standalone and as the base of a character-class subtraction —
// the exact constructs the intrinsic pattern facets of xs:Name (\i\c*),
// xs:NMTOKEN (\c+) and xs:NCName ([\i-[:]][\c-[:]]*) require.
func TestNameCharEscapes(t *testing.T) {
	// \i is NameStartChar: a letter/'_'/':' but not a digit or '-'.
	name := mustCompile(t, mustTranslate(t, `\i\c*`, FlavorXSD, ""))
	for _, ok := range []string{"a", "abc", "a:b", "_x", "a-b.c9"} {
		if !name.MatchString(ok) {
			t.Errorf(`\i\c* should match %q`, ok)
		}
	}
	for _, bad := range []string{"1abc", "-a", ".x", ""} {
		if name.MatchString(bad) {
			t.Errorf(`\i\c* should not match %q`, bad)
		}
	}

	// \c is NameChar: also matches leading digits and '-'.
	nmtoken := mustCompile(t, mustTranslate(t, `\c+`, FlavorXSD, ""))
	for _, ok := range []string{"1abc", "-x", "a.b:c", "9"} {
		if !nmtoken.MatchString(ok) {
			t.Errorf(`\c+ should match %q`, ok)
		}
	}
	for _, bad := range []string{"a b", "a/b", ""} {
		if nmtoken.MatchString(bad) {
			t.Errorf(`\c+ should not match %q`, bad)
		}
	}

	// NCName's own pattern: \i/\c minus the colon, via class subtraction.
	ncname := mustCompile(t, mustTranslate(t, `[\i-[:]][\c-[:]]*`, FlavorXSD, ""))
	for _, ok := range []string{"abc", "a-b.c", "_x9"} {
		if !ncname.MatchString(ok) {
			t.Errorf(`NCName pattern should match %q`, ok)
		}
	}
	for _, bad := range []string{"a:b", ":ab", "ab:", "1ab"} {
		if ncname.MatchString(bad) {
			t.Errorf(`NCName pattern should not match %q`, bad)
		}
	}

	// Complements \I/\C match exactly what \i/\c reject.
	notStart := mustCompile(t, mustTranslate(t, `\I`, FlavorXSD, ""))
	if notStart.MatchString("a") || !notStart.MatchString("1") {
		t.Errorf(`\I should reject a name-start char and accept a digit`)
	}
	notChar := mustCompile(t, mustTranslate(t, `\C`, FlavorXSD, ""))
	if notChar.MatchString("9") || !notChar.MatchString(" ") {
		t.Errorf(`\C should reject a name char and accept a space`)
	}
}

func TestFlagsHonoredInFO(t *testing.T) {
	// s: dot-all lets . match a newline.
	dotAll := mustCompile(t, mustTranslate(t, "a.b", FlavorFO, "s"))
	if !dotAll.MatchString("a\nb") {
		t.Fatalf("s flag should make . match newline")
	}
	if plain := mustCompile(t, mustTranslate(t, "a.b", FlavorFO, "")); plain.MatchString("a\nb") {
		t.Fatalf("without s flag, . must not match newline")
	}
	// i: case-insensitive.
	if ci := mustCompile(t, mustTranslate(t, "abc", FlavorFO, "i")); !ci.MatchString("ABC") {
		t.Fatalf("i flag should match case-insensitively")
	}
	// m: multi-line anchors match at line boundaries.
	ml := mustCompile(t, mustTranslate(t, "^b$", FlavorFO, "m"))
	if !ml.MatchString("a\nb\nc") {
		t.Fatalf("m flag should let ^b$ match a middle line")
	}
	// x: insignificant whitespace stripped outside classes, kept inside.
	if xf := mustCompile(t, mustTranslate(t, "a b c", FlavorFO, "x")); !xf.MatchString("abc") {
		t.Fatalf("x flag should strip whitespace outside classes")
	}
	if xc := mustCompile(t, mustTranslate(t, "a[ ]b", FlavorFO, "x")); !xc.MatchString("a b") {
		t.Fatalf("x flag must preserve whitespace inside a character class")
	}
}

func TestErrorsSurfaceNeverSilentlyAccepted(t *testing.T) {
	cases := []struct {
		name   string
		pat    string
		flavor Flavor
		flags  string
		rule   xsderr.Rule
	}{
		{"fo-rejects-q-flag", "abc", FlavorFO, "q", ruleFOFlags},
		{"fo-rejects-unknown-flag", "abc", FlavorFO, "z", ruleFOFlags},
		{"fo-backreference", `(a)\1`, FlavorFO, "", ruleFOPattern},
		{"xsd-backreference", `(a)\1`, FlavorXSD, "", ruleXSDPattern},
		{"unknown-block", `\p{IsNoSuchBlock}`, FlavorXSD, "", ruleXSDPattern},
		{"unknown-category", `\p{Xy}`, FlavorFO, "", ruleFOPattern},
		{"counted-repeat-over-limit", "a{1001}", FlavorXSD, "", ruleXSDPattern},
		{"counted-repeat-range-over-limit", "a{1,2000}", FlavorFO, "", ruleFOPattern},
		{"trailing-backslash", `abc\`, FlavorXSD, "", ruleXSDPattern},
		{"unclosed-group", "(abc", FlavorFO, "", ruleFOPattern},
		{"unclosed-class", "[abc", FlavorXSD, "", ruleXSDPattern},
		{"empty-negated-class", "[^]", FlavorXSD, "", ruleXSDPattern},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Translate(tc.pat, tc.flavor, tc.flags)
			if err == nil {
				t.Fatalf("Translate(%q) succeeded; want error", tc.pat)
			}
			rule, ok := xsderr.RuleOf(err)
			if !ok {
				t.Fatalf("error is not an *xsderr.Error: %v", err)
			}
			if rule != tc.rule {
				t.Fatalf("rule = %q, want %q (err: %v)", rule, tc.rule, err)
			}
		})
	}
}

func TestCountedRepeatBoundaryAccepted(t *testing.T) {
	// Exactly 1000 is the largest RE2 accepts; it must translate and compile.
	mustCompile(t, mustTranslate(t, "a{1000}", FlavorXSD, ""))
	mustCompile(t, mustTranslate(t, "a{0,1000}", FlavorFO, ""))
}

func TestErrorsCarryByteOffset(t *testing.T) {
	_, err := Translate(`abcde\9`, FlavorFO, "")
	if err == nil {
		t.Fatal("expected error for back-reference")
	}
	// The offending construct begins at byte offset 5 (the backslash).
	if !strings.Contains(err.Error(), "offset 5") {
		t.Fatalf("error should report the byte offset of the construct: %v", err)
	}
}

func TestDeterministic(t *testing.T) {
	pats := []string{`[\w-[aeiou]]`, `\p{IsBasicLatin}`, `(a|b)*c{2,5}`, `[a-z-[m-p]]`}
	for _, p := range pats {
		first := mustTranslate(t, p, FlavorXSD, "")
		for i := 0; i < 5; i++ {
			if got := mustTranslate(t, p, FlavorXSD, ""); got != first {
				t.Fatalf("Translate(%q) not deterministic: %q vs %q", p, got, first)
			}
		}
	}
}

// mustPanic runs fn and fails unless it panics; caller-contract violations are
// programming errors, not returned *xsderr.Errors.
func mustPanic(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic, got none")
		}
	}()
	fn()
}

func TestInvalidFlavorPanics(t *testing.T) {
	// The zero Flavor and any out-of-range cast reach the switch default.
	mustPanic(t, func() { _, _ = Translate("abc", Flavor(0), "") })
	mustPanic(t, func() { _, _ = Translate("abc", Flavor(99), "") })
}

func TestXSDFlagsPanic(t *testing.T) {
	// Passing flags to the flagless XSD flavor is caller misuse, not a pattern
	// error, so it panics rather than returning a src-pattern-value error.
	mustPanic(t, func() { _, _ = Translate("abc", FlavorXSD, "m") })
	mustPanic(t, func() { _, _ = Translate("abc", FlavorXSD, "x") })
}

func TestMultiCharEscapes(t *testing.T) {
	// \d, \s, \w outside and inside classes.
	dre := mustCompile(t, mustTranslate(t, `\d+`, FlavorXSD, ""))
	if !dre.MatchString("2026") || dre.MatchString("x") {
		t.Fatalf("\\d translation wrong")
	}
	wre := mustCompile(t, mustTranslate(t, `\w`, FlavorXSD, ""))
	if !wre.MatchString("a") || wre.MatchString(" ") || wre.MatchString(".") {
		t.Fatalf("\\w translation wrong (must exclude punctuation and separators)")
	}
	Wre := mustCompile(t, mustTranslate(t, `\W`, FlavorXSD, ""))
	if Wre.MatchString("a") || !Wre.MatchString(".") {
		t.Fatalf("\\W translation wrong (must be the complement of \\w)")
	}
	sre := mustCompile(t, mustTranslate(t, `[\s]`, FlavorXSD, ""))
	if !sre.MatchString(" ") || !sre.MatchString("\t") || sre.MatchString("a") {
		t.Fatalf("\\s inside a class wrong")
	}
}

func TestPropertyClassKeepsStrideGaps(t *testing.T) {
	// Inside a class a \p{...} category is materialized through addTable, which
	// adds a contiguous unicode.RangeTable interval whole and walks only a wider
	// stride. Lu's Latin Extended-A run is stride 2 — U+0100, U+0102, U+0104 are
	// Lu and the odd code points between them are Ll — so adding that interval
	// whole would silently widen \p{Lu} to every other letter it excludes, and
	// accepting a literal the pattern rejects is the direction propSet's comment
	// forbids (PRINCIPLES 20).
	re := mustCompile(t, mustTranslate(t, `[\p{Lu}]`, FlavorXSD, ""))
	for _, in := range []string{"Ā", "Ă", "A"} {
		if !re.MatchString(in) {
			t.Errorf("[\\p{Lu}] must match %q", in)
		}
	}
	for _, out := range []string{"ā", "ă", "a"} {
		if re.MatchString(out) {
			t.Errorf("[\\p{Lu}] must not match %q — the stride-2 gap was filled in", out)
		}
	}
}

// TestCheckSyntaxRejectsAppendixGDefects pins the half of CheckSyntax's
// contract that a schema-construction pass rejects on: a pattern Appendix G's
// grammar or its disambiguation rules exclude. The first two are §G.4.1's
// hyphen cases (the spec names "[--z]" itself as excluded); the third is §G.2's
// quantity grammar, which has no omitted-lower-bound form and says so in a Note.
func TestCheckSyntaxRejectsAppendixGDefects(t *testing.T) {
	cases := []struct {
		pattern string
		detail  string
	}{
		{`[--z]*`, "cannot start a character range"},
		{`[!--]*`, "cannot end a character range"},
		{`[0-9]{,5}`, "expected a repetition count"},
		{`[0-9]{5,2}`, "out of order"},
		{`(abc`, "unclosed group"},
		{`[abc`, "unclosed character class"},
		{`abc\`, "trailing backslash"},
		// An unrecognized Unicode CATEGORY is a syntax defect, not a gap in this
		// module: Appendix G §G.4.2.2 enumerates the category names, and one
		// outside that enumeration names nothing. It must NOT take the
		// unsupported-block exit below.
		{`\p{Zork}`, "unrecognized Unicode category"},
		{`[\p{Zork}]`, "unrecognized Unicode category"},
		// IsBlock ::= 'Is' [a-zA-Z0-9#x2D]+ (production [96]) admits one or more
		// hyphens, digits and Basic Latin letters and nothing else, so each of
		// these names no block and matches no charProp. They must take the defect
		// path, not the unsupported-block one, even though both run through
		// blockSet — so each detail pins the offending name, not just the kind of
		// defect. "\p{Is-}" below is the boundary on the other side: a hyphen IS
		// production [96] material, so it stays the unsupported-block gap.
		{`\p{Is}`, `malformed Unicode block name in \p{Is}`},
		{`\p{IsThai$}`, `malformed Unicode block name in \p{IsThai$}`},
		{`[\p{IsThai$}]`, `malformed Unicode block name in \p{IsThai$}`},
		{`\p{Is.}`, `malformed Unicode block name in \p{Is.}`},
		// The message quotes the name the AUTHOR wrote, not the normalized one the
		// production is applied to: this name loses its space to §G.4.2.3 before
		// the '$' disqualifies it, and the report still has to be findable in the
		// schema document.
		{`\p{IsThai Extra$}`, `malformed Unicode block name in \p{IsThai Extra$}`},
	}
	for _, c := range cases {
		t.Run(c.pattern, func(t *testing.T) {
			err := CheckSyntax(c.pattern, FlavorXSD, "")
			if err == nil {
				t.Fatalf("CheckSyntax(%q) = nil, want a src-pattern-value error", c.pattern)
			}
			rule, ok := xsderr.RuleOf(err)
			if !ok || rule != ruleXSDPattern {
				t.Fatalf("rule = %q (ok=%v), want %q", rule, ok, ruleXSDPattern)
			}
			if !strings.Contains(err.Error(), c.detail) {
				t.Errorf("message %q does not name the defect %q", err.Error(), c.detail)
			}
		})
	}
}

// TestCheckSyntaxPassesUnsupportedBlocks pins the other half, the half that is
// CheckSyntax's whole reason to exist: a block name outside class.go's curated
// unicodeBlocks table is a gap in THIS module (GAP(regex), #1473), not a defect
// in the pattern, so CheckSyntax reports nothing — while Translate, whose caller
// needs the compiled regex and cannot proceed, keeps failing on it unchanged.
func TestCheckSyntaxPassesUnsupportedBlocks(t *testing.T) {
	// Both the standalone atom path (regex.go's atomCategoryEscape) and the
	// inside-a-class path (classparse.go's parseClassEscape) reach blockSet, and
	// the sentinel has to survive each one's error wrapping. "\p{Is-}" and
	// "\p{IsThai Extra}" are the boundary against the defect cases in
	// TestCheckSyntaxRejectsAppendixGDefects: a hyphen is production [96]
	// material outright, and a space is stripped by §G.4.2.3 normalization before
	// the production is applied, so neither name is malformed — each is merely
	// absent from the table.
	for _, pat := range []string{`\p{IsThai}*`, `[\p{IsOgham}]+`, `[\p{IsRunic}a-z]`, `\P{IsTibetan}`, `\p{Is-}`, `\p{IsThai Extra}`} {
		t.Run(pat, func(t *testing.T) {
			if err := CheckSyntax(pat, FlavorXSD, ""); err != nil {
				t.Fatalf("CheckSyntax(%q) = %v, want nil: the block table is this module's gap, not the pattern's defect", pat, err)
			}
			if _, err := Translate(pat, FlavorXSD, ""); err == nil {
				t.Fatalf("Translate(%q) = nil error — the premise of this test is gone; Translate must keep surfacing the gap", pat)
			}
		})
	}
}

// TestCheckSyntaxPassesCountedRepeatOverLimit pins the sentinel's second
// producer. Production [71] is QuantExact ::= [0-9]+ and [69] is quantRange ::=
// QuantExact ',' QuantExact, neither capped, so each of these is a regExp and
// the 1000 that stops it is maxRepeat's (GAP(regex), #1474) — the same
// classification the block table gets, reached through a different path, and
// the boundary case {1000} must stay translatable on the other side of it.
func TestCheckSyntaxPassesCountedRepeatOverLimit(t *testing.T) {
	for _, pat := range []string{"a{1001}", "a{0,2000}", "a{1001,2000}", "[0-9]{5000}", "(ab){2,1500}"} {
		t.Run(pat, func(t *testing.T) {
			if err := CheckSyntax(pat, FlavorXSD, ""); err != nil {
				t.Fatalf("CheckSyntax(%q) = %v, want nil: the RE2 repeat ceiling is this module's gap, not the pattern's defect", pat, err)
			}
			if _, err := Translate(pat, FlavorXSD, ""); err == nil {
				t.Fatalf("Translate(%q) = nil error — the premise of this test is gone; Translate must keep surfacing the gap", pat)
			}
		})
	}
	// quantity reaches the ceiling before it compares the two bounds, so a range
	// that is BOTH over the ceiling and out of order — §G.2's piece table admits
	// S{n,m} only "for … non-negative integers n, m such that n <= m" — is passed
	// over as the gap rather than reported as the defect. That is the masking
	// CheckSyntax documents, in the direction it documents; below the ceiling the
	// defect is still reported.
	if err := CheckSyntax("a{2000,1500}", FlavorXSD, ""); err != nil {
		t.Errorf(`CheckSyntax("a{2000,1500}") = %v, want nil: the ceiling is reached first`, err)
	}
	if err := CheckSyntax("a{1000,999}", FlavorXSD, ""); err == nil {
		t.Error(`CheckSyntax("a{1000,999}") = nil, want the out-of-order defect`)
	}
}

// TestCheckSyntaxAcceptsValidPatterns guards the direction a too-wide
// classification would break: an ordinary pattern must report nothing.
func TestCheckSyntaxAcceptsValidPatterns(t *testing.T) {
	for _, pat := range []string{`[0-9]{1,5}`, `[a-z-[m]]`, `\p{Lu}+`, `\i\c*`, `a|b|`, `[-a-z]`, `[a-z-]`, `a{1000}`, `a{0,1000}`} {
		if err := CheckSyntax(pat, FlavorXSD, ""); err != nil {
			t.Errorf("CheckSyntax(%q) = %v, want nil", pat, err)
		}
	}
}
