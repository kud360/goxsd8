package parser_test

import (
	"errors"
	"testing"

	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// wrapDefaults is wrap with extra <schema> attributes, for the document-level
// defaults §4.2.5 makes an <override>'s children obey (those of the OVERRIDDEN
// document, not the overriding one).
func wrapDefaults(target, extra, body string) string {
	return `<xs:schema xmlns:xs="` + xsdNS + `" targetNamespace="` + target +
		`" xmlns:tns="` + target + `" ` + extra + `>` + body + `</xs:schema>`
}

// mustElementType fails unless the assembled schema declares a global element
// named {space}local whose {type definition} reference is want.
func mustElementType(t *testing.T, s *xsd.Schema, name, want xsd.QName) {
	t.Helper()
	ed, ok := s.Element(name)
	if !ok {
		t.Fatalf("element %s not found in assembled schema", name)
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != want {
		t.Fatalf("element %s type = %s, want %s", name, got, want)
	}
}

// xsType names a builtin datatype, the target of every type= in this file.
func xsType(local string) xsd.QName { return xsd.QName{Space: xsdNS, Local: local} }

// TestParseOverrideReplacesNamedComponent is §F.2 clause 1: the overriding
// <element> replaces the identically-named source declaration of the overridden
// document, which is then produced "as if the overridden component had never
// existed" (§4.2.5).
func TestParseOverrideReplacesNamedComponent(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsType("date"))
}

// TestParseOverridePassthrough is §F.2 clause 2: a source declaration the
// override does not name is produced unchanged, alongside the replaced one.
func TestParseOverridePassthrough(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`+
			`<xs:element name="para" type="xs:string"/>`+
			`<xs:simpleType name="code"><xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsType("date"))
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "para"}, xsType("string"))
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
}

