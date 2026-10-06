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
// whatever Name the notation carries, NameChars and non-ASCII ones included,
// and whether its ExternalID is SYSTEM or PUBLIC: a parameter entity, a parsed
// external entity and an internal one are not members — the last even where
// its literal spells NDATA — nor is a name that appears only inside a literal,
// a processing instruction or a comment, while an entity whose name holds
// U+0133, a NameStartChar in [4], is. A reference to a parameter entity that
// is not read — here an external one — ends processing, so a declaration after
// it is not a member (XML 1.0 §5.1). The <!ENTITY> declarations that are no
// [70] EntityDecl are TestEntityDeclIsWellFormed's rows, each a fault.
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
  <!ENTITY % pe SYSTEM "pe.gif">
  <!ENTITY parsed SYSTEM "parsed.xml">
  <!ENTITY text "a literal naming NDATA gif">
  <!ENTITY a SYSTEM "x" NDATA gif>
  <!ENTITY Dĳkstra SYSTEM "x" NDATA gif>
  <!ATTLIST r a CDATA "&#60;!ENTITY inattlist SYSTEM 'x' NDATA gif>">
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
		{"pe", false}, {"parsed", false}, {"text", false}, {"inattlist", false},
		{"inpi", false}, {"incomment", false}, {"gif", false}, {"undeclared", false},
		{"cjk", true}, {"undertie", true}, {"a", true}, {"Dĳkstra", true},
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
// subset. A DOCTYPE out of place is a fault that declares none, a row of
// TestMisplacedDeclarationIsAFault.
func TestHasUnparsedEntityWithoutAnInternalSubset(t *testing.T) {
	for _, doc := range []string{
		`<r/>`,
		`<!DOCTYPE r SYSTEM "r.dtd"><r/>`,
		`<!DOCTYPE r SYSTEM "[<!ENTITY pic SYSTEM 'p' NDATA gif>]"><r/>`,
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
// off: the internal subset is read before it. After a declined reference the
// rest of the subset is only checked: a parameter entity declared there is not
// bound, and one declared before it is not expanded, so neither declares late
// and neither's "junk" is read as stray text.
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
		{`<!DOCTYPE r SYSTEM "x.dtd" [` + late + `]><r/>`, false, true},
		{`<!DOCTYPE r SYSTEM "[%x;]"><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % p "<!ENTITY late SYSTEM 'l' NDATA n>"> %p;]><r/>`, true, true},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x PUBLIC "-//x//y" "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [%undeclared; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [%x; <!ENTITY % x "">` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"><!ENTITY % x "">%x; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % a "&#37;a;"> %a; ` + late + `]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % c '<![INCLUDE[` + late + `]]>'> %c;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> ` + late + ` %x;]><r/>`, false, true},
		{no + `<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, false},
		{yes + `<!DOCTYPE r [<!ENTITY % x SYSTEM "x.ent"> %x; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [%undeclared; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [<!ENTITY % c "<![INCLUDE[]]>"> %c; ` + late + `]><r/>`, false, true},
		{yes + `<!DOCTYPE r [` + late + `]><r/>`, true, true},
		{`<!DOCTYPE r [%undeclared; <!ENTITY % p '` + late + `'> %p;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % p '` + late + `'> %undeclared; %p;]><r/>`, false, false},
		{`<!DOCTYPE r [<!ENTITY % p "junk"> %undeclared; %p; <!ENTITY % q "junk"> %q;]><r/>`, false, false},
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

// A directive outside the document element is a doctypedecl, '<!DOCTYPE' S
// Name, or the document is not well-formed (XML 1.0 [22] prolog, [27] Misc,
// [28] doctypedecl): a keyword run on into the name, with or without an
// internal subset after it, another keyword, the keyword in lower case, a
// markup declaration outside the internal subset, and a directive after the
// document element are faults located at the directive, and the run-on
// subset declares nothing. The keyword followed by any S character reads,
// its subset declaring pic.
func TestDirectiveIsDoctypedecl(t *testing.T) {
	const decl = "<?xml version=\"1.0\"?>\n"
	const subset = ` [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>]`
	const rule = ` is no doctypedecl, '<!DOCTYPE' S Name, and no other directive may stand outside the document element (XML 1.0 [22] prolog, [27] Misc, [28] doctypedecl)`
	for _, tc := range []struct {
		doc  string
		want string // the whole error
	}{
		{decl + `<!DOCTYPEr><r/>`, `t.xml:2:1: [xml-wf] directive "<!DOCTYPEr"` + rule},
		{decl + `<!DOCTYPEr` + subset + `><r/>`, `t.xml:2:1: [xml-wf] directive "<!DOCTYPEr"` + rule},
		{decl + `<!FOO><r/>`, `t.xml:2:1: [xml-wf] directive "<!FOO"` + rule},
		{decl + `<!doctype r><r/>`, `t.xml:2:1: [xml-wf] directive "<!doctype"` + rule},
		{decl + `<r/><!FOO>`, `t.xml:2:5: [xml-wf] directive "<!FOO"` + rule},
		{decl + `<!DOCTYPE r SYSTEM "x.dtd"><!ELEMENT r ANY><r/>`, `t.xml:2:28: [xml-wf] directive "<!ELEMENT"` + rule},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			wantSubsetFault(t, tc.doc, tc.want)
		})
	}
	for _, doc := range []string{
		"<!DOCTYPE r><r/>",
		"<!DOCTYPE\tr><r/>",
		"<!DOCTYPE\nr><r/>",
		"<!DOCTYPE\rr><r/>",
		"<!DOCTYPE\r\nr><r/>",
	} {
		t.Run(doc, func(t *testing.T) {
			drained(t, decl+doc)
		})
	}
	for _, doc := range []string{
		"<!DOCTYPE r" + subset + "><r/>",
		"<!DOCTYPE\tr" + subset + "><r/>",
		"<!DOCTYPE\rr" + subset + "><r/>",
	} {
		t.Run(doc, func(t *testing.T) {
			if !drained(t, decl+doc).HasUnparsedEntity("pic") {
				t.Errorf(`HasUnparsedEntity("pic") = false, want true`)
			}
		})
	}
}

