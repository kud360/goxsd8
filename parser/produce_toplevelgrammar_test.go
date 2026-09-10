package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestProduceUnmappedTopLevelChildRejected pins what a <schema> may hold: its
// content model (xmlschema11-1.md:4554) is xs:composition* then an optional
// <defaultOpenContent> then xs:schemaTop*, with annotations interspersed, and
// neither group has a wildcard arm or a lax position — so an XSD-namespace child
// named anything else is admitted by no particle and the document is not fully
// valid against the schema for schema documents (§5.1's first bullet, :4296).
//
// Before this check every row here was silently ACCEPTED: the census recorded
// the child as UnmappedNoDispatch and run's dispatch skipped it, so a one-word
// typo in a root construct compiled to a schema declaring nothing and
// `goxsd8 parse` exited 0 (#1380).
//
// The rows are the suite's own witnesses, one per name, so a guard narrowed to
// the identity-constraint names or to the particle names is caught here rather
// than in the lane score. The last row is a name the suite does not carry at
// all: the guard is the complement of a vocabulary, not a denylist.
//
// The fault carries no rule ID (§5.1's first bullet again): §2.4 clause 1
// (sd-valid, :615) is anchored but absent from Appendix B's three tables, so a
// cvc-*/src-*/cos-* verdict here would be fabricated (STYLE E2, xsderr/doc.go).
func TestProduceUnmappedTopLevelChildRejected(t *testing.T) {
	for _, tc := range []struct {
		name  string
		child string
	}{
		{"unique (idA020)", `<xs:unique name="u"><xs:selector xpath="."/><xs:field xpath="@a"/></xs:unique>`},
		{"key (idB020)", `<xs:key name="k"><xs:selector xpath="."/><xs:field xpath="@a"/></xs:key>`},
		{"keyref (idC020)", `<xs:keyref name="r" refer="k"><xs:selector xpath="."/><xs:field xpath="@a"/></xs:keyref>`},
		{"selector (idD007)", `<xs:selector xpath="."/>`},
		{"field (idE007)", `<xs:field xpath="@a"/>`},
		{"extension (mgP059)", `<xs:extension base="xs:string"/>`},
		{"restriction (mgP060)", `<xs:restriction base="xs:string"/>`},
		{"sequence (mgP061)", `<xs:sequence/>`},
		{"choice (mgP062)", `<xs:choice/>`},
		{"any (notatF005)", `<xs:any/>`},
		{"all", `<xs:all/>`},
		{"a name no production spells", `<xs:bogusElement name="foo"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// On its own line, so the rejection can be asserted to name the child at
			// line 3 column 1 while the <schema> opens on line 1.
			_, err := produce(t, wrap("", "\n\n"+tc.child+"\n"))
			if err == nil {
				t.Fatalf("Produce accepted a top-level %s", tc.child)
			}
			if _, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, want a plain grammar fault rather than a rule verdict", err)
			}
			local := strings.SplitN(strings.TrimPrefix(tc.child, "<xs:"), " ", 2)[0]
			local = strings.TrimSuffix(strings.TrimSuffix(local, "/>"), ">")
			// The subject and its location are asserted as ONE prefix: two locations
			// swapped inside the message leave every substring of it present (#1048).
			want := "parser: <" + local + "> at " + produceURI + ":3:1 fills no position"
			if !strings.HasPrefix(err.Error(), want) {
				t.Errorf("error = %v, want it to open %q", err, want)
			}
			if !strings.Contains(err.Error(), "the <schema> at "+produceURI+":1:1") {
				t.Errorf("error = %v, want it to name the enclosing <schema> at line 1", err)
			}
		})
	}
}

// TestProduceTopLevelCompositionAndAnnotationAccepted pins the other side of the
// guard for the three admitted names no other test of this package writes at the
// top level beside a declaration — an over-broad rejection would take them with
// it. The six named declaration kinds and <notation> are covered by the census
// tests and by TestProduceMisplacedNotationRejected's accepted rows.
func TestProduceTopLevelCompositionAndAnnotationAccepted(t *testing.T) {
	doc := wrap("",
		`<xs:annotation><xs:documentation>ok</xs:documentation></xs:annotation>`+
			`<xs:defaultOpenContent mode="suffix"><xs:any/></xs:defaultOpenContent>`+
			`<xs:annotation><xs:appinfo>ok</xs:appinfo></xs:annotation>`+
			`<xs:element name="e" type="xs:string"/>`)
	schema, err := produce(t, doc)
	if err != nil {
		t.Fatalf("Produce: %v", err)
	}
	if got := len(schema.Elements()); got != 1 {
		t.Errorf("Elements() = %d declarations, want 1", got)
	}
}
