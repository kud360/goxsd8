package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The fixtures every outcome test turns on: the schema parse's own tests
// compile, an instance of it that charges nothing, one whose sku lexical is
// outside its Sku pattern, and a second namespace's schema paired with an
// instance that names it through xsi:schemaLocation and nothing else.
const (
	validInstance   = "testdata/order-valid.xml"
	invalidInstance = "testdata/order-invalid.xml"
	orderSchema     = "testdata/order.xsd"
	hintedInstance  = "testdata/hinted.xml"
	hintedSchema    = "testdata/hinted.xsd"
)

// TestValidateCleanInstance pins the quiet outcome: an instance that charges
// nothing writes nothing on either stream and exits 0, so a script can run
// validate in a pipeline and see output only when there is something to see.
func TestValidateCleanInstance(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"validate", "-schema", orderSchema, validInstance}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("code = %d, want %d (stdout %q, stderr %q)", code, exitOK, stdout.String(), stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Errorf("stdout = %q, stderr = %q, want both empty", stdout.String(), stderr.String())
	}
}

// TestValidateViolationsGoToStdout pins the stream and the rendering #1066
// decided and this landing implements: validate's product is the violation
// report a script pipes into grep, so it lands on stdout in the
// "<loc>: [<rule>] <message>" shape parse gives a schema error on stderr, and
// stderr stays empty because nothing went wrong with the run.
func TestValidateViolationsGoToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"validate", "-schema", orderSchema, invalidInstance}, &stdout, &stderr)
	if code != exitInvalid {
		t.Fatalf("code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty: an invalid instance is a verdict, not a fault", stderr.String())
	}
	line, _, _ := strings.Cut(stdout.String(), "\n")
	if !strings.HasPrefix(line, invalidInstance+":") {
		t.Errorf("first line = %q, want it to open with the instance location", line)
	}
	if !strings.Contains(line, "[cvc-attribute]") {
		t.Errorf("first line = %q, want the charged rule ID in brackets", line)
	}
}

