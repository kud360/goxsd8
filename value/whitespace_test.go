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

	if got, err := effectiveWhiteSpace(noSchema{}, preservePrim); got != preserveWS || err != nil {
		t.Errorf("effectiveWhiteSpace(noSchema{}, string primitive) = (%d, %v), want (preserve %d, nil)", got, err, preserveWS)
	}
	if got, err := effectiveWhiteSpace(noSchema{}, collapsePrim); got != collapseWS || err != nil {
		t.Errorf("effectiveWhiteSpace(noSchema{}, decimal primitive) = (%d, %v), want (collapse %d, nil)", got, err, collapseWS)
	}

	// A derivation with no own whiteSpace facet inherits the primitive's entry.
	derived, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "d"},
		xsd.RestrictionDerivation{}, collapsePrim, nil, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	if got, err := effectiveWhiteSpace(noSchema{}, derived); got != collapseWS || err != nil {
		t.Errorf("effectiveWhiteSpace(noSchema{}, inheriting derivation) = (%d, %v), want (collapse %d, nil)", got, err, collapseWS)
	}
}

// TestEffectiveWhiteSpaceOverride confirms a legal more-derived whiteSpace facet
// supersedes the inherited one under the ordinary same-kind replace overlay
// (§3.16.6.4) — the correctness upgrade over a primitive-only side-table lookup.
func TestEffectiveWhiteSpaceOverride(t *testing.T) {
	stringPrim := primType(t, "string", "preserve") // primitive says preserve
	// A derived step re-declares whiteSpace=collapse; the overlay surfaces it.
	collapsed, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "token-like"},
		xsd.RestrictionDerivation{}, stringPrim,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, false)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType: %v", err)
	}
	if got, err := effectiveWhiteSpace(noSchema{}, collapsed); got != collapseWS || err != nil {
		t.Errorf("effectiveWhiteSpace(noSchema{}, overriding derivation) = (%d, %v), want (collapse %d, nil) (§3.16.6.4)", got, err, collapseWS)
	}
}

// TestEffectiveWhiteSpaceNoFacetErrors confirms an ATOMIC type with no whiteSpace
// facet in force is a caught construction fault (§3.16.7.4 guarantees every atomic
// primitive's {facets} carries a whiteSpace entry) — a returned, discriminable
// component-invariant error, not a silent default and NOT the "no facets applicable"
// (0, nil) outcome. This is the guard the no-facets-applicable widening must not
// weaken: if effectiveWhiteSpace answered (0, nil) for every absent case regardless
// of {variety}, this test would fail.
//
// A primitive built by NewPrimitiveType has a PRESENT {primitive type definition}
// (itself, §3.16.1), which is what keeps it out of the xs:anyAtomicType arm.
func TestEffectiveWhiteSpaceNoFacetErrors(t *testing.T) {
	bare, err := xsd.NewPrimitiveType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "bare"}, nil, nil)
	if err != nil {
		t.Fatalf("NewPrimitiveType: %v", err)
	}
	ws, err := effectiveWhiteSpace(noSchema{}, bare)
	if err == nil {
		t.Fatalf("effectiveWhiteSpace(noSchema{}, atomic, no whiteSpace facet) = (%d, nil), want a component-invariant error", ws)
	}
	if ws != whiteSpace(0) {
		t.Errorf("effectiveWhiteSpace(noSchema{}, atomic, no whiteSpace facet) ws = %d, want zero value 0", ws)
	}
	if r, _ := xsderr.RuleOf(err); r != xsderr.RuleComponentInvariant {
		t.Errorf("effectiveWhiteSpace(noSchema{}, atomic, no whiteSpace facet) charged %s, want %s (§4.3.6.3: no Validation Rules are associated with whiteSpace)", r, xsderr.RuleComponentInvariant)
	}
	if !IsFacetPrecondition(err) {
		t.Error("effectiveWhiteSpace(noSchema{}, atomic, no whiteSpace facet): IsFacetPrecondition = false, want true — callers cannot tell the fault from a verdict")
	}
}

// TestEffectiveWhiteSpaceListNoUsableModeErrors confirms the no-usable-mode fault is
// LIST as well as atomic: a list-variety type with no whiteSpace mode in force
// is still a construction fault, never the "no facets applicable" outcome. This is the
// mutation guard proving the relaxation is the three §4.1.5 cases ONLY, not "any
// variety that is not a primitive-bearing atomic": a blanket "not atomic-with-a-
// primitive ⇒ (0, nil)" would silently pass a broken list here.
//
// The list reaches that state through an UNRECOGNIZED {value}, not an absent
// facet: cos-st-restricts clause 2.2.1.2 now rejects a constructed list carrying
// anything but whiteSpace=collapse fixed, and every list restricting one
// inherits that facet, so the absent-facet state is no longer constructible
// through xsd's constructors at all. The malformed-{value} state is, because
// §4.3.6.4's restriction SCC deliberately leaves an out-of-domain {value} to the
// normalization stage here (STYLE E2) — and effectiveWhiteSpace collapses all
// three no-usable-mode states onto this one error, so it guards the same branch.
func TestEffectiveWhiteSpaceListNoUsableModeErrors(t *testing.T) {
	item := primType(t, "string", "preserve")
	constructed, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "goodList"},
		listOf(item), xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(constructed list): %v", err)
	}
	list, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "bareList"},
		listOf(item), constructed,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"not-a-whiteSpace-token"}, false)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list): %v", err)
	}
	ws, err := effectiveWhiteSpace(noSchema{}, list)
	if err == nil {
		t.Fatalf("effectiveWhiteSpace(noSchema{}, list, unrecognized whiteSpace {value}) = (%d, nil), want a component-invariant error", ws)
	}
	if r, _ := xsderr.RuleOf(err); r != xsderr.RuleComponentInvariant {
		t.Errorf("effectiveWhiteSpace(noSchema{}, list, unrecognized whiteSpace {value}) charged %s, want %s", r, xsderr.RuleComponentInvariant)
	}
	if !IsFacetPrecondition(err) {
		t.Error("effectiveWhiteSpace(noSchema{}, list, unrecognized whiteSpace {value}): IsFacetPrecondition = false, want true")
	}
}

