package conformance

import (
	"slices"
	"testing"
)

// censusCases is a synthetic lane's worth of cases in case-ID order (the order
// parseSuite guarantees), covering every branch takeDeclineCensus distinguishes:
// a decided-and-agreed case, a decided-and-disagreed one, two declined ones out
// of insertion order, an indeterminate one, and one the lane does not claim.
func censusCases() []caseSpec {
	return []caseSpec{
		{id: "set/g/schema/agree", kind: kindSchema, expect: expectValid()},
		{id: "set/g/schema/declineB", kind: kindSchema, expect: expectInvalid()},
		{id: "set/g/schema/disagree", kind: kindSchema, expect: expectValid()},
		{id: "set/g/schema/indeterminate", kind: kindSchema, expect: expectIndeterminate()},
		{id: "set/g/schema/unclaimed", kind: kindSchema, expect: expectValid()},
		{id: "set/g/schema/zdeclineA", kind: kindSchema, expect: expectValid()},
	}
}

// censusExec is a fake executor with the exact shape every real one in this
// package has: a case whose observation it can produce is DECIDED by comparing
// that observation with the declared outcome, and every other case is DECLINED
// with a Fail() that never reads the outcome, naming the refusal
// "refused-<last letter of its ID>". observations names the cases it can
// decide and the validity it observes for each.
func censusExec(observations map[string]bool, calls *[]string) laneExecutor {
	return func(c caseSpec) (Status, refusal) {
		*calls = append(*calls, c.id)
		observed, decidable := observations[c.id]
		if !decidable {
			return Fail(), refusal("refused-" + c.id[len(c.id)-1:])
		}
		if observed == c.expect.wantsValid() {
			return Pass(), ""
		}
		return Fail(), ""
	}
}

// TestTakeDeclineCensusSeparatesDeclinesFromDecisions pins the guard of issue
// #327: a recorded fail is re-attempted with its declared outcome flipped, which
// splits the failures an executor DECIDED (and disagreed with) from the ones it
// DECLINED. Only the declines are harvest candidates — a decided-and-disagreed
// case is an engine that answered, wrongly, not a shape nothing recognized — so
// mis-classifying "disagree" here would refill the queue with noise and hide the
// signal the issue exists to expose.
func TestTakeDeclineCensusSeparatesDeclinesFromDecisions(t *testing.T) {
	cases := censusCases()
	observations := map[string]bool{
		"set/g/schema/agree":     true,  // declared valid, observed valid  => Pass
		"set/g/schema/disagree":  false, // declared valid, observed invalid => Fail, but DECIDED
		"set/g/schema/unclaimed": true,
	}
	var calls []string
	l := lane{
		name:    "fake",
		selects: func(c caseSpec) bool { return c.id != "set/g/schema/unclaimed" },
		exec:    censusExec(observations, &calls),
	}

	actual := runLane(l, cases)
	calls = nil
	census := takeDeclineCensus(l, cases, actual)

	// Each candidate carries the refusal the probe's own run named, so no
	// second run is needed to read it (#2008).
	wantCandidates := []declinedCase{{"set/g/schema/declineB", "refused-B"}, {"set/g/schema/zdeclineA", "refused-A"}}
	if !slices.Equal(census.candidates, wantCandidates) {
		t.Errorf("candidates = %v, want %v", census.candidates, wantCandidates)
	}
	wantIndeterminate := []string{"set/g/schema/indeterminate"}
	if !slices.Equal(census.indeterminate, wantIndeterminate) {
		t.Errorf("indeterminate = %v, want %v", census.indeterminate, wantIndeterminate)
	}
	var disagreed []string
	for _, c := range census.disagreed {
		disagreed = append(disagreed, c.id)
	}
	wantDisagreed := []string{"set/g/schema/disagree"}
	if !slices.Equal(disagreed, wantDisagreed) {
		t.Errorf("disagreed = %v, want %v", disagreed, wantDisagreed)
	}

	// The probe re-runs exactly the claimed, non-indeterminate failures, in
	// case-ID order: never a passing case (decided by definition), never an
	// unclaimed one, never an indeterminate one (runLane declines it before any
	// executor, issue #277).
	wantCalls := []string{"set/g/schema/declineB", "set/g/schema/disagree", "set/g/schema/zdeclineA"}
	if !slices.Equal(calls, wantCalls) {
		t.Errorf("probe re-ran %v, want %v", calls, wantCalls)
	}
}

