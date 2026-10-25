package codegen

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	acceptedDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e" type="xs:string"/></xs:schema>`
	declinedDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e" type="xs:decimal"/></xs:schema>`
)

// TestGenerateInvalidTargets pins the Target checks: each fails phase 1, so
// the valid target before it is not written either.
func TestGenerateInvalidTargets(t *testing.T) {
	schema := parseString(t, acceptedDoc)
	tests := []struct {
		name   string
		target func(dir string) Target
		want   string
	}{
		{"nil schema", func(dir string) Target { return Target{Dir: filepath.Join(dir, "b"), Package: "p"} },
			"codegen: target 1: the Schema field is nil"},
		{"empty dir", func(string) Target { return Target{Schema: schema, Package: "p"} },
			"codegen: target 1: no output directory given"},
		{"keyword package", func(dir string) Target { return Target{Schema: schema, Dir: filepath.Join(dir, "b"), Package: "type"} },
			`codegen: target 1: the package name "type" is a Go keyword`},
		{"blank package", func(dir string) Target { return Target{Schema: schema, Dir: filepath.Join(dir, "b"), Package: "_"} },
			`codegen: target 1: the package name "_" is the blank identifier`},
		{"non-identifier package", func(dir string) Target { return Target{Schema: schema, Dir: filepath.Join(dir, "b"), Package: "1p"} },
			`codegen: target 1: the package name "1p" is not a Go identifier`},
		{"main package", func(dir string) Target { return Target{Schema: schema, Dir: filepath.Join(dir, "b"), Package: "main"} },
			`codegen: target 1: the package name is "main", but the emitted package declares no func main`},
		{"duplicate dir", func(dir string) Target {
			return Target{Schema: schema, Dir: filepath.Join(dir, "a") + "/./", Package: "q"}
		},
			"codegen: target 1: the output directory %DIR%/a/./ is also the output directory of target 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			first := Target{Schema: schema, Dir: filepath.Join(dir, "a"), Package: "p"}
			err := Generate([]Target{first, tt.target(dir)})
			want := strings.ReplaceAll(tt.want, "%DIR%", dir)
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Fatalf("Generate = %v, want an error starting %q", err, want)
			}
			if errors.Is(err, errors.ErrUnsupported) {
				t.Errorf("an invalid Target wraps errors.ErrUnsupported: %v", err)
			}
			assertAbsent(t, first.Dir)
		})
	}
}

// TestGenerateUnresolvableDir pins that a relative Dir that cannot be made
// absolute, because the working directory was removed, fails phase 1 as a
// plain error, not a decline, and writes no target.
func TestGenerateUnresolvableDir(t *testing.T) {
	schema := parseString(t, acceptedDoc)
	dir := t.TempDir()
	gone := filepath.Join(dir, "gone")
	if err := os.Mkdir(gone, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(gone)
	if err := os.Remove(gone); err != nil {
		t.Fatal(err)
	}
	first := Target{Schema: schema, Dir: filepath.Join(dir, "a"), Package: "p"}
	err := Generate([]Target{first, {Schema: schema, Dir: "b", Package: "p"}})
	want := "codegen: target 1: resolving the output directory b: "
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("Generate = %v, want an error starting %q", err, want)
	}
	if errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("an unresolvable Dir wraps errors.ErrUnsupported: %v", err)
	}
	assertAbsent(t, first.Dir)
}

// TestGenerateNamedTargets pins that a named target's error opens with its
// Name and no "codegen: " lead-in, and that the duplicate-Dir refusal names
// the earlier target by its Name, or by "target N" when it has none.
func TestGenerateNamedTargets(t *testing.T) {
	accepted := parseString(t, acceptedDoc)
	declined := parseString(t, declinedDoc)
	tests := []struct {
		name    string
		targets func(dir string) []Target
		want    string
		decline bool
	}{
		{"phase-1 refusal", func(dir string) []Target {
			return []Target{
				{Schema: accepted, Dir: filepath.Join(dir, "a"), Package: "p", Name: "first"},
				{Schema: accepted, Dir: filepath.Join(dir, "b"), Package: "type", Name: "second"},
			}
		}, `second: the package name "type" is a Go keyword`, false},
		{"decline", func(dir string) []Target {
			return []Target{
				{Schema: accepted, Dir: filepath.Join(dir, "a"), Package: "p", Name: "first"},
				{Schema: declined, Dir: filepath.Join(dir, "b"), Package: "q", Name: "second"},
			}
		}, "second: element e with {type definition}", true},
		{"duplicate dir", func(dir string) []Target {
			return []Target{
				{Schema: accepted, Dir: filepath.Join(dir, "a"), Package: "p", Name: "first"},
				{Schema: accepted, Dir: filepath.Join(dir, "a") + "/./", Package: "q", Name: "second"},
			}
		}, "second: the output directory %DIR%/a/./ is also the output directory of first", false},
		{"duplicate of an unnamed dir", func(dir string) []Target {
			return []Target{
				{Schema: accepted, Dir: filepath.Join(dir, "a"), Package: "p", Name: "first"},
				{Schema: accepted, Dir: filepath.Join(dir, "b"), Package: "q"},
				{Schema: accepted, Dir: filepath.Join(dir, "b") + "/./", Package: "r", Name: "third"},
			}
		}, "third: the output directory %DIR%/b/./ is also the output directory of target 1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			targets := tt.targets(dir)
			err := Generate(targets)
			want := strings.ReplaceAll(tt.want, "%DIR%", dir)
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Fatalf("Generate = %v, want an error starting %q", err, want)
			}
			if !tt.decline && err.Error() != want {
				t.Errorf("Generate = %q, want exactly %q", err, want)
			}
			if got := errors.Is(err, errors.ErrUnsupported); got != tt.decline {
				t.Errorf("errors.Is(err, errors.ErrUnsupported) = %v, want %v", got, tt.decline)
			}
			if strings.Contains(err.Error(), "codegen: ") {
				t.Errorf("a named target's error carries the codegen: lead-in: %v", err)
			}
			assertAbsent(t, targets[0].Dir)
		})
	}
}

