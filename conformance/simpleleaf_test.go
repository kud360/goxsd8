package conformance

import (
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// xsiNS binds xsi on an instance root.
const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

// TestInstanceExecutorDecidesSimpleLeafRoot proves the lane's first "valid"
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
		// Datatype Valid holds for every literal against the two ·special·
		// datatypes (Datatypes §4.1.4), so validate decides both (#1788).
		{"an xs:anySimpleType-typed leaf root", `<xs:element name="known" type="xs:anySimpleType"/>`, `<known>x</known>`},
		{"an xs:anyAtomicType-typed leaf root", `<xs:element name="known" type="xs:anyAtomicType"/>`, `<known>x</known>`},
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
			// Empty, because a root WITH content and no xsi:nil is the
			// assessed-subtree-root gate's, which admits {nillable} there.
			"a {nillable} declaration (cvc-elt clause 3, both arms deferred)",
			`<xs:element name="known" type="xs:string" nillable="true"/>`, `<known/>`,
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
		// The three ID- and ENTITY-family rows are EMPTY roots whose ·initial
		// value· a default {value constraint} supplies (cvc-elt clause 5.1): a
		// root with content is the assessed-subtree-root gate's, which admits
		// both families, the walk deciding them (#1857). A union whose ID,
		// IDREF or ENTITY member never becomes the ·validating type· walks
		// clean.
		{
			"a union with an ID member (cvc-elt clause 7, cvc-id), even where another member validates",
			`<xs:element name="known" type="U" default="5"/><xs:simpleType name="U"><xs:union memberTypes="xs:int xs:ID"/></xs:simpleType>`,
			`<known/>`,
		},
		{
			"a list whose {item type definition} has an IDREF member",
			`<xs:element name="known" type="L" default="1 2"/><xs:simpleType name="L"><xs:list itemType="U"/></xs:simpleType>` +
				`<xs:simpleType name="U"><xs:union memberTypes="xs:int xs:IDREF"/></xs:simpleType>`,
			`<known/>`,
		},
		{
			"a union with an ENTITY member, even where another member validates",
			`<xs:element name="known" type="U" default="5"/><xs:simpleType name="U"><xs:union memberTypes="xs:int xs:ENTITY"/></xs:simpleType>`,
			`<known/>`,
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
// the empty violation list as "valid": validate's cvc-elt clause 5.2.2.2.2
// decline over a fixed {value constraint} it cannot compare in
// xs:anySimpleType's value space (#1738 routed it into Unevaluated), and an
// assertions facet, whose {test} validate records and never evaluates.
func TestInstanceExecutorDeclinesUnevaluatedLeaf(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody string }{
		{"a fixed-value comparison withheld over xs:anySimpleType", `<xs:element name="known" type="xs:anySimpleType" fixed="x"/>`},
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
// vc:maxVersion GAP(parser) (#1002) that lets VC/vc006.n1 walk clean. The
// attribute counts in EVERY document the assembly read, not only the root one
// — an <include>d or <import>ed document carrying it declines too (#1789) — and
// under both shape gates.
func TestInstanceExecutorDeclinesVersionedSchema(t *testing.T) {
	const (
		plain     = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`
		versioned = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` +
			` xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">`
		vOther = `<xs:element name="other" type="xs:string" vc:minVersion="1.0"/></xs:schema>`
	)
	exec := newInstanceExec()
	for _, tc := range []struct {
		why, instance string
		// docs are the schema documents written beside the instance, the first
		// one being the case's schemaDoc.
		docs []fixture
	}{
		{
			"a root schema document carrying a versioning-namespace attribute",
			`<known>x</known>`,
			[]fixture{{"s.xsd", versioned + `<xs:element name="known" type="xs:string" vc:minVersion="1.0"/></xs:schema>`}},
		},
		{
			"a root schema document carrying one, under the complex-empty-leaf-root gate",
			`<known/>`,
			[]fixture{{"s.xsd", versioned + `<xs:element name="known" vc:minVersion="1.0"><xs:complexType/></xs:element></xs:schema>`}},
		},
		{
			"an <include>d schema document carrying one, the root document carrying none",
			`<known>x</known>`,
			[]fixture{
				{"s.xsd", plain + `<xs:include schemaLocation="inc.xsd"/>` + knownRoot + `</xs:schema>`},
				{"inc.xsd", versioned + vOther},
			},
		},
		{
			"an <import>ed schema document carrying one, the root document carrying none",
			`<known>x</known>`,
			[]fixture{
				{"s.xsd", plain + `<xs:import namespace="urn:other" schemaLocation="imp.xsd"/>` + knownRoot + `</xs:schema>`},
				{"imp.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:other"` +
					` xmlns:vc="http://www.w3.org/2007/XMLSchema-versioning">` + vOther},
			},
		},
	} {
		dir := t.TempDir()
		for _, f := range tc.docs {
			writeFixture(t, filepath.Join(dir, f.name), f.content)
		}
		instancePath := filepath.Join(dir, "i.xml")
		writeFixture(t, instancePath, tc.instance)
		declinesBothPolarities(t, exec,
			caseSpec{kind: kindInstance, doc: instancePath, schemaDoc: filepath.Join(dir, tc.docs[0].name)}, tc.why)
	}
}

// TestInstanceExecutorDecidesUnversionedComposition is the control for the
// include and import rows of TestInstanceExecutorDeclinesVersionedSchema: the
// same composition with no versioning-namespace attribute anywhere is decided,
// so it is the attribute those rows decline for and not the composition.
func TestInstanceExecutorDecidesUnversionedComposition(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct {
		why       string
		other     fixture
		directive string
	}{
		{
			"an <include>",
			fixture{"inc.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="other" type="xs:string"/></xs:schema>`},
			`<xs:include schemaLocation="inc.xsd"/>`,
		},
		{
			"an <import>",
			fixture{"imp.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:other">` +
				`<xs:element name="other" type="xs:string"/></xs:schema>`},
			`<xs:import namespace="urn:other" schemaLocation="imp.xsd"/>`,
		},
	} {
		dir := t.TempDir()
		schemaPath := filepath.Join(dir, "s.xsd")
		instancePath := filepath.Join(dir, "i.xml")
		writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+tc.directive+knownRoot+`</xs:schema>`)
		writeFixture(t, filepath.Join(dir, tc.other.name), tc.other.content)
		writeFixture(t, instancePath, `<known>x</known>`)
		c := caseSpec{kind: kindInstance, doc: instancePath, schemaDoc: schemaPath, expect: expectValidity(true)}
		if !exec(c).IsPass() {
			t.Errorf("%s with no versioning attribute: the executor must decide the simple leaf root valid", tc.why)
		}
	}
}

// fixture is one file a test writes into its case directory.
type fixture struct{ name, content string }

// TestPeekRootLeafContent pins the instance-side conditions the walk ALSO
// enforces — an element child (cvc-type clause 3.1.2), an attribute outside the
// xsi: four (3.1.1), an xsi:nil (cvc-elt clause 3) — at the peek itself, since
// the gate is a precondition in its own right and not a restatement of those
// charges. A peek error and an abstract declaration are refused too.
func TestPeekRootLeafContent(t *testing.T) {
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
		if _, got := peekRoot(path, leafContent); got != tc.want {
			t.Errorf("%s: peekRoot(leafContent) = %v, want %v", tc.why, got, tc.want)
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
	for _, tc := range []struct {
		local string
		want  bool
	}{
		{"abstract", false},
		{"concrete", true},
	} {
		d, ok := schema.Element(xsd.QName{Local: tc.local})
		if !ok {
			t.Fatalf("no top-level declaration %q", tc.local)
		}
		if got := simpleLeafDeclaration(schema, d); got != tc.want {
			t.Errorf("simpleLeafDeclaration(%s) = %v, want %v", tc.local, got, tc.want)
		}
	}
}

// emptyRoot declares <known> with an anonymous complex type of empty {content
// type} and nothing else: the complex-empty-leaf-root shape's declaration.
const emptyRoot = `<xs:element name="known"><xs:complexType/></xs:element>`

// TestInstanceExecutorDecidesComplexEmptyLeafRoot proves the lane's second
// "valid" observation (#1808): a root of the complex-empty-leaf-root shape whose
// walk charged nothing and recorded nothing unevaluated is decided VALID — it
// agrees with a suite-valid case and disagrees with a suite-invalid one. The
// location hints, namespace declarations, comments and processing instructions
// are what the shape admits besides the bare element, and a named type or one
// derived to an empty {content type} is read off the folded component.
func TestInstanceExecutorDecidesComplexEmptyLeafRoot(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{"an empty-element root", emptyRoot, `<known/>`},
		{"a root with a start and an end tag and nothing between", emptyRoot, `<known></known>`},
		{"a root holding a comment and a processing instruction", emptyRoot, `<known><!--c--><?pi x?></known>`},
		{
			"a root carrying the two xsi: location hints and namespace declarations",
			emptyRoot,
			`<known ` + xsiNS + ` xmlns:p="urn:p" xsi:noNamespaceSchemaLocation="s.xsd" xsi:schemaLocation="urn:p p.xsd"/>`,
		},
		{"a named empty type", `<xs:element name="known" type="E"/><xs:complexType name="E"/>`, `<known/>`},
		{
			"an empty restriction of xs:anyType, extended by nothing",
			`<xs:element name="known" type="F"/>` +
				`<xs:complexType name="E"><xs:complexContent><xs:restriction base="xs:anyType"/></xs:complexContent></xs:complexType>` +
				`<xs:complexType name="F"><xs:complexContent><xs:extension base="E"/></xs:complexContent></xs:complexType>`,
			`<known/>`,
		},
	}
	for _, tc := range cases {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: a clean walk of the complex-empty-leaf-root shape is valid; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorDeclinesOutsideComplexEmptyLeafRoot proves each condition
// of the complex-empty-leaf-root gate that the WALK does not also enforce
// declines on its own. Every row walks clean — no violation, no unevaluated
// record — and is no simple leaf root either, so it is complexEmptyLeafRoot
// alone that keeps the empty Result from reading as "valid".
func TestInstanceExecutorDeclinesOutsideComplexEmptyLeafRoot(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{
			"a {nillable} declaration (cvc-elt clause 3)",
			`<xs:element name="known" nillable="true"><xs:complexType/></xs:element>`, `<known/>`,
		},
		{
			"an {identity-constraint definitions} member (cvc-elt clause 6)",
			`<xs:element name="known"><xs:complexType/><xs:unique name="u">` +
				`<xs:selector xpath="."/><xs:field xpath="@a"/></xs:unique></xs:element>`,
			`<known/>`,
		},
		{
			"a {type table} (cvc-elt clause 4, conditional type assignment)",
			`<xs:element name="known" type="E"><xs:alternative type="E"/></xs:element><xs:complexType name="E"/>`,
			`<known/>`,
		},
		{
			"an element-only {content type} whose particle is emptiable",
			`<xs:element name="known"><xs:complexType><xs:sequence>` +
				`<xs:element name="c" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"a mixed {content type}",
			`<xs:element name="known"><xs:complexType mixed="true"><xs:sequence>` +
				`<xs:element name="c" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"a simple {content type}",
			`<xs:element name="known"><xs:complexType><xs:simpleContent>` +
				`<xs:extension base="xs:string"/></xs:simpleContent></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"an optional {attribute uses} member the root does not carry (cvc-complex-type clause 4)",
			`<xs:element name="known"><xs:complexType><xs:attribute name="a"/></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"an {attribute wildcard}",
			`<xs:element name="known"><xs:complexType><xs:anyAttribute/></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"an xsi:type on the root (cvc-elt clause 4 deferred)",
			`<xs:element name="known" type="E"/><xs:complexType name="E"/>`,
			`<known ` + xsiNS + ` xsi:type="E"/>`,
		},
		{
			"a DOCTYPE, whose DTD could default an attribute the peek does not see",
			emptyRoot, `<!DOCTYPE known><known/>`,
		},
	}
	for _, tc := range cases {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.why)
	}
}

// TestInstanceExecutorDeclinesUnevaluatedComplexEmptyRoot proves a
// complex-empty-leaf-root instance whose governing type carries {assertions}
// declines on the walk's own record (validate's elementAssertions), whatever
// the gate says: TestComplexEmptyDeclaration pins the gate's own refusal.
func TestInstanceExecutorDeclinesUnevaluatedComplexEmptyRoot(t *testing.T) {
	declinesBothPolarities(t, newInstanceExec(),
		instanceCase(t, `<xs:element name="known"><xs:complexType><xs:assert test="true()"/></xs:complexType></xs:element>`, `<known/>`, false),
		"an unevaluated {assertions} member")
}

// TestComplexEmptyDeclaration pins every declaration condition of the
// complex-empty-leaf-root gate against complexEmptyDeclaration directly, one
// top-level declaration per condition plus the control that meets them all.
// Several of them the walk ALSO decides — an abstract declaration or type
// (cvc-elt clause 2, cvc-type clause 2) — or records ({assertions}), so this
// is where the gate's own refusal of them is seen.
func TestComplexEmptyDeclaration(t *testing.T) {
	dir := t.TempDir()
	schemaPath := filepath.Join(dir, "s.xsd")
	writeFixture(t, schemaPath, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+
		`<xs:complexType name="E"/>`+
		`<xs:element name="control" type="E"/>`+
		`<xs:element name="abstract" type="E" abstract="true"/>`+
		`<xs:element name="nillable" type="E" nillable="true"/>`+
		`<xs:element name="identity" type="E"><xs:key name="k"><xs:selector xpath="."/><xs:field xpath="@a"/></xs:key></xs:element>`+
		`<xs:element name="table" type="E"><xs:alternative type="E"/></xs:element>`+
		`<xs:element name="simpleType" type="xs:string"/>`+
		`<xs:complexType name="A" abstract="true"/>`+
		`<xs:element name="abstractType" type="A"/>`+
		`<xs:element name="elementOnly"><xs:complexType><xs:sequence><xs:element name="c" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`+
		`<xs:element name="mixed"><xs:complexType mixed="true"><xs:sequence><xs:element name="c" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`+
		`<xs:element name="simpleContent"><xs:complexType><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType></xs:element>`+
		`<xs:element name="attributeUse"><xs:complexType><xs:attribute name="a"/></xs:complexType></xs:element>`+
		`<xs:element name="attributeWildcard"><xs:complexType><xs:anyAttribute/></xs:complexType></xs:element>`+
		`<xs:element name="assertions"><xs:complexType><xs:assert test="true()"/></xs:complexType></xs:element>`+
		`</xs:schema>`)
	schema, _, decidable, err := assembleCase(strict.New(), schemaPath, nil)
	if err != nil || !decidable {
		t.Fatalf("assembling the schema: decidable %v, err %v", decidable, err)
	}
	for _, tc := range []struct {
		local string
		want  bool
	}{
		{"control", true},
		{"abstract", false},
		{"nillable", false},
		{"identity", false},
		{"table", false},
		{"simpleType", false},
		{"abstractType", false},
		{"elementOnly", false},
		{"mixed", false},
		{"simpleContent", false},
		{"attributeUse", false},
		{"attributeWildcard", false},
		{"assertions", false},
	} {
		d, ok := schema.Element(xsd.QName{Local: tc.local})
		if !ok {
			t.Fatalf("no top-level declaration %q", tc.local)
		}
		if got := complexEmptyDeclaration(schema, d); got != tc.want {
			t.Errorf("complexEmptyDeclaration(%s) = %v, want %v", tc.local, got, tc.want)
		}
	}

	// A {value constraint} on a declaration of an empty {content type} is
	// schema-invalid (e-props-correct clause 2, cos-valid-default clause 2.1), so
	// the assembly rejects it before any gate is read and the condition is pinned
	// on declarations built directly instead, typed by the assembled E. The
	// absent row is the control that construction alone flips nothing.
	constraint := func(kind xsd.ValueConstraintKind) *xsd.ValueConstraint {
		vc := xsd.NewValueConstraint(kind, "", nil, nil)
		return &vc
	}
	for _, tc := range []struct {
		why  string
		vc   *xsd.ValueConstraint
		want bool
	}{
		{"no {value constraint}", nil, true},
		{"a default {value constraint}", constraint(xsd.ValueDefault), false},
		{"a fixed {value constraint}", constraint(xsd.ValueFixed), false},
	} {
		d, err := xsd.NewElementDeclaration(xsderr.Loc{}, xsd.QName{Local: "built"},
			xsd.TypeDefinitionRef{Name: xsd.QName{Local: "E"}}, nil, xsd.NewGlobalScope(), tc.vc,
			false, nil, nil, nil, false, nil)
		if err != nil {
			t.Fatalf("%s: building the declaration: %v", tc.why, err)
		}
		if got := complexEmptyDeclaration(schema, d); got != tc.want {
			t.Errorf("complexEmptyDeclaration with %s = %v, want %v", tc.why, got, tc.want)
		}
	}
}

// TestPeekRootEmptyContent pins the instance-side conditions of the
// complex-empty-leaf-root gate at peekRoot with emptyContent. The walk ALSO
// charges most of them against an empty {content type} — a child or character
// data (cvc-complex-type clause 1.1), an attribute (clause 2), an xsi:nil
// (cvc-elt clause 3.1) — so the peek is pinned here in its own right, and
// white space is refused where leafContent would admit it.
func TestPeekRootEmptyContent(t *testing.T) {
	dir := t.TempDir()
	for _, tc := range []struct {
		why, instance string
		want          bool
	}{
		{"an empty-element tag", `<known/>`, true},
		{"a start and an end tag", `<known></known>`, true},
		{"a comment and a processing instruction", `<known><!--c--><?pi x?></known>`, true},
		{"the two xsi: location hints", `<known ` + xsiNS + ` xsi:noNamespaceSchemaLocation="s.xsd"/>`, true},
		{"an element child", `<known><a/></known>`, false},
		{"character data", `<known>x</known>`, false},
		{"white space alone", `<known> </known>`, false},
		{"an empty CDATA section", `<known><![CDATA[]]></known>`, false},
		{"an attribute outside the xsi: four", `<known foo="1"/>`, false},
		{"an xsi:nil", `<known ` + xsiNS + ` xsi:nil="false"/>`, false},
		{"an xsi:type", `<known ` + xsiNS + ` xsi:type="E"/>`, false},
		{"a DOCTYPE", `<!DOCTYPE known><known/>`, false},
		{"a malformed document", `<known>`, false},
	} {
		path := filepath.Join(dir, "i.xml")
		writeFixture(t, path, tc.instance)
		if _, got := peekRoot(path, emptyContent); got != tc.want {
			t.Errorf("%s: peekRoot(emptyContent) = %v, want %v", tc.why, got, tc.want)
		}
	}
}

// TestComplexEmptyLeafRootReadsEmptyContent pins that complexEmptyLeafRoot
// peeks with emptyContent and not leafContent: white space under an empty
// {content type} is charged by the walk (cvc-complex-type clause 1.1), so no
// executor row can see which peek the gate uses, and the gate is called
// directly instead, with the bare root as the control.
func TestComplexEmptyLeafRootReadsEmptyContent(t *testing.T) {
	for _, tc := range []struct {
		why, instance string
		want          bool
	}{
		{"an empty-element root", `<known/>`, true},
		{"a root holding white space alone", `<known> </known>`, false},
	} {
		c := instanceCase(t, emptyRoot, tc.instance, true)
		schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
		if err != nil || !decidable {
			t.Fatalf("%s: assembling the schema: decidable %v, err %v", tc.why, decidable, err)
		}
		if got := complexEmptyLeafRoot(schema, report, c.doc); got != tc.want {
			t.Errorf("%s: complexEmptyLeafRoot = %v, want %v", tc.why, got, tc.want)
		}
	}
}
