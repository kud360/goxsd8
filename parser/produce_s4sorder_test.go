package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// s4sDoc builds a schema whose one <complexType name="D"> holds lines as its
// subtree, each line on a source line of its own: <schema> is line 1, the
// <complexType> line 2, and lines[i] line 3+i. Every rejection below is asserted
// to name an exact line, so a check that fires at the wrong position fails here
// rather than passing on the strength of rejecting something.
func s4sDoc(lines ...string) string {
	return wrap("urn:x", "\n<xs:complexType name=\"D\">\n"+strings.Join(lines, "\n")+"\n</xs:complexType>")
}

// s4sTopLevelDoc is s4sDoc without the <complexType> wrapper: <schema> is line 1
// and lines[i] line 2+i. The three models #1076 adds order declarations no
// complex type need enclose, and their top-level form is where each is written
// without one.
func s4sTopLevelDoc(lines ...string) string {
	return wrap("urn:x", "\n"+strings.Join(lines, "\n"))
}

// TestProduceS4SChildOrderRejected pins the schema-for-schema-documents child
// ORDER and maxOccurs of every element position a complex type is written
// through: xs:complexTypeModel on both its wrapped and its implicit-content
// disjunct (xmlschema11-1.md:1649), the xs:simpleContent (:1687) and
// xs:complexContent (:1713) wrappers, and all four derivation alternants —
// xs:simpleRestrictionType (:1692), xs:simpleExtensionType (:1697),
// xs:complexRestrictionType (:1718) and xs:extensionType (:1723) — and, since
// #1076, the three declarations that carry a content model of their own: xs:element
// (:1120), xs:attribute (:828) and xs:simpleType (xmlschema11-2.md:2743), each
// ordered by ONE model whichever form it is written in.
//
// Before this check the producer read every one of these subtrees by name and
// never by position, so all of these documents assembled clean (#956). The four
// suite cases that exposed it — ctC011, ctD034, ctD042, ctD043 — are all
// <simpleContent> <restriction> shapes; each is repeated here at the sibling
// alternants that carry the same positions and were equally unrejected.
//
// The fault carries NO rule ID: §5.1's first bullet (:4296) is what binds, and
// src-ct's own preamble (§3.4.3, :1945) scopes its five clauses as additional to
// the grammar rather than a restatement of it, so charging src-ct would be a
// fabricated verdict (STYLE E2). Each row asserts the plain error, the offending
// child's position, and the owning element's.
func TestProduceS4SChildOrderRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name      string
		lines     []string
		topLevel  bool   // lines are the <schema>'s own children, not a <complexType>'s
		wantChild string // "<local> at <uri>:<line>:1"
		wantOwner string
		wantKind  string // the phrase distinguishing an order fault from a maxOccurs one
	}{
		{
			// ctC011 itself: <annotation> written after the alternant, where
			// xs:simpleContent puts "annotation?" first.
			name: "annotation after the alternant of a simpleContent",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:restriction base="xs:string"/>`,
				`<xs:annotation/>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<annotation> at " + produceURI + ":5:1",
			wantOwner: "<simpleContent> at " + produceURI + ":3:1",
			wantKind:  "out of the child order",
		},
		{
			name: "annotation after the alternant of a complexContent",
			lines: []string{
				`<xs:complexContent>`,
				`<xs:extension base="xs:anyType"/>`,
				`<xs:annotation/>`,
				`</xs:complexContent>`,
			},
			wantChild: "<annotation> at " + produceURI + ":5:1",
			wantOwner: "<complexContent> at " + produceURI + ":3:1",
			wantKind:  "out of the child order",
		},
		{
			// One level out, on xs:complexTypeModel's own "annotation?".
			name: "annotation after the complexContent of a complexType",
			lines: []string{
				`<xs:complexContent><xs:extension base="xs:anyType"/></xs:complexContent>`,
				`<xs:annotation/>`,
			},
			wantChild: "<annotation> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			name: "annotation after the model group of an implicit-content complexType",
			lines: []string{
				`<xs:sequence/>`,
				`<xs:annotation/>`,
			},
			wantChild: "<annotation> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			// ctD042's shape: the facet position is sequenced strictly before the
			// attribute tail, so a facet after an <attribute> is out of order.
			name: "facet after an attribute under a simpleContent restriction",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:restriction base="xs:string">`,
				`<xs:attribute name="a"/>`,
				`<xs:length value="5"/>`,
				`</xs:restriction>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<length> at " + produceURI + ":6:1",
			wantOwner: "<restriction> at " + produceURI + ":4:1",
			wantKind:  "out of the child order",
		},
		{
			// ctD034's shape: "anyAttribute?" admits exactly one, whatever the two
			// namespace constraints are.
			name: "two anyAttribute under a simpleContent restriction",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:restriction base="xs:string">`,
				`<xs:anyAttribute namespace="##local"/>`,
				`<xs:anyAttribute namespace="##other"/>`,
				`</xs:restriction>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<anyAttribute> at " + produceURI + ":6:1",
			wantOwner: "<restriction> at " + produceURI + ":4:1",
			wantKind:  "repeats a position",
		},
		{
			// ctD043's shape: "anyAttribute?" follows the whole
			// "(attribute | attributeGroup)*" block, and nothing re-enters it.
			name: "attribute after anyAttribute under a simpleContent restriction",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:restriction base="xs:string">`,
				`<xs:anyAttribute namespace="##local"/>`,
				`<xs:attribute name="a"/>`,
				`</xs:restriction>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<attribute> at " + produceURI + ":6:1",
			wantOwner: "<restriction> at " + produceURI + ":4:1",
			wantKind:  "out of the child order",
		},
		{
			name: "simpleType after a facet under a simpleContent restriction",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:restriction base="xs:string">`,
				`<xs:length value="5"/>`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`</xs:restriction>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<simpleType> at " + produceURI + ":6:1",
			wantOwner: "<restriction> at " + produceURI + ":4:1",
			wantKind:  "out of the child order",
		},
		{
			name: "attributeGroup after anyAttribute under a simpleContent extension",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:extension base="xs:string">`,
				`<xs:anyAttribute namespace="##other"/>`,
				`<xs:attributeGroup ref="tns:AG"/>`,
				`</xs:extension>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<attributeGroup> at " + produceURI + ":6:1",
			wantOwner: "<extension> at " + produceURI + ":4:1",
			wantKind:  "out of the child order",
		},
		{
			name: "two anyAttribute under a complexContent restriction",
			lines: []string{
				`<xs:complexContent>`,
				`<xs:restriction base="xs:anyType">`,
				`<xs:anyAttribute namespace="##local"/>`,
				`<xs:anyAttribute namespace="##other"/>`,
				`</xs:restriction>`,
				`</xs:complexContent>`,
			},
			wantChild: "<anyAttribute> at " + produceURI + ":6:1",
			wantOwner: "<restriction> at " + produceURI + ":4:1",
			wantKind:  "repeats a position",
		},
		{
			name: "model group after an attribute under a complexContent extension",
			lines: []string{
				`<xs:complexContent>`,
				`<xs:extension base="xs:anyType">`,
				`<xs:attribute name="a"/>`,
				`<xs:sequence/>`,
				`</xs:extension>`,
				`</xs:complexContent>`,
			},
			wantChild: "<sequence> at " + produceURI + ":6:1",
			wantOwner: "<extension> at " + produceURI + ":4:1",
			wantKind:  "out of the child order",
		},
		{
			name: "openContent after the model group of a complexContent restriction",
			lines: []string{
				`<xs:complexContent>`,
				`<xs:restriction base="xs:anyType">`,
				`<xs:sequence/>`,
				`<xs:openContent><xs:any namespace="##other" processContents="lax"/></xs:openContent>`,
				`</xs:restriction>`,
				`</xs:complexContent>`,
			},
			wantChild: "<openContent> at " + produceURI + ":6:1",
			wantOwner: "<restriction> at " + produceURI + ":4:1",
			wantKind:  "out of the child order",
		},
		{
			name: "attribute after anyAttribute on an implicit-content complexType",
			lines: []string{
				`<xs:anyAttribute namespace="##local"/>`,
				`<xs:attribute name="a"/>`,
			},
			wantChild: "<attribute> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			name: "two anyAttribute on an implicit-content complexType",
			lines: []string{
				`<xs:anyAttribute namespace="##local"/>`,
				`<xs:anyAttribute namespace="##other"/>`,
			},
			wantChild: "<anyAttribute> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
			wantKind:  "repeats a position",
		},
		{
			name: "model group after an attribute on an implicit-content complexType",
			lines: []string{
				`<xs:attribute name="a"/>`,
				`<xs:choice/>`,
			},
			wantChild: "<choice> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			name: "two model groups on an implicit-content complexType",
			lines: []string{
				`<xs:sequence/>`,
				`<xs:choice/>`,
			},
			wantChild: "<choice> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
			wantKind:  "repeats a position",
		},
		{
			// "assert*" closes every one of these models, so nothing may follow it.
			name: "attribute after an assert on an implicit-content complexType",
			lines: []string{
				`<xs:assert test="true()"/>`,
				`<xs:attribute name="a"/>`,
			},
			wantChild: "<attribute> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			// xs:element's tail is "alternative*, (unique | key | keyref)*" — two
			// separately-cardinalitied positions in that order, so an <alternative>
			// after an identity constraint is late even though both positions repeat.
			name:     "alternative after an identity constraint on a top-level element",
			topLevel: true,
			lines: []string{
				`<xs:element name="e" type="xs:string">`,
				`<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>`,
				`<xs:alternative type="xs:string"/>`,
				`</xs:element>`,
			},
			wantChild: "<alternative> at " + produceURI + ":4:1",
			wantOwner: "<element> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			// "simpleType?" on xs:attribute is not repeated, so the second inline base
			// is charged here rather than dropped by declaredType's first-child lookup.
			name:     "two simpleType children on a top-level attribute",
			topLevel: true,
			lines: []string{
				`<xs:attribute name="a">`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`</xs:attribute>`,
			},
			wantChild: "<simpleType> at " + produceURI + ":4:1",
			wantOwner: "<attribute> at " + produceURI + ":2:1",
			wantKind:  "repeats a position",
		},
		{
			name:     "annotation after the alternative of a top-level simpleType",
			topLevel: true,
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:restriction base="xs:string"/>`,
				`<xs:annotation/>`,
				`</xs:simpleType>`,
			},
			wantChild: "<annotation> at " + produceURI + ":4:1",
			wantOwner: "<simpleType> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			// The same walk one level in, on a LOCAL <element>: one model serves both
			// forms, so the local one is ordered exactly as the top-level one is.
			name: "annotation after the inline type of a local element",
			lines: []string{
				`<xs:sequence>`,
				`<xs:element name="e">`,
				`<xs:complexType/>`,
				`<xs:annotation/>`,
				`</xs:element>`,
				`</xs:sequence>`,
			},
			wantChild: "<annotation> at " + produceURI + ":6:1",
			wantOwner: "<element> at " + produceURI + ":4:1",
			wantKind:  "out of the child order",
		},
		{
			name:     "two alternatives on a top-level simpleType",
			topLevel: true,
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:restriction base="xs:string"/>`,
				`<xs:list itemType="xs:string"/>`,
				`</xs:simpleType>`,
			},
			wantChild: "<list> at " + produceURI + ":4:1",
			wantOwner: "<simpleType> at " + produceURI + ":2:1",
			wantKind:  "repeats a position",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := s4sDoc(tc.lines...)
			if tc.topLevel {
				doc = s4sTopLevelDoc(tc.lines...)
			}
			_, err := produce(t, doc)
			if err == nil {
				t.Fatal("Produce accepted a declaration whose children are out of the s4s order")
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
			}
			if !strings.Contains(err.Error(), tc.wantChild) {
				t.Errorf("error = %v, want it to name the offending %s", err, tc.wantChild)
			}
			if !strings.Contains(err.Error(), tc.wantOwner) {
				t.Errorf("error = %v, want it to name the owning %s", err, tc.wantOwner)
			}
			if !strings.Contains(err.Error(), tc.wantKind) {
				t.Errorf("error = %v, want the %q fault", err, tc.wantKind)
			}
		})
	}
}

