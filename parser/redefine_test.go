package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// mustRule fails unless err is an *xsderr.Error charging want whose message
// names every clause given — the clause number is asserted because several of
// src-redefine's clauses share a rule ID and a test that only checked the ID
// would pass on the wrong verdict.
func mustRule(t *testing.T, err error, want xsderr.Rule, clauses ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("Parse succeeded, want a %s rejection", want)
	}
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("Parse error = %v (%T), want an *xsderr.Error charging %s", err, err, want)
	}
	if xe.Rule != want {
		t.Fatalf("Parse error rule = %s, want %s (%v)", xe.Rule, want, err)
	}
	for _, clause := range clauses {
		if !strings.Contains(xe.Error(), clause) {
			t.Fatalf("error %q does not name %s clause %s", xe.Error(), want, clause)
		}
	}
}

// mustModelGroup returns the assembled schema's model group definition named
// name. xsd.Schema exposes {model group definitions} only as a document-ordered
// slice (no by-name lookup has a consumer yet), so the scan is the reader.
func mustModelGroup(t *testing.T, s *xsd.Schema, name xsd.QName) xsd.ModelGroupDefinition {
	t.Helper()
	for _, mgd := range s.ModelGroups() {
		if mgd.Name() == name {
			return mgd
		}
	}
	t.Fatalf("model group definition %s not found in assembled schema", name)
	return xsd.ModelGroupDefinition{}
}

// mustAttributeGroup is mustModelGroup for {attribute group definitions}.
func mustAttributeGroup(t *testing.T, s *xsd.Schema, name xsd.QName) xsd.AttributeGroupDefinition {
	t.Helper()
	for _, ag := range s.AttributeGroups() {
		if ag.Name() == name {
			return ag
		}
	}
	t.Fatalf("attribute group definition %s not found in assembled schema", name)
	return xsd.AttributeGroupDefinition{}
}

// mustSimpleType returns the assembled schema's simple type named name.
func mustSimpleType(t *testing.T, s *xsd.Schema, name xsd.QName) *xsd.SimpleType {
	t.Helper()
	td, ok := s.Type(name)
	if !ok {
		t.Fatalf("type %s not found in assembled schema", name)
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("type %s is %T, want *xsd.SimpleType", name, td)
	}
	return st
}

// TestParseRedefineSimpleTypeResolvesToOriginal is the src-expredef pairing, and
// the test the whole slice turns on. The redefining <simpleType> restricts a
// type of its OWN name, so a naive symbol-table lookup would find the
// redefinition and report a circular derivation (st-props-correct clause 2).
// Clause 1.1 makes the base "one component which corresponds to the top-level
// definition item with the same name in the <redefine>d schema document … except
// that its {name} is ·absent·", so the assembly must instead build the ORIGINAL
// and hand it over as the base — a valid derivation, not a self-loop.
func TestParseRedefineSimpleTypeResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code">`+
			`<xs:maxLength value="2"/></xs:restriction></xs:simpleType>`+
			`</xs:redefine>`+
			`<xs:element name="root" type="tns:code"/>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"><xs:maxLength value="8"/></xs:restriction>`+
			`</xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// The VISIBLE {urn:a}code is the redefinition (§4.2.4 clause 4.1.2 excepts the
	// original from the components lib.xsd contributes), and a reference in the
	// redefining document reaches it.
	code := mustSimpleType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
	if got := facetValue(t, code, xsd.FacetMaxLength); got != "2" {
		t.Fatalf("{urn:a}code own maxLength = %q, want the redefinition's 2", got)
	}
	// Its base is the hidden original — the same name, {name} absent — carrying
	// lib.xsd's own facet. Anything else (a nil base, a base named {urn:a}code) is
	// the false circularity src-expredef's pairing exists to prevent.
	base := code.Base()
	if base == nil {
		t.Fatalf("redefined {urn:a}code has no {base type definition}")
	}
	if got := base.Name(); got != (xsd.QName{}) {
		t.Fatalf("hidden original's {name} = %s, want ·absent· (src-expredef clause 1.1)", got)
	}
	if got := facetValue(t, base, xsd.FacetMaxLength); got != "8" {
		t.Fatalf("hidden original maxLength = %q, want lib.xsd's 8", got)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "root"}, xsd.QName{Space: "urn:a", Local: "code"})
}

// TestParseRedefineIncludesUnmentionedComponents covers §4.2.4 clause 4.1.2's
// other half: everything the redefined document declares that the <redefine>
// does NOT mention comes through unchanged, exactly as a plain <include> brings
// it. A redefine that only replaced would silently lose the rest of the library.
func TestParseRedefineIncludesUnmentionedComponents(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code"/></xs:simpleType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a",
			`<xs:simpleType name="code"><xs:restriction base="xs:string"/></xs:simpleType>`+
				`<xs:element name="untouched" type="xs:date"/>`+
				`<xs:simpleType name="other"><xs:restriction base="xs:int"/></xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "other"})
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "untouched"}, xsType("date"))
}

