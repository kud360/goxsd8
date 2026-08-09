package parser_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

const produceURI = "mem://produce.xsd"

// produce reads doc and runs Produce with the strict backend, returning the
// finalized schema or the first error.
func produce(t *testing.T, doc string) (*xsd.Schema, error) {
	t.Helper()
	d, err := parser.ReadDocument(produceURI, strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	return parser.Produce(d, strict.New())
}

// wrap wraps body children inside a <schema> with the xs prefix bound and an
// optional targetNamespace.
func wrap(target, body string) string {
	tns := ""
	if target != "" {
		tns = ` targetNamespace="` + target + `" xmlns:tns="` + target + `"`
	}
	return `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + tns + `>` + body + `</xs:schema>`
}

const xsdNS = "http://www.w3.org/2001/XMLSchema"

// declaredTypeName returns the expanded name a declaration's {type definition}
// slot references, failing the test when the slot is not the by-name arm — the
// shape every type=/default assertion in this package makes. An inline anonymous
// type (xsd.InlineTypeDefinition) is asserted on directly instead, never through
// this helper.
func declaredTypeName(t *testing.T, ref xsd.TypeDefinitionOrRef) xsd.QName {
	t.Helper()
	byName, ok := ref.(xsd.TypeDefinitionRef)
	if !ok {
		t.Fatalf("{type definition} = %#v, want a by-name xsd.TypeDefinitionRef", ref)
	}
	return byName.Name
}

func TestProduceTopLevelElement(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:element name="root" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Local: "root"})
	if !ok {
		t.Fatalf("element root not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("type = %s, want {xs}string", got)
	}
	if ed.ScopeVariety() != xsd.ScopeGlobal {
		t.Fatalf("scope = %s, want global", ed.ScopeVariety())
	}
}

func TestProduceTopLevelAttribute(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:attribute name="count" type="xs:int"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ad, ok := s.Attribute(xsd.QName{Local: "count"})
	if !ok {
		t.Fatalf("attribute count not found")
	}
	if got := declaredTypeName(t, ad.TypeDefinition()); got != (xsd.QName{Space: xsdNS, Local: "int"}) {
		t.Fatalf("type = %s, want {xs}int", got)
	}
}

func TestProduceElementTargetNamespace(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `<xs:element name="root" type="xs:string"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Space: "urn:x", Local: "root"}); !ok {
		t.Fatalf("element {urn:x}root not found; targetNamespace not applied to {name}")
	}
}

func TestProduceSimpleTypeWithFacetAndBackReference(t *testing.T) {
	// A named simpleType, then an element referencing it (backward reference),
	// proving resolution through finalize.
	body := `<xs:simpleType name="Foo"><xs:restriction base="xs:string"><xs:minLength value="1"/></xs:restriction></xs:simpleType>` +
		`<xs:element name="e" type="tns:Foo"/>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Space: "urn:x", Local: "Foo"})
	if !ok {
		t.Fatalf("type {urn:x}Foo not found")
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("Foo is not a *SimpleType")
	}
	if base := st.Base(); base == nil || base.Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("Foo base = %v, want {xs}string", base)
	}
	// The base's own primitive pointer must be propagated (warden finding #4).
	at, ok := st.Variety().(xsd.Atomic)
	if !ok {
		t.Fatalf("Foo variety = %T, want Atomic", st.Variety())
	}
	if at.Primitive() == nil || at.Primitive().Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("Foo {primitive} = %v, want {xs}string", at.Primitive())
	}
	if fs := st.OwnFacets(); len(fs) != 1 || fs[0].Kind() != xsd.FacetMinLength {
		t.Fatalf("Foo own facets = %v, want one minLength", fs)
	}
}

// simpleTypeOf produces doc's body and returns the named simple type.
func simpleTypeOf(t *testing.T, name, body string) *xsd.SimpleType {
	t.Helper()
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: name})
	if !ok {
		t.Fatalf("type %s not found", name)
	}
	st, ok := td.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("type %s = %T, want *xsd.SimpleType", name, td)
	}
	return st
}

// TestProduceSiblingPatternsFoldIntoOneFacet pins xr-pattern (§4.3.4.2): the
// <pattern> children of one <restriction> are branches of a SINGLE pattern facet
// (same-step OR), not one facet each — one-per-sibling both misreads them as
// separate ANDed steps (cvc-pattern-valid §4.3.4.4) and, being two same-kind
// ownFacets, is rejected by st-props-correct clause 4 before that.
func TestProduceSiblingPatternsFoldIntoOneFacet(t *testing.T) {
	st := simpleTypeOf(t, "st", `<xs:simpleType name="st">
	  <xs:restriction base="xs:string">
	    <xs:minLength value="1"/>
	    <xs:pattern value="[a-z]+"/>
	    <xs:pattern value="[0-9]+"/>
	    <xs:maxLength value="8"/>
	  </xs:restriction>
	</xs:simpleType>`)

	facets := st.OwnFacets()
	kinds := make([]xsd.FacetKind, 0, len(facets))
	for _, f := range facets {
		kinds = append(kinds, f.Kind())
	}
	want := []xsd.FacetKind{xsd.FacetMinLength, xsd.FacetPattern, xsd.FacetMaxLength}
	if !slices.Equal(kinds, want) {
		t.Fatalf("facet kinds = %v, want %v (one folded pattern facet, in document order)", kinds, want)
	}
	if got := facets[1].Values(); !slices.Equal(got, []string{"[a-z]+", "[0-9]+"}) {
		t.Fatalf("pattern {value} = %q, want both branches in document order", got)
	}

	for _, lex := range []string{"abc", "123"} {
		if _, err := value.ValidateLexical(strict.New(), st, lex, nil); err != nil {
			t.Errorf("ValidateLexical(%q) = %v, want accept (matches one same-step pattern)", lex, err)
		}
	}
	if _, err := value.ValidateLexical(strict.New(), st, "a1", nil); err == nil {
		t.Error(`ValidateLexical("a1") = nil, want a cvc-pattern-valid rejection (matches neither branch)`)
	}
}

// TestProduceSinglePatternUnchanged keeps the one-<pattern> case at exactly one
// {value} member (xr-pattern case 1: R is that value).
func TestProduceSinglePatternUnchanged(t *testing.T) {
	st := simpleTypeOf(t, "st", `<xs:simpleType name="st">
	  <xs:restriction base="xs:string"><xs:pattern value="[a-z]+"/></xs:restriction>
	</xs:simpleType>`)

	facets := st.OwnFacets()
	if len(facets) != 1 || facets[0].Kind() != xsd.FacetPattern {
		t.Fatalf("own facets = %v, want one pattern facet", facets)
	}
	if got := facets[0].Values(); !slices.Equal(got, []string{"[a-z]+"}) {
		t.Fatalf("pattern {value} = %q, want [[a-z]+]", got)
	}
	if _, err := value.ValidateLexical(strict.New(), st, "abc", nil); err != nil {
		t.Errorf(`ValidateLexical("abc") = %v, want accept`, err)
	}
	if _, err := value.ValidateLexical(strict.New(), st, "ABC", nil); err == nil {
		t.Error(`ValidateLexical("ABC") = nil, want a cvc-pattern-valid rejection`)
	}
}

// TestProduceCrossStepPatternsStillANDed guards the other half of xr-pattern:
// folding same-step siblings must not fold ACROSS derivation steps, which stay
// separate effective facets and are ANDed.
func TestProduceCrossStepPatternsStillANDed(t *testing.T) {
	st := simpleTypeOf(t, "derived", `<xs:simpleType name="base">
	  <xs:restriction base="xs:string">
	    <xs:pattern value="[a-z]+"/>
	    <xs:pattern value="[0-9]+"/>
	  </xs:restriction>
	</xs:simpleType>
	<xs:simpleType name="derived">
	  <xs:restriction base="base"><xs:pattern value=".{3}"/></xs:restriction>
	</xs:simpleType>`)

	patterns := 0
	for _, ef := range st.EffectiveFacets() {
		if ef.Facet().Kind() == xsd.FacetPattern {
			patterns++
		}
	}
	if patterns != 2 {
		t.Fatalf("effective pattern facets = %d, want 2 (one per derivation step)", patterns)
	}
	if _, err := value.ValidateLexical(strict.New(), st, "abc", nil); err != nil {
		t.Errorf(`ValidateLexical("abc") = %v, want accept (matches both steps)`, err)
	}
	// "ab" satisfies the base step's OR-set but not the derived step's .{3}.
	if _, err := value.ValidateLexical(strict.New(), st, "ab", nil); err == nil {
		t.Error(`ValidateLexical("ab") = nil, want rejection: cross-step patterns are ANDed`)
	}
	// "A1c" satisfies the derived step but neither base branch.
	if _, err := value.ValidateLexical(strict.New(), st, "A1c", nil); err == nil {
		t.Error(`ValidateLexical("A1c") = nil, want rejection: the base step's patterns still apply`)
	}
}

