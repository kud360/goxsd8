// Command decodecensus measures codec over the W3C suite: of the instance
// cases whose schema codegen generates, how many codec.Decode decodes into the
// type codegen emitted for the document element's declaration, how many it
// declines, counted by the first construct each declines and largest first,
// and how many fail any other way. It is M10's measure, as `go tool
// gencensus` is M9's.
//
// Package codeccensus owns the population, the assembly and the drivers the
// census runs; this command owns the report. The driver classifies each
// result as package assess states, and validates the instance itself, so that
// "decoded where validate rejects" reads its own validation, never Decode's
// error.
//
// Usage:
//
//	go tool decodecensus [-paths] [dir]
//
// dir defaults to the suite at [suiteroot.Default]. -paths adds, after the
// caveats, the IDs of the cases under each declined construct, in the
// table's order, then under each listed count: rejected codegen's own
// type, decoded where validate rejects, and the placeholder's
// ErrTypeMismatch. A case is named by its conformance.CatalogEntry ID, and
// each list is in byte order of those IDs, whichever driver assessed each
// case. An absent suite is a supported mode, not a failure: the submodule is
// absent in a fresh container (#659), so the tool says so, names the command
// that initializes it, and exits 0. It exits 0 for a completed census
// whatever it finds, and 2 for an operational error.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/kud360/goxsd8/tools/internal/codeccensus"
	"github.com/kud360/goxsd8/tools/internal/codeccensus/assess"
	"github.com/kud360/goxsd8/tools/internal/suiteroot"
)

// usage is printed for any argument the tool cannot act on.
const usage = "usage: decodecensus [-paths] [dir]; -paths lists the cases under each construct and each listed count"

func main() {
	if err := run(os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "decodecensus: %v\n", err)
		os.Exit(2)
	}
}

// run drives the census end to end, codeccensus.Walk's work and then the
// report. It takes its destination and arguments so a test can drive the
// whole command without a subprocess of its own.
func run(stdout io.Writer, args []string) error {
	flags := flag.NewFlagSet("decodecensus", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	withPaths := flags.Bool("paths", false, "list the cases under each construct and each listed count")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("%w\n%s", err, usage)
	}
	if flags.NArg() > 1 {
		return errors.New(usage)
	}
	root := suiteroot.Default
	if flags.NArg() == 1 {
		root = flags.Arg(0)
	}
	var rep report
	pop, present, err := codeccensus.Walk(stdout, "decodecensus", root, rep.merge)
	if !present || err != nil {
		return err
	}
	rep.Population = pop
	return printReport(stdout, rep, *withPaths)
}

// report is what the census found: Walk's population and the drivers'
// merged summary. The declined count is not stored: it is the sum of the
// tallies' case lists' lengths (STYLE D3).
type report struct {
	codeccensus.Population
	sum assess.Summary
}

// merge adds one driver's summary to r: its decline table by
// codeccensus.MergeTallies, and its listed counts' case lists concatenated.
func (r *report) merge(s assess.Summary) {
	r.sum.Decoded += s.Decoded
	r.sum.Undecided += s.Undecided
	r.sum.FailedOther += s.FailedOther
	r.sum.Rejected = append(r.sum.Rejected, s.Rejected...)
	r.sum.Differential = append(r.sum.Differential, s.Differential...)
	r.sum.PlaceholderMismatch = append(r.sum.PlaceholderMismatch, s.PlaceholderMismatch...)
	r.sum.Declines = codeccensus.MergeTallies(r.sum.Declines, s.Declines)
}

// declined is how many cases Decode declined.
func (r report) declined() int {
	n := 0
	for _, t := range r.sum.Declines {
		n += len(t.Cases)
	}
	return n
}

