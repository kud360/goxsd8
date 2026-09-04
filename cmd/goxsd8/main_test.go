package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// helpCases reach the help path: usage on stdout, nothing on stderr, exit 0.
// Everything after the four bare spellings pins a decision doc.go's argument
// vocabulary states rather than an accident — a help flag before its
// subcommand, a help flag alongside an unrecognized one, -- with no
// end-of-options meaning, a help flag after a name outside the vocabulary
// altogether, and `parse -h` / `validate -h`: those two own the flag.FlagSets
// in the binary, so they are the rows where a subcommand's own flag parsing
// could take -h away from the help path.
var helpCases = [][]string{
	nil,
	{"-h"},
	{"-help"},
	{"--help"},
	{"validate", "-help"},
	{"-help", "validate"},
	{"-xyz", "-help"},
	{"--", "-help"},
	{"frobnicate", "-h"},
	{"parse", "-h"},
	{"validate", "-h"},
	{"validate", "-schema", "testdata/order.xsd", "-h"},
}

// dispatchCases is the diagnosis every non-help invocation that reaches no
// built subcommand earns: a reserved name, a name outside the vocabulary, a
// flag before the subcommand it qualifies, or no subcommand at all. Each is
// followed on stderr by helpPointer. One encoding of the matrix, driven twice
// — through run below and through the built binary in TestBuiltBinaryMatrix.
//
// parse and validate have no row here: both are built, and parse_test.go and
// validate_test.go are their matrices.
var dispatchCases = []struct {
	args []string
	want string
}{
	{[]string{"gen", "-schema", "a.xsd", "-out", "d"}, `goxsd8: gen is not yet implemented`},
	{[]string{"frobnicate"}, `goxsd8: unknown subcommand "frobnicate"`},
	{[]string{"parsee", "a.xsd"}, `goxsd8: unknown subcommand "parsee"`},
	// Case-sensitive matching: a contract name in the wrong case is unknown,
	// not reserved.
	{[]string{"VALIDATE"}, `goxsd8: unknown subcommand "VALIDATE"`},
	// help is not a contract name (doc.go); version is not one either.
	{[]string{"help"}, `goxsd8: unknown subcommand "help"`},
	{[]string{"version"}, `goxsd8: unknown subcommand "version"`},
	// A flag as args[0] with no subcommand anywhere after it is no subcommand
	// at all — not an unknown one. -q is documented and -version is not, and
	// neither is a subcommand name.
	{[]string{"-q"}, noSubcommand},
	{[]string{"-xyz"}, noSubcommand},
	{[]string{"-version"}, noSubcommand},
	{[]string{"-help=true"}, noSubcommand},
	{[]string{"-q", "frobnicate"}, noSubcommand},
	// A flag BEFORE a real subcommand is the one #472 settled: the common
	// flags follow the name they qualify, and the diagnosis says so instead of
	// claiming a subcommand that is right there was never given.
	{[]string{"-q", "parse", "a.xsd"}, `goxsd8: -q must follow the subcommand: goxsd8 parse -q ...`},
	{[]string{"-v", "gen", "-schema", "a.xsd"}, `goxsd8: -v must follow the subcommand: goxsd8 gen -v ...`},
}

// TestRunHelp pins issue #251: a help request — including the bare
// invocation — never reaches a usage error. Usage goes to stdout, stderr
// stays empty, exit code 0.
func TestRunHelp(t *testing.T) {
	for _, args := range helpCases {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)
		if code != 0 {
			t.Errorf("run(%q) = %d, want 0", args, code)
		}
		if stdout.String() != usage {
			t.Errorf("run(%q) stdout = %q, want the usage contract", args, stdout.String())
		}
		if stderr.Len() != 0 {
			t.Errorf("run(%q) stderr = %q, want empty", args, stderr.String())
		}
	}
}

// TestRunDispatch pins #514 and #472: the four diagnoses are distinct, and
// each is the true one for its input.
func TestRunDispatch(t *testing.T) {
	for _, c := range dispatchCases {
		var stdout, stderr bytes.Buffer
		code := run(c.args, &stdout, &stderr)
		if code != 2 {
			t.Errorf("run(%q) = %d, want 2", c.args, code)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%q) stdout = %q, want empty", c.args, stdout.String())
		}
		want := c.want + "\n" + helpPointer + "\n"
		if stderr.String() != want {
			t.Errorf("run(%q) stderr = %q, want %q", c.args, stderr.String(), want)
		}
	}
}

