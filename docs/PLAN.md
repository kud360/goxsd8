# goxsd8 Roadmap

Milestones map one-to-one to GitHub milestones. The cartographer carves
each into session-sized `ready` issues; the develop loop closes them one
per session. Prefer vertical slices that move a conformance lane over
horizontal completeness.

**Conformance-lane counts in this file are date-stamped, never live**
(settled 2026-08-03 for #411; see that pass's paragraph in M4 for the
reasoning). `conformance/testdata/expectations/*.txt` is the only source
of truth for a lane score. Any count restated here — in a milestone
heading, a narrative sentence or a `/backlog` table — carries the date it
was read, so a reader can tell staleness from wrongness without opening
the expectations files. A stamped count is never "corrected" in place
when the lane moves; the next dated paragraph carries the new number.

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

## M3 — Datatypes vertical slice (complete — all 20 primitives mapped, `datatypes` lane 1043 pass / 31 fail (1074) **as of 2026-07-23**, after the list-variety Facets cohort #75 and the `value.effectiveWhiteSpace` union not-applicable path #98 landed that day; the IBM precisionDecimal cohort (#162) and the `Mapping.Canonical` doc (#166) landed 2026-07-19; open datatypes-lane follow-ups: anyURI-triage #190, union member-dispatch #223, integer-family list fixtures #224)

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

Update (2026-07-22, weekly backlog): **#174 (producer spine), #46 (cross-type
variety/base shape + `st-props-correct`/`cos-st-restricts`), and #157
(precisionDecimal maxScale/minScale schema-construction SCCs) all landed since
the prior backlog.** The M4 critical path has therefore advanced to **#176**
(complex-type + content-model producer) — its dependency #174 is closed, so it
is `ready`; it is the single spine item that gates the whole producer fan-out
(#177/#178) and the finalize-validity tail (#180/#181/#206) and composition
(#183). **#175** (schema-lane bring-up — the first real `schema`-lane movement,
the driver that flips `schema.txt` fail→pass) also flipped `ready` (its only dep
#174 is closed) and is the highest-conformance-value item in the queue; #176 and
#175 are the two M4 priorities. The `schema` lane is still **0 pass / 15432
fail** (`stubFail`) until #175 lands.

The develop loop's post-land passes kept the follow-up ledger clean: #214
(producer must OR multiple same-step `<pattern>` facets into one FacetPattern,
§4.3.4.2 — harvested from #174's arbiter advisory), #219 (producer `facetKindOf`
silently drops `maxScale`/`minScale`, leaving #157's new construction-time scale
SCCs unreachable from real schema documents), #215 (tighten
`Atomic.Primitive`/`List.Item`/`Union.Members` to unexported+accessors, T1 — from
#46's warden pre-flight), and #217 (the `cos-st-restricts` facet-value
sub-clauses #46 deferred as out of pure-leaf `xsd`'s reach, needs `value`) are
all filed and `ready`. No untracked GAP debt (both `xsd/namespaceconstraint.go`
GAP markers remain owned by #51).

**#46's landing resolved the long-standing #98/#75 tangle:** #98 (`value`
effectiveWhiteSpace not-applicable path for union/list varieties) was unblocked
`blocked`→`ready` this backlog — #46 makes a non-atomic-variety `SimpleType`
constructible and routable through `value.ValidateLexical` leaf-only, which was
#98's real (previously "unfiled") precondition. Landing #98 then flips **#75**
(datatypes-lane widening to list/union Facets cohorts) to `ready` — a genuine
datatypes-lane vertical slice. #75 stays `blocked` on #98 for now.

Ready queue (16, deep by design — over the 8–10 band but every item is a
well-specified single-session issue, and the depth guards against a stall given
the develop loop's multi-issue/day throughput and the self-feeding M4 spine):
**#176, #175** (M4 spine, top priority), then the independent pool #210, #214,
#219, #98, #179, #217, #215, #211, #208, #203, #202, #189, #195, #190. Persona
coverage: the `xsd` `Schema`/`SchemaBuilder`/`Finalize` surface had a fresh
libuser pass last backlog (#210/#211/#203/#202); a dedicated libuser review of
the newly-published `parser.Produce` surface is **deferred until #176/#178
stabilize the producer** — reviewing the intentionally-partial top-level-only
surface now would mostly re-derive #176. **Branch-namespace note:** `wip/issue-145`
remains the only non-`main` ref (still stale, unchanged since 2026-07-18, issue
#145 closed, tip already on `main`) — still flagged for human triage, not deleted.

Update (2026-07-23, weekly backlog): **the `schema` lane is now live and
moving.** #175 (schema-lane bring-up — the `conformance/schema.go` driver +
first ratchet) and #176 (complex-type + content-model producer, +715 cases)
both landed since the prior backlog, taking the `schema` lane from 0 to
**2731 pass / 12701 fail** (15432). The strictly-serial M4 spine is over; the
producer/finalize fan-out is open and the ready frontier is genuinely
parallel now. Also landed since 2026-07-22: #75 (datatypes list-variety
Facets) and #98 (`value.effectiveWhiteSpace` union not-applicable), draining
the last list/union tangle — `datatypes` now stands at 1043 pass / 31 fail.
The develop loop's post-land passes filed the #176 deferred scope as
#228/#229/#230/#231 and #177's group-ref work, plus #223/#224/#226; this
backlog filed the one remaining #176 arbiter advisory as **#232** (producer
drops an unresolvable `notQName` member — latent false-accept). **No blocked
issue unblocked this pass** — #51/#52/#63/#72/#182/#183/#229 all remain
correctly gated on still-open #178/#179/#181/#210. **Highest-value ready
frontier** (conformance + unblock order): #178 (IDC/assertion/notation/
wildcard producers — also unblocks #51/#63/#72), #210 (Finalize
anonymous-QName false-reject bug — unblocks #229, clears the common
inline-anonymous path), #228 (resolved-base simpleContent/extension content),
#177 (attribute-group/model-group producers), #180 + #181 (finalize UPA/EDC +
derivation validity), #229 (inline anonymous types), #230 (open content),
#231 (`&lt;all&gt;` {0,1}), #179 (composition include — unblocks #182/#183). Ready
queue is **26** — deep by design and above the 8–10 band, but every item is a
well-specified single-session issue and the depth is self-fed by the develop
loop's multi-issue/day harvest, not a planning gap; all labels verified
honest (deps checked). **Branch-namespace note:** two stale leftover refs
remain for human triage — `wip/issue-145` (issue closed 2026-07-18, content on
`main`, unchanged) and now **`wip/issue-98`** (issue #98 closed this cycle, its
work squash-merged onto `main` as `9eac00d`; the branch is a pre-squash
leftover). No `parked/*` refs. No untracked GAP debt: the two
`xsd/namespaceconstraint.go` markers plus the newly-noted
`parser/produce_complex.go:571-574` producer skip are all owned by #51; the
literal-QName drop below it is #232.

Update (2026-07-25, post-land pass for #178 — not a full backlog sweep):
**#177 and #178 both landed**, taking the `schema` lane 2731 → 2866 (+135,
attribute-group/model-group producers) → **2936 / 15432** (+70,
IDC/assertion/notation producers). `datatypes` unchanged at 1043/1074. The
producer fan-out of the M4 carve is now **complete**: #174–#178 are all
closed, and every remaining M4 sub-slice is either composition (#179 include
+ chameleon — **in flight**, #182 import, #183 redefine/override) or
finalize-validity (#180 UPA/EDC, #181 derivation validity). This corrects the
2026-07-23 paragraph above on two points that have gone stale: #178 is no
longer on the ready frontier (it landed), and its claim that
"#51/#52/#63/#72 … all remain correctly gated on still-open #178" no longer
holds for three of those four. **Three of the five M4 leaf follow-ups
resolved this pass:** #51 (`##defined`/`##definedSibling` resolution — deps
#173/#176/#178 now all closed) and #63 (IDC `{referenced key}` +
`c-props-correct` cl.2) flipped `blocked` → `ready`; **#72 closed as
subsumed** — #178 folded in the assertions-facet producer outright, exactly
as #72's own Notes anticipated ("co-schedule with, or fold into, #178"), so
queueing it would have sent a session to rebuild what is already on `main`.
#52 correctly stays `blocked` on still-open #181. Note that #178 touched **no
wildcard code**: the oracle grounding established `produceWildcard` /
`src-wildcard` were already discharged by #176/#177, so the "wildcard
producer" quarter of that issue was a no-op — worth knowing when reading
#51's dependency list, which names #178 for exactly that reason. Post-land
harvest filed **#240** (the `ref=` form of `<unique>/<key>/<keyref>`, declined
honestly by #178 and gated in `conformance/schema.go` so it cannot pass for
the wrong reason) and **#241** (settle the `parser.Element` tree-API export
convention — an arbiter non-blocking note); #177's carried-forward
`conformance/runner.go:164` defect was already filed as **#238**, so nothing
from either landing is left unfiled. Ready queue is **29** — still well above
the 8–10 band and still self-fed by the develop loop's harvest rather than a
planning gap. **Branch-namespace note: the namespace is clean for the first
time in a week** — the two stale refs flagged in the prior two backlogs
(`wip/issue-145`, `wip/issue-98`) are both gone, and `wip/issue-179` is the
only non-`main` ref, live (tip minutes old) and correctly claimed by open
issue #179. No `parked/*` refs. No untracked GAP debt: the three real
`GAP(` sites (`xsd/namespaceconstraint.go:154`/`:276`, `xsd/wildcard.go:110`)
remain owned by #51, which is now `ready` and chartered to retire them.

Update (2026-07-25, weekly backlog): **the M4 carve is down to four
sub-slices.** #179 (multi-document assembly via `<xs:include>` + §F.1
chameleon coercion) and #51 (`##defined`/`##definedSibling` resolution)
landed since the post-land pass above, along with #70 (Attribute Use
`{value constraint}`) and the `cc0397b` index-fix for #244. Of the 17
sub-slices #79 was carved into, only **#180** (UPA/EDC), **#181**
(derivation validity incl. particle restriction §3.9.6), **#182**
(`<xs:import>`) and **#183** (`<xs:redefine>`/`<xs:override>`) remain
open, and none of the four blocks another — the serial spine is fully
behind us. Of the five M4 leaf follow-ups the carve repointed (#72, #70,
#63, #51, #46), **four are now closed** and only **#63** (IDC
`{referenced key}` + `c-props-correct` cl.2) is left, `ready`; #52
correctly stays `blocked` on #181, its only open dependency.

**The `schema` lane has not moved since #178 — 2936 pass / 12496 fail
(15432) — and that is structural, not a quality signal.** #179 moved it
by design zero (the harness still runs `os.Open` → `ReadDocument` →
`parser.Produce` per document, and `schemaShapeDecidable` declines any
top-level `<include>`, so no include or chameleon case is decidable
however correct `Parse` is), and #51's resolution half has no caller
until M5. **#242 is therefore the single highest-conformance-value issue
in the queue**: it reroutes the lane onto `parser.Parse` with a rooted
`loader.Dir` and admits the include/chameleon cohort, converting #179's
22 unit tests into lane evidence. Treat it as the top of the frontier.
Other lanes: `datatypes` unchanged at 1043 / 31 (1074); `instance` at 0 /
26426; `xpath`, `json`, `ber` still empty by design.
*(Superseded 2026-07-25 by the post-land paragraph at the end of this
section: #242 landed, and the lane now stands at 2951 / 12481.)*

**Priority order for the ready frontier** (lane movement first, per this
file's own preamble — the queue is deep enough that an explicit order is
the useful output of a backlog). All 34 `ready` issues appear exactly
once, so a session can take the first item it can do rather than scan:

1. **Schema-lane movers.** ~~#242~~ (**LANDED** 2026-07-25, `71b07a2`),
   **#180** + **#181** (finalize
   validity; the only things that unblock #52, and #181 is what licenses
   the M5 matcher's no-backtracking guarantee), **#210** (the `Finalize`
   anonymous-QName false-reject — unblocks both #229 and #249 and clears
   the common inline-anonymous path), **#228** (resolved-base
   simpleContent/extension `{content type}`), **#230** (open content),
   **#231** (`<all>` {0,1} occurrence grammar), then **#257** → **#182**
   → **#183** (composition tail, in that order — see the post-land
   paragraph below for why #257 precedes #182).
2. **Correctness / false-accept debt.** **#214** (patterns wrongly
   AND-ed), **#219** (scale facets dropped, making #157's SCCs
   unreachable), **#240** (`ref=` form of
   `<unique>`/`<key>`/`<keyref>`), **#238** (multiple
   `<ts:schemaDocument>` children — wrong-document decisions), **#226**
   (UTF-16 BOM misread as invalid UTF-8), **#202** (Required name/ref
   slots accept the absent zero-QName), **#253** (silent short assembly).
   *Retired: #232 (unresolvable `notQName` dropped) landed 2026-07-31 as
   `07ca0178` (PR #355) — the literal-QName arm now propagates
   `src-resolve`; schema lane +2 (`wild036`, `wild037`).*
3. **Remaining feature leaves.** **#63** (IDC `{referenced key}` +
   `c-props-correct` cl.2 — the last open M4 leaf follow-up, and an M5
   prerequisite per #250), **#206** ({context} / {scope}.{parent}
   containment back-pointers), **#217** (`cos-st-restricts` facet-value
   sub-clauses), **#236** (the §3.5.4 effective value constraint, now
   computed over an Attribute Use `{value constraint}` the producer
   actually populates).
   *Retired: #235 (attribute `default=`/`fixed=` → Attribute Use
   `{value constraint}`) landed 2026-07-31 as `84a7431` (PR #357) — both
   `NewAttributeUse` call sites stopped passing `nil`, so `key-evc` feeds
   `loc-testSubP` clause 5.2 on real input; schema lane +2 (`attZ008_f`,
   `attZ008_h`). Post-land harvest filed **#358** (`src-attribute`
   clauses 2/5 unenforced; `use="prohibited"` short-circuits clause 1)
   and **#359** (`defaultbinding.go`'s five zero-`Loc` rejections). The
   `au-props-correct` clause-3 `{value}`-identity half is NOT discharged
   — it belongs to #236, not to the closed #173, whatever the older doc
   comments in `xsd/attributeuse.go` still say.*
4. **Datatypes-lane widening.** **#223** (union member-dispatch),
   **#224** (integer-family list fixtures), **#190** (anyURI triage).
5. **Surface, docs, process — real work, but it moves no lane, so it
   should be picked deliberately rather than by falling off the top of a
   list.** **#252** (Schema enumeration accessors — a prerequisite for
   the first `goxsd8 parse`), **#251** (`-help`), **#189** (README sync),
   **#203** (the consolidated `xsd` doc/example session), **#208**
   (`Loc()` accessors), **#215** + **#241** (exported-surface tightening),
   **#195** + **#246** (process guards).

