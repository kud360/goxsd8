// Command issuefeed reads the repository's issues over REST in one of two
// modes. Its feed writes the issue lists `go tool wipsurvey`, `go tool
// gapaudit` and `go tool lanepartition` read on stdin: `issues.json`, rows of
// `{number, state, labels: [{name}]}`, and `gapissues.json`, the same rows
// plus `title` and `body`. Its -owner-comments survey prints the comments the
// repository owner wrote outside the Claude app on `blocked` issues. Each
// mode walks REST listings page by page through `gh api`, as one plain
// command the worktree-isolation guard admits, where a shell loop is refused
// (#2235).
//
// Every listing — the issue listing, and the survey's per-issue events and
// comments — is walked under the rules this paragraph and the next two
// state. The walk requests every page by an explicit `page=N` and never
// through `gh api --paginate`, whose Link header carries numeric-ID repository paths the
// session proxy answers with a 403 after writing the pages it did fetch. A
// failed fetch — `gh` exiting non-zero, or stdout that is not a JSON array,
// as the proxy's error body is not — aborts the run and is never retried.
//
// A page shorter than per_page is either the end of the list or a transient
// truncation of a full page, so every such page, empty included, is
// re-requested once after a pause and the answer with more rows is kept
// (#1145). The walk ends at the first page still empty after its re-request.
// Two checks then guard against what a re-request cannot catch: every page
// but the last non-empty one must hold exactly per_page rows, and no page may
// repeat a row an earlier page — or itself — already carried, which is how a
// page served from another page's offset shows (#1520). An issue row is keyed
// by its number, an event or comment row by its id.
//
// The feed's distinct-number stamp printed to stderr — `rows N distinct D min A max
// B no gaps`, or `... missing numbers K` — is computed over every row the
// walk kept, pull requests included, because issues and pull requests share
// one number space; it is informational and a gap does not abort.
//
// Usage:
//
//	go tool issuefeed -dir /abs/dir/outside/the/checkout
//	go tool issuefeed -owner-comments
//
// The feed's -dir is required and created if absent; the two files are written there
// and nowhere else, so a directory outside the checkout keeps them out of a
// `git add -A` (#1217). Both files are written or neither is: the run first
// removes any earlier copies, builds both in memory, and renames them into
// place only once every check has passed.
//
// The feed's exit status is 0 when both files are written, 2 for a usage
// error, and 1 for a failed fetch, a failed check or a failed write — and
// after a non-zero exit -dir holds neither file. A usage error clears a -dir
// the parse read before it, unless the parse also read -owner-comments,
// whose survey writes no -dir and so clears none; an unknown flag ahead of
// -dir stops the parse before -dir is read, so that command line names no
// directory and nothing is cleared.
//
// The -owner-comments survey takes no -dir and writes no file. It lists the
// open issues labelled `blocked`, pull requests dropped, and for each walks
// its events and its comments. A comment is the owner's own when its REST
// `performed_via_github_app` is null: every agent comment posts under the
// owner's login through the Claude app, which that field names, a session
// container's `gh api` writes included. The survey prints each owner comment
// whose `created_at` is strictly later than the issue's latest `labeled`
// event for `blocked`, one `#N <comment id> <created_at>` line each, issues
// in ascending number and each issue's comments in REST order. When none
// qualifies it prints one line saying so, which does not start with `#`. An
// open `blocked` issue whose events carry no `labeled` event for `blocked`,
// or a comment carrying no `performed_via_github_app` field, aborts the
// survey, since neither can be dated or attributed. Nothing is printed to
// stdout until every fetch has succeeded; exit status is 0 when the lines
// are printed, 2 for a usage error, and 1 for a failed fetch, a failed check
// or either abort, with stdout empty.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// issuesPath is the repository-scoped REST issues path every endpoint the
// tool walks starts with. The slug is fixed: this repository's surveys are
// the tool's only consumer.
const issuesPath = "repos/kud360/goxsd8/issues"

// blockedLabel is the label whose latest `labeled` event dates an issue's
// block for the -owner-comments survey.
const blockedLabel = "blocked"

