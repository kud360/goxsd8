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

	if got, ok := effectiveWhiteSpace(preservePrim); got != preserveWS || !ok {
		t.Errorf("effectiveWhiteSpace(string primitive) = (%d, %v), want (preserve %d, true)", got, ok, preserveWS)
	}
	if got, ok := effectiveWhiteSpace(collapsePrim); got != collapseWS || !ok {
		t.Errorf("effectiveWhiteSpace(decimal primitive) = (%d, %v), want (collapse %d, true)", got, ok, collapseWS)
	}

	// A derivation with no own whiteSpace facet inherits the primitive's entry.
	derived, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "d"},
		xsd.Atomic{Primitive: collapsePrim}, collapsePrim, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	if got, ok := effectiveWhiteSpace(derived); got != collapseWS || !ok {
		t.Errorf("effectiveWhiteSpace(inheriting derivation) = (%d, %v), want (collapse %d, true)", got, ok, collapseWS)
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
	if got, ok := effectiveWhiteSpace(collapsed); got != collapseWS || !ok {
		t.Errorf("effectiveWhiteSpace(overriding derivation) = (%d, %v), want (collapse %d, true) (§3.16.6.4)", got, ok, collapseWS)
	}
}

// TestEffectiveWhiteSpaceNoFacetPanics confirms an ATOMIC type with no
// whiteSpace facet in force is a caught construction bug (§3.16.7.4 guarantees
// every atomic primitive's {facets} carries a whiteSpace entry), not a silent
// default and NOT the relaxed union "(0, false)" outcome. This is the guard the
// union widening must not weaken: if effectiveWhiteSpace returned (0, false) for
// every absent case regardless of variety, this test would fail.
func TestEffectiveWhiteSpaceNoFacetPanics(t *testing.T) {
	bare, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "bare"}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("effectiveWhiteSpace(atomic, no whiteSpace facet): want panic, got none")
		}
	}()
	_, _ = effectiveWhiteSpace(bare)
}

// TestEffectiveWhiteSpaceListNoFacetPanics confirms the absent-facet panic is
// LIST as well as atomic: a list-variety type that (contrary to §4.3.6.1) lacks
// its materialized whiteSpace=collapse facet is still a construction bug, never
// the relaxed union outcome. This is the mutation guard proving the relaxation
// is union-ONLY, not "any non-atomic variety": a blanket
// "non-atomic ⇒ (0, false)" would silently pass a broken list here.
func TestEffectiveWhiteSpaceListNoFacetPanics(t *testing.T) {
	item := primType(t, "string", "preserve")
	// A list with no own whiteSpace facet: EffectiveFacets surfaces none.
	list, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "bareList"},
		xsd.List{Item: item}, xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list): %v", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("effectiveWhiteSpace(list, no whiteSpace facet): want panic, got none")
		}
	}()
	_, _ = effectiveWhiteSpace(list)
}

// TestEffectiveWhiteSpaceUnionNotApplicable confirms the ONE relaxed case: a
// union {variety} carries no whiteSpace facet (cos-applicable-facets §4.1.5,
// whiteSpace is not applicable to union), so effectiveWhiteSpace answers
// (0, false) — "not applicable" — rather than panicking.
func TestEffectiveWhiteSpaceUnionNotApplicable(t *testing.T) {
	union := unionType(t)
	got, applicable := effectiveWhiteSpace(union)
	if applicable {
		t.Errorf("effectiveWhiteSpace(union) applicable = true, want false (cos-applicable-facets §4.1.5)")
	}
	if got != whiteSpace(0) {
		t.Errorf("effectiveWhiteSpace(union) ws = %d, want zero value 0", got)
	}
}

// TestEffectiveWhiteSpaceListResolvesCollapse confirms a list-variety type that
// carries its materialized whiteSpace=collapse facet (as builtin.Seed
// materializes NMTOKENS/IDREFS/ENTITIES per §4.3.6.1) resolves through the
// ordinary EffectiveFacets scan to (collapse, true) — hitting neither the union
// branch nor the panic branch, so no list special-casing is needed.
func TestEffectiveWhiteSpaceListResolvesCollapse(t *testing.T) {
	item := primType(t, "string", "preserve")
	list, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "collapseList"},
		xsd.List{Item: item}, xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list): %v", err)
	}
	if got, ok := effectiveWhiteSpace(list); got != collapseWS || !ok {
		t.Errorf("effectiveWhiteSpace(list) = (%d, %v), want (collapse %d, true) (§4.3.6.1)", got, ok, collapseWS)
	}
}

// TestValidateLexicalUnionWhiteSpaceStageNoPanic drives a union-variety type
// end-to-end through the exported ValidateLexical pipeline to prove the
// whiteSpace stage no longer panics on a union. This does NOT assert correct
// union validation (member-dispatch is out of scope, §4.1.4 cl.2.3): with no
// governing mapping the pipeline errors out at governingMapping with a real
// *xsderr cvc-datatype-valid error — which is the correct outcome, a returned
// error, never a panic and never a false (Value, nil) accept.
func TestValidateLexicalUnionWhiteSpaceStageNoPanic(t *testing.T) {
	union := unionType(t)
	// emptyBackend maps nothing, so governingMapping finds no mapping for the
	// union and ValidateLexical returns its normal cvc-datatype-valid error.
	v, err := ValidateLexical(emptyBackend{}, union, "  raw  literal  ", nil)
	if err == nil {
		t.Fatalf("ValidateLexical(union) = (%v, nil), want a real error (no governing mapping)", v)
	}
	if v != nil {
		t.Errorf("ValidateLexical(union) value = %v, want nil on error", v)
	}
}

// unionType builds a union-variety *xsd.SimpleType over two atomic primitive
// members via the #46-hardened public constructors (constructed from
// xs:anySimpleType), for the union-path whiteSpace tests.
func unionType(t *testing.T) *xsd.SimpleType {
	t.Helper()
	strPrim := primType(t, "string", "preserve")
	decPrim := primType(t, "decimal", "collapse")
	union, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "u"},
		xsd.Union{Members: []*xsd.SimpleType{strPrim, decPrim}}, xsd.AnySimpleType(), nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(union): %v", err)
	}
	return union
}

// emptyBackend is a value.Backend that maps no type, so governingMapping never
// resolves a mapping. It lets a ValidateLexical test drive the pipeline far
// enough to exercise the whiteSpace stage without needing a real value mapping.
type emptyBackend struct{}

func (emptyBackend) Mapping(xsd.QName) (Mapping, bool) { return Mapping{}, false }
