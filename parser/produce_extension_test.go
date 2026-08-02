package parser_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// These tests pin the two resolved-base halves of complex-type content the
// producer computes as of #228: §3.4.2.3.3 clause 4.2 (the extension {content
// type}, sub-cases 4.2.1/4.2.2/4.2.3.1/4.2.3.2/4.2.3.3) and §3.4.2.2 cases 3-5
// (the <simpleContent> {simple type definition}). They assert the STRUCTURE of
// the produced component, not merely that Produce returned nil.

// elementContentOf returns the ElementContent {content type} of a top-level
// complex type, failing when the variety is not element-only or mixed.
func elementContentOf(t *testing.T, s *xsd.Schema, name xsd.QName) xsd.ElementContent {
	t.Helper()
	td, ok := s.Type(name)
	if !ok {
		t.Fatalf("complex type %s not found", name)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type (%T)", name, td)
	}
	ec, ok := ct.ContentType().(xsd.ElementContent)
	if !ok {
		t.Fatalf("complex type %s {content type} is %T, want ElementContent", name, ct.ContentType())
	}
	return ec
}

// contentTypeOf returns the {content type} of a top-level complex type.
func contentTypeOf(t *testing.T, s *xsd.Schema, name xsd.QName) xsd.ContentType {
	t.Helper()
	td, ok := s.Type(name)
	if !ok {
		t.Fatalf("complex type %s not found", name)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type (%T)", name, td)
	}
	return ct.ContentType()
}

// xq is the expanded name of a top-level component of the urn:x test schemas.
func xq(local string) xsd.QName { return xsd.QName{Space: "urn:x", Local: local} }

// elementLocals returns the local names of the element declarations a model
// group's {particles} directly hold, in document order.
func elementLocals(t *testing.T, g xsd.ModelGroup) []string {
	t.Helper()
	var names []string
	for _, p := range g.Particles() {
		names = append(names, elementTermOf(t, p).Name().Local)
	}
	return names
}

// TestProduceExtensionSuffixSequence pins §3.4.2.3.3 clause 4.2.3.3
// (c-suffix-extension), the general case: the derived {particle} is a 1..1
// particle over a SEQUENCE whose {particles} are the ·base particle· followed by
// the ·effective content·, both spliced in unchanged (xr.ctd.n4-bis).
func TestProduceExtensionSuffixSequence(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">
			<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
		</xs:extension></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	d, ok := s.Type(xq("D"))
	if !ok {
		t.Fatal("complex type D not found")
	}
	dct := d.(xsd.ComplexType)
	if dct.DerivationMethod() != xsd.DerivationExtension {
		t.Fatalf("D {derivation method} = %s, want extension", dct.DerivationMethod())
	}
	if dct.BaseTypeDefinitionName() != xq("B") {
		t.Fatalf("D {base type definition} = %s, want %s", dct.BaseTypeDefinitionName(), xq("B"))
	}

	ec := elementContentOf(t, s, xq("D"))
	if ec.Mixed {
		t.Error("D {variety} is mixed, want element-only (·effective mixed· is false)")
	}
	if min := ec.Particle.Occurs().Min(); min != 1 {
		t.Errorf("derived particle {min occurs} = %d, want 1 (clause 4.2.3.3)", min)
	}
	if max, bounded := ec.Particle.Occurs().Max(); !bounded || max != 1 {
		t.Errorf("derived particle {max occurs} = (%d,%t), want 1 (clause 4.2.3.3)", max, bounded)
	}
	seq := groupTermOf(t, ec.Particle)
	if seq.Compositor() != xsd.CompositorSequence {
		t.Fatalf("derived {term} {compositor} = %s, want sequence (clause 4.2.3.3)", seq.Compositor())
	}
	parts := seq.Particles()
	if len(parts) != 2 {
		t.Fatalf("derived sequence has %d particles, want 2 (base particle then effective content)", len(parts))
	}

	// xr.ctd.n4-bis: the base particle is REUSED, not rebuilt from source — the
	// first member is the very particle the base's own {content type} holds.
	baseParticle := elementContentOf(t, s, xq("B")).Particle
	if !reflect.DeepEqual(parts[0], baseParticle) {
		t.Error("derived sequence[0] is not the base's own {particle}; clause 4.2.3.3 reuses it without copying")
	}
	if got := elementLocals(t, groupTermOf(t, parts[0])); !reflect.DeepEqual(got, []string{"a"}) {
		t.Errorf("base particle's elements = %v, want [a]", got)
	}
	if got := elementLocals(t, groupTermOf(t, parts[1])); !reflect.DeepEqual(got, []string{"b"}) {
		t.Errorf("effective content's elements = %v, want [b]", got)
	}
}

