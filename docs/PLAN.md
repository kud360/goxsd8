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

## Status — 2026-09-04 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **SIX** landings — the last band's rows **1 through 5 taken in order** (#1199, #1206, #1188, #720, #999), plus **#1181**, whose merge the last stamp's own lane read missed. It measured **`schema` +32 across two of the six** (#1181 +16, #1206 +16), watched **#1206 beat its own prediction by one** on a UTF-16LE fixture no UTF-8 grep can see — the same blind spot that then produced **#1215** and **#1216**, this queue's two grounded lane slices — found the branch namespace **completely idle for the first time in this stamp's memory** (zero LIVE, zero CLAIMED), saw the marker census **flat at 69 across all six landings**, and ran the **FIFTEENTH persona consultation** against a materially changed CLI surface, filing **eight** issues from it (#1229–#1236) and folding two findings into #1122 and #755. Two open bodies were corrected where this window falsified them — **#1007** (its `validate` row, its `gen` line numbers and its sixteen-pin count) and **#1122** (its README citations). **#1140 is BANDED rather than closed**, discharging the last stamp's standing instruction in the other available direction)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13928 | 1470 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### Two of six landings moved `schema`, and the four that did not are all `cmd` or process

**`schema` 13896 → 13928 (+32); `instance` flat at 11029; `datatypes` flat.**

| landing | commit | lane movement | milestone |
|---|---|---|---|
| **#1181** | `7df8d7f` | **`schema` +16** | none |
| **#1199** | `b57d37d` | unchanged | none |
| **#1206** | `1780670` | **`schema` +16** | **M4** |
| **#1188** | `4841fa9` | unchanged | none |
| **#720** | `f4d8f76` | unchanged | **M5** |
| **#999** | `1d2edfb` | unchanged | none |

**#1181 is in this table and not the last one, and the reason is a measurement
order rather than a late merge.** It landed at `7df8d7f`, *before* the
2026-09-03 stamp commit `9c12a89`, but that stamp's lane table read **13896** —
the pre-#1181 figure — so its +16 has never been attributed until now. The
stamp was right to call it in-flight when it wrote its namespace section; the
lane paste beside it was simply taken earlier. **A stamp's lane table dates
from when `lanestatus` ran, not from when the section was committed**, and this
is the first time the two have visibly disagreed.

**#1206 banked SIXTEEN against a predicted FIFTEEN, and the extra one is the
finding that fed the next two band rows.** `targetNamespace`'s only fixture in
the whole W3C suite lives in `ibmData/`, which is **UTF-16LE**, so every UTF-8
grep over `testdata/xsdtests` had been blind to it. The arbiter's independent
UTF-16-aware census found the sixteenth case and every case was accounted for
before banking (`e70c1ae`). The post-land scan of the decoded
`ibmData/schema_invalid/S3_2_3/` then found **six further `fail` cases** in the
same blind spot — three element-side and three attribute-side — which are now
**#1215** and **#1216**, band rows 1 and 2. **The blind spot, not the clause,
is what has been producing this queue's lane slices**, and a session grounding
anything against the suite decodes `ibmData/` before concluding a fixture does
not exist.

**Four of the six landings moved no lane and three of those four are
`area/cmd`.** #1188, #720 and #999 are all ratchet-neutral **by construction**
and each measured it rather than asserting it: `go list -deps ./conformance |
grep -c cmd/goxsd8` is `0`, so no lane can see that package at all. This is the
count-versus-lane divergence earlier stamps kept restating, in its cleanest
form yet — the two lane movers carried one milestone between them, and the CLI
milestone (M5) moved on a landing no lane could observe.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and
this pass's 698-issue post-write fetch:

```
ISSUE  BRANCH         LEASE AGE  VERDICT  REASON
732    wip/issue-732  285h14m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  463h12m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846  235h33m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872  429h13m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  357h26m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  296h31m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993  240h54m0s  RETIRED  wip/issue-993: issue #993 is closed
```

**ZERO LIVE and ZERO CLAIMED — the namespace is completely idle**, the first
stamp in this section's memory where nothing is in flight. Every band row below
is startable without colliding with a claim, and `ready` overstates startable
work by **nothing** this time, where the last two stamps each had to subtract
one. **`wip/issue-1181` is GONE**, deleted on merge, so its lease closed cleanly
and that is the **third consecutive** stamp whose only `wip/` churn was a clean
open-and-close.

