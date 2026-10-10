package main

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	// restamp is the boilerplate line most entries end in: the line git
	// factors out of a docs/LOG tail conflict as common suffix (#2413).
	restamp = "  - Restamp nothing in `docs/PLAN.md` until the next stamp (#1630).\n"

	// logBase is docs/LOG/2026-10.md as main carries it when the branch is
	// cut: lines 1-9, line 9 the first entry's restamp.
	logBase = "# Session Log — 2026-10\n" +
		"\n" +
		"Append-only.\n" +
		"\n" +
		"## 2026-10-06 — #2400 — landed\n" +
		"\n" +
		"- Shipped x.\n" +
		"  - Close #2400.\n" +
		restamp

	// mainEntry is another session's entry, landed on main after the branch
	// was cut: lines 10-15 of main's file, line 15 its restamp.
	mainEntry = "\n" +
		"## 2026-10-07 — #2473 — landed\n" +
		"\n" +
		"- Shipped y.\n" +
		"  - Close #2473.\n" +
		restamp

	// ownEntry is the branch's own entry, naming the issue being landed.
	ownEntry = "\n" +
		"## 2026-10-08 — #2506 — landed\n" +
		"\n" +
		"- Shipped z, measured at 3 sites.\n" +
		"  - Close #2506.\n" +
		restamp

	// postLand is a post-land entry following mainEntry: lines 16-19 of
	// main's file when both are on it.
	postLand = "\n" +
		"## 2026-10-07 — post-land (#2473) — nothing unblocked\n" +
		"\n" +
		"- Closed #2473.\n"

	logPath = "docs/LOG/2026-10.md"
)

// newLogBranch builds a clone under t.TempDir() with a bare remote as
// origin: main carries docs/LOG/2026-10.md as log, pushed, and branch
// wip/issue-2506 is cut from it with its upstream set. It returns the
// clone's path.
func newLogBranch(t *testing.T, log string) string {
	t.Helper()
	root := t.TempDir()
	remote := root + "/remote.git"
	work := root + "/work"
	gitIn(t, root, "init", "-q", "--bare", "-b", "main", remote)
	gitIn(t, root, "init", "-q", "-b", "main", work)
	gitIn(t, work, "remote", "add", "origin", remote)
	commitFile(t, work, logPath, log, "base")
	gitIn(t, work, "push", "-q", "-u", "origin", "main")
	gitIn(t, work, "checkout", "-q", "-b", "wip/issue-2506")
	gitIn(t, work, "push", "-q", "-u", "origin", "wip/issue-2506")
	return work
}

// commitFile writes content to rel in dir and commits it.
func commitFile(t *testing.T, dir, rel, content, msg string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(dir+"/"+rel), 0o755); err != nil {
		t.Fatalf("creating the directory of %s: %v", rel, err)
	}
	if err := os.WriteFile(dir+"/"+rel, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", rel, err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", msg)
}

// landOnMain commits log as main's docs/LOG/2026-10.md, pushes it and
// returns to the branch: another session landing while this one works.
func landOnMain(t *testing.T, dir, log string) {
	t.Helper()
	gitIn(t, dir, "checkout", "-q", "main")
	commitFile(t, dir, logPath, log, "another session lands")
	gitIn(t, dir, "push", "-q")
	gitIn(t, dir, "checkout", "-q", "wip/issue-2506")
}

// mergeMain merges origin/main forward, which must conflict in
// docs/LOG/2026-10.md, writes resolve's answer to the conflicted file and
// commits the merge.
func mergeMain(t *testing.T, dir string, resolve func(t *testing.T, conflicted string) string) {
	t.Helper()
	if out, err := fixtureGit(dir, "merge", "-q", "origin/main").CombinedOutput(); err == nil {
		t.Fatalf("git merge origin/main: want a docs/LOG conflict, got a clean merge\n%s", out)
	}
	data, err := os.ReadFile(dir + "/" + logPath)
	if err != nil {
		t.Fatalf("reading the conflicted %s: %v", logPath, err)
	}
	if err := os.WriteFile(dir+"/"+logPath, []byte(resolve(t, string(data))), 0o644); err != nil {
		t.Fatalf("writing the resolved %s: %v", logPath, err)
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "--no-edit")
}

