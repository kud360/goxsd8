package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// orderSummary is the whole stdout block `goxsd8 parse testdata/order.xsd`
// prints, pinned byte for byte. It is what makes "print a summary" a contract:
// the namespace line, one count per §3.17.1 property the schema document
// declares into, and their total. The type count is 2 and not 54 — the seeded
// built-in datatypes carry no source position and are not the schema's own
// declarations (see declaredNames).
const orderSummary = `testdata/order.xsd
  namespace: http://example.com/order
  types: 2
  elements: 1
  attributes: 1
  attribute groups: 1
  model groups: 1
  notations: 1
  identity constraints: 1
  components: 8
`

// importsSummary is the whole stdout block `goxsd8 parse testdata/imports.xsd`
// prints. Its two namespace lines are what makes namespacesOf's ordering
// observable at all: with one namespace the order slice never holds two
// entries, so ranging the seen map instead would print the same bytes.
//
// The order is first appearance over the buckets, which is the root document's
// namespace and then the imported one — not sorted, which would put
// .../imported first.
const importsSummary = `testdata/imports.xsd
  namespace: http://example.com/root
  namespace: http://example.com/imported
  types: 0
  elements: 2
  attributes: 0
  attribute groups: 0
  model groups: 0
  notations: 0
  identity constraints: 0
  components: 2
`

// brokenRule is the src-resolve charge testdata/broken.xsd earns. The line is
// asserted by its shape and this substring rather than in full: the message is
// the parser's and the location is absolute, so pinning either here would make
// this test a copy of parser's wording.
const brokenRule = "[src-resolve]"

// malformedRule is the charge testdata/malformed.xsd earns. Unlike broken.xsd's
// src-resolve, the parser returns this one WRAPPED in the assembly context it
// was read under, so it is the fixture that reaches violationLine's unwrap.
const malformedRule = "[xml-wf]"

func TestParseSummary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"parse", "testdata/order.xsd"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("parse of a valid schema = %d, want 0 (stderr %q)", code, stderr.String())
	}
	if stdout.String() != orderSummary {
		t.Errorf("summary =\n%s\nwant\n%s", stdout.String(), orderSummary)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

// TestParseNamespaceOrder pins the one order the summary derives rather than
// reads off an enumeration: namespacesOf's first-appearance order over the
// buckets. A two-namespace set is what makes it assertable — with a single
// namespace, ranging the seen map produces the same bytes.
func TestParseNamespaceOrder(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"parse", "testdata/imports.xsd"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("parse of an importing schema = %d, want 0 (stderr %q)", code, stderr.String())
	}
	if stdout.String() != importsSummary {
		t.Errorf("summary =\n%s\nwant\n%s", stdout.String(), importsSummary)
	}
}

// TestParseSummaryIsDeterministic is the acceptance bar, not a nicety (STYLE
// D1/D2): the same input must produce the same bytes, run after run. A summary
// assembled by ranging a map would pass TestParseSummary most of the time and
// fail here — but only on the importing fixture, whose set carries the two
// namespaces map iteration has something to permute between.
func TestParseSummaryIsDeterministic(t *testing.T) {
	for _, location := range []string{"testdata/order.xsd", "testdata/imports.xsd"} {
		var first bytes.Buffer
		if code := run([]string{"parse", location}, &first, &bytes.Buffer{}); code != 0 {
			t.Fatalf("parse %s = %d, want 0", location, code)
		}
		for i := range 20 {
			var again bytes.Buffer
			if code := run([]string{"parse", location}, &again, &bytes.Buffer{}); code != 0 {
				t.Fatalf("parse %s = %d, want 0", location, code)
			}
			if again.String() != first.String() {
				t.Fatalf("%s run %d differs from run 0:\n%s\nvs\n%s", location, i+1, again.String(), first.String())
			}
		}
	}
}