// TestProduceRestrictionFacetInterleaving pins the document-order guarantee
// restrictionFacets gives when its TWO folds — every <pattern> child into one
// pattern facet (xr-pattern §4.3.4.2) and every <assertion> child into one
// assertions facet (Datatypes §4.3.13) — are interleaved with plain one-to-one
// facets: each fold lands at the position of its KIND's FIRST child, and the
// pattern facet's {value} branches stay in document order across arbitrarily
// many intervening siblings.
//
// The cross-KIND ordering is a determinism choice of this codebase (STYLE D2),
// not a spec rule: the oracle's grounding on #314 confirms {facets} is
// spec-defined as "A set of Constraining Facet components" (std-facets) and
// §4.3 is silent on where a folded facet sits among other kinds. What IS
// spec-mandated is the ordering WITHIN each fold's own {value} — xr-pattern's
// concatenation "in order" and xr-assertions clause 2's "in document order" —
// and this test proves interleaving leaves that untouched.
//
// FAILURE CAPABILITY: reworking restrictionFacets to collect pattern values in
// the loop and splice them with a SECOND post-loop slices.Insert — mirroring
// the assertions insert, the design #214 deliberately rejected — fails the
// "assertion before patterns" and "minLength, assertion, patterns" cases: two
// independent post-loop inserts invalidate each other's recorded index, so the
// second one inserts against a slice the first already shifted. The in-loop
// facets[patternAt] rewrite is what keeps exactly one insert with nothing to
// shift against. Load-bearing across the package boundary: xsd.NewFacet COPIES
// its values argument, so the repeated NewFacet(kind, patterns, ...) inside the
// loop never aliases the growing patterns slice; if that ever changed, the
// Values() assertions here are where it surfaces.
func TestProduceRestrictionFacetInterleaving(t *testing.T) {
	// The pattern values need only be distinguishable, not meaningful regexes:
	// only their accumulation order into the folded {value} is under test.
	const (
		patternA = `<xs:pattern value="a"/>`
		patternB = `<xs:pattern value="b"/>`
		patternC = `<xs:pattern value="c"/>`
		assert   = `<xs:assertion test="true()"/>`
		minLen   = `<xs:minLength value="1"/>`
		maxLen   = `<xs:maxLength value="8"/>`
	)
	cases := []struct {
		name     string
		children string
		kinds    []xsd.FacetKind
		values   []string
	}{{
		name:     "pattern assertion pattern minLength",
		children: patternA + assert + patternB + minLen,
		kinds:    []xsd.FacetKind{xsd.FacetPattern, xsd.FacetAssertions, xsd.FacetMinLength},
		values:   []string{"a", "b"},
	}, {
		name:     "assertion pattern pattern",
		children: assert + patternA + patternB,
		kinds:    []xsd.FacetKind{xsd.FacetAssertions, xsd.FacetPattern},
		values:   []string{"a", "b"},
	}, {
		name:     "minLength assertion pattern pattern",
		children: minLen + assert + patternA + patternB,
		kinds:    []xsd.FacetKind{xsd.FacetMinLength, xsd.FacetAssertions, xsd.FacetPattern},
		values:   []string{"a", "b"},
	}, {
		name:     "minLength pattern pattern assertion",
		children: minLen + patternA + patternB + assert,
		kinds:    []xsd.FacetKind{xsd.FacetMinLength, xsd.FacetPattern, xsd.FacetAssertions},
		values:   []string{"a", "b"},
	}, {
		name:     "pattern minLength pattern maxLength pattern",
		children: patternA + minLen + patternB + maxLen + patternC,
		kinds:    []xsd.FacetKind{xsd.FacetPattern, xsd.FacetMinLength, xsd.FacetMaxLength},
		values:   []string{"a", "b", "c"},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := simpleTypeOf(t, "st", `<xs:simpleType name="st">
			  <xs:restriction base="xs:string">`+tc.children+`</xs:restriction>
			</xs:simpleType>`)

			facets := st.OwnFacets()
			kinds := make([]xsd.FacetKind, 0, len(facets))
			for _, f := range facets {
				kinds = append(kinds, f.Kind())
			}
			if !slices.Equal(kinds, tc.kinds) {
				t.Fatalf("facet kinds = %v, want %v (each fold at its kind's first child)", kinds, tc.kinds)
			}
			at := slices.Index(kinds, xsd.FacetPattern)
			if at < 0 {
				t.Fatalf("facet kinds = %v, want one folded pattern facet", kinds)
			}
			if got := facets[at].Values(); !slices.Equal(got, tc.values) {
				t.Fatalf("pattern {value} = %q, want %q in document order", got, tc.values)
			}
		})
	}
}

func TestProduceSimpleTypeForwardReferenceChain(t *testing.T) {
	// B is declared before A in document order, but A restricts B and B restricts
	// xs:string. Additionally a C forward-references A. Proves the topological
	// build resolves both directions.
	body := `<xs:simpleType name="B"><xs:restriction base="tns:A"><xs:maxLength value="9"/></xs:restriction></xs:simpleType>` +
		`<xs:simpleType name="A"><xs:restriction base="xs:string"><xs:minLength value="1"/></xs:restriction></xs:simpleType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	bTD, _ := s.Type(xsd.QName{Space: "urn:x", Local: "B"})
	bST := bTD.(*xsd.SimpleType)
	aTD, _ := s.Type(xsd.QName{Space: "urn:x", Local: "A"})
	aST := aTD.(*xsd.SimpleType)
	if bST.Base() != aST {
		t.Fatalf("B.Base() is not the same *SimpleType as A (pointer identity broken)")
	}
	if aST.Base() == nil || aST.Base().Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("A base = %v, want {xs}string", aST.Base())
	}
}

func TestProduceSimpleTypeCircularRejected(t *testing.T) {
	// A restricts B, B restricts A: a circular base chain.
	body := `<xs:simpleType name="A"><xs:restriction base="tns:B"/></xs:simpleType>` +
		`<xs:simpleType name="B"><xs:restriction base="tns:A"/></xs:simpleType>`
	_, err := produce(t, wrap("urn:x", body))
	assertRule(t, err, "st-props-correct")
}

func TestProduceRestrictionBaseAndInlineRejected(t *testing.T) {
	body := `<xs:simpleType name="Bad"><xs:restriction base="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:restriction></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceRestrictionNeitherBaseNorInlineRejected(t *testing.T) {
	body := `<xs:simpleType name="Bad"><xs:restriction/></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceSimpleTypeListRejected(t *testing.T) {
	body := `<xs:simpleType name="L"><xs:list itemType="xs:string"/></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceEnumerationFacetRejected(t *testing.T) {
	body := `<xs:simpleType name="E"><xs:restriction base="xs:string"><xs:enumeration value="a"/></xs:restriction></xs:simpleType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-simple-type")
}

func TestProduceElementTypeAndInlineRejected(t *testing.T) {
	body := `<xs:element name="e" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-element")
}

// TestProduceElementInlineSimpleTypeStillDeclined pins the surviving asymmetry on
// the GLOBAL path: #340 widened tier 1 for the inline <complexType> child only,
// so the inline <simpleType> child of a TOP-LEVEL <element> is still unproduced.
// It must fail as a LIMITATION carrying no rule — the schema is legal, and the
// src-element verdict this used to fabricate said otherwise (STYLE E2).
func TestProduceElementInlineSimpleTypeStillDeclined(t *testing.T) {
	body := `<xs:element name="e"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>`
	_, err := produce(t, wrap("", body))
	if err == nil {
		t.Fatal("Produce accepted a top-level <element> with an inline <simpleType>")
	}
	if rule, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("error carries rule %s, want a plain limitation error for a legal schema", rule)
	}
	if !strings.Contains(err.Error(), "inline <simpleType> is not yet produced") {
		t.Fatalf("error = %v, want the inline-<simpleType> limitation", err)
	}
}

func TestProduceElementDefaultAndFixedRejected(t *testing.T) {
	body := `<xs:element name="e" type="xs:string" default="a" fixed="b"/>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-element")
}

func TestProduceAttributeTypeAndInlineRejected(t *testing.T) {
	body := `<xs:attribute name="a" type="xs:string"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-attribute")
}

func TestProduceBadPrefixRejected(t *testing.T) {
	body := `<xs:element name="e" type="nope:string"/>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-resolve")
}

func TestProduceUnresolvableBaseRejected(t *testing.T) {
	body := `<xs:simpleType name="S"><xs:restriction base="tns:Missing"/></xs:simpleType>`
	_, err := produce(t, wrap("urn:x", body))
	assertRule(t, err, "src-resolve")
}

func TestProduceElementNoTypeDefaultsAnyType(t *testing.T) {
	// A bare <element> defaults its {type definition} to xs:anyType (§3.3.2.1
	// case 4). xs:anyType is now seeded as a Complex Type Definition (§3.4.7), so
	// the deferred reference discharges at finalize and the schema is accepted;
	// the element's {type definition} resolves to the seeded xs:anyType.
	s, err := produce(t, wrap("", `<xs:element name="e"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Local: "e"})
	if !ok {
		t.Fatalf("element e not found")
	}
	if got := declaredTypeName(t, ed.TypeDefinition()); got != anyTypeQN {
		t.Fatalf("type = %s, want {xs}anyType", got)
	}
	td, ok := s.Type(anyTypeQN)
	if !ok {
		t.Fatalf("xs:anyType not present in {type definitions}")
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("xs:anyType is %T, want xsd.ComplexType", td)
	}
	if ct.ContentType().Variety() != xsd.ContentMixed {
		t.Fatalf("xs:anyType {content type} variety = %s, want mixed", ct.ContentType().Variety())
	}
}

var anyTypeQN = xsd.QName{Space: xsdNS, Local: "anyType"}

