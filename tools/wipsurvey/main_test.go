package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC)

// TestClassify exercises the pure classification function directly: no
// git, no process calls, no filesystem. Each case mutates exactly one
// input away from a baseline that would otherwise land on a different
// verdict, so a broken condition in classify flips the case it covers.
func TestClassify(t *testing.T) {
	tip := func(agoMinutes int) *time.Time {
		t := fixedNow.Add(-time.Duration(agoMinutes) * time.Minute)
		return &t
	}

	// A case that omits anc gets ancestryUnresolved — git declined to
	// answer — which reaches the same verdict it did before this tool could
	// ask about ancestry at all, with a reason that says the ancestry was
	// undecided (#806).
	cases := []struct {
		name        string
		branch      string
		tip         *time.Time
		anc         ancestry
		issue       *issueState
		wantVerdict verdict
		wantReason  string // substring the reason must contain
	}{
		{
			name:        "fresh tip is live",
			branch:      "wip/issue-1",
			tip:         tip(5),
			issue:       nil,
			wantVerdict: live,
			wantReason:  "within",
		},
		{
			name:        "tip exactly at the TTL boundary is live",
			branch:      "wip/issue-2",
			tip:         tip(120), // exactly claimTTL (2h)
			issue:       nil,
			wantVerdict: live,
			wantReason:  "within",
		},
		{
			name:        "tip one minute past the TTL boundary is expired",
			branch:      "wip/issue-3",
			tip:         tip(121),
			issue:       nil,
			wantVerdict: expired,
			wantReason:  "past",
		},
		{
			name:        "old tip is expired",
			branch:      "wip/issue-4",
			tip:         tip(600),
			issue:       nil,
			wantVerdict: expired,
			wantReason:  "past",
		},
		{
			name:        "closed issue retires a fresh-tip branch",
			branch:      "wip/issue-5",
			tip:         tip(1),
			issue:       &issueState{number: 5, closed: true},
			wantVerdict: retired,
			wantReason:  "closed",
		},
		{
			name:        "closed issue retires even with no fetched tip",
			branch:      "wip/issue-6",
			tip:         nil,
			issue:       &issueState{number: 6, closed: true},
			wantVerdict: retired,
			wantReason:  "closed",
		},
		{
			name:        "needs-replan label retires regardless of tip age",
			branch:      "wip/issue-7",
			tip:         tip(1),
			issue:       &issueState{number: 7, needsReplan: true},
			wantVerdict: retired,
			wantReason:  "needs-replan",
		},
		{
			name:        "open issue with fresh tip is live, not retired",
			branch:      "wip/issue-8",
			tip:         tip(1),
			issue:       &issueState{number: 8, closed: false, needsReplan: false},
			wantVerdict: live,
			wantReason:  "within",
		},
		{
			name:        "missing issue data on a fresh tip reports lease-only live",
			branch:      "wip/issue-9",
			tip:         tip(1),
			issue:       nil,
			wantVerdict: live,
			wantReason:  "lease-only",
		},
		{
			name:        "missing issue data on an old tip reports lease-only expired",
			branch:      "wip/issue-10",
			tip:         tip(600),
			issue:       nil,
			wantVerdict: expired,
			wantReason:  "lease-only",
		},
		{
			name:        "absent tip with no issue data is unknown, not expired",
			branch:      "wip/issue-11",
			tip:         nil,
			issue:       nil,
			wantVerdict: unknown,
			wantReason:  "not fetched",
		},
		{
			name:        "absent tip with an open issue is still unknown",
			branch:      "wip/issue-12",
			tip:         nil,
			issue:       &issueState{number: 12, closed: false, needsReplan: false},
			wantVerdict: unknown,
			wantReason:  "not fetched",
		},
		{
			// The #722 shape: a lease pushed minutes ago at main's tip,
			// where the tip's own age is hours past the TTL.
			name:        "zero-commit branch with a stale borrowed tip is claimed, not expired",
			branch:      "wip/issue-13",
			tip:         tip(600),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 13, closed: false, needsReplan: false},
			wantVerdict: claimed,
			wantReason:  "no commits of its own",
		},
		{
			name:        "zero-commit branch with a fresh borrowed tip is claimed, not live",
			branch:      "wip/issue-14",
			tip:         tip(1),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 14, closed: false, needsReplan: false},
			wantVerdict: claimed,
			wantReason:  "tip age is main's",
		},
		{
			name:        "closed issue retires a zero-commit branch",
			branch:      "wip/issue-15",
			tip:         tip(600),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 15, closed: true},
			wantVerdict: retired,
			wantReason:  "closed",
		},
		{
			name:        "needs-replan retires a zero-commit branch",
			branch:      "wip/issue-16",
			tip:         tip(600),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 16, needsReplan: true},
			wantVerdict: retired,
			wantReason:  "needs-replan",
		},
		{
			name:        "unfetched tip beats the ancestry answer",
			branch:      "wip/issue-17",
			tip:         nil,
			anc:         ancestryNoCommits,
			issue:       nil,
			wantVerdict: unknown,
			wantReason:  "not fetched",
		},
		{
			name:        "branch with commits of its own is still dated from its tip",
			branch:      "wip/issue-18",
			tip:         tip(600),
			anc:         ancestryOwnCommits,
			issue:       &issueState{number: 18, closed: false, needsReplan: false},
			wantVerdict: expired,
			wantReason:  "past",
		},
		{
			name:        "branch with commits of its own and a fresh tip is live",
			branch:      "wip/issue-19",
			tip:         tip(5),
			anc:         ancestryOwnCommits,
			issue:       &issueState{number: 19, closed: false, needsReplan: false},
			wantVerdict: live,
			wantReason:  "within",
		},
		{
			// git could not decide (exit 128, or no main on the remote):
			// fall back to the tip age rather than invent a second unknown.
			name:        "unresolved ancestry falls back to the tip age",
			branch:      "wip/issue-20",
			tip:         tip(600),
			anc:         ancestryUnresolved,
			issue:       &issueState{number: 20, closed: false, needsReplan: false},
			wantVerdict: expired,
			wantReason:  "past",
		},
		{
			name:        "unresolved ancestry says so on an expired row",
			branch:      "wip/issue-22",
			tip:         tip(600),
			anc:         ancestryUnresolved,
			issue:       &issueState{number: 22, closed: false, needsReplan: false},
			wantVerdict: expired,
			wantReason:  "ancestry against main undecided",
		},
		{
			name:        "unresolved ancestry says so on a live row too",
			branch:      "wip/issue-23",
			tip:         tip(5),
			anc:         ancestryUnresolved,
			issue:       &issueState{number: 23, closed: false, needsReplan: false},
			wantVerdict: live,
			wantReason:  "ancestry against main undecided",
		},
		{
			name:        "unresolved ancestry names the fetch that would decide it",
			branch:      "wip/issue-24",
			tip:         tip(600),
			anc:         ancestryUnresolved,
			issue:       nil,
			wantVerdict: expired,
			wantReason:  "run `git fetch origin`",
		},
		{
			name:        "zero-commit branch with no issue data still says lease-only",
			branch:      "wip/issue-21",
			tip:         tip(600),
			anc:         ancestryNoCommits,
			issue:       nil,
			wantVerdict: claimed,
			wantReason:  "lease-only",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotVerdict, gotReason := classify(c.branch, c.tip, c.anc, fixedNow, c.issue)
			if gotVerdict != c.wantVerdict {
				t.Errorf("classify(%q) verdict = %s, want %s (reason: %s)", c.branch, gotVerdict, c.wantVerdict, gotReason)
			}
			if !strings.Contains(gotReason, c.wantReason) {
				t.Errorf("classify(%q) reason = %q, want substring %q", c.branch, gotReason, c.wantReason)
			}
		})
	}
}

