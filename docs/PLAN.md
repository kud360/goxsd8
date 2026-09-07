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

## Status — 2026-09-07 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **FIVE** landings — the last band's rows **1 through 5**, in order, with nothing inserted and nothing taken from outside: **#1282, #1270, #1275, #1246, #1260**. **Every lane is FLAT and every one of the five predicted exactly that**, which is the first clean five-for-five in this record and the direct answer to the last stamp's #1205 embarrassment. **The finding is the other half of that sentence**: three of the five were M4 lane slices banded on lane movement, and all three censused a suite population of ZERO before implementing — so the band spent a full window ordering on a criterion that returned nothing, and the next band is re-ordered onto slices with a *named* failing case (#455 → #456) or a direction guaranteed by construction (#725). The namespace has **ONE LIVE claim**, ending three idle stamps: `wip/issue-1007` is in flight right now. The marker census moved **69 → 68**, and the falsehood the last stamp banded for deletion is **gone from the tree**. The **EIGHTEENTH persona consultation**: **five findings — four filed (#1312, #1313, #1314, #1315), one dismissed into the thread that owns it (#1292)**)

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

### Five band rows landed in order, five predictions were right, and three lane slices measured a population of zero

**Every lane is flat: `schema` 13942, `instance` 11029, `datatypes` 1161,
unchanged cell for cell from the last stamp.**

| landing | commit | predicted | banked | milestone |
|---|---|---|---|---|
| **#1282** | `2ed0f60` | unchanged, measured AND structural (`go list -deps ./conformance` names no `tools/`) | unchanged | none |
| **#1270** | `d0fd4c6` | unchanged, **by construct census before implementing** | unchanged | **M4** |
| **#1275** | `34c39be` | unchanged, **by two independent corpus scans** (232 occurrences, 81 fixtures) | unchanged | **M4** |
| **#1246** | `a363592` | unchanged, **by construct census before implementing** | unchanged | **M4** |
| **#1260** | `03b961e` | unchanged, measured AND structural | unchanged | none |

**This is the first window in this record where the band's own order was
followed exactly — rows 1 through 5, nothing inserted, nothing unbanded taken —
and where every ratchet prediction was made before implementing and then held.**
The last stamp's sharpest datum was the opposite: #1205 carried
*"ratchet-neutral by construction"* from before `suiteindex` existed and banked
`schema` +5. Four of these five censused by construct or ruled the movement
structurally; none guessed. **The prediction discipline is working and needs no
further attention.**

**What needs attention is that three of the five were lane slices and the lane
did not move.** #1270, #1275 and #1246 are M4, each banded here on the argument
that a producer that decides more is a lane mover — and each, censusing its own
construct before implementing, found the suite holds **no fixture exercising the
shape**: #1270's `use="prohibited"` arm across 22 sites, #1275's 232
`xs:alternative` occurrences with no out-of-model child, #1246's 16106
`element@type` occurrences with three fixtures reached and zero flips. **Every
one of those zeros is a correct, useful measurement and none of them is a
failure.** The failure would be to keep ordering the band on "the producer
decides more" while the census keeps answering zero.

