package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kud360/goxsd8/codegen"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
)

// flatFixture is codegen's own fixture of the shapes its first slice accepts.
// The CLI owns no golden: the file it writes names the schema by its absolute
// path, so the comparand is what codegen.Generate writes for the same schema.
const flatFixture = "../../codegen/testdata/flat.xsd"

// absFixture returns path resolved absolute, the location gen hands the parser.
func absFixture(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// generatedFile reads the xsd_gen.go Generate writes into dir.
func generatedFile(t *testing.T, dir string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(dir, "xsd_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	return body
}

// assertAbsent fails unless nothing exists at path.
func assertAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s exists (lstat: %v), want nothing written there", path, err)
	}
}

// TestGenWritesWhatGenerateWrites pins exit 0: each pair's -out receives the
// file codegen.Generate writes for a Target built from parser.Parse of the
// absolute location gen hands the parser, in the package the -out's last path
// element after filepath.Clean names: a trailing slash and a trailing ..
// included. -q changes nothing. Before #2964 this run exited 2, reporting gen
// as not yet implemented.
func TestGenWritesWhatGenerateWrites(t *testing.T) {
	flat := absFixture(t, flatFixture)
	tmp := t.TempDir()
	out := filepath.Join(tmp, "cli", "flat")
	order := filepath.Join(tmp, "cli", "order") + string(filepath.Separator)
	up := filepath.Join(tmp, "cli", "up", "sub") + string(filepath.Separator) + ".."
	var stdout, stderr bytes.Buffer
	code := run([]string{"gen", "-q", "-schema", flatFixture, "-out", out, "-schema", flatFixture, "-out", order, "-schema", flatFixture, "-out", up}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("gen = %d, want %d (stderr %q)", code, exitOK, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("gen stdout = %q, stderr = %q, want both empty", stdout.String(), stderr.String())
	}

	schema, err := parser.Parse(flat, parser.WithResolver(loader.Dir(filesystemRoot(flat))))
	if err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(tmp, "lib")
	if err := codegen.Generate([]codegen.Target{
		{Schema: schema, Dir: filepath.Join(lib, "flat"), Package: "flat"},
		{Schema: schema, Dir: filepath.Join(lib, "order"), Package: "order"},
		{Schema: schema, Dir: filepath.Join(lib, "up"), Package: "up"},
	}); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"flat", "order", "up"} {
		got := generatedFile(t, filepath.Join(tmp, "cli", dir))
		want := generatedFile(t, filepath.Join(lib, dir))
		if !bytes.Equal(got, want) {
			t.Errorf("gen wrote for -out %s:\n%s\nwant what Generate writes:\n%s", dir, got, want)
		}
		if !bytes.Contains(got, []byte("\npackage "+dir+"\n")) {
			t.Errorf("gen wrote for -out %s no package %s clause:\n%s", dir, dir, got)
		}
		// The usage block's claim that the comments name absolute locations.
		if !bytes.Contains(got, []byte(" "+flat+":")) {
			t.Errorf("gen wrote for -out %s no comment naming %s:\n%s", dir, flat, got)
		}
	}
}

// TestGenRejectedSchemaWritesNothing pins exit 1: a rejected -schema prints its
// first error as parse does, and no pair is generated, the compiling pair
// before it included.
func TestGenRejectedSchemaWritesNothing(t *testing.T) {
	broken := absFixture(t, "testdata/broken.xsd")
	tmp := t.TempDir()
	first, second := filepath.Join(tmp, "first"), filepath.Join(tmp, "second")
	var stdout, stderr bytes.Buffer
	code := run([]string{"gen", "-schema", flatFixture, "-out", first, "-schema", "testdata/broken.xsd", "-out", second}, &stdout, &stderr)
	if code != exitInvalid {
		t.Fatalf("gen = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("gen stdout = %q, want empty", stdout.String())
	}
	line := stderr.String()
	if !strings.HasPrefix(line, broken+":") || !strings.Contains(line, ": [src-resolve] ") || strings.Count(line, "\n") != 1 {
		t.Errorf("gen stderr = %q, want one line %s:<line>:<col>: [src-resolve] <message>", line, broken)
	}
	assertAbsent(t, first)
	assertAbsent(t, second)
}

// TestGenReportsEveryPair pins that every pair is compiled and reported before
// the run answers: an unreadable -schema does not hide the next pair's
// rejection, and the run answers with the higher-ranked of the two, which is
// the first.
func TestGenReportsEveryPair(t *testing.T) {
	tmp := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := run([]string{"gen", "-schema", "testdata/nosuch.xsd", "-out", filepath.Join(tmp, "a"), "-schema", "testdata/broken.xsd", "-out", filepath.Join(tmp, "b")}, &stdout, &stderr)
	if code != exitUsage {
		t.Fatalf("gen = %d, want %d (stderr %q)", code, exitUsage, stderr.String())
	}
	for _, want := range []string{"[src-resolve]", "goxsd8: gen: open testdata/nosuch.xsd: no such file or directory"} {
		if !strings.Contains(stderr.String(), want) {
			t.Errorf("gen stderr = %q, want it to contain %q", stderr.String(), want)
		}
	}
}

// TestGenNamesAnUnresolvedDirective pins that gen names a schemaLocation
// resolving to no document as parse does, and still generates.
func TestGenNamesAnUnresolvedDirective(t *testing.T) {
	abs, err := filepath.Abs(shortSchema)
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "short")
	var stdout, stderr bytes.Buffer
	code := run([]string{"gen", "-schema", shortSchema, "-out", out}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("gen = %d, want %d (stderr %q)", code, exitOK, stderr.String())
	}
	want := "goxsd8: gen: " + abs + ":10:3: this schemaLocation resolved to no document"
	if !strings.HasPrefix(stderr.String(), want) {
		t.Errorf("gen stderr = %q, want it to open %q", stderr.String(), want)
	}
	generatedFile(t, out)
}

