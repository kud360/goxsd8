package strict_test

import (
	"strings"
	"testing"
)

// TestLexicalSpaceRejectionNamesTheDatatypeXS pins that a lexical-space
// rejection names its datatype in the XML Schema namespace as xs:local, the
// spelling a validate message gives the same type through xsd.QName.Display
// (#3688). Each row reaches a pair of a "lexical space of" message site and the
// datatype it names that no other row reaches, and fails if that site, or the
// caller's name literal a shared site (float and double, QName and NOTATION)
// prints, names the datatype bare.
func TestLexicalSpaceRejectionNamesTheDatatypeXS(t *testing.T) {
	for _, c := range []struct{ local, lex string }{
		{"hexBinary", "abc"},
		{"base64Binary", "!"},
		{"boolean", "yes"},
		{"dateTime", "x"},
		{"dateTime", "2001-02-30T00:00:00"},
		{"duration", "x"},
		{"decimal", "x"},
		{"float", "x"},
		{"double", "x"},
		{"time", "x"},
		{"date", "x"},
		{"date", "2001-02-30"},
		{"gYearMonth", "x"},
		{"gYear", "x"},
		{"gMonthDay", "x"},
		{"gMonthDay", "--02-30"},
		{"gDay", "x"},
		{"gMonth", "x"},
		{"precisionDecimal", "x"},
		{"QName", "1a"},
		{"QName", "a:b:c"},
		{"NOTATION", "1a"},
	} {
		_, err := mappingFor(t, c.local).Parse(c.lex, nil)
		if err == nil {
			t.Errorf("xs:%s Parse(%q) = nil error, want a lexical-space rejection", c.local, c.lex)
			continue
		}
		if want := "lexical space of xs:" + c.local + ","; !strings.Contains(err.Error(), want) {
			t.Errorf("xs:%s Parse(%q) = %q, want it to contain %q", c.local, c.lex, err.Error(), want)
		}
	}
}
