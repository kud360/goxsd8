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

// whiteSpaceInForce resolves the whiteSpace mode in force on st by scanning its
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
// It is THE single whiteSpace resolution in this package: the instance-validation
// stage reaches it through effectiveWhiteSpace, and facet-{value} parsing reaches
// it directly (facets.go's declaringFacetSpace, restriction.go's
// CheckFacetRestriction). It returns the ZERO whiteSpace — an invalid mode by
// construction, see the iota+1 above — for every state in which no mode applies:
// a nil st, no whiteSpace facet in force at all, a multi-valued whiteSpace facet,
// or a {value} outside the §4.3.6.1 three-token domain. That single "no mode"
// encoding is what lets facetValue leave a lexical unchanged without a second
// comma-ok flag beside the mode (STYLE D3), and what lets effectiveWhiteSpace
// decide, from st's {variety} alone, whether the absence is spec-mandated or a
// construction bug.
func whiteSpaceInForce(st *xsd.SimpleType) whiteSpace {
	if st == nil {
		return 0
	}
	for _, ef := range st.EffectiveFacets() {
		if ef.Facet().Kind() != xsd.FacetWhiteSpace {
			continue
		}
		values := ef.Facet().Values()
		if len(values) != 1 {
			return 0
		}
		return whiteSpaceOf(values[0])
	}
	return 0
}

// effectiveWhiteSpace is the INSTANCE-VALIDATION view of whiteSpaceInForce: the
// same resolution, but with the missing-mode states classified.
//
// The comma-ok result models whether a whiteSpace facet is in force at all.
// applicable=true returns the resolved mode. applicable=false (ws is the zero
// value) means whiteSpace is CATEGORICALLY not applicable to st: this happens
// only for a union {variety}, whose applicable facets are pattern, enumeration
// and assertions — whiteSpace is conspicuously absent (cos-applicable-facets
// §4.1.5), and a union's normalization is instead deferred per active basic
// member (§4.3.6). A caller that ignores the bool and feeds the zero ws to
// normalizeWhiteSpace still panics there, so the false result cannot silently
// degrade into a wrong normalization.
//
// Every OTHER missing-mode state is an UNCHANGED internal-consistency panic, not
// relaxed: a whiteSpace facet whose Values() is multi-valued (a malformed
// generated table), an unrecognized {value} string (table/code drift), and an
// ABSENT facet on a non-union (atomic or list) type — an atomic type's {facets}
// always carries a whiteSpace entry (§3.16.7.4) and a list's carries the
// materialized fixed collapse facet (§4.3.6.1), so its absence there is a
// schema-construction bug, never a legitimate "not applicable" case. Only the
// confirmed-union case is relaxed to (0, false). The three now share one panic
// site and so one message, naming all three causes: they are indistinguishable
// to a caller anyway (each is "this type has no usable whiteSpace mode and is not
// a union"), and collapsing them is what keeps the resolution itself single.
func effectiveWhiteSpace(st *xsd.SimpleType) (ws whiteSpace, applicable bool) {
	if ws := whiteSpaceInForce(st); ws != 0 {
		return ws, true
	}
	// No usable mode. For a union {variety} this is spec-mandated (whiteSpace is
	// not an applicable facet, cos-applicable-facets §4.1.5), so the stage is
	// "not applicable" rather than an error. Drive it off the sealed xsd.Variety
	// sum, matching lengthExemptPrimitive's .(xsd.Atomic) idiom.
	if _, isUnion := st.Variety().(xsd.Union); isUnion {
		return 0, false
	}
	panic(fmt.Sprintf("value: type %s has no whiteSpace facet in force with exactly one {value} drawn from preserve/replace/collapse (§3.16.7.4, §4.3.6.1)", st.Name()))
}

// whiteSpaceOf maps a whiteSpace facet's {value} token to its typed mode
// (§4.3.6.1: the {value} domain is exactly preserve/replace/collapse), returning
// the zero whiteSpace for any other string. It is the single place the three spec
// tokens are spelled, kept as its own function so the token table stays one fact
// separate from whiteSpaceInForce's facet-scan.
func whiteSpaceOf(v string) whiteSpace {
	switch v {
	case "preserve":
		return preserveWS
	case "replace":
		return replaceWS
	case "collapse":
		return collapseWS
	}
	return 0
}
