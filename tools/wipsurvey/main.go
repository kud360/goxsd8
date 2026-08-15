// Command wipsurvey classifies every wip/issue-<N> branch on the origin
// remote as LIVE, CLAIMED, EXPIRED, RETIRED, or UNKNOWN, so a /develop
// session does not have to re-derive by hand which branches are contended
// and which are dead. Issue #399 measured seven consecutive develop
// passes each re-deriving the same three unchanged facts about the same
// dead branches; PRINCIPLES 27 turns exactly that shape of repetitive,
// deterministic work into a tool.
//
// Classification needs two sources that neither shows on its own:
//   - the branch namespace, for how recently a branch's tip was pushed
//     (WORKFLOW.md's 2-hour claim TTL, PRINCIPLES 28); and
//   - the GitHub issue thread, for whether the branch's owner issue has
//     since closed or been retired with needs-replan — retirement is
//     recorded on the issue, never in the branch namespace, so a survey
//     built from git alone cannot see it.
//
// The branch set itself must come from the remote, not from local
// remote-tracking refs. `git for-each-ref refs/remotes/origin/wip/*`
// returns every branch this checkout has EVER seen, including ones GitHub
// auto-deleted at merge that nobody has since `git fetch --prune`d away —
// reporting those as LIVE or EXPIRED invents in-flight work out of stale
// local state, exactly backwards for a tool whose purpose is telling a
// session what still exists. `git ls-remote --heads origin` asks the
// remote directly instead, matching the WIP discovery index WORKFLOW.md
// and PRINCIPLES 28 both name authoritative.
//
// ls-remote reports only a SHA per branch, not a date, so each tip's
// commit time still comes from the local object store (`git log -1
// --format=%cI <sha>`). When that SHA has never been fetched into this
// checkout, the tool does not guess an age for it: a guessed age could
// read as EXPIRED (resumable), the dangerous direction for a lease it has
// no data behind. It reports the branch UNKNOWN instead and says to
// fetch.
//
// A tip age is only evidence about the branch that authored it. A
// wip/issue-<N> branch pushed as a bare claim, with no commits yet, points
// at whatever main pointed at when the session started, so its tip age
// measures the PREVIOUS LANDING and not the claim: a claim made minutes
// ago reports as hours old, EXPIRED, and therefore resumable, and the
// session that acts on it races a live holder (#722). Such a branch is an
// ancestor of main, which `git merge-base --is-ancestor` decides against
// the main SHA the same ls-remote round trip already fetched. Those
// branches are reported CLAIMED: the branch name establishes a claim,
// nothing establishes its age, so the branch is off-limits for retirement
// and is not resumable on age evidence alone. What settles a CLAIMED
// branch is the issue thread's own clock, which this tool does not read.
//
// That ancestry test needs main's commit object in this checkout, while
// the SHA it tests against comes live from ls-remote, so a checkout that
// last fetched before the most recent landing cannot decide it. Such a row
// falls back to its tip age and its reason says the ancestry was
// undecided, which is what makes a LIVE or EXPIRED there provisional
// rather than settled (#806).
//
// Usage:
//
//	git fetch origin
//	gh issue list --state all --json number,state,labels | go tool wipsurvey
//	go tool wipsurvey < /dev/null   # issue data omitted: lease-only report
//
// Exit status is 0 for a normal report (whatever the branches' verdicts
// turn out to be) and 2 for an operational error: git is not on PATH, the
// remote is unreachable, or stdin carries malformed JSON.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

// claimTTL is the lease window WORKFLOW.md's branch scheme gives a pushed
// wip/ branch before another session may resume it: "tip newer than the
// claim TTL (2 hours) -> LIVE ... tip older than the TTL -> EXPIRED."
const claimTTL = 2 * time.Hour

func main() {
	if err := run(os.Stdout, os.Stderr, os.Stdin, time.Now()); err != nil {
		fmt.Fprintf(os.Stderr, "wipsurvey: %v\n", err)
		os.Exit(2)
	}
}

