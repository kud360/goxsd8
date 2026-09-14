package parser_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// clause1Doc builds a schema whose one top-level <simpleType name="S"> holds
// lines as its <restriction>'s children, each on a source line of its own:
// <schema> is line 1, the <simpleType> line 2, the <restriction> line 3 and
// lines[i] line 4+i. Every rejection below is asserted to name an exact line, so
// a check that fires at the wrong position fails here rather than passing on the
// strength of rejecting something.
func clause1Doc(base string, lines ...string) string {
	return wrap("urn:x", "\n<xs:simpleType name=\"S\">\n<xs:restriction"+base+">\n"+
		strings.Join(lines, "\n")+"\n</xs:restriction>\n</xs:simpleType>")
}

// TestProduceSrcSimpleTypeClause1Rejected pins src-simple-type clause 1
// (§3.16.3, xmlschema11-1.md:3658-3659) over a <simpleType>'s <restriction>: no
// two of its children share an expanded name in the xs namespace unless that
// name is <enumeration>, <pattern> or <assertion>. Two inline <simpleType>
// children (stC011) PRODUCED clean before this charge — resolveBase reads the
// first and the second was discarded unseen — so a row that merely asserted
// "rejected" would be pinning nothing there.
//
// The verdict is an xsderr.Error carrying src-simple-type, not the plain §5.1
// grammar fault its sibling checks charge: the clause is a named Schema
// Representation Constraint scoped "in addition to the conditions imposed … by
// the schema for schema documents" (STYLE E2). Each row asserts the rule ID, the
// clause named inline (STYLE E4) and the SECOND child's own position (STYLE E3),
// so a check reporting the <restriction>'s position or the first child's would
// not pass.
//
// rejectDuplicateRestrictionChildren's doc records which shapes inside the
// clause's reach are answered ahead of it and why; the rows here are the ones
// this charge itself answers.
func TestProduceSrcSimpleTypeClause1Rejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name  string
		base  string
		lines []string
		want  string // "<local> child", as the message spells the duplicate
	}{
		{
			name: "two inline simpleType children (stC011)",
			lines: []string{
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`<xs:simpleType><xs:restriction base="xs:integer"/></xs:simpleType>`,
			},
			want: "second <simpleType> child",
		},
		{
			name: "two length facets",
			base: ` base="xs:string"`,
			lines: []string{
				`<xs:length value="4"/>`,
				`<xs:length value="5"/>`,
			},
			want: "second <length> child",
		},
		{
			name: "two whiteSpace facets",
			base: ` base="xs:string"`,
			lines: []string{
				`<xs:whiteSpace value="collapse"/>`,
				`<xs:whiteSpace value="collapse"/>`,
			},
			want: "second <whiteSpace> child",
		},
		{
			name: "a repeated facet separated by an excepted one",
			base: ` base="xs:string"`,
			lines: []string{
				`<xs:minLength value="1"/>`,
				`<xs:pattern value="a+"/>`,
				`<xs:minLength value="2"/>`,
			},
			want: "second <minLength> child",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, clause1Doc(tc.base, tc.lines...))
			if err == nil {
				t.Fatal("Produce accepted a <restriction> with two children sharing one expanded name")
			}
			rule, ok := xsderr.RuleOf(err)
			if !ok || rule != "src-simple-type" {
				t.Fatalf("error = %v, charged %q; want src-simple-type", err, rule)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want it to name the %s", err, tc.want)
			}
			if !strings.Contains(err.Error(), "src-simple-type clause 1") {
				t.Fatalf("error = %v, want it to name clause 1 inline (E4)", err)
			}
			// The duplicate is the LAST line of each row's body, so it stands at
			// 4+len(lines)-1.
			at := produceURI + ":" + strconv.Itoa(4+len(tc.lines)-1) + ":1"
			if !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at the second child, %s (E3)", err, at)
			}
		})
	}
}

// TestProduceSrcSimpleTypeClause1ExceptedNames pins the other side of the
// clause: its three excepted names may repeat, and each is what makes
// restrictionFacets' three folds reachable at all. A check that counted every
// duplicate name would reject all three of these documents.
func TestProduceSrcSimpleTypeClause1ExceptedNames(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name  string
		lines []string
	}{
		{
			name: "repeated enumeration",
			lines: []string{
				`<xs:enumeration value="a"/>`,
				`<xs:enumeration value="b"/>`,
			},
		},
		{
			name: "repeated pattern",
			lines: []string{
				`<xs:pattern value="a+"/>`,
				`<xs:pattern value="b+"/>`,
			},
		},
		{
			name: "repeated assertion",
			lines: []string{
				`<xs:assertion test="true()"/>`,
				`<xs:assertion test="true()"/>`,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, clause1Doc(` base="xs:string"`, tc.lines...)); err != nil {
				t.Fatalf("Produce: %v, want the excepted name admitted twice", err)
			}
		})
	}
}
