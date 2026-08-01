package parser_test

import (
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// idcOf produces doc and returns the identity constraints of its top-level
// element named name.
func idcOf(t *testing.T, doc string, name xsd.QName) []xsd.IdentityConstraint {
	t.Helper()
	s, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(name)
	if !ok {
		t.Fatalf("element %s not found", name)
	}
	return ed.IdentityConstraints()
}

func TestProduceIdentityConstraints(t *testing.T) {
	constraints := idcOf(t, wrap("", `<xs:element name="root">
	  <xs:unique name="u">
	    <xs:selector xpath="tns:a"/>
	    <xs:field xpath="@x"/>
	    <xs:field xpath="@y"/>
	  </xs:unique>
	</xs:element>`), xsd.QName{Local: "root"})
	if len(constraints) != 1 {
		t.Fatalf("got %d identity constraints, want 1", len(constraints))
	}
	ic := constraints[0]
	if ic.Category() != xsd.IdentityConstraintUnique {
		t.Errorf("category = %s, want unique", ic.Category())
	}
	if got := ic.Name(); got != (xsd.QName{Local: "u"}) {
		t.Errorf("name = %s, want {}u", got)
	}
	if got := ic.Selector().Expression(); got != "tns:a" {
		t.Errorf("selector = %q, want %q", got, "tns:a")
	}
	fields := ic.Fields()
	if len(fields) != 2 || fields[0].Expression() != "@x" || fields[1].Expression() != "@y" {
		t.Fatalf("fields = %v, want @x then @y in document order", fields)
	}
	if _, isKeyref := ic.ReferencedKeyName(); isKeyref {
		t.Errorf("a <unique> must have no {referenced key}")
	}
}

func TestProduceIdentityConstraintsInDocumentOrder(t *testing.T) {
	constraints := idcOf(t, wrap("urn:t", `<xs:element name="root">
	  <xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key>
	  <xs:keyref name="kr" refer="tns:k"><xs:selector xpath="b"/><xs:field xpath="@ref"/></xs:keyref>
	</xs:element>`), xsd.QName{Space: "urn:t", Local: "root"})
	if len(constraints) != 2 {
		t.Fatalf("got %d identity constraints, want 2", len(constraints))
	}
	if constraints[0].Category() != xsd.IdentityConstraintKey {
		t.Errorf("constraints[0] category = %s, want key", constraints[0].Category())
	}
	ref, isKeyref := constraints[1].ReferencedKeyName()
	if !isKeyref {
		t.Fatalf("constraints[1] is not a keyref")
	}
	if want := (xsd.QName{Space: "urn:t", Local: "k"}); ref != want {
		t.Errorf("refer = %s, want %s", ref, want)
	}
}

func TestProduceIdentityConstraintOnLocalElement(t *testing.T) {
	// §3.3.2.1's Common Mapping Rules apply to local declarations too, and
	// §3.17.1 collects every constraint in the document — a duplicate name
	// between a global and a local declaration is therefore a genuine
	// sch-props-correct clause 2 collision, which proves both were registered.
	_, err := produce(t, wrap("", `<xs:element name="root">
	  <xs:key name="dup"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key>
	</xs:element>
	<xs:complexType name="ct">
	  <xs:sequence>
	    <xs:element name="local">
	      <xs:key name="dup"><xs:selector xpath="b"/><xs:field xpath="@id"/></xs:key>
	    </xs:element>
	  </xs:sequence>
	</xs:complexType>`))
	if err == nil {
		t.Fatalf("two identity constraints named dup must collide (sch-props-correct clause 2)")
	}
	assertRule(t, err, "sch-props-correct")
}

func TestProduceKeyrefResolvesAtFinalize(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:element name="root">
	  <xs:keyref name="kr" refer="missing"><xs:selector xpath="a"/><xs:field xpath="@r"/></xs:keyref>
	</xs:element>`))
	if err == nil {
		t.Fatalf("a keyref whose refer names no definition must fail src-resolve clause 1.7")
	}
	assertRule(t, err, "src-resolve")
}

func TestProduceIdentityConstraintRejections(t *testing.T) {
	tests := []struct {
		name string
		body string
		rule xsderr.Rule
	}{{
		name: "no selector",
		body: `<xs:unique name="u"><xs:field xpath="@x"/></xs:unique>`,
		rule: "src-identity-constraint", // clause 2
	}, {
		name: "keyref without refer",
		body: `<xs:keyref name="kr"><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:keyref>`,
		rule: "src-identity-constraint", // clause 3
	}, {
		name: "neither name nor ref",
		body: `<xs:unique><xs:selector xpath="a"/><xs:field xpath="@x"/></xs:unique>`,
		rule: "src-identity-constraint", // clause 1
	}, {
		name: "no field",
		body: `<xs:unique name="u"><xs:selector xpath="a"/></xs:unique>`,
		rule: "c-props-correct", // clause 1: {fields} must be non-empty
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := produce(t, wrap("", `<xs:element name="root">`+tt.body+`</xs:element>`))
			assertRule(t, err, tt.rule)
		})
	}
}

func TestProduceIdentityConstraintRefForm(t *testing.T) {
	// §3.11.2: "the corresponding schema component is the identity-constraint
	// definition ·resolved· to by the ·actual value· of the ref attribute" — the
	// reference contributes the DEFINITION's own component, and the reference
	// itself contributes none. The definition here follows the reference in
	// document order, so this also pins forward resolution (§3.1.3).
	//
	// That Produce SUCCEEDS is itself the load-bearing assertion that the ref=
	// form was not registered as a second {identity-constraint definitions}
	// member: a duplicate name is a sch-props-correct clause 2 rejection at
	// finalize (TestProduceIdentityConstraintOnLocalElement pins that verdict).
	s, err := produce(t, wrap("urn:t", `<xs:element name="user"><xs:key ref="tns:k"/></xs:element>
	<xs:element name="owner">
	  <xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key>
	</xs:element>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	user, ok := s.Element(xsd.QName{Space: "urn:t", Local: "user"})
	if !ok {
		t.Fatalf("element user not found")
	}
	owner, ok := s.Element(xsd.QName{Space: "urn:t", Local: "owner"})
	if !ok {
		t.Fatalf("element owner not found")
	}
	referenced := user.IdentityConstraints()
	defined := owner.IdentityConstraints()
	if len(referenced) != 1 || len(defined) != 1 {
		t.Fatalf("got %d referenced and %d defined constraints, want 1 each", len(referenced), len(defined))
	}
	if !reflect.DeepEqual(referenced[0], defined[0]) {
		t.Fatalf("<key ref=> carries %#v, want the definition's own component %#v", referenced[0], defined[0])
	}
	// Not a hollow match: the reference borrows the definition's whole mapping.
	if got := referenced[0].Selector().Expression(); got != "a" {
		t.Errorf("referenced selector = %q, want the definition's %q", got, "a")
	}
	if got := referenced[0].Name(); got != (xsd.QName{Space: "urn:t", Local: "k"}) {
		t.Errorf("referenced name = %s, want {urn:t}k", got)
	}
}

func TestProduceIdentityConstraintRefFormOnLocalElement(t *testing.T) {
	// §3.17.2 sources {identity-constraint definitions} from the constraints
	// "anywhere within the [[children]]", so a local <element>'s ref= resolves to
	// a definition declared on another local <element> just as a top-level one
	// does — in either direction, here backwards through the content model.
	s, err := produce(t, wrap("", `<xs:complexType name="ct">
	  <xs:sequence>
	    <xs:element name="def">
	      <xs:unique name="u"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:unique>
	    </xs:element>
	    <xs:element name="use"><xs:unique ref="u"/></xs:element>
	  </xs:sequence>
	</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	constraints := localElementConstraints(t, s, xsd.QName{Local: "ct"}, "use")
	if len(constraints) != 1 {
		t.Fatalf("got %d identity constraints on <use>, want 1", len(constraints))
	}
	if got := constraints[0].Name(); got != (xsd.QName{Local: "u"}) {
		t.Errorf("name = %s, want {}u", got)
	}
	if got := constraints[0].Selector().Expression(); got != "a" {
		t.Errorf("selector = %q, want the definition's %q", got, "a")
	}
}

// localElementConstraints returns the {identity-constraint definitions} of the
// local element declaration named local in the content model of the complex type
// named typeName.
func localElementConstraints(t *testing.T, s *xsd.Schema, typeName xsd.QName, local string) []xsd.IdentityConstraint {
	t.Helper()
	def, ok := s.Type(typeName)
	if !ok {
		t.Fatalf("type %s not found", typeName)
	}
	ct, ok := def.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type", typeName)
	}
	content, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("type %s has no element-only content", typeName)
	}
	group, ok := content.Particle.Term().(xsd.ResolvedTerm).Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("type %s's {content type} particle is not a model group", typeName)
	}
	for _, part := range group.Particles() {
		term, ok := part.Term().(xsd.ResolvedTerm)
		if !ok {
			continue
		}
		decl, ok := term.Term.(xsd.ElementDeclaration)
		if !ok || decl.Name().Local != local {
			continue
		}
		return decl.IdentityConstraints()
	}
	t.Fatalf("no local element %q in type %s", local, typeName)
	return nil
}

func TestProduceIdentityConstraintRefFormRejections(t *testing.T) {
	// The definition every well-formed reference below resolves to.
	const def = `<xs:element name="owner">
	  <xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key>
	</xs:element>`
	tests := []struct {
		name string
		body string
		rule xsderr.Rule
	}{{
		name: "both name and ref",
		body: `<xs:key name="n" ref="k"/>` + def,
		rule: "src-identity-constraint", // clause 1
	}, {
		name: "selector child alongside ref",
		body: `<xs:key ref="k"><xs:selector xpath="a"/></xs:key>` + def,
		rule: "src-identity-constraint", // clause 4
	}, {
		name: "field child alongside ref",
		body: `<xs:key ref="k"><xs:field xpath="@id"/></xs:key>` + def,
		rule: "src-identity-constraint", // clause 4
	}, {
		name: "refer alongside ref",
		body: `<xs:keyref ref="k" refer="k"/>` + def,
		rule: "src-identity-constraint", // clause 4
	}, {
		name: "category mismatch",
		body: `<xs:unique ref="k"/>` + def,
		rule: "src-identity-constraint", // clause 5: k is a key, not a unique
	}, {
		name: "keyref referencing a key",
		body: `<xs:keyref ref="k"/>` + def,
		rule: "src-identity-constraint", // clause 5: ref= is same-category reuse, not refer=
	}, {
		name: "unresolvable ref",
		body: `<xs:key ref="missing"/>` + def,
		rule: "src-resolve", // clause 1.7
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := produce(t, wrap("", `<xs:element name="root">`+tt.body+`</xs:element>`))
			assertRule(t, err, tt.rule)
		})
	}
}

func TestProduceIdentityConstraintRefFormAdmitsIDAndAnnotation(t *testing.T) {
	// src-identity-constraint clause 4 admits exactly id and <annotation>
	// alongside ref; the clause-4 check must not over-reject either.
	constraints := idcOf(t, wrap("", `<xs:element name="root">
	  <xs:key ref="k" id="r1"><xs:annotation><xs:documentation>reused</xs:documentation></xs:annotation></xs:key>
	</xs:element>
	<xs:element name="owner">
	  <xs:key name="k"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key>
	</xs:element>`), xsd.QName{Local: "root"})
	if len(constraints) != 1 {
		t.Fatalf("got %d identity constraints, want 1", len(constraints))
	}
	if got := constraints[0].Name(); got != (xsd.QName{Local: "k"}) {
		t.Errorf("name = %s, want {}k", got)
	}
}

func TestProduceIdentityConstraintRefIgnoresAnnotationMarkup(t *testing.T) {
	// §3: "neither the correspondences described nor the XML Representation
	// Constraints apply to elements in the Schema namespace which occur as
	// descendants of <appinfo> or <documentation>". A <key name="k"> written in
	// prose is mapped to no component, so it must not enter the index a ref=
	// resolves against (src-resolve clause 1.7) and must not shadow the real
	// definition of k.
	//
	// The ref= comes FIRST in document order in both cases: the build-once memo
	// masks a bad index entry whenever the real definition is produced earlier,
	// so this ordering is the one that observes the index directly.
	tests := []struct {
		name   string
		shadow string
	}{{
		// Charged src-identity-constraint clause 2 against the prose <key>
		// before the fix — a false REJECT of a valid schema.
		name:   "truncated shadow",
		shadow: `<xs:key name="k"/>`,
	}, {
		// Resolved to the prose <key> before the fix — no error at all, and
		// <user> silently carried selector "FAKE".
		name:   "well-formed shadow with different selector",
		shadow: `<xs:key name="k"><xs:selector xpath="FAKE"/><xs:field xpath="@fake"/></xs:key>`,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := produce(t, wrap("", `<xs:element name="user"><xs:key ref="k"/></xs:element>
			<xs:element name="owner">
			  <xs:key name="k"><xs:selector xpath="REAL"/><xs:field xpath="@id"/></xs:key>
			</xs:element>
			<xs:element name="doc">
			  <xs:annotation><xs:documentation>`+tt.shadow+`</xs:documentation></xs:annotation>
			</xs:element>`))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			user, ok := s.Element(xsd.QName{Local: "user"})
			if !ok {
				t.Fatalf("element user not found")
			}
			constraints := user.IdentityConstraints()
			if len(constraints) != 1 {
				t.Fatalf("got %d identity constraints on <user>, want 1", len(constraints))
			}
			if got := constraints[0].Selector().Expression(); got != "REAL" {
				t.Errorf("selector = %q, want the real definition's %q", got, "REAL")
			}
			owner, ok := s.Element(xsd.QName{Local: "owner"})
			if !ok {
				t.Fatalf("element owner not found")
			}
			defined := owner.IdentityConstraints()
			if len(defined) != 1 || !reflect.DeepEqual(constraints[0], defined[0]) {
				t.Errorf("<key ref=\"k\"> carries %#v, want the real definition's component %#v", constraints[0], defined)
			}
		})
	}
}

func TestProduceIdentityConstraintRefToAnnotationOnlyNameUnresolvable(t *testing.T) {
	// The only <key name="ghost"> in the document is prose inside
	// <documentation>, so it is mapped to no component and {identity-constraint
	// definitions} has no ghost at all: the ref= must fail src-resolve clause
	// 1.7, not resolve to the illustration.
	_, err := produce(t, wrap("", `<xs:element name="doc">
	  <xs:annotation><xs:documentation>
	    <xs:key name="ghost"><xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key>
	  </xs:documentation></xs:annotation>
	</xs:element>
	<xs:element name="user"><xs:key ref="ghost"/></xs:element>`))
	assertRule(t, err, "src-resolve")
}

func TestProduceXPathExpressionProperties(t *testing.T) {
	constraints := idcOf(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
		` targetNamespace="urn:t" xmlns:tns="urn:t" xmlns="urn:d">`+
		`<xs:element name="root"><xs:unique name="u">`+
		`<xs:selector xmlns:s="urn:s" xpath="s:a"/>`+
		`<xs:field xpath="@x"/>`+
		`</xs:unique></xs:element></xs:schema>`, xsd.QName{Space: "urn:t", Local: "root"})
	selector := constraints[0].Selector()

	// {namespace bindings}: every PREFIXED in-scope binding of <selector>, sorted
	// by prefix; the default namespace (urn:d) is excluded.
	bindings := selector.NamespaceBindings()
	want := []xsd.NamespaceBinding{
		xsd.NewNamespaceBinding("s", "urn:s"),
		xsd.NewNamespaceBinding("tns", "urn:t"),
		xsd.NewNamespaceBinding("xml", "http://www.w3.org/XML/1998/namespace"),
		xsd.NewNamespaceBinding("xs", "http://www.w3.org/2001/XMLSchema"),
	}
	if len(bindings) != len(want) {
		t.Fatalf("bindings = %v, want %v", bindings, want)
	}
	for i, b := range bindings {
		if b != want[i] {
			t.Errorf("bindings[%d] = %v, want %v", i, b, want[i])
		}
	}
	// {default namespace} is absent with no xpathDefaultNamespace anywhere
	// (<schema>'s own default is ##local, §3.17.2).
	if _, ok := selector.DefaultNamespace(); ok {
		t.Errorf("{default namespace} should be absent when xpathDefaultNamespace is absent")
	}
	// {base URI} is the host element's base URI, present for every parsed element.
	if base, ok := selector.BaseURI(); !ok || base != produceURI {
		t.Errorf("{base URI} = %q, %v; want %q, true", base, ok, produceURI)
	}
}

func TestProduceXPathDefaultNamespace(t *testing.T) {
	tests := []struct {
		name       string
		schemaAttr string
		hostAttr   string
		want       string
		wantOK     bool
	}{{
		name: "absent everywhere is ##local", // §3.17.2's own default
	}, {
		name:       "schema ##targetNamespace",
		schemaAttr: ` xpathDefaultNamespace="##targetNamespace"`,
		want:       "urn:t",
		wantOK:     true,
	}, {
		name:       "host overrides schema",
		schemaAttr: ` xpathDefaultNamespace="##targetNamespace"`,
		hostAttr:   ` xpathDefaultNamespace="##local"`,
	}, {
		name:     "host ##defaultNamespace",
		hostAttr: ` xpathDefaultNamespace="##defaultNamespace"`,
		want:     "urn:d",
		wantOK:   true,
	}, {
		name:     "host literal anyURI",
		hostAttr: ` xpathDefaultNamespace="urn:literal"`,
		want:     "urn:literal",
		wantOK:   true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints := idcOf(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
				` targetNamespace="urn:t" xmlns:tns="urn:t" xmlns="urn:d"`+tt.schemaAttr+`>`+
				`<xs:element name="root"><xs:unique name="u">`+
				`<xs:selector xpath="a"`+tt.hostAttr+`/><xs:field xpath="@x"/>`+
				`</xs:unique></xs:element></xs:schema>`, xsd.QName{Space: "urn:t", Local: "root"})
			got, ok := constraints[0].Selector().DefaultNamespace()
			if ok != tt.wantOK {
				t.Fatalf("{default namespace} present = %v, want %v", ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("{default namespace} = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProduceXPathDefaultNamespaceInScope(t *testing.T) {
	// §3.13.2 {default namespace} clause 1: ##defaultNamespace is the namespace
	// name of the in-scope-namespaces entry whose prefix is absent (clause 1.1),
	// and ·absent· when there is no such entry (clause 1.2) — both when nothing
	// declares a default namespace and when xmlns="" undeclares one.
	tests := []struct {
		name        string
		schemaXMLNS string
		hostXMLNS   string
		want        string
		wantOK      bool
	}{{
		name: "no default namespace anywhere is clause 1.2",
	}, {
		name:        "default namespace undeclared on host is clause 1.2",
		schemaXMLNS: ` xmlns="urn:d"`,
		hostXMLNS:   ` xmlns=""`,
	}, {
		name:      "default namespace declared on host is clause 1.1",
		hostXMLNS: ` xmlns="urn:h"`,
		want:      "urn:h",
		wantOK:    true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constraints := idcOf(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
				tt.schemaXMLNS+`>`+
				`<xs:element name="root"><xs:unique name="u">`+
				`<xs:selector xpath="a" xpathDefaultNamespace="##defaultNamespace"`+tt.hostXMLNS+`/>`+
				`<xs:field xpath="@x"/>`+
				`</xs:unique></xs:element></xs:schema>`, xsd.QName{Local: "root"})
			got, ok := constraints[0].Selector().DefaultNamespace()
			if ok != tt.wantOK {
				t.Fatalf("{default namespace} = %q, present = %v; want present = %v", got, ok, tt.wantOK)
			}
			if ok && got != tt.want {
				t.Errorf("{default namespace} = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProduceXPathDefaultNamespaceTargetAbsent(t *testing.T) {
	// ##targetNamespace with no targetNamespace on <schema> is ·absent·, not "".
	constraints := idcOf(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+
		` xpathDefaultNamespace="##targetNamespace">`+
		`<xs:element name="root"><xs:unique name="u">`+
		`<xs:selector xpath="a"/><xs:field xpath="@x"/>`+
		`</xs:unique></xs:element></xs:schema>`, xsd.QName{Local: "root"})
	if ns, ok := constraints[0].Selector().DefaultNamespace(); ok {
		t.Errorf("{default namespace} = %q, want absent", ns)
	}
}

// complexTypeOf produces doc and returns its top-level complex type named local.
func complexTypeOf(t *testing.T, doc, local string) xsd.ComplexType {
	t.Helper()
	s, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: local})
	if !ok {
		t.Fatalf("type %s not found", local)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s = %T, want xsd.ComplexType", local, td)
	}
	return ct
}

func TestProduceComplexTypeAssertions(t *testing.T) {
	ct := complexTypeOf(t, wrap("", `<xs:complexType name="ct">
	  <xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
	  <xs:assert test="@x > 0"/>
	  <xs:assert test="@y > 0"/>
	</xs:complexType>`), "ct")
	assertions := ct.Assertions()
	if len(assertions) != 2 {
		t.Fatalf("got %d assertions, want 2", len(assertions))
	}
	if got := assertions[0].Test().Expression(); got != "@x > 0" {
		t.Errorf("assertions[0] = %q, want %q", got, "@x > 0")
	}
	if got := assertions[1].Test().Expression(); got != "@y > 0" {
		t.Errorf("assertions[1] = %q, want %q", got, "@y > 0")
	}
}

func TestProduceComplexContentRestrictionAssertions(t *testing.T) {
	ct := complexTypeOf(t, wrap("", `<xs:complexType name="base"><xs:sequence/></xs:complexType>
	<xs:complexType name="ct">
	  <xs:complexContent>
	    <xs:restriction base="base">
	      <xs:sequence/>
	      <xs:assert test="true()"/>
	    </xs:restriction>
	  </xs:complexContent>
	</xs:complexType>`), "ct")
	assertions := ct.Assertions()
	if len(assertions) != 1 || assertions[0].Test().Expression() != "true()" {
		t.Fatalf("assertions = %v, want one <assert> from <restriction>", assertions)
	}
}

func TestProduceSimpleTypeAssertionFacet(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:simpleType name="st">
	  <xs:restriction base="xs:int">
	    <xs:minInclusive value="0"/>
	    <xs:assertion test="$value mod 2 = 0"/>
	    <xs:assertion test="$value &lt; 100"/>
	    <xs:maxInclusive value="98"/>
	  </xs:restriction>
	</xs:simpleType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: "st"})
	if !ok {
		t.Fatalf("type st not found")
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("type st = %T, want *xsd.SimpleType", td)
	}
	facets := st.OwnFacets()
	// All <assertion> children fold into ONE assertions facet (§4.3.13), placed
	// where the first one occurred so the slice stays in document order.
	kinds := make([]xsd.FacetKind, 0, len(facets))
	for _, f := range facets {
		kinds = append(kinds, f.Kind())
	}
	want := []xsd.FacetKind{xsd.FacetMinInclusive, xsd.FacetAssertions, xsd.FacetMaxInclusive}
	if len(kinds) != len(want) {
		t.Fatalf("facet kinds = %v, want %v", kinds, want)
	}
	for i, k := range kinds {
		if k != want[i] {
			t.Fatalf("facet kinds = %v, want %v", kinds, want)
		}
	}
	assertions, ok := facets[1].Assertions()
	if !ok {
		t.Fatalf("facets[1] carries no assertions")
	}
	if len(assertions) != 2 || assertions[0].Test().Expression() != "$value mod 2 = 0" ||
		assertions[1].Test().Expression() != "$value < 100" {
		t.Fatalf("assertions = %v, want both tests in document order", assertions)
	}
}

func TestProduceEnumerationFacetStillDeclined(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:simpleType name="st">
	  <xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction>
	</xs:simpleType>`))
	assertRule(t, err, "src-simple-type")
}

func TestProduceNotation(t *testing.T) {
	// The finalized Schema exposes no notation accessor yet (the index is
	// populated but unread until a NOTATION-value validator exists), so
	// registration is observed through the duplicate-name rejection it enables.
	if _, err := produce(t, wrap("", `<xs:notation name="n" public="-//x//y" system="x.dtd"/>`)); err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, err := produce(t, wrap("", `<xs:notation name="n" system="x.dtd"/>`)); err != nil {
		t.Fatalf("Produce with system only: %v", err)
	}
	_, err := produce(t, wrap("", `<xs:notation name="n" system="a.dtd"/>
	<xs:notation name="n" system="b.dtd"/>`))
	if err == nil {
		t.Fatalf("two notations named n must collide (sch-props-correct clause 2)")
	}
	assertRule(t, err, "sch-props-correct")
}

func TestProduceNotationWithoutIdentifiers(t *testing.T) {
	_, err := produce(t, wrap("", `<xs:notation name="n"/>`))
	assertRule(t, err, "n-props-correct")
}
