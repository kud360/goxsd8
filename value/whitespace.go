package value

import (
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/xsd"
)

// whiteSpace is the value of the whiteSpace facet (Datatypes §4.3.6,
// rf-whiteSpace): the pre-lexical normalization a type applies to a raw literal
// before the pattern and lexical-mapping stages run (key-nv, §3.1.4: whiteSpace
// is applied first among the pre-lexical facets). iota+1 leaves the zero value
// an invalid sentinel that catches an unset mode (STYLE T1, matching
// regex.Flavor).
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
// the normalized lexical, not a LexicalFacet check (which only accepts or
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
	panic(fmt.Sprintf("value: invalid whiteSpace mode %d", ws))
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

// effectiveWhiteSpace resolves st's whiteSpace mode by scanning its
// EffectiveFacets for the whiteSpace facet and mapping its {value}
// ("preserve"/"replace"/"collapse") to the typed mode (§4.3.6). Reading the
// facet off EffectiveFacets — rather than the primitive's per-type default in a
// side table — honors a legal derived whiteSpace override under the ordinary
// same-kind replace overlay (key-facets-overlay §3.16.6.4): a more-derived
// whiteSpace facet supersedes the primitive's, and EffectiveFacets surfaces the
// winner. For the atomic cohort the primitive node's own {facets} always carries
// a whiteSpace entry (§3.16.7.4), so a derived type that does not itself declare
// one still resolves through the inherited primitive facet.
//
// A type with NO whiteSpace facet in force (e.g. a union variety, to which
// whiteSpace "does not apply directly", §4.3.6) is outside this atomic-only
// cohort — the precondition ValidateLexical documents — never instance data, so
// it panics rather than returning an error. An unrecognized {value} is likewise
// an internal-consistency failure between a generated table and this code.
func effectiveWhiteSpace(st *xsd.SimpleType) whiteSpace {
	for _, ef := range st.EffectiveFacets() {
		if ef.Facet().Kind() != xsd.FacetWhiteSpace {
			continue
		}
		values := ef.Facet().Values()
		if len(values) != 1 {
			panic(fmt.Sprintf("value: whiteSpace facet on %s must carry exactly one value, has %d", st.Name(), len(values)))
		}
		switch values[0] {
		case "preserve":
			return preserveWS
		case "replace":
			return replaceWS
		case "collapse":
			return collapseWS
		}
		panic(fmt.Sprintf("value: unrecognized whiteSpace facet value %q on %s", values[0], st.Name()))
	}
	panic(fmt.Sprintf("value: type %s has no whiteSpace facet in force", st.Name()))
}
