package conformance

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/parser/xmltree"
)

// TestRawReadsAdmitTheLabelXmltreeAdmits pins that the harness's re-reads,
// assessedSubtreeRoot and instanceHints through rawDecoder and documentCarries
// through parser/xmltree itself, admit a UTF-8 document's 1.x version label
// exactly when parser/xmltree admits it, a byte-order mark before the label
// included (XML 1.0 §2.8 Note, §4.3.3). The marked 1.1 and 1.10 rows fail when
// rawDecoder hands As10 the mark: As10 then meets no '<?xml', and encoding/xml
// stops on the unrewritten label with `unsupported version`. The doubled-mark
// rows fail if the mark is read past more than once: xmltree drops one mark
// and refuses the second as character data before the document element that
// is no S ([22] prolog, [27] Misc), which rawDecoder's readers refuse in
// rootStart.
func TestRawReadsAdmitTheLabelXmltreeAdmits(t *testing.T) {
	const mark = "\xEF\xBB\xBF"
	const body = `<known>x</known>`
	for _, tc := range []struct {
		prolog string
		admit  bool
	}{
		{`<?xml version="1.0"?>`, true},
		{`<?xml version="1.1"?>`, true},
		{`<?xml version='1.10' standalone='no'?>`, true},
		{mark + `<?xml version="1.0"?>`, true},
		{mark + `<?xml version="1.1"?>`, true},
		{mark + `<?xml version='1.10' standalone='no'?>`, true},
		{mark + mark + `<?xml version="1.0"?>`, false},
		{mark + mark + `<?xml version="1.1"?>`, false},
		{mark + mark, false},
		{`<?xml version="2.1"?>`, false},
	} {
		doc := tc.prolog + body
		if err := readAll(doc); (err == nil) != tc.admit {
			t.Fatalf("%q: xmltree fault %v, want admitted %v: the row misstates what the harness is held to", tc.prolog, err, tc.admit)
		}
		c := instanceCase(t, knownRoot, doc, true)
		schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
		if err != nil || !decidable {
			t.Fatalf("%q: assembling the schema: decidable %v, err %v", tc.prolog, decidable, err)
		}
		if got := assessedSubtreeRoot(strict.New(), schema, report, c.doc); (got == "") != tc.admit {
			t.Errorf("%q: assessedSubtreeRoot refused as %q, want admitted %v, as xmltree admits", tc.prolog, got, tc.admit)
		}
		if got := documentCarries(c.doc, isVersioningAttr); got == tc.admit {
			t.Errorf("%q: documentCarries(isVersioningAttr) = %v, want %v, as xmltree admits", tc.prolog, got, !tc.admit)
		}
		if _, _, _, got := instanceHints(c.doc); (got == "") != tc.admit {
			t.Errorf("%q: instanceHints refused as %q, want admitted %v, as xmltree admits", tc.prolog, got, tc.admit)
		}
	}
}

// TestRawReadsDecodeUTF16 pins that the harness's re-reads, assessedSubtreeRoot
// and instanceHints through rawDecoder and documentCarries through
// parser/xmltree itself, read a byte-order-marked UTF-16 document, in either
// byte order, exactly as parser/xmltree reads it (XML 1.0 §4.3.3,
// Appendix F.1). The 1.1 rows pin that the transcode happens before As10 meets
// the label, so the label is admitted as xmltree admits it; XML 1.0 rules
// nothing on UTF-16 by version, so the rows differ in the label alone. Every
// row's rawDecoder assertions fail when rawDecoder reads the bytes without
// xmlenc.Decode: encoding/xml reads the mark as invalid UTF-8.
func TestRawReadsDecodeUTF16(t *testing.T) {
	for _, version := range []string{"1.0", "1.1"} {
		for _, bigEnd := range []bool{false, true} {
			doc := `<?xml version="` + version + `" encoding="UTF-16"?><known>x</known>`
			raw := utf16Marked(doc, bigEnd)
			row := fmt.Sprintf("version %s, big-endian %v", version, bigEnd)
			if err := readAll(string(raw)); err != nil {
				t.Fatalf("%s: xmltree rejects the row: %v", row, err)
			}
			c := instanceCase(t, knownRoot, "", true)
			if err := os.WriteFile(c.doc, raw, 0o600); err != nil {
				t.Fatal(err)
			}
			schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
			if err != nil || !decidable {
				t.Fatalf("%s: assembling the schema: decidable %v, err %v", row, decidable, err)
			}
			if why := assessedSubtreeRoot(strict.New(), schema, report, c.doc); why != "" {
				t.Errorf("%s: assessedSubtreeRoot(%s) refused as %q, want admitted, as xmltree reads it", row, filepath.Base(c.doc), why)
			}
			if documentCarries(c.doc, isVersioningAttr) {
				t.Errorf("%s: documentCarries(%s, isVersioningAttr) = true, want false, as xmltree reads it", row, filepath.Base(c.doc))
			}
			if _, _, _, why := instanceHints(c.doc); why != "" {
				t.Errorf("%s: instanceHints(%s) refused as %q, want admitted, as xmltree reads it", row, filepath.Base(c.doc), why)
			}
		}
	}
}

