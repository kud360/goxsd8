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

## Status — 2026-08-16 (post-land, #800)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1324 | 25037 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12836 | 2562 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This stamp absorbs FOUR landings, not one.** The previous stamp was #722's,
which moved no lane, and the three landings after it filed no docs commit here
(#400). `instance` 1017 → **1324** across **#718** (+116, banked `ecc6e11`),
**#716** (+183) and **#813** (+9, unioned onto #716's at `2d0a38d`); `schema`
12832 → **12836** across **#800** (+4, banked `04f2c55` — `CTA/cta0043` and
`Wild/wild078|079|081`, the four dead `{type table}` readers waking);
`datatypes` 1161 untouched since M3 closed. The per-landing narrative is
`docs/LOG/2026-08.md`, not here.

Milestones, read from GitHub this pass. **M0–M2 and M3 are carried**: both are
complete, and no issue filed since the last stamp carries either.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 83 | 42 | **active** |
| M5 — Instance validation (XML) | 13 | 12 | **active** |
| M6–M12 | 0 | 0 | not filed |

Queue, read fresh across all three pages: **202 open issues — 180 `ready`, 22
`blocked`, 0 `needs-replan`, 2 `epic`** (both `blocked`), against **294
closed**. 180 + 22 = 202 exactly, so the arithmetic proves the label sweep
rather than a sweep asserting it, and no open issue carries neither queue label.
Read the milestone table as feature progress, not as the queue: **148** of the
202 carry no milestone (202 − 42 − 12).

