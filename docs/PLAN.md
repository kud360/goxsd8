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

## Status — 2026-08-22 (backlog pass; six landings absorbed, lanes/milestones/queue and both surveys re-derived, this pass's persona findings folded, one stale-premise body rewritten and banded first)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10752 | 15609 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13076 | 2322 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**`schema` moved 12983 → 13076 (+93), `instance` 10746 → 10752 (+6), and
`datatypes` did not move.** The attribution is per commit and verified against
the tree, not summed from verdicts: `git show <commit>:…/schema.txt` counted
across `6bc5df9..9c10af8` puts **+12 on #945** (`24e47c6`), **+80 schema and +6
instance on #909** (`3d3ab5f`), **+1 on #957** (`9c10af8`), and **zero on #932,
#565 and #924** — exactly as each declared. 12983 + 12 + 80 + 1 = 13076 and
10746 + 6 = 10752, so the three moves are the whole of it. **All three are clean
in the strong sense**: every flipped line is `fail` → `pass`, zero downward, and
the case-ID column is byte-identical before and after each, so nothing was added
or removed from any lane file.

- **#945's twelve** are `MS-Notations2006-07-15/notatF001`, `F005`, `F013`,
  `F015`, `F019`, `F027`, `F029`, `F031`, `F039`, `F055`, `F063`, `F067` — the
  harness residue #928 left, banked one landing later. The previous stamp
  forecast "up to +13" with a floor of 0; it delivered **12, one under its
  ceiling**, with the thirteenth (`notatF035`) dismissed to #404 on measured
  facts rather than on a reading. **`notatF067` was the case shared with #786**,
  and #945 took it, exactly as that issue's Acceptance 4 said whichever landed
  first should — the note is already on #786's thread.
- **#909's eighty `schema` cases** spread across ten suite directories:
  `MS-ComplexType2006-07-15` 36,
  `Simple` 13, `TypeAlternativeTests` 10, `MS-SimpleType2006-07-15` 8,
  `MS-Particles2006-07-15` 5, `CTA` 4, and one each from `Wild`,
  `MS-Additional2006-07-15`, `CType` and `Assert`. Its six `instance` cases are
  `CTA/cta0001.n01`, `cta0001.n02`, `CType/basetd00101m1/Negative` and three
  `TypeAlternativeTests` instances. **The issue's own upper bound was 103 `.xsd`
  files carrying the shape and the previous band said the sizing needed a
  `GOXSD_DECLINES=1` census first; the census was run and the landing came in at
  80.**
- **#957's one** is `CTA/cta9001err`. A single case is the honest yield of
  charging `src-element` clause 5, and the issue said so before it was worked.

Landings absorbed by this stamp, all six at `6bc5df9..9c10af8`:

- **#945** (`24e47c6`) — one unexported pre-pass, `holdsMisplacedNotation`
  (`conformance/schema.go:719`), mirrors `rejectS4SFaults`' walk so the schema
  lane stops declining documents the parser already rejects unconditionally. Per-case
  measurement replaced the body's hypothesis table, and the per-case disposition
  for all thirteen is on the thread. Arbiter REJECT round 1, repaired by the
  orchestrator rather than a second mason round.
  `Ratchet: schema 12983 -> 12995 (+12)`.
- **#932** (`35ecd76`) — a malformed `minOccurs`/`maxOccurs` is charged
  `cvc-datatype-valid`, and **`parser` now charges `p-props-correct` nowhere**:
  `nonNegativeInt` had borrowed a Schema Component Constraint over a Particle's
  properties for a lexical fault that builds no Particle at all. `ruleParticleCorr`
  deleted rather than parked. `Ratchet: unchanged`, measured.
- **#565** (`738db7a`) — `.claude/agents/mason.md` gains a required "The
  implementation account" and `docs/WORKFLOW.md`'s Landing section a **third
  verified precondition**, coverage-by-SHA over the thread's comments. The
  process row the previous stamp put at the head of the band, landed the next
  day. Two arbiter rounds, REJECT then ACCEPT. `Ratchet: unchanged`, measured.
