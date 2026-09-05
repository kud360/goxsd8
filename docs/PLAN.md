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

## Status — 2026-09-05 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **SIX** landings and they are the last band's rows **1 through 6, taken in order** (#1215, #1216, #1136, #1229, #1230, #1223) — the longest in-order run this section has recorded, beating the five the last stamp reported. It measured **`schema` +9 across the two lane movers**, and **BOTH beat their own predictions again** (#1215 +4 against +2 firm, #1216 +5 against +3 firm), which makes **five consecutive landings in one family under-predicted** and moves **#1239**, the census instrument, to band row 1 over every lane slice. The namespace is **idle for the second consecutive stamp** (zero LIVE, zero CLAIMED) with **six clean open-and-close leases** in one window. The marker census moved **69 → 71**, its first move in two stamps, both markers left by #1216 and both already tracked (#1242/#1243). The **SIXTEENTH persona consultation** ran against a materially changed CLI surface: **seven findings — three filed (#1260, #1261, #1262), three folded (#1189, #1122, #1003), one dismissed with the reason recorded**. Seven open bodies were corrected where this window falsified them — #1189, #1122, #1003, #1224, #1233, #1251, #1156 — plus #1140, #1242, #1243 and #1258 on evidence this pass measured. **#1140 is NOT closed**: the standing instruction to close it rested on a premise this pass falsified with a second measured sighting)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13937 | 1461 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### Six of six band rows landed in order, and both lane movers beat their own predictions

**`schema` 13928 → 13937 (+9); `instance` flat at 11029; `datatypes` flat.**

| landing | commit | predicted | banked | milestone |
|---|---|---|---|---|
| **#1215** | `976bdc2` | `schema` **+2 firm** plus one conditional | **`schema` +4** | **M4** |
| **#1216** | `aeaf59e` | `schema` **+3 firm** | **`schema` +5** | **M4** |
| **#1136** | `d854e1d` | unchanged | unchanged | none |
| **#1229** | `20010ed` | unchanged | unchanged | **M5** |
| **#1230** | `38ade3e` | unchanged | unchanged | **M5** |
| **#1223** | `4ab65d6` | unchanged | unchanged | **M5** |

**The whole band consumed itself in order, and nothing was inserted ahead of
it.** Rows 1 through 6 landed as ranked. That has never happened before — the
previous best was five — and it is the strongest evidence this section has that
the band ordering is being read and used rather than re-derived per session.

**BOTH lane movers over-delivered, and that is the fifth consecutive time in
this family.** #1206 under-predicted on ENCODING, #1215 on DIRECTORY SCOPE
(`saxonData/TargetNS/target002.n.xsd`, plain UTF-8, one grep away from a census
scoped to `ibmData/`), #1216 on CLAUSE SHAPE (`s3_2_3si03.xsd`, sitting beside
si02/si05/si09 in the same decoded directory, failing clause 6.2 where the
body's table had been built by matching 6.3.2's shape). **Each fix has been
aimed at the last mechanism and none at the cause.** Both over-deliveries were
caught before banking by gate part 4's `-v`, so nothing leaked into the
expectations — this is a cost, not a correctness fault, and the cost is an
unexplained figure the arbiter must account for under the ratchet rule before it
can write. **#1239 is the filed instrument and it is now band row 1**, which is
the cartographer banding rule applied as written: friction the log records in
consecutive sessions outranks a lane slice, and nothing else will ever lift a
`kind/tooling` issue that moves no lane by construction.

**Four of the six moved no lane and all four are correct to.** #1136 is a
`parser` refactor that replaced four local run-order guesses with one recorded
decision; #1229, #1230 and #1223 are all `cmd/goxsd8`, which no lane can reach —
`go list -deps ./conformance | grep -c cmd/goxsd8` is **`0`**, re-measured by
this pass. **#1223 is the sharpest of the three**: it added a fifth exit code
(exit 4, an instance the assessment declined to decide) and it is the CLI half
of a verdict-reading the library half of which is still open as **#1122**.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and this
pass's 711-issue post-write feed:

