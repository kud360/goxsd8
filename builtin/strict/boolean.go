package strict

import (
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// boolVal is a boolean value (§3.3.2). It is NOT ordered (ordered=false,
// §3.3.2.3): it defines no Cmp, so it structurally lacks value.Ordered.
type boolVal bool

// parseBoolean maps a boolean lexical to its value (f-booleanLexmap, §3.3.2.2).
// The lexical space is EXACTLY the four literals true/false/1/0
// (boolean-lexical-mapping) — no case or whitespace variants (whiteSpace is a
// fixed pre-lexical stage; " true" is simply not one of the four literals).
func parseBoolean(lexical string, _ value.Context) (value.Value, error) {
	switch lexical {
	case "true", "1":
		return boolVal(true), nil
	case "false", "0":
		return boolVal(false), nil
	}
	return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
		"boolean: %q is not in the lexical space (boolean-lexical-mapping, §3.3.2.1)", lexical)
}

// canonicalBoolean is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail).
func canonicalBoolean(v value.Value) (string, error) {
	b, ok := v.(boolVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"boolean canonical: value of type %T is not a strict boolean", v)
	}
	return b.Canonical(), nil
}

// Canonical is always the word form true/false (f-booleanCanmap, §3.3.2.2),
// never 1/0.
func (b boolVal) Canonical() string {
	if bool(b) {
		return "true"
	}
	return "false"
}

// Eq is boolean equality (§2.2.2). A non-boolean argument is unequal.
func (b boolVal) Eq(other value.Value) bool {
	o, ok := other.(boolVal)
	if !ok {
		return false
	}
	return b == o
}

// Identical coincides with Eq for boolean: §2.2.2 gives boolean no special
// identity carve-out (unlike NaN or signed zero), so identity is exactly
// equality here (one encoding of that fact, STYLE D3).
func (b boolVal) Identical(other value.Value) bool { return b.Eq(other) }
