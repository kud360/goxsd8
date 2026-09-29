// Package declinecensus reads one lane's GOXSD_DECLINES=1 decline census off
// a conformance run's -v log: the three sorted case-ID lists
// conformance/doc.go's "The decline census" section defines — decline
// candidates, indeterminate declines, and decided disagreements — which
// partition that run's recorded failures.
//
// It is the one reader of that listing for every tool that takes a run log
// (`go tool lanepartition -log`, `go tool casejoin -log`), so the tools cannot
// disagree about what a log says (STYLE T4). The listing's labels are
// constructed a second time here, reportDeclines in
// conformance/conformance_test.go being the first: a rename there is a rename
// here, or every log reads as carrying no listing.
package declinecensus
