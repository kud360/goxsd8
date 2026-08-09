package builtin

import (
	"strings"
	"testing"

	"github.com/kud360/goxsd8/xsd"
	"github.com/kud360/goxsd8/xsderr"
)

// TestInterposeListBase pins the discriminant interposeListBase branches on: the
// row's {variety} ALONE. A List row is the two-step shape of Datatypes §3.4.5/
// §3.4.10/§3.4.12 and gets the anonymous intermediate list; an Atomic row keeps
// the base it was handed.
//
// The refusal case is the one that matters. On a generated row Base is set to
// "anySimpleType" BECAUSE the variety is list (tools/hfnextract/builtins), so it
// carries no independent information — and if the two ever disagreed, quietly
// flattening the row back onto its stated base would false-accept exactly the
// shape cos-st-restricts clause 2.2.1.2 exists to reject. So the disagreement is
// a loud construction error instead, in the manner of ownFacets' unknown-
// FacetKind refusal.
func TestInterposeListBase(t *testing.T) {
	item, err := xsd.NewPrimitiveType(xsderr.Loc{}, qname("item"), nil, nil)
	if err != nil {
		t.Fatalf("build item primitive: %v", err)
	}
	listDerivation := xsd.ListDerivation{Item: item}

	t.Run("list row interposes the anonymous list", func(t *testing.T) {
		spec := TypeSpec{Name: "NMTOKENS", Base: "anySimpleType", Variety: List{Item: "item"}}
		got, derivation, err := interposeListBase(spec, listDerivation, xsd.AnySimpleType())
		if err != nil {
			t.Fatalf("interposeListBase: %v", err)
		}
		if got == xsd.AnySimpleType() {
			t.Fatal("interposeListBase returned xs:anySimpleType unchanged, want the interposed list")
		}
		if got.Name() != (xsd.QName{}) {
			t.Errorf("interposed list {name} = %v, want absent", got.Name())
		}
		if base, err := got.Base(noSchema{}); err != nil || base != xsd.AnySimpleType() {
			t.Errorf("interposed list {base type definition} is not xs:anySimpleType")
		}
		// The ListDerivation — and with it the {item type definition} — is minted
		// ONCE, on the interposed node; the named row restricts it and re-derives
		// the item from the base rather than carrying a second copy (STYLE D3).
		if gotItem, err := got.Item(noSchema{}); err != nil || gotItem != item {
			t.Errorf("interposed list {item type definition} is not the item primitive")
		}
		if derivation != (xsd.SimpleTypeDerivation)(xsd.RestrictionDerivation{}) {
			t.Errorf("named list row derivation = %#v, want xsd.RestrictionDerivation{}", derivation)
		}
	})

	t.Run("atomic row keeps its base", func(t *testing.T) {
		spec := TypeSpec{Name: "token", Base: "normalizedString", Variety: Atomic{}}
		got, derivation, err := interposeListBase(spec, xsd.RestrictionDerivation{}, item)
		if err != nil {
			t.Fatalf("interposeListBase: %v", err)
		}
		if got != item {
			t.Errorf("interposeListBase(atomic row) = %v, want the base it was handed", got)
		}
		if derivation != (xsd.SimpleTypeDerivation)(xsd.RestrictionDerivation{}) {
			t.Errorf("atomic row derivation = %#v, want the one it was handed", derivation)
		}
	})

	t.Run("list row with a non-anySimpleType base is refused", func(t *testing.T) {
		spec := TypeSpec{Name: "weirdList", Base: "token", Variety: List{Item: "item"}}
		got, _, err := interposeListBase(spec, listDerivation, item)
		if err == nil {
			t.Fatalf("interposeListBase(list row based on %q) = %v, want a refusal", spec.Base, got)
		}
		if !strings.Contains(err.Error(), "weirdList") || !strings.Contains(err.Error(), "token") {
			t.Errorf("refusal %q must name the offending row and its base", err.Error())
		}
	})
}
