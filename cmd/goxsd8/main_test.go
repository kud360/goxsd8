package main

import (
	"bytes"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// helpCases reach the help path: usage on stdout, nothing on stderr, exit 0.
// The last four pin decisions doc.go's argument vocabulary states rather than
// accidents — a help flag before its subcommand, a help flag alongside an
// unrecognized one, -- with no end-of-options meaning, and a help flag after a
// name that is not in the vocabulary at all.
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
}

// dispatchCases is the diagnosis every non-help invocation earns: a reserved
// name, a name outside the vocabulary, or no subcommand at all. Each is
// followed on stderr by helpPointer. One encoding of the matrix, driven twice
// — through run below and through the built binary in TestBuiltBinaryMatrix.
var dispatchCases = []struct {
	args []string
	want string
}{
	{[]string{"parse", "schema.xsd"}, `goxsd8: parse is not yet implemented`},
	{[]string{"validate", "-schema", "a.xsd", "a.xml"}, `goxsd8: validate is not yet implemented`},
	{[]string{"gen", "-schema", "a.xsd", "-out", "d"}, `goxsd8: gen is not yet implemented`},
	{[]string{"frobnicate"}, `goxsd8: unknown subcommand "frobnicate"`},
	{[]string{"parsee", "a.xsd"}, `goxsd8: unknown subcommand "parsee"`},
	// Case-sensitive matching: a contract name in the wrong case is unknown,
	// not reserved.
	{[]string{"VALIDATE"}, `goxsd8: unknown subcommand "VALIDATE"`},
	// help is not a contract name (doc.go); version is not one either.
	{[]string{"help"}, `goxsd8: unknown subcommand "help"`},
	{[]string{"version"}, `goxsd8: unknown subcommand "version"`},
	// A flag as args[0] is no subcommand at all — not an unknown one. -q is
	// documented and -version is not, and neither is a subcommand name.
	{[]string{"-q"}, noSubcommand},
	{[]string{"-xyz"}, noSubcommand},
	{[]string{"-version"}, noSubcommand},
	{[]string{"-help=true"}, noSubcommand},
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

// TestRunDispatch pins #514: the three diagnoses are distinct, and each is
// the true one for its input.
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

// TestDiagnosesAreDistinct is the property #514 exists for: the answers to
// "you typed a name I do not have", "that one is reserved but unbuilt" and
// "you gave me no subcommand" must not collapse into one another.
func TestDiagnosesAreDistinct(t *testing.T) {
	kinds := []string{
		diagnose("validate"),   // reserved by the contract, unbuilt
		diagnose("frobnicate"), // outside the vocabulary
		diagnose("-q"),         // a flag, so no subcommand at all
	}
	for i, a := range kinds {
		for _, b := range kinds[i+1:] {
			if a == b {
				t.Errorf("two of the three diagnoses collapse to %q", a)
			}
		}
	}
	if got := diagnose("VALIDATE"); got == diagnose("validate") {
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
		"goxsd8 parse <schema.xsd>...",
		"goxsd8 validate -schema <schema.xsd>... <instance>...",
		"goxsd8 gen -schema <schema.xsd> -out <dir>",
		"GOXSD_DEBUG=parser,validate,codec",
		"Implemented today: the help path only.",
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
}

// runBinary runs bin with args and returns its stdout, stderr and exit code.
// A non-zero exit is an expected outcome here, not a test failure — only a
// failure to run the binary at all is.
func runBinary(t *testing.T, bin string, args []string) (string, string, int) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	var ee *exec.ExitError
	if err != nil && !errors.As(err, &ee) {
		t.Fatalf("%s %q: %v", bin, args, err)
	}
	return stdout.String(), stderr.String(), cmd.ProcessState.ExitCode()
}
