package schemaloc

import "testing"

func TestResolve(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		location string
		want     string
	}{
		// An empty location is the bare <import> case: nothing to resolve, so the
		// base itself comes back and the caller decides what that means.
		{"empty location, absolute base", "http://example.com/a/main.xsd", "", "http://example.com/a/main.xsd"},
		{"empty location, relative base", "schemas/main.xsd", "", "schemas/main.xsd"},
		{"empty location, empty base", "", "", ""},

		// An absolute reference wins outright, whatever the base is.
		{"absolute location over absolute base", "http://example.com/a/main.xsd", "http://other.example/x.xsd", "http://other.example/x.xsd"},
		{"absolute location over relative base", "schemas/main.xsd", "http://other.example/x.xsd", "http://other.example/x.xsd"},
		{"absolute location over empty base", "", "file:///srv/x.xsd", "file:///srv/x.xsd"},

		// Absolute base: RFC 3986 reference resolution.
		{"sibling against absolute base", "http://example.com/a/main.xsd", "child.xsd", "http://example.com/a/child.xsd"},
		{"subdirectory against absolute base", "http://example.com/a/main.xsd", "sub/child.xsd", "http://example.com/a/sub/child.xsd"},
		{"parent against absolute base", "http://example.com/a/b/main.xsd", "../child.xsd", "http://example.com/a/child.xsd"},
		{"rooted against absolute base", "http://example.com/a/b/main.xsd", "/child.xsd", "http://example.com/child.xsd"},
		{"query is carried through", "http://example.com/a/main.xsd", "child.xsd?v=1", "http://example.com/a/child.xsd?v=1"},

		// Non-absolute base: path-wise resolution, so a resolver-relative location
		// stays resolver-relative instead of being rooted at "/".
		{"sibling against relative base", "schemas/main.xsd", "child.xsd", "schemas/child.xsd"},
		{"subdirectory against relative base", "schemas/main.xsd", "sub/child.xsd", "schemas/sub/child.xsd"},
		{"parent against relative base", "schemas/sub/main.xsd", "../child.xsd", "schemas/child.xsd"},
		{"bare base has no directory", "main.xsd", "child.xsd", "child.xsd"},
		{"empty base", "", "child.xsd", "child.xsd"},
		{"rooted location against relative base is cleaned, not joined", "schemas/main.xsd", "/abs/../child.xsd", "/child.xsd"},

		// A location or base that is not a parsable URI reference cannot take the
		// absolute or RFC 3986 branch; it falls through to path-wise resolution
		// rather than being dropped, which collapses an authority "//" the resolver
		// is then handed as-is.
		{"unparsable location falls through to path join", "http://example.com/a/main.xsd", "%zz.xsd", "http:/example.com/a/%zz.xsd"},
		{"unparsable base falls through to path join", "http://%zz/main.xsd", "child.xsd", "http:/%zz/child.xsd"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Resolve(tc.base, tc.location); got != tc.want {
				t.Fatalf("Resolve(%q, %q) = %q, want %q", tc.base, tc.location, got, tc.want)
			}
		})
	}
}
