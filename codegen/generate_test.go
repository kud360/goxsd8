package codegen

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	acceptedDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e" type="xs:string"/></xs:schema>`
	declinedDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e" type="xs:QName"/></xs:schema>`
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

// TestGenerateRefusalPrecedesDecline pins that a target's own refusal is the
// error Generate returns even when its Schema also holds a construct the
// generator declines: a plain error naming the destination, or the Dir that
// cannot be checked, not a decline.
func TestGenerateRefusalPrecedesDecline(t *testing.T) {
	declined := parseString(t, declinedDoc)
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string) string
		want  string
	}{
		{"hand-written xsd_gen.go", func(t *testing.T, dir string) string {
			if err := os.Mkdir(filepath.Join(dir, "q"), 0o755); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(dir, "q", fileName), "package q\n")
			return filepath.Join(dir, "q")
		}, "codegen: target 0: %DIR%/q/xsd_gen.go exists and its first line does not mark it generated, so it is never overwritten"},
		{"xsd_gen.go a directory", func(t *testing.T, dir string) string {
			if err := os.MkdirAll(filepath.Join(dir, "q", fileName), 0o755); err != nil {
				t.Fatal(err)
			}
			return filepath.Join(dir, "q")
		}, "codegen: target 0: %DIR%/q/xsd_gen.go exists and is not a regular file"},
		{"Dir under a file", func(t *testing.T, dir string) string {
			write(t, filepath.Join(dir, "f"), "")
			return filepath.Join(dir, "f", "q")
		}, "codegen: target 0: checking %DIR%/f/q: stat %DIR%/f/q: not a directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			err := Generate([]Target{{Schema: declined, Dir: tt.setup(t, dir), Package: "q"}})
			want := strings.ReplaceAll(tt.want, "%DIR%", dir)
			if err == nil || err.Error() != want {
				t.Fatalf("Generate = %v, want %q", err, want)
			}
			if errors.Is(err, errors.ErrUnsupported) {
				t.Errorf("the refusal wraps errors.ErrUnsupported: %v", err)
			}
		})
	}
}

