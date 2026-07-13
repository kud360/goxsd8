package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// typeVectors is the generator's view of one builtin type's vectors. It mirrors
// the (unexported) schema in value/backendtest; the generator cannot import
// those private types, so it carries its own string-only copy and emits the
// literal.
type typeVectors struct {
	Local        string // builtin local name (namespace is the fixed XSD one)
	Valid        []roundtrip
	Invalid      []string
	NarrowReject []string
	// ApplicableFacets are the type's applicable constraining facets in spec
	// order (cos-applicable-facets, §4.1.5), read from the shared builtin parser.
	ApplicableFacets []string
}

// roundtrip is a valid lexical paired with the canonical form its value renders.
type roundtrip struct {
	Lexical   string
	Canonical string
}

// litRE matches a spec literal written as '`X`' (apostrophe, backtick, token,
// backtick, apostrophe), the form the Datatypes spec uses for booleanRep
// literals and for the operands of the boolean lexical/canonical mappings.
var litRE = regexp.MustCompile("'`([^`]+)`'")

// backtickRE matches a bare `X` backtick span, the form the decimal Lexical
// Mapping section uses for the lexical-space regular expression.
var backtickRE = regexp.MustCompile("`([^`]+)`")

// parseBoolean derives the boolean vectors from the Datatypes spec, purely from
// the normative mapping definitions (§3.3.2.2):
//
//   - the four lexicals from the booleanRep production (nt-booleanRep);
//   - the subset mapping to true from booleanLexicalMap (f-booleanLexmap);
//   - the canonical forms of true/false from booleanCanonicalMap
//     (f-booleanCanmap).
//
// The valid round-trips are then composed: each lexical maps to true or false
// per the lexical map, and canonicalizes per the canonical map. Invalid
// lexicals are the complement of nt-booleanRep, which cvc-datatype-valid
// (§4.1.4) rejects; a deterministic near-miss sample is derived mechanically
// from the valid literals (case variants of the alphabetic literals, the empty
// string, and a decimal digit outside the numeric literals).
func parseBoolean(spec string) (typeVectors, error) {
	lexicals, err := booleanLexicals(spec)
	if err != nil {
		return typeVectors{}, err
	}
	trueSet, err := booleanTrueSet(spec, lexicals)
	if err != nil {
		return typeVectors{}, err
	}
	canonTrue, canonFalse, err := booleanCanonical(spec, lexicals)
	if err != nil {
		return typeVectors{}, err
	}

	var valid []roundtrip
	for _, lex := range lexicals {
		canon := canonFalse
		if trueSet[lex] {
			canon = canonTrue
		}
		valid = append(valid, roundtrip{Lexical: lex, Canonical: canon})
	}

	return typeVectors{
		Local:   "boolean",
		Valid:   valid,
		Invalid: booleanInvalids(lexicals),
	}, nil
}

// booleanLexicals extracts the four lexicals of the booleanRep production.
func booleanLexicals(spec string) ([]string, error) {
	line, ok := findLine(spec, `id="nt-booleanRep"`)
	if !ok {
		return nil, fmt.Errorf("booleanRep production (nt-booleanRep) not found")
	}
	lexicals := literalsIn(line)
	if len(lexicals) != 4 {
		return nil, fmt.Errorf("booleanRep: found %d lexicals %q, want 4", len(lexicals), lexicals)
	}
	return lexicals, nil
}

// booleanTrueSet extracts the lexicals booleanLexicalMap maps to true — the
// operands of its "***true***when …" clause, up to the "***false***" clause.
func booleanTrueSet(spec string, lexicals []string) (map[string]bool, error) {
	sec, err := section(spec, `id="f-booleanLexmap"`, `id="f-stringCanmap"`)
	if err != nil {
		return nil, fmt.Errorf("booleanLexicalMap: %w", err)
	}
	clause, err := between(sec, "***true***", "***false***")
	if err != nil {
		return nil, fmt.Errorf("booleanLexicalMap true clause: %w", err)
	}
	trueLits := literalsIn(clause)
	if len(trueLits) == 0 {
		return nil, fmt.Errorf("booleanLexicalMap: no lexicals map to true")
	}
	set := make(map[string]bool, len(trueLits))
	for _, l := range trueLits {
		if !contains(lexicals, l) {
			return nil, fmt.Errorf("booleanLexicalMap: true-lexical %q not in lexical space %q", l, lexicals)
		}
		set[l] = true
	}
	return set, nil
}

