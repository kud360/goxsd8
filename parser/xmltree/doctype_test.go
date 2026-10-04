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
// and only an external general entity with an NDATA notation is unparsed —
// whatever Name the notation carries, NameChars and non-ASCII ones included: a
// parameter entity, a parsed external entity and an internal one are not
// members — the last even where an NDATA keyword follows its literal — nor is
// a name that appears only inside a literal, a processing instruction or a
// comment, nor a declaration whose ExternalID and NDataDecl do not read by
// position — PUBLIC with one literal, an unquoted system identifier, a
// notation name carrying ']', a token after the notation name, a keyword that
// is not NDATA — nor one that is not well-formed (XML 1.0 [75], [76], [5]): no
// S between SYSTEM or PUBLIC and its first literal, between two literals, or
// between the literal and NDATA, a literal with text run on after it, a
// notation name carrying '&' or U+00D7 '×', or starting with a digit or with
// U+203F '‿', a NameChar that is no NameStartChar. Nor is an entity whose
// declared name is not a Name (XML 1.0 [71] GEDecl, [5]) — `1x`, `a&b`, or
// `a"b"`, a literal run on into it, which leaves a later `a` declared — while
// one whose name holds U+0133, a NameStartChar in [4], is. A reference to
// a parameter entity that is not read — here an external one — cuts the scan
// off, so a declaration after it is not a member (XML 1.0 §5.1).
func TestHasUnparsedEntityReadsTheInternalSubset(t *testing.T) {
	r := drained(t, `<?xml version="1.0"?>
<!DOCTYPE r SYSTEM "r.dtd" [
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY pic SYSTEM "pic.gif" NDATA gif>
  <!ENTITY pub PUBLIC "-//x//y" 'pub.gif' NDATA gif>
  <!ENTITY namechars SYSTEM "x" NDATA _g-i.f:9>
  <!ENTITY accented SYSTEM "x" NDATA ïmage>
  <!ENTITY cjk SYSTEM "x" NDATA 画像>
  <!ENTITY undertie SYSTEM "x" NDATA a‿·b>
  <!ENTITY times SYSTEM "x" NDATA a×b>
  <!ENTITY undertiefirst SYSTEM "x" NDATA ‿b>
  <!ENTITY % pe SYSTEM "pe.gif" NDATA gif>
  <!ENTITY parsed SYSTEM "parsed.xml">
  <!ENTITY text "a literal naming NDATA gif">
  <!ENTITY internal "a literal" NDATA gif>
  <!ENTITY onelit PUBLIC "x" NDATA gif>
  <!ENTITY bare SYSTEM x NDATA gif>
  <!ENTITY bracket SYSTEM "x" NDATA gif]>
  <!ENTITY trailing SYSTEM "x" NDATA gif gif>
  <!ENTITY keyword SYSTEM "x" XNDATA gif>
  <!ENTITY nos SYSTEM "x"NDATA gif>
  <!ENTITY runon SYSTEM "x"y"" NDATA gif>
  <!ENTITY nospace SYSTEM"x" NDATA gif>
  <!ENTITY pubnospace PUBLIC"p" "x" NDATA gif>
  <!ENTITY publits PUBLIC "p""x" NDATA gif>
  <!ENTITY amp SYSTEM "x" NDATA g&h>
  <!ENTITY digit SYSTEM "x" NDATA 1gif>
  <!ENTITY 1x SYSTEM "x" NDATA gif>
  <!ENTITY a&b SYSTEM "x" NDATA gif>
  <!ENTITY a"b" SYSTEM "x" NDATA gif>
  <!ENTITY a SYSTEM "x" NDATA gif>
  <!ENTITY Dĳkstra SYSTEM "x" NDATA gif>
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
		{"pic", true}, {"pub", true}, {"namechars", true}, {"accented", true}, {"after", false},
		{"pe", false}, {"parsed", false}, {"text", false}, {"internal", false}, {"inattlist", false},
		{"inpi", false}, {"incomment", false}, {"gif", false}, {"undeclared", false},
		{"onelit", false}, {"bare", false}, {"bracket", false}, {"trailing", false},
		{"keyword", false}, {"nos", false}, {"runon", false}, {"amp", false}, {"digit", false},
		{"nospace", false}, {"pubnospace", false}, {"publits", false},
		{"cjk", true}, {"undertie", true}, {"times", false}, {"undertiefirst", false},
		{"1x", false}, {"a&b", false}, {`a"b"`, false}, {"a", true}, {"Dĳkstra", true},
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
		{`<!DOCTYPE r [<!ENTITY % p "<!ENTITY late SYSTEM 'l' NDATA n>"> %p;]><r/>`, true, true},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x PUBLIC "-//x//y" "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [%undeclared; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [%x; <!ENTITY % x "">` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"><!ENTITY % x "">%x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % a "&#37;a;"> %a; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % q "<!ENTITY late SYSTEM 'l' NDATA n>"><!ENTITY % pct "%q;"> %pct;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % nul "&#0;"> %nul; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % c "<![INCLUDE[` + late + `]]>"> %c;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> ` + late + ` %x;]><r/>`, false, true},
		{no + `<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{yes + `<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [%undeclared; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [<!ENTITY % c "<![INCLUDE[]]>"> %c; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [` + late + `]><r/>`, true, true},
		{`<!DOCTYPE r [<!ENTITY % p "<!ENTITY late SYSTEM 'l' NDATA n"> %p;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % p "<!ENTITY pic SYSTEM 'u' NDATA n"> %p; ` + late + `]><r/>`, false, false},
		{yes + `<!DOCTYPE r [<!ENTITY % p "<!ENTITY pic SYSTEM 'u' NDATA n"> %p; ` + late + `]><r/>`, false, true},
		{`<!DOCTYPE r [<!ENTITY % p "<!ELEMENT r ANY"> %p; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % p "<!--"> %p; ` + late + `]><r/>`, false, false},
		{yes + `<!DOCTYPE r [<!ENTITY % p "<!--"> %p; ` + late + `]><r/>`, false, true},
		{`<!DOCTYPE r [<!ENTITY % p "<?pi"> %p; ` + late + `]><r/>`, false, false},
		{yes + `<!DOCTYPE r [<!ENTITY % p "<?pi"> %p; ` + late + `]><r/>`, false, true},
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

// A DOCTYPE's document type name is a Name or the document is not well-formed
// (XML 1.0 [28] doctypedecl, [5] Name): a name starting with a digit, one
// holding '&', a missing one, a literal, and one with a literal run on into it
// with no S between are faults located at the directive, while a name
// starting with U+00C0, a NameStartChar in [4], reads.
func TestDoctypeNameIsAName(t *testing.T) {
	const decl = "<?xml version=\"1.0\"?>\n"
	for _, tc := range []struct {
		doc  string
		want string // the error's opening; "" when the document reads
	}{
		{`<!DOCTYPE r><r/>`, ""},
		{`<!DOCTYPE Àr><Àr/>`, ""},
		{`<!DOCTYPE r[]><r/>`, ""},
		{`<!DOCTYPE 1r><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE document type name "1r" is not a Name`},
		{`<!DOCTYPE a&b><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE document type name "a&b" is not a Name`},
		{`<!DOCTYPE><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE document type name "" is not a Name`},
		{`<!DOCTYPE [<!ENTITY e "x">]><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE document type name "" is not a Name`},
		{`<!DOCTYPE "r"SYSTEM "x.dtd"><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE document type name "\"r\"SYSTEM" is not a Name`},
		{`<!DOCTYPE r"x"><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE document type name "r\"x\"" is not a Name`},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			_, err := collect(t, "t.xml", decl+tc.doc)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("collect: %v, want the document read", err)
				}
				return
			}
			wantWellFormednessError(t, err)
			if !strings.HasPrefix(err.Error(), tc.want) {
				t.Errorf("error = %q, want it to open %q", err, tc.want)
			}
		})
	}
}

// Between declarations only S and PEReferences may stand (XML 1.0 [28b]
// intSubset, [28a] DeclSep), and a parameter entity referenced there must
// expand to the same (WFC: PE Between Declarations): stray text before or
// after a declaration, at depth 0 or in replacement text, a replacement text
// that is stray whole, a '%' run that is no PEReference ([69]) — standalone
// or not — a ']' in replacement text, a start tag, and text other than S
// between the subset's ']' and '>' ([28] doctypedecl) are faults located at
// the directive, even where a declaration around them would read.
func TestSubsetStrayTextIsNotWellFormed(t *testing.T) {
	const decl = "<?xml version=\"1.0\"?>\n"
	const yes = "<?xml version=\"1.0\" standalone=\"yes\"?>\n"
	const notation = `<!NOTATION n SYSTEM 'x'>`
	const pic = `<!ENTITY pic SYSTEM 'u' NDATA n>`
	const subset = `t.xml:2:1: [xml-wf] DOCTYPE internal subset holds `
	const subsetRule = ` between declarations, which is no markup declaration, PEReference or S (XML 1.0 [28b] intSubset, [28a] DeclSep`
	const inPE = `t.xml:2:1: [xml-wf] replacement text of a parameter entity referenced between DOCTYPE declarations holds `
	const peRule = ` between declarations, which is no markup declaration, PEReference or S (XML 1.0 WFC: PE Between Declarations, [31] extSubsetDecl`
	for _, tc := range []struct {
		doc  string
		want string // the whole error
	}{
		{decl + `<!DOCTYPE r [` + notation + pic + ` junk]><r/>`, subset + `"junk"` + subsetRule + `)`},
		{decl + `<!DOCTYPE r [junk ` + notation + pic + `]><r/>`, subset + `"junk"` + subsetRule + `)`},
		{decl + `<!DOCTYPE r [` + notation + `<!ENTITY % p "` + pic + ` junk"> %p;]><r/>`, inPE + `"junk"` + peRule + `)`},
		{decl + `<!DOCTYPE r [` + notation + `<!ENTITY % p "junk ` + pic + `"> %p;]><r/>`, inPE + `"junk"` + peRule + `)`},
		{decl + `<!DOCTYPE r [` + notation + `<!ENTITY % p "x"> %p; ` + pic + `]><r/>`, inPE + `"x"` + peRule + `)`},
		{decl + `<!DOCTYPE r [%1x; ` + notation + pic + `]><r/>`, subset + `"%1x;"` + subsetRule + `, [69] PEReference)`},
		{yes + `<!DOCTYPE r [%1x; ` + notation + pic + `]><r/>`, subset + `"%1x;"` + subsetRule + `, [69] PEReference)`},
		{decl + `<!DOCTYPE r [<!ENTITY % 1p "` + pic + `"> %1p;]><r/>`, subset + `"%1p;"` + subsetRule + `, [69] PEReference)`},
		{decl + `<!DOCTYPE r [<!ENTITY % open "&#60;"> %open ` + pic + `]><r/>`, subset + `"%open"` + subsetRule + `, [69] PEReference)`},
		{decl + `<!DOCTYPE r [<!ENTITY % p "%q;"> % p; ` + pic + `]><r/>`, subset + `"%"` + subsetRule + `, [69] PEReference)`},
		{decl + `<!DOCTYPE r [<!ENTITY % p "]"> %p; ` + notation + pic + `]><r/>`, inPE + `"]"` + peRule + `)`},
		{decl + `<!DOCTYPE r [<r/> ` + notation + pic + `]><r/>`, subset + `"<r/>"` + subsetRule + `)`},
		{decl + `<!DOCTYPE r [` + notation + pic + `] junk><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE holds "junk" between its internal subset's ']' and '>', where only S may stand (XML 1.0 [28] doctypedecl)`},
		{decl + `<!DOCTYPE r [` + notation + pic + `] ]><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE holds "]" between its internal subset's ']' and '>', where only S may stand (XML 1.0 [28] doctypedecl)`},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			_, err := collect(t, "t.xml", tc.doc)
			wantWellFormednessError(t, err)
			if err.Error() != tc.want {
				t.Errorf("error = %q, want %q", err, tc.want)
			}
		})
	}
}

