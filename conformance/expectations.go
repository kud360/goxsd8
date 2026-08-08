package conformance

import (
	"bufio"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
)

// Status is the recorded outcome for a single conformance case: whether this
// processor agrees with the suite's declared outcome (Pass) or a known gap is
// recorded so a regression is loud and an improvement is harvestable (Fail).
// It is a closed set (STYLE T1); construct values only via Pass and Fail.
type Status struct {
	pass bool
}

// Pass reports agreement with the suite's declared outcome for a case.
func Pass() Status { return Status{pass: true} }

// Fail records a known gap: this processor does not yet match the suite's
// declared outcome for a case.
func Fail() Status { return Status{pass: false} }

// IsPass reports whether the status is a pass. It is the derived accessor for
// the sole fact a Status carries (STYLE D3), used by the ratchet mechanics and
// by callers formatting a lane.
func (s Status) IsPass() bool { return s.pass }

// String renders the status in expectation-file token form ("pass" or "fail").
func (s Status) String() string {
	if s.pass {
		return "pass"
	}
	return "fail"
}

// Delta partitions a comparison of committed expectations against an observed
// run into the five disjoint change classes the ratchet reasons about. Each
// field lists case IDs in sorted order (STYLE D1); a case appears in at most
// one field. Cases that are expected and still observed at the same status are
// unchanged and appear in no field.
type Delta struct {
	// Improved lists cases expected to fail that the run now passes — the
	// harvestable wins the ratchet flips upward.
	Improved []string
	// Regressed lists cases expected to pass that the run now fails — never
	// acceptable; their presence forbids any ratchet movement.
	Regressed []string
	// New lists observed cases that carry no committed expectation yet.
	New []string
	// Vanished lists expected cases the run no longer produced and that
	// discovery did NOT claim to withhold — an expected case that silently
	// disappeared is treated as a regression by the ratchet.
	Vanished []string
	// Removed lists expected cases the run no longer produced BECAUSE
	// discovery ruled them inapplicable: the W3C suite's own applicability
	// metadata scopes them away from this XSD 1.1 processor, so they were
	// never this processor's cases to score (issue #576). This is a
	// sanctioned removal, not a regression — but only the arbiter's ratchet
	// run, asserting how many it expects, may bank one; see Ratchet.
	Removed []string
}

// LoadExpectations reads a lane's committed expectation file into a map keyed by
// case ID. A missing file is an empty lane, not an error (per conformance
// doc.go and the expectations README). Each non-blank, non-comment line is
// `<case-id> <pass|fail>`; `#` starts a comment and blank lines are allowed.
func LoadExpectations(path string) (map[string]Status, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]Status{}, nil
		}
		return nil, fmt.Errorf("opening expectations %s: %w", path, err)
	}
	defer func() { _ = f.Close() }() // read-only handle: close error cannot affect parsed result

	m := map[string]Status{}
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		id, status, skip, perr := parseExpectationLine(sc.Text())
		if perr != nil {
			return nil, fmt.Errorf("parsing %s line %d: %w", path, line, perr)
		}
		if skip {
			continue
		}
		if _, dup := m[id]; dup {
			return nil, fmt.Errorf("parsing %s line %d: duplicate case %q", path, line, id)
		}
		m[id] = status
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("reading expectations %s: %w", path, err)
	}
	return m, nil
}

// parseExpectationLine parses one raw line. skip is true for blank or
// comment-only lines, which carry no case.
func parseExpectationLine(raw string) (id string, status Status, skip bool, err error) {
	text := raw
	if hash := strings.IndexByte(text, '#'); hash >= 0 {
		text = text[:hash]
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", Status{}, true, nil
	}
	if len(fields) != 2 {
		return "", Status{}, false, fmt.Errorf("want `<case-id> <pass|fail>`, got %d fields", len(fields))
	}
	status, err = parseStatus(fields[1])
	if err != nil {
		return "", Status{}, false, err
	}
	return fields[0], status, false, nil
}

// parseStatus maps an expectation-file token to a Status.
func parseStatus(tok string) (Status, error) {
	switch tok {
	case "pass":
		return Pass(), nil
	case "fail":
		return Fail(), nil
	default:
		return Status{}, fmt.Errorf("want `pass` or `fail`, got %q", tok)
	}
}