// TestParseRedefineChameleon covers src-redefine clause 3.3: a redefined
// document with no targetNamespace of its own is coerced into the redefining
// document's namespace (§F.1), on the same terms an <include>d one is — so both
// the redefinition and the hidden original it is paired with live in urn:a.
func TestParseRedefineChameleon(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code">`+
			`<xs:maxLength value="2"/></xs:restriction></xs:simpleType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"><xs:maxLength value="8"/></xs:restriction>`+
			`</xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	code := mustSimpleType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
	if got := facetValue(t, code, xsd.FacetMaxLength); got != "2" {
		t.Fatalf("{urn:a}code maxLength = %q, want the redefinition's 2", got)
	}
	if _, ok := s.Type(xsd.QName{Local: "code"}); ok {
		t.Fatalf("the chameleon-redefined type must not also appear in the ·absent· namespace")
	}
}

// TestParseRedefineUnresolvableIsAnError is the semantic <redefine> does NOT
// share with <include>: src-redefine clause 1 makes a failed de-reference an
// error whenever the element has children other than <annotation>, where
// src-include clause 2.4 says in as many words that failing to resolve "is not
// an error". The two mechanisms are asserted side by side, on the same
// unresolvable location, so the difference cannot silently collapse.
func TestParseRedefineUnresolvableIsAnError(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="gone.xsd"/>`+
			`<xs:element name="root" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("an unresolvable <include> is not an error (src-include clause 2.4), got: %v", err)
	}

	_, err = parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="gone.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code"/></xs:simpleType>`+
			`</xs:redefine>`),
	})
	mustRule(t, err, "src-redefine", "clause 1")
}

// TestParseRedefineNoOriginalRejected covers src-expredef's closing sentence:
// "In all cases there must be a top-level definition item of the appropriate
// name and kind in the <redefine>d schema document." It is charged before
// clause 5, so redefining a name the library never declared is reported as the
// missing pairing rather than as a circular derivation.
func TestParseRedefineNoOriginalRejected(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="absent"><xs:restriction base="tns:absent"/></xs:simpleType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	mustRule(t, err, "src-expredef")
}

// TestParseRedefineWrongKindRejected pins that the pairing is per (element type,
// name), not per name: a <group> named like a top-level <simpleType> of the
// redefined document redefines nothing, so src-expredef's "of the appropriate
// name AND KIND" fires.
func TestParseRedefineWrongKindRejected(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:group name="code"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:group>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	mustRule(t, err, "src-expredef")
}

