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

## Status — 2026-08-19 (post-land pass for #904)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1337 | 25024 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12931 | 2467 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**Three consecutive lane-moving landings: `schema` 12913 → 12931, +18.** #883
(+8), #868 (+6) and now #904 (+4), all `area/parser` schema-for-schema-documents
grammar work, all measured per case, all disjoint. The case count is unchanged at
15398, so `fail` fell by the same 18.

**The previous stamp's correction held, and this landing is the first test of it
that was set up in advance.** That stamp replaced *"grammar work is the seam that
does not pay"* with **census the population before banding, not the mechanism**.
#904 was then banded off a 15470-file scan that found four suite witnesses, and
it paid **exactly four** — the first forecast in this window to land on its
number rather than near it. #836's standing warning still cuts the other way (a
+9 forecast taken over `msData/annotations/` alone paid +53), and both readings
say the same thing: **state the population beside the number, or the number is
not a measurement.**

Landing absorbed by this stamp:

- **#904** at `830f9a8` — `produceComplexType` dispatched on `childElement`'s
  first hit, so a `complexType` carrying two content wrappers assembled from
  whichever came first, and a wrapper carrying both `restriction` and `extension`
  produced from one alternant and dropped the other. Both are now charged as
  plain grammar faults on §5.1's first **unnumbered** bullet — the oracle
  confirmed no numbered rule ID exists for either, and `src-ta` proves the
  omission is deliberate. **`schema` +4, banked and attributed per case**
  (`ctB005`, `ctB006`, `ctB019`, `ctB020`). Arbiter **REJECT then ACCEPT**: round
  1 read an Acceptance item that exempted a half from the *ratchet figure* as
  exempting it from *scope* and shipped half the Goal.

Milestones, read from GitHub this pass.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 87 | 46 | **active** |
| M5 — Instance validation (XML) | 13 | 12 | **active** |
| M6–M12 | 0 | 0 | not filed |

M4 moved **86 → 87 closed** and **45 → 46 open**, reconciling as 45 − 1 + 2:
#904 closed out of it, #908 and #909 filed into it by this pass. #907 was filed
without a milestone, like the rest of the tooling and process queue. M5 and
M0–M2 are unchanged.

**M3's `open_issues` counter still says 1 and the row above still says 0. The
row is right** — resolved at the 2026-08-18 stamp, carried, not re-derived. No
item carrying M3 is open; the counter was left stale by #875 being reassigned
M3 → M4 without decrementing. **A GitHub milestone counter is not a source; the
issue list is.**

Queue: **227 open issues — 203 `ready`, 24 `blocked`, 0 `needs-replan`,
2 `epic`** (both `blocked`, so both counted inside the 24), against
**322 closed**. 203 + 24 = 227 exactly, and **every one of the 227 carries a
queue label** — the class #773 and #774 fell into is empty for the third
consecutive stamp. Read the milestone table as feature progress and not as the
queue: **169** of the 227 carry no milestone (227 − 46 − 12).

**The move reconciles exactly: 225 + 3 − 1 = 227.** The one closure is #904. The
three filings are all this pass's, all out of the landing's own follow-up list:
**#907** (the `childElement` survey), **#908** (`xs:notation`'s ignored children)
and **#909** (the one complex-type form the producer still declines).

**The unblock sweep moved nothing, for the ninth consecutive pass, and it was run
as a parse rather than by eye.** All 227 open bodies were matched for `#904`:
**none mentions it at all.** All 24 `blocked` bodies were then re-read for their
`## Depends on` — sixteen name at least one still-open issue and eight name a
**trigger** rather than an issue (a `/retro` pass, a ruling, an epic reaching
zero), which is the distribution #779's script is built to keep separate.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, fed from #840's recipe:

```
ISSUE  BRANCH         TIP AGE   VERDICT  REASON
822    wip/issue-822  74h48m0s  RETIRED  wip/issue-822: issue #822 is closed
867    wip/issue-867  main's    CLAIMED  wip/issue-867: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  40h50m0s  RETIRED  wip/issue-872: issue #872 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-867` and `wip/issue-872`. Nothing is EXPIRED and there is no
`parked/*` ref. **The namespace is unchanged since the previous stamp**:
`wip/issue-904` went by GitHub's auto-delete on merge and no new claim was made.