// TestDiagnosesAreDistinct is the property #514 exists for, widened by #472's
// fourth answer: "you typed a name I do not have", "that one is reserved but
// unbuilt", "your flag is on the wrong side of the subcommand" and "you gave
// me no subcommand" must not collapse into one another.
func TestDiagnosesAreDistinct(t *testing.T) {
	kinds := []string{
		diagnose([]string{"gen"}),        // reserved by the contract, unbuilt
		diagnose([]string{"frobnicate"}), // outside the vocabulary
		diagnose([]string{"-q"}),         // a flag, so no subcommand at all
		diagnose([]string{"-q", "gen"}),  // a flag before the name it qualifies
	}
	for i, a := range kinds {
		for _, b := range kinds[i+1:] {
			if a == b {
				t.Errorf("two of the four diagnoses collapse to %q", a)
			}
		}
	}
	if got := diagnose([]string{"GEN"}); got == diagnose([]string{"gen"}) {
		t.Errorf("a reserved name and its wrong-case spelling both diagnose as %q", got)
	}
}

// TestHelpPointerResolvesOutsideTheModule pins #870's second half: the remedy
// a usage error names must work where an installed binary runs. A `go doc`
// invocation does not — it needs the module tree — so no message may name one.
func TestHelpPointerResolvesOutsideTheModule(t *testing.T) {
	if strings.Contains(helpPointer, "go doc") {
		t.Errorf("helpPointer = %q names go doc, which fails outside the module tree", helpPointer)
	}
	if !strings.Contains(helpPointer, "goxsd8 -help") {
		t.Errorf("helpPointer = %q does not name the binary's own help path", helpPointer)
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, errors.New("closed pipe") }

// TestRunHelpWriteFailure covers the usage/IO exit code: help the user never
// received is not a success.
func TestRunHelpWriteFailure(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"-help"}, errWriter{}, &stderr); code != 2 {
		t.Errorf("run with a failing stdout = %d, want 2", code)
	}
}

// TestUsageCoversContract guards the usage constant against drifting away
// from the doc.go contract it renders.
func TestUsageCoversContract(t *testing.T) {
	want := []string{
		"goxsd8 parse [-q] [-v] <schema.xsd>...",
		"goxsd8 validate -schema <schema.xsd> [-schema <s2>]... <instance>...",
		"goxsd8 gen -schema <schema.xsd> -out <dir>",
		"GOXSD_DEBUG=parser,validate,codec",
		"Implemented today: the help path, parse and validate.",
		// The four answers a batch script needs and the page withheld
		// (#1066, #1031): which stream carries parse's summary and its
		// error lines, which carries validate's violations, that the
		// exit code aggregates over the instance arguments, and what
		// -format accepts.
		"summary on stdout",
		"first error on stderr as <loc>: [<rule>] <message>",
		"assembly\n      stops there",
		"1 if any one of them is invalid.",
		"prints one line on stdout",
		"-format xml|json|ber",
		"case-sensitively",
		// #720's own answers, none of which any earlier copy carried: that
		// several -schema arguments are ONE set, that a broken schema set has
		// an exit code of its own, that the run reports every instance rather
		// than stopping at the first bad one, what an unrecognized -format
		// token earns, that two of the three tokens are reserved and unbuilt,
		// that - is an instance spelling alone, and which elements' hints are
		// followed.
		"compose into ONE set",
		"3 when the schema set does not compile",
		"Every\n      instance is assessed — the run never stops at the first invalid",
		"are usage errors listing the\n      values.",
		"Only xml is assessed today",
		"- names standard input as an\n      instance, never as a schema.",
		"hints on the document element of an XML",
		// #472's own four decisions, each of which a user can only learn
		// from this block: what several schema arguments mean, that a
		// document parse cannot read is exit 2 rather than a verdict,
		// where the common flags may stand, and what -q suppresses.
		"several\n      arguments are several compilations, not one set.",
		"2 when an\n      argument cannot be read",
		"qualify a subcommand and follow its name",
		"-q suppresses a subcommand's informational",
	}
	for _, w := range want {
		if !strings.Contains(usage, w) {
			t.Errorf("usage is missing %q", w)
		}
	}
	// The rows above pin each subcommand's argument syntax; this pins the
	// vocabulary dispatch reads against the text a user is shown, so the two
	// cannot become separate lists (STYLE D3/T4).
	for _, name := range subcommands {
		if !strings.Contains(usage, "goxsd8 "+name+" ") {
			t.Errorf("usage documents no %q subcommand, but dispatch reserves it", name)
		}
	}
}

