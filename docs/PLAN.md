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

## Status — 2026-09-17 (`/backlog`, the twenty-seventh)

**The twenty-sixth stamp's band made a falsifiable claim and the window
CONFIRMED it.** That band was rebuilt around the ten `ready` issues whose
`Ratchet:` clause could move a lane, after a seven-landing window in which none
moved. Its rows 1–4 — #1516, #782, #783, #1378 — **were all taken and all
landed**, in four landings, and `instance` banked **11151 → 11203 (+52)**. Row 5
(#812) is the only LIVE claim on the remote as this is written. The preceding
window had rows containing zero lane-movers and moved zero; this one had four at
the top and moved all four. **That is the strongest evidence this record holds
that the band's ORDERING is the deliverable and not decoration.**

**The counter-finding is the same window's filings, and it outranks the first.**
Four consecutive post-land passes filed **five issues all against one thing: the
ratchet-prediction chain does not work.** #1552 (no `/develop` round owns turning
a population BOUND into a lane PREDICTION), #1554 (`suiteindex` censuses one
element name per query, so #1516's hand-taken bound **missed twelve
`defaultOpenContent` fixtures, six of them in the flipped set**), #1561 (the join
admits candidates the `instance` lane can never observe — 387 lines, zero banked
pass), #1545 (neither statement of the join rule names `GOXSD_WITHHELD=1`) and
#1536 (`suiteindex` answers a confident 0 on a non-NCName local name). The
twenty-sixth stamp's own next-action item 2 asked this stamp to check *"that a
prediction has actually been made through the chain, not merely that the chain
exists."* **The answer is NO, and #1516 is the proof: it was the first issue
positioned to use the chain, it took its bound by hand instead, and the bound was
wrong.** `/backlog`'s banding rule puts `kind/process` and `kind/tooling` work on
the sessions it costs, and this is friction the log records in four consecutive
sessions — so rows 2, 3 and 5 below are that bundle, interleaved with the lane
slices that pay the tax.

**This section is deliberately about a third the length of the one it replaces.**
Its reader is a `/develop` session choosing an issue. Provenance belongs in
`docs/LOG`; the measurement that the replacement rule bounds this section and
nothing else in this file is on #1522's thread, where the issue that owns the
unbounded part can use it.

### Conformance lanes

