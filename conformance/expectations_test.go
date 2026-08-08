package conformance

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadExpectationsMissingFileIsEmptyLane(t *testing.T) {
	path := filepath.Join(t.TempDir(), "absent.txt")
	m, err := LoadExpectations(path)
	if err != nil {
		t.Fatalf("missing file must not error: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("missing file must be empty lane, got %d entries", len(m))
	}
}

func TestLoadExpectationsParsesLinesCommentsAndBlanks(t *testing.T) {
	path := writeFile(t, `# header comment

alpha pass
beta fail   # trailing comment
   gamma   pass
`)
	m, err := LoadExpectations(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	want := map[string]Status{
		"alpha": Pass(),
		"beta":  Fail(),
		"gamma": Pass(),
	}
	if len(m) != len(want) {
		t.Fatalf("got %d entries, want %d", len(m), len(want))
	}
	for id, w := range want {
		if m[id] != w {
			t.Errorf("case %q: got %v, want %v", id, m[id], w)
		}
	}
}

func TestLoadExpectationsRejectsMalformed(t *testing.T) {
	cases := map[string]string{
		"bad status token": "alpha maybe\n",
		"too few fields":   "alpha\n",
		"too many fields":  "alpha pass extra\n",
		"duplicate case":   "alpha pass\nalpha fail\n",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			path := writeFile(t, body)
			if _, err := LoadExpectations(path); err == nil {
				t.Fatalf("expected error for %s", name)
			}
		})
	}
}

func TestCompareAllPartitions(t *testing.T) {
	expected := map[string]Status{
		"unchanged-pass": Pass(),
		"unchanged-fail": Fail(),
		"improved":       Fail(),
		"regressed":      Pass(),
		"vanished":       Pass(),
	}
	actual := map[string]Status{
		"unchanged-pass": Pass(),
		"unchanged-fail": Fail(),
		"improved":       Pass(),
		"regressed":      Fail(),
		"new":            Pass(),
	}
	d, err := Compare(expected, actual, nil)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}

	assertSlice(t, "Improved", d.Improved, []string{"improved"})
	assertSlice(t, "Regressed", d.Regressed, []string{"regressed"})
	assertSlice(t, "New", d.New, []string{"new"})
	assertSlice(t, "Vanished", d.Vanished, []string{"vanished"})
	assertSlice(t, "Removed", d.Removed, nil)
}

func TestCompareSlicesAreSorted(t *testing.T) {
	expected := map[string]Status{"c": Fail(), "a": Fail(), "b": Fail()}
	actual := map[string]Status{"c": Pass(), "a": Pass(), "b": Pass()}
	d, err := Compare(expected, actual, nil)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !slices.IsSorted(d.Improved) {
		t.Fatalf("Improved not sorted: %v", d.Improved)
	}
}

// TestCompareSeparatesSanctionedRemovalFromVanished pins the fourth change class
// (issue #576): an expected case the run no longer produced is Removed when — and
// only when — discovery NAMED it as withheld, and Vanished in every other
// circumstance. The classification is authoritative input, so the table's
// withheld column is the only thing that moves a case between the two.
func TestCompareSeparatesSanctionedRemovalFromVanished(t *testing.T) {
	cases := []struct {
		name         string
		expected     map[string]Status
		actual       map[string]Status
		withheld     []string
		wantRemoved  []string
		wantVanished []string
	}{
		{
			name:        "a withheld case with a banked fail is Removed",
			expected:    map[string]Status{"gone": Fail()},
			withheld:    []string{"gone"},
			wantRemoved: []string{"gone"},
		},
		{
			name:        "a withheld case with a banked PASS is Removed too",
			expected:    map[string]Status{"gone": Pass()},
			withheld:    []string{"gone"},
			wantRemoved: []string{"gone"},
		},
		{
			name:         "the same absence with nothing withheld stays Vanished",
			expected:     map[string]Status{"gone": Pass()},
			withheld:     nil,
			wantVanished: []string{"gone"},
		},
		{
			name:         "withholding one case does not sanction another's disappearance",
			expected:     map[string]Status{"gone": Pass(), "also-gone": Pass()},
			withheld:     []string{"gone"},
			wantRemoved:  []string{"gone"},
			wantVanished: []string{"also-gone"},
		},
		{
			name:     "a withheld id with no committed expectation is a no-op",
			expected: map[string]Status{"kept": Pass()},
			actual:   map[string]Status{"kept": Pass()},
			withheld: []string{"never-banked"},
		},
		{
			name:        "Removed is sorted and de-duplicated from an unordered withheld list",
			expected:    map[string]Status{"b": Fail(), "a": Fail()},
			withheld:    []string{"b", "a", "b"},
			wantRemoved: []string{"a", "b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.actual
			if actual == nil {
				actual = map[string]Status{}
			}
			d, err := Compare(tc.expected, actual, tc.withheld)
			if err != nil {
				t.Fatalf("compare: %v", err)
			}
			assertSlice(t, "Removed", d.Removed, tc.wantRemoved)
			assertSlice(t, "Vanished", d.Vanished, tc.wantVanished)
		})
	}
}

