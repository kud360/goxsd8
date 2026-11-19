package jsonsrc

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/kud360/goxsd8/validate"
)

// charsSchema declares, in no namespace, r, whose optional children are m of
// xs:anyType, c of a mixed empty type, s of xs:string, n of N, which nests n
// and m again, t of B, which "$type" overrides with D, adding the attribute d
// of xs:anySimpleType and the child y of xs:anyType, and w and v, whose
// sequences are a lax and a skip urn:w element wildcard; and whose attribute
// uses are k of xs:string and ks of xs:anySimpleType. lax is laxExtensionSchema's
// r, whose {type table} makes walker.candidates call json.Decoder.More before
// the member after a scalar y is read.
const charsSchema = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="N"><xs:sequence>
    <xs:element name="n" type="N" minOccurs="0"/>
    <xs:element name="m" type="xs:anyType" minOccurs="0"/>
  </xs:sequence></xs:complexType>
  <xs:complexType name="B"><xs:sequence>
    <xs:element name="z" type="xs:anyType" minOccurs="0"/>
  </xs:sequence></xs:complexType>
  <xs:complexType name="D"><xs:complexContent><xs:extension base="B">
    <xs:sequence><xs:element name="y" type="xs:anyType" minOccurs="0"/></xs:sequence>
    <xs:attribute name="d" type="xs:anySimpleType"/>
  </xs:extension></xs:complexContent></xs:complexType>
  <xs:element name="r"><xs:complexType><xs:sequence>
    <xs:element name="m" type="xs:anyType" minOccurs="0"/>
    <xs:element name="c" minOccurs="0"><xs:complexType mixed="true"/></xs:element>
    <xs:element name="s" type="xs:string" minOccurs="0"/>
    <xs:element name="n" type="N" minOccurs="0"/>
    <xs:element name="t" type="B" minOccurs="0"/>
    <xs:element name="w" minOccurs="0"><xs:complexType><xs:sequence>
      <xs:any namespace="urn:w" processContents="lax" minOccurs="0"/>
    </xs:sequence></xs:complexType></xs:element>
    <xs:element name="v" minOccurs="0"><xs:complexType><xs:sequence>
      <xs:any namespace="urn:w" processContents="skip" minOccurs="0"/>
    </xs:sequence></xs:complexType></xs:element>
  </xs:sequence>
  <xs:attribute name="k" type="xs:string"/>
  <xs:attribute name="ks" type="xs:anySimpleType"/>
  </xs:complexType></xs:element>
