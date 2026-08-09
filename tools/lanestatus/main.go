// Command lanestatus reports the committed conformance score per lane as a
// Markdown table, for docs/PLAN.md's Status section.
//
// The expectation files are the only source of truth for a lane score
// (conformance/testdata/expectations/README.md). Counting them is
// deterministic and mechanical, so it is a tool rather than a grep
// (PRINCIPLES 27): a hand-count is indistinguishable from a fresh one once
// it is stamped into docs/PLAN.md, and stays wrong until someone re-reads
// 15,000 lines.
//
// It reads only the committed files through conformance.LoadExpectations, so
// it needs neither the W3C suite submodule nor a conformance run, and the
// file format keeps one reader in the module. Both packages are repo
// infrastructure rather than library (docs/ARCHITECTURE.md's two tiers), so
// this import crosses no published boundary.
//
// Usage:
//
//	go tool lanestatus                  # the default expectations directory
//	go tool lanestatus -dir <path>      # another directory
//
// It exits 0 on success and 2 on an operational error (unreadable or
// malformed input). Lanes are reported in filename order, which is stable
// across runs and independent of the runner's execution order.
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kud360/goxsd8/conformance"
)

// defaultDir is the committed expectations directory, relative to the module
// root — the same path conformance/runner.go resolves against its own package.
const defaultDir = "conformance/testdata/expectations"

// laneScore is one lane's committed score. Total is not stored: it is Pass +
// Fail, and storing it would be a second encoding of the same fact (STYLE D3).
type laneScore struct {
	name string
	pass int
	fail int
}

func (l laneScore) total() int { return l.pass + l.fail }

func main() {
	dir := flag.String("dir", defaultDir, "directory holding the per-lane expectation files")
	flag.Parse()

	scores, err := readLanes(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lanestatus: %v\n", err)
		os.Exit(2)
	}
	fmt.Print(renderMarkdown(scores))
}

// readLanes reads every `<lane>.txt` in dir and returns one score per lane,
// ordered by lane name so the output is stable (STYLE D1).
func readLanes(dir string) ([]laneScore, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading expectations directory %s: %w", dir, err)
	}

	var scores []laneScore
	for _, e := range entries {
		name, ok := laneNameOf(e)
		if !ok {
			continue
		}
		cases, err := conformance.LoadExpectations(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("lane %s: %w", name, err)
		}
		scores = append(scores, scoreOf(name, cases))
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].name < scores[j].name })
	return scores, nil
}

// laneNameOf reports the lane a directory entry carries, and whether it is a
// lane file at all. Only regular `.txt` files are lanes; README.md and any
// subdirectory are not.
func laneNameOf(e fs.DirEntry) (string, bool) {
	if e.IsDir() {
		return "", false
	}
	name := e.Name()
	if filepath.Ext(name) != ".txt" {
		return "", false
	}
	return strings.TrimSuffix(name, ".txt"), true
}

// scoreOf tallies one lane's loaded cases. Iteration order does not reach the
// output — only the two counts do — so ranging the map is safe here (STYLE D2).
func scoreOf(name string, cases map[string]conformance.Status) laneScore {
	score := laneScore{name: name}
	for _, st := range cases {
		if st.IsPass() {
			score.pass++
			continue
		}
		score.fail++
	}
	return score
}

// renderMarkdown renders the score table PLAN.md's Status section carries. A
// lane with no cases prints em dashes for pass and fail: it has no score yet,
// which is a different statement from scoring zero.
func renderMarkdown(scores []laneScore) string {
	var b strings.Builder
	b.WriteString("| Lane | Pass | Fail | Total |\n")
	b.WriteString("|---|---:|---:|---:|\n")
	for _, s := range scores {
		if s.total() == 0 {
			fmt.Fprintf(&b, "| `%s` | — | — | 0 |\n", s.name)
			continue
		}
		fmt.Fprintf(&b, "| `%s` | %d | %d | %d |\n", s.name, s.pass, s.fail, s.total())
	}
	return b.String()
}
