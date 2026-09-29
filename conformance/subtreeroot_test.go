package conformance

import (
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// xsiXS binds xsi and xs on an instance root, for an xsi:type naming a builtin.
const xsiXS = xsiNS + ` xmlns:xs="http://www.w3.org/2001/XMLSchema"`

// notationN declares the notation n and the simple type N, a NOTATION
// enumeration admitting it alone (enumeration-required-notation).
const notationN = `<xs:notation name="n" public="p"/>` +
	`<xs:simpleType name="N"><xs:restriction base="xs:NOTATION"><xs:enumeration value="n"/></xs:restriction></xs:simpleType>`

// TestInstanceExecutorDecidesAssessedSubtreeRoot proves the lane's third
// "valid" observation (#1841): a root WITH content whose every element and
// attribute meets the assessed-subtree-root gate, and whose walk charged
// nothing and recorded nothing unevaluated, is decided VALID — it agrees with a
// suite-valid case and disagrees with a suite-invalid one. Each row is outside
// both leaf-root shapes, so it is this gate that decides it.
func TestInstanceExecutorDecidesAssessedSubtreeRoot(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{
			"a sequence of a local element and a referenced top-level one, beside a matched attribute",
			`<xs:element name="known"><xs:complexType><xs:sequence>` +
				`<xs:element name="a" type="xs:int"/><xs:element ref="b" minOccurs="0"/>` +
				`</xs:sequence><xs:attribute name="at" type="xs:string"/></xs:complexType></xs:element>` +
				`<xs:element name="b" type="xs:string"/>`,
			`<known at="v"><a>1</a><b>x</b></known>`,
		},
		{
			"mixed content, a nested complex child carrying the xsi: location hints, and a defaulted empty leaf",
			`<xs:element name="known"><xs:complexType mixed="true"><xs:sequence>` +
				`<xs:element name="c"><xs:complexType><xs:sequence><xs:element name="d" type="xs:int" default="3"/></xs:sequence></xs:complexType></xs:element>` +
				`</xs:sequence></xs:complexType></xs:element>`,
			`<known ` + xsiNS + `>text<c xsi:noNamespaceSchemaLocation="s.xsd"><d/></c>more</known>`,
		},
		{
			"a simple {content type} root carrying a matched attribute",
			`<xs:element name="known"><xs:complexType><xs:simpleContent><xs:extension base="xs:int">` +
				`<xs:attribute name="at" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType></xs:element>`,
			`<known at="v">1</known>`,
		},
		{
			// cvc-elt clause 3 is live only through an xsi:nil, which the root does
			// not carry; the simple-leaf-root gate refuses {nillable} regardless.
			"a {nillable} simple-typed root with character content and no xsi:nil",
			`<xs:element name="known" type="xs:string" nillable="true"/>`, `<known>x</known>`,
		},
		{
			"a {nillable} child with content and no xsi:nil",
			`<xs:element name="known"><xs:complexType><xs:sequence>` +
				`<xs:element name="a" type="xs:int" nillable="true"/></xs:sequence></xs:complexType></xs:element>`,
			`<known><a>1</a></known>`,
		},
		// The three ID-family rows, one per closureReaches site, are admitted
		// because the walk decides cvc-id at every depth and records each item it
		// cannot read (#1857); NOTATION is the one name the sites still refuse.
		{
			"an ID on a child element, binding its parent (element value type)",
			`<xs:element name="known"><xs:complexType><xs:sequence>` +
				`<xs:element name="a" type="xs:ID" maxOccurs="2"/></xs:sequence></xs:complexType></xs:element>`,
			`<known><a>i1</a><a>i2</a></known>`,
		},
		{
			"an IDREFS list resolving to IDs on sibling attributes ({attribute uses})",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="2"><xs:complexType>` +
				`<xs:attribute name="id" type="xs:ID"/><xs:attribute name="refs" type="xs:IDREFS"/>` +
				`</xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			`<known><a id="i1"/><a id="i2" refs="i1 i2"/></known>`,
		},
		{
			// Below the root: the root's own ID value binds its parent, which it
			// has none of, so that binding is empty and cvc-id clause 1 charges.
			"a union with an ID member as a child's simple {content type}",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a"><xs:complexType>` +
				`<xs:simpleContent><xs:extension base="U"><xs:attribute name="at" type="xs:string"/></xs:extension></xs:simpleContent>` +
				`</xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>` +
				`<xs:simpleType name="U"><xs:union memberTypes="xs:int xs:ID"/></xs:simpleType>`,
			`<known><a at="v">i1</a></known>`,
		},
	}
	for _, tc := range cases {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: a clean walk of the assessed-subtree-root shape is valid; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesCvcIDBelowTheRoot is a regression guard for the
// ID-family lift (#1857): with ID and IDREF admitted at depth, a dangling IDREF
// (cvc-id clause 1) and a duplicate ID (clause 2) in the subtree of the
// ·validation root· are still decided INVALID. Both are the walk's charges at
// the root (validate's idTable.charge), so neither row passes through the gate.
func TestInstanceExecutorChargesCvcIDBelowTheRoot(t *testing.T) {
	const idAndRef = `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="2"><xs:complexType>` +
		`<xs:attribute name="id" type="xs:ID"/><xs:attribute name="ref" type="xs:IDREF"/>` +
		`</xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
	exec := newInstanceExec()
	for _, tc := range []struct{ why, instance string }{
		{"a dangling IDREF (cvc-id clause 1)", `<known><a id="i1"/><a ref="i2"/></known>`},
		{"a duplicate ID (cvc-id clause 2)", `<known><a id="i1"/><a id="i1"/></known>`},
	} {
		if !exec(instanceCase(t, idAndRef, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges cvc-id at the validation root; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, idAndRef, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// TestInstanceExecutorDeclinesOutsideAssessedSubtreeRoot pins each condition of
// the assessed-subtree-root gate by name. Every row walks clean — no violation,
// no unevaluated record — and is no leaf root, so it is assessedSubtreeRoot alone
// that keeps the empty Result from reading as "valid": with the row's condition
// removed from the gate, the row passes under its valid expectation and this
// test fails.
func TestInstanceExecutorDeclinesOutsideAssessedSubtreeRoot(t *testing.T) {
	const (
		// aInt declares <known> with one required child <a> of type xs:int, and
		// wantInt is the instance that meets it.
		aInt    = `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType></xs:element>`
		wantInt = `<known><a>1</a></known>`
	)
	exec := newInstanceExec()
	cases := []struct {
		condition  string
		schemaBody string
		instance   string
	}{
		{"DOCTYPE present", aInt, `<!DOCTYPE known><known><a>1</a></known>`},
		{
			"skip-wildcard subtree (cvc-assess-elt clause 3.2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:any processContents="skip"/></xs:sequence></xs:complexType></xs:element>`,
			`<known><b><c/></b></known>`,
		},
		{"xsi:type below the root (cvc-elt clause 4)", aInt, `<known ` + xsiXS + `><a xsi:type="xs:int">1</a></known>`},
		{
			"xsi:nil below the root (cvc-elt clause 3)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int" nillable="true"/></xs:sequence></xs:complexType></xs:element>`,
			`<known ` + xsiNS + `><a xsi:nil="false">1</a></known>`,
		},
		{
			"an abstract declaration below the root (cvc-elt clause 2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element ref="b"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:element name="b" type="xs:string" abstract="true"/>`,
			`<known><b>x</b></known>`,
		},
		{
			"an abstract complex type below the root (cvc-type clause 2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="T"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:complexType name="T" abstract="true"><xs:sequence><xs:element name="c" type="xs:int"/></xs:sequence></xs:complexType>`,
			`<known><a><c>1</c></a></known>`,
		},
		{
			"an abstract complex type at the root (cvc-type clause 2)",
			`<xs:element name="known" type="T"/>` +
				`<xs:complexType name="T" abstract="true"><xs:sequence><xs:element name="c" type="xs:int"/></xs:sequence></xs:complexType>`,
			`<known><c>1</c></known>`,
		},
		{
			"a {type table} below the root (cvc-elt clause 4)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int">` +
				`<xs:alternative type="xs:int"/></xs:element></xs:sequence></xs:complexType></xs:element>`,
			wantInt,
		},
		{
			"identity constraints (cvc-elt clause 6)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType>` +
				`<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="."/></xs:unique></xs:element>`,
			wantInt,
		},
		{
			"a fixed {value constraint} below the root (cvc-elt clause 5.2.2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int" fixed="1"/></xs:sequence></xs:complexType></xs:element>`,
			wantInt,
		},
		// The three NOTATION rows, one per closureReaches site, name a declared
		// notation: the walk checks no NOTATION value against the schema's
		// notations (Datatypes §3.3.19, enumeration-required-notation), so a
		// value naming an undeclared one walks clean too (over027.n01).
		{
			"a NOTATION closure in an element value type below the root",
			notationN + `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="N"/></xs:sequence></xs:complexType></xs:element>`,
			`<known><a>n</a></known>`,
		},
		{
			"a NOTATION closure in a simple {content type}",
			notationN + `<xs:element name="known"><xs:complexType><xs:simpleContent><xs:extension base="N">` +
				`<xs:attribute name="at" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType></xs:element>`,
			`<known at="v">n</known>`,
		},
		{
			"an attribute typed by a NOTATION enumeration below the root",
			notationN + `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a">` +
				`<xs:complexType><xs:attribute name="n" type="N"/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`,
			`<known><a n="n"/></known>`,
		},
		{
			"an attribute matched by an attribute wildcard (cvc-complex-type clause 2.2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence>` +
				`<xs:anyAttribute processContents="lax"/></xs:complexType></xs:element>`,
			`<known foo="1"><a>1</a></known>`,
		},
		{
			"a child attributed to a lax wildcard particle (cvc-complex-type clause 5)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:any processContents="lax"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:element name="b" type="xs:string"/>`,
			`<known><b>x</b></known>`,
		},
		{
			"a child attributed to a strict wildcard particle (cvc-complex-type clause 5)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:any processContents="strict"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:element name="b" type="xs:string"/>`,
			`<known><b>x</b></known>`,
		},
		{
			"a child attributed to the {open content} (cvc-complex-type clause 5)",
			`<xs:element name="known"><xs:complexType><xs:openContent mode="interleave"><xs:any processContents="lax"/></xs:openContent>` +
				`<xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:element name="b" type="xs:string"/>`,
			`<known><b>x</b><a>1</a></known>`,
		},
		{
			"a child attributed through its substitution group head (cvc-accept clause 2.3.2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:element name="h" type="xs:string"/><xs:element name="m" substitutionGroup="h"/>`,
			`<known><m>x</m></known>`,
		},
		{
			"a root with no content, outside the leaf-root shapes",
			`<xs:element name="known"><xs:complexType><xs:attribute name="at" type="xs:string"/></xs:complexType></xs:element>`,
			`<known at="v"/>`,
		},
	}
	for _, tc := range cases {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.condition)
	}
}

