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

## Status — 2026-09-22 (`/backlog`, the thirtieth)

**The band was followed exactly AGAIN and this time the lane moved: TEN of the
twelve rows were taken in order, NINE landed, and `instance` banked
11205 → 11222 (+17)** against +2 the window before on the same ordering
principle. **FIFTEEN landings over FOURTEEN issues in four days** — #1601 and
#1102 each landed twice, re-scoped in place between the two. **The tenth is row 2 (#1588), which
PARKED** — both its routes falsified by one corpus walk — and re-routed to
**#1601**, which landed twice and banked +4, so the row produced lane movement
without the issue it named surviving. The two rows left standing are 11 (#1579)
and 12 (#1499), both `kind/tooling`/`kind/process`, and neither was skipped for
being unstartable.

**The falsifiable claim the last stamp set was answered, and the answer is that
the ratchet-prediction chain is CLOSED.** Row 9's **#1494** fired `/develop`
step 3's `PREDICTION:` block, answered **`not derived`** for the second time in
the record, and banked **+13** anyway. That second miss is what filed **#1642**,
the join instrument
(`go tool suiteindex -paths <query> | go tool casejoin join <lane>`), and
**#1642's landing then reproduced #1494's figure to the digit**: 25 candidate
cases at HEAD, 38 against #1494's pre-landing expectations file, a difference of
exactly the +13 that landed. **A prediction chain that reproduces a landed lane
figure exactly is no longer a chain under construction**, and rows 1–3 below are
ordered on that.

**The pass's own finding is that the TWENTY-NINTH STAMP IS NOT ON `main`.** The
2026-09-21 `/backlog` ran in full — it filed #1626 and #1627, PATCHed #1156, and
posted #779's third measurement, all live on GitHub — committed its docs half as
`b9eaa76`, opened **PR #1632**, and never merged it. That PR has stood open since
2026-09-21T15:22:28Z on `claude/dazzling-cerf-u3xy2k`, `ahead=1`, carrying a
**256-line `docs/LOG/2026-09.md` entry that `main` does not have**. `git log`
cannot distinguish that pass from a skipped one, which is #400's failure mode
reached by a route #400 does not cover, and **nothing detected it for a day
because no survey looks at the `claude/*` namespace**. The recovery is **#1649**;
the detection is **#1627** arm 2, whose body said that defect did not exist yet
and has been corrected. **This section is therefore replacing the 2026-09-18
stamp, not the 2026-09-21 one** — read `git log -- docs/PLAN.md` accordingly.

### Conformance lanes

**Paste `go tool lanestatus` verbatim, never a hand-count.** This table is
`main` at `da0eb95`, and it is the committed expectations census —
`conformance/testdata/expectations/<lane>.txt`'s `pass`/`fail` line counts —
which `docs/WORKFLOW.md` names as the lane score (#1120) and which is the only
lane figure a later session can re-derive from the committed tree alone.

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11222 | 15139 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14697 | 701 | 15398 |
| `xpath` | — | — | 0 |

**SIXTEEN landings this window and ONE lane moved, by SEVENTEEN cases**, every
one of them `instance`, **zero regressed in any lane**: `git diff a2a72bf da0eb95
-- conformance/testdata/expectations/` shows seventeen `fail`→`pass` lines and no
line moving the other way. Attribution splits clean — **four to #1601**
(`particlesZ007.i`, `particlesZ034_a2/_a3/_b.i`, the `maxPartitionStates`
2048 → 65536 raise at `46dc042`) and **thirteen to #1494** (the `cvc-attribute`
clause 5 charge for an unresolvable `xsi:type` at `585c31b`). `schema` and
`datatypes` are byte-identical to the last stamp.

**Thirteen of the fifteen landings moved nothing and every one of them correctly
so** — #1585, #1565, #1564, #1545, #1560, #1584, #1642 and #1589 are tooling,
process, comment and refactor work that touches no executor; #1102 (twice), #499
and **#1601's SECOND landing** are rulings — the ceiling raise at `46dc042` moved
the lane, the P3b ruling at `a8b3b71` did not; #267 narrows a marker; and #1414
ships one test. **#1414 is the one to read**: its filed premise was spec-FALSE, the
deliverable inverted from a new rejection to a test pinning an acceptance the
tree already had, and it was **re-scoped in place rather than closed and
refiled** — same number, same labels, opposite deliverable, no wrong code ever
written.

**`datatypes ⊆ instance` is INTENDED** (#1507 ruled arm A against outright), so
`instance` 11222/26361 is a figure to quote and not to hedge. **The column does
not sum to the corpus and that is SETTLED**: the three lanes hold 41759 distinct
case IDs against a column summing to 42932, because every one of the 1173
`datatypes` cases is also committed in `instance.txt`, 701 with a contradictory
verdict — two differently mature executors over one fixture.

**An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero.** `datatypes` is M3 and complete; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**No conformance measurement was taken this pass**: `git submodule status` reads
`-7bc3365…`, uninitialized. Every suite figure here is cited from the landing
that measured it, and the lane table needs no submodule.

### Branch namespace, `origin` — report-only; a session never deletes a ref

`go tool wipsurvey` after `git fetch --unshallow origin` — the checkout arrives
shallow and every ahead/behind below would otherwise be fiction (#802).

**THIRTEEN `wip/*` refs, every row RETIRED, NO LIVE CLAIM, so every band row
below is startable.** Twelve are the same twelve the last four stamps carried
(#434, #732, #822, #846, #872, #933, #968, #993, #1332, #1356, #1426, #1451) and
**`wip/issue-1588` is new** — #1588 was parked mid-window and closed
`needs-replan`, so the branch is the re-planning evidence the label retires in
place, which is the lifecycle working rather than a leak. Nothing is owed on any
of the thirteen. Two rows still print a lease age borrowed from `main`'s tip
(#933, #968) — that is **#1548**, still open and still cosmetic.

**`wip/issue-1557`, `wip/issue-1494` and the rest of this window's claims are
GONE**, auto-deleted at merge. Fifteen landings produced zero surviving leases.

**The `parked/*` count this section has reported for FIVE stamps is one the tool
cannot compute, and this is the fifth.** `wipsurvey`'s refspec is
`refs/heads/wip/* refs/heads/main`; `refs/heads/parked/*` is not in it. The
figure is zero, it is right, and it is right by a `git ls-remote` this section
runs and not by the survey it is attributed to. **#1627** arm 1 owns it.

**NINE refs stand outside `wip/*` and no survey reports one of them — and one of
them is now a DEFECT rather than cleanup.** Eight are `ahead=0`:
`claude/eloquent-cerf-39rk64`, `-3xu0ki`, `-8jq9o6`, `-adewly`, `-g3rfvu`,
`-kk1f7v`, `-ldi2dl` at 270/228/275/216/18/144/82 behind, each a merged session
branch whose auto-delete did not fire, plus this session's own ref. **The ninth,
`claude/dazzling-cerf-u3xy2k`, is `behind=12 ahead=1`** and carries the
twenty-ninth stamp's whole docs commit under open **PR #1632**. **The 2026-09-18
stamp set the filing condition — *"if the next stamp finds the same six plus new
ones, file it"* — the 2026-09-21 pass met it and filed #1627, and this stamp
finds those seven plus two more with one of them `ahead>0`.** So arm 2 has its
defect and its body has been corrected to carry it. A session cannot delete a
remote ref here and must not try; #1632 is a human's to close.

### Marker census

`go tool gapaudit` over the whole-repository feed, on `main` at `da0eb95`:
**69 markers, 9 areas, group 1 at 18, group 2 at 28.** Against the last stamp's
69/21/33 the arithmetic closes exactly: #1102's landing DELETED one `xsd` marker
outright and #1494 added one `validate` marker (`validate` 14 → 15), netting the
total flat; group 1 fell by three because #499, #267, #1601, #1565 and #1564
rewrote four markers into STYLE **P3b** citation form, against #1494's one new
untracked site; group 2 fell by five as the `kind/gap` issues those landings
closed left the queue.

- **ZERO dead-end rows tree-wide, for the SECOND consecutive pass.** No marker
  anywhere cites a CLOSED issue as an owner. The row that stood for four stamps —
  `xsd/contentrestricts.go:684`, citing CLOSED #501 and CLOSED #1378 — was
  retired by #1565/#1564 at `314f795`, and nothing has replaced it.
- **FOUR markers now carry P3b rulings** (`contentrestricts.go:697` and `:855`,
  `contentmatcher.go:310`, `defaultbinding.go:90`), which is a new population and
  a new exposure: **#1619** is the observation that a marker retired by
  `matchRuled` has **nothing else in it audited**, so a stale owner citation
  sitting beside a valid ruling is silenced. P3b's last sentence is a prose
  control with no check, and it now governs four sites instead of zero.
- **No new `kind/gap` issue is owed by this pass.** Every one of the eighteen
  group-1 rows carries at least one live candidate owner, and the newest row —
  `validate/cvcattribute.go:136` — was filed as **#1641** by #1494's own post-land
  pass within the hour.
- Group 2's 28 rows are the `kind/gap` issues no marker cites, which includes the
  conformance-lane gaps that never carry a marker and belong there permanently.

### Milestones and queue

**One full page-numbered `state=all` REST walk, clean on the first request,
stamped `rows 1648 distinct 1648 min 1 max 1648 no gaps`**: 1648 rows, 751 PRs
excluded, **897 issues — 347 open, 550 closed.**

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **136** | active |
| **M5 — Instance validation (XML)** | **15** | **35** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **333 `ready`, 12 `blocked`, 0 `needs-replan`,
2 `epic`** — summing to 347, and **every open issue carries exactly one**,
verified over all 347 by grouping each issue's own label set rather than by
summing counts. **Every open issue also carries at least one `kind/` and at least
one `area/` label; there are zero exceptions in either direction.** #779 owns the
mechanical check.

By kind: `kind/refactor` 82, `kind/process` 68, `kind/gap` 53, `kind/tooling`
49, `kind/story` 42, `kind/bug` 34, `kind/docs` 33, `kind/feature` 3. By area:
`meta` 102, `parser` 82, `xsd` 60, `docs` 52, `conformance` 35, `cmd` 34,
`validate` 29, `value` 18, `builtin` 12, `xsderr` 9, `regex` 6, `xpath` 6,
`model` 4, `loader` 2, `cli` 1, `icpath` 1 — sixteen area labels, of which
`icpath` is still named in no document, which is #1019's subject.
**277 of 347 open issues carry no milestone at all**, so read the `area/` census
and never the milestone counts.

**M4 has not moved for a FIFTH window** — 54 open, byte-identical across five
stamps; closed 135 → 136 with #1414. **M5 took the window's weight again**:
30 → 35 closed, open 18 → 15, the largest single-window M5 movement on record.
Of M4's 54 open, **22 are lane-visible** (`kind/gap`, `kind/bug` or
`kind/feature`) against 32 that move no lane by construction — the lane-visible
share fell by one while the open total held exactly, because **#1102** (`kind/gap`)
closed into the milestone and a non-lane issue was filed back into it; of M5's
15, **8**,
and the `instance` remainder of the original eighteen-slice carve is still
exactly **#773 and #774**.

**`ready` 326 → 333.** **SEVENTEEN issues closed and TWENTY-THREE were filed**
in the window, counted over `closed_at`/`created_at` against the walk — the
landings and their post-land passes, plus this pass's one filing. A net of +7 on
a turnover of forty is #347's shape: `ready` is an output, not a target, and a
window that lands fourteen issues and files twenty-three findings is working, not
leaking.

**The unblock sweep measured ZERO**, over all twelve `blocked` bodies'
`## Depends on` sections read byte-faithfully from repository-scoped REST. **The
set is twelve for the fifth stamp running but its MEMBERSHIP changed**: #731 left
it (closed `completed` when #1414 landed the carve its own body said had nothing
left to implement) and **#1609** joined, filed by #499's post-land pass. Its
shape: **five are triggers rather than issues** and say so (#1374, #1224, #555,
#1002, #1042 — **do not re-scan these**); **two state in their own bodies why
every closed dependency is not a discharge** (#16, #1051); **four have exactly
one open named dependency each, and all four targets are `ready` today** —
#593←#591, #871←#831, #1344←#1324, #1458←#479. **The graph is depth-1, no
cycles, no `blocked` body names an epic.**

**#1609 is the first mechanically-checkable trigger this queue has held and it
was CHECKED, not skipped.** Its condition is the `schema` lane widening past
14697; `lanestatus` reads `schema` 14697 and the trigger has **not fired**. Its
body asks every sweep to re-scan it, which is the opposite of the other five, and
that asymmetry is the point — a trigger a tool can evaluate is worth more than
four that a sweep must take on trust.

**The `ready` audit found ZERO queue mislabels and ZERO stale-premise
duplicates.** Class 6 (a required body section absent) is **4 of 333** — #586,
#845, #848, #1419, the same four for the third consecutive measurement, all four
deliberately-shaped bodies. **Three passes have now declined the #845/#848
steward ruling and the last one said what a third decline would mean, so the
reading is recorded rather than the decline repeated**: what those two are
missing is a ruling on whether a steward's `## Current state`/`## Direction`/
`## Cost of delay` shape satisfies the template or is exempt from it, and that
ruling belongs to `/retro`. Class 7 (a lane-visible body with no ratchet bar) is
**6 of 81 by the `(?i)ratchet` test and FOUR by reading the bodies** — #584,
#753, #1049, #1419 are real; **#785 and #1641 state a bar in full without
spelling the word**. That is a SECOND measurement trap on #779, in the opposite
direction from the case-sensitivity one already recorded there, and **#1641 was
deliberately not PATCHed to satisfy it**: editing the evidence to make a check
look right is not a repair.

### Persona consultations — NONE folded this pass

**The cartographer role-plays no persona and does not spawn one** (#416): it has
read the source, so a verdict it produced would launder an insider's opinion as
an outsider's, which is worse than none. **No persona report was handed to this
pass, so nothing was folded** — stated rather than omitted, because a stamp that
silently drops a section is indistinguishable from one whose section came back
empty.

**A consultation is NOT owed this window, and for the first time that judgment
rests on a measured backlog rather than on a schedule.** Two consultations' worth
of findings are open, `ready` and entirely unconsumed: **#1593, #1594, #1595,
#1596** (2026-09-18, four `kind/story` — `parser`'s go-doc ordering, the
`value.Backend` goroutine-safety contract, the *"on the same grounds as [X]"*
idiom, `gen`'s flag diagnosis) and **#1626** (2026-09-21, the `-version`
error-path asymmetry, the only new filing from that pass's run), alongside
**#1568–#1571** from the twenty-seventh. **Nine open persona findings, zero
consumed by fifteen landings.** Running an eighteenth consultation against a
surface that has not changed for the personas would file a tenth, and the
constraint is evidently not the supply of findings.

### API- and CLI-facing routing, for the next consultation

- **M5 — Instance validation (XML)** is the one milestone that is BOTH. Its own
  `## Surface` names `validate.Validator`, `Result`, `Option`, the abstract
  infoset interfaces and `xmlsrc.Validate` — the second-largest exported surface
  in the project — and **5 of its 15 open issues also carry `area/cmd`** (#16,
  #1224, #1250, #1257, #1346). #1494 closed out of that overlap this window.
- **M4 — Schema parsing** is API-facing only: the exported `parser`/`xsd`
  surface, 54 open and unmoved for five stamps.
- **M6 — XPath required subset** is neither yet. Its one open issue is #1042
  (assertion evaluation) and no surface exists to consult about.
- **The CLI-facing work is NOT a milestone.** 35 open issues carry `area/cmd` or
  `area/cli` and most carry no milestone at all, so milestone counts cannot route
  a `cliuser` run. **The `cmd` contract family stands at TWELVE issues over two
  files** — #1407, #1549, #1521, #1381, #1368, #1419, #1421, #1318, #1569,
  #1570, #1571, #1626 — and #1407's five-row exit-code table is the structure
  four of the others want to hang a clause on.
- **42 open issues carry `kind/story`** and are where a persona report lands
  first; the library half is the subset in an `area/` other than `cmd`/`cli`.
- **`conformance` gained TWO exported identifiers this window** — `Catalog` and
  `CatalogEntry` (#1642) — read by `tools/casejoin`. They are infrastructure-tier
  and inside `docs/ARCHITECTURE.md`'s stated allowance, but they are the only
  surface addition of the window and a `libuser` run has never seen them.

### Working band

Ordered for a `/develop` session: take the highest row you can start.
**NO claim stands anywhere on the remote** — all thirteen `wip/*` refs are
RETIRED — so every row below is startable. **Re-run `wipsurvey` before starting
anything anyway**: this is a snapshot, and a landing arrived mid-pass in three of
the last five stamps.

**Every row names exactly ONE issue**, for the first stamp in four. #1636 filed
the observation that a two-issue row has no rule behind it and that the same-file
economy argument was spent by its own escape clause; both of last window's paired
rows landed as separate commits into files that were rebased anyway. Splitting
cost this band two rows of length and cost nothing else.

**The ordering principle this stamp:** the prediction chain is closed, so the
question the last two stamps could not answer — does the band pick MOVING work,
or only startable work — is finally askable. **Row 3 is that question**, and the
two rows above it are the instruments this pass's own figures came out of, one of
which has a live defect standing on the remote right now.

| # | issue | why here |
|---|---|---|
| 1 | #1627 | **The only row whose defect is visible from `main` today.** `claude/dazzling-cerf-u3xy2k` is `ahead=1` with the twenty-ninth stamp's 256-line LOG entry on it, PR #1632 unmerged for a day, and no survey reports the namespace. Arm 2 was filed as *"a judgment call with no defect behind it yet"*; that sentence stood for one day. Arm 1 is cheaper still — one refspec argument closing a chain of FIVE `parked/*` figures this section attributed to a tool that cannot compute them. `kind/tooling` banded on the sessions it costs, which is the rule |
| 2 | #1604 | **Both survey tools invert their verdicts on a feed missing `state` — ALL CLOSED in `gapaudit`, ALL OPEN in `wipsurvey` — and every figure in this stamp came out of them.** Confident wrong numbers, not missing ones: the feed stays well-formed JSON, passes the distinct-number stamp and survives `jq -s add`, so no defence in the Survey input recipe can reach it. Third instance of the silent-truncation class (#1145, #1520) and the first one INSIDE the tools. Rebases row 1's file — take them in band order, and #1627 first if only one goes |
| 3 | #1641 | **The band's falsifiable claim, and the first issue in the record whose `Ratchet:` prediction is DERIVABLE before mason is delegated.** `casejoin` exists and reproduced #1494's +13 exactly; this population is the near-complement of that one, in the same file, against the same built-in declaration. **Run the join at grounding and put the figure in the `PREDICTION:` block.** If it answers `not derived` on a population this close to one the instrument just measured, the defect is the instrument and the next stamp must say so. Warm — #1494 landed four commits ago. **Delete the marker, do not renumber it** |
| 4 | #1649 | **The record of the twenty-ninth pass is on an unmerged branch and `git log` cannot see it.** Carry `b9eaa76`'s `docs/LOG/2026-09.md` hunk onto `main` and NOT its `docs/PLAN.md` hunk, which this section has already superseded. Cheap, one file, and it stops being cheap the day that ref goes. Second sighting of the class — the eighth `/retro`'s `meta/audit-2026-09-06` half was the first |
| 5 | #1599 | **Compounding, and #1642 made it worse by exactly the amount it helped.** `CLAUDE.md`'s `suiteindex` paragraph is 39 unbroken lines teaching six query shapes and now two tools, each appended after the last. Every session that owes a `Ratchet:` prediction reads it — which, after rows 1–3, is every session. *"Length is a signal"* is the repo's own rule and it reads on this paragraph |
| 6 | #1610 | **Four sites in one file disagree about §3.4.6.3's (a)/(b)/(c) sentence after #499, and ONE OF THEM CITES TEXT THAT LANDING DELETED.** The worst class of comment defect: not imprecise but referring to something not there. Three landings rewrote `xsd/contentrestricts.go` this window, so it is warm, and the fourth disagreeing site was found by #499's own post-land pass rather than by the issue |
| 7 | #773 | **M5 `instance` lane, and half of the eighteen-slice carve's entire remainder.** String Valid clause 3 — every ENTITY value is a declared entity — at `matchedAttribute`'s `GAP(validate)`. The M5 section's own prose names #773 and #774 as what is left, and it has for three stamps |
| 8 | #774 | **The twin, and the only row here that wants a warden pre-flight** — its own `## Surface` demands one. Split from #773 rather than paired with it, per #1636; they touch one file and whichever goes second rebases it, which last window's evidence says costs nothing |
| 9 | #479 | **The only route to a `blocked` issue that does not mint a second resolution mechanism.** `<attributeGroup ref>` resolved at finalize via one ordered `AttributeUseOrGroupRef` sum; **#1458** (seeding `xml:specialAttrs`) is startable the moment it lands and says so in its own `## Depends on`. `schema` lane, M4 — which has not moved in five windows |
| 10 | #831 | **Discharges #871 on landing and is a real bug either way.** `produceAttribute` hardcodes `{inheritable}` FALSE on every global attribute declaration, so §3.12.4 clause 1.1.3's ·inherited attributes· merge would implement against a property that cannot be true. `kind/bug`, `area/parser`, and the cheapest of the four depth-1 discharges |
| 11 | #1619 | **A population that did not exist two stamps ago.** FOUR markers now carry P3b rulings, and a marker `matchRuled` retires has **nothing else in it audited** — a stale owner citation beside a valid ruling is silenced outright. P3b's last sentence is a prose control with no check, and the census this section publishes is what it silences into |
| 12 | #1647 | **Warm, cheap, and #1589 made it worse.** The one-path premise is derived in full at THREE sites in `contentmatcher.go` with three different citation sets, two of them added by last window's landing. STYLE D3 on a file four landings touched. Take it with nothing else; it is the last row and the shortest |

**Named below the band, deliberately.** **#1617** and **#1618** are the other two
`tools/gapaudit` issues — same file as row 11, and #1617 (an untested `(?i)` in
the SILENCING direction) is the one that matters. **#1613** and **#1624** are the
two warm comment repairs this window's landings left, and **#1616** (STYLE P3
carrying no pointer to P3b) is the third, now that four markers depend on the
distinction. **#1579** was row 11 last stamp and is still the `suiteindex`
package-doc twin of row 5. **#1499** is discussed in Next planning action 2 and is
deliberately NOT banded this time. **#1629**, **#1630** and **#1636** are the
three process findings this window's post-land passes filed about the band and
about re-scoped bodies; all three were exercised by this pass and all three are
commented. **#1572** owns the two unowned `GAP(xpath)` markers in `icpath`.
**#1576** is #1553's own open question and is the cheapest M5 `kind/gap` after
rows 7 and 8. **#591** and **#1324** each discharge exactly one `blocked` issue;
**#1324** is the cheapest, a ruling that discharges #1344 in either direction.
**#1536** and **#1153** are the two untested survey instruments and both compound
with rows 1 and 2. **#345**, **#1379**, **#1462**, **#1502** and **#1511** are the
fail-open bookkeeping family, `schema` up-or-flat; **#1156** is the same family
and half-discharged already. **#1473**/**#1474**/**#1476**/**#1477** are the
`regex` Appendix G family, four issues over two files. **#743** and **#744** are
the `src-redefine` pair. **#578** is the construction-cost twin of the ceiling
rulings that landed this window. **#888**, **#889** and **#1119** each want M6/M7
machinery that does not exist. **#1522** (the 400-line M4/M5 prose) and
**#1019** (`icpath` named in no document) are both compounding and both untouched
for five stamps. **#1546**, **#1548**, **#1429**, **#1539**, **#1518**,
**#1452**, **#1496**, **#1497**, **#1508**, **#1454** and **#1291** are each real
and each one session. **#1593**, **#1594**, **#1595**, **#1596**, **#1626** and
**#1568**–**#1571** are the nine unconsumed persona findings.

### Next planning action

1. **Row 3 is this band's falsifiable claim and it is the narrowest one yet.**
   The prediction chain is closed and checked against a landed figure. If #1641's
   `PREDICTION:` block answers `not derived` on a population this close to
   #1494's, the instrument does not reach the `instance` lane and the next stamp
   says what does instead. **If it answers a figure, this is the first banded row
   in the record ranked on predicted movement rather than on startability**, and
   the ordering question two stamps have carried is answered.
2. **The `kind/refactor` ranking test FIRED, and the answer narrows #1499 rather
   than lifting it.** The 2026-09-18 stamp set it explicitly: *"if row 10 lands
   and #1499 is still unconsumed at the next stamp, the reading is that the
   MEASUREMENT — not the ranking rule — is what a refactor needs."* **#1589
   landed** (`928ef25`, allocs/op −30%, reproduced by the arbiter) **on a cost of
   delay no steward wrote** — #1557's own landing benchmark measured it. #1499 was
   banded twelfth for the third stamp and was not taken. So a refactor was banded
   and landed without the ranking #1499 exists to install, and **#1499's body has
   been corrected to carry that counter-example and its refreshed figures** (82
   open refactors of 347, four bodies carrying `Cost of delay`, still zero reading
   `Increasing`). It is not banded this stamp; the next one should either take it
   on the narrowed scope or say why an issue three stamps have ranked last is
   still `ready`.
3. **M4 has not moved in FIVE windows** — 54 open, 23 lane-visible, 31 that move
   no lane, byte-identical across five stamps. Whether the remaining 31 (22 of
   them `kind/refactor`) belong in M4 at all, or whether M4 has become a directory
   of `parser`/`xsd` work that outlived the parsing milestone, is a **`/retro`
   question and no `/backlog` can answer it.** Carried for a FIFTH stamp and still
   the longest-carried item on this list. Row 9 (#479) is the only M4 row banded.
4. **TWO `/retro` cycles are now missed, not one.** The last retro landed
   2026-09-06 (`f97c317`); **2026-09-13 and 2026-09-20 both did not**, and
   2026-09-20 was a Sunday two days before this stamp, so it is owed rather than
   early. Sixteen days and fifteen landings without one. Items 2, 3 and the
   #845/#848 steward ruling under the queue audit are all routed there, and so is
   the persona trigger. Re-derived here from `git log -- docs/LOG/` rather than
   carried from prose, because a stamp has already regressed this count once.
   `docs/ROUTINES.md` owns the cron and nothing in this container can read whether
   the routine fired.
5. **A maintenance pass can now complete on GitHub and leave NO trace on `main`,
   and it has done so twice.** #1649 recovers the one stranded entry and #1627
   arm 2 is the survey that would have caught it on the day. **Neither closes the
   process question**: `.claude/commands/backlog.md:22-24` already says *"never
   leave the commit on an unmerged branch"*, so the rule is not missing and was
   broken anyway. Whether that wants a check or only a survey is a `/retro`
   question; this pass files no second issue for it, because #1627 route (a)
   supplies the detection and a duplicate remedy is what the filing discipline
   forbids.
6. **The persona trigger is `/retro`'s and this pass hands it a NEW datum: the
   CONSUMPTION rate, which is zero.** Nine persona findings from three
   consultations are open, `ready` and untouched by fifteen landings. The datum
   the last two stamps said to decide on — the REPAIR rate, seven folds of which
   four corrected a scope or a premise — is unchanged and is now the weaker of the
   two. A consultation whose output nothing consumes is not short of findings.
7. **The human decision blocking #1002 is unchanged and is carried for a
   TWENTY-FIFTH stamp.** It waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent, so no `/backlog` can
   move it and none should try.

**What this pass did, so the next one can tell a completed run from a skipped
one.**

**ONE issue filed, ZERO closed, ZERO label changes, ZERO relabels.** **#1649** —
the twenty-ninth pass's stranded `docs/LOG` entry, filed rather than handed on,
because a hand-off tracks nothing (#330). Nothing else was owed: the marker
census found no untracked gap whose annotations name no owner, the branch sweep
found nothing to retire that is not already retired, the unblock sweep measured
zero, and the twenty-two issues this window's landings and post-land passes
filed were checked — all twenty-two carry every required section, so class 6 is
still the same four bodies it has been for three measurements. **The `claude/*` namespace finding
was deliberately NOT filed as its own issue** — #1627 arm 2 already owns the
detection and its body was corrected instead.

**THREE body PATCHes and ZERO title PATCHes.** **#1627** — arm 2 rewritten from a
trend to a defect: the ref census refreshed from seven to nine with
`claude/dazzling-cerf-u3xy2k` at `ahead=1`, the filing hedge *"a judgment call
with no defect behind it yet"* retired, route (a)'s acceptance made checkable
against that ref, and arm 1's stamp count corrected from four to five.
**#1624** — its `Ratchet:` bar restamped from `instance 11209/15152` at `89167f1`
to `11222/15139` at `da0eb95`, with the filed figures kept as history and the
reason named (#1494's +13 landed between the readings, so the old bar would read a
real landing as something riding along). **#1499** — figures refreshed to the
2026-09-22 walk beside the 2026-09-14 ones as a trend, the fourth `Cost of delay`
body (#1333) that its first measurement missed, and **#1589 added as the second
landed counter-example**, which narrows its second done-condition.

**FIVE thread comments** — **#779** (classes 6 and 7 re-measured with both
denominators, and a SECOND measurement trap recorded: `(?i)ratchet` over-reports
by two because #785 and #1641 state a bar without the word), **#1630** (all four
in-band correction paragraphs cleared by this replacement, the rule question
untouched, and one new datum that cuts FOR the habit), **#1636** (this band names
one issue per row and the splitting cost nothing), **#1641** and **#1604**
(banding rationale, and the one constraint each carries into its landing).

**Every body PATCH and comment went through `gh api -F body=@FILE`** and is
byte-faithful; every body READ came from repository-scoped REST rather than the
MCP channel (#764). **`gapaudit` and `wipsurvey` were each run once, on `main` at
`da0eb95` after `git fetch --unshallow origin`; ONE `state=all` walk, clean on
the first request, stamped `rows 1648 distinct 1648 min 1 max 1648 no gaps`.**
**`testdata/xsdtests` was NOT initialized**, so no suite figure here is this
pass's own measurement — the lane table is the committed expectations census,
which needs no submodule, and the seventeen flips were read from
`git diff a2a72bf da0eb95 -- conformance/testdata/expectations/`.

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
and **#1516** for the `{open content}` clauses. **SIXTEEN of the eighteen have
landed, and the whole remainder is #773 and #774** — #1516, #782 and #783 all
landed 2026-09-16/17 and took `instance` 11151 → 11203. Of the thirteen that had
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

**#913 added the second class.** Seven CTA documents are false-charged through `cvc-type` clause 3.1 until
§3.12.4's `{inherited attributes}` merge lands (#831, #871) — an
honest-decline-to-wrong-decision trade the ratchet's zero-flip-down cannot
register, escalated on #831's thread.

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
issue had already closed the same day. **#871** is the §3.12.4 clause 1.1.3
·inherited attributes· merge, blocked on M4's #831. None of the five carries a
milestone, which is the same pattern M4's tail records.

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
