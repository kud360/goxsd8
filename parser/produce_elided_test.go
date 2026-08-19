package parser_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// elidedTargets are the top-level definitions every row in this file's tables
// wraps its body beside: a named <group> so every tns:G reference resolves, and a
// complex type so the <complexContent>/<extension> position has a base.
const elidedTargets = `<xs:group name="G"><xs:sequence><xs:element name="g1" type="xs:string"/></xs:sequence></xs:group>` +
	`<xs:complexType name="B"><xs:sequence><xs:element name="b1" type="xs:string"/></xs:sequence></xs:complexType>`

// elidedFaults are the schema-for-schema-documents grammar faults §5.1's first
// bullet binds whatever the enclosing subtree maps to, each paired with the
// fragment of the diagnostic that names IT — so a row cannot pass on some other
// rejection the document happens to earn (#883).
//
// Every fragment is charged by a site that already existed: rejectProhibitedRefAttrs
// (produce.go), produceGroupRefParticle's missing-ref fault, groupParticles' default
// arm, and allOccursGrammar.
var elidedFaults = []struct {
	name string
	body string
	want string
}{
	{
		name: `<group ref=> carrying the prohibited name`,
		body: `<xs:group ref="tns:G" name="X"/>`,
		want: `carries a name attribute`,
	},
	{
		name: `<group> with no ref`,
		body: `<xs:group/>`,
		want: `must be a reference`,
	},
	{
		name: `unexpected model group child`,
		body: `<xs:attribute name="z" type="xs:string"/>`,
		want: `unexpected model group child`,
	},
	{
		name: `<all> occurrence outside the {0,1} enumeration`,
		body: `<xs:all minOccurs="2"><xs:element name="a1" type="xs:string"/></xs:all>`,
		want: `outside the enumeration 0, 1`,
	},
}

// elidedControl wraps a fault in a LIVE <sequence>, the position nothing elides.
// Its diagnostic is the one every elided position must reproduce.
func elidedControl(fault string) string {
	return `<xs:complexType name="T"><xs:sequence>` + fault + `</xs:sequence></xs:complexType>`
}

// elidedPositions wrap one of elidedFaults' bodies in a subtree an elision test
// reaches, spanning the depths and container forms the fault must be rejected at.
var elidedPositions = []struct {
	name string
	wrap func(fault string) string
}{
	{
		name: `<sequence maxOccurs="0"> at the top model-group position (clause 2.1.4)`,
		wrap: func(f string) string {
			return `<xs:complexType name="T"><xs:sequence maxOccurs="0">` + f + `</xs:sequence></xs:complexType>`
		},
	},
	{
		name: `<sequence minOccurs="0" maxOccurs="0"> at the top model-group position`,
		wrap: func(f string) string {
			return `<xs:complexType name="T"><xs:sequence minOccurs="0" maxOccurs="0">` + f + `</xs:sequence></xs:complexType>`
		},
	},
	{
		name: `nested two levels, the inner one mapping to no component`,
		wrap: func(f string) string {
			return `<xs:complexType name="T"><xs:sequence><xs:sequence minOccurs="0" maxOccurs="0">` + f +
				`</xs:sequence></xs:sequence></xs:complexType>`
		},
	},
	{
		name: `under <complexContent>/<extension>`,
		wrap: func(f string) string {
			return `<xs:complexType name="T"><xs:complexContent><xs:extension base="tns:B">` +
				`<xs:sequence maxOccurs="0">` + f + `</xs:sequence></xs:extension></xs:complexContent></xs:complexType>`
		},
	},
	{
		name: `inside a named <group> body`,
		wrap: func(f string) string {
			return `<xs:group name="G3"><xs:sequence><xs:sequence minOccurs="0" maxOccurs="0">` + f +
				`</xs:sequence></xs:sequence></xs:group>`
		},
	},
}

// elidedLoc matches the source position a diagnostic carries, so two rejections
// of the same fault at different offsets compare equal.
var elidedLoc = regexp.MustCompile(regexp.QuoteMeta(produceURI) + `:\d+:\d+`)

// elidedReject produces doc, requiring a rejection, and returns its diagnostic
// with every source position replaced by a placeholder.
func elidedReject(t *testing.T, doc string) string {
	t.Helper()
	_, err := produce(t, wrap("urn:po", elidedTargets+doc))
	if err == nil {
		t.Fatalf("Produce accepted a document the schema for schema documents rejects")
	}
	return elidedLoc.ReplaceAllString(err.Error(), "<loc>")
}

// TestProduceElidedSubtreeGrammarCharged pins that §3.4.2.3.3 clause 2.1.4's
// maxOccurs="0" elision, and the minOccurs=maxOccurs=0 "maps to no component"
// gate of §3.7.2/§3.8.2/§3.3.2.3, decide what a subtree maps TO and nothing else:
// §5.1's first bullet binds the schema DOCUMENT to the Schema for Schema
// Documents at every depth, so each fault in elidedFaults is rejected in each
// position of elidedPositions with the message its live control earns.
//
// The control's diagnostic is what the elided rows are compared against, rather
// than a literal transcribed here: a walk that rejected the elided subtree with
// SOME other diagnostic — the p-props-correct verdict the enclosing particle
// earns, say — passes a table that only asserts rejection.
func TestProduceElidedSubtreeGrammarCharged(t *testing.T) {
	for _, fault := range elidedFaults {
		t.Run(fault.name, func(t *testing.T) {
			want := elidedReject(t, elidedControl(fault.body))
			if !strings.Contains(want, fault.want) {
				t.Fatalf("control diagnostic = %q, want it to name the fault (%q)", want, fault.want)
			}
			for _, pos := range elidedPositions {
				t.Run(pos.name, func(t *testing.T) {
					if got := elidedReject(t, pos.wrap(fault.body)); got != want {
						t.Fatalf("error = %q, want the control's own diagnostic %q", got, want)
					}
				})
			}
		})
	}
}