// TestParseRedefineSimpleTypeClause5Rejected covers src-redefine clause 5's
// simple-type half in both its failure modes: a <restriction> whose base is some
// OTHER type, and no <restriction> at all.
func TestParseRedefineSimpleTypeClause5Rejected(t *testing.T) {
	lib := wrap("urn:a", `<xs:simpleType name="code">`+
		`<xs:restriction base="xs:string"/></xs:simpleType>`)
	for _, tc := range []struct {
		name  string
		child string
	}{
		{"restricts another type", `<xs:simpleType name="code"><xs:restriction base="xs:string"/></xs:simpleType>`},
		{"no restriction child", `<xs:simpleType name="code"><xs:list itemType="xs:string"/></xs:simpleType>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", map[string]string{
				"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+tc.child+`</xs:redefine>`),
				"lib.xsd":  lib,
			})
			mustRule(t, err, "src-redefine")
		})
	}
}

// TestParseRedefineComplexTypeClause5Rejected covers src-redefine clause 5's
// complex-type half — "a restriction or extension among its grand-children the
// ·actual value· of whose base [attribute] must be the same as … its own name
// attribute plus target namespace". The clause is charged BEFORE the declined
// production of a redefining complex type, so the verdict is the rule the
// document breaks rather than the implementation's limit.
func TestParseRedefineComplexTypeClause5Rejected(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:complexType name="ct"><xs:complexContent><xs:extension base="tns:other">`+
			`<xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a",
			`<xs:complexType name="ct"><xs:sequence/></xs:complexType>`+
				`<xs:complexType name="other"><xs:sequence/></xs:complexType>`),
	})
	mustRule(t, err, "src-redefine")
}

// TestParseRedefineComplexTypeDeclined pins the slice's one deliberate decline
// (see parser/redefine.go's GAP header): a well-formed self-deriving redefining
// <complexType> is refused with a plain "not yet produced" error, never with a
// fabricated rule verdict and never by emitting a self-derivation. It
// over-rejects, which is the safe direction; the assertion is what will fail
// loudly when the xsd component model grows an anonymous resolved base.
func TestParseRedefineComplexTypeDeclined(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:complexType name="ct"><xs:complexContent><xs:extension base="tns:ct">`+
			`<xs:sequence><xs:element name="extra" type="xs:string"/></xs:sequence>`+
			`</xs:extension></xs:complexContent></xs:complexType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:complexType name="ct">`+
			`<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`),
	})
	if err == nil {
		t.Fatalf("Parse succeeded: a redefining <complexType> is not produced yet and must decline")
	}
	var xe *xsderr.Error
	if errors.As(err, &xe) {
		t.Fatalf("error = %v, want a plain \"not yet produced\" Go error rather than a rule verdict", err)
	}
	if !strings.Contains(err.Error(), "not yet produced") {
		t.Fatalf("error = %v, want it to say the redefining <complexType> is not yet produced", err)
	}
}

// TestParseRedefineGroupResolvesToOriginal is src-expredef clause 2 for a model
// group definition: the self-reference "is ·resolved·" to "a component which
// corresponds to the top-level definition item of that name and the appropriate
// kind in S2". The redefinition therefore contains the ORIGINAL's particles
// followed by its own — an extension by redefinition, the canonical use — rather
// than a <group ref> that would read as a circular group at finalize
// (mg-props-correct clause 2).
func TestParseRedefineGroupResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`<xs:element name="b" type="xs:string"/>`+
			`</xs:sequence></xs:group>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mgd := mustModelGroup(t, s, xsd.QName{Space: "urn:a", Local: "g"})
	particles := mgd.ModelGroup().Particles()
	if len(particles) != 2 {
		t.Fatalf("redefined {urn:a}g has %d particles, want 2", len(particles))
	}
	// The first particle is the self-reference, and it must be a RESOLVED inline
	// model group — the original's — not a by-name ModelGroupRef back to {urn:a}g.
	resolved, ok := particles[0].Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("self-reference {term} = %#v, want a resolved inline model group (the original's)", particles[0].Term())
	}
	inner, ok := resolved.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("self-reference {term} = %#v, want an xsd.ModelGroup", resolved.Term)
	}
	if got := len(inner.Particles()); got != 1 {
		t.Fatalf("inlined original has %d particles, want lib.xsd's 1", got)
	}
}

// TestParseRedefineGroupClause611Rejected covers src-redefine clause 6.1.1: with
// a qualifying self-reference present, "it has exactly one such group".
func TestParseRedefineGroupClause611Rejected(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/><xs:group ref="tns:g"/>`+
			`</xs:sequence></xs:group>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
	})
	mustRule(t, err, "src-redefine", "6.1.1")
}

// TestParseRedefineGroupClause612Rejected covers src-redefine clause 6.1.2: "the
// ·actual value· of both that group's minOccurs and maxOccurs [attribute] is 1
// (or ·absent·)". Both directions are exercised — an optional self-reference and
// a repeatable one — since a check written for only one of them would pass half
// the clause.
func TestParseRedefineGroupClause612Rejected(t *testing.T) {
	for _, occurs := range []string{`minOccurs="0"`, `maxOccurs="2"`, `maxOccurs="unbounded"`} {
		t.Run(occurs, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", map[string]string{
				"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
					`<xs:group name="g"><xs:sequence>`+
					`<xs:group ref="tns:g" `+occurs+`/>`+
					`</xs:sequence></xs:group>`+
					`</xs:redefine>`),
				"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
					`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
			})
			mustRule(t, err, "src-redefine", "6.1.2")
		})
	}
}