// TestProduceExtensionEmptyEffectiveContentReusesBase pins clause 4.2.2: a
// complex base with element-only or mixed content and an ***empty*** ·effective
// content· yields the base's ENTIRE Content Type record, not just its particle.
func TestProduceExtensionEmptyEffectiveContentReusesBase(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">
			<xs:attribute name="at" type="xs:string"/>
		</xs:extension></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	base := contentTypeOf(t, s, xq("B"))
	derived := contentTypeOf(t, s, xq("D"))
	if !reflect.DeepEqual(derived, base) {
		t.Fatalf("D {content type} = %#v, want the base's whole record %#v (clause 4.2.2)", derived, base)
	}
	// The extension's own attribute use still lands on the derived type.
	d, _ := s.Type(xq("D"))
	if !hasAttrUse(d.(xsd.ComplexType).AttributeUses(), "at") {
		t.Error("D lost the <extension>'s own attribute use")
	}
}

// TestProduceExtensionAllBaseWithEmptyExplicitContent pins clause 4.2.3.1: when
// the ·base particle·'s {term} is an all group and the ·explicit content· is
// empty, the derived {particle} IS the base particle. It is reachable only
// because ·effective content· and ·explicit content· differ here: mixed="true"
// with no model-group child makes the explicit content empty (clause 2.1.1)
// while clause 3.1.1 substitutes a non-empty effective content.
//
// The BASE is mixed too, and must be: cos-ct-extends clause 1.4.3.2.2.1 (#264)
// requires B and T to agree on mixed versus element-only, so an extension may
// not acquire mixed content its base lacks. The clause under test is unaffected
// — ·effective mixed· is true either way, which is what selects clause 3.1.1.
func TestProduceExtensionAllBaseWithEmptyExplicitContent(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B" mixed="true"><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>
		<xs:complexType name="D" mixed="true"><xs:complexContent><xs:extension base="tns:B"/></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ec := elementContentOf(t, s, xq("D"))
	if !ec.Mixed {
		t.Error("D {variety} is element-only, want mixed (·effective mixed· is true)")
	}
	baseParticle := elementContentOf(t, s, xq("B")).Particle
	if !reflect.DeepEqual(ec.Particle, baseParticle) {
		t.Fatal("D {particle} is not the base particle itself (clause 4.2.3.1)")
	}
	if g := groupTermOf(t, ec.Particle); g.Compositor() != xsd.CompositorAll {
		t.Fatalf("D {particle} {term} {compositor} = %s, want all", g.Compositor())
	}
}

// TestProduceExtensionAllPlusAllMerges pins clause 4.2.3.2: two all groups merge
// into ONE all group holding the base's {particles} followed by the effective
// content's, with {min occurs} from the effective content and {max occurs} 1.
//
// Both all groups carry minOccurs="0", and they must agree: cos-particle-extend
// (§3.9.6.2) clause 3.1 requires E.{min occurs} = B.{min occurs} for the
// all-group branch (#264), so no VALID schema can exhibit a merged particle
// whose {min occurs} differs from the base's. What the assertion still pins is
// that the merged particle takes an occurrence range at all rather than the
// clause 4.2.3.3 default of 1..1.
func TestProduceExtensionAllPlusAllMerges(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:all minOccurs="0"><xs:element name="a" type="xs:string"/></xs:all></xs:complexType>
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">
			<xs:all minOccurs="0"><xs:element name="b" type="xs:string"/></xs:all>
		</xs:extension></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	ec := elementContentOf(t, s, xq("D"))
	if min := ec.Particle.Occurs().Min(); min != 0 {
		t.Errorf("merged particle {min occurs} = %d, want 0 (the effective content's)", min)
	}
	if max, bounded := ec.Particle.Occurs().Max(); !bounded || max != 1 {
		t.Errorf("merged particle {max occurs} = (%d,%t), want 1 (clause 4.2.3.2)", max, bounded)
	}
	all := groupTermOf(t, ec.Particle)
	if all.Compositor() != xsd.CompositorAll {
		t.Fatalf("merged {compositor} = %s, want all", all.Compositor())
	}
	if got := elementLocals(t, all); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("merged {particles} = %v, want [a b] (base's followed by the effective content's)", got)
	}
}

