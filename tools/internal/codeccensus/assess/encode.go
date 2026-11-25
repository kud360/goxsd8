package assess

import (
	"bytes"
	"errors"
	"io"
	"math"
	"reflect"

	"github.com/kud360/goxsd8/codec"
	"github.com/kud360/goxsd8/parser/xmltree"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// An EncodeSummary is what [DriveEncode] writes: each case DriveEncode
// round-trips lands in exactly one of its rows (roundTrip), and every other
// case in none. Encoded is a count; every other row holds the IDs of its
// cases, in the order DriveEncode met them, and its count is its list's
// length (STYLE D3), so that the census's -paths can name the cases behind
// it. Every field is exported for the JSON the census reads.
type EncodeSummary struct {
	// Encoded counts the cases whose value codec.Append wrote a document
	// that validation accepts and that decodes to the same value.
	Encoded int
	// Declines holds one Tally per construct codec.Append declined, in the
	// order the cases first met each.
	Declines []Tally
	// AppendFailed holds the cases whose codec.Append returned an error that
	// is no decline: on a value decoded from a document validation accepts,
	// each is a defect in codec.
	AppendFailed []string
	// ReValidateRejected holds the cases whose encoded document the census's
	// validation did not accept.
	ReValidateRejected []string
	// ReDecodeFailed holds the cases whose encoded document validation
	// accepts and codec.Decode, under the options of the first decode,
	// returned an error for.
	ReDecodeFailed []string
	// RoundTripUnequal holds the cases whose encoded document decodes to a
	// value that is not the same as the first decode's (same).
	RoundTripUnequal []string
}

// DriveEncode reads one JSON [Request] from r and writes one JSON
// [EncodeSummary] to w. It round-trips each case whose index names one of
// codecs, never the placeholder, and that [Drive] would count decoded and
// not under Summary.Differential: it encodes the decoded value with the
// Codec as the instance's document element names ([DocumentElement]),
// validates the encoded document, decodes it under the same options, and
// compares the two values (same). Its errors and its decline key are
// Drive's.
func DriveEncode(r io.Reader, w io.Writer, codecs []Codec) error {
	var sum EncodeSummary
	// index maps a construct to its position in sum.Declines, for lookup
	// only; nothing ranges over it (STYLE D2).
	index := map[codec.Construct]int{}
	err := walk(r, codecs, func(schema *xsd.Schema, backend value.Backend, data []byte, c Case, k Codec, own bool) {
		if !own {
			return
		}
		o, v := assess(schema, backend, data, c.Instance, k, own)
		if o.bucket != decoded || o.differential {
			return
		}
		tallyRoundTrip(&sum, index, c.ID, roundTrip(schema, backend, data, c.Instance, k, v))
	})
	if err != nil {
		return err
	}
	return writeSummary(w, sum)
}

// encodeBucket is the class of one round trip. The zero value is invalid
// (STYLE T1).
type encodeBucket uint8

const (
	// encoded: the round trip reached a value the same as the first.
	encoded encodeBucket = iota + 1
	// encodeDeclined: an error errors.As retrieves a *codec.DeclineError
	// from, items 4 and 7 of codec.Append's doc.
	encodeDeclined
	// appendFailed: every other codec.Append error.
	appendFailed
	// reValidateRejected: the census's validation did not accept the
	// encoded document.
	reValidateRejected
	// reDecodeFailed: decoding the encoded document returned an error.
	reDecodeFailed
	// roundTripUnequal: the second decode's value is not the same as the
	// first's.
	roundTripUnequal
)

// encodeOutcome is one round trip's classification: its bucket, and the
// construct of a declined one.
type encodeOutcome struct {
	bucket    encodeBucket
	construct codec.Construct
}

// roundTrip encodes v, the value k decoded from the instance data named
// uri, through k as data's document element names, and classifies the
// result in the first bucket that applies, in encodeBucket's order: a
// decline or other error of the encode, a document the census's validation
// does not accept, an error of decoding it under the same options as the
// first decode, a value that is not the same as v, and otherwise encoded.
func roundTrip(schema *xsd.Schema, backend value.Backend, data []byte, uri string, k Codec, v any) encodeOutcome {
	out, err := k.encode(schema, backend, DocumentElement(bytes.NewReader(data), uri), v)
	if err != nil {
		var de *codec.DeclineError
		if errors.As(err, &de) {
			return encodeOutcome{bucket: encodeDeclined, construct: de.Construct()}
		}
		return encodeOutcome{bucket: appendFailed}
	}
	if !accepts(schema, backend, out, uri) {
		return encodeOutcome{bucket: reValidateRejected}
	}
	again, err := k.decode(schema, backend, out, codec.WithURI(uri))
	if err != nil {
		return encodeOutcome{bucket: reDecodeFailed}
	}
	if !same(reflect.ValueOf(v), reflect.ValueOf(again)) {
		return encodeOutcome{bucket: roundTripUnequal}
	}
	return encodeOutcome{bucket: encoded}
}

// tallyRoundTrip adds o, the round trip of the case id, to sum, index
// mapping a construct to its Tally's position in sum.Declines.
func tallyRoundTrip(sum *EncodeSummary, index map[codec.Construct]int, id string, o encodeOutcome) {
	switch o.bucket {
	case encoded:
		sum.Encoded++
	case encodeDeclined:
		sum.Declines = addDecline(sum.Declines, index, o.construct, id)
	case appendFailed:
		sum.AppendFailed = append(sum.AppendFailed, id)
	case reValidateRejected:
		sum.ReValidateRejected = append(sum.ReValidateRejected, id)
	case reDecodeFailed:
		sum.ReDecodeFailed = append(sum.ReDecodeFailed, id)
	case roundTripUnequal:
		sum.RoundTripUnequal = append(sum.RoundTripUnequal, id)
	}
}

// same reports whether a and b hold the same value, the round trip's
// equality: of one type, each float by its bits with every NaN one, which
// is xs:double's identity (Datatypes §3.3.5.1: 0 and −0 are not identical,
// and NaN is identical to itself) and neither Go's == nor
// reflect.DeepEqual; each pointer and interface nil in both or holding the
// same value; each struct field by field, but for the Namespaces field of a
// struct of codegen's AnyType shape (unread); each slice and array by its
// length and items, so that a nil slice and an empty one are the same; and
// every other kind by ==, a kind == does not compare being no same value.
func same(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return a.IsValid() == b.IsValid()
	}
	if a.Type() != b.Type() {
		return false
	}
	switch a.Kind() {
	case reflect.Float32, reflect.Float64:
		x, y := a.Float(), b.Float()
		return math.Float64bits(x) == math.Float64bits(y) || math.IsNaN(x) && math.IsNaN(y)
	case reflect.Pointer, reflect.Interface:
		if a.IsNil() || b.IsNil() {
			return a.IsNil() && b.IsNil()
		}
		return same(a.Elem(), b.Elem())
	case reflect.Struct:
		for i := range a.NumField() {
			if unread(a.Type(), i) {
				continue
			}
			if !same(a.Field(i), b.Field(i)) {
				return false
			}
		}
		return true
	case reflect.Slice, reflect.Array:
		if a.Len() != b.Len() {
			return false
		}
		for i := range a.Len() {
			if !same(a.Index(i), b.Index(i)) {
				return false
			}
		}
		return true
	default:
		return a.Comparable() && a.Equal(b)
	}
}