// Between declarations only S and PEReferences may stand (XML 1.0 [28b]
// intSubset, [28a] DeclSep), and a parameter entity referenced there must
// expand to the same (WFC: PE Between Declarations): stray text before or
// after a declaration, at depth 0 or in replacement text, a replacement text
// that is stray whole, a '%' run that is no PEReference ([69]) — standalone
// or not — a ']' in replacement text, a start tag, a '<!' opening no markup
// declaration, a conditional section in the internal subset itself, which
// [28b] does not admit, and text other than S between the subset's ']' and
// '>' ([28] doctypedecl) are faults located at the directive, even where a
// declaration around them would read. Stray text after a declined reference is
// a fault whether or not the document is standalone: §5.1 has the entire
// internal subset checked.
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
		{decl + `<!DOCTYPE r [<!ENTITY % open "&#60;"> %open ` + pic + `]><r/>`, subset + `"%open"` + subsetRule + `, [69] PEReference)`},
		{decl + `<!DOCTYPE r [<!ENTITY % p "<!-- -->"> % p; ` + pic + `]><r/>`, subset + `"%"` + subsetRule + `, [69] PEReference)`},
		{decl + `<!DOCTYPE r [<!ENTITY % p "]"> %p; ` + notation + pic + `]><r/>`, inPE + `"]"` + peRule + `)`},
		{decl + `<!DOCTYPE r [<r/> ` + notation + pic + `]><r/>`, subset + `"<r/>"` + subsetRule + `)`},
		{decl + `<!DOCTYPE r [` + notation + pic + `<!FOO x>]><r/>`, subset + `"<!FOO"` + subsetRule + `)`},
		{decl + `<!DOCTYPE r [<![INCLUDE[ ` + notation + pic + ` ]]>]><r/>`, subset + `"<![INCLUDE["` + subsetRule + `)`},
		{decl + `<!DOCTYPE r [` + notation + pic + `%undeclared; junk]><r/>`, subset + `"junk"` + subsetRule + `)`},
		{yes + `<!DOCTYPE r [` + notation + pic + `%undeclared; junk]><r/>`, subset + `"junk"` + subsetRule + `)`},
		{decl + `<!DOCTYPE r [` + notation + pic + `] junk><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE holds "junk" between its internal subset's ']' and '>', where only S may stand (XML 1.0 [28] doctypedecl)`},
		{decl + `<!DOCTYPE r [` + notation + pic + `] ]><r/>`, `t.xml:2:1: [xml-wf] DOCTYPE holds "]" between its internal subset's ']' and '>', where only S may stand (XML 1.0 [28] doctypedecl)`},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			wantSubsetFault(t, tc.doc, tc.want)
		})
	}
}

// wantSubsetFault reads doc to its first error, which must be the
// well-formedness fault want, whole, and must leave pic undeclared: a faulty
// subset declares nothing. The fault must wrap no cause: the subset's checks
// are definite faults (doc.go's Contract).
func wantSubsetFault(t *testing.T, doc, want string) {
	t.Helper()
	r := xmltree.NewReader("t.xml", strings.NewReader(doc))
	var err error
	for err == nil {
		_, err = r.Token()
	}
	if errors.Is(err, io.EOF) {
		t.Fatalf("read to EOF, want the fault %q", want)
	}
	wantWellFormednessError(t, err)
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err, want)
	}
	if errors.Unwrap(err) != nil {
		t.Errorf("error %v wraps the cause %v, want a definite fault wrapping none", err, errors.Unwrap(err))
	}
	if r.HasUnparsedEntity("pic") {
		t.Errorf("HasUnparsedEntity(%q) = true, want false: the subset is not well-formed", "pic")
	}
}