// TestProduceExtensionNonParticleBases pins clause 4.2.1 (c-ctes): a simple base,
// or a complex base whose {content type}.{variety} is empty or simple,
// contributes NO particle — the result is clause 4.1.1/4.1.2's, exactly as for a
// restriction.
//
// Only the EMPTY-content base yields a schema a complete processor accepts, so
// only that row can assert on the produced component. Clause 4.2.1's
// fold-nothing behaviour is precisely what makes the other two invalid: with no
// base particle folded in, the derived type is element-only over a base whose
// {content type} is simple, and cos-ct-extends (§3.4.6.2, #264) has no branch
// for that pair — clause 1.4.3.2 for the complex simple-content base, clause 2.1
// for the simple-type base. Those two rows therefore pin the REJECTION, which is
// the only observable this mapping has on such a schema.
func TestProduceExtensionNonParticleBases(t *testing.T) {
	simpleContentBase := `<xs:complexType name="B"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>`
	emptyBase := `<xs:complexType name="B"/>`
	for _, tc := range []struct {
		name       string
		base       string
		ref        string
		wantReject bool
	}{
		{"empty-content complex base", emptyBase, "tns:B", false},
		{"simple-content complex base", simpleContentBase, "tns:B", true},
		{"simple type base", "", "xs:string", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, wrap("urn:x", tc.base+`
				<xs:complexType name="D"><xs:complexContent><xs:extension base="`+tc.ref+`">
					<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
				</xs:extension></xs:complexContent></xs:complexType>`))
			if tc.wantReject {
				assertRule(t, err, "cos-ct-extends")
				return
			}
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			ec := elementContentOf(t, s, xq("D"))
			// clause 4.1.2 via 4.2.1: the {particle} is the effective content
			// itself, with no base particle wrapped around it.
			if got := elementLocals(t, groupTermOf(t, ec.Particle)); !reflect.DeepEqual(got, []string{"b"}) {
				t.Fatalf("D content model = %v, want just [b] — clause 4.2.1 folds in no base particle", got)
			}
		})
	}
}

// TestProduceExtensionEmptyEverywhere pins clause 4.2.1 routing through clause
// 4.1.1: an empty base and an empty effective content give {variety} EMPTY, which
// admits no character content at all — never element-only (PRINCIPLES 13).
func TestProduceExtensionEmptyEverywhere(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"/>
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B"/></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if ct := contentTypeOf(t, s, xq("D")); ct.Variety() != xsd.ContentEmpty {
		t.Fatalf("D {content type}.{variety} = %s, want empty (clause 4.2.1 → 4.1.1)", ct.Variety())
	}
}

// TestProduceExtensionForwardBaseReference proves the base is built ON DEMAND: D
// precedes B in document order, so B's component does not exist when D is mapped.
func TestProduceExtensionForwardBaseReference(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">
			<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
		</xs:extension></xs:complexContent></xs:complexType>
		<xs:complexType name="B"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	seq := groupTermOf(t, elementContentOf(t, s, xq("D")).Particle)
	if len(seq.Particles()) != 2 {
		t.Fatalf("D content has %d particles, want 2 — the forward base= did not fold in", len(seq.Particles()))
	}
	// The on-demand build must not steal B's document-order slot in
	// {type definitions}: B is still registered, once, by its own top-level pass.
	if _, ok := s.Type(xq("B")); !ok {
		t.Fatal("complex type B is absent from {type definitions}; the on-demand build swallowed its registration")
	}
}

