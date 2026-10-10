package xmltree_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An <!ELEMENT> declaration that is no XML 1.0 [45] elementdecl, or an
// <!ATTLIST> declaration that is no [52] AttlistDecl, is not well-formed, at
// depth 0, in replacement text and after a declined reference alike (§5.1):
// no S after the keyword, between an element type name and its contentspec or
// before an AttDef, AttType or DefaultDecl ([45], [52], [53]), a name that is
// no Name ([5]), a contentspec that is not 'EMPTY' or 'ANY' in upper case
// ([46]), a children model with an empty particle, mixed separators, a
// '#PCDATA', an unbalanced parenthesis or S before an occurrence indicator
// ([47]–[50]), a Mixed naming an element type without ")*" or with an
// indicator on "(#PCDATA)" ([51]), an AttType keyword in lower case or run on
// into what follows ([54]–[56]), a NotationType or Enumeration with no S after
// 'NOTATION', an empty, non-Name or non-Nmtoken member or no '|' between two
// ([58], [59], [7]), a missing or lower-case DefaultDecl, or '#FIXED' with no S
// and AttValue after it ([60]), and a default value holding '<', a '&' that
// opens no Reference or a character reference naming no Char ([10], [66]–[68],
// WFC: Legal Character). Each is a fault at the directive that declares
// nothing.
func TestElementAndAttlistDeclAreWellFormed(t *testing.T) {
	const decl = "<?xml version=\"1.0\"?>\n"
	const head = decl + `<!DOCTYPE r [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>`
	const tail = `]><r ent="pic"/>`
	const subset = `t.xml:2:1: [xml-wf] DOCTYPE internal subset holds `
	const inPE = `t.xml:2:1: [xml-wf] replacement text of a parameter entity referenced between DOCTYPE declarations holds `
	const ext = `<!ENTITY % ext SYSTEM "x.ent"> %ext; `
	const elemNoS = `an <!ELEMENT> declaration with no S after its keyword (XML 1.0 [45] elementdecl)`
	elemName := func(n string) string {
		return fmt.Sprintf(`an <!ELEMENT> declaration whose element type name %q is not a Name (XML 1.0 [45] elementdecl, [5] Name)`, n)
	}
	elemSpace := func(at string) string {
		return fmt.Sprintf(`an <!ELEMENT> declaration of "r" with %q where S and a contentspec must stand (XML 1.0 [45] elementdecl)`, at)
	}
	const keyword, mixedRule, childRule = "[46] contentspec", "[46] contentspec, [51] Mixed", "[46] contentspec, [47] children, [48] cp, [49] choice, [50] seq"
	spec := func(spec, rule string) string {
		return fmt.Sprintf(`an <!ELEMENT> declaration of "r" whose content specification %q is not 'EMPTY', 'ANY', Mixed or children (XML 1.0 %s)`, spec, rule)
	}
	const attNoS = `an <!ATTLIST> declaration with no S after its keyword (XML 1.0 [52] AttlistDecl)`
	attElem := func(n string) string {
		return fmt.Sprintf(`an <!ATTLIST> declaration whose element type name %q is not a Name (XML 1.0 [52] AttlistDecl, [5] Name)`, n)
	}
	attDef := func(at string) string {
		return fmt.Sprintf(`an <!ATTLIST> declaration of "r" with %q where S and an AttDef, or '>', must stand (XML 1.0 [52] AttlistDecl, [53] AttDef)`, at)
	}
	attName := func(n string) string {
		return fmt.Sprintf(`an <!ATTLIST> declaration of "r" whose attribute name %q is not a Name (XML 1.0 [53] AttDef, [5] Name)`, n)
	}
	attType := func(a, at string) string {
		return fmt.Sprintf(`an <!ATTLIST> declaration of "r" whose attribute %q has %q where S and an AttType must stand (XML 1.0 [53] AttDef, [54] AttType, [55] StringType, [56] TokenizedType, [57] EnumeratedType)`, a, at)
	}
	notation := func(at string) string {
		return fmt.Sprintf(`an <!ATTLIST> declaration of "r" whose attribute "a" has %q where S and a parenthesized list of Names must follow 'NOTATION' (XML 1.0 [58] NotationType, [5] Name)`, at)
	}
	enum := func(at string) string {
		return fmt.Sprintf(`an <!ATTLIST> declaration of "r" whose attribute "a" has the enumerated type %q, which is no Enumeration (XML 1.0 [59] Enumeration, [7] Nmtoken)`, at)
	}
	dflt := func(at string) string {
		return fmt.Sprintf(`an <!ATTLIST> declaration of "r" whose attribute "a" has %q where S and a DefaultDecl, '#REQUIRED', '#IMPLIED' or an AttValue with ('#FIXED' S)? before it, must stand (XML 1.0 [53] AttDef, [60] DefaultDecl)`, at)
	}
	const value = `an <!ATTLIST> declaration of "r" whose attribute "a" has a default value`
	const lt = value + ` with a '<' in it (XML 1.0 [10] AttValue)`
	const ref = value + ` with a '&' that opens no Reference (XML 1.0 [10] AttValue, [67] Reference, [68] EntityRef, [66] CharRef)`
	legal := func(digits string) string {
		return fmt.Sprintf(value+` whose character reference &#%s; names no XML character (XML 1.0 [66] CharRef, WFC: Legal Character)`, digits)
	}
	for _, tc := range []struct {
		doc  string
		want string // the whole error
	}{
		{head + `<!ELEMENT r (a,,b)>` + tail, subset + spec("(a,,b)", childRule)},
		{head + `<!ELEMENT r (#PCDATA|a)>` + tail, subset + spec("(#PCDATA|a)", mixedRule)},
		{head + `<!ELEMENT r ANYTHING>` + tail, subset + spec("ANYTHING", keyword)},
		{head + `<!ELEMENTr ANY>` + tail, subset + elemNoS},
		{head + `<!ELEMENT 1r ANY>` + tail, subset + elemName("1r")},
		{head + `<!ELEMENT r(a)>` + tail, subset + elemSpace("(a)")},
		{head + `<!ELEMENT r>` + tail, subset + elemSpace("")},
		{head + `<!ELEMENT r empty>` + tail, subset + spec("empty", keyword)},
		{head + `<!ELEMENT r EMPTY ANY>` + tail, subset + spec("EMPTY ANY", keyword)},
		{head + `<!ELEMENT r a>` + tail, subset + spec("a", keyword)},
		{head + `<!ELEMENT r (a|b,c)>` + tail, subset + spec("(a|b,c)", childRule)},
		{head + `<!ELEMENT r (a,b|c)>` + tail, subset + spec("(a,b|c)", childRule)},
		{head + `<!ELEMENT r ()>` + tail, subset + spec("()", childRule)},
		{head + `<!ELEMENT r (a|)>` + tail, subset + spec("(a|)", childRule)},
		{head + `<!ELEMENT r (a ?)>` + tail, subset + spec("(a ?)", childRule)},
		{head + `<!ELEMENT r (a) +>` + tail, subset + spec("(a) +", childRule)},
		{head + `<!ELEMENT r (a)+*>` + tail, subset + spec("(a)+*", childRule)},
		{head + `<!ELEMENT r (a))>` + tail, subset + spec("(a))", childRule)},
		{head + `<!ELEMENT r ((a)>` + tail, subset + spec("((a)", childRule)},
		{head + `<!ELEMENT r (a)(b)>` + tail, subset + spec("(a)(b)", childRule)},
		{head + `<!ELEMENT r (a,#PCDATA)>` + tail, subset + spec("(a,#PCDATA)", childRule)},
		{head + `<!ELEMENT r (#pcdata)>` + tail, subset + spec("(#pcdata)", childRule)},
		{head + `<!ELEMENT r (1a)>` + tail, subset + spec("(1a)", childRule)},
		{head + `<!ELEMENT r (#PCDATA)+>` + tail, subset + spec("(#PCDATA)+", mixedRule)},
		{head + `<!ELEMENT r (#PCDATA)?>` + tail, subset + spec("(#PCDATA)?", mixedRule)},
		{head + `<!ELEMENT r (#PCDATA|a)+>` + tail, subset + spec("(#PCDATA|a)+", mixedRule)},
		{head + `<!ELEMENT r (#PCDATA|a) *>` + tail, subset + spec("(#PCDATA|a) *", mixedRule)},
		{head + `<!ELEMENT r (#PCDATA|a*)*>` + tail, subset + spec("(#PCDATA|a*)*", mixedRule)},
		{head + `<!ELEMENT r (#PCDATA,a)*>` + tail, subset + spec("(#PCDATA,a)*", mixedRule)},
		{head + `<!ELEMENT r (#PCDATA|)*>` + tail, subset + spec("(#PCDATA|)*", mixedRule)},
		{head + `<!ELEMENT r (#PCDATAa)>` + tail, subset + spec("(#PCDATAa)", mixedRule)},
		{head + `<!ATTLIST r a CDATA>` + tail, subset + dflt("")},
		{head + `<!ATTLIST r a (x|) #IMPLIED>` + tail, subset + enum("(x|)")},
		{head + `<!ATTLIST r a CDATA "<">` + tail, subset + lt},
		{head + `<!ATTLIST r a CDATA 'a<b'>` + tail, subset + lt},
		{head + `<!ATTLISTr a CDATA #IMPLIED>` + tail, subset + attNoS},
		{head + `<!ATTLIST>` + tail, subset + attNoS},
		{head + `<!ATTLIST 1r>` + tail, subset + attElem("1r")},
		{head + `<!ATTLIST r 1a CDATA #IMPLIED>` + tail, subset + attName("1a")},
		{head + `<!ATTLIST r a>` + tail, subset + attType("a", "")},
		{head + `<!ATTLIST r a(x) #IMPLIED>` + tail, subset + attType("a", "(x)")},
		{head + `<!ATTLIST r a cdata #IMPLIED>` + tail, subset + attType("a", "cdata")},
		{head + `<!ATTLIST r a CDATA#IMPLIED>` + tail, subset + attType("a", "CDATA#IMPLIED")},
		{head + `<!ATTLIST r a "x" #IMPLIED>` + tail, subset + attType("a", `"x"`)},
		{head + `<!ATTLIST r a ID #IMPLIED b>` + tail, subset + attType("b", "")},
		{head + `<!ATTLIST r a notation (n) #IMPLIED>` + tail, subset + attType("a", "notation")},
		{head + `<!ATTLIST r a NOTATION(n) #IMPLIED>` + tail, subset + notation("(n)")},
		{head + `<!ATTLIST r a NOTATION (1n) #IMPLIED>` + tail, subset + notation("(1n)")},
		{head + `<!ATTLIST r a NOTATION #IMPLIED>` + tail, subset + notation("#IMPLIED")},
		{head + `<!ATTLIST r a (x y) #IMPLIED>` + tail, subset + enum("(x")},
		{head + `<!ATTLIST r a (x,y) #IMPLIED>` + tail, subset + enum("(x,y)")},
		{head + `<!ATTLIST r a ( ) #IMPLIED>` + tail, subset + enum("(")},
		{head + `<!ATTLIST r a (a×b) #IMPLIED>` + tail, subset + enum("(a×b)")},
		{head + `<!ATTLIST r a CDATA #implied>` + tail, subset + dflt("#implied")},
		{head + `<!ATTLIST r a CDATA #FIXED"v">` + tail, subset + dflt(`#FIXED"v"`)},
		{head + `<!ATTLIST r a CDATA #FIXED #IMPLIED>` + tail, subset + dflt("#FIXED")},
		{head + `<!ATTLIST r a CDATA #FIXED>` + tail, subset + dflt("#FIXED")},
		{head + `<!ATTLIST r a CDATA v>` + tail, subset + dflt("v")},
		{head + `<!ATTLIST r a CDATA #BOGUS"v">` + tail, subset + dflt(`#BOGUS"v"`)},
		{head + `<!ATTLIST r a CDATA v"x">` + tail, subset + dflt(`v"x"`)},
		{head + `<!ATTLIST r a (x|y)#IMPLIED>` + tail, subset + dflt("#IMPLIED")},
		{head + `<!ATTLIST r a CDATA #IMPLIED"v">` + tail, subset + attDef(`"v"`)},
		{head + `<!ATTLIST r a CDATA "v"b CDATA #IMPLIED>` + tail, subset + attDef("b")},
		{head + `<!ATTLIST r a CDATA "&b">` + tail, subset + ref},
		{head + `<!ATTLIST r a CDATA "a & b">` + tail, subset + ref},
		{head + `<!ATTLIST r a CDATA "&#0;">` + tail, subset + legal("0")},
		{head + `<!ATTLIST r a CDATA "&#xFFFE;">` + tail, subset + legal("xFFFE")},
		{head + `<!ENTITY % p "<!ELEMENT r (a,,b)>"> %p;` + tail, inPE + spec("(a,,b)", childRule)},
		{head + `<!ENTITY % p "<!ATTLIST r a CDATA>"> %p;` + tail, inPE + dflt("")},
		{head + `<!ENTITY % p "<!ATTLIST r a CDATA '<'>"> %p;` + tail, inPE + lt},
		{head + ext + `<!ELEMENT r ANYTHING>` + tail, subset + spec("ANYTHING", keyword)},
		{head + ext + `<!ATTLIST r a CDATA "<">` + tail, subset + lt},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			wantSubsetFault(t, tc.doc, tc.want)
		})
	}
}

