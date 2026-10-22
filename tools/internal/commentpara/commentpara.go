// Package commentpara reads Go source the way a reader of its `//` comments
// does: the Go files under a set of path arguments, and each comment group's
// prose paragraphs, with the source line of every comment line in them. It is
// the one reader of that layout for `go tool commentwrap` and `go tool
// restatements`, so the two cannot disagree about where a paragraph starts or
// ends.
//
// A paragraph is a run of whole-line `//` comments. It ends at a line that
// can carry no paragraph text (a blank `//`, a directive, a `#` heading, a
// markdown table row, a `/* */` block, a comment sharing its line with code),
// where a list item starts, and where the indentation leaves the band of the
// line that opened it; an [Example function]'s expected-output block ends the
// group's prose. A generated file carries none: its comments belong to its
// generator.
//
// [Example function]: https://pkg.go.dev/testing#hdr-Examples
package commentpara

import (
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	// verbatimIndent is the comment-body indentation (in columns after the
	// `//`) at which `go doc` stops treating text as prose and renders it
	// preformatted: spec quotes, XML fragments, code samples.
	verbatimIndent = 4
	// hangingIndent is how far a continuation line may be indented past the
	// line that opened its paragraph and still belong to it — enough for a list
	// item's hanging indent ("//   - text" continued at "//     text"), not
	// enough for a nested block.
	hangingIndent = 3
)

// Files expands path arguments into a sorted, de-duplicated list of Go files,
// so a rerun on an unchanged tree reads them in the same order (STYLE D1). A
// `...` suffix (as in `./...`) walks the directory recursively; a plain
// directory takes the Go files directly inside it; a file argument is taken
// as given. Directories holding no hand-written comment prose of ours — the
// git store, the vendored W3C suite and other testdata, hidden directories —
// are skipped wholesale.
func Files(args []string) ([]string, error) {
	seen := make(map[string]struct{})
	var files []string
	add := func(path string) {
		if _, dup := seen[path]; dup {
			return
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}

	for _, arg := range args {
		cleaned := filepath.Clean(arg)
		root, recursive := strings.CutSuffix(cleaned, string(filepath.Separator)+"...")
		if cleaned == "..." {
			root, recursive = ".", true
		}

		info, err := os.Stat(root)
		if err != nil {
			return nil, fmt.Errorf("resolving path %s: %w", arg, err)
		}
		if !info.IsDir() {
			add(root)
			continue
		}
		if err := walk(root, recursive, add); err != nil {
			return nil, fmt.Errorf("collecting from %s: %w", arg, err)
		}
	}

	sort.Strings(files)
	return files, nil
}

// walk feeds every Go file under root to add, skipping the directories Files
// names.
func walk(root string, recursive bool, add func(string)) error {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			if !recursive || skipDir(d.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) == ".go" {
			add(path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("walking directory tree: %w", err)
	}
	return nil
}

func skipDir(name string) bool {
	return name == "testdata" || name == "vendor" || strings.HasPrefix(name, ".")
}

// IsGenerated reports whether the file whose lines are given carries the
// conventional generated-code header (https://go.dev/s/generatedcode), which
// by convention precedes the package clause.
func IsGenerated(lines []string) bool {
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			return false
		}
		if strings.HasPrefix(line, "// Code generated ") && strings.HasSuffix(line, " DO NOT EDIT.") {
			return true
		}
	}
	return false
}

// Line is one whole-line `//` comment: its 1-based source line number, the raw
// line (indentation and marker included) and Body, the text after the marker.
type Line struct {
	Num  int
	Raw  string
	Body string
}

// Indent is how far the line's text sits past its `//`, counting a tab as a
// full verbatim step since that is how `go doc` reads it.
func (l Line) Indent() int {
	n := 0
	for _, r := range l.Body {
		if r == ' ' {
			n++
			continue
		}
		if r == '\t' {
			n += verbatimIndent
			continue
		}
		break
	}
	return n
}

// IsVerbatim reports whether a paragraph opening at this line is laid out by
// hand: indented to where `go doc` renders it preformatted, and not a list
// item.
func (l Line) IsVerbatim() bool {
	return l.Indent() >= verbatimIndent && !l.IsListItem()
}

