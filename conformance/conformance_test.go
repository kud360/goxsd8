package conformance

import (
	"os"
	"testing"
)

// TestConformance runs the W3C suite through the lane executors and enforces
// the ratchet, per conformance/doc.go.
//
//   - Submodule absent: fail with a pointer to `git submodule update --init`
//     (issue #309), unless a read-only run explicitly opted out with
//     GOXSD_SUITE_OPTIONAL=1. A run that executed zero cases must never be
//     mistaken for a green one, and the opt-out never covers a ratchet run.
//   - Read-only (default): Compare each lane's observed run against its
//     committed expectations and fail only on a Regressed case; New/Improved/
//     Vanished do not fail the read-only run (doc.go "Running"). Improved is
//     logged so an agent that cannot run the ratchet can still see the upward
//     movement its branch earned but has not banked (issue #303).
//   - Every lane, on either path, reports its decline census (issue #327): the
//     count of recorded failures no executor decided at all, re-checked this run
//     rather than trusted. GOXSD_DECLINES=1 also lists their IDs.
//   - Sanctioned applicability removals — cases discovery withheld because the
//     suite's own metadata scopes them away from this processor (issue #576) —
//     are logged on the read-only path and never fail it.
//   - GOXSD_RATCHET=1: additionally Ratchet each lane and rewrite its file;
//     a Ratchet refusal (regression, vanished, or a removal count the run did not
//     assert) fails the test. Arbiter only.
//   - GOXSD_RATCHET_REMOVALS=<lane>=<n>,…: arbiter-only, ratchet-path-only
//     assertion of the sanctioned removals each lane is expected to bank; set
//     without GOXSD_RATCHET=1 it fails the run rather than half-applying.
//   - GOXSD_CASE=<id>: narrow execution to one case across all lanes.
//
// At M1 no real executor is registered, so every case is a stub Fail and, with
// empty committed lane files, every case is New — the read-only run passes.
func TestConformance(t *testing.T) {
	ratcheting := os.Getenv("GOXSD_RATCHET") == "1"
	removals := assertedRemovals(t, ratcheting)
	index := suitePath()
	if err := checkSuitePresent(index); err != nil {
		endUnusableSuiteRun(t, err, ratcheting)
	}

	found, err := parseSuite(index)
	if err != nil {
		t.Fatalf("parsing suite: %v", err)
	}
	t.Logf("discovered %d cases across %d lanes, %d withheld as inapplicable",
		len(found.cases), len(defaultLanes()), len(found.withheld))

	cases := found.cases
	if only, ok := os.LookupEnv("GOXSD_CASE"); ok {
		cases = narrowToCase(t, cases, only)
	}

	for _, l := range defaultLanes() {
		runConformanceLane(t, l, cases, found.withheld, ratcheting, removals[l.name])
	}
}

// assertedRemovals reads the arbiter's per-lane removal assertion
// (ratchetRemovalsEnv) and ends the run on anything removalAssertions refuses —
// a value set off the ratchet path, or a malformed one. The gate lives in
// removalAssertions so it is testable; this is only its env binding, in the same
// shape endUnusableSuiteRun uses for the suite opt-out (issue #309).
func assertedRemovals(t *testing.T, ratcheting bool) map[string]RemovalAssertion {
	t.Helper()
	raw, set := os.LookupEnv(ratchetRemovalsEnv)
	byLane, err := removalAssertions(raw, set, ratcheting)
	if err != nil {
		t.Fatalf("%s: %v", ratchetRemovalsEnv, err)
	}
	return byLane
}

// endUnusableSuiteRun ends the run when the suite index is unusable. The default
// is a hard failure so a container without the submodule cannot exit `ok`
// (issue #309); GOXSD_SUITE_OPTIONAL=1 downgrades that to a skip for a read-only
// run only, never for a ratchet run whose whole output would otherwise be an
// unearned "no movement".
func endUnusableSuiteRun(t *testing.T, err error, ratcheting bool) {
	t.Helper()
	if os.Getenv(suiteOptionalEnv) != "1" {
		t.Fatalf("%v (set %s=1 only in an environment that legitimately has no suite)", err, suiteOptionalEnv)
	}
	if ratcheting {
		t.Fatalf("%v (%s=1 does not cover a ratchet run: an empty suite must not report no movement)", err, suiteOptionalEnv)
	}
	t.Skipf("%v (skipped: %s=1)", err, suiteOptionalEnv)
}

