package nearname

import "testing"

func local(n Name) string { return n.Local }

// TestHint pins each part of the package doc's rule by a row that fails when
// that part is changed.
func TestHint(t *testing.T) {
	const xs = "http://www.w3.org/2001/XMLSchema"
	tests := []struct {
		name       string
		missing    Name
		candidates []Name
		want       string
	}{
		{"one substitution", Name{xs, "strng"}, []Name{{xs, "token"}, {xs, "string"}}, "; did you mean string?"},
		{"no candidate", Name{"", "nope"}, nil, ""},
		{"namespace differs", Name{"", "strng"}, []Name{{xs, "string"}}, ""},
		{"absent equals only absent", Name{"urn:a", "strng"}, []Name{{"", "string"}}, ""},
		{"anonymous never qualifies", Name{"", "a"}, []Name{{}}, ""},
		{"short name allows 1", Name{"", "fo"}, []Name{{"", "foo"}}, "; did you mean foo?"},
		{"short name refuses 2", Name{"", "ab"}, []Name{{"", "xy"}}, ""},
		{"five runes refuse 2", Name{"", "abcde"}, []Name{{"", "abcxy"}}, ""},
		{"six runes allow 2", Name{"", "anyTyp"}, []Name{{"", "anyType"}, {"", "anyTxpe"}}, "; did you mean anyType?"},
		{"six runes allow 2 exactly", Name{"", "abcdef"}, []Name{{"", "abcdxy"}}, "; did you mean abcdxy?"},
		{"cap is 2", Name{"", "abcdefghijkl"}, []Name{{"", "abcdefghixyz"}}, ""},
		{"least distance wins", Name{"", "abcdef"}, []Name{{"", "abcdxy"}, {"", "abcdex"}}, "; did you mean abcdex?"},
		{"tie goes to earliest", Name{"", "abc"}, []Name{{"", "abd"}, {"", "abe"}}, "; did you mean abd?"},
		{"case-sensitive", Name{"", "abc"}, []Name{{"", "ABC"}}, ""},
		{"runes not bytes", Name{"", "café"}, []Name{{"", "cafe"}}, "; did you mean cafe?"},
		{"insertion", Name{"", "strin"}, []Name{{"", "string"}}, "; did you mean string?"},
		{"deletion", Name{"", "striing"}, []Name{{"", "string"}}, "; did you mean string?"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Hint(tt.missing, tt.candidates, local); got != tt.want {
				t.Errorf("Hint(%v) = %q, want %q", tt.missing, got, tt.want)
			}
		})
	}
}
