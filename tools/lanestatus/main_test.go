package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLanes materializes an expectations directory for one case.
func writeLanes(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}

// TestReadLanesCountsAndOrders pins the two facts the status table rests on:
// the pass/fail split per lane, and an order that does not depend on the
// filesystem's enumeration order (STYLE D1). The `README.md` entry is the
// control — a non-lane file in the real directory, which must not be counted.
func TestReadLanesCountsAndOrders(t *testing.T) {
	dir := writeLanes(t, map[string]string{
		"schema.txt":    "b/two fail\na/one pass\nc/three pass\n",
		"datatypes.txt": "x pass\n",
		"ber.txt":       "",
		"README.md":     "not a lane\n",
	})

	scores, err := readLanes(dir)
	if err != nil {
		t.Fatalf("readLanes: %v", err)
	}

	want := []laneScore{
		{name: "ber", pass: 0, fail: 0},
		{name: "datatypes", pass: 1, fail: 0},
		{name: "schema", pass: 2, fail: 1},
	}
	if len(scores) != len(want) {
		t.Fatalf("got %d lanes, want %d (README.md must not count): %+v", len(scores), len(want), scores)
	}
	for i, w := range want {
		if scores[i] != w {
			t.Errorf("lane %d = %+v, want %+v", i, scores[i], w)
		}
	}
}

// TestReadLanesSkipsBlanksAndComments pins that the shapes the expectations
// README allows beside a case line are not counted as cases. Each line here is
// a case the tally would get wrong if the loader's skip handling were dropped.
func TestReadLanesSkipsBlanksAndComments(t *testing.T) {
	dir := writeLanes(t, map[string]string{
		"schema.txt": "# a header comment\n\na/one pass\n   \nb/two fail\n",
	})

	scores, err := readLanes(dir)
	if err != nil {
		t.Fatalf("readLanes: %v", err)
	}
	if want := (laneScore{name: "schema", pass: 1, fail: 1}); scores[0] != want {
		t.Errorf("got %+v, want %+v", scores[0], want)
	}
}

// TestReadLanesRejectsMalformed pins that a malformed line fails the run rather
// than dropping a case: an undercount in the status table is indistinguishable
// from real lane movement, and would be stamped into PLAN.md as fact.
func TestReadLanesRejectsMalformed(t *testing.T) {
	for name, body := range map[string]string{
		"one field":     "only-one-field\n",
		"three fields":  "a b c\n",
		"unknown token": "a/b/c maybe\n",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readLanes(writeLanes(t, map[string]string{"schema.txt": body})); err == nil {
				t.Errorf("readLanes on %q = nil error, want a rejection", body)
			}
		})
	}
}

// TestReadLanesMissingDirectoryIsAnError pins that a wrong -dir is operational
// failure, not an empty table that would render as "every lane unscored".
func TestReadLanesMissingDirectoryIsAnError(t *testing.T) {
	if _, err := readLanes(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("readLanes on a missing directory = nil error, want a rejection")
	}
}

// TestRenderMarkdownDistinguishesUnscoredLane pins that a lane with no cases
// renders em dashes rather than zeros — "no score yet" and "scored zero" are
// different claims, and PLAN.md's table makes both.
func TestRenderMarkdownDistinguishesUnscoredLane(t *testing.T) {
	out := renderMarkdown([]laneScore{
		{name: "datatypes", pass: 1156, fail: 17},
		{name: "xpath"},
	})

	if !strings.Contains(out, "| `datatypes` | 1156 | 17 | 1173 |") {
		t.Errorf("scored lane row missing from:\n%s", out)
	}
	if !strings.Contains(out, "| `xpath` | — | — | 0 |") {
		t.Errorf("unscored lane must render em dashes, got:\n%s", out)
	}
}
