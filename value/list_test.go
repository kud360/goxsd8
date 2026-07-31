package value

import (
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// itemStub is a test-only atomic item value carrying the §2.2.1/§2.2.2 identity
// and equality relations, so a listValue built from itemStubs exercises the
// pairwise comparison listValue.Identical/Eq perform. n distinguishes members.
type itemStub struct{ n int }

func (i itemStub) Eq(other Value) bool {
	o, ok := other.(itemStub)
	return ok && o.n == i.n
}

// Identical coincides with Eq here: itemStub has no NaN/signed-zero carve-out,
// so identity is exactly equality (the boolVal convention).
func (i itemStub) Identical(other Value) bool { return i.Eq(other) }

// bareItem is a test-only item value with NEITHER the Eq nor the Identical
// capability, proving listValue.Identical/Eq return false — a normal "no match"
// outcome — rather than panicking when an item cannot be compared.
type bareItem struct{}

// stubItemBackend maps exactly one item QName to a Mapping that parses a token
// to an itemStub keyed by token length, and rejects the sentinel token "bad"
// with a cvc-datatype-valid error (so an invalid item token's Parse error can be
// observed propagating through listMapping and ValidateLexical). It stands in
// for a real atomic backend, matching this package's mock idiom (emptyBackend,
// scaledStub) while keeping value's tests free of the builtin/strict import
// cycle (strict imports value).
type stubItemBackend struct{ item xsd.QName }

func (b stubItemBackend) Mapping(typ xsd.QName) (Mapping, bool) {
	if typ != b.item {
		return Mapping{}, false
	}
	return Mapping{Parse: func(lexical string, _ Context) (Value, error) {
		if lexical == "bad" {
			return nil, xsderr.New("cvc-datatype-valid", xsderr.Loc{},
				"item: %q is not in the lexical space", lexical)
		}
		return itemStub{n: len(lexical)}, nil
	}}, true
}

// listType builds a list-variety *xsd.SimpleType over item (restricting
// xs:anySimpleType), carrying own facets — mirroring how the conformance list
// cohort synthesizes its leaf and how whitespace_test's list helpers build one.
func listType(t *testing.T, item *xsd.SimpleType, own []xsd.Facet) *xsd.SimpleType {
	t.Helper()
	lst, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "lst"},
		xsd.NewList(item), xsd.AnySimpleType(), own, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(list): %v", err)
	}
	return lst
}

// TestGoverningMappingListResolves confirms governingMapping wraps the item
// type's mapping in a listMapping for a list-variety type (cvc-datatype-valid
// clause dv_list, §4.1.4 cl.2.2), so Parse splits the lexical into items and the
// resulting value's Lengthed.Len counts them (§4.3.1.3). An item type with no
// governing mapping leaves the list ungoverned, the atomic path's outcome.
func TestGoverningMappingListResolves(t *testing.T) {
	item := primType(t, "myitem", "collapse")
	lst := listType(t, item, nil)

	m, ok := governingMapping(stubItemBackend{item: item.Name()}, lst)
	if !ok {
		t.Fatal("governingMapping(list) ok = false, want a resolved listMapping")
	}
	v, err := m.Parse("true false", nil)
	if err != nil {
		t.Fatalf("listMapping.Parse(%q): %v", "true false", err)
	}
	lv, ok := v.(Lengthed)
	if !ok {
		t.Fatalf("list value %T does not implement Lengthed", v)
	}
	if lv.Len() != 2 {
		t.Errorf("list value Len = %d, want 2 (measured in list items, §4.3.1.3)", lv.Len())
	}

	// An item type the backend does not map leaves the list ungoverned.
	if _, ok := governingMapping(emptyBackend{}, lst); ok {
		t.Error("governingMapping(list, ungoverned item) ok = true, want false")
	}
}

// TestListValueIdentityEquality exercises listValue's §2.2.1 identity and §2.2.2
// equality relations: same items in the same order match (both relations); a
// different length or a different item value does not; and an item lacking the
// comparison capability yields a non-match rather than a panic.
func TestListValueIdentityEquality(t *testing.T) {
	ab := listValue{items: []Value{itemStub{1}, itemStub{2}}}
	abAgain := listValue{items: []Value{itemStub{1}, itemStub{2}}}
	abc := listValue{items: []Value{itemStub{1}, itemStub{2}, itemStub{3}}}
	axb := listValue{items: []Value{itemStub{1}, itemStub{9}}}

	if !ab.Identical(abAgain) || !ab.Eq(abAgain) {
		t.Error("equal-token lists: want Identical and Eq true")
	}
	if ab.Identical(abc) || ab.Eq(abc) {
		t.Error("different-length lists: want Identical and Eq false (§2.2.1/§2.2.2 require equal length)")
	}
	if ab.Identical(axb) || ab.Eq(axb) {
		t.Error("different-item lists: want Identical and Eq false")
	}

	// A cross-type argument (not a listValue) never matches.
	if ab.Identical(itemStub{1}) || ab.Eq(itemStub{1}) {
		t.Error("listValue vs non-list: want Identical and Eq false")
	}

	// An item without the Identical/Eq capability is a non-match, not a panic.
	bare := listValue{items: []Value{bareItem{}}}
	bareAgain := listValue{items: []Value{bareItem{}}}
	if bare.Identical(bareAgain) || bare.Eq(bareAgain) {
		t.Error("list of capability-less items: want Identical and Eq false (no-match convention)")
	}
}

