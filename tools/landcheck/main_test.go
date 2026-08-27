package main

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// Regression fixtures are real commits from this repo's own history, not
// synthetic diffs — #304's Notes and #735 both warn that a synthetic
// docs/LOG/ fixture is exactly the kind of case that stops resembling what a
// real landing produces. Each pair below is a historical commit and its own
// parent, so `checkLanding` runs the identical git invocation it would have
// run at that landing.
const (
	// fixtureNoLogPath is #924's squash (`53bf113`): eight files changed,
	// no docs/LOG/ path at all. Its parent is `738db7a`.
	fixtureNoLogPathBase = "738db7a"
	fixtureNoLogPathHead = "53bf113"

	// fixtureWrongIssue is #813's mid-branch forward merge (`2d0a38d`): the
	// docs/LOG/ diff against its own pre-merge parent (`0e49fff`) is +161
	// lines, entirely #716's entry forward-merged in — none of it #813's.
	fixtureWrongIssueBase = "0e49fff"
	fixtureWrongIssueHead = "2d0a38d"

	// fixtureOwnEntry is #820's own squash (`311ada8`): the entry is
	// present and names #820. Its diff against its own parent (`e3d866f`)
	// also contains a literal `#8201` alongside `#820` on the same line,
	// which is what makes this fixture double as the substring-rejection
	// case, not just the pass case.
	fixtureOwnEntryBase = "e3d866f"
	fixtureOwnEntryHead = "311ada8"
)

// requireFixtures skips the test if this checkout cannot see the fixture
// commits — a shallow clone hides them (docs/ROUTINES.md), and a test that
// silently no-ops on a shallow clone is worse than one that says so.
func requireFixtures(t *testing.T, dir string, shas ...string) {
	t.Helper()
	for _, sha := range shas {
		if err := exec.Command("git", "-C", dir, "cat-file", "-e", sha).Run(); err != nil {
			t.Skipf("fixture commit %s not reachable in this checkout (shallow clone? run: git fetch --unshallow): %v", sha, err)
		}
	}
}

func TestCheckLandingRealHistory(t *testing.T) {
	dir, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
	requireFixtures(t, dir,
		fixtureNoLogPathBase, fixtureNoLogPathHead,
		fixtureWrongIssueBase, fixtureWrongIssueHead,
		fixtureOwnEntryBase, fixtureOwnEntryHead,
	)

	tests := []struct {
		name       string
		base, head string
		issue      int
		wantCode   int
	}{
		{
			name:     "924 squash carries no docs/LOG path at all: defect",
			base:     fixtureNoLogPathBase,
			head:     fixtureNoLogPathHead,
			issue:    924,
			wantCode: 1,
		},
		{
			name:     "813 mid-branch diff is entirely 716's forward-merged entry: defect",
			base:     fixtureWrongIssueBase,
			head:     fixtureWrongIssueHead,
			issue:    813,
			wantCode: 1,
		},
		{
			name:     "820 squash names its own entry: clean",
			base:     fixtureOwnEntryBase,
			head:     fixtureOwnEntryHead,
			issue:    820,
			wantCode: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			code, err := checkLanding(dir, tc.base, tc.head, tc.issue, &buf)
			if err != nil {
				t.Fatalf("checkLanding: %v", err)
			}
			if code != tc.wantCode {
				t.Errorf("code = %d, want %d; output:\n%s", code, tc.wantCode, buf.String())
			}
			if buf.Len() == 0 {
				t.Error("checkLanding wrote nothing to stdout; every path must say what it saw")
			}
		})
	}
}

// TestCheckBaseCurrentStaleBase exercises precondition 2 against real
// history: `origin/main`'s tip is never an ancestor of an old commit deep in
// its own past, which is the same shape a stale local base ref takes when a
// landing does not re-fetch before checking precondition 1.
func TestCheckBaseCurrentStaleBase(t *testing.T) {
	dir, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
	requireFixtures(t, dir, fixtureOwnEntryBase, fixtureOwnEntryHead)

	// fixtureOwnEntryHead is an ancestor of fixtureOwnEntryBase's own later
	// history only in the wrong direction: using head as the "base" and an
	// older commit as "head" makes the base NOT an ancestor, the stale-base
	// shape precondition 2 exists to catch.
	err = checkBaseCurrent(dir, fixtureOwnEntryHead, fixtureOwnEntryBase)
	if err == nil {
		t.Fatal("checkBaseCurrent: want an error when base is not an ancestor of head")
	}
	if !strings.Contains(err.Error(), "not current") {
		t.Errorf("error = %q, want it to say the base is not current", err.Error())
	}
}

// TestCheckLandingStaleBaseExitsOperational checks that checkLanding treats
// a stale base as an error regardless of what precondition 1 would have
// found — precondition 2 gates precondition 1, per the issue's Acceptance
// item 2.
func TestCheckLandingStaleBaseExitsOperational(t *testing.T) {
	dir, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
	requireFixtures(t, dir, fixtureOwnEntryBase, fixtureOwnEntryHead)

	var buf bytes.Buffer
	_, err = checkLanding(dir, fixtureOwnEntryHead, fixtureOwnEntryBase, 820, &buf)
	if err == nil {
		t.Fatal("checkLanding: want an error for a stale base, even though 820's own entry is present in the reverse diff")
	}
}

// TestIssueTokenPattern is the pure regression case for the token-boundary
// rule: #820 must match only as a whole token, never as a prefix of a
// longer number.
func TestIssueTokenPattern(t *testing.T) {
	p := issueTokenPattern(820)

	matches := []string{
		"landed at (#820)",
		"see #820 for context",
		"tracked in issues/820",
		"`#820` inside a future `#8201`", // the literal #820 token still matches here
	}
	for _, s := range matches {
		if !p.MatchString(s) {
			t.Errorf("issueTokenPattern(820).MatchString(%q) = false, want true", s)
		}
	}

	nonMatches := []string{
		"#8201",
		"issues/8201",
		"#96820",
		"no reference here",
	}
	for _, s := range nonMatches {
		if p.MatchString(s) {
			t.Errorf("issueTokenPattern(820).MatchString(%q) = true, want false", s)
		}
	}
}

// TestAddedLines checks the diff-parsing helper against literal unified diff
// text, including the "+++" file-header line a naive "+" prefix check would
// misread as an added content line.
func TestAddedLines(t *testing.T) {
	diff := `diff --git a/docs/LOG/2026-08.md b/docs/LOG/2026-08.md
index 111..222 100644
--- a/docs/LOG/2026-08.md
+++ b/docs/LOG/2026-08.md
@@ -1,2 +1,3 @@
 unchanged line
-removed line
+added line one (#820)
+added line two
`
	got := addedLines(diff)
	want := []string{"added line one (#820)", "added line two"}
	if len(got) != len(want) {
		t.Fatalf("addedLines = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("addedLines[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestAddedLinesEmptyDiff checks the #924 shape directly: no docs/LOG/ path
// touched at all means git prints nothing, and addedLines must report no
// added lines rather than one empty line.
func TestAddedLinesEmptyDiff(t *testing.T) {
	if got := addedLines(""); len(got) != 0 {
		t.Errorf("addedLines(\"\") = %+v, want none", got)
	}
}