// TestValidateQuietDoesNotSuppressViolations pins the constraint #16 still
// carries and doc.go states: -q suppresses a subcommand's INFORMATIONAL
// output, and validate's violations are its product, not information about it.
func TestValidateQuietDoesNotSuppressViolations(t *testing.T) {
	var quiet, loud bytes.Buffer
	var stderr bytes.Buffer
	if code := run([]string{"validate", "-q", "-schema", orderSchema, invalidInstance}, &quiet, &stderr); code != exitInvalid {
		t.Fatalf("code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"validate", "-schema", orderSchema, invalidInstance}, &loud, &stderr); code != exitInvalid {
		t.Fatalf("code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	if quiet.String() != loud.String() {
		t.Errorf("-q changed the violation report:\n%s\nwant\n%s", quiet.String(), loud.String())
	}
	if quiet.Len() == 0 {
		t.Error("-q suppressed the whole report")
	}
}

// TestValidateReportsEveryInstance is the half #1066 left to this issue: the
// run does not stop at the first invalid instance. Both files' violations are
// on stdout, in argument order, from one invocation.
func TestValidateReportsEveryInstance(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-no-hints", "-schema", orderSchema, invalidInstance, validInstance, hintedInstance}
	if code := run(args, &stdout, &stderr); code != exitInvalid {
		t.Fatalf("code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	first := strings.Index(stdout.String(), invalidInstance)
	last := strings.Index(stdout.String(), hintedInstance)
	if first < 0 || last < 0 {
		t.Fatalf("stdout =\n%s\nwant a line for both %s and %s", stdout.String(), invalidInstance, hintedInstance)
	}
	if first > last {
		t.Errorf("stdout =\n%s\nwant the instances reported in argument order", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[cvc-assess-elt]") {
		t.Errorf("stdout =\n%s\nwant the third instance's own charge, which a short-circuit would have skipped", stdout.String())
	}
}

// TestValidateSchemaFaultHasItsOwnExitCode is this issue's first Acceptance
// bullet: a broken schema and broken data do not share a code, so a CI script
// branches on "your schema is wrong" without reading a message. The schema
// fault is also reported ONCE, before any instance is read, and on stderr —
// stdout carries verdicts about instances alone.
func TestValidateSchemaFaultHasItsOwnExitCode(t *testing.T) {
	// The code is its own before any run proves a schema set earns it: a
	// contract that promised the split and spent an existing code on it would
	// leave every assertion below passing.
	if exitSchema == exitOK || exitSchema == exitInvalid || exitSchema == exitUsage {
		t.Fatalf("exitSchema = %d, want a code of its own beside %d, %d and %d", exitSchema, exitOK, exitInvalid, exitUsage)
	}

	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", "testdata/broken.xsd", validInstance, invalidInstance}
	code := run(args, &stdout, &stderr)
	if code != exitSchema {
		t.Fatalf("code = %d, want %d", code, exitSchema)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty", stdout.String())
	}
	if got := strings.Count(stderr.String(), "\n"); got != 1 {
		t.Errorf("stderr = %q, want one line however many instances were named", stderr.String())
	}
	if !strings.Contains(stderr.String(), "[src-resolve]") {
		t.Errorf("stderr = %q, want the rule the assembly charged", stderr.String())
	}
}

// TestValidateSchemasComposeIntoOneSet pins the contract sentence a user can
// learn nowhere else: several -schema arguments are ONE compilation, not one
// each. hinted.xml's root is declared only by hinted.xsd, so it can be
// assessed with hints OFF exactly when the two documents compose.
func TestValidateSchemasComposeIntoOneSet(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-no-hints", "-schema", orderSchema, "-schema", hintedSchema, hintedInstance}
	if code := run(args, &stdout, &stderr); code != exitOK {
		t.Fatalf("code = %d, want %d (stdout %q, stderr %q)", code, exitOK, stdout.String(), stderr.String())
	}

	// The same instance against the same partial set alone: the composition,
	// not the order.xsd argument, is what made the run above succeed.
	stdout.Reset()
	stderr.Reset()
	partial := []string{"validate", "-no-hints", "-schema", orderSchema, hintedInstance}
	if code := run(partial, &stdout, &stderr); code != exitInvalid {
		t.Fatalf("partial set: code = %d, want %d", code, exitInvalid)
	}
}

// TestValidateSchemasShareATargetNamespace is the composition case a wrapper
// that <include>d everything would get wrong and the one the persona's Story C
// describes: two documents in ONE target namespace, one declaring the element
// and the other the type it references. They compose only if the set is a real
// assembly, since neither document alone resolves the reference.
func TestValidateSchemasShareATargetNamespace(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	declaration := write("element.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:x="urn:x" targetNamespace="urn:x"><xs:element name="root" type="x:Count"/></xs:schema>`)
	definition := write("type.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:x"><xs:simpleType name="Count"><xs:restriction base="xs:int"/></xs:simpleType></xs:schema>`)
	valid := write("valid.xml", `<root xmlns="urn:x">42</root>`)
	invalid := write("invalid.xml", `<root xmlns="urn:x">forty-two</root>`)

	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-no-hints", "-schema", declaration, "-schema", definition, valid}
	if code := run(args, &stdout, &stderr); code != exitOK {
		t.Fatalf("code = %d, want %d (stdout %q, stderr %q)", code, exitOK, stdout.String(), stderr.String())
	}

	// The set still decides: composing is not the same as accepting.
	stdout.Reset()
	stderr.Reset()
	args = []string{"validate", "-no-hints", "-schema", declaration, "-schema", definition, invalid}
	if code := run(args, &stdout, &stderr); code != exitInvalid {
		t.Fatalf("invalid instance: code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}

	// And the reference is unresolved without the second document, which is a
	// schema fault rather than a verdict about the instance.
	stdout.Reset()
	stderr.Reset()
	args = []string{"validate", "-no-hints", "-schema", declaration, valid}
	if code := run(args, &stdout, &stderr); code != exitSchema {
		t.Fatalf("half the set: code = %d, want %d (stderr %q)", code, exitSchema, stderr.String())
	}
	if !strings.Contains(stderr.String(), "[src-resolve]") {
		t.Errorf("stderr = %q, want the unresolved reference charged", stderr.String())
	}
}

// TestValidateHints is the -no-hints Acceptance bullet, both halves: a partial
// -schema set plus the instance's own xsi:schemaLocation succeeds, and the
// same set with -no-hints fails under a rule ID that names the fault —
// cvc-assess-elt, a validation root the set declares nothing for — rather than
// a generic error.
func TestValidateHints(t *testing.T) {
	var stdout, stderr bytes.Buffer
	followed := []string{"validate", "-schema", orderSchema, hintedInstance}
	if code := run(followed, &stdout, &stderr); code != exitOK {
		t.Fatalf("hints followed: code = %d, want %d (stdout %q, stderr %q)", code, exitOK, stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	ignored := []string{"validate", "-no-hints", "-schema", orderSchema, hintedInstance}
	if code := run(ignored, &stdout, &stderr); code != exitInvalid {
		t.Fatalf("-no-hints: code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[cvc-assess-elt]") {
		t.Errorf("-no-hints stdout = %q, want the cvc-assess-elt charge", stdout.String())
	}
}

// TestValidateNoNamespaceHint pins the other half of §2.7.3: the
// no-namespace hint names one location and no namespace to pair it with, so
// the wrapper root <include>s that document rather than <import>ing it —
// src-import clause 1.2 forbids a namespace-less <import> from a wrapper with
// no target namespace, so an <import> here would fail the whole set.
func TestValidateNoNamespaceHint(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	write("plain.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="plain" type="xs:int"/></xs:schema>`)
	valid := write("valid.xml", `<plain xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="plain.xsd">7</plain>`)
	invalid := write("invalid.xml", `<plain xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="plain.xsd">seven</plain>`)

	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate", "-schema", orderSchema, valid}, &stdout, &stderr); code != exitOK {
		t.Fatalf("code = %d, want %d (stdout %q, stderr %q)", code, exitOK, stdout.String(), stderr.String())
	}

	// The hinted document decides, so a wrong lexical is still charged.
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"validate", "-schema", orderSchema, invalid}, &stdout, &stderr); code != exitInvalid {
		t.Fatalf("invalid instance: code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[cvc-type]") {
		t.Errorf("stdout = %q, want the hinted type's charge", stdout.String())
	}

	// And -no-hints leaves the root undeclared, as it does for a namespaced
	// hint.
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"validate", "-no-hints", "-schema", orderSchema, valid}, &stdout, &stderr); code != exitInvalid {
		t.Fatalf("-no-hints: code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	if !strings.Contains(stdout.String(), "[cvc-assess-elt]") {
		t.Errorf("-no-hints stdout = %q, want the cvc-assess-elt charge", stdout.String())
	}
}

// TestValidateUnusableHintIsTheInstancesFault pins the attribution a hint
// failure gets: a hint is the INSTANCE's own advisory claim (§4.3.2 clause 3),
// so a set that stops compiling once that instance's hints are folded in is a
// fault of the instance, never of the -schema set the invocation named. The
// three ways a hint can be unusable — pairing a namespace with a document that
// declares another, naming a document that is not well-formed, naming one that
// is not there — therefore degrade identically: the -schema set alone decides,
// which charges cvc-assess-elt here, and exitSchema stays the answer to a
// -schema set that does not compile.
//
// No line may cite schemaSetLocation, which is this process's own synthesis and
// names no document the reader can open (STYLE E3).
func TestValidateUnusableHintIsTheInstancesFault(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	// The document the mis-paired hint names: its own targetNamespace is not
	// the one the hint pairs it with, which is src-import clause 3.1.
	write("actual.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="http://example.com/actual"><xs:element name="note" type="xs:string"/></xs:schema>`)
	write("junk.xsd", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"`)
	instance := func(name, location string) string {
		return write(name, `<h:note xmlns:h="http://example.com/hinted" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://example.com/hinted `+location+`">ok</h:note>`)
	}
	cases := []struct {
		name string
		path string
		// want is the hinted document the diagnosis must name, empty where the
		// set composes and there is nothing to report.
		want string
	}{
		{"namespace mis-paired", instance("mispaired.xml", "actual.xsd"), "actual.xsd"},
		{"hinted document malformed", instance("malformed-hint.xml", "junk.xsd"), "junk.xsd"},
		{"hinted document missing", instance("missing-hint.xml", "nosuch.xsd"), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{"validate", "-schema", orderSchema, c.path}, &stdout, &stderr)
			if code != exitInvalid {
				t.Fatalf("code = %d, want %d (stdout %q, stderr %q)", code, exitInvalid, stdout.String(), stderr.String())
			}
			if !strings.Contains(stdout.String(), "[cvc-assess-elt]") {
				t.Errorf("stdout = %q, want the -schema set's own charge for an undeclared root", stdout.String())
			}
			if strings.Contains(stderr.String(), schemaSetLocation) {
				t.Errorf("stderr = %q, want no line citing the synthesized wrapper root", stderr.String())
			}
			if strings.Contains(stdout.String(), schemaSetLocation) {
				t.Errorf("stdout = %q, want no line citing the synthesized wrapper root", stdout.String())
			}
			if c.want == "" {
				if stderr.Len() != 0 {
					t.Errorf("stderr = %q, want empty: the set composed", stderr.String())
				}
				return
			}
			if !strings.Contains(stderr.String(), c.path) {
				t.Errorf("stderr = %q, want the instance that carried the hint named", stderr.String())
			}
			if !strings.Contains(stderr.String(), c.want) {
				t.Errorf("stderr = %q, want the hinted document %s named", stderr.String(), c.want)
			}
		})
	}
}

// TestValidateHintRepeatsASchemaArgument is the case a naive union would break
// on and the resolver's dedup contract covers: the instance hints the very
// document -schema already named, so the set holds it twice by location and
// once by identity. Composing it twice would collide every component under
// sch-props-correct clause 2.
func TestValidateHintRepeatsASchemaArgument(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", hintedSchema, hintedInstance}
	if code := run(args, &stdout, &stderr); code != exitOK {
		t.Fatalf("code = %d, want %d (stdout %q, stderr %q)", code, exitOK, stdout.String(), stderr.String())
	}
}

// TestValidateHintsResolveAgainstTheInstance pins §4.3.2 clause 4 as the CLI
// applies it: a relative hint names a document beside the INSTANCE, not beside
// the working directory, so an instance moved into another directory carries
// its schema with it.
func TestValidateHintsResolveAgainstTheInstance(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"hinted.xml", "hinted.xsd"} {
		body, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", orderSchema, filepath.Join(dir, "hinted.xml")}
	if code := run(args, &stdout, &stderr); code != exitOK {
		t.Fatalf("code = %d, want %d (stdout %q, stderr %q)", code, exitOK, stdout.String(), stderr.String())
	}
}

// TestValidateFormatVocabulary pins the -format enum this issue narrowed: the
// three tokens are matched case-sensitively, an unrecognized one is a usage
// error LISTING the valid values, an instance whose extension names none of
// them is the same error rather than a guess, and the two tokens whose
// adapters are still doc.go stubs fail cleanly instead of passing silently.
func TestValidateFormatVocabulary(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unrecognized token", []string{"-format", "yaml"}, `-format "yaml" is not a source format; the values are xml, json, ber`},
		{"wrong case", []string{"-format", "XML"}, `-format "XML" is not a source format`},
		{"forced json is reserved", []string{"-format", "json"}, "-format json is reserved by the contract and not yet implemented"},
		{"forced ber is reserved", []string{"-format", "ber"}, "-format ber is reserved by the contract and not yet implemented"},
		{"extension is the default", []string{}, "the extension \".txt\" names no source format; pass -format xml, json, ber"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := append([]string{"validate"}, c.args...)
			args = append(args, "-schema", orderSchema, "instance.txt")
			var stdout, stderr bytes.Buffer
			if code := run(args, &stdout, &stderr); code != exitUsage {
				t.Fatalf("code = %d, want %d", code, exitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
			if !strings.Contains(stderr.String(), c.want) {
				t.Errorf("stderr = %q, want it to contain %q", stderr.String(), c.want)
			}
		})
	}
}

// TestValidateJSONInstanceIsReservedNotSilent pins the same answer reached
// through the extension rather than the flag: validate/jsonsrc carries a
// doc.go and no Validate, so a .json instance earns the usage code and a
// not-yet-implemented line — never a pass.
func TestValidateJSONInstanceIsReservedNotSilent(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", orderSchema, "instance.json"}
	if code := run(args, &stdout, &stderr); code != exitUsage {
		t.Fatalf("code = %d, want %d", code, exitUsage)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty: nothing was assessed", stdout.String())
	}
	if !strings.Contains(stderr.String(), "not yet implemented") {
		t.Errorf("stderr = %q, want a not-yet-implemented diagnosis", stderr.String())
	}
}

// TestValidateSchemaFlagIsRepeatable pins the grammar doc.go states and the
// flag.Value implements: every schema needs its own -schema, so a second
// location standing beside the first is a positional argument — an instance —
// and not a second schema.
func TestValidateSchemaFlagIsRepeatable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", orderSchema, hintedSchema, validInstance}
	if code := run(args, &stdout, &stderr); code != exitUsage {
		t.Fatalf("code = %d, want %d (stderr %q)", code, exitUsage, stderr.String())
	}
	if !strings.Contains(stderr.String(), hintedSchema) {
		t.Errorf("stderr = %q, want %s reported as an instance argument", stderr.String(), hintedSchema)
	}
}

// TestValidateUsageErrors covers the usage/IO code's whole surface for this
// subcommand: each diagnosis reaches stderr, stdout stays empty, and no two of
// them collapse into one line.
func TestValidateUsageErrors(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no schema", []string{"validate", validInstance}, "goxsd8: validate: no schema given"},
		{"no instance", []string{"validate", "-schema", orderSchema}, "goxsd8: validate: no instance given"},
		{"dangling -schema", []string{"validate", "-schema"}, "flag needs an argument: -schema"},
		{"undefined flag", []string{"validate", "-out", dir, "-schema", orderSchema, validInstance}, "flag provided but not defined: -out"},
		{"missing schema", []string{"validate", "-schema", "testdata/nosuch.xsd", validInstance}, "no such file or directory"},
		{"schema is a directory", []string{"validate", "-schema", dir, validInstance}, "is a directory"},
		{"schema from stdin", []string{"validate", "-schema", "-", validInstance}, "standard input is not a schema location"},
		{"missing instance", []string{"validate", "-schema", orderSchema, "testdata/nosuch.xml"}, "no such file or directory"},
		{"stdin needs -format", []string{"validate", "-schema", orderSchema, "-"}, "carries no extension to name a source format"},
		// -help=true is not one of the three help spellings, at any position
		// (doc.go's argument vocabulary); after a subcommand it is a flag whose
		// value that subcommand does not accept.
		{"help as a flag value", []string{"validate", "-help=true"}, fmt.Sprintf(helpNotAFlagValueFmt, "validate")},
	}
	seen := make(map[string]string)
	for _, c := range cases {
		var stdout, stderr bytes.Buffer
		if code := run(c.args, &stdout, &stderr); code != exitUsage {
			t.Errorf("%s: run(%q) = %d, want %d", c.name, c.args, code, exitUsage)
		}
		if stdout.Len() != 0 {
			t.Errorf("%s: stdout = %q, want empty", c.name, stdout.String())
		}
		if !strings.Contains(stderr.String(), c.want) {
			t.Errorf("%s: stderr = %q, want it to contain %q", c.name, stderr.String(), c.want)
		}
		if !strings.HasSuffix(stderr.String(), helpPointer+"\n") {
			t.Errorf("%s: stderr = %q, want it to end with the remedy line", c.name, stderr.String())
		}
		if other, ok := seen[stderr.String()]; ok {
			t.Errorf("%s and %s produce the same diagnosis %q", c.name, other, stderr.String())
		}
		seen[stderr.String()] = c.name
	}
}

// TestValidateSchemaFileNamedDash pins the escape hatch the -schema -
// refusal leaves open: the exact spelling - is refused and nothing else is, so
// a file genuinely named - is still compiled as a schema document when a path
// names it — ./- from its own directory, or an absolute path ending in it.
//
// The instance it rejects is asserted too, because an argument that was read
// and an argument that was ignored both exit 0 on a clean instance.
func TestValidateSchemaFileNamedDash(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return path
	}
	dash := write("-", `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:x"><xs:element name="root" type="xs:int"/></xs:schema>`)
	valid := write("valid.xml", `<root xmlns="urn:x">42</root>`)
	invalid := write("invalid.xml", `<root xmlns="urn:x">forty-two</root>`)

	// ./- names that file only from the directory holding it, so the run happens
	// there; the absolute path names the same file from anywhere.
	t.Chdir(dir)
	for _, schema := range []string{dash, "./-"} {
		var stdout, stderr bytes.Buffer
		if code := run([]string{"validate", "-schema", schema, valid}, &stdout, &stderr); code != exitOK {
			t.Errorf("-schema %s: code = %d, want %d (stdout %q, stderr %q)", schema, code, exitOK, stdout.String(), stderr.String())
		}
		stdout.Reset()
		stderr.Reset()
		if code := run([]string{"validate", "-schema", schema, invalid}, &stdout, &stderr); code != exitInvalid {
			t.Errorf("-schema %s on the invalid instance: code = %d, want %d (stdout %q, stderr %q)", schema, code, exitInvalid, stdout.String(), stderr.String())
		}
	}
}

