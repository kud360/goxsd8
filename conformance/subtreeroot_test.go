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
		// cvc-elt clause 6 is the walk's at every depth (#1858).
		{
			"a satisfied unique on the root over its own value (cvc-identity-constraint clause 4.1)",
			`<xs:element name="known" type="xs:string"><xs:unique name="u"><xs:selector xpath="."/><xs:field xpath="."/></xs:unique></xs:element>`,
			`<known>x</known>`,
		},
		{
			"a satisfied unique over child elements (cvc-identity-constraint clause 4.1)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int" maxOccurs="2"/></xs:sequence></xs:complexType>` +
				`<xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="."/></xs:unique></xs:element>`,
			`<known><a>1</a><a>2</a></known>`,
		},
		{
			"a satisfied key over child elements (cvc-identity-constraint clause 4.2)",
			keyAndRef, `<known><a>1</a><a>2</a></known>`,
		},
		{
			"a satisfied keyref over child elements (cvc-identity-constraint clause 4.3)",
			keyAndRef, `<known><a>1</a><a>2</a><b>2</b><b>1</b></known>`,
		},
		{
			"a satisfied key declared on an element below the root",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="c" maxOccurs="2"><xs:complexType><xs:sequence>` +
				`<xs:element name="d" type="xs:int" maxOccurs="2"/></xs:sequence></xs:complexType>` +
				`<xs:key name="k"><xs:selector xpath="d"/><xs:field xpath="."/></xs:key></xs:element></xs:sequence></xs:complexType></xs:element>`,
			`<known><c><d>1</d><d>2</d></c><c><d>1</d></c></known>`,
		},
		{
			// Present on every target, the attribute is no ·defaulted attribute·.
			"a key over an attribute whose use carries a default, the attribute present",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="e" maxOccurs="2"><xs:complexType>` +
				`<xs:attribute name="att" type="xs:string" default="a"/></xs:complexType></xs:element></xs:sequence></xs:complexType>` +
				`<xs:key name="k"><xs:selector xpath="e"/><xs:field xpath="@att"/></xs:key></xs:element>`,
			`<known><e att="1"/><e att="2"/></known>`,
		},
		{
			// The refusal of a ·defaulted attribute· is scoped to the subtree of
			// the declaration carrying the identity constraint.
			"a ·defaulted attribute· outside every identity-constrained subtree",
			`<xs:element name="known"><xs:complexType><xs:sequence>` +
				`<xs:element name="c"><xs:complexType><xs:sequence><xs:element name="d" type="xs:int"/></xs:sequence></xs:complexType>` +
				`<xs:unique name="u"><xs:selector xpath="d"/><xs:field xpath="."/></xs:unique></xs:element>` +
				`<xs:element name="e"><xs:complexType><xs:attribute name="att" type="xs:string" default="a"/></xs:complexType></xs:element>` +
				`</xs:sequence></xs:complexType></xs:element>`,
			`<known><c><d>1</d></c><e/></known>`,
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

// keyAndRef declares <known> with int children <a>, a key k over them, and int
// children <b>, a keyref r over them referring to k.
const keyAndRef = `<xs:element name="known"><xs:complexType><xs:sequence>` +
	`<xs:element name="a" type="xs:int" maxOccurs="2"/><xs:element name="b" type="xs:int" minOccurs="0" maxOccurs="2"/>` +
	`</xs:sequence></xs:complexType>` +
	`<xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="."/></xs:key>` +
	`<xs:keyref name="r" refer="k"><xs:selector xpath="b"/><xs:field xpath="."/></xs:keyref></xs:element>`

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
			// The field selects nothing, so the root's ·key-sequence· is short
			// and it is outside the ·qualified node set· (§3.11.4 clause 4.1).
			"a unique on a content-less root whose field selects no node (cvc-identity-constraint clause 4.1)",
			`<xs:element name="known"><xs:complexType/><xs:unique name="u"><xs:selector xpath="."/><xs:field xpath="@a"/></xs:unique></xs:element>`,
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

// TestInstanceExecutorChargesIdentityConstraintBelowTheRoot is a regression
// guard for the identity-constraint lift (#1858): with {identity-constraint
// definitions} admitted, a duplicate key (cvc-identity-constraint clause 4.2.2)
// and a dangling keyref (clause 4.3) over the root's children are still decided
// INVALID. Both are the walk's charges (validate's icFrame.duplicates and
// icCheck.keyrefs), so neither row passes through the gate.
func TestInstanceExecutorChargesIdentityConstraintBelowTheRoot(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, instance string }{
		{"a duplicate key (cvc-identity-constraint clause 4.2.2)", `<known><a>1</a><a>1</a></known>`},
		{"a dangling keyref (cvc-identity-constraint clause 4.3)", `<known><a>1</a><a>2</a><b>3</b></known>`},
	} {
		if !exec(instanceCase(t, keyAndRef, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges cvc-identity-constraint; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, keyAndRef, tc.instance, true)).IsPass() {
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
		{"a fixed {value constraint} on the root (cvc-elt clause 5.2.2)", `<xs:element name="known" type="xs:string" fixed="x"/>`, `<known>x</known>`},
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
			// idZ011_a's shape: both <e> carry the default "a", so the unique is
			// violated (§3.11.4 clause 3's Note), which validate does not see.
			"a {fields} path selecting a ·defaulted attribute· (cvc-identity-constraint clause 3)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="e" maxOccurs="2"><xs:complexType>` +
				`<xs:attribute name="att" type="xs:string" default="a"/></xs:complexType></xs:element></xs:sequence></xs:complexType>` +
				`<xs:unique name="u"><xs:selector xpath="e"/><xs:field xpath="@att"/></xs:unique></xs:element>`,
			`<known><e/><e/></known>`,
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
			// A guard, not a charged row: the gate refused this shape before #1860
			// too. The walk charges nothing and records nothing for it, a reading
			// the suite does not share (#1912).
			"a strict {attribute wildcard}'s attribute whose name resolves no declaration (cvc-complex-type clause 2.2)",
			wildcardKnown("strict"), `<known foo="1"><a>1</a></known>`,
		},
		{
			"a lax {attribute wildcard}'s attribute resolving to a declaration typed by a NOTATION enumeration",
			notationN + `<xs:attribute name="n" type="N"/>` + wildcardKnown("lax"), `<known n="n"><a>1</a></known>`,
		},
		// The wildcard-child refusals (#1931): subtreeGate.resolvedChild's.
		{"a lax wildcard particle's child resolving no declaration (#1911)", wildcardChild("lax"), `<known><u>x</u></known>`},
		{"an {open content} child resolving no declaration", openChild, `<known><u>x</u><a>1</a></known>`},
		{
			// Strictly assessed against N through the xsi:type, the child takes
			// the walk's e-validity clause 1.1.3 charge away.
			"a strict wildcard particle's child resolving no declaration, typed by an xsi:type naming a NOTATION enumeration",
			notationN + wildcardChild("strict"), `<known ` + xsiNS + `><u xsi:type="N">n</u></known>`,
		},
		{
			// ·locally declared type· key-ldt-elem case 2, through a <group ref>.
			"a lax wildcard particle's child whose name a local declaration in the parent's content model carries (cvc-complex-type clause 5)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:any processContents="lax"/><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:group name="g"><xs:sequence><xs:element name="b" type="xs:string" minOccurs="0"/></xs:sequence></xs:group>` +
				`<xs:element name="b" type="xs:string"/>`,
			`<known><b>x</b></known>`,
		},
		{
			// key-ldt-elem case 3: R's content model contains no b, its base's does.
			"a strict wildcard particle's child whose name the restricted base's content model declares (cvc-complex-type clause 5)",
			`<xs:complexType name="B"><xs:choice><xs:element name="b" type="xs:string"/><xs:any processContents="strict"/></xs:choice></xs:complexType>` +
				`<xs:complexType name="R"><xs:complexContent><xs:restriction base="B"><xs:sequence><xs:any processContents="strict"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>` +
				`<xs:element name="known" type="R"/><xs:element name="b" type="xs:string"/>`,
			`<known><b>x</b></known>`,
		},
		{
			// key-impl-cont: the parent's content model contains the head h, whose
			// ·substitution group· holds m.
			"a strict wildcard particle's child whose substitution group head the parent's content model contains (cvc-complex-type clause 5)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:any processContents="strict"/><xs:element ref="h" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:element name="h" type="xs:string"/><xs:element name="m" substitutionGroup="h"/>`,
			`<known><m>x</m></known>`,
		},
		{
			// key-ldt-att case 3: the restriction prohibits its base's use of ta.
			"a strict {attribute wildcard}'s attribute whose name the restricted base's attribute uses declare (cvc-complex-type clause 5)",
			`<xs:complexType name="B"><xs:attribute ref="ta"/><xs:anyAttribute processContents="strict"/></xs:complexType>` +
				`<xs:complexType name="R"><xs:complexContent><xs:restriction base="B">` +
				`<xs:attribute ref="ta" use="prohibited"/><xs:anyAttribute processContents="strict"/></xs:restriction></xs:complexContent></xs:complexType>` +
				`<xs:element name="known" type="R"/><xs:attribute name="ta" type="xs:int"/>`,
			`<known ta="1"/>`,
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

// wildcardKnown declares <known> with one required child <a> of type xs:int
// and an {attribute wildcard} of {process contents} pc, beside top-level
// attribute declarations ta of type xs:int and tf fixed to 1.
func wildcardKnown(pc string) string {
	return `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence>` +
		`<xs:anyAttribute processContents="` + pc + `"/></xs:complexType></xs:element>` +
		`<xs:attribute name="ta" type="xs:int"/><xs:attribute name="tf" type="xs:int" fixed="1"/>`
}

// TestInstanceExecutorDecidesWildcardAttribute proves the gate admits an
// attribute ·attributed to· the {attribute wildcard} (cvc-complex-type clause
// 2.2) wherever the walk decides it (#1860): ·skipped· under skip, assessed
// against the top-level declaration its name ·resolves· to under lax and
// strict (key-sva clause 2.1, cvc-attribute clauses 3 and 4), and not
// assessed under lax where it resolves to none (key-sva clause 2.2). Every row
// is refused with subtreeGate.wildcardAttribute answering false.
func TestInstanceExecutorDecidesWildcardAttribute(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, pc, instance string }{
		{"skip, a name resolving no declaration", "skip", `<known foo="x"><a>1</a></known>`},
		// Assessed, "x" would be charged against ta's xs:int.
		{"skip, a name resolving a declaration the value is not valid against", "skip", `<known ta="x"><a>1</a></known>`},
		{"lax, a name resolving a declaration", "lax", `<known ta="1"><a>1</a></known>`},
		{"lax, a name resolving a fixed declaration, the value agreeing", "lax", `<known tf="01"><a>1</a></known>`},
		{"lax, a name resolving no declaration", "lax", `<known foo="x"><a>1</a></known>`},
		{"strict, a name resolving a declaration", "strict", `<known ta="1"><a>1</a></known>`},
	} {
		schemaBody := wildcardKnown(tc.pc)
		if !exec(instanceCase(t, schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the walk decides the wildcard attribute and the gate admits it; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesWildcardAttribute is a regression guard for the
// wildcard-attribute lift (#1860): the walk charges each row (validate's
// walk.unmatchedAttribute and walk.wildcardAttribute), so each is decided
// INVALID whatever the gate answers, and none passes through it.
func TestInstanceExecutorChargesWildcardAttribute(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"lax, a value not valid against the resolved declaration (cvc-attribute clause 3)", wildcardKnown("lax"), `<known ta="x"><a>1</a></known>`},
		{"strict, a value not valid against the resolved declaration (cvc-attribute clause 3)", wildcardKnown("strict"), `<known ta="x"><a>1</a></known>`},
		{"strict, a value disagreeing with the resolved declaration's fixed one (cvc-attribute clause 4)", wildcardKnown("strict"), `<known tf="2"><a>1</a></known>`},
		{
			"a name the {attribute wildcard} does not admit (cvc-complex-type clause 2.2.2)",
			`<xs:element name="known"><xs:complexType><xs:anyAttribute namespace="urn:other" processContents="skip"/></xs:complexType></xs:element>`,
			`<known foo="x"/>`,
		},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges the attribute; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// TestAssessedSubtreeRootUnadmittedAttribute pins, at the gate itself, the
// refusal of an attribute the {attribute wildcard} does not admit (cvc-wildcard,
// §3.10.4.1), which no executor row can see: the walk charges cvc-complex-type
// clause 2.2.2 for it first. The admitted name is the control.
func TestAssessedSubtreeRootUnadmittedAttribute(t *testing.T) {
	const schemaBody = `<xs:element name="known"><xs:complexType>` +
		`<xs:anyAttribute namespace="urn:other" processContents="skip"/></xs:complexType></xs:element>`
	for _, tc := range []struct {
		why, instance string
		want          bool
	}{
		{"a name the wildcard admits", `<known xmlns:o="urn:other" o:foo="x"/>`, true},
		{"a name the wildcard does not admit", `<known foo="x"/>`, false},
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

// wildcardChild declares <known>, whose content model is one wildcard particle
// of {process contents} pc and declares no element, beside the top-level
// declarations b of type xs:int and i of type xs:ID. Its ·locally declared
// type· is ·absent· for every child (key-ldt-elem), as for the NIST2004-01-14
// wrapper <out>.
func wildcardChild(pc string) string {
	return `<xs:element name="known"><xs:complexType><xs:sequence><xs:any processContents="` + pc + `"/></xs:sequence></xs:complexType></xs:element>` +
		`<xs:element name="b" type="xs:int"/><xs:element name="i" type="xs:ID"/>`
}

// openChild declares <known> with one required child <a> of type xs:int and a
// lax interleave {open content}, beside the top-level declaration b of type
// xs:int, which the content model does not declare.
const openChild = `<xs:element name="known"><xs:complexType><xs:openContent mode="interleave"><xs:any processContents="lax"/></xs:openContent>` +
	`<xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType></xs:element>` +
	`<xs:element name="b" type="xs:int"/>`

// TestInstanceExecutorDecidesWildcardChild proves the gate admits a child
// ·attributed to· a strict or lax ·wildcard particle· or to the {open content}
// wherever its name ·resolves· to a top-level declaration and its ·locally
// declared type· is ·absent·, which makes cvc-complex-type clause 5 vacuous
// (#1931). Each row walks clean against the resolved declaration, and each is
// refused with subtreeGate.resolvedChild answering false for a resolved name,
// or with child answering false for the {open content} arm.
func TestInstanceExecutorDecidesWildcardChild(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"strict, a child resolving a declaration", wildcardChild("strict"), `<known><b>1</b></known>`},
		{"lax, a child resolving a declaration", wildcardChild("lax"), `<known><b>1</b></known>`},
		// NIST2004-01-14's shape: an ID-typed child binding the wrapper.
		{"strict, a child resolving an xs:ID declaration", wildcardChild("strict"), `<known><i>i1</i></known>`},
		{"an {open content} child resolving a declaration", openChild, `<known><b>1</b><a>1</a></known>`},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the walk decides the wildcard child and the gate admits it; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesWildcardChild is a regression guard for the
// wildcard-child lift (#1931): the walk charges each row, so each is decided
// INVALID whatever the gate answers.
func TestInstanceExecutorChargesWildcardChild(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"strict, a value not valid against the resolved declaration (cvc-type clause 3.1.3)", wildcardChild("strict"), `<known><b>x</b></known>`},
		{"an {open content} child's value not valid against the resolved declaration", openChild, `<known><b>x</b><a>1</a></known>`},
		{"strict, a child resolving no declaration (e-validity clause 1.1.3)", wildcardChild("strict"), `<known><u>x</u></known>`},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges the child; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// TestAssessedSubtreeRootUnresolvedStrictChild pins, at the gate itself, its
// answer for a strict ·wildcard particle·'s child resolving no declaration,
// which no executor row can see: the walk charges e-validity clause 1.1.3 for
// it first (TestInstanceExecutorChargesWildcardChild). With no xsi:type the gate
// admits it, its subtree unread; the lax row is the refused control.
func TestAssessedSubtreeRootUnresolvedStrictChild(t *testing.T) {
	for _, tc := range []struct {
		why, pc string
		want    bool
	}{
		{"strict, a child resolving no declaration", "strict", true},
		{"lax, a child resolving no declaration", "lax", false},
	} {
		c := instanceCase(t, wildcardChild(tc.pc), `<known><u><v/></u></known>`, true)
		schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
		if err != nil || !decidable {
			t.Fatalf("%s: assembling the schema: decidable %v, err %v", tc.why, decidable, err)
		}
		if got := assessedSubtreeRoot(schema, report, c.doc); got != tc.want {
			t.Errorf("%s: assessedSubtreeRoot = %v, want %v", tc.why, got, tc.want)
		}
	}
}

// xsiTypes declares the types the xsi:type rows name: CA, an element-only type
// of one <x>; ECA, CA extended by a <z>, which CA does not admit; and CI, a
// simple {content type} extending xs:int.
const xsiTypes = `<xs:complexType name="CA"><xs:sequence><xs:element name="x" type="xs:int"/></xs:sequence></xs:complexType>` +
	`<xs:complexType name="ECA"><xs:complexContent><xs:extension base="CA"><xs:sequence>` +
	`<xs:element name="z" type="xs:int"/></xs:sequence></xs:extension></xs:complexContent></xs:complexType>` +
	`<xs:complexType name="CI"><xs:simpleContent><xs:extension base="xs:int"/></xs:simpleContent></xs:complexType>`

// xsiChild declares <known> with the one child declaration a, beside xsiTypes.
func xsiChild(a string) string {
	return `<xs:element name="known"><xs:complexType><xs:sequence>` + a + `</xs:sequence></xs:complexType></xs:element>` + xsiTypes
}

// xsiKnown is an instance <known> binding xsi and xs, holding child.
func xsiKnown(child string) string {
	return `<known ` + xsiXS + `>` + child + `</known>`
}

// TestInstanceExecutorDecidesXsiType proves the gate admits an xsi:type whose
// override the walk decides (cvc-elt clause 4, §3.3.4.2 key-overrides) and
// follows the ·instance-specified type definition· below it
// (key-governing-type-elem clause 3, #1859). Each row names the gate condition
// that, removed or inverted, turns it into a decline.
func TestInstanceExecutorDecidesXsiType(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"xsi:type naming the root's declared type", knownRoot, `<known ` + xsiXS + ` xsi:type="xs:string">x</known>`},
		{
			"xsi:type naming a content-less root's declared type",
			`<xs:element name="known" type="E"/><xs:complexType name="E"/>`, `<known ` + xsiNS + ` xsi:type="E"/>`,
		},
		{
			// Following d.{type definition} instead, the <z> is unattributable.
			"an extension of the declared type below the root, content only the extension admits",
			xsiChild(`<xs:element name="a" type="CA"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{"a restriction of a simple declared type below the root", xsiChild(`<xs:element name="a" type="xs:decimal"/>`), xsiKnown(`<a xsi:type="xs:int">1</a>`)},
		{
			// blockingUnread's identity exception: cos-st-derived-ok clause 1.
			"the declared simple type itself under block=\"restriction\"",
			xsiChild(`<xs:element name="a" type="xs:decimal" block="restriction"/>`), xsiKnown(`<a xsi:type="xs:decimal">1</a>`),
		},
		{
			// elemT040's shape: cos-st-derived-ok reads restriction alone.
			"a simple type for an xs:anyType declaration under block=\"extension\"",
			xsiChild(`<xs:element name="a" block="extension"/>`), xsiKnown(`<a xsi:type="xs:int">1</a>`),
		},
		{
			// xsd reads blocked for a complex declared type other than xs:anyType.
			"an extension of a complex declared type under block=\"restriction\"",
			xsiChild(`<xs:element name="a" type="CA" block="restriction"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{
			// cos-ct-derived-ok reads extension and restriction alone.
			"a complex type for an xs:anyType declaration under block=\"substitution\"",
			xsiChild(`<xs:element name="a" block="substitution"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{
			// The innermost binding of p wins; the root's names no type.
			"an xsi:type prefix rebound on its own element",
			xsiChild(`<xs:element name="a" type="xs:decimal"/>`),
			`<known ` + xsiNS + ` xmlns:p="urn:wrong"><a xmlns:p="http://www.w3.org/2001/XMLSchema" xsi:type="p:int">1</a></known>`,
		},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the walk decides the override and the gate follows it; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesXsiType proves the walk, not the gate, decides an
// xsi:type that does not ·override· (cvc-elt clause 4), does not ·resolve·
// (cvc-attribute clause 5) or is no QName (clause 3): each row is decided
// INVALID.
func TestInstanceExecutorChargesXsiType(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"a type not derived from the declared one (cvc-elt clause 4)", xsiChild(`<xs:element name="a" type="xs:int"/>`), xsiKnown(`<a xsi:type="xs:boolean">1</a>`)},
		{
			"an extension a complex declared type's declaration blocks (cvc-elt clause 4, cos-ct-derived-ok clause 1)",
			xsiChild(`<xs:element name="a" type="CA" block="extension"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{
			// Pinned: the walk DECIDES a QName lexical that resolves to no type.
			"a QName naming no type definition (cvc-attribute clause 5)",
			xsiChild(`<xs:element name="a" type="xs:int"/>`), xsiKnown(`<a xsi:type="Nope">1</a>`),
		},
		{
			// Pinned: the walk CHARGES an unbound prefix under cvc-attribute clause
			// 3 and records clause 5 in Result.Unevaluated; the executor reads the
			// charge first, so the record never makes the case decline.
			"a QName whose prefix is unbound (cvc-attribute clause 3)",
			xsiChild(`<xs:element name="a" type="xs:int"/>`), xsiKnown(`<a xsi:type="q:int">1</a>`),
		},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges the xsi:type; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// TestInstanceExecutorDeclinesUnrecordedOverride pins blockingUnread: every row
// is an xsi:type xsd.Schema.ValidlySubstitutable answers TRUE for although the
// declaration's {disallowed substitutions} blocks it, so the walk charges
// nothing and records nothing, and the gate alone keeps the empty Result from
// reading as "valid".
func TestInstanceExecutorDeclinesUnrecordedOverride(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{
			"a complex type for an xs:anyType declaration under block=\"extension\" (cos-ct-derived-ok clause 1)",
			xsiChild(`<xs:element name="a" block="extension"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{
			"a complex type for an xs:anyType declaration under block=\"restriction\" (cos-ct-derived-ok clause 1)",
			xsiChild(`<xs:element name="a" block="restriction"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{
			// elemT026's shape.
			"a simple type for an xs:anyType declaration under block=\"restriction\" (cos-st-derived-ok clause 2.1)",
			xsiChild(`<xs:element name="a" block="restriction"/>`), xsiKnown(`<a xsi:type="xs:int">1</a>`),
		},
		{
			"a simple restriction of a simple declared type under block=\"restriction\" (cos-st-derived-ok clause 2.1)",
			xsiChild(`<xs:element name="a" type="xs:decimal" block="restriction"/>`), xsiKnown(`<a xsi:type="xs:int">1</a>`),
		},
		{
			"a complex type over a simple declared type under block=\"restriction\" (cos-ct-derived-ok clause 2.3.2.2)",
			xsiChild(`<xs:element name="a" type="xs:decimal" block="restriction"/>`), xsiKnown(`<a xsi:type="CI">1</a>`),
		},
	} {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.why)
	}
}

// TestInstanceExecutorDeclinesUnevaluatedRoot proves a root the gate admits
// whose walk RECORDED a check it did not perform declines rather than reading
// the empty violation list as "valid": validate's cvc-elt clause 5.2.2.2.2
// decline over a fixed {value constraint} it cannot compare in
// xs:anySimpleType's value space (#1738 routed it into Unevaluated), an
// identity constraint whose {selector} icpath does not compile, an
// assertions facet, whose {test} validate records and never evaluates, and a
// complex type's {assertions} (cvc-complex-type clause 6), which validate's
// elementAssertions records the same way.
func TestInstanceExecutorDeclinesUnevaluatedRoot(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"a fixed-value comparison withheld over xs:anySimpleType", `<xs:element name="known" type="xs:anySimpleType" fixed="x"/>`, `<known>x</known>`},
		{
			// A predicate icpath's lexer does not read (validate's icFrame
			// GAP(xpath)): the unique is declined and recorded, not decided.
			"an identity-constraint {selector} outside the ·selector subset· (§3.11.6.2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int" maxOccurs="2"/></xs:sequence></xs:complexType>` +
				`<xs:unique name="u"><xs:selector xpath="a[1]"/><xs:field xpath="."/></xs:unique></xs:element>`,
			`<known><a>1</a><a>1</a></known>`,
		},
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
// clause 3), an xsi:type that does not resolve (cvc-attribute clause 3 or 5) or
// does not ·override· (cvc-elt clause 4) — at the gate itself, since it is a
// precondition in its own right and not a restatement of those charges. No
// executor row can see them, so the gate is called directly, with the declared
// root, with content and without, as the controls. A decoder error is refused
// too.
//
// The no-namespace simple type int is there for the unbound-prefix row: a
// gate that dropped an unbound prefix rather than refusing it would resolve
// "xs:int" to it, a restriction of the declared xs:string.
func TestAssessedSubtreeRootRootConditions(t *testing.T) {
	const schemaBody = knownRoot + `<xs:element name="abstract" type="xs:string" abstract="true"/>` +
		`<xs:simpleType name="int"><xs:restriction base="xs:string"/></xs:simpleType>`
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
		// The walk charges each refused xsi:type below, so only a
		// direct call sees the gate's own refusal.
		{"an xsi:type naming the declared type", `<known ` + xsiXS + ` xsi:type="xs:string">1</known>`, true},
		{"an xsi:type naming no type definition", `<known ` + xsiXS + ` xsi:type="Nope">1</known>`, false},
		{"an xsi:type whose prefix is unbound", `<known ` + xsiNS + ` xsi:type="xs:int">1</known>`, false},
		{"an xsi:type that does not ·override· the declared type", `<known ` + xsiXS + ` xsi:type="xs:int">1</known>`, false},
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
