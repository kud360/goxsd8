package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/tools/internal/commentpara"
)

// mkLine builds a comment line of exactly width columns out of four-letter
// words, so a test can place a break at a chosen column and know that the
// following line's first word is four columns wide.
func mkLine(indent, width int) string {
	line := strings.Repeat("\t", indent) + "//"
	for len(line)+len(" abcd") <= width {
		line += " abcd"
	}
	for len(line) < width {
		line += "x"
	}
	return line
}

// analyze writes src to a temporary Go file and runs the check over it,
// returning one string per finding: "<line>: short" or "<line>: long".
func analyze(t *testing.T, src string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	findings, err := process(path, false, map[int]bool{})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	got := make([]string, 0, len(findings))
	for _, f := range findings {
		kind := "short"
		if strings.Contains(f.msg, "past") {
			kind = "long"
		}
		got = append(got, kind)
	}
	return got
}

// lines is analyze's companion: the reported line numbers, which is what a
// caller has to match against the fixture it wrote.
func lines(t *testing.T, src string) []int {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	findings, err := process(path, false, map[int]bool{})
	if err != nil {
		t.Fatalf("process: %v", err)
	}
	got := make([]int, 0, len(findings))
	for _, f := range findings {
		got = append(got, f.line)
	}
	return got
}

func equal(a []int, b ...int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestReportsOrphanedShortLine(t *testing.T) {
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 50) + "\n" + // 30 columns short, next word fits: ragged
		mkLine(0, 79) + "\n" +
		mkLine(0, 40) + "\n" + // the paragraph's last line: short by right
		"var x = 1\n"
	if got := lines(t, src); !equal(got, 4) {
		t.Fatalf("reported lines = %v, want [4] (only the stranded mid-paragraph line)", got)
	}
}

func TestKeepsBreaksTheNextWordCannotFill(t *testing.T) {
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 79) + "\n" +
		mkLine(0, 55) + "\n" + // 25 columns short, but the next word cannot be pulled up
		"// " + strings.Repeat("z", 40) + " abcd\n" + // first word 40 wide
		"var x = 1\n"
	if got := lines(t, src); len(got) != 0 {
		t.Fatalf("reported lines = %v, want none: a break before an unpullable word is not ragged", got)
	}
}

func TestKeepsBreaksWithinSlack(t *testing.T) {
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 66) + "\n" + // 14 short — inside shortSlack, so a hand-wrap choice
		mkLine(0, 79) + "\n" +
		mkLine(0, 30) + "\n" +
		"var x = 1\n"
	if got := lines(t, src); len(got) != 0 {
		t.Fatalf("reported lines = %v, want none: %d columns short is inside shortSlack (%d)", got, 80-66, shortSlack)
	}
}

func TestReportsOverLongLineOnlyPastMaxFillWidth(t *testing.T) {
	// A line standing clear of its paragraph by more than longSlack, but still
	// inside a width paragraphs here are filled to, is left alone.
	inside := "package p\n\n" +
		mkLine(0, 86) + "\n" +
		mkLine(0, 70) + "\n" +
		mkLine(0, 69) + "\n" +
		mkLine(0, 40) + "\n" +
		"var x = 1\n"
	if got := lines(t, inside); len(got) != 0 {
		t.Fatalf("reported lines = %v, want none: 86 columns is inside maxFillWidth (%d)", got, maxFillWidth)
	}

	past := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 100) + "\n" +
		mkLine(0, 79) + "\n" +
		mkLine(0, 78) + "\n" +
		mkLine(0, 40) + "\n" +
		"var x = 1\n"
	got := lines(t, past)
	if !equal(got, 4) {
		t.Fatalf("reported lines = %v, want [4] (the over-long line)", got)
	}
	if kinds := analyze(t, past); len(kinds) != 1 || kinds[0] != "long" {
		t.Fatalf("finding kinds = %v, want [long]", kinds)
	}
}

