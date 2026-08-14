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

## Status — 2026-08-14

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 193 | 26168 | 26361 |
| `json` | — | — | 0 |
| `schema` | 11635 | 3763 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5, active, and **no longer at zero**; `xpath`, `json` and `ber`
wait on M6/M7, M8 and M11.

**`instance` is the lane that moved, and it moved by 193 in four landings**,
each figure copied from its commit's `Ratchet:` line rather than recomputed:
**#713** 0 → 18 (the lane driver), **#761** +1, **#714** 19 → 29
(`cvc-complex-type` clauses 2–3, the attribute-existence half), and **#766**
29 → 193 (**+164**, the `value.Backend` seam and the datatype-valid charges).
`schema` (11635) and `datatypes` (1161) have not moved since the previous stamp
and were not meant to.

**What moved the lane is not what #766 was named for.** 149 of its 164 flips are
`MS-Regex2006-07-15` — single-attribute negative documents against an anonymous
`xs:string` restriction carrying one `<xs:pattern>` — so the largest `instance`
movement to date is the `regex` package arriving in `validate`'s import closure
behind `value`, through cvc-attribute clause 3's pattern stage. The fixed-value
comparisons the issue is titled for account for a handful, and the ·defaulted
attribute· charge for none, by construction. Expect the next charging slice's
figure to be dominated by whichever facet stage it newly reaches rather than by
its headline rule.

Milestones, from GitHub:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 82 | 42 | **active** |
| M5 — Instance validation (XML) | 6 | 13 | **active** — six slices landed |
| M6–M12 | 0 | 0 | not filed |

M5's row moved by four without a landing behind the move: **#766, #773, #774 and
#775 carried no milestone at all**, and all four are M5 work filed by #766's own
session. M0–M2's row is the one figure here not re-read from GitHub — all three
are closed with no open issue and nothing has been filed against them since M2
finished.

Queue: **186 open issues — 163 `ready`, 23 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `blocked`). **55 of the 186 carry a milestone; 131 carry none** — every
milestone-carrying open issue is M4 (42) or M5 (13), and the process, doc and
comment-accuracy issues that post-land passes file are deliberately outside them.
Read the milestone table as feature progress, not as the queue.

### What this pass did

A post-land pass for **#766** (PR #776, `43b421a`), not a full backlog run: this
section is re-derived whole, but no body-by-body audit of the 163 `ready` issues
was run, no issue was closed and no duplicate merged.

- **Two issues were flipped to `ready`, and neither was `blocked`.** #773 and
  #774 were filed mid-session by #766's landing carrying **no queue label at
  all** — not `ready`, not `blocked` — so they were invisible to both queues
  while the only dependency they name closed under them. **This is a different
  failure from the one the last three passes checked for**: the unblock sweep
  reads `blocked` bodies, and an issue in neither queue is not in that sweep's
  input. Both also carried no milestone, as did #766 and #775; all four are now
  M5.
- **Nothing was unblocked in the ordinary sense, and that is checked rather than
  assumed.** All 23 open `blocked` bodies were read for a `## Depends on` naming
  #766; **none does**. #716 mentions #714 in its Acceptance but depends on #715,
  and #716–#719 all wait on #715 rather than on each other.
- **One follow-up was filed: #777.** The warden's #766 diff review noted, under
  *"NOT held against the landing"*, that `TestNonDefaultedAttributesEscapeClauseFour`'s
  comment claims four ·defaulted attribute· conjuncts and exercises three —
  key-dflt-att clause 4 (`isInstanceAttribute`) is uncovered. The repair round
  `a2f3099` took the three FINDINGS only and the landing shipped with it. Filed
  rather than carried, per #330.