// run drives the survey end to end: discover branches from the remote,
// read optional issue data from stdin, classify each branch, and render
// the report to stdout. A branch whose name does not fit the
// wip/issue-<N> shape is skipped with a warning to stderr rather than
// aborting the whole report — an unexpected refs/heads/wip/* branch is
// evidence worth a warning, not a reason to withhold every other row.
func run(stdout, stderr io.Writer, stdin io.Reader, now time.Time) error {
	refs, mainSHA, err := remoteRefs()
	if err != nil {
		return err
	}

	var parsed []branchRef
	for _, ref := range refs {
		br, err := parseWipRef(ref.sha, ref.ref)
		if err != nil {
			// Best-effort diagnostic: a write failure here would mean stderr
			// itself is broken, which cannot change what this loop still
			// owes the caller — skip the unparseable ref and keep surveying
			// the rest.
			_, _ = fmt.Fprintf(stderr, "wipsurvey: skipping %s: %v\n", ref.ref, err)
			continue
		}
		parsed = append(parsed, br)
	}

	issues, err := readIssues(stdin)
	if err != nil {
		return err
	}

	rows := make([]row, 0, len(parsed))
	for _, br := range parsed {
		tip, err := gitTip(br.sha)
		if err != nil {
			return fmt.Errorf("resolving tip for %s: %w", br.branch, err)
		}
		anc, err := gitAncestry(br.sha, mainSHA)
		if err != nil {
			return fmt.Errorf("resolving ancestry for %s: %w", br.branch, err)
		}
		var issue *issueState
		if state, ok := issues[br.issue]; ok {
			issue = &state
		}
		got, reason := classify(br.branch, tip, anc, now, issue)
		rows = append(rows, row{issue: br.issue, branch: br.branch, tip: tip, verdict: got, reason: reason})
	}
	sortRows(rows)

	return renderTable(stdout, rows, now)
}

// refSHA is one line of `git ls-remote --heads` output: a remote head's
// SHA paired with its full ref name.
type refSHA struct {
	sha string
	ref string
}

// remoteRefs asks the origin remote directly for every wip/* branch's head
// SHA and, in the same round trip, main's — one network call for both, so
// the ancestry test costs no extra remote access. See the package doc
// comment for why this — not the local refs/remotes/origin/wip/* cache —
// is the authoritative branch set. The main SHA is empty when the remote
// reports no refs/heads/main, which leaves every ancestry unresolved and
// the report dated from tip ages alone.
func remoteRefs() ([]refSHA, string, error) {
	out, err := exec.Command("git", "ls-remote", "--heads", "origin", "refs/heads/wip/*", "refs/heads/main").Output()
	if err != nil {
		return nil, "", fmt.Errorf("running git ls-remote --heads origin: %w", err)
	}
	refs, errs := parseLsRemote(string(out))
	if len(errs) > 0 {
		return nil, "", fmt.Errorf("parsing git ls-remote output: %w", errors.Join(errs...))
	}
	wips, mainSHA := partitionRefs(refs)
	return wips, mainSHA, nil
}

// partitionRefs splits ls-remote's refs into the wip/* branches to survey
// and main's SHA, which is the ancestry test's right-hand side rather than
// a surveyed branch. It is pure — no git or process calls — so tests
// exercise it directly.
func partitionRefs(refs []refSHA) ([]refSHA, string) {
	var wips []refSHA
	mainSHA := ""
	for _, ref := range refs {
		if ref.ref == "refs/heads/main" {
			mainSHA = ref.sha
			continue
		}
		wips = append(wips, ref)
	}
	return wips, mainSHA
}

// parseLsRemote parses `git ls-remote --heads` output, one "<sha>\t<ref>"
// pair per line. It is pure text parsing — no git or process calls — so
// tests exercise it directly against literal ls-remote output instead of
// a repository.
func parseLsRemote(output string) ([]refSHA, []error) {
	var refs []refSHA
	var errs []error
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if line == "" {
			continue
		}
		sha, ref, ok := strings.Cut(line, "\t")
		if !ok {
			errs = append(errs, fmt.Errorf("malformed ls-remote line %q: expected \"<sha>\\t<ref>\"", line))
			continue
		}
		refs = append(refs, refSHA{sha: sha, ref: ref})
	}
	return refs, errs
}

// branchRef is one refs/heads/wip/issue-<N> branch resolved to its issue
// number.
type branchRef struct {
	sha    string
	branch string
	issue  int
}

// wipBranchPattern matches the one-issue-one-branch shape WORKFLOW.md
// mandates: "wip/issue-<N>", N a decimal integer with no leading zero.
var wipBranchPattern = regexp.MustCompile(`^wip/issue-([1-9][0-9]*)$`)

// parseWipRef validates that ref is a refs/heads/wip/issue-<N> branch and
// extracts its branch name and issue number. It is pure — no git or
// process calls — so tests exercise it directly, including the malformed
// shapes it must reject.
func parseWipRef(sha, ref string) (branchRef, error) {
	branch, ok := strings.CutPrefix(ref, "refs/heads/")
	if !ok {
		return branchRef{}, fmt.Errorf("ref %q is not under refs/heads/", ref)
	}
	m := wipBranchPattern.FindStringSubmatch(branch)
	if m == nil {
		return branchRef{}, fmt.Errorf("branch %q does not match wip/issue-<N>", branch)
	}
	issue, err := strconv.Atoi(m[1])
	if err != nil {
		return branchRef{}, fmt.Errorf("parsing issue number from branch %q: %w", branch, err)
	}
	return branchRef{sha: sha, branch: branch, issue: issue}, nil
}

