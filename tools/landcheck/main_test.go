package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// Regression fixtures are real commits from this repo's own history, not
// synthetic diffs — #304's Notes and #735 both warn that a synthetic
// docs/LOG/ fixture is exactly the kind of case that stops resembling what a
// real landing produces. Each pair below is a historical commit and its own
// parent, so `checkLanding` runs the identical git invocation it would have
// run at that landing.
//
// The SHAs are pinned in full 40-character form because `git fetch origin`
// takes a hash and not a prefix: #813's pair sits on a wip branch that was
// deleted, no ref reaches it at any depth, and a fetch by full hash is the
// only way to get it back — which an abbreviated constant denies even to a
// reader sitting in the checkout (#1359).
const (
	// fixtureNoLogPath is #924's squash (`53bf113`): eight files changed,
	// no docs/LOG/ path at all. Its parent is `738db7a`.
	fixtureNoLogPathBase = "738db7a6caa56d501858d18b6445d415eafabeef"
	fixtureNoLogPathHead = "53bf1130f18811b9fd83586805135f1eb381656c"

	// fixtureWrongIssue is #813's mid-branch forward merge (`2d0a38d`): the
	// docs/LOG/ diff against its own pre-merge parent (`0e49fff`) is +161
	// lines, entirely #716's entry forward-merged in — none of it #813's.
	fixtureWrongIssueBase = "0e49ffffe999a5718decc047d4e49630e32f6079"
	fixtureWrongIssueHead = "2d0a38dcc9e164c22bd5e5e6917c1e2e94bd90c7"

	// fixtureOwnEntry is #820's own squash (`311ada8`): the entry is
	// present and names #820. Its diff against its own parent (`e3d866f`)
	// also contains a literal `#8201` alongside `#820` on the same line,
	// which is what makes this fixture double as the substring-rejection
	// case, not just the pass case.
	fixtureOwnEntryBase = "e3d866f93454ea59010852f8fa5090022e70f726"
	fixtureOwnEntryHead = "311ada8b900f27bdf934bd5d0856993f48058b8f"
)

// fixtureFetchTimeout bounds each fetch below. Without it an unreachable
// origin costs the whole `go test` timeout and reports a panic rather than a
// diagnosis.
const fixtureFetchTimeout = 60 * time.Second

// requireFixtures makes one fixture pair usable in this checkout, fetching
// the two commits from origin by full hash when it is not, and fails the
// calling test when it still cannot. It never skips: a skip leaves the gate
// reading `ok` over cases that did not run. Shallowness is no exception — a
// fetch by full hash recovers these commits in a shallow checkout as readily
// as in a complete one, so a shallow escape hatch would restore that silence
// in the containers that most need the check (#1359).
//
// Call it inside the subtest that needs the pair, never once over every
// fixture: one unreachable pair must not take down the cases whose commits
// are already in the object store.
func requireFixtures(t *testing.T, dir, base, head string) {
	t.Helper()
	unusable := fixtureUsable(dir, base, head)
	if unusable == nil {
		return
	}
	out, err := fetchCommits(dir, base, head)
	if err != nil {
		t.Fatalf("fixture pair %s..%s cannot run here: %v\ngit fetch origin %s %s: %v\n%s%s"+
			"every recovery path here goes through origin, so check this environment's access to it first",
			base, head, unusable, base, head, err, out, recoveryHint(dir))
	}
	if remaining := fixtureUsable(dir, base, head); remaining != nil {
		t.Fatalf("fixture pair %s..%s is still unusable after git fetch origin reported success: %v\n%s%s"+
			"origin served these objects but not the history linking them; re-run, or fetch the two hashes by hand",
			base, head, remaining, out, recoveryHint(dir))
	}
}

// fixtureUsable reports what stops checkLanding from running against the
// pair, or nil when nothing does. Object presence is not the bar: a commit
// fetched at --depth=1 resolves while its parent links do not, and the pair
// then fails precondition 2 as a stale base — a defect verdict pinned on
// what is really a fixture problem.
func fixtureUsable(dir, base, head string) error {
	for _, sha := range []string{base, head} {
		if err := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", sha+"^{commit}").Run(); err != nil {
			return fmt.Errorf("commit %s does not resolve in this checkout: %w", sha, err)
		}
	}
	if err := exec.Command("git", "-C", dir, "merge-base", "--is-ancestor", base, head).Run(); err != nil {
		return fmt.Errorf("commit %s is not visible as an ancestor of %s, so the history between them is truncated here: %w", base, head, err)
	}
	return nil
}

// fetchCommits asks origin for these two commits by hash. The fetch carries
// no --depth on purpose: --depth=1 returns the objects without their parent
// links, leaving the pair resolvable and still unusable, and on a complete
// clone any --depth would newly truncate history the developer had (#1359).
func fetchCommits(dir, base, head string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fixtureFetchTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "fetch", "origin", base, head)
	// A credential prompt would block on a stdin no test is watching.
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}
	if ctx.Err() != nil {
		return string(out), fmt.Errorf("timed out after %s: %w", fixtureFetchTimeout, err)
	}
	return string(out), err
}

// recoveryHint states what this checkout is, measured now rather than
// assumed. Truncated depth and absence-from-every-ref are different faults
// with different remedies, and a message naming only the first misdiagnoses
// the pair that trips this guard most often (#1359).
func recoveryHint(dir string) string {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--is-shallow-repository").Output()
	if err != nil {
		return fmt.Sprintf("observed: could not measure whether this checkout is shallow: %v\n", err)
	}
	if strings.TrimSpace(string(out)) == "true" {
		return "observed: this checkout IS shallow (git rev-parse --is-shallow-repository = true). " +
			"`git fetch --unshallow origin` deepens along the refs that exist, so it recovers a commit still " +
			"on a live branch and never one whose branch was deleted.\n"
	}
	return "observed: this checkout is NOT shallow (git rev-parse --is-shallow-repository = false), so depth " +
		"is not the mechanism and `git fetch --unshallow origin` has nothing to deepen: the commit is " +
		"reachable from no ref here.\n"
}

func TestCheckLandingRealHistory(t *testing.T) {
	dir, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
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
			requireFixtures(t, dir, tc.base, tc.head)

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
