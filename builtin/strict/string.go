package strict

import (
	"github.com/kud360/goxsd8/internal/xmlchar"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// stringVal is a string value (§3.3.1). It is NOT ordered (ordered=false,
// §3.3.1.3): it defines no Cmp, so it structurally lacks value.Ordered.
type stringVal string

// parseString maps a string lexical to its value: f-stringLexmap is the
// identity on the lexical space (§3.3.1.2), so Parse transforms nothing and
// rejects only a lexical outside it, as New's xs:string bullet states.
func parseString(lexical string, _ value.Context) (value.Value, error) {
	if err := checkChars("string", lexical); err != nil {
		return nil, err
	}
	return stringVal(lexical), nil
}

// checkChars returns the cvc-datatype-valid clause 2.1 error for a lexical of
// xs:typ that is not UTF-8 of XML 1.0 [2] Char characters only, and nil for
// one that is.
func checkChars(typ, lexical string) error {
	i := xmlchar.IndexNonChar(lexical)
	if i < 0 {
		return nil
	}
	return xsderr.New(ruleDatatypeValid, xsderr.Loc{},
		"%q holds a byte at offset %d that starts no XML 1.0 Char, so it is not in the lexical space of xs:%s, which cvc-datatype-valid clause 2.1 requires it to be in", lexical, i, typ)
}

// canonicalString is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail). It never fails
// for a string value — the canonical mapping is the identity.
func canonicalString(v value.Value) (string, error) {
	s, ok := v.(stringVal)
	if !ok {
		return "", xsderr.New(ruleDatatypeValid, xsderr.Loc{},
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