**All seven RETIRED refs closed `not_planned`** — re-checked one by one over
REST, not carried from the last stamp — so they are parks and supersedes whose
content is *supposed* not to be in `main`, and none owes a supersede. Cloud
containers cannot delete remote refs, so these accumulate by design and are not
a finding. **Zero `parked/*`.**

**FOUR non-`wip` `claude/*` refs stand, and one of them is this stamp's
sharpest witness of the shallow-clone premise:**

| ref | tip | dated | reads |
|---|---|---|---|
| `claude/eloquent-cerf-39rk64` | `0abeab6` | 2026-08-29 | **NOT-ancestor — and it read ANCESTOR at the last stamp** |
| `claude/eloquent-cerf-3xu0ki` | `62d5143` | 2026-09-03 | ancestor, carries nothing (#404's squash) |
| `claude/eloquent-cerf-8jq9o6` | `7841e98` | 2026-08-29 | NOT-ancestor, unchanged from the last stamp |
| `claude/eloquent-cerf-adewly` (**NEW**) | `5d03049` | 2026-09-04 | ancestor, carries nothing (#1188's post-land squash) |

**`39rk64`'s tip has not moved and its verdict flipped anyway.** The last stamp
read it an ancestor carrying nothing; this container sees **51 commits** of
`origin/main` and `0abeab6` (2026-08-29) now sits beyond that horizon, so
`git merge-base` returns empty and `--is-ancestor` reports false. That is a
measurement artefact of the clone depth and **nothing about the ref changed**.
It is the cleanest demonstration yet of **#802**, which owns this and is open:
an ancestry read taken here is a function of when you take it, not of the
history. Both NOT-ancestor rows were dispositioned from GitHub and commit dates
rather than from `git log`. Listed for human triage, not acted on.

### Marker census

`go tool gapaudit` over this pass's whole 698-issue post-write feed: **69
markers across 8 areas** — `xsd` 33, `validate` 17, `xpath` 6, `xml` 4,
`parser` 3, `value` 3, `conformance` 2, `cmd` 1.

**The census is FLAT across all six landings — 69, and the same eight areas,
marker for marker.** The last two stamps each opened a new area (`cmd`, then
`conformance`); this window opened none, retired none and moved none. Three of
the six landings touched no `.go` file outside `cmd/goxsd8`, which carries one
marker, and #1206's parser change retired nothing because the clause it charged
was never marked.

**Group 1 held at 17 and group 2 moved 25 → 27.** Group 1 is unchanged row for
row. Group 2's move is fully accounted for: **#1215**, **#1216** and **#1223**
were filed as `kind/gap` this window, and **#1206** left it by closing. Group 2
is *"OPEN `kind/gap` issues no marker cites"*, and `kind/gap` also labels
conformance-lane gaps that never carry a marker — all three new members are
that kind, so none is a stale tracker.

**ZERO group-1 rows carry no annotation at all, so the tool's own filing rule
selects nothing and there are zero untracked GAP sites — SIXTH consecutive
stamp.** Every group-1 row prints at least one candidate owner or resemblance.
**Two of the seventeen are instrument defects rather than ownership defects, and
#1156 owns both** — unchanged, and its census absolute at **69** (restamped by
the last stamp) is still correct at this stamp, which is worth recording because
it is the first figure in that body that a whole window has not falsified.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 13 pages: **698 issues, 262 open, 436 closed**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **50** | **109** | active |
| **M5 — Instance validation (XML)** | **14** | **20** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **241 `ready`, 21 `blocked`, 0 `needs-replan`** — every
open issue carries `ready` or `blocked`, verified mechanically, and the two sum
to 262 with no gap. By kind: `kind/refactor` 72, `kind/process` 55, `kind/gap`
54, `kind/tooling` 32, `kind/bug` 26, `kind/story` 23, `kind/docs` 12,
`kind/feature` 4, `epic` 2. By area: `parser` 75, `meta` 69, `xsd` 59,
`conformance` 30, `docs` 26, `validate` 22, `cmd` 16, `value` 14, `builtin` 10,
`xpath` 6, `xsderr` 3, `loader` 2, `regex` 2.

**M4 moved 49 → 50 open and 108 → 109 closed**: #1206 closed and #1215/#1216
were filed onto it, a net of +1 open and +1 closed. **M5 moved 11 → 14 open and
19 → 20 closed**: #720 closed and four issues were filed onto it — #1223 and
#1224 by that landing's own post-land pass, #1229 and #1230 by this one. Neither
milestone's count is its lane's remaining work, which the two milestone sections
below say in their own words and which this window demonstrates twice: the M5
mover moved no lane at all, and the M4 mover was one of only two that did.

**`ready` 241 is the honest startable count, with nothing to subtract**, because
the namespace is idle. That is a change from the last two stamps and is the
reason there is no exclusion-on-flight row in the band.

### Persona consultations — the FIFTEENTH ran, and it was owed

The cartographer role-plays no persona and does not spawn one (#416): it has
read the source, so a verdict it produced would launder an insider's opinion as
an outsider's. **The orchestrating session ran both personas against the current
published surface and handed this pass their reports; what follows is the
folding, not the consultation.**

**It was owed and the last stamp said so.** The fourteenth consultation was
declined on a measured byte-identical surface, with the standing note that *"the
next full consultation is still owed against #720's landing … which is the next
thing that changes what a cliuser can observe"*. #720 landed (`f4d8f76`) and
#1188 (`4841fa9`) rewrote the `parse` contract in all three copies, so the
trigger fired exactly as written.

**Ten findings, eight filed, two folded, zero discarded.**

- **cliuser, five findings, all novel, all filed** — #1229, #1230, #1231,
  #1232, #1233. Every one is against `goxsd8 validate`, the surface #720 shipped
  four hours before the consultation ran. Its baseline section is worth as much
  as its findings: the exit-code split, the multi-`-schema` wrapper composition,
  `sch-props-correct` clause 2 across two arguments, per-instance aggregation,
  the `-format` vocabulary and its case-sensitivity, the reserved-format exit-2
  messages, `-no-hints`, relative-`schemaLocation` resolution against the
  instance's directory, `-q` not silencing violations, and README's own
  wrapped-cause rendering example — **all reproduced exactly**, so the next
  consultation does not re-check them.
- **libuser, five findings, three filed and two folded** — #1234, #1235, #1236
  filed; the `Violations()`-versus-`Unevaluated()` finding is **#1122**, already
  open and now restamped with its correct line numbers and a note that a second
  persona reached it independently; the `xsi:schemaLocation`-hint-reader finding
  is **#755**, whose body and thread now carry the **third promise site** the
  persona surfaced (`loader/doc.go:36-39`, present-tense, and **not false
  today** — recorded as a site a taker must read, not as a correction to make).
  Both snippets in README's Library section **compile against the documented
  signatures exactly**, argument order and types verbatim — the persona checked
  and said so.

**The sharpest finding is #1229 and it changed shape under verification.** The
cliuser reported a behaviour-versus-contract gap: a hint naming a nonexistent
document is dropped with an empty stderr where README promises a diagnosis.
Checking it against the tree found it is **also contract-versus-contract** —
`cmd/goxsd8/doc.go:129-136` names two reported cases and calls the third a
silent degradation, and `doc.go` is the copy README itself calls authoritative.
So README is false however the behaviour question is ruled, which settles the
cheap half of the fork before a session starts.

**Two findings the personas would not have reached and this pass did not
invent.** Neither report is treated as more than testimony: every citation in
the eight filed bodies was re-checked against the tree at `7a49ca9` before
filing, and two findings changed materially in the process (#1229 above, and
#1230, whose claim turned out to be written into the very `const` doc comment
that would have to enforce it, `cmd/goxsd8/validate.go:28-32`).

**#1033 remains the one row no persona has ever looked at.** The next
consultation is owed against whatever next changes the observable surface;
nothing in the band below does, so it may be a while.

### Working band

**Re-derived from this pass's evidence.** The last band's rows **1 through 5 all
landed, in order** (#1199, #1206, #1188, #720, #999) — the longest run of
in-order band consumption this section has recorded. Nothing is in flight, so
take from the top; re-run `wipsurvey` first anyway.

| # | issue | why here |
|---|---|---|
| 1 | #1215 | **The lane slice with its grounding already done, and the direct continuation of the landing that found it.** `src-element` clause 4 (`ed-with-ns`) is charged nowhere — `grep -rn 'ed-with-ns' --include=*.go .` returns nothing — and three `ibmData/schema_invalid/S3_2_3/` fixtures write `targetNamespace` on a `name=` local element and are all **accepted** against a suite that marks them schema-invalid. #1206's own post-land located them by decoding the UTF-16LE corpus that had hidden them. **Not ratchet-neutral: `schema` +2 firm (`s3_2_3si06`, `s3_2_3si08`) plus `s3_2_3si01` conditionally** — measure case by case and predict before running, per CLAUDE.md. `s3_2_3si01` carries an attribute-side violation too and is claimed conditionally by row 2; whichever lands second inherits it, and the commit body says which |
| 2 | #1216 | **The attribute-side twin, and its prediction is the firmer of the two.** `src-attribute` clause 6 (`att-with-ns`) is likewise charged nowhere, and clauses 6.1/6.2/6.3.1/6.3.2 mirror `src-element` 4.1/4.2/4.3.1/4.3.2 clause for clause. Three fixtures, **`schema` +3, all firm**, none contested with row 1. Filed apart from #1215 rather than folded because the rule ID and the producer both differ (`ruleSrcAttribute` on the local-attribute path, not `ruleSrcElement` on `produceLocalElement`) — the discipline #471 set and #1205/#1206 kept. Either order works; taking #1215 first keeps `s3_2_3si01` where its issue claims it |
| 3 | #1136 | **Third stamp carried, and every landing since has made it more expensive — this is the last stamp it may sit below a lane row.** `elementParticleTerm`'s `ref=` arm ran two checks when #1099 proved its mutation, three after #471, and **FOUR after #1206**; **#1205 makes it five and adds one to `produceLocalElement` besides**, and rows 1 and 2 charge into the same neighbourhood. Whether `checkS4SChildOrder` runs before or after the clause charge decides which fault class a doubly-violating schema reports, and the two sides sit on opposite sides of the STYLE E2 line, so the order is observable to a reader of the error. Its own body marks the spec premise as an **unproven hypothesis** and says to ground it first; #1099's mutation proof was made against a two-check arm and must be re-run, not quoted. Take it in either order with #1135, never together |
| 4 | #1229 | **The fifteenth consultation's sharpest finding, and half of it is settled before a session starts.** README enumerates three unusable-hint cases as reported on stderr; `cmd/goxsd8/doc.go` names two and calls the third — a hint naming a document that is **not there** — a silent degradation, and the code agrees, because an unresolvable `schemaLocation` is legal under §4.2.6.2 so `compileSet` returns nil and the diagnosis branch is never reached. README is false against the copy it itself calls authoritative, so the correcting branch needs no ruling. The **reporting** branch is the open half and is cheap too: `AssemblyReport.Unfollowed()` already carries what a diagnosis would say |
| 5 | #1230 | **A wrong answer with exit 0 on it, and the claim is written into four carriers including the one that would enforce it.** `-schema -` is documented as unsupported in README, in `doc.go`'s argument vocabulary and in `stdinArg`'s own `const` doc comment (`cmd/goxsd8/validate.go:28-32`) — and `rootLocation` has no `-` arm at all, so a file literally named `-` is compiled as the schema set and the run exits 0. Today the restriction holds only because that filename is rare. Above the remaining doc rows because it is the only persona finding whose failure mode is a silent wrong verdict rather than a missing sentence |
| 6 | #1223 | **#720's own arbiter follow-up, filed with a reachable-today witness rather than a hypothesis.** `validate` drops `Result.Unevaluated()`, so an instance the assessment **declined to decide** is byte-identical at the terminal to one that passed every check — exit 0, no output. An `<xs:assert>` records under `cvc-assertion` and a declined Type Table under `key-cta-ta-select`, so the case exists at HEAD. Both directions close it (surface the records, or state in the three copies that `validate` reports charged violations only), and the body pins the two settled points a fix must not re-open. **Not a false sentence today** — all three copies say *charged a violation*, never *valid* — which is why it is a follow-up and sits below the falsity rows |
| 7 | #1203 | **A latent ratchet ambush #404's landing created, and the queue now has two issues that could trip it.** Two of #404's eleven banked `schema` passes — `addB014` and `schZ006` — reject for a fault the suite does not intend, so when #603 or #703 repairs the over-rejection those passes flip down as a spurious `Regressed` the repairing session did not cause. A banked pass resting on an over-rejection is invisible in the lane file; this measures it into a known figure. No production change — a measurement and a ruling. Cross-references #1002 for the wrong-reason-pass class, whose human ruling #1203's measurement may feed |
| 8 | #1227 | **The pen bound is written and nothing checks it — first sighting AFTER the rule existed, which is what makes it fileable.** `docs/WORKFLOW.md:190-192` binds who may hold the pen and `develop.md:60` binds step 4, but **landing precondition 3 iterates over MASON commits** (`docs/WORKFLOW.md:278-283`), so a branch with none has nothing to iterate — and on #999 the honest disclosure that replaced the `MASON:` comment is itself what let the precondition pass. #636 is the only other witness and is **pre-rule** by one day, so this is n=1 post-rule and the deliverable is a runnable check, not a fifty-eighth `kind/process` issue. Both landed precedents are named in its body: #1018's doc-only precondition 4 and #963's `landcheck` |
| 9 | #1007 with #1231, #1123 and #1189 | **Now FOUR issues, one session, three contract copies, one coupling test.** All four edit `cmd/goxsd8/doc.go`, `main.go`'s `usage` const and `README.md` together, and `TestUsageCoversContract` pins the first two by **twenty-four** substrings — a fourth move of that count (5 → 12 → 16 → 24) with the property it guards unchanged, which is #398's whole argument. #1231 is the newest and the smallest: `gen` prints *"is not yet implemented"*, README says *"reserved but not yet implemented"*, `doc.go` and `usage` say *"reserved but not yet built"* — three renderings, no two alike. #1007's body was **restamped by this pass** for #720's fourth exit code and its moved line numbers. Taking them apart means rebasing the same three files four times |
| 10 | #1140 | **BANDED rather than closed, which discharges the last stamp's standing instruction in the other available direction.** That instruction — *"if this stamp still finds it unbanded and untaken, the next should close it as accepted"* — rests on a premise this row removes: it has never had a band row, so five stamps of slippage is evidence about the banding and not about the issue. The issue itself is correct, cheap and doc-only: a session entering `docs/ROUTINES.md` at `## Survey input` — which `CLAUDE.md:81` routes it to directly — never crosses `:42-61`'s caveat that `gh auth status` is not evidence about repository-scoped REST, and on #1117 one mason built a whole substitute feed on that misreading. **One sentence or one pointer, not both.** Take it with **#1217**, the other cheap `docs/ROUTINES.md` fix, and the rebase is free |
| 11 | #1167 | **Two measured sightings, #414 and #1115.** `gapaudit` reconciles marker → issue and issue → marker; nothing audits prose that points *at* a marker or its file, and the stale-marker sweep has now missed an inbound site in the deleting package twice. gapaudit's own group 1 candidate-owns it against `validate/assess.go:853`. PRINCIPLES 27 says a repeated grep wants a tool |
| 12 | #1205 | **Ratchet-neutral by construction like #471, and it foots the band for that reason — but it is what makes row 3's arm five checks deep.** `final=`/`abstract=` on a local element are prohibited `use="prohibited"` unconditionally by `xs:localElement`, on both the `ref=` path and the inline path, and neither charges either attribute. Cheap and correct and moves no lane. The **#768 collision is named on both sides** — its fixture stops reaching `produceLocalElement` the moment this lands — and is a hazard, not a blocker. Neither is labelled `blocked`. Sequence it after row 3 if both are taken in one window, so #1136's table is not re-derived a fifth time |

**Below the band, and why**: **#1156** is comment text only and its one stale
figure was already corrected — it drops out of the band this stamp because the
census that motivated it has been flat for a whole window. **#1201** is correct
and needs a warden pre-flight for its 1a fix, but **zero suite cases exercise
it**, so it moves nothing; take it opportunistically with #1203 or whoever next
touches `fabricatedRejection`, and it retires the two `GAP(conformance)` markers
when it lands. **#1183** is cheap, correct, moves nothing and makes #1051's
stage-2 deletion three predicates smaller. **#1196** measures the live residual
for #1051 and **its own body forbids banding on the numbers it produces** —
take it opportunistically, never together with #1171. **#1217** pairs with band
row 10 (both `docs/ROUTINES.md`, both cheap) and is the more mechanical of the
two. **#1135** pairs with #1136 (never together). **#1036** carries one settled
disposition in either direction and is unchanged. The four remaining fifteenth-
consultation rows — **#1232** (exit-3-over-exit-2 precedence unstated),
**#1233** (the stdin hint base URI unstated), **#1234** (the root package's
guided tour omits `loader` and `xsderr`), **#1235** (README's closure comment
omits `<xs:redefine>`), **#1236** (README names `Finalize()` and never
`FinalizeWith`) — are all one-sentence doc fixes with settled directions and are
ranked on ordinary grounds; #1236 is the README twin of **#513** and the two are
one session. **#849** carries a `## Cost of delay` reading *"a steward
re-ranking is warranted"* — an unranked issue asking for a rank, routed to
`/retro`'s steward drift review as a third item beside #841 and #1080.
**#1051** stays `blocked` on three unfiled buckets. **#1224** stays `blocked` on
a trigger — a new rule becoming chargeable at `schemaSetLocation` — and #755 is
the nearest candidate to fire it. **#1033** is still the only row no persona has
ever looked at.

