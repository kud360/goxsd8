package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
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

// parseDuration derives the duration vectors from the Datatypes spec (§3.3.6):
// the lexical space is exactly the combined regular expression nt-durationRep
// gives (extracted verbatim, legibility whitespace removed as the spec
// instructs), so a spec edit that moved it would drop a now-mismatched sample
// rather than mislabel it. Like string (§3.3.1.2) the space is unbounded, so the
// valid sample is a deterministic representative set — a year, a year+month, a
// month, the zero duration, a day, a minute, an hour value that normalizes into
// days, a seconds value that normalizes into minutes, an all-fields value, a
// fractional-seconds value, and a negative — each canonicalised by
// durationCanonicalOf, an INDEPENDENT oracle implementing durationCanonicalMap
// (f-durationCanMap, §E.2), never an echo of the backend. Invalid near-misses (a
// missing 'P', bare "P"/"PT", an out-of-place 'S', and a sign inside a field) are
// kept only if the extracted regex rejects them.
func parseDuration(spec string) (typeVectors, error) {
	re, err := durationLexicalRegex(spec)
	if err != nil {
		return typeVectors{}, err
	}

	sample := []string{
		"P1Y", "P1Y2M", "P1M", "P0M", "P1D", "PT1M",
		"PT36H", "PT60S", "P1DT2H3M4S", "PT1.5S", "-P1Y2M3DT4H5M6S",
	}
	valid := make([]roundtrip, 0, len(sample))
	for _, lex := range sample {
		if !re.MatchString(lex) {
			return typeVectors{}, fmt.Errorf("duration: sample lexical %q does not match its own production regex", lex)
		}
		canon, err := durationCanonicalOf(lex)
		if err != nil {
			return typeVectors{}, fmt.Errorf("duration: canonical of %q: %w", lex, err)
		}
		valid = append(valid, roundtrip{Lexical: lex, Canonical: canon})
	}

	return typeVectors{
		Local:   "duration",
		Valid:   valid,
		Invalid: binaryInvalids(re, []string{"P", "PT", "P1S", "1Y", "PT1D", "PY"}),
	}, nil
}

// durationLexicalRegex extracts duration's combined lexical-space regular
// expression (nt-durationRep, §3.3.6.2) from the fenced block the spec gives as
// "equivalent to the following (after removal of the white space inserted here
// for legibility)" and returns it anchored (^…$) with that legibility whitespace
// removed exactly as the spec instructs.
func durationLexicalRegex(spec string) (*regexp.Regexp, error) {
	i := strings.Index(spec, "equivalent to the following")
	if i == -1 {
		return nil, fmt.Errorf("duration: combined lexical regex marker not found")
	}
	rest := spec[i:]
	open := strings.Index(rest, "```")
	if open == -1 {
		return nil, fmt.Errorf("duration: opening code fence not found")
	}
	rest = rest[open+len("```"):]
	end := strings.Index(rest, "```")
	if end == -1 {
		return nil, fmt.Errorf("duration: closing code fence not found")
	}
	expr := strings.Join(strings.FieldsFunc(rest[:end], func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}), "")
	re, err := regexp.Compile("^(?:" + expr + ")$")
	if err != nil {
		return nil, fmt.Errorf("duration: compiling extracted lexical regex %q: %w", expr, err)
	}
	return re, nil
}

// durationFieldRE extracts the six numeric fragments of a durationLexicalRep in
// order (year, month, day, then post-'T' hour, minute, second). Applied only to
// a lexical the combined regex already accepted, so its all-optional shape is
// safe. It mirrors builtin/strict's durationFields; the generator cannot import
// the private backend, so it carries its own copy (the same discipline the
// float/hexBinary oracles use).
var durationFieldRE = regexp.MustCompile(`^(-)?P(?:([0-9]+)Y)?(?:([0-9]+)M)?(?:([0-9]+)D)?(?:T(?:([0-9]+)H)?(?:([0-9]+)M)?(?:([0-9]+(?:\.[0-9]+)?)S)?)?$`)

