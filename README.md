# goxsd8

An **XSD 1.1 processor for Go**: schema parser, instance validator
(XML, JSON, and BER sources), XPath 2.0 engine, `precisionDecimal`
support, and a code generator that emits allocation-frugal, type-safe
marshalling code.

> **Status: early.** The architecture, package contracts, and conformance
> harness are committed; implementations land milestone by milestone
> (see [docs/PLAN.md](docs/PLAN.md)). Every package's `doc.go` states its
> committed contract — `go doc` is the source of truth for what each
> package will do.

## What it does (the contract)

- **Parse** one or more XSD 1.1 schemas — imports, includes, redefines,
  overrides, chameleon composition — into one immutable, deterministic
  component model with **query** (lookup by name) and **walk**
  (content-model traversal) APIs.
- **Validate** instance documents against the compiled set. Sources are
  adapters over one abstract infoset: XML first, then JSON, then BER.
  Every violation carries the spec rule ID (`cvc-…`) and an exact
  file:line:column location.
- **XPath 2.0** for assertions, conditional type assignment, and
  identity constraints — the required subset first, growing to the full
  grammar and F&O function library, conformance-tracked.
- **Generate Go code**: `xs:choice` becomes a sealed interface (one
  concrete type per branch — no "five pointers, one non-nil" structs),
  values typed by your chosen backend, decode paths specialized for
  minimal allocation. Multiple schemas map to multiple output
  directories.
- **Bring your own value backend.** One ships today — `builtin/strict`
  (spec-exact: arbitrary precision, `precisionDecimal`, seven-property
  temporal model); a second, `builtin/native` (Go-friendly: `int64`,
  `time.Time`, documented deviations), is a fixed planned contract
  (M12 — see its `doc.go`). Backends are composable per type, and
  `value/backendtest` certifies third-party backends.

## Quickstart

```sh
git clone https://github.com/kud360/goxsd8
cd goxsd8
git submodule update --init testdata/xsdtests   # W3C suite, ~215 MB
go build ./... && go test ./...
go install ./cmd/goxsd8                         # goxsd8 onto $(go env GOPATH)/bin
```

`go build ./...` compiles every package and writes no executable; the
`go install` line is what produces the `goxsd8` binary the CLI section below
invokes.

### CLI (contract; `parse` and `validate` are implemented, `gen` is not)