func TestProduceAttributeNoTypeDefaultsAnySimpleType(t *testing.T) {
	s, err := produce(t, wrap("", `<xs:attribute name="a"/>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ad, _ := s.Attribute(xsd.QName{Local: "a"})
	if got := declaredTypeName(t, ad.TypeDefinition()); got != (xsd.QName{Space: xsdNS, Local: "anySimpleType"}) {
		t.Fatalf("type = %s, want {xs}anySimpleType", got)
	}
}

func TestProduceAnonymousInlineBaseRestriction(t *testing.T) {
	// A restriction whose base is an inline anonymous <simpleType>.
	body := `<xs:simpleType name="Wrap"><xs:restriction><xs:simpleType><xs:restriction base="xs:string"><xs:minLength value="2"/></xs:restriction></xs:simpleType><xs:maxLength value="5"/></xs:restriction></xs:simpleType>`
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, _ := s.Type(xsd.QName{Local: "Wrap"})
	st := td.(*xsd.SimpleType)
	anon := st.Base()
	if anon == nil || anon.Name() != (xsd.QName{}) {
		t.Fatalf("Wrap base is not an anonymous (zero-QName) simple type: %v", anon)
	}
	if anon.Base() == nil || anon.Base().Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("anonymous base's base = %v, want {xs}string", anon.Base())
	}
}

func TestProduceNonSchemaRootRejected(t *testing.T) {
	d, err := parser.ReadDocument(produceURI, strings.NewReader(`<notschema/>`))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	if _, err := parser.Produce(d, strict.New()); err == nil {
		t.Fatalf("Produce accepted a non-<schema> root")
	}
}

func TestProduceSkipsOutOfScope(t *testing.T) {
	// annotation, group and friends are skipped, not rejected. (complexType is no
	// longer out of scope — it is produced; see the complex-type tests below.)
	body := `<xs:annotation><xs:documentation>hi</xs:documentation></xs:annotation>` +
		`<xs:group name="g"><xs:sequence/></xs:group>` +
		`<xs:element name="e" type="xs:string"/>`
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if _, ok := s.Element(xsd.QName{Local: "e"}); !ok {
		t.Fatalf("element e not produced alongside skipped out-of-scope elements")
	}
	if _, ok := s.Type(xsd.QName{Local: "g"}); ok {
		t.Fatalf("group g should have been skipped, not produced")
	}
}

// complexType reads the produced complex type named local (no namespace) from a
// schema built from body, failing on any Produce error or a missing/ wrong-kind
// type.
func complexType(t *testing.T, body, local string) xsd.ComplexType {
	t.Helper()
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: local})
	if !ok {
		t.Fatalf("complexType %s not found", local)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("%s is %T, want xsd.ComplexType", local, td)
	}
	return ct
}

// topGroup extracts the top model group of an element-content complex type.
func topGroup(t *testing.T, ct xsd.ComplexType) xsd.ModelGroup {
	t.Helper()
	ec, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("content type = %T, want ElementContent", ct.ContentType())
	}
	rt, ok := ec.Particle.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("top term = %T, want ResolvedTerm", ec.Particle.Term())
	}
	mg, ok := rt.Term.(xsd.ModelGroup)
	if !ok {
		t.Fatalf("top term inner = %T, want ModelGroup", rt.Term)
	}
	return mg
}

func TestProduceComplexTypeEmpty(t *testing.T) {
	ct := complexType(t, `<xs:complexType name="CT"/>`, "CT")
	if ct.ContentType().Variety() != xsd.ContentEmpty {
		t.Fatalf("variety = %s, want empty", ct.ContentType().Variety())
	}
	if ct.DerivationMethod() != xsd.DerivationRestriction {
		t.Fatalf("derivation = %s, want restriction", ct.DerivationMethod())
	}
	if ct.Base() != (xsd.TypeDefinitionOrRef(xsd.TypeDefinitionRef{Name: anyTypeQN})) {
		t.Fatalf("base = %#v, want a TypeDefinitionRef naming xs:anyType", ct.Base())
	}
}

func TestProduceComplexTypeSequence(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:element name="a" type="xs:string"/>` +
		`<xs:element name="b" type="xs:int" minOccurs="0" maxOccurs="unbounded"/>` +
		`</xs:sequence></xs:complexType>`
	ct := complexType(t, body, "CT")
	if ct.ContentType().Variety() != xsd.ContentElementOnly {
		t.Fatalf("variety = %s, want element-only", ct.ContentType().Variety())
	}
	mg := topGroup(t, ct)
	if mg.Compositor() != xsd.CompositorSequence {
		t.Fatalf("compositor = %s, want sequence", mg.Compositor())
	}
	ps := mg.Particles()
	if len(ps) != 2 {
		t.Fatalf("particles = %d, want 2", len(ps))
	}
	// Second particle b: 0..unbounded, local element decl.
	if ps[1].Occurs().Min() != 0 || !ps[1].Occurs().IsUnbounded() {
		t.Fatalf("b occurs = %s, want 0..unbounded", ps[1].Occurs())
	}
	rt := ps[0].Term().(xsd.ResolvedTerm)
	ed := rt.Term.(xsd.ElementDeclaration)
	if ed.Name() != (xsd.QName{Local: "a"}) || ed.ScopeVariety() != xsd.ScopeLocal {
		t.Fatalf("a decl = %s / %s, want {}a / local", ed.Name(), ed.ScopeVariety())
	}
}

func TestProduceComplexTypeChoiceAndAll(t *testing.T) {
	for _, tc := range []struct {
		name       string
		body       string
		compositor xsd.Compositor
	}{
		{"choice", `<xs:complexType name="CT"><xs:choice><xs:element name="a" type="xs:string"/></xs:choice></xs:complexType>`, xsd.CompositorChoice},
		{"all", `<xs:complexType name="CT"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`, xsd.CompositorAll},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mg := topGroup(t, complexType(t, tc.body, "CT"))
			if mg.Compositor() != tc.compositor {
				t.Fatalf("compositor = %s, want %s", mg.Compositor(), tc.compositor)
			}
		})
	}
}

func TestProduceComplexTypeMixed(t *testing.T) {
	body := `<xs:complexType name="CT" mixed="true"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`
	ct := complexType(t, body, "CT")
	if ct.ContentType().Variety() != xsd.ContentMixed {
		t.Fatalf("variety = %s, want mixed", ct.ContentType().Variety())
	}
}

func TestProduceComplexTypeMixedEmptySynthesizesSequence(t *testing.T) {
	// mixed with no content model → an empty 1..1 sequence stands in (§3.4.2.3.3
	// clause 3.1.1), and the variety is mixed, not empty.
	ct := complexType(t, `<xs:complexType name="CT" mixed="true"/>`, "CT")
	if ct.ContentType().Variety() != xsd.ContentMixed {
		t.Fatalf("variety = %s, want mixed", ct.ContentType().Variety())
	}
	mg := topGroup(t, ct)
	if mg.Compositor() != xsd.CompositorSequence || len(mg.Particles()) != 0 {
		t.Fatalf("mixed-empty group = %s/%d, want empty sequence", mg.Compositor(), len(mg.Particles()))
	}
}

func TestProduceComplexContentRestriction(t *testing.T) {
	body := `<xs:complexType name="Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>` +
		`<xs:complexType name="CT"><xs:complexContent><xs:restriction base="tns:Base"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, _ := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	ct := td.(xsd.ComplexType)
	if ct.Base() != (xsd.TypeDefinitionOrRef(xsd.TypeDefinitionRef{Name: xsd.QName{Space: "urn:x", Local: "Base"}})) {
		t.Fatalf("base = %#v, want a TypeDefinitionRef naming {urn:x}Base", ct.Base())
	}
	if ct.ContentType().Variety() != xsd.ContentElementOnly {
		t.Fatalf("variety = %s, want element-only", ct.ContentType().Variety())
	}
}

func TestProduceElementZeroOccursElided(t *testing.T) {
	// An element and a nested group each with minOccurs=maxOccurs=0 map to no
	// component at all (§3.9.2/§3.8.2): they must not appear in {particles}.
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:element name="keep" type="xs:string"/>` +
		`<xs:element name="drop" type="xs:string" minOccurs="0" maxOccurs="0"/>` +
		`<xs:choice minOccurs="0" maxOccurs="0"><xs:element name="x" type="xs:string"/></xs:choice>` +
		`</xs:sequence></xs:complexType>`
	mg := topGroup(t, complexType(t, body, "CT"))
	ps := mg.Particles()
	if len(ps) != 1 {
		t.Fatalf("particles = %d, want 1 (zero-occurs element and group elided)", len(ps))
	}
	ed := ps[0].Term().(xsd.ResolvedTerm).Term.(xsd.ElementDeclaration)
	if ed.Name() != (xsd.QName{Local: "keep"}) {
		t.Fatalf("surviving particle = %s, want keep", ed.Name())
	}
}