// booleanCanonical extracts the canonical forms booleanCanonicalMap assigns to
// the true and false values — the two literals in its two bullets, in order.
func booleanCanonical(spec string, lexicals []string) (canonTrue, canonFalse string, err error) {
	sec, err := section(spec, `id="f-booleanCanmap"`, "####")
	if err != nil {
		return "", "", fmt.Errorf("booleanCanonicalMap: %w", err)
	}
	lits := literalsIn(sec)
	if len(lits) != 2 {
		return "", "", fmt.Errorf("booleanCanonicalMap: found %d canonical literals %q, want 2", len(lits), lits)
	}
	canonTrue, canonFalse = lits[0], lits[1]
	if !contains(lexicals, canonTrue) || !contains(lexicals, canonFalse) {
		return "", "", fmt.Errorf("booleanCanonicalMap: canonical forms %q/%q not in lexical space %q", canonTrue, canonFalse, lexicals)
	}
	if canonTrue == canonFalse {
		return "", "", fmt.Errorf("booleanCanonicalMap: true and false share canonical form %q", canonTrue)
	}
	return canonTrue, canonFalse, nil
}

// booleanInvalids derives a deterministic near-miss sample of lexicals outside
// nt-booleanRep (rejected by cvc-datatype-valid): for each alphabetic literal
// its capitalized and upper-cased variants, then the empty string, then the
// first decimal digit not among the numeric literals.
func booleanInvalids(lexicals []string) []string {
	var out []string
	seen := map[string]bool{}
	add := func(s string) {
		if contains(lexicals, s) || seen[s] {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, lex := range lexicals {
		if !hasLetter(lex) {
			continue
		}
		add(capitalize(lex))
		add(strings.ToUpper(lex))
	}
	add("")
	for d := '0'; d <= '9'; d++ {
		if !contains(lexicals, string(d)) {
			add(string(d))
			break
		}
	}
	return out
}

// parseString derives the string vectors. f-stringLexmap and f-stringCanmap are
// the identity on the whole domain (§3.3.1.2), and every finite Char* sequence
// is in the lexical space (nt-stringRep) — so there is NO invalid string lexical
// and the Invalid slot is deliberately left empty; inventing "invalid strings"
// would contradict the lexical space. string's space is unbounded, so unlike
// boolean/decimal there is no finite spec production to enumerate: the valid
// sample is a small deterministic set of representative literals — the empty
// string, ASCII, a multi-byte (combining) codepoint and an astral codepoint —
// each round-tripping to itself under the identity canonical mapping. The spec
// anchors are consulted only to fail loud if the identity mappings move.
func parseString(spec string) (typeVectors, error) {
	for _, anchor := range []string{`id="f-stringLexmap"`, `id="f-stringCanmap"`} {
		if !strings.Contains(spec, anchor) {
			return typeVectors{}, fmt.Errorf("string identity mapping anchor %q not found", anchor)
		}
	}
	sample := []string{"", "abc", "café", "𝔘nicode"}
	valid := make([]roundtrip, 0, len(sample))
	for _, s := range sample {
		valid = append(valid, roundtrip{Lexical: s, Canonical: s})
	}
	return typeVectors{Local: "string", Valid: valid}, nil
}

// parseDecimal derives the decimal vectors from the Datatypes spec (§3.3.3):
//
//   - the valid lexicals are the worked examples in the decimal Lexical Mapping
//     prose (decimal-lexical-representation, nt-decimalRep), each paired with the
//     canonical form decimalCanonicalMap (f-decimalCanmap, §3.3.3.2) assigns it,
//     computed by decimalCanonical here so the vectors are an INDEPENDENT oracle
//     rather than an echo of the backend under test;
//   - the lexical space is exactly the regular expression the same production
//     gives; the invalid sample is a deterministic set of near-misses that regex
//     rejects (an exponent form, which the prose explicitly excludes; a bare
//     sign; a bare point; the empty string), each verified against the extracted
//     regex so a spec change widening the space would drop it rather than let a
//     now-valid lexical masquerade as invalid.
func parseDecimal(spec string) (typeVectors, error) {
	sec, err := section(spec, `id="decimal-lexical-representation"`, `id="decimal-facets"`)
	if err != nil {
		return typeVectors{}, fmt.Errorf("decimal lexical mapping: %w", err)
	}
	lexicals := literalsIn(sec)
	if len(lexicals) == 0 {
		return typeVectors{}, fmt.Errorf("decimal lexical mapping: no example lexicals found")
	}
	re, err := decimalLexicalRegex(sec)
	if err != nil {
		return typeVectors{}, err
	}

	valid := make([]roundtrip, 0, len(lexicals))
	for _, lex := range lexicals {
		if !re.MatchString(lex) {
			return typeVectors{}, fmt.Errorf("decimal: example lexical %q does not match its own production regex", lex)
		}
		valid = append(valid, roundtrip{Lexical: lex, Canonical: decimalCanonical(lex)})
	}

	return typeVectors{
		Local:   "decimal",
		Valid:   valid,
		Invalid: decimalInvalids(re, lexicals),
	}, nil
}

// decimalLexicalRegex extracts the decimal lexical-space regular expression from
// the production prose and returns it anchored (^…$). The spec writes it as a
// bare `…` backtick span containing [0-9] and \. ; it is valid Go regexp syntax.
func decimalLexicalRegex(sec string) (*regexp.Regexp, error) {
	for _, line := range strings.Split(sec, "\n") {
		if !strings.Contains(line, `[0-9]+`) || !strings.Contains(line, `\.[0-9]`) {
			continue
		}
		m := backtickRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("decimal: regex line has no backtick span: %q", line)
		}
		re, err := regexp.Compile("^(?:" + m[1] + ")$")
		if err != nil {
			return nil, fmt.Errorf("decimal: compiling extracted lexical regex %q: %w", m[1], err)
		}
		return re, nil
	}
	return nil, fmt.Errorf("decimal: lexical-space regular expression not found")
}

// decimalCanonical implements decimalCanonicalMap (f-decimalCanmap, §3.3.3.2):
// drop a '+' sign; an integer value has no point or fractional part; otherwise a
// mandatory point with at least one digit on each side and no superfluous
// leading/trailing zeros; no sign on zero. Its input is a lexical the production
// regex already accepted, so body[0] is a sign or digit or '.'.
func decimalCanonical(lexical string) string {
	neg := false
	body := lexical
	switch body[0] {
	case '+':
		body = body[1:]
	case '-':
		neg = true
		body = body[1:]
	}
	intPart, fracPart := body, ""
	if i := strings.IndexByte(body, '.'); i >= 0 {
		intPart, fracPart = body[:i], body[i+1:]
	}
	intPart = strings.TrimLeft(intPart, "0")
	fracPart = strings.TrimRight(fracPart, "0")
	if intPart == "" {
		intPart = "0"
	}
	sign := ""
	if neg && (intPart != "0" || fracPart != "") {
		sign = "-"
	}
	if fracPart == "" {
		return sign + intPart
	}
	return sign + intPart + "." + fracPart
}

// decimalInvalids derives a deterministic near-miss sample of lexicals outside
// the decimal production, keeping only those the extracted regex rejects: an
// exponent form built from the first valid example (the prose explicitly
// excludes exponents), a bare sign and a bare point (the production requires at
// least one digit), and the empty string.
func decimalInvalids(re *regexp.Regexp, lexicals []string) []string {
	candidates := []string{lexicals[0] + "E2", "+", ".", ""}
	var out []string
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c] || re.MatchString(c) {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// parseFloating derives the float (bitSize 32) or double (bitSize 64) vectors
// from the Datatypes spec (§3.3.4/§3.3.5). float and double share one lexical
// space and one canonical algorithm; only the IEEE precision differs, carried
// here by bitSize (strconv's rounding at 32 vs 64 bits IS floatingPointRound,
// f-floatLexmap Note). The valid sample pairs the special literals (extracted
// from nt-numSpecReps) and a deterministic representative set of numerals — like
// string (§3.3.1.2) the numeric space is unbounded, so a sample exercises each
// structural feature (special values, signed zero, integer, fraction, exponent,
// sign) — each canonicalised by floatingCanonicalOf, an INDEPENDENT oracle
// implementing scientificCanonicalMap (f-sciCanFragMap), never an echo of the
// backend. Invalid near-misses are verified against the extracted regex so a
// spec change widening the space drops them rather than mislabelling them.
func parseFloating(spec, local string, bitSize int) (typeVectors, error) {
	sec, err := section(spec, `id="sec-lex-`+local+`"`, `id="`+local+`-facets"`)
	if err != nil {
		return typeVectors{}, fmt.Errorf("%s lexical mapping: %w", local, err)
	}
	re, err := floatingLexicalRegex(sec)
	if err != nil {
		return typeVectors{}, fmt.Errorf("%s: %w", local, err)
	}
	specials, err := floatingSpecials(spec)
	if err != nil {
		return typeVectors{}, fmt.Errorf("%s: %w", local, err)
	}

	numerals := []string{"0", "-0", "1", "-1", "1.5E1", "100", ".5", "-0.001", "3.14"}
	sample := append(append([]string{}, specials...), numerals...)

	valid := make([]roundtrip, 0, len(sample))
	for _, lex := range sample {
		if !re.MatchString(lex) {
			return typeVectors{}, fmt.Errorf("%s: sample lexical %q does not match its own production regex", local, lex)
		}
		canon, err := floatingCanonicalOf(lex, bitSize)
		if err != nil {
			return typeVectors{}, fmt.Errorf("%s: canonical of %q: %w", local, lex, err)
		}
		valid = append(valid, roundtrip{Lexical: lex, Canonical: canon})
	}

	return typeVectors{
		Local:   local,
		Valid:   valid,
		Invalid: floatingInvalids(re),
	}, nil
}

// floatingLexicalRegex extracts the shared float/double lexical-space regular
// expression from a lexical section and returns it anchored (^…$). The spec
// writes it as a bare `…` backtick span with a display space (" |") that must be
// removed ("after whitespace is removed from the regular expression").
func floatingLexicalRegex(sec string) (*regexp.Regexp, error) {
	for _, line := range strings.Split(sec, "\n") {
		if !strings.Contains(line, `[Ee]`) || !strings.Contains(line, `INF`) {
			continue
		}
		m := backtickRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("regex line has no backtick span: %q", line)
		}
		expr := strings.ReplaceAll(m[1], " ", "")
		re, err := regexp.Compile("^(?:" + expr + ")$")
		if err != nil {
			return nil, fmt.Errorf("compiling extracted lexical regex %q: %w", expr, err)
		}
		return re, nil
	}
	return nil, fmt.Errorf("lexical-space regular expression not found")
}

// floatingSpecials extracts the numericalSpecialRep literals (nt-numSpecReps,
// nt-minNumSpecReps) and verifies the set is exactly INF, +INF, -INF, NaN — the
// stricter special sub-grammar (no +NaN/-NaN). It fails loud if the spec moves.
func floatingSpecials(spec string) ([]string, error) {
	sec, err := section(spec, `id="nt-minNumSpecReps"`, `Lexical Mapping for Non-numerical`)
	if err != nil {
		return nil, fmt.Errorf("numericalSpecialRep production: %w", err)
	}
	lits := literalsIn(sec)
	want := map[string]bool{"INF": true, "+INF": true, "-INF": true, "NaN": true}
	if len(lits) != len(want) {
		return nil, fmt.Errorf("numericalSpecialRep: found %q, want the 4 literals INF/+INF/-INF/NaN", lits)
	}
	for _, l := range lits {
		if !want[l] {
			return nil, fmt.Errorf("numericalSpecialRep: unexpected special literal %q", l)
		}
	}
	return lits, nil
}

// floatingInvalids is a deterministic near-miss sample outside the shared
// production (rejected by cvc-datatype-valid): the signed NaN spellings the
// stricter special grammar excludes (+NaN/-NaN, whereas +INF is allowed), a
// foreign infinity spelling, a trailing-whitespace literal (whiteSpace is a
// separate stage), a dangling exponent, a double sign and a double point, and
// the empty string. Each is kept only if the extracted regex rejects it.
func floatingInvalids(re *regexp.Regexp) []string {
	candidates := []string{"+NaN", "-NaN", "Infinity", "INF ", "1.5e", "++1", "1.0.0", ""}
	var out []string
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c] || re.MatchString(c) {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// floatingCanonicalOf computes the canonical form of a float/double lexical at
// the given bitSize, the independent oracle the vectors pin. Specials map per
// specialRepCanonicalMap (f-specValCanMap); numerals parse at bitSize (strconv's
// rounding is floatingPointRound) then render via floatingCanonical. An
// out-of-range numeral (ErrRange) is valid — it yields ±INF or a signed zero.
func floatingCanonicalOf(lex string, bitSize int) (string, error) {
	switch lex {
	case "INF", "+INF":
		return "INF", nil
	case "-INF":
		return "-INF", nil
	case "NaN":
		return "NaN", nil
	}
	f, err := strconv.ParseFloat(lex, bitSize)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return "", err
	}
	return floatingCanonical(f, bitSize), nil
}

// floatingCanonical implements floatCanonicalMap/doubleCanonicalMap (§3.3.4.2/
// §3.3.5.2): the special forms and signed zeros, else scientificCanonicalMap —
// the shortest round-tripping decimal (strconv.FormatFloat precision -1) in
// scientific notation, reshaped to the spec numeral: one leading mantissa digit,
// a mandatory decimal point, uppercase E, and a minimal signless-plus exponent.
func floatingCanonical(f float64, bitSize int) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "INF"
	case math.IsInf(f, -1):
		return "-INF"
	case f == 0:
		if math.Signbit(f) {
			return "-0.0E0"
		}
		return "0.0E0"
	}
	s := strconv.FormatFloat(f, 'e', -1, bitSize)
	i := strings.IndexByte(s, 'e')
	mantissa, exp := s[:i], s[i+1:]
	if !strings.ContainsRune(mantissa, '.') {
		mantissa += ".0"
	}
	neg := exp[0] == '-'
	exp = strings.TrimLeft(exp[1:], "0")
	if exp == "" {
		exp = "0"
	}
	if neg && exp != "0" {
		exp = "-" + exp
	}
	return mantissa + "E" + exp
}