// durationCanonicalOf computes the canonical form of a duration lexical, the
// independent oracle the vectors pin: durationMap (f-durationMap) into the
// (months, seconds) tuple, then durationCanonicalMap (f-durationCanMap) back to
// a lexical.
func durationCanonicalOf(lex string) (string, error) {
	neg, months, seconds, err := durationValueOf(lex)
	if err != nil {
		return "", err
	}
	sgn := ""
	if neg {
		sgn = "-"
	}
	monthsZero := months.Sign() == 0
	secondsZero := seconds.Sign() == 0
	switch {
	case !monthsZero && !secondsZero:
		return sgn + "P" + duYearMonthCanon(months) + duDayTimeCanon(seconds), nil
	case !monthsZero:
		return sgn + "P" + duYearMonthCanon(months), nil
	default:
		return sgn + "P" + duDayTimeCanon(seconds), nil
	}
}

// durationValueOf maps a duration lexical to its (negative, months, seconds)
// value tuple (durationMap, f-durationMap): the two halves are computed
// independently and the single leading '-' negates both together; the zero
// duration is signless.
func durationValueOf(lex string) (bool, *big.Int, *big.Rat, error) {
	f := durationFieldRE.FindStringSubmatch(lex)
	if f == nil {
		return false, nil, nil, fmt.Errorf("%q does not parse into duration fields", lex)
	}
	neg := f[1] == "-"
	months := new(big.Int)
	addDurMonths(months, f[2], 12)
	addDurMonths(months, f[3], 1)
	seconds := new(big.Rat)
	addDurSeconds(seconds, f[4], 86400)
	addDurSeconds(seconds, f[5], 3600)
	addDurSeconds(seconds, f[6], 60)
	if f[7] != "" {
		s, ok := new(big.Rat).SetString(f[7])
		if !ok {
			return false, nil, nil, fmt.Errorf("bad second numeral %q", f[7])
		}
		seconds.Add(seconds, s)
	}
	if months.Sign() == 0 && seconds.Sign() == 0 {
		neg = false
	}
	return neg, months, seconds, nil
}

func addDurMonths(acc *big.Int, field string, weight int64) {
	if field == "" {
		return
	}
	n, _ := new(big.Int).SetString(field, 10)
	acc.Add(acc, n.Mul(n, big.NewInt(weight)))
}

func addDurSeconds(acc *big.Rat, field string, weight int64) {
	if field == "" {
		return
	}
	n, _ := new(big.Int).SetString(field, 10)
	term := new(big.Rat).SetInt(n)
	acc.Add(acc, term.Mul(term, new(big.Rat).SetInt64(weight)))
}

// duYearMonthCanon implements duYearMonthCanonicalFragmentMap (f-duYMCan) for a
// nonzero months magnitude.
func duYearMonthCanon(months *big.Int) string {
	y, m := new(big.Int), new(big.Int)
	y.DivMod(months, big.NewInt(12), m)
	switch {
	case y.Sign() != 0 && m.Sign() != 0:
		return y.String() + "Y" + m.String() + "M"
	case y.Sign() != 0:
		return y.String() + "Y"
	default:
		return m.String() + "M"
	}
}

// duDayTimeCanon implements duDayTimeCanonicalFragmentMap (f-duDTCan): "T0S" for
// a zero magnitude, else days plus the time fragment.
func duDayTimeCanon(seconds *big.Rat) string {
	if seconds.Sign() == 0 {
		return "T0S"
	}
	day, rem := durRatDivMod(seconds, 86400)
	hour, rem := durRatDivMod(rem, 3600)
	minute, second := durRatDivMod(rem, 60)
	dayFrag := ""
	if day.Sign() != 0 {
		dayFrag = day.String() + "D"
	}
	return dayFrag + duTimeCanon(hour, minute, second)
}

// duTimeCanon implements duTimeCanonicalFragmentMap (f-duTCan): 'T' then each
// nonzero component, or "" when all three are zero.
func duTimeCanon(hour, minute *big.Int, second *big.Rat) string {
	if hour.Sign() == 0 && minute.Sign() == 0 && second.Sign() == 0 {
		return ""
	}
	out := "T"
	if hour.Sign() != 0 {
		out += hour.String() + "H"
	}
	if minute.Sign() != 0 {
		out += minute.String() + "M"
	}
	if second.Sign() != 0 {
		out += duSecondCanon(second) + "S"
	}
	return out
}