- **#924** (`53bf113`, its LOG entry at `aeae89f`) — `xsd.ValueSpace.ValidDefault`
  returns `(cause error, decided bool)`, so `cvc-complex-type` **clause 4** — the
  fourth and last String Valid delegation — carries the Datatype Valid verdict it
  already held. Warden pre-flight and warden diff review both ran, per #484's
  condition. `Ratchet: unchanged`, measured. **Its squash landed without the LOG
  entry and a second commit carried it**, recorded on the thread as a process
  correction.
- **#909** (`3d3ab5f`) — `<simpleContent>` with `<restriction>` is BUILT:
  `produceSimpleContent` runs both alternants through one body and the five-case
  §3.4.2.2 tableau. **The one whole complex-type representation form still
  declined outright is now produced**, and the gate widened in lockstep with the
  producer. `Ratchet: schema 12995 -> 13075 (+80); instance 10746 -> 10752 (+6)`.
- **#957** (`9c10af8`) — `src-element` **clause 5** is charged: a non-final
  `<alternative>` with no `test` is REJECTED at mapping time instead of mapping
  to nothing and being accepted. The site had carried **prose instead of a
  `GAP(` marker**, so no survey could see it. One mason pass, no repair round.
  `Ratchet: schema 13075 -> 13076 (+1)`.

**Every landing's follow-up ledger is disposed, and this pass found nothing left
to catch up.** The previous stamp had one landing's ledger to catch up (#867's);
this one had none. Checked one thread at a time rather than assumed: #945's two owed notes
are posted (#786's banked-witness note, and the per-case disposition on #945
itself); #932's `processContentsOf` follow-up is **#950** and its thread carries
the three records the commit body did not; #924's `go tool surface`
signature-blindness hand-off went to **#741**, which already owned it — the
post-land pass checked the queue first and filed nothing, which is the behaviour
#943 failed to show — and its FAIL-OPEN CONTRACT half is **#953**; #909's
four-alternant grammar fault is **#956** and its three remaining findings are
**#958**; #957's prose-gap-invisibility class is **#960**. **#565's own sibling
#566 was routed nowhere and correctly so** — it is open, `ready`, and the develop
loop takes it in its own iteration.

**No duplicate reached the queue this pass and none was closed.** The one
candidate was checked and is not one: the 2026-08-22 libuser reported the
`codegen` doc-marker gap as a NEW finding, and **#409 has owned it since
2026-08-02** — the sighting is recorded there rather than filed again. One
genuine cross-reference gap was closed by comment: **#958**'s acceptance item 1
and **#561** are the gate side and the producer side of the same `assertions`
facet-name/element-name seam, and #958's body named neither.

Milestones, read from `repos/kud360/goxsd8/milestones` this pass and
cross-checked against the paginated issue list, which agrees exactly.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 94 | 48 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**M4 absorbed four of the six landings and its open count did not move**: #945,
#932, #909 and #957 closed on it (90 → 94) while #950, #956, #957 and #958 were
filed on it, so 48 stays 48. M5 moved for the first time in two stamps — #924
closed on it, 14 → 15 closed and 14 → 13 open, with no M5 filing behind it.
**170 of the 231 open issues carry no milestone** (231 − 48 − 13), so the
milestone rows are feature progress and the queue paragraph below is the queue.
#565 is this stamp's instance of the gap: a process landing that took a band head
off the queue, closed with no milestone, as every `kind/process` issue is.

