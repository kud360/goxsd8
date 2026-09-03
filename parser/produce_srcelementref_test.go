package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// refElementDoc builds a schema holding the <element name="E"> every row below
// denotes and a <complexType name="D"> whose <sequence> wraps lines, each on a
// source line of its own: <schema> is line 1, the <element name="E"> line 2, the
// <complexType> line 3, the <sequence> line 4, and lines[i] line 5+i. The
// rejections are asserted to name an exact line, so a check that fires at the
// wrong position fails here rather than passing on the strength of rejecting
// something.
func refElementDoc(lines ...string) string {
	return wrap("urn:x", "\n<xs:element name=\"E\" type=\"xs:string\"/>\n<xs:complexType name=\"D\">\n<xs:sequence>\n"+
		strings.Join(lines, "\n")+"\n</xs:sequence>\n</xs:complexType>")
}

// TestProduceRefElementChildRejected pins the two REJECTIONS a local <element
// ref="..."> with children can draw, and that they stay two: src-element clause
// 2.2's child half (§3.3.3, xmlschema11-1.md:1321 — "If ref is present, then …
// no children in the Schema namespace (xs) other than <annotation>"), charged by
// rejectRefElementChildren, and the schema-for-schema-documents child-order
// fault checkS4SChildOrder charges over the same element against s4sElement
// (#1076).
//
// The two are independent, and neither row would pass on the other's check:
//
//   - the <simpleType> row is a document s4sElement ACCEPTS outright — the model
//     Appendix A gives every form of <element> has a "(simpleType | complexType)?"
//     position and the ref= form does not narrow it — so before #1099 it produced
//     cleanly, and it fails now only because clause 2.2 is charged;
//   - the <annotation>-after-<simpleType> row violates BOTH, and asserts the ORDER
//     the two run in: the grammar walk answers first, so the fault it was already
//     charged before #1099 is the fault it keeps.
//
// The fault CLASS is asserted per row, not just the rejection: clause 2.2 is a
// Schema Representation Constraint and carries the src-element rule ID, while
// the order fault is the uncataloged §5.1 class (STYLE E2) and must carry none.
// A single check answering both rows would fail one of these assertions.
//
// Only the child half of clause 2.2 is in reach here; its attribute half —
// "no unqualified attributes … other than minOccurs, maxOccurs, and id" — is
// TestProduceRefElementSubstitutionGroupRejected's and
// TestProduceRefElementDeclarationAttrRejected's.
func TestProduceRefElementChildRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name     string
		lines    []string
		wantRule xsderr.Rule // empty for the plain §5.1 grammar fault, which carries none
		wantLine int
		wantMsg  string
	}{
		{
			name: "simpleType child of a ref element",
			lines: []string{
				`<xs:element ref="tns:E">`,
				`<xs:simpleType>`,
				`<xs:restriction base="xs:string"/>`,
				`</xs:simpleType>`,
				`</xs:element>`,
			},
			wantRule: "src-element",
			wantLine: 6,
			wantMsg:  "src-element clause 2.2 admits no child in the Schema namespace other than <annotation>",
		},
		{
			name: "complexType child of a ref element",
			lines: []string{
				`<xs:element ref="tns:E">`,
				`<xs:complexType/>`,
				`</xs:element>`,
			},
			wantRule: "src-element",
			wantLine: 6,
			wantMsg:  "carries a <complexType> child",
		},
		{
			name: "unique child of a ref element",
			lines: []string{
				`<xs:element ref="tns:E">`,
				`<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>`,
				`</xs:element>`,
			},
			wantRule: "src-element",
			wantLine: 6,
			wantMsg:  "carries a <unique> child",
		},
		{
			name: "annotation after another child of a ref element",
			lines: []string{
				`<xs:element ref="tns:E">`,
				`<xs:simpleType>`,
				`<xs:restriction base="xs:string"/>`,
				`</xs:simpleType>`,
				`<xs:annotation/>`,
				`</xs:element>`,
			},
			wantLine: 9,
			wantMsg:  "out of the child order",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, refElementDoc(tc.lines...))
			if err == nil {
				t.Fatalf("Produce accepted the <element ref> at %s:5:1, want the %q fault", produceURI, tc.wantMsg)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error = %v, want it to state %q", err, tc.wantMsg)
			}
			if tc.wantRule == "" {
				if rule, ok := xsderr.RuleOf(err); ok {
					t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
				}
				at := fmt.Sprintf("%s:%d:1", produceURI, tc.wantLine)
				if !strings.Contains(err.Error(), at) {
					t.Errorf("error = %v, want it to name the offending child at %s", err, at)
				}
				return
			}
			assertRule(t, err, tc.wantRule)
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:%d:col (E3)", err, produceURI, tc.wantLine)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the offending child at %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}

// TestProduceRefElementSubstitutionGroupRejected pins clause 2.2's ATTRIBUTE
// half for substitutionGroup on a local <element ref="..."> (§3.3.3,
// xmlschema11-1.md:1321 — "If ref is present, then no unqualified attributes are
// present other than minOccurs, maxOccurs, and id"): before this charge the
// attribute was read by nothing on the ref= path and the document was accepted
// outright.
//
// The rule ID is the assertion that matters. The INLINE local form is pinned to
// e-props-correct clause 3 by TestProduceLocalElementSubstitutionGroupRejected,
// and the two must not converge: the ref= form maps to an
// ElementDeclarationRef, which has no {substitution group affiliations}
// property for that component constraint to be violated by, so src-element is
// the only footing that is not fabricated (STYLE E2).
//
// The second row asserts the order of the clause's two halves on a document that
// violates both: the attribute half answers, at the <element> itself, not at the
// child.
func TestProduceRefElementSubstitutionGroupRejected(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "substitutionGroup on a ref element",
			lines: []string{
				`<xs:element ref="tns:E" substitutionGroup="tns:E"/>`,
			},
		},
		{
			name: "substitutionGroup on a ref element that also has a child",
			lines: []string{
				`<xs:element ref="tns:E" substitutionGroup="tns:E">`,
				`<xs:simpleType>`,
				`<xs:restriction base="xs:string"/>`,
				`</xs:simpleType>`,
				`</xs:element>`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, refElementDoc(tc.lines...))
			if err == nil {
				t.Fatalf("Produce accepted an <element ref> carrying substitutionGroup, want the src-element clause 2.2 fault")
			}
			const want = `the <element ref="..."> carries a substitutionGroup attribute, but src-element clause 2.2 admits no unqualified attribute other than minOccurs, maxOccurs and id`
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error = %v, want it to state %q", err, want)
			}
			// The rendered error states its position ONCE, from the prefix xsderr
			// adds (STYLE E3). Unlike rejectRefElementChildren, whose two locations
			// differ — the fault at the child, the message naming the parent — this
			// check has only el.Loc(), so restating it in the message text would be
			// a copy of the prefix.
			if at := produceURI + ":5:"; strings.Count(err.Error(), at) != 1 {
				t.Errorf("error = %v, names %s %d times, want exactly once (E3)",
					err, at, strings.Count(err.Error(), at))
			}
			assertRule(t, err, "src-element")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:5:col (E3)", err, produceURI)
			}
			if loc.URI != produceURI || loc.Line != 5 || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the <element ref> itself at %s:5 with a column",
					loc.URI, loc.Line, loc.Col, produceURI)
			}
		})
	}
}

