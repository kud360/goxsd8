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
	// Both clocks are read the same way — minutes before fixedNow — and the
	// alias keeps each case saying which clock it is setting.
	heartbeat := tip

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
		// wantCell is the LEASE AGE cell the row classify returns must
		// render to. Empty means the case does not speak to the column —
		// no real cell is ever empty, so absent and asserted stay
		// distinguishable without a second field.
		wantCell string
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
			// The branch may have commits of its own, so its tip's age is
			// its own to print.
			wantCell: "1m0s",
		},
		{
			name:        "closed issue retires even with no fetched tip",
			branch:      "wip/issue-6",
			tip:         nil,
			issue:       &issueState{number: 6, closed: true},
			wantVerdict: retired,
			wantReason:  "closed",
			wantCell:    "unknown",
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
			// Retirement does not make main's 10h the branch's own lease
			// age; the cell must say whose it is (#809).
			wantCell: "main's",
		},
		{
			name:        "needs-replan retires a zero-commit branch",
			branch:      "wip/issue-16",
			tip:         tip(600),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 16, needsReplan: true},
			wantVerdict: retired,
			wantReason:  "needs-replan",
			wantCell:    "main's",
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
		{
			// The empty claim's own clock: a fresh heartbeat holds the lease
			// even though the borrowed tip is ten hours old.
			name:        "zero-commit branch with a fresh heartbeat is live",
			branch:      "wip/issue-25",
			tip:         tip(600),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 25, commentsRead: true, heartbeat: heartbeat(30)},
			wantVerdict: live,
			wantReason:  "lease dated by its newest RESUME:/TAKEOVER: comment, posted 30m0s ago",
		},
		{
			name:        "zero-commit branch with a heartbeat exactly at the TTL is live",
			branch:      "wip/issue-26",
			tip:         tip(1),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 26, commentsRead: true, heartbeat: heartbeat(120)},
			wantVerdict: live,
			wantReason:  "within",
		},
		{
			// The #981 sighting: the takeover comment that dated the lease was
			// 4h18m old, so the claim was takeable.
			name:        "zero-commit branch with a heartbeat past the TTL is takeable",
			branch:      "wip/issue-27",
			tip:         tip(1),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 27, commentsRead: true, heartbeat: heartbeat(258)},
			wantVerdict: expired,
			wantReason:  "takeable",
			// The heartbeat's own age, not the borrowed tip's 1m.
			wantCell: "4h18m0s",
		},
		{
			// /develop pushes the claim before it posts anything, so this is
			// also the shape of an issue being grounded right now: only an
			// AGED heartbeat may demote a claim (#981).
			name:        "zero-commit branch whose thread carries no heartbeat stays claimed",
			branch:      "wip/issue-28",
			tip:         tip(1),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 28, commentsRead: true},
			wantVerdict: claimed,
			wantReason:  "a claim is born undated, so this is not a lapsed lease",
			wantCell:    "main's",
		},
		{
			name:        "comments not supplied leaves the empty claim undated",
			branch:      "wip/issue-29",
			tip:         tip(1),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 29},
			wantVerdict: claimed,
			wantReason:  "supply this issue's comments",
		},
		{
			name:        "a closed issue retires an empty claim with a fresh heartbeat",
			branch:      "wip/issue-30",
			tip:         tip(1),
			anc:         ancestryNoCommits,
			issue:       &issueState{number: 30, closed: true, commentsRead: true, heartbeat: heartbeat(1)},
			wantVerdict: retired,
			wantReason:  "closed",
			// Retirement short-circuits the thread's clock too, so nothing
			// dates this row and the tip it holds is main's.
			wantCell: "main's",
		},
		{
			// A branch with commits of its own is dated by its tip and nothing
			// else: a stale tip is EXPIRED however fresh the thread is.
			name:        "a heartbeat does not extend a branch that has its own tip",
			branch:      "wip/issue-31",
			tip:         tip(600),
			anc:         ancestryOwnCommits,
			issue:       &issueState{number: 31, commentsRead: true, heartbeat: heartbeat(1)},
			wantVerdict: expired,
			wantReason:  "tip pushed",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotVerdict, gotLease, gotReason := classify(c.branch, c.tip, c.anc, fixedNow, c.issue)
			if gotVerdict != c.wantVerdict {
				t.Errorf("classify(%q) verdict = %s, want %s (reason: %s)", c.branch, gotVerdict, c.wantVerdict, gotReason)
			}
			if !strings.Contains(gotReason, c.wantReason) {
				t.Errorf("classify(%q) reason = %q, want substring %q", c.branch, gotReason, c.wantReason)
			}
			if c.wantCell == "" {
				return
			}
			got := leaseAgeCell(row{lease: gotLease, anc: c.anc, verdict: gotVerdict})
			if got != c.wantCell {
				t.Errorf("classify(%q) LEASE AGE cell = %q, want %q", c.branch, got, c.wantCell)
			}
		})
	}
}