// The markup the internal subset holds must match its own production, at depth
// 0 and in replacement text alike, and the subset must close, or the document
// is not well-formed: a comment holding "--" ([15] Comment), a processing
// instruction whose target is "xml" in any case or no Name ([16] PI, [17]
// PITarget), a notation declaration that is no NotationDecl ([82], [75],
// [83], [12] PubidLiteral), a '<' outside a markup declaration's literals, a
// comment's among them ([45], [52], [70], [82]), a parameter-entity reference
// inside any markup declaration, an entity value literal included, declared or
// not (WFC: PEs in Internal Subset), any other '%' in an entity value literal
// ([9] EntityValue), replacement text that ends inside a comment, processing
// instruction or markup declaration (WFC: PE Between Declarations), standalone
// or not, a '<' in the DOCTYPE header, a comment's among them, and a subset no
// ']' closes ([28] doctypedecl). A quote inside such a comment must not leave
// the declaration open, or hide the subset. The check runs on after a declined
// reference (§5.1). Each is a fault at the directive that declares nothing.
func TestSubsetMarkupIsWellFormed(t *testing.T) {
	const decl = "<?xml version=\"1.0\"?>\n"
	const yes = "<?xml version=\"1.0\" standalone=\"yes\"?>\n"
	const head = decl + `<!DOCTYPE r [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>`
	const tail = `]><r ent="pic"/>`
	const subset = `t.xml:2:1: [xml-wf] DOCTYPE internal subset `
	const inPE = `t.xml:2:1: [xml-wf] replacement text of a parameter entity referenced between DOCTYPE declarations `
	const comment = `holds a comment with "--" inside it (XML 1.0 [15] Comment)`
	const target = ` is no PITarget (XML 1.0 [16] PI, [17] PITarget)`
	const notation = `holds a <!NOTATION> declaration that is no Name followed by an ExternalID or PublicID (XML 1.0 [82] NotationDecl, [75] ExternalID, [83] PublicID, [12] PubidLiteral)`
	const peRef = ` inside a markup declaration (XML 1.0 WFC: PEs in Internal Subset)`
	const open = ` open where it ends (XML 1.0 WFC: PE Between Declarations, [31] extSubsetDecl)`
	const unclosedSubset = subset + `is closed by no ']' before the directive's '>' (XML 1.0 [28] doctypedecl)`
	const lt = ` inside a markup declaration, outside its literals, where no markup declaration admits a '<' (XML 1.0 [45] elementdecl, [52] AttlistDecl, [70] EntityDecl, [82] NotationDecl)`
	const percent = `holds an entity value literal with a '%' that opens no PEReference (XML 1.0 [9] EntityValue, [69] PEReference)`
	const header = `t.xml:2:1: [xml-wf] DOCTYPE holds "<!--" before its internal subset, where only S, its name and an ExternalID may stand (XML 1.0 [28] doctypedecl)`
	for _, tc := range []struct {
		doc  string
		want string // the whole error
	}{
		{head + `<!-- a -- b -->` + tail, subset + comment},
		{head + `<!-- a --->` + tail, subset + comment},
		{head + `<!ENTITY % p "<!-- a -- b -->"> %p;` + tail, inPE + comment},
		{head + `<?xml x?>` + tail, subset + `holds a processing instruction whose target "xml"` + target},
		{head + `<?XmL?>` + tail, subset + `holds a processing instruction whose target "XmL"` + target},
		{head + `<?1x y?>` + tail, subset + `holds a processing instruction whose target "1x"` + target},
		{head + `<? x?>` + tail, subset + `holds a processing instruction whose target ""` + target},
		{head + `<!ENTITY % p "<?xml x?>"> %p;` + tail, inPE + `holds a processing instruction whose target "xml"` + target},
		{head + `<!NOTATION n SYSTEM>` + tail, subset + notation},
		{head + `<!NOTATION n>` + tail, subset + notation},
		{head + `<!NOTATIONm SYSTEM 'x'>` + tail, subset + notation},
		{head + `<!NOTATION 1m SYSTEM 'x'>` + tail, subset + notation},
		{head + `<!NOTATION m SYSTEM'x'>` + tail, subset + notation},
		{head + `<!NOTATION m SYSTEM 'x' 'y'>` + tail, subset + notation},
		{head + `<!NOTATION m PUBLIC 'p''s'>` + tail, subset + notation},
		{head + `<!NOTATION m PUBLIC 'p' 's' 't'>` + tail, subset + notation},
		{head + `<!NOTATION m PUBLIC 'p{'>` + tail, subset + notation},
		{head + "<!NOTATION m PUBLIC 'p\tq'>" + tail, subset + notation},
		{head + `<!NOTATION m PUBLIC>` + tail, subset + notation},
		{head + `<!NOTATION m FOO 'x'>` + tail, subset + notation},
		{head + `<!ENTITY % p "<!NOTATION m SYSTEM>"> %p;` + tail, inPE + notation},
		{head + `<!ENTITY % s "SYSTEM 'x'"><!NOTATION m %s;>` + tail, subset + `holds the parameter-entity reference %s;` + peRef},
		{head + `<!ENTITY % c "ANY"><!ELEMENT r %c;>` + tail, subset + `holds the parameter-entity reference %c;` + peRef},
		{head + `<!ENTITY % a "x CDATA #IMPLIED"><!ATTLIST r %a;>` + tail, subset + `holds the parameter-entity reference %a;` + peRef},
		{head + `<!ENTITY % m "n"><!ENTITY e SYSTEM 'x' NDATA %m;>` + tail, subset + `holds the parameter-entity reference %m;` + peRef},
		{head + `<!ENTITY % p "<!ELEMENT r &#37;c;>"> %p;` + tail, inPE + `holds the parameter-entity reference %c;` + peRef},
		{head + `<!ENTITY % p "<!-- x"> %p; ` + tail, inPE + `leaves "<!--"` + open},
		{head + `<!ENTITY % p "<?pi"> %p; ` + tail, inPE + `leaves "<?pi"` + open},
		{head + `<!ENTITY % p "<!ENTITY late SYSTEM 'l' NDATA n"> %p;` + tail, inPE + `leaves "<!ENTITY"` + open},
		{head + `<!ENTITY % p "<!ELEMENT r ANY"> %p;` + tail, inPE + `leaves "<!ELEMENT"` + open},
		{head + `<!ENTITY % p "<!ATTLIST r a CDATA #IMPLIED"> %p;` + tail, inPE + `leaves "<!ATTLIST"` + open},
		{head + `<!ENTITY % p "<!NOTATION m SYSTEM 'x'"> %p;` + tail, inPE + `leaves "<!NOTATION"` + open},
		{yes + `<!DOCTYPE r [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n><!ENTITY % p "<!--"> %p;` + tail, inPE + `leaves "<!--"` + open},
		{yes + `<!DOCTYPE r [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n><!ENTITY % p "<?pi"> %p;` + tail, inPE + `leaves "<?pi"` + open},
		{head + `<!ENTITY a "b">><r ent="pic"/>`, unclosedSubset},
		{decl + `<!DOCTYPE r [><r/>`, unclosedSubset},
		{head + `%undeclared; <!NOTATION m SYSTEM>` + tail, subset + notation},
		{head + `%undeclared; <!-- a -- b -->` + tail, subset + comment},
		{head + `%undeclared; <!ENTITY a "b">><r ent="pic"/>`, unclosedSubset},
		{head + `<!ENTITY % x 'y'><!ENTITY a "%x;">` + tail, subset + `holds the parameter-entity reference %x;` + peRef},
		{head + `<!ENTITY a 'v%u;w'>` + tail, subset + `holds the parameter-entity reference %u;` + peRef},
		{head + `<!ENTITY % q "<!ENTITY late SYSTEM 'l' NDATA n>"><!ENTITY % pct "%q;"> %pct;` + tail, subset + `holds the parameter-entity reference %q;` + peRef},
		{head + `<!ENTITY % p "<!ENTITY a '&#37;x;'>"> %p;` + tail, inPE + `holds the parameter-entity reference %x;` + peRef},
		{head + `%undeclared; <!ENTITY a "%x;">` + tail, subset + `holds the parameter-entity reference %x;` + peRef},
		{head + `<!ENTITY a "50%">` + tail, subset + percent},
		{head + `<!ENTITY % p '% x;'>` + tail, subset + percent},
		{head + `<!NOTATION m SYSTEM 'x' <!-- ' -->>` + tail, subset + `holds "<!--"` + lt},
		{head + `<!ELEMENT r ANY <!-- ' -->>` + tail, subset + `holds "<!--"` + lt},
		{head + `<!ATTLIST r a CDATA #IMPLIED <!-- " -->>` + tail, subset + `holds "<!--"` + lt},
		{head + `<!ENTITY e SYSTEM 'u' <!-- c --> NDATA n>` + tail, subset + `holds "<!--"` + lt},
		{head + `<!ELEMENT r <a>>` + tail, subset + `holds "<a>>"` + lt},
		{head + `<!ENTITY % p "<!ELEMENT r ANY <!-- ' -->>"> %p;` + tail, inPE + `holds "<!--"` + lt},
		{head + `%undeclared; <!NOTATION m SYSTEM 'x' <!-- ' -->>` + tail, subset + `holds "<!--"` + lt},
		{decl + `<!DOCTYPE r <!-- ' --> [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>]><r ent="pic"/>`, header},
		{decl + `<!DOCTYPE r <!-- c --> [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>]><r ent="pic"/>`, header},
		{decl + `<!DOCTYPE r SYSTEM 'x' <!-- ' --> [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>]><r ent="pic"/>`, header},
		{decl + `<!DOCTYPE r <!-- ' -->><r/>`, header},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			wantSubsetFault(t, tc.doc, tc.want)
		})
	}
}

