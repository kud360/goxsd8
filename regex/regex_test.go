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
		{"unsupported-name-escape", `a\i`, FlavorXSD, "", ruleXSDPattern},
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
