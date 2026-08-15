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

## Status — 2026-08-15 (post-land, #790)

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

**`instance` moved 535 → 1017 (+482)** with #790 (`5387eec`, PR #799) — the
largest single movement any lane has recorded, and the first `instance` movement
not counted in single or double digits. `validate/assess.go`'s `walk.child`
stopped discarding the `xsd.Attribution` its own content check had just computed
and threads it into the recursive descent, so every element below the validation
root is assessed against the ·governing element declaration· its parent's content
model ·attributed· it to (§3.3.4.6 clause 3.1) instead of against nil. All three
wildcard `{process contents}` arms landed with it, skip as a hard stop on the
whole subtree rather than a permissive pass. 482 expectation lines flipped `fail`
→ `pass`, none downward, line count unchanged at 26361; the flipped set is
byte-identical to the read-only run's improved-but-unbanked list.

**The +482 is one mechanism, not 482 fixes.** Every flip is a document whose
defect sits below the root and which the engine could not previously see at all.
No new rule was implemented — the same six rule IDs are charged, which is why
`decidedNotValid` needed no edit.

**This is a post-land pass, not a full backlog run.** The lane table, the queue
counts, the branch namespace and the whole `blocked` set were re-read from their
sources today. The working band **was re-derived** — its rows 1 and 2 both
resolved. No issue body was audited that this landing did not touch.

Milestones — only M5 moved. The rows are the previous stamp's GitHub reads plus
that named delta, because the MCP issue tools expose no milestone filter and
`gh` is 403 for both GraphQL and REST (#527); every other figure in this section
was re-read:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 83 | 41 | **active** |
| M5 — Instance validation (XML) | 10 | 12 | **active** — #790 closed, #718 unblocked |
| M6–M12 | 0 | 0 | not filed |

Queue: **195 open issues — 174 `ready`, 21 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `blocked`), against **286 closed**. One of the 195 is this pass's own
filing. Every open issue carries a queue label — 174 + 21 = 195 exactly, so the
arithmetic proves it rather than a sweep asserting it. Read the milestone table
as feature progress, not as the queue: 141 of the 195 carry no milestone at all,
and the process, doc and comment-accuracy issues that post-land passes file are
deliberately outside them.

### What this pass did

**The unblock sweep found exactly one, and it is the one the landing was for.**
All 21 `blocked` bodies were read and their `## Depends on` lines checked against
live issue state. **#718** — `cvc-identity-constraint` and `cvc-id` — named #790
and nothing else, and moved `blocked` → `ready`. No other open issue names #790
as a dependency.