// TestProduceExtensionBaseCycle pins the produce-time termination condition for
// demand-driven base construction: a circular {base type definition} chain is
// rejected ct-props-correct clause 3 (§3.4.6.1) — the SAME rule
// xsd/resolve.go's checkComplexBaseAcyclic charges on the programmatic path.
func TestProduceExtensionBaseCycle(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="A"><xs:complexContent><xs:extension base="tns:B"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>
		<xs:complexType name="B"><xs:complexContent><xs:extension base="tns:A"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`))
	assertRule(t, err, "ct-props-correct")
}

// TestProduceExtensionSelfCycle pins the one-node case of the same guard.
func TestProduceExtensionSelfCycle(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="A"><xs:complexContent><xs:extension base="tns:A"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`))
	assertRule(t, err, "ct-props-correct")
}

// TestProduceExtensionDanglingBase pins that a base= naming no type at all is
// charged src-resolve clause 1.1 — the same rule and clause finalize charges for
// an unresolvable {base type definition}, and the same one the simple-type base
// path already charges.
func TestProduceExtensionDanglingBase(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:missing"><xs:sequence/></xs:extension></xs:complexContent></xs:complexType>`))
	assertRule(t, err, "src-resolve")
	if !strings.Contains(err.Error(), "clause 1.1") {
		t.Fatalf("error = %v, want it to cite src-resolve clause 1.1", err)
	}
}

// TestProduceExtensionOfAnyType proves xs:anyType is reachable as a base: it is
// seeded DONE in the build memo, so <extension base="xs:anyType"> resolves to the
// very ur-type component the builder registered, and clause 4.2.3.3 wraps its
// mixed ##any content with the derivation's own.
//
// D is mixed="true" because it must be: xs:anyType's own {content type}.{variety}
// is mixed (§3.4.7), and cos-ct-extends (#264) grants the ur-type NO exemption
// from clause 1.4.3.2.2.1 — unlike derivation-ok-restriction clause 2.1, which
// bypasses the variety match for an xs:anyType base outright. An extension that
// adds a genuine local particle must therefore keep the ur-type's mixedness.
func TestProduceExtensionOfAnyType(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="D" mixed="true"><xs:complexContent><xs:extension base="xs:anyType">
			<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
		</xs:extension></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	seq := groupTermOf(t, elementContentOf(t, s, xq("D")).Particle)
	if len(seq.Particles()) != 2 {
		t.Fatalf("D content has %d particles, want 2 (xs:anyType's particle plus the extension's)", len(seq.Particles()))
	}
}

// TestProduceExtensionSiblingElementsFolded is #51's ask, from the producing end:
// clause 4.2's merge must put the BASE's element declarations inside the derived
// type's OWN {content type}, which is what lets xsd's ##definedSibling walk
// (wildcardadmit.go) see them without ever following {base type definition}. If
// this fails, the bug is in the merge, not in the walk.
func TestProduceExtensionSiblingElementsFolded(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B"><xs:sequence><xs:element name="fromBase" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">
			<xs:sequence><xs:element name="fromDerived" type="xs:string"/><xs:any notQName="##definedSibling"/></xs:sequence>
		</xs:extension></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	var found []string
	var walk func(g xsd.ModelGroup)
	walk = func(g xsd.ModelGroup) {
		for _, p := range g.Particles() {
			rt, ok := p.Term().(xsd.ResolvedTerm)
			if !ok {
				continue
			}
			switch term := rt.Term.(type) {
			case xsd.ElementDeclaration:
				found = append(found, term.Name().Local)
			case xsd.ModelGroup:
				walk(term)
			}
		}
	}
	walk(groupTermOf(t, elementContentOf(t, s, xq("D")).Particle))
	if !reflect.DeepEqual(found, []string{"fromBase", "fromDerived"}) {
		t.Fatalf("D's own content model declares %v, want [fromBase fromDerived] — the base's element must be ·contained· in the derived {content type}", found)
	}
}