// An entity reference in an <!ATTLIST> default value is checked at the
// declaration against the entity it names, whether or not the default is ever
// applied. Each fault row breaks one constraint, its entities declared before
// the <!ATTLIST> unless the row is about precedence: a name declared nowhere,
// or only after the <!ATTLIST>, or, in a standalone="yes" document, only in a
// parameter entity's replacement text, and one declared nowhere that another
// entity's replacement text references, even where a default value in a
// parameter entity, which the constraint does not bind, has walked that entity
// first, or where that entity, declared outside every parameter entity, is
// reached through one whose binding declaration stands in a parameter entity
// (XML 1.0 WFC: Entity Declared, which a standalone="yes" document is bound by
// after a parameter-entity reference); an unparsed entity, directly,
// after a parameter-entity reference read and in one's replacement text, and,
// in a standalone="yes" document, one declared, with the <!ATTLIST>, after a
// parameter-entity reference the reader did not read, which §5.1 has the
// reader process (#2459) (WFC: Parsed Entity); an entity reaching itself in
// one step or two (WFC: No Recursion); an external entity, directly, through
// another entity and declared only after the <!ATTLIST> (WFC: No External
// Entity References); and a '<' in replacement text, directly, through another
// entity and in a name's first, binding declaration (WFC: No < in Attribute
// Values); and a '&' in replacement text that begins no Reference, where the
// literal spelled it `&#38;` or `&#x26;`, directly, before a ';' that closes
// no Name, through another entity and where Entity Declared does not bind
// (§4.4.5, [10] AttValue, [67] Reference), and one that begins an EntityRef to
// a name declared nowhere (WFC: Entity Declared). Every fault wraps no cause.
// Each control reads: a predefined entity, a CharRef, one spelled with a
// character reference to '&', an entity whose replacement text is a CharRef to
// '<' or to '&', and one whose replacement text references a declared entity
// through a '&' its literal spelled `&#38;`; an undeclared name where Entity
// Declared does not bind — after an unread or a read parameter-entity
// reference, or before a read one, and in replacement text after a read one;
// an indirect reference to an entity declared after the <!ATTLIST>, which is
// VC: Entity Declared's; an entity first declared after a declined reference,
// which the reader does not process (§5.1); a name whose first declaration is
// clean; an entity reached twice by one walk, and the "billion laughs"
// entities, which the walk reads once each or not in this test's lifetime; and
// a clean <!ATTLIST> never applied, and that standalone="yes" row's <!ATTLIST>
// without standalone="yes", which the reader does not process (§5.1). Each
// control reads with an internal subset alone and again beside an external
// one. So do an undeclared name under an external subset, where Entity
// Declared does not bind either, and one in a parameter entity's replacement
// text in a standalone="yes" document, which it does not reach; the
// standalone="yes" row whose entity is declared only in a parameter entity,
// without standalone="yes", where the reference read lifts the constraint; a
// standalone="yes" default value referencing f, declared outside every
// parameter entity but bound (§4.2) by its first declaration, in one, whose
// replacement text references x, declared only in that parameter entity, a
// reference that occurs within a parameter entity (#2365); an entity whose
// replacement text references a name declared nowhere, which no default value
// references; and an entity whose replacement text holds a '&' that begins no
// Reference, which no default value references either.
func TestAttlistDefaultEntityReferencesAreWellFormed(t *testing.T) {
	const decl = "<?xml version=\"1.0\"?>\n"
	const alone = "<?xml version=\"1.0\" standalone=\"yes\"?>\n"
	const head = `<!DOCTYPE r [<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n>`
	const tail = `]><r ent="pic"/>`
	const ext = `<!ENTITY % ext SYSTEM "x.ent"> %ext; `
	const extPic = `<!DOCTYPE r [` + ext + `<!NOTATION n SYSTEM 'x'><!ENTITY pic SYSTEM 'u' NDATA n><!ATTLIST r a CDATA "&pic;">`
	const read = `<!ENTITY % p ""> %p; `
	const peDeclared = `<!ENTITY % p "<!ENTITY e 'x'>">%p;<!ATTLIST r a CDATA "&e;">`
	const peBound = `<!ENTITY % p "<!ENTITY f '[&x;]'><!ENTITY x 'y'>">%p;<!ENTITY f "z">`
	const peBoundViaG = `<!ENTITY % p "<!ENTITY f '&g;'><!ENTITY x 'y'>">%p;<!ENTITY f "z"><!ENTITY g "[&x;]">`
	laughs := `<!ENTITY l0 "lol">`
	for i := 1; i <= 9; i++ {
		laughs += fmt.Sprintf(`<!ENTITY l%d "%s">`, i, strings.Repeat(fmt.Sprintf("&l%d;", i-1), 10))
	}
	const value = `t.xml:2:1: [xml-wf] DOCTYPE internal subset holds an <!ATTLIST> declaration of "r" whose attribute "a" has a default value that references`
	const inPE = `t.xml:2:1: [xml-wf] replacement text of a parameter entity referenced between DOCTYPE declarations holds an <!ATTLIST> declaration of "r" whose attribute "a" has a default value that references`
	declared := func(n string) string {
		return fmt.Sprintf(" entity %s, which no general entity declaration before it, outside every parameter entity, declares (XML 1.0 WFC: Entity Declared)", n)
	}
	undeclared := func(via, n string) string {
		return fmt.Sprintf(", directly or indirectly, entity %s, whose replacement text references entity %s, which no general entity declaration outside every parameter entity declares (XML 1.0 WFC: Entity Declared)", via, n)
	}
	parsed := func(n string) string {
		return fmt.Sprintf(", directly or indirectly, the unparsed entity %s (XML 1.0 WFC: Parsed Entity)", n)
	}
	recursion := func(n string) string {
		return fmt.Sprintf(", directly or indirectly, entity %s, which references itself (XML 1.0 WFC: No Recursion)", n)
	}
	external := func(n string) string {
		return fmt.Sprintf(", directly or indirectly, the external entity %s (XML 1.0 WFC: No External Entity References)", n)
	}
	lt := func(n string) string {
		return fmt.Sprintf(", directly or indirectly, entity %s, whose replacement text holds '<' (XML 1.0 WFC: No < in Attribute Values)", n)
	}
	stray := func(n string) string {
		return fmt.Sprintf(", directly or indirectly, entity %s, whose replacement text holds a '&' that begins no Reference, '&' Name ';' or a character reference (XML 1.0 §4.4.5, [10] AttValue, [67] Reference)", n)
	}
	for _, tc := range []struct {
		doc  string
		want string // the whole error
	}{
		{decl + head + `<!ATTLIST r a CDATA "&u;">` + tail, value + declared("u")},
		{decl + head + `<!ATTLIST r a CDATA "&e;"><!ENTITY e "x">` + tail, value + declared("e")},
		{decl + head + `<!ATTLIST r a CDATA "x&amp;y&#60;&e;"><!ENTITY e "x">` + tail, value + declared("e")},
		{alone + head + ext + `<!ATTLIST r a CDATA "&u;">` + tail, value + declared("u")},
		{alone + head + read + `<!ATTLIST r a CDATA "&u;">` + tail, value + declared("u")},
		{alone + head + peDeclared + tail, value + declared("e")},
		{decl + head + `<!ENTITY f "&u;"><!ATTLIST r a CDATA "&f;">` + tail, value + undeclared("f", "u")},
		{alone + head + `<!ENTITY f "&u;"><!ENTITY % q "<!ATTLIST r b CDATA '&f;'>"> %q;<!ATTLIST r a CDATA "&f;">` + tail, value + undeclared("f", "u")},
		{alone + head + peBoundViaG + `<!ATTLIST r a CDATA "&f;">` + tail, value + undeclared("g", "x")},
		{decl + head + `<!ATTLIST r a CDATA "&pic;">` + tail, value + parsed("pic")},
		{decl + head + read + `<!ATTLIST r a CDATA "&pic;">` + tail, value + parsed("pic")},
		{decl + head + `<!ENTITY % q "<!ATTLIST r a CDATA '&pic;'>"> %q;` + tail, inPE + parsed("pic")},
		{alone + extPic + tail, value + parsed("pic")},
		{decl + head + `<!ENTITY e "&e;"><!ATTLIST r a CDATA "&e;">` + tail, value + recursion("e")},
		{decl + head + `<!ENTITY e "&f;"><!ENTITY f "&e;"><!ATTLIST r a CDATA "&e;">` + tail, value + recursion("e")},
		{decl + head + `<!ENTITY x SYSTEM "x.ent"><!ATTLIST r a CDATA "&x;">` + tail, value + external("x")},
		{decl + head + `<!ENTITY x SYSTEM "x.ent"><!ENTITY y "&x;"><!ATTLIST r a CDATA "&y;">` + tail, value + external("x")},
		{decl + head + read + `<!ATTLIST r a CDATA "&x;"><!ENTITY x SYSTEM "x.ent">` + tail, value + external("x")},
		{decl + head + `<!ENTITY lt2 "<"><!ATTLIST r a CDATA "&lt2;">` + tail, value + lt("lt2")},
		{decl + head + `<!ENTITY lt2 "<"><!ENTITY m "&lt2;"><!ATTLIST r a CDATA "&m;">` + tail, value + lt("lt2")},
		{decl + head + `<!ENTITY e "a<b"><!ENTITY e "x"><!ATTLIST r a CDATA "&e;">` + tail, value + lt("e")},
		{decl + head + `<!ENTITY f "a&#38;b"><!ATTLIST r a CDATA "&f;">` + tail, value + stray("f")},
		{decl + head + `<!ENTITY f "a&#x26;b"><!ATTLIST r a CDATA "&f;">` + tail, value + stray("f")},
		{decl + head + `<!ENTITY f "a&#38;b c;"><!ENTITY g "&f;"><!ATTLIST r a CDATA "&g;">` + tail, value + stray("f")},
		{decl + head + read + `<!ENTITY f "&#38;"><!ATTLIST r a CDATA "&f;">` + tail, value + stray("f")},
		{decl + head + `<!ENTITY f "a&#38;b;"><!ATTLIST r a CDATA "&f;">` + tail, value + undeclared("f", "b")},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			wantSubsetFault(t, tc.doc, tc.want)
		})
	}
	for _, subset := range []string{
		`<!ATTLIST r a CDATA "&amp;&lt;&gt;&apos;&quot;">`,
		`<!ATTLIST r a CDATA "&#60;&#x3C;">`,
		`<!ATTLIST r a CDATA "&#38;u;">`,
		`<!ENTITY e "&#38;#60;"><!ATTLIST r a CDATA "&e;">`,
		`<!ENTITY e "a&#38;#38;b"><!ATTLIST r a CDATA "&e;">`,
		`<!ENTITY b "v"><!ENTITY e "a&#38;b;"><!ATTLIST r a CDATA "&e;">`,
		ext + `<!ATTLIST r a CDATA "&u;">`,
		read + `<!ATTLIST r a CDATA "&u;">`,
		`<!ATTLIST r a CDATA "&u;">` + read,
		`<!ENTITY y "&x;"><!ATTLIST r a CDATA "&y;"><!ENTITY x "v">`,
		read + `<!ENTITY f "&u;"><!ATTLIST r a CDATA "&f;">`,
		ext + `<!ENTITY lt2 "<"><!ATTLIST r a CDATA "&lt2;">`,
		`<!ENTITY e "x"><!ENTITY e "a<b"><!ATTLIST r a CDATA "&e;">`,
		`<!ENTITY e "&f;&f;"><!ENTITY f "v"><!ATTLIST r a CDATA "&e;&e;" b CDATA "&f;">`,
		`<!ENTITY e "v"><!ATTLIST q a CDATA "&e;">`,
		laughs + `<!ATTLIST r a CDATA "&l9;">`,
	} {
		for _, doc := range []string{
			`<!DOCTYPE r [` + subset + `]><r/>`,
			`<!DOCTYPE r SYSTEM "r.dtd" [` + subset + `]><r/>`,
		} {
			t.Run(doc, func(t *testing.T) {
				drained(t, doc)
			})
		}
	}
	for _, doc := range []string{
		`<!DOCTYPE r SYSTEM "r.dtd" [<!ATTLIST r a CDATA "&u;">]><r/>`,
		alone + `<!DOCTYPE r [<!ENTITY % q "<!ATTLIST r a CDATA '&u;'>"> %q;]><r/>`,
		decl + head + peDeclared + tail,
		alone + head + peBound + `<!ATTLIST r a CDATA "&f;">` + tail,
		decl + head + `<!ENTITY f "&u;"><!ATTLIST r a CDATA "x">` + tail,
		decl + head + `<!ENTITY f "a&#38;b"><!ATTLIST r a CDATA "x">` + tail,
		decl + extPic + tail,
	} {
		t.Run(doc, func(t *testing.T) {
			drained(t, doc)
		})
	}
}