func TestProduceLocalAttributeUses(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/>` +
		`<xs:attribute name="a" type="xs:string" use="required"/>` +
		`<xs:attribute name="b" type="xs:int"/>` +
		`<xs:attribute name="gone" type="xs:string" use="prohibited"/>` +
		`</xs:complexType>`
	ct := complexType(t, body, "CT")
	uses := ct.AttributeUses()
	if len(uses) != 2 {
		t.Fatalf("attribute uses = %d, want 2 (prohibited elided)", len(uses))
	}
	if !uses[0].Required() {
		t.Fatalf("use a should be required")
	}
	if uses[1].Required() {
		t.Fatalf("use b should be optional")
	}
	decl := uses[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration
	if decl.ScopeVariety() != xsd.ScopeLocal || decl.Name() != (xsd.QName{Local: "a"}) {
		t.Fatalf("a decl = %s / %s, want {}a / local", decl.Name(), decl.ScopeVariety())
	}
}

// TestProduceLocalAttributeScopeParent is the §3.2.1 sc_a round trip at the
// parser level, for both discriminants of §3.2.2.2 dcl.att.local's {parent} row:
// an <attribute> with a <complexType> ancestor names that complex type, and one
// with none — a child of a top-level <attributeGroup>, reached here through a
// ref — names the attribute group instead, NOT the complex type that referenced
// it. The two are read back off the same produced Schema, so the parent each
// declaration reports is the expanded name the container is registered under.
func TestProduceLocalAttributeScopeParent(t *testing.T) {
	body := `<xs:attributeGroup name="G"><xs:attribute name="grouped" type="xs:string"/></xs:attributeGroup>` +
		`<xs:complexType name="CT"><xs:sequence/>` +
		`<xs:attribute name="own" type="xs:string"/>` +
		`<xs:attributeGroup ref="tns:G"/>` +
		`</xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	if !ok {
		t.Fatal("complex type {urn:x}CT not found")
	}
	ct := td.(xsd.ComplexType)
	uses := ct.AttributeUses()
	if len(uses) != 2 {
		t.Fatalf("{attribute uses} has %d members, want 2 (CT's own and G's)", len(uses))
	}
	for _, tc := range []struct {
		name string
		use  xsd.AttributeUse
		want xsd.AttributeScopeParent
	}{
		{"declared in the complex type", uses[0], xsd.AttributeComplexTypeScopeParent{Name: ct.Name()}},
		{"declared in the attribute group", uses[1], xsd.AttributeGroupScopeParent{Name: xsd.QName{Space: "urn:x", Local: "G"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			decl, ok := tc.use.AttributeDeclaration().(xsd.LocalAttributeDeclaration)
			if !ok {
				t.Fatalf("{attribute declaration} = %T, want a local declaration", tc.use.AttributeDeclaration())
			}
			if decl.Declaration.ScopeVariety() != xsd.ScopeLocal {
				t.Errorf("ScopeVariety() = %s, want local (§3.2.2.2)", decl.Declaration.ScopeVariety())
			}
			got, ok := decl.Declaration.Scope().Parent()
			if !ok {
				t.Fatal("Scope().Parent() ok = false, which §3.2.1 makes Required when {variety} is local")
			}
			if got != tc.want {
				t.Errorf("Scope().Parent() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestProduceAttributeRefUse(t *testing.T) {
	body := `<xs:attribute name="g" type="xs:string"/>` +
		`<xs:complexType name="CT"><xs:sequence/><xs:attribute ref="tns:g"/></xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, _ := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	uses := td.(xsd.ComplexType).AttributeUses()
	if len(uses) != 1 {
		t.Fatalf("uses = %d, want 1", len(uses))
	}
	ref, ok := uses[0].AttributeDeclaration().(xsd.AttributeDeclarationRef)
	if !ok || ref.Name != (xsd.QName{Space: "urn:x", Local: "g"}) {
		t.Fatalf("attr use decl = %v, want ref {urn:x}g", uses[0].AttributeDeclaration())
	}
}

// TestProduceLocalAttributeUseValueConstraints pins §3.5.1 vc_au for the
// dcl.att.local mapping (§3.2.2.2): default=/fixed= on a local <attribute>
// populate the Attribute USE's own {value constraint}, in document order, while
// the co-produced sibling local declaration's own {value constraint} stays
// ·absent· unconditionally.
func TestProduceLocalAttributeUseValueConstraints(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/>` +
		`<xs:attribute name="a" type="xs:string" default="dv"/>` +
		`<xs:attribute name="b" type="xs:string" fixed="fv"/>` +
		`<xs:attribute name="c" type="xs:string"/>` +
		`</xs:complexType>`
	uses := complexType(t, body, "CT").AttributeUses()
	want := []struct {
		local   string
		present bool
		kind    xsd.ValueConstraintKind
		lexical string
	}{
		{local: "a", present: true, kind: xsd.ValueDefault, lexical: "dv"},
		{local: "b", present: true, kind: xsd.ValueFixed, lexical: "fv"},
		{local: "c"},
	}
	if len(uses) != len(want) {
		t.Fatalf("attribute uses = %d, want %d", len(uses), len(want))
	}
	for i, w := range want {
		if got := attrUseLocal(uses[i]); got != w.local {
			t.Fatalf("use %d names %q, want %q (document order)", i, got, w.local)
		}
		vc, ok := uses[i].ValueConstraint()
		if ok != w.present {
			t.Fatalf("use %s {value constraint} present = %t, want %t", w.local, ok, w.present)
		}
		if ok && (vc.Kind() != w.kind || vc.LexicalForm() != w.lexical) {
			t.Fatalf("use %s {value constraint} = %s/%q, want %s/%q", w.local, vc.Kind(), vc.LexicalForm(), w.kind, w.lexical)
		}
		decl := uses[i].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration
		if _, ok := decl.ValueConstraint(); ok {
			t.Fatalf("local declaration %s carries a {value constraint}, but dcl.att.local leaves it absent", w.local)
		}
	}
}

// TestProduceAttributeRefUseValueConstraint pins §3.5.1 vc_au for the
// ref.att.local mapping (§3.2.2.3): fixed= on the <attribute ref="..."> element
// populates that USE's own {value constraint}, leaving the referenced top-level
// declaration's own {value constraint} untouched.
func TestProduceAttributeRefUseValueConstraint(t *testing.T) {
	body := `<xs:attribute name="g" type="xs:string"/>` +
		`<xs:complexType name="CT"><xs:sequence/><xs:attribute ref="tns:g" fixed="fv"/></xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, _ := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	uses := td.(xsd.ComplexType).AttributeUses()
	if len(uses) != 1 {
		t.Fatalf("uses = %d, want 1", len(uses))
	}
	vc, ok := uses[0].ValueConstraint()
	if !ok {
		t.Fatalf("ref use {value constraint} absent, want fixed/%q", "fv")
	}
	if vc.Kind() != xsd.ValueFixed || vc.LexicalForm() != "fv" {
		t.Fatalf("ref use {value constraint} = %s/%q, want fixed/%q", vc.Kind(), vc.LexicalForm(), "fv")
	}
	decl, ok := s.Attribute(xsd.QName{Space: "urn:x", Local: "g"})
	if !ok {
		t.Fatalf("top-level attribute {urn:x}g not found")
	}
	if _, ok := decl.ValueConstraint(); ok {
		t.Fatalf("top-level declaration g gained a {value constraint} from the ref use, want absent")
	}
}

// TestProduceLocalAttributeDefaultAndFixedRejected is src-attribute clause 1
// (§3.2.3) on the attribute-use path: default and fixed must not both appear.
func TestProduceLocalAttributeDefaultAndFixedRejected(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/>` +
		`<xs:attribute name="a" type="xs:string" default="dv" fixed="fv"/></xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-attribute")
}

func TestProduceAnyAttributeWildcard(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute namespace="##other" processContents="lax"/></xs:complexType>`
	ct := complexType(t, body, "CT")
	wc, ok := ct.AttributeWildcard()
	if !ok {
		t.Fatalf("attribute wildcard absent, want present")
	}
	if wc.ProcessContents() != xsd.ProcessLax {
		t.Fatalf("processContents = %s, want lax", wc.ProcessContents())
	}
	// ##other in a no-target-namespace schema admits any present namespace but not
	// ·absent· (unqualified) names.
	if wc.AllowsName(xsd.QName{Local: "x"}) {
		t.Fatalf("##other should reject an unqualified (absent-namespace) name")
	}
	if !wc.AllowsName(xsd.QName{Space: "urn:z", Local: "x"}) {
		t.Fatalf("##other should admit a foreign-namespace name")
	}
}

func TestProduceAnyElementWildcardParticle(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence><xs:any namespace="##any" minOccurs="0" maxOccurs="unbounded"/></xs:sequence></xs:complexType>`
	mg := topGroup(t, complexType(t, body, "CT"))
	ps := mg.Particles()
	if len(ps) != 1 {
		t.Fatalf("particles = %d, want 1", len(ps))
	}
	wc, ok := ps[0].Term().(xsd.ResolvedTerm).Term.(xsd.Wildcard)
	if !ok {
		t.Fatalf("term = %T, want Wildcard", ps[0].Term().(xsd.ResolvedTerm).Term)
	}
	if !wc.AllowsName(xsd.QName{Space: "urn:z", Local: "x"}) {
		t.Fatalf("##any wildcard should admit any name")
	}
}

func TestProduceComplexContentMixedMismatchRejected(t *testing.T) {
	// src-ct clause 5: mixed on both <complexType> and <complexContent> must agree.
	body := `<xs:complexType name="Base"><xs:sequence/></xs:complexType>` +
		`<xs:complexType name="CT" mixed="true"><xs:complexContent mixed="false"><xs:restriction base="tns:Base"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`
	_, err := produce(t, wrap("urn:x", body))
	assertRule(t, err, "src-ct")
}

func TestProduceAllNestedRejected(t *testing.T) {
	// cos-all-limited: an <all> may not be nested inside a <sequence>/<choice>.
	body := `<xs:complexType name="CT"><xs:sequence><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:sequence></xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "cos-all-limited")
}

func TestProduceAllOccursGrammar(t *testing.T) {
	// The schema for schema documents restricts <all>'s OWN minOccurs/maxOccurs by
	// enumeration to 0 and 1 (§3.8.2 XML representation summary, Appendix A
	// xs:complexType "all"). §3.8.3 gives <all> no Schema Representation
	// Constraint, so an excluded value is charged the generic cvc-datatype-valid —
	// independent of cos-all-limited, which is about nesting.
	for _, tc := range []struct {
		name   string
		occurs string
		want   xsderr.Rule // empty: the document must be accepted
	}{
		{name: "absent defaults to 1", occurs: ``},
		{name: "one one", occurs: ` minOccurs="1" maxOccurs="1"`},
		{name: "zero one", occurs: ` minOccurs="0" maxOccurs="1"`},
		{name: "zero zero elides the particle", occurs: ` minOccurs="0" maxOccurs="0"`},
		{name: "minOccurs two", occurs: ` minOccurs="2" maxOccurs="2"`, want: "cvc-datatype-valid"},
		{name: "maxOccurs five", occurs: ` maxOccurs="5"`, want: "cvc-datatype-valid"},
		{name: "maxOccurs unbounded", occurs: ` maxOccurs="unbounded"`, want: "cvc-datatype-valid"},
		{name: "minOccurs two elided by maxOccurs zero", occurs: ` minOccurs="2" maxOccurs="0"`, want: "cvc-datatype-valid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `<xs:complexType name="CT"><xs:all` + tc.occurs +
				`><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`
			_, err := produce(t, wrap("", body))
			if tc.want != "" {
				assertRule(t, err, tc.want)
				return
			}
			if err != nil {
				t.Fatalf("produce: %v, want the document accepted", err)
			}
		})
	}
}

func TestProduceWildcardBothNamespaceFormsRejected(t *testing.T) {
	// src-wildcard: namespace and notNamespace must not both be present.
	body := `<xs:complexType name="CT"><xs:sequence><xs:any namespace="##any" notNamespace="urn:z"/></xs:sequence></xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-wildcard")
}

func TestProduceSimpleContentRestrictionDeclined(t *testing.T) {
	// <simpleContent><restriction> synthesizes a NEW anonymous simple type from
	// the <restriction>'s facet children (§3.4.2.2 cases 1-2) — not yet produced.
	// Declined with a non-xsderr limitation error, never a fabricated rule
	// verdict. <simpleContent><extension> (cases 3-5) IS produced; see
	// produce_extension_test.go.
	body := `<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>` +
		`<xs:complexType name="CT"><xs:simpleContent><xs:restriction base="tns:B">` +
		`<xs:maxLength value="4"/></xs:restriction></xs:simpleContent></xs:complexType>`
	_, err := produce(t, wrap("urn:x", body))
	if err == nil {
		t.Fatalf("expected a decline error for <simpleContent><restriction>, got nil")
	}
	if _, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("simpleContent restriction decline should be a plain limitation error, not an xsderr rule: %v", err)
	}
}

// assertRule fails unless err is non-nil and its first *xsderr.Error carries the
// expected rule.
func assertRule(t *testing.T, err error, want xsderr.Rule) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error charged %s, got nil", want)
	}
	got, ok := xsderr.RuleOf(err)
	if !ok {
		t.Fatalf("error %v carries no xsderr rule, want %s", err, want)
	}
	if got != want {
		t.Fatalf("error charged %s, want %s (%v)", got, want, err)
	}
}

// TestProduceNotQNameKeywords pins the §3.10.2.2 notQName mapping: the ##defined
// and ##definedSibling keyword tokens are mapped (no longer skipped) alongside
// the literal QName members, which keep their cvc-wildcard-name clause-2 effect.
func TestProduceNotQNameKeywords(t *testing.T) {
	// A target namespace is needed so the tns: prefix of the literal member
	// resolves; an unresolvable one is a hard src-resolve error, pinned below.
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:any notQName="##defined ##definedSibling tns:foo"/>` +
		`</xs:sequence></xs:complexType>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
	if !ok {
		t.Fatalf("complex type CT not found")
	}
	mg := topGroup(t, td.(xsd.ComplexType))
	wc, ok := mg.Particles()[0].Term().(xsd.ResolvedTerm).Term.(xsd.Wildcard)
	if !ok {
		t.Fatalf("term = %T, want Wildcard", mg.Particles()[0].Term())
	}
	// The literal member still governs cvc-wildcard-name (§3.10.4.2) clause 2;
	// the keywords are resolved by cvc-wildcard (§3.10.4.1) clauses 2-3, a
	// different rule at the declaration-graph layer, so they leave AllowsName
	// alone.
	if wc.AllowsName(xsd.QName{Space: "urn:x", Local: "foo"}) {
		t.Errorf("AllowsName admitted the literal notQName member {urn:x}foo")
	}
	if !wc.AllowsName(xsd.QName{Space: "urn:x", Local: "bar"}) {
		t.Errorf("AllowsName rejected a name in no half of {disallowed names}")
	}
}

// TestProduceNotQNameRejections pins that an unrecognized ## token — including
// ##definedSibling on an <anyAttribute>, whose notQName is typed xs:qnameListA
// (§3.10.2, the machine-checkable form of w-props-correct clause 5) — is a
// datatype-validity rejection, never a silent skip.
func TestProduceNotQNameRejections(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"unknown-token-on-any", `<xs:complexType name="CT"><xs:sequence><xs:any notQName="##foo"/></xs:sequence></xs:complexType>`},
		{"unknown-token-on-anyAttribute", `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute notQName="##foo"/></xs:complexType>`},
		{"definedSibling-on-anyAttribute", `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute notQName="##definedSibling"/></xs:complexType>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", c.body))
			if err == nil {
				t.Fatalf("Produce accepted %s, want a cvc-datatype-valid rejection", c.body)
			}
			if rule, ok := xsderr.RuleOf(err); !ok || rule != xsderr.Rule("cvc-datatype-valid") {
				t.Errorf("RuleOf = (%q, %v), want (%q, true)", rule, ok, "cvc-datatype-valid")
			}
		})
	}
	// ##defined stays legal on an attribute wildcard (cvc-wildcard clause 2.2).
	if _, err := produce(t, wrap("urn:x", `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute notQName="##defined"/></xs:complexType>`)); err != nil {
		t.Errorf("Produce rejected notQName=\"##defined\" on <anyAttribute>: %v", err)
	}
}

// TestProduceNotQNameLiteralMembers pins the literal-QName arm of the
// §3.10.2.2 notQName mapping: every literal member with a bound prefix lands in
// {disallowed names}, and a member whose prefix has no in-scope binding is a
// hard src-resolve (§3.17.6.2) rejection rather than a silently dropped member
// that would leave the wildcard more permissive than declared.
func TestProduceNotQNameLiteralMembers(t *testing.T) {
	t.Run("all-prefixes-bound", func(t *testing.T) {
		body := `<xs:complexType name="CT"><xs:sequence>` +
			`<xs:any notQName="tns:foo tns:bar xs:string"/>` +
			`</xs:sequence></xs:complexType>`
		s, err := produce(t, wrap("urn:x", body))
		if err != nil {
			t.Fatalf("Produce: %v", err)
		}
		td, ok := s.Type(xsd.QName{Space: "urn:x", Local: "CT"})
		if !ok {
			t.Fatalf("complex type CT not found")
		}
		mg := topGroup(t, td.(xsd.ComplexType))
		wc, ok := mg.Particles()[0].Term().(xsd.ResolvedTerm).Term.(xsd.Wildcard)
		if !ok {
			t.Fatalf("term = %T, want Wildcard", mg.Particles()[0].Term())
		}
		for _, name := range []xsd.QName{
			{Space: "urn:x", Local: "foo"},
			{Space: "urn:x", Local: "bar"},
			{Space: xsdNS, Local: "string"},
		} {
			if wc.AllowsName(name) {
				t.Errorf("AllowsName admitted the literal notQName member %v", name)
			}
		}
		if !wc.AllowsName(xsd.QName{Space: "urn:x", Local: "other"}) {
			t.Errorf("AllowsName rejected a name in no half of {disallowed names}")
		}
	})
	unbound := []struct {
		name string
		body string
	}{
		{"any", `<xs:complexType name="CT"><xs:sequence><xs:any notQName="nope:foo"/></xs:sequence></xs:complexType>`},
		{"anyAttribute", `<xs:complexType name="CT"><xs:sequence/><xs:anyAttribute notQName="nope:foo"/></xs:complexType>`},
	}
	for _, c := range unbound {
		t.Run("unbound-prefix-on-"+c.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", c.body))
			assertRule(t, err, xsderr.Rule("src-resolve"))
		})
	}
}

