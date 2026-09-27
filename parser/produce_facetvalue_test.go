package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// facetValueSchema wraps one facet child in a named simple type's <restriction>
// over base, putting the facet element on line 3 of the document.
func facetValueSchema(base, facet string) string {
	return wrap("", "\n"+`<xs:simpleType name="T"><xs:restriction base="xs:`+base+`">`+"\n"+
		facet+"\n"+`</xs:restriction></xs:simpleType>`)
}

// TestProduceFacetCountValueRejected pins that a length or digits facet's value
// literal outside the type the schema for schema documents declares for it —
// xs:nonNegativeInteger on xs:numFacet, xs:positiveInteger on <totalDigits> — is
// charged cvc-datatype-valid at the facet element, from Produce alone with no
// instance in play (#1774). The message is pinned from its opening subject, so a
// swapped argument cannot pass.
func TestProduceFacetCountValueRejected(t *testing.T) {
	cases := []struct {
		name    string
		base    string
		facet   string
		wantMsg string
	}{
		{"maxLength empty", "string", `<xs:maxLength value=""/>`, `<maxLength> value "" is not a nonNegativeInteger`},
		{"maxLength negative", "string", `<xs:maxLength value="-1"/>`, `<maxLength> value "-1" is not a nonNegativeInteger`},
		{"maxLength non-numeric", "string", `<xs:maxLength value="a"/>`, `<maxLength> value "a" is not a nonNegativeInteger`},
		{"maxLength exponent", "string", `<xs:maxLength value="1e2"/>`, `<maxLength> value "1e2" is not a nonNegativeInteger`},
		{"length negative", "string", `<xs:length value="-1"/>`, `<length> value "-1" is not a nonNegativeInteger`},
		{"minLength non-numeric", "string", `<xs:minLength value="a"/>`, `<minLength> value "a" is not a nonNegativeInteger`},
		{"fractionDigits exponent", "decimal", `<xs:fractionDigits value="1e2"/>`, `<fractionDigits> value "1e2" is not a nonNegativeInteger`},
		{"fractionDigits non-XML-whitespace padding", "decimal", "<xs:fractionDigits value=\"\u00a05\"/>", `<fractionDigits> value "\u00a05" is not a nonNegativeInteger`},
		// "0" is a nonNegativeInteger, so only the positiveInteger arm rejects it.
		{"totalDigits zero", "decimal", `<xs:totalDigits value="0"/>`, `<totalDigits> value "0" is not a positiveInteger`},
		{"totalDigits negative", "decimal", `<xs:totalDigits value="-3"/>`, `<totalDigits> value "-3" is not a positiveInteger`},
		{"totalDigits empty", "decimal", `<xs:totalDigits value=""/>`, `<totalDigits> value "" is not a positiveInteger`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, facetValueSchema(tc.base, tc.facet))
			assertRule(t, err, "cvc-datatype-valid")
			if !strings.Contains(err.Error(), "] "+tc.wantMsg) {
				t.Fatalf("diagnostic %q, want it to open with %q", err, tc.wantMsg)
			}
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position (STYLE E3)", err)
			}
			if loc.URI != produceURI || loc.Line != 3 || loc.Col != 1 {
				t.Fatalf("position = %s:%d:%d, want the facet element at %s:3:1",
					loc.URI, loc.Line, loc.Col, produceURI)
			}
		})
	}
}

// TestProduceFacetCountValueAdmitted pins the valid lexical forms no suite
// fixture spells for these facets, which a wrongly strict check would reject
// with nothing in the conformance lanes to notice: a leading "+", surrounding XML
// whitespace (collapsed before the lexical test, §4.1.4), and leading zeros.
// fractionDigits "0" is the nonNegativeInteger that totalDigits' narrower
// positiveInteger excludes.
func TestProduceFacetCountValueAdmitted(t *testing.T) {
	cases := []struct {
		name  string
		base  string
		facet string
	}{
		{"maxLength leading plus", "string", `<xs:maxLength value="+5"/>`},
		{"maxLength padded", "string", `<xs:maxLength value=" 5 "/>`},
		{"maxLength tab and newline padded", "string", `<xs:maxLength value="&#x9;5&#xA;"/>`},
		{"maxLength leading zeros", "string", `<xs:maxLength value="007"/>`},
		{"length leading plus", "string", `<xs:length value="+5"/>`},
		{"minLength leading zeros", "string", `<xs:minLength value="007"/>`},
		{"fractionDigits zero", "decimal", `<xs:fractionDigits value="0"/>`},
		{"fractionDigits padded", "decimal", `<xs:fractionDigits value=" 2 "/>`},
		{"totalDigits leading plus", "decimal", `<xs:totalDigits value="+5"/>`},
		{"totalDigits padded", "decimal", `<xs:totalDigits value=" 5 "/>`},
		{"totalDigits leading zeros", "decimal", `<xs:totalDigits value="007"/>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, facetValueSchema(tc.base, tc.facet)); err != nil {
				t.Fatalf("Produce: %v", err)
			}
		})
	}
}
