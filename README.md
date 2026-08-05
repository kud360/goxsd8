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

### CLI (contract; no subcommand is implemented yet)

This block is a **contract**, exactly like the Library one below: today the
`goxsd8` binary is a stub — every invocation prints a pointer to the contract
and exits 2 ([issue #251](https://github.com/kud360/goxsd8/issues/251)).
Subcommands land with their milestones (`parse` M4, `validate` M5, `gen` M9).

```sh
goxsd8 parse order.xsd items.xsd                # compile + summary, exit 0/1
goxsd8 validate -schema order.xsd order1.xml order2.json
                                                # exit 0 valid, 1 invalid, 2 usage
goxsd8 gen -schema order.xsd -out ./gen/order \
           -schema items.xsd -out ./gen/items  # one package per -schema/-out pair
```

Beyond `-schema` and `-out`, the contract carries `-format` (force the
instance source format instead of deriving it from the extension),
`-no-hints` (ignore `xsi:schemaLocation` hints in XML instances),
`-backend strict|native` (which value backend `gen` emits against), `-q`
(quiet) and `-v` (debug logging to stderr via `slog`, scoped with
`GOXSD_DEBUG=parser,validate,codec`).
**`go doc github.com/kud360/goxsd8/cmd/goxsd8` is the authoritative flag
list**; this section summarizes it.

Violations print one per line as `<loc>: [<rule>] <message>`, where `<loc>`
is `<file>:<line>:<col>` (`?` when unknown) and `<rule>` is the spec
validation rule ID:

```
order.xml:12:5: [cvc-datatype-valid] decimal: "12,50" is not in the lexical space (decimal-lexical-representation, §3.3.3.1)
```

### Library (seeding and parsing work today; validation is contract)

Seeding the builtin datatypes and compiling a schema set both work TODAY:

```go
// A value backend supplies the builtin datatypes' value spaces. builtin.Seed
// needs one that maps all 20 builtin primitives; builtin/strict maps all 20 on
// its own, so nothing has to be composed in.
backend := strict.New()

// Seed the builtin datatype components. Call this when you want the
// components yourself — parser.Parse seeds its own from its backend.
builtins, err := builtin.Seed(backend)  // []*xsd.SimpleType, deterministic order

// parser.Parse assembles the <xs:include> / <xs:import> / <xs:override>
// closure of the root document — including chameleon coercion of a
// no-targetNamespace included (or overridden) document into the including
// namespace — and returns it finalized. Every option has a default, so
// parser.Parse("order.xsd") alone is valid too.
schema, err := parser.Parse("order.xsd",
	parser.WithBackend(backend),                 // default: builtin/strict
	parser.WithResolver(loader.Dir("schemas")),  // default: loader.Dir(".")
	parser.WithLogger(logger),                   // default: silent
)
```

To back one type with your own mapping and inherit the rest, compose:
`value.Override(strict.New(), money)` yields `money`'s mapping for every
type `money` defines and `strict`'s for all the others.

Two limits of `parser.Parse` worth knowing up front: a `<xs:redefine>` that
redefines a `complexType` is declined (its `simpleType`, `group` and
`attributeGroup` forms are followed in full); and
`Parse` returns only the FIRST error, not a list of them.

The component model is also constructible directly, without a schema
document: `xsd.NewSchemaBuilder()` → `Add*` → `Finalize()` returns an
immutable `*xsd.Schema` you query by `xsd.QName` (`Type`, `Element`,
`Attribute`). See `go doc github.com/kud360/goxsd8/xsd SchemaBuilder`; a
worked example is tracked in
[issue #203](https://github.com/kud360/goxsd8/issues/203).

The instance-validation step below is still the PLANNED contract —
`validate.New` / `xmlsrc.Validate` (M5) do not exist yet. Shown here for the
shape the API will take, not code you can build today.

```go
v, err := validate.New(schema)
res := xmlsrc.Validate(v, r)  // res.Errors: []*xsderr.Error
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