// TestProduceRefElementSubstitutionGroupAccepted is the other side of the
// attribute half: clause 2.2 reaches UNQUALIFIED attributes only — Element.Attr
// answers for those alone, and xs:localElement admits foreign ones outright
// (<anyAttribute namespace="##other"> at xmlschema11-1.md:5127) — and it exempts
// minOccurs, maxOccurs and id by name. A check written as "a ref= element has no
// substitutionGroup-shaped attribute", or one that swept the exempt set in with
// it, would reject these.
func TestProduceRefElementSubstitutionGroupAccepted(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "foreign-namespace substitutionGroup on a ref element",
			lines: []string{
				`<xs:element ref="tns:E" xmlns:o="urn:other" o:substitutionGroup="tns:E"/>`,
			},
		},
		{
			name: "the exempt attributes on a ref element",
			lines: []string{
				`<xs:element ref="tns:E" minOccurs="0" maxOccurs="2" id="r1"/>`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, refElementDoc(tc.lines...)); err != nil {
				t.Fatalf("Produce rejected an <element ref> clause 2.2 admits: %v", err)
			}
		})
	}
}

// TestProduceRefElementDeclarationAttrRejected pins clause 2.2's attribute half
// for the eight attributes xs:localElement leaves syntactically LEGAL under
// ref= (§3.3.3, xmlschema11-1.md:1321). Unlike substitutionGroup, none of these
// carries a use="prohibited" narrowing on xs:localElement
// (xmlschema11-1.md:5124-5126 prohibits three, and none of these is among them),
// so clause 2.2's prose is the only thing rejecting them and every row here
// produced cleanly before this charge.
//
// Two rows carry an attribute a SECOND clause of src-element governs
// independently — name (clause 2.1, ref and name both present) and
// targetNamespace (clause 4.1, ed-with-ns requiring name) — and both are
// asserted to report clause 2.2 all the same. The rule ID is one per attribute:
// a later session charging 2.1 or 4 must decide what these rows say, not
// discover the choice was never made.
//
// The last row is the order assertion, and it is the one a rewrite is most
// likely to break: the attributes are checked in xs:element's DECLARATION
// order, so a ref= element carrying type= before name= in the document is still
// rejected for name (STYLE D2 — the reported attribute is output, and output
// does not follow a document's or a map's order).
func TestProduceRefElementDeclarationAttrRejected(t *testing.T) {
	for _, tc := range []struct {
		name     string
		line     string
		wantAttr string
	}{
		{name: "name", line: `<xs:element ref="tns:E" name="F"/>`, wantAttr: "name"},
		{name: "type", line: `<xs:element ref="tns:E" type="xs:string"/>`, wantAttr: "type"},
		{name: "default", line: `<xs:element ref="tns:E" default="d"/>`, wantAttr: "default"},
		{name: "fixed", line: `<xs:element ref="tns:E" fixed="f"/>`, wantAttr: "fixed"},
		{name: "nillable", line: `<xs:element ref="tns:E" nillable="true"/>`, wantAttr: "nillable"},
		{name: "block", line: `<xs:element ref="tns:E" block="#all"/>`, wantAttr: "block"},
		{name: "form", line: `<xs:element ref="tns:E" form="qualified"/>`, wantAttr: "form"},
		{name: "targetNamespace", line: `<xs:element ref="tns:E" targetNamespace="urn:x"/>`, wantAttr: "targetNamespace"},
		{
			name:     "declaration order wins over document order",
			line:     `<xs:element ref="tns:E" type="xs:string" name="F"/>`,
			wantAttr: "name",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, refElementDoc(tc.line))
			if err == nil {
				t.Fatalf("Produce accepted an <element ref> carrying %s, want the src-element clause 2.2 fault", tc.wantAttr)
			}
			want := fmt.Sprintf(`the <element ref="..."> carries a %s attribute, but src-element clause 2.2 admits no unqualified attribute other than minOccurs, maxOccurs and id`, tc.wantAttr)
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error = %v, want it to state %q", err, want)
			}
			assertRule(t, err, "src-element")
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want %s:5:col (E3)", err, produceURI)
			}
			if loc.URI != produceURI || loc.Line != 5 || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the <element ref> itself at %s:5 with a column",
					loc.URI, loc.Line, loc.Col, produceURI)
			}
		})
	}
}