// TestValidateLexicalListItemErrorPropagates drives a list-variety leaf through
// the full ValidateLexical pipeline (whiteSpace collapse → listMapping) and
// confirms an invalid item token's Parse error surfaces unchanged (dv_list
// clause 2.2: each item is itself Datatype-Valid, so its own
// cvc-datatype-valid-family error is the right one). A list of valid tokens
// validates and yields a listValue.
func TestValidateLexicalListItemErrorPropagates(t *testing.T) {
	item := primType(t, "myitem", "collapse")
	// The list's mandatory fixed whiteSpace=collapse facet (§4.3.6.1 f-w-fixed):
	// effectiveWhiteSpace panics on a list carrying none.
	own := []xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}
	leaf := listType(t, item, own)
	b := stubItemBackend{item: item.Name()}

	v, err := ValidateLexical(b, leaf, "aa bb ccc", nil)
	if err != nil {
		t.Fatalf("ValidateLexical(valid list) = %v, want accept", err)
	}
	if _, ok := v.(Lengthed); !ok {
		t.Fatalf("ValidateLexical(valid list) value %T does not implement Lengthed", v)
	}

	_, err = ValidateLexical(b, leaf, "aa bad", nil)
	if err == nil {
		t.Fatal("ValidateLexical(list with invalid item token) = nil, want the item's Parse error")
	}
	if r, _ := xsderr.RuleOf(err); r != "cvc-datatype-valid" {
		t.Errorf("ValidateLexical(invalid item) charged %s, want cvc-datatype-valid (dv_list item propagation)", r)
	}
}

// TestValidateLexicalListItemTypeFacetsApply is the regression guard for issue
// #224: dv_list (§4.1.4 cl.2.2) says each item is "Datatype Valid with respect
// to the {item type definition}", which is the WHOLE cvc-datatype-valid rule
// against that type — clause 3 (dv_vfacets, the item type's own value-based
// facets) included, not just clause 2.1's lexical mapping. That distinction is
// invisible when the item type IS its own {primitive type definition}, and
// decisive when it is not: xs:byte maps through xs:decimal, so mapping-only item
// parsing accepts "128" and "-129" that byte's own minInclusive/maxInclusive
// reject.
//
// The item type here is a DERIVED type (a restriction of the mapped primitive)
// carrying an enumeration the primitive does not have, mirroring that shape
// without importing builtin/strict (which imports value). A list of accepted
// items validates; one item outside the item type's enumeration rejects with
// cvc-enumeration-valid — the item type's OWN facet rule, charged per item.
func TestValidateLexicalListItemTypeFacetsApply(t *testing.T) {
	prim := primType(t, "myitem", "collapse")
	b := stubItemBackend{item: prim.Name()}
	// stubItemBackend keys its value by token length, so "aa" and "bb" share a
	// value and "ccc" does not: the enumeration admits the two-character items
	// and excludes the three-character one.
	item, err := xsd.NewSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "enumitem"},
		prim.Variety(), prim, []xsd.Facet{enumOf("aa")}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(item restriction): %v", err)
	}
	own := []xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}
	leaf := listType(t, item, own)

	if _, err := ValidateLexical(b, leaf, "aa bb", nil); err != nil {
		t.Fatalf("ValidateLexical(list of enumerated items) = %v, want accept", err)
	}

	_, err = ValidateLexical(b, leaf, "aa ccc", nil)
	if err == nil {
		t.Fatal("ValidateLexical(list with out-of-enumeration item) = nil, want the item type's own facet rejection (dv_list → dv_vfacets)")
	}
	if r, _ := xsderr.RuleOf(err); r != "cvc-enumeration-valid" {
		t.Errorf("out-of-enumeration item charged %s, want cvc-enumeration-valid (§4.3.5.4 on the ITEM type, not the list)", r)
	}
}
