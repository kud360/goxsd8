package conformance

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/xsderr"
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
		// The three ID-family rows, one per value-type site the gate reads (an
		// element's simple type, an attribute's, a simple {content type}), are
		// admitted on the terms of the third shape's cvc-elt clause 7 bullet
		// (instance.go) (#1857).
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
			defaultedAtt("key"), `<known><e att="1"/><e att="2"/></known>`,
		},
		{
			// The defaulted "a" fills the first <e>'s ·key-sequence· (§3.11.4
			// clause 3's Note), so clause 4.2.1 holds.
			"a key whose {fields} path selects a ·defaulted attribute·, the values distinct",
			defaultedAtt("key"), `<known><e/><e att="b"/></known>`,
		},
		// The NOTATION rows, one per value-type site the gate reads and one
		// through the lax {attribute wildcard}, name a declared notation the
		// enumeration admits: String Valid is the walk's (#1904), NOTATION's
		// ·value space· decided by walk.notationsDeclared (Datatypes §3.3.19).
		// TestInstanceExecutorChargesNotation holds the undeclared values.
		{"a NOTATION enumeration as the root's value type", notationN + `<xs:element name="known" type="N"/>`, `<known>n</known>`},
		{
			"a NOTATION enumeration as an element value type below the root",
			notationN + `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="N"/></xs:sequence></xs:complexType></xs:element>`,
			`<known><a>n</a></known>`,
		},
		{"a NOTATION enumeration as a simple {content type}", notationN + notationContent, `<known at="v">n</known>`},
		{"an attribute typed by a NOTATION enumeration below the root", notationN + notationAttribute, `<known><a n="n"/></known>`},
		{
			"a lax {attribute wildcard}'s attribute resolving to a declaration typed by a NOTATION enumeration",
			notationN + `<xs:attribute name="n" type="N"/>` + wildcardKnown("lax"), `<known n="n"><a>1</a></known>`,
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

// notationContent declares <known> with a simple {content type} of the NOTATION
// enumeration N (notationN) and an xs:string attribute at; notationAttribute
// declares <known> with one required child <a> carrying an attribute n of type
// N.
const (
	notationContent = `<xs:element name="known"><xs:complexType><xs:simpleContent><xs:extension base="N">` +
		`<xs:attribute name="at" type="xs:string"/></xs:extension></xs:simpleContent></xs:complexType></xs:element>`
	notationAttribute = `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a">` +
		`<xs:complexType><xs:attribute name="n" type="N"/></xs:complexType></xs:element></xs:sequence></xs:complexType></xs:element>`
)

// TestInstanceExecutorChargesNotation is a regression guard for the NOTATION
// lift (#1904): the value bez, naming a notation N's enumeration does not admit
// and the schema does not declare, is charged at each site the
// decides rows admit — String Valid's cvc-enumeration-valid verdict (Datatypes
// §4.3.5.4) wrapped under the rule each row names — so each is decided
// INVALID. The walk charges it before the gate is asked, so no row here can
// see the gate.
func TestInstanceExecutorChargesNotation(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"the root's value type (cvc-type clause 3.1.3)", notationN + `<xs:element name="known" type="N"/>`, `<known>bez</known>`},
		{"a simple {content type} (cvc-complex-type clause 1.2)", notationN + notationContent, `<known at="v">bez</known>`},
		{"an attribute below the root (cvc-attribute clause 3)", notationN + notationAttribute, `<known><a n="bez"/></known>`},
		{
			"a lax {attribute wildcard}'s attribute resolving a declaration (cvc-attribute clause 3)",
			notationN + `<xs:attribute name="n" type="N"/>` + wildcardKnown("lax"), `<known n="bez"><a>1</a></known>`,
		},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges the undeclared NOTATION value; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
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
// definitions} admitted, a duplicate key (cvc-identity-constraint clause 4.2.2),
// a dangling keyref (clause 4.3) and a unique whose two targets share a
// ·defaulted attribute·'s value (clause 4.1) over the root's children are still
// decided INVALID. Each is the walk's charge (validate's icFrame.duplicates and
// icCheck.keyrefs), so no row passes through the gate.
func TestInstanceExecutorChargesIdentityConstraintBelowTheRoot(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"a duplicate key (cvc-identity-constraint clause 4.2.2)", keyAndRef, `<known><a>1</a><a>1</a></known>`},
		{"a dangling keyref (cvc-identity-constraint clause 4.3)", keyAndRef, `<known><a>1</a><a>2</a><b>3</b></known>`},
		{
			// idZ011_a's shape: both <e> carry the default "a" (§3.11.4 clause
			// 3's Note, validate's icCheck.fieldDefaultedAttributes).
			"a unique whose {fields} path selects a ·defaulted attribute· both targets share (cvc-identity-constraint clause 4.1)",
			defaultedAtt("unique"), `<known><e/><e/></known>`,
		},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges cvc-identity-constraint; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// defaultedAtt declares <known> with up to two children <e>, each with an
// optional xs:string attribute att defaulting to "a", and an identity
// constraint of category ic (unique or key) on <known> selecting e/@att.
func defaultedAtt(ic string) string {
	return `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="e" maxOccurs="2"><xs:complexType>` +
		`<xs:attribute name="att" type="xs:string" default="a"/></xs:complexType></xs:element></xs:sequence></xs:complexType>` +
		`<xs:` + ic + ` name="u"><xs:selector xpath="e"/><xs:field xpath="@att"/></xs:` + ic + `></xs:element>`
}

// entityDefault declares <known>, empty, with an xs:ENTITY attribute ent
// defaulting to pic (cvc-attribute clause 3 over a ·defaulted attribute·,
// §3.4.2.4), and pics is the DOCTYPE whose internal subset declares pic an
// unparsed entity of notation gif — id017's shape.
const (
	entityDefault = `<xs:element name="known"><xs:complexType>` +
		`<xs:attribute name="ent" type="xs:ENTITY" default="pic"/></xs:complexType></xs:element>`
	pics = `<!DOCTYPE known [<!ENTITY pic SYSTEM "pic.gif" NDATA gif><!NOTATION gif SYSTEM "gif">]>`
)

// TestInstanceExecutorDecidesInternalSubsetEntity proves the gate admits a
// DOCTYPE whose DTD defaults no attribute (#2063): an xs:ENTITY default naming
// the unparsed entity its internal subset declares is decided VALID (key-vde,
// cvc-simple-type clause 3). With rootStart refusing every directive again the
// row declines and this test fails.
func TestInstanceExecutorDecidesInternalSubsetEntity(t *testing.T) {
	exec := newInstanceExec()
	c := instanceCase(t, entityDefault, pics+`<known/>`, true)
	if !exec(c).IsPass() {
		t.Error("an xs:ENTITY default naming a declared unparsed entity: the executor must agree with a suite-valid case")
	}
	c.expect = expectValidity(false)
	if exec(c).IsPass() {
		t.Error("an xs:ENTITY default naming a declared unparsed entity: the executor must Fail under a flipped expectation")
	}
}

// TestInstanceExecutorChargesInternalSubsetEntity is a regression guard for
// the DOCTYPE lift (#2063): under an admitted DOCTYPE, an xs:ENTITY default
// naming no unparsed entity the subset declares is not a ·declared entity
// name· (key-vde), which the walk charges under cvc-attribute clause 3, so the
// row is decided INVALID and never reaches the gate.
func TestInstanceExecutorChargesInternalSubsetEntity(t *testing.T) {
	exec := newInstanceExec()
	c := instanceCase(t, entityDefault, `<!DOCTYPE known [<!ENTITY other SYSTEM "o.gif" NDATA gif><!NOTATION gif SYSTEM "gif">]><known/>`, false)
	if !exec(c).IsPass() {
		t.Error("an xs:ENTITY default naming an undeclared entity: the walk charges it; the executor must agree with a suite-invalid case")
	}
	c.expect = expectValidity(true)
	if exec(c).IsPass() {
		t.Error("an xs:ENTITY default naming an undeclared entity: the executor must Fail under a flipped expectation")
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
		// rootStart's defaultsNoAttribute: each DTD could default an attribute
		// the gate does not see (XML 1.0 §3.3.2). The two internal-subset rows
		// do default a="x" onto <known>, which emptyRoot's type does not admit.
		{"a DOCTYPE naming an external subset by a SYSTEM id", aInt, `<!DOCTYPE known SYSTEM "k.dtd"><known><a>1</a></known>`},
		{"a DOCTYPE naming an external subset by a PUBLIC id", aInt, `<!DOCTYPE known PUBLIC "-//k//EN" "k.dtd"><known><a>1</a></known>`},
		{"a DOCTYPE whose internal subset holds an <!ATTLIST", emptyRoot, `<!DOCTYPE known [<!ATTLIST known a CDATA "x">]><known/>`},
		{
			// §4.4.8: %p; expands to an <!ATTLIST the subset spells only as
			// &#60;!ATTLIST, which no literal scan sees.
			"a DOCTYPE whose internal subset holds a parameter-entity reference", emptyRoot,
			`<!DOCTYPE known [<!ENTITY % p "&#60;!ATTLIST known a CDATA 'x'>"> %p;]><known/>`,
		},
		{
			// rootStart's doc: the quote in the PI ends encoding/xml's DOCTYPE
			// at the second PI's ?>, so the <!ATTLIST arrives as a directive of
			// its own.
			"an <!ATTLIST after a DOCTYPE encoding/xml delimits short", emptyRoot,
			`<!DOCTYPE known [<?pi '?><!ENTITY e 'a>b'><?pi '?> <!ATTLIST known a CDATA 'x'>]><known/>`,
		},
		{
			// rootStart's doc: encoding/xml fails on &g;, which §4.4.2 includes.
			"a reference to a general entity an admitted DOCTYPE declares", aInt,
			`<!DOCTYPE known [<!ENTITY g "1">]><known><a>&g;</a></known>`,
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
			// subtreeGate's nilValue: governed by §3.2.7's built-in declaration
			// (key-governing-ad), the attribute is not ·valid· against xs:boolean
			// (cvc-attribute clause 3), which the walk does not check on an
			// element with no declaration.
			"an xsi:nil with no ·actual value· on a lax wildcard particle's child resolving no declaration",
			wildcardChild("lax"), `<known ` + xsiNS + `><u xsi:nil="maybe"/></known>`,
		},
		{
			"an abstract declaration below the root (cvc-elt clause 2)",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:element ref="b"/></xs:sequence></xs:complexType></xs:element>` +
				`<xs:element name="b" type="xs:string" abstract="true"/>`,
			`<known><b>x</b></known>`,
		},
		{
			// cvc-accept clause 2.3.1 attributes <h> to h itself, not to a member.
			"an abstract substitution group head used directly below the root (cvc-elt clause 2)",
			headKnown(`<xs:element name="h" type="A" abstract="true"/><xs:element name="m" type="R" substitutionGroup="h"/>`),
			`<known><h>1</h></known>`,
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
			// A guard, not a charged row: the gate refused this shape before #1860
			// too. The walk charges nothing and records nothing for it, a reading
			// the suite does not share (#1912).
			"a strict {attribute wildcard}'s attribute whose name resolves no declaration (cvc-complex-type clause 2.2)",
			wildcardKnown("strict"), `<known foo="1"><a>1</a></known>`,
		},
		// The wildcard-child refusals (#1931): subtreeGate.resolvedChild's.
		{
			// resolvedChild's GAP(conformance) xsi:type refusal, under lax as
			// under strict (#1911).
			"a lax wildcard particle's child resolving no declaration, typed by an xsi:type",
			wildcardChild("lax"), `<known ` + xsiXS + `><u xsi:type="xs:int">1</u></known>`,
		},
		{
			// Strictly assessed against N through the xsi:type, the child takes
			// the walk's e-validity clause 1.1.3 charge away; resolvedChild's
			// GAP(conformance) xsi:type refusal declines it, not N's NOTATION.
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
	}
	for _, tc := range cases {
		declinesBothPolarities(t, exec, instanceCase(t, tc.schemaBody, tc.instance, false), tc.condition)
	}
}

// fixedRoot declares <known> of type xs:int, fixed to 1; fixedBelow declares
// <known> with one required child <a> of type xs:int, fixed to 1; fixedMixed
// declares <known> of a mixed type with one optional child <c>, fixed to x.
const (
	fixedRoot  = `<xs:element name="known" type="xs:int" fixed="1"/>`
	fixedBelow = `<xs:element name="known"><xs:complexType><xs:sequence><xs:element name="a" type="xs:int" fixed="1"/></xs:sequence></xs:complexType></xs:element>`
	fixedMixed = `<xs:element name="known" fixed="x"><xs:complexType mixed="true"><xs:sequence>` +
		`<xs:element name="c" minOccurs="0"/></xs:sequence></xs:complexType></xs:element>`
)

// TestInstanceExecutorDecidesFixedValueConstraint proves the gate admits a
// declaration carrying a fixed {value constraint}, at the root and below it,
// where the walk settles cvc-elt clause 5 for it (#1979): each row is decided
// VALID. With assessedDeclaration refusing ValueFixed again, every row
// declines and this test fails.
func TestInstanceExecutorDecidesFixedValueConstraint(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		{"the root's ·initial value· equal to the fixed one (cvc-elt clause 5.2.2.2.2)", fixedRoot, `<known>1</known>`},
		{"the root's ·initial value· lexically different, equal in value space (cvc-elt clause 5.2.2.2.2)", fixedRoot, `<known> 01 </known>`},
		{"an empty root taking the fixed {lexical form} (cvc-elt clause 5.1)", fixedRoot, `<known/>`},
		{"a mixed root's ·initial value· matching the fixed {lexical form} (cvc-elt clause 5.2.2.2.1)", fixedMixed, `<known>x</known>`},
		{"a child's ·initial value· equal to the fixed one (cvc-elt clause 5.2.2.2.2)", fixedBelow, `<known><a>1</a></known>`},
		{"a child's ·initial value· lexically different, equal in value space (cvc-elt clause 5.2.2.2.2)", fixedBelow, `<known><a>01</a></known>`},
		{"an empty child taking the fixed {lexical form} (cvc-elt clause 5.1)", fixedBelow, `<known><a/></known>`},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the walk settles cvc-elt clause 5 and the gate admits the fixed declaration; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesFixedValueConstraint proves the walk, not the gate,
// decides a fixed {value constraint} the [[children]] disagree with, at the root
// and below it: each row is charged cvc-elt under the clause it names, and is
// decided INVALID.
func TestInstanceExecutorChargesFixedValueConstraint(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance, clause string }{
		{"the root's ·actual value· unequal to the fixed one", fixedRoot, `<known>2</known>`, "5.2.2.2.2"},
		{"a mixed root's ·initial value· not matching the fixed {lexical form}", fixedMixed, `<known>y</known>`, "5.2.2.2.1"},
		{"a mixed root with element [[children]]", fixedMixed, `<known>x<c/></known>`, "5.2.2.1"},
		{"a child's ·actual value· unequal to the fixed one", fixedBelow, `<known><a>2</a></known>`, "5.2.2.2.2"},
	} {
		c := instanceCase(t, tc.schemaBody, tc.instance, false)
		if !chargedCvcElt(t, c, tc.clause) {
			t.Errorf("%s: the walk must charge cvc-elt clause %s", tc.why, tc.clause)
		}
		if !exec(c).IsPass() {
			t.Errorf("%s: the walk charges cvc-elt clause %s; the executor must agree with a suite-invalid case", tc.why, tc.clause)
		}
		c.expect = expectValid()
		if exec(c).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// nilInt declares <known> of type xs:int, {nillable} true; nilKnown declares
// <known> with an optional {nillable} child <a> of type xs:int and an optional
// {nillable} child <c>, whose type requires one child <d> of type xs:int and
// declares an attribute at of type xs:string.
const (
	nilInt   = `<xs:element name="known" type="xs:int" nillable="true"/>`
	nilKnown = `<xs:element name="known"><xs:complexType><xs:sequence>` +
		`<xs:element name="a" type="xs:int" nillable="true" minOccurs="0"/>` +
		`<xs:element name="c" nillable="true" minOccurs="0"><xs:complexType><xs:sequence><xs:element name="d" type="xs:int"/></xs:sequence>` +
		`<xs:attribute name="at" type="xs:string"/></xs:complexType></xs:element>` +
		`</xs:sequence></xs:complexType></xs:element>`
)

// TestInstanceExecutorDecidesXsiNil proves the gate admits an element carrying
// xsi:nil wherever the walk decides cvc-elt clause 3 for it (#2053): a ·nilled·
// one (key-nilled) with no [[children]] (3.2.3), and xsi:nil false under a
// {nillable} declaration (3.2.2), read as if absent. Each row is decided VALID,
// and each declines with subtreeGate refusing every xsi:nil again. The rows
// whose content model rejects the empty sequence decline too with complex
// reading a ·nilled· element's content through the ContentMatcher, which
// cvc-complex-type clause 1, applying only to an element that is not ·nilled·,
// does not ask.
func TestInstanceExecutorDecidesXsiNil(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		// "" is no xs:int, but cvc-type clause 3.1.3 skips a ·nilled· element.
		{"a ·nilled· simple-typed root with no content", nilInt, `<known ` + xsiNS + ` xsi:nil="true"/>`},
		{
			"a ·nilled· root whose element-only content model rejects the empty sequence",
			`<xs:element name="known" nillable="true"><xs:complexType><xs:sequence><xs:element name="d" type="xs:int"/></xs:sequence></xs:complexType></xs:element>`,
			`<known ` + xsiNS + ` xsi:nil="1"/>`,
		},
		{"a ·nilled· child carrying a matched attribute, its required child absent", nilKnown, `<known ` + xsiNS + `><c xsi:nil="true" at="v"/></known>`},
		{"a ·nilled· simple-typed child, its xsi:nil padded with white space", nilKnown, `<known ` + xsiNS + `><a xsi:nil=" true "/></known>`},
		{"xsi:nil false on a {nillable} child with content (cvc-elt clause 3.2.2)", nilKnown, `<known ` + xsiNS + `><a xsi:nil="false">1</a></known>`},
		{"xsi:nil 0 on a {nillable} root with content (cvc-elt clause 3.2.2)", nilInt, `<known ` + xsiNS + ` xsi:nil="0">1</known>`},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the walk decides cvc-elt clause 3 and the gate admits the xsi:nil; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesXsiNil is a regression guard for the xsi:nil lift
// (#2053): each row is charged cvc-elt under the clause it names — by
// validate's nilCheck, or its contentCheck for 3.2.3.1 — and decided INVALID
// before the gate is read.
func TestInstanceExecutorChargesXsiNil(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance, clause string }{
		{"a ·nilled· root with character content", nilInt, `<known ` + xsiNS + ` xsi:nil="true">1</known>`, "3.2.3.1"},
		{"a ·nilled· child holding white space", nilKnown, `<known ` + xsiNS + `><a xsi:nil="true"> </a></known>`, "3.2.3.1"},
		{"a ·nilled· child with an element child", nilKnown, `<known ` + xsiNS + `><c xsi:nil="true"><d>1</d></c></known>`, "3.2.3.1"},
		{"xsi:nil false on a declaration whose {nillable} is false", knownRoot, `<known ` + xsiNS + ` xsi:nil="false">x</known>`, "3.1"},
		{"xsi:nil true on a declaration whose {nillable} is false", knownRoot, `<known ` + xsiNS + ` xsi:nil="true"/>`, "3.1"},
		{
			"a ·nilled· root under a fixed {value constraint}",
			`<xs:element name="known" type="xs:int" nillable="true" fixed="1"/>`, `<known ` + xsiNS + ` xsi:nil="true"/>`, "3.2.3.2",
		},
		{"an xsi:nil with no ·actual value· on a {nillable} root", nilInt, `<known ` + xsiNS + ` xsi:nil="maybe">1</known>`, "3.2"},
	} {
		c := instanceCase(t, tc.schemaBody, tc.instance, false)
		if !chargedCvcElt(t, c, tc.clause) {
			t.Errorf("%s: the walk must charge cvc-elt clause %s", tc.why, tc.clause)
		}
		if !exec(c).IsPass() {
			t.Errorf("%s: the walk charges cvc-elt clause %s; the executor must agree with a suite-invalid case", tc.why, tc.clause)
		}
		c.expect = expectValid()
		if exec(c).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// TestAssessedSubtreeRootNilled pins, at the gate itself, that a ·nilled·
// element's content is read as a leaf: an element child is refused, which no
// executor row can see because the walk charges cvc-elt clause 3.2.3.1 for it
// first. The empty ·nilled· element is the control.
func TestAssessedSubtreeRootNilled(t *testing.T) {
	for _, tc := range []struct {
		why, instance string
		want          bool
	}{
		{"a ·nilled· child with no content", `<known ` + xsiNS + `><c xsi:nil="true"/></known>`, true},
		{"a ·nilled· child with an element child", `<known ` + xsiNS + `><c xsi:nil="true"><d>1</d></c></known>`, false},
	} {
		c := instanceCase(t, nilKnown, tc.instance, true)
		schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
		if err != nil || !decidable {
			t.Fatalf("%s: assembling the schema: decidable %v, err %v", tc.why, decidable, err)
		}
		if got := assessedSubtreeRoot(schema, report, c.doc); got != tc.want {
			t.Errorf("%s: assessedSubtreeRoot = %v, want %v", tc.why, got, tc.want)
		}
	}
}

// chargedCvcElt reports whether assessing c's instance against its schema
// charges cvc-elt with a message naming clause, the walk's spelling of the
// clause it settled.
func chargedCvcElt(t *testing.T, c caseSpec, clause string) bool {
	t.Helper()
	schema, _, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
	if err != nil || !decidable {
		t.Fatalf("assembling the schema: decidable %v, err %v", decidable, err)
	}
	v, err := validate.New(schema, strict.New())
	if err != nil {
		t.Fatalf("validate.New: %v", err)
	}
	result, ok := assessInstance(v, c.doc)
	if !ok {
		t.Fatalf("assessing %s: declined", c.doc)
	}
	return slices.ContainsFunc(result.Violations(), func(e *xsderr.Error) bool {
		return e.Rule == ruleCvcElt && strings.Contains(e.Msg, "cvc-elt clause "+clause+" ")
	})
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

// openChild declares <known> with one required child <a> of type xs:int and an
// interleave {open content} whose wildcard has {process contents} pc, beside
// the top-level declaration b of type xs:int, which the content model does not
// declare.
func openChild(pc string) string {
	return `<xs:element name="known"><xs:complexType><xs:openContent mode="interleave"><xs:any processContents="` + pc + `"/></xs:openContent>` +
		`<xs:sequence><xs:element name="a" type="xs:int"/></xs:sequence></xs:complexType></xs:element>` +
		`<xs:element name="b" type="xs:int"/>`
}

// TestInstanceExecutorDecidesWildcardChild proves the gate admits a child
// ·attributed to· a strict or lax ·wildcard particle· or to a lax {open
// content} wherever its name ·resolves· to a top-level declaration and its
// ·locally declared type· is ·absent·, which makes cvc-complex-type clause 5
// vacuous (#1931), a skip ·wildcard particle·'s or skip {open content}'s child
// whatever its subtree holds (key-sva clause 3.2, #1861, #1969), and a lax
// one's child resolving to none, ·laxly assessed· with its subtree (#1911).
// Each row walks clean, and each is refused with subtreeGate.resolvedChild
// answering false for a resolved name, with laxlyAssessed answering false for
// an unresolved one, or with child answering false for the {open content} or
// skip Wildcard arm.
func TestInstanceExecutorDecidesWildcardChild(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
		// Assessed, <b> would be charged against b's xs:int, <c> would resolve
		// nothing, and <i>'s value is no xs:ID: skipped, none of it is read
		// (cvc-assess-elt clause 2).
		{"skip, a child with arbitrary undeclared content", wildcardChild("skip"), `<known><b u="x">x<c><i>not an ID</i></c></b></known>`},
		{"strict, a child resolving a declaration", wildcardChild("strict"), `<known><b>1</b></known>`},
		{"lax, a child resolving a declaration", wildcardChild("lax"), `<known><b>1</b></known>`},
		// NIST2004-01-14's shape: an ID-typed child binding the wrapper.
		{"strict, a child resolving an xs:ID declaration", wildcardChild("strict"), `<known><i>i1</i></known>`},
		{"an {open content} child resolving a declaration", openChild("lax"), `<known><b>1</b><a>1</a></known>`},
		// Assessed, <b> would be charged against b's xs:int: skipped, none of it
		// is read (cvc-assess-elt clause 2, #1969).
		{"a skip {open content} child with arbitrary content", openChild("skip"), `<known><b u="x">x<c/></b><a>1</a></known>`},
		// ·laxly assessed· against ·xs:anyType· (cvc-assess-elt clause 3,
		// key-lva), [validity] notKnown, blocking no ancestor (#1911).
		{"lax, a child resolving no declaration", wildcardChild("lax"), `<known><u>x</u></known>`},
		{"an {open content} child resolving no declaration", openChild("lax"), `<known><u>x</u><a>1</a></known>`},
		// Below it: an attribute resolving none (key-sva clause 2.2), one
		// resolving none again (laxly assessed), and <b> strictly assessed
		// against b (#1823).
		{"lax, a child resolving no declaration over a subtree the walk assesses", wildcardChild("lax"), `<known><u foo="x">t<v><w/></v><b>1</b></u></known>`},
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
		{"an {open content} child's value not valid against the resolved declaration", openChild("lax"), `<known><b>x</b><a>1</a></known>`},
		{"strict, a child resolving no declaration (e-validity clause 1.1.3)", wildcardChild("strict"), `<known><u>x</u></known>`},
		// §2.5 key-deep-valid-doc clauses 3 and 4: invalid below a ·laxly
		// assessed· <u>, notKnown, which clause 1.1.2 does not propagate past
		// (#1911).
		{"lax, below a child resolving no declaration, a value not valid against a resolved declaration (#1823)", wildcardChild("lax"), `<known><u><b>x</b></u></known>`},
		{
			"lax, on a child resolving no declaration, an attribute value not valid against a resolved declaration (#1891)",
			wildcardChild("lax") + `<xs:attribute name="ta" type="xs:int"/>`, `<known><u ta="x"/></known>`,
		},
		{
			// The name check survives skip (cvc-wildcard clause 1): the Matcher
			// ·attributes· <u> to no particle (cvc-complex-type clause 1.4).
			"skip, a child whose name the wildcard's namespace constraint refuses",
			`<xs:element name="known"><xs:complexType><xs:sequence><xs:any namespace="urn:other" processContents="skip"/></xs:sequence></xs:complexType></xs:element>`,
			`<known><u/></known>`,
		},
	} {
		if !exec(instanceCase(t, tc.schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges the child; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, tc.schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
		}
	}
}

// TestAssessedSubtreeRootUnresolvedChild pins, at the gate itself, its answer
// for a wildcard particle's child resolving no declaration. Under strict, which
// no executor row can see because the walk charges e-validity clause 1.1.3 for
// it first (TestInstanceExecutorChargesWildcardChild), the gate admits it with
// its subtree unread. Under lax it reads the subtree as ·laxly assessed·
// against ·xs:anyType· (subtreeGate.laxlyAssessed, #1911), and the refused rows
// are each a condition element puts on an element at or below the lax child,
// which the gate holds there too.
func TestAssessedSubtreeRootUnresolvedChild(t *testing.T) {
	for _, tc := range []struct {
		why, schemaBody, instance string
		want                      bool
	}{
		{"strict, a child resolving no declaration", wildcardChild("strict"), `<known><u><v/></u></known>`, true},
		{"lax, a child resolving no declaration", wildcardChild("lax"), `<known><u><v/></u></known>`, true},
		// u's own binding of z must reach resolveQName for b's xsi:type.
		{
			"lax, below a child resolving no declaration, an xsi:type bound by a prefix that child declares",
			wildcardChild("lax"), `<known ` + xsiNS + `><u xmlns:z="http://www.w3.org/2001/XMLSchema"><b xsi:type="z:int">1</b></u></known>`, true,
		},
		// key-nilled is relative to a declaration, so a laxly assessed element
		// is never ·nilled·: its element child is read, not refused.
		{"lax, a child resolving no declaration carrying xsi:nil true over an element child", wildcardChild("lax"), `<known ` + xsiNS + `><u xsi:nil="true"><v/></u></known>`, true},
		{"lax, a child resolving no declaration carrying an xsi:nil with no ·actual value·", wildcardChild("lax"), `<known ` + xsiNS + `><u xsi:nil="maybe"/></known>`, false},
		{"lax, below a child resolving no declaration, an xsi:nil with no ·actual value·", wildcardChild("lax"), `<known ` + xsiNS + `><u><v xsi:nil="maybe"/></u></known>`, false},
		{
			// The walk settles cvc-elt clause 5.2.2 for f, below the lax <u> too (#1979).
			"lax, below a child resolving no declaration, a resolved declaration with a fixed {value constraint} (cvc-elt clause 5.2.2)",
			wildcardChild("lax") + `<xs:element name="f" type="xs:int" fixed="1"/>`, `<known><u><f>1</f></u></known>`, true,
		},
	} {
		c := instanceCase(t, tc.schemaBody, tc.instance, true)
		schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
		if err != nil || !decidable {
			t.Fatalf("%s: assembling the schema: decidable %v, err %v", tc.why, decidable, err)
		}
		if got := assessedSubtreeRoot(schema, report, c.doc); got != tc.want {
			t.Errorf("%s: assessedSubtreeRoot = %v, want %v", tc.why, got, tc.want)
		}
	}
}

// headKnown declares <known>, whose content model is one element particle
// referencing the top-level h, beside decls, which declare h and its
// ·substitution group· member m, and the simple types A, an xs:int
// restriction, and R, a restriction of A.
func headKnown(decls string) string {
	return `<xs:element name="known"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>` +
		`<xs:simpleType name="A"><xs:restriction base="xs:int"/></xs:simpleType>` +
		`<xs:simpleType name="R"><xs:restriction base="A"/></xs:simpleType>` + decls
}

// extendsA declares EA, a simpleContent extension of A.
const extendsA = `<xs:complexType name="EA"><xs:simpleContent><xs:extension base="A"/></xs:simpleContent></xs:complexType>`

// TestInstanceExecutorDecidesSubstitutionGroupMember proves the gate admits a
// child cvc-accept clause 2.3.2 ·attributes· to an element particle of another
// name, as a member of its {term}'s ·substitution group·, and reads it on
// against the member's declaration (key-governing-ed clause 2, #1932). Each row
// names the subtreeGate condition that, removed, turns it into a decline.
func TestInstanceExecutorDecidesSubstitutionGroupMember(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, decls string }{
		// child's lift itself: m inherits A.
		{"an unblocked member", `<xs:element name="h" type="A"/><xs:element name="m" substitutionGroup="h"/>`},
		// child passes element the member s, not the particle's abstract {term}.
		{"a member of an abstract head", `<xs:element name="h" type="A" abstract="true"/><xs:element name="m" type="R" substitutionGroup="h"/>`},
		// A complex step reaching A takes no simple step (MS-Element elemT062.v
		// and elemT064.v's sa2).
		{
			`a simpleContent extension of the head's simple type under block="restriction"`,
			`<xs:element name="h" type="A" block="restriction"/><xs:element name="m" type="EA" substitutionGroup="h"/>` + extendsA,
		},
		// m inherits h's anonymous simple type, so no step is taken and no {derivation method} is involved
		// (cos-equiv-derived-ok-rec clause 2.3).
		{
			`a member inheriting the head's anonymous simple type under block="restriction"`,
			`<xs:element name="h" block="restriction"><xs:simpleType><xs:restriction base="xs:int"/></xs:simpleType></xs:element>` +
				`<xs:element name="m" substitutionGroup="h"/>`,
		},
		// A simple step, restriction unblocked (cos-equiv-derived-ok-rec clause 2.3).
		{
			`a simple restriction of the head's type under block="extension"`,
			`<xs:element name="h" type="A" block="extension"/><xs:element name="m" type="R" substitutionGroup="h"/>`,
		},
		// s.{type definition}'s own {prohibited substitutions} is not in clause
		// 2.3's union.
		{
			"a member type prohibiting restriction itself, over a simple step",
			`<xs:element name="h" type="A"/><xs:element name="m" type="ER" substitutionGroup="h"/>` +
				`<xs:complexType name="ER" block="restriction"><xs:simpleContent><xs:extension base="R"/></xs:simpleContent></xs:complexType>`,
		},
	} {
		schemaBody := headKnown(tc.decls)
		if !exec(instanceCase(t, schemaBody, `<known><m>1</m></known>`, true)).IsPass() {
			t.Errorf("%s: the walk decides the member and the gate admits it; the executor must agree with a suite-valid case", tc.why)
		}
		if exec(instanceCase(t, schemaBody, `<known><m>1</m></known>`, false)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation (it decides for real)", tc.why)
		}
	}
}

// TestInstanceExecutorChargesSubstitutionGroupMember is a regression guard for
// the member lift (#1932): the walk charges each row, so each is decided
// INVALID whatever the gate answers. In the first two the Matcher attributes
// the member to nothing (cvc-accept clause 2.3.2, cos-equiv-derived-ok-rec
// clauses 2.1 and 2.3).
func TestInstanceExecutorChargesSubstitutionGroupMember(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, decls, instance string }{
		{
			`a head under block="substitution" (clause 2.1)`,
			`<xs:element name="h" type="A" block="substitution"/><xs:element name="m" type="R" substitutionGroup="h"/>`,
			`<known><m>1</m></known>`,
		},
		{
			`an extension under the head's block="extension" (clause 2.3)`,
			`<xs:element name="h" type="A" block="extension"/><xs:element name="m" type="EA" substitutionGroup="h"/>` + extendsA,
			`<known><m>1</m></known>`,
		},
		// MS-Element elemT063.i's shape (#1942): a simple restriction of the
		// head's type is restriction in clause 2.3's union, which the head's
		// {disallowed substitutions} holds.
		{
			`a simple restriction of the head's type under block="restriction" (clause 2.3)`,
			`<xs:element name="h" type="A" block="restriction"/><xs:element name="m" type="R" substitutionGroup="h"/>`,
			`<known><m>1</m></known>`,
		},
		// The intermediate arm of the union: C, strictly between ER2 and A,
		// prohibits restriction, and R's step below A is simple (#1942).
		{
			"an intermediate type prohibiting restriction, over a simple step (clause 2.3)",
			`<xs:element name="h" type="A"/><xs:element name="m" type="ER2" substitutionGroup="h"/>` +
				`<xs:complexType name="C" block="restriction"><xs:simpleContent><xs:extension base="R"/></xs:simpleContent></xs:complexType>` +
				`<xs:complexType name="ER2"><xs:simpleContent><xs:extension base="C"/></xs:simpleContent></xs:complexType>`,
			`<known><m>1</m></known>`,
		},
		{
			"a member value valid against the head's type and not its own (cvc-type clause 3.1.3)",
			`<xs:element name="h" type="xs:decimal"/><xs:element name="m" type="A" substitutionGroup="h"/>`,
			`<known><m>1.5</m></known>`,
		},
	} {
		schemaBody := headKnown(tc.decls)
		if !exec(instanceCase(t, schemaBody, tc.instance, false)).IsPass() {
			t.Errorf("%s: the walk charges the child; the executor must agree with a suite-invalid case", tc.why)
		}
		if exec(instanceCase(t, schemaBody, tc.instance, true)).IsPass() {
			t.Errorf("%s: the executor must Fail under a flipped expectation", tc.why)
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
			// cos-st-derived-ok clause 1: identity is admitted whatever blocked holds.
			"the declared simple type itself under block=\"restriction\"",
			xsiChild(`<xs:element name="a" type="xs:decimal" block="restriction"/>`), xsiKnown(`<a xsi:type="xs:decimal">1</a>`),
		},
		{
			// MS-Element elemT058.v's shape: cos-ct-derived-ok clause 1 reads
			// extension, not in {restriction}, and clause 2.2 ends the walk at the
			// declared type before cos-st-derived-ok is reached.
			"a simpleContent extension of the declared simple type under block=\"restriction\"",
			xsiChild(`<xs:element name="a" type="xs:int" block="restriction"/>`), xsiKnown(`<a xsi:type="CI">1</a>`),
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
			"a simple restriction of a simple declared type under block=\"restriction\" (cvc-elt clause 4, cos-st-derived-ok clause 2.1)",
			xsiChild(`<xs:element name="a" type="xs:decimal" block="restriction"/>`), xsiKnown(`<a xsi:type="xs:int">1</a>`),
		},
		{
			"a complex type over a simple declared type under block=\"restriction\" (cvc-elt clause 4, cos-ct-derived-ok clause 2.3.2.2)",
			xsiChild(`<xs:element name="a" type="xs:decimal" block="restriction"/>`), xsiKnown(`<a xsi:type="CI">1</a>`),
		},
		{
			"a complex type for an xs:anyType declaration under block=\"extension\" (cvc-elt clause 4, cos-ct-derived-ok clause 1)",
			xsiChild(`<xs:element name="a" block="extension"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{
			// ECA's base CA restricts xs:anyType: clause 1 turns that step away.
			"a complex type for an xs:anyType declaration under block=\"restriction\" (cvc-elt clause 4, cos-ct-derived-ok clause 1)",
			xsiChild(`<xs:element name="a" block="restriction"/>`), xsiKnown(`<a xsi:type="ECA"><x>1</x><z>2</z></a>`),
		},
		{
			// elemT026's shape.
			"a simple type for an xs:anyType declaration under block=\"restriction\" (cvc-elt clause 4, cos-st-derived-ok clause 2.1)",
			xsiChild(`<xs:element name="a" block="restriction"/>`), xsiKnown(`<a xsi:type="xs:int">1</a>`),
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

// TestInstanceExecutorDeclinesUnevaluatedRoot proves a root the gate admits
// whose walk RECORDED a check it did not perform declines rather than reading
// the empty violation list as "valid": an identity constraint whose
// {selector} icpath does not compile, an assertions facet, whose {test}
// validate records and never evaluates, and a complex type's {assertions}
// (cvc-complex-type clause 6), which validate's elementAssertions records the
// same way.
func TestInstanceExecutorDeclinesUnevaluatedRoot(t *testing.T) {
	exec := newInstanceExec()
	for _, tc := range []struct{ why, schemaBody, instance string }{
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
// 3.1.2), an attribute outside the xsi: four (3.1.1), an xsi:nil with no
// ·actual value· (cvc-elt clause 3.1, the declaration not {nillable}), an
// xsi:type that does not resolve (cvc-attribute clause 3 or 5) or
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
		{"an xsi:nil with no ·actual value·", `<known ` + xsiNS + ` xsi:nil="maybe">x</known>`, false},
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
