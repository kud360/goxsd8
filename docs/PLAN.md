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

## Status — 2026-09-21 (`/backlog`, the twenty-ninth)

**The lane moved, the prediction that called it was EXACT, and the band had marked
that row DEAD. That is this stamp's finding.** `instance` banked **11205 → 11209
(+4)** and all four flips came from one landing — #1601's ceiling raise at
`46dc042`, whose grounding named `particlesZ007.i`, `particlesZ034_a2.i`,
`particlesZ034_a3.i` and `particlesZ034_b.i` in advance and banked those four and
no others. **Verified here, not inherited**: `diff` of
`expectations/instance.txt` between `a2a72bf` and `40eaeac` is exactly those four
lines, `fail` → `pass`; `schema.txt` and `datatypes.txt` are byte-identical.
**#1601 was row 2's declared successor and row 2 read "#1588 is DEAD — start
elsewhere."**

**So the twenty-eighth stamp's falsifiable test was never RUN, and something
better happened instead.** The test was *"if #1585 lands and #1588's `PREDICTION:`
block still answers `not derived`, the defect is not the query language."* #1585
landed (`81671d7`). **#1588 was then closed `not_planned` rather than taken**, so
no `PREDICTION:` block was ever fired at it. #1601 took the same population by a
third route and **derived a case-exact prediction there** — which answers the
question the test was asking, in the affirmative, one row over.

**Read the mechanism before ranking anything, because it is not the instrument.**
#1601's prediction was derived by **MEASUREMENT** — a `DIAG=1` corpus probe
committed and deleted again (`3f0998d`, `fdfae51`), then a big-int
`partitionsBounded` recomputation by the arbiter — and by **#1561's subtraction
rule**, which correctly withheld three cases the suite declares valid. **It was
not derived from `suiteindex`.** #1585's `//` ancestor census landed one commit
earlier and **has still been exercised on zero predictions.** The chain works; the
census instrument inside it is untested in the field.

### Conformance lanes