**Ready queue is 34 — over the 8–10 band for the fourth backlog running,
and this is now a standing condition rather than a one-off.** Every item
was re-checked this pass and every `ready`/`blocked` label is honest (no
`ready` issue has an open hard dependency; #236's open reference to #235
is a soft sibling, not a gate). The cause is structural: the develop
loop's post-land harvest files follow-ups faster than one-issue-per-
session consumes them, which is the harvest working as designed. The
mitigation is the explicit priority order above, not deletion of real
work — but note that **10 of the 34 move no conformance lane**
(#189, #195, #203, #208, #215, #241, #246, #251, #252 and the doc half of
#253), so a session picking off the top of an unordered list will tend to
drift away from the lanes.

Filed this backlog: **#250** (the M5 epic — see below), **#251**
(`-help` prints the not-implemented stub and exits 2; cliuser),
**#252** (`*xsd.Schema` has point lookups but no document-order
enumeration accessors, so `goxsd8 parse`'s summary cannot be written
against the published surface — libuser and cliuser converged on this
independently), **#253** (`parser.Parse` assembles short on
import/redefine/override with no observable signal; libuser).
Consolidated: **#211 closed as a duplicate of #203** — both were
libuser-harvested doc-only `area/xsd` issues adding an `Example*` to the
same example-test file and editing the same package doc, so they are one
session; #203 now carries all of #211's criteria plus two new libuser
gaps (the manual `ContentType`→`Particle`→`TermOrRef`→`Term` traversal
example, and naming `SchemaBuilder`'s intended caller). **#16** was the
only open issue carrying neither `ready` nor `blocked` nor
`needs-replan` — now `blocked`, which is the honest encoding of its own
"a reference, not a session" note, with today's cliuser criteria for the
first working `goxsd8 parse` folded into its thread.

**GAP debt: exactly ONE real site, which corrects the paragraph above.**
That paragraph's "three real `GAP(` sites ... remain owned by #51" is
stale in both halves: #51 *landed*, retiring both
`xsd/namespaceconstraint.go` markers substantively, and the surviving
marker is at `xsd/wildcard.go:107` (not `:110`), owned by **#248**.
`xpath/doc.go:29` is the convention's own template, not a site. The
marker's text still names #51 rather than #248; repointing it is an
acceptance criterion of #249.

**Branch-namespace reconciliation: clean.** `git ls-remote --heads
origin` returns `main` alone — no `wip/*`, no `parked/*`, nothing to
retire, and no closed issue references a branch that never landed
(#179's work is `547b42f` and #51's is `566ef1f`, both on `main`).

**Known drift this session could not fix:** none of the 39 open issues
carries a GitHub milestone, so the preamble's "milestones map one-to-one
to GitHub milestones" is currently aspirational — issue labels and this
file are doing that work alone. This session's GitHub toolset can neither
enumerate nor create milestones (no milestone tool; the REST fallback is
unauthorized), so it is flagged for a human or a session with a broader
token rather than guessed at.

Update (2026-07-25, post-land pass for #242 — not a full backlog sweep):
**#242 landed** (PR #258, squash-merged `71b07a2`), taking the `schema`
lane **2936 → 2951 pass / 12481 fail (15432)** — the first movement since
#178, and the discharge of the paragraph above that called the lane's
stasis "structural, not a quality signal". `execSchemaCase` now decides
via `parser.Parse` over a rooted `loader.Dir` (multi-document
`<xs:include>` assembly) instead of single-document `parser.Produce`, and
a new independent closure-discovery walk (`conformance/schema_closure.go`)
runs `schemaShapeDecidable` over **every** document of the assembly rather
than only the root, so a silently-skipped representation inside an
`<include>`d document cannot false-accept. The 15 flipped cases span
MS-Schema, MS-Particles, MS-SimpleType, MS-AttributeGroup, MS-Additional
(the include-the-same-schema-twice diamond), MS-DataTypes, MS-ComplexType
and MS-Annotations. Zero regressions on any lane. Other lanes unchanged:
`datatypes` 1043 / 31 (1074); `instance` 0 / 26426; `xpath`, `json`, `ber`
empty by design.

**Two in-place edits were made above rather than left to be superseded,
because that text is operational and would misdirect a session:** the
"#242 is therefore the single highest-conformance-value issue" sentence is
marked superseded, and #242 is struck from priority band 1. Nothing else
in the 2026-07-25 weekly backlog paragraph was rewritten.

**Nothing was unblocked by this landing.** All 8 open `blocked` issues
were checked; none names #242 in its `## Depends on` (#250→#79, #249→#210
+#229, #248→#250, #229→#210, #79→none, #56→the unfiled M6 evaluator,
#52→#176/#181, #16→none). #242 was a harness-wiring change with no issue
gated behind it — expected, and recorded so the next pass does not re-derive it.

**Post-land harvest: one issue filed, one folded, one confirmed.** #257
(the `loader.Dir` `..`-confinement vs. `src-include` clause 2.4
indistinguishability) was filed *during* landing as an explicit
precondition of the arbiter's ACCEPT, not as routine harvest;
re-checked this pass and confirmed correctly labeled and scoped
(`ready`, `kind/gap`, `area/conformance`, no dependency). Of the two
non-blocking arbiter findings the log records as "deliberately not
filed", the judgment **split**:

- **Filed as #259** — the duplicated `resolveSchemaLocation`
  (`parser/parse.go:312` and a verbatim copy at
  `conformance/schema_closure.go:204`). Re-reading both copies changed
  the assessment from the "small T4 concern" framing: the sync
  obligation is asserted only on the copy, `parser/parse.go` carries no
  back-reference, and the drift consequence is not a red test but the
  closure walk under-discovering — i.e. a document `Parse` reads whose
  shape was never gated, which is the exact false accept the walk
  exists to close. **A false accept moves the lane UP, and the ratchet
  only guards downward movement, so drift would be recorded as progress
  and locked in.** That is priority band 2 (false-accept debt), not
  band 5. The T5 "exporting adds public surface" justification in the
  copy's comment is also a false dichotomy — a module-`internal/`
  package satisfies T4 and T5 at once (there is no `internal/` tree
  today, so placement wants a steward/warden read). The concrete drift
  trigger is near-term: **#182** resolves import hints through the same
  §4.3.2 clause-4 path and must add an import edge to the closure walk.
- **Not filed; folded into #257 as a rider** — the doc-completeness nit
  in `conformance/schema.go`'s step-4 comment (its list of eliminated
  plain-error types omits the missing-`schemaLocation` grammar fault,
  which `closureScan.include` does correctly decline). Verified against
  the landed code: genuinely cosmetic, the conclusion it supports still
  holds, and it is a one-line edit in a file #257 already opens — filing
  a session for it would cost more than the fix. Precedent: #249 carries
  #248's one-line marker repoint the same way.

**Next priority, and a correction to the log entry's speculation.** The
log entry suggested #182 (`<xs:import>`) as the natural successor. Its
dependency #179 is indeed closed (`547b42f`) and it is correctly `ready`
— but it is **not** promoted over #180/#181, which keep the top of band 1:
they are the last non-composition M4 slices, #181 is the only thing that
unblocks #52 and is what licenses the M5 matcher's no-backtracking
guarantee, and nothing in #242's landing weakened either. More
importantly, the log's second argument for #182 — that it is "the
scenario under which #257's latent path could become live" — is an
argument for **sequencing #257 first**, not for promoting #182: closing a
latent false-accept path before the change that activates it is the cheap
order, and the reverse ships #182 with a live gap and no test watching it.
Band 1 is therefore ordered **#180 / #181 / #210 / #228 / #230 / #231,
then #257 → #182 → #183**, with **#259 landing before #182** for the same
reason (so #182 extends one resolver instead of remembering to edit two).

Ready queue is **35** (34 + #259) — still well above the 8–10 band, still
self-fed by the develop loop's harvest rather than a planning gap. Branch
namespace: `git ls-remote --heads origin` returns `main` alone; `wip/issue-242`
was auto-deleted at merge and its content is on `main` as `71b07a2`. No
`parked/*` refs. GAP debt unchanged: one real site
(`xsd/wildcard.go:107`, owned by #248).

Update (2026-07-25/26, develop loop — #63, #180, #262 — not a full backlog
sweep): **the `schema` lane moved twice more since the paragraph above.** #63
(`c-props-correct` clause 2 — keyref `{fields}` cardinality vs. the resolved
`{referenced key}`, the last open M4 leaf follow-up) landed 2026-07-25 with no
lane movement (no driver exercises keyref resolution yet). #180 (Unique
Particle Attribution `cos-nonambig` + Element Declarations Consistent
`cos-element-consistent`, a Glushkov position-automaton reduction of Appendix
J's non-normative guidance) landed the same day, taking `schema`
**2951 → 2988 (+37)** — the session's highest-stakes change, closed with a
first-pass arbiter ACCEPT after a design pre-flight and three grounding
rounds; see `docs/LOG/2026-07.md`'s 2026-07-25 #180 entry for the full
account, including the non-blocking finding that the `<all>`-emptiability
carve-out it introduced was correct but unpinned (filed as **#261**). #262
(`derivation-ok-restriction` §3.4.6.3 clauses 1/2.1–2.3/2.4.1/3/4/5 plus the
remaining `ct-props-correct` clauses 2/4) landed 2026-07-26, taking `schema`
**2988 → 2997 (+9)**, zero new exported identifiers. **#181 is formally
retired**, split three ways as its own closing comment records: #262
(closed, above), **#263** (`cos-content-act-restrict` §3.4.6.4 — clause
2.4.2's delegate, the marathon slice, `ready`), and **#264**
(`cos-ct-extends` §3.4.6.2 — `blocked` on #228, since nothing produces
`<extension>` content yet to validate). #52 and #265 were repointed off the
retired #181 onto #263/#264 as their real `cos-ns-subset` consumers.
`datatypes` unchanged at 1043/31 (1074); `instance` 0/26426; `xpath`/`json`/
`ber` empty by design.

**Post-land harvest, both landings:** #180's harvest is #249 (rewritten to
absorb #180's substitution-group split — `mayBe`/`certainlyInSubstitutionGroupOf`
— rather than filed twice) and #261 (the emptiability regression test) — both
already `ready`/`blocked` correctly, re-verified this pass. #262's harvest is
**#265** (three attribute-side seams: `cos-ns-subset` for clause 3's wildcard
half, the §3.4.2.4 clause-3 attribute-use inheritance fold, and — added by a
same-day post-land amendment — the §3.4.2.5 clause-2.2 extension
base-wildcard fold; `blocked` on #52, though its section 2 is independently
`ready`-shaped per its own body). Two of #262's four disclosed GAPs were
already routed to existing issues by same-day post-land comments before this
backlog ran: `xsd/defaultbinding.go:323` (the fixed-value lexical-mismatch
outcome) to **#236**, and the assertions-prefix vacuity note folded into
**#265**'s body. **This backlog's own GAP sweep found the one disclosed GAP
that neither post-land pass had adopted**: `xsd/defaultbinding.go:81`
(`key-dft-binding` case 3 — a wildcard-attributed item with a ·governing
attribute declaration· synthesizes an Attribute Use, which needs
assessment-episode information the static `xsd` layer does not have) —
filed as **#267**, `blocked` on #250, since the likely resolution is a
documented permanent-fail-open ruling once the M5 instance validator exists
to decide case 3 correctly at assessment time, not new schema-static code.
**GAP debt is otherwise clean**: of the 13 `GAP(` sites in the tree
(`xpath/doc.go:29` is the convention's own template, not a site), 12 are
now owned by an open issue (#248, #249, #263, #265, #267 — #236 and #265
each own one via a post-land comment rather than their body) and zero are
unowned.

**Ready-queue reconciliation:** net effect of the two landings plus the
retirement is a wash — #262 and #63 left the queue, #263/#264/#265/#267
joined it (three `blocked`, one `ready`) — so the queue is **35** open
`ready` issues, unchanged in size from the paragraph above, still
self-fed rather than a planning gap. **Priority band 1 is amended in
place** (the stale text otherwise misdirects the next session): strike
`#180` (landed) and replace the unified `#181` line with **`#263`**
(the direct continuation of #262's restriction-validity work — same
finalize phase, same file, already `ready`), keeping the rest of the
order — band 1 is now **#263, #210, #228, #230, #231, then #257 → #182 →
#183** (`#259` still lands before `#182` for the same reason as before).
`#264` and `#265` stay off band 1 pending #228/#52. Band 3 drops `#63`
(landed). No other band changes.

**Branch namespace: still clean.** `git ls-remote --heads origin` returns
`main` alone — no `wip/*`, no `parked/*`. `wip/issue-262` was auto-deleted
at merge; its content is on `main` as `f02e934`.

**#250's own body is now slightly stale** (its `## Depends on` says "in
practice the blocking tail is #180 UPA/EDC and #181 derivation validity" —
both true when #250 was filed, both since resolved/retired) — flagged with
a clarifying comment on the issue rather than rewritten, since #250 is an
epic body and the rewrite would lose the original filing rationale; the
comment names the current tail (#182, #183, #263, #264, #265) instead.

Update (2026-07-31, weekly backlog): **the ORIGINAL M4 carve is fully
closed — every one of #79's #167–#183 sub-slices** (#182 landed
`aea7be2`, #183 landed `80352b3`, both 2026-07-27; #181 closed as
`[SPLIT → #262 / #263 / #264]`). #79 now shows **19 of 22 sub-issues
complete**, the extra five being continuations chartered after the carve:
#262 and #263 closed, and **#264, #286 and #287 still open**. So the
epic stays open and `blocked` — but on three named slices, not on an
unknown remainder, and **the M5 carve's precondition is those three
closing**, not the #180/#181/#182/#183 tail the M5 section below still
names. Thirteen landings since the paragraph above: #202, #203, #206,
#208, #210, #214, #215, #217, #219, #223, #224, #228, #229. **What M4 is
missing is mostly fan-out now**: `&lt;xs:redefine&gt;` (#286, the deferred half
of #183), §F.2 duplicate-`&lt;xs:override&gt;`-child handling (#287),
`&lt;openContent&gt;` (#230), the `ref=` form of the identity constraints
(#240), and the two open halves of derivation validity (#264, #265).

**Both lanes moved hard, and the M3 and M4 headings above are now stale
in their numbers — read this paragraph, not them.** `schema` is **4284
pass / 11148 fail (15432)**, up from 2951 at the #242 post-land paragraph
and 2997 at #262's baseline: **+1287** across the window, the bulk of it
#229's inline-anonymous-`simpleType` cohort (+1190, a cohort nobody had
predicted). `datatypes` is **1059 pass / 22 fail (1081)**, up from
1043 / 31 (1074) — #190, #223 and #224 all landed, so the M3 heading's
"open datatypes-lane follow-ups: anyURI-triage #190, union
member-dispatch #223, integer-family list fixtures #224" names three
**closed** issues. The heading is left as filed per this file's
add-don't-rewrite convention; this sentence is the correction. `instance`
0 / 26426; `xpath`, `json`, `ber` empty by design.

**Priority band 1 is replaced wholesale** — every issue in the previous
band-1 line has landed except the composition tail. The working queue,
lane movement first, dependency-ordered, is now:

1. **#368** — `symbols.attributeGroups` carries no owning producer, so a
   foreign `<attributeGroup ref>` is folded under the ASKING document.
   A **chameleon false reject** — the one direction the ratchet cannot
   forgive. This slot is *substituted*, not vacated: #368 is the same
   defect class as the #337 that used to sit here, one index over and the
   third instance after #228, so retiring #337 without promoting it would
   contradict the very reason #337 was ranked first. Strictly the worse
   of the two — an `<attributeGroup>` body holds local `<attribute>`
   declarations, so `localTargetNS` is misread as well as
   `unqualifiedRefNS`, and the fix has two landed precedents to copy.
2. **#264** → **#265** — the two open halves of complex-type derivation
   validity. Together they unblock **#336**, which is where the schema
   lane widens to admit the extension forms; #265 also now owns the
   `contentrestricts.go:356` sibling-keyword fold.
3. **#301** — `ComplexType.{context}`, the component-handle identity
   **#340** consumes (inline anonymous `complexType` on local
   declarations — the complexType twin of what #229 just landed for
   `simpleType`, and the same cohort shape).
4. **#281** — `substitutionGroup=` into `{substitution group
   affiliations}` (the producer passes nil today). Retires
   `contentrestricts.go:462`'s global/global fail-open and unblocks
   **#342**.
5. **#346** — the base type's `{assertions}` fold (§3.4.2.1 cl.1), which
   makes `derivation-ok-restriction` clause 5 chargeable instead of a
   guaranteed false reject. Filed this pass.
6. **#324** — `xsdBool` reads `fixed=` too narrowly; an out-of-lexical-
   space value silently becomes `false` instead of charging
   `cvc-datatype-valid`. False-accept debt.
7. **#277** + **#276** — the two ratchet-integrity soft spots in the
   harness (`indeterminate` mapped to expects-invalid, so 16 cases are
   winnable by any rejection; an unresolvable `&lt;xs:include&gt;` location
   admitted where #182 made `&lt;xs:import&gt;` decline the same shape). These
   move no lane by design and are listed last, but they are the only
   items here whose *absence* can lock a false accept into the ratchet.

*Retired from this band, all four on 2026-08-01, in the order they were
listed ("both" above was written when there were two; the count is the only
word changed):*

- *#344 (UTF-16 BOM decode, the #226 replan) landed as `2c70354` (PR #362)
  — `schema +5`, not the `+4` this band predicted; the fifth flip
  (`assertion/d4_3_15ii30`, another `ibmData` `ff fe` document) was traced
  to the diff before banking rather than tolerated. **The "measured,
  unbanked `schema +4`" this entry used to carry is therefore discharged**
  — it is banked, and no later lane movement can be misattributed to it.
  Post-land harvest filed #361 (BOM-less UTF-16 declared only by
  `encoding=`, plus the legacy single-byte `EncName`s — the residual
  `GAP(xml)`) and #363 (three advisory leftovers). Retired here by the
  #331 post-land pass, which found the entry stale; it is not that pass's
  own work.*
- *#331 (route seeded-but-unmapped lexical types through
  `value.ValidateLexical`) landed as `c488b47` (PR #364) — **`datatypes`
  +48, 1081 → 1129, all 48 `New@pass`, 0 Regressed, 0 Vanished**, arbiter
  ACCEPT on round 1 and mason's `Ratchet:` forecast confirmed exactly.
  `execLexicalCase` no longer demands a **direct** `backend.Mapping` for
  the tested type; `value.governingMapping` walks the base chain to
  `xs:decimal` and the type's own effective facets decide the literal
  (`cvc-datatype-valid` §4.1.4 is a conjunction — `parseOK` satisfied only
  clause 2.1 and provably false-accepted `"128"` as an `xs:byte`).
  `fixesTimezone` stayed an independent OR arm because the oracle proved
  the two conditions independent, not by oversight. #224's
  `integerListCase` was retired as a strict subset (STYLE D3). Post-land
  harvest filed **#365** — the `xs:int`/`xs:integer` lexical fixtures
  #331 deliberately left outside its enumerated 48, which that pass found
  to be **24 files (int001–008, integer001–016)**, not the 20 that #331's
  prose, its LOG entry and three `conformance/datatypes.go` comments all
  name; `integer013–016` exist with the identical shape. #365 owns
  correcting the count as well as widening the selector, and unlike #331
  it inherits **no measurement**, so its `Ratchet:` figure must come from
  a diagnostic run (#354).*
- *#337 (`symbols.simpleTypes` gains an owning producer) landed as `1cd1ce1`
  (PR #367) — **ratchet unchanged** (schema 4312/15432, datatypes 1107/1129,
  instance 0/26426, md5-verified flat by the arbiter), arbiter ACCEPT on round
  1. That flatness is the expected result, not a disappointment: **no W3C
  fixture combines chameleon inclusion with an unqualified simple-type
  `base=`**, so this was a false-reject fix guarded by its own two-order unit
  test rather than a lane mover — and the band ranked it first for exactly
  that reason. #228's `complexSource{elem, owner}` was **generalized into
  `typeSource`** rather than twinned (STYLE T4), so there is now one
  source-entry encoding for both type indexes. The stale "simpleTypes needs no
  owner" doc comment was deleted, not softened. Post-land harvest filed
  **#368**, which takes over this band's item 1 above.*
- *#368 (`symbols.attributeGroups` gains an owning producer) landed as
  `3e97b77` (PR #378) — **ratchet unchanged** (schema 4247/15432, datatypes
  1107/1129, instance 0/26426, md5-verified flat by the arbiter), arbiter
  ACCEPT on round 1. **This slot is now vacated, not substituted**: #368 was
  the third and last instance of the no-owning-producer class (#228
  `complexTypes`, #337 `simpleTypes`, #368 `attributeGroups`), so unlike
  #337 there is no fourth index to promote into item 1, and **#264 → #265 is
  now this band's head**. Flatness was again the predicted result — no W3C
  fixture combines chameleon inclusion with a cross-document
  `<attributeGroup ref>` — so this was a false-reject-only fix guarded by a
  two-order unit test. `typeSource{elem, owner}` was reused for the third
  time running rather than twinned (STYLE T4). The one thing this landing did
  **not** inherit from its two precedents: the oracle recommended reusing the
  already-built `AttributeGroupDefinition` component (its reading of
  `c-add2`/§3.6.2.1 was correct) and mason shipped the raw-element walk routed
  through `src.owner` instead; the arbiter settled it by finding that
  `compile()` runs every `prescan()` before any `run()`
  (`parser/parse.go:541-551`), so in a referrer-first order the component does
  not exist yet at fold time. **A grounding's *what* can be sound while its
  *how* is a guess about code the oracle cannot read** — worth carrying into
  future groundings that recommend an implementation shape.
  Post-land harvest filed **nothing**, deliberately, on three separate
  advisories; see the two paragraphs below.*

***Recorded, deliberately not filed: no fourth raw-`*Element` index.*** The
#368 arbiter closed its verdict by asking for "a standing check that no fourth
raw-`*Element` index appears" now that the class is closed at all three sites.
There is nothing to file: the check is a *review* habit, not a change, and the
three landed precedents plus `typeSource`'s now-generalized doc comment already
encode it at the only place a fourth index could be minted. An issue whose
`## Acceptance` is "no new occurrences of a pattern" can never be closed by the
develop loop and would sit in the queue forever, so it is recorded **here**, in
the ledger `/backlog` surveys, as a thing to look for in the exported-surface
and symbols-table diff of any future `parser/produce.go` change.

***Also deliberately not filed, and NOT a defect:*** the same verdict's D4
observation that `collectAttributeContent`'s `visited` set is a "seen set"
smell. The arbiter itself ruled it **pre-existing on `main`, unchanged by
#368, and correct** — it raises no error and computes the transitive closure
§3.6.2.1 mandates. It is written down in the #368 LOG entry and here **only**
so a future D4 sweep does not misread it as introduced by this landing; it is
not a follow-up, and re-filing it as one would contradict the verdict.

***Recorded, deliberately not filed: `<list itemType=…>` and
`<union memberTypes=…>` will need the same owner routing when they land.***
§F.1's normative XSLT matches `xs:list/@itemType` and `xs:union/@memberTypes`
alongside the `@base` this pass fixed, so those two attributes carry the
identical unqualified-QName exposure. **There is no defect today and no issue
to file it against**: `restrictionOf` (`parser/produce.go:547`) rejects a
`<simpleType>` with no `<restriction>` child outright, so no code path can
reach either attribute, and no open issue or milestone tracks adding the
list/union varieties to the *producer* — #46 and #75 closed the `xsd` and
`value` halves of the variety shape, but the parser half is unfiled and
unscheduled. Filing a speculative `blocked` issue against a dependency that
does not exist would be a body full of "n/a" and would deepen a `ready` queue
already at 65 (#347). Filing it as a `## Notes` line on #368 would only move
the leak: #368 closes, and the note becomes exactly the untracked
closed-issue advisory #270 and #315 exist to stop. So it is recorded **here**,
in the ledger a session actually surveys, to be picked up by whatever issue
first carves parser-side list/union support. Grounding to reuse when that
happens: #337's oracle comment,
<https://github.com/kud360/goxsd8/issues/337#issuecomment-5149740049>
(EDGE CASES, final bullet).

**#286 and #287 sit alongside this band rather than inside it**, and a
session that wants the highest *planning* leverage should take them: with
#264 they are the last three open sub-slices of #79, so closing all three
closes the M4 epic and makes the M5 carve the next planning action. They
are ranked below band 1 only because each moves less lane than the items
above it.

Bands 2-5 from the 2026-07-25 paragraph still stand for everything not
listed above, minus the landed items.

**GAP debt is no longer clean, and the 2026-07-26 paragraph's "12 of 13
owned, zero unowned" is overtaken.** `grep -rn "GAP(" --include=*.go`
now returns **23 real sites** (plus `xpath/doc.go:29`, the convention's
own template, and two `_test.go` references pinning existing markers) —
#262, #263, #52, #228 and #229 each disclosed new fail-open as they
landed, which is STYLE 9 working. This pass swept all 23 and found
**five with no owning issue**; all five are now owned:

- Filed **#345** (`blocked` on #250) for the three that only an
  **assessment episode** can decide, so no amount of schema-static code
  retires them: `defaultbinding.go:266` (`loc-testSubP` clause 3 — the
  W3C `MS-ComplexType` `ctG007`/`ctO003` pattern would false-reject),
  `defaultbinding.go:353` (element-side fixed `{value constraint}`
  compared lexically because `xsd` must not depend on `value`), and
  `contentrestricts.go:515` (element-side `key-dft-binding` cases 4/5).
  They are the family #267 already belongs to.
- Filed **#346** for `complexderivation.go:189` — closable today, hence
  `ready` while the other three stay `blocked`.
- Claimed `contentrestricts.go:356` for **#265** by comment: the
  sibling-keyword drop is `cos-ns-subset`'s missing case, and building a
  private subset test for it would put one relation in two encodings.

**Branch namespace: two stale `wip/*` refs, both correctly retired in
place, neither deletable by a session** (docs/WORKFLOW.md makes this
report-only). `git ls-remote --heads origin` returns `main`,
`wip/issue-195` and `wip/issue-226`; no `parked/*`.

- **`wip/issue-195` @ `63c2e69`** — issue **closed** 2026-07-28 carrying
  `needs-replan`. Content is deliberately **not** on `main` (a park is
  not a land) and the replan is tracked by **#304**, `ready`. Nothing is
  owed; the ref is a leftover flagged for human deletion.
- **`wip/issue-226` @ `80ef0c3`** — parked 2026-07-31 after two arbiter
  rejections. It had **no replacement issue**, which is the one real gap
  this pass found in the branch namespace: **#344** is now filed and
  #226 is closed as superseded, matching the #195 → #304 disposition.
  The branch stays retired and is never resumed; #344 copies the files
  forward off `origin/main`. Its unbanked `schema +4` is recorded in
  band 1 above so it cannot be misattributed to whatever next touches
  the lane.

**Ready queue is 65, and this pass is the first to file the reason as an
issue rather than re-explain it here.** Reconciliation this pass was
real but net-zero: **#317, #332 and #339 were absorbed into #315** (all
four are one prose clause block in one `docs/WORKFLOW.md` bullet list,
and #317 and #332 each asked in writing to be absorbed; #315's body was
rewritten in place to carry all five defect classes with their evidence),
**#300 was absorbed into #338** (the same one-sentence `xsd` doc-truth
edit at an eighth site, exactly the fold-in #300 asked for), and
**#226 was closed as superseded**. Filed: #344, #345, #346, #347. Three
`ready` out, three `ready` in.

That flatness is the finding. The band was last met on 2026-07-21 (9);
since then every backlog has recorded the overrun and applied the same
mitigation — 16, 26, 34, 35, 65 — because there are exactly three
ways to shrink `ready` and all three are unavailable: closing real work
violates the anti-leak convention this repo has defended nine times
(#111, #112, #118, #128, #149, #195, #270, #303, #327); relabelling to
`blocked` would be dishonest and would destroy the invariant the
2026-07-25 pass established (*no `ready` issue has an open hard
dependency* — spot-checked this pass across band 1 and every issue this
pass touched, not re-verified across all 65); and merging only
works for genuinely coupled slices, which this pass exhausted. The cause
is unchanged and is a feature: **the post-land harvest files follow-ups
faster than one-issue-per-session consumes them.** So the fix is not
another paragraph here — it is a decision about what `ready` *means*,
and that is now **#347**: either separate the working queue from the
backlog with one new label, or retire the numeric band and formalize the
ordering as the cartographer's real deliverable. It is filed rather than
applied because it changes `.claude/agents/cartographer.md` itself, which
a cartographer should propose through the queue, not apply to itself
mid-sweep. **Until #347 is decided, band 1 above IS the working queue** —
11 issues across its 9 entries, which is the band, near enough — and the
other 54 `ready` issues are the backlog behind it.

**Personas were NOT consulted this pass, and the API/CLI-facing issues
are unrefreshed as a result.** This session's toolset has no
subagent-spawn tool, so libuser and cliuser could not be run under the
isolation their value depends on (godoc + README only, never source) —
and a persona pass faked by a session that has read the source is worth
less than no persona pass. #251 (`-help` behind the stub), #252 (no
document-order enumeration accessors) and #16 (the CLI contract
acceptance criteria) therefore carry their 2026-07-25 criteria
unchanged. Flagged for a session with a broader toolset, in the same
spirit as the milestone-tool gap the 2026-07-25 pass flagged — which is
**still open**: no open issue carries a GitHub milestone, so the
preamble's "milestones map one-to-one to GitHub milestones" remains
aspirational and this file plus issue labels are doing that work alone.

**Addendum (same date, broader-toolset session): the persona gap above
is closed.** libuser and cliuser ran under the orchestrating session
(which does have subagent-spawn access) against the current published
surface — README + `go doc`, and the real binary for cliuser — never the
source. Result: **no new scope, both confirmations posted as comments.**
#252's gap reproduces exactly as filed (`*xsd.Schema` still exposes only
`Element`/`Attribute`/`Type` point lookups); #251's bug reproduces
exactly as filed (`-help`/`-h`/`--help` still exit 2 via the generic
stub); #189 (closed 2026-07-28) has **not regressed** — the README still
matches `go doc` in both directions, confirming the fix held. One
genuinely new, non-blocking cross-cutting finding from cliuser, posted to
#251 and #16: no flag is reserved for version discovery, and `-v` is
already claimed for debug verbosity, so pin a `-version` convention
before any subcommand's flag parsing lands rather than colliding with it
later.

Update (2026-08-01, weekly backlog — the second `/backlog` in as many
days, run after eleven landings): **the `schema` lane is LOWER than the
paragraph above records, and that is the most important fact on this
page.** Current numbers, read off
`conformance/testdata/expectations/*.txt` at `ee1226b` and cross-checked
against every `Ratchet:` trailer in the window:

| lane | pass / total | movement since 2026-07-31 |
|---|---|---|
| `schema` | **4247 / 15432** | 4284 → 4247, **net −37** |
| `datatypes` | **1107 / 1129** | 1059 / 1081 → 1107 / 1129, **+48** |
| `instance` | 0 / 26426 | unchanged |
| `xpath`, `json`, `ber` | empty by design | unchanged |

The `schema` arithmetic, in landing order, so no one re-derives it:
4284 → #230 **+12** → 4296 → #231 **+7** → 4303 → #232 **+2** → 4305 →
#235 **+2** → 4307 → #344 **+5** → 4312 → #236 **+4** → 4316 → #238
**−69** → **4247**. Eight landings earned **+32** gross and one
authorized downward correction erased it and more. #309, #337 and #368
were verified flat by the arbiter (#309 over a *non-empty* suite —
41858 cases, all seven expectation files md5-identical — which is the
distinction #309 existed to create).

**The −69 is the finding, not the loss.** #238 discovered that
`validityTest.SchemaDoc` was a scalar, so `encoding/xml` overwrote it on
each repeated `<schemaDocument>` child and the harness had been rooting
`parser.Parse` at whichever document the catalog happened to list LAST.
Sixty-nine `pass` lines were therefore wins against a document the suite
had not put under test — right answers to the wrong question. The
arbiter corrected them downward under #238's own authorization, zero
upward flips, the flipped set identical to the verified set. **Nothing
about the ratchet's one rule bent**: a wrong-reason pass is not a score
to protect.

**Band 1 is reordered on the strength of that, and the reordering is
this pass's main output.** The 2026-07-31 band ranked #277 and #276 dead
last with the note *"These move no lane by design and are listed last,
but they are the only items here whose absence can lock a false accept
into the ratchet."* #238 converted that hypothetical into a measured 69,
in the same package, eight days later. The cost of a wrong-reason pass
compounds — the never-regress wall makes it expensive to unwind *later*,
not now — so harness integrity goes to the front while the lane is still
young enough to pay for it. The working queue, dependency-ordered, is
now:

1. **#277** — `resolveExpected` maps `validity="indeterminate"` to
   expects-invalid, so 16 suite cases are winnable by ANY rejection.
   Re-verified against the CURRENT `schema.txt`: exactly two are banked
   today, `MS-Element2006-07-15/elemZ031` (line 4679) and
   `MS-Schema2006-07-15/schZ015` (line 9845), both still `pass` at the
   line numbers the issue body cites — the #238 correction did not touch
   them. The other 14 are `fail` and are harvestable by any future rule
   that rejects for any reason at all.
2. **#276** — the closure scan ADMITS an unresolvable `<xs:include>`
   location where #182 made `<xs:import>` DECLINE the same shape. Same
   class as #277 and now the same neighbourhood: #238 rewrote
   `conformance/schema_closure.go`'s decide path, and
   `importDirective`'s doc comment (`:253-256`) now argues the asymmetry
   explicitly rather than leaving it implicit — so the file is fresh and
   the argument is written down to agree or disagree with. **#377 rides
   with this one** (`extraDocsInClosure`'s decide path has no
   `<override>` arm — the one directive `decidable` follows that #238's
   round-1 rejection was about); take both or neither.
3. **#264** → **#265** — the two open halves of complex-type derivation
   validity, unchanged as a pair and still the head of the *lane* work.
   Together they unblock **#336**, where the schema lane widens to admit
   the extension forms. #265 additionally owns **seven** of the tree's
   28 `GAP(` markers, the largest single-issue concentration there is.
4. **#301** — `ComplexType.{context}`, the component-handle identity
   **#340** consumes.
5. **#281** — `substitutionGroup=` into `{substitution group
   affiliations}`. Retires `contentrestricts.go:465` and unblocks
   **#342**.
6. **#365** — the 24 `xs:int`/`xs:integer` lexical fixtures #331 left
   outside its enumerated 48. **The only issue in the whole queue whose
   acceptance is literally a lane number**, and the cheapest movement
   available: the route already exists, no engine change is expected,
   and the selector-pinning tests #331 planted (`int001.xml`,
   `integer001.xml` as negative rows) force the widening to argue rather
   than edit a regex. It inherits **no measurement**, so its `Ratchet:`
   figure must come from a diagnostic run — the #354 hazard, named in
   its own body.
7. **#346** — the base type's `{assertions}` fold (§3.4.2.1 cl.1),
   which makes `derivation-ok-restriction` clause 5 chargeable instead
   of a guaranteed false reject.
8. **#324** — `xsdBool` reads `fixed=` too narrowly; an
   out-of-lexical-space value silently becomes `false` instead of
   charging `cvc-datatype-valid`. False-accept debt.

Eight entries, eight issues (nine counting #377 riding with #276) —
**the 8-10 band, met for the working queue for the first time since
2026-07-21**, though only because #347's question is still unanswered
and this list is doing the band's work by hand.

***Retired from the previous band:*** #368 landed and vacated its slot
(recorded above by its own post-land pass; nothing to add). #344, #331
and #337 were retired there too. Everything else carried forward.

***Retired from THIS band, 2026-08-01 (post-land pass):***

- *#277 (`validity="indeterminate"` scored as its own outcome, declined)
  landed as `2040c54` (PR #381) — **`Ratchet: schema 4247/15432 →
  4245/15432`, an authorized downward correction of 2 wrong-reason
  passes**; datatypes / instance / xpath / json / ber unchanged. Arbiter
  ACCEPT on round 1, reusing #238's authorized-correction procedure
  unchanged. **This slot is vacated, not substituted: #276, with #377
  riding, is now this band's head** and the rest of the ordering (#264 →
  #265 → #301 → #281 → #365 → #346 → #324) stands as filed — no reorder
  was warranted, and the #277 → #276 adjacency this band was built on is
  exactly why. **The smaller half of the 16 was the interesting half.**
  The issue framed 16 cases as winnable by any rejection; only **2** were
  actually banked (`elemZ031`, `schZ015`), and the other 14 were already
  `fail` — this processor was also failing to reject them, so the soft
  spot was mostly **latent rather than cashed**. That is the band's
  premise confirmed with a number: landing it at 4247 cost 2, and landing
  it after the schema lane matured would have cost whatever fraction of
  the 16 had gone green by then. The reordering that put harness
  integrity at the front of this band was correct on its own terms, and
  the same argument now transfers intact to #276.*
- *#276 (the closure scan's unresolvable-`include` admit) landed as
  `649fe1d` (PR #385) — **`Ratchet: schema 4248/15432 → 4257/15432`,
  +9, zero regressions, zero vanished**; every other lane unchanged.
  Arbiter **REJECT on round 1, ACCEPT on round 2**. **The band's premise
  is now confirmed twice, and the second confirmation is the stronger
  one, because it came with a correction.** #277 showed the cost of
  delay (2 banked wrong-reason passes, 14 latent); #276 shows that
  closing a false-accept **is not the same edit as declining on its
  premise** — the round-1 symmetric decline surrendered four genuine
  `src-include` cl. 2.4 passes (`annotA014`, `anyURI_a006_1341`,
  `schB8`, `schD8`; `schD8`'s target is literally
  `must%20not%20resolve.xyzzy`, the suite's own positive test of cl. 2.4
  tolerance). **When a rule says a condition "is not an error", a
  harness that declines on that condition is asserting the rule rather
  than testing it.** The landed form declines on the **consequence** (a
  broken parse) rather than the premise (an unresolved location), which
  is why it both closed the hazard **and** harvested +9 — the nine being
  exactly the `import` cases #182's asymmetry had been declining. Worth
  carrying into the remaining harness-integrity work: a decline that
  costs the lane cases the suite ships to prove the tolerant half of a
  rule is a mis-specified decline, not a price.*
- ***#377 loses its rider status and is re-homed, not retired.*** The
  band said *"#377 rides with this one ... take both or neither"* on a
  file-adjacency rationale. #276 was taken and #377 was not — and the
  chronicler recorded the miss explicitly (*"note this issue moved
  adjacent code and still did not produce one"*). The adjacency is now
  **spent**: #276 rewrote `compose` and `importDirective` around the new
  `closureScan.unresolved` field, so a later session picking up #377
  gets no cheaper by pairing. Judged on its own merits it is
  test-only, `Ratchet: unchanged`, and moves no lane, so it **does not
  inherit #276's head slot**; it moves to the band's tail. It stays in
  the band rather than dropping to the backlog because the `override`
  arm has now been missed **three** times in this file (#238's round-1
  prose, #377's filing, and this landing).

***Band 1 after this pass*** — head returns to the lane work, eight
entries, the band still met: **#264 → #265 → #301 → #281 → #365 → #346
→ #324 → #377**. No reorder among the carried-forward six; the two
harness-integrity items that led the band have both landed and the
reason they led it (pay for a wrong-reason pass while the lane is young)
is discharged for this cycle.

***A new tracked invariant, recorded here because `/backlog` is what
surveys it.*** #277's landing makes the per-lane count of
`validity="indeterminate"` cases a **baseline to diff against, not a
one-time measurement**: **schema 12, instance 4, datatypes 0.** A
`testdata/xsdtests` submodule bump that moves any of the three changes
the set of cases the harness declines, and per #277's own acceptance
criterion 5 that must be *noticed* rather than absorbed. Whoever bumps
the submodule re-measures these and says so.

***Post-land harvest: one issue filed, one item deliberately not.***
**#382** (`ready`, `kind/process` + `kind/tooling`) takes the
`STYLE T7` / `STYLE D6` citation drift the #277 arbiter surfaced and
correctly refused to fix inside a conformance-scoring diff. The census
in the issue is **23 Go sites across 15 files** — larger than the 10-11
the verdict reported — citing `T7` (×19), `T8` (×3, one malformed as
`T5/8`), `D6` and `L6`, none of which `docs/STYLE.md` defines (it runs
S1-S3, E1-E3, D1-D5, **T1-T6**, P1-P4, L1). Two things make it worth a
session rather than a shrug. It is the **second** surfacing: #215's
landing saw it in July and *dismissed* it (`docs/LOG/2026-07.md:9632`,
"Dismissed as a governance/numbering issue"), which was reasonable
against one sighting and is not against 23. And **#299 would have
cemented it** — #299's `## Acceptance` offers `STYLE T2/T7` as the
exemplar of a *correct* citation, in the guidance it wants written into
`docs/STYLE.md` itself. #382 is therefore sequenced **before** #299;
coordination only, the #317/#354 shape, and **neither was relabelled**
(a note is on the #299 thread instead, since relabelling a startable
`ready` issue to `blocked` for a sequencing preference is the queue
inflation #347 is about). This is also the third distinct sweep queued
over the same comment lines in `xsd/` and `builtin/strict/` — #299,
#329 (rewrapping) and now #382 — and whichever lands first churns the
line numbers the other two recorded.

*Net effect on the census below: none.* One issue closed (#277), one
filed (#382), so the tally is **exactly** where the 2026-08-01 backlog
left it — 81 open, 69 `ready`, 11 `blocked`, 1 deliberately unlabelled
(#291). A landing that files one follow-up holds the queue flat; the
overrun that paragraph documents comes from passes that file several at
once. One datapoint for #347, recorded rather than argued from.

***Post-land harvest for #276: ZERO issues filed, three dispositions
recorded on threads, and the reasoning is here so a later pass can
overturn it cheaply.*** Nothing was left implied — every item the
chronicler's entry raised is discharged somewhere readable.

- *The `closureScan.unresolved` residual* (the entry's own
  `[to file — post-land pass]` marker) was **deliberately not filed**,
  and folded into **#327** instead. The flag is scan-scoped
  (`conformance/schema_closure.go:104`, set at `:244`, `:309`, `:319`)
  and pairs with any parse failure at `conformance/schema.go:443`, so a
  case whose unresolved directive is unrelated to what broke the parse
  declines too. Three things had to be true to file and none is: the
  harm is **conservative** (an honest decline, never a fabricated
  verdict), it was a **deliberate** non-tightening rather than an
  oversight, and a body could not fill `## Acceptance` today — no
  fixture exhibits it and no lane movement can be promised, which is
  the decayed-ratchet-promise shape **#315 class 5** catalogues. #327's
  *preferred* fix (surface the set "declined **and** currently recorded
  `fail`") is the detector that would make the trigger observable,
  which is the only thing that can convert this from a conditional into
  a fileable issue. **The trigger that WOULD change this answer**: a
  suite case measurably declining through the conjunction for an
  unrelated reason. File then, with the fixture in hand.
- *#257 was verified rather than re-scoped* — the traversal-confinement
  false-accept lives on the `perr == nil` branch the new conjunction
  never reaches, so it survives #276 untouched and still `ready` as
  filed. Three sharpenings posted to its thread: this landing shipped
  **no** `loader` change (option (a)'s cost is unchanged and there is
  nothing to coordinate with, retiring #276's `## Surface` caveat); the
  fix must be an **outright decline**, not a record on `unresolved`;
  and it now has **three** sites to split, because `compose`'s include
  arm widened from `errors.Is(err, loader.ErrNotFound)` to an
  undiscriminated `if err != nil`.
- *#272 gained an obligation and is the one item with real risk
  attached.* #276's landing-order forecast resolved in the direction
  where **this decision must survive the refactor**: #272 proposes
  having `parser.Parse` report *the set of documents it assembled*, and
  that set **cannot** reconstruct the gate — a directive whose target
  went missing contributes nothing to the assembled set and is
  invisible in the report by construction. Deleting the walk without
  also reporting **unresolved directives** silently deletes the +9 and
  reopens the false accept, while satisfying #272's stated
  `Ratchet: unchanged` criterion on its face. Recorded on the thread;
  worth a body edit whenever #272 is picked up.

*Census unchanged again: one issue closed (#276), none filed — **80
open, 68 `ready`, 11 `blocked`, 1 unlabelled**.* Two consecutive
post-land passes have now held the queue flat or shrunk it, which is
the first movement against the six-pass `ready` overrun **#347**
documents, and it came from harvest **discipline** rather than from any
mechanism — three items that would each have been filed by an earlier
pass were dispositioned onto existing trackers instead. That is a
second datapoint for #347 and a better one than the first: it suggests
the overrun is at least partly a filing-reflex problem, not only a
consumption-rate problem.

**#286, #287 and #264 remain the three open sub-slices of #79**, so
closing all three closes the M4 epic and makes the M5 carve the next
planning action — the same standing note as 2026-07-31, unchanged. They
sit alongside band 1, not inside it, because each moves less lane than
the items above. **#379 (filed this pass) joins #287 as the second
`<xs:override>`-identity slice** and should be sequenced with it.

**GAP debt: 28 marker sites now, up from 23, and after this pass all 28
are owned.** The growth is #236 (+3) and #344 (+2) disclosing their own
fail-open as they landed, which is STYLE 9 working exactly as intended.
The sweep found **one** site with no owning issue and filed it. The full
map, so the next sweep starts from a ledger instead of a `grep`:

| owner | sites |
|---|---|
| **#265** | `parser/produce_complex.go:1226`; `xsd/complexderivation.go:132`, `:348`, `:376`; `xsd/contentrestricts.go:220`, `:359`; `xsd/defaultbinding.go:126` |
| **#372** | `value/valuespace.go:133`; `xsd/defaultbinding.go:362`, `:562` |
| **#279** | `parser/doc.go:103`, `:109` |
| **#287** | `parser/doc.go:130`; `parser/override.go:129` |
| **#361** | `parser/xmltree/doc.go:32`; `parser/xmltree/encoding.go:60` |
| **#249** | `xsd/substitutiongroup.go:162`, `:168` |
| **#345** | `xsd/contentrestricts.go:518`; `xsd/defaultbinding.go:268` |
| **#379** *(new)* | `parser/doc.go:118` |
| ~~**#334**~~ | ~~`parser/produce_complex.go:446`~~ — **RETIRED 2026-08-02 by `594da84`** (marker deleted, not reworded; see the post-land note at the end of M4) |
| **#342** | `parser/produce_complex.go:1170` |
| ~~**#346**~~ | ~~`xsd/complexderivation.go:189`~~ — **RETIRED 2026-08-02 by `a8c9381`** (marker deleted; the clause-5 comparison is now charged, see the post-land note at the end of M4) |
| **#282** | `xsd/contentrestricts.go:289` |
| **#281** | `xsd/contentrestricts.go:465` |
| **#267** | `xsd/defaultbinding.go:89` |
| **#248** | `xsd/wildcard.go:111` |

(`xpath/doc.go:29` is the convention's own template; three `_test.go`
hits pin existing markers; ~~four~~ **three** in-prose back-references —
~~`parser/produce_complex.go:491`~~ *(deleted with the #334 marker,
`594da84`)*, `value/valuespace.go:156`,
`xsd/defaultbinding.go:368` and `:568` — plus
`conformance/schema.go:266` cite markers rather than declaring them.
None are sites.)

**Every `file:line` in the table above is a 2026-08-01 snapshot and has
since drifted** (#236/#264/#265/#281/#334 all moved lines in
`parser/produce_complex.go`, `xsd/contentrestricts.go`,
`xsd/defaultbinding.go` and `xsd/complexextension.go`). The *owner →
site* mapping is still the useful part; the next `/backlog` should
re-derive the line numbers from `grep -rn "GAP(" --include=*.go` rather
than trusting these. Only #334's row was corrected here, because that
one is not drift — the site is **gone**.

**#379** is the sibling #287 explicitly refused to conflate itself with:
two DISTINCT `<xs:override>` elements that transform a document
identically get different `docKey`s, because `buildOverrideSet`
(`parser/override.go:154-170`) writes `e.elem.Loc()` into the identity
string. The target is then read twice, mints duplicate components, and
the assembly dies under `sch-props-correct` clause 2 — where §4.2.5's
note says *"multiple equivalent overrides of the same schema document
will not constitute a violation"*. Over-rejection only, so nothing in a
lane file is a lie. **#287 and #379 must not be settled in contradictory
directions** — both make `overrideSet.id` a function of the
transformation rather than the syntax — and a cross-reference comment
now says so on #287.

**Three issues carried stale `file:line` citations and were corrected by
comment rather than by rewriting bodies** (the #315 class, caught by the
sweep): #279 (`parser/doc.go:81`/`:87` → `:103`/`:109`, shifted by
#183's override bullets), #267 (`defaultbinding.go:81` → `:89`), and
#345 (`:266`/`:353`/`:515` → `:268`/`:362`/`:518`, shifted by #236's
`ValueSpace` seam). **#345 also shrank from three sites to two and does
not know it**: #236 replaced the lexical `{value constraint}` comparison
at what is now `defaultbinding.go:362` with the `ValueSpace` seam, and
the residue there — QName and NOTATION — belongs to **#372**, not to
#345. Recorded on the thread for whoever picks it up. #267 additionally
has a marker citing the *wrong* owner (`(#265)` where #267 is correct);
whichever lands first repoints it.

**Branch namespace: three stale `wip/*` refs now, all report-only, none
deletable by a session** (docs/WORKFLOW.md). `git ls-remote --heads
origin 'refs/heads/wip/*' 'refs/heads/parked/*'` returns
`wip/issue-195`, `wip/issue-226`, `wip/issue-368`; still no `parked/*`.

- **`wip/issue-368` @ `79c4700` — NEW, and a different species from the
  other two.** #368 **landed** (PR #378, squashed as `3e97b77`), so
  GitHub should have auto-deleted this ref; it survived because the
  post-land `docs/PLAN.md` commit was pushed to the branch *after* the
  merge. `git diff origin/wip/issue-368 HEAD` is **empty** — the tree is
  byte-identical to `main`, so nothing is owed and nothing is at risk.
  Flagged for human deletion. Worth naming the mechanism: **a post-land
  commit pushed to a merged branch resurrects the ref**, and a future
  `/backlog` that sees a `wip/issue-<N>` for a closed issue should diff
  it before assuming work was lost.
- **`wip/issue-195` @ `63c2e69`** — unchanged from 2026-07-31. Issue
  closed `needs-replan`; content deliberately not on `main`; replan
  tracked by **#304**, `ready`. Leftover, flagged for human deletion.
- **`wip/issue-226` @ `80ef0c3`** — **now fully discharged.** The
  2026-07-31 pass filed #344 as its replacement; #344 **landed**
  2026-08-01 (`2c70354`, PR #362) with `schema +5`, and its unbanked
  `+4` prediction is banked and superseded. The branch is retired, was
  never resumed, and is a leftover flagged for human deletion. **The
  #195 → #304 and #226 → #344 dispositions are now both proven**: a
  park that gets a replacement issue lands; a park without one is
  invisible.

***Re-verified 2026-08-01 by #277's post-land pass — unchanged, and the
count is now STABLE across two independent passes rather than growing.***
`git ls-remote --heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'`
still returns exactly `wip/issue-195` @ `63c2e69`, `wip/issue-226` @
`80ef0c3`, `wip/issue-368` @ `79c4700`, and still no `parked/*`. All
three map to **closed** issues, so none is resumable work — which is why
#277's session correctly started a new issue rather than adopting one.

**Deliberately NOT filed as an issue, and the reasoning is recorded so a
later pass can overturn it cheaply rather than re-derive it.** Three
things have to be true together to justify filing, and none is:
**nothing is owed** (each disposition is discharged — #195 → #304
`ready`, #226 → #344 landed, #368's ref is byte-identical to `main`),
**nothing is blocked** by their existence (they are cosmetic refs, and a
`wip/issue-<N>` for a closed issue is exactly what the reconcile step is
built to see through), and **no session can close such an issue** —
`docs/WORKFLOW.md` makes branch deletion human-only, so the issue would
sit in the queue permanently, the same failure mode this document already
named when it declined to file the "no fourth raw-`*Element` index"
standing check. Filing one would add to the `ready` overrun #347 is about
while discharging nothing.

**The trigger that WOULD change this answer**, stated now so the judgement
is not re-litigated from zero every pass: a **fourth** ref accumulating,
**or** these three surviving another two passes. Either way the response
is **one** human-owned housekeeping issue covering all of them, never one
per ref. Until then the flag lives here and in each pass's report, which
is where a human reviewing the plan actually reads it.

*Re-surveyed at the #276 post-land pass (2026-08-01):* `git ls-remote`
still shows **exactly these three** — `wip/issue-195`, `wip/issue-226`,
`wip/issue-368` — so **no fourth has accumulated**, and the counter on
"surviving another two passes" stands at **one**. Positive evidence
alongside it: `wip/issue-276` is **gone**, deleted by GitHub at the
squash-merge of PR #385, which is the branch scheme working as
`docs/WORKFLOW.md` describes. The three that remain are the exceptions,
not the norm, and each is still discharged (#195 → #304 `ready`, #226 →
#344 landed, #368 landed). Nothing filed; no `needs-replan` relabel is
warranted, since none of the three is a live claim on an open issue.

**Ready queue is 69** (81 open: 69 `ready`, 11 `blocked`, 1 deliberately
unlabelled). The sequence is **9 → 16 → 26 → 34 → 35 → 65 → 69**, and
this is the **sixth** consecutive overrun. What is new is the strongest
version of the finding yet: **eleven issues closed in one day — far
above the "several a day" the procedure assumes — and `ready` still grew
by four.** The harvest rate is not a symptom of a slow loop. All three
shrink mechanisms were exhausted again: nothing was stale enough to
close, nothing was duplicated (the two near-misses, #345/#372 and
#267/#265, are genuinely different sites and were disambiguated by
comment), and no `ready` issue was found with an open dependency. One
issue was *added* (#379) because the anti-leak convention required it.
**#347 remains undecided and this pass did not decide it** — CLAUDE.md
reserves changes to `.claude/agents/cartographer.md` for a human-filed
issue, and both of #347's options change it. Until then, **band 1 above
IS the working queue** and the other 61 `ready` issues are the backlog
behind it. New evidence for #347, posted to its thread: **a third label
state already exists in practice and is undocumented** — **#291**
carries neither `ready` nor `blocked`, deliberately (*"on the steward's
radar for the next `/retro` (Part 2), not in the develop loop's
queue"*). That is #347's own option (a), invented ad hoc by one filer,
with no label and no way for `/backlog` to tell it from a filing
mistake. Whatever #347 decides should account for #291 rather than leave
it a one-off.

**The 2026-07-25 note that "no open issue carries a GitHub milestone" is
overtaken and was wrong in the other direction too.** Exactly one
milestone is in use — **`M4 — Schema parsing`** (33 closed) — and at
survey time it was carried by **21 of the 80** open issues, assigned by
nothing but filing date. That is worse than none: partial coverage reads
as a claim. The
**working queue was made milestone-consistent this pass** (#265, #276,
#277, #281, #301, #346 and the new #379 assigned M4, plus the #79 epic
itself, which had none — 29 of 81 now carry it). The backlog behind it
was **deliberately left
alone and not filed as an issue**: what a milestone should mean for a
backlog issue versus a working-queue issue is #347's question, not a
separate one, and M5-M12 have no GitHub milestones to assign to because
those milestones are not carved. Recorded here, in the ledger
`/backlog` surveys, to be settled with #347.

**Personas: cliuser correctly skipped, libuser genuinely stale and this
session could not run it.** The addendum above landed in `738fea2`, the
2026-07-31 backlog commit — so it predates **all eleven** of the
2026-08-01 landings, and re-checking it was the right call rather than a
formality.

- **cliuser: no re-run needed, and the reason is mechanical.**
  `git diff 738fea2..HEAD -- cmd/ README.md` is **empty**. Nothing
  cliuser can see has changed, so #251, #16 and the `-version`
  reservation carry their criteria unchanged and a re-run could only
  reproduce them.
- **libuser: the addendum IS stale, by four exported identifiers, none
  of which existed when it was written.** #236 added
  **`xsd.ValueSpace`** (interface), **`value.NewValueSpace(Backend)
  xsd.ValueSpace`**, and — the one that matters — **`(*xsd.
  SchemaBuilder).FinalizeWith(ValueSpace) (*Schema, error)`, a SECOND
  finalize entry point on the primary builder**, where a consumer now
  has to choose between fail-open and decided value comparisons with
  only godoc to guide them. #230 additionally **promoted
  `unionNamespaceConstraint` to exported `xsd.UnionNamespaceConstraint`**
  (`xsd/namespaceconstraint_union.go:85`). That is precisely the
  material a libuser pass exists to judge, and #252's finding (point
  lookups, no document-order enumeration) was made against a surface
  that did not have it.
- **It was not run, deliberately.** This session has no subagent-spawn
  tool, so libuser could not be run under the isolation its value
  depends on (godoc + README only, never source) — the same constraint
  the 2026-07-31 pass hit, and a persona pass faked by a session that
  has read the source is worth less than no persona pass. **Flagged for
  the next session with a broader toolset, with the specific question
  named: does `Finalize` vs `FinalizeWith` read as a coherent choice
  from godoc alone?**

Bands 2-5 from the 2026-07-25 paragraph still stand for everything not
listed above, minus the landed items.

***Post-land pass, 2026-08-02 (#264, `0b459d1`, PR #391, schema
4257/15432, datatypes 1107/1129, ratchet unchanged — `complexTypeDecidable`
still declines every `<extension>` shape, so #336 is what actually exercises
this code).*** Band 1 loses its head: **#264 → #265 → #301 → #281 → #365 →
#346 → #324 → #377** becomes seven, **#265 → #301 → #281 → #365 → #346 →
#324 → #377**, no reorder among the rest. #265 inherits the head slot on its
own merits, not by default — #264 turned it into the unblocker for both
#336 (still `blocked` on #264 **and** #265; #264 alone did not clear it) and
two of the three `GAP(xsd)` markers #264 added (clauses 1.3 and 1.7,
`xsd/complexextension.go:111-126` and `:127-134`). One new issue,
**#392** (`ready`, `kind/gap`, M4), filed at this pass for the third marker
(clause 1.5's extension/restriction-mixed chain, `:353-367` — the one GAP in
the tree that landed naming no retiring issue, flagged by the arbiter as
owed at land) — it does **not** enter band 1: fail-open only, moves no lane
while #336 is blocked, and #265 buys strictly more per issue. Unblock scan:
nothing else clears — #250 names #264 only in a stale comment, its real
`## Depends on` is #79.

***Post-land pass, 2026-08-02 (#334, `594da84`, PR #437, schema
4283/15432, datatypes 1107/1129, ratchet unchanged — proved flat by a
write-free `GOXSD_RATCHET=1` run, exactly as the issue predicted: the
`<group ref>`-to-`<all>` shape 4.2.3.1/4.2.3.2 now reach is not
exercised by any suite case that `complexTypeDecidable` admits).***

**The GAP ledger shrinks by one and the census is now 27 owned sites,
down from 28.** #334 is the first entry to leave the table by
**retirement** rather than by re-pointing: `allGroupOf` now resolves a
`<group ref>` particle's `{term}` through the prescan index to the
referenced definition's real `{compositor}`, so the fail-open the marker
disclosed no longer exists and the marker was **deleted, not
reworded** (arbiter-verified at land). Its in-prose back-reference in
`allGroupOf`'s own doc comment went with it. Both citations are struck
in the table above.

**One issue filed: #439** (`ready`, `kind/refactor`, `area/parser`, M4) —
`produceIdentityConstraint`'s doc comment
(`parser/produce_xpath.go:123-127`) claims named identity constraints are
registered *"never at a demand-driven build site"*. False since
`buildComplexType`, and #334 added the **second** route to the identical
behaviour via `buildModelGroupDefinition`; it also now contradicts
`symbols.builtGroups`' own doc (`parser/produce.go:214-221`), which says
the opposite in as many words. Prose only — spec-immaterial (§3.17.1
imposes no order), non-regressing, registration still happens exactly
once. Third of its species after #423 and #425 and foldable into either.

**Unblock scan: nothing clears.** No open `blocked` issue names #334 in
its `## Depends on` — #334 itself depended on nothing, and the two open
issues whose bodies mention it (#345, #392) cite it as neighbouring GAP
context, not as a dependency. Ready-queue depth is unchanged by this
landing.

**Architecture-drift observation handed to the steward, not filed.**
`buildModelGroupDefinition` (`parser/produce.go`, landed here) is the
**third** instance of the identical *assembly-wide prescan index +
tri-state memo* shape, after `buildSimpleType` and `buildComplexType`.
Three hand-copied instances of one structure is a generalization
question (a shared generic memo helper) and therefore belongs to the
steward's `/retro` Part 2 architecture audit, not to a `kind/refactor`
ticket filed blind by a post-land pass. Logged as a Part 2 input here
and in the #334 thread; **do not file it as a normal issue**. Folded
into the same input: `extensionParticle` now resolves the effective
content's `ModelGroupRef` unconditionally, including when the base is
not an all group and the answer cannot change the outcome — a possible
short-circuit, cheap (the memoized build `run` performs anyway) and
deliberately symmetric with §3.4.2.3.3 clause 4.2.3's own symmetric
statement; the arbiter accepted it as-is and it is **not** a defect.

**Watch item, one instance, nothing filed.** #334's issue thread carries
the oracle's grounding and the arbiter's verdict but **no mason
implementation-summary comment** — the file-by-file account lives only
in the commit message and the LOG entry. CLAUDE.md and docs/WORKFLOW.md
make the issue thread the cross-session channel, so a reader arriving at
#334 from GitHub alone gets less than a reader with the repo. One
instance is not a pattern; recorded here so a second instance has
something to be the second of.

***Post-land pass, 2026-08-02 (#365, `53bf811`, PR #448, datatypes
1107/1129 → **1131/1153 (+24)**, instance 0/26426 and schema 9155/15432
unchanged, zero regressions — the first band-1 landing in a while whose
whole diff is one regexp alternation and the prose around it, with no
engine code at all).***

**Band 1 loses another head: `#365 → #346 → #324 → #377` becomes three,
`#346 → #324 → #377`, no reorder among the rest.** That is the tail of
the eight-issue band the #264 pass recorded (`#264 → #265 → #301 → #281
→ #365 → #346 → #324 → #377`); five have now landed in order and the
band has not been re-derived since, which is the weekly `/backlog`'s job,
not a post-land pass's.

**Unblock scan: nothing clears, and #365 was a clean leaf in both
directions.** All nine open `blocked` issues were read at this pass and
none names #365 in its `## Depends on` — #438→#414, #415→#407,
#345→#250, #267→#250, #250→#79, #248→#250, #79 (the dependency target
itself), #56 (an unfiled M6 evaluator issue), #16 (which has **no**
`## Depends on` section at all — a body defect worth repairing whenever
someone touches it, not worth a ticket of its own). A full-body sweep of
every open issue for the literal `#365` returns zero hits, so nothing
depended on this landing even informally, and #365 itself depended on
nothing.

**Two issues filed.**

**#449** (`ready`, `kind/feature`, `area/conformance`) — claim the twenty
`negativeInteger` / `nonNegativeInteger` / `nonPositiveInteger` /
`positiveInteger` `001–005` lexical fixtures. This is the harvest the
#365 LOG entry left open as an explicit ledger item, and it is unusually
well-prepared: the arbiter's round-2 **exhaustiveness sweep** certifies
*"four sub-families, twenty files, and no fifth"*, and the scope was
already written into the code by #365's repair round at
`conformance/datatypes.go:464-476`. Re-verified at this pass rather than
inherited: all 20 fixtures exist, **0** appear in
`expectations/datatypes.txt`, all 20 sit in `expectations/instance.txt`
as `fail`, and each `.xsd` is byte-identical to `integer.xsd` modulo the
type name. The issue body carries the per-fixture table (keys, suite
validity, literals) and the four types' facet sections read fresh
(`nonPositiveInteger` §3.4.14.3 `maxInclusive = 0`; `negativeInteger`
§3.4.15.3 `maxInclusive = -1`; `nonNegativeInteger` §3.4.20.3
`minInclusive = 0`; `positiveInteger` §3.4.25.3 `minInclusive = 1`).
**What makes it more than "same again": it is the first cohort with a
HALF-bounded arm.** #331's 48 all carry both bounds, #365's `xs:integer`
carries neither, these four carry exactly one each — so the rule-ID
table has real work, charging `cvc-pattern-valid` §4.3.4.4 for the four
empty-after-collapse `001` fixtures against `cvc-minInclusive-valid`
§4.3.10.3 / `cvc-maxInclusive-valid` §4.3.7.3 for the six single-bound
violations. **#449 does not enter band 1** — genuine lane movement, but
small, uncontested, and the band's ordering is a `/backlog` decision.
It also explicitly **owns rewriting `conformance/datatypes.go:464-476`**,
because landing it without that would reproduce the P3a defect a third
consecutive time in the same file.

**#450** (`ready`, `kind/tooling`, `kind/process`, `area/meta`) —
`.golangci.yml`'s anchored `^tools/` and `^cmd/` exclusions do not apply
when the linter runs from a **linked git worktree**, which is exactly
what the arbiter's throwaway ratchet measurement creates. Reproduced
independently at this pass on `53bf811` with golangci-lint 2.12.2: from
a detached worktree, 2 spurious `forbidigo` hits on
`tools/fetchspecs/main.go:52` and `tools/spec2md/main.go:105` (the only
two `fmt.Printf` calls under `tools/`); from the primary checkout,
`0 issues.` **Sibling of #426, not a duplicate** — #426 is about
*invocation* (binary on neither `PATH` nor `go.mod`'s `tool` block, five
call sites disagreeing), #450 about *result stability once it runs*, and
#426's option 1 (`go tool golangci-lint run`) would not fix #450 because
it changes how the binary is found and nothing about the path the
reporter prints. Filed rather than shrugged at because the false
positive lands in the **judge's** environment: the one persona whose job
is to reject diffs is the one most exposed to findings the diff did not
cause. `cmd/` carries the identical anchor with no current trigger —
latent, fix both.

**Three dismissals, nothing filed.**

1. **The prose-accuracy repair-round pattern is retro material, not an
   issue.** #417, #253 and now #365 have each spent a full arbiter round
   on findings about what a *correct* diff said about itself. Three
   consecutive sessions is past the two-session threshold, and the
   chronicler recorded it in `docs/LOG/2026-08.md` under the #365 entry's
   *Friction* item 2 — confirmed present at this pass, which is where
   `/retro` mines. Filing it as a normal issue would put a process
   pattern in a queue that cannot decide it.
2. **The `EffectiveFacets()` triple-`pattern` observation** (three
   `pattern "[\-+]?[0-9]+"` rows on `xs:int`, declared by `integer`,
   `long` and `int`) stays dismissed with the arbiter's reasons: each
   carries a distinct `declaring` QName, which is what
   `st-restrict-facets` §3.16.6.4's overlay walk produces and what error
   reporting attributes a facet to, so collapsing them would destroy
   information rather than deduplicate it — **not** STYLE D3. Carried
   into #449's Notes so the next mason in that function does not
   re-discover it.
3. **`docs/LOG/2026-08.md:305`'s stale `integer001–012`** stays stale.
   The arbiter's ruling that an append-only session record must not be
   retro-edited is the right one; #365's "three prose sites" were
   satisfied by the two code comments plus this file.

**Branch namespace: empty.** `git ls-remote --heads origin
'refs/heads/wip/*' 'refs/heads/parked/*'` returns nothing —
`wip/issue-365` was auto-deleted at squash-merge as expected, and no
`wip/` or `parked/` ref is outstanding repo-wide. Nothing retired,
nothing for human triage. (The throwaway worktree this pass created to
reproduce #450 was removed; `git worktree list` shows the primary
checkout alone.)

**Lane-count staleness, already owned.** This landing moves the
`datatypes` lane and therefore deepens the drift **#411** tracks (the M3
milestone heading still carries pre-#331 counts, PRINCIPLES 32). Not
re-filed and not fixed here: #411 owns it, and a post-land pass
correcting a heading it did not survey would be the kind of partial
sweep #411 exists to end. Note for whoever takes it — **#411 carries no
`ready`/`blocked` label at all**, so it is invisible to the ready-queue
count; label it at the next `/backlog`.

***Post-land pass, 2026-08-02 (#346, `a8c9381`, PR #451, ratchet
unchanged — datatypes 1153, schema 15432, instance 26426; the fold and
both prefix clauses land on code paths no admitted suite case exercises
yet, exactly as the issue predicted).***

**Band 1 is down to two: `#346 → #324 → #377` becomes `#324 → #377`.**
No reorder. That is the last of the eight-issue band the #264 pass
recorded (`#264 → #265 → #301 → #281 → #365 → #346 → #324 → #377`) that
this file has been drawing down one landing at a time; six of the eight
have now landed in filed order. **The band is spent and a full
`/backlog` re-derivation is due** — a two-entry band cannot feed a loop
that consumes several issues a day, and neither survivor moves a lane
(#324 is false-accept debt, #377 is test-only). A post-land pass does
not re-derive the band; the next weekly `/backlog` must, and it inherits
#347's unanswered question about what `ready` means while it does.

**Unblock scan: nothing clears; #346 was a clean leaf in both
directions.** All nine open `blocked` issues were read at this pass and
none names #346 in its `## Depends on` — #438→#414, #415→#407, #345→#250,
#267→#250, #250→#79, #248→#250, #79 (the dependency target itself), #56
(an unfiled M6 evaluator issue), #16 (still carrying **no**
`## Depends on` section, the body defect the #365 pass recorded — second
sighting, still not worth its own ticket). A full-body sweep of every
open issue for the literal `#346` returns zero hits. The
`xsd/complex-derivation` neighbours worth naming because a reader would
expect them here — **#414**, **#413**, **#392** — were already `ready`
before this landing and are unaffected: none of them ever waited on the
`{assertions}` fold.

**One GAP marker retired, table row struck above.**
`xsd/complexderivation.go:189` is gone, not reworded — the clause-5
comparison it disclosed as unimplemented is now `assertionsPrefix` in
the new `xsd/assertionprefix.go`, charged from both
`derivation-ok-restriction` clause 5 and `cos-ct-extends` clause 1.7.
The only surviving `GAP(` in that file is `:409` (#265's, drifted from
the snapshot's `:348`/`:376`).

**Two issues filed, one advisory dismissed** — the arbiter's three
non-blocking findings from the round-1 ACCEPT, all reproduced against
`main` at this pass rather than inherited from the thread.

**#452** (`ready`, `kind/refactor`, `area/xsd`, M4) — package `xsd` now
decides "are these two XPath Expression property records the same?" in
**two** encodings. `xsd/assertionprefix.go`'s `namespaceBindingsIdentical`
and `xsd/elementconsistent.go:378`'s `namespaceBindingsEquivalent` are the
same set comparison with the same position-is-not-significant
justification written out twice; `xpathExpressionsIdentical` compares the
same four `XPathExpression` fields that `typeAlternativesEquivalent`
(`:332`) compares inline as key-equiv-ta clauses 1-4. STYLE **T4** allows
the split only if the commit message argues it, and `a8c9381`'s does not
attempt to. The shared layer already half-exists and is already
mis-homed — `optionalStringsEqual` lives in a rule file
(`elementconsistent.go:366`) and is called from another rule file — so the
issue asks for placement next to the type being compared
(`xsd/xpathexpression.go`) rather than another rule-file resident. The
direction of risk is what makes it worth a commit: both callers read "not
identical" as **reject**, so drift between the two encodings surfaces as
a false reject, the one direction `namespaceBindingsIdentical`'s own doc
comment cites PRINCIPLES 9 to forbid. Same class as **#419** and **#323**.

**#453** (`ready`, `kind/refactor`, `area/parser`, M4) —
`produceComplexType`'s `GAP(xsd)` paragraph
(`parser/produce_complex.go:206-208`) says the new fold
*"runs HERE, on every produced type"*. It does not:
`assertionsWithBase` has two call sites (`:302`, `:398`) and
`produceImplicitContent` is neither — that arm passes `p.assertionsOf(el)`
directly and documents at `:245-252` exactly why it may (base is
unconditionally `xs:anyType`, whose `{assertions}` is empty by §3.4.7, so
the fold is provably the identity). The paragraph's **conclusion** — that
clause 1's fold needs no issue of its own, unlike the two folds #414 owns
— survives; only its premise is wrong, and the cost is that a reader who
greps `assertionsWithBase` finds three producer arms, two call sites, and
no way to tell whether the third is a bug. The orphan half-line the edit
left (`// included (#346). The`) is folded into the same comment-only
commit rather than into **#329**, whose scope is the mechanical sweep.
Filed separately from #452 on **#396**/**#445**'s settled precedent: a
comment-only correction gets its own commit, and #452 edits function
bodies.

**Dismissed: the "documentation-only test".**
`TestProduceImplicitContentAssertionsUnfolded`
(`parser/produce_xpath_test.go:653`) does not clear the decorative-test
bar #326/#261 were filed against, and the distinction is worth recording
because it will recur. Those two are about behaviour **decided by code
and pinned by nothing** — delete the carve-out, land green. This is the
opposite shape: the test makes a falsifiable claim (an implicit-content
type's `{assertions}` are its own `<assert>` children, in order) that
goes red if `produceImplicitContent` drops or reorders them. What it
cannot do is discriminate *fold ran* from *fold skipped* — because on
that path the fold **is** the identity, so there are not two behaviours
to tell apart. A test that cannot distinguish two provably identical
behaviours is not dead weight; deleting it would forfeit a real
regression guard to buy nothing. The residue is one word: the doc calls
it *"the control on the one call site that does NOT route through
`assertionsWithBase`"*, and it is not a control. That overclaim rides
along in #453 item (2). Mason flagged the limitation unprompted in the
test's own comment, which is why this is a two-word edit and not an
issue.

**The follow-up-ledger leak the chronicler flagged is closed, both
halves.** The #365 post-land pass's twenty-fixture integer harvest is
**already filed** as **#449** (`ready`, verified open at this pass) — not
outstanding, and the LOG's ledger line predates the filing. The three
#346 advisories above are now filed or dismissed in writing. No promised
follow-up from either landing is unowned.

**Branch namespace: empty.** `git ls-remote --heads origin
'refs/heads/wip/*' 'refs/heads/parked/*'` returns nothing —
`wip/issue-346` was auto-deleted at squash-merge along with four other
stale refs the fetch pruned (`wip/issue-301`, `-334`, `-340`, `-365`),
all of whose issues are closed and whose content is in `main`. Nothing
retired in place, nothing `parked/`, nothing for human triage.

**Not fixed here, already owned.** The GAP ledger's `file:line` column
above remains a 2026-08-01 snapshot with known drift; only #346's row was
touched, because that one is not drift — the site is **gone**. The full
re-derivation from `grep -rn "GAP(" --include=*.go` is the next
`/backlog`'s job, and it now has two struck rows (#334, #346) telling it
the table shrank.

***Post-land pass, 2026-08-02 (#377, `c138f67`, PR #457, ratchet
unchanged — the change is a unit arm over a synthetic `writeSchemaTree`
tree and decides no suite fixture, so there is no lane it could move;
the arbiter proved it by running `GOXSD_RATCHET=1` and finding
`git status --porcelain` empty afterward).***

**Band 1 is EMPTY.** `#324 → #377` became `#377` when #324 landed at
`9a26ac2`, and #377 has now landed too. That closes out the eight-issue
band the #264 pass recorded — `#264 → #265 → #301 → #281 → #365 → #346 →
#324 → #377` — which this file has drawn down one landing at a time
across seven post-land notes; all eight landed, in filed order. **There
is no next item. A full `/backlog` re-derivation is now the blocking
planning action, not merely "due"** — the previous note called it due
while two entries remained; there is no longer even a shallow queue to
stall on. Neither of the last two entries moved a lane (#324 was
false-accept debt, #377 test-only), so the band ended without a lane
number to show for its tail, which is itself an input to the re-derivation.

**No band is derived here, on purpose.** A post-land pass unblocks and
harvests; it does not re-derive priority. The next `/backlog` owns that,
and it inherits three things this file has been accumulating for it:
#347's still-unanswered question about what `ready` means (the `ready`
label conflates "filed" with "in the working queue", and until it is
decided the band above IS the working queue), the GAP ledger's
`file:line` column as a drifted 2026-08-01 snapshot with two struck rows
(#334, #346), and a `ready` queue now well past the 8–10 target.

**Unblock scan: nothing clears; #377 was a leaf, as its body predicted.**
All ten open `blocked` issues were read at this pass and none names #377
in its `## Depends on` — #456→#455, #438→#414, #415→#407, #345→#250,
#267→#250, #250→#79, #248→#250, #79 (the dependency target itself), #56
(an unfiled M6 evaluator issue), #16 (still carrying **no**
`## Depends on` section — third sighting of that body defect, still not
worth its own ticket). A full-body sweep of all 108 open issues for the
literal `#377` returns one hit, **#400**, which cites it in a leak-list
reconciliation rather than as a dependency and is already `ready`.

**One issue filed, one advisory routed, one dismissed** — the arbiter's
two non-blocking findings from the round-1 ACCEPT plus the session's
process finding.

**#458** (`ready`, `kind/refactor`, `area/conformance`) — three
closure-prose defects in one commit. `conformance/schema_closure.go:142`
(`decidable`'s doc) and `conformance/schema.go:379` (`execSchemaCase`'s
doc) both describe the closure as `<xs:include>` or `<xs:import>`,
**omitting `<xs:override>`** — pre-existing on `main`, not introduced by
#377, and the same prose-omission class #238's round-1 rejection was
originally about. The tell that they are defects rather than shorthand:
`schema_closure.go:13` and `:78` already name all three, so the file
contradicts itself. Folded in with them, rather than left to "the next
touch of this file" as the verdict suggested, is
`schema_closure_test.go:440`'s parenthetical citing **§F.2 clause 1** for
a **no-match** outcome when clause 1 is the *match* case
(`c-override-xslt-match`) — substance right, pointer loose. Filed
together because they are one subsystem's prose and one focused commit
(precedent: #445, #423, #425). **#458 does not enter any band** — comment
text only, no lane, no behaviour.

**Routed, not filed: the `golangci-lint` invocation.** This session ran
gate part 2 for real via `go run
github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6 run`
(0 issues, by mason and independently by the arbiter), breaking a
three-session streak — #417, #301 and #324 could not run it at all. That
is **#426**'s territory (gate part 2 is not reproducible), which was
verified at this pass to list only two candidate fixes, neither of them
this one; the incantation is recorded there as a third, because it is
zero-install **and** version-pinned, the combination neither existing
option achieves. No duplicate issue. It does not touch **#450** (the
`^tools/` anchoring half), which survives any change to how the binary is
found.

**Dismissed in writing:** `cba8dab`'s commit message cites
`conformance/schema_closure.go:148` for `decidable`'s switch; the switch
is at `:163` and `:148` is inside the doc comment. Squash-bound and now
immutable, so there is nothing an issue could fix — the correction is
recorded in the LOG entry and on the issue thread, which is the only
durable form available.

**Branch namespace: one live ref, left untouched.** `git ls-remote
--heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'` returns
`wip/issue-256` @ `c56d9f7`, tip pushed 2026-08-03 00:34Z — another
session's in-flight claim on **#256** (`ready`, open). Outside the 2h
live window and therefore resumable rather than off-limits, but nowhere
near stale; not retired, not relabelled, reported only. `wip/issue-377`
was auto-deleted at squash-merge. Nothing `parked/`, nothing for human
triage.

***Post-land pass, 2026-08-03 (#371, `cea9c2a`, PR #461, ratchet
**schema 9155 → 9166, `+11`, zero downward flips** — the arbiter
attributed all eleven case by case against the W3C suite's declared
`msMeta` validity, 7 through the new declaration-side walk and 4 through
the new use-side clause 2).***

**Unblock scan: nothing clears; #371 was a leaf.** All ten open `blocked`
issues were read and none names #371 in its `## Depends on` — #456→#455,
#438→#414, #415→#407, #345→#250, #267→#250, #250→#79's tail, #248→#250,
#79 (the dependency target itself), #56 (an unfiled M6 evaluator issue),
#16 (**still** no `## Depends on` section — fourth sighting, and the
first pass to say out loud that a body defect surviving four sightings
is no longer "not worth a ticket"; the next `/backlog` should either fix
the body or stop counting). A full-body sweep of every open issue for
the literal `#371` returns two hits, neither a dependency: **#372**
(sibling context) and **#441** (a nearest-neighbour in its own filing
search). Both already `ready`. No label changed at this pass.

**Three issues filed, one advisory routed, one class dismissed** — the
LOG entry's "Next" items 1–4, harvested while fresh.

- **#462** (`ready`, `kind/gap`, `area/value`) — the residue #371
  documented but did not fix: a construction-stage failure inside a
  list's ITEM type or a union's MEMBER type still reaches the caller as
  a **decided reject**, because gate 3 (`compile`) covers T's own
  effective facets only and the item/member compiles happen inside the
  dispatch. Filed with its blast radius pre-costed from the warden's own
  measurement (219 fixtures carry an attribute default → 14 also contain
  a `<list>`/`<union>` → 5 contain any `<pattern>`) and with the ratchet
  prediction **unchanged**: zero suite fixtures reach it, so this buys
  correctness and fail-open honesty, not a lane number. Acceptance
  requires the `GAP(value)` marker **deleted**, not reworded, when it
  closes — the #334/#346 precedent in the ledger above.
- **#463** (`ready`, `kind/gap`, `area/xsd`) — `e-props-correct` clause 2
  via `cos-valid-default` §3.3.6.2, the element-side default-value
  constraint. #371's body deliberately deferred this as *"the natural
  THIRD caller of `checkSimpleDefault` once this lands; file it then
  rather than now"*; this lands it, so it is filed. Grounded from the
  local spec at filing time (§3.3.6.1 clause 2 and §3.3.6.2's two case
  arms, verbatim with `file:line`) and scoped against what already
  exists: clause 1 **is** `checkSimpleDefault`, clause 2.2 **is** the
  existing `particleEmptiable`. The one genuinely new rejection shape is
  element-only/empty content, which neither §3.3.6.2 case arm can
  satisfy. Carries #438's anonymous-inline-type reachability warning so
  the walk shape gets measured rather than assumed.
- **#464** (`ready`, `kind/refactor`, `area/xsd` + `area/value`) — the
  five non-blocking findings the warden's diff review and the arbiter's
  verdict raised **independently and item-for-item**, as ONE ticket. The
  LOG recommended folding them into "whatever issue next touches these
  two files"; this pass owns that call and files instead. Precedent is a
  combined ticket, not a fold — #419 (#314's two), #363 (#344's three),
  #458 (#377's), and the #445/#423/#425 grouping the LOG itself cites.
  And one of the five is not prose: the hardcoded `clause :=
  "a-props-correct clause 2"` branch has a **live consumer**, since #463
  is precisely the "future third caller" it silently mislabels. #464
  sequences that item before or with #463, on both threads.

**Routed, not filed: the `GAP(xsd)` panic surface.** It is **#321**'s —
filed for exactly this cohort ("settle the contract for
`value/facets.go`'s six capability panics") and a new ticket would have
duplicated the decision. Commented there with the three facts #321 did
not have, and its body amended to point at them: (1) the panics are
**newly reachable from `xsd`'s own finalize phase**, because before #371
no `xsd` path entered the facet pipeline at all and `FinalizeWith` could
not panic on component data; (2) the cohort has a **seventh** member of
a different class, `effectiveWhiteSpace` (§3.16.7.4/§4.3.6.1), which the
six-site marker string does not match — a decision covering six of seven
leaves the contract as ambiguous as today; (3) the "enforce it in `xsd`"
option is narrower than #321's body implies, since
`checkVarietyApplicableFacets` already covers list and union and only the
**atomic** per-primitive table is outside the leaf.

**Dismissed in writing:** the `GAP(xsd)` paragraph's own defects — naming
only §4.1.5, and the half-true *"closing it means enforcing
cos-applicable-facets inside this package"* — are **not** #321's work.
They are #464 items. The split is deliberate: #321 stays a decision
ticket rather than becoming a decision-plus-typo ticket, and the
disclosure gets accurate now instead of whenever the decision is taken.

**The GAP ledger above is further out of date, by four sites.** #371 added
`GAP(value)` gates 1–3 plus the item/member residue in
`value/valuespace.go`, and the consolidated `GAP(xsd)` paragraph in
`xsd/valueconstraintvalid.go`. Gates 1–3 are permanent fail-open contract,
not incompleteness awaiting an owner; the residue is #462's and its
acceptance deletes it; the `GAP(xsd)` paragraph is #321's decision with
#464's disclosure fix. Nothing untracked — but the next `/backlog`'s
`grep -rn "GAP(" --include=*.go` re-derivation now has a fourth reason to
run beyond the two struck rows (#334, #346) and the drifted `file:line`
column.

**Branch namespace: one ref, and its status CHANGED since the last pass.**
`git ls-remote --heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'`
returns `wip/issue-256` @ `c56d9f7` (tip 2026-08-03 00:34Z, two commits,
unchanged since #377's pass) — but **#256 is now CLOSED** under
`needs-replan`, retired in place per the 2026-08-03 LOG entry, where
#377's pass saw it open and live. Verified the branch's content is **not**
in `main`, and does not need to be: the attempt was parked, not landed
(WORKFLOW — abandoned attempts are retired in place, never resumed), so
this is a correctly-retired ref, not a lost merge. **Reported for human
deletion; not acted on** (sessions never delete refs). `wip/issue-371` was
auto-deleted at squash-merge. Nothing `parked/`, nothing else for triage.
**#256's supersede-and-refile is still owed** and is a `/backlog` action,
not a post-land one.

**No band is derived here, on purpose**, and the queue arithmetic did not
improve: #371 removed one `ready` issue and this pass added three, leaving
**98 open `ready` issues** — an order of magnitude past the 8–10 target
this file's own procedure asks for. **#377's finding that band 1 is EMPTY
still stands** — a full `/backlog` re-derivation remains the blocking
planning action, and it now inherits one more input: the harvest keeps
filing follow-ups faster than the loop consumes them, so a 98-deep
`ready` label is no longer a queue at all. That is **#347**'s question
(the label conflates "filed and unblocked" with "in the working queue")
becoming load-bearing rather than theoretical — the next `/backlog`
cannot pick a band without answering it first.

**Also worth the next `/backlog`'s attention, observed but not acted on
here:** the two landings before this one (#257, `2026-08-02`; #256,
retired) left **no post-land note in this file** — #377's is the previous
one. That is **#400**'s territory (a post-land pass leaves no signal on
`main` when it files no docs commit) and is recorded here as a second
sighting rather than re-filed.

Update (2026-08-03, weekly backlog — **the full re-derivation the last
three post-land passes each named as the blocking planning action**):
band 1 drained to nothing at #377's landing and stayed empty through
three further landings. **Band 2 is derived below and is this pass's
main output.**

Lanes first, read off `conformance/testdata/expectations/*.txt` at
`origin/main` @ `ecf3d79` on 2026-08-03. Date-stamped per the new
convention in this file's preamble (#411, settled below):

| lane | pass / total | movement since 2026-08-01 |
|---|---|---|
| `schema` | **9166 / 15432** | 4247 → 9166, **+4919** |
| `datatypes` | **1131 / 1153** | 1107 / 1129 → 1131 / 1153, **+24** |
| `instance` | 0 / 26426 | unchanged |
| `xpath`, `json`, `ber` | empty by design | unchanged |

**The `schema` +4919 is not eight landings' worth of rule work and
nobody should read it as one.** The recorded per-landing deltas in this
window are small and individually attributed — #277 **−2** (authorized
downward correction), #276 **+9**, #365 **+24** on `datatypes`, #371
**+11**. The step change came from the lane *widening*, not from the
processor improving by three orders of magnitude in two days: the
decidability predicates in `conformance/schema.go` were narrowed
repeatedly across #238/#276/#340/#377/#259 so that cases previously
DECLINED now get decided, and a decided-and-correct case scores `pass`.
**That is exactly why #443 and #446 are in the band below.** A lane that
grew 2.2× by admitting cases is a lane whose admission criteria are now
load-bearing for 9166 banked passes, and `anonymousComplexTypeDecidable`'s
narrowing — the safety argument under a large share of them — is pinned
by **no test at all**. The next agent to touch that file can delete it
and land green.

***Band 2 — the working queue, dependency-ordered.*** Nine entries, in
band. Ordering doctrine, stated so the next pass can disagree with it
rather than re-derive it: **measured lane movement first, then the
integrity of the measurement itself, then false rejects, then
under-rejects, then producer completeness.** PRINCIPLES 9 is what puts
false rejects ahead of under-rejects; #238's −69 and #277's −2 are what
put measurement integrity ahead of both.

1. **#449** — claim the twenty `negativeInteger`/`nonNegativeInteger`/
   `nonPositiveInteger`/`positiveInteger` `001–005` lexical fixtures.
   **The only issue in the queue whose acceptance is literally a lane
   number, and the cheapest movement available**: the route exists
   (#331), it was widened once already (#365, a clean +24 with no engine
   change), the cohort is arbiter-verified as exactly 20 files with no
   fifth sub-family, and the body carries the per-fixture expected
   rejection rule IDs. It adds a genuinely new shape — a **half-bounded**
   arm, where #331's 48 were all doubly bounded and #365's 24 were
   unbounded. Its `Ratchet:` figure must come from a diagnostic run, not
   arithmetic: twenty claimed fixtures is not +20 (#354).
2. **#336** — narrow `complexTypeDecidable` to admit the two extension
   forms and measure the `schema` lane. **Both dependencies are closed**
   (#264 `0b459d1`, #265 `54fd100`, both 2026-08-02), verified issue by
   issue at this pass rather than inferred from the label; the stale
   `(blocked on #264/#265)` has been struck from its title. Two full
   sessions already paid their cost here — both landed
   `Ratchet: unchanged` on the explicit promise that *"the movement
   lands in THIS issue"* — so until it runs that investment is
   unmeasured. Largest expected `schema` movement in the queue.
3. **#443** — pin `anonymousComplexTypeDecidable`'s implicit-content
   narrowing. **Sequenced immediately with #336 because it is the safety
   net under it**: #336 narrows a sibling decidability predicate in the
   same file, and the existing narrowing that keeps the lane honest is
   defended by nothing executable. Either order is fine; neither is not.
4. **#446** — `testGroup/@version` is never read, so 8 XSD-1.0-scoped
   groups are scored against this 1.1 processor and `pdecimal001a` is
   unwinnable by construction. Same class as #277 and #276, promoted on
   the same argument those two proved twice: a wrong-reason pass costs
   more to unwind later than to prevent now, and the never-regress wall
   is what makes it expensive. With the lane at 9166 rather than 4247,
   "while the lane is still young" is a claim with a shrinking shelf
   life.
5. **#468** — `addAll`'s returned `last` set claims an `<all>` may end
   after ANY member, so an enclosing sequence's successor **false-rejects**
   `sequence(all(a,b), a)` on `cos-nonambig`. A live false reject of a
   valid schema — the one direction PRINCIPLES 9 forbids — and it comes
   straight out of #261's landing (`ecf3d79`), which pinned the
   carve-out and in doing so exposed that the star transcription's
   "exactly equivalent" claim holds only for a **bare** `<all>`.
6. **#436** — `final=`/`finalDefault=` and `<complexType>`'s `block=`
   are never read, so `{final}` and `{prohibited substitutions}` are
   universally empty. **Nine implemented reader sites across six rules,
   all dead on every parsed schema** — `st-props-correct` cl.3,
   `cos-st-restricts` cl.2.2.1.1 and 3.2.1.1, `cos-ct-extends` cl.1.1
   and 2.2, `derivation-ok-restriction` cl.1, `cos-ct-derived-ok`, and
   `cos-equiv-derived-ok-rec` cl.2.2/2.3's blocking union. The largest
   single concentration of inert-but-correct code in the tree, and #281
   is what made the last three sites reachable at all. Its body already
   carries the producer site table, the suite counts (108 files with
   `final=`, 30 `finalDefault=`, 27 `blockDefault=`) and the trap: **the
   keyword sets differ per property**, so copying `disallowedSubstitutions`'
   three-keyword `#all` expansion is the way to get this wrong. Direction
   is a tightening, so a downward flip is possible and must be explained
   case by case (#264 is the precedent).
7. **#464 → #463**, taken as a pair and in that order. #464 folds
   #371's five twice-raised advisories, and one of them is not prose:
   the hardcoded `clause := "a-props-correct clause 2"` branch has a
   **live consumer**, because #463 is precisely the "future third
   caller" it silently mislabels. #463 is `e-props-correct` clause 2 via
   `cos-valid-default` (§3.3.6.2), the element-side default-value
   constraint and the third caller of #371's shared `checkSimpleDefault`
   — whose first two callers were worth **+11**. #463 carries #438's
   anonymous-inline-type reachability warning, so its walk shape gets
   measured rather than assumed.
8. **#342** — `dcl.elt.common` clause 3: an element inherits its
   `{type definition}` from its `substitutionGroup` head, and today
   falls through to `xs:anyType` on **both** paths. Unblocked by #281
   (`2f9c0c4`), which put the affiliations in the slot this reads.
   **#395** (`e-props-correct` clause 4, `c-vs-sg` — a member whose type
   does not derive from its head's is accepted today) rides behind it on
   the same component data, and **#471** (filed this pass) is in the same
   producer neighbourhood; take them in that order if the queue reaches
   them.
9. **#447** — a `<simpleType>` whose body is `<list>` or `<union>` is
   unproduced at `parser/produce.go:837`, an untracked and unmarked
   producer limitation with a **measured** cost (`pdecimal019`/`020`).
   Last in band because it is the narrowest, not because it is the
   least real: it is the only band entry that is both a producer hole
   *and* has named fixtures waiting on it.

**Entries 1–3 are what a `/backlog` exists to produce and 4–9 are what
the harvest produced; the band's job is to keep the second set from
crowding out the first.** Seven of the nine were filed by post-land
passes in the last four days.

***Not in band, and why, for the three most obvious candidates.***
**#442** (a top-level `<element>` with an inline `<simpleType>` child,
the last unwidened §3.3.2.1 tier-1 shape) is a real producer slice with
a banked fixture resting on its decline, and it lost to #447 only on
"measured cost named in the body". **#472** (`goxsd8 parse`, filed this
pass) moves no conformance lane at all and is deliberately outside a
lane-ordered band — it is the first user-visible deliverable this
project would ship, which is a different axis, and a human should say
whether that axis outranks the lanes before a band encodes the answer.
**#426** (gate part 2 is not reproducible) stays out because #377's pass
already routed a working zero-install, version-pinned incantation onto
its thread; it is now a decision, not a blocker.

***Queue arithmetic, and it got worse.*** **99 open `ready`** at this
pass (110 open issues: 99 `ready`, 10 `blocked`, 1 unlabelled — #411,
settled below). The 2026-08-03 post-land note recorded 98; this pass
filed three (#470, #471, #472) and closed none, because **a cartographer
does not close issues as done**. Sixth consecutive backlog over the 8–10
band, and the band above is again doing that instruction's work by hand.

**#347 remains undecided, and this pass deliberately did not decide it.**
Nothing has changed on that thread since 2026-08-02. Its own filing note
is the reason and it still binds: the fix edits
`.claude/agents/cartographer.md`, and *"a cartographer should propose
through the queue, not apply to itself mid-sweep."* Recording it plainly
rather than re-arguing it, as its fifth restatement would earn nothing.
One new datum for whoever takes it: **~20 of the 99 `ready` issues are
comment-and-doc-accuracy items** (#290, #291, #296, #299, #313, #338,
#382, #387, #390, #396, #409, #423, #425, #428, #429, #439, #445, #453,
#458 and the prose half of #464) — a fifth of the queue whose deliverable
is English, filed one per subsystem on the settled #445/#423/#425
precedent. That is the single largest coherent cohort in the label, it
is the one place where option (a)'s `backlog`-vs-`ready` split would
change the queue's readability most, and **it is not a candidate for
merging** — the precedent for keeping them separate is explicit and was
reaffirmed at #453's filing.

***The GAP ledger is re-derived from scratch — the first full
re-derivation since the 2026-08-01 snapshot, which the last four passes
each flagged as drifted.*** `grep -rn "GAP(" --include=*.go .` at
`ecf3d79` returns **43 hits across 21 files**. Of these, 2 are
references from `_test.go` files naming a marker they pin
(`parser/override_test.go:371`, `value/valuespace_test.go:138`), 1 is a
template in `xpath/doc.go:29` showing the marker's own syntax, and 1
(`conformance/schema.go:273`) is a cross-reference to another package's
marker. **Every remaining fail-open site reconciles to an open owning
issue, with exactly one exception.**

- **The one unowned site is `parser/produce_complex.go:1235`** — an
  `<element ref="…" substitutionGroup="…">` is silently accepted, the
  attribute ignored, because the `ref=` arm returns before
  `produceLocalElement` (the one place `e-props-correct` clause 3 is
  charged on the attribute's presence) ever runs. It says so in its own
  words: *"Unowned: no issue tracks it yet."* **Filed as #471**
  (`ready`, `kind/gap`, `area/parser`). Its disclosure is unusually good
  — it already names the blast radius (no component property affected,
  no downstream rule reads a different value) and argues the direction
  correctly (under-reject, never a false accept), which is why it is
  filed at ordinary priority rather than promoted.
- **Three ownership corrections against the old snapshot.**
  `xsd/complexderivation.go:409` is **#430**'s, not #265's — the row the
  #346 pass carried forward as *"the only surviving `GAP(` in that file
  is `:409` (#265's)"* was wrong about the owner, and the site's own text
  says `#430`. `xsd/contentrestricts.go:514` has drifted to **`:522`**
  (#345's, element-side `key-dft-binding` cases 4/5).
  `parser/doc.go:111` (§5.3 Missing Sub-components never reported) is
  **#434**'s, which the site now names explicitly.
- **Two sites are permanent contract, not incompleteness awaiting an
  owner, and should stop being counted as debt.**
  `xsd/contentrestricts.go:359` argues at length that its missing bullet
  *"is NOT an omission in cos-aw-union and is not a future issue's to
  fix"* — `sibling` is defined only for element wildcards and
  `##definedSibling` is not grammatically available on `<anyAttribute>`.
  `value/valuespace.go:87-107`'s gates 1–3 are likewise the fail-open
  **contract** #371 established, not a gap; only the item/member residue
  below them (`:114`) is owned, by **#462**.
- Everything else maps cleanly: `parser/override.go:130` → #287;
  `parser/parse.go:262` + `parser/doc.go:97` → #286 (bare-marker P3 debt
  → #429); `parser/doc.go:105` → #279; `:129` → #379; `:141` → #287;
  `parser/xmltree/{encoding.go:60,doc.go:32}` → #361;
  `parser/produce_complex.go:189` → #438 (which depends on #414);
  `:1408` → #342; `:1485` → #414; `value/valuespace.go:212,235` → #372;
  `xsd/resolve.go:465` → #434; `xsd/wildcard.go:111` → #248;
  `xsd/valueconstraintvalid.go:344` + `xsd/schema.go:233` → #321
  (decision) and #464 (disclosure); `xsd/defaultbinding.go:87` → #267;
  `:224` → #345; `:318`, `:518` → #372/#462;
  `xsd/substitutiongroup.go:157` → #395;
  `xsd/contentrestricts.go:220` → #413; `:289` → #282;
  `xsd/complextype.go:705,737` → #414;
  `xsd/complexextension.go:401,420` → #392.
- **Not visible to this grep, and deliberately so:** #283 owns two
  fail-open sites that carry **no `GAP(` token at all**
  (`someBindingSubsumes`, and `unfoldCopies`' copy-cap approximation).
  A grep-derived ledger cannot find those. Whoever next re-derives this
  table should treat #283 as the standing reminder that the grep is a
  lower bound.

***Branch namespace: one ref, one long-owed action discharged.***
`git ls-remote --heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'`
returns exactly `wip/issue-256` @ `c56d9f7`, unchanged since #377's pass.
`git log --oneline main..origin/wip/issue-256` shows **two commits not in
`main`** (`3c0c918`, `c56d9f7`, branched from `9a26ac2`), and
`xsd/schema.go:95` on `main` still reads *"appends a top-level
identity-constraint definition"* — so the branch's content is genuinely
**not** landed, which is the correct outcome for a **park** rather than
a lost merge. **Reported for human deletion; not acted on** (sessions
never delete refs). Nothing under `parked/`, nothing else for triage.

**#256's supersede-and-refile — owed across four passes — is done.**
#256 was already CLOSED under `needs-replan`; the outstanding half was
the replacement, now **#470** (`ready`, `kind/bug`, `area/xsd`, M4).
**Its scope is the PAIR, and that is the whole difference from #256.**
#256 covered the writer only, and fixing one half of a consistently-wrong
pair is how a uniformly-wrong-but-coherent file became a
self-contradiction 440 lines wide on the exported surface — which is what
the second arbiter rejection was about. Verified against `main` at filing:
`AddIdentityConstraint` (`xsd/schema.go:95`) and
`Schema.IdentityConstraints` (`:527`) **both** still say "top-level", and
the second is false by execution. #470 carries forward, unchanged, the
three things both arbiter rounds independently verified — the **§3.17.2**
attribution (§3.17.1 kept only for where the property is *declared*), the
`ref=` carve-out with its `sch-props-correct` cl.2 rationale, and
`TestProduceKeyrefOnLocalElementResolvesAcrossDeclarations`
(mutation-checked twice) — and points at #256's grounding comment rather
than re-grounding. **#256's own body is not the spec of record**: it
carried the §3.17.1 misattribution that round 1 caught, and it propagated
through the oracle grounding and mason's first commit before anyone
grepped the spec.

***#411 is settled here, because it asked the cartographer to settle
it.*** The choice is **(b) date-stamp**, not (a) strip. Stripping loses
information the milestone narratives use — M3's heading is *about* the
lane draining — while a stamp makes staleness legible instead of silently
wrong, which is what PRINCIPLES 32 is protecting. The convention is
stated **once**, in this file's preamble, and nowhere else (#195's
precedent is binding: an under-settled scope propagated into four
documents is how that issue reached the park cap). Applied to the two
sites that carry live counts: M3's heading (`1043 / 31 (1074)`, now
stamped **as of 2026-07-23**) and M5's `instance`-lane sentence (now
stamped **as of 2026-08-03**, with its "unmoved by design" reason stated
so the zero is not mistaken for staleness). No other milestone heading
carries a count; the pattern the issue suspected recurs does not.
**#411 is left open** for the session that lands this file to close —
`/backlog` files and reconciles, it does not close issues as done.

***Issue reconciliation, everything else this pass touched.***

- **#16's body is rewritten to the mandatory template.** The missing
  `## Depends on` had been sighted **four** times across consecutive
  post-land passes, each time recorded as "not worth its own ticket" and
  each time re-costing the next unblock sweep a read; #371's pass handed
  this one the choice of fixing it or dropping the count. Fixed, with its
  stale *"all subcommands, `-help` included, still exit 2"* sentence
  corrected against #251's landing. It stays `blocked` and stays open as
  the durable cliuser reference for `validate` (M5) and `gen` (M9).
- **#336's title lost its stale `(blocked on #264/#265)`.** Both closed
  2026-08-02. Nothing else changed; the label had been `ready` since.
- **Blocked-issue audit — all ten are honest, no relabels.** #456→#455
  (open), #438→#414 (open), #415→#407 (open), #345→#250, #267→#250,
  #250→#79's tail, #248→#250, #79 (the dependency target itself), #56 (an
  M6 evaluator issue **not yet filed** — repoint at the concrete `#N` at
  the M6 carve), #16 (now `## Depends on` **#472**, plus two unfiled
  subcommand issues). No `ready` issue was found carrying an open hard
  dependency, so the invariant the 2026-07-25 pass established still
  holds.
- **Nothing closed as stale, obsolete or duplicate.** The 2026-07-31
  survey's finding stands re-verified at this pass: the overwhelming
  majority of the 99 describe live defects or unbuilt spec clauses, and
  the obvious consolidation candidates are exhausted.

***Step 4 (consult libuser/cliuser) did not run, and the reason is
#416 — fifth consecutive sighting.*** A cartographer subagent cannot
spawn the persona subagents its own procedure requires, so the step is
structurally unrunnable from inside a `/backlog`, exactly as #416 says.
**This pass is the first where that cost something concrete rather than
being a bookkeeping complaint**, and the difference is worth recording:
the CLI/API surface **did** move since the last real persona pass
(2026-08-01). #251 (`3b98af7`) made `-help` work in any argument
position; #252 (`4fd77dd`) published eight document-order enumeration
accessors on `*xsd.Schema`. A fresh cliuser run would see a binary that
does something for the first time, and a fresh libuser run would see the
accessor set both personas independently asked for.

**What this pass did instead, within what a cartographer can do
unaided:** it read the current `README.md` CLI section, `cmd/goxsd8/doc.go`
and `xsd/schema.go`'s exported accessor set directly, established that
#252 discharged the gate #16's thread had named since 2026-07-25
(*"#252 must land first or alongside"*), and **filed #472** — the first
non-stub subcommand — folding in cliuser's already-recorded `parse`
acceptance criteria: the 0/1/2 exit-code table, greppable
`file:line:col: [rule-id] message` failure lines, the requirement that
the summary's exact shape be pinned (*"print a summary"* is not a
contract), the multi-schema decision `parser.Parse`'s single-root
signature forces, the exit-2 overload #251 explicitly left out of scope,
and the **`-version` reservation**, unowned since 2026-07-09 and now
homed on the first subcommand issue that reaches a real flag set. That
is a fold of *recorded* persona output, **not** a persona pass, and it
does not discharge #416.

Update (2026-08-04, weekly backlog — **the pass that found out what
happened to band 2**): five issues landed in the twenty-four hours since
the band was published and **not one of them was a band entry**. All
nine entries are still open, still `ready`, untouched. That is the
central fact of this pass and everything below is organised around it.

Lanes first, read off `conformance/testdata/expectations/*.txt` at
`origin/main` @ `0dd6f71` on 2026-08-04. Date-stamped per the preamble
convention (#411, settled 2026-08-03 and verified applied at this pass —
the 2026-08-03 numbers are **not** corrected in place below, which is
the convention doing its job):

| lane | pass / total | movement since 2026-08-03 |
|---|---|---|
| `schema` | **9170 / 15432** | 9166 → 9170, **+4** |
| `datatypes` | **1131 / 1153** | unchanged |
| `instance` | 0 / 26426 | unchanged |
| `xpath`, `json`, `ber` | empty by design | unchanged |

**+4 across five landings, and that is the honest number.** #270 and
#470 were doc-and-guard work, #271 landed no code at all (a replan
record), #275 recorded `Ratchet: unchanged` substantiated by a read-only
`Compare` probe rather than a write, and #272/#273 were a report
refactor and a rule-constant convergence. Nothing in the window was a
rule implementation. Compare the 2026-08-03 entry's `+4919`, which was a
lane *widening* rather than processor improvement: this window is the
opposite shape — real engineering, no lane movement — and the two
together are the argument for why the band leads with #449 and #336.

***Band 3 — the working queue, re-derived, ten entries.*** Ordering
doctrine unchanged and restated so this pass can be disagreed with
rather than re-derived: **measured lane movement first, then the
integrity of the measurement itself, then false rejects, then
under-rejects, then producer completeness.** PRINCIPLES 9 is what puts
false rejects ahead of under-rejects; #238's −69 and #277's −2 are what
put measurement integrity ahead of both.

1. **#449** — the twenty `negativeInteger`/`nonNegativeInteger`/
   `nonPositiveInteger`/`positiveInteger` `001–005` lexical fixtures.
   Position unchanged: still the only entry whose acceptance is
   literally a lane number, still the cheapest movement available, and
   after a window that produced **+4** across five sessions that
   argument is stronger than it was, not weaker.
2. **#336** — narrow `complexTypeDecidable` to admit the two extension
   forms and measure the `schema` lane. Position unchanged; both
   dependencies (#264, #265) still closed; still the largest expected
   `schema` movement in the queue and still carrying two prior sessions'
   unmeasured `Ratchet: unchanged` investment.
3. **#443** — pin `anonymousComplexTypeDecidable`'s implicit-content
   narrowing. Sequenced with #336 as its safety net, as before. **Its
   title now understates the exposure by fifteen**: it says "9155 banked
   schema-lane passes rest on its safety argument" and the lane is at
   9170. Not amended — the number in a title is a snapshot and the
   direction only ever goes up; noted so the next reader does not think
   the lane shrank.
4. **#446** — `testGroup/@version` is never read, so 8 XSD-1.0-scoped
   groups are scored against this 1.1 processor. Position unchanged.
   Same argument as 2026-08-03 and it decays with every banked pass.
5. **#468** — `addAll`'s `last` set false-rejects
   `sequence(all(a,b), a)` on `cos-nonambig`. Position unchanged; the
   one live false reject the last pass found.
6. **#430** — **new to the band, and this is the pass's one genuine
   ordering change.** `checkRestrictionAttributeWildcard`'s
   `cos-ns-subset` comparison ignores `{attribute uses}` already
   covering the name, *"so it can false-reject a valid restriction"* —
   its own title. That is the same class as #468 and the same class the
   doctrine ranks above every under-reject below it, and it has been
   carried on the `docs/LOG` follow-up ledger as undischarged for
   several sessions while sitting unranked in the `ready` label. It was
   not promoted on 2026-08-03 because that pass was drawing the band
   from the recent-harvest set; that is a filing-order artefact, not a
   priority judgement, and correcting it is what re-deriving a band is
   for.
7. **#436** — `final=`/`finalDefault=`/`block=` never read, nine
   implemented reader sites inert across six rules. Position unchanged
   (was 6, now 7, displaced by #430 only).
8. **#464 → #463**, still a pair and still in that order. #464's
   hardcoded `clause := "a-props-correct clause 2"` branch mislabels
   exactly the third caller #463 introduces.
9. **#342** — `dcl.elt.common` clause 3, the substitution-group type
   inheritance that falls through to `xs:anyType` on both paths. #395
   and #471 still ride behind it on the same component data.
10. **#478 → #447**, in that order, and the ordering is the point.
    **#478** (defer ALL simple-type cross-references to finalize — the
    replan of #271) is a placement refactor with an explicit
    cost-of-delay argument: `<list>`/`<union>` production landing in the
    producer is the last cheap moment to fix the representation before a
    third eager guard accumulates. **#447** is precisely that
    `<list>`/`<union>` production. Taking #447 first is not wrong, but
    it is the more expensive order and nothing recorded it in a place
    the develop loop reads — it lived only in a `docs/LOG` "Next" line.
    It lives here now.

**Why the band did not get consumed, stated plainly rather than
re-litigated.** The band is prose in `docs/PLAN.md`; the develop loop
selects on the `ready` label, where all 108 entries are equal. Five
sessions each picked a reasonable `ready` issue and none of them picked
a band entry, which is the expected outcome of a priority signal that
exists in one document and a selection mechanism that reads another.
**This is #347's thesis with a second, sharper piece of evidence**, and
it is recorded on that thread, not resolved here — the fix edits
`.claude/agents/cartographer.md` and a cartographer proposes through the
queue rather than applying to itself mid-sweep. Fourth restatement; no
new argument, one new fact.

***Queue arithmetic, and the ratio is now measured.*** **120 open**
(108 `ready`, 11 `blocked`, 1 unlabelled — #411). The 2026-08-03 pass
recorded 110 (99 / 10 / 1). **Seventh consecutive backlog over the 8–10
band**, and `ready` grew **+9 in a single day**. The decomposition is
what matters: five landings closed five `ready` issues, and those same
five landings' post-land harvests filed **thirteen** (#476, #477, #478,
#479, #480, #482, #483, #484, #486, #487, #488, #489, #491), with two
more from this pass (#492, #493). **A harvest ratio of roughly 2.6 filed
per issue landed.** At that ratio no consumption rate reaches the band —
the band is not a target the develop loop can hit, which is exactly the
structural claim #347 makes and which the 2026-08-03 pass could only
assert. It is now arithmetic.

The comment-and-doc-accuracy cohort noted last pass grew with the
harvest — #476, #477, #482, #483, #488 and #492 all belong to it, so it
is **~26 of 108**, still the largest coherent cohort in the label and
still not a merge candidate (the keep-them-separate precedent was
reaffirmed at #453's filing and nothing this pass saw disturbs it).

***The #271 → #478/#479/#480 replan resolved cleanly, with one
defect.*** All three replacements were read in full at this pass. #478
and #479 are `ready` with complete template bodies; #478's is unusually
strong — eleven numbered acceptance items transcribing the warden's
pre-flight findings, two of #271's *factual errors* corrected in the
body so they cannot be re-inherited (`builtin.Seed`'s contract does
**not** change; the `xs:anyAtomicType` question is a pre-existing gap,
not a blocker), and an explicit "this may need splitting again" clause
with the defensible split line named. #480 is `blocked` with a stated,
non-issue reason (held pending a Phase 1 oracle grounding on its own
thread) — a legitimate use of the label that the blocked-issue audit
below treats as documented rather than as drift. **#271 itself is
closed, carries `needs-replan`, and its `wip/` branch is retired.**

**The defect: #271 is closed with `state_reason: completed`, and so is
#256.** Neither landed. Two data points make it a convention rather than
a slip, and the failure mode is silent — `reason:completed` is the one
signal that tells a searching session "stop reading, this is done", and
two of this repo's closed issues now point that way falsely. #470's own
history is the proof that a false premise inherited from a closed
issue's body survives an oracle grounding and a mason commit before
anyone re-greps. **Filed as #493**, scoped to the two-line
`docs/WORKFLOW.md` change plus the decision about whether correcting the
two existing parks is worth a reopen-and-reclose.

***Branch namespace: two refs, and one of them is empty.***
`git ls-remote --heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'`
returns exactly two; nothing under `parked/`.

| ref | tip | commits not in `main` | issue |
|---|---|---|---|
| `wip/issue-256` | `c56d9f7` | **2** (`3c0c918`, `c56d9f7`) | #256, closed, `needs-replan` |
| `wip/issue-271` | `c1c3824` | **0** | #271, closed, `needs-replan` |

**`wip/issue-271` carries no work at all.** Its tip is #470's landed
commit — `git merge-base --is-ancestor c1c3824 origin/main` is true — so
the branch was pushed as a claim and the attempt died at the warden
pre-flight before a single commit. Deleting it loses nothing. That is a
strictly easier case than `wip/issue-256`, whose two commits genuinely
are not in `main`, which is the correct outcome for a **park**. Both
**reported for human deletion, not acted on** (sessions never delete or
rename refs). Recorded on #399's thread with the observation that a
survey could join the two sources mechanically: a `wip/issue-<N>` whose
issue #N is closed is retired by definition, and the only question left
is the one-line ancestor check.

***The GAP ledger is unchanged, and that is a real result rather than a
skipped step.*** `grep -rn "GAP(" --include=*.go .` at `0dd6f71`
returns **43 hits across 21 files** — the same count, the same files,
and the same owner mapping as the 2026-08-03 re-derivation. **No new
unowned site appeared** across #270/#470/#272/#273/#275, which is what
one would hope for from a window with no rule implementation in it, and
worth stating because four consecutive passes before 2026-08-03 flagged
the ledger as drifted. Line numbers moved within `parser/doc.go`
(`:97/:105/:111/:129/:141` → `:112/:123/:129/:147/:159`),
`parser/parse.go` (`:262` → `:321`), `xsd/schema.go` (`:233` → `:254`)
and `conformance/schema.go` (`:273` → `:282`); the sites and their
owners are identical. The one formerly-unowned site,
`parser/produce_complex.go:1235`, still carries its *"Unowned: no issue
tracks it yet"* sentence — correctly, because **#471** owns its
retirement and #471's acceptance bar is literally
`grep -n 'Unowned: no issue' parser/produce_complex.go` returning
nothing. #283 remains the standing reminder that this grep is a lower
bound: it owns two fail-open sites carrying no `GAP(` token at all.

***Issue reconciliation, everything this pass touched.***

- **#471's milestone corrected M3 → M4.** It is a `parser/produce_complex.go`
  element-declaration producer issue charging `e-props-correct` cl.3;
  M3 (Datatypes vertical slice) is complete and hosts only its own
  datatypes-lane follow-ups. Filed into the wrong milestone on
  2026-08-03. Milestones mirror this file, so this is the file's rule
  being enforced.
- **#492 filed** — the step-4 substitute's output; see below.
- **#493 filed** — the `needs-replan`/`state_reason` defect above.
- **#489's ledger reconciled on-thread.** The carried "undischarged"
  block in `docs/LOG/2026-08.md` was matched line by line against the
  open-issue list: **~80% of it is already filed** (#324's A1/A2 → #455
  and #456; #346's two advisories → #452 and #453; #365's 20-file
  harvest → #449; #259 → #466; #261 → #468 and #469; #270's facet note
  → folded into #408; #470's nits → #476 and #477; #272's two
  cartographer items → #482 and #483; #273's three → #486/#487/#488;
  #257's Friction 1 → folded into #458; the `.agent/` pointer defect →
  #351), several lines are ordinary open issues restated (#400, #430,
  #347, #426, #336, #413, #414/#438, #458), and **one is simply wrong —
  #417 is CLOSED**, landed 2026-08-02 as `9315ce1`, carried as
  undischarged ever since. The finding recorded on that thread is that
  the ledger is append-only and nothing retires a line, so the
  deliverable is probably the *retirement rule*, not a one-time sweep.
- **#295 ↔ #433 overlap recorded, neither closed.** #295 is the defect
  (sibling M4 constructors accept a zero `{name}`); #433 is a mechanism
  (a validated QName constructor retiring the eight ad-hoc
  `Local == ""` guards) that would make it unrepresentable. Whichever
  lands second needs re-scoping; that is now on #295's thread rather
  than waiting to be discovered mid-implementation.
- **#411 re-verified and left open, unlabelled, deliberately.** Its
  deliverable is on `main` in `a136838` (preamble convention at
  `docs/PLAN.md:8-15`, M3's stamp in its milestone heading, M5's in its
  `instance`-lane sentence, both stamped in place); there is
  no work left in it, so it is not `ready`, and nothing gates it, so it
  is not `blocked`. A `/backlog` does not close issues as done. The next
  session to land a commit touching this file should close it.
- **Blocked-issue audit — all eleven honest, no relabels.** #16→#472,
  #56 (M6 evaluator, still unfiled — repoint at the concrete `#N` at the
  M6 carve), #79, #248→#250, #250→#79's tail, #267→#250, #345→#250,
  #415→#407 (open), #438→#414 (open), #456→#455 (open), and #480 (held
  pending its own Phase 1 oracle grounding, documented in its body). No
  dependency closed in this window unblocked anything: the five closures
  were #270, #470, #271, #272, #275 and none appears in any open
  `## Depends on`. No `ready` issue carries an open hard dependency, so
  the invariant the 2026-07-25 pass established still holds.
- **Nothing closed as stale, obsolete or duplicate.** The candidate
  pairs were checked rather than assumed: #426/#450 (the latter's title
  states it is the anchoring half the former does not cover), #310/#359
  (different files), #303/#491 (run-vs-bank atomicity vs. substantiating
  "unchanged" without a write), #351/#489, #429/#396/#283 (three bare-
  marker P3 issues in three different packages, kept separate on the
  precedent reaffirmed at #453's filing). The 2026-07-31 finding stands
  a third time: the consolidation candidates are exhausted.

***Step 4 (consult libuser/cliuser) did not run — #416, sixth
consecutive sighting — and this time the substitute dated the staleness
in hours.*** A cartographer subagent cannot spawn the persona subagents
its own procedure requires. What this pass did instead, within what a
cartographer can do unaided: read `README.md`'s CLI and Library
sections, `go doc ./cmd/goxsd8`, and the exported surfaces of `xsd` and
`parser` directly, as a consumer would.

**It found that the README's library contract went stale within hours of
a landing that changed it.** #272 landed `a667253` on 2026-08-04 and
published a new top-level entry point — `parser.ParseReport(location,
opts...) (*xsd.Schema, *AssemblyReport, error)` — plus `AssemblyReport`,
`AssembledDocument`, `UnfollowedDirective` and `UnfollowedReason`.
`README.md` was not touched by that commit and mentions none of it.
Worse, its Library section still carries the paragraph *"Two limits of
`parser.Parse` worth knowing up front: `<xs:redefine>` is skipped rather
than followed, so a schema that needs it assembles short…"* — a prose
warning whose programmatic remedy, `AssemblyReport.Unfollowed()`, now
sits one exported function away in the same package, unmentioned, in the
same document. **Filed as #492.**

The trend across the six sightings is worth more than the finding. On
2026-08-03 the cost was measured in *days between persona passes*
(#251 and #252 had moved the surface since 2026-08-01). One day later it
is measured in *hours since a landing*, and the only reason it was
caught is that a `/backlog` happened to run the next morning. **A weekly
persona pass would not have caught this either.** The personas answer
*"is the published contract good?"*; nothing today answers *"is the
published contract current?"*, and that is a landing-time check, not a
weekly one. Recorded on #416's thread as a second, narrower observation
for whoever takes it to scope. None of this discharges #416.

**One residue this pass did not act on, by convention.** `docs/PLAN.md`
still carries the #272 hazard note as live guidance (*"worth a body edit
whenever #272 is picked up"*) although #272 has landed. **#482** owns
superseding it without rewriting the dated historical passage, and the
add-don't-rewrite convention is why this pass records the tension here
instead of editing the earlier paragraph.

Update (2026-08-05, weekly backlog — **the pass where step 4 finally
ran**): seven issues landed since the last backlog and, for the second
window running, **not one of them was a band entry**. The band published
on 2026-08-04 is intact and untouched, which now makes **twelve
consecutive landings that ignored it**. That is the second central fact
of this pass; the first is that #416 is decided.

Lanes first, read off `conformance/testdata/expectations/*.txt` at
`origin/main` @ `6b99b2f` on 2026-08-05. Date-stamped per the preamble
convention (#411); the 2026-08-04 numbers are **not** corrected in place
above:

| lane | pass / total | movement since 2026-08-04 |
|---|---|---|
| `schema` | **9241 / 15432** | 9170 → 9241, **+71** |
| `datatypes` | **1131 / 1153** | unchanged |
| `instance` | 0 / 26426 | unchanged |
| `xpath`, `json`, `ber` | empty by design | unchanged |

**+71, and it is real processor movement rather than a lane widening.**
#286 implemented `<xs:redefine>` composition; #279 gated QName resolution
on the containing document's own imports; #294 pinned five absent-name /
empty-ref rejections end to end. Compare the two previous windows —
`+4919` (a widening) and `+4` (real engineering, no lane movement). This
window is the shape the band's ordering doctrine is written to produce.

***#416 is DECIDED — option (a) — and this pass is the evidence.*** For
seven consecutive passes `docs/PLAN.md` and #416's thread recorded that
`/backlog` step 4 (consult libuser/cliuser) is structurally unrunnable
by a cartographer, which has no tool with which to spawn a persona
subagent. **This pass ran it**: the orchestrating session ran libuser and
cliuser itself, before invoking the cartographer, and handed over their
reports. That *is* option (a), executed before it was written down.

Three files changed, and **exactly one of them states the rule** (the
#195 precedent, binding here): `docs/WORKFLOW.md`'s `/backlog` bullet now
says the launching session runs the personas and hands their stories
over. The other two only route to it — `.claude/agents/cartographer.md`
step 4 states the receiving half (fold what you are handed, **never
role-play a persona yourself**, say so when handed nothing), and
`.claude/commands/backlog.md` assigns the delegation to the session that
reads it. **That third file is why this was found at all**: it is the
trigger prompt, it said *"have the cartographer consult libuser and
cliuser"*, and it is the most load-bearing statement of the old rule —
leaving it would have re-created the defect on the next `/backlog` no
matter what the other two said. **`cartographer.md` and `backlog.md` are
agent configuration, so both diffs are flagged for human review**; they
touch neither CLAUDE.md's one rule nor the arbiter's ratchet-integrity
section, the only two texts CLAUDE.md reserves.

**What step 4 produced, now that it runs.** libuser (godoc + README only)
answered the `Finalize`/`FinalizeWith` coherence question that had been
open across two API-changing days — **coherent, not a defect**, with one
discoverability story filed as **#513** — and independently corroborated
#492, whose acceptance now carries its sharper framing (`Parse` is
documented as *"ParseReport without the report"*, so `ParseReport` is the
primitive). It **declined** to judge #407's annotation removal, because
#407 has not landed and the surface still exists: a persona refusing to
fabricate against a diff that does not exist is the isolation working,
and the criterion it leaves behind is on #407's thread. cliuser (README +
`-help` only) **disbelieved its own briefing** — which wrongly said #472
had landed — tested the binary, and re-derived the whole current CLI
contract; its criteria are on #472's thread, its re-verification of #16's
state-of-the-binary paragraph on #16's, and its one new finding is
**#514**.

**Residue, filed not carried: #512.** `docs/WORKFLOW.md`'s `/story`
bullet still says *"cartographer interviews libuser and cliuser"* — the
identical defect one trigger over, out of #416's scope because fixing it
changes what `/story` *is*. #512 carries it together with the second,
narrower observation #416's thread raised and explicitly left to be
scoped: the personas answer *"is the published contract good?"* and
**nothing answers "is it current?"**, which is a landing-time check.

***Band 3 — re-derived, ten entries, one insertion.*** Ordering doctrine
unchanged: **measured lane movement first, then the integrity of the
measurement, then false rejects, then under-rejects, then producer
completeness.**

1. **#449** — the twenty `negativeInteger`/`nonNegativeInteger`/
   `nonPositiveInteger`/`positiveInteger` `001–005` lexical fixtures.
   Head slot unchanged for a third pass: still the only entry whose
   acceptance is literally a lane number.
2. **#336** — narrow `complexTypeDecidable`, measure the `schema` lane.
   Unchanged; both dependencies still closed.
3. **#443** — pin `anonymousComplexTypeDecidable`'s implicit-content
   narrowing; still sequenced as #336's safety net. **Its title now
   understates the exposure by 86** (it says 9155 banked passes; the lane
   is at 9241). Titles are snapshots and the direction only goes up; not
   amended, noted so no reader thinks the lane shrank.
4. **#446** — `testGroup/@version` unread, 8 XSD-1.0 groups scored
   against a 1.1 processor. Unchanged, and decaying with every banked
   pass.
5. **#501 — new to the band, and the pass's one ordering change.**
   `unfoldCopies`' 2/2 copy cap **false-rejects conforming schemas** in
   `cos-content-act-restrict` — `e{3,6}` inside `e{0,100}`, and even
   inside `e{0,6}`. The doctrine ranks false rejects above everything
   below, and this one has a second claim the others lack: **it hard-
   blocks #504**, whose body says wiring `<xs:redefine>` onto that engine
   before #501 lands would propagate a live false reject into a new call
   site. It enters at 5 rather than 3 only because #336/#443 are a
   measured-movement pair.
6. **#468** — `addAll`'s `last` set false-rejects
   `sequence(all(a,b), a)` on `cos-nonambig`. Was 5.
7. **#430** — `checkRestrictionAttributeWildcard`'s `cos-ns-subset`
   comparison ignores `{attribute uses}` already covering the name. Was
   6.
8. **#436** — `final=`/`finalDefault=`/`block=` never read; nine reader
   sites inert across six rules. Was 7.
9. **#464 → #463**, still a pair, still in that order.
10. **#342** — `dcl.elt.common` clause 3; #395 and #471 ride behind it on
    the same component data.

**On deck, unchanged and recorded so the ordering is not re-derived:**
**#478 → #447** in that order (the placement refactor before the
`<list>`/`<union>` production that would otherwise mint a third eager
guard), and **#499** — adjacent to #501 in the same walk but **not** a
blocker for it, because the `maxProductStates` ceiling fails *open*: it
costs score, not correctness.

**Why the band keeps not being consumed — with the labelling explanation
now weakened.** Twelve consecutive landings have ignored it. The standing
explanation was that the band is prose here while the develop loop
selects on the `ready` label. That is still true, but this pass noticed
it does not finish the argument: the sessions that ignored the band were
choosing from `ready` **on merit**, and a `backlog`/`ready` split would
not have made a band entry look more attractive than the issue they
picked. **The selection problem may not be a labelling problem**, which
is new evidence bearing on #347's option (a) and is recorded on that
thread.

***Queue arithmetic.*** **128 open** (115 `ready`, 12 `blocked`, 1
unlabelled — still #411). The 2026-08-04 pass recorded 120 (108 / 11 /
1). Seven landings closed seven `ready`; the develop loop's post-land
harvests filed eight and this pass filed four (#512, #513, #514, #515).
**Harvest ratio ~1.7 this window, against 2.6 last** — materially lower,
and the queue still grew by seven. **Eighth consecutive overrun.**

***#347 was re-examined and deliberately held — but the reason for
holding it has expired, and the thread now says so.*** Every prior hold
rests on the claim that the fix edits an agent definition and therefore
needs a human filing. **That claim was wrong**: CLAUDE.md reserves
exactly two texts (its own one rule, and the arbiter's ratchet-integrity
section) and a cartographer procedure step is neither — which #416
demonstrated this pass by being decided and edited on the orchestrator's
direction. So the real blocker was only ever that **nobody had handed the
cartographer a decision to implement**, and the remedy is one sentence in
a `/backlog` launching prompt. The thread now carries a recommendation
with an argument — **option (b) (retire the numeric band, formalize the
ordering as the deliverable) plus the one-sentence widening of `blocked`
to "waiting on a named dependency, issue **or** trigger"** — so the next
pass can ratify rather than re-derive. Not taken unilaterally, because
#347's own body requires the shape be settled on its thread first and one
edit is not a settlement.

***Branch namespace: three refs, all retired, all reported for human
deletion.*** `git ls-remote --heads origin 'refs/heads/wip/*'
'refs/heads/parked/*'` returns exactly three; nothing under `parked/`.

| ref | tip | issue | disposition |
|---|---|---|---|
| `wip/issue-256` | `c56d9f7` | #256, closed `needs-replan` | park; two commits genuinely not in `main`, which is the correct outcome for a park |
| `wip/issue-271` | `c1c3824` | #271, closed `needs-replan` | died at the warden pre-flight; tip is #470's landed commit, so it carries **no work at all** |
| `wip/issue-287` | `6d79251` | #287, closed **completed** | the retired implementation attempt; #287 landed instead as a doc-only resolution on `wip/issue-287-doc` (PR #511, `6b99b2f`), deliberately **not** resuming this branch |

All three correspond to closed issues and are retired in place. **None
was re-verified this pass** — the orchestrating session had already done
it, and #399 exists precisely because every session re-derives that these
are dead. Sessions never delete or rename refs; **listed here for human
deletion**, as they have been for four passes.

***GAP ledger: 47 hits across 20 files*** at `6b99b2f`, against 43 / 21
at `0dd6f71`. The delta decomposes exactly: `parser/redefine.go` **+4**
(new, from #286), `xsd/contentrestricts.go` **4 → 6** (from #283
completing that file's disclosure), `parser/parse.go` **−1** and
`parser/override_test.go` **−1**. **Both removals were checked rather
than assumed**: `parse.go`'s marker was *"`<xs:redefine>` (§4.2.4) is not
followed"*, legitimately retired by #286, and `override_test.go`'s was
reworded by #287. **No marker was deleted while its gap stayed open.**

**The ledger's real finding is an acceptance criterion that cannot be
met.** #396 item 5 requires that after it lands, the tree-wide grep shows
*"no marker naming no issue"* — but #396 repoints **two** of the 47, and
**seven other markers name no issue while a filed issue owns each**:
`contentrestricts.go:77` → #501, `:319` → #499, `:597` → #345,
`complexextension.go:401`/`:420` → #392 (item 4's `:353`/`:372` line
numbers have drifted by ~48; same marker), `valuespace.go:212` → #372,
`:87` → #372, `:107`/`:114` → #462. The full mapping is on #396's thread
with a recommendation to widen the diff rather than narrow the
postcondition. **This is #510's class, found again within hours of
#510 being filed** — an Acceptance section asserting a repo-wide fact
nobody re-grepped.

**And a class STYLE P3 has no vocabulary for.** Four markers name no
issue **and should not**: `contentrestricts.go:423` and `:571`, and
`valuespace.go:95`, each argue at the site that they are permanent
licensed approximations rather than deferred work, and `xpath/doc.go:29`
is the marker *format documentation*. **#499 is the issue that must
decide this for real** (*"or rule the ceiling a permanent documented
approximation"*), and whatever it decides is the precedent for the other
three. Not settled here.

***Issue reconciliation, everything this pass touched.***

- **#512, #513, #514, #515 filed.** #512 — `/story`'s persona ownership +
  the unowned currency check. #513 — libuser's `SchemaBuilder`/
  `FinalizeWith` discoverability story. #514 — cliuser's finding that
  `run` never inspects `args[0]`, so a typo'd subcommand and a
  reserved-but-unbuilt one are byte-identical. #515 — see below.
- **#492 amended** with libuser's language; **#472, #16, #407, #396 and
  #347 amended by comment** rather than by body edit, for the reason
  #515 records.
- **#515 — a tooling hazard found the hard way, and it changes how every
  future pass edits an issue.** The GitHub MCP tool returns issue bodies
  with `<...>` tokens **stripped**; the write path replaces the whole
  body. So read-modify-write silently deletes every XML element name in a
  body — and this repo's bodies are made of `<xs:redefine>`,
  `<simpleType>`, `<element ref=…>`. This pass updated #492's body before
  noticing and had to restore six tokens by inference; #472's and #16's
  were consequently amended by comment, since #472 carries a bare
  autolink that is **unrecoverable** from the returned text. `gh` is
  403-blocked on this session type, so there is no lossless fallback
  channel. **Until #515 lands, amend bodies by comment.**
- **Nothing closed as stale, obsolete or duplicate.** Four near-miss
  pairs were checked rather than assumed and each is genuinely distinct:
  **#472/#514** (the exit-2 overload for `parse` vs. the shared stub's
  unknown-name problem, which #472 does not touch), **#398/#514** (#398
  is fenced as changing no behaviour; #514 changes behaviour),
  **#396/#513** (an arbiter advisory vs. a persona story, different
  files), **#512/#484** (who runs a step vs. when the warden pre-flight
  fires). The 2026-07-31 finding stands a fourth time: the consolidation
  candidates are exhausted.
- **Blocked-issue audit — all twelve honest, no relabels.** #16 (the
  `validate`/`gen` subcommand issues are still unfiled — this is what it
  is blocked on, and the `parse` slice was already lifted to #472),
  #56, #79, #248→#250, #250→#79's tail, #267→#250, #345→#250,
  #415→#407, #438→#414, #456→#455, #480 (held pending its own
  oracle grounding),
  #504→#501 (open, and #504's body explicitly warns a later pass not to
  read the **closed** #263 into that section and flip it to `ready`).
  **None of the seven closures in this window appears in any open
  `## Depends on`**, so nothing unblocked; no `ready` issue carries an
  open hard dependency, and the invariant holds at 115.
- **#411 unchanged, unlabelled, deliberately** — its deliverable is on
  `main` and a `/backlog` does not close issues as done. Fourth pass
  recording this.
- **#472's milestone is M3 and should be M4**, by the same rule the
  2026-08-04 pass applied to #471 (M3 is the completed datatypes slice
  and hosts only its own follow-ups; `goxsd8 parse` is a `cmd` issue on
  M4's schema-parsing work). **Not corrected**: this session's GitHub
  tooling exposes no milestone listing, so the numeric id could only be
  guessed, and a wrong milestone is worse than a stale one. Left for the
  next pass that can read the milestone list.

Update (2026-08-06, weekly backlog — **the pass where the band was
consumed for the first time, and where #347's shape is ratified**): five
issues landed since the last backlog and **one of them was a band
entry** — #449, slot 1 for three consecutive passes. That breaks a
twelve-landing streak of the band being ignored and is the first
evidence in the band's history that publishing an ordering does
anything. It is the second central fact of this pass; the first is that
**#347 is decided in shape and blocked on an actor, not on a decision.**

Lanes first, read off `conformance/testdata/expectations/*.txt` at
`origin/main` @ `779d61e` on 2026-08-06. Date-stamped per the preamble
convention (#411); the 2026-08-05 numbers are **not** corrected in place
above:

| lane | pass / total | movement since 2026-08-05 |
|---|---|---|
| `schema` | **9251 / 15432** | 9241 → 9251, **+10** |
| `datatypes` | **1151 / 1173** | 1131 / 1153 → 1151 / 1173, **+20 pass, +20 total** |
| `instance` | 0 / 26426 | unchanged |
| `xpath`, `json`, `ber` | empty by design | unchanged |

**The two lanes moved for opposite reasons and the distinction is the
point.** `schema` **+10** with the case-ID set **held at 15432** is rule
work — #295's five constructors rejecting an empty `{name}`, ten W3C
cases the non-ratchet harness is structurally silent about. `datatypes`
**+20 pass on +20 total** is a **widening**: #449 claimed twenty
`negativeInteger`/`nonNegativeInteger`/`nonPositiveInteger`/
`positiveInteger` `001–005` fixtures by extending one regexp
alternation, so the lane started *counting* twenty cases it had never
counted, and all twenty pass. Both are honest and neither is the other;
a reader comparing only the pass column would score them identically.
The 2026-08-05 window's `+71` was the first kind, the 2026-08-03
window's `+4919` the second.

***The post-land harvest is CLEAN — every promised follow-up from all
five landings is discharged, and this is the first pass where that is
true.*** #489 was filed on 2026-08-04 because the ledger had gone four
consecutive sessions unaudited; #286's entry said nine. This pass
audited each of the five landings' `Next:` lists against reality and
found **nothing leaked**:

| owed by | disposition |
|---|---|
| #295 Next 1 (`attP003` citation precision) | filed as **#518** |
| #295 Next 2 (`xsd/modelgroupdefinition.go:26` stale "no parser producer") | **folded into #338** by comment, with a named falsifier and a corrected 7→4 site list |
| #295 Next 4 (#433's guard count stale at eight) | **comment on #433** — re-grepped to **sixteen**, three-way classification added |
| #296 Next 1+2 (two §3.8.2-for-§3.7.2 comments in `parser/`) | filed as **#520** |
| #449 Next 1 (duplicated `strict.New()`/`builtin.Seed` prologue) | filed as **#522** |
| #305 Next 1 (tell #518 which went first) | **comment on #518** — witness table re-derived from the seven fixtures, Acceptance corrected, failure-capability hazard named |
| #305 Next 2 (nameless top-level `<simpleType>`) | filed as **#523** |
| #305 Next 3 (the thrice-spelled "has no usable name" sentence) | filed as **#525** |
| #305 Next 4 (`produce_complex.go:1152` comma splice) | **absorbed into #525's Acceptance** as an explicit same-commit drive-by |
| #299 Next 1 (three amendments owed to #382) | **comment on #382** |
| #299 rider (`citations_test.go:7` says 38, truth is 37) | **folded into #382**, which will be editing that file |
| #299 process residue | filed as **#527** (the `gh` 403 fallback rule) and **#528** (who notices a missing LOG entry) |

Seven filings, four fold-by-comment discharges, zero leaks. The
fold-don't-file routing #283 proved out did the work in four of the
twelve: **#338, #433, #382 and #525 each already owned the thing being
harvested**, and filing a sibling would have been the duplicate this
project keeps not having to merge. Recorded because the ledger's value
is exactly that every site maps to an issue, and this is the first
window where the mapping is total.

***#347 — RATIFIED in shape, and the blocker is now correctly named.***
The 2026-08-05 pass left a recommendation on the thread "for the next
pass to ratify rather than re-derive"; this pass's launching prompt
delegated the judgment, and the shape is **ratified**: retire the
numeric 8–10 band, formalize the dependency **ordering** as the
cartographer's deliverable in its place, and widen `blocked` to
*"waiting on a named dependency — an issue **or** a trigger — recorded
in `## Depends on`."*

Three pieces of evidence this pass added rather than re-agreed:

1. **The `blocked` widening documents existing practice.** Of the twelve
   `blocked` issues, **four are already blocked on a trigger and not on
   any open issue** — #480 (its own oracle grounding, and its body says
   `Depends on: none` in as many words), #56 (an M6 issue not yet
   filed), #16 (the `validate`/`gen` issues not yet filed), #79 (an
   uncarved epic, *"none — it is the dependency target"*). The widening
   relabels nothing; it licenses what a third of the set already does.
2. **The band was consumed** — see above. Weak evidence, and it cuts
   *toward* (b): the ordering worked, slowly, with no label enforcing
   it, which is what (b) formalizes rather than what (a) relabels
   around.
3. **The arithmetic nearly flattened and it does not rescue the number.**
   **116 `ready` of 129 open** (116 / 12 `blocked` / 1 unlabelled). The
   sequence: `9 → 16 → 26 → 34 → 35 → 65 → 69 → 77 → 81 → 99 → 108 →
   115 → 116`. Six closed in the window (the five landings plus **#416**,
   closed at the 2026-08-05 pass boundary at 14:37:43Z and therefore
   outside that pass's own count — which is what reconciles 115 + 7 − 6
   to 116), seven filed, **harvest ratio 1.4** against 1.7 and 2.6 in
   the two prior windows. A ratio converging on 1.0 holds the queue
   steady **at 116**, not at 10.

**The edit is NOT made here, and the reason is new.** The 2026-08-05
comment concluded the blocker was that no launching prompt had ever
named an option; that arrived and is discharged. What remains is
narrower and survives that refutation: **the cartographer's operating
instructions state that no agent message can authorize changing the
agent's own configuration, and `.claude/agents/cartographer.md` is that
configuration.** The orchestrating session is an agent, so an
orchestrator instruction to make this edit is exactly the class of
authorization that rule refuses — independently of the instruction being
clear and the decision being right. This is *not* the claim demolished
on 2026-08-05 (*"CLAUDE.md reserves agent-definition changes for a
human-filed issue"* — false; it reserves two texts and this is neither);
it is a claim about **which actor may direct a change to the acting
agent's own configuration**.

So #347 re-reads as **"the option is ratified; the edit requires an
actor the `/backlog` loop structurally does not contain"** — the same
diagnosis #416 carried for seven passes, with the same resolution
available: **#416's step was run by the orchestrating session, not by
the cartographer.** The three edits are specified verbatim on #347's
thread so they need no re-derivation, and they are agent configuration,
so the diff is flagged for human review as #416's was.

***Band 4 — re-derived, ten entries, one substitution and one
promotion.*** Ordering doctrine unchanged: **measured lane movement
first, then the integrity of the measurement, then false rejects, then
under-rejects, then producer completeness.**

1. **#336** — narrow `complexTypeDecidable` to admit the two extension
   forms, measure the `schema` lane. **Promoted from 2 to the head slot
   vacated by #449's landing**; both dependencies still closed, and it
   is now the only entry whose acceptance is literally a lane number.
2. **#443** — pin `anonymousComplexTypeDecidable`'s implicit-content
   narrowing; still sequenced as #336's safety net. **Its title now
   understates the exposure by 96** (it says 9155 banked passes; the
   lane is at 9251, up from the 86 recorded last pass). Titles are
   snapshots and the direction only goes up; not amended, noted again so
   no reader thinks the lane shrank.
3. **#446** — `testGroup/@version` unread, 8 XSD-1.0 groups scored
   against a 1.1 processor. Unchanged, and decaying with every banked
   pass — twenty more were banked this window.
4. **#501** — `unfoldCopies`' 2/2 copy cap false-rejects conforming
   schemas in `cos-content-act-restrict`, and **hard-blocks #504**.
   Unchanged at 4.
5. **#468** — `addAll`'s `last` set false-rejects `sequence(all(a,b),
   a)` on `cos-nonambig`. Unchanged.
6. **#430** — `checkRestrictionAttributeWildcard`'s `cos-ns-subset`
   comparison ignores `{attribute uses}` already covering the name.
   Unchanged, and **checked against this window's landings rather than
   assumed**: #295 and #296 touched absent-`{name}` guards and message
   wording in `xsd/`, which is the same package but neither the same
   file nor the same rule — #430's defect is in the wildcard-subset
   comparison and is untouched. No re-scoping owed.
7. **#436** — `final=`/`finalDefault=`/`block=` never read; nine reader
   sites inert across six rules. Unchanged.
8. **#464 → #463**, still a pair, still in that order.
9. **#342** — `dcl.elt.common` clause 3; #395 and #471 ride behind it on
   the same component data.
10. **#478 → #447 — promoted into the band from on deck**, as a pair and
    in that order. #447 is the substitution for #449 on the doctrine's
    first tier: it is a **measured** producer limitation (`pdecimal019`
    and `pdecimal020` are the recorded cost of a `<simpleType>` whose
    body is `<list>` or `<union>` going unproduced) and it is the last
    `GAP(` site in the tree that no issue owned before it was filed.
    #478 stays ahead of it for the reason on deck already recorded — the
    placement refactor lands before the production that would otherwise
    mint a third eager guard.

**On deck:** **#499** (adjacent to #501 in the same walk but not a
blocker for it — the `maxProductStates` ceiling fails *open*, so it
costs score, not correctness), **#442** (the last unwidened §3.3.2.1
tier-1 shape, with a banked fixture resting on the decline), and
**#506** (an `<override>`-substituted child losing its self-reference
resolution — a false reject, and the only tier-3 candidate not already
in the band).

- **`GAP(` ledger: 47 → 48 across 21 files, and the one new site is
  owned.** The delta is `parser/produce.go`, which had no marker at
  `bfdb920` and now carries one at `:729` — #305's disclosure that a
  top-level `<simpleType>` is the one kind still outside `topLevelName`,
  registered under `QName{target, ""}` and produced without error. Its
  prose names **#523**, filed the same session. File-by-file counts are
  otherwise byte-identical to the 2026-08-05 sweep. **Every one of the
  48 maps to an issue**; the ledger has been total since #471 claimed
  the last unowned marker.
- **Step 4 (libuser/cliuser) was SKIPPED this pass, and the reason is
  surface-unchanged — not #416's structural inability, which is
  decided.** `git diff --stat bfdb920..779d61e -- README.md cmd/` is
  empty: this window's five landings are internal to `xsd/`, `parser/`
  and `conformance/` (empty-`{name}` rejection, error-message quality, a
  shared `topLevelName` helper, fixture claims, doc-citation fixes). The
  orchestrating session — which owns the step per `docs/WORKFLOW.md`'s
  `/backlog` bullet — judged that a fresh pass would re-read an
  unchanged published surface and reproduce 2026-08-05's verdict, and
  handed the cartographer nothing. **Nothing was folded, and per step 4
  as #416 rewrote it, that is said here rather than silently omitted.**
  The next window that touches `README.md` or `cmd/` owes a real pass;
  #472 and #514 will both trigger one.
- **#472's milestone CORRECTED, M3 → M4.** The 2026-08-05 pass flagged
  this and could not act — *"this session's GitHub tooling exposes no
  milestone listing, so the numeric id could only be guessed."* It is
  readable through the issue-search path rather than the issue-list
  path: **M4 — Schema parsing is milestone number 5**. Corrected, labels
  verified intact afterwards. **No body field was sent**, which is what
  makes this safe under #515 — the stripping happens on write, so a
  metadata-only update cannot damage a body. Worth recording because
  #472's body **already carries #515 damage** from the 2026-08-05 pass
  (its `<include>`/`<import>` closure sentence and its bare autolink are
  both hollowed out), and that damage is unrecoverable from the returned
  text.
- **Nothing closed as stale, obsolete or duplicate.** Three near-miss
  pairs checked rather than assumed, all clustered on the new `parser/`
  filings and each genuinely distinct: **#520/#525** (a wrong `§3.8.2`
  citation vs. a triplicated error *sentence* — different defects in
  overlapping files), **#523/#525** (a missing guard vs. the message
  that guard would spell), **#518/#525** (a *prohibited-`ref`* sentence
  vs. an *absent-`name`* one — #525's own Notes pre-disambiguates this
  and says explicitly they must **not** be merged). The 2026-07-31
  finding stands a fifth time: the consolidation candidates are
  exhausted.
- **Blocked-issue audit — all twelve honest, no relabels**, and this
  pass read every `## Depends on` rather than carrying the prior list
  forward. #16, #56, #79 and #480 wait on triggers (see #347 above);
  #248→#250, #250→#79's tail, #267→#250, #345→#250, #415→#407,
  #438→#414, #456→#455, #504→#501 all name issues that are **still
  open**. **None of the six closures in this window appears in any open
  `## Depends on`**, so nothing unblocked; no `ready` issue carries an
  open hard dependency, and the invariant holds at 116.
- **#411 unchanged, unlabelled, deliberately** — its deliverable is on
  `main` and a `/backlog` does not close issues as done. Fifth pass
  recording this, and #347's ratified widening does not touch it: #411
  waits on a *commit*, not a dependency.
- **Branch namespace: three refs, unchanged in name and SHA since
  2026-08-04**, re-verified this pass and still awaiting human deletion
  (sessions never delete or rename refs). `wip/issue-256` @ `c56d9f7` —
  #256 closed `needs-replan`, two commits genuinely not in `main`, a
  correct park. `wip/issue-271` @ `c1c3824` — #271 closed
  `needs-replan`, tip is #470's already-landed commit, **carries no work
  at all**. `wip/issue-287` @ `6d79251` — #287 closed **completed**, the
  retired implementation attempt superseded by the doc-only #511
  landing (`6b99b2f`). No `parked/untriaged-*` refs exist. #399 owns the
  fact that this survey re-derives the same three deaths every session;
  it is `ready` and unpicked.

Update (2026-08-07, daily backlog — **the first pass under the retired
band, and the pass where #347's ratified edit exists in the tree**).
Five landings since `779d61e`: #304 (`da76d02`), #303 (`7a24098`), #306
(`86ce634`), #336 (`26a20df`), #307 (`4144016`). Two facts dominate.
The first is that **#347 is resolved in substance** — the three-part
edit it ratified on 2026-08-06 was made by the orchestrating session
before this pass launched, and the cartographer's half was to verify it
rather than to author it. The second is that **#336 moved the `schema`
lane by +278**, the largest rule-work movement since #229's cohort, and
it did so by *widening an admitted shape* rather than by fixing a rule —
which reshuffles the band's head.

Lanes, read off `conformance/testdata/expectations/*.txt` at
`4144016` on 2026-08-07. Date-stamped per the preamble convention
(#411); earlier paragraphs are **not** corrected in place:

| lane | pass / total | movement since 2026-08-06 |
|---|---|---|
| `schema` | **9529 / 15432** | 9251 → 9529, **+278** |
| `datatypes` | **1151 / 1173** | unchanged |
| `instance` | 0 / 26426 | unchanged |
| `xpath`, `json`, `ber` | empty by design | unchanged |

**The `schema` +278 is the third kind, and the file has not had to name
it before.** The 2026-08-06 paragraph distinguished *rule work* (pass
column moves, case-ID set held) from *widening* (both columns move). #336
is neither: the case-ID set held at **15432** — so by the first test it
reads as rule work — but no rule was implemented. #336 narrowed
`complexTypeDecidable` to **admit two `<extension>` forms the harness had
been declining**, and 278 of the newly-decided cases turned out to
already pass. The lane did not get better at XSD; it stopped refusing to
look. Call it **harness widening within a fixed case set** — invisible to
both prior tests, and the reason #443 and #538 now sit at the head of the
band. A reader comparing only the pass column would score it as #295's
kind, and would be wrong about what was bought.

***#347 — RESOLVED, and the resolution is #416's template applied
verbatim.*** The 2026-08-06 comment ratified option (b) and named the
blocker exactly: *"the option is ratified; the edit requires an actor the
`/backlog` loop structurally does not contain."* The orchestrating
session was that actor. Five edits, verified this pass against the
ratified shape and against the **#195 precedent that exactly one file
states a rule**:

| ratified part | file | verdict |
|---|---|---|
| retire the numeric 8–10 band | `.claude/agents/cartographer.md` step 3 | matches — *"no numeric cap … its size is an output, not a target"* |
| formalize the ordering as the deliverable | same, step 3 | matches — *"the **ordering is the deliverable**"*, `docs/PLAN.md`'s band named as the working queue |
| widen `blocked` | same, Labels line | matches the thread's draft verbatim — *"an issue **or** a trigger … recorded in `## Depends on`"* |

The handoff also asked for `docs/WORKFLOW.md` *if and only if* it
restated the number. It did, and the orchestrator found **two further
live restatements the handoff had not enumerated** — `docs/ROUTINES.md`'s
schedule-table row and `.claude/commands/backlog.md`'s front-matter
description *and* body. So the rule is now stated once in
`.claude/agents/cartographer.md` and routed to from three documents,
which is #195 satisfied rather than gestured at. **This file's dated
paragraphs were deliberately left alone** — they are a record, not a live
rule, and add-don't-rewrite governs them.

**The governance objection was not overturned; it was routed around, and
the distinction matters.** The 2026-08-06 refusal barred the cartographer
from *making* the edit on an agent's say-so. It never barred verifying
one already made. Both refusals stand as written and neither is weakened
by this resolution. **The close is handed to the orchestrating session**,
`state_reason: completed`, at the pass boundary — as #416 was closed on
2026-08-05. The five edits landed as **`9e41fc8`** mid-pass (*"process:
retire the numeric 8-10 ready-queue band, ratified at #347 (#347)"*),
which is one commit ahead of `origin/main` and **not yet merged**, so the
reason holds unchanged: closing an issue whose deliverable an unmerged
branch could still drop is **#400**'s exact open defect, and the close
belongs to the actor that can watch the merge succeed.

***Queue size, reported and not adjudicated. 117 `ready` of 130 open***
(117 / 12 `blocked` / 1 unlabelled). **That sentence is the whole of it.**
No delta against a target, no consecutive-overrun counter, no harvest
ratio pressed into service as an argument about a number that no longer
exists. The sequence `9 → … → 115 → 116 → 117` is retired here; it was
only ever evidence for #347, and #347 is answered. What replaces it as
the pass's deliverable is **Band 5**, below.

For the record, since the composition did change and the composition is
still meaningful: **five closed** (the five landings), **six filed**
(#531, #532, #534, #536, #538, #540 — every one of them by the develop
loop's own post-land harvests), plus one restored to `ready` (#291,
below), which reconciles 116 + 6 − 5 to **117**. **This pass filed
nothing** — the first `/backlog` in the recorded sequence to file zero
issues, and not for want of looking: every harvestable item this window
had already been filed by the landing that generated it, which is the
post-land harvest working exactly as #489 asked for.

***The post-land harvest is CLEAN for the second consecutive pass — and
this time the develop loop did it, not the backlog.*** Each of the five
landings' `Next:` lists was audited against reality:

| owed by | disposition |
|---|---|
| #304 Next (mason.md's fourth gate restatement) | filed as **#534** by #304's own post-land pass |
| #303 Next (arbiter verdict-template ratchet slot) | filed as **#532**; the `-v` divergence as **#531** |
| #306 Next (`Scope` → `ElementScope` rename, both reviewers deferred) | filed as **#536** |
| #336 Next 1 (`{final}` data-dead on cos-ct-extends 1.1/2.2) | filed as **#538** |
| #336 Next 2 (the rewrap nit) | **folded into #329**, which already owns the class |
| #307 Next 1 (`xsd/resolve.go:141` cites numeric `STYLE 8`) | **folded into #540**, filed the same day with a 7-site census |
| #307 Next 2 (#291's label/body inconsistency) | acted on — **and got it backwards; corrected this pass, see below** |

Eleven of twelve discharged correctly with zero leaks. **The twelfth is
the finding.**

***The one real defect this pass found is a post-land pass that reverted
an argued relabel by reading a body and not a thread — #291.*** #307's
post-land pass stripped `ready` from #291 six hours before this sweep,
quoting the body's *"Deliberately NOT labelled `ready` — this is on the
steward's radar for the next `/retro`"* and reasoning that there was *"no
evidence the steward's intent has changed since filing."*

**The evidence was two comments up the same thread.** The 2026-08-02
`/backlog` had applied `ready` deliberately and with an argument, because
the trigger #291 was waiting on had **fired**: the steward's audit ran on
2026-08-02, ruled **ruling 1** (`docs/LOG/2026-08.md:3392`,
*"the 2026-07-19 cohort-isolation ruling stands"*), and the remaining
deliverable was rescoped to one paragraph in `docs/ARCHITECTURE.md`. All
three facts were re-verified from `main` this pass rather than taken from
the comment; the §"Cohort isolation is deliberate" section exists at
`docs/ARCHITECTURE.md:470` and is pitched entirely at the cohort-triple
level, leaving the inner-struct question the issue actually asked
unanswered in either direction. **`ready` restored.**

Two things this costs, both worth writing down:

1. **It is #315's class with a fresh witness.** A body line that was true
   when written became a stale premise the moment its trigger fired, and
   a later pass acted on the body in preference to the thread. The
   project's own stated remedy — *"GitHub issues are the cross-session
   channel"* — only works if the channel is read to the end.
2. **It is the widened `blocked`'s first use in the unblocking
   direction**, and it works cleanly: #291's `## Depends on` names a
   trigger (*"sequenced by the steward's `/retro` audit"*), the trigger
   fired, so the disposition is `ready` — not `blocked`, because nothing
   is waited on, and not unlabelled, because the scheme has no such
   state. Under the pre-#347 vocabulary this issue had **no honest
   label**, which is exactly why it drifted for five passes.

A correction was also handed to whoever picks #291 up: re-deriving its
own table found the mirror set is **three anonymous inner
`Restriction struct` sites plus one already-named `d34Restriction`**, not
four identical anonymous ones, and every line number in the body has
drifted (`:1803`, `:2790`, `:3034`, `:3468`). Posted as a comment, not a
body edit, per #515.

***#411 — the ask is discharged, and the sixth recording names an actor
instead of restating it.*** Five passes wrote *"unchanged, unlabelled,
deliberately."* Verified at `4144016`: the convention **was** decided
(option (b), date-stamp), it is stated once in this file's preamble
(lines 8–16), it **was** applied to the M3 heading (*"as of
2026-07-23"*), and *"any other milestone headings with the same
pattern"* — checked one by one — **is the empty set**. M3 was the only
instance, so the issue's own hedge that the pattern *"probably recurs"*
is falsified. Nothing is left to do. **Recommended to the orchestrating
session for closure as `completed`**, on the reasoning posted to the
thread: a cartographer closes issues as obsolete or duplicate, not as
done, and forcing `not_planned` onto genuinely-finished work would
manufacture **#493**'s defect rather than avoid it.

***GAP ledger: 48 → 50 across 21 files, and both new sites are
cross-references, not new fail-open.*** File-by-file counts at `4144016`
are byte-identical to the 2026-08-06 sweep **except**
`conformance/schema.go`, **1 → 3**. Both additions came in with #336 and
both are prose *citations* of a marker that lives elsewhere —
`:58` and `:861` each name `xsd/complexextension.go`'s clause-1.5
under-rejection, owned by **#392**, and both are additionally covered by
**#538**. The pre-existing `:284` cites `xsd/contentrestricts.go`'s
open-content marker, owned by **#413**. No new fail-open site was
introduced by any of the five landings, no marker was deleted while its
gap stayed open, and **every one of the 50 maps to an issue** — the
ledger has been total since #471 claimed the last unowned marker, and
this is the third consecutive pass it has held.

***Band 5 — the deliverable, now that it is the only one.*** Ordering
doctrine unchanged: **measured lane movement first, then the integrity of
the measurement, then false rejects, then under-rejects, then producer
completeness.** One head promotion and one insertion; every other entry
is unchanged and said to be so, so the ordering is not re-derived.

1. **#443** — pin `anonymousComplexTypeDecidable`'s implicit-content
   narrowing. **Promoted from 2 to the head slot vacated by #336's
   landing, which is precisely the event it was sequenced as the safety
   net for.** Its exposure is no longer hypothetical: #336 widened the
   admitted shape and banked **+278** against a narrowing that **no test
   pins** — deleting it still lands green. **9529** banked `schema`
   passes now rest on a safety argument nothing checks, up from 9251 last
   pass. Its title says 9155 and is now understated by 374; titles are
   snapshots and the direction only goes up, so it is **not amended**,
   noted a third time so no reader concludes the lane shrank.
2. **#538 — new this window, entering at 2.** Same tier, same file, and
   the disclosure #336's own landing incurred: `conformance/schema.go`'s
   fail-open inventory for the newly-admitted `<extension>` shape names
   **one** clause when **three** cannot reject — `cos-ct-extends` 1.1 and
   2.2 are data-dead on an unmapped `{final}`. Sequenced immediately
   behind #443 because the two together are the single "make the
   measurement honest about what #336 bought" slice, and because it is
   cheapest while the landing is fresh. Note the coupling: the *cause* of
   the unmapped `{final}` is **#436** (slot 7) — #538 is the honest
   disclosure, not the fix, and must not be closed by fixing #436.
3. **#446** — `testGroup/@version` unread, so 8 XSD-1.0-scoped groups are
   scored against a 1.1 processor and `pdecimal001a` is unwinnable by
   construction. Unchanged in kind and **decaying faster than at any
   prior reading**: 278 more cases were banked this window against a
   version filter that does not exist.
4. **#501** — `unfoldCopies`' 2/2 copy cap false-rejects conforming
   schemas in `cos-content-act-restrict`, and **hard-blocks #504**. Was 4.
5. **#468** — `addAll`'s `last` set false-rejects `sequence(all(a,b), a)`
   on `cos-nonambig`. Was 5.
6. **#430** — `checkRestrictionAttributeWildcard`'s `cos-ns-subset`
   comparison ignores `{attribute uses}` already covering the name. Was
   6, and **re-checked against this window rather than assumed**: #306
   wired `AttributeDeclaration {scope}.{parent}` in the same package but
   touches neither the wildcard-subset comparison nor its rule. No
   re-scoping owed.
7. **#436** — `final=`/`finalDefault=`/`block=` never read; nine reader
   sites inert across six rules. Was 7, and its stock rose without
   moving: #538 is a second independent consequence of the same unmapped
   `{final}`, which is now costing in two places.
8. **#464 → #463**, still a pair, still in that order.
9. **#342** — `dcl.elt.common` clause 3; #395 and #471 ride behind it on
   the same component data.
10. **#478 → #447**, still a pair, still in that order — the placement
    refactor before the `<list>`/`<union>` production that would
    otherwise mint a third eager guard.

**On deck:** **#392** — *added this pass*, the substantive retirement of
the clause-1.5 under-rejection that #538 merely discloses and that two of
`conformance/schema.go`'s three GAP citations now point at. Sequenced
**behind** #538 deliberately: a correct fail-open inventory is a
precondition for measuring what #392 buys, and landing them in the other
order means the measurement changes under the fix. Also on deck,
unchanged: **#499** (adjacent to #501 in the same walk but not a blocker
for it — the `maxProductStates` ceiling fails *open*, so it costs score,
not correctness), **#442**, and **#506**.

- **Nothing closed as stale, obsolete or duplicate.** Two near-miss pairs
  checked rather than assumed, both thrown up by this window's filings
  and both genuinely distinct: **#538/#436** (a fail-open *inventory* that
  understates by two clauses vs. the unread `final=` attribute that makes
  those clauses data-dead — disclosure and cause, and #538 stays open if
  #436 lands alone with the inventory uncorrected) and **#538/#392** (an
  inventory correction vs. the clause-1.5 implementation, adjacent in the
  same file and neither a superset of the other). The 2026-07-31 finding
  stands a **sixth** time: the consolidation candidates are exhausted, and
  after six passes that is a property of the queue rather than a run of
  luck.
- **Blocked-issue audit — all twelve honest, no relabels, and for the
  first time the vocabulary fits.** #16, #56, #79 and #480 wait on
  triggers with nothing open in `## Depends on`; under #347's ratified
  widening these are **correctly labelled**, and this is the first pass
  that can say so without an apology or a footnote. #248→#250,
  #250→#79's tail, #267→#250, #345→#250, #415→#407, #438→#414,
  #456→#455, #504→#501 all name issues **verified still open** at
  `4144016`. **None of the five closures in this window (#303, #304,
  #306, #307, #336) appears in any open `## Depends on`**, so nothing
  unblocked. No `ready` issue carries an open hard dependency; the
  honesty invariant holds at 117.
- **Step 4 (libuser/cliuser) was SKIPPED, and the reason is
  surface-unchanged.** `git diff --stat 779d61e..4144016 -- README.md
  cmd/` is empty: all five landings are internal to `xsd/`, `parser/`,
  `conformance/` and `.claude/`. The orchestrating session — which owns
  the step per `docs/WORKFLOW.md`'s `/backlog` bullet and #416's
  resolution — judged that a fresh pass would reproduce 2026-08-05's
  verdict against an identical published surface, and handed the
  cartographer nothing. **Nothing was folded, and per step 4 that is
  said here rather than silently omitted.** #472 and #514 each still owe
  the next window a real pass.
- **Branch namespace: three refs, unchanged in name and SHA since
  2026-08-04**, and **not re-verified by this pass** — the orchestrating
  session had already done it, which is the fourth consecutive session to
  spend the same derivation on the same three dead branches. `wip/issue-256`
  @ `c56d9f7` (#256 closed `needs-replan`, two commits genuinely not in
  `main`, a correct park); `wip/issue-271` @ `c1c3824` (#271 closed
  `needs-replan`, tip already on `main` via #470, **carries no work at
  all**); `wip/issue-287` @ `6d79251` (#287 closed **completed**,
  superseded by #511's doc-only landing). No `parked/untriaged-*` refs.
  **Listed for human deletion**, as they have been for five passes;
  sessions never delete or rename refs. **#399 owns this and is `ready`
  and unpicked** — and it is worth noting that its cost is now measurable
  rather than notional: five passes × one full `git ls-remote` +
  ancestry check, to re-learn three facts that have not changed in four
  days.

## M5 — Instance validation (XML) — epic #250, not yet carved

`validate` engine + `validate/xmlsrc`; greedy deterministic matching, IDC,
xsi:type/nil, wildcards, default/fixed values. **`instance` lane** (0 pass
/ 26426 fail **as of 2026-08-03**; unmoved since the lane was created, by
design — nothing decides it until this milestone).

The epic was filed as **#250** in the 2026-07-25 backlog, `blocked` on
#79's tail, deliberately **uncarved**: M4 still has four sub-slices open
and the ready queue is at 34, so carving now would deepen a queue that is
already over band. **Carving M5 is the next planning action once #180 /
#181 / #182 / #183 drain**, and it should follow the M4 pattern that
worked (#79 → #167–#183): model/infoset shape slices first, then a
lane-bring-up slice that produces the first real `instance` number early
(the #175 analogue), then the validator fan-out.

Filing it was not bookkeeping for its own sake — #248 had been `blocked`
on "the M5 epic, NOT yet filed" and is now repointed at #250. The
signatures in `validate/doc.go` and `validate/xmlsrc/doc.go` under
"Planned contract (M5 — not yet implemented)" are the contract the epic
discharges, and the design constraints there (PRINCIPLES 8, 11, 13, 14,
15) are settled — the carve implements them, it does not reopen them.
Known M5-time debts to fold in rather than rediscover: #248 (the
wildcard-admission entry point that retires the last `GAP(`), #236
(`cvc-au`'s effective value constraint), #63 (`cvc-identity-constraint`'s
`{referenced key}`), and the deferred memoization measurement on
`inSubstitutionGroupOf`.

## M6 — XPath required subset

CTA restricted subset + assertion essentials; fail-open with GAP markers;
IDC selector/field paths. Dynamic-error direction per PRINCIPLES 20.

Not carved, and deliberately not filed as an epic yet (2026-07-25
backlog): M5 is the next carve and speculative epics two milestones out
earn nothing. One dangling placeholder is outstanding and is recorded
here so it is not lost — **#56** (per-assertion/CTA results must
distinguish a genuine PASS from a fail-open "unevaluated", i.e. an
`Evaluated bool`) is `blocked` on "the M6 assertion/CTA XPath evaluator
issue, NOT yet filed". Repoint it at the concrete `#N` at the M6 carve.
It is the same situation #248 was in for M5 until #250 was filed, and it
matters for the same reason: STYLE 9's fail-open discipline is only
honest if a fail-open answer is distinguishable from a real pass.

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