// Compare partitions the observed run (actual) against committed expectations
// into a Delta. The five classes are disjoint and every listed slice is sorted
// (STYLE D1/D2): Improved (expected fail, now pass), Regressed (expected pass,
// now fail), New (observed but unexpected), Removed (expected, and absent
// because discovery withheld it) and Vanished (expected, absent, and NOT
// withheld).
//
// withheld carries the case IDs discovery deliberately did not produce because
// the suite's own applicability metadata scopes them away from this processor
// (conformance/runner.go). It is AUTHORITATIVE input from the runner, never a
// heuristic Compare infers from the diff: an expected-but-absent case is Removed
// only when the runner named it, and Vanished in every other circumstance. Order
// and duplicates within withheld are insignificant, and an ID listed there that
// carries no committed expectation is a no-op rather than an error.
//
// A withheld ID the run nevertheless observed is a runner bug — discovery
// claiming both to withhold and to produce one case — and is reported as an
// error rather than resolved by letting one side win.
func Compare(expected, actual map[string]Status, withheld []string) (Delta, error) {
	held := make(map[string]struct{}, len(withheld))
	var contradicted []string
	for _, id := range withheld {
		held[id] = struct{}{}
		if _, observed := actual[id]; observed {
			contradicted = append(contradicted, id)
		}
	}
	if len(contradicted) > 0 {
		slices.Sort(contradicted)
		contradicted = slices.Compact(contradicted)
		return Delta{}, fmt.Errorf(
			"discovery both withheld and produced %d case(s): %v",
			len(contradicted), contradicted)
	}

	var d Delta
	for id, want := range expected {
		got, observed := actual[id]
		if !observed {
			if _, sanctioned := held[id]; sanctioned {
				d.Removed = append(d.Removed, id)
				continue
			}
			d.Vanished = append(d.Vanished, id)
			continue
		}
		if want.IsPass() && !got.IsPass() {
			d.Regressed = append(d.Regressed, id)
			continue
		}
		if !want.IsPass() && got.IsPass() {
			d.Improved = append(d.Improved, id)
		}
	}
	for id := range actual {
		if _, ok := expected[id]; !ok {
			d.New = append(d.New, id)
		}
	}
	slices.Sort(d.Improved)
	slices.Sort(d.Regressed)
	slices.Sort(d.New)
	slices.Sort(d.Vanished)
	slices.Sort(d.Removed)
	return d, nil
}

// RemovalAssertion is the arbiter's prediction of how many sanctioned
// applicability removals one lane's ratchet run will bank (issue #576). It is a
// SECOND, independent lock, not the classifier: it never decides which cases are
// sanctioned — the runner's withheld set does that — it only refuses to bank a
// set whose size the arbiter did not predict. Construct values only via
// AssertRemovals (STYLE T7).
//
// The zero value asserts none, which is the pre-#576 behaviour: a caller that
// says nothing refuses any removal at all. There is no "did the caller assert?"
// flag beside the count, because asserting zero and asserting nothing permit
// exactly the same set — one fact, one encoding (STYLE D3).
type RemovalAssertion struct {
	n int
}

// AssertRemovals is the arbiter's assertion that exactly n sanctioned
// applicability removals are expected in this lane. Ratchet refuses the whole
// merge on any other number, so the count is checked machinery rather than a
// claim an agent can make in prose.
func AssertRemovals(n int) RemovalAssertion { return RemovalAssertion{n: n} }

