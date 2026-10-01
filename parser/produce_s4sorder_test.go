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
// ordered by ONE model whichever form it is written in — and since #1275
// xs:altType (:3210) on the same terms.
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
			// xs:element opens "annotation?" AHEAD of the "(simpleType | complexType)?"
			// choice (:5090-5094 topLevelElement, :5114-5118 localElement), so an
			// <annotation> written behind the inline type is late, not repeated and
			// not unadmitted.
			name:     "annotation after the inline simpleType of a top-level element (elemQ004)",
			topLevel: true,
			lines: []string{
				`<xs:element name="myElem">`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`<xs:annotation/>`,
				`</xs:element>`,
			},
			wantChild: "<annotation> at " + produceURI + ":4:1",
			wantOwner: "<element> at " + produceURI + ":2:1",
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
		{
			// The blind spot checkSrcTA cannot see (#1275): it counts FORMS through
			// childElement, which answers with the FIRST <simpleType> alone, so a
			// second one left the count at 1 and passed. s4sAlternative's single
			// "(simpleType | complexType)?" position charges it.
			name:     "two simpleType children on an alternative",
			topLevel: true,
			lines: []string{
				`<xs:element name="e" type="xs:string">`,
				`<xs:alternative test="true()">`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`</xs:alternative>`,
				`</xs:element>`,
			},
			wantChild: "<simpleType> at " + produceURI + ":5:1",
			wantOwner: "<alternative> at " + produceURI + ":3:1",
			wantKind:  "repeats a position",
		},
		{
			// The two names share ONE position, so the second is a repeat of it and
			// not a second optional — and this walk answers it ahead of src-ta, which
			// counts the same document as two forms (checkSrcTA, produce_typetable.go).
			name:     "complexType after a simpleType on an alternative",
			topLevel: true,
			lines: []string{
				`<xs:element name="e" type="xs:string">`,
				`<xs:alternative test="true()">`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`<xs:complexType><xs:sequence/></xs:complexType>`,
				`</xs:alternative>`,
				`</xs:element>`,
			},
			wantChild: "<complexType> at " + produceURI + ":5:1",
			wantOwner: "<alternative> at " + produceURI + ":3:1",
			wantKind:  "repeats a position",
		},
		{
			// xs:altType opens with the "annotation?" xs:annotated contributes, so an
			// <annotation> written after the type child is late.
			name:     "annotation after the type child of an alternative",
			topLevel: true,
			lines: []string{
				`<xs:element name="e" type="xs:string">`,
				`<xs:alternative test="true()">`,
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`,
				`<xs:annotation/>`,
				`</xs:alternative>`,
				`</xs:element>`,
			},
			wantChild: "<annotation> at " + produceURI + ":5:1",
			wantOwner: "<alternative> at " + produceURI + ":3:1",
			wantKind:  "out of the child order",
		},
		{
			// xs:namedAttributeGroup puts "anyAttribute?" after the attribute block, so
			// an <attribute> written once <anyAttribute> has matched cannot re-enter
			// the block behind it (#1876).
			name:     "attribute after the anyAttribute of a top-level attributeGroup",
			topLevel: true,
			lines: []string{
				`<xs:attributeGroup name="g">`,
				`<xs:anyAttribute/>`,
				`<xs:attribute name="a"/>`,
				`</xs:attributeGroup>`,
			},
			wantChild: "<attribute> at " + produceURI + ":4:1",
			wantOwner: "<attributeGroup> at " + produceURI + ":2:1",
			wantKind:  "out of the child order",
		},
		{
			name:     "second anyAttribute of a top-level attributeGroup",
			topLevel: true,
			lines: []string{
				`<xs:attributeGroup name="g">`,
				`<xs:anyAttribute/>`,
				`<xs:anyAttribute/>`,
				`</xs:attributeGroup>`,
			},
			wantChild: "<anyAttribute> at " + produceURI + ":4:1",
			wantOwner: "<attributeGroup> at " + produceURI + ":2:1",
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
		{
			// xs:altType carries the two inline type names and nothing else, so an
			// <element> under an <alternative> fills no position of it — the name
			// checkSrcTA's form count never looked at (#1275).
			name:     "element child on an alternative",
			topLevel: true,
			lines: []string{
				`<xs:element name="e" type="xs:string">`,
				`<xs:alternative test="true()" type="xs:string">`,
				`<xs:element name="f"/>`,
				`</xs:alternative>`,
				`</xs:element>`,
			},
			wantChild: "<element> at " + produceURI + ":4:1",
			wantOwner: "<alternative> at " + produceURI + ":3:1",
		},
		{
			// xs:namedGroup is "(annotation?, (all | choice | sequence))": a child
			// BESIDE the body fills no position of it, after the body or before it,
			// and compositorChild's first-match read used to drop it (#1876).
			name:     "unique after the body of a named group",
			topLevel: true,
			lines: []string{
				`<xs:group name="g">`,
				`<xs:sequence/>`,
				`<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>`,
				`</xs:group>`,
			},
			wantChild: "<unique> at " + produceURI + ":4:1",
			wantOwner: "<group> at " + produceURI + ":2:1",
		},
		{
			name:     "unique before the body of a named group",
			topLevel: true,
			lines: []string{
				`<xs:group name="g">`,
				`<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>`,
				`<xs:sequence/>`,
				`</xs:group>`,
			},
			wantChild: "<unique> at " + produceURI + ":3:1",
			wantOwner: "<group> at " + produceURI + ":2:1",
		},
		{
			name:     "element after the body of a named group",
			topLevel: true,
			lines: []string{
				`<xs:group name="g">`,
				`<xs:sequence/>`,
				`<xs:element name="x"/>`,
				`</xs:group>`,
			},
			wantChild: "<element> at " + produceURI + ":4:1",
			wantOwner: "<group> at " + produceURI + ":2:1",
		},
		{
			// attgD012's shape: xs:namedAttributeGroup carries xs:attrDecls alone, and
			// collectAttributeContent used to drop an <element> in silence (#1876).
			name:     "element child of a top-level attributeGroup",
			topLevel: true,
			lines: []string{
				`<xs:attributeGroup name="g">`,
				`<xs:attribute name="a"/>`,
				`<xs:element name="x"/>`,
				`</xs:attributeGroup>`,
			},
			wantChild: "<element> at " + produceURI + ":4:1",
			wantOwner: "<attributeGroup> at " + produceURI + ":2:1",
		},
		{
			// groupO025's shape: a model group reference is no attribute declaration.
			name:     "group ref child of a top-level attributeGroup",
			topLevel: true,
			lines: []string{
				`<xs:attributeGroup name="g">`,
				`<xs:group ref="tns:G"/>`,
				`</xs:attributeGroup>`,
				`<xs:group name="G"><xs:sequence/></xs:group>`,
			},
			wantChild: "<group> at " + produceURI + ":3:1",
			wantOwner: "<attributeGroup> at " + produceURI + ":2:1",
		},
		{
			// The attribute tail of a complex type ends in "assert*"; xs:attrDecls
			// does not.
			name:     "assert child of a top-level attributeGroup",
			topLevel: true,
			lines: []string{
				`<xs:attributeGroup name="g">`,
				`<xs:assert test="true()"/>`,
				`</xs:attributeGroup>`,
			},
			wantChild: "<assert> at " + produceURI + ":3:1",
			wantOwner: "<attributeGroup> at " + produceURI + ":2:1",
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
			// xs:simpleRestrictionModel's "{any with namespace: ##other}" arm shares the
			// repeated facet position, so a foreign child interleaves with the facets
			// (#1982); under every other model TestProduceS4SForeignChildRejected
			// rejects it.
			name: "foreign-namespace children among the facets",
			body: `<xs:complexType name="D" xmlns:o="urn:other"><xs:simpleContent>` +
				`<xs:restriction base="tns:B"><o:hint/><xs:length value="5"/><o:hint/>` +
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
			// Every position of xs:altType filled, over both arms of its one
			// "(simpleType | complexType)?" choice and beside the whole attribute set
			// the type carries — test, type and xpathDefaultNamespace.
			name: "alternative annotation before each inline type in turn",
			body: `<xs:element name="D" type="tns:A">` +
				`<xs:alternative test="true()" xpathDefaultNamespace="##targetNamespace">` +
				`<xs:annotation/><xs:complexType><xs:complexContent>` +
				`<xs:restriction base="tns:A"><xs:sequence/></xs:restriction>` +
				`</xs:complexContent></xs:complexType></xs:alternative>` +
				`<xs:alternative type="tns:A"/></xs:element>` +
				`<xs:element name="D2" type="xs:string">` +
				`<xs:alternative test="true()"><xs:annotation/>` +
				`<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:alternative>` +
				`<xs:alternative type="xs:string"/></xs:element>`,
		},
		{
			// Every position of xs:namedAttributeGroup filled, the attribute block
			// interleaved; and a named group's one body behind its annotation
			// (#1876). Neither model has an ##other element position, so no foreign
			// child is written here.
			name: "top-level attributeGroup and named group in order",
			body: `<xs:attributeGroup name="AG2"><xs:annotation/>` +
				`<xs:attribute name="p"/><xs:attributeGroup ref="tns:AG"/><xs:attribute name="q"/>` +
				`<xs:anyAttribute namespace="##other"/></xs:attributeGroup>` +
				`<xs:group name="G"><xs:annotation/>` +
				`<xs:choice><xs:element name="c"/></xs:choice></xs:group>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:x", attrGroup+simpleBase+stampBase+refTarget+tc.body)); err != nil {
				t.Fatalf("Produce rejected a legal permutation: %v", err)
			}
		})
	}
}

