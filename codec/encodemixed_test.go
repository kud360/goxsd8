package codec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/xsd"
)

// TestAppendMixed pins the mixed-content lift of Append (§3.4.1 {variety}
// mixed), written as codec/doc.go "# What encodes" states, each written
// document validating and decoding back equal. The rows cover every field
// shape mixed content gives (codegen/doc.go "# Complex types"), an attribute
// use after the content's fields (binding.lead), a choice's branch field
// after Text, an empty {particle}, an {open content}'s AnyChild children
// after the particle's child, each with its Tail, the escaping of '<', '&',
// '>' and CR in Text and Tails, and a {value constraint}: a fixed one's
// {lexical form} as Text, and an empty element under a fixed or default one,
// which takes on nothing Decode returns (cvc-elt clause 5.1).
func TestAppendMixed(t *testing.T) {
	str := func(s string) *string { return &s }
	const r = `<r xmlns="urn:s">`
	q, p := ocX("q", "q"), ocX("p", "p")
	rows := []struct {
		name string
		run  func(t *testing.T)
	}{
		{"a sequence, Text, two children and a Tail on each", ocRoundTrip(mxPairSchema,
			mxPair{Text: "x", A: mixedChild[string]{Element: "1", Tail: "y"}, B: mixedChild[int32]{Element: 2, Tail: "z"}}, r+`x<a>1</a>y<b>2</b>z</r>`)},
		{"an attribute use after the content's fields", ocRoundTrip(mxExtAttrSchema,
			mxExtAttr{Text: "x", A: mixedChild[string]{Element: "v", Tail: "y"}, K: str("kv")}, `<r xmlns="urn:s" k="kv">x<a>v</a>y</r>`)},
		{"a choice's MixedChild branch", ocRoundTrip(mxPickSchema,
			mxPick{Text: "x", Choice: mxPickB{B: mixedChild[int32]{Element: 2, Tail: "y"}}}, r+`x<b>2</b>y</r>`, mxPickBranches)},
		{"a wildcard particle's AnyChild and its Tail", ocRoundTrip(mxWildSchema,
			mxWild{Text: "t", Any: anyChild{Element: q, Tail: "u"}}, r+`t<p1:q xmlns="" xmlns:p1="urn:o">q</p1:q>u</r>`)},
		{"guard: Text and Tails holding <, &, > and CR, escaped", ocRoundTrip(mxPairSchema,
			mxPair{Text: "x<&>\r", A: mixedChild[string]{Element: "1", Tail: "]]>"}, B: mixedChild[int32]{Element: 2, Tail: "a\r\nb"}}, r+`x&lt;&amp;&gt;&#xD;<a>1</a>]]&gt;<b>2</b>a&#xD;
b</r>`)},
		{"guard: an {open content}'s AnyChild children and Tails after the particle's child", ocRoundTrip(mxOpenSchema,
			mxOpen{Text: "t", A: mixedChild[string]{Element: "v", Tail: "y"}, OpenContent: []anyChild{{Element: q, Tail: "w"}, {Element: p, Tail: "z"}}},
			r+`t<a>v</a>y<p1:q xmlns="" xmlns:p1="urn:o">q</p1:q>w<p1:p xmlns="" xmlns:p1="urn:o">p</p1:p>z</r>`)},
		{"white space around children, verbatim", ocRoundTrip(mxPairSchema,
			mxPair{Text: "\n  ", A: mixedChild[string]{Element: "1", Tail: "\n  "}, B: mixedChild[int32]{Element: 2, Tail: "\n"}}, "")},
		{"an empty {particle}, Text and an attribute", ocRoundTrip(mxEmptySchema, mxEmpty{Text: "hello", K: str("kv")}, `<r xmlns="urn:s" k="kv">hello</r>`)},
		{"no Text and no child, an empty-element tag", ocRoundTrip(mxOptSchema, mxOpt{}, `<r xmlns="urn:s"/>`)},
		{"a 0..1 particle, absent", ocRoundTrip(mxOptSchema, mxOpt{Text: "just text"}, r+`just text</r>`)},
		{"an extension of a mixed base adding a particle", ocRoundTrip(mxExtSchema,
			mxPair{Text: "x", A: mixedChild[string]{Element: "v", Tail: "y"}, B: mixedChild[int32]{Element: 1, Tail: "z"}}, "")},
		{"an element of type xs:anyType", ocRoundTrip(mxAnyTypeSchema,
			mxAny{Text: "x", A: mixedChild[anyType]{Element: anyType{Text: "in", Children: []anyChild{{Element: anyElement{Space: "urn:s", Local: "c"}, Tail: "out"}}}, Tail: "y"}}, "")},
		{"a flattened content model", ocRoundTrip(mxFlatSchema,
			mxFlat{Text: "a", Sequence: []mxFlatI{mxFlatA{A: mixedChild[string]{Element: "1", Tail: "b"}}, mxFlatB{B: mixedChild[int32]{Element: 2, Tail: "c"}}}}, r+`a<a>1</a>b<b>2</b>c</r>`, WithBranches[mxFlatI](mxFlatA{}, mxFlatB{}))},
		{"a 0..1 element of a sealed interface", ocRoundTrip(mxSealedOptSchema,
			mxSealedOpt{Text: "t", A: &mixedChild[mxAI]{Element: mxAY{Y: 2}, Tail: "u"}}, r+`t<a><y>2</y></a>u</r>`, WithBranches[mxAI](mxAX{}, mxAY{}))},
		{"a repeated choice, held in an occurrence field", ocRoundTrip(mxRepChoiceSchema,
			mxRepChoice{Text: "a", Choice: []mxPickI{mxPickA{A: mixedChild[string]{Element: "1", Tail: "b"}}, mxPickB{B: mixedChild[int32]{Element: 2, Tail: "c"}}}}, r+`a<a>1</a>b<b>2</b>c</r>`, mxPickBranches)},
		{"an occurrence content model", ocRoundTrip(mxOccSchema,
			mxOcc{Text: "w", Sequence: []mxOccItem{{A: mixedChild[string]{Element: "1", Tail: "x"}, B: mixedChild[int32]{Element: 2, Tail: "y"}}, {A: mixedChild[string]{Element: "3"}, B: mixedChild[int32]{Element: 4, Tail: "z"}}}}, r+`w<a>1</a>x<b>2</b>y<a>3</a><b>4</b>z</r>`)},
		{"an all group, its wildcard's AnyChild after its element", ocRoundTrip(mxAllSchema,
			mxAll{Text: "t", A: mixedChild[string]{Element: "v", Tail: "y"}, Any: anyChild{Element: q, Tail: "w"}}, r+`t<a>v</a>y<p1:q xmlns="" xmlns:p1="urn:o">q</p1:q>w</r>`)},
		{"a choice's wildcard branch", ocRoundTrip(mxNameAnySchema,
			mxNameAny{Text: "x", Choice: mxNameAnyAny{Any: anyChild{Element: q, Tail: "y"}}}, "", WithBranches[mxNameAnyI](mxNameAnyAny{}, mxNameAnyAny2{}))},
		{"a choice's element branch beside a wildcard", ocRoundTrip(mxNameAnySchema,
			mxNameAny{Text: "x", Choice: mxNameAnyAny2{Any: mixedChild[string]{Element: "v", Tail: "y"}}}, r+`x<any>v</any>y</r>`, WithBranches[mxNameAnyI](mxNameAnyAny{}, mxNameAnyAny2{}))},
		{"a mixed restriction of xs:anyType", ocRoundTrip(mxAnyRestrictSchema, mxB{Text: "t", B: &mixedChild[string]{Element: "v"}}, r+`t<b>v</b></r>`)},
		{"a fixed {value constraint}'s {lexical form} as Text", ocRoundTrip(mxFixedSchema, mxOpt{Text: "abc"}, r+`abc</r>`)},
		{"an empty element under a fixed {value constraint}", ocRoundTrip(mxFixedSchema, mxOpt{}, `<r xmlns="urn:s"/>`)},
		{"an empty element under a default", ocRoundTrip(mxDefaultSchema, mxOpt{}, `<r xmlns="urn:s"/>`)},
		{"Text and a child under a default", ocRoundTrip(mxDefaultSchema, mxOpt{Text: "x", A: &mixedChild[string]{Element: "v", Tail: "y"}}, r+`x<a>v</a>y</r>`)},
	}
	for _, row := range rows {
		t.Run(row.name, row.run)
	}
}

