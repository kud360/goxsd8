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

## Status — 2026-08-23 (BAND RE-CUT ONLY, by #966's post-land pass: #966 landed as PR #974 / `ebf6fa2`, row 1 is dropped, **#884** is promoted to the head, **#975** is filed and deliberately unbanded. The lane, milestone, queue and marker figures below are the earlier 2026-08-23 #968-park derivation CARRIED UNCHANGED — not re-read here, and #966's close has moved the queue counts since. The next `/backlog` replaces the whole section)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10752 | 15609 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13225 | 2173 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**No lane moved, and no lane COULD have.** The only commit since the previous
stamp is `b562f4a`, an append to `docs/LOG/2026-08.md` (PR #971). `git diff
--stat 5304863..b562f4a` is one file, 237 insertions, no deletions — and it is
not a `.go` file — so every
figure above is byte-identical to the previous stamp's by construction rather
than by coincidence — the distinction that matters when a stamp reports movement
of zero.

**What this window was: an attempt that ran the check its own Acceptance
demanded, disagreed with its own premise, and banked nothing.** #968 asked
`simpleTypeDecidable`'s `<restriction>` arm to decline an XSD-namespace child
Part 2 §4.1.2's content model has no position for. The oracle confirmed every
premise of the body and corrected none. Mason implemented it exactly as
specified, added twelve tests with both failure-capability checks, and then
measured the flip set — **`schema` declines 788 → 823, 35 cases added, 0
removed** — and found **8 banked `pass` turning to `fail`**. Nothing was
committed; `wip/issue-968` still carries no commit of its own. The whole record
is on the thread, and this pass's job was to dispose of it.

### The re-plan — #968 closed as superseded by #972

**The two regression families do not split into two replacements, because both
dissolve one layer down.** That is the finding, and it is why this pass filed one
issue rather than two:

- **Family A** — `stC012`, `stC013`, `stC023`, `stC028`, each carrying a SECOND
  independent fault the producer already rejects (`[src-resolve]` clause 1.1 on
  a base that is no builtin in three, `[src-attribute]` clause 3 in the fourth;
  two re-read in the tree this pass). A gate decline throws away a rejection
  that was already correct. A producer rejection reaches the same `invalid` and
  moves nothing. **Cost at the producer: zero.** Their value is as evidence —
  the fault that decides each document lives in a `base` attribute or a sibling
  top-level `<attributeGroup>`, never in `<restriction>`'s child list, so no
  predicate reading that list can tell a fabricated verdict from a correct one.
  That is **#846**'s thesis stated as a measurement, and it is on #846's thread.
- **Family B** — `VC/vc003`–`vc006`, where §4.2.2 conditional inclusion strips
  the offending child before any check sees it. **This mechanism already had an
  owner: #732, filed 2026-08-12, `ready`, with `vc007` and `vc_003_1` named in
  its Acceptance.** #968's park entry in `docs/LOG` says it "has no issue to
  split it onto" and that filing one is a prerequisite; that was derived from
  `git grep` over `*.go` — where the mechanism genuinely is unimplemented and
  unmentioned outside one warning comment in `conformance/runner.go` — and not
  from the queue. The four fixtures are recorded on #732 with the double
  negative that makes them pass today.

**#972** is the replacement: `restrictionFacets` REJECTS the XSD-namespace child
it cannot map, at the `<simpleType><restriction>` caller only, charging §5.1's
first bullet as a plain wrapped `fmt.Errorf`. It is **`blocked` on #732** —
until the `vc:` cohort is pruned, rejecting an out-of-model child turns four
banked passes into four false rejects — and it states in its own Acceptance that
`conformance/schema.go` is not touched at all.

**The re-plan found something the gate route could never have bought, and it is
the reason the replacement is worth having.** Of the 27 cases mason classified as
neutral (`fail` before, `fail` after), at least one is a case the producer route
BANKS: `MS-SimpleType2006-07-15/stF005`'s only fault is an `<xsd:whitespace>`
child — lowercase `s`, where the legal name is `whiteSpace` —
`testdata/xsdtests/msMeta/SimpleType_w3c.xml` records `<expected
validity="invalid"/>`, and `conformance/testdata/expectations/schema.txt`
records it `fail`, so the harness reports `valid` for it today. **Verified for
that one case only; the other 26 are hypothesis** and #972 requires each to be
reported by name. A gate decline could only ever have kept all 27 at `fail`. So
the replacement's expected ratchet direction is **UP** where the parked issue's
was flat at best.

**Filed: one. Closed: one. Bodies corrected: four. Nothing was handed off.**

- **#972** filed — `blocked`, `kind/gap`, `area/parser`, `area/conformance`.
- **#968** closed `state_reason: not_planned`, the reason **#493** asks
  `docs/WORKFLOW.md` to name and **the first park in this repo to use it** —
  #256 and #271 are both closed `completed` and remain #493's to correct.
- **#493**'s body gained a fourth Acceptance bullet: **the same sentence that
  names the close reason says to clear `ready`.** #968's session had to
  discover that by hand — `wipsurvey` reads the branch namespace and says
  nothing about an issue's queue labels, so a parked issue carrying `ready` is
  pickable while the band still lists it at its old rank, which is exactly where
  #968 sat. A comment would have left the next park to rediscover it. Also
  recorded there: **#287 is a THIRD park**, closed `completed` with the
  `needs-replan` label since removed, so `is:closed label:needs-replan` finds
  #256 and #271 and misses it entirely.
- **#786**'s body carried a premise this re-plan falsified. It read *"DELETION
  IS OFF THE TABLE UNTIL #968 IS SETTLED — … the recursion demonstrably buys
  something the moment that arm consults `facetElement`."* The arm is not going
  to consult `facetElement`: #972 fixes this at the producer and leaves the gate
  alone. Both passages rewritten, the defect's description kept (it is still
  true of the tree) and repointed, and the "whichever lands second reads the
  other" instruction retired — the two issues no longer share a file.
- **#561**'s deferral is discharged and its collision named. Its Notes ask that
  the skip-versus-rejection question be left until someone grounds it; the
  oracle grounded it on #968, and #972 asks it. #561's half A asserts `err ==
  nil` for the plural-`assertions` repro and #972 makes that repro an error, so
  whichever lands second rewrites that one assertion — and the pin survives
  either order, since `facetKindOf`'s exclusion becomes load-bearing for the
  REJECTION rather than belt-and-suspenders against a panic. Body untouched:
  nothing in it is stale, and #561 is very likely to land first.
- **#852** gained the mirror of its own defect, reproduced (below).

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed the paginated issue JSON, 578 issues):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
822    wip/issue-822  170h44m0s  RETIRED  wip/issue-822: issue #822 is closed
862    wip/issue-862  main's     CLAIMED  wip/issue-862: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  136h46m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  64h59m0s   RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  4h4m0s     RETIRED  wip/issue-968: issue #968 is closed
```

`git ls-remote --heads origin` returns exactly `main` and those five `wip/*`
refs, re-read this pass. **`wip/issue-968` is the new row and it changed reason
mid-pass**: it read `RETIRED … labelled needs-replan` before #968 closed and
reads `… is closed` after, which is the same verdict reached by the second of
the two conditions WORKFLOW's branch scheme states. It sits at `5304863` with
**no commit of its own** — `main`'s previous tip — so its "4h4m" is that
landing's age, not a claim's, and mason's diff exists nowhere but the issue
thread. Nothing is EXPIRED, no `parked/*` ref exists, and the other four rows
are unchanged in verdict.

**`wip/issue-862` is still a LIVE empty claim and its clock keeps running.** Its
thread's last comment is the GROUNDING of **2026-08-20T20:20Z** — **~62 hours as
this stamp was written**, up from ~57 at the previous stamp and well past the
~2-day threshold #867's takeover used and that **no document states**. That rule
is **#946**'s to settle and #946 is `blocked` on the next `/retro`; until then
#862 is off-limits by the same judgment the previous five stamps applied.
**`wip/issue-822`, `wip/issue-872` and `wip/issue-933` are RETIRED and kept**,
superseded by #851 and #878 and by #862's duplicate ruling. All five deletions
are a human's call.

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON — 138
issues, 50 of them open):

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

