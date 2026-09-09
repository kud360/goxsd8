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

## Status — 2026-09-09 (`/backlog`, the SECOND today. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a `git fetch --unshallow` plus a `git ls-remote --heads`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. **The window is ONE commit and ZERO landings** — `aeb482f`, #413's post-land pass — so all twelve of the last band's rows survive and this is a re-cut, not a fresh ordering. **Every lane is unchanged, cell for cell**, which is what a zero-landing window looks like. **The namespace carries TWO held claims, and the last stamp shows neither**: `wip/issue-1167` is now LIVE (it was EXPIRED and was taken) and `wip/issue-1328` is CLAIMED with an active grounding — band rows 1 and 10 of the last band, both off-limits. **The marker census holds at 68 and the ZERO-untracked streak ENDS at ten**: three markers say *"No issue owns"* in prose `gapaudit` cannot read, and #1378 and #1379 now own them. The **TWENTIETH persona consultation**: **nine findings — three filed (#1380, #1381, #1382), four deduped into the issues that already own them (#1283, #1005, #1369, #1286), two dismissed on measurement with the residue routed (#1293, #1318)**)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13968 | 1430 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### A zero-landing window, and what a daily cadence against one looks like

**Nothing landed.** The window since the last stamp is a single commit —
`aeb482f`, #413's post-land pass, which filed #1374, #1375 and #1376. Every lane
figure above is identical to the last stamp's, and **all twelve band rows are
still open**.

This is the first stamp in this record with nothing to score, and it is worth
naming why rather than treating it as a gap in the evidence. **Two develop
sessions are in flight right now**, both started since the last stamp: one holds
`wip/issue-1328` (band row 1) and one holds `wip/issue-1167` (band row 10). The
band did not fail to be consumed; it is being consumed, and the two sessions
doing it had not finished when this pass ran. **The previous stamp ran at
roughly the same hour and measured fourteen landings in two days**, so the
honest reading is that a daily `/backlog` sometimes lands inside a window rather
than after it, and produces a stamp about work in progress.