// parseHexBinary derives the hexBinary vectors from the Datatypes spec (§3.3.15):
// the lexical space is exactly the regular expression '`([0-9a-fA-F]{2})*`' the
// production gives (nt-hexBinary, §3.3.15.2), extracted here so a spec edit that
// moved it would drop a now-mismatched sample rather than mislabel it. The valid
// sample is a deterministic representative set — like string (§3.3.1.2) the space
// is unbounded — exercising the empty sequence, lowercase input, uppercase input
// and a multi-octet value; each is canonicalised by hexBinaryCanonicalOf, an
// INDEPENDENT oracle implementing f-hexBinaryCanonical (uppercase A–F, E.4.1),
// never an echo of the backend. Invalid near-misses (odd length, a non-hex digit)
// are kept only if the extracted regex rejects them.
func parseHexBinary(spec string) (typeVectors, error) {
	re, err := hexBinaryLexicalRegex(spec)
	if err != nil {
		return typeVectors{}, err
	}

	sample := []string{"", "0FB7", "0fb7", "deadBEEF", "ff"}
	valid := make([]roundtrip, 0, len(sample))
	for _, lex := range sample {
		if !re.MatchString(lex) {
			return typeVectors{}, fmt.Errorf("hexBinary: sample lexical %q does not match its own production regex", lex)
		}
		canon, err := hexBinaryCanonicalOf(lex)
		if err != nil {
			return typeVectors{}, fmt.Errorf("hexBinary: canonical of %q: %w", lex, err)
		}
		valid = append(valid, roundtrip{Lexical: lex, Canonical: canon})
	}

	return typeVectors{
		Local:   "hexBinary",
		Valid:   valid,
		Invalid: binaryInvalids(re, []string{"F", "0FB", "0G", "gg"}),
	}, nil
}

