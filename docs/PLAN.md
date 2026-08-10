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

## Status — 2026-08-10

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1156 | 17 | 1173 |
| `instance` | 0 | 26361 | 26361 |
| `json` | — | — | 0 |
| `schema` | 9648 | 5750 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a
lane scoring zero. `datatypes` is M3 and effectively complete; `schema` is
M4 and active; `instance` waits on M5; `xpath`, `json` and `ber` wait on
M6/M7, M8 and M11.

Milestones, from GitHub:

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 1 | complete; one follow-up open |
| M4 — Schema parsing | 71 | 43 | **active** |
| M5 — Instance validation (XML) | 0 | 1 | epic #250 filed, **uncarved** |
| M6–M12 | 0 | 0 | not filed |

Queue: 152 open issues — 137 `ready`, 15 `blocked`, 0 `needs-replan`,
2 `epic`. **45 of the 152 carry a milestone; 107 carry none** — the
milestones track feature scope, and the process, doc and comment-accuracy
issues that post-land passes file are deliberately outside them. Read the
milestone table as feature progress, not as the queue.

Branch namespace, `origin`: `wip/issue-256`, `wip/issue-271` and
`wip/issue-287` survive, all three retired in place with their issues
closed and superseded (#256 → #470; #271 → #478/#479/#480; #287 answered
in the opposite direction by PR #511). Listed for human triage; a session
never deletes a ref.

**Next planning action: carve M5 (#250) — still owed, and this pass did
not do it.** Its stated precondition, #180–#183 draining, has been met
since 2026-08-09. Follow the pattern that worked for M4 (#79 →
#167–#183): model and infoset shape slices first, then a lane bring-up
slice that produces a real `instance` number early (the #175 analogue),
then the validator fan-out. The carve is a whole pass's work — it wants a
`/backlog` that spends its budget there rather than on reconciliation.

**Working band** — dependency-ordered top of the `ready` queue, so a
session need not scan 137 issues. Take from the top; the ordering prefers
slices that move a lane over horizontal completeness.

| # | Issue | Why here |
|---:|---|---|
| 1 | #632 | declaration `name` is never NCName-checked — **28 banked `fail` schema-lane cases are this one fault**, the largest named lane movement in the queue |
| 2 | #631 | `type=":T"` empty-prefix false accept — the prefix half of #343, same lexical-name family as #632, cheaper once #632 is in hand |
| 3 | #523 | nameless top-level `simpleType` produces without error — third of the same family |
| 4 | #518 | top-level `element`/`attribute` never reads `ref`; seven suite cases |
| 5 | #652 | the rest of the prohibited top-level attributes. **Depends on #518** — it is the half #518 declined |
| 6 | #585 | a chained `redefine` is rejected — false-reject on a valid schema |
| 7 | #506 | an `override`-substituted `group` child loses self-reference resolution — false-reject |
| 8 | #469 | cos-all-limited charged on the XML spelling, not the component — false accept in both clauses |
| 9 | #442 | top-level `attribute` with an inline `simpleType` unproduced; a banked fixture rests on the decline |
| 10 | #447 | `simpleType` with a `union`/`list` body unproduced — `datatypes` lane, pdecimal019/020 the measured cost |
| 11 | #626 | README's CLI section is false about exit codes today; one sentence, and it should precede #472 |
| 12 | #669 | README's Library snippet does not compile, and the example pointer omits `parser` and `xsd` |
| 13 | #514 | a typo'd subcommand and an unbuilt one are indistinguishable — fix before #472 makes it misleading |
| 14 | #472 | implement `goxsd8 parse`, the first non-stub subcommand |

Rows 1–10 move `schema` (row 10 moves `datatypes`); rows 11–14 are the
M4-facing published surface, where two isolated persona passes have now
found the same defects twice.

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
