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

## M3 — Datatypes vertical slice (complete — all 20 primitives mapped, `datatypes` lane 1036 pass / 38 fail (1074); the IBM precisionDecimal cohort (#162) and the `Mapping.Canonical` doc (#166) landed 2026-07-19; only the independent anyURI-triage #190 remains as optional follow-up)

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

Update (2026-07-19, weekly backlog): the M3 datatypes tail has **drained**.
Since 2026-07-18 the following landed — derived `yearMonthDuration`/
`dayTimeDuration` (#141), dateTimeStamp lexical Parse-only false-accept fix (#140),
enumeration-facet namespace context for QName/NOTATION (#152), the NOTATION
Facets two-step shape (#153), `compile()` fail-loud default (#158), and the
"LOG-is-the-dismissal-record" process rule (#149); #145 (wider-primitive Facets
cohort) was **closed as already-satisfied** (no boolean fixtures in the checkout).
The `datatypes` lane now stands at **1025 pass / 30 fail** (1055 cases).
**Remaining datatypes cleanup (all `ready`):** claim the IBM `D3_3_4`
multi-type-per-schema precisionDecimal cohort (#162); document
`value.Mapping.Canonical`'s per-value partial-domain error (#166, doc-only,
harvested from #141); and triage the 8 untracked MS-DataTypes
`anyURI_a*`/`anyURI_b*` lane fails — real gap vs spec-correct suite disagreement
(#190, filed this backlog). With the tail drained the develop loop has rolled
onto the M4 first wave. A cross-cutting README-to-published-surface doc sync
(#189, cliuser+libuser harvested) is also `ready`. **Blocked tail:** the four out-of-scope
precisionDecimal schema-construction SCCs (valid-restriction narrowing,
minScale≤maxScale, {fixed} inheritance) are #157 (blocked on the M4 producer #79).
The list/union-variety executor + `value.effectiveWhiteSpace` not-applicable path
(#98 / rescoped #75) — including the pdecimal016/019/020 two-step/list/union
shapes — still waits on the `xsd` list/union variety shape (M4, #46). The NIST
corpus is a follow-up (#145 was closed as already-satisfied, not landed — no
boolean fixtures in the checkout).

Update (2026-07-20, weekly backlog): the M3 datatypes tail is fully drained. #162
(IBM `D3_3_4` multi-type-per-schema precisionDecimal cohort, +11 pass / +8
honestly-declined-fail) and #166 (`Mapping.Canonical` doc) landed 2026-07-19, so
the `datatypes` lane now stands at **1036 pass / 38 fail** (1074 cases). The only
open M3-adjacent cleanup is the independent anyURI-triage #190 (`ready`); it is not
on any milestone critical path. The develop loop has moved fully onto M4.

## M4 — Schema parsing (epic #79 — gate lifted 2026-07-18, carved)

Three-phase parser over the composition model (include/import/redefine/
override, chameleon coercion), UPA/EDC/particle-restriction designed into
the model shape from the start. **`schema` lane.**

The human owner **lifted the human gate on 2026-07-18** and epic #79 was
carved into 17 session-sized sub-slices (#167–#183) in dependency order:
parse phase (#167); `xsd` model shapes (#168 element decl, #169 attribute
decl/group/use, #170 particle/model-group, #171 complex type); schema
container + phase seam (#172) and finalize/resolve — `src-resolve`,
dependency-ordered finalization, named-circularity rejection (#173);
producer spine (#174) and the **first `schema`-lane movement** via the
`conformance/schema.go` driver + first ratchet (#175); producer widening
(#176 complex-type/content-model, #177 attribute-group/model-group defs,
#178 IDC/assertion/notation/wildcard); composition loader (#179 include +
chameleon, #182 import, #183 redefine/override); and finalize model-validity
(#180 UPA/EDC, #181 complex-type derivation validity incl. particle
restriction §3.9.6). Ready first wave (no open deps): #167, #168, #169,
#170. Each new-exported-surface slice carries a warden pre-flight
(esp. the `parser` package shape #167 and `xsd` additions #168–#172).

The five leaf follow-ups (#72, #70, #63, #51, #46) plus siblings #52 and
#157 have had their `## Depends on` **repointed** from the unfiled-phase
placeholders / bare #79 to the concrete sub-slice numbers above (done in the
carve); they stay `blocked` and flip `ready` via the post-land unblock pass
as their named producer/finalize sub-slices land.

Update (2026-07-20, weekly backlog): the M4 **first wave (#167 parse phase,
#168 element decl, #169 attribute decl/group/use, #170 particle/model-group)
is all landed.** The next actionable M4 leaf is **#171** (Complex Type
Definition — its deps #168/#169/#170 are all closed, so it is `ready`); it is
the single item on the M4 critical path right now. The chain behind it is a
strict serial spine — #172 (schema container) unblocks only when #171 lands,
#173 (finalize/resolve) when #172 lands, then the producer fan-out
(#174→#175/#176/#177/#178/#179) and the finalize-validity/composition tail
(#180/#181/#182/#183) — each link flips `ready` via the post-land unblock pass
as its named producer lands. So the ready frontier is **dependency-capped**:
#171 (critical path) plus independent, off-critical-path cleanup that can run in
parallel — three `xsd`-leaf/doc items harvested this backlog from the #170
landing and a libuser godoc review (#201 the `ResolvedTerm{Term: nil}` guard,
#202 the absent-zero-QName gap in the M4 Required-name/ref constructors, #203 a
worked M4-shape Example + not-implemented markers on the `xsd` Query/Walk doc
sections), plus #190 (anyURI datatypes-lane triage), #189 (README surface sync),
and #195 (mason docs/LOG guard, process/tooling). The **`schema` lane is still
at 0 pass / 15432 fail** (`stubFail`); its first real movement lands with #175.
The shallow-looking `ready` count is the serial M4 spine, not a planning gap —
the deep cascade is behind #171 and self-feeds through the post-land passes.

Update (2026-07-21, weekly backlog): **#171, #172, and #173 all landed since the
prior backlog**, so the spine has moved a full link further: Complex Type
Definition + {content type}/derivation shapes (#171), the `Schema`/`SchemaBuilder`
container + symbol tables + Query views (#172), and finalize/resolve —
`src-resolve` QName resolution, dependency-ordered finalization, named-circularity
rejection (#173) — are done. #201 (the `ResolvedTerm{Term: nil}` guard) closed
as done-there, absorbed by #173's landing exactly as its own Notes anticipated.
The develop loop's own post-land pass already unblocked **#174** (producer
spine — top-level simpleType/element/attribute → `xsd` components) the same
session #173 landed; it is now the single item on the M4 critical path. #175
(schema-lane bring-up — the first real `schema`-lane movement) stays `blocked`
on #174, and the rest of the fan-out/finalize-validity tail stays blocked
behind it — this is still the same dependency-capped spine, not a planning
gap. A fresh libuser pass over the newly-landed `Schema`/`SchemaBuilder`/
`Finalize`/Query-view surface (the first review of that surface, mirroring the
#170→#201/#202/#203 harvest) surfaced a real bug — **#210**: `Finalize`'s
`sch-props-correct` clause-2 duplicate-name check false-rejects two legitimately
anonymous (zero-QName) components (e.g. two anonymous `ComplexType`s), which
will very likely block #176 in practice since inline/anonymous complex types are
common — recommend landing #210 before or alongside #176. The same pass also
produced **#211** (worked construct→Finalize→query `Example` + a
`Schema`-implements-`{Type,Element,Attribute}Resolver` doc cross-reference), and
folded a README omission (Library quickstart never mentions `SchemaBuilder`/
`Finalize` despite it being real, working surface) into **#189**'s scope rather
than filing a fourth issue. Ready queue: #210, #211, #208, #203, #202, #195,
#190, #189, #174 (9, within the 8–10 band). **Branch-namespace note:**
`wip/issue-145` is a stale leftover ref (tip `ea21ecd`, 2026-07-18, no unique
commits vs `origin/main` — its issue #145 was closed 2026-07-18 as
already-satisfied and its tip commit is already on `main`); flagged here for
human triage, not deleted by this session.

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
