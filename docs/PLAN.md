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

## Status — 2026-08-28 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin main` at `032d402`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh 631-issue page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **SEVEN** landings, found **FOUR of the last band's twelve rows already cleared**, and re-derived the whole ordering rather than shifting it. It found **the lane table moving again after four flat stamps**, found **all five of #1023's issues discharged in the tree**, found **an empty branch namespace for the first time in five stamps**, filed **#1088** and **#1089**, corrected **#625**, and folded the **tenth consecutive** persona consultation)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10775 | 15586 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13332 | 2066 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### The table moved, and the last three stamps' diagnosis is answered

**`schema` 13259 → 13332 (+73) and `instance` 10760 → 10775 (+15)** — 88
verdicts, the first movement in four stamps, and the end of a
**thirteen-landing `Ratchet: unchanged` streak**. Three consecutive landings
each moved a lane:

| landing | commit | expectations diff | lane movement |
|---|---|---|---|
| #1047 | `57ad014` | `schema.txt` 34±34 | `schema` **+34** |
| #1048 | `fc58dc4` | `schema.txt` 16±16 | `schema` **+16** |
| #1046 | `b14158c` | `schema.txt` 23±23, `instance.txt` 15±15 | `schema` **+23**, `instance` **+15** |

`git diff --stat b5a45a6 032d402 -- conformance/testdata/expectations/` totals
88 insertions and 88 deletions across the two files, which is the paste above.

**The last stamp set a trigger and it did not fire, which is the useful
result.** It said: *"If a third consecutive stamp publishes a flat table after
those two land, the diagnosis is wrong and the thing to re-examine is the
measurement, not the queue."* The table is not flat. The diagnosis — that
seam work is lane-flat by construction and only a landing that *decides a new
rule on a shape the engine was declining* moves a lane — held, and the
instrument that produced the candidates (#1030's stage-1 unmapped-construct
census) is now the only instrument in this project with a track record.

**This is also the first time the band's own predictions can be scored, and
they are good but not exact.** #1047's body promised 52 suite documents and
delivered 34; #1048 promised 16 and delivered 16; #1046 promised 31 and
delivered 38 across two lanes. So a measured document count is a **usable but
biased** estimator — it bounds the shape, not the verdict count, because one
document can carry several cases and a producer rejection can reach documents
the census could not name. **Band on measured counts; do not quote one as a
promised figure in a `Ratchet:` line.**

