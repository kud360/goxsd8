package xmltree_test

import "testing"

// TestAttributeValueIsNormalized reads every attribute value as its normalized
// value per XML 1.0 §3.3.3, the attribute read as CDATA: each literal #x9, #xA
// or #xD is #x20, a CR LF line end one #x20 (§2.11), while a character
// reference stays the character it names. The literal rows fail when a value
// that references no entity keeps the decoder's value, which applies §2.11 but
// not §3.3.3's step 3; the character-reference rows fail when the decoder's
// value is normalized instead of the source.
func TestAttributeValueIsNormalized(t *testing.T) {
	for _, tc := range []struct {
		name, doc, want string
	}{
		{"character reference to a tab", `<r a="1&#9;2"/>`, `<r a="1\t2"></r>`},
		{"character references to CR and LF", `<r a="1&#13;&#10;2"/>`, `<r a="1\r\n2"></r>`},
		{"literal tab, LF and CR LF", "<r a=\"1\t2\n3\r\n4\"/>", `<r a="1 2 3 4"></r>`},
		{"lone literal CR, single-quoted", "<r a='1\r2'/>", `<r a="1 2"></r>`},
		{"no collapse, no trim", "<r a=\"\t 1\n\n2 \"/>", `<r a="  1  2 "></r>`},
		{"literal next to a character reference", "<r a=\"\t&#9;\"/>", `<r a=" \t"></r>`},
		{"namespace declaration", "<p:r xmlns:p=\"urn:\tp\"/>", `<{urn: p}r></r>`},
		{"beside an entity-referencing attribute", "<!DOCTYPE r [<!ENTITY ent \"e\tf\">]><r a=\"x\ny\" b=\"&ent;\"/>",
			`<r a="x y" b="e f"></r>`},
	} {
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