**The previous stamp's one unexplained issue is still exactly one, and has not
grown.** That stamp published 193 open / 289 closed; eleven issues were filed
since (#809–#823 less the PR numbers) and three of its 193 have closed (#718,
#716, #800), which predicts 201 against the 202 measured. #813 was filed and
closed between the two stamps and never entered either open count. Treat 202 as
the measurement; the next full `/backlog` reconciles, and does not re-derive
from either figure.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim.** `gh` is 403 on both GraphQL and REST in this
container (#527, #682), so this is the lease-only report the tool's own usage
block documents — `go tool wipsurvey < /dev/null`:

```
ISSUE  BRANCH  TIP AGE  VERDICT  REASON
```

**The namespace holds `main` and nothing else** — `git ls-remote --heads
origin` returns one ref. Every branch the previous stamp listed as deletable is
gone, and `wip/issue-718`, `wip/issue-716`, `wip/issue-813` and `wip/issue-800`
were auto-deleted by GitHub at merge, as the setting intends. Nothing is
`CLAIMED`, nothing is `EXPIRED`, there are no `parked/*` refs, and no branch
needs a human look: **the next develop session claims from `ready` and resumes
nothing.** The shallow-clone finding (#802) still binds whatever the next
non-empty namespace reports.

**`go tool gapaudit` could not run, for the same 403** — a third consecutive
session, recorded in `docs/LOG/2026-08.md` at #813 and again at #800, which is
the argument #682 makes. The one check this pass could make by hand is the one
#800 owed: `grep -rn '#800' --include=*.go` is **empty**, so the three
`GAP(xsd)` markers that landing added name #822, #821 and #823, and no marker in
the tree cites a closed issue through it. The census itself is owed wherever
`gh` is authenticated.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 180 of
them. Take from the top. **Rows 1–3 of the previous band have all landed** —
#718, #716 and #800 — and rows 1 and 2 below are #800's own children, filed
with measured corpus figures. The rest is carried from the `2d6ec5c` full
re-derivation with its justifications intact; no landing since falsifies one.

| # | Issue | Why here |
|---:|---|---|
| 1 | #821 | **·xs:error· is seeded as no component at all**, so `type="xs:error"` is FALSE-REJECTED at `src-resolve` clause 1.1 — reproduced through `parser.Parse` on a schema whose whole body is one element declaration. 22 suite schema documents name it on an `<xs:alternative>`, **18 of them un-withheld by this issue alone**, and one more names it on an `<xs:attribute>`. Measured `schema` headroom, no dependency, and it is the only thing #823 waits on |
| 2 | #822 | the other half of #800's withheld set: `TypeAlternative` carries `{type definition}` as a QName ONLY, so §3.12.2's inline arm — and a synthesized `{default type definition}` over an anonymous declared type — withholds the WHOLE `{type table}` of the declaration. **46 inline-arm alternatives across 20 schema documents.** Its §3.4.2.1 ownership grounding is a first step inside its own scope, not an external dependency |
| 3 | #733 | **promoted four places.** A top-level `xs:attribute` with an inline `xs:simpleType` is unproduced — #442's attribute analogue, and **#442 moved `schema` +82**. It was banded below five rows that move no lane at all, which is precisely what four passes of carrying produced |
| 4 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced. Its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `<list>` and `<union>`; converting it turns declines into decisions in `schema` |
| 5 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. Below #733/#786 because it converts silent wrongness into honest declines, which the lane scores the same. It also decides the encoding **#56** needs one milestone later |
| 6 | #625 → #669 → #748 → #492 | **README's Library block, now ONE row in file order.** Splitting it across two band rows is why it sat four passes. #625 fixes the `SchemaBuilder` pointer at closed #203 (:123-124); #669 the "works TODAY" snippet, the example list and `go doc ./...`; #748 the M5 block that denies a shipped API (:126-133); #492 `ParseReport`, which belongs in the sentence at `README.md:116` rather than a new paragraph |
| 7 | #747 + #514 + #687 + #672 | the CLI contract, all four decided **before** #472 — the missing "Implemented today" paragraph, typo-vs-unbuilt, scoped help, `-version`. Each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 8 | #472 | implement `goxsd8 parse`, the first non-stub subcommand — and the issue that owns the exit-2 split, plus the forward-compat sentence folded onto it |
| 9 | #682 + #668 + #802 | **the container tax, actionable half.** #682 and #668: `wipsurvey` and `gapaudit` both have a working no-`gh` mode documented only in their own package docs, so the three places a session reads show a dead `gh issue list \|` pipeline. #802: the clone is shallow and nothing says so. **This pass paid all three by hand for the third stamp running.** **#659** and **#527** are the prose half of the same paragraph and land with it |

**Deliberately unbanded, and why.** **#744** (chained-redefine successor, highest-
value M4 gap), **#773** and **#721** are held out on one shared condition: each
needs a warden pre-flight on an entry point that does not exist. **That condition
has an owner — #484** — which is why a process issue is worth more than its label
suggests here: it unblocks three band-grade issues at once. **#774** is three
fail-open declines to close or rule permanent; which way it moves a lane is
unknown until the first is decided, and **#795** pins the element-side one either
way. **#771** is now — not "once #800 lands" — the `instance` lane's LAST
decided-and-disagreeing case. **#823** is the one `blocked` issue this landing
created and it joins the band the moment #821 closes; do not start it early,
because a clause-7.1-only charge is sound today and wrong the instant #821
lands. **#782**, **#783**, **#785**, **#787**, **#788**, **#793**, **#794**,
**#796** are earlier landings' follow-ups; **#805**, **#806**, **#809** and
**#810** are `tools/wipsurvey` hygiene on a tool that now works; **#812**,
**#814**, **#817**, **#819** and **#820** are #718's, #716's and #813's, filed
by passes that left no PLAN commit; **#825** is this pass's own, the two
sub-threshold nits #800's ACCEPT flagged and its landing did not absorb.
**#815** is the standing marker-repoint
seam, and **#800 is its first worked example** — that landing discharged its own
repoint inside the PR rather than leaving it to a post-land pass, which is the
shape #815 is asking for; its thread carries the note, and one row of its
Acceptance table is now retired.

**`parser/redefine.go` has been rewritten five times in a week, so read it rather
than an issue body that describes it.** #686, #699, #506, #503 and #504 each
moved it; #744, #706 and #726 all open it next.

### Next planning action

**A full `/backlog` is owed, and this stamp is its own evidence.** Three
landings ran between #722's stamp and this one without touching this file, so
the working band spent a day pointing a develop session at three closed issues
and the lane table understated `instance` by 307. **#400** is the mechanism — a
pass that files no docs commit leaves no signal on `main` — and it is still
open. The band above is a post-land repair of that drift, not a re-derivation:
its rows 3–9 have now been carried through two stamps.

**Re-derive the decline census before the next carve.** It predates #766, #715,
#740, #790, #718, #716 and #813 — and is now, by a wide margin, the oldest
measurement this roadmap argues from. Rows 4 and 5 of the band above are ordered
on *reasoning* about which declines are convertible, not on a measurement, and
that is as far as judgment can carry the ordering. **#570** is the issue that
makes it cheap and permanent: bank a per-lane decline baseline so every landing
announces the cases it just made decidable, instead of each pass re-running a
standing count. **#571** is its soundness half.

The follow-up-ledger debt this section carried for eleven stamps is **discharged
and closed** (#489); do not re-file it. WORKFLOW step 7(b)'s *a hand-off is not
a disposition* rule is what closed the inflow.

Everything else this queue needs is a develop iteration. **Start with #821.**


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
Twelve have landed: the infoset
seam and engine skeleton (**#710**), the `xmlsrc` adapter (**#711**) and root
dispatch (**#712**), none of which moved the lane and correctly so — the first
two decide no `cvc-` rule and nothing yet ran the cases; **the lane driver
(#713), which took `instance` off zero — 0 → 18 pass** (the #175 analogue, placed
fourth so every slice after it reports a real number); `cvc-complex-type`
clauses 2–3 (**#714**, 19 → 29); the `value.Backend` seam with the four
datatype-valid attribute charges (**#766**, 29 → 193); the greedy
non-backtracking content matcher (**#715**, 193 → 520); `cvc-complex-type`
clause 1.2's ·initial value· against String Valid (**#775**, 532 → 535), which
also closed #759 and landed `docs/STYLE.md` **E4**; **the descent (#790,
535 → 1017)**, which threads each descendant's ·context-determined declaration·
into the recursive walk (§3.3.4.6 clause 3.1) — the largest single move any lane
has recorded; identity constraints and the ID/IDREF table (**#718**,
1017 → 1133); `xsi:type` and `xsi:nil` deciding rather than declining (**#716**,
+183); and a union-governed item classified by its ·validating type· (**#813**,
+9, unioned onto #716's). `instance` stands at **1324** — #740, an M4 landing,
took it 520 → 532 on a merged tree neither parent could measure.

**The milestone's shape changed with #790, not just its number.** The first eight
slices decided the ·validation root· and nothing else, so each one bought tens of
cases; #790 bought 482 by making the same rules reach the other 99% of every
document. **The remaining slices inherited that multiplier and spent it**: #718
and #716, the band's top two at the time, bought a further 299 between them by
deciding their rules at every node rather than one.

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

**1324 is still a floor built for soundness, and no jump has changed what the
number means.** The lane emits only "not valid" observations; a violation-free
`Result` DECLINES rather than passing, because `Assess` evaluates none of
`e-validity`'s other conjuncts. **Every passing case is an expected-INVALID one
by construction**, not by measurement, and the 25037 that still fail are
overwhelmingly declines rather than disagreements. The milestone's remaining
slices are what turn declines into decisions.

**Do not read 1324 as 5% of the suite passing.** It is the count of documents
this engine can honestly call not-valid, and it grew because the same rules now
reach every node instead of one. A slice that decides a *new* rule will move the
number far less than #790 did and be worth more.

**ONE case is decided and decided WRONG — #771**, a root whose declaring schema
is reachable only through the instance's own `xsi:schemaLocation`. It was four,
and #800 retired two of them: `Assert/assert_019/instance/assert_019_2` and
`CTA/typeAlternatives_001/instance/typeAlternatives_001_2` now decline honestly
instead of rejecting a document the ·conditionally selected· type admits.
**That is two, not three, and `CTA/cta0008.v01` was never among them** — it
takes §3.12.2's inline arm, which #800 deferred to **#822**, and the count was
measured by diffing `GOXSD_DECLINES=1` across the two trees rather than
predicted. All were already banked `fail` before #790 and are not its
regression; the descent is what made them visible, and the trade of a wrong
decision for an honest decline is one the lane cannot register as movement.

The decline census that separated harvest candidates from indeterminates
predates #766, #715, #740, #790, #718, #716 and #813 — and is not re-derived
here. **It is now, by a wide margin, the oldest measurement this milestone still
argues from**, and #786 is the nearest issue to it.

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
