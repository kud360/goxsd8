package parser

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// overrideElementFrom reads a one-<override> schema document under a FIXED URI
// and returns the <override> element. The URI is fixed because an override set's
// identity is built from its entries' source locations, so two documents can be
// compared for identity only when they agree on everything ahead of the child
// under test.
func overrideElementFrom(t *testing.T, body string) *Element {
	t.Helper()
	const uri = "mem://main.xsd"
	doc, err := ReadDocument(uri, strings.NewReader(
		`<xs:schema xmlns:xs="`+xsd.XMLSchemaNS+`" targetNamespace="urn:a">`+body+`</xs:schema>`))
	if err != nil {
		t.Fatalf("ReadDocument: %v", err)
	}
	el := childElement(doc.Root(), xsd.XMLSchemaNS, "override")
	if el == nil {
		t.Fatal("the test document has no top-level <override>")
	}
	return el
}

// TestNewOverrideSetShadowedChildIsInert pins the two halves of §F.2 clause 1's
// ($replacement, $original)[1] that are not observable through [Parse].
//
// The first is that the FIRST of two children sharing an (element type, name)
// pair is the one the lookup returns — a set whose index held the second would
// still assemble, just wrongly, and only the substitution's own body would
// differ.
//
// The second is the identity decision: the discarded child contributes nothing
// to the transformation, so it contributes nothing to the set's [overrideSet.id]
// either, and two <override> elements differing ONLY in it produce the same
// docKey — the load-once document identity — and therefore read the overridden
// document ONCE, which is the outcome §4.2.5's note on sch-props-correct clause
// 2 asks for ("multiple equivalent overrides of the same schema document will
// not constitute a violation"). That equality cannot be shown through [Parse]:
// the identity string carries each entry's source location, so two <override>
// elements in two different documents differ in it whatever their children are
// (the standing GAP(xsd) on textual override identity, see [parser] doc.go), and
// the losing child's contribution can only be isolated against a document
// identical up to that child.
func TestNewOverrideSetShadowedChildIsInert(t *testing.T) {
	const winner = `<xs:element name="doc" type="xs:date"/>`
	const loser = `<xs:element name="doc" type="xs:int"/>`
	log := slog.New(slog.DiscardHandler)

	withLoser := newOverrideSet(overrideElementFrom(t,
		`<xs:override schemaLocation="lib.xsd">`+winner+loser+`</xs:override>`), log)
	if withLoser == nil {
		t.Fatal("newOverrideSet returned the identity set for an <override> declaring a substitution")
	}
	if len(withLoser.entries) != 1 {
		t.Fatalf("override set has %d entries, want the one surviving substitution", len(withLoser.entries))
	}
	key := componentKey{kind: "element", name: "doc"}
	repl, matched := withLoser.index[key]
	if !matched {
		t.Fatalf("override set does not substitute for %v", key)
	}
	if got, _ := repl.Attr("type"); got != "xs:date" {
		t.Fatalf("substitution for %v is type=%q, want the FIRST child's xs:date (§F.2 clause 1 takes [1] of the replacement sequence)", key, got)
	}

	without := newOverrideSet(overrideElementFrom(t,
		`<xs:override schemaLocation="lib.xsd">`+winner+`</xs:override>`), log)
	if without == nil {
		t.Fatal("newOverrideSet returned the identity set for an <override> declaring a substitution")
	}
	if withLoser.key() != without.key() {
		t.Fatalf("override key with a shadowed child = %q, without = %q; a child §F.2 clause 1 copies nowhere must not change the set's identity", withLoser.key(), without.key())
	}
	dup := docKey{resolved: "lib.xsd", namespace: "urn:a", override: withLoser.key()}
	sole := docKey{resolved: "lib.xsd", namespace: "urn:a", override: without.key()}
	if dup != sole {
		t.Fatalf("docKey with a shadowed child = %#v, without = %#v; the two overrides yield the same Dold′ and must read it once", dup, sole)
	}
}
