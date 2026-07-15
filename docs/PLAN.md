# goxsd8 Roadmap

Milestones map one-to-one to GitHub milestones. The cartographer carves
each into session-sized `ready` issues; the develop loop closes them one
per session. Prefer vertical slices that move a conformance lane over
horizontal completeness.

## M0 — Scaffold (done at bootstrap)

Repo layout, docs (STYLE/PRINCIPLES/ARCHITECTURE/WORKFLOW/ROUTINES/PLAN),
local specs + conversion tooling, W3C suite submodule, package contracts
(`doc.go` per package), agent personas and commands, lint gate.

## M1 — Spec infrastructure (done)

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

## M2 — Foundation leaves (done)

`xsderr` (Error/Rule/Loc + narrowing helpers), `loader` (Resolver +
Dir/FS/HTTP/Map/Chain), `parser/xmltree` (streaming position-tracking
decoder), and the `xsd.QName` expanded-name value type that
`value.Backend` and the builtin table key on (the datatypes-facing
`xsd.SimpleType` component follows in M3 alongside `Seed`; the rest of the
`xsd` component model waits for M4). Full unit tests; fuzz targets for
xmltree.

## M3 — Datatypes vertical slice (in progress — tail: 3 primitives + cohort widening)

`value` contracts finalized; `builtin/strict` primitive mappings + the
facet pipeline (pattern facets via package `regex`, XSD flavor) +
`builtin.Seed` — including the datatypes-facing `xsd.SimpleType` component
that `Seed` builds one of per builtin (the rest of the `xsd` component
model stays M4); `value/backendtest` kit running against strict. First
**`datatypes` ratchet lane** produces real numbers.

Status (2026-07-15): the shared facet pipeline hoisted into `value` (#87);
`builtin/strict` maps 17 of 20 primitives — decimal/float/double, the
string family, anyURI, hex/base64Binary, duration, and the seven-property
temporal family incl. dateTime (#103/#109). The `datatypes` lane claims
~716 cases (~700 pass). **Remaining tail:** map `precisionDecimal` (#115),
`QName`/`NOTATION` (#114), and the derived `dateTimeStamp` (#122); widen the
Facets cohort to the temporal (#123), binary/anyURI (#124), name-type (#116),
and QName/NOTATION (#125, blocked on #114) cases; the list/union-variety
executor + `value.effectiveWhiteSpace` not-applicable path (#98/#75) waits on
the `xsd` list/union variety shape (M4, #46). The NIST corpus and full
list/union cohort are tracked under the umbrella #75.

## M4 — Schema parsing (next — epic #79)

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

## v1.0 — the stability line

1.0 is declared by a human, not by a milestone rollover (expected after
M12). Until then, **pre-1.0 mobility** applies: interfaces, package
boundaries, and exported names move freely whenever the steward's
audit finds a better placement — the ratchet and the gate are the only
compatibility promises. After 1.0, exported-surface changes require a
deprecation path and a compatibility argument; the audit's posture
flips from "move it now" to "guard the surface". (Narrower freezes may
land earlier where a milestone says so — e.g. `value.Emitter` at M9.)

## Non-goals

- Schema mutation/editing APIs.
- XSD 1.0 compatibility quirks (this is an XSD 1.1 processor).
