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

## Status — 2026-08-24 (`/backlog`. Replaced whole, per step 6: the lane table is a fresh `go tool lanestatus` paste, the branch namespace a fresh `wipsurvey` against a fresh `git fetch -p origin main`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh 595-issue page-numbered `state=all` fetch taken after this pass's own filings. **A `/backlog` re-derives the band ordering rather than shifting it**, so every row's rank below is argued from this pass's evidence, not carried. This pass also re-planned **#732**, closed it as superseded, and folded the **sixth consecutive** persona consultation)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10755 | 15606 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13225 | 2173 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**No lane moved, and the table is byte-identical to the previous stamp for the
third consecutive pass** — carried on a re-run, not on an inference. The one
landing between the stamps is the **#870 bundle** (`abe781a`), whose whole diff
is `README.md`, `cmd/goxsd8/{doc.go,main.go,main_test.go}` and `docs/LOG`: no
conformance lane observes `cmd`, and the accepting verdict measured
`Ratchet: unchanged`. The last movement remains `instance` +3 at `109beb9`
(#862, 2026-08-23). The `instance` lane's indeterminate census stays at 5.

### The landing this pass follows

**`abe781a` (PR #998) closed FIVE issues in one squash and made the CLI
reachable from its own documentation.** `diagnose` (`cmd/goxsd8/main.go:98-106`)
tells three truths apart — a reserved-but-unbuilt subcommand, a name outside the
vocabulary, and a leading flag — each with its own message and the same
`helpPointer`; `doc.go` gained a `# Argument vocabulary` section stating
case-sensitivity, the three help spellings, `--`, and the decline of a version
entry point; `usage` gained the implementation-status paragraph; the Quickstart
gained `go install ./cmd/goxsd8`; and the remedy pointer stopped naming `go doc`,
which fails outside the module tree where an installed binary runs. Arbiter
**ACCEPT round 1**, zero blocking findings, `Ratchet: unchanged`,
`surface: unchanged` (`issuecomment-5396032329` on #870). **Fifty-nine minutes
from grounding to verdict for a bundle that had been banded five passes** — one
oracle round bought a negative (no clause in any of the five local specs governs
CLI mechanics), and that negative is what let five issues share one grounding.

**All five closed, and that is a measurement about #993 rather than a formality.**
The squash body wrote `Closes #870.` … `Closes #672.` — one sentence per issue,
never the comma form — after PR #991 had written the comma form one day earlier
and closed one issue of six. The closed count moved 359 → 364. The workaround
works; it lives in a LOG entry and not in `docs/WORKFLOW.md`, which is why #993
is banded at row 3 rather than discharged.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (re-run this pass against a fresh
`git fetch -p origin main` and the 595-issue `state=all` fetch, after #732 was
closed):

```
ISSUE  BRANCH         TIP AGE   VERDICT  REASON
732    wip/issue-732  unknown   RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  unknown   RETIRED  wip/issue-822: issue #822 is closed
872    wip/issue-872  unknown   RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  93h32m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  32h37m0s  RETIRED  wip/issue-968: issue #968 is closed
```

**Nothing arrived and nothing left: five refs, all RETIRED, and for the first
time all five for the same reason.** `wip/issue-732`'s REASON changed from
*"issue #732 is labelled needs-replan"* to *"issue #732 is closed"* because this
pass closed it as superseded — the re-plan below. **No branch is CLAIMED, none is
EXPIRED, no `parked/*` ref exists, and nothing is UNKNOWN**, so **every band row
below is takeable by the next session that reads this table**. `wip/issue-998`
never existed as a survey row: a landing branch is auto-deleted at merge.

**Three TIP AGE cells read `unknown`, and that is the tool working, not
failing.** This container's checkout is shallow (`git rev-parse
--is-shallow-repository` → `true`) and `6334ffc7`, `cc2d54e6` and `0b34c21a` are
not in its object store — `git log -1` on each answers `fatal: bad object`.
`wipsurvey` tests RETIRED **before** it tests for an unfetched tip, so all five
verdicts stand on issue state rather than on a borrowed age, and an unfetched tip
on a live issue would have read `UNKNOWN` with *"tip not fetched"* rather than
inventing an EXPIRED. **This is a fresh witness for the environment premise
#809 rests on** (a shallow clone makes `wip/*` tips unreadable locally), recorded
on that thread; it is not a new defect and nothing was filed.

**#981 gained no sighting this pass, for the second consecutive pass and for the
same reason**: the empty-claim lease rule only mis-serves a CLAIMED row, and
there is none. It stays banded on the three sightings it has.

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON —
141 issues, 52 of them open, the same input shape as the previous stamp; the
count moved because this pass filed three `kind/gap` issues):

```
gapaudit: 64 GAP marker(s) across 5 area(s)

=== Per-area census ===
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
  #921 conformance: <current status="queried"> is unmodeled, so the two gMonth XSD-1.0-lexical cases are permanently unbankable disagreements with no owner and no stated reason
  #1002 parser: §4.2.2's vc:maxVersion arm — the half of #732 that cannot land until the repo owner rules on VC/vc002/instance/vc002.n1.xml
```

**64 stands for the third consecutive pass, and this pass re-ran the tool against
a `.go` landing.** `abe781a` touched only `cmd/goxsd8`, which holds no `GAP(`
marker at all (`grep -rn "GAP(" cmd/` is empty — the hole **#398** exists to
close), so the census could not have moved and did not.

**Group 2 gained exactly one row, #1002, and it belongs there**: the marker it
will retire does not exist yet — **#1001** creates it. **#1001 and #1005 are
`kind/gap` issues that are equally markerless and appear in NEITHER group**,
which is the file-path matcher artefact **#852** owns in its false-negative
direction: both bodies cite `parser/doc.go`, and a path cited for any reason
matches every marker in that file. So Group 2 is not a complete list of
markerless trackers this pass, and Group 1's emptiness stays qualified for the
same reason. **#960** still owns the class the census structurally cannot see —
a fail-open disclosed in PROSE with no `GAP(` marker — and **#1005**, filed this
pass, is a fresh named instance of it: `parser/doc.go:255-258` promises
multi-error collection as PLANNED, with no marker and, until today, no owner.

### Milestones and queue

Both columns and the queue below are RE-DERIVED this pass from one page-numbered
`state=all` fetch — 11 pages, **1008 items, 595 issues** once pull requests are
dropped — taken after this pass's own filings, not from the milestones endpoint. `gh api
--paginate` is still unusable — its Link header carries numeric-ID repository
paths the proxy answers 403 to, after writing the pages it did fetch — so the
count was raised until page 11 came back short.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 98 | 45 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**No milestone row moved, across a five-issue landing and eight filings.** All
five issues `abe781a` closed are `area/cmd` and carried no milestone, and the
seven issues filed this pass carry none either — deliberately, matching #732 and
#972, which the two §4.2.2 replacements inherit from. **172 of the 230 open
issues carry no milestone** (230 − 45 − 13), so the rows above are feature
progress and the paragraph below is the queue.

Queue: **230 open — 209 `ready`, 21 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 21), against **365 closed**.
209 + 21 + 0 = 230 exactly, and **every one of the 230 carries exactly one queue
label** — checked mechanically, not by eye: the only two open issues wearing a
second are #79 and #250, which wear `blocked` **and** `epic`, the one
combination the label list treats as a pair. The class #773/#774 fell into is
empty for the sixteenth consecutive stamp, and the double-label class #968
warned about is empty too.

**`needs-replan` is zero for the first time since 2026-08-23**, because #732 was
the only one and this pass retired it.

**Every move reconciles.** From the previous stamp's 227 open / 206 `ready` /
20 `blocked` / 1 `needs-replan` / 359 closed:

| move | open | `ready` | `blocked` | `needs-replan` | closed |
|---|---:|---:|---:|---:|---:|
| previous stamp | 227 | 206 | 20 | 1 | 359 |
| `abe781a` closed #870 #747 #514 #687 #672 | −5 | −5 | | | +5 |
| #999, #1000 filed `ready` (the #870 post-land pass) | +2 | +2 | | | |
| #732 closed as **superseded** by this pass | −1 | | | −1 | +1 |
| #1001 filed `ready` | +1 | +1 | | | |
| #1002 filed `blocked` | +1 | | +1 | | |
| #1003–#1007 filed `ready` | +5 | +5 | | | |
| **this stamp** | **230** | **209** | **21** | **0** | **365** |

**Five `ready` issues are STILL done, and the count above still overstates what
is startable by five.** **#625**, **#748**, **#492**, **#934** and **#896** were
discharged by `34a8043` and judged with it, and all five remain open because
GitHub bound `Closes #669, #625, …` to the one reference following the keyword.
Each carries a comment naming the landing and telling a session not to take it;
**#896** gained a second this pass. **They are not closed here**: the
cartographer files, unblocks and restamps, and never closes an issue as done —
that is the develop loop's act, and it is the first item of the next planning
action below, now carried for a **third** stamp.

**The unblock sweep found two dependents and relabelled neither, and the zero is
correct.** All open bodies were fetched over `gh api` — byte-faithful, where MCP
`issue_read` is lossy (#764) — and their `## Depends on` sections searched for
the five just-closed numbers. **Two hits: #720 and #398**, and both were already
repaired *inside* the landing window by the #870 post-land pass, which struck
#720's *"Take #514, #672 and #687 before this lands"* paragraph as DISCHARGED
and repointed #398's sequence-related list. Neither is `ready`-eligible on this
evidence — **#720** stays `blocked` on **#472** alone, and **#398** was already
`ready`. Nothing else in the queue names any of the five in a dependency
position.

**One stale premise was found and corrected in a body this pass**, plus four
inside one issue. **#972**'s `## Depends on` named **#732**, which no longer
exists as an open issue; it now names **#1001**, with the arm-level reason
stated. Four further `#732` references inside #972 (Acceptance 5, Acceptance 7's
heading and its FALSE-REJECTS sentence, and the pre-filing search list) were
corrected in the body rather than only in a comment, per
`docs/WORKFLOW.md`'s filing discipline.

### The #732 re-plan — the pass's largest act

**#732 is closed as superseded and replaced by two issues, and the SPLIT is the
re-plan.** `docs/WORKFLOW.md`'s park rule asks the cartographer to close and
refile; refiling the same scope under a new number would have tracked nothing,
because the arbiter proved that scope cannot pass the gate in **any** landing
order (`issuecomment-5387489167` on #732).

- **#1001** — `vc:minVersion` plus the four availability attributes.
  **`ready`, startable on `main` today**, and it is what **#972** now depends on.
- **#1002** — the `vc:maxVersion` arm. **`blocked` on a ruling only the repo
  owner can give**, recorded as a trigger in its `## Depends on`.

**What made the split findable.** One case blocks §4.2.2:
`VC/vc002/instance/vc002.n1.xml`, a banked `pass` standing on the conjunction of
two gaps — no §4.2.2 pruning **and** no assertion evaluation — and doomed by
either being fixed alone. `saxonData/VC/vc002.xsd`, read in the tree this pass,
puts `vc:minVersion="1.1"` on the `<xs:assertion>` (RETAINED, 1.1 ≤ 1.1) and
`vc:maxVersion="1.1"` on the `<xs:pattern>` (pruned **only** by the max rule).
**So the min arm cannot touch that case.** Every one of the sixteen cases the
arbiter verified was then read back to the attribute its fixture actually uses:
thirteen need the min or availability arms and go to #1001 (schema `s4_2_2v01s`,
`s4_2_2ii01s`, `vc901`, `vc903`, `vc904`, `vc905`, `vc_008_1`, `vc_009_1`;
instance `s4_2_2ii01i`, `vc011.n1`, `vc012.n1`, `vc021.n1`, `vc022.n1`), and
three plus the regression need `vc:maxVersion` and go to #1002 (`vc007.xsd`,
`vc_003_1`, `vc006.n1` up, `vc002.n1` down).

**#1001 therefore predicts `schema` +8 / `instance` +5, zero regressions** — a
prediction derived from a measurement of a *different* diff, and labelled as such
in its body. **It does not retire the false reject #732's title named**; both
false rejects need the max arm. What it retires is the mechanism gap blocking
#972.

**The ruling #1002 waits on is a human's and no agent's.** The arbiter's two
routes are (1) a constitutional "superseded pass" ratchet class alongside
`GOXSD_RATCHET_REMOVALS`, which CLAUDE.md says changes *"only via a human-filed
issue"*, or (2) real assertion evaluation, which **#56**'s body records as still
unfiled and which is M6 XPath growth-tier-2 work. **This pass filed neither**,
which is the point: #1002 exists so the decision has an address.

### Persona consultations — the SIXTH consecutive, and the first run against a shipped CLI

Run by the orchestrating session, never by the cartographer (#416): each persona
saw only the README, `go doc` output and, for cliuser, the built binary. **Every
finding was checked against the tree before it was folded**, and two did not
survive that check as written.

**Filed, five:** **#1003** (README's opening pitch lists JSON, BER, XPath and
codegen with no per-capability status), **#1004** (no package doc states a
concurrency contract for `*xsd.Schema` or `*validate.Validator` — the whole
module matches `concurren|goroutine|thread-safe` in `doc.go` exactly **once**,
at `loader/doc.go:43`, about something else), **#1005** (`parser/doc.go`'s
multi-error PLANNED promise, unmarked and unowned), **#1006** (no Example
implements `Canonical` or a capability interface, and `backendtest.Run` is
exercised by none), and **#1007** (exit-code documentation uneven across the
three subcommand blurbs — `parse` omits exit 2, `gen` states none).

**Reconfirmed on standing issues, four:** **#409** (`codec`/`codegen` doc.go
present-tense with no milestone banner — and the exemplar to copy is now named,
`xsd/doc.go:227`'s `# Planned contract (M9 — not yet implemented)`), **#854**
(the `xsd` package doc's Query API buried behind ComponentID minting and
redefine-chain Loc charging — second independent sighting), **#56** (no exported
signal for *"was this fully decided"*, generalised past assertions to every
unevaluated conjunct of [validity]) with its doc half cross-referenced on
**#896**, and **#472** (the `-q parse` ordering trap, now a second witness one
day after the arbiter's).

**Two findings were corrected rather than filed as reported, and the corrections
are the reason to check.** libuser's *"`ExampleOverride` composes two existing
backends"* is **false about the tree** — `value/example_test.go:10-23` defines
`oneType`, a caller-owned `value.Backend`, and the example builds two of them —
so #1006 is scoped to what is genuinely absent (`Canonical`, a capability
interface, `backendtest.Run`) rather than to a from-scratch backend that already
ships. cliuser's attribution of *"Flags common to all subcommands"* to the README
is also false: `grep` finds it at `cmd/goxsd8/doc.go:23` and `main.go:43` and
**nowhere in `README.md`** — which makes the finding *stronger*, since that line
is what `goxsd8 -help` prints. **This pass posted that citation before checking
it and corrected itself on the same thread** (`issuecomment-5396771015` on #472);
the correction is the record, not the slip.

**One finding was DISMISSED with its reason, on #16.** cliuser asked that `-help`
point at `go doc github.com/kud360/goxsd8/cmd/goxsd8`. #870 decided the opposite
deliberately — `main.go:56-58`: *"a `go doc <import path>` invocation needs the
module tree and fails for an installed binary"* — and
`TestHelpPointerResolvesOutsideTheModule` fails if `helpPointer` names `go doc`.
The residual is recorded as a criterion on #16 rather than filed: `usage`
(`main.go:22-52`) carries **no** onward pointer of any kind, so the page every
error message sends a user to has no next step, and whether it owes a
resolves-anywhere pointer is #472's to answer when it rewrites that block.

**One finding is a confirmation and is recorded so the question is not
re-opened.** The three dispatch outcomes are distinguishable from the built
binary exactly as #514 promised, and the exit-2 message conflation cliuser once
called this CLI's sharpest trap is paid off.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
of it. Take from the top. Cross-references are stated by ISSUE rather than by row
number, which decays at each re-cut. **This is a `/backlog` re-cut: every rank is
re-derived from this pass's evidence.** Nothing in the band is claimed and
nothing in the `wip/*` namespace is live.

| # | Issue | Why here |
|---:|---|---|
| 1 | #843 | **PROMOTED to the head, and the steward rule is what does it.** *"A `kind/refactor` carrying a steward cost-of-delay ranking is banded on that ranking … a divergence the steward measured as increasing outranks a lane slice, and nothing else will ever lift it."* #843 **is** that rule's named exemplar and is now the **last of the three it banded still unlanded** — #979 landed 08-23, #978 landed 08-24. Four hand-maintained copies of one complex-type descent have **already diverged** on the redefine-original containment edge; two of the four do not descend `c.Base()` at all, and the bugs are **fail-open**, invisible to the ratchet, which is why an audit and not the suite found this. M4's open count is **45**, unchanged for a third pass and up from the 42 the 2026-08-16 audit recorded, so every new finalize-time constraint is a fifth copy picking its edges by eye. **Sizing is the open question, not value** — a session returning only a design comment naming the parameterization has done the right amount |
| 2 | #1001 | **Filed by this pass's #732 re-plan, and it is the only lane-moving vertical slice in the band with its grounding already bought.** `schema` +8 / `instance` +5 predicted with zero regressions, every case attributed to the fixture attribute it needs. The oracle memo settling all six `vc:` attributes, transform-vs-skip, the cascade, the ordering against include/override/redefine, `src-cip` and the empty-list inversion is on #732's thread and **does not decay** — it is a comment, not a container. A reference implementation the arbiter judged sound sits on `wip/issue-732` at `6334ffc`. **It unblocks #972**, whose own census names 27 candidate cases, and it is the head of the chain the last two stamps' next-planning-action asked for |
| 3 | #993 | **Two witnesses one day apart, and the unpaid cost is sitting in the queue right now.** PR #991 wrote `Closes #669, #625, #748, #492, #934, #896.` and closed **one**; PR #998 wrote five separate sentences and closed **five**. The mechanism is confirmed in both directions in 24 hours. `docs/WORKFLOW.md`'s Landing section still says only the singular *"`Closes #<N>` in the body closes the issue"*, so #998's defence was one session's LOG entry, not a rule. Meanwhile **#625, #748, #492, #934 and #896 are `ready` and done**, and a session may pick one tomorrow and spend a container discovering it. One WORKFLOW sentence plus a post-merge check; the only band member whose check must run AFTER the merge |
| 4 | #493 | **PROMOTED — the friction was paid a THIRD time, by this pass, and this is the first pass to pay it in the act the issue describes.** `docs/WORKFLOW.md`'s park paragraph says *"the cartographer closes the issue as superseded and files a replacement"* and names **neither the GitHub close reason nor the `ready` clear**; closing #732 today meant choosing `state_reason=not_planned` from nothing. The two standing corroborations hold: #968's session had to discover that a parked issue carrying `ready` is pickable, and `wipsurvey`'s RETIRED rows still include #933, closed `not_planned` and still carrying `ready`. Doc-only, one session, and it targets the **park** paragraph — not the filing-discipline paragraph #510, #646, #679 and #912 all target — so it rebases against none of them |
| 5 | #963 | **The tax falls on every landing and its discriminator STILL could not fire.** `git show --stat abe781a` is five files with `docs/LOG/2026-08.md` among them (+180 lines) — **carried**, so the discriminator did not fire on evidence for the **fifth** consecutive window. #820 landed the *form* of landing precondition 1; nothing checks that the check was run, which is **#924**'s shape and is not reachable by prose. One session — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part (#304 made CLAUDE.md's Commands block the sole gate definition). Banded on cost, not on sightings |
| 6 | #719 | **PROMOTED on a fresh outsider witness for the issue it gates.** `cvc-assertion` is wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56 alone**, and this pass's libuser reconfirmed #56 from the published surface only: there is no exported signal for *"was this schema+document combination fully decided"*, so a *"reject bad requests, accept the rest"* story cannot be built on this library at all today. #719 decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today. **It does not rescue #1002** — its own acceptance declines every case whose outcome turns on an assertion, which kills `vc002.n1` too |
| 7 | #981 | **Filed by the 2026-08-23 `/backlog`, which charged its friction to itself; still at three sightings of the tool half against one of the rule half, and this pass added neither.** `docs/WORKFLOW.md`'s empty-claim lease is dated by ANY thread comment, so an oracle grounding posted 44 minutes earlier locked that pass's band head for two hours — and `wipsurvey` cannot apply the rule at all, printing *"settle it from the issue thread"* for every CLAIMED row. The namespace has held no CLAIMED row for two passes, which is why the count is flat. Banded on CLAUDE.md's cost rule: one session, doc-first, and the tool arm may split off |
| 8 | #846 | **#909 paid the shadow tax and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`.** Its promotion rule asks for a third landing that pays the tax, and `abe781a` could not — it touches `cmd/goxsd8` and comes nowhere near the shadow. Its structural argument from #968's family A stands unchanged: the fault deciding those four documents lives outside `<restriction>`'s child list, so no predicate over that list can tell a fabricated verdict from a correct one. **#972 is its first measured witness and now sits behind #1001**, so the next landing that could pay this tax is two issues out. Still a ~700-line refactor with no evidence it fits one session |
| 9 | #941 | **#387's own unfiled debt, and the coldest files of any band row** (`07117dc`, 2026-08-21). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete, `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call. **Read #1001 first if both are in flight**: the parked §4.2.2 implementation gives `parser.Element` its own attribute slice, which is the same identifier |
| 10 | #953 | **A doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says *"every caller turns a decided negative into a schema rejection"*; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules |
| 11 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 12 | #1000 | **Filed by the #870 post-land pass, and it is the last link of the chain that landing opened.** `abe781a` added `go install ./cmd/goxsd8` because `go build ./...` over a multi-package module writes no executable — so the binary now *exists*. It is still not invocable under the name `README.md:66` uses twenty lines later: `grep -n PATH README.md` matches the one comment on line 46 and the token `$PATH` never appears in the file. One sentence, and a persona reached a runnable binary only by guessing a command #870 deliberately declined |

**Left the band this pass, and why.** **Row 1's five-issue bundle**
(#870 + #747 + #514 + #687 + #672) — **landed at `abe781a`** after five
consecutive reconfirmations, in fifty-nine minutes and one container. Worth
recording once: the band's cost argument for it was *"each is a sentence or a
dispatch branch while the CLI surface is still empty; taken afterwards every one
of them is a change to shipped behaviour"*, and the landing bought exactly that —
five issues, one grounding, one mason round, one arbiter round, `surface:
unchanged`. **#975** leaves the band **displaced rather than devalued**: nine
s4s-grammar rejection messages naming no Appendix A production, cheap and
mechanical, its #884 ordering discharged and its 21-site criterion re-run to the
same nine sites — but it gained no evidence this window and every row above it
did. It is the first thing to promote back if a cheap session wants a certain
win.

**Deliberately unbanded, and why.** **#1002** is `blocked` and belongs to no
band: a ruling, not a queue position, is what moves it. **#972** is the first
issue that becomes startable when #1001 lands and is worth reading beside it.
**#1003–#1007** are this pass's persona filings and are all real, all cheap and
all below the fold on the #992/#993 discriminator — each is a doc or example gap
whose cost is a wasted read, not a wrong row in the queue; **#1004** is the one
to promote first if a session wants a small high-value doc win, because it is
the only one whose answer must be *measured* (`go test -race`) rather than
written. **#999** is #398's sibling in the same test file and most likely
*deletes* a test; it waits on nothing and is a good pairing with #398.
**#996** and **#992** stay unbanded on the same discriminator, unchanged.
**#989** is #979's post-land filing — whether PRINCIPLES 26 requires generating
`regex`'s NameStartChar/NameChar tables from `docs/specs/md/xml.md`;
decision-first and cheap, and it still moves no lane and carries no steward
ranking. **#894** is clear to proceed — its #978 ordering is discharged on its
thread and its `## Depends on` is `none` — and stays unbanded only because it is
a lane-flat correctness fix whose first step is an unsettled oracle question.
**#888** and **#889** still await a suite census in their range. **#907**'s
`childElement` census is stale by at least five landings and must be re-derived
before anything is designed from it. **#885**'s three discriminators still have
one sighting each. **#409** is `ready` since 2026-08-02 with a **fifth**
independent sighting, this pass's libuser among them; it stays unbanded only
because it is one row of a five-file convention landing. **#854** gained its
second independent persona sighting and is one edit; promote it next pass if it
gains a third. **#670** asks for `parser/example_test.go` and is untouched — read
it beside **#1006**, which asks for the same thing in `value`. **#937** is
naturally folded by the next landing touching `rejectRepeatedAnnotations`.
**#920** and **#921** are conformance-bookkeeping follow-ups below the fold.
**#929** and **#931** are the small parser occurrence / rule-mapping gaps #901
exposed. **#455** is the live owner of the `strings.TrimSpace`-versus-§4.3.6
character class at **ten** sites, and **#456** stays `blocked` on it.
**#843–#849** are the 2026-08-16 audit's findings, **six open**, of which #843 is
now the band head and **#841** is `blocked` on a trigger that fired without a
ruling. **#566** is #565's open sibling, routed nowhere by #565's landing and
correctly so. **#852** owns both directions of the `gapaudit` matcher defect and
gained a fresh false-negative witness this pass (#1001 and #1005), but stays
below the fold because the tool again ran with reconciliation. **#692** and
**#925** are still `blocked` on a `/retro` trigger that fired without ruling on
them. **#570** carries the standing `schema` decline-count argument at 893
against a re-measured 788, unchanged this pass because no conformance
measurement was taken here.

### Next planning action

1. **Close the five. Third stamp, and the mechanism is now proven in both
   directions.** #625, #748, #492, #934 and #896 are discharged by `34a8043` and
   judged with it; only #669 closed. Closing them is the develop loop's act — the
   cartographer never closes an issue as done — and it is one
   `gh api ... -f state=closed -f state_reason=completed` per issue. Until it
   happens the `ready` count above overstates what is startable by five. The
   mechanism is **#993**, banded at row 3, and `abe781a` demonstrated the
   one-sentence-per-issue workaround works.
2. **A HUMAN DECISION IS NOW BLOCKING A FILED ISSUE, and this is the first stamp
   to carry one.** **#1002** waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID with a per-case justification in the verdict, and (b) holding
   §4.2.2's `vc:maxVersion` arm until real assertion evaluation lands. CLAUDE.md
   puts (a) beyond any agent — *"changes only via a human-filed issue"* — and (b)
   depends on M6 work **#56** records as unfiled. **No agent should attempt
   either.** Record the ruling as a comment on #1002; that comment is what moves
   it off `blocked`.
3. **Assertion EVALUATION is the largest unfiled thing this project has**, and it
   now blocks two issues rather than one: **#56** (through #719's encoding) and
   **#1002** (route b). It is M6 XPath growth tier 2 — `$value` binding, an F&O
   function library, typed comparison — and is too big for one issue. **Carving
   it is a `/backlog` act and it should happen at the M6 opening**, not sooner
   and not as a single ticket. Named here so that the next pass inherits it as
   work rather than as a footnote.
4. **The whole band is startable and unclaimed**, for the second consecutive
   stamp: no ref is CLAIMED and no ref is EXPIRED. Row 1 (#843) has now stood
   while both of its steward-ranked siblings landed, which is the starvation the
   rule exists to prevent.

**The two standing promotion discriminators both had a subject this window.**
**#963's** (did the landing carry its `docs/LOG` entry inside the squash?):
`git show --stat abe781a` shows `docs/LOG/2026-08.md` at +180 — **carried**, so
it did not fire on evidence for the fifth consecutive window and #963 stays
banded on cost. **#846's** (did the entry record the shadow tax?): it did not,
and it could not — `abe781a` touches `cmd/goxsd8` only. Say which way both fell
on the next stamp.

**The steward-ranking rule has produced two landings out of three bandings, and
the third is now row 1.** #979 landed 08-23, #978 landed 08-24, #843 stands.
**#841 remains the counter-example the rule cannot reach** — a `kind/refactor`
with a steward ranking that stays `blocked` because its trigger has no mailbox.
That gap is on #841's thread and is still not filed as an issue, because the fix
belongs to whichever pass gives Part 2 of the `/retro` the mailbox Part 1 has.

**Standing, unchanged, and still true.** Four unlanded corrections target one
paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**, **#646**,
**#679**, **#912** — and whichever lands last rebases three times; #493 is not a
fifth and #992 is not a sixth, both targeting different paragraphs of the same
file. **The next `/retro` inherits three issues it already owed** — #692, #925
and #841. **The CTA cohort's 45 banked `instance` failures remain
unattributed**, fourteenth consecutive stamp carrying it. **`gate.yml` runs but
is still not a required status check**, which only the repository owner can
change.

**Two environment costs stay in the log at one witness each, and neither gained
a second.** Uncached conformance test binaries hang under the default sandbox, so
conformance runs must be issued unsandboxed — this pass took no conformance
measurement, so it neither corroborated nor weakened it. Four consecutive arbiter
launches died on transient platform errors on 2026-08-24 with **no document
saying how many is enough**; no arbiter ran this pass. #978's log entry set the
bar for both: a second sighting promotes them into `docs/WORKFLOW.md`'s
checkpoint paragraph. **A third environment fact was observed this pass and is
NOT a cost**: a shallow clone leaves three `wip/*` tips unfetched, `wipsurvey`
reports their age as `unknown`, and its verdict ordering makes that harmless —
recorded on **#809**, filed nowhere.

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
`b3f295a`. Live ones in the same family are **#471** (a local `<element ref=>`
carrying `substitutionGroup=`, silently accepted), **#931** (occurrence
attributes on a named `<group>`'s child compositor), **#929**, **#455**, and
**#972** (an XSD-namespace child §4.1.2's `<simpleType><restriction>` has no
position for, dropped by `restrictionFacets` — `blocked` on **#1001**, which owns
the §4.2.2 conditional inclusion the same site needs first; that dependency named
#732 until 2026-08-24, when #732 was closed as superseded and re-planned into
#1001 plus #1002). A second, narrower family has opened beside it: the
rejections the producer already makes
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
+9, unioned onto #716's). `instance` stands at **10755** — #913's cvc-type
clause 3.1 landing added **9409**, itself M5 and the largest single lane move
this project has recorded — and **twenty-five** of the pre-#913 cases were not
M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure (+12), #821
added 1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). (The
*thirteen* this paragraph carried predated the CTA pair and did not sum; the
five figures do, and the sum is the number.) **It happened again after #913:
#909 — an M4 landing — took `instance` 10746 → 10752 (+6) by producing
`<simpleContent>` `<restriction>`, and **#862** — a `conformance` landing — took
it 10752 → 10755 (+3) at `109beb9`, so the outside-M5 total is now 34.** A slice
that produces a component the engine could not previously see, or decides a
`{type table}` it previously withheld, moves `instance` without deciding a new
`cvc-` rule. **#862 is a THIRD mechanism and the paragraph would be wrong to
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

**10755 is still a floor built for soundness, and #913's +9409 jump did not
change what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the 15606 that
still fail are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read 10755 as 41% of the suite passing.** It is the count of documents
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
