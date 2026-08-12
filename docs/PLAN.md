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

## Status — 2026-08-12

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1156 | 17 | 1173 |
| `instance` | 0 | 26361 | 26361 |
| `json` | — | — | 0 |
| `schema` | 9745 | 5653 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a
lane scoring zero. `datatypes` is M3 and effectively complete; `schema` is
M4 and active; `instance` is M5 and is **carved as of today**; `xpath`,
`json` and `ber` wait on M6/M7, M8 and M11.

`schema` moved **9707 → 9745, +38**, across six landings since the last
pass. Four of the six moved it — #675 (+6), #684 (+2), #699 (+2) and #469
(+28) — and #469 alone is three quarters of the total, on `cos-all-limited`
becoming a finalize check over the resolved component graph. #506 and #686
banked nothing and were not meant to: both are false-reject and
unit-test-only fixes.

Milestones, from GitHub:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 1 | complete; one follow-up open |
| M4 — Schema parsing | 79 | 40 | **active** |
| M5 — Instance validation (XML) | 0 | 13 | **carved today** — #710–#719, plus #720, #16 and the epic |
| M6–M12 | 0 | 0 | not filed |

Queue: 170 open issues — 142 `ready`, 28 `blocked`, 0 `needs-replan`,
2 `epic`. **54 of the 170 carry a milestone; 116 carry none** — the
milestones track feature scope, and the process, doc and comment-accuracy
issues that post-land passes file are deliberately outside them. Read the
milestone table as feature progress, not as the queue.

