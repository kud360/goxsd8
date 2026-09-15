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

## Status — 2026-09-15 (`/backlog`, the twenty-fifth. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` after `git fetch --unshallow origin`, the marker census a fresh `gapaudit` run TWICE — before and after this pass's writes — and the milestone and queue counts a page-numbered `state=all` fetch taken **after** them. **The window is SIX landings and TWO lanes moved** — `schema` 14690 → **14692** (+2, all of it #1465, which predicted +2 and measured +2) and `instance` 11044 → **11151** (+107, #717, which **landed after this section was written and is folded in by merge**). `datatypes` is flat for a third stamp. **The sixth landing arrived mid-stamp and this section was corrected to it rather than shipped one landing behind** — which is the clearest demonstration this file has of why a band it publishes is not an authority on what is startable. **The event is that the survey channel itself was caught lying**: this pass took FOUR full `state=all` walks and **TWO came back corrupt** — one duplicated a page and silently dropped 100 issues at exit 0 with valid JSON, passing every check `docs/ROUTINES.md` prescribes — which is **#1520**, filed here and band row 2. **The epic-as-gate ruling the last stamp made is applied completely**: #267 and #345 were both `blocked` on the M5 epic by the same mechanism that held #248 ↔ #717 for seven weeks, and both are now `ready`. **The marker census grew 73 → 74, group 1 held at 18 before AND after this pass's writes, and the `dead end:` annotations fell FOUR → ONE** — #1498 cleared three, exactly as it was banded to. The **TWENTY-FIFTH persona consultation**: **five findings — ONE filed (#1521), ONE dismissed on a falsified premise, and THREE already filed and recorded as corroboration**, which is the first consultation in this record whose majority finding was a re-discovery)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11151 | 15210 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14692 | 706 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This table is `main` at `1880f0b`, and it is the committed expectations, which
`docs/WORKFLOW.md` names as the lane score (#1120).** **It was re-read after
#717's landing rather than left at the tip this pass surveyed** — at `ca0ada5`
the `instance` row read `11044 | 15317`, and #717 merged mid-pass for **+107**.
**ONE `wip/` branch stands outside this table** — `wip/issue-1512`
(`kind/process`, a one-paragraph doc edit) — and it can move no lane.

**The column does NOT sum to the corpus, and this stamp is the first to say so.**
The three lanes hold **41759 distinct case IDs** against a column summing to
**42932** — because **every one of the 1173 `datatypes` cases is also committed in
`instance.txt`**, and **701 of them carry contradictory verdicts there**. That is
**#1507**, band row 1, and until it is ruled, *"the `instance` lane is
11151/26361"* is a figure whose denominator the tree disputes. **#717's +107
lands squarely inside that dispute**: the lane it moved is the one holding 1173
case IDs it does not own.

### The window: SIX landings, `schema` +2 and `instance` +107, and a prediction that hit exactly

| commit | issue | ratchet |
|---|---|---|
| `1a626e5` | **#1498** | rebuild both §5.3 `GAP(` marker paragraphs off no live owner — unchanged |
| `6fb0f4c` | #1491 | widen the comment-fetch recipe to CLAIMED-or-EXPIRED — unchanged |
| `dab7194` | #1493 | name the missing input on a tip-age EXPIRED row — unchanged |
| `7ba2cd1` | **#1465** | charge `no-xmlns` (§3.2.6.3) at both attribute-declaration productions — **`schema` +2** |
| `bfefd01` | #415 | correct `tree.go`'s `Node`/`Text` doc comments — unchanged |
| `1880f0b` | **#717** | decide `cvc-wildcard` admission on the instance side — **`instance` +107**, and it **absorbed #248**, which closed `completed` with it |

Plus **#1451 closed `not_planned`** at `docs/WORKFLOW.md`'s three-lost-round cap,
re-planned into **#1512**, **#1513** and **#1514**.

**#1465 predicted +2 and measured +2**, which continues the exact-prediction run
for *named-cohort* issues that #1428, #1446 and #1413 opened — now four
consecutive.

**And #717 is the window's *mechanism* issue, which is the case #1411 exposed and
this stamp had said nothing tested.** Its landing did not merely bank a number: it
**reconciled the arithmetic**, and that is the datum the JOIN question has been
waiting for. The decline census fell by **109** while **107** improved, and the
two-case gap was chased rather than left hanging — `particlesB013.v` and
`schA1.v` moved from honest decline to a DECIDED and WRONG verdict, both the
**#771** class (a child declared only in a schema document the instance reaches
through its own `xsi:schemaLocation`, which `conformance/instance.go` consults for
nothing). The arbiter re-derived `fail` 15210 = 15192 + 5 + **13** independently of
mason's figures. **So a mechanism issue CAN be predicted, if the prediction is
stated as a census and the residue is reconciled rather than rounded** — which is
more than #1411 could say, and it is the first evidence pointing at what a
normative JOIN rule would have to require.

