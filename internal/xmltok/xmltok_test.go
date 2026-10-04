package xmltok

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"
)

// TestFifthEditionNames pins the rows encoding/xml rejects and this package
// reads: names whose characters are XML 1.0 5e [4]/[4a] but outside 4e's
// Appendix B tables (U+0133 is in 5e's [#xF8-#x2FF]). Each row first checks
// that encoding/xml still rejects it, so the row cannot pass on a decoder
// that merely re-exports encoding/xml.
func TestFifthEditionNames(t *testing.T) {
	tests := []struct {
		doc  string
		want []xml.Token
	}{
		{
			doc: `<Dĳkstra vrĳtag="0"/>`,
			want: []xml.Token{
				xml.StartElement{
					Name: xml.Name{Local: "Dĳkstra"},
					Attr: []xml.Attr{{Name: xml.Name{Local: "vrĳtag"}, Value: "0"}},
				},
				xml.EndElement{Name: xml.Name{Local: "Dĳkstra"}},
			},
		},
		{
			doc: `<ĳ:a xmlns:ĳ="urn:x"><?ĳ data?></ĳ:a>`,
			want: []xml.Token{
				xml.StartElement{
					Name: xml.Name{Space: "urn:x", Local: "a"},
					Attr: []xml.Attr{{Name: xml.Name{Space: "xmlns", Local: "ĳ"}, Value: "urn:x"}},
				},
				xml.ProcInst{Target: "ĳ", Inst: []byte("data")},
				xml.EndElement{Name: xml.Name{Space: "urn:x", Local: "a"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.doc, func(t *testing.T) {
			if _, err := drain(xml.NewDecoder(strings.NewReader(tt.doc))); err == nil {
				t.Fatalf("encoding/xml read %q: the row no longer tests the 5e tables", tt.doc)
			}
			got, err := drain(NewDecoder(strings.NewReader(tt.doc)))
			if err != nil {
				t.Fatalf("Token on %q: %v", tt.doc, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Token on %q = %#v, want %#v", tt.doc, got, tt.want)
			}
		})
	}
}

// TestFifthEditionEntityName pins [68] EntityRef's Name as 5e: a d.Entity
// entry named with U+0133 resolves, where encoding/xml's 4e check leaves the
// reference unresolved.
func TestFifthEditionEntityName(t *testing.T) {
	const doc = `<a>&ĳ;</a>`
	ent := map[string]string{"ĳ": "ij"}

	std := xml.NewDecoder(strings.NewReader(doc))
	std.Entity = ent
	if _, err := drain(std); err == nil {
		t.Fatalf("encoding/xml resolved &ĳ;: the row no longer tests the 5e tables")
	}

	d := NewDecoder(strings.NewReader(doc))
	d.Entity = ent
	got, err := drain(d)
	if err != nil {
		t.Fatalf("Token on %q: %v", doc, err)
	}
	want := []xml.Token{
		xml.StartElement{Name: xml.Name{Local: "a"}, Attr: []xml.Attr{}},
		xml.CharData("ij"),
		xml.EndElement{Name: xml.Name{Local: "a"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Token on %q = %#v, want %#v", doc, got, want)
	}
}

// TestSyntaxErrors pins each fault's message, line and the offset it is
// reported at, against encoding/xml's own report of the same document.
func TestSyntaxErrors(t *testing.T) {
	tests := []struct {
		doc    string
		msg    string
		line   int
		offset int64
	}{
		// '-' is a [4a] NameChar but not a [4] NameStartChar in either
		// edition; the fault is reported after the name.
		{`<-04-29/>`, "invalid XML name: -04-29", 1, 7},
		// The name ends at '&', which no [41] Attribute continues; this is
		// not a name-table fault.
		{`<a a&b="0"/>`, "attribute name without = in element", 1, 5},
		// U+00B7 is [4a] only.
		{"<a>\n<\u00B7b/></a>", "invalid XML name: \u00B7b", 2, 8},
		// U+2041 is no NameChar in either edition.
		{"<a\u2041/>", "invalid XML name: a\u2041", 1, 5},
		{"<a>\n</a\u2041>", "invalid XML name: a\u2041", 2, 10},
	}
	for _, tt := range tests {
		t.Run(tt.doc, func(t *testing.T) {
			d := NewDecoder(strings.NewReader(tt.doc))
			_, err := drain(d)
			var se *xml.SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("Token on %q: err = %v, want *xml.SyntaxError", tt.doc, err)
			}
			if se.Msg != tt.msg || se.Line != tt.line {
				t.Errorf("Token on %q: err = %q at line %d, want %q at line %d", tt.doc, se.Msg, se.Line, tt.msg, tt.line)
			}
			if got := d.InputOffset(); got != tt.offset {
				t.Errorf("Token on %q: InputOffset after error = %d, want %d", tt.doc, got, tt.offset)
			}
		})
	}
}

// drain reads every token, copied, up to the first error, returning io.EOF
// as nil.
func drain(d interface{ Token() (xml.Token, error) }) ([]xml.Token, error) {
	var toks []xml.Token
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			return toks, nil
		}
		if err != nil {
			return toks, err
		}
		toks = append(toks, xml.CopyToken(tok))
	}
}

// accepted holds documents encoding/xml reads to the end, some of them not
// well-formed (it does not parse an XML declaration), and all of whose names
// are XML 1.0 4th-edition names, so this package must read them alike.
var accepted = []string{
	``,
	`<a/>`,
	`<a></a>`,
	`<a b="1" c='2'>text</a>`,
	"<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<root>\r\n  <child x=\"a&amp;b\"/>\r</root>\n",
	`<?xml version='1.0'?><a/>`,
	`<p:a xmlns:p="urn:p" xmlns="urn:d"><b p:c="1" d="2"><p:e/></b></p:a>`,
	`<a xmlns="urn:1"><b xmlns="urn:2"/><c/></a>`,
	`<a xml:lang="en" xmlns:q="urn:q"><q:b/></a>`,
	`<a>x<!-- comment - here -->y<![CDATA[<raw> & ]] ]]>z</a>`,
	`<a><?pi some data?><?other?></a>`,
	`<!DOCTYPE a [<!ENTITY e "v"> <!-- c > --> <!ATTLIST a b CDATA "x>y">]><a>&lt;&gt;&amp;&apos;&quot;</a>`,
	`<!DOCTYPE a SYSTEM "a.dtd"><a/>`,
	`<!DOCTYPE a [<!ELEMENT a (#PCDATA)> <x <y>>]><a/>`,
	`<a>&#65;&#x42;&#x1F600;&#10;</a>`,
	"<a b=\"line\r\nend\">\r\n\r</a>",
	`<日本語 属性="値">テキスト</日本語>`,
	`<a.b-c_d:e xmlns:a.b-c_d="urn:x"/>`,
	`<:a/>`,
	`<a:/>`,
	"<a>\n<b>\n<c/>\n</b>\n</a>",
	`<a><b><c/></b><d/></a>`,
	`<a>]]</a>`,
	`<a b="]]>"/>`,
	`<a>&e;</a>`,
	`<a>&#xd;&#13;</a>`,
	"<a>\u00e9\u4e2d\U0001F600</a>",
	`  <a/>  `,
	`<xmlns/>`,
	`<?xml version=1.0 encoding=x?><a/>`,
	`<?xml version="1.0?><a/>`,
}

// rejected holds documents encoding/xml rejects, none for a name only 5e
// admits, so the error, its line and the offset must agree.
var rejected = []string{
	`<!DOCTYPE a [<!x <!y >]><a/>`,
	`<a>`,
	`<a></b>`,
	`</a>`,
	`<a></a></a>`,
	`<p:a xmlns:p="urn:p"></q:a>`,
	`<p:a xmlns:p="urn:p" xmlns:q="urn:p"></q:a>`,
	`<-04-29/>`,
	`<a a&b="0"/>`,
	`<a b=c/>`,
	`<a b/>`,
	`<a b="1"`,
	`<a b="1`,
	`<a b="<"/>`,
	`<a/ >`,
	`<a></a b>`,
	`<a></ a>`,
	`</>`,
	`<>`,
	`<a:b:c/>`,
	`<a b:c:d="1"/>`,
	`< a/>`,
	`<!- x>`,
	`<!-- a -- b -->`,
	`<!-- a`,
	`<![CDAT[x]]>`,
	`<a><![CDATA[x</a>`,
	`<a>x]]>y</a>`,
	`<a>&foo;</a>`,
	`<a>&lt</a>`,
	`<a>&lt x</a>`,
	`<a>&#xZZ;</a>`,
	`<a>&#1114112;</a>`,
	`<a>&#;</a>`,
	`<a>&;</a>`,
	`<a>&1x;</a>`,
	"<a>\xff</a>",
	"<a b=\"\xfe\"/>",
	"<a>\x01</a>",
	"<a\xc3/>",
	`<?xml version="1.1"?><a/>`,
	`<?xml version="1.0" encoding="latin1"?><a/>`,
	`<? x?>`,
	`<?pi`,
	`<!DOCTYPE a`,
	`<!DOCTYPE a [<!-- x`,
	"<a>\n<b>\n</c>\n</a>",
	"<a\n\nb='1'\nc>\n</a>",
	"<\u00B7/>",
	"<a\u2041/>",
	"<a>\u00B7</a\u037E>",
	`<a></a`,
	`<a></a `,
	`<`,
	`<a`,
	`<!`,
	`<p:a xmlns:p="urn:p"></a>`,
	`<a ="1"/>`,
	`<a>&`,
	`<a>&#`,
	`<a>&#x`,
	"<\xc3/>",
	`<a>&x`,
	`<a b`,
	`<a b `,
}

// charsetReader converts the "x-upper" charset by upper-casing ASCII
// letters, so a decoder that failed to switch readers reads lower case, and
// fails on any other charset.
func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	if charset == "x-upper" {
		return upper{input}, nil
	}
	return nil, fmt.Errorf("no converter for %q", charset)
}

// upper upper-cases the ASCII letters of r.
type upper struct{ r io.Reader }

func (u upper) Read(p []byte) (int, error) {
	n, err := u.r.Read(p)
	for i, c := range p[:n] {
		if 'a' <= c && c <= 'z' {
			p[i] = c - 'a' + 'A'
		}
	}
	return n, err
}

// entities is the Entity map both decoders read in the differential.
var entities = map[string]string{"e": "<expanded>"}

// charsetDocs are read with charsetReader installed.
var charsetDocs = []string{
	`<?xml version="1.0" encoding="x-upper"?><a b="c">x</a>`,
	`<?xml version="1.0" encoding="x-bad"?><a>x</a>`,
	`<?xml version="1.0" encoding="utf-8"?><a/>`,
	`<?xml version="1.0" encoding="UtF-8"?><a/>`,
}

// TestDifferential requires this package to return the same token, error and
// InputOffset as encoding/xml after every call, over documents whose names
// are 4th-edition names: accepted, rejected, and every proper prefix of
// each accepted document (an end of input at each byte). Each is read
// three ways: by Token, by RawToken, and by Token with Skip after the first
// start element.
func TestDifferential(t *testing.T) {
	var docs []string
	docs = append(docs, accepted...)
	docs = append(docs, rejected...)
	for _, doc := range accepted {
		for i := range len(doc) {
			docs = append(docs, doc[:i])
		}
	}

	for _, doc := range accepted {
		std := xml.NewDecoder(strings.NewReader(doc))
		std.Entity = entities
		if _, err := drain(std); err != nil {
			t.Errorf("encoding/xml rejects accepted document %q: %v", doc, err)
		}
	}
	for _, doc := range rejected {
		if _, err := drain(xml.NewDecoder(strings.NewReader(doc))); err == nil {
			t.Errorf("encoding/xml reads rejected document %q", doc)
		}
	}

	var failed int
	for _, doc := range docs {
		std := xml.NewDecoder(strings.NewReader(doc))
		std.Entity = entities
		d := NewDecoder(strings.NewReader(doc))
		d.Entity = entities
		failed += compare(t, "Token", doc, std.Token, d.Token, std, d)

		std = xml.NewDecoder(strings.NewReader(doc))
		d = NewDecoder(strings.NewReader(doc))
		failed += compare(t, "RawToken", doc, std.RawToken, d.RawToken, std, d)

		failed += compareSkip(t, doc)
	}
	for _, doc := range charsetDocs {
		std := xml.NewDecoder(iotest.OneByteReader(strings.NewReader(doc)))
		std.CharsetReader = charsetReader
		d := NewDecoder(iotest.OneByteReader(strings.NewReader(doc)))
		d.CharsetReader = charsetReader
		failed += compare(t, "Token", doc, std.Token, d.Token, std, d)
	}
	if failed > 0 {
		t.Fatalf("%d of %d documents diverge from encoding/xml", failed, len(docs)+len(charsetDocs))
	}
}

// compare steps both token functions in lockstep and reports the first
// divergence, returning 1 when there is one.
func compare(t *testing.T, mode, doc string, stdNext, next func() (xml.Token, error), std *xml.Decoder, d *Decoder) int {
	t.Helper()
	for i := 0; ; i++ {
		wantTok, wantErr := stdNext()
		gotTok, gotErr := next()
		if !reflect.DeepEqual(gotTok, wantTok) || !sameError(gotErr, wantErr) || d.InputOffset() != std.InputOffset() {
			t.Errorf("%s on %q, call %d: got (%#v, %v) at offset %d, encoding/xml gives (%#v, %v) at offset %d",
				mode, doc, i, gotTok, gotErr, d.InputOffset(), wantTok, wantErr, std.InputOffset())
			return 1
		}
		if wantErr != nil {
			return 0
		}
	}
}

// compareSkip reads both decoders by Token through the first start element,
// calls Skip on each, and then compares the rest of the stream.
func compareSkip(t *testing.T, doc string) int {
	t.Helper()
	std := xml.NewDecoder(strings.NewReader(doc))
	d := NewDecoder(strings.NewReader(doc))
	for {
		wantTok, wantErr := std.Token()
		gotTok, gotErr := d.Token()
		if !reflect.DeepEqual(gotTok, wantTok) || !sameError(gotErr, wantErr) {
			t.Errorf("Skip on %q: tokens before Skip diverge: got (%#v, %v), encoding/xml gives (%#v, %v)",
				doc, gotTok, gotErr, wantTok, wantErr)
			return 1
		}
		if wantErr != nil {
			return 0
		}
		if _, ok := wantTok.(xml.StartElement); ok {
			break
		}
	}
	wantErr, gotErr := std.Skip(), d.Skip()
	if !sameError(gotErr, wantErr) || d.InputOffset() != std.InputOffset() {
		t.Errorf("Skip on %q: got %v at offset %d, encoding/xml gives %v at offset %d",
			doc, gotErr, d.InputOffset(), wantErr, std.InputOffset())
		return 1
	}
	if wantErr != nil {
		return 0
	}
	return compare(t, "Token after Skip", doc, std.Token, d.Token, std, d)
}

// sameError reports whether two errors have the same dynamic type and
// message; a *xml.SyntaxError's Line is part of its message.
func sameError(got, want error) bool {
	if got == nil || want == nil {
		return got == nil && want == nil
	}
	return reflect.TypeOf(got) == reflect.TypeOf(want) && got.Error() == want.Error()
}