// Markup that matches its production reads, pic declared after it: an
// <!ELEMENT> with mixed and children content, an <!ATTLIST> with enumerated
// and #FIXED defaults, a NOTATION with a PublicID alone or with an ExternalID
// of either kind, a PI whose target only starts with "xml", or that has no
// data, comments holding single '-'s, empty or ending "- -->", a '%' in an
// ExternalID's literals, which recognize no parameter-entity reference, and a
// '%' a character reference spells, or a comment, inside an entity value
// literal.
func TestSubsetMarkupControls(t *testing.T) {
	const notation = `<!NOTATION n SYSTEM 'x'>`
	const pic = `<!ENTITY pic SYSTEM 'u' NDATA n>`
	for _, markup := range []string{
		`<!ELEMENT r (#PCDATA|a)*><!ELEMENT a (b,(c|d)+)?><!ELEMENT b (#PCDATA)>`,
		`<!ATTLIST r t (x|y) 'x' f CDATA #FIXED "v" g NOTATION (n) #IMPLIED>`,
		`<!NOTATION m PUBLIC 'p'>`,
		`<!NOTATION m PUBLIC "-//A//DTD x 1.0//EN" 's'>`,
		"<!NOTATION m PUBLIC \"it's (a)+,./:=?;!*#@$_%\r\n\" ''>",
		`<!NOTATION m SYSTEM "" >`,
		"<!NOTATION\tm\nSYSTEM\r\n'%m;'>",
		`<?xml-stylesheet href=a?>`,
		`<?xmlfoo?>`,
		`<?x ?>`,
		`<!---->`,
		`<!-- a - b - -->`,
		`<!ENTITY % p "<!-- a - b --><?xml-stylesheet x?><!NOTATION m PUBLIC 'p'>"> %p;`,
		`<!ENTITY s SYSTEM 'a%x;b'><!ENTITY t PUBLIC 'p' '%'><!ENTITY u SYSTEM '%u;' NDATA n>`,
		`<!ENTITY g "&#37;x; <!-- ' -->"><!ENTITY % q '&#37;g;'>`,
	} {
		for _, doc := range []string{
			`<!DOCTYPE r [` + notation + markup + pic + `]><r/>`,
			`<!DOCTYPE r [` + notation + pic + markup + `]><r/>`,
		} {
			t.Run(doc, func(t *testing.T) {
				if !drained(t, doc).HasUnparsedEntity("pic") {
					t.Errorf("HasUnparsedEntity(%q) = false, want true", "pic")
				}
			})
		}
	}
}

