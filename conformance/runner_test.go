package conformance

import (
	"path/filepath"
	"testing"
)

// fakeCases is a tiny synthetic suite used to exercise the runner's lane
// iteration and ratchet integration without the real submodule.
func fakeCases() []caseSpec {
	return []caseSpec{
		{id: "set/g/schema/a", kind: kindSchema, expectValid: true},
		{id: "set/g/schema/b", kind: kindSchema, expectValid: false},
		{id: "set/g/instance/c", kind: kindInstance, expectValid: true},
	}
}

// passSchema is a fake executor: schema cases pass, everything else fails.
func passSchema(c caseSpec) Status {
	if c.kind == kindSchema {
		return Pass()
	}
	return Fail()
}

func TestRunLaneSelectsOnlyClaimedCases(t *testing.T) {
	l := lane{name: "fake", selects: selectsKind(kindSchema), exec: passSchema}
	actual := runLane(l, fakeCases())

	if len(actual) != 2 {
		t.Fatalf("selector must claim the 2 schema cases, got %d: %v", len(actual), actual)
	}
	if _, ok := actual["set/g/instance/c"]; ok {
		t.Errorf("instance case must not be claimed by a schema-kind lane")
	}
	if !actual["set/g/schema/a"].IsPass() {
		t.Errorf("executor result not recorded for claimed case")
	}
}

// TestRunLaneRatchetRoundTrip exercises the runner's integration of runLane
// with Ratchet + WriteExpectations against a temp-dir lane file (never the
// real committed path): a fresh lane starts empty, records observed New cases,
// and reloads byte-stably.
func TestRunLaneRatchetRoundTrip(t *testing.T) {
	l := lane{name: "fake", selects: func(caseSpec) bool { return true }, exec: passSchema}
	actual := runLane(l, fakeCases())

	path := filepath.Join(t.TempDir(), "fake.txt")
	expected, err := LoadExpectations(path) // missing file => empty lane
	if err != nil {
		t.Fatalf("load empty lane: %v", err)
	}
	merged, err := Ratchet(expected, actual)
	if err != nil {
		t.Fatalf("ratchet must accept all-New cases: %v", err)
	}
	if err := WriteExpectations(path, merged); err != nil {
		t.Fatalf("write: %v", err)
	}

	reloaded, err := LoadExpectations(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	want := map[string]Status{
		"set/g/schema/a":   Pass(),
		"set/g/schema/b":   Pass(),
		"set/g/instance/c": Fail(),
	}
	if len(reloaded) != len(want) {
		t.Fatalf("reloaded %d cases, want %d", len(reloaded), len(want))
	}
	for id, w := range want {
		if reloaded[id] != w {
			t.Errorf("case %q: got %v, want %v", id, reloaded[id], w)
		}
	}
}

// TestRunLaneRatchetRefusesRegression proves the runner surfaces a ratchet
// refusal: a case committed as pass that the lane now fails blocks the merge.
func TestRunLaneRatchetRefusesRegression(t *testing.T) {
	// Lane executor fails the instance case; commit it as an expected pass.
	l := lane{name: "fake", selects: func(caseSpec) bool { return true }, exec: passSchema}
	actual := runLane(l, fakeCases())

	expected := map[string]Status{"set/g/instance/c": Pass()}
	if _, err := Ratchet(expected, actual); err == nil {
		t.Fatal("ratchet must refuse when an executor regresses a committed pass")
	}
}
