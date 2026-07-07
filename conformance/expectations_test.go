package conformance

import (
	"os"
	"path/filepath"
	"slices"
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
	d := Compare(expected, actual)

	assertSlice(t, "Improved", d.Improved, []string{"improved"})
	assertSlice(t, "Regressed", d.Regressed, []string{"regressed"})
	assertSlice(t, "New", d.New, []string{"new"})
	assertSlice(t, "Vanished", d.Vanished, []string{"vanished"})
}

func TestCompareSlicesAreSorted(t *testing.T) {
	expected := map[string]Status{"c": Fail(), "a": Fail(), "b": Fail()}
	actual := map[string]Status{"c": Pass(), "a": Pass(), "b": Pass()}
	d := Compare(expected, actual)
	if !slices.IsSorted(d.Improved) {
		t.Fatalf("Improved not sorted: %v", d.Improved)
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
	merged, err := Ratchet(expected, actual)
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
	merged, err := Ratchet(expected, actual)
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
	merged, err := Ratchet(expected, actual)
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
	if _, err := Ratchet(expected, actual); err != nil {
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