// TestProduceChargesFacetValueRestriction proves the producer's sole
// xsd.NewSimpleType call site is followed by builtin.CheckSimpleTypeRestriction:
// a widening bound (cos-st-restricts clause 1.3.2 via
// minInclusive-valid-restriction §4.3.10.4) and an inapplicable facet (clause
// 1.3.1 via cos-applicable-facets §4.1.5) are BOTH rejected here, neither of
// which xsd.NewSimpleType can decide on its own.
func TestProduceChargesFacetValueRestriction(t *testing.T) {
	cases := []struct {
		name     string
		body     string
		wantRule xsderr.Rule
	}{
		{
			"bound widens the base value space",
			`<xs:simpleType name="wide">
			   <xs:restriction base="xs:byte">
			     <xs:minInclusive value="-200"/>
			   </xs:restriction>
			 </xs:simpleType>`,
			"minInclusive-valid-restriction",
		},
		{
			"facet inapplicable to the primitive",
			`<xs:simpleType name="bad">
			   <xs:restriction base="xs:string">
			     <xs:maxInclusive value="5"/>
			   </xs:restriction>
			 </xs:simpleType>`,
			"cos-st-restricts",
		},
		{
			// #219/#323 regression guard: maxScale is a precisionDecimal-only
			// extension facet (xsd-precisionDecimal.md §4.2), so on xs:int it must be
			// RECOGNIZED as a facet, land in {facets}, and only then be rejected on
			// applicability grounds. A lookup that silently dropped it — the defect
			// two hand-typed name↔kind tables already shipped once — would produce no
			// error at all, so a nil err here is the false-accept, not a pass.
			"precisionDecimal extension facet inapplicable to the primitive",
			`<xs:simpleType name="scaled">
			   <xs:restriction base="xs:int">
			     <xs:maxScale value="2"/>
			   </xs:restriction>
			 </xs:simpleType>`,
			"cos-st-restricts",
		},
		{
			"length may not move across a restriction",
			`<xs:simpleType name="lenbase">
			   <xs:restriction base="xs:string">
			     <xs:length value="5"/>
			   </xs:restriction>
			 </xs:simpleType>
			 <xs:simpleType name="lenderived">
			   <xs:restriction base="tns:lenbase">
			     <xs:length value="4"/>
			   </xs:restriction>
			 </xs:simpleType>`,
			"length-valid-restriction",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:facets", c.body))
			if err == nil {
				t.Fatalf("Produce accepted an invalid restriction, want %s", c.wantRule)
			}
			got, ok := xsderr.RuleOf(err)
			if !ok || got != c.wantRule {
				t.Fatalf("rule = %q (ok=%v), want %q; err=%v", got, ok, c.wantRule, err)
			}
		})
	}
}

