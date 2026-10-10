package xmltree_test

import "testing"

// readRow is one document and its expected render.
type readRow struct {
	name, doc, want string
}

// readsAs reads each row's document and checks it renders as the row wants.
func readsAs(t *testing.T, rows []readRow) {
	t.Helper()
	for _, tc := range rows {
		t.Run(tc.name, func(t *testing.T) {
			nodes, err := collect(t, "doc.xml", tc.doc)
			if err != nil {
				t.Fatalf("Token: %v", err)
			}
			if got := render(nodes); got != tc.want {
				t.Errorf("read %s\n want %s", got, tc.want)
			}
		})
	}
}

// TestAttributeValueIsNormalized reads every attribute value as its normalized
// value per XML 1.0 §3.3.3, steps 1–3, for an attribute read as CDATA — one the
// internal subset defines as CDATA or does not define: each literal #x9, #xA or
// #xD is #x20, a CR LF line end one #x20 (§2.11), while a character reference
// stays the character it names. The literal rows fail when a value that
// references no entity keeps the decoder's value, which applies §2.11 but not
// §3.3.3's step 3; the character-reference rows fail when the decoder's value
// is normalized instead of the source. TestDeclaredTypeCollapsesSpace reads an
// attribute the subset defines with another type.
func TestAttributeValueIsNormalized(t *testing.T) {
	readsAs(t, []readRow{
		{"character reference to a tab", `<r a="1&#9;2"/>`, `<r a="1\t2"></r>`},
		{"character references to CR and LF", `<r a="1&#13;&#10;2"/>`, `<r a="1\r\n2"></r>`},
		{"literal tab, LF and CR LF", "<r a=\"1\t2\n3\r\n4\"/>", `<r a="1 2 3 4"></r>`},
		{"lone literal CR, single-quoted", "<r a='1\r2'/>", `<r a="1 2"></r>`},
		{"no collapse, no trim", "<r a=\"\t 1\n\n2 \"/>", `<r a="  1  2 "></r>`},
		{"literal next to a character reference", "<r a=\"\t&#9;\"/>", `<r a=" \t"></r>`},
		{"namespace declaration", "<p:r xmlns:p=\"urn:\tp\"/>", `<{urn: p}r></r>`},
		{"beside an entity-referencing attribute", "<!DOCTYPE r [<!ENTITY ent \"e\tf\">]><r a=\"x\ny\" b=\"&ent;\"/>",
			`<r a="x y" b="e f"></r>`},
	})
}