// An <!ENTITY> declaration that is no XML 1.0 [70] EntityDecl is not
// well-formed, at depth 0, in replacement text and after a declined reference
// alike (§5.1): no S after its keyword ([71] GEDecl, [72] PEDecl), a declared
// name that is no Name, a literal run on into it among them ([5]), no
// definition, an entity value literal with text run on after it ([9]), a token
// after an EntityValue ([73] EntityDef, [74] PEDef), a definition that opens no
// ExternalID — an unquoted, missing or run-on literal, a keyword in lower case,
// PUBLIC with one literal or a PubidChar outside [13] ([75], [11], [12], [13])
// — an NDataDecl in a PEDef ([74]), anything but one 'NDATA' S Name after a
// general entity's ExternalID ([76], [5]), a '&' that opens no EntityRef or
// CharRef ([67], [68], [66]) and a character reference naming no Char (WFC:
// Legal Character), referenced or not. Each is a fault at the directive that
// declares nothing.
func TestEntityDeclIsWellFormed(t *testing.T) {
	const decl = "<?xml version=\"1.0\"?>\n"
	const head = decl + `<!DOCTYPE r [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>`
	const tail = `]><r ent="pic"/>`
	const subset = `t.xml:2:1: [xml-wf] DOCTYPE internal subset holds `
	const inPE = `t.xml:2:1: [xml-wf] replacement text of a parameter entity referenced between DOCTYPE declarations holds `
	const noS = `an <!ENTITY> declaration with no S after its keyword (XML 1.0 [71] GEDecl, [72] PEDecl)`
	name := func(n, rule string) string {
		return fmt.Sprintf(`an <!ENTITY> declaration whose name %q is not a Name (XML 1.0 %s, [5] Name)`, n, rule)
	}
	noDef := func(n, rules string) string {
		return fmt.Sprintf(`an <!ENTITY> declaration of %q with no definition (XML 1.0 %s)`, n, rules)
	}
	value := func(v string) string {
		return fmt.Sprintf(`an <!ENTITY> declaration whose entity value %q is no quoted literal followed by S or '>' (XML 1.0 [9] EntityValue, [71] GEDecl, [72] PEDecl)`, v)
	}
	after := func(tok string) string {
		return fmt.Sprintf(`an <!ENTITY> declaration with %q after its entity value, which ends the definition (XML 1.0 [73] EntityDef, [74] PEDef)`, tok)
	}
	extID := func(def string) string {
		return fmt.Sprintf(`an <!ENTITY> declaration whose definition %q is neither an entity value literal nor an ExternalID, 'SYSTEM' S SystemLiteral or 'PUBLIC' S PubidLiteral S SystemLiteral (XML 1.0 [73] EntityDef, [74] PEDef, [75] ExternalID, [11] SystemLiteral, [12] PubidLiteral)`, def)
	}
	const peNData = `a parameter entity declaration with "NDATA" after its ExternalID, where a PEDef admits no NDataDecl (XML 1.0 [72] PEDecl, [74] PEDef)`
	nData := func(rest string) string {
		return fmt.Sprintf(`an <!ENTITY> declaration with %q after its ExternalID, where only an NDataDecl, 'NDATA' S Name, may stand (XML 1.0 [71] GEDecl, [73] EntityDef, [76] NDataDecl)`, rest)
	}
	const ref = `an entity value literal with a '&' that opens no Reference (XML 1.0 [9] EntityValue, [67] Reference, [68] EntityRef, [66] CharRef)`
	legal := func(digits string) string {
		return fmt.Sprintf(`an entity value literal whose character reference &#%s; names no XML character (XML 1.0 [66] CharRef, WFC: Legal Character)`, digits)
	}
	const ext = `<!ENTITY % ext SYSTEM "x.ent"> %ext; `
	for _, tc := range []struct {
		doc  string
		want string // the whole error
	}{
		{head + `<!ENTITYa "x">` + tail, subset + noS},
		{head + `<!ENTITY% p "x">` + tail, subset + noS},
		{head + `<!ENTITY 1x SYSTEM "x">` + tail, subset + name("1x", "[71] GEDecl")},
		{head + `<!ENTITY 1x SYSTEM "x" NDATA n>` + tail, subset + name("1x", "[71] GEDecl")},
		{head + `<!ENTITY a&b SYSTEM "x" NDATA n>` + tail, subset + name("a&b", "[71] GEDecl")},
		{head + `<!ENTITY a"b" SYSTEM "x" NDATA n>` + tail, subset + name(`a"b"`, "[71] GEDecl")},
		{head + `<!ENTITY a"x">` + tail, subset + name(`a"x"`, "[71] GEDecl")},
		{head + `<!ENTITY >` + tail, subset + name("", "[71] GEDecl")},
		{head + `<!ENTITY % 1p "x">` + tail, subset + name("1p", "[72] PEDecl")},
		{head + `<!ENTITY a>` + tail, subset + noDef("a", "[71] GEDecl, [73] EntityDef")},
		{head + `<!ENTITY % p >` + tail, subset + noDef("p", "[72] PEDecl, [74] PEDef")},
		{head + `<!ENTITY a "x"NDATA n>` + tail, subset + value(`"x"NDATA`)},
		{head + `<!ENTITY a "x"y"">` + tail, subset + value(`"x"y""`)},
		{head + `<!ENTITY a "x" "%x;">` + tail, subset + after(`"%x;"`)},
		{head + `<!ENTITY a "x" "y">` + tail, subset + after(`"y"`)},
		{head + `<!ENTITY a "x" NDATA n>` + tail, subset + after("NDATA")},
		{head + `<!ENTITY internal "a literal" NDATA n>` + tail, subset + after("NDATA")},
		{head + `<!ENTITY % p "x" "y">` + tail, subset + after(`"y"`)},
		{head + `<!ENTITY a SYSTEM x NDATA n>` + tail, subset + extID("SYSTEM x NDATA n")},
		{head + `<!ENTITY a SYSTEM>` + tail, subset + extID("SYSTEM")},
		{head + `<!ENTITY onelit PUBLIC "x" NDATA n>` + tail, subset + extID(`PUBLIC "x" NDATA n`)},
		{head + `<!ENTITY a PUBLIC "p">` + tail, subset + extID(`PUBLIC "p"`)},
		{head + `<!ENTITY nos SYSTEM "x"NDATA n>` + tail, subset + extID(`SYSTEM "x"NDATA n`)},
		{head + `<!ENTITY runon SYSTEM "x"y"" NDATA n>` + tail, subset + extID(`SYSTEM "x"y"" NDATA n`)},
		{head + `<!ENTITY nospace SYSTEM"x" NDATA n>` + tail, subset + extID(`SYSTEM"x" NDATA n`)},
		{head + `<!ENTITY pubnospace PUBLIC"p" "x" NDATA n>` + tail, subset + extID(`PUBLIC"p" "x" NDATA n`)},
		{head + `<!ENTITY publits PUBLIC "p""x" NDATA n>` + tail, subset + extID(`PUBLIC "p""x" NDATA n`)},
		{head + `<!ENTITY a system "x">` + tail, subset + extID(`system "x"`)},
		{head + `<!ENTITY a FOO "x">` + tail, subset + extID(`FOO "x"`)},
		{head + `<!ENTITY a PUBLIC "p{" "s">` + tail, subset + extID(`PUBLIC "p{" "s"`)},
		{head + `<!ENTITY a PUBLIC 'it's'' "s">` + tail, subset + extID(`PUBLIC 'it's'' "s"`)},
		{head + `<!ENTITY a PUBLIC "p" s>` + tail, subset + extID(`PUBLIC "p" s`)},
		{head + `<!ENTITY % p SYSTEM x>` + tail, subset + extID("SYSTEM x")},
		{head + `<!ENTITY % p SYSTEM "x" NDATA n>` + tail, subset + peNData},
		{head + `<!ENTITY % pe SYSTEM "pe.gif" NDATA n>` + tail, subset + peNData},
		{head + `<!ENTITY a SYSTEM "x" NDATA n n>` + tail, subset + nData("NDATA n n")},
		{head + `<!ENTITY a SYSTEM "x" NDATA>` + tail, subset + nData("NDATA")},
		{head + `<!ENTITY bracket SYSTEM "x" NDATA n]>` + tail, subset + nData("NDATA n]")},
		{head + `<!ENTITY keyword SYSTEM "x" XNDATA n>` + tail, subset + nData("XNDATA n")},
		{head + `<!ENTITY a SYSTEM "x" ndata n>` + tail, subset + nData("ndata n")},
		{head + `<!ENTITY amp SYSTEM "x" NDATA g&h>` + tail, subset + nData("NDATA g&h")},
		{head + `<!ENTITY digit SYSTEM "x" NDATA 1gif>` + tail, subset + nData("NDATA 1gif")},
		{head + `<!ENTITY times SYSTEM "x" NDATA a×b>` + tail, subset + nData("NDATA a×b")},
		{head + `<!ENTITY undertiefirst SYSTEM "x" NDATA ‿b>` + tail, subset + nData("NDATA ‿b")},
		{head + `<!ENTITY a "&;">` + tail, subset + ref},
		{head + `<!ENTITY a "a & b">` + tail, subset + ref},
		{head + `<!ENTITY a "&b">` + tail, subset + ref},
		{head + `<!ENTITY a "&1b;">` + tail, subset + ref},
		{head + `<!ENTITY a "&#;">` + tail, subset + ref},
		{head + `<!ENTITY a "&#x;">` + tail, subset + ref},
		{head + `<!ENTITY a "&#X41;">` + tail, subset + ref},
		{head + `<!ENTITY a "&#12a;">` + tail, subset + ref},
		{head + `<!ENTITY a "&#xG;">` + tail, subset + ref},
		{head + `<!ENTITY a "&#-1;">` + tail, subset + ref},
		{head + `<!ENTITY a "&#0;">` + tail, subset + legal("0")},
		{head + `<!ENTITY a "&#1;">` + tail, subset + legal("1")},
		{head + `<!ENTITY a "&#xD800;">` + tail, subset + legal("xD800")},
		{head + `<!ENTITY a "&#xFFFE;">` + tail, subset + legal("xFFFE")},
		{head + `<!ENTITY a "&#xFFFF;">` + tail, subset + legal("xFFFF")},
		{head + `<!ENTITY a "&#x110000;">` + tail, subset + legal("x110000")},
		{head + `<!ENTITY a "&#99999999999;">` + tail, subset + legal("99999999999")},
		{head + `<!ENTITY % nul "&#0;"> %nul;` + tail, subset + legal("0")},
		{head + `<!ENTITY % p "<!ENTITY a 'x' 'y'>"> %p;` + tail, inPE + after(`'y'`)},
		{head + ext + `<!ENTITY a "x" NDATA n>` + tail, subset + after("NDATA")},
		{head + ext + `<!ENTITY a SYSTEM "x" NDATA n n>` + tail, subset + nData("NDATA n n")},
		{head + ext + `<!ENTITY a "&#0;">` + tail, subset + legal("0")},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			wantSubsetFault(t, tc.doc, tc.want)
		})
	}
}

