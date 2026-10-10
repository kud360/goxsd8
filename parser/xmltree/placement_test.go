package xmltree_test

import "testing"

// A processing instruction targeting "xml" and a directive must stand where
// XML 1.0's grammar puts them, or the document is not well-formed: [17]
// PITarget excludes "xml" in every case, which only the [23] XMLDecl opening
// [22] prolog spells, in lower case at the document's first character, after
// no white space, comment or other declaration; [43] content admits no
// directive; [22] prolog admits one doctypedecl, before the document element,
// and [27] Misc after it none. Replacement text included in content must match
// [43] content (§4.3.2), so the same PI or directive there is a fault located
// at the reference. Each is a fault at the offending token that declares
// nothing: a DOCTYPE after the document element or after another declares no
// pic, and a misplaced standalone="yes" declaration is charged before it can
// put the subset after an unread parameter entity under §5.1's standalone
// reading, which would declare pic.
func TestMisplacedDeclarationIsAFault(t *testing.T) {
	const (
		pic         = `<!ENTITY pic SYSTEM "p" NDATA gif>`
		xmlTarget   = `[xml-wf] processing instruction target "xml" after the document's first character, which XML 1.0 [17] PITarget excludes everywhere but in the [23] XMLDecl that opens [22] prolog`
		noDirective = ` XML 1.0 [43] content admits no directive`
		afterRoot   = `[xml-wf] DOCTYPE after the document element, where XML 1.0 [27] Misc admits none`
		second      = `[xml-wf] second DOCTYPE, where XML 1.0 [22] prolog admits only one`
		otherCase   = `" is "xml" in another case, which XML 1.0 [17] PITarget excludes`
		inEntity    = ` in the replacement text of entity e, `
		entityHead  = "<!DOCTYPE r [<!ENTITY e \""
		entityTail  = "\">]>\n<r>&e;</r>"
	)
	for _, tc := range []struct {
		doc, want string
	}{
		{`<r><?xml version="1.0"?></r>`, `t.xml:1:4: ` + xmlTarget},
		{`<r><?XmL x?></r>`, `t.xml:1:4: [xml-wf] processing instruction target "XmL` + otherCase},
		{`<r/><?xml version="1.0"?>`, `t.xml:1:5: ` + xmlTarget},
		{` <?xml version="1.0"?><r/>`, `t.xml:1:2: ` + xmlTarget},
		{`<!--c--><?xml version="1.0"?><r/>`, `t.xml:1:9: ` + xmlTarget},
		{`<?xml version="1.0"?><?xml version="1.0"?><r/>`, `t.xml:1:22: ` + xmlTarget},
		{`<?XML version="1.0"?><r/>`, `t.xml:1:1: [xml-wf] processing instruction target "XML` + otherCase},
		{`<!--c--><?xml version="1.0" standalone="yes"?><!DOCTYPE r [<!ENTITY % p SYSTEM "p.ent">%p;` + pic + `]><r/>`, `t.xml:1:9: ` + xmlTarget},
		{`<r><!DOCTYPE r></r>`, `t.xml:1:4: [xml-wf] directive "<!DOCTYPE" inside element r, where` + noDirective},
		{`<r><!DOCTYPE r [` + pic + `]></r>`, `t.xml:1:4: [xml-wf] directive "<!DOCTYPE" inside element r, where` + noDirective},
		{`<r><!FOO></r>`, `t.xml:1:4: [xml-wf] directive "<!FOO" inside element r, where` + noDirective},
		{`<r xmlns="u"><a><!FOO></a></r>`, `t.xml:1:17: [xml-wf] directive "<!FOO" inside element {u}a, where` + noDirective},
		{`<r/><!DOCTYPE r>`, `t.xml:1:5: ` + afterRoot},
		{`<r/><!DOCTYPE r [` + pic + `]>`, `t.xml:1:5: ` + afterRoot},
		{`<!DOCTYPE a><!DOCTYPE b><r/>`, `t.xml:1:13: ` + second},
		{`<!DOCTYPE a><!DOCTYPE b [` + pic + `]><r/>`, `t.xml:1:13: ` + second},
		{entityHead + `<?xml version='1.0'?>` + entityTail, `t.xml:2:4: [xml-wf] processing instruction target "xml"` + inEntity + `a name XML 1.0 [17] PITarget excludes`},
		{entityHead + `<?XmL x?>` + entityTail, `t.xml:2:4: [xml-wf] processing instruction target "XmL"` + inEntity + `a name XML 1.0 [17] PITarget excludes`},
		{entityHead + `<!DOCTYPE r>` + entityTail, `t.xml:2:4: [xml-wf] directive "<!DOCTYPE"` + inEntity + `where` + noDirective},
		{entityHead + `<!FOO>` + entityTail, `t.xml:2:4: [xml-wf] directive "<!FOO"` + inEntity + `where` + noDirective},
	} {
		t.Run(tc.doc, func(t *testing.T) {
			wantSubsetFault(t, tc.doc, tc.want)
		})
	}
}

// An XML declaration first, after an encoding signature too, a processing
// instruction whose target only begins "xml", in content and in included
// replacement text, and an entity whose replacement text holds an "xml" PI or
// a directive but is never referenced (§4.3.2's Note) all read.
func TestPlacedDeclarationReads(t *testing.T) {
	for _, doc := range []string{
		`<?xml version="1.0"?><r/>`,
		"\xEF\xBB\xBF<?xml version=\"1.0\"?><r/>",
		`<r><?xml-stylesheet href="s"?><?xmlfoo x?></r>`,
		`<!DOCTYPE r [<!ENTITY e "<?xml-stylesheet x?><!--c-->">]><r>&e;</r>`,
		`<!DOCTYPE r [<!ENTITY e "<?xml version='1.0'?><!DOCTYPE r>">]><r/>`,
	} {
		t.Run(doc, func(t *testing.T) {
			drained(t, doc)
		})
	}
}