func TestCarveOuts(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{{
		// A list item opens its own paragraph, so the short line above it is a
		// last line and the item's own break is judged against its own lines.
		name: "list item",
		src: "package p\n\n" +
			mkLine(0, 80) + "\n" +
			mkLine(0, 79) + "\n" +
			mkLine(0, 50) + "\n" +
			"//   - abcd abcd\n" +
			"//     abcd abcd abcd\n" +
			"var x = 1\n",
	}, {
		// A verbatim block (four columns of body indentation) is laid out by
		// hand and rendered preformatted by go doc.
		name: "verbatim block",
		src: "package p\n\n" +
			"//\tabcd abcd\n" +
			"//\tabcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd\n" +
			"var x = 1\n",
	}, {
		name: "heading marker",
		src: "package p\n\n" +
			mkLine(0, 80) + "\n" +
			"// # Section\n" +
			mkLine(0, 79) + "\n" +
			mkLine(0, 40) + "\n" +
			"var x = 1\n",
	}, {
		name: "directive",
		src: "package p\n\n" +
			"//go:generate go tool commentwrap\n" +
			mkLine(0, 79) + "\n" +
			"var x = 1\n",
	}, {
		// A `//` comment sharing its line with anything else joins the group
		// the full-line comment below it opens, but its width is set by what
		// sits to its left, not by a wrap, so it is no part of the paragraph.
		name: "comment sharing its line",
		src: "package p\n\n" +
			"/* note */ // abcd\n" +
			mkLine(0, 79) + "\n" +
			mkLine(0, 40) + "\n" +
			"var y = 2\n",
	}, {
		// The expected-output block of an Example is the string go test
		// compares against, not prose.
		name: "example output block",
		src: "package p\n\nfunc ExampleX() {\n" +
			"\t// Output: abcd abcd\n" +
			mkLine(1, 79) + "\n" +
			mkLine(1, 40) + "\n" +
			"}\n",
	}, {
		// Nothing here was ever filled to a width, so no break in it can be
		// ragged against one.
		name: "narrow paragraph",
		src: "package p\n\n" +
			mkLine(0, 55) + "\n" +
			mkLine(0, 20) + "\n" +
			mkLine(0, 50) + "\n" +
			"var x = 1\n",
	}, {
		name: "generated file",
		src: "// Code generated by tools/x; DO NOT EDIT.\n\npackage p\n\n" +
			mkLine(0, 80) + "\n" +
			mkLine(0, 50) + "\n" +
			mkLine(0, 79) + "\n" +
			mkLine(0, 40) + "\n" +
			"var x = 1\n",
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := lines(t, tc.src); len(got) != 0 {
				t.Fatalf("reported lines = %v, want none", got)
			}
		})
	}
}

func TestIssueReferenceIsNotAHeading(t *testing.T) {
	// "#329" opens paragraph text constantly in this repo; only "# " marks a
	// section. Were it treated as a break, the paragraph would split and the
	// short line above it would pass as a last line.
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 50) + "\n" +
		"// #329 abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd\n" +
		mkLine(0, 40) + "\n" +
		"var x = 1\n"
	if got := lines(t, src); !equal(got, 4) {
		t.Fatalf("reported lines = %v, want [4]", got)
	}
}

func TestFixReflowsFromTheFirstRaggedBreak(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	// 70 columns is inside shortSlack of the paragraph's 79, so this line is not
	// itself ragged — but a reflow that started at the paragraph's top instead
	// of at the ragged break would pull words up into it.
	first := mkLine(0, 70)
	src := "package p\n\n" +
		first + "\n" +
		mkLine(0, 50) + "\n" +
		mkLine(0, 79) + "\n" +
		mkLine(0, 40) + "\n" +
		"var x = 1\n"
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	if _, err := process(path, true, map[int]bool{}); err != nil {
		t.Fatalf("process(fix): %v", err)
	}
	fixed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixed file: %v", err)
	}

	got := strings.Split(string(fixed), "\n")
	if got[2] != first {
		t.Fatalf("line above the ragged break was rewritten:\n got %q\nwant %q", got[2], first)
	}
	for i, line := range got[3:6] {
		if len(line) > 79 {
			t.Fatalf("reflowed line %d is %d columns, past the paragraph's 79", i+4, len(line))
		}
	}

	findings, err := process(path, false, map[int]bool{})
	if err != nil {
		t.Fatalf("process(recheck): %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("the fixed file still reports %d finding(s): %v", len(findings), findings)
	}
}