- **`gapaudit` reports no real leak**, over 44 markers in 5 areas. All three new
  `GAP(validate)` markers in `validate/cvcattribute.go` pair with #773 and #774.
  Group 1's two entries — `validate/assess.go:129` and `:318` — cite **#56** and
  **#717**, both open and both simply not labelled `kind/gap`, so the tool never
  had them in its input. That is a heuristic miss in a matcher whose own doc says
  it is one, not untracked debt. Group 2's eight entries are the permanent kind:
  conformance-lane and test-coverage gaps that never carry a marker (#398, #404,
  #569, #591, #592, #593, #740) plus #719, whose marker lands with its M6 seam.
- **#714's post-land export debt is discharged and needs no filing.**
  `(*Schema).ResolvedAttributeDeclaration` landed with no consumer outside `xsd`
  (STYLE T5); `matchedAttribute` and `defaultedAttributes` now both call it. The
  obligation lived in a comment on #766's thread which said outright that no
  tracking issue was filed and that the landing discharges it. It did.
- **#462's carried note was corrected in the landing, not by this pass.** The
  union residue's comment said closing it needs a change to `dispatchUnion`'s
  error model; `value.IsDatatypeVerdict` **is** that change, and what is left is
  ratchet attribution alone. Whoever takes #462 should expect verdict movement in
  `datatypes` as well as `instance` (PRINCIPLES 22).
- **#753's band carry-forward is retired here and is not carried again.** Four
  LOG entries carried its decline-census trigger and the 2026-08-13 pass showed
  that trigger could never fire; the residue — *"still queue-eligible and unfiled
  against the band"* — is a band decision, and the band is this file's. The
  decision: **#753 stays `ready` and unbanded.** It is a fail-open that can never
  falsely reject, so it costs no lane today, and it is startable exactly as
  filed. Nothing further is owed.

`gh` returned 403 on **both** REST and GraphQL in this container again, so every
GitHub read and write here went through the MCP server and both survey tools were
fed hand-built JSON rather than the pipelines CLAUDE.md prints. That is #527,
with #668 and #682 owning the downstream consequence.

### Branch namespace, `origin` — report-only; a session never deletes a ref

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-715` | EXPIRED (2h TTL) — **do not act on it** | **Zero commits ahead of `main`**: a lease, not work. The thread carries an oracle grounding and a warden surface pre-flight, both 2026-08-14 07:56, and the ref was pushed minutes later. #722's exact shape — read the thread, never the verdict. |
| `wip/issue-740` | EXPIRED (2h TTL) — mid-landing | Three commits: the implementation, the **arbiter's own ratchet bank** (`c415bf9`, `schema` lane) and a merge-forward of this landing. An accepted verdict with an unlanded branch, so #740 is **not** a band row, on #714's precedent. |
| `wip/issue-256` | RETIRED | Issue closed `needs-replan`, superseded by #470. |
| `wip/issue-271` | RETIRED | Issue closed `needs-replan`, superseded by #478/#479/#480. |
| `wip/issue-287` | RETIRED | Issue closed, answered in the opposite direction by PR #511. Not a park — it carries no `needs-replan`. |
| `claude/eloquent-cerf-7ckw7b` | merged | 0 commits ahead of `main`. Deletable. |
| `claude/eloquent-cerf-8f1mrv` | merged | 0 commits ahead of `main`. Deletable. |
| `claude/eloquent-cerf-patxs3` | superseded | **253** commits ahead by SHA, none by content — tip `2196c3f`, unmoved since 2026-08-07, and its content reached `main` as squash commits. Deletable. |

`origin/wip/issue-766` is gone, deleted at merge, which is what a landed branch
is supposed to do. No `parked/*` branches.

**The previous section recorded `claude/eloquent-cerf-patxs3` as 53 commits
ahead; it is 253.** The tip has not moved since 2026-08-07 and `main` advancing
can only lower that count, so the old figure was wrong when it was written. The
verdict it supported is unchanged — the count is by SHA and means nothing against
a squash-merged history — which is exactly why nobody caught it.

**Read a zero-commit `wip/issue-N` against its issue thread before believing an
EXPIRED verdict.** This is **#722**: `wipsurvey` dates a lease-only branch from
the *previous* landing's committer date, so a claim made minutes ago reports
EXPIRED, and the prescribed response to EXPIRED is to label the issue
`needs-replan`. A pass following it blindly retires a live session's issue in the
hour its design is approved. `wip/issue-715` is that case today. The caution
stands until #722 lands.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
list. Take from the top; the ordering prefers slices that move a lane over
horizontal completeness.

| # | Issue | Why here |
|---:|---|---|
| 1 | #715 | the greedy non-backtracking content matcher — the only band row that unblocks five others (#716, #717, #718, #719, #720), and by its own Acceptance the largest single move M5 has left. **CLAIMED**: grounding and warden pre-flight on the thread, lease-only branch. Read it before taking it (#722) |
| 2 | #775 | `cvc-complex-type` clause 1.2, the simple-content charge #766 split off — an `instance` mover whose whole seam is already on `main`, and **the top unclaimed row** |
| 3 | #659 + #527 | the environment tax, one sentence of prose each. Both bodies were corrected on 2026-08-13 and neither was actionable as filed; #527 was paid again by this pass |
| 4 | #733 | a top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and #442 moved `schema` **+82** |
| 5 | #747 + #748 | the two persona findings the 2026-08-13 pass filed. #747's second fact is a divergence **#626 introduced**; #748's snippet will not compile against the signature `xmlsrc/doc.go` already commits |
| 6 | #625 → #669 → #492 | the rest of README's Library block, in that order — #625's fix names the file #669 must add to the example list, and #492's belongs in the sentence at `README.md:115-116` rather than a new paragraph |
| 7 | #514 + #687 + #672 | the open CLI-contract decisions: typo-vs-unbuilt, scoped help, and `-version` |
| 8 | #472 | implement `goxsd8 parse`, the first non-stub subcommand |

**Two rows left the band and only one of them landed.** #766 was row 2 and is
closed. **#740 was row 3 and is mid-landing** — its branch carries the arbiter's
own ratchet bank, so it has an accepted verdict and an unlanded branch, which
#714's post-land pass established is not a band row: the band exists so a session
can take a **startable** issue, and banding one with a live claim invites a second
session onto it. **#715 is the case that precedent does not cover** — a claim
with no verdict — and it is banded anyway, marked, because a lease-only branch is
the one thing #722 says may be nothing at all. A session that will not read a
thread should take row 2.

Rows 3–8 are the previous band's rows 4–9, promoted by two, with their
justifications unchanged: nothing this landing touched any of them.

**#773 and #774 are `ready` and deliberately unbanded**, though this pass flipped
both. #773 needs a **new optional capability interface** for `[unparsedEntities]`
and so its own warden surface pass before it is workable — the same condition
holding #744 and #721 out. #774 is three fail-open declines to close or rule
permanent; it can move a lane, but which way and by how much is unknown until the
first is decided, so it is not ordered against slices whose movement is
predictable. **#777** is a one-line comment correction and is not a band row.

**Deliberately unbanded, and why.** **#744** — the chained-redefine successor —
is the highest-value M4 gap in the queue and is held out only because its
`## Surface` requires a warden pre-flight on an `xsd` entry point that does not
exist. **#722** is the one to promote if a later pass wants a cheap high-value
row: it is still the only unbanded issue whose absence can cause a future pass to
actively damage the queue, and this pass met its hazard twice in one `wipsurvey`
run. **#753** is settled above. **#743**, **#742**, **#741**, **#731**, **#732**,
**#726**, **#725**, **#702**, **#703**, **#706**, **#755**, **#756**, **#759**,
**#763**, **#764**, **#768**, **#771**, **#772** and **#777** are this month's
post-land follow-ups: real, filed, and none of them moves a lane. The remaining
`blocked` M5 slices cannot be banded at all.

**#759 is the one unbanded follow-up with an owner by default.** Violation
messages have two forms for naming a sub-clause and STYLE picks neither;
`validate/assess.go:82` is the only non-test site using the leading
`"clause 2: "` label. It needs no session of its own — **the next M5 charging
slice carries it**, which is #715 or #775, whichever lands first, and both
threads say so.

**`parser/redefine.go` has been rewritten five times in six days, so read it
rather than an issue body that describes it.** #686, #699, #506, #503 and #504
each moved it; #744, #706 and #726 all open it next.

### Next planning action

**Sweep the carried follow-up ledger (#489)** — file or dismiss each unfiled
advisory, one comment per item — before the next carve. It has now gone seven
consecutive passes undone, it grows with every landing, and #330 is the standing
proof that a hand-off tracks nothing. This pass filed one such advisory (#777)
and retired one carried band note (#753), which is the shape the sweep is for and
is not a substitute for running it. Everything else this queue needs is a develop
iteration, not a planning one — **starting with #775**, the band's top unclaimed
row.

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

Carved 2026-08-12 into ten slices, #710–#719, and now **fourteen** — #766 was
split out of #714 by a warden pre-flight, and #773, #774 and #775 out of #766 by
another. Six have landed: the infoset seam and engine skeleton (**#710**), the
`xmlsrc` adapter (**#711**) and root dispatch (**#712**), none of which moved the
lane and correctly so — the first two decide no `cvc-` rule and nothing yet ran
the cases; **the lane driver (#713), which took `instance` off zero — 0 → 18
pass** (the #175 analogue, placed fourth so every slice after it reports a real
number); `cvc-complex-type` clauses 2–3 (**#714**, 19 → 29); and the
`value.Backend` seam with the four datatype-valid attribute charges (**#766**,
29 → 193).

**The carve is no longer a chain, and reading it as one loses work.** The
original ordering assumed a slice becomes `ready` as the one before it lands.
That held until #714, which had **two** successors directly behind it, and #766
has **three**; #716, #717, #718 and #719 all wait on **#715** rather than on each
other. Relabel from the `## Depends on` lines, never from the milestone or from
the issue numbers' order — and sweep for issues carrying **no queue label at
all**, which is how #773 and #774 sat outside both queues for a day. The CLI's
own `validate` subcommand is #720, `blocked` behind #472 and #715.

**193 is still a floor built for soundness, not a measure of the engine.** The
lane emits only "not valid" observations; a violation-free `Result` DECLINES
rather than passing, because `Assess` evaluates none of `e-validity`'s other
conjuncts. The milestone's remaining slices are what turn declines into
decisions. Exactly one case is decided and decided WRONG (**#771**); the decline
census that separated harvest candidates from indeterminates was taken before
#766 moved 164 cases and is not re-derived here.

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