**`wip/issue-867` is still an EMPTY claim, now 18 hours old, and the thread still
settles it: no work exists on that branch.** Its tip is `ea0650a` — #859's
landing, the commit it branched from — so it has pushed nothing of its own and is
never EXPIRED and never resumable on age (#722). Its one comment is a completed
GROUNDING at 2026-08-18T16:18:06Z. **The grounding is durable and is the asset**;
a session taking #867 reads it and starts from it rather than re-paying an oracle
round. Deliberately NOT `needs-replan`: there is no work to supersede, only a
claim that was never cashed.

**`wip/issue-822` @ `cc2d54e` and `wip/issue-872` @ `0b34c21` are both RETIRED
and both deliberately kept**, superseded by #851 and #878. Their content was
verified absent from `main` by reading `main`, not by ahead/behind arithmetic the
shallow clone (#802) forbids. They are never force-pushed, never renamed, never a
base to branch from, and **their deletion is a human's call, not a session's**.

**`go tool gapaudit`: 64 `GAP(` markers across 5 areas** — `xsd` 37,
`validate` 14, `xpath` 6, `xml` 4, `value` 3 — run **with reconciliation**, not
census-only. **Group 1 is EMPTY: every marker in the tree has an open tracking
issue.** Total and composition are both unchanged against the previous stamp,
which is what a landing that minted no rule ID and left no marker should produce,
and the arbiter's own grep over the branch diff confirmed it independently.
Group 2 went **nine → ten**: #908, a conformance-lane gap tracker that will never
carry a marker, which is where that group is permanent.

**A marker census is not a debt census, and this pass found the gap.** #909 is a
**whole declined representation form** — `<simpleContent>` with `<restriction>`,
§3.4.2.2 cases 1-2 — documented at `parser/produce_complex.go:429`-`:436`,
declined again gate-side at `conformance/schema.go:980`-`:983`, carrying **no
`GAP(` marker** and, until this pass, **no open issue**. `gapaudit` could not see
it and never could. Treat "Group 1 empty" as "every *marked* site is tracked",
not as "every deferral is tracked".

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 203 of
them. Take from the top. **Row 2 of the previous band (#904) has landed**; the
rest are shifted rather than re-cut, with three of this pass's filings placed on
their own evidence.

**Rows 1 and 2 are `kind/process` and they outrank the measured lane slices on
purpose** (#527, #565): friction the log records in consecutive landings
compounds every pass, while the fix is one session. **They have swapped since the
previous stamp, on this landing's evidence** — #659 was paid again and #900 was
not.

| # | Issue | Why here |
|---:|---|---|
| 1 | #659 | **Eighteen landings, and it was paid again by this one — ten of the last eleven.** A fresh checkout starts with `testdata/xsdtests` unpopulated and the gate fails on the missing-suite guard (#309); #904's arbiter paid it in both rounds. #886 is still the single exception. The body already names `.claude/agents/arbiter.md` as the correction that matters — the arbiter's container is the payer, and it reads neither document the original Acceptance nominated — and already forbids the mason-copy-plus-WORKFLOW-copy restatement. One line, one session, and its escape hatch (*close it with the finding if the only real fix is outside the repo*) has been open eighteen landings without being taken |
| 2 | #900 | **`gh api -f body=@file` posts the literal string `@/path`; `-F` is the flag that expands a file, and no document says so.** Third witness and second independently reproduced one (#892's grounding comment, then #868's, two threads, two consecutive landings, two sessions). **Not paid by this landing — which is not evidence it is fixed.** The corruption is silent, on the channel `docs/WORKFLOW.md` names as one of the three durable things, and both known instances were caught only because the author happened to read the comment back. The deliverable is prose in one file and the body already forbids a wrapper |
| 3 | #867 | **`schema` +2, measured, and the oracle round is already paid.** An `annotation` carrying `annotation` CHILDREN is still accepted — `annotation`'s own content model (`:5755`), not `xs:annotated`'s cardinality, which is what #836 landed. `annotB001` and `annotB005` are the suite's only two such documents, both `invalid`, both `fail`, neither declined. **The GROUNDING is done and on this issue's own thread**; `wip/issue-867` is an empty claim holding no work, so a session starts from the grounding and pushes the first commit that branch will carry |
| 4 | #908 | **`schema` +2, decided, and the conformance gate needs no change.** `produceNotation` (`parser/produce.go:2383`-`:2396`) reads three attributes and never touches `Children()`, so every child under an `<xs:notation>` is ignored. `xs:notation` (`:5696`) extends `xs:annotated` (`:4426`), whose content is `<xs:annotation>?` and nothing else; **§3.14.3 reads "None as such."**, so it is #884's shape exactly — a plain `fmt.Errorf` on §5.1's first bullet, no rule ID to mint. `notatF018` and `notatF066` are both suite-`invalid`, both `fail`, and both admitted unconditionally by `conformance/schema.go:618`. Named with no owner by two consecutive LOG entries before this pass filed it; same family as row 3 and cheap to take beside it |
| 5 | #901 | **#883's own follow-up, and the file is still warm.** §3.4.2.3.3 clause 2.1.2 answers before 2.1.4, so an EMPTY `sequence`/`all` carrying `maxOccurs="0"` at the TOP model-group position escapes `p-props-correct` while the identical element one level down is charged. `Ratchet:` **unmeasured and expected unchanged** — a census during #883's grounding found no witness among 39 candidates — which is why it sits below the measured rows. It owns the one `GAP(xsd)` marker #883 landed, whose text carries a typo to fix rather than carry forward |
| 6 | #909 | **The last complex-type representation the producer declines outright, and the largest unbanded `schema` movement in M4.** `<simpleContent>` with `<restriction>` — §3.4.2.2 cases 1-2 synthesize an anonymous simple type from the restriction's facet children — errors at `parser/produce_complex.go:586` and is declined gate-side at `conformance/schema.go:980`. **103 suite `.xsd` files carry the shape**, which is an upper bound and not a forecast. It sits below the measured rows for exactly that reason: **census the decline set (`GOXSD_DECLINES=1`) and settle the sizing before starting**, because the two base arms plus the facet synthesis may not be one landing, and a half-built form must not reach `main`. Filed by this pass; it had no owner and no marker |
| 7 | #907 | **Four hand-written guards against one mechanism, across 39 `childElement` call sites, each written only after a suite case tripped over it.** `rejectRepeatedAnnotations` (#836), `rejectBothInlineTypes` (#340), and #904's two. The catalogue this asks for reproduces all four and the one the #904 grounding names as still unguarded (`restrictionType`'s inner choice, `:4835`-`:4842`). **Banded below the lane rows and not with rows 1-2, deliberately:** the tax here was paid four times over months, not in consecutive sessions, and #904 has just guarded the three hottest sites — so it is real debt on a cooling seam, not compounding friction. It carries an escape hatch to close with the catalogue and no tool |
| 8 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in **three** rules at once — the ·initial value· charge, the ID/IDREF binding and the ·key-sequence· member. `instance` candidate, unmeasured, direction can only be up because all three decline today. First step is an oracle question: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 9 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced, and its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. Same function family as #830, #836, #867 and #904, all landed or banded; read those diffs first, and #868's in particular, which is the most recent demonstration that one of these declines was collateral rather than forced |
| 10 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. It **gates #56**, and it decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. #56's `## Depends on` now names it alone, #842 having been struck when it landed |
| 11 | #669 → #625 → #748 → #896 → #492 | **README's Library block, ONE row in the order the issues themselves name.** Splitting it across band rows is why it sat seven passes, and the five overlap by paragraph rather than partition cleanly, so whichever lands second rebases on the first. #669 fixes the "works TODAY" snippet that does not compile and the example list that omits `xsd/example_test.go`; #625 the `SchemaBuilder` pointer at closed #203; #748 the M5 block that denies a shipped API; **#896** the package doc that never says which accessor is the verdict; #492 folds `ParseReport` in. **#748 led the libuser report for the THIRD consecutive consultation.** #896's is the sharp one: `Result.Err()` is a walk-fault indicator, not a verdict, and a caller taking README's own `err`-named variable at face value **silently passes documents carrying real `cvc-*` violations** |
| 12 | #870 + #747 + #514 + #687 + #672 | **The CLI contract, all five decided BEFORE #472.** The 2026-08-18 cliuser reconfirmed every one and filed nothing new — which is itself the result: the gap is disclosure, not discovery. #870 is the one a user hits first (Quickstart's `go build ./...` writes no binary; the stub's own `go doc` remedy fails wherever an installed CLI runs), #747 the missing "Implemented today" paragraph, #514 typo-versus-unbuilt, #687 scoped help (also carrying `goxsd8 -help validate`, the flag-first spelling), #672 `-version`. Each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 13 | #885 | **The scope rule that has now cost a reject round TWICE, and it still survives only in verdict comments.** #876's round 1 classified a surviving hole as out-of-scope pre-existing and the arbiter had to re-derive that it was not; **#904's round 1 read an Acceptance exemption from the *ratchet figure* as an exemption from *scope* and shipped half the Goal.** Same class, two different discriminators, one sighting each — which is why this stays a single issue and no second one was filed. #876's, in that arbiter's words: *a fault escaping an early return is a REPAIR when the offending element is the direct dispatch target of the function holding the return, and a SEPARATE ISSUE when it is not.* #904's datum is recorded on the thread, explicitly not as a widening of the Acceptance |

**Deliberately unbanded, and why.** **#888**, **#889** and **#894** are the three
`area/xpath` gaps #858 and #886 filed; correct, startable and still below the
fold, because **no census has been taken of what the suite holds in their
range**, and #889 states a warden pre-flight as its first step, which is
**#484**'s standing condition. **#444** is annotated but not banded: #904's
grounding answers its question 2 in the general form (an element-vs-element
exclusion expressible as a plain `xs:choice` carries no numbered rule ID;
`src-ta` is the contrast), which shrinks its oracle round without closing it —
its other three questions and its gate-asymmetry item are untouched. **#453**'s
body was corrected this pass: one of its two defects was already discharged by
`dd4f1d8`. **#871** stays `blocked` on **#831**. **#881** is `blocked` on the
next `/retro`. **#875** is #706's own follow-up and **#884** is #876's and
#883's neighbourhood — read each beside its parent. **#843–#849** are the
2026-08-16 architecture audit's seven findings; **#843** is still the one whose
cost of delay is stated as increasing steeply. **#846** is the root cause under
row 6 — the ~700-line hand-maintained shadow of producer coverage that #909 must
edit in lockstep — and it stays unbanded only because #909 is the slice that
proves the tax rather than the one that removes it. **#852** is `gapaudit`'s
matcher, out of the band for the third consecutive stamp because the tool again
ran with reconciliation and Group 1 empty. **#744**, **#773** and **#721** are
still held out on one shared condition, and **#484** owns it.

### Next planning action

**Make the census a tool, and stop paying for it per landing.** Three
consecutive lane-moving landings were all banded off measured witness counts, and
this pass ran four throwaway censuses of its own — a 39-site `childElement` call
survey, a 15470-file scan for `<simpleContent><restriction>` (103 hits), a
decidability read of `conformance/schema.go` for two notation cases, and a
227-issue body match for `#904`. **#510** already requires a cartographer to grep
the suite before an Acceptance asserts "no suite case reaches X", and **#836** is
the standing warning about doing it by directory. Neither is a tool.

**The general form is #570, and its cost of delay is now quantified.** Bank a
per-lane decline baseline so every landing announces the cases it just made
decidable; **#571** is its soundness half. The standing `schema` decline count is
**893** as of `c116408` and has not been re-derived since — it now predates
twenty-four landings, and it is by a wide margin the oldest measurement this plan
still argues from. Row 6 (#909) is the sharpest case for it: **its Acceptance
cannot be written without a decline census**, its 103 candidate files are an
upper bound nobody can narrow by hand, and it is the largest single `schema`
movement left in M4.

**A marker census is not a debt census.** This pass found a whole declined
§3.4.2.2 representation form with no `GAP(` marker and no owning issue, which
means `gapaudit` reporting "Group 1 empty" for the third consecutive stamp has
been answering a narrower question than it reads as. Either the deferral gets a
marker convention that `gapaudit` can see, or the audit's own output says which
question it is answering. **#852** owns the matcher and is the place to decide
this; it has been unbanded three stamps running on the strength of exactly the
signal this paragraph is qualifying.

**The CTA cohort's 45 banked `instance` failures are still unattributed**, and
this stamp does not promote that question either — it was the previous two
stamps' next action, no landing has touched it, and nothing on record says why
any of the 45 fails. It stays open and stays true.

**The queue is 227 and the band is thirteen rows, and the gap is not a backlog
problem.** `ready` means filed and unblocked; its size is an output and never a
target (#347). Every one of the 227 carries a queue label for the third
consecutive stamp.

Take from the top: **start at row 1 (#659)** — one line in one agent file, and
this landing paid its tax twice. If the next session must move a number instead,
**row 3 (#867)** is `schema` +2 with its grounding already banked on the thread,
and **row 4 (#908)** is another +2 beside it.

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
producer widening, finalize validity and composition edge cases —
**`<simpleContent>` with `<restriction>` (#909) is the one whole
representation form still declined outright**, and everything else in
§3.4.2 is produced. The GitHub milestone holds the feature slices; the
comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it, so the milestone is a floor on
M4's remaining work and not the whole of it.

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
+9, unioned onto #716's). `instance` stands at **1337**, and **twenty-five** of those
cases are not M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure (+12), #821
added 1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). (The
*thirteen* this paragraph carried predated the CTA pair and did not sum; the
five figures do, and the sum is the number.) A slice
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

**1337 is still a floor built for soundness, and no jump has changed what the
number means.** The lane emits only "not valid" observations; a violation-free
`Result` DECLINES rather than passing, because `Assess` evaluates none of
`e-validity`'s other conjuncts. **Every passing case is an expected-INVALID one
by construction**, not by measurement, and the 25024 that still fail are
overwhelmingly declines rather than disagreements. The milestone's remaining
slices are what turn declines into decisions.

**Do not read 1337 as 5% of the suite passing.** It is the count of documents
this engine can honestly call not-valid, and it grew because the same rules now
reach every node instead of one. A slice that decides a *new* rule will move the
number far less than #790 did and be worth more.

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
