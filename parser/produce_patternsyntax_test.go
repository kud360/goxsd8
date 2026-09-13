package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestParsePatternSyntaxRejectedWithoutAnyInstance is the reach test for
// src-pattern-value (§4.3.4.3): Parse alone rejects a malformed <pattern>
// value, with no instance document anywhere and nothing ever validated against
// the type. That is the whole defect #1411 fixes — the facet-compile path
// value/facets.go charges the same rule from is lazy, so before the finalize
// walk gained the eager charge these schemas parsed clean and the suite's
// simple041/simple042/elemE007-9 fixtures were accepted.
//
// The anonymous case matters on its own: an inline <simpleType> is in no index
// a by-name walk could consult, so a check that reached only the schema's named
// types would pass this and still leave the hole.
func TestParsePatternSyntaxRejectedWithoutAnyInstance(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"named type", `<xs:simpleType name="t">
			<xs:restriction base="xs:string"><xs:pattern value="[--z]*"/></xs:restriction>
		 </xs:simpleType>`},
		{"anonymous inline type", `<xs:element name="root">
			<xs:simpleType>
				<xs:restriction base="xs:string"><xs:pattern value="[--z]*"/></xs:restriction>
			</xs:simpleType>
		 </xs:element>`},
		{"anonymous attribute type", `<xs:attribute name="a">
			<xs:simpleType>
				<xs:restriction base="xs:string"><xs:pattern value="[0-9]{,5}"/></xs:restriction>
			</xs:simpleType>
		 </xs:attribute>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": wrap("urn:a", c.body)})
			if err == nil {
				t.Fatal("Parse accepted a schema whose pattern facet is no regular expression")
			}
			rule, ok := xsderr.RuleOf(err)
			if !ok || rule != "src-pattern-value" {
				t.Fatalf("rule = %q (ok=%v), want src-pattern-value; err=%v", rule, ok, err)
			}
			loc, ok := xsderr.LocOf(err)
			if !ok || loc.Line == 0 {
				t.Errorf("loc = %v (ok=%v), want the offending type's position", loc, ok)
			}
		})
	}
}

// TestParseUnsupportedUnicodeBlockPatternAccepted is the false-rejection guard
// on the same walk, and the reason regex.CheckSyntax exists rather than
// regex.Translate being called directly: \p{IsThai} is a perfectly good
// Appendix G pattern that this module cannot yet compile (GAP(regex), #1473),
// and 208 of the pattern values in testdata/xsdtests are of that shape. A check
// that rejected on any translation failure would newly reject every one of
// those schemas.
func TestParseUnsupportedUnicodeBlockPatternAccepted(t *testing.T) {
	body := `<xs:simpleType name="t">
		<xs:restriction base="xs:string"><xs:pattern value="\p{IsThai}*"/></xs:restriction>
	 </xs:simpleType>`
	if _, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": wrap("urn:a", body)}); err != nil {
		t.Fatalf("Parse rejected a spec-valid pattern for a gap in this module: %v", err)
	}
}

// TestParsePatternSyntaxChargedAtTheDeclaringType pins the attribution the
// OwnFacets operand buys. Two types derive from the one that declares the bad
// pattern and are written BEFORE it, so a check reading the accumulated
// {facets} would charge one of them — the pattern is in their {facets} too —
// and report a position whose <pattern> element the author never wrote. The
// declaring type's own line is the only defensible answer.
func TestParsePatternSyntaxChargedAtTheDeclaringType(t *testing.T) {
	body := "\n" +
		`<xs:simpleType name="d2"><xs:restriction base="tns:d1"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>` + "\n" +
		`<xs:simpleType name="d1"><xs:restriction base="tns:bad"><xs:maxLength value="4"/></xs:restriction></xs:simpleType>` + "\n" +
		`<xs:simpleType name="bad"><xs:restriction base="xs:string"><xs:pattern value="[--z]*"/></xs:restriction></xs:simpleType>`
	const declaringLine = 4
	_, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": wrap("urn:a", body)})
	if err == nil {
		t.Fatal("Parse accepted the schema")
	}
	loc, ok := xsderr.LocOf(err)
	if !ok || loc.Line != declaringLine {
		t.Fatalf("loc = %v (ok=%v), want line %d — the type that DECLARES the pattern; err=%v",
			loc, ok, declaringLine, err)
	}
	if !strings.Contains(err.Error(), `"[--z]*"`) {
		t.Errorf("message %q does not quote the offending value", err.Error())
	}
}