This block is a **contract**, exactly like the Library one below: the help
path, `parse` and `validate` run today. A bare `goxsd8`, or
`-h`/`-help`/`--help` in any argument position, prints the usage contract to
stdout and exits 0
([issue #251](https://github.com/kud360/goxsd8/issues/251)). Every invocation
that reaches no built subcommand exits 2 with one of four lines on stderr:
`gen` is reserved but not yet implemented, any other name is an unknown
subcommand, a flag stands before the subcommand it qualifies, and a leading
flag with no subcommand after it is no subcommand at all. `gen` lands with M9.

**Exit 2 narrowed when `parse` landed, and again when `validate` did.** It
used to mean "this binary is a stub" for every invocation; for both built
subcommands it now means a usage or IO fault — a missing argument, an
undefined flag, a document that cannot be read, and for `validate` a
`-format` token outside `xml|json|ber` or an instance whose format is
`json` or `ber`, which the contract reserves and no milestone has built. It
is never a verdict about a schema or an instance. A script that read exit 2
as "skip, unimplemented" must read the stderr line, or branch on the other
codes instead: for `parse`, 0 compiled and 1 rejected; for `validate`, 0 no
violation charged, 1 an invalid instance, and 3 a schema set that does not
compile.

```sh
goxsd8 parse order.xsd items.xsd                # compile + summary, exit 0/1/2
goxsd8 validate -schema order.xsd -schema items.xsd order1.xml order2.xml
                                                # exit 0 clean, 1 invalid, 2 usage, 3 schema
goxsd8 gen -schema order.xsd -out ./gen/order \
           -schema items.xsd -out ./gen/items  # one package per -schema/-out pair
```

`validate` needs its own `-schema` for every schema, and reads every
positional argument as an instance. The `-schema` documents compose into
**one schema set** — several of them are one compilation, not one each,
through a synthesized wrapper document that `<xs:import>`s each schema with a
target namespace of its own and `<xs:include>`s each with none — so a name
two of them collide on is `sch-props-correct` clause 2 and a set that does
not compile exits **3**, distinctly from an invalid instance (1). **Every
instance is assessed**, in argument order: the run never stops at the first
invalid one, and its exit code is the worst outcome over them — 1 if any one
of them is invalid. An instance argument spelled `-` is standard input;
`-schema -` is not supported, a schema document's location being the base URI
its own relative references resolve against.

`validate` derives each instance's source format from its extension, or from
`-format` when that is given; an instance whose extension names none of
`.xml`, `.json` and `.ber` — `-` among them — is a usage error naming the
values rather than a guess. Only `xml` is assessed today: `json` and `ber`
are recognized tokens whose adapters (`validate/jsonsrc`, `validate/bersrc`)
are M8 and M11 stubs, so an instance in either exits 2 saying so. An
`xsi:schemaLocation` or `xsi:noNamespaceSchemaLocation` hint on an XML
instance's **document element** augments that instance's schema set, resolved
against the instance's own path; `-no-hints` turns that off, so a `-schema`
set that declares nothing for the validation root is then charged
`cvc-assess-elt` instead of quietly succeeding on a schema the instance
itself named.

`parse` compiles **each argument separately**, in argument order — several
schema arguments are several compilations, not one set — and prints each
summary on stdout: the distinct namespaces of the components that compilation
declares (the argument document and every one it includes, imports, overrides
or redefines), in first-appearance order and none when it declares nothing,
then a count of each kind of declaration those documents make. A rejected
schema prints its first error on stderr as `<loc>: [<rule>] <message>` and
assembly stops there, so that is one line per rejected argument; the exit code
is the worst outcome over the arguments: 0 when every one compiles, 1 when any
is rejected, 2 when any cannot be read.

Beyond `-schema` and `-out`, the contract carries `-format xml|json|ber`
(force the instance source format instead of deriving it from the
extension; matched case-sensitively and applying to every instance of the
invocation, there being no per-instance spelling), `-no-hints` (ignore
`xsi:schemaLocation` hints in XML instances), `-backend strict|native`
(which value backend `gen` emits against), `-q` (quiet) and `-v` (debug
logging to stderr via `slog`, scoped with
`GOXSD_DEBUG=parser,validate,codec` — a scoping neither `parse` nor
`validate` honours yet). The common flags qualify a subcommand and **follow its name**:
`goxsd8 parse -q order.xsd`, not `goxsd8 -q parse order.xsd`. `-q`
suppresses a subcommand's informational output — `parse`'s summary — and
never a diagnosis: neither `parse`'s error lines nor `validate`'s
violations.
**`go doc github.com/kud360/goxsd8/cmd/goxsd8` is the authoritative CLI
contract**, and this section summarizes it. `goxsd8 -help` prints the usage
block from that contract — the subcommand syntax, the common flags and the
implementation status — and not its argument vocabulary (which spellings of
the help flag are recognized, whether `help` and `-version` are names, what
`--` means, how a schema path is resolved) or its library relationship.

`validate`'s violations print one per line on stdout as
`<loc>: [<rule>] <message>` — the same rendering `parse` gives a schema
error on stderr — where `<loc>` is `<file>:<line>:<col>` (`?` when unknown)
and `<rule>` is the spec validation rule ID:

```
order.xml:3:3: [cvc-type] the ·initial value· of the element amount is not ·valid· with respect to its ·governing type definition· {http://www.w3.org/2001/XMLSchema}decimal, which cvc-type clause 3.1.3 requires as per String Valid (§3.16.4): ?: [cvc-datatype-valid] decimal: "12,50" is not in the lexical space (decimal-lexical-representation, §3.3.3.1)
```

`<rule>` is the rule CHARGED, and for the content of an element or attribute
that is never `cvc-datatype-valid`: the charge is `cvc-type` (clause 3.1.3),
`cvc-attribute` (clause 3) or `cvc-complex-type` (clause 1.2), each of which
delegates through String Valid (§3.16.4) to Datatype Valid and carries that
verdict as a WRAPPED cause — rendered into the message, as above, and
reachable as an `*xsderr.Error` of its own through `errors.As` and
`xsderr.RuleOf`. Key a dispatcher on the outer rule.

### Library

Seeding the builtin datatypes and compiling a schema set both work TODAY:

```go
package main

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/kud360/goxsd8/builtin"
	"github.com/kud360/goxsd8/builtin/strict"
	"github.com/kud360/goxsd8/loader"
	"github.com/kud360/goxsd8/parser"
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

	// parser.Parse assembles the <xs:include> / <xs:import> / <xs:override>
	// closure of the root document — including chameleon coercion of a
	// no-targetNamespace included (or overridden) document into the including
	// namespace — and returns it finalized. Every option has a default, so
	// parser.Parse("order.xsd") alone is valid too.
	schema, err := parser.Parse("order.xsd",
		parser.WithBackend(backend),                // default: builtin/strict
		parser.WithResolver(loader.Dir("schemas")), // default: loader.Dir(".")
		parser.WithLogger(slog.Default()),          // default: silent
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("builtin types:", len(builtins), "top-level elements:", len(schema.Elements()))
}
```

To back one type with your own mapping and inherit the rest, compose:
`value.Override(strict.New(), money)` yields `money`'s mapping for every
type `money` defines and `strict`'s for all the others.

One limit of `parser.Parse` worth knowing up front: it returns only the
FIRST error, not a list of them.

`parser.ParseReport(location string, opts ...Option) (*xsd.Schema,
*AssemblyReport, error)` is the primitive, and `parser.Parse` is it without
the report. Reach for it to diagnose an assembly that came back smaller than
expected: `AssemblyReport.Documents()` lists the readings the assembly
performed, and `Unfollowed()` the `xs:include` / `xs:redefine` /
`xs:override` / `xs:import` references that yielded no document, each with
the directive element's position and the reason — a `schemaLocation` that
resolved to nothing, or a bare `xs:import` naming no document at all, which
§4.2.6.2 makes legal. The report is never nil and is populated as far as
assembly got even when an error comes back; `go doc
github.com/kud360/goxsd8/parser AssemblyReport` is its contract. It does not
lift the first-error limit above — `ParseReport` stops at the first error
too, so directives it never reached are neither followed nor reported.

The component model is also constructible directly, without a schema
document: `xsd.NewSchemaBuilder()` → `Add*` → `Finalize()` returns an
immutable `*xsd.Schema` you query by `xsd.QName` (`Type`, `Element`,
`Attribute`). That builder is PRODUCER surface, not application surface:
every `Add*` takes an already-validated component value, and building one
correctly means honoring every §3 tableau and cross-property invariant its
constructors cannot check — which is `parser.Produce`'s job. An application
that has a schema DOCUMENT calls `parser.Parse` and receives the finalized
`*xsd.Schema`. See `go doc github.com/kud360/goxsd8/xsd SchemaBuilder`, and
`Example_buildFinalizeQuery` in `xsd/example_test.go` for the construct →
`Finalize` → query sequence.

Instance validation runs today: `validate.New` builds the engine and
`validate/xmlsrc` drives an XML document through it. `go doc
github.com/kud360/goxsd8/validate`'s "Contract (M5, landing rule by rule)"
section is the authoritative account of which `cvc-` rules are charged and
what is left undecided — this block only shows the calls. It continues the
program above, with `schema` and `backend` in scope, and adds `os`,
`github.com/kud360/goxsd8/validate` and
`github.com/kud360/goxsd8/validate/xmlsrc` to its imports:

```go
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

// err means the assessment never RAN: a nil argument, or a document with no
// well-formed document element to start it. A source fault that stopped the
// walk mid-document lives in res.Err() alone and is never also returned here.
res, err := xmlsrc.Validate(v, f, xmlsrc.WithURI("order.xml"))
if err != nil {
	log.Fatal(err)
}
// The verdict is res.Violations(); a non-nil res.Err() means the assessment
// is INCOMPLETE, so an empty Violations() then proves nothing.
if err := res.Err(); err != nil {
	log.Fatal(err)
}
for _, e := range res.Violations() { // []*xsderr.Error, in document order
	fmt.Println(e)
}
```

Start at `go doc github.com/kud360/goxsd8` and follow the package list;
each package's godoc is its contract. Plain-text `go doc` does not print
the runnable `Example*` funcs, so for working, tested end-to-end code
(seed builtins → parse a lexical → assert capabilities) read the example
tests directly: `value/example_test.go` (`ExampleOverride`),
`builtin/example_test.go` (`ExampleSeed`, `ExampleSeed_missingPrimitive`),
`builtin/strict/example_test.go` (`ExampleNew`),
`loader/example_test.go` (`Example_chain`), `xsd/example_test.go`
(`Example_termOrRefDiscrimination`, `Example_buildFinalizeQuery`,
`Example_schemaEnumeration`, `Example_contentModelTraversal`,
`Example_modelGroupScopeParent`), and `xsd/namespaceconstraint_test.go`
(`ExampleNamespaceConstraint_AllowsNamespace`).

## Documentation map

| Doc | What it holds |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | package DAG, facet pipeline, backends, codegen design |
| [docs/STYLE.md](docs/STYLE.md) | non-negotiable code rules (cited by ID in reviews) |
| [docs/PRINCIPLES.md](docs/PRINCIPLES.md) | the invariants and spec traps behind the rules |
| [docs/PLAN.md](docs/PLAN.md) | roadmap M0–M12 |
| [docs/WORKFLOW.md](docs/WORKFLOW.md) | the rules every development session obeys |
| [docs/ROUTINES.md](docs/ROUTINES.md) | running the loop on Claude routines |
| [docs/specs/](docs/specs/README.md) | the local W3C specs (ground truth, greppable) |

## Conformance

The W3C XSD test suite (pinned submodule) drives a **ratchet**: expected
outcomes are committed per lane under
`conformance/testdata/expectations/`, regressions fail CI loudly, and
expectations only ever move up. See
[conformance's godoc](conformance/doc.go).

```sh
go test ./conformance -run TestConformance -count=1
```

## How this repo is developed

goxsd8 is built primarily by AI agents — scheduled Claude Code routines
running the slash commands in `.claude/commands/` (`/develop`, `/backlog`,
`/ratchet`, `/retro`, `/story`), with specialized personas in
`.claude/agents/` (implementer, judge, spec oracle, API warden, planner,
architecture steward, historian, and two simulated users who test the
docs you are reading right now). GitHub issues are the project's memory; every change is
judged against [docs/STYLE.md](docs/STYLE.md) and the conformance
ratchet before it lands. Humans are welcome — file issues, or run the
same commands locally.

## License

Apache 2.0 — see [LICENSE](LICENSE).
