package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser"
)

// TestHintsOfSplitOnXMLSpaceAlone pins how hintsOf reads the two hint
// attributes. xsi:schemaLocation is a list of xs:anyURI (§3.2.7.3), delimited
// on XML white space alone (cvc-datatype-valid, Datatypes §4.1.4 clause 2.2),
// and xs:anyURI admits every XML Char, so a U+00A0 or U+2028 inside a location
// is part of it. xsi:noNamespaceSchemaLocation is ONE xs:anyURI (§3.2.7.4):
// its collapsed value is one location, whatever spaces it holds.
func TestHintsOfSplitOnXMLSpaceAlone(t *testing.T) {
	const base = "/d/inst.xml"
	for _, tc := range []struct {
		name, attrs string
		want        []parser.Root
	}{
		{"schemaLocation location holding U+00A0", `xsi:schemaLocation="urn:n a&#xA0;b.xsd"`,
			[]parser.Root{parser.HintAt("urn:n", "/d/a b.xsd")}},
		{"schemaLocation location holding U+2028", `xsi:schemaLocation="urn:n a&#x2028;b.xsd"`,
			[]parser.Root{parser.HintAt("urn:n", "/d/a b.xsd")}},
		{"noNamespaceSchemaLocation holding #x20", `xsi:noNamespaceSchemaLocation="a b.xsd"`,
			[]parser.Root{parser.HintAt("", "/d/a b.xsd")}},
		{"noNamespaceSchemaLocation collapsed", `xsi:noNamespaceSchemaLocation="&#9; a &#10;&#13; b.xsd "`,
			[]parser.Root{parser.HintAt("", "/d/a b.xsd")}},
		{"noNamespaceSchemaLocation holding U+00A0", `xsi:noNamespaceSchemaLocation="a&#xA0;b.xsd"`,
			[]parser.Root{parser.HintAt("", "/d/a b.xsd")}},
		{"noNamespaceSchemaLocation empty names nothing", `xsi:noNamespaceSchemaLocation=" "`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := `<r xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" ` + tc.attrs + `/>`
			got, _ := instanceHints(base, base, strings.NewReader(doc))
			if !slices.Equal(got, tc.want) {
				t.Errorf("hints = %v, want %v", got, tc.want)
			}
		})
	}
}