// TestProduceRefElementDeclarationAttrAccepted is the other side: clause 2.2
// reaches UNQUALIFIED attributes only, so the same eight names in a foreign
// namespace are admitted (xs:localElement takes <anyAttribute
// namespace="##other"> at xmlschema11-1.md:5127), and the eight are rejected
// under ref= ALONE — the inline form declares every one of them legitimately, so
// a check hoisted out of the ref= branch would reject the whole language.
func TestProduceRefElementDeclarationAttrAccepted(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "foreign-namespace declaration attributes on a ref element",
			lines: []string{
				`<xs:element ref="tns:E" xmlns:o="urn:other" o:name="F" o:type="xs:string" o:form="qualified"/>`,
			},
		},
		{
			name: "the same attributes on the inline local form",
			lines: []string{
				`<xs:element name="F" type="xs:string" default="d" nillable="true" block="#all" form="qualified"/>`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, refElementDoc(tc.lines...)); err != nil {
				t.Fatalf("Produce rejected an <element> clause 2.2 admits: %v", err)
			}
		})
	}
}

// TestProduceRefElementChildAccepted is the other side of clause 2.2's child
// half: the clause reaches the Schema namespace alone, and it excepts
// <annotation> by name. A check written as "a ref= element has no children"
// would reject both of these.
func TestProduceRefElementChildAccepted(t *testing.T) {
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "annotation child of a ref element",
			lines: []string{
				`<xs:element ref="tns:E"><xs:annotation/></xs:element>`,
			},
		},
		{
			name: "foreign-namespace child of a ref element",
			lines: []string{
				`<xs:element ref="tns:E" xmlns:o="urn:other"><xs:annotation/><o:hint/></xs:element>`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, refElementDoc(tc.lines...)); err != nil {
				t.Fatalf("Produce rejected an <element ref> clause 2.2 admits: %v", err)
			}
		})
	}
}