// hexBinaryLexicalRegex extracts hexBinary's lexical-space regular expression
// ('`([0-9a-fA-F]{2})*`', nt-hexBinary) from the production prose and returns it
// anchored (^…$). It matches the backtick span carrying the '{2})*' quantifier so
// the neighbouring bare hexDigit class ('[0-9a-fA-F]') is never picked instead.
func hexBinaryLexicalRegex(spec string) (*regexp.Regexp, error) {
	for _, line := range strings.Split(spec, "\n") {
		if !strings.Contains(line, `{2})*`) || !strings.Contains(line, "0-9a-fA-F") {
			continue
		}
		for _, m := range backtickRE.FindAllStringSubmatch(line, -1) {
			if !strings.Contains(m[1], `{2})*`) {
				continue
			}
			re, err := regexp.Compile("^(?:" + m[1] + ")$")
			if err != nil {
				return nil, fmt.Errorf("hexBinary: compiling extracted lexical regex %q: %w", m[1], err)
			}
			return re, nil
		}
	}
	return nil, fmt.Errorf("hexBinary: lexical-space regular expression not found")
}

// hexBinaryCanonicalOf computes the canonical form of a hexBinary lexical, the
// independent oracle the vectors pin: hexBinaryCanonical uppercases the octets'
// hex digits (E.4.1). Its input is a lexical the production regex already
// accepted, so hex.DecodeString cannot fail.
func hexBinaryCanonicalOf(lex string) (string, error) {
	octets, err := hex.DecodeString(lex)
	if err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(octets)), nil
}