// isBreak reports whether a comment line can carry no wrapped paragraph text
// at all: a blank `//`, a directive (`//go:generate`, `//nolint:…` — no space
// after the marker), a `#` heading marker inside a package doc, or a markdown
// table row.
func (l Line) isBreak() bool {
	trimmed := strings.TrimSpace(l.Body)
	if trimmed == "" {
		return true
	}
	if !strings.HasPrefix(l.Body, " ") && !strings.HasPrefix(l.Body, "\t") {
		return true
	}
	return isHeading(trimmed) || strings.HasPrefix(trimmed, "|")
}

// isHeading reports whether trimmed is a `#`-style section marker: one or more
// hashes, then a space or nothing. The space is load-bearing — this repo's
// comments open lines with "#329" issue references constantly, and those are
// ordinary paragraph text.
func isHeading(trimmed string) bool {
	hashes := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
	if hashes == 0 {
		return false
	}
	return len(trimmed) == hashes || trimmed[hashes] == ' '
}

// isExampleOutput reports whether a comment line opens the expected-output
// block of an Example function. Everything from there to the end of the
// comment is the string `go test` compares against, not prose.
func (l Line) isExampleOutput() bool {
	trimmed := strings.ToLower(strings.TrimSpace(l.Body))
	trimmed = strings.TrimPrefix(trimmed, "unordered ")
	return strings.HasPrefix(trimmed, "output:")
}

// IsListItem reports whether a comment line opens a list item — a bullet
// (`- `, `* `, `• `) or an enumerator (`1. `, `2) `, `(3) `, `a. `). Its
// continuation lines belong to it, so it opens a paragraph rather than
// breaking one, and the line above it is a legitimate short last line.
func (l Line) IsListItem() bool {
	trimmed := strings.TrimSpace(l.Body)
	for _, bullet := range []string{"- ", "* ", "• "} {
		if strings.HasPrefix(trimmed, bullet) {
			return true
		}
	}
	return isEnumerator(trimmed)
}

// isEnumerator reports whether trimmed opens with a numbered or lettered list
// marker: an optional "(", then digits or a single letter, then "." or ")" and
// a space.
func isEnumerator(trimmed string) bool {
	rest := strings.TrimPrefix(trimmed, "(")
	n := 0
	for n < len(rest) && rest[n] >= '0' && rest[n] <= '9' {
		n++
	}
	if n == 0 && len(rest) > 0 && isLetter(rest[0]) {
		n = 1
	}
	if n == 0 || len(rest) < n+2 {
		return false
	}
	if rest[n] != '.' && rest[n] != ')' {
		return false
	}
	return rest[n+1] == ' '
}

func isLetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// Paragraphs returns the paragraphs of one comment group, in source order,
// one-line ones included. fset positions the group, and lines are its file's
// source lines.
func Paragraphs(fset *token.FileSet, group *ast.CommentGroup, lines []string) [][]Line {
	// A comment group runs `//` lines and `/* */` blocks together, so a line
	// that cannot carry paragraph text ends the run rather than being stepped
	// over: a paragraph accumulated across it would span its line, and a
	// reflow splices over every line it spans, deleting it (#329).
	var paras [][]Line
	block := make([]Line, 0, len(group.List))
	flush := func() {
		paras = append(paras, split(block)...)
		block = nil
	}
	for _, c := range group.List {
		pos := fset.Position(c.Slash)
		if !strings.HasPrefix(c.Text, "//") || pos.Line > len(lines) {
			flush()
			continue
		}
		raw := lines[pos.Line-1]
		// A comment sharing its line with code is not part of a wrapped
		// paragraph: the code to its left, not a wrap width, decides where it
		// starts and how much room is left.
		if strings.TrimSpace(raw) != c.Text {
			flush()
			continue
		}
		block = append(block, Line{
			Num:  pos.Line,
			Raw:  raw,
			Body: strings.TrimPrefix(c.Text, "//"),
		})
	}
	flush()
	return paras
}

// split cuts a run of whole-line comments into paragraphs at the boundaries
// the package doc names.
func split(block []Line) [][]Line {
	var out [][]Line
	var cur []Line
	margin := 0
	flush := func() {
		if len(cur) > 0 {
			out = append(out, cur)
		}
		cur = nil
	}

	for _, line := range block {
		if line.isExampleOutput() {
			flush()
			return out
		}
		if line.isBreak() {
			flush()
			continue
		}
		indent := line.Indent()
		if line.IsListItem() || indent < margin || indent > margin+hangingIndent {
			flush()
		}
		if len(cur) == 0 {
			margin = indent
		}
		cur = append(cur, line)
	}
	flush()
	return out
}
