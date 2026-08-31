package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/xmlsrc"
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

// TestInstanceExecutorDecidesAbstractRoot proves the slice's SECOND verdict is
// reachable from a schema DOCUMENT: a root whose declaration has {abstract} true
// is locally invalid by cvc-elt clause 2 (§3.3.4.3), so e-validity clause 1.1.1.1
// fails and the document is NOT VALID whatever else it holds. It decides nothing
// unless producer.produceElement maps {abstract} off the attribute (#761).
func TestInstanceExecutorDecidesAbstractRoot(t *testing.T) {
	exec := newInstanceExec()
	const abstractRoot = `<xs:element name="known" type="xs:string" abstract="true"/>`
	if !exec(instanceCase(t, abstractRoot, `<known/>`, false)).IsPass() {
		t.Error("an abstract root is not valid: the executor must agree with a suite-invalid case")
	}
	if exec(instanceCase(t, abstractRoot, `<known/>`, true)).IsPass() {
		t.Error("the executor must Fail under a flipped expectation (it decides for real)")
	}
	// The control that the verdict turns on {abstract} and not on the shape of the
	// instance is TestInstanceExecutorDeclinesUndecidableShapes' first row: the
	// same document under a non-abstract declaration charges nothing and declines.
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
			// An undeclared root whose xsi:type ·resolves· determines a ·governing
			// type definition· of its own (key-governing-type-elem clause 8), so it
			// is ·strictly assessed· against that type and is NOT the
			// undeclared-root case above. It charges nothing here, and no charge is
			// not a verdict.
			"an undeclared root typed by a resolved xsi:type charges nothing",
			knownRoot + `<xs:complexType name="T"/>`,
			`<unknown xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="T"/>`,
		},
		{
			"an instance document the reader rejects is a gap, not a well-formedness verdict",
			knownRoot, `<unknown`,
		},
		{
			"a schema document outside the producer's decidable subset (a simpleType naming no §3.16.2.1 alternative)",
			knownRoot + `<xs:simpleType name="undec"/>`,
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
// declaration has {abstract} true. It assembles that schema from a DOCUMENT,
// through the same gate the lane itself uses, because the producer maps
// {abstract} from the attribute (§3.3.2.1 dcl.elt.common, #761) — so the cvc-elt
// branch this exercises is the one a real case reaches, not a hand-built
// declaration's.
func abstractRootValidator(t *testing.T) *validate.Validator {
	t.Helper()
	schemaPath := filepath.Join(t.TempDir(), "s.xsd")
	writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+
		`<xs:element name="e" type="xs:string" abstract="true"/></xs:schema>`)
	schema, _, err := assembleCase(strict.New(), schemaPath, nil)
	if err != nil {
		t.Fatalf("assembling the schema: %v", err)
	}
	v, err := validate.New(schema, strict.New())
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
	v := abstractRootValidator(t)
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
		{"cvc-type alone", []*xsderr.Error{charge(ruleCvcType)}, true},
		{"cvc-complex-type alone", []*xsderr.Error{charge(ruleCvcComplexType)}, true},
		{"cvc-attribute alone", []*xsderr.Error{charge(ruleCvcAttribute)}, true},
		{"cvc-au alone", []*xsderr.Error{charge(ruleCvcAu)}, true},
		{"a rule outside the enumeration", []*xsderr.Error{charge("cvc-assertion")}, false},
		{"two charges, both enumerated", []*xsderr.Error{charge(ruleCvcAttribute), charge(ruleCvcAu)}, true},
		{"one enumerated, one not", []*xsderr.Error{charge(ruleCvcElt), charge("cvc-assertion")}, false},
	}
	for _, tc := range cases {
		if got := decidedNotValid(tc.violations); got != tc.want {
			t.Errorf("%s: decidedNotValid = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// TestParsedAnonymousExtensionIsNotFalselyRejected proves the shape whose two
// attribute properties are folded through the OWNING SLOT alone (#414) is
// reachable from a PARSED document and not only from a hand-assembled
// xsd.Schema: produceComplexType dispatches on the
// <complexContent>/<simpleContent> children before it considers whether the type
// is named, so an inline <complexContent><extension> under an <element> is
// produced exactly as a top-level one is.
//
// The document below is VALID — @fromBase is the base's own attribute use and
// @own the extension's — so the assessment has to get both of the anonymous
// type's folded attribute properties right or charge cvc-complex-type clause 2
// for an attribute the base declares. This test lives here rather than in
// validate because it needs the parser, which validate's import closure excludes
// (validate/imports_test.go); it goes through parser.Parse and xmlsrc.Validate
// directly rather than through the lane executor, since assembleCase's
// decidability gate declines this schema before validate ever sees it — which is
// why no suite case scores this shape either way.
func TestParsedAnonymousExtensionIsNotFalselyRejected(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, filepath.Join(dir, "s.xsd"), `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
	  <xs:complexType name="B">
	    <xs:attribute name="fromBase" type="xs:string"/>
	    <xs:anyAttribute namespace="##any"/>
	  </xs:complexType>
	  <xs:element name="root"><xs:complexType><xs:complexContent>
	    <xs:extension base="B"><xs:attribute name="own" type="xs:string"/></xs:extension>
	  </xs:complexContent></xs:complexType></xs:element>
	  <xs:element name="plain"><xs:complexType>
	    <xs:attribute name="a" type="xs:string" use="required"/>
	  </xs:complexType></xs:element>
	</xs:schema>`)
	schema, err := parser.Parse("s.xsd", parser.WithResolver(loader.Dir(dir)), parser.WithBackend(strict.New()))
	if err != nil {
		t.Fatalf("parsing the schema: %v", err)
	}
	v, err := validate.New(schema, strict.New())
	if err != nil {
		t.Fatalf("validate.New: %v", err)
	}
	assess := func(instance string) []*xsderr.Error {
		t.Helper()
		result, err := xmlsrc.Validate(v, strings.NewReader(instance))
		if err != nil {
			t.Fatalf("xmlsrc.Validate(%s): %v", instance, err)
		}
		if result.Err() != nil {
			t.Fatalf("Err() = %v, want nil", result.Err())
		}
		return result.Violations()
	}
	if got := assess(`<root fromBase="x" own="y"/>`); len(got) != 0 {
		t.Errorf("Violations() = %v, want none: the anonymous extension's {attribute uses} hold both @fromBase and @own once §3.4.2.4 clause 3 has folded them, so cvc-complex-type clause 2 matches each of them to a use", got)
	}
	// The control, over an anonymous type of the other shape: the sibling
	// implicit-content one IS a restriction of xs:anyType, both folds are the
	// identity on it, and it charges its own {required} use — so the assertion
	// above is about the FOLDS and not about which anonymous shapes the lane
	// admits, which since #1126 is all of them (conformance/schema.go's
	// complexTypeDecidable).
	if got := assess(`<plain/>`); len(got) != 1 {
		t.Fatalf("Violations() = %v, want exactly one: an implicit-content anonymous type is still assessed", got)
	}
}

// TestInstanceExecutorAgreesWithSuite drives the real executor over a real suite
// instanceTest fixture and its real sibling schema, so the discovery-side wiring
// and the executor are proved together on the shape the lane actually meets.
// Skips when the submodule is absent.
func TestInstanceExecutorAgreesWithSuite(t *testing.T) {
	skipWithoutSuite(t)
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
