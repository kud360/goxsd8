package conformance

import (
	"bufio"
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
// run into the four disjoint change classes the ratchet reasons about. Each
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
	// Vanished lists expected cases the run no longer produced — an expected
	// case that silently disappeared is treated as a regression by the ratchet.
	Vanished []string
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
// into a Delta. The four classes are disjoint and every listed slice is sorted
// (STYLE D1/D2): Improved (expected fail, now pass), Regressed (expected pass,
// now fail), New (observed but unexpected), and Vanished (expected but no
// longer observed).
func Compare(expected, actual map[string]Status) Delta {
	var d Delta
	for id, want := range expected {
		got, ok := actual[id]
		if !ok {
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
	return d
}

// Ratchet computes the upward-only merge of expectations with an observed run.
// Improved cases flip to pass and New cases are recorded at their observed
// status; unchanged cases keep their expectation. Any Regressed or Vanished
// case aborts the entire merge with an error — the ratchet refuses to move at
// all rather than record a downgrade. The input maps are never mutated.
func Ratchet(expected, actual map[string]Status) (map[string]Status, error) {
	d := Compare(expected, actual)
	if len(d.Regressed) > 0 || len(d.Vanished) > 0 {
		return nil, fmt.Errorf(
			"ratchet refuses to move: %d regressed %v, %d vanished %v",
			len(d.Regressed), d.Regressed, len(d.Vanished), d.Vanished)
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
	return merged, nil
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