**The marker census did not move — no `.go` file changed — and Group 2 went 10 →
9, which is an ARTEFACT and not a disposition.** #968 left the group by closing;
**#972 should have replaced it and did not**. #972 is a `kind/gap` issue over a
producer/lane gap that carries no `GAP(` marker and never will, exactly like
#591, #592 and #593 beside it. It is absent because the matcher's **file-path
signal fired**: #972's Acceptance names `parser/produce_complex.go` in order to
say that `restrictionFacets`' other caller must **not** get the check, and
`matches` reads a path mentioned for any reason at all as a tracking claim.
Reproduced in one command, `diff <(go tool gapaudit < /dev/null) <(go tool
gapaudit < only972.json)`: four markers leave Group 1 —
`parser/produce_complex.go:469`, `:1913`, `:2189`, `:3002` — none of which has
anything to do with #972, and three of which cite a different issue.

**So "Group 1 is EMPTY", carried for eleven consecutive stamps, is now
qualified rather than repeated.** `parser/produce_complex.go:1913`'s own text
reads *"Unowned: no issue tracks it yet"*, and after today's filing the tool
reports it tracked. Group 1 emptiness is not evidence that every marker has an
owner: **any issue body citing a busy file can empty a row of it.** Recorded on
**#852** — which already owns the matcher's opposite defect, the ignored `(#N)`
citation — as a case of the same `matches` function rather than a sibling issue,
so neither direction rebases onto the other. **#960** still owns the class the
census structurally cannot see: a fail-open disclosed in PROSE carries no `GAP(`
marker, so it appears in neither group.

