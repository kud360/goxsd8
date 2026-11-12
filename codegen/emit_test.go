package codegen

import (
	"slices"
	"testing"

	"github.com/kud360/goxsd8/internal/goshape"
	"github.com/kud360/goxsd8/xsd"
)

// TestScopeKeyBranchStep pins the "|" nestedKey writes before a {type table}
// branch step: a branch step and an element step whose {target namespace} is
// the branch's slot and whose {name} is its local name (both legal, the
// {target namespace} being an anyURI) give two keys under one root and prefix.
// Each row checks that branchScope ends in that branch step, so the colliding
// element step is the one it would collide with.
func TestScopeKeyBranchStep(t *testing.T) {
	schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="b"/>
  <xs:element name="e" type="b">
    <xs:alternative test="@a"><xs:complexType><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType></xs:alternative>
    <xs:alternative test="@c"><xs:complexType><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType></xs:alternative>
    <xs:alternative><xs:complexType><xs:complexContent><xs:extension base="b"/></xs:complexContent></xs:complexType></xs:alternative>
  </xs:element>
</xs:schema>`)
	decl, ok := schema.Element(xsd.QName{Local: "e"})
	if !ok {
		t.Fatal("no element e")
	}
	branches, _, err := goshape.GoverningTypes(schema, decl, nil)
	if err != nil {
		t.Fatal(err)
	}
	ifc := scope{root: elementRoot, path: elementPath(xsd.QName{Space: "urn:t", Local: "e"})}
	tests := []struct {
		slot  goshape.Slot
		local string
	}{
		{goshape.DefaultSlot{}, "defaultType"},
		{goshape.AlternativeSlot(2), "alternative2Type"},
	}
	for _, tt := range tests {
		t.Run(tt.slot.Name(), func(t *testing.T) {
			i := slices.IndexFunc(branches, func(b goshape.GoverningType) bool { return b.Slot() == tt.slot })
			if i < 0 || branches[i].Type().Name() != (xsd.QName{}) {
				t.Fatalf("element e has no anonymous governing type at its %s", tt.slot.Name())
			}
			branch := branchScope(ifc, branches[i])
			wantStep := pathStep{name: xsd.QName{Local: tt.local}, slot: tt.slot.Name()}
			if got := branch.path[len(branch.path)-1]; got != wantStep {
				t.Fatalf("branchScope's last step = %+v, want %+v", got, wantStep)
			}
			element := scope{root: ifc.root, path: append(slices.Clone(ifc.path), pathStep{name: xsd.QName{Space: tt.slot.Name(), Local: tt.local}})}
			if branch.key() == element.key() {
				t.Errorf("branch step and element step {%s}%s share the nestedKey %s", tt.slot.Name(), tt.local, branch.key())
			}
		})
	}
}