// TestCompareRejectsACaseDiscoveryBothWithheldAndProduced proves the overlap is
// surfaced rather than resolved by letting one side win: a runner that claims to
// have withheld a case it also produced is broken, and neither reading of it is
// safe to score.
func TestCompareRejectsACaseDiscoveryBothWithheldAndProduced(t *testing.T) {
	expected := map[string]Status{"x": Fail()}
	actual := map[string]Status{"x": Fail()}

	d, err := Compare(expected, actual, []string{"x"})
	if err == nil {
		t.Fatal("a withheld case the run also produced must error, not be silently classified")
	}
	if len(d.Removed) > 0 || len(d.Vanished) > 0 || len(d.New) > 0 {
		t.Errorf("a contradicting Delta must be empty, got %+v", d)
	}
	if _, err := Ratchet(expected, actual, []string{"x"}, RemovalAssertion{}); err == nil {
		t.Error("Ratchet must propagate the contradiction, not merge past it")
	}
}

// TestRatchetBanksAssertedRemovalsByDeletingTheirLines proves banking a
// sanctioned removal DELETES the case's line, including a line banked as `pass`.
// Merely tolerating the removal would leave the stale line behind for the next
// run to re-refuse — the permanent-freeze failure mode this class exists to end
// (issue #576, motivated by #446).
func TestRatchetBanksAssertedRemovalsByDeletingTheirLines(t *testing.T) {
	expected := map[string]Status{
		"improved":     Fail(),
		"removed-pass": Pass(),
		"removed-fail": Fail(),
	}
	actual := map[string]Status{"improved": Pass()}
	withheld := []string{"removed-fail", "removed-pass"}

	merged, err := Ratchet(expected, actual, withheld, AssertRemovals(2))
	if err != nil {
		t.Fatalf("ratchet must bank an asserted removal: %v", err)
	}
	for _, id := range withheld {
		if _, still := merged[id]; still {
			t.Errorf("case %q survived the merge: a banked removal must delete the line, not tolerate it", id)
		}
	}
	if merged["improved"] != Pass() {
		t.Errorf("the upward flip in the same run must still be banked, got %v", merged["improved"])
	}
	if len(merged) != 1 {
		t.Errorf("merged lane = %v, want only the improved case", merged)
	}
}

// TestRatchetRefusesUnassertedOrMiscountedRemovals proves the second lock: the
// asserted count never decides WHICH cases are sanctioned, it only refuses a set
// whose size the arbiter did not predict — in either direction, and including the
// zero assertion a caller that says nothing supplies.
func TestRatchetRefusesUnassertedOrMiscountedRemovals(t *testing.T) {
	expected := map[string]Status{"a": Pass(), "b": Fail()}
	actual := map[string]Status{}
	withheld := []string{"a", "b"}

	cases := []struct {
		name     string
		removals RemovalAssertion
	}{
		{"no assertion at all refuses, exactly as before #576", RemovalAssertion{}},
		{"asserting fewer than the run withheld refuses", AssertRemovals(1)},
		{"asserting more than the run withheld refuses", AssertRemovals(3)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			merged, err := Ratchet(expected, actual, withheld, tc.removals)
			if err == nil {
				t.Fatal("ratchet must refuse a removal count the run did not assert")
			}
			if merged != nil {
				t.Errorf("refusal must return nil map, got %v", merged)
			}
		})
	}

	t.Run("asserting removals that did not happen refuses too", func(t *testing.T) {
		steady := map[string]Status{"a": Pass()}
		if _, err := Ratchet(steady, steady, nil, AssertRemovals(1)); err == nil {
			t.Fatal("an assertion no removal satisfies must refuse, not pass unnoticed")
		}
	})
}

