package conformance

import (
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// xsiNS binds xsi on an instance root.
const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

// TestInstanceExecutorDecidesSimpleLeafRoot proves the lane's one "valid"
// observation (#1738): a root of the simple-leaf-root shape whose walk charged
// nothing and recorded nothing unevaluated is decided VALID — it agrees with a
// suite-valid case and disagrees with a suite-invalid one, which a decline never
// does. The location hints and namespace declarations are the attributes the
// shape admits, and a default {value constraint} is decided by the walk (cvc-elt
// clause 5.1) and admitted with them.
func TestInstanceExecutorDecidesSimpleLeafRoot(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{"a string-typed leaf root", knownRoot, `<known>x</known>`},
		{"an empty string-typed leaf root", knownRoot, `<known/>`},
		{
			"a leaf root carrying the two xsi: location hints and namespace declarations",
			knownRoot,
			`<known ` + xsiNS + ` xmlns:p="urn:p" xsi:noNamespaceSchemaLocation="s.xsd" xsi:schemaLocation="urn:p p.xsd">x</known>`,
		},
		{
			"an empty leaf root whose default is a value of its type",
			`<xs:element name="known" type="xs:int" default="7"/>`, `<known/>`,
		},
		{
			"a leaf root of a restricted, listed and unioned type",
			`<xs:element name="known" type="U"/>` +
				`<xs:simpleType name="R"><xs:restriction base="xs:int"><xs:minInclusive value="0"/></xs:restriction></xs:simpleType>` +
				`<xs:simpleType name="L"><xs:list itemType="R"/></xs:simpleType>` +
				`<xs:simpleType name="U"><xs:union memberTypes="L xs:boolean"/></xs:simpleType>`,
			`<known>1 2 3</known>`,
		},
	}
	for _, tc := range cases {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: a clean walk of the simple-leaf-root shape is valid; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorDeclinesOutsideSimpleLeafRoot proves each condition of
// the simple-leaf-root gate that the WALK does not also enforce declines on its
// own. Every row walks clean — no violation, no unevaluated record — so it is
// the gate alone that keeps the empty Result from reading as "valid": with
// simpleLeafRoot answering true, each row passes under its valid expectation and
// this test fails.
func TestInstanceExecutorDeclinesOutsideSimpleLeafRoot(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{
			"a {nillable} declaration (cvc-elt clause 3, both arms deferred)",
			`<xs:element name="known" type="xs:string" nillable="true"/>`, `<known>x</known>`,
		},
		{
			"a fixed {value constraint} (cvc-elt clause 5.2.2)",
			`<xs:element name="known" type="xs:string" fixed="x"/>`, `<known>x</known>`,
		},
		{
			"an {identity-constraint definitions} member (cvc-elt clause 6)",
			`<xs:element name="known" type="xs:string"><xs:unique name="u">` +
				`<xs:selector xpath="."/><xs:field xpath="."/></xs:unique></xs:element>`,
			`<known>x</known>`,
		},
		{
			"a {type table} (conditional type assignment)",
			`<xs:element name="known" type="xs:string"><xs:alternative type="xs:string"/></xs:element>`,
			`<known>x</known>`,
		},
		{
			// An ID-typed root is not a row: the walk charges cvc-id for it today,
			// so the gate would not be what declines it. A union whose ID member
			// never becomes the ·validating type· walks clean instead.
			"a union with an ID member (cvc-elt clause 7, cvc-id), even where another member validates",
			`<xs:element name="known" type="U"/><xs:simpleType name="U"><xs:union memberTypes="xs:int xs:ID"/></xs:simpleType>`,
			`<known>5</known>`,
		},
		{
			"a list whose {item type definition} has an IDREF member",
			`<xs:element name="known" type="L"/><xs:simpleType name="L"><xs:list itemType="U"/></xs:simpleType>` +
				`<xs:simpleType name="U"><xs:union memberTypes="xs:int xs:IDREF"/></xs:simpleType>`,
			`<known>1 2</known>`,
		},
		{
			"a union with an ENTITY member, even where another member validates",
			`<xs:element name="known" type="U"/><xs:simpleType name="U"><xs:union memberTypes="xs:int xs:ENTITY"/></xs:simpleType>`,
			`<known>5</known>`,
		},
		{
			"a NOTATION-derived type",
			`<xs:notation name="n" public="p"/><xs:element name="known" type="N"/>` +
				`<xs:simpleType name="N"><xs:restriction base="xs:NOTATION"><xs:enumeration value="n"/></xs:restriction></xs:simpleType>`,
			`<known>n</known>`,
		},
		{
			"an xsi:type on the root (cvc-elt clause 4 deferred)",
			knownRoot, `<known ` + xsiNS + ` xmlns:xs="http://www.w3.org/2001/XMLSchema" xsi:type="xs:string">x</known>`,
		},
		{
			"a DOCTYPE, whose DTD could default an attribute the peek does not see",
			knownRoot, `<!DOCTYPE known><known>x</known>`,
		},
	}
	for _, tc := range cases {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.why)
	}
}

