package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/doc/comment"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// helpCases reach the help path: usage on stdout, nothing on stderr, exit 0.
// Everything after the four bare spellings pins a decision usage or doc.go's
// argument vocabulary states rather than an accident — a help flag before its
// subcommand, a help flag alongside an unrecognized one, -- given no
// end-of-options meaning by the help scan, even where it ends a subcommand's
// options, a help flag after a name outside the vocabulary
// altogether, and `parse -h`, `validate -h` and `gen -h`: those three own the
// flag.FlagSets in the binary, so they are the rows where a subcommand's own
// flag parsing could take -h away from the help path.
var helpCases = [][]string{
	nil,
	{"-h"},
	{"-help"},
	{"--help"},
	{"validate", "-help"},
	{"-help", "validate"},
	{"-xyz", "-help"},
	{"--", "-help"},
	{"validate", "-schema", "s.xsd", "--", "-help"},
	{"frobnicate", "-h"},
	{"parse", "-h"},
	{"validate", "-h"},
	{"validate", "-schema", "testdata/order.xsd", "-h"},
	{"gen", "-h"},
	{"gen", "-schema", "testdata/order.xsd", "-out", "d", "-h"},
}

// dispatchCases is the diagnosis every non-help invocation that reaches no
// subcommand earns: a name outside the vocabulary, a help request carrying a
// value, a flag before the subcommand it qualifies, or no subcommand at all.
// Each is followed on stderr by helpPointer. One encoding of the matrix, driven
// twice — through run below and through the built binary in
// TestBuiltBinaryMatrix.
//
// parse, validate and gen have no row here: all three are built, and
// parse_test.go, validate_test.go and gen_test.go are their matrices.
var dispatchCases = []struct {
	args []string
	want string
}{
	{[]string{"frobnicate"}, `goxsd8: unknown subcommand "frobnicate"`},
	{[]string{"parsee", "a.xsd"}, `goxsd8: unknown subcommand "parsee"`},
	// Case-sensitive matching: a contract name in the wrong case is unknown,
	// not that subcommand.
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
	{[]string{"-q", "frobnicate"}, noSubcommand},
	// A valued help spelling is an answer of its own (#1189): it is a usage error
	// in every position, and before a subcommand it names the three spellings
	// rather than a rewrite that fails there too. The first row expected
	// noSubcommand until this landed.
	{[]string{"-help=true"}, helpNotAFlagValue},
	{[]string{"-h=1"}, helpNotAFlagValue},
	{[]string{"-help=true", "parse", "a.xsd"}, helpNotAFlagValue},
	{[]string{"-h=1", "parse", "a.xsd"}, helpNotAFlagValue},
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

// TestRunDispatch pins #472 and #1189: the four diagnoses are distinct, and
// each is the true one for its input. It supersedes TestDiagnosesAreDistinct
// (removed, #999): that test asserted the same distinctness by comparing
// diagnose's rendered strings, which cannot fail on a collapse between two
// branches whose diagnosis interpolates the input argument —
// unknownSubcommandFmt and leadingFlagFmt both do, so two inputs those branches
// misclassify as the same kind still render different strings. It stays armed
// against a collapse into noSubcommand, a bare constant with nothing to
// interpolate — which is exactly the collapse a pre-#472 diagnose regresses to,
// and dispatchCases below still catches it.
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

// The usage: lines a subcommand's diagnosis of the command line's shape prints
// (#3333), held as literals so that a test comparing stderr against them fails
// when usage loses or changes the synopsis line: an expectation read through
// synopsis would follow usage wherever it went (#3110).
const (
	parseUsageLine    = "usage: goxsd8 parse [-q] [-v] [-strict] <schema.xsd>..."
	validateUsageLine = "usage: goxsd8 validate [-q] [-v] [-strict] [-format xml|json|ber] [-no-hints] -schema <schema.xsd> [-schema <s2>]... <instance>..."
	genUsageLine      = "usage: goxsd8 gen [-q] [-v] [-strict] -schema <schema.xsd> -out <dir> [-schema <s2> -out <d2>]..."

	// genShapeTail is what follows a gen diagnosis of the command line's
	// shape on stderr, short of the final newline.
	genShapeTail = "\n" + genUsageLine + "\n" + helpPointer
)

// TestSubcommandDiagnosisPrintsSynopsis pins #3110: a diagnosis of the command
// line's shape charged inside a subcommand prints three lines on stderr, the
// diagnosis, usage: and that subcommand's synopsis line, then helpPointer.
// dispatchCases pins the diagnoses charged before a subcommand is known, which
// print no synopsis; TestArgumentFaultPrintsOneLine pins the faults the shape
// did not cause, which print neither (#3333).
func TestSubcommandDiagnosisPrintsSynopsis(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"validate", "x.xml"}, "goxsd8: validate: no schema given\n" + validateUsageLine + "\n" + helpPointer + "\n"},
		{[]string{"validate", "testdata/order-valid.xml", "-schema", "testdata/order.xsd"},
			fmt.Sprintf(flagAfterPositionalFmt, "validate", "-schema") + "\n" + validateUsageLine + "\n" + helpPointer + "\n"},
		{[]string{"gen", "-schema"}, "goxsd8: gen: flag needs an argument: -schema\n" + genUsageLine + "\n" + helpPointer + "\n"},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(c.args, &stdout, &stderr); code != exitUsage {
			t.Errorf("run(%q) = %d, want %d", c.args, code, exitUsage)
		}
		if stdout.Len() != 0 {
			t.Errorf("run(%q) stdout = %q, want empty", c.args, stdout.String())
		}
		if stderr.String() != c.want {
			t.Errorf("run(%q) stderr = %q, want %q", c.args, stderr.String(), c.want)
		}
	}
}

