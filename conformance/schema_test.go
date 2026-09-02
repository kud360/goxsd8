package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser"
)

// schemaDoc builds an in-memory schema document from body children wrapped in a
// <schema> with the xs prefix bound, mirroring parser/produce_test.go's wrap.
func schemaDoc(t *testing.T, body string) *parser.Document {
	t.Helper()
	src := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` + body + `</xs:schema>`
	d, err := parser.ReadDocument("mem://schema.xsd", strings.NewReader(src))
	if err != nil {
		t.Fatalf("ReadDocument(%q): %v", body, err)
	}
	return d
}

// TestSchemaShapeDecidableAccepts proves schemaShapeDecidable admits exactly the
// producer's decidable subset: type=-form elements, bare-or-typed attributes,
// simpleTypes in every §3.16.2.1 shape and in none of them (including a recursed
// anonymous inline base), and annotations — the shapes parser.Produce either
// genuinely decides or genuinely rejects.
func TestSchemaShapeDecidableAccepts(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"typed element", `<xs:element name="e" type="xs:string"/>`},
		{"bare attribute (defaults to anySimpleType)", `<xs:attribute name="a"/>`},
		{"typed attribute", `<xs:attribute name="a" type="xs:string"/>`},
		{"restriction simpleType with pattern", `<xs:simpleType name="T"><xs:restriction base="xs:string"><xs:pattern value="1|2"/></xs:restriction></xs:simpleType>`},
		{"annotation", `<xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation>`},
		{"anonymous inline base (recursed)", `<xs:simpleType name="N"><xs:restriction><xs:simpleType><xs:restriction base="xs:string"><xs:pattern value="1*"/></xs:restriction></xs:simpleType><xs:minLength value="1"/></xs:restriction></xs:simpleType>`},
		{"bare element (defaults to anyType, now seeded)", `<xs:element name="e"/>`},
		{"empty complexType", `<xs:complexType name="CT"/>`},
		{"complexType with sequence + local element + attribute", `<xs:complexType name="CT"><xs:sequence><xs:element name="a" type="xs:string"/><xs:any/></xs:sequence><xs:attribute name="at" type="xs:int"/></xs:complexType>`},
		{"complexType with choice + anyAttribute", `<xs:complexType name="CT"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice><xs:anyAttribute namespace="##other"/></xs:complexType>`},
		{"complexContent restriction", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="CT"><xs:complexContent><xs:restriction base="B"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
		// #336: both <extension> forms are produced (#228) and judged by
		// cos-ct-extends (§3.4.6.2, #264) over the §3.4.2 base folds #401/#265/#346
		// completed, so a complex type using either is decided rather than
		// declined — a NAMED one here, and since #1126 an inline anonymous one on
		// the same terms, through the same predicate.
		{"complexContent extension", `<xs:complexType name="T"><xs:complexContent><xs:extension base="xs:anyType"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`},
		{"complexContent extension with attributes and a group ref", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:complexContent><xs:extension base="B"><xs:sequence><xs:group ref="g"/></xs:sequence><xs:attribute name="a" type="xs:string"/><xs:anyAttribute namespace="##other"/></xs:extension></xs:complexContent></xs:complexType>`},
		{"simpleContent extension", `<xs:complexType name="T"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`},
		{"simpleContent extension with attributes and an assert", `<xs:complexType name="T"><xs:simpleContent><xs:extension base="xs:string"><xs:attribute name="a" type="xs:int"/><xs:attributeGroup ref="ag"/><xs:anyAttribute namespace="##other"/><xs:assert test="true()"/></xs:extension></xs:simpleContent></xs:complexType>`},
		// #909: <simpleContent> <restriction> builds §3.4.2.2 cases 1-2, so every
		// item of xs:simpleRestrictionType's content model is mapped — the facet
		// children and the optional inline <simpleType> into the synthesized
		// anonymous simple type, the rest into the CTD's own properties.
		{"simpleContent restriction with facets", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:maxLength value="4"/><xs:enumeration value="ab"/><xs:pattern value="a*"/><xs:assertion test="true()"/></xs:restriction></xs:simpleContent></xs:complexType>`},
		{"simpleContent restriction with an inline simpleType base", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction></xs:simpleType></xs:restriction></xs:simpleContent></xs:complexType>`},
		{"simpleContent restriction with attributes and an assert", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:maxLength value="4"/><xs:attribute name="a" type="xs:int"/><xs:attributeGroup ref="ag"/><xs:anyAttribute namespace="##other"/><xs:assert test="true()"/></xs:restriction></xs:simpleContent></xs:complexType>`},
		// The SINGULAR <assertion> is the facet element xs:simpleRestrictionType's
		// facet choice admits (§4.3.13, xmlschema11-1.md:1692) and restrictionFacets
		// folds; the plural names the FACET, not any element, and declines below.
		{"simpleContent restriction with an assertion facet", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:assertion test="true()"/></xs:restriction></xs:simpleContent></xs:complexType>`},
		// #868: neither alternant under <simpleContent>/<complexContent> is a
		// grammar fault (§3.4.2.2 and §3.4.2.3 each require one), and the producer
		// rejects it as one — a genuine verdict, so no decline. The <simpleContent>
		// half is NOT the unproduced <restriction> limitation that shares its arm,
		// which still declines below.
		{"complexType with a complexContent carrying neither alternant", `<xs:complexType name="T"><xs:complexContent/></xs:complexType>`},
		{"complexType with a simpleContent carrying neither alternant", `<xs:complexType name="T"><xs:simpleContent/></xs:complexType>`},
		// annotB030's shape, which the repeated-<annotation> guard (#836) and this
		// fault each reject on their own.
		{"complexType with a simpleContent holding only annotations", `<xs:complexType name="T"><xs:simpleContent><xs:annotation/><xs:annotation/></xs:simpleContent></xs:complexType>`},
		{"top-level group definition (§3.7.2)", `<xs:group name="g"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`},
		{"top-level group with choice + any", `<xs:group name="g"><xs:choice><xs:element name="a" type="xs:string"/><xs:any/></xs:choice></xs:group>`},
		{"top-level attributeGroup definition (§3.6.2)", `<xs:attributeGroup name="ag"><xs:attribute name="a" type="xs:string"/><xs:anyAttribute namespace="##other"/></xs:attributeGroup>`},
		{"attributeGroup referencing another group", `<xs:attributeGroup name="ag"><xs:attributeGroup ref="base"/><xs:attribute name="a"/></xs:attributeGroup>`},
		{"complexType with group ref content", `<xs:complexType name="T"><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType>`},
		{"complexType with top-level group ref as content", `<xs:complexType name="T"><xs:group ref="g"/></xs:complexType>`},
		{"complexType with attributeGroup ref", `<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="ag"/></xs:complexType>`},
		{"all decidable kinds together", `<xs:element name="e" type="T"/><xs:attribute name="a"/><xs:simpleType name="T"><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType>`},
		{"top-level notation (§3.14.2)", `<xs:notation name="n" public="-//x//y" system="x.dtd"/>`},
		// #286/#505: <redefine> is admitted for all four redefinable kinds, each
		// gated by the very predicate its top-level form is gated by.
		{"redefine of a simpleType", `<xs:redefine schemaLocation="b.xsd"><xs:simpleType name="T"><xs:restriction base="tns:T"><xs:maxLength value="3"/></xs:restriction></xs:simpleType></xs:redefine>`},
		{"redefine of a complexType", `<xs:redefine schemaLocation="b.xsd"><xs:complexType name="T"><xs:complexContent><xs:extension base="tns:T"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType></xs:redefine>`},
		{"redefine of a group", `<xs:redefine schemaLocation="b.xsd"><xs:group name="g"><xs:sequence><xs:group ref="tns:g"/><xs:element name="a" type="xs:string"/></xs:sequence></xs:group></xs:redefine>`},
		{"redefine of an attributeGroup", `<xs:redefine schemaLocation="b.xsd"><xs:attributeGroup name="ag"><xs:attributeGroup ref="tns:ag"/><xs:attribute name="a" type="xs:string"/></xs:attributeGroup></xs:redefine>`},
		{"empty redefine (a plain include)", `<xs:redefine schemaLocation="b.xsd"><xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation></xs:redefine>`},
		{"element with name= identity constraint", `<xs:element name="e"><xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key></xs:element>`},
		// #229: an inline anonymous <simpleType> on a LOCAL element or attribute is
		// produced (§3.3.2.1 dcl.elt.common clause 1, §3.2.2.2 dcl.att.local), so the
		// gates admit it — provided the inline type's own shape is produced too.
		// #442 produced the same tier 1 on the GLOBAL element path, which
		// §3.3.2.2 dcl.elt.global leaves untouched, and #733 the tier-1 §3.2.2.1
		// dcl.att.global states in dcl.att.local's own words, so both top-level
		// forms are admitted on the same proviso and their both-present-with-type=
		// cases become genuine src-element clause 3 / src-attribute clause 4
		// verdicts.
		{"top-level element with inline anonymous simpleType", `<xs:element name="e"><xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType></xs:element>`},
		{"element with both type= and an inline simpleType (src-element clause 3)", `<xs:element name="e" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`},
		{"top-level attribute with inline anonymous simpleType", `<xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:int"><xs:minInclusive value="1"/></xs:restriction></xs:simpleType></xs:attribute>`},
		{"attribute with both type= and an inline simpleType (src-attribute clause 4)", `<xs:attribute name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`},
		{"substitution group member whose head is typed by an inline simpleType", `<xs:element name="head"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element><xs:element name="member" substitutionGroup="head"/>`},
		{"local element with inline anonymous simpleType", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:simpleType><xs:restriction base="xs:string"><xs:maxLength value="3"/></xs:restriction></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		{"local attribute with inline anonymous simpleType", `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:attribute></xs:complexType>`},
		{"attributeGroup attribute with inline anonymous simpleType", `<xs:attributeGroup name="ag"><xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute></xs:attributeGroup>`},
		// Both type= and an inline <simpleType> on a LOCAL declaration is now a
		// genuine src-element/src-attribute rejection, not a limitation, so the case
		// is decided rather than declined.
		{"local element with both type= and an inline simpleType (src-element clause 3)", `<xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		{"local attribute with both type= and an inline simpleType (src-attribute clause 4)", `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute></xs:complexType>`},
		// #340 produced §3.3.2.1 dcl.elt.common clause 1's inline <complexType>
		// child on BOTH the global and the local path, so the form is decided
		// rather than declined at every nesting depth — and its
		// both-present-with-type= case becomes a genuine src-element clause 3
		// verdict. #1126 took the last narrowing off that admission: the inline
		// anonymous type goes through complexTypeDecidable, so the EXPLICIT-content
		// forms below are admitted exactly where their named counterparts above
		// are, and declined exactly where those are.
		{"element with an inline implicit-content complexType", `<xs:element name="e"><xs:complexType/></xs:element>`},
		{"element with an inline complexType holding attributes and content", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence><xs:attribute name="x" type="xs:string"/></xs:complexType></xs:element>`},
		{"local element with an inline complexType", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:complexType/></xs:element></xs:sequence></xs:complexType>`},
		{"inline complexType nested inside an inline complexType", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a"><xs:complexType><xs:sequence/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`},
		{"element with an inline complexType using complexContent", `<xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="xs:anyType"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType></xs:element>`},
		{"element with an inline complexType using simpleContent", `<xs:element name="e"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType></xs:element>`},
		{"local element with an inline complexType using complexContent", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:complexType><xs:complexContent><xs:restriction base="xs:anyType"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType></xs:element></xs:sequence></xs:complexType>`},
		{"inline complexType nesting an explicit-content inline complexType", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`},
		{"element with both type= and an inline complexType (src-element clause 3)", `<xs:element name="e" type="xs:string"><xs:complexType/></xs:element>`},
		{"local element with both type= and an inline complexType (src-element clause 3)", `<xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"><xs:complexType/></xs:element></xs:sequence></xs:complexType>`},
		{"override child with an inline complexType", `<xs:override schemaLocation="b.xsd"><xs:element name="e"><xs:complexType/></xs:element></xs:override>`},
		{"local element with name= identity constraint", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:unique name="u"><xs:selector xpath="b"/><xs:field xpath="@x"/></xs:unique></xs:element></xs:sequence></xs:complexType>`},
		// #240 produced the ref= form too — it maps to the definition it names
		// (§3.11.2), so src-identity-constraint clauses 1/4/5 and src-resolve on it
		// are genuine verdicts, not limitations.
		{"element with ref= identity constraint", `<xs:element name="e"><xs:key ref="k"/></xs:element><xs:element name="d"><xs:key name="k"><xs:selector xpath="b"/><xs:field xpath="@x"/></xs:key></xs:element>`},
		{"local element with ref= identity constraint", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:keyref ref="kr"/></xs:element></xs:sequence></xs:complexType>`},
		{"complexType with assert", `<xs:complexType name="T"><xs:sequence/><xs:assert test="true()"/></xs:complexType>`},
		{"complexContent restriction with assert", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:complexContent><xs:restriction base="B"><xs:sequence/><xs:assert test="true()"/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"restriction with assertion facet", `<xs:simpleType name="A"><xs:restriction base="xs:int"><xs:assertion test="$value > 0"/></xs:restriction></xs:simpleType>`},
		// #740 folds the <enumeration> children of one <restriction> into a single
		// facet (§4.3.5.2 xr-enumeration), so the shape is admitted wherever a
		// simple type can sit, one child or several.
		{"restriction with one enumeration facet", `<xs:simpleType name="E"><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType>`},
		{"restriction with several enumeration facets", `<xs:simpleType name="E"><xs:restriction base="xs:string"><xs:enumeration value="a"/><xs:enumeration value="b"/></xs:restriction></xs:simpleType>`},
		{"local element with an inline enumeration-bearing simpleType", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:simpleType><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		// #447 produces the <list> alternative in both its forms, so both are
		// admitted, at every position an anonymous simple type can sit in — and an
		// inline item child is recursed into, which since #786 can no longer
		// decline: see the alternative-less block below.
		{"list-variety simpleType by itemType=", `<xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`},
		{"list-variety simpleType with an inline item", `<xs:simpleType name="L"><xs:list><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:list></xs:simpleType>`},
		{"local element with an inline list-variety simpleType", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:simpleType><xs:list itemType="xs:string"/></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		{"redefine child that is a list-variety simpleType", `<xs:redefine schemaLocation="b.xsd"><xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType></xs:redefine>`},
		// #738 produces the <union> alternative in all three of its forms —
		// memberTypes= alone, inline <simpleType> children alone, and both at once
		// — so each is admitted wherever a simple type can sit, and an inline
		// member child is recursed into exactly as a list's item child is.
		{"union-variety simpleType by memberTypes=", `<xs:simpleType name="U"><xs:union memberTypes="xs:string xs:int"/></xs:simpleType>`},
		{"union-variety simpleType with inline members", `<xs:simpleType name="U"><xs:union><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:union></xs:simpleType>`},
		{"union-variety simpleType with both member sources", `<xs:simpleType name="U"><xs:union memberTypes="xs:string"><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:union></xs:simpleType>`},
		{"local element with an inline union-variety simpleType", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:simpleType><xs:union memberTypes="xs:string"/></xs:simpleType></xs:element></xs:sequence></xs:complexType>`},
		{"redefine child that is a union-variety simpleType", `<xs:redefine schemaLocation="b.xsd"><xs:simpleType name="U"><xs:union memberTypes="xs:string"/></xs:simpleType></xs:redefine>`},
		// #786 admits the last shape simpleTypeDecidable declined: a <simpleType>
		// naming NONE of §3.16.2.1's three alternatives. simpleTypeBody rejects
		// that document unconditionally — §5.1's first bullet, an unfilled
		// alternative position — so the verdict is the producer's own and this
		// gate fabricates nothing. Every row below was a DECLINE before that
		// widening, one per position an anonymous or named simple type can sit in.
		{"simpleType with only an annotation", `<xs:simpleType name="T"><xs:annotation/></xs:simpleType>`},
		{"simpleType with two annotations", `<xs:simpleType name="T"><xs:annotation/><xs:annotation/></xs:simpleType>`},
		{"simpleType whose only child is a particle", `<xs:simpleType name="T"><xs:group ref="g"/></xs:simpleType>`},
		{"simpleType whose only child is an identity constraint", `<xs:simpleType name="T"><xs:unique name="u"><xs:selector xpath="b"/><xs:field xpath="@x"/></xs:unique></xs:simpleType>`},
		{"simpleType with no children at all", `<xs:simpleType name="T"/>`},
		{"restriction whose inline base names none of the three", `<xs:simpleType name="N"><xs:restriction base="xs:string"><xs:simpleType><xs:length value="5"/></xs:simpleType></xs:restriction></xs:simpleType>`},
		{"list whose inline item names none of the three", `<xs:simpleType name="L"><xs:list><xs:simpleType/></xs:list></xs:simpleType>`},
		{"union whose inline member names none of the three", `<xs:simpleType name="U"><xs:union memberTypes="xs:string"><xs:simpleType/></xs:union></xs:simpleType>`},
		{"top-level element with an inline simpleType naming none of the three", `<xs:element name="e"><xs:simpleType/></xs:element>`},
		{"top-level attribute with an inline simpleType naming none of the three", `<xs:attribute name="a"><xs:simpleType/></xs:attribute>`},
		{"local element with an inline simpleType naming none of the three", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:simpleType/></xs:element></xs:sequence></xs:complexType>`},
		{"local attribute with an inline simpleType naming none of the three", `<xs:complexType name="T"><xs:sequence/><xs:attribute name="a"><xs:simpleType/></xs:attribute></xs:complexType>`},
		{"attributeGroup attribute with an inline simpleType naming none of the three", `<xs:attributeGroup name="ag"><xs:attribute name="a"><xs:simpleType/></xs:attribute></xs:attributeGroup>`},
		{"alternative's inline simpleType naming none of the three", `<xs:element name="e" type="xs:string"><xs:alternative test="@x"><xs:simpleType/></xs:alternative></xs:element>`},
		{"simpleContent restriction whose inline simpleType names none of the three", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:simpleType/></xs:restriction></xs:simpleContent></xs:complexType>`},
		{"redefine child that is a simpleType naming none of the three", `<xs:redefine schemaLocation="b.xsd"><xs:simpleType name="U"/></xs:redefine>`},
		{"override child that is a simpleType naming none of the three", `<xs:override schemaLocation="b.xsd"><xs:simpleType name="U"/></xs:override>`},
		{"alternative-less simpleType beside a decidable kind", `<xs:element name="e" type="xs:string"/><xs:simpleType name="U"/>`},
		// #242: <include> contributes no component of its own, so it is admitted
		// here; the decidability of what it points at is the closure gate's job
		// (closureDecidable over the assembly's report), not this allowlist's.
		{"top-level include", `<xs:include schemaLocation="lib.xsd"/>`},
		{"include beside decidable kinds", `<xs:include schemaLocation="lib.xsd"/><xs:element name="e" type="xs:string"/>`},
		// #182: <import> likewise contributes no component of its own. The stricter
		// no-D2 rule that governs it lives in the Unfollowed conjunction
		// execSchemaCase applies, not in this shape allowlist, so a bare <import>
		// is admitted HERE and declined THERE.
		{"top-level import", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/>`},
		{"import beside include and decidable kinds", `<xs:import namespace="urn:b" schemaLocation="b.xsd"/><xs:include schemaLocation="lib.xsd"/><xs:element name="e" type="xs:string"/>`},
		// #183: an <override> is admitted when each of its children is a decidable
		// source declaration, since §F.2 clause 1 makes those children top-level
		// declarations of the OVERRIDDEN document. What it points at is the closure
		// gate's job (closureDecidable over the assembly's report), not this
		// allowlist's.
		{"override with decidable children", `<xs:override schemaLocation="b.xsd"><xs:element name="e" type="xs:string"/><xs:simpleType name="T"><xs:restriction base="xs:string"/></xs:simpleType></xs:override>`},
		{"override with only an annotation", `<xs:override schemaLocation="b.xsd"><xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation></xs:override>`},
		{"empty override", `<xs:override schemaLocation="b.xsd"/>`},
		{"override beside decidable kinds", `<xs:override schemaLocation="b.xsd"><xs:notation name="n" public="p"/></xs:override><xs:element name="e" type="xs:string"/>`},
		// #230: <openContent> maps to {open content} (§3.4.2.3.3 clauses 5-6) in
		// every position the schema for schema documents allows one, and the
		// schema-level <defaultOpenContent> it falls back to is read rather than
		// skipped — so all four shapes are decided rather than declined.
		{"complexType with openContent", `<xs:complexType name="T"><xs:openContent mode="interleave"><xs:any/></xs:openContent><xs:sequence/></xs:complexType>`},
		{"complexType with openContent mode=none", `<xs:complexType name="T"><xs:openContent mode="none"/><xs:sequence/></xs:complexType>`},
		{"complexContent restriction with openContent", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:complexContent><xs:restriction base="B"><xs:openContent mode="suffix"><xs:any/></xs:openContent><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"top-level defaultOpenContent with any", `<xs:defaultOpenContent><xs:any/></xs:defaultOpenContent><xs:complexType name="T"><xs:sequence/></xs:complexType>`},
		// #352 admits the two malformed shapes #230 had to decline as well. Both are
		// now rejected by the producer's pre-produce pass for ANY document that
		// declares a <defaultOpenContent>, so each is a real verdict rather than one
		// contingent on some complex type of the document reaching clause 5.2: the
		// childless form its content model forbids, and a mode outside its
		// (interleave|suffix) enumeration — "none" is legal on a type's OWN
		// <openContent> but not here, and every other token is out of the
		// enumeration outright.
		{"top-level defaultOpenContent with no any child", `<xs:defaultOpenContent/>`},
		{"top-level defaultOpenContent with mode=none", `<xs:defaultOpenContent mode="none"><xs:any/></xs:defaultOpenContent>`},
		{"top-level defaultOpenContent with an out-of-enumeration mode", `<xs:defaultOpenContent mode="bogus"><xs:any/></xs:defaultOpenContent>`},
		// #945: a document holding a MISPLACED <notation> is admitted whatever else
		// it holds, because the producer rejects it before any producer dispatches
		// (rejectS4SFaults) — so the shape of the rest can neither be silently
		// skipped into a vacuous accept nor rejected for a limitation. Every parent
		// below is one the W3C MS-Notations2006-07-15/notatF* family writes, and
		// none of them is <schema> or <override>.
		{"notation inside an <all>", `<xs:complexType name="foo"><xs:all><xs:notation name="jpeg" public="image/jpeg"/></xs:all></xs:complexType>`},
		{"notation inside a <choice>", `<xs:complexType name="foo"><xs:choice><xs:notation name="jpeg" public="image/jpeg"/></xs:choice></xs:complexType>`},
		{"notation inside a <sequence>", `<xs:complexType name="foo"><xs:sequence><xs:notation name="jpeg" public="image/jpeg"/></xs:sequence></xs:complexType>`},
		{"notation directly inside a <complexType>", `<xs:complexType name="foo"><xs:notation name="jpeg" public="image/jpeg"/></xs:complexType>`},
		{"notation inside a complexContent <extension>", `<xs:complexType name="bar"><xs:complexContent><xs:extension><xs:notation name="jpeg" public="image/jpeg"/></xs:extension></xs:complexContent></xs:complexType>`},
		{"notation inside a named <group>'s body", `<xs:group name="foo"><xs:all><xs:notation name="jpeg" public="image/jpeg"/></xs:all></xs:group>`},
		{"notation inside a named <attributeGroup>", `<xs:attributeGroup name="bar"><xs:notation name="jpeg" public="image/jpeg"/></xs:attributeGroup>`},
		// This parent is the one the allowlist would now admit on its own too
		// (#786 admits a <simpleType> naming no alternative), so it pins the
		// notatF067 family's parent coverage and NOT the domination claim, which
		// the last row of this table carries.
		{"notation inside a <simpleType> naming no alternative", `<xs:simpleType name="foo"><xs:notation name="jpeg" public="image/jpeg"/></xs:simpleType>`},
		{"notation inside a <redefine> (§4.2.4 admits none)", `<xs:redefine schemaLocation="foo"><xs:notation name="jpeg" public="image/jpeg"/></xs:redefine>`},
		// The parent is itself no xs:schemaTop arm in these three, so the admission
		// rests on the rejection being genuine and unconditional, never on the
		// parent's own shape becoming decidable.
		{"notation inside a top-level <any>", `<xs:any><xs:notation name="jpeg" public="image/jpeg"/></xs:any>`},
		{"notation inside a top-level <key>'s field", `<xs:key name="foo"><xs:field><xs:notation name="jpeg" public="image/jpeg"/></xs:field></xs:key>`},
		{"notation inside a top-level <keyref>", `<xs:keyref name="r" refer="fullName"><xs:notation name="jpeg" public="image/jpeg"/><xs:selector xpath=".//p"/><xs:field xpath="@first"/></xs:keyref>`},
		// The short-circuit DOMINATES: a shape the allowlist would decline on its
		// own is admitted alongside a misplaced notation, since the parse cannot
		// reach it.
		{"misplaced notation beside an otherwise undecidable group", `<xs:group ref="g"/><xs:complexType name="T"><xs:sequence><xs:notation name="jpeg" public="image/jpeg"/></xs:sequence></xs:complexType>`},
	}
	for _, tc := range cases {
		if !schemaShapeDecidable(schemaDoc(t, tc.body)) {
			t.Errorf("%s: schemaShapeDecidable = false, want true", tc.name)
		}
	}
}

// TestSchemaShapeDecidableDeclines proves schemaShapeDecidable declines every
// shape whose Produce verdict would be a limitation-in-disguise (a false reject or
// an unsupported-form rejection) or a vacuous pass over silently-skipped content.
func TestSchemaShapeDecidableDeclines(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"top-level group without name (reference form is malformed)", `<xs:group ref="g"/>`},
		{"complexType with bare nested group (no ref)", `<xs:complexType name="T"><xs:sequence><xs:group name="inner"><xs:sequence/></xs:group></xs:sequence></xs:complexType>`},
		// #909 admits <simpleContent> <restriction> on the same terms #336 admitted
		// its <extension> sibling: the alternant's own content model is mapped, and
		// the shapes outside it that the producer would drop in silence decline.
		{"simpleContent restriction carrying a particle the producer drops", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:sequence/></xs:restriction></xs:simpleContent></xs:complexType>`},
		{"simpleContent restriction carrying a group ref the producer drops", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:group ref="g"/></xs:restriction></xs:simpleContent></xs:complexType>`},
		// The FACET is spelled "assertions" (§4.3.13) and the element contributing to
		// it <assertion>, so the plural names no element of the facet choice at all:
		// restrictionFacets matches only the singular and facetKindOf excludes the
		// assertions kind, leaving the child dropped in silence. The singular is
		// admitted above.
		{"simpleContent restriction carrying an <assertions> the producer drops", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="B"><xs:assertions test="true()"/></xs:restriction></xs:simpleContent></xs:complexType>`},
		// #336 admits <simpleContent> <extension>, but only in the shape
		// xs:simpleExtensionType allows: §3.4.2.2 builds {content type} from the base
		// alone, so a particle child is DROPPED without an error — a false accept, not
		// a verdict.
		{"simpleContent extension carrying a particle the producer drops", `<xs:complexType name="T"><xs:simpleContent><xs:extension base="xs:string"><xs:sequence/></xs:extension></xs:simpleContent></xs:complexType>`},
		{"simpleContent extension carrying a group ref the producer drops", `<xs:complexType name="T"><xs:simpleContent><xs:extension base="xs:string"><xs:group ref="g"/></xs:extension></xs:simpleContent></xs:complexType>`},
		{"complexContent extension with a bare nested group (no ref)", `<xs:complexType name="T"><xs:complexContent><xs:extension base="xs:anyType"><xs:sequence><xs:group name="inner"><xs:sequence/></xs:group></xs:sequence></xs:extension></xs:complexContent></xs:complexType>`},
		// No inline SIMPLE type declines any more (#786 admitted the last shape
		// that did), so every recursion below carries a complex-type or bare-group
		// specimen instead. The simple-type positions those rows held moved to the
		// decidable table, verdict reversed and shapes unchanged.
		//
		// An inline anonymous <complexType> declines on exactly the terms a NAMED
		// one does, and on no others (#1126): the shape the producer would drop in
		// silence, or one whose own content is outside the produced subset, at
		// every nesting depth. Anonymity itself narrows nothing — the admitted
		// explicit-content forms are in the decidable table above.
		{"inline complexType whose own content is undecidable", `<xs:element name="e"><xs:complexType><xs:sequence><xs:group name="inner"><xs:sequence/></xs:group></xs:sequence></xs:complexType></xs:element>`},
		{"inline complexType using simpleContent that drops a particle", `<xs:element name="e"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"><xs:sequence/></xs:extension></xs:simpleContent></xs:complexType></xs:element>`},
		{"local element's inline complexType using complexContent over a bare group", `<xs:complexType name="T"><xs:sequence><xs:element name="a"><xs:complexType><xs:complexContent><xs:restriction base="xs:anyType"><xs:group name="inner"><xs:sequence/></xs:group></xs:restriction></xs:complexContent></xs:complexType></xs:element></xs:sequence></xs:complexType>`},
		{"inline complexType nesting an inline complexType the gate declines", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"><xs:sequence/></xs:extension></xs:simpleContent></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`},
		{"one decidable + one undecidable child declines whole", `<xs:element name="e" type="xs:string"/><xs:group ref="g"/>`},
		// A redefining <complexType> is gated by complexTypeDecidable like any
		// other, so a shape THAT predicate declines declines the whole case; the
		// self-deriving shape itself is admitted — see the decidable cases.
		{"redefine child that is a complexType the gate declines", `<xs:redefine schemaLocation="b.xsd"><xs:complexType name="T"><xs:sequence><xs:group name="inner"><xs:sequence/></xs:group></xs:sequence></xs:complexType></xs:redefine>`},
		{"redefine child with no name (no pairing possible)", `<xs:redefine schemaLocation="b.xsd"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:redefine>`},
		{"redefine child of an out-of-model kind", `<xs:redefine schemaLocation="b.xsd"><xs:element name="e" type="xs:string"/></xs:redefine>`},
		// An <override> child the parser can only ignore, or one whose own shape is
		// outside the decidable subset, declines the whole case: after substitution
		// it would be an unmapped or undecidable TOP-LEVEL declaration.
		{"override child with no name (matches nothing, silently ignored)", `<xs:override schemaLocation="b.xsd"><xs:element type="xs:string"/></xs:override>`},
		{"override child with an undecidable inline anonymous type", `<xs:override schemaLocation="b.xsd"><xs:element name="e"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"><xs:sequence/></xs:extension></xs:simpleContent></xs:complexType></xs:element></xs:override>`},
		{"override child of an out-of-model kind", `<xs:override schemaLocation="b.xsd"><xs:include schemaLocation="c.xsd"/></xs:override>`},
		{"include beside an undecidable kind still declines", `<xs:include schemaLocation="lib.xsd"/><xs:group ref="g"/>`},
		// #945's short-circuit must not reach an element merely NAMED
		// {XMLSchemaNS}notation inside <appinfo>/<documentation>: that is
		// <xs:any processContents="lax"> content the producer charges no guard on
		// (MS-Notations2006-07-15/notatF009 is suite-VALID), so it licenses nothing
		// and the undecidable sibling still declines the case.
		{"notation inside appinfo does not launder an undecidable sibling", `<xs:annotation><xs:appinfo><xs:notation name="jpeg" public="image/jpeg"/></xs:appinfo></xs:annotation><xs:group ref="g"/>`},
		{"notation inside documentation does not launder an undecidable sibling", `<xs:annotation><xs:documentation><xs:notation name="jpeg" public="image/jpeg"/></xs:documentation></xs:annotation><xs:group ref="g"/>`},
		// A notation in a legal slot is no rejection either: <schema> and
		// <override> are the two content models that reach xs:schemaTop.
		{"top-level notation does not launder an undecidable sibling", `<xs:notation name="jpeg" public="image/jpeg"/><xs:group ref="g"/>`},
		{"override notation does not launder an undecidable sibling", `<xs:override schemaLocation="b.xsd"><xs:notation name="jpeg" public="image/jpeg"/></xs:override><xs:group ref="g"/>`},
	}
	for _, tc := range cases {
		if schemaShapeDecidable(schemaDoc(t, tc.body)) {
			t.Errorf("%s: schemaShapeDecidable = true, want false", tc.name)
		}
	}
}

// TestSchemaExecutorReadErrorDeclines proves a ReadDocument failure is DECLINED
// (Fail) for BOTH polarities, never turned into an observed-invalid verdict: the
// error cannot distinguish a genuine XML well-formedness fault from a parser
// encoding limitation (e.g. well-formed UTF-16 misread as invalid UTF-8), so
// claiming "invalid" would fabricate a verdict for a possibly-well-formed document.
func TestSchemaExecutorReadErrorDeclines(t *testing.T) {
	exec := newSchemaExec()
	dir := t.TempDir()
	malformed := filepath.Join(dir, "malformed.xsd")
	// Unclosed root element: a ReadDocument error (here an XML well-formedness fault).
	if err := os.WriteFile(malformed, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e"`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, ev := range []bool{true, false} {
		if exec(caseSpec{kind: kindSchema, doc: malformed, expect: expectValidity(ev)}).IsPass() {
			t.Errorf("a ReadDocument error must Fail (decline) regardless of expectValid=%v", ev)
		}
	}
}

// TestSchemaExecutorDeclinesNonSchemaRoot proves a well-formed document whose root
// is not <schema> is DECLINED unconditionally (§3.17.2 does not require a <schema>
// root, so it is not decidable for this lane) — Fail for both polarities.
func TestSchemaExecutorDeclinesNonSchemaRoot(t *testing.T) {
	exec := newSchemaExec()
	dir := t.TempDir()
	nonSchema := filepath.Join(dir, "notschema.xml")
	if err := os.WriteFile(nonSchema, []byte(`<root/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, ev := range []bool{true, false} {
		if exec(caseSpec{kind: kindSchema, doc: nonSchema, expect: expectValidity(ev)}).IsPass() {
			t.Errorf("non-schema root must Fail (decline) regardless of expectValid=%v", ev)
		}
	}
}

// TestSchemaExecutorAgreesWithSuite drives the real executor over real suite
// schemaTest fixtures and asserts it agrees with the suite's declared validity for
// the right reason: a decidable valid schema Produces cleanly, a duplicate
// top-level simpleType name is rejected (sch-props-correct §3.17.6.1 clause 2), and
// a wrong expectation yields Fail so the test can actually fail. Skips when the
// submodule is absent.
func TestSchemaExecutorAgreesWithSuite(t *testing.T) {
	skipWithoutSuite(t)
	exec := newSchemaExec()

	sunSType := filepath.Join(suiteRoot, "sunData", "SType")
	cases := []struct {
		rel         string
		expectValid bool
		why         string
	}{
		// Decidable VALID: top-level element type="Test" + restriction-only simpleType.
		{"ST_baseTD/ST_baseTD00101m/ST_baseTD00101m.xsd", true, "element type= + restriction simpleType (pattern)"},
		// Decidable VALID: anonymous inline base reached through the restriction chain.
		{"ST_facets/ST_facets00101m/ST_facets00101m.xsd", true, "restriction over an inline anonymous simpleType base"},
		// Decidable INVALID: two top-level simpleTypes named "Test" collide per kind.
		{"ST_name/ST_name00301m/ST_name00301m.xsd", false, "duplicate top-level simpleType name (sch-props-correct clause 2)"},
	}
	for _, tc := range cases {
		doc := filepath.Join(sunSType, filepath.FromSlash(tc.rel))
		c := caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(tc.expectValid)}
		if got := exec(c); !got.IsPass() {
			t.Errorf("%s (%s): executor disagreed with suite (expectValid=%v)", tc.rel, tc.why, tc.expectValid)
		}
		// A flipped expectation must Fail, proving the executor really decides.
		flipped := caseSpec{kind: kindSchema, doc: doc, expect: expectValidity(!tc.expectValid)}
		if exec(flipped).IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", tc.rel)
		}
	}
}

// TestSchemaExecutorDecidesInlineSimpleTypeSuiteCase drives the real executor over
// typeDef00201m.xsd, whose <element name="root"> carries an inline anonymous
// <simpleType> — §3.3.2.1 dcl.elt.common clause 1 on the GLOBAL path, produced as
// of #442. The suite declares it VALID, so the executor must decide it and agree,
// and must Fail under the flipped expectation rather than passing either way.
// Skips when the submodule is absent.
func TestSchemaExecutorDecidesInlineSimpleTypeSuiteCase(t *testing.T) {
	skipWithoutSuite(t)
	exec := newSchemaExec()
	doc := filepath.Join(suiteRoot, "sunData", "ElemDecl", "typeDef", "typeDef00201m", "typeDef00201m.xsd")
	if !exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValid()}).IsPass() {
		t.Error("a suite-valid top-level <element> with an inline <simpleType> must now be decided and agreed with")
	}
	if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectInvalid()}).IsPass() {
		t.Error("executor must Fail under a flipped expectation (decides for real)")
	}
}

// TestSchemaExecutorDecidesGlobalAttributeInlineSimpleTypeSuiteCase drives the real
// executor over AD_type00102m.xsd, whose top-level <attribute name="number">
// carries an inline anonymous <simpleType> — §3.2.2.1 dcl.att.global's tier 1,
// produced as of #733. The suite declares it VALID, so the executor must decide it
// and agree, and must Fail under the flipped expectation rather than passing
// either way. Skips when the submodule is absent.
func TestSchemaExecutorDecidesGlobalAttributeInlineSimpleTypeSuiteCase(t *testing.T) {
	skipWithoutSuite(t)
	exec := newSchemaExec()
	doc := filepath.Join(suiteRoot, "sunData", "AttrDecl", "AD_type", "AD_type00102m", "AD_type00102m.xsd")
	if !exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValid()}).IsPass() {
		t.Error("a suite-valid top-level <attribute> with an inline <simpleType> must now be decided and agreed with")
	}
	if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectInvalid()}).IsPass() {
		t.Error("executor must Fail under a flipped expectation (decides for real)")
	}
}

// TestSchemaExecutorDecidesMisplacedNotationSuiteCases drives the real executor
// over the MS-Notations fixtures whose only out-of-subset feature is a
// <notation> the grammar admits nowhere (#945). Each is suite-INVALID and the
// producer rejects it unconditionally, so the executor must decide it and agree,
// and must Fail under the flipped expectation rather than passing either way.
//
// The last three rows are the ones whose parent is itself no xs:schemaTop arm,
// and the <redefine> row is the one charged by <redefine>'s own content-model
// guard rather than by rejectS4SFaults: all four are admitted on the rejection
// being genuine, never on the parent's shape. Skips when the submodule is
// absent.
func TestSchemaExecutorDecidesMisplacedNotationSuiteCases(t *testing.T) {
	skipWithoutSuite(t)
	exec := newSchemaExec()
	for _, name := range []string{
		"notatF001", "notatF013", "notatF015", "notatF019", "notatF027",
		"notatF031", "notatF063", "notatF067", "notatF055", "notatF005",
		"notatF029", "notatF039",
	} {
		doc := filepath.Join(suiteRoot, "msData", "notations", name+".xsd")
		if !exec(caseSpec{kind: kindSchema, doc: doc, expect: expectInvalid()}).IsPass() {
			t.Errorf("%s: a suite-invalid misplaced <notation> must be decided and agreed with", name)
		}
		if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValid()}).IsPass() {
			t.Errorf("%s: executor must Fail under a flipped expectation (decides for real)", name)
		}
	}
}

// TestSchemaExecutorDecidesNotationInAppinfoSuiteCase proves the #945
// short-circuit stops where rejectS4SFaults' walk stops: notatF009 writes its
// <notation> inside <appinfo>, whose <xs:any processContents="lax"> content the
// producer charges no guard on, so the case is decided by the ordinary allowlist
// and observes the VALID the suite declares — not by an unconditional rejection
// that never comes. Skips when the submodule is absent.
func TestSchemaExecutorDecidesNotationInAppinfoSuiteCase(t *testing.T) {
	skipWithoutSuite(t)
	exec := newSchemaExec()
	doc := filepath.Join(suiteRoot, "msData", "notations", "notatF009.xsd")
	if !exec(caseSpec{kind: kindSchema, doc: doc, expect: expectValid()}).IsPass() {
		t.Error("a <notation> inside <appinfo> is lax wildcard content: the case stays suite-valid and decided")
	}
	if exec(caseSpec{kind: kindSchema, doc: doc, expect: expectInvalid()}).IsPass() {
		t.Error("executor must Fail under a flipped expectation (decides for real)")
	}
}

// The suite-fixture withholding guard is RETIRED (#786). It stood in turn on
// baseTD00101m1.xsd, disallowedSubst00105m.xsd and stB001.xsd, each repointing
// forced by the widening that made the previous fixture's shape decidable, and
// the last of them stood on a <simpleType> naming none of §3.16.2.1's three
// alternatives. Do not repoint it a fourth time: the guard it enforced is
// TestSchemaExecutorDeclinesUndecidableInclusion's (schema_closure_test.go),
// which drives the same withholding end-to-end over a written tree and so needs
// no suite fixture to stay pointed at a live shape.
