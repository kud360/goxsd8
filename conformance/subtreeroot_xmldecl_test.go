package conformance

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/parser/xmltree"
)

// TestRawReadsAdmitTheLabelXmltreeAdmits pins that both raw re-reads,
// assessedSubtreeRoot and documentVersioned, admit a UTF-8 document's 1.x
// version label exactly when parser/xmltree admits it, a byte-order mark before
// the label included (XML 1.0 §2.8 Note, §4.3.3). The marked 1.1 and 1.10 rows
// fail when rawDecoder hands As10 the mark: As10 then meets no '<?xml', and
// encoding/xml stops on the unrewritten label with `unsupported version`. The
// doubled-mark 1.1 row fails if the mark is read past more than once: xmltree
// drops one mark and refuses that document.
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
		{mark + mark + `<?xml version="1.0"?>`, true},
		{mark + mark + `<?xml version="1.1"?>`, false},
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
		if got := assessedSubtreeRoot(schema, report, c.doc); got != tc.admit {
			t.Errorf("%q: assessedSubtreeRoot = %v, want %v, as xmltree admits", tc.prolog, got, tc.admit)
		}
		if got := documentVersioned(c.doc); got == tc.admit {
			t.Errorf("%q: documentVersioned = %v, want %v, as xmltree admits", tc.prolog, got, !tc.admit)
		}
	}
}

// TestRawReadsRefuseUTF16 pins rawDecoder's doc: a UTF-16 document, which
// xmltree reads, is read by neither raw re-read, under any label, and each
// answers as for an unreadable document.
func TestRawReadsRefuseUTF16(t *testing.T) {
	doc := `<?xml version="1.0" encoding="UTF-16"?><known>x</known>`
	units := utf16.Encode([]rune(doc))
	raw := []byte{0xFF, 0xFE}
	for _, u := range units {
		raw = append(raw, byte(u), byte(u>>8))
	}
	if err := readAll(string(raw)); err != nil {
		t.Fatalf("xmltree rejects the UTF-16 row: %v", err)
	}
	c := instanceCase(t, knownRoot, "", true)
	if err := os.WriteFile(c.doc, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	schema, report, decidable, err := assembleCase(strict.New(), c.schemaDoc, nil)
	if err != nil || !decidable {
		t.Fatalf("assembling the schema: decidable %v, err %v", decidable, err)
	}
	if assessedSubtreeRoot(schema, report, c.doc) {
		t.Errorf("assessedSubtreeRoot(%s) = true, want false for a UTF-16 document", filepath.Base(c.doc))
	}
	if !documentVersioned(c.doc) {
		t.Errorf("documentVersioned(%s) = false, want true for a UTF-16 document", filepath.Base(c.doc))
	}
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
// reports no more data: dropped, the decoder's re-read would see a bare EOF,
// which documentVersioned reads as a document with no versioning attribute.
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