// TestProduceSimpleContentExtensionCases pins the §3.4.2.2 {simple type
// definition} tableau for <extension>: case 4 (simple base) and case 3 (complex
// base whose own {content type} has {variety} simple). Both REUSE an existing
// component — the assertions compare POINTERS, since simple-type component
// identity is load-bearing.
//
// Case 5 (c-ctsc-bad, "otherwise ·xs:anySimpleType·") is pinned separately by
// TestProduceSimpleContentExtensionCase5Rejected: it is the arm for a
// <simpleContent> <extension> over a base whose {content type} is NOT simple,
// which cos-ct-extends clause 1.4.1 (#264) rejects, so the recovery value it
// produces is not observable on a schema that finalizes.
func TestProduceSimpleContentExtensionCases(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="Case4"><xs:simpleContent><xs:extension base="xs:string"/></xs:simpleContent></xs:complexType>
		<xs:complexType name="Case3"><xs:simpleContent><xs:extension base="tns:Case4"/></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	xsString, ok := s.Type(xsd.QName{Space: xsdNS, Local: "string"})
	if !ok {
		t.Fatal("xs:string not seeded")
	}
	for _, tc := range []struct {
		name string
		want xsd.TypeDefinition
	}{
		{"Case4", xsString}, // case 4: the resolved simple base itself
		{"Case3", xsString}, // case 3: the base complex type's own {simple type definition}
	} {
		t.Run(tc.name, func(t *testing.T) {
			ct := contentTypeOf(t, s, xq(tc.name))
			sc, ok := ct.(xsd.SimpleContent)
			if !ok {
				t.Fatalf("%s {content type} is %T, want SimpleContent (§3.4.2.2 {variety} simple)", tc.name, ct)
			}
			if sc.SimpleType != tc.want.(*xsd.SimpleType) {
				t.Fatalf("%s {simple type definition} = %s, want the very %s component (reused, not rebuilt)",
					tc.name, sc.SimpleType.Name(), tc.want.Name())
			}
		})
	}
	// {derivation method} and {base type definition} come from the alternant.
	d, _ := s.Type(xq("Case4"))
	if m := d.(xsd.ComplexType).DerivationMethod(); m != xsd.DerivationExtension {
		t.Fatalf("Case4 {derivation method} = %s, want extension", m)
	}
}

