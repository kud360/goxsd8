package xmltree_test

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/parser/xmltree"
)

// drained is a reader over doc read to io.EOF, so every declaration its
// DOCTYPE carries has been seen.
func drained(t *testing.T, doc string) *xmltree.Reader {
	t.Helper()
	r := xmltree.NewReader("doc.xml", strings.NewReader(doc))
	for {
		_, err := r.Token()
		if errors.Is(err, io.EOF) {
			return r
		}
		if err != nil {
			t.Fatalf("Token: %v", err)
		}
	}
}

// HasUnparsedEntity answers from the internal subset's <!ENTITY> declarations,
// and only an external general entity with an NDATA notation is unparsed: a
// parameter entity, a parsed external entity and an internal one are not
// members — the last even where an NDATA keyword follows its literal — nor is
// a name that appears only inside a literal, a processing instruction or a
// comment. A parameter-entity reference is stepped over, and a declaration
// after it still counts.
func TestHasUnparsedEntityReadsTheInternalSubset(t *testing.T) {
	r := drained(t, `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "r.dtd" [
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY pic SYSTEM "pic.gif" NDATA gif>
  <!ENTITY pub PUBLIC "-//x//y" 'pub.gif' NDATA gif>
  <!ENTITY % pe SYSTEM "pe.gif" NDATA gif>
  <!ENTITY parsed SYSTEM "parsed.xml">
  <!ENTITY text "a literal naming NDATA gif">
  <!ENTITY internal "a literal" NDATA gif>
  <!ATTLIST r a CDATA "<!ENTITY inattlist SYSTEM 'x' NDATA gif>">
  <?pi <!ENTITY inpi SYSTEM "x" NDATA gif> ?>
  <!-- <!ENTITY incomment SYSTEM "x" NDATA gif> -->
  %pe;
  <!ENTITY after SYSTEM "after.gif" NDATA gif>
]>
<r/>`)
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"pic", true}, {"pub", true}, {"after", true},
		{"pe", false}, {"parsed", false}, {"text", false}, {"internal", false}, {"inattlist", false},
		{"inpi", false}, {"incomment", false}, {"gif", false}, {"undeclared", false},
	} {
		if got := r.HasUnparsedEntity(tc.name); got != tc.want {
			t.Errorf("HasUnparsedEntity(%q) = %t, want %t", tc.name, got, tc.want)
		}
	}
}

// The first declaration of an entity binds (XML 1.0 §4.2): a name declared
// parsed and then redeclared with NDATA is not an unparsed entity, and one
// declared unparsed first stays one.
func TestHasUnparsedEntityKeepsTheFirstDeclaration(t *testing.T) {
	r := drained(t, `<!DOCTYPE r [
  <!ENTITY a "parsed first">
  <!ENTITY a SYSTEM "a.gif" NDATA gif>
  <!ENTITY b SYSTEM "b.gif" NDATA gif>
  <!ENTITY b "parsed second">
]><r/>`)
	if r.HasUnparsedEntity("a") {
		t.Error(`HasUnparsedEntity("a") = true, want false: its first declaration is parsed`)
	}
	if !r.HasUnparsedEntity("b") {
		t.Error(`HasUnparsedEntity("b") = false, want true: its first declaration is unparsed`)
	}
}

// A document with no DOCTYPE, or one with no internal subset, declares no
// unparsed entity — a '[' inside the external identifier's literal opens no
// subset — and neither does a directive inside an element, which is no
// doctypedecl.
func TestHasUnparsedEntityWithoutAnInternalSubset(t *testing.T) {
	for _, doc := range []string{
		`<r/>`,
		`<!DOCTYPE r SYSTEM "r.dtd"><r/>`,
		`<!DOCTYPE r SYSTEM "[<!ENTITY pic SYSTEM 'p' NDATA gif>]"><r/>`,
		`<r><!DOCTYPE r [<!ENTITY pic SYSTEM "p" NDATA gif>]></r>`,
	} {
		if drained(t, doc).HasUnparsedEntity("pic") {
			t.Errorf("%s: HasUnparsedEntity(%q) = true, want false", doc, "pic")
		}
	}
}

// The answer is final once the document element's start tag has been read.
func TestHasUnparsedEntityIsFinalAtTheDocumentElement(t *testing.T) {
	r := xmltree.NewReader("doc.xml", strings.NewReader(`<!DOCTYPE r [<!ENTITY pic SYSTEM "p" NDATA gif>]><r><x/></r>`))
	n, err := r.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if _, ok := n.(*xmltree.StartElement); !ok {
		t.Fatalf("first node = %T, want the document element's *StartElement", n)
	}
	if !r.HasUnparsedEntity("pic") {
		t.Error(`HasUnparsedEntity("pic") = false at the document element, want true`)
	}
}