</xs:schema>`

// nonCharDecline is the message of walker.chars' decline of the non-Char r.
func nonCharDecline(r rune) string {
	return fmt.Sprintf("the string holds U+%04X, which is no XML 1.0 Char (XML 1.0 §2.2 [2]): no XML document holds it, as text or as a character reference", r)
}

// loneDecline is the message of walker.chars' decline of the lone surrogate
// escape esc.
func loneDecline(esc string) string {
	return `the string holds the lone surrogate escape ` + esc + `, which names no character, and so no XML 1.0 Char (XML 1.0 §2.2 [2]): no XML document holds one`
}

// notUTF8 is the message of walker.chars' malformed error.
const notUTF8 = "malformed JSON: a string's source text is not UTF-8 (RFC 8259 §8.1)"

// TestStringChars pins walker.chars over every string token walker.token
// reads — content, an attribute member's value, a key at the root, under a
// lax or skip wildcard, nested and under a "$type", a "$xmlns" prefix and
// namespace name, a "$type" — each read whole from a strings.Reader and a
// byte per Read: a string holding a non-Char, escaped or literal, or a lone
// surrogate escape declines, the first in document order named, one whose
// source is not UTF-8 is malformed wherever a decline's character also is,
// and the guards, which the decoder decodes to U+FFFD, a paired surrogate or
// an escaped Char, or which escape the backslash before the text ud800, are
// assessed with no violation. A guard beside an escaped U+FFFD is one the
// source scan reads, the others being answered off the decoded string alone;
// the last row, read whole, pins the scan to the token, a scan to the end of
// what the decoder has read declining the U+0000 of the string after it at
// the U+FFFD's location. Error is the whole of
// err.Error(), the location and message prefix included; an empty one asks
// for a result standing NotProvenInvalid with no violation and nothing
// unevaluated.
func TestStringChars(t *testing.T) {
	v := validatorOver(t, charsSchema)
	lax := validatorOver(t, laxExtensionSchema)
	at := func(col int, msg string) string { return fmt.Sprintf("jsonsrc: instance.json:1:%d: %s", col, msg) }
	for _, tc := range []struct {
		name  string
		v     *validate.Validator
		json  string
		class Class
		err   string
	}{
		{"escaped U+0000 under anyType", v, `{"r":{"m":"a\u0000b"}}`, Decline, at(11, nonCharDecline(0))},
		{"escaped U+0001 under a mixed type", v, `{"r":{"c":"a\u0001b"}}`, Decline, at(11, nonCharDecline(1))},
		{"escaped U+0000 under xs:string, charged at base", v, `{"r":{"s":"a\u0000b"}}`, Decline, at(11, nonCharDecline(0))},
		{"literal U+FFFE", v, "{\"r\":{\"m\":\"a\uFFFEb\"}}", Decline, at(11, nonCharDecline(0xFFFE))},
		{"escaped U+FFFF", v, `{"r":{"m":"\uffff"}}`, Decline, at(11, nonCharDecline(0xFFFF))},
		{"backspace escape", v, `{"r":{"m":"a\bb"}}`, Decline, at(11, nonCharDecline(8))},
		{"form feed escape", v, `{"r":{"m":"a\fb"}}`, Decline, at(11, nonCharDecline(0xC))},
		{"lone high surrogate", v, `{"r":{"m":"a\ud800b"}}`, Decline, at(11, loneDecline(`\ud800`))},
		{"lone high surrogate under xs:string", v, `{"r":{"s":"a\ud800b"}}`, Decline, at(11, loneDecline(`\ud800`))},
		{"lone low surrogate", v, `{"r":{"m":"\uDC00"}}`, Decline, at(11, loneDecline(`\uDC00`))},
		{"high surrogate before a non-surrogate escape", v, `{"r":{"m":"\ud800\u0041"}}`, Decline, at(11, loneDecline(`\ud800`))},
		{"low surrogate before a high one", v, `{"r":{"m":"\udc00\ud800"}}`, Decline, at(11, loneDecline(`\udc00`))},
		{"escaped backslash before a lone surrogate", v, `{"r":{"m":"\\\ud800"}}`, Decline, at(11, loneDecline(`\ud800`))},
		{"lone surrogate first", v, `{"r":{"m":"\ud800\u0000"}}`, Decline, at(11, loneDecline(`\ud800`))},
		{"non-Char first", v, `{"r":{"m":"\u0000\ud800"}}`, Decline, at(11, nonCharDecline(0))},
		{"non-Char after a paired surrogate", v, `{"r":{"m":"\uD834\uDD1E\u0000"}}`, Decline, at(11, nonCharDecline(0))},
		{"attribute under xs:string, charged at base", v, `{"r":{"k":"a\u0000b"}}`, Decline, at(11, nonCharDecline(0))},
		{"attribute under xs:anySimpleType", v, `{"r":{"ks":"a\u0000b"}}`, Decline, at(12, nonCharDecline(0))},
		{"lone surrogate in an attribute", v, `{"r":{"ks":"a\ud800b"}}`, Decline, at(12, loneDecline(`\ud800`))},
		{"nested", v, `{"r":{"n":{"n":{"n":{"m":"a\ud800b"}}}}}`, Decline, at(26, loneDecline(`\ud800`))},
		{"under a \"$type\"", v, `{"r":{"t":{"$type":"D","d":"a\ud800b","y":"x"}}}`, Decline, at(28, loneDecline(`\ud800`))},
		{"child under a \"$type\"", v, `{"r":{"t":{"$type":"D","y":"a\u0000b"}}}`, Decline, at(28, nonCharDecline(0))},
		{"\"$type\" value", v, `{"r":{"t":{"$type":"D\u0000"}}}`, Decline, at(20, nonCharDecline(0))},
		{"prefixed key under a lax wildcard", v, `{"r":{"w":{"$xmlns":{"p":"urn:w"},"p:q\ud800":"x"}}}`, Decline, at(35, loneDecline(`\ud800`))},
		{"under a lax wildcard", v, `{"r":{"w":{"$xmlns":{"p":"urn:w"},"p:q":"a\ud800b"}}}`, Decline, at(41, loneDecline(`\ud800`))},
		{"under a skip wildcard", v, `{"r":{"v":{"$xmlns":{"p":"urn:w"},"p:q":"a\u0000b"}}}`, Decline, at(41, nonCharDecline(0))},
		{"root key", v, `{"r\ud800":{}}`, Decline, at(2, loneDecline(`\ud800`))},
		{"member key", v, `{"r":{"x\ud800":"v"}}`, Decline, at(7, loneDecline(`\ud800`))},
		{"non-Char member key", v, `{"r":{"x\u0000":"v"}}`, Decline, at(7, nonCharDecline(0))},
		{"\"$xmlns\" prefix", v, `{"r":{"$xmlns":{"p\ud800":"urn:x"},"m":"x"}}`, Decline, at(17, loneDecline(`\ud800`))},
		{"\"$xmlns\" namespace name", v, `{"r":{"$xmlns":{"p":"urn:\u0000"},"m":"x"}}`, Decline, at(21, nonCharDecline(0))},
		{"lone surrogate \"$xmlns\" namespace name", v, `{"r":{"$xmlns":{"p":"urn:\ud800"},"m":"x"}}`, Decline, at(21, loneDecline(`\ud800`))},
		{"after json.Decoder.More reads ahead", lax, `{"r":{"y":"1","z":"a\ud800b"}}`, Decline, at(19, loneDecline(`\ud800`))},
		{"key after json.Decoder.More reads ahead", lax, `{"r":{"y":"1","z\ud800":"2"}}`, Decline, at(15, loneDecline(`\ud800`))},
		{"raw 0xFF", v, "{\"r\":{\"m\":\"a\xffb\"}}", Malformed, at(11, notUTF8)},
		{"raw CESU-8 surrogate", v, "{\"r\":{\"m\":\"a\xed\xa0\x80b\"}}", Malformed, at(11, notUTF8)},
		{"raw 0xFF in a key", v, "{\"r\":{\"m\xff\":\"v\"}}", Malformed, at(7, notUTF8)},
		{"raw 0xFF after a non-Char", v, "{\"r\":{\"m\":\"\\u0000\xff\"}}", Malformed, at(11, notUTF8)},
		{"raw 0xFF after a lone surrogate", v, "{\"r\":{\"m\":\"\\ud800\xff\"}}", Malformed, at(11, notUTF8)},
		{"escaped U+FFFD", v, `{"r":{"m":"\ufffd"}}`, 0, ""},
		{"literal U+FFFD", v, `{"r":{"m":"a�b"}}`, 0, ""},
		{"paired surrogate escapes", v, `{"r":{"m":"\ud834\udd1e"}}`, 0, ""},
		{"upper-case paired surrogate escapes", v, `{"r":{"m":"\uD834\uDD1E"}}`, 0, ""},
		{"tab, LF and CR escapes", v, `{"r":{"m":"\t\n\r"}}`, 0, ""},
		{"the other escapes", v, `{"r":{"m":"\"\\\/\u0041\u00e9"}}`, 0, ""},
		{"the empty string", v, `{"r":{"m":""}}`, 0, ""},
		{"escaped backslash before the text ud800", v, `{"r":{"m":"a\\ud800b"}}`, 0, ""},
		{"escaped U+FFFD in an attribute", v, `{"r":{"ks":"\ufffd"}}`, 0, ""},
		{"supplementary-plane literal", v, "{\"r\":{\"m\":\"\U0001D11E\"}}", 0, ""},
		{"paired surrogate escapes beside an escaped U+FFFD", v, `{"r":{"m":"\ud834\udd1e\ufffd"}}`, 0, ""},
		{"tab, LF and CR escapes beside an escaped U+FFFD", v, `{"r":{"m":"\t\n\r\ufffd"}}`, 0, ""},
		{"the other escapes beside an escaped U+FFFD", v, `{"r":{"m":"\"\\\/\u0041\u00e9\ufffd"}}`, 0, ""},
		{"escaped backslash before the text ud800, beside an escaped U+FFFD", v, `{"r":{"m":"a\\ud800b\ufffd"}}`, 0, ""},
		{"a non-Char in the string after an escaped U+FFFD", v, `{"r":{"m":"\ufffd","s":"\u0000"}}`, Decline, at(24, nonCharDecline(0))},
	} {
		for _, rd := range []struct {
			name string
			r    func(string) io.Reader
		}{
			{"whole", func(s string) io.Reader { return strings.NewReader(s) }},
			{"a byte per Read", func(s string) io.Reader { return iotest.OneByteReader(strings.NewReader(s)) }},
		} {
			t.Run(tc.name+", "+rd.name, func(t *testing.T) {
				res, err := Validate(tc.v, rd.r(tc.json), WithURI(jsonURI))
				if tc.class == 0 {
					if err != nil {
						t.Fatalf("Validate(%s) = %v, want a result", tc.json, err)
					}
					if res.Standing() != validate.NotProvenInvalid || len(res.Violations()) != 0 || len(res.Unevaluated()) != 0 {
						t.Fatalf("Validate(%s) stands %v with %d violations, %d unevaluated; want NotProvenInvalid 0/0", tc.json, res.Standing(), len(res.Violations()), len(res.Unevaluated()))
					}
					return
				}
				var e *Error
				if !errors.As(err, &e) {
					t.Fatalf("Validate(%s) = %v, want an *Error", tc.json, err)
				}
				if e.Class() != tc.class {
					t.Errorf("Validate(%s): Class() = %d, want %d", tc.json, e.Class(), tc.class)
				}
				if err.Error() != tc.err {
					t.Errorf("Validate(%s) = %q, want %q", tc.json, err.Error(), tc.err)
				}
			})
		}
	}
}
