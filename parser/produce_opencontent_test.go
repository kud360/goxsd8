package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin §3.4.2.3.3 (dcl.ctd.ctcc.common) clauses 5 and 6 — the
// ·wildcard element· selection and the {open content} fold into {content type} —
// and the src-ct (§3.4.3) clauses 3-4 that gate the source shape (#230).

// openContentOfType returns the {open content} of a top-level complex type's
// {content type}, failing when the content type is not element-only/mixed.
func openContentOfType(t *testing.T, s *xsd.Schema, local string) *xsd.OpenContent {
	t.Helper()
	return elementContentOf(t, s, xq(local)).OpenContent
}

// openContentNamespaces returns the {namespace constraint} {variety} and
// {namespaces} of an {open content}'s {wildcard}, as comparable values.
func openContentNamespaces(oc *xsd.OpenContent) (xsd.NamespaceConstraintVariety, []xsd.Namespace) {
	nc := oc.Wildcard().NamespaceConstraint()
	return nc.Variety(), nc.Namespaces()
}

// TestProduceOpenContentModes pins clause 6.2's {mode}: the mode attribute's
// ·actual value·, defaulting to interleave when the attribute is absent.
func TestProduceOpenContentModes(t *testing.T) {
	cases := []struct {
		name string
		mode string
		want xsd.OpenContentMode
	}{
		{"explicit interleave", ` mode="interleave"`, xsd.OpenContentInterleave},
		{"explicit suffix", ` mode="suffix"`, xsd.OpenContentSuffix},
		{"absent mode defaults to interleave", "", xsd.OpenContentInterleave},
	}
	for _, tc := range cases {
		s, err := produce(t, wrap("urn:x", `
			<xs:complexType name="T">
				<xs:openContent`+tc.mode+`><xs:any namespace="##other"/></xs:openContent>
				<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
			</xs:complexType>`))
		if err != nil {
			t.Fatalf("%s: Produce: %v", tc.name, err)
		}
		oc := openContentOfType(t, s, "T")
		if oc == nil {
			t.Fatalf("%s: {open content} is absent, want present", tc.name)
		}
		if oc.Mode() != tc.want {
			t.Errorf("%s: {mode} = %s, want %s", tc.name, oc.Mode(), tc.want)
		}
	}
}

// TestProduceOpenContentWildcardIsTheAnyChild pins clause 6.2's {wildcard} in the
// no-inherited-open-content case: W itself, the wildcard corresponding to the
// ·wildcard element·'s <any> child, mapped by the ordinary §3.10.2.2 producer.
func TestProduceOpenContentWildcardIsTheAnyChild(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="T">
			<xs:openContent><xs:any namespace="urn:a urn:b" processContents="lax"/></xs:openContent>
			<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
		</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	oc := openContentOfType(t, s, "T")
	if oc == nil {
		t.Fatal("{open content} is absent, want present")
	}
	if got := oc.Wildcard().ProcessContents(); got != xsd.ProcessLax {
		t.Errorf("{wildcard}.{process contents} = %v, want lax", got)
	}
	variety, spaces := openContentNamespaces(oc)
	if variety != xsd.NamespaceConstraintEnumeration {
		t.Errorf("{namespace constraint}.{variety} = %v, want enumeration", variety)
	}
	want := []xsd.Namespace{xsd.NamespaceName("urn:a"), xsd.NamespaceName("urn:b")}
	if len(spaces) != len(want) || spaces[0] != want[0] || spaces[1] != want[1] {
		t.Errorf("{namespaces} = %v, want %v", spaces, want)
	}
}

// TestProduceOpenContentModeNoneIsAbsent pins clause 6.1: a ·wildcard element·
// with mode="none" leaves {content type} exactly the ·explicit content type· —
// no Open Content record with a third mode, and no forced element-only variety
// over an empty explicit content.
func TestProduceOpenContentModeNoneIsAbsent(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="Empty"><xs:openContent mode="none"/><xs:sequence/></xs:complexType>
		<xs:complexType name="Seq">
			<xs:openContent mode="none"/>
			<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
		</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if ct := contentTypeOf(t, s, xq("Empty")); ct.Variety() != xsd.ContentEmpty {
		t.Errorf("Empty {content type} is %T (variety %v), want EmptyContent", ct, ct.Variety())
	}
	if oc := openContentOfType(t, s, "Seq"); oc != nil {
		t.Errorf("Seq {open content} = %+v, want absent (clause 6.1)", oc)
	}
}

