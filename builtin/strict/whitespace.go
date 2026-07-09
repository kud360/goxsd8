package strict

import (
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/builtin"
)

// whiteSpace is the value of the whiteSpace facet (Datatypes §4.3.6,
// rf-whiteSpace): the pre-lexical normalization a type applies to a raw literal
// before the pattern and lexical-mapping stages run (key-nv, §3.1.4: whiteSpace
// is applied first among the pre-lexical facets). iota+1 leaves the zero value
// an invalid sentinel that catches an unset mode (STYLE T1, matching
// builtin.Ordered and regex.Flavor).
type whiteSpace uint8

const (
	preserveWS whiteSpace = iota + 1 // no change
	replaceWS                        // #x9/#xA/#xD → #x20
	collapseWS                       // replace, then collapse #x20 runs to one and trim ends
)

// normalizeWhiteSpace applies the whiteSpace facet's normalization to s exactly
// as §4.3.6 defines it: preserve leaves s unchanged; replace maps each
// tab (#x9), line feed (#xA) and carriage return (#xD) to a space (#x20);
// collapse does the replace step, then collapses every run of #x20 to a single
// space and trims leading and trailing spaces. It is a transform that PRODUCES
// the normalized lexical, not a value.LexicalFacet check (which only accepts or
// rejects) — the two are kept structurally separate (warden pre-flight).
func normalizeWhiteSpace(s string, ws whiteSpace) string {
	switch ws {
	case preserveWS:
		return s
	case replaceWS:
		return replaceSpace(s)
	case collapseWS:
		return collapseSpace(replaceSpace(s))
	}
	// The zero value (or any unlisted mode) is an internal bug, never user input.
	panic(fmt.Sprintf("strict: invalid whiteSpace mode %d", ws))
}

// replaceSpace maps #x9/#xA/#xD to #x20 (the replace step of §4.3.6), leaving
// every other character — including other Unicode whitespace — untouched.
func replaceSpace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\t', '\n', '\r':
			return ' '
		}
		return r
	}, s)
}

// collapseSpace collapses runs of #x20 to a single space and trims leading and
// trailing #x20, the collapse step of §4.3.6 (its input has already had
// #x9/#xA/#xD mapped to #x20 by replaceSpace). It collapses ONLY #x20; other
// Unicode whitespace is not a space per §4.3.6 and is preserved. Byte-wise
// scanning is safe because #x20 never appears inside a multi-byte UTF-8
// sequence (continuation bytes are ≥ 0x80).
func collapseSpace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	pendingSpace := false
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			pendingSpace = true
			continue
		}
		if pendingSpace && b.Len() > 0 {
			b.WriteByte(' ')
		}
		pendingSpace = false
		b.WriteByte(s[i])
	}
	return b.String()
}

// whiteSpaceOf resolves the cohort type named local to its whiteSpace mode,
// reading the mode from its generated TypeSpec in builtin.Types — the single
// source of the per-type whiteSpace default (§4.3.6: string=preserve,
// boolean/decimal=collapse fixed). The fact is never hand-duplicated here
// (STYLE D3, warden pre-flight). An unknown local name, a type with no
// whiteSpace facet, or an unrecognized default string is an internal-consistency
// failure (the generated table and this cohort disagree), not user input, so it
// panics rather than returning an error.
func whiteSpaceOf(local string) whiteSpace {
	for i := range builtin.Types {
		t := builtin.Types[i]
		if t.Name != local {
			continue
		}
		for j := range t.Facets {
			if t.Facets[j].Name != "whiteSpace" {
				continue
			}
			return parseWhiteSpace(t.Facets[j].Default)
		}
		panic(fmt.Sprintf("strict: builtin %q has no whiteSpace facet in builtin.Types", local))
	}
	panic(fmt.Sprintf("strict: no builtin TypeSpec named %q", local))
}

// parseWhiteSpace maps a whiteSpace facet default string to the typed mode.
func parseWhiteSpace(def string) whiteSpace {
	switch def {
	case "preserve":
		return preserveWS
	case "replace":
		return replaceWS
	case "collapse":
		return collapseWS
	}
	panic(fmt.Sprintf("strict: unrecognized whiteSpace default %q in builtin.Types", def))
}

// normalizeForParse runs the whiteSpace pre-lexical stage (§4.3.6) for the
// cohort type named local over the raw instance literal, returning the
// normalized lexical ready to feed to that type's Parse. It is the stage that
// runs BEFORE the lexical mapping: value/backend.go's Mapping.Parse contract
// expects already-normalized input, so parseDecimal/parseBoolean/parseString are
// unchanged — this composes ahead of them rather than folding normalization into
// them.
func normalizeForParse(local, raw string) string {
	return normalizeWhiteSpace(raw, whiteSpaceOf(local))
}