// TestParseRedefineGroupSelfReferenceUnderElementNotCounted is the trap a naive
// "grep for a ref= matching my own name" implementation falls into. Clause 6.1's
// precondition requires the self-referencing <group> to have NO <element>
// ancestor, so the two <group ref="tns:g"> below — one at top level, one buried
// under a local element's inline complex type — are ONE counted self-reference,
// not two: clause 6.1.1 is satisfied, and the case is accepted.
//
// The uncounted reference is nonetheless RESOLVED to the original, per
// src-expredef clause 2, whose rule carries no such exclusion — otherwise it
// would name the redefinition and be rejected as a circular <group ref> graph.
func TestParseRedefineGroupSelfReferenceUnderElementNotCounted(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:group name="g"><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`<xs:element name="nested"><xs:complexType><xs:sequence>`+
			`<xs:group ref="tns:g"/>`+
			`</xs:sequence></xs:complexType></xs:element>`+
			`</xs:sequence></xs:group>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:group name="g"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:group>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v — a self-reference under an <element> ancestor is not counted by src-redefine clause 6.1", err)
	}
	mgd := mustModelGroup(t, s, xsd.QName{Space: "urn:a", Local: "g"})
	if got := len(mgd.ModelGroup().Particles()); got != 2 {
		t.Fatalf("redefined {urn:a}g has %d particles, want 2", got)
	}
}

// TestParseRedefineAttributeGroupResolvesToOriginal is src-expredef clause 2's
// attributeGroup half: the self-reference splices in the ORIGINAL group's
// {attribute uses}, which is the whole point of redefining an attribute group by
// self-reference. Without it §3.6.2.1's ordinary inlining would meet the
// build's own visited set and contribute nothing at all.
func TestParseRedefineAttributeGroupResolvesToOriginal(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:attributeGroup name="ag">`+
			`<xs:attributeGroup ref="tns:ag"/>`+
			`<xs:attribute name="b" type="xs:string"/>`+
			`</xs:attributeGroup>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:attributeGroup name="ag">`+
			`<xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ag := mustAttributeGroup(t, s, xsd.QName{Space: "urn:a", Local: "ag"})
	if got := len(ag.AttributeUses()); got != 2 {
		t.Fatalf("redefined {urn:a}ag has %d attribute uses, want 2 (the original's a plus the redefinition's b)", got)
	}
}

// TestParseRedefineAttributeGroupClause71Rejected covers src-redefine clause
// 7.1's "then it has exactly one such group". Clause 7.1 carries NEITHER of
// clause 6.1's extra conditions, so the sibling test for 6.1.2 has deliberately
// no attributeGroup twin.
func TestParseRedefineAttributeGroupClause71Rejected(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:attributeGroup name="ag">`+
			`<xs:attributeGroup ref="tns:ag"/><xs:attributeGroup ref="tns:ag"/>`+
			`</xs:attributeGroup>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:attributeGroup name="ag">`+
			`<xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`),
	})
	mustRule(t, err, "src-redefine", "7.1")
}

