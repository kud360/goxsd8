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

## M3 — Datatypes vertical slice (in progress — all 20 primitives mapped; tail: derived durations, remaining Facets/precisionDecimal cohorts, NOTATION shape)

`value` contracts finalized; `builtin/strict` primitive mappings + the
facet pipeline (pattern facets via package `regex`, XSD flavor) +
`builtin.Seed` — including the datatypes-facing `xsd.SimpleType` component
that `Seed` builds one of per builtin (the rest of the `xsd` component
model stays M4); `value/backendtest` kit running against strict. First
**`datatypes` ratchet lane** produces real numbers.

Status (2026-07-18, weekly backlog): the shared facet pipeline is hoisted into
`value` (#87); `builtin/strict` now maps **all 20 builtin primitives** —
decimal/float/double, the string family, anyURI, hex/base64Binary, duration, the
seven-property temporal family incl. dateTime (#103/#109), QName/NOTATION (#114),
and precisionDecimal (#115). The `datatypes` lane now stands at **1006 pass / 34
fail** (1040 cases), up from 939/25/964 the prior week; the +76 cases are the
Saxon `PDecimal` precisionDecimal instance cohort discovered and claimed via the
new `extra-suite.xml` discovery path (#135) plus the QName lexical/Facets cohorts.
It was widened through the ID/IDREF/ENTITY name-type (#116), temporal (#123),
anyURI/hex/base64Binary (#124), and QName (#125) Facets cohorts; the derived
`dateTimeStamp` is mapped (#122), the `lengthFacet` §4.3.1.3 clause-1.3
QName/NOTATION exemption is fixed (#130), the QName/NOTATION namespace-context
adapter for the lexical cohort landed (#131), the redundant `fallbackPrimitives`
shim was removed (#134), and the `<item>` lexical sub-shape is routed (#146).
precisionDecimal `maxScale`/`minScale` instance-time enforcement landed (#133 —
`cvc-maxScale-valid`/`cvc-minScale-valid`, `GAP(facet)` retired), which unblocked
and closed the precisionDecimal instance selectors (#135).

**Remaining tail (all `ready`):** map the derived `yearMonthDuration`/
`dayTimeDuration` (#141); fix the dateTimeStamp lexical-cohort Parse-only
false-accept (#140, latent — no tz-absent case in the pinned checkout yet); widen
the lane to the wider-primitive / native-space Facets cohort (#145); thread the
declaring-schema namespace context into enumeration-facet `{value}` parsing for
QName/NOTATION (#152, new library surface — warden pre-flight); model the NOTATION
Facets-cohort two-step-restriction shape (#153, expect to split once scoped);
claim the IBM `D3_3_4` multi-type-per-schema precisionDecimal cohort (#162);
harden `value/facets.go` `compile()` with a fail-loud default on unhandled
FacetKind (#158); and establish the "LOG-entry-not-expectations-comment is the
dismissal record" process rule (#149). **Blocked tail:** the four out-of-scope
precisionDecimal schema-construction SCCs (valid-restriction narrowing,
minScale≤maxScale, {fixed} inheritance) are #157 (blocked on the M4 producer #79).
The list/union-variety executor + `value.effectiveWhiteSpace` not-applicable path
(#98 / rescoped #75) — including the pdecimal016/019/020 two-step/list/union
shapes — still waits on the `xsd` list/union variety shape (M4, #46). The NIST
corpus is a follow-up once #145 lands.

## M4 — Schema parsing (next — epic #79, human-gated)

Three-phase parser over the composition model (include/import/redefine/
override, chameleon coercion), UPA/EDC/particle-restriction designed into
the model shape from the start. **`schema` lane.**

Epic #79 is **human-gated**: do not carve it into `ready` sub-slices or
start the develop loop on it ahead of finishing the M3 datatypes vertical
slice unless a human reprioritizes. Its five leaf follow-ups (#72, #70,
#63, #51, #46) plus the sibling #52 today list their long-closed seed deps;
repointing their `## Depends on` to the #79 epic is **deferred to the M4
carve** (an epic does not close per-slice, so repointing now would not drive
the post-land unblock pass).

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
