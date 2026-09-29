package declinecensus

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Census is one lane's GOXSD_DECLINES=1 listing: the three lists that
// partition that run's failures.
type Census struct {
	Declined      []string
	Indeterminate []string
	// Decided maps each decided disagreement to what the run charged it — the
	// text after `=`, empty on a lane that charges nothing.
	Decided map[string]string
}

// The three census lists, as reportDeclines in conformance_test.go labels
// them.
const (
	listDeclined      = "decline candidates"
	listIndeterminate = "indeterminate declines"
	listDecided       = "decided disagreements"
)

// censusLine matches one census listing line wherever the test framework's
// `file:line:` prefix puts it: the lane, the list's label, and the bracketed,
// space-separated IDs %v renders.
var censusLine = regexp.MustCompile(`lane (\S+): (` + listDeclined + `|` + listIndeterminate + `|` + listDecided + `): \[(.*)\]\s*$`)

// Read reads lane's three census lists from the run log at path. Each must
// appear exactly once: a missing one is a log from a run without
// GOXSD_DECLINES=1, or from before the listing carried all three, and a
// repeated one is two runs concatenated. Either is an error, as is a log that
// cannot be read.
func Read(path, lane string) (Census, error) {
	f, err := os.Open(path)
	if err != nil {
		return Census{}, fmt.Errorf("reading run log: %w", err)
	}
	defer func() { _ = f.Close() }() // read-only handle: a close error changes nothing read
	lists := map[string][]string{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 64*1024*1024)
	for sc.Scan() {
		m := censusLine.FindStringSubmatch(sc.Text())
		if m == nil || m[1] != lane {
			continue
		}
		if _, dup := lists[m[2]]; dup {
			return Census{}, fmt.Errorf("run log %s: lane %s lists %q twice — one run per log", path, lane, m[2])
		}
		lists[m[2]] = strings.Fields(m[3])
	}
	if err := sc.Err(); err != nil {
		return Census{}, fmt.Errorf("reading run log %s: %w", path, err)
	}
	for _, label := range []string{listDeclined, listIndeterminate, listDecided} {
		if _, ok := lists[label]; !ok {
			return Census{}, fmt.Errorf("run log %s carries no %q listing for lane %s — run the suite with GOXSD_DECLINES=1 and -v", path, label, lane)
		}
	}
	decided := map[string]string{}
	for _, entry := range lists[listDecided] {
		id, charge, _ := strings.Cut(entry, "=")
		decided[id] = charge
	}
	return Census{Declined: lists[listDeclined], Indeterminate: lists[listIndeterminate], Decided: decided}, nil
}
