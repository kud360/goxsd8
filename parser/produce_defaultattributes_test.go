package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// These tests pin §3.4.2.4's (dcl.ctd.attuses) and §3.4.2.5's (dcl.ctd.anyatt)
// shared precondition: a <schema defaultAttributes> is folded into every
// <complexType> that does not carry defaultAttributesApply="false", as if the
// type had written an <attributeGroup ref> naming it after all its own (#1046).

// defaultAttributesSchema wraps body in a <schema targetNamespace="urn:x">
// declaring defaultAttributes="tns:DA" and an attribute group DA contributing a
// single "da" attribute — the group every fold in this file is looking for.
func defaultAttributesSchema(body string) string {
	return wrapDefaults("urn:x", `defaultAttributes="tns:DA"`,
		`<xs:attributeGroup name="DA"><xs:attribute name="da" type="xs:string"/></xs:attributeGroup>`+body)
}

// TestDefaultAttributesFoldReachesEveryComplexTypeForm folds the default group
// into the three <complexType> shapes the producer maps by three different
// paths: the implicit-content form, the <simpleContent> derivation, and the
// <complexContent> derivation. §3.4.2.4 states the precondition once, for all of
// them ("This mapping rule is the same for all complex type definitions").
func TestDefaultAttributesFoldReachesEveryComplexTypeForm(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"implicit content", `<xs:complexType name="T"><xs:sequence/></xs:complexType>`},
		{"simpleContent derivation", `<xs:complexType name="T"><xs:simpleContent>` +
			`<xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`},
		// Base opts OUT so the ONE folded da comes from T. Were it to fold too,
		// clause 3.1 would inherit its da unconditionally beside T's and
		// ct-props-correct clause 4 would reject — the same verdict a base and its
		// extension both writing one <attributeGroup ref> already earn, and not a
		// question this fold decides.
		{"complexContent derivation", `<xs:complexType name="Base" defaultAttributesApply="false"><xs:sequence/></xs:complexType>` +
			`<xs:complexType name="T"><xs:complexContent><xs:extension base="tns:Base">` +
			`<xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`},
	}
	for _, tc := range cases {
		s, err := produce(t, defaultAttributesSchema(tc.body))
		if err != nil {
			t.Fatalf("%s: Produce: %v", tc.name, err)
		}
		uses := topComplexTypeIn(t, s, xq("T")).AttributeUses()
		if !hasAttrUse(uses, "da") {
			t.Errorf("%s: {attribute uses} = %d uses, none named da — §3.4.2.4's precondition folds the <schema defaultAttributes> group into this form too", tc.name, len(uses))
		}
	}
}

// TestDefaultAttributesApplyOptOut pins the defaultAttributesApply test: only an
// ·actual value· of false opts out. The attribute's declared default is true, so
// an absent one folds, and so does an explicit "true" — while both lexicals of
// false, "false" and "0", suppress the fold.
func TestDefaultAttributesApplyOptOut(t *testing.T) {
	cases := []struct {
		name  string
		attr  string
		folds bool
	}{
		{"absent defaults to true", "", true},
		{`defaultAttributesApply="true"`, ` defaultAttributesApply="true"`, true},
		{`defaultAttributesApply="false"`, ` defaultAttributesApply="false"`, false},
		{`defaultAttributesApply="0"`, ` defaultAttributesApply="0"`, false},
	}
	for _, tc := range cases {
		s, err := produce(t, defaultAttributesSchema(
			`<xs:complexType name="T"`+tc.attr+`><xs:sequence/></xs:complexType>`))
		if err != nil {
			t.Fatalf("%s: Produce: %v", tc.name, err)
		}
		uses := topComplexTypeIn(t, s, xq("T")).AttributeUses()
		if got := hasAttrUse(uses, "da"); got != tc.folds {
			t.Errorf("%s: folded the default group = %v, want %v", tc.name, got, tc.folds)
		}
	}
}