// TestStdoutFaultPrintsOneLine pins the stdout arm of #3333: a failed write of
// parse's summary or of validate's report is charged exit 2 inside the
// subcommand, and its line stands alone, with no synopsis or helpPointer under
// it; validate's counts line follows it, counting the instance faulted.
func TestStdoutFaultPrintsOneLine(t *testing.T) {
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"parse", "testdata/order.xsd"}, "goxsd8: parse: writing the summary for testdata/order.xsd: closed pipe\n"},
		{[]string{"validate", "-schema", "testdata/order.xsd", "testdata/order-invalid.xml"}, "goxsd8: validate: writing the report of testdata/order-invalid.xml: closed pipe\n" +
			"goxsd8: validate: instances=1 clean=0 undecided=0 invalid=0 faulted=1\n"},
	} {
		var stderr bytes.Buffer
		if code := run(c.args, errWriter{}, &stderr); code != exitUsage {
			t.Errorf("run(%q) = %d, want %d", c.args, code, exitUsage)
		}
		if stderr.String() != c.want {
			t.Errorf("run(%q) stderr = %q, want %q", c.args, stderr.String(), c.want)
		}
	}
}

// TestArgumentFaultPrintsOneLine pins #3333: an instance that cannot be opened
// is charged exit 2 with its one diagnosis line and no synopsis or
// helpPointer, so with a second instance after it, the second's report
// follows the line directly. The run with one writer as both stdout and
// stderr is what makes "directly" observable; order-invalid.xml's report is
// read from a run of its own under -q rather than restated. The counts line
// closes each run, the faulted instance counted in it.
func TestArgumentFaultPrintsOneLine(t *testing.T) {
	const fault = "goxsd8: validate: open testdata/nosuch.xml: no such file or directory\n"
	var stdout, stderr bytes.Buffer
	args := []string{"validate", "-schema", "testdata/order.xsd", "testdata/nosuch.xml"}
	if code := run(args, &stdout, &stderr); code != exitUsage {
		t.Errorf("run(%q) = %d, want %d", args, code, exitUsage)
	}
	if stdout.Len() != 0 {
		t.Errorf("run(%q) stdout = %q, want empty", args, stdout.String())
	}
	if want := fault + "goxsd8: validate: instances=1 clean=0 undecided=0 invalid=0 faulted=1\n"; stderr.String() != want {
		t.Errorf("run(%q) stderr = %q, want %q", args, stderr.String(), want)
	}

	var report, reportErr bytes.Buffer
	if code := run([]string{"validate", "-q", "-schema", "testdata/order.xsd", "testdata/order-invalid.xml"}, &report, &reportErr); code != exitInvalid || report.Len() == 0 || reportErr.Len() != 0 {
		t.Fatalf("order-invalid.xml alone = %d with report %q and stderr %q, want %d, a report and no stderr", code, report.String(), reportErr.String(), exitInvalid)
	}
	var both bytes.Buffer
	args = []string{"validate", "-schema", "testdata/order.xsd", "testdata/nosuch.xml", "testdata/order-invalid.xml"}
	if code := run(args, &both, &both); code != exitUsage {
		t.Errorf("run(%q) = %d, want %d", args, code, exitUsage)
	}
	if want := fault + report.String() + "goxsd8: validate: instances=2 clean=0 undecided=0 invalid=1 faulted=1\n"; both.String() != want {
		t.Errorf("run(%q) output = %q, want %q", args, both.String(), want)
	}
	if strings.Contains(both.String(), "usage: ") || strings.Contains(both.String(), helpPointer) {
		t.Errorf("run(%q) output = %q, want no usage: line and no helpPointer", args, both.String())
	}
}