// TestClassifyNeverFallsThroughToExpired guards specifically against the
// hazard the coordinator's correction called out: a nil tip must never
// silently read as EXPIRED, because EXPIRED means resumable and that is
// the dangerous direction for a branch this tool has no age data for.
func TestClassifyNeverFallsThroughToExpired(t *testing.T) {
	got, _ := classify("wip/issue-99", nil, ancestryUnresolved, fixedNow, nil)
	if got == expired {
		t.Fatalf("classify with a nil tip returned EXPIRED; an absent tip must never be treated as an old one")
	}
	if got != unknown {
		t.Fatalf("classify with a nil tip and no issue data = %s, want UNKNOWN", got)
	}
}

// TestClassifyNeverExpiresABorrowedTip guards the #722 hazard at every tip
// age: a branch with no commits of its own has no age of its own, so no
// arrangement of the clock may make it EXPIRED — the verdict /develop
// reads as "resumable" and /backlog reads as grounds for needs-replan.
func TestClassifyNeverExpiresABorrowedTip(t *testing.T) {
	for _, agoMinutes := range []int{0, 1, 119, 120, 121, 600, 100000} {
		tip := fixedNow.Add(-time.Duration(agoMinutes) * time.Minute)
		got, reason := classify("wip/issue-98", &tip, ancestryNoCommits, fixedNow, nil)
		if got != claimed {
			t.Errorf("classify with a %dm-old borrowed tip = %s, want CLAIMED (reason: %s)", agoMinutes, got, reason)
		}
	}
}

