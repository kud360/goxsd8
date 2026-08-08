package conformance

// This file is the DECLINE CENSUS (issue #327): the guard that re-attempts
// every decline instead of trusting it.
//
// # The problem it closes
//
// A lane executor either DECIDES a case — it observes this processor's outcome
// for the fixture and compares that with the suite's declared outcome — or
// DECLINES it: a reader does not recognize the fixture's shape (the `ok=false`
// returns throughout datatypes.go), a document will not resolve, a closure
// leaves the producer's decidable subset. Both record Fail(), and an expectation
// file has only the two tokens, so once the run is scored the two are
// indistinguishable. A decline is therefore a snapshot of engine capability at
// the moment the DECLINE CODE was written, and nothing ever re-checked it:
// d3_3_4v16 was decidable for nine days (its list-variety capability landed with
// #75) before an unrelated reader widening (#223) happened to cover it, and the
// wrongly-declined case was invisible the whole time. The set "declined AND
// recorded fail" is precisely the harvest queue.
//
// # How a decline is detected without touching a single executor
//
// Every executor in this package ends at ONE comparison of its observation with
// the case's declared outcome — `observedValid == c.expect.wantsValid()` in
// datatypes.go, decideSchema(observed, expected) in schema.go — and every
// decline returns Fail() BEFORE reaching it. That yields an exact test rather
// than a heuristic: re-run the case with its declared outcome FLIPPED. An
// executor that decided the case compares the same observation against the
// opposite outcome and so returns the opposite Status, meaning exactly one of
// the two polarities passes; an executor that declined never reads the outcome
// at all and returns Fail under both. Fail under both polarities is therefore
// "no executor decided this case".
//
// The census costs one extra executor run per FAILING case — a passing case was
// decided by definition — and it decides nothing: it scores no case, writes no
// expectation, and flips a local copy of the caseSpec.

// declinesEnv names the opt-in that additionally logs the candidate case IDs
// themselves. The per-lane candidate COUNT is always logged, so the queue's size
// (and its movement at an engine-widening landing) is never invisible; the IDs
// are opt-in because a lane awaiting its milestone declines every case it claims
// and would otherwise bury the run's other reporting.
const declinesEnv = "GOXSD_DECLINES"

// declineCensus is one lane's decline audit for a single run: the partition of
// that lane's recorded failures into the ones an executor decided (absent from
// both fields) and the ones no executor decided at all.
type declineCensus struct {
	// candidates lists the cases this run declined and recorded fail, in case-ID
	// order (STYLE D1/D2). This is the harvest queue: an engine widening may have
	// already made some of them decidable without any reader edit.
	candidates []string
	// indeterminate counts the cases declined by the issue #277 convention —
	// the Working Group left them undecided, so runLane never dispatches them.
	// They are NOT candidates, because no executor may ever score them a pass,
	// but they are declines, so their number is reported rather than dropped.
	indeterminate int
}

// takeDeclineCensus re-attempts every case lane l recorded as a failure in this
// run and partitions those failures into decided and declined. cases must be the
// slice runLane consumed and actual the map it returned, so the census re-runs
// only the executor and never rediscovers the suite. Iteration is over the
// case-ID-sorted cases slice, never over actual, so the candidate list is
// deterministic (STYLE D1/D2).
func takeDeclineCensus(l lane, cases []caseSpec, actual map[string]Status) declineCensus {
	var census declineCensus
	for _, c := range cases {
		st, claimed := actual[c.id]
		if !claimed || st.IsPass() {
			continue
		}
		if c.expect.isIndeterminate() {
			census.indeterminate++
			continue
		}
		if l.exec(flipExpectation(c)).IsPass() {
			continue
		}
		census.candidates = append(census.candidates, c.id)
	}
	return census
}

// flipExpectation returns a COPY of c carrying the other decided outcome. It is
// the census probe's one moving part: an executor that decides c reaches its
// closing comparison and answers the opposite Status for the flipped copy, while
// one that declines c never reads the outcome and answers Fail for both. Only
// the caller's local copy changes; the discovered caseSpec is untouched.
//
// An indeterminate expectation has no opposite — takeDeclineCensus classifies
// such a case before probing it — so the not-valid branch simply covers it.
func flipExpectation(c caseSpec) caseSpec {
	if c.expect.wantsValid() {
		c.expect = expectInvalid()
		return c
	}
	c.expect = expectValid()
	return c
}