// TestFormatlessInstanceRefusesTheInvocation pins #3358: an instance whose name
// gives no source format refuses the whole validate invocation before any
// instance is assessed, so with one writer as both stdout and stderr the
// output is that instance's formatOf line, one synopsis and one helpPointer,
// and no report of order-invalid.xml before it. Several such instances get a
// line each in argument order above one synopsis. -format names every
// instance's format, and ber is a per-instance capability refusal, so neither
// pin row hoists anything: order-invalid.xml's report precedes the one
// subcommandFault line, and the counts line closes the run. A refusal prints
// no counts line, as usage's validate paragraph states.
func TestFormatlessInstanceRefusesTheInvocation(t *testing.T) {
	const (
		pass    = "; pass -format with one of xml, json or ber\n"
		txtLine = `goxsd8: validate: testdata/order.txt: the extension ".txt" names no source format` + pass
		stdLine = "goxsd8: validate: -: standard input has no extension to name a source format" + pass
		counts  = "goxsd8: validate: instances=2 clean=0 undecided=0 invalid=1 faulted=1\n"
	)
	refusal := validateUsageLine + "\n" + helpPointer + "\n"
	var report, reportErr bytes.Buffer
	if code := run([]string{"validate", "-q", "-schema", orderSchema, invalidInstance}, &report, &reportErr); code != exitInvalid || report.Len() == 0 || reportErr.Len() != 0 {
		t.Fatalf("order-invalid.xml alone = %d with report %q and stderr %q, want %d, a report and no stderr", code, report.String(), reportErr.String(), exitInvalid)
	}
	cases := []struct {
		args []string
		code int
		want string
	}{
		{[]string{"-schema", orderSchema, invalidInstance, "testdata/order.txt"}, exitUsage, txtLine + refusal},
		{[]string{"-schema", orderSchema, invalidInstance, stdinArg}, exitUsage, stdLine + refusal},
		{[]string{"-schema", orderSchema, invalidInstance, "testdata/a.txt", "testdata/b", stdinArg}, exitUsage,
			`goxsd8: validate: testdata/a.txt: the extension ".txt" names no source format` + pass +
				`goxsd8: validate: testdata/b: the extension "" names no source format` + pass +
				stdLine + refusal},
		{[]string{"-format", "xml", "-schema", orderSchema, invalidInstance, "testdata/order.txt"}, exitUsage,
			report.String() + "goxsd8: validate: open testdata/order.txt: no such file or directory\n" + counts},
		{[]string{"-schema", orderSchema, invalidInstance, "testdata/x.ber"}, exitUsage,
			report.String() + "goxsd8: validate: testdata/x.ber: -format ber is reserved by the contract and not yet implemented; only xml and json instances are assessed today\n" + counts},
	}
	for _, c := range cases {
		args := append([]string{"validate"}, c.args...)
		var both bytes.Buffer
		if code := run(args, &both, &both); code != c.code {
			t.Errorf("run(%q) = %d, want %d", args, code, c.code)
		}
		if both.String() != c.want {
			t.Errorf("run(%q) output = %q, want %q", args, both.String(), c.want)
		}
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

// TestExitSeverityRanksEveryCode holds exitSeverity to the const block it
// orders. A code missing from it ranks below every code that is in it, so an
// invocation would answer with the outcome that code should have beaten —
// silently, since nothing about a wrong exit code reaches a stream.
func TestExitSeverityRanksEveryCode(t *testing.T) {
	codes := []int{exitOK, exitInvalid, exitUsage, exitSchema, exitUndecided}
	for _, c := range codes {
		if !slices.Contains(exitSeverity, c) {
			t.Errorf("exit code %d is ranked by nothing in exitSeverity", c)
		}
	}
	if len(exitSeverity) != len(codes) {
		t.Errorf("exitSeverity ranks %d codes, and the const block defines %d", len(exitSeverity), len(codes))
	}

	// The ordering the numbers do not carry, in both argument orders: a batch
	// answers with the instance it decided against, not the one it declined
	// to decide.
	if worse(exitInvalid, exitUndecided) != exitInvalid || worse(exitUndecided, exitInvalid) != exitInvalid {
		t.Errorf("worse over %d and %d = %d and %d, want %d both ways", exitInvalid, exitUndecided, worse(exitInvalid, exitUndecided), worse(exitUndecided, exitInvalid), exitInvalid)
	}
	if worse(exitOK, exitUndecided) != exitUndecided {
		t.Errorf("worse(%d, %d) = %d, want %d: undecided is not a clean run", exitOK, exitUndecided, worse(exitOK, exitUndecided), exitUndecided)
	}
	if worse(exitSchema, exitUsage) != exitSchema {
		t.Errorf("worse(%d, %d) = %d, want %d: exitSeverity ranks exitSchema last only so that every code has a rank", exitSchema, exitUsage, worse(exitSchema, exitUsage), exitSchema)
	}
}

// usageRankRow is a row of usage's exit-code table: its code and its rank,
// a number or none.
var usageRankRow = regexp.MustCompile(`(?m)^  (\d)     (\d|none)  `)

// usagePrecedence is the one-line reading of the rank column under the table:
// the ranked codes, highest first, then the unranked one.
var usagePrecedence = regexp.MustCompile(`Precedence, by that rank: (\d) over (\d) over (\d) over (\d), with validate's (\d) answering alone`)

// TestUsagePrecedenceReadsTheRanks holds usage's precedence line to both the
// rank column it reads and exitSeverity, which worse orders by (#2967): each
// code the line puts over the next one must win over it through worse, in both
// argument orders, and must hold the next rank up in the table. The unranked
// code is exitSchema, which the line ranks against nothing.
func TestUsagePrecedenceReadsTheRanks(t *testing.T) {
	ranks := map[int]string{}
	for _, m := range usageRankRow.FindAllStringSubmatch(usage, -1) {
		code, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("rank row %q: %v", m[0], err)
		}
		ranks[code] = m[2]
	}
	if len(ranks) != len(exitSeverity) {
		t.Fatalf("usage's exit-code table has %d rows, and exitSeverity ranks %d codes", len(ranks), len(exitSeverity))
	}
	for i, code := range exitSeverity {
		want := strconv.Itoa(i + 1)
		if code == exitSchema {
			want = "none"
		}
		if ranks[code] != want {
			t.Errorf("usage's exit-code table ranks %d as %s, and exitSeverity as %s", code, ranks[code], want)
		}
	}

	m := usagePrecedence.FindStringSubmatch(strings.Join(strings.Fields(usage), " "))
	if m == nil {
		t.Fatalf("usage has no line matching %q", usagePrecedence)
	}
	var line []int
	for _, s := range m[1:] {
		code, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("precedence code %q: %v", s, err)
		}
		line = append(line, code)
	}
	ranked, alone := line[:len(line)-1], line[len(line)-1]
	if alone != exitSchema {
		t.Errorf("usage's precedence line leaves %d unranked, and the table leaves %d", alone, exitSchema)
	}
	for i := 0; i+1 < len(ranked); i++ {
		hi, lo := ranked[i], ranked[i+1]
		if worse(hi, lo) != hi || worse(lo, hi) != hi {
			t.Errorf("usage's precedence line puts %d over %d, and worse answers %d and %d", hi, lo, worse(hi, lo), worse(lo, hi))
		}
		hiRank, errHi := strconv.Atoi(ranks[hi])
		loRank, errLo := strconv.Atoi(ranks[lo])
		if errHi != nil || errLo != nil || hiRank != loRank+1 {
			t.Errorf("usage's precedence line puts %d over %d, and the table ranks them %s and %s", hi, lo, ranks[hi], ranks[lo])
		}
	}
	if len(ranked) != len(exitSeverity)-1 {
		t.Errorf("usage's precedence line ranks %d codes, and exitSeverity ranks %d besides exitSchema", len(ranked), len(exitSeverity)-1)
	}
}

// usagePipelineLead opens usage's paragraph for a pipeline author, which runs
// to the first blank line after it.
const usagePipelineLead = "\n  In a pipeline, pass -strict:"

// The lines of that paragraph's case snippet and worked example the test reads:
// an arm's codes, an invocation, the line asking for its exit code, and the
// placeholder standing for the example's working directory.
var usageCaseArm = regexp.MustCompile(`^        ([0-9|]+|\*)\) +(.*?) *;;`)

// usageArmFails matches an arm's action that fails the job.
var usageArmFails = regexp.MustCompile(`^exit [1-9][0-9]*$`)

const (
	usageExampleCommand = "      $ goxsd8 "
	usageExampleStatus  = "      $ echo $?"
	usageExampleDir     = "<readme>"
)