**The one methodological consequence, and it is real.** With no landing to score,
this pass has no ratchet prediction to check and no banked-versus-predicted row
to add. The last two stamps' headline findings were both about prediction
defects (#1205's, then #931's); this window supplies no third data point either
way. **#1332 stays banded on the strength of the existing evidence**, not on new
evidence, and this stamp does not claim the tax is compounding — one sighting is
still one sighting.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a `git fetch --unshallow origin` and
this pass's 768-issue pre-write feed, with #1328's comments supplied on the
second pass to date its lease:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
732    wip/issue-732   404h53m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   582h51m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   355h12m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   548h52m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   main's     RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   main's     RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   360h33m0s  RETIRED  wip/issue-993: issue #993 is closed
1167   wip/issue-1167  1h44m0s    LIVE     wip/issue-1167: tip pushed 1h44m0s ago, within the 2h0m0s claim TTL
1328   wip/issue-1328  main's     CLAIMED  wip/issue-1328: no commits of its own and no RESUME:/TAKEOVER: comment ever posted -- a claim is born undated, so this is not a lapsed lease; settle it from the issue thread
```

**TWO claims are held and the last stamp shows neither. Re-run `wipsurvey`
before starting anything.**

**`wip/issue-1167` is LIVE**, tip `f3d2de4` (*"heartbeat — taking over expired
lease"*) at 12:29 UTC, 1h44m before this survey. The last stamp reported it
EXPIRED and invited a takeover; the takeover happened. **It is band row 10 of the
last band and is not banded here.**

**`wip/issue-1328` is CLAIMED, undated by the tool, and ACTIVE in substance —
settled from the thread, exactly as the tool's own reason line instructs.** Its
tip is `main` (`aeb482f`) with no commits of its own, so `wipsurvey` cannot date
the lease and correctly refuses to retire it on age. The thread settles it: a
`GROUNDING:` comment at 13:16 UTC and a body edit at 13:18 UTC, **55 minutes
before this survey**, the latter restating Acceptance item 2 on the grounding's
ruling. A session is mid-grounding. **Not relabelled `needs-replan`** — that
label retires an issue, and an active claim is the opposite of stale.

**The same seven RETIRED refs, unchanged row for row — SEVEN stamps now — and
this pass re-verified the supersede chain rather than inheriting the claim.** All
seven closed `not_planned`, and each names its replacement on its own thread:
**#732**→#1001/#1002, **#822**→#851, **#846**→#1029/#1030, **#872**→#878,
**#993**→#1018. Six of those replacements have since landed; **#1002** is the one
still open and is properly `blocked` on the repo owner's ruling. **No supersede
is owed and none is missing.**

**`wip/issue-933` and `wip/issue-968` now read `main's` where the last stamp read
466h and 405h.** That is the unshallow, not a change of state: both tips are
ancestors of `origin/main`, which a shallow checkout could not see. Their
verdicts are unchanged.

**Zero `needs-replan`. Zero `parked/*`. Zero `meta/*`.** **FIVE non-`wip`
`claude/*` refs stand**, unchanged in tip from the last stamp — `-39rk64`,
`-3xu0ki`, `-8jq9o6`, `-adewly`, and `-kk1f7v` (which is `origin/main`'s
predecessor `1ba9b40`). Listed for human triage, not acted on. `wipsurvey` reads
`wip/*` and `parked/*` only, so these are found by `git ls-remote --heads` and
never by the survey.

### Marker census — the ZERO-untracked streak ends at ten, and it was an artifact

`go tool gapaudit` over this pass's whole 768-issue pre-write feed: **68 markers
across 8 areas** — `xsd` 32, `validate` 17, `xpath` 6, `xml` 4, `parser` 3,
`value` 3, `conformance` 2, `cmd` 1. **The total and the per-area split are
identical to the last stamp**, which a zero-landing window guarantees.

**Group 1 moved 16 → 18 and group 2 23 → 26.** Both new group-1 rows are
`dead end: cites CLOSED #413`, at `xsd/complexextension.go:433` and
`xsd/contentrestricts.go:612` — exactly what #413's post-land pass predicted
when it filed **#1376** to repoint them, and **#1376 is band row 6 below**.

**Every group-1 row carries annotations, so the tool's mechanical filing rule
selects nothing — and applying that rule is what produced ten consecutive stamps
of "ZERO untracked GAP sites".** `gapaudit`'s own header says filing is *"a
judgment on the annotations, not on their absence"*, and this pass took that
judgment. `grep` for the class returns **exactly three sites that declare
themselves unowned in prose the tool cannot read**:

- `xsd/contentrestricts.go:694` — *"No issue owns that retirement"*
- `xsd/defaultbinding.go:355` — *"No issue owns this residual"*
- `xsd/defaultbinding.go:549` — the same, *"exactly as at
  fixedValueConstraintSubsumes' clause 4.2 twin"*

Every `candidate owner` annotation on those rows is a bare `file-matches` on the
containing file, and each was checked against the queue: **none owns the gap.**
Two issues now do.

- **#1378** owns the retirement of `contentTypeRestricts`' `maxContentPositions`
  ceiling. It is the twin of **#499** and carries **strictly stronger evidence**:
  #499's ceiling is inert on the W3C suite, this one is **REACHED** — the check
  fires 1538 times, 1532 models sit at ≤ 2970 positions and **six exceed the
  ceiling** (30001, 999999, 9999999 ×3, one past 16777216). Six declined
  verdicts standing today. **#578 is the neighbour, not the owner** — it owns
  construction cost at the same boundary — and both threads now say so.
- **#1379** owns the two `loc-testSubP` fixed-value residuals at
  `xsd/defaultbinding.go:345` (clause 4.2) and `:541` (clause 5.2), the `xsd`-side
  twins of **#774**'s `validate`-side declines. Different package, different
  rules, different phase; #774's ruling that the `xs:anySimpleType` case is not a
  gap transfers, and its thread now says so.

**The lesson is about the instrument, not the tree.** A clean census has never
meant a clean tree — the last stamp said so, with #1369 as its witness. What this
pass adds is narrower and more actionable: **a clean census does not even mean
every MARKED site has an owner**, because `gapaudit` joins on citations and a
marker can say "nobody owns me" in a sentence. **#1156** is the nearest
instrument and is about a different blindness (the `paragraph()` boundary); the
prose-declaration blindness has no owner and is not filed as one, because two
greps over a 68-marker corpus is not yet a tool.

**Citation drift, measured in a second file family and deliberately NOT
repaired.** #413's landing moved `xsd/contentrestricts.go` by +72/−18, and nine
open bodies carry `file:line` into the two files it touched. **Two citations now
point at unrelated text**: #1156's `contentrestricts.go:742` (its central claim's
coordinate, now `:838`) and #1156's and #345's `:1047` (now `:1097`). Both claims
are still TRUE at `aeb482f`, verified against this pass's own `gapaudit` output;
only the coordinates are false. **This pass re-anchored neither body.** #1356
owns the ruling on whether a planning pass may re-anchor citations, its body
records that two post-land passes have already done so as precedent with no
document asking them to, and a third would settle by habit the question #1356
exists to settle. The measurement went onto **#1356** as evidence instead, with a
pointer comment on #1156 so its taker is not misled — and it makes one of #1356's
arms cheaper, since a `citecheck` can detect both rows mechanically without
judging whether a premise survived. **#1356 is band row 10 on that evidence.**

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 14 pages (13 full at 100, page 14 at 82), PRs filtered locally:
**1382 rows, 609 PRs excluded, 773 issues — 290 open, 483 closed.** Coverage
verified rather than assumed: 1382 distinct numbers, min 1, max 1382, no gaps.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **52** | **124** | active |
| **M5 — Instance validation (XML)** | **15** | **24** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **273 `ready`, 15 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 290 with no gap, and **every open issue carries exactly
one, verified over all 290 by grouping each issue's own label set rather than by
summing two counts**. **#779** owns the mechanical check and carries this as its
fifth class.

