package strict

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// anyURIVal is an anyURI value (§3.3.17). Its value space is the set of
// finite-length sequences of Char (§3.3.17.1), so it is codepoint-identical in
// shape to a string; it is NOT ordered (ordered=false, §3.3.17.3): it defines no
// Cmp, so it structurally lacks value.Ordered.
type anyURIVal string

// parseAnyURI maps an anyURI lexical to its value. The lexical mapping is the
// IDENTITY on the domain (§3.3.17.2), value space == lexical space, and every
// XML-well-formed string reaching this code already matches Char* — so anyURI
// has no invalid lexical. The spec deliberately disclaims URI-syntax checking
// (§3.3.17.2: "Because it is impractical for processors to check that a value is
// a context-appropriate URI reference, neither the syntactic constraints defined
// by the definitions of individual schemes nor the generic syntactic constraints
// defined by [RFC 3987] and [RFC 3986] ... are part of this datatype"), so Parse
// must NOT transform or reject anything, and never returns an error — mirroring
// parseString exactly.
func parseAnyURI(lexical string, _ value.Context) (value.Value, error) {
	return anyURIVal(lexical), nil
}

// canonicalAnyURI is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail). It never fails
// for an anyURI value — the canonical mapping is the identity.
func canonicalAnyURI(v value.Value) (string, error) {
	u, ok := v.(anyURIVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"anyURI canonical: value of type %T is not a strict anyURI", v)
	}
	return u.Canonical(), nil
}

// Canonical is the identity: no dedicated canonical hfn exists for anyURI because
// value space == lexical space (§3.3.17.2). URI equivalences (percent-encoding
// case, trailing slash, scheme case) are deliberately NOT canonicalized
// (§3.3.17.2 Note: "if two 'equivalent' URIs or IRIs are different character
// sequences, they map to different values in this datatype").
func (u anyURIVal) Canonical() string { return string(u) }

// Eq is codepoint identity (§3.3.17.2 Note: distinct character sequences are
// distinct values). A non-anyURI argument is unequal.
func (u anyURIVal) Eq(other value.Value) bool {
	o, ok := other.(anyURIVal)
	if !ok {
		return false
	}
	return u == o
}

// Len is the length the length/minLength/maxLength facets measure (§4.3.1–3):
// the number of characters, i.e. Unicode codepoints, not bytes.
func (u anyURIVal) Len() int { return len([]rune(string(u))) }