// TestParseRedefineContentModelRejectsOverrideOnlyKinds pins the load-bearing
// structural difference between the two mechanisms: §4.2.4's content model is
// (annotation | (simpleType | complexType | group | attributeGroup))*, so the
// three kinds only <override> admits — <element>, <attribute>, <notation> — are
// a grammar fault here and are reported, never silently dropped.
func TestParseRedefineContentModelRejectsOverrideOnlyKinds(t *testing.T) {
	for _, child := range []string{
		`<xs:element name="doc" type="xs:string"/>`,
		`<xs:attribute name="a" type="xs:string"/>`,
		`<xs:notation name="n" public="p"/>`,
	} {
		t.Run(child, func(t *testing.T) {
			_, err := parseMap(t, "main.xsd", map[string]string{
				"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+child+`</xs:redefine>`),
				"lib.xsd":  wrap("urn:a", `<xs:element name="doc" type="xs:date"/>`),
			})
			if err == nil {
				t.Fatalf("Parse succeeded, want a content-model fault for a <redefine> child §4.2.4 does not admit")
			}
			var xe *xsderr.Error
			if errors.As(err, &xe) {
				t.Fatalf("error = %v, want a plain Go error rather than a rule verdict", err)
			}
		})
	}
}

// TestParseRedefineAndIncludeOfOneDocumentCollide covers the cross-mechanism
// conflict the load-once index deliberately does NOT paper over: the document is
// reached once plainly and once under a redefinition, and the two readings' key
// halves differ, so both are composed. §4.2.4 clause 4.1.2 makes the second
// reading's component set differ from the first's, and the duplicate expanded
// name is a genuine sch-props-correct clause 2 (c-nmd) rejection — not something
// to silently pick a winner for.
func TestParseRedefineAndIncludeOfOneDocumentCollide(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code"/></xs:simpleType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	mustRule(t, err, "sch-props-correct")
}

// TestParseRedefineSameDocumentTwiceDiffers pins the docKey widening this slice
// made: the redefine half of the load-once identity is what lets ONE document be
// redefined two different ways, each reading contributing its own component set
// (§4.2.4 clause 4.1.2). Without it the second <redefine> would silently dedup
// onto the first and its redefinition would be dropped — so this asserts the
// collision the spec's own outcome produces, which is the observable proof that
// two readings happened.
func TestParseRedefineSameDocumentTwiceDiffers(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code">`+
			`<xs:maxLength value="2"/></xs:restriction></xs:simpleType>`+
			`</xs:redefine>`+
			`<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="tns:code">`+
			`<xs:maxLength value="3"/></xs:restriction></xs:simpleType>`+
			`</xs:redefine>`),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="code">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	mustRule(t, err, "sch-props-correct")
}

// TestProduceRedefineIsSkipped pins that [Produce] — the single-document entry
// point, which follows no ·inter-schema-document reference· — still skips a
// <redefine> entirely rather than half-applying it. There is no redefined
// document, hence no original to pair with, so producing the children would
// fabricate a src-expredef verdict against a document that is fine.
func TestProduceRedefineIsSkipped(t *testing.T) {
	s, err := produce(t, wrap("urn:a", `<xs:redefine schemaLocation="lib.xsd">`+
		`<xs:simpleType name="code"><xs:restriction base="tns:code"/></xs:simpleType>`+
		`</xs:redefine>`+
		`<xs:element name="root" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "root"}); !ok {
		t.Fatalf("element {urn:a}root not found")
	}
	if _, ok := s.Type(xsd.QName{Space: "urn:a", Local: "code"}); ok {
		t.Fatalf("Produce must contribute nothing for an unfollowed <redefine>")
	}
}