```
ISSUE  BRANCH         LEASE AGE  VERDICT  REASON
732    wip/issue-732  309h0m0s   RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  486h58m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846  259h19m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872  452h59m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  381h12m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  320h17m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993  264h40m0s  RETIRED  wip/issue-993: issue #993 is closed
```

**ZERO LIVE and ZERO CLAIMED for the SECOND consecutive stamp**, and this one
covers six landings rather than none: `wip/issue-1215`, `-1216`, `-1136`,
`-1229`, `-1230` and `-1223` were all opened and all deleted at merge inside the
window, so **six leases opened and closed cleanly and none survived to be
counted here**. Every band row below is startable without colliding with a
claim, and `ready` overstates startable work by **nothing**.

**The same seven RETIRED refs, unchanged row for row** — the set has not moved
in three stamps. All seven closed `not_planned`; they are parks and supersedes
whose content is *supposed* not to be in `main`, and none owes a supersede.
Cloud containers cannot delete remote refs, so these accumulate by design and
are not a finding. **Zero `parked/*`.**

**FOUR non-`wip` `claude/*` refs stand, unchanged in tip and in verdict from the
last stamp:**

| ref | tip | dated | reads |
|---|---|---|---|
| `claude/eloquent-cerf-39rk64` | `0abeab6` | 2026-08-29 | NOT-ancestor (a clone-depth artefact, #802) |
| `claude/eloquent-cerf-3xu0ki` | `62d5143` | 2026-09-03 | ancestor, carries nothing (#404's squash) |
| `claude/eloquent-cerf-8jq9o6` | `7841e98` | 2026-08-29 | NOT-ancestor, unchanged |
| `claude/eloquent-cerf-adewly` | `5d03049` | 2026-09-04 | ancestor, carries nothing (#1188's post-land squash) |

**No new `claude/*` ref appeared across six landings and six post-land passes**,
which is what a clean `wip/`-only workflow looks like from here. This container
sees **50 commits** of `origin/main` (the last stamp saw 51), so both
NOT-ancestor rows are the shallow-clone artefact **#802** owns and were
dispositioned from commit dates rather than from `git log`. Listed for human
triage, not acted on.

### Marker census

`go tool gapaudit` over this pass's whole 711-issue post-write feed: **71
markers across 8 areas** — `xsd` 35, `validate` 17, `xpath` 6, `xml` 4,
`parser` 3, `value` 3, `conformance` 2, `cmd` 1.

**The census MOVED for the first time in two stamps: 69 → 71, and both new
markers are #1216's.** They sit in `parser/produce_complex.go` at `:3112` and
`:3125`, are tagged `[xsd]` (which is why `xsd` moved 33 → 35 while `parser`
stayed at 3), and both were **filed as trackers by that landing's own post-land
pass** — #1242 and #1243. This is the family replenishing itself as it is
worked, which is what to expect.

**Group 1 moved 17 → 19 and group 2 held at 27.** Both new group-1 rows are the
two markers above: `gapaudit` retires a row on a **citation and on nothing
else**, and #1244 deliberately named the repointing rather than doing it, making
it part of #1242's and #1243's own landings instead. Group 2's membership
churned without its count moving — #1215 and #1216 left by closing, #1242,
#1243 and #1251 joined.

**ZERO group-1 rows carry no annotation at all, so the tool's own filing rule
selects nothing and there are ZERO untracked GAP sites — SEVENTH consecutive
stamp.** Every group-1 row prints at least one candidate owner or resemblance;
the two new ones print 34 and 33 respectively, which is the whole-repository
feed behaving as `gapaudit`'s own doc predicts (#1108) rather than a signal.

**One finding this pass acted on rather than only counted.** Both new markers
end *"No open issue owns retiring it"* — true when written, false within the
hour, because #1244 filed #1242 and #1243 against them. **The sentence is a
false claim standing in the tree today**, and it is now written into both
issues' Acceptance as something the landing must delete or repoint, never leave.
**#1156** — the two `contentrestricts.go` markers invisible to the owner join —
was restamped for the same reason: its expected totals said census 69, group 1
17 → 15, group 2 25 → 24, and all three absolutes had drifted while its two
DELTAS had not.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 13 pages, PRs filtered locally: **711 issues, 269 open, 442 closed**
(551 PRs excluded).

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **52** | **111** | active |
| **M5 — Instance validation (XML)** | **14** | **23** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **248 `ready`, 21 `blocked`, 0 `needs-replan`** — every
open issue carries exactly one, verified mechanically, and the two sum to 269
with no gap and no double-labelled row. By kind: `kind/refactor` 73,
`kind/process` 56, `kind/gap` 55, `kind/tooling` 34, `kind/bug` 27, `kind/story`
23, `kind/docs` 14, `kind/feature` 4, `epic` 2. By area: `parser` 77, `meta` 71,
`xsd` 59, `conformance` 32, `docs` 29, `validate` 22, `cmd` 19, `value` 14,
`builtin` 10, `xpath` 6, `xsderr` 4, `loader` 2, `regex` 2.

**M4 moved 50 → 52 open and 109 → 111 closed**: #1215 and #1216 closed, and
**four** were filed onto it — #1242 and #1243 by #1216's post-land, #1246 and
#1247 by #1136's. **M5 held at 14 open and moved 20 → 23 closed**: three closed
(#1229, #1230, #1223) and three filed (#1250, #1251 by #1229's post-land, #1257
by #1223's). **M5's count did not move and its lane did not either**, for the
same mechanical reason: every issue that entered and left it this window is
`cmd/goxsd8`, which no lane can see. Neither milestone's count is its lane's
remaining work, which the two milestone sections below say in their own words.

**`ready` 248 is the honest startable count, with nothing to subtract**, because
the namespace is idle — the second consecutive stamp with no exclusion-on-flight
row in the band. There is no numeric cap on `ready` and its size is an output
(#347); it grew by seven this window because seven post-land and `/backlog`
filings outpaced six closures.

**The unblock sweep measured a clean zero for the EIGHTH consecutive stamp.**
All **21** open `blocked` bodies were read and their `## Depends on` sections
checked against this window's six closures: **not one of the 21 names any of
them**, in any position. **Zero relabelled.** The 21 partition cleanly and
should be re-read that way rather than re-scanned whole:

- **NINE are triggers rather than issues** and say so in their own
  `## Depends on` — #79, #555, #692, #841, #925, #1002, #1080, #1220, #1224.
  **Do not re-scan these on the next sweep.**
- **THREE have every named issue dependency CLOSED and each states in its own
  body why that is not a discharge** — **#16** (a cliuser reference that is
  never worked directly; its `gen` criteria are unfileable until M9), **#1042**
  (its live dependency is the XPath evaluator itself, a trigger; the closed
  #719 it names landed the site inventory it replaces) and **#1051** (three of
  six conservative-decline buckets still unfiled). All three were re-read in
  full this pass, not inferred from label state.
- **NINE still have at least one OPEN named dependency**, checked issue by
  issue: #248, #250, #267, #345, #415, #456, #593, #717, #871.

### Persona consultations — the SIXTEENTH ran, and the trigger fired as named

The cartographer role-plays no persona and does not spawn one (#416): it has
read the source, so a verdict it produced would launder an insider's opinion as
an outsider's. **The orchestrating session ran both personas against the
current published surface at `40011a2` and handed this pass their reports; what
follows is the folding, not the consultation.**

**The last stamp named a trigger rather than scheduling a pass** — *"a
`cmd/goxsd8` landing, a README Library-section landing, or a new `xsd` query
accessor"* — and three of the six landings fired it: #1223 added an exit code
and rewrote the CLI contract, #1230 added a refusal, #1229 rewrote a README
paragraph. **Naming the trigger worked**, and it is the second consecutive
window in which it has.

**Seven findings: three filed, three folded, one dismissed.**

- **cliuser, three findings.** **#1261** is new and filed — `parse`'s summary
  BODY is specified nowhere: the seven bucket labels, their fixed order, the
  `namespace:`/`components:` lines, the two-space indent, that `types` merges
  simple and complex, and that `components:` is the sum of the seven. README
  commits that block to **stdout** so scripts can read it (#1066) and calls
  `go doc ./cmd/goxsd8` authoritative, and neither copy says what the block
  contains. **#1260** is new and filed — see below. The third **folded into
  #1189**, which already owned the `-help=true` three-spellings gap.
- **libuser, four findings.** **#1262** is new and filed —
  `handleReadErr` charges `xml-wf` for **every** non-`io.EOF` error the caller's
  `io.Reader` produces, so a dropped connection and a client's bad markup are
  indistinguishable through the `xsderr.RuleOf` narrowing pattern `xsderr`'s own
  doc recommends, while that sentinel's two doc copies scope it to well-formedness
  faults alone. Two folded — the `Unevaluated` formatter into **#1122** and the
  buried `xs:assert` disclosure into **#1003**. One **dismissed**; see below.

**#1260 is the one whose failure mode is a green gate.** An unresolvable
`<xs:include>` in a schema ARGUMENT degrades in total silence: `goxsd8 parse
inc.xsd` prints a full summary and exits **0** with **zero bytes on stderr**,
and `goxsd8 validate -schema inc.xsd order.xml` does the same, off an assembly
that lost a document. Both reproduced from a built binary by this pass. **It is
the ruling #1251's own Acceptance says it defers** — that issue owns the
instance-hint side and states in as many words that the `-schema` side *"needs
its own ruling"* — and it covers `parse`, which #1251 could never reach.
Neither gates the other and both bodies now say so.

**Every citation in the three filed bodies was re-checked against the tree at
`40011a2` before filing**, and every repro was re-run here rather than taken on
the report's word: the three `-help` argument positions, both silent
degradations, the nine summary labels read out of `summarize`'s own `buckets`
slice, and the `xml-wf` collapse probed one layer BELOW where the persona found
it — directly against `xmltree.NewReader`, so the issue is charged to the line
that makes the choice rather than to the adapter that surfaces it.

**ONE finding dismissed, and the reason is a rule rather than a judgment call.**
libuser reported that `xsderr.Error.Msg` textually embeds the wrapped cause's own
rendered `<loc>: [<rule>] <msg>` line, duplicating what `Err` carries
structurally — and then ruled it a design tradeoff itself, because
`validate/doc.go:118` states it outright (*"Error() still renders that verdict
into the message as well, for a reader who holds only the string"*). Its ask was
a callout in README's Library section. **That is a second copy of a fact
`validate/doc.go` owns, in a second file** — STYLE D3, and exactly what #1122's
own Acceptance already refuses (*"Do not restate `doc.go`'s paragraph — point at
it"*). Nothing to file and nothing to fold. **The baseline half of both reports
is worth as much as the findings**: exit 4 and its severity aggregation, the
`-schema -` refusal, the missing-hint silence, the multi-`-schema`
`sch-props-correct` clause 2 message, the `-format` vocabulary, flag
positioning, `--` carrying no end-of-options meaning, `GOXSD_DEBUG`'s honest
inertness, and README's worked violation example all reproduced **byte-exactly**.
The next consultation does not re-check them.

**#1033 remains the one row no persona has ever looked at.**

### Working band

**Re-derived from this pass's evidence.** Rows 1 through 6 of the last band all
landed, in order; nothing is in flight, so take from the top and re-run
`wipsurvey` first anyway.

| # | issue | why here |
|---|---|---|
| 1 | #1239 | **A `kind/tooling` row above two lane slices, and the banding rule says so in as many words.** Nothing takes a whole-suite fixture census, and **FIVE consecutive landings in one family have now under-predicted** — #1206 on encoding, #1215 on directory scope, #1216 on clause shape, with #1215 and #1216 both banking over inside this window. Each fix was aimed at the last mechanism and none at the cause. **Band rows 2 and 3 both name this tool in their own Acceptance** (*"ask which fixtures carry the CONSTRUCT, over the whole corpus, decoded"*), so it is what makes their predictions honest without gating either. Friction recorded in consecutive sessions outranks a lane slice, and a tool that moves no lane will never be lifted on lane grounds |
| 2 | #1242 | **The direct continuation of the landing that found it, and the family's only remaining reachable failure of clause 6.1.** A local `<attribute ref="...">` writing `targetNamespace` is accepted and clause 6 is charged on it nowhere; #1216's grounding established that on the `name=` path 6.1 is discharged upstream by 3.1, so this is 6.1's one reachable failure **anywhere**. `TestProduceRefAttributeTargetNamespaceStaysAccepted` pins today's acceptance and is the issue's inverse — leaving it green proves the charge did not land. **Predicts no figure on purpose**; census by construct, with row 1's instrument if it exists by then. Also deletes the marker sentence that falsely claims no issue owns it |
| 3 | #1243 | **The `use="prohibited"` twin, and the harder of the pair — take it second.** `produceAttributeUse` returns on the prohibited token ahead of both its `hasRef` arm and `produceLocalAttribute`, so clause 6 never runs; closing it means lifting the call up ahead of that return, which **reorders clause 6 against clause 4** at a call site of its own and inverts the order #1216 deliberately chose. It also carries a research task the marker deliberately omits: `prohibitedAttributeNames` DOES read the `targetNamespace` on this path (`:2726`), so the `{prohibited attribute names}` QName is minted in whatever namespace the attribute names. Same marker-sentence deletion as row 2 |
| 4 | #1140 | **n=2 as of this pass, and the second sighting is sharper than the first because the misreading arrived PRE-FORMED in the session brief.** This `/backlog` was briefed *"you do NOT have `gh` CLI access"*; three commands in the same container settle it — `gh auth status` reports the token invalid, `gh api repos/kud360/goxsd8` returns **200**, and `gh issue list` 403s with a GraphQL error whose own text says *"Use REST via `gh api repos/{owner}/{repo}/...` instead"*. `docs/ROUTINES.md:42-61` predicts all three and a reader entering at `## Survey input` never crosses it. **The last stamp's standing instruction to close this if it went untaken from a band row is DISCHARGED IN THE OTHER DIRECTION**: the premise was that slippage was the only new evidence, and n=2 is different evidence |
| 5 | #1205 | **Cheap, correct, ratchet-neutral by construction — and the argument that kept it at the band's foot is gone.** `final=`/`abstract=` on a local element are prohibited `use="prohibited"` unconditionally by `xs:localElement`, on both the `ref=` and inline paths, and neither charges either attribute. It footed the last band because it was what made #1136's arm five checks deep; **#1136 LANDED (`d854e1d`) and replaced those four local guesses with one recorded convention**, so that cost is discharged and only the cheapness remains. The **#768 collision is named on both sides** — its fixture stops reaching `produceLocalElement` the moment this lands — and is a hazard, not a blocker |
| 6 | #1260 | **The sixteenth consultation's sharpest finding, and the only new one whose failure mode is a GREEN GATE rather than a missing sentence.** `parse` prints a full summary and exits 0, and `validate` exits 0, off an assembly whose `<xs:include>` resolved to nothing — zero bytes on stderr in both, the fact reachable only behind `-v`, which README frames purely as debug logging. Two settled dispositions and the body picks neither: report it, or state the silence the way README already states the missing-hint case. **It is the ruling #1251's Acceptance defers**, and it is the half that covers `parse` |
| 7 | #1203 | **A latent ratchet ambush #404's landing created, and the queue still has two issues that could trip it.** Two of #404's eleven banked `schema` passes — `addB014` and `schZ006` — reject for a fault the suite does not intend, so when #603 or #703 repairs the over-rejection those passes flip down as a spurious `Regressed` the repairing session did not cause. A banked pass resting on an over-rejection is invisible in the lane file; this measures it into a known figure. No production change — a measurement and a ruling. Cross-references #1002, whose human ruling its measurement may feed |
| 8 | #1007 with #1231, #1123, #1189 and #1261 | **Now FIVE issues, one session, three contract copies, one coupling test.** All five edit `cmd/goxsd8/doc.go`, `main.go`'s `usage` const and `README.md` together, and `TestUsageCoversContract` pins the first two by **29** substrings — a fifth move of that count (5 → 12 → 16 → 24 → 29) with the property it guards unchanged, which is #398's whole argument. **#1189 grew this window** and is now the strongest single row: its `-help=true` claim is false in **both** pre-subcommand positions, and the `leadingFlagFmt` one tells the reader to move the flag after the subcommand where it is rejected again — a fix-it that does not fix. **#1261 is the newest**, and it is the only one of the five about what the CLI PRINTS rather than what it says about itself. Taking them apart means rebasing the same three files five times |
| 9 | #1227 | **The pen bound is written and nothing checks it — first sighting AFTER the rule existed, which is what makes it fileable.** `docs/WORKFLOW.md:190-192` binds who may hold the pen and `develop.md:60` binds step 4, but **landing precondition 3 iterates over MASON commits** (`docs/WORKFLOW.md:280`), so a branch with none has nothing to iterate. #636 is the only other witness and is **pre-rule** by one day, so this is n=1 post-rule and the deliverable is a runnable check, not a fifty-seventh `kind/process` issue. Both landed precedents are named in its body: #1018's doc-only precondition 4 and #963's `landcheck` |
| 10 | #1262 | **A defect inside a pattern the package documents, which is what lifts it above the doc rows.** `xsderr/doc.go:71-73` offers `RuleOf` as the narrowing path *"for callers that hold a plain error"*, and `xml-wf` is what a transport failure comes back as — so a service routing 4xx-versus-5xx on it misroutes. **Branch (b) is three doc sentences; branch (a) is not small**, since `encoding/xml` labels none of its own errors, so ground the branch question first. Below the contract-copy row because the fact IS recoverable — `errors.Is`/`errors.As` reach the cause through the wrap, verified — just not through the advertised `Rule` |
| 11 | #1167 | **Two measured sightings, #414 and #1115.** `gapaudit` reconciles marker → issue and issue → marker; nothing audits prose that points *at* a marker or its file, and the stale-marker sweep has missed an inbound site in the deleting package twice. `gapaudit`'s own group 1 candidate-owns it against `validate/assess.go:853`. PRINCIPLES 27 says a repeated grep wants a tool |
| 12 | #1251 | **The hint-side twin of row 6, and it foots the band because row 6 subsumes its harder half.** A hint naming a document that is not there costs an operator exit 1 with an empty stderr, and the plumbing — `AssemblyReport.Unfollowed()` — already records it. Its three carried facts are worth reading before row 6 is taken, whichever lands first: `UnfollowedDirective` carries no `schemaLocation` value, every `-schema` argument and every hint is a directive in the SAME synthesized wrapper root, and that wrapper is emitted on one line so attribution within it is column-only. Its line citations were restamped by this pass; #1230 and #1223 had moved all of them |

**Below the band, and why**: **#1246** and **#1247** are #1136's own follow-ups
and are ranked on ordinary grounds — #1247 is a comment-roster correction that
pairs with whoever next opens `checkS4SChildOrder`, and **#1246 is a behaviour
change with real ratchet exposure** (it changes which fault class a doubly
violating document reports) that needs its own grounding and should not be taken
in the same window as rows 2 and 3. **#1250** is #1251's sibling on the same code
path and either can land first. **#1156** is comment text only; its absolutes
were restamped this pass and its deltas are what it owns. **#1201** is correct,
needs a warden pre-flight for its 1a fix, and **zero suite cases exercise it**;
take it opportunistically with #1203, and it retires the two `GAP(conformance)`
markers when it lands. **#1183** is cheap, correct, moves nothing and makes
#1051's stage-2 deletion three predicates smaller. **#1196** measures the live
residual for #1051 and **its own body forbids banding on the numbers it
produces**. **#1217** pairs with band row 4 (both cheap, both `meta`).
**#1135** pairs with #1205 and #1243's neighbourhood; never take it together
with the arm they charge into. **#1036** carries one settled disposition in
either direction and is unchanged. **#1258** is the MCP `list_issues` divergence
and **is still n=1**: this pass re-probed it a third time, from a fresh
container and a different day, and #872 came back **correctly** as
`CLOSED`/`needs-replan` — a non-reproduction of a fault its own body calls
transient, recorded on it and explicitly not counted as a second sighting.
**#1257** is opportunistic (take it with whoever next opens
`cmd/goxsd8/validate.go`). The remaining fifteenth-consultation doc rows —
**#1232**, **#1233** (restamped this pass; #1229 landed and rewrote the very
README paragraph it cites), **#1234**, **#1235**, **#1236** — are one-sentence
fixes with settled directions; #1236 is the README twin of **#513** and the two
are one session. **#849** carries a `## Cost of delay` reading *"a steward
re-ranking is warranted"* — an unranked issue asking for a rank, routed to
`/retro`'s steward drift review as a third item beside #841 and #1080.
**#1122** and **#1003** both grew this window and both stay below the band as
one-line README fixes. **#1033** is still the only row no persona has ever
looked at.

**There is no `Increasing` steward ranking anywhere in the band.** #849 is the
nearest, and its own body records the ranking as falsified and unreplaced, so
the `kind/refactor` cost-of-delay clause of the banding rule selects nothing
this stamp.

### Next planning action

1. **Take #1239 before the next lane slice, and hold it to the five
   under-predictions it was filed against.** Rows 2 and 3 are the two lane
   slices in the queue and **both of their bodies refuse to predict a figure**,
   naming this tool as what a census-by-construct would need. Taking a lane
   slice first is permitted and the band says so by ordering, not by blocking —
   but a session that does take #1242 or #1243 first must **census by CONSTRUCT
   and not by clause shape, over the whole corpus, decoded**, because that is
   the exact mistake #1216 made and the third distinct mechanism in a row.
2. **#1140 is not closed and the question is not to be re-derived.** The last
   stamp conditioned closing it on *"still untaken at the next stamp FROM a band
   row"*; it was untaken, and a **second measured sighting arrived in the
   interval** — in this pass's own container, with the misreading pre-formed in
   its brief. That is different evidence, and the standing instruction is
   discharged rather than carried. If a THIRD window passes with no new sighting
   and no take, reopen the question then.
3. **The sixteenth persona consultation is folded and nothing from it is handed
   off.** Seven findings: three filed (#1260, #1261, #1262), three folded into
   issues that already owned them (#1189, #1122, #1003), one dismissed on STYLE
   D3 with the reason written into this section. **The next consultation is owed
   against whatever next changes the observable surface** — name the trigger
   rather than scheduling a pass, which has now worked two windows running. Band
   rows 6, 8, 10 and 12 all change it; rows 1 through 5 do not.
4. **The two markers #1216 left assert a falsehood in the tree and their
   deletion now rides on #1242 and #1243.** Neither issue may land leaving *"No
   open issue owns retiring it"* standing. This is the concrete form of the
   general question **#1167** owns — prose that points at a marker with nothing
   auditing it — and if a third instance of a stale marker-adjacent sentence
   arrives before #1167 is taken, band #1167 rather than filing a third
   instance.
5. **The human decision blocking #1002 is unchanged and is now carried for a
   TWELFTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — and (b) depends on **#1042**, filed and `blocked`. **No
   agent should attempt either.** Band row 7 (#1203) may surface a fresh instance
   of the wrong-reason-pass class (a) is meant to cover; that is a datum for the
   ruling, not a licence to act on it.
6. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3). **M6 tier 2 itself is uncarved** —
   `$value` binding, an F&O function library, typed comparison — and is too big
   for one issue. That carve is a `/backlog` act at the M6 opening, and #1042 is
   the thing it must slice around rather than a blank page. **This window raised
   its visibility rather than its readiness**: #1223 shipped exit 4 and #1122
   gained the measured fact that a schema carrying one trivial `xs:assert`
   leaves **every** instance undecided, so the cost of #1042 not existing is now
   observable from the terminal. Note that #1042's `## Depends on` names #719,
   which is **closed**: the live dependency is the XPath evaluator itself, a
   trigger, and the body says so.

**Standing, and re-checked rather than restated.** Four unlanded corrections still
target one paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**,
**#646**, **#679**, **#912** — and whichever lands last rebases three times.
**The next `/retro` inherits seven**: #692, #925, #841, #1080, the
fold-the-five-species question (#635, #912, #609, #510, #646), the
`[tests that cannot fail]` **pattern** (routed by #472's post-land, with no filed
carrier left since #999 landed), and **#849's owed steward re-ranking**, the
third item for the steward's drift review. **#841 is still the counter-example
the steward-ranking rule cannot reach**: a `kind/refactor` with a steward
ranking, `blocked` because its trigger has no mailbox, fired twice without a
ruling. **#1136's landing banked one more `/retro` datum and deliberately did
NOT file it**: STYLE P3a forbids an unenumerated DIRECTION claim on reasoning
that reaches a MEMBERSHIP claim identically, and nothing in `docs/STYLE.md`,
`mason.md` or `arbiter.md` binds a policy roster to that standard. That entry
searched all 263 open issues and the closed `kind/process` queue for an owner and
found none, so it recorded an **anecdote** rather than a fifty-seventh
`kind/process` issue — #1247 is the concrete instance, not the rule. The
`gh --paginate` page-2 403 trap stays a `/retro` datum and not a filing. The
CTA cohort's 45 banked `instance` failures remain unattributed.
`gate.yml` runs and is still not a required status check, which only the
repository owner can change.

**Environment, one witness each.** Repository-scoped REST served every read and
every write here: **14 issues written** — **3 filed** (#1260, #1261, #1262) and
**11 open bodies corrected** (#1189, #1122, #1003, #1224, #1233, #1251, #1156,
#1140, #1242, #1243, #1258) — plus this section's own replacement. **#1260's
creation event carries a `__probe__` title in its history**: the write path was
tested with a throwaway issue and that issue was then rewritten into #1260
rather than closed as junk. Read its body, never its first revision.
**The paginate recipe ran to 13 pages twice**, before and after the writes: 12
full at 100 both times, page 13 short at **59** then **62**, and the post-loop
fullness check passed on both runs. **`gh api` works in this container and
`gh auth status` says it does not** — the whole of band row 4, measured rather
than argued. GraphQL is refused for `gh issue list`
by the session's own proxy, which names REST as the remedy in its error text.
The shallow clone truncated `origin/main` to **50 commits** (**#802**), which is
why two `claude/*` ancestry reads came from commit dates. **No conformance
measurement was taken by this pass**: the lane table above is the committed
expectations, which `docs/WORKFLOW.md` names as the lane score (#1120).

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
(`final=`/`abstract=`, the other grammar-prohibited pair, ratchet-neutral by the
same construction, still open) and **#1206**, which **landed 2026-09-04**
(`1780670`, `schema` **+16**) — `src-element` clause 2.2's prose-only eight
attributes on a local `<element ref=>`, and the demonstration that this family's
fixtures do exist where the grammar pair's do not. **#1206's own landing then
carved two more**, from a UTF-16LE corner of the suite no UTF-8 grep had
reached: **#1215** (`src-element` clause 4, `ed-with-ns`) and **#1216**
(`src-attribute` clause 6, `att-with-ns`). **Both LANDED 2026-09-04** —
`976bdc2` for `schema` **+4** and `aeaf59e` for **+5**, each banking one and two
cases past its own prediction — and **each carved a successor from its own
boundary**: #1216's two `GAP(xsd)` markers are now **#1242** (clause 6.1's one
reachable failure, a local `<attribute ref=>`) and **#1243** (an
`<attribute use="prohibited">` escaping clause 6 entirely), both open and both
M4. Live producer-decides-and-accepts members are those two plus **#931**
(occurrence attributes on a named `<group>`'s child compositor), **#929** and
**#455**. **The family has now carved a successor at every landing since #471**,
which is the pattern to expect from it and not tail growth. A
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