// TestBuiltBinaryMatrix drives the real executable, following #251: exit
// code, stdout and stderr of the shipped binary, not of a seam over run.
func TestBuiltBinaryMatrix(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "goxsd8")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build -o %s .: %v: %s", bin, err, out)
	}

	for _, args := range helpCases {
		stdout, stderr, code := runBinary(t, bin, args)
		if code != 0 {
			t.Errorf("%s %q = %d, want 0 (stderr %q)", bin, args, code, stderr)
		}
		if stdout != usage {
			t.Errorf("%s %q stdout = %q, want the usage contract", bin, args, stdout)
		}
		if stderr != "" {
			t.Errorf("%s %q stderr = %q, want empty", bin, args, stderr)
		}
	}

	for _, c := range dispatchCases {
		stdout, stderr, code := runBinary(t, bin, c.args)
		if code != 2 {
			t.Errorf("%s %q = %d, want 2", bin, c.args, code)
		}
		if stdout != "" {
			t.Errorf("%s %q stdout = %q, want empty", bin, c.args, stdout)
		}
		want := c.want + "\n" + helpPointer + "\n"
		if stderr != want {
			t.Errorf("%s %q stderr = %q, want %q", bin, c.args, stderr, want)
		}
	}

	// parse's and validate's own outcomes, through the shipped executable
	// rather than through run: the exit code, the stream each answer lands on,
	// and the summary's bytes are what a script sees (#251's shape, #472's
	// subject). The two stdin rows are the only place standard input is
	// reachable at all — run takes its writers as arguments and its reader from
	// the process — and they pin the hint scan's replay as well as the
	// spelling: the scan consumes the document's prefix, so a broken replay
	// would leave the assessment nothing to read.
	for _, c := range []struct {
		args        []string
		stdin       string
		code        int
		stdout      string
		stdoutMatch string
		stderrMatch string
	}{
		{args: []string{"parse", "testdata/order.xsd"}, code: 0, stdout: orderSummary},
		{args: []string{"parse", "-q", "testdata/order.xsd"}, code: 0},
		{args: []string{"parse", "testdata/broken.xsd"}, code: 1, stderrMatch: "[src-resolve]"},
		{args: []string{"parse", "testdata/nosuch.xsd"}, code: 2, stderrMatch: "no such file or directory"},
		{args: []string{"parse"}, code: 2, stderrMatch: "goxsd8: parse: no schema given"},
		{args: []string{"validate", "-schema", orderSchema, validInstance}, code: 0},
		{args: []string{"validate", "-schema", orderSchema, invalidInstance}, code: 1,
			stdoutMatch: invalidInstance + ":5:3: [cvc-attribute]"},
		{args: []string{"validate", "-schema", "testdata/broken.xsd", validInstance}, code: 3,
			stderrMatch: "[src-resolve]"},
		{args: []string{"validate", "-schema", orderSchema, "testdata/nosuch.xml"}, code: 2,
			stderrMatch: "no such file or directory"},
		{args: []string{"validate"}, code: 2, stderrMatch: "goxsd8: validate: no schema given"},
		{args: []string{"validate", "-format", "xml", "-schema", orderSchema, "-"},
			stdin: readFixture(t, validInstance), code: 0},
		{args: []string{"validate", "-format", "xml", "-schema", orderSchema, "-"},
			stdin: readFixture(t, invalidInstance), code: 1, stdoutMatch: "-:5:3: [cvc-attribute]"},
	} {
		stdout, stderr, code := runBinaryStdin(t, bin, c.args, c.stdin)
		if code != c.code {
			t.Errorf("%s %q = %d, want %d (stderr %q)", bin, c.args, code, c.code, stderr)
		}
		if c.stdoutMatch == "" && stdout != c.stdout {
			t.Errorf("%s %q stdout = %q, want %q", bin, c.args, stdout, c.stdout)
		}
		if c.stdoutMatch != "" && !strings.Contains(stdout, c.stdoutMatch) {
			t.Errorf("%s %q stdout = %q, want it to contain %q", bin, c.args, stdout, c.stdoutMatch)
		}
		if c.stderrMatch == "" && stderr != "" {
			t.Errorf("%s %q stderr = %q, want empty", bin, c.args, stderr)
		}
		if c.stderrMatch != "" && !strings.Contains(stderr, c.stderrMatch) {
			t.Errorf("%s %q stderr = %q, want it to contain %q", bin, c.args, stderr, c.stderrMatch)
		}
	}
}

// readFixture returns the bytes of a testdata file, for the rows that feed one
// to the binary's standard input instead of naming it.
func readFixture(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

// runBinary runs bin with args and an empty standard input.
func runBinary(t *testing.T, bin string, args []string) (string, string, int) {
	t.Helper()
	return runBinaryStdin(t, bin, args, "")
}

// runBinaryStdin runs bin with args and stdin, and returns its stdout, stderr
// and exit code. A non-zero exit is an expected outcome here, not a test
// failure — only a failure to run the binary at all is.
func runBinaryStdin(t *testing.T, bin string, args []string, stdin string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		t.Fatalf("%s %q: %v", bin, args, err)
	}
	return stdout.String(), stderr.String(), cmd.ProcessState.ExitCode()
}
