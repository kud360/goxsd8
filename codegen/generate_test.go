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
	declinedDoc = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e" type="xs:precisionDecimal"/></xs:schema>`
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

// TestTargetError pins that every element of Generate's error, in phase 1 and
// phase 2, and every element of CheckTargets' result is a *TargetError whose
// Index is the failing target's slice index and whose Unwrap is the failure
// its opening precedes, a decline's *DeclineError reached through it.
func TestTargetError(t *testing.T) {
	accepted := parseString(t, acceptedDoc)
	declined := parseString(t, declinedDoc)
	tests := []struct {
		name    string
		errs    func(dir string) []error
		cause   string
		decline bool
	}{
		{"phase-1 refusal", func(dir string) []error {
			return elements(t, Generate([]Target{
				{Schema: accepted, Dir: filepath.Join(dir, "a"), Package: "p"},
				{Schema: accepted, Dir: filepath.Join(dir, "b"), Package: "type"},
			}))
		}, `the package name "type"`, false},
		{"decline", func(dir string) []error {
			return elements(t, Generate([]Target{
				{Schema: accepted, Dir: filepath.Join(dir, "a"), Package: "p"},
				{Schema: declined, Dir: filepath.Join(dir, "b"), Package: "q"},
			}))
		}, "element e with {type definition}", true},
		{"phase-2 fault", func(dir string) []error {
			return elements(t, Generate([]Target{
				{Schema: accepted, Dir: filepath.Join(dir, "a"), Package: "p"},
				{Schema: accepted, Dir: filepath.Join(dir, "a", fileName, "sub"), Package: "q"},
			}))
		}, "creating ", false},
		{"CheckTargets refusal", func(dir string) []error {
			return CheckTargets([]Target{
				{Dir: filepath.Join(dir, "a"), Package: "p"},
				{Dir: filepath.Join(dir, "b"), Package: "type"},
			})
		}, `the package name "type"`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.errs(t.TempDir())
			if len(errs) != 1 {
				t.Fatalf("got %d errors %q, want 1", len(errs), errs)
			}
			var te *TargetError
			if !errors.As(errs[0], &te) {
				t.Fatalf("errors.As(%v, *TargetError) = false", errs[0])
			}
			if te.Index() != 1 {
				t.Errorf("Index() = %d, want 1", te.Index())
			}
			if te.Unwrap() == nil || !strings.HasPrefix(te.Unwrap().Error(), tt.cause) || te.Error() != "codegen: target 1: "+te.Unwrap().Error() {
				t.Errorf("Unwrap() = %v, want the failure starting %q that %q opens", te.Unwrap(), tt.cause, te.Error())
			}
			var de *DeclineError
			if errors.As(errs[0], &de) != tt.decline {
				t.Fatalf("errors.As(%v, *DeclineError) = %t, want %t", errs[0], !tt.decline, tt.decline)
			}
			if tt.decline && de.Construct() != (Construct{constructUncoveredPrimitive}) {
				t.Errorf("Construct() = %v, want %v", de.Construct(), Construct{constructUncoveredPrimitive})
			}
			if errors.Is(errs[0], errors.ErrUnsupported) != tt.decline {
				t.Errorf("errors.Is(%v, errors.ErrUnsupported) = %t, want %t", errs[0], !tt.decline, tt.decline)
			}
		})
	}
}

// elements is the errors.Join elements of Generate's error err.
func elements(t *testing.T, err error) []error {
	t.Helper()
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("Generate = %v, want an errors.Join", err)
	}
	return joined.Unwrap()
}

// TestCheckTargetsEmptyEarlierDir pins that an earlier target with an empty
// Dir, which filepath.Clean makes ".", is no duplicate of a later "." Dir,
// while two "." Dirs still are.
func TestCheckTargetsEmptyEarlierDir(t *testing.T) {
	tests := []struct {
		name    string
		targets []Target
		want    string
	}{
		{"empty then dot", []Target{{Dir: "", Package: "p"}, {Dir: ".", Package: "p"}},
			"codegen: target 0: no output directory given"},
		{"dot twice", []Target{{Dir: ".", Package: "p"}, {Dir: ".", Package: "q"}},
			"codegen: target 1: the output directory . is also the output directory of target 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := make([]string, 0, 1)
			for _, err := range CheckTargets(tt.targets) {
				got = append(got, err.Error())
			}
			if len(got) != 1 || got[0] != tt.want {
				t.Errorf("CheckTargets = %q, want [%q]", got, tt.want)
			}
		})
	}
}