// TestProduceElidedSubtreeMapsToNothing pins the other half: a subtree that is
// legal contributes NO component whichever elision reaches it. ·explicit content·
// stays ***empty*** (clause 2.1.4) and the {content type} the complex type ends up
// with is the one it would have with no model-group child at all, so the walk
// added for the grammar faults above changes what the elided subtree maps to in
// no way.
func TestProduceElidedSubtreeMapsToNothing(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: `no model-group child at all (the baseline {content type})`,
			body: `<xs:complexType name="T"/>`,
		},
		{
			name: `<sequence minOccurs="0" maxOccurs="0"> holding a live element`,
			body: `<xs:complexType name="T"><xs:sequence minOccurs="0" maxOccurs="0">` +
				`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
		},
		{
			name: `<all minOccurs="0" maxOccurs="0"> holding a live element`,
			body: `<xs:complexType name="T"><xs:all minOccurs="0" maxOccurs="0">` +
				`<xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`,
		},
		{
			name: `<group ref= minOccurs="0" maxOccurs="0"> at the top model-group position`,
			body: `<xs:complexType name="T"><xs:group ref="tns:G" minOccurs="0" maxOccurs="0"/></xs:complexType>`,
		},
		{
			name: `<sequence minOccurs="0" maxOccurs="0"> holding a resolvable <group ref>`,
			body: `<xs:complexType name="T"><xs:sequence minOccurs="0" maxOccurs="0">` +
				`<xs:group ref="tns:G"/></xs:sequence></xs:complexType>`,
		},
		{
			name: `<sequence minOccurs="0" maxOccurs="0"> holding an <any> and an elided <element>`,
			body: `<xs:complexType name="T"><xs:sequence minOccurs="0" maxOccurs="0">` +
				`<xs:any processContents="lax"/><xs:element name="a" type="xs:string" minOccurs="0" maxOccurs="0"/>` +
				`</xs:sequence></xs:complexType>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, wrap("urn:po", elidedTargets+tc.body))
			if err != nil {
				t.Fatalf("Produce rejected a legal elided subtree: %v", err)
			}
			ct := contentTypeOf(t, s, xsd.QName{Space: "urn:po", Local: "T"})
			if _, ok := ct.(xsd.EmptyContent); !ok {
				t.Fatalf("{content type} = %T, want xsd.EmptyContent — the elided subtree contributed a component", ct)
			}
		})
	}
}

// TestProduceMaxOccursZeroParticleCharged pins the correction the two elision
// predicates keep apart. Clause 2.1.4 tests maxOccurs ALONE, so a model-group
// child carrying only maxOccurs="0" empties the enclosing ·explicit content·;
// §3.7.2/§3.8.2's "corresponds to no component at all" gate is minOccurs=maxOccurs=0
// and does NOT hold there, so the child still maps to a real Particle whose
// {min occurs} of 1 exceeds its {max occurs} of 0 — p-props-correct clause 2.1,
// the same verdict the identical child already earned one level down.
//
// Reconciling the two predicates in either direction breaks one of these rows:
// widening clause 2.1.4 to both-zero fails the ACCEPT sibling above, narrowing the
// component gate to maxOccurs alone fails every row here.
func TestProduceMaxOccursZeroParticleCharged(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2).
	for _, tc := range []struct {
		name string
		body string
	}{
		{
			name: `<sequence maxOccurs="0"> at the top model-group position`,
			body: `<xs:complexType name="T"><xs:sequence maxOccurs="0">` +
				`<xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType>`,
		},
		{
			name: `<all maxOccurs="0"> at the top model-group position`,
			body: `<xs:complexType name="T"><xs:all maxOccurs="0">` +
				`<xs:element name="a" type="xs:string"/></xs:all></xs:complexType>`,
		},
		{
			name: `<group ref= maxOccurs="0"> at the top model-group position`,
			body: `<xs:complexType name="T"><xs:group ref="tns:G" maxOccurs="0"/></xs:complexType>`,
		},
		{
			name: `<choice minOccurs="1" maxOccurs="0"> at the top model-group position`,
			body: `<xs:complexType name="T"><xs:choice minOccurs="1" maxOccurs="0">` +
				`<xs:element name="a" type="xs:string"/></xs:choice></xs:complexType>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, wrap("urn:po", elidedTargets+tc.body))
			if err == nil {
				t.Fatalf("Produce accepted a particle with {min occurs} 1 greater than {max occurs} 0")
			}
			if !strings.Contains(err.Error(), "p-props-correct") {
				t.Fatalf("error = %v, want the p-props-correct verdict", err)
			}
		})
	}
}
