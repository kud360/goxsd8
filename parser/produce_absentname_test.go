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

// TestProduceTopLevelDefinitionProhibitedAttrsRejected pins, END TO END from a
// schema DOCUMENT, the same family on the two DEFINITION kinds: xs:namedGroup
// restricts ref, minOccurs and maxOccurs to use="prohibited"
// (xmlschema11-1.md:5210-:5212) and xs:namedAttributeGroup restricts ref
// (:5511), each alongside the required name (:5209, :5510). The occurrence pair
// is the <group> side's alone — xs:attributeGroup's grammar never pulls in
// xs:occurs, so there is nothing on the <attributeGroup> side to prohibit.
//
// The fault is plain, never a rule verdict, and its footing is NOT the
// declaration kinds': §3.7.3 and src-attribute_group (§3.6.3) both read "None as
// such." in full, there is no src-mgd, and mgd-props-correct/ag-props-correct
// see the property tableau rather than the XML attribute. What reaches this
// document is §5.1's requirement that a schema document be fully valid against
// the Schema for Schema Documents, which carries no numbered ID either — so
// charging any of those rules would be fabricated (STYLE E2).
//
// Every ref row DECLARES the definition its ref names, so no row can pass as a
// dangling reference, and the position assertion pins the offending element's
// own line (STYLE E3, carried in the message text since a plain error holds no
// xsderr.Loc). The rows that write a name of their own are otherwise documents
// that PRODUCE cleanly, so reverting either guard arm fails them on the verdict;
// the four nameless ones are rejected either way, so their assertions are the
// message ones — the diagnostic must name ref and must NOT be topLevelName's "no
// usable name", which is what answers if the guard runs in the wrong order or
// not at all.
func TestProduceTopLevelDefinitionProhibitedAttrsRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts
	// on line 2 of the wrapped document.
	const (
		targetGroup = `<xs:group name="G"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`
		targetAG    = `<xs:attributeGroup name="AG"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`
		groupModel  = `<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>`
	)
	cases := []struct {
		name     string
		body     string
		wantAttr string
		wantLine int
	}{
		{
			name:     `top-level <group ref=>`,
			body:     "\n" + targetGroup + "\n" + `<xs:group ref="tns:G"/>`,
			wantAttr: "ref",
			wantLine: 3,
		},
		{
			name:     `top-level <group ref=> with a name of its own`,
			body:     "\n" + targetGroup + "\n" + `<xs:group name="other" ref="tns:G">` + groupModel + `</xs:group>`,
			wantAttr: "ref",
			wantLine: 3,
		},
		{
			name:     `top-level <group ref="">`,
			body:     "\n" + `<xs:group ref=""/>`,
			wantAttr: "ref",
			wantLine: 2,
		},
		{
			name:     `top-level <group minOccurs=>`,
			body:     "\n" + `<xs:group name="G" minOccurs="0">` + groupModel + `</xs:group>`,
			wantAttr: "minOccurs",
			wantLine: 2,
		},
		{
			name:     `top-level <group maxOccurs=>`,
			body:     "\n" + `<xs:group name="G" maxOccurs="unbounded">` + groupModel + `</xs:group>`,
			wantAttr: "maxOccurs",
			wantLine: 2,
		},
		{
			// Two prohibited attributes at once, written in the REVERSE of the
			// grammar's declaration order: the check order is the grammar's, not the
			// document's, so the reported attribute is stable (STYLE D2).
			name:     `top-level <group maxOccurs= minOccurs=>`,
			body:     "\n" + `<xs:group name="G" maxOccurs="unbounded" minOccurs="0">` + groupModel + `</xs:group>`,
			wantAttr: "minOccurs",
			wantLine: 2,
		},
		{
			name:     `top-level <attributeGroup ref=>`,
			body:     "\n" + targetAG + "\n" + `<xs:attributeGroup ref="tns:AG"/>`,
			wantAttr: "ref",
			wantLine: 3,
		},
		{
			name: `top-level <attributeGroup ref=> with a name of its own`,
			body: "\n" + targetAG + "\n" +
				`<xs:attributeGroup name="other" ref="tns:AG"><xs:attribute name="b" type="xs:string"/></xs:attributeGroup>`,
			wantAttr: "ref",
			wantLine: 3,
		},
		{
			name:     `top-level <attributeGroup ref="">`,
			body:     "\n" + `<xs:attributeGroup ref=""/>`,
			wantAttr: "ref",
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

// redefiningLib is the redefined document the two tables below pair with: it
// declares every name they redefine, so no row can pass as a src-expredef
// pairing miss — except the one row written to have one.
const redefiningLib = `<xs:group name="G"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>` +
	`<xs:attributeGroup name="AG"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>` +
	`<xs:complexType name="CT"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`

// redefining wraps one <redefine> child as the whole of main.xsd, putting the
// child on line 3 for the position assertions (STYLE E3).
func redefining(child string) map[string]string {
	return map[string]string{
		"main.xsd": wrap("urn:a", "\n"+`<xs:redefine schemaLocation="lib.xsd">`+"\n"+child+"\n"+`</xs:redefine>`),
		"lib.xsd":  wrap("urn:a", redefiningLib),
	}
}

// TestParseRedefiningDefinitionProhibitedAttrsRejected is the REDEFINE half of
// the family the table above covers at the top level, and the same grammar types
// govern both: §4.2.4's content model reaches the GLOBAL <group> and
// <attributeGroup> element declarations through xs:redefinable
// (xmlschema11-1.md:4465, :5331, :5528), so a redefining <group> is an
// xs:namedGroup — ref, minOccurs and maxOccurs prohibited (:5210-:5212) — and a
// redefining <attributeGroup> an xs:namedAttributeGroup, prohibiting ref alone
// (:5511). The footing is §5.1's schema-for-schema-documents validity, which
// carries no numbered rule ID, so every row asserts a PLAIN error: charging
// src-redefine, src-expredef, mgd-props-correct or ag-props-correct here would
// be fabricated (STYLE E2).
//
// It parses two documents rather than producing one because a redefinition needs
// the redefined document — [Produce] follows no ·inter-schema-document
// reference·, so its <redefine> children map to nothing at all and the tables
// above cannot reach this half.
//
// Two ordering claims are pinned, and each is why the guard sits in
// newRedefineSet rather than in produceRedefinition:
//
//   - the nameless rows, against the sibling grammar fault just below the guard
//     — a <group ref="tns:G"/> writes no name, so without the guard it is
//     reported for the name the ref displaced, which is the suite's own shape;
//   - the LAST row, against src-expredef — its redefining <group> names nothing
//     lib.xsd declares, so the pairing miss is live over the same child, and the
//     ref must still be what is reported.
//
// Every named row but the last is a document that would otherwise PARSE, so
// reverting the guard fails it on the verdict. The nameless rows and the last
// one are rejected either way, so their assertions are the message ones: the
// diagnostic names the attribute and the form, and is neither newRedefineSet's
// absent-name wording nor a rule verdict.
func TestParseRedefiningDefinitionProhibitedAttrsRejected(t *testing.T) {
	const (
		groupBody = `<xs:sequence><xs:group ref="tns:G"/><xs:element name="b" type="xs:string"/></xs:sequence>`
		agBody    = `<xs:attributeGroup ref="tns:AG"/><xs:attribute name="b" type="xs:string"/>`
	)
	// A slice, not a map: subtest order is output (STYLE D2).
	cases := []struct {
		name     string
		child    string
		wantAttr string
	}{
		{
			name:     `redefining <group ref=>`,
			child:    `<xs:group ref="tns:G"/>`,
			wantAttr: "ref",
		},
		{
			name:     `redefining <group ref=> with a name of its own`,
			child:    `<xs:group name="G" ref="tns:G">` + groupBody + `</xs:group>`,
			wantAttr: "ref",
		},
		{
			name:     `redefining <group minOccurs=>`,
			child:    `<xs:group name="G" minOccurs="0">` + groupBody + `</xs:group>`,
			wantAttr: "minOccurs",
		},
		{
			name:     `redefining <group maxOccurs=>`,
			child:    `<xs:group name="G" maxOccurs="100">` + groupBody + `</xs:group>`,
			wantAttr: "maxOccurs",
		},
		{
			// Two prohibited attributes at once, written in the REVERSE of the
			// grammar's declaration order: the check order is the grammar's, not the
			// document's, so the reported attribute is stable (STYLE D2).
			name:     `redefining <group maxOccurs= minOccurs=>`,
			child:    `<xs:group name="G" maxOccurs="100" minOccurs="0">` + groupBody + `</xs:group>`,
			wantAttr: "minOccurs",
		},
		{
			name:     `redefining <attributeGroup ref=>`,
			child:    `<xs:attributeGroup ref="tns:AG"/>`,
			wantAttr: "ref",
		},
		{
			name:     `redefining <attributeGroup ref=> with a name of its own`,
			child:    `<xs:attributeGroup name="AG" ref="tns:AG">` + agBody + `</xs:attributeGroup>`,
			wantAttr: "ref",
		},
		{
			name:     `redefining <group ref=> naming no original`,
			child:    `<xs:group name="Absent" ref="tns:G"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`,
			wantAttr: "ref",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", redefining(tc.child))
			if err == nil {
				t.Fatalf("Parse succeeded, want a grammar fault for the prohibited %s", tc.wantAttr)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if want := fmt.Sprintf("carries a %s attribute", tc.wantAttr); !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to name %s as the prohibited attribute", err, tc.wantAttr)
			}
			if !strings.Contains(err.Error(), "prohibits on the redefining form") {
				t.Fatalf("error = %v, want it to name the form it judged, which here is the redefining one", err)
			}
			if strings.Contains(err.Error(), "has no name attribute") {
				t.Fatalf("error = %v, want the prohibited-attribute fault rather than the absent-name one it causes", err)
			}
			if !strings.Contains(err.Error(), "main.xsd:3:") {
				t.Fatalf("error = %v, want it positioned at the redefining child's own line main.xsd:3 (E3)", err)
			}
		})
	}
}

// TestParseRedefiningProhibitedAttrsLegalElsewhere is the reverse hazard of the
// table above: the guard reads the <redefine> element's OWN children and nothing
// below them, and it claims only what each kind's grammar declares.
//
// The parsing rows are a well-formed redefinition of each kind — the control
// that fails if the guard ever fires on a redefining declaration carrying none
// of the prohibited attributes — plus a local <group ref> with both occurrence
// attributes DESCENDING from a redefining <complexType>, where xs:groupRef makes
// ref required and xs:occurs makes minOccurs/maxOccurs legal, so a guard that
// walked descendants would reject a legal schema.
//
// The last rows pin what the guard must NOT CLAIM: xs:attributeGroup's grammar
// never pulls in xs:occurs, so xs:namedAttributeGroup has no occurrence pair to
// prohibit and minOccurs= on a redefining <attributeGroup> is ABSENT from that
// grammar, a different fault this guard may not name. They are asserted on the
// message rather than on acceptance, since a later §A attribute-set check may
// legitimately reject those documents for that other, true reason.
func TestParseRedefiningProhibitedAttrsLegalElsewhere(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name  string
		child string
	}{
		{
			name:  `redefining <group>`,
			child: `<xs:group name="G"><xs:sequence><xs:group ref="tns:G"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`,
		},
		{
			name:  `redefining <attributeGroup>`,
			child: `<xs:attributeGroup name="AG"><xs:attributeGroup ref="tns:AG"/><xs:attribute name="b" type="xs:string"/></xs:attributeGroup>`,
		},
		{
			name: `local <group ref= minOccurs= maxOccurs=> under a redefining <complexType>`,
			child: `<xs:complexType name="CT"><xs:complexContent><xs:extension base="tns:CT">` +
				`<xs:sequence><xs:group ref="tns:G" minOccurs="0" maxOccurs="unbounded"/></xs:sequence>` +
				`</xs:extension></xs:complexContent></xs:complexType>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMap(t, "main.xsd", redefining(tc.child)); err != nil {
				t.Fatalf("Parse rejected a redefinition the grammar admits: %v", err)
			}
		})
	}
	for _, tc := range []struct {
		name  string
		child string
	}{
		{
			name:  `redefining <attributeGroup minOccurs=>`,
			child: `<xs:attributeGroup name="AG" minOccurs="0"><xs:attributeGroup ref="tns:AG"/></xs:attributeGroup>`,
		},
		{
			name:  `redefining <attributeGroup maxOccurs=>`,
			child: `<xs:attributeGroup name="AG" maxOccurs="unbounded"><xs:attributeGroup ref="tns:AG"/></xs:attributeGroup>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", redefining(tc.child))
			if err == nil {
				return
			}
			if strings.Contains(err.Error(), "prohibits on the redefining form") {
				t.Fatalf("error = %v, want no prohibition claimed for an attribute that kind's grammar never declares", err)
			}
		})
	}
}