By kind: `kind/refactor` 80, `kind/process` 56, `kind/gap` 53, `kind/tooling`
39, `kind/bug` 31, `kind/story` 25, `kind/docs` 21, `kind/feature` 4. By area:
`parser` 80, `meta` 78, `xsd` 63, `docs` 31, `conformance` 30, `validate` 25,
`cmd` 21, `value` 14, `builtin` 11, `xpath` 6, `xsderr` 6, `model` 4, `loader`
2, `regex` 2, `cli` 1.

**Both active milestones grew and neither closed anything**, which is what a
zero-landing window plus a filing pass produces: **M4 51 → 52 open** (#1380) and
**M5 14 → 15** (#1382), closed counts unchanged at 124 and 24. `ready` moved
**268 → 273**: five filed by this pass (#1378, #1379, #1380, #1381, #1382) and
nothing retired. That is #347's shape — `ready` is an output, not a target — and
neither milestone's count is its lane's remaining work; the M4 and M5 scope
sections below say so in their own words.

**`ready` 273 is the honest startable count with TWO to subtract**: #1328 and
#1167 both carry live claims, so **271** are startable without colliding. This is
the first stamp with two held at once, and neither is an expired lease that could
be taken.

**The unblock sweep measured a clean zero for the TWELFTH consecutive stamp**,
measured at the source over all **15** open `blocked` bodies rather than
inherited from #413's post-land pass. No `## Depends on` names #413 — the
window's only closure. Every named issue dependency in the partition — **#1324**,
**#407**, **#250**, **#831**, **#591**, **#248** — was checked individually and
is **open**. **Zero relabelled.** The set is 15, up one from 14: **#1374** joined
it and nothing left. Re-read the partition below rather than re-deriving it:

- **FIVE are triggers rather than issues** and say so in their own
  `## Depends on` — **#1374**, **#1224**, **#555**, **#1002** and **#1042**.
  **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **EIGHT still have at least one OPEN named dependency** — **#248**, **#267**
  and **#345** (all on #250), **#415** (#407), **#593** (#591), **#717**
  (#248), **#871** (#831) and **#1344** (#1324). **#1344's is band row 12**,
  which is the one place in this partition the band can reach — and it
  discharges in EITHER direction, #1324 closing `not_planned` being a recorded
  ruling just as much as #1324 landing is.

### Persona consultations — the TWENTIETH ran, and the named trigger did NOT fire

The cartographer role-plays no persona and does not spawn one (#416): it has
read the source, so a verdict it produced would launder an insider's opinion as
an outsider's. **The orchestrating session ran both personas against the
published surface — README, `go doc`, and a CLI built from the tree — and handed
this pass their reports; what follows is the folding, not the consultation.**

**The trigger the last stamp named did NOT fire, and the consultation ran
anyway.** The trigger was *"whatever next changes the observable surface"*, with
band rows 1, 4, 5, 6, 7, 9 and 12 listed as the ones that would. **Nothing
landed, so nothing changed the surface** — this is the first consultation in six
windows to run against an unchanged surface. **It is also the most productive of
the six by count**, which is the finding: nine findings, three of them defects
nobody had reported, against a surface two prior consultations had already read.
**A consultation is not exhausted by a previous consultation**, and scheduling
them on surface change alone would have skipped this one.

**Nine findings: THREE filed, FOUR deduped into the issues that already own
them, TWO dismissed on measurement. SEVEN of the nine were reproduced or
verified by this pass rather than taken on the reports' word, and THREE of those
seven corrected their report's mechanism while confirming its outcome.**

**Filed.**

- **#1380** (cliuser 1, `kind/bug`, M4) — a schema whose only top-level content
  is `xs:bogusElement` compiles to `components: 0` and **exits 0**, while the
  same fault one level deeper is charged correctly. Reproduced against a binary
  built at `aeb482f`. **The report's mechanism was corrected**: it read the
  construct as silently dropped with no detection anywhere, when in fact
  #1029/#1030's coverage census DOES report it as `UnmappedNoDispatch` and
  publishes it on `parser.AssembledDocument.Unmapped` — the gap is between
  detection and both of its possible consumers, since `run`'s dispatch skips the
  child and `grep -rn "Unmapped" cmd/` is **empty**. It is the XSD-namespace
  ELEMENT twin of #1369's attribute axis and of **#1036**'s foreign-namespace
  arm, and unlike #1369 its population **is** censusable by `suiteindex`.
- **#1381** (cliuser 3, `kind/bug`, `area/cmd`+`area/docs`) — `parse`'s `types:`
  and `components:` counts exclude anonymous type definitions, so two schemas
  describing the identical content model report **1 vs 2**. Reproduced.
  **The body adds the banding fact the report did not have**: the `-help`
  contract knows how to say "top-level" and does, one clause later, on the
  `model groups` line — so the enumeration is internally inconsistent about its
  own scope, which is what makes "disclose it" the natural arm rather than an
  arbitrary choice.
- **#1382** (libuser 2, `kind/story`, M5) — `validate.New`'s *"backend MUST be
  the backend schema was compiled with… Nothing here can detect that"* is the one
  representable illegal state on the library's front path, and its failure is
  silent: a wrong verdict on a valid document. **Verified rather than assumed** —
  `go doc -all ./xsd` matches nothing on `Backend` or `ValueSpace`, so the doc's
  reason is exact and the invariant has nowhere to live. Three routes; the body
  argues route (c) deserves serious weight because `New`'s own doc already
  reasons this way about the nil-backend default.

**Deduped, with the sighting and what it adds recorded on the thread.**

- **#1283** (libuser 1, its worst finding) — **second independent sighting, from
  the opposite direction.** This body argues from inside (`compileSet` copies a
  wrapper as a string); libuser argues from outside, having seen only README and
  `go doc`, and experienced it not as a roadmap gap but as a **self-contradiction
  between two published documents**: `parser`'s "Composition gaps" says no
  multi-root entry point is exported, and README's CLI section documents the
  capability shipping. **#1286 item 3 owns half of it; landing #1283 retires that
  item, landing #1286 alone does not retire #1283.** Nothing filed for an interim
  documented workaround — that would be debt with a known expiry — and the
  comment says to revisit if #1283 goes untaken.
- **#1005** (libuser 3) — **THIRD sighting**, and it brings one new fact: a
  consumer tried **`ParseReport.Unfollowed()` as the accumulation API** and it is
  not one. That cuts both ways on this issue's (a)/(b)/(c) ruling and the comment
  states both directions; the immediately actionable part is that
  `Unfollowed()`'s own doc does not say what it is not, which is one clause under
  any of the three answers.
- **#1369** (cliuser 2) — **second independent sighting, reproduced.** No body
  edit: the report offered nested and top-level as separate observations, and
  this body's own reproduction block already carries all three positions.
  The contrast worth carrying into the landing is cliuser's: **required-attribute
  enforcement works while unknown-attribute rejection does not**, so the s4s
  grammar is consulted for one half of the attribute rule and not the other.
- **#1286** (libuser 5) — a **sixth site** for its enumeration: `codegen/doc.go`
  and `codec/doc.go` both say *"The package declares no exported surface today"*
  and `builtin/native/doc.go`, with zero exports too, never uses that framing.
  **Offered as a comment and deliberately NOT written into `## Acceptance`** —
  #912 owns the finding that a body scoping work by enumerating sites goes stale,
  and this body's acceptance is literally *"Each of the five reads true"*. The
  taker adopts or rejects it explicitly.

**Dismissed on measurement, with the residue routed rather than dropped.**

- **libuser 4 — README overstates `go doc`'s sufficiency. FALSE, and checked
  exhaustively.** README **states the limitation itself** at `:336-338` and then
  **lists every Example by file and function name**; this pass verified **all
  eleven exist, in the file named**. The report's empirical step — "zero Example
  funcs printed" — is what a correct `go doc` run looks like, not a missing file.
  The residue is real and belongs to **#1293**, which owns `xsd/doc.go` pointing
  at Examples `go doc` will not show: **the disclaimer README pairs with its
  pointer is exactly what that pointer lacks**, one clause, already shipping in
  README and available to copy. A twelfth Example the README list omits
  (`value/backendtest`'s `ExampleRun`) went to #1286 as a ride-along.
- **cliuser 4 — unknown-flag errors fall back to raw Go boilerplate. FALSE in
  mechanism.** Reproduced: the line carries the documented `goxsd8: <subcommand>: `
  exit-2 prefix and the hand-crafted actionable follow-up every other exit-2 path
  writes, so the claimed inconsistency with the rest of the CLI's error style
  does not exist. The narrow residue that survives — **neither line names the
  flags that ARE valid for the subcommand**, and `-help` prints the whole
  multi-subcommand contract — went to **#1318** as a ride-along, since that issue
  already owns the class *"a `flag`-package behaviour the contract does not
  describe"*. The comment says the landing must adopt or reject it out loud,
  because that comment is the only record it was looked at.

**#1033 remains the one row no persona has ever looked at.**

### Working band

**Re-cut, not re-derived.** Nothing landed, so all twelve of the last band's rows
survive; what changed is that **rows 1 and 10 are now held** (#1328 CLAIMED and
active, #1167 LIVE), this pass filed five issues, and two rows moved on new
evidence. **Re-run `wipsurvey` before starting anything.**

| # | issue | why here |
|---|---|---|
| 1 | #1369 | **The best available M4 lane slice now that #1328 is held, and the consultation confirmed it independently.** Direction is **guaranteed by construction** — `xs:openAttrs` admits `##other` only, so an undeclared no-namespace attribute is an s4s fault, and every shape it changes is an acceptance today, so the lane moves UP or flat. Now carries **two sightings**: the persona that found it and the cliuser that reproduced `minOccur="2"` compiling to exit 0 without knowing it was filed. Its two preconditions stand — census the attribute names **directly** (`suiteindex` censuses by construct and cannot reach this axis, so do it the way #456 did), and the roster must be **GENERATED** from Appendix A, never hand-typed (PRINCIPLES 26) |
| 2 | #1380 | **The element-axis twin of row 1, filed by this pass, and it beats row 1 on the one axis the last two stamps made decisive: its population is CENSUSABLE.** `suiteindex` censuses by construct — namespace URI plus local name — which is exactly what an unrecognized top-level `xs:schema` child is, so the enumerated candidate set the last stamp asked for is one command away. Direction guaranteed the same way. It is also the smaller session: **the detection already landed** with #1029/#1030's census, and what is missing is the charge plus a ruling on whether the CLI should report the census at all. Banded below row 1 only because row 1's surface is the wider one |
| 3 | #1361 | **A correctness defect #1349's landing exposed rather than introduced, in the code that landing rewrote.** The name-keyed component memos hand back the ORIGINAL for a redefining `xs:group`/`xs:complexType` whose expanded name is also contributed, so `sch-props-correct` clause 2 cites one location twice and `src-redefine` 6.2.2 goes vacuous. Freshest evidence in the queue, and its centre of gravity (`parser/redefine.go`, `xsd/ownedtypefold.go`) is the furthest of the three parser slices from the held branch |
| 4 | #1332 | **The window's own measured cost, doc-only, one session.** #931's band row instructed a `suiteindex` census in as many words; nobody re-read it, an eighteen-day-old hand probe reached the verdict intact, and the lane moved +6 where the body named 4. **Held at row 4 rather than promoted**: this window landed nothing, so it supplies no second sighting, and the process clause bands on friction that compounds across sessions rather than on friction recorded once |
| 5 | #1359 | **A check that reports `ok` while its cases never execute.** `TestCheckLandingRealHistory` SKIPS in a fresh container: #813's fixture pair sits on a deleted branch, `requireFixtures` discards the two REACHABLE cases with it, and `go test ./tools/landcheck/` returns `ok`. `landcheck` mechanizes WORKFLOW's landing preconditions, so the thing not running is the thing that guards landings. Three arms, argued in the body, and the issue owes the choice |
| 6 | #1376 | **The only row whose cost is visible in THIS stamp's own tool output.** Both `{open content}` markers cite CLOSED #413; `gapaudit` group 1 grew 16 → 18 and both new rows are `dead end`. STYLE P3's named failure, one small session, and #413's post-land pass already sequenced it — *"#1376 before #1375 if only one is taken"* — because it clears a measurement every future `/backlog` would otherwise re-report |
| 7 | #1354 | **Take this before rows 8 and 9 — it rules on the MEMBERSHIP those two describe.** `violationLine`'s doc enumerates two members of the no-rule-ID branch and a third prints through it: `parse.go:737`'s non-`ErrNotFound` resolver fault, reachable through the CLI's own `loader.Dir`. Doc-only, and it is the cheapest of the three |
| 8 | #1367 | **A false sentence in the copy `README.md` itself calls authoritative, about the one thing an operator scripts against.** All three copies promise the bare message; both members print `parser: `, an internal package name where every other stderr line writes the program name and subcommand — and this pass saw it again incidentally while reproducing #1380 (`parser: unexpected model group child`). **Its wording is constrained by row 7's ruling**, so take them in that order or together |
| 9 | #1368 | **The `validate`-side half of rows 7 and 8, and the one that puts an operator in front of a filename that does not exist.** A `-schema` document whose root is not `xs:schema` is charged `[src-include]` against `goxsd8-schema-set.xsd:2:1`; the same input through `parse` is bare and rule-free. `readSchemaDoc` reads only `targetNamespace` and never the root name. Distinct from **#1224**, which owns the hint path and the opposite behaviour |
| 10 | #1356 | **Promoted from row 11 on this pass's own evidence.** Its 55 measured bodies were all `cmd/`; it now has a second file family, with `contentrestricts.go:742` and `:1047` pointing at unrelated text after #413's landing — and the failure mode its body predicts is observed with a name, since a reader who follows one **cannot tell whether the premise died or the line moved**. One arm is now demonstrably separable: a `citecheck` detects both rows mechanically without judging premises, so the tooling arm need not wait on the doctrine arm |
| 11 | #1283 | **Promoted into the band on a second independent sighting, from the audience it is written for.** libuser ranked it the worst finding on the API surface, reaching from outside what this body argues from inside. It is the largest row here and needs a warden pre-flight, but it **retires #1286 item 3, unblocks #1051's unexport, and deletes the wrapper-as-a-string copy** that #1224, #1250 and #1346 all work around. Read #1251's and #1260's two signature facts in the body before designing the entry point — the report half and the per-root provenance are both load-bearing |
| 12 | #1324 | **The only row in this band that discharges a `blocked` entry, and it discharges in either direction.** It challenges the eighth `/retro`'s deliberate one-copy ruling on the foreground-gate rule, asking whether `CLAUDE.md`'s gate block — the arrival path #1279 governs — should carry a pointer. #1344 waits on the ruling and closes with it whichever way it goes. The body states both outcomes as checkable `grep` results, which is what makes it one session |

**Held and NOT banded**: **#1328** (CLAIMED, grounding posted 55 minutes before
this survey) and **#1167** (LIVE, 1h44m). Both were rows in the last band and
both are being worked now.

**One banding hazard, and it is new.** `wip/issue-1328` sits on
`parser/produce.go` and `parser/produce_complex.go`, and **all three of the
queue's best M4 slices touch `parser/produce.go`** — rows 1 and 2 partially, and
**#929 collides on both files**, which is why #929 stays below the band rather
than rising into #1328's vacancy. Rows 1, 2 and 3 should expect to rebase when
#1328 lands, and row 3 is deliberately the one furthest from it.

**Below the band, and why**: **#1375** is NOT banded despite #413's post-land pass
asking for it, and the reason is specific rather than a demotion — its own Notes
carry an **unchecked hypothesis** that its apparatus and #1374's are the SAME
construction (§3.4.4.3 clause 3.1 reaching §3.8.4.1.3's interleave operator), and
a grounding on **#743** that confirms it **retires #1375 outright**. Banding a
session onto an issue that one cheaper check may dissolve is the wrong order; the
grounding is the next act, not the implementation. **#1318** just absorbed
cliuser 4's residue and should ride rows 8 and 9, which share its three files.
**#1378** and **#1379** are this pass's own gap filings and are startable, but
unvetted by any second reader; #1378's route (b) is a one-session ruling and its
route (a) is a construction wanting an oracle first. **#929** is blocked in
practice by the held branch, not by a label. **#1381** is this pass's `cmd`
filing and rides rows 8/9/12 cheaply, sharing `cmd/goxsd8/doc.go`. **#1382**
wants a warden pre-flight and should be read with **#1283**, **#1315** and
**#1051**. **#1036** is #1380's foreign-namespace arm and may reasonably be taken
with it. **#1297**, **#1266** and **#1267** are the `suiteindex` follow-ups;
**#1276** is the derived check row 2 would exercise. **#1370**, **#1286**,
**#1371**, **#1301**, **#1307**, **#1310**, **#1346**, **#1333**, **#1342**,
**#1343**, **#1339**, **#1351**, **#1355**, **#1319**, **#1325**, **#1363**,
**#1336**, **#1314**, **#1315**, **#1250**, **#1257**, **#1291**, **#1292**,
**#1293**, **#1285**, **#1287**, **#1201**, **#1183**, **#1196**, **#1122**,
**#1232**, **#1233**, **#1234**, **#1235**, **#1236**, **#513**, **#1003** are
unmoved from the last stamp. **#1033** is still the only row no persona has ever
looked at.

**There is no `Increasing` steward ranking anywhere in the band, for the FIFTH
consecutive stamp.** A scan of all 290 open bodies for `Cost of delay` returns
exactly **three** — **#849**, **#848** and **#845** — and none reads
`Increasing`: #848 and #845 are *"stable debt"*, and #849's own body records that
its ranking is falsified. The `kind/refactor` cost-of-delay clause of the banding
rule therefore selects nothing again. **No `/retro` ran this window** — the last
was 2026-09-06 and the routine is weekly, so the next is Sunday 2026-09-13.

### Next planning action

1. **Judge `gapaudit`'s group 1 on its annotations, not on their count.** Ten
   consecutive stamps read "every row carries an annotation" as "zero untracked
   sites"; three markers said *"No issue owns"* in their own prose the whole
   time. **#1378** and **#1379** now own them. The general lesson is on the
   instrument: `gapaudit` joins on citations, so a marker that disclaims
   ownership in a sentence is invisible to it, and the two greps this pass ran
   (`No issue owns`, `owns this residual`) are the cheap check until someone
   decides that is a tool. **Do not file that tool yet** — two greps over a
   68-marker corpus is not yet PRINCIPLES 27's threshold.
2. **Prefer a CENSUSABLE population over an enumerated one, and an enumerated one
   over a named case.** The last stamp established the second half; row 2
   (**#1380**) sharpens it. #1369 and #1380 are the same species of defect with
   the same guaranteed direction, and the only thing separating them is that
   `suiteindex` can census one axis and not the other. **Row 2's band row says to
   run that census first**; if it comes back empty, row 2 drops and says so.
3. **TWO claims are held and neither is takeable.** #1328 is CLAIMED with a
   grounding 55 minutes old; #1167 is LIVE at 1h44m. **Re-run `wipsurvey` before
   starting anything** — this is the first stamp with two concurrent sessions,
   and the previous stamp is stale about both.
4. **The persona trigger did not fire and the consultation was the most
   productive in six windows.** The naming-a-trigger scheme has worked five
   windows running, but it would have skipped this one: nothing changed the
   surface and nine findings came back, three of them defects. **Keep the trigger
   and add a floor**: consult when the surface changes **or** when two windows
   pass without one, whichever comes first. Band rows 1, 2, 7, 8, 9 and 11 change
   the surface; rows 3, 4, 5, 6, 10 and 12 do not.
5. **#849's steward re-ranking is owed for a FIFTH consecutive stamp and the
   condition is STILL not met.** The standing condition is *"file it at the next
   `/retro`, or at the next `/backlog` if the retro passes over it again"*. No
   `/retro` has run since 2026-09-06, so the retro has not had another turn and
   nothing is filed here. **Sunday 2026-09-13 is the deadline**: if that retro
   passes over #849 again, the next `/backlog` files the routing issue — the
   subject being that a routed re-ranking has no ledger and therefore never
   happens, which is #330's rule applied to the steward's own hand-offs.
6. **A zero-landing window is not a zero-information window, but it is a
   zero-PREDICTION window.** Two stamps running produced headline findings about
   ratchet predictions; this one has no landing to check, so **#1332 is banded on
   the old evidence and this stamp claims no trend**. The next stamp with a
   landing should re-read #1332's standing before deciding whether the tax
   compounds.
7. **The human decision blocking #1002 is unchanged and is now carried for a
   SIXTEENTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — and (b) depends on **#1042**, filed and `blocked`.
   **No agent should attempt either.**
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
inherits five**: the fold-the-five-species question (#635, #912, #609, #510,
#646), the `[tests that cannot fail]` **pattern** (routed by #472's post-land,
with no filed carrier), **#849's owed steward re-ranking**, which item 5 above
escalates to a dated deadline, the **band-versus-window throughput** question the
last stamp raised, and now the **persona-trigger floor** item 4 proposes. The
`gh --paginate` page-2 403 trap stays a `/retro` datum and not a filing. The CTA
cohort's 45 banked `instance` failures remain unattributed. `gate.yml` runs and
is still not a required status check, which only the repository owner can change.

**Environment, one witness each.** **`main` did NOT move under this pass** —
`origin/main` was `aeb482f` at the session brief and at every measurement, which
is the opposite of the last stamp's experience and is worth recording as the
control case. **The checkout was SHALLOW and was unshallowed before any range
reading** (`git fetch --unshallow origin`, #802); two `wipsurvey` rows changed
verdict text as a direct result. Repository-scoped REST served every read and
every write: **twelve writes across twelve distinct issues** — **five filed**
(#1378, #1379, #1380, #1381, #1382) and **seven thread comments** (#499, #578,
#774 for the gap families; #1356 and #1156 for the citation drift; then #1283,
#1005, #1369, #1286, #1293, #1318 for the consultation) — plus this section's own
replacement. **No issue body was PATCHed**, and the one place a body edit was
arguable — #1156's and #345's drifted coordinates — was deliberately left to
#1356's ruling, with the reason recorded on both threads. **The paginate recipe
was run FOUR times and needed retries on THREE of them**: the pre-write feed came
back short on pages 5 and 7, and the post-write feed short on **six** pages at
once (one of which took five retries), which is the #1145 transient that returns
valid JSON and exit 0. Every short page was re-fetched to 100 before any reshape,
and final coverage was verified by distinct-number count rather than by the
fullness check alone. **Five probes were run against a binary built from the
tree** — #1380's top-level and nested contrast, #1369's two attribute shapes,
#1381's anonymous-versus-named counts, cliuser 4's unknown-flag rendering, and
README's eleven named Examples checked one by one against the tree — and **three
of the five corrected their report in mechanism while confirming or refuting it
in outcome**. **No conformance measurement was taken by this pass**: the lane
table above is the committed expectations, which `docs/WORKFLOW.md` names as the
lane score (#1120).

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
