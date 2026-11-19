package value_test

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xpath"
	"github.com/kud360/goxsd8/xsderr"
)

// TestValidateLexicalChargesInvalidUTF8UnderReplaceAndCollapse pins that the
// whiteSpace stage hands strict the literal's real bytes (§4.3.6
// rf-whiteSpace), so an invalid UTF-8 byte is charged cvc-datatype-valid under
// xs:normalizedString (replace), xs:token (collapse) and, at its item,
// xs:NMTOKENS (collapse). Under strict, value.GoverningMapping of each type, or
// of its item type, is xs:string's, so the message names xs:string and quotes
// the normalized lexical with the invalid byte kept. Each row fails
// under a rune-decoding replaceSpace such as strings.Map, which substitutes
// U+FFFD, a legal Char.
func TestValidateLexicalChargesInvalidUTF8UnderReplaceAndCollapse(t *testing.T) {
	b := strict.New()
	for _, row := range []struct {
		local, lexical, normalized string
	}{
		{"normalizedString", "a\xff\tb", "a\xff b"},
		{"token", " a\xffb ", "a\xffb"},
		{"token", "a\xe2\x82", "a\xe2\x82"},
		{"NMTOKENS", "a\xffb", "a\xffb"},
	} {
		v, err := value.ValidateLexical(b, builtinsOnly{}, seededType(t, b, row.local), row.lexical, nil, xpath.FacetAssertions(time.Time{}))
		if err == nil {
			t.Errorf("ValidateLexical(xs:%s, %q) = (%q, nil), want cvc-datatype-valid", row.local, row.lexical, v)
			continue
		}
		if r, _ := xsderr.RuleOf(err); r != "cvc-datatype-valid" {
			t.Errorf("ValidateLexical(xs:%s, %q) charged %s (%v), want cvc-datatype-valid", row.local, row.lexical, r, err)
		}
		subject := strconv.Quote(row.normalized) + " holds a byte at offset 1 that starts no XML 1.0 Char"
		if msg := err.Error(); !strings.Contains(msg, subject) || !strings.Contains(msg, "lexical space of xs:string") {
			t.Errorf("ValidateLexical(xs:%s, %q) = %v, want %s ... lexical space of xs:string", row.local, row.lexical, err, subject)
		}
	}
}