// TestTakeDeclineCensusLeavesDiscoveryUntouched pins that the probe flips a COPY:
// the discovered caseSpec keeps the suite's declared outcome, so a census can
// never perturb the scoring of the run it audits.
func TestTakeDeclineCensusLeavesDiscoveryUntouched(t *testing.T) {
	cases := censusCases()
	before := make([]expectation, len(cases))
	for i, c := range cases {
		before[i] = c.expect
	}
	var calls []string
	l := lane{name: "fake", selects: func(caseSpec) bool { return true }, exec: censusExec(nil, &calls)}

	takeDeclineCensus(l, cases, runLane(l, cases))

	for i, c := range cases {
		if c.expect != before[i] {
			t.Errorf("case %s: expectation mutated to %v, want %v", c.id, c.expect, before[i])
		}
	}
}

// TestChargedIDsWritesTheChargeOnlyForALaneThatHasOne pins the GOXSD_DECLINES=1
// decided-disagreements rendering tools/internal/declinecensus parses (#1740):
// `<id>=<charge>` in case order for a lane with a charge probe, the bare ID for a
// lane without one.
func TestChargedIDsWritesTheChargeOnlyForALaneThatHasOne(t *testing.T) {
	disagreed := []caseSpec{{id: "set/g/schema/a"}, {id: "set/g/schema/b"}}
	charging := lane{name: "fake", charge: func(c caseSpec) string { return "rule-of-" + c.id[len(c.id)-1:] }}
	want := []string{"set/g/schema/a=rule-of-a", "set/g/schema/b=rule-of-b"}
	if got := chargedIDs(charging, disagreed); !slices.Equal(got, want) {
		t.Errorf("charging lane: chargedIDs = %v, want %v", got, want)
	}
	want = []string{"set/g/schema/a", "set/g/schema/b"}
	if got := chargedIDs(lane{name: "fake"}, disagreed); !slices.Equal(got, want) {
		t.Errorf("non-charging lane: chargedIDs = %v, want %v", got, want)
	}
}

// TestRefusedIDsWritesTheRefusalWhereTheExecutorNamedOne pins the
// GOXSD_DECLINES=1 decline-candidates rendering tools/internal/declinecensus
// parses (#2008): `<id>=<refusal>` for a candidate whose executor named one,
// the bare ID for one whose executor names none, in census order.
func TestRefusedIDsWritesTheRefusalWhereTheExecutorNamedOne(t *testing.T) {
	candidates := []declinedCase{{"set/g/instance/a", refuseErrorType}, {"set/g/schema/b", ""}}
	want := []string{"set/g/instance/a=error-type", "set/g/schema/b"}
	if got := refusedIDs(candidates); !slices.Equal(got, want) {
		t.Errorf("refusedIDs = %v, want %v", got, want)
	}
}

// TestFlipExpectationExchangesDecidedOutcomes pins the probe's premise: the two
// decided outcomes map onto each other, so an executor that compares its
// observation with the declared outcome necessarily answers the opposite Status
// for the flipped copy.
func TestFlipExpectationExchangesDecidedOutcomes(t *testing.T) {
	cases := []struct {
		name string
		in   expectation
		want expectation
	}{
		{"valid flips to invalid", expectValid(), expectInvalid()},
		{"invalid flips to valid", expectInvalid(), expectValid()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := flipExpectation(caseSpec{id: "c", expect: tc.in}).expect
			if got != tc.want {
				t.Errorf("flipExpectation(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
