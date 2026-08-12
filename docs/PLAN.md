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

## Status — 2026-08-11

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1156 | 17 | 1173 |
| `instance` | 0 | 26361 | 26361 |
| `json` | — | — | 0 |
| `schema` | 9707 | 5691 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a
lane scoring zero. `datatypes` is M3 and effectively complete; `schema` is
M4 and active; `instance` waits on M5; `xpath`, `json` and `ber` wait on
M6/M7, M8 and M11.

`schema` moved **9648 → 9706 pass, +58**, across the five landings since the
last pass — all five of them the top five rows of that pass's working band,
and all five one grammar family: #632 (NCName declaration names, +28), #631
(empty QName prefix), #523 (nameless top-level `xs:simpleType`), #518
(top-level `ref`) and #652 (the rest of the prohibited top-level attributes,
+25). A band drained in family order is the ordering doctrine working; row 1
below continues the same family. **#585 landed mid-pass** (PR #689,
`schema` 9706 → 9707, +1, `MS-Additional2006-07-15/addB007`) after this
section was first drafted; the table above is the merged tree's true figure,
so the six-landing total is **9648 → 9707, +59**.

Milestones, from GitHub:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 1 | complete; one follow-up open |
| M4 — Schema parsing | 75 | 42 | **active** |
| M5 — Instance validation (XML) | 0 | 1 | epic #250 filed, **uncarved** |
| M6–M12 | 0 | 0 | not filed |

Queue: 156 open issues — 140 `ready`, 16 `blocked`, 0 `needs-replan`,
2 `epic`. **44 of the 156 carry a milestone; 112 carry none** — the
milestones track feature scope, and the process, doc and comment-accuracy
issues that post-land passes file are deliberately outside them. Read the
milestone table as feature progress, not as the queue.

