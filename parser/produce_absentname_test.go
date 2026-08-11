package parser_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceTopLevelRefProhibited pins, END TO END from a schema DOCUMENT, that
// a top-level <element> or <attribute> carrying ref is rejected FOR THE ref —
// xs:topLevelElement and xs:topLevelAttribute both restrict it to
// use="prohibited" (xmlschema11-1.md:5100, :4710), so the author's mistake is
// the attribute, not the name it displaces.
//
// The fault is a plain grammar fault, never a rule verdict: src-element clause 2
// and src-attribute clause 3 state the ref/name exclusivity inside a "if the
// item's parent is not <schema>" antecedent that excludes exactly this case, and
// the grammar reaches the two constraints only through their unnumbered
// preamble, so charging src-element, src-attribute, e-props-correct or
// a-props-correct would be fabricated (STYLE E2).
//
// Two row shapes, and each carries its own failure mode:
//
//   - ref ALONE — the seven suite witnesses' shape, which is rejected either
//     way, so its assertions are the message ones: the diagnostic must name ref
//     and must NOT be topLevelName's "no usable name", which is what answers if
//     the guard runs in the wrong order or not at all;
//   - ref AND name TOGETHER — where topLevelName is satisfied and nothing else
//     in the producer reads ref, so without the guard the document PRODUCES and
//     the row fails on the verdict, not on the wording.
//
// Every row DECLARES the target its ref names, so no row can pass as a dangling
// reference, and the position assertion pins the offending element's own line
// (STYLE E3, carried in the message text since a plain error holds no
// xsderr.Loc). That ref stays legal on the LOCAL forms is not re-pinned here —
// TestProduceAttributeRefUse and the substitution-group tables produce documents
// full of local ref=, and a widened guard would fail them.
func TestProduceTopLevelRefProhibited(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts
	// on line 2 of the wrapped document.
	cases := []struct {
		name     string
		body     string
		wantLine int
	}{
		{
			name: `top-level <element ref=>`,
			body: "\n" + `<xs:element name="head" type="xs:string"/>` + "\n" +
				`<xs:element ref="tns:head"/>`,
			wantLine: 3,
		},
		{
			name: `top-level <element ref=> with a name of its own`,
			body: "\n" + `<xs:element name="head" type="xs:string"/>` + "\n" +
				`<xs:element name="other" ref="tns:head" type="xs:string"/>`,
			wantLine: 3,
		},
		{
			name:     `top-level <element ref="">`,
			body:     "\n" + `<xs:element ref=""/>`,
			wantLine: 2,
		},
		{
			name: `top-level <attribute ref=>`,
			body: "\n" + `<xs:attribute name="head" type="xs:string"/>` + "\n" +
				`<xs:attribute ref="tns:head"/>`,
			wantLine: 3,
		},
		{
			name: `top-level <attribute ref=> with a name of its own`,
			body: "\n" + `<xs:attribute name="head" type="xs:string"/>` + "\n" +
				`<xs:attribute name="other" ref="tns:head" type="xs:string"/>`,
			wantLine: 3,
		},
		{
			name:     `top-level <attribute ref="">`,
			body:     "\n" + `<xs:attribute ref=""/>`,
			wantLine: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.body))
			if err == nil {
				t.Fatalf("Produce succeeded, want a grammar fault for the prohibited ref")
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if !strings.Contains(err.Error(), "carries a ref attribute") {
				t.Fatalf("error = %v, want it to name ref as the prohibited attribute", err)
			}
			if strings.Contains(err.Error(), "no usable name") {
				t.Fatalf("error = %v, want the ref fault rather than the absent-name one it causes", err)
			}
			if at := fmt.Sprintf("%s:%d:", produceURI, tc.wantLine); !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at %s (E3)", err, at)
			}
		})
	}
}

