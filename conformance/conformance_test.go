package conformance

import (
	"os"
	"testing"
)

// TestConformance runs the W3C suite through the lane executors and enforces
// the ratchet, per conformance/doc.go.
//
//   - Submodule absent: skip with a pointer to `git submodule update --init`.
//   - Read-only (default): Compare each lane's observed run against its
//     committed expectations and fail only on a Regressed case; New/Improved/
//     Vanished do not fail the read-only run (doc.go "Running").
//   - GOXSD_RATCHET=1: additionally Ratchet each lane and rewrite its file;
//     a Ratchet refusal (regression or vanished) fails the test. Arbiter only.
//   - GOXSD_CASE=<id>: narrow execution to one case across all lanes.
//
// At M1 no real executor is registered, so every case is a stub Fail and, with
// empty committed lane files, every case is New — the read-only run passes.
func TestConformance(t *testing.T) {
	index := suitePath()
	if _, err := os.Stat(index); err != nil {
		if os.IsNotExist(err) {
			t.Skipf("W3C suite not present at %s; run `git submodule update --init %s`", index, suiteRoot)
		}
		t.Fatalf("stat suite index %s: %v", index, err)
	}

	cases, err := parseSuite(index)
	if err != nil {
		t.Fatalf("parsing suite: %v", err)
	}
	t.Logf("discovered %d cases across %d lanes", len(cases), len(defaultLanes()))

	if only, ok := os.LookupEnv("GOXSD_CASE"); ok {
		cases = narrowToCase(t, cases, only)
	}

	ratcheting := os.Getenv("GOXSD_RATCHET") == "1"
	for _, l := range defaultLanes() {
		runConformanceLane(t, l, cases, ratcheting)
	}
}

// runConformanceLane executes one lane and applies the read-only or ratcheting
// policy to its result.
func runConformanceLane(t *testing.T, l lane, cases []caseSpec, ratcheting bool) {
	t.Helper()
	actual := runLane(l, cases)
	path := laneFile(l.name)
	expected, err := LoadExpectations(path)
	if err != nil {
		t.Fatalf("lane %s: loading expectations: %v", l.name, err)
	}
	t.Logf("lane %s: %d cases", l.name, len(actual))

	if !ratcheting {
		d := Compare(expected, actual)
		if len(d.Regressed) > 0 {
			t.Errorf("lane %s: %d regressed case(s): %v", l.name, len(d.Regressed), d.Regressed)
		}
		return
	}

	merged, err := Ratchet(expected, actual)
	if err != nil {
		t.Errorf("lane %s: %v", l.name, err)
		return
	}
	if err := WriteExpectations(path, merged); err != nil {
		t.Fatalf("lane %s: writing expectations: %v", l.name, err)
	}
}

// narrowToCase keeps only the case whose ID equals only, failing clearly if no
// discovered case matches (GOXSD_CASE debugging aid).
func narrowToCase(t *testing.T, cases []caseSpec, only string) []caseSpec {
	t.Helper()
	for _, c := range cases {
		if c.id == only {
			return []caseSpec{c}
		}
	}
	t.Fatalf("GOXSD_CASE=%q matched none of the %d discovered cases", only, len(cases))
	return nil
}
