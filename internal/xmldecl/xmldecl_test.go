package xmldecl

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestAs10(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"1.1 double-quoted", `<?xml version="1.1"?><a/>`, `<?xml version="1.0"?><a/>`},
		{"1.1 single-quoted", `<?xml version='1.1' encoding='UTF-8'?><a/>`, `<?xml version='1.0' encoding='UTF-8'?><a/>`},
		{"S and Eq spacing", "<?xml\r\n version\t=\n'1.9'?>", "<?xml\r\n version\t=\n'1.0'?>"},
		{"two digits before S", `<?xml version="1.10" standalone="no"?>`, `<?xml version="1.0"  standalone="no"?>`},
		{"three digits before ?>", `<?xml version='1.234'?>`, `<?xml version='1.0'  ?>`},

		// Left as they are.
		{"1.0", `<?xml version="1.0"?><a/>`, `<?xml version="1.0"?><a/>`},
		{"no declaration", `<a v="1.1"/>`, `<a v="1.1"/>`},
		{"empty", ``, ``},
		{"other PI", `<?xml-stylesheet version="1.1"?>`, `<?xml-stylesheet version="1.1"?>`},
		{"declaration not first", ` <?xml version="1.1"?>`, ` <?xml version="1.1"?>`},
		{"no version first", `<?xml encoding="UTF-8" version="1.1"?>`, `<?xml encoding="UTF-8" version="1.1"?>`},
		{"major version 2", `<?xml version="2.1"?>`, `<?xml version="2.1"?>`},
		{"no minor digits", `<?xml version="1."?>`, `<?xml version="1."?>`},
		{"non-digit minor", `<?xml version="1.1a"?>`, `<?xml version="1.1a"?>`},
		{"mismatched quotes", `<?xml version="1.1'?>`, `<?xml version="1.1'?>`},
		{"two digits, no S after", `<?xml version="1.10"encoding="UTF-8"?>`, `<?xml version="1.10"encoding="UTF-8"?>`},
		{"truncated in digits", `<?xml version="1.1`, `<?xml version="1.1`},
		{"truncated after two digits", `<?xml version="1.10"`, `<?xml version="1.10"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// OneByteReader also proves the read-ahead is served across calls.
			got, err := io.ReadAll(iotest.OneByteReader(As10(strings.NewReader(tc.in))))
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if string(got) != tc.want {
				t.Errorf("As10(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestAs10ReportsSourceFailure pins that a failure ending the read-ahead is
// reported after the bytes read before it, and not swallowed.
func TestAs10ReportsSourceFailure(t *testing.T) {
	boom := errors.New("boom")
	src := io.MultiReader(strings.NewReader(`<?xml vers`), iotest.ErrReader(boom))
	got, err := io.ReadAll(As10(src))
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, want %v", err, boom)
	}
	if string(got) != `<?xml vers` {
		t.Errorf("read %q before the failure, want %q", got, `<?xml vers`)
	}
}
