package conformance

import (
	"path/filepath"
	"testing"
)

// childAssertCase is an instance case over a schema document whose <xs:schema>
// carries schemaAttrs and holds body, and an instance whose root element
// carries rootAttrs and holds kids: the shape of the D4_3_15 xpathDefaultNamespace
// fixtures, whose schema element varies where instanceCase's is fixed.
func childAssertCase(t *testing.T, schemaAttrs, body, rootAttrs, kids string, valid bool) caseSpec {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "s.xsd")
	instancePath := filepath.Join(dir, "i.xml")
	writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+schemaAttrs+`>`+body+`</xs:schema>`)
	writeFixture(t, instancePath, `<root`+rootAttrs+`>`+kids+`</root>`)
	return caseSpec{kind: kindInstance, doc: instancePath, schemaDoc: schemaPath, expect: expectValidity(valid)}
}

// e1Asserted is <root> over one xs:string <e1>, asserting `e1 = 'present'`
// under the xpathDefaultNamespace spelling given.
func e1Asserted(xpathDefault string) string {
	return `<xs:element name="root"><xs:complexType><xs:sequence>` +
		`<xs:element name="e1" type="xs:string"/>` +
		`</xs:sequence><xs:assert test="e1 = 'present'" xpathDefaultNamespace="` + xpathDefault + `"/>` +
		`</xs:complexType></xs:element>`
}

// An assertion naming <root>'s child by an unprefixed child-axis step is
// decided through the instance lane's executor, the assessed-subtree-root gate
// included, under each xpathDefaultNamespace spelling the D4_3_15 fixtures use
// (ii21/v21 through ii27/v27): ##targetNamespace and the explicit target URI
// over a qualified <e1>, and ##defaultNamespace with no default namespace
// declared and ##local over an unqualified one. <e1>present</e1> is decided
// VALID and <e1>absent</e1> or an empty <e1> INVALID, each agreeing with its
// declared outcome and disagreeing with the flipped one. ##local over a
// QUALIFIED <e1> names a child whose ·locally declared type· is ·absent·, so
// the {test} is declined, and the case is decided neither way. The test fails
// with xpath's ctaAssertionFacade.child declining every child step, and with
// ctaParser.elementName ignoring the {default namespace}.
func TestInstanceExecutorDecidesChildElementAssertions(t *testing.T) {
	const tns = ` targetNamespace="urn:t" elementFormDefault="qualified"`
	exec := newInstanceExec().status()
	for _, tc := range []struct {
		why, schemaAttrs, xpathDefault, rootAttrs string
	}{
		{"##targetNamespace", tns, "##targetNamespace", ` xmlns="urn:t"`},
		{"the target namespace's URI", tns, "urn:t", ` xmlns="urn:t"`},
		{"##defaultNamespace with no default namespace", "", "##defaultNamespace", ""},
		{"##local", "", "##local", ""},
	} {
		for _, row := range []struct {
			kids  string
			valid bool
		}{
			{`<e1>present</e1>`, true},
			{`<e1>absent</e1>`, false},
			{`<e1/>`, false},
		} {
			c := childAssertCase(t, tc.schemaAttrs, e1Asserted(tc.xpathDefault), tc.rootAttrs, row.kids, row.valid)
			if !exec(c).IsPass() {
				t.Errorf("%s over %s: want decided %v", tc.why, row.kids, row.valid)
			}
			c.expect = expectValidity(!row.valid)
			if exec(c).IsPass() {
				t.Errorf("%s over %s: the executor must Fail under a flipped expectation", tc.why, row.kids)
			}
		}
	}
	for _, valid := range []bool{true, false} {
		c := childAssertCase(t, tns, e1Asserted("##local"), ` xmlns="urn:t"`, `<e1>present</e1>`, valid)
		if exec(c).IsPass() {
			t.Errorf("##local over a qualified <e1>, expectValid=%v: want declined, decided neither way", valid)
		}
	}
}
