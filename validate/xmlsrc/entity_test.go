package xmlsrc

import (
	"testing"
)

// entityDoc declares one unparsed entity, pic, in its internal subset, and one
// parsed entity, txt, that is no member of [unparsedEntities].
const entityDoc = `<?xml version="1.0"?>
<!DOCTYPE r [
  <!NOTATION gif SYSTEM "image/gif">
  <!ENTITY pic SYSTEM "pic.gif" NDATA gif>
  <!ENTITY txt "parsed">
]>
<r><c/></r>
`

// Every element of one document answers from its DOCTYPE, the document element
// and a child alike.
func TestElementsAnswerTheDocumentsUnparsedEntities(t *testing.T) {
	root := rootOf(t, entityDoc)
	child, ok := root.Children().Next()
	if !ok {
		t.Fatal("the document element yielded no child")
	}
	c, ok := child.Element()
	if !ok {
		t.Fatal("the first child is not an element")
	}
	for _, e := range []*element{root, c.(*element)} {
		if !e.HasUnparsedEntity("pic") {
			t.Errorf("%s: HasUnparsedEntity(%q) = false, want true", e.Name(), "pic")
		}
		if e.HasUnparsedEntity("txt") {
			t.Errorf("%s: HasUnparsedEntity(%q) = true, want false: txt is a parsed entity", e.Name(), "txt")
		}
	}
}
