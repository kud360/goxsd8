package codegen

import (
	"slices"
	"testing"

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
  <xs:element name="e"><xs:complexType/></xs:element>
</xs:schema>`)
	decl, ok := schema.Element(xsd.QName{Local: "e"})
	if !ok {
		t.Fatal("no element e")
	}
	anonymous, ok := schema.ResolvedType(decl.TypeDefinition())
	if !ok || anonymous.Name() != (xsd.QName{}) {
		t.Fatalf("element e's type %v does not resolve to an anonymous type", decl.TypeDefinition())
	}
	ifc := scope{root: elementRoot, path: elementPath(xsd.QName{Space: "urn:t", Local: "e"})}
	tests := []struct {
		slot  branchSlot
		local string
	}{
		{defaultSlot{}, "defaultType"},
		{alternativeSlot(2), "alternative2Type"},
	}
	for _, tt := range tests {
		t.Run(tt.slot.name(), func(t *testing.T) {
			branch := branchScope(ifc, tableBranch{slot: tt.slot, td: anonymous})
			wantStep := pathStep{name: xsd.QName{Local: tt.local}, slot: tt.slot.name()}
			if got := branch.path[len(branch.path)-1]; got != wantStep {
				t.Fatalf("branchScope's last step = %+v, want %+v", got, wantStep)
			}
			element := scope{root: ifc.root, path: append(slices.Clone(ifc.path), pathStep{name: xsd.QName{Space: tt.slot.name(), Local: tt.local}})}
			if branch.key() == element.key() {
				t.Errorf("branch step and element step {%s}%s share the nestedKey %s", tt.slot.name(), tt.local, branch.key())
			}
		})
	}
}
