package conformance

import (
	"path/filepath"
	"testing"
)

// fixtureFile is one document a hintedCase writes beside its instance.
type fixtureFile struct{ name, content string }

// hintedCase writes files and the instance document into a fresh directory and
// returns the caseSpec discovery builds for an instanceTest whose group
// declares no schemaTest: no group schema, so any schema comes from the
// instance's own hints (#2013).
func hintedCase(t *testing.T, files []fixtureFile, instance string, valid bool) caseSpec {
	t.Helper()
	dir := t.TempDir()
	for _, f := range files {
		writeFixture(t, filepath.Join(dir, f.name), f.content)
	}
	instancePath := filepath.Join(dir, "i.xml")
	writeFixture(t, instancePath, instance)
	return caseSpec{kind: kindInstance, doc: instancePath, expect: expectValidity(valid)}
}

// xsdDoc wraps body in a <schema>, with targetNamespace tns unless tns is "".
func xsdDoc(tns, body string) string {
	open := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	if tns != "" {
		open += ` targetNamespace="` + tns + `"`
	}
	return open + `>` + body + `</xs:schema>`
}

// skipKnown declares <known> with one child of any name, ·skipped· by a skip
// Wildcard, so whatever that child carries the walk reads none of it.
const skipKnown = `<xs:element name="known"><xs:complexType><xs:sequence>` +
	`<xs:any processContents="skip"/></xs:sequence></xs:complexType></xs:element>`

// TestInstanceExecutorDecidesFromHints proves a case with no group schema is
// decided against the schema its root's hints locate (§4.3.2 clauses 3-5,
// #2013), in both directions and through both hint attributes. Each row
// declines with caseSchema's hint arm removed, as every no-group-schema case
// did before it.
func TestInstanceExecutorDecidesFromHints(t *testing.T) {
	exec := newInstanceExec()
	noNS := []fixtureFile{{"s.xsd", xsdDoc("", knownRoot)}}
	withNS := []fixtureFile{{"s.xsd", xsdDoc("urn:t", knownRoot)}}
	for _, tc := range []struct {
		why      string
		files    []fixtureFile
		instance string
		valid    bool
	}{
		{"noNamespaceSchemaLocation, a valid leaf root", noNS,
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd">x</known>`, true},
		// cvc-type clause 3.1.2: an element [[child]] under a simple type.
		{"noNamespaceSchemaLocation, a leaf root with an element child", noNS,
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"><x/></known>`, false},
		{"schemaLocation, a valid leaf root", withNS,
			`<t:known xmlns:t="urn:t" ` + xsiNS + ` xsi:schemaLocation="urn:t s.xsd">x</t:known>`, true},
		{"schemaLocation, a leaf root with an element child", withNS,
			`<t:known xmlns:t="urn:t" ` + xsiNS + ` xsi:schemaLocation="urn:t s.xsd"><x/></t:known>`, false},
	} {
		if !exec(hintedCase(t, tc.files, tc.instance, tc.valid)).IsPass() {
			t.Errorf("%s: decided against the hinted schema, the executor must agree with a suite-%s case", tc.why, validityWord(tc.valid))
		}
		if exec(hintedCase(t, tc.files, tc.instance, !tc.valid)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

func validityWord(valid bool) string {
	if valid {
		return "valid"
	}
	return "invalid"
}

// TestInstanceExecutorDeclinesUnreadableHints proves every hinted shape the
// lane cannot read completely is DECLINED in both directions (instanceHints,
// assembleHints). Each row is decided with the condition it names removed.
func TestInstanceExecutorDeclinesUnreadableHints(t *testing.T) {
	exec := newInstanceExec()
	skip := []fixtureFile{{"s.xsd", xsdDoc("", skipKnown)}, {"o.xsd", xsdDoc("", `<xs:element name="other" type="xs:int"/>`)}}
	known := []fixtureFile{{"s.xsd", xsdDoc("", knownRoot)}}
	for _, tc := range []struct {
		why      string
		files    []fixtureFile
		instance string
	}{
		// §4.3.2 clause 5: a hint below the root is global to the assessment.
		{"a hint on a non-root element", skip,
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"><c xsi:noNamespaceSchemaLocation="o.xsd"/></known>`},
		{"an inline xs:schema below the root", skip,
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"><xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"/></known>`},
		// Decided invalid otherwise (cvc-type clause 3.1.2), with a DTD that
		// could default a hint onto any element.
		{"a DOCTYPE", known,
			`<!DOCTYPE known []><known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"><x/></known>`},
		{"an xml:base on the root", known,
			`<known ` + xsiNS + ` xml:base="sub/" xsi:noNamespaceSchemaLocation="s.xsd">x</known>`},
		{"an xsi:schemaLocation with an odd member count", known,
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd" xsi:schemaLocation="urn:t">x</known>`},
		// Decided valid otherwise, against a schema short of missing.xsd.
		{"a hint resolving to no document", known,
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd missing.xsd">x</known>`},
		// addB063's shape: the hinted schema declares no root.
		{"a hinted schema declaring no top-level element for the root", known,
			`<unknown ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"/>`},
		// addB139's shape: xmlns:p="" is not namespace-well-formed.
		{"a hinted schema carrying an empty prefixed namespace declaration",
			[]fixtureFile{{"s.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="">` + knownRoot + `</xs:schema>`}},
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd">x</known>`},
	} {
		declinesBothPolarities(t, exec, hintedCase(t, tc.files, tc.instance, true), tc.why)
	}
}