// parseBase64Binary derives the base64Binary vectors from the Datatypes spec
// (§3.3.16): the lexical space is exactly the equivalent regular expression the
// production gives (nt-Base64Binary, §3.3.16.2), extracted here so its
// restricted-final-character constraint (B16char/B04char) is pinned to the spec.
// The valid sample is a deterministic representative set exercising the empty
// sequence, an unpadded quad, single-'=' (two-octet) and double-'=' (one-octet)
// padding; each is canonicalised by base64CanonicalOf, an INDEPENDENT oracle
// implementing the §3.3.16.2 encoding (the whitespace-free Base64 form). Invalid
// near-misses (a non-multiple-of-four count, a bad restricted final char under
// each padding width) are kept only if the extracted regex rejects them.
func parseBase64Binary(spec string) (typeVectors, error) {
	re, err := base64LexicalRegex(spec)
	if err != nil {
		return typeVectors{}, err
	}

	sample := []string{"", "AQID", "AQI=", "AQ==", "TWFu"}
	valid := make([]roundtrip, 0, len(sample))
	for _, lex := range sample {
		if !re.MatchString(lex) {
			return typeVectors{}, fmt.Errorf("base64Binary: sample lexical %q does not match its own production regex", lex)
		}
		canon, err := base64CanonicalOf(lex)
		if err != nil {
			return typeVectors{}, fmt.Errorf("base64Binary: canonical of %q: %w", lex, err)
		}
		valid = append(valid, roundtrip{Lexical: lex, Canonical: canon})
	}

	return typeVectors{
		Local:   "base64Binary",
		Valid:   valid,
		Invalid: binaryInvalids(re, []string{"AQI", "AQJ=", "AB==", "A==="}),
	}, nil
}

