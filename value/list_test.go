package value

import (
	"strings"
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

// listType builds a CONSTRUCTED list-variety *xsd.SimpleType over item
// (restricting xs:anySimpleType) — mirroring how the conformance list cohort
// synthesizes its constructed step and how whitespace_test's list helpers build
// one. Its {facets} is the one set cos-st-restricts clause 2.2.1.2 admits there:
// the fixed whiteSpace=collapse facet §3.16.2.1 map.std.common case 3
// manufactures for every <list>, which is also the facet effectiveWhiteSpace
// needs in force (§4.3.6.1 f-w-fixed). A list carrying anything else is a
// SECOND derivation step and is built by the test that needs one.
func listType(t *testing.T, item *xsd.SimpleType) *xsd.SimpleType {
	t.Helper()
	lst, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "lst"},
		listOf(item), xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
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
	lst := listType(t, item)

	m, ok, gerr := governingMapping(stubItemBackend{item: item.Name()}, noSchema{}, lst)
	if gerr != nil {
		t.Fatalf("governingMapping(list): %v", gerr)
	}
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

	// An item type the backend does not map leaves the list ungoverned — the
	// listGoverned half of the branch, pinned through governingMapping exactly as
	// TestGoverningMappingUnionRequiresEveryMember pins unionGoverned.
	if _, ok, gerr := governingMapping(emptyBackend{}, noSchema{}, lst); gerr != nil || ok {
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
	leaf := listType(t, item)
	b := stubItemBackend{item: item.Name()}

	v, err := ValidateLexical(b, noSchema{}, leaf, "aa bb ccc", nil)
	if err != nil {
		t.Fatalf("ValidateLexical(valid list) = %v, want accept", err)
	}
	if _, ok := v.(Lengthed); !ok {
		t.Fatalf("ValidateLexical(valid list) value %T does not implement Lengthed", v)
	}

	_, err = ValidateLexical(b, noSchema{}, leaf, "aa bad", nil)
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
	item, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "enumitem"},
		xsd.RestrictionDerivation{}, prim, []xsd.Facet{enumOf("aa")}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(item restriction): %v", err)
	}
	leaf := listType(t, item)

	if _, err := ValidateLexical(b, noSchema{}, leaf, "aa bb", nil); err != nil {
		t.Fatalf("ValidateLexical(list of enumerated items) = %v, want accept", err)
	}

	_, err = ValidateLexical(b, noSchema{}, leaf, "aa ccc", nil)
	if err == nil {
		t.Fatal("ValidateLexical(list with out-of-enumeration item) = nil, want the item type's own facet rejection (dv_list → dv_vfacets)")
	}
	if r, _ := xsderr.RuleOf(err); r != "cvc-enumeration-valid" {
		t.Errorf("out-of-enumeration item charged %s, want cvc-enumeration-valid (§4.3.5.4 on the ITEM type, not the list)", r)
	}
}