// An <!ENTITY> declaration that matches XML 1.0 [70] EntityDecl reads, pic
// declared after it and every declaration processed: an internal general or
// parameter entity, an ExternalID of either kind with or without an NDataDecl,
// whose notation need not be declared (VC: Notation Declared binds a
// validating processor only), a '%' a character reference spells, a reference
// to an undeclared entity, which an entity value bypasses (§4.4.7), character
// references to Chars, the predefined entities, a non-BMP character, '<' and
// '>' in an entity value, '%' and '#' in a SystemLiteral, '%' in a
// PubidLiteral and "'" in a '"'-quoted one ([13]), empty literals, S of every
// kind before '>' and a name declared twice.
func TestEntityDeclControls(t *testing.T) {
	const notation = `<!NOTATION n SYSTEM 'x'>`
	const pic = `<!ENTITY pic SYSTEM 'u' NDATA n>`
	for _, markup := range []string{
		`<!ENTITY a "x">`,
		`<!ENTITY % p "x">`,
		`<!ENTITY % p SYSTEM "x">`,
		`<!ENTITY a PUBLIC "p" "s" NDATA n>`,
		`<!ENTITY a PUBLIC "p" 's'>`,
		`<!ENTITY a "&#37;x;">`,
		`<!ENTITY a "&b;">`,
		`<!ENTITY a "&amp;&lt;&gt;&apos;&quot;">`,
		`<!ENTITY a "&#x9;&#xa;&#65;&#x10FFFD;&#xFFFD;">`,
		`<!ENTITY a "𝄞">`,
		`<!ENTITY a "<b>x</b>">`,
		`<!ENTITY a SYSTEM "%p;">`,
		`<!ENTITY a SYSTEM "x#frag">`,
		`<!ENTITY a PUBLIC "50%" "s">`,
		`<!ENTITY a PUBLIC "it's" "s">`,
		`<!ENTITY a SYSTEM "x" NDATA undeclared>`,
		`<!ENTITY a ""><!ENTITY b SYSTEM ''><!ENTITY c PUBLIC '' "">`,
		"<!ENTITY\ta\r\nSYSTEM\n\"x\"\tNDATA n \t\r\n>",
		`<!ENTITY a "x" ><!ENTITY % p 'y'  >`,
		`<!ENTITY a "x"><!ENTITY a "y">`,
	} {
		for _, doc := range []string{
			`<!DOCTYPE r [` + notation + markup + pic + `]><r/>`,
			`<!DOCTYPE r [` + notation + pic + markup + `]><r/>`,
		} {
			t.Run(doc, func(t *testing.T) {
				r := drained(t, doc)
				if !r.HasUnparsedEntity("pic") {
					t.Errorf("HasUnparsedEntity(%q) = false, want true", "pic")
				}
				if !r.AllDeclarationsProcessed() {
					t.Error("AllDeclarationsProcessed() = false, want true: every declaration is read")
				}
			})
		}
	}
}