// TestClassifyMarksOnlyUndecidedAncestry pins the marker to the rows it
// belongs on. An age-based verdict earns it when git could not place the
// branch against main, because the age it rests on may then be the
// borrowed one #722 is about; a verdict whose ancestry git did decide must
// NOT carry it, or the marker stops distinguishing anything (#806).
func TestClassifyMarksOnlyUndecidedAncestry(t *testing.T) {
	const marker = "ancestry against main undecided"
	tips := []struct {
		name string
		tip  time.Time
	}{
		{"fresh", fixedNow.Add(-5 * time.Minute)},
		{"stale", fixedNow.Add(-10 * time.Hour)},
	}
	decided := []struct {
		name string
		anc  ancestry
	}{
		{"ownCommits", ancestryOwnCommits},
		{"noCommits", ancestryNoCommits},
	}

	for _, tc := range tips {
		got, reason := classify("wip/issue-1", &tc.tip, ancestryUnresolved, fixedNow, nil)
		if !strings.Contains(reason, marker) {
			t.Errorf("classify(%s tip, unresolved) = %s, reason %q lacks %q", tc.name, got, reason, marker)
		}
		for _, d := range decided {
			got, reason := classify("wip/issue-1", &tc.tip, d.anc, fixedNow, nil)
			if strings.Contains(reason, marker) {
				t.Errorf("classify(%s tip, %s) = %s, reason %q calls an ancestry git decided undecided", tc.name, d.name, got, reason)
			}
		}
	}
}

// TestAncestryFromExit pins the exit-status mapping `git merge-base
// --is-ancestor` defines, including the 128 that means git declined to
// answer: mapping that to "has its own commits" would silently restore the
// borrowed-age EXPIRED (#722), so it must land on unresolved instead.
func TestAncestryFromExit(t *testing.T) {
	cases := []struct {
		code int
		want ancestry
	}{
		{0, ancestryNoCommits},
		{1, ancestryOwnCommits},
		{128, ancestryUnresolved},
		{129, ancestryUnresolved},
	}
	for _, c := range cases {
		if got := ancestryFromExit(c.code); got != c.want {
			t.Errorf("ancestryFromExit(%d) = %v, want %v", c.code, got, c.want)
		}
	}
}