// duSecondCanon implements duSecondCanonicalFragmentMap (f-duSCan) without the
// trailing 'S': a bare integer or a terminating decimal.
func duSecondCanon(second *big.Rat) string {
	if second.IsInt() {
		return second.Num().String()
	}
	num, den := new(big.Int).Set(second.Num()), second.Denom()
	intPart, rem := new(big.Int), new(big.Int)
	intPart.QuoRem(num, den, rem)
	var frac []byte
	for rem.Sign() != 0 {
		rem.Mul(rem, big.NewInt(10))
		digit, mod := new(big.Int), new(big.Int)
		digit.QuoRem(rem, den, mod)
		frac = append(frac, byte('0'+digit.Int64()))
		rem = mod
	}
	return intPart.String() + "." + string(frac)
}

// durRatDivMod splits a nonnegative rational r as q·w + rem with q integer and
// 0 ≤ rem < w (the spec's ·div·/·mod· on decimals).
func durRatDivMod(r *big.Rat, w int64) (*big.Int, *big.Rat) {
	weight := big.NewInt(w)
	q := new(big.Int).Quo(r.Num(), new(big.Int).Mul(r.Denom(), weight))
	rem := new(big.Rat).Sub(r, new(big.Rat).SetInt(new(big.Int).Mul(q, weight)))
	return q, rem
}

// parseDateTime derives the dateTime vectors from the Datatypes spec (§3.3.7):
// the grammar's lexical space is the combined regular expression nt-dateTimeRep
// gives (extracted verbatim from the "equivalent to this regular expression"
// block, legibility whitespace removed as the spec instructs), so a spec edit
// that moved it would drop a now-mismatched sample rather than mislabel it. Like
// string the space is unbounded, so the valid sample is a deterministic
// representative set — a plain instant, a fractional-second, the three timezone
// spellings (+hh:mm, Z, −hh:mm), a leap-day, an endOfDayFrag that carries into
// the next day, a negative year, and a year needing four-digit padding — each
// canonicalised by dateTimeCanonicalOf, an INDEPENDENT oracle implementing
// dateTimeLexicalMap + dateTimeCanonicalMap (vp-dateTimeLexRep/vp-dateTimeCanRep,
// §E.3.5/§E.3.6), never an echo of the backend. Invalid near-misses are kept via
// dateTimeInvalids (not binaryInvalids): the day-of-month constraint
// con-dateTime-dayValue (§3.3.7.1) is BEYOND the grammar regex, so validity is
// decided by the full oracle, letting bad-day cases like 2023-02-30 count as
// invalid even though the pure regex accepts them.
func parseDateTime(spec string) (typeVectors, error) {
	re, err := dateTimeLexicalRegex(spec)
	if err != nil {
		return typeVectors{}, err
	}

	sample := []string{
		"2001-10-26T21:32:52",
		"2001-10-26T21:32:52.125",
		"2001-10-26T21:32:52+02:00",
		"2001-10-26T19:32:52Z",
		"2001-10-26T21:32:52-05:00",
		"2024-02-29T00:00:00", // leap-year February 29 is valid
		"2023-01-01T24:00:00", // endOfDayFrag carries into the next day
		"-0045-03-15T00:00:00Z",
		"0001-01-01T00:00:00",
	}
	valid := make([]roundtrip, 0, len(sample))
	for _, lex := range sample {
		if !re.MatchString(lex) {
			return typeVectors{}, fmt.Errorf("dateTime: sample lexical %q does not match its own production regex", lex)
		}
		canon, err := dateTimeCanonicalOf(lex)
		if err != nil {
			return typeVectors{}, fmt.Errorf("dateTime: canonical of %q: %w", lex, err)
		}
		valid = append(valid, roundtrip{Lexical: lex, Canonical: canon})
	}

	return typeVectors{
		Local: "dateTime",
		Valid: valid,
		Invalid: dateTimeInvalids([]string{
			"2023-13-01T00:00:00",       // month out of range
			"2023-02-30T00:00:00",       // day beyond February (regex-valid, value-invalid)
			"2023-02-29T00:00:00",       // February 29 in a non-leap year
			"2023-01-01T25:00:00",       // hour out of range
			"2023-01-01T00:60:00",       // minute out of range
			"2023-01-0100:00:00",        // missing 'T' separator
			"2023-01-01T00:00:00+15:00", // timezone offset beyond ±14:00
			"2023-1-01T00:00:00",        // month not two digits
		}),
	}, nil
}

