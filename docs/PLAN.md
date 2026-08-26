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

## Status — 2026-08-26 (`/backlog`. Replaced whole, per step 6: the lane table is a fresh `go tool lanestatus` paste, the branch namespace a fresh `wipsurvey` against a fresh `git fetch -p origin main`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh 606-issue page-numbered `state=all` fetch taken after this pass's own writes. **A `/backlog` re-derives the band ordering rather than shifting it**, so every row's rank below is argued from this pass's evidence. This pass follows THREE landings, re-planned and closed **#846** after its park into **#1029** + **#1030**, **REOPENED #809** — closed by a squash body that says in words it does not close it — filed **#1031**–**#1034**, corrected EIGHT open bodies, and folded the **eighth consecutive** persona consultation)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10760 | 15601 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13259 | 2139 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**The table is byte-identical to the 2026-08-25 stamp, and the reason is not
that a landing failed to move a lane — it is that NO LANDING THIS WINDOW TOUCHED
LIBRARY CODE.** Three landings, and `git show --stat` on each says so:

| landing | commit | files | library code? |
|---|---|---|---|
| #1018 | `f50b7e6` | `.claude/commands/develop.md`, `docs/LOG`, `docs/WORKFLOW.md` | none |
| #981 | `0c7ac49` | `docs/LOG`, `docs/ROUTINES.md`, `docs/WORKFLOW.md`, `tools/wipsurvey/*` | `tools/` only |
| #493 | `c978dc4` | `docs/LOG`, `docs/WORKFLOW.md` | none |