// substitutedRedefiningGroup and its attributeGroup twin are the redefining
// declarations the tables below substitute FOR: legal in every particular, so a
// row's only possible fault is the substitute main.xsd writes in their place.
const (
	substitutedRedefiningGroup = `<xs:group name="G"><xs:sequence>` +
		`<xs:group ref="tns:G"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`
	substitutedRedefiningAG = `<xs:attributeGroup name="AG">` +
		`<xs:attributeGroup ref="tns:AG"/><xs:attribute name="b" type="xs:string"/></xs:attributeGroup>`
)

// substitutedRedefining is redefining's §F.2 clause 1 twin: main.xsd's
// <override> substitutes child for mid.xsd's redefining declaration of the same
// element type and name, and mid.xsd <redefine>s lib.xsd, which declares the
// originals. The substitute is written on line 3 of main.xsd, and the
// declaration it replaces on line 1 of mid.xsd, so the two positions are
// distinguishable (STYLE E3).
func substitutedRedefining(child, redefined string) map[string]string {
	return map[string]string{
		"main.xsd": wrap("urn:a", "\n"+`<xs:override schemaLocation="mid.xsd">`+"\n"+child+"\n"+`</xs:override>`),
		"mid.xsd":  wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+redefined+`</xs:redefine>`),
		"lib.xsd":  wrap("urn:a", redefiningLib),
	}
}