// TestParseOverrideMatchesOnElementType proves the §F.2 clause 1 key is the PAIR
// (element type, name): an <element> named "code" does not override a
// <simpleType> named "code", so the type survives and nothing is added.
func TestParseOverrideMatchesOnElementType(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="code" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("urn:a",
			`<xs:simpleType name="code"><xs:restriction base="xs:string"/></xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustType(t, s, xsd.QName{Space: "urn:a", Local: "code"})
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "code"}); ok {
		t.Fatalf("an <override> child matching nothing must be ignored (§4.2.5), not added as a new declaration")
	}
}

// TestParseOverrideUnmatchedNameIgnored is §4.2.5's "It is not an error for an
// <override> element to contain a source declaration which matches nothing in
// the target set, but it will be ignored": the assembly succeeds and gains no
// component from the unmatched child.
func TestParseOverrideUnmatchedNameIgnored(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="absent" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsType("string"))
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "absent"}); ok {
		t.Fatalf("element {urn:a}absent was added, but an unmatched <override> child must be ignored")
	}
}

// TestParseOverrideCascadesThroughInclude is §F.2 clause 3: an <include> inside
// the overridden document becomes an <override> carrying the same children, so
// the substitution reaches the whole ·target set·, not just the document the
// <override> points at directly.
func TestParseOverrideCascadesThroughInclude(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:element name="para" type="xs:date"/></xs:override>`),
		"mid.xsd":  wrap("urn:a", `<xs:include schemaLocation="leaf.xsd"/>`),
		"leaf.xsd": wrap("urn:a", `<xs:element name="para" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "para"}, xsType("date"))
}

// TestParseOverrideStopsAtImport is §F.2 clause 5: an <import> is copied
// unchanged, so §4.2.5's ·target set· "does not include schema documents which
// are pointed to by <import> … elements" and the imported namespace's
// declaration keeps its own type.
func TestParseOverrideStopsAtImport(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:element name="para" type="xs:date"/></xs:override>`),
		"mid.xsd": wrapImporting("urn:a", "urn:b",
			`<xs:import namespace="urn:b" schemaLocation="other.xsd"/>`),
		"other.xsd": wrap("urn:b", `<xs:element name="para" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:b", Local: "para"}, xsType("string"))
}

// TestParseOverrideNestedMerge is §F.2 clause 4, all three sub-clauses at once.
// main overrides mid, which itself overrides leaf:
//
//	p1  is overridden by both — main's copy wins (clause 4.1);
//	p2  is overridden only by mid — mid's copy is kept in place (clause 4.2);
//	p3  is overridden only by main — appended to the merged override (clause 4.3).
func TestParseOverrideNestedMerge(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="mid.xsd">`+
			`<xs:element name="p1" type="xs:date"/>`+
			`<xs:element name="p3" type="xs:date"/></xs:override>`),
		"mid.xsd": wrap("urn:a", `<xs:override schemaLocation="leaf.xsd">`+
			`<xs:element name="p1" type="xs:int"/>`+
			`<xs:element name="p2" type="xs:int"/></xs:override>`),
		"leaf.xsd": wrap("urn:a", `<xs:element name="p1" type="xs:string"/>`+
			`<xs:element name="p2" type="xs:string"/>`+
			`<xs:element name="p3" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "p1"}, xsType("date"))
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "p2"}, xsType("int"))
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "p3"}, xsType("date"))
}

// TestParseOverrideReplacementIsReferenceable proves the substituted declaration
// is what the assembly-wide pre-scan registers: a base= naming the overridden
// simple type reaches the OVERRIDING definition, so the facets that take effect
// are the override's.
func TestParseOverrideReplacementIsReferenceable(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:simpleType name="code"><xs:restriction base="xs:string">`+
			`<xs:maxLength value="2"/></xs:restriction></xs:simpleType></xs:override>`),
		"lib.xsd": wrap("urn:a",
			`<xs:simpleType name="code"><xs:restriction base="xs:string">`+
				`<xs:maxLength value="9"/></xs:restriction></xs:simpleType>`+
				`<xs:simpleType name="short"><xs:restriction base="tns:code"/></xs:simpleType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	td, ok := s.Type(xsd.QName{Space: "urn:a", Local: "short"})
	if !ok {
		t.Fatalf("type {urn:a}short not found")
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("type {urn:a}short is %T, want *xsd.SimpleType", td)
	}
	base := mustBase(t, s, st)
	if base == nil {
		t.Fatalf("type {urn:a}short has no base type definition")
	}
	if got := facetValue(t, base, xsd.FacetMaxLength); got != "2" {
		t.Fatalf("base {urn:a}code maxLength = %q, want the override's 2", got)
	}
}

// TestParseOverrideIdentityConstraintBelongsToOverriddenDocument proves the
// identity-constraint pre-scan attributes an <override>'s constraints to the
// document that PRODUCES them, never to the one that merely writes them down.
// §F.2 clause 1 makes an <override>'s children top-level declarations of the
// OVERRIDDEN document, so a MATCHED child's constraint is produced, registered
// and resolvable under lib.xsd's producer.
func TestParseOverrideIdentityConstraintBelongsToOverriddenDocument(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="holder"><xs:key name="k">`+
			`<xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key></xs:element>`+
			`</xs:override>`+
			`<xs:element name="user"><xs:key ref="tns:k"/></xs:element>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="holder" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Space: "urn:a", Local: "user"})
	if !ok {
		t.Fatalf("element {urn:a}user not found")
	}
	constraints := ed.IdentityConstraints()
	if len(constraints) != 1 {
		t.Fatalf("got %d identity constraints on user, want 1", len(constraints))
	}
	if got := constraints[0].Selector().Expression(); got != "a" {
		t.Errorf("selector = %q, want the substituted definition's %q", got, "a")
	}
}

// TestParseOverrideUnmatchedIdentityConstraintUnreferenceable is the other half:
// an <override> child matching NOTHING in the target set "will be ignored"
// (§4.2.5), so no producer maps it and the constraint it carries corresponds to
// no component at all. A <key ref> naming it must therefore fail src-resolve
// (clause 1.7) rather than silently borrow a definition that is in no schema's
// {identity-constraint definitions} — which is why the pre-scan withholds the
// composition directives' subtrees from the overriding document's own index.
func TestParseOverrideUnmatchedIdentityConstraintUnreferenceable(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="ghost"><xs:key name="gk">`+
			`<xs:selector xpath="a"/><xs:field xpath="@id"/></xs:key></xs:element>`+
			`</xs:override>`+
			`<xs:element name="user"><xs:key ref="tns:gk"/></xs:element>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	})
	var xe *xsderr.Error
	if !errors.As(err, &xe) || xe.Rule != "src-resolve" {
		t.Fatalf("Parse error = %v, want an *xsderr.Error with rule src-resolve", err)
	}
}

// facetValue returns the single lexical value of st's own facet of kind k.
func facetValue(t *testing.T, st *xsd.SimpleType, k xsd.FacetKind) string {
	t.Helper()
	for _, f := range st.OwnFacets() {
		if f.Kind() != k {
			continue
		}
		values := f.Values()
		if len(values) != 1 {
			t.Fatalf("facet %v has %d values, want 1", k, len(values))
		}
		return values[0]
	}
	t.Fatalf("simple type %s has no %v facet", st.Name(), k)
	return ""
}

// TestParseOverrideChameleon is src-override clause 2.3 (c-o-chamir): a
// no-targetNamespace document overridden by one that has a targetNamespace is
// chameleon-coerced first (§4.2.5 case 3.2), so the substituted declaration is
// minted in the OVERRIDING document's namespace.
func TestParseOverrideChameleon(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("", `<xs:element name="doc" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsType("date"))
	if _, ok := s.Element(xsd.QName{Local: "doc"}); ok {
		t.Fatalf("element doc must be coerced into urn:a, not left in no namespace")
	}
}