// base64LexicalRegex extracts base64Binary's equivalent regular expression
// (nt-Base64Binary, §3.3.16.2) from the production prose and returns it anchored
// (^…$). Unlike the float/decimal regexes, its single spaces are the grammar's
// inter-character #x20? and are kept verbatim; the B04char class '[AQgw]' pins the
// line so the wrong backtick span is never picked.
func base64LexicalRegex(spec string) (*regexp.Regexp, error) {
	for _, line := range strings.Split(spec, "\n") {
		if !strings.Contains(line, "A-Za-z0-9+/") || !strings.Contains(line, "){4}") {
			continue
		}
		for _, m := range backtickRE.FindAllStringSubmatch(line, -1) {
			if !strings.Contains(m[1], "){4}") {
				continue
			}
			re, err := regexp.Compile("^(?:" + m[1] + ")$")
			if err != nil {
				return nil, fmt.Errorf("base64Binary: compiling extracted lexical regex %q: %w", m[1], err)
			}
			return re, nil
		}
	}
	return nil, fmt.Errorf("base64Binary: lexical-space regular expression not found")
}

// base64CanonicalOf computes the canonical form of a base64Binary lexical, the
// independent oracle the vectors pin: the §3.3.16.2 encoding is StdEncoding of the
// decoded octets (standard alphabet, '=' padding, no line breaks). Its input is a
// lexical the production regex already accepted, so DecodeString cannot fail.
func base64CanonicalOf(lex string) (string, error) {
	octets, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(lex, " ", ""))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(octets), nil
}

