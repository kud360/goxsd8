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

## Status — 2026-09-03 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **FIVE** landings since the last stamp — the last band's rows 1 through 3 (#1174, #1164, #1182) taken in order, plus **#404** and **#471** from the M4 tail — with **#1160 completing its post-land pass** in the same window. It measured **`schema` +28 across two of the five** (#1182 +17, #404 +11), saw **#1164 rule the 584/588 near-match a coincidence** — the standing question four stamps carried, now closed — watched the census open a **SEVENTH-then-EIGHTH area** with the first `GAP(conformance)` markers, found the persona surface **byte-identical since the fourteenth consultation** so ran no fresh pass, corrected **#1156**'s now-stale census absolute, fixed the M4 producer paragraph's **#471 stale token**, and **dismissed the `src-element` clause 4 follow-up #471's post-land deferred here**. Five new issues sit filed and `ready` — #1196, #1199, #1201, #1203 — with #1205 and #1206 on M4)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13896 | 1502 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### Two of five landings moved `schema`, and NEITHER carried a milestone

**`schema` 13868 → 13896 (+28); `instance` flat at 11029; `datatypes` flat.**
The two movers are attributable to the line and the other three said `unchanged`
and were right to.

| landing | commit | lane movement | milestone |
|---|---|---|---|
| **#1174** | `cf8d3bb` | unchanged | none |
| **#1164** | `2cc7ca6` | unchanged | none |
| **#1182** | `933f5ec` | **`schema` +17** | none |
| **#404** | `62d5143` | **`schema` +11** | none |
| **#471** | `31263ed` | unchanged | **M4** |

**#1182** admitted the bare and prohibited-attribute `<group>`/`<attributeGroup>`
forms at all five decidability-gate sites — the producer already rejected them
as s4s attribute-use faults, so the declines were conservative and admitting
them fabricated no verdict — and banked +17 on `schema` alone, zero regressed.
**#404** replaced the schema lane's blanket `Unfollowed() > 0 && perr != nil`
conjunction with `fabricatedRejection`, a two-arm predicate that declines only
the failures a missing document could actually have fabricated, and banked +11.
**#471** rejected a local `<element ref=>` carrying `substitutionGroup=`
end-to-end — grammar-level `use="prohibited"`, so no case in the corpus to bank,
`Ratchet: unchanged` and right to be.