// TestGenerateAllOrNothing pins that a decline in a later target leaves an
// earlier, accepted one unwritten, and that the error names the later one.
func TestGenerateAllOrNothing(t *testing.T) {
	dir := t.TempDir()
	targets := []Target{
		{Schema: parseString(t, acceptedDoc), Dir: filepath.Join(dir, "a"), Package: "a"},
		{Schema: parseString(t, declinedDoc), Dir: filepath.Join(dir, "b"), Package: "b"},
	}
	err := Generate(targets)
	if err == nil || !strings.HasPrefix(err.Error(), "codegen: target 1: element e with {type definition}") || !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("Generate = %v, want target 1's decline", err)
	}
	assertAbsent(t, targets[0].Dir)
}

// TestGenerateDestination pins the destination rules: Dir and its parents are
// created, a generated file is overwritten, any other file in Dir is left
// alone, and a destination that is not a generated file fails phase 1.
func TestGenerateDestination(t *testing.T) {
	schema := parseString(t, acceptedDoc)
	t.Run("creates parents and overwrites a generated file", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "x", "y")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		other := filepath.Join(dir, "keep.go")
		write(t, other, "package p\n")
		write(t, filepath.Join(dir, fileName), "// Code generated by hand. DO NOT EDIT.\nstale\n")
		if err := Generate([]Target{{Schema: schema, Dir: dir, Package: "p"}}); err != nil {
			t.Fatal(err)
		}
		if got := read(t, filepath.Join(dir, fileName)); !strings.Contains(got, "type E string") {
			t.Errorf("generated file not overwritten:\n%s", got)
		}
		if got := read(t, other); got != "package p\n" {
			t.Errorf("other file changed: %q", got)
		}
		fresh := filepath.Join(t.TempDir(), "m", "n")
		if err := Generate([]Target{{Schema: schema, Dir: fresh, Package: "p"}}); err != nil {
			t.Fatal(err)
		}
		read(t, filepath.Join(fresh, fileName))
	})
	t.Run("refuses a hand-written file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, fileName)
		write(t, path, "package p\n// Code generated by hand. DO NOT EDIT.\n")
		err := Generate([]Target{{Schema: schema, Dir: dir, Package: "p"}})
		if err == nil || !strings.Contains(err.Error(), "does not mark it generated") {
			t.Fatalf("Generate = %v, want a refusal", err)
		}
		if got := read(t, path); got != "package p\n// Code generated by hand. DO NOT EDIT.\n" {
			t.Errorf("hand-written file changed: %q", got)
		}
	})
	t.Run("refuses a Dir that is a file", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "f")
		write(t, dir, "")
		err := Generate([]Target{{Schema: schema, Dir: dir, Package: "p"}})
		if err == nil || !strings.Contains(err.Error(), "is not a directory") {
			t.Fatalf("Generate = %v, want a refusal", err)
		}
	})
}

// TestGenerateVacuousSequence pins that a sequence whose children all have
// maxOccurs 0 is accepted as an empty struct: an <element> with
// minOccurs=maxOccurs=0 maps to no component at all (§3.3.2), and
// dcl.ctd.ctcc.common clause 2.1 makes the explicit content empty only for
// a group with no children or a group whose own maxOccurs is 0, so the
// {content type} is element-only.
func TestGenerateVacuousSequence(t *testing.T) {
	schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" minOccurs="0" maxOccurs="0" type="xs:int"/></xs:sequence></xs:complexType></xs:element></xs:schema>`)
	dir := t.TempDir()
	if err := Generate([]Target{{Schema: schema, Dir: dir, Package: "p"}}); err != nil {
		t.Fatal(err)
	}
	if got := read(t, filepath.Join(dir, fileName)); !strings.Contains(got, "type E struct {\n}\n") {
		t.Errorf("want an empty struct E:\n%s", got)
	}
}

func assertAbsent(t *testing.T, dir string) {
	t.Helper()
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s exists after a failed Generate (stat: %v)", dir, err)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