**Paste `go tool lanestatus` verbatim, never a hand-count.** This table is `main`
at `40eaeac`, and it is the committed expectations census —
`conformance/testdata/expectations/<lane>.txt`'s `pass`/`fail` line counts — which
`docs/WORKFLOW.md` names as the lane score (#1120) and which is the only lane
figure a later session can re-derive from the committed tree alone.

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11209 | 15152 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14697 | 701 | 15398 |
| `xpath` | — | — | 0 |

**SEVEN issue landings this window and ONE lane moved, by FOUR, all from ONE of
them.** The other six moved nothing and **five of the six correctly so**: #1585 is
tooling; #1565 + #1564 (`314f795`), #499 (`84dda95`), #1601's second landing
(`a8b3b71`) and #267 (`89167f1`) are comment-only permanent-gap rulings whose
diffs contain no executable line, each verified as such at both review rounds.
#1102 landed twice — `bdf49f8` ruled its over-report permanent and `e0e3c46`
retired the gap outright — and banked nothing either time, because the valid
schema it unblocked has no case in the suite.

**The window's genre split is the ranking problem, stated plainly.** Six of seven
landings were **rulings, tooling or process**; the one that moved a lane was a
**ceiling RAISE**. #1601 shipped BOTH genres — `46dc042` raised the ceiling for
`instance` **+4**, `a8b3b71` ruled the residual permanent for **0** — from one
issue, under one band row, and the route was chosen at grounding, which is
**after** banding. **An issue offering a retire-it route and a rule-it-permanent
route cannot be banded on lane movement, because its movement is decided after the
band is published.** That is not a defect in the ordering; it is a property of the
issue shape, and it is the first thing this record has found that the ordering
cannot fix.

**`datatypes ⊆ instance` is INTENDED** (#1507 ruled arm A against outright), so
`instance` 11209/26361 is a figure to quote and not to hedge. **The column does
not sum to the corpus and that is SETTLED**: the three lanes hold 41759 distinct
case IDs against a column summing to 42932, because every one of the 1173
`datatypes` cases is also committed in `instance.txt`, 701 with a contradictory
verdict — two differently mature executors over one fixture.

**An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero.** `datatypes` is M3 and complete; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**No conformance measurement was taken this pass**: `git submodule status` reads
`-7bc3365…`, uninitialized. Every suite figure here is cited from the landing that
measured it, and the `7bc3365` pin is read off the gitlink.

### Branch namespace, `origin` — report-only; a session never deletes a ref

`go tool wipsurvey` after `git fetch --unshallow origin`. **FOURTEEN `wip/*` refs,
and for the first time in five stamps there is a LIVE CLAIM.**

**`wip/issue-1414` is LIVE and a `/develop` session is on it RIGHT NOW.** Tip
`1dde8f2`, pushed 2026-09-21 14:23Z, `ahead=2 behind=0`, inside the 2h TTL —
*"parser: fix produce_test.go comment on #1414's wrong-kind pin (#1414)"*. **#1414
is therefore NOT in the band below and must not be claimed**, and it is the one
open dependency of `blocked` **#731**, whose ledger closes when it does.

**The other THIRTEEN are all RETIRED** — #434, #732, #822, #846, #872, #933, #968,
#993, #1332, #1356, #1426, #1451 as in the last four stamps, plus **#1588**, whose
branch survived its issue's `not_planned` closure. Nothing is owed by any of them:
twelve closed `not_planned`, so the branch is the re-planning evidence the label
retires in place, and #1332's content is in `main` in a superseding form. **Two
RETIRED rows still print a lease age borrowed from `main`'s tip** (#933, #968) —
that is **#1548**, still open and still cosmetic here.

**SEVEN `claude/*` refs stand on the remote, one more than the last stamp found,
and the filing condition it set has FIRED.** `eloquent-cerf-39rk64`, `-3xu0ki`,
`-8jq9o6`, `-adewly`, **`-g3rfvu` (new)**, `-kk1f7v`, `-ldi2dl`. **Every one is
`ahead=0` of `main`** — 256/214/261/202/4/130/68 behind, measured after
unshallowing — so each is a merged session branch whose auto-delete did not fire
and **nothing is owed and nothing is at risk.** The last stamp said *"if the next
stamp finds the same six plus new ones, file it"*; the same six stand plus one, so
**#1627 is filed** and carries a second and sharper arm: **`wipsurvey`'s refspec
is `refs/heads/wip/*` alone** (`tools/wipsurvey/main.go:206`) while
`docs/WORKFLOW.md:79` names the discovery index as `wip/*` **and** `parked/*`. So
**four consecutive stamps have quoted a `parked/*` count this tool cannot
compute** — including this one, which reports **zero `parked/*` from
`git ls-remote --heads origin` directly** and says so rather than crediting the
survey.

### Marker census

`go tool gapaudit` over the whole-repository feed, on `main` at `40eaeac`: **68
markers, 9 areas, group 1 at 17, group 2 at 30.** Against the last stamp's
70/21/33 that is the healthiest reading this record holds, and the cause is one
landing.

- **ZERO dead-end rows, tree-wide, for the FIRST time.** The row that had survived
  four stamps — `xsd/contentrestricts.go`'s ceiling markers citing CLOSED #501 and
  CLOSED #1378 — is gone, retired by **#1565**'s **STYLE P3b** (`RULED permanent
  by #N`, which `gapaudit` reads as a `matchKind` that retires a marker whatever
  the cited issue's state). **Four markers in the tree now cite a CLOSED issue
  legally**: `xsd/contentrestricts.go:699` (#1378), `:856` (#499),
  `xsd/contentmatcher.go:312` (#1601) and `xsd/defaultbinding.go:100` (#267).
- **That is also the whole of the group-1 fall, 21 → 17**, and it is the strongest
  argument this stamp has for the ruling genre: four comment-only landings that
  moved no lane cleared four rows and a five-stamp-old dead end. **Do not read the
  lane column as the only ledger.**
- **P3b is one week old and STYLE P3 still reads as though it were not.** P3's
  *"a marker pointing at a closed issue is a dead end"* carries no pointer to the
  exception, so a session that greps `dead end` lands on an unqualified rule and
  would repair one of those four markers as a defect. That is **#1616**, and
  `gapaudit` itself names it as the file-match candidate owner of three group-1
  rows in `contentrestricts.go`. Row 4 below.
- **Every group-1 row names at least one live candidate owner, so no `kind/gap`
  issue is owed by this pass.** The two `GAP(xpath)` markers #812 left in `icpath`
  are still #1572's; the two deliberately-unowned `xsd/defaultbinding.go` sites are
  now at `:392` and `:588`.
- Group 2 fell 33 → 30. Four `kind/gap` trackers closed this window (#1601, #499,
  #1102, #267) and #1588 was declined, which is the right order of magnitude;
  neither of this pass's filings is `kind/gap`, so neither enters it. **The exact
  attribution is not re-derived here** — group 2 is *"OPEN `kind/gap` issues no
  marker cites"*, so a landing that writes a number into a marker moves it too.

### Milestones and queue

**One full page-numbered `state=all` REST walk, clean on the first request,
stamped `rows 1625 distinct 1625 min 1 max 1625 no gaps`**: 1625 rows, 736 PRs
excluded, **889 issues — 347 open, 542 closed.** Those are the walk's figures.
**This pass then filed two (#1626, #1627), so the queue stands at 891 issues, 349
open, 334 `ready`** — stated as arithmetic on a measurement rather than as a
second measurement, because the walk ran before the filings and `search/issues` is
a 403 here.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **136** | active |
| **M5 — Instance validation (XML)** | **17** | **32** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels at the walk, open only: **332 `ready`, 13 `blocked`, 0
`needs-replan`, 2 `epic`** — summing to 347, and **every open issue carries
exactly one**, verified over all 347 by grouping each issue's own label set rather
than by summing counts. **Every open issue also carries at least one `kind/` and
at least one `area/` label; there are zero exceptions in either direction.** #779
owns the mechanical check.

By kind: `kind/refactor` 83, `kind/process` 66, `kind/gap` 55, `kind/tooling` 48,
`kind/story` 41, `kind/bug` 34, `kind/docs` 34, `kind/feature` 3. By area: `meta`
99, `parser` 84, `xsd` 62, `docs` 50, `conformance` 36, `cmd` 34, `validate` 29,
`value` 18, `builtin` 12, `xsderr` 9, `regex` 6, `xpath` 6, `model` 4, `loader` 2,
`cli` 1, `icpath` 1 — sixteen area labels, of which `icpath` is still named in no
document, which is #1019's subject. **275 of 347 open issues carry no milestone at
all**, so read the `area/` census and never the milestone counts.

**M4 has not moved for a FOURTH window** — 54 open, byte-identical across FIVE
stamps; closed 135 → 136 with #1102. Of M4's 54 open, **22 are lane-visible**
(`kind/gap`, `kind/bug` or `kind/feature`) against 23 `kind/refactor` that move no
lane by construction. **M5 absorbed this window's closures**: 30 → 32 closed
(#1102, #267), open 18 → 17 (#1588 declined out of it). Of M5's 17 open, **9** are
lane-visible.

**`ready` 326 → 332 at the walk, 334 after this pass, and the turnover reconciles
exactly.** **SEVEN left** with the closures (#1585, #1102, #499, #1565, #1564,
#267, #1588), leaving **319** of the old set — measured directly, by counting open
`ready` issues numbered ≤ 1591. **THIRTEEN were filed into it** (#1593, #1594,
#1595, #1596, #1599, #1604, #1610, #1613, #1616, #1617, #1618, #1619, #1624), and
**#1601 was filed and closed inside the same window**, netting zero. 319 + 13 =
332, plus this pass's two. **A net of +6 on a turnover of twenty-one is #347's
shape**: `ready` is an output, not a target.

**The unblock sweep measured ZERO**, over all thirteen `blocked` bodies'
`## Depends on` sections read byte-faithfully from repository-scoped REST. **No
entry names #1585, #1601, #1102, #499, #1565, #1564, #267 or #1588.** The set grew
by one with **#1609** and its shape is otherwise the one the last five stamps
recorded: **five are triggers rather than issues** and say so (#1374, #1224, #555,
#1002, #1042 — **do not re-scan these**); **two state in their own bodies why
every closed dependency is not a discharge** (#16, #1051); **five have exactly one
open named dependency each** — #593←#591, #731←**#1414**, #871←#831, #1344←#1324,
#1458←#479, and four of those five targets are `ready` today while #1414 is the
LIVE claim above. **The graph is depth-1, no cycles, no `blocked` body names an
epic.**

**#1609 is the one `blocked` trigger that is mechanically checkable, and it did
NOT fire.** It waits on the `schema` lane widening past **14697**, the score #499
was ruled at; `schema` reads 14697 today, unmoved. **Re-scan it every sweep** — its
own body says so, and it is the only trigger in the set that does.

**The `ready` audit found ZERO queue mislabels, and both hygiene classes fell
again.** Class 6 (a required body section absent) is **4 of 332**, down from six:
**#743 and #744 left outright**, the last pass's repairs held, and **#845 gained
the `## Notes` it was missing** by some landing's edit rather than by this pass.
What is left — **#586, #845, #848, #1419** — is the same four the last stamp named,
and the reading it recorded stands: all four are deliberately-shaped bodies
(steward `## Current state`/`## Direction`/`## Cost of delay`, or evidence-first
numbered findings), so the prescribed repair preserves the filing's structure
rather than stamping the template over it. **For #845/#848 that is a steward
question, and TWO consecutive passes have now declined to rule it** — if a third
declines, the ruling is what is missing, not the repair. Class 7 (a lane-visible
body with no ratchet bar) is **5 of 82** — #584, #753, #785, #1049, #1419 — and
**#743 left it exactly as the last measurement predicted**, the first prediction
this thread has made and had confirmed. The measured tables, both denominators and
the case-insensitivity trap (a naive `Ratchet` grep reports **14**, not 5) are on
#779.

### Persona consultations — TWO reports folded this pass

**The cartographer role-plays no persona and does not spawn one** (#416). Both
reports below were gathered by the orchestrating session and handed to this pass.
**This is the first stamp in three with anything to fold.**

**The headline is that neither persona found a wall.** The `libuser` — README and
`go doc` only — called M4 and M5 *"usable as documented — unusually so"*, found the
README's Library snippet copy-pasteable and compiling with every option's default
stated inline, and **found no unjustified exports**: `xsd`'s producer surface,
`value`'s capability interfaces and `parser.Document`/`Node`/`Element`/`Text` each
carry a named consumer. The `cliuser` found the CLI contract matched observed
behavior **exactly** on every one of 0/1/2/3/4 and every stderr form it exercised.
**Record that as a finding, not as an absence of one**: the queue's `area/cmd` and
`area/docs` families are 34 and 50 issues deep precisely because that contract is
written down, and two outside readers just confirmed it is written down
accurately.

**FILED — two issues, both with the mechanism re-run against the tree before
filing.**

- **#1626** (`area/cmd`, `area/docs`, `kind/story`) — `goxsd8 -version` answers
  `goxsd8: no subcommand given` and names **neither** the token nor
  `doc.go:232-233`'s actual answer, while `goxsd8 -xyz parse order.xsd` names
  both. One branch, `diagnose` (`main.go:278-287`): `leadingFlagFmt` fills three
  `%s` from the argument list and `noSubcommand` is a bare constant. **The current
  answer is DELIBERATE and pinned** at `main_test.go:65`, so the issue overturns a
  tested decision or declines to. Three routes offered; **(b) — give
  `noSubcommand` the token, covering `-q`/`-xyz`/`-version` alike — is named
  smaller than the persona's own (a)**.
- **#1627** (`area/meta`, `kind/tooling`) — the `wipsurvey` namespace finding
  above. Not a persona finding; filed by this pass's own branch sweep, on the
  condition the last stamp set.

**ALREADY COVERED — three findings, each recorded on its existing thread rather
than re-filed.**

- **The accept-path verdict** (*"no `IsValid()`/`IsComplete()`… the 200-OK path has
  no sanctioned check"*) is **#1292**, and this is its **THIRD** independent
  libuser sighting. Commented with the one datum the body lacked: the persona
  called the M5-in-progress disclosure honest and then asked *"what should my
  service actually do today"* — so the doc arm's bar is now that a consumer can
  write their own 200-OK branch from the published surface, not merely learn what
  they may not conclude.
- **`-help` naming no route to the fuller contract** is **#1089**, second
  independent cliuser sighting, same recommended route (a). Commented with
  **counter-evidence three lines from the constant (a) would edit**:
  `helpPointer`'s own doc comment (`main.go:203-207`) records that *"a `go doc
  <import path>` invocation needs the module tree and fails for an installed
  binary (#870)"* — the tree already refused that pointer on the error path, for
  the same reader. Route (a) is not thereby refused, but a landing that adds the
  line without engaging #870 has overturned a committed decision silently.
- **`parser.Parse`'s first-error-only stop** is **#1005**, which owns the
  `doc.go:255-258` PLANNED promise. The persona called it *"clearly stated… a
  confirmed dead end, not a gap I could work around"* — an outside confirmation
  that the disclosure works, and no new arm.

**NOT FILED — one finding, falsified against the tree, and the falsification is
recorded on #1382 so no later pass re-derives it.** The report asked for
`validate.New`'s backend-must-match invariant to be *"promoted onto `New`'s own doc
comment"* on the claim that *"`go doc ./validate.New` itself carries no such
warning."* **It does**, as its second paragraph, byte-identical to the block #1382
has quoted since it was filed. The substance — a mismatched pair fails silently —
is #1382, already `ready` with three ruled routes.

**How that false premise arose is itself already filed, and naming the issue is
worth more than blaming the reader.** The most likely reading is that the persona
read `go doc ./validate` — the package doc, where the sentence is not — and never
reached `go doc ./validate New`. That is exactly **#1593**'s subject on the
neighbouring package: *"`go doc` opens on 96 lines of phase exegesis before the
`Parse`/`ParseReport` signatures, and the declaration index is at line 301 of
320."* **A consultation's false premise is evidence for the doc-structure issue,
not against the consultation.** #1594 and #1595 are the same batch and are also
still open and `ready`.

**A libuser batch landed in the queue between the last stamp and this one and is
not this pass's to re-fold**: **#1593**, **#1594** (`value.Backend` publishes no
goroutine-safety contract while `validate` delegates one to it) and **#1595** (six
exported doc comments defer their rationale to a sibling with *"on the same grounds
as [X]"*). All three are `kind/story`, all three `ready`. **#1595 is the nearest
thing the queue holds to this report's `Result.Err` finding**, which was checked
and did not survive: `Result.Err`'s own godoc DOES carry the incomplete-vs-invalid
nuance and `xmlsrc.Validate`'s DOES state the two-channel split in full, so the
persona's copy-paste hazard is served at both sites a reader reaches. Not filed,
and #1088 (the package ships zero runnable Examples) is where a worked guard
against that hazard belongs.

**The REPAIR rate the twenty-seventh stamp said to decide on now has a second
window.** Of this consultation's six distinct findings, **three re-found an
existing issue, one was new, and TWO were FALSE against the tree** — the `New`
backend-invariant claim above and this section's own `Result.Err` finding two
paragraphs up. That is the first time a persona report has produced a premise
the queue had to reject, and it is the datum `/retro` needs: a consultation's
value is not only what it files, it is also what it costs to check.

### API- and CLI-facing routing, for the next consultation

- **M5 — Instance validation (XML)** is the one milestone that is BOTH. Its own
  `## Surface` names `validate.Validator`, `Result`, `Option`, the abstract
  infoset interfaces and `xmlsrc.Validate`, and **6 of its 17 open issues also
  carry `area/cmd`** (#16, #1224, #1250, #1257, #1346, #1494).
- **M4 — Schema parsing** is API-facing only: the exported `parser`/`xsd`
  surface, 54 open.
- **M6 — XPath required subset** is neither yet. Its one open issue is #1042 and
  no surface exists to consult about.
- **The CLI-facing work is NOT a milestone.** 35 open issues carry `area/cmd` or
  `area/cli` and most carry no milestone, so milestone counts cannot route a
  `cliuser` run. **The `cmd` contract family now stands at THIRTEEN issues over two
  files**: the last stamp's eleven (#1407, #1549, #1521, #1381, #1368, #1419,
  #1421, #1318, #1569, #1570, #1571), plus **#1433**, which that stamp omitted (the
  121 lines of `-help` contract prose with no worked invocation), plus **#1626**.
  #1407's five-row exit-code table is the structure four of the others want to hang
  a clause on.
- **Both personas have now been consulted against the SAME surface twice and the
  second run found less.** The next `cliuser` run should be pointed at a surface
  that has CHANGED — `gen` (#1596) or the multi-`-schema` set (#1432) — rather
  than re-walking `parse`/`validate`, or it will re-find #1089 a third time.

### Working band

Ordered for a `/develop` session: take the highest row you can start. **ONE claim
stands: `wip/issue-1414`, LIVE, pushed 14:23Z today.** #1414 is therefore absent
from this band and from the list below it. **Re-run `wipsurvey` before starting
anything anyway** — this is a snapshot, and a landing arrived mid-pass in three of
the last five stamps.

**The ordering principle this stamp, and it is a correction rather than a
re-argument.** The lane-first rebuild is KEPT. But this window proved that the one
genre which moved a lane (**measure a ceiling, then raise it**) and the genre which
moved none (**rule a ceiling permanent**) are *the same issue shape*, routed at
grounding. **So among ceiling and approximation issues, prefer the ones whose
MEASUREMENT has not been taken yet** — that is where the +4 came from, twice now
(#1378, #1601) — and do not re-band one already carrying a ruling. Everything else
ranks on the two ledgers that actually improved: the marker census and the
sessions an issue costs.

| # | issue | why here |
|---|---|---|
| 1 | #1604 + #1617 + #1618 + #1619 | **The instrument this stamp's own figures came out of, and four issues over one file.** #1604 is the one that matters: an issue feed missing `state` decodes as ALL CLOSED in `gapaudit` and ALL OPEN in `wipsurvey`, so the survey prints **confident false dead ends** — in the silencing direction, against the exact census this section quotes. Three consecutive post-land passes filed into this file; `kind/tooling` banded on the sessions it costs, and the cost is measured rather than asserted. Take all four or the file rebases four times |
| 2 | #1494 | **M5, and THE falsifiable test of this band.** An unresolvable `xsi:type` exits 0, silent. **Oracle first** — do not implement a charge before the rule ID exists, and *"the spec charges nothing"* is a legitimate outcome. **Banded FIVE times without being taken**, and its own body says its prediction is not derivable with the tools in the tree. #1585 landed those tools and no prediction has used them yet. **If its `PREDICTION:` block still answers `not derived`, the instrument is the wrong SHAPE and not the missing one** — and the next stamp says so instead of banding more census tooling |
| 3 | #773 + #774 | **The M5 eighteen-slice carve's entire remainder, `instance`-lane, and named below the band TWICE.** Both bodies were repaired two passes ago and both have held. #744 needs the warden pre-flight its own `## Surface` demands. Being named-below-the-band twice without being taken is the starvation #1499 documents; this is the row that tests whether naming works |
| 4 | #1616 | **The cheapest row here and the only one whose absence actively MISLEADS.** STYLE P3's *"a marker pointing at a closed issue is a dead end"* has no pointer to P3b's exception, one week after P3b landed and took the tree's dead-end count to zero. Four markers now cite a CLOSED issue legally and a session grepping P3 would repair one as a defect. `gapaudit` names it the candidate owner of three group-1 rows |
| 5 | #1599 | **Row 2's instructions.** `CLAUDE.md`'s `suiteindex` paragraph is 31 unbroken lines teaching six query shapes, each appended after the last, and *"Length is a signal"* reads on it. The instrument it documents has been exercised on **zero** predictions; whether that is the tool or the paragraph is undecided, and this is the cheaper half to find out |
| 6 | #1589 | **A `kind/refactor` carrying a MEASURED cost of delay, which is the one thing that lifts a refactor** (#843). `region.path` is one fact stored N times, and #1557's landing measured the clone at ~700 µs per item over a wide model. Banded tenth last stamp and unconsumed. Its honest outcome may be *"the measurement says it does not pay and this issue closes on that figure"* — a legitimate landing, and it must be stated as such before mason is delegated |
| 7 | #1610 + #1613 + #1624 | **This window's own bookkeeping residue, in the three files it rewrote, and all three are cheap only while the landings are warm.** #1610: four sites in `contentrestricts.go` no longer agree on §3.4.6.3's (a)/(b)/(c) sentence and the `{open content}` arm still asserts the reading #499 ruled OUT. #1613: three `OverReport` test identifiers name an over-report `e0e3c46` retired, so all three now pin its absence. #1624: #267's case-3 marker states its narrowing inside the skip bullet while the `##defined` bullet needs it too |
| 8 | #1560 + #1584 | **Both land in `xsd/contentmatcher.go`'s comments and nowhere else**, in code this window rewrote twice MORE (`46dc042`, `a8b3b71`). #1560 is the EXPORTED contract — `(*Schema).ContentMatcher` never states that `t` must be a complex type of the receiver, and a `<group ref>` resolves against the RECEIVER's index. #1584 is now **seven** sites, widened twice. Neither may re-open #1601's ruling |
| 9 | #1324 | **The cheapest discharge in the queue**: one ruling, on whether `CLAUDE.md`'s gate block takes a hazard sentence at all, that discharges `blocked` **#1344** in either direction. #1344's body admits a trigger of exactly this shape |
| 10 | #1626 + #1089 | **The CLI disclosure pair, and both halves were re-found by an outside reader within one week.** #1626 is the error path dropping a token it holds; #1089 the success path naming no route. Same file, both `package main`-internal, `## Surface: none` on each. **Read #1089's new comment before either**: `helpPointer`'s own godoc already refused a `go doc` pointer on #870's grounds, and that must be answered rather than stepped around |
| 11 | #1627 | **This pass's own instrument finding, and arm 1 is one refspec argument.** `wipsurvey`'s refspec is `wip/*` while `WORKFLOW.md:79`'s index is `wip/*` AND `parked/*`, so four stamps have quoted a `parked/*` count no tool computed. Arm 2 (the seven `claude/*` refs) is a judgment call with no defect behind it yet and may be declined on the thread |
| 12 | #1499 | **Banded twelfth for a THIRD time, and the test the last stamp set cannot be run.** It ranks `kind/refactor` — **83 open, still the largest kind in the queue**. The last stamp made #1589 the counter-example and said *"if row 10 lands and #1499 is still unconsumed, the reading is that the MEASUREMENT is what a refactor needs"*. **Row 10 did not land either.** So the prediction is untestable and the row's own argument has narrowed to: nothing lifts a refactor here, measured or not |

**Named below the band, deliberately.** **#1576** (whether §3.10.4.1 key-skipped
reads THROUGH an `{open content}` record) is #1553's open question and is no longer
warm; take it with any `validate` row. **#1572** owns the two unowned
`GAP(xpath)` markers in `icpath` — same class as row 4, and `gapaudit` names it as
a candidate owner at **seven** group-1 rows, more than any other issue. **#591**, **#831**, **#1324** and **#479** each discharge
exactly one `blocked` issue on landing and none moves a lane; **#1324** is banded
ninth because it is the cheapest, and **#479** is the largest of the four.
**#1545** is the chain's last unfinished piece — neither statement of the join rule
names `GOXSD_WITHHELD=1`, and **its `## Spec` was re-anchored by the last pass**,
so locate by sentence and not by line. **#1536** matters more than its size
suggests now that `suiteindex` is the prescribed prediction instrument. **#345**,
**#1379**, **#1462** and **#1511** are the fail-open bookkeeping family, `schema`
up-or-flat, and **#267 left it this window**. **#1473**/**#1474**/**#1476**/**#1477**
are the `regex` Appendix G family, four issues over two files. **#743** is the
`src-redefine` leniency and is in class 7 above. **#888**, **#889** and **#1119**
each want M6/M7 machinery that does not exist. **#1522** (the 400-line M4/M5
prose) and **#1153** (the survey recipe has no test) are both compounding and both
untouched for FIVE stamps — #1153 is now adjacent to rows 1 and 11. **#1596** (a
`gen` invocation answers the same two lines whatever its flags) is where the next
`cliuser` run should be pointed. **#1546**, **#1548**, **#1429**, **#1539**,
**#1518**, **#1452**, **#1496**, **#1497**, **#1502**, **#1508**, **#1454** and
**#1291** are each real and each one session.

### Next planning action

1. **Row 2 is this band's whole falsifiable claim and it is narrower than the last
   two.** #1494 has been banded five times; if it is taken and its `PREDICTION:`
   block still answers `not derived` with `suiteindex`'s `//` census in the tree,
   then **the instrument is the wrong shape, not the missing one**, and the next
   stamp must say what shape is right instead of banding a sixth census feature.
   **If it is banded a sixth time without being taken, that is a different finding
   and the more important one**: five stamps of naming a row has not moved it, and
   the band's ordering is not the mechanism that would.
2. **Do not re-argue the lane-first rebuild; DO record the shape it cannot see.**
   **FIFTEEN rows over three windows were taken in order; FOURTEEN landed and one
   (#1588) was declined `not_planned` rather than abandoned**, so the ordering
   picks startable work reliably. The magnitudes were +52, +2 and +4.
   **The new datum is #1601**: one issue, two landings, `instance` +4 and 0, the
   route chosen after banding. **An issue offering both a retirement and a
   permanent-ruling route has its lane movement decided at grounding**, so no
   ordering can rank it on movement. Whether such issues should be SPLIT at filing
   — the measurement as one issue, the ruling as its fallback — is a
   **`/retro`-or-steward question and no `/backlog` can answer it.** Filed nowhere
   yet, deliberately: one window is one reading.
3. **The permanent-ruling genre is the window's real product and the lane column
   hides it.** Four comment-only landings that banked nothing took `gapaudit` group
   1 from **21 to 17** and the tree's dead-end count from 2 to **0**, first time
   ever. A stamp that ranks only on lanes would have banded none of them. Rows 4,
   7 and 8 exist because of this.
4. **M4 has not moved in FOUR windows** — 54 open, 22 lane-visible, 23
   `kind/refactor`, byte-identical across FIVE stamps. Whether the remaining
   non-lane-visible work belongs in M4 at all, or whether M4 has become a
   directory of `parser`/`xsd` work that outlived the parsing milestone, is a
   **`/retro` question and no `/backlog` can answer it.** Carried for a **FIFTH**
   stamp and still the longest-carried item on this list.
5. **`/retro` has now missed TWO cycles, not one, and this is an escalation from
   the last stamp.** Re-derived here from `git log -- docs/LOG/` rather than
   carried from prose: the last retro landed **2026-09-06** (`f97c317`, *"retro
   2026-09-06 (weekly, eighth)"*). **09-13 was missed; 09-20 was DUE YESTERDAY and
   did not land.** Items 2, 4 and 6 are all routed there, and item 2 is new. This
   is an observation about the schedule, not a filing: `docs/ROUTINES.md` owns the
   cron and nothing in this container can read whether the routine fired.
6. **The persona trigger is still `/retro`'s to decide, and this pass finally adds
   a datum.** Six distinct findings: **three** re-found an existing issue, **one**
   was new (#1626), **two** were FALSE against the tree and cost a verification
   round to reject. A second window of the repair rate, plus the first cost figure
   the record holds for a consultation. Both personas also found LESS than their
   predecessors on the same surface, which argues for rotating the surface (#1596,
   #1432) rather than the cadence.
7. **The human decision blocking #1002 is unchanged and is carried for a
   TWENTY-FIFTH stamp.** It waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent, so no `/backlog` can move
   it and none should try.

**What this pass did, so the next one can tell a completed run from a skipped
one.**

**TWO issues filed, ZERO closed, ZERO label changes.** **#1626** from the
`cliuser` report and **#1627** on the filing condition the last stamp set for the
branch namespace. Nothing was closed because nothing was stale: the marker census
found zero dead ends and no unowned group-1 row, the unblock sweep measured zero,
the queue audit found no mislabel and no duplicate, and the eleven issues this
window's post-land passes filed were all checked and all carry the required
sections. **The pass's products are the ORDERING, two filings, one body repair and
four thread comments.**

**ONE body PATCH, and ZERO title PATCHes.** **#1156** — five stale premises, all
from landings that had nothing to do with it: Row 2's marker moved `:1168` →
**`:1189`** (its third anchor, two of them already wrong); the census absolutes
re-measured 69/18→17 to **68/17→16**; **#499 has CLOSED** and the marker Row 1
discharged still cites it, which P3b makes legal and a reader would otherwise
repair; **#345 is `ready`** where the body says `blocked` on #250; and the
deliberately-unowned pair re-anchored `:345`/`:541` → **`:392`/`:588`**, with the
new P3b-ruled `:100` named so it is not read as a fourth unowned site.

**FOUR thread comments.** **#1292** (third libuser sighting, and the *"what should
my service do today"* bar the doc arm now has to clear), **#1089** (second cliuser
sighting, and `helpPointer`'s #870 counter-evidence against its own route (a)),
**#1382** (the falsified premise, recorded so it is not re-derived), **#779**
(classes 6 and 7 re-measured with both denominators, the first confirmed
prediction this thread has made, and the over-report now NINE rather than ten).

**Every body PATCH and comment went through `gh api -F body=@FILE`** and is
byte-faithful; every body READ came from repository-scoped REST rather than the
MCP channel (#764). **GraphQL is still a 403** — `gh issue list` was tried once
and answered *"GitHub GraphQL is not available from Claude Code sessions"*,
exactly as `docs/ROUTINES.md` records, and the REST walk under **Survey input**
served every read and write after it. **`gapaudit` and `wipsurvey` were each run
once, on `main` at `40eaeac` after `git fetch --unshallow origin`; ONE `state=all`
walk, clean on the first request, both integrity checks passing.**
**`testdata/xsdtests` was NOT initialized**, so no suite figure here is this
pass's own measurement — the lane table is the committed expectations census,
which needs no submodule, and the four flips were verified by `diff` against
`a2a72bf`.

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