// TestProduceSpecialBaseRejected pins #480: neither ·special· datatype may be
// named as the {base type definition} of a user-defined simple type
// (xmlschema11-2.md §2.4.2 "No ·user-defined· datatype may have anyAtomicType as
// its ·base type·", Structures §3.16.1's parallel statement for both). Both
// rejections are st-props-correct clause 1 against the Datatypes §4.1.1 property
// tableau, reached through the {variety} the XML mapping copies off the base
// (§3.16.2.1): xs:anySimpleType's is ·absent·, which only xs:anySimpleType may
// be, and xs:anyAtomicType's carries an ·absent· {primitive type definition},
// which only xs:anyAtomicType may have. Both fire with NO facet present — the
// pre-existing facet-applicability path (cos-st-restricts clause 1.3.1) only saw
// a schema that named one, which is why three of these four cases were accepted
// before.
func TestProduceSpecialBaseRejected(t *testing.T) {
	for _, c := range []struct {
		name string
		body string
	}{
		{"anyAtomicType, no facets",
			`<xs:simpleType name="t"><xs:restriction base="xs:anyAtomicType"/></xs:simpleType>`},
		{"anyAtomicType, facet present",
			`<xs:simpleType name="t"><xs:restriction base="xs:anyAtomicType"><xs:pattern value="a"/></xs:restriction></xs:simpleType>`},
		{"anySimpleType, no facets",
			`<xs:simpleType name="t"><xs:restriction base="xs:anySimpleType"/></xs:simpleType>`},
		{"anySimpleType, facet present",
			`<xs:simpleType name="t"><xs:restriction base="xs:anySimpleType"><xs:pattern value="a"/></xs:restriction></xs:simpleType>`},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:special", c.body))
			assertRule(t, err, xsderr.Rule("st-props-correct"))
		})
	}
}

// TestProduceSpecialTypesStayUsable is the false-reject guard for #480's
// rejection: it is scoped to the {base type definition} of a ·restriction·, so
// the built-in primitives — which legitimately DO have xs:anyAtomicType as their
// base (xmlschema11-2.md §2.4.2) — must still seed and resolve, a restriction of
// one must still produce, and naming either ·special· type where a type is merely
// REFERENCED (an element's type=) must stay legal.
func TestProduceSpecialTypesStayUsable(t *testing.T) {
	s, err := produce(t, wrap("urn:special",
		`<xs:simpleType name="short"><xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction></xs:simpleType>`+
			`<xs:element name="a" type="xs:anyAtomicType"/>`+
			`<xs:element name="b" type="xs:anySimpleType"/>`))
	if err != nil {
		t.Fatalf("Produce rejected a schema that only REFERENCES the special types: %v", err)
	}
	prim, ok := s.Type(xsd.QName{Space: xsdNS, Local: "string"})
	if !ok {
		t.Fatal("xs:string is not in {type definitions}: the primitives must still seed")
	}
	st, ok := prim.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("xs:string = %T, want *xsd.SimpleType", prim)
	}
	if !st.IsPrimitive() {
		t.Error("xs:string.IsPrimitive() = false, want true: a primitive's base is still xs:anyAtomicType")
	}
}

// TestProduceAcceptsNarrowingRestriction is the false-reject guard for the same
// seam: a legitimately narrowing restriction of a builtin must still produce.
func TestProduceAcceptsNarrowingRestriction(t *testing.T) {
	_, err := produce(t, wrap("urn:facets", `<xs:simpleType name="small">
	   <xs:restriction base="xs:byte">
	     <xs:minInclusive value="0"/>
	     <xs:maxInclusive value="100"/>
	   </xs:restriction>
	 </xs:simpleType>`))
	if err != nil {
		t.Fatalf("narrowing restriction rejected: %v", err)
	}
}

// TestProduceScaleFacetsReachTheFacetSet pins xr-maxScale/xr-minScale
// (xsd-precisionDecimal.md §4.2.2/§4.3.2): the two precisionDecimal extension
// facets are plain-lexical value=/fixed= facets, so <maxScale>/<minScale>
// children must land in {facets} carrying both properties. Before this, they hit
// facetKindOf's unknown-name branch and were silently dropped, which also left
// every scale Schema Component Constraint unreachable from a parsed schema.
func TestProduceScaleFacetsReachTheFacetSet(t *testing.T) {
	st := simpleTypeOf(t, "Scaled", `<xs:simpleType name="Scaled">
	   <xs:restriction base="xs:precisionDecimal">
	     <xs:maxScale value="4" fixed="true"/>
	     <xs:minScale value="2"/>
	   </xs:restriction>
	 </xs:simpleType>`)
	want := []struct {
		kind  xsd.FacetKind
		value string
		fixed bool
	}{
		{xsd.FacetMaxScale, "4", true},
		{xsd.FacetMinScale, "2", false},
	}
	facets := st.OwnFacets()
	if len(facets) != len(want) {
		t.Fatalf("own facets = %v, want %d (maxScale, minScale)", facets, len(want))
	}
	for i, w := range want {
		got := facets[i]
		if got.Kind() != w.kind {
			t.Errorf("facet %d kind = %v, want %v", i, got.Kind(), w.kind)
		}
		if vs := got.Values(); len(vs) != 1 || vs[0] != w.value {
			t.Errorf("facet %d {value} = %v, want [%q]", i, vs, w.value)
		}
		fixed, ok := got.Fixed()
		if !ok {
			t.Errorf("facet %d reports no {fixed}, but %v carries one", i, w.kind)
		}
		if fixed != w.fixed {
			t.Errorf("facet %d {fixed} = %v, want %v", i, fixed, w.fixed)
		}
	}
}

// TestProduceChargesScaleRestriction is the end-to-end proof that the
// construction-time scale SCCs (xsd/derivation.go, issue #157) now see parsed
// input: a derived maxScale WIDER than the base's violates
// maxScale-valid-restriction (xsd-precisionDecimal.md §4.2.4) and must be
// rejected at produce time. While the facet was dropped in the producer this
// schema was silently accepted.
func TestProduceChargesScaleRestriction(t *testing.T) {
	_, err := produce(t, wrap("urn:facets", `<xs:simpleType name="tight">
	   <xs:restriction base="xs:precisionDecimal">
	     <xs:maxScale value="2"/>
	   </xs:restriction>
	 </xs:simpleType>
	 <xs:simpleType name="wider">
	   <xs:restriction base="tns:tight">
	     <xs:maxScale value="5"/>
	   </xs:restriction>
	 </xs:simpleType>`))
	if err == nil {
		t.Fatalf("Produce accepted a widening maxScale, want maxScale-valid-restriction")
	}
	got, ok := xsderr.RuleOf(err)
	if !ok || got != xsderr.Rule("maxScale-valid-restriction") {
		t.Fatalf("rule = %q (ok=%v), want %q; err=%v", got, ok, "maxScale-valid-restriction", err)
	}
}