// TestParseSubstitutedRedefiningProhibitedAttrsRejected is the table above under
// §F.2 clause 1: the declaration an <override> SUBSTITUTES for a <redefine>
// child is the one the prohibitions bind, since §4.2.5 src-override clause 3's
// closing Note makes it Dold′ — the transformed document — that must correspond
// to a conforming schema (xmlschema11-1.md:4171), and clause 1 puts "a copy of
// E1", the <override> child with its own attributes, in the child's place
// (:6568). The grammar types are the table above's, unchanged: xs:namedGroup
// prohibits ref, minOccurs and maxOccurs, xs:namedAttributeGroup prohibits ref
// alone, and the fault carries no numbered rule ID.
//
// Every row but the last is a document that PARSES CLEANLY without the guard —
// mid.xsd's own redefining child is legal, so newRedefineSet's charge cannot
// stand in for it — so reverting the guard fails it on the verdict rather than
// on the wording.
//
// Two claims beyond the attribute itself:
//
//   - the POSITION is the substitute's own line in main.xsd, never the replaced
//     child's in mid.xsd, which is what an author has to edit;
//   - the LAST row's redefining child names nothing lib.xsd declares, so the
//     src-expredef pairing miss is live over the same entry and the row is
//     rejected either way. Its assertion is the ordering one: the prohibited
//     attribute must still be what is reported, which only a PRE-SCAN charge
//     preserves, since produceRedefinition charges that miss before it computes
//     the substitute at all.
//
// There is no nameless row, and none is possible: overrideSet.replacement keys
// on (element type, name), so a substitute with no name attribute stands in for
// nothing.
func TestParseSubstitutedRedefiningProhibitedAttrsRejected(t *testing.T) {
	const (
		groupBody = `<xs:sequence><xs:group ref="tns:G"/><xs:element name="c" type="xs:string"/></xs:sequence>`
		agBody    = `<xs:attributeGroup ref="tns:AG"/><xs:attribute name="c" type="xs:string"/>`
	)
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name      string
		child     string
		redefined string
		wantAttr  string
	}{
		{
			name:      `substituted <group ref=>`,
			child:     `<xs:group name="G" ref="tns:G">` + groupBody + `</xs:group>`,
			redefined: substitutedRedefiningGroup,
			wantAttr:  "ref",
		},
		{
			name:      `substituted <group minOccurs=>`,
			child:     `<xs:group name="G" minOccurs="0">` + groupBody + `</xs:group>`,
			redefined: substitutedRedefiningGroup,
			wantAttr:  "minOccurs",
		},
		{
			name:      `substituted <group maxOccurs=>`,
			child:     `<xs:group name="G" maxOccurs="100">` + groupBody + `</xs:group>`,
			redefined: substitutedRedefiningGroup,
			wantAttr:  "maxOccurs",
		},
		{
			// Two prohibited attributes at once, written in the REVERSE of the
			// grammar's declaration order: the check order is the grammar's, not the
			// document's, so the reported attribute is stable (STYLE D2).
			name:      `substituted <group maxOccurs= minOccurs=>`,
			child:     `<xs:group name="G" maxOccurs="100" minOccurs="0">` + groupBody + `</xs:group>`,
			redefined: substitutedRedefiningGroup,
			wantAttr:  "minOccurs",
		},
		{
			name:      `substituted <attributeGroup ref=>`,
			child:     `<xs:attributeGroup name="AG" ref="tns:AG">` + agBody + `</xs:attributeGroup>`,
			redefined: substitutedRedefiningAG,
			wantAttr:  "ref",
		},
		{
			name:      `substituted <group ref=> naming no original`,
			child:     `<xs:group name="Absent" ref="tns:G"><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:group>`,
			redefined: `<xs:group name="Absent"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`,
			wantAttr:  "ref",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", substitutedRedefining(tc.child, tc.redefined))
			if err == nil {
				t.Fatalf("Parse succeeded, want a grammar fault for the prohibited %s on the substitute", tc.wantAttr)
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if want := fmt.Sprintf("carries a %s attribute", tc.wantAttr); !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to name %s as the prohibited attribute", err, tc.wantAttr)
			}
			if !strings.Contains(err.Error(), "prohibits on the redefining form") {
				t.Fatalf("error = %v, want it to name the form it judged, which here is the redefining one", err)
			}
			if !strings.Contains(err.Error(), "main.xsd:3:") {
				t.Fatalf("error = %v, want it positioned at the substitute's own line main.xsd:3 (E3)", err)
			}
			if strings.Contains(err.Error(), "mid.xsd") {
				t.Fatalf("error = %v, want the substitute's position rather than the replaced child's in mid.xsd", err)
			}
		})
	}
}