// caveats are the limits on reading any figure the census prints,
// codeccensus.Caveats among them; the report adds the placeholder's, which
// carries its counts.
var caveats = slices.Concat([]string{
	"Defects: \"" + rejectedLabel + "\" and \"" + differentialLabel + "\" are defects in codec, and any count other than 0 is one to file.",
	"\"failed other\" is no defect count: it holds every Decode error that is no decline, no undecided validation and no type mismatch on codegen's type: a source fault, every instance validate charges, as the suite's invalid instances are meant to be, and an error of Decode's second read after validation.",
	"One decline per case: Decode returns its first decline, so a case counts once, under that construct.",
}, codeccensus.Caveats, []string{
	"No count is a yield: lifting a construct exposes each of its cases' next decline, so its count bounds from above the cases lifting it alone makes decode.",
})

// The labels of the counts -paths lists the cases of, each printed on its
// count's line and over its list.
const (
	rejectedLabel     = "rejected codegen's own type"
	differentialLabel = "decoded where validate rejects"
	placeholderLabel  = "Decode's ErrTypeMismatch on the placeholder"
)

// printReport renders the census: the counts, the declined constructs largest
// first, the caveats, and with withPaths the cases under each. A bufio.Writer
// keeps its first write error and Flush returns it, so checking Flush checks
// every write.
func printReport(dst io.Writer, rep report, withPaths bool) error {
	w := bufio.NewWriter(dst)
	_, _ = fmt.Fprintf(w, "decodecensus: %d instance case(s) under %s\n", rep.Considered, rep.Root)
	_, _ = fmt.Fprintf(w, "  schema generates: %d\n", rep.Generates)
	_, _ = fmt.Fprintf(w, "  decoded: %d\n", rep.sum.Decoded)
	_, _ = fmt.Fprintf(w, "  declined: %d\n", rep.declined())
	_, _ = fmt.Fprintf(w, "  undecided: %d\n", rep.sum.Undecided)
	_, _ = fmt.Fprintf(w, "  %s: %d\n", rejectedLabel, len(rep.sum.Rejected))
	_, _ = fmt.Fprintf(w, "  failed other: %d\n", rep.sum.FailedOther)
	_, _ = fmt.Fprintf(w, "  %s: %d\n", differentialLabel, len(rep.sum.Differential))
	order := codeccensus.Ranked(rep.sum.Declines)
	_, _ = fmt.Fprintf(w, "\n=== First decline, by construct: %d construct(s) ===\n", len(order))
	for _, t := range order {
		_, _ = fmt.Fprintf(w, "  %6d  %s\n", len(t.Cases), t.Construct)
	}
	_, _ = fmt.Fprintln(w, "\n=== Caveats ===")
	for _, c := range caveats {
		_, _ = fmt.Fprintf(w, "  - %s\n", c)
	}
	_, _ = fmt.Fprintf(w, "  - %s\n", placeholderCaveat(rep))
	if withPaths {
		printPaths(w, rep, order)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("writing the report: %w", err)
	}
	return nil
}

// printPaths renders the -paths sections: each construct's cases, in order's
// order, then each listed count's, each list as the package doc's -paths
// states.
func printPaths(w *bufio.Writer, rep report, order []assess.Tally) {
	_, _ = fmt.Fprintln(w, "\n=== Cases, by construct ===")
	for _, t := range order {
		codeccensus.PrintCases(w, t.Construct, t.Cases)
	}
	_, _ = fmt.Fprintln(w, "\n=== Cases, by listed count ===")
	codeccensus.PrintCases(w, rejectedLabel, rep.sum.Rejected)
	codeccensus.PrintCases(w, differentialLabel, rep.sum.Differential)
	codeccensus.PrintCases(w, placeholderLabel, rep.sum.PlaceholderMismatch)
}

// placeholderCaveat is the caveat on the cases decoded into the placeholder.
func placeholderCaveat(rep report) string {
	return fmt.Sprintf("Placeholder: %d case(s) decode into a placeholder struct, not codegen's type: their document element is not well-formed, "+
		"names no top-level declaration with {abstract} false, or, for %d of them, names one the census found no single generated type for. "+
		"Decode's ErrTypeMismatch on the placeholder, %d case(s), counts under \"failed other\", never under \""+rejectedLabel+"\".",
		rep.Placeholder, rep.Unresolved, len(rep.sum.PlaceholderMismatch))
}