// TestProduceSimpleContentExtensionCase5Rejected pins the one thing observable
// about §3.4.2.2 case 5 (c-ctsc-bad): the shape that reaches it — a
// <simpleContent> <extension> whose base is a complex type with element-only
// content — is rejected by cos-ct-extends clause 1.4.1, which requires B and T
// to have the SAME {content type}.{simple type definition} and so both to be
// simple. Case 5 is a §5.3-style recovery value for a schema no complete
// processor accepts, not a mapping a valid schema can exercise.
func TestProduceSimpleContentExtensionCase5Rejected(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="Elems"><xs:sequence><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>
		<xs:complexType name="Case5"><xs:simpleContent><xs:extension base="tns:Elems"/></xs:simpleContent></xs:complexType>`))
	assertRule(t, err, "cos-ct-extends")
}

// TestProduceSimpleContentExtensionAttributes proves the <extension>'s own
// attribute uses and assertions reach the derived type (§3.4.2.1 clause 2,
// §3.4.2.4): a <simpleContent><extension> exists precisely to add them.
func TestProduceSimpleContentExtensionAttributes(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="D"><xs:simpleContent><xs:extension base="xs:string">
			<xs:attribute name="unit" type="xs:string"/>
			<xs:assert test="true()"/>
		</xs:extension></xs:simpleContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	d, _ := s.Type(xq("D"))
	ct := d.(xsd.ComplexType)
	if !hasAttrUse(ct.AttributeUses(), "unit") {
		t.Error("D is missing the <extension>'s own attribute use 'unit'")
	}
	if len(ct.Assertions()) != 1 {
		t.Errorf("D has %d {assertions}, want the <extension>'s one", len(ct.Assertions()))
	}
}

// TestProduceSimpleContentMixedRejected pins src-ct clause 1
// (simple-content-rules, §3.4.3): with the <simpleContent> alternative chosen,
// the <complexType> must not have mixed="true". It is charged BEFORE the
// <restriction> limitation decline, so the verdict is the rule either way.
func TestProduceSimpleContentMixedRejected(t *testing.T) {
	for _, tc := range []struct{ name, alternant string }{
		{"extension", `<xs:extension base="xs:string"/>`},
		{"restriction", `<xs:restriction base="xs:string"><xs:maxLength value="4"/></xs:restriction>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:x", `
				<xs:complexType name="D" mixed="true"><xs:simpleContent>`+tc.alternant+`</xs:simpleContent></xs:complexType>`))
			assertRule(t, err, "src-ct")
			if !strings.Contains(err.Error(), "clause 1") {
				t.Fatalf("error = %v, want it to cite src-ct clause 1", err)
			}
		})
	}
}

// TestProduceComplexContentRestrictionOfSimpleTypeRejected pins ct-props-correct
// clause 2 as REACHABLE from the producer: a complex type whose {base type
// definition} is a simple type must have {derivation method} extension, so the
// restriction form is rejected at finalize (xsd/complexderivation.go's
// checkSimpleBaseIsExtension). The mapping code deliberately holds no second copy
// of that rule.
func TestProduceComplexContentRestrictionOfSimpleTypeRejected(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `
		<xs:complexType name="D"><xs:complexContent><xs:restriction base="xs:string">
			<xs:sequence/>
		</xs:restriction></xs:complexContent></xs:complexType>`))
	assertRule(t, err, "ct-props-correct")
	if !strings.Contains(err.Error(), "clause 2") {
		t.Fatalf("error = %v, want it to cite ct-props-correct clause 2", err)
	}
}

// TestProduceComplexContentWithoutDerivation pins that a <complexContent> with
// neither alternant is a plain grammar fault, not a fabricated rule verdict:
// §3.4.2.3 requires one of them through the schema for schema documents, which
// src-ct incorporates by reference without a clause of its own.
func TestProduceComplexContentWithoutDerivation(t *testing.T) {
	_, err := produce(t, wrap("urn:x", `<xs:complexType name="D"><xs:complexContent/></xs:complexType>`))
	if err == nil {
		t.Fatal("Produce accepted a <complexContent> with no <restriction>/<extension>")
	}
	if _, ok := xsderr.RuleOf(err); ok {
		t.Fatalf("error = %v, want a plain grammar fault rather than a rule verdict", err)
	}
}

// TestProduceExtensionOpenContentUnionsWithBase pins §3.4.2.3.3 clause 6.2's
// {wildcard} on the extension path, where clause 4.2.3 has already handed the
// base's {open content} through into the ·explicit content type·: the derived
// type's own <openContent> does NOT replace it but widens it, the {namespace
// constraint} being the §3.10.6.3 wildcard union of the two, with {process
// contents} and {annotations} taken from the derivation's own W alone.
func TestProduceExtensionOpenContentUnionsWithBase(t *testing.T) {
	s, err := produce(t, wrap("urn:x", `
		<xs:complexType name="B">
			<xs:openContent mode="suffix"><xs:any namespace="urn:a" processContents="skip"/></xs:openContent>
			<xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence>
		</xs:complexType>
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">
			<xs:openContent mode="interleave"><xs:any namespace="urn:b" processContents="lax"/></xs:openContent>
			<xs:sequence><xs:element name="y" type="xs:string"/></xs:sequence>
		</xs:extension></xs:complexContent></xs:complexType>`))
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	oc := elementContentOf(t, s, xq("D")).OpenContent
	if oc == nil {
		t.Fatal("D {open content} is absent, want the clause 6.2 record")
	}
	if oc.Mode() != xsd.OpenContentInterleave {
		t.Errorf("{mode} = %s, want interleave — the derivation's own mode, not the base's", oc.Mode())
	}
	if got := oc.Wildcard().ProcessContents(); got != xsd.ProcessLax {
		t.Errorf("{wildcard}.{process contents} = %v, want lax — W's own, not the base's skip", got)
	}
	nc := oc.Wildcard().NamespaceConstraint()
	if nc.Variety() != xsd.NamespaceConstraintEnumeration {
		t.Fatalf("{namespace constraint}.{variety} = %v, want enumeration", nc.Variety())
	}
	want := []xsd.Namespace{xsd.NamespaceName("urn:b"), xsd.NamespaceName("urn:a")}
	if !reflect.DeepEqual(nc.Namespaces(), want) {
		t.Errorf("{namespaces} = %v, want %v (cos-aw-union of W and the base's)", nc.Namespaces(), want)
	}
}

// TestParseExtensionBaseBuiltUnderItsOwnProducer is the regression guard for the
// cross-document half of on-demand base construction: a complex type reached
// through a base= in ANOTHER document must be mapped by the producer of the
// document that DECLARES it, never by the referring one.
//
// The two documents disagree about elementFormDefault, and lib.xsd is a
// chameleon (§4.2.3 clause 2.3, §F.1) — so a local element declaration inside
// the base takes {urn:a}x under lib's own producer and {}x under main's. main is
// produced FIRST (the root heads the assembly), so B does not exist when D's
// <extension base="tns:B"> is mapped and the on-demand path is the one under
// test. If the base were built under the referring producer, x would land in the
// absent namespace and this test would fail.
func TestParseExtensionBaseBuiltUnderItsOwnProducer(t *testing.T) {
	const xs = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`
	s, err := parseMap(t, "main.xsd", map[string]string{
		"main.xsd": `<xs:schema ` + xs + ` targetNamespace="urn:a" xmlns:tns="urn:a">` +
			`<xs:include schemaLocation="lib.xsd"/>` +
			`<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">` +
			`<xs:sequence><xs:element name="y" type="xs:string"/></xs:sequence>` +
			`</xs:extension></xs:complexContent></xs:complexType>` +
			`</xs:schema>`,
		// No targetNamespace of its own: chameleon-coerced into urn:a, but keeping
		// ITS OWN elementFormDefault.
		"lib.xsd": `<xs:schema ` + xs + ` elementFormDefault="qualified">` +
			`<xs:complexType name="B"><xs:sequence><xs:element name="x" type="xs:string"/></xs:sequence></xs:complexType>` +
			`</xs:schema>`,
	})
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	// The base's own component: its local element is qualified by lib's
	// elementFormDefault, in the chameleon-coerced namespace.
	baseElem := elementTermOf(t, groupTermOf(t, elementContentOf(t, s, xsd.QName{Space: "urn:a", Local: "B"}).Particle).Particles()[0])
	if got := baseElem.Name(); got != (xsd.QName{Space: "urn:a", Local: "x"}) {
		t.Fatalf("base's local element = %s, want {urn:a}x — the base was built under the wrong document's producer", got)
	}
	// The derived type folds in that very declaration (clause 4.2.3.3) and adds
	// its own, which main's own (unqualified) form default leaves namespace-less.
	seq := groupTermOf(t, elementContentOf(t, s, xsd.QName{Space: "urn:a", Local: "D"}).Particle)
	if len(seq.Particles()) != 2 {
		t.Fatalf("D content has %d particles, want 2", len(seq.Particles()))
	}
	folded := elementTermOf(t, groupTermOf(t, seq.Particles()[0]).Particles()[0])
	if got := folded.Name(); got != (xsd.QName{Space: "urn:a", Local: "x"}) {
		t.Fatalf("folded base element = %s, want {urn:a}x", got)
	}
	own := elementTermOf(t, groupTermOf(t, seq.Particles()[1]).Particles()[0])
	if got := own.Name(); got != (xsd.QName{Local: "y"}) {
		t.Fatalf("D's own local element = %s, want {}y (main declares no elementFormDefault)", got)
	}
}