// TestParseSubstitutedRedefiningProhibitedAttrsLegalElsewhere is the reverse
// hazard of the table above, in three parts.
//
// The parsing rows are §F.2 clause 1 substituting a well-formed redefining
// declaration of each kind, self-reference and all: the control that fails if
// the guard ever fires on a substitute carrying none of the prohibited
// attributes, which would break the resolution parser/override_redefine_test.go
// pins component by component.
//
// The mid.xsd row pins that a redefining child NOTHING substitutes for is still
// charged where it always was, at its own position in the document that writes
// it: newRedefineSet charges every <redefine> child unconditionally, and the
// prescan charge on a substitute is an addition to that rather than an alternate.
//
// The last row pins what the guard must NOT CLAIM: xs:attributeGroup's grammar
// never pulls in xs:occurs, so minOccurs= on a substituted <attributeGroup> is
// ABSENT from that grammar rather than prohibited, and the list stays per kind.
func TestParseSubstitutedRedefiningProhibitedAttrsLegalElsewhere(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name      string
		child     string
		redefined string
	}{
		{
			name: `substituted <group>`,
			child: `<xs:group name="G"><xs:sequence>` +
				`<xs:group ref="tns:G"/><xs:element name="c" type="xs:string"/></xs:sequence></xs:group>`,
			redefined: substitutedRedefiningGroup,
		},
		{
			name: `substituted <attributeGroup>`,
			child: `<xs:attributeGroup name="AG">` +
				`<xs:attributeGroup ref="tns:AG"/><xs:attribute name="c" type="xs:string"/></xs:attributeGroup>`,
			redefined: substitutedRedefiningAG,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMap(t, "main.xsd", substitutedRedefining(tc.child, tc.redefined)); err != nil {
				t.Fatalf("Parse rejected a substituted redefinition the grammar admits: %v", err)
			}
		})
	}
	t.Run(`unsubstituted <group ref=> under an override`, func(t *testing.T) {
		_, err := parseMap(t, "main.xsd", substitutedRedefining(
			`<xs:group name="Other"><xs:sequence><xs:element name="c" type="xs:string"/></xs:sequence></xs:group>`,
			`<xs:group name="G" ref="tns:G"><xs:sequence>`+
				`<xs:group ref="tns:G"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`))
		if err == nil {
			t.Fatal("Parse succeeded, want the redefining child's own ref rejected as before")
		}
		if !strings.Contains(err.Error(), "carries a ref attribute") {
			t.Fatalf("error = %v, want it to name ref as the prohibited attribute", err)
		}
		if !strings.Contains(err.Error(), "mid.xsd:1:") {
			t.Fatalf("error = %v, want it positioned at the redefining child's own line mid.xsd:1 (E3)", err)
		}
	})
	t.Run(`substituted <attributeGroup minOccurs=>`, func(t *testing.T) {
		_, err := parseMap(t, "main.xsd", substitutedRedefining(
			`<xs:attributeGroup name="AG" minOccurs="0"><xs:attributeGroup ref="tns:AG"/></xs:attributeGroup>`,
			substitutedRedefiningAG))
		if err == nil {
			return
		}
		if strings.Contains(err.Error(), "prohibits on the redefining form") {
			t.Fatalf("error = %v, want no prohibition claimed for an attribute that kind's grammar never declares", err)
		}
	})
}

