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

## Status — 2026-08-25 (`/backlog`. Replaced whole, per step 6: the lane table is a fresh `go tool lanestatus` paste, the branch namespace a fresh `wipsurvey` against a fresh `git fetch -p origin main`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh 599-issue page-numbered `state=all` fetch taken after this pass's own filings. **A `/backlog` re-derives the band ordering rather than shifting it**, so every row's rank below is argued from this pass's evidence. This pass follows THREE landings, closed **#993** as superseded after its park, filed its replacement **#1018** plus **#1019**, corrected three open bodies, and folded the **seventh consecutive** persona consultation)

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

**TWO lanes moved, ending three consecutive stamps of a byte-identical table.**
`schema` 13225 → **13259** (+34, fail 2173 → 2139); `instance` 10755 → **10760**
(+5, fail 15606 → 15601); `datatypes` unchanged. Both moves are attributed to a
named landing rather than to the interval:

| landing | commit | lane movement, from its own `Ratchet:` trailer |
|---|---|---|
| #843 | `7751980` | `unchanged` |
| #1001 | `684b2b4` | `schema` +8, `instance` +5, zero regressions |
| #972 | `e491ddb` | `schema` 13233 → 13259 (+26), zero regressions |

8 + 26 = 34 and 5 + 0 = 5, so the two lane deltas are fully accounted for and
nothing moved that no landing claims. The `instance` lane's indeterminate census
stays at 5. **No conformance measurement was taken by this pass** — the numbers
above are the committed expectations read through `lanestatus`, and the arbiter
runs quoted are each landing's own.

### The three landings this pass follows

