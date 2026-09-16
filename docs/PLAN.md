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

## Status — 2026-09-16 (`/backlog`, the twenty-sixth. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` after `git fetch --unshallow origin`, the marker census a fresh `gapaudit` run THREE times — before this pass's writes, after them, and after the persona dispositions, diffed row for row — and the milestone and queue counts a page-numbered `state=all` walk taken **after** all of them. **The window is SEVEN landings and NO lane moved at all.** Not one expectations file has changed since `1880f0b` on 2026-09-15; every landing was `kind/process`, `kind/tooling` or `kind/docs`, and each one's `Ratchet: unchanged` is correct rather than a miss. **The finding is WHY, and it is a measurement of the QUEUE rather than of the sessions**: of the **316** `ready` issues standing when it was taken, **87** carry a lane-visible kind, and of those **10** carry a `Ratchet:` clause that predicts or invites lane movement — and **the twenty-fifth band's twelve rows contained ZERO of the ten**. Its top SIX were consumed whole in about a day and its bottom six are untouched, which is what a band does when sessions take from the top. **This band is rebuilt around those ten**, and rows 1–3 are one vertical slice through `xsd/contentmatcher.go`'s decline list. **#1507 settled the denominator the last stamp said the tree disputed**: the `datatypes ⊆ instance` overlap is INTENDED and arm A was ruled against outright, so `instance` 11151/26361 is a figure to quote rather than hedge. **The marker census is FLAT** — 71 markers, 9 areas, group 1 at 18, group 2 at 31, one `dead end:` — identical across all three runs and identical to the twenty-fifth stamp's merged reading. **The branch namespace carries NO live claim** and has not before in this record: twelve `wip/*` refs, all RETIRED, zero `parked/*`, zero `needs-replan`, zero `meta/*`. **The survey channel behaved** — four full walks, four clean — which is the second data point on the corruption rate #1520 was filed for and reads "bad afternoon" rather than "standing condition". The **TWENTY-SIXTH persona consultation**: **eleven findings, ONE of them unowned** — **#1549 filed**, ONE dismissed as outside the library's remit, and **NINE already tracked, THREE of which had their tracker CORRECTED or WIDENED by what the outside view saw** (#1521's `-help` premise was false, #1381 was a line too narrow, #1407's table two codes too small), which is a re-discovery repairing its own tracker and is new in this record)

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