// Only "?>" closes a processing instruction in the internal subset (XML 1.0
// [16] PI): a '>', '<' or quote inside one is data, so the DOCTYPE reads as one
// directive whose every declaration is processed (§5.1), the declarations
// after the PI included, and no prolog text is left behind it. The rows with a
// '>', '<' or quote in the PI failed with "unexpected EOF", or dropped what
// followed the '>' and read "]>" as prolog character data, before the decoder
// read the PI whole; the rest are controls, read either way: a PI holding none
// of these, a comment holding a quote and a '>', and a PI in the prolog
// outside the DOCTYPE. The pic rows that drop pic without the fix are the
// truncation shape #2214 and #753 must keep reading.
func TestSubsetProcessingInstructionHoldsMarkupCharacters(t *testing.T) {
	const notation = `<!NOTATION n SYSTEM 'x'>`
	const pic = `<!ENTITY pic SYSTEM 'u' NDATA n>`
	for _, tc := range []struct {
		doc     string
		text    string // the document element's character data
		wantPic bool
	}{
		{`<!DOCTYPE r [<?x it's?>]><r/>`, "", false},
		{`<!DOCTYPE r [<?x say "a?>]><r/>`, "", false},
		{`<!DOCTYPE r [<?x a < b?>]><r/>`, "", false},
		{`<!DOCTYPE r [<?x a > b?>` + notation + pic + `]><r ent="pic"/>`, "", true},
		{`<!DOCTYPE r [` + notation + pic + `<?x a > b?>]><r ent="pic"/>`, "", true},
		{`<!DOCTYPE r [<?x a > b?><!ENTITY e 'v'>]><r>&e;</r>`, "v", false},
		{`<!DOCTYPE r [<?x 'a' > "b" <c>?>` + notation + pic + `]><r ent="pic"/>`, "", true},
		{`<!DOCTYPE r [<?x a b?>` + notation + pic + `]><r ent="pic"/>`, "", true},
		{`<!DOCTYPE r [<!-- it's > -->` + notation + pic + `]><r ent="pic"/>`, "", true},
		{`<?xml-stylesheet href='a'?><!DOCTYPE r [` + notation + pic + `]><?y '>?><r ent="pic"/>`, "", true},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			r := xmltree.NewReader("t.xml", strings.NewReader(tc.doc))
			var text strings.Builder
			for started := false; ; {
				n, err := r.Token()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					t.Fatalf("Token: %v", err)
				}
				switch n := n.(type) {
				case *xmltree.StartElement:
					started = true
				case *xmltree.CharData:
					if !started {
						t.Errorf("prolog character data %q, want none", n.Data())
					}
					text.WriteString(n.Data())
				}
			}
			if text.String() != tc.text {
				t.Errorf("character data = %q, want %q", text.String(), tc.text)
			}
			if got := r.HasUnparsedEntity("pic"); got != tc.wantPic {
				t.Errorf("HasUnparsedEntity(%q) = %v, want %v", "pic", got, tc.wantPic)
			}
			if !r.AllDeclarationsProcessed() {
				t.Error("AllDeclarationsProcessed() = false, want true: every declaration is read")
			}
		})
	}
}