// TestDeclaredTypeCollapsesSpace reads an attribute the internal subset defines
// on its element type with an AttType other than CDATA — tokenized, NOTATION or
// an enumeration (XML 1.0 [54]–[59]) — with its normalized value trimmed of
// #x20 and each run of #x20 collapsed to one: §3.3.3's paragraph after step 3,
// applied after steps 1–3, so a literal tab collapses and a character reference
// to one survives. The first definition of an attribute binds (§3.3), keyed on
// the element type and attribute names as source spells them, and a definition
// after a parameter-entity reference the reader does not read — the external
// %p; here — is not processed outside a standalone="yes" document (§5.1).
func TestDeclaredTypeCollapsesSpace(t *testing.T) {
	const alone = `<?xml version="1.0" standalone="yes"?>`
	const unread = `<!ENTITY % p SYSTEM "p.ent"> %p;`
	readsAs(t, []readRow{
		{"NMTOKENS", `<!DOCTYPE r [<!ATTLIST r a NMTOKENS #IMPLIED>]><r a="  x   y  "/>`, `<r a="x y"></r>`},
		{"ID", `<!DOCTYPE r [<!ATTLIST r a ID #IMPLIED>]><r a="  x  "/>`, `<r a="x"></r>`},
		{"enumeration", `<!DOCTYPE r [<!ATTLIST r a (x|y) #IMPLIED>]><r a="  x  "/>`, `<r a="x"></r>`},
		{"NOTATION", `<!DOCTYPE r [<!NOTATION n SYSTEM "n"><!ATTLIST r a NOTATION (n) #IMPLIED>]><r a=" n "/>`,
			`<r a="n"></r>`},
		{"trims to nothing", `<!DOCTYPE r [<!ATTLIST r a NMTOKENS #IMPLIED>]><r a="   "/>`, `<r a=""></r>`},
		{"CDATA keeps its space", `<!DOCTYPE r [<!ATTLIST r a CDATA #IMPLIED>]><r a="  x   y  "/>`,
			`<r a="  x   y  "></r>`},
		{"undeclared keeps its space", `<!DOCTYPE r [<!ATTLIST r b NMTOKENS #IMPLIED>]><r a="  x   y  " b=" z "/>`,
			`<r a="  x   y  " b="z"></r>`},
		{"literal tabs collapse", "<!DOCTYPE r [<!ATTLIST r a NMTOKENS #IMPLIED>]><r a=\"x\t\ty\"/>", `<r a="x y"></r>`},
		{"character reference to a tab survives", `<!DOCTYPE r [<!ATTLIST r a NMTOKENS #IMPLIED>]><r a="x&#9;y"/>`,
			`<r a="x\ty"></r>`},
		{"character references to a space collapse",
			`<!DOCTYPE r [<!ATTLIST r a NMTOKENS #IMPLIED>]><r a="&#32;x&#32;&#32;y&#32;"/>`, `<r a="x y"></r>`},
		{"after an unread parameter entity",
			`<!DOCTYPE r [` + unread + `<!ATTLIST r a NMTOKENS #IMPLIED>]><r a="  x   y  "/>`, `<r a="  x   y  "></r>`},
		{"after an unread parameter entity, standalone",
			alone + `<!DOCTYPE r [` + unread + `<!ATTLIST r a NMTOKENS #IMPLIED>]><r a="  x   y  "/>`, `<r a="x y"></r>`},
		{"before an unread parameter entity",
			`<!DOCTYPE r [<!ATTLIST r a NMTOKENS #IMPLIED>` + unread + `]><r a="  x   y  "/>`, `<r a="x y"></r>`},
		{"in a parameter entity read",
			`<!DOCTYPE r [<!ENTITY % d "<!ATTLIST r a NMTOKENS #IMPLIED>"> %d;]><r a="  x   y  "/>`, `<r a="x y"></r>`},
		{"first declaration binds, CDATA first",
			`<!DOCTYPE r [<!ATTLIST r a CDATA #IMPLIED><!ATTLIST r a NMTOKENS #IMPLIED>]><r a=" x "/>`, `<r a=" x "></r>`},
		{"first declaration binds, NMTOKENS first",
			`<!DOCTYPE r [<!ATTLIST r a NMTOKENS #IMPLIED><!ATTLIST r a CDATA #IMPLIED>]><r a=" x "/>`, `<r a="x"></r>`},
		{"first definition binds within one declaration",
			`<!DOCTYPE r [<!ATTLIST r a CDATA #IMPLIED a NMTOKENS #IMPLIED>]><r a=" x "/>`, `<r a=" x "></r>`},
		{"scoped to its element type",
			`<!DOCTYPE r [<!ATTLIST s a NMTOKENS #IMPLIED>]><r a=" x "><s a=" x "/></r>`, `<r a=" x "><s a="x"></s></r>`},
		{"keyed on the names as source spells them",
			`<!DOCTYPE p:r [<!ATTLIST p:r p:a NMTOKENS #IMPLIED>]><p:r xmlns:p="urn:p" p:a=" x "><q:r xmlns:q="urn:p" q:a=" x "/></p:r>`,
			`<{urn:p}r {urn:p}a="x"><{urn:p}r {urn:p}a=" x "></r></r>`},
		{"namespace declaration binds the collapsed value",
			`<!DOCTYPE p:r [<!ATTLIST p:r xmlns:p NMTOKEN #IMPLIED>]><p:r xmlns:p=" urn:p "/>`, `<{urn:p}r></r>`},
		{"element in an entity's replacement text",
			`<!DOCTYPE r [<!ENTITY e "<s a=' x '/>"><!ATTLIST s a NMTOKEN #IMPLIED>]><r>&e;</r>`, `<r><s a="x"></s></r>`},
	})
}