// TestProduceS4SChildNoPositionRejected pins the OTHER fault the same walk
// charges: a child no position of the chosen content model admits at all, which
// the walk passed over silently until #1047. xs:complexTypeModel (:4757) is one
// xs:choice of three arms, so a <complexType> that wrote <simpleContent> or
// <complexContent> is on an arm holding "annotation?" and that one child alone —
// an <attribute> or a model group beside it belongs to the third arm, which this
// document did not write, and the document is not fully valid against Appendix A
// under any arm (§5.1's first bullet, :4296). The same holds one level in, where
// each wrapper and each alternant has a content model of its own.
//
// Every row asserts the fault names the offending child and the element that owns
// the model, and that it carries no rule ID: the class is uncataloged
// (xsderr/doc.go), exactly as the order and maxOccurs rows above.
func TestProduceS4SChildNoPositionRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name      string
		lines     []string
		topLevel  bool // lines are the <schema>'s own children, not a <complexType>'s
		wantChild string
		wantOwner string
	}{
		{
			// The largest suite shape: the attribute tail written beside a
			// <complexContent>, where only xs:complexTypeModel's third arm carries it.
			name: "attribute beside a complexContent on a complexType",
			lines: []string{
				`<xs:complexContent><xs:extension base="xs:anyType"/></xs:complexContent>`,
				`<xs:attribute name="a"/>`,
			},
			wantChild: "<attribute> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
		},
		{
			// Written BEFORE the wrapper, so the fault cannot be read as lateness.
			name: "model group before a simpleContent on a complexType",
			lines: []string{
				`<xs:sequence/>`,
				`<xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent>`,
			},
			wantChild: "<sequence> at " + produceURI + ":3:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
		},
		{
			name: "attribute under a simpleContent wrapper",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:extension base="xs:string"/>`,
				`<xs:attribute name="a"/>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<attribute> at " + produceURI + ":5:1",
			wantOwner: "<simpleContent> at " + produceURI + ":3:1",
		},
		{
			name: "anyAttribute under a complexContent wrapper",
			lines: []string{
				`<xs:complexContent>`,
				`<xs:extension base="xs:anyType"/>`,
				`<xs:anyAttribute namespace="##other"/>`,
				`</xs:complexContent>`,
			},
			wantChild: "<anyAttribute> at " + produceURI + ":5:1",
			wantOwner: "<complexContent> at " + produceURI + ":3:1",
		},
		{
			// xs:simpleExtensionType (:4979) has no structural position at all.
			name: "model group under a simpleContent extension",
			lines: []string{
				`<xs:simpleContent>`,
				`<xs:extension base="xs:string">`,
				`<xs:sequence/>`,
				`</xs:extension>`,
				`</xs:simpleContent>`,
			},
			wantChild: "<sequence> at " + produceURI + ":5:1",
			wantOwner: "<extension> at " + produceURI + ":4:1",
		},
		{
			// The mirror image: xs:complexRestrictionType (:4850) has no facet
			// position, which only xs:simpleRestrictionType carries.
			name: "facet under a complexContent restriction",
			lines: []string{
				`<xs:complexContent>`,
				`<xs:restriction base="xs:anyType">`,
				`<xs:length value="5"/>`,
				`</xs:restriction>`,
				`</xs:complexContent>`,
			},
			wantChild: "<length> at " + produceURI + ":5:1",
			wantOwner: "<restriction> at " + produceURI + ":4:1",
		},
		{
			// A name NO arm of xs:complexTypeModel carries, on the arm that carries
			// the most.
			name: "identity constraint on an implicit-content complexType",
			lines: []string{
				`<xs:sequence/>`,
				`<xs:key name="k"/>`,
			},
			wantChild: "<key> at " + produceURI + ":4:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
		},
		{
			// <simpleType> is a position of xs:simpleRestrictionType alone, and an
			// implicit-content <complexType> is not it.
			name: "simpleType on an implicit-content complexType",
			lines: []string{
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
			},
			wantChild: "<simpleType> at " + produceURI + ":3:1",
			wantOwner: "<complexType> at " + produceURI + ":2:1",
		},
		{
			// xs:element's model carries no attribute position at all: <attribute> is
			// xs:complexType's, and an <element> is not one.
			name:     "attribute on a top-level element",
			topLevel: true,
			lines: []string{
				`<xs:element name="e">`,
				`<xs:attribute name="a"/>`,
				`</xs:element>`,
			},
			wantChild: "<attribute> at " + produceURI + ":3:1",
			wantOwner: "<element> at " + produceURI + ":2:1",
		},
		{
			name:     "model group on a top-level attribute",
			topLevel: true,
			lines: []string{
				`<xs:attribute name="a">`,
				`<xs:sequence/>`,
				`</xs:attribute>`,
			},
			wantChild: "<sequence> at " + produceURI + ":3:1",
			wantOwner: "<attribute> at " + produceURI + ":2:1",
		},
		{
			name:     "attribute on a top-level simpleType",
			topLevel: true,
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:restriction base="xs:string"/>`,
				`<xs:attribute name="a"/>`,
				`</xs:simpleType>`,
			},
			wantChild: "<attribute> at " + produceURI + ":4:1",
			wantOwner: "<simpleType> at " + produceURI + ":2:1",
		},
		{
			// The <simpleType> that names no alternative at all is still charged over
			// the child, not over the missing body: the walk runs before simpleTypeBody.
			name:     "attribute on a top-level simpleType with no alternative",
			topLevel: true,
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:attribute name="a"/>`,
				`</xs:simpleType>`,
			},
			wantChild: "<attribute> at " + produceURI + ":3:1",
			wantOwner: "<simpleType> at " + produceURI + ":2:1",
		},
		{
			// Every LOCAL form is ordered against the same three models: an inline
			// <element>, the <attribute> of a complex type's tail, and the anonymous
			// <simpleType> that attribute owns.
			name: "attribute on a local element",
			lines: []string{
				`<xs:sequence>`,
				`<xs:element name="e">`,
				`<xs:attribute name="a"/>`,
				`</xs:element>`,
				`</xs:sequence>`,
			},
			wantChild: "<attribute> at " + produceURI + ":5:1",
			wantOwner: "<element> at " + produceURI + ":4:1",
		},
		{
			// The ref= form reads no child of its own, and is ordered all the same:
			// elementParticleTerm charges the model before it resolves the name.
			name: "attribute on a local element ref",
			lines: []string{
				`<xs:sequence>`,
				`<xs:element ref="tns:E">`,
				`<xs:attribute name="a"/>`,
				`</xs:element>`,
				`</xs:sequence>`,
			},
			wantChild: "<attribute> at " + produceURI + ":5:1",
			wantOwner: "<element> at " + produceURI + ":4:1",
		},
		{
			name: "model group on a local attribute",
			lines: []string{
				`<xs:attribute name="a">`,
				`<xs:sequence/>`,
				`</xs:attribute>`,
			},
			wantChild: "<sequence> at " + produceURI + ":4:1",
			wantOwner: "<attribute> at " + produceURI + ":3:1",
		},
		{
			// use="prohibited" maps to no component, which bounds what the subtree
			// CONTRIBUTES and not how §5.1 binds the way it is spelled.
			name: "model group on a prohibited local attribute",
			lines: []string{
				`<xs:attribute name="a" use="prohibited">`,
				`<xs:sequence/>`,
				`</xs:attribute>`,
			},
			wantChild: "<sequence> at " + produceURI + ":4:1",
			wantOwner: "<attribute> at " + produceURI + ":3:1",
		},
		{
			name: "attribute on the anonymous simpleType of a local attribute",
			lines: []string{
				`<xs:attribute name="a">`,
				`<xs:simpleType>`,
				`<xs:restriction base="xs:string"/>`,
				`<xs:attribute name="b"/>`,
				`</xs:simpleType>`,
				`</xs:attribute>`,
			},
			wantChild: "<attribute> at " + produceURI + ":6:1",
			wantOwner: "<simpleType> at " + produceURI + ":4:1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := s4sDoc(tc.lines...)
			if tc.topLevel {
				doc = s4sTopLevelDoc(tc.lines...)
			}
			_, err := produce(t, doc)
			if err == nil {
				t.Fatal("Produce accepted a declaration carrying a child no position of its content model admits")
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
			}
			if !strings.Contains(err.Error(), tc.wantChild) {
				t.Errorf("error = %v, want it to name the offending %s", err, tc.wantChild)
			}
			if !strings.Contains(err.Error(), tc.wantOwner) {
				t.Errorf("error = %v, want it to name the owning %s", err, tc.wantOwner)
			}
			if !strings.Contains(err.Error(), "fills no position") {
				t.Errorf("error = %v, want the %q fault rather than an order or maxOccurs one", err, "fills no position")
			}
		})
	}
}

