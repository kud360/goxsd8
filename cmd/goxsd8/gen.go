package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/kud360/goxsd8/codegen"
)

// The two flags a gen pair is spelled with, named once for the flag set and
// for pairTargets' diagnoses.
const (
	schemaFlag = "schema"
	outFlag    = "out"
)

// genFlag is one occurrence of -schema or -out, in argument order. The flag
// package hands each repeatable flag its values without the interleaving, and
// the interleaving is the pairing, so both flags record into one sequence.
type genFlag struct {
	name, value string
}

// genPair is one -schema/-out pair as the user spelled it.
type genPair struct {
	schema, out string
}

// runGen implements `goxsd8 gen`: it compiles the -schema of each -schema/-out
// pair alone, as parse compiles an argument, and, when every one compiled,
// calls codegen.Generate once with one Target per pair in argument order.
// args excludes the subcommand name itself.
//
// runGen restates none of Generate's own checks (STYLE D3): it routes
// Generate's one error by errors.ErrUnsupported alone, a decline to exit 4 and
// anything else to 2, each line naming the pair the error came from.
//
// The flags are parsed here rather than by run, for the reason runParse states.
func runGen(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("gen", flag.ContinueOnError)
	// Discarding the flag set's output silences its own usage rendering, which
	// would print a second, narrower contract next to doc.go's; helpPointer
	// names the real one instead.
	flags.SetOutput(io.Discard)
	var occurrences []genFlag
	for _, name := range []string{schemaFlag, outFlag} {
		flags.Func(name, "one half of a -schema/-out pair", func(v string) error {
			occurrences = append(occurrences, genFlag{name, v})
			return nil
		})
	}
	verbose := flags.Bool("v", false, "log the parser's assembly at debug level to stderr")
	// -q is defined because the contract makes it common to every subcommand,
	// and is read by nothing: gen writes no informational output.
	_ = flags.Bool("q", false, "accepted; gen has no informational output to suppress")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return usageError(stderr, fmt.Sprintf(helpNotAFlagValueFmt, "gen"))
		}
		return usageError(stderr, fmt.Sprintf("goxsd8: gen: %v", err))
	}
	if arg, ok := flagShapedIn(flags.Args()); ok {
		return usageError(stderr, fmt.Sprintf(flagAfterPositionalFmt, "gen", arg))
	}
	if len(flags.Args()) > 0 {
		return usageError(stderr, fmt.Sprintf("goxsd8: gen: %s: gen takes no positional argument, only -schema/-out pairs", flags.Arg(0)))
	}
	pairs, err := pairTargets(occurrences)
	if err != nil {
		return usageError(stderr, fmt.Sprintf("goxsd8: gen: %v", err))
	}

	// A nil logger selects parser's silent default, for the reason parse.go's
	// GAP(cmd) marker records (#1185).
	var log *slog.Logger
	if *verbose {
		log = slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}

	// Every pair is compiled and reported before Generate is called, as
	// usage's gen block states.
	code := exitOK
	targets := make([]codegen.Target, 0, len(pairs))
	for _, p := range pairs {
		schema, c := compileArgument("gen", p.schema, log, stderr)
		code = worse(code, c)
		targets = append(targets, codegen.Target{
			Schema:  schema,
			Dir:     p.out,
			Package: filepath.Base(filepath.Clean(p.out)),
		})
	}
	if code != exitOK {
		return code
	}
	err = codegen.Generate(targets)
	if err == nil {
		return exitOK
	}
	line := generateLine(err, pairs)
	if errors.Is(err, errors.ErrUnsupported) {
		// A failed stderr write cannot change the outcome: the exit code is
		// settled either way, and stderr is the only channel this line has.
		_, _ = fmt.Fprintln(stderr, line)
		return exitUndecided
	}
	return usageError(stderr, line)
}

// pairTargets reads the -schema/-out occurrences as pairs, each -schema
// followed by its own -out, and fails on the first occurrence that breaks the
// alternation, or on none at all.
func pairTargets(occurrences []genFlag) ([]genPair, error) {
	var pairs []genPair
	for i := 0; i < len(occurrences); i += 2 {
		first := occurrences[i]
		if first.name != schemaFlag {
			return nil, fmt.Errorf("-%s %s stands without a -%s before it", outFlag, first.value, schemaFlag)
		}
		if i+1 == len(occurrences) || occurrences[i+1].name != outFlag {
			return nil, fmt.Errorf("-%s %s stands without its -%s", schemaFlag, first.value, outFlag)
		}
		pairs = append(pairs, genPair{schema: first.value, out: occurrences[i+1].value})
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no -%s/-%s pair given", schemaFlag, outFlag)
	}
	return pairs, nil
}

// generatePrefix opens every error codegen.Generate returns, followed by the
// failing target's index and ": ". Generate's doc promises the index, and this
// spelling, like earlierDir's, is its code's (codegen/generate.go):
// TestGenDeclines and TestGenUsageAndIOFaults pin the mapping against it.
const generatePrefix = "codegen: target "

// earlierDir is the one other place a Generate error names a target by index:
// a Dir equal, after filepath.Clean, to an earlier target's.
var earlierDir = regexp.MustCompile(`target (\d+)'s Dir`)

// generateLine renders an error from codegen.Generate as gen's line, with each
// target index Generate names replaced by the pair it was built from: an index
// means nothing to a user who spelled pairs. An error not in Generate's shape
// is printed whole behind the lead-in, so nothing it says is lost.
func generateLine(err error, pairs []genPair) string {
	msg := err.Error()
	after, prefixed := strings.CutPrefix(msg, generatePrefix)
	index, rest, indexed := strings.Cut(after, ": ")
	i, convErr := strconv.Atoi(index)
	if !prefixed || !indexed || convErr != nil || i < 0 || i >= len(pairs) {
		return "goxsd8: gen: " + msg
	}
	rest = earlierDir.ReplaceAllStringFunc(rest, func(m string) string {
		j, err := strconv.Atoi(earlierDir.FindStringSubmatch(m)[1])
		if err != nil || j < 0 || j >= len(pairs) {
			return m
		}
		return "the Dir of " + pairs[j].label()
	})
	return "goxsd8: gen: " + pairs[i].label() + ": " + rest
}

// label names a pair on a diagnosis as its user spelled it.
func (p genPair) label() string {
	return fmt.Sprintf("-%s %s -%s %s", schemaFlag, p.schema, outFlag, p.out)
}
