# goxsd8 Roadmap

**This file is status, not history.** It says where the project stands and
what each milestone is for. It does not narrate how it got there — that is
`docs/LOG/` — and it is not the work queue: GitHub issues are, and the
milestone links below go to the live lists.

Milestones map one-to-one to GitHub milestones. The cartographer carves
each into session-sized `ready` issues; the develop loop closes them one
per landing. Prefer vertical slices that move a conformance lane over
horizontal completeness.

**The Status section is REPLACED, never appended to.** `/backlog` rewrites
it wholesale from the sources — `go tool lanestatus` for the lane scores,
GitHub for milestone and queue counts — and stamps it with the date it
read them. One stamp for the whole section, so a reader can tell staleness
from wrongness at a glance. Never add a dated paragraph beside the old
one — appending is what this replaces.

## Status — 2026-09-25 (`/backlog`, the thirty-third)

**The ordinal skips from thirty-one to thirty-three, and there was a
thirty-second.** The 2026-09-24 pass (PR #1689, branch
`claude/dazzling-cerf-c1o1wx`) finished all of its GitHub work: it filed
#1684–#1688 and ruled on #848 and #1419. Its PR, which carried the `docs/PLAN.md`
stamp and the LOG entry, was never merged. This pass closed #1689 as superseded.
It replaces that PR's stale Status draft with this stamp, and this pass's LOG
commit carries the stranded LOG entry onto `main`. This is the third time a
`/backlog` pass has stranded its own write-up. **#1705** files the detector.

**The 2026-09-23 band landed twelve for twelve.** Every row landed, each with its
own post-land pass on `main`. Rows 1 and 2 went first, as predicted: #1657 at
`b3b775e`, then #773 at `c1851ae`. Row 7 (#1499) went third. After that, rows
3–6 and 8–12 went in band order. `main` went from `dc12a98` to `16b4057`: 25
commits, which are the last stamp's commit, twelve landings and twelve post-land
passes.

**The three claims the last stamp made, and how each resolved.**
- **Rows 1–3 (the PREDICTION claim).** #1657 landed before #773, as the claim
  required. #773's PREDICTION answered `not derived`, as #1657's rule allows. It
  did not narrow by hand, and it banked **+9**. The reserved tooling route (a
  census mode that resolves a QName-valued attribute against its bindings) is
  therefore owed, and **#1671** files it.
- **#1499 (the `kind/refactor` clause).** It landed at `0117c5e`, and the clause
  now has input. **#849** is the one open refactor with a measured cost of delay.
  Its command re-run on `16b4057` gives **4**, the same as its recorded figure,
  and the body is restamped to this commit. The other 78 open `kind/refactor`
  issues are unmeasured and are ordered by dependency alone.
- **M4 moved for the first time in seven windows**, from 54/136 to 53/138.

### Conformance lanes