**This pass filed thirteen issues and closed none.** Eleven are the M5 carve
(#710–#719 plus #720); one is a libuser finding with no existing owner
(#721); one is a defect in `wipsurvey` that this pass's own branch survey
tripped over (#722).

### What this pass did, and what it deliberately did not

The last three Status sections named "carve M5 (#250)" as the next planning
action and did not do it, and the 2026-08-11 section replaced that intention
with a mechanism: *the next `/backlog` carves M5 and does nothing else,
accepting a stale band for one cycle.* **The carve is done** — epic #250 is
ten dependency-ordered slices, only the first of them `ready`.

Three things were done beside it, each because skipping it would have left a
known thing untracked:

- **The two persona reports were folded** (step 5). They were produced fresh
  today against the current published surface and handed to this pass;
  leaving them undisposed is exactly the unfiled-advisory leak #489 and #330
  exist to stop. One new issue (#721), one new subcommand filing (#720), and
  seven confirmations on existing threads (#669, #409, #626, #672, #687,
  #492, #472). Most findings were **already owned**: #409 and #626 are now
  on their second and third independent sighting, and re-filing either would
  have been the duplicate the queue search exists to prevent.
- **#722 was filed** because step 2's own survey produced it: `wipsurvey`
  reported a live session's branch as EXPIRED, and the prescribed response to
  EXPIRED is to label its issue `needs-replan`. Dismissing that would have
  been filing a known hazard as nothing.
- **#250's body carries three wrong spec citations**, found while carving and
  corrected on its thread (`cvc-assess-elt` is §3.3.4.6, `cvc-simple-type` is
  §3.16.4, `cvc-model-group` is §3.8.4.3), along with a stale absolute lane
  figure (#646's pattern). Corrected by comment, not by body edit, per #515.

**What was not done: step 3's queue-wide reconciliation.** No stale issue was
closed, no duplicate merged, no oversized issue split, and no open body
outside the M5 family was audited for a stale premise. `gapaudit`'s group 1 is
**empty** — no `GAP(` marker in the tree lacks an open tracker — so no
`kind/gap` issue was owed. The working band below is re-ranked at its head
only, not re-derived across 142 `ready` issues. That is the "and nothing else"
the standing instruction bought, and it is spent here rather than silently.

### Branch namespace, `origin` — report-only; a session never deletes a ref

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-256` | EXPIRED | Retired in place; issue closed and superseded by #470. |
| `wip/issue-271` | EXPIRED | Retired in place; issue closed and superseded by #478/#479/#480. |
| `wip/issue-287` | EXPIRED | Retired in place; issue closed, answered in the opposite direction by PR #511. |

No `parked/*` branches. The three EXPIRED refs are listed for human triage and
have been for several passes.

**`wip/issue-503` read EXPIRED and was not** — this is #722, and the branch has
since merged and been auto-deleted, which settles the question the way the
survey did not. At survey time the ref pointed at exactly `origin/main`
(`adf0356`, zero commits ahead), because a develop session's first act is to
push the claim before any work commit, so `wipsurvey` dates the branch from the
*previous landing's* committer date and reported *"tip pushed 4h37m ago, past
the 2h claim TTL"*. Meanwhile #503's thread carried an oracle grounding, a
placement decision and a **passing warden pre-flight**, all that day, the last
of them 80 minutes before the survey; it landed three hours later. Applying the
prescribed EXPIRED response would have labelled a live issue `needs-replan` in
the hour its design was approved. Until #722 lands, **read a zero-commit
`wip/issue-N` against its issue thread before believing the verdict.**

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 142
issues. Take from the top; the ordering prefers slices that move a lane over
horizontal completeness.

| # | Issue | Why here |
|---:|---|---|
| 1 | #447 | `xs:simpleType` with a `union`/`list` body unproduced — `datatypes`, pdecimal019/020 the measured cost |
| 2 | #590 | pdecimal016, the Saxon PDecimal cohort's five-case chain — `datatypes`, #574's sibling widening |
| 3 | #504 | `src-redefine` clause 6.2.2, fail-open. **Startable now — #503 has landed**, and this landing carries the decidability widening. Read the coupling note below |
| 4 | #626 | README's CLI section is false about exit codes today; one sentence, and it precedes #472 |
| 5 | #669 | README's Library snippet does not compile, and the example pointer omits `parser`, `xsd`, and `xsd/example_test.go` |
| 6 | #514 | a typo'd subcommand and an unbuilt one are indistinguishable — fix before #472 makes it misleading |
| 7 | #672 + #687 | the open CLI-contract decisions (`-version`; scoped help, the bareword `help`, and the help-spelling variants a third persona pass just added) |
| 8 | #472 | implement `goxsd8 parse`, the first non-stub subcommand |

**#442 held row 1 and has landed** (PR #730, `1dfd13c`, `schema` 9751 → 9833).
Its row is dropped rather than replaced. **That row's text was wrong about its
own issue** — it read *"top-level `xs:attribute` with an inline
`xs:simpleType`"*, and #442 was about `xs:element`. The `xs:attribute` analogue
(§3.2.2.1 `dcl.att.global`) is real, still unproduced, and was owned by no issue
precisely because this row read as though it were already in flight; it is now
**#733**. #442's reclassified cases are **#731** and **#732**. None of the four
is banded — ranking them is a re-derivation this pass does not do.

**#710's landing still stands two rows back**: the M5 chain's head is #711, and
#713 — the lane driver that takes `instance` off zero — is two slices behind it.

**Rows 4–7 sit ahead of row 8 on cost, not importance.** Each is a sentence
or a dispatch branch while the CLI surface is still empty; taken after #472
every one of them is a change to shipped behaviour. #472's own Acceptance carries the `-version` decision, so if it is
taken first it must discharge #672 rather than leave it contradicting the
landing. **#720** (`goxsd8 validate`) is `blocked` behind #472 and #715 and is
not a band row.

**#503 landed** (PR #724, `c7d178f`, `schema` 9745 → 9751), so the #503/#504
coupling now has a direction rather than two safe orderings: **#504's landing
carries the decidability widening.** The coupling itself is unchanged and
durable on `main` — retiring `conformance/schema.go`'s decidability residue for
`group`/`attributeGroup` flips `MS-Schema2006-07-15/schL10` and `/schM5` from
`pass` to `fail` unless **both** clauses are in hand, because each case turns on
one clause alone — and #503 correctly touched no gate, which is the only safe
shape for whichever goes first. The binding comment is in `redefineDecidable`
(`conformance/schema.go`). Its **prose** is now stale there and at three other
sites, two of which still name the closed #503; that is **#726**, prose-only and
forbidden to touch the gate code. #503's landing also left one `GAP(xsd)` naming
no issue, now **#725** — so `gapaudit`'s group 1, reported empty above, has an
owner again even though the marker text will not name it until #726 lands. Both
were filed by #503's post-land pass, which **unblocked nothing**: all 28
`blocked` bodies were scanned and none names #503 in any section. Every count in
this section predates that pass — it closed one `ready` issue and filed two, and
a post-land pass corrects no number here; the next `/backlog` re-derives them
with the rest of the section.

**Five issues are deliberately unbanded — #702, #703, #706, #721 and #722 —
as are #711 and #712, and the seven M5 slices still `blocked`, which are not
`ready` and so cannot be banded at all.** The band's doctrine is "slices that
move a lane", and none of the five does — #711 and #712 do, and are unbanded
only because #710's post-land pass ranks nothing.
#702 sweeps 17 comments citing §4.2.4 clause 4.1.1
for what `src-expredef` says; #703 is a duplicate top-level declaration never
being produced at all; #706 is a real false-accept but a measurably unreachable
one — no document in the 15470-file suite carries both an `xs:override` and an
`xs:redefine`, so it is unit-test-only. #721 wants a warden call on two shapes
before it is workable. **#722 is the one to promote if a seventh pass wants a
cheap high-value row**: it is the only unbanded issue whose absence can cause a
future pass to actively damage the queue.

**`parser/redefine.go` has been rewritten four times in five days, so read it
rather than an issue body that describes it.** #686 put its duplicate-key
charge in `recordOriginal`; #699 added `rejectProhibitedAttrs` to
`newRedefineSet`; #506 inserted ~45 lines and moved every marker line number
the #585 and #505 threads cite; #503 added the clause-7.1-vs-7.2 branch,
repointed the `chainedOriginal` marker to #504 alone and deleted the clause-7.2
gap. Row 4 (#504), #706 and #726 all open that file next.

**Next planning action: the queue-wide reconciliation this pass bought its
carve by skipping.** Step 3 has not run in full since 2026-08-10, and this
pass audited a stale premise in exactly one body (#250's, by comment) out of
142 `ready` issues. Run it against the whole `ready` set and re-derive the band
from the result rather than re-ranking its head. **The M5 condition this
section set is discharged**: #710 landed and #711/#712 were unblocked by its
post-land pass, so the chain now has two `ready` heads (#711, #712) competing
with the band's M4 rows — that ranking is the reconciliation's, and it is the
first thing it should settle.

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
