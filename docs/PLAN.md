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

## Status — 2026-08-15

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 532 | 25829 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12832 | 2566 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**`schema`'s number is no longer stale on `main`, and the flag the last two
stamps carried is retired.** #740 landed as `6a50a373`, taking `schema`
**11635 → 12832 pass (+1197)** — the largest single lane movement this project
has recorded, now where `lanestatus` can read it. `instance` moved twice since
the last stamp: **193 → 520** with #715 (`ca7966b`), then **520 → 532** with
#740's own merge-forward round, which found 12 cases that existed on neither
parent.

**This is a post-land pass, not a full backlog run.** The lane table, the queue
counts and the branch namespace were re-read from their sources today. The
working band was pruned of what landed but **not re-derived**, and no issue body
was audited that this landing did not touch.

Milestones — M4 and M5 both moved. The rows are the previous stamp's GitHub reads
plus that named delta, because the MCP issue tools expose no milestone filter and
`gh` is 403 (#527); every other figure in this section was re-read:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 83 | 41 | **active** — #740 closed |
| M5 — Instance validation (XML) | 7 | 14 | **active** — #715 closed, #782/#783 filed behind it |
| M6–M12 | 0 | 0 | not filed |

Queue: **191 open issues — 171 `ready`, 20 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `blocked`), against **283 closed**. Four of the 171 are this pass's own
filings. Every open issue carries a queue label. Read the milestone table as
feature progress, not as the queue: 136 of the 191 carry no milestone at all, and
the process, doc and comment-accuracy issues that post-land passes file are
deliberately outside them.

### What this pass did