// TestInstanceExecutorDeclinesUnevaluatedLeaf proves a simple-leaf-root shape
// whose walk RECORDED a check it did not perform declines rather than reading
// the empty violation list as "valid": validate's cvc-type clause 3.1.3 decline
// over a type no backend maps (#1738 routed it into Unevaluated), and an
// assertions facet, whose {test} validate records and never evaluates.
func TestInstanceExecutorDeclinesUnevaluatedLeaf(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody string }{
		{"String Valid withheld over xs:anySimpleType", `<xs:element name="known" type="xs:anySimpleType"/>`},
		{
			"an unevaluated assertions facet",
			`<xs:element name="known" type="A"/><xs:simpleType name="A"><xs:restriction base="xs:string">` +
				`<xs:assertion test="true()"/></xs:restriction></xs:simpleType>`,
		},
	} {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, `<known>x</known>`, false), tc.why)
	}
}

// TestInstanceExecutorDeclinesVersionedSchema proves a schema document carrying
// any attribute in the versioning namespace declines the valid observation,
// even one §4.2.2 retains the element under: the gate is a superset of the
// vc:maxVersion GAP(parser) (#1002) that lets VC/vc006.n1 walk clean.
func TestInstanceExecutorDeclinesVersionedSchema(t *testing.T) {
	exec := newInstanceExec()
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "s.xsd")
	instancePath := filepath.Join(dir, "i.xml")
	writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
		` xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">`+
		`<xs:element name="known" type="xs:string" vc:minVersion="1.0"/></xs:schema>`)
	writeFixture(t, instancePath, `<known>x</known>`)
	declinesBothPolarities(t, exec, caseSpec{kind: kindInstance, doc: instancePath, schemaDoc: schemaPath},
		"a schema document carrying a versioning-namespace attribute")
}

// TestPeekLeafRootRefusesWhatTheWalkAlsoCharges pins the instance-side
// conditions the walk ALSO enforces — an element child (cvc-type clause 3.1.2),
// an attribute outside the xsi: four (3.1.1), an xsi:nil (cvc-elt clause 3) —
// at the peek itself, since the gate is a precondition in its own right and
// not a restatement of those charges. A peek error and an abstract declaration
// are refused too.
func TestPeekLeafRootRefusesWhatTheWalkAlsoCharges(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		why, instance string
		want          bool
	}{
		{"a bare leaf", `<known>x</known>`, true},
		{"an element child", `<known><a/></known>`, false},
		{"an attribute outside the xsi: four", `<known foo="1">x</known>`, false},
		{"an xsi:nil", `<known ` + xsiNS + ` xsi:nil="false">x</known>`, false},
		{"a malformed document", `<known>`, false},
	} {
		path := filepath.Join(dir, "i.xml")
		writeFixture(t, path, tc.instance)
		if _, got := peekLeafRoot(path); got != tc.want {
			t.Errorf("%s: peekLeafRoot = %v, want %v", tc.why, got, tc.want)
		}
	}

	// cvc-elt clause 2 charges an abstract root before the gate is read, so the
	// declaration condition is pinned against simpleLeafDeclaration directly,
	// with the same declaration minus {abstract} as the control.
	schemaPath := filepath.Join(dir, "s.xsd")
	writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+
		`<xs:element name="abstract" type="xs:string" abstract="true"/>`+
		`<xs:element name="concrete" type="xs:string"/></xs:schema>`)
	schema, _, _, err := assembleCase(strict.New(), schemaPath, nil)
	if err != nil {
		t.Fatalf("assembling the schema: %v", err)
	}
	for local, want := range map[string]bool{"abstract": false, "concrete": true} {
		d, ok := schema.Element(xsd.QName{Local: local})
		if !ok {
			t.Fatalf("no top-level declaration %q", local)
		}
		if got := simpleLeafDeclaration(schema, d); got != want {
			t.Errorf("simpleLeafDeclaration(%s) = %v, want %v", local, got, want)
		}
	}
}
