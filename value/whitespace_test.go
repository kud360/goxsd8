package value

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestNormalizeWhiteSpaceModes exercises the three §4.3.6 modes directly on raw
// text carrying tabs, newlines, carriage returns and space runs.
func TestNormalizeWhiteSpaceModes(t *testing.T) {
	const raw = "  a\tb\r\nc  d  "
	cases := []struct {
		mode whiteSpace
		want string
	}{
		{preserveWS, "  a\tb\r\nc  d  "}, // identity
		{replaceWS, "  a b  c  d  "},     // #x9/#xA/#xD → #x20 (\r\n → two spaces), no collapse/trim
		{collapseWS, "a b c d"},          // replace, then collapse runs and trim ends
	}
	for _, c := range cases {
		if got := normalizeWhiteSpace(raw, c.mode); got != c.want {
			t.Errorf("normalizeWhiteSpace(%q, %d) = %q, want %q", raw, c.mode, got, c.want)
		}
	}
}

// TestCollapseKeepsNonAsciiSpace confirms collapse touches only #x20:
// a non-breaking space (U+00A0) is not collapsed or trimmed.
func TestCollapseKeepsNonAsciiSpace(t *testing.T) {
	const nbsp = " "
	got := normalizeWhiteSpace(nbsp+"a"+nbsp, collapseWS)
	if got != nbsp+"a"+nbsp {
		t.Errorf("collapse altered non-#x20 whitespace: got %q", got)
	}
}

// TestNormalizeWhiteSpaceInvalidModePanics confirms the zero-value sentinel is a
// caught bug, not a silent identity.
func TestNormalizeWhiteSpaceInvalidModePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("normalizeWhiteSpace(zero mode): want panic, got none")
		}
	}()
	_ = normalizeWhiteSpace("x", whiteSpace(0))
}

// primType builds a primitive named local carrying a single whiteSpace facet of
// value ws, mirroring how builtin.Seed materializes the primitive's own
// {facets} whiteSpace entry (§3.16.7.4) so effectiveWhiteSpace resolves it off
// EffectiveFacets.
func primType(t *testing.T, local, ws string) *xsd.SimpleType {
	t.Helper()
	p, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: local},
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{ws}, ws != "preserve")}, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType(%q): %v", local, err)
	}
	return p
}

// TestEffectiveWhiteSpace resolves the mode off EffectiveFacets for a primitive
// and for a derivation that inherits the primitive's whiteSpace facet without
// re-declaring it (§3.16.6.4 overlay surfaces the inherited entry).
func TestEffectiveWhiteSpace(t *testing.T) {
	preservePrim := primType(t, "string", "preserve")
	collapsePrim := primType(t, "decimal", "collapse")

	if got := effectiveWhiteSpace(preservePrim); got != preserveWS {
		t.Errorf("effectiveWhiteSpace(string primitive) = %d, want preserve %d", got, preserveWS)
	}
	if got := effectiveWhiteSpace(collapsePrim); got != collapseWS {
		t.Errorf("effectiveWhiteSpace(decimal primitive) = %d, want collapse %d", got, collapseWS)
	}

	// A derivation with no own whiteSpace facet inherits the primitive's entry.
	derived, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "d"},
		xsd.Atomic{Primitive: collapsePrim}, collapsePrim, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	if got := effectiveWhiteSpace(derived); got != collapseWS {
		t.Errorf("effectiveWhiteSpace(inheriting derivation) = %d, want collapse %d", got, collapseWS)
	}
}

// TestEffectiveWhiteSpaceOverride confirms a legal more-derived whiteSpace facet
// supersedes the inherited one under the ordinary same-kind replace overlay
// (§3.16.6.4) — the correctness upgrade over a primitive-only side-table lookup.
func TestEffectiveWhiteSpaceOverride(t *testing.T) {
	stringPrim := primType(t, "string", "preserve") // primitive says preserve
	// A derived step re-declares whiteSpace=collapse; the overlay surfaces it.
	collapsed, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "token-like"},
		xsd.Atomic{Primitive: stringPrim}, stringPrim,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, false)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	if got := effectiveWhiteSpace(collapsed); got != collapseWS {
		t.Errorf("effectiveWhiteSpace(overriding derivation) = %d, want collapse %d (§3.16.6.4)", got, collapseWS)
	}
}

// TestEffectiveWhiteSpaceNoFacetPanics confirms a type with no whiteSpace facet
// in force (outside the atomic cohort ValidateLexical's precondition scopes) is
// a caught programming error, not a silent default.
func TestEffectiveWhiteSpaceNoFacetPanics(t *testing.T) {
	bare, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "bare"}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("effectiveWhiteSpace(no whiteSpace facet): want panic, got none")
		}
	}()
	_ = effectiveWhiteSpace(bare)
}