// TestValidateLexicalListUnionItemTypeOwnFacetsApply pins the second list-item
// shape listMapping's per-item recursion decides (issue #326): an item type of
// UNION {variety}. dv_list (§4.1.4 cl.2.2) says each token is "Datatype Valid
// with respect to the {item type definition}", and that is the WHOLE
// cvc-datatype-valid conjunction against the union — clause 1 (dv_pattern, the
// union's OWN pattern), clause 2.3 (dv_union, the member dispatch) AND clause 3
// (dv_vfacets, the union's OWN enumeration), layered on top of the dispatch
// rather than instead of it. pattern and enumeration are, with assertions,
// exactly the facets cos-applicable-facets (§4.1.5) makes applicable to a union,
// so they are the whole of what clauses 1 and 3 can contribute at the item level.
//
// Both item types below carry a facet NO MEMBER carries, so each rejected token
// is one the member dispatch alone ACCEPTS: parsing tokens through the item
// type's governing mapping only (unionMapping, which is dispatch and nothing
// else) would false-accept every one of them, and the test would fail. The list
// types carry no pattern or enumeration of their own, so a cvc-pattern-valid or
// cvc-enumeration-valid rejection here can only be the item type's — and the
// pattern message names the TOKEN, not the whole lexical, which is what charges
// it per item.
//
// std-item_type_definition (Structures §3.16.1) keeps this one level deep: the
// union item type has no list among its basic members, so the recursion bottoms
// out in the atomic members below.
func TestValidateLexicalListUnionItemTypeOwnFacetsApply(t *testing.T) {
	num := primType(t, "numeric", "collapse")
	text := primType(t, "text", "preserve")
	b := memberBackend{
		num.Name():  allDigits,
		text.Name(): func(string) bool { return true },
	}
	base := unionType2(t, "itemBase", num, text)

	// clause 3 (dv_vfacets): the union item type's own enumeration. "8" is
	// accepted by the numeric member — the dispatch succeeds — and rejected only
	// by the enumeration the union itself declares.
	enumItem := unionRestriction(t, "enumItem", base, []xsd.Facet{enumOf("7")})
	enumList := listType(t, enumItem)
	if _, err := ValidateLexical(b, noSchema{}, enumList, "7 7", nil); err != nil {
		t.Fatalf("ValidateLexical(list of enumerated union items, %q) = %v, want accept", "7 7", err)
	}
	_, err := ValidateLexical(b, noSchema{}, enumList, "7 8", nil)
	if err == nil {
		t.Fatal("ValidateLexical(list, noSchema{}, token outside the union item type's enumeration) = nil, want the item type's own clause-3 rejection (dv_list → dv_vfacets)")
	}
	if r, _ := xsderr.RuleOf(err); r != "cvc-enumeration-valid" {
		t.Errorf("out-of-enumeration union item charged %s, want cvc-enumeration-valid (§4.3.5.4 on the union ITEM type)", r)
	}

	// clause 1 (dv_pattern): the union item type's own pattern. "abc" is accepted
	// by the text member — the dispatch succeeds — and rejected only by the
	// pattern the union itself declares.
	patItem := unionRestriction(t, "patItem", base,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetPattern, []string{"[0-9]+"}, false)})
	patList := listType(t, patItem)
	if _, err := ValidateLexical(b, noSchema{}, patList, "7 42", nil); err != nil {
		t.Fatalf("ValidateLexical(list of pattern-matching union items, %q) = %v, want accept", "7 42", err)
	}
	_, err = ValidateLexical(b, noSchema{}, patList, "7 abc", nil)
	if err == nil {
		t.Fatal("ValidateLexical(list, noSchema{}, token violating the union item type's pattern) = nil, want the item type's own clause-1 rejection (dv_list → dv_pattern)")
	}
	if r, _ := xsderr.RuleOf(err); r != "cvc-pattern-valid" {
		t.Errorf("pattern-violating union item charged %s, want cvc-pattern-valid (§4.3.4.4 on the union ITEM type)", r)
	}
	// The pattern was matched against the TOKEN "abc", not the whole lexical
	// "7 abc": the item type decides per space-delimited substring (dv_list).
	if msg := err.Error(); !strings.Contains(msg, `"abc"`) || strings.Contains(msg, `"7 abc"`) {
		t.Errorf("pattern rejection = %q, want it to name the token %q, not the whole list lexical %q", msg, "abc", "7 abc")
	}
}

// TestListEnumerationResolvesThroughAnonymousBase is the regression guard for
// the shape builtin.Seed now produces: every list datatype restricts an
// ANONYMOUS intermediate list (Datatypes §3.4.5/§3.4.10/§3.4.12), so a
// zero-QName component sits on the base chain of any user type derived from
// xs:NMTOKENS/xs:IDREFS/xs:ENTITIES. This mirrors that chain — anonymous
// constructed list, the named plural type restricting it with minLength = 1, the
// user's own restriction carrying an enumeration — and drives it end to end.
//
// Two resolutions have to survive the extra hop. The enumeration's {value} is
// parsed in its DECLARING type's space (declaringFacetSpace), whose whiteSpace
// mode is read off the declaring type's BASE — here the plural type, which owns
// no whiteSpace facet and inherits collapse across the anonymous node through the
// §3.16.6.4 overlay. If either resolution missed, the enumeration would decide
// nothing and the out-of-enumeration list would be false-accepted.
func TestListEnumerationResolvesThroughAnonymousBase(t *testing.T) {
	item := primType(t, "myitem", "collapse")
	b := stubItemBackend{item: item.Name()}

	anon, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{},
		listOf(item), xsd.AnySimpleType(),
		[]xsd.Facet{xsd.NewFacet(xsd.FacetWhiteSpace, []string{"collapse"}, true)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(anonymous intermediate list): %v", err)
	}
	plural, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: xsd.XMLSchemaNS, Local: "NMTOKENS"},
		listOf(item), anon,
		[]xsd.Facet{xsd.NewFacet(xsd.FacetMinLength, []string{"1"}, false)}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(xs:NMTOKENS): %v", err)
	}
	// stubItemBackend keys an item's value by token length, so "aa bb" and
	// "aa ccc" are different list values.
	leaf, err := newCheckedSimpleType(xsderr.Loc{}, xsd.QName{Space: "urn:test", Local: "twoShortTokens"},
		listOf(item), plural, []xsd.Facet{enumOf("aa bb")}, nil)
	if err != nil {
		t.Fatalf("NewSimpleType(user restriction of xs:NMTOKENS): %v", err)
	}

	if _, err := ValidateLexical(b, noSchema{}, leaf, "aa bb", nil); err != nil {
		t.Fatalf("ValidateLexical(enumerated list value) = %v, want accept", err)
	}
	_, err = ValidateLexical(b, noSchema{}, leaf, "aa ccc", nil)
	if err == nil {
		t.Fatal("ValidateLexical(out-of-enumeration list) = nil, want the enumeration's rejection")
	}
	if r, _ := xsderr.RuleOf(err); r != "cvc-enumeration-valid" {
		t.Errorf("out-of-enumeration list charged %s, want cvc-enumeration-valid (§4.3.5.4)", r)
	}
}
