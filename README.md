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
```

### CLI (contract; subcommands land with their milestones)

```sh
goxsd8 parse order.xsd items.xsd                # compile + summary, exit 0/1
goxsd8 validate -schema order.xsd order1.xml order2.json
                                                # exit 0 valid, 1 invalid, 2 usage
goxsd8 gen -schema order.xsd -out ./gen/order \
           -schema items.xsd -out ./gen/items  # one package per -schema/-out pair
```

Violations print one per line: `<loc>: [<rule>] <message>`.

### Library (contract; APIs land with their milestones)

```go
// Seed the builtin datatypes from a value backend (compose to cover every
// primitive). This step works TODAY — the components feed the parser's
// symbol table. See builtin/example_test.go and builtin/strict/example_test.go.
backend := value.Override(fallback, strict.New())  // cover all primitives
builtins, err := builtin.Seed(backend)             // []*xsd.SimpleType

// The parse → validate steps below are the PLANNED contract — parser.Parse
// (M4), validate.New / xmlsrc.Validate (M5) do not exist yet. Shown here for
// the shape the API will take, not code you can build today.
set, err := parser.Parse("order.xsd")           // or ParseMultiple
v, err := validate.New(set)
res := xmlsrc.Validate(v, r)                     // res.Errors: []*xsderr.Error
```

Start at `go doc github.com/kud360/goxsd8` and follow the package list;
each package's godoc is its contract. Plain-text `go doc` does not print
the runnable `Example*` funcs, so for working, tested end-to-end code
(seed builtins → parse a lexical → assert capabilities) read the example
tests directly: `value/example_test.go` (`ExampleOverride`),
`builtin/example_test.go` (`ExampleSeed`, `ExampleSeed_missingPrimitive`),
`builtin/strict/example_test.go` (`ExampleNew`), and
`loader/example_test.go` (`Example_chain`).

## Documentation map

| Doc | What it holds |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | package DAG, facet pipeline, backends, codegen design |
| [docs/STYLE.md](docs/STYLE.md) | non-negotiable code rules (cited by ID in reviews) |
| [docs/PRINCIPLES.md](docs/PRINCIPLES.md) | the invariants and spec traps behind the rules |
| [docs/PLAN.md](docs/PLAN.md) | roadmap M0–M12 |
| [docs/WORKFLOW.md](docs/WORKFLOW.md) | the development loop |
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
