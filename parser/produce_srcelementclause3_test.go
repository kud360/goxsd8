package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceSrcElementClause3BehindS4SWalk pins WHICH fault an <element> that
// violates src-element clause 3 (§3.3.3, xmlschema11-1.md:1320 — a type
// attribute together with a <simpleType> or <complexType> child) AND the schema
// for schema documents' child order is answered by. The walk answers it
// (#1246): clause 3 is charged behind checkS4SChildOrder on both element paths,
// the order every other src-* charge in this parser already took.
//
// The three rows are one table because no one of them constrains the order by
// itself. A clause-3-only document rejects the same way whichever side of the
// walk the charge sits on, and so does a walk-only document; only the doubly
// invalid rows below distinguish them, and only against the two singly invalid
// rows do they pin that the move cost neither charge its own document.
func TestProduceSrcElementClause3BehindS4SWalk(t *testing.T) {
	const inlineSimple = `<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`
	const key = `<xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:key>`

	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name     string
		lines    []string
		topLevel bool // lines are the <schema>'s own children, not a <complexType>'s
		// wantRule is the rule the rejection carries, or "" for the walk's own §5.1
		// fault, which carries none (STYLE E2).
		wantRule xsderr.Rule
		// wantPrefix is the whole "parser: <subject> at <loc>" opening of the walk's
		// message, asserted as a PREFIX: a verdict naming a different child, or the
		// right child at the wrong position, fails here rather than passing on a
		// substring the two messages share (#1048). It is empty on the clause-3 rows,
		// whose position is asserted through xsderr.LocOf instead.
		wantPrefix string
		wantMsg    string
		// wantLine is the 1-based line the offending <element> opens on, asserted on
		// the clause-3 rows so a charge firing over the wrong declaration fails here.
		wantLine int
	}{
		{
			name:     "global element with type= and an inline simpleType keeps clause 3",
			topLevel: true,
			lines: []string{
				`<xs:element name="e" type="xs:string">`,
				inlineSimple,
				`</xs:element>`,
			},
			wantRule: "src-element",
			wantMsg:  `element has both a type attribute and an inline <simpleType>/<complexType> child, but src-element clause 3 forbids both`,
			wantLine: 2,
		},
		{
			name: "local element with type= and an inline complexType keeps clause 3",
			lines: []string{
				`<xs:sequence>`,
				`<xs:element name="e" type="xs:string">`,
				`<xs:complexType><xs:sequence/></xs:complexType>`,
				`</xs:element>`,
				`</xs:sequence>`,
			},
			wantRule: "src-element",
			wantMsg:  `element has both a type attribute and an inline <simpleType>/<complexType> child, but src-element clause 3 forbids both`,
			wantLine: 4,
		},
		{
			// Clause 3's antecedent is unmet — no type= — so the walk is the only fault
			// live on this document, and it answers with no rule at all.
			name:     "global element whose children are out of order and carry no type= is the walk's",
			topLevel: true,
			lines: []string{
				`<xs:element name="e">`,
				key,
				inlineSimple,
				`</xs:element>`,
			},
			wantPrefix: "parser: <simpleType> at " + produceURI + ":4:1 is out of the child order",
		},
		{
			name: "local element whose children are out of order and carry no type= is the walk's",
			lines: []string{
				`<xs:sequence>`,
				`<xs:element name="e">`,
				key,
				inlineSimple,
				`</xs:element>`,
				`</xs:sequence>`,
			},
			wantPrefix: "parser: <simpleType> at " + produceURI + ":6:1 is out of the child order",
		},
		{
			// The row the ordering rests on: BOTH faults are live — the <simpleType>
			// follows an identity constraint s4sElement positions after it, and type= is
			// written beside it — and the walk's verdict is the one reported.
			name:     "global element violating clause 3 and the child order is the walk's",
			topLevel: true,
			lines: []string{
				`<xs:element name="e" type="xs:string">`,
				key,
				inlineSimple,
				`</xs:element>`,
			},
			wantPrefix: "parser: <simpleType> at " + produceURI + ":4:1 is out of the child order",
		},
		{
			name: "local element violating clause 3 and the child order is the walk's",
			lines: []string{
				`<xs:sequence>`,
				`<xs:element name="e" type="xs:string">`,
				key,
				inlineSimple,
				`</xs:element>`,
				`</xs:sequence>`,
			},
			wantPrefix: "parser: <simpleType> at " + produceURI + ":6:1 is out of the child order",
		},
		{
			// Neither fault: the same children IN s4sElement's order, and no type= beside
			// them. A charge keyed off the inline child alone, or a walk that read the
			// model as a fixed linear order, would take this row down with it.
			name:     "global element with an inline simpleType before its identity constraint is accepted",
			topLevel: true,
			lines: []string{
				`<xs:element name="e">`,
				inlineSimple,
				key,
				`</xs:element>`,
			},
		},
		{
			name: "local element with an inline simpleType before its identity constraint is accepted",
			lines: []string{
				`<xs:sequence>`,
				`<xs:element name="e">`,
				inlineSimple,
				key,
				`</xs:element>`,
				`</xs:sequence>`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := s4sDoc(tc.lines...)
			if tc.topLevel {
				doc = s4sTopLevelDoc(tc.lines...)
			}
			_, err := produce(t, doc)
			if tc.wantRule == "" && tc.wantPrefix == "" {
				if err != nil {
					t.Fatalf("Produce rejected a document both checks admit: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Produce accepted the document, want a rejection")
			}
			if tc.wantPrefix != "" {
				if rule, ok := xsderr.RuleOf(err); ok {
					t.Fatalf("error = %v, charged %s; want the walk's plain grammar fault, carrying no rule ID", err, rule)
				}
				if !strings.HasPrefix(err.Error(), tc.wantPrefix) {
					t.Fatalf("error = %v, want it to open %q", err, tc.wantPrefix)
				}
				return
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("error = %v, want it to state %q", err, tc.wantMsg)
			}
			assertRule(t, err, tc.wantRule)
			loc, ok := xsderr.LocOf(err)
			if !ok {
				t.Fatalf("error %v carries no position, want the <element>'s (E3)", err)
			}
			if loc.URI != produceURI || loc.Line != tc.wantLine || loc.Col == 0 {
				t.Fatalf("position = %s:%d:%d, want the offending <element> at %s:%d with a column",
					loc.URI, loc.Line, loc.Col, produceURI, tc.wantLine)
			}
		})
	}
}
