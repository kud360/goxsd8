package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestWaiverCoversItsParagraph(t *testing.T) {
	if len(waivers) == 0 {
		t.Skip("no waivers to exercise")
	}
	w := waivers[0]
	para := []commentLine{
		{num: 1, raw: "// abcd", body: " abcd"},
		{num: 2, raw: w.text, body: strings.TrimPrefix(w.text, "//")},
	}
	if _, ok := waivedBy(filepath.Join("some", "root", w.file), para); !ok {
		t.Fatalf("waiver %s did not cover the paragraph carrying its line", w.issue)
	}
	if _, ok := waivedBy(filepath.Join("some", "root", w.file), para[:1]); ok {
		t.Fatalf("waiver %s covered a paragraph that does not carry its line", w.issue)
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