// TestUsagePipelineExample runs the invocations usage's pipeline paragraph
// shows, from testdata/readme as it says, and holds each to the stderr lines
// and the exit code shown under it, <readme> read as that directory's absolute
// path and stdout as empty, as -q leaves it. A shown -strict line must also end
// as strictMarker closes it. The case snippet's numbered arms must name exactly
// the codes of the exit-code table, each once, and a catch-all arm must follow
// them. Each arm's action is held to the table's reading: 0 passes (an empty
// arm), and every other code, the catch-all's included, fails (exit with a
// nonzero code). An edit to a shown code, line, arm or arm's action that the
// binary or the table no longer bears fails it (#3853).
func TestUsagePipelineExample(t *testing.T) {
	_, para, ok := strings.Cut(usage, usagePipelineLead)
	if !ok {
		t.Fatalf("usage has no %q line", strings.TrimPrefix(usagePipelineLead, "\n"))
	}
	para, _, _ = strings.Cut(para, "\n\n")
	lines := strings.Split(para, "\n")

	var armCodes []string
	catchAll := false
	for _, line := range lines {
		m := usageCaseArm.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if catchAll {
			t.Errorf("the pipeline paragraph's case arm %q follows its catch-all arm, which already matched it", m[1])
		}
		fails := usageArmFails.MatchString(m[2])
		if !fails && m[2] != "" {
			t.Errorf("the pipeline paragraph's case arm %q acts %q, which neither passes (empty) nor fails (exit nonzero)", m[1], m[2])
		}
		if m[1] == "*" {
			catchAll = true
			if !fails {
				t.Errorf("the pipeline paragraph's catch-all case arm acts %q, and a code the table does not define fails the job", m[2])
			}
			continue
		}
		for _, code := range strings.Split(m[1], "|") {
			armCodes = append(armCodes, code)
			if pass := code == "0"; pass == fails {
				t.Errorf("the pipeline paragraph's case arm %q acts %q on %s, and the exit-code table passes 0 alone", m[1], m[2], code)
			}
		}
	}
	if !catchAll {
		t.Errorf("the pipeline paragraph's case snippet has no catch-all arm, so a code the table does not define passes the job")
	}
	var tableCodes []string
	for _, m := range usageRankRow.FindAllStringSubmatch(usage, -1) {
		tableCodes = append(tableCodes, m[1])
	}
	slices.Sort(armCodes)
	slices.Sort(tableCodes)
	if !slices.Equal(armCodes, tableCodes) {
		t.Errorf("the pipeline paragraph's case arms name codes %q, and the exit-code table defines %q", armCodes, tableCodes)
	}

	t.Chdir(filepath.Join("..", "..", "testdata", "readme"))
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	ran := 0
	for i := 0; i < len(lines); i++ {
		rest, ok := strings.CutPrefix(lines[i], usageExampleCommand)
		if !ok {
			continue
		}
		args := strings.Fields(rest)
		var shown []string
		j := i + 1
		for j < len(lines) && lines[j] != usageExampleStatus {
			shown = append(shown, strings.TrimPrefix(lines[j], "      "))
			j++
		}
		if j+1 >= len(lines) {
			t.Fatalf("usage's example %q shows no %q line and code after it", args, strings.TrimSpace(usageExampleStatus))
		}
		wantCode, err := strconv.Atoi(strings.TrimSpace(lines[j+1]))
		if err != nil {
			t.Fatalf("usage's example %q shows exit code %q: %v", args, lines[j+1], err)
		}
		i = j + 1
		ran++

		var wantStderr strings.Builder
		for _, line := range shown {
			if strings.Contains(line, "-strict charges") && !strings.HasSuffix(line, strictMarker(true)) {
				t.Errorf("usage's example %q shows %q, which does not end %q", args, line, strictMarker(true))
			}
			wantStderr.WriteString(strings.ReplaceAll(line, usageExampleDir, dir) + "\n")
		}
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)
		if code != wantCode {
			t.Errorf("usage's example %q exits %d, and usage shows %d", args, code, wantCode)
		}
		if stdout.Len() != 0 {
			t.Errorf("usage's example %q prints %q on stdout, and usage shows nothing", args, stdout.String())
		}
		if stderr.String() != wantStderr.String() {
			t.Errorf("usage's example %q prints on stderr\n%q\nand usage shows\n%q", args, stderr.String(), wantStderr.String())
		}
	}
	if ran == 0 {
		t.Errorf("usage's pipeline paragraph shows no %q line to run", strings.TrimSpace(usageExampleCommand))
	}
}

// usageKindsLead opens usage's list of the kinds of declined check, whose
// items follow it one per "- <kind>:" line.
const usageKindsLead = "A declined check is one of these kinds:\n"

// usageKindItem is one item of that list.
var usageKindItem = regexp.MustCompile(`^        - ([^:]+):`)

// TestUsageListsEveryDeclinedKind holds usage's list of the kinds of declined
// check to the bullet labels of validate.Unevaluated's doc, read from its
// source, which stays their owner (#2967): a kind missing from usage, one usage
// adds, or one out of the doc's order fails it. Labels compare with the doc's
// · marks dropped.
func TestUsageListsEveryDeclinedKind(t *testing.T) {
	want := unevaluatedKinds(t)

	_, rest, ok := strings.Cut(usage, usageKindsLead)
	if !ok {
		t.Fatalf("usage has no %q line", strings.TrimSpace(usageKindsLead))
	}
	var got []string
	for _, line := range strings.Split(rest, "\n") {
		if strings.HasPrefix(line, "          ") {
			continue // an item's continuation line
		}
		m := usageKindItem.FindStringSubmatch(line)
		if m == nil {
			break
		}
		got = append(got, m[1])
	}
	if !slices.Equal(got, want) {
		t.Errorf("usage lists the declined kinds %q, and validate.Unevaluated's doc names %q", got, want)
	}
}