// TestClassifyNeverFallsThroughToExpired guards specifically against the
// hazard the coordinator's correction called out: a nil tip must never
// silently read as EXPIRED, because EXPIRED means resumable and that is
// the dangerous direction for a branch this tool has no age data for.
func TestClassifyNeverFallsThroughToExpired(t *testing.T) {
	got, _, _ := classify("wip/issue-99", nil, ancestryUnresolved, fixedNow, nil)
	if got == expired {
		t.Fatalf("classify with a nil tip returned EXPIRED; an absent tip must never be treated as an old one")
	}
	if got != unknown {
		t.Fatalf("classify with a nil tip and no issue data = %s, want UNKNOWN", got)
	}
}

// TestClassifyNeverExpiresABorrowedTip guards the #722 hazard at every tip
// age: a branch with no commits of its own has no age of its own, so no
// arrangement of the TIP clock may make it EXPIRED — the verdict /develop
// reads as "resumable" and /backlog reads as grounds for needs-replan.
// Only the thread's own clock decides such a branch, and this test hands
// classify no heartbeat to read on any thread: absent issue data, an issue
// whose comments were not supplied, and a thread that was read and carries
// no heartbeat. None of the three is a lapsed lease (#981).
func TestClassifyNeverExpiresABorrowedTip(t *testing.T) {
	for _, agoMinutes := range []int{0, 1, 119, 120, 121, 600, 100000} {
		tip := fixedNow.Add(-time.Duration(agoMinutes) * time.Minute)
		for _, issue := range []*issueState{nil, {number: 98}, {number: 98, commentsRead: true}} {
			got, lease, reason := classify("wip/issue-98", &tip, ancestryNoCommits, fixedNow, issue)
			if got != claimed {
				t.Errorf("classify with a %dm-old borrowed tip = %s, want CLAIMED (reason: %s)", agoMinutes, got, reason)
			}
			if lease != nil {
				t.Errorf("classify with a %dm-old borrowed tip dated the lease %v; a borrowed tip dates nothing", agoMinutes, *lease)
			}
		}
	}
}

// TestClassifyEmptyClaimIgnoresTheBorrowedTip pins the same hazard on the
// path that CAN return EXPIRED for an empty claim: once the thread dates
// the lease, the tip's age must not reach the arithmetic at all, in either
// direction.
func TestClassifyEmptyClaimIgnoresTheBorrowedTip(t *testing.T) {
	fresh := fixedNow.Add(-30 * time.Minute)
	stale := fixedNow.Add(-500 * time.Minute)
	cases := []struct {
		name      string
		heartbeat time.Time
		want      verdict
	}{
		{"heartbeat within the TTL", fresh, live},
		{"heartbeat past the TTL", stale, expired},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, tipAgoMinutes := range []int{0, 1, 119, 120, 121, 600, 100000} {
				tip := fixedNow.Add(-time.Duration(tipAgoMinutes) * time.Minute)
				issue := &issueState{number: 97, commentsRead: true, heartbeat: &c.heartbeat}
				got, lease, reason := classify("wip/issue-97", &tip, ancestryNoCommits, fixedNow, issue)
				if got != c.want {
					t.Errorf("tip %dm old, %s: verdict = %s, want %s (reason: %s)", tipAgoMinutes, c.name, got, c.want, reason)
				}
				if lease == nil || *lease != fixedNow.Sub(c.heartbeat) {
					t.Errorf("tip %dm old, %s: lease age = %v, want the heartbeat's %v", tipAgoMinutes, c.name, lease, fixedNow.Sub(c.heartbeat))
				}
			}
		})
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
		got, _, reason := classify("wip/issue-1", &tc.tip, ancestryUnresolved, fixedNow, nil)
		if !strings.Contains(reason, marker) {
			t.Errorf("classify(%s tip, unresolved) = %s, reason %q lacks %q", tc.name, got, reason, marker)
		}
		for _, d := range decided {
			got, _, reason := classify("wip/issue-1", &tc.tip, d.anc, fixedNow, nil)
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

	t.Run("an absent comments key is not an empty thread", func(t *testing.T) {
		in := `[
			{"number":1,"state":"OPEN","labels":[]},
			{"number":2,"state":"OPEN","labels":[],"comments":[]}
		]`
		issues, err := readIssues(strings.NewReader(in))
		if err != nil {
			t.Fatalf("readIssues: %v", err)
		}
		if got := issues[1]; got.commentsRead {
			t.Errorf("issue 1 = %+v, want commentsRead false: nobody supplied its comments", got)
		}
		if got := issues[2]; !got.commentsRead || got.heartbeat != nil {
			t.Errorf("issue 2 = %+v, want commentsRead true with no heartbeat", got)
		}
	})

	t.Run("the newest heartbeat comment dates the issue", func(t *testing.T) {
		in := `[{"number":3,"state":"OPEN","labels":[],"comments":[
			{"body":"RESUME: next action is the gate.","createdAt":"2026-08-26T06:00:00Z"},
			{"body":"TAKEOVER: resuming this issue.","createdAt":"2026-08-26T08:22:50Z"},
			{"body":"GROUNDING: no XSD rule is in scope.","createdAt":"2026-08-26T09:15:00Z"}
		]}]`
		issues, err := readIssues(strings.NewReader(in))
		if err != nil {
			t.Fatalf("readIssues: %v", err)
		}
		want := time.Date(2026, 8, 26, 8, 22, 50, 0, time.UTC)
		got := issues[3]
		if got.heartbeat == nil || !got.heartbeat.Equal(want) {
			t.Errorf("issue 3 heartbeat = %v, want the 08:22:50Z takeover (the newer grounding dates nothing)", got.heartbeat)
		}
	})
}

