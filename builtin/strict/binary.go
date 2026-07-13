package strict

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"regexp"
	"strings"

	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsderr"
)

// hexBinaryLexical is hexBinary's lexical space (§3.3.15.2, nt-hexBinary): an
// even number of hex digits, matching '`([0-9a-fA-F]{2})*`'. Odd-length or
// non-hex input is outside the space, so Parse rejects it. whiteSpace=collapse is
// a pre-lexical pipeline stage, so no whitespace reaches this anchored production
// (mirrors floatingLexical/decimalLexical, which likewise reject stray space).
var hexBinaryLexical = regexp.MustCompile(`^([0-9a-fA-F]{2})*$`)

// base64BinaryLexical is base64Binary's lexical space (§3.3.16.2, nt-Base64Binary),
// written here as the spec's equivalent regular expression. Beyond the base64
// alphabet it enforces three constraints a naive decoder ignores: the
// non-whitespace character count is a multiple of four, '`=`' padding appears only
// at the end, and the character immediately before padding is drawn from the
// RESTRICTED B16char (`[AEIMQUYcgkosw048]`, low two bits zero) or B04char
// (`[AQgw]`, low four bits zero) subset — so the bits the padding discards really
// are zero. The optional single spaces are the grammar's inter-character #x20?;
// whiteSpace=collapse has already run before Parse, so any surviving space is a
// legal internal separator that carries no value.
var base64BinaryLexical = regexp.MustCompile(
	`^((([A-Za-z0-9+/] ?){4})*(([A-Za-z0-9+/] ?){3}[A-Za-z0-9+/]|([A-Za-z0-9+/] ?){2}[AEIMQUYcgkosw048] ?=|[A-Za-z0-9+/] ?[AQgw] ?= ?=))?$`)

// hexBinaryVal is an xs:hexBinary value (§3.3.15.1): a finite-length sequence of
// octets whose length is its octet count. It is value.Lengthed (octets, not hex
// characters — §4.3.1.3 clause 1.2), value.Eq and value.Canonical, and
// deliberately NOT value.Ordered (ordered=false, §3.3.15 fundamental facets): no
// bound facet applies to it (cos-applicable-facets §4.1.5).
type hexBinaryVal []byte

// base64BinaryVal is an xs:base64Binary value (§3.3.16.1): a finite-length
// sequence of octets whose length is its octet count. It carries the identical
// capability set to hexBinaryVal — value.Lengthed/Eq/Canonical, NOT value.Ordered
// (ordered=false, §3.3.16) — since both binary types share one applicable-facet
// set (cos-applicable-facets §4.1.5).
type base64BinaryVal []byte

// parseHexBinary maps a hexBinary lexical to its octet value (hexBinaryMap /
// hexOctetMap / hexDigitMap, E.4.1): each hexOctet is one octet, high nibble
// first. Lowercase a–f is accepted on input; the canonical mapping uppercases
// (f-hexBinaryCanonical, §3.3.15.2).
func parseHexBinary(lexical string, _ value.Context) (value.Value, error) {
	if !hexBinaryLexical.MatchString(lexical) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"hexBinary: %q is not in the lexical space (§3.3.15.2, nt-hexBinary: an even count of [0-9a-fA-F])", lexical)
	}
	octets, err := hex.DecodeString(lexical)
	if err != nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"hexBinary: decoding %q: %v", lexical, err)
	}
	return hexBinaryVal(octets), nil
}

// parseBase64Binary maps a base64Binary lexical to its octet value (the RFC 2045/
// 3548 decoding, §3.3.16.2). The lexical grammar is validated first — Go's decoder
// alone does not enforce the multiple-of-four, padding-position and restricted
// final-character rules — then the value is the decode of the space-stripped
// literal (the grammar's inter-character #x20 carries no value: the §3.3.16.2
// length pseudo-code likewise strips whitespace before decoding).
func parseBase64Binary(lexical string, _ value.Context) (value.Value, error) {
	if !base64BinaryLexical.MatchString(lexical) {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"base64Binary: %q is not in the lexical space (§3.3.16.2, nt-Base64Binary)", lexical)
	}
	octets, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(lexical, " ", ""))
	if err != nil {
		return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"base64Binary: decoding %q: %v", lexical, err)
	}
	return base64BinaryVal(octets), nil
}

// canonicalHexBinary is the Mapping.Canonical wrapper: it rejects a foreign value
// as an *xsderr.Error rather than panicking (warden guardrail).
func canonicalHexBinary(v value.Value) (string, error) {
	h, ok := v.(hexBinaryVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"hexBinary canonical: value of type %T is not a strict hexBinary", v)
	}
	return h.Canonical(), nil
}

// canonicalBase64Binary is the Mapping.Canonical wrapper for base64Binary.
func canonicalBase64Binary(v value.Value) (string, error) {
	b, ok := v.(base64BinaryVal)
	if !ok {
		return "", xsderr.New("cvc-datatype-valid", xsderr.Loc{},
			"base64Binary canonical: value of type %T is not a strict base64Binary", v)
	}
	return b.Canonical(), nil
}

// Canonical renders the hexBinary canonical lexical (f-hexBinaryCanonical, E.4.1):
// two uppercase hex digits per octet, A–F only.
func (h hexBinaryVal) Canonical() string { return strings.ToUpper(hex.EncodeToString(h)) }

// Canonical renders the base64Binary canonical lexical (§3.3.16.2, Canonical-
// base64Binary): the whitespace-free Base64 encoding of the octet value, which
// StdEncoding emits (standard alphabet, '=' padding, no line breaks).
func (b base64BinaryVal) Canonical() string { return base64.StdEncoding.EncodeToString(b) }

// Len is the octet count the length/minLength/maxLength facets measure — octets
// of binary data, NOT the count of lexical hex characters (§4.3.1.3 clause 1.2).
func (h hexBinaryVal) Len() int { return len(h) }

// Len is the octet count the length facets measure — decoded octets, NOT lexical
// base64 characters (§4.3.1.3 clause 1.2; §3.3.16.2 length pseudo-code).
func (b base64BinaryVal) Len() int { return len(b) }

// Eq is octet-sequence equality (§3.3.15.1): two hexBinary values are equal iff
// their octet sequences are identical. A non-hexBinary argument is unequal.
func (h hexBinaryVal) Eq(other value.Value) bool {
	o, ok := other.(hexBinaryVal)
	if !ok {
		return false
	}
	return bytes.Equal(h, o)
}

// Eq is octet-sequence equality for base64Binary (§3.3.16.1).
func (b base64BinaryVal) Eq(other value.Value) bool {
	o, ok := other.(base64BinaryVal)
	if !ok {
		return false
	}
	return bytes.Equal(b, o)
}