// TestParseSchemaErrors pins the failure contract: exit 1, nothing on stdout,
// and the rejected schema's first error on stderr in xsderr.Error's own
// rendering — a location a reader can open, the rule ID in brackets, then the
// message. Assembly stops at that error, so one rejected argument is one line.
func TestParseSchemaErrors(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"parse", "testdata/broken.xsd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("parse of a rejected schema = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty — a rejected schema has no summary", stdout.String())
	}
	line := strings.TrimSuffix(stderr.String(), "\n")
	if strings.Contains(line, "\n") {
		t.Errorf("stderr = %q, want one line", stderr.String())
	}
	if !strings.Contains(line, brokenRule) {
		t.Errorf("stderr = %q, want the charged rule in brackets", line)
	}
	abs, err := filepath.Abs("testdata/broken.xsd")
	if err != nil {
		t.Fatal(err)
	}
	// The location must be openable from anywhere, and 1-indexed: the
	// offending element sits on line 9 of the fixture.
	if !strings.HasPrefix(line, abs+":9:") {
		t.Errorf("stderr = %q, want it to open with %q", line, abs+":9:")
	}
}

// TestParseWrappedSchemaError pins violationLine's unwrap, which no
// src-resolve fixture can reach: the parser returns that class unwrapped.
// A document that is not well-formed XML comes back WRAPPED in the assembly
// context it was read under, and the contract's first field is a location a
// reader can open, not the package that did the reading. Deleting the
// errors.As branch fails here and nowhere else.
func TestParseWrappedSchemaError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"parse", "testdata/malformed.xsd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("parse of a malformed document = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty — a rejected schema has no summary", stdout.String())
	}
	line := strings.TrimSuffix(stderr.String(), "\n")
	if strings.Contains(line, "\n") {
		t.Errorf("stderr = %q, want one line", stderr.String())
	}
	if !strings.Contains(line, malformedRule) {
		t.Errorf("stderr = %q, want the charged rule in brackets", line)
	}
	abs, err := filepath.Abs("testdata/malformed.xsd")
	if err != nil {
		t.Fatal(err)
	}
	// The unclosed start tag runs to line 9 of the fixture.
	if !strings.HasPrefix(line, abs+":9:") {
		t.Errorf("stderr = %q, want it to open with %q", line, abs+":9:")
	}
	if strings.HasPrefix(line, "parser:") {
		t.Errorf("stderr = %q, want xsderr.Error's own rendering, unwrapped", line)
	}
}

// TestParseQuiet pins -q's scope, which binds validate too: it suppresses the
// summary and nothing else. A -q that swallowed the error lines would break
// every grep-based script, which is why the second half of this test is the
// load-bearing one.
func TestParseQuiet(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"parse", "-q", "testdata/order.xsd"}, &stdout, &stderr); code != 0 {
		t.Errorf("parse -q of a valid schema = %d, want 0 (stderr %q)", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("parse -q stdout = %q, want empty", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"parse", "-q", "testdata/broken.xsd"}, &stdout, &stderr); code != 1 {
		t.Errorf("parse -q of a rejected schema = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), brokenRule) {
		t.Errorf("parse -q stderr = %q, want the error line — -q never silences a diagnosis", stderr.String())
	}
}

// TestParseVerbose pins the other half of STYLE L1: silent by default, and -v
// is the whole of the injection. The parser logs under its own group, so any
// output at all proves the logger reached parser.WithLogger.
func TestParseVerbose(t *testing.T) {
	var quietErr bytes.Buffer
	run([]string{"parse", "testdata/order.xsd"}, &bytes.Buffer{}, &quietErr)
	if quietErr.Len() != 0 {
		t.Errorf("parse without -v wrote %q to stderr, want silence", quietErr.String())
	}
	var loudErr bytes.Buffer
	run([]string{"parse", "-v", "testdata/order.xsd"}, &bytes.Buffer{}, &loudErr)
	if loudErr.Len() == 0 {
		t.Error("parse -v wrote nothing to stderr, want the parser's debug log")
	}
}

// TestParseMultipleSchemas pins the multi-schema decision: one run per
// argument, in argument order, every argument reported however the one before
// it went, and the worst outcome as the exit code.
func TestParseMultipleSchemas(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"parse", "testdata/order.xsd", "testdata/broken.xsd", "testdata/order.xsd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("parse of a valid and a rejected schema = %d, want 1", code)
	}
	if stdout.String() != orderSummary+orderSummary {
		t.Errorf("stdout =\n%s\nwant the valid schema's summary twice", stdout.String())
	}
	if !strings.Contains(stderr.String(), brokenRule) {
		t.Errorf("stderr = %q, want the rejected schema's error", stderr.String())
	}

	// An unreadable argument outranks a schema verdict: 2 says no verdict was
	// reached, and it must not be masked by the 1 an earlier argument earned.
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"parse", "testdata/broken.xsd", "testdata/nosuch.xsd"}, &stdout, &stderr); code != 2 {
		t.Errorf("parse of a rejected and an unreadable schema = %d, want 2", code)
	}
}