// TestParseOverrideDocumentDefaults pins §4.2.5's document-level-defaults note
// (PRINCIPLES 16): the schema-level defaults applied to an <override>'s children
// are those of the OVERRIDDEN document (Dold), not of the document containing the
// <override> (Dnew). lib.xsd sets attributeFormDefault="qualified" and main.xsd
// does not, so the overriding attribute group's local attribute must land in the
// target namespace.
func TestParseOverrideDocumentDefaults(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:attributeGroup name="g"><xs:attribute name="a" type="xs:string"/></xs:attributeGroup>`+
			`</xs:override>`),
		"lib.xsd": wrapDefaults("urn:a", `attributeFormDefault="qualified"`,
			`<xs:attributeGroup name="g"/>`+
				`<xs:complexType name="T"><xs:sequence/><xs:attributeGroup ref="tns:g"/></xs:complexType>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	uses := topComplexTypeIn(t, s, xsd.QName{Space: "urn:a", Local: "T"}).AttributeUses()
	if len(uses) != 1 {
		t.Fatalf("complex type {urn:a}T has %d attribute uses, want the overriding group's one", len(uses))
	}
	decl, ok := uses[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration)
	if !ok {
		t.Fatalf("attribute use declaration is %T, want a local declaration", uses[0].AttributeDeclaration())
	}
	if got := decl.Declaration.Name(); got != (xsd.QName{Space: "urn:a", Local: "a"}) {
		t.Fatalf("overriding attribute name = %s, want {urn:a}a from lib.xsd's attributeFormDefault", got)
	}
}

// topComplexTypeIn fetches a top-level complex type by expanded name.
func topComplexTypeIn(t *testing.T, s *xsd.Schema, name xsd.QName) xsd.ComplexType {
	t.Helper()
	td, ok := s.Type(name)
	if !ok {
		t.Fatalf("complex type %s not found", name)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is %T, want a complex type", name, td)
	}
	return ct
}

// TestParseOverrideNamespaceMismatch is src-override clause 2: Dold's
// targetNamespace must be Dnew's or absent — clauses 2.1, 2.2 and 2.3 all fail
// when it is a third namespace.
func TestParseOverrideNamespaceMismatch(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("urn:b", `<xs:element name="doc" type="xs:string"/>`),
	})
	mustXSDRule(t, err, "src-override", "main.xsd")
}