// fixedFacetSchemas renders one single-facet <simpleType> per {fixed}-bearing
// facet family the shared reader serves — a Part-2 facet (totalDigits, §4.3.11)
// and a precisionDecimal scale facet (maxScale, xsd-precisionDecimal.md §4.2) —
// with attrs spliced verbatim into the facet's start tag. The facet element sits
// on line 3 of the document so a charge's position can be pinned. The result is a
// slice, not a map: subtest order is output (STYLE D2).
func fixedFacetSchemas(attrs string) []struct{ facet, body string } {
	schema := func(base, facet string) string {
		return "\n" + `<xs:simpleType name="st"><xs:restriction base="` + base + `">` + "\n" +
			`<xs:` + facet + ` value="4"` + attrs + "/>\n</xs:restriction></xs:simpleType>"
	}
	return []struct{ facet, body string }{
		{"totalDigits", schema("xs:decimal", "totalDigits")},
		{"maxScale", schema("xs:precisionDecimal", "maxScale")},
	}
}

// TestProduceFacetFixedActualValue pins the {fixed} property mapping shared by
// all thirteen {fixed}-bearing facets: {fixed} is "the actual value of the fixed
// [attribute], if present, otherwise false" (xsd-precisionDecimal.md §4.2.2,
// §4.3.2; Datatypes §4.3.x). "Actual value" is the xs:boolean VALUE, so the
// pre-lexical whiteSpace = collapse xs:boolean fixes (§3.3.2.3, §4.3.6) runs
// before the four-literal lexical space booleanRep (§3.3.2.2) is tested: a padded
// " true " is the value true, not a non-literal. The producer compared the RAW
// attribute string against two literals, so every padded form silently became
// false — and on a base maxScale that silently disabled f-ms-fixed.
func TestProduceFacetFixedActualValue(t *testing.T) {
	cases := []struct {
		name  string
		attrs string
		want  bool
	}{
		{"absent", "", false},
		{"true", ` fixed="true"`, true},
		{"one", ` fixed="1"`, true},
		{"false", ` fixed="false"`, false},
		{"zero", ` fixed="0"`, false},
		{"space padded true", ` fixed=" true "`, true},
		{"tab and newline padded true", ` fixed="&#x9;true&#xA;"`, true},
		{"padded zero", ` fixed="  0  "`, false},
	}
	for _, c := range cases {
		for _, s := range fixedFacetSchemas(c.attrs) {
			t.Run(s.facet+"/"+c.name, func(t *testing.T) {
				st := simpleTypeOf(t, "st", s.body)
				facets := st.OwnFacets()
				if len(facets) != 1 {
					t.Fatalf("own facets = %v, want exactly one <%s>", facets, s.facet)
				}
				fixed, ok := facets[0].Fixed()
				if !ok {
					t.Fatalf("%s reports no {fixed}, but it is a {fixed}-bearing facet", s.facet)
				}
				if fixed != c.want {
					t.Errorf("fixed=%q → {fixed} = %v, want %v", c.attrs, fixed, c.want)
				}
			})
		}
	}
}

// TestProduceFacetFixedOutOfLexicalSpaceRejected is the other half of the same
// mapping: a fixed value outside xs:boolean's lexical space is Datatype Valid
// against nothing (§4.1.4 cvc-datatype-valid) and there is no clause letting it
// default — so it is a positioned rejection, not a silent {fixed} = false. Case
// matters ("TRUE" is not a booleanRep), and collapse never rescues an empty
// value. The &#xA0; case pins §4.3.6's whitespace class: NBSP is a legal XML Char
// that collapse PRESERVES, so " true" stays outside booleanRep even though
// Go's unicode.IsSpace (and hence strings.TrimSpace) would cut it.
func TestProduceFacetFixedOutOfLexicalSpaceRejected(t *testing.T) {
	for _, lexical := range []string{"yes", "TRUE", "True", "", "  ", "2", "true false", "&#xA0;true"} {
		for _, s := range fixedFacetSchemas(` fixed="` + lexical + `"`) {
			t.Run(s.facet+"/"+lexical, func(t *testing.T) {
				_, err := produce(t, wrap("", s.body))
				assertRule(t, err, xsderr.Rule("cvc-datatype-valid"))
				loc, ok := xsderr.LocOf(err)
				if !ok {
					t.Fatalf("error %v carries no position", err)
				}
				if loc.URI != produceURI || loc.Line != 3 {
					t.Fatalf("position = %s:%d:%d, want the facet element's own line 3 of %s", loc.URI, loc.Line, loc.Col, produceURI)
				}
			})
		}
	}
}

// inlineSimpleType asserts that a {type definition} slot is the InlineTypeDefinition
// arm and returns the anonymous simple type it owns (#229).
func inlineSimpleType(t *testing.T, ref xsd.TypeDefinitionOrRef) *xsd.SimpleType {
	t.Helper()
	inline, ok := ref.(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("{type definition} = %#v, want an xsd.InlineTypeDefinition", ref)
	}
	st, ok := inline.Definition.(*xsd.SimpleType)
	if !ok {
		t.Fatalf("inline {type definition} is %T, want *xsd.SimpleType", inline.Definition)
	}
	if st.Name() != (xsd.QName{}) {
		t.Fatalf("inline {type definition} {name} = %s, want the absent (zero) QName", st.Name())
	}
	return st
}

// TestProduceLocalElementInlineSimpleType is §3.3.2.1 dcl.elt.common clause 1 for
// a LOCAL element: the <simpleType> child maps to the anonymous type that becomes
// the declaration's {type definition}, reachable only through the declaration
// (it is in no symbol table).
func TestProduceLocalElementInlineSimpleType(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:element name="code"><xs:simpleType><xs:restriction base="xs:string">` +
		`<xs:maxLength value="4"/></xs:restriction></xs:simpleType></xs:element>` +
		`</xs:sequence></xs:complexType>`
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	td, ok := s.Type(xsd.QName{Local: "CT"})
	if !ok {
		t.Fatalf("complexType CT not found")
	}
	mg := topGroup(t, td.(xsd.ComplexType))
	ed := mg.Particles()[0].Term().(xsd.ResolvedTerm).Term.(xsd.ElementDeclaration)
	if ed.Name() != (xsd.QName{Local: "code"}) {
		t.Fatalf("local element = %s, want {}code", ed.Name())
	}
	st := inlineSimpleType(t, ed.TypeDefinition())
	if st.Base() == nil || st.Base().Name() != (xsd.QName{Space: xsdNS, Local: "string"}) {
		t.Fatalf("inline type base = %v, want {xs}string", st.Base())
	}
	if got := st.OwnFacets(); len(got) != 1 || got[0].Kind() != xsd.FacetMaxLength {
		t.Fatalf("inline type own facets = %v, want one maxLength", got)
	}
	// The anonymous type is NOT registered in {type definitions}: it has no name
	// to be resolved by.
	if _, ok := s.Type(xsd.QName{}); ok {
		t.Fatalf("the anonymous inline type was registered in {type definitions}")
	}
}

// TestProduceLocalAttributeInlineSimpleType is the §3.2.2.2 dcl.att.local half:
// the first tier of the attribute's three-tier {type definition} chain.
func TestProduceLocalAttributeInlineSimpleType(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/>` +
		`<xs:attribute name="a"><xs:simpleType><xs:restriction base="xs:int">` +
		`<xs:minInclusive value="1"/></xs:restriction></xs:simpleType></xs:attribute>` +
		`</xs:complexType>`
	ct := complexType(t, body, "CT")
	uses := ct.AttributeUses()
	if len(uses) != 1 {
		t.Fatalf("attribute uses = %d, want 1", len(uses))
	}
	decl := uses[0].AttributeDeclaration().(xsd.LocalAttributeDeclaration).Declaration
	if decl.ScopeVariety() != xsd.ScopeLocal {
		t.Fatalf("decl scope = %s, want local", decl.ScopeVariety())
	}
	st := inlineSimpleType(t, decl.TypeDefinition())
	if st.Base() == nil || st.Base().Name() != (xsd.QName{Space: xsdNS, Local: "int"}) {
		t.Fatalf("inline type base = %v, want {xs}int", st.Base())
	}
}

// TestProduceLocalElementTypeAndInlineRejected is src-element clause 3 (§3.3.3)
// on the LOCAL path: a type= attribute and an inline <simpleType> child together
// are a schema-representation violation, not a case where type= silently wins.
func TestProduceLocalElementTypeAndInlineRejected(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:element name="e" type="xs:string"><xs:simpleType>` +
		`<xs:restriction base="xs:string"/></xs:simpleType></xs:element>` +
		`</xs:sequence></xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-element")
}

// TestProduceLocalAttributeTypeAndInlineRejected is the src-attribute clause 4
// (§3.2.3) half of the same rule.
func TestProduceLocalAttributeTypeAndInlineRejected(t *testing.T) {
	body := `<xs:complexType name="CT"><xs:sequence/>` +
		`<xs:attribute name="a" type="xs:string"><xs:simpleType>` +
		`<xs:restriction base="xs:string"/></xs:simpleType></xs:attribute>` +
		`</xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-attribute")
}

// TestProduceElementSubstitutionGroupHeads pins §3.3.2.1's {substitution group
// affiliations} mapping: substitutionGroup is `List of QName` (§3.3.2), so EVERY
// item is ·resolved· and carried, in the list's own lexical order (STYLE D2). The
// two heads are named through different prefixes — the target namespace's own
// prefix and the default binding — so the assertion also pins that each item goes
// through the in-scope-bindings resolver rather than being defaulted wholesale to
// the target namespace.
func TestProduceElementSubstitutionGroupHeads(t *testing.T) {
	body := `<xs:element name="h1" type="xs:string"/>` +
		`<xs:element name="h2" type="xs:string"/>` +
		`<xs:element name="member" type="xs:string" substitutionGroup="tns:h1 h2"/>`
	s, err := produce(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns="urn:x"`+
		` targetNamespace="urn:x" xmlns:tns="urn:x">`+body+`</xs:schema>`)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Space: "urn:x", Local: "member"})
	if !ok {
		t.Fatalf("element {urn:x}member not found")
	}
	want := []xsd.QName{{Space: "urn:x", Local: "h1"}, {Space: "urn:x", Local: "h2"}}
	if got := ed.SubstitutionGroupAffiliationNames(); !slices.Equal(got, want) {
		t.Fatalf("{substitution group affiliations} = %v, want %v in document order", got, want)
	}
}

