# goxsd8 Roadmap

Milestones map one-to-one to GitHub milestones. The cartographer carves
each into session-sized `ready` issues; the develop loop closes them one
per session. Prefer vertical slices that move a conformance lane over
horizontal completeness.

## M0 — Scaffold (done at bootstrap)

Repo layout, docs (STYLE/PRINCIPLES/ARCHITECTURE/WORKFLOW/ROUTINES/PLAN),
local specs + conversion tooling, W3C suite submodule, package contracts
(`doc.go` per package), agent personas and commands, lint gate.

## M1 — Spec infrastructure

- **hfn → TypeSpec generator**: extend `tools/hfnextract` with a generator
  that emits `builtin/gen_typespec.go` — the backend-neutral data table for
  all 49 builtins **including precisionDecimal** (name, base, variety,
  fundamental facets, applicable facets + defaults), sourced from the
  Appendix E function definitions and per-type property tables in
  `docs/specs/md/xmlschema11-2.md` and `xsd-precisionDecimal.md`. Wired to
  `go generate`; acceptance = byte-identical regeneration, zero hand-typed
  rows.
- **Conformance ratchet**: implement `conformance` per its doc.go —
  expectations load/compare/merge (upward-only, refuse regressions and
  vanished cases), `suite.xml` runner skeleton, lane files.
- **Rule catalog**: `xsderr` gains its `Rule`/catalog wiring so
  `tools/rulecat` output compiles and `go generate ./...` is green.

## M2 — Foundation leaves

`xsderr` (Error/Rule/Loc + narrowing helpers), `loader` (Resolver +
Dir/FS/HTTP/Map/Chain), `parser/xmltree` (streaming position-tracking
decoder), and the `xsd.QName` expanded-name value type that
`value.Backend` and the builtin table key on (the rest of the `xsd`
component model waits for M4). Full unit tests; fuzz targets for xmltree.

## M3 — Datatypes vertical slice

`value` contracts finalized; `builtin/strict` primitive mappings + the
facet pipeline (pattern facets via package `regex`, XSD flavor) +
`builtin.Seed`; `value/backendtest` kit running against strict. First
**`datatypes` ratchet lane** produces real numbers.

## M4 — Schema parsing

Three-phase parser over the composition model (include/import/redefine/
override, chameleon coercion), UPA/EDC/particle-restriction designed into
the model shape from the start. **`schema` lane.**

## M5 — Instance validation (XML)

`validate` engine + `validate/xmlsrc`; greedy deterministic matching, IDC,
xsi:type/nil, wildcards, default/fixed values. **`instance` lane.**

## M6 — XPath required subset

CTA restricted subset + assertion essentials; fail-open with GAP markers;
IDC selector/field paths. Dynamic-error direction per PRINCIPLES 20.

## M7 — XPath 2.0 growth

Grammar completion toward full XPath 2.0 + the F&O function library
(`docs/specs/md/xpath20.md`, `xpath-functions.md`). **`xpath` lane.**

## M8 — JSON instance adapter

`validate/jsonsrc` mapping JSON onto the abstract infoset. **`json` lane**
(curated cases; the W3C suite has no JSON lane).

## M9 — Codegen

Deterministic emission, namer, sealed choice sums, capability-view
interfaces, multiple schemas → multiple output dirs, golden-file tests.
The public `value.Emitter` API freezes here.

## M10 — Codec

Runtime path + generated fast path; differential tests (identical values,
identical error rule IDs) and `testing.AllocsPerRun` budgets.

## M11 — BER instance adapter

`validate/bersrc`. **`ber` lane** (curated cases).

## M12 — Native backend completion

`builtin/native` mappings + emitter, backendtest green, performance pass.

## Non-goals

- Schema mutation/editing APIs.
- XSD 1.0 compatibility quirks (this is an XSD 1.1 processor).