// unevaluatedKinds returns the labels of the first list in the doc comment of
// validate.Unevaluated, parsed from validate/validate.go: each item's text up
// to its first colon, · marks dropped.
func unevaluatedKinds(t *testing.T) []string {
	t.Helper()
	path := filepath.Join("..", "..", "validate", "validate.go")
	f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	var doc *ast.CommentGroup
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			if spec.(*ast.TypeSpec).Name.Name == "Unevaluated" {
				doc = gen.Doc
			}
		}
	}
	if doc == nil {
		t.Fatalf("%s declares no documented type Unevaluated", path)
	}
	var p comment.Parser
	for _, block := range p.Parse(doc.Text()).Content {
		list, ok := block.(*comment.List)
		if !ok {
			continue
		}
		var kinds []string
		for _, item := range list.Items {
			var text strings.Builder
			for _, b := range item.Content {
				para, ok := b.(*comment.Paragraph)
				if !ok {
					continue
				}
				for _, s := range para.Text {
					if plain, ok := s.(comment.Plain); ok {
						text.WriteString(string(plain))
					}
				}
			}
			label, _, ok := strings.Cut(text.String(), ":")
			if !ok {
				t.Fatalf("validate.Unevaluated's list item %q has no colon ending its kind", text.String())
			}
			kinds = append(kinds, strings.ReplaceAll(label, "·", ""))
		}
		return kinds
	}
	t.Fatalf("validate.Unevaluated's doc has no list of kinds")
	return nil
}

// update makes TestDocRendersUsage rewrite doc.go's # Usage section instead of
// failing on it; main.go's go:generate directive is what passes it.
var update = flag.Bool("update", false, "rewrite doc.go's # Usage section from usage")

// The generated section of doc.go's package comment runs from the # Usage
// heading up to the hand-written heading that follows it.
const (
	docUsageStart = "// # Usage"
	docUsageEnd   = "// # Argument vocabulary"
)

// docUsageBlock renders usage as doc.go's # Usage section, which is the only
// place the mapping between the two lives. The title paragraph is dropped —
// doc.go's opening sentence states it — and the heading line becomes a doc
// comment heading. Every line indented two spaces is a subcommand block and
// renders preformatted, so go doc keeps its layout; every other line is prose
// and renders as it is wrapped in usage, which is why a ragged prose paragraph
// in usage surfaces as a go tool commentwrap finding on doc.go and is fixed by
// reflowing usage, never doc.go.
func docUsageBlock(u string) (string, error) {
	_, rest, ok := strings.Cut(u, "\n\n")
	if !ok {
		return "", errors.New("usage has no title paragraph to drop")
	}
	heading, body, ok := strings.Cut(rest, "\n\n")
	if !ok || strings.Contains(heading, "\n") || !strings.HasSuffix(heading, ":") {
		return "", fmt.Errorf("usage's second paragraph %q is not a one-line heading ending in a colon", heading)
	}
	var b strings.Builder
	b.WriteString("// # " + strings.TrimSuffix(heading, ":") + "\n//\n")
	for _, line := range strings.Split(strings.TrimSuffix(body, "\n"), "\n") {
		switch {
		case line == "":
			b.WriteString("//\n")
		case strings.HasPrefix(line, "  "):
			b.WriteString("//\t" + line[2:] + "\n")
		case strings.HasPrefix(line, " "), strings.HasPrefix(line, "\t"):
			return "", fmt.Errorf("usage line %q is neither prose nor indented two spaces as a subcommand block", line)
		default:
			b.WriteString("// " + line + "\n")
		}
	}
	b.WriteString("//\n")
	return b.String(), nil
}

// TestDocRendersUsage holds doc.go's # Usage section to the rendering of usage,
// so the contract is edited in one place (#1772). It compares the section as
// read from doc.go against docUsageBlock(usage), so an edit to either side
// alone fails it. With -update (go generate) it rewrites the section instead.
func TestDocRendersUsage(t *testing.T) {
	want, err := docUsageBlock(usage)
	if err != nil {
		t.Fatalf("rendering usage: %v", err)
	}
	src, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("reading doc.go: %v", err)
	}
	doc := string(src)
	start := strings.Index(doc, "\n"+docUsageStart) + 1
	end := strings.Index(doc, "\n"+docUsageEnd) + 1
	if start == 0 || end <= start {
		t.Fatalf("doc.go has no %q line followed by a %q line to bound the generated section", docUsageStart, docUsageEnd)
	}
	got := doc[start:end]
	if got == want {
		return
	}
	if *update {
		if err := os.WriteFile("doc.go", []byte(doc[:start]+want+doc[end:]), 0o644); err != nil {
			t.Fatalf("writing doc.go: %v", err)
		}
		return
	}
	gotLines, wantLines := strings.Split(got, "\n"), strings.Split(want, "\n")
	i := 0
	for i < len(gotLines) && i < len(wantLines) && gotLines[i] == wantLines[i] {
		i++
	}
	t.Errorf("doc.go's # Usage section is not the rendering of usage: edit usage in main.go and run go generate ./cmd/goxsd8, never doc.go's copy\nfirst difference, section line %d:\n doc.go: %q\n  usage: %q",
		i+1, lineAt(gotLines, i), lineAt(wantLines, i))
}

// lineAt returns lines[i], or a marker when the section ran out first.
func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "(section ended)"
}