// Ratchet computes the upward-only merge of expectations with an observed run.
// Improved cases flip to pass and New cases are recorded at their observed
// status; unchanged cases keep their expectation. The input maps are never
// mutated.
//
// Three conditions abort the entire merge — the ratchet refuses to move at all
// rather than record a downgrade:
//
//   - any Regressed or Vanished case, unconditionally and whatever removals
//     asserts;
//   - a withheld ID the run also produced (Compare's runner-bug error);
//   - a Removed count other than the one removals asserts, in either direction,
//     which for the zero RemovalAssertion means any removal at all.
//
// Banking a sanctioned removal DELETES the case's line from the merged map: a
// removal that were merely tolerated would be re-offered, and re-refused, on
// every subsequent run.
func Ratchet(expected, actual map[string]Status, withheld []string, removals RemovalAssertion) (map[string]Status, error) {
	d, err := Compare(expected, actual, withheld)
	if err != nil {
		return nil, fmt.Errorf("ratchet refuses to move: %w", err)
	}
	if len(d.Regressed) > 0 || len(d.Vanished) > 0 {
		return nil, fmt.Errorf(
			"ratchet refuses to move: %d regressed %v, %d vanished %v",
			len(d.Regressed), d.Regressed, len(d.Vanished), d.Vanished)
	}
	if len(d.Removed) != removals.n {
		return nil, fmt.Errorf(
			"ratchet refuses to move: run asserted %d sanctioned applicability removal(s), found %d %v",
			removals.n, len(d.Removed), d.Removed)
	}

	merged := maps.Clone(expected)
	if merged == nil {
		merged = map[string]Status{}
	}
	for _, id := range d.Improved {
		merged[id] = Pass()
	}
	for _, id := range d.New {
		merged[id] = actual[id]
	}
	for _, id := range d.Removed {
		delete(merged, id)
	}
	return merged, nil
}

// laneRun is one lane's completed run: the lane's name, the expectations
// committed for it, and the statuses this run observed. It carries a lane from
// the observation pass to the merge, so every lane can be merged before any
// lane is written (ratchetAll). Nothing derivable is stored (STYLE D3): the
// lane's file comes from its name, and the withheld set is the run's rather
// than any one lane's.
type laneRun struct {
	name     string
	expected map[string]Status
	actual   map[string]Status
}

// ratchetAll merges every lane and then writes every lane, in two separated
// phases: no lane's file is written unless EVERY lane merged without error.
// Ratchet is a per-lane function and refuses only the merge it was asked for;
// this is what makes one lane's refusal cost the whole run, so a single wrong
// removal assertion can no longer leave a half-banked tree behind a failing run
// (issue #581). Every refusal is collected rather than the first one ending the
// pass — an arbiter correcting one asserted count should see all of them at
// once, and none of them wrote anything.
//
// runs is a slice so lanes merge and write in the caller's fixed order (STYLE
// D1); removals is an internal lookup keyed by lane name, read by key and never
// iterated (STYLE D2), so a lane it does not name asserts the zero
// RemovalAssertion. dir is the directory holding the committed lane files.
//
// The write phase is not itself transactional: WriteExpectations renames each
// lane's file into place separately, so an I/O fault partway through can still
// leave earlier lanes written. That is a failing disk rather than a refused
// merge — the refusals this function exists to contain are all decided in the
// first phase, before anything is opened for writing.
func ratchetAll(dir string, runs []laneRun, withheld []string, removals map[string]RemovalAssertion) error {
	merged := make([]map[string]Status, len(runs))
	var refused []error
	for i, r := range runs {
		m, err := Ratchet(r.expected, r.actual, withheld, removals[r.name])
		if err != nil {
			refused = append(refused, fmt.Errorf("lane %s: %w", r.name, err))
			continue
		}
		merged[i] = m
	}
	if len(refused) > 0 {
		return fmt.Errorf("no lane was written: %w", errors.Join(refused...))
	}

	for i, r := range runs {
		if err := WriteExpectations(laneFileIn(dir, r.name), merged[i]); err != nil {
			return fmt.Errorf("lane %s: writing expectations: %w", r.name, err)
		}
	}
	return nil
}

// WriteExpectations writes a lane's expectations to path, one case per line as
// `<case-id> <pass|fail>`, always sorted by case ID so identical inputs produce
// byte-identical files (STYLE D1/D2). The write is atomic: it renders to a temp
// file in the same directory and renames it into place.
func WriteExpectations(path string, m map[string]Status) error {
	var b strings.Builder
	for _, id := range slices.Sorted(maps.Keys(m)) {
		fmt.Fprintf(&b, "%s %s\n", id, m[id])
	}

	tmp, err := os.CreateTemp(dirOf(path), ".expectations-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }() // best-effort cleanup; no-op once renamed into place

	if _, err := tmp.WriteString(b.String()); err != nil {
		_ = tmp.Close() // write already failed; the write error is the one that matters
		return fmt.Errorf("writing temp for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp into %s: %w", path, err)
	}
	return nil
}

// dirOf returns the directory to hold path's temp file. An empty directory
// (path has no separator) means the current directory, which os.CreateTemp
// accepts as "".
func dirOf(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return ""
}