// binaryInvalids keeps the near-miss candidates the extracted regex actually
// rejects, deduplicated and in order — the same discipline decimalInvalids and
// floatingInvalids use, so a spec change that widened the space would drop a
// now-valid candidate rather than let it masquerade as invalid.
func binaryInvalids(re *regexp.Regexp, candidates []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c] || re.MatchString(c) {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// literalsIn returns every '`X`' literal in text, in order.
func literalsIn(text string) []string {
	ms := litRE.FindAllStringSubmatch(text, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
}

// findLine returns the first line of spec containing marker.
func findLine(spec, marker string) (string, bool) {
	for _, line := range strings.Split(spec, "\n") {
		if strings.Contains(line, marker) {
			return line, true
		}
	}
	return "", false
}

// section returns the slice of spec from the first occurrence of startMarker to
// the next occurrence of endMarker after it.
func section(spec, startMarker, endMarker string) (string, error) {
	start := strings.Index(spec, startMarker)
	if start == -1 {
		return "", fmt.Errorf("marker %q not found", startMarker)
	}
	rest := spec[start+len(startMarker):]
	end := strings.Index(rest, endMarker)
	if end == -1 {
		return "", fmt.Errorf("end marker %q not found after %q", endMarker, startMarker)
	}
	return rest[:end], nil
}

// between returns the slice of text strictly between the first open marker and
// the first closing marker after it.
func between(text, open, closing string) (string, error) {
	i := strings.Index(text, open)
	if i == -1 {
		return "", fmt.Errorf("open marker %q not found", open)
	}
	rest := text[i+len(open):]
	j := strings.Index(rest, closing)
	if j == -1 {
		return "", fmt.Errorf("close marker %q not found", closing)
	}
	return rest[:j], nil
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func hasLetter(s string) bool {
	return strings.IndexFunc(s, func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
	}) != -1
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