// <!ELEMENT> and <!ATTLIST> declarations that match their productions read,
// pic declared before or after them and every declaration processed: each
// contentspec form ([46]–[51]), a seq of one cp, nested groups, S of every
// kind wherever it is optional, NameChars in names and Nmtokens that are no
// Names ([7]), an <!ATTLIST> with no AttDef, every AttType ([54]–[59]) and
// DefaultDecl ([60]), a default value holding '>', '%', the other quote or a
// Reference ([10]), and declarations expanded from a parameter entity. A
// violation of a validity constraint alone is no fault: a second declaration
// of an element type, a name repeated in Mixed, a duplicate enumeration
// token, two ID attributes, an ID attribute's default, an undeclared notation.
func TestElementAndAttlistDeclControls(t *testing.T) {
	const notation = `<!NOTATION n SYSTEM 'x'>`
	const pic = `<!ENTITY pic SYSTEM 'u' NDATA n>`
	for _, markup := range []string{
		`<!ELEMENT r (#PCDATA|a)*>`,
		`<!ELEMENT r (#PCDATA)>`,
		`<!ELEMENT r (#PCDATA)*>`,
		`<!ELEMENT r (a,(b|c)+)?>`,
		`<!ELEMENT r EMPTY>`,
		`<!ELEMENT r ANY>`,
		`<!ATTLIST r a (x|y) "x" b CDATA #FIXED "v" c NOTATION (n) #IMPLIED>`,
		`<!ELEMENT r EMPTY><!ELEMENT r EMPTY>`,
		`<!ELEMENT r (a)>`,
		`<!ELEMENT r ((((a))))*>`,
		`<!ELEMENT r (a?,b*,c+,(d|e)*)+>`,
		`<!ELEMENT r (a|b|c)>`,
		`<!ELEMENT r (x:a.b-c_d·9)>`,
		`<!ELEMENT r ( #PCDATA | a | b )* >`,
		"<!ELEMENT\tr\n(\r\na\t,\n( b | c )\r\n)\n>",
		`<!ELEMENT r (#PCDATA|a|a)*>`,
		`<!ATTLIST r>`,
		"<!ATTLIST r \t>",
		`<!ATTLIST r a CDATA "a>b" b CDATA "%x;" c CDATA '"' d CDATA "'" e CDATA "">`,
		`<!ATTLIST r a CDATA "&amp;&#60;&#x3E;&#x10FFFD;">`,
		`<!ENTITY e "v"><!ATTLIST r a CDATA "&e;">`,
		`<!ATTLIST r a (x|x) #IMPLIED>`,
		`<!ATTLIST r a ID #IMPLIED b ID #REQUIRED>`,
		`<!ATTLIST r a ID "x">`,
		`<!ATTLIST r a NOTATION (undeclared) #IMPLIED>`,
		`<!ATTLIST r a (1|-x|.5|x:y) #IMPLIED>`,
		`<!ATTLIST r a IDREF #IMPLIED b IDREFS #IMPLIED c ENTITY #IMPLIED d ENTITIES #IMPLIED e NMTOKEN #IMPLIED f NMTOKENS #REQUIRED>`,
		"<!ATTLIST\tr\n\ta\r\nCDATA\t#FIXED\n'v'\n>",
		`<!ATTLIST r a NOTATION ( n | m ) #IMPLIED b ( x | y ) 'y' >`,
		`<!ENTITY % d "<!ELEMENT r (a,b)><!ATTLIST r a CDATA #IMPLIED>"> %d;`,
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

// The W3C suite documents whose internal subset carries <!ELEMENT> or
// <!ATTLIST> declarations — test65699.inc's inside a comment — and the
// saxonData/Id id017–id021 instances, which declare an unparsed entity, read
// to EOF: a fault on any one is a false reject. Skipped without the suite
// submodule.
func TestSuiteInternalSubsetsRead(t *testing.T) {
	const suite = "../../testdata/xsdtests"
	if _, err := os.Stat(filepath.Join(suite, "suite.xml")); err != nil {
		t.Skip("suite submodule absent: git submodule update --init testdata/xsdtests")
	}
	paths := []string{
		filepath.Join(suite, "msData/additional/test65699.inc"),
		filepath.Join(suite, "wgData/iri/TypeLibrary-IRI-RFC3987.xsd"),
		filepath.Join(suite, "wgData/iri/TypeLibrary-URI-RFC3986.xsd"),
		filepath.Join(suite, "common/xsd.xsl"),
		filepath.Join(suite, "wgMeta/ancillary/xsts.xsl"),
	}
	for _, glob := range []string{"id01[7-9].*.xml", "id02[01].*.xml"} {
		ids, err := filepath.Glob(filepath.Join(suite, "saxonData/Id", glob))
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, ids...)
	}
	if len(paths) != 5+21 {
		t.Fatalf("found %d documents, want 26: 5 internal-subset carriers and 21 id017–id021 instances", len(paths))
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			src, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			drained(t, string(src))
		})
	}
}
