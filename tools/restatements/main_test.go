package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures rebuild the sites #2723's and #2781's round-1 verdicts
// (issuecomment-6261422314 findings 1 and 2, issuecomment-6260225049 finding
// 2) rejected, and the sites those verdicts ruled sound, in reconstructed
// frames: the trees are gone with their squash merges, so each carries the
// verdict's quoted words verbatim.

// ownerLocations is validate/jsonsrc/doc.go's end-of-input rule, the owner F1
// and F2 restate. It is on the working-tree side, as #2723 added it too.
const ownerLocations = `An error at the end of the input, which ended while a value or a close was still due, is on the input's last line, the one holding its last byte: at that byte where it is the '\n' ending the line, and one past it otherwise, 1:1 for an input with no byte.`

// ownerClause is jsonsrc.Validate's clause bullet, the owner F3 restates.
const ownerClause = `the cvc-complex-type charge at the scalar of an element whose type has empty content (clause 1.1), or element-only content where the scalar's lexical form is not all white space (clause 1.3): object expected.`

// ownerPointer is (*Error).Loc's doc, the tree's second pointer at the
// "Locations" section: the guard rows hold it in the owner population, where
// the bare pointer would otherwise match it.
const ownerPointer = `Loc is the location of the construct e names, as the package doc's "Locations" section places it, under the URI [WithURI] gives: empty, and rendered "?", without one.`

const (
	f1 = `TestEndOfInputLocation pins the location of an error at the end of the input, on the line holding the input's last byte, at that byte where it is a final '\n', one past it otherwise, whatever sizes the reader returns (positions.start, #2723).`
	f2 = `Past the bytes read so far it locates one past the last of them, but at the last where r ended and that byte is '\n': the end of the input, which the package doc's "Locations" section places on the line holding its last byte, so the final '\n' opens no line.`
	f3 = `A cvc-complex-type violation at a scalar member whose element's type has empty or element-only content ends with a clause.`

	g1  = `the end of the input, as the package doc's "Locations" section places it.`
	g2a = `Past the bytes read so far it locates one past the last of them, but at the last where r ended and that byte is '\n': the end of the input, as the package doc's "Locations" section places it.`
	g2b = `Where r has not ended, as after a fault reading it, a last '\n' read is counted, since the input goes on past it.`

	// rendered is a top-level usage sentence, which cmd/goxsd8/doc.go
	// renders word for word.
	rendered = `Implemented today: the help path, parse and validate, and validate assesses XML and JSON instances against the compiled schema set.`
)

