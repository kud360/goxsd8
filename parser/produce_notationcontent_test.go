package parser_test

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// notationWith builds a schema whose one <notation> carries body as its content,
// on a line of its own, so a rejection can be asserted to name both positions:
// the <notation> opens at line 2 column 1 and body at line 3 column 1.
func notationWith(body string) string {
	return wrap("", "\n"+`<xs:notation name="jpeg" public="image/jpeg" system="viewer.exe">`+"\n"+body+"\n</xs:notation>")
}

// TestProduceNotationIllegalChildRejected pins the content model of <notation>:
// xs:notation extends xs:annotated and adds attributes only
// (xmlschema11-1.md:5701), so "(annotation?)" (:3376, :4426) is the whole of it
// and any other child element is a schema for schema documents fault. Before
// this check produceNotation read name/system/public and never looked at
// elem.Children(), silently accepting every one of these documents.
//
// The four subtests are the suite's own witnesses: notatF018 and notatF066 are
// the <complexContent>/<simpleContent> pair, notatF010 puts an <appinfo> outside
// the <annotation> that alone may hold it, and notatG002 is well-formed XML in
// no namespace — foreign markup is no more admitted than XSD markup, since
// xs:openAttrs opens attributes and not elements.
//
// The fault carries no rule ID (§5.1's first bullet, :4296): §3.14.3 and §3.14.4
// both read "None as such.", so a verdict here would be fabricated (STYLE E2).
func TestProduceNotationIllegalChildRejected(t *testing.T) {
	for _, tc := range []struct {
		name      string
		body      string
		wantLocal string
	}{
		{"complexContent (notatF018)", "<xs:complexContent/>", "complexContent"},
		{"simpleContent (notatF066)", "<xs:simpleContent/>", "simpleContent"},
		{"appinfo (notatF010)", "<xs:appinfo/>", "appinfo"},
		{"foreign element (notatG002)", "<a><b/></a>", "a"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, notationWith(tc.body))
			if err == nil {
				t.Fatalf("Produce accepted a <notation> carrying <%s>", tc.wantLocal)
			}
			if _, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, want a plain grammar fault rather than a rule verdict", err)
			}
			if !strings.Contains(err.Error(), "<"+tc.wantLocal+"> at "+produceURI+":3:1") {
				t.Errorf("error = %v, want it to name the <%s> at line 3", err, tc.wantLocal)
			}
			if !strings.Contains(err.Error(), "<notation> at "+produceURI+":2:1") {
				t.Errorf("error = %v, want it to name the enclosing <notation> at line 2", err)
			}
		})
	}
}

// TestProduceNotationCharacterDataRejected pins the other half of the same
// content model: xs:annotated descends from xs:openAttrs, which restricts
// xs:anyType and opens ATTRIBUTES alone, so <notation>'s content is element-only
// and character data is outside it. notatG001 is plain text and notatG003 is
// text arriving through entity references, which the reader has already expanded
// by the time the tree holds it — both suite-invalid.
func TestProduceNotationCharacterDataRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{"plain text (notatG001)", "Some Text"},
		{"expanded entities (notatG003)", `&amp; &quot; '`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := produce(t, notationWith(tc.body))
			if err == nil {
				t.Fatal("Produce accepted a <notation> carrying character data")
			}
			if _, ok := xsderr.RuleOf(err); ok {
				t.Errorf("error = %v, want a plain grammar fault rather than a rule verdict", err)
			}
			if !strings.Contains(err.Error(), "character data at "+produceURI+":2:") {
				t.Errorf("error = %v, want it to name the character data run, which starts on line 2 at the newline after the start tag", err)
			}
			if !strings.Contains(err.Error(), "<notation> at "+produceURI+":2:1") {
				t.Errorf("error = %v, want it to name the enclosing <notation> at line 2", err)
			}
		})
	}
}

// TestProduceNotationLegalContentAccepted pins what the content model DOES
// admit, so the rejection above cannot be widened into one: an empty <notation>,
// one whose only content is the whitespace between its tags, and one carrying
// the single <annotation> "(annotation?)" allows. notatB001 and notatF004 are
// the suite's valid witnesses for the last two.
func TestProduceNotationLegalContentAccepted(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  string
	}{
		{"no content", wrap("", `<xs:notation name="jpeg" public="image/jpeg" system="viewer.exe"/>`)},
		{"whitespace only", notationWith("\t")},
		{"one annotation (notatF004)", notationWith("<xs:annotation/>")},
		{"annotation with documentation", notationWith("<xs:annotation><xs:documentation>a notation</xs:documentation></xs:annotation>")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schema, err := produce(t, tc.doc)
			if err != nil {
				t.Fatalf("Produce: %v", err)
			}
			notations := schema.Notations()
			if len(notations) != 1 {
				t.Fatalf("Notations() = %d declarations, want 1", len(notations))
			}
			if got := notations[0].Name().Local; got != "jpeg" {
				t.Errorf("Name().Local = %q, want %q", got, "jpeg")
			}
		})
	}
}