// TestGitAncestryWithoutMain checks the no-main-on-the-remote path
// resolves to unresolved without running git at all — the survey still
// reports, dated from tip ages.
func TestGitAncestryWithoutMain(t *testing.T) {
	got, err := gitAncestry("aaaa111", "")
	if err != nil {
		t.Fatalf("gitAncestry with no main SHA: unexpected error: %v", err)
	}
	if got != ancestryUnresolved {
		t.Fatalf("gitAncestry with no main SHA = %v, want ancestryUnresolved", got)
	}
}

// TestPartitionRefs checks main is taken out of the surveyed branch set:
// it is the ancestry test's right-hand side, never a row.
func TestPartitionRefs(t *testing.T) {
	refs := []refSHA{
		{sha: "aaaa111", ref: "refs/heads/wip/issue-1"},
		{sha: "mmmm999", ref: "refs/heads/main"},
		{sha: "bbbb222", ref: "refs/heads/wip/issue-2"},
	}
	wips, mainSHA := partitionRefs(refs)
	if mainSHA != "mmmm999" {
		t.Errorf("mainSHA = %q, want %q", mainSHA, "mmmm999")
	}
	if len(wips) != 2 || wips[0].ref != "refs/heads/wip/issue-1" || wips[1].ref != "refs/heads/wip/issue-2" {
		t.Errorf("wips = %+v, want the two wip refs in ls-remote order", wips)
	}

	t.Run("no main on the remote yields an empty SHA", func(t *testing.T) {
		wips, mainSHA := partitionRefs([]refSHA{{sha: "aaaa111", ref: "refs/heads/wip/issue-1"}})
		if mainSHA != "" {
			t.Errorf("mainSHA = %q, want empty", mainSHA)
		}
		if len(wips) != 1 {
			t.Errorf("wips = %+v, want the one wip ref", wips)
		}
	})
}

