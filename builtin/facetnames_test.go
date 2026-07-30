package builtin

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

// TestFacetNamesBijection pins the consolidated bridge table (restriction.go's
// facetNames): both lookup directions agree by construction, and the table covers
// the WHOLE closed xsd.FacetKind set — including maxScale/minScale, which the old
// hand-typed inverse switch in seed.go silently omitted. A future FacetKind added
// without a facetNames row fails here instead of turning into a silent lookup
// miss at schema construction.
func TestFacetNamesBijection(t *testing.T) {
	for _, e := range facetNames {
		name, ok := facetName(e.kind)
		if !ok || name != e.name {
			t.Errorf("facetName(%s) = (%q, %v), want (%q, true)", e.kind, name, ok, e.name)
		}
		kind, ok := facetKind(e.name)
		if !ok || kind != e.kind {
			t.Errorf("facetKind(%q) = (%s, %v), want (%s, true)", e.name, kind, ok, e.kind)
		}
	}
	for k := xsd.FacetLength; k <= xsd.FacetMinScale; k++ {
		if _, ok := facetName(k); !ok {
			t.Errorf("facetNames has no row for FacetKind %s", k)
		}
	}
	if got := len(facetNames); got != int(xsd.FacetMinScale-xsd.FacetLength)+1 {
		t.Errorf("facetNames has %d rows, want one per FacetKind (%d)", got, int(xsd.FacetMinScale-xsd.FacetLength)+1)
	}
}
