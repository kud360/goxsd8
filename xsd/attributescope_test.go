package xsd

import (
	"testing"

	"github.com/kud360/goxsd8/xsderr"
)

// This file is package-internal only to serve the internal fixtures: the tests
// that build an Attribute Declaration while exercising an unexported finalize
// helper (Phase E in valueconstraintvalid_test.go, the derivation folds in
// complexderivation_test.go) need a local {scope} (§3.2.1 sc_a), and
// NewAttributeLocalScope returns an error every one of them would otherwise have
// to handle inline. AttributeScope's own exported behaviour is pinned from
// outside the package, in attributedeclaration_test.go.

// aLocalScope is a local attribute {scope} whose {parent} names a containing
// complex type called container, for the internal tests that only need a
// non-global declaration.
func aLocalScope(t *testing.T) AttributeScope {
	t.Helper()
	s, err := NewAttributeLocalScope(xsderr.Loc{}, AttributeComplexTypeScopeParent{Name: QName{Local: "container"}})
	if err != nil {
		t.Fatalf("NewAttributeLocalScope: %v", err)
	}
	return s
}