**This table is `go tool lanestatus`, pasted verbatim, on `main` at `16b4057`.**
It is the committed expectations census, which `docs/WORKFLOW.md` names as the
lane score (#1120).

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11232 | 15129 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14697 | 701 | 15398 |
| `xpath` | — | — | 0 |

**`instance` gained 9, and no lane regressed.** #773 banked all nine
(`c1851ae`, String Valid clause 3: every ENTITY value names a declared unparsed
entity). The other eleven landings each banked `Ratchet: unchanged`. `schema`
and `datatypes` are byte-identical to the last stamp.

**`datatypes ⊆ instance` is intended** (#1507), so quote `instance`
11232/26361 without hedging it. **An em dash means a lane with no cases yet, not
a lane scoring zero.** `datatypes` is M3 and complete. `schema` is M4, active.
`instance` is M5, active. `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This pass took no conformance measurement.** The lane table needs no
submodule, and the +9 is read from the committed diff.

### Branch namespace, `origin` (report-only; a session never deletes a ref)

`go tool wipsurvey` was run on an unshallow checkout against this pass's walk.

- **`wip/*`: 13 refs, all RETIRED, and no LIVE claim.** They are the same
  thirteen as the last stamp (#434, #732, #822, #846, #872, #933, #968, #993,
  #1332, #1356, #1426, #1451, #1588). Nothing is owed on any of them. #933 and
  #968 still print `main's` as their lease age, which is #1548. All twelve of
  this window's claims were deleted automatically at merge.
- **`parked/*`: none**, as printed by the tool itself.
- **Other branches: 9, for human triage.**
  - **`claude/dazzling-cerf-c1o1wx`** (`ahead=1 behind=12`) is the stranded
    thirty-second pass. **Do not delete it yet.** Its one unique commit holds
    the LOG entry that this pass's commit carries onto `main`. Once that entry
    is on `main`, a human may delete the ref.
  - **`claude/dazzling-cerf-u3xy2k`** (`ahead=1 behind=50`) was certified safe
    to delete by #1660. Its content is on `main`. Only a human can delete it.
  - The seven **`claude/eloquent-cerf-*`** refs are `ahead=0`, behind 56–313.
    Each carries nothing `main` lacks.

### Marker census

`go tool gapaudit` was fed this pass's `state=all` walk on `16b4057`. It reports
**70 markers across 9 areas, with 17 in group 1 and 28 in group 2.**

- **The total went from 69 to 70.** #773 retired the two String Valid clause 3
  markers (`validate/cvcattribute.go`, `validate/cvccomplexcontent.go`) and
  added the owned `[unparsedEntities]` marker at `cvcattribute.go:25`. #774
  added the owned `validate/cvcsimpletype.go:57`.
- **Group 1 stayed at 17, and its membership changed.** #1156 wrote `#345` into
  `xsd/contentrestricts.go:1200`, and that row left the group. Run against
  today's feed, `dc12a98`'s tree scores 27. The extra rows are `:1200` plus nine
  markers in `cvcattribute.go`, `cvccomplexcontent.go`, `cvcid.go` and
  `cvcidentityconstraint.go` that cited #773 or #774, both now closed. Those two
  landings rewrote all nine, so none became a dead end. The group stayed flat
  rather than dropping to 16, so one standing row entered it. This pass did not
  identify that row.
- **Zero dead-end rows, for the fourth consecutive pass.** Every group-1 row
  carries at least one open candidate owner, except `parser/doc.go:221`, the
  §5.3 policy row that points to a human-reserved decision.
- **No new `kind/gap` issue is owed.**
- **Group 2 went from 27 to 28.** The new entry is **#1670**
  (`parser/xmltree` NDataDecl shapes), filed this window. Group 2 also holds the
  conformance-lane gaps, which never carry a marker.

### Milestones and queue

**One page-numbered `state=all` REST walk, stamped `rows 1705 distinct 1705 min
1 max 1705 no gaps`**, taken after this pass filed #1705: 915 issues (PRs
excluded), **347 open, 568 closed.**

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **53** | **138** | active |
| **M5 — Instance validation (XML)** | **15** | **38** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels on open issues are **334 `ready`, 11 `blocked`, 0 `needs-replan`
and 2 `epic`**, which sums to 347. Every open issue carries exactly one queue
label, at least one `kind/` label and at least one `area/` label. By kind:
`kind/refactor` 79, `kind/process` 69, `kind/gap` 53, `kind/tooling` 49,
`kind/story` 42, `kind/docs` 38, `kind/bug` 33, `kind/feature` 3.

**M4 moved: 53 open, 138 closed, the first movement in seven windows.** #479 and
#601 closed in it, and #1701 was filed into it. **M5's carve is fully landed**
now that #773 and #774 have closed. Its 15 open issues are follow-ups and
`kind/gap`s that grew beyond the original eighteen slices, and the M5 section's
scope paragraph is corrected to match.

**`blocked` went from 12 to 11.** #831's post-land pass unblocked #871, and
#479's unblocked #1458. **#1698** (gate part 1 outruns the 10-minute package
timeout) was filed `blocked`. **This pass's unblock sweep found zero.** #1609's
trigger (`schema` above 14697) has not fired.

**The `ready` audit found zero mislabels and zero duplicates.** Class 6 (a
required body section missing) is **0 of 333**. The 09-24 pass repaired the four
that four consecutive stamps had carried (#586, #845, #848, #1419). Class 7 (a
lane-visible body with no ratchet bar) is **3 real of 82**.

**What this pass wrote to GitHub.**
- Closed **PR #1689** as superseded, and left its branch alone.
- Filed **#1705** (`kind/process`, `kind/tooling`): a `/backlog` pass whose own
  docs PR never merges strands its write-up, now sighted three times
  (`meta/audit-2026-09-06`, PR #1632, PR #1689). `backlog.md` line 22 already
  forbids it. It does not overlap #1674, which covers a `/develop` LOG commit
  left on local HEAD, or #1649, which was the one-off recovery and is closed.
- Made **two body PATCHes**, each via `gh api -F body=@FILE` after a
  byte-faithful read:
  - **#345**: its *"`contentrestricts.go` marker still names nobody"* premise
    was false after `5f4b197`. The body and title are restamped to `:1200` and
    `defaultbinding.go:298` at `16b4057`.
  - **#849**: the cost-of-delay figure is restamped to `16b4057` (4, flat).
- Posted **one disposition comment on #601**. The generic-agent grounding round
  recorded at `docs/LOG/2026-09.md:44031` is dismissed as a self-corrected first
  sighting, and a second sighting makes it a `develop.md` finding.

### Persona consultations: none owed this pass, with provenance recorded

**The cartographer role-plays no persona and does not spawn one** (#416). **The
orchestrating session handed this pass no persona report, and it judged that no
consultation was owed.** `go tool surface` since the last stamp that reached
`main` reads **+10/−1**, all from #479 (`AttributeUseOrGroupRef` and the
attribute-content sum) and #773. **The stranded thirty-second pass already ran
`libuser` and `cliuser` on exactly that surface**, plus #1642's `conformance`
package, #848's `Validator.Schema()`, #1668's ENTITY/DTD handling and #1419's
exit codes. Its findings are live on GitHub even though its write-up never
reached `main`:
- five `kind/docs` filings, **#1684–#1688**;
- **#848** ruled: unexport `Schema()`;
- **#1419** ruled: no exit-code change, re-kinded to `kind/docs`.

**Fourteen persona findings are open and unconsumed:** #1568–#1571,
#1593–#1596, #1626, and #1684–#1688. Rows 10, 11 and 13 below take three of
them. Routing for the next consultation is unchanged. **M5** is API- and
CLI-facing. **M4** is API-facing only. **M6** has no surface yet. CLI work is not
a milestone, and it is tracked by `area/cmd` and `area/cli`.

### Working band

This is ordered for a `/develop` session: take the highest row you can start.
**No claim stands on the remote**, so every row is startable. **Re-run
`wipsurvey` before starting anyway**, because this is a snapshot. Each row names
one issue (#1636).

**The ordering principle for this stamp:** process and tooling friction that
every landing pays comes first. Then come the lane rows the last window
unblocked, then the one measured refactor, then persona consumption.

| # | issue | why here |
|---|---|---|
| 1 | #1674 | **A landing gate that passes lost work.** `landcheck` precondition 1 reads the local HEAD, so #1499's unpushed LOG commit passed and the squash landed without it. Every landing runs it. `kind/process` + `kind/tooling`, one session |
| 2 | #1705 | **Third sighting, and this pass is the recovery cost.** A stranded `/backlog` PR is invisible until a later stamp notices it. A detector (a `wipsurvey` STRANDED PASS row, or a survey step) replaces a rule that has already failed three times |
| 3 | #1671 | **The reserved route from the last stamp, now owed.** #773 answered `not derived` and banked +9, and #1641 predicted `unchanged` and banked +1. No `suiteindex` query resolves a QName-valued attribute against its bindings. Every `xsi:type` or QName prediction pays for this |
| 4 | #1695 | **Warm from #1579**, which landed 2026-09-25. The CLAUDE.md `suiteindex` block keeps a stale count, and its `|` over-count sentence misses the `{open content}` a derived type inherits. Doc-only, and cheap |
| 5 | #1682 | **Real bug, warm from #831.** A `ref=` attribute use with no `inheritable` gets `{inheritable}` false instead of its declaration's value (§3.2.2.3). M4, `kind/bug` |
| 6 | #871 | **Unblocked this window by #831.** §3.12.4 clause 1.1.3's `{inherited attributes}` merge. It is the first candidate owner `gapaudit` prints for the group-1 row `validate/cta.go:121`. `instance` lane |
| 7 | #1676 | **M5 `instance` gap.** An absent ID/IDREF-governed defaulted attribute declines in `idDefaultedAttributes`, which silences cvc-id clause 1 for the whole validation |
| 8 | #1458 | **Unblocked this window by #479.** Supply `xml:specialAttrs` so `<attributeGroup ref="xml:specialAttrs"/>` resolves. `schema` lane |
| 9 | #849 | **The one measured `kind/refactor`: 4 copies, flat against its recorded figure.** It enters the band under step 4 and does not outrank a lane slice, because the figure did not grow |
| 10 | #1419 | **Persona consumption, already ruled.** No exit-code change. State in `validate`'s exit-code contract that an I/O fault on a referenced schema exits 3. Doc-only |
| 11 | #1686 | **Persona consumption.** `UnionNamespaceConstraint`'s exported doc points at an unexported function that `go doc` cannot show since #479 |
| 12 | #1617 | **The census this stamp quotes has an untested tolerance in the silencing direction.** Dropping `rulingPattern`'s `(?i)` breaks no test |
| 13 | #1688 | **Persona consumption.** The `conformance` package doc says "test-only" and never says why an importable package with exported `Catalog` sits outside the library |

**Named below the band, on purpose.**
- **Persona findings:** **#1684**, the `AttributeUseOrGroupRef` example, is the
  thirteenth-row alternative to #1688. **#1685** and **#1687** are the other
  09-24 persona filings, and **#1568**–**#1571**, **#1593**–**#1596** and
  **#1626** are the older nine.
- **Not yet banded:** **#1668** (`HasUnparsedEntity` omits the external DTD
  subset) follows on from #773's ENTITY work.
- **Filed this window:** **#1670** is the newest group-2 gap. **#1701** pins
  `addAll`'s continuation half, filed by #601's post-land pass.
- **Cheapest unblocks:** **#1324** unblocks #1344, and **#591** unblocks #593.
- **Unmeasured refactors:** **#848** (ruled unexport) and **#845** are ordered by
  dependency alone.
- **Tooling and comment repairs:** **#1618** and **#1616**, the other
  gapaudit/STYLE pair. **#1624** and **#1613** are comment repairs.
- **Cheapest M5 gaps after the band:** **#1576** and **#1572**.
- **Arm (b) startable today:** **#345**, body restamped this pass, has an
  arm (b) that needs only a ruling.
- **Process findings:** **#1629**, **#1630** and **#1636** cover the band and
  re-scope process.
- **Compounding:** **#1522** and **#1019** are compounding and still untouched.

### Next planning action

1. **`/retro` is due Sunday 2026-09-27**, having missed 2026-09-13 and
   2026-09-20. The last one landed 2026-09-06. It owes three items:
   - **#1698**, the gate-timeout decision it is blocked on;
   - **the M4 question**: M4 moved this window, but whether it is still a
     parsing milestone or a directory of `parser`/`xsd` work is still open;
   - **the persona trigger**: fourteen findings are open, and this band takes
     three.
2. **Band test for the next stamp.** If rows 1–2 land, the next `/backlog` checks
   that its own PR merged. It also checks that `wipsurvey`, or whichever
   instrument #1705 chose, would have flagged #1689.
3. **A human can delete `claude/dazzling-cerf-u3xy2k` now**, as certified by
   #1660. **`claude/dazzling-cerf-c1o1wx` becomes deletable** once this pass's
   commit (which carries its LOG entry) is on `main`. A session cannot delete
   either.
4. **The human decision blocking #1002 is unchanged, carried for a
   TWENTY-SEVENTH stamp.** The choice is between (a) a constitutional
   "superseded pass" ratchet class and (b) holding §4.2.2's `vc:maxVersion` arm
   until assertions land. CLAUDE.md puts (a) beyond any agent.

## Milestones

### M0 — Scaffold — done

Repo layout, docs, local specs plus conversion tooling, W3C suite
submodule, per-package `doc.go` contracts, agent personas and commands,
lint gate.

### M1 — Spec infrastructure — done

The hfn → TypeSpec generator emitting `builtin/gen_typespec.go` for all 49
builtins including precisionDecimal, sourced from Appendix E and the
per-type property tables; the conformance ratchet (expectations
load/compare/merge, upward-only, `suite.xml` runner, lane files); and
`xsderr`'s `Rule`/catalog wiring.

### M2 — Foundation leaves — done

`xsderr` (Error/Rule/Loc plus narrowing helpers), `loader` (Resolver with
Dir/FS/HTTP/Map/Chain), `parser/xmltree` (streaming position-tracking
decoder), and the `xsd.QName` expanded-name type that `value.Backend` and
the builtin table key on. Full unit tests; fuzz targets for xmltree.

### M3 — Datatypes vertical slice — complete

All 20 primitives mapped; the facet pipeline, value spaces and canonical
mappings behind the **`datatypes` lane**. Remaining open work is
follow-up, not milestone scope.

### M4 — Schema parsing — active — [epic #79](https://github.com/kud360/goxsd8/issues/79)

Three-phase parser over the composition model — include, import, redefine,
override, chameleon coercion — with UPA, EDC and particle restriction
designed into the model shape from the start, plus the `xsd` component
model and the finalize/resolve phase (`src-resolve`, dependency-ordered
finalization, named-circularity rejection). **`schema` lane.**

The original carve (#167–#183) is landed. **Every §3.4.2 complex-type
representation form is now produced**: `<simpleContent>` with
`<restriction>` was the last one declined outright and #909 built it on
2026-08-22. Open work is the long tail of producer widening, finalize
validity and composition edge cases, and it has changed shape — what
remains is predominantly **s4s grammar faults the producer decides and
ACCEPTS** rather than forms it cannot build. The two exemplars this
paragraph used to name, #956 (child order and `maxOccurs` across the
derivation alternants) and #958 (a facet element name the gate admits and
the producer drops), both landed on 2026-08-22/23; **#884** (malformed named
`<group>` bodies collapsing into one `mgd-props-correct` message) joined them at
`b3f295a`, and **#972** — an XSD-namespace child §4.1.2's
`<simpleType><restriction>` has no position for, dropped by `restrictionFacets`
so the producer built a schema the harness called `valid` — landed at `e491ddb`
for **`schema` +26** — the family's largest single move until **#1047** took
`schema` +34 on 2026-08-28. **#471** landed the local `<element ref=>` carrying
`substitutionGroup=` on 2026-09-03 (`31263ed`, `Ratchet: unchanged` — the
grammar-level `use="prohibited"` shape rejects with no case in the corpus to
bank) and its grounding carved two siblings from its scope: **#1205**
(`final=`/`abstract=`, the other grammar-prohibited pair) and **#1206**, which
**landed 2026-09-04** (`1780670`, `schema` **+16**) — `src-element` clause 2.2's
prose-only eight attributes on a local `<element ref=>`. **#1205 LANDED
2026-09-06** (`24ac7d3`) for `schema` **+5**, and it retired this family's
"ratchet-neutral by construction" inheritance: #471's neutrality came from
reaching the `ref=` form ALONE, while #1205 also reaches the inline form, where
six local fixtures exist and five moved. Its `ref=` arm IS neutral as predicted
(**0** fixtures suite-wide pair `ref=` with either attribute), and the inherited
prediction never shipped as a claim because that issue's own Acceptance made
measurement a precondition. **#1206's own landing then
carved two more**, from a UTF-16LE corner of the suite no UTF-8 grep had
reached: **#1215** (`src-element` clause 4, `ed-with-ns`) and **#1216**
(`src-attribute` clause 6, `att-with-ns`). **Both LANDED 2026-09-04** —
`976bdc2` for `schema` **+4** and `aeaf59e` for **+5**, each banking one and two
cases past its own prediction — and **each carved a successor from its own
boundary**: #1216's two `GAP(xsd)` markers became **#1242** (clause 6.1's one
reachable failure, a local `<attribute ref=>`) and **#1243** (an
`<attribute use="prohibited">` escaping clause 6 entirely), and **both have now
LANDED** — #1243 on 2026-09-05 (`3febb22`) and #1242 on 2026-09-06 (`6e95f25`),
**each `Ratchet: unchanged` and each measured that way by `go tool suiteindex`
construct census rather than assumed** — ratchet-neutral like #471 before them
(NOT like #1205, which banked **+5** on its inline arm), but the first two of
the family to prove it with an instrument instead of a grep. **#1243 carved one successor, #1242 carved none**,
which ENDS the run below: #1270 (`src-attribute` clause 4 charged nowhere on the
`use="prohibited"` branch) was #1243's, and #1242's landing closed its arm with
no residue. **#1270 has itself now LANDED — 2026-09-06, `d0fd4c6`,
`Ratchet: unchanged` and measured that way by construct census — and it carved
NO successor of this family either.** Its two residues are #1301 (a duplicate
clause-4 predicate surviving at the top-level producer, still open) and #1300 (a
process defect about how a gate command is invoked — **LANDED 2026-09-07,
`34aee07`**, `Ratchet: unchanged`); neither is a producer that decides and
ACCEPTS, so neither extends this family. **#455 has now LANDED too — 2026-09-07,
`8aca2a3`, `Ratchet: unchanged`, structurally excluded before it was measured
twice — and it CARVED a successor: #1328** (`use`, `form` and the two
`*FormDefault` attributes compared raw at every site, measured accepting
`use="foo"` and reading `use=" required "` as `{required}`=false). It also
unblocked **#456**, which was filed long before and is a discharge rather than a
carve. **#931 has now LANDED too — 2026-09-08, `c5e5e375`, `schema` +6
(fail → pass), zero regressions** — the occurrence attributes on a named
`<group>`'s body compositor, and this chain's first non-zero since #1205's +5.
**It ends the zero run**: the four landings before it all read
`Ratchet: unchanged` (#1243, #1242, #1270, #455) and the fifth banked six, so
ratchet-neutrality is a property of the individual member and not of the family,
and no further member may inherit the prediction. **It is also the one member
whose count was NOT taken by construct census** — its prediction came from a
hand-written producer-level probe, was wrong on three of its four named
candidates and under-predicted the lane by two, which is the defect **#1332**
owns. It carved no producer-side successor: #1332 is `kind/process` and **#1333**
pins the `xs:redefine` entry into `buildDefinitionModelGroup` by test, and
neither decides and ACCEPTS. **#456 has now LANDED too — 2026-09-08,
`5b84206`, `schema` +20 (fail → pass), zero regressions** — every one of the
twenty a suite-declared-`invalid` fixture writing an xs:boolean schema
attribute outside `booleanRep`. **It is the SECOND consecutive non-zero and the
largest single bank in this chain's record** (against `unchanged`, #1205's +5
and #931's +6), which settles what #931 opened: the run of four ratchet-neutral
landings before it was the anomaly, not the rule, and no member may inherit
either prediction. **It also answers #931's census defect from the other
side.** `go tool suiteindex` could not take this census either — it censuses by
construct, and this population is defined by attribute VALUE — but the member
said so before being asked and predicted the twenty EXACTLY by censusing the
attribute values directly. **Naming the instrument's limit is what #931 did not
do**, which is the distinction #1332 now owns. It carved no successor: its one
unmet acceptance item was dispositioned to the existing #717, not filed.
**#1328 has now LANDED too — 2026-09-09, `51d9cf9`, `schema` +18 (13968 →
13986, fail → pass), zero regressions** — `use=`, `form=` and the two
`*FormDefault` attributes read as ·actual values· rather than raw literals, all
four being `xs:NMTOKEN` restrictions Appendix A enumerates and `cvc-datatype-valid`
stating no fallback clause. **It is the THIRD consecutive non-zero** (#931 +6,
#456 +20, now +18) and **the first member of this family to predict its flip set
EXACTLY**: banded on 24 enumerated fixtures plus ten measured shapes, it banked
the body's 18 `fail` rows to the letter and left the 6 `pass` rows the body
flagged as the risk unmoved — the census re-taken against the tree rather than
trusted from the body. **It carved NO producer-side successor.**
`effectiveDerivationSet`'s silent drop of unrecognized
`block`/`final`/`blockDefault`/`finalDefault` tokens is ruled out of scope AND
unfiled by its own Notes, on a different spec reading (`blockDefault` *"may
include values other than"* the relevant set); its arbiter's one non-blocking
observation was ruled not a STYLE T4 defect and recorded rather than filed; and
#653, whose items 3 and 4 this landing discharged, took a thread comment and a
body re-statement rather than a new issue. Live producer-decides-and-accepts
members are now **#929** alone — which #455's landing widened to own
`minOccursZero`'s clause-2.1.3 compare as well as `maxOccursZero`'s. **The family
carved a successor at
every landing from #471 through #1243, and #1242 is the first that did not** —
one reading is the tail reaching zero, the other is that this pair was narrower
than its predecessors; two more landings decide it. **#1270 was the first of
those two and came back empty; #455 is the SECOND and came back with #1328**, so
the test the last stamp set is answered — but **#1328 is itself the THIRD
CONSECUTIVE member to carve nothing**, after #931 and #456, so *"the family is
still carving"* is no longer the reading the record supports: the generator is
producing landings and lane movement, not successors, and the queue this family
feeds has narrowed to #929 alone. Note what carried the one carve there was —
#455's successor came out of an ARBITER's non-blocking scope note, not out of a
construct census, which is a different generator from the one #1270 and #1242
exhausted; #1328's arbiter produced a non-blocking observation too, and that one
was ruled not a defect, so the generator fired and came back empty. A
second, narrower family opened beside it — the rejections the
producer already makes **correctly but describes badly**, whose bar
`xsderr/doc.go` set with #966 — and it is **discharged**: #975 landed at
`1dcffbf`, so every s4s-grammar rejection now names its Appendix A production.

**A THIRD family is the one that has been paying, and it has SIX landed
members.** #1030's unmapped-construct census turned the "decides and ACCEPTS"
family from a shape into a list, and the list delivered `schema` **+34**
(#1047), **+23** (#1046, plus `instance` +15), **+21** (#1076), **+16** (#1048)
and **+2** (#1099). For #1046/#1047/#1048 the gate-side alternative was
**measured and ruled out** — widening `conformance/schema.go`'s shape gate costs
a banked ratchet win — and #1076 and #1099 arrived with no measured count at
all, banded on the criterion *"the shape is in the suite's invalid corpus"*
instead. **That criterion is five-for-five on DIRECTION and predicts nothing
about MAGNITUDE**; the +21 and the +2 came from the same family four days
apart. The Status section carries the consequence for how the band is ordered.

**The sixth member is #1275 (`34c39be`, 2026-09-07) and it is the first to bank
NOTHING — because it did not use the criterion, it measured.** A twelfth
`s4sModel` orders an `<alternative>`'s own children against `xs:altType`, the
same section 5.1 first-bullet fault #1047 and #1076 charge at their positions;
the corpus was censused before implementation and holds **no** `<alternative>`
carrying an out-of-model child across 232 occurrences in 81 fixtures, so
`Ratchet: unchanged` was predicted and then measured. **The criterion is a proxy
for a census that `go tool suiteindex` can now take directly**, and where the
two disagree the census is the fact — that is the methodological change this
member carries, not a sixth data point for the proxy. Read the criterion's
five-for-five as five-for-five *among issues banded on it*, which is a smaller
claim than it looks.

**The family replenishes itself as it is worked, which is what to expect and
not tail growth** — every issue it has added was filed by the post-land pass of
a landing that moved the lane. **Its shape is also changing**: #1047, #1076 and
#1099 all widened `checkS4SChildOrder` or its callers, and the issues arriving
now — #1097 (`32070b8`) and **#1136** (`d854e1d`, 2026-09-05), both landed, and
open #1098, #1133 and #1135 — are defects *in* that widening rather than further
sites for it. **A producer that decides more is a producer with more to get
wrong, and the census does not name that class.** #1136 is the one that stopped
the accumulation: it replaced four independent local run-order guesses with ONE
recorded decision, and its own follow-ups (#1246, landed 2026-09-07 as
`a363592`, and #1247, landed 2026-09-08 as `77b0419`) are about that record
rather than about a fifth guess. **That record now states a rule scoped by
WHERE rather than a family roster**, and #1247's own follow-ups (#1342, #1343)
are about its wording rather than its membership.

**#1275 shows the site-widening half is NOT exhausted, and it arrived from a
third route again.** It is a further SITE — a twelfth `s4sModel` at a position
nothing had ordered — not a defect in the widening, and it came from neither
#1030's census list nor a run-order defect: #1050's reconciliation of
`parser/census.go`'s Scope prose NAMED the silence in the tree and tracked it
with nothing, and #1275 was the filing that was owed. **Every widening site the
producer still owes is discoverable from that Scope section**, whose remaining
live silences it enumerates; the open **#1276** would make that section's truth a
DERIVED test rather than prose, which is what would turn the remainder into a
list the way #1030 did once already.

**Two more of this family landed 2026-09-09/10, and between them they closed
the two AXES rather than two more sites.** **#1369** rejects an unprefixed
attribute no Appendix A production of its element declares (`schema` **+17**),
on a hand-transcribed roster of all 49 productions; **#1380** rejects an
XSD-namespace `<schema>` child Appendix A admits at no top-level position
(**+9**), and its load-bearing finding was that **the parser charge alone banks
ZERO** — all twelve suite witnesses were declined by `conformance/schema.go`'s
top-level `default` arm, so the harness widening had to be absorbed. **What
remains after them is neither axis but the VALUES on them**: `maxOccurs="00"`
(#929), `##defined` (#606) and the eleven cases #731 carved into #1411–#1414 are
all attribute-VALUE faults. **#1391 landed the attribute axis they needed**, so
`go tool suiteindex '*@maxOccurs'` and its siblings now census them; #929 and
#606 both still predict `unchanged` for want of a census that is no longer
missing, and re-ranking them on one is owed.

**A schema-construction slice can sit OUTSIDE §5.1's grammar-and-`src-*` frame
entirely, and the expensive case is §5.3.** **That question is now RULED**:
§5.3 governs a failed `·resolution·`, so `Finalize`'s hard-fail is a deliberate
implementation policy choice rather than a spec requirement, and reversing it
costs 35 measured `schema` cases with no ratchet mechanism able to record the
loss — a call CLAUDE.md reserves to a human-filed issue. The ruling is on #1426's
thread; #1426 is closed `not_planned` and **#1498** carries the marker prose that
records it. **#1429** owns the harness guard the attempt broke. **Both missing
built-in component sets have LANDED** — **#1427** on 2026-09-12 (`c1f31b5`,
`schema` +7, `instance` +15) and **#1428** on 2026-09-12 (`cd36ad8`, `schema`
+2). **Neither waited on the ruling, and that is the finding** — the withheld
flips were a missing component set, not a §5.3 deferral, which is now measured
rather than argued.

**Read the milestone count as a floor.** The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it — #1135 is M4 work carrying no
milestone today, and #1136 was too until it landed. **The sharpest instance for
LANE MOVEMENT is #1126**, which moved `schema` +475 and `instance` +113 in one
landing and carried no milestone, so the count did not move at all. (**#1411
passed it on the `schema` side on 2026-09-13 with +609**, banking the
`MS-Regex2006-07-15` corpus whole; #1126 remains the largest that moved two lanes
at once.) **The sharpest for SCOPE is #434**: a core §5.3
schema-construction issue, banded as the best lane slice in the queue, carried no
milestone for its whole life and closed outside M4's count.

### M5 — Instance validation (XML) — [epic #250](https://github.com/kud360/goxsd8/issues/250)

`validate` engine plus `validate/xmlsrc`: non-backtracking content matching,
identity constraints, `xsi:type`/`xsi:nil`, wildcards, open content, default
and fixed values. **`instance` lane.** *"Greedy"* is struck here and from
PRINCIPLES 14: #782 replaced the greedy walk with a live SET of partitions,
which `cvc-accept` clause 3.1's existential requires, and the invariant that
survives is no-backtracking rather than greed. **"Bounded" is struck too, and
by #1557 rather than by #782**: arm (b) made `partitionsBounded` an ESTIMATE of
the cover instead of an upper bound on live entries, and what survives
unconditionally is a per-item BUDGET — one item's cost is a constant of the
SCHEMA, never a function of the items already taken.

Carved 2026-08-12 into ten slices, #710–#719, and now **eighteen** — #766 was
split out of #714 by a warden pre-flight, #773, #774 and #775 out of #766 by
another, #782 and #783 out of #715 by its own, **#790** filed from #715's
warden verdict, which had named the seam and been read by nobody who filed it,
and **#1516** for the `{open content}` clauses. **All eighteen have landed.**
The last two, #773 (`c1851ae`, `instance` +9) and #774 (`5fb2cd5`, `Ratchet:
unchanged`), landed on 2026-09-24. #1516, #782 and #783 all landed 2026-09-16/17
and took `instance` 11151 → 11203. Of the thirteen that had
landed before this window, **#719** joined them on 2026-08-27 at `bd887cd`, wiring
`cvc-assertion` fail-open at every variety level and shipping the
`validate.Unevaluated` channel; it moved no lane and correctly so, because a
fail-open declines exactly where the engine already declined. It unblocked #56
and its successor **#1042** (assertion EVALUATION) is the first issue this
project has ever placed on **M6**. The other twelve: the infoset
seam and engine skeleton (**#710**), the `xmlsrc` adapter (**#711**) and root
dispatch (**#712**), none of which moved the lane and correctly so — the first
two decide no `cvc-` rule and nothing yet ran the cases; **the lane driver
(#713), which took `instance` off zero — 0 → 18 pass** (the #175 analogue, placed
fourth so every slice after it reports a real number); `cvc-complex-type`
clauses 2–3 (**#714**, 19 → 29); the `value.Backend` seam with the four
datatype-valid attribute charges (**#766**, 29 → 193); the greedy
non-backtracking content matcher (**#715**, 193 → 520); `cvc-complex-type`
clause 1.2's ·initial value· against String Valid (**#775**, 532 → 535), which
also closed #759 and landed `docs/STYLE.md` **E4**; **the descent (#790,
535 → 1017)**, which threads each descendant's ·context-determined declaration·
into the recursive walk (§3.3.4.6 clause 3.1) — the largest single move any lane
has recorded; identity constraints and the ID/IDREF table (**#718**,
1017 → 1133); `xsi:type` and `xsi:nil` deciding rather than declining (**#716**,
+183); and a union-governed item classified by its ·validating type· (**#813**,
+9, unioned onto #716's). #913's cvc-type clause 3.1 landing added **9409**,
itself M5 and the largest single lane move this project has recorded.

**Landings OUTSIDE this milestone keep moving this lane, by four distinct
mechanisms, and a running total of them is not maintained here** — take the
figure from the Status section's table (#646). The four:

- **A slice PRODUCES a component the engine could not previously see** — #733
  (a top-level `<xs:attribute>`'s inline `<xs:simpleType>`), #909
  (`<simpleContent>` `<restriction>`), the CTA pair #842/#851. **#1427 LANDED as
  the next of these on 2026-09-12** (`c1f31b5`) and is the mechanism's cleanest
  instance: §3.2.7's four built-in `xsi:` attribute declarations were seeded as
  COMPONENTS, a schema referencing one stopped hard-failing `src-resolve`, and
  **`instance` banked +15** alongside `schema` +7. **The figure this paragraph
  used to carry — 14 — was an inference transcribed from `wip/issue-434` and was
  WITHDRAWN at grounding** before the landing measured 15; read it as the worked
  example of why a transcribed prediction is re-derived rather than inherited
  (#1332).
- **A slice DECIDES a `{type table}` the engine previously withheld.**
- **The MEASUREMENT is fixed and the engine never changed** — #862, where
  `resolveExpected` was picking the wrong feature-scoped expected verdict, so
  the answers had been right all along. A lane can move because the measurement
  was wrong.
- **The SCHEMA the engine was handed is fixed** — #1001, where §4.2.2's
  ·conditional inclusion· removed declarations colliding under
  `sch-props-correct` clause 2, so five documents the engine had never been
  given a usable schema for became decidable.

**The largest of them is #853 (+141 at `2310710`) and it carries no milestone**,
which is a finding about the LABEL and not about M5's carve: it decides a `cvc-`
rule at assessment time on an XML instance, which is M5's own definition of its
scope, and it sat outside only because a `kind/gap` filed by a post-land pass
never acquired one. **Read the M5 milestone count as a floor and never as the
lane's remaining work** — the same caveat M4's section makes, for the same
mechanical reason, and the `instance` lane is where it costs most.

**#853 repeats #913's lesson rather than #790's.** It was banded UNMEASURED,
with an instruction to go count first, and outperformed every candidate that
arrived with a count: a slice that decides a *new* rule on a commonly-declined
shape moves the number far more than its rule count suggests. **The Status
section carries what three stamps of this have settled** — a document count is
a filter on candidates, never a sort key, and the invalid-corpus criterion
predicts direction and not magnitude.

**A flat M5 landing is routinely CORRECT and the section should not read as if
it were not.** #1043 (`5d3d222`) declined the ·governing type definition· of a
skip-wildcard attribute, withdrawing a `cvc-id` charge §3.10.4.1's Note says was
never owed; #1116 (`9919faf`) charged cvc-complex-type clause 2 against an
anonymous governing type behind a harness gate that admits no case of the shape.
Withdrawing a wrong charge on a document already banked `fail`, or deciding a
shape the executor withholds, cannot register as movement. **That is what the
ratchet's zero-flip-down means, and it is a result to explain rather than a
null one** — and #1116's explanation was banked and then paid out. **#1126
(`d433f7f`) deleted the harness gate #1116's `Ratchet:` trailer named**, and both
lanes moved on the same run: `schema` +475, `instance` +113. The charge was
right when it measured flat; the executor was what withheld it.

**The milestone's shape changed with #790, not just its number.** The first eight
slices decided the ·validation root· and nothing else, so each one bought tens of
cases; #790 bought 482 by making the same rules reach the other 99% of every
document. **The remaining slices inherited that multiplier and spent it**: #718
and #716, the band's top two at the time, bought a further 299 between them by
deciding their rules at every node rather than one.

**The carve is no longer a chain, and reading it as one loses work.** The
original ordering assumed a slice becomes `ready` as the one before it lands.
That held until #714, which had **two** successors directly behind it, and #766
has **three**; #715 had four, of which #716, #718 and #719 became `ready` when it
landed while #717 waits on #248 as well. **A slice can also travel backwards, and
then forwards again**: #718 went `ready` → `blocked` when its own grounding
falsified its premise and filed #790 as the enabler it needs, and returned to
`ready` when #790 landed. Both moves were made by reading the *code* against the
falsifying claim, not by reading the dependency's closed state — the round trip
is the worked example of what a `## Depends on` line is for. Relabel from those
lines,
never from the milestone or from the issue numbers' order — and sweep for issues
carrying **no queue label at all**, which is how #773 and #774 sat outside both
queues for a day. **The CLI's own `validate` subcommand, #720, LANDED
2026-09-04** (`f4d8f76`) and moved no lane by construction — `go list -deps
./conformance | grep -c cmd/goxsd8` is `0`. It is the reason this milestone's
count moves independently of its lane, and the 2026-09-05 window is the cleanest
demonstration yet: **three of its four subcommand issues LANDED** — #1223
(`4ab65d6`, exit 4 for an instance the assessment declined to decide), #1229
(`20010ed`) and #1230 (`38ade3e`) — **three more were filed onto it** (#1250,
#1251, #1257), and the milestone's open count did not move at all while
`instance` stayed flat at 11029. **Every M5 issue that entered or left in that
window was `cmd/goxsd8`.** **The roster of open subcommand issues is not written
here.** It went stale twice inside four days — #1260 and #1261 landed, then
#1251 — for the reason the lane figure below is not written here either (#646).
Read it off the queue: `area/cmd` on this milestone. The durable half is the
milestone rule behind that roster, and it is the reason the count reads low: **a
subcommand issue spanning `parse` (M4) as well as `validate` carries NO
milestone deliberately**, so this milestone's `cmd/goxsd8` work is undercounted
here rather than overcounted. Read the count as a floor and never as the lane's
remaining work.

**The lane score is a floor built for soundness, and no jump has ever changed
what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the failures
that remain are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read the lane score as a pass rate.** It is the count of documents this
engine can honestly call not-valid. **Read the current figure from the Status
section's table and never from this paragraph** — an absolute figure written
here is stale before the next landing (#646). It grew most because #913
decided `cvc-type` clause 3.1 — the commonest simple-typed-leaf shape the lane
had declined outright — which is the counterpart to #790's lesson, not a
contradiction of it: a slice that decides a *new* rule moves the number far
MORE than its rule count suggests when the declined shape is common, and #913
moved it more than #790's descent did.

**All ELEVEN decided-and-wrong `instance` cases carry an owner, and the two
classes below account for nine of them.** At `5b84206`: 15332 fail = 15316
decline candidates + 5 declined indeterminate + **11** decided-and-wrong. The
other two are owned outside these paragraphs (#1160) —
`MS-DataTypes2006-07-15/gMonth002_2061/instance/gMonth002_2061.v` and
`gMonth004_2063.v`, both by **#921**.

**It was TWELVE, and `Open/open013/instance/open013.v1.xml` is the case that
left — reclassified by #456, not fixed by it.** #456 landed the `boolAttr` fix
(2026-09-08, `5b84206`), so the fixture's `<xs:complexType mixed=" 1 ">` reads
its ·actual value· `true` and the false reject is gone; but a correct `mixed`
makes the ·effective content· a particle (§3.4.2.3.3 clause 2.1), routing to
clause 5.2.1's default open content, and `xsd.Schema.ContentMatcher` declines
any `{content type}` carrying `{open content}` under a `GAP(xsd)` owned by
**#717**. Same banked line, a decline candidate now rather than a wrong
decision — correctness up, lane score flat. The banked figure is
`go tool lanestatus`'s; the decline figures are the arbiter's, taken at the
same tree.

**TWO classes are decided and decided WRONG, and the first is two cases, both
#771**: `ElemDecl/targetns00101m/instance/targetNS00101m1_p`, whose root
element `{ElemDecl/targetNSa}number` is declared in `targetNS00101m1a.xsd`,
and `SType/st_targetns00101m/instance/ST_targetNS00101m2_p`, whose root
`{ST_targetNSa}test` is declared by neither of its group's schema documents
and whose rescuing `xsi:type` resolves to `{ST_targetNSa}Test` in
`ST_targetNS00101ma.xsd` (attributed by #1160). **One root cause, two routes
to the same charge**: each turns on a schema document the harness never
assembles, because only the instance's own `xsi:schemaLocation` points at it,
so the root-name lookup fails in the first case and the `xsi:type` lookup
fails in the second — and since `validate/cvcelt.go`'s
`instanceTypeDefinition` treats an absent, non-QName and unresolvable
`xsi:type` alike, both land on `cvc-assess-elt`. The first case was one of
four, and #800 retired two of them:
`Assert/assert_019/instance/assert_019_2` and
`CTA/typeAlternatives_001/instance/typeAlternatives_001_2` now decline honestly
instead of rejecting a document the ·conditionally selected· type admits.
**That is two, not three, and `CTA/cta0008.v01` was never among them** — it
takes §3.12.2's inline arm, which #800 deferred to #822 and which **#851 landed
on 2026-08-17** (#822 closed, superseded), and the count was
measured by diffing `GOXSD_DECLINES=1` across the two trees rather than
predicted. All were already banked `fail` before #790 and are not its
regression; the descent is what made them visible, and the trade of a wrong
decision for an honest decline is one the lane cannot register as movement.

**#913 added the second class, and #871 retired it.** Seven declared-valid CTA
documents (`cta0010.v01`, `cta0011.v01`/`.v02`, `cta0013.v01`/`.v02`,
`cta0014.v01`/`.v02`) were false-charged through `cvc-type` clause 3.1 until
§3.12.4's `{inherited attributes}` merge landed (#871, `2e9ab84`, 2026-09-25).
They now decline rather than reject, and stay banked `fail` under #1561, so the
lane cannot register the trade; #871's own `instance` +2 came from `cta0009`.

The decline census that separated harvest candidates from indeterminates
predates every M5 landing from #766 onward and every outside-M5 mover — #909,
#853 and #1116 included — and is not re-derived here. **It is now, by a wide margin, the oldest measurement this milestone still
argues from**, and #786 is the nearest issue to it.

The design constraints are fixed by `validate/doc.go` and PRINCIPLES 8, 11,
13, 14 and 15, and the carve does not reopen them.

### M6 — XPath required subset

CTA restricted subset plus assertion essentials, fail-open with `GAP`
markers, IDC selector and field paths. Dynamic-error direction per
PRINCIPLES 20. Not filed as an epic — speculative epics two milestones out
earn nothing.

**Part of this milestone has already shipped, ahead of it and outside it.**
**#842** landed the §3.12.6 required-subset CTA evaluator on 2026-08-16 at
`3509de6` — pulled forward because M5's `instance` lane cannot decide a
conditionally-typed element without it — and `xpath` stopped being a
`doc.go`-only package. **#861** then corrected its clause-2 error handling and
**#851** gave §3.12.2's inline arm a real anonymous component. Three more
landed on 2026-08-18: **#858** (`d02aa59`, the three cast-shaped constructs),
**#886** (`3867fb5`, `ta-props-correct` clause 2 charged at schema construction
time) and **#887** (`f1250c0`, comparator legality decided from `xpath20.md`
Appendix B.2's generated rows). **All three moved no lane, and all three said
so** — the required subset is now largely built, and building it has not been
what converts cases.

What remains of the CTA subset is **#888** (a cast target that is `xs:QName`),
**#889** (value-level numeric widening, §B.1 rule 1.1's float→double promotion,
which #858 withheld rather than faked) and **#894** (err:XPST0051/XPST0080, the
static remainder #886 did not charge). **#859**, the wildcard `ta-AttrName`
arms, **landed 2026-08-18 at `ea0650a`** — this paragraph carried it as
remaining work for fifteen days, arguing about a stale lease on a branch whose
issue had already closed the same day. **#871**, the §3.12.4 clause 1.1.3
·inherited attributes· merge, landed on 2026-09-25 at `2e9ab84`; #1682 (the
`ref=` use-level `{inheritable}` fallback) now has it as a live reader. None of
the three still open (#888, #889, #894) carries a milestone, which is the same pattern M4's tail
records.

**Assertion evaluation is FILED — #1042, `blocked`, and the first issue this
project has ever placed on this milestone.** It owns `cvc-assertion` (§3.13.4.1)
and `cvc-assertions-valid` (§4.3.13.3) and retires the `GAP(validate)` markers
**#719** landed on 2026-08-27 at `bd887cd`, one milestone early, because the
`instance` lane must decline every case whose outcome turns on an assertion. Its
`## Depends on` is a trigger rather than an issue — an XPath 2.0 evaluator able
to run an assertion `{test}` — so nothing in the queue can start it.

**What is still uncarved is tier 2 itself**: `$value` binding, an F&O function
library and typed comparison, which is too big for one issue. Carving it is a
`/backlog` act and #1042 is now the thing that carve slices, rather than a blank.

**#56 LANDED on 2026-08-28 at `3160813`**, ten days after #719 unblocked it and
`Ratchet: unchanged` as its body predicted. It records the CTA compile-time
withhold into `Result.Unevaluated` under `key-cta-ta-select` (§3.12.4), with no
second type and no `Evaluated bool` — the encoding #719 shipped
(`Rule()`/`Loc()`/`Msg()`, `Result.Unevaluated()` in document order), reused
rather than paralleled (STYLE D4), exactly as #842's warden pre-flight had ruled
on D3. **Surface: none new.** STYLE 9's fail-open discipline is only honest if a
fail-open answer is distinguishable from a real pass, and both of this
milestone's fail-open channels — the assertion sites #719 collects and the CTA
withhold #56 records — now reach the same slice under their own rule IDs.

**Its first consequence is a documentation defect, not a code one, and it is
filed.** `validate/doc.go:101-104` states that an empty `Violations()` beside a
non-empty `Unevaluated()` is not a pass; `README.md:217-219` — the module's only
working validation snippet, since `validate` ships no runnable `Example` (#1088)
— still names `res.Err()` as the sole incompleteness signal and never mentions
`Unevaluated` at all. #56's landing is what made that reachable on an ordinary
conditionally-typed document rather than only on one carrying an assertion.
**#1122** owns it.

### M7 — XPath 2.0 growth

Grammar completion toward full XPath 2.0 plus the F&O function library
(`docs/specs/md/xpath20.md`, `xpath-functions.md`). **`xpath` lane.**

### M8 — JSON instance adapter

`validate/jsonsrc` mapping JSON onto the abstract infoset. **`json` lane**
— curated cases; the W3C suite has no JSON lane.

### M9 — Codegen

Deterministic emission, namer, sealed choice sums, capability-view
interfaces, multiple schemas to multiple output dirs, golden-file tests.
The public `value.Emitter` API freezes here.

### M10 — Codec

Runtime path plus generated fast path; differential tests (identical
values, identical error rule IDs) and `testing.AllocsPerRun` budgets.

### M11 — BER instance adapter

`validate/bersrc`. **`ber` lane** — curated cases.

### M12 — Native backend completion

`builtin/native` mappings and emitter, backendtest green, performance
pass.

## v1.0 — the stability line

1.0 is declared by a human, not by a milestone rollover (expected after
M12). Until then **pre-1.0 mobility** applies: interfaces, package
boundaries and exported names move freely whenever the steward's audit
finds a better placement, and the ratchet and the gate are the only
compatibility promises. After 1.0, exported-surface changes require a
deprecation path and a compatibility argument, and the audit's posture
flips from "move it now" to "guard the surface". Narrower freezes may land
earlier where a milestone says so — `value.Emitter` at M9.

## Non-goals

- Schema mutation and editing APIs.
- XSD 1.0 compatibility quirks — this is an XSD 1.1 processor.
