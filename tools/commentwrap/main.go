// Command commentwrap reports `//` comment paragraphs whose line breaks no
// longer match the paragraph's own wrap width — the layout an edit leaves
// behind when the words it inserted or removed are not followed by a reflow.
//
// It is a gate step (CLAUDE.md's "Commands" block): a mechanical check for a
// mechanical defect. Judgment about what a comment SAYS stays with the arbiter
// and warden, per .golangci.yml's header; this tool only measures where the
// lines end. Two shapes are reported, the two an unreflowed edit leaves: a line
// stranded short of its paragraph's width with room to spare for the next
// line's first word, and a line pushed past that width on its own.
//
// Usage:
//
//	go tool commentwrap ./...        # whole module (the gate step)
//	go tool commentwrap ./value      # one package directory
//	go tool commentwrap -fix ./...   # reflow every reported paragraph in place
//
// It exits 0 when clean, 1 when it reports findings, and 2 on an operational
// error (unreadable or unparseable input).
//
// It exists because the defect it measures survived four sightings and a direct
// blocking review instruction naming the exact line (#329). That is the family
// #270 (advisory follow-ups with no tracker) and #315 (a body premise corrected
// only in its thread) belong to as well — a correction stated somewhere
// durable-looking but landing nowhere enforced. Their mechanisms differ from
// this one's and they are deliberately not merged with it.
package main

import (
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/kud360/goxsd8/tools/internal/commentpara"
)

const (
	// shortSlack is how many columns short of its paragraph's wrap width a
	// line must break before the break counts as ragged rather than as an
	// ordinary hand-wrap choice. Tuned against this repo's comments: at 20
	// every reported line is an orphan left mid-paragraph, while at 10 the
	// report is dominated by breaks a human placed on purpose (before a long
	// parenthetical, after a clause) that no later edit disturbed.
	shortSlack = 20
	// longSlack is the mirror bound: how far past every other line of its
	// paragraph one line may run before it counts as an edit that grew a line
	// instead of reflowing it.
	longSlack = 12
	// maxFillWidth is the widest any paragraph in this repo is filled to. An
	// over-long line has to clear it as well as longSlack, because standing
	// clear of the paragraph is on its own a weak signal in a three-line
	// paragraph whose other two lines end on a clause; clearing a width no
	// paragraph here is wrapped to is what says the line was never wrapped at
	// all. Measured, not chosen: comment lines cluster up to 88 columns and
	// thin out immediately after, and every line in the repo past it is an
	// unreflowed insertion (#329's tabled mirror-defect site is 93).
	maxFillWidth = 88
	// minWrapWidth is the narrowest paragraph the check will judge. A paragraph
	// whose longest line stays under this was never filled to a wrap width —
	// a quoted lexical form, a short spec extract, a two-word aside — so there
	// is no width for its breaks to be ragged against.
	minWrapWidth = 60
)

func main() {
	fix := flag.Bool("fix", false, "reflow every reported paragraph in place instead of reporting it")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"./..."}
	}

	clean, err := run(args, *fix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "commentwrap: %v\n", err)
		os.Exit(2)
	}
	if !clean {
		os.Exit(1)
	}
}

// run checks (or, with fix, reflows) every file named by args, reporting what
// it found. It returns false when it reported anything, which is the exit
// status main turns it into.
func run(args []string, fix bool) (bool, error) {
	files, err := commentpara.Files(args)
	if err != nil {
		return false, err
	}

	used := make(map[int]bool, len(waivers))
	var findings []finding
	for _, path := range files {
		got, err := process(path, fix, used)
		if err != nil {
			return false, err
		}
		findings = append(findings, got...)
	}
	findings = append(findings, staleWaivers(files, used)...)

	if len(findings) == 0 {
		return true, nil
	}
	// Findings go to stderr, where this repo's tools report (and where the
	// lint gate's errcheck rules take a diagnostic write as unchecked by
	// design).
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "%s:%d: %s\n", f.file, f.line, f.msg)
	}
	fmt.Fprintf(os.Stderr, "commentwrap: %d ragged comment line(s); reflow each reported paragraph at its own wrap width (go tool commentwrap -fix)\n", len(findings))
	return false, nil
}

// finding is one ragged line: a break a reflow of its paragraph, at that
// paragraph's own wrap width, would not have placed there.
type finding struct {
	file string
	line int
	msg  string
}

