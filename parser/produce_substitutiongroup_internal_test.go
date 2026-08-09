package parser

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// TestProduceElementInlineTypeSkipsAnonymousHeadLookup pins the ORDER of
// §3.3.2.1 dcl.elt.common's {type definition} chain, which the external
// TestProduceElementInlineTypeOutranksSubstitutionGroupHead cannot reach: that
// case gives the head a NAMED type, so clause 3's lookup succeeds there and the
// clause order is unobservable. Here the head's type is an inline <simpleType>,
// the one shape substitutionGroupHeadType still declines as a producer
// limitation after #342 — because produceElement cannot build such a head at all
// — and clause 1 decides this member outright, so that lookup must never run for
// it.
//
// The head deliberately carries a <simpleType> rather than the <complexType>
// this test used before #342: the complexType shape is now MAPPED, so running
// clause 3 for it would no longer fail and the ordering would go unobserved
// again. What the clause order decides is WHICH error the schema gets — the
// spec's verdict on the member's own type, or a fabricated limitation about a
// head type the member never inherits (STYLE E2).
//
// It is package-internal because the mapping is not observable through Produce:
// two DISTINCT anonymous types are not the same type definition, so the same
// member is rejected at finalize by e-props-correct clause 4 (c-vs-sg) whatever
// the producer maps, and every end-to-end run of this shape ends in an error.
// Calling produceElement directly is the only way to read the mapped {type
// definition}, the same reason TestProduceModelGroupDefinitionScopesLocalElements
// is internal.
func TestProduceElementInlineTypeSkipsAnonymousHeadLookup(t *testing.T) {
	const doc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
		<xs:element name="head"><xs:simpleType><xs:restriction base="xs:string"/></xs:simpleType></xs:element>
		<xs:element name="member" substitutionGroup="head"><xs:complexType><xs:sequence/></xs:complexType></xs:element>
	</xs:schema>`
	d, err := ReadDocument("mem://sghead.xsd", strings.NewReader(doc))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	builder := xsd.NewSchemaBuilder()
	sym, err := newSymbols(builder, strict.New())
	if err != nil {
		t.Fatalf("newSymbols: %v", err)
	}
	p := newProducer(d, "", nil, nil, nil, builder, sym)
	p.prescan() // registers head, so clause 3 would find it and decline

	member := topLevelElementNamed(t, d, "member")
	ed, err := p.produceElement(xsd.QName{Local: "member"}, member)
	if err != nil {
		t.Fatalf("produceElement: %v", err)
	}
	inline, ok := ed.TypeDefinition().(xsd.InlineTypeDefinition)
	if !ok {
		t.Fatalf("{type definition} = %#v, want the member's own inline anonymous type", ed.TypeDefinition())
	}
	if got := inline.Definition.Loc(); got.Line != member.Loc().Line {
		t.Fatalf("the inline {type definition} is at %s, want the member's own <complexType> on line %d", got, member.Loc().Line)
	}
}

// topLevelElementNamed returns the top-level <element> of d carrying name.
func topLevelElementNamed(t *testing.T, d *Document, name string) *Element {
	t.Helper()
	for _, child := range d.Root().Children() {
		el, ok := child.(*Element)
		if !ok || !isXSD(el, "element") {
			continue
		}
		if got, has := el.Attr("name"); has && got == name {
			return el
		}
	}
	t.Fatalf("the test document has no top-level <element name=%q>", name)
	return nil
}