// unread reports whether field i of the struct type t is the Namespaces field
// of a struct whose fields are exactly codegen's AnyType's, in order
// (codegen/doc.go "# xs:anyType"), which [same] does not compare.
// GAP(codec): #4125 — codec.Append writes none of the bindings Namespaces
// holds, so a re-decoded value holds those Append's own output puts in scope,
// not the source's; #4125 replaces this exception with its own comparison of
// them.
func unread(t reflect.Type, i int) bool {
	names := []string{"Attributes", "Namespaces", "Text", "Children"}
	if t.NumField() != len(names) || t.Field(i).Name != "Namespaces" {
		return false
	}
	for j, name := range names {
		if t.Field(j).Name != name {
			return false
		}
	}
	return true
}

// DocumentElement returns the expanded name of the document element of the
// instance r reads, named uri, or the zero QName when the document is not
// well-formed before it. The census reads it to resolve a case's type, and
// [DriveEncode] to name the declaration it encodes as.
func DocumentElement(r io.Reader, uri string) xsd.QName {
	x := xmltree.NewReader(uri, r)
	for {
		tok, err := x.Token()
		if err != nil {
			return xsd.QName{}
		}
		if start, ok := tok.(*xmltree.StartElement); ok {
			return xsd.QName{Space: start.Name().Space(), Local: start.Name().Local()}
		}
	}
}
