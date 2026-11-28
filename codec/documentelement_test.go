package codec

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestDocumentElementName pins DocumentElementName's doc, one row per shape:
// the namespace of a default-namespace and of a prefixed document element,
// the comment, processing instruction and white space before it, a document
// ill-formed only after its start tag, the Reader's own error returned
// unchanged for an unbound prefix, and the xml-wf error, never io.EOF, for a
// document with no element.
func TestDocumentElementName(t *testing.T) {
	const uri = "i.xml"
	for _, row := range []struct {
		data string
		want xsd.QName
	}{
		{`<r xmlns="urn:s"/>`, xsd.QName{Space: "urn:s", Local: "r"}},
		{`<p:n xmlns:p="urn:s"/>`, xsd.QName{Space: "urn:s", Local: "n"}},
		{"<!-- c -->\n<?p x?>\n<r xmlns=\"urn:s\"/>", xsd.QName{Space: "urn:s", Local: "r"}},
		{`<r xmlns="urn:s"><a></r>`, xsd.QName{Space: "urn:s", Local: "r"}},
	} {
		got, err := DocumentElementName([]byte(row.data), WithURI(uri))
		if err != nil || got != row.want {
			t.Errorf("DocumentElementName(%q) = %v, %v; want %v, nil", row.data, got, err, row.want)
		}
	}
	const rootless = "i.xml: [xml-wf] document has no document element"
	for _, row := range []struct {
		data, prefix string
		at           xsderr.Loc
	}{
		{`<q:r xmlns:p="urn:s"/>`, "i.xml:1:1: [xml-wf] ", xsderr.Loc{URI: uri, Line: 1, Col: 1}},
		{"", rootless, xsderr.Loc{URI: uri}},
		{"<!-- only -->", rootless, xsderr.Loc{URI: uri}},
	} {
		got, err := DocumentElementName([]byte(row.data), WithURI(uri))
		var xe *xsderr.Error
		if got != (xsd.QName{}) || !errors.As(err, &xe) || xe.Rule != xsderr.RuleXMLWellFormed || xe.Loc != row.at ||
			errors.Is(err, io.EOF) || !strings.HasPrefix(err.Error(), row.prefix) {
			t.Errorf("DocumentElementName(%q) = %v, %v; want the zero QName and an xml-wf *xsderr.Error at %v opening %q, not io.EOF",
				row.data, got, err, row.at, row.prefix)
		}
	}
}