// TestProduceElementSubstitutionGroupSingleHead is the one-item case, and pins
// that a bare <element> with no substitutionGroup gets the EMPTY set rather than
// a one-element slice holding the absent QName.
func TestProduceElementSubstitutionGroupSingleHead(t *testing.T) {
	body := `<xs:element name="head" type="xs:string"/>` +
		`<xs:element name="member" type="xs:string" substitutionGroup="head"/>`
	s, err := produce(t, wrap("", body))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	member, ok := s.Element(xsd.QName{Local: "member"})
	if !ok {
		t.Fatalf("element member not found")
	}
	want := []xsd.QName{{Local: "head"}}
	if got := member.SubstitutionGroupAffiliationNames(); !slices.Equal(got, want) {
		t.Fatalf("{substitution group affiliations} = %v, want %v", got, want)
	}
	head, ok := s.Element(xsd.QName{Local: "head"})
	if !ok {
		t.Fatalf("element head not found")
	}
	if got := head.SubstitutionGroupAffiliationNames(); len(got) != 0 {
		t.Fatalf("head {substitution group affiliations} = %v, want the empty set", got)
	}
}

// TestProduceElementSubstitutionGroupUnknownHeadAccepted pins §5.3 (Missing
// Sub-components) for this one slot: a substitutionGroup naming no declaration
// resolves to ·absent·, which is NOT a schema-construction error — the schema
// stands and the affiliation simply carries a name nothing answers to. W3C
// saxonData/Missing missing002 is this case, expected valid.
func TestProduceElementSubstitutionGroupUnknownHeadAccepted(t *testing.T) {
	body := `<xs:element name="member" type="xs:string" substitutionGroup="tns:missing"/>`
	s, err := produce(t, wrap("urn:x", body))
	if err != nil {
		t.Fatalf("Produce rejected an unresolvable substitutionGroup head, but §5.3 makes it ·absent·, not an error: %v", err)
	}
	ed, ok := s.Element(xsd.QName{Space: "urn:x", Local: "member"})
	if !ok {
		t.Fatalf("element {urn:x}member not found")
	}
	want := []xsd.QName{{Space: "urn:x", Local: "missing"}}
	if got := ed.SubstitutionGroupAffiliationNames(); !slices.Equal(got, want) {
		t.Fatalf("{substitution group affiliations} = %v, want the unresolved %v retained", got, want)
	}
	// The contrast that makes the exemption specific to this slot rather than a
	// blanket one: an unresolvable type= is still charged src-resolve at finalize.
	_, err = produce(t, wrap("urn:x", `<xs:element name="e" type="tns:missing"/>`))
	assertRule(t, err, "src-resolve")
}

// TestProduceElementSubstitutionGroupBadPrefixRejected pins the half the producer
// DOES decide: an item whose prefix has no in-scope binding cannot be mapped to a
// QName value at all, so it is charged src-resolve here, at the referring
// element's own position.
func TestProduceElementSubstitutionGroupBadPrefixRejected(t *testing.T) {
	body := `<xs:element name="head" type="xs:string"/>` +
		`<xs:element name="member" type="xs:string" substitutionGroup="head nope:other"/>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "src-resolve")
}

// TestProduceElementSubstitutionGroupCircularRejected proves the whole pipe is
// live end to end: with real affiliation edges flowing, finalize's Phase B
// circularity check (e-props-correct clause 5) has a graph to walk, which it
// never did while every producer passed a nil {substitution group affiliations}.
func TestProduceElementSubstitutionGroupCircularRejected(t *testing.T) {
	body := `<xs:element name="a" type="xs:string" substitutionGroup="b"/>` +
		`<xs:element name="b" type="xs:string" substitutionGroup="a"/>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "e-props-correct")
}

// TestProduceElementBlockMapped pins §3.3.2.1's {disallowed substitutions} row:
// the ·effective block value· is block=, else the <schema>'s blockDefault, else
// the empty string; "#all" names all three keywords; any other value names the
// keywords its list contains, with unrecognized items IGNORED per the row's own
// Note. The result is in the spec's canonical order whatever order the attribute
// spells it in, so one set has one encoding.
func TestProduceElementBlockMapped(t *testing.T) {
	all := []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationRestriction, xsd.DerivationSubstitution}
	for _, tc := range []struct {
		name         string
		blockDefault string
		block        string
		want         []xsd.DerivationMethod
	}{
		{name: "absent is the empty set"},
		{name: "empty block is the empty set", block: `block=""`},
		{name: "#all", block: `block="#all"`, want: all},
		{name: "one keyword", block: `block="substitution"`, want: []xsd.DerivationMethod{xsd.DerivationSubstitution}},
		{name: "canonical order not lexical", block: `block="substitution extension"`,
			want: []xsd.DerivationMethod{xsd.DerivationExtension, xsd.DerivationSubstitution}},
		{name: "unrecognized items ignored", block: `block="list union restriction"`,
			want: []xsd.DerivationMethod{xsd.DerivationRestriction}},
		{name: "blockDefault fallback", blockDefault: ` blockDefault="restriction"`,
			want: []xsd.DerivationMethod{xsd.DerivationRestriction}},
		{name: "block overrides blockDefault", blockDefault: ` blockDefault="#all"`, block: `block="extension"`,
			want: []xsd.DerivationMethod{xsd.DerivationExtension}},
		// An EMPTY block= is present, so it wins over blockDefault and maps to the
		// empty set — the case that would be lost by treating "" as absent.
		{name: "empty block overrides blockDefault", blockDefault: ` blockDefault="#all"`, block: `block=""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"` + tc.blockDefault + `>` +
				`<xs:element name="e" type="xs:string" ` + tc.block + `/></xs:schema>`
			s, err := produce(t, doc)
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			ed, ok := s.Element(xsd.QName{Local: "e"})
			if !ok {
				t.Fatalf("element e not found")
			}
			if got := ed.DisallowedSubstitutions(); !slices.Equal(got, tc.want) {
				t.Fatalf("{disallowed substitutions} = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestProduceElementBlockSubstitutionNarrowsGroup is why {disallowed
// substitutions} is mapped in the same slice as {substitution group
// affiliations}: cos-equiv-derived-ok-rec clause 2.1 reads it, so with it
// unmapped a blocking head would still admit its members and two names that do
// NOT ·overlap· would be charged cos-nonambig. It is the shape of W3C MS-Element
// elemZ028a, reduced to two declarations.
func TestProduceElementBlockSubstitutionNarrowsGroup(t *testing.T) {
	schema := func(block string) string {
		return `<xs:element name="head" type="xs:anyType" ` + block + `/>` +
			`<xs:element name="member" type="xs:anyType" substitutionGroup="head"/>` +
			// minOccurs="0" on the first makes both particles live at the start
			// state, so the two ·compete· exactly when they ·overlap·.
			`<xs:complexType name="CT"><xs:sequence>` +
			`<xs:element ref="member" minOccurs="0"/><xs:element ref="head"/>` +
			`</xs:sequence></xs:complexType>`
	}
	if _, err := produce(t, wrap("", schema(`block="substitution"`))); err != nil {
		t.Fatalf("head blocks substitution, so member is in no group of head and the two do not ·overlap·: %v", err)
	}
	// The control: drop the block and the two genuinely do overlap.
	_, err := produce(t, wrap("", schema("")))
	assertRule(t, err, "cos-nonambig")
}

// TestProduceLocalElementSubstitutionGroupRejected is e-props-correct clause 3
// (§3.3.6.1) on the LOCAL path: the attribute is use="prohibited" on
// xs:localElement (§3.3.2), and this producer runs no meta-schema validation pass
// ahead of mapping, so silently ignoring it would be a false accept. The charge
// is positioned on the local <element> itself.
func TestProduceLocalElementSubstitutionGroupRejected(t *testing.T) {
	body := `<xs:element name="head" type="xs:string"/>` +
		"\n" + `<xs:complexType name="CT"><xs:sequence>` +
		`<xs:element name="e" type="xs:string" substitutionGroup="head"/>` +
		`</xs:sequence></xs:complexType>`
	_, err := produce(t, wrap("", body))
	assertRule(t, err, "e-props-correct")
	loc, ok := xsderr.LocOf(err)
	if !ok {
		t.Fatalf("error %v carries no position", err)
	}
	if loc.URI != produceURI || loc.Line != 2 {
		t.Fatalf("position = %s:%d:%d, want the local <element>'s own line 2 of %s", loc.URI, loc.Line, loc.Col, produceURI)
	}
}