// TestIsHeartbeat pins the criterion Acceptance item 1 asked to be
// checkable from what a comment carries: the marker the body opens with,
// never the author it is presumed to have.
func TestIsHeartbeat(t *testing.T) {
	cases := []struct {
		body string
		want bool
	}{
		{"RESUME: next action is the gate.", true},
		{"TAKEOVER: resuming this issue. `wip/issue-981` carries no commits.", true},
		{"\n  RESUME: leading whitespace is not a marker change.", true},
		{"GROUNDING: no XSD/XPath/F&O rule is in scope.", false},
		{"MASON: implementation account. Commits: abc1234.", false},
		{"ARBITER: ACCEPT.", false},
		{"CARTOGRAPHER (`/backlog`, 2026-08-25): promoted four rows.", false},
		{"I will RESUME: this later.", false},
		{"resume: lowercase is not the marker.", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isHeartbeat(c.body); got != c.want {
			t.Errorf("isHeartbeat(%q) = %v, want %v", c.body, got, c.want)
		}
	}
}

// TestNewestHeartbeat checks the newest heartbeat is taken by timestamp
// and that non-heartbeat comments never win, however recent they are.
func TestNewestHeartbeat(t *testing.T) {
	at := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("parsing fixture time %q: %v", s, err)
		}
		return ts
	}
	comments := []ghComment{
		{Body: "GROUNDING: posted last, dates nothing.", CreatedAt: at("2026-08-26T10:00:00Z")},
		{Body: "TAKEOVER: taking the claim.", CreatedAt: at("2026-08-26T08:22:50Z")},
		{Body: "RESUME: earlier heartbeat.", CreatedAt: at("2026-08-26T06:00:00Z")},
	}
	got := newestHeartbeat(comments)
	if got == nil || !got.Equal(at("2026-08-26T08:22:50Z")) {
		t.Errorf("newestHeartbeat = %v, want the 08:22:50Z takeover", got)
	}

	if got := newestHeartbeat([]ghComment{comments[0]}); got != nil {
		t.Errorf("newestHeartbeat(grounding only) = %v, want nil", got)
	}
	if got := newestHeartbeat(nil); got != nil {
		t.Errorf("newestHeartbeat(nil) = %v, want nil", got)
	}
}