// TestProduceS4SChildOrderAccepted is the other side of the check, and the row
// that matters most: a positional check written as a fixed linear order over
// element names would falsely reject most of these (PRINCIPLES 14). Every one is
// a legal permutation of the same content models.
func TestProduceS4SChildOrderAccepted(t *testing.T) {
	// A top-level <attributeGroup> the alternants below reference, so an
	// <attributeGroup ref> row is a complete document rather than a dangling one.
	const attrGroup = `<xs:attributeGroup name="AG"><xs:attribute name="g"/></xs:attributeGroup>`
	// A complex type with simple content for the <restriction> rows to derive
	// from: ct-props-correct clause 2 admits a simple base under <extension>
	// alone, so a simpleContent restriction needs a complex one, and its open
	// attribute wildcard is what lets derivation-ok-restriction clause 3 admit the
	// attribute uses those rows write.
	const simpleBase = `<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string">` +
		`<xs:anyAttribute namespace="##any"/></xs:extension></xs:simpleContent></xs:complexType>`
	// The same over xs:dateTime: cos-applicable-facets (§4.1.5,
	// xmlschema11-2.md:2823) applies <explicitTimezone> to the date/time primitives
	// alone, so over simpleBase the row below is rejected for inapplicability
	// instead and pins nothing about child order.
	const stampBase = `<xs:complexType name="T"><xs:simpleContent><xs:extension base="xs:dateTime">` +
		`<xs:anyAttribute namespace="##any"/></xs:extension></xs:simpleContent></xs:complexType>`
	// A top-level <element> for the <element ref> rows to denote, and a complex
	// type for the <alternative> rows to select between.
	const refTarget = `<xs:element name="E" type="xs:string"/>` +
		`<xs:complexType name="A"><xs:sequence/></xs:complexType>`
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			// The (attribute | attributeGroup)* block is a repeated CHOICE, so the
			// two names interleave freely inside it.
			name: "attribute and attributeGroup interleaved under a simpleContent extension",
			body: `<xs:complexType name="D"><xs:simpleContent><xs:extension base="xs:string">` +
				`<xs:attribute name="a"/><xs:attributeGroup ref="tns:AG"/><xs:attribute name="b"/>` +
				`<xs:attributeGroup ref="tns:AG"/><xs:anyAttribute namespace="##other"/>` +
				`<xs:assert test="true()"/><xs:assert test="true()"/>` +
				`</xs:extension></xs:simpleContent></xs:complexType>`,
		},
		{
			// Every position of xs:simpleRestrictionType filled, in order.
			name: "full simpleContent restriction in order",
			body: `<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">` +
				`<xs:annotation/><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>` +
				`<xs:minLength value="1"/><xs:maxLength value="8"/>` +
				`<xs:attribute name="a"/><xs:attributeGroup ref="tns:AG"/>` +
				`<xs:anyAttribute namespace="##other"/><xs:assert test="true()"/>` +
				`</xs:restriction></xs:simpleContent></xs:complexType>`,
		},
		{
			// Facet repetition is src-ct clause 2's to bound, not this check's: the
			// three names that clause excepts stay accepted.
			name: "repeated enumeration pattern and assertion facets",
			body: `<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:B">` +
				`<xs:enumeration value="a"/><xs:enumeration value="b"/>` +
				`<xs:pattern value="a"/><xs:pattern value="b"/>` +
				`<xs:assertion test="true()"/><xs:assertion test="true()"/>` +
				`<xs:attribute name="a"/></xs:restriction></xs:simpleContent></xs:complexType>`,
		},
		{
			// s4sFacetElement asks builtin.FacetKindByName rather than carrying a
			// name list, and since #1047 a facet name the position does not admit is
			// REJECTED rather than skipped — so that bridge's completeness is all
			// that stands between this row and a false reject. <explicitTimezone> is
			// the name a hand-typed list would have missed: the XML Representation
			// Summary s4sSimpleRestriction quotes (xmlschema11-1.md:1692) omits it
			// where Appendix A's grammar admits it.
			name: "explicitTimezone facet under a simpleContent restriction",
			body: `<xs:complexType name="D"><xs:simpleContent><xs:restriction base="tns:T">` +
				`<xs:explicitTimezone value="required"/><xs:attribute name="a"/>` +
				`</xs:restriction></xs:simpleContent></xs:complexType>`,
		},
		{
			name: "full complexContent restriction in order",
			body: `<xs:complexType name="D"><xs:complexContent><xs:restriction base="xs:anyType">` +
				`<xs:annotation/><xs:openContent mode="interleave">` +
				`<xs:any namespace="##other" processContents="lax"/></xs:openContent>` +
				`<xs:sequence/><xs:attribute name="a"/><xs:attributeGroup ref="tns:AG"/>` +
				`<xs:anyAttribute namespace="##other"/><xs:assert test="true()"/>` +
				`</xs:restriction></xs:complexContent></xs:complexType>`,
		},
		{
			name: "full complexContent extension in order",
			body: `<xs:complexType name="D"><xs:complexContent><xs:extension base="xs:anyType">` +
				`<xs:annotation/><xs:sequence/><xs:attributeGroup ref="tns:AG"/><xs:attribute name="a"/>` +
				`<xs:anyAttribute namespace="##other"/><xs:assert test="true()"/>` +
				`</xs:extension></xs:complexContent></xs:complexType>`,
		},
		{
			name: "full implicit content in order",
			body: `<xs:complexType name="D"><xs:annotation/>` +
				`<xs:openContent mode="interleave"><xs:any namespace="##other" processContents="lax"/></xs:openContent>` +
				`<xs:sequence/><xs:attribute name="a"/><xs:attributeGroup ref="tns:AG"/>` +
				`<xs:anyAttribute namespace="##other"/><xs:assert test="true()"/></xs:complexType>`,
		},
		{
			name: "annotation before the wrapper of a complexType",
			body: `<xs:complexType name="D"><xs:annotation/><xs:simpleContent>` +
				`<xs:annotation/><xs:extension base="xs:string"/>` +
				`</xs:simpleContent></xs:complexType>`,
		},
		{
			// A child outside the XSD namespace is stepped over whatever position it
			// sits in — #928's fault to charge if any — where an XSD-namespace name
			// no position admits is rejected by the test above.
			name: "a foreign-namespace child among the facets",
			body: `<xs:complexType name="D" xmlns:o="urn:other"><xs:simpleContent>` +
				`<xs:restriction base="tns:B"><o:hint/><xs:length value="5"/>` +
				`<xs:attribute name="a"/></xs:restriction></xs:simpleContent></xs:complexType>`,
		},
		{
			// Every position of xs:element filled, in order, with both tail positions
			// repeated: "alternative*" then "(unique | key | keyref)*".
			name: "full top-level element in order",
			body: `<xs:element name="D" type="tns:A"><xs:annotation/>` +
				`<xs:alternative test="true()" type="tns:A"/><xs:alternative type="tns:A"/>` +
				`<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>` +
				`<xs:key name="k"><xs:selector xpath="b"/><xs:field xpath="@y"/></xs:key>` +
				`<xs:keyref name="kr" refer="tns:k"><xs:selector xpath="c"/><xs:field xpath="@z"/></xs:keyref>` +
				`</xs:element>`,
		},
		{
			// The inline type arm of the same single "(simpleType | complexType)?"
			// position, on a LOCAL <element>, beside the ref= form that fills none of it.
			name: "local element inline type and local element ref in one content model",
			body: `<xs:complexType name="D"><xs:sequence>` +
				`<xs:element name="a"><xs:annotation/><xs:simpleType>` +
				`<xs:restriction base="xs:string"/></xs:simpleType>` +
				`<xs:unique name="u2"><xs:selector xpath="p"/><xs:field xpath="@q"/></xs:unique></xs:element>` +
				`<xs:element ref="tns:E"><xs:annotation/></xs:element>` +
				`</xs:sequence></xs:complexType>`,
		},
		{
			// xs:attribute's whole model, on the top-level and the local form alike,
			// with the local one's use="prohibited" mapping to no component at all.
			name: "attribute annotation and simpleType in order, top-level and local",
			body: `<xs:attribute name="G"><xs:annotation/><xs:simpleType>` +
				`<xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>` +
				`<xs:complexType name="D"><xs:attribute name="a"><xs:annotation/>` +
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>` +
				`<xs:attribute name="b" use="prohibited"><xs:annotation/></xs:attribute></xs:complexType>`,
		},
		{
			// All three of xs:simpleType's alternatives, each behind the annotation
			// position, top-level and inline.
			name: "simpleType annotation before each of the three alternatives",
			body: `<xs:simpleType name="R"><xs:annotation/><xs:restriction base="xs:string"/></xs:simpleType>` +
				`<xs:simpleType name="L"><xs:annotation/><xs:list itemType="xs:string"/></xs:simpleType>` +
				`<xs:simpleType name="U"><xs:annotation/><xs:union memberTypes="xs:string"/></xs:simpleType>` +
				`<xs:element name="D"><xs:simpleType><xs:annotation/>` +
				`<xs:restriction base="xs:string"/></xs:simpleType></xs:element>`,
		},
		{
			// None of the three models has a wildcard position, and none needs one: the
			// namespace guard steps over a foreign child wherever it sits.
			name: "foreign-namespace children under the three declarations",
			body: `<xs:element name="D" xmlns:o="urn:other"><o:hint/><xs:simpleType>` +
				`<o:hint/><xs:restriction base="xs:string"/></xs:simpleType>` +
				`<o:hint/><xs:unique name="u3"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>` +
				`</xs:element>` +
				`<xs:attribute name="H" xmlns:o="urn:other"><o:hint/>` +
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:x", attrGroup+simpleBase+stampBase+refTarget+tc.body)); err != nil {
				t.Fatalf("Produce rejected a legal permutation: %v", err)
			}
		})
	}
}