// TestParseUsageErrors pins the exit-2 half of the split this issue owns: a
// fault in the command line or in reading a file is never a verdict about a
// schema. Each answer is distinct, lands on stderr with helpPointer under it,
// and leaves stdout empty.
func TestParseUsageErrors(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"no schema", []string{"parse"}, "goxsd8: parse: no schema given"},
		{"missing file", []string{"parse", "testdata/nosuch.xsd"}, "no such file or directory"},
		{"a directory", []string{"parse", dir}, "is a directory"},
		{"undefined flag", []string{"parse", "-bogus", "testdata/order.xsd"}, "flag provided but not defined: -bogus"},
		// -help=true is not one of the three help spellings, at any position
		// (doc.go's argument vocabulary); after a subcommand it is a flag
		// whose value that subcommand does not accept.
		{"help as a flag value", []string{"parse", "-help=true"}, helpNotAFlagValue},
	}
	seen := make(map[string]string)
	for _, c := range cases {
		var stdout, stderr bytes.Buffer
		if code := run(c.args, &stdout, &stderr); code != 2 {
			t.Errorf("%s: run(%q) = %d, want 2", c.name, c.args, code)
		}
		if stdout.Len() != 0 {
			t.Errorf("%s: stdout = %q, want empty", c.name, stdout.String())
		}
		if !strings.Contains(stderr.String(), c.want) {
			t.Errorf("%s: stderr = %q, want it to contain %q", c.name, stderr.String(), c.want)
		}
		if !strings.HasSuffix(stderr.String(), helpPointer+"\n") {
			t.Errorf("%s: stderr = %q, want helpPointer under it", c.name, stderr.String())
		}
		if prior, dup := seen[stderr.String()]; dup {
			t.Errorf("%s answers exactly what %s answers: %q", c.name, prior, stderr.String())
		}
		seen[stderr.String()] = c.name
	}
}

// TestParseFaultsAreDistinguishable is the cliuser criterion this issue was
// filed to satisfy: a script must be able to tell "your command line is wrong"
// from "that subcommand does nothing yet" from "your schema is invalid",
// without parsing prose. The exit code separates the last from the first two,
// and the stderr line separates those two from each other.
func TestParseFaultsAreDistinguishable(t *testing.T) {
	kinds := []struct {
		name string
		args []string
		code int
	}{
		{"no schema argument", []string{"parse"}, 2},
		{"unreadable schema", []string{"parse", "testdata/nosuch.xsd"}, 2},
		{"reserved subcommand", []string{"validate", "a.xml"}, 2},
		{"invalid schema", []string{"parse", "testdata/broken.xsd"}, 1},
		{"valid schema", []string{"parse", "testdata/order.xsd"}, 0},
	}
	seen := make(map[string]string)
	for _, k := range kinds {
		var stdout, stderr bytes.Buffer
		if code := run(k.args, &stdout, &stderr); code != k.code {
			t.Errorf("%s: run(%q) = %d, want %d", k.name, k.args, code, k.code)
		}
		if prior, dup := seen[stderr.String()]; dup {
			t.Errorf("%s is byte-identical on stderr to %s: %q", k.name, prior, stderr.String())
		}
		seen[stderr.String()] = k.name
	}
}

// TestParsePathSpellings pins what the resolver rooting buys: an argument
// spelled absolutely, or reaching out of the working directory through "..",
// names the same schema as the relative spelling and compiles the same.
func TestParsePathSpellings(t *testing.T) {
	abs, err := filepath.Abs("testdata/order.xsd")
	if err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// The same file reached by climbing out of the working directory and back
	// in: loader.Dir's default confinement to "." would refuse this spelling.
	climbing := filepath.Join("..", filepath.Base(cwd), "testdata", "order.xsd")
	for _, given := range []string{abs, climbing} {
		var stdout, stderr bytes.Buffer
		if code := run([]string{"parse", given}, &stdout, &stderr); code != 0 {
			t.Errorf("parse %q = %d, want 0 (stderr %q)", given, code, stderr.String())
		}
		// The header echoes the argument as spelled; the counts do not move.
		want := given + strings.TrimPrefix(orderSummary, "testdata/order.xsd")
		if stdout.String() != want {
			t.Errorf("parse %q stdout =\n%s\nwant\n%s", given, stdout.String(), want)
		}
	}
}

