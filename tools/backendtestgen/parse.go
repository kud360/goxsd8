package main

import (
	"fmt"
	"regexp"
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