**There is no `Increasing` steward ranking anywhere in the band.** #849 is the
nearest, and its own body records the ranking as falsified and unreplaced.

### Next planning action

1. **Take #1215, then #1216, and hold the ratchet to account on both.** They are
   the only two grounded lane slices in the queue and their predictions are
   already written down — **+2 firm plus one conditional** and **+3 firm**.
   `s3_2_3si01` is claimed conditionally by #1215 and excluded from #1216's
   list; **whichever lands second inherits it, and the commit body says which**.
   **Decode `ibmData/` before grounding anything against the suite** — it is
   UTF-16LE, a UTF-8 grep does not see it, and that blind spot has now produced
   one under-prediction and six filed fixtures in a single window.
2. **#1136 does not sit below a lane row again.** Three stamps carried, and
   #471, #1206 and (next) #1205 have each added a charge to the arm it is
   about — its own evidence table has been restamped twice by landings into it.
   Its spec premise is marked an **unproven hypothesis** in its own body: ground
   that first, and **re-run #1099's mutation against the four-check arm rather
   than quoting the two-check proof**. If the next stamp finds it unbanded, band
   it at row 1.
3. **The fifteenth persona consultation is folded and nothing from it is
   handed off.** Ten findings: eight filed (#1229–#1236), two folded into open
   issues that already owned them (#1122, #755), zero dismissed and zero left as
   a note. **The next consultation is owed against whatever next changes the
   observable surface** — no band row does, so name the trigger rather than
   scheduling a pass: a `cmd/goxsd8` landing, a README Library-section landing,
   or a new `xsd` query accessor. #1033 is still the one row no persona has ever
   looked at.
4. **#1140 is banded, not closed, and the next stamp should not re-derive the
   question.** The last stamp's *"close it as accepted rather than carry it a
   sixth time"* was conditioned on it being **unbanded and untaken**; this stamp
   removes the first half. If it is still untaken at the next stamp **from a
   band row**, that is different evidence and closing it is then the right call.
5. **The human decision blocking #1002 is unchanged and is now carried for an
   ELEVENTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — and (b) depends on **#1042**, filed and `blocked`. **No
   agent should attempt either.** Band row 7 (#1203) may surface a fresh instance
   of the wrong-reason-pass class (a) is meant to cover; that is a datum for the
   ruling, not a licence to act on it.
6. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3). **M6 tier 2 itself is uncarved** — `$value`
   binding, an F&O function library, typed comparison — and is too big for one
   issue. That carve is a `/backlog` act at the M6 opening, and #1042 is the thing
   it must slice around rather than a blank page. Note that #1042's
   `## Depends on` names #719, which is **closed**: the live dependency is the
   XPath evaluator itself, a trigger, and the body says so.
7. **The unblock sweep measured a clean zero for the SEVENTH consecutive
   stamp.** All **21** open `blocked` bodies were read and their
   `## Depends on` sections checked against this window's six closures (#1181,
   #1199, #1206, #1188, #720, #999); only #16 and #1051 name any of them, both
   as struck-through discharged provenance rather than a live dependency, and
   #1220 names #1188 inside a list of filing precedents. **Zero relabelled.**
   **NINE of the 21 are triggers rather than issues** and say so in their own
   `## Depends on` — #79, #555, #692, #841, #925, #1002, #1080 and now **#1220**
   and **#1224**, the two this window added; **do not re-scan those on the next
   sweep.** Of the remaining twelve, every one still has at least one **open**
   named dependency, checked issue by issue rather than inferred.

**Standing, and re-checked rather than restated.** Four unlanded corrections still
target one paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**,
**#646**, **#679**, **#912** — and whichever lands last rebases three times.
**The next `/retro` inherits seven**: #692, #925, #841, #1080, the
fold-the-five-species question (#635, #912, #609, #510, #646), the
`[tests that cannot fail]` **pattern** (routed by #472's post-land; its one
remaining queue instance, #999, has since **landed**, so what `/retro` inherits
is the pattern with no filed carrier left), and **#849's owed steward
re-ranking**, the third item for the steward's drift review. **#841 is still the
counter-example the steward-ranking rule cannot reach**: a `kind/refactor` with a
steward ranking, `blocked` because its trigger has no mailbox, fired twice
without a ruling. **#999's post-land banked one more `/retro` datum and did not
file it**: the n=2 friction of an arbiter's blocking finding landing on an item
the mason account had itself declined to prove, which that pass deliberately
left banded because overturning a retro banding is not a post-land pass's to do.
The `gh --paginate` page-2 403 trap is **already named** in docs/ROUTINES.md's
Survey-input recipe comment, so it stays a `/retro` datum and not a filing. The
CTA cohort's 45 banked `instance` failures remain unattributed. `gate.yml` runs
and is still not a required status check, which only the repository owner can
change.

**Environment, one witness each.** Repository-scoped `gh api` REST served every
read and **twelve writes** here — **8** issues filed (#1229–#1236), **3** bodies
edited (#1007, #1122, #755) and **1** thread comment (#755) — plus this
section's own replacement. The paginate recipe ran to **13 pages** twice, before
and after the writes: 12 full at 100 both times, page 13 short at **28** then
**36**, each re-requested per the recipe with no larger answer, and the
post-loop fullness check passed on both runs. GraphQL was not attempted; the
ranking docs/ROUTINES.md owns was followed from the top. The shallow clone
truncated `origin/main` to **51 commits** (**#802**), which is why two
`claude/*` ancestry reads came from commit dates and why one of them **flipped
verdict on an unmoved tip** since the last stamp. No conformance measurement was
taken by this pass: the lane table above is the committed expectations, which
`docs/WORKFLOW.md` names as the lane score (#1120).

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
same construction, still open) and **#1206**, which **landed 2026-09-04**
(`1780670`, `schema` **+16**) — `src-element` clause 2.2's prose-only eight
attributes on a local `<element ref=>`, and the demonstration that this family's
fixtures do exist where the grammar pair's do not. **#1206's own landing then
carved two more**, from a UTF-16LE corner of the suite no UTF-8 grep had
reached: **#1215** (`src-element` clause 4, `ed-with-ns`) and **#1216**
(`src-attribute` clause 6, `att-with-ns`), three fixtures each and both
open. Live producer-decides-and-accepts members are those two plus **#931**
(occurrence attributes on a named `<group>`'s child compositor), **#929** and
**#455**. A
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
queues for a day. **The CLI's own `validate` subcommand, #720, LANDED
2026-09-04** (`f4d8f76`) and moved no lane by construction — `go list -deps
./conformance | grep -c cmd/goxsd8` is `0`. It is the reason this milestone's
open count grew while its lane did not: four issues now sit on M5 that no lane
can see (#1223, #1224, #1229, #1230), all filed against the subcommand rather
than against the engine.

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