// TestDefaultAttributesFoldsTheGroupWildcard pins §3.4.2.5: the synthesized
// reference feeds {attribute wildcard} on exactly the same footing as a written
// <attributeGroup ref>, so the default group's <anyAttribute> is INTERSECTED
// with the type's own (§3.6.2.2, cos-aw-intersect). urn:b is in both; urn:a is
// the default group's alone and urn:c the type's alone, and a fold that threaded
// only the uses through would leave both admitted.
func TestDefaultAttributesFoldsTheGroupWildcard(t *testing.T) {
	s, err := produce(t, wrapDefaults("urn:x", `defaultAttributes="tns:DA"`,
		`<xs:attributeGroup name="DA"><xs:anyAttribute namespace="urn:a urn:b"/></xs:attributeGroup>`+
			`<xs:complexType name="T"><xs:sequence/><xs:anyAttribute namespace="urn:b urn:c" processContents="lax"/></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	w, ok := topComplexTypeIn(t, s, xq("T")).AttributeWildcard()
	if !ok {
		t.Fatal("complex type T has no {attribute wildcard}, want the intersection with the default group's")
	}
	if !w.AllowsName(xsd.QName{Space: "urn:b", Local: "z"}) {
		t.Error("intersection must admit urn:b (in both the type's wildcard and the default group's)")
	}
	if w.AllowsName(xsd.QName{Space: "urn:a", Local: "z"}) {
		t.Error("intersection must reject urn:a (the default group's alone)")
	}
	if w.AllowsName(xsd.QName{Space: "urn:c", Local: "z"}) {
		t.Error("intersection must reject urn:c (the type's own alone) — §3.4.2.5 carries §3.4.2.4's precondition, so the default group's wildcard is combined here")
	}
	// {process contents} comes from the FIRST member of the §3.6.2.2 pre-order,
	// which is the type's own <anyAttribute> — the synthesized reference is
	// spliced "after any other <attributeGroup> [children]", so it can never
	// displace L at the head and hand the combination the group's strict.
	if w.ProcessContents() != xsd.ProcessLax {
		t.Errorf("{process contents} = %v, want lax from the type's own <anyAttribute>", w.ProcessContents())
	}
}

// TestDefaultAttributesFoldComesLast pins the precondition's placement: the
// synthesized <attributeGroup ref> appears "after any other <attributeGroup>
// [children]", so its uses land behind those of every ref the type wrote.
func TestDefaultAttributesFoldComesLast(t *testing.T) {
	s, err := produce(t, defaultAttributesSchema(
		`<xs:attributeGroup name="Other"><xs:attribute name="oa" type="xs:string"/></xs:attributeGroup>`+
			`<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="tns:Other"/></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	var order []string
	for _, u := range topComplexTypeIn(t, s, xq("T")).AttributeUses() {
		order = append(order, attrUseLocal(u))
	}
	if len(order) != 2 || order[0] != "oa" || order[1] != "da" {
		t.Fatalf("{attribute uses} order = %v, want [oa da]", order)
	}
}

// TestDefaultAttributesExplicitRefSplicesOnce pins the shared visited set: a
// <complexType> that ALSO writes an <attributeGroup ref> naming the default
// group contributes that group ONCE. §3.4.2.4 clause 2 is a set union, so the
// group's use appears once however many references reach it — splicing it twice
// would manufacture a ct-props-correct clause 4 collision the spec does not have.
func TestDefaultAttributesExplicitRefSplicesOnce(t *testing.T) {
	s, err := produce(t, defaultAttributesSchema(
		`<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="tns:DA"/></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce rejected a type naming the default group explicitly: %v", err)
	}
	uses := topComplexTypeIn(t, s, xq("T")).AttributeUses()
	n := 0
	for _, u := range uses {
		if attrUseLocal(u) == "da" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("{attribute uses} holds %d uses named da (of %d), want exactly 1", n, len(uses))
	}
}

// TestDefaultAttributesCollisionRejected proves the fold is a real contribution
// to {attribute uses} and not a lenient no-op: a type declaring its own
// <attribute> with the default group's expanded name holds two distinct uses
// sharing one name, which ct-props-correct (§3.4.6.1) clause 4 forbids.
func TestDefaultAttributesCollisionRejected(t *testing.T) {
	_, err := produce(t, defaultAttributesSchema(
		`<xs:complexType name="T"><xs:sequence/><xs:attribute name="da" type="xs:int"/></xs:complexType>`))
	if err == nil {
		t.Fatal("Produce accepted a type whose own <attribute> collides with the folded default group's, want ct-props-correct clause 4")
	}
	if !strings.Contains(err.Error(), "ct-props-correct") {
		t.Fatalf("error = %q, want it to cite ct-props-correct", err)
	}
}

// TestDefaultAttributesUnresolvableRejected pins the src-resolve (§3.17.6.2
// clause 1.4) verdict on a defaultAttributes naming no top-level attribute
// group: the synthesized reference resolves through the ordinary
// <attributeGroup ref> mechanism, and its position is the <schema> element that
// carries the attribute, not the <complexType> that triggered the fold.
func TestDefaultAttributesUnresolvableRejected(t *testing.T) {
	_, err := produce(t, wrapDefaults("urn:x", `defaultAttributes="tns:missing"`,
		`<xs:complexType name="T"><xs:sequence/></xs:complexType>`))
	if err == nil {
		t.Fatal("Produce accepted a defaultAttributes naming no attribute group, want src-resolve")
	}
	if !strings.Contains(err.Error(), "src-resolve") {
		t.Fatalf("error = %q, want it to cite src-resolve", err)
	}
	if !strings.Contains(err.Error(), "<schema defaultAttributes>") {
		t.Fatalf("error = %q, want it to name the <schema defaultAttributes> construct the author wrote", err)
	}
}

// TestDefaultAttributesUnresolvableUnusedIsNotCharged is the other half of that
// verdict: a document declaring defaultAttributes and defining NO <complexType>
// invokes ·resolve· on it nowhere, so an unresolvable QName costs nothing.
// §3.4.2.4's precondition is the only reader, and it synthesizes the reference
// per complex type rather than once per document.
func TestDefaultAttributesUnresolvableUnusedIsNotCharged(t *testing.T) {
	if _, err := produce(t, wrapDefaults("urn:x", `defaultAttributes="tns:missing"`,
		`<xs:element name="e" type="xs:string"/>`)); err != nil {
		t.Fatalf("Produce rejected an unreferenced defaultAttributes: %v", err)
	}
}

// TestDefaultAttributesChameleonRewrite pins §F.1 task (b) over the attribute:
// an <include>d document with no targetNamespace of its own writes an
// UNPREFIXED defaultAttributes="g", which the coercion binds to the INCLUDING
// document's namespace — the same namespace its own <attributeGroup name="g">
// is minted in. Resolving the QName at the <complexType> instead of at the
// <schema> that carries it would be invisible here; resolving it in the ·absent·
// namespace would not.
func TestDefaultAttributesChameleonRewrite(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:x", `<xs:include schemaLocation="lib.xsd"/>`),
		"lib.xsd": `<xs:schema xmlns:xs="` + xsdNS + `" defaultAttributes="g">` +
			`<xs:attributeGroup name="g"><xs:attribute name="da" type="xs:string"/></xs:attributeGroup>` +
			`<xs:complexType name="T"><xs:sequence/></xs:complexType>` +
			`</xs:schema>`,
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	uses := topComplexTypeIn(t, s, xq("T")).AttributeUses()
	if !hasAttrUse(uses, "da") {
		t.Fatalf("{attribute uses} = %d uses, none named da — a chameleon document's unprefixed defaultAttributes binds to the including namespace (§F.1)", len(uses))
	}
}

// TestDefaultAttributesUnderOverrideIsTheOverriddenDocuments pins which <schema>
// ancestor §3.4.2.4 means for a <complexType> substituted by an <override>: the
// OVERRIDDEN document's own root, because §4.2.5 makes the substituted
// declaration a top-level declaration of that document. main.xsd declares a
// defaultAttributes of its own precisely so the wrong reading is visible — an
// ancestor walk from the substituted element climbs into main.xsd and folds MAIN's
// group instead.
func TestDefaultAttributesUnderOverrideIsTheOverriddenDocuments(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrapDefaults("urn:a", `defaultAttributes="tns:mainDA"`,
			`<xs:attributeGroup name="mainDA"><xs:attribute name="fromMain" type="xs:string"/></xs:attributeGroup>`+
				`<xs:override schemaLocation="lib.xsd">`+
				`<xs:complexType name="T"><xs:sequence/></xs:complexType>`+
				`</xs:override>`),
		"lib.xsd": wrapDefaults("urn:a", `defaultAttributes="tns:libDA"`,
			`<xs:attributeGroup name="libDA"><xs:attribute name="fromLib" type="xs:string"/></xs:attributeGroup>`+
				`<xs:complexType name="T"><xs:sequence/></xs:complexType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	uses := topComplexTypeIn(t, s, xsd.QName{Space: "urn:a", Local: "T"}).AttributeUses()
	if !hasAttrUse(uses, "fromLib") {
		t.Errorf("{attribute uses} = %d uses, none named fromLib — a substituted <complexType> takes the OVERRIDDEN document's defaultAttributes (§4.2.5)", len(uses))
	}
	if hasAttrUse(uses, "fromMain") {
		t.Error("{attribute uses} holds fromMain — the overriding document's defaultAttributes governs its own complex types, never one it substitutes into another document")
	}
}