// positional resolves every conflict region the way docs/WORKFLOW.md's tail
// rule reads at a glance: main's side, then the branch's. It first requires
// the #2413 shape — the shared restamp line after the last region's
// closing marker, factored out of both sides — so the fixture fails loudly
// if git stops producing it.
func positional(t *testing.T, conflicted string) string {
	t.Helper()
	end := strings.LastIndex(conflicted, ">>>>>>> ")
	if end < 0 || !strings.Contains(conflicted[end:], restamp) {
		t.Fatalf("conflict is not #2413's shape: want the shared restamp line after the last >>>>>>> marker\n%s", conflicted)
	}
	var out, ours, theirs strings.Builder
	side := 0 // 0 outside a region, 1 in the branch's half, 2 in main's
	for _, line := range strings.SplitAfter(conflicted, "\n") {
		switch {
		case strings.HasPrefix(line, "<<<<<<< "):
			side = 1
		case strings.HasPrefix(line, "=======") && side == 1:
			side = 2
		case strings.HasPrefix(line, ">>>>>>> ") && side == 2:
			out.WriteString(theirs.String() + "\n" + ours.String())
			ours.Reset()
			theirs.Reset()
			side = 0
		case side == 1:
			ours.WriteString(line)
		case side == 2:
			theirs.WriteString(line)
		default:
			out.WriteString(line)
		}
	}
	return out.String()
}

// tailResolved is the correct resolution: main's file, then the branch's
// entry whole.
func tailResolved(t *testing.T, _ string) string {
	return logBase + mainEntry + ownEntry
}

