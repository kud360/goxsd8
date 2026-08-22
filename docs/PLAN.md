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

## Status — 2026-08-22 (post-land pass for #820; one landing absorbed, lanes/milestones/queue and both surveys re-derived, the band re-cut and its head replaced, one follow-up filed)

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

**No lane moved, and that is measured rather than inferred.** The one landing in
this window touches no `.go` file, and the accepting verdict ran
`GOXSD_RATCHET=1 go test ./conformance` anyway, checking
`git status --porcelain -- conformance/testdata/expectations/` immediately
after: empty. The previous stamp's `schema` 12983 → 13076 (+93) and `instance`
10746 → 10752 (+6) are its attribution and are not restated here; `docs/LOG` is
history.

Landing absorbed by this stamp, one, at `e3d866f..311ada8`:

- **#820** (`311ada8`) — `docs/WORKFLOW.md`'s **Landing** section states the
  `docs/LOG` entry's due point **structurally**: `/develop` step 5 (Judge) runs
  before step 6 (Land), where the chronicler writes it, so an empty `docs/LOG/`
  diff at verdict time is the documented order and not a skipped step. Landing
  precondition 1 is **replaced, not supplemented** — the bare path-emptiness
  check becomes an issue-number grep over the branch's own **added** lines, with
  #813's forward merge named *inside* the rule so a later editor cannot simplify
  it back. `.claude/agents/arbiter.md` gains one line: a missing entry is never
  a finding. The pattern **nine landings recorded and none could route** — every
  one citing #528, closed since 2026-08-09, and not one citing #820. Arbiter
  ACCEPT in one round, zero findings. Doc-only, 2 files, +17/−3.
  `Ratchet: unchanged`, measured.

**This landing's follow-up ledger is disposed, and one hand-off in it was
broken.** The LOG entry's Next 3 routes the mechanization of precondition 1 to
*"#304 (`go tool logguard`), which owns that territory"* — **#304 is closed
(2026-08-07, PR #533) and its resolution was to STRIKE `logguard`, not build
it**, so that hand-off tracked nothing, which is #820's own leak one issue over.
The live owner is **#963**, filed by this pass: a `landcheck` tool for
precondition 1, explicitly not a gate part (#304's ruling that CLAUDE.md's
Commands block is the sole gate definition stands). Two further items — Friction
2's regex loosenings and Surprise 1's fetched-base residual — were **dismissed
as prose and absorbed into #963's acceptance**, and Next 4 ("the real test is
the next landing") names no deliverable and was not filed. Every disposition is
written out on the thread.

**One claim in #820's GROUNDING is false and is corrected on the thread rather
than left standing.** The ruling reads *"not one of the nine sightings shows the
entry missing at actual landing time; every one of them landed with its entry
present."* **#924 did not**: `git show --stat 53bf113` lists eight files and no
`docs/LOG/` path, and the entry arrived afterwards as `aeae89f` (one file,
+220). Reproduced from git by this pass. **Ruling (i) is undisturbed** — it is
about the diff at *verdict* time, and #924 is a failure at *landing* time — but
the "no such failure exists" argument is what made the mechanization look
unnecessary, and it is #963's lead evidence.

Milestones, read from `repos/kud360/goxsd8/milestones` this pass and
cross-checked against the paginated issue list, which agrees exactly.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 94 | 48 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**Every milestone row is unchanged from the previous stamp**, which is the
expected reading of a `kind/process` landing: #820 carried no milestone, as no
process issue does, and #963 carries none either. **170 of the 231 open issues
carry no milestone** (231 − 48 − 13), so the milestone rows are feature progress
and the queue paragraph below is the queue.

Queue: **231 open — 204 `ready`, 27 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 27), against **344 closed**.
204 + 27 = 231 exactly, and **every one of the 231 carries a queue label** — the
class #773/#774 fell into is empty for the ninth consecutive stamp. Both figures
were re-derived by paginating the issue list (page-numbered, not `--paginate`,
whose Link header uses numeric-ID URLs the proxy blocks), raising the page count
until a page came back empty, and discarding pull requests, which share the
endpoint. **The move reconciles exactly**: closed 343 → 344 is #820 and nothing
else; open 231 → 230 → 231 is #820 closing and **#963** filing, this pass's own
and only filing.

**The unblock sweep moved nothing, for the fifteenth consecutive pass, run as a
parse rather than by eye.** Every one of the 27 `blocked` bodies was fetched over
`gh api` — byte-faithful, where MCP `issue_read` is lossy (#764) — and its
`## Depends on` scanned for #820: **not one names it, and `#820` appears in no
open body or title at all.** Every live dependency line still names an issue that
is **open, re-checked by number this pass** — #831, #719, #472, #248, #591,
#414, #455, #407, #250, #79 — and the rest are triggers (the next `/retro`, a
steward drift review, an epic target, a ruling), several of which say in their
own text not to re-scan them. **No `## Depends on` was repaired**, and no
`blocked` issue is startable today.