**Paste `go tool lanestatus` verbatim, never a hand-count.** This table is
`main` at `e064471`, and it is the committed expectations census —
`conformance/testdata/expectations/<lane>.txt`'s `pass`/`fail` line counts —
which `docs/WORKFLOW.md` names as the lane score (#1120) and which is the only
lane figure a later session can re-derive from the committed tree alone.

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11203 | 15158 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14692 | 706 | 15398 |
| `xpath` | — | — | 0 |

`instance` **11151 → 11203 (+52)** this window: #1516 +41, #782 +8, #783 +3.
`schema` **14692, unchanged** — #1378 ruled a ceiling a permanent documented
approximation and moved nothing by construction. `datatypes` unchanged.
`datatypes ⊆ instance` is INTENDED (#1507 ruled arm A against outright), so
`instance` 11203/26361 is a figure to quote and not to hedge.

**An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero.** `datatypes` is M3 and complete; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**The column does not sum to the corpus, and that is SETTLED.** The three lanes
hold 41759 distinct case IDs against a column summing to 42932, because every
one of the 1173 `datatypes` cases is also committed in `instance.txt`, 701 of
them with a contradictory verdict — two differently mature executors over one
fixture. #1507 ruled that overlap INTENDED and ruled arm A (disjoint lanes)
against outright, since it would move 468 committed `instance` passes downward.

**No conformance measurement was taken this pass**: `git submodule status`
reads `-7bc3365…`, uninitialized. Every suite figure below is cited from the
landing that measured it, and the `7bc3365` pin is read off the gitlink rather
than off a checkout.

### Branch namespace, `origin` — report-only; a session never deletes a ref

`go tool wipsurvey` after `git fetch --unshallow origin`, run twice, second run
after this pass's writes. **Thirteen `wip/*` refs, zero `parked/*`.**

- **ONE LIVE claim: `wip/issue-812`**, heartbeat 3 minutes old at the second run.
  #812 and its branch are off-limits, which is why #812 is not banded below
  although it was row 5 last stamp.
- **TWELVE RETIRED**: #434, #732, #822, #846, #872, #933, #968, #993, #1332,
  #1356, #1426, #1451 — the same twelve as the last two stamps, no new ones.
- **Every retired branch's content is accounted for and NOTHING is owed.**
  Eleven of the twelve issues closed `not_planned`, so the branch is the
  re-planning evidence the label retires in place. The twelfth, **#1332, closed
  `completed`** and its ref survived the merge, 14 commits ahead with a 14-line
  diff in `.claude/agents/arbiter.md`: **that content IS in `main`, in a
  superseding form** — the paragraph landed and #1513 then folded the
  expectations-file join into it. No supersession issue is owed.
- Two RETIRED rows still print a LEASE AGE borrowed from `main`'s tip. That is
  **#1548**, still open, and it is cosmetic here because the verdict does not
  read it.

### Marker census

`go tool gapaudit` over the whole repository feed, run twice, **identical row for
row**: **69 markers, 9 areas, group 1 at 19, group 2 at 31.** Down from 71 —
#1516, #782 and #783 deleted declines as they landed.

- **ONE dead-end row, and it now cites TWO closed issues**:
  `xsd/contentrestricts.go:684` names CLOSED #501 and now also CLOSED
  **#1378** — the issue whose landing RULED the ceiling permanent. **#1565** owns
  exactly that shape (STYLE P3 has no form for a marker whose gap was ruled
  permanent) and **#1564** owns the same two markers' missing P3a consumer
  enumeration. Both `ready`, both filed by #1378's own post-land pass, and they
  are one session.
- **Group 1 gained one row, and it is a bookkeeping artefact rather than an
  untracked gap**: `validate/assess.go:428`, filed this window as **#1553**,
  whose `xsd` half IS cited at `xsd/contentmatcher.go:151` while the `validate`
  half names no number. So one gap appears in group 1 and in group 2 at once.
  **Recorded in #1553's own `## Notes` by this pass** so the next one does not
  re-investigate it; both rows retire when the arm lands.
- No other group-1 row is newly untracked, and no new `kind/gap` issue is owed.

### Milestones and queue

**THREE full page-numbered `state=all` REST walks, all clean on the first
request** — before this pass's writes, after them, and after the persona fold.
The third is the one quoted here, stamped
**`rows 1571 distinct 1571 min 1 max 1571 no gaps`**: **1571 rows, 704 PRs
excluded, 867 issues — 339 open, 528 closed.** The first two read
`rows 1566 distinct 1566 … no gaps` and **335 open**; the difference is this
pass's own four filings plus one landing, and it is stated rather than smoothed
over so a reader can see which walk any figure below came from.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **134** | active |
| **M5 — Instance validation (XML)** | **17** | **28** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **325 `ready`, 12 `blocked`, 0 `needs-replan`,
2 `epic`** — summing to 339, and **every open issue carries exactly one,
verified over all 339 by grouping each issue's own label set rather than by
summing counts**. #779 owns the mechanical check.

By kind: `kind/refactor` 81, `kind/process` 67, `kind/gap` 59, `kind/tooling`
45, `kind/story` 37, `kind/bug` 34, `kind/docs` 31, `kind/feature` 3. By area:
`meta` 96, `parser` 83, `xsd` 64, `docs` 46, `conformance` 37, `cmd` 33,
`validate` 28, `value` 17, `builtin` 11, `xsderr` 9, `regex` 6, `xpath` 6,
`model` 4, `loader` 2, `cli` 1. **267 of 339 open issues carry no milestone at
all**, so read the `area/` census, not the milestone counts.

**M4 did not move at all this window** — 54 open, 134 closed, byte-identical to
the last two stamps — because none of the four landings carried it. **M5 took
all four's weight**: 25 → 28 closed, open unchanged at 17. Of M4's 54 open, at
most **23 are lane-visible** (15 `gap`, 7 `bug`, 1 `feature`) against 31 that
move no lane by construction; of M5's 17, at most **11** (8 `gap`, 2 `bug`,
1 `feature`).

**`ready` 317 → 325.** Four closed with the landings; eight were filed by their
post-land passes (#1552, #1553, #1554, #1557, #1560, #1561, #1564, #1565) and
**four by this pass's persona fold** (#1568, #1569, #1570, #1571). A net of +8 on
a turnover of sixteen is #347's shape: `ready` is an output, not a target.

**The unblock sweep measured ZERO**, over all twelve `blocked` bodies'
`## Depends on` sections read byte-faithfully from REST. **No entry names #1516,
#782, #783 or #1378.** The set is unchanged at twelve, and its shape is the one
the last three stamps recorded: **five are triggers rather than issues** and say
so (#1374, #1224, #555, #1002, #1042 — **do not re-scan these**); **two state in
their own bodies why every closed dependency is not a discharge** (#16, #1051);
**five have exactly one open named dependency each, and all five targets are
`ready` today** — #593←#591, #731←#1414, #871←#831, #1344←#1324, #1458←#479.
**The graph is depth-1, no cycles, and no `blocked` body names an epic.**

**The `ready` audit found ZERO queue mislabels and EIGHT defective bodies.** It
was run over the **321** `ready` issues standing at the second walk, before this
pass's four filings — which is why its denominator is 321 and not the 325 above,
and the four new bodies were checked separately and carry all six sections. All
321 carry a `## Depends on`; 63 name an open `#N` inside it under an explicit
"none"/"n/a" opener with a not-gating clause, and the nine that appeared to name
one otherwise were all bodies quoting `` `## Depends on` `` in prose — checked
one by one against REST, all nine clean. **What the audit DID find is new**:
**7 of 321 `ready` bodies drop a section the issue template requires** (#586,
#743, #773, #774, #845, #848, #1419) and **8 of 85 lane-visible `ready` bodies
carry no `Ratchet` token at all** (#584, #743, #753, #773, #774, #785, #1049,
#1419). A missing `## Surface` is not tidiness: `/develop` step 3 rules that
section against the tree to decide whether a warden pre-flight is owed, and an
absent section reads exactly like a ruled `none`. **Filed as a sixth and seventh
class on #779** rather than beside it, with the table. **#773 and #774 were
repaired here** — they are the M5 carve's whole remainder and will be picked.

### Persona consultations — the TWENTY-SEVENTH, and TWO findings in eleven had a FALSE premise

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **The orchestrating session ran both personas fresh against the
published surface** — `libuser` on README plus `go doc` only, `cliuser` on README
plus `-help` plus a binary built from the tree — and handed the reports here.
**Every finding was re-derived against the tree before disposition.**

**Eleven findings: FOUR filed, SEVEN folded into the issue that already owned
them.** Filed: **#1568** (README invites an application author without a schema
document to the component-model constructors and fences it one sentence later),
**#1569** (a repeated identical `-schema` is silently collapsed while two
byte-identical files under different names collide at `sch-props-correct`),
**#1570** (rule on subcommand-scoped help), **#1571** (`-help` describes exit 4
abstractly and names no construct class, so a binary-only operator cannot predict
permanent exit 4 on `xs:assert`). Folded: **#1005**, **#1283**, **#1315**,
**#1292**, **#405**, **#1381**, **#1521**, plus a design-input comment on
**#1553** and reciprocal cross-references on **#1434**, **#1433**, **#1089**,
**#1407** and **#405**.

**The finding of this consultation is that TWO findings rested on premises that
are false against the tree, and both were corrected rather than filed as
reported.** This is new in this record and it is the reason a persona report is
re-derived rather than transcribed:

- `libuser` reported that README *"explicitly tells a plain library consumer to
  reach for `SchemaBuilder`"*. **README says the opposite** — *"That builder is
  PRODUCER surface, not application surface"*, `README.md:292`. **#1568 states
  the contradiction is not there** and files the defect that IS: the invitation
  is sentence 1, the fence is sentence 2, and the reader the invitation catches
  is addressed by no sentence after it. The persona is first-hand evidence for
  the arrival order, which is the one thing an insider cannot supply.
- `cliuser` reported that the `xs:assert` fail-open pointer *"lives only in the
  library README section, not in `-help`/`doc.go`'s validate section"*.
  **`cmd/goxsd8/doc.go:177-182` names it directly.** **#1571 corrects the premise
  and keeps the real defect**, which is narrower and is #1279's shape: `doc.go`
  names the construct classes, `main.go`'s `usage` — what the binary prints — does
  not.

**That is the THIRD consecutive consultation whose real finding was a SITING
error rather than a wrong statement** (#1521's arm (a), #1549's *"not
well-formed"* sentence, now #1571), and a **fourth** consecutive one finding no
accuracy bug in the CLI contract: `cliuser` verified `-help`, `go doc
./cmd/goxsd8` and README's CLI section **word-for-word identical**, and every
exit code, stream and message shape it probed matched. **Whether the persona
trigger should change is `/retro`'s to decide**, and the datum to decide it on is
now the *repair* rate — seven folds, of which #405, #1381, #1521 and #1292 each
had a scope or a premise corrected by what the outside view saw.

**One fold is a correction of a WARDEN ruling, not of a body.** #405 carried the
warden's cleared rationale that *"the mis-order hazard is same-typed adjacency,
and there is none"*, and its roster lists only adjacent pairs. The persona
tripped on **same-typed NON-ADJACENT `bool` parameters** —
`NewElementDeclaration` positions 7 and 11 (`nillable`, `abstract`) and
`NewAttributeUse` positions 2 and 5 (`required`, `inheritable`), both read from
`go doc ./xsd` — which transpose and type-check. #405's fence against
pre-emptive conversion is untouched; what changed is what its comment must say.

### API- and CLI-facing routing, for the next consultation

- **M5 — Instance validation (XML)** is the one milestone that is BOTH. Its own
  `## Surface` names `validate.Validator`, `Result`, `Option`, the abstract
  infoset interfaces and `xmlsrc.Validate` — the second-largest exported surface
  in the project — and **6 of its 17 open issues also carry `area/cmd`** (#16,
  #1224, #1250, #1257, #1346, #1494).
- **M4 — Schema parsing** is API-facing only: the exported `parser`/`xsd`
  surface, 54 open.
- **The CLI-facing work is NOT a milestone.** 33 open issues carry `area/cmd` or
  `area/cli` and most carry no milestone at all, so milestone counts cannot route
  a `cliuser` run. **The `cmd` contract family is now ELEVEN issues over two
  files** — #1407, #1549, #1521, #1381, #1368, #1419, #1421, #1318, #1569, #1570,
  #1571 — and #1407's five-row exit-code table is the structure three of the
  others want to hang a clause on.
- **40 open issues in a library area carry a non-empty `## Surface`**, and 37
  carry `kind/story`. Those two sets are where a `libuser` report lands.

### Working band

Ordered for a `/develop` session: take the highest row you can start.
**`wip/issue-812` is LIVE** — re-run `wipsurvey` before starting anything, and do
not take #812.

**The ordering principle this stamp:** the lane-first rebuild worked and is kept,
but the prediction chain that every lane slice depends on is measurably broken,
so rows 2, 3 and 5 are that bundle and are ranked on the sessions they cost
rather than on the lane they do not move.

| # | issue | why here |
|---|---|---|
| 1 | #1553 | **The direct continuation of #1516 and the largest remaining `instance` slice.** A third arm on the exported sealed sum `xsd.Attribution`, so `{open content}` admission stops spelling itself as a bare `Wildcard` and `validate` can stop withholding e-validity clause 1.1.3 for the wildcard-PARTICLE case. Its body already commits to stating the `Ratchet:` as a PREDICTION before mason is delegated, which is what rows 2 and 3 exist to make honest. Retires the group-1 marker at `validate/assess.go:428` and the group-2 row for itself, together. **Warden pre-flight — this adds exported surface** |
| 2 | #1554 | **The cheapest fix to the chain, and the only one whose cost is already MEASURED.** `suiteindex` censuses one element name per query, so a population defined by a FEATURE is undercounted: #1516's bound missed twelve `defaultOpenContent` fixtures and six of them were in the set that actually flipped. Every row above and below pays this until it lands |
| 3 | #1552 | **The process half of row 2, and they are one sitting.** `/develop`'s sequence never invokes #1512's join or #1514's `GOXSD_WITHHELD=1`, so no round OWNS turning a bound into a prediction — #1516 proved it by not doing it. Naming the round is the whole change |
| 4 | #1557 | **The WIDTH #782 left declined**, on the same decline list and the same lane: a nested repetition whose clamped occurrence product exceeds `maxPartitionStates`. `Ratchet: measure, do not predict, and take the prediction through the chain` — so it is the first issue that can exercise rows 2 and 3 once they land, and the first honest test of whether they worked |
| 5 | #1561 | **The join's `instance` arm, and it deletes work rather than adding it.** That lane observes only "not valid", so every suite-VALID case is dead weight in every `instance`-lane bound — 387 lines carrying zero banked pass. A `CLAUDE.md` edit; finishes the bundle rows 2–3 start |
| 6 | #1102 | **M4, and the one row here whose `Ratchet:` clause promises a MEASURED figure.** `ownAttributeUses` is a proven over-approximation since #1082, not a recovery, and the unowned `GAP(xsd)` at `attributeusefold.go:296` is **fail-CLOSED** at `checkAttributeRestrictionRequired` — the direction that costs a rejection rather than withholding one. Row 6 last stamp and unconsumed. Read **#1539** first |
| 7 | #499 | **#1378's twin, and #1378 just landed the template for ruling it.** The `maxProductStates` ceiling: route (a) retires it by a containment procedure, route (b) rules it a documented permanent approximation on measured cost. #1378 settled that route (a) is *not a spec question at all* (§G.1.6 records XSD 1.1 deliberately removing its restriction-checking appendix), so the argument is written and this is the cheapest ceiling ruling the queue will ever hold. Its body was corrected by #1378's post-land pass |
| 8 | #1565 + #1564 | **The tree's ONLY dead-end marker row, and both owners are one session.** `xsd/contentrestricts.go:684` cites two CLOSED issues; #1565 owns the general shape (a permanently-ruled gap has no STYLE P3 form) and #1564 the same markers' missing P3a consumer enumeration. Taking them separately rebases one file twice |
| 9 | #1494 | M5: an unresolvable `xsi:type` exits 0, silent. **Oracle first** — do not implement a charge before the rule ID exists, and *"the spec charges nothing"* is a legitimate outcome. **Banded three times without being taken**; its `Ratchet:` clause says the prediction is not derivable with the tools in the tree, which is exactly what rows 2 and 5 are about, so it is better after them than before |
| 10 | #1545 | **The last piece of #1451's re-plan**, and the bundle's tail: the join rule is stated in two places and neither names `GOXSD_WITHHELD=1`. One issue where #1512 and #1513 were two, because those decided what the rule SAYS and this only points at the instrument twice |
| 11 | #1414 | **The smallest landing that retires a whole ledger.** `substitutionGroup=` naming an ABSENT component or a non-element is accepted — src-resolve's *"resolves to an element declaration"* condition is unenforced. It is the last open carve of **#731**, whose own body says there is nothing left there to implement, so the post-land pass can close that ledger too |
| 12 | #1499 | **Named below the band twice, banded twelfth once, and still nothing has lifted it.** It ranks `kind/refactor` — now **81 open, the largest kind in the queue** and grown by one this window (#1564) — and it is the only issue that can, because a refactor moves no lane by construction and costs no per-session friction to compound |

**Named below the band, deliberately.** **#773** and **#774** are the M5 carve's
entire remainder and both had their bodies repaired here (a `## Surface` and a
`Ratchet:` bar each); they are `instance`-lane work and belong in the next band
if they are not taken from here. **#591**, **#831**, **#1324** and **#479** each
discharge exactly one `blocked` issue on landing and none moves a lane;
**#1324** is the cheapest, a ruling that discharges #1344 in either direction.
**#1536** matters more than its size suggests now that `suiteindex` is the
*prescribed* prediction instrument. **#345**, **#267**, **#1379**, **#1462** and
**#1511** are the fail-open bookkeeping family, `schema` up-or-flat.
**#1473**/**#1474**/**#1476**/**#1477** are the `regex` Appendix G family and
are four issues over two files. **#888**, **#889** and **#1119** each want M6/M7
machinery that does not exist. **#1522** (the 400-line M4/M5 prose) and
**#1153** (the survey recipe has no test) are both compounding and both
untouched for three stamps; #1522 gained a measurement this pass confirming its
own scope ruling. **#1560**, **#1502**, **#1531**, **#1546**, **#1548**,
**#1429**, **#1539**, **#1518**, **#1452**, **#1496**, **#1497**, **#1508**,
**#1454** and **#1291** are each real and each one session.

### Next planning action

1. **Rows 2, 3 and 5 are this band's falsifiable claim.** If the chain bundle
   lands and the next lane slice still takes its bound by hand, the defect is
   not the tooling and the next stamp says what it is instead. **Row 4 (#1557)
   is the test**: it is the first lane slice positioned to use the chain after
   the bundle, and its own body already says to take the prediction through it.
2. **The band's lane-first rebuild is VALIDATED and should not be re-argued.**
   Rows 1–4 of the twenty-sixth band were taken in order and all four landed,
   moving `instance` +52 after a seven-landing flat window. Two stamps now
   measure the same thing in opposite directions, which settles it.
3. **M4 has not moved in two windows** — 54 open, at most 23 lane-visible, 31
   that move no lane, byte-identical across three stamps. Whether the remaining
   31 (22 of them `kind/refactor`) belong in M4 at all, or whether M4 has become
   a directory of `parser`/`xsd` work that outlived the parsing milestone, is a
   **`/retro` question and no `/backlog` can answer it.** Carried for a third
   stamp.
4. **The survey channel's failure rate has a THIRD data point and the question
   is closed.** The twenty-fifth stamp took four walks and two came back
   corrupt; the twenty-sixth took four clean; this one took **three, all clean on
   the first request**, with #1520's page-content assertion in the recipe
   throughout. **Read the 25th as an incident, not a rate**, and keep quoting the
   distinct-number stamp wherever a survey figure is quoted.
5. **The persona trigger is `/retro`'s to decide, and this pass changes the
   datum it should be decided on.** Four stamps have now measured the
   filed-versus-re-found ratio; this one measured something better. **TWO of
   eleven findings rested on premises that are FALSE against the tree** (README
   does say "PRODUCER surface, not application surface"; `doc.go:180` does name
   `xs:assert`), and both produced a real, narrower issue once re-derived — so
   **a persona report is an input to be checked, never a filing to be
   transcribed**, and the value of the consultation is not measurable by how many
   findings were new. **Decide the trigger against the REPAIR rate**: seven folds,
   four of which corrected a scope or a premise in the issue they re-found.
6. **The human decision blocking #1002 is unchanged and is carried for a
   TWENTY-THIRD stamp.** It waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent, so no `/backlog` can
   move it and none should try.
7. **No `/retro` has landed since 2026-09-06 and that is now TWO missed cycles.**
   2026-09-06, 09-13 and — as this stamp is written — the 09-20 run is still
   ahead. Items 3 and 5 are both routed there. This is an observation about the
   schedule, not a filing: `docs/ROUTINES.md` owns the cron and nothing in this
   container can read whether the routine fired.

**What this pass did, so the next one can tell a completed run from a skipped
one.**

**FOUR issues filed, ZERO closed, ZERO label changes** — and all four came from
the persona fold, not from the surveys. **The surveys themselves turned up no
untracked gap, no stale issue and no duplicate**: every defect the marker census,
the branch sweep and the queue audit found already had a live owner, the unblock
sweep measured zero, and the queue audit found no mislabel. The four are
**#1568**, **#1569**, **#1570** and **#1571**, all `ready`, all carrying the six
required sections, and each naming its adjacent issues with the reciprocal
cross-reference posted.

**TWELVE body PATCHes.** From the surveys: **#779** widened with a sixth and
seventh hygiene class and the measured table; **#773** and **#774** each given
the `## Surface` and the `Ratchet:` bar they were filed without (and #774 a
`## Spec`); **#250**'s three stale premises corrected — its "five unlanded
slices" list named three issues that landed this window, its milestone count, and
a present-tense `instance` figure of 1337; **#1553** given the `## Notes` bullet
recording why `gapaudit` reports it in two groups at once. From the persona fold:
**#1005**, **#1283**, **#1315**, **#1292**, **#405**, **#1381**, **#1521** and
**#1318**, of which **#405**, **#1381**, **#1521** and **#1292** each had a scope
or a premise corrected by what the outside view saw.

**SEVEN thread comments** — **#1522** (fourteen stamps of section-length data
corroborating its own scope ruling), **#1553** (design input for the third arm),
and reciprocal cross-references on **#1434**, **#1433**, **#1089**, **#1407** and
**#405**.

**Two statements in this section were FALSE when first written and are corrected
here rather than left standing**: it said the persona consultation had not run
and that zero issues were filed, both true at the moment the section was drafted
and both false once the reports arrived. **A stamp that misreports its own
actions is the #400 failure mode**, and the queue counts above were re-read from
a THIRD walk rather than carried forward.

**Every body PATCH and comment went through `gh api -F body=@FILE`** and is
byte-faithful; every body READ came from repository-scoped REST rather than the
MCP channel (#764). GraphQL is still a 403, exactly as `docs/ROUTINES.md`
records. **`gapaudit` was run twice and is identical row for row; `wipsurvey`
twice; THREE `state=all` walks, all three clean on the first request.**
**`testdata/xsdtests` was NOT initialized**, so no suite figure here is this
pass's own measurement — the lane table is the committed expectations census,
which needs no submodule, and every CLI behaviour quoted from the `cliuser`
report is marked as that persona's reproduction rather than this pass's.

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
PRINCIPLES 14: #782 replaced the greedy walk with a bounded set of live
partitions, which `cvc-accept` clause 3.1's existential requires, and the
invariant that survives is no-backtracking rather than greed.

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