**The two lane movers carried NO milestone; the one M4 landing moved no lane.**
That is the sharpest restatement yet of the floor caveat both milestone sections
make: read neither milestone count as its lane's remaining work. It is the same
shape the last two stamps recorded (#438 and #786 moved the lane carrying no
milestone; #472 and #748 moved a milestone count carrying no lane).

### #1164 ruled the 584/588 near-match a COINCIDENCE, and the standing question is closed

Four stamps carried *"read a residual bucket count as a candidate filter and
never as a sort key"* as a rule with an open question under it — whether the
584-predicted-588 pair and #786's exact-10 hit could both be weighed as
predictions. **#1164 landed the ruling at `docs/PLAN.md:38-97`**: the chain from
bucket to lane is `584 discoveries → 580 documents → 475 movers → 588 lines`,
whose three factors (0.993 documents per discovery, a yield of 0.819 movers per
bucket document, a fan-out of 1.238 lines per mover) multiply to 1.007, so the
agreement is an 18% shortfall and a 24% fan-out cancelling and NONE of the three
is statable before a landing. The 584 was re-derived at `adb6d57`, the commit it
was measured on, not quoted. **The rule now carries a case-by-case
reconciliation under it instead of an open question, and no stamp may cite
584-predicted-588 as a prediction again.**

**Its live consumer is filed: #1196** buckets the residual AT HEAD, so every
per-class count the tree quotes stops coming from a measurement commit five
landings back. The last stamp's 45-vs-46 gap between the label-sum and the live
instrument is what motivated it; #1182 widened the gap by moving 17 expectation
lines against two rows labelled 9 + 9 document discoveries. **#1196's own body
forbids banding on the numbers it produces** — the ruling above is why — so it
sits below the band as an opportunistic feed to #1051, not a queue slice.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and this
pass's 683-issue post-write fetch:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
732    wip/issue-732   260h54m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   438h52m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   211h13m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   404h53m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   333h6m0s   RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   272h11m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   216h33m0s  RETIRED  wip/issue-993: issue #993 is closed
1181   wip/issue-1181  19m0s      LIVE     wip/issue-1181: tip pushed 19m0s ago, within the 2h0m0s claim TTL
```

**`wip/issue-1160` is GONE** — deleted on merge, so the last stamp's one LIVE
lease (#1160) closed cleanly and its only churn was a clean open-and-close, the
second consecutive stamp in which that happened.

**ONE LIVE lease, and #1181 is mid-flight as this pass runs** — its tip pushed 19
minutes ago, inside the 2h claim TTL. **#1181 is therefore NOT banded below.** It
was band row 4 last stamp; it is being taken now. Its body records that it
rewrites more producer-mechanism prose than #786 or #1182 did, which is why
**#1199** (row 1) names it as the likely third instance of the defect that issue
tracks.

**All seven RETIRED refs closed `not_planned`** — parks and supersedes, their
content *supposed* not to be in `main`, none owing a supersede. Cloud containers
cannot delete remote refs, so these accumulate by design and are not a finding.
**Zero `parked/*`.** Three non-`wip` `claude/*` refs stand:
`claude/eloquent-cerf-39rk64` (`0abeab6`) and the NEW
`claude/eloquent-cerf-3xu0ki` (`62d5143`, exactly #404's squash) are both
ancestors of `main` and carry nothing. **`claude/eloquent-cerf-8jq9o6`
(`7841e98`) reads NOT-an-ancestor in this container, and that is a shallow-clone
artifact, not divergence (#802).** `7841e98` is *"meta: post-land pass for #853
(#1121)"*, a landed squash dated 2026-08-29 that is on `main`'s real history but
sits beyond this container's 50-commit horizon, so `git merge-base` returns empty
and `--is-ancestor` reports false. The last stamp, with `7841e98` still in
reach, correctly called it an ancestor carrying nothing; the tip has not moved.
Listed for human triage, not acted on.

**The shallow-clone premise is unchanged and this pass has the sharpest witness
of it yet.** This container sees **50 commits** of `origin/main`, so every
retired ref's disposition and the `8jq9o6` ancestry read came from GitHub rather
than `git log`. **#802** owns this and is open.

### Marker census

`go tool gapaudit` over this pass's whole 683-issue feed: **69 markers across 8
areas** — `xsd` 33, `validate` 17, `xpath` 6, `xml` 4, `parser` 3, `value` 3,
**`conformance` 2**, `cmd` 1.

**Census 67 → 69, and an EIGHTH area appeared.** #404 landed the project's first
two `GAP(conformance)` markers, at `conformance/schema.go:287` and `:618`, both
citing OPEN **#1201** — the marker convention working on its first use in a new
package, exactly as `GAP(cmd)` did last stamp. Because they cite an open issue
they sit in neither unreconciled group. Net of #471's one deletion (its retired
`GAP(xsd)` paragraph) and #1182's churn.

**Group 1 held at 17 and group 2 at 25**, both unchanged from the last stamp
across the five landings. **ZERO group-1 rows carry no annotation at all, so the
tool's own filing rule selects nothing and there are zero untracked GAP sites —
fifth consecutive stamp.** Every group-1 row prints at least one candidate owner
or resemblance; the two new `conformance` markers are tracked and out of both
groups.

**Two of the seventeen surviving group-1 rows are instrument defects, not
ownership defects, and #1156 owns both** — unchanged. **This pass corrected that
issue's now-stale census absolute**: its "Expected totals afterwards" line pinned
census **67**, which #404's two markers falsified; it now reads **69** with the
reason, and its baseline at `c720206` (67, still true at that commit) and its row
deltas (group 1 17 → 15, group 2 25 → 24) stand exactly as filed.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 13 pages: **683 issues, 253 open, 430 closed**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **49** | **108** | active |
| **M5 — Instance validation (XML)** | **11** | **19** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **234 `ready`, 19 `blocked`, 0 `needs-replan`** — every
open issue carries `ready` or `blocked`, verified mechanically, and the two sum
to 253 with no gap. By kind: `kind/refactor` 71, `kind/process` 54, `kind/gap`
52, `kind/tooling` 30, `kind/bug` 26, `kind/story` 17, `kind/docs` 12,
`kind/feature` 6, `epic` 2. By area: `parser` 73, `meta` 67, `xsd` 58,
`conformance` 31, `validate` 22, `docs` 18, `value` 14, `cmd` 12, `builtin` 10,
`xpath` 6, `xsderr` 3, `loader` 2, `regex` 2.

**M4 moved 48 → 49 open and 107 → 108 closed: #471 closed and #1205/#1206 filed
onto it, a net of +2 open and +1 closed.** M5 is flat at 11/19 — no M5 landing
this window. **Neither M4 mover was a lane mover and the two lane movers carried
no milestone at all** — the count-versus-lane divergence the previous section
tabulates.

**`ready` overstates startable work by ONE**, as it did last stamp: #1181 is
`ready` and claimed. The honest startable count is **252**. Every post-land pass
this window left **zero open issues carrying no state label**, verified again
here (`ready` + `blocked` = 253 = open).

### Persona consultations — the fourteenth stands; the surface has not moved since

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **This pass ran no fresh consultation, and that is the correct call,
not a skipped step.** The published API and CLI surface is byte-identical to what
the fourteenth consultation reviewed: `git log c720206..origin/main --
README.md cmd/goxsd8/` returns **zero commits over the 13 that landed since**, so
there is no new surface for an outsider to observe. The fourteenth consultation's
findings — #1188, #1189 (CLI behaviour) and the reconfirmed documentation rows —
remain live in the queue and in the band; nothing about them changed.

**The next full consultation is still owed against #720's landing** (band row 4),
which is the next thing that changes what a cliuser can observe. Until #720 or
something touching README's Library section or the `xsd` query surface lands, a
fresh pass would re-review an unchanged binary. **#1033 remains the one row no
persona has ever looked at.**

**No narrow persona ask this round.** The two CLI behaviour defects the
fourteenth consultation surfaced (#1188, #1189) are already filed with settled
forks and need no persona to proceed; nothing this pass's survey turned up poses
a question only a persona could answer.

### Working band

**Re-derived from this pass's evidence.** The last band's rows **1–3 all landed,
in order** (#1174, #1164, #1182); **#404 and #471 landed from below the band**;
**#1181 (row 4) is IN FLIGHT** — take it and you collide with a live claim. Take
from the top; re-run `wipsurvey` first.

| # | issue | why here |
|---|---|---|
| 1 | #1199 | **LANDED 2026-09-03 on a ruling — squash `b57d37d`, PR #1212 — no new rule is warranted, and this row is closed out here only until the next `/backlog` re-derives the band.** Its base is two consecutive round-1 rejects in one predicate family, the second inside the landing that repaired the first. Read at the verdicts rather than at an account of them: #786's reject was carried by a `[tests that cannot fail]` finding on a FIXTURE built on the false *"drops in silence"* mechanism (`instance_test.go:132-145`, the witness producer-rejected) plus three P2s that are the same mechanism in a new comment, a new godoc and rewritten file prose (`instance_test.go:132-136`, `schema_closure_test.go:56-61`, `schema.go:300-301`); #1182's reject was a false *"folds into nothing"* one written into the very sentence #786's repair had just corrected. **The *"a repair round quotes the correct mechanism at the writer, and the discipline failed there anyway"* premise this row carried is WITHDRAWN** by §3 of the accepted ruling (`issuecomment-5529902800`): #1182's false mechanisms were written in its round-1 commit `40fef85`, before any verdict existed, and its repair `8027a1c` was comment-only and accepted clean. What was in front of that writer was the corrected text itself — `74ea322` had just put the `<override>` shapes into `conformance/schema.go:300-304` and `40fef85` overwrote that sentence — which reads as mild evidence FOR a standing obligation, not against one. The ruling keeps the no-new-rule disposition anyway, on #1181's clean round-1 accept (`7df8d7f`), on binding-versus-naming, and on a named reopening condition. #642 may overturn it on its own evidence |
| 2 | #1206 | **The one lane-mover in the queue with its grounding already done.** `src-element` clause 2.2's prose-only eight attributes (`name`/`type`/`default`/`fixed`/`nillable`/`block`/`form`/`targetNamespace`) on a local `<element ref=>` are uncharged, and unlike #471's grammar pair they carry **real suite fixtures** — `ElemDecl/name00401m{3,4,5}`, `name00501m{2,4,6,8,10}`, `elemR003`/`elemR006` and more, currently under-rejected `fail`. #471's own grounding located them and carved this issue from its scope; this is a genuine `schema` tightening. **Not ratchet-neutral — measure case by case, predict a figure before running, per CLAUDE.md.** The `name` fork (clause 2.1 also bites) needs the grounding to rule which clause is reported, exactly as #471 ruled 2.2 over `e-props-correct` clause 3 |
| 3 | #1188 | **The first CLI BEHAVIOUR defect this queue carries, from the fourteenth consultation.** `parse`'s summary reports no namespace for a schema that has one and `(absent)` for a schema that has none, and nothing in the three contract copies says the list is component-derived. **§3.17.1 gives the Schema component no `{target namespace}` property**, so the reporting branch is an `xsd` surface addition and a warden question, and the documenting branch is one sentence in three places — a genuine fork, banded above the doc rows. The CLI ceiling lifted with #472 exactly as predicted, and this is startable now |
| 4 | #720 | **`goxsd8 validate` — unblocked when #472 landed, the second increment of the CLI ceiling #16 has carried since 2026-07-07, and the next persona-consultation target.** Its `## Depends on` reads `none` now that #472 answered the reserved-but-unbuilt exit-code question. It moves no conformance lane by construction (`go list -deps ./conformance \| grep -c cmd/goxsd8` is `0`), which is why it sits below the behaviour and lane rows — but it is what the next full libuser/cliuser pass must be pointed at |
| 5 | #999 | **The `[tests that cannot fail]` pattern, still its only filed instance and now two consecutive code landings deep.** #786 repaired a guard fixture with a witness carrying the identical defect; #472 shipped two tests its own mason account called load-bearing that could not fail. Both were found by the arbiter running a mutation mason had described and not executed; both cost a reject round. #472's post-land routed the *pattern* to `/retro` and rightly declined a fifty-seventh `kind/process` issue, leaving this the one queue row that discharges any of it. Re-scoped last stamp — #472 falsified its `## Spec` quote and the defect survived the re-cut *wider* |
| 6 | #1136 | **A wrong-order defect in the arm this family keeps widening, and the pressure on it is rising.** #471 made `elementParticleTerm`'s ref= arm run three checks where its evidence table described two, and **#1205 and #1206 queue two more charges into the same arm** — so whether `checkS4SChildOrder` runs before or after the clause-2.2 charge decides which fault class a doubly-violating schema reports, on an arm about to carry five. #471's post-land already corrected this body; #1099's mutation proof was made against a two-check arm and must be re-run, not quoted. Take it in either order with #1135, never together |
| 7 | #1203 | **A latent ratchet ambush #404's landing created, filed by its post-land.** Two of #404's eleven banked `schema` passes — `addB014` (an S4S copy colliding `anyType` with our seeded builtins) and `schZ006` (one document composed twice) — reject for a fault the suite does not intend, so when #603 or #703 repairs the over-rejection those passes flip down as a spurious `Regressed` the repairing session did not cause. A banked pass resting on an over-rejection is invisible in the lane file; this measures it into a known figure. No production change — a measurement and a ruling, filed or dismissed here, not attempted. Cross-references #1002 for the wrong-reason-pass class |
| 8 | #1167 | **Two measured sightings, #414 and #1115.** `gapaudit` reconciles marker → issue and issue → marker; nothing audits prose that points *at* a marker or its file, and the stale-marker sweep has now missed an inbound site in the deleting package twice. gapaudit's own group-1 candidate-owns it against `validate/assess.go:853`. PRINCIPLES 27 says a repeated grep wants a tool |
| 9 | #1156 | **Comment text only, and this pass corrected the one figure that had gone stale.** Two group-1 rows are the survey misreading the tree: `contentrestricts.go:742` names open #499 four paragraphs below the marker head where `paragraph()` never reaches, and `:1047` names nobody while #345 owns it. Its census absolute was restamped to **69** this pass; the row deltas are unchanged, and row 1 has a landed precedent in #815's repair `627ed25` |
| 10 | #1007 with #1123 and #1189 | **Three issues, one session, three contract copies, one coupling test.** All three edit `cmd/goxsd8/doc.go`, `main.go`'s `usage` const and `README.md` together, and `TestUsageCoversContract` pins the first two by sixteen substrings. #1007 is a clean one-of-three (`gen` alone names no exit code); #1123's claim is self-refuting to a reader who follows it; #1189 is a false sentence in the section both others touch. Taking them apart means rebasing the same three files three times |
| 11 | #1036 | **The silence #1029's landing exposed, carried a fifth-plus stamp, and STYLE P3 does not permit it to stay.** A top-level `<xs:schema>` child outside the XSD namespace is neither charged as the §5.1 first-bullet grammar fault it is — §A's `<xs:schema>` content model has no wildcard arm, and `xs:openAttrs` admits foreign *attributes*, not foreign element children — nor reported through `parser.AssembledDocument.Unmapped`. **One settled disposition, either direction.** #1047's body names this issue as the owner of the foreign-namespace skip |
| 12 | #1205 | **The grammar-prohibited pair `final=`/`abstract=` on a local element, ratchet-neutral by construction like #471.** Neither the ref= path nor the inline path charges either attribute; `xs:localElement` prohibits both `use="prohibited"`, unconditionally. Cheap and correct and moves no lane, which is why it foots the band — but the **#768 collision is named on both sides** (its fixture stops reaching `produceLocalElement` the moment this lands) and is a hazard, not a blocker. Neither is labelled `blocked` |

**Below the band, and why**: **#1181 is IN FLIGHT**, the sole exclusion on that
ground. **#1196** measures the live residual for #1051 and its own body forbids
banding on the numbers it produces — take it opportunistically, never together
with #1171 (a comment fix that needs no suite run). **#1201** is correct and
needs a warden pre-flight for its 1a fix, but **zero suite cases exercise it**,
so it moves nothing; take it opportunistically with #1203 or whoever next touches
`fabricatedRejection`, and it retires the two new `GAP(conformance)` markers when
it lands. **#1183** is cheap, correct, moves nothing and makes #1051's stage-2
deletion three predicates smaller — take it with #1181's family. **#1140** is
carried a **fifth** stamp after three *"take it now, while the rebase is free"*
rankings; the last stamp's instruction stands — **if this stamp still finds it
unbanded and untaken, the next should close it as accepted rather than carry it a
sixth time.** **#1135** pairs with #1136 (never together). **#849** carries a
`## Cost of delay` reading *"a steward re-ranking is warranted"* — an unranked
issue asking for a rank, routed to `/retro`'s steward drift review as a third
item beside #841 and #1080. **#1051** stays `blocked` on three unfiled buckets.
The persona-family and process tails are ranked on ordinary grounds; **#1033** is
still the only row no persona has ever looked at.

**There is no `Increasing` steward ranking anywhere in the band.** #849 is the
nearest, and its own body records the ranking as falsified and unreplaced.

### Next planning action

1. ~~**Take #1199, on the compounding-tax rule, and let #1181 decide it.**~~
   **DISCHARGED — do not take it.** The trigger set here resolved: #1181 landed
   clean (`7df8d7f`), and #1199 closed on the no-new-rule ruling
   (`issuecomment-5529902800`, squash `b57d37d`). Band row 1 carries the
   corrected record. **Start at #1206.**
2. **#1206 is the lane slice with its grounding already done — take it and hold
   the ratchet to account.** **Trigger set here**: #1206 predicts genuine
   `schema` movement (its fixtures are under-rejected `fail` today). Predict a
   figure before running and account for it case by case, per CLAUDE.md; the
   `name` attribute is the one where clause 2.1 also bites, and the grounding
   must rule which clause is reported rather than letting statement order decide.
3. **#1164's ruling is landed — the near-match is a coincidence — and #1196 is
   the instrument that replaces every stale per-class count.** **Standing
   instruction unchanged**: do not band on the numbers #1196 produces, and take
   the residual total from the instrument at HEAD, never by subtracting a landed
   bucket from 670. #1051's stage-2 deletion waits on the three unfiled buckets
   (the named `<group>` body, `groupParticles`' default arm, the `<redefine>`
   child), which #1051's standing instruction says to file as they are taken.
4. **The `src-element` clause 4 follow-up is DISMISSED, with the reason.** #471's
   post-land deferred to this pass whether to file a tracker for clause 4
   (`ed-with-ns`), which governs `targetNamespace` on a local declaration and is
   **charged nowhere in `parser`**. Dismissed: the ref= overlap has **zero suite
   fixtures**, the rule's other clauses are unimplemented, and #1206's Notes
   already record it where the session that meets the `targetNamespace` bullet
   reads it. Filing an ungrounded spec claim with no case to exercise it is the
   speculative widening #471's own Notes and #1051's standing instruction both
   forbid. It becomes fileable the moment a fixture surfaces; until then it is a
   recorded scope boundary, not a queue row.
5. **The human decision blocking #1002 is unchanged and is now carried for a
   TENTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — and (b) depends on **#1042**, filed and `blocked`. **No
   agent should attempt either.** #1203's measurement may surface a fresh
   instance of the wrong-reason-pass class (a) is meant to cover; that is a datum
   for the ruling, not a licence to act on it.
6. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3). **M6 tier 2 itself is uncarved** — `$value`
   binding, an F&O function library, typed comparison — and is too big for one
   issue. That carve is a `/backlog` act at the M6 opening, and #1042 is the thing
   it must slice around rather than a blank page.
7. **The unblock sweep measured a clean zero for the SIXTH consecutive stamp.**
   All 19 open `blocked` bodies were read and their `## Depends on` sections
   checked against the six closures (#1160, #1174, #1164, #1182, #404, #471); only
   #1051 mentions any of them, and only as struck-through discharged provenance,
   not a live dependency. Each landing's own post-land pass had already measured
   zero. Seven of the 19 are **triggers rather than issues** (#79, #555, #692,
   #841, #925, #1002, #1080) and say so in their own `## Depends on`; **do not
   re-scan those on the next sweep.**

**Standing, and re-checked rather than restated.** Four unlanded corrections still
target one paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**,
**#646**, **#679**, **#912** — and whichever lands last rebases three times.
**The next `/retro` inherits seven**: #692, #925, #841, #1080, the
fold-the-five-species question (#635, #912, #609, #510, #646), the
`[tests that cannot fail]` pattern (routed by #472's post-land, now band row 5's
#999), and **#849's owed steward re-ranking**, the third item for the steward's
drift review. **#841 is still the counter-example the steward-ranking rule cannot
reach**: a `kind/refactor` with a steward ranking, `blocked` because its trigger
has no mailbox, fired twice without a ruling. The `gh --paginate` page-2 403 trap
#471's post-land recorded is **already named** in docs/ROUTINES.md's Survey-input
recipe comment (*"gh api --paginate is a trap ... HTTP 403"*), so it is a `/retro`
datum and not a filing. The CTA cohort's 45 banked `instance` failures remain
unattributed. `gate.yml` runs and is still not a required status check, which only
the repository owner can change.

**Environment, one witness each.** Repository-scoped `gh api` REST served every
read and **TWO writes** here — **0** issues filed (every follow-up this window was
already filed by its landing's own pass) and **1** body edited (#1156's census
absolute), plus **1** working-tree edit to `docs/PLAN.md`'s M4 producer paragraph
(the #471 stale token). The paginate recipe ran to **13 pages** — 12 full at 100,
page 13 short at **8** and re-requested per the recipe with no larger answer, the
genuine-last-page case the retry arm falls through — and the post-loop fullness
check passed. The shallow clone truncated `origin/main` to 50 commits (**#802**),
which is why `claude/eloquent-cerf-8jq9o6`'s ancestry read came from a commit
date rather than `git log`. No conformance measurement was taken by this pass: the
lane table above is the committed expectations, which `docs/WORKFLOW.md` names as
the lane score (#1120).

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

The original carve (#167–#183) is landed. **Every §3.4.2 complex-type
representation form is now produced**: `<simpleContent>` with
`<restriction>` was the last one declined outright and #909 built it on
2026-08-22. Open work is the long tail of producer widening, finalize
validity and composition edge cases, and it has changed shape — what
remains is predominantly **s4s grammar faults the producer decides and
ACCEPTS** rather than forms it cannot build. The two exemplars this
paragraph used to name, #956 (child order and `maxOccurs` across the
derivation alternants) and #958 (a facet element name the gate admits and
the producer drops), both landed on 2026-08-22/23; **#884** (malformed named
`<group>` bodies collapsing into one `mgd-props-correct` message) joined them at
`b3f295a`, and **#972** — an XSD-namespace child §4.1.2's
`<simpleType><restriction>` has no position for, dropped by `restrictionFacets`
so the producer built a schema the harness called `valid` — landed at `e491ddb`
for **`schema` +26** — the family's largest single move until **#1047** took
`schema` +34 on 2026-08-28. **#471** landed the local `<element ref=>` carrying
`substitutionGroup=` on 2026-09-03 (`31263ed`, `Ratchet: unchanged` — the
grammar-level `use="prohibited"` shape rejects with no case in the corpus to
bank) and its grounding carved two siblings from its scope: **#1205**
(`final=`/`abstract=`, the other grammar-prohibited pair, ratchet-neutral by the
same construction) and **#1206** (`src-element` clause 2.2's prose-only eight
attributes, which unlike the grammar pair carry real suite fixtures and can move
the lane). Live producer-decides-and-accepts members are **#931** (occurrence
attributes on a named `<group>`'s child compositor), **#929** and **#455**. A
second, narrower family opened beside it — the rejections the
producer already makes **correctly but describes badly**, whose bar
`xsderr/doc.go` set with #966 — and it is **discharged**: #975 landed at
`1dcffbf`, so every s4s-grammar rejection now names its Appendix A production.

**A THIRD family is the one that has been paying, and it has FIVE landed
members.** #1030's unmapped-construct census turned the "decides and ACCEPTS"
family from a shape into a list, and the list delivered `schema` **+34**
(#1047), **+23** (#1046, plus `instance` +15), **+21** (#1076), **+16** (#1048)
and **+2** (#1099). For #1046/#1047/#1048 the gate-side alternative was
**measured and ruled out** — widening `conformance/schema.go`'s shape gate costs
a banked ratchet win — and #1076 and #1099 arrived with no measured count at
all, banded on the criterion *"the shape is in the suite's invalid corpus"*
instead. **That criterion is five-for-five on DIRECTION and predicts nothing
about MAGNITUDE**; the +21 and the +2 came from the same family four days
apart. The Status section carries the consequence for how the band is ordered.

**The family replenishes itself as it is worked, which is what to expect and
not tail growth** — every issue it has added was filed by the post-land pass of
a landing that moved the lane. **Its shape is also changing**: #1047, #1076 and
#1099 all widened `checkS4SChildOrder` or its callers, and the issues arriving
now — #1097 (landed `32070b8`), and open #1098, #1133, #1135 and #1136 — are
defects *in* that widening rather than further sites for it. **A producer that decides
more is a producer with more to get wrong, and the census does not name that
class.**

**Read the milestone count as a floor.** The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it — #1135 and #1136 are M4 work carrying
no milestone today. **The sharpest instance is #1126**, which moved `schema`
+475 and `instance` +113 — the largest lane movement this project has recorded —
and carried no milestone either, so the count did not move at all.

### M5 — Instance validation (XML) — [epic #250](https://github.com/kud360/goxsd8/issues/250)

`validate` engine plus `validate/xmlsrc`: greedy deterministic matching,
identity constraints, `xsi:type`/`xsi:nil`, wildcards, default and fixed
values. **`instance` lane.**

Carved 2026-08-12 into ten slices, #710–#719, and now **seventeen** — #766 was
split out of #714 by a warden pre-flight, #773, #774 and #775 out of #766 by
another, #782 and #783 out of #715 by its own, and **#790** filed from #715's
warden verdict, which had named the seam and been read by nobody who filed it.
Thirteen have landed — **#719** joined them on 2026-08-27 at `bd887cd`, wiring
`cvc-assertion` fail-open at every variety level and shipping the
`validate.Unevaluated` channel; it moved no lane and correctly so, because a
fail-open declines exactly where the engine already declined. It unblocked #56
and its successor **#1042** (assertion EVALUATION) is the first issue this
project has ever placed on **M6**. The other twelve: the infoset
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
+9, unioned onto #716's). #913's cvc-type clause 3.1 landing added **9409**,
itself M5 and the largest single lane move this project has recorded.

**Landings OUTSIDE this milestone keep moving this lane, by four distinct
mechanisms, and a running total of them is not maintained here** — take the
figure from the Status section's table (#646). The four:

- **A slice PRODUCES a component the engine could not previously see** — #733
  (a top-level `<xs:attribute>`'s inline `<xs:simpleType>`), #909
  (`<simpleContent>` `<restriction>`), the CTA pair #842/#851.
- **A slice DECIDES a `{type table}` the engine previously withheld.**
- **The MEASUREMENT is fixed and the engine never changed** — #862, where
  `resolveExpected` was picking the wrong feature-scoped expected verdict, so
  the answers had been right all along. A lane can move because the measurement
  was wrong.
- **The SCHEMA the engine was handed is fixed** — #1001, where §4.2.2's
  ·conditional inclusion· removed declarations colliding under
  `sch-props-correct` clause 2, so five documents the engine had never been
  given a usable schema for became decidable.

**The largest of them is #853 (+141 at `2310710`) and it carries no milestone**,
which is a finding about the LABEL and not about M5's carve: it decides a `cvc-`
rule at assessment time on an XML instance, which is M5's own definition of its
scope, and it sat outside only because a `kind/gap` filed by a post-land pass
never acquired one. **Read the M5 milestone count as a floor and never as the
lane's remaining work** — the same caveat M4's section makes, for the same
mechanical reason, and the `instance` lane is where it costs most.

**#853 repeats #913's lesson rather than #790's.** It was banded UNMEASURED,
with an instruction to go count first, and outperformed every candidate that
arrived with a count: a slice that decides a *new* rule on a commonly-declined
shape moves the number far more than its rule count suggests. **The Status
section carries what three stamps of this have settled** — a document count is
a filter on candidates, never a sort key, and the invalid-corpus criterion
predicts direction and not magnitude.

**A flat M5 landing is routinely CORRECT and the section should not read as if
it were not.** #1043 (`5d3d222`) declined the ·governing type definition· of a
skip-wildcard attribute, withdrawing a `cvc-id` charge §3.10.4.1's Note says was
never owed; #1116 (`9919faf`) charged cvc-complex-type clause 2 against an
anonymous governing type behind a harness gate that admits no case of the shape.
Withdrawing a wrong charge on a document already banked `fail`, or deciding a
shape the executor withholds, cannot register as movement. **That is what the
ratchet's zero-flip-down means, and it is a result to explain rather than a
null one** — and #1116's explanation was banked and then paid out. **#1126
(`d433f7f`) deleted the harness gate #1116's `Ratchet:` trailer named**, and both
lanes moved on the same run: `schema` +475, `instance` +113. The charge was
right when it measured flat; the executor was what withheld it.

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

**The lane score is a floor built for soundness, and no jump has ever changed
what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the failures
that remain are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read the lane score as a pass rate.** It is the count of documents this
engine can honestly call not-valid. **Read the current figure from the Status
section's table and never from this paragraph** — an absolute figure written
here is stale before the next landing (#646). It grew most because #913
decided `cvc-type` clause 3.1 — the commonest simple-typed-leaf shape the lane
had declined outright — which is the counterpart to #790's lesson, not a
contradiction of it: a slice that decides a *new* rule moves the number far
MORE than its rule count suggests when the declined shape is common, and #913
moved it more than #790's descent did.

**All TWELVE decided-and-wrong `instance` cases carry an owner, and the two
classes below account for nine of them.** Measured at `c720206`: 15332 fail =
15315 decline candidates + 5 declined indeterminate + **12** decided-and-wrong.
The other three are owned outside these paragraphs (#1160) —
`MS-DataTypes2006-07-15/gMonth002_2061/instance/gMonth002_2061.v` and
`gMonth004_2063.v` by **#921**, and `Open/open013/instance/open013.v1.xml` by
**#456**.

**TWO classes are decided and decided WRONG, and the first is two cases, both
#771**: `ElemDecl/targetns00101m/instance/targetNS00101m1_p`, whose root
element `{ElemDecl/targetNSa}number` is declared in `targetNS00101m1a.xsd`,
and `SType/st_targetns00101m/instance/ST_targetNS00101m2_p`, whose root
`{ST_targetNSa}test` is declared by neither of its group's schema documents
and whose rescuing `xsi:type` resolves to `{ST_targetNSa}Test` in
`ST_targetNS00101ma.xsd` (attributed by #1160). **One root cause, two routes
to the same charge**: each turns on a schema document the harness never
assembles, because only the instance's own `xsi:schemaLocation` points at it,
so the root-name lookup fails in the first case and the `xsi:type` lookup
fails in the second — and since `validate/cvcelt.go`'s
`instanceTypeDefinition` treats an absent, non-QName and unresolvable
`xsi:type` alike, both land on `cvc-assess-elt`. The first case was one of
four, and #800 retired two of them:
`Assert/assert_019/instance/assert_019_2` and
`CTA/typeAlternatives_001/instance/typeAlternatives_001_2` now decline honestly
instead of rejecting a document the ·conditionally selected· type admits.
**That is two, not three, and `CTA/cta0008.v01` was never among them** — it
takes §3.12.2's inline arm, which #800 deferred to #822 and which **#851 landed
on 2026-08-17** (#822 closed, superseded), and the count was
measured by diffing `GOXSD_DECLINES=1` across the two trees rather than
predicted. All were already banked `fail` before #790 and are not its
regression; the descent is what made them visible, and the trade of a wrong
decision for an honest decline is one the lane cannot register as movement.

**#913 added the second class.** Seven CTA documents are false-charged through `cvc-type` clause 3.1 until
§3.12.4's `{inherited attributes}` merge lands (#831, #871) — an
honest-decline-to-wrong-decision trade the ratchet's zero-flip-down cannot
register, escalated on #831's thread.

The decline census that separated harvest candidates from indeterminates
predates every M5 landing from #766 onward and every outside-M5 mover — #909,
#853 and #1116 included — and is not re-derived here. **It is now, by a wide margin, the oldest measurement this milestone still
argues from**, and #786 is the nearest issue to it.

The design constraints are fixed by `validate/doc.go` and PRINCIPLES 8, 11,
13, 14 and 15, and the carve does not reopen them.

### M6 — XPath required subset

CTA restricted subset plus assertion essentials, fail-open with `GAP`
markers, IDC selector and field paths. Dynamic-error direction per
PRINCIPLES 20. Not filed as an epic — speculative epics two milestones out
earn nothing.

**Part of this milestone has already shipped, ahead of it and outside it.**
**#842** landed the §3.12.6 required-subset CTA evaluator on 2026-08-16 at
`3509de6` — pulled forward because M5's `instance` lane cannot decide a
conditionally-typed element without it — and `xpath` stopped being a
`doc.go`-only package. **#861** then corrected its clause-2 error handling and
**#851** gave §3.12.2's inline arm a real anonymous component. Three more
landed on 2026-08-18: **#858** (`d02aa59`, the three cast-shaped constructs),
**#886** (`3867fb5`, `ta-props-correct` clause 2 charged at schema construction
time) and **#887** (`f1250c0`, comparator legality decided from `xpath20.md`
Appendix B.2's generated rows). **All three moved no lane, and all three said
so** — the required subset is now largely built, and building it has not been
what converts cases.

What remains of the CTA subset is **#888** (a cast target that is `xs:QName`),
**#889** (value-level numeric widening, §B.1 rule 1.1's float→double promotion,
which #858 withheld rather than faked) and **#894** (err:XPST0051/XPST0080, the
static remainder #886 did not charge). **#859**, the wildcard `ta-AttrName`
arms, **landed 2026-08-18 at `ea0650a`** — this paragraph carried it as
remaining work for fifteen days, arguing about a stale lease on a branch whose
issue had already closed the same day. **#871** is the §3.12.4 clause 1.1.3
·inherited attributes· merge, blocked on M4's #831. None of the five carries a
milestone, which is the same pattern M4's tail records.

**Assertion evaluation is FILED — #1042, `blocked`, and the first issue this
project has ever placed on this milestone.** It owns `cvc-assertion` (§3.13.4.1)
and `cvc-assertions-valid` (§4.3.13.3) and retires the `GAP(validate)` markers
**#719** landed on 2026-08-27 at `bd887cd`, one milestone early, because the
`instance` lane must decline every case whose outcome turns on an assertion. Its
`## Depends on` is a trigger rather than an issue — an XPath 2.0 evaluator able
to run an assertion `{test}` — so nothing in the queue can start it.

**What is still uncarved is tier 2 itself**: `$value` binding, an F&O function
library and typed comparison, which is too big for one issue. Carving it is a
`/backlog` act and #1042 is now the thing that carve slices, rather than a blank.

**#56 LANDED on 2026-08-28 at `3160813`**, ten days after #719 unblocked it and
`Ratchet: unchanged` as its body predicted. It records the CTA compile-time
withhold into `Result.Unevaluated` under `key-cta-ta-select` (§3.12.4), with no
second type and no `Evaluated bool` — the encoding #719 shipped
(`Rule()`/`Loc()`/`Msg()`, `Result.Unevaluated()` in document order), reused
rather than paralleled (STYLE D4), exactly as #842's warden pre-flight had ruled
on D3. **Surface: none new.** STYLE 9's fail-open discipline is only honest if a
fail-open answer is distinguishable from a real pass, and both of this
milestone's fail-open channels — the assertion sites #719 collects and the CTA
withhold #56 records — now reach the same slice under their own rule IDs.

**Its first consequence is a documentation defect, not a code one, and it is
filed.** `validate/doc.go:101-104` states that an empty `Violations()` beside a
non-empty `Unevaluated()` is not a pass; `README.md:217-219` — the module's only
working validation snippet, since `validate` ships no runnable `Example` (#1088)
— still names `res.Err()` as the sole incompleteness signal and never mentions
`Unevaluated` at all. #56's landing is what made that reachable on an ordinary
conditionally-typed document rather than only on one carrying an assertion.
**#1122** owns it.

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