// dateTimeLexicalRegex extracts dateTime's combined lexical-space regular
// expression (nt-dateTimeRep, §3.3.7.2) from the fenced block the spec gives as
// "The dateTimeLexicalRep production is equivalent to this regular expression once
// whitespace is removed" and returns it anchored (^…$) with that legibility
// whitespace removed exactly as the spec instructs.
func dateTimeLexicalRegex(spec string) (*regexp.Regexp, error) {
	i := strings.Index(spec, "regular expression once whitespace is removed")
	if i == -1 {
		return nil, fmt.Errorf("dateTime: combined lexical regex marker not found")
	}
	rest := spec[i:]
	open := strings.Index(rest, "```")
	if open == -1 {
		return nil, fmt.Errorf("dateTime: opening code fence not found")
	}
	rest = rest[open+len("```"):]
	end := strings.Index(rest, "```")
	if end == -1 {
		return nil, fmt.Errorf("dateTime: closing code fence not found")
	}
	expr := strings.Join(strings.FieldsFunc(rest[:end], func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	}), "")
	compiled, err := regexp.Compile("^(?:" + expr + ")$")
	if err != nil {
		return nil, fmt.Errorf("dateTime: compiling extracted lexical regex %q: %w", expr, err)
	}
	return compiled, nil
}

// dateTimeFieldRE extracts the fragments of a dateTimeLexicalRep. Applied only to
// a lexical the combined regex already accepted, it mirrors builtin/strict's
// dateTimeLexical; the generator cannot import the private backend, so it carries
// its own copy (the same discipline the duration/hexBinary oracles use). Groups:
// 1 year, 2 month, 3 day, 4 hour, 5 minute, 6 second-int, 7 second-frac,
// 8 endOfDayFrag, 9 timezoneFrag.
var dateTimeFieldRE = regexp.MustCompile(
	`^(-?(?:[1-9][0-9]{3,}|0[0-9]{3}))-(0[1-9]|1[0-2])-(0[1-9]|[12][0-9]|3[01])` +
		`T(?:([01][0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])(\.[0-9]+)?|(24:00:00(?:\.0+)?))` +
		`(Z|[+-](?:(?:0[0-9]|1[0-3]):[0-5][0-9]|14:00))?$`)

// dateTimeCanonicalOf is the independent oracle the vectors pin: dateTimeLexicalMap
// (vp-dateTimeLexRep) into the seven-property value, then dateTimeCanonicalMap
// (vp-dateTimeCanRep) back to a lexical. It returns an error when the lexical is
// not in the value space — a regex mismatch OR a day-of-month value violation
// (con-dateTime-dayValue) — so dateTimeInvalids can decide validity through it.
func dateTimeCanonicalOf(lex string) (string, error) {
	m := dateTimeFieldRE.FindStringSubmatch(lex)
	if m == nil {
		return "", fmt.Errorf("%q does not match dateTimeLexicalRep", lex)
	}
	year, _ := new(big.Int).SetString(m[1], 10)
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	if day > dtDaysInMonth(year, month) {
		return "", fmt.Errorf("%q has day %d out of range for month %d", lex, day, month)
	}

	var tz *int
	if m[9] != "" {
		off := dtTimezoneOffset(m[9])
		tz = &off
	}

	var hour, minute int
	second := new(big.Rat)
	if m[8] != "" { // endOfDayFrag: hour 24 carries into the next calendar day
		year, month, day = dtNextDay(year, month, day)
	} else {
		hour, _ = strconv.Atoi(m[4])
		minute, _ = strconv.Atoi(m[5])
		second, _ = new(big.Rat).SetString(m[6] + m[7])
	}

	var b strings.Builder
	b.WriteString(dtYearCanon(year))
	fmt.Fprintf(&b, "-%02d-%02dT%02d:%02d:", month, day, hour, minute)
	b.WriteString(dtSecondCanon(second))
	if tz != nil {
		b.WriteString(dtTzCanon(*tz))
	}
	return b.String(), nil
}