// TestRunLogHistory drives run against fixture branches whose docs/LOG/
// diff against origin/main does or does not lose or displace a base line.
// Every branch but the -no-issue row carries an entry naming #2506, so
// precondition 1's entry check passes and each non-zero code is the
// history check's.
func TestRunLogHistory(t *testing.T) {
	tests := []struct {
		name    string
		noIssue bool
		// log is main's docs/LOG/2026-10.md when the branch is cut.
		log string
		// arrange takes newLogBranch(log)'s clone to the case's HEAD; the
		// test pushes it afterwards.
		arrange func(t *testing.T, dir string)
		// wantCode is run's exit code; want is its stdout up to the
		// closing-keyword lines.
		wantCode int
		want     string
	}{
		{
			name: "append only: clean",
			log:  logBase,
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, logPath, logBase+ownEntry, "log")
			},
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ keeps every line of origin/main in place\n",
		},
		{
			name: "append after a forward merge resolved at the tail: clean",
			log:  logBase,
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, logPath, logBase+ownEntry, "log")
				landOnMain(t, dir, logBase+mainEntry)
				mergeMain(t, dir, tailResolved)
			},
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ keeps every line of origin/main in place\n",
		},
		{
			name: "2413 shape: positional resolution splits main's entry from its restamp: defect",
			log:  logBase,
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, logPath, logBase+ownEntry, "log")
				landOnMain(t, dir, logBase+mainEntry)
				mergeMain(t, dir, positional)
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ loses or displaces lines of origin/main, 1 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md:15: branch lines inserted inside a base entry, ahead of this base line: \"  - Restamp nothing in `docs/PLAN.md` until the next stamp (#1630).\"\n",
		},
		{
			name: "forward merge resolved to the branch's side alone: defect",
			log:  logBase,
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, logPath, logBase+ownEntry, "log")
				landOnMain(t, dir, logBase+mainEntry)
				mergeMain(t, dir, func(t *testing.T, _ string) string { return logBase + ownEntry })
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ loses or displaces lines of origin/main, 3 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md:11: base line deleted: \"## 2026-10-07 — #2473 — landed\"\n" +
				"landcheck:   docs/LOG/2026-10.md:13: base line deleted: \"- Shipped y.\"\n" +
				"landcheck:   docs/LOG/2026-10.md:14: base line deleted: \"  - Close #2473.\"\n",
		},
		{
			// 98fec4a's shape: the branch's entry lands inside main's last
			// entry but one, ahead of its restamp, which no line is deleted
			// for, and a stray bullet lands after the last entry.
			name: "98fec4a shape: entry inside a base entry, bullet after another: defect",
			log:  logBase + mainEntry + postLand,
			arrange: func(t *testing.T, dir string) {
				head := logBase + strings.TrimSuffix(mainEntry, restamp) + strings.TrimSuffix(ownEntry, restamp) + restamp +
					postLand + "\n- Merged forward before landing: nothing.\n"
				commitFile(t, dir, logPath, head, "log")
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ loses or displaces lines of origin/main, 2 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md:15: branch lines inserted inside a base entry, ahead of this base line: \"  - Restamp nothing in `docs/PLAN.md` until the next stamp (#1630).\"\n" +
				"landcheck:   docs/LOG/2026-10.md:19: branch lines open with \"- Merged forward before landing: nothing.\", not a `## ` heading, after this base line: \"- Closed #2473.\"\n",
		},
		{
			name: "bullet appended after the last base entry: defect",
			log:  logBase + mainEntry,
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, "docs/LOG/2026-11.md", ownEntry, "log")
				commitFile(t, dir, logPath, logBase+mainEntry+"- Merged forward before landing: nothing.\n", "log")
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ loses or displaces lines of origin/main, 1 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md:15: branch lines open with \"- Merged forward before landing: nothing.\", not a `## ` heading, after this base line: \"  - Restamp nothing in `docs/PLAN.md` until the next stamp (#1630).\"\n",
		},
		{
			name: "whole entry inserted between two base entries: clean",
			log:  logBase + mainEntry,
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, logPath, logBase+ownEntry+mainEntry, "log")
			},
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ keeps every line of origin/main in place\n",
		},
		{
			name: "append to a base file lacking its final newline: clean",
			log:  strings.TrimSuffix(logBase, "\n"),
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, logPath, logBase+ownEntry, "log")
			},
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ keeps every line of origin/main in place\n",
		},
		{
			// The second commit edits a line of the first's entry, and its
			// dated marker sits two lines away, outside the -U0 hunk that
			// edit makes: a diff from the branch's first commit would charge
			// the edit, and the three-dot diff from origin/main sees one
			// insertion.
			name: "dated in-place correction to the branch's own entry: clean",
			log:  logBase,
			arrange: func(t *testing.T, dir string) {
				commitFile(t, dir, logPath, logBase+ownEntry, "log")
				corrected := strings.Replace(ownEntry, "3 sites", "4 sites", 1) + "- Site count corrected 2026-10-08: the first count missed one.\n"
				commitFile(t, dir, logPath, logBase+corrected, "log: correct the site count")
			},
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ keeps every line of origin/main in place\n",
		},
		{
			name: "dated in-place correction to an entry on main: clean, exempt",
			log:  logBase + mainEntry,
			arrange: func(t *testing.T, dir string) {
				head := strings.Replace(logBase+mainEntry, "- Shipped y.", "- Shipped y and w (corrected 2026-10-08: w was omitted).", 1) + ownEntry
				commitFile(t, dir, logPath, head, "log")
			},
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ keeps every line of origin/main in place\n" +
				"landcheck:   docs/LOG/2026-10.md:13: exempt, a dated correction: - Shipped y and w (corrected 2026-10-08: w was omitted).\n",
		},
		{
			name: "the same correction undated: defect",
			log:  logBase + mainEntry,
			arrange: func(t *testing.T, dir string) {
				head := strings.Replace(logBase+mainEntry, "- Shipped y.", "- Shipped y and w (w was omitted).", 1) + ownEntry
				commitFile(t, dir, logPath, head, "log")
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ loses or displaces lines of origin/main, 1 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md:13: base line deleted: \"- Shipped y.\"\n",
		},
		{
			name: "base LOG file deleted: defect",
			log:  logBase,
			arrange: func(t *testing.T, dir string) {
				gitIn(t, dir, "rm", "-q", logPath)
				commitFile(t, dir, "docs/LOG/2026-11.md", ownEntry, "log")
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ loses or displaces lines of origin/main, 1 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md: base file deleted or renamed: every line it carries on origin/main is gone\n",
		},
		{
			name: "base LOG file renamed: defect",
			log:  logBase,
			arrange: func(t *testing.T, dir string) {
				gitIn(t, dir, "mv", logPath, "docs/LOG/2026-10-old.md")
				commitFile(t, dir, "docs/LOG/2026-11.md", ownEntry, "log")
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ names #2506: ## 2026-10-08 — #2506 — landed\n" +
				"landcheck: docs/LOG/ loses or displaces lines of origin/main, 1 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md: base file deleted or renamed: every line it carries on origin/main is gone\n",
		},
		{
			name:    "no-issue: a post-land pass rewrites a base line undated: defect",
			noIssue: true,
			log:     logBase + mainEntry,
			arrange: func(t *testing.T, dir string) {
				head := strings.Replace(logBase+mainEntry, "- Shipped y.", "- Shipped y and w.", 1) + postLand
				commitFile(t, dir, logPath, head, "log")
			},
			wantCode: 1,
			want: "landcheck: docs/LOG/ loses or displaces lines of origin/main, 1 finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n" +
				"landcheck:   docs/LOG/2026-10.md:13: base line deleted: \"- Shipped y.\"\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := newLogBranch(t, tc.log)
			tc.arrange(t, dir)
			gitIn(t, dir, "push", "-q")
			t.Chdir(dir)

			flags := []string{"-issue", "2506"}
			squash, prBody := "log: landed (#2506)\n\nCloses #2506.\n", "Closes #2506.\n"
			if tc.noIssue {
				flags = []string{"-no-issue"}
				squash, prBody = "meta: post-land pass\n", "Post-land pass.\n"
			}
			var buf bytes.Buffer
			code, err := run(append(flags, textArgs(t, squash, prBody)...), &buf)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			got := buf.String()
			// The closing-keyword lines are TestRunClosingKeywords'; this
			// test owns the docs/LOG lines before them.
			if i := strings.Index(got, "landcheck: squash text:"); i >= 0 {
				got = got[:i]
			}
			if code != tc.wantCode || got != tc.want {
				t.Errorf("code = %d, output:\n%s\nwant code %d, output:\n%s", code, got, tc.wantCode, tc.want)
				data, _ := os.ReadFile(dir + "/" + logPath)
				t.Logf("HEAD's %s:\n%s", logPath, data)
			}
		})
	}
}