// TestProduceOpenContentOverEmptyExplicitContent pins clause 6.2's other half: an
// ***empty*** ·explicit content type· carrying an Open Content becomes
// ELEMENT-ONLY over a synthesized 1..1 empty-sequence particle, so the wildcard
// has a content model to interleave with.
func TestProduceOpenContentOverEmptyExplicitContent(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="T"><xs:openContent><xs:any/></xs:openContent><xs:sequence/></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ec := elementContentOf(t, s, xq("T"))
	if ec.Mixed {
		t.Error("{variety} is mixed, want element-only (clause 6.2 names it outright)")
	}
	if ec.OpenContent == nil {
		t.Fatal("{open content} is absent, want present")
	}
	max, bounded := ec.Particle.Occurs().Max()
	if ec.Particle.Occurs().Min() != 1 || !bounded || max != 1 {
		t.Errorf("{particle} occurs = %d..%d (bounded=%v), want 1..1", ec.Particle.Occurs().Min(), max, bounded)
	}
	rt, ok := ec.Particle.Term().(xsd.ResolvedTerm)
	if !ok {
		t.Fatalf("{particle}.{term} is %T, want a resolved model group", ec.Particle.Term())
	}
	mg, ok := rt.Term.(xsd.ModelGroup)
	if !ok || mg.Compositor() != xsd.CompositorSequence || len(mg.Particles()) != 0 {
		t.Errorf("{particle}.{term} = %+v, want an empty sequence model group", rt.Term)
	}
}

// TestProduceOpenContentKeepsMixed pins that a NON-empty ·explicit content type·
// keeps its own {variety}: clause 6.2's element-only substitution applies to the
// empty case alone.
func TestProduceOpenContentKeepsMixed(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="T" mixed="true">
			<xs:openContent><xs:any/></xs:openContent>
			<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
		</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ec := elementContentOf(t, s, xq("T"))
	if !ec.Mixed {
		t.Error("{variety} is element-only, want mixed")
	}
	if ec.OpenContent == nil {
		t.Error("{open content} is absent, want present")
	}
}

// TestProduceDefaultOpenContentApplies pins clause 5.2.1: a type with no
// <openContent> of its own and a non-empty ·explicit content type· picks up the
// document's <defaultOpenContent>.
func TestProduceDefaultOpenContentApplies(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:defaultOpenContent mode="suffix"><xs:any namespace="urn:d"/></xs:defaultOpenContent>
		<xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	oc := openContentOfType(t, s, "T")
	if oc == nil {
		t.Fatal("{open content} is absent, want the <defaultOpenContent> fallback (clause 5.2.1)")
	}
	if oc.Mode() != xsd.OpenContentSuffix {
		t.Errorf("{mode} = %s, want suffix", oc.Mode())
	}
	variety, spaces := openContentNamespaces(oc)
	if variety != xsd.NamespaceConstraintEnumeration || len(spaces) != 1 || spaces[0] != xsd.NamespaceName("urn:d") {
		t.Errorf("{namespace constraint} = %v %v, want enumeration {urn:d}", variety, spaces)
	}
}

// TestProduceDefaultOpenContentAppliesToEmpty pins clause 5.2.2's gate: an
// ***empty*** ·explicit content type· picks the default up only when the
// <defaultOpenContent> carries appliesToEmpty="true".
func TestProduceDefaultOpenContentAppliesToEmpty(t *testing.T) {
	cases := []struct {
		name  string
		attr  string
		wantP bool
	}{
		{"appliesToEmpty absent (defaults to false)", "", false},
		{"appliesToEmpty=false", ` appliesToEmpty="false"`, false},
		{"appliesToEmpty=true", ` appliesToEmpty="true"`, true},
	}
	for _, tc := range cases {
		s, err := produce(t, wrap("urn:x", `
			<xs:defaultOpenContent`+tc.attr+`><xs:any/></xs:defaultOpenContent>
			<xs:complexType name="T"><xs:sequence/></xs:complexType>`))
		if err != nil {
			t.Fatalf("%s: Produce: %v", tc.name, err)
		}
		ct := contentTypeOf(t, s, xq("T"))
		if !tc.wantP {
			if ct.Variety() != xsd.ContentEmpty {
				t.Errorf("%s: {content type} variety = %v, want empty (clause 5.3)", tc.name, ct.Variety())
			}
			continue
		}
		if openContentOfType(t, s, "T") == nil {
			t.Errorf("%s: {open content} is absent, want the default (clause 5.2.2)", tc.name)
		}
	}
}

// TestProduceOwnOpenContentBlocksDefault pins clause 5.1's precedence: the type's
// OWN <openContent> is selected on presence alone, so an <openContent mode="none">
// is how a complex type opts out of the document's <defaultOpenContent> entirely.
func TestProduceOwnOpenContentBlocksDefault(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:defaultOpenContent><xs:any namespace="urn:d"/></xs:defaultOpenContent>
		<xs:complexType name="OptOut">
			<xs:openContent mode="none"/>
			<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
		</xs:complexType>
		<xs:complexType name="Own">
			<xs:openContent><xs:any namespace="urn:o"/></xs:openContent>
			<xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence>
		</xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if oc := openContentOfType(t, s, "OptOut"); oc != nil {
		t.Errorf("OptOut {open content} = %+v, want absent — its own mode=none wins over the default", oc)
	}
	oc := openContentOfType(t, s, "Own")
	if oc == nil {
		t.Fatal("Own {open content} is absent, want its own <openContent>")
	}
	_, spaces := openContentNamespaces(oc)
	if len(spaces) != 1 || spaces[0] != xsd.NamespaceName("urn:o") {
		t.Errorf("Own {namespaces} = %v, want {urn:o} — the default must not leak in", spaces)
	}
}