// TestProduceS4SSimpleTypeAndOpenContentModelsRejected pins the four models #1951
// adds: xs:list (xmlschema11-2.md:2753), a <simpleType>'s xs:restriction (:2748),
// xs:openContent (xmlschema11-1.md:1728) and xs:defaultOpenContent (:3787). Each
// document assembled clean before them; the first row is stD015's shape.
//
// Each row pins the diagnostic's opening "parser: <child> at <loc>" as a PREFIX,
// so a message naming the owner where it means the child fails here, and the
// grammar the rejection quotes, so a fault charged by some other model does too.
func TestProduceS4SSimpleTypeAndOpenContentModelsRejected(t *testing.T) {
	const at = " at " + produceURI + ":"
	const inline = `<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name      string
		lines     []string // the <schema>'s own children, from line 2
		wantChild string   // "<local> at <uri>:<line>:1"
		wantOwner string
		grammar   string
		wantKind  string
	}{
		{
			name: "second simpleType of a list",
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:list>`,
				`<xs:annotation/>`,
				inline,
				inline,
				`</xs:list>`,
				`</xs:simpleType>`,
			},
			wantChild: "<simpleType>" + at + "6:1",
			wantOwner: "<list>" + at + "3:1",
			grammar:   "xs:list",
			wantKind:  "repeats a position",
		},
		{
			name: "element child of a list with itemType",
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:list itemType="xs:string">`,
				`<xs:element name="e"/>`,
				`</xs:list>`,
				`</xs:simpleType>`,
			},
			wantChild: "<element>" + at + "4:1",
			wantOwner: "<list>" + at + "3:1",
			grammar:   "xs:list",
			wantKind:  "fills no position",
		},
		{
			name: "annotation after a facet of a simpleType restriction",
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:restriction base="xs:string">`,
				`<xs:maxLength value="3"/>`,
				`<xs:annotation/>`,
				`</xs:restriction>`,
				`</xs:simpleType>`,
			},
			wantChild: "<annotation>" + at + "5:1",
			wantOwner: "<restriction>" + at + "3:1",
			grammar:   "xs:restriction",
			wantKind:  "out of the child order",
		},
		{
			name: "annotation after the inline base of a simpleType restriction",
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:restriction>`,
				inline,
				`<xs:annotation/>`,
				`</xs:restriction>`,
				`</xs:simpleType>`,
			},
			wantChild: "<annotation>" + at + "5:1",
			wantOwner: "<restriction>" + at + "3:1",
			grammar:   "xs:restriction",
			wantKind:  "out of the child order",
		},
		{
			name: "inline base after a facet of a simpleType restriction",
			lines: []string{
				`<xs:simpleType name="S">`,
				`<xs:restriction>`,
				`<xs:maxLength value="3"/>`,
				inline,
				`</xs:restriction>`,
				`</xs:simpleType>`,
			},
			wantChild: "<simpleType>" + at + "5:1",
			wantOwner: "<restriction>" + at + "3:1",
			grammar:   "xs:restriction",
			wantKind:  "out of the child order",
		},
		{
			name: "annotation after the any of an openContent",
			lines: []string{
				`<xs:complexType name="C">`,
				`<xs:openContent>`,
				`<xs:any namespace="##other"/>`,
				`<xs:annotation/>`,
				`</xs:openContent>`,
				`<xs:sequence/>`,
				`</xs:complexType>`,
			},
			wantChild: "<annotation>" + at + "5:1",
			wantOwner: "<openContent>" + at + "3:1",
			grammar:   "xs:openContent",
			wantKind:  "out of the child order",
		},
		{
			name: "second any of an openContent",
			lines: []string{
				`<xs:complexType name="C">`,
				`<xs:openContent>`,
				`<xs:any namespace="##other"/>`,
				`<xs:any namespace="##other"/>`,
				`</xs:openContent>`,
				`<xs:sequence/>`,
				`</xs:complexType>`,
			},
			wantChild: "<any>" + at + "5:1",
			wantOwner: "<openContent>" + at + "3:1",
			grammar:   "xs:openContent",
			wantKind:  "repeats a position",
		},
		{
			name: "annotation after the any of a defaultOpenContent",
			lines: []string{
				`<xs:defaultOpenContent>`,
				`<xs:any namespace="##other"/>`,
				`<xs:annotation/>`,
				`</xs:defaultOpenContent>`,
				`<xs:complexType name="C"><xs:sequence/></xs:complexType>`,
			},
			wantChild: "<annotation>" + at + "4:1",
			wantOwner: "<defaultOpenContent>" + at + "2:1",
			grammar:   "xs:defaultOpenContent",
			wantKind:  "out of the child order",
		},
		{
			name: "second any of a defaultOpenContent",
			lines: []string{
				`<xs:defaultOpenContent>`,
				`<xs:any namespace="##other"/>`,
				`<xs:any namespace="##other"/>`,
				`</xs:defaultOpenContent>`,
				`<xs:complexType name="C"><xs:sequence/></xs:complexType>`,
			},
			wantChild: "<any>" + at + "4:1",
			wantOwner: "<defaultOpenContent>" + at + "2:1",
			grammar:   "xs:defaultOpenContent",
			wantKind:  "repeats a position",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, s4sTopLevelDoc(tc.lines...))
			if err == nil {
				t.Fatal("Produce accepted a document its s4s content model does not admit")
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
			}
			if !strings.HasPrefix(err.Error(), "parser: "+tc.wantChild+" ") {
				t.Errorf("error = %v, want it to open with the offending %s", err, tc.wantChild)
			}
			if !strings.Contains(err.Error(), tc.wantOwner) {
				t.Errorf("error = %v, want it to name the owning %s", err, tc.wantOwner)
			}
			if !strings.Contains(err.Error(), tc.grammar+"'s content model") {
				t.Errorf("error = %v, want it to quote %s's content model", err, tc.grammar)
			}
			if !strings.Contains(err.Error(), tc.wantKind) {
				t.Errorf("error = %v, want the %q fault", err, tc.wantKind)
			}
		})
	}
}

// TestProduceS4SSimpleTypeAndOpenContentGuardsKeepTheirVerdicts pins the guards
// that charged these four owners before #1951 gave them models: on every shape
// below the older guard answers, never the new walk — either because the model
// admits the shape (src-simple-type clauses 2 and 3, src-ct clauses 3 and 4, the
// mandatory <any>, each ruled no model fault by #1951's grounding), or because
// the guard runs first on a shape both reach (checkS4SChildOrder's doc names the
// two exceptions to its order this pins).
func TestProduceS4SSimpleTypeAndOpenContentGuardsKeepTheirVerdicts(t *testing.T) {
	const inline = `<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`
	const ct = `<xs:complexType name="C"><xs:sequence/></xs:complexType>`
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name     string
		body     string
		wantRule string // "" for a plain grammar fault
		want     string // a phrase of the older guard's message
	}{
		{
			// rejectLateAnnotation, whose message a list keeps: why s4sAnnotationLedOwner
			// still admits <list>.
			name: "late annotation of a list",
			body: `<xs:simpleType name="S"><xs:list>` + inline + `<xs:annotation/></xs:list></xs:simpleType>`,
			want: "follows the <simpleType>",
		},
		{
			name: "late annotation of a union",
			body: `<xs:simpleType name="S"><xs:union memberTypes="xs:string">` + inline + `<xs:annotation/></xs:union></xs:simpleType>`,
			want: "follows the <simpleType>",
		},
		{
			name:     "itemType and an inline simpleType on a list",
			body:     `<xs:simpleType name="S"><xs:list itemType="xs:string">` + inline + `</xs:list></xs:simpleType>`,
			wantRule: "src-simple-type",
			want:     "clause 3",
		},
		{
			// The first exception: clause 1 runs ahead of the restriction's walk.
			name:     "second simpleType of a simpleType restriction",
			body:     `<xs:simpleType name="S"><xs:restriction><xs:annotation/>` + inline + inline + `</xs:restriction></xs:simpleType>`,
			wantRule: "src-simple-type",
			want:     "clause 1",
		},
		{
			name:     "base and an inline simpleType on a simpleType restriction",
			body:     `<xs:simpleType name="S"><xs:restriction base="xs:string">` + inline + `</xs:restriction></xs:simpleType>`,
			wantRule: "src-simple-type",
			want:     "clause 2",
		},
		{
			// rejectOutOfModelFacetChildren answers an out-of-model name ahead of the
			// late <annotation> the walk would charge.
			name: "late annotation beside an out-of-model child of a simpleType restriction",
			body: `<xs:simpleType name="S"><xs:restriction base="xs:string"><xs:maxLength value="3"/>` +
				`<xs:annotation/><xs:encoding value="x"/></xs:restriction></xs:simpleType>`,
			want: "<encoding>",
		},
		{
			name:     "openContent with no any",
			body:     `<xs:complexType name="C"><xs:openContent><xs:annotation/></xs:openContent><xs:sequence/></xs:complexType>`,
			wantRule: "src-ct",
			want:     "clause 3",
		},
		{
			// The second exception: clause 3 runs ahead of the openContent's walk.
			name:     "openContent with no any and an element child",
			body:     `<xs:complexType name="C"><xs:openContent><xs:element name="e"/></xs:openContent><xs:sequence/></xs:complexType>`,
			wantRule: "src-ct",
			want:     "clause 3",
		},
		{
			name:     "openContent mode none with an any",
			body:     `<xs:complexType name="C"><xs:openContent mode="none"><xs:any/></xs:openContent><xs:sequence/></xs:complexType>`,
			wantRule: "src-ct",
			want:     "clause 4",
		},
		{
			name:     "openContent mode none with two any",
			body:     `<xs:complexType name="C"><xs:openContent mode="none"><xs:any/><xs:any/></xs:openContent><xs:sequence/></xs:complexType>`,
			wantRule: "src-ct",
			want:     "clause 4",
		},
		{
			name: "defaultOpenContent with no any and an element child",
			body: `<xs:defaultOpenContent><xs:element name="e"/></xs:defaultOpenContent>` + ct,
			want: "has no <any> child",
		},
		{
			name: "defaultOpenContent mode none with a late annotation",
			body: `<xs:defaultOpenContent mode="none"><xs:any/><xs:annotation/></xs:defaultOpenContent>` + ct,
			want: `has mode="none"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", tc.body))
			if err == nil {
				t.Fatal("Produce accepted the document")
			}
			rule, ok := xsderr.RuleOf(err)
			if string(rule) != tc.wantRule || ok != (tc.wantRule != "") {
				t.Errorf("error = %v, charged %q; want %q", err, rule, tc.wantRule)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error = %v, want the older guard's %q", err, tc.want)
			}
		})
	}
}