// What may stand between declarations is never stray: S of every kind, before
// and after the subset's ']', a PEReference expanding to a declaration, a
// comment or processing instruction holding any text, a '%' or ']' inside an
// ATTLIST default, a ']' or PEReference inside an EntityValue literal, and
// INCLUDE, IGNORE and PE-keyword conditional sections in an internal parameter
// entity's replacement text, with the ']' and ']]>' that close them, which XML
// 1.0 leaves unresolved there (ruled on #1733): each section is declined, as
// doctypeEntities states, never rejected. pic is declared in every one.
func TestSubsetDeclSepIsNotStray(t *testing.T) {
	const yes = `<?xml version="1.0" standalone="yes"?>`
	const notation = `<!NOTATION n SYSTEM 'x'>`
	const pic = `<!ENTITY pic SYSTEM 'u' NDATA n>`
	for _, doc := range []string{
		`<!DOCTYPE r [` + notation + pic + `]><r/>`,
		`<!DOCTYPE r [` + notation + `<!ENTITY % p "` + pic + `"> %p;]><r/>`,
		"<!DOCTYPE r [ \t\r\n" + notation + "\n\t" + pic + "\r\n] \t\n><r/>",
		`<!DOCTYPE r [` + notation + `<?pi junk % ] ?>` + pic + `]><r/>`,
		`<!DOCTYPE r [` + notation + `<!ENTITY % p "<!-- junk ] --><?pi junk ] ?>"> %p;` + pic + `]><r/>`,
		`<!DOCTYPE r [` + notation + `<!ATTLIST r a CDATA "junk % ] %p;">` + pic + `]><r/>`,
		`<!DOCTYPE r [` + notation + `<!ENTITY g "junk ] %p;"><!ENTITY % q "%p;">` + pic + `]><r/>`,
		`<!DOCTYPE r [` + notation + pic + `<!ENTITY % c "<![INCLUDE[ <!ENTITY e 'v'> ]]> junk"> %c;]><r/>`,
		`<!DOCTYPE r [` + notation + pic + `<!ENTITY % c "<![IGNORE[ junk ] % ]]>"> %c;]><r/>`,
		`<!DOCTYPE r [` + notation + pic + `<!ENTITY % k "INCLUDE"><!ENTITY % c "<![%k;[ junk ]]>"> %c;]><r/>`,
		yes + `<!DOCTYPE r [` + notation + `<!ENTITY % c "<![IGNORE[ junk ] % ]]> junk"> %c; ` + pic + `]><r/>`,
	} {
		t.Run(doc, func(t *testing.T) {
			if !drained(t, doc).HasUnparsedEntity("pic") {
				t.Errorf("HasUnparsedEntity(%q) = false, want true", "pic")
			}
		})
	}
}
