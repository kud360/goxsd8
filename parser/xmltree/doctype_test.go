package xmltree_test

import (
	"errors"
	"fmt"
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
// comment, nor a declaration whose ExternalID and NDataDecl do not read by
// position — PUBLIC with one literal, an unquoted system identifier, a
// notation name carrying ']', a token after the notation name. A reference to
// a parameter entity that is not read — here an external one — cuts the scan
// off, so a declaration after it is not a member (XML 1.0 §5.1).
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
  <!ENTITY onelit PUBLIC "x" NDATA gif>
  <!ENTITY bare SYSTEM x NDATA gif>
  <!ENTITY bracket SYSTEM "x" NDATA gif]>
  <!ENTITY trailing SYSTEM "x" NDATA gif gif>
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
		{"pic", true}, {"pub", true}, {"after", false},
		{"pe", false}, {"parsed", false}, {"text", false}, {"internal", false}, {"inattlist", false},
		{"inpi", false}, {"incomment", false}, {"gif", false}, {"undeclared", false},
		{"onelit", false}, {"bare", false}, {"bracket", false}, {"trailing", false},
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

// An internal parameter entity referenced between declarations is expanded in
// place, so an unparsed entity declared in its replacement text is a member:
// directly, behind a character reference, from a parameter entity declared
// inside another's replacement text, and through a reference a character
// reference spells.
func TestHasUnparsedEntityExpandsInternalParameterEntities(t *testing.T) {
	r := drained(t, `<!DOCTYPE r [
  <!ENTITY % p "<!ENTITY e SYSTEM 'u.bin' NDATA n>">
  %p;
  <!ENTITY % charref "&#60;!ENTITY f SYSTEM 'f.bin' NDATA n&#x3E;">
  %charref;
  <!ENTITY % decl "<!ENTITY &#37; inner '&#60;!ENTITY g SYSTEM &#34;g.bin&#34; NDATA n>'>">
  %decl;
  %inner;
  <!ENTITY % outer "&#37;p2;">
  <!ENTITY % p2 "<!ENTITY h SYSTEM 'h.bin' NDATA n>">
  %outer;
  <!NOTATION n SYSTEM "x">
]><r/>`)
	for _, name := range []string{"e", "f", "g", "h"} {
		if !r.HasUnparsedEntity(name) {
			t.Errorf("HasUnparsedEntity(%q) = false, want true: its declaration is in an expanded internal parameter entity", name)
		}
	}
	if !r.AllDeclarationsProcessed() {
		t.Error("AllDeclarationsProcessed() = false, want true: every parameter entity is internal and read")
	}
}

// peChain declares parameter entities p0 through p<n>, each p<i> referencing
// p<i-1> and p0 declaring the unparsed entity deep, then references p<n>: an
// expansion nested n+1 deep.
func peChain(n int) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE r [<!ENTITY % p0 "<!ENTITY deep SYSTEM 'd' NDATA n>">`)
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, `<!ENTITY %% p%d "&#37;p%d;">`, i, i-1)
	}
	fmt.Fprintf(&b, `%%p%d;]><r/>`, n)
	return b.String()
}

// peBlowUp declares parameter entities whose replacement texts reference the
// level below ten times over, six levels deep, over a 1000-byte comment: a
// million-fold expansion, which the scan must decline rather than perform.
func peBlowUp(xmlDecl string) string {
	var b strings.Builder
	b.WriteString(xmlDecl + `<!DOCTYPE r [<!ENTITY % l0 "<!--` + strings.Repeat("x", 1000) + `-->">`)
	for i := 1; i <= 6; i++ {
		fmt.Fprintf(&b, `<!ENTITY %% l%d "%s">`, i, strings.Repeat(fmt.Sprintf("&#37;l%d;", i-1), 10))
	}
	b.WriteString(`%l6; <!ENTITY late SYSTEM "late.bin" NDATA n>]><r/>`)
	return b.String()
}

// AllDeclarationsProcessed is false exactly when a declaration went unread —
// an external subset, or a parameter-entity reference the reader declines —
// and the first declined reference cuts off every entity declaration after it
// (XML 1.0 §5.1) unless the XML declaration says standalone="yes", which
// admits them without making the DTD read. An external subset cuts nothing
// off: the internal subset is read before it.
func TestAllDeclarationsProcessed(t *testing.T) {
	const late = `<!ENTITY late SYSTEM "late.bin" NDATA n>`
	const yes = `<?xml version="1.0" standalone="yes"?>`
	const no = `<?xml version="1.0" standalone="no"?>`
	for _, tc := range []struct {
		doc       string
		processed bool
		late      bool
	}{
		{`<r/>`, true, false},
		{`<!DOCTYPE r [` + late + `]><r/>`, true, true},
		{`<!DOCTYPE r SYSTEM "x.dtd"><r/>`, false, false},
		{`<!DOCTYPE r PUBLIC "-//x//y" "x.dtd"><r/>`, false, false},
		{`<!DOCTYPE r SYSTEM "x.dtd"><!ELEMENT r ANY><r/>`, false, false},
		{`<!DOCTYPE r SYSTEM "x.dtd" [` + late + `]><r/>`, false, true},
		{`<!DOCTYPE r SYSTEM "[%x;]"><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x PUBLIC "-//x//y" "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [%undeclared; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [%x; <!ENTITY % x "">` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"><!ENTITY % x "">%x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % a "&#37;a;"> %a; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % q "<!ENTITY late SYSTEM 'l' NDATA n>"><!ENTITY % pct "%q;"> %pct;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % nul "&#0;"> %nul; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % open "&#60;"> %open ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % c "<![INCLUDE[` + late + `]]>"> %c;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> ` + late + ` %x;]><r/>`, false, true},
		{no + `<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{yes + `<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [%undeclared; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [<!ENTITY % c "<![INCLUDE[]]>"> %c; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [` + late + `]><r/>`, true, true},
		{peBlowUp(""), false, false},
		{peBlowUp(yes), false, true},
	} {
		r := drained(t, tc.doc)
		if got := r.AllDeclarationsProcessed(); got != tc.processed {
			t.Errorf("%.200s: AllDeclarationsProcessed() = %t, want %t", tc.doc, got, tc.processed)
		}
		if got := r.HasUnparsedEntity("late"); got != tc.late {
			t.Errorf("%.200s: HasUnparsedEntity(%q) = %t, want %t", tc.doc, "late", got, tc.late)
		}
	}
}

// maxPEDepthForTest is the reader's expansion depth bound, restated: a chain
// nested exactly that deep is read and one level more is declined.
const maxPEDepthForTest = 64

// An expansion nested as deep as the bound is read; one level deeper is
// declined, not read.
func TestParameterEntityDepthBound(t *testing.T) {
	for _, tc := range []struct {
		depth int
		read  bool
	}{{maxPEDepthForTest, true}, {maxPEDepthForTest + 1, false}} {
		r := drained(t, peChain(tc.depth-1))
		if got := r.HasUnparsedEntity("deep"); got != tc.read {
			t.Errorf("depth %d: HasUnparsedEntity(%q) = %t, want %t", tc.depth, "deep", got, tc.read)
		}
		if got := r.AllDeclarationsProcessed(); got != tc.read {
			t.Errorf("depth %d: AllDeclarationsProcessed() = %t, want %t", tc.depth, got, tc.read)
		}
	}
}