// TestParseIncludeResolvesAgainstItsOwnDocument pins the other half of the
// rooting: a relative schemaLocation inside a schema is served from that
// schema's directory, not from the process's working directory.
func TestParseIncludeResolvesAgainstItsOwnDocument(t *testing.T) {
	dir := t.TempDir()
	const ns = `xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="http://example.com/inc"`
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("common.xsd", `<xs:schema `+ns+`><xs:element name="shared" type="xs:string"/></xs:schema>`)
	write("main.xsd", `<xs:schema `+ns+`><xs:include schemaLocation="common.xsd"/><xs:element name="root" type="xs:string"/></xs:schema>`)

	main := filepath.Join(dir, "main.xsd")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"parse", main}, &stdout, &stderr); code != 0 {
		t.Fatalf("parse %q = %d, want 0 (stderr %q)", main, code, stderr.String())
	}
	// Two elements only if the <include> resolved: main.xsd declares one.
	if want := "  elements: 2\n"; !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout =\n%s\nwant %q — the include did not compose", stdout.String(), want)
	}
}

// TestParseAbsentNamespace pins the summary's rendering of the ·absent· target
// namespace (§2.2), which a QName carries as an empty Space: a no-namespace
// schema gets a namespace line like any other rather than a blank one.
func TestParseAbsentNamespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nons.xsd")
	body := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a" type="xs:string"/></xs:schema>`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"parse", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("parse = %d, want 0 (stderr %q)", code, stderr.String())
	}
	if want := "  namespace: " + absentNamespace + "\n"; !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout =\n%s\nwant %q", stdout.String(), want)
	}
}

// TestParseNamespaceWithNoComponents pins the other end of the same
// derivation: §3.17.1 gives the Schema component no {target namespace}
// property, so a document's targetNamespace is observable only through the
// components it declares, and one that declares none reports no namespace at
// all — not the ·absent· one, which would claim a schema with no namespace.
func TestParseNamespaceWithNoComponents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nocomponents.xsd")
	body := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="http://example.com/ns">` +
		`<xs:annotation><xs:documentation>declares nothing</xs:documentation></xs:annotation></xs:schema>`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"parse", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("parse = %d, want 0 (stderr %q)", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "  namespace:") {
		t.Errorf("stdout =\n%s\nwant no namespace line at all", stdout.String())
	}
}

// TestParseAdversarialArguments is the robustness bar the stub set and this
// landing must not fall below (#472, the 2026-08-12 cliuser pass): no panic,
// an exit code in the contract's three, and the streams kept apart, whatever
// the argument list.
func TestParseAdversarialArguments(t *testing.T) {
	lists := [][]string{
		{"parse", ""},
		{"parse", "--"},
		{"parse", "\x00\x01"},
		{"parse", "-"},
		{"parse", "-q", "-q"},
		{"parse", "-v", ""},
		{"parse", strings.Repeat("a/", 200) + "x.xsd"},
	}
	for _, args := range lists {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 2 {
			t.Errorf("run(%q) = %d, want 2 — none of these names a readable schema", args, code)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%q) stdout = %q, want empty", args, stdout.String())
		}
		if stderr.Len() == 0 {
			t.Errorf("run(%q) reported nothing on stderr", args)
		}
	}
}

// TestParseFlagAfterPositional pins the consequence doc.go's argument
// vocabulary states: a subcommand's flags precede its positional arguments,
// the flag package stopping at the first of them, so a trailing -q is a schema
// location and not a request for quiet.
func TestParseFlagAfterPositional(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"parse", "testdata/order.xsd", "-q"}, &stdout, &stderr)
	if code != 2 {
		t.Errorf("run = %d, want 2 — the trailing -q names no readable schema", code)
	}
	if stdout.String() != orderSummary {
		t.Errorf("stdout =\n%s\nwant the summary, which a trailing -q does not suppress", stdout.String())
	}
}