### Milestones and queue

Milestones, read from `repos/kud360/goxsd8/milestones` this pass and
cross-checked against the paginated issue list, which agrees exactly.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 97 | 45 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**Not one milestone row moved**, because #968 carried no milestone and #972 was
filed without one, following its predecessor rather than inventing a
classification for it. **173 of the 230 open issues carry no milestone** (230 −
45 − 13), so the milestone rows are feature progress and the paragraph below is
the queue.

Queue: **230 open — 202 `ready`, 28 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 28), against **348 closed**.
202 + 28 = 230 exactly, and **every one of the 230 carries a queue label** — the
class #773/#774 fell into is empty for the eleventh consecutive stamp. Both
figures were re-derived by paginating the issue list (page-numbered, not
`--paginate`, whose Link header uses numeric-ID URLs the proxy blocks), raising
the page count until a page came back empty, and discarding pull requests, which
share the endpoint. **Every move reconciles to this pass**: closed 347 → 348 is
#968 alone; open 230 → 230 is #968 closing against #972 filing; `ready` 203 →
202 is #968's park clearing the label, one landing before this pass ran;
`blocked` 27 → 28 is #972, filed `blocked` on #732; `needs-replan` 1 → 0 is #968
closing.

**The unblock sweep moved nothing, and this time the reason is structural rather
than measured.** #968 is the only issue closed this window and it closed as
SUPERSEDED, not landed — a dependency on it would not be satisfied by its
closing, it would be **misdirected**, and relabelling anything `ready` on that
basis would be the sweep's worst failure mode. All 230 open bodies were fetched
over `gh api` — byte-faithful, where MCP `issue_read` is lossy (#764) — and
searched for `#968` and `#971`: **`#971` appears nowhere**, and `#968`'s single
hit is **#786**, which cites it as an argument and not as a dependency, and whose
body is corrected above for that very reason. **No `blocked` issue names #968 in
its `## Depends on` at all.** No label changed and no `## Depends on` was
repaired.

**No duplicate was closed.** #972's filing search ran over all 230 open bodies
for `restrictionFacets`, `facetKindOf`, `simpleTypeDecidable`, `vc:minVersion`
and "conditional inclusion"; seven hits (#968, #561, #786, #846, #966, #731,
#732), every one cross-referenced in #972's Notes, none a duplicate. **#593 was
checked by name and is NOT the `vc:` owner** — it is `readFacetsCase`'s
root-attribute shape, adjacent only through the word "conditional".

**No persona stories were folded, because none were handed to this pass.** A
post-land pass is not a consultation: the cartographer has read the source, and
a persona it role-played itself would launder an insider's opinion as an
outsider's (#416). The 2026-08-22 `/backlog` consultation's findings stand as
that stamp recorded them, and rows 13 and 14 of the band below carry them
unchanged.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the
whole of it. Take from the top. **Row 1 was #966 and it LANDED** (PR #974,
`ebf6fa2`), so the band is re-cut from the head down: **#884** to row 1 and
every row below shifted up by one. Cross-references are stated by ISSUE rather
than by row number, which decays at each re-cut.

**The shift is the ONLY thing this pass re-derived.** Every row's argument is
carried verbatim from the previous stamp and was not re-checked — with one
exception, at the new row 1: #884's row said *"sequence it AFTER row 1"*, naming
a dependency #966's landing has discharged, and left verbatim it would read as
an instruction to sequence the head after itself. That sentence is replaced by
what the ruling settled; the rest of the row is untouched.

**The unblock sweep moved nothing.** All 229 open bodies were searched for the
literal `#966`; the single hit is **#972**, whose `## Depends on` names #732
alone and whose Notes call #966 *"not a hard dependency"*. No `## Depends on` in
the queue names #966, and no label changed.

**#975 is filed and deliberately unbanded.** It owns the nine s4s-grammar
rejection messages that name no Appendix A production — the bar `xsderr/doc.go`
states as of #966 — and it is cheap, mechanical and moves no lane. Ranking it
needs a survey a post-land pass does not run; the next `/backlog` places it.

**#972 is deliberately NOT in the band**, and the reason is the band's own
contract: it is the top of the **`ready`** queue, and #972 is `blocked` on #732.
Banding a blocked issue is how a `/develop` session ends up picking one.

| # | Issue | Why here |
|---:|---|---|
| 1 | #884 | **#950's own adjacent shape, named in its body as *"the adjacent shape and closes nothing here"*.** Every malformed named `<group>` body (a nested `group ref`, an `<element>`, annotation-only, empty) collapses into ONE `mgd-props-correct` message naming an internal invariant, located at the definition rather than the fault; all four verdicts are reproduced in the body against a named tip. **The ordering that held it at row 2 is DISCHARGED and its Spec section is now CONFIRMED rather than merely unchallenged**: that section concludes the fault carries no rule ID and is a plain `fmt.Errorf` on §5.1's first bullet, and #966 (`ebf6fa2`) has ruled exactly that footing correct — so this session implements a stated answer instead of inheriting a precedent, and `xsderr/doc.go` gives it the message shape to copy. See #884's thread |
| 2 | #732 | **NEW to the band, and it stopped being one issue's floor-of-two-cases the moment #968 was measured.** §4.2.2 `vc:minVersion`/`vc:maxVersion` conditional inclusion is unimplemented, and it is the direction CLAUDE.md's conformance stance treats as never acceptable: **a FALSE REJECT** of two suite-valid schemas (`VC/vc007`, `VC/vc_003_1`), both reproduced in its body by running `parser.Parse`. Three things it now carries that its body did not: it **gates #972**; four MORE cases (`VC/vc003`–`vc006`) are banked `pass` today only through a double negative — the pruning unimplemented AND the producer admitting the child it should have pruned — so they are correct for a reason that expires the moment either half is fixed alone; and its own Acceptance already says the two named cases are *"the floor, not the extent"*, with `saxonData/VC/` holding ~30 fixtures. **Unsized and it says so** — its body asks for an oracle grounding first (document-transform versus construction-time skip, composition with `<override>`'s §F.2 and `<include>`'s §F.1, an ill-formed version value), which is why it is ranked below two measured one-session rows and not above them. Evidence: <https://github.com/kud360/goxsd8/issues/732#issuecomment-5385424295> |
| 3 | #963 | **The tax falls on every landing and it has two witnessed failures; its own discriminator did NOT fire this window and the row says so rather than promoting it quietly.** #820 landed the *form* of landing precondition 1; nothing checks that the check was run, which is **#924**'s shape and is not reachable by prose. One session's work — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part, since #304 struck `logguard` and made CLAUDE.md's Commands block the sole gate definition. **The discriminator could not fire this window: no code landed**, so the only landing (PR #971) is a `docs/LOG` append and carries its entry by definition. Three clean landings plus one vacuous one is still weak evidence that prose suffices and no evidence that #924 decays |
| 4 | #493 | **NEW to the band, on CLAUDE.md's cost rule rather than on lane movement: this window PAID its friction, by hand, and the paying was itself an unlanded judgment call.** `docs/WORKFLOW.md`'s park step names neither the close reason nor the `ready` clear, so #968's session had to discover that a parked issue carrying `ready` is pickable — `wipsurvey` reads the branch namespace and says nothing about queue labels — and clear it without a document to cite. Its body now carries that as a fourth Acceptance bullet plus a **third** park (#287, closed `completed` with `needs-replan` since removed, invisible to the search that finds #256 and #271). One session, doc-only, and it targets the **park** paragraph — not the filing-discipline paragraph #510, #646, #679 and #912 all target — so it rebases against none of them |
| 5 | #846 | **#909 paid the shadow tax and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`.** Its promotion rule asks for a third landing that pays the tax and **no landing occurred this window, so the discriminator did not fire.** It did gain evidence of a different kind, and the row records the distinction rather than blurring it: #968's family A is the first MEASURED demonstration that the shadow model cannot reach the right answer at all — the fault deciding each of those four documents lives outside `<restriction>`'s child list, so no predicate over that list can tell a fabricated verdict from a correct one. That is structural, where the standing discriminator counts upkeep. On the thread; still a ~700-line refactor with no evidence it fits one session |
| 6 | #941 | **#387's own unfiled debt, and the coldest files of any band row** (`07117dc`, 2026-08-21). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete and `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 7 | #953 | **#924's other post-land filing, and a doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says "every caller turns a decided negative into a schema rejection"; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules |
| 8 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 9 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 10 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced; its premise is a candidate for expiry now that #447 and #738 landed `list` and `union`. **Its body was corrected this pass and the correction changes how to read this row**: deletion was held off the table until #968 settled, #968 settled the other way, and the arm is not going to consult `facetElement` after all — so the recursion buys nothing on that count, and once #972 lands an admissive gate here can no longer fabricate a verdict, which is the strongest argument this issue has ever had for its deletion branch. Still prefer the godoc answer; #868's diff is the place to start |
| 11 | #907 | **A repo tool catalogues every `childElement` sibling-alternative site and reports which is unguarded** — hand-written guards accumulated over months, each after a suite case tripped over it. `kind/tooling`, banded below the rows above because the tax was paid over months rather than in consecutive sessions. **Its census is stale by at least FOUR landings and is NOT re-derived here**: #909 rewrote 418 lines of `produce_complex.go`, #957 moved `produce_typetable.go`, and #956 added `produce_s4sorder.go` outright — re-run the census before designing from the body's figures |
| 12 | #669 → #625 → #748 → #896 → #492 (+ #934) | **README's Library block, ONE row in the order the issues name; splitting it is why it sat.** 2026-08-22 libuser reconfirmed all six and filed nothing: **#669** the "works TODAY" snippet still fails to compile; **#625** still points at closed #203 while `xsd.Example_buildFinalizeQuery` exists and passes; **#748** still says shipped `validate.New`/`xmlsrc.Validate` "do not exist yet"; **#896** the package "Contract" prose still never names `Err()`; **#492** README omits `ParseReport`/`AssemblyReport`/`ReadDocument`/`Produce`; **#934** the violation example still shows `[cvc-datatype-valid]` where #913/#914 now charge `[cvc-type]`. Citations were re-checked at the 2026-08-22 stamp against `79b0bd8` and are **carried, not re-run, here** |
| 13 | #870 + #747 + #514 + #687 + #672 | **The CLI is unreachable from its own documentation; 2026-08-22 cliuser reconfirmed all five and filed nothing — fourth consecutive such verdict, so the gap is disclosure not discovery.** **#687 gained two behaviours and its body a third Acceptance question**: `goxsd8 -xyz -help` prints full help and exits **0**, swallowing the bogus flag, and `-help=true` is NOT recognized and falls to the stub with exit 2 — both following from `wantsHelp` being a raw token scan. **#870** Quickstart's `go build ./...` writes no executable; **#747** `-help` is a strict subset of `go doc`; **#514** every non-help input is byte-identical stderr plus exit 2; **#672** `-version` in any spelling hits the stub |
| 14 | #885 | **The scope rule that has cost a reject round TWICE and still survives only in verdict comments.** #876 and #904 each shipped half a Goal because an Acceptance exemption was read as a scope exemption; #374's finding 1 added a third datum. Three discriminators, one sighting each — the body says if the third will not fit the sentence, state the two-datum rule and why |

**Deliberately unbanded, and why.** **#972** is `blocked` on #732 and belongs
nowhere in a `ready` band; when #732 lands it enters at the rank #968 held, and
for a better reason than #968 had. **#561** is `ready`, one test and one
sentence, and it now has a live sequencing relation with #972 rather than a
standing one — it stays unbanded because it changes no behaviour and blocks
nothing, but a session taking it should read #972's Notes first. **#409** is
`ready` since 2026-08-02 with a **third** independent sighting (the 2026-08-02
steward audit that filed it, and the 2026-08-11 and 2026-08-22 libuser passes,
both reaching it from the published surface alone) — `codegen/doc.go` prints
`Generate` and `Target` in a code block while the package exports nothing. It
stays unbanded only because it is one row of a five-file convention landing.
**#937** says in its own body that it is naturally folded by the next landing
touching `rejectRepeatedAnnotations`. **#920** and **#921** are conformance-
bookkeeping follow-ups below the fold. **#929** and **#931** are the small parser
occurrence / rule-mapping gaps #901 exposed; read each beside #901's thread.
**#455** is the live owner of the `strings.TrimSpace`-versus-§4.3.6 character
class at **ten** sites — unbanded because it is a pure false-accept narrowing
with a provably flat ratchet, and **#456** stays `blocked` on it. **#862** is
`ready` and its grounding is banked, but its branch is a LIVE empty claim whose
clock has now run ~62 hours past its last comment — off-limits until #946 rules,
and it is the worked example #946 asks for. **#888**, **#889**, **#894** are the
three `area/xpath` gaps still awaiting a suite census in their range. **#843–
#849** are the 2026-08-16 audit's findings, **six open**, of which **#843** has
the steepest cost of delay and **#846** is banded above. **#566** is #565's open
sibling, routed nowhere by #565's landing and correctly so. **#871** stays
`blocked` on #831. **#852** gained a second, opposite matcher defect this pass
and stays below the fold because the tool again ran with reconciliation — but
its Group 1 result is now qualified rather than clean, which is a change in kind.
**#881**, **#548**, **#622**, **#681**, **#692**, **#696**, **#796**, **#841**,
**#925**, **#946**, **#960** are `blocked` on the next `/retro` (or a ruling),
not on any landing — **ELEVEN of the 28**, carried from the previous stamp's
re-derivation and not re-parsed here, since no `## Depends on` changed and the
only new `blocked` issue (#972) names #732. **#570** carries the standing
`schema` decline-count argument at 893; the previous stamp measured **788** at
`ea00f84` and mason re-measured **788** on `origin/main` this window — the third
consecutive reading of that figure, and the 823 in #968's account is the
*narrowed* count of a branch that was never committed, not a movement. **#925**
was checked by name this pass and is NOT the owner of the park convention; it
owns the other-copies-of-a-corrected-claim question, and the park paragraph is
#493's.

### Next planning action

**Take from the top: start at #884.** It is a code session, one file, and it
inherits nothing: #966's landing (`ebf6fa2`) settled the footing #884's Spec
section had reasoned its way to on its own, so the session implements a stated
ruling rather than adding a site to a precedent. `xsderr/doc.go`'s s4s-grammar
section gives it the message shape — offending item, location, the Appendix A
production — and `rejectProhibitedAttrs`/`checkS4SChildOrder` are the exemplars
to copy. All four malformed-body verdicts are reproduced in its body against a
named tip, so the session starts from a measurement rather than a claim.

**#732 is the row to take if a code session is wanted, and it is the one whose
value changed most this window.** It went from "the two false rejects #442
exposed" to the gate on #972, with four more cases (`VC/vc003`–`vc006`) shown to
be passing today only because the mechanism is unimplemented. It is also the
band's only unsized row: **its first step is an oracle grounding, not code**, and
its own body lists the three questions to ask. A session that takes it and
returns only a grounding comment has done the right amount.

**The band's one standing ordering is DISCHARGED: #884-after-#966 is satisfied,
not carried forward.** #966 landed, and #884's thread records what the ruling
settled for it. **#972's preference for landing after #966 is discharged by the
same landing** — it stays `blocked` on #732 alone, which is the only reason it
is out of the band.

**#975 was filed this pass and is deliberately unbanded.** It owns the nine
s4s-grammar rejection messages that name no Appendix A production, censused at
`ebf6fa2` with the criterion that re-derives the set stated ahead of the list.
It is a cleanup that moves no lane, and one of its nine sites
(`produceModelGroupDefinition`) sits beside #884's in the same file — so #884
first, #975 whenever a cheap session wants it. **#444 already owns the tenth
deficient site** (`rejectBothInlineTypes`) on a stronger footing and is excluded
from #975 rather than duplicated by it.

**Both standing promotion discriminators were checked and NEITHER COULD FIRE
this window, which is a different statement from "did not fire".** #963's asks
whether a landing carried its `docs/LOG` entry inside the squash, and #846's asks
whether a landing's entry recorded the shadow tax; **no code landed**, so both
questions had no subject. The next pass runs them against the next real landing,
not against PR #971.

**A survey result was qualified rather than repeated for the first time in
eleven stamps.** `gapaudit`'s Group 1 is empty, and this stamp says why that is
now weaker evidence than it reads: a file path cited in an issue body — even
cited to say "do not touch this site" — matches every marker in that file, and
#972's filing alone moved four `parser/produce_complex.go` markers out of Group
1, one of which says in its own text that nothing tracks it. **#852** owns the
fix and now owns both directions of it. The lesson for the next pass is narrow:
**Group 1 emptiness is not proof that every marker has an owner.**

**Four unlanded corrections still target one paragraph of `docs/WORKFLOW.md`'s
filing discipline** — **#510**, **#646**, **#679**, **#912** — and whichever
lands last rebases three times. **#493 is not a fifth**: it targets the park
paragraph, which none of the four touches, and it absorbed this window's process
finding instead of that finding becoming a sixth issue. **The next `/retro` has
ELEVEN `blocked` issues waiting on it** — #881, #548, #622, #681, #692, #696,
#796, #841, #925, #946, #960 — unchanged, with #946 (the branch-claim TTL, whose
worked example is #862 at ~62 hours) and #960 (prose-only gaps) still the two
that leave live questions unadjudicated today. **The CTA cohort's 45 banked
`instance` failures remain unattributed**, tenth consecutive stamp carrying it.
**`gate.yml` runs but is still not a required status check**, which only the
repository owner can change. All of these stay open and stay true.

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
the producer drops), both landed on 2026-08-22/23; live ones in the same
family are **#471** (a local `<element ref=>` carrying `substitutionGroup=`,
silently accepted), **#931** (occurrence attributes on a named `<group>`'s
child compositor), **#929**, **#455**, and **#972** (an XSD-namespace child
§4.1.2's `<simpleType><restriction>` has no position for, dropped by
`restrictionFacets` — `blocked` on #732, which owns the §4.2.2 conditional
inclusion the same site needs first). The GitHub milestone holds the feature
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