**No body was rewritten and no duplicate was closed this pass.** The one body
that needed rewriting last stamp was #820's and it has since landed; **#963** was
checked against #735 (`commentonly`), #779 (`queueaudit`), #797 and #600 before
filing and duplicates none of them — three WORKFLOW rules that each name a
mechanical proof and ship no mechanism are three issues, and #963's body says so
rather than merging them.

**No persona stories were folded, because none were handed to this pass.** A
post-land pass is not a consultation: the cartographer has read the source and a
persona it role-played itself would launder an insider's opinion as an
outsider's (#416). The 2026-08-22 `/backlog` consultation's findings stand as
that stamp recorded them, and rows 12 and 13 of the band below carry them
unchanged.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed the paginated issue JSON):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
822    wip/issue-822  153h45m0s  RETIRED  wip/issue-822: issue #822 is closed
862    wip/issue-862  main's     CLAIMED  wip/issue-862: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  119h47m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  48h0m0s    RETIRED  wip/issue-933: issue #933 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-862`, `wip/issue-872` and `wip/issue-933`, re-read this pass — the
same five as the previous stamp. **`wip/issue-820` left no ref behind**,
GitHub's auto-delete having taken it at merge. Nothing is EXPIRED, no `parked/*`
ref exists, and the four rows are unchanged in verdict.

