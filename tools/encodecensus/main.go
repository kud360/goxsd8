// Command encodecensus measures codec's encode over the W3C suite: of the
// instance cases `go tool decodecensus` decodes, how many codec.Append
// encodes into a document that validation accepts and that decodes back to
// the same value, how many it declines, counted by the first construct each
// declines and largest first, and how many fail each other way, every one of
// them a defect. It is M10's measure of encode, as decodecensus is of decode.
//
// Package codeccensus owns the population, the assembly and the drivers the
// census runs; this command owns the report. assess.DriveEncode's doc
// states which cases the census round-trips and how, assess.EncodeSummary's
// the row each lands in, the report printing one count per row, and the
// round trip's equality is assess's same.
//
// Usage:
//
//	go tool encodecensus [-paths] [dir]
//
// dir defaults to the suite at [suiteroot.Default]. -paths adds, after the
// caveats, the IDs of the cases under each declined construct, in the
// table's order, then under each of the four defect counts. A case is named
// by its conformance.CatalogEntry ID, and each list is in byte order of
// those IDs, whichever driver assessed each case. An absent suite is a
// supported mode, not a failure: the tool says so, names the command that
// initializes it, and exits 0. It exits 0 for a completed census whatever it
// finds, and 2 for an operational error.
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
const usage = "usage: encodecensus [-paths] [dir]; -paths lists the cases under each construct and each defect count"

func main() {
	if err := run(os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "encodecensus: %v\n", err)
		os.Exit(2)
	}
}

// run drives the census end to end, codeccensus.Walk's work and then the
// report. It takes its destination and arguments so a test can drive the
// whole command without a subprocess of its own.
func run(stdout io.Writer, args []string) error {
	flags := flag.NewFlagSet("encodecensus", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	withPaths := flags.Bool("paths", false, "list the cases under each construct and each defect count")
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
	pop, present, err := codeccensus.Walk(stdout, "encodecensus", root, rep.merge)
	if !present || err != nil {
		return err
	}
	rep.Population = pop
	return printReport(stdout, rep, *withPaths)
}

// report is what the census found: Walk's population and the drivers'
// merged summary. Neither the decoded nor the declined count is stored: the
// one is the sum of every row, the other of the tallies' case lists'
// lengths (STYLE D3).
type report struct {
	codeccensus.Population
	sum assess.EncodeSummary
}

// merge adds one driver's summary to r: its decline table by
// codeccensus.MergeTallies, and its defect counts' case lists concatenated.
func (r *report) merge(s assess.EncodeSummary) {
	r.sum.Encoded += s.Encoded
	r.sum.AppendFailed = append(r.sum.AppendFailed, s.AppendFailed...)
	r.sum.ReValidateRejected = append(r.sum.ReValidateRejected, s.ReValidateRejected...)
	r.sum.ReDecodeFailed = append(r.sum.ReDecodeFailed, s.ReDecodeFailed...)
	r.sum.RoundTripUnequal = append(r.sum.RoundTripUnequal, s.RoundTripUnequal...)
	r.sum.Declines = codeccensus.MergeTallies(r.sum.Declines, s.Declines)
}

// declined is how many cases Append declined.
func (r report) declined() int {
	n := 0
	for _, t := range r.sum.Declines {
		n += len(t.Cases)
	}
	return n
}

// defects are the four defect counts, each with its label, in the order the
// report prints them and -paths lists their cases.
func (r report) defects() []defect {
	return []defect{
		{appendFailedLabel, r.sum.AppendFailed},
		{reValidateLabel, r.sum.ReValidateRejected},
		{reDecodeLabel, r.sum.ReDecodeFailed},
		{unequalLabel, r.sum.RoundTripUnequal},
	}
}

// defect is one defect count: its label and its cases.
type defect struct {
	label string
	cases []string
}

// decoded is the population's size: every row's count summed.
func (r report) decoded() int {
	n := r.sum.Encoded + r.declined()
	for _, d := range r.defects() {
		n += len(d.cases)
	}
	return n
}

// The labels of the defect counts, each printed on its count's line and over
// its -paths list.
const (
	appendFailedLabel = "append failed, no decline"
	reValidateLabel   = "re-validate rejected"
	reDecodeLabel     = "re-decode failed"
	unequalLabel      = "round trip unequal"
)

// caveats are the limits on reading any figure the census prints,
// codeccensus.Caveats among them; the report adds the placeholder's, which
// carries its count.
var caveats = slices.Concat([]string{
	"Defects: \"" + appendFailedLabel + "\", \"" + reValidateLabel + "\", \"" + reDecodeLabel + "\" and \"" + unequalLabel + "\" are defects in codec, each count exact, and any count other than 0 is one to file.",
	"Population: \"decoded\" is the cases decodecensus counts decoded into codegen's own type, never the placeholder, less those it counts under \"decoded where validate rejects\"; every other case is outside this census.",
	"Same value: the round trip compares decoded values, never bytes, each float by its bits with every NaN one, xs:double's identity, so 0 and -0 differ; an instance spelled other than canonically counts as encoded.",
	"One decline per case: Append returns its first decline, so a case counts once, under that construct.",
}, codeccensus.Caveats, []string{
	"No count is a yield: lifting a construct exposes each of its cases' next decline, so its count bounds from above the cases lifting it alone makes encode.",
})

// printReport renders the census: the counts, the declined constructs largest
// first, the caveats, and with withPaths the cases under each. A bufio.Writer
// keeps its first write error and Flush returns it, so checking Flush checks
// every write.
func printReport(dst io.Writer, rep report, withPaths bool) error {
	w := bufio.NewWriter(dst)
	_, _ = fmt.Fprintf(w, "encodecensus: %d instance case(s) under %s\n", rep.Considered, rep.Root)
	_, _ = fmt.Fprintf(w, "  schema generates: %d\n", rep.Generates)
	_, _ = fmt.Fprintf(w, "  decoded: %d\n", rep.decoded())
	_, _ = fmt.Fprintf(w, "  encoded: %d\n", rep.sum.Encoded)
	_, _ = fmt.Fprintf(w, "  declined: %d\n", rep.declined())
	for _, d := range rep.defects() {
		_, _ = fmt.Fprintf(w, "  %s: %d\n", d.label, len(d.cases))
	}
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
// order, then each defect count's, each list as the package doc's -paths
// states.
func printPaths(w *bufio.Writer, rep report, order []assess.Tally) {
	_, _ = fmt.Fprintln(w, "\n=== Cases, by construct ===")
	for _, t := range order {
		codeccensus.PrintCases(w, t.Construct, t.Cases)
	}
	_, _ = fmt.Fprintln(w, "\n=== Cases, by defect count ===")
	for _, d := range rep.defects() {
		codeccensus.PrintCases(w, d.label, d.cases)
	}
}

// placeholderCaveat is the caveat on the cases decoded into the placeholder.
func placeholderCaveat(rep report) string {
	return fmt.Sprintf("Placeholder: %d case(s) decode into a placeholder struct, not codegen's type, and are outside this census: "+
		"Append has no declaration of codegen's to encode them as.", rep.Placeholder)
}
