// Command landcheck runs docs/WORKFLOW.md's Landing preconditions 1 and 2 by
// git, so a landing precondition is checked rather than remembered.
//
// Precondition 1 is not "the chronicler was invoked" and not a non-empty
// `docs/LOG/` path diff: a forward merge can carry another issue's entry into
// that diff, which reads PRESENT while the branch's own entry is absent
// (#813, `2d0a38d`). It is also not "some `docs/LOG/` path changed at all":
// a branch can add no `docs/LOG/` path whatsoever (#924, `53bf113`). The only
// check that distinguishes both from a real entry is: among `git diff
// <base>...HEAD -- docs/LOG/`'s ADDED lines, at least one names the issue
// being landed, as a whole `#<N>` or `issues/<N>` token — `#820` must not
// match inside `#8201` (#820's own diff contains both, `311ada8`).
//
// Precondition 2 — the base is current — is verified first, because the
// added-lines form is only sound when `merge-base(base, HEAD) == base`:
// against a stale base the merge-base is older than intended, a forward
// merge's entries reappear as added lines, and only the issue-number filter
// still separates them from the branch's own. A stale base is reported as an
// operational error (exit 2), not a defect (exit 1): the check did not run
// to a verdict, it declined to run at all.
//
// Usage:
//
//	go tool landcheck -issue 963                      # base defaults to origin/main
//	go tool landcheck -issue 963 -base origin/main
//
// Exit codes mirror tools/lint, not the report-only survey tools: 0 for a
// clean run (the entry is found and the base is current), 1 for a defect (no
// matching added line), 2 for an operational error (a stale base, a bad git
// ref, or git failing to run at all).
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	code, err := run(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "landcheck: %v\n", err)
		os.Exit(2)
	}
	os.Exit(code)
}

// run is main's testable body: parse flags, resolve the repo root so the
// check gives the same answer from any working directory, and delegate to
// checkLanding against the current HEAD.
func run(args []string, stdout io.Writer) (int, error) {
	fs := flag.NewFlagSet("landcheck", flag.ContinueOnError)
	issue := fs.Int("issue", 0, "issue number the branch's docs/LOG/ entry must name")
	base := fs.String("base", "origin/main", "git ref the branch is landing onto")
	if err := fs.Parse(args); err != nil {
		return 0, err
	}
	if *issue <= 0 {
		return 0, fmt.Errorf("-issue is required and must be a positive integer, got %d", *issue)
	}

	root, err := repoRoot()
	if err != nil {
		return 0, err
	}
	return checkLanding(root, *base, "HEAD", *issue, stdout)
}

// checkLanding verifies precondition 2 (base is current) and, only once that
// holds, precondition 1 (the added lines under docs/LOG/ name issue) between
// base and head in the git repository at dir. head is a parameter rather
// than a literal "HEAD" so tests can point it at a historical commit instead
// of the checkout's current branch tip.
func checkLanding(dir, base, head string, issue int, stdout io.Writer) (int, error) {
	if err := checkBaseCurrent(dir, base, head); err != nil {
		return 0, err
	}
	return checkLogEntry(dir, base, head, issue, stdout)
}

// checkBaseCurrent verifies docs/WORKFLOW.md's landing precondition 2:
// merge-base(base, head) == base, i.e. base is an ancestor of head with no
// divergence. A non-nil return is always an operational error (exit 2) per
// the issue's Acceptance item 2 — a stale base is not a defect the added-
// lines check can even meaningfully report on.
func checkBaseCurrent(dir, base, head string) error {
	cmd := exec.Command("git", "-C", dir, "merge-base", "--is-ancestor", base, head)
	err := cmd.Run()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return fmt.Errorf("base %q is not current: it is not an ancestor of %s — fetch %q before landing (docs/WORKFLOW.md Landing precondition 2)", base, head, base)
	}
	return fmt.Errorf("running git merge-base --is-ancestor %s %s: %w", base, head, err)
}

// checkLogEntry runs docs/WORKFLOW.md's landing precondition 1: at least one
// ADDED line under docs/LOG/ between base and head names issue as a whole
// token. It always writes what it saw to stdout, on both the clean and the
// defect path, since exit code alone does not say which of the two failure
// shapes (#924's — no docs/LOG/ diff at all — or #813's — a diff naming a
// different issue) a caller ran into.
func checkLogEntry(dir, base, head string, issue int, stdout io.Writer) (int, error) {
	diff, err := gitDiffLog(dir, base, head)
	if err != nil {
		return 0, err
	}

	added := addedLines(diff)
	if len(added) == 0 {
		_, err := fmt.Fprintf(stdout, "landcheck: no docs/LOG/ changes between %s and %s\n", base, head)
		return 1, err
	}

	pattern := issueTokenPattern(issue)
	for _, line := range added {
		if pattern.MatchString(line) {
			_, err := fmt.Fprintf(stdout, "landcheck: docs/LOG/ names #%d: %s\n", issue, strings.TrimSpace(line))
			return 0, err
		}
	}
	_, err = fmt.Fprintf(stdout, "landcheck: docs/LOG/ diff present (%d added line(s)) but none names #%d\n", len(added), issue)
	return 1, err
}

// gitDiffLog runs the exact command docs/WORKFLOW.md's precondition 1 names:
// `git diff <base>...<head> -- docs/LOG/`.
func gitDiffLog(dir, base, head string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "diff", base+"..."+head, "--", "docs/LOG/").Output()
	if err != nil {
		return "", fmt.Errorf("running git diff %s...%s -- docs/LOG/: %w", base, head, err)
	}
	return string(out), nil
}

// addedLines extracts a unified diff's added content lines: those starting
// with "+", excluding the "+++" file header, with the leading "+" stripped.
// It is pure text parsing, so tests exercise it directly against literal
// diff output.
func addedLines(diff string) []string {
	var lines []string
	for _, line := range strings.Split(diff, "\n") {
		if !strings.HasPrefix(line, "+") {
			continue
		}
		if strings.HasPrefix(line, "+++") {
			continue
		}
		lines = append(lines, strings.TrimPrefix(line, "+"))
	}
	return lines
}

// issueTokenPattern matches issue as a whole token, either bare (#<N>) or as
// an issues/<N> path — the two forms docs/WORKFLOW.md's precondition 1
// names (its third alternative, \(#<N>\), is already a substring of the
// bare form and is not restated). \b after the digits is what rejects a
// longer number sharing issue's prefix: `#820` must not match inside
// `#8201`, a real case in this repo's own log (docs/LOG/2026-08.md, added by
// `311ada8`).
func issueTokenPattern(issue int) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(`(#|issues/)%d\b`, issue))
}

// repoRoot resolves the git repository's top-level directory, so the check
// gives the same answer run from any working directory inside the checkout
// — mirroring tools/lint's moduleRoot, for the same reason.
func repoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("running git rev-parse --show-toplevel: %w", err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("git rev-parse --show-toplevel returned no path")
	}
	return root, nil
}