Queue: **231 open — 204 `ready`, 27 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 27), against **343 closed**.
204 + 27 = 231 exactly, and **every one of the 231 carries a queue label** — the
class #773/#774 fell into is empty for the eighth consecutive stamp. Both figures
were re-derived by paginating the issue list (page-numbered, not `--paginate`,
whose Link header uses numeric-ID URLs the proxy blocks), raising the page count
until a page came back empty, and discarding pull requests, which share the
endpoint. **The move reconciles exactly, and in both directions**: closed
337 → 343 is the six landings and nothing else; open stays at 231 because six
closures met six filings — **#950**, **#953**, **#956**, **#957**, **#958**,
**#960** — of which **#957 was filed and landed inside the same window** and so
appears on both sides. On the label rows, `ready` 205 → 204 is six `ready`
closures against five `ready` filings, and `blocked` 26 → 27 is #960 alone.

**The unblock sweep moved nothing, for the fourteenth consecutive pass, run as a
parse rather than by eye.** Every one of the 27 `blocked` bodies was fetched and
its `## Depends on` scanned for the six just-closed issues (#945, #932, #565,
#924, #909, #957): **not one names any of them.** Every live dependency line
still names an issue that is **open, re-checked by number this pass** — #831,
#719, #472, #248, #591, #414, #455, #407, #250, #79 — and the rest are triggers
(the next `/retro`, a steward drift review, an epic target, a ruling), several of
which say in their own text not to re-scan them. **No `## Depends on` was
repaired**, and no `blocked` issue is startable today.

**One open body carried a stale premise and was REWRITTEN, not commented at —
and it is this pass's band head.** **#820** was filed on 2026-08-16 saying "two
instances, one day apart"; the count is now nine, of which **seven are
consecutive landings** (#928, #945, #932, #565, #924, #909, #957), each recording
that the `docs/LOG` entry was absent from the branch when the arbiter judged, and
each numbering itself. **All seven cite #528, which has been CLOSED since
2026-08-09, and not one cites #820** — counted mechanically over the whole of
`docs/LOG/2026-08.md`: `#528` appears **45** times, `#820` appears **0**. The body's
"hypothesis, not a finding" note was also wrong by 2026-08-22 and is replaced:
#565's and #924's entries **state** the mechanism, and it contradicts #528's
ruling rather than confirming the guess — #528's step 6.0(a) checks the entry
*before the PR opens*, which is **after** the verdict, so on a branch with no
other reason to commit again the precondition is satisfiable only post-verdict.
The rewritten body therefore asks the defect question first and defers the
check-form work behind it.

**No README line citation needed correcting this pass, which is the first stamp
in three that can say so.** `README.md` has not changed since `6cd5b34`, and the
ranges the previous stamp fixed in #669 (`:90-110`, `:135-142`), #625
(`:119-124`) and #492 (`:116-117`, `:86-142`) were all re-checked against the
file here and all hold. The one attribution slip this pass came from the persona
side, not the tree: the 2026-08-22 libuser filed "README's example-test roundup
never lists `xsd/example_test.go`" under #625, and it is **#669**'s — that issue's
title names it and its Acceptance already requires the pointer list be re-derived
from the tree rather than extended by one entry, which also catches the
`value/backendtest/example_test.go` omission neither persona saw.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed the paginated issue JSON):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
822    wip/issue-822  151h9m0s   RETIRED  wip/issue-822: issue #822 is closed
862    wip/issue-862  main's     CLAIMED  wip/issue-862: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  117h11m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  45h24m0s   RETIRED  wip/issue-933: issue #933 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-862`, `wip/issue-872` and `wip/issue-933`, re-read this pass — the
same five as the previous stamp. **All six of this window's landings left no
ref behind**, GitHub's auto-delete having taken each at merge. Nothing is
EXPIRED, no `parked/*` ref exists, and the four rows are unchanged in verdict.

**`wip/issue-862` is still a LIVE empty claim, and its clock has now crossed
the threshold the previous stamp said it was well inside.** It sits at
`c2ba631`, which is no longer `main`'s tip but is still not a commit of its own,
so it stays `ahead 0`, can never be EXPIRED, and can never be retired on age
(#722). Its thread's last comment is the GROUNDING posted **2026-08-20T20:20Z**
and nothing since — **42 hours as this stamp was written**, against the ~2-day threshold #867's
takeover used and that **no document states**. That rule is **#946**'s to settle
and #946 is `blocked` on the next `/retro`; until then #862 is off-limits by the
same judgment the previous two stamps applied, and the grounding remains the
asset a session taking it would start from. **`wip/issue-933` is RETIRED and
kept** — #933 closed as #862's duplicate and the branch is the grounding
session's, at the same SHA. **`wip/issue-822` and `wip/issue-872` are RETIRED and
kept**, superseded by #851 and #878. All three deletions are a human's call.

**`go tool gapaudit`, verbatim** (fed `--label kind/gap --state all`-shaped JSON):

```
gapaudit: 64 GAP marker(s) across 5 area(s)

=== Per-area census ===
  validate         14
  value            3
  xml              4
  xpath            6
  xsd              37

=== Group 1: markers with no OPEN tracking issue found ===
(matching is heuristic — file path or a distinctive phrase; treat a
listing here as "needs a human look", not as proven untracked)
(none)

=== Group 2: OPEN kind/gap issues with no surviving marker ===
(a stale tracker if the gap was a marked fail-open site — but
kind/gap also labels conformance-lane gaps, which never carry a
marker and belong here permanently)
  #398 cmd: the notImplemented stub is untracked P3 debt (no GAP(cmd) marker), and TestUsageCoversContract guards four substrings of a whole-block coupling
  #404 conformance: closureScan.unresolved is scan-scoped, so the decline conjunction is coarser than the hazard it models
  #569 parser: the ROOT half of the pre-scan's `<annotation>` exclusion is unpinned — a top-level `<xs:annotation><xs:appinfo><xs:key name="…">` was falsely indexed before #384 and is correctly excluded after, in both cases with no test
  #591 conformance: an instanceTest caseSpec DROPS its sibling schemaTest <schemaDocument>, so readFacetsCase declines time_minInclusive006_1163.i for a schema the catalog names explicitly
  #592 conformance: string_pattern002_1031.i — a <list itemType> over a USER-DEFINED item type in a target namespace, with a multi-leaf instance shape no cohort reader matches
  #593 conformance: decimal_totalDigits004_1060.v carries its tested value on the ROOT attribute, a shape readFacetsCase exactly-one-<foo> guard declines
  #719 validate: cvc-assertion wired fail-open at every variety level — the M6 seam, marked and measured
  #787 value: an enumeration member outside the base's value space is charged enumeration-valid-restriction (§4.3.5.5) at schema construction and never src-enumeration-value (§4.3.5.3) — which rule the spec assigns is ungrounded, and restriction.go's wrap shadows newEnumFacet's remap
  #921 conformance: <current status="queried"> is unmodeled, so the two gMonth XSD-1.0-lexical cases are permanently unbankable disagreements with no owner and no stated reason
```

**Group 1 is EMPTY for the eighth consecutive stamp: every marked site has an
open tracker.** 64 markers, composition unchanged, and the raw `GAP(` token
count is **96 at all four of `6bc5df9`, `24e47c6`, `3d3ab5f` and `9c10af8`** —
six landings across `parser/`, `conformance/`, `validate/`, `value/` and `xsd/`
added no new fail-open site and removed none. Group 2 is **unchanged at 9**: no
`kind/gap` issue on that list closed and none was added to it, though three
`kind/gap` issues were filed this window (#956, #957, #958) and #957 closed the
same day.

**#957 is the counter-example that makes the marker census worth reading
sceptically, and it is now filed as such.** Its site had disclosed a real
fail-open in **prose with no `GAP(` marker**, so it appeared in neither the
census nor Group 1, and the only reason it was ever worked is that an arbiter
judging an unrelated landing happened to read the site. That class is **#960**,
`blocked` on the next `/retro`, with three witnesses already on the record
(#909's `[P3]`, #913, #957). **#852** still owns the matcher qualification —
the raw-token count of 96 against the tool's 64 — and stays below the fold
because the tool again ran with reconciliation and Group 1 empty.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 204 of
them. Take from the top. **Four rows of the previous band LANDED** (#565, #945,
#924, #909); the band is re-cut by dropping them and re-deriving every
cross-reference by ISSUE, never by row number, which decays at each re-cut.

**The head is a process row for the second consecutive stamp, and for the same
reason with a different issue.** #565 was ranked first on 2026-08-21 and landed
within a day; #820 replaces it on the identical argument, with a longer streak
(seven consecutive landings against #565's five at the time of its ranking) and
the same leak — a filed, `ready`, specified issue that the sessions re-observing
its friction routed to a closed one instead. CLAUDE.md's rule is not
conditional, and this is now the second time in eight days it has been the
correct read.

The two persona rows ran a fresh 2026-08-22 consultation against the published
surface only, by the orchestrating session — a cartographer that has read the
source cannot role-play a consumer. **Both personas reconfirmed all eleven
issues and filed NOTHING NEW: cliuser's is the fourth consecutive such verdict
and libuser's the second.** Comments were posted on exactly two threads (#687
and #409) and only where the personas produced mechanism the bodies did not
carry; the other nine hold substantively current evidence already, so no
re-stamp was posted — the "if not already current" rule.

| # | Issue | Why here |
|---:|---|---|
| 1 | #820 | **The process row that outranks every lane slice below it, on CLAUDE.md's own rule, with SEVEN consecutive landings recording it** (#928, #945, #932, #565, #924, #909, #957) — the `docs/LOG` entry absent from the branch when the arbiter judges. The body was rewritten this pass: its "two instances" premise, and its "why the entry is late is a hypothesis" note, were both false by 2026-08-22. **The first move is not the check's form** — it is whether there is a defect at all, because #565's and #924's entries state a mechanism that CONTRADICTS #528's ruling: 6.0(a) checks before the PR opens, which is after the verdict, so the empty diff may be the designed order. #528 is closed and cannot be re-scoped; whichever way this rules, the seven entries stop logging it |
| 2 | #956 | **Four false accepts, `schema` candidate, and `parser/produce_complex.go` is warm from a landing hours old.** No complex-type derivation alternant enforces the s4s child ORDER or its `maxOccurs`, so `ctC011`/`ctD034`/`ctD042`/`ctD043` are decided-and-ACCEPTED at all four alternant sites. Filed by #909's post-land pass on the arbiter's own words ("worth an issue of its own"); the four cases were declines yesterday and are false accepts today, so **no banked expectation depends on it either way** and the floor is a correctness fix rather than a lane number |
| 3 | #958 | **#909's three remaining one-site defects, one landing, same warm files.** `facetElement` (`conformance/schema.go:1173`) admits the FACET name `assertions`, which the s4s grammar has no element for and the producer silently drops; a repointed test at `schema_test.go:484` still names the fixture shape it no longer uses; and case 5's inline-`<simpleType>` drop has no test. **Acceptance item 1 shares its seam with #561** — the gate side here, the untested producer-side exclusion there — cross-referenced on both threads this pass, sequence rather than merge |
| 4 | #950 | **#932's own follow-up, filed by its post-land pass and specified by MECHANISM rather than by symptom.** `produceWildcard` calls `processContentsOf` (`produce_complex.go:2694`) **before** `xsd.NewWildcard`, so a lexical `processContents` fault is charged `w-props-correct` clause 1 — a constraint over a component that does not exist — which is exactly the two-layer split #932 corrected one layer up. Requires its own GROUNDING on whether §4.1.4 reaches an `xs:NMTOKEN` enumeration in Appendix A; **#884** is the adjacent shape and closes neither |
| 5 | #846 | **#909 paid the shadow tax once more and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`, correct only because the arbiter walked `attrDeclsDecidable` against `main` by hand.** Rows 2 and 3 will both pay it again. Ranked BELOW them and the tension is stated rather than hidden: landing this first would make them cheaper, but it is a ~700-line refactor with no evidence it fits one session, and two warm measured slices should not wait behind an unsized one. If it grows a third witness, it moves above them |
| 6 | #941 | **#387's own unfiled debt, and the only band row whose files have gone cold** (`07117dc` is now four days back). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete (the field/method collision that forced #387's) and `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 7 | #953 | **#924's other post-land filing, and a doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says "every caller turns a decided negative into a schema rejection"; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules. Pre-existing on `main`, out of scope for #924 by its arbiter's own ruling, and discharged there only by a commit body naming it in prose |
| 8 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 9 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 10 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced; its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. **Its measured delta shrank this window**: `notatF067` was one of its witnesses and #945 banked it, which is recorded on the thread, and the `restriction == nil` arm is untouched, so the measurement is still the thing to run. Read #868's diff first |
| 11 | #907 | **A repo tool catalogues every `childElement` sibling-alternative site and reports which is unguarded** — hand-written guards accumulated over months, each after a suite case tripped over it. `kind/tooling`, banded below the lane rows because the tax was paid over months rather than in consecutive sessions — the discriminator row 1 satisfies and this one does not. **Its census is stale by at least two more landings and is NOT re-derived here**: #909 rewrote 418 lines of `produce_complex.go` and #957 moved `produce_typetable.go`, so re-run the census before designing from the body's figures |
| 12 | #669 → #625 → #748 → #896 → #492 (+ #934) | **README's Library block, ONE row in the order the issues name; splitting it is why it sat.** 2026-08-22 libuser reconfirmed all six and filed nothing: **#669** the "works TODAY" snippet still fails to compile (`parser.WithLogger(logger)` names an identifier the block never declares, beside three unused-variable errors); **#625** still points at closed #203 while `xsd.Example_buildFinalizeQuery` exists and passes; **#748** still says shipped `validate.New`/`xmlsrc.Validate` "do not exist yet", with all three signature mismatches holding; **#896** the package "Contract" prose still never names `Err()`; **#492** README omits `ParseReport`/`AssemblyReport`/`ReadDocument`/`Produce` (grep: zero matches); **#934** the violation example still shows `[cvc-datatype-valid]` where #913/#914 now charge `[cvc-type]` with the old rule reachable only as the wrapped cause. **No citation needed correcting this pass** |
| 13 | #870 + #747 + #514 + #687 + #672 | **The CLI is unreachable from its own documentation; 2026-08-22 cliuser reconfirmed all five and filed nothing — fourth consecutive such verdict, so the gap is disclosure not discovery.** **#687 gained two behaviours and its body a third Acceptance question**, both re-reproduced here at `9c10af8`: `goxsd8 -xyz -help` prints full help and exits **0**, silently swallowing the bogus flag, and `-help=true` — the stdlib boolean-flag idiom — is NOT recognized and falls to the stub with exit 2. Both follow from `wantsHelp` being a raw token scan with three exact string comparisons rather than flag parsing. **#870** Quickstart's `go build ./...` writes no executable and the stub's own `go doc` remedy fails outside the module root; **#747** `-help` is a strict subset of `go doc`; **#514** every non-help input is byte-identical stderr plus exit 2; **#672** `-version` in any spelling hits the stub |
| 14 | #885 | **The scope rule that has cost a reject round TWICE and still survives only in verdict comments.** #876 and #904 each shipped half a Goal because an Acceptance exemption was read as a scope exemption; #374's finding 1 added a third datum. Three discriminators, one sighting each — the body says if the third will not fit the sentence, state the two-datum rule and why |

**Deliberately unbanded, and why.** **#409** is `ready` since 2026-08-02 and now
carries its **third** independent sighting — the 2026-08-02 steward audit that
filed it, and the 2026-08-11 and 2026-08-22 libuser passes, both of which reached
it from the published surface alone with no knowledge that it existed — `codegen/doc.go` prints `Generate` and
`Target` in a code block while the package exports nothing, and `#203`'s landed
`xsd/doc.go:213` heading is the exact spelling to copy. It stays unbanded only
because it is one row of a five-file convention landing and no session has been
blocked by it. **#937** is correct and `ready` but says in its own body that it
is naturally folded by the next landing touching `rejectRepeatedAnnotations`.
**#920** and **#921** are conformance-bookkeeping follow-ups below the fold.
**#929** and **#931** are the small parser occurrence / rule-mapping gaps #901
exposed (#932 took the third); read each beside #901's thread. **#862** is
`ready` and its grounding is banked, but its branch is a LIVE empty claim whose
clock has now crossed #867's unstated threshold — off-limits until #946 rules,
and it is the worked example #946 asks for. **#888**, **#889**, **#894** are the
three `area/xpath` gaps still awaiting a suite census in their range (#889 states
a warden pre-flight per #484). **#843–#849** are the 2026-08-16 audit's findings,
**six open** — #847 closed `not_planned` on 2026-08-17 — of which **#843** has
the steepest cost of delay and **#846** is banded above. **#566** is #565's open
sibling, routed nowhere by #565's landing and correctly so. **#871** stays
`blocked` on #831. **#881**, **#548**, **#622**, **#692**, **#696**, **#796**,
**#841**, **#925**, **#946**, **#960** are `blocked` on the next `/retro` (or a
ruling), not on any landing — **ten of the 27**, and #960 is this pass's addition
to that list. **#570** carries the standing `schema` decline-count argument at
893; that figure predates three lane-moving landings including #909's +80 and is
**not re-derived here**, which makes re-deriving it #570's first move rather than
a background fact.

### Next planning action

**Take from the top: start at #820**, and do not read past it to a lane row. It
is `ready`, gated by nothing, and seven consecutive landing entries have now
recorded the same friction while routing it to an issue closed on 2026-08-09.
The previous stamp made this call for #565 and #565 landed the next day; the
argument here is stronger, not weaker, because the streak is longer and the
tracker's own body was wrong until this pass rewrote it. **Its first move is a
ruling, not an edit** — decide whether an empty `docs/LOG/` diff at verdict time
is a defect at all, because #528 closed on the opposite reading from the one
#565's and #924's entries now state.

**#956 and #958 are the warm lane follow-ons**, both filed by #909's post-land
pass hours after that landing, both against `parser/produce_complex.go` and
`conformance/schema.go` while the five-case §3.4.2.2 tableau is still legible.
**#950 is the third**, #932's own mechanism-scoped follow-up, and it needs a
grounding before code.

**The consultation produced no new issues for the second consecutive pass, and
the value it produced instead was mechanism rather than citations.** The
2026-08-19 personas produced #913 (`instance` +9409); 2026-08-20 produced #934;
2026-08-21 produced nothing new but exposed five wrong README line citations;
2026-08-22 produced nothing new and **no citation was wrong** — instead cliuser
produced two reproducible CLI behaviours that went into #687's body, and libuser
re-found a defect #409 has owned for three weeks. **A persona that keeps
re-finding the same open issues is measuring disclosure, not discovery**, and the
right response is to land the row rather than to schedule a fifth reconfirmation:
#409, #669's block and #870's block are all one-sentence or one-branch fixes
against a surface that is still empty enough to change freely.

**Four unlanded corrections still target one paragraph of `docs/WORKFLOW.md`'s
filing discipline** — **#510**, **#646**, **#679**, **#912** — and whichever
lands last rebases three times. Decide one issue or four before filing a fifth.
**The CTA cohort's 45 banked `instance` failures remain unattributed**, seventh
consecutive stamp carrying it. **`gate.yml` runs but is still not a required
status check**, which only the repository owner can change. All three stay open
and stay true.

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
ACCEPTS** (child order and `maxOccurs` across the derivation alternants,
#956; a facet element name the gate admits and the producer drops, #958)
rather than forms it cannot build. The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes
file against the same packages sit outside it, so the milestone is a floor
on M4's remaining work and not the whole of it.

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
+9, unioned onto #716's). `instance` stands at **10752** — #913's cvc-type
clause 3.1 landing added **9409**, itself M5 and the largest single lane move
this project has recorded — and **twenty-five** of the pre-#913 cases were not
M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure (+12), #821
added 1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). (The
*thirteen* this paragraph carried predated the CTA pair and did not sum; the
five figures do, and the sum is the number.) **It happened again after #913:
#909 — an M4 landing — took `instance` 10746 → 10752 (+6) by producing
`<simpleContent>` `<restriction>`, so the outside-M5 total is now 31.** A slice
that produces a component the engine could not previously see, or decides a
`{type table}` it previously withheld, moves `instance` without deciding a new
`cvc-` rule.

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

**10752 is still a floor built for soundness, and #913's +9409 jump did not
change what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the 15609 that
still fail are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read 10752 as 41% of the suite passing.** It is the count of documents
this engine can honestly call not-valid. It grew most because #913 decided
`cvc-type` clause 3.1 — the commonest simple-typed-leaf shape the lane had
declined outright — which is the counterpart to #790's lesson, not a
contradiction of it: a slice that decides a *new* rule moves the number far MORE
than its rule count suggests when the declined shape is common, and #913 moved
it more than #790's descent did. The number stays a soundness floor, not a pass
rate.

**ONE case is decided and decided WRONG — #771**, a root whose declaring schema
is reachable only through the instance's own `xsi:schemaLocation`. It was four,
and #800 retired two of them: `Assert/assert_019/instance/assert_019_2` and
`CTA/typeAlternatives_001/instance/typeAlternatives_001_2` now decline honestly
instead of rejecting a document the ·conditionally selected· type admits.
**That is two, not three, and `CTA/cta0008.v01` was never among them** — it
takes §3.12.2's inline arm, which #800 deferred to #822 and which **#851 landed
on 2026-08-17** (#822 closed, superseded), and the count was
measured by diffing `GOXSD_DECLINES=1` across the two trees rather than
predicted. All were already banked `fail` before #790 and are not its
regression; the descent is what made them visible, and the trade of a wrong
decision for an honest decline is one the lane cannot register as movement.

**#913 added a second decided-wrong class, so the count is no longer one.**
Seven CTA documents are false-charged through `cvc-type` clause 3.1 until
§3.12.4's `{inherited attributes}` merge lands (#831, #871) — an
honest-decline-to-wrong-decision trade the ratchet's zero-flip-down cannot
register, escalated on #831's thread.

The decline census that separated harvest candidates from indeterminates
predates #766, #715, #740, #790, #718, #716, #813, #913 and now #909 — and is
not re-derived
here. **It is now, by a wide margin, the oldest measurement this milestone still
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

What remains of the CTA subset is **#859** (the wildcard `ta-AttrName` arms,
under a live lease as of 2026-08-18), **#888** (a cast target that is
`xs:QName`), **#889** (value-level numeric widening, §B.1 rule 1.1's
float→double promotion, which #858 withheld rather than faked) and **#894**
(err:XPST0051/XPST0080, the static remainder #886 did not charge). **#871** is
the §3.12.4 clause 1.1.3 ·inherited attributes· merge, blocked on M4's #831.
None of the seven carries a milestone, which is the same pattern M4's tail
records.

**Assertion evaluation is still unfiled, and it is this milestone's real body of
work.** **#719** wires `cvc-assertion` fail-open at every variety level today,
one milestone early, because the `instance` lane must decline every case whose
outcome turns on an assertion. **#56** — a fail-open "unevaluated" must never
read as a genuine PASS — now depends on **#719 alone**: #842 was struck from its
`## Depends on` when it landed, because it settled the *evaluator* side of the
encoding (a compile-time `(CTATest, bool)`, `ok` false being the withhold, with
`Evaluate` always deciding) and left only the consumer side, which is #719's.
One encoding, decided in #719 and reused here (STYLE D4). STYLE 9's fail-open
discipline is only honest if a fail-open answer is distinguishable from a real
pass.

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
