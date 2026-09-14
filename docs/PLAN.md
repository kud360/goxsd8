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

## Status — 2026-09-14 (`/backlog`, the twenty-fourth. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against `git fetch --unshallow origin` plus `git ls-remote --heads`, the marker census a fresh `gapaudit` run TWICE — before and after this pass's writes — and the milestone and queue counts a page-numbered `state=all` fetch taken **after** them. **The window is SEVEN landings and ONE lane moved, by +615** — `schema` 14075 → **14690**, with **609 of it ONE landing and 601 of that one corpus** (#1411's `MS-Regex2006-07-15`) — a larger `schema` move than #1126's +475, and the M4 scope paragraph is corrected below to say so. `instance` and `datatypes` are flat. **The event is that a circular dependency was broken**: #248 and #717 had each named the other's precondition since 2026-07-25, so the `instance` lane's largest named lever could never start; **#717 is now `ready`** and is band row 4. **#1426 is closed `not_planned` and replaced by #1498**, the park the 2026-09-14 log recorded as owed and undone. **The marker census grew 69 → 73 and gained a NINTH area, `regex`** — #1411 landed two markers and their four trackers in one window — while **group 1 held at 18 before AND after this pass's writes** and the four `dead end:` annotations across three markers held too. The **TWENTY-FOURTH persona consultation**: **six findings — three filed (#1494, #1496, #1497), two dismissed on falsified premises, one recorded as corroboration** — plus two issues this pass found while grounding them, #1495 and #1499)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11044 | 15317 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14690 | 708 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This table is `main` at `5425f92` and nothing stands outside it.** No `wip/`
branch carries a banked expectation this pass could find; the two live branches
(`wip/issue-1451`, `wip/issue-1491`) are both `kind/tooling`/`kind/bug` work on
`tools/`, which no lane can see. The number above is the committed expectations
on `main`, which `docs/WORKFLOW.md` names as the lane score (#1120).

**`schema` is now 95.4% and its remaining 708 failures are the whole of M4's
lane-visible work.** That is the first stamp in this record where the schema
lane's residual is smaller than the number of open issues, and it is the
argument for reading the `area/` census below rather than the milestone counts.

### The window: seven landings, `schema` +615, and one corpus explains 601 of it

| commit | issue | ratchet |
|---|---|---|
| `cd36ad8` | **#1428** — the XML namespace's built-in attribute declarations for an explicit `<import>` | **`schema` +2** (addC001, isDefault078) |
| `f996bd4` | **#1446** — `no-xsi` on an attribute declaration in the `xsi` namespace | **`schema` +3** (attKa015, attKb018a, wild041) |
| `3852d54` | #1443 — bound the loss of an in-flight mason round to a container restart | unchanged |
| `1942341` | **#1411** — charge `src-pattern-value` eagerly at schema construction | **`schema` +609** |
| `b3b2537` | #1437 — `wipsurvey` names the remedy when an empty-diff `TAKEOVER` count repeats | unchanged |
| `a1b4fd5` | #407 — remove the `{annotation}` machinery, option (b) part 1 | unchanged |
| `23b7ad1` | **#1413** — charge the four s4s grammar faults, bank `stC011` | **`schema` +1** |

**#1411's +609 is five cases of its own cohort and 604 it did not predict**, and
its commit body says so: 601 `MS-Regex2006-07-15` fixtures carrying Perl-only
constructs Appendix G has no production for, plus three other invalid-pattern
fixtures. That is **one mechanism banked whole** (PRINCIPLES 22) rather than a
prediction beaten — the opposite failure mode from the four consecutive
under-predictions before this window, and it should not be read as a fifth exact
prediction. **The three that did predict exactly are #1428 (+2, named cases),
#1446 (+3, named cases) and #1413 (+1)**, which continues the run the last stamp
opened.

**#1411 also created a new `area/`**: `regex` had no open issue and no `GAP(`
marker before this window and now has six open issues (#1473, #1474, #1476,
#1477, plus #989 and #849 re-tagged) and two markers, each with its tracker
filed in the same landing. That is the shape the last stamp named for #1427/#1446
and it has now happened twice.

**Four of the seven were band rows** (#1446 was row 2, #1411 row 4, #407 row 6,
#1413 row 7) and **#1443 was row 1**, so five of the twelve banded rows landed in
two days. **The one held claim from the last stamp, `wip/issue-1428`, landed** —
two for two on banded held claims.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against `git fetch --unshallow origin` and
this pass's **835-issue post-write feed**:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
434    wip/issue-434   74h20m0s   RETIRED  wip/issue-434: issue #434 is closed
732    wip/issue-732   525h14m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   703h12m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   475h34m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   669h14m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   main's     RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   main's     RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   480h54m0s  RETIRED  wip/issue-993: issue #993 is closed
1332   wip/issue-1332  53h28m0s   RETIRED  wip/issue-1332: issue #1332 is closed
1356   wip/issue-1356  28h30m0s   RETIRED  wip/issue-1356: issue #1356 is closed
1426   wip/issue-1426  1h33m0s    RETIRED  wip/issue-1426: issue #1426 is closed
1451   wip/issue-1451  2h0m0s     EXPIRED  wip/issue-1451: tip pushed 2h0m0s ago, past the 2h0m0s claim TTL
1491   wip/issue-1491  47m0s      LIVE     wip/issue-1491: tip pushed 47m0s ago, within the 2h0m0s claim TTL
```

**A row changed verdict WHILE this pass ran, and that is the stamp's own
demonstration of #1460.** The pre-write run read `1451 … 1h45m0s LIVE`; the
post-write run reads `2h0m0s EXPIRED`. Nothing about the branch changed — the
lease simply crossed the TTL between two reads forty minutes apart. **A
`wipsurvey` verdict is a snapshot and this table is already one.** Re-run it
before starting anything.

**`wip/issue-1426`'s verdict changed for a different reason, and it is this
pass's write**: pre-write it read `RETIRED … labelled needs-replan`, post-write
`RETIRED … is closed`. Same verdict, different route, because this pass closed
the issue `not_planned` and filed **#1498** in its place.

**ELEVEN RETIRED rows, and the newest three are all deliberate preservations.**
`wip/issue-1426` is at `a8ddcbd` and carries `eeef230` (the park checkpoint),
`98c7ec8` (the round-2 commit the arbiter judged) and `f0eb394` (round 1) as
ancestors — **#1498 says explicitly to read it and not build on it**. The
2026-09-14 log recorded the tip as `eeef230`; that is true of the content and not
of the ref, which moved when the chronicler's entry was merged onto it. No
supersede is owed for any of the three: #1356 → #1471, #1426 → #1498, and
`wip/issue-1332`'s content was verified present on `main` at the last stamp.

**Ancestry was re-measured, not inherited.** `wip/issue-933` and `wip/issue-968`
ARE ancestors of `main`; the other nine are NOT, which is the expected shape for
squash-merged and superseded branches alike. Tips run 2026-08-16 to 2026-09-14.

**Zero `needs-replan` in the whole queue for the first time in this record**, and
zero `parked/*`, zero `meta/*`. **FIVE non-`wip` `claude/*` refs stand**,
unchanged in tip for a FIFTH stamp and still drifting: `-kk1f7v`, `-adewly`,
`-3xu0ki`, `-39rk64` and `-8jq9o6` at **58, 130, 142, 184 and 189 commits
behind** `main`, **zero ahead in every case**. Listed for human triage, not acted
on; `wipsurvey` reads `wip/*` and `parked/*` only, so these are found by
`git ls-remote --heads` and never by the survey.

### Marker census — 73 markers, NINE areas, group 1 flat at 18 across this pass's writes

**`go tool gapaudit` was run TWICE, before and after this pass's writes**, each
against a full-repository feed as `docs/ROUTINES.md` requires.

- **Pre-write, 829-issue feed: 73 markers across 9 areas, group 1 at 18, group 2 at 31.**
- **Post-write, 835-issue feed: 73 markers across 9 areas, group 1 at 18, group 2 at 32.**

**The census grew 69 → 73 and every new marker is tracked.** `parser` went 3 → 5
and **`regex` is a new area at 2** — `regex/regex.go`'s `maxRepeat` (#1474) and
`regex/class.go`'s `unicodeBlocks` (#1473), both landed by #1411 with their
trackers. The rest is flat: `xsd` 33, `validate` 17, `xpath` 6, `xml` 4, `value`
3, `conformance` 2, `cmd` 1.

**Group 2's move is entirely this pass's own writes**: #1426 left it by closing,
#1494 and #1498 joined it as new `kind/gap` issues, 31 − 1 + 2 = 32. Group 1 did
not move at all.

**THREE markers carry FOUR `dead end:` annotations, and the last stamp
mis-located two of them. Measured here, not inherited:**

- `xsd/resolve.go:681` — `cites CLOSED #281` **and** `cites CLOSED #434`. Both,
  on the same marker.
- `parser/doc.go:219` — `cites CLOSED #434`. The `#281` annotation the last stamp
  put here is not here.
- `xsd/contentrestricts.go:681` — `cites CLOSED #501`. The last stamp placed this
  at `:794` and assigned it to #1156; `:794` is the `maxProductStates` marker
  (#499's subject), and `:681` is `maxContentPositions`, whose live
  subject-matter owner is **#1378**. Neither marker is #1156's, whose subject is
  the two markers `gapaudit`'s owner join cannot SEE.

**#1498 owns the first two, unconditionally and as its whole deliverable.** The
third is unowned by any issue that names the marker; #1378 is its subject-matter
owner and is not obliged by its body to repoint it.

**Zero untracked GAP sites — a judgment on the annotations, not the tool's
mechanical rule.** All eighteen group-1 rows carry either a `dead end` naming a
successor this pass filed, or `candidate owner` annotations `gapaudit` itself
labels *"too weak to retire this tracker"* against markers whose prose names
their owner. `gapaudit`'s own doc reports a match-less marker as *"no tracking
issue found"* and never as untracked, which is the reading applied here.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** every write
this pass made, 15 pages (14 full at 100, page 15 at 99), PRs filtered locally:
**1499 rows, 664 PRs excluded, 835 issues — 327 open, 508 closed.** Coverage
verified rather than assumed: 1499 distinct numbers, min 1, max 1499, no gaps.
The paginate recipe ran **three times with zero retries**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **132** | active |
| **M5 — Instance validation (XML)** | **17** | **24** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **310 `ready`, 15 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 327 with no gap, and **every open issue carries exactly
one, verified over all 327 by grouping each issue's own label set rather than by
summing counts**. **#779** owns the mechanical check.

By kind: `kind/refactor` 79, `kind/process` 64, `kind/gap` 61, `kind/tooling`
42, `kind/story` 35, `kind/bug` 34, `kind/docs` 25, `kind/feature` 4. By area:
`meta` 91, `parser` 85, `xsd` 63, `docs` 41, `conformance` 34, `cmd` 28,
`validate` 28, `value` 17, `builtin` 11, `xsderr` 9, `regex` 6, `xpath` 6,
`model` 4, `loader` 2, `cli` 1.

**M4 shed three and M5 gained one.** #1413, #1446 and #1426 left M4 (two landed,
one closed `not_planned`); #1494 was filed into M5. **255 of 327 open issues
carry no milestone**, so neither milestone count is its lane's remaining work —
read the `area/` census for that, and note `area/regex` appearing at 6 where the
last stamp had 2.

**`ready` moved 292 → 310.** Six were filed by this pass (#1494–#1499); #717 was
relabelled from `blocked`; the rest is post-land passes earlier in the window
against the window's closures. That is #347's shape — `ready` is an output, not
a target, and **310 is the honest startable count with ONE to subtract**: #1491
carries a live claim, so **309** are startable without colliding.

**The unblock sweep measured a clean zero for the SIXTEENTH consecutive stamp**,
measured at the source over all 16 pre-write `blocked` bodies' `## Depends on`
sections. **No entry names any of the window's landings.** **One issue was
relabelled, and not by the sweep** — see below. The set is now **15**.

- **FIVE are triggers rather than issues** and say so in their own
  `## Depends on` — **#1374**, **#1224**, **#555**, **#1002** and **#1042**.
  **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **EIGHT still have at least one OPEN named dependency** — **#267** and **#345**
  (both on #250), **#248** (now #717, repointed by this pass), **#593** (#591),
  **#731** (#1414, its last open carve), **#871** (#831), **#1344** (#1324) and
  **#1458** (#479).

**The one relabelling is the pass's largest structural finding and the sweep
could not have made it.** #248 and #717 were each other's dependency:

- #248's `## Depends on` named #250, the M5 epic, and said *"Relabel `ready` when
  the M5 carve produces the concrete validator slice that calls this — not when
  #250 merely exists."*
- #717 IS that slice, and was `blocked` on #248, whose export its own Acceptance
  says *"lands here or is ruled unnecessary on this thread."*

Neither could ever start, and the stall dated from #250's filing on 2026-07-25 —
**seven weeks**, on the `instance` lane's largest named lever. An epic is a label
on a milestone, not a gate that closes, and STYLE 8 is satisfied by an export
landing in the same commit as its first caller, which is what #717 already
prescribes. **#717 is `ready` with no dependency; #248 is `blocked` on #717.**
The sweep reads `## Depends on` for CLOSED entries and this pair had none — a
mutual wait is invisible to it, which is worth one line in whatever eventually
mechanizes #779.

### Persona consultations — the TWENTY-FOURTH, and two findings died on their premises

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **The orchestrating session ran both personas fresh against the
published surface** (README plus `go doc` plus CLI `-help`, never source) and
handed the reports here. Both were told #1452–#1455 were already filed and
neither re-reported them.

**Six findings, and every one was re-derived here before disposition.** Two died
on their premises, and the two that died are the CLI-facing ones that looked most
like bugs:

| finding | disposition |
|---|---|
| `cliuser`: an unresolvable `xsi:type` exits 0, silent | **FILED #1494.** Reproduced. The tree's `cvc-elt` clause-4 reading is correct and documented; what is answered nowhere is the change log's separate claim that the ATTRIBUTE is invalid. Routed to the oracle, with "the spec charges nothing" named as a legitimate outcome |
| `libuser`: `value`'s doc names an UNEXPORTED helper as the four #1412 rules' only handle | **FILED #1496.** Confirmed from `go doc` output both ways — `boundConsistencyViolates` appears in `go doc ./value`, and `go doc ./xsderr` spells rule IDs as quoted literals. Explicitly independent of #1454's export ruling |
| `libuser`: the four seeded `xsi:` declarations' `{type definition}` is undocumented | **FILED #1497**, and sharpened past what the persona could see: three of the four are `TypeDefinitionRef` and **`xsi:schemaLocation`'s is an anonymous inline list whose `{name}` is the zero QName**, so a consumer reading `ref.Name` for all four gets nothing for it and no diagnostic |
| `cliuser`: `minExclusive` = `maxExclusive` should reject like the other three | **DISMISSED, falsified premise.** §4.3.8.4 verbatim is *"greater than"*, not *"greater than or equal to"*; the four rules disagree by design and `boundConsistencyViolates`' doc says so before the code. Recorded on #1412 with the spec line |
| `cliuser`: no documented way to escalate exit 4 to a CI failure | **DISMISSED, falsified premise.** Exit 4 is non-zero, so escalation is the default; nobody asked for the downgrade. The residual — *"less severe"* reading as *"benign"* — is **#1407's**, and is recorded there as one more sighting |
| `cliuser`: bound-consistency `<loc>` names the enclosing type | **CORROBORATION for #1453**, recorded on its thread. Independently reproduced across all four #1412 rules **and across a cross-document `-schema` narrowing**, which #1453's body did not test. Its three ruled arms stand untouched |

**Two issues were filed from grounding the findings rather than from the findings
themselves**, and both are the pass's own cost:

- **#1495** — `suiteindex`'s ELEMENT position fixes its namespace URI and
  wildcards only the local name, so *"any element in any namespace carrying
  `{xsi}type`"* is inexpressible and answers **0 occurrences** rather than saying
  so. CLAUDE.md tells a session to predict from `suiteindex` rather than from a
  grep (#1239); **#1494 therefore ships with no lane prediction**, only a
  population bound (400 occurrences in 179 no-namespace fixtures).
- **#1499** — the `kind/refactor` cost-of-delay banding clause. Measured over all
  327 open bodies: **79 carry `kind/refactor`**, three carry `Cost of delay`
  (#849, #848, #845), **zero read `Increasing`**. The clause selects nothing and
  has on every stamp that measured it; the standing disposition was to file about
  the routing on a third sighting and this is well past it. Commented on #849,
  which is explicitly not what #1499 asks about.

**`testdata/xsdtests` WAS initialized by this pass** (`git submodule update
--init`, suite at `7bc3365`), so the `suiteindex` figures above are measured
rather than cited — the first stamp in four able to say that. **No conformance
measurement was taken**: the lane table is the committed expectations, which
`docs/WORKFLOW.md` names as the lane score (#1120).

**The environment brief for this pass said `gh` was unavailable and it was
wrong.** `gh api repos/kud360/goxsd8/...` served every read and every write,
including three 15-page `state=all` fetches with zero retries — which is
`docs/ROUTINES.md`'s **Survey input** working exactly as documented. Every issue
body and comment this pass wrote went through `-F body=@FILE` and is therefore
byte-faithful, and every body it READ came from REST rather than the MCP channel
(#764).

### Working band

Ordered for a `/develop` session: take the highest row you can start. **#1491 is
band row 1 AND carries the only live claim** — re-run `wipsurvey` before starting
anything, because this stamp's own table changed verdict mid-pass.

| # | issue | why here |
|---|---|---|
| 1 | #1491 | **The `kind/process` clause at full strength: the remedy #1437 landed in this window can NEVER fire.** `ROUTINES.md` fetches comments for `CLAIMED` rows only, and an empty-diff branch is `EXPIRED`, so the empty-diff `TAKEOVER` counter #1437 shipped has no input path. Three of this window's sessions turned on this machinery (#1437's own landing, #1443's park, #1426's park) and the fix is one session. **Carries a LIVE claim** — it is banded because it is the right pick, not because it is free |
| 2 | #1498 | **New this pass, and it is the park recovery the 2026-09-14 log recorded as owed.** Two-file comment rewrite, zero open questions: the §5.3 ruling is settled and re-confirmed twice, and the five-point replan brief is restated in the body so it stands alone. **It clears three of the four `dead end:` annotations** this stamp measured, which no other queued issue can. Start from `main`; `wip/issue-1426` is evidence, not a base |
| 3 | #1465 | **The best-evidenced lane slice in the queue and the only MEASURED prediction in it.** `no-xmlns` (§3.2.6.3): **+2 `schema`**, measured at `f996bd4` rather than inferred, with the double-charge guard named (`xmlns:` and `xmlns:a` fail `cvc-datatype-valid` first and that test must pass unchanged) and #1446 as a landed template for the whole shape including the `rulecat` regeneration. M4 |
| 4 | #717 | **Unblocked by this pass after a seven-week circular wait, and it is the `instance` lane's largest named lever** — `cvc-wildcard` admission (§3.10.4.1 clauses 1–3) plus `{open content}`. The lane is 11044/26361 and flat for two stamps. It absorbs #248, so the export lands with its first caller in one commit. **Warden pre-flight mandatory** — #248's whole premise is that the shape could not be settled without a caller, and now there is one. **Read the new `{open content}` Acceptance bullet**: a landing that ships wildcard admission alone must record that decline and name its successor |
| 5 | #415 | **#407's part 2 of 2, and #407 landed in this window precisely so this can be SHOWN rather than argued.** With the `{annotation}` machinery gone, `tree.go`'s `Node`/`Text` character-data retention is justified by its own doc comment's reference to a subsystem that no longer exists. Its blocker closed three commits ago; nothing else in the queue is this ripe. `Ratchet: unchanged`, verified not assumed |
| 6 | #1494 | **New this pass, grounded before filing, and the only band row whose deliverable may be "the spec charges nothing".** An `xsi:type` naming an absent type exits 0 silently. The `cvc-elt` clause-4 half is correct and must not be disturbed; the open half is the change log's *"the `xsi:type` attribute is invalid"*, which has no normative site this pass could find. **Oracle first** — do not implement a charge before the rule ID exists. M5 |
| 7 | #1429 | **The measurement apparatus, and #434 proved it reports green while pinning nothing.** Both #276 decline guards decline only because the root's reference fails `src-resolve` at finalize; remove that charge and both PASS under `expectValid=false`. **The mutation check IS the issue**: deleting the finalize charge must leave both tests RED. Read **#763** first. Banded below #1494 only because #1494 is a persona finding with a live reproduction |
| 8 | #1451 | **Banded a second stamp, and its costed sighting is a whole session (#434).** A construct census cannot see whether a case is SCORED, `versionApplicable` withholds silently, and `discovery.withheld` already computes the answer on every run and shows it to nobody. **#1498 prescribes the manual version of it** — check every case ID with `GOXSD_CASE` before writing it — which is one more sighting since the last stamp. **Its claim EXPIRED mid-pass**; the branch is startable |
| 9 | #1495 | **New this pass and it cost this pass a prediction.** `suiteindex` cannot express an instance-side census and answers `0` rather than saying so, which is the failure mode #1239 filed the tool to prevent. One session. Take it with row 8 in mind — the two together are what makes a lane prediction re-derivable |
| 10 | #1368 | **The one that puts an operator in front of a filename that does not exist.** A `-schema` document whose root is not `xs:schema` is charged `[src-include]` against `goxsd8-schema-set.xsd:2:1`. Distinct from **#1224**, which owns the hint path. **Take it with #1419 and #1421** — all three are `cmd` exit-code-and-message defects in the same two files, and three separate landings rebase three times |
| 11 | #1324 | **The only row that discharges a `blocked` entry by RULING rather than by landing code**, and it discharges in either direction. **#1344** waits on it and closes with it whichever way it goes. The body states both outcomes as checkable `grep` results, which is what makes it one session |
| 12 | #1499 | **New this pass, and it is the only row that can retire a standing obligation this file has carried for nine stamps.** Not a refactor and not about #849: it asks whether the cost-of-delay clause has an input at all, given 79 open refactors and zero `Increasing` rankings. Banded last because the queue survives without it and the other eleven rows do not depend on it |

**Named below the band, deliberately:** **#1473** and **#1474** are the new
`regex` rulings and are each one session, but both land a RULING whose licensed
work may be larger, and #1474's own Acceptance says `Ratchet: unchanged` — they
wait until a stamp can say what #1411's corpus bank left reachable. **#1414**
(#731 mechanism 4) is startable and is **not** banded: its entire named cohort
went `pass` under #1411, so it now banks nothing and is a pure under-rejection.
**#1496** and **#1497** are this pass's doc filings and neither is banded — both
are one-paragraph landings whose value is real and whose urgency is not.
**#1462**, **#1283**, **#1417**, **#1400** and **#1355** are each real and each
one session. **#1452** remains the closest to bandable of the twenty-third
consultation's filings and is unbanded a second stamp.

### Next planning action

1. **Say whether the JOIN should become normative — the question is now three
   stamps old and this window gave it a NEGATIVE data point.** #1411 banked +609
   against a prediction of 5, and it was right to: one mechanism banked whole is
   PRINCIPLES 22, not a missed prediction. So the census-joined-to-`schema.txt`
   method predicts well for a *named-cohort* issue (#1428, #1446, #1413 all
   exact) and says nothing useful about a *mechanism* issue. **That distinction
   is what a normative rule would have to carry**, and neither `CLAUDE.md`'s
   surveys block nor `arbiter.md`'s #1332 paragraph has a place to put it. **#1451
   and #1495 together own the instruments**; the method itself is still unfiled,
   deliberately.
2. **A mutual `blocked` wait is invisible to the unblock sweep, and one cost
   seven weeks on the `instance` lane.** The sweep reads `## Depends on` for
   CLOSED entries; #248 ↔ #717 had none, both being open. **The next stamp should
   say whether a second such pair exists** — the cheap test is a cycle check over
   the 15 `blocked` bodies' named dependencies, which is small enough to do by
   hand and is exactly the kind of thing **#779** would mechanize. **Not filed
   here**: one instance, and #779 already owns the queue-hygiene mechanism.
3. **`schema` is at 95.4% and M4 has 54 open issues.** The lane's residual (708)
   is now smaller than the open-issue count for the first time, and most of those
   issues are `kind/refactor` and `kind/docs` that move no lane. **The next stamp
   should say what fraction of M4's 54 is lane-visible at all**, because the
   milestone count has stopped being a proxy for remaining work and the band has
   been picking around that fact rather than stating it.
4. **The persona trigger fired on a window with no library-surface change and
   still produced three filings.** Four of the seven landings touched `parser`,
   `value` or `xsd` internals with `surface: unchanged` and no new CLI behaviour,
   yet both personas found gaps — which suggests the trigger's surface arm is not
   what is generating value here and the standing documentation debt is. **Two
   consecutive consultations have now found zero accuracy bugs in the CLI
   contract**, and this one found two findings that died on their own premises,
   which is the first time that has happened. **Carried as a datum**; `/retro` is
   where it becomes a pattern.
5. **The human decision blocking #1002 is unchanged and is now carried for a
   TWENTIETH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — so no `/backlog` can move it and none should try.
6. **No `/retro` landed on 2026-09-13.** The last one is 2026-09-06 (eighth
   weekly), and `git log` since carries no retro commit, so the weekly routine
   missed a cycle. Four items above are explicitly routed to `/retro` and none of
   them can be answered by a `/backlog`. **This is an observation about the
   schedule, not a filing** — `docs/ROUTINES.md` owns the cron and nothing in this
   container can read whether the routine fired.

**What this pass did, so the next one can tell a completed run from a skipped
one:** **six issues filed** (#1494–#1499); **two body PATCHes** (#717's
`## Depends on` and one new Acceptance bullet; #248's `## Depends on` plus two
stale `GAP(` counts corrected against `gapaudit`'s 73); **one label change**
(#717 `blocked` → `ready`); **one closure** (#1426, `not_planned`, replaced by
#1498); and **SEVEN thread comments** (#717, #248, #1412, #1453, #1407, #1426
and #849 — the dismissals and the corroboration included, each naming what was
dismissed and on what evidence). **Every body PATCH and comment went through
`gh api -F body=@FILE`** and is byte-faithful. **`gapaudit` was run TWICE**,
before and after the writes, and group 1 held at 18 both times.

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

`validate` engine plus `validate/xmlsrc`: greedy deterministic matching,
identity constraints, `xsi:type`/`xsi:nil`, wildcards, default and fixed
values. **`instance` lane.**

Carved 2026-08-12 into ten slices, #710–#719, and now **seventeen** — #766 was
split out of #714 by a warden pre-flight, #773, #774 and #775 out of #766 by
another, #782 and #783 out of #715 by its own, and **#790** filed from #715's
warden verdict, which had named the seam and been read by nobody who filed it.
Thirteen have landed — **#719** joined them on 2026-08-27 at `bd887cd`, wiring
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