// TestRatchetRefusesRegressionAndVanishWhateverTheRemovalAssertion proves the two
// existing refusals are untouchable: a correctly asserted, correctly classified
// removal buys no tolerance for a genuine regression or a genuine vanish sharing
// the same run.
func TestRatchetRefusesRegressionAndVanishWhateverTheRemovalAssertion(t *testing.T) {
	withheld := []string{"removed"}

	regressed := map[string]Status{"scored": Pass(), "removed": Fail()}
	if _, err := Ratchet(regressed, map[string]Status{"scored": Fail()}, withheld, AssertRemovals(1)); err == nil {
		t.Error("a genuine regression must refuse the merge whatever the removal count says")
	}

	vanished := map[string]Status{"scored": Pass(), "removed": Fail()}
	if _, err := Ratchet(vanished, map[string]Status{}, withheld, AssertRemovals(1)); err == nil {
		t.Error("a genuine vanish must refuse the merge whatever the removal count says")
	}
}

func TestRatchetImprovedFlipsAndNewRecorded(t *testing.T) {
	expected := map[string]Status{
		"keep":     Pass(),
		"improved": Fail(),
	}
	actual := map[string]Status{
		"keep":     Pass(),
		"improved": Pass(),
		"new-pass": Pass(),
		"new-fail": Fail(),
	}
	merged, err := Ratchet(expected, actual, nil, RemovalAssertion{})
	if err != nil {
		t.Fatalf("ratchet: %v", err)
	}
	want := map[string]Status{
		"keep":     Pass(),
		"improved": Pass(),
		"new-pass": Pass(),
		"new-fail": Fail(),
	}
	if len(merged) != len(want) {
		t.Fatalf("got %d entries, want %d", len(merged), len(want))
	}
	for id, w := range want {
		if merged[id] != w {
			t.Errorf("case %q: got %v, want %v", id, merged[id], w)
		}
	}
}

func TestRatchetRefusesOnRegressed(t *testing.T) {
	expected := map[string]Status{"x": Pass()}
	actual := map[string]Status{"x": Fail()}
	merged, err := Ratchet(expected, actual, nil, RemovalAssertion{})
	if err == nil {
		t.Fatal("ratchet must refuse on a regressed case")
	}
	if merged != nil {
		t.Fatalf("refusal must return nil map, got %v", merged)
	}
}

func TestRatchetRefusesOnVanished(t *testing.T) {
	expected := map[string]Status{"x": Pass(), "gone": Pass()}
	actual := map[string]Status{"x": Pass()}
	merged, err := Ratchet(expected, actual, nil, RemovalAssertion{})
	if err == nil {
		t.Fatal("ratchet must refuse on a vanished case")
	}
	if merged != nil {
		t.Fatalf("refusal must return nil map, got %v", merged)
	}
}

func TestRatchetDoesNotMutateInputs(t *testing.T) {
	expected := map[string]Status{"improved": Fail()}
	actual := map[string]Status{"improved": Pass(), "new": Pass()}
	if _, err := Ratchet(expected, actual, nil, RemovalAssertion{}); err != nil {
		t.Fatalf("ratchet: %v", err)
	}
	if expected["improved"] != Fail() {
		t.Errorf("Ratchet mutated expected input")
	}
	if len(expected) != 1 {
		t.Errorf("Ratchet added key to expected input: %v", expected)
	}
}