// TestProduceS4SSimpleTypeAndOpenContentModelsAccepted is the other side of the
// four models: every position filled in order, the precisionDecimal scale facets
// the facet position must hold (xsd-precisionDecimal.md), foreign children among
// the facets (the "{any with namespace: ##other}" arm), and a child §4.2.2's
// conditional-inclusion pre-pass removes before any walk — written where, kept,
// it would be out of order.
func TestProduceS4SSimpleTypeAndOpenContentModelsAccepted(t *testing.T) {
	const inline = `<xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType>`
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: "list annotation and inline item",
			body: `<xs:simpleType name="S"><xs:list><xs:annotation/>` + inline + `</xs:list></xs:simpleType>`,
		},
		{
			name: "simpleType restriction in order with foreign children among the facets",
			body: `<xs:simpleType name="S" xmlns:o="urn:other"><xs:restriction><xs:annotation/>` + inline +
				`<xs:minLength value="1"/><o:hint/><xs:maxLength value="8"/><o:hint/>` +
				`<xs:enumeration value="a"/><xs:enumeration value="b"/></xs:restriction></xs:simpleType>`,
		},
		{
			name: "precisionDecimal scale facets",
			body: `<xs:simpleType name="S"><xs:restriction base="xs:precisionDecimal"><xs:annotation/>` +
				`<xs:minScale value="1"/><xs:maxScale value="3"/></xs:restriction></xs:simpleType>`,
		},
		{
			name: "late annotation excluded by vc:minVersion",
			body: `<xs:simpleType name="S"><xs:restriction base="xs:string"><xs:maxLength value="3"/>` +
				`<xs:annotation vc:minVersion="5.0"/></xs:restriction></xs:simpleType>`,
		},
		{
			name: "openContent annotation and any, and mode none with annotation alone",
			body: `<xs:complexType name="C"><xs:openContent><xs:annotation/><xs:any namespace="##other"/>` +
				`</xs:openContent><xs:sequence/></xs:complexType>` +
				`<xs:complexType name="N"><xs:openContent mode="none"><xs:annotation/>` +
				`</xs:openContent><xs:sequence/></xs:complexType>`,
		},
		{
			name: "defaultOpenContent annotation and any",
			body: `<xs:defaultOpenContent><xs:annotation/><xs:any namespace="##other"/></xs:defaultOpenContent>` +
				`<xs:complexType name="C"><xs:sequence/></xs:complexType>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, vcWrap("", tc.body)); err != nil {
				t.Fatalf("Produce rejected a document its s4s content model admits: %v", err)
			}
		})
	}
}

// TestProduceS4SForeignChildRejected pins #1982: a child element outside the XSD
// namespace fills a position of exactly one walked content model's positions —
// xs:simpleRestrictionModel's "{any with namespace: ##other}" arm, at the facet
// position of a <simpleContent>'s and a <simpleType>'s <restriction>. Under every
// other model the walk reaches it is charged §5.1's first bullet (:4296), as an
// XSD-namespace name no position admits is; a child in NO namespace is charged at
// the facet position too, since ##other excludes ·absent·; and a foreign child
// after the attribute tail of a <simpleContent> <restriction> is out of order,
// since the wildcard arm shares the facet position the tail follows.
//
// Every row assembled clean before #1982. Each pins the diagnostic's opening
// "parser: <child> at <loc>" as a PREFIX, the owner, the grammar quoted, and the
// closing clause naming the fault and the child's namespace.
func TestProduceS4SForeignChildRejected(t *testing.T) {
	const at = " at " + produceURI + ":"
	const o = `<o:hint xmlns:o="urn:other"/>`
	const ct = `<xs:complexType name="C">`
	const ctEnd = `</xs:complexType>`
	const elem = `<xs:element name="e">`
	const elemEnd = `</xs:element>`
	// A complex type with simple content for the <simpleContent> <restriction>
	// rows to derive from, on its own line 2.
	const simpleBase = `<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`
	const foreign = `<{urn:other}hint>`
	const noPosition = `which admits no element in namespace "urn:other" in any position`
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name      string
		lines     []string // the <schema>'s own children, from line 2
		wantChild string   // "<name> at <uri>:<line>:1"
		wantOwner string
		grammar   string
		wantTail  string
	}{
		{
			name:      "after the body of a named group",
			lines:     []string{`<xs:group name="g">`, `<xs:sequence/>`, o, `</xs:group>`},
			wantChild: foreign + at + "4:1",
			wantOwner: "<group>" + at + "2:1",
			grammar:   "xs:namedGroup",
			wantTail:  noPosition,
		},
		{
			name:      "alone under a top-level attributeGroup",
			lines:     []string{`<xs:attributeGroup name="g">`, o, `</xs:attributeGroup>`},
			wantChild: foreign + at + "3:1",
			wantOwner: "<attributeGroup>" + at + "2:1",
			grammar:   "xs:namedAttributeGroup",
			wantTail:  noPosition,
		},
		{
			name:      "after the model group of an implicit-content complexType",
			lines:     []string{ct, `<xs:sequence/>`, o, ctEnd},
			wantChild: foreign + at + "4:1",
			wantOwner: "<complexType>" + at + "2:1",
			grammar:   "xs:complexTypeModel",
			wantTail:  noPosition,
		},
		{
			name:      "beside the simpleContent of a complexType",
			lines:     []string{ct, `<xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent>`, o, ctEnd},
			wantChild: foreign + at + "4:1",
			wantOwner: "<complexType>" + at + "2:1",
			grammar:   "xs:complexTypeModel",
			wantTail:  noPosition,
		},
		{
			name:      "beside the alternant of a simpleContent",
			lines:     []string{ct, `<xs:simpleContent>`, `<xs:extension base="xs:string"/>`, o, `</xs:simpleContent>`, ctEnd},
			wantChild: foreign + at + "5:1",
			wantOwner: "<simpleContent>" + at + "3:1",
			grammar:   "xs:simpleContent",
			wantTail:  noPosition,
		},
		{
			name:      "beside the alternant of a complexContent",
			lines:     []string{ct, `<xs:complexContent>`, `<xs:extension base="xs:anyType"/>`, o, `</xs:complexContent>`, ctEnd},
			wantChild: foreign + at + "5:1",
			wantOwner: "<complexContent>" + at + "3:1",
			grammar:   "xs:complexContent",
			wantTail:  noPosition,
		},
		{
			name:      "under a simpleContent extension",
			lines:     []string{ct, `<xs:simpleContent>`, `<xs:extension base="xs:string">`, o, `</xs:extension>`, `</xs:simpleContent>`, ctEnd},
			wantChild: foreign + at + "5:1",
			wantOwner: "<extension>" + at + "4:1",
			grammar:   "xs:simpleExtensionType",
			wantTail:  noPosition,
		},
		{
			// xs:restrictionType (:4831) offers xs:simpleRestrictionModel, and its
			// ##other arm, as a choice xs:complexRestrictionType (:4850) drops.
			name:      "under a complexContent restriction",
			lines:     []string{ct, `<xs:complexContent>`, `<xs:restriction base="xs:anyType">`, o, `</xs:restriction>`, `</xs:complexContent>`, ctEnd},
			wantChild: foreign + at + "5:1",
			wantOwner: "<restriction>" + at + "4:1",
			grammar:   "xs:complexRestrictionType",
			wantTail:  noPosition,
		},
		{
			name:      "under a complexContent extension",
			lines:     []string{ct, `<xs:complexContent>`, `<xs:extension base="xs:anyType">`, o, `</xs:extension>`, `</xs:complexContent>`, ctEnd},
			wantChild: foreign + at + "5:1",
			wantOwner: "<extension>" + at + "4:1",
			grammar:   "xs:extensionType",
			wantTail:  noPosition,
		},
		{
			name:      "under a top-level element",
			lines:     []string{elem, o, elemEnd},
			wantChild: foreign + at + "3:1",
			wantOwner: "<element>" + at + "2:1",
			grammar:   "xs:element",
			wantTail:  noPosition,
		},
		{
			// src-element clause 2.2 reaches the XSD namespace alone, so the
			// grammar fault, carrying no rule ID, is the one charged.
			name:      "under a local element ref",
			lines:     []string{ct, `<xs:sequence>`, `<xs:element ref="tns:E">`, `<xs:annotation/>`, o, elemEnd, `</xs:sequence>`, ctEnd, `<xs:element name="E"/>`},
			wantChild: foreign + at + "6:1",
			wantOwner: "<element>" + at + "4:1",
			grammar:   "xs:element",
			wantTail:  noPosition,
		},
		{
			name:      "under a top-level attribute",
			lines:     []string{`<xs:attribute name="a">`, o, `</xs:attribute>`},
			wantChild: foreign + at + "3:1",
			wantOwner: "<attribute>" + at + "2:1",
			grammar:   "xs:attribute",
			wantTail:  noPosition,
		},
		{
			name:      "beside the restriction of a simpleType",
			lines:     []string{`<xs:simpleType name="S">`, `<xs:restriction base="xs:string"/>`, o, `</xs:simpleType>`},
			wantChild: foreign + at + "4:1",
			wantOwner: "<simpleType>" + at + "2:1",
			grammar:   "xs:simpleType",
			wantTail:  noPosition,
		},
		{
			name:      "under an alternative",
			lines:     []string{`<xs:element name="e" type="xs:string">`, `<xs:alternative test="true()" type="xs:string">`, o, `</xs:alternative>`, elemEnd},
			wantChild: foreign + at + "4:1",
			wantOwner: "<alternative>" + at + "3:1",
			grammar:   "xs:altType",
			wantTail:  noPosition,
		},
		{
			name:      "after the fields of a unique",
			lines:     []string{elem, `<xs:unique name="u">`, `<xs:selector xpath="a"/>`, `<xs:field xpath="@x"/>`, o, `</xs:unique>`, elemEnd},
			wantChild: foreign + at + "6:1",
			wantOwner: "<unique>" + at + "3:1",
			grammar:   "xs:keybase",
			wantTail:  noPosition,
		},
		{
			name:      "under a selector",
			lines:     []string{elem, `<xs:unique name="u">`, `<xs:selector xpath="a">`, o, `</xs:selector>`, `<xs:field xpath="@x"/>`, `</xs:unique>`, elemEnd},
			wantChild: foreign + at + "5:1",
			wantOwner: "<selector>" + at + "4:1",
			grammar:   "xs:selector",
			wantTail:  noPosition,
		},
		{
			name:      "under a field",
			lines:     []string{elem, `<xs:unique name="u">`, `<xs:selector xpath="a"/>`, `<xs:field xpath="@x">`, o, `</xs:field>`, `</xs:unique>`, elemEnd},
			wantChild: foreign + at + "6:1",
			wantOwner: "<field>" + at + "5:1",
			grammar:   "xs:field",
			wantTail:  noPosition,
		},
		{
			name:      "under a list with itemType",
			lines:     []string{`<xs:simpleType name="S">`, `<xs:list itemType="xs:string">`, o, `</xs:list>`, `</xs:simpleType>`},
			wantChild: foreign + at + "4:1",
			wantOwner: "<list>" + at + "3:1",
			grammar:   "xs:list",
			wantTail:  noPosition,
		},
		{
			name:      "after the any of an openContent",
			lines:     []string{ct, `<xs:openContent>`, `<xs:any namespace="##other"/>`, o, `</xs:openContent>`, `<xs:sequence/>`, ctEnd},
			wantChild: foreign + at + "5:1",
			wantOwner: "<openContent>" + at + "3:1",
			grammar:   "xs:openContent",
			wantTail:  noPosition,
		},
		{
			name:      "after the any of a defaultOpenContent",
			lines:     []string{`<xs:defaultOpenContent>`, `<xs:any namespace="##other"/>`, o, `</xs:defaultOpenContent>`, `<xs:complexType name="C"><xs:sequence/></xs:complexType>`},
			wantChild: foreign + at + "4:1",
			wantOwner: "<defaultOpenContent>" + at + "2:1",
			grammar:   "xs:defaultOpenContent",
			wantTail:  noPosition,
		},
		{
			// ##other excludes ·absent·: the facet position's wildcard arm does not
			// hold a child in no namespace.
			name:      "no-namespace child among the facets of a simpleType restriction",
			lines:     []string{`<xs:simpleType name="S">`, `<xs:restriction base="xs:string">`, `<xs:maxLength value="3"/>`, `<hint/>`, `</xs:restriction>`, `</xs:simpleType>`},
			wantChild: "<hint>" + at + "5:1",
			wantOwner: "<restriction>" + at + "3:1",
			grammar:   "xs:restriction",
			wantTail:  "which admits no element in no namespace in any position",
		},
		{
			name:      "no-namespace child among the facets of a simpleContent restriction",
			lines:     []string{simpleBase, `<xs:complexType name="D">`, `<xs:simpleContent>`, `<xs:restriction base="tns:B">`, `<xs:length value="5"/>`, `<hint/>`, `</xs:restriction>`, `</xs:simpleContent>`, ctEnd},
			wantChild: "<hint>" + at + "7:1",
			wantOwner: "<restriction>" + at + "5:1",
			grammar:   "xs:simpleRestrictionType",
			wantTail:  "which admits no element in no namespace in any position",
		},
		{
			// The wildcard arm is the facet position's own, so the attribute tail
			// closes it behind it.
			name:      "foreign child after the assert of a simpleContent restriction",
			lines:     []string{simpleBase, `<xs:complexType name="D">`, `<xs:simpleContent>`, `<xs:restriction base="tns:B">`, `<xs:assert test="true()"/>`, o, `</xs:restriction>`, `</xs:simpleContent>`, ctEnd},
			wantChild: foreign + at + "7:1",
			wantOwner: "<restriction>" + at + "5:1",
			grammar:   "xs:simpleRestrictionType",
			wantTail:  "and a " + foreign + " may not follow the children written before it here",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, s4sTopLevelDoc(tc.lines...))
			if err == nil {
				t.Fatal("Produce accepted a child its s4s content model does not admit")
			}
			if rule, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
			}
			if !strings.HasPrefix(err.Error(), "parser: "+tc.wantChild+" ") {
				t.Errorf("error = %v, want it to open with the offending %s", err, tc.wantChild)
			}
			if !strings.Contains(err.Error(), tc.wantOwner) {
				t.Errorf("error = %v, want it to name the owning %s", err, tc.wantOwner)
			}
			if !strings.Contains(err.Error(), tc.grammar+"'s content model") {
				t.Errorf("error = %v, want it to quote %s's content model", err, tc.grammar)
			}
			if !strings.HasSuffix(err.Error(), tc.wantTail) {
				t.Errorf("error = %v, want it to close %q", err, tc.wantTail)
			}
		})
	}
}

// TestProduceLateAnnotationAfterForeignChildRejected pins rejectLateAnnotation's
// half of #1982: a child outside the XSD namespace counts as a child written
// before an <annotation>, which "annotation?" admits only first, and the
// diagnostic names it by its expanded name.
func TestProduceLateAnnotationAfterForeignChildRejected(t *testing.T) {
	const at = " at " + produceURI + ":"
	_, err := produce(t, s4sTopLevelDoc(
		`<xs:complexType name="C">`,
		`<xs:sequence>`,
		`<o:hint xmlns:o="urn:other"/>`,
		`<xs:annotation/>`,
		`</xs:sequence>`,
		`</xs:complexType>`,
	))
	if err == nil {
		t.Fatal("Produce accepted an <annotation> written after a foreign child of a <sequence>")
	}
	if rule, ok := xsderr.RuleOf(err); ok {
		t.Errorf("error = %v, charged %s; want a plain grammar fault carrying no rule ID", err, rule)
	}
	want := "parser: <annotation>" + at + "5:1 follows the <{urn:other}hint>" + at + "4:1 among the children of the <sequence>" + at + "3:1: "
	if !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error = %v, want it to open %q", err, want)
	}
}
