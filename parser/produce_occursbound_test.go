package parser_test

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// TestProduceOccursAboveMaxIntAdmitted pins that a minOccurs/maxOccurs literal
// past math.MaxInt is datatype-valid: xs:nonNegativeInteger, and the numeric
// member of xs:allNNI with it, is unbounded (Datatypes §3.4.20), so
// cvc-datatype-valid cannot reject a literal for its size (#1780). The value
// reaches the particle saturated at math.MaxInt, never at xsd.Occurs' unbounded
// sentinel.
func TestProduceOccursAboveMaxIntAdmitted(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name      string
		attrs     string
		wantMin   int
		wantMax   int
		unbounded bool
	}{
		{
			// particlesZ033_a's element: both past MaxInt, min below max.
			name:    "particlesZ033_a pair",
			attrs:   `minOccurs="79228162514244337593543950335" maxOccurs="79228162514264337593543950335"`,
			wantMin: math.MaxInt,
			wantMax: math.MaxInt,
		},
		{
			name:    "maxOccurs alone past MaxInt",
			attrs:   `maxOccurs="99999999999999999999"`,
			wantMin: 1,
			wantMax: math.MaxInt,
		},
		{
			// Equal values spelled differently: leading zeros and a sign carry no
			// value, so clause 2.1 sees min == max.
			name:    "equal values past MaxInt, one spelled with a sign and leading zeros",
			attrs:   `minOccurs="+00099999999999999999999" maxOccurs="99999999999999999999"`,
			wantMin: math.MaxInt,
			wantMax: math.MaxInt,
		},
		{
			name:      "minOccurs past MaxInt with maxOccurs unbounded",
			attrs:     `minOccurs="99999999999999999999" maxOccurs="unbounded"`,
			wantMin:   math.MaxInt,
			unbounded: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ct := complexType(t, `<xs:complexType name="CT"><xs:sequence>`+
				`<xs:element name="e" `+tc.attrs+`/>`+
				`</xs:sequence></xs:complexType>`, "CT")
			ps := topGroup(t, ct).Particles()
			if len(ps) != 1 {
				t.Fatalf("particles = %d, want 1", len(ps))
			}
			occ := ps[0].Occurs()
			max, bounded := occ.Max()
			if occ.Min() != tc.wantMin || bounded == tc.unbounded || (bounded && max != tc.wantMax) {
				t.Fatalf("e occurs = %s, want min %d, max %d (unbounded=%v)", occ, tc.wantMin, tc.wantMax, tc.unbounded)
			}
		})
	}
}

// TestProduceOccursAboveMaxIntOrderKept pins that saturating at math.MaxInt does
// not lose p-props-correct clause 2.1 (§3.9.6.1): {min occurs} greater than a
// numeric {max occurs} is still charged when both values saturate to the same
// host int, and every row names the document's values, never the saturated
// math.MaxInt (STYLE E1, #1780). The message is pinned whole, so a swap of its
// two operands fails the row.
func TestProduceOccursAboveMaxIntOrderKept(t *testing.T) {
	maxInt := strconv.Itoa(math.MaxInt)
	maxIntPlusOne := strconv.FormatUint(uint64(math.MaxInt)+1, 10)
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name    string
		attrs   string
		wantMsg string
	}{
		{
			// Both past MaxInt: saturated, the pair would be MaxInt, MaxInt.
			name:    "both past MaxInt, min greater",
			attrs:   `minOccurs="79228162514264337593543950335" maxOccurs="79228162514244337593543950335"`,
			wantMsg: "particle {min occurs} 79228162514264337593543950335 is greater than {max occurs} 79228162514244337593543950335",
		},
		{
			// The longer numeral is the larger although it orders first as a
			// string: the comparison is numeric, not lexicographic.
			name:    "both past MaxInt, min greater by length alone",
			attrs:   `minOccurs="100000000000000000000" maxOccurs="99999999999999999999"`,
			wantMsg: "particle {min occurs} 100000000000000000000 is greater than {max occurs} 99999999999999999999",
		},
		{
			// maxOccurs exactly MaxInt, minOccurs past it: the one pair where only
			// min saturates and the two still meet.
			name:    "maxOccurs exactly MaxInt, min past it",
			attrs:   `minOccurs="` + maxIntPlusOne + `" maxOccurs="` + maxInt + `"`,
			wantMsg: "particle {min occurs} " + maxIntPlusOne + " is greater than {max occurs} " + maxInt,
		},
		{
			// maxOccurs below MaxInt: saturation keeps the order but would name
			// MaxInt, a value the document never spelled.
			name:    "min past MaxInt, max small",
			attrs:   `minOccurs="99999999999999999999" maxOccurs="5"`,
			wantMsg: "particle {min occurs} 99999999999999999999 is greater than {max occurs} 5",
		},
		{
			// Neither saturates: the message xsd.NewOccurs would give.
			name:    "both small, min greater",
			attrs:   `minOccurs="7" maxOccurs="5"`,
			wantMsg: "particle {min occurs} 7 is greater than {max occurs} 5",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", `<xs:complexType name="CT"><xs:sequence>`+
				`<xs:element name="e" `+tc.attrs+`/>`+
				`</xs:sequence></xs:complexType>`))
			assertRule(t, err, "p-props-correct")
			if !strings.Contains(err.Error(), "] "+tc.wantMsg) {
				t.Fatalf("error = %v, want the message %q", err, tc.wantMsg)
			}
		})
	}
}

// TestProduceAllOccursAboveMaxIntOutsideEnumeration pins that <all>'s {0,1}
// enumeration reads a literal past math.MaxInt as a valid xs:nonNegativeInteger
// outside the enumeration, not as a lexical that fails its base type (#1780).
func TestProduceAllOccursAboveMaxIntOutsideEnumeration(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:complexType name="CT">`+
		`<xs:all maxOccurs="99999999999999999999"><xs:element name="a" type="xs:string"/></xs:all>`+
		`</xs:complexType>`))
	assertRule(t, err, "cvc-datatype-valid")
	if !strings.Contains(err.Error(), "is outside the enumeration 0, 1") {
		t.Fatalf("error = %v, want the {0,1} enumeration verdict, not a base-type one", err)
	}
}