// TestProduceTopLevelProhibitedAttrsRejected pins, END TO END from a schema
// DOCUMENT, the REST of the prohibited family the sibling table above covers for
// ref: xs:topLevelElement restricts form, targetNamespace, minOccurs and
// maxOccurs to use="prohibited" (xmlschema11-1.md:5101-:5104) and
// xs:topLevelAttribute restricts form, use and targetNamespace (:4711-:4713).
// The two lists are not symmetric — use is the <attribute> side's alone and the
// occurrence pair the <element> side's — because neither attribute exists in the
// other kind's grammar at any level.
//
// Every row DECLARES a name, so no row can pass as topLevelName's absent-name
// fault, and each asserts the diagnostic names ITS OWN attribute: a guard that
// checked one attribute and reported another would pass a table that only
// asserted rejection. The fault is plain, never a rule verdict — the grammar
// binds through src-element's and src-attribute's unnumbered "In addition to the
// conditions imposed … by the schema for schema documents" preamble alone, and
// src-attribute clause 6 / src-element clause 4 govern targetNamespace only on
// the LOCAL declaration that may carry one (xmlschema11-1.md:868, :1321), so
// charging any of them here would be fabricated (STYLE E2).
func TestProduceTopLevelProhibitedAttrsRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts
	// on line 2 of the wrapped document.
	cases := []struct {
		name     string
		body     string
		wantAttr string
		wantLine int
	}{
		{
			name:     `top-level <element form=>`,
			body:     "\n" + `<xs:element name="e" type="xs:string" form="qualified"/>`,
			wantAttr: "form",
			wantLine: 2,
		},
		{
			// The value equals the schema's own target namespace, so no row can pass
			// for being a namespace MISMATCH — the top-level form may not carry the
			// attribute at all, whatever it says.
			name:     `top-level <element targetNamespace=>`,
			body:     "\n" + `<xs:element name="e" type="xs:string" targetNamespace="urn:po"/>`,
			wantAttr: "targetNamespace",
			wantLine: 2,
		},
		{
			name:     `top-level <element minOccurs=>`,
			body:     "\n" + `<xs:element name="e" type="xs:string" minOccurs="0"/>`,
			wantAttr: "minOccurs",
			wantLine: 2,
		},
		{
			name:     `top-level <element maxOccurs=>`,
			body:     "\n" + `<xs:element name="e" type="xs:string" maxOccurs="unbounded"/>`,
			wantAttr: "maxOccurs",
			wantLine: 2,
		},
		{
			// Two prohibited attributes at once, written in the REVERSE of the
			// grammar's declaration order: the check order is the grammar's, not the
			// document's, so the reported attribute is stable (STYLE D2).
			name:     `top-level <element maxOccurs= minOccurs=>`,
			body:     "\n" + `<xs:element name="e" type="xs:string" maxOccurs="unbounded" minOccurs="0"/>`,
			wantAttr: "minOccurs",
			wantLine: 2,
		},
		{
			name:     `top-level <attribute form=>`,
			body:     "\n" + `<xs:attribute name="a" type="xs:string" form="qualified"/>`,
			wantAttr: "form",
			wantLine: 2,
		},
		{
			name:     `top-level <attribute use=>`,
			body:     "\n" + `<xs:attribute name="a" type="xs:string" use="required"/>`,
			wantAttr: "use",
			wantLine: 2,
		},
		{
			// The pairing src-attribute clause 2 would otherwise charge: the
			// prohibited attribute is the fault, and it wins because run rejects it
			// before produceAttribute maps anything (the row this took over from
			// TestProduceAttributeUseValueConstraintClauses).
			name:     `top-level <attribute default= use=>`,
			body:     "\n" + `<xs:attribute name="a" type="xs:string" default="dv" use="required"/>`,
			wantAttr: "use",
			wantLine: 2,
		},
		{
			name:     `top-level <attribute targetNamespace=>`,
			body:     "\n" + `<xs:attribute name="a" type="xs:string" targetNamespace="urn:po"/>`,
			wantAttr: "targetNamespace",
			wantLine: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.body))
			if err == nil {
				t.Fatalf("Produce succeeded, want a grammar fault for the prohibited %s", tc.wantAttr)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if want := fmt.Sprintf("carries a %s attribute", tc.wantAttr); !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to name %s as the prohibited attribute", err, tc.wantAttr)
			}
			if strings.Contains(err.Error(), "no usable name") {
				t.Fatalf("error = %v, want the prohibited-attribute fault rather than an absent-name one", err)
			}
			if at := fmt.Sprintf("%s:%d:", produceURI, tc.wantLine); !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at %s (E3)", err, at)
			}
		})
	}
}