// TestFixKeepsABlockCommentInsideItsGroup covers the shape go/ast collects into
// a single CommentGroup: `//` lines, a whole-line `/* */`, then `//` lines
// again. The block comment carries no paragraph text, so it has to END the run
// rather than be stepped over — a paragraph accumulated across it spans its
// line, and a reflow splices over every line it spans, deleting the block
// comment silently (#329).
func TestFixKeepsABlockCommentInsideItsGroup(t *testing.T) {
	const block = "/* keep me */"
	// The block comment sits on line 7, between two runs that each carry a
	// ragged break of their own, so -fix splices on both sides of it.
	const blockLine = 7
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 50) + "\n" +
		mkLine(0, 79) + "\n" +
		mkLine(0, 40) + "\n" +
		block + "\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 50) + "\n" +
		mkLine(0, 79) + "\n" +
		mkLine(0, 40) + "\n" +
		"var x = 1\n"

	dir := t.TempDir()
	path := filepath.Join(dir, "x.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if got := strings.Split(src, "\n")[blockLine-1]; got != block {
		t.Fatalf("fixture line %d = %q, want %q", blockLine, got, block)
	}

	if _, err := process(path, true, map[int]bool{}); err != nil {
		t.Fatalf("process(fix): %v", err)
	}
	fixed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixed file: %v", err)
	}
	if string(fixed) == src {
		t.Fatal("-fix reflowed nothing, so this fixture no longer exercises a splice beside a block comment")
	}

	got := strings.Split(string(fixed), "\n")
	kept := 0
	for _, line := range got {
		if line == block {
			kept++
		}
	}
	if kept != 1 {
		t.Fatalf("the block comment survives on %d whole lines after -fix, want 1:\n%s", kept, fixed)
	}

	findings, err := process(path, false, map[int]bool{})
	if err != nil {
		t.Fatalf("process(recheck): %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("the fixed file still reports %d finding(s): %v", len(findings), findings)
	}

	// The same defect stated on the paragraphs themselves, where -fix reads its
	// splice ranges from: neither run may reach across the block comment's line.
	// comments parses the source it is handed, so this reads the fixture as
	// written, not the reflowed file on disk.
	paras, err := comments(path, src, strings.Split(src, "\n"))
	if err != nil {
		t.Fatalf("comments: %v", err)
	}
	if len(paras) != 2 {
		t.Fatalf("got %d paragraph(s), want 2: one on each side of the block comment", len(paras))
	}
	for _, para := range paras {
		first, last := para[0].Num, para[len(para)-1].Num
		if first < blockLine && last > blockLine {
			t.Fatalf("paragraph spans lines %d-%d, the range -fix splices over, and line %d in it is %q",
				first, last, blockLine, block)
		}
	}
}

// fix writes src to a temporary Go file, runs -fix over it and returns the
// fixed file's lines.
func fix(t *testing.T, src string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "x.go")
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}
	if _, err := process(path, true, map[int]bool{}); err != nil {
		t.Fatalf("process(fix): %v", err)
	}
	fixed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading fixed file: %v", err)
	}
	return strings.Split(string(fixed), "\n")
}

// TestFixKeepsInteriorRunsOfSpaces is #3903's sighting: the ragged line holds a
// quoted literal whose two-space runs are its meaning, and stays the first
// output line, so its runs are inside a line and not on a break.
func TestFixKeepsInteriorRunsOfSpaces(t *testing.T) {
	const literal = `// abcd abcd an xs:token holding "  a  b" is kept`
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		literal + "\n" +
		mkLine(0, 79) + "\n" +
		mkLine(0, 40) + "\n" +
		"var x = 1\n"
	got := fix(t, src)
	if want := literal + strings.Repeat(" abcd", 6); got[3] != want {
		t.Fatalf("reflowed line 4:\n got %q\nwant %q", got[3], want)
	}
}