// gitTip resolves sha's committer time from the local object store. It
// returns a nil time, with no error, when git reports the object is not
// present locally (the SHA ls-remote reported has never been fetched into
// this checkout) — that is an expected state the caller turns into an
// UNKNOWN verdict, not a failure this tool aborts on. A nil time is never
// returned alongside a non-nil error: callers only need to check err to
// know which case they are in.
func gitTip(sha string) (*time.Time, error) {
	out, err := exec.Command("git", "log", "-1", "--format=%cI", sha, "--").Output()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("running git log -1 for %s: %w", sha, err)
	}
	ts := strings.TrimSpace(string(out))
	tip, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return nil, fmt.Errorf("parsing committer date %q for %s: %w", ts, sha, err)
	}
	return &tip, nil
}

// ancestry is what this checkout could establish about a branch tip's
// position relative to main: whether the branch carries commits of its
// own. Its zero value is ancestryUnresolved, so a caller with no answer
// carries none.
type ancestry int

const (
	// ancestryUnresolved means git could not decide — the objects are not
	// in this checkout, or the remote reported no main. Callers fall back
	// to the tip age rather than inventing a second unknown verdict, and
	// say in the reason that they did.
	ancestryUnresolved ancestry = iota
	// ancestryOwnCommits means the branch is not an ancestor of main: its
	// tip is a commit the branch itself authored, so the tip age is the
	// claim's age.
	ancestryOwnCommits
	// ancestryNoCommits means the branch is an ancestor of main: it has
	// pushed no commits, its tip is a landing's, and its tip age says
	// nothing about the claim.
	ancestryNoCommits
)

// gitAncestry asks whether sha is an ancestor of mainSHA — whether the
// branch at sha has any commits of its own. It returns ancestryUnresolved,
// with no error, when the question cannot be answered here: mainSHA is
// empty, or git cannot resolve one of the two objects (an unfetched tip,
// which exits 128). That mirrors gitTip's posture on an unfetched SHA —
// report what is not known rather than guess, since guessing "has its own
// commits" restores exactly the borrowed-age EXPIRED this distinction
// exists to prevent.
func gitAncestry(sha, mainSHA string) (ancestry, error) {
	if mainSHA == "" {
		return ancestryUnresolved, nil
	}
	err := exec.Command("git", "merge-base", "--is-ancestor", sha, mainSHA).Run()
	if err == nil {
		return ancestryFromExit(0), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return ancestryFromExit(exitErr.ExitCode()), nil
	}
	return ancestryUnresolved, fmt.Errorf("running git merge-base --is-ancestor %s %s: %w", sha, mainSHA, err)
}

// ancestryFromExit maps `git merge-base --is-ancestor`'s exit status: 0 is
// "is an ancestor", 1 is "is not", and every other status (128 for an
// object this checkout does not have) is git declining to answer. It is
// pure — no git or process calls — so tests exercise each status directly.
func ancestryFromExit(code int) ancestry {
	switch code {
	case 0:
		return ancestryNoCommits
	case 1:
		return ancestryOwnCommits
	}
	return ancestryUnresolved
}

// ghLabel is one label object in `gh issue list --json labels`'s shape.
type ghLabel struct {
	Name string `json:"name"`
}

// ghIssue is one element of `gh issue list --state all --json
// number,state,labels`'s JSON array — the exact fields wipsurvey reads
// from it.
type ghIssue struct {
	Number int       `json:"number"`
	State  string    `json:"state"`
	Labels []ghLabel `json:"labels"`
}

// issueState is the subset of a GitHub issue's state classify needs:
// whether it is closed, and whether it carries the needs-replan label
// WORKFLOW.md uses to retire an abandoned attempt in place.
type issueState struct {
	number      int
	closed      bool
	needsReplan bool
}

// readIssues parses `gh issue list --state all --json
// number,state,labels`'s JSON array from r into a lookup by issue number.
// Empty input — no stdin redirected, or an explicit `< /dev/null` — is not
// an error: it means the caller has no issue data to offer, and readIssues
// returns a nil map so every branch is judged lease-only.
func readIssues(r io.Reader) (map[int]issueState, error) {
	dec := json.NewDecoder(r)
	var raw []ghIssue
	if err := dec.Decode(&raw); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("parsing issue JSON from stdin: %w", err)
	}

	issues := make(map[int]issueState, len(raw))
	for _, gi := range raw {
		state := issueState{
			number: gi.Number,
			closed: strings.EqualFold(gi.State, "CLOSED"),
		}
		for _, l := range gi.Labels {
			if l.Name == "needs-replan" {
				state.needsReplan = true
			}
		}
		issues[gi.Number] = state
	}
	return issues, nil
}

// verdict is one of the outcomes classify produces for a wip/issue-<N>
// branch.
type verdict string