// TestParseOverrideNotASchema is src-override clause 1: the schemaLocation must
// resolve to a <schema> element information item.
func TestParseOverrideNotASchema(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": `<notASchema/>`,
	})
	mustXSDRule(t, err, "src-override", "main.xsd")
}

// TestParseOverrideNotWellFormed charges src-override on a document that DID
// resolve but is not well-formed: clause 1 requires a <schema> item "in a
// well-formed information set".
func TestParseOverrideNotWellFormed(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": `<xs:schema xmlns:xs="` + xsdNS + `"><xs:element name="doc"`,
	})
	mustXSDRule(t, err, "src-override", "main.xsd")
}

// TestParseOverrideDuplicateChildren pins the rejection of two children of one
// <override> with the same element type and name, which §F.2's normative
// stylesheet would resolve as first-match-wins. It tracks the Working Group's
// resolution on W3C Bugzilla 17617 and W3C Override/over021, which is `accepted`
// on the strength of it — not a reading of the spec as silent.
func TestParseOverrideDuplicateChildren(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/>`+
			`<xs:element name="doc" type="xs:int"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	})
	mustXSDRule(t, err, "src-override", "main.xsd")
}

// TestParseOverrideMissingSchemaLocation reports the absent required attribute as
// a plain grammar fault: the schema for schema documents requires it and
// src-override governs only what its value resolves to.
func TestParseOverrideMissingSchemaLocation(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override><xs:element name="doc" type="xs:date"/></xs:override>`),
	})
	if err == nil {
		t.Fatalf("Parse succeeded, want a grammar fault for the missing schemaLocation")
	}
	var xe *xsderr.Error
	if errors.As(err, &xe) {
		t.Fatalf("error = %v, want a plain Go error rather than a rule verdict", err)
	}
}

// TestParseOverrideUnresolvable performs no composition and is not an error:
// §4.3.2 requires the attempt to dereference, and only a NON-EMPTY <redefine>
// makes failure an error.
func TestParseOverrideUnresolvable(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="gone.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`+
			`<xs:element name="root" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "root"}, xsType("string"))
	if _, ok := s.Element(xsd.QName{Space: "urn:a", Local: "doc"}); ok {
		t.Fatalf("an unresolvable <override> must contribute nothing")
	}
}

// TestParseOverrideEmptyLoadsOnce covers §4.2.5's idempotence condition "the fact
// that E is empty": an empty <override> rewrites nothing, so the document it
// names is the same reading a plain <include> of it produces and must be loaded
// once, not twice into a sch-props-correct clause 2 collision.
func TestParseOverrideEmptyLoadsOnce(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:override schemaLocation="lib.xsd"/>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsType("string"))
}

// TestParseOverrideRepeatedIdenticallyLoadsOnce covers the same idempotence from
// the other side: the SAME <override> element reached twice — here through a
// diamond — contributes its components once (§4.2.5's note on sch-props-correct
// clause 2), which is also what terminates an <include>/<override> cycle.
func TestParseOverrideRepeatedIdenticallyLoadsOnce(t *testing.T) {
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="left.xsd"/>`+
			`<xs:include schemaLocation="right.xsd"/>`),
		"left.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"right.xsd": wrap("urn:a", `<xs:include schemaLocation="left.xsd"/>`),
		"lib.xsd":   wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsType("date"))
}

// readingsOf counts how many times the assembly of root read the document at
// resolved location loc — [parser.AssemblyReport.Documents] is a list of
// READINGS, one per docKey, so the count is exactly what the load-once index
// decided. It reports the assembly error rather than failing on it, since a
// duplicate reading is expected to be rejected.
func readingsOf(t *testing.T, root, loc string, docs map[string]string) (int, error) {
	t.Helper()
	_, report, err := parser.ParseReport(root, parser.WithResolver(loader.Map(docs)))
	n := 0
	for _, d := range report.Documents() {
		if d.Location == loc {
			n++
		}
	}
	return n, err
}

