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

## Status — 2026-09-06 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **SIX** landings: the last band's rows **1 through 5** (#1239, #1243, #1242, #1140, #1205) plus **#1050 from outside the band** — so the in-order run ended at five, rows 2 and 3 were taken in the reverse of the order the band advised, and one slot went to an unbanded issue. It measured **`schema` +5, all of it #1205's**. **The prediction story inverted and that is this stamp's finding**: the three landings that censused by CONSTRUCT predicted correctly, and the ONE that under-predicted — #1205, filed and banded *"ratchet-neutral by construction"* and banked **+5** — is the one whose prediction predated the instrument. **#1239 shipped that instrument and #1205's session bypassed it**, both agents hand-writing a namespace-aware parent-classifying walk at **n=139**, because `suiteindex` reports no PARENT and local-vs-top-level is inexpressible at any number of queries. **#1282 is that one missing field and it is band row 1.** The namespace is **idle for the third consecutive stamp** (zero LIVE, zero CLAIMED) and carries **two maintenance branches**, one of which stands unmerged with no PR. The marker census moved **71 → 69**. **A QUEUE-LABEL DEFECT was found and repaired**: #79 and #250 wore `blocked` AND `epic`, which `docs/WORKFLOW.md` forbids, while the last stamp asserted the opposite mechanically — the queue is now clean and #779 carries the fifth class. The **SEVENTEENTH persona consultation**: **seven findings — four filed (#1290, #1291, #1292, #1293), three dismissed with reasons on the threads that own them**)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13942 | 1456 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### Five band rows landed, one landing came from outside the band, and the one under-prediction is the one that predated the instrument

**`schema` 13937 → 13942 (+5); `instance` flat at 11029; `datatypes` flat.**

| landing | commit | predicted | banked | milestone |
|---|---|---|---|---|
| **#1239** | `9b27bb6` | unchanged (`go list -deps ./conformance` names no `tools/` package) | unchanged | none |
| **#1243** | `3febb22` | unchanged, **by construct census** (404/170, 22 `prohibited` across 10 fixtures, read decoded) | unchanged | **M4** |
| **#1242** | `6e95f25` | **refused a figure on purpose**; censused by construct | unchanged | **M4** |
| **#1050** | `70850cd` | unchanged (doc-only) | unchanged | none |
| **#1140** | `b9d3c88` | unchanged (doc-only) | unchanged | none |
| **#1205** | `24ac7d3` | **"ratchet-neutral by construction"** | **`schema` +5** | **M4** |

