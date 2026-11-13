package parser_test

import (
	"testing"
)

// TestFacetValueOutsideLexicalSpaceRendersItsCause pins the whole rejection
// line of a bound or enumeration facet whose {value} is outside the value space
// its valid-restriction rule holds it to: the outer rule's message leads with
// its own fact — the facet, its {value} and the type whose value space it is
// not in — and closes on its own citation, and the wrapped verdict follows
// after ": " as its own "[rule] msg", with no "?: " for the verdict's unknown
// position (#3542, #3581, STYLE E5).
//
// The base-facet rows reach checkBoundAgainstBase's parse of a {base type
// definition}'s bound facet: D is checked before B, so B's malformed {value} is
// first noticed at D's position. The type named is the base of the type that
// DECLARES the facet, which for the inherited row is A, not D's base B.
func TestFacetValueOutsideLexicalSpaceRendersItsCause(t *testing.T) {
	const xs = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`
	named := func(base, facet string) string {
		return xs + `<xs:simpleType name="t"><xs:restriction base="` + base + `">` + facet + `</xs:restriction></xs:simpleType></xs:schema>`
	}
	for _, tc := range []struct {
		name, schema, want string
	}{
		{"maxInclusive", named("xs:integer", `<xs:maxInclusive value="abc"/>`),
			`main.xsd:1:56: [maxInclusive-valid-restriction] the maxInclusive facet's {value} "abc" is not in the value space of the simple type {http://www.w3.org/2001/XMLSchema}integer, which maxInclusive-valid-restriction requires: [cvc-datatype-valid] "abc" is not in the lexical space of decimal, which cvc-datatype-valid clause 2.1 requires it to be in`},
		{"minExclusive", named("xs:integer", `<xs:minExclusive value="abc"/>`),
			`main.xsd:1:56: [minExclusive-valid-restriction] the minExclusive facet's {value} "abc" is not in the value space of the simple type {http://www.w3.org/2001/XMLSchema}integer, which minExclusive-valid-restriction requires: [cvc-datatype-valid] "abc" is not in the lexical space of decimal, which cvc-datatype-valid clause 2.1 requires it to be in`},
		{"enumeration", named("xs:integer", `<xs:enumeration value="abc"/>`),
			`main.xsd:1:56: [enumeration-valid-restriction] the enumeration facet's {value} member "abc" is not in the value space of the simple type {http://www.w3.org/2001/XMLSchema}integer, which enumeration-valid-restriction requires: [cvc-pattern-valid] "abc" matches no member of the pattern facet of the simple type {http://www.w3.org/2001/XMLSchema}integer, whose {value} holds "[\-+]?[0-9]+", but cvc-pattern-valid requires one to match`},
		{"enumeration outside the base's value facets", named("xs:byte", `<xs:enumeration value="200"/>`),
			`main.xsd:1:56: [enumeration-valid-restriction] the enumeration facet's {value} member "200" is not in the value space of the simple type {http://www.w3.org/2001/XMLSchema}byte, which enumeration-valid-restriction requires: [cvc-maxInclusive-valid] "200" violates the maxInclusive facet of the simple type {http://www.w3.org/2001/XMLSchema}byte, whose {value} is "127", but cvc-maxInclusive-valid requires a value less than or equal to it`},
		{"enumeration on an anonymous base", xs + `<xs:simpleType name="t"><xs:restriction><xs:simpleType><xs:restriction base="xs:integer"/></xs:simpleType><xs:enumeration value="abc"/></xs:restriction></xs:simpleType></xs:schema>`,
			`main.xsd:1:56: [enumeration-valid-restriction] the enumeration facet's {value} member "abc" is not in the value space of an anonymous simple type, which enumeration-valid-restriction requires: [cvc-pattern-valid] "abc" matches no member of the pattern facet that an anonymous simple type inherits from the simple type {http://www.w3.org/2001/XMLSchema}integer, whose {value} holds "[\-+]?[0-9]+", but cvc-pattern-valid requires one to match`},
		{"base facet", xs + `<xs:simpleType name="D"><xs:restriction base="B"><xs:maxInclusive value="5"/></xs:restriction></xs:simpleType><xs:simpleType name="B"><xs:restriction base="xs:int"><xs:maxInclusive value="abc"/></xs:restriction></xs:simpleType></xs:schema>`,
			`main.xsd:1:56: [maxInclusive-valid-restriction] the maxInclusive facet's {value} "abc" is not in the value space of the {base type definition} of the simple type B, the type that declares it, which maxInclusive-valid-restriction requires: [cvc-datatype-valid] "abc" is not in the lexical space of decimal, which cvc-datatype-valid clause 2.1 requires it to be in`},
		{"inherited base facet", xs + `<xs:simpleType name="D"><xs:restriction base="B"><xs:maxInclusive value="5"/></xs:restriction></xs:simpleType><xs:simpleType name="B"><xs:restriction base="A"><xs:pattern value="\d+"/></xs:restriction></xs:simpleType><xs:simpleType name="A"><xs:restriction base="xs:int"><xs:minInclusive value="abc"/></xs:restriction></xs:simpleType></xs:schema>`,
			`main.xsd:1:56: [minInclusive-valid-restriction] the minInclusive facet's {value} "abc" is not in the value space of the {base type definition} of the simple type A, the type that declares it, which minInclusive-valid-restriction requires: [cvc-datatype-valid] "abc" is not in the lexical space of decimal, which cvc-datatype-valid clause 2.1 requires it to be in`},
	} {
		_, err := parseMap(t, "main.xsd", map[string]string{"main.xsd": tc.schema})
		if err == nil {
			t.Errorf("%s: parse = nil, want the facet's {value} rejected", tc.name)
			continue
		}
		if got := err.Error(); got != tc.want {
			t.Errorf("%s: err = %q, want %q", tc.name, got, tc.want)
		}
	}
}