// TestEffectiveWhiteSpaceNoFacetsApplicable pins all THREE cos-applicable-facets
// (§4.1.5) no-facets-applicable cases as "not applicable" — (0, nil) — rather than as
// faults: an ABSENT {variety} (xs:anySimpleType), an ATOMIC {variety} with an absent
// {primitive type definition} (xs:anyAtomicType), and a UNION, whose applicable
// facets are pattern/enumeration/assertions with whiteSpace absent.
//
// The two ·specials· are the live-bug half (warden pre-flight on #321): before the
// widening only the union arm was relaxed, so effectiveWhiteSpace faulted on
// xs:anySimpleType and xs:anyAtomicType — reachable from SchemaBuilder and from any
// library caller — and a returned fault there would false-reject the two widest types
// in the language, which §4.1.4 makes unconditionally Datatype Valid.
func TestEffectiveWhiteSpaceNoFacetsApplicable(t *testing.T) {
	cases := []struct {
		name string
		st   *xsd.SimpleType
	}{
		{"anySimpleType ({variety} absent)", xsd.AnySimpleType()},
		{"anyAtomicType (atomic, {primitive type definition} absent)", xsd.AnyAtomicType()},
		{"union (whiteSpace not in its applicable set)", unionType(t)},
	}
	for _, c := range cases {
		ws, err := effectiveWhiteSpace(noSchema{}, c.st)
		if err != nil {
			t.Errorf("effectiveWhiteSpace(noSchema{}, %s) = (%d, %v), want (0, nil) — §4.1.5 makes no facet applicable", c.name, ws, err)
			continue
		}
		if ws != whiteSpace(0) {
			t.Errorf("effectiveWhiteSpace(noSchema{}, %s) ws = %d, want zero value 0", c.name, ws)
		}
	}
}

// TestValidateLexicalSpecialDatatypesDoNotFault drives the two ·special· datatypes
// through the exported pipeline to prove the whiteSpace stage no longer faults on
// them (it runs BEFORE the governing-mapping gate, so they cannot be filtered out
// earlier). The expected outcome is the ordinary "no backend mapping governs"
// cvc-datatype-valid error — no backend maps a ·special· (§4.1) — and NOT a facet
// precondition fault, which valueSpace.ValidDefault would have to answer undecided and
// a naive caller would read as a false reject.
func TestValidateLexicalSpecialDatatypesDoNotFault(t *testing.T) {
	for _, st := range []*xsd.SimpleType{xsd.AnySimpleType(), xsd.AnyAtomicType()} {
		_, err := ValidateLexical(emptyBackend{}, noSchema{}, st, "  raw  literal  ", nil)
		if err == nil {
			t.Errorf("ValidateLexical(%s) = nil error, want the ungoverned cvc-datatype-valid error", st.Name())
			continue
		}
		if IsFacetPrecondition(err) {
			t.Errorf("ValidateLexical(%s) reported a facet precondition fault: %v", st.Name(), err)
		}
		if r, _ := xsderr.RuleOf(err); r != "cvc-datatype-valid" {
			t.Errorf("ValidateLexical(%s) charged %s, want cvc-datatype-valid (no backend mapping governs)", st.Name(), r)
		}
	}
}

// TestEffectiveWhiteSpaceListResolvesCollapse confirms a list-variety type that
// carries its materialized whiteSpace=collapse facet (as builtin.Seed
// materializes NMTOKENS/IDREFS/ENTITIES per §4.3.6.1) resolves through the
// ordinary EffectiveFacets scan to (collapse, nil) — hitting neither the
// no-facets-applicable arm nor the fault, so no list special-casing is needed.
func TestEffectiveWhiteSpaceListResolvesCollapse(t *testing.T) {
	item := primType(t, "string", "preserve")
	list, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "collapseList"},
		listOf(item), xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list): %v", err)
	}
	if got, err := effectiveWhiteSpace(noSchema{}, list); got != collapseWS || err != nil {
		t.Errorf("effectiveWhiteSpace(noSchema{}, list) = (%d, %v), want (collapse %d, nil) (§4.3.6.1)", got, err, collapseWS)
	}
}

// TestValidateLexicalUnionWhiteSpaceStageNoPanic drives a union-variety type
// end-to-end through the exported ValidateLexical pipeline to prove no whiteSpace
// stage runs on a union and nothing panics for want of one (§4.1.5: whiteSpace is
// not applicable to a union). It pins the UNGOVERNED half of the union path —
// correct member-dispatch itself is pinned in union_test.go: with a backend that
// maps none of the members, unionGoverned reports the union ungoverned and
// ValidateLexical returns a real *xsderr cvc-datatype-valid error, which is the
// correct outcome — a returned error, never a panic and never a false
// (Value, nil) accept.
func TestValidateLexicalUnionWhiteSpaceStageNoPanic(t *testing.T) {
	union := unionType(t)
	// emptyBackend maps nothing, so no member of the union is governed and
	// ValidateLexical returns its normal cvc-datatype-valid error.
	v, err := ValidateLexical(emptyBackend{}, noSchema{}, union, "  raw  literal  ", nil)
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
	union, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "u"},
		unionOf(strPrim, decPrim), xsd.AnySimpleType(), nil, nil)
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
