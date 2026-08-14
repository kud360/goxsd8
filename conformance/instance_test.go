package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// knownRoot is the one top-level element declaration most cases below assemble,
// so an instance rooted at <known> is DECLARED and one rooted at anything else is
// not — the dispatch the whole lane turns on.
const knownRoot = `<xs:element name="known" type="xs:string"/>`

// instanceCase writes one schema document and one instance document into a fresh
// directory and returns the caseSpec discovery builds for an instanceTest of that
// pair: the instance is the document under test, the schema is the group's.
// schemaBody is wrapped in a no-targetNamespace <schema>, so an instance root in
// no namespace is the one that can resolve to a declaration.
func instanceCase(t *testing.T, schemaBody, instance string, valid bool) caseSpec {
	t.Helper()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "s.xsd")
	instancePath := filepath.Join(dir, "i.xml")
	writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+schemaBody+`</xs:schema>`)
	writeFixture(t, instancePath, instance)
	return caseSpec{
		kind:      kindInstance,
		doc:       instancePath,
		schemaDoc: schemaPath,
		expect:    expectValidity(valid),
	}
}

func writeFixture(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// declinesBothPolarities asserts the executor records Fail for c AND for c with
// its declared outcome flipped — the exact test declines.go's census applies, and
// the only honest reading of "no verdict was reached". Asserting one polarity
// would pass for an executor that decided the case wrongly.
func declinesBothPolarities(t *testing.T, exec executor, c caseSpec, why string) {
	t.Helper()
	for _, valid := range []bool{true, false} {
		c.expect = expectValidity(valid)
		if exec(c).IsPass() {
			t.Errorf("%s: must Fail (decline) regardless of expectValid=%v", why, valid)
		}
	}
}

// TestInstanceExecutorDecidesUndeclaredRoot proves the one verdict this
// bring-up slice really reaches: a validation root with no top-level element
// declaration and no xsi:type determines neither a ·governing element
// declaration· nor a ·governing type definition· (cvc-assess-elt, §3.3.4.6), so
// under §5.2 ·strict wildcard validation· the document is NOT VALID whatever
// else it holds. The executor must agree with a suite-invalid case and disagree
// with a suite-valid one — never pass under both, which is what a decline looks
// like.
func TestInstanceExecutorDecidesUndeclaredRoot(t *testing.T) {
	exec := newInstanceExec()
	if !exec(instanceCase(t, knownRoot, `<unknown/>`, false)).IsPass() {
		t.Error("an undeclared root is not valid: the executor must agree with a suite-invalid case")
	}
	if exec(instanceCase(t, knownRoot, `<unknown/>`, true)).IsPass() {
		t.Error("the executor must Fail under a flipped expectation (it decides for real)")
	}
}

// TestInstanceExecutorDeclinesUndecidableShapes proves every shape this slice
// cannot decide is DECLINED in BOTH directions rather than guessed. The
// load-bearing row is the first: a declared, non-abstract root charges NOTHING,
// and an empty validate.Result is not evidence of validity — §3.3.5.1's
// e-validity is a conjunction whose descendant clauses Assess never evaluates, so
// neither "valid" nor "invalid" may be claimed.
func TestInstanceExecutorDeclinesUndecidableShapes(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{
			"a declared, non-abstract root charges nothing, and no charge is not a verdict",
			knownRoot, `<known>x</known>`,
		},
		{
			// #716: the attribute is DETECTED, never ·resolved· (cvc-resolve-instance,
			// §3.17.6.3), so Assess withholds the cvc-assess-elt charge — this is NOT
			// the undeclared-root case above and must not be scored as one.
			"an undeclared root carrying an unresolvable xsi:type withholds the charge",
			knownRoot, `<unknown xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xs:string"/>`,
		},
		{
			"an instance document the reader rejects is a gap, not a well-formedness verdict",
			knownRoot, `<unknown`,
		},
		{
			"a schema document outside the producer's decidable subset (inline global attribute type)",
			knownRoot + `<xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`,
			`<unknown/>`,
		},
		{
			"a schema the assembly REJECTED is not the schema the suite declared",
			knownRoot + `<xs:simpleType name="T"><xs:restriction base="xs:string"/></xs:simpleType>` +
				`<xs:simpleType name="T"><xs:restriction base="xs:int"/></xs:simpleType>`,
			`<unknown/>`,
		},
	}
	for _, tc := range cases {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.why)
	}

	// The non-<schema> schema document needs a root instanceCase's wrapper cannot
	// produce, so it is written out here rather than joining the table.
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "s.xsd")
	instancePath := filepath.Join(dir, "i.xml")
	writeFixture(t, schemaPath, `<notaschema/>`)
	writeFixture(t, instancePath, `<unknown/>`)
	declinesBothPolarities(t, exec, caseSpec{kind: kindInstance, doc: instancePath, schemaDoc: schemaPath},
		"a schema document that is not <schema>-rooted")
}

// TestInstanceExecutorDeclinesCaseWithNoGroupSchema proves a case discovery could
// not attach a schema to — a group not declaring exactly one schemaTest
// (groupSchemaDocs) — is DECLINED rather than assessed against a guessed or empty
// schema.
func TestInstanceExecutorDeclinesCaseWithNoGroupSchema(t *testing.T) {
	exec := newInstanceExec()
	c := instanceCase(t, knownRoot, `<unknown/>`, false)
	c.schemaDoc = ""
	declinesBothPolarities(t, exec, c, "an instance case with no group schema reference")
}