// TestProduceOpenContentSrcCT pins src-ct (§3.4.3) clauses 3 and 4 as positioned,
// mapped rule verdicts on the <openContent> element itself.
func TestProduceOpenContentSrcCT(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"clause 3: mode=interleave with no <any>", `<xs:complexType name="T"><xs:openContent mode="interleave"/><xs:sequence/></xs:complexType>`},
		{"clause 3: default mode with no <any>", `<xs:complexType name="T"><xs:openContent/><xs:sequence/></xs:complexType>`},
		{"clause 4: mode=none with an <any>", `<xs:complexType name="T"><xs:openContent mode="none"><xs:any/></xs:openContent><xs:sequence/></xs:complexType>`},
		{"clause 3 under a complexContent restriction", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:complexContent><xs:restriction base="tns:B"><xs:openContent mode="suffix"/><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
	}
	for _, tc := range cases {
		_, err := produce(t, wrap("urn:x", tc.body))
		if err == nil {
			t.Errorf("%s: Produce accepted it, want a src-ct rejection", tc.name)
			continue
		}
		rule, ok := xsderr.RuleOf(err)
		if !ok || rule != "src-ct" {
			t.Errorf("%s: rule = %q (ok=%v), want src-ct: %v", tc.name, rule, ok, err)
		}
	}
}

// TestProduceOpenContentBadMode pins an out-of-enumeration mode as a mapped
// ct-props-correct verdict rather than a silent interleave.
func TestProduceOpenContentBadMode(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="T"><xs:openContent mode="both"><xs:any/></xs:openContent><xs:sequence/></xs:complexType>`))
	if err == nil {
		t.Fatal(`Produce accepted mode="both"`)
	}
	if rule, ok := xsderr.RuleOf(err); !ok || rule != "ct-props-correct" {
		t.Fatalf("rule = %q (ok=%v), want ct-props-correct: %v", rule, ok, err)
	}
}

// TestProduceMisplacedOpenContent pins the positions the schema for schema
// documents does not allow an <openContent> in, each of which the producer
// SILENTLY DROPPED before #230. They are plain grammar faults, not rule verdicts:
// src-ct states no clause for them.
func TestProduceMisplacedOpenContent(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"beside <simpleContent>", `<xs:complexType name="T"><xs:openContent><xs:any/></xs:openContent><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`},
		{"beside <complexContent>", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:openContent><xs:any/></xs:openContent><xs:complexContent><xs:restriction base="tns:B"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
		{"directly under <complexContent>", `<xs:complexType name="B"><xs:sequence/></xs:complexType><xs:complexType name="T"><xs:complexContent><xs:openContent><xs:any/></xs:openContent><xs:restriction base="tns:B"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`},
		// No position inside a <simpleContent> subtree is legal: neither its
		// <restriction> nor its <extension> alternant admits an <openContent>.
		{"directly under <simpleContent>", `<xs:complexType name="T"><xs:simpleContent><xs:openContent><xs:any/></xs:openContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`},
		{"under <simpleContent><extension>", `<xs:complexType name="T"><xs:simpleContent><xs:extension base="xs:string"><xs:openContent><xs:any/></xs:openContent></xs:extension></xs:simpleContent></xs:complexType>`},
		{"under <simpleContent><restriction>", `<xs:complexType name="T"><xs:simpleContent><xs:restriction base="xs:string"><xs:openContent><xs:any/></xs:openContent></xs:restriction></xs:simpleContent></xs:complexType>`},
	}
	for _, tc := range cases {
		_, err := produce(t, wrap("urn:x", tc.body))
		if err == nil {
			t.Errorf("%s: Produce accepted a misplaced <openContent>", tc.name)
			continue
		}
		if _, ok := xsderr.RuleOf(err); ok {
			t.Errorf("%s: error = %v, want a plain grammar fault rather than a rule verdict", tc.name, err)
		}
		if !strings.Contains(err.Error(), "<openContent>") {
			t.Errorf("%s: error = %v, want it to name <openContent>", tc.name, err)
		}
	}
}

// TestProduceDefaultOpenContentGrammar pins the two <defaultOpenContent> shapes
// the schema for schema documents forbids — a missing <any> (its content model is
// (annotation?, any)) and mode="none" (its enumeration is interleave|suffix
// alone) — as plain grammar faults rather than a silently ignored default.
func TestProduceDefaultOpenContentGrammar(t *testing.T) {
	cases := []struct {
		name string
		def  string
		want string
	}{
		{"no <any> child", `<xs:defaultOpenContent/>`, "no <any> child"},
		{"mode=none", `<xs:defaultOpenContent mode="none"><xs:any/></xs:defaultOpenContent>`, `mode="none"`},
	}
	for _, tc := range cases {
		_, err := produce(t, wrap("urn:x", tc.def+`
			<xs:complexType name="T"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`))
		if err == nil {
			t.Errorf("%s: Produce accepted it", tc.name)
			continue
		}
		if _, ok := xsderr.RuleOf(err); ok {
			t.Errorf("%s: error = %v, want a plain grammar fault rather than a rule verdict", tc.name, err)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: error = %v, want it to mention %s", tc.name, err, tc.want)
		}
	}
}