// listing is the issue listing the feed walks; blockedListing is the one the
// -owner-comments survey walks.
const (
	listing        = issuesPath + "?state=all"
	blockedListing = issuesPath + "?state=open&labels=" + blockedLabel
)

// perPage is the page size every request asks for, REST's maximum.
const perPage = 100

// retryPause is how long the real walk waits before re-requesting a short
// page.
const retryPause = 2 * time.Second

// The two output files, named as docs/ROUTINES.md's "Survey input" and the
// survey tools' own docs name them.
const (
	issuesFile    = "issues.json"
	gapIssuesFile = "gapissues.json"
)

const usage = "usage: issuefeed -dir <dir>   (dir outside the checkout; created if absent)\n       issuefeed -owner-comments"

// errUsage is what every command-line error wraps, so main exits 2 for it and
// 1 for everything else.
var errUsage = errors.New("usage error")

// fetchFunc returns the raw stdout of one request for the 1-based page of
// endpoint, a path under issuesPath with any query but the paging. ghFetch is
// the real one; tests substitute a fake page source.
type fetchFunc func(endpoint string, page int) ([]byte, error)

func main() {
	err := run(os.Stdout, os.Stderr, os.Args[1:], ghFetch, func() { time.Sleep(retryPause) })
	if err == nil {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "issuefeed: %v\n", err)
	if errors.Is(err, errUsage) {
		os.Exit(2)
	}
	os.Exit(1)
}