func TestWriteExpectationsSortedAndByteStable(t *testing.T) {
	// Randomized insertion order must still yield sorted, identical bytes.
	m := map[string]Status{}
	for _, id := range []string{"delta", "alpha", "charlie", "bravo"} {
		m[id] = Pass()
	}
	m["bravo"] = Fail()

	path := filepath.Join(t.TempDir(), "lane.txt")
	if err := WriteExpectations(path, m); err != nil {
		t.Fatalf("first write: %v", err)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first: %v", err)
	}

	wantText := "alpha pass\nbravo fail\ncharlie pass\ndelta pass\n"
	if string(first) != wantText {
		t.Fatalf("output not sorted/canonical:\ngot:\n%s\nwant:\n%s", first, wantText)
	}

	if err := WriteExpectations(path, m); err != nil {
		t.Fatalf("second write: %v", err)
	}
	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read second: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("writes not byte-stable:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestWriteThenLoadRoundTrips(t *testing.T) {
	m := map[string]Status{"a": Pass(), "b": Fail(), "c": Pass()}
	path := filepath.Join(t.TempDir(), "lane.txt")
	if err := WriteExpectations(path, m); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := LoadExpectations(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != len(m) {
		t.Fatalf("round-trip changed size: got %d want %d", len(got), len(m))
	}
	for id, w := range m {
		if got[id] != w {
			t.Errorf("case %q: got %v want %v", id, got[id], w)
		}
	}
}

// TestRatchetAllWritesNoLaneUnlessEveryLaneMerged is the regression test for
// issue #581. A refusal used to cost only the lane that earned it: the run
// failed, but every sibling lane was still merged and REWRITTEN, so a single
// typo'd count left a half-banked tree behind a FAIL. The negative control that
// found it asserted schema=33 against a true 34 while instance was right; this
// is that shape at unit scale, and the assertion that matters is that the
// correct lane's file is untouched too.
func TestRatchetAllWritesNoLaneUnlessEveryLaneMerged(t *testing.T) {
	const (
		schemaBefore   = "gone fail\nkept pass\n"
		instanceBefore = "improving fail\n"
	)
	dir := t.TempDir()
	seedLane(t, dir, "schema", schemaBefore)
	seedLane(t, dir, "instance", instanceBefore)

	runs := []laneRun{
		{
			name:     "schema",
			expected: map[string]Status{"gone": Fail(), "kept": Pass()},
			actual:   map[string]Status{"kept": Pass()},
		},
		{
			name:     "instance",
			expected: map[string]Status{"improving": Fail()},
			actual:   map[string]Status{"improving": Pass()},
		},
	}
	withheld := []string{"gone"}

	// The schema lane asserts 2 removals against the 1 discovery withheld; the
	// instance lane is asserted correctly and on its own would bank an upward
	// flip.
	wrong := map[string]RemovalAssertion{"schema": AssertRemovals(2), "instance": AssertRemovals(0)}
	err := ratchetAll(dir, runs, withheld, wrong)
	if err == nil {
		t.Fatal("a removal count one lane did not assert must refuse the run")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Errorf("the refusal must name the lane that earned it: %v", err)
	}
	assertLaneFile(t, dir, "schema", schemaBefore)
	assertLaneFile(t, dir, "instance", instanceBefore)

	// Positive control: asserted right, every lane is written — the refusal
	// above withheld the write, it did not merely fail to compute one.
	right := map[string]RemovalAssertion{"schema": AssertRemovals(1)}
	if err := ratchetAll(dir, runs, withheld, right); err != nil {
		t.Fatalf("every lane merged, so every lane must be written: %v", err)
	}
	assertLaneFile(t, dir, "schema", "kept pass\n")
	assertLaneFile(t, dir, "instance", "improving pass\n")
}

// TestRatchetAllReportsEveryRefusal proves the merge pass collects refusals
// instead of ending at the first one: an arbiter correcting one lane should not
// have to re-run to discover the next lane refuses too. Neither lane's file is
// created, which is the same guarantee stated over a lane that has none yet.
func TestRatchetAllReportsEveryRefusal(t *testing.T) {
	dir := t.TempDir()
	runs := []laneRun{
		{name: "schema", expected: map[string]Status{"a": Pass()}, actual: map[string]Status{}},
		{name: "instance", expected: map[string]Status{"b": Pass()}, actual: map[string]Status{}},
	}

	err := ratchetAll(dir, runs, nil, nil)
	if err == nil {
		t.Fatal("two vanished cases in two lanes must refuse the run")
	}
	for _, name := range []string{"schema", "instance"} {
		if !strings.Contains(err.Error(), "lane "+name) {
			t.Errorf("lane %s refused but is unreported: %v", name, err)
		}
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatalf("reading lane directory: %v", readErr)
	}
	if len(entries) != 0 {
		t.Errorf("a refused run wrote %d file(s): %v", len(entries), entries)
	}
}

// seedLane writes a lane's committed file before a ratchetAll call, so the test
// can prove afterwards whether that exact content survived.
func seedLane(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(laneFileIn(dir, name), []byte(body), 0o644); err != nil {
		t.Fatalf("seeding lane %s: %v", name, err)
	}
}

// assertLaneFile checks a lane's file holds exactly want.
func assertLaneFile(t *testing.T, dir, name, want string) {
	t.Helper()
	got, err := os.ReadFile(laneFileIn(dir, name))
	if err != nil {
		t.Fatalf("reading lane %s: %v", name, err)
	}
	if string(got) != want {
		t.Errorf("lane %s:\ngot:\n%s\nwant:\n%s", name, got, want)
	}
}

func writeFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "lane.txt")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func assertSlice(t *testing.T, name string, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Errorf("%s: got %v, want %v", name, got, want)
	}
}
