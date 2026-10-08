package main

import (
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// datedCorrection is the marker that exempts an edit to an entry already on
// the base: .claude/agents/chronicler.md Duty 1 corrects a figure where it
// stands and marks the edit with the correcting date (#1417).
var datedCorrection = regexp.MustCompile(`(?i)\bcorrected (in place )?\d{4}-\d{2}-\d{2}\b`)

// hunkHeader is a unified diff hunk header; groups 1 and 2 are the base
// side's start line and optional line count.
var hunkHeader = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+\d+(?:,(\d+))? @@`)

// logFileDiff is one docs/LOG/ file's part of a -U0 diff.
type logFileDiff struct {
	path    string
	created bool // absent at base: nothing of the base's to lose
	deleted bool // absent at head, renames included
	hunks   []logHunk
}

// logHunk is one -U0 hunk: base lines oldStart..oldStart+len(dels)-1
// replaced by adds. With no dels, adds go in after base line oldStart, and
// oldStart 0 is the start of the file.
type logHunk struct {
	oldStart int
	dels     []string
	adds     []string
}

// checkLogHistory charges every line the branch deletes or displaces from a
// docs/LOG/ file present at base — docs/WORKFLOW.md's Merge-conflict
// resolution rule that a docs/LOG tail resolves positionally, run as Landing
// precondition 1's second half. It reads `git diff -U0 base...head --
// docs/LOG/` and needs merge-base(base, head) == base, which checkBaseCurrent
// has established: only then is every hunk's left side the base itself.
//
// Each hunk must (a) delete no base line, and, inserting only, (b) insert at
// an entry boundary — the first non-blank base line after the insertion
// begins `## `, or there is none — and (c) open its first non-blank inserted
// line with `## `, so the branch adds whole entries only (#2506's
// GROUNDING). A hunk that deletes is reported under (a) alone.
//
// A hunk that breaks one of them is exempt, and reported as exempt, when
// any of its added lines carries datedCorrection: a correction may rewrite
// a base line or add one inside a base entry. GAP(landcheck): the
// exemption covers the whole hunk, so a base line a forward merge lost, or
// branch lines it inserted mid-entry, in the same hunk as a dated
// correction passes: a false accept, RULED permanent by #2506 (its
// GROUNDING's dated-correction recognition is per hunk).
//
// It reports violations in diff order, then returns 1 when there is any.
func checkLogHistory(dir, base, head string, stdout io.Writer) (int, error) {
	out, err := exec.Command("git", "-C", dir, "diff", "-U0", "--minimal", "--no-renames", "--no-color", "--no-ext-diff",
		"--src-prefix=a/", "--dst-prefix=b/", base+"..."+head, "--", "docs/LOG/").Output()
	if err != nil {
		return 0, fmt.Errorf("running git diff -U0 %s...%s -- docs/LOG/: %w", base, head, err)
	}
	files, err := parseLogDiff(string(out))
	if err != nil {
		return 0, fmt.Errorf("reading git diff -U0 %s...%s -- docs/LOG/: %w", base, head, err)
	}

	var report []string
	violations := 0
	for _, f := range files {
		if f.created {
			continue
		}
		if f.deleted {
			report = append(report, fmt.Sprintf("%s: base file deleted or renamed: every line it carries on %s is gone", f.path, base))
			violations++
			continue
		}
		baseLines, err := blobLines(dir, base, f.path)
		if err != nil {
			return 0, err
		}
		for _, h := range f.hunks {
			found, err := h.violations(f.path, baseLines)
			if err != nil {
				return 0, err
			}
			if len(found) == 0 {
				continue
			}
			if marker := h.correction(); marker != "" {
				report = append(report, fmt.Sprintf("%s:%d: exempt, a dated correction: %s", f.path, h.oldStart, strings.TrimSpace(marker)))
				continue
			}
			report = append(report, found...)
			violations += len(found)
		}
	}

	summary := fmt.Sprintf("landcheck: docs/LOG/ keeps every line of %s in place\n", base)
	if violations > 0 {
		summary = fmt.Sprintf("landcheck: docs/LOG/ loses or displaces lines of %s, %d finding(s) below; resolve the tail positionally (docs/WORKFLOW.md Merge-conflict resolution)\n", base, violations)
	}
	if _, err := io.WriteString(stdout, summary); err != nil {
		return 0, err
	}
	for _, line := range report {
		if _, err := fmt.Fprintf(stdout, "landcheck:   %s\n", line); err != nil {
			return 0, err
		}
	}
	if violations > 0 {
		return 1, nil
	}
	return 0, nil
}

// correction returns the first added line carrying datedCorrection, or "".
func (h logHunk) correction() string {
	for _, line := range h.adds {
		if datedCorrection.MatchString(line) {
			return line
		}
	}
	return ""
}

// violations reports what h breaks of checkLogHistory's (a), (b) and (c)
// against base, the base file's lines.
func (h logHunk) violations(path string, base []string) ([]string, error) {
	if len(h.dels) > 0 {
		found := make([]string, 0, len(h.dels))
		for i, line := range h.dels {
			found = append(found, fmt.Sprintf("%s:%d: base line deleted: %q", path, h.oldStart+i, line))
		}
		return found, nil
	}
	if len(h.adds) == 0 {
		return nil, nil
	}
	if h.oldStart > len(base) {
		return nil, fmt.Errorf("%s: a hunk inserts after base line %d, but the base file has %d line(s)", path, h.oldStart, len(base))
	}
	var found []string
	if next := nextNonBlank(base, h.oldStart); next >= 0 && !isHeading(base[next]) {
		found = append(found, fmt.Sprintf("%s:%d: branch lines inserted inside a base entry, ahead of this base line: %q", path, next+1, base[next]))
	}
	if first := firstNonBlank(h.adds); first >= 0 && !isHeading(h.adds[first]) {
		found = append(found, fmt.Sprintf("%s:%d: branch lines open with %q, not a `## ` heading, after this base line: %q", path, h.oldStart, h.adds[first], lineAt(base, h.oldStart)))
	}
	return found, nil
}

// nextNonBlank is the index into base of the first non-blank line at or
// after index from, or -1.
func nextNonBlank(base []string, from int) int {
	for i := from; i < len(base); i++ {
		if strings.TrimSpace(base[i]) != "" {
			return i
		}
	}
	return -1
}

// firstNonBlank is the index of lines' first non-blank line, or -1.
func firstNonBlank(lines []string) int {
	return nextNonBlank(lines, 0)
}

// isHeading reports whether line opens a docs/LOG entry.
func isHeading(line string) bool {
	return strings.HasPrefix(line, "## ")
}

// lineAt is base line n (1-based), or a placeholder for the start of the
// file.
func lineAt(base []string, n int) string {
	if n == 0 {
		return "(start of file)"
	}
	return base[n-1]
}

// blobLines reads path as of rev and splits it into lines.
func blobLines(dir, rev, path string) ([]string, error) {
	out, err := exec.Command("git", "-C", dir, "cat-file", "-p", rev+":"+path).Output()
	if err != nil {
		return nil, fmt.Errorf("running git cat-file -p %s:%s: %w", rev, path, err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(string(out), "\n"), "\n"), nil
}

// parseLogDiff splits a `git diff -U0 --no-renames --src-prefix=a/
// --dst-prefix=b/` into files and hunks. Hunk bodies are read by the counts
// in their headers, so a content line that itself begins `--- ` or `+++ `
// is not mistaken for a file header.
//
// A base file's last line whose trailing newline the branch adds or drops
// prints as that line deleted and re-added; the pair is dropped here, so a
// hunk that also appends reads as the pure append it is.
func parseLogDiff(diff string) ([]logFileDiff, error) {
	var files []logFileDiff
	lines := strings.Split(diff, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(line, "diff --git ") {
			files = append(files, logFileDiff{})
			continue
		}
		if len(files) == 0 {
			continue
		}
		f := &files[len(files)-1]
		switch {
		case strings.HasPrefix(line, "new file mode"):
			f.created = true
		case strings.HasPrefix(line, "deleted file mode"):
			f.deleted = true
		case strings.HasPrefix(line, "--- a/"):
			f.path = strings.TrimPrefix(line, "--- a/")
		case strings.HasPrefix(line, "+++ b/") && f.path == "":
			f.path = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "@@ "):
			h, next, err := parseHunk(lines, i)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", f.path, err)
			}
			f.hunks = append(f.hunks, h)
			i = next - 1
		}
	}
	return files, nil
}

