package parser_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// twoAnnotations is a pair of sibling <annotation> children, each on its own
// line, so a body written as "\n" + open + twoAnnotations + close inside wrap
// puts the SECOND one on line 4 of the document.
const twoAnnotations = "\n" +
	`<xs:annotation><xs:documentation>first</xs:documentation></xs:annotation>` + "\n" +
	`<xs:annotation><xs:documentation>second</xs:documentation></xs:annotation>` + "\n"

// TestProduceRepeatedAnnotationRejected pins that an element carrying a second
// <annotation> child is REJECTED: xs:annotated declares <annotation> with the
// default maxOccurs="1" (xmlschema11-1.md:4436) and "is extended by all types
// which allow annotation other than <schema> itself" (:4429). Most rows carry
// the shape of the MS-Annotations2006-07-15 fixture each names; <override> is
// there because its own particle (:5576) gives it the same cardinality even
// though it bypasses xs:annotated, and <element> because it is the plainest
// xs:annotated derivative there is. The annotB020 row is the only one whose
// conformance case does not move with this guard: that fixture <include>s a
// document the suite does not ship, and the harness declines a rejection made
// alongside an unfollowed directive.
//
// The fault is a plain grammar fault, never a rule verdict: §3.15.3, §3.15.4 and
// §3.15.5 each answer "None as such" (:3499, :3503, :3507), so charging any
// src-*/cos-* rule would be fabricated (STYLE E2). That a NAMED rule also reaches
// one row's shape from another direction is no exception to it — see the
// <restriction> row. Each row asserts the diagnostic is positioned at the SECOND
// <annotation>'s own line rather than at the parent's or the first's (STYLE E3,
// carried in the message text since a plain error holds no xsderr.Loc), and that
// it names the parent — so a guard that reported the parent's position, or the
// enclosing declaration, would not pass.
func TestProduceRepeatedAnnotationRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name   string
		parent string
		open   string
		close  string
	}{
		{
			name:   "<include> (annotB020)",
			parent: "include",
			open:   `<xs:include schemaLocation="other.xsd">`,
			close:  `</xs:include>`,
		},
		{
			name:   "<key> (annotB021)",
			parent: "key",
			open:   `<xs:element name="foo"><xs:key name="k">`,
			close:  `<xs:selector xpath="*"/><xs:field xpath="*"/></xs:key></xs:element>`,
		},
		{
			name:   "<keyref> (annotB022)",
			parent: "keyref",
			open:   `<xs:element name="foo"><xs:key name="k"><xs:selector xpath="*"/><xs:field xpath="*"/></xs:key><xs:keyref name="r" refer="k">`,
			close:  `<xs:selector xpath="*"/><xs:field xpath="*"/></xs:keyref></xs:element>`,
		},
		{
			name:   "<list> (annotB023)",
			parent: "list",
			open:   `<xs:simpleType name="st"><xs:list itemType="xs:integer">`,
			close:  `</xs:list></xs:simpleType>`,
		},
		{
			name:   "<notation> (annotB024)",
			parent: "notation",
			open:   `<xs:notation name="jpeg" public="image/jpeg" system="viewer.exe">`,
			close:  `</xs:notation>`,
		},
		{
			// A <simpleType>'s <restriction>, whose second <annotation> is inside
			// src-simple-type clause 1's reach as well —
			// rejectDuplicateRestrictionChildren's doc records which charge answers
			// this shape and why.
			name:   "<restriction> (annotB026)",
			parent: "restriction",
			open:   `<xs:simpleType name="st"><xs:restriction base="xs:integer">`,
			close:  `</xs:restriction></xs:simpleType>`,
		},
		{
			name:   "<selector> (annotB028)",
			parent: "selector",
			open:   `<xs:element name="foo"><xs:unique name="u"><xs:selector xpath="*">`,
			close:  `</xs:selector><xs:field xpath="*"/></xs:unique></xs:element>`,
		},
		{
			name:   "<sequence> (annotB029)",
			parent: "sequence",
			open:   `<xs:complexType name="ct"><xs:sequence>`,
			close:  `</xs:sequence></xs:complexType>`,
		},
		{
			name:   "<simpleType> (annotB031)",
			parent: "simpleType",
			open:   `<xs:simpleType name="st">`,
			close:  `<xs:restriction base="xs:integer"/></xs:simpleType>`,
		},
		{
			name:   "<union> (annotB032)",
			parent: "union",
			open:   `<xs:simpleType name="st"><xs:union memberTypes="xs:integer">`,
			close:  `</xs:union></xs:simpleType>`,
		},
		{
			name:   "<unique> (annotB033)",
			parent: "unique",
			open:   `<xs:element name="foo"><xs:unique name="u">`,
			close:  `<xs:selector xpath="*"/><xs:field xpath="*"/></xs:unique></xs:element>`,
		},
		{
			name:   "<override> is not exempt",
			parent: "override",
			open:   `<xs:override schemaLocation="other.xsd">`,
			close:  `</xs:override>`,
		},
		{
			name:   "<element>",
			parent: "element",
			open:   `<xs:element name="foo" type="xs:string">`,
			close:  `</xs:element>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", "\n"+tc.open+twoAnnotations+tc.close))
			if err == nil {
				t.Fatalf("Produce succeeded, want <%s> rejected for its second <annotation>", tc.parent)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if want := fmt.Sprintf("<%s>", tc.parent); !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to name %s as the parent", err, want)
			}
			if at := fmt.Sprintf("%s:4:", produceURI); !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at the second <annotation>, %s (E3)", err, at)
			}
		})
	}
}

// TestProduceNestedAnnotationRejected pins the SECOND, distinct fault: an
// <annotation> whose own children include an <annotation> is rejected, because
// <annotation>'s content model is (appinfo | documentation)* (xmlschema11-1.md
// :5747-5763, prose "Content: (appinfo | documentation)*" at :3480) with no
// <annotation> branch — inadmissible at any cardinality, so a single nested
// child (annotB001) is rejected no less than two (annotB005). This is NOT the
// xs:annotated maxOccurs="1" cardinality fault TestProduceRepeatedAnnotation*
// exercises, and does NOT inherit its {schema, redefine} exemption: the outer
// <annotation> here is itself a child of <schema>, which the cardinality check
// exempts, yet the nested <annotation> is still rejected.
//
// Like the cardinality fault it is a plain grammar fault, never a rule verdict
// (§3.15.3/§3.15.4/§3.15.5 all answer "None as such", :3499/:3503/:3507; STYLE
// E2), and each row asserts the diagnostic is positioned at the offending nested
// <annotation>'s own line (STYLE D2/E3) — the FIRST one when there are two, the
// document-order head of the child list.
func TestProduceNestedAnnotationRejected(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			// annotB001-shaped: one nested <annotation> child, alongside the
			// <appinfo>/<documentation> the content model does admit.
			name: "one nested (annotB001)",
			body: "\n" +
				`<xs:annotation>` + "\n" +
				`<xs:annotation><xs:documentation>d</xs:documentation></xs:annotation>` + "\n" +
				`<xs:appinfo>a</xs:appinfo>` + "\n" +
				`</xs:annotation>` + "\n",
		},
		{
			// annotB005-shaped: two nested <annotation> children; the diagnostic
			// names the FIRST (line 3), not the second.
			name: "two nested (annotB005)",
			body: "\n" +
				`<xs:annotation>` + "\n" +
				`<xs:annotation><xs:appinfo>a</xs:appinfo></xs:annotation>` + "\n" +
				`<xs:annotation><xs:appinfo>a</xs:appinfo></xs:annotation>` + "\n" +
				`</xs:annotation>` + "\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.body))
			if err == nil {
				t.Fatalf("Produce succeeded, want the nested <annotation> rejected")
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if !strings.Contains(err.Error(), "content model") {
				t.Fatalf("error = %v, want it to cite <annotation>'s content model, not the xs:annotated cardinality", err)
			}
			if at := fmt.Sprintf("%s:3:", produceURI); !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at the (first) nested <annotation>, %s (E3)", err, at)
			}
		})
	}
}

// TestProduceRepeatedAnnotationAccepted pins the other side of the guard: the
// two elements whose own content models admit <annotation> unboundedly —
// <schema> (xmlschema11-1.md:4558, :4563) and <redefine> (:5556-5559) — keep
// producing with two of them, as do <appinfo>/<documentation> holding elements
// that merely happen to be named xs:annotation, which their <xs:any
// processContents="lax"> content (:5727, :5740) never counts. A guard that
// rejected uniformly, or that descended into annotation content, would fail
// here.
func TestProduceRepeatedAnnotationAccepted(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			name: "<schema> (annotB025)",
			body: twoAnnotations,
		},
		{
			name: "<redefine> (annotB027)",
			body: `<xs:redefine schemaLocation="other.xsd">` + twoAnnotations + `</xs:redefine>`,
		},
		{
			name: "inside <appinfo>",
			body: `<xs:element name="foo" type="xs:string"><xs:annotation><xs:appinfo>` +
				`<xs:annotation/><xs:annotation/>` +
				`</xs:appinfo></xs:annotation></xs:element>`,
		},
		{
			name: "inside <documentation>",
			body: `<xs:element name="foo" type="xs:string"><xs:annotation><xs:documentation>` +
				`<xs:annotation/><xs:annotation/>` +
				`</xs:documentation></xs:annotation></xs:element>`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:po", tc.body)); err != nil {
				t.Fatalf("Produce: %v, want the repeated <annotation> accepted here", err)
			}
		})
	}
}

// lateAnnotationOwners are the seven owners rejectLateAnnotation holds to
// "annotation?" as their first particle — the <all>, <choice> and <sequence>
// compositors, a named <group>, a top-level <attributeGroup>, a <list> and a
// <union> (#1941) — each with its open and close tags and the one non-annotation
// child written beside the <annotation>, named by childName.
var lateAnnotationOwners = []struct {
	owner     string
	open      string
	child     string
	childName string
	close     string
}{
	{"all", `<xs:complexType name="ct"><xs:all>`, `<xs:element name="a"/>`, "element", `</xs:all></xs:complexType>`},
	{"choice", `<xs:complexType name="ct"><xs:choice>`, `<xs:element name="a"/>`, "element", `</xs:choice></xs:complexType>`},
	{"sequence", `<xs:complexType name="ct"><xs:sequence>`, `<xs:element name="a"/>`, "element", `</xs:sequence></xs:complexType>`},
	{"group", `<xs:group name="g">`, `<xs:sequence/>`, "sequence", `</xs:group>`},
	{"attributeGroup", `<xs:attributeGroup name="ag">`, `<xs:attribute name="a"/>`, "attribute", `</xs:attributeGroup>`},
	{"list", `<xs:simpleType name="st"><xs:list>`, `<xs:simpleType><xs:restriction base="xs:integer"/></xs:simpleType>`, "simpleType", `</xs:list></xs:simpleType>`},
	{"union", `<xs:simpleType name="st"><xs:union>`, `<xs:simpleType><xs:restriction base="xs:integer"/></xs:simpleType>`, "simpleType", `</xs:union></xs:simpleType>`},
}

// TestProduceLateAnnotationRejected pins that an <annotation> written AFTER a
// non-annotation child of each owner in lateAnnotationOwners is rejected: every
// form of each opens its content model with "annotation?" and admits it nowhere
// else (xmlschema11-1.md:5187, :5229, :5259, :5502; xmlschema11-2.md:3957,
// :3977). The child before the <annotation> is never another <annotation>:
// rejectRepeatedAnnotations already rejects that shape, so such a row would pass
// without rejectLateAnnotation.
//
// The fault is §5.1's first bullet and carries no rule ID (STYLE E2), so each row
// wants a plain Go error. Each pins the message's opening subject and position as
// a prefix — the <annotation> on line 4 — and the positions of the child it
// follows (line 3) and of the owner (line 2), so a message naming the wrong
// element, or swapping two of them, fails.
func TestProduceLateAnnotationRejected(t *testing.T) {
	for _, tc := range lateAnnotationOwners {
		t.Run(tc.owner, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", "\n"+tc.open+"\n"+tc.child+"\n<xs:annotation/>\n"+tc.close))
			if err == nil {
				t.Fatalf("Produce succeeded, want the <annotation> after the <%s>'s first child rejected", tc.owner)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			msg := err.Error()
			if want := fmt.Sprintf("parser: <annotation> at %s:4:", produceURI); !strings.HasPrefix(msg, want) {
				t.Fatalf("error = %v, want it to open %q (E3)", err, want)
			}
			if want := fmt.Sprintf("follows the <%s> at %s:3:", tc.childName, produceURI); !strings.Contains(msg, want) {
				t.Fatalf("error = %v, want it to contain %q", err, want)
			}
			if want := fmt.Sprintf("children of the <%s> at %s:2:", tc.owner, produceURI); !strings.Contains(msg, want) {
				t.Fatalf("error = %v, want it to contain %q", err, want)
			}
		})
	}
}

// TestProduceLeadingAnnotationAccepted pins the other side of
// rejectLateAnnotation: each owner in lateAnnotationOwners keeps producing with
// its <annotation> written FIRST, ahead of the same child
// TestProduceLateAnnotationRejected writes before it.
func TestProduceLeadingAnnotationAccepted(t *testing.T) {
	for _, tc := range lateAnnotationOwners {
		t.Run(tc.owner, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:po", tc.open+"<xs:annotation/>"+tc.child+tc.close)); err != nil {
				t.Fatalf("Produce: %v, want the leading <annotation> of the <%s> accepted", err, tc.owner)
			}
		})
	}
}
