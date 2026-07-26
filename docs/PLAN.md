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
   unreachable), **#232** (unresolvable `notQName` dropped), **#240**
   (`ref=` form of `<unique>`/`<key>`/`<keyref>`), **#238** (multiple
   `<ts:schemaDocument>` children — wrong-document decisions), **#226**
   (UTF-16 BOM misread as invalid UTF-8), **#202** (Required name/ref
   slots accept the absent zero-QName), **#253** (silent short assembly).
3. **Remaining feature leaves.** **#63** (IDC `{referenced key}` +
   `c-props-correct` cl.2 — the last open M4 leaf follow-up, and an M5
   prerequisite per #250), **#206** ({context} / {scope}.{parent}
   containment back-pointers), **#217** (`cos-st-restricts` facet-value
   sub-clauses), **#235** → **#236** (attribute default/fixed producer
   wiring, then the §3.5.4 effective value constraint — #236 is a soft
   sibling of #235, not gated on it, but landing #235 first is cheaper).
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
