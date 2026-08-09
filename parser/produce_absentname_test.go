package parser_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceAbsentNameAndEmptyRefRejected pins, END TO END from a schema
// DOCUMENT, four shapes an author can write with no usable name: a nameless
// <element> on the local path (e-props-correct clause 1, §3.3.6.1 — the §3.3.1
// tableau types {name} as a Required xs:NCName, and §3.3.2.1 dcl.elt.common maps
// it identically for both {scope} varieties, so neither path may map an empty
// one), and the three empty-ref shapes <group ref="">, <element ref=""> and
// <attribute ref="">.
//
// The GLOBAL path's row moved out at #305: a top-level <element> now takes its
// {name} from the parser's topLevelName, which rejects an unusable one as a
// grammar fault before the declaration is built, so that shape never reaches
// the constructor and is pinned by TestProduceNamelessTopLevelRejected instead.
// The constructor's own e-props-correct verdict is unchanged and still covered
// here, by the local row — its single remaining parser-visible call site for a
// missing {name}.
//
// THE THREE REF ROWS WERE RECLASSIFIED AT #343, from the xsderr.RuleComponentInvariant
// they carried since #202 to cvc-datatype-valid (Datatypes §4.1.4). The earlier
// reasoning weighed only the MAPPING rules (§3.2.2.3 ref.att.local, §3.3.2.4
// ref.elt.global, §3.8.2), which state no "ref must be non-empty" clause, and
// src-resolve (§3.17.6.2), which presupposes a well-formed QName and governs
// ·resolution· alone; it never weighed the schema for schema documents itself.
// That is where the rule lives: Appendix A types ref as xs:QName, whose ·lexical
// space· (§3.3.18) admits only a QName, whose local part is an NCName (§3.4.7.1)
// and is never empty, and Structures §5.1 requires the document to be valid
// against that schema. So ref="" IS a numbered-rule violation after all, charged
// by the producer at the attribute the author wrote (bindQName), and the sentinel
// no longer applies here by its own godoc precondition — the state is plainly
// reachable from a schema document.
//
// The xsd constructor guards (xsd/elementdeclaration.go's absent {name} guard,
// xsd/particle.go's two ref-term guards, xsd/attributeuse.go's ref guard) STAY
// and keep charging RuleComponentInvariant; they are the backstop for a
// programmatic caller building components directly, whose misuse IS a caller
// fault. What changed is that the parser no longer trips them: these rows now
// pin the producer's own lexical verdict, and only the nameless-<element> row
// still pins a propagated constructor verdict (reverting
// xsd/elementdeclaration.go's guard makes it fail).
//
// Behavior is pinned, never message text: each case asserts the document is
// rejected, that xsderr.RuleOf reports the expected rule, and that the error is
// positioned at URI:line:col on the offending element's own line (STYLE E3).
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
			// The ModelGroupRef this would have minted is never built: the lexical
			// verdict lands first (#343).
			name: `<group ref=""> in a content model`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:group ref=""/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			// Likewise for the ElementDeclarationRef term (#343).
			name: `<element ref=""> in a content model`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:element ref=""/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantRule: "cvc-datatype-valid",
			wantLine: 3,
		},
		{
			// Likewise for the attribute use's AttributeDeclarationRef (#343).
			name: `<attribute ref=""> on a complex type`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence/>` + "\n" +
				`<xs:attribute ref=""/>` + "\n" +
				`</xs:complexType>`,
			wantRule: "cvc-datatype-valid",
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

// TestProduceEmptyQNameLocalPartRejected pins bindQName's #343 lexical check —
// a QName-valued schema attribute whose LOCAL PART is empty is not in the
// ·lexical space· of xs:QName (Datatypes §3.3.18, whose local part is the NCName
// of §3.4.7.1), the type Appendix A declares for every such attribute, so it
// fails cvc-datatype-valid (§4.1.4) at the attribute the author wrote rather than
// travelling on as the zero xsd.QName.
//
// One row per CODE PATH into the check, not per attribute — every QName-valued
// attribute in the producer reaches it through one of these three, and the
// sibling table above covers ref= on all three of its element shapes:
//
//   - resolveQName over a whole attribute value (type=, and refer=, which before
//     #343 was a silent FALSE ACCEPT: <keyref refer=""> mapped to a {referenced
//     key} holding the zero QName and no rejection ever came);
//   - resolveQName over one item of a whitespace-separated list
//     (substitutionGroup, whose items §3.3.2's xs:QName list type governs
//     individually);
//   - bindQName DIRECTLY, the notQName path (§3.10.2's {disallowed names} takes
//     its items as QName VALUES and never ·resolves· them), which proves the
//     check sits in bindQName rather than in resolveQName's licensing half.
//
// Each prefixed row spells a BOUND prefix (xs:), so a passing row cannot be the
// unbound-prefix src-resolve rejection wearing this rule's name; and the two
// lexical shapes the grounding calls the same fault — the whole value empty and
// a prefix with nothing after the colon — are both exercised, since strings.Cut
// reaches an empty local part by two different routes.
func TestProduceEmptyQNameLocalPartRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts on
	// line 2 of the wrapped document and puts the offending element on line 3.
	cases := []struct {
		name string
		body string
	}{
		{
			name: `type="" on a top-level <element>`,
			body: "\n" + `<xs:element name="e" type=""/>`,
		},
		{
			name: `type="xs:" on a top-level <element>`,
			body: "\n" + `<xs:element name="e" type="xs:"/>`,
		},
		{
			name: `refer="" on a <keyref>`,
			body: "\n" + `<xs:element name="e" type="xs:string">` + "\n" +
				`<xs:keyref name="k" refer=""><xs:selector xpath="."/><xs:field xpath="."/></xs:keyref>` + "\n" +
				`</xs:element>`,
		},
		{
			name: `substitutionGroup item with an empty local part`,
			body: "\n" + `<xs:element name="e" type="xs:string" substitutionGroup="xs:"/>`,
		},
		{
			name: `notQName item with an empty local part`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:any notQName="xs:"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("", tc.body))
			assertRule(t, err, "cvc-datatype-valid")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:line:col (E3)", err, produceURI)
			}
			if loc.URI != produceURI || loc.Line == 0 || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the offending element in %s with a line and column",
					loc.URI, loc.Line, loc.Col, produceURI)
			}
		})
	}
}
