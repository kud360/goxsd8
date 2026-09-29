package conformance

import (
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
)

// xsiNS binds xsi on an instance root.
const xsiNS = `xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"`

// xsiXS binds xsi and xs on an instance root, for an xsi:type naming a builtin.
const xsiXS = xsiNS + ` xmlns:xs="http://www.w3.org/2001/XMLSchema"`

// notationN declares the notation n and the simple type N, a NOTATION
// enumeration admitting it alone (enumeration-required-notation).
const notationN = `<xs:notation name="n" public="p"/>` +
	`<xs:simpleType name="N"><xs:restriction base="xs:NOTATION"><xs:enumeration value="n"/></xs:restriction></xs:simpleType>`

// TestInstanceExecutorDecidesAssessedSubtreeRoot proves the lane's "valid"
// observation (#1738, #1841): a root WITH content whose every element and
// attribute meets the assessed-subtree-root gate, and whose walk charged
// nothing and recorded nothing unevaluated, is decided VALID — it agrees with a
// suite-valid case and disagrees with a suite-invalid one.
// TestInstanceExecutorDecidesContentLessRoot holds the roots with no content.
func TestInstanceExecutorDecidesAssessedSubtreeRoot(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{"a string-typed leaf root", knownRoot, `<known>x</known>`},
		{
			"a leaf root carrying the two xsi: location hints and namespace declarations",
			knownRoot,
			`<known ` + xsiNS + ` xmlns:p="urn:p" xsi:noNamespaceSchemaLocation="s.xsd" xsi:schemaLocation="urn:p p.xsd">x</known>`,
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
			// cvc-elt clause 3.2.1: the root carries no xsi:nil.
			"a {nillable} simple-typed root with character content and no xsi:nil",
			`<xs:element name="known" type="xs:string" nillable="true"/>`, `<known>x</known>`,
		},
		{
			"a {nillable} child with content and no xsi:nil",
			`<xs:element name="known"><xs:complexType><xs:sequence>` +
				`<xs:element name="a" type="xs:int" nillable="true"/></xs:sequence></xs:complexType></xs:element>`,
			`<known><a>1</a></known>`,
		},
		// The three ID-family rows, one per closureReaches site, are admitted on
		// the terms of the third shape's cvc-elt clause 7 bullet (instance.go)
		// (#1857).
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

// emptyRoot declares <known> with an anonymous complex type of empty {content
// type} and nothing else.
const emptyRoot = `<xs:element name="known"><xs:complexType/></xs:element>`

// TestInstanceExecutorDecidesContentLessRoot proves the assessed-subtree-root
// gate decides a root with NO element or character [[children]] (#1855) on
// the same terms as one with content: the children clauses of key-sva and
// sic-e-outcome are vacuous, and each clause the empty content leaves live is
// the walk's to decide, which each row names. Every row agrees with a
// suite-valid case and disagrees with a suite-invalid one;
// TestInstanceExecutorChargesContentLessRoot holds the invalid counterparts.
func TestInstanceExecutorDecidesContentLessRoot(t *testing.T) {
	exec := newInstanceExec()
	cases := []struct {
		why        string
		schemaBody string
		instance   string
	}{
		{"an empty string-typed root (cvc-type clause 3.1.3 over \"\")", knownRoot, `<known/>`},
		{
			"an empty root whose default is a value of its type (cvc-elt clause 5.1.2)",
			`<xs:element name="known" type="xs:int" default="7"/>`, `<known/>`,
		},
		{
			"a {nillable} simple-typed root with no xsi:nil (cvc-elt clause 3.2.1)",
			`<xs:element name="known" type="xs:string" nillable="true"/>`, `<known/>`,
		},
		// The ID- and ENTITY-family rows are the walk's at the root as at depth
		// (#1857): the default's ·validating type· is xs:int, so no ID or IDREF
		// binding and no ENTITY value arises (cvc-elt clauses 5.1.2 and 7).
		{
			"a union with an ID member, another member validating the default",
			`<xs:element name="known" type="U" default="5"/><xs:simpleType name="U"><xs:union memberTypes="xs:int xs:ID"/></xs:simpleType>`,
			`<known/>`,
		},
		{
			"a list whose {item type definition} has an IDREF member, another member validating the default",
			`<xs:element name="known" type="L" default="1 2"/><xs:simpleType name="L"><xs:list itemType="U"/></xs:simpleType>` +
				`<xs:simpleType name="U"><xs:union memberTypes="xs:int xs:IDREF"/></xs:simpleType>`,
			`<known/>`,
		},
		{
			"a union with an ENTITY member, another member validating the default",
			`<xs:element name="known" type="U" default="5"/><xs:simpleType name="U"><xs:union memberTypes="xs:int xs:ENTITY"/></xs:simpleType>`,
			`<known/>`,
		},
		{"an empty-element root of an empty {content type} (cvc-complex-type clause 1.1)", emptyRoot, `<known/>`},
		{"a root with a start and an end tag and nothing between", emptyRoot, `<known></known>`},
		{"a root holding a comment and a processing instruction", emptyRoot, `<known><!--c--><?pi x?></known>`},
		{
			"a content-less root carrying the two xsi: location hints and namespace declarations",
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
		{
			"a {nillable} root of an empty {content type} with no xsi:nil (cvc-elt clause 3.2.1)",
			`<xs:element name="known" nillable="true"><xs:complexType/></xs:element>`, `<known/>`,
		},
		{
			"an element-only {content type} whose particle is emptiable (cvc-complex-type clause 1.4)",
			`<xs:element name="known"><xs:complexType><xs:sequence>` +
				`<xs:element name="c" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"a mixed {content type} whose particle is emptiable (cvc-complex-type clause 1.4)",
			`<xs:element name="known"><xs:complexType mixed="true"><xs:sequence>` +
				`<xs:element name="c" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"a simple {content type} admitting \"\" (cvc-complex-type clause 1.2)",
			`<xs:element name="known"><xs:complexType><xs:simpleContent>` +
				`<xs:extension base="xs:string"/></xs:simpleContent></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"an optional {attribute uses} member the root does not carry (cvc-complex-type clauses 3 and 4)",
			`<xs:element name="known"><xs:complexType><xs:attribute name="a"/></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"an {attribute wildcard} no attribute is ·attributed to· (cvc-complex-type clause 2)",
			`<xs:element name="known"><xs:complexType><xs:anyAttribute/></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"a content-less root carrying an attribute its use matches (cvc-complex-type clause 2.1)",
			`<xs:element name="known"><xs:complexType><xs:attribute name="at" type="xs:string"/></xs:complexType></xs:element>`,
			`<known at="v"/>`,
		},
	}
	for _, tc := range cases {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: a clean walk of a content-less assessed subtree root is valid; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesContentLessRoot proves the walk, not the gate,
// decides what a content-less root's clauses leave live: each row is a
// content-less root the walk charges, so it is decided INVALID, never read as
// an empty Result.
func TestInstanceExecutorChargesContentLessRoot(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"an empty int-typed root (cvc-type clause 3.1.3 over \"\")", `<xs:element name="known" type="xs:int"/>`, `<known/>`},
		{
			"a required child missing (cvc-complex-type clause 1.4, cvc-complex-content)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="c"/></xs:sequence></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{
			"a {required} attribute use the root does not carry (cvc-complex-type clause 3)",
			`<xs:element name="known"><xs:complexType><xs:attribute name="a" use="required"/></xs:complexType></xs:element>`,
			`<known/>`,
		},
		{"white space under an empty {content type} (cvc-complex-type clause 1.1)", emptyRoot, `<known> </known>`},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges the root; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
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
// no unevaluated record — so it is assessedSubtreeRoot alone that keeps the
// empty Result from reading as "valid": with the row's condition removed from
// the gate, the row passes under its valid expectation and this test fails.
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
		{"DOCTYPE present on a content-less root", emptyRoot, `<!DOCTYPE known><known/>`},
		{
			"xsi:type on the root (cvc-elt clause 4)",
			knownRoot, `<known ` + xsiXS + ` xsi:type="xs:string">x</known>`,
		},
		{
			"xsi:type on a content-less root (cvc-elt clause 4)",
			`<xs:element name="known" type="E"/><xs:complexType name="E"/>`, `<known ` + xsiNS + ` xsi:type="E"/>`,
		},
		{"a fixed {value constraint} on the root (cvc-elt clause 5.2.2)", `<xs:element name="known" type="xs:string" fixed="x"/>`, `<known>x</known>`},
		{
			"identity constraints on the root (cvc-elt clause 6)",
			`<xs:element name="known" type="xs:string"><xs:unique name="u"><xs:selector xpath="."/><xs:field xpath="."/></xs:unique></xs:element>`,
			`<known>x</known>`,
		},
		{
			"identity constraints on a content-less root (cvc-elt clause 6)",
			`<xs:element name="known"><xs:complexType/><xs:unique name="u"><xs:selector xpath="."/><xs:field xpath="@a"/></xs:unique></xs:element>`,
			`<known/>`,
		},
		{
			"a {type table} on the root (cvc-elt clause 4)",
			`<xs:element name="known" type="xs:string"><xs:alternative type="xs:string"/></xs:element>`,
			`<known>x</known>`,
		},
		{
			"a {type table} on a content-less root (cvc-elt clause 4)",
			`<xs:element name="known" type="E"><xs:alternative type="E"/></xs:element><xs:complexType name="E"/>`,
			`<known/>`,
		},
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
		// notation; a value naming an undeclared one would be refused all the
		// same, for the reason walkUnrecorded's doc gives (subtreeroot.go).
		{"a NOTATION closure in the root's value type", notationN + `<xs:element name="known" type="N"/>`, `<known>n</known>`},
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
	}
	for _, tc := range cases {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.condition)
	}
}

// TestInstanceExecutorDeclinesUnevaluatedRoot proves a root the gate admits
// whose walk RECORDED a check it did not perform declines rather than reading
// the empty violation list as "valid": validate's cvc-elt clause 5.2.2.2.2
// decline over a fixed {value constraint} it cannot compare in
// xs:anySimpleType's value space (#1738 routed it into Unevaluated), an
// assertions facet, whose {test} validate records and never evaluates, and a
// complex type's {assertions} (cvc-complex-type clause 6), which validate's
// elementAssertions records the same way.
func TestInstanceExecutorDeclinesUnevaluatedRoot(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"a fixed-value comparison withheld over xs:anySimpleType", `<xs:element name="known" type="xs:anySimpleType" fixed="x"/>`, `<known>x</known>`},
		{
			"an unevaluated assertions facet",
			`<xs:element name="known" type="A"/><xs:simpleType name="A"><xs:restriction base="xs:string">` +
				`<xs:assertion test="true()"/></xs:restriction></xs:simpleType>`,
			`<known>x</known>`,
		},
		{
			"an unevaluated {assertions} member on a content-less root",
			`<xs:element name="known"><xs:complexType><xs:assert test="true()"/></xs:complexType></xs:element>`,
			`<known/>`,
		},
	} {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.why)
	}
}

// TestInstanceExecutorDeclinesVersionedSchema proves a schema document carrying
// any attribute in the versioning namespace declines the valid observation,
// even one §4.2.2 retains the element under: the gate is a superset of the
// vc:maxVersion GAP(parser) (#1002) that lets VC/vc006.n1 walk clean. The
// attribute counts in EVERY document the assembly read, not only the root one
// — an <include>d or <import>ed document carrying it declines too (#1789) —
// whatever the root holds.
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
			"a root schema document carrying one, under a content-less root",
			`<known/>`,
			[]fixture{{"s.xsd", versioned + `<xs:element name="known" vc:minVersion="1.0"><xs:complexType/></xs:element></xs:schema>`}},
		},
		{
			"a root schema document carrying one, under a root with an element child",
			`<known><a>1</a></known>`,
			[]fixture{{"s.xsd", versioned + `<xs:element name="known" vc:minVersion="1.0"><xs:complexType><xs:sequence>` +
				`<xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType></xs:element></xs:schema>`}},
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
			t.Errorf("%s with no versioning attribute: the executor must decide the root valid", tc.why)
		}
	}
}

// fixture is one file a test writes into its case directory.
type fixture struct{ name, content string }

// TestAssessedSubtreeRootRootConditions pins the root conditions the walk
// charges first — an undeclared root (cvc-assess-elt), an abstract declaration
// (cvc-elt clause 2), an element child of a simple type (cvc-type clause
// 3.1.2), an attribute outside the xsi: four (3.1.1), an xsi:nil (cvc-elt
// clause 3) — at the gate itself, since it is a precondition in its own right
// and not a restatement of those charges. No executor row can see them, so the
// gate is called directly, with the declared root, with content and without,
// as the controls. A decoder error is refused too.
func TestAssessedSubtreeRootRootConditions(t *testing.T) {
	const schemaBody = knownRoot + `<xs:element name="abstract" type="xs:string" abstract="true"/>`
	for _, tc := range []struct {
		why, instance string
		want          bool
	}{
		{"a declared root", `<known>x</known>`, true},
		{"a declared content-less root", `<known/>`, true},
		{"an undeclared root", `<unknown>x</unknown>`, false},
		{"an abstract declaration", `<abstract>x</abstract>`, false},
		{"an element child of a simple type", `<known><a/></known>`, false},
		{"an attribute outside the xsi: four", `<known foo="1">x</known>`, false},
		{"an xsi:nil", `<known ` + xsiNS + ` xsi:nil="false">x</known>`, false},
		{"a malformed document", `<known>`, false},
	} {
		c := instanceCase(t, schemaBody, tc.instance, true)
		schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
		if err != nil || !decidable {
			t.Fatalf("%s: assembling the schema: decidable %v, err %v", tc.why, decidable, err)
		}
		if got := assessedSubtreeRoot(schema, report, c.doc); got != tc.want {
			t.Errorf("%s: assessedSubtreeRoot = %v, want %v", tc.why, got, tc.want)
		}
	}
}
