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

## M3 — Datatypes vertical slice (complete — all 20 primitives mapped, `datatypes` lane 1043 pass / 31 fail (1074) after the list-variety Facets cohort #75 and the `value.effectiveWhiteSpace` union not-applicable path #98 landed 2026-07-23; the IBM precisionDecimal cohort (#162) and the `Mapping.Canonical` doc (#166) landed 2026-07-19; open datatypes-lane follow-ups: anyURI-triage #190, union member-dispatch #223, integer-family list fixtures #224)

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
| **#334** | `parser/produce_complex.go:446` |
| **#342** | `parser/produce_complex.go:1170` |
| **#346** | `xsd/complexderivation.go:189` |
| **#282** | `xsd/contentrestricts.go:289` |
| **#281** | `xsd/contentrestricts.go:465` |
| **#267** | `xsd/defaultbinding.go:89` |
| **#248** | `xsd/wildcard.go:111` |

(`xpath/doc.go:29` is the convention's own template; three `_test.go`
hits pin existing markers; four in-prose back-references —
`parser/produce_complex.go:491`, `value/valuespace.go:156`,
`xsd/defaultbinding.go:368` and `:568` — plus
`conformance/schema.go:266` cite markers rather than declaring them.
None are sites.)

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

## M5 — Instance validation (XML) — epic #250, not yet carved

`validate` engine + `validate/xmlsrc`; greedy deterministic matching, IDC,
xsi:type/nil, wildcards, default/fixed values. **`instance` lane** (0 pass
/ 26426 fail today).

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