// utf16Marked encodes doc as UTF-16 in the given byte order behind its mark.
func utf16Marked(doc string, bigEnd bool) []byte {
	var raw []byte
	for _, u := range utf16.Encode(append([]rune{0xFEFF}, []rune(doc)...)) {
		hi, lo := byte(u>>8), byte(u)
		if !bigEnd {
			hi, lo = lo, hi
		}
		raw = append(raw, hi, lo)
	}
	return raw
}

// readAll reads doc through parser/xmltree to its end and returns the fault
// that stopped it, if any.
func readAll(doc string) error {
	r := xmltree.NewReader("t.xml", strings.NewReader(doc))
	for {
		_, err := r.Token()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// TestRawDecoderReportsPeekFailure pins that a read failure met while peeking
// for the mark reaches the decoder, from a source that fails once and then
// reports no more data: dropped, the decoder's re-read would see a bare EOF in
// place of the failure that ended the document.
func TestRawDecoderReportsPeekFailure(t *testing.T) {
	boom := errors.New("boom")
	_, err := rawDecoder(&failOnce{err: boom}).Token()
	if !errors.Is(err, boom) {
		t.Errorf("first Token err = %v, want %v", err, boom)
	}
}

// failOnce reports err on its first read and io.EOF on every later one.
type failOnce struct {
	err  error
	done bool
}

func (f *failOnce) Read([]byte) (int, error) {
	if f.done {
		return 0, io.EOF
	}
	f.done = true
	return 0, f.err
}

// TestDocumentCarriesReadsIncludedMarkup pins that documentCarries reads a
// document referencing internal general entities as parser/xmltree includes
// them (XML 1.0 §4.4.2): a vc: attribute only an entity's replacement text
// carries is found, and a document whose entities carry none answers false
// rather than failing closed. The second row fails when documentCarries reads
// through rawDecoder, which refuses every such reference and so answers true;
// the first when that decoder is made to read on past them (Strict false),
// keeping each reference as text and the markup in it unread.
func TestDocumentCarriesReadsIncludedMarkup(t *testing.T) {
	const vc = `xmlns:vc="` + versioningNS + `"`
	for _, tc := range []struct {
		name, doc string
		want      bool
	}{
		{"vc: attribute only replacement text carries",
			`<!DOCTYPE r [<!ENTITY e "<a vc:minVersion='1.1'/>">]><r ` + vc + `>&e;</r>`, true},
		{"entities carrying no vc: attribute",
			`<!DOCTYPE r [<!ENTITY e "<a b='&d;'/>"><!ENTITY d "x">]><r ` + vc + ` v="&d;">&e;</r>`, false},
	} {
		path := filepath.Join(t.TempDir(), "doc.xsd")
		if err := os.WriteFile(path, []byte(tc.doc), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := readAll(tc.doc); err != nil {
			t.Fatalf("%s: xmltree rejects the row: %v", tc.name, err)
		}
		if got := documentCarries(path, isVersioningAttr); got != tc.want {
			t.Errorf("%s: documentCarries = %v, want %v", tc.name, got, tc.want)
		}
	}
}