// TestFixDropsTheRunABreakLandsOn places a two-space run where the greedy fill
// reaches it at column 78 of an 80-column paragraph: kept and counted, `  y`
// would end at 81, so the break lands on the run and drops it whole, where a
// single space would have fit `y` at 80.
func TestFixDropsTheRunABreakLandsOn(t *testing.T) {
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		mkLine(0, 50) + "\n" +
		"//" + strings.Repeat(" abcd", 5) + " ab  y" + strings.Repeat(" abcd", 9) + "\n" +
		mkLine(0, 40) + "\n" +
		"var x = 1\n"
	got := fix(t, src)
	if want := mkLine(0, 50) + strings.Repeat(" abcd", 5) + " ab"; got[3] != want {
		t.Fatalf("reflowed line 4:\n got %q\nwant %q", got[3], want)
	}
	if !strings.HasPrefix(got[4], "// y abcd") {
		t.Fatalf("reflowed line 5 = %q, want it to open with the lead, then y", got[4])
	}
}

// TestFixIsOnePassOverARunItWouldBreakOn is #4032's shape: the greedy fill
// reaches `a  ABCDEFGHIJKLMNOPQRST` at column 59 of an 80-column paragraph,
// where the two-space run overflows the line by one and a single space fits it
// exactly. Breaking on the run would end the line 21 columns short, before a
// word the check reads as fitting at a one-space join, so the run collapses
// instead: the check is clean on what -fix wrote and a second -fix is a no-op.
func TestFixIsOnePassOverARunItWouldBreakOn(t *testing.T) {
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		"//" + strings.Repeat(" abcd", 9) + "\n" +
		"// abcd abcd a  ABCDEFGHIJKLMNOPQRST" + strings.Repeat(" abcd", 8) + "\n" +
		"// abcd abcd\n" +
		"var x = 1\n"
	once := strings.Join(fix(t, src), "\n")
	if got := lines(t, once); len(got) != 0 {
		t.Fatalf("the check reports lines %v on -fix's output:\n%s", got, once)
	}
	if twice := strings.Join(fix(t, once), "\n"); twice != once {
		t.Fatalf("a second -fix changed the file:\n got %q\nwant %q", twice, once)
	}
}

// TestReportsAShortLineBeforeASingleSpacedWord is the guard on that collapse:
// the same 21-column shortfall before the same word, joined to the next line
// by no run, is still the ragged shape the check reports.
func TestReportsAShortLineBeforeASingleSpacedWord(t *testing.T) {
	src := "package p\n\n" +
		mkLine(0, 80) + "\n" +
		"//" + strings.Repeat(" abcd", 11) + " a\n" +
		"// ABCDEFGHIJKLMNOPQRST" + strings.Repeat(" abcd", 10) + "\n" +
		"var x = 1\n"
	if got := lines(t, src); !equal(got, 4) {
		t.Fatalf("reported lines %v, want [4]", got)
	}
}

// TestReflowDropsATrailingRun hands reflow a body ending in a run directly:
// commentpara takes a `//` line with trailing whitespace for one sharing its
// line with code, so no such body reaches reflow through process. The run joins
// nothing, and the next line's first word follows by one space.
func TestReflowDropsATrailingRun(t *testing.T) {
	para := []commentpara.Line{
		{Num: 1, Raw: "// abcd  b  ", Body: " abcd  b  "},
		{Num: 2, Raw: "// c", Body: " c"},
	}
	got := reflow(para, 80, 0)
	if len(got) != 1 || got[0] != "// abcd  b c" {
		t.Fatalf("reflow = %q, want [%q]", got, "// abcd  b c")
	}
}