// TestValidateAdversarialArguments is the robustness bar the stub set and this
// landing must not fall below (#472, the 2026-08-12 cliuser pass): no panic,
// an exit code in the contract's four, and the streams kept apart, whatever
// the argument list. None of these names a readable schema set and a readable
// instance, so every one is a usage or IO fault.
func TestValidateAdversarialArguments(t *testing.T) {
	lists := [][]string{
		{"validate", ""},
		{"validate", "--"},
		{"validate", "\x00\x01"},
		{"validate", "-schema", "", validInstance},
		{"validate", "-schema", "\x00\x01", validInstance},
		{"validate", "-schema", "--", validInstance},
		{"validate", "-schema"},
		{"validate", "-format"},
		{"validate", "-out"},
		{"validate", "-no-hints"},
		{"validate", "-schema", orderSchema, ""},
		{"validate", "-schema", orderSchema, "-"},
		{"validate", "-schema", orderSchema, "\x00\x01.xml"},
		{"validate", "-schema", strings.Repeat("a/", 200) + "x.xsd", validInstance},
		{"validate", "-schema", orderSchema, strings.Repeat("a/", 200) + "x.xml"},
	}
	for _, args := range lists {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != exitUsage {
			t.Errorf("run(%q) = %d, want %d", args, code, exitUsage)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%q) stdout = %q, want empty", args, stdout.String())
		}
		if stderr.Len() == 0 {
			t.Errorf("run(%q) reported nothing on stderr", args)
		}
	}
}

