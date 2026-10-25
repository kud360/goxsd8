package codegen

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// TestHeaderLoc pins headerLoc against the package doc's "Naming" rule, under
// xsderr.Loc.String's "?", line and column rules.
func TestHeaderLoc(t *testing.T) {
	const base = "/root/gen/x"
	for _, tt := range []struct {
		name string
		loc  xsderr.Loc
		want string
	}{
		{"absolute under base", xsderr.Loc{URI: "/root/gen/x/s/a.xsd", Line: 2, Col: 3}, "s/a.xsd:2:3"},
		{"absolute outside base", xsderr.Loc{URI: "/root/schemas/a.xsd", Line: 2, Col: 3}, "../../schemas/a.xsd:2:3"},
		{"absolute equal to base", xsderr.Loc{URI: "/root/gen/x", Line: 1, Col: 1}, ".:1:1"},
		{"absolute unclean", xsderr.Loc{URI: "/root/gen/./x/../y/a.xsd", Line: 4, Col: 5}, "../y/a.xsd:4:5"},
		{"relative as is", xsderr.Loc{URI: "testdata/expanded.xsd", Line: 2, Col: 3}, "testdata/expanded.xsd:2:3"},
		{"climbing relative as is", xsderr.Loc{URI: "../a.xsd", Line: 2, Col: 3}, "../a.xsd:2:3"},
		{"URL as is", xsderr.Loc{URI: "http://example.com/a.xsd", Line: 2, Col: 3}, "http://example.com/a.xsd:2:3"},
		{"empty URI", xsderr.Loc{Line: 2, Col: 3}, "?:2:3"},
		{"line without column", xsderr.Loc{URI: "/root/gen/x/a.xsd", Line: 2}, "a.xsd:2"},
		{"column without line", xsderr.Loc{URI: "/root/gen/x/a.xsd", Col: 3}, "a.xsd"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := headerLoc(base, tt.loc); got != tt.want {
				t.Errorf("headerLoc(%q, %+v) = %q, want %q", base, tt.loc, got, tt.want)
			}
		})
	}
}
