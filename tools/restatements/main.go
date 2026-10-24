// Command restatements lists the comment and doc sentences the working tree
// added since a git ref that restate a sentence of an owner: a package doc
// (any comment in a `doc.go`, or a package clause's doc), the doc comment of
// an exported identifier, or `cmd/goxsd8`'s usage text. It is a survey
// (PRINCIPLES 27) for STYLE D3 applied to prose, "One fact, one encoding".
// Any comment can be an added sentence, but only an owner's is compared
// against: a copy in an unexported identifier's doc or a test comment is
// listed only when it is the added one.
//
// Usage:
//
//	go tool restatements -base origin/main
//
// The ref's tree is materialized beside the checkout as tools/internal/reftree's
// package doc states, and both trees are read the same way: every
// `//` comment paragraph tools/internal/commentpara finds in a Go file outside
// the directories it skips, less generated files and preformatted paragraphs,
// plus usage's prose, cut into sentences. A sentence is added when its file
// at the ref holds no sentence of the same words, so a reflow adds none. A
// sentence of cmd/goxsd8/doc.go spelling a usage sentence word for word is
// usage rendered (cmd/goxsd8's TestDocRendersUsage) and is read only at
// usage, whose sentences are located at the main.go line each starts on.
//
// Every added sentence is compared with every owner sentence of the working
// tree but itself, and each pair that overlaps is printed: the added
// sentence's file:line and text, then, indented, the owner's. The overlap
// measure: an added sentence is cut into clauses at , ; : ( ) and a spaced
// dash, an owner sentence is taken whole, and each is reduced to its content
// words — lowercased, a possessive 's and a plural s on a word of more than
// four letters stripped, '\n' kept as a word, function words and the
// document-reference words "package", "doc", "section" and "see" dropped. A
// pair overlaps when one clause shares at least two distinct adjacent
// content-word pairs with the owner sentence, counting only pairs that at
// most 20 owner sentences of the working tree hold: a pair more hold is the
// tree's common phrasing, not one fact's.
//
// An added sentence that names an exported identifier of another package is
// also printed beside one sentence of that identifier's doc, whatever their
// overlap, marked "(names <selector>)" where an overlap pair prints "(N
// shared)", the selector being the first of the sentence naming it, unless a
// sentence of that doc already overlaps it: the doc's sentence sharing the
// most counted pairs with one clause, the first on a tie. A name is resolved
// lexically, from the sentence's raw text: a selector q.Ident or
// q.Type.Member, a `*`, `[` or possessive 's beside it read as any other
// punctuation, names the identifier Ident, or Type's method, field or
// interface method Member, declared and documented outside package main and
// outside the added sentence's own directory, in a package whose import path
// is q or ends in a slash and q. It does not resolve an import alias, an
// unqualified Ident, an embedded or promoted member, or (*q.Type).Member past
// q.Type; and it reads a standard-library selector whose qualifier is a
// module package's last element as that package's, so it pairs only where
// that package exports the same name. A main package exports nothing to
// resolve, so goxsd8 names the module's root package and not cmd/goxsd8.
//
// A summary line closes every run, saying how many added sentences overlap
// an owner or name an identifier, how many of those name one, and that the
// matching is heuristic. Exit codes: 0 whatever it prints, 2 on an
// operational error (a missing -base, an unresolvable ref, an unreadable or
// unparseable file, a working tree whose go.mod has no module directive).
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/kud360/goxsd8/tools/internal/reftree"
)

func main() {
	os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr))
}

// cli is main's testable body: it parses args, surveys the working tree of the
// module holding the working directory against -base, prints the result on
// stdout and returns the exit code.
func cli(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("restatements", flag.ContinueOnError)
	flags.SetOutput(stderr)
	base := flags.String("base", "", "git ref whose tree the added sentences are measured against (required)")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if err := run(*base, stdout, stderr); err != nil {
		_, _ = fmt.Fprintf(stderr, "restatements: %v\n", err)
		return 2
	}
	return 0
}

// run surveys the module's working tree against base and prints the report.
// A temporary worktree that will not go away is reported on stderr, as `go
// tool surface` reports it, and fails nothing.
func run(base string, stdout, stderr io.Writer) error {
	if base == "" {
		return fmt.Errorf("-base <ref> is required")
	}
	root, err := reftree.ModuleRoot()
	if err != nil {
		return err
	}
	baseDir, remove, err := reftree.Checkout(root, base)
	if err != nil {
		return err
	}
	defer func() {
		if err := remove(); err != nil {
			_, _ = fmt.Fprintf(stderr, "restatements: %v\n", err)
		}
	}()
	rep, err := survey(baseDir, root)
	if err != nil {
		return err
	}
	return rep.write(stdout)
}
