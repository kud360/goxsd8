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

## Status — 2026-08-15 (post-land, #775)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 535 | 25826 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12832 | 2566 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**`instance` moved 532 → 535 (+3)** with #775 (`cbc0253`), which charges
`cvc-complex-type` clause 1.2's String Valid half — a simple `{content type}`
element's ·initial value·. Three expected-invalid `CType` cases, each a
lexically invalid ·initial value·, no expectations line edited downward. #759
closed in the same commit, landing `docs/STYLE.md` **E4** and converting twelve
`validate` messages. **Cite 532 → 535 and never 193 → 196**: `eaa9dc6`'s own
trailer carries a correct +3 pinned to a two-landings-stale baseline, and a
pushed ref cannot be amended (#796).

**This is a post-land pass, not a full backlog run.** The lane table, the queue
counts, the branch namespace and the whole `blocked` set were re-read from their
sources today. The working band **was re-derived** — it had to be, since its
row 1 landed and its row 2 broke — but no issue body was audited that this
landing did not touch.

Milestones — only M5 moved. The rows are the previous stamp's GitHub reads plus
that named delta, because the MCP issue tools expose no milestone filter and
`gh` is 403 for both GraphQL and REST (#527); every other figure in this section
was re-read:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 83 | 41 | **active** |
| M5 — Instance validation (XML) | 9 | 13 | **active** — #775 and #759 closed, #790 filed |
| M6–M12 | 0 | 0 | not filed |

Queue: **194 open issues — 172 `ready`, 22 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `blocked`), against **285 closed**. Five of the 194 are this pass's own
filings. Every open issue carries a queue label — verified across both pages, not
assumed. Read the milestone table as feature progress, not as the queue: 140 of
the 194 carry no milestone at all, and the process, doc and comment-accuracy
issues that post-land passes file are deliberately outside them.

### What this pass did

**The unblock sweep found nothing to unblock, and that is a verified result.**
All 21 `blocked` bodies as of the landing were read and their `## Depends on`
lines checked against live issue state: **no open issue names #775 or #759 as a
dependency.** #718 moved the other way on the same day — `ready` → `blocked`
behind the newly filed #790 — after its own grounding falsified its premise, and
that relabel was the grounding session's, not this pass's.

**Every follow-up #775 raised is now filed or dismissed on the record.** Five
issues and one dismissal, none of them a hand-off:

- **#793** — the arbiter's non-blocking `[P3a]` finding. `initialValue`'s `#774`
  `GAP(validate)` asserts a fail-open direction and delegates its consumer
  enumeration to `cvcattribute.go`, which names no identifiers for it. The claim
  was verified TRUE end to end, so this is where the identifiers are written,
  not what they would say.
- **#794** — `contentCheck.end`'s `ContentType` type switch has no `default`,
  where `xsd/complexextension.go` panics on the same sum and `validate`'s own
  `attributedTo` falls back gracefully. The package has ruled neither. Latent on
  a fourth variant, not a live bug.
- **#795** — `TestSimpleContentDeclinesAnUngovernedType` and the first arm of
  `TestSimpleContentDeclinesAnEmptyElement` assert SILENCE, which a decided-VALID
  outcome also produces. The `"declined"` log record already exists and the
  attribute side already asserts on it; what is missing is the content-side
  reader.
- **#796** — a `Ratchet:` trailer can carry a correct delta pinned to stale
  endpoints, and nothing reads a trailer until after the ref is unamendable.
  `blocked` on the next `/retro`, on #354's own precedent: first instance of
  this mechanism, filed rather than fixed.
- **#797** — the race-losing session's 154-line LOG entry is stranded on
  `chore/log-20260815-develop` and is not on `main`. A code-less develop
  iteration has no PR for its entry to ride. #112 is the closed precedent and
  covers retro branches only.
- **DISMISSED, in a comment on #759** — the arbiter's Note 2, that E4's bold
  lead-in is a fragment where the other entries are complete headline sentences.
  Checked against the file: E4's bold span is a complete clause, `D1` and `D3`
  are not sentences at all, and `L1` already runs its headline into the body on a
  comma. What survives is one entry in 23 ending its bold span with an em dash
  instead of a period, which is below this queue's filing threshold.

**#722's second incident was acted on rather than re-recorded.** The grounding
session had already written the full second-incident report on that thread at
05:35Z; this pass added the disposition it asked for — band promotion to row 1 —
and the live fourth sighting (`wip/issue-718`, below). Whether the recurrence
also warrants a process ruling beyond the tooling fix is the next `/retro`'s.

**Two LOG items needed no filing.** Friction 3 (`testdata/xsdtests` uninitialized
in the arbiter's container) is already documented as an environment requirement in
`docs/ROUTINES.md` and is #659/#527's territory. Surprise 3 (whitespace-only
simple content is DECIDED, not declined) is a verification result absorbed into
the landing, and the test that pins it is in the commit.

**#584's structural defect is unchanged**: it carries `blocked` with no
`## Depends on` section at all, so every future unblock sweep reads past it. Its
real gate is #414, recorded only on its thread; #779 owns the class. Not
re-litigated here.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`origin/wip/issue-775` survived its own merge** and should not have. PR #791
squash-merged as `cbc0253` and `git diff origin/wip/issue-775 origin/main` is
**empty** — the content is on `main` in full, the issue is closed, and the ref is
a stale duplicate. Deletable.

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-718` | **LIVE-shaped, actually finished** | **Zero commits ahead** of `e3ed308`, a claim-only lease. `wipsurvey` says `EXPIRED` on `e3ed308`'s committer date — the #722 defect, fourth sighting. The verdict is right by accident: the holder grounded #718, had its premise falsified, filed #790 and relabelled #718 `blocked`. Deletable, but **not on `wipsurvey`'s say-so**. |
| `wip/issue-775` | merged | Content identical to `main`. Deletable. |
| `chore/log-20260815-develop` | **strands real history** | 1 ahead, 1 behind. Its one commit `c9f1616` is a 154-line `docs/LOG` entry that is on no other ref. **#797 owns it.** |
| `wip/issue-256` | RETIRED | Issue closed `needs-replan`, superseded by #470. |
| `wip/issue-271` | RETIRED | Issue closed `needs-replan`, superseded by #478/#479/#480. |
| `wip/issue-287` | RETIRED | Issue closed, answered in the opposite direction by PR #511. Not a park — it carries no `needs-replan`. |
| `claude/eloquent-cerf-8f1mrv` | merged | 0 ahead of `main`. Deletable. |
| `claude/eloquent-cerf-patxs3` | superseded | Ahead by SHA, none by content — tip `2196c3f`, unmoved since 2026-08-07. Deletable. |
| `claude/eloquent-cerf-7ckw7b` | **needs a human look** | 7 ahead, carried forward from the previous stamp and not re-measured today. The seven commits name #662, #359, #649, #426, #358, #643 and #636, whose work is believed to be on `main` as squashes, but no commit on `main` carries those trailers and no pass has verified the content diff. |

No `parked/*` branches. **No `wip/*` branch holds a live claim.**

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
list. Take from the top. **Re-derived this pass**, not pruned: row 1 landed and
row 2 broke apart when #718 went `blocked`.

| # | Issue | Why here |
|---:|---|---|
| 1 | #722 | **the lease-clock trap, and the only queue item whose absence actively destroys throughput** — it has now done so twice in twelve days, the second time costing a full discarded mason round. Four lease-only claims have been misclassified (#503, #715, #775, and `wip/issue-718` today); the written workaround failed on the branch it was written for. One tooling change, no lane movement, and it protects every session after it |
| 2 | #790 | **M5's head** — thread the ·context-determined declaration· into descendant assessment (§3.3.4.6 clause 3.1). `ready`, dependency-free, and the enabler #718 now waits on. It re-opens every decline `rootComplexType` makes today, which is what turns the lane's declines into decisions; its grounding is already written across #715's warden pre-flight and #718's oracle answer, and neither needs re-running |
| 3 | #716 + #719 | the M5 slices #715 unblocked that are still startable, **unordered against each other**. #718 has left this row — it is `blocked` behind #790. **#716 interacts with #790 and neither blocks the other**: they meet where `xsi:type` changes a descendant's ·governing type definition·, and whichever lands second owns making them meet (#790's Notes say how, in both orders) |
| 4 | #659 + #527 | the environment tax, one sentence of prose each. #527 has now been paid by every pass for thirteen consecutive sessions, and this one paid it twice — GraphQL *and* REST are both 403 |
| 5 | #733 | a top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and #442 moved `schema` **+82** |
| 6 | #748 + #747 | the two persona findings, both re-verified and corrected on their threads on 2026-08-14. #748 is four facts, three of them one-line edits; #747 is two, and its proposed re-scope was refused |
| 7 | #625 → #669 → #492 | the rest of README's Library block, in that order — #625's fix names the file #669 must add to the example list, and #492's belongs in the sentence at `README.md:115-116` rather than a new paragraph |
| 8 | #514 + #687 + #672 | the open CLI-contract decisions: typo-vs-unbuilt, scoped help, and `-version`. Take them **before** #472 — each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 9 | #472 | implement `goxsd8 parse`, the first non-stub subcommand — and the issue that owns the exit-2 split |

**Rows 4 through 9 are the previous band's rows 3 through 8, unchanged in
substance**, with their justifications intact: nothing landed that touches any of
them.

**Deliberately unbanded, and why.** **#744** — the chained-redefine successor — is
the highest-value M4 gap in the queue and is held out only because its
`## Surface` requires a warden pre-flight on an `xsd` entry point that does not
exist; **#773** and **#721** are held out on the same condition. **#774** is
three fail-open declines to close or rule permanent — it can move a lane, but
which way is unknown until the first is decided, and #795 is what would pin the
element-side one either way. **#782**, **#783**, **#785**, **#786**, **#787** and
**#788** are earlier landings' post-land follow-ups; **#786** is the one of them
that can move a lane, and a later pass should weigh it against row 5.
**#793**, **#794**, **#795**, **#796** and **#797** are this pass's own filings
and are not banded on first appearance.

**`parser/redefine.go` has been rewritten five times in a week, so read it rather
than an issue body that describes it.** #686, #699, #506, #503 and #504 each
moved it; #744, #706 and #726 all open it next.

### Next planning action

**Sweep the carried follow-up ledger (#489)** — file or dismiss each unfiled
advisory, one comment per item — before the next carve. It has now gone ten
consecutive passes undone and grows with every landing; #330 is the standing
proof that a hand-off tracks nothing. This pass filed five issues and dismissed
one against a single landing, which is the shape the sweep is for and is not a
substitute for running it.

The band's ordering debt from the previous stamp is **discharged**: rows 1–3 are
now ordered against each other rather than listed together. What a full
`/backlog` still owes is the other end — rows 4 through 9 have been carried
forward on their original justifications across three passes and have not been
re-derived against anything that has landed since.

Everything else this queue needs is a develop iteration. **Start with #722.**

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
Eight have landed: the infoset
seam and engine skeleton (**#710**), the `xmlsrc` adapter (**#711**) and root
dispatch (**#712**), none of which moved the lane and correctly so — the first
two decide no `cvc-` rule and nothing yet ran the cases; **the lane driver
(#713), which took `instance` off zero — 0 → 18 pass** (the #175 analogue, placed
fourth so every slice after it reports a real number); `cvc-complex-type`
clauses 2–3 (**#714**, 19 → 29); the `value.Backend` seam with the four
datatype-valid attribute charges (**#766**, 29 → 193); and the greedy
non-backtracking content matcher (**#715**, 193 → 520), the largest single move
the milestone has recorded; and `cvc-complex-type` clause 1.2's ·initial value·
against String Valid (**#775**, 532 → 535), which also closed #759 and landed
`docs/STYLE.md` **E4**. `instance` stands at **535** — #740, an M4 landing, took
it 520 → 532 on a merged tree neither parent could measure.

**The carve is no longer a chain, and reading it as one loses work.** The
original ordering assumed a slice becomes `ready` as the one before it lands.
That held until #714, which had **two** successors directly behind it, and #766
has **three**; #715 had four, of which #716, #718 and #719 became `ready` when it
landed while #717 waits on #248 as well. **A slice can also travel backwards**:
#718 went `ready` → `blocked` when its own grounding falsified its premise and
filed #790 as the enabler it needs. Relabel from the `## Depends on` lines,
never from the milestone or from the issue numbers' order — and sweep for issues
carrying **no queue label at all**, which is how #773 and #774 sat outside both
queues for a day. The CLI's own `validate` subcommand is #720, `blocked` behind
#472 alone now that #715 has landed.

**535 is still a floor built for soundness, not a measure of the engine.** The
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
