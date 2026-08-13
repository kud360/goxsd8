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

## Status — 2026-08-13

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 0 | 26361 | 26361 |
| `json` | — | — | 0 |
| `schema` | 11635 | 3763 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete** — its last open issue closed
this week; `schema` is M4 and active; `instance` is M5, carved, and **still at
zero**; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

`schema` moved **9745 → 11635, +1890**, across eight landings since the last
pass, each figure copied from its commit's `Ratchet:` line rather than
recomputed: #503 (+6), #442 (+82), **#447 (+1572)**, #504 (+6), **#738 (+224)**,
with #710 and #626 unchanged and not meant to move. `datatypes` moved **1156 →
1161, +5**, all of it #590. #447 alone is 83% of the schema total, and the figure
is not the cohort its issue named: `simpleTypeDecidable` required a `restriction`
child, so every schema using `xs:list` anywhere was declined wholesale regardless
of what it tested. The arbiter attributed all 1572 case by case.

**`instance` is the lane to read.** M5's engine seam landed (#710) and the lane
did not move, correctly — a seam has no source adapter and no driver. 26361
cases, none of them ever run, is the largest single block of unclaimed
conformance in the project, and the band below is ordered accordingly.

Milestones, from GitHub:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 81 | 43 | **active** |
| M5 — Instance validation (XML) | 1 | 13 | **active** — first slice landed |
| M6–M12 | 0 | 0 | not filed |

Queue: 177 open issues — 151 `ready`, 26 `blocked`, 0 `needs-replan`, 2 `epic`
(both `blocked`). **56 of the 177 carry a milestone; 121 carry none** — the
milestones track feature scope, and the process, doc and comment-accuracy issues
that post-land passes file are deliberately outside them. Read the milestone
table as feature progress, not as the queue.

**This pass filed two issues, closed none, and commented on six.** Both filings
are persona findings with no existing owner: **#747** (the CLI's own `-help`
never says the subcommands are stubs, and README:75 claims `go doc` renders the
same text) and **#748** (README's M5 block is wrong in both directions). The six
comments are confirmations on **#688**, **#625**, **#669**, **#492**, **#514**
and **#687**.

### What this pass did, and what it deliberately did not

The standing "carve M5 and nothing else" instruction that governed the last pass
is **spent**: M5 is carved, its first slice has landed, and the band below is a
full re-derivation of the head rather than an edit of it.

- **Both persona reports were folded** (step 5), produced today by libuser and
  cliuser against README and `go doc` only. Eight findings: two had no owner and
  were filed (#747, #748); six duplicated open threads and were confirmed there
  rather than re-filed. Two of the confirmations carry more than agreement —
  **#688** is widened beyond `area/xsd` (`parser.AssembledDocument` and
  `*validate.Validator` degrade the same way, and worse), and **#669**'s
  "does-not-compile" premise is contradicted by the persona running the snippet
  successfully, so that half is flagged for re-verification before it is worked.
- **`gapaudit`'s group 1 is empty.** No `GAP(` marker in the tree lacks an open
  tracking issue, over 37 markers in 4 areas. Group 2's six entries are the
  permanent kind — five conformance-lane gaps that never carry a marker (#404,
  #591, #592, #593, plus #398, which says in its own title that it has no
  marker) and #719, whose marker lands with its M6 seam. No `kind/gap` issue was
  owed.
- **`docs/PLAN.md`'s #503/#504 coupling record was wrong and is deleted, not
  edited.** The previous section stated that #504's landing would carry a
  decidability widening in `conformance/schema.go`. It did not: `groupDecidable`
  and `attributeGroupDecidable` carry no redefine-specific decline, so there was
  nothing to widen. What keeps `MS-Schema2006-07-15/schL10` and `/schM5` banked
  `pass` is `parser/redefine.go`'s `chainedOriginal` guard, and **#744** owns
  collapsing it. #504's post-land pass had already corrected #726's body the same
  way; this section was the last carrier.
- **#493 was left alone deliberately.** Its acceptance owns correcting #256's and
  #271's `state_reason` and explicitly asks the implementing session to decide
  between a reopen-and-reclose and a comment. Doing either here would have
  spent that decision without recording it. #287 is **not** a third instance —
  it carries no `needs-replan` label.

