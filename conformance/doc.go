// Package conformance runs the W3C XSD test suite against this processor
// and enforces the ratchet: conformance only moves up.
//
// This package is test-only. Nothing in the library imports it.
//
// # The suite
//
// The W3C suite lives at testdata/xsdtests (a pinned git submodule;
// populate with `git submodule update --init testdata/xsdtests`). Its
// index is suite.xml, which references test sets; each test group carries
// schema tests and instance tests with declared expected outcomes. The
// auxiliary extra-suite.xml sibling is discovered alongside it (issue #135):
// it carries the precisionDecimal test sets the W3C suite moved out of
// suite.xml when the type left XSD 1.1, and shares one test set
// (common/introspection.testSet) with the main index, discovered once.
//
// A test set, test group, schema test or instance test may declare a
// `version` attribute listing the versions and FEATURES it applies to,
// OR-connected: a processor supporting any listed token runs it. A level
// listing no token this processor supports is not applicable and yields no
// cases — it is absent from the run and from every expectation file, not
// declined and not scored (issue #446). This processor supports exactly the
// token `1.1`, so groups scoped to XSD 1.0 only, and groups scoped to the
// `full-xpath-in-CTA` feature whose engine is unlanded, are both out of
// scope. An absent `version` applies to everything. This is unrelated to
// `expected/@version`, whose tokens are AND-connected and only choose which
// declared outcome binds.
//
// # Lanes and expectation files
//
// Expectations are committed at conformance/testdata/expectations/, one
// lane per file:
//
//	datatypes.txt   simple-type / facet cases            (from M3)
//	schema.txt      schema-validity cases                (from M4)
//	instance.txt    instance-validity cases              (from M5)
//	xpath.txt       XPath engine cases                   (from M7)
//	json.txt        JSON-adapter cases (curated)         (from M8)
//	ber.txt         BER-adapter cases (curated)          (from M11)
//
// File format: one case per line, `<case-id> <pass|fail>`, sorted by case
// ID; `#` starts a comment. `pass` means this processor agrees with the
// suite's declared outcome; `fail` records a known gap so a regression is
// loud and an improvement is harvestable. A missing lane file is an empty
// lane, not an error. Expectation files are machine-written only — never
// edited by hand, and NEVER edited downward.
//
// A suite case may declare validity="indeterminate": the Working Group could
// not agree the case has one right answer, and the suite's own catalog DTD
// makes that a category disjoint from valid|invalid. Such a case is DECLINED —
// no executor is run for it and it is always recorded `fail` — so it can never
// score a pass (issue #277). Scoring it as if it meant "invalid" would have
// credited this processor for rejecting a document nobody agrees should be
// rejected. The decline is a harness-scoring convention, not a spec
// requirement: XSD 1.1 makes [validity] three-valued and states that schema
// validity is not a binary predicate, so there is no basis for equating
// indeterminate with invalid.
//
// # Mechanics (the M1 implementation contract)
//
//	LoadExpectations(path) (map[string]Status, error)
//	    Missing file => empty lane.
//
//	Compare(expected, actual) Delta
//	    Delta partitions cases into Improved (expected fail, now pass),
//	    Regressed (expected pass, now fail — never acceptable), New (no
//	    expectation yet), and Vanished (expected case the run no longer
//	    produced).
//
//	Ratchet(expected, actual) (map[string]Status, error)
//	    Upward-only merge: Improved flips to pass, New is recorded at its
//	    observed status. Any Regressed or Vanished case aborts the entire
//	    merge with an error — the ratchet refuses to move at all rather
//	    than record a downgrade.
//
//	WriteExpectations(path, m) error
//	    Always sorted by case ID (STYLE D1/D2).
//
// # Running
//
//	go test ./conformance -run TestConformance -count=1
//	    Read-only: runs the suite, Compares against committed
//	    expectations, fails on any Regressed case. Improved cases are
//	    logged (visible with -v), never written: a non-arbiter agent can
//	    thus see the upward movement pending banking (issue #303).
//
//	GOXSD_RATCHET=1 go test ./conformance -run TestConformance -count=1
//	    Additionally Ratchets each lane and rewrites its file. Arbiter
//	    only (see docs/WORKFLOW.md); every flipped case must be
//	    explainable by the diff under judgment.
//
// The runner supports single-case reproduction for debugging:
// GOXSD_CASE=<case-id> narrows the run to one case with debug logging.
//
// # The decline census
//
// An executor either DECIDES a case — it observes this processor's outcome and
// compares that with the suite's declared one — or DECLINES it, because no
// reader recognized the fixture's shape. Both record `fail`, so a decline is a
// snapshot of engine capability taken when the decline code was WRITTEN, and
// nothing re-checked it: d3_3_4v16 stayed wrongly declined for nine days after
// the capability that covers it landed, until an unrelated reader edit stumbled
// over it (issue #327).
//
// Every run therefore re-attempts each recorded failure with its declared
// outcome flipped, which separates decided failures (the flipped run passes)
// from declines (Fail under both polarities), and logs the per-lane decline
// count (visible with -v). That count is the harvest queue's size, so it moves
// visibly at the landing that widens an engine. The census reports only: it
// scores no case and writes no expectation.
//
//	GOXSD_DECLINES=1
//	    Additionally logs the candidate case IDs themselves, sorted. Opt-in
//	    because a lane still awaiting its milestone declines every case it
//	    claims and would bury the run's other reporting.
//
// # Missing suite
//
// TestConformance FAILS when testdata/xsdtests is not populated, naming
// `git submodule update --init testdata/xsdtests` (issue #309). A run that
// executed zero cases must never be reportable as a green run, and a ratchet
// run over an empty suite must never be reportable as "no movement".
//
//	GOXSD_SUITE_OPTIONAL=1
//	    Explicit opt-out for an environment that legitimately has no suite
//	    checkout: the missing-suite failure becomes a skip. It applies to
//	    read-only runs ONLY — under GOXSD_RATCHET=1 the run still fails.
//
// The supplementary fixture-driven tests in this package (datatypes_test.go,
// schema_test.go) keep their plain t.Skipf on a missing suite, unconditionally.
// They are deliberately treated differently from TestConformance: each drives a
// named cohort of suite documents to prove one executor decides them for the
// right reason, so skipping them loses unit coverage but cannot dress an empty
// run up as suite-wide conformance. TestConformance is the only test whose
// silence was mistakable for a score.
package conformance
