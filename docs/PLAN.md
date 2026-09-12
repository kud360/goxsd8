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

## Status — 2026-09-12 (`/backlog`, the twenty-third. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against `git fetch --unshallow origin` plus `git ls-remote --heads`, the marker census a fresh `gapaudit` run TWICE — before and after this pass's writes — and the milestone and queue counts a page-numbered `state=all` fetch taken **after** them. **The window is FIVE landings and TWO lanes MOVED**, ending the flat window: `schema` 14012 → **14075** and `instance` 11029 → **11044**. **The event is that PREDICTION started working** — #1412 predicted **+56** from a census joined to `schema.txt` and banked +56 case for case, #1427 predicted 7/14 and banked **7/15**, and #1428 has banked **exactly** its two named cases on an unlanded branch: three consecutive exact predictions after four consecutive misses. **The band under-predicted #1412 by 54** by inheriting a two-case table from its body, which is the same failure as the last stamp's false claim that **#843's clause had nothing to act on** — #407 has carried a steward ranking since 2026-08-02. **ONE claim is held, `wip/issue-1428`, and it IS band row 3** — the first stamp in this record where a held claim was banded. **The marker census is 69; group 1 holds at 18 before AND after this pass's writes.** The **TWENTY-THIRD persona consultation**: **nine findings — four filed (#1452–#1455), one deduped (#1432), one dismissed against a landed ruling (#687), one split (dismissed on a falsified premise, general half filed), two confirmations**)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11044 | 15317 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14075 | 1323 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This table is `main` at `b2d2490` and a banked +2 is already standing outside
it.** `wip/issue-1428` carries `conformance: bank the schema lane's two xml:
namespace flips` at `c27bb85`, making `schema` **14077** on that branch. It is
LIVE, not merged, and the number above is the committed expectations on `main`,
which `docs/WORKFLOW.md` names as the lane score (#1120). The 2026-09-10 stamp
recorded its predecessor publishing a table that was stale at the moment it
landed; this one says where the next two cases already are.

### The window: five landings, two lanes moved, and prediction starts working

| commit | issue | ratchet |
|---|---|---|
| `c88575e` | #1297 — `suiteindex` spells a no-namespace name `{}local` | unchanged |
| `4d6214c` | #1332 — the arbiter re-derives ratchet/tree-state Acceptance claims | unchanged |
| `de6016b` | **#1412** — a `simpleType` restriction whose own bound facets are inconsistent | **`schema` +56** |
| `c1f31b5` | **#1427** — §3.2.7's four built-in `xsi:` attribute declarations | **`schema` +7, `instance` +15** |
| `34c1f26` | #1137 — the mason hand-off names both mechanisms and forbids replay | unchanged |

**Three of the five were band rows** (1, 2 and 5) and **both held claims
landed** — against last window's zero of two. The `kind/process`-above-lane-slice
clause held on its first two-member application: row 1 landed and rows 2 and 3
were not starved.

**The prediction question has six data points and the last three are exact.**
#1369 banked +17 off a hand-rolled census; #1380 predicted movement and measured
zero; #931 under-predicted by two off a producer probe; **#434 named three
fixtures that are not scored at all**. Then:

- **#1412 predicted +56 BEFORE the suite ran** — from a census of the 347
  `restriction` elements carrying both a lower and an upper bound **joined
  through the meta files to `schema.txt`** — and measured +56 case for case.
- **#1427** had its filed prediction ruled UNSATISFIABLE at grounding and its
  body corrected twice before a line was written, under the rule #1332 landed in
  this same window; it then banked 7 `schema` and 15 `instance` against a
  corrected 7/14.
- **#1428** has banked `addC001` (`xml:base`) and `isDefault078` (`xml:space`) —
  **exactly the two cases the last stamp's band row 3 named**.

**What makes the three work is the JOIN, and no document names it.** A construct
census reads the corpus; a lane figure lives in `conformance/testdata/expectations/`.
#1332's landed rule names `go tool suiteindex` as the re-derivation instrument
and therefore **cannot reach the #434 case**: `schema.txt` is 15398 lines and
`grep -ic missing` returns **0**, while `versionApplicable` (#446) withholds the
whole `saxonMeta/Missing` set silently and `discovery.withheld` (#576) computes
that list on every run and prints it to nobody — `GOXSD_DECLINES=1` exists,
no withheld counterpart does. **#1451** owns it and is band row 9.

**The band under-predicted #1412 by 54.** It was ranked *"the cheapest lane
movement in the queue … two cases"*, a figure inherited from the issue body's
case table rather than re-derived. **The same failure produced the last stamp's
claim that no open `kind/refactor` carried a steward cost-of-delay ranking, so
#843's clause *"had nothing to act on this pass"*: #407's `## Notes` has read
*"ranked S-A, cost-of-delay #1"* since 2026-08-02**, and five more open
`kind/refactor` issues carry a `## Cost of delay` section (#410, #845, #848,
#849, #1333). Both are the band taking a number from a body instead of
re-deriving it, and both are corrected here — #407 is banded at row 6 on a
census this pass ran.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against `git fetch --unshallow origin` and
this pass's **809-issue post-write feed**:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
434    wip/issue-434   26h26m0s   RETIRED  wip/issue-434: issue #434 is closed
732    wip/issue-732   477h20m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   655h18m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   427h39m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   621h19m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   main's     RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   main's     RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   433h0m0s   RETIRED  wip/issue-993: issue #993 is closed
1332   wip/issue-1332  5h34m0s    RETIRED  wip/issue-1332: issue #1332 is closed
1428   wip/issue-1428  12m0s      LIVE     wip/issue-1428: tip pushed 12m0s ago, within the 2h0m0s claim TTL
```

**Four of the window's five branches vanished at merge and ONE did not.**
`wip/issue-1297`, `-1412`, `-1427` and `-1137` are gone, which is the auto-delete
working. **`wip/issue-1332` survived its own squash merge** and is the NINTH
RETIRED row: tip `b2719379`, `NOT-ancestor` of `main` as every squash-merged
branch is, and its content verified present on `main` at `4d6214c`
(`.claude/agents/arbiter.md` `+22/−0`). **No supersede is owed.** Why it survived
when four siblings did not is unexplained and is a one-line observation, not a
filing.

**ONE claim is held and it IS BAND ROW 3 — the first time in this record.**
`wip/issue-1428` moved twice under this pass (`3bdd27a` → `c27bb85`) and carries
real content plus an arbiter bank, not a heartbeat. **Re-run `wipsurvey` before
starting anything**; a mid-pass read of it returned `UNKNOWN — tip not fetched`
and needed a `git fetch` before it read LIVE again.

**The seven older RETIRED refs are unchanged row for row — TEN stamps now — and
ancestry was re-measured rather than inherited.** `wip/issue-933` and
`wip/issue-968` ARE ancestors of `main`; the other five are NOT, tips dated
2026-08-16 to 2026-08-25. That is the expected shape: all seven closed
`not_planned` and were superseded rather than merged (#732→#1001/#1002,
#822→#851, #846→#1029/#1030, #872→#878, #993→#1018). **`wip/issue-434` is
preserved deliberately** — it is #434's measurement and #1426 names it as
evidence not to re-pay for.

**Zero `needs-replan`. Zero `parked/*`. Zero `meta/*`. FIVE non-`wip` `claude/*`
refs stand**, unchanged in tip for a FOURTH stamp — `-39rk64`, `-3xu0ki`,
`-8jq9o6`, `-adewly`, `-kk1f7v` — and they are drifting: **40, 112, 124, 166 and
171 commits behind** `main` respectively, where `-kk1f7v` was ELEVEN behind at
the last stamp. Listed for human triage, not acted on; `wipsurvey` reads `wip/*`
and `parked/*` only, so these are found by `git ls-remote --heads` and never by
the survey.

### Marker census — 69 markers; group 1 flat at 18 across this pass's writes

**`go tool gapaudit` was run TWICE, before and after this pass's writes.**

- **Pre-write, against the 804-issue feed: 69 markers across 8 areas, group 1 at
  18.**
- **Post-write, against the 809-issue feed: 69 markers, group 1 at 18** — the
  same eighteen rows in the same order.

**The census grew 68 → 69 and the new marker is #1427's own**: a `GAP(xsd):
no-xsi (§3.2.6.4)` in `parser`, taking `xsd` from 32 to 33. It is NOT in group 1,
because **#1446** was filed from the same landing's warden pre-flight and cites
it — a landing that added a marker and its tracker in one window, which is the
shape to want.

**This pass's only measured effect on the audit is one weak annotation it caused
itself**: #1426's group-2 row gained a `phrase-matches xsd/schema.go:426` line
after this pass edited that body. `gapaudit` labels it *"too weak to retire this
tracker"*, and it is an annotation rather than a row.

**Both `dead end: cites CLOSED #434` rows still stand** — `xsd/resolve.go:681`
and `parser/doc.go:219` — and **#1426's first Acceptance item owns repointing
them, unconditionally and ahead of its ruling**, copying #1376's landed pattern.
`parser/doc.go:219` also carries a `cites CLOSED #281` annotation, which is
provenance rather than ownership, and `contentrestricts.go:794`'s `cites CLOSED
#501` is #1156's.

**The area census is otherwise flat**: `validate` 17, `xpath` 6, `xml` 4,
`parser` 3, `value` 3, `conformance` 2, `cmd` 1.

**Zero untracked GAP sites — a judgment on the annotations, not the tool's
mechanical rule.** The two greps the 2026-09-09 stamp used return **three**
accounted-for hits, all owned: `contentrestricts.go:696` (#1378),
`defaultbinding.go:355` and `:549` (#1379). The other two sites that stamp
counted — `xsd/wildcard.go:121` and `contentrestricts.go:840` — **name their
owners in the prose** (#248 and #499 respectively) and so do not match a
"no issue owns" grep at all, which is a better state rather than a discrepancy.
**Zero new prose disclaimers.**

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** every write
this pass made, 15 pages (14 full at 100, page 15 at 55), PRs filtered locally:
**1455 rows, 646 PRs excluded, 809 issues — 310 open, 499 closed.** Coverage
verified rather than assumed: 1455 distinct numbers, min 1, max 1455, no gaps.
The paginate recipe ran **three times with zero retries**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **54** | **129** | active |
| **M5 — Instance validation (XML)** | **15** | **24** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **292 `ready`, 16 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 310 with no gap, and **every open issue carries exactly
one, verified over all 310 by grouping each issue's own label set rather than by
summing counts**. **#779** owns the mechanical check.

By kind: `kind/refactor` 80, `kind/gap` 59, `kind/process` 58, `kind/tooling`
38, `kind/story` 35, `kind/bug` 30, `kind/docs` 23, `kind/feature` 4. By area:
`parser` 87, `meta` 80, `xsd` 62, `docs` 39, `conformance` 32, `cmd` 27,
`validate` 26, `value` 17, `builtin` 11, `xsderr` 9, `xpath` 6, `model` 4,
`loader` 2, `regex` 2, `cli` 1.

**M4's closed count rose 128 → 129 and its open count did NOT move**: #1427
closed out of it and **#1446 was filed into it by that same landing's post-land
pass**, so the milestone replenished itself in one window. **M5 is unchanged at
15 / 24 for a third stamp** — and yet `instance` moved +15, banked by an M4
issue, which is the milestone-count-as-a-floor problem stated from the lane side
rather than the scope side. **238 of 310 open issues carry no milestone**, so
neither milestone count is its lane's remaining work; read the `area/` census for
that.

`ready` moved **288 → 292**: **five filed by this pass** (#1451–#1455) and
**#1412, #1427 and #1137 left the queue** by closing, against four filed by
post-land passes earlier in the window. That is #347's shape — `ready` is an
output, not a target.

**`ready` 292 is the honest startable count with ONE to subtract**: #1428 carries
a live claim, so **291** are startable without colliding.

**The unblock sweep measured a clean zero for the FIFTEENTH consecutive stamp**,
measured at the source over all **16** open `blocked` bodies' `## Depends on`
sections rather than inherited from any post-land pass. **No entry names any of
the window's five closures.** **Zero relabelled.** The set is 16, unchanged in
membership from the last stamp. **One partial discharge is worth recording:
#731's dependency list went from FOUR open carves to THREE** — #1412 closed — and
it correctly stays `blocked` and untouched. Re-read the partition below rather
than re-deriving it:

- **FIVE are triggers rather than issues** and say so in their own
  `## Depends on` — **#1374**, **#1224**, **#555**, **#1002** and **#1042**.
  **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **NINE still have at least one OPEN named dependency** — **#248**, **#267**
  and **#345** (all on #250), **#415** (#407), **#593** (#591), **#717**
  (#248), **#871** (#831), **#1344** (#1324) and **#731** (three of four carves
  now). **#415 is the one the band can discharge fastest**: its sole dependency
  is **#407**, band row 6. **#731 needs three landings** — rows 4 and 7 are two
  of them. **#1344's dependency is band row 12** and discharges in EITHER
  direction, since #1324 closing `not_planned` is a recorded ruling just as much
  as #1324 landing is.

### Persona consultations — the TWENTY-THIRD, and the surface arm fired on merits

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **The orchestrating session ran both personas fresh against the
published surface** (README plus `go doc` plus CLI `-help`, never source) and
handed the reports here. Both were told #1430–#1434 were already filed and
neither re-reported them.

**The trigger fired on its surface arm and this window earned it twice.** #1412
put **four new rule IDs into the `[<rule>]` slot operators script against** and
made a schema that used to compile exit 1; #1427 made a schema that used to be
rejected compile. Both reported `surface: unchanged` from the tool and **both
changed observable behaviour**, which is the case the surface arm exists for.

**Nine findings. FOUR were corrected in mechanism before disposition, and TWO of
those corrections REVERSED the outcome.**

**FOUR FILED:**

| # | finding | what this pass added |
|---|---|---|
| **#1452** | the four bound-consistency messages end by repeating their own rule ID verbatim — found independently by BOTH personas, from the CLI line and from `Msg` | **The remedy is REVERSED.** Both reports proposed deleting the parenthetical as noise. `README.md:208` shows the convention is **(constraint name, §section)** and that the two names normally DIFFER — bracket `[cvc-datatype-valid]`, tail `(decimal-lexical-representation, §3.3.3.1)`. These four are the degenerate case where they coincide, so **the missing half is the §section**, which `boundConsistencyViolates`' own doc comment already states for all four. The body also refuses the over-general rule: `src-resolve`'s tail is `(src-resolve clause 1.3)` and is correct |
| **#1453** | a facet charge is located at the enclosing `simpleType`, never at the offending facet | **Re-framed off #1412 and off #546.** The charge already passes `rc.owner.Loc()`, which IS the target state #546 and #664 are working toward for the zero-Loc family. **Facets carry no position anywhere in `value`** — `boundFacet` holds `limit` and `kind` — so this is one rung above that family and must not be closed by #546's landing |
| **#1454** | the four rule IDs are not referenceable symbolically; no catalog enumeration exists at all | **The narrow claim is DISMISSED INSIDE the issue on a falsified premise.** libuser argued the four are *"invented by this module"*; **all four appear in `xmlschema11-2.md`**, and the two IDs that ARE exported — `component-invariant`, `xml-wf` — appear in **zero** spec files. That is a coherent policy (export exactly what the spec does not name) followed in the tree and stated nowhere, which is what the issue now asks to publish |
| **#1455** | `xs:attribute ref="xsi:type"` compiles and the capability is discoverable only by cross-reading two paragraphs | Both were read first-hand and both say what the report says. One documents the ORDER of `Attributes()`, the other QName resolution; **neither names the syntax an author writes**, which is #1279 exactly. The body rules OUT the reporter's README-changelog remedy: `docs/LOG` owns history and a changelog is a second encoding (STYLE D3) |

**ONE DEDUPED, and the persona's own non-overlap claim was falsified:**

- **#1432** takes cliuser's *"no CLI command compiles a multi-file schema set
  without a throwaway instance"* at full width. The report argued explicitly that
  it does **not** overlap #1432, *"which is about multi-schema NAMESPACE COLLISION
  detection"*. **#1432's `## Spec` names `src-resolve` alongside
  `sch-props-correct`**, and its defect sentence is the general one. Two things
  went into the body rather than a new issue: **the import route answers WRONGLY
  where the collision route merely fails to answer** (`parse order.xsd items.xsd`
  exits **1** with a real `src-resolve` charge against a set `validate` composes
  at 0), and **`README.md:84-85` is a trap** — the `parse` and `validate` example
  lines are ADJACENT and use the SAME two filenames. **Two personas have now
  reached this defect by two different rules**, which is the strongest argument
  yet for its arm (a) over its documented-workaround arm (b).

**ONE DISMISSED, on the thread that owns the decision:**

- **#687** — *"no per-subcommand help; all four `-help` spellings print an
  identical 121-line block."* Re-measured here and byte-identical, exactly as
  reported — and it is **#687's shipped answer**, landed 2026-08-24 in the #870
  bundle: *"Help is never scoped to a subcommand … so both `goxsd8 parse -h` and
  `goxsd8 -xyz -help` print it and exit 0."* **This thread has now dismissed BOTH
  halves of its own decision on independent outside re-probes** — the bareword
  `help` half on 2026-09-06, the scoped-`-help` half today. The decline's forward
  clause (*"A subcommand that parses its own flags may narrow this when it
  lands"*) **has not fired**, and the comment says a future sighting is real only
  if it has. Distinct from #1433 (no worked example), which is untouched.

**TWO CONFIRMATIONS, recorded as results.** The four facet-bound rules fire
correctly and match the `<loc>: [<rule>] <message>` contract exactly; all four
`xsi:` names compile and validate. cliuser's broader verdict repeats the
twenty-second's shape — every flag, all five exit codes, exit-code dominance,
`-q`/`-v` scoping, `-schema -` rejection, flag-before-subcommand and
unknown-subcommand all matched the contract — so **two consecutive consultations
have now found zero accuracy bugs in the CLI contract**, and every finding in
both was a gap rather than a falsehood.

### Working band

Ordered for a `/develop` session: take the highest row you can start. **#1428 is
held LIVE and is band row 3's subject — it is NOT a row here.** Re-run
`wipsurvey` before starting anything.

| # | issue | why here |
|---|---|---|
| 1 | #1443 | **The `kind/process` clause at full strength, and the largest unowned cost in this record: TWELVE mason delegations lost to container restarts on ONE branch**, each a whole session, with grounding complete from day one and the branch diff empty for twelve cycles. **#1137 landed the hand-off MECHANISM in this window and explicitly does not touch the loss window.** Twice the data points row 1 carried last stamp |
| 2 | #1446 | **The best-evidenced lane slice in the queue, and the only one whose prediction is MEASURED rather than inferred.** `no-xsi` (§3.2.6.4): **+3 `schema`**, three named fixtures, and **TWO named guard cases that must not move** — `attKb018`/`attKc018`, banked `pass`, differing from a flip case only in `attributeFormDefault`. The population is proven complete by `suiteindex 'schema@targetNamespace'` (six fixtures, no seventh), and the wrong implementation is named: read the declaration's RESOLVED `{target namespace}`, never the document's `targetNamespace`. `no-xmlns` is the template. M4 |
| 3 | #1437 | **The repetition half of row 1's family**, and its own instance discharged this pass when #1332 closed — the class is untouched: `RETIRED` is reachable only through the ISSUE's state, so any open issue whose branch produces nothing stays the pick step's preferred `EXPIRED` target forever. Take it after row 1, whose fix may change what a "produced nothing" branch looks like |
| 4 | #1411 | **#731 mechanism 1, and it must precede #1414.** Three cases, guaranteed direction. The ordering is not preference: `elemE007`/`E008`/`E009` carry its `[0-9]{,5}` pattern as well as their own faults, so landing this first banks them and leaves #1414 witnessless. Read **#546** first |
| 5 | #1356 | **Banded a FOURTH consecutive window, and the escalation is qualitative.** Two issues filed this window (#1442, #1443) omit line numbers ENTIRELY and cite #1356 as the reason, and #1332's landed text cites CLAUDE.md's surveys block *"by NAME and with no line number"*. **The defect now shapes how issues are written rather than merely costing corrections** — three artifacts routing around it in one window |
| 6 | #407 | **#843's clause, first real application in this record — and the clause was reported as having nothing to act on while this issue sat in the queue.** Steward **S-A, cost-of-delay #1** since 2026-08-02; re-measured by this pass at `b2d2490` to **17 constructors / 38 call sites**. **It is banded but not higher, because the divergence is FLAT**: 13→17 constructors in the three weeks to 2026-08-23, then **+0 constructors and +1 call site** in the three weeks since. **Landing it discharges #415** from the `blocked` partition. Warden pre-flight mandatory — 17 signature changes |
| 7 | #1413 | **#731 mechanism 3 — five cases, the biggest carve, and the one that owes a grounding.** The grammar-versus-`src-*` fork is **#444**'s question and must be answered before a line is written. **Re-check against `main` first**: #1369 and #1380 landed s4s tightenings and may already reject some of the five. Its ratchet prediction is required per fault, not in aggregate |
| 8 | #1426 | **The replacement for closed #434, and its first Acceptance item is UNCONDITIONAL**: repoint `xsd/resolve.go:681` and `parser/doc.go:219` off closed #434, which this pass re-confirmed are still group 1's only two dead ends. **#1427's landing turned this issue's central bullet from an inference into a measurement** — seeding one of the two missing component sets banked +7/+15 with the §5.3 charge untouched — and the body now says so. **Under outcome (b) the session STOPS and escalates** |
| 9 | #1451 | **New this pass, and #434 is its costed sighting: a whole session.** A construct census cannot see whether a case is SCORED, `versionApplicable` withholds silently, and `discovery.withheld` already computes the answer on every run and shows it to nobody. **Arm (a) mirrors `GOXSD_DECLINES=1`**, which already exists, so it is plausibly one session |
| 10 | #1429 | **The measurement apparatus, and #434 proved it reports green while pinning nothing.** Both #276 decline guards decline only because the root's reference fails `src-resolve` at finalize; remove that charge and both PASS under `expectValid=false`. **The mutation check IS the issue**: deleting the finalize charge must leave both tests RED. Read **#763** first |
| 11 | #1368 | **The one that puts an operator in front of a filename that does not exist.** A `-schema` document whose root is not `xs:schema` is charged `[src-include]` against `goxsd8-schema-set.xsd:2:1`. Distinct from **#1224**, which owns the hint path. **Take it with #1419 and #1421** — all three are `cmd` exit-code-and-message defects in the same two files, and three separate landings rebase three times |
| 12 | #1324 | **The only row that discharges a `blocked` entry by RULING rather than by landing code**, and it discharges in either direction. **#1344** waits on it and closes with it whichever way it goes. The body states both outcomes as checkable `grep` results, which is what makes it one session |

**Named below the band, deliberately:** **#1414** (#731 mechanism 4) is startable
and is **not** banded, because rows 4 and it are sequenced — if #1411 lands
first, #1414's entire named cohort is already `pass`. **#1419** and **#1421**
belong with row 11 and are named there. **#1442** (the pen carve-out's
"text no compiler reads" arm) has ONE sighting and its own arbiter ruled it not a
precedent, which is a weaker footing than rows 1 and 3 in the same family.
**#1283** is the largest single row available, retires #1286 item 3 and unblocks
#1051's unexport, and wants a warden pre-flight. **#1417**, **#1400** and
**#1355** are each real and each one session. **#1452–#1455** are this pass's
consultation filings and none is banded: #1453 and #1454 are decision-shaped,
#1455 is doc-only, and **#1452 is the closest to bandable** — a contained
message fix whose remedy this pass already settled.

### Next planning action

1. **Say whether the JOIN should become normative, and note that #1451 asks the
   question from only one side.** Three consecutive exact predictions all used a
   corpus census joined to the lane's expectations file; the two documents that
   tell a session how to predict (`CLAUDE.md`'s surveys block, `arbiter.md`'s
   #1332 paragraph) name only the corpus half. #1451 owns the missing
   instrument; **whether the JOIN itself is the documented method is a separate
   question this stamp raises and does not file**, because one more exact
   prediction would make it a pattern and one miss would refute it.
2. **The band inherited a figure from a body TWICE and both are corrected here.**
   #1412 was ranked on a two-case table and banked 56; #843's clause was reported
   as having nothing to act on while #407 sat in the queue carrying a steward
   ranking. **The rule this suggests — a band row's magnitude claim is re-derived
   at banding time, not copied from the issue** — is the same shape #1332 landed
   for `## Acceptance` bullets at grounding time. **Do not file it yet**: it is
   two sightings in one pass, and the next stamp should say whether banding-time
   re-derivation actually happened or whether this paragraph is the only thing
   that changed.
3. **Band adherence is now measurably better and the metric should keep being
   taken.** Three of five landings were band rows, both held claims landed, and
   **for the first time in this record the one held claim IS a band row** (#1428,
   row 3). The `/retro` question the last two stamps routed — *what fraction of
   held claims were ever banded* — now has a second data point (1 of 1 this
   stamp, 0 of 2 last), and the answer is trending the right way for a reason
   nobody has established.
4. **#849's steward re-ranking is owed for an EIGHTH consecutive stamp and the
   deadline is TOMORROW.** The standing condition is *"file it at the next
   `/retro`, or at the next `/backlog` if the retro passes over it again"*. The
   last `/retro` was **2026-09-06** (eighth weekly, both parts) and the routine is
   weekly, so **Sunday 2026-09-13** stands: **if that retro passes over #849
   again, the next `/backlog` files the routing issue.** It is not filed here
   because the deadline has not passed. **The last stamp's supporting claim was
   FALSE and is corrected above** — #843's clause had #407 to act on, and acting
   on it is row 6.
5. **No step rules on whether an issue AS A WHOLE can land before a mason round
   pays for the answer.** #434 absorbed an oracle grounding, a warden pre-flight
   and three cartographer body corrections, each finding a further defect, and was
   still unlandable for a reason none of them is positioned to ask about. **One
   sighting, carried as a datum** — `/retro` is where it becomes a pattern or does
   not. Note that this window ran the counter-example: **#1427's filed prediction
   was ruled UNSATISFIABLE at grounding and the body corrected TWICE before
   implementation**, which is the same class of defect caught at the cheap end.
6. **The human decision blocking #1002 is unchanged and is now carried for a
   NINETEENTH stamp.** #1002 waits on a ruling between (a) a constitutional
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
times; **#1451's `## Notes` names that paragraph as a host to AVOID** for exactly
that reason. **The next `/retro` inherits eight**: the fold-the-five-species
question (#635, #912, #609, #510, #646), the `[tests that cannot fail]` pattern
(routed by #472's post-land, no filed carrier — **#1429** is a concrete
instance), **#849's owed steward re-ranking**, the **band-versus-window
throughput** question sharpened by item 3, the **persona-trigger floor**, the
**carve-labelling question**, item 5's **can-this-issue-land-at-all** question,
and now item 2's **banding-time re-derivation** question. #1137's post-land also
routed its hand-off-merge conflict case there, unfiled and unsighted. The
`gh --paginate` page-2 403 trap stays a `/retro` datum and not a filing. The CTA
cohort's 45 banked `instance` failures remain unattributed. `gate.yml` runs and
is still not a required status check, which only the repository owner can change.

**Environment, one witness each.** **`main` did NOT move under this pass** —
`b2d2490` at the session brief and at every measurement — but **`wip/issue-1428`
moved twice**, and a `wipsurvey` run between its pushes returned
`UNKNOWN — tip not fetched` until a `git fetch` was issued: the survey reads the
local remote-tracking ref, not the remote. **The checkout was SHALLOW and was
unshallowed before any range reading** (`git fetch --unshallow origin`, #802);
the ten-branch ancestor check above is impossible without it. **`gh` REST served
every read and every write**; the paginate recipe ran **three times and needed
ZERO retries**, 15 pages each time, coverage verified by distinct-number count.
**TWELVE writes across TEN distinct issues**: **five creates** (#1451–#1455);
**four body PATCHes** — #1437 (stale premise corrected, hypothesis routed to
#1443), #1426 (inference to measurement), #407 (census pointer) and #1432
(second sighting plus the README evidence); and **three thread comments** —
#407's re-measurement, #1432's dedupe and #687's dismissal. Every body PATCH was
read back and diffed; the only delta anywhere is #407 picking up the idempotent
trailer (#900). **No issue was closed, no label changed and no milestone moved by
this pass.**
**Probes were run against a binary built from the tree** — the four `-help`
spellings captured and `diff`ed, a bound-inconsistency fixture with its facets at
known line numbers, and an importing `order.xsd`/`items.xsd` pair through both
`parse` and `validate` — and **two of them falsified a finding's premise**.
**`gapaudit` was run TWICE, before and after the writes**, and held at 18 both
times. **No conformance measurement was taken by this pass**: the lane table
above is the committed expectations, which `docs/WORKFLOW.md` names as the lane
score (#1120). **`testdata/xsdtests` is NOT initialized here** (`go tool
suiteindex` reports corpus-absent mode), which is why #1446's case table is cited
from its own measured filing rather than re-measured, and why #1451's
`version="1.0"` half is marked a hypothesis while its `schema.txt` half is
measured.

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
schema error at all is unsettled — **#1426** owns the ruling, and **#1429** owns
the harness guard the attempt broke. **The two missing built-in component sets
the question surfaced are no longer open**: **#1427** LANDED 2026-09-12
(`c1f31b5`, `schema` +7, `instance` +15) and **#1428** is in flight with its two
cases already banked on its branch. **Neither waited on the ruling, and that is
the finding** — the withheld flips were a missing component set, not a §5.3
deferral, which is now measured rather than argued.

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
