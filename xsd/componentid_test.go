package xsd_test

import (
	"reflect"
	"testing"

	"github.com/kud360/goxsd8/xsd"
)

func TestNewComponentIDMintsDistinctIdentities(t *testing.T) {
	if a, b := xsd.NewComponentID(), xsd.NewComponentID(); a == b {
		t.Fatal("NewComponentID() == NewComponentID(): two mints share one identity")
	}
}

// TestNewComponentIDDistinctWhenEscaping guards the zero-size-allocation trap
// identityCell's blank byte field exists to defeat: an escaping new(struct{})
// is served from runtime.zerobase, so every zero-size cell would share one
// address. Holding all the IDs in a slice forces them to escape, which is the
// configuration where a struct{} cell collides (a stack-allocated pair does
// not, which is why this test does not simply mint two locals).
func TestNewComponentIDDistinctWhenEscaping(t *testing.T) {
	const n = 1000
	ids := make([]xsd.ComponentID, 0, n)
	for range n {
		ids = append(ids, xsd.NewComponentID())
	}
	seen := make(map[xsd.ComponentID]int, n)
	for i, id := range ids {
		if first, dup := seen[id]; dup {
			t.Fatalf("minted identity %d collides with %d", i, first)
		}
		seen[id] = i
	}
}

func TestZeroComponentIDIsUnminted(t *testing.T) {
	if xsd.NewComponentID() == (xsd.ComponentID{}) {
		t.Fatal("NewComponentID() == ComponentID{}: a minted identity must never equal the unminted one")
	}
	// The zero value is the sanctioned "absent" spelling: two independently
	// written absences are the same absence, so `id == ComponentID{}` is a
	// sound presence test without an IsPresent helper.
	var absent xsd.ComponentID
	if absent != (xsd.ComponentID{}) {
		t.Error("a zero-valued ComponentID does not equal the ComponentID{} literal")
	}
}

// TestComponentIDSurvivesValueCopy proves the property the whole scheme rests
// on: identity is a pointer, so it survives being copied by value — through a
// plain assignment, through a struct field, and through a function's by-value
// parameter.
func TestComponentIDSurvivesValueCopy(t *testing.T) {
	id := xsd.NewComponentID()

	copied := id
	if copied != id {
		t.Error("assignment copy lost identity")
	}

	type holder struct{ id xsd.ComponentID }
	h := holder{id: id}
	h2 := h
	if h2.id != id {
		t.Error("struct-field copy lost identity")
	}

	byValue := func(got xsd.ComponentID) bool { return got == id }
	if !byValue(id) {
		t.Error("by-value argument lost identity")
	}
}

// TestComponentIDDeepEqualIsIdentityBlind pins a KNOWN, DOCUMENTED,
// INTENTIONAL gap — it is not a bug to "fix". reflect.DeepEqual follows the
// pointer inside a ComponentID and compares the pointees, and every
// identityCell is the same zero byte, so two DISTINCT identities compare
// DeepEqual-equal. That makes DeepEqual useless for asserting a {context} is
// the RIGHT one; use == on the ID, as ComponentID's doc says.
//
// The assertion is deliberately that DeepEqual reports TRUE: if a future
// stdlib or runtime change made it false, ComponentID's documented footgun
// would be stale and this test is what notices.
func TestComponentIDDeepEqualIsIdentityBlind(t *testing.T) {
	a, b := xsd.NewComponentID(), xsd.NewComponentID()
	if a == b {
		t.Fatal("precondition failed: NewComponentID minted the same identity twice")
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("reflect.DeepEqual(a, b) = false for two distinct ComponentIDs; ComponentID's doc says it is identity-blind and reports true — update the doc if this is now genuinely fixed")
	}
}