// process reports the ragged lines of one file, or reflows them when fix is
// set. Waivers matched along the way are recorded in used so a waiver that has
// stopped matching can be reported as stale.
func process(path string, fix bool, used map[int]bool) ([]finding, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	lines := strings.Split(string(src), "\n")
	paras, err := comments(path, string(src), lines)
	if err != nil {
		return nil, err
	}

	var findings []finding
	var splices []splice
	for _, para := range paras {
		width, flaws := flaws(para)
		if len(flaws) == 0 {
			continue
		}
		if w, waived := waivedBy(path, para); waived {
			used[w] = true
			continue
		}
		if !fix {
			findings = append(findings, report(path, para, width, flaws)...)
			continue
		}
		from := flaws[0].line
		splices = append(splices, splice{
			start: para[from].Num,
			end:   para[len(para)-1].Num,
			lines: reflow(para, width, from),
		})
	}

	if !fix || len(splices) == 0 {
		return findings, nil
	}
	if err := rewrite(path, lines, splices); err != nil {
		return nil, err
	}
	return findings, nil
}

// comments parses src for its comments and returns the paragraphs a reflow
// acts on, in source order: commentpara's, less the one-line ones, since a
// single line has no wrap to be ragged.
func comments(path, src string, lines []string) ([][]commentpara.Line, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if commentpara.IsGenerated(lines) {
		// A generated file's layout belongs to its generator, not to whoever
		// last edited the tree.
		return nil, nil
	}

	var paras [][]commentpara.Line
	for _, group := range file.Comments {
		for _, para := range commentpara.Paragraphs(fset, group, lines) {
			if len(para) > 1 {
				paras = append(paras, para)
			}
		}
	}
	return paras, nil
}

// columns is the line's total width in columns, the quantity a wrap width bounds.
func columns(l commentpara.Line) int {
	return utf8.RuneCountInString(l.Raw)
}

// lead is everything a reflowed line repeats verbatim: the line's
// indentation, its `//`, and the body indentation that follows.
func lead(l commentpara.Line) string {
	marker := strings.Index(l.Raw, "//") + len("//")
	body := l.Raw[marker:]
	return l.Raw[:marker] + body[:len(body)-len(strings.TrimLeft(body, " \t"))]
}

// flaw is one line of a paragraph whose break a reflow would move: line is its
// index within the paragraph, over says which of the two shapes it is, and off
// is how many columns it misses the wrap width by.
type flaw struct {
	line int
	over bool
	off  int
}

// flaws returns the paragraph's wrap width and the lines that miss it. A
// paragraph whose opening line is verbatim-indented, or that was never filled
// to a wrap width at all, has none by construction.
func flaws(para []commentpara.Line) (int, []flaw) {
	if para[0].IsVerbatim() {
		return 0, nil
	}
	width := wrapWidth(para)
	if width < minWrapWidth {
		return 0, nil
	}

	var out []flaw
	for i, line := range para {
		if columns(line) > width+longSlack && len(strings.Fields(line.Body)) > 1 {
			out = append(out, flaw{line: i, over: true, off: columns(line) - width})
			continue
		}
		if i == len(para)-1 || width-columns(line) < shortSlack {
			continue
		}
		word := firstWord(para[i+1].Body)
		if word == 0 || columns(line)+1+word > width {
			continue
		}
		out = append(out, flaw{line: i, off: width - columns(line)})
	}
	return width, out
}

// wrapWidth is the width the paragraph was filled to: its longest line, except
// where that line is itself the over-long defect — standing clear of every
// other line by more than longSlack AND past maxFillWidth. Such a line does not
// get to define the width it broke; the runner-up does. Standing clear takes at
// least two other lines to stand clear of, so a two-line paragraph is always
// its own width.
func wrapWidth(para []commentpara.Line) int {
	widths := make([]int, len(para))
	for i, line := range para {
		widths[i] = columns(line)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(widths)))

	if len(widths) > 2 && widths[0]-widths[1] > longSlack && widths[0] > maxFillWidth {
		return widths[1]
	}
	return widths[0]
}

// firstWord is the width in columns of the first whitespace-delimited word of
// a comment body — the word a reflow would pull up onto the line above.
func firstWord(body string) int {
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return 0
	}
	return utf8.RuneCountInString(fields[0])
}