// TestInstanceExecutorDeclinesVersionedSubtreeRoot pins the versioned-closure
// condition under this gate: a root WITH content, against a schema document
// carrying an attribute in the versioning namespace, declines (#1002).
func TestInstanceExecutorDeclinesVersionedSubtreeRoot(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "s.xsd")
	instancePath := filepath.Join(dir, "i.xml")
	writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
		` xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">`+
		`<xs:element name="known" vc:minVersion="1.0"><xs:complexType><xs:sequence>`+
		`<xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType></xs:element></xs:schema>`)
	writeFixture(t, instancePath, `<known><a>1</a></known>`)
	declinesBothPolarities(t, newInstanceExec(),
		caseSpec{kind: kindInstance, doc: instancePath, schemaDoc: schemaPath}, "a versioned closure")
}

// TestAssessedSubtreeRootUntypedRoot pins the untyped-element condition at the
// one place the gate can meet it outside a wildcard: a root no top-level
// declaration matches. The walk charges cvc-assess-elt for it first, so the gate
// is called directly, with the declared root as the control.
func TestAssessedSubtreeRootUntypedRoot(t *testing.T) {
	for _, tc := range []struct {
		why, instance string
		want          bool
	}{
		{"a declared root", `<known>x</known>`, true},
		{"an undeclared root", `<unknown>x</unknown>`, false},
	} {
		c := instanceCase(t, knownRoot, tc.instance, true)
		schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
		if err != nil || !decidable {
			t.Fatalf("%s: assembling the schema: decidable %v, err %v", tc.why, decidable, err)
		}
		if got := assessedSubtreeRoot(schema, report, c.doc); got != tc.want {
			t.Errorf("%s: assessedSubtreeRoot = %v, want %v", tc.why, got, tc.want)
		}
	}
}
