package main

import (
	"bytes"
	"os"
	"os/exec"
	"slices"
	"testing"
)

// The closing-keyword fixtures are real landing texts, the two sightings
// #1178 records. `2355fe0`'s squash text is read from git by full hash, as
// the fixture pairs in main_test.go are; PR #1706's description lives only
// on GitHub, so testdata/pr1706-body.md holds it byte for byte, as `gh api
// repos/kud360/goxsd8/pulls/1706` returns its `.body`.
const (
	// fixtureCommaForm is #1023's squash (`2355fe0`): `Closed #625, #748,
	// …` heads a comma list, and a later note names the same five as "not
	// closing keywords". Its parent is `46ecc79`.
	fixtureCommaFormBase = "46ecc791d8ca7c9007e55cddd9f124669699b0f9"
	fixtureCommaFormHead = "2355fe0ce016aefd61602182e0d31ec5e3cb1e7a"

	fixturePR1706Body = "testdata/pr1706-body.md"
)

// commitText returns sha's full commit text, subject and body, as
// `git log -1 --format=%B` prints it — docs/WORKFLOW.md precondition 4's
// own reading of a squash.
func commitText(t *testing.T, dir, sha string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "log", "-1", "--format=%B", sha).Output()
	if err != nil {
		t.Fatalf("git log -1 --format=%%B %s: %v", sha, err)
	}
	return string(out)
}

func TestCheckClosingKeywordsRealHistory(t *testing.T) {
	dir, err := repoRoot()
	if err != nil {
		t.Fatalf("repoRoot: %v", err)
	}
	pr1706, err := os.ReadFile(fixturePR1706Body)
	if err != nil {
		t.Fatalf("reading %s: %v", fixturePR1706Body, err)
	}

	t.Run("2355fe0 squash: comma form and contradiction on #625, #1023 clean", func(t *testing.T) {
		requireFixtures(t, dir, fixtureCommaFormBase, fixtureCommaFormHead)
		texts := []landingText{{name: "squash text", body: commitText(t, dir, fixtureCommaFormHead)}}
		var buf bytes.Buffer
		code, err := checkClosingKeywords(texts, false, &buf)
		if err != nil {
			t.Fatalf("checkClosingKeywords: %v", err)
		}
		want := `landcheck: squash text: "Closed #625" is the comma form: a further reference follows #625, and the keyword closes only #625
landcheck: squash text: "Closed #625" binds #625, which a "names and leaves open" or "not closing keywords" clause also names
`
		if code != 1 || buf.String() != want {
			t.Errorf("code = %d, output:\n%s\nwant code 1, output:\n%s", code, buf.String(), want)
		}
	})

	t.Run("2355fe0 squash: the contradiction alone names all five", func(t *testing.T) {
		requireFixtures(t, dir, fixtureCommaFormBase, fixtureCommaFormHead)
		body := commitText(t, dir, fixtureCommaFormHead)
		got, err := leftOpen(body)
		if err != nil {
			t.Fatalf("leftOpen: %v", err)
		}
		if want := []int{625, 748, 492, 934, 896}; !slices.Equal(got, want) {
			t.Errorf("leftOpen = %v, want %v", got, want)
		}
	})

	t.Run("PR 1706 description closing an issue: Fixes #345's binds, nothing else wrong", func(t *testing.T) {
		var buf bytes.Buffer
		code, err := checkClosingKeywords([]landingText{{name: "PR description", body: string(pr1706)}}, false, &buf)
		if err != nil {
			t.Fatalf("checkClosingKeywords: %v", err)
		}
		want := "landcheck: PR description: closing keywords bind #345\n"
		if code != 0 || buf.String() != want {
			t.Errorf("code = %d, output:\n%s\nwant code 0, output:\n%s", code, buf.String(), want)
		}
	})

	t.Run("PR 1706 description in a PR closing no issue: rejected", func(t *testing.T) {
		var buf bytes.Buffer
		code, err := checkClosingKeywords([]landingText{{name: "PR description", body: string(pr1706)}}, true, &buf)
		if err != nil {
			t.Fatalf("checkClosingKeywords: %v", err)
		}
		want := "landcheck: PR description: \"Fixes #345\" binds #345 in a PR that closes no issue\n"
		if code != 1 || buf.String() != want {
			t.Errorf("code = %d, output:\n%s\nwant code 1, output:\n%s", code, buf.String(), want)
		}
	})
}