// TestEmptyClaimLeaseFixtures is the regression named by #981, driven end
// to end from the input shape so both halves of the rule are exercised:
// which comment dates the lease, and the TTL arithmetic on it.
//
// The negative is #884's: a grounding posted 44 minutes before the read
// must not hold the claim for a further two hours. What it leaves behind
// is CLAIMED and not EXPIRED, because a thread with no heartbeat on it is
// also every claim's first minutes. The positive is #993's takeover
// comment, which holds the lease while it is inside the TTL and stops
// holding it once it is not.
func TestEmptyClaimLeaseFixtures(t *testing.T) {
	const read = "2026-08-26T09:00:00Z"
	now, err := time.Parse(time.RFC3339, read)
	if err != nil {
		t.Fatalf("parsing the read time: %v", err)
	}
	borrowed := now.Add(-10 * time.Hour) // main's tip, ten hours old

	cases := []struct {
		name string
		json string
		want verdict
	}{
		{
			// A grounding does not hold the lease — but it does not lapse one
			// either, and a thread whose only comment is a grounding is
			// exactly what /develop's step 3 leaves behind, so the claim
			// stays CLAIMED for a reader to settle (#981).
			name: "a grounding posted 44 minutes ago neither holds nor lapses the lease",
			json: `[{"number":884,"state":"OPEN","labels":[{"name":"ready"}],"comments":[
				{"body":"GROUNDING: no XSD/XPath/F&O rule is in scope.","createdAt":"2026-08-26T08:16:00Z"}
			]}]`,
			want: claimed,
		},
		{
			name: "a takeover comment posted 38 minutes ago holds it",
			json: `[{"number":884,"state":"OPEN","labels":[{"name":"ready"}],"comments":[
				{"body":"TAKEOVER: resuming this issue.","createdAt":"2026-08-26T08:22:00Z"},
				{"body":"GROUNDING: no XSD/XPath/F&O rule is in scope.","createdAt":"2026-08-26T08:16:00Z"}
			]}]`,
			want: live,
		},
		{
			name: "the same takeover comment four hours later does not",
			json: `[{"number":884,"state":"OPEN","labels":[{"name":"ready"}],"comments":[
				{"body":"TAKEOVER: resuming this issue.","createdAt":"2026-08-26T04:22:00Z"},
				{"body":"GROUNDING: no XSD/XPath/F&O rule is in scope.","createdAt":"2026-08-26T08:16:00Z"}
			]}]`,
			want: expired,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			issues, err := readIssues(strings.NewReader(c.json))
			if err != nil {
				t.Fatalf("readIssues: %v", err)
			}
			state := issues[884]
			got, _, reason := classify("wip/issue-884", &borrowed, ancestryNoCommits, now, &state)
			if got != c.want {
				t.Fatalf("verdict = %s, want %s (reason: %s)", got, c.want, reason)
			}
		})
	}
}

