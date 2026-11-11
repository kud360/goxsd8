package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceOccursAttributeFaultsCharged pins which rule an invalid
// minOccurs/maxOccurs is charged, drawing the line at whether a Particle exists
// to be constrained (#932).
//
// The schema for schema documents declares minOccurs type="xs:nonNegativeInteger"
// and maxOccurs type="xs:allNNI" in the occurs attribute group "for all
// particles" (xmlschema11-1.md:4627-4634), and <all> narrows both by an
// enumeration admitting only 0 and 1 (§3.8.2). A lexical outside either of those
// declared types is §5.1's first bullet — the document is not valid against the
// Schema for Schema Documents — charged cvc-datatype-valid (Datatypes §4.1.4),
// the convention parser/produce.go's ruleDatatypeValid doc comment states for a
// schema document attribute that fails its own declared type with no Structures
// Schema Representation Constraint covering the case. The producer never reaches
// xsd.NewOccurs on these rows: occursOf returns the lexical verdict first, so no
// Occurs and no Particle is ever built.
//
// p-props-correct is a Schema Component Constraint over a particle's PROPERTIES
// (§3.9.6.1): clause 1 reads "the values of the properties of a particle", and
// clause 2 is conditioned on {max occurs} having a numeric value before clause
// 2.1 compares it to {min occurs}. It therefore needs both occurrence values
// already parsed, which is exactly and only the min-greater-than-max row — the
// control below, charged by occursOf once both values are read.
//
// Every row REJECTS, before and after #932: this table pins the rule ID, which a
// test asserting only that an error occurred cannot see.
func TestProduceOccursAttributeFaultsCharged(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts
	// on line 2 of the wrapped document and puts the offending element, and so
	// the faulted attribute a lexical charge is located at, on the line named by
	// wantLine; TestProduceAttributeValueChargeLocated pins the column.
	cases := []struct {
		name     string
		body     string
		wantRule xsderr.Rule
		wantLine int
	}{
		{
			// The <all> enumeration path: allOccursGrammar -> allOccursEnum ->
			// nonNegativeInt, which fails on the base type before the {0,1}
			// enumeration is ever consulted.
			name: `<all> maxOccurs with a non-numeric lexical`,
			body: "\n" + `<xs:complexType name="CT">` + "\n" +
				`<xs:all maxOccurs="y"><xs:element name="a" type="xs:string"/></xs:all>` + "\n" +
				`</xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			name: `<all> minOccurs with a non-numeric lexical`,
			body: "\n" + `<xs:complexType name="CT">` + "\n" +
				`<xs:all minOccurs="x"><xs:element name="a" type="xs:string"/></xs:all>` + "\n" +
				`</xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			// Negative is a lexical fault too: xs:nonNegativeInteger's value space
			// excludes it, so no {min occurs} is produced.
			name: `<all> minOccurs="-1"`,
			body: "\n" + `<xs:complexType name="CT">` + "\n" +
				`<xs:all minOccurs="-1"><xs:element name="a" type="xs:string"/></xs:all>` + "\n" +
				`</xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			// The occursOf path, which no enumeration narrows: "unbounded" is a
			// member of xs:allNNI and so legal on maxOccurs, but minOccurs is
			// declared xs:nonNegativeInteger and never admits it.
			name: `<sequence> minOccurs="unbounded"`,
			body: "\n" + `<xs:complexType name="CT">` + "\n" +
				`<xs:sequence minOccurs="unbounded"><xs:element name="a" type="xs:string"/></xs:sequence>` + "\n" +
				`</xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			name: `<sequence> minOccurs with a non-numeric lexical`,
			body: "\n" + `<xs:complexType name="CT">` + "\n" +
				`<xs:sequence minOccurs="x"><xs:element name="a" type="xs:string"/></xs:sequence>` + "\n" +
				`</xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			// The nested <element> position takes the same occursOf path, and the
			// charge is positioned at the minOccurs the <element> carries.
			name: `nested <element> minOccurs with a non-numeric lexical`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:element name="a" type="xs:string" minOccurs="x"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			// UNCHANGED CONTROL. A valid nonNegativeInteger outside <all>'s {0,1}
			// enumeration was already charged cvc-datatype-valid, by allOccursEnum,
			// and this row is what makes the two faults on one attribute agree.
			name: `<all> maxOccurs="2", a valid integer outside the {0,1} enumeration`,
			body: "\n" + `<xs:complexType name="CT">` + "\n" +
				`<xs:all maxOccurs="2"><xs:element name="a" type="xs:string"/></xs:all>` + "\n" +
				`</xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			// UNCHANGED CONTROL, and the one row whose particle has properties to
			// constrain: both lexicals parse, and occursOf charges p-props-correct
			// clause 2.1 on the range they give. #901's landing depends on this
			// charge staying put.
			name: `<sequence> minOccurs greater than maxOccurs`,
			body: "\n" + `<xs:complexType name="CT">` + "\n" +
				`<xs:sequence minOccurs="2" maxOccurs="1"><xs:element name="a" type="xs:string"/></xs:sequence>` + "\n" +
				`</xs:complexType>`,
			wantRule: "p-props-correct",
			wantLine: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			assertRule(t, err, tc.wantRule)
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:%d:col (STYLE E3)", err, produceURI, tc.wantLine)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the charge on the offending element's line %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}

// TestProduceOccursFaultMessageCitesItsDeclaredType pins nonNegativeNumeral's
// whole message at each of its three callers — occursOf, allOccursEnum and the
// redefine self-reference check: the attribute, its lexical and the type the
// schema for schema documents declares for it, then the cvc-datatype-valid
// clause it fails (STYLE E4/E5, #3581). minOccurs is xs:nonNegativeInteger,
// clause 1 outside xs:integer's pattern and clause 3 below minInclusive 0;
// maxOccurs is the union xs:allNNI, clause 2.3. It names no p-props-correct,
// which has no particle to constrain here (#932).
func TestProduceOccursFaultMessageCitesItsDeclaredType(t *testing.T) {
	seq := func(attrs string) string {
		return `<xs:complexType name="CT"><xs:sequence ` + attrs + `>` +
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`
	}
	all := func(attrs string) string {
		return `<xs:complexType name="CT"><xs:all ` + attrs + `>` +
			`<xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`
	}
	redefine := func(attrs string) map[string]string {
		return map[string]string{
			"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
				`<xs:group name="g"><xs:sequence><xs:group ref="tns:g" `+attrs+`/></xs:sequence></xs:group>`+
				`</xs:redefine>`),
			"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
				`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
		}
	}
	cases := []struct {
		name    string
		body    string
		docs    map[string]string
		wantMsg string
	}{
		{"occursOf minOccurs non-numeric", seq(`minOccurs="x"`), nil,
			`minOccurs value "x" is not a valid xs:nonNegativeInteger, which cvc-datatype-valid clause 1 requires`},
		{"occursOf minOccurs negative", seq(`minOccurs="-1"`), nil,
			`minOccurs value "-1" is not a valid xs:nonNegativeInteger, which cvc-datatype-valid clause 3 requires`},
		{"occursOf maxOccurs non-numeric", seq(`maxOccurs="x"`), nil,
			`maxOccurs value "x" is not a valid xs:allNNI, which cvc-datatype-valid clause 2.3 requires`},
		{"occursOf maxOccurs negative", seq(`maxOccurs="-1"`), nil,
			`maxOccurs value "-1" is not a valid xs:allNNI, which cvc-datatype-valid clause 2.3 requires`},
		{"allOccursEnum minOccurs negative", all(`minOccurs="-1"`), nil,
			`minOccurs value "-1" is not a valid xs:nonNegativeInteger, which cvc-datatype-valid clause 3 requires`},
		{"allOccursEnum maxOccurs non-numeric", all(`maxOccurs="x"`), nil,
			`maxOccurs value "x" is not a valid xs:allNNI, which cvc-datatype-valid clause 2.3 requires`},
		{"redefine self-reference minOccurs negative", "", redefine(`minOccurs="-1"`),
			`minOccurs value "-1" is not a valid xs:nonNegativeInteger, which cvc-datatype-valid clause 3 requires`},
		{"redefine self-reference maxOccurs non-numeric", "", redefine(`maxOccurs="x"`),
			`maxOccurs value "x" is not a valid xs:allNNI, which cvc-datatype-valid clause 2.3 requires`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.docs != nil {
				_, err = parseMap(t, "main.xsd", tc.docs)
			} else {
				_, err = produce(t, wrap("", tc.body))
			}
			assertRule(t, err, "cvc-datatype-valid")
			msg := err.Error()
			if !strings.HasSuffix(msg, ": [cvc-datatype-valid] "+tc.wantMsg) {
				t.Fatalf("diagnostic %q, want its whole message to be %q", msg, tc.wantMsg)
			}
		})
	}
}
