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

## Status — 2026-08-15 (post-land, #722)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1017 | 25344 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12832 | 2566 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**No lane moved.** #722 (`fdbdd55`, PR #804) changed `tools/wipsurvey` and three
process documents and touches no validator; the arbiter ran the ratchet on the
merged tree and the three non-empty lane files came back byte-identical. The
standing `instance` figure of 1017 is #790's, carried through the merge.

**This is a post-land pass, not a full backlog run.** The lane table, the queue
counts, the branch namespace and all 21 `blocked` bodies were re-read from their
sources. **The working band below is NOT re-derived and did not need to be** —
a full `/backlog` re-derived it three hours before this landing (`2d6ec5c`) and
had already unbanded #722 as in-flight; the only change is that the paragraph
explaining its absence is now false and has been removed. No issue body was
audited that this landing did not touch.

Milestones — carried from the `2d6ec5c` stamp, which read them from GitHub this
morning. **#722 carries no milestone**, so no row can have moved; this is a
carry with a reason, not a carry for want of a read.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 83 | 41 | **active** |
| M5 — Instance validation (XML) | 10 | 12 | **active** |
| M6–M12 | 0 | 0 | not filed |

Queue, read fresh across both pages: **193 open issues — 172 `ready`, 21
`blocked`, 0 `needs-replan`, 2 `epic`** (both `blocked`), against **289 closed**.
172 + 21 = 193 exactly, so the arithmetic proves the label sweep rather than a
sweep asserting it. Two of the 193 are this pass's own filings. Read the
milestone table as feature progress, not as the queue: **140** of the 193 carry
no milestone (193 − 41 − 12), and the process, doc and comment-accuracy issues
post-land passes file are deliberately outside them.

**One issue of the previous stamp's 195 is unaccounted for, and is flagged
rather than absorbed.** That stamp published 195 open / 288 closed; the live read
is 193 / 289. #722 closing explains one of the two departures from `ready` and
the one arrival in `closed`. The second is not explained by any closure this pass
can see, and the same stamp records an off-by-one it had to correct in its own
no-milestone count. Treat 193 as the measurement and 195 as suspect; the next
full `/backlog` should reconcile rather than re-derive from either.

### What this pass did

**Unblock sweep: nothing to unblock, verified rather than assumed.** All 21
`blocked` bodies were read and their `## Depends on` lines checked: **no open
issue names #722**. #718 was already unblocked by #790's post-land pass earlier
today and is `ready`; that relabel is not re-done here.

**#722's follow-up ledger is fully dispositioned** — two filed, one declined,
one routed, with no hand-offs. The reasoning is on that issue's thread
(comment 5302972381); the outcomes are:

- **#805** — `TestGitAncestryWithoutMain` passes with the `mainSHA == ""` guard
  it names deleted, because `merge-base --is-ancestor <sha> ""` exits 128 and
  maps to the same unresolved result. Found by the arbiter's mutation pass, which
  killed every other test by reverting its own arm. The test pins a real outcome
  contract, so the choice is between asserting the un-spawned subprocess and
  correcting the comment — #805 asks the implementer to weigh that, not to guess.
- **#806** — an unresolved ancestry falls back to the borrowed tip age with
  nothing in the report saying so. Accepted design at #722, filed as the marker
  the arbiter suggested — but its body carries an **unreproduced hypothesis**
  worth more than the marker: `remoteRefs` reads main's SHA live from `ls-remote`
  while the ancestry test resolves it against the local object store, so a
  checkout that last fetched before the newest landing may resolve no ancestry at
  all and fall every lease back onto the borrowed age. Measure before coding; if
  the hypothesis fails, closing #806 as cosmetic is the right outcome.
- **The borrowed-age repetition is DECLINED as a standalone issue**, per the
  arbiter's own disposition (*"a reason to resist the next paragraph"*). It is
  prose redundancy in one 500-line file, not two encodings of state that can
  drift, and an issue whose acceptance is "delete four comments" invites a
  session to strip the prose four incidents argued for. The count is corrected on
  the record: **eight sites, not six** — six in `tools/wipsurvey/main.go` plus
  `docs/WORKFLOW.md:65` and `.claude/commands/develop.md:31`, which are separate
  audiences and the two least worth collapsing. The `STYLE P2` citation both
  records carry is logged as a **candidate fourth sighting on #548**: `P2` is
  "Comment only constraints" and says nothing about stating a constraint twice;
  the text that governs is CLAUDE.md's *"state it once"*, which has no letter ID.
- **"CLAIMED has no automated release path" is `/retro`'s, not a new issue.** It
  is in this landing's `docs/LOG/2026-08.md` entry twice — Friction item 1 and
  `Next:` item 3 — and a second tracker for it is the duplicate this queue does
  not need.