// TestParseWipRef exercises the pure branch-name parser, including the
// malformed shapes it must reject.
func TestParseWipRef(t *testing.T) {
	cases := []struct {
		name       string
		sha, ref   string
		wantBranch string
		wantIssue  int
		wantErr    bool
	}{
		{name: "valid", sha: "abc123", ref: "refs/heads/wip/issue-636", wantBranch: "wip/issue-636", wantIssue: 636},
		{name: "single digit", sha: "abc123", ref: "refs/heads/wip/issue-1", wantBranch: "wip/issue-1", wantIssue: 1},
		{name: "not under refs/heads", sha: "abc123", ref: "refs/tags/wip/issue-1", wantErr: true},
		{name: "not a wip branch", sha: "abc123", ref: "refs/heads/main", wantErr: true},
		{name: "missing issue number", sha: "abc123", ref: "refs/heads/wip/issue-", wantErr: true},
		{name: "non-numeric issue", sha: "abc123", ref: "refs/heads/wip/issue-abc", wantErr: true},
		{name: "leading zero rejected", sha: "abc123", ref: "refs/heads/wip/issue-007", wantErr: true},
		{name: "zero rejected", sha: "abc123", ref: "refs/heads/wip/issue-0", wantErr: true},
		{name: "other wip-prefixed branch", sha: "abc123", ref: "refs/heads/wip/issue-42/extra", wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseWipRef(c.sha, c.ref)
			if c.wantErr {
				if err == nil {
					t.Fatalf("parseWipRef(%q, %q) = %+v, want error", c.sha, c.ref, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseWipRef(%q, %q) unexpected error: %v", c.sha, c.ref, err)
			}
			if got.branch != c.wantBranch || got.issue != c.wantIssue || got.sha != c.sha {
				t.Errorf("parseWipRef(%q, %q) = %+v, want branch=%s issue=%d sha=%s", c.sha, c.ref, got, c.wantBranch, c.wantIssue, c.sha)
			}
		})
	}
}

// TestParseLsRemote exercises the pure ls-remote output parser: no git
// call, just the text `git ls-remote --heads` would have produced.
func TestParseLsRemote(t *testing.T) {
	t.Run("multiple well-formed lines", func(t *testing.T) {
		out := "aaaa111\trefs/heads/wip/issue-1\n" +
			"bbbb222\trefs/heads/wip/issue-2\n"
		refs, errs := parseLsRemote(out)
		if len(errs) != 0 {
			t.Fatalf("unexpected errors: %v", errs)
		}
		if len(refs) != 2 {
			t.Fatalf("got %d refs, want 2: %+v", len(refs), refs)
		}
		if refs[0].sha != "aaaa111" || refs[0].ref != "refs/heads/wip/issue-1" {
			t.Errorf("refs[0] = %+v", refs[0])
		}
		if refs[1].sha != "bbbb222" || refs[1].ref != "refs/heads/wip/issue-2" {
			t.Errorf("refs[1] = %+v", refs[1])
		}
	})

	t.Run("empty output yields no refs and no errors", func(t *testing.T) {
		refs, errs := parseLsRemote("")
		if refs != nil || errs != nil {
			t.Fatalf("parseLsRemote(\"\") = %+v, %+v, want nil, nil", refs, errs)
		}
	})

	t.Run("line without a tab is reported, not silently dropped", func(t *testing.T) {
		out := "aaaa111\trefs/heads/wip/issue-1\n" +
			"not-tab-separated\n"
		refs, errs := parseLsRemote(out)
		if len(refs) != 1 {
			t.Fatalf("got %d refs, want 1 (the well-formed line): %+v", len(refs), refs)
		}
		if len(errs) != 1 {
			t.Fatalf("got %d errors, want 1 for the malformed line: %v", len(errs), errs)
		}
	})
}

// TestReadIssues exercises the JSON parsing against the exact shape `gh
// issue list --state all --json number,state,labels` produces.
func TestReadIssues(t *testing.T) {
	t.Run("parses number, state, and needs-replan label", func(t *testing.T) {
		in := `[
			{"number":636,"state":"OPEN","labels":[{"name":"ready"}]},
			{"number":300,"state":"CLOSED","labels":[]},
			{"number":301,"state":"open","labels":[{"name":"needs-replan"},{"name":"kind/bug"}]}
		]`
		issues, err := readIssues(strings.NewReader(in))
		if err != nil {
			t.Fatalf("readIssues: %v", err)
		}
		if len(issues) != 3 {
			t.Fatalf("got %d issues, want 3: %+v", len(issues), issues)
		}
		if got := issues[636]; got.closed || got.needsReplan {
			t.Errorf("issue 636 = %+v, want open, no needs-replan", got)
		}
		if got := issues[300]; !got.closed {
			t.Errorf("issue 300 = %+v, want closed", got)
		}
		if got := issues[301]; !got.needsReplan {
			t.Errorf("issue 301 = %+v, want needs-replan", got)
		}
		// state matching must be case-insensitive: "open" (lowercase) above
		// must not be mistaken for closed.
		if got := issues[301]; got.closed {
			t.Errorf("issue 301 = %+v, want open despite lowercase state", got)
		}
	})

	t.Run("empty stdin reports lease-only, not an error", func(t *testing.T) {
		issues, err := readIssues(strings.NewReader(""))
		if err != nil {
			t.Fatalf("readIssues(empty): unexpected error: %v", err)
		}
		if issues != nil {
			t.Fatalf("readIssues(empty) = %+v, want nil map", issues)
		}
	})

	t.Run("malformed JSON is an error", func(t *testing.T) {
		_, err := readIssues(strings.NewReader("{not valid json"))
		if err == nil {
			t.Fatalf("readIssues(malformed): want error, got nil")
		}
	})
}

// TestSortRowsAndRenderTable exercises row ordering and rendering
// together: rows are built out of issue-number order on purpose, so a
// broken sort would show up as a non-ascending ISSUE column.
func TestSortRowsAndRenderTable(t *testing.T) {
	t1 := fixedNow.Add(-5 * time.Minute)
	rows := []row{
		{issue: 10, branch: "wip/issue-10", tip: &t1, verdict: live, reason: "wip/issue-10: tip pushed 5m0s ago, within the 2h0m0s claim TTL"},
		{issue: 2, branch: "wip/issue-2", tip: nil, verdict: unknown, reason: "wip/issue-2: tip not fetched -- run `git fetch origin`"},
		{issue: 5, branch: "wip/issue-5", tip: &t1, verdict: retired, reason: "wip/issue-5: issue #5 is closed"},
	}
	sortRows(rows)

	if rows[0].issue != 2 || rows[1].issue != 5 || rows[2].issue != 10 {
		t.Fatalf("sortRows did not order by issue number: %+v", rows)
	}

	var buf bytes.Buffer
	if err := renderTable(&buf, rows, fixedNow); err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	got := buf.String()

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != len(rows)+1 {
		t.Fatalf("got %d lines, want %d (header + %d rows):\n%s", len(lines), len(rows)+1, len(rows), got)
	}
	if !strings.HasPrefix(lines[0], "ISSUE") {
		t.Fatalf("first line %q is not the header", lines[0])
	}
	// Row order in the rendered text must follow the sorted order (2, 5, 10),
	// not the order the rows were constructed in (10, 2, 5).
	if !strings.Contains(lines[1], "wip/issue-2") || !strings.Contains(lines[1], "UNKNOWN") {
		t.Errorf("row 1 = %q, want issue 2 (UNKNOWN)", lines[1])
	}
	if !strings.Contains(lines[2], "wip/issue-5") || !strings.Contains(lines[2], "RETIRED") {
		t.Errorf("row 2 = %q, want issue 5 (RETIRED)", lines[2])
	}
	if !strings.Contains(lines[3], "wip/issue-10") || !strings.Contains(lines[3], "LIVE") {
		t.Errorf("row 3 = %q, want issue 10 (LIVE)", lines[3])
	}
}

// TestRenderTableClaimedAge checks a CLAIMED row does not print a
// duration in TIP AGE. The duration would be real and would belong to
// main, and printing it there is what got acted on (#722).
func TestRenderTableClaimedAge(t *testing.T) {
	borrowed := fixedNow.Add(-10 * time.Hour)
	rows := []row{
		{issue: 1, branch: "wip/issue-1", tip: &borrowed, verdict: claimed, reason: "wip/issue-1: no commits of its own"},
	}
	var buf bytes.Buffer
	if err := renderTable(&buf, rows, fixedNow); err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "main's") {
		t.Errorf("CLAIMED row does not name whose tip age it is:\n%s", got)
	}
	if strings.Contains(got, "10h0m0s") {
		t.Errorf("CLAIMED row prints main's age as the branch's:\n%s", got)
	}
	if !strings.Contains(got, "CLAIMED") {
		t.Errorf("CLAIMED row does not carry the verdict:\n%s", got)
	}
}

// TestRenderTableDeterministic guards STYLE D1: rendering the same rows
// twice must produce byte-identical output.
func TestRenderTableDeterministic(t *testing.T) {
	tip := fixedNow.Add(-30 * time.Minute)
	rows := []row{
		{issue: 1, branch: "wip/issue-1", tip: &tip, verdict: live, reason: "wip/issue-1: tip pushed 30m0s ago, within the 2h0m0s claim TTL"},
	}

	var a, b bytes.Buffer
	if err := renderTable(&a, rows, fixedNow); err != nil {
		t.Fatalf("renderTable (a): %v", err)
	}
	if err := renderTable(&b, rows, fixedNow); err != nil {
		t.Fatalf("renderTable (b): %v", err)
	}
	if a.String() != b.String() {
		t.Fatalf("renderTable is not deterministic:\n--- a ---\n%s\n--- b ---\n%s", a.String(), b.String())
	}
}

// TestFormatAge checks the rounding renderTable and classify's reason
// text both rely on.
func TestFormatAge(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{90 * time.Second, "2m0s"}, // rounds up to the nearest minute
		{2 * time.Hour, "2h0m0s"},
	}
	for _, c := range cases {
		if got := formatAge(c.d); got != c.want {
			t.Errorf("formatAge(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