**Rows 1 through 5 of the last band landed; row 6 (#1260) did not, and #1050
was taken from outside the band entirely.** The previous stamp recorded six
in-order rows and *"nothing was inserted ahead of it"*; this window is five,
with the 2/3 pair taken in the reverse of the order row 3's own text advised
(*"the harder of the pair — take it second"*). Both landed clean, so the
inversion cost nothing measurable and is recorded rather than charged.

**Three of the four ratchet predictions that used a construct census were
right, and the fourth refused to predict — which is the census discipline
working.** #1243 pre-measured 404/170 with 22 `prohibited` occurrences across
10 fixtures, all read decoded, and banked exactly the `unchanged` it named.
#1242 declined a figure on purpose and banked `unchanged`. That is the first
window in this family with **no** unexplained ratchet figure to account for.

**The exception is #1205, and it is the sharpest datum this stamp has.** It was
filed, banded and carried forward as *"ratchet-neutral by construction"* — a
prediction written before `suiteindex` existed — and it banked **`schema` +5**:
five `fail` → `pass` flips, every one a local `<element>` carrying `abstract=`
or `final=` (`errC005`, `errC006`, `particlesDa011`, `schZ001_78029-a`,
`schZ002_78029-b`). **Nothing leaked**: the issue's own Acceptance made
measurement a precondition, and that is what caught it before banking. The
mitigation is not the problem.

**The problem is that the instrument shipped and was bypassed on the very next
lane mover.** #1239 landed `go tool suiteindex` as band row 1 precisely to
retire the narrow greps that under-predicted three landings running. #1205's
session ran a census — and **neither the mason nor the arbiter account records a
`suiteindex` run**. Each wrote its own namespace-aware, parent-classifying walk
over the whole submodule, and they reconciled at **139 / 133 / 6 / 0**. The
reason is structural and one field wide: `suiteindex`'s `hit` record is
`{File, Line, Col, Values}` (`tools/suiteindex/main.go:217-225`) and
`scanFixture` (`:370-396`) discards `EndElement`, so it keeps no element stack
and **local-vs-top-level is not expressible at any number of queries**. That is
**#1282**, and it is why a `kind/tooling` row sits above two lane slices again.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and
this pass's 728-issue post-write feed:

```
ISSUE  BRANCH         LEASE AGE  VERDICT  REASON
732    wip/issue-732  333h11m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  511h9m0s   RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846  283h31m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872  477h11m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  405h24m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  344h29m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993  288h51m0s  RETIRED  wip/issue-993: issue #993 is closed
```

**ZERO LIVE and ZERO CLAIMED for the THIRD consecutive stamp**, over six
landings: `wip/issue-1239`, `-1243`, `-1242`, `-1050`, `-1140` and `-1205` were
all opened and all deleted at merge inside the window. **Six leases opened and
closed cleanly and none survived to be counted here.** Every band row below is
startable without colliding with a claim, and `ready` overstates startable work
by **nothing**.

**The same seven RETIRED refs, unchanged row for row** — four stamps now. All
seven closed `not_planned`; they are parks and supersedes whose content is
*supposed* not to be in `main`, and none owes a supersede. Cloud containers
cannot delete remote refs, so these accumulate by design and are not a finding.
**Zero `parked/*`.**

**Two `meta/*` maintenance branches appeared this window, and one of them is a
finding.** `docs/WORKFLOW.md` says a maintenance command *"opens and
squash-merges its PR in the same session, so it holds no lease and never
appears in a survey"* — `wipsurvey` reads `wip/*` and `parked/*` only, so
neither is in the block above and both were found by `git ls-remote --heads`.

| ref | tip | reads |
|---|---|---|
| `meta/retro-2026-09-06` | `6497c08` → merged | **LANDED as `f97c317` during this pass**; branch auto-deleted. PR #1289 |
| `meta/audit-2026-09-06` | `55a278b` | **STANDS UNMERGED WITH NO PR**, two commits, `docs/ARCHITECTURE.md` only |

**The audit branch is the finding, and it is not idle debris.** It carries the
2026-09-06 steward architecture audit's eight corrections to
`docs/ARCHITECTURE.md`, and **#1285 and #1287 — filed by that audit and open in
the queue today — both cite corrections that are not on `main`.** #1285's Notes
say *"the 2026-09-06 audit corrected the ARCHITECTURE.md attribution"*; at
`f97c317` it has not. Listed for the orchestrating session and for human
triage, not acted on: a session never deletes or force-pushes a ref, and it is
not this pass's to land.

**FOUR non-`wip` `claude/*` refs stand, unchanged in tip from the last stamp**
— `eloquent-cerf-39rk64` (`0abeab6`), `-3xu0ki` (`62d5143`), `-8jq9o6`
(`7841e98`), `-adewly` (`5d03049`). The two NOT-ancestor readings are the
shallow-clone artefact **#802** owns and were dispositioned from commit dates.
Listed for human triage, not acted on.

### Marker census

`go tool gapaudit` over this pass's whole 728-issue post-write feed: **69
markers across 8 areas** — `xsd` 33, `validate` 17, `xpath` 6, `xml` 4,
`parser` 3, `value` 3, `conformance` 2, `cmd` 1.

**The census moved 71 → 69, and both retirements are landings paying their own
debt.** `xsd` went 35 → 33 while `parser` held at 3: the two `[xsd]`-tagged
markers #1216 left in `parser/produce_complex.go` are gone, deleted by #1243
and #1242 — the two issues #1216's post-land filed against them. That is the
full cycle, marker to tracker to deletion, inside two windows.

**Group 1 moved 19 → 18 and group 2 held at 27.** **ZERO group-1 rows carry no
annotation at all, so the tool's own filing rule selects nothing and there are
ZERO untracked GAP sites — EIGHTH consecutive stamp.**

**One group-1 row is still asserting a falsehood in the tree, and it is band row
2's to delete.** `parser/produce_complex.go:3125` ends *"No open issue owns
retiring it"*; **#1270 owns it** and has since #1243's post-land pass, which
wrote the deletion into #1270's Acceptance. Re-verified standing at `402ef82`
by this pass's own `gapaudit` run — the marker appears in group 1 (#1270 the
33rd of 33 annotations) and #1270 appears in group 2 as an open `kind/gap` no
marker cites, unreconciled on both sides. **#1270 may not land leaving it.**

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 13 pages (12 full at 100, page 13 at 93), PRs filtered locally:
**1293 rows, 565 PRs excluded, 728 issues — 272 open, 456 closed.**

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **52** | **114** | active |
| **M5 — Instance validation (XML)** | **14** | **23** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **256 `ready`, 14 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 272 with no gap, and **every open issue now carries
exactly one, verified mechanically over all 272**. By kind: `kind/refactor` 77,
`kind/gap` 53, `kind/process` 49, `kind/tooling` 38, `kind/bug` 30,
`kind/story` 25, `kind/docs` 15, `kind/feature` 4. By area: `parser` 77, `meta`
68, `xsd` 59, `conformance` 33, `docs` 32, `validate` 24, `cmd` 20, `value` 14,
`builtin` 10, `xpath` 6, `xsderr` 5, `model` 4, `loader` 2, `regex` 2, `cli` 1.