// TestUsageCoversContract pins usage against the code it describes, which
// TestDocRendersUsage cannot: that test holds doc.go to usage, and nothing but
// these checks holds usage to dispatch.
func TestUsageCoversContract(t *testing.T) {
	// This pins the vocabulary dispatch reads against the text a user is
	// shown, so the two cannot become separate lists (STYLE D3/T4). Exactly
	// one synopsis line per name: synopsis panics on any other count, and a
	// prose line naming the subcommand does not count (#3110).
	for _, name := range subcommands {
		if got := synopsisLines(name); len(got) != 1 {
			t.Errorf("usage spells %d synopsis lines for %q (%q), want 1: dispatch reserves it, and subcommandUsageError prints its one line", len(got), name, got)
		}
	}
	// jsonsrcLine's routing of jsonsrc's error channel by class, each class's
	// line shape and exit code, which jsonsrcLine renders (#2721, #2893).
	// Matched over the text with its line breaks folded, as the sentences wrap.
	folded := strings.Join(strings.Fields(usage), " ")
	for _, want := range []string{
		"validate routes each by the class jsonsrc gives it",
		fmt.Sprintf("Malformed JSON and content after the top-level object print <loc>: [%s] <message> and exit 1, as malformed XML does", jsonsrcToken),
		"an I/O fault reading the instance prints the same line and exits 1, as an I/O fault reading an XML instance does",
		fmt.Sprintf("A decline prints <loc>: [%s] undecided: <message> and exits 4", jsonsrcToken),
	} {
		if !strings.Contains(folded, want) {
			t.Errorf("usage does not state how a jsonsrc error routes (%q)", want)
		}
	}
	// gen's sharing between pairs couples a later pair's package to an
	// earlier one's, and nothing carries between invocations (gen.go calls
	// codegen.Generate once per run), so the help names both (#3289).
	for _, want := range []string{
		"The later package imports the earlier by its import path, so deleting or moving the earlier pair's -out breaks the later one's build.",
		"Sharing is among the pairs of one invocation: running gen once per pair declares every type in its own package.",
	} {
		if !strings.Contains(folded, want) {
			t.Errorf("usage does not state gen's sharing coupling or its one-run-per-pair route (%q)", want)
		}
	}
	// validate's sentence on the permanence of its line shape (#3687).
	permanent := "The violation and undecided lines keep this full shape permanently, a delegated cause included: validate has no shorter rendering, a script keys on the first [<rule>], on a violation line the charged rule, and never on one inside <message>, and <message> is UTF-8 and may set a spec term between U+00B7 (·) marks."
	if !strings.Contains(folded, permanent) {
		t.Errorf("usage does not state that the violation line's full shape is permanent and which [<rule>] to key on (%q)", permanent)
	}
	// The one statement that the undecided: literal marks both withholds,
	// which doc.go's hand-written Diagnostic lines section carries (#2893).
	src, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("reading doc.go: %v", err)
	}
	both := "The literal undecided: marks both withholds on purpose, a declined check and a jsonsrc decline, and the token before it, a rule ID or jsonsrc, tells them apart."
	if !strings.Contains(strings.Join(strings.Fields(strings.ReplaceAll(string(src), "\n//", "\n")), " "), both) {
		t.Errorf("doc.go does not state that undecided: marks both withholds (%q)", both)
	}
}

// usageFirstScreen is how many lines of usage, -help's stdout as TestRunHelp
// pins it, the synopses and the flag index stand within (#3866).
const usageFirstScreen = 30

// registeredFlags is every flag the parse, validate and gen flag sets register
// (parse.go, validate.go, gen.go). The help flags run answers before dispatch
// are in none of them.
var registeredFlags = []string{"-q", "-v", "-strict", "-format", "-no-hints", "-schema", "-out"}

// usageSynopsisFlag matches a flag a synopsis line spells, bracketed or bare.
var usageSynopsisFlag = regexp.MustCompile(`(?:^| |\[)(-[a-z][a-z-]*)`)

// usageIndexLine matches a line of usage's flag index by its shape: indented
// four spaces, the flag, the subcommands taking it, its topic and the paragraph
// stating it, each column two or more spaces from the next.
var usageIndexLine = regexp.MustCompile(`^    (-[a-z][a-z-]*) {2,}(` + alternation(subcommands) + `(?: ` + alternation(subcommands) + `)*) {2,}(\S+(?: \S+)*) {2,}(\S+(?: \S+)*)$`)

// TestUsageFirstScreen holds the opening of usage, which -help prints, to what
// a reader types (#3866): every synopsis line within its first
// usageFirstScreen lines, and exactly one flag-index line per registered flag,
// there too, naming exactly the subcommands whose synopsis spells the flag, in
// synopsis order. The subcommand column is read against the synopses, which
// parseUsageLine, validateUsageLine and genUsageLine pin as literals, never
// against the index itself.
func TestUsageFirstScreen(t *testing.T) {
	lines := strings.Split(usage, "\n")
	takes := map[string][]string{}
	for _, name := range subcommands {
		line := synopsis(name)
		if i := slices.Index(lines, synopsisIndent+line); i >= usageFirstScreen {
			t.Errorf("usage's synopsis line for %s is line %d, past the first %d", name, i+1, usageFirstScreen)
		}
		for _, m := range usageSynopsisFlag.FindAllStringSubmatch(line, -1) {
			if !slices.Contains(registeredFlags, m[1]) {
				t.Errorf("%s's synopsis spells %s, which no flag set registers", name, m[1])
			}
			if !slices.Contains(takes[m[1]], name) {
				takes[m[1]] = append(takes[m[1]], name)
			}
		}
	}
	indexed := map[string]int{}
	for i, line := range lines {
		m := usageIndexLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if !slices.Contains(registeredFlags, m[1]) {
			t.Errorf("usage line %d indexes %s, which no flag set registers", i+1, m[1])
			continue
		}
		indexed[m[1]]++
		if i >= usageFirstScreen {
			t.Errorf("usage's flag-index line for %s is line %d, past the first %d", m[1], i+1, usageFirstScreen)
		}
		if got := strings.Fields(m[2]); !slices.Equal(got, takes[m[1]]) {
			t.Errorf("usage's flag-index line for %s names %q, and the synopses spelling it are %q", m[1], got, takes[m[1]])
		}
	}
	for _, f := range registeredFlags {
		if len(takes[f]) == 0 {
			t.Errorf("no synopsis spells %s, which a flag set registers", f)
		}
		if indexed[f] != 1 {
			t.Errorf("usage has %d flag-index lines for %s, want 1", indexed[f], f)
		}
	}
}

// stderrBlockHeading opens usage's Stderr lines block. Its template lines are
// the ones indented four spaces below it, up to the blank line closing it; the
// lines indented further say what tells a template apart and are not read.
const stderrBlockHeading = "  Stderr lines:"

