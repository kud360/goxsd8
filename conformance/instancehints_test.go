package conformance

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
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
		// Namespaces in XML 1.1 lets a 1.1 document undeclare a prefix, so the
		// closure is read and decided like any other.
		{"noNamespaceSchemaLocation, an XML 1.1 schema undeclaring a prefix",
			[]fixtureFile{{"s.xsd", `<?xml version="1.1"?><xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="">` + knownRoot + `</xs:schema>`}},
			`<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd">x</known>`, true},
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
		{"a DOCTYPE whose internal subset holds an <!ATTLIST", known,
			`<!DOCTYPE known [<!ATTLIST known a CDATA #IMPLIED>]><known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"><x/></known>`},
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
		// §4.3.2 clause 5: an inline xs:schema is global to the assessment,
		// the document element included. Decided valid otherwise: s.xsd
		// imports a declaration of the root itself, so no undeclared-root
		// decline masks the row.
		{"an inline xs:schema as the root carrying a hint",
			[]fixtureFile{
				{"s.xsd", xsdDoc("", `<xs:import namespace="http://www.w3.org/2001/XMLSchema" schemaLocation="x.xsd"/>`)},
				{"x.xsd", xsdDoc("http://www.w3.org/2001/XMLSchema", `<xs:element name="schema"/>`)},
			},
			`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"/>`},
	} {
		declinesBothPolarities(t, exec, hintedCase(t, tc.files, tc.instance, true), tc.why)
	}
}

// TestHintedSchemaUndeclaringPrefixIsRejected pins addB139's shape: an XML 1.0
// hinted schema carrying xmlns:f="" is decidable, and its assembly is the
// parser's rejection under nsc-NoPrefixUndecl rather than a decline or a schema
// built as if the declaration were legal.
func TestHintedSchemaUndeclaringPrefixIsRejected(t *testing.T) {
	c := hintedCase(t,
		[]fixtureFile{{"s.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="">` + knownRoot + `</xs:schema>`}},
		`<known `+xsiNS+` xsi:noNamespaceSchemaLocation="s.xsd">x</known>`, false)
	_, _, decidable, perr := caseSchema(strict.New(), c)
	if !decidable {
		t.Fatal("caseSchema declined, want the closure read and decided")
	}
	if perr == nil || !strings.Contains(perr.Error(), "[xml-wf] namespace declaration xmlns:f has an empty value") {
		t.Errorf("caseSchema error = %v, want the parser's xml-wf rejection of xmlns:f=\"\"", perr)
	}
}

// hintedRejection is one hinted case whose assembly caseSchema reads as
// decidable and rejected, so execInstanceCase's perr arm is the gate it meets.
type hintedRejection struct {
	why      string
	files    []fixtureFile
	instance string
}

// noNSHint is a <known> root hinting s.xsd through xsi:noNamespaceSchemaLocation.
const noNSHint = `<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd">x</known>`

// reachesPerrArm fails t unless caseSchema reads c as decidable with a non-nil
// error, so no earlier gate declines it ahead of execInstanceCase's perr arm.
func reachesPerrArm(t *testing.T, c caseSpec, why string) {
	t.Helper()
	_, _, decidable, perr := caseSchema(strict.New(), c)
	if !decidable || perr == nil {
		t.Fatalf("%s: caseSchema = (decidable %v, %v), want a decidable rejection", why, decidable, perr)
	}
}

// TestInstanceExecutorDecidesNotWellFormedHintedSchema proves a hinted schema
// document the reader charged not well-formed is decided "not valid", the
// harness convention execInstanceCase states under §5.1 (#2058): addB138 and
// addB139's xmlns:f="" in an XML 1.0 document, and addB124's element left
// unclosed. Each row declines in both polarities with the decide arm removed.
func TestInstanceExecutorDecidesNotWellFormedHintedSchema(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []hintedRejection{
		{"an XML 1.0 schema undeclaring a prefix",
			[]fixtureFile{{"s.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:f="">` + knownRoot + `</xs:schema>`}},
			noNSHint},
		{"a schema leaving an element unclosed",
			[]fixtureFile{{"s.xsd", xsdDoc("", `<xs:simpleType name="t"><xs:restriction base="xs:string"><xs:whiteSpace value="collapse"></xs:restriction></xs:simpleType>`+knownRoot)}},
			noNSHint},
	} {
		c := hintedCase(t, tc.files, tc.instance, false)
		reachesPerrArm(t, c, tc.why)
		if !exec(c).IsPass() {
			t.Errorf("%s: must be decided not valid, agreeing with a suite-invalid case", tc.why)
		}
		c.expect = expectValidity(true)
		if exec(c).IsPass() {
			t.Errorf("%s: must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorDeclinesRejectedHintedSchema proves every other rejection
// of a hinted assembly is DECLINED in both polarities (#2058): a rejection of a
// document that read (schA8.i's src-import clause 3.1), a resolver fault on a
// composed document (#1201), a read failure wrapping a cause, and a prefix
// bound only by the namespace declaration an internal-subset ATTLIST defaults
// (unboundPrefixCharge). Every row reaches execInstanceCase's perr arm, and
// each is decided with wellFormednessFault removed from it.
func TestInstanceExecutorDeclinesRejectedHintedSchema(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []hintedRejection{
		{"a hinted document whose targetNamespace is not the hint's namespace",
			[]fixtureFile{{"s.xsd", xsdDoc("urn:other", knownRoot)}},
			`<t:known xmlns:t="urn:t" ` + xsiNS + ` xsi:schemaLocation="urn:t s.xsd">x</t:known>`},
		// ENOTDIR from loader.Dir: a resolver fault, not loader.ErrNotFound.
		{"an <include> whose location the resolver faults on",
			[]fixtureFile{{"s.xsd", xsdDoc("", `<xs:include schemaLocation="s.xsd/x.xsd"/>`+knownRoot)}},
			noNSHint},
		{"an encoding declaration the reader does not decode",
			[]fixtureFile{{"s.xsd", `<?xml version="1.0" encoding="ISO-8859-1"?>` + xsdDoc("", knownRoot)}},
			noNSHint},
		// #2067's shape: a well-formed document (XML 1.0 §5.1) the reader
		// charges with an unbound prefix, as it applies no attribute default.
		{"a namespace declaration an internal-subset ATTLIST defaults",
			[]fixtureFile{{"s.xsd", `<!DOCTYPE xs:schema [` +
				`<!ATTLIST xs:schema xmlns:xs CDATA #FIXED "http://www.w3.org/2001/XMLSchema">]>` +
				`<xs:schema>` + knownRoot + `</xs:schema>`}},
			noNSHint},
	} {
		c := hintedCase(t, tc.files, tc.instance, true)
		reachesPerrArm(t, c, tc.why)
		declinesBothPolarities(t, exec, c, tc.why)
	}
}