// TestGenerateCollectsDeclines pins that Generate's error holds every decline,
// in target order and then document order, each opened with its target's
// name: one per top-level element declaration's turn, a decline met again
// collected once where first met, and a named type whose body declined
// entered again by a later turn, which here meets the same decline and
// contributes nothing.
func TestGenerateCollectsDeclines(t *testing.T) {
	const tail = " is not supported by codegen, a generator limitation, not a schema fault"
	const pdecimal = " with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal"
	uncovered := Construct{constructUncoveredPrimitive}
	nillable := Construct{constructElementNillable}
	type decline struct {
		text      string
		construct Construct
	}
	tests := []struct {
		name    string
		targets func(dir string) []Target
		want    []decline
	}{
		{"two targets, one with two", func(dir string) []Target {
			return []Target{
				{Schema: parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xs:element name="a" type="xs:precisionDecimal"/>
<xs:element name="c"><xs:complexType><xs:sequence><xs:element ref="b"/></xs:sequence></xs:complexType></xs:element>
<xs:element name="b" nillable="true" type="xs:string"/>
</xs:schema>`), Dir: filepath.Join(dir, "a"), Package: "a", Name: "first"},
				{Schema: parseString(t, declinedDoc), Dir: filepath.Join(dir, "b"), Package: "b"},
			}
		}, []decline{
			{"first: element a" + pdecimal + " at s.xsd:2:1" + tail, uncovered},
			// Met at c's turn, through its ref; b's own turn is the dropped duplicate.
			{"first: element b with {nillable} true at s.xsd:4:1" + tail, nillable},
			{"codegen: target 1: element e" + pdecimal + " at s.xsd:1:56" + tail, uncovered},
		}},
		{"a named type two top-level elements share", func(dir string) []Target {
			return []Target{{Schema: parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xs:complexType name="T"><xs:sequence><xs:element name="x" nillable="true" type="xs:string"/></xs:sequence></xs:complexType>
<xs:element name="e1" type="T"/>
<xs:element name="e2" type="T"/>
</xs:schema>`), Dir: filepath.Join(dir, "a"), Package: "a"}}
		}, []decline{
			{"codegen: target 0: element x with {nillable} true at s.xsd:2:39" + tail, nillable},
		}},
		// b's walk enters T again, whose body declined at a's turn, so its first
		// decline is x's duplicate, and m's decline after it is never met.
		{"a named type whose body declined, reached again locally", func(dir string) []Target {
			return []Target{{Schema: parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
<xs:complexType name="T"><xs:sequence><xs:element name="x" nillable="true" type="xs:string"/></xs:sequence></xs:complexType>
<xs:element name="a"><xs:complexType><xs:sequence><xs:element name="l1" type="T"/></xs:sequence></xs:complexType></xs:element>
<xs:element name="b"><xs:complexType><xs:sequence><xs:element name="l2" type="T"/><xs:element name="m" type="xs:precisionDecimal"/></xs:sequence></xs:complexType></xs:element>
</xs:schema>`), Dir: filepath.Join(dir, "a"), Package: "a"}}
		}, []decline{
			{"codegen: target 0: element x with {nillable} true at s.xsd:2:39" + tail, nillable},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets := tt.targets(t.TempDir())
			err := Generate(targets)
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Errorf("errors.Is(%v, errors.ErrUnsupported) = false, want true", err)
			}
			elems := joined(t, err)
			want := make([]string, len(tt.want))
			for i, d := range tt.want {
				want[i] = d.text
			}
			got := make([]string, len(elems))
			for i, e := range elems {
				got[i] = e.Error()
			}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Fatalf("Generate's elements =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			if err.Error() != strings.Join(want, "\n") {
				t.Errorf("Generate = %q, want %q", err.Error(), strings.Join(want, "\n"))
			}
			for i, e := range elems {
				var de *DeclineError
				if !errors.As(e, &de) || de.Construct() != tt.want[i].construct {
					t.Errorf("element %d: errors.As = %v, Construct %v, want %v", i, de != nil, de.Construct(), tt.want[i].construct)
				}
			}
			for _, target := range targets {
				assertAbsent(t, target.Dir)
			}
		})
	}
}

// TestGenerateStopsAtNonDecline pins that phase 1 stops at a failure that is
// not a decline, which follows the declines collected before it: the third
// target's decline is never collected.
func TestGenerateStopsAtNonDecline(t *testing.T) {
	const first = "first: element e with {type definition} {http://www.w3.org/2001/XMLSchema}precisionDecimal, whose {primitive type definition} is {http://www.w3.org/2001/XMLSchema}precisionDecimal at s.xsd:1:56 is not supported by codegen, a generator limitation, not a schema fault"
	declined := parseString(t, declinedDoc)
	tests := []struct {
		name   string
		second Target
		want   string
	}{
		{"refusal", Target{Schema: parseString(t, acceptedDoc), Package: "type", Name: "second"}, `second: the package name "type" is a Go keyword`},
		{"nil Schema", Target{Package: "q", Name: "second"}, "second: the Schema field is nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.second.Dir = filepath.Join(dir, "b")
			targets := []Target{
				{Schema: declined, Dir: filepath.Join(dir, "a"), Package: "p", Name: "first"},
				tt.second,
				{Schema: declined, Dir: filepath.Join(dir, "c"), Package: "r", Name: "third"},
			}
			err := Generate(targets)
			if !errors.Is(err, errors.ErrUnsupported) {
				t.Errorf("errors.Is(%v, errors.ErrUnsupported) = false, want true", err)
			}
			elems := joined(t, err)
			if len(elems) != 2 || elems[0].Error() != first || elems[1].Error() != tt.want {
				t.Errorf("Generate's elements = %q, want [%q %q]", elems, first, tt.want)
			}
			for _, target := range targets {
				assertAbsent(t, target.Dir)
			}
		})
	}
}

// joined returns the elements of the errors.Join Generate returned as err.
func joined(t *testing.T, err error) []error {
	t.Helper()
	j, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("Generate = %v, want an errors.Join", err)
	}
	return j.Unwrap()
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

// TestGenerateRestriction pins complex-content restrictions of a base other
// than xs:anyType beyond the golden's: each gets a type from its own {content
// type} (§3.4.2.3.3 dcl.ctd.ctcc.common) and {attribute uses} (§3.4.2.4
// dcl.ctd.attuses), whatever its base declines for. The base's {abstract}
// (§3.4.2.1 dcl.ctd.common), mixed content (dcl.ctd.ctcc.common 1.3) and
// {attribute wildcard} (§3.4.2.5 dcl.ctd.anyatt 2.1) are none of the
// derived type's, and e inherits b's attribute a.
func TestGenerateRestriction(t *testing.T) {
	const str = `type="xs:string"`
	tests := []struct{ name, body, want string }{
		{"empty", `<xs:complexType name="b"/><xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="b"/></xs:complexContent></xs:complexType></xs:element>`,
			"type E struct{}\n"},
		{"abstract base", `<xs:complexType name="b" abstract="true"><xs:attribute name="a" ` + str + `/></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="b"/></xs:complexContent></xs:complexType></xs:element>`,
			"type E struct {\n\tA *string\n}\n"},
		{"mixed base", `<xs:complexType name="b" mixed="true"><xs:sequence><xs:element name="a" minOccurs="0" ` + str + `/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:element>`,
			"type E struct {\n\tA string\n}\n"},
		{"attribute wildcard base", `<xs:complexType name="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence><xs:anyAttribute/></xs:complexType><xs:element name="e"><xs:complexType><xs:complexContent><xs:restriction base="b"><xs:sequence><xs:element name="a" ` + str + `/></xs:sequence></xs:restriction></xs:complexContent></xs:complexType></xs:element>`,
			"type E struct {\n\tA string\n}\n"},
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

// TestGenerateGroupRefs pins <group ref> particles beyond the golden's. The
// recursion rows reach the local element e again through a <group ref> to the
// group g declaring it, as e's type's content particle and inside an inline
// sequence; e occurs optionally, so the cycle passes through a pointer, and
// the walk ends at e's type, emitted once under g. In the sibling row a's type
// reaches g again only through a pointer, and its value field b, a sibling of
// a in g, closes no cycle. In the symbol-space row a complex type and a model
// group share the expanded name g, and each declares a local element e of an
// anonymous type: the two types are distinct.
func TestGenerateGroupRefs(t *testing.T) {
	tests := []struct{ name, body, want string }{
		{"recursion through content particle", `<xs:group name="g"><xs:sequence><xs:element name="e" minOccurs="0"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element></xs:sequence></xs:group><xs:element name="r"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>`,
			"type R struct {\n\tE *GE\n}\n\n// GE is generated from element e declared at s.xsd:1:88.\ntype GE struct {\n\tE *GE\n}\n"},
		{"recursion through inline sequence", `<xs:group name="g"><xs:sequence><xs:element name="e" minOccurs="0"><xs:complexType><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:group><xs:element name="r"><xs:complexType><xs:sequence><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element>`,
			"type R struct {\n\tE *GE\n}\n\n// GE is generated from element e declared at s.xsd:1:88.\ntype GE struct {\n\tE *GE\n}\n"},
		{"value field to a sibling", `<xs:group name="g"><xs:sequence><xs:element name="a" minOccurs="0"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:sequence><xs:element name="s" type="xs:string"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:group><xs:element name="r"><xs:complexType><xs:group ref="g"/></xs:complexType></xs:element>`,
			"type GA struct {\n\tA *GA\n\tB GB\n}\n"},
		{"complex type and group of one name", `<xs:complexType name="g"><xs:sequence><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="i" type="xs:string"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:complexType><xs:group name="g"><xs:sequence><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="j" type="xs:int"/></xs:sequence></xs:complexType></xs:element></xs:sequence></xs:group><xs:element name="r"><xs:complexType><xs:sequence><xs:element name="x" type="g"/><xs:group ref="g"/></xs:sequence></xs:complexType></xs:element>`,
			"type GE struct {\n\tI string\n}\n\n// GE2 is generated from element e declared at s.xsd:1:"},
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

// TestGenerateWildcardVariants pins that a wildcard particle of a sequence
// gives the same field whatever its {namespace constraint} and {process
// contents} (§3.10.1), neither of which "# Coverage" represents: each row's
// output is byte-identical to that of a lax ##any wildcard in the same schema
// element, whose attributes the row's schema field holds.
func TestGenerateWildcardVariants(t *testing.T) {
	tests := []struct{ name, schema, wildcard string }{
		{"bare", ``, `<xs:any/>`},
		{"skip", ``, `<xs:any processContents="skip"/>`},
		{"other", ``, `<xs:any namespace="##other"/>`},
		{"local and target namespace", ` targetNamespace="urn:t"`, `<xs:any namespace="##local ##targetNamespace"/>`},
		{"notNamespace", ``, `<xs:any notNamespace="urn:x"/>`},
		{"notQName defined and sibling", ``, `<xs:any notQName="##defined ##definedSibling"/>`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen := func(wildcard string) []byte {
				schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`+tt.schema+">\n"+
					`<xs:element name="e"><xs:complexType><xs:sequence>`+wildcard+`</xs:sequence></xs:complexType></xs:element></xs:schema>`)
				return generateOne(t, Target{Schema: schema, Dir: t.TempDir(), Package: "p"})
			}
			want := gen(`<xs:any processContents="lax"/>`)
			if got := gen(tt.wildcard); string(got) != string(want) {
				t.Errorf("output for %s differs from the lax ##any wildcard's:\n%s\n---\n%s", tt.wildcard, got, want)
			}
		})
	}
}

// TestGenerateWildcardName pins that a wildcard particle's field is claimed
// from the local name any in particle order, before every attribute field, so
// that beside a local element any and an attribute any it takes the suffix 2.
func TestGenerateWildcardName(t *testing.T) {
	schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="any" type="xs:string"/><xs:any/></xs:sequence><xs:attribute name="any" type="xs:string"/></xs:complexType></xs:element></xs:schema>`)
	src := generateOne(t, Target{Schema: schema, Dir: t.TempDir(), Package: "p"})
	if got, want := string(src), "type E struct {\n\tAny  string\n\tAny2 AnyElement\n\tAny3 *string\n}\n"; !strings.Contains(got, want) {
		t.Errorf("want %q in:\n%s", want, got)
	}
	typeCheck(t, src)
}

// TestGenerateAttributeWildcardName pins how an {attribute wildcard}'s field
// and the raw-content structs are named ("Naming"): the field is claimed from
// the local name anyAttribute after every attribute field, so that beside an
// attribute anyAttribute it takes the suffix 2; and the raw-content structs are
// claimed at that field, after the struct's own name and its other fields'
// types, so that a top-level element anyType, or a complex type anyType a
// local element reaches, takes AnyType and the raw-content struct AnyType2.
func TestGenerateAttributeWildcardName(t *testing.T) {
	tests := []struct {
		name, body string
		want       []string
	}{
		{"attribute anyAttribute", `<xs:element name="e"><xs:complexType><xs:attribute name="anyAttribute" type="xs:string"/><xs:anyAttribute/></xs:complexType></xs:element>`,
			[]string{"type E struct {\n\tAnyAttribute  *string\n\tAnyAttribute2 []AnyAttribute\n}\n"}},
		{"element anyType", `<xs:element name="anyType"><xs:complexType><xs:anyAttribute/></xs:complexType></xs:element>`,
			[]string{"type AnyType struct {\n\tAnyAttribute []AnyAttribute\n}\n", "type AnyType2 struct {\n\tAttributes []AnyAttribute\n"}},
		{"field type anyType", `<xs:complexType name="anyType"><xs:sequence><xs:element name="s" type="xs:string"/></xs:sequence></xs:complexType><xs:element name="e"><xs:complexType><xs:sequence><xs:element name="x" type="anyType"/></xs:sequence><xs:anyAttribute/></xs:complexType></xs:element>`,
			[]string{"type E struct {\n\tX            AnyType\n\tAnyAttribute []AnyAttribute\n}\n", "type AnyType2 struct {\n\tAttributes []AnyAttribute\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := parseString(t, `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">`+tt.body+`</xs:schema>`)
			src := generateOne(t, Target{Schema: schema, Dir: t.TempDir(), Package: "p"})
			for _, want := range tt.want {
				if got := string(src); !strings.Contains(got, want) {
					t.Errorf("want %q in:\n%s", want, got)
				}
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
// it the whole diamond, before the emitting walk reaches d0. TestGenerateGroupDiamond
// is the same diamond through model groups.
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

// TestGenerateGroupDiamond pins that a struct-value recursion check costs time
// linear in the model groups it reaches: each group g<i> declares two local
// elements a and b, each of an anonymous type whose content is a <group ref>
// to g<i+1>, so b's type reaches the types of g<i+1>'s a and b, which a's
// walk emitted, as value fields, and a check that walks each such type again
// doubles per level and does not return.
func TestGenerateGroupDiamond(t *testing.T) {
	const depth = 40
	var b strings.Builder
	b.WriteString(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="r"><xs:complexType><xs:group ref="g0"/></xs:complexType></xs:element>`)
	for i := range depth {
		fmt.Fprintf(&b, `<xs:group name="g%d"><xs:sequence><xs:element name="a"><xs:complexType><xs:group ref="g%d"/></xs:complexType></xs:element><xs:element name="b"><xs:complexType><xs:group ref="g%d"/></xs:complexType></xs:element></xs:sequence></xs:group>`, i, i+1, i+1)
	}
	fmt.Fprintf(&b, `<xs:group name="g%d"><xs:sequence><xs:element name="s" type="xs:string"/></xs:sequence></xs:group></xs:schema>`, depth)
	schema := parseString(t, b.String())
	dir := t.TempDir()
	if err := Generate([]Target{{Schema: schema, Dir: dir, Package: "p"}}); err != nil {
		t.Fatal(err)
	}
	if got, want := read(t, filepath.Join(dir, fileName)), fmt.Sprintf("type G%dA struct {\n\tA G%dA\n\tB G%dB\n}\n", depth-2, depth-1, depth-1); !strings.Contains(got, want) {
		t.Errorf("want %q in:\n%s", want, got)
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