**The `CLAIMED` hole has a live instance at the top of the queue, and it is
handled on the thread.** `wip/issue-718` is a stood-down zero-commit lease on the
band's row 1, so `wipsurvey` now says `CLAIMED` — off-limits, with no age that
releases it — while this roadmap says start there. Nothing is wrong with the
landing: `CLAIMED` is deliberately conservative and its documented release path
is the issue thread, so that thread now carries the note
(<https://github.com/kud360/goxsd8/issues/718#issuecomment-5302974476>) telling a
session to take #718 and branch fresh from `origin/main`. **#718 is not
`needs-replan` and must not be relabelled** — the claim was stood down by
falsification, not retired.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (issue JSON supplied from MCP, since `gh` is
403 on both routes — #527, #682). **This is the first snapshot taken with #722's
fix in place**, and the `CLAIMED` row is the defect that produced four incidents
in twelve days, now reading honestly:

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
256    wip/issue-256  303h7m0s   RETIRED  wip/issue-256: issue #256 is closed
271    wip/issue-271  282h51m0s  RETIRED  wip/issue-271: issue #271 is closed
287    wip/issue-287  251h13m0s  RETIRED  wip/issue-287: issue #287 is closed
718    wip/issue-718  main's     CLAIMED  wip/issue-718: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
```

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-718` | **CLAIMED** | 0 commits; tip `e3ed308` is a `main` commit, so it holds no work. The claim is **stood down** on the issue thread, which is what `CLAIMED` defers to. #718 is `ready` and band row 1 — branch fresh from `origin/main`, never resume this ref. Deletable. |
| `wip/issue-256` | RETIRED | issue closed `needs-replan`, superseded by #470 |
| `wip/issue-271` | RETIRED | issue closed `needs-replan`, superseded by #478/#479/#480 |
| `wip/issue-287` | RETIRED | issue closed, answered in the opposite direction by PR #511. Not a park — it carries no `needs-replan` |
| `claude/eloquent-cerf-15r30e` | merged | tip `9ca50fc`, an ancestor of `main` — verified this pass, not carried. Deletable. |
| `claude/eloquent-cerf-8f1mrv` | merged | 0 ahead; tip `77dd47a` is a commit on `main`. Deletable. |
| `claude/eloquent-cerf-patxs3` | superseded | tip `2196c3f`, unmoved since 2026-08-07. Deletable. |
| `claude/eloquent-cerf-7ckw7b` | **merged — VERIFIED BY CONTENT** | all seven commits are on `main` (per-file check at `2d6ec5c`; the four-stamp "needs a human look" flag stays retired). Deletable. |

`wip/issue-722` was auto-deleted by GitHub at merge — the single deletion in the
system, working as documented. No `parked/*` branches, and **no `wip/*` branch
holds a live claim**: every open issue with a branch is either retired or stood
down, so the next develop session claims from `ready` rather than resuming.

**The checkout is SHALLOW, and every ahead/behind figure this table has ever
published was arithmetic on a truncated history (#802).**
`git rev-parse --is-shallow-repository` is `true` and `.git/shallow` holds two
boundary commits, so five of the eight branches have no merge base with `main`
at all and `git rev-list --left-right --count` does not fail on that — it counts
every commit on each side and returns a plausible pair. **Under squash-merge,
ahead/behind was never the right question anyway**: a squashed branch is always
"N ahead" of the `main` that already contains it. The table answers by content
and by ancestry instead. `wipsurvey` is unaffected — it keys on `ls-remote`, tip
timestamps and `merge-base --is-ancestor`, never on counts.

**`go tool gapaudit`: no P3 leak** — 53 markers across 5 areas, from the
`2d6ec5c` full run; this landing added no `GAP(` site, so the census stands. Its
two group-1 entries are heuristic misses, not untracked sites — both
`validate/cvccomplexcontent.go` markers name open issues (#716, #774) in their
own text and the matcher failed only because the marker prose is long. Its seven
group-2 entries are all expected: #404, #591, #592 and #593 are conformance-lane
gaps, which carry no marker by construction; #398 exists precisely because the
`cmd` stub has none; and #719 and #787 are the issues whose markers land with
them.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 172 of
them. Take from the top. **Carried unchanged from the `2d6ec5c` full re-derivation
three hours before this landing** — every row's justification survives a tooling
change that moved no lane.

| # | Issue | Why here |
|---:|---|---|
| 1 | #718 | **M5's head.** `cvc-identity-constraint` and `cvc-id`. Largest remaining `instance` mover in the queue; its enabler (#790) is on `main`, its grounding (comment 5300735646) is durable and scoped to the rule text, and its XPath subset is §3.11.6.2/§3.11.6.3's restricted path grammar — it **does not wait on M6**, and a fail-open to the M6 evaluator here would be wrong rather than conservative. Its stale lease branch reads `CLAIMED`; see the note above before acting on that |
| 2 | #716 | `xsi:type` and `xsi:nil`. #790 **raised** its value: an `xsi:type` decline used to cost one element and now costs a subtree, since everything below an unassessed node inherits that. Also the residual #718 would otherwise GAP-mark. Dependency-free |
| 3 | #800 | the `<xs:alternative>` producer gap. It **cannot move `instance`** — it converts three false rejects into declines, which the lane scores identically — but it is the only queue item that retires a decided-and-WRONG case, may move `schema` through four readers it makes live, and a `schema` regression is the real risk. Needs a warden pre-flight on `xsd.TypeAlternative` |
| 4 | #733 | **promoted four places.** A top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and **#442 moved `schema` +82**. It was banded below five rows that move no lane at all, which is precisely what four passes of carrying produced |
| 5 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced. Its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `<list>` and `<union>`; converting it turns declines into decisions in `schema` |
| 6 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. Below #733/#786 because it converts silent wrongness into honest declines, which the lane scores the same. It also decides the encoding **#56** needs one milestone later |
| 7 | #625 → #669 → #748 → #492 | **README's Library block, now ONE row in file order.** Splitting it across two band rows is why it sat four passes. #625 fixes the `SchemaBuilder` pointer at closed #203 (:123-124); #669 the "works TODAY" snippet, the example list and `go doc ./...`; #748 the M5 block that denies a shipped API (:126-133); #492 `ParseReport`, which belongs in the sentence at `README.md:116` rather than a new paragraph |
| 8 | #747 + #514 + #687 + #672 | the CLI contract, all four decided **before** #472 — the missing "Implemented today" paragraph, typo-vs-unbuilt, scoped help, `-version`. Each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 9 | #472 | implement `goxsd8 parse`, the first non-stub subcommand — and the issue that owns the exit-2 split, plus the forward-compat sentence folded onto it |
| 10 | #682 + #668 + #802 | **the container tax, actionable half.** #682 and #668: `wipsurvey` and `gapaudit` both have a working no-`gh` mode documented only in their own package docs, so the three places a session reads show a dead `gh issue list \|` pipeline. #802: the clone is shallow and nothing says so. Both of the last two full passes paid all three by hand. **#659** and **#527** are the prose half of the same paragraph and land with it |

**Deliberately unbanded, and why.** **#744** (chained-redefine successor, highest-
value M4 gap), **#773** and **#721** are held out on one shared condition: each
needs a warden pre-flight on an entry point that does not exist. **That condition
has an owner — #484** — which is why a process issue is worth more than its label
suggests here: it unblocks three band-grade issues at once. **#774** is three
fail-open declines to close or rule permanent; which way it moves a lane is
unknown until the first is decided, and **#795** pins the element-side one either
way. **#771** is the last decided-and-disagreeing `instance` case once #800 lands.
**#782**, **#783**, **#785**, **#787**, **#788**, **#793**, **#794**, **#796** are
earlier landings' follow-ups. **#805** and **#806** are this landing's and join
them: both are `tools/wipsurvey` hygiene on a tool that now works, and #806 wants
a measurement before it wants code.

**`parser/redefine.go` has been rewritten five times in a week, so read it rather
than an issue body that describes it.** #686, #699, #506, #503 and #504 each
moved it; #744, #706 and #726 all open it next.

### Next planning action

**Re-derive the decline census before the next carve.** It predates #766, #715,
#740 and #790 — four landings that moved 985 cases between them — and it is now
the oldest measurement this roadmap argues from. Rows 4, 5 and 6 of the band
above are ordered on *reasoning* about which declines are convertible, not on a
measurement, and that is as far as judgment can carry the ordering. **#570** is
the issue that makes it cheap and permanent: bank a per-lane decline baseline so
every landing announces the cases it just made decidable, instead of each pass
re-running a standing count. **#571** is its soundness half.

**Reconcile the queue count, do not re-derive it.** The 195 → 193 gap above is
one unexplained issue, and a fresh count that happens to agree with neither
figure will not settle it.

The follow-up-ledger debt this section carried for eleven stamps is **discharged
and closed** (#489); do not re-file it. WORKFLOW step 7(b)'s *a hand-off is not
a disposition* rule is what closed the inflow, and **#400** — a pass that files
no docs commit leaves no signal on `main` — is the mechanism that let #489's
false premise stand that long. It is still open and it is still the reason a
skipped pass and a clean one look alike.

Everything else this queue needs is a develop iteration. **Start with #718.**


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

The original carve (#167–#183) is landed. Open work is the long tail of
producer widening, finalize validity and composition edge cases. The
GitHub milestone holds the feature slices; the comment-accuracy, doc and
process issues that post-land passes file against the same packages sit
outside it, so the milestone is a floor on M4's remaining work and not
the whole of it.

### M5 — Instance validation (XML) — [epic #250](https://github.com/kud360/goxsd8/issues/250)

`validate` engine plus `validate/xmlsrc`: greedy deterministic matching,
identity constraints, `xsi:type`/`xsi:nil`, wildcards, default and fixed
values. **`instance` lane.**

Carved 2026-08-12 into ten slices, #710–#719, and now **seventeen** — #766 was
split out of #714 by a warden pre-flight, #773, #774 and #775 out of #766 by
another, #782 and #783 out of #715 by its own, and **#790** filed from #715's
warden verdict, which had named the seam and been read by nobody who filed it.
Nine have landed: the infoset
seam and engine skeleton (**#710**), the `xmlsrc` adapter (**#711**) and root
dispatch (**#712**), none of which moved the lane and correctly so — the first
two decide no `cvc-` rule and nothing yet ran the cases; **the lane driver
(#713), which took `instance` off zero — 0 → 18 pass** (the #175 analogue, placed
fourth so every slice after it reports a real number); `cvc-complex-type`
clauses 2–3 (**#714**, 19 → 29); the `value.Backend` seam with the four
datatype-valid attribute charges (**#766**, 29 → 193); the greedy
non-backtracking content matcher (**#715**, 193 → 520); `cvc-complex-type`
clause 1.2's ·initial value· against String Valid (**#775**, 532 → 535), which
also closed #759 and landed `docs/STYLE.md` **E4**; and **the descent (#790,
535 → 1017)**, which threads each descendant's ·context-determined declaration·
into the recursive walk (§3.3.4.6 clause 3.1) — the largest single move any lane
has recorded. `instance` stands at **1017** — #740, an M4 landing, took it
520 → 532 on a merged tree neither parent could measure.

**The milestone's shape changed with #790, not just its number.** The first eight
slices decided the ·validation root· and nothing else, so each one bought tens of
cases; #790 bought 482 by making the same rules reach the other 99% of every
document. **The remaining slices inherit that multiplier**, which is why #718 and
#716 are the queue's top two: they no longer buy a rule at one node, they buy it
at every node.

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

**1017 is still a floor built for soundness, and the jump to it did not change
what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every one of the 482 is an
expected-INVALID case**, and the ~25,300 that still fail are overwhelmingly
declines rather than disagreements. The milestone's remaining slices are what
turn declines into decisions.

**Do not read 1017 as 4% of the suite passing.** It is the count of documents
this engine can honestly call not-valid, and it grew because the same six rules
now reach every node instead of one. A slice that decides a *new* rule will move
the number far less than #790 did and be worth more.

**Four cases are decided and decided WRONG**, not one: **#771** (a root whose
declaring schema is reachable only through the instance's own
`xsi:schemaLocation`) plus the three `<xs:alternative>` false rejects **#800**
owns — `Assert/assert_019_2`, `CTA/cta0008.v01`, `CTA/typeAlternatives_001_2`.
The three were already banked `fail` before #790 and are not its regression; the
descent is what made them visible. #800 returns them to honest declines, which
the lane cannot register as movement.

The decline census that separated harvest candidates from indeterminates
predates #766, #715, #740 and #790 — four landings that moved 985 cases between
them — and is not re-derived here. **It is now the oldest measurement this
milestone still argues from**, and #786 is the nearest issue to it.

The design constraints are fixed by `validate/doc.go` and PRINCIPLES 8, 11,
13, 14 and 15, and the carve does not reopen them.

### M6 — XPath required subset

CTA restricted subset plus assertion essentials, fail-open with `GAP`
markers, IDC selector and field paths. Dynamic-error direction per
PRINCIPLES 20. Not filed as an epic — speculative epics two milestones out
earn nothing.

**#56** (a per-assertion or CTA result must distinguish a genuine PASS from
a fail-open "unevaluated") stays blocked on the not-yet-filed evaluator
issue, but its design question is no longer M6's alone: **#719** needs the
same distinction one milestone early, because `cvc-assertion` is wired
fail-open in M5 and the `instance` lane must decline every case whose
outcome turns on an assertion. One encoding, decided in #719 and reused
here (STYLE D4). STYLE 9's fail-open discipline is only honest if a
fail-open answer is distinguishable from a real pass.

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