// TestProduceProhibitedAttrsLegalElsewhere is the reverse hazard of the tables
// above, and the reason the guard carries a list per KIND rather than one merged
// list.
//
// The local rows produce cleanly: form, use, targetNamespace, minOccurs and
// maxOccurs are exactly what xs:localElement and xs:localAttribute admit, and
// ref is REQUIRED on xs:groupRef and xs:attributeGroupRef, which are the only
// forms a <group>/<attributeGroup> inside a complex type may take — so a guard
// that widened to every <element>, <attribute>, <group> or <attributeGroup>
// would reject legal schemas wholesale. Each local targetNamespace= repeats the
// schema's own, which src-attribute clause 6.3 and src-element clause 4.3 leave
// unconstrained.
//
// The asymmetric rows pin what the guard must NOT CLAIM: use= on an <element>,
// minOccurs=/maxOccurs= on an <attribute> and minOccurs=/maxOccurs= on an
// <attributeGroup> appear in no grammar for those elements at any level, so
// xs:topLevelElement, xs:topLevelAttribute and xs:namedAttributeGroup do not
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
	const (
		targetGroup = `<xs:group name="G"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`
		targetAG    = `<xs:attributeGroup name="AG"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`
	)
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
		{`local <group ref=>`, targetGroup + local(`<xs:group ref="tns:G"/>`)},
		{`local <group ref= minOccurs=>`, targetGroup + local(`<xs:group ref="tns:G" minOccurs="0"/>`)},
		{`local <group ref= maxOccurs=>`, targetGroup + local(`<xs:group ref="tns:G" maxOccurs="unbounded"/>`)},
		{`local <attributeGroup ref=>`, targetAG + localAttr(`<xs:attributeGroup ref="tns:AG"/>`)},
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
		{`top-level <attributeGroup minOccurs=>`, `<xs:attributeGroup name="AG" minOccurs="0"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`},
		{`top-level <attributeGroup maxOccurs=>`, `<xs:attributeGroup name="AG" maxOccurs="unbounded"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.body))
			if err == nil {
				return
			}
			if strings.Contains(err.Error(), "prohibits on the top-level form") {
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

// TestProduceRefFormNameProhibited is the REFERENCE half of the family the
// definition tables above cover, and the two grammar types are the mirror image
// of theirs: xs:groupRef makes ref required and restricts name to
// use="prohibited" (xmlschema11-1.md:5223-:5224), and xs:attributeGroupRef does
// the same (:5522-:5523). Each row asserts the grammar type BY NAME, so a guard
// that rejected both positions under one label would fail.
//
// The fault is plain, never a rule verdict: §3.7.3 and src-attribute_group
// (§3.6.3) both read "None as such." in full, so the only footing is §5.1's
// requirement that a schema document be fully valid against the Schema for Schema
// Documents (xmlschema11-1.md:4296), which carries no numbered ID — charging
// src-resolve, mgd-props-correct or ag-props-correct here would be fabricated
// (STYLE E2).
//
// Two row shapes, and each carries its own failure mode:
//
//   - ref AND name TOGETHER — documents that PRODUCE cleanly without the guard,
//     since nothing else in the producer reads name at these positions, so the row
//     fails on the verdict rather than on the wording;
//   - name ALONE, a definition form written where a reference belongs — rejected
//     either way, so its assertion is the ordering one: the diagnostic must name
//     name and must NOT be the sibling missing-ref fault, which is what answers if
//     the guard runs after the ref lookup instead of before it.
//
// Every ref row DECLARES the definition its ref names, so no row can pass as a
// dangling reference, and the position assertion pins the offending element's own
// line (STYLE E3, carried in the message text since a plain error holds no
// xsderr.Loc).
func TestProduceRefFormNameProhibited(t *testing.T) {
	const (
		targetGroup = `<xs:group name="G"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`
		targetAG    = `<xs:attributeGroup name="AG"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`
	)
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts on
	// line 2 of the wrapped document.
	cases := []struct {
		name        string
		body        string
		wantGrammar string
		wantLine    int
	}{
		{
			name: `local <group ref= name=> in a content model`,
			body: "\n" + targetGroup + "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:group ref="tns:G" name="X"/>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantGrammar: "xs:groupRef",
			wantLine:    4,
		},
		{
			// The definition form written where a reference belongs: without the guard
			// this is the missing-ref fault, the consequence rather than the mistake.
			name: `local <group name=> with no ref`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence>` + "\n" +
				`<xs:group name="X"><xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>` + "\n" +
				`</xs:sequence></xs:complexType>`,
			wantGrammar: "xs:groupRef",
			wantLine:    3,
		},
		{
			// The reference nested inside a DEFINITION of the same kind, whose own name
			// attribute is required and must not be confused for this one.
			name: `local <group ref= name=> inside a <group> definition`,
			body: "\n" + targetGroup + "\n" + `<xs:group name="G2"><xs:sequence>` + "\n" +
				`<xs:group ref="tns:G" name="X"/>` + "\n" +
				`</xs:sequence></xs:group>`,
			wantGrammar: "xs:groupRef",
			wantLine:    4,
		},
		{
			name: `nested <attributeGroup ref= name=> on a complex type`,
			body: "\n" + targetAG + "\n" + `<xs:complexType name="CT"><xs:sequence/>` + "\n" +
				`<xs:attributeGroup ref="tns:AG" name="X"/>` + "\n" +
				`</xs:complexType>`,
			wantGrammar: "xs:attributeGroupRef",
			wantLine:    4,
		},
		{
			name: `nested <attributeGroup name=> with no ref`,
			body: "\n" + `<xs:complexType name="CT"><xs:sequence/>` + "\n" +
				`<xs:attributeGroup name="X"><xs:attribute name="b" type="xs:string"/></xs:attributeGroup>` + "\n" +
				`</xs:complexType>`,
			wantGrammar: "xs:attributeGroupRef",
			wantLine:    3,
		},
		{
			// The member-list position: a nested reference inside a top-level
			// <attributeGroup>, whose own name is required.
			name: `nested <attributeGroup ref= name=> in a member list`,
			body: "\n" + targetAG + "\n" + `<xs:attributeGroup name="AG2">` + "\n" +
				`<xs:attributeGroup ref="tns:AG" name="X"/>` + "\n" +
				`</xs:attributeGroup>`,
			wantGrammar: "xs:attributeGroupRef",
			wantLine:    4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", tc.body))
			if err == nil {
				t.Fatal("Produce succeeded, want a grammar fault for the prohibited name")
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v (rule %s), want a plain Go error rather than a rule verdict", err, xe.Rule)
			}
			if !strings.Contains(err.Error(), "carries a name attribute") {
				t.Fatalf("error = %v, want it to name name as the prohibited attribute", err)
			}
			if !strings.Contains(err.Error(), "prohibits on the reference form") {
				t.Fatalf("error = %v, want it to name the form it judged, which here is the reference one", err)
			}
			if want := tc.wantGrammar + " restricts name"; !strings.Contains(err.Error(), want) {
				t.Fatalf("error = %v, want it to cite the grammar type %s", err, tc.wantGrammar)
			}
			if strings.Contains(err.Error(), "must be a reference") {
				t.Fatalf("error = %v, want the prohibited-name fault rather than the missing-ref one it causes", err)
			}
			if at := fmt.Sprintf("%s:%d:", produceURI, tc.wantLine); !strings.Contains(err.Error(), at) {
				t.Fatalf("error = %v, want it positioned at %s (E3)", err, at)
			}
		})
	}
}