**`7751980` (#843) retired the band head that had stood for four stamps.** Four
hand-maintained descents of one ComplexType component tree became one
`componentWalk` in `xsd/componentwalk.go` with five optional per-component-kind
charge callbacks; **13 hand-written walk functions deleted**. The divergence the
issue named was carrying a live fail-open — two of the four never descended
`c.Base()`, the one slot reaching a `<redefine>`-minted original, so
`e-props-correct` clauses 2 and 7 and `au-props-correct` clause 3 were uncharged
for everything nested inside one. Invisible to the suite by construction, which
is why an audit and not the ratchet found it. Arbiter REJECT round 1 **on three
sentences of prose with the code explicitly confirmed sound**, one repair round,
fresh ACCEPT with zero findings. `Ratchet: unchanged` measured twice.

**`684b2b4` (#1001) is the §4.2.2 split's first half, and it landed its
predicted table case for case.** Every schema document in the assembly closure is
now ·conditional-inclusion pre-processed· as it is read, for five of §4.2.2's six
`vc:*` attributes; the four availability attributes route through one
"every item is known" predicate so the empty-list inversion falls out of vacuous
truth. `vc:maxVersion` pruning is declined behind a `GAP(parser)` marker naming
**#1002**. **The 2026-08-24 stamp predicted `schema` +8 / `instance` +5 with zero
regressions and that is exactly what banked** — a prediction derived from a
measurement of a *different* diff, and the strongest evidence this project has
that a re-plan can be costed before it is taken.

**`e491ddb` (#972) is one predicate at ONE of two call sites, and it moved the
`schema` lane +26.** `restrictionFacets` discarded every XSD-namespace child of a
`<simpleType>`'s `<restriction>` it could not map, so the producer built a
component out of an s4s-invalid document and the harness called it `valid`.
`rejectOutOfModelFacetChildren` charges §5.1's first bullet at
`constructSimpleType`'s `<restriction>` arm alone — **not** in the shared
`restrictionFacets`, whose other caller reads `xs:simpleRestrictionType` — and
runs **before `resolveBase`**, so the grammar fault does not wait on whether
`base=` resolves. Arbiter ACCEPT in one round, zero blocking findings. It
supersedes **#968**'s gate-level route, which could not clear the ratchet.

**All three used one `Closes #<N>` sentence and closed exactly one issue each**,
verified per landing against the commit trailers. No new orphan was created this
window — which is the workaround working, not the check existing.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (re-run this pass against a fresh
`git fetch -p origin main` and the 599-issue `state=all` fetch, after #993 was
closed):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
732    wip/issue-732  unknown    RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  unknown    RETIRED  wip/issue-822: issue #822 is closed
872    wip/issue-872  unknown    RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  117h20m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  56h25m0s   RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993  unknown    RETIRED  wip/issue-993: issue #993 is closed
```

**One ref arrived and none left: six now, all RETIRED.** `wip/issue-993` is the
new row — created 2026-08-25, carrying three real commits (`39613f7`, `980c1e5`,
`246bba7`), parked `needs-replan` on a second arbiter rejection and closed as
superseded by this pass. Its REASON therefore reads *"is closed"* rather than
*"is labelled needs-replan"*; both verdicts are RETIRED and the change is
cosmetic. **No branch is CLAIMED, none is EXPIRED, no `parked/*` ref exists, and
nothing is UNKNOWN**, so **every band row below is takeable by the next session
that reads this table**. `wip/issue-843`, `wip/issue-1001` and `wip/issue-972`
never appear as survey rows: a landing branch is auto-deleted at merge.

**Four TIP AGE cells now read `unknown`, up from three, and that is still the
tool working.** This container's checkout is shallow
(`git rev-parse --is-shallow-repository` → `true`) and `6334ffc7`, `cc2d54e6`,
`0b34c21a` and now `246bba71` are not in its object store — `git log -1` on each
answers `unknown revision`, while `c2ba6315` and `53048638` read normally.
`wipsurvey` tests RETIRED **before** it tests for an unfetched tip, so all six
verdicts stand on issue state rather than on a borrowed age. **A fourth instance
of the environment premise #809 rests on**, recorded there; not a defect and
nothing filed.

**`wip/issue-993` is the first CLAIMED subject the namespace has held in three
passes, and it produced #981's fourth and fifth sightings before the branch
retired** — see band row 3. The survey above cannot show that, because by the
time it runs the issue is closed.

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON —
141 issues, 50 of them open; the open count moved 52 → 50 because #972 and
#1001 closed):

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

**The census moved for the first time in four stamps: 64 → 66, and a SIXTH area
opened.** Both new markers are `parser`, both from `684b2b4`
(`parser/conditional.go:208` and `parser/doc.go:89`) — package `parser` had
carried `GAP(` markers before only in `parser/xmltree`, `parser/produce_complex.go`
and `parser/redefine.go`, none of which the tool attributes to a `parser` area
tag. **#1002 correctly left group 2**: the marker it will retire did not exist
at the last stamp and now does, at the site `#1001` created.

**Group 2 also lost a row it should have kept, and that is a NEW tool defect
found by this pass.** **#921** — a conformance-lane tracker with no marker
anywhere, which belongs in group 2 permanently — vanished, and nothing about
#921 changed. `parser/conditional.go:208` matched it: `phraseMatch` runs at
`minPhraseWords = 5` and the marker's text and #921's body share exactly one
five-word window, **`claude md s one rule`** — both cite CLAUDE.md's one rule,
an idiom this repo writes everywhere. Isolating #921 as gapaudit's whole input
reproduces it (65 of 66 markers untracked, group 2 empty). Filed onto **#852**
as **defect 4**, with the derivation on that thread and a scope note added to
its body. **So neither group of this report is a complete list**: group 1 is
qualified by the file-path false-negative the last stamp recorded (**#1005** is
markerless and appears in neither group, because its body cites `parser/doc.go`
and a path cited for any reason matches every marker in that file), and group 2
is now qualified by defect 4. **#960** still owns the class the census
structurally cannot see — a fail-open disclosed in PROSE with no `GAP(` marker.

### Milestones and queue

Both columns and the queue below are RE-DERIVED this pass from one page-numbered
`state=all` fetch — 11 pages, **1019 items, 599 issues** once pull requests are
dropped — taken after this pass's own filings, not from the milestones endpoint.
`gh api --paginate` is still unusable (numeric-ID repository paths in the Link
header, 403 from the proxy, after writing the pages it did fetch), and
`gh issue list` is unusable for a different reason confirmed again this pass —
it is a GraphQL path, and GraphQL answers
`HTTP 403: This GraphQL query is not enabled for this session`. Repository-scoped
REST serves both reads and writes. `docs/ROUTINES.md` is accurate on all three.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 98 | 45 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**No milestone row moved, across three landings and four filings — the fourth
consecutive stamp with a flat table.** None of #843, #1001 or #972 carries a
milestone, and neither do #1011, #1014, #1018 or #1019. **172 of the 230 open
issues carry no milestone** (230 − 45 − 13), so the rows above are feature
progress and the paragraph below is the queue. M4's open count has now stood at
45 for four passes, against the 42 the 2026-08-16 steward audit recorded.

Queue: **230 open — 210 `ready`, 20 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 20), against **369 closed**.
210 + 20 + 0 = 230 exactly, and **every one of the 230 carries exactly one queue
label** — checked mechanically, not by eye. **The hygiene sweep is clean on all
three classes this stamp**: zero open issues with no queue label, zero with two,
and — after two repairs by this pass — **zero with no `area/` label**. #845 and
#839 were the last two carrying none and were given `area/parser` and
`area/meta`; the class #773/#774 fell into stays empty for the seventeenth
consecutive stamp.

**Every move reconciles.** From the previous stamp's 230 open / 209 `ready` /
21 `blocked` / 0 `needs-replan` / 365 closed:

| move | open | `ready` | `blocked` | `needs-replan` | closed |
|---|---:|---:|---:|---:|---:|
| previous stamp | 230 | 209 | 21 | 0 | 365 |
| #843 landed at `7751980` | −1 | −1 | | | +1 |
| #1001 landed at `684b2b4` | −1 | −1 | | | +1 |
| #972 relabelled `ready` by #1001's post-land pass | | +1 | −1 | | |
| #972 landed at `e491ddb` | −1 | −1 | | | +1 |
| #1011 filed `ready` (#843's post-land pass) | +1 | +1 | | | |
| #1014 filed `ready` (#1001's post-land pass) | +1 | +1 | | | |
| #993 parked `needs-replan` on its second reject | | −1 | | +1 | |
| #1018 filed `ready` (this pass — #993's replacement) | +1 | +1 | | | |
| #1019 filed `ready` (this pass) | +1 | +1 | | | |
| #993 closed as **superseded** by this pass | −1 | | | −1 | +1 |
| **this stamp** | **230** | **210** | **20** | **0** | **369** |

**Five `ready` issues are STILL done, and the count above still overstates what
is startable by five. This is the FOURTH stamp carrying it.** **#625**,
**#748**, **#492**, **#934** and **#896** were discharged by `34a8043` and judged
with it, and all five remain open because GitHub bound `Closes #669, #625, …` to
the one reference following the keyword. Each carries a comment naming the
landing and telling a session not to take it. **They are not closed here**: the
cartographer files, unblocks and restamps, and never closes an issue as done —
that is the develop loop's act, and it is the first item of the next planning
action below.

**The unblock sweep relabelled nothing, and the zero is measured.** All open
bodies were fetched over `gh api` — byte-faithful, where MCP `issue_read` is
lossy (#764) and, per **#1014**, silently truncating — and every `## Depends on`
section was split out and searched for #843, #1001, #972 and #993. **No open
issue names any of the four in a dependency position.** The only `blocked` body
naming one is **#1002**, whose `## Depends on` cites #1001 as discharged in
writing; it stays `blocked` on a human ruling and nothing else. The three
landings' own post-land passes had each already run this sweep to the same zero.

**Three open bodies were corrected rather than commented at**, per
`docs/WORKFLOW.md`'s filing discipline:

- **#941** — the largest correction, and the only one that changed an argument
  rather than a citation. Its Acceptance 2 argued the two methods are *not* the
  same shape because *"`Attributes` … Its body is `return e.src.Attributes()` —
  genuine delegation"*. `684b2b4` changed that body to **`return e.attrs`**, so
  the stated reason is dead while the conclusion survives on the field name
  (`attributes()` collides with nothing; `baseURI()` would collide with
  `baseURI`). Five line citations were re-measured, and a new **Acceptance 6a**
  warns that `parser/conditional.go` reads and writes `e.attrs` directly at five
  sites — §4.2.2's transform makes `e.attrs` the schema document and
  `e.src.Attributes()` the pre-processing input, so tidying the accessor back to
  `e.src` would silently undo §4.2.2. Derivation:
  `issuecomment-5411942305`.
- **#852** — gained **defect 4** (above) plus a Notes bullet saying group 2 is in
  scope, because a fix repairing group 1 alone leaves the stale-tracker group
  wrong in the opposite direction.
- **#992** — its sibling-family bullet named **#993** twice, which is now closed;
  both now name **#1018**, with the supersession stated in the body rather than
  left to a comment.

### The #993 park and re-plan — the pass's largest act

**#993 is closed as superseded and replaced by #1018, and the re-plan is a
NARROWING rather than a split.** The issue parked `needs-replan` on 2026-08-25
after two arbiter rejections — the hard cap (PRINCIPLES 30) — and its parking
comment named this close as the cartographer's next step.

**Both rejections hit the same defect through two different mechanisms, and
neither hit the code.** The gate ran green all four parts on both rounds and
`surface: unchanged` held mechanically; the diff is two `.md` files. What never
converged is Acceptance item 2's post-merge check, specifically **what it
iterates over**:

| attempt | iteration set | on PR #991's body it enumerates | verdict |
|---|---|---|---|
| round 1 | the body's bound `Closes #<N>` sentences | `{#669}` | reject |
| round 2 | `/develop` step 2's pick | `{#669}` | reject |

`/develop` step 2 picks exactly one issue and names one branch for it, so a
multi-issue landing never originates there — it originates at absorption, whose
only durable record is the commit body. **Both attempts therefore passed on the
regression fixture they were written for, and passed identically on the
compliant one**, which is no discriminating power at all.

**#1018 encodes the fix the second verdict states, and adds the test neither
round had.** Its iteration set is *every issue reference the squash body names,
bound or plain mention*; and its Acceptance 3 is an executable discriminator —
against PR **#991** (`34a8043`) it must yield six and FAIL, against PR **#998**
(`abe781a`) five and PASS. **The scope shrank rather than grew**: two of the
first verdict's three findings were delivered on `246bba7` and confirmed there,
the `.claude/commands/develop.md` absorption was ruled correct twice, and
`docs/WORKFLOW.md:253-255` is the only region that must be rewritten. #1018's
Notes point a new attempt at `git diff 2e3bf73...246bba7` as its starting diff.
`wip/issue-993` stays retired in place as re-planning evidence; the branch scheme
forbids re-attempting an issue under its own number.

**`needs-replan` was retained on the closed #993 rather than cleared**, on the
#968 precedent — the label is what retires the branch in the survey, and
clearing it would change only the stated REASON. **That decision, and the choice
of `state_reason=not_planned`, both had to be made from nothing in the document**,
which is band row 4.

### Persona consultations — the SEVENTH consecutive, and the first to file NOTHING

Run by the orchestrating session, never by the cartographer (#416): each persona
saw only the README, `go doc` output and, for cliuser, the built binary.
**Nothing was filed, and the silence is a result rather than an omission** —
recorded here so the next pass does not read a filing count of zero as a
consultation that was skipped.

**cliuser** built the binary and exercised `-help`/`-h`/`--help` in every
argument position, `-help=true`, `--`, case sensitivity, all subcommands, and
the `-q`/`-v`/`GOXSD_DEBUG` combinations. **No new disagreement between
`README.md`, `-help`, `go doc` and the binary.** Reconfirmed **#472** (the
ordering trap: `goxsd8 -q parse order.xsd` still exits 2 with *"no subcommand
given"*), **#1007** (exit-code unevenness, now confirmed against the shipped
binary rather than against `go doc` — second sighting one day after filing), and
**#16** (#870's error-path fix is present, and the residual is exactly what the
last stamp recorded: `usage` itself carries no onward pointer). One incidental
positive: **the `-help=true` guard behaves exactly as `go doc` documents**,
falling through to *"no subcommand given"* rather than triggering help — a
documented behaviour with no test naming it, and worth preserving when #472
changes the flag parser.

**libuser** copied the README's `parser.Parse` → `validate.New` →
`xmlsrc.Validate` → `Result.Violations()` snippet into a scratch module against
this repo via `replace`. **It built and type-checked with zero edits.** That is
the one fully-realized end-to-end story in the documentation and it works as
advertised — banked here as a positive regression datum, because nothing in this
repo compiles README snippets and the next stamp should say whether it still
holds. Reconfirmed **#1004** (with a sharper argument: `xsd.Matcher`'s doc says
outright *"not safe for concurrent use"*, which proves the project writes that
sentence when it matters and makes its absence on `xsd.Schema` and
`validate.Validator` read as an oversight rather than a convention), **#1005**
(every other forward-looking doc claim in the codebase carries an M-number —
`xsd/doc.go:227`'s `M9` is the exemplar — and `parser/doc.go`'s multi-error
promise carries none, so it reads as permanently unowned), **#472**
independently, and **#1003** (the README's opening capability list still reads
present-tense at a skim; the XPath bullet carries no milestone number at all).

**Both sharper arguments were banked on their threads**, not only here, because
`docs/PLAN.md` is replaced each pass and an issue thread is not:
`issuecomment-5411888920` on #1004, `issuecomment-5411893453` on #1005,
`issuecomment-5411893683` on #1007.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
of it. Take from the top. Cross-references are stated by ISSUE rather than by row
number, which decays at each re-cut. **This is a `/backlog` re-cut: every rank is
re-derived from this pass's evidence.** Nothing in the band is claimed and
nothing in the `wip/*` namespace is live.

| # | Issue | Why here |
|---:|---|---|
| 1 | #846 | **The steward rule's only remaining subject, and this is the first pass to MEASURE the increase its ranking asserts.** *"A `kind/refactor` carrying a steward cost-of-delay ranking is banded on that ranking … a divergence the steward measured as increasing outranks a lane slice, and nothing else will ever lift it."* All three issues that rule previously banded have landed — #979 on 08-23, #978 and #843 on 08-24 — and of the 2026-08-16 audit's open findings only **#844** carries the other `Increasing` ranking, on named future consumers rather than on a measurement. #846's is measured: at `2e3bf73` the shadow is `conformance/schema.go:593`–`1479`, `schemaShapeDecidable` plus **seventeen** `*Decidable` predicates, **887 lines**, against the audit's `563`–`1229` and fifteen — **two predicates and ~220 lines of growth**, re-derived independently this pass and not copied. Row 8's old *"~700-line refactor"* caveat understates by about a quarter. The upkeep discriminator did **not** fire on any of the three landings (#843 is `xsd/`-only; #1001 is a genuine `parser` widening that needed no mirror because §4.2.2 is a pre-pass; #972 was a narrowing) — but #732's parked attempt supplies a third *structural* witness in this issue's own words: mason's `vc:` probe was rejected as *"the same shadow-model shape #846 exists to retire, added one layer further out."* **Sizing is the open question, not value** — a session returning only a design comment naming the seam has done the right amount |
| 2 | #1018 | **The cost went UP this window, in both of the ways it can.** The five done-`ready` issues (#625, #748, #492, #934, #896) stand for a **fourth** stamp; and #993 itself consumed a full session, two mason rounds and two arbiter rounds and landed **nothing**. **But the second cost bought the design**: the arbiter's second verdict names the correct iteration set outright, #1018 encodes it, and its Acceptance 3 adds the discriminator neither round tested for — six-and-FAIL on PR #991, five-and-PASS on PR #998. **Scope shrank**: two of three findings are delivered on `246bba7` and confirmed there, and only `docs/WORKFLOW.md:253-255` must be rewritten over a diff an arbiter already judged sound. Highest value-per-session in the band. The only member whose check runs AFTER the merge |
| 3 | #981 | **PROMOTED four rows on two fresh sightings in a single day — and the namespace's first CLAIMED subject in three passes is what supplied them.** The last two stamps recorded no new sighting because the rule only mis-serves a CLAIMED row and there was none; `wip/issue-993` was one. Both settlements were on a branch with no commits of its own: at 08:21Z a session dated the lease by an **oracle GROUNDING** posted 4h 09m earlier, and at 12:39Z another dated it by the **08:21 TAKEOVER comment**, 4h 18m earlier. `wipsurvey` contributed nothing to either — it reads `{number, state, labels}` and never sees a comment timestamp, so it printed its `CLAIMED … settle it from the issue thread` non-answer twice, four hours apart. **The rule half now has BOTH fixtures**: #884's grounding at 44 minutes is the negative, #993's 08:21 takeover the positive, and whatever item 1 writes must return takeable for the first and LIVE for the second. One session, doc-first, and the tool arm may split off |
| 4 | #493 | **The friction was paid a FOURTH time and, for the SECOND CONSECUTIVE `/backlog`, in the exact act the issue describes.** Closing #993 as superseded today meant choosing `state_reason=not_planned` from nothing in `docs/WORKFLOW.md`'s park paragraph, and then deciding by precedent alone whether to clear `needs-replan` — both decisions had to be written into the closing comment so the next pass would not re-derive them. That is CLAUDE.md's consecutive-sessions cost rule with a clean streak. It also gained a **counter-example that corrects its own body**: the fourth Acceptance bullet claims *"Every park to date has cleared `ready`"*, and **#933** did not — closed `not_planned`, still carrying `ready`, carrying no `needs-replan` at all, with `wipsurvey` reporting RETIRED on the closed state and never seeing the contradiction. Doc-only, one session, and it targets the **park** paragraph, not the filing-discipline paragraph #510, #646, #679 and #912 all target |
| 5 | #719 | **The band's highest lane-relevant row, and nothing gates it.** `cvc-assertion` is wired fail-open at every variety level — the M6 seam, marked and measured, and group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56 alone**, and #56 is the "was this schema+document combination fully decided" question a *"reject bad requests, accept the rest"* story cannot be built without. It decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. **Its promotion argument is one witness weaker than last stamp** — the seventh consultation reconfirmed #1004, #1005, #472, #1003, #1007 and #16, and did *not* revisit #56 — so it is ranked here on the seam and not on a fresh sighting. **It does not rescue #1002**: its own acceptance declines every case whose outcome turns on an assertion, which kills `vc002.n1` too |
| 6 | #852 | **PROMOTED — the tool the cartographer runs every pass now reports something FALSE, not merely noisy, and this pass supplied the reproduction.** Defect 4, filed into its body today: `parser/conditional.go:208` matched **#921** on the five-word window `claude md s one rule` — pure house boilerplate — and silently emptied #921's group-2 row. Defect 3's phrase collisions surface as a visible `dead end:` line a reader can check; this one is a **suppression** with nothing to check. Combined with the file-path false negative that hides #1005 from both groups, **no group of this report is now a complete list**, and every `/backlog` pays a paragraph writing that caveat down. Third distinct witness class in two passes, and the first with a reproduction command. Defect 1's citation-first matcher does **not** cover it — neither text carries the other's number — so group 2 needs its own answer |
| 7 | #963 | **The tax falls on every landing and its discriminator STILL could not fire — three more clean windows, eight in a row.** `git show --stat` on `7751980`, `684b2b4` and `e491ddb` shows `docs/LOG/2026-08.md` at +245, +218 and +157: **carried, three for three**. A long run of non-firings reads like a case for demotion and is not one — #820 landed the *form*, and what is unchecked is that the check was RUN, which is #924's shape and is not reachable by prose. One session — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part (#304). Banded on cost, not on sightings. Whichever of this and #1018 lands first should say on the other's thread whether one tool holds both checks |
| 8 | #941 | **Its body was repaired by this pass and a session now starts from a correct table for the first time in two decays.** `684b2b4` rewrote `Attributes`' body from `return e.src.Attributes()` to `return e.attrs`, killing Acceptance 2's stated reason while leaving its conclusion intact, and moved five line citations. Both methods still fail STYLE T5 on both prongs — `parser/tree.go:31` still says so in the tree — and `surface: +0 -2`, strictly shrinking. **New Acceptance 6a is the trap to read first**: `parser/conditional.go` reads and writes `e.attrs` directly at five sites, so `e.attrs` is the schema document and `e.src.Attributes()` is §4.2.2's pre-processing INPUT; the two are no longer the same list. **Warden pre-flight required**; do NOT start from #387's item-2 table |
| 9 | #844 | **The second `Increasing` steward ranking in the queue, and the consumer it names is the row three places above it.** `dispatchUnion` computes a union's ·active member· and discards it, while ·validating type· is already needed by `cvc-id` and identity-constraint key members — and its named upcoming consumers are `cvc-assertion` (**#719**, row 5) and CTA. Ranked below #846 because its ranking is argued from future consumers where #846's is measured on the record, and below the process rows because none of its consumers has landed yet. **Worth reading beside #719**: a session taking #719 should say on this thread whether the seam it lands makes this a smaller job |
| 10 | #975 | **Displaced rather than devalued, for a second stamp, and it is the band's certain win.** Nine s4s-grammar rejection messages naming no Appendix A production, against the bar `xsderr/doc.go` set with #966; its #884 ordering is discharged and its 21-site criterion re-ran to the same nine. It gained no evidence this window and rows above it did — but nothing about it decayed either, and it is the first thing to promote if a session wants a bounded mechanical landing rather than a design question |
| 11 | #953 | **A doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says *"every caller turns a decided negative into a schema rejection"*; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules |
| 12 | #853 | **The band's second lane row, and its first step is not code.** `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **Start with an oracle question**: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |

**Left the band this pass, and why.** **#843** — landed at `7751980` after
standing at or near the head for four stamps, exactly the starvation the steward
rule exists to prevent and the third of three the rule banded. **#1001** —
landed at `684b2b4`, and it is worth recording once that the band's predicted
`schema` +8 / `instance` +5 banked case for case; a re-plan that costs its own
outcome before it is taken is repeatable. **#993** — parked, closed as
superseded, and its row is inherited by **#1018** at row 2 rather than vacated.
**#1000** leaves the band **displaced rather than devalued**: one sentence about
`$PATH` in the README's Quickstart, still true, still cheap, and it gained no
evidence this window while five rows above it did.

**Deliberately unbanded, and why.** **#1002** is `blocked` and belongs to no
band: a ruling, not a queue position, is what moves it, and its two spent
sequencing premises were struck from its body by #972's post-land pass so the
ruling is now the only thing holding it. **#1019** is this pass's own filing —
`docs/WORKFLOW.md`'s label list names four labels never used once in 599 issues
(`area/model`, `area/cli`, `area/codegen`, `area/codec`) and omits the four most
used, including **`area/xsd` at 164 issues, the single most-used area label in
the repository**; a decision plus two edits, unbanded only because no session has
ever been misled by it in a way the log records. **#1011** is #843's own
follow-up — each finalize phase still picks its ROOT tables independently of the
callbacks it sets, with **#725** as its live victim — and is the natural second
session for whoever lands #846. **#1014** is the MCP `issue_read` truncation
(#1001 came back at 55% with three sections never returned); it is real and it is
below the fold only because `gh api` is the documented byte-faithful path and
every survey in this stamp used it. **#1003–#1007** are the sixth consultation's
filings, all reconfirmed by the seventh and none promoted; **#1004** is still the
one to take first, because it is the only one whose answer must be *measured*
(`go test -race`) rather than written. **#999** is #398's sibling and most likely
*deletes* a test. **#996** and **#992** stay unbanded on the discriminator that
#992's cost was paid-and-absorbed. **#989** is #979's post-land filing,
decision-first and cheap, moving no lane and carrying no steward ranking.
**#894** is clear to proceed and stays unbanded only because it is a lane-flat
correctness fix whose first step is an unsettled oracle question. **#888** and
**#889** still await a suite census in their range. **#907**'s `childElement`
census is stale by at least eight landings. **#885**'s three discriminators still
have one sighting each. **#409** is `ready` since 2026-08-02 with a fifth
independent sighting and stays unbanded only because it is one row of a five-file
convention landing. **#854** did not gain a third sighting this pass and stays
put. **#670** asks for `parser/example_test.go` — read it beside **#1006**.
**#937** is naturally folded by the next landing touching
`rejectRepeatedAnnotations`. **#920** and **#921** are conformance-bookkeeping
follow-ups below the fold; #921 additionally now carries the gapaudit-suppression
note. **#929** and **#931** are the small parser occurrence / rule-mapping gaps
#901 exposed. **#455** is the live owner of the `strings.TrimSpace`-versus-§4.3.6
character class at **ten** sites, and **#456** stays `blocked` on it. **#845**,
**#848** and **#849** are the 2026-08-16 audit's remaining open findings, all
ranked *stable debt* by the steward and therefore all outside the rule that bands
#846 and #844 — though #849's stable ranking is already falsified on its own
thread, the copy count having grown while the pattern held. **#841** is
`blocked` on a trigger that fired without a ruling. **#566** is #565's open
sibling, routed nowhere by #565's landing and correctly so. **#692** and **#925**
are still `blocked` on a `/retro` trigger that fired without ruling on them.
**#570** carries the standing `schema` decline-count argument at 893 against a
re-measured 788, and it is now **two landings staler** — no conformance
measurement was taken here.

### Next planning action

1. **Close the five. FOURTH stamp, and the cost is now measurable in sessions as
   well as in queue rows.** #625, #748, #492, #934 and #896 are discharged by
   `34a8043` and judged with it; only #669 closed. Closing them is the develop
   loop's act — the cartographer never closes an issue as done — and it is one
   `gh api ... -f state=closed -f state_reason=completed` per issue. Until it
   happens the `ready` count above overstates what is startable by five. The
   mechanism issue is **#1018**, banded at row 2, and the three landings of this
   window all used the one-sentence-per-issue workaround correctly.
2. **The human decision blocking #1002 is unchanged and is now carried for a
   second stamp.** **#1002** waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID with a per-case justification in the verdict, and (b) holding
   §4.2.2's `vc:maxVersion` arm until real assertion evaluation lands. CLAUDE.md
   puts (a) beyond any agent — *"changes only via a human-filed issue"* — and (b)
   depends on M6 work **#56** records as unfiled. **No agent should attempt
   either.** Its `GAP(parser)` marker is live at `parser/conditional.go:208`;
   record the ruling as a comment on #1002, and that comment is what moves it off
   `blocked`.
3. **Assertion EVALUATION is still the largest unfiled thing this project has**,
   and it still blocks two issues: **#56** (through #719's encoding) and
   **#1002** (route b). M6 XPath growth tier 2 — `$value` binding, an F&O
   function library, typed comparison — too big for one issue. **Carving it is a
   `/backlog` act and it should happen at the M6 opening**, not sooner and not as
   a single ticket.
4. **The whole band is startable and unclaimed**, for the third consecutive
   stamp: all six `wip/*` refs are RETIRED, none is CLAIMED, none is EXPIRED.
   Row 1 (#846) is now the steward rule's last standing subject, and the rule's
   record is three landings out of four bandings.

**The two standing promotion discriminators both had three subjects this window
and neither fired.** **#963's** (did the landing carry its `docs/LOG` entry
inside the squash?): `docs/LOG/2026-08.md` is in all three squashes, at +245,
+218 and +157 — **carried**, eighth consecutive clean window, and #963 stays
banded on cost. **#846's** (did the entry record the shadow tax?): **no**, and
the three reasons differ — #843 is `xsd/`-only, #1001 is a pre-pass widening that
owed no mirror, #972 is a narrowing that owed none either. **The compensating
evidence is the re-measurement**, which is why #846 rose rather than fell. Say
which way both fell on the next stamp, and note that #846's discriminator has now
been unable to fire in six consecutive windows — if a seventh passes without a
producer widening that edits both files, the discriminator itself is the thing to
re-examine, not the issue.

**The steward-ranking rule has produced three landings out of four bandings, and
the fourth is now row 1.** #979 landed 08-23, #978 and #843 on 08-24, #846
stands. **#841 remains the counter-example the rule cannot reach** — a
`kind/refactor` with a steward ranking that stays `blocked` because its trigger
has no mailbox. That gap is on #841's thread and is still not filed as an issue,
because the fix belongs to whichever pass gives Part 2 of the `/retro` the
mailbox Part 1 has.

**Standing, unchanged, and still true.** Four unlanded corrections target one
paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**, **#646**,
**#679**, **#912** — and whichever lands last rebases three times; #493, #992,
#1018 and #1019 are not among them, all targeting different paragraphs of the
same file. **The next `/retro` inherits three issues it already owed** — #692,
#925 and #841. **The CTA cohort's 45 banked `instance` failures remain
unattributed**, fifteenth consecutive stamp carrying it. **`gate.yml` runs but is
still not a required status check**, which only the repository owner can change.

**Environment costs stay in the log at one witness each and none gained a
second.** Uncached conformance test binaries hang under the default sandbox, so
conformance runs must be issued unsandboxed — this pass took no conformance
measurement, so it neither corroborated nor weakened it. Four consecutive arbiter
launches died on transient platform errors on 2026-08-24 with no document saying
how many is enough; no arbiter ran this pass. #978's log entry set the bar for
both: a second sighting promotes them into `docs/WORKFLOW.md`'s checkpoint
paragraph. **Two environment facts were re-observed and neither is a cost**: a
shallow clone leaves four `wip/*` tips unfetched and `wipsurvey`'s verdict
ordering makes that harmless (recorded on **#809**, filed nowhere); and
`gh issue list` fails with `HTTP 403: This GraphQL query is not enabled for this
session` while repository-scoped `gh api` REST serves every read and write this
pass needed — which is exactly what `docs/ROUTINES.md` already says, confirmed
rather than newly found.

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