// TestAppendMixedPara round-trips codegen/testdata/mixed.xsd's para, whose
// struct holds every shape a mixed-content field takes (TestDecodeMixedPara),
// a substitution group's member and a nested complex type among them.
func TestAppendMixedPara(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "codegen", "testdata", "mixed.xsd"))
	if err != nil {
		t.Fatal(err)
	}
	schema := parseString(t, string(doc))
	opts := []Option{WithBranches[mxHeadI](mxHeadHead{}, mxHeadMember{}), WithBranches[mxParaChoiceI](mxParaChoiceX{}, mxParaChoiceSeq{})}
	v := mxPara{
		Text:     "t0",
		Text2:    mixedChild[int32]{Element: 7, Tail: "t1"},
		Em:       &mixedChild[string]{Element: "e", Tail: "t2"},
		B:        []mixedChild[string]{{Element: "b1", Tail: "t3"}, {Element: "b2", Tail: "t4"}},
		Mark:     &mixedChild[mxMark]{Element: "m", Tail: "t5"},
		Head:     &mixedChild[mxHeadI]{Element: mxHeadMember{Member: "s"}, Tail: "t6"},
		Link:     mixedChild[mxLink]{Element: mxLink{Sequence: []mxLinkSeq{{Href: "h"}}}, Tail: "t7"},
		Any:      []anyChild{{Element: anyElement{Space: "urn:w", Local: "p", Content: anyType{Text: "w"}}, Tail: "t8"}},
		Choice:   mxParaChoiceSeq{Y: mixedChild[string]{Element: "y", Tail: "t9"}, Z: []mixedChild[string]{{Element: "z1", Tail: "t10"}}},
		Sequence: []mxParaSeq{{Href: mixedChild[string]{Element: "h2", Tail: "t11"}}},
	}
	const want = `<para>t0<text>7</text>t1<em>e</em>t2<b>b1</b>t3<b>b2</b>t4<mark>m</mark>t5<member>s</member>t6<link><href>h</href></link>t7<p1:p xmlns:p1="urn:w">w</p1:p>t8<y>y</y>t9<z>z1</z>t10<href>h2</href>t11</para>`
	roundTrip(t, schema, xsd.QName{Local: "para"}, v, want, deepEqual[mxPara], opts...)
}