// stderrPlaceholders is what each <placeholder> of a template stands for, as a
// regular expression, but the two placeholderPattern reads from their own
// encodings. Every other character of a template is literal. <loc> is ?, or an
// absolute path, the Paths paragraph's location of a schema document, with or
// without :<line>:<col>.
var stderrPlaceholders = map[string]string{
	"<loc>":     `(?:\?(?::[0-9]+){0,2}|/.*?)`,
	"<rule>":    `[^\]\s]+`,
	"<message>": `.*`,
	"<n>":       `[0-9]+`,
	"<time>":    `\S+`,
	"<level>":   `[A-Z]+(?:[+-][0-9]+)?`,
}

// The shapes of usage's Stderr lines block, by the position of their template
// there. TestStderrLineShapes fails when the block holds any other number.
const (
	shapeRejection = iota
	shapeCounts
	shapeSubcommand
	shapePreSubcommand
	shapeUsageLine
	shapeHelpPointer
	shapeRecord
	shapeComponent
	shapeCount
)

// stderrTemplates reads the template lines of usage's Stderr lines block and
// compiles each into a regular expression matching a whole line.
func stderrTemplates(t *testing.T) ([]string, []*regexp.Regexp) {
	t.Helper()
	_, block, ok := strings.Cut(usage, "\n"+stderrBlockHeading)
	if !ok {
		t.Fatalf("usage has no %q line", stderrBlockHeading)
	}
	block, _, _ = strings.Cut(block, "\n\n")
	placeholder := regexp.MustCompile(`<[a-z]+>`)
	var texts []string
	var res []*regexp.Regexp
	for _, line := range strings.Split(block, "\n") {
		text, ok := strings.CutPrefix(line, "    ")
		if !ok || strings.HasPrefix(text, " ") {
			continue
		}
		var re strings.Builder
		re.WriteString("^")
		last := 0
		for _, at := range placeholder.FindAllStringIndex(text, -1) {
			name := text[at[0]:at[1]]
			pattern, ok := placeholderPattern(name)
			if !ok {
				t.Fatalf("usage's Stderr lines template %q has placeholder %s, which stderrPlaceholders does not read", text, name)
			}
			re.WriteString(regexp.QuoteMeta(text[last:at[0]]) + pattern)
			last = at[1]
		}
		re.WriteString(regexp.QuoteMeta(text[last:]) + "$")
		texts = append(texts, text)
		res = append(res, regexp.MustCompile(re.String()))
	}
	return texts, res
}

// placeholderPattern returns the regular expression a template's placeholder
// stands for: <subcommand> any name of subcommands, <synopsis> any of their
// synopsis lines, and every other one stderrPlaceholders' entry.
func placeholderPattern(name string) (string, bool) {
	switch name {
	case "<subcommand>":
		return alternation(subcommands), true
	case "<synopsis>":
		var lines []string
		for _, s := range subcommands {
			lines = append(lines, synopsis(s))
		}
		return alternation(lines), true
	}
	pattern, ok := stderrPlaceholders[name]
	return pattern, ok
}

// alternation matches exactly one of words.
func alternation(words []string) string {
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = regexp.QuoteMeta(w)
	}
	return "(?:" + strings.Join(quoted, "|") + ")"
}