// A processing instruction in the internal subset that no "?>" closes is not
// well-formed (XML 1.0 [16] PI, [28b] intSubset), whatever '>' and ']' follow
// it: the decoder reads it to the end of input and faults there.
func TestSubsetUnclosedProcessingInstructionIsNotWellFormed(t *testing.T) {
	const doc = `<!DOCTYPE r [<?x a > b]><r/>`
	_, err := collect(t, "t.xml", doc)
	wantWellFormednessError(t, err)
	const want = `t.xml:1:29: [xml-wf] XML syntax error on line 1: unexpected EOF`
	if err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
}

// What may stand between declarations is never stray: S of every kind, before
// and after the subset's ']', a PEReference expanding to a declaration, a
// comment or processing instruction holding text that would be stray between
// declarations, a '%' or ']' inside an ATTLIST default, a ']' or a '%' a
// character reference spells inside an EntityValue literal, and INCLUDE,
// IGNORE and PE-keyword conditional sections in an internal parameter entity's
// replacement text, with the ']' and ']]>' that close them, which XML 1.0
// leaves unresolved there (ruled on #1733): each section is declined, as
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
		`<!DOCTYPE r [` + notation + `<!ENTITY g "junk ] &#37;p;"><!ENTITY % q "&#37;p;">` + pic + `]><r/>`,
		`<!DOCTYPE r [` + notation + pic + `<!ENTITY % c "<![INCLUDE[ <!ENTITY e 'v'> ]]> junk"> %c;]><r/>`,
		`<!DOCTYPE r [` + notation + pic + `<!ENTITY % c "<![IGNORE[ junk ] &#37; ]]>"> %c;]><r/>`,
		`<!DOCTYPE r [` + notation + pic + `<!ENTITY % k "INCLUDE"><!ENTITY % c "<![&#37;k;[ junk ]]>"> %c;]><r/>`,
		yes + `<!DOCTYPE r [` + notation + `<!ENTITY % c "<![IGNORE[ junk ] &#37; ]]> junk"> %c; ` + pic + `]><r/>`,
	} {
		t.Run(doc, func(t *testing.T) {
			if !drained(t, doc).HasUnparsedEntity("pic") {
				t.Errorf("HasUnparsedEntity(%q) = false, want true", "pic")
			}
		})
	}
}
