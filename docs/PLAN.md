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
`instance` is M5, active, and no longer at zero; `xpath`, `json` and `ber` wait
on M6/M7, M8 and M11.

**No lane moved, because nothing landed.** `main` is still `dd4f1d8`, the
post-land pass for #766 — the four `instance` landings that took the lane 0 → 193
are described in the previous stamp and are not re-narrated here. This is a full
backlog run, not a post-land pass: the queue was audited body-by-body where it
was cheap to do so, the branch namespace was re-surveyed, and two persona reports
were folded in.

**`schema`'s number is stale on `main` and correct in this table**, which is a
distinction worth holding. `origin/wip/issue-740` carries an accepted
**`schema` +1197 (11635 → 12832 pass, 3763 → 2566 fail)** — the arbiter's own
`GOXSD_RATCHET=1` bank, machine-written, 1197 cases all flipping fail → pass with
no other lane moving. It is the largest single `schema` movement this project has
recorded and **it is not on `main`**, so `lanestatus` cannot see it and neither
can this table. See the branch namespace section.

Milestones, from GitHub at the previous stamp and **unchanged by this pass** —
no issue closed, no milestone was reassigned, and the one issue filed here
(#779) deliberately carries none:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 82 | 42 | **active** |
| M5 — Instance validation (XML) | 6 | 13 | **active** — six slices landed |
| M6–M12 | 0 | 0 | not filed |

These four rows are the one part of this section not re-read from GitHub today,
and the reason is stated rather than hidden: the MCP issue tools expose no
milestone filter and `gh` is 403 (#527), so the counts are the 07:14 stamp's
GitHub reads carried forward across a provably empty delta. Every other figure
below was re-read.

Queue: **187 open issues — 164 `ready`, 23 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `blocked`), against 281 closed. **Every open issue carries a queue label**
— the #773/#774 defect the last pass found is not present today, checked across
all 187 rather than assumed. Read the milestone table as feature progress, not as
the queue: 132 of the 187 carry no milestone at all, and the process, doc and
comment-accuracy issues that post-land passes file are deliberately outside them.

### What this pass did

**The unblock sweep found nothing to unblock, and that is a verified result
rather than a consequence of nothing having landed.** All 23 `blocked` bodies
were read and their `## Depends on` lines checked against live issue state. Every
one names either an open issue (#414 for #438, #455 for #456, #591 for #593,
#407 for #415, #250 for #248/#267/#345, #715 for #716–#719, #472 and #715 for
#720, #472 for #16) or a **trigger** — six bodies (#548, #555, #622, #681, #692,
#696) name the next `/retro` process audit and say in their own words not to
re-scan them. Two epics (#79, #250) are dependency targets. Nothing is startable
that is not already labelled so.

- **#584 is `blocked` and names no dependency at all** — its body ends at
  `## Notes` with no `## Depends on` section. `blocked` means waiting on a named
  dependency in that section, so an issue carrying the label and no payload is
  **structurally unreachable by every future unblock sweep**: the sweep reads
  those lines and correctly concludes there is nothing to check. Its real gate is
  **#414**, recoverable only from its Notes prose (#438's fold-ordering argument,
  which #584 inherits). Recorded in a comment on the thread; the body was not
  edited, because the MCP write path strips angle-bracket tokens (#515, #764) and
  would delete every element name in it.
- **#779 was filed for the class.** Three consecutive passes have now found a
  queue-invisible issue by hand — #773/#774 with no queue label, #584 with no
  dependency, and #715/#766 found only by reading all 25 `blocked` bodies. The
  classes are cheap to check and expensive to miss, which is PRINCIPLES 27's
  exact shape, and `wipsurvey` and `gapaudit` are the precedent. #779 reports and
  never relabels: #722 is the standing proof that a mechanical verdict acted on
  without reading the thread damages the queue.
- **Two persona reports were folded in, and each finding was reproduced against
  the tree before it was acted on.** Ten findings — five from libuser, five from
  cliuser. **Two** produced a correcting comment (#748), **three** produced
  confirmations or an added acceptance probe (#409 twice, #625), **three** were
  declined because they reproduce exactly as already filed with nothing to
  correct (#514, #687, #672), **one** was declined as a persona misreading
  (#747), and **one** as a duplicate of an existing Acceptance clause (#472).
  **No finding produced a new issue**, which is the expected shape for a surface
  four passes have already audited — the two that survived scrutiny were both
  corrections to issues that already existed.
- **The `#747` re-scope was refused.** cliuser reported it "PARTIALLY fixed, but
  in the WRONG artifact" — `go doc ./cmd/goxsd8` carries an "Implemented today"
  paragraph that `-help` does not. Nothing was fixed: **#251 added that paragraph
  on 2026-08-02**, ten days before #747 was filed, and #747's Acceptance already
  quotes it as the text to port into the `usage` const. The persona read a
  pre-existing asymmetry as a partial repair because it has no commit history,
  which is correct persona discipline and exactly why findings are reproduced
  before filing. The proposed narrowing would also have dropped fact 2 —
  `README.md:74-76`, the divergence **#626 introduced** — leaving a false README
  sentence with no owner.
- **The exit-2 overload was declined as a duplicate of #472's own Acceptance.**
  cliuser proposed a forward-looking issue, `blocked` behind #472, for exit 2
  meaning both "usage error" and "this binary is a stub". #472 already says it,
  in stronger terms, under *"The exit-2 overload must be resolved here, not
  deferred"*, and already credits cliuser for it. Re-verified at `dd4f1d8`:
  `run` has exactly two exits, and since there is no flag parsing at all, today's
  bucket is not merely coarse but **singular** — which makes the clause cheaper
  than it reads, because there is no usage-error behaviour to preserve.
- **#748 gained the correction that matters most**: its own Acceptance quotes
  `func New(schema *xsd.Schema, opts ...Option)`, and **#766 added the required
  second positional `backend value.Backend`**. Its prescribed replacement text is
  stale too — it says "no source adapter exists", and **#711 landed `xmlsrc` the
  same day #748 was filed**. A third snippet defect it never named: the README
  line `validate.New(schema)` is missing the backend argument, so the snippet
  will not compile even after both filed facts are repaired.
- **#409 gained a third independent libuser sighting and one addition.**
  `go doc ./codegen Generate` still answers "no symbol"; the root `doc.go` says
  codegen is *"(eventually)"* while `codegen/doc.go` says it *"emits"*. The root
  doc is the correct one, so the contradiction dissolves when #409 lands — filed
  as an acceptance **probe**, not a new issue.
- **#625 and #514/#687/#672 reproduce exactly as filed** and needed no
  correction. #625 got a one-paragraph second-sighting note (a defect two
  isolated consumers find five days apart is not a matter of taste); the three
  CLI-contract issues got nothing, because three comments saying "still true" is
  noise.

**`gapaudit` was not re-run, and the reason is that its answer is provably
unchanged**: the tree is byte-identical to the head the 07:14 pass audited, so
its 44 markers in 5 areas and its "no real leak" verdict stand. Worth one line
for whoever reaches for a grep instead — the naive `grep -rn "GAP("` returns
**77** lines, six of them `GAP(real)` and `GAP(oops)` inside `gapaudit`'s own
test fixtures. That is the tool's whole reason for existing (PRINCIPLES 27).

`gh` returned 403 on **both** REST and GraphQL in this container again, so every
GitHub read and write here went through the MCP server and `wipsurvey` was fed
hand-built JSON. That is **#527**, now at eleven-plus consecutive sessions, with
#668 and #682 owning the downstream consequence.

### Branch namespace, `origin` — report-only; a session never deletes a ref

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-740` | EXPIRED (6h28m) — **mid-landing, and the one to read first** | Three commits: the implementation `68c7a7f`, the arbiter's own ratchet bank `c415bf9` (**`schema` +1197**), and a merge-forward. Age is genuine — the branch has commits of its own, so #722 does not apply. An accepted verdict on an unlanded branch, so #740 is **not** a band row, on #714's precedent. Recovery is a resume, not a re-implementation. |
| `wip/issue-715` | EXPIRED (6h49m) — **claim dead, grounding alive** | **Zero commits ahead of `main`**: a lease. The `wipsurvey` age is the figure #722 says not to trust; the thread's own clock is what settles it — last comment 07:56 UTC, six and a half hours, no RESUME. **Not retired**: hours, not days, and this is the queue's highest-value M5 slice. |
| `wip/issue-256` | RETIRED | Issue closed `needs-replan`, superseded by #470. |
| `wip/issue-271` | RETIRED | Issue closed `needs-replan`, superseded by #478/#479/#480. |
| `wip/issue-287` | RETIRED | Issue closed, answered in the opposite direction by PR #511. Not a park — it carries no `needs-replan`. |
| `claude/eloquent-cerf-7ckw7b` | merged | 0 commits ahead of `main`. Deletable. |
| `claude/eloquent-cerf-8f1mrv` | merged | 0 commits ahead of `main`. Deletable. |
| `claude/eloquent-cerf-patxs3` | superseded | 253 commits ahead by SHA, none by content — tip `2196c3f`, unmoved since 2026-08-07, content reached `main` as squash commits. Deletable. |

No `parked/*` branches. `origin/wip/issue-766` is gone, deleted at merge, which
is what a landed branch is supposed to do.

**The two EXPIRED branches fail in opposite directions and want opposite
responses.** #740's expiry hides *finished, judged, banked* work — the risk is
that it is silently redone. #715's expiry hides *nothing*, because the branch is
empty — the risk is that its issue is retired and its grounding lost. A single
EXPIRED verdict cannot distinguish them; only the commit count and the thread
can. This is the sharpest instance of #722's hazard yet recorded, and both halves
of it appeared in one `wipsurvey` run.

**#715's pre-flight survives its claim, and a session must not re-run it.** The
expired session produced no code but did produce an oracle grounding across
§3.4.4.2, §3.4.4.3, §3.9.4.2, §3.9.4.3 and §3.8.4.3 with the load-bearing
cvc-accept Note quoted in full, plus a **warden verdict that rejected #715's own
`## Notes` section** and approved a different named shape:
`xsd/contentmatcher.go`, five exported identifiers, a counting walk rather than
automaton unfolding, `(*Schema).ContentMatcher` as a method so a
pre-`cos-nonambig` matcher is unrepresentable. Whoever takes #715 implements the
grounding comment, not the body.

**Read a zero-commit `wip/issue-N` against its issue thread before believing an
EXPIRED verdict.** This is **#722**: `wipsurvey` dates a lease-only branch from
the *previous* landing's committer date, and the prescribed response to EXPIRED
is `needs-replan`. A pass following it blindly retires a live session's issue.
The caution stands until #722 lands.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
list. Take from the top; the ordering prefers slices that move a lane over
horizontal completeness.

| # | Issue | Why here |
|---:|---|---|
| 1 | #715 | the greedy non-backtracking content matcher — the only band row that unblocks five others (#716, #717, #718, #719, #720), and by its own Acceptance the largest single move M5 has left. **The claim on it expired with zero commits and the pre-flight banked**, so it is now the queue's cheapest high-value row rather than a contended one |
| 2 | #775 | `cvc-complex-type` clause 1.2, the simple-content charge #766 split off — an `instance` mover whose whole seam is already on `main` |
| 3 | #659 + #527 | the environment tax, one sentence of prose each. Both bodies were corrected on 2026-08-13; #527 was paid again by this pass, as it has been by every pass for eleven sessions |
| 4 | #733 | a top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and #442 moved `schema` **+82** |
| 5 | #748 + #747 | the two persona findings, both re-verified and both corrected on their threads today. #748 is now four facts, three of them one-line edits; #747 is two facts and its re-scope was refused |
| 6 | #625 → #669 → #492 | the rest of README's Library block, in that order — #625's fix names the file #669 must add to the example list, and #492's belongs in the sentence at `README.md:115-116` rather than a new paragraph |
| 7 | #514 + #687 + #672 | the open CLI-contract decisions: typo-vs-unbuilt, scoped help, and `-version`. All three re-reproduced today against the built binary. Take them **before** #472 — each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 8 | #472 | implement `goxsd8 parse`, the first non-stub subcommand — and the issue that owns the exit-2 split |

**#715 moves from "CLAIMED, read it before taking it" to the top row outright.**
The previous band marked it contended and told a session unwilling to read a
thread to take row 2 instead; that advice is withdrawn. The claim is dead, the
branch is empty, and the grounding plus warden verdict are banked on the thread —
which makes #715 *less* expensive than any other row of comparable value, not
more.

**#740 is deliberately not banded** despite being the largest pending lane
movement in the project. It has an accepted verdict and an unlanded branch, which
#714's post-land pass established is not a band row: the band exists so a session
can take a **startable** issue, and banding judged work invites a second session
onto it. Row 5 swapped #748 ahead of #747 because #748 now carries a correction
its body does not, and reading the thread first is cheaper there.

**Rows 3, 4, 6, 7 and 8 are unchanged in substance from the previous band**, with
their justifications intact: nothing landed to touch any of them.

**#773 and #774 stay `ready` and unbanded.** #773 needs a new optional capability
interface for `[unparsedEntities]` and so its own warden surface pass before it
is workable — the same condition holding #744 and #721 out. #774 is three
fail-open declines to close or rule permanent; it can move a lane, but which way
and by how much is unknown until the first is decided, so it is not ordered
against slices whose movement is predictable.

**Deliberately unbanded, and why.** **#744** — the chained-redefine successor —
is the highest-value M4 gap in the queue and is held out only because its
`## Surface` requires a warden pre-flight on an `xsd` entry point that does not
exist. **#722** is the one to promote if a later pass wants a cheap high-value
row: it is still the only unbanded issue whose absence can cause a future pass to
actively damage the queue, and this pass met **both halves** of its hazard in one
`wipsurvey` run. **#753** was settled by the previous pass — `ready`, unbanded,
nothing further owed — and is not re-litigated here. **#779** is this pass's own
filing and is not banded on its first appearance. **#743**, **#742**, **#741**,
**#731**, **#732**, **#726**, **#725**, **#702**, **#703**, **#706**, **#755**,
**#756**, **#759**, **#763**, **#764**, **#768**, **#771**, **#772** and **#777**
are this month's post-land follow-ups: real, filed, and none of them moves a
lane. The remaining `blocked` M5 slices cannot be banded at all.

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
advisory, one comment per item — before the next carve. It has now gone eight
consecutive passes undone, it grows with every landing, and #330 is the standing
proof that a hand-off tracks nothing. This pass disposed of nine persona findings
and filed one issue, which is the shape the sweep is for and is not a substitute
for running it.

Everything else this queue needs is a develop iteration, not a planning one.
**Start with #740** — not because it is a band row, which it deliberately is not,
but because a judged, banked `schema` +1197 sitting off `main` is the largest
recoverable value in the repository and it decays: the branch is two commits
behind already, and every landing widens the merge it will need. If it is
unclaimed, resume it; **#715 is the top row for anyone starting fresh.**
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