// abstractRootValidator is a validator over a schema whose one top-level element
// declaration has {abstract} true. It is built PROGRAMMATICALLY because the
// producer hardcodes {abstract} false for a top-level <element>
// (producer.produceElement in parser/produce.go, #761), so no schema this lane
// assembles from a document can reach the cvc-elt branch — see instance.go's file
// comment.
func abstractRootValidator(t *testing.T, name xsd.QName) *validate.Validator {
	t.Helper()
	e, err := xsd.NewElementDeclaration(xsderr.Loc{}, name, nil, nil, xsd.NewGlobalScope(),
		nil, false, nil, nil, nil, true, nil, nil)
	if err != nil {
		t.Fatalf("building the %s element declaration: %v", name, err)
	}
	b := xsd.NewSchemaBuilder()
	b.AddElement(e)
	schema, err := b.Finalize()
	if err != nil {
		t.Fatalf("finalizing the schema: %v", err)
	}
	v, err := validate.New(schema)
	if err != nil {
		t.Fatalf("validate.New: %v", err)
	}
	return v
}

// TestAssessInstanceDeclinesFaultedWalk proves the validate.Result.Err gate is
// load-bearing rather than decorative. An abstract root is the one Assess branch
// that charges a violation and STILL walks the subtree, so a source fault below it
// yields a Result that carries a decidable violation AND stopped early. Reading
// that as a verdict would score a document the walk never finished reading.
//
// The well-formed row is the control: the same schema and the same charge, with
// the walk reaching the end, is accepted and IS decidable — so the faulted row
// fails for the fault and not for the shape.
func TestAssessInstanceDeclinesFaultedWalk(t *testing.T) {
	v := abstractRootValidator(t, xsd.QName{Local: "e"})
	dir := t.TempDir()

	whole := filepath.Join(dir, "whole.xml")
	writeFixture(t, whole, `<e><a/></e>`)
	result, ok := assessInstance(v, whole)
	if !ok {
		t.Fatal("a walk that reached the end of the document must be accepted")
	}
	if !decidedNotValid(result.Violations()) {
		t.Errorf("an abstract root charges cvc-elt clause 2, which is decidable; got %d violation(s)", len(result.Violations()))
	}

	faulted := filepath.Join(dir, "faulted.xml")
	writeFixture(t, faulted, `<e><a></b></e>`)
	if _, ok := assessInstance(v, faulted); ok {
		t.Error("a walk stopped by a source fault mid-document must be DECLINED, whatever it charged before stopping")
	}
}

// TestDecidedNotValidEnumeratesTheDecidableCharges pins the gate itself: a
// non-empty violation set whose every rule is one Assess charges is evidence the
// document is not valid, and every other shape declines rather than being read as
// a verdict a later, wider Assess might charge under an approximation.
func TestDecidedNotValidEnumeratesTheDecidableCharges(t *testing.T) {
	charge := func(rule xsderr.Rule) *xsderr.Error {
		return xsderr.New(rule, xsderr.Loc{}, "charged")
	}
	cases := []struct {
		name       string
		violations []*xsderr.Error
		want       bool
	}{
		{"no violation at all", nil, false},
		{"cvc-assess-elt alone", []*xsderr.Error{charge(ruleCvcAssessElt)}, true},
		{"cvc-elt alone", []*xsderr.Error{charge(ruleCvcElt)}, true},
		{"cvc-complex-type alone", []*xsderr.Error{charge(ruleCvcComplexType)}, true},
		{"a rule outside the enumeration", []*xsderr.Error{charge("cvc-attribute")}, false},
		{"two charges, both enumerated", []*xsderr.Error{charge(ruleCvcComplexType), charge(ruleCvcComplexType)}, true},
		{"one enumerated, one not", []*xsderr.Error{charge(ruleCvcElt), charge("cvc-attribute")}, false},
	}
	for _, tc := range cases {
		if got := decidedNotValid(tc.violations); got != tc.want {
			t.Errorf("%s: decidedNotValid = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestInstanceExecutorAgreesWithSuite drives the real executor over a real suite
// instanceTest fixture and its real sibling schema, so the discovery-side wiring
// and the executor are proved together on the shape the lane actually meets.
// Skips when the submodule is absent.
func TestInstanceExecutorAgreesWithSuite(t *testing.T) {
	if _, err := os.Stat(suitePath()); err != nil {
		t.Skipf("W3C suite not present; run `git submodule update --init %s`", suiteRoot)
	}
	exec := newInstanceExec()
	dir := filepath.Join(suiteRoot, "sunData", "ElemDecl", "typeDef", "typeDef00201m")
	c := caseSpec{
		kind:      kindInstance,
		doc:       filepath.Join(dir, "typeDef00201m1.xml"),
		schemaDoc: filepath.Join(dir, "typeDef00201m.xsd"),
		expect:    expectValid(),
	}
	// The root IS declared by that schema, so nothing is charged and the case
	// declines — the honest recorded gap this slice is bounded to.
	declinesBothPolarities(t, exec, c, "a declared root against a real suite schema")
}