// TestParseLogDiff reads literal -U0 diffs: a hunk body is read by its
// header's counts, and a final line deleted and re-added for its newline
// alone is no change.
func TestParseLogDiff(t *testing.T) {
	tests := []struct {
		name string
		diff string
		want []logFileDiff
	}{
		{
			name: "content lines that read as file headers",
			diff: "diff --git a/docs/LOG/x.md b/docs/LOG/x.md\n" +
				"--- a/docs/LOG/x.md\n" +
				"+++ b/docs/LOG/x.md\n" +
				"@@ -3 +3,2 @@ ctx\n" +
				"--- a/old\n" +
				"+++ b/new\n" +
				"+## heading\n",
			want: []logFileDiff{{path: "docs/LOG/x.md", hunks: []logHunk{{oldStart: 3, dels: []string{"-- a/old"}, adds: []string{"++ b/new", "## heading"}}}}},
		},
		{
			name: "base lacks its final newline, the branch appends",
			diff: "diff --git a/docs/LOG/x.md b/docs/LOG/x.md\n" +
				"--- a/docs/LOG/x.md\n" +
				"+++ b/docs/LOG/x.md\n" +
				"@@ -9 +9,3 @@ ctx\n" +
				"-last\n" +
				"\\ No newline at end of file\n" +
				"+last\n" +
				"+\n" +
				"+## next\n",
			want: []logFileDiff{{path: "docs/LOG/x.md", hunks: []logHunk{{oldStart: 9, dels: []string{}, adds: []string{"", "## next"}}}}},
		},
		{
			name: "the branch drops the final newline",
			diff: "diff --git a/docs/LOG/x.md b/docs/LOG/x.md\n" +
				"--- a/docs/LOG/x.md\n" +
				"+++ b/docs/LOG/x.md\n" +
				"@@ -9 +12 @@ ctx\n" +
				"-last\n" +
				"+last\n" +
				"\\ No newline at end of file\n",
			want: []logFileDiff{{path: "docs/LOG/x.md", hunks: []logHunk{{oldStart: 9, dels: []string{}, adds: []string{}}}}},
		},
		{
			name: "a deleted file and a created one",
			diff: "diff --git a/docs/LOG/a.md b/docs/LOG/a.md\n" +
				"deleted file mode 100644\n" +
				"--- a/docs/LOG/a.md\n" +
				"+++ /dev/null\n" +
				"@@ -1 +0,0 @@\n" +
				"-gone\n" +
				"diff --git a/docs/LOG/b.md b/docs/LOG/b.md\n" +
				"new file mode 100644\n" +
				"--- /dev/null\n" +
				"+++ b/docs/LOG/b.md\n" +
				"@@ -0,0 +1 @@\n" +
				"+new\n",
			want: []logFileDiff{
				{path: "docs/LOG/a.md", deleted: true, hunks: []logHunk{{oldStart: 1, dels: []string{"gone"}}}},
				{path: "docs/LOG/b.md", created: true, hunks: []logHunk{{oldStart: 0, adds: []string{"new"}}}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLogDiff(tc.diff)
			if err != nil {
				t.Fatalf("parseLogDiff: %v", err)
			}
			if !slices.EqualFunc(got, tc.want, equalFileDiff) {
				t.Errorf("parseLogDiff =\n%+v\nwant\n%+v", got, tc.want)
			}
		})
	}
}

func equalFileDiff(a, b logFileDiff) bool {
	return a.path == b.path && a.created == b.created && a.deleted == b.deleted &&
		slices.EqualFunc(a.hunks, b.hunks, func(x, y logHunk) bool {
			return x.oldStart == y.oldStart && slices.Equal(x.dels, y.dels) && slices.Equal(x.adds, y.adds)
		})
}