**What was not done: a body-by-body audit of all 151 `ready` issues.** No stale
issue was closed and no duplicate merged, because none was found in the bounded
sweep this pass ran — the seven landings' follow-ups, the M5 family, the
README/CLI cluster the personas touched, and the three retired branches. Four
clusters were checked for merge candidates and each is genuinely distinct rather
than duplicated: the STYLE-citation family (#382, #540, #543, #548) differ in
defect shape, not subject; the README family (#492, #625, #669, #748) are four
separate paragraphs of one section. **#489**'s carried-follow-up ledger sweep is
again undone, and is named as the next planning action rather than left implicit.

### Branch namespace, `origin` — report-only; a session never deletes a ref

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-256` | EXPIRED | Retired in place; issue closed `needs-replan`, superseded by #470. |
| `wip/issue-271` | EXPIRED | Retired in place; issue closed `needs-replan`, superseded by #478/#479/#480. |
| `wip/issue-287` | EXPIRED | Retired in place; issue closed, answered in the opposite direction by PR #511. Not a park — it carries no `needs-replan`. |
| `claude/eloquent-cerf-7ckw7b` | merged | 0 commits ahead of `main`. Deletable. |
| `claude/eloquent-cerf-8f1mrv` | merged | 0 commits ahead of `main`. Deletable. |
| `claude/eloquent-cerf-patxs3` | superseded | 53 commits ahead by SHA, none by content — its tip twelve are landed issues (#307, #336, #306, #304, #303, #299, #305, #449, #296, #295) that reached `main` as squash commits. Deletable. |

No `parked/*` branches. All six refs are listed for human triage and the three
`wip/*` have been for several passes.

**None of the three `wip/*` refs shares an ancestor with today's `main`** —
`git merge-base origin/main origin/wip/issue-N` returns empty for all three, so
`git log main..branch` counts are meaningless for them and "verify its content is
in `main`" has to be done by reading files, not by revision arithmetic. That is
why #493's finding (`wip/issue-271` carries zero commits not in `main`) and a
naive 28-commit count disagree: the count is right and the inference from it is
not.

**Read a zero-commit `wip/issue-N` against its issue thread before believing an
EXPIRED verdict.** This is **#722**: `wipsurvey` dates a lease-only branch from
the *previous* landing's committer date, so a claim made minutes ago reports
EXPIRED, and the prescribed response to EXPIRED is to label the issue
`needs-replan`. A pass following it blindly retires a live session's issue in the
hour its design is approved. The caution stands until #722 lands.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 151
issues. Take from the top; the ordering prefers slices that move a lane over
horizontal completeness.

| # | Issue | Why here |
|---:|---|---|
| 1 | #711 | `validate/xmlsrc`: adapt `parser/xmltree` onto the infoset. **The `instance` lane's first mover** — 0 of 26361, and #713's driver is blocked on this and row 2 |
| 2 | #712 | `cvc-assess-elt` and `cvc-elt` clauses 1–2 — root dispatch and the first real violation. The engine half of the same pair; a driver with no source or no dispatch moves nothing |
| 3 | #740 | the `enumeration` facet is declined by the gate and, past it, refused with a **fabricated** `src-simple-type` verdict on a legal schema (STYLE E2). `pdecimal006` is the named cost, and it is the same two-refusal shape #447 already worked through |
| 4 | #733 | a top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and #442 moved `schema` **+82** |
| 5 | #747 + #748 | the two findings the 2026-08-13 pass filed. #747's second fact is a divergence **#626 introduced**; #748's snippet will not compile against the signature `xmlsrc/doc.go` already commits |
| 6 | #625 → #669 → #492 | the rest of README's Library block, in that order — #625's fix names the file #669 must add to the example list, and #492's belongs in the sentence at `README.md:115-116` rather than a new paragraph |
| 7 | #514 + #687 + #672 | the open CLI-contract decisions: typo-vs-unbuilt, scoped help (now with a third case — `-h` after an unknown subcommand exits 0 and flags nothing), and `-version` |
| 8 | #472 | implement `goxsd8 parse`, the first non-stub subcommand |

**#738 held row 1 and has landed** (PR #750, `c754427`, `schema` 11411 → 11635,
**+224**). Its row is dropped rather than replaced, and unlike the last three
landings **its text was right about its own issue** — the sibling framing, the
seven placeholder sites and the #447 analogy all held. What the row got wrong was
the *size*: it invited a +1572-shaped expectation and the measurement is +224,
because `union` is rarer in the suite than `list` and many union-bearing closures
still decline on the `enumeration` fault row 3 owns. Mason predicted, then
measured, then attributed the shortfall before the arbiter ran the lane, which is
the behaviour the row was written to provoke.

**The landing unblocked nothing.** All 26 `blocked` bodies were surveyed and none
names #738 in `## Depends on` or anywhere else; the two whose subject is close
enough to be worth reading in full — #555 (`length-minLength-maxLength`, whose
`## Depends on` is a trigger and says in as many words not to re-scan it) and
#593 (blocked on #591) — are unaffected. No label was flipped.

**One follow-up filed, one issue amended, one confirmed, none banded** — ranking
them is a re-derivation this pass does not do. **#751**: `cos-st-restricts`
clause 3.2.1.1 became reachable from a schema document with this landing and
nothing drives it that way — the union-member `final="union"` rejection is pinned
only at the component level and an arbiter hand-run is the only evidence it fires
end to end. **#740**'s `## Current state` section is amended by comment (three
line references drifted, and its worked precedent
`TestProduceSimpleContentRestrictionDeclined`, since the `union` decline it named
was deleted by this landing); its scope and startability are unchanged. **#624**
gets the `SimpleTypeOrRef` arm × slot line #738's Notes asked for — both arms are
now producer-written, which makes it usable prior art for the pass-through that
issue wants, and it stays `ready` with no code overlap. `gapaudit`'s marker side
is clean: the one `GAP(parser)` marker was **deleted** rather than repointed, the
sweep fell 46 → 45, and no marker names the now-closed #738.

**Only the `schema` lane row and the per-landing ratchet sentence above were
re-read; every other count in this section still dates from the 2026-08-13
`/backlog` and the next one re-derives them.** A post-land pass is not `/backlog`
and does not rewrite the section.

**Rows 5–7 sit ahead of row 8 on cost, not importance.** Each is a sentence or a
dispatch branch while the CLI surface is still empty; taken after #472 every one
of them is a change to shipped behaviour. #472's own Acceptance carries the
`-version` decision, so if it is taken first it must discharge #672 rather than
leave it contradicting the landing. **#720** (`goxsd8 validate`) is `blocked`
behind #472 and #715 and is not a band row.

**Rows 1 and 2 are the pass's judgment call.** Horizontal completeness would put
four more M4 parser gaps here; the doctrine says a slice that moves a lane wins,
and `instance` is the only lane that has never moved. #713 is the payoff and it
is `blocked` on exactly these two.

**Deliberately unbanded, and why.** **#744** — the chained-redefine successor —
is the highest-value M4 gap in the queue and is held out only because its
`## Surface` requires a warden pre-flight on an `xsd` entry point that does not
exist, in three candidate shapes; it is a band row the moment that call is made.
**#743**, **#742**, **#741**, **#731**, **#732**, **#726**, **#725**, **#702**,
**#703**, **#706** are this month's post-land follow-ups: real, filed, and none
of them moves a lane. **#721** wants a warden call on two shapes before it is
workable. **#722** is the one to promote if a later pass wants a cheap
high-value row — it is still the only unbanded issue whose absence can cause a
future pass to actively damage the queue. The seven `blocked` M5 slices cannot
be banded at all.

**`parser/redefine.go` has been rewritten five times in six days, so read it
rather than an issue body that describes it.** #686 put its duplicate-key charge
in `recordOriginal`; #699 added `rejectProhibitedAttrs` to `newRedefineSet`;
#506 inserted ~45 lines and moved every marker line number the #585 and #505
threads cite; #503 added the clause-7.1-vs-7.2 branch; #504 charged clause 6.2.2
and deleted the `checkRedefinedGroup` marker along with `parser/doc.go`'s
composition-gaps bullet. #744, #706 and #726 all open that file next.

### Next planning action

**Sweep the carried follow-up ledger (#489)** — file or dismiss each unfiled
advisory, one comment per item — before the next carve. It has now gone five
consecutive passes undone, it grows with every landing, and #330 is the standing
proof that a "handed to the post-land pass" hand-off tracks nothing. Everything
else this queue needs is a develop iteration, not a planning one.

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

Carved 2026-08-12 into ten slices, #710–#719, dependency-ordered so that a
slice becomes `ready` as the one before it lands: the infoset seam and
engine skeleton (**#710, landed**), the `xmlsrc` adapter (#711) and root
dispatch (#712) — both `ready` — then **the lane driver (#713), which
takes `instance` off zero before the bulk of the `cvc-` work
lands** — the #175 analogue, placed fourth so every slice after it reports
a real number. The fan-out is #714–#719. The CLI's own `validate`
subcommand is #720, `blocked` behind #472 and #715.

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
