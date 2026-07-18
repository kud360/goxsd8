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
//	    expectations, fails on any Regressed case.
//
//	GOXSD_RATCHET=1 go test ./conformance -run TestConformance -count=1
//	    Additionally Ratchets each lane and rewrites its file. Arbiter
//	    only (see docs/WORKFLOW.md); every flipped case must be
//	    explainable by the diff under judgment.
//
// The runner supports single-case reproduction for debugging:
// GOXSD_CASE=<case-id> narrows the run to one case with debug logging.
package conformance
