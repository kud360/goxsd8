package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// readmeTranscriptPrompt opens a README.md transcript line: the command after
// it is run, and the lines after it are what the run prints.
const readmeTranscriptPrompt = "$ goxsd8 "

// TestReadmeTranscripts holds every `$ goxsd8 …` transcript in README.md's sh
// fences to what run prints for it (#3784). A transcript is a fence line
// opening "$ goxsd8 " and ending in a "# exit N" comment, and the lines after
// it up to the next line opening "$ " or the fence's end; it holds the
// command's exit code to N and its stdout's lines followed by its stderr's to
// those lines. Each command runs from testdata/readme, which holds a committed
// copy of every file a transcript names, the files README's heredocs write
// into a fresh directory included; README's other goxsd8 lines open with no
// "$ " and show no output, and are not run.
func TestReadmeTranscripts(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(filepath.Join("..", "..", "testdata", "readme"))
	lines := strings.Split(string(src), "\n")
	ran := 0
	open := false
	for i := 0; i < len(lines); i++ {
		switch {
		case !open && lines[i] == "```sh":
			open = true
			continue
		case open && lines[i] == "```":
			open = false
			continue
		}
		rest, ok := strings.CutPrefix(lines[i], readmeTranscriptPrompt)
		if !open || !ok {
			continue
		}
		command, comment, ok := strings.Cut(rest, "# exit ")
		if !ok {
			t.Errorf("README.md:%d: transcript %q carries no \"# exit N\" comment", i+1, lines[i])
			continue
		}
		wantCode, err := strconv.Atoi(strings.TrimSpace(comment))
		if err != nil {
			t.Errorf("README.md:%d: transcript %q: exit code: %v", i+1, lines[i], err)
			continue
		}
		var want strings.Builder
		j := i + 1
		for j < len(lines) && lines[j] != "```" && !strings.HasPrefix(lines[j], "$ ") {
			want.WriteString(lines[j] + "\n")
			j++
		}
		args := strings.Fields(command)
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)
		ran++
		if code != wantCode {
			t.Errorf("README.md:%d: goxsd8 %s exits %d, and README shows %d", i+1, strings.Join(args, " "), code, wantCode)
		}
		if got := stdout.String() + stderr.String(); got != want.String() {
			t.Errorf("README.md:%d-%d: goxsd8 %s prints, stdout then stderr:\n%s\nand README shows:\n%s", i+1, j, strings.Join(args, " "), got, want.String())
		}
		i = j - 1
	}
	if ran == 0 {
		t.Errorf("README.md shows no %q transcript in an sh fence to run", strings.TrimSpace(readmeTranscriptPrompt))
	}
}
