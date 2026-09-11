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

## Status — 2026-09-11 (`/backlog`, the twenty-second. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against `git fetch --unshallow origin` plus `git ls-remote --heads`, the marker census a fresh `gapaudit` run TWICE — before and after this pass's writes — and the milestone and queue counts a page-numbered `state=all` fetch taken **after** the writes. **The window is THREE landings and ONE PARK, and the park is the event**: #434, band row 2, was implemented in full and measured **35 regressed `schema` cases**, so it is closed `not_planned` and re-planned into four issues. **Every lane is UNCHANGED** — `schema` holds at 14012 — and that is correct rather than a stall: all three landings (#1391, #1354, #1367) ran `Ratchet: unchanged` by construction. **The namespace carries TWO held claims**, `wip/issue-1297` and `wip/issue-1332`, and **neither was in the last band**. **The marker census holds at 68; group 1 moved 16 → 18 BY THIS PASS'S OWN WRITE** — closing #434 made two markers dead ends, measured rather than predicted, and **#1426 owns repointing them**. The **TWENTY-SECOND persona consultation**: **ten findings — five filed (#1430–#1434), two deduped (#755, #1382), two dismissed against landed rulings (#672, #1031), one a null report**)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14012 | 1386 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### The window: three landings, one park, and every lane flat

**No lane moved, and the attribution is that nothing was offered.** All four
numbers are byte-identical to the last stamp's. The three landings are
**#1391** (`suiteindex` gains an unanchored attribute axis), **#1354** and
**#1367** (two halves of the no-rule-ID line's contract), and each ran
`Ratchet: unchanged` in write mode rather than inferring it. A tool and two
contract-copy corrections cannot move a lane, which is what banding them said
they would do.

**The park is the window's real content, and it cost more than the three
landings together.** #434 — **band row 2, described by the last stamp as *"the
best lane slice in the queue, and the only banded row that names the cases it
will flip"*** — was grounded by an oracle, pre-flighted by a warden,
body-corrected three times by the cartographer, implemented in full (34 files
`+674/−666`, `surface: unchanged`, gate parts 1-3 green), and **gate part 4
measured 35 regressed `schema` cases**. Every one of the six charge sites
regresses on its own, so **the only subset that lands green is the empty one**;
and the issue's single predicted win does not exist to re-scope toward —
`saxonMeta/Missing.testSet` carries `version="1.0"` on the testSet and on all six
testGroups, so the whole set is withheld as inapplicable and has no line in
`schema.txt`. `grep -i missing conformance/testdata/expectations/schema.txt`
returns nothing and always did.

**Three findings from that round outrank any conclusion about §5.3, and all three
are now filed.**

1. **The spec question underneath is not settled the way the grounding assumed.**
   §5.1's second bullet against §5.3, §4.2.3 and `src-resolve` clause 1 is
   addressed by neither the body nor the grounding, and
   `saxonMeta/Override.testSet:909`'s `over026` is `version="1.1"`,
   `status="accepted"`, `expected validity="invalid"`, written by the 1.1
   reference implementor, with its own fixture comment reading *"the reference to
   zonedDate is an error"* — scoped IN by the suite and contradicting a blanket
   §5.3 reading. **#1426** owns the ruling.