// TestProduceProhibitedAttrsLegalElsewhere is the reverse hazard of the table
// above, and the reason the guard carries a list per KIND rather than one merged
// list.
//
// The local rows produce cleanly: form, use, targetNamespace, minOccurs and
// maxOccurs are exactly what xs:localElement and xs:localAttribute admit, so a
// guard that widened to every <element>/<attribute> would reject legal schemas
// wholesale. Each local targetNamespace= repeats the schema's own, which
// src-attribute clause 6.3 and src-element clause 4.3 leave unconstrained.
//
// The asymmetric rows pin what the guard must NOT CLAIM: use= on an <element>
// and minOccurs=/maxOccurs= on an <attribute> appear in no grammar for those
// elements at any level, so xs:topLevelElement and xs:topLevelAttribute do not
// prohibit them and this guard may not say they do. They are asserted on the
// message rather than on acceptance, since a later §A attribute-set check may
// legitimately reject those documents for a different, true reason.
func TestProduceProhibitedAttrsLegalElsewhere(t *testing.T) {
	local := func(inner string) string {
		return `<xs:complexType name="CT"><xs:sequence>` + inner + `</xs:sequence></xs:complexType>`
	}
	localAttr := func(inner string) string {
		return `<xs:complexType name="CT"><xs:sequence/>` + inner + `</xs:complexType>`
	}
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		body string
	}{
		{`local <element form=>`, local(`<xs:element name="e" type="xs:string" form="qualified"/>`)},
		{`local <element targetNamespace=>`, local(`<xs:element name="e" type="xs:string" targetNamespace="urn:po"/>`)},
		{`local <element minOccurs=>`, local(`<xs:element name="e" type="xs:string" minOccurs="0"/>`)},
		{`local <element maxOccurs=>`, local(`<xs:element name="e" type="xs:string" maxOccurs="unbounded"/>`)},
		{`local <attribute form=>`, localAttr(`<xs:attribute name="a" type="xs:string" form="qualified"/>`)},
		{`local <attribute use=>`, localAttr(`<xs:attribute name="a" type="xs:string" use="required"/>`)},
		{`local <attribute targetNamespace=>`, localAttr(`<xs:attribute name="a" type="xs:string" targetNamespace="urn:po"/>`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:po", tc.body)); err != nil {
				t.Fatalf("Produce rejected a local declaration the grammar admits: %v", err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		body string
	}{
		{`top-level <element use=>`, `<xs:element name="e" type="xs:string" use="required"/>`},
		{`top-level <attribute minOccurs=>`, `<xs:attribute name="a" type="xs:string" minOccurs="0"/>`},
		{`top-level <attribute maxOccurs=>`, `<xs:attribute name="a" type="xs:string" maxOccurs="unbounded"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.body))
			if err == nil {
				return
			}
			if strings.Contains(err.Error(), "prohibits on a top-level declaration") {
				t.Fatalf("error = %v, want no top-level prohibition claimed for an attribute that kind's grammar never declares", err)
			}
		})
	}
}

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

// TestProduceQNameOutsideLexicalSpaceRejected pins bindQName's lexical check — a
// QName-valued schema attribute outside the ·lexical space· of xs:QName
// (Datatypes §3.3.18, whose prefix and local part are both the NCName of
// §3.4.7.1), the type Appendix A declares for every such attribute, fails
// cvc-datatype-valid (§4.1.4) at the attribute the author wrote rather than
// travelling on as the zero xsd.QName or as a name nobody could declare.
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
//
// #631 WIDENED the check to the two remaining shapes and their rows are here on
// the same footing, each written so it cannot pass for the wrong reason:
//
//   - an EMPTY PREFIX (:T), whose row DECLARES the target the lexical names, so
//     dropping the check makes the document produce cleanly (that was the silent
//     false accept: :T bound to the in-scope default namespace as a synonym for
//     T);
//   - MORE THAN ONE COLON (xs:a:b), whose row would otherwise be rejected as
//     src-resolve clause 1.1 for a type named "a:b" — so the row fails on the
//     RULE ID, not on the verdict direction.
func TestProduceQNameOutsideLexicalSpaceRejected(t *testing.T) {
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
		{
			// The declared T is what makes this row the false accept and not a
			// missing-target rejection: :T resolves to T without the check (#631).
			name: `type=":T" on a top-level <element>, with T declared`,
			body: "\n" + `<xs:simpleType name="T"><xs:restriction base="xs:string"/></xs:simpleType>` + "\n" +
				`<xs:element name="e" type=":T"/>`,
		},
		{
			// The list-item path, same shape: the head it names is declared.
			name: `substitutionGroup item with an empty prefix`,
			body: "\n" + `<xs:element name="e" type="xs:string"/>` + "\n" +
				`<xs:element name="f" type="xs:string" substitutionGroup=":e"/>`,
		},
		{
			name: `type="xs:a:b" on a top-level <element>`,
			body: "\n" + `<xs:element name="e" type="xs:a:b"/>`,
		},
		{
			// bindQName DIRECTLY, where a second colon otherwise reaches {disallowed
			// names} as a local part and no rejection ever comes (#631).
			name: `notQName item with a colon in its local part`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:any notQName="xs:a:b"/>` + "\n" +
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