The last landing to touch `parser`, `xsd`, `validate` or `value` is **`e491ddb`
(#972), 2026-08-24**. **No conformance measurement was taken by this pass** —
the numbers above are the committed expectations read through `lanestatus`.
`schema`'s and `instance`'s figures therefore carry forward unattributed to
anything in this window, which is correct rather than a bookkeeping gap: nothing
in the window claimed a lane.

**Three consecutive process landings against a flat lane table is the ranking
signal this stamp acts on**, and it cuts against the process queue rather than
for it — see band row 3.

### The three landings this pass follows

**`f50b7e6` (#1018) landed the post-merge closing check, and it was defeated
thirteen hours later.** `docs/WORKFLOW.md`'s Landing section now carries a
one-`Closes`-sentence-per-issue rule and a **fourth precondition**: after the
merge, extract every `#<N>` the squash body names — bound keyword and plain
mention alike — read each state back over the GitHub channel, and close by hand
whatever reads `open`. The check is correct and it is one-directional; #1034
below is the case it cannot see.

**`0c7ac49` (#981) gave `wipsurvey` the heartbeat rule and a `LEASE AGE`
column.** An empty `wip/` claim is dated only from the newest `RESUME:` or
`TAKEOVER:` comment, never from a `GROUNDING:`, a verdict or a `MASON:`
account; `ghIssue.Comments` is a pointer so *absent* and *empty* cannot
collapse; a bare branch pushes `git commit --allow-empty` plus a `RESUME:` at
every checkpoint. The column header in the survey below is `LEASE AGE` because
of this landing, where every prior stamp pasted `TIP AGE`.

**`c978dc4` (#493) closed the park protocol, and this pass is its first
user.** A park now relabels `needs-replan` **and clears `ready`** in the same
clause, and the cartographer *"files the replacement, names it on the parked
thread, and closes the `needs-replan` issue `not_planned` — never `completed`."*
It also reopened and re-closed **#256** and **#271** `not_planned`, whose
`completed` reason had made two historical parks indistinguishable from landed
work in every issue search. Neither re-close moves a count: both were already
closed.

**All three used one `Closes #<N>` sentence and closed exactly one intended
issue each.** The comma-form workaround held for the fourth consecutive window.
What did not hold is the *other* direction — see the reopen below.

### #809 was closed by a body that says it does not close it — REOPENED

**The single most consequential reconciliation this pass made, and no standing
sweep would have found it.** PR **#1026** (`d13d8d1`, the #981 post-land pass)
carries this block in its squash body:

```
Issues this body names and does NOT close:
  #809  open, ready — body rewritten with the live reproduction
  #805 #779  open, ready — re-read, unchanged
  #857  open, ready — carries this pass's REST body round-trip datum
```

GitHub's closing-keyword parser matched `close:` and bound it across the line
break to the reference that follows. **#809 closed `completed` at `d13d8d1`**,
actor `claude[bot]`, 2026-08-26T10:15:15Z — three seconds after a merge whose
own body says twice that it does not close it (*"Not a new issue: #809 owns
it."*).

**The single-binding rule is what proves the mechanism**, and it is the same
fact #1018 landed for the comma form: a keyword binds exactly ONE reference. Of
the seven numbers under that heading, read back this pass, **#805, #779, #857,
#802 and #1014 are all open and only the first — #809 — is closed.** That
signature admits no other reading.

**Nothing was fixed.** `tools/wipsurvey/main.go`'s last two touching commits are
`0c7ac49` and `246c9dd`; neither changes `gitAncestry`, where #809's defect
lives. #809 is reopened `ready`, its body gained a fourth Acceptance item, and
the mechanism is filed as **#1034**.

**#1018's precondition 4 landed thirteen hours before this closure and could
not fire**, because it remediates only the `open` direction. #1026's body listed
*"#809 open, ready"* — an **assertion of intent**, where precondition 4 asks for
a post-merge read-back. That distinction is the whole of #1034.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (re-run against a fresh `git fetch -p origin
main` at `9a8f3ef` and the 606-issue `state=all` fetch, after #846 was closed):

```
ISSUE  BRANCH         LEASE AGE  VERDICT  REASON
732    wip/issue-732  unknown    RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  unknown    RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846  19h41m0s   RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872  unknown    RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  141h34m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  80h39m0s   RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993  unknown    RETIRED  wip/issue-993: issue #993 is closed
```

**One ref arrived and none left: seven now, all RETIRED.** `wip/issue-846` is
the new row — created 2026-08-25, carrying two real commits (`3d33d0b`,
`ad036ed`), parked `needs-replan` on a second arbiter rejection and closed
`not_planned` by this pass. Its REASON reads *"is closed"* rather than *"is
labelled needs-replan"* because `wipsurvey` tests the closed state first; both
verdicts are RETIRED and the change is cosmetic. **No branch is CLAIMED, none is
EXPIRED, no `parked/*` ref exists, and nothing is UNKNOWN**, so **every band row
below is takeable by the next session that reads this table** — fourth
consecutive stamp. The second, comment-enriched pass `docs/ROUTINES.md` spells
under *"Dating an empty claim"* was not run and was not owed: it exists to date
CLAIMED rows, and there are none.

**A row's LEASE AGE changed twice inside this session with the ref unchanged,
and that is a NEW symptom of #809 rather than the tool working.** `wipsurvey`
ran three times today against these same seven refs. `wip/issue-968` read
**`main's`** on the first run and **`80h33m`/`80h39m`** on the second and third
— same ref, same tool binary, same issue state. The only variable was the local
object store: this session ran `git fetch -p origin main` (main moved
`c978dc4` → `9a8f3ef`) and one `git fetch --depth=3 origin wip/issue-846` in
between. `wip/issue-968`'s tip `5304863` is *"meta: post-land pass for #950 —
PLAN Status replaced…"*, a `main` squash, so the branch has **no commits of its
own** and `main's` was the correct answer; `merge-base --is-ancestor 5304863
origin/main` now exits **1** here (`git rev-parse --is-shallow-repository` →
`true`). **So the defect is not "shallow clones misclassify" — it is that the
misclassification is a function of WHEN IN THE SESSION THE LAST FETCH HAPPENED**,
because a shallow fetch re-anchors the graft. This section pastes that column
verbatim every pass, so two stamps can publish different values for an unchanged
namespace and neither is wrong about what it ran. Recorded on #809 as its fourth
Acceptance item.

**Three TIP AGE cells still read `unknown`, down from four**, and that is the
same environment premise from the other side: `6334ffc7`, `cc2d54e6` and
`0b34c21a` are not in this container's object store, while `ad036ed` now IS —
because this session fetched `wip/issue-846` at `--depth=3` to verify the
parked branch's content before re-planning it. `wipsurvey` tests RETIRED
**before** it tests for an unfetched tip, so all seven verdicts stand on issue
state rather than on a borrowed age.

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON —
141 issues, 50 of them open; both counts unchanged from the last stamp, since
nothing `kind/gap` opened or closed this window):

```
gapaudit: 66 GAP marker(s) across 6 area(s)

=== Per-area census ===
  parser           2
  validate         14
  value            3
  xml              4
  xpath            6
  xsd              37

=== Group 1: markers with no OPEN tracking issue found ===
(matching is heuristic — file path or a distinctive phrase; treat a
listing here as "needs a human look", not as proven untracked)
(none)

=== Group 2: OPEN kind/gap issues with no surviving marker ===
(a stale tracker if the gap was a marked fail-open site — but
kind/gap also labels conformance-lane gaps, which never carry a
marker and belong here permanently)
  #398 cmd: the notImplemented stub is untracked P3 debt (no GAP(cmd) marker), and TestUsageCoversContract guards four substrings of a whole-block coupling
  #404 conformance: closureScan.unresolved is scan-scoped, so the decline conjunction is coarser than the hazard it models
  #569 parser: the ROOT half of the pre-scan's `<annotation>` exclusion is unpinned — a top-level `<xs:annotation><xs:appinfo><xs:key name="…">` was falsely indexed before #384 and is correctly excluded after, in both cases with no test
  #591 conformance: an instanceTest caseSpec DROPS its sibling schemaTest <schemaDocument>, so readFacetsCase declines time_minInclusive006_1163.i for a schema the catalog names explicitly
  #592 conformance: string_pattern002_1031.i — a <list itemType> over a USER-DEFINED item type in a target namespace, with a multi-leaf instance shape no cohort reader matches
  #593 conformance: decimal_totalDigits004_1060.v carries its tested value on the ROOT attribute, a shape readFacetsCase exactly-one-<foo> guard declines
  #719 validate: cvc-assertion wired fail-open at every variety level — the M6 seam, marked and measured
  #787 value: an enumeration member outside the base's value space is charged enumeration-valid-restriction (§4.3.5.5) at schema construction and never src-enumeration-value (§4.3.5.3) — which rule the spec assigns is ungrounded, and restriction.go's wrap shadows newEnumFacet's remap
```

**The census is unchanged — 66 markers, 6 areas, both groups identical to the
last stamp — which follows from a window that landed no library code.**
**Neither group is a complete list**, and #852 is why: its file-path false
negative hides #1005 from both groups, and its five-word-window false positive
silently empties a group-2 row. Group 1's `(none)` is therefore *"nothing the
heuristic could see"*, not *"nothing untracked"*. **Every `/backlog` pays this
paragraph**, which is #852's whole argument and why it is banded.

### Milestones and queue

Both columns and the queue below are RE-DERIVED this pass from one page-numbered
`state=all` fetch — 11 pages, **1034 items, 606 issues** once the 428 pull
requests are dropped — taken after this pass's own writes, not from the
milestones endpoint.
`gh api --paginate` is still unusable (numeric-ID repository paths in the Link
header, 403 from the proxy, after writing the pages it did fetch), and
`gh issue list` is unusable for a different reason: it is a GraphQL path.
Repository-scoped REST served **every** read and write this pass made —
**29 writes**: six issue creations, nine body PATCHes, one close, one reopen and
twelve comments, plus three full eleven-page listings and every body and comment
thread read behind them. `docs/ROUTINES.md` is accurate on all three.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 98 | 45 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**No milestone row moved, across three landings and six filings — the fifth
consecutive stamp with a flat table.** None of #1018, #981 or #493 carries a
milestone, and neither do #1023, #1029, #1030, #1031, #1032, #1033 or #1034.
**175 of the 233 open issues carry no milestone** (233 − 45 − 13), so the rows
above are feature progress and the paragraph below is the queue. M4's open count
has now stood at 45 for five passes, against the 42 the 2026-08-16 steward audit
recorded.

Queue: **233 open — 212 `ready`, 21 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 21), against **373 closed**.
212 + 21 + 0 = 233 exactly, and **every one of the 233 carries exactly one queue
label** — checked mechanically, not by eye. **The hygiene sweep is clean on all
four classes this stamp**: zero open issues with no queue label, zero with two,
zero with no `area/` label, and zero with no `kind/` label. The class #773/#774
fell into stays empty for the eighteenth consecutive stamp.

**Every move reconciles.** From the previous stamp's 230 open / 210 `ready` /
20 `blocked` / 0 `needs-replan` / 369 closed:

| move | open | `ready` | `blocked` | `needs-replan` | closed |
|---|---:|---:|---:|---:|---:|
| previous stamp | 230 | 210 | 20 | 0 | 369 |
| #1018 landed at `f50b7e6` | −1 | −1 | | | +1 |
| #1023 filed `ready` (#1018's post-land pass) | +1 | +1 | | | |
| #846 parked `needs-replan`, `ready` cleared | | −1 | | +1 | |
| #981 landed at `0c7ac49` | −1 | −1 | | | +1 |
| #809 closed by `d13d8d1`'s keyword accident | −1 | −1 | | | +1 |
| #493 landed at `c978dc4` | −1 | −1 | | | +1 |
| **#809 REOPENED by this pass** | +1 | +1 | | | −1 |
| **#846 closed `not_planned` by this pass** | −1 | | | −1 | +1 |
| **#1029 filed `ready`** (#846's replacement) | +1 | +1 | | | |
| **#1030 filed `blocked`** (#846's remainder) | +1 | | +1 | | |
| **#1031, #1032, #1033, #1034 filed `ready`** | +4 | +4 | | | |
| **this stamp** | **233** | **212** | **21** | **0** | **373** |

#256 and #271 were reopened and re-closed `not_planned` by #493's landing; both
were already closed, so neither moves a count. **The `closed` column falls by
one for the first time in this project's recorded stamps**, and the reopen is
why.

**Five `ready` issues are STILL done, and the count above still overstates what
is startable by five. This is the FIFTH stamp carrying it.** **#625**, **#748**,
**#492**, **#934** and **#896** were discharged by `34a8043` and judged with it,
and all five remain open because GitHub bound `Closes #669, #625, …` to the one
reference following the keyword. **#1023** now tracks them as a filed issue
rather than as a `docs/PLAN.md` line — filed 2026-08-26 by #1018's post-land
pass, after the item was carried six times in prose. Each of the five also
carries a comment naming the landing and telling a session not to take it.
**They are not closed here**: the cartographer files, unblocks and restamps, and
never closes an issue as done.

**The unblock sweep relabelled nothing, and the zero is measured.** All 227 open
bodies were fetched over `gh api` — byte-faithful, where MCP `issue_read` is
lossy (#764) and, per **#1014**, silently truncating — every `## Depends on`
section was split out, and **no open issue names #1018, #981, #809, #493, #846,
#256 or #271 in a dependency position.** Independently: **every one of the 227
open bodies carries a `## Depends on` section**, so the sweep has no blind spot
from a missing heading. The only `blocked` body naming a landed issue is
**#1002**, whose `## Depends on` cites #1001 as discharged in writing; it stays
`blocked` on a human ruling and nothing else. #493's own post-land pass
(`9a8f3ef`) had already run this sweep to the same zero.

**Eight open bodies were corrected rather than commented at**, per
`docs/WORKFLOW.md`'s filing discipline:

- **#16** — its `## Notes` "Current state of the binary" paragraph was dated
  `ecf3d79` (2026-08-03) and said every subcommand *"hits the `notImplemented`
  stub, prints `goxsd8: not yet implemented — …`"*. **That single-message claim
  has been false since `abe781a`**: #514 split it into three distinct
  diagnoses. Re-measured at `c978dc4` and now naming all three plus the shared
  `helpPointer` line, with a new paragraph stating why the eighth
  consultation's *"close it"* recommendation is refused — this body's own
  Acceptance says the issue is done only when every criterion is lifted into a
  **closed subcommand issue**, and the whole `gen` block is still carried here
  against none. `issuecomment-5426867179`.
- **#472** — its Acceptance argued the `-q parse` trap *"is honest today,
  because no grammar is stated"*, while its own `## Spec` section cites
  `doc.go:33-56` as the stated vocabulary. The trap still reproduces bit for
  bit, but the behaviour is disclosed in **both** contracts, so the bullet now
  frames what survives as a forward-looking UX decision. `issuecomment-5426856348`.
- **#809** — a fourth Acceptance item, the re-fetch instability measured above.
- **#812** — its `## Notes` promised to fix `xsd/identityconstraint.go`'s stale
  premise *"in the same landing"*; that bullet named one of **three**
  contradicting `go doc` pages and bound a doc-only fix to a design question
  this issue's own `## Surface` calls *"most of this issue"*. It now defers to
  **#1032**, and states that the `c-selector-xpath`/`c-fields-xpaths` half of
  the stale sentence is still true and still this issue's.
  `issuecomment-5426876440`.
- **#1003** — the XPath bullet's three named capabilities, broken down against
  the tree into shipped-and-partial (CTA), unstarted (assertions) and
  wrong-package (identity constraints). `issuecomment-5426826998`.
- **#1004** — the `-race` measurement, added as evidence and explicitly **not**
  as discharge of the Acceptance. `issuecomment-5426815604`.
- **#1005** — two stale citations: `gapaudit` reports 66/6 where the body said
  64/5, and the sentence moved from `parser/doc.go:255-258` to `:287-289`. The
  citation is now the quoted text plus the enclosing paragraph, not the line.
- **#1007** — the concrete `gen` acceptance criterion and the CI-script cost.
  `issuecomment-5426844434`.

### The #846 re-plan — the pass's largest act

**#846 is closed `not_planned` and replaced by TWO issues, and the split is the
whole content of the re-plan.** It parked `needs-replan` on 2026-08-25 at the
two-rejection cap (PRINCIPLES 30); this pass is the first to run the park
protocol `c978dc4` landed that same morning.

**Both rejections were about comment text and neither touched the mechanism.**
The gate ran green all four parts on both rounds, `go tool surface -base
origin/main` read **+8 −0** both times, and `Ratchet: unchanged` was verified
twice with `Improved`/`Regressed`/`Vanished` all zero over the full `-v` log.
Round 1 found three doc-comment accuracy defects, **all in the false-accept
direction**; the repair round fixed all three plus a fourth it found itself, and
correctly overturned the arbiter's own round-1 framing of the `<annotation>`
question against §3.15.1/§3.15.2 versus §3.15.4 — sustained on re-verification.
Round 2 rejected on a **fifth instance of the same false claim**, in `run`'s
`case "annotation", "defaultOpenContent":` arm, contradicting `topLevelMapped`'s
corrected comment eleven lines away.

| replacement | scope | label |
|---|---|---|
| **#1029** | step 1 — resume `ad036ed`, fix the one surviving comment, give the non-XSD-namespace child a `GAP(parser)` marker, tighten the §3.15.1 citation | `ready` |
| **#1030** | steps 2–3 — widen the census region by region, then delete `schemaShapeDecidable` and its seventeen predicates in one commit | `blocked` on #1029 |

**Why split rather than carry all three stages forward under one number.** #846
held the whole programme on the warden's staging, and **stage 1 alone consumed
a full session, two mason rounds and two arbiter rounds and landed nothing.**
Repeating that shape would put the 887-line deletion behind the same undivided
ticket that just failed to deliver a 507-line slice. The halves also differ in
kind: #1029 is a bounded repair over a diff two arbiters have judged sound, and
#1030 is the measured win.

**The measurement carries to #1030, where the deletion lives**: at `e491ddb`,
`schemaShapeDecidable` plus **seventeen** `*Decidable` predicates, **887 lines**
of `conformance/schema.go`, against the 2026-08-16 audit's fifteen and 667 —
two predicates and ~220 lines of growth in nine days. #846's title said
"~700-line" and understated it by about a quarter.

**`needs-replan` was retained on the closed #846 rather than cleared**, on the
#968 and #993 precedent: the label is what retires the branch in the survey, and
clearing it changes only the stated REASON. `wip/issue-846` stays as re-planning
evidence at `ad036ed`; the branch scheme forbids re-attempting an issue under
its own number, so #1029's fresh attempt starts as `wip/issue-1029` — **from
`ad036ed`'s content, not from scratch**, and `origin/main` has moved six commits
past that branch point, so a merge forward and a full re-gate are owed.

### #933 is not a park — the counter-example the last stamp published is withdrawn

The 2026-08-25 band's row 4 cited **#933** as a counter-example correcting
#493's own body (*"Every park to date has cleared `ready`"*). **#493's post-land
pass (`9a8f3ef`) dissolved it, and the ruling is right**: #933 never entered the
park path at all. No implementation attempt, no arbiter round, no `needs-replan`
ever owed — a develop-loop grounding pass closed it as a duplicate of #862,
which has since landed at `109beb9`. `wip/issue-933` exists but carries no
commits of its own, so the ref is a bare claim and nothing else.

Its residual `ready` therefore needs no ruling: **this repo does not strip queue
labels at close**, and #862, #972, #1001 and #843 all wear one while closed
`completed`. This pass posted a ruling framed as a re-plan judgment before
reading `9a8f3ef`, and corrected itself on the thread; the two comments crossed
in flight and the outcome — leave the label — is the same either way.

### Persona consultations — the EIGHTH consecutive, and the first to file THREE

Run by the orchestrating session, never by the cartographer (#416): each persona
saw only the README, `go doc` output and, for cliuser, the built binary. **Three
new issues filed and five reconfirmed**, against a seventh consultation that
filed nothing.

**libuser's headline is a three-way contradiction, and it is the first persona
finding this project has that was settled by RUNNING the library.** `go doc
xsd.IdentityConstraint` says identity-constraint evaluation *"is deferred to the
M6+ XPath engine and is out of scope"*; `xpath/doc.go` says the package serves
*"identity-constraint paths"*; `validate/doc.go` says they are evaluated per
§3.11.6.2/3 *"directly and never through the XPath engine"*. The persona built a
schema and an instance and drove a real `cvc-identity-constraint` violation out
of the README's own pipeline, which settles it: `validate` is right, `xsd`'s
comment predates #718, and `xpath`'s topic sentence overclaims a capability that
belongs to a sibling package **by design**. Filed as **#1032** — and *not* as a
duplicate of #812, whose Notes owned one of the three sites; #812's bullet is
corrected to defer.

**libuser's second: `builtin/doc.go` says both 20 and ~25.** Four sites say the
19 §3.3 primitives plus `precisionDecimal`, `typespec_test.go` asserts
`IsPrimitive()` counts exactly 20 and `len(Types) == 49`, and the persona ran
`Seed` and got 51 components (49 rows plus `anySimpleType` and `error`) —
consistent with 20 and with nothing near 25. The one dissenting sentence is in
the *"Mapping resolution: nearest mapped ancestor"* section, which is the one
paragraph a **backend author** reads for the mandatory floor of the mapping
walk. Filed as **#1033**.

**cliuser's new finding: `-format` has no enumerated vocabulary and no stated
scope**, where the sibling `-backend strict|native` one line below enumerates in
both `doc.go:19` and `README.md:76`. Neither page says whether the value is
`xml`, `.xml` or `XML`, nor whether one `-format` covers every instance of a
`validate` invocation — and the contract's own example names two instances of
two different formats. Undocumented, the only safe script is one invocation per
file, which defeats single-run, single-exit-code batch validation. Filed as
**#1031** on #1007's precedent: the blurb with the hole has no owner before its
milestone.

**cliuser reported #16 RESOLVED and this pass refused the close.** Every error
path — no subcommand, unknown subcommand including case variants, and *"not yet
implemented"* for all three reserved names — carries the identical onward-pointer
second line, with no residual across `-help=true`, `-h=true`, `-parse`, `--` and
`-1`. Verified structurally: `main.go` prints `diagnose(args[0])` and
`helpPointer` through **one** `Fprintf`, so the pointer cannot drift between
branches. But #16's own Acceptance says it is done only when every criterion is
lifted into a **closed subcommand issue**, and the whole `gen` block is still
carried there against none.

**#472 re-triaged, #1007 sharpened, #1004 measured, #1003 broken down, #1005
reconfirmed** — all folded into their bodies above. The README library snippet
built and ran with zero edits for the **third consecutive stamp**; nothing in
this repo compiles README snippets, so the next stamp should say whether it
still holds.

**Every sharper argument was banked on its thread**, not only here, because
`docs/PLAN.md` is replaced each pass and an issue thread is not:
`issuecomment-5426815604` on #1004, `issuecomment-5426826998` on #1003,
`issuecomment-5426844434` on #1007, `issuecomment-5426856348` on #472,
`issuecomment-5426867179` on #16, `issuecomment-5426876440` on #812.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
of it. Take from the top. Cross-references are stated by ISSUE rather than by row
number, which decays at each re-cut. **This is a `/backlog` re-cut: every rank is
re-derived from this pass's evidence.** Nothing in the band is claimed and
nothing in the `wip/*` namespace is live.

**All four of the last stamp's top four rows left the band** — #846 parked and
re-planned, #1018, #981 and #493 landed — so this band is cut from row 5 down
plus this pass's own filings, and no rank below is inherited.

| # | Issue | Why here |
|---:|---|---|
| 1 | #1029 | **The only thing that can move the steward's last standing `Increasing` ranking, and the cheapest landing in the band by a distance.** `wip/issue-846` at `ad036ed` already carries the whole slice — 8 files, +507/−15 — and both arbiter rounds found the CODE sound: gate green all four parts twice, `surface -base origin/main` +8 −0 twice, `Ratchet: unchanged` verified twice with Improved/Regressed/Vanished all zero. What it owes is **one comment**, one `GAP(parser)` marker and one citation tighten. Its cost strictly INCREASES with delay and nothing else in the band does: `origin/main` has already moved six commits past the branch point, and every further landing widens the merge forward that a re-attempt must gate through. It gates **#1030**, where the 887-line deletion and the measured ranking live — seventeen `*Decidable` predicates at `e491ddb` against the 2026-08-16 audit's fifteen, ~220 lines of growth in nine days, re-derived and not copied. **Do not restart from scratch**; the park comment and #1029's Acceptance both point at `git diff origin/main...ad036ed` |
| 2 | #1034 | **Reproduced on `main` TODAY, defeating a check that landed thirteen hours earlier, with a failure mode no standing sweep catches.** A squash body's *"Issues this body names and does NOT close:"* heading closed **#809** — GitHub bound the keyword across the line break — and this pass found it only by reading a commit body against the issue list, then reopened the issue by hand. **The proof is which issues did NOT close**: of the seven under that heading, #805, #779, #857, #802 and #1014 are all open and only the first is closed, which is the single-binding rule #1018 landed for the comma form. #1018's precondition 4 exists and could not fire, because it remediates only the `open` direction. **Fourth event in the closing-keyword family in four days** (#993 → #1018 → #1023 → this), and the first whose damage is an issue silently leaving the queue rather than silently staying in it. One session, doc-only, two sentences — and the spelling half (*"names and leaves open"*, which contains no keyword) removes the hazard rather than detecting it. Its Acceptance is a fixture: a rule that does not flag `d13d8d1`'s real squash body has not been tested |
| 3 | #719 | **PROMOTED two rows, and the promotion argument is the lane table above rather than a fresh sighting.** The table is byte-identical for the second consecutive stamp, and the reason is that **no landing this window touched library code at all** — #1018 and #493 are `.md` only, #981 is `tools/` — with the last `parser`/`xsd`/`validate` landing being `e491ddb` on 2026-08-24. Three consecutive process landings against a flat lane table is exactly the starvation the steward rule exists to prevent, running in the opposite direction, and this band answers it here. #719 is the band's only row that can move `instance`: `cvc-assertion` is wired fail-open at every variety level — the M6 seam, marked and measured, and group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56 alone**, and #56 is the *"was this schema+document combination fully decided"* question a *"reject bad requests, accept the rest"* story cannot be built without. It decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. **It does not rescue #1002**: its own acceptance declines every case whose outcome turns on an assertion, which kills `vc002.n1` too |
| 4 | #963 | **The tax falls on every landing, its discriminator STILL could not fire — three more clean windows, NINE in a row — and it now has a sibling that wants the same tool.** `git show --stat` on `f50b7e6`, `0c7ac49` and `c978dc4` shows `docs/LOG/2026-08.md` at +119, +134 and +82: **carried, three for three**. A long run of non-firings reads like a case for demotion and is not one — #820 landed the *form*, and what is unchecked is that the check was RUN, which is #924's shape and is not reachable by prose. **What changed this pass is the consolidation argument**: `#1034` asks for a post-merge read-back of every reference a squash body names, and precondition 1 asks whether that same body's landing carried its `docs/LOG` entry. One `tools/landcheck` reading one squash body can hold both. Whichever of this and #1034 lands first should say so on the other's thread. One session, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part (#304). Banded on cost, not on sightings |
| 5 | #852 | **The tool the cartographer runs every pass reports something FALSE, and this pass paid the caveat paragraph for the third consecutive stamp.** Group 1 printed `(none)` above — which means *"nothing the heuristic could see"*, not *"nothing untracked"*, because the file-path false negative hides #1005 from both groups and defect 4's five-word-window match (`parser/conditional.go` against **#921** on `claude md s one rule`, pure house boilerplate) silently EMPTIES a group-2 row. Defect 3's phrase collisions surface as a visible `dead end:` line a reader can check; the suppression has nothing to check. **No group of this report is a complete list**, and every `/backlog` writes that caveat down. Defect 1's citation-first matcher does **not** cover it — neither text carries the other's number — so group 2 needs its own answer. No new witness this pass and no decay either |
| 6 | #1032 | **The first persona-family row this project has banded, and it is banded because a reader is told a SHIPPED feature does not exist.** Three `go doc` pages give three answers on identity-constraint evaluation, and libuser settled which is right by *running the library* — a real `cvc-identity-constraint` violation out of the README's own pipeline. A consumer who starts at `xsd.IdentityConstraint`, the natural entry point, is told the feature is deferred to M6+; one who starts at `xpath` is sent to the wrong package; only `validate/doc.go`, 190 lines in, is correct. **The family is the argument for banding one of them**: eight persona doc issues are open (#1003–#1007, #1031–#1033), the eighth consultation added three, and none has ever been taken. Doc-only, three files, one session. Its `c-selector-xpath` half stays with #812, whose body now says so |
| 7 | #1004 | **The only member of the persona family whose answer had to be MEASURED, and this pass is where the measurement arrived.** libuser shared one `*xsd.Schema` and one `*validate.Validator` across 50 goroutines under `-race` and the detector reported nothing — so the sentence the issue asks for is probably *"safe for concurrent use after construction"* and there is now evidence for it. **The measurement does not discharge the acceptance and the body now says so**: it lived in a scratch module and died with the session, and a clean `-race` run only exercises the paths those goroutines took. What remains is bounded — read `validate.Validator`'s fields and `value.Backend`'s implementations, write the sentence, commit the test. Ranked below #1032 because a service author can still ship without the sentence; a reader misled about identity constraints cannot |
| 8 | #844 | **The second `Increasing` steward ranking in the queue, and the consumer it names is now three rows above it instead of two.** `dispatchUnion` computes a union's ·active member· and discards it, while ·validating type· is already needed by `cvc-id` and identity-constraint key members — and its named upcoming consumers are `cvc-assertion` (**#719**) and CTA. Ranked below #1029 because its ranking is argued from future consumers where #1029's chain is measured on the record, and below the process rows because none of its consumers has landed yet. **Worth reading beside #719**: a session taking #719 should say on this thread whether the seam it lands makes this a smaller job |
| 9 | #941 | **Its body was repaired by the last pass and a session still starts from a correct table.** `684b2b4` rewrote `Attributes`' body from `return e.src.Attributes()` to `return e.attrs`, killing Acceptance 2's stated reason while leaving its conclusion intact. Both methods still fail STYLE T5 on both prongs — `parser/tree.go:31` still says so in the tree — and `surface: +0 -2`, strictly shrinking. **Acceptance 6a is the trap to read first**: `parser/conditional.go` reads and writes `e.attrs` directly at five sites, so `e.attrs` is the schema document and `e.src.Attributes()` is §4.2.2's pre-processing INPUT. **Warden pre-flight required.** Note the ordering #1029/#1030 imposes on its neighbours: `Element.Name`/`Attr`/`Children`, `Node` and `Text` cannot be unexported until the shadow model is deleted, which is #1030's job, not this one's |
| 10 | #975 | **Displaced rather than devalued, for a third stamp, and it is still the band's certain win.** Nine s4s-grammar rejection messages naming no Appendix A production, against the bar `xsderr/doc.go` set with #966; its #884 ordering is discharged and its 21-site criterion re-ran to the same nine. It gained no evidence this window and rows above it did — but nothing about it decayed either, and it is the first thing to promote if a session wants a bounded mechanical landing rather than a design question |
| 11 | #953 | **A doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says *"every caller turns a decided negative into a schema rejection"*; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules |
| 12 | #853 | **The band's second lane row, and its first step is not code.** `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **Start with an oracle question**: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |

**Left the band this pass, and why.** **#846** — parked at the two-rejection
cap, closed `not_planned`, and its row is inherited by **#1029** at row 1 rather
than vacated; **#1030** holds the value and is `blocked`. **#1018**, **#981**
and **#493** — all three landed, and all three were band rows on the last
stamp, which is the band working rather than a coincidence: four of four top
rows cleared in one window. **#1007** did not enter the band despite being
sharpened this pass — it is a documentation sentence whose cost is a script
author's, and #1031 is now its sibling; take them together or not at all.

**Deliberately unbanded, and why.** **#1030** is `blocked` and belongs to no
band: **#1029** is what moves it, and its `Increasing` steward ranking is
argued at row 1 rather than here. **#1002** is `blocked` and belongs to no band
either: a ruling, not a queue position, is what moves it. **#809** is reopened
and `ready`, and stays below the fold **only because #1034 is what stops the
accident recurring** — the tool defect itself is real, now has a fourth
Acceptance item, and is the thing to take right after #1034 if a session wants
the pair. **#1023** is the five-issue residue tracker and is the first item of
the next planning action rather than a band row, because closing an issue as
done is the develop loop's act. **#1031** and **#1033** are this consultation's
other two filings, both one-sentence doc fixes below the fold. **#1019** names
four labels never used in 606 issues and omits **`area/xsd` at 164 issues**, the
single most-used area label in the repository; a decision plus two edits,
unbanded only because no session has ever been misled by it in a way the log
records. **#1011** is #843's follow-up with **#725** as its live victim.
**#1014** is the MCP `issue_read` truncation; it is below the fold only because
`gh api` is the documented byte-faithful path and every read and write in this
stamp used it, 25 calls without a failure. **#1003**, **#1005**, **#1006** and
**#1007** are the sixth consultation's surviving filings, all reconfirmed by the
eighth. **#999** is #398's sibling and most likely *deletes* a test. **#996**
and **#992** stay unbanded on the discriminator that #992's cost was
paid-and-absorbed. **#989** is #979's post-land filing, decision-first and
cheap. **#894** is clear to proceed and stays unbanded only because it is a
lane-flat correctness fix whose first step is an unsettled oracle question.
**#888** and **#889** still await a suite census in their range. **#907**'s
`childElement` census is stale by at least eleven landings. **#885**'s three
discriminators still have one sighting each. **#409** is `ready` since
2026-08-02 with a fifth independent sighting and stays unbanded only because it
is one row of a five-file convention landing. **#854** did not gain a third
sighting this pass. **#670** asks for `parser/example_test.go` — read it beside
**#1006**. **#937** is naturally folded by the next landing touching
`rejectRepeatedAnnotations`. **#920** and **#921** are conformance-bookkeeping
follow-ups; #921 additionally carries the gapaudit-suppression note. **#929** and
**#931** are the small parser occurrence / rule-mapping gaps #901 exposed.
**#455** is the live owner of the `strings.TrimSpace`-versus-§4.3.6 character
class at **ten** sites, and **#456** stays `blocked` on it. **#845**, **#848**
and **#849** are the 2026-08-16 audit's remaining open findings, all ranked
*stable debt* by the steward and therefore all outside the rule that bands
#1029/#1030 and #844 — though #849's stable ranking is already falsified on its
own thread. **#841** is `blocked` on a trigger that fired without a ruling.
**#566** is #565's open sibling. **#692** and **#925** are still `blocked` on a
`/retro` trigger that fired without ruling on them. **#16** and **#472** were
both corrected this pass and neither moves: #16 is a `blocked` reference issue
that is never worked directly, and #472 is the `goxsd8 parse` implementation
whose position is set by M4's tail, not by a doc bullet. **#570** carries the
standing `schema` decline-count argument at 893 against a re-measured 788, and
it is now **two landings staler** — no conformance measurement was taken here
either.

### Next planning action

1. **Close the five. FIFTH stamp — but for the first time the item is a FILED
   ISSUE rather than a `docs/PLAN.md` line.** #625, #748, #492, #934 and #896
   are discharged by `34a8043` and judged with it; only #669 closed. **#1023**
   was filed 2026-08-26 by #1018's post-land pass after the item had been
   carried six times in prose, which is the disposition rule working. Closing
   them is still the develop loop's act — the cartographer never closes an issue
   as done — and it is one `gh api ... -f state=closed -f state_reason=completed`
   per issue. Until it happens the `ready` count above overstates what is
   startable by five.
2. **The human decision blocking #1002 is unchanged and is now carried for a
   third stamp.** **#1002** waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID with a per-case justification in the verdict, and (b) holding
   §4.2.2's `vc:maxVersion` arm until real assertion evaluation lands. CLAUDE.md
   puts (a) beyond any agent — *"changes only via a human-filed issue"* — and (b)
   depends on M6 work **#56** records as unfiled. **No agent should attempt
   either.** Its `GAP(parser)` marker is live in `parser/conditional.go`; record
   the ruling as a comment on #1002, and that comment is what moves it off
   `blocked`.
3. **Assertion EVALUATION is still the largest unfiled thing this project has**,
   and it still blocks two issues: **#56** (through #719's encoding) and
   **#1002** (route b). M6 XPath growth tier 2 — `$value` binding, an F&O
   function library, typed comparison — too big for one issue. **Carving it is a
   `/backlog` act and it should happen at the M6 opening**, not sooner and not as
   a single ticket. Row 3's promotion this pass makes the carve nearer, not
   further: #719 is what the carve will need an encoding decision from.
4. **The whole band is startable and unclaimed**, for the fourth consecutive
   stamp: all seven `wip/*` refs are RETIRED, none is CLAIMED, none is EXPIRED.
5. **NEW, and the reason row 3 moved: three consecutive landings touched no
   library code.** #1018 and #493 are `.md` only; #981 is `tools/`. The lane
   table has now been byte-identical across two stamps for a reason that is not
   about the lanes. The process queue earned those three landings on
   CLAUDE.md's cost rule and each was worth taking — but a fourth consecutive
   process-only window should be read as a scheduling fact rather than a queue
   fact, and this stamp answers it by ranking the band's only `instance`-moving
   row third rather than fifth.

**The two standing promotion discriminators had three subjects this window; one
did not fire and the other's trigger has now expired.** **#963's** (did the
landing carry its `docs/LOG` entry inside the squash?): `docs/LOG/2026-08.md` is
in all three squashes at +119, +134 and +82 — **carried, ninth consecutive clean
window**, and #963 stays banded on cost. **#846's** (did a landing's entry record
the shadow tax — a `conformance/schema.go` edit mirroring a
`parser/produce_complex.go` one?): **it could not fire, for the seventh
consecutive window**, and the last stamp set a trigger on exactly that count —
*"if a seventh passes without a producer widening that edits both files, the
discriminator itself is the thing to re-examine."* **Re-examined, and the
discriminator is not the problem.** All three landings are `.md` and `tools/`;
none comes within a package of either file. A discriminator that requires a
producer widening cannot fire in a window with no producer landing at all, which
is a fact about the develop loop's last three sessions and not about the
instrument. It carries to **#1030** unchanged, and the next window with a
`parser` landing is the first that can answer it.

**The steward-ranking rule has produced three landings out of four bandings, and
the fourth is now #1030 behind #1029 at row 1.** #979 landed 08-23, #978 and
#843 on 08-24, and #846 became the rule's first parked subject rather than its
fourth landing — **which is not a failure of the rule**: the park was on
comment accuracy over a gate-green diff, and the value the rule identified is
intact and re-filed. **#841 remains the counter-example the rule cannot reach**
— a `kind/refactor` with a steward ranking that stays `blocked` because its
trigger has no mailbox. That gap is on #841's thread and is still not filed as
an issue, because the fix belongs to whichever pass gives Part 2 of the `/retro`
the mailbox Part 1 has.

**Standing, unchanged, and still true.** Four unlanded corrections target one
paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**, **#646**,
**#679**, **#912** — and whichever lands last rebases three times; #992, #1019
and #1034 are not among them, all targeting different paragraphs of the same
file, and #493 landed out of that neighbourhood without touching it. **The next
`/retro` inherits three issues it already owed** — #692, #925 and #841. **The
CTA cohort's 45 banked `instance` failures remain unattributed**, sixteenth
consecutive stamp carrying it. **`gate.yml` runs but is still not a required
status check**, which only the repository owner can change.

**Environment costs stay in the log at one witness each; one gained a second and
one is newly sharpened.** Uncached conformance test binaries hang under the
default sandbox, so conformance runs must be issued unsandboxed — this pass took
no conformance measurement, so it neither corroborated nor weakened it. Four
consecutive arbiter launches died on transient platform errors on 2026-08-24; no
arbiter ran this pass. **The shallow-clone premise gained a second witness of a
NEW kind** — not an unfetched tip but an *unstable* one, `wip/issue-968` reading
`main's` and then `80h39m` inside one session as fetches moved the graft — and
it is recorded on **#809**, which this pass reopened. `gh issue list` still fails
with `HTTP 403: This GraphQL query is not enabled for this session` while
repository-scoped `gh api` REST served **29 writes and every read behind them**
here without one failure, which is exactly what `docs/ROUTINES.md` says. **One environment
observation is new and is not a cost**: a `git fetch --depth=N` of a `wip/*`
branch, taken to inspect a parked attempt, permanently changes what `wipsurvey`
reports for *other* branches in the same session — which is #809's mechanism seen
from the tooling side rather than the survey side.

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
for **`schema` +26**, the family's largest single move. Live ones in the same
family are **#471** (a local `<element ref=>` carrying `substitutionGroup=`,
silently accepted), **#931** (occurrence attributes on a named `<group>`'s child
compositor), **#929** and **#455**. A second, narrower family has opened beside
it: the rejections the producer already makes
**correctly but describes badly**, whose bar `xsderr/doc.go` set with #966 and
whose nine deficient sites **#975** owns. The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes
file against the same packages sit outside it, so the milestone is a floor
on M4's remaining work and not the whole of it.

### M5 — Instance validation (XML) — [epic #250](https://github.com/kud360/goxsd8/issues/250)

`validate` engine plus `validate/xmlsrc`: greedy deterministic matching,
identity constraints, `xsi:type`/`xsi:nil`, wildcards, default and fixed
values. **`instance` lane.**

Carved 2026-08-12 into ten slices, #710–#719, and now **seventeen** — #766 was
split out of #714 by a warden pre-flight, #773, #774 and #775 out of #766 by
another, #782 and #783 out of #715 by its own, and **#790** filed from #715's
warden verdict, which had named the seam and been read by nobody who filed it.
Twelve have landed: the infoset
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
+9, unioned onto #716's). `instance` stands at **10760** — #913's cvc-type
clause 3.1 landing added **9409**, itself M5 and the largest single lane move
this project has recorded — and **twenty-five** of the pre-#913 cases were not
M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure (+12), #821
added 1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). (The
*thirteen* this paragraph carried predated the CTA pair and did not sum; the
five figures do, and the sum is the number.) **It happened again after #913:
#909 — an M4 landing — took `instance` 10746 → 10752 (+6) by producing
`<simpleContent>` `<restriction>`, **#862** — a `conformance` landing — took
it 10752 → 10755 (+3) at `109beb9`, and **#1001** — a `parser` landing — took it
10755 → 10760 (+5) at `684b2b4`, so the outside-M5 total is now 39.** A slice
that produces a component the engine could not previously see, or decides a
`{type table}` it previously withheld, moves `instance` without deciding a new
`cvc-` rule. **#1001 is the first FOURTH-mechanism instance**: it changed neither
the engine nor the harness, but pre-processed the schema document itself —
§4.2.2's ·conditional inclusion· removed declarations that were colliding under
`sch-props-correct` clause 2, so five documents the engine had never been given a
usable schema for became decidable. A lane can move because the schema the engine
was handed was wrong. **#862 is a THIRD mechanism and the paragraph would be wrong to
fold it into the other two**: it changed no engine behaviour at all, only how
`resolveExpected` picks a feature-scoped expected verdict, so the engine's
answers were already right for those three cases and the harness was reading the
wrong expectation. A lane can move because the measurement was fixed.

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
queues for a day. The CLI's own `validate` subcommand is #720, `blocked` behind
#472 alone now that #715 has landed.

**10760 is still a floor built for soundness, and #913's +9409 jump did not
change what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the 15601 that
still fail are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read 10760 as 41% of the suite passing.** It is the count of documents
this engine can honestly call not-valid. It grew most because #913 decided
`cvc-type` clause 3.1 — the commonest simple-typed-leaf shape the lane had
declined outright — which is the counterpart to #790's lesson, not a
contradiction of it: a slice that decides a *new* rule moves the number far MORE
than its rule count suggests when the declined shape is common, and #913 moved
it more than #790's descent did. The number stays a soundness floor, not a pass
rate.

**ONE case is decided and decided WRONG — #771**, a root whose declaring schema
is reachable only through the instance's own `xsi:schemaLocation`. It was four,
and #800 retired two of them: `Assert/assert_019/instance/assert_019_2` and
`CTA/typeAlternatives_001/instance/typeAlternatives_001_2` now decline honestly
instead of rejecting a document the ·conditionally selected· type admits.
**That is two, not three, and `CTA/cta0008.v01` was never among them** — it
takes §3.12.2's inline arm, which #800 deferred to #822 and which **#851 landed
on 2026-08-17** (#822 closed, superseded), and the count was
measured by diffing `GOXSD_DECLINES=1` across the two trees rather than
predicted. All were already banked `fail` before #790 and are not its
regression; the descent is what made them visible, and the trade of a wrong
decision for an honest decline is one the lane cannot register as movement.

**#913 added a second decided-wrong class, so the count is no longer one.**
Seven CTA documents are false-charged through `cvc-type` clause 3.1 until
§3.12.4's `{inherited attributes}` merge lands (#831, #871) — an
honest-decline-to-wrong-decision trade the ratchet's zero-flip-down cannot
register, escalated on #831's thread.

The decline census that separated harvest candidates from indeterminates
predates #766, #715, #740, #790, #718, #716, #813, #913 and now #909 — and is
not re-derived
here. **It is now, by a wide margin, the oldest measurement this milestone still
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

What remains of the CTA subset is **#859** (the wildcard `ta-AttrName` arms; the
live lease this paragraph recorded on 2026-08-18 is long gone — `wipsurvey` shows
no `wip/issue-859` ref, so it is unclaimed), **#888** (a cast target that is
`xs:QName`), **#889** (value-level numeric widening, §B.1 rule 1.1's
float→double promotion, which #858 withheld rather than faked) and **#894**
(err:XPST0051/XPST0080, the static remainder #886 did not charge). **#871** is
the §3.12.4 clause 1.1.3 ·inherited attributes· merge, blocked on M4's #831.
None of the seven carries a milestone, which is the same pattern M4's tail
records.

**Assertion evaluation is still unfiled, and it is this milestone's real body of
work.** **#719** wires `cvc-assertion` fail-open at every variety level today,
one milestone early, because the `instance` lane must decline every case whose
outcome turns on an assertion. **#56** — a fail-open "unevaluated" must never
read as a genuine PASS — now depends on **#719 alone**: #842 was struck from its
`## Depends on` when it landed, because it settled the *evaluator* side of the
encoding (a compile-time `(CTATest, bool)`, `ok` false being the withhold, with
`Evaluate` always deciding) and left only the consumer side, which is #719's.
One encoding, decided in #719 and reused here (STYLE D4). STYLE 9's fail-open
discipline is only honest if a fail-open answer is distinguishable from a real
pass.

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