**#718's dependency was checked against the code, not against the closure.** Its
`blocked` state rested on one factual claim, from the grounding that falsified
it: *"every selector target and field node is an untyped descendant today"*. That
claim is now false. Every element in the root's subtree is visited at every
depth; attribute field nodes are fully typed, since `walk.attributes` runs at
each descendant and `matchedAttribute` already resolves the declaration's
`xsd.SimpleType`; element field nodes under a simple `{content type}` reach their
`{simple type definition}` through `contentCheck.initialValue`. **One piece of
plumbing is #718's own and is not a new dependency**: `walk.element` carries a
`*xsd.ComplexType`, and `governingComplexType` returns nil for a declaration
whose `{type definition}` resolves to a *simple* type, so a field node declared
`type="xs:string"` arrives untyped and must have its type read off the
declaration `childGoverning` already holds. The two residual untyped shapes —
`xsi:type` (#716) and ·skipped· subtrees — are both already licensed by #718's
own Acceptance, and the skip case is not a gap at all: §3.11.4 `key-tns` omits
·skipped· nodes from the ·target node set· outright. Reasoning on the thread.

**Every follow-up #790 raised is filed or folded on the record.** One issue, two
foldings, none of them a hand-off:

- **#800** — the producer never maps `<xs:alternative>` to `{type table}`.
  Both `NewElementDeclaration` call sites pass `nil`, `xsd.NewTypeTable` has no
  non-test caller, and **four readers are dead code on any parsed schema**:
  `validate`'s `governingComplexType` decline, `xsd`'s `checkTypeTablesAgree`
  (`cos-element-consistent`) and `declaredTypeRestricts`, and `resolveTypeTable`.
  The arbiter required this filing at Land. Its `## Surface` is the live design
  question — §3.12.2's second arm is an `<alternative>` with an inline type,
  which `xsd.TypeAlternative`'s QName-only `{type definition}` cannot represent.
- **FOLDED into #794** — `childGoverning`'s `default:` arm returns `(nil, true)`
  where its `xsd.Attribution` sibling `attributedTo` falls back and `xsd` panics.
  #794 already exists to rule on exactly this for `validate`, and its body's
  *"those two are `validate`'s only sealed-sum type switches"* is now false —
  there are three. Answering it twice risks answering it two ways. The comment
  also records that `default: panic` alone would be wrong here: `childGoverning`
  documents nil as a live, reachable attribution, so an explicit `case nil:` is
  mandatory, which is #772's hazard demonstrated on a second site.
- **FOLDED into #800** — `conformance/instance.go:100`'s "charges at depth stay
  unconditional" paragraph is unsound precisely where the producer dropped an
  `<alternative>`, and three cases prove it. It is an Acceptance line on #800
  rather than an issue, because the edit is only decidable once that landing
  knows how much of the gap survives.

**Three suite-VALID cases are decided and WRONG, and they were already banked
`fail`.** `Assert/assert_019_2`, `CTA/cta0008.v01` and
`CTA/typeAlternatives_001_2` are assessed against the declared type where an
`<alternative>` should have ·conditionally selected· another. Not a #790
regression — pre-existing debt the descent made visible. **The count of
decided-and-disagreeing `instance` cases is therefore 4, not 1**: #771 plus these
three. #800 removes the three by turning them into honest declines.

**#800 cannot move `instance` and that is the correct outcome.** Mapping the
table makes `governingComplexType` return nil, so nothing is charged, so
`execInstanceCase` declines — and `conformance/instance.go:252` scores a decline
as `Fail()` exactly as it scores a wrong decision. The gain is soundness, which
this lane has no way to register. `schema` is where movement can come from, via
the readers the mapping makes live, and a `schema` **regression** is the real
risk there.

**Two arbiter observations needed no filing.** The `testdata/xsdtests` submodule
missing from the arbiter's container is #659/#527's territory and already an
environment requirement in `docs/ROUTINES.md`. The check that no banked pass
*rests* on the unsound `<alternative>` path was run and came back clean —
`CTA/cta0008.n01` rejects soundly on its own terms — which is a verification
result, not debt.

**#584's structural defect is unchanged**: it carries `blocked` with no
`## Depends on` section at all, so every unblock sweep reads past it. Its real
gate is #414, recorded only on its thread; #779 owns the class. Not re-litigated
here, but note it survived a sweep that read all 21 `blocked` bodies — the second
consecutive pass to confirm the defect by hand.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**Two refs the previous stamp reported as deletable are gone, and one carried
real history home.** `origin/wip/issue-775` and `origin/wip/issue-790` were both
deleted at merge, correctly. **`chore/log-20260815-develop` is also gone, and its
content is on `main`** as `8e8b45b` at `docs/LOG/2026-08.md:30454` — verified by
content, not by commit subject. That retires #797's *incident* while leaving
#797's *mechanism* open; recorded as a comment on that thread, since its title
now asserts something false.

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-722` | **LIVE** | 1 ahead, 1 behind, tip today 13:33Z. A real session is working #722 right now. **Do not take #722**, and do not read `wipsurvey` against it — that is the very defect #722 exists to fix. |
| `claude/eloquent-cerf-15r30e` | merged | PR #799's branch, 0 ahead / 0 behind `main`. Deletable. |
| `wip/issue-718` | **dead lease** | 0 ahead, 4 behind. Claim-only, holds no work, and its issue went `ready` today — the next attempt branches fresh from `origin/main` and must not resume this ref. Deletable. |
| `wip/issue-256` | RETIRED | Issue closed `needs-replan`, superseded by #470. |
| `wip/issue-271` | RETIRED | Issue closed `needs-replan`, superseded by #478/#479/#480. |
| `wip/issue-287` | RETIRED | Issue closed, answered in the opposite direction by PR #511. Not a park — it carries no `needs-replan`. |
| `claude/eloquent-cerf-8f1mrv` | merged | 0 ahead. Deletable. |
| `claude/eloquent-cerf-patxs3` | superseded | Tip `2196c3f`, unmoved since 2026-08-07. Deletable. |
| `claude/eloquent-cerf-7ckw7b` | **needs a human look** | 7 ahead, 51 behind, unmoved since 2026-08-10 and **carried forward across four stamps without re-measurement**. Its seven commits name #662, #359, #649, #426, #358, #643 and #636, believed on `main` as squashes, but no `main` commit carries those trailers and no pass has verified the content diff. |

No `parked/*` branches. **One `wip/*` branch holds a live claim** (`wip/issue-722`).

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
list. Take from the top. **Re-derived this pass**: rows 1 and 2 of the previous
band both resolved — #790 landed, and #722 is now claimed rather than waiting.

| # | Issue | Why here |
|---:|---|---|
| 1 | #718 | **M5's head, unblocked today** — `cvc-identity-constraint` and `cvc-id`. It is the largest remaining `instance` mover in the queue and its enabler is now on `main`; its grounding (comment 5300735646) is durable, was scoped to the rule text rather than to the engine, and does not need re-running. Its XPath subset is §3.11.6.2/§3.11.6.3's restricted path grammar and **does not wait on M6** — a fail-open to the M6 evaluator here would be wrong, not conservative |
| 2 | #716 | `xsi:type` and `xsi:nil`. **#790 raised its value rather than leaving it flat**: the `xsi:type` decline used to cost one element and now costs a subtree, since a descendant carrying `xsi:type` is assessed against no type and everything below it inherits that. It is also the residual #718 will otherwise GAP-mark. Dependency-free; #790's Notes already say how the two meet |
| 3 | #800 | the `<xs:alternative>` producer gap. Below #716 because it **cannot move `instance`** — it converts three false rejects into declines, which the lane scores identically. Above the tail because it is the only queue item that retires a *decided-and-wrong* case, may move `schema`, and makes four dead readers live. Needs a warden pre-flight on `xsd.TypeAlternative` first |
| 4 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. Unordered against #800; take whichever the session is shaped for |
| 5 | #659 + #527 | the environment tax, one sentence of prose each. #527 has now been paid by every pass for fourteen consecutive sessions, and this one paid it twice again — GraphQL *and* REST both 403 |
| 6 | #733 | a top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and #442 moved `schema` **+82** |
| 7 | #748 + #747 | the two persona findings, both re-verified and corrected on their threads on 2026-08-14. #748 is four facts, three of them one-line edits; #747 is two, and its proposed re-scope was refused |
| 8 | #625 → #669 → #492 | the rest of README's Library block, in that order — #625's fix names the file #669 must add to the example list, and #492's belongs in the sentence at `README.md:115-116` rather than a new paragraph |
| 9 | #514 + #687 + #672 | the open CLI-contract decisions: typo-vs-unbuilt, scoped help, and `-version`. Take them **before** #472 — each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 10 | #472 | implement `goxsd8 parse`, the first non-stub subcommand — and the issue that owns the exit-2 split |

**#722 has left the band because it is being worked**, not because it stopped
mattering. If `wip/issue-722` goes stale without landing, it returns to row 1.

**Rows 5 through 10 are the previous band's rows 4 through 9, unchanged in
substance**, with their justifications intact: nothing landed that touches any of
them. They have now been carried on their original justifications across **four**
passes.

**Deliberately unbanded, and why.** **#744** — the chained-redefine successor — is
the highest-value M4 gap in the queue and is held out only because its
`## Surface` requires a warden pre-flight on an `xsd` entry point that does not
exist; **#773** and **#721** are held out on the same condition. **#774** is
three fail-open declines to close or rule permanent — it can move a lane, but
which way is unknown until the first is decided, and #795 is what would pin the
element-side one either way. **#771** is the last decided-and-disagreeing
`instance` case once #800 lands, and a later pass should weigh it against row 3.
**#782**, **#783**, **#785**, **#786**, **#787** and **#788** are earlier
landings' post-land follow-ups; **#786** is the one of them that can move a lane.
**#793**, **#794**, **#795**, **#796** and **#797** are the previous pass's
filings; **#794** grew a third site today and is now a slightly larger job than
when it was filed.

**`parser/redefine.go` has been rewritten five times in a week, so read it rather
than an issue body that describes it.** #686, #699, #506, #503 and #504 each
moved it; #744, #706 and #726 all open it next.

### Next planning action

**Sweep the carried follow-up ledger (#489)** — file or dismiss each unfiled
advisory, one comment per item — before the next carve. It has now gone eleven
consecutive passes undone and grows with every landing; #330 is the standing
proof that a hand-off tracks nothing. This pass filed one issue and folded two
findings into existing ones against a single landing, which is the shape the
sweep is for and is not a substitute for running it.

**The band's tail is the debt a full `/backlog` owes.** Rows 5 through 10 have
now been carried forward on their original justifications across four passes and
have not been re-derived against anything that has landed since — and `instance`
has nearly doubled in that window, which is exactly the kind of movement that
should reorder a queue and has not been allowed to.

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