// TestValidateFlagAfterPositional pins the same consequence parse's own test
// does: a subcommand's flags precede its positional arguments, the flag
// package stopping at the first of them, so a trailing -no-hints is an
// instance argument.
func TestValidateFlagAfterPositional(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", orderSchema, hintedInstance, "-no-hints"}
	if code := run(args, &stdout, &stderr); code != exitUsage {
		t.Fatalf("code = %d, want %d — the trailing -no-hints names no instance", code, exitUsage)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty: the one real instance charges nothing", stdout.String())
	}
	if !strings.Contains(stderr.String(), "-no-hints") {
		t.Errorf("stderr = %q, want the trailing argument reported as an instance", stderr.String())
	}
}

// TestValidateMalformedInstanceIsAVerdict pins where a source fault lands: an
// instance the reader cannot finish was not shown valid, so the fault is
// rendered beside the violations on stdout and charged the invalid code — it
// is a verdict about the document, not a fault in the run.
func TestValidateMalformedInstanceIsAVerdict(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", orderSchema, "testdata/malformed.xml"}
	if code := run(args, &stdout, &stderr); code != exitInvalid {
		t.Fatalf("code = %d, want %d (stderr %q)", code, exitInvalid, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
	if !strings.Contains(stdout.String(), "[xml-wf]") {
		t.Errorf("stdout = %q, want the well-formedness fault reported", stdout.String())
	}
}

// TestSchemaSetSource pins the wrapper root's two shapes and its ordering: a
// document with a target namespace of its own is <import>ed and one with none
// is <include>d (src-import clause 1.2 forbids the other spelling from a
// wrapper with no target namespace), in argument order, with every value
// escaped as attribute content.
func TestSchemaSetSource(t *testing.T) {
	got := schemaSetSource([]schemaDoc{
		{location: "/tmp/a.xsd", namespace: "http://example.com/a"},
		{location: "/tmp/b&c.xsd"},
		{location: `/tmp/"d".xsd`, namespace: "urn:e<f"},
	})
	want := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
		`<xs:import namespace="http://example.com/a" schemaLocation="/tmp/a.xsd"/>` +
		`<xs:include schemaLocation="/tmp/b&amp;c.xsd"/>` +
		`<xs:import namespace="urn:e&lt;f" schemaLocation="/tmp/&#34;d&#34;.xsd"/>` +
		`</xs:schema>`
	if got != want {
		t.Errorf("schemaSetSource =\n%s\nwant\n%s", got, want)
	}
}