The queue grew by five across five landings: five issues closed, ten filed.
Eight of the ten are named follow-ups from one of the five landings, none of
them a hand-off (#330) — the post-land passes working as specified. The other
two (#687, #688) came from this pass's persona reports and are the only
issues here that no landing produced. **#585 landed mid-pass** (PR #689,
merged after this section was first drafted but before this branch's base
was refreshed) — a sixth closure, counted in the totals above but not in the
"five landings" figure, which still names the reconciliation this pass
actually performed.

Branch namespace, `origin` — report-only; a session never deletes a ref:

| Branch | Verdict | Disposition |
|---|---|---|
| `wip/issue-256` | EXPIRED | Retired in place; issue closed and superseded by #470. |
| `wip/issue-271` | EXPIRED | Retired in place; issue closed and superseded by #478/#479/#480. |
| `wip/issue-287` | EXPIRED | Retired in place; issue closed, answered in the opposite direction by PR #511. |

No `parked/*` branches. The three EXPIRED refs are listed for human triage
and have been for several passes; `wipsurvey` reports them EXPIRED rather
than RETIRED because a `gh`-less container runs it lease-only and cannot see
issue state (#682).

**Next planning action: carve M5 (#250). This is the THIRD consecutive pass
to name it and the third not to do it** (2026-08-09, 2026-08-10, today). Its
stated precondition, #180–#183 draining, has been met since 2026-08-09, so
nothing is blocking it but budget. Follow the pattern that worked for M4 (#79
→ #167–#183): model and infoset shape slices first, then a lane bring-up
slice that produces a real `instance` number early (the #175 analogue), then
the validator fan-out.

The reason it keeps sliding is now legible and should be acted on rather than
restated: each pass finds enough real reconciliation — stale bodies, undisposed
follow-ups, a band to reorder — to consume its whole budget, and the carve is a
whole pass's work on its own. **The next `/backlog` should carve M5 and do
nothing else**, accepting a stale band for one cycle. A fourth naming without
that is evidence the trigger is wrong, not that the queue is busy.

**Working band** — dependency-ordered top of the `ready` queue, so a session
need not scan 140 issues. Take from the top; the ordering prefers slices that
move a lane over horizontal completeness. **#699 (PR #705) has landed** and is
dropped from the band below rather than left as a stale claimed row; #686
(PR #701), #684 (PR #698), #506 (PR #694), #675 (PR #695) and #585 (PR #689)
were dropped the same way in the four passes before it.

**Every row below is now the 2026-08-11 `/backlog`'s own ordering,
contiguously** — its rows 5–14, renumbered. The two rows a post-land pass
placed rather than a queue-wide survey, #686 and then #699, have both drained.
Re-rank at the next `/backlog` — which, per the mechanism stated above, carves
M5 and re-ranks nothing.

**Three issues filed by the last two post-land passes are deliberately
unbanded: #702, #703 and #706.** The band's ordering doctrine is "slices that
move a lane", and none of the three does. #702 sweeps 17 comments citing
§4.2.4 clause 4.1.1 for what `src-expredef` says; #703 is a duplicate
top-level declaration never being produced at all, which is why
`indexByName`'s clause-2 message cites one location twice; **#706 is a real
false-accept but a measurably unreachable one** — an
`xs:override`-substituted `xs:redefine` child bypasses the guard #699 just
landed, and a walk of all 15470 suite `.xsd` files finds no document carrying
both an `xs:override` and an `xs:redefine`, so it can move no lane and is
unit-test-only. **#702 touches `parser/redefine.go`, `parser/doc.go`,
`parser/parse.go`, `parser/produce.go` and `conformance/schema.go` — the files
row 2 opens**, and it is cheaper for that landing to read a corrected comment
than to copy the wrong clause number a third time.

| # | Issue | Why here |
|---:|---|---|
| 1 | #469 | cos-all-limited charged on the XML spelling, not the component — false accept in both clauses |
| 2 | #503 + #504 | `src-redefine` clauses 7.2.2 and 6.2.2, both fail-open. **Coupled — read the note below before starting either** |
| 3 | #442 | top-level `xs:attribute` with an inline `xs:simpleType` unproduced; a banked fixture rests on the decline |
| 4 | #447 | `xs:simpleType` with a `union`/`list` body unproduced — `datatypes` lane, pdecimal019/020 the measured cost |
| 5 | #590 | pdecimal016, the Saxon PDecimal cohort's five-case chain — `datatypes` lane, #574's sibling widening |
| 6 | #626 | README's CLI section is false about exit codes today; one sentence, and it should precede #472 |
| 7 | #669 | README's Library snippet does not compile, and the example pointer omits `parser` and `xsd` |
| 8 | #514 | a typo'd subcommand and an unbuilt one are indistinguishable — fix before #472 makes it misleading |
| 9 | #672 + #687 | the two open CLI-contract decisions (`-version`; scoped help and the bareword `help`). One line each in `doc.go` now, a retrofit around a live dispatch after #472 |
| 10 | #472 | implement `goxsd8 parse`, the first non-stub subcommand |

**`parser/redefine.go` has been rewritten twice in two days, so read it rather
than an issue body that describes it.** #686 put its duplicate-key charge in
`recordOriginal` and retired the `GAP(xsd)` there; #699 then added
`rejectProhibitedAttrs` to `newRedefineSet`, a different feeder of the same
struct. Row 2 and #706 both open that file next.

Rows 1–5 move `schema` (rows 4 and 5 move `datatypes`); rows 6–10 are the
M4-facing published surface, where two isolated persona passes have now found
the same defects twice.

**Rows 6–9 are ordered before row 10 on cost, not importance.** Each is a
sentence or a dispatch branch while the CLI surface is still empty; taken
after #472 every one of them is a change to shipped behaviour. #472's own
Acceptance already carries the `-version` decision, so if it is taken first it
must discharge #672 rather than leave it contradicting the landing.

**Row 2's coupling is now landed on `main`, not only on an issue thread.**
#585's warden review established that retiring the `conformance/schema.go`
decidability residue for `group`/`attributeGroup` flips
`MS-Schema2006-07-15/schL10` and `/schM5` from `pass` to `fail` unless
**both** #503 and #504 are in hand — each case turns on one clause alone, so
whichever of the two lands second must carry the widening with it, and
neither may retire the residue on its own. #585 landed with the binding
comment written into `redefineDecidable` (`conformance/schema.go:648-666`,
naming #503/#504 directly), so the coupling is now durable on `main` rather
than tied to a branch's lifetime; it is also restated on both issue threads.

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
values. **`instance` lane.** Filed as an epic and still uncarved; the
carve is the Status section's named next planning action.

### M6 — XPath required subset

CTA restricted subset plus assertion essentials, fail-open with `GAP`
markers, IDC selector and field paths. Dynamic-error direction per
PRINCIPLES 20. Not filed as an epic — speculative epics two milestones out
earn nothing.

One dangling dependency to repoint at the carve: **#56** (a per-assertion
or CTA result must distinguish a genuine PASS from a fail-open
"unevaluated") is blocked on the not-yet-filed evaluator issue. STYLE 9's
fail-open discipline is only honest if a fail-open answer is
distinguishable from a real pass.

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