// TestStderrLineShapes pins usage's Stderr lines block to what run prints on
// stderr (#3798). Each row drives run once and names the shape of every stderr
// line, the first template in the block's order that matches it whole, with a
// run of lines of one shape named once. A row fails when a printed line
// changes shape without the block changing with it, and when a line falls
// through to the component's message, the template matching every line, in a
// row naming another shape. Together the rows reach every template.
func TestStderrLineShapes(t *testing.T) {
	texts, templates := stderrTemplates(t)
	if len(templates) != shapeCount {
		t.Fatalf("usage's Stderr lines block holds %d templates (%q), want %d", len(templates), texts, shapeCount)
	}
	dir := t.TempDir()
	// A <complexType> extending one whose {final} holds extension is charged
	// cos-ct-extends at no known location, which prints <loc> as ?.
	unlocated := filepath.Join(dir, "final.xsd")
	if err := os.WriteFile(unlocated, []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:complexType name="A" final="extension"/><xs:complexType name="B"><xs:complexContent><xs:extension base="A"/></xs:complexContent></xs:complexType></xs:schema>`), 0o644); err != nil {
		t.Fatal(err)
	}
	// A ber instance argument is refused by name before it is opened, so its
	// line opens with the argument as typed: spelled as the counts line's
	// opening, it is the note the counts template must not take.
	const countsSpelled = "instances=1 clean=0 undecided=0 invalid=0 faulted=0.ber"
	// An instance hinting a document that declares another namespace than the
	// one its hint pairs: the note dropping its hints quotes the rejection, so
	// a <loc>: [<rule>] sits inside a goxsd8: validate: line.
	if err := os.WriteFile(filepath.Join(dir, "actual.xsd"), []byte(`<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema" targetNamespace="urn:actual"><xs:element name="note" type="xs:string"/></xs:schema>`), 0o644); err != nil {
		t.Fatal(err)
	}
	valid, err := os.ReadFile(validInstance)
	if err != nil {
		t.Fatal(err)
	}
	mispaired := filepath.Join(dir, "mispaired.xml")
	body := strings.Replace(string(valid), `createdBy="ada"`, `createdBy="ada" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:wrong actual.xsd"`, 1)
	if err := os.WriteFile(mispaired, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	var reached [shapeCount]bool
	for _, c := range []struct {
		name string
		args []string
		code int
		want []int
	}{
		{"parse rejection", []string{"parse", brokenSchema}, exitInvalid, []int{shapeRejection}},
		{"validate rejection", []string{"validate", "-schema", brokenSchema, validInstance}, exitSchema, []int{shapeRejection}},
		{"rejection at ?", []string{"parse", unlocated}, exitInvalid, []int{shapeRejection}},
		{"component's message", []string{"parse", "testdata/notschema.xml"}, exitInvalid, []int{shapeComponent}},
		{"note, then component's message", []string{"parse", "testdata/notdir-include.xsd"}, exitInvalid, []int{shapeSubcommand, shapeComponent}},
		{"note", []string{"parse", shortSchema}, exitOK, []int{shapeSubcommand}},
		{"I/O fault, then counts", []string{"validate", "-schema", orderSchema, "testdata/nosuch.xml"}, exitUsage, []int{shapeSubcommand, shapeCounts}},
		{"instances= note, then counts", []string{"validate", "-schema", orderSchema, countsSpelled}, exitUsage, []int{shapeSubcommand, shapeCounts}},
		{"note quoting a rejection, then counts", []string{"validate", "-schema", orderSchema, mispaired}, exitOK, []int{shapeSubcommand, shapeCounts}},
		{"clean, counts alone", []string{"validate", "-schema", orderSchema, validInstance}, exitOK, []int{shapeCounts}},
		{"-q I/O fault, no counts", []string{"validate", "-q", "-schema", orderSchema, "testdata/nosuch.xml"}, exitUsage, []int{shapeSubcommand}},
		{"-q clean, silent", []string{"validate", "-q", "-schema", orderSchema, validInstance}, exitOK, nil},
		{"parse -v records", []string{"parse", "-v", "-q", orderSchema}, exitOK, []int{shapeRecord}},
		{"validate -v records, then counts", []string{"validate", "-v", "-schema", orderSchema, validInstance}, exitOK, []int{shapeRecord, shapeCounts}},
		{"gen refusal", []string{"gen", "-schema", orderSchema, "-out", filepath.Join(dir, "my-pkg")}, exitUsage, []int{shapeSubcommand}},
		{"subcommand usage diagnosis", []string{"parse"}, exitUsage, []int{shapeSubcommand, shapeUsageLine, shapeHelpPointer}},
		{"pre-subcommand diagnosis", []string{"frobnicate"}, exitUsage, []int{shapePreSubcommand, shapeHelpPointer}},
		{"misplaced flag", []string{"-q", "parse", orderSchema}, exitUsage, []int{shapePreSubcommand, shapeHelpPointer}},
	} {
		for _, shape := range c.want {
			reached[shape] = true
		}
		var stdout, stderr bytes.Buffer
		if code := run(c.args, &stdout, &stderr); code != c.code {
			t.Errorf("%s: run(%q) = %d, want %d", c.name, c.args, code, c.code)
		}
		var got []int
		for line := range strings.Lines(stderr.String()) {
			line = strings.TrimSuffix(line, "\n")
			shape := slices.IndexFunc(templates, func(re *regexp.Regexp) bool { return re.MatchString(line) })
			if len(got) == 0 || got[len(got)-1] != shape {
				got = append(got, shape)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: run(%q) stderr lines are shapes %v, want %v (templates %q)\nstderr:\n%s", c.name, c.args, got, c.want, texts, stderr.String())
		}
	}
	for shape, ok := range reached {
		if !ok {
			t.Errorf("no row prints a line of usage's Stderr lines template %q", texts[shape])
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
	// subject). The six stdin rows are the only place standard input is
	// reachable at all — run takes its writers as arguments and its reader from
	// the process — and the two -format xml ones pin the hint scan's replay as
	// well as the spelling: the scan consumes the document's prefix, so a
	// broken replay would leave the assessment nothing to read.
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
		// The misplaced flag, through the shipped binary: the summary -q asked
		// to suppress must not reach stdout ahead of the diagnosis (#1290).
		{args: []string{"parse", "testdata/order.xsd", "-q"}, code: 2,
			stderrMatch: "-q stands after a positional argument"},
		{args: []string{"validate", "-schema", orderSchema, validInstance}, code: 0,
			stderrMatch: "goxsd8: validate: instances=1 clean=1 undecided=0 invalid=0 faulted=0\n"},
		{args: []string{"validate", "-q", "-schema", orderSchema, validInstance}, code: 0},
		{args: []string{"validate", "-schema", orderSchema, invalidInstance}, code: 1,
			stdoutMatch: invalidInstance + ":5:11: [cvc-attribute]",
			stderrMatch: "goxsd8: validate: instances=1 clean=0 undecided=0 invalid=1 faulted=0\n"},
		{args: []string{"validate", "-schema", "testdata/broken.xsd", validInstance}, code: 3,
			stderrMatch: "[src-resolve]"},
		{args: []string{"validate", "-schema", orderSchema, "testdata/nosuch.xml"}, code: 2,
			stderrMatch: "no such file or directory"},
		{args: []string{"validate"}, code: 2, stderrMatch: "goxsd8: validate: no schema given"},
		{args: []string{"validate", "-q", "-format", "xml", "-schema", orderSchema, "-"},
			stdin: readFixture(t, validInstance), code: 0},
		{args: []string{"validate", "-q", "-format", "xml", "-schema", orderSchema, "-"},
			stdin: readFixture(t, invalidInstance), code: 1, stdoutMatch: "-:5:11: [cvc-attribute]"},
		// - names no source format once xml and json are both assessed, so
		// without -format it exits 2 asking for one, whatever it holds (#2403).
		{args: []string{"validate", "-schema", orderSchema, "-"},
			stdin: readFixture(t, validInstance), code: 2, stderrMatch: "-: standard input has no extension to name a source format"},
		{args: []string{"validate", "-schema", orderSchema, "-"},
			stdin: readFixture(t, invalidJSON), code: 2, stderrMatch: "-: standard input has no extension to name a source format"},
		{args: []string{"validate", "-q", "-format", "json", "-schema", orderSchema, "-"},
			stdin: readFixture(t, validJSON), code: 0},
		{args: []string{"validate", "-q", "-format", "json", "-schema", orderSchema, "-"},
			stdin: readFixture(t, invalidJSON), code: 1, stdoutMatch: "-:1:19: [cvc-attribute]"},
		// gen's outcomes through the shipped binary; gen_test.go is the matrix.
		{args: []string{"gen", "-schema", flatFixture, "-out", filepath.Join(moduleTempDir(t), "flat")}, code: 0},
		{args: []string{"gen", "-schema", "testdata/broken.xsd", "-out", filepath.Join(moduleTempDir(t), "broken")}, code: 1,
			stderrMatch: "[src-resolve]"},
		{args: []string{"gen", "-schema", flatFixture}, code: 2,
			stderrMatch: "goxsd8: gen: -schema " + flatFixture + " stands without its -out"},
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