**So the band below is re-ordered onto a different criterion for its lane
slices, and the criterion is stated rather than implied: prefer a slice whose
failing case is NAMED, or whose direction is guaranteed by construction, over
one whose population is a hypothesis.** Row 3 (**#455**) is the only thing
between the queue and **#456**, whose body names a confirmed false reject
(`Open/open013.v1`) — a suite case, by ID, failing today for a reason the fix
removes. Row 5 (**#725**) closes an under-rejection: `resolveReferences` never
walks `s.attributeGroups`, so the movement can only be **UP**, and any downward
movement is a defect in the landing rather than a surprise in the corpus. Row 4
(**#931**) is the s4s family's next live member and is banded BELOW both, on
exactly this reasoning — `docs/PLAN.md`'s own M4 scope section names #931, #929
and #455 as the live "producer decides and ACCEPTS" members, and after three
consecutive zeros the family's remaining members must census first and be
re-banded on what the census says.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and
this pass's 738-issue post-write feed:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
732    wip/issue-732   357h9m0s   RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   535h7m0s   RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   307h29m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   501h9m0s   RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   429h22m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   368h27m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   312h49m0s  RETIRED  wip/issue-993: issue #993 is closed
1007   wip/issue-1007  11m0s      LIVE     wip/issue-1007: tip pushed 11m0s ago, within the 2h0m0s claim TTL
```

**ONE LIVE claim, which ends three consecutive idle stamps.** `wip/issue-1007`
advanced under this pass — `a25ddb4` at the survey's first run, `3ef48dc`
eleven minutes before its last — so a `/develop` session is working **#1007**
right now. **#1007 and the five issues banded with it last stamp (#1231, #1123,
#1189, #1261, #1290) are therefore NOT startable and are not banded below**:
they all edit `cmd/goxsd8/doc.go`, `main.go`'s `usage` const and `README.md`
together, and a second session in those files collides with a live lease. **Take
the band from the top and re-run `wipsurvey` first anyway** — this stamp's own
survey changed verdict between two runs twenty minutes apart, which is the whole
reason the rule exists.

**The same seven RETIRED refs, unchanged row for row — five stamps now.** All
seven closed `not_planned`; they are parks and supersedes whose content is
*supposed* not to be in `main`, and none owes a supersede. Cloud containers
cannot delete remote refs, so these accumulate by design and are not a finding.
**Zero `parked/*`.**

**The last stamp's `meta/audit-2026-09-06` finding is DISCHARGED and there is no
second sighting.** That branch stood unmerged with no PR while two open issues
cited its `docs/ARCHITECTURE.md` corrections as landed. It landed as **`fd99b18`**
(*"meta: audit 2026-09-06 — architecture audit (steward)"*), verified an ancestor
of `origin/main` by this pass, and the branch is gone from `git ls-remote`.
**#1285's Notes now read true.** Zero `meta/*` refs stand.

**FOUR non-`wip` `claude/*` refs stand, unchanged in tip from the last stamp** —
`eloquent-cerf-39rk64` (`0abeab6`), `-3xu0ki` (`62d5143`), `-8jq9o6` (`7841e98`),
`-adewly` (`5d03049`). Listed for human triage, not acted on. `wipsurvey` reads
`wip/*` and `parked/*` only, so these and any `meta/*` are found by
`git ls-remote --heads` and never by the survey — which is how the audit branch
went unseen for a stamp.

### Marker census

`go tool gapaudit` over this pass's whole 738-issue post-write feed: **68
markers across 8 areas** — `xsd` 32, `validate` 17, `xpath` 6, `xml` 4,
`parser` 3, `value` 3, `conformance` 2, `cmd` 1.

**The census moved 69 → 68, and the one retirement is the landing paying its own
debt.** `xsd` went 33 → 32 while every other area held. **The falsehood the last
stamp banded for deletion is gone**: `parser/produce_complex.go:3125` ended *"No
open issue owns retiring it"* while #1270 owned it, and #1270's landing deleted
the marker rather than correcting it — which is the stronger outcome and the one
its Acceptance required.

**Group 1 moved 18 → 17 and group 2 27 → 26. ZERO group-1 rows carry no
annotation at all, so the tool's own filing rule selects nothing and there are
ZERO untracked GAP sites — NINTH consecutive stamp.** No `kind/gap` issue was
filed from the census this pass, and that is the census reporting a clean tree
rather than a pass skipping the step.

**Two group-2 trackers were reconciled against the tree by this pass, both
carrying premises the tree had falsified.**

- **#725's body was corrected on TWO false sentences.** Its Goal called the
  marker *"the only marker in the tree naming no owning issue"* and its
  Acceptance said *"`gapaudit`'s untracked group returns to empty"*. Neither is
  true: `xsd/redefinition.go:111` reads *"and #725 owns its retirement"*, so the
  marker is not in group 1 at all — it is in **group 2**, under #726 — and the
  untracked set has measured empty for nine stamps, so nothing #725 does can
  return it to empty and this marker never made it non-empty. The coordinate was
  also stale (`:105` → `:108`). The Acceptance now states what retiring the
  marker actually moves: the census by one, and group 2 by one if #726 closes
  with it.
- **#726 shrank to its real remainder and was RETITLED off a discharged
  premise.** Its Acceptance asked the taker to *"name #725 as the marker's
  owner"* because the marker *"still reads 'no issue owns its retirement yet'"*.
  It does not, and has not for some time. Only the DIRECTION defect is left —
  one bullet at `xsd/redefinition.go:130`-`:131` claiming FAIL-OPEN for a check
  that is fail-CLOSED on the derived side — and the title no longer advertises a
  marker-ownership half that is already paid.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 14 pages (13 full at 100, page 14 at 15), PRs filtered locally:
**1315 rows, 577 PRs excluded, 738 issues — 277 open, 461 closed.**

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **50** | **117** | active |
| **M5 — Instance validation (XML)** | **14** | **23** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **261 `ready`, 14 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 277 with no gap, and **every open issue carries exactly
one, verified over all 277 by grouping each issue's own label set rather than by
summing two counts**. That distinction is the last stamp's repair holding: a
check that adds `ready` and `blocked` and balances against the open total cannot
see an `epic` hiding inside `blocked`, which is exactly how the defect it
repaired was published as verified. **#779** owns the mechanical check and
carries this as its fifth class.

By kind: `kind/refactor` 79, `kind/gap` 52, `kind/process` 50, `kind/tooling`
39, `kind/bug` 30, `kind/story` 26, `kind/docs` 17, `kind/feature` 4. By area:
`parser` 78, `meta` 70, `xsd` 59, `docs` 35, `conformance` 32, `validate` 24,
`cmd` 23, `value` 14, `builtin` 10, `xpath` 6, `xsderr` 5, `model` 4, `loader`
2, `regex` 2, `cli` 1.

**M4 moved 114 → 117 closed and 52 → 50 open**: #1270, #1275 and #1246 closed,
and one was filed onto it (#1307, by #1246's post-land). **M5 is unchanged at 14
open and 23 closed** for the second consecutive stamp — correct, since every
landing this window was `parser`, `tools/` or `cmd`. Neither milestone's count
is its lane's remaining work; the M4 scope section below says so in its own
words and this stamp's zero-population finding is a second reason.

**`ready` 261 is the honest startable count with ONE to subtract**, and that is
new: #1007 carries a live claim, so 260 are startable without colliding. There
is no numeric cap on `ready` and its size is an output (#347); it grew by five
this window as the post-land passes' filings (#1297, #1300, #1301, #1304, #1307,
#1310) and this pass's four outpaced the five closures.

**The unblock sweep measured a clean zero for the TENTH consecutive stamp.**
All **14** open `blocked` bodies were read and their `## Depends on` sections
checked against this window's five closures. **Exactly one blocked body mentions
any of them — #1224, which names #1260 — and it is not a discharge**: #1260's
own post-land pass measured that trigger at the source and ruled it did not
fire (the trigger is a change to `compileSet`'s synthesized wrapper; what
changed was its signature, and `schemaSetSource` and `const schemaSetLocation`
are byte-identical across the landing). The measurement is on #1224's thread and
is not to be re-derived. **Zero relabelled.** The set is unchanged at 14 and
partitions exactly as the last stamp recorded — re-read it that way rather than
re-deriving it:

- **FOUR are triggers rather than issues** and say so in their own
  `## Depends on` — **#1224**, **#555**, **#1002** and **#1042**. **Do not
  re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **EIGHT still have at least one OPEN named dependency** — **#248**, **#267**
  and **#345** (all on #250), **#415** (#407), **#456** (#455), **#593**
  (#591), **#717** (#248) and **#871** (#831). **#456's is band row 3**, which
  is the one place in this partition where the band can discharge an entry.

### Persona consultations — the EIGHTEENTH ran, and the named trigger fired

The cartographer role-plays no persona and does not spawn one (#416): it has
read the source, so a verdict it produced would launder an insider's opinion as
an outsider's. **The orchestrating session ran both personas against the
published surface at `9f2419d` and handed this pass their reports; what follows
is the folding, not the consultation.**

**The trigger the last stamp named fired exactly as named.** It was *"whatever
next changes the observable surface"*, with band rows 5, 6, 8, 11 and 12 listed
as the ones that would. **Row 5 — #1260 — landed and fired it.** Naming the
trigger rather than scheduling a pass has now worked four windows running, and
this is the first where the firing landing is one the previous stamp predicted.

**Five findings: four filed, one dismissed, none folded, and the two most
substantive were REPRODUCED by this pass from a binary built out of the tree
rather than taken on the reports' word.**

- **cliuser, one finding.** **#1312** filed — the unresolved-directive note
  #1260 shipped is **dropped on every compile-failure path**. `reportUnfollowed`
  sits at `cmd/goxsd8/parse.go:96` and `validate.go:181`, *after* the error
  returns at `:86`-`:92` and `:170`-`:176`, so the line vanishes exactly when
  the assembly is both SHORT and REJECTED — including when the rejection is in
  an unrelated `-schema` document, since `compileSet` compiles the set as one
  assembly. The report is in hand and discarded: `AssemblyReport`'s own doc
  promises it is *"populated as far as assembly got even when ParseReport
  returns an error"*. **This pass reproduced it and corrected the report in the
  process**: cliuser had the note surviving as a DEBUG `slog` line under `-v`;
  it does not survive at any verbosity, because the line is a `cmd` rendering
  and not a log record, so the fact is unreachable to an operator by any flag.
  Everything else cliuser probed against #1260's published contract matched it
  exactly — the benign-skip note for all four directive kinds, `-q` not
  silencing it, the bare `xs:import` correctly unreported, the exit code not
  escalated, the hint case correctly silent — which is corroboration and not a
  finding.
- **libuser, four findings, three filed.** **#1313** — both CLI contract copies
  state a rejected schema's stderr line as `<loc>: [<rule>] <message>`
  unconditionally, and it is false for the whole no-rule-ID class
  `violationLine`'s own doc enumerates. **Sharpened past the report**: libuser
  cited `README.md:138`; the same claim is at `cmd/goxsd8/doc.go:16`-`:17`,
  which `README.md:163` names as the **authoritative** copy, so the falsehood is
  in the authority. Reproduced on an empty `xs:redefine`, which prints a bare
  sentence with the location mid-message and no `[rule]` at all. **#1314** —
  `UnfollowedDirective.At`'s doc names no identity for `At.URI` while
  `AssembledDocument.Location`'s says *"Compare locations through this field"*,
  and `At.URI` is the OTHER identity (the requested location, `el.Loc()` at
  `parser/parse.go:320`), so a consumer correlating `Unfollowed()` back to
  `Documents()` silently matches nothing under any canonicalizing resolver.
  **#1315** — `AssembledDocument` carries no target namespace and no provenance,
  and the only route to either is `Element.Attr`, which `parser/tree.go:26`-`:32`
  says is exported *"for a single consumer, this module's own conformance
  harness"* and *"may be unexported without a deprecation path"* — a sentence
  the module's own CLI already falsifies at `cmd/goxsd8/validate.go:463`.

**ONE finding dismissed, into the thread that owns the decision rather than in
this section, so the next consultation's report is triaged against a comment
instead of re-derived.**

- **`Unevaluated` has no formatted-string method → dismissed into #1292**
  (`issuecomment-5572062833`). The persona flagged it for judgment rather than
  asserting it, and it is the same missing seam #1292 already owns, seen from
  the rendering side instead of the decidedness side: the correct
  *not proven invalid* test is a three-way conjunction a consumer must know to
  write, and the rendering of one of those three is likewise re-derived per
  consumer, with `cmd/goxsd8` the copy that exists in both cases. A second issue
  would fragment the one ruling #1292 owes. **Reverses** if #1292 lands ruling
  decidedness and saying nothing about rendering.

**#1033 remains the one row no persona has ever looked at.**

### Working band

**Re-derived from this pass's evidence, and re-ordered on the zero-population
finding above.** Rows 1 through 5 of the last band all landed in order; **#1007
carries a live claim and is not here** — re-run `wipsurvey` before starting
anything.

| # | issue | why here |
|---|---|---|
| 1 | #1304 | **The same argument that put #1282 at row 1 last stamp, now at FIVE sightings across THREE CONSECUTIVE landings — and #1282's landing proved the argument rather than exhausting it.** `suiteindex` reports a hit's `Parent` and never its CHILDREN, so every content-model census is hand-written: #1270 hand-read 22 sites decoded (UTF-16), and #1275 and #1246 EACH had mason and the arbiter write two independent scans in one session, four throwaway walks in two days, none surviving for the next issue that needed it. The tool cannot express it at any number of queries — `hit` is `{File, Line, Col, Parent, Values}` and nothing in the query language reaches a descendant. Its own body warns that this is NOT #1282's "one stack read away": a child arrives after the record is closed, so the loop's shape changes. Pairs with **#1297** (a braceless name prints alike for "no namespace" and the XSD namespace) — same file, and #1304's Acceptance already defers the rendering to it |
| 2 | #1300 | **A gate step that silently did not run, at FOUR sightings across four weeks, two gate parts and three command shapes — and one of the four MASKED A REAL LINT FAILURE from a `&&` chain.** A gate command piped into `tail` exits with `tail`'s status, always 0, and nothing in the repo says so anywhere. `kind/process` is banded on the sessions it costs and this one costs the gate its meaning: a failing step that reports success is worse than a failing step. The body already carries the ruling it needs (three homes, a preferred one, and the reason a mechanical check would be false coverage), so it is one session. Sighting 2 is a post-land pass, so this is not a mason-only hazard |
| 3 | #455 | **The first lane slice in this band with a NAMED failing case behind it, and it is the only thing standing between the queue and that case.** One collapse-trim helper spelling §4.3.6's four whitespace characters once, replacing `strings.TrimSpace`'s Unicode class at ten sites — and its dependent **#456** names a confirmed false reject by suite ID (`Open/open013.v1`). That is a measured population of at least one, which is more than the last three lane slices had between them. It also discharges the only entry in the `blocked` partition the band can reach. #324 already landed the two-case equivalence proof its doc comment must carry |
| 4 | #931 | **The s4s family's next live member, banded BELOW two rows it would have outranked last stamp — deliberately.** A named `xs:group` definition whose CHILD compositor carries `minOccurs`/`maxOccurs` is accepted; `xs:simpleExplicitGroup` and `xs:namedGroup`'s `all` arm both declare them `use="prohibited"`, and `rejectProhibitedAttrs` switches on the declaration element and stops there. The spec work is done in the body, down to line numbers re-verified against the tree, and the fault is PRESENCE not value. **Census by construct FIRST, with #1282's `Parent` field, and re-band on what it says** — three consecutive members of this family measured zero, and a fourth zero is a reason to stop banding the family on lane grounds, not a surprise |
| 5 | #725 | **The one lane slice whose DIRECTION is guaranteed by construction: it closes an under-rejection, so the `schema` lane can only go up.** `resolveReferences` walks five of the schema's properties and never `s.attributeGroups`, so an `attribute ref=` or a local `type=` inside ANY top-level attribute group definition gets no `src-resolve` verdict — redefinition and original alike, and the cause is Phase A's coverage rather than §4.2.4 clause 4.1.2, verified independently by #503's warden and arbiter. Any downward movement means a resolvable reference is being refused. **Its body was corrected by this pass** on two false `gapaudit` sentences and a stale coordinate; read #726 before deleting the DIRECTION bullets it would delete |
| 6 | #1312 | **A bug in the feature that landed last window, found from outside by the persona whose consultation that landing triggered.** The unresolved-directive note is dropped on every compile-failure path — present when the schema compiles, absent the moment anything in the same compile unit fails, and unreachable at any verbosity. It is one statement order in two files, but the ruling is not free: reporting before the return means the message can no longer say *"the compiled schema is short of…"*, and `reportUnfollowed`'s own doc opens *"of a schema that DID compile"*. The body carries both options and picks neither. **Its ruling constrains row 7's wording**, so take them in this order or take them together |
| 7 | #1251 | **The hint-side twin of row 6, twelve lines away in the same function.** A hint naming a document that is not there costs an operator exit 1 with an empty stderr, and the plumbing — `AssemblyReport.Unfollowed()` — already records it. Row 6 is that the accessor is read on one path and not the other; this is that the hint-side entry is read on none. Same accessor, disjoint arms. Its three carried facts are worth reading before either is taken, and #1260's landing settled the line shape as prior art in this exact file |
| 8 | #1247 | **THREE consecutive landings have now hand-edited the same parenthetical, which turns the characterize-vs-enumerate argument into a measurement.** `checkS4SChildOrder`'s roster names five members of a class with at least nine; #1275 hand-added `src-ta`, #1246 hand-added clause 3, and #1270 needed no edit because its family is covered by characterization — the comparison that decides the ruling. **#1246 already put the universal half in the tree**, so what is left is the roster sentence between two correct sentences. Doc-only and provably so, fully re-stamped at `a363592` by two post-land passes, and it carries a second finding of its own (`produceSimpleContent`'s doc pairs `src-ct` clause 1 with the wrong two walks) |
| 9 | #1203 | **A latent ratchet ambush #404's landing created, and the queue still holds two issues that would trip it.** Two of #404's eleven banked `schema` passes — `addB014` and `schZ006` — reject for a fault the suite does not intend, so when #603 or #703 repairs the over-rejection those passes flip down as a spurious `Regressed` the repairing session did not cause. A banked pass resting on an over-rejection is invisible in the lane file. No production change — a measurement and a ruling, cross-referencing #1002 |
| 10 | #1313 | **A false sentence in the copy `README.md` itself calls authoritative, about the one thing an operator scripts against.** `cmd/goxsd8/doc.go:16`-`:17` and `README.md:138` promise `<loc>: [<rule>] <message>` for every rejection; the s4s-grammar class prints a bare message with no rule and no `<loc>:` prefix, which `violationLine`'s own doc has enumerated all along. Above the other doc rows because a contract copy that is WRONG outranks one that is merely incomplete, and because `TestUsageCoversContract`'s substring coupling (#398) cannot see a claim's truth — only its presence |
| 11 | #1167 | **Two measured sightings, #414 and #1115, and PRINCIPLES 27 says a repeated grep wants a tool.** `gapaudit` reconciles marker → issue and issue → marker; nothing audits prose that points *at* a marker or its file, and the stale-marker sweep has missed an inbound site in the deleting package twice. **Still exactly two sightings after this window**: #1270 deleted its marker cleanly and #725's stale sentences are a marker's own text and a body's, neither of which is prose pointing at a marker. The count is kept honest deliberately |
| 12 | #1227 | **The pen bound is written and nothing checks it — first sighting AFTER the rule existed, which is what makes it fileable.** `docs/WORKFLOW.md` binds who may hold the pen, but landing precondition 3 iterates over MASON commits, so a branch with none has nothing to iterate. #636 is the only other witness and is pre-rule by one day, so this is n=1 post-rule and the deliverable is a runnable check, not a fiftieth `kind/process` issue. Both landed precedents are named in its body |

**Below the band, and why**: **#1007's cluster is CLAIMED** — #1007, #1231,
#1123, #1189, #1261 and #1290 all edit the same three files and a live lease
holds them; re-band whatever survives that landing. **#1297** pairs with row 1
and should ride it; **#1266** and **#1267** are the other two `suiteindex`
follow-ups and neither gates row 1 — #1266 now owns a measured 232-vs-236 corpus
discrepancy handed to it by #1304's filing. **#1301** and **#1307** are the two
inline-copy refactors #1270 and #1246 carved, and they are the same shape in the
attribute and element producers: one session, one helper each, and #1307's own
body says `rejectBothInlineTypes` already ships the pattern. **#1310** pins the
two directive kinds #1260 published and did not test; it is two fixtures and two
table rows and should ride row 6 or row 7. **#1314** and **#1315** are this
pass's other two libuser filings — #1314 is a one-clause doc fix and pairs with
anything in `parser/report.go` (**#483**), while #1315 is a surface ruling that
must be read with **#1283** and **#1051** and wants a warden pre-flight for two
of its three options. **#1291** and **#1292** and **#1293** are the previous
consultation's filings and are unmoved: #1291 still sits behind **#486**'s
sequencing question, #1292 now carries this pass's dismissal comment, #1293
pairs with **#854**. **#1285**, **#1286** and **#1287** are the 2026-09-06
steward audit's findings, now that its branch has landed; **#1286** is five
`doc.go` contracts stating as shipped what does not ship and is the natural
partner for row 10, sharing a reader if not a line. **#726** is smaller than it
was — this pass discharged half of it — and pairs with row 5. **#1276** is the
derived check that makes #1050's Scope section unfalsifiable by drift and pairs
with whoever next opens `parser/census.go`. **#1250** is #1251's sibling on the
same code path. **#1201** retires the two `GAP(conformance)` markers when it
lands and pairs with row 9. **#1183** makes #1051's stage-2 deletion three
predicates smaller. **#1196** measures the live residual for #1051 and its own
body forbids banding on the numbers it produces. **#1135** pairs with row 4's
neighbourhood. **#1258** is still n=1 after a fourth non-reproduction. **#1257**
should be taken with #1292. **#768** and **#845** were re-scoped by earlier
passes and both carry two honest options. The remaining doc rows — **#1232**,
**#1233**, **#1234**, **#1235**, **#1236**, **#513**, **#1122**, **#1003** — are
one-sentence fixes with settled directions. **#1033** is still the only row no
persona has ever looked at.

**There is no `Increasing` steward ranking anywhere in the band, for the THIRD
consecutive stamp.** A scan of all 277 open bodies for `Cost of delay` returns
exactly **three** — **#849**, **#848** and **#845** — and none reads
`Increasing`: #848 and #845 are *"stable debt"*, and #849's own body records
that its ranking is falsified and that *"a steward re-ranking is warranted"*.
The `kind/refactor` cost-of-delay clause of the banding rule therefore selects
nothing again. **The 2026-09-06 architecture audit has now LANDED and it filed
two `kind/refactor` issues (#1285, #1287) with no `## Cost of delay` section on
either**, so the ranked backlog did not grow when the one instrument that can
grow it ran. That is the second audit in a row to do it.

### Next planning action

1. **Re-order lane slices on measured population, not on "the producer decides
   more".** Three consecutive M4 lane slices censused zero and banked zero. The
   census discipline is not the problem — it is working, and it is what caught
   all three before they claimed anything. The banding criterion is: prefer a
   slice with a NAMED failing case (row 3, #455 → #456's `Open/open013.v1`) or a
   direction guaranteed by construction (row 5, #725, an under-rejection closing
   UP-only) over one whose population is a hypothesis (row 4, #931). **#931 must
   census by construct before implementing and be re-banded on what the census
   says**; a fourth zero from this family is a reason to stop banding it on lane
   grounds, and the M4 scope section's list of live members (#931, #929, #455)
   should then be read as three candidates rather than three slices.
2. **Take #1304 before the next content-model census, and hold it to the
   sightings it was filed against.** #1282 landed and its thesis held; #1304 is
   the remaining axis of the same instrument, and the cost is now five hand
   walks across three consecutive landings, twice with two agents duplicating
   each other in one session. A `kind/tooling` row sits above every lane slice
   for the THIRD consecutive stamp, which is the banding rule applied as
   written: a tool that moves no lane will never be lifted on lane grounds.
3. **#1007 is claimed and its five banded siblings are unstartable with it.**
   This is the first LIVE claim in four stamps, and this pass's own survey
   changed verdict between two runs twenty minutes apart. **Re-run `wipsurvey`
   before starting anything**, and do not read this stamp's namespace block as
   current.
4. **The eighteenth persona consultation is folded and nothing from it is handed
   off.** Five findings: four filed (#1312, #1313, #1314, #1315), one dismissed
   into #1292 with the condition that reverses it. **The next consultation is
   owed against whatever next changes the observable surface** — name the
   trigger rather than scheduling a pass, which has now worked four windows
   running and correctly predicted the firing landing for the first time. Band
   rows 3, 5, 6, 7 and 10 change it; rows 1, 2, 4, 8, 9, 11 and 12 do not.
5. **`meta/audit-2026-09-06` LANDED and the last stamp's item 5 is discharged —
   there is no second sighting to file.** It merged as `fd99b18`, verified an
   ancestor of `origin/main`, and #1285's Notes read true. The general lesson
   stands and is worth keeping: `wipsurvey` reads `wip/*` and `parked/*` only, so
   a `meta/*` branch outliving its session is invisible to the survey and is
   found by `git ls-remote --heads` alone. **Run that command as part of the
   namespace step, every stamp.**
6. **#849's steward re-ranking is owed for a THIRD consecutive stamp, and the
   audit that owed it has now landed without producing one.** The last stamp
   ruled that *"a third leaves it a pattern, and the issue to file then is about
   the routing, not about NCName"*. This is that third. **File it at the next
   `/retro`, or at the next `/backlog` if the retro passes over it again** — the
   subject is that a routed re-ranking has no ledger and therefore never
   happens, which is #330's rule applied to the steward's own hand-offs.
7. **The human decision blocking #1002 is unchanged and is now carried for a
   FOURTEENTH stamp.** #1002 waits on a ruling between (a) a constitutional
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
times; the 2026-09-06 `/retro` rewrote a neighbouring bullet of that same
paragraph, so re-read it before grounding any of the four. **The next `/retro`
inherits three**: the fold-the-five-species question (#635, #912, #609, #510,
#646), the `[tests that cannot fail]` **pattern** (routed by #472's post-land,
with no filed carrier), and **#849's owed steward re-ranking**, which item 6
above now escalates. The `gh --paginate` page-2 403 trap stays a `/retro` datum
and not a filing. The CTA cohort's 45 banked `instance` failures remain
unattributed. `gate.yml` runs and is still not a required status check, which
only the repository owner can change.

**Environment, one witness each.** Repository-scoped REST served every read and
every write here: **8 writes across 6 distinct issues** — **4 filed** (#1312,
#1313, #1314, #1315), **2 body PATCHes** (#725, #726, the latter retitled in the
same call) and **2 thread comments** (#1292's dismissal, #1251's cross-reference
to #1312) — plus this section's own replacement. **The paginate recipe ran to 14
pages twice**, 13 full at 100 each time and the last page short at 11 then 15,
with the post-loop fullness check passing on both. `gh auth status` reports the
token invalid in this container and `gh api repos/kud360/goxsd8/...` served
everything anyway, which is **#1140**'s landed caveat behaving as documented.
**`main` did not move under this pass** — `9f2419d` at the first survey and at
the last. **Two probes were run against a binary built from the tree** rather
than reasoned about: #1312's suppression and #1313's bare s4s-grammar line, both
on fixtures written for the purpose and both matching the persona reports in
outcome while correcting one of them in mechanism. **No conformance measurement
was taken by this pass**: the lane table above is the committed expectations,
which `docs/WORKFLOW.md` names as the lane score (#1120).

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
ACCEPTS, so neither extends this family. Live producer-decides-and-accepts members are **#931**
(occurrence attributes on a named `<group>`'s child compositor), **#929** and
**#455**. **The family carved a successor at every landing from #471 through
#1243, and #1242 is the first that did not** — one reading is the tail reaching
zero, the other is that this pair was narrower than its predecessors; two more
landings decide it. **#1270 is the FIRST of those two and it came back empty**,
which is the tail reading; one more landing settles it, and until then neither
reading is the one to plan on. A
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
`a363592`, and open #1247) are about that record rather than about a fifth
guess.

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