**This table is `main` at `bed0f7f`, and it is the committed expectations, which
`docs/WORKFLOW.md` names as the lane score (#1120).** **Every cell is byte-identical
to the twenty-fifth stamp's**, and that is a measurement rather than a copy: no
file under `conformance/testdata/expectations/` has changed since `1880f0b`
(#717, 2026-09-15). #1507 touched that directory in this window and touched
`README.md` alone.

**The column still does not sum to the corpus, and that is now SETTLED rather
than disputed.** The three lanes hold 41759 distinct case IDs against a column
summing to 42932 because every one of the 1173 `datatypes` cases is also
committed in `instance.txt`, 701 of them with a contradictory verdict. **#1507
ruled arm B: the overlap is intended.** `runLane` hands every lane the full case
list by design, the two lanes run differently mature executors over the same
fixture, and arm A — forcing disjoint lanes — was ruled against outright, because
it would move 468 committed `instance` passes downward and the one rule forbids
that. **So "the `instance` lane is 11151/26361" is a figure to quote, and the
twenty-fifth stamp's hedge against it is withdrawn.**

**No conformance measurement was taken this pass and `testdata/xsdtests` was NOT
initialized** — `git submodule status` reads `-7bc3365…`, uninitialized. Every
suite figure below is cited from the landing that measured it, never re-derived
here, and the `7bc3365` pin is read off the gitlink rather than off a checkout.

### The window: SEVEN landings, ZERO lane movement, and the band's top half consumed whole

`85b497b..bed0f7f` — seventeen commits: seven landings, eight post-land and
meta commits, one park log, one precondition-4 record. **Every landing in the
window has its own post-land pass on `main`**, so no follow-up is owed to this
one.

| commit | issue | ratchet |
|---|---|---|
| `8b6a440` | **#1512** | `CLAUDE.md`'s surveys block: a prediction joins the census through the lane's expectations file — unchanged |
| `48d28c5` | **#1507** | conformance lanes overlap **by design**, not first-match — unchanged (`expectations/README.md` only) |
| `16a01d1` | **#1520** | the Survey input recipe asserts page CONTENTS, not only page size — unchanged |
| `0605a39` | **#1495** | `suiteindex` censuses every namespace at once with `{*}` — unchanged |
| `2391176` | #396 | `typeDefinitionSlotsIdentical`'s stale `foldedAttributeUse` mechanism clause — unchanged |
| `e40175c` | **#1513** | `arbiter.md` rules the prediction bullet against the lane's expectations file — unchanged |
| `e15ee43` | **#1514** | `GOXSD_WITHHELD=1` lists the withheld-minus-banked case IDs — unchanged, **RUN rather than inferred** |

Plus `19fbede`, logging #1451's park at the third lost round.

**Five of the seven are one thread of work and it is finished.** #1451 was parked
at `docs/WORKFLOW.md`'s three-lost-round cap and re-planned into #1512, #1513 and
#1514; all three landed in this window. #1512 and #1513 state the join rule in the
two places a session arrives at it, #1514 builds the instrument that enumerates
the set the rule says to exclude, and #1507 removed the reason the rule could not
be phrased. **The remaining piece is #1545** — neither statement of the rule names
the instrument — filed by #1514's own post-land pass and band row 11.

**Zero lane movement is the correct outcome of these seven and is not a miss.**
Six are prose, tooling or process. The seventh, #1514, measured its ratchet rather
than inferring it (`GOXSD_RATCHET=1`, exit 0, `git status --porcelain` over the
expectations directory empty immediately after).

**The band's top SIX were consumed and its bottom six were not touched.** Rows 1–6
of the twenty-fifth band — #1507, #1520, #1495, #1512, #1513, #1514 — all landed.
Rows 7–12 — #1494, #1429, #1502, #1511, #267, #1324 — are all still open and
`ready`. #396 landed and was in no row at all. **A band is consumed from the top,
so what sits in its first three rows is the whole of its effect.**

### The queue holds ten issues that can move a lane, and the last band ranked none of them

Measured over all **316** `ready` bodies from this pass's verified feed, by
reading each body's `Ratchet:` clause. The classification is judgment over a
grep, not a tool, and is stated so it can be re-derived:

| population | count |
|---|---:|
| `ready` total when the measurement was taken | **316** |
| carrying `kind/gap`, `kind/bug` or `kind/feature` | **87** |
| of those, a `Ratchet:` clause predicting or inviting movement | **10** |
| of those, a `Ratchet:` clause stating `unchanged` / no lane can move | 60 |
| of those, carrying no `Ratchet:` clause at all | 17 |

The ten: **#1516**, **#782**, **#783** (one file, one decline list), **#1378**,
**#499** (the two `contentrestricts.go` ceilings), **#812**, **#1102** (M4),
**#888**, **#889** (M6/M7 value and XPath), **#1119**. **This pass's own two
filings do not move the figures**: #1548 is `kind/tooling` and #1549 `kind/docs`,
so neither enters the 87 and neither could enter the 10.

**The twenty-fifth band's twelve rows contained none of them.** That is not a
charge against that band — every row it carried was justified on its own clause,
and five of the seven landings discharged a forced re-plan — but it is the
mechanism behind a lane-flat window, and it is an ordering fact this file owns.
**This band is rebuilt around those ten and rows 1–6 are drawn from them.**

**The two clauses of step 4's ranking rule are not symmetric, and this pass
measured both in one pass.** The `kind/process` / `kind/tooling` friction clause
has a trigger a `/backlog` can read — the log records the cost in consecutive
sessions — and it selected the whole top of the last band. The `kind/refactor`
cost-of-delay clause has no such input: **80 open refactors, three bodies carrying
`## Cost of delay`, zero reading `Increasing`**, unchanged across two stamps and
fifteen landings. That is **#1499**, band row 12, and the measurement is recorded
on its thread rather than restated here.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against `git fetch --unshallow origin` and this
pass's verified post-write feed. No row came back CLAIMED or EXPIRED, so the
comments second pass `docs/ROUTINES.md` prescribes had nothing to fetch:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
434    wip/issue-434   122h9m0s   RETIRED  wip/issue-434: issue #434 is closed
732    wip/issue-732   573h3m0s   RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   751h1m0s   RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   523h22m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   717h3m0s   RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   main's     RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   main's     RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   528h43m0s  RETIRED  wip/issue-993: issue #993 is closed
1332   wip/issue-1332  101h17m0s  RETIRED  wip/issue-1332: issue #1332 is closed
1356   wip/issue-1356  76h19m0s   RETIRED  wip/issue-1356: issue #1356 is closed
1426   wip/issue-1426  49h22m0s   RETIRED  wip/issue-1426: issue #1426 is closed
1451   wip/issue-1451  21h17m0s   RETIRED  wip/issue-1451: issue #1451 is closed
```

**No live claim stands anywhere on the remote** — zero LIVE, zero CLAIMED, zero
EXPIRED, zero UNKNOWN. The whole `ready` queue is startable without colliding with
anything, which is a state this file has not been able to report before, and it is
the reason the band below is unusually safe to act on. **It is still a snapshot:
re-run `wipsurvey` before starting anything.**

**The supersede-or-verify duty is clean and the closure reasons were re-read this
pass rather than inherited.** Eleven of the twelve retired issues closed
`not_planned`, so nothing was owed on `main`. The one `completed` closure, **#1332**,
merged through PR #1441 and its residual was chased to the end by the twenty-fifth
stamp. **No supersede is owed anywhere**, and the set is the twenty-fifth stamp's
minus `wip/issue-717` and `wip/issue-1512`, both auto-deleted at merge.

**Zero `parked/*`, zero `needs-replan` open, zero `meta/*` — a third stamp.**
**SEVEN `claude/*` refs stand**, one of them this pass's own; the other six are at
**26, 88, 160, 172, 214 and 219 commits behind `main`, ZERO AHEAD in every case**.
Listed for human triage, not acted on; `wipsurvey` reads `wip/*` and `parked/*`
only.

**Two instrument notes, both caught inside this pass.**

- **`git branch -r` invented a branch that no longer exists.** The local
  remote-tracking cache still held `origin/meta/post-land-1514` after PR #1547
  merged and GitHub's auto-delete fired; `git ls-remote --heads origin` showed it
  gone. `docs/WORKFLOW.md`'s branch-scheme section names exactly this and says to
  read the remote. **The `meta/*` count above is `ls-remote`'s, not the cache's.**
- **`wipsurvey` printed a BORROWED lease age, twice, and nothing in the report
  said so.** The first run of this pass — taken two minutes after
  `git fetch --unshallow origin` and one minute after PR #1547 merged as `bed0f7f`
  — dated `wip/issue-933` at `645h3m0s` and `wip/issue-968` at `584h8m0s`. The
  second, twelve minutes later and after a plain `git fetch origin`, printed
  `main's` for both, which is what the twenty-fifth stamp read. No ref changed
  between them: `remoteRefs` takes `main`'s SHA live from `ls-remote` while the
  ancestry test resolves it against the local object store, and a landing between
  the fetch and the survey leaves a SHA the checkout cannot resolve. `classify`
  answers retirement before the `ancestryUnresolved` marker is built, so a RETIRED
  row's borrowed age carries no marker at all. **That is #1548**, filed here; it is
  #806's own HYPOTHESIS reproduced in the wild, and it also falsifies
  `docs/ROUTINES.md`'s *"one call, after which every reading is trustworthy"* —
  a landing arrived mid-pass in **two consecutive stamps**. **Measured cost: zero**,
  because both affected rows were RETIRED, where nothing acts on the age.

### Marker census — 71 markers, NINE areas, group 1 at 18, group 2 at 31, ONE dead end

**`go tool gapaudit` was run THREE times — before this pass's writes, after them,
and again after the persona dispositions** — each against a full-repository feed as
`docs/ROUTINES.md` requires, each from a walk verified `rows == distinct` with no
gaps.

- **Pre-write, `bed0f7f`: 71 markers across 9 areas, group 1 at 18, group 2 at 31.**
- **Post-write, `bed0f7f`: 71 / 9 / 18 / 31.**
- **Post-persona, `bed0f7f`: 71 / 9 / 18 / 31.**

**All three runs are identical row for row**, diffed rather than eyeballed — the
eighteen group-1 paths in the same order and the same **262** annotations. This
pass edited five issue bodies and two titles and filed two issues, and `gapaudit`
resolves its candidate owners against open titles and bodies, so the diff is the
check that none of those edits moved a marker's ownership.

Per area, flat against the twenty-fifth stamp: `xsd` 33, `validate` 15, `xpath` 6,
`parser` 5, `xml` 4, `value` 3, `conformance` 2, `regex` 2, `cmd` 1.

**Group 1 is the same eighteen sites the twenty-fifth stamp judged**, and the
judgments stand: `parser/doc.go:219` and `xsd/resolve.go:681` sit there **by
design**, their §5.3 provenance deliberately spelled without a `#` sigil so
`gapaudit`'s bare-`#(\d+)` match cannot read it, and filing an owner for them is
what CLAUDE.md's closing sentence reserves to a human. **The one `dead end:`**
remains `xsd/contentrestricts.go:681` citing CLOSED #501, and **#1378 owns its
retirement under either arm of its own body** — which is band row 4 this stamp,
so the annotation has a live route out for the first time.

**Zero untracked GAP sites** — a judgment on the annotations, not the tool's
mechanical rule.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** every write
this pass made, the persona dispositions included, **verified
`rows 1549 distinct 1549 min 1 max 1549 no gaps`**: **1549 rows, 694 PRs excluded,
855 issues — 331 open, 524 closed.** All four walks this pass took came back clean
on their first request.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **134** | active |
| **M5 — Instance validation (XML)** | **17** | **25** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **317 `ready`, 12 `blocked`, 0 `needs-replan`, 2 `epic`**
— summing to 331 with no gap, and **every open issue carries exactly one, verified
over all 331 by grouping each issue's own label set rather than by summing
counts**. **#779** owns the mechanical check.

By kind: `kind/refactor` 80, `kind/process` 64, `kind/gap` 61, `kind/tooling` 43,
`kind/story` 35, `kind/bug` 33, `kind/docs` 29, `kind/feature` 3. By area: `meta`
92, `parser` 83, `xsd` 63, `docs` 44, `conformance` 35, `cmd` 30, `validate` 28,
`value` 17, `builtin` 11, `xsderr` 9, `regex` 6, `xpath` 6, `model` 4, `loader` 2,
`cli` 1. **259 of 331 open issues carry no milestone at all**, so read the `area/`
census, not the milestone counts.

**M4's composition is unchanged to the issue, which is the answer to the last
stamp's question and not a restatement of it.** Of M4's **54** open, **at most 23
are lane-visible** — 15 `kind/gap`, 7 `kind/bug`, 1 `kind/feature` — against **31**
(`refactor` 22, `docs` 6, `tooling` 2, `process` 1) that move no lane by
construction. Both figures are byte-identical to the twenty-fifth stamp's. **M5 is
17 open: at most 11 lane-visible** (8 `gap`, 2 `bug`, 1 `feature`) against 6.
Neither milestone moved this window, because none of the seven landings carried
one.

**`ready` stands at 317 — the twenty-fifth stamp's own figure, over a set that has
turned over by fourteen.** SEVEN closed with this window's landings (#396, #1495,
#1507, #1512, #1513, #1514, #1520) and SEVEN were filed into it: five by those
landings' post-land passes (#1531, #1536, #1539, #1545, #1546) and **two by this
pass** (#1548, #1549). That a churn of fourteen nets to zero is #347's shape
exactly — `ready` is an output, not a target, and an unchanged total is not a
still queue. With **no live claim anywhere**, all 317 are startable without
collision.

**The M4 and M5 SCOPE SECTIONS below are still not fit to read, and PLAN.md was
untouched by all seven landings, so they are exactly as #1522 found them.** M4 is
220 lines carrying 23 "has now LANDED" clauses and M5 is 202; each is a running
chronicle of individual landings against this file's own preamble. **#1522** owns
the reduction and the missing replacement rule. Not fixed here for the reason the
last stamp gave and this pass re-confirmed: a 400-line prose deletion needs a
per-clause recoverability check and its own reviewable commit.

**The unblock sweep measured a clean ZERO**, over all twelve pre-write `blocked`
bodies' `## Depends on` sections, read byte-faithfully from REST. **No entry names
#1512, #1507, #1520, #1495, #396, #1513, #1514 or #1451.** The set is unchanged at
**12**.

- **FIVE are triggers rather than issues** and say so in their own
  `## Depends on` — **#1374**, **#1224**, **#555**, **#1002** and **#1042**.
  **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **FIVE still have exactly one OPEN named dependency each, and every one of those
  five targets is `ready` today** — **#593** (#591), **#731** (#1414), **#871**
  (#831), **#1344** (#1324) and **#1458** (#479). **Five landings clear five
  `blocked` issues**, and none of the five moves a lane, which is why they are
  named below the band rather than in it.

**The cycle check was run and the answer is NO.** Over all twelve `blocked` bodies,
every named dependency target is itself `ready` — never `blocked`, never `epic` —
so the graph is depth-1 and carries no mutual pair. **And the epic-as-gate disease
the twenty-fifth stamp cured has not recurred**: no `blocked` body names #250 or
#79. #79's title still claimed *"the producer/finalize dependency 5 blocked issues
point at"* while none does; **corrected here**, since its own body has recorded all
five as closed since 2026-08-18.

**The `ready` set was audited mechanically this stamp, and it is clean.** All 316
bodies carry a `## Depends on` section. **66 named an OPEN `#N` somewhere inside
it** — which is the heuristic #779 exists to replace, not a defect — and **65 of
the 66 opened with "none" or "n/a" followed by an explicit not-gating clause**
("coordinates with", "adjacent, not a prerequisite", "sequence-related, not
gating"). **ZERO mislabels.** The one exception was **#622**, whose
`## Depends on` still explained what *"`blocked` here means"* though the issue was
relabelled `ready` on 2026-08-23 under its own stated fallback, with the relabel
recorded on its thread. **Corrected here** to say the fallback already fired,
which is also why the audit re-run against the post-write feed reports 65.

**#1019 was corrected on a premise this pass falsified.** Its Acceptance table said
four documented `area/` labels *"have never been used, not once"*. Two of them now
are: **`area/model` carries 4 open `ready` issues** (#841, #1284, #1285, #1287) and
**`area/cli` carries 1** (#1283), all filed by the 2026-09-06 architecture audit.
A session taking that issue's "strike them" arm today would delete labels five open
issues wear. The table now carries both measurements side by side, the five issues
are named, and the `kind/docs` enumeration — *"the ten are …"*, now **37 of which
28 open** — is replaced with an instruction to enumerate from the queue.

### Persona consultations — the TWENTY-SIXTH, and ONE finding in eleven had no owner

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **The orchestrating session ran both personas fresh against the
published surface** — `libuser` on README plus `go doc` only, `cliuser` on README
plus `-help` plus a binary built from the tree — and handed the reports here.
**Every finding was re-derived here before disposition**, the CLI ones against a
binary built at `bed0f7f`.

| finding | disposition |
|---|---|
| `libuser`: the README's validate snippet warns on `res.Err()` and never reads `Result.Unevaluated()`, which `validate/doc.go` says is also not a pass | **ALREADY FILED — #1122**, and this is the **FOURTH** independent run to reach it. Re-derived: both quoted sentences byte-identical, only the line numbers moved (`doc.go:101-105`→`:102-105`, `README.md:281-295`→`:311-339`), and `grep -n Unevaluated README.md` still returns nothing. **The new argument is about scope, not the defect**: the persona read `Unevaluated`'s own godoc and found the type already built to prevent this — no `Error` method, unexported fields, *"must not grow one"* — so the fix is wholly in `README.md` and a session must not reach for the API. Recorded on the thread |
| `libuser`: nothing says what an HTTP-facing consumer should DO when `Unevaluated()` is non-empty and `Violations()` is empty | **DISMISSED, out of the library's remit**, and recorded on **#1292** so it is not re-filed. The processor's obligation ends at reporting that the document stands undecided against exactly the rules named; which status code a caller maps that to depends on the caller's risk posture and on nothing in XSD 1.1. **The arrival half is not dismissed** — the persona reached the warning a hundred lines from the accessor it was calling, which is direct corroboration for #1292's arm (b) over its arm (a) |
| `libuser`: no multi-root entry point, and `go doc ./loader` reads as if there were one | **ALREADY FILED, and it is a PAIR** — **#1283** (export multi-root assembly; `cmd/goxsd8` synthesizes the wrapper as a string) and **#1286 item 4** (`loader/doc.go:35` states two capabilities the library does not export, in the present tense). Both verified present. **The persona reached both halves from outside the tree**, where #1286 was filed by an architecture audit that could see it; its account of the cost is the one the audit could not supply, and is recorded on #1286 |
| `libuser`: `xsderr` exports no compiler-checked rule catalog | **ALREADY FILED — #1454**, and the **second consecutive** consultation to re-derive it. **No new argument this time** — the twenty-fifth recorded the measured one (`catalog.go` 217 → 224 over five landings), and nothing in this window touched it, so nothing was posted |
| `libuser`: the s4s-grammar class carries no `Rule` and no structured `Loc`, so it forces string-scraping | **ALREADY FILED — #1371**, whose Goal is this sentence: a caller can tell the class without matching English prose, or the surface says outright that it cannot. Disclosed in `xsderr`'s own doc, which the persona correctly read as disclosure rather than as a hidden defect |
| `libuser`: `parser.Parse`/`Produce`/`ParseReport` return only the FIRST error | **ALREADY FILED — #1005**, which owns the promise itself: `parser/doc.go`'s *"PLANNED (not yet implemented): collecting them in document order"* carries no `GAP(` marker and no owning issue, and the issue is to decide the promise or withdraw it. The persona graded it *"disclosed, but a real capability gap"*, which is exactly the state #1005 records |
| `libuser`: a parse-time/validate-time backend mismatch is undetectable and no capability lets a caller assert it | **ALREADY FILED — #1382**, verbatim the same invariant: *"enforced by CONVENTION ONLY — `xsd.Schema` carries no record of the value space it was finalized with."* The persona graded it a defensible consequence of `Value = any`; #1382's arm (c) is that ruling, written down |
| `cliuser`: schema violations cite an absolute path while instance violations echo the argument as spelled | **ALREADY FILED — #1521** (the twenty-fifth filed it) — **but the persona falsified one of its premises and the body is corrected here.** #1521 said the contract is *"printed by `goxsd8 -help`"*. It is not: `-help` prints `main.go`'s `usage` constant, a different and shorter text, and `grep -c 'resolved to an absolute' cmd/goxsd8/main.go` is **0** while `doc.go:160` carries it — with `README.md:194-198` recording the omission as deliberate. **So arm (a) as written would land the new clause where a binary-only CI operator never reads it**, which is the #1279 error one level up; an Acceptance bar now makes arm (a) rule on that explicitly |
| `cliuser`: `parse`'s counts silently drop anonymous types **and local element declarations** | **ALREADY FILED — #1381 — and WIDENED here.** #1381 owned the anonymous-type half on the `types:` line; the persona carried it to `elements:`, a different line by a different mechanism (scope, not namelessness). Reproduced: a schema holding two type definitions and three element declarations reports `types: 1`, `elements: 1`, `components: 2`. Both arms now have to cover both lines — `top-level` is the right qualifier for `elements:` and the wrong one for `types:` — and the pinning test takes a third schema that exercises both omissions at once |
| `cliuser`: the exit-code severity order is stated for 4-vs-1 only, and a usage/IO fault co-occurring with an invalid instance exits 2 | **ALREADY FILED — #1407 — and WIDENED here.** #1407 asked the contract to lead with a five-row table annotating the rank of 4 and 1; the persona measured the two facts that make that insufficient. Both reproduced: a run whose arguments are good, unreadable and invalid **does not abort** at the unreadable one — the later instance is still assessed and its violation printed — and the run exits **2**. The order the binary implements is `0 < 4 < 1 < 2` with 3 reserved, and only the `4 < 1` step is written anywhere. The table must now rank every code that can co-occur. **No verdict changes**; every code returned was correct |
| `cliuser`: a not-well-formed INSTANCE prints as an ordinary violation and exits 1, and nothing in README or `-help` says so | **FILED — #1549, the consultation's one unowned finding — and sharpened past what the persona could see.** Reproduced (`[xml-wf]` on stdout, exit 1). **The defect is worse than silence**: the *only* occurrence of "not well-formed" anywhere in the CLI surface (`doc.go:207`, `README.md:121`) is about a **hinted schema document**, and routes that case the opposite way — *stderr, no rule ID, no move in the exit code*. A reader who searches the contract for the phrase finds the one sentence there is and is taught the contrary answer for the case they are in. The Acceptance therefore bars the obvious siting beside `:207` |

**Eleven findings: ONE filed, ONE dismissed, NINE already tracked.** That is a
second consecutive consultation whose majority was re-discovery, and a fourth
consecutive one finding no accuracy bug in the CLI contract — every verdict, exit
code, stream and message shape the `cliuser` probed matched the contract, and it
probed a long list. **`libuser`'s Story C confirms a disclosure rather than
breaking one**: `go doc -short ./validate/jsonsrc` and `./validate/bersrc` both
print nothing, which is what those packages' own doc comments say (M8, M11), so
the JSON story is honestly unbuildable rather than quietly broken.

**The naive reading of "eight of eleven were already filed" is wrong, and this is
the first stamp able to say so.** THREE of the nine tracked findings corrected
the tracker they re-found: **#1521** asserted the contract is printed by
`goxsd8 -help` and it is not, **#1381** owned one line of a seven-line block where
the same silence costs two, and **#1407** asked for a table annotating two codes
where five can co-occur. **A re-discovery that repairs its own tracker is not a
wasted run** — an issue filed from inside the tree carries the filer's knowledge
of the tree as an unstated premise, and only an outside reader trips over it.

**And the two findings the personas moved forward were moved by WIDENING existing
issues, not by filing beside them** — six `cmd` issues already edit
`cmd/goxsd8/doc.go` and `README.md`'s CLI section, and a seventh and eighth would
have bought two more landings rebasing the same two files. **Whether the
consultation's trigger should change is `/retro`'s to decide**, and it should be
decided against the repair rate rather than against the filed-versus-re-found
ratio alone.

### Working band

Ordered for a `/develop` session: take the highest row you can start. **No claim
stands anywhere on the remote**, so every row below is startable today — but
**re-run `wipsurvey` before starting anything**, because that is a snapshot and
because a landing arrived mid-pass in each of the last two stamps.

**The ordering principle changed this stamp and the change is the deliverable.**
Rows 1–6 are drawn from the ten `ready` issues whose `Ratchet:` clause can move a
lane; the twenty-fifth band contained none of them, and the window it governed
moved no lane. Rows 1–3 are one vertical slice through a single decline list.

| # | issue | why here |
|---|---|---|
| 1 | #1516 | **The largest lane slice the queue holds, and the first issue that can apply the whole instrument chain this window landed.** `{open content}` interleave and suffix — cvc-complex-content clauses 2 and 3 — retiring a `GAP(xsd)` and the `GAP(validate)` #717 filed against it. Its `Ratchet:` clause names `instance` and carries a **measured population bound** (`openContent` 75 occurrences in 51 fixtures, `defaultOpenContent` 29 in 29) that the body itself marks as a bound and **not** a prediction. Turning that bound into a prediction is exactly what #1512's join rule and #1514's `GOXSD_WITHHELD=1` are for, and nothing has used them yet. **Initialize the submodule first** (#659) |
| 2 | #782 | **Same file, same decline list, same lane.** A repeatable particle nested inside a repeatable particle — cvc-accept's own named non-determinism. #1516's body orders these three explicitly and **forbids widening the other two**, so the three do not collide and either order works between rows 2 and 3. `Ratchet: measure, do not predict` — the body says why, and #1514's instrument is what makes that honest rather than an excuse |
| 3 | #783 | The third decline: an `all` group holding a nested `all` group, cos-all-limited clause 2's **one legal nesting**, and the per-member positions the counting walk does not keep. M5 |
| 4 | #1378 | **The only `dead end:` in the marker census has a live route out and this is it.** `contentTypeRestricts`' `maxContentPositions` ceiling, whose marker cites CLOSED #501 — and **SIX W3C suite content models actually reach it**, so the incompleteness is live rather than latent. `Ratchet: the schema lane moves up or down`, which is the shape a measurement should have and the opposite of a promise. Arm (b) rewrites the marker to cite this issue; arm (a) deletes it |
| 5 | #812 | **M4, and the largest unmeasured `schema` candidate in the queue.** `c-selector-xpath` and `c-fields-xpaths` are unimplemented, so an identity-constraint expression outside the §3.11.6.2/§3.11.6.3 subset reaches `validate` and declines the **whole** constraint at runtime. `Ratchet: candidate schema movement, UNMEASURED` — measure it at grounding before scoping |
| 6 | #1102 | **M4, and the one row here whose `Ratchet:` clause promises a MEASURED figure.** `ownAttributeUses` is a proven over-approximation since #1082, not a recovery, and the unowned `GAP(xsd)` at `attributeusefold.go:296` is **fail-CLOSED** at `checkAttributeRestrictionRequired` — the direction that costs a rejection rather than withholding one. Read **#1539** first: it corrects the same file's doc comment about exactly this asymmetry |
| 7 | #1494 | M5: an unresolvable `xsi:type` exits 0, silent. **Oracle first** — do not implement a charge before the rule ID exists, and *"the spec charges nothing"* is a legitimate outcome. It has been banded twice without being taken; it now has the lane prediction it shipped without, since #1495 and #1514 both landed |
| 8 | #345 | **The re-rank the twenty-fifth stamp instructed this one to make.** Its arm (a) was waiting for an assessment-time wildcard-attribution consumer; **#717 landed one**, so the premise the last band held it down on has fired. Arm (b) is the documented-permanent-fail-open ruling its body has offered since 2026-07-31. Five stale `contentrestricts.go` citations were corrected to `:1099` last pass, so the body is startable as it stands |
| 9 | #267 | Its sibling, and **Acceptance bullet 1 is oracle exegesis over §3.10.4.3 needing no M5 code**. Unblocked off the M5 epic last pass after seven weeks. **Read #717's thread first**: it is the first assessment-time wildcard-attribution consumer in the tree and arm (a)'s premise moved under this issue's feet |
| 10 | #1546 | **The cheapest real friction fix in the queue, and its evidence is three agents in ONE thread.** `runner.go`'s "# Case IDs" section explains the segment that is *not* load-bearing for cardinality and is silent on the one that is; mason said six, the verdict said thirteen, the chronicler counted fourteen, off the same printed list. Scoped to that one section and **explicitly excluding the process half** |
| 11 | #1545 | **The last piece of #1451's re-plan.** The join rule is now stated in two places and has an instrument, and **neither statement names it**. Filed as ONE issue where #1512 and #1513 were two, because those decided what the rule SAYS and this only points at `GOXSD_WITHHELD=1` twice. The arbiter's non-blocking finding 2 rides as an explicitly OPTIONAL arm |
| 12 | #1499 | **Named below the band TWICE and moved into it here.** Two stamps a week apart measured the `kind/refactor` cost-of-delay clause selecting **nothing** — 80 refactors, three bodies carrying the phrase, zero reading `Increasing` — and neither could act on it, because a refactor moves no lane and costs no per-session friction, which is the exact starvation this issue describes. It ranks the largest kind in the queue and nothing else will lift it |

**Named below the band, deliberately.** **#1414**, **#831**, **#591**, **#1324**
and **#479** each discharge exactly one `blocked` issue on landing — #731, #871,
#593, #1344 and #1458 respectively — and **none of the five moves a lane**, three
saying `Ratchet: unchanged` and predicting exactly that. **Five small landings
clear five `blocked` issues**, which is the best value in the queue for a session
that wants a clean close; **#1324** is the cheapest of them, a ruling that
discharges #1344 in either direction. **#499** is #1378's twin — the
`maxProductStates` ceiling, unreached by the whole suite, so latent where #1378's
is live. **#888**, **#889** and **#1119** are the other movement-capable bodies and
each wants M6/M7 machinery that does not exist yet. **#1536** (`suiteindex` answers
a confident 0 on a non-NCName local name) matters more than its size suggests now
that `suiteindex` is the *prescribed* prediction instrument; **#1531** had its
citations corrected by #1514's post-land pass and is freshly startable; **#1548**
is this pass's own filing and its measured cost is zero. **#1511** and **#1462**
are the `no-xmlns`/`no-xsi` component-footing pair and neither moves a lane by
construction. **#1502** is #1498's last unchecked suite claim, comment-only.
**#1522** (the 400-line M4/M5 prose) and **#1153** (the survey recipe has no test)
are both compounding and both untouched for two stamps. **#1429**, **#1539**,
**#1518**, **#1473**, **#1474**, **#1452**, **#1496**, **#1497**, **#1508**,
**#1454** and **#1291** are each real and each one session.

**The `cmd` contract family is now SIX issues over two files and should be taken
as one session, not six.** **#1521** (instance vs schema path spelling),
**#1381** (the `parse` counts, widened this stamp), **#1407** (the exit-code
table, widened this stamp), **#1549** (the `xml-wf` charge, filed this stamp),
**#1368** and **#1419**/**#1421** all edit `cmd/goxsd8/doc.go` and `README.md`'s
CLI section, and four of the six now carry a persona reproduction. Six separate
landings would rebase the same two files six times, and #1407's table is the
structure three of the others want to hang a clause on — **so whoever takes
#1407 should read #1549 and #1521 in the same sitting**, and the three
exit-code-adjacent ones (#1407, #1549, #1419) are the natural bundle.

### Next planning action

1. **This band's whole claim is falsifiable in one window: rows 1–6 can move a
   lane and the last band's could not.** If the next window is lane-flat *again*
   with this band published and unconsumed at the top, the defect is not the
   ordering and the next stamp should say what it is instead. If it is lane-flat
   because the top rows were taken and measured `unchanged`, that is a finding
   about the ten and belongs on their threads.
2. **The JOIN question is DISCHARGED as a question and is now a filing.** Four
   stamps carried it. #1512 and #1513 landed the rule at both arrival paths, #1514
   built the instrument, and **#1507 removed the obstacle** — the overlap is
   intended, so the rule can be phrased against *the lane whose score the
   prediction is about* without presuming a function from case to lane file.
   **#1545 is the remainder** and is band row 11. The next stamp should check that
   a prediction has actually been made through the chain, not merely that the
   chain exists; **#1516 is the first issue positioned to make one**.
3. **M4's composition is unchanged to the issue across two stamps** — 54 open, at
   most 23 lane-visible, 31 that move no lane — so the twenty-fifth stamp's
   question stands unanswered: whether the remaining 31, 22 of them
   `kind/refactor`, belong in M4 at all or whether M4 has become a directory of
   `parser`/`xsd` work that outlived the parsing milestone. **That is a `/retro`
   question and no `/backlog` can answer it.**
4. **The survey channel's failure rate has a SECOND data point and it reads "bad
   afternoon".** The twenty-fifth stamp took four walks and two came back corrupt;
   this one took **four and all four were clean on their first request**, with
   #1520's page-content assertion now in the recipe. One clean stamp does not
   settle a rate. **A third stamp decides**, and until then quote the
   distinct-number stamp wherever a survey figure is quoted, which is what
   #1520 landed.
5. **The persona consultation's yield is now measured twice and the trigger
   should be re-examined by `/retro`, not here.** Eleven findings, eight already
   filed, one dismissed, one new — a second consecutive consultation whose
   majority was re-discovery, and a **fourth** consecutive one finding no accuracy
   bug in the CLI contract. **But the naive reading of that is wrong and this
   stamp is the first able to say so**: three of the re-discoveries corrected a
   premise or a scope in the issue they re-found (#1521's `-help` claim was
   false, #1381 was a line too narrow, #1407's table was two codes too small).
   A re-discovery that repairs its own tracker is not a wasted run, and whatever
   `/retro` decides about the trigger should be decided against that, not against
   the filed/re-found ratio alone.
6. **The human decision blocking #1002 is unchanged and is now carried for a
   TWENTY-SECOND stamp.** It waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent, so no `/backlog` can
   move it and none should try.
7. **#1522 is untouched and `docs/PLAN.md` was untouched by all seven landings**,
   which is the replacement rule working exactly as written (#879) and also why
   the M4 and M5 scope sections have not improved. The next stamp should check
   that the missing BOUND landed with the deletion, not only the deletion.
8. **No `/retro` has landed since 2026-09-06, and that is ONE missed cycle, not
   two** — correcting the twenty-fifth stamp. 2026-09-06 and 2026-09-13 are both
   Sundays; the 09-13 run did not land, and the next is due 2026-09-20. **Items 1,
   3 and 5 are all routed there and none can be answered by a `/backlog`.** This
   is an observation about the schedule, not a filing — `docs/ROUTINES.md` owns
   the cron and nothing in this container can read whether the routine fired.

**What this pass did, so the next one can tell a completed run from a skipped
one:** **TWO issues filed** — **#1548** (the `wipsurvey` borrowed lease age) and
**#1549** (the `xml-wf` charge the CLI contract contradicts). **SIX body PATCHes
and TWO title PATCHes**: #1019's falsified "never used" table plus its stale
`kind/docs` enumeration; #622's stale `blocked` paragraph; #79's stale title;
#1381 widened to `elements:` and local declarations, title included; #1407 widened
to the full severity order; #1521's falsified `-help` premise corrected and a bar
added to its arm (a). **FOUR thread comments** — #1499's re-measurement, and
#1122, #1292 and #1286 for the persona dispositions. **ZERO closures and ZERO
label changes**: the unblock sweep measured zero, the `ready` audit found no
mislabel, and every persona finding that was already tracked was dispositioned by
correcting its tracker rather than by filing beside it.

**Every body PATCH and comment went through `gh api -F body=@FILE`** and is
byte-faithful; every body READ came from REST rather than the MCP channel (#764).
`gh api repos/kud360/goxsd8/...` served every read and every write; GraphQL is
still a 403, exactly as `docs/ROUTINES.md` records. **`gapaudit` was run THREE
times** — pre-write, post-write and post-persona — and all three are identical
row for row. **`wipsurvey` was run twice** and two of its LEASE AGE cells differed
between the runs, which is #1548. **Four `state=all` walks, four clean.**
**`testdata/xsdtests` was NOT initialized**, so no suite figure here is this
pass's own measurement; the CLI reproductions were run against a binary built from
`bed0f7f`.

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