**The unblock sweep found nothing to unblock, and that is a verified result.**
All 20 `blocked` bodies were read and their `## Depends on` lines checked against
live issue state: **no open issue names #740 as a dependency.** The one body
whose text contains those digits (#622) contains them inside a comment URL, not a
dependency. #715's four dependents were relabelled by its own post-land pass;
#717 (`#715, #248`) and #720 (`#472, #715`) stay `blocked` on a still-open second
dependency, correctly.

**Every follow-up #740 raised is now filed.** Four issues, none of them a
hand-off:

- **#785** — the arbiter's non-blocking T6 finding. `conformance/schema.go:45`
  and `:160-171` still call the `ref=` identity-constraint form silently skipped
  and DECLINED; #240 produces it, `elementDecidable` imposes no condition on it,
  and `:825` and `:1014` of the same file already say so. Pre-existing, but #740
  corrected the sibling claims in those same two paragraphs, so the file is now
  internally inconsistent in a way it was not before.
- **#786** — `simpleTypeDecidable`'s last decline is conservative rather than
  forced. Named by both the mason report and the arbiter's Note 1, and correctly
  kept out of #740 so its +1197 stayed attributable. Measure the widening; if it
  is flat, answer whether the function still earns its recursion.
- **#787** — `src-enumeration-value` (§4.3.5.3) versus
  `enumeration-valid-restriction` (§4.3.5.5) at schema construction. Filed from
  the arbiter's Note 2 and **reproduced, which corrected that note's premise**:
  `Produce` does not accept an out-of-value-space enumeration member, it rejects
  it under the other rule ID. A grounding question, not a missing check.
- **#788** — WORKFLOW's *"only the base moved → a gate-only round"* class does not
  say what to do when gate part 4 surfaces movement only the merged tree has.
  #740's second round hit exactly that, banked `instance` +12, and charged itself
  a `[process]` finding for it. The action was right and the brief was wrong.

The arbiter's third note — the absent `docs/LOG` entry — was landing precondition
1 and was discharged by the landing itself. Nothing is owed for it.

**#584's structural defect is unchanged**: it carries `blocked` with no
`## Depends on` section at all, so every future unblock sweep reads past it. Its
real gate is #414, recorded only on its thread; #779 owns the class. Not
re-litigated here.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`origin/wip/issue-740` and `origin/wip/issue-715` are both gone**, deleted at
merge, which is what a landed branch is supposed to do. **No `wip/*` branch holds
a live claim** — the three that remain are all RETIRED, and the whole namespace
is now stale refs and human triage.

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-256` | RETIRED | Issue closed `needs-replan`, superseded by #470. |
| `wip/issue-271` | RETIRED | Issue closed `needs-replan`, superseded by #478/#479/#480. |
| `wip/issue-287` | RETIRED | Issue closed, answered in the opposite direction by PR #511. Not a park — it carries no `needs-replan`. |
| `claude/eloquent-cerf-8f1mrv` | merged | 0 ahead of `main`, 43 behind. Deletable. |
| `claude/eloquent-cerf-patxs3` | superseded | 253 ahead by SHA, none by content — tip `2196c3f`, unmoved since 2026-08-07. Deletable. |
| `claude/eloquent-cerf-7ckw7b` | **needs a human look** | **7 ahead, 51 behind** — measured today, and the last stamp recorded it as 0 ahead and deletable. Do not act on that verdict. The seven commits name #662, #359, #649, #426, #358, #643 and #636, whose work is believed to be on `main` as squashes, but no commit on `main` carries those trailers and this pass did not verify the content diff. |

No `parked/*` branches.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
list. Take from the top. **This band was pruned, not re-derived** — #715 landed
out of row 1 and #740 out of the unbanded set; the next full `/backlog` owes a
real ordering across the M5 slices #715 unblocked and this pass's four filings.

| # | Issue | Why here |
|---:|---|---|
| 1 | #775 | `cvc-complex-type` clause 1.2, the simple-content charge #766 split off — an `instance` mover whose whole seam is on `main`, and the highest-value row that moves the active milestone's lane |
| 2 | #716 + #718 + #719 | the three M5 slices **#715 unblocked when it landed**, now `ready` and **unordered against each other**. A session taking one reads #715's thread first — its warden pre-flight and the `Matcher` shape it approved govern all three |
| 3 | #659 + #527 | the environment tax, one sentence of prose each. #527 has now been paid by every pass for twelve consecutive sessions |
| 4 | #733 | a top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and #442 moved `schema` **+82** |
| 5 | #748 + #747 | the two persona findings, both re-verified and corrected on their threads on 2026-08-14. #748 is four facts, three of them one-line edits; #747 is two, and its proposed re-scope was refused |
| 6 | #625 → #669 → #492 | the rest of README's Library block, in that order — #625's fix names the file #669 must add to the example list, and #492's belongs in the sentence at `README.md:115-116` rather than a new paragraph |
| 7 | #514 + #687 + #672 | the open CLI-contract decisions: typo-vs-unbuilt, scoped help, and `-version`. Take them **before** #472 — each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 8 | #472 | implement `goxsd8 parse`, the first non-stub subcommand — and the issue that owns the exit-2 split |

**Rows 3 through 8 are unchanged in substance from the previous band**, with their
justifications intact: nothing landed that touches any of them.

**Deliberately unbanded, and why.** **#744** — the chained-redefine successor — is
the highest-value M4 gap in the queue and is held out only because its
`## Surface` requires a warden pre-flight on an `xsd` entry point that does not
exist; **#773** and **#721** are held out on the same condition. **#722** is the
one to promote if a later pass wants a cheap high-value row: it is still the only
unbanded issue whose absence can cause a future pass to actively damage the
queue. **#774** is three fail-open declines to close or rule permanent — it can
move a lane, but which way is unknown until the first is decided. **#782** and
**#783** are #715's own post-land follow-ups and are not banded on first
appearance; the same holds for **#785**, **#786**, **#787** and **#788** filed
today. **#786** is the one of those four that can move a lane, and a later pass
should weigh it against row 4.

**#759 still has an owner by default** — violation messages have two forms for
naming a sub-clause and STYLE picks neither. The next M5 charging slice carries
it, which is now #775 or one of row 2, and those threads say so.

**`parser/redefine.go` has been rewritten five times in a week, so read it rather
than an issue body that describes it.** #686, #699, #506, #503 and #504 each
moved it; #744, #706 and #726 all open it next.

### Next planning action

**Sweep the carried follow-up ledger (#489)** — file or dismiss each unfiled
advisory, one comment per item — before the next carve. It has now gone nine
consecutive passes undone and grows with every landing; #330 is the standing
proof that a hand-off tracks nothing. This pass filed four issues against one
landing, which is the shape the sweep is for and is not a substitute for running
it.

The other planning debt is **the band's row 2**: #715's landing made three M5
slices startable at once and this post-land pass did not order them. That
ordering is a full `/backlog`'s job and is the first thing it should do.

Everything else this queue needs is a develop iteration. **Start with #775.**

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

Carved 2026-08-12 into ten slices, #710–#719, and now **sixteen** — #766 was
split out of #714 by a warden pre-flight, #773, #774 and #775 out of #766 by
another, and #782 and #783 out of #715 by its own. Seven have landed: the infoset
seam and engine skeleton (**#710**), the `xmlsrc` adapter (**#711**) and root
dispatch (**#712**), none of which moved the lane and correctly so — the first
two decide no `cvc-` rule and nothing yet ran the cases; **the lane driver
(#713), which took `instance` off zero — 0 → 18 pass** (the #175 analogue, placed
fourth so every slice after it reports a real number); `cvc-complex-type`
clauses 2–3 (**#714**, 19 → 29); the `value.Backend` seam with the four
datatype-valid attribute charges (**#766**, 29 → 193); and the greedy
non-backtracking content matcher (**#715**, 193 → 520), the largest single move
the milestone has recorded. `instance` stands at **532** — #740, an M4 landing,
took it the last 12 on a merged tree neither parent could measure.

**The carve is no longer a chain, and reading it as one loses work.** The
original ordering assumed a slice becomes `ready` as the one before it lands.
That held until #714, which had **two** successors directly behind it, and #766
has **three**; #715 had four, of which #716, #718 and #719 became `ready` when it
landed while #717 waits on #248 as well. Relabel from the `## Depends on` lines,
never from the milestone or from the issue numbers' order — and sweep for issues
carrying **no queue label at all**, which is how #773 and #774 sat outside both
queues for a day. The CLI's own `validate` subcommand is #720, `blocked` behind
#472 alone now that #715 has landed.

**532 is still a floor built for soundness, not a measure of the engine.** The
lane emits only "not valid" observations; a violation-free `Result` DECLINES
rather than passing, because `Assess` evaluates none of `e-validity`'s other
conjuncts. The milestone's remaining slices are what turn declines into
decisions. Exactly one case is decided and decided WRONG (**#771**); the decline
census that separated harvest candidates from indeterminates predates #766, #715
and #740 — three landings that moved 503 cases between them — and is not
re-derived here.

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
