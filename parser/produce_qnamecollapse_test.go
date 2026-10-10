package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// qnamePads are the paddings a QName-valued attribute's ·actual value· sheds
// under xs:QName's fixed whiteSpace = collapse (Datatypes §3.3.18.1). Only the
// character-reference row delivers a raw tab, CR or LF to bindQName: XML
// attribute-value normalization (XML 1.0 §3.3.3) turns a LITERAL one into #x20
// before the producer sees the string. The literal row is coverage, not a pin
// of that normalization, which collapse would hide; parser/xmltree's
// TestAttributeValueIsNormalized pins it.
var qnamePads = []struct {
	name, before, after string
}{
	{"leading spaces", "    ", ""},
	{"trailing space", "", " "},
	{"both ends, addB106's shape", "    ", " "},
	{"tab, CR and LF references", "&#9;&#13;&#10;", "&#10;&#9;&#13;"},
	{"literal tab, CR and LF", "\t\r\n", "\n\t\r"},
}

// TestProduceQNameAttributeCollapsed reads every padding in qnamePads on base=
// and on type=, and asserts the reference resolves to xs:string. Before
// bindQName collapsed its lexical, each row was charged src-resolve: the
// leading pad as an unbound prefix ("    xs"), the trailing one as a local part
// ("string ") naming no type.
func TestProduceQNameAttributeCollapsed(t *testing.T) {
	stringQN := xsd.QName{Space: xsdNS, Local: "string"}
	for _, pad := range qnamePads {
		lex := pad.before + "xs:string" + pad.after
		t.Run("simpleType restriction base, "+pad.name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:simpleType name="T"><xs:restriction base="`+lex+`"/></xs:simpleType>`))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			if got := mustBase(t, s, mustSimpleType(t, s, xsd.QName{Local: "T"})).Name(); got != stringQN {
				t.Fatalf("{base type definition} = %s, want %s", got, stringQN)
			}
		})
		t.Run("simpleContent extension base, "+pad.name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:complexType name="C"><xs:simpleContent><xs:extension base="`+lex+`"/></xs:simpleContent></xs:complexType>`))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			if got := declaredTypeName(t, mustComplexType(t, s, xsd.QName{Local: "C"}).Base()); got != stringQN {
				t.Fatalf("{base type definition} = %s, want %s", got, stringQN)
			}
		})
		t.Run("element type, "+pad.name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:element name="e" type="`+lex+`"/>`))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			ed, ok := s.Element(xsd.QName{Local: "e"})
			if !ok {
				t.Fatalf("element e not found")
			}
			if got := declaredTypeName(t, ed.TypeDefinition()); got != stringQN {
				t.Fatalf("{type definition} = %s, want %s", got, stringQN)
			}
		})
		t.Run("attribute type, "+pad.name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:attribute name="a" type="`+lex+`"/>`))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			ad, ok := s.Attribute(xsd.QName{Local: "a"})
			if !ok {
				t.Fatalf("attribute a not found")
			}
			if got := declaredTypeName(t, ad.TypeDefinition()); got != stringQN {
				t.Fatalf("{type definition} = %s, want %s", got, stringQN)
			}
		})
	}
}

// TestProduceQNameAttributeCollapseKeepsInteriorAndNonXMLSpace pins what
// collapse does NOT remove. An interior space survives it (collapse folds a run
// to one #x20, it deletes none), and U+00A0 is not §4.3.6 whitespace at all, so
// strings.TrimSpace in place of collapseTrim would accept the third row.
//
// Each is charged src-resolve, not cvc-datatype-valid: bindQName's GAP(parser)
// marker leaves the full NCName test on each half unapplied, so the interior
// space reaches resolution as a local part no type carries and the U+00A0 as a
// prefix nothing binds.
func TestProduceQNameAttributeCollapseKeepsInteriorAndNonXMLSpace(t *testing.T) {
	cases := []struct {
		name, lex, wantMsg string
	}{
		{"interior space", "xs: string", "no type definition with that expanded name"},
		{"interior tab reference", "xs:&#9;string", "no type definition with that expanded name"},
		{"U+00A0 pad", "&#xA0;xs:string", "does not resolve to an in-scope namespace"},
	}
	for _, tc := range cases {
		t.Run("base, "+tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", `<xs:simpleType name="T"><xs:restriction base="`+tc.lex+`"/></xs:simpleType>`))
			mustRule(t, err, "src-resolve", tc.wantMsg)
		})
		t.Run("type, "+tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", `<xs:element name="e" type="`+tc.lex+`"/>`))
			mustRule(t, err, "src-resolve", tc.wantMsg)
		})
	}
}
