package assess

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/codec"
	"github.com/kud360/goxsd8/value"
	"github.com/kud360/goxsd8/xsd"
)

// driveEncode runs DriveEncode as driveWith does.
func driveEncode(t *testing.T, codecs []Codec, cases ...Case) EncodeSummary {
	t.Helper()
	return driveWith[EncodeSummary](t, DriveEncode, codecs, cases...)
}

// secondDecode returns a's Codec with its decode replaced, from its second
// call on, by second over the value the first call returned: the first
// decode is codec's, and the round trip's decode is second's.
func secondDecode(second func(first *a) (any, error)) Codec {
	k := CodecFor[a]()
	var first *a
	decode := k.decode
	k.decode = func(schema *xsd.Schema, backend value.Backend, data []byte, opts ...codec.Option) (any, error) {
		if first != nil {
			return second(first)
		}
		v, err := decode(schema, backend, data, opts...)
		first, _ = v.(*a)
		return v, err
	}
	return k
}

// encoding returns a's Codec with its encode replaced by one returning out
// and err.
func encoding(out []byte, err error) Codec {
	k := CodecFor[a]()
	k.encode = func(*xsd.Schema, value.Backend, xsd.QName, any) ([]byte, error) { return out, err }
	return k
}

// TestDriveEncodeCountsEachRow holds DriveEncode, the loop the census's
// encode driver runs, to one row per bucket of a round trip, each row a
// case that lands in one bucket alone, and to no row for a case outside its
// population. encoded and declined run codec end to end; the rest inject
// the encode or the second decode no correct codec returns. Each of the six
// rows fails with its bucket's arm of tallyRoundTrip removed; the row over
// empty.xml, which the placeholder decodes (TestDriveCountsEachBucket),
// fails with DriveEncode's placeholder test removed, the differential's row
// with its differential test removed, and the row encoded under bound
// branches with CodecFor's encode passing codec.Append no bound option.
func TestDriveEncodeCountsEachRow(t *testing.T) {
	own := []Codec{CodecFor[a]()}
	cases := []struct {
		name   string
		codecs []Codec
		c      Case
		want   EncodeSummary
	}{
		{"encoded", own, Case{"S/g/instance/e", "valid.xml", 0}, EncodeSummary{Encoded: 1}},
		{"encoded under bound branches", []Codec{CodecFor[p](codec.WithBranches[p](pq{}, pr{}))}, Case{"S/g/instance/b", "choice.xml", 0}, EncodeSummary{Encoded: 1}},
		{"declined", []Codec{CodecFor[l](codec.WithBranches[lItem](lm{}))}, Case{"S/g/instance/c", "flat.xml", 0},
			EncodeSummary{Declines: []Tally{{Construct: "a simple type with an assertions facet", Cases: []string{"S/g/instance/c"}}}}},
		{"append failed", []Codec{encoding(nil, errors.New("a fault"))}, Case{"S/g/instance/f", "valid.xml", 0},
			EncodeSummary{AppendFailed: []string{"S/g/instance/f"}}},
		{"re-validate rejected", []Codec{encoding([]byte(`<a/>`), nil)}, Case{"S/g/instance/r", "valid.xml", 0},
			EncodeSummary{ReValidateRejected: []string{"S/g/instance/r"}}},
		{"re-decode failed", []Codec{secondDecode(func(*a) (any, error) { return nil, errors.New("a fault") })}, Case{"S/g/instance/d", "valid.xml", 0},
			EncodeSummary{ReDecodeFailed: []string{"S/g/instance/d"}}},
		{"round trip unequal", []Codec{secondDecode(func(first *a) (any, error) { return &a{B: first.B + "y"}, nil })}, Case{"S/g/instance/u", "valid.xml", 0},
			EncodeSummary{RoundTripUnequal: []string{"S/g/instance/u"}}},
		{"the placeholder is outside", nil, Case{"S/g/instance/m", "valid.xml", -1}, EncodeSummary{}},
		{"the placeholder is outside where it decodes", nil, Case{"S/g/instance/p", "empty.xml", -1}, EncodeSummary{}},
		{"an undecoded case is outside", own, Case{"S/g/instance/i", "invalid.xml", 0}, EncodeSummary{}},
		{"decode's differential is outside", []Codec{accepting}, Case{"S/g/instance/x", "invalid.xml", 0}, EncodeSummary{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := driveEncode(t, tc.codecs, tc.c); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("EncodeSummary = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestSameSkipsHeldNamespaces pins unread's exception: same does not compare
// the Namespaces of a struct of exactly AnyType's fields, at any depth, and
// compares every other field of it, and the Namespaces of a struct of any
// other fields, as it compares any field.
func TestSameSkipsHeldNamespaces(t *testing.T) {
	type (
		namespace struct{ Prefix, Space string }
		anyType   struct {
			Attributes []struct{ Space, Local, Value string }
			Namespaces []namespace
			Text       string
			Children   []anyType
		}
		fourNames struct{ Attributes, Namespaces, Text, Children string }
		other     struct {
			Namespaces []namespace
			Text       string
		}
	)
	p := []namespace{{Prefix: "p", Space: "urn:p"}}
	cases := []struct {
		name string
		a, b any
		want bool
	}{
		{"AnyType's Namespaces", anyType{Namespaces: p}, anyType{}, true},
		{"AnyType's Namespaces at depth", anyType{Children: []anyType{{Namespaces: p}}}, anyType{Children: []anyType{{}}}, true},
		{"AnyType's Text", anyType{Namespaces: p, Text: "x"}, anyType{Namespaces: p, Text: "y"}, false},
		{"another struct's Namespaces", other{Namespaces: p}, other{}, false},
		{"a struct of AnyType's field names whose Namespaces is a string", fourNames{Namespaces: "p"}, fourNames{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := same(reflect.ValueOf(tc.a), reflect.ValueOf(tc.b)); got != tc.want {
				t.Errorf("same(%#v, %#v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

// TestSameIsDatatypeIdentity pins same against the two answers Go's == and
// reflect.DeepEqual give otherwise: NaN, of two payloads, is the same as
// itself, and 0 is not the same as −0 (Datatypes §3.3.5.1), at a float64
// reached through a pointer, a struct field, a slice and an interface; a
// nil slice is the same as an empty one; and values of two types, or of
// different strings, are not the same.
func TestSameIsDatatypeIdentity(t *testing.T) {
	type doubles struct {
		V []float64
		W *float64
		X any
	}
	nan, other := math.NaN(), math.Float64frombits(0x7FF4000000000000)
	zero, negative := 0.0, math.Copysign(0, -1)
	cases := []struct {
		name string
		a, b any
		want bool
	}{
		{"NaN", &doubles{V: []float64{nan}, W: &nan, X: nan}, &doubles{V: []float64{other}, W: &other, X: other}, true},
		{"zero and negative zero in a slice", &doubles{V: []float64{zero}}, &doubles{V: []float64{negative}}, false},
		{"zero and negative zero through a pointer", &doubles{W: &zero}, &doubles{W: &negative}, false},
		{"zero and negative zero in an interface", &doubles{X: zero}, &doubles{X: negative}, false},
		{"a float32 zero and negative zero", float32(zero), float32(negative), false},
		{"nil and empty slice", &doubles{}, &doubles{V: []float64{}}, true},
		{"nil and non-nil pointer", &doubles{}, &doubles{W: &zero}, false},
		{"two types in an interface", &doubles{X: 1.0}, &doubles{X: 1}, false},
		{"two strings", &a{B: "x"}, &a{B: "y"}, false},
		{"one string", &a{B: "x"}, &a{B: "x"}, true},
		{"no value and a value", nil, &a{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := same(reflect.ValueOf(tc.a), reflect.ValueOf(tc.b)); got != tc.want {
				t.Errorf("same(%#v, %#v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
