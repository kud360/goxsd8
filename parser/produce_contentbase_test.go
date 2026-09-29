package parser_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin that ct-props-correct clause 3 (§3.4.6.1) is charged over
// {base type definition} only (#1842). A named complex type whose content model
// holds a local element with an anonymous type deriving from that named type is
// containment, not a circular chain: the anonymous type's chain runs through the
// named type to xs:anyType.

// contentBaseTypes are the two forms of the named type T that carry a content
// model, each holding a local element whose anonymous type restricts T and adds
// an assertion of its own. T carries one assertion, which §3.4.2.1 clause 1
// folds into the anonymous type ahead of its own.
var contentBaseTypes = []struct {
	name, t string
}{
	{"implicit content (particlesEb041)", `
		<xs:complexType name="T">
			<xs:sequence>
				<xs:element name="a"/>
				<xs:element name="inner" minOccurs="0">
					<xs:complexType><xs:complexContent><xs:restriction base="tns:T">
						<xs:sequence><xs:element name="a"/></xs:sequence>
						<xs:assert test="not(false())"/>
					</xs:restriction></xs:complexContent></xs:complexType>
				</xs:element>
			</xs:sequence>
			<xs:assert test="true()"/>
		</xs:complexType>`},
	{"complexContent restriction of xs:anyType", `
		<xs:complexType name="T"><xs:complexContent><xs:restriction base="xs:anyType">
			<xs:sequence>
				<xs:element name="a"/>
				<xs:element name="inner" minOccurs="0">
					<xs:complexType><xs:complexContent><xs:restriction base="tns:T">
						<xs:sequence><xs:element name="a"/></xs:sequence>
						<xs:assert test="not(false())"/>
					</xs:restriction></xs:complexContent></xs:complexType>
				</xs:element>
			</xs:sequence>
			<xs:assert test="true()"/>
		</xs:restriction></xs:complexContent></xs:complexType>`},
}

// TestProduceContentModelBaseIsNotACycle pins both forms. Each row fails with
// buildComplexType's guard covering T's whole production instead of its base
// hop alone: T is then still on the guard when inner's anonymous type resolves
// base="tns:T", and the reference is charged ct-props-correct clause 3.
func TestProduceContentModelBaseIsNotACycle(t *testing.T) {
	for _, tc := range contentBaseTypes {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, wrap("urn:x", tc.t))
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			seq := groupTermOf(t, elementContentOf(t, s, xq("T")).Particle)
			inner := elementTermOf(t, seq.Particles()[1])
			ct := inlineComplexType(t, inner.TypeDefinition())
			if ref, ok := ct.Base().(xsd.TypeDefinitionRef); !ok || ref.Name != xq("T") {
				t.Fatalf("inner's type has {base type definition} %#v, want a reference to %s", ct.Base(), xq("T"))
			}
			var tests []string
			for _, a := range ct.Assertions() {
				tests = append(tests, a.Test().Expression())
			}
			if want := []string{"true()", "not(false())"}; !slices.Equal(tests, want) {
				t.Fatalf("inner's type has {assertions} %q, want %q: T's own, then the restriction's (§3.4.2.1 clauses 1, 2)", tests, want)
			}
		})
	}
}

// TestProduceContentModelBaseStillJudged pins that the containment reference is
// still judged as a derivation: particlesEb040's shape, where inner is required,
// so the anonymous restriction's content drops a particle T requires and
// derivation-ok-restriction (§3.4.6.3) rejects it — not ct-props-correct.
func TestProduceContentModelBaseStillJudged(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="T">
			<xs:sequence>
				<xs:element name="a"/>
				<xs:element name="inner">
					<xs:complexType><xs:complexContent><xs:restriction base="tns:T">
						<xs:sequence><xs:element name="a"/></xs:sequence>
					</xs:restriction></xs:complexContent></xs:complexType>
				</xs:element>
			</xs:sequence>
		</xs:complexType>`))
	assertRule(t, err, "derivation-ok-restriction")
}

// TestProduceRestrictionBaseCycle pins that a genuine circular chain through
// <restriction> alone is still charged ct-props-correct clause 3 at the type the
// chain comes back to — TestProduceExtensionBaseCycle's twin on the other
// alternant, whose base hop is now the only one the guard covers.
func TestProduceRestrictionBaseCycle(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="A"><xs:complexContent><xs:restriction base="tns:B"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>
		<xs:complexType name="B"><xs:complexContent><xs:restriction base="tns:A"><xs:sequence/></xs:restriction></xs:complexContent></xs:complexType>`))
	assertRule(t, err, "ct-props-correct")
	if !strings.Contains(err.Error(), "circular complex type definition: {urn:x}A") {
		t.Fatalf("error = %v, want the cycle charged at A, the type the chain comes back to", err)
	}
}

// TestProduceContentModelExtensionRefused pins the GAP(parser) marker on
// baseComponent: an anonymous <extension> of T inside T's own content
// model reads T's unfinished {content type}, and is refused as a producer limit
// charged to no rule, never as ct-props-correct clause 3.
func TestProduceContentModelExtensionRefused(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="T">
			<xs:sequence>
				<xs:element name="inner" minOccurs="0">
					<xs:complexType><xs:complexContent><xs:extension base="tns:T"/></xs:complexContent></xs:complexType>
				</xs:element>
			</xs:sequence>
		</xs:complexType>`))
	if err == nil {
		t.Fatal("Produce accepted an extension of T inside T's own content model, which this producer cannot map")
	}
	if rule, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("error charged %s, want a producer limit charged to no rule (%v)", rule, err)
	}
	if !strings.HasPrefix(err.Error(), "parser: the <extension> at ") || !strings.Contains(err.Error(), "derives from {urn:x}T from inside {urn:x}T's own content model") {
		t.Fatalf("error = %v, want the refusal naming the <extension> and T", err)
	}
}

// TestParseRedefineOriginalLeavesNamedMemoAlone pins that only a NAMED identity
// moves a declaration to complexContentPending (enterContentModel). root.xsd
// reaches r.xsd, which redefines lib.xsd's ct, BEFORE it reaches lib.xsd
// plainly, so src-expredef clause 1.1's anonymous original is produced from
// lib.xsd's declaration while that declaration's own named build has not
// started. Were the original to mark it pending, run's later named build of it
// would find that entry and fail as a producer fault instead of reaching the
// sch-props-correct clause 2 collision the two claims on {urn:t}ct make.
func TestParseRedefineOriginalLeavesNamedMemoAlone(t *testing.T) {
	_, err := parseMap(t, "root.xsd", map[string]string{
		"root.xsd": wrap("urn:t", `<xs:include schemaLocation="r.xsd"/><xs:include schemaLocation="lib.xsd"/>`),
		"lib.xsd": wrap("urn:t", `<xs:complexType name="ct"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`),
		"r.xsd": wrap("urn:t", `<xs:redefine schemaLocation="lib.xsd">`+
			`<xs:complexType name="ct"><xs:complexContent><xs:restriction base="tns:ct"><xs:sequence>`+
			`<xs:element name="a" type="xs:string"/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType>`+
			`</xs:redefine>`),
	})
	mustRule(t, err, "sch-props-correct")
}