// wrap renders sentences as one `//` paragraph, each sentence opening a line
// and wrapped at width columns.
func wrap(width int, indent string, sentences ...string) string {
	var b strings.Builder
	for _, s := range sentences {
		line := indent + "//"
		for _, w := range strings.Fields(s) {
			if len(line)+1+len(w) > width && line != indent+"//" {
				b.WriteString(line + "\n")
				line = indent + "//"
			}
			line += " " + w
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

// comment is wrap at 76 columns.
func comment(indent string, sentences ...string) string {
	return wrap(76, indent, sentences...)
}

// usageSrc renders cmd/goxsd8/main.go with a usage constant whose validate
// block holds the prose sentences, and whose closing top-level paragraph is
// tail.
func usageSrc(tail string, prose ...string) string {
	var b strings.Builder
	b.WriteString("package main\n\nconst usage = `goxsd8 — XSD 1.1 schema compilation.\n\nUsage:\n\n")
	b.WriteString("  goxsd8 validate -schema <schema.xsd> <instance>...\n")
	for _, l := range strings.Split(strings.TrimSuffix(comment("", prose...), "\n"), "\n") {
		b.WriteString("      " + strings.TrimPrefix(l, "// ") + "\n")
	}
	b.WriteString("         {\"o:order\":{\"kind\":\"x\"}}\n\n")
	b.WriteString(tail + "\n`\n\nfunc main() {}\n")
	return b.String()
}

// baseTree is the tree every row's working tree adds to: the jsonsrc package
// with its owners but the end-of-input rule, and cmd/goxsd8 with its usage
// and the doc.go rendering usage's top-level prose.
func baseTree() map[string]string {
	return map[string]string{
		"go.mod": "module example.com/m\n\ngo 1.26\n",
		"validate/jsonsrc/doc.go": "// Package jsonsrc validates JSON instances.\n//\n// # Locations\n//\n" +
			comment("", "Every Loc's line and column count from the first byte after the byte order mark.") +
			"package jsonsrc\n",
		"validate/jsonsrc/jsonsrc.go": "package jsonsrc\n\n" +
			comment("", "Validate assesses a JSON instance.") + "//\n" +
			comment("", `A violation's rule ID and location are the engine's, and so is its message but for one clause Validate appends to it, opening "; JSON member" and naming the member's key as spelled, an array item's being its array's:`) +
			"//\n" + comment("", "- "+ownerClause) +
			"func Validate() {}\n\n" +
			comment("", "Error is an error Validate returns.") + "type Error struct{}\n\n" +
			comment("", ownerPointer) + "func (e *Error) Loc() string { return \"\" }\n",
		"validate/jsonsrc/positions.go": "package jsonsrc\n\ntype positions struct{}\n\n" +
			comment("", "start locates the token that follows offset off.") +
			"func (p *positions) start(off int64) (line, col int) { return 0, 0 }\n",
		"cmd/goxsd8/main.go": usageSrc("Flags common to all subcommands: -q and -v.", "Assess instances against the compiled set."),
		"cmd/goxsd8/doc.go": comment("", "Command goxsd8 compiles schemas.") + "//\n// # Usage\n//\n" +
			comment("", "Flags common to all subcommands: -q and -v.") + "package main\n",
	}
}

// withOwner adds the end-of-input rule to the tree's "Locations" section,
// where #2723 added it, wrapped at width columns.
func withOwner(tree map[string]string, width int) map[string]string {
	tree["validate/jsonsrc/doc.go"] = "// Package jsonsrc validates JSON instances.\n//\n// # Locations\n//\n" +
		wrap(width, "", "Every Loc's line and column count from the first byte after the byte order mark.", ownerLocations) +
		"package jsonsrc\n"
	return tree
}

func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, src := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
}

// lineOf is the line of files[rel] that text opens: the first whose words,
// past any `//` and list bullet, start with text's first three. The fixtures
// open every sentence on a line of its own.
func lineOf(t *testing.T, files map[string]string, rel, text string) int {
	t.Helper()
	fields := strings.Fields(text)
	head := strings.Join(fields[:min(3, len(fields))], " ")
	for i, l := range strings.Split(files[rel], "\n") {
		l = strings.TrimPrefix(strings.TrimSpace(l), "//")
		l = strings.TrimPrefix(strings.TrimSpace(l), "- ")
		if strings.HasPrefix(strings.Join(strings.Fields(l), " "), head) {
			return i + 1
		}
	}
	t.Fatalf("no line of %s opens %q", rel, head)
	return 0
}

// surveyTrees writes base and head to temporary directories and surveys them.
func surveyTrees(t *testing.T, base, head map[string]string) report {
	t.Helper()
	b, h := t.TempDir(), t.TempDir()
	writeTree(t, b, base)
	writeTree(t, h, head)
	rep, err := survey(b, h)
	if err != nil {
		t.Fatalf("survey: %v", err)
	}
	return rep
}

// site is a file:line.
type site struct {
	file string
	line int
}

// findingAt returns the finding whose added sentence is at s.
func findingAt(rep report, s site) (finding, bool) {
	for _, f := range rep.findings {
		if f.added.file == s.file && f.added.line == s.line {
			return f, true
		}
	}
	return finding{}, false
}

// TestRestatementsPrinted holds F1-F3: each added sentence is printed with the
// owner sentence it restates. Each fails when the matcher pairs nothing:
// shared returning 0 fails all three rows.
func TestRestatementsPrinted(t *testing.T) {
	rows := []struct {
		name  string
		add   func(map[string]string) map[string]string
		added func(map[string]string) site
		owner func(map[string]string) site
	}{{
		name: "F1 test comment restating the Locations end-of-input rule",
		add: func(tree map[string]string) map[string]string {
			tree["validate/jsonsrc/positions_test.go"] = "package jsonsrc\n\nimport \"testing\"\n\n" +
				comment("", f1) + "func TestEndOfInputLocation(t *testing.T) {}\n"
			return tree
		},
		added: func(tree map[string]string) site {
			return site{"validate/jsonsrc/positions_test.go", lineOf(t, tree, "validate/jsonsrc/positions_test.go", f1)}
		},
		owner: func(tree map[string]string) site {
			return site{"validate/jsonsrc/doc.go", lineOf(t, tree, "validate/jsonsrc/doc.go", ownerLocations)}
		},
	}, {
		name: "F2 positions.go pointer paraphrasing the rule it points at",
		add: func(tree map[string]string) map[string]string {
			tree["validate/jsonsrc/positions.go"] = "package jsonsrc\n\ntype positions struct{}\n\n" +
				comment("", "start locates the token that follows offset off.") + "//\n" +
				comment("", f2) +
				"func (p *positions) start(off int64) (line, col int) { return 0, 0 }\n"
			return tree
		},
		added: func(tree map[string]string) site {
			return site{"validate/jsonsrc/positions.go", lineOf(t, tree, "validate/jsonsrc/positions.go", f2)}
		},
		owner: func(tree map[string]string) site {
			return site{"validate/jsonsrc/doc.go", lineOf(t, tree, "validate/jsonsrc/doc.go", ownerLocations)}
		},
	}, {
		name: "F3 usage restating jsonsrc.Validate's clause bullet",
		add: func(tree map[string]string) map[string]string {
			tree["cmd/goxsd8/main.go"] = usageSrc("Flags common to all subcommands: -q and -v.", "Assess instances against the compiled set.", f3)
			return tree
		},
		added: func(tree map[string]string) site {
			return site{"cmd/goxsd8/main.go", lineOf(t, tree, "cmd/goxsd8/main.go", f3)}
		},
		owner: func(tree map[string]string) site {
			return site{"validate/jsonsrc/jsonsrc.go", lineOf(t, tree, "validate/jsonsrc/jsonsrc.go", ownerClause)}
		},
	}}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			head := row.add(withOwner(baseTree(), 76))
			rep := surveyTrees(t, baseTree(), head)
			added, owner := row.added(head), row.owner(head)
			f, ok := findingAt(rep, added)
			if !ok {
				t.Fatalf("no finding at %s:%d; findings: %+v", added.file, added.line, rep.findings)
			}
			for _, m := range f.matches {
				if m.owner.file == owner.file && m.owner.line == owner.line {
					return
				}
			}
			t.Fatalf("finding at %s:%d does not name the owner at %s:%d: %+v", added.file, added.line, owner.file, owner.line, f.matches)
		})
	}
}

