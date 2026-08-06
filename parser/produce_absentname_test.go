package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceAbsentNameAndEmptyRefRejected pins, END TO END from a schema
// DOCUMENT, the four shapes whose only rejection lives in an xsd component
// constructor: a nameless <element> on the local path (e-props-correct clause 1,
// §3.3.6.1 — the §3.3.1 tableau types {name} as a Required xs:NCName, and
// §3.3.2.1 dcl.elt.common maps it identically for both {scope} varieties, so
// neither path may map an empty one), and the three empty-ref shapes whose
// reference variant carries the absent (zero) QName: <group ref="">,
// <element ref=""> and <attribute ref="">.
//
// The GLOBAL path's row moved out at #305: a top-level <element> now takes its
// {name} from the parser's topLevelName, which rejects an unusable one as a
// grammar fault before the declaration is built, so that shape never reaches
// the constructor and is pinned by TestProduceNamelessTopLevelRejected instead.
// The constructor's own e-props-correct verdict is unchanged and still covered
// here, by the local row — its single remaining parser-visible call site for a
// missing {name}.
//
// The three ref shapes are charged xsderr.RuleComponentInvariant, the non-spec
// sentinel: §3.2.2.3 ref.att.local, §3.3.2.4 ref.elt.global and §3.8.2 are
// MAPPING rules stating no "ref must be non-empty" clause to violate, and
// src-resolve (§3.17.6.2) presupposes a non-absent QName rather than covering an
// absent one. Borrowing any of them would be a fabricated verdict (STYLE E2).
//
// The rejection deliberately lives in the xsd constructor, NOT in the parser's
// resolveQName — one fact, one encoding (STYLE D3/D4). resolveQName maps ref=""
// to QName{Local: ""} without error, so what these tests pin is that the parser
// PROPAGATES the constructor's verdict, positioned on the offending element; they
// do not ask the parser to re-check the same fact.
//
// Behavior is pinned, never message text: each case asserts the document is
// rejected, that xsderr.RuleOf reports the expected rule, and that the error is
// positioned at URI:line:col on the offending element's own line (STYLE E3).
// Reverting the three xsd constructor checks (xsd/elementdeclaration.go's absent
// {name} guard, xsd/particle.go's two ref-term guards, xsd/attributeuse.go's ref
// guard) to their pre-#202 form makes every case here fail.
func TestProduceAbsentNameAndEmptyRefRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts
	// on line 2 of the wrapped document and puts the offending element on the
	// line named by wantLine, so the charged position is pinned exactly.
	cases := []struct {
		name     string
		body     string
		wantRule xsderr.Rule
		wantLine int
	}{
		{
			// §3.3.2.3 dcl.elt.local: the local path, produceLocalElement.
			// src-element clause 2.1's name-XOR-ref representation constraint is a
			// different rule and is deliberately not what is asserted here.
			name: `nameless local <element>`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:element name="" type="xs:string"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantRule: "e-props-correct",
			wantLine: 3,
		},
		{
			// Particle {term} = ModelGroupRef with the absent (zero) QName.
			name: `<group ref=""> in a content model`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:group ref=""/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantRule: xsderr.RuleComponentInvariant,
			wantLine: 3,
		},
		{
			// Particle {term} = ElementDeclarationRef with the absent QName.
			name: `<element ref=""> in a content model`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:element ref=""/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantRule: xsderr.RuleComponentInvariant,
			wantLine: 3,
		},
		{
			// Attribute use {attribute declaration} = AttributeDeclarationRef with
			// the absent QName.
			name: `<attribute ref=""> on a complex type`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence/>` + "\n" +
				`<xs:attribute ref=""/>` + "\n" +
				`</xs:complexType>`,
			wantRule: xsderr.RuleComponentInvariant,
			wantLine: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			assertRule(t, err, tc.wantRule)
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:%d:col (E3)", err, produceURI, tc.wantLine)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the offending element at %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}