// TestParseOverrideEquivalentDistinctElementsLoadOnce is §4.2.5's note that
// "multiple equivalent overrides of the same schema document will not constitute
// a violation of clause 2 of Schema Properties Correct", for two DISTINCT
// <override> elements — one in left.xsd, one in right.xsd — declaring the same
// substitution over lib.xsd. Unlike
// TestParseOverrideRepeatedIdenticallyLoadsOnce, no single element is reached
// twice: the two agree only in content, down to an attribute order the
// serialization normalizes because XML gives it no meaning.
func TestParseOverrideEquivalentDistinctElementsLoadOnce(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="left.xsd"/>`+
			`<xs:include schemaLocation="right.xsd"/>`),
		"left.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"right.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element type="xs:date" name="doc"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	}
	s, err := parseMap(t, "main.xsd", docs)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsType("date"))

	n, err := readingsOf(t, "main.xsd", "lib.xsd", docs)
	if err != nil {
		t.Fatalf("ParseReport: %v", err)
	}
	if n != 1 {
		t.Fatalf("lib.xsd read %d times, want 1: two equivalent <override>s are one override", n)
	}
}

// TestParseOverrideDifferentSubstitutionLoadsTwice is the other half of the same
// note — "if the same schema document [is] overridden twice in different ways,
// then the resulting schema will have duplicate and conflicting versions of some
// components and will not be conforming". Content equality must not collapse two
// substitutions that transform lib.xsd differently, however alike their (element
// type, name) keys are.
func TestParseOverrideDifferentSubstitutionLoadsTwice(t *testing.T) {
	docs := map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="left.xsd"/>`+
			`<xs:include schemaLocation="right.xsd"/>`),
		"left.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"right.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:time"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	}
	_, err := parseMap(t, "main.xsd", docs)
	if err == nil {
		t.Fatalf("Parse succeeded, want a sch-props-correct clause 2 duplicate for the two readings of lib.xsd")
	}
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("error = %v, want an *xsderr.Error", err)
	}
	if xe.Rule != "sch-props-correct" {
		t.Fatalf("rule = %q, want sch-props-correct", xe.Rule)
	}

	n, _ := readingsOf(t, "main.xsd", "lib.xsd", docs)
	if n != 2 {
		t.Fatalf("lib.xsd read %d times, want 2: two different overrides are two overrides", n)
	}
}

// TestParseOverrideEquivalentTextDifferentBindingsLoadsTwice pins the same
// conservatism where the difference is invisible in the markup: both <override>s
// substitute a textually identical type="p:date", but p is bound to urn:a in one
// document and to the XSD namespace in the other, so they name DIFFERENT types
// (Datatypes §3.3.18 resolves a QName against the bindings in scope where it
// occurs) and are two different transformations of lib.xsd.
func TestParseOverrideEquivalentTextDifferentBindingsLoadsTwice(t *testing.T) {
	override := `<xs:override schemaLocation="lib.xsd">` +
		`<xs:element name="doc" type="p:date"/></xs:override>`
	docs := map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="left.xsd"/>`+
			`<xs:include schemaLocation="right.xsd"/>`),
		"left.xsd":  wrapDefaults("urn:a", `xmlns:p="urn:a"`, override),
		"right.xsd": wrapDefaults("urn:a", `xmlns:p="`+xsdNS+`"`, override),
		"lib.xsd": wrap("urn:a", `<xs:simpleType name="date">`+
			`<xs:restriction base="xs:string"/></xs:simpleType>`+
			`<xs:element name="doc" type="xs:string"/>`),
	}
	n, err := readingsOf(t, "main.xsd", "lib.xsd", docs)
	if err == nil {
		t.Fatalf("Parse succeeded, want a rejection: p:date names {urn:a}date on one side and xs:date on the other")
	}
	if n != 2 {
		t.Fatalf("lib.xsd read %d times, want 2: equal text under different prefix bindings is not an equal override", n)
	}
}

// TestParseOverrideAndIncludeConflict is §4.2.5's closing note: a document that
// is both <include>d and NON-VACUOUSLY <override>n yields "duplicate and
// conflicting versions of some components", exactly as if two schema documents
// with different declarations for one name had been included — a
// sch-props-correct clause 2 rejection.
func TestParseOverrideAndIncludeConflict(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:include schemaLocation="lib.xsd"/>`+
			`<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:element name="doc" type="xs:string"/>`),
	})
	if err == nil {
		t.Fatalf("Parse succeeded, want a sch-props-correct clause 2 duplicate for the two readings of lib.xsd")
	}
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("error = %v, want an *xsderr.Error", err)
	}
	if xe.Rule != "sch-props-correct" {
		t.Fatalf("rule = %q, want sch-props-correct", xe.Rule)
	}
}