// dateTimeInvalids keeps the near-miss candidates the oracle rejects,
// deduplicated and in order. Unlike binaryInvalids it filters through the FULL
// validity oracle (dateTimeCanonicalOf), not the grammar regex alone, so a
// day-of-month violation (con-dateTime-dayValue, §3.3.7.1) — which the regex
// cannot express — still counts as invalid.
func dateTimeInvalids(candidates []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range candidates {
		if seen[c] {
			continue
		}
		if _, err := dateTimeCanonicalOf(c); err == nil {
			continue // the oracle accepts it — not an invalid lexical
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// dtTimezoneOffset maps a timezoneFrag to ·timezoneOffset· minutes (f-dt-tzMap):
// 'Z' is 0, else the signed hh:mm.
func dtTimezoneOffset(frag string) int {
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

// dtNextDay rolls (year, month, day) forward one calendar day for the endOfDayFrag
// carry (§3.3.7.2), overflowing month and year at a month boundary.
func dtNextDay(year *big.Int, month, day int) (*big.Int, int, int) {
	if day < dtDaysInMonth(year, month) {
		return year, month, day + 1
	}
	if month < 12 {
		return year, month + 1, 1
	}
	return new(big.Int).Add(year, big.NewInt(1)), 1, 1
}

// dtDaysInMonth is the month length in year (con-dateTime-dayValue, §3.3.7.1),
// leap-year aware for February.
func dtDaysInMonth(year *big.Int, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	}
	if dtIsLeap(year) {
		return 29
	}
	return 28
}

// dtIsLeap applies the proleptic-Gregorian leap rule (divisible by 4, except
// centuries not divisible by 400); divisibility is sign-independent.
func dtIsLeap(year *big.Int) bool {
	div := func(n int64) bool { return new(big.Int).Rem(year, big.NewInt(n)).Sign() == 0 }
	if !div(4) {
		return false
	}
	if !div(100) {
		return true
	}
	return div(400)
}

// dtYearCanon implements yearCanonicalFragmentMap (f-yrCanFragMap): a plain signed
// numeral when |year| > 9999, else a four-digit numeral with the sign preserved.
func dtYearCanon(year *big.Int) string {
	abs := new(big.Int).Abs(year)
	if abs.Cmp(big.NewInt(9999)) > 0 {
		return year.String()
	}
	if year.Sign() < 0 {
		return fmt.Sprintf("-%04d", abs.Int64())
	}
	return fmt.Sprintf("%04d", abs.Int64())
}

// dtSecondCanon implements secondCanonicalFragmentMap (f-seCanFragMap): a
// two-digit integer part plus the exact terminating fractional digits when the
// value is not integral.
func dtSecondCanon(second *big.Rat) string {
	intPart, rem := new(big.Int), new(big.Int)
	intPart.QuoRem(second.Num(), second.Denom(), rem)
	whole := fmt.Sprintf("%02d", intPart.Int64())
	if rem.Sign() == 0 {
		return whole
	}
	den := second.Denom()
	var frac []byte
	for rem.Sign() != 0 {
		rem.Mul(rem, big.NewInt(10))
		digit, mod := new(big.Int), new(big.Int)
		digit.QuoRem(rem, den, mod)
		frac = append(frac, byte('0'+digit.Int64()))
		rem = mod
	}
	return whole + "." + string(frac)
}

// dtTzCanon implements timezoneCanonicalFragmentMap (f-tzCanFragMap): 'Z' for
// offset 0, else the signed hh:mm.
func dtTzCanon(offset int) string {
	if offset == 0 {
		return "Z"
	}
	if offset < 0 {
		return fmt.Sprintf("-%02d:%02d", -offset/60, -offset%60)
	}
	return fmt.Sprintf("+%02d:%02d", offset/60, offset%60)
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