// TestPointersNotPrinted holds the guard rows: a bare pointer at the
// "Locations" section (G1) and positions.start's own sentences, a mechanism
// sharing the owner's identifiers (G2), are not printed, with (*Error).Loc's
// pointer in the owner population. minShared at 1 fails every row.
func TestPointersNotPrinted(t *testing.T) {
	rows := []struct {
		name  string
		file  string
		src   string
		texts []string
	}{{
		name:  "G1 bare pointer",
		file:  "validate/jsonsrc/atend.go",
		src:   "package jsonsrc\n\nfunc atEnd() bool {\n" + comment("\t", g1) + "\treturn true\n}\n",
		texts: []string{g1},
	}, {
		name: "G2 positions.start's own sentences",
		file: "validate/jsonsrc/positions.go",
		src: "package jsonsrc\n\ntype positions struct{}\n\n" +
			comment("", "start locates the token that follows offset off.") + "//\n" +
			comment("", g2a, g2b) +
			"func (p *positions) start(off int64) (line, col int) { return 0, 0 }\n",
		texts: []string{g2a, g2b},
	}}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			head := withOwner(baseTree(), 76)
			head[row.file] = row.src
			rep := surveyTrees(t, baseTree(), head)
			if rep.added == 0 {
				t.Fatal("the working tree added no sentence")
			}
			for _, text := range row.texts {
				at := site{row.file, lineOf(t, head, row.file, text)}
				if f, ok := findingAt(rep, at); ok {
					t.Errorf("%q is printed against %+v", text, f.matches)
				}
			}
		})
	}
}

// TestUsageNotPairedWithItsRendering adds a top-level usage sentence and its
// cmd/goxsd8/doc.go rendering: neither is paired with the other or with
// itself, and the rendering is read as no added sentence.
func TestUsageNotPairedWithItsRendering(t *testing.T) {
	head := baseTree()
	head["cmd/goxsd8/main.go"] = usageSrc(rendered, "Assess instances against the compiled set.")
	head["cmd/goxsd8/doc.go"] = comment("", "Command goxsd8 compiles schemas.") + "//\n// # Usage\n//\n" +
		comment("", rendered) + "package main\n"
	rep := surveyTrees(t, baseTree(), head)
	if rep.added != 1 {
		t.Errorf("added = %d, want 1: the usage sentence, its rendering not counted", rep.added)
	}
	for _, f := range rep.findings {
		if f.added.file == renderFile {
			t.Errorf("the rendering at %s:%d is read as an added sentence", f.added.file, f.added.line)
		}
		for _, m := range f.matches {
			if m.owner.text == f.added.text {
				t.Errorf("%s:%d is paired with its own words at %s:%d", f.added.file, f.added.line, m.owner.file, m.owner.line)
			}
		}
	}
}