// report renders one paragraph's flaws as findings.
func report(path string, para []commentpara.Line, width int, flaws []flaw) []finding {
	findings := make([]finding, 0, len(flaws))
	for _, f := range flaws {
		msg := fmt.Sprintf("comment line ends %d columns short of the paragraph's %d-column wrap width, and the next line's first word would fit", f.off, width)
		if f.over {
			msg = fmt.Sprintf("comment line runs %d columns past the paragraph's %d-column wrap width", f.off, width)
		}
		findings = append(findings, finding{file: path, line: para[f.line].Num, msg: msg})
	}
	return findings
}

// reflow re-fills the paragraph from its first ragged line onward, greedily, at
// the paragraph's own wrap width. Everything above that line is left byte-identical:
// the break the edit disturbed is the first one that has to move, and a reflow
// that reaches further would rewrite lines no edit touched.
func reflow(para []commentpara.Line, width, from int) []string {
	first := lead(para[from])
	cont := first
	if from+1 < len(para) {
		cont = lead(para[from+1])
	}

	var words []string
	for _, line := range para[from:] {
		words = append(words, strings.Fields(line.Body)...)
	}

	var out []string
	cur := first
	empty := true
	for _, w := range words {
		if !empty && utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) > width {
			out = append(out, cur)
			cur, empty = cont, true
		}
		if !empty {
			cur += " "
		}
		cur += w
		empty = false
	}
	if !empty {
		out = append(out, cur)
	}
	return out
}

// splice is one reflowed run: source lines [start, end] (1-based, inclusive)
// become lines.
type splice struct {
	start int
	end   int
	lines []string
}

// rewrite applies the splices to the file's lines and writes it back. Splices
// are disjoint and rise through the file, so they are applied back to front and
// no line number shifts under one still to be applied.
func rewrite(path string, lines []string, splices []splice) error {
	for i := len(splices) - 1; i >= 0; i-- {
		s := splices[i]
		out := make([]string, 0, len(lines))
		out = append(out, lines[:s.start-1]...)
		out = append(out, s.lines...)
		out = append(out, lines[s.end:]...)
		lines = out
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// waiver suppresses one paragraph whose reflow belongs to another open issue,
// so this check does not land a fix that issue's branch is already carrying. A
// waiver names the line by its exact text rather than by number (line numbers
// drift under every landing) and names the issue that retires it. It is the
// same discipline STYLE P3 puts on `GAP(` markers: a waiver that no longer
// matches a ragged line is reported as stale and must be deleted, so it cannot
// outlive the issue that justified it.
type waiver struct {
	file  string
	text  string
	issue string
}

var waivers = []waiver{{
	file:  "value/list.go",
	text:  "// terminates structurally: neither the item type nor any",
	issue: "#564",
}}

// covers reports whether path is the file the waiver names. The waiver names a
// repo-relative path and the check runs on paths relative to wherever it was
// invoked, so the match is on whole path elements: a bare suffix match would
// let a waiver keyed "value/list.go" cover "xvalue/list.go" too.
func (w waiver) covers(path string) bool {
	slash := filepath.ToSlash(filepath.Clean(path))
	return slash == w.file || strings.HasSuffix(slash, "/"+w.file)
}

// waivedBy reports which waiver, if any, covers a paragraph. The whole
// paragraph is waived, not the single line: a reflow rewrites the paragraph
// from its first ragged break onward, so there is no way to fix a neighbouring
// break without touching the waived one.
func waivedBy(path string, para []commentpara.Line) (int, bool) {
	for i, w := range waivers {
		if !w.covers(path) {
			continue
		}
		for _, line := range para {
			if strings.TrimSpace(line.Raw) == w.text {
				return i, true
			}
		}
	}
	return 0, false
}

// staleWaivers reports every waiver whose file was checked but whose line is no
// longer a ragged one — the issue that owned it has landed, and the waiver has
// to go with it.
func staleWaivers(files []string, used map[int]bool) []finding {
	var findings []finding
	for i, w := range waivers {
		if used[i] {
			continue
		}
		for _, path := range files {
			if !w.covers(path) {
				continue
			}
			findings = append(findings, finding{
				file: path,
				line: 1,
				msg: fmt.Sprintf("stale waiver: %q is no longer a ragged line, so the waiver deferring it to %s must be deleted from tools/commentwrap",
					w.text, w.issue),
			})
			break
		}
	}
	return findings
}