// TestSortRowsAndRenderTable exercises row ordering and rendering
// together: rows are built out of issue-number order on purpose, so a
// broken sort would show up as a non-ascending ISSUE column.
func TestSortRowsAndRenderTable(t *testing.T) {
	age := 5 * time.Minute
	rows := []row{
		{issue: 10, branch: "wip/issue-10", lease: &age, verdict: live, reason: "wip/issue-10: tip pushed 5m0s ago, within the 2h0m0s claim TTL"},
		{issue: 2, branch: "wip/issue-2", lease: nil, verdict: unknown, reason: "wip/issue-2: tip not fetched -- run `git fetch origin`"},
		{issue: 5, branch: "wip/issue-5", lease: &age, verdict: retired, reason: "wip/issue-5: issue #5 is closed"},
	}
	sortRows(rows)

	if rows[0].issue != 2 || rows[1].issue != 5 || rows[2].issue != 10 {
		t.Fatalf("sortRows did not order by issue number: %+v", rows)
	}

	var buf bytes.Buffer
	if err := renderTable(&buf, rows); err != nil {
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
	// A row nothing dated keeps reading "unknown" in the age column — the
	// only place an unfetched tip shows on a RETIRED row, where the REASON
	// speaks about the issue instead (#809 rests on that cell).
	if got := strings.Fields(lines[1])[2]; got != "unknown" {
		t.Errorf("row 1 age cell = %q, want `unknown` for an unfetched tip: %q", got, lines[1])
	}
	if !strings.Contains(lines[2], "wip/issue-5") || !strings.Contains(lines[2], "RETIRED") {
		t.Errorf("row 2 = %q, want issue 5 (RETIRED)", lines[2])
	}
	if !strings.Contains(lines[3], "wip/issue-10") || !strings.Contains(lines[3], "LIVE") {
		t.Errorf("row 3 = %q, want issue 10 (LIVE)", lines[3])
	}
}

// TestRenderTableClaimedAge checks a CLAIMED row does not print a
// duration in LEASE AGE. The duration would be real and would belong to
// main, and printing it there is what got acted on (#722). The row is
// built the way run builds one — from classify's own return values — so
// the test pins the whole chain rather than a hand-assembled row classify
// could never produce.
func TestRenderTableClaimedAge(t *testing.T) {
	borrowedTip := fixedNow.Add(-10 * time.Hour)
	got, lease, reason := classify("wip/issue-1", &borrowedTip, ancestryNoCommits, fixedNow, nil)
	if got != claimed {
		t.Fatalf("classify with a 10h borrowed tip = %s, want CLAIMED", got)
	}
	rows := []row{
		{issue: 1, branch: "wip/issue-1", lease: lease, anc: ancestryNoCommits, verdict: got, reason: reason},
	}
	var buf bytes.Buffer
	if err := renderTable(&buf, rows); err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	rendered := buf.String()
	if !strings.Contains(rendered, "main's") {
		t.Errorf("CLAIMED row does not name whose tip age it is:\n%s", rendered)
	}
	if strings.Contains(rendered, "10h0m0s") {
		t.Errorf("CLAIMED row prints main's age as the branch's:\n%s", rendered)
	}
	if !strings.Contains(rendered, "CLAIMED") {
		t.Errorf("CLAIMED row does not carry the verdict:\n%s", rendered)
	}
}

// TestRenderTableRetiredBorrowedAge is the same guard on the RETIRED rows.
// Retirement changes the verdict, not whose clock the tip is, so a retired
// branch that never committed must not print main's age under LEASE AGE
// either (#809).
func TestRenderTableRetiredBorrowedAge(t *testing.T) {
	borrowedTip := fixedNow.Add(-10 * time.Hour)
	issue := &issueState{number: 1, closed: true}
	got, lease, reason := classify("wip/issue-1", &borrowedTip, ancestryNoCommits, fixedNow, issue)
	if got != retired {
		t.Fatalf("classify with a closed issue = %s, want RETIRED", got)
	}
	if lease != nil {
		t.Errorf("retired zero-commit branch dated its lease %v; a borrowed tip dates nothing", *lease)
	}
	var buf bytes.Buffer
	if err := renderTable(&buf, []row{{issue: 1, branch: "wip/issue-1", lease: lease, anc: ancestryNoCommits, verdict: got, reason: reason}}); err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	cell := strings.Fields(strings.Split(buf.String(), "\n")[1])[2]
	if cell != "main's" {
		t.Errorf("RETIRED zero-commit LEASE AGE cell = %q, want `main's`", cell)
	}
}

// TestRenderTableHeartbeatAge checks an empty claim the thread dated
// prints the HEARTBEAT's age, under a header that says LEASE AGE rather
// than TIP AGE — the two clocks differ on exactly these rows, and the one
// that decided the verdict is the one the column owes the reader.
func TestRenderTableHeartbeatAge(t *testing.T) {
	hb := 44 * time.Minute
	rows := []row{
		{issue: 1, branch: "wip/issue-1", lease: &hb, anc: ancestryNoCommits, verdict: live, reason: "wip/issue-1: no commits of its own; lease dated by its newest RESUME:/TAKEOVER: comment"},
	}
	var buf bytes.Buffer
	if err := renderTable(&buf, rows); err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "LEASE AGE") {
		t.Errorf("header does not say whose age the column is:\n%s", got)
	}
	if !strings.Contains(got, "44m0s") {
		t.Errorf("heartbeat-dated row does not print the heartbeat's age:\n%s", got)
	}

	// An empty claim no heartbeat ever dated has no age of its own to
	// print. It stays CLAIMED and says whose the tip is, rather than
	// borrowing main's age as a duration.
	undated := []row{
		{issue: 2, branch: "wip/issue-2", lease: nil, anc: ancestryNoCommits, verdict: claimed, reason: "wip/issue-2: no commits of its own and no RESUME:/TAKEOVER: comment ever posted"},
	}
	buf.Reset()
	if err := renderTable(&buf, undated); err != nil {
		t.Fatalf("renderTable: %v", err)
	}
	if cell := strings.Fields(strings.Split(buf.String(), "\n")[1])[2]; cell != "main's" {
		t.Errorf("undated empty claim age cell = %q, want `main's`", cell)
	}
}

// TestRenderTableDeterministic guards STYLE D1: rendering the same rows
// twice must produce byte-identical output.
func TestRenderTableDeterministic(t *testing.T) {
	age := 30 * time.Minute
	rows := []row{
		{issue: 1, branch: "wip/issue-1", lease: &age, verdict: live, reason: "wip/issue-1: tip pushed 30m0s ago, within the 2h0m0s claim TTL"},
	}

	var a, b bytes.Buffer
	if err := renderTable(&a, rows); err != nil {
		t.Fatalf("renderTable (a): %v", err)
	}
	if err := renderTable(&b, rows); err != nil {
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