// parseHunk reads the hunk whose header is lines[at] and returns it with the
// index of the first line after it.
func parseHunk(lines []string, at int) (logHunk, int, error) {
	m := hunkHeader.FindStringSubmatch(lines[at])
	if m == nil {
		return logHunk{}, 0, fmt.Errorf("unreadable hunk header %q", lines[at])
	}
	oldStart, err := strconv.Atoi(m[1])
	if err != nil {
		return logHunk{}, 0, fmt.Errorf("hunk header %q: %w", lines[at], err)
	}
	oldCount, err := headerCount(m[2])
	if err != nil {
		return logHunk{}, 0, fmt.Errorf("hunk header %q: %w", lines[at], err)
	}
	newCount, err := headerCount(m[3])
	if err != nil {
		return logHunk{}, 0, fmt.Errorf("hunk header %q: %w", lines[at], err)
	}
	h := logHunk{oldStart: oldStart}
	i := at + 1
	noNewline := false
	for ; i < len(lines) && (len(h.dels) < oldCount || len(h.adds) < newCount || strings.HasPrefix(lines[i], `\`)); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, `\`):
			noNewline = true
		case strings.HasPrefix(line, "-") && len(h.dels) < oldCount:
			h.dels = append(h.dels, line[1:])
		case strings.HasPrefix(line, "+") && len(h.adds) < newCount:
			h.adds = append(h.adds, line[1:])
		default:
			return logHunk{}, 0, fmt.Errorf("hunk %q: unexpected line %q", lines[at], line)
		}
	}
	if len(h.dels) != oldCount || len(h.adds) != newCount {
		return logHunk{}, 0, fmt.Errorf("hunk %q ends after %d deleted and %d added line(s)", lines[at], len(h.dels), len(h.adds))
	}
	// A `\ No newline at end of file` marks either side's final line, so a
	// deleted and an added line that read the same differ in that newline
	// alone: the line itself survives.
	if noNewline && len(h.dels) > 0 && len(h.adds) > 0 && h.adds[0] == h.dels[len(h.dels)-1] {
		h.dels = h.dels[:len(h.dels)-1]
		h.adds = h.adds[1:]
	}
	return h, i, nil
}

// headerCount is a hunk header's line count, which git omits when it is 1.
func headerCount(s string) (int, error) {
	if s == "" {
		return 1, nil
	}
	return strconv.Atoi(s)
}