**That "exactly one" is a repair this pass made, not a property it inherited —
and the last stamp asserted it while it was false.** #79 and #250, the
repository's only two `epic`s, each wore `blocked` **and** `epic`, which
`docs/WORKFLOW.md` forbids in as many words (#968: *"an issue wearing two is
pickable and retired at once, and nothing this repo runs can see the
contradiction"*). The 2026-09-05 stamp read *"248 `ready`, 21 `blocked`, 0
`needs-replan` — every open issue carries exactly one, verified mechanically …
with no gap and no double-labelled row"*, and 248 + 21 = 269 balanced against
the open total **because both epics were counted inside the 21**. A check that
sums two of the four labels cannot see a third overlapping either, and it
published the exact property it could not check. `blocked` was removed from
both and `epic` kept — `epic` already carries the whole of what `blocked` was
being used to say — and both `## Depends on` sections were rewritten so the
body no longer argues for a label the issue does not wear. **#779** owns the
queue-hygiene tool; this is a **fifth class** for it (*more than one queue
label*), now written into its Acceptance, and it retires class 4, whose entire
population was these two mis-labelled epics.

**M4 moved 111 → 114 closed and held at 52 open**: #1243, #1242 and #1205
closed, and three were filed onto it — #1270 by #1243's post-land, #1275 and
#1276 by #1050's. **M5 is unchanged at 14 open and 23 closed**: nothing in this
window touched it, which is correct — every landing was `parser`, `tools/` or
`docs`. Neither milestone's count is its lane's remaining work, which the two
milestone sections below say in their own words.

**`ready` 256 is the honest startable count, with nothing to subtract**,
because the namespace is idle for a third stamp. There is no numeric cap on
`ready` and its size is an output (#347); it grew by eight this window as this
pass's four filings, the post-land passes' filings and the audit's two
outpaced the closures.

**The unblock sweep measured a clean zero for the NINTH consecutive stamp.**
All **14** open `blocked` bodies were read and their `## Depends on` sections
checked against this window's closures — the six landings plus the `/retro`'s
seven (#1279, #1220, #925, #692, #484, #1105, #1087). **Not one of the 14 names
any of them**, in any position, and neither does any open issue at all: a
full-body scan for each closure returns only adjacency paragraphs. **Zero
relabelled by the sweep.** The set shrank 21 → 14 without a single unblock, and
the arithmetic is worth stating because none of it is a discharge: the `/retro`
closed **three** of the standing 21 (#692, #925, #1220) and relabelled **#1080**
and **#841** `ready` on rulings of its own, and this pass's epic repair removed
the last two (#79, #250). **#1279** was filed `blocked` and closed inside the
same window and nets to nothing. It partitions as before and should be re-read that way:

- **FOUR are triggers rather than issues** and say so in their own
  `## Depends on` — **#1224** (a change making a new rule chargeable at
  `schemaSetLocation`), **#555** (an oracle grounding, a spec snapshot or a
  deviation ruling), **#1002** (a ruling from the repo owner) and **#1042**
  (the XPath evaluator itself). **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own
  body why that is not a discharge** — **#16** (a cliuser reference never
  worked directly; its `gen` criteria are unfileable until M9) and **#1051**
  (three of six conservative-decline buckets still unfiled).
- **EIGHT still have at least one OPEN named dependency**, checked issue by
  issue and target by target: **#248**, **#267** and **#345** (all on #250),
  **#415** (#407), **#456** (#455), **#593** (#591), **#717** (#248) and
  **#871** (#831). All eight targets were re-read and all eight are open —
  including #250, which is no longer `blocked` but is still open and still the
  gate its three dependents name.

### Persona consultations — the SEVENTEENTH ran, and the trigger fired as named

The cartographer role-plays no persona and does not spawn one (#416): it has
read the source, so a verdict it produced would launder an insider's opinion as
an outsider's. **The orchestrating session ran both personas against the
published surface at `402ef82` and handed this pass their reports; what follows
is the folding, not the consultation.**

**The last stamp named a trigger rather than scheduling a pass** — *"whatever
next changes the observable surface"*, with band rows 6, 8, 10 and 12 named as
the ones that would. None of those landed. What fired it instead is that
**#1205 changed the library's rejection surface** and the orchestrating session
ran the consultation anyway. Naming the trigger has now worked three windows
running, and this is the first where the firing landing was not one of the rows
predicted to fire it.

**Seven findings: four filed, three dismissed, none folded.**

- **libuser, four findings.** **#1291** filed — a violation's only positional
  currency is `file:line:col`; `xsderr.Error` is `{Rule, Loc, Msg, Err}`
  (`xsderr/error.go:77-85`) and nothing module-wide names the instance ITEM a
  charge is about, so a service mapping violations onto a structured response
  has line:col as its only key. **#1292** filed — `validate.Result` is three
  accessors and no combinator, so the correct *not proven invalid* test is a
  three-way conjunction a consumer must know to write; **the consumer already
  exists and re-derives it**, in `cmd/goxsd8/validate.go`'s `assessmentLines`
  (`:330-352`), which is the copy **#1257** already owns the defect in.
  **#1293** filed, and this pass sharpened it past what the persona reported:
  `xsd/doc.go` twice sends the reader to *"the package Examples"* (`:141`,
  `:242`) and **`go doc ./xsd` and `go doc -all ./xsd` both print zero Example
  functions**, while four exist in `xsd/example_test.go` — the pointer resolves
  nowhere on the surface `doc.go` is written for. One **dismissed**; see below.
- **cliuser, three findings.** **#1290** filed, and it is the pass's own
  reproduction rather than the report's: a flag written AFTER the positional is
  silently unparsed, because both subcommands use `flag.FlagSet.Parse`
  (`parse.go:37`, `validate.go:107`), which stops at the first non-flag
  argument. Two **dismissed**; see below.

**#1290's `parse` arm is worse than the arm cliuser found, and this pass found
it by reading the same mechanism into the other subcommand and then running
it.** `goxsd8 parse order.xsd -q` prints the summary `-q` asked to suppress,
then answers `goxsd8: parse: open -q: no such file or directory` at exit 2 —
`runParse` takes `flags.Args()` as `locations` (`parse.go:43`) and opens every
one of them. cliuser's arm is `validate`: `goxsd8 validate order.xml -schema
order.xsd` answers `goxsd8: validate: no schema given`, true of the parsed flag
set, false of the command line, and byte-identical to the answer for genuinely
omitting `-schema` — which `validate_test.go:527` pins as its own row. Both
reproduced here from a binary built out of the tree at `402ef82`, not taken on
the report's word.

**THREE findings dismissed, each on the thread that owns the decision rather
than in this section, so the next consultation's report is triaged against a
comment instead of re-derived.**

- **`-version` → dismissed on #672.** The persona asked for a README and
  `-help` mention that no version entry point exists. #672 DECIDED this and
  published the decline in the copy that owns it — `cmd/goxsd8/doc.go:167-168`
  — and its landing explicitly refused a second copy in README (STYLE D3).
  README already routes the reader there at `:154-157`, naming `-version` among
  the argument vocabulary `go doc` owns. The ask is the duplication that thread
  argued against.
- **bare `help` → dismissed on #687.** The persona verified it against the
  contract and ruled it by design itself. It is #687's shipped answer,
  re-probed from outside and held.
- **`parser`'s first-error-only limitation → dismissed on #1005.** The persona
  found the disclosure three times without being told where to look and called
  it *"already tracked, not a new finding"*. It is — and the finding is
  evidence FOR #1005's Goal rather than a second defect. No premise in #1005 is
  falsified, so no correction is owed.

**#1033 remains the one row no persona has ever looked at.**

### Working band

**Re-derived from this pass's evidence.** Rows 1 through 5 of the last band all
landed; nothing is in flight, so take from the top and re-run `wipsurvey` first
anyway.

| # | issue | why here |
|---|---|---|
| 1 | #1282 | **A `kind/tooling` row above two lane slices for the second consecutive stamp, and the argument is now measured rather than predicted.** #1239 shipped the instrument and #1205's session **bypassed it** — at **n=139**, both the mason and the arbiter wrote their own namespace-aware parent-classifying walk and reconciled at 139/133/6/0, with no `suiteindex` run in either account. The cause is one field: `hit` is `{File, Line, Col, Values}` (`tools/suiteindex/main.go:217-225`) and `scanFixture` (`:370-396`) discards `EndElement`, so **local-vs-top-level is inexpressible at any number of queries**. Rows 2 through 4 are all local-attribute or local-element work and every one of them will ask exactly that question. #1205's 139/133/6 are already written into its Acceptance as the reproduction pin. Pairs cheaply with **#1267** (`suiteindex -h` runs a 3s census instead of printing usage) — same file, same session |
| 2 | #1270 | **The family's last uncharged `src-attribute` clause-4 hole, and the one issue in the queue that must delete a false sentence from the tree.** A local `<attribute use="prohibited">` writing both `type=` and an inline `<simpleType>` is **ACCEPTED**, while `optional` and `required` both reject it — reproduced by #1243's post-land before filing, with the `ref=` arm caught by clause 3.2 and carried as an acceptance row. The marker at `parser/produce_complex.go:3125` still ends *"No open issue owns retiring it"*, which this pass re-verified standing; #1270's Acceptance already binds its deletion and **#1270 may not land leaving it**. Census by CONSTRUCT for the ratchet figure, with row 1's parent field if it exists by then |
| 3 | #1275 | **The lane slice with a measured population floor already on it.** An `xs:alternative`'s own children are checked against **NO** s4s content model — `checkSrcTA` counts three forms and never rejects a stray XSD-namespace child standing beside a legal one. #1050's post-land measured **232 occurrences across 81 suite fixtures** before filing, so this row starts with a population rather than a guess, and it deliberately routed its ordering question to row 4 rather than pre-deciding it |
| 4 | #1246 | **The ordering ruling three other issues defer to, and the only row here with real ratchet exposure of its own.** `src-attribute` clause 4 is charged BEHIND the s4s walk while `src-element` clause 3 — the same both-present fault — is charged AHEAD of it, and only the element side ever had an argument. It changes which fault class a doubly-violating document reports, so it needs its own grounding and must not share a window with row 2. Its table was re-read at `24ac7d3` by #1205's post-land — both `produce_complex.go` rows moved (`produceLocalElement` `:2376`→`:2437`, `produceLocalAttribute` `:3233`→`:3309`) and Option 2's destination is a more crowded run order than the table conveys |
| 5 | #1260 | **The only open row whose failure mode is a GREEN GATE rather than a missing sentence, and it is unmoved from last stamp because nothing took it.** `goxsd8 parse inc.xsd` prints a full summary and exits **0** with **zero bytes on stderr**, off an assembly whose `<xs:include>` resolved to nothing; `validate` does the same. Two settled dispositions and the body picks neither: report it, or state the silence the way README already states the missing-hint case. **It is the ruling #1251's own Acceptance defers**, and it is the half that covers `parse` |
| 6 | #1007 with #1231, #1123, #1189, #1261 and #1290 | **Now SIX issues, one session, three contract copies, one coupling test — and #1290 makes it a bug fix rather than a doc sweep.** All six edit `cmd/goxsd8/doc.go`, `main.go`'s `usage` const and `README.md` together, and `TestUsageCoversContract` pins the first two by **29** substrings. **#1290 is the newest and the only one that is a reproduced wrong answer**: a misplaced `-q` is diagnosed as a missing file named `-q` while the summary it asked to suppress is printed, and a misplaced `-schema` produces the message for having omitted it. Taking them apart means rebasing the same three files six times |
| 7 | #1203 | **A latent ratchet ambush #404's landing created, and the queue still has two issues that could trip it.** Two of #404's eleven banked `schema` passes — `addB014` and `schZ006` — reject for a fault the suite does not intend, so when #603 or #703 repairs the over-rejection those passes flip down as a spurious `Regressed` the repairing session did not cause. A banked pass resting on an over-rejection is invisible in the lane file; this measures it into a known figure. No production change — a measurement and a ruling. Cross-references #1002, whose human ruling its measurement may feed |
| 8 | #1251 | **The hint-side twin of row 5, and it sits below because row 5 subsumes its harder half.** A hint naming a document that is not there costs an operator exit 1 with an empty stderr, and the plumbing — `AssemblyReport.Unfollowed()` — already records it. Its three carried facts are worth reading before row 5 is taken, whichever lands first: `UnfollowedDirective` carries no `schemaLocation` value, every `-schema` argument and every hint is a directive in the SAME synthesized wrapper root, and that wrapper is emitted on one line so attribution within it is column-only |
| 9 | #1167 | **Two measured sightings, #414 and #1115, and PRINCIPLES 27 says a repeated grep wants a tool.** `gapaudit` reconciles marker → issue and issue → marker; nothing audits prose that points *at* a marker or its file, and the stale-marker sweep has missed an inbound site in the deleting package twice. `gapaudit`'s own group 1 candidate-owns it against `validate/assess.go:853`. Row 2 is the near miss that is NOT a third sighting — that marker's own sentence is stale, not a sentence pointing at it — and the distinction is worth keeping so the count stays honest |
| 10 | #1227 | **The pen bound is written and nothing checks it — first sighting AFTER the rule existed, which is what makes it fileable.** `docs/WORKFLOW.md` binds who may hold the pen, but **landing precondition 3 iterates over MASON commits**, so a branch with none has nothing to iterate. #636 is the only other witness and is **pre-rule** by one day, so this is n=1 post-rule and the deliverable is a runnable check, not a fiftieth `kind/process` issue. Both landed precedents are named in its body: #1018's doc-only precondition 4 and #963's `landcheck` |
| 11 | #1292 | **The library half of the verdict #1223 already shipped in the CLI, and its consumer is in the tree.** `validate.Result` exposes `Violations`, `Unevaluated` and `Err` and no combinator, so *not proven invalid* is a three-way conjunction; `assessmentLines` computes exactly that relation in `cmd/goxsd8` and **#1257 owns the defect in that copy**. Read the two together — #1257 is that the copy is wrong, this is that there is a copy. The honest outcome may be the doc arm rather than a guard, and the body says so rather than presupposing the addition |
| 12 | #1293 | **A doc pointer that resolves nowhere, measured on both `go doc` invocations rather than argued.** `xsd/doc.go:141` and `:242` send a reader to *"the package Examples"*; `go doc ./xsd` and `go doc -all ./xsd` print zero Example functions, and four exist. **The inverse of the zero-Examples family** (#670, #1088, #1006 ship none; `xsd` ships four and cannot show them), so none of those three fixes it. Pairs with **#854** — same document, same reader, one session and one rebase |

**Below the band, and why**: **#1265** and **#1266** are #1239's other two
follow-ups and are ordinary; **#1267** pairs with band row 1 and should ride it.
**#1276** is the derived check that makes #1050's Scope section unfalsifiable by
drift — cheap, and it pairs with whoever next opens `parser/census.go`.
**#1250** is #1251's sibling on the same code path and either can land first.
**#1291** is the fourth persona filing and stays below the band on a real
sequencing question its own Surface section names: **#486** wants
`xsderr.Error`'s four fields unexported behind `New`/`Wrap`, which decides how a
fifth would be reached. **#1285** and **#1287** are the 2026-09-06 steward
audit's two findings; **neither carries a `## Cost of delay` section**, so the
`kind/refactor` steward-ranking clause of the banding rule selects neither, and
both are ordinary `xsd` hygiene. **#1286** is five `doc.go` contracts stating as
shipped what does not ship. **#1283** and **#1284** are `xsd`/`parser` surface
questions; #1283 now blocks #1051's unexport and should be read with it.
**#1156** is comment text only. **#1201** is correct, needs a warden pre-flight
for its 1a fix, and **zero suite cases exercise it**; take it opportunistically
with row 7, and it retires the two `GAP(conformance)` markers when it lands.
**#1183** is cheap, correct, moves nothing and makes #1051's stage-2 deletion
three predicates smaller. **#1196** measures the live residual for #1051 and
**its own body forbids banding on the numbers it produces**. **#1135** pairs
with row 2 and row 4's neighbourhood; its absorption invitation to #1205/#1206
is now recorded as **spent**. **#1258** is the MCP `list_issues` divergence and
**is still n=1** after a fourth non-reproduction. **#1257** is opportunistic and
should be taken with band row 11. **#768** was RE-SCOPED by #1205's post-land,
not re-stamped — its replacement plan does not exist, because `dcl.elt.common`
gives `{abstract}` exactly one source and `xs:schema` declares no
`abstractDefault`, so the FAIL cell of its own mutation matrix is unreachable;
it now carries two honest options and is a comment-sized change. **#845** was
RE-SCOPED by this pass — see below. The remaining doc rows —
**#1232**, **#1233**, **#1234**, **#1235**, **#1236**, **#513** — are
one-sentence fixes with settled directions; #1236 is the README twin of #513 and
the two are one session. **#1122** and **#1003** stay below the band as one-line
README fixes. **#1033** is still the only row no persona has ever looked at.

**There is no `Increasing` steward ranking anywhere in the band, and the audit
that owed one ran without producing it.** A grep of all 272 open bodies for
`Cost of delay` returns exactly **three** — **#849**, **#848** and **#845** —
and none reads `Increasing`: #848 and #845 are both *"stable debt"*, and #849's
reading is falsified and unreplaced by its own body (*"a steward re-ranking is
warranted"*). The `kind/refactor` cost-of-delay clause of the banding rule
therefore selects nothing this stamp, for the second stamp running. **#845 was
re-scoped by this pass on a measurement the tree forced**: its Goal —
*"`internal/schemaloc` is folded into `parser` (its sole remaining consumer)"* —
is false, because `cmd/goxsd8/validate.go:597`/`:605` is a second production
consumer beside `parser/parse.go:485`/`:585`. Its body now carries two options:
(a) rewrite `internal/schemaloc/doc.go`'s justification to name its two real
consumers, startable today, and (b) the fold, which needs **#755** first. It
stays `ready` because (a) is startable.

### Next planning action

1. **Take #1282 before the next lane slice, and hold it to the bypass it was
   filed against.** The instrument exists; the question rows 2 through 4 will
   each ask — local versus top-level — is the one query it cannot express, and
   a session that takes a lane slice first must census by CONSTRUCT with a
   hand-written parent walk and say in the commit body that it did. Band row 1
   is a `kind/tooling` issue above two lane slices for the second consecutive
   stamp, which is the banding rule applied as written: a tool that moves no
   lane will never be lifted on lane grounds.
2. **`parser/produce_complex.go:3125` asserts a falsehood in the tree and its
   deletion rides on #1270.** This is the concrete form of the general question
   **#1167** owns. **The near miss is worth naming so the count stays honest**:
   this is a marker whose OWN sentence is stale, not prose pointing at a marker,
   so #1167 stays at two sightings and is not banded up on it.
3. **The queue-label defect is repaired and the check is specified — do not
   re-derive either.** #79 and #250 wear `epic` alone, both bodies are
   rewritten, and all 272 open issues carry exactly one queue label, verified
   mechanically after this pass's writes. **#779** now carries the fifth class
   (*more than one queue label*) in its Acceptance and the note that class 4's
   entire population was these two epics. If a `/backlog` ever again asserts
   "exactly one, verified mechanically" from `ready + blocked` summing to the
   open total, that is the same false check: the sum balances while an `epic`
   hides inside `blocked`.
4. **The seventeenth persona consultation is folded and nothing from it is
   handed off.** Seven findings: four filed (#1290, #1291, #1292, #1293), three
   dismissed on the threads that own the decisions (#672, #687, #1005). **The
   next consultation is owed against whatever next changes the observable
   surface** — name the trigger rather than scheduling a pass, which has now
   worked three windows running. Band rows 5, 6, 8, 11 and 12 all change it;
   rows 1 through 4, 7, 9 and 10 do not.
5. **`meta/audit-2026-09-06` stands unmerged with no PR and two open issues cite
   it as landed.** `docs/WORKFLOW.md` gives a maintenance command one session to
   open and squash-merge its own PR; this one did not, and #1285's Notes say
   *"the 2026-09-06 audit corrected the ARCHITECTURE.md attribution"* while
   `main` at `f97c317` still carries the wrong attribution. **A session never
   deletes or force-pushes a ref and this pass acted on nothing** — it is for
   the orchestrating session or a human to land or retire. If it still stands at
   the next stamp, that is a second sighting of a maintenance branch outliving
   its session and worth filing as one.
6. **#849's steward re-ranking is owed for a SECOND consecutive stamp, and the
   `/retro` that owed it has now run.** The 2026-09-06 architecture audit filed
   #1285 and #1287 and ranked neither, so the ranked backlog did not grow. One
   `/retro` passing over a routed re-ranking is an anecdote; a third leaves it a
   pattern, and the issue to file then is about the routing, not about NCName.
7. **The human decision blocking #1002 is unchanged and is now carried for a
   THIRTEENTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`,
   enumerated by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until
   real assertion evaluation lands. CLAUDE.md puts (a) beyond any agent —
   *"changes only via a human-filed issue"* — and (b) depends on **#1042**,
   filed and `blocked`. **No agent should attempt either.**
8. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3). **M6 tier 2 itself is uncarved** —
   `$value` binding, an F&O function library, typed comparison — and is too big
   for one issue. That carve is a `/backlog` act at the M6 opening, and #1042 is
   the thing it must slice around rather than a blank page. #1042's
   `## Depends on` names #719, which is **closed**: the live dependency is the
   XPath evaluator itself, a trigger, and the body says so.

**Standing, and re-checked rather than restated.** Four unlanded corrections
still target one paragraph of `docs/WORKFLOW.md`'s filing discipline —
**#510**, **#646**, **#679**, **#912** — and whichever lands last rebases three
times; note the 2026-09-06 `/retro` rewrote a neighbouring bullet of that same
paragraph, so re-read it before grounding any of the four. **The next `/retro`
inherits three rather than seven**: #841 and #1080 were relabelled `ready` by
this one, and #692, #925, #1220 and #1279 were closed by it — what remains is
the fold-the-five-species question (#635, #912, #609, #510, #646), the
`[tests that cannot fail]` **pattern** (routed by #472's post-land, with no
filed carrier), and **#849's owed steward re-ranking**. The `gh --paginate`
page-2 403 trap stays a `/retro` datum and not a filing. The CTA cohort's 45
banked `instance` failures remain unattributed. `gate.yml` runs and is still not
a required status check, which only the repository owner can change.

**Environment, one witness each.** Repository-scoped REST served every read and
every write here: **22 writes across 12 distinct issues** — **4 filed** (#1290,
#1291, #1292, #1293), **6 body PATCHes** (#79, #250, #779, #845, plus #1291 and
#1292 conformed to the filing-discipline bullet the `/retro` landed mid-pass),
**4 label operations** (`blocked` removed from #79 and #250, `area/cmd` added to
#845, and #845 retitled off the premise its body no longer holds) and **8 thread
comments** (#79, #250, #779, #845, #849, #672, #687, #1005) — plus this
section's own replacement.
**The paginate recipe ran to 13 pages three times**, 12 full at 100 each time
and page 13 short at 93 on the post-write run, with the post-loop fullness check
passing on all three. `gh auth status` reports the token invalid in this
container and `gh api repos/kud360/goxsd8/...` served everything anyway, which is
**#1140**'s landed caveat behaving as documented. **`main` moved under this pass
at `402ef82` → `f97c317`** when the 2026-09-06 `/retro` landed mid-run; it
touches no `conformance/testdata/expectations/` file and no `docs/PLAN.md` line,
so neither the lane table nor this replacement is affected, but two bodies filed
before it were conformed to its new filing-discipline bullet afterwards. **No
conformance measurement was taken by this pass**: the lane table above is the
committed expectations, which `docs/WORKFLOW.md` names as the lane score
(#1120).

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
`use="prohibited"` branch) is #1243's, and #1242's landing closed its arm with no
residue. Live producer-decides-and-accepts members are **#1270**, **#931**
(occurrence attributes on a named `<group>`'s child compositor), **#929** and
**#455**. **The family carved a successor at every landing from #471 through
#1243, and #1242 is the first that did not** — one reading is the tail reaching
zero, the other is that this pair was narrower than its predecessors; two more
landings decide it, and until they do neither reading is the one to plan on. A
second, narrower family opened beside it — the rejections the
producer already makes **correctly but describes badly**, whose bar
`xsderr/doc.go` set with #966 — and it is **discharged**: #975 landed at
`1dcffbf`, so every s4s-grammar rejection now names its Appendix A production.

**A THIRD family is the one that has been paying, and it has FIVE landed
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

**The family replenishes itself as it is worked, which is what to expect and
not tail growth** — every issue it has added was filed by the post-land pass of
a landing that moved the lane. **Its shape is also changing**: #1047, #1076 and
#1099 all widened `checkS4SChildOrder` or its callers, and the issues arriving
now — #1097 (`32070b8`) and **#1136** (`d854e1d`, 2026-09-05), both landed, and
open #1098, #1133 and #1135 — are defects *in* that widening rather than further
sites for it. **A producer that decides more is a producer with more to get
wrong, and the census does not name that class.** #1136 is the one that stopped
the accumulation: it replaced four independent local run-order guesses with ONE
recorded decision, and its own follow-ups (#1246, #1247) are about that record
rather than about a fifth guess.

**Read the milestone count as a floor.** The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it — #1135 is M4 work carrying no
milestone today, and #1136 was too until it landed. **The sharpest instance is
#1126**, which moved `schema` +475 and `instance` +113 — the largest lane
movement this project has recorded — and carried no milestone either, so the
count did not move at all.

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
  (`<simpleContent>` `<restriction>`), the CTA pair #842/#851.
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
window was `cmd/goxsd8`.** The subcommand issues open here today are #1224
(`blocked` on a trigger), #1250, #1251 and #1257; **#1260 and #1261 are the same
shape and carry NO milestone deliberately, because they span `parse` (M4) as
well.** Read the count as a floor and never as the lane's remaining work.

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

**All TWELVE decided-and-wrong `instance` cases carry an owner, and the two
classes below account for nine of them.** Measured at `c720206`: 15332 fail =
15315 decline candidates + 5 declined indeterminate + **12** decided-and-wrong.
The other three are owned outside these paragraphs (#1160) —
`MS-DataTypes2006-07-15/gMonth002_2061/instance/gMonth002_2061.v` and
`gMonth004_2063.v` by **#921**, and `Open/open013/instance/open013.v1.xml` by
**#456**.

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