// run parses the command line and runs the mode it names. Fetches go through
// fetch, and pause runs before each re-request.
func run(stdout, stderr io.Writer, args []string, fetch fetchFunc, pause func()) error {
	flags := flag.NewFlagSet("issuefeed", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	dir := flags.String("dir", "", "directory to write issues.json and gapissues.json into (required)")
	owner := flags.Bool("owner-comments", false, "print the owner's comments on blocked issues since each was blocked")
	if err := flags.Parse(args); err != nil {
		return usageError(*dir, *owner, fmt.Errorf("%w\n%s", err, usage))
	}
	if flags.NArg() != 0 {
		return usageError(*dir, *owner, fmt.Errorf("unexpected argument %q\n%s", flags.Arg(0), usage))
	}
	if *owner && *dir != "" {
		return fmt.Errorf("%w: -owner-comments writes no file and takes no -dir\n%s", errUsage, usage)
	}
	if *owner {
		return ownerComments(stdout, fetch, pause)
	}
	return writeFeed(stdout, stderr, *dir, fetch, pause)
}

// writeFeed drives one feed end to end: clear the earlier outputs from dir,
// walk the listing, print the stamp to stderr, and write both files, naming
// each on stdout.
func writeFeed(stdout, stderr io.Writer, dir string, fetch fetchFunc, pause func()) error {
	if dir == "" {
		return fmt.Errorf("%w: -dir is required, so the files never land in the working directory\n%s", errUsage, usage)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating -dir: %w", err)
	}
	if err := removeOutputs(dir); err != nil {
		return err
	}
	all, err := fetchAll(fetch, listing, pause, issueKey)
	if err != nil {
		return err
	}
	if len(all) == 0 {
		return errors.New("the walk found no rows at all; re-run it")
	}
	_, _ = fmt.Fprintln(stderr, stamp(all))
	issues, gapIssues, err := reshape(all)
	if err != nil {
		return err
	}
	if err := writeOutputs(dir, issues, gapIssues); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(stdout, filepath.Join(dir, issuesFile))
	_, _ = fmt.Fprintln(stdout, filepath.Join(dir, gapIssuesFile))
	return nil
}

// usageError wraps err in errUsage after clearing the earlier outputs from
// dir, so a usage error that still named a -dir leaves neither file there. It
// clears nothing when dir is empty — an unknown flag before -dir stops the
// parse before -dir is read, and no directory was named at all — nor when
// owner is set, since the -owner-comments survey writes no -dir and must not
// clear a feed another run wrote. A failed removal is reported beside the
// usage error, which still exits 2.
func usageError(dir string, owner bool, err error) error {
	err = fmt.Errorf("%w: %w", errUsage, err)
	if dir == "" || owner {
		return err
	}
	return errors.Join(err, removeOutputs(dir))
}

// ghArgs is the `gh` argv requesting one page of endpoint: an explicit page
// number, never --paginate.
func ghArgs(endpoint string, page int) []string {
	sep := "?"
	if strings.Contains(endpoint, "?") {
		sep = "&"
	}
	return []string{"api", fmt.Sprintf("%s%sper_page=%d&page=%d", endpoint, sep, perPage, page)}
}

// ghFetch execs `gh api` for one page of endpoint and returns its stdout.
func ghFetch(endpoint string, page int) ([]byte, error) {
	args := ghArgs(endpoint, page)
	out, err := exec.Command("gh", args...).Output()
	if err == nil {
		return out, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("gh %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
}

// row is one listing entry as the walk reads it. State passes through
// unchanged, null or absent alike written back as null; a null Title or Body
// decodes as "", as every consumer's string field reads it. PullRequest is
// non-nil exactly when the entry carries a non-null `pull_request`.
type row struct {
	Number      int              `json:"number"`
	Title       string           `json:"title"`
	State       json.RawMessage  `json:"state"`
	Body        string           `json:"body"`
	Labels      []label          `json:"labels"`
	PullRequest *json.RawMessage `json:"pull_request"`
}

type label struct {
	Name string `json:"name"`
}

// issueKey keys an issue row for the repeated-row check by its number.
func issueKey(r row) string {
	return fmt.Sprintf("#%d", r.Number)
}

// fetchPage requests one page of endpoint and decodes it. A fetch error, or
// stdout that is not a JSON array of entries, is a failed fetch.
func fetchPage[T any](fetch fetchFunc, endpoint string, page int) ([]T, error) {
	out, err := fetch(endpoint, page)
	if err != nil {
		return nil, fmt.Errorf("fetching page %d: %w", page, err)
	}
	trimmed := bytes.TrimSpace(out)
	if !bytes.HasPrefix(trimmed, []byte("[")) {
		return nil, fmt.Errorf("fetching page %d: the answer is not a JSON array: %.200s", page, trimmed)
	}
	var rows []T
	if err := json.Unmarshal(trimmed, &rows); err != nil {
		return nil, fmt.Errorf("fetching page %d: decoding the answer: %w", page, err)
	}
	return rows, nil
}

// fetchAll walks endpoint and checks the walk, returning every row it kept in
// page order.
func fetchAll[T any](fetch fetchFunc, endpoint string, pause func(), key func(T) string) ([]T, error) {
	pages, err := walk(fetch, endpoint, pause, key)
	if err != nil {
		return nil, err
	}
	if err := checkFullness(pages); err != nil {
		return nil, err
	}
	return slices.Concat(pages...), nil
}

// walk returns endpoint's non-empty pages in page order. A page shorter than
// perPage, empty included, is re-requested once after pause and the longer
// answer kept; the walk ends at the first page still empty after that. A
// page repeating a row key already seen — on an earlier page or on itself —
// aborts the walk (#1520), which also bounds it against a source serving the
// same page forever.
func walk[T any](fetch fetchFunc, endpoint string, pause func(), key func(T) string) ([][]T, error) {
	var pages [][]T
	seen := map[string]int{}
	for page := 1; ; page++ {
		rows, err := fetchPage[T](fetch, endpoint, page)
		if err != nil {
			return nil, err
		}
		if len(rows) < perPage {
			pause()
			again, err := fetchPage[T](fetch, endpoint, page)
			if err != nil {
				return nil, err
			}
			if len(again) > len(rows) {
				rows = again
			}
		}
		if len(rows) == 0 {
			return pages, nil
		}
		for _, r := range rows {
			k := key(r)
			if first, ok := seen[k]; ok {
				return nil, fmt.Errorf("page %d repeats %s from page %d; re-run the walk", page, k, first)
			}
			seen[k] = page
		}
		pages = append(pages, rows)
	}
}

// checkFullness reports the first page other than the last that does not
// hold exactly perPage rows: a short page the walk passed is a transient
// truncation its re-request did not recover.
func checkFullness[T any](pages [][]T) error {
	for i, rows := range pages[:max(len(pages)-1, 0)] {
		if len(rows) != perPage {
			return fmt.Errorf("page %d came back with %d rows and is not the last page; re-run the walk", i+1, len(rows))
		}
	}
	return nil
}

// stamp is the distinct-number line over rows, which must be non-empty.
func stamp(rows []row) string {
	nums := make([]int, 0, len(rows))
	for _, r := range rows {
		nums = append(nums, r.Number)
	}
	slices.Sort(nums)
	distinct := slices.Compact(nums)
	lo, hi := distinct[0], distinct[len(distinct)-1]
	line := fmt.Sprintf("rows %d distinct %d min %d max %d", len(rows), len(distinct), lo, hi)
	if missing := hi - lo + 1 - len(distinct); missing != 0 {
		return fmt.Sprintf("%s missing numbers %d", line, missing)
	}
	return line + " no gaps"
}

// issueOut and gapIssueOut are the two output row shapes, fields in the order
// docs/ROUTINES.md has always written them.
type issueOut struct {
	Number int             `json:"number"`
	State  json.RawMessage `json:"state"`
	Labels []label         `json:"labels"`
}

type gapIssueOut struct {
	Number int             `json:"number"`
	Title  string          `json:"title"`
	State  json.RawMessage `json:"state"`
	Body   string          `json:"body"`
	Labels []label         `json:"labels"`
}

// reshape drops the pull requests, which share the issues endpoint, and
// renders the remaining rows, in walk order, as the two files' contents.
func reshape(rows []row) (issues, gapIssues []byte, err error) {
	is := []issueOut{}
	gs := []gapIssueOut{}
	for _, r := range rows {
		if r.PullRequest != nil {
			continue
		}
		labels := make([]label, 0, len(r.Labels))
		labels = append(labels, r.Labels...)
		is = append(is, issueOut{Number: r.Number, State: r.State, Labels: labels})
		gs = append(gs, gapIssueOut{Number: r.Number, Title: r.Title, State: r.State, Body: r.Body, Labels: labels})
	}
	issues, err = encode(is)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding %s: %w", issuesFile, err)
	}
	gapIssues, err = encode(gs)
	if err != nil {
		return nil, nil, fmt.Errorf("encoding %s: %w", gapIssuesFile, err)
	}
	return issues, gapIssues, nil
}

// encode renders v as jq prints it: two-space indent, a trailing newline, and
// `<`, `>` and `&` left as written rather than escaped, so a grep of a file
// finds the text the issue carries.
func encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// removeOutputs deletes both output files from dir if present, so a failed
// run never leaves an earlier run's lists to be read as current.
func removeOutputs(dir string) error {
	for _, name := range []string{issuesFile, gapIssuesFile} {
		err := os.Remove(filepath.Join(dir, name))
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("removing the earlier %s: %w", name, err)
		}
	}
	return nil
}

// writeOutputs writes both files into dir, both or neither: each goes to a
// temporary file first, and a failure at any step removes whatever this call
// already placed.
func writeOutputs(dir string, issues, gapIssues []byte) error {
	type output struct {
		name string
		data []byte
		tmp  string
	}
	outs := []*output{{name: issuesFile, data: issues}, {name: gapIssuesFile, data: gapIssues}}
	var placed []string
	cleanup := func() {
		// Best effort: the write already failed, and that error is the one
		// reported; a file this cannot remove is named by no success line.
		for _, o := range outs {
			if o.tmp != "" {
				_ = os.Remove(o.tmp)
			}
		}
		for _, p := range placed {
			_ = os.Remove(p)
		}
	}
	for _, o := range outs {
		tmp, err := writeTemp(dir, o.name, o.data)
		if err != nil {
			cleanup()
			return err
		}
		o.tmp = tmp
	}
	for _, o := range outs {
		dst := filepath.Join(dir, o.name)
		if err := os.Rename(o.tmp, dst); err != nil {
			cleanup()
			return fmt.Errorf("placing %s: %w", o.name, err)
		}
		o.tmp = ""
		placed = append(placed, dst)
	}
	return nil
}

// writeTemp writes data to a new temporary file in dir and returns its path.
func writeTemp(dir, name string, data []byte) (string, error) {
	f, err := os.CreateTemp(dir, name+".tmp*")
	if err != nil {
		return "", fmt.Errorf("writing %s: %w", name, err)
	}
	_, werr := f.Write(data)
	cerr := f.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(f.Name()) // best effort: the write error is the one reported
		return "", fmt.Errorf("writing %s: %w", name, err)
	}
	return f.Name(), nil
}

