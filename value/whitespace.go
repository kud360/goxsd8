package value

import (
	"fmt"
	"strings"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
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
//
// r resolves st's {base type definition} chain, which the §3.16.6.4 overlay
// walks; an unresolvable hop is returned as an error rather than folded into the
// zero mode, because the zero mode means "no normalization applies" and a
// truncated overlay would silently drop an inherited whiteSpace facet — leaving
// a collapse-normalized type parsing its literals raw.
func whiteSpaceInForce(r xsd.TypeResolver, st *xsd.SimpleType) (whiteSpace, error) {
	if st == nil {
		return 0, nil
	}
	eff, err := st.EffectiveFacets(r)
	if err != nil {
		return 0, err
	}
	for _, ef := range eff {
		if ef.Facet().Kind() != xsd.FacetWhiteSpace {
			continue
		}
		values := ef.Facet().Values()
		if len(values) != 1 {
			return 0, nil
		}
		return whiteSpaceOf(values[0]), nil
	}
	return 0, nil
}

// effectiveWhiteSpace is the INSTANCE-VALIDATION view of whiteSpaceInForce: the
// same resolution, but with the missing-mode states classified. Three states, no
// redundant encoding (STYLE D3 — the "is a mode in force" bool the comma-ok result
// once carried is derivable from ws itself, so it is not a third result):
//
//   - err == nil, ws != 0 — the resolved mode.
//   - err == nil, ws == 0 — whiteSpace is CATEGORICALLY not applicable to st, and
//     the normalization stage is skipped rather than defaulted.
//   - err != nil — st carries no usable mode where §3.16.7.4/§4.3.6.1 guarantee
//     one: a facetPrecondition fault of whoever built st (ValidateLexical).
//
// The not-applicable arm is exactly cos-applicable-facets' (§4.1.5) three
// no-facets-applicable cases: an ABSENT {variety} (xs:anySimpleType), an ATOMIC
// {variety} whose {primitive type definition} is absent (xs:anyAtomicType), and a
// UNION, whose applicable facets are pattern, enumeration and assertions —
// whiteSpace conspicuously absent, its normalization deferred per ·active basic
// member· instead (§4.3.6). The first two are the ·special· datatypes, which
// §4.1.4 makes unconditionally Datatype Valid, so demanding a whiteSpace facet of
// them would false-reject the two widest types in the language.
//
// Every OTHER missing-mode state is the fault: a whiteSpace facet whose Values() is
// multi-valued (a malformed generated table), an unrecognized {value} string
// (table/code drift), and an ABSENT facet on an atomic type WITH a primitive or on a
// list — an atomic type's {facets} always carries a whiteSpace entry (§3.16.7.4) and
// a list's carries the materialized fixed collapse facet (§4.3.6.1). The three share
// one error site and so one message, naming all three causes: they are
// indistinguishable to a caller anyway (each is "this type has no usable whiteSpace
// mode and is not one §4.1.5 exempts"), and collapsing them is what keeps the
// resolution itself single.
//
// The rule is xsderr.RuleComponentInvariant, deliberately NOT a §4.3.6 facet ID:
// §4.3.6.3 states outright that "there are no Validation Rules associated with
// whiteSpace", so there is no cvc-* to charge, and §4.3.6.4
// whiteSpace-valid-restriction constrains restriction ORDERING alone (a {value} more
// permissive than the parent whiteSpace's), which says nothing about a mode's
// presence or its {value} domain.
func effectiveWhiteSpace(r xsd.TypeResolver, st *xsd.SimpleType) (whiteSpace, error) {
	ws, err := whiteSpaceInForce(r, st)
	if err != nil {
		return 0, err
	}
	if ws != 0 {
		return ws, nil
	}
	none, err := noFacetsApplicable(r, st)
	if err != nil {
		return 0, err
	}
	if none {
		return 0, nil
	}
	return 0, facetPrecondition(xsderr.RuleComponentInvariant, st.Loc(),
		"value: type %s has no whiteSpace facet in force with exactly one {value} drawn from preserve/replace/collapse (§3.16.7.4, §4.3.6.1)", st.Name())
}

// noFacetsApplicable reports whether cos-applicable-facets (§4.1.5) makes NO
// ·constraining facet· applicable to st: its first three cases, which the clause
// keys on {variety} and {primitive type definition} alone —
//
//	"{variety} is absent … no facets are applicable"
//	"{variety} is atomic and {primitive type definition} is absent … no facets"
//	"{variety} is union … pattern, enumeration, assertions" (no whiteSpace)
//
// — where the union case belongs because whiteSpace is not in its permitted set.
// It is driven off the sealed xsd.Variety sum rather than a name comparison against
// xs:anySimpleType/xs:anyAtomicType, so it holds for any component in those shapes
// and matches lengthExemptPrimitive's .(xsd.Atomic) idiom.
func noFacetsApplicable(r xsd.TypeResolver, st *xsd.SimpleType) (bool, error) {
	variety, err := st.Variety(r)
	if err != nil {
		return false, err
	}
	switch variety.(type) {
	case nil:
		return true, nil
	case xsd.Atomic:
		primitive, err := st.Primitive(r)
		return primitive == nil, err
	case xsd.Union:
		return true, nil
	}
	return false, nil
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