// TestCheckClosingKeywords pins each matching rule #1178's Acceptance binds
// on synthetic text: the per-reference contradiction, line breaks inside
// keyword, reference and phrase, `'s` after a reference, and the no-issue
// mode.
func TestCheckClosingKeywords(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		closesNone bool
		wantCode   int
		wantOutput string
	}{
		{
			name:       "one Closes sentence: clean",
			body:       "tools: check closing keywords (#1178)\n\nCloses #1178.\n",
			wantOutput: "landcheck: t: closing keywords bind #1178\n",
		},
		{
			name:       "names and leaves open a different issue: clean",
			body:       "Closes #625. Names and leaves open #830.",
			wantOutput: "landcheck: t: closing keywords bind #625\n",
		},
		{
			name:       "one Closes sentence per issue: clean",
			body:       "Closes #669. Closes #625. Closes #748.",
			wantOutput: "landcheck: t: closing keywords bind #669 #625 #748\n",
		},
		{
			name:       "a longer number sharing the prefix is another issue: clean",
			body:       "Closes #820. Names and leaves open #8201.",
			wantOutput: "landcheck: t: closing keywords bind #820\n",
		},
		{
			name:       "a keyword inside a word or before a non-reference binds nothing: clean",
			body:       "Enclosed #5 is prefixed. Closes PR #1689 as superseded. Fixed-width #7.",
			wantOutput: "landcheck: t: closing keywords bind nothing\n",
		},
		{
			name:       "separate sentences are separate clauses: clean",
			body:       "Closes #625. See #625's thread! Names and leaves open #830.",
			wantOutput: "landcheck: t: closing keywords bind #625\n",
		},
		{
			name:       "separate paragraphs are separate clauses: clean",
			body:       "Closes #625\n\nThe #625 follow-up is filed\n\nNames and leaves open #830\n",
			wantOutput: "landcheck: t: closing keywords bind #625\n",
		},
		{
			name:       "separate list items are separate clauses: clean",
			body:       "- Closes #5\n- #5's follow-up is filed\n- names and leaves open #6\n",
			wantOutput: "landcheck: t: closing keywords bind #5\n",
		},
		{
			name:       "comma form",
			body:       "Closes #669, #625, #748.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"Closes #669\" is the comma form: a further reference follows #669, and the keyword closes only #669\n",
		},
		{
			name:       "same issue bound and named and left open",
			body:       "Closes #625. Names and leaves open #625.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"Closes #625\" binds #625, which a \"names and leaves open\" or \"not closing keywords\" clause also names\n",
		},
		{
			name:       "one sentence binds an issue inside its own leave-open clause",
			body:       "Closes #625 and names and leaves open #830.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"Closes #625\" binds #625, which a \"names and leaves open\" or \"not closing keywords\" clause also names\n",
		},
		{
			name:       "the binding itself sits in a not-closing-keywords clause",
			body:       "Fixes #12 here is not closing keywords.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"Fixes #12\" binds #12, which a \"names and leaves open\" or \"not closing keywords\" clause also names\n",
		},
		{
			name:       "an abbreviation's period does not end the leave-open clause",
			body:       "Closes #5. Names and leaves open, e.g. #5.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"Closes #5\" binds #5, which a \"names and leaves open\" or \"not closing keywords\" clause also names\n",
		},
		{
			name:       "cf., vs. and viz. do not end the leave-open clause",
			body:       "Closes #5. Names and leaves open, cf. #6 vs. #7 viz. #5.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"Closes #5\" binds #5, which a \"names and leaves open\" or \"not closing keywords\" clause also names\n",
		},
		{
			name:       "a blank line after an abbreviation's letters still ends the clause: clean",
			body:       "Closes #5 vs\n\nnames and leaves open #6",
			wantOutput: "landcheck: t: closing keywords bind #5\n",
		},
		{
			name:       "an undotted abbreviation outside the list ends the clause: the GAP(landcheck) false accept",
			body:       "Closes #5. Names and leaves open #6 etc. and #5.",
			wantOutput: "landcheck: t: closing keywords bind #5\n",
		},
		{
			name:       "keyword, reference and phrase across line breaks",
			body:       "fixes\n#12 as a verb.\n\n#12 is not\nclosing\nkeywords here.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"fixes #12\" binds #12, which a \"names and leaves open\" or \"not closing keywords\" clause also names\n",
		},
		{
			name:       "possessive reference after a colon still binds",
			body:       "Resolved: #345's premise. Names and leaves open #345.",
			wantCode:   1,
			wantOutput: "landcheck: t: \"Resolved: #345\" binds #345, which a \"names and leaves open\" or \"not closing keywords\" clause also names\n",
		},
		{
			name:       "cross-repository and URL references bind",
			body:       "Closes kud360/goxsd8#3. Fixes https://github.com/kud360/goxsd8/issues/4.",
			wantOutput: "landcheck: t: closing keywords bind #3 #4\n",
		},
		{
			name:       "no-issue mode with no keyword: clean",
			body:       "meta: backlog 2026-09-26\n\nNames and leaves open #345.\n",
			closesNone: true,
			wantOutput: "landcheck: t: closing keywords bind nothing\n",
		},
		{
			name:       "no-issue mode rejects every binding",
			body:       "Closes #1. Fixes\n#2's premise.",
			closesNone: true,
			wantCode:   1,
			wantOutput: "landcheck: t: \"Closes #1\" binds #1 in a PR that closes no issue\n" +
				"landcheck: t: \"Fixes #2\" binds #2 in a PR that closes no issue\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			code, err := checkClosingKeywords([]landingText{{name: "t", body: tc.body}}, tc.closesNone, &buf)
			if err != nil {
				t.Fatalf("checkClosingKeywords: %v", err)
			}
			if code != tc.wantCode || buf.String() != tc.wantOutput {
				t.Errorf("code = %d, output:\n%s\nwant code %d, output:\n%s", code, buf.String(), tc.wantCode, tc.wantOutput)
			}
		})
	}
}

// TestCheckClosingKeywordsReadsEveryText is #1706's shape: the squash text
// is clean and the binding lives only in the PR description, so a check
// that stopped at the first text would pass it.
func TestCheckClosingKeywordsReadsEveryText(t *testing.T) {
	texts := []landingText{
		{name: "squash text", body: "meta: backlog 2026-09-25\n\nSpec: n/a\nRatchet: unchanged\n"},
		{name: "PR description", body: "- Fixes #345's stale premise.\n"},
	}
	var buf bytes.Buffer
	code, err := checkClosingKeywords(texts, true, &buf)
	if err != nil {
		t.Fatalf("checkClosingKeywords: %v", err)
	}
	want := "landcheck: squash text: closing keywords bind nothing\n" +
		"landcheck: PR description: \"Fixes #345\" binds #345 in a PR that closes no issue\n"
	if code != 1 || buf.String() != want {
		t.Errorf("code = %d, output:\n%s\nwant code 1, output:\n%s", code, buf.String(), want)
	}
}
