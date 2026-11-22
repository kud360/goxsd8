package main

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/codegen"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
	"github.com/kud360/goxsd8/validate"
	"github.com/kud360/goxsd8/validate/jsonsrc"
	"github.com/kud360/goxsd8/validate/xmlsrc"
)

func main() {
	// A value backend supplies the builtin datatypes' value spaces. builtin.Seed
	// needs one that maps all 20 builtin primitives; builtin/strict maps all 20 on
	// its own, so nothing has to be composed in.
	backend := strict.New()

	// Seed the builtin datatype components. Call this when you want the
	// components yourself — parser.Parse seeds its own from its backend.
	builtins, err := builtin.Seed(backend) // []*xsd.SimpleType, deterministic order
	if err != nil {
		log.Fatal(err)
	}

	// parser.Parse assembles the <xs:include> / <xs:override> /
	// <xs:redefine> / <xs:import> closure of the root document — including
	// chameleon coercion of a no-targetNamespace included (or overridden, or
	// redefined) document into the including namespace — and returns it
	// finalized. Every option has a default, so parser.Parse("order.xsd")
	// alone is valid too.
	schema, err := parser.Parse("order.xsd",
		parser.WithBackend(backend),          // default: builtin/strict
		parser.WithResolver(loader.Dir(".")), // default: loader.Dir("."), the working directory
		parser.WithLogger(slog.Default()),    // default: silent
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("builtin types:", len(builtins), "top-level elements:", len(schema.Elements()))

	// backend MUST be the one the schema was compiled with: a different value
	// space rejects documents the schema admits, and nothing here can detect it.
	v, err := validate.New(schema, backend)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.Open("order.xml")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// err means no assessment stands: a nil argument, or a document that is not
	// well-formed where the walk did not read it — before the document element,
	// after it, or in a subtree the walk did not enter. A fault that stopped the
	// walk lives in res.Err() alone and is never also returned here.
	res, err := xmlsrc.Validate(v, f, xmlsrc.WithURI("order.xml"))
	if err != nil {
		log.Fatal(err)
	}
	// Standing() is the acceptance test, not an empty Violations(): it is Charged
	// when a violation was charged, Undecided when none was but a check was left
	// unperformed or res.Err() stopped the walk, and NotProvenInvalid otherwise,
	// which means not proven invalid rather than valid.
	fmt.Println(res.Standing())
	for _, e := range res.Violations() { // []*xsderr.Error, in the order Result.Violations states
		fmt.Println(e)
	}
	if err := res.Err(); err != nil { // the assessment is INCOMPLETE, not invalid
		fmt.Println(err)
	}
	for _, u := range res.Unevaluated() { // no String method: this is the CLI's form
		fmt.Printf("%s: [%s] undecided: %s\n", u.Loc(), u.Rule(), u.Msg())
	}

	jf, err := os.Open("order.json") // the CLI section's order.json
	if err != nil {
		log.Fatal(err)
	}
	defer jf.Close()

	// err is every fault, with no result: a nil argument, malformed JSON or
	// content after it, a read fault, or a decline of an input the mapping does
	// not map. res.Err() is nil on every result jsonsrc returns, so Standing() is
	// Undecided by Unevaluated() alone. Pass WithURI: without it, a decline or a
	// violation is located at ?, as in ?:1:2.
	res, err = jsonsrc.Validate(v, jf, jsonsrc.WithURI("order.json"))
	// A decline is no verdict on the document: report it as undecided, in the
	// line the CLI prints for it before exiting 4.
	var je *jsonsrc.Error
	if errors.As(err, &je) && je.Class() == jsonsrc.Decline {
		fmt.Printf("%s: [jsonsrc] undecided: %s\n", je.Loc(), je.Msg())
		return
	}
	// Every other error stays fatal. The CLI exits 1 on malformed JSON and on a
	// read fault.
	if err != nil {
		log.Fatal(err)
	}
	// Read res.Standing(), res.Violations() and res.Unevaluated() as above.

	// Writes package order into gen/order.
	if errs := codegen.Generate([]codegen.Target{{Schema: schema, Dir: "gen/order", Package: "order"}}); errs != nil {
		// Generate returns one *codegen.TargetError per failure: print each, a
		// decline by its Loc and Construct, any other failure as it reads.
		for _, e := range errs {
			var de *codegen.DeclineError
			if errors.As(e, &de) {
				fmt.Printf("%s: codegen declined %s\n", de.Loc(), de.Construct())
				continue
			}
			fmt.Println(e)
		}
		os.Exit(1)
	}
}