// TestGenVerboseLogs pins -v: the parser's assembly is logged at debug level
// on stderr, and the run still generates.
func TestGenVerboseLogs(t *testing.T) {
	out := filepath.Join(t.TempDir(), "flat")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"gen", "-v", "-schema", flatFixture, "-out", out}, &stdout, &stderr); code != exitOK {
		t.Fatalf("gen -v = %d, want %d (stderr %q)", code, exitOK, stderr.String())
	}
	if !strings.Contains(stderr.String(), "level=DEBUG") {
		t.Errorf("gen -v stderr = %q, want debug log lines", stderr.String())
	}
	generatedFile(t, out)
}

// TestGenUsageAndIOFaults pins exit 2: each row's stderr is exactly its line
// and helpPointer, and nothing is written at the paths it names. Every row
// runs in a fresh working directory, so the -out . row cannot write into the
// package.
func TestGenUsageAndIOFaults(t *testing.T) {
	flat := absFixture(t, flatFixture)
	type row struct {
		name   string
		args   []string
		want   string
		absent []string
	}
	rows := func(tmp string) []row {
		at := func(p string) string { return filepath.Join(tmp, p) }
		return []row{
			{"no pair", []string{}, "goxsd8: gen: no -schema/-out pair given", nil},
			{"-schema without -out", []string{"-schema", flat},
				"goxsd8: gen: -schema " + flat + " stands without its -out", nil},
			{"-schema followed by -schema", []string{"-schema", flat, "-schema", flat, "-out", at("d")},
				"goxsd8: gen: -schema " + flat + " stands without its -out", []string{at("d")}},
			{"-out without -schema", []string{"-out", at("d")},
				"goxsd8: gen: -out " + at("d") + " stands without a -schema before it", []string{at("d")}},
			{"second -out", []string{"-schema", flat, "-out", at("d"), "-out", at("e")},
				"goxsd8: gen: -out " + at("e") + " stands without a -schema before it", []string{at("d"), at("e")}},
			{"unreadable -schema", []string{"-schema", "nosuch.xsd", "-out", at("d")},
				"goxsd8: gen: open nosuch.xsd: no such file or directory", []string{at("d")}},
			{"-backend", []string{"-backend", "strict", "-schema", flat, "-out", at("d")},
				"goxsd8: gen: flag provided but not defined: -backend", []string{at("d")}},
			{"positional argument", []string{"-schema", flat, "-out", at("d"), "extra.xsd"},
				"goxsd8: gen: extra.xsd: gen takes no positional argument, only -schema/-out pairs", []string{at("d")}},
			{"flag after a positional", []string{"extra.xsd", "-out", at("d")},
				fmt.Sprintf(flagAfterPositionalFmt, "gen", "-out"), []string{at("d")}},
			{"valued help", []string{"-help=true"}, fmt.Sprintf(helpNotAFlagValueFmt, "gen"), nil},
			{"package not an identifier", []string{"-schema", flat, "-out", at("my-pkg")},
				"goxsd8: gen: -schema " + flat + " -out " + at("my-pkg") + `: the package name "my-pkg" is not a Go identifier`,
				[]string{at("my-pkg")}},
			{"package a keyword", []string{"-schema", flat, "-out", at("type")},
				"goxsd8: gen: -schema " + flat + " -out " + at("type") + `: the package name "type" is a Go keyword`,
				[]string{at("type")}},
			{"package blank", []string{"-schema", flat, "-out", at("_")},
				"goxsd8: gen: -schema " + flat + " -out " + at("_") + `: the package name "_" is the blank identifier`,
				[]string{at("_")}},
			{"empty -out", []string{"-schema", flat, "-out", ""},
				"goxsd8: gen: -schema " + flat + " -out : no output directory given",
				[]string{at("xsd_gen.go")}},
			{"package main", []string{"-schema", flat, "-out", at("main")},
				"goxsd8: gen: -schema " + flat + " -out " + at("main") + `: the package name is "main", but the emitted package declares no func main and would not build`,
				[]string{at("main")}},
			{"-out .", []string{"-schema", flat, "-out", "."},
				"goxsd8: gen: -schema " + flat + ` -out .: the package name "." is not a Go identifier`,
				[]string{at("xsd_gen.go")}},
			// Phase 1 fails on the second pair, so the first is not written.
			{"duplicate -out", []string{"-schema", flat, "-out", at("d"), "-schema", flat, "-out", at("d") + "/"},
				"goxsd8: gen: -schema " + flat + " -out " + at("d") + "/: the output directory " + at("d") + "/ is also the output directory of -schema " + flat + " -out " + at("d"),
				[]string{at("d")}},
			{"-out a regular file", []string{"-schema", flat, "-out", at("ok"), "-schema", flat, "-out", at("file")},
				"goxsd8: gen: -schema " + flat + " -out " + at("file") + ": the output directory " + at("file") + " exists and is not a directory",
				[]string{at("ok")}},
			{"existing file not generated", []string{"-schema", flat, "-out", at("ok"), "-schema", flat, "-out", at("keep")},
				"goxsd8: gen: -schema " + flat + " -out " + at("keep") + ": " + filepath.Join(at("keep"), "xsd_gen.go") + " exists and its first line does not mark it generated, so it is never overwritten",
				[]string{at("ok")}},
			// Phase 2: the dangling symlink passes every check and fails
			// creating the directory, after the first pair was written.
			{"write fault", []string{"-schema", flat, "-out", at("written"), "-schema", flat, "-out", at("dangling")},
				"goxsd8: gen: -schema " + flat + " -out " + at("dangling") + ": creating " + at("dangling") + ": mkdir " + at("dangling") + ": file exists",
				nil},
		}
	}
	probe := rows("")
	for i := range probe {
		t.Run(probe[i].name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Chdir(tmp)
			if err := os.WriteFile(filepath.Join(tmp, "file"), []byte("not a directory\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(filepath.Join(tmp, "keep"), 0o755); err != nil {
				t.Fatal(err)
			}
			kept := []byte("package keep\n")
			if err := os.WriteFile(filepath.Join(tmp, "keep", "xsd_gen.go"), kept, 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(tmp, "nowhere"), filepath.Join(tmp, "dangling")); err != nil {
				t.Fatal(err)
			}
			r := rows(tmp)[i]
			var stdout, stderr bytes.Buffer
			code := run(append([]string{"gen"}, r.args...), &stdout, &stderr)
			if code != exitUsage {
				t.Errorf("gen %q = %d, want %d", r.args, code, exitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("gen %q stdout = %q, want empty", r.args, stdout.String())
			}
			if want := r.want + "\n" + helpPointer + "\n"; stderr.String() != want {
				t.Errorf("gen %q stderr = %q, want %q", r.args, stderr.String(), want)
			}
			for _, p := range r.absent {
				assertAbsent(t, p)
			}
			if got, err := os.ReadFile(filepath.Join(tmp, "keep", "xsd_gen.go")); err != nil || !bytes.Equal(got, kept) {
				t.Errorf("keep/xsd_gen.go = %q (%v), want it untouched", got, err)
			}
			if r.name == "write fault" {
				generatedFile(t, filepath.Join(tmp, "written"))
			}
		})
	}
}

// TestGenDeclines pins exit 4: a construct outside codegen's slice prints one
// line on stderr naming the pair, the construct and its absolute location, with
// no helpPointer under it, and nothing is written. The body is
// codegen/decline_test.go's all row.
func TestGenDeclines(t *testing.T) {
	tmp := t.TempDir()
	schema := filepath.Join(tmp, "s.xsd")
	doc := "<xs:schema xmlns:xs=\"http://www.w3.org/2001/XMLSchema\">\n" +
		`<xs:element name="e"><xs:complexType><xs:all><xs:element name="a" type="xs:string"/></xs:all></xs:complexType></xs:element>` +
		"\n</xs:schema>\n"
	if err := os.WriteFile(schema, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(tmp, "e")
	var stdout, stderr bytes.Buffer
	code := run([]string{"gen", "-schema", schema, "-out", out}, &stdout, &stderr)
	if code != exitUndecided {
		t.Errorf("gen = %d, want %d", code, exitUndecided)
	}
	if stdout.Len() != 0 {
		t.Errorf("gen stdout = %q, want empty", stdout.String())
	}
	want := "goxsd8: gen: -schema " + schema + " -out " + out + ": complex type (anonymous) of element e with an all compositor at " + schema + ":2:22 is not supported by codegen, a generator limitation, not a schema fault\n"
	if stderr.String() != want {
		t.Errorf("gen stderr = %q, want %q", stderr.String(), want)
	}
	assertAbsent(t, out)
}