const (
	live    verdict = "LIVE"
	claimed verdict = "CLAIMED"
	expired verdict = "EXPIRED"
	retired verdict = "RETIRED"
	unknown verdict = "UNKNOWN"
)

// classify decides one branch's verdict and a short reason from its
// remote tip time, its ancestry against main, and, optionally, the GitHub
// issue it implements. It is pure — no git or process calls — so tests
// exercise it directly without a repository or network access.
//
// tip is nil when the branch's SHA has never been fetched into this
// checkout's object store; classify then reports unknown rather than
// guessing an age, because a guessed age could read as expired
// (resumable) — the dangerous direction for a lease with no data behind
// it. issue is nil when no issue data was supplied (stdin was empty) or
// the branch's issue number was not present in it; the reason then notes
// the verdict is lease-only.
//
// The order of the four questions is itself the contract:
//
//   - Retirement first: a closed or needs-replan issue retires its branch
//     outright, with or without a fetched tip and with or without commits
//     of its own, per WORKFLOW.md's "abandoned attempts are retired in
//     place ... never resumed."
//   - Then an unfetched tip, which is unknown.
//   - Then anc == ancestryNoCommits, which is claimed: the tip belongs to
//     a landing, not to this branch, so no age question may be asked of
//     it at all.
//   - Only then the tip age, against the claim TTL. ancestryUnresolved
//     lands here too — the age is the best evidence left, and it is the
//     evidence this tool reported before it could ask about ancestry — but
//     its reason says the ancestry was undecided, so the verdict reads as
//     provisional rather than settled (#806).
func classify(branch string, tip *time.Time, anc ancestry, now time.Time, issue *issueState) (verdict, string) {
	if issue != nil && issue.closed {
		return retired, fmt.Sprintf("%s: issue #%d is closed", branch, issue.number)
	}
	if issue != nil && issue.needsReplan {
		return retired, fmt.Sprintf("%s: issue #%d is labelled needs-replan", branch, issue.number)
	}
	if tip == nil {
		return unknown, fmt.Sprintf("%s: tip not fetched -- run `git fetch origin`", branch)
	}

	leaseNote := ""
	if issue == nil {
		leaseNote = "; no issue data for this branch, lease-only"
	}
	if anc == ancestryNoCommits {
		return claimed, fmt.Sprintf("%s: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread%s", branch, leaseNote)
	}
	ancestryNote := ""
	if anc == ancestryUnresolved {
		ancestryNote = "; ancestry against main undecided, so this age may be main's rather than the claim's -- run `git fetch origin`"
	}
	age := now.Sub(*tip)
	if age <= claimTTL {
		return live, fmt.Sprintf("%s: tip pushed %s ago, within the %s claim TTL%s%s", branch, formatAge(age), formatAge(claimTTL), ancestryNote, leaseNote)
	}
	return expired, fmt.Sprintf("%s: tip pushed %s ago, past the %s claim TTL%s%s", branch, formatAge(age), formatAge(claimTTL), ancestryNote, leaseNote)
}

// formatAge renders a duration the way the report shows one: rounded to
// the minute, so two runs a few seconds apart render identically (STYLE
// D1) instead of drifting by the wall-clock second between them.
func formatAge(d time.Duration) string {
	return d.Round(time.Minute).String()
}

// row is one line of the survey's output table.
type row struct {
	issue   int
	branch  string
	tip     *time.Time
	verdict verdict
	reason  string
}

// sortRows orders rows by issue number, the report's deterministic order
// (STYLE D1); ties (which do not occur for well-formed input, since a
// branch name is unique per issue) break on branch name so the order
// never depends on git's or a map's iteration order.
func sortRows(rows []row) {
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].issue != rows[j].issue {
			return rows[i].issue < rows[j].issue
		}
		return rows[i].branch < rows[j].branch
	})
}

// renderTable writes rows as an aligned table: ISSUE, BRANCH, TIP AGE,
// VERDICT, REASON. It does not sort — callers order rows first (run calls
// sortRows) — so the same rows always render the same text (STYLE D1).
//
// A claimed row's TIP AGE reads "main's" rather than a duration: the
// number would be a real duration attached to the wrong branch, and a
// column scanned faster than it is read is the whole reason that age got
// acted on (#722).
func renderTable(w io.Writer, rows []row, now time.Time) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ISSUE\tBRANCH\tTIP AGE\tVERDICT\tREASON"); err != nil {
		return fmt.Errorf("writing table header: %w", err)
	}
	for _, r := range rows {
		age := "unknown"
		switch {
		case r.verdict == claimed:
			age = "main's"
		case r.tip != nil:
			age = formatAge(now.Sub(*r.tip))
		}
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\n", r.issue, r.branch, age, r.verdict, r.reason); err != nil {
			return fmt.Errorf("writing table row for issue #%d: %w", r.issue, err)
		}
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flushing report table: %w", err)
	}
	return nil
}