// TestProduceRefFormNameLegalElsewhere is the reverse hazard of the table above:
// name is REQUIRED on the definition forms of both kinds (xs:namedGroup :5209,
// xs:namedAttributeGroup :5510) and on the local <element>/<attribute> siblings a
// content model or member list holds, so a guard reading name wherever it saw one
// would reject legal schemas wholesale.
//
// The redefining rows put a nameless reference form and a required-name
// definition form of the SAME element name in one pair — §4.2.4's licensed
// self-reference, which is what breaks under a guard keyed on the element's local
// name rather than on the position it is mapped in.
func TestProduceRefFormNameLegalElsewhere(t *testing.T) {
	const (
		targetGroup = `<xs:group name="G"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`
		targetAG    = `<xs:attributeGroup name="AG"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`
	)
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: `local <group ref=> beside a named local <element>`,
			body: targetGroup + `<xs:complexType name="CT"><xs:sequence>` +
				`<xs:group ref="tns:G"/><xs:element name="b" type="xs:string"/>` +
				`</xs:sequence></xs:complexType>`,
		},
		{
			name: `nested <attributeGroup ref=> beside a named local <attribute>`,
			body: targetAG + `<xs:complexType name="CT"><xs:sequence/>` +
				`<xs:attributeGroup ref="tns:AG"/><xs:attribute name="b" type="xs:string"/>` +
				`</xs:complexType>`,
		},
		{
			name: `<group ref=> inside a named <group> definition`,
			body: targetGroup + `<xs:group name="G2"><xs:sequence><xs:group ref="tns:G"/></xs:sequence></xs:group>`,
		},
		{
			name: `<attributeGroup ref=> inside a named <attributeGroup> definition`,
			body: targetAG + `<xs:attributeGroup name="AG2"><xs:attributeGroup ref="tns:AG"/></xs:attributeGroup>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := produce(t, wrap("urn:po", tc.body)); err != nil {
				t.Fatalf("Produce rejected a reference the grammar admits: %v", err)
			}
		})
	}
	for _, tc := range []struct {
		name  string
		child string
	}{
		{
			name:  `redefining <group name=> holding its own nameless self-reference`,
			child: `<xs:group name="G"><xs:sequence><xs:group ref="tns:G"/><xs:element name="b" type="xs:string"/></xs:sequence></xs:group>`,
		},
		{
			name:  `redefining <attributeGroup name=> holding its own nameless self-reference`,
			child: `<xs:attributeGroup name="AG"><xs:attributeGroup ref="tns:AG"/><xs:attribute name="b" type="xs:string"/></xs:attributeGroup>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseMap(t, "main.xsd", redefining(tc.child)); err != nil {
				t.Fatalf("Parse rejected a redefinition the grammar admits: %v", err)
			}
		})
	}
}