// TestAppendMixedRefusals pins the mixed-content values item 6 of Append's
// doc refuses, each error opening with its subject, the value's Go path
// among it: under a fixed {value constraint}, a Text other than "" and the
// {lexical form} (cvc-elt clause 5.2.2.2.1), and a child (clause 5.2.2.1),
// which comes ahead of an error inside that child and after a Text's; a
// MixedChild whose Element is a nil sealed interface; and a Text or Tail
// holding a byte that starts no XML 1.0 Char, an {open content} child's Tail
// among them.
func TestAppendMixedRefusals(t *testing.T) {
	sealed := WithBranches[mxAI](mxAX{}, mxAY{})
	rows := []struct {
		name   string
		decls  string
		v      any
		prefix string
		want   string
	}{
		{"fixed, a Text other than the {lexical form}", mxFixedSchema, mxOpt{Text: "xyz"},
			`codec: Append: mxOpt.Text holds "xyz", which does not match "abc", the {lexical form} of the fixed {value constraint} of `, "cvc-elt clause 5.2.2.2.1"},
		{"fixed, a child", mxFixedSchema, mxOpt{A: &mixedChild[string]{Element: "v"}},
			`codec: Append: mxOpt holds a child element, though the fixed {value constraint} "abc" of `, "cvc-elt clause 5.2.2.1"},
		{"fixed, the {lexical form} as Text and a child", mxFixedSchema, mxOpt{Text: "abc", A: &mixedChild[string]{Element: "v"}},
			`codec: Append: mxOpt holds a child element, though the fixed {value constraint} "abc" of `, "cvc-elt clause 5.2.2.1"},
		{"fixed, a child ahead of the error inside it", mxFixedSchema, mxOpt{A: &mixedChild[string]{Element: "\x00"}},
			`codec: Append: mxOpt holds a child element, though the fixed {value constraint} "abc" of `, "cvc-elt clause 5.2.2.1"},
		{"fixed, a Text other than the {lexical form} ahead of a child", mxFixedSchema, mxOpt{Text: "xyz", A: &mixedChild[string]{Element: "v"}},
			`codec: Append: mxOpt.Text holds "xyz", which does not match "abc"`, "cvc-elt clause 5.2.2.2.1"},
		{"a MixedChild whose Element is a nil sealed interface", mxSealedOptSchema, mxSealedOpt{Text: "t", A: &mixedChild[mxAI]{Tail: "u"}},
			"codec: Append: mxSealedOpt.A.Element is nil, a MixedChild holding no element of ", "which Decode never returns"},
		{"a Text holding no Char", mxPairSchema, mxPair{Text: "\x00"},
			`codec: Append: mxPair.Text holds "\x00", whose byte offset 0 starts no XML 1.0 Char`, ""},
		{"a Tail holding no Char", mxPairSchema, mxPair{B: mixedChild[int32]{Tail: "a\x01"}},
			`codec: Append: mxPair.B.Tail holds "a\x01", whose byte offset 1 starts no XML 1.0 Char`, ""},
		{"an {open content} child's Tail holding no Char", mxOpenSchema, mxOpen{OpenContent: []anyChild{{Element: ocX("q", ""), Tail: "\x00"}}},
			`codec: Append: mxOpen.OpenContent[0].Tail holds "\x00", whose byte offset 0 starts no XML 1.0 Char`, ""},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			schema := schemaOf(t, row.decls)
			var err error
			// The row's T is the dynamic type of row.v, a closed set of the
			// types above, so the switch is STYLE T2's sealed-sum exception.
			switch v := row.v.(type) {
			case mxOpt:
				_, err = appendChecked(t, nil, schema, strict.New(), sname("r"), v)
			case mxSealedOpt:
				_, err = appendChecked(t, nil, schema, strict.New(), sname("r"), v, sealed)
			case mxPair:
				_, err = appendChecked(t, nil, schema, strict.New(), sname("r"), v)
			case mxOpen:
				_, err = appendChecked(t, nil, schema, strict.New(), sname("r"), v)
			}
			invalidValue(t, err)
			prefixed(t, err, row.prefix, row.want)
		})
	}
}
