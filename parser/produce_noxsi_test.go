package parser_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// xsiNS is the namespace no-xsi (§3.2.6.4) closes to attribute declarations.
const xsiNS = xsd.XMLSchemaInstanceNS

// wrapXSITarget wraps body in a <schema> whose own targetNamespace IS the xsi
// namespace — the shape both the rejected and the accepted cases below share, so
// the only thing separating them is how each declaration's own {target
// namespace} resolves. extra carries the <schema> attributes a row varies
// (attributeFormDefault).
func wrapXSITarget(extra, body string) string {
	return `<xs:schema xmlns:xs="` + xsdNS + `" targetNamespace="` + xsiNS +
		`" xmlns:xsi="` + xsiNS + `" ` + extra + `>` + body + `</xs:schema>`
}

// TestProduceXSIAttributeDeclarationRejected pins no-xsi (§3.2.6.4): an
// attribute declaration whose {target namespace} is the xsi namespace is
// rejected, "whether local or top-level" and whatever its local name.
//
// One row per CODE PATH and per ARM, because reverting each call site fails
// them three different ways. A TOP-LEVEL reserved name (xsi:type) collides with
// the §3.2.7 declaration every schema holds by definition and is charged
// sch-props-correct clause 2 — a generic duplicate-name fault standing in for
// the rule that governs it. A top-level NON-reserved name (xsi:foo) collides
// with nothing and is accepted outright. A LOCAL declaration is accepted
// whatever its name, reserved included, because a local one enters no by-name
// index to collide in. No W3C suite fixture uses a reserved name, so that arm is
// pinned here or nowhere (#1446).
//
// The message assertion pins the SUBJECT: the charge names the declaration whose
// property broke the rule, so a helper reading some other declaration's pair
// cannot satisfy the row by charging the right rule at the wrong component.
func TestProduceXSIAttributeDeclarationRejected(t *testing.T) {
	// A slice, not a map: subtest order is output (STYLE D2). Every body starts on
	// line 2 and puts the offending <attribute> on wantLine, so the charged
	// position is pinned to the declaration's own element.
	cases := []struct {
		name     string
		doc      string
		wantName string
		wantLine int
	}{
		{
			// §3.2.2.1: a top-level declaration's {target namespace} IS the
			// ancestor <schema>'s targetNamespace. Reserved name.
			name:     "top-level, reserved name",
			doc:      wrapXSITarget("", "\n"+`<xs:attribute name="type" type="xs:string"/>`),
			wantName: "type",
			wantLine: 2,
		},
		{
			// The arm the §3.2.6.4 parenthetical does NOT exempt: it carves out the
			// four §3.2.7 DECLARATIONS, not the namespace, so an unreserved name in
			// it is as forbidden as a reserved one.
			name:     "top-level, non-reserved name",
			doc:      wrapXSITarget("", "\n"+`<xs:attribute name="foo" type="xs:string"/>`),
			wantName: "foo",
			wantLine: 2,
		},
		{
			// §3.2.2.2 clause 2.2: attributeFormDefault lifts an unqualified local
			// declaration into the document's namespace. This is attKb018a, which
			// differs from the ACCEPTED attKb018 in nothing else.
			name: "local in <attributeGroup>, attributeFormDefault=qualified, reserved name",
			doc: wrapXSITarget(`attributeFormDefault="qualified"`, "\n"+
				`<xs:attributeGroup name="attG">`+"\n"+
				`<xs:attribute name="nil"/>`+"\n"+
				`</xs:attributeGroup>`),
			wantName: "nil",
			wantLine: 3,
		},
		{
			// §3.2.2.2 clause 2.1: form on the declaration itself, with no
			// attributeFormDefault to read.
			name: "local in <complexType>, form=qualified, non-reserved name",
			doc: wrapXSITarget("", "\n"+
				`<xs:complexType name="CT">`+"\n"+
				`<xs:attribute name="foo" form="qualified"/>`+"\n"+
				`</xs:complexType>`),
			wantName: "foo",
			wantLine: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, tc.doc)
			assertRule(t, err, "no-xsi")
			var xe *xsderr.Error
			if !errors.As(err, &xe) {
				t.Fatalf("error %v carries no *xsderr.Error", err)
			}
			if xe.Loc.Line != tc.wantLine {
				t.Fatalf("charged at line %d, want the <attribute> at line %d (%v)", xe.Loc.Line, tc.wantLine, err)
			}
			want := "attribute declaration {" + xsiNS + "}" + tc.wantName +
				" has the XML Schema Instance namespace as its {target namespace}"
			if !strings.Contains(xe.Msg, want) {
				t.Fatalf("message = %q, want it to name the offending declaration: %q", xe.Msg, want)
			}
		})
	}
}

// TestProduceUnqualifiedLocalAttributeInXSIDocumentAccepted is the guard against
// the wrong implementation of the test above: reading the ancestor <schema>'s
// targetNamespace as a stand-in for every attribute in the document. That
// shortcut is §3.2.2.1's rule for a TOP-LEVEL declaration and is wrong for a
// local one — §3.2.2.2 clause 3 leaves an UNQUALIFIED local attribute in no
// namespace at all, however the enclosing document is namespaced, so no-xsi
// never reaches it.
//
// Both containers §3.2.2 admits for a local <attribute> are covered, because
// they are the two the W3C suite declares VALID for exactly this reason —
// attKb018 (<attributeGroup>) and attKc018 (<complexType>) — and a shortcut
// implementation false-rejects them both.
func TestProduceUnqualifiedLocalAttributeInXSIDocumentAccepted(t *testing.T) {
	cases := []struct {
		name string
		doc  string
	}{
		{
			name: "local in <attributeGroup>",
			doc: wrapXSITarget("", `<xs:attributeGroup name="attG">`+
				`<xs:attribute name="aga1"/></xs:attributeGroup>`+
				`<xs:complexType name="CT"><xs:attributeGroup ref="xsi:attG"/></xs:complexType>`),
		},
		{
			name: "local in <complexType>",
			doc: wrapXSITarget("", `<xs:complexType name="CT">`+
				`<xs:attribute name="ca1"/></xs:complexType>`),
		},
		{
			// localTargetNS's other unqualified branch: form present and read, so
			// attributeFormDefault is never consulted (§3.2.2.2 clause 2.1 failing).
			name: "local with form=unqualified",
			doc: wrapXSITarget(`attributeFormDefault="qualified"`,
				`<xs:complexType name="CT">`+
					`<xs:attribute name="ca1" form="unqualified"/></xs:complexType>`),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := produce(t, tc.doc)
			if err != nil {
				t.Fatalf("Produce: %v, want an unqualified local attribute in an xsi-namespaced document accepted: §3.2.2.2 clause 3 gives it an absent {target namespace}", err)
			}
			uses := topComplexTypeIn(t, s, xsd.QName{Space: xsiNS, Local: "CT"}).AttributeUses()
			if len(uses) != 1 {
				t.Fatalf("{attribute uses} = %d, want the one local attribute declared", len(uses))
			}
			if got := uses[0].DeclarationName(); got.Space != "" {
				t.Fatalf("{target namespace} of %s is present, want ·absent· (§3.2.2.2 clause 3)", got)
			}
		})
	}
}