2. **None of the 23 withheld "improvements" is explained by §5.3.** Every one is
   explained by a component set this module does not supply — §3.2.7's four
   `xsi:` attribute declarations (**#1427**, 7 `schema` + 14 `instance`) and the
   well-known `xml:` namespace declarations (**#1428**, 2 `schema`). Banking them
   there would have written green over two genuine under-implementations
   (PRINCIPLES 22). **Neither depends on the ruling**, and both are band rows
   here.
3. **The harness's own #276 protection was broken by the attempt, not merely
   pinned by it.** `TestSchemaExecutorDeclines{Unresolved,ConfinementRefused}DirectiveTarget`
   decline because the root's reference fails `src-resolve` at finalize; with the
   charge gone the schema composes, the executor decides, and **both report PASS
   under `expectValid=false`** — for exactly the fabricated reason #276 exists to
   refuse. Five failures outside the implementation account's enumerated 34.
   **#1429** owns it, plus the decline-flip comparison the round left unmeasured.

**The prediction question gained a fourth data point and it is the worst one
yet.** #1369 predicted from a hand-rolled census and banked +17; #1380 predicted
movement and mason measured zero; **#434 predicted three named fixtures that are
not scored at all**, and the one-command check that settles it
(`grep -i missing …/schema.txt`) was first run in the implementation round, the
most expensive place available. `suiteindex` could not have answered it —
it censuses by construct and has no version-or-discovery axis. **#1332 is LIVE
against exactly this question** and is not banded here.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against `git fetch --unshallow origin` and
this pass's **800-issue post-write feed**:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
434    wip/issue-434   2h17m0s    RETIRED  wip/issue-434: issue #434 is closed
732    wip/issue-732   453h11m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   631h9m0s   RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   403h31m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   597h11m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   main's     RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   main's     RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   408h51m0s  RETIRED  wip/issue-993: issue #993 is closed
1297   wip/issue-1297  38m0s      LIVE     wip/issue-1297: tip pushed 38m0s ago, within the 2h0m0s claim TTL
1332   wip/issue-1332  1h34m0s    LIVE     wip/issue-1332: tip pushed 1h34m0s ago, within the 2h0m0s claim TTL
```

**`wip/issue-434` is the eighth RETIRED row and is PRESERVED DELIBERATELY.**
Pushed at `e86b5c36aac66dc1ed8f7fc0d7f092bedcbae8b8` on
`36accb28bf57fbc4b03809b9e0ccec4728ae2f84`, `NOT-ancestor` of `main` and
correctly so — **it is the measurement**, and #1426 names it as evidence not to
re-pay for. It was `RETIRED … labelled needs-replan` before this pass's write and
reads `issue #434 is closed` after; both verdicts are RETIRED and **neither
licenses deleting the ref**. Nothing was renamed, force-pushed or deleted.

**TWO claims are held, and NEITHER was in the last band.** `wip/issue-1297`
(tip `0ad4382f`, 38m) and `wip/issue-1332` (tip `a03bd9de`, 1h34m) are both LIVE
and both unbanded — #1297 has never been banded, #1332 was row 4 two stamps ago
and was dropped from the last band as held. **Re-run `wipsurvey` before starting
anything**, and note that a band is not where sessions are actually going.

**The seven older RETIRED refs are unchanged row for row — NINE stamps now — and
the content question was re-measured rather than inherited.**
`git merge-base --is-ancestor <tip> origin/main` over all seven: **`wip/issue-933`
and `wip/issue-968` ARE ancestors of `main`; the other five are NOT**, tips dated
2026-08-16 to 2026-08-25. **That is the expected shape, not a discrepancy** — all
seven closed `not_planned` and were superseded rather than merged. The supersede
chain (#732→#1001/#1002, #822→#851, #846→#1029/#1030, #872→#878, #993→#1018) is
inherited from the last three stamps' verification, with **#1002 re-confirmed
OPEN and properly `blocked`**. **No supersede is owed and none is missing.**

**Zero `needs-replan` for the first time in two stamps** (#434 was the only
member and is closed). **Zero `parked/*`. Zero `meta/*`.** **FIVE non-`wip`
`claude/*` refs stand**, unchanged in tip for a third stamp — `-39rk64`,
`-3xu0ki`, `-8jq9o6`, `-adewly`, `-kk1f7v`. `-kk1f7v` is `1ba9b40`, now **eleven
commits behind** `main` (`06add85`). Listed for human triage, not acted on;
`wipsurvey` reads `wip/*` and `parked/*` only, so these are found by
`git ls-remote --heads` and never by the survey.

### Marker census — 68 markers; group 1 moved 16 → 18 by this pass's own write

**`go tool gapaudit` was run TWICE, before and after this pass's writes**, and
the difference is the point.

- **Pre-write, against the 791-issue feed: 68 markers across 8 areas, group 1 at
  16** — identical to the last stamp, row for row. Nothing added, nothing
  deleted, which is what three landings that touched no marker look like.
- **Post-write, against the 800-issue feed: 68 markers, group 1 at 18.** The two
  new rows are **`xsd/resolve.go:681`** and **`parser/doc.go:219`**, each carrying
  `dead end: cites CLOSED #434`.

**Closing #434 created two STYLE P3 dead ends, and this stamp reports that as its
own effect rather than as a finding about the tree.** `xsd/resolve.go:688` reads
*"Aligning the rest is #434"* and `parser/doc.go:240` *"are #434, which must also
supply the ·lax assessment· fallback"*. **#1426's `## Acceptance` carries
repointing both as its FIRST and UNCONDITIONAL item** — before the ruling, and
whatever the ruling turns out to be — and names **#1376** as the landed fix
pattern to copy rather than reinvent. `parser/doc.go:219` also picks up a
`dead end: cites CLOSED #281` annotation; #281 is cited there as the *aligned*
slot, so that one is provenance and not ownership.

**The area census is flat**: `xsd` 32, `validate` 17, `xpath` 6, `xml` 4,
`parser` 3, `value` 3, `conformance` 2, `cmd` 1. `contentrestricts.go:794`'s
`dead end: cites CLOSED #501` is unchanged and is #1156's, named explicitly
there.

**Zero untracked GAP sites — a judgment on the annotations, not the tool's
mechanical rule.** The two greps the 2026-09-09 stamp found an artifact with
(`No issue owns`, `owns this residual`) were re-run and return the same five
accounted-for hits: `xsd/wildcard.go:121` (#248), `contentrestricts.go:696`
(#1378), `defaultbinding.go:355` and `:549` (#1379), `contentrestricts.go:840`
(#499). **Zero new prose disclaimers.** The three #1378 and #1379 own stay in
group 1 until those issues land, which is the work they own.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 15 pages (14 full at 100, page 15 at 34), PRs filtered locally:
**1434 rows, 634 PRs excluded, 800 issues — 306 open, 494 closed.** Coverage
verified rather than assumed: 1434 distinct numbers, min 1, max 1434, no gaps.
The paginate recipe ran **twice with zero retries**, the second clean pair in
this record.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **128** | active |
| **M5 — Instance validation (XML)** | **15** | **24** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **288 `ready`, 16 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 306 with no gap, and **every open issue carries exactly
one, verified over all 306 by grouping each issue's own label set rather than by
summing counts**. **#779** owns the mechanical check.

By kind: `kind/refactor` 80, `kind/gap` 59, `kind/process` 57, `kind/tooling`
38, `kind/story` 35, `kind/bug` 30, `kind/docs` 21, `kind/feature` 4. By area:
`parser` 85, `meta` 79, `xsd` 64, `docs` 38, `conformance` 31, `cmd` 27,
`validate` 26, `value` 16, `builtin` 11, `xsderr` 7, `xpath` 6, `model` 4,
`regex` 2, `loader` 2, `cli` 1.

**M4 rose 51 → 54 and the three are this pass's own filings** (#1426, #1427,
#1428); its closed count is FLAT at 128, because the window's three landings are
all unmilestoned `cmd`/`tooling` work. **M5 is unchanged at 15 / 24 for a second
stamp** — and #1427 is the first issue in several windows that predicts `instance`
movement with named witnesses, though it sits in M4 because the component seeding
is M4's.

**#434 carried NO milestone for its whole life**, which is worth one sentence
because it undercuts the table above: a core §5.3 schema-construction issue,
banded as the best lane slice in the queue, was outside M4's count. **236 of 306
open issues carry no milestone**, so **neither milestone count is its lane's
remaining work and neither is the area's** — read the `area/` census for that.
Mass-milestoning is not proposed; the undercount is.

`ready` moved **279 → 288**: **nine filed by this pass** (#1426–#1434) and
**#434 left the queue entirely** by closing. That is #347's shape — `ready` is an
output, not a target.

**`ready` 288 is the honest startable count with TWO to subtract**: #1297 and
#1332 each carry a live claim, so **286** are startable without colliding.

**The unblock sweep measured a clean zero for the FOURTEENTH consecutive stamp**,
measured at the source over all **16** open `blocked` bodies rather than
inherited from any post-land pass. **No `## Depends on` in the partition names
any of the window's three closures** (#1354, #1367, #1391) **or #434**. Every
named issue dependency — **#1324**, **#407**, **#250**, **#831**, **#591**,
**#248**, **#1411**, **#1412**, **#1413**, **#1414** — was checked individually
and is **open**. **Zero relabelled.** The set is 16, unchanged in membership from
the last stamp. Re-read the partition below rather than re-deriving it:

- **FIVE are triggers rather than issues** and say so in their own
  `## Depends on` — **#1374**, **#1224**, **#555**, **#1002** and **#1042**.
  **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **NINE still have at least one OPEN named dependency** — **#248**, **#267**
  and **#345** (all on #250), **#415** (#407), **#593** (#591), **#717**
  (#248), **#871** (#831), **#1344** (#1324) and **#731** (the four carves).
  **#731 is the one the band can discharge fastest**: rows 5, 6 and 7 are three
  of its four dependencies, so a window taking all three leaves it one issue from
  closing. **#1344's dependency is band row 12**, and it discharges in EITHER
  direction — #1324 closing `not_planned` is a recorded ruling just as much as
  #1324 landing is.

### #434 RE-PLANNED — what the park bought, and why four issues rather than one

**docs/WORKFLOW.md's Parking rule is the procedure and this pass executed it**:
the arbiter parked at the FIRST rejection without spending the nominally
available repair round (*"PRINCIPLES 30 is a convergence horizon, not a quota to
be run down"*), the cartographer re-planned, filed the replacement, named it on
the thread, and closed #434 **`not_planned`, never `completed`** (#493).

**The replacement is a RULING, not a re-scoped implementation**, because the
arbiter's replan order is an order: every other question is downstream of the
spec question, and no scope cut exists to make anyway.

| # | what it owns | why it is separate |
|---|---|---|
| **#1426** | the §5.1-vs-§5.3 ruling, plus repointing the two markers that now cite closed #434 | the replacement. Carries BOTH outcomes as closeable, including the one where #434's substance is a won't-do |
| **#1427** | §3.2.7's four built-in `xsi:` attribute declarations | **independent of the ruling.** 7 `schema` + 14 `instance`, the largest named-witness prize in the queue |
| **#1428** | the well-known `xml:` namespace declarations | **independent of the ruling**, and a DIFFERENT spec footing from #1427's — §1.3.2, not §3.2.7. The arbiter found this half; the implementation account named only #1427's |
| **#1429** | the #276 decline predicate, re-derived, and the unmeasured decline-flip comparison | wanted under EITHER outcome: if `src-resolve` is upheld the protection is still coincidental, and if §5.3 governs it is already gone |

**#1427's and #1428's predictions are TRANSCRIBED and are INFERENCES, and both
bodies say so in those words.** What was measured on `wip/issue-434` is *the
charge is gone*, not *the reference resolves* — so re-measuring on `main` is each
issue's first grounding obligation. `testdata/xsdtests` is not initialized in this
container (`go tool suiteindex` reports corpus-absent mode), so no case table
here was re-measured.

**Two narrowings were searched for and ruled out IN WRITING so no session
re-spends the search**: keeping the charge where the expanded name is present
under another KIND (35 → 19, refused because `src-resolve` clause 1 makes
resolution kind-specific while §5.3 is unscoped as to why resolution failed), and
exempting references into the XML Schema namespace itself (refused against §5.3's
closing paragraph). Both are carried into #1426's `## Notes`.

**Under outcome (b) a session must STOP.** Losing ≥35 suite cases is a repo-owner
call, and the mechanism it would need does not exist: `Ratchet` refuses on
`len(d.Regressed) > 0` before reading any removal assertion
(`conformance/expectations.go:246`), all 35 are produced and `fail`, and the
classifier is the suite's own `@version`. A new exemption class keyed to anything
else is **a human-filed issue against the ratchet rules**, which CLAUDE.md
reserves. #1426 says this in its own Acceptance.

### Persona consultations — the TWENTY-SECOND ran on the surface arm

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **The orchestrating session ran both personas fresh against the
published surface** (README plus `go doc` plus CLI `-help`, never source) and
handed the reports here.

**The trigger fired on its surface arm again and the floor is still untested.**
#1354 and #1367 both changed what the CLI's no-rule-ID line says, which is the
one thing an operator scripts against, so the surface arm fired on its merits.
**Keep both halves.**

**Ten findings, disposed of as follows — and THREE were corrected in mechanism
before being recorded, in two cases strengthening the disposition.**

**FIVE FILED:**

| # | finding | why it survived, and what this pass added |
|---|---|---|
| **#1430** | README:216 says a delegated verdict is *"reachable as an `*xsderr.Error` of its own through `errors.As` and `xsderr.RuleOf`"* — but a violation IS an `*xsderr.Error`, so both calls match at depth 0 and hand back the OUTER rule | **The persona found it for `errors.As` only. `xsderr.RuleOf` is named in the same sentence and has the identical defect** — its own godoc says *"the first `*Error` in err's chain"* — which a narrow fix would leave behind. This pass also settled the WORDING: prescribe `errors.Unwrap`, **not** `e.Err`, because **#486** unexports Error's fields and would delete the field route. The persona's alternative `Cause()` is explicitly refused — a new export with the README as its only consumer (STYLE 8) |
| **#1431** | `go doc ./parser`'s synopsis lists `Document`/`Element`/`Node`/`Text` beside `Parse` and `ParseReport`, and `{ ... }` elides every word that separates them | The justification exists and is excellent — but only in the per-type view. **#241 and #387 are CLOSED and settled the convention**, correctly, for a reader arriving by `go doc <type>`; this is the reader arriving by `go doc <package>`, which is #1279's rule applied to `go doc`'s own two views. The body enumerates the class so a partial marker cannot make an unmarked harness type read as an entry point |
| **#1432** | no invocation of this binary compiles a multi-`-schema` SET and reports on it | **Measured in this container against a binary built from the tree**: two colliding no-namespace schemas exit **2** (`no instance given`) with zero instances, **3** with one, and `parse a.xsd b.xsd` exits **0** — because `parse`'s own contract says each argument is its own compilation. Filed as a DECISION, because answering a user who forgot an argument with a schema verdict is a real behaviour change for a CLI whose exit codes scripts key on |
| **#1433** | `goxsd8 -help` is 121 lines of contract prose with no worked invocation anywhere | Measured: 121 lines, exit 0, no `Example` or `e.g.` line. README's Quickstart DOES show one-liners, which is the evidence the shorter form is wanted and that the help text is the copy an operator without the repository reaches. The body forbids trimming any normative sentence to make room |
| **#1434** | `-schema` takes one document per flag, with no glob or directory form | Documented behaviour, ranked last by the persona, and filed so the decision is recorded once. The body makes (b) answer three questions the report does not: **whose glob** (shell vs `filepath.Glob`), **what order the set composes in** (order is observable — `sch-props-correct` clause 2 names a *first declared at*), and **whether `gen` gets it** (its `-schema`/`-out` pairing makes a one-to-many flag ambiguous) |

**TWO DEDUPED:**

- **#755** takes libuser's *"no library-level `xsi:schemaLocation` hint reader
  despite the CLI having one"* at full width — its title already names all three
  sites the persona reached from outside. **The README-note half rides with it
  rather than becoming a doc issue**: the note's entire content would be a pointer
  to work already filed, and this issue's landing deletes the reason for it.
- **#1382** takes libuser's value-backend/schema coupling finding, which the
  persona offered as a POSITIVE report. **A second independent reader verified the
  doc's claim and reached the same reason not to fix it** — identity-tracking STYLE
  refuses without a measured need — which is the strongest available argument for
  that issue's outcome (c) and an objection any warden pre-flight reaching (a) or
  (b) must now answer twice. **What the sighting does not supply is the measured
  need**, so it stays unbanded.

**TWO DISMISSED, each on the thread that owns the decision:**

- **#672** — *"no `-version`/`--version`; neither README nor `-help` mentions a
  version flag."* **Dismissed for the THIRD consecutive consultation, and the
  premise is FALSE for README**: `README.md:197` names `-version` explicitly, in
  the sentence announcing that `-help` deliberately omits the argument vocabulary,
  and routes the reader to `cmd/goxsd8/doc.go:232` where the decline is published.
  **Two of three sightings have now run against a documentation premise that was
  wrong.** One mechanism correction recorded: `goxsd8 -version` and
  `goxsd8 validate -version` take DIFFERENT paths (`no subcommand given` versus
  `flag provided but not defined`), both correct under the landed contract. The
  residue — one publication copy, unreachable from an installed binary — is
  **#1089**'s and **#1144**'s, both open, so nothing new is filed.
- **#1031** — *"`-format` is invocation-wide, not per-instance."* **Dismissed on
  the merits, not routed.** The scope IS published, by that issue's own landing
  (*"applying to every instance of the invocation (there is no per-instance
  spelling)"*), and the change **has no reachable witness**: `json` and `ber` both
  exit 2 today, so no invocation distinguishes per-invocation from per-instance and
  no test could pin one. Filing it would produce an Acceptance reading *"re-rule
  the ruling"*. **Explicitly NOT handed to an M8 carve** — that milestone holds
  zero issues, so the hand-off would track nothing (#330). The persona's use case
  (many instances, mixed formats, one CI step) is recorded on the thread as a
  constraint for whoever designs the JSON adapter's entry point.

**ONE NULL REPORT, recorded as a result rather than dropped.** libuser's fifth
item is that `parser.Parse`/`ParseReport`, the `xsd.Schema` query API,
`xsderr.Error` rendering and `validate.Result`/`Unevaluated` all matched `go doc`
exactly. **cliuser's overall verdict is the same shape and is stronger**: every
flag, all five exit codes, the `<loc>: [<rule>] <message>` format, multi-`-schema`
composition, hint augmentation, `-no-hints`, `-format` and every triggered error
message matched the contract — **zero accuracy bugs across ~30 probes**. All five
cliuser findings are UX gaps, which is the first consultation in this record where
that is true of the whole report.

### Working band

Ordered for a `/develop` session: take the highest row you can start. **#1297 and
#1332 are held LIVE and are not here.** Re-run `wipsurvey` before starting
anything.

| # | issue | why here |
|---|---|---|
| 1 | #1137 | **The `kind/process` clause at full strength: a SIXTH data point, and the only friction in this record that costs a whole session per occurrence.** The takeover heartbeat and the mason worktree hand-off do not compose. #1367's session opened on #1332, found the claim EXPIRED, took it (the **sixth** `TAKEOVER:` on that thread), merged forward, delegated to mason — and **lost the entire delegate cycle** when the container restarted, then correctly stood down and took band row 7 instead. `git diff --stat origin/main...2fcbab0` is **empty** after six takeovers. The tax compounds per takeover and the queue will not lift it otherwise (#527, #565) |
| 2 | #1427 | **The largest named-witness prize in the queue, and the only row that predicts `instance` movement with named witnesses.** §3.2.7's four built-in `xsi:` attribute declarations: 7 `schema` cases (`Complex/complex003`–`complex010` carrying `<xs:attribute ref="xsi:type"/>`) and **14 `instance`**. Direction is removal of false rejects. **Its prediction is an INFERENCE from `wip/issue-434`, not a measurement — re-measure on `main` first**, which the body makes the first grounding obligation. The one real regression risk is named in its Notes: seeding four globals collides under `sch-props-correct` clause 2 with any schema that declares them, so seed COMPONENTS, never a synthesized document |
| 3 | #1428 | **#1427's sibling at a tenth the size, and a DIFFERENT spec footing — take it second, not together.** §1.3.2's special-status namespaces, not §3.2.7: the `xml:` declarations are **not** "present in every schema by definition", they live in a well-known document a schema imports, and this processor cannot fetch `http://`. Two cases, `addC001` (`xml:base`) and `isDefault078` (`xml:space`). The decision it must record first is unconditional-versus-on-import, and the answer is not #1427's |
| 4 | #1356 | **Banded on the `kind/process` clause for a THIRD consecutive window, and its scope grew again.** #1354's landing paid two more coordinate corrections and its own log entry calls them *"the second such instance on #1354 alone"*, on top of the instance the last stamp recorded from that same body. The last stamp promoted it on a THIRD file family; the defect tracks whichever files a window touched, which a `citecheck` scoped to three filenames would miss entirely. Cheaper per occurrence than row 1, which is the only reason it is not above the lane slices |
| 5 | #1412 | **The cheapest lane movement in the queue.** #731 mechanism 2: two cases, one rule family, no grammar-versus-`src-*` fork to ground, no overlap with its siblings. Under-rejection, so `schema` up or flat by construction. Carries the thing that is easy to get wrong — the fixtures are the `Schemas/` copies, **not** the same-named files under `Facets/integer/` |
| 6 | #1411 | **#731 mechanism 1, and it must precede #1414.** Three cases, guaranteed direction. The ordering is not preference: `elemE007`/`E008`/`E009` carry its `[0-9]{,5}` pattern as well as their own faults, so landing this first banks them and leaves #1414 witnessless. Read **#546** first — the `src-pattern-value` rejection mechanism already exists in `value/facets.go` and #546 is working on its `Loc` |
| 7 | #1413 | **#731 mechanism 3 — five cases, the biggest carve, and the one that owes a grounding.** The grammar-versus-`src-*` fork is **#444**'s question and must be answered before a line is written. **Re-check against `main` first**: #1369 and #1380 landed s4s tightenings and may already reject some of the five. Its ratchet prediction is required per fault, not in aggregate, because three of the five share `stA012`'s `name=` fault |
| 8 | #1426 | **The replacement for closed #434, and it carries a dead-end repair this pass's own write created.** One oracle pass over §5.1's second bullet against §5.3, §4.2.3 and `src-resolve` clause 1, ruling against `over026`'s 1.1-scoped contrary expectation. **Its first Acceptance item is unconditional and independent of the ruling**: repoint `xsd/resolve.go:688` and `parser/doc.go:240` off closed #434, copying #1376's landed pattern. **Under outcome (b) the session STOPS and escalates** — the body says so, and no agent may open that door |
| 9 | #1429 | **The measurement apparatus, and #434 proved it reports green while pinning nothing.** Both #276 decline guards decline only because the root's reference fails `src-resolve` at finalize; remove that charge and both PASS under `expectValid=false`, for exactly the fabricated reason #276 exists to refuse. The mutation check is the issue: **deleting the finalize charge must leave both tests RED.** Read **#763** first — same family, same fix direction |
| 10 | #1368 | **The `validate`-side half of the landed no-rule-ID work, and the one that puts an operator in front of a filename that does not exist.** A `-schema` document whose root is not `xs:schema` is charged `[src-include]` against `goxsd8-schema-set.xsd:2:1`, while the same input through `parse` prints a bare, rule-free `parser: …`. Distinct from **#1224**, which owns the hint path. **Take it with #1419 and #1421** — all three are `cmd` exit-code-and-message defects in the same two files, and three separate landings rebase three times |
| 11 | #1283 | **The largest row here, banded on a second independent sighting from the audience it is written for.** It **retires #1286 item 3, unblocks #1051's unexport, and deletes the wrapper-as-a-string copy** that #1224, #1250 and #1346 all work around. Needs a warden pre-flight. Read #1251's and #1260's two signature facts in the body before designing the entry point |
| 12 | #1324 | **The only row that discharges a `blocked` entry, and it discharges in either direction.** It challenges the eighth `/retro`'s one-copy ruling on the foreground-gate rule, asking whether `CLAUDE.md`'s gate block — the arrival path #1279 governs — should carry a pointer. **#1344** waits on the ruling and closes with it whichever way it goes. The body states both outcomes as checkable `grep` results, which is what makes it one session |

**Named below the band, deliberately:** **#1414** (#731 mechanism 4) is startable
and is **not** banded, because rows 6 and it are sequenced — if #1411 lands first,
#1414's entire named cohort is already `pass` and it banks nothing. **#1419** and
**#1421** belong with row 10 and are named there rather than given rows of their
own. **#1417** (the chronicler agent file's append-only sentence against a
practice that has corrected a landed entry in place three times) and **#1400**
(`landcheck`'s short-circuiting fixture guard) are both real, both one session,
and both below twelve rows of higher-cost work. **#1430–#1434** are this pass's
consultation filings and none is banded: four are doc- or decision-shaped and
#1430's wording is constrained by #486.

### Next planning action

1. **Band the friction that costs a SESSION above the friction that costs a
   pass, and both above a lane slice.** Row 1 (#1137, six data points, a lost
   delegate cycle) and row 4 (#1356, three windows, a correction pass) are both
   `kind/process` banded under the same clause, and the ordering between them is
   cost per occurrence. **This is the clause's first two-member application in
   this record** — say whether it held at the next stamp, because the alternative
   reading (all consecutive-friction process work above all lane slices) would
   have starved rows 2 and 3, which are the best named-witness prizes in the
   queue.
2. **A band is not where sessions are going, and this is now measurable.** TWO
   claims are held and NEITHER is in the last band: #1297 was never banded at
   all, #1332 was dropped as held. The window's three landings were rows 1, 6 and
   7, and row 2 was taken and parked. **The band-versus-window throughput
   question is already owed to the next `/retro`**; this stamp adds the sharper
   version — *what fraction of held claims were ever banded* — which is a
   countable thing the next two stamps can answer.
3. **The prediction question now has FOUR data points and #434 is the one that
   matters.** #1369 banked +17 off a hand-rolled census; #1380 predicted movement
   and measured zero; **#434 named three fixtures that are not scored at all**,
   discoverable by one `grep` against `schema.txt` and first run in the
   implementation round. `suiteindex` has no version-or-discovery axis and could
   not have answered it. **#1332 is LIVE against exactly this** — when it lands,
   ask whether its rule reaches the #434 case, because a body-versus-banked check
   catches a wrong figure and not a census that never ran.
4. **No step rules on whether an issue AS A WHOLE can land before a mason round
   pays for the answer.** #434 absorbed an oracle grounding, a warden pre-flight
   and three cartographer body corrections; each found a further defect in the
   issue; and it was still unlandable for a reason none of them is positioned to
   ask about. **One sighting, recorded as a datum rather than filed** — `/retro`
   is where it becomes a pattern or does not, and it should be put there with
   item 2.
5. **#849's steward re-ranking is owed for a SEVENTH consecutive stamp and the
   deadline is TWO DAYS out.** The standing condition is *"file it at the next
   `/retro`, or at the next `/backlog` if the retro passes over it again"*. The
   last `/retro` was **2026-09-06** and the routine is weekly, so **Sunday
   2026-09-13** stands: **if that retro passes over #849 again, the next
   `/backlog` files the routing issue** — the subject being that a routed
   re-ranking has no ledger and therefore never happens, which is #330's rule
   applied to the steward's own hand-offs. It is not filed here because the
   deadline has not passed. **No `kind/refactor` in the queue currently carries a
   steward cost-of-delay ranking**, so #843's banding clause had nothing to act on
   this pass — with 80 open `kind/refactor` issues, that absence IS the defect
   #849 names, measured from the banding side.
6. **The human decision blocking #1002 is unchanged and is now carried for an
   EIGHTEENTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — and (b) depends on **#1042**, filed and `blocked`.
   **No agent should attempt either.** **#1426's outcome (b) is the same shape
   and the same prohibition**, and the two should be ruled together if they are
   ever ruled.
7. **M6's carve is still owed and #1042 is still its only member.** #1042
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
times; the 2026-09-06 `/retro` rewrote a neighbouring bullet, so re-read it
before grounding any of the four. **The next `/retro` inherits seven**: the
fold-the-five-species question (#635, #912, #609, #510, #646), the
`[tests that cannot fail]` **pattern** (routed by #472's post-land, with no filed
carrier — and **#1429** is now a concrete instance of it), **#849's owed steward
re-ranking**, the **band-versus-window throughput** question sharpened by item 2,
the **persona-trigger floor**, the **carve-labelling question** the last stamp
raised, and now **item 4's can-this-issue-land-at-all question**. The
`gh --paginate` page-2 403 trap stays a `/retro` datum and not a filing. The CTA
cohort's 45 banked `instance` failures remain unattributed. `gate.yml` runs and
is still not a required status check, which only the repository owner can change.

**Environment, one witness each.** **`main` did NOT move under this pass** —
`06add85` at the session brief and at every measurement. **The checkout was
SHALLOW and was unshallowed before any range reading** (`git fetch --unshallow
origin`, #802); the eight-branch ancestor check under Branch namespace is
impossible without it. **`gh` REST served every read and every write**; the
paginate recipe was run **twice and needed ZERO retries**, 15 pages each time,
coverage verified by distinct-number count. **EIGHTEEN writes across FOURTEEN
distinct issues**: **nine creates** (#1426–#1434); **three PATCHes on #1426**,
two to its body (adding the three sibling numbers, then the unconditional
marker-repoint item) and one to its title; **one state PATCH** closing #434
`not_planned`; and **five thread comments** (#434 for the re-plan; #755 and
#1382 for the two dedupes; #672 and #1031 for the two dismissals). **Probes were run against a binary built
from the tree** — the zero-instance versus one-instance `validate` contrast on a
colliding two-document set, `parse` on the same pair, both `-version` spellings,
and the full 121-line `-help` — and **one of them falsified a finding's premise**.
**`gapaudit` was run TWICE, before and after the writes**, which is how the
16 → 18 group-1 move is reported as this pass's own effect rather than as a
finding. **No conformance measurement was taken by this pass**: the lane table
above is the committed expectations, which `docs/WORKFLOW.md` names as the lane
score (#1120). **`testdata/xsdtests` is NOT initialized here**, which is why
#1427's and #1428's case tables are marked transcribed inferences rather than
measurements.

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
entirely, and the expensive case is §5.3.** Whether a failed `·resolution·` is a
schema error at all is unsettled — **#1426** owns the ruling, **#1427** and
**#1428** own the two missing built-in component sets the question surfaced, and
**#1429** owns the harness guard it broke. None of the three implementation
issues waits on the ruling.

**Read the milestone count as a floor.** The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it — #1135 is M4 work carrying no
milestone today, and #1136 was too until it landed. **The sharpest instance for
LANE MOVEMENT is #1126**, which moved `schema` +475 and `instance` +113 — the
largest this project has recorded — and carried no milestone, so the count did
not move at all. **The sharpest for SCOPE is #434**: a core §5.3
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
  (`<simpleContent>` `<restriction>`), the CTA pair #842/#851. **#1427 is the
  next of these and is M4**: §3.2.7's four built-in `xsi:` attribute
  declarations are not seeded, so a schema referencing one hard-fails
  `src-resolve` and **14 `instance` cases never reach the engine at all**. That
  prediction is an inference transcribed from `wip/issue-434` and is that issue's
  first grounding obligation, not a banked figure.
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