**The other four landings were lane-flat and declared it.** #941 (`72e40a5`,
deleted `Element.Attributes`/`Element.BaseURI`), #953 (`a3373c7`, enumerated
`ValueSpace`'s fail-open readers), #975 (`1dcffbf`, every s4s-grammar rejection
names its Appendix A production) and #1052 (`032d402`, the `## Acceptance`
satisfiability ruling) are seam, doc and process work.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin main`
at `032d402` and this pass's 631-issue fetch:

```
ISSUE  BRANCH         LEASE AGE  VERDICT  REASON
732    wip/issue-732  unknown    RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  unknown    RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846  unknown    RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872  unknown    RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  unknown    RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  unknown    RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993  unknown    RETIRED  wip/issue-993: issue #993 is closed
```

**Zero LIVE, zero CLAIMED, zero `parked/*` — the namespace is entirely
startable, for the first time in five stamps.** `wip/issue-953` and
`wip/issue-975` left by landing; nothing arrived. `git ls-remote --heads
origin` returns `main` plus exactly these seven refs and nothing else.

**All seven RETIRED refs are correct and none owes a supersede.** Every one of
the seven issues closed **`not_planned`** — #732, #822, #846, #872, #933, #968,
#993, checked through `state_reason` on this pass's fetch — so they are parks
and supersedes, not landings, and their content is *supposed* not to be in
`main`. A `not_planned` close names its replacement (#493's rule); #846's, for
instance, is #1029 + #1030, both landed. Cloud containers cannot delete remote
refs, so these are retired in place and will accumulate; that is by design and
is not a finding.

**The shallow-clone premise bit again and is worth one line.** This container
sees **50 commits** of `origin/main`, back to 2026-08-24, so
`git log --grep="(#N)"` cannot reach any of the seven closures and answers
empty rather than "not found". The disposition above was taken from GitHub's
`state_reason`, never from `git log`. **#802** owns this and is open.

### Marker census

`go tool gapaudit` over 143 open `kind/gap` issues: **70 markers across 6
areas** — `xsd` 38, `validate` 16, `xpath` 6, `xml` 4, `parser` 3, `value` 3.

**Group 1 printed `(none)`, and that is FALSE for the fifth consecutive
stamp.** Four markers in `validate/` name no issue at all, each read from the
tree by this pass: `validate/cvccomplexcontent.go:445`, `validate/cvcid.go:232`,
`validate/cvcidentityconstraint.go:390` and `validate/assess.go:874`. Three of
the four are the cvc-elt clause 5.1 decline **#853** owns; the fourth is
`assess.go`'s untyped-child decline. `anyOpenMatch` still uses
`matchKind.found()`, so a five-word phrase run still empties a group-1 row —
#852 raised group 2's bar only, deliberately. **#1060** is the fix and is band
row 4.

Group 2 lists eight open trackers with no surviving marker, unchanged from the
last stamp: #398, #404, #591, #592, #593, #731, #787, #921. Three carry the
phrase-match annotation #852 added, which is working as designed — a row that
prints its collision is a row the reader can adjudicate.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes: **631 issues, 240 open, 391 closed**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **48** | **102** | active |
| **M5 — Instance validation (XML)** | **12** | **17** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **218 `ready`, 22 `blocked`** — every open issue
carries one or the other, and the two sum to 240 with no gap. By kind:
`kind/refactor` 68, `kind/gap` 51, `kind/process` 47, `kind/tooling` 27,
`kind/bug` 25, `kind/story` 19, `kind/feature` 7, `epic` 2. By area:
`parser` 69, `xsd` 60, `meta` 58, `conformance` 28, `validate` 25, `docs` 18,
`value` 14, `builtin` 9, `cmd` 9, `xpath` 7.

**M4's count is unchanged at 48 and the stability is misleading.** Four M4
issues closed this window (#975, #1046, #1047, #1048) and four opened
(#1073, #1076, #1078, #1082) — three of the four new ones filed by the post-land
passes of the three that moved lanes. **The census family replenishes itself as
it is worked**, which is what a working measurement instrument looks like and
not tail growth.

**`ready` overstates startable work by five.** #625, #748, #492, #934 and #896
are all discharged in `main` and all still open and `ready` — see the next
section. The honest startable count is **213**.

### #1023's five are DISCHARGED — verified in the tree, seventh stamp carrying them

All five were re-read against `032d402` by this pass rather than against
`34a8043`'s diff, and the evidence table is on
[#1023](https://github.com/kud360/goxsd8/issues/1023#issuecomment-5453728234)
with a STOP pointer now standing on each of the five threads:

| issue | verified |
|---|---|
| #625 | `README.md:175-176` carries the PRODUCER-surface caveat; `grep "issues/203\|#203" README.md` → no match; `:181` names `Example_buildFinalizeQuery` |
| #748 | `README.md:184-185` — *"Instance validation runs today"*; `:196` calls the real `validate.New(schema, backend)` |
| #492 | `README.md:158` carries `ParseReport`'s full signature; `:169` explains what it lifts |
| #934 | `README.md:91` prints `[cvc-type]` as the outer rule wrapping `[cvc-datatype-valid]`; `:95` states it in prose |
| #896 | `validate/doc.go:85` — *"Read the verdict off [Result.Violations]; [Result.Err] reports whether the …"* |

**A cartographer cannot close them.** `.claude/agents/cartographer.md` reserves
closing-as-done to the develop loop, and these are done rather than obsolete, so
the `not_planned` route is not available either. Five `-f state=closed -f
state_reason=completed` calls from any session discharge #1023's visible half.

### Persona consultations — the TENTH consecutive

Handed to this pass by the orchestrating session; the cartographer role-plays
no persona and verified every claim below against the tree before recording it.
**Two filings, six threads updated, three persona claims corrected.**

**Filed.** **#1088** — `validate` ships **zero** runnable `Example` funcs
(`validate/xmlsrc` likewise) and `validate/doc.go`'s 236 lines carry no code
block, so plain `go doc ./validate` prints no compilable line; the only working
snippet is README's, and `README.md:225-228` itself says `go doc` will not
surface the alternative. **#1089** — `goxsd8 -help` names no route to
`doc.go`'s `# Argument vocabulary` section, so an installed binary cannot reach
it; the ERROR path prints the repo URL under every diagnosis
(`main.go:59`) and the SUCCESS path prints nothing —
`goxsd8 -help | grep -i "go doc\|github"` returns empty.

**Reconfirmed, with the increment recorded on the thread and no re-scope.**
**#409** (seventh sighting, two personas) gains an in-module precedent: `gen`
— `codegen`'s own CLI face — states *"Implemented today: the help path only"*
and exits 2, while `codegen` states a present-tense contract and exports zero
symbols. **#1066**'s three holes are byte-unchanged. **#1031**'s `-format`
vocabulary is unchanged, and the synopsis-completeness observation beside it
folds into its existing first bullet, leaving only a `-no-hints` judgment call.
**#1033** is unchanged: `builtin/doc.go:44` still says `~25` against five sites
saying 20.

**Three persona claims were wrong and are corrected on the threads, which is
why the verification step exists.** (1) libuser recommended **closing #1006**;
the recommendation answers the acceptance bullet that issue's 2026-08-27
re-scope had already struck as unsatisfiable, and all three re-scoped items
verify unmet — `backendtest.Run` is called once, on the module's own backend;
zero capability interfaces appear in either example file; README's curated list
still omits `value/backendtest/example_test.go`. **Not closed.** A new datum
went on the thread: README describes its example set as *"seed builtins → parse
a lexical → assert capabilities"* and **no example asserts a capability**, so
items 2 and 3 are one defect from two ends. (2) cliuser reported the `-schema`
synopsis as *"three byte-identical copies"*; there are **two** (`doc.go:11`,
`main.go:31`), and `README.md:67` is a worked example, which changes what the
fix edits. (3) libuser placed `SchemaBuilder`'s INTENDED CALLER caveat at
*"paragraph 4"*; it is paragraph **2 of 3**, and the request to hoist it above
the sentence naming what the type is was **dismissed on #625's thread** with
reasons rather than filed.

**#625's body was corrected** — it had claimed since filing that the caveat is
*"the first paragraph a reader of `go doc ./xsd SchemaBuilder` sees"*, which
`xsd/schema.go:17-24` falsifies.

### Working band

**Re-derived from this pass's evidence; not shifted.** Four of the last band's
twelve rows landed (#1047, #1048, #1046, #1052) and are gone. Take from the top;
re-run `wipsurvey` first, though the namespace is currently empty.

| # | issue | why here |
|---|---|---|
| 1 | #1076 | **The successor to the mechanism that just moved three lanes in a row, and the highest-confidence lane candidate in the queue.** `xs:element`, `xs:attribute` and `xs:simpleType`'s own children are checked against **no** s4s content model, so an out-of-model child is silently dropped — the same defect `checkS4SChildOrder` now charges under `xs:complexType`, at the sites #1047 proved separate from complexType's five. Every affected document is in the suite's INVALID corpus, so the direction can only be up. **UNMEASURED**: #1047's "9" was a hypothesis its body flagged as such, and this issue's Acceptance says `Ratchet:` must name a measured figure. Its site set is decided by a criterion rather than a list (#912), and the three Appendix A productions are the **grounding question, not a settled citation** — #1073's starting points are to be verified, not trusted |
| 2 | #1082 | **A FALSE REJECT reproducible on `main` today, and the landing three days old made it more reachable.** §3.4.2.4 builds `{attribute uses}` as a union of **sets**; `inheritAttributeUses` concatenates, so a complex type reaching one attribute group through both its own clause 2 fold and clause 3.1's inheritance carries the uses twice and `ct-props-correct` clause 4 rejects a schema declaring no duplicate. Two reproducing shapes, one needing no `defaultAttributes` at all. **#1046 (`b14158c`) widened it**: `<schema defaultAttributes>` synthesizes both folds, so nobody has to write a duplicate `ref` for the base and its extension to collide. Withdrawing a wrong rejection can only hold or raise `schema`, and a wrong decision outranks a decline. Grounding still required on the dedup criterion — §3.4.6.2 `cos-ct-extends` clause 1.2's recursive-identity reading is a candidate, not the answer |
| 3 | #1043 | **The band's `instance` row, and the lane is live again after four flat stamps.** `walk.attributeType` (`validate/cvcid.go:574`) falls through to `topLevelAttributeType` (`:586`) for every attribute matching no `{attribute use}`, with no wildcard check and no `{process contents}` check at all — so an attribute admitted by a ***skip*** wildcard binds an ·ID value· and `cvc-id` clause 2 charges a duplicate on a document §3.10.4.1's Note says has no ·governing· declaration. `walk.keyMember` has the identical exposure for a `@NameTest` field. **`xsd.ProcessContents` and `xsd.ProcessSkip` already exist and `validate` reads neither**; reading them is the whole change. Ratchet unchanged-or-upward: it can only WITHDRAW a charge |
| 4 | #1060 | **Banded on the sessions it costs, and this stamp paid it for the FIFTH consecutive time.** `gapaudit`'s group 1 printed `(none)` above across 70 markers while four markers in `validate/` cite no issue at all — all four re-read from the tree here. `anyOpenMatch` still uses `matchKind.found()`, so a five-word phrase run empties a group-1 row; #852 landed and raised group **2**'s bar only, deliberately. The second half is measured too: `matchFile` retires a tracker on a bare path mention, witnessed by an exclusion. **Both directions are one landing.** The cartographer runs this tool every pass and writes the caveat paragraph every pass; nothing else in the queue will ever lift that |
| 5 | #1062 | **The same clause, second consecutive stamp paying it, and this pass paid the cheaper half twice.** `docs/ROUTINES.md`'s survey-input recipe says `seq 1 9`; the repository needed **11** pages, and a short read is silent — `gh api --paginate` writes the pages it fetched and then exits non-zero, which a redirect into a survey tool does not notice. Its other half is why group 2 above cannot be adjudicated further: the input shape carries no labels, so a marker owned by an issue outside `kind/gap` can never resolve. Ranked below #1060 because #1060 makes the tool report something **false** where this makes it report something **incomplete** |
| 6 | #853 | **The named owner of three of row 4's four hidden markers, and the band's second `instance` candidate.** `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — `validate/cvccomplexcontent.go:445`, `validate/cvcid.go:232`, `validate/cvcidentityconstraint.go:390`, each verified here to name no issue (STYLE P3). **Start with an oracle question, not with code**: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type. **UNMEASURED** — run `GOXSD_DECLINES=1` and count before promising a figure. The ·nilled· shape is explicitly out of scope and each marker keeps its ·nilled· bullet |
| 7 | #414 | **Head of the longest chain in the queue, and now its largest remaining link.** Two bare `GAP(` markers in `xsd/complextype.go` — on `AttributeUses` and `AttributeWildcard` — name no owning issue, and the §3.4.2.4 clause 3 / §3.4.2.5 clause 2 folds walk the finalized Schema's type definitions only, so an anonymous complex type nested in a particle tree is folded for neither. **One decision, taken once, applied to both folds**: widen the reach, or record the narrow reach as a decided permanent under-approximation. It gates #438, #584 and — through both — #1051, whose parity residual is **584 documents** of exactly this shape. #1046 already discharged the 31-document link of that chain, so this is what is left. Ranked below the direct movers because it is a decision whose own lane movement is zero |
| 8 | #56 | **Unblocked ten days ago, still unbanked, and its design question is settled.** #719 shipped `validate.Unevaluated` with `Rule()`/`Loc()`/`Msg()` and `Result.Unevaluated()`, and `validate/doc.go` now states that an empty `Violations()` beside a non-empty `Unevaluated()` is not a pass. This issue records the CTA withhold into that same slice under a third rule ID — **no second type and no `Evaluated bool`**, which #842's warden pre-flight ruled out on D3. Expected surface: none new. It moves no lane by construction and is banded anyway, because STYLE 9's fail-open discipline is only honest if a fail-open answer is distinguishable from a real pass |
| 9 | #1066 | **The tenth consultation reconfirmed all three holes byte-unchanged, and a script author is blocked by them today.** The `-schema` repetition notation (lifted from #16, which can never be worked directly), exit-code aggregation across `<instance>...`, and the unstated stream for `parse`'s summary and `validate`'s violation lines. **Take it WITH #1031**, whose `-format` bullet edits the same synopsis line, and take both **before #1089**, whose text depends on what they leave in `usage`. One landing, three pages, and `TestUsageCoversContract` keeps them coupled |
| 10 | #1087 | **#1052's own advisory finding, and the asymmetry runs the wrong way.** `/develop` step 3 routes the `## Acceptance` satisfiability ruling to the oracle where the issue takes spec grounding and to the arbiter where it does not. The oracle branch got a template field (`ACCEPTANCE:`, `oracle.md:43`); the arbiter branch got a mandate and no shape (`arbiter.md:41-46`) — **and the arbiter branch is the one carrying three of #1052's four motivating failures** (#1030, #1029 twice, #719). A missing ruling therefore leaves no trace on exactly the branch where it matters. One paragraph in one agent file; must stay distinguishable from the `VERDICT:` block so an "unsatisfiable" is never miscounted against the two-rejection cap |
| 11 | #409 | **Seventh independent sighting, by two personas, none told the issue existed — the most-corroborated doc defect in this repo.** `codegen` and `codec` print `Generate`/`Target`/`AppendCanonical` in `go doc` code blocks while exporting **zero** symbols, and are the only two library packages for which `grep -in "not yet\|planned"` finds nothing about the surface shown. Four sites, one convention, **do not split**. This pass added the argument that should settle it: `gen`, `codegen`'s own CLI face, states *"Implemented today: the help path only"* and exits 2 — the same feature, honest on one surface and misleading on the other, inside one module |
| 12 | #1036 | **The silence #1029's landing exposed and STYLE P3 does not permit to stay.** A top-level `<xs:schema>` child outside the XSD namespace is neither charged as the §5.1 first-bullet grammar fault it is — §A's `<xs:schema>` content model has no wildcard arm, and `xs:openAttrs` admits foreign *attributes*, not foreign element children — nor reported through `parser.AssembledDocument.Unmapped`. **One settled disposition, either direction.** Adjacent and distinct from row 1: #1076 is about three other elements' own children, and #1047's body already named this issue as the owner of the foreign-namespace skip |

**Below the band, and why**: #1084 (the STYLE P3 widening-reach ruling) cost one
arbiter round on one issue — one sighting, not yet compounding, and it rises the
moment a second landing pays it. #1078 (retire-or-keep the five census
vocabularies #1047 made redundant) is cheap and correct but moves nothing and
blocks nothing. #1088 and #1089 were filed today and neither has a second
sighting. #1033, #1006, #1003, #1005, #1007 and #1031 are the persona-family
tail; #1031 rides with row 9 rather than ranking on its own.

### Next planning action

1. **Close the five. SEVENTH stamp, and now with the tree evidence attached.**
   #625, #748, #492, #934 and #896 are discharged in `main` — each verified
   here, each carrying a STOP comment, all five tabulated on **#1023**. It is
   five `gh api repos/kud360/goxsd8/issues/<n> -X PATCH -f state=closed -f
   state_reason=completed` calls and it is the develop loop's act, not the
   cartographer's. Until it happens `ready` reads 218 where 213 is true.
2. **The last stamp's trigger did not fire, and the replacement question is
   narrower.** The measurement instrument works; what is unproven is whether it
   works on a family it has not been run against. **#1076 is that test** — the
   same mechanism at three new elements, but UNMEASURED, where all three of the
   delivered rows arrived with a count. **If #1076 lands flat, the thing to
   re-examine is whether "in the suite's invalid corpus" is doing the work the
   census was credited with.** Trigger set here.
3. **The human decision blocking #1002 is unchanged and is now carried for a
   fifth stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`,
   enumerated by case ID with a per-case justification, and (b) holding
   §4.2.2's `vc:maxVersion` arm until real assertion evaluation lands. CLAUDE.md
   puts (a) beyond any agent — *"changes only via a human-filed issue"* — and
   (b) depends on **#1042**, filed and `blocked`. **No agent should attempt
   either.** The ruling is a comment on #1002; that comment is what moves it.
4. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3) and retires #719's `GAP(validate)`
   markers. **M6 tier 2 itself is uncarved** — `$value` binding, an F&O function
   library, typed comparison — and is too big for one issue. That carve is a
   `/backlog` act at the M6 opening, and #1042 is the thing it must slice
   around rather than a blank page.
5. **The unblock sweep measured a clean zero and the method is worth keeping.**
   All 22 `blocked` bodies were fetched over REST and their `## Depends on`
   sections read: **no open issue names any of this window's seven closures as a
   dependency**. #1051's body already records #1046 as DISCHARGED and correctly
   stays `blocked` on #414 → #438 (584 documents) and #786 plus five unfiled
   gate widenings (55). Eight of the 22 are **triggers rather than issues**
   (#79, #692, #841, #925, #1002, #1080, #555, #16) and say so in their own
   `## Depends on`; **do not re-scan those on the next sweep** — each states the
   instruction in its own body.
6. **#1052's post-land pass ran and disposed of everything, but its LOG signal
   is not on `main`.** Its four items are closed out on the thread (#1087 filed,
   the two-copy sentence dismissed with reasons, the wrong-citation instance
   recorded on #635, the stray worktree removed with `git worktree remove` and
   verified empty) and its own closing line promises *"a dated `post-land` entry
   in `docs/LOG/2026-08.md`"* — which `main` does not carry at `032d402`. That
   is the #400 failure mode from the other side: a completed pass indistinguishable
   from a skipped one in `git log`. **Chronicler owns the fix**; recorded here
   because a `/backlog` is the only pass that reads both surfaces.

**Standing, and re-checked rather than restated.** Four unlanded corrections
still target one paragraph of `docs/WORKFLOW.md`'s filing discipline —
**#510**, **#646**, **#679**, **#912** — and whichever lands last rebases three
times. **The next `/retro` inherits five** — #692, #925, #841, #1080 and the
fold-the-five-species question #1052's Notes hand it (#635, #912, #609, #510,
#646). **#841 is still the counter-example the steward-ranking rule cannot
reach**: a `kind/refactor` with a steward ranking, `blocked` because its trigger
has no mailbox, fired twice without a ruling. **There is no `Increasing`
steward ranking anywhere in the band.** The CTA cohort's 45 banked `instance`
failures remain unattributed, eighteenth consecutive stamp. `gate.yml` runs and
is still not a required status check, which only the repository owner can
change.

**Environment, one witness each.** Repository-scoped `gh api` REST served
**every read and 14 writes** here without a failure, exactly as
`docs/ROUTINES.md` says; `gh issue list` and `gh api --paginate` were not
attempted, on the strength of the last several stamps. The paginate recipe's
9-page cap was short by 2 for the second consecutive stamp (**#1062**). The
shallow clone truncated `origin/main` to 50 commits, which is why the retired
branches' dispositions were taken from GitHub rather than `git log` (**#802**).
No conformance measurement was taken by this pass: the lane table above is the
committed expectations, and `git diff --stat b5a45a6 032d402 --
conformance/testdata/expectations/` accounts for every verdict in it.

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
`schema` +34 on 2026-08-28. Live ones in the same family are **#471** (a local
`<element ref=>` carrying `substitutionGroup=`, silently accepted), **#931**
(occurrence attributes on a named `<group>`'s child compositor), **#929** and
**#455**. A second, narrower family opened beside it — the rejections the
producer already makes **correctly but describes badly**, whose bar
`xsderr/doc.go` set with #966 — and it is **discharged**: #975 landed at
`1dcffbf`, so every s4s-grammar rejection now names its Appendix A production.

**A THIRD family opened on 2026-08-27, it is the first whose members arrive
with a measured case count, and its first three members have LANDED.**
#1030's unmapped-construct census turned the "decides and ACCEPTS" family from
a shape into a list, and the list delivered: **#1047** (`57ad014`, `schema`
+34 — `checkS4SChildOrder` skips a child no position of the chosen model
admits), **#1048** (`fc58dc4`, `schema` +16 — a named `<group>` with two
compositor children loses the second) and **#1046** (`b14158c`, `schema` +23
and `instance` +15 — `<schema defaultAttributes=>` was unmodelled, so §3.4.2.4
clause 3's `{attribute uses}` fold never ran). All three are producer
widenings, and for all three the gate-side alternative was **measured and ruled
out**: widening `conformance/schema.go`'s shape gate costs a banked ratchet win.

**The family replenishes itself as it is worked, which is what to expect and
not tail growth.** M4's open count is unchanged at 48 across the window because
four closed (#975 with those three) and four opened — **#1076** (the same
missing s4s model at `xs:element`, `xs:attribute` and `xs:simpleType`),
**#1073**, **#1078** and **#1082**, three of the four filed by the post-land
passes of the landings that moved the lane. The GitHub milestone holds the
feature slices; the comment-accuracy, doc and process issues that post-land
passes file against the same packages sit outside it, so the milestone is a
floor on M4's remaining work and not the whole of it.

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
+9, unioned onto #716's). `instance` stands at **10760** — #913's cvc-type
clause 3.1 landing added **9409**, itself M5 and the largest single lane move
this project has recorded — and **twenty-five** of the pre-#913 cases were not
M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure (+12), #821
added 1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). (The
*thirteen* this paragraph carried predated the CTA pair and did not sum; the
five figures do, and the sum is the number.) **It happened again after #913:
#909 — an M4 landing — took `instance` 10746 → 10752 (+6) by producing
`<simpleContent>` `<restriction>`, **#862** — a `conformance` landing — took
it 10752 → 10755 (+3) at `109beb9`, and **#1001** — a `parser` landing — took it
10755 → 10760 (+5) at `684b2b4`, so the outside-M5 total is now 39.** A slice
that produces a component the engine could not previously see, or decides a
`{type table}` it previously withheld, moves `instance` without deciding a new
`cvc-` rule. **#1001 is the first FOURTH-mechanism instance**: it changed neither
the engine nor the harness, but pre-processed the schema document itself —
§4.2.2's ·conditional inclusion· removed declarations that were colliding under
`sch-props-correct` clause 2, so five documents the engine had never been given a
usable schema for became decidable. A lane can move because the schema the engine
was handed was wrong. **#862 is a THIRD mechanism and the paragraph would be wrong to
fold it into the other two**: it changed no engine behaviour at all, only how
`resolveExpected` picks a feature-scoped expected verdict, so the engine's
answers were already right for those three cases and the harness was reading the
wrong expectation. A lane can move because the measurement was fixed.

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

**10760 is still a floor built for soundness, and #913's +9409 jump did not
change what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the 15601 that
still fail are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read 10760 as 41% of the suite passing.** It is the count of documents
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

What remains of the CTA subset is **#859** (the wildcard `ta-AttrName` arms; the
live lease this paragraph recorded on 2026-08-18 is long gone — `wipsurvey` shows
no `wip/issue-859` ref, so it is unclaimed), **#888** (a cast target that is
`xs:QName`), **#889** (value-level numeric widening, §B.1 rule 1.1's
float→double promotion, which #858 withheld rather than faked) and **#894**
(err:XPST0051/XPST0080, the static remainder #886 did not charge). **#871** is
the §3.12.4 clause 1.1.3 ·inherited attributes· merge, blocked on M4's #831.
None of the seven carries a milestone, which is the same pattern M4's tail
records.

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

**#56 is STARTABLE, and #719's landing is what made it so.** A fail-open
"unevaluated" must never read as a genuine PASS; its `## Depends on` is now
empty, because #719 shipped the encoding (`validate.Unevaluated` with
`Rule()`/`Loc()`/`Msg()`, `Result.Unevaluated()` in document order) and #842 had
already settled the *evaluator* side (a compile-time `(CTATest, bool)`, `ok`
false being the withhold, with `Evaluate` always deciding). One encoding,
decided in #719 and reused there (STYLE D4). STYLE 9's fail-open discipline is
only honest if a fail-open answer is distinguishable from a real pass.

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