// TestProduceExtensionBaseMappedExactlyOnce guards the memo: a base built ON
// DEMAND to serve a derivation must not be built AGAIN when its own top-level
// position is reached. A second mapping would register its local element's
// identity constraint twice and collide under sch-props-correct clause 2
// (§3.17.6.1) — a duplicate the source never declared.
func TestProduceExtensionBaseMappedExactlyOnce(t *testing.T) {
	if _, err := produce(t, wrap("urn:x", `
		<xs:complexType name="D"><xs:complexContent><xs:extension base="tns:B">
			<xs:sequence><xs:element name="b" type="xs:string"/></xs:sequence>
		</xs:extension></xs:complexContent></xs:complexType>
		<xs:complexType name="B"><xs:sequence>
			<xs:element name="a" type="xs:string">
				<xs:key name="k"><xs:selector xpath="."/><xs:field xpath="@id"/></xs:key>
			</xs:element>
		</xs:sequence></xs:complexType>`)); err != nil {
		t.Fatalf("Produce: %v — the base was mapped twice, duplicating its identity constraint", err)
	}
}

// TestProhibitedAttributeNotInheritedPastRestriction is the end-to-end §3.4.2.4
// clause 3.2.2 repro, driven through the real producer rather than a hand-built
// component set: A declares @x, B restricts A with <attribute ref="x"
// use="prohibited"/>, and E extends B declaring its own @x.
//
// The schema is VALID. B.{attribute uses} is empty — clause 3.2.2 blocks the
// inherited x, and the Note confirms the prohibited <attribute> corresponds to no
// component of B either — so clause 3.1 gives E B's empty set and E's own @x is
// the single member named x. Leaving clause 3.2.2 unapplied leaves B carrying A's
// x, E then holds two members named x, and ct-props-correct clause 4 rejects a
// duplicate the source never wrote (#401).
//
// The prohibition-free row is the control: with B inheriting @x silently, E's
// re-declaration IS the clause 4 collision and must still be charged, so the
// first row cannot be bought by weakening the uniqueness check.
func TestProhibitedAttributeNotInheritedPastRestriction(t *testing.T) {
	for _, tc := range []struct {
		name       string
		bProhibits string
		wantRule   xsderr.Rule
	}{
		{"B prohibits @x, so E may declare its own", `<xs:attribute ref="x" use="prohibited"/>`, ""},
		{"B inherits @x silently, so E may not re-declare it", ``, xsderr.Rule("ct-props-correct")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := `<xs:attribute name="x" type="xs:string"/>` +
				`<xs:complexType name="A"><xs:sequence/><xs:attribute ref="x"/></xs:complexType>` +
				`<xs:complexType name="B"><xs:complexContent><xs:restriction base="A">` +
				`<xs:sequence/>` + tc.bProhibits +
				`</xs:restriction></xs:complexContent></xs:complexType>` +
				`<xs:complexType name="E"><xs:complexContent><xs:extension base="B">` +
				`<xs:sequence/><xs:attribute ref="x"/>` +
				`</xs:extension></xs:complexContent></xs:complexType>`
			s, err := produce(t, wrap("", body))
			if tc.wantRule != "" {
				assertRule(t, err, tc.wantRule)
				return
			}
			if err != nil {
				t.Fatalf("an extension re-declaring a name its restriction ancestor prohibited was rejected: %v", err)
			}
			if got := attributeUseLocals(t, s, xsd.QName{Local: "B"}); len(got) != 0 {
				t.Errorf("B.{attribute uses} = %v, want empty (clause 3.2.2 blocks the inherited x)", got)
			}
			if got := attributeUseLocals(t, s, xsd.QName{Local: "E"}); !reflect.DeepEqual(got, []string{"x"}) {
				t.Errorf("E.{attribute uses} = %v, want exactly [x]", got)
			}
		})
	}
}

// attributeUseLocals is the local part of each {attribute uses} member's expanded
// name, in the property's own order.
func attributeUseLocals(t *testing.T, s *xsd.Schema, name xsd.QName) []string {
	t.Helper()
	td, ok := s.Type(name)
	if !ok {
		t.Fatalf("complex type %s not found", name)
	}
	ct, ok := td.(xsd.ComplexType)
	if !ok {
		t.Fatalf("type %s is not a complex type (%T)", name, td)
	}
	var names []string
	for _, u := range ct.AttributeUses() {
		switch d := u.AttributeDeclaration().(type) {
		case xsd.LocalAttributeDeclaration:
			names = append(names, d.Declaration.Name().Local)
		case xsd.AttributeDeclarationRef:
			names = append(names, d.Name.Local)
		default:
			t.Fatalf("unexpected {attribute declaration} shape %T", d)
		}
	}
	return names
}