// runConformanceLane executes one lane and applies the read-only or ratcheting
// policy to its result. withheld is the suite-wide set of case IDs discovery
// ruled inapplicable; it needs no per-lane routing because a withheld ID only
// classifies against the lane whose committed expectations carry it.
func runConformanceLane(t *testing.T, l lane, cases []caseSpec, withheld []string, ratcheting bool, removals RemovalAssertion) {
	t.Helper()
	actual := runLane(l, cases)
	path := laneFile(l.name)
	expected, err := LoadExpectations(path)
	if err != nil {
		t.Fatalf("lane %s: loading expectations: %v", l.name, err)
	}
	t.Logf("lane %s: %d cases", l.name, len(actual))
	reportDeclines(t, l, cases, actual)

	if !ratcheting {
		reportLaneReadOnly(t, l, expected, actual, withheld)
		return
	}

	merged, err := Ratchet(expected, actual, withheld, removals)
	if err != nil {
		t.Errorf("lane %s: %v", l.name, err)
		return
	}
	if err := WriteExpectations(path, merged); err != nil {
		t.Fatalf("lane %s: writing expectations: %v", l.name, err)
	}
}

// reportLaneReadOnly applies the read-only policy: fail on a Regressed case, and
// on nothing else about the score. Improved cases and sanctioned applicability
// removals are reported through t.Logf so an agent who cannot run the ratchet
// still sees what its branch would bank; banking either needs GOXSD_RATCHET=1,
// and a removal needs the arbiter's asserted count on top of that.
//
// A Compare error is a different animal and does fail the read-only run: it means
// discovery both withheld and produced one case, which is a runner bug rather
// than a conformance result, and no run should proceed on a self-contradicting
// case list.
func reportLaneReadOnly(t *testing.T, l lane, expected, actual map[string]Status, withheld []string) {
	t.Helper()
	d, err := Compare(expected, actual, withheld)
	if err != nil {
		t.Errorf("lane %s: %v", l.name, err)
		return
	}
	if len(d.Regressed) > 0 {
		t.Errorf("lane %s: %d regressed case(s): %v", l.name, len(d.Regressed), d.Regressed)
	}
	if len(d.Improved) > 0 {
		t.Logf("lane %s: %d case(s) improved, not yet banked (GOXSD_RATCHET=1 required to write): %v",
			l.name, len(d.Improved), d.Improved)
	}
	if len(d.Removed) > 0 {
		t.Logf("lane %s: %d sanctioned applicability removal(s), not yet banked (GOXSD_RATCHET=1 with %s=%s=%d required to write): %v",
			l.name, len(d.Removed), ratchetRemovalsEnv, l.name, len(d.Removed), d.Removed)
	}
}

// reportDeclines surfaces the lane's decline census (issue #327) through the
// same t.Logf channel the Improved cases already use, so a -v run shows how many
// of the lane's recorded failures no executor decided at all. That count is the
// harvest queue's size: it drops at the landing that widens an engine, instead
// of a wrongly-declined case waiting for an unrelated reader edit to stumble
// over it. The census reports only — it scores nothing and writes nothing, so it
// runs on the ratcheting path too.
func reportDeclines(t *testing.T, l lane, cases []caseSpec, actual map[string]Status) {
	t.Helper()
	census := takeDeclineCensus(l, cases, actual)
	if len(census.candidates) > 0 {
		t.Logf("lane %s: %d declined case(s) recorded fail — harvest candidates re-checked this run (%s=1 lists them)",
			l.name, len(census.candidates), declinesEnv)
	}
	if census.indeterminate > 0 {
		t.Logf("lane %s: %d further case(s) declined as indeterminate, never harvestable (issue #277)",
			l.name, census.indeterminate)
	}
	if os.Getenv(declinesEnv) != "1" {
		return
	}
	t.Logf("lane %s: decline candidates: %v", l.name, census.candidates)
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
