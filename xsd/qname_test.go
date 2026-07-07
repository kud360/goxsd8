package xsd_test

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestQNameString(t *testing.T) {
	cases := []struct {
		name string
		q    xsd.QName
		want string
	}{
		{"namespaced", xsd.QName{Space: "http://example.com/ns", Local: "elem"}, "{http://example.com/ns}elem"},
		{"no-namespace", xsd.QName{Local: "elem"}, "elem"},
		{"zero-value-absent", xsd.QName{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.q.String(); got != tc.want {
				t.Errorf("QName%+v.String() = %q, want %q", tc.q, got, tc.want)
			}
		})
	}
}

func TestQNameComparable(t *testing.T) {
	a := xsd.QName{Space: "urn:x", Local: "n"}
	b := xsd.QName{Space: "urn:x", Local: "n"}
	c := xsd.QName{Space: "urn:y", Local: "n"}

	if a != b {
		t.Errorf("equal QNames compared unequal: %v != %v", a, b)
	}
	if a == c {
		t.Errorf("distinct QNames compared equal: %v == %v", a, c)
	}

	m := map[xsd.QName]int{}
	m[a]++
	m[b]++ // same key as a — must collide
	m[c]++
	if m[a] != 2 {
		t.Errorf("equal QNames did not collide as map keys: got count %d, want 2", m[a])
	}
	if len(m) != 2 {
		t.Errorf("map has %d distinct keys, want 2", len(m))
	}
}