// event is one issue-events entry as the survey reads it. Label is nil for
// an event that carries none.
type event struct {
	ID        int64     `json:"id"`
	Event     string    `json:"event"`
	Label     *label    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// comment is one issue-comments entry as the survey reads it. App holds the
// raw `performed_via_github_app` value, nil when the field is absent.
type comment struct {
	ID        int64           `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	App       json.RawMessage `json:"performed_via_github_app"`
}

func eventKey(e event) string     { return fmt.Sprintf("id %d", e.ID) }
func commentKey(c comment) string { return fmt.Sprintf("id %d", c.ID) }

// ownerComments runs the -owner-comments survey the package doc states,
// buffering every line until the last fetch has succeeded.
func ownerComments(stdout io.Writer, fetch fetchFunc, pause func()) error {
	listed, err := fetchAll(fetch, blockedListing, pause, issueKey)
	if err != nil {
		return fmt.Errorf("listing the open %s issues: %w", blockedLabel, err)
	}
	var nums []int
	for _, r := range listed {
		if r.PullRequest != nil {
			continue
		}
		nums = append(nums, r.Number)
	}
	slices.Sort(nums)
	var lines []string
	for _, n := range nums {
		found, err := ownerCommentsOn(fetch, pause, n)
		if err != nil {
			return err
		}
		lines = append(lines, found...)
	}
	if len(lines) == 0 {
		_, _ = fmt.Fprintf(stdout, "no owner comment since the %s label on any of the %d open %s issues\n", blockedLabel, len(nums), blockedLabel)
		return nil
	}
	for _, l := range lines {
		_, _ = fmt.Fprintln(stdout, l)
	}
	return nil
}

// ownerCommentsOn returns issue n's `#N <comment id> <created_at>` lines, in
// REST order, for each comment with a null `performed_via_github_app`
// created strictly after n's latest `labeled` event for blockedLabel.
func ownerCommentsOn(fetch fetchFunc, pause func(), n int) ([]string, error) {
	events, err := fetchAll(fetch, fmt.Sprintf("%s/%d/events", issuesPath, n), pause, eventKey)
	if err != nil {
		return nil, fmt.Errorf("issue #%d events: %w", n, err)
	}
	var since time.Time
	for _, e := range events {
		if e.Event != "labeled" || e.Label == nil || e.Label.Name != blockedLabel {
			continue
		}
		if e.CreatedAt.After(since) {
			since = e.CreatedAt
		}
	}
	if since.IsZero() {
		return nil, fmt.Errorf("issue #%d events: no labeled event for %s dates the block", n, blockedLabel)
	}
	comments, err := fetchAll(fetch, fmt.Sprintf("%s/%d/comments", issuesPath, n), pause, commentKey)
	if err != nil {
		return nil, fmt.Errorf("issue #%d comments: %w", n, err)
	}
	var lines []string
	for _, c := range comments {
		if c.App == nil {
			return nil, fmt.Errorf("issue #%d comments: comment id %d carries no performed_via_github_app field", n, c.ID)
		}
		if string(c.App) != "null" || !c.CreatedAt.After(since) {
			continue
		}
		lines = append(lines, fmt.Sprintf("#%d %d %s", n, c.ID, c.CreatedAt.Format(time.RFC3339)))
	}
	return lines, nil
}
