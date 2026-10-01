package xmlenc

import (
	"io"
	"strings"
	"testing"
	"unicode/utf16"
)

// TestDecodeTranscodesEachMark pins Decode's per-mark contract (XML 1.0
// §4.3.3, Appendix F.1): each mark is consumed, a UTF-16 body is transcoded to
// UTF-8, and the Mark names what was found. parser/xmltree's reader tests
// cover streaming, ill-formed input and source failures through NewReader.
func TestDecodeTranscodesEachMark(t *testing.T) {
	const doc = "<a>clef \U0001D11E</a>"
	for _, tc := range []struct {
		name string
		in   string
		want Mark
	}{
		{"no mark", doc, markNone},
		{"UTF-8 mark", "\xEF\xBB\xBF" + doc, markUTF8},
		{"UTF-16BE mark", utf16Bytes(doc, true), markUTF16BE},
		{"UTF-16LE mark", utf16Bytes(doc, false), markUTF16LE},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, m := Decode(strings.NewReader(tc.in))
			got, err := io.ReadAll(body)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if m != tc.want {
				t.Errorf("mark = %v, want %v", m, tc.want)
			}
			if string(got) != doc {
				t.Errorf("body = %q, want %q", got, doc)
			}
		})
	}
}

// TestCharsetReaderAgreement pins §4.3.3's fatal error as CharsetReader
// reports it: a name agreeing with the mark passes the stream through, and a
// disagreeing one fails.
func TestCharsetReaderAgreement(t *testing.T) {
	for _, tc := range []struct {
		m     Mark
		name  string
		agree bool
	}{
		{markUTF16LE, "utf-16", true},
		{markUTF16BE, "UTF-16", true},
		{markUTF16BE, "UTF-16BE", true},
		{markUTF16LE, "UTF-16BE", false},
		{markNone, "UTF-16", false},
		{markUTF16LE, "ISO-8859-1", false},
	} {
		in := strings.NewReader("x")
		got, err := tc.m.CharsetReader(tc.name, in)
		if (err == nil) != tc.agree {
			t.Errorf("%v.CharsetReader(%q) err = %v, want agreement %v", tc.m, tc.name, err, tc.agree)
		}
		if tc.agree && got != io.Reader(in) {
			t.Errorf("%v.CharsetReader(%q) wrapped the stream, want it passed through", tc.m, tc.name)
		}
	}
}

// utf16Bytes encodes doc as UTF-16 in the given byte order behind its mark.
func utf16Bytes(doc string, bigEnd bool) string {
	var b strings.Builder
	for _, u := range utf16.Encode(append([]rune{0xFEFF}, []rune(doc)...)) {
		hi, lo := byte(u>>8), byte(u)
		if !bigEnd {
			hi, lo = lo, hi
		}
		b.WriteByte(hi)
		b.WriteByte(lo)
	}
	return b.String()
}