// TestCheckTargets pins CheckTargets: with every Schema nil it reports each
// refused target once, in slice order, opened with the target's name as
// Generate's errors are, checks a Dir against an earlier refused target's too,
// writes nothing, and returns nil when no target is refused.
func TestCheckTargets(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "keep", fileName), "package keep\n")
	write(t, filepath.Join(dir, "file"), "")
	targets := []Target{
		{Dir: filepath.Join(dir, "a"), Package: "a"},
		{Dir: filepath.Join(dir, "b"), Package: "type"},
		{Dir: filepath.Join(dir, "b") + "/", Package: "b", Name: "dup"},
		{Dir: filepath.Join(dir, "keep"), Package: "keep"},
		{Dir: filepath.Join(dir, "file"), Package: "file", Name: "file"},
		{Dir: "", Package: "main"},
		{Dir: filepath.Join(dir, "c"), Package: "c"},
	}
	want := []string{
		`codegen: target 1: the package name "type" is a Go keyword`,
		"dup: the output directory %DIR%/b/ is also the output directory of target 1",
		"codegen: target 3: %DIR%/keep/xsd_gen.go exists and its first line does not mark it generated, so it is never overwritten",
		"file: the output directory %DIR%/file exists and is not a directory",
		"codegen: target 5: no output directory given",
	}
	for i := range want {
		want[i] = strings.ReplaceAll(want[i], "%DIR%", dir)
	}
	errs := CheckTargets(targets)
	got := make([]string, len(errs))
	for i, err := range errs {
		got[i] = err.Error()
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("CheckTargets =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	assertAbsent(t, targets[0].Dir)
	assertAbsent(t, targets[6].Dir)
	if errs := CheckTargets([]Target{targets[0], targets[6]}); errs != nil {
		t.Errorf("CheckTargets of accepted targets = %v, want nil", errs)
	}
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
	if got := read(t, filepath.Join(dir, fileName)); !strings.Contains(got, "type E struct{}\n") {
		t.Errorf("want an empty struct E:\n%s", got)
	}
}

// TestGenerateEmptyContent pins the forms whose {content type} has {variety}
// empty (§3.4.1) beyond the golden's, each emitted as the package doc's
// "# Coverage" states. Explicit content is empty for no model group child, an
// all or sequence with no children, a choice with minOccurs 0 and no
// children, or a model group with maxOccurs 0 (§3.4.2.3.3 dcl.ctd.ctcc.common
// clause 2.1); {variety} is then empty when effective mixed is false (clause
// 4.1.1), and a <defaultOpenContent> without appliesToEmpty is no wildcard
// element for it (clause 5.2), so clause 6.1 keeps that empty content type.
func TestGenerateEmptyContent(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"empty all", `<xs:element name="e"><xs:complexType><xs:all/></xs:complexType></xs:element>`,
			"type E struct{}\n"},
		{"empty sequence", `<xs:element name="e"><xs:complexType><xs:sequence/></xs:complexType></xs:element>`,
			"type E struct{}\n"},
		{"optional empty choice", `<xs:element name="e"><xs:complexType><xs:choice minOccurs="0"/></xs:complexType></xs:element>`,
			"type E struct{}\n"},
		{"maxOccurs 0 sequence", `<xs:element name="e"><xs:complexType><xs:sequence minOccurs="0" maxOccurs="0"><xs:element name="a" type="xs:string"/></xs:sequence></xs:complexType></xs:element>`,
			"type E struct{}\n"},
		{"explicit restriction of anyType", `<xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="xs:anyType"><xs:attribute name="a" type="xs:string"/></xs:restriction></xs:complexContent></xs:complexType></xs:element>`,
			"type E struct {\n\tA *string\n}\n"},
		{"named type", `<xs:complexType name="c"><xs:attribute name="a" type="xs:boolean" use="required"/></xs:complexType><xs:element name="e" type="c"/>`,
			"type E struct {\n\tA bool\n}\n"},
		{"default open content", `<xs:defaultOpenContent><xs:any/></xs:defaultOpenContent><xs:element name="e"><xs:complexType/></xs:element>`,
			"type E struct{}\n"},
		{"choice alternative", `<xs:element name="e"><xs:complexType><xs:choice><xs:element name="a"><xs:complexType/></xs:element><xs:element name="b" type="xs:string"/></xs:choice></xs:complexType></xs:element>`,
			"type EA struct {\n\tA EA2\n}\n\nfunc (EA) isE() {}\n\n// EA2 is generated from element a declared at s.xsd:1:"},
		{"repeated local element", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="a" maxOccurs="unbounded"><xs:complexType/></xs:element></xs:sequence></xs:complexType></xs:element>`,
			"type E struct {\n\tA []EA\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+tt.body+`</xs:schema>`)
			dir := t.TempDir()
			if err := Generate([]Target{{Schema: schema, Dir: dir, Package: "p"}}); err != nil {
				t.Fatal(err)
			}
			if got := read(t, filepath.Join(dir, fileName)); !strings.Contains(got, tt.want) {
				t.Errorf("want %q in:\n%s", tt.want, got)
			}
		})
	}
}

// TestGenerateElementRefs pins <element ref> particles beyond the golden's:
// each field holds the type emitted for the top-level declaration the ref
// names, and the emitted package type-checks. The recursion rows close a
// cycle through refs that is not a struct-value recursion, since one ref is
// a slice or a pointer field; the symbol-space row has an element and a
// complex type share the expanded name n, which no cycle passes through twice.
func TestGenerateElementRefs(t *testing.T) {
	const str = `type="xs:string"`
	tests := []struct{ name, body, want string }{
		{"sequence", `<xs:element name="h" ` + str + `/><xs:element name="e"><xs:complexType><xs:sequence><xs:element ref="h"/></xs:sequence></xs:complexType></xs:element>`,
			"type E struct {\n\tH H\n}\n"},
		{"choice", `<xs:element name="h" ` + str + `/><xs:element name="e"><xs:complexType><xs:choice><xs:element name="a" ` + str + `/><xs:element ref="h"/></xs:choice></xs:complexType></xs:element>`,
			"type EH struct {\n\tH H\n}\n\nfunc (EH) isE() {}\n"},
		{"recursion through a repeated ref", `<xs:element name="a"><xs:complexType><xs:sequence><xs:element ref="b" maxOccurs="unbounded"/></xs:sequence></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:sequence><xs:element ref="a"/></xs:sequence></xs:complexType></xs:element>`,
			"type A struct {\n\tB []B\n}\n"},
		{"recursion through an optional ref", `<xs:element name="a"><xs:complexType><xs:sequence><xs:element ref="b" minOccurs="0"/></xs:sequence></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:sequence><xs:element ref="a"/></xs:sequence></xs:complexType></xs:element>`,
			"type A struct {\n\tB *B\n}\n"},
		{"element and complex type of one name", `<xs:element name="n" type="t"/><xs:complexType name="t"><xs:sequence><xs:element name="m" type="n"/></xs:sequence></xs:complexType><xs:complexType name="n"><xs:sequence><xs:element ref="o"/></xs:sequence></xs:complexType><xs:element name="o" ` + str + `/>`,
			"type N struct {\n\tM N2\n}\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+tt.body+`</xs:schema>`)
			src := generateOne(t, Target{Schema: schema, Dir: t.TempDir(), Package: "p"})
			if got := string(src); !strings.Contains(got, tt.want) {
				t.Errorf("want %q in:\n%s", tt.want, got)
			}
			typeCheck(t, src)
		})
	}
}

// TestGenerateDiamond pins that a struct-value recursion check costs time
// linear in the named types it reaches: each complex type d<i> holds two value
// fields of type d<i+1>, so a check that walks a type again on every reach
// doubles per level and does not return. In "rewalked", the value field v
// reaches p while p's body is being emitted, so the check walks p, and through
// it the whole diamond, before the emitting walk reaches d0.
func TestGenerateDiamond(t *testing.T) {
	const depth = 40
	var diamond strings.Builder
	for i := range depth {
		fmt.Fprintf(&diamond, `<xs:complexType name="d%d"><xs:sequence><xs:element name="a" type="d%d"/><xs:element name="b" type="d%d"/></xs:sequence></xs:complexType>`, i, i+1, i+1)
	}
	fmt.Fprintf(&diamond, `<xs:complexType name="d%d"><xs:sequence><xs:element name="s" type="xs:string"/></xs:sequence></xs:complexType>`, depth)
	tests := []struct{ name, body string }{
		{"emitted", `<xs:element name="e"><xs:complexType><xs:sequence><xs:element name="x" type="d0"/></xs:sequence></xs:complexType></xs:element>`},
		{"rewalked", `<xs:complexType name="p"><xs:sequence><xs:element name="w" type="w" minOccurs="0"/><xs:element name="d" type="d0"/></xs:sequence></xs:complexType><xs:complexType name="w"><xs:sequence><xs:element name="v" type="p"/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="x" type="p"/></xs:sequence></xs:complexType></xs:element>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+tt.body+diamond.String()+`</xs:schema>`)
			dir := t.TempDir()
			if err := Generate([]Target{{Schema: schema, Dir: dir, Package: "p"}}); err != nil {
				t.Fatal(err)
			}
			if got, want := read(t, filepath.Join(dir, fileName)), fmt.Sprintf("type D%d struct {\n\tA D%d\n\tB D%d\n}\n", depth-1, depth, depth); !strings.Contains(got, want) {
				t.Errorf("want %q in:\n%s", want, got)
			}
		})
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
