package strict

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// stringVal is a string value (§3.3.1). It is NOT ordered (ordered=false,
// §3.3.1.3): it defines no Cmp, so it structurally lacks value.Ordered.
type stringVal string

// parseString maps a string lexical to its value. Both f-stringLexmap and
// f-stringCanmap are the identity on the domain (§3.3.1.2), and every string is
// in the lexical space (there is no invalid string lexical; whiteSpace=preserve
// is not fixed). So Parse must NOT transform or reject anything, and never
// returns an error.
func parseString(lexical string, _ value.Context) (value.Value, error) {
	return stringVal(lexical), nil
}

// canonicalString is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail). It never fails
// for a string value — the canonical mapping is the identity.
func canonicalString(v value.Value) (string, error) {
	s, ok := v.(stringVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"string canonical: value of type %T is not a strict string", v)
	}
	return s.Canonical(), nil
}

// Canonical is the identity (f-stringCanmap, §3.3.1.2).
func (s stringVal) Canonical() string { return string(s) }

// Eq is codepoint identity (sec-vs-string, §2.2.2: "Equality for string is
// identity. No order is prescribed."). A non-string argument is unequal.
func (s stringVal) Eq(other value.Value) bool {
	o, ok := other.(stringVal)
	if !ok {
		return false
	}
	return s == o
}

// Len is the length the length/minLength/maxLength facets measure (§4.3.1–3):
// the number of characters, i.e. Unicode codepoints, not bytes.
func (s stringVal) Len() int { return len([]rune(string(s))) }