// TestReflowKeepsANonASCIISpace is parser/produce_test.go's `" true"` shape:
// the quoted value opens with a U+00A0, which strings.Fields splits on and a
// single-space join turned into U+0020, changing the value the comment quotes.
func TestReflowKeepsANonASCIISpace(t *testing.T) {
	para := []commentpara.Line{
		{Num: 1, Raw: "// so \" true\" stays", Body: " so \" true\" stays"},
		{Num: 2, Raw: "// out", Body: " out"},
	}
	want := "// so \" true\" stays out"
	if got := reflow(para, 80, 0); len(got) != 1 || got[0] != want {
		t.Fatalf("reflow = %q, want [%q]", got, want)
	}
}

// TestFixWithoutInteriorRunsIsUnchanged pins -fix's output on paragraphs with
// no interior run to what the tool produced before #3903, byte for byte: the
// want lines were produced by the base tool, not computed here. The list item's
// continuation lines carry five columns of body indentation, which is lead and
// not an interior run, so the words they open join the line above by one space.
func TestFixWithoutInteriorRunsIsUnchanged(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{{
		name: "plain",
		src: "package p\n\n" +
			mkLine(0, 80) + "\n" +
			mkLine(0, 50) + "\n" +
			mkLine(0, 79) + "\n" +
			mkLine(0, 40) + "\n" +
			"var x = 1\n",
		want: "package p\n\n// abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcd abcdxxx\n// abcd abcd abcd abcd abcd abcd abcd abcd abcdxxx abcd abcd abcd abcd abcd abcd\n// abcd abcd abcd abcd abcd abcd abcd abcd abcdxx abcd abcd abcd abcd abcd abcd\n// abcdxxx\nvar x = 1\n",
	}, {
		name: "list item continuation",
		src: "package p\n\n" +
			"//  6. A value no valid document holds, the first in document order, an\n" +
			"//     element's attributes before its content: an\n" +
			"//     error, which errors.Is reports as [ErrInvalidValue], naming the Go\n" +
			"//     path of its field from T and, for a simple value, its simple type.\n" +
			"var x = 1\n",
		want: "package p\n\n//  6. A value no valid document holds, the first in document order, an\n//     element's attributes before its content: an error, which errors.Is\n//     reports as [ErrInvalidValue], naming the Go path of its field from\n//     T and, for a simple value, its simple type.\nvar x = 1\n",
	}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.Join(fix(t, tc.src), "\n"); got != tc.want {
				t.Fatalf("-fix output:\n got %q\nwant %q", got, tc.want)
			}
		})
	}
}

func TestWaiverCoversItsParagraph(t *testing.T) {
	if len(waivers) == 0 {
		t.Skip("no waivers to exercise")
	}
	w := waivers[0]
	para := []commentpara.Line{
		{Num: 1, Raw: "// abcd", Body: " abcd"},
		{Num: 2, Raw: w.text, Body: strings.TrimPrefix(w.text, "//")},
	}
	if _, ok := waivedBy(filepath.Join("some", "root", w.file), para); !ok {
		t.Fatalf("waiver %s did not cover the paragraph carrying its line", w.issue)
	}
	if _, ok := waivedBy(filepath.Join("some", "root", w.file), para[:1]); ok {
		t.Fatalf("waiver %s covered a paragraph that does not carry its line", w.issue)
	}
	// The waiver names whole path elements: a file whose name merely ends in the
	// waived path is a different file.
	if _, ok := waivedBy("x"+w.file, para); ok {
		t.Fatalf("waiver %s covered %q, a different file that only ends in the waived path", w.issue, "x"+w.file)
	}
}

func TestStaleWaiverIsReported(t *testing.T) {
	if len(waivers) == 0 {
		t.Skip("no waivers to exercise")
	}
	got := staleWaivers([]string{waivers[0].file}, map[int]bool{})
	if len(got) == 0 {
		t.Fatal("a waiver that matched nothing in a checked file was not reported as stale")
	}
	if !strings.Contains(got[0].msg, waivers[0].issue) {
		t.Fatalf("stale-waiver message %q does not name the issue that retires it", got[0].msg)
	}
	if unreported := staleWaivers([]string{"other/file.go"}, map[int]bool{}); len(unreported) != 0 {
		t.Fatalf("a waiver whose file was not checked was reported stale: %v", unreported)
	}
}