// TestReflowAddsNothing rewraps an unchanged owner paragraph and an unchanged
// test comment: no sentence is added.
func TestReflowAddsNothing(t *testing.T) {
	base := withOwner(baseTree(), 76)
	base["validate/jsonsrc/positions_test.go"] = "package jsonsrc\n\n" + wrap(76, "", f1) + "var x = 1\n"
	head := withOwner(baseTree(), 50)
	head["validate/jsonsrc/positions_test.go"] = "package jsonsrc\n\n" + wrap(50, "", f1) + "var x = 1\n"
	for _, rel := range []string{"validate/jsonsrc/doc.go", "validate/jsonsrc/positions_test.go"} {
		if head[rel] == base[rel] {
			t.Fatalf("the fixture reflowed nothing in %s", rel)
		}
	}
	rep := surveyTrees(t, base, head)
	if rep.added != 0 {
		t.Fatalf("a reflow added %d sentence(s): %+v", rep.added, rep.findings)
	}
}

// TestSentences pins how prose is cut: at a full stop, question or
// exclamation mark closing a word, but not at an abbreviation or an ellipsis,
// each sentence at the line its first word is on.
func TestSentences(t *testing.T) {
	var words []word
	words = appendWords(words, `One fact (e.g. this one). Is it? "Yes."`, 3)
	words = appendWords(words, `Then ... more, ending here`, 4)
	got := sentences(words, "f.go", false)
	want := []sentence{
		{file: "f.go", line: 3, text: "One fact (e.g. this one)."},
		{file: "f.go", line: 3, text: "Is it?"},
		{file: "f.go", line: 3, text: `"Yes."`},
		{file: "f.go", line: 4, text: "Then ... more, ending here"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d sentences %+v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sentence %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// git runs a git command in dir with a throwaway identity.
func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "core.hooksPath=/dev/null"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// TestCLIExitsZero runs the command against a committed base in a scratch
// repository: it exits 0 with matches and without, and the summary says the
// matching is heuristic either way. An unresolvable ref, and no -base, exit 2.
func TestCLIExitsZero(t *testing.T) {
	repo := t.TempDir()
	writeTree(t, repo, withOwner(baseTree(), 76))
	git(t, repo, "init", "-q")
	git(t, repo, "add", "-A")
	git(t, repo, "commit", "-q", "-m", "base")
	t.Chdir(repo)

	const notice = "the matching is heuristic"
	var out, errOut bytes.Buffer
	if code := cli([]string{"-base", "HEAD"}, &out, &errOut); code != 0 {
		t.Fatalf("no change: exit %d, want 0; stderr: %s", code, errOut.String())
	}
	if out.String() != "restatements: 0 of 0 added sentence(s) overlap a sentence of a package doc, an exported identifier's doc or cmd/goxsd8's usage; the matching is heuristic (2 adjacent content-word pairs shared with one clause), so read each pair before acting on it\n" {
		t.Errorf("no change printed %q", out.String())
	}

	writeTree(t, repo, map[string]string{
		"validate/jsonsrc/positions_test.go": "package jsonsrc\n\nimport \"testing\"\n\n" + comment("", f1) + "func TestEndOfInputLocation(t *testing.T) {}\n",
	})
	out.Reset()
	errOut.Reset()
	if code := cli([]string{"-base", "HEAD"}, &out, &errOut); code != 0 {
		t.Fatalf("a restatement: exit %d, want 0; stderr: %s", code, errOut.String())
	}
	if !strings.HasPrefix(out.String(), "validate/jsonsrc/positions_test.go:5: TestEndOfInputLocation pins") ||
		!strings.Contains(out.String(), "\n\tvalidate/jsonsrc/doc.go:") || !strings.Contains(out.String(), notice) {
		t.Errorf("a restatement printed %q", out.String())
	}

	for _, args := range [][]string{{"-base", "no-such-ref"}, nil} {
		out.Reset()
		errOut.Reset()
		if code := cli(args, &out, &errOut); code != 2 {
			t.Errorf("cli(%q): exit %d, want 2", args, code)
		}
		if !strings.HasPrefix(errOut.String(), "restatements: ") {
			t.Errorf("cli(%q) stderr %q", args, errOut.String())
		}
	}
}