**`wip/issue-862` is still a LIVE empty claim and its clock keeps running.** It
sits at `c2ba631`, which is not `main`'s tip and is still not a commit of its
own, so it stays `ahead 0`, can never be EXPIRED, and can never be retired on
age (#722) — the "main's" in the TIP AGE column is the tool saying it has no
clock of its own to read, not a claim that the branch is current. Its thread's
last comment is the GROUNDING posted **2026-08-20T20:20Z** and nothing since —
**~45 hours as this stamp was written**, against the ~2-day threshold #867's
takeover used and that **no document states**. That rule is **#946**'s to settle
and #946 is `blocked` on the next `/retro`; until then #862 is off-limits by the
same judgment the previous three stamps applied, and the grounding remains the
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

**Both survey outputs are byte-identical to the previous stamp's, which is the
only correct result for a diff holding no `.go` file** — the raw `GAP(` token
count is **96** at both `9c10af8` and `311ada8`, the tool's 64 and its five-area
composition are unchanged, Group 1 is **EMPTY for the ninth consecutive stamp**,
and Group 2 is unchanged at 9. **#852** still owns the matcher qualification —
the raw-token count of 96 against the tool's 64 — and stays below the fold
because the tool again ran with reconciliation and Group 1 empty. **#960** still
owns the class the census structurally cannot see: a fail-open disclosed in
PROSE carries no `GAP(` marker, so it appears in neither the census nor Group 1,
and #957 is the worked example.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 204 of
them. Take from the top. **The previous band's head LANDED** (#820); the band is
re-cut by dropping it and re-deriving every cross-reference by ISSUE, never by
row number, which decays at each re-cut.

**The head is a lane row again, for the first time in three stamps.** #565 was
ranked first on 2026-08-21 and #820 on 2026-08-22, both on CLAUDE.md's rule that
a friction the log records in consecutive sessions outranks a lane slice, and
both landed within a day of being ranked. Neither streak has a successor: no
open process issue carries a comparable one, and the two warm lane rows below
were filed hours after the landing whose files they touch. **#963 is this
pass's process/tooling row and is deliberately NOT the head** — see row 4.

| # | Issue | Why here |
|---:|---|---|
| 1 | #956 | **Four false accepts, `schema` candidate, and `parser/produce_complex.go` is still the warmest file in the tree.** No complex-type derivation alternant enforces the s4s child ORDER or its `maxOccurs`, so `ctC011`/`ctD034`/`ctD042`/`ctD043` are decided-and-ACCEPTED at all four alternant sites. Filed by #909's post-land pass on the arbiter's own words ("worth an issue of its own"); the four cases were declines before #909 and are false accepts after, so **no banked expectation depends on it either way** and the floor is a correctness fix rather than a lane number |
| 2 | #958 | **#909's three remaining one-site defects, one landing, same warm files.** `facetElement` (`conformance/schema.go:1173`) admits the FACET name `assertions`, which the s4s grammar has no element for and the producer silently drops; a repointed test at `schema_test.go:484` still names the fixture shape it no longer uses; and case 5's inline-`<simpleType>` drop has no test. **Acceptance item 1 shares its seam with #561** — the gate side here, the untested producer-side exclusion there — cross-referenced on both threads, sequence rather than merge |
| 3 | #950 | **#932's own follow-up, filed by its post-land pass and specified by MECHANISM rather than by symptom.** `produceWildcard` calls `processContentsOf` (`produce_complex.go:2694`) **before** `xsd.NewWildcard`, so a lexical `processContents` fault is charged `w-props-correct` clause 1 — a constraint over a component that does not exist — which is exactly the two-layer split #932 corrected one layer up. Requires its own GROUNDING on whether §4.1.4 reaches an `xs:NMTOKEN` enumeration in Appendix A; **#884** is the adjacent shape and closes neither |
| 4 | #963 | **The tax falls on every landing, and it has two witnessed failures rather than nine narrated non-events — but it is ranked BELOW the warm lane rows and the reason is stated rather than hidden.** #820 landed the *form* of landing precondition 1 hours ago; nothing checks that the check was run, which is **#924**'s shape (`53bf113` squashed eight files and no `docs/LOG/` path; `aeae89f` carried the entry afterwards) and is not reachable by prose. It is one session's work — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part, since #304 struck `logguard` and made CLAUDE.md's Commands block the sole gate definition. Ranked here because the three rows above are measured and warm and this one's own prose sibling is hours old; **if a landing under the new prose misses precondition 1 again, it moves to the head** |
| 5 | #846 | **#909 paid the shadow tax once more and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`, correct only because the arbiter walked `attrDeclsDecidable` against `main` by hand.** Rows 1 and 2 will both pay it again. Ranked BELOW them and the tension is stated rather than hidden: landing this first would make them cheaper, but it is a ~700-line refactor with no evidence it fits one session, and two warm measured slices should not wait behind an unsized one. If it grows a third witness, it moves above them |
| 6 | #941 | **#387's own unfiled debt, and the coldest files of any band row** (`07117dc`, 2026-08-21 — the previous stamp's "four days back" was wrong and is not carried). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete (the field/method collision that forced #387's) and `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 7 | #953 | **#924's other post-land filing, and a doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says "every caller turns a decided negative into a schema rejection"; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules. Pre-existing on `main`, out of scope for #924 by its arbiter's own ruling, and discharged there only by a commit body naming it in prose |
| 8 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 9 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 10 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced; its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. **Its measured delta shrank in the previous window**: `notatF067` was one of its witnesses and #945 banked it, which is recorded on the thread, and the `restriction == nil` arm is untouched, so the measurement is still the thing to run. Read #868's diff first |
| 11 | #907 | **A repo tool catalogues every `childElement` sibling-alternative site and reports which is unguarded** — hand-written guards accumulated over months, each after a suite case tripped over it. `kind/tooling`, banded below the lane rows because the tax was paid over months rather than in consecutive sessions — the discriminator rows 1–4 satisfy and this one does not. **Its census is stale by at least two landings and is NOT re-derived here**: #909 rewrote 418 lines of `produce_complex.go` and #957 moved `produce_typetable.go`, so re-run the census before designing from the body's figures |
| 12 | #669 → #625 → #748 → #896 → #492 (+ #934) | **README's Library block, ONE row in the order the issues name; splitting it is why it sat.** 2026-08-22 libuser reconfirmed all six and filed nothing: **#669** the "works TODAY" snippet still fails to compile (`parser.WithLogger(logger)` names an identifier the block never declares, beside three unused-variable errors); **#625** still points at closed #203 while `xsd.Example_buildFinalizeQuery` exists and passes; **#748** still says shipped `validate.New`/`xmlsrc.Validate` "do not exist yet", with all three signature mismatches holding; **#896** the package "Contract" prose still never names `Err()`; **#492** README omits `ParseReport`/`AssemblyReport`/`ReadDocument`/`Produce` (grep: zero matches); **#934** the violation example still shows `[cvc-datatype-valid]` where #913/#914 now charge `[cvc-type]` with the old rule reachable only as the wrapped cause. `README.md` is unchanged since **`79b0bd8`** (2026-08-16) and every citation above was re-checked at the previous stamp, which cited `6cd5b34` — 21 minutes earlier the same day, and not the commit that last touched the file |
| 13 | #870 + #747 + #514 + #687 + #672 | **The CLI is unreachable from its own documentation; 2026-08-22 cliuser reconfirmed all five and filed nothing — fourth consecutive such verdict, so the gap is disclosure not discovery.** **#687 gained two behaviours and its body a third Acceptance question**, both reproduced at `9c10af8`: `goxsd8 -xyz -help` prints full help and exits **0**, silently swallowing the bogus flag, and `-help=true` — the stdlib boolean-flag idiom — is NOT recognized and falls to the stub with exit 2. Both follow from `wantsHelp` being a raw token scan with three exact string comparisons rather than flag parsing. **#870** Quickstart's `go build ./...` writes no executable and the stub's own `go doc` remedy fails outside the module root; **#747** `-help` is a strict subset of `go doc`; **#514** every non-help input is byte-identical stderr plus exit 2; **#672** `-version` in any spelling hits the stub |
| 14 | #885 | **The scope rule that has cost a reject round TWICE and still survives only in verdict comments.** #876 and #904 each shipped half a Goal because an Acceptance exemption was read as a scope exemption; #374's finding 1 added a third datum. Three discriminators, one sighting each — the body says if the third will not fit the sentence, state the two-datum rule and why |

**Deliberately unbanded, and why.** **#409** is `ready` since 2026-08-02 and now
carries its **third** independent sighting — the 2026-08-02 steward audit that
filed it, and the 2026-08-11 and 2026-08-22 libuser passes, both of which reached
it from the published surface alone with no knowledge that it existed —
`codegen/doc.go` prints `Generate` and `Target` in a code block while the package
exports nothing, and `#203`'s landed `xsd/doc.go:213` heading is the exact
spelling to copy. It stays unbanded only because it is one row of a five-file
convention landing and no session has been blocked by it. **#937** is correct and
`ready` but says in its own body that it is naturally folded by the next landing
touching `rejectRepeatedAnnotations`. **#920** and **#921** are conformance-
bookkeeping follow-ups below the fold. **#929** and **#931** are the small parser
occurrence / rule-mapping gaps #901 exposed (#932 took the third); read each
beside #901's thread. **#862** is `ready` and its grounding is banked, but its
branch is a LIVE empty claim whose clock has now run ~45 hours past its last
comment — off-limits until #946 rules, and it is the worked example #946 asks
for. **#888**, **#889**, **#894** are the three `area/xpath` gaps still awaiting
a suite census in their range (#889 states a warden pre-flight per #484).
**#843–#849** are the 2026-08-16 audit's findings, **six open** — #847 closed
`not_planned` on 2026-08-17 — of which **#843** has the steepest cost of delay
and **#846** is banded above. **#566** is #565's open sibling, routed nowhere by
#565's landing and correctly so. **#871** stays `blocked` on #831. **#881**,
**#548**, **#622**, **#692**, **#696**, **#796**, **#841**, **#925**, **#946**,
**#960** are `blocked` on the next `/retro` (or a ruling), not on any landing —
**ten of the 27**, unchanged this pass. **#570** carries the standing `schema`
decline-count argument at 893, and that figure predates three lane-moving
landings including #909's +80. **This pass measured the current one rather than
carrying the stale one**: gate part 4 at `311ada8` reports *"lane schema: 788
declined case(s) recorded fail"* plus 11 more declined as indeterminate (#277).
Whether 893 counted the same quantity is #570's to confirm — but it starts from
788, not from a number three landings old.

### Next planning action

**Take from the top: start at #956**, and take #958 with it or straight after —
both were filed by #909's post-land pass against `parser/produce_complex.go` and
`conformance/schema.go`, both are one-session slices, and the five-case §3.4.2.2
tableau they read is still legible. **#950 is the third**, #932's own
mechanism-scoped follow-up, and it needs a grounding before code. The two stamps
before this one both put a process row at the head and both landed within a day;
**there is no process row with that argument today**, and ranking one anyway
would be the mirror of the failure CLAUDE.md's rule exists to prevent.

**#963 is the exception to watch, not to promote yet.** It is this pass's own
filing and its case rests on #924's squash landing with no `docs/LOG/` path —
one real failure, plus #813's, against a rule whose prose fix is hours old. **The
discriminator is written into row 4**: the next landing that misses precondition
1 moves it to the head; a clean run leaves it where it is. Do not re-argue that
from scratch next pass — the evidence is #924's `53bf113` and it does not decay.

**The next `/retro` has ten `blocked` issues waiting on it, and the number is
RISING** — #881, #548, #622, #692, #696, #796, #841, #925, #946, #960; it was
nine at the previous stamp, which added #960, and nothing has left the list. Two of them (#946, #960) are the reason live branches and prose-only
gaps go unadjudicated today. A retro that rules on nothing else should still rule
on those two.

**Four unlanded corrections still target one paragraph of `docs/WORKFLOW.md`'s
filing discipline** — **#510**, **#646**, **#679**, **#912** — and whichever
lands last rebases three times. Decide one issue or four before filing a fifth;
note that **#963 also edits `docs/WORKFLOW.md`**, in the Landing section rather
than that paragraph, so it does not join the pile-up. **The CTA cohort's 45
banked `instance` failures remain unattributed**, eighth consecutive stamp
carrying it. **`gate.yml` runs but is still not a required status check**, which
only the repository owner can change. All three stay open and stay true.

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