**The band was consumed whole at the top.** Rows 1, 2 and 3 (#1491, #1498,
#1465) landed; row 5 (#415) landed **during this pass**; row 8 (#1451) was parked
and re-planned; and **row 4 (#717) landed during this pass too, as `1880f0b`,
after this section was written** — it is folded in by merge, not by survey.
**SIX of twelve banded rows resolved inside about a day**, and the one claim
still held (#1512) is a banded row. **That is the last band's vindication and
this one's warning in the same fact**: a band this file publishes is stale within
hours, so `wipsurvey` is the authority on what is startable, never this table.

**#1498 did what it was banded to do and the census proves it**: the `dead end:`
annotations this file carried fell **FOUR across three markers → ONE**.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against `git fetch --unshallow origin` and this
pass's verified post-write feed, second pass with #1512's comments supplied:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
434    wip/issue-434   98h15m0s   RETIRED  wip/issue-434: issue #434 is closed
717    wip/issue-717   3m0s       LIVE     wip/issue-717: tip pushed 3m0s ago, within the 2h0m0s claim TTL
732    wip/issue-732   549h9m0s   RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   727h7m0s   RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   499h28m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   693h8m0s   RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   main's     RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   main's     RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   504h48m0s  RETIRED  wip/issue-993: issue #993 is closed
1332   wip/issue-1332  77h23m0s   RETIRED  wip/issue-1332: issue #1332 is closed
1356   wip/issue-1356  52h25m0s   RETIRED  wip/issue-1356: issue #1356 is closed
1426   wip/issue-1426  25h28m0s   RETIRED  wip/issue-1426: issue #1426 is closed
1451   wip/issue-1451  10h10m0s   RETIRED  wip/issue-1451: issue #1451 is closed
1512   wip/issue-1512  10m0s      LIVE     wip/issue-1512: tip pushed 10m0s ago, within the 2h0m0s claim TTL
```

**`wip/issue-415` and `wip/issue-1491` are gone**, auto-deleted at merge as the
scheme requires — and **`wip/issue-717` went the same way after this table was
taken**, when #717 merged as `1880f0b`. **The table above is therefore already one
row out of date, deliberately left as it was read**: it is the survey this pass
actually ran, and the correction belongs beside it rather than inside it. The
live set as this section is merged is **`wip/issue-1512` alone**.

**`wip/issue-1512` demonstrated two of `wipsurvey`'s own remedies inside one
pass.** It first read `CLAIMED — undated` (no commits of its own, so the tip age
is `main`'s), then `UNKNOWN — tip not fetched, run git fetch origin` because the
holder pushed between this pass's fetch and its survey, then `LIVE` after the
re-fetch. Each verdict was correct when taken. **A `wipsurvey` verdict is a
snapshot; re-run it before starting anything.**

**The supersede-or-verify duty is clean, and the instrument was changed to make
it so.** An ancestry test is the wrong one: a three-dot diff against `main` shows
a squash-merged branch's own commits as additions, so eleven branches "differ
from `main`" for a reason that is not a missing landing. Measured instead by
closure reason and merged PR: **every RETIRED branch with a non-empty diff
belongs to an issue closed `not_planned`** — nothing was owed on `main` — and the
one `completed` closure, **#1332**, merged through PR #1441. #1332's 14-line
residual was chased to the end: it is a **pre-repair-round draft** of the
arbiter.md paragraph, and the version on `main` is strictly larger (three
outcomes, the instance-lane distinction, the #609 fold-instruction). **No
supersede is owed anywhere.**

**Zero `needs-replan` open, zero `parked/*`, zero `meta/*`** — a second stamp.
**SIX non-`wip` `claude/*` refs stand** (was five; `-ldi2dl` is new) at **7, 69,
141, 153, 195 and 200 commits behind `main`, ZERO AHEAD in every case**. Listed
for human triage, not acted on; `wipsurvey` reads `wip/*` and `parked/*` only.

### The survey channel itself is unreliable, and nothing in the tree checks it

**This is the pass's largest finding and it is not about any issue's content.**
Four full `state=all` walks were taken with `docs/ROUTINES.md`'s **Survey input**
recipe verbatim. **Two came back corrupt.**

| walk | rows | distinct | verdict |
|---|---:|---:|---|
| 1 | 1516 | 1516 | clean — and it exercised #1145's transient short page (page 5 returned 46; the re-request returned 100 and repaired it) |
| 2 | 1518 | **1418** | **CORRUPT** — page 3, whose offset is `#1318 → #1219`, held `#1117 → #901`. **#1219–#1318 never fetched** |
| 3 | 1406 | 1404 | **CORRUPT** — ~116 absent, 2 duplicated |
| 4 | 1520 | 1520 | clean; used for every figure below |

**Walk 2 passed every check the recipe makes.** Sixteen pages, fifteen full at
100 and the last at 18, so the fullness rule held; `jq -s add` succeeded; the
reshape emitted well-formed JSON; **the run exited 0**. The recipe's only
integrity check is CARDINALITY, and a page served from the wrong offset has the
right cardinality.

**Measured blast radius**: walk 2's hole holds **29 open issues, one of them
`kind/gap`** (#1310), so every queue figure derived from it is short by 29 and
`gapaudit`'s group-2 census undercounts by one. **The stronger failure is a
HYPOTHESIS and is marked as one in the issue**: a marker citing an issue inside a
hole would read as needing a look, and a `wip/issue-N` whose issue fell in one
would classify with no issue data — **neither is reproducible at `ca0ada5` nor at
the merged `1880f0b`**, re-checked after #717's landing (no Go file cites any of
the 29; no `wip/` ref names one).

**#1520 is filed for it** and is band row 2. **#1153** (the harness) gained the
scenario; the two are independent, and a mock answering *how many items* cannot
express *which items*, which is why that scenario had to be written down rather
than assumed.

**Every figure in this stamp comes from a walk verified `rows == distinct`, no
gaps.** That check is this pass's practice and is not yet in `docs/ROUTINES.md`;
#1520 is what puts it there.

### Marker census — 71 markers at the merged tip, NINE areas, group 1 flat at 18, dead ends 4 → 1

**`go tool gapaudit` was run THREE times: before this pass's writes, after them,
and again after #717's landing merged in** — each against a full-repository feed
as `docs/ROUTINES.md` requires.

- **Pre-write, `ca0ada5`: 74 markers across 9 areas, group 1 at 18, group 2 at 32.**
- **Post-write, `ca0ada5`: 74 markers across 9 areas, group 1 at 18, group 2 at 32.**
- **Merged, `1880f0b`: 71 markers across 9 areas, group 1 at 18, group 2 at 31.**

**Group 1 held at 18 across all three**, which is the reading this section exists
to give: neither this pass's writes nor a 107-case landing left an unowned marker
behind.

**Over the pass's own writes the census grew 73 → 74 and the new marker is
tracked**: `xsd` went 33 → 34 — `xsd/attributedeclaration.go`'s `no-xmlns`
marker, landed by #1465 and owned by **#1511**, written in by #1465's own
post-land pass.

**Then #717 RETIRED three, net, which is the largest marker reduction in this
record**: `validate` **17 → 15** and `xsd` **34 → 33**, as cvc-wildcard admission
replaced declines with decisions. Group 2 fell 32 → 31 with #1516 leaving it. The
rest is flat: `xpath` 6, `parser` 5, `xml` 4, `value` 3, `conformance` 2, `regex`
2, `cmd` 1. **#1516 is the successor #717 filed out of its own boundary** and is
now `ready`, so the markers that remain in that file are owned.

**The `dead end:` annotations fell FOUR across three markers → ONE.** #1498
cleared three by rebuilding both §5.3 paragraphs to carry **zero `#N` tokens** —
provenance spelled without the sigil, deliberately, because `gapaudit` matches
bare `#(\d+)` and never reads the prose around it. Those two markers
(`parser/doc.go:219`, `xsd/resolve.go:681`) therefore sit in group 1 **by
design**, and filing an owner for them is what CLAUDE.md's closing sentence
reserves to a human.

**The one survivor, and a correction to the last stamp.**
`xsd/contentrestricts.go:681` (`maxContentPositions`) cites CLOSED #501. The last
stamp said #1378 *"is not obliged by its body to repoint it"*. **That is wrong,
and re-reading #1378 is what found it**: its arm (b) requires, in as many words,
*"the marker rewritten to cite **this** issue"*, and its arm (a) deletes the
marker outright. **#1378 owns the retirement under either arm**, so this is a
tracked site, not an orphan.

**Zero untracked GAP sites** — a judgment on the annotations, not the tool's
mechanical rule, and `gapaudit`'s own doc reports a match-less marker as *"no
tracking issue found"* rather than as untracked.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** every write
this pass made **and re-read again after #717's landing merged into it**,
**verified `rows == distinct`, min 1, max 1524, no gaps**: **1524 rows, 677 PRs
excluded, 847 issues — 331 open, 516 closed.** The re-read is why these are
`1880f0b`'s numbers and not `ca0ada5`'s: **#717 and #248 both closed
`completed`** in that landing.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **133** | active |
| **M5 — Instance validation (XML)** | **17** | **25** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **317 `ready`, 12 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 331 with no gap, and **every open issue carries exactly
one, verified over all 331 by grouping each issue's own label set rather than by
summing counts**. **#779** owns the mechanical check.

By kind: `kind/refactor` 79, `kind/process` 67, `kind/gap` 61, `kind/tooling` 42,
`kind/story` 35, `kind/bug` 34, `kind/docs` 27, `kind/feature` 3. By area: `meta`
93, `parser` 83, `xsd` 63, `docs` 43, `conformance` 35, `cmd` 29, `validate` 28,
`value` 17, `builtin` 11, `xsderr` 9, `regex` 6, `xpath` 6, `model` 4, `loader`
2, `cli` 1.

**M4's open count is not its remaining lane work, and this stamp answers the last
one's question with a number.** Of M4's **54** open issues, **at most 23 are
lane-visible** — 15 `kind/gap`, 7 `kind/bug`, 1 `kind/feature` — against **31**
(`refactor` 21, `docs` 6, `process` 2, `tooling` 2) that move no lane by
construction. **23 is an upper bound, not an estimate**: #1511 is `kind/gap` and
its own Acceptance says `Ratchet: unchanged` *structurally*, because every suite
fixture is a schema document and the rule is already charged there. **259 of 331
open issues carry no milestone at all**, so read the `area/` census, not the
milestone counts.

**The M4 and M5 SCOPE SECTIONS below are not fit to read and this stamp files
rather than fixes them.** Each has become a running chronicle of individual
landings — **M4 is 220 lines carrying 23 "has now LANDED" clauses**, each with a
date, a SHA and a lane delta that `docs/LOG` already owns — against this file's
own preamble (*"status, not history"*). They accrete because, unlike the Status
section, nothing replaces them. **#1522** owns the reduction; it was not done here
because a 400-line prose deletion needs a per-clause recoverability check and its
own reviewable commit.

**`ready` moved 310 → 317.** Three filed by this pass (#1520, #1521, #1522), two
relabelled from `blocked` (#267, #345), one unblocked by #717's landing (#1516),
one closed by it (#717 itself), the rest from post-land passes earlier in the
window. That is #347's shape — `ready` is an output, not a target — and **317 is
the startable count with ONE to subtract**: #1512 carries the only live claim, so
**316** are startable without colliding.

**The unblock sweep measured a clean zero for the SEVENTEENTH consecutive
stamp**, over all sixteen pre-write `blocked` bodies' `## Depends on` sections.
No entry names #1498, #1491, #1493, #1465, #1451 or #415. **Two issues were
relabelled and the sweep could not have made either** — see below. **And then
#717 landed and made the eighteenth sweep non-zero**, which is recorded here
rather than deferred: **#248 closed `completed` absorbed into that landing**, and
**#1516 relabelled `ready`**, its sole dependency discharged. The set is now
**12**.

- **FIVE are triggers rather than issues** and say so in their own
  `## Depends on` — **#1374**, **#1224**, **#555**, **#1002** and **#1042**.
  **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **FIVE still have at least one OPEN named dependency** — **#593** (#591),
  **#731** (#1414), **#871** (#831), **#1344** (#1324) and **#1458** (#479).
  **The two that left were both waiting on #717** and both discharged the moment
  it landed, which is the shape a `## Depends on` naming a concrete issue is
  supposed to have — and the exact contrast with the epic-as-gate pair below.

**The cycle check the last stamp asked for was run, and its answer is NO — but it
found the same disease by another route.** Over all sixteen `blocked` bodies:
every named dependency target is itself `ready` or `epic`, never `blocked`, so
the graph is depth-1 and **contains no second mutual pair**. What it found
instead is **#267 and #345, both `blocked` on #250, the M5 epic**.

An epic is a label on a milestone, not an issue that closes, so waiting on #250
is waiting for the whole of M5 — and the unblock sweep can never see it, because
the sweep reads `## Depends on` for CLOSED entries and #250 will be open until M5
completes. **The last stamp established exactly this ruling while breaking
#248 ↔ #717, and applied it only to the literal cycle.** Both issues carry a
discharge arm needing no M5 code: #267's Acceptance bullet 1 is oracle exegesis
over §3.10.4.3, and #345's arm (b) is the documented-permanent-fail-open ruling
its body has offered since 2026-07-31, pointing at #267's own precedent. **Both
are now `ready`**, with the reasoning on their threads and #250 kept as a soft
reference for the arm that does need it. Five stale `contentrestricts.go:1047`
citations in #345 were corrected to `:1099` in the same edit.

### Persona consultations — the TWENTY-FIFTH, and four of five were re-discoveries

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **The orchestrating session ran both personas fresh against the
published surface** (README plus `go doc` plus CLI `-help`, never source) and
handed the reports here. **Every finding was re-derived here — against the tree,
and for the CLI ones against a binary built from it — before disposition.**

| finding | disposition |
|---|---|
| `cliuser`: an instance violation cites the argument AS SPELLED while a schema violation is absolute | **FILED #1521.** Reproduced, and sharpened past what the persona could see: the instance URI is **not canonicalized at all**, so `sub/x.xml`, `./sub/x.xml` and the absolute path give three different `<loc>` prefixes while the schema side gives one answer for all three. `doc.go:160-162` documents the schema rule (*"an error cites a path the reader can open"*) and nothing documents the instance one. Not a broken promise — which is why documenting it is a full discharge |
| `libuser`: `xsderr` exports no rule catalog, so a dispatcher has no compiler-checked vocabulary | **ALREADY FILED — #1454**, re-derived independently by a persona told only #1452–#1455 by number. **One new argument, measured**: `catalog.go` grew **217 → 224 over five landings and +1 in this window** (#1465's `no-xmlns`), so a caller who transcribes string literals holds a vocabulary that decays with no compiler error. Recorded on the thread |
| `cliuser`: `types:`/`components:` silently exclude anonymous type definitions | **ALREADY FILED — #1381** (2026-09-09). Reproduced here: `types: 0` for an inline-only schema, `types: 1` where one named and one anonymous stand. The persona independently noticed the `top-level` qualifier on the `model groups` line and its absence on this one, which is already in #1381's title |
| `libuser`: `xsderr.Loc` carries no structural locator | **ALREADY FILED — #1291**, and the persona independently reached that issue's own counter-argument, grading itself *"not filing-strength on its own"* on the grounds #1291 already records. Two library passes a fortnight apart declining to call it a defect is evidence for its **second** Acceptance arm — say `file:line:col` is the only positional currency before 1.0 |
| `cliuser`: a missing instance file gives a raw `os.Open` error instead of the `<file>: <sentence>` shape | **DISMISSED, falsified premise.** The raw shape is what **every** unopenable-file path gives — missing instance, missing schema, and `parse` — all at exit 2, the documented code for an argument the process could not read. There is no instance-side inconsistency; the contrast drawn was with a different fault class (an argument that WAS read, whose extension names no format). Recorded on #1419 |

**Four of five findings were already filed, and that is the datum.** No previous
consultation in this record has had its majority be re-discovery. The two
personas found one new gap between them, and it is a documentation asymmetry
rather than an accuracy bug — **a third consecutive consultation finding zero
accuracy bugs in the CLI contract**.

**`testdata/xsdtests` WAS initialized by this pass** (`git submodule update
--init`, suite at `7bc3365`), so the `suiteindex` figures here are measured rather
than cited. **No conformance measurement was taken**: the lane table is the
committed expectations (#1120).

**`gh api repos/kud360/goxsd8/...` served every read and every write.** GraphQL
(`gh issue list`) is still a 403, exactly as `docs/ROUTINES.md` records. Every
body PATCH and comment went through `-F body=@FILE` and is byte-faithful; every
body READ came from REST rather than the MCP channel (#764).

### Working band

Ordered for a `/develop` session: take the highest row you can start. **#1512
carries the only LIVE claim** — #717 held one when this band was written and has
since landed. **Re-run `wipsurvey` before starting anything**: one of this stamp's
own rows changed verdict twice mid-pass, and another landed between the band being
written and the section being merged.

| # | issue | why here |
|---|---|---|
| 1 | #1507 | **The measuring stick the one rule is stated over.** Every claim re-derived here from the committed files: `datatypes ⊆ instance`, all 1173; union 41759 against a column summing to 42932; **701 contradictory verdicts** (697 `pass`/`fail`, 4 `fail`/`pass`). `runner.go:192-193` asserts lanes are disjoint and the tree it ships in says otherwise. `instance`'s **15210** failures include 697 cases `datatypes` banks as passing — the residual **#717 just moved by 107** and **#1516** is scoped against. **#717's +107 is the first real test of the dispute**: it banked into the one lane holding 1173 case IDs it does not own, so whether that figure is 107 units of M5 progress is partly this issue's answer to give. **Rule plus arm B is one session; arm A is a 1173-case re-partition and should be CARVED, not attempted** |
| 2 | #1520 | **New this pass, and the `kind/process` clause at full strength.** Every survey this project runs is fed by one recipe, and that recipe **corrupted two of four walks today**, silently, at exit 0, passing its own only check. The fix is one assertion before the reshape. It is not banded above #1507 because a corrupt feed is re-derivable next session and a mis-stated lane score is not |
| 3 | #1495 | **Two costed sightings on consecutive days by two different sessions** — and the second is a live issue body (#1516) instructing a future session to run a query that answers `0` against a real population of **51 fixtures**. That is the `kind/tooling` clause's own trigger, not a restatement |
| 4 | #1512 | **CARRIES A LIVE CLAIM.** Three mason rounds died on this two-paragraph diff and nothing was ever found wrong with its content; the body already rules that a doc edit no compiler reads needs no delegation. The cost is paid and the sentence is still unwritten |
| 5 | #1513 | The `arbiter.md` half. Shares no line with #1512 and neither depends on the other — but check whether #1512's holder took both before starting |
| 6 | #1514 | The withheld-minus-banked instrument — the neighbour of #1507's question, and what makes any lane prediction re-derivable rather than asserted. Its Acceptance must FAIL on `main` before the arm is built |
| 7 | #1494 | M5: an unresolvable `xsi:type` exits 0, silent. **Oracle first** — do not implement a charge before the rule ID exists, and *"the spec charges nothing"* is a legitimate outcome. Gains the lane prediction it shipped without once #1495 lands |
| 8 | #1429 | The measurement apparatus, and #434 proved it reports green while pinning nothing. **The mutation check IS the issue**: deleting the finalize charge must leave both tests RED. Read **#763** first |
| 9 | #1502 | **#1498's last unchecked suite claim**, and the only thing left on the paragraph #1498 was filed to make accurate. One file, comment-only, two stated options and either is defensible. No lane can move and the body says why |
| 10 | #1511 | `no-xmlns` on the component footing. M4. `rulecat` is already discharged (#1465 did it), so this owes it nothing — and the **warden trigger is the CONTRACT, not the signature** (#1168): `go tool surface` will read `unchanged` while an exported constructor starts refusing input |
| 11 | #267 | **Unblocked by this pass after seven weeks on an epic that cannot close.** Acceptance bullet 1 is oracle exegesis and needs no M5 code; arm (b) is the ruling the body has called *"the likely resolution"* since July. **Read #717's thread first** — it **LANDED as `1880f0b` while this band was being written**, and it is the first assessment-time wildcard-attribution consumer in the tree, so arm (a)'s premise has already moved under this issue's feet and a session must re-read before choosing an arm |
| 12 | #1324 | **The only row that discharges a `blocked` entry by RULING rather than by landing code**, and it discharges in either direction. **#1344** waits on it and closes with it whichever way it goes |

**Named below the band, deliberately:** **#345** — unblocked alongside #267 and
held one rank lower for the same reason that lifts #267 — except that the reason
has now FIRED: **#717 landed**, so the assessment-time wildcard-attribution
consumer its arm (a) was waiting for exists, and the next stamp should re-rank
#345 on that fact rather than inherit this placement. **#1499** now carries a sharper diagnosis than it did — the clause has no
PRODUCER, not merely no selection: 79 open refactors, three carrying a
`## Cost of delay`, **zero reading `Increasing`**, and **#849 has said "a steward
re-ranking is warranted" since 2026-08 with none landed**. **#1521** and
**#1381** are this stamp's persona-facing `cmd` work and neither is urgent;
**#1368** is the third of that family and should be taken **with #1419 and
#1421**, since three separate landings rebase the same two files three times.
**#1518** closes the #407 → #415 → #1518 tail. **#1473** and **#1474** are the
`regex` rulings, and **the submodule is now initialized**, so the next session
that wants them can finally measure what #1411's corpus bank left reachable
instead of waiting for a stamp to say. **#1414**, **#1462**, **#1378**, **#1496**,
**#1497**, **#1452**, **#1508**, **#1454** and **#1291** are each real and each
one session.

### Next planning action

1. **Trust no survey figure that has not been checked for `rows == distinct`.**
   This pass took four walks and two were corrupt — one of them in a way the
   recipe's own guard cannot see. **#1520** carries the fix, and until it lands
   every `/backlog`, every `/develop` WIP survey and every `gapaudit` run is
   reading a channel with an observed ~50% per-walk failure rate in this
   container. **This is the first stamp to measure that rate**; a second stamp
   should say whether it was a bad afternoon or the standing condition.
2. **The JOIN question is now FOUR stamps old, and this window moved it twice —
   #1507 changes what it can mean and #717 shows what it should require.** #717
   reconciled a 109-case census fall against a 107-case bank by naming the two
   that diverged and why, which is the shape a *mechanism* prediction has to take
   and the thing #1411 could not do. **A normative rule should require the
   reconciliation, not the number.** Against that, the method
   census-joined-to-`<lane>.txt` presumes a function from case to lane file.
   **There is no such function** for 1173 case IDs, and for 701 of them the two
   files disagree. So the normative rule #1512 and #1513 are about to write must
   be phrased against *the lane whose score the prediction is about*, not *the
   case's lane* — recorded on both threads. **The method itself stays unfiled**,
   deliberately, until #1507 rules.
3. **`schema` is at 95.4%, and M4's lane-visible residue is at most 23 issues
   against a lane residual of 706.** The count is now stated rather than gestured
   at (above). The next stamp should say whether the *remaining* 31 — 21 of them
   `kind/refactor` — belong in M4 at all, or whether M4's milestone has become a
   directory of `parser`/`xsd` work that outlived the parsing milestone.
4. **The persona consultation's marginal yield has fallen and the trigger should
   be re-examined — by `/retro`, not here.** Five findings, **four already
   filed**, one new, one dismissed, and **a third consecutive consultation with
   zero accuracy bugs in the CLI contract**. Two readings fit: the documentation
   debt is now well enough mapped that fresh eyes re-find it, or the personas are
   being run against a surface that has stopped changing. **Carried as a datum.**
5. **The human decision blocking #1002 is unchanged and is now carried for a
   TWENTY-FIRST stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — so no `/backlog` can move it and none should try.
   **#1507's arm A would need the same machinery**, which is the first time
   anything else in the queue has touched that decision.
6. **`/backlog` step 6's second duty has had no owner and this stamp is the
   first to act on it.** *"Fix any milestone scope paragraph that reality has
   outgrown"* has been in the standard throughout, and M4 grew to 220 lines and
   M5 to 202 under it — because the duty names no bound and no replacement rule,
   so every pass could honestly say nothing was WRONG. **#1522** fixes the
   sections; what it also has to fix is the missing rule, and the next stamp
   should check that the bound landed and not just the deletion.
7. **No `/retro` has landed since 2026-09-06** — the ninth weekly is now more than
   a week overdue and `git log` since carries no retro commit, so the routine has
   missed two cycles. **Items 1, 3 and 4 above are all routed to `/retro` and none
   of them can be answered by a `/backlog`.** This is an observation about the
   schedule, not a filing — `docs/ROUTINES.md` owns the cron and nothing in this
   container can read whether the routine fired.

**#717 landed concurrently with this pass and is folded in by MERGE, not by
survey.** Everything above was surveyed at `ca0ada5`; the lane table, the tip
citation, the marker census, the milestone and queue counts, the blocked set and
the band header were then **re-read at `1880f0b`** and corrected rather than
shipped a landing behind. Where a figure is the pass's own survey and has been
superseded, both readings are given and labelled. **No `/backlog` write was
re-run against the merged tip** — the reconciliation this pass made stands as
made, and #717's own post-land pass owns its follow-ups.

**What this pass did, so the next one can tell a completed run from a skipped
one:** **three issues filed** (#1520, #1521, #1522); **three body PATCHes** (#267's and
#345's `## Depends on` off the M5 epic, plus five stale line citations in #345;
#1516's malformed `suiteindex` query replaced with the measured census);
**two label changes** (#267 and #345, `blocked` → `ready`); **zero closures**;
and **NINE thread comments** (#267, #345, #1516, #1495, #1507, #1512, #1513,
#1499, #1153 for the reconciliation, plus #1454, #1291, #1381 and #1419 for the
persona dispositions). **Every body PATCH and comment went through
`gh api -F body=@FILE`** and is byte-faithful. **`gapaudit` was run TWICE**,
before and after the writes, and group 1 held at 18 both times. **`wipsurvey` was
run four times** and three of its rows changed verdict between runs, each
correctly.

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