// TestParseOverrideCycleTerminates exercises §4.2.5's explicit warning that a
// naive algorithm "may fail to terminate in the case where the graph of include
// and override references among schema documents contains cycles": main overrides
// lib, and lib includes main straight back. The load-once index keyed on
// (location, namespace, override) closes the walk.
func TestParseOverrideCycleTerminates(t *testing.T) {
	_, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="lib.xsd">`+
			`<xs:element name="doc" type="xs:date"/></xs:override>`),
		"lib.xsd": wrap("urn:a", `<xs:include schemaLocation="main.xsd"/>`+
			`<xs:element name="doc" type="xs:string"/>`),
	})
	// main.xsd is read a second time under the override, so its own <override> of
	// lib.xsd runs again; the assembly must terminate, whatever verdict it reaches.
	if err != nil {
		var xe *xsderr.Error
		if !errors.As(err, &xe) {
			t.Fatalf("Parse error = %v, want termination with a rule verdict or none", err)
		}
	}
}

// TestParseOverrideChameleonDoesNotCoerceOverrideRefs pins §4.2.5 clause 3.2.1's
// ORDERING: chameleon pre-processing runs on Dold FIRST and the override
// substitution second, so §F.1 task (b) — "updates all unqualified QName
// references so that their namespace names become the ·actual value· of the
// targetNamespace" — never reaches the <override>'s own children. Dold's own
// unqualified type= is coerced into urn:a and resolves; the override's identical
// unqualified type= stays in the ·absent· namespace and does not.
func TestParseOverrideChameleonDoesNotCoerceOverrideRefs(t *testing.T) {
	cham := `<xs:simpleType name="code"><xs:restriction base="xs:string"/></xs:simpleType>` +
		`<xs:element name="kept" type="code"/>` +
		`<xs:element name="doc" type="code"/>`

	// Prefixed in the override: bound to urn:a in main.xsd, so it resolves.
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="cham.xsd">`+
			`<xs:element name="doc" type="tns:code"/></xs:override>`),
		"cham.xsd": wrap("", cham),
	})
	if err != nil {
		t.Fatalf("Parse with a prefixed reference in the override: %v", err)
	}
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "kept"}, xsd.QName{Space: "urn:a", Local: "code"})
	mustElementType(t, s, xsd.QName{Space: "urn:a", Local: "doc"}, xsd.QName{Space: "urn:a", Local: "code"})

	// Unqualified in the override: the ·absent· namespace, where no "code" exists.
	_, err = parseMap(t, "main.xsd", map[string]string{
		"main.xsd": wrap("urn:a", `<xs:override schemaLocation="cham.xsd">`+
			`<xs:element name="doc" type="code"/></xs:override>`),
		"cham.xsd": wrap("", cham),
	})
	if err == nil {
		t.Fatalf("Parse succeeded, want the override's unqualified type= to stay in the absent namespace and fail src-resolve")
	}
	var xe *xsderr.Error
	if !errors.As(err, &xe) {
		t.Fatalf("error = %v, want an *xsderr.Error", err)
	}
	if xe.Rule != "src-resolve" {
		t.Fatalf("rule = %q, want src-resolve", xe.Rule)
	}
}
