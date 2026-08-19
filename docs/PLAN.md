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

## Status — 2026-08-19 (post-land pass for #868)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1337 | 25024 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12927 | 2471 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**The five-landing lane freeze the previous stamp headlined is over:
`schema` 12913 → 12927, +14 in two consecutive landings.** Both are
`area/parser` schema-for-schema-documents grammar work, both were measured per
case rather than assumed, and the two figures are provably disjoint — `comm -12`
over #883's eight (`groupN023 mgAb003 mgC009 mgG028 mgJ028 mgP055 mgP056
particlesEa024`) and #868's six (`annotB030 ctC009 ctF012 ctF015 notatF017
notatF065`) is empty, so neither landing's flips are double-counted. The case
count is unchanged at 15398, so `fail` fell by the same 14.

**What the previous stamp got wrong is worth naming, because the correction is
the planning content.** It read five zero-movement landings and concluded *"the
seam that pays and the seam that is being worked are not the same seam"*, with
grammar work as the seam that does not pay. Two grammar landings later that
generalisation is false: what those five landings shared was not *grammar*, it
was **no suite document in range** — three were `area/xpath` against a suite
whose 100 distinct `alternative/@test` values none of them touched. #883 and
#868 are the same kind of work against shapes the suite carries **nine and six
witnesses of**, and they converted immediately. **Census the population before
banding, not the mechanism** — which is #836's standing warning (a +9 forecast
taken over `msData/annotations/` alone paid +53) arriving from the other
direction.

Landings absorbed by this stamp, newest first:

- **#868** at `76ddaa8` — `complexTypeDecidable` stops declining a
  `simpleContent`/`complexContent` carrying NEITHER alternant, because the shape
  is a grammar fault the producer already rejects genuinely, and
  `produceSimpleContent`'s diagnostic is split so the neither-alternant fault
  stops naming a `restriction` the author never wrote. **`schema` +6, banked
  and attributed per case.** Arbiter ACCEPT round 1, zero findings; then #883
  landed under an already-accepted tree and a **gate-only second round**
  re-measured the merged figure live (12927) rather than adding the two.
- **#883** at `156ffd1` — §3.4.2.3.3 clause 2.1.4's `maxOccurs="0"` elision no
  longer returns before every per-element grammar check, so an entire elided
  subtree is validated. **`schema` +8.** It left one `GAP(xsd)` marker behind,
  owned by **#901**, which it filed.
- **#892 + #668 + #840** at `e0dd7c1` — `docs/ROUTINES.md`'s GitHub-channel
  facts corrected in both directions and the survey tools given a working input
  recipe. **`Ratchet: unchanged`**, and it retired two more issues as superseded
  (#527, #682, both `not_planned`). The recipe was run verbatim by this pass and
  produced both survey inputs losslessly, which is the first independent use.
- **#859** at `ea0650a` — the wildcard arms of `[17] ta-AttrName`'s NameTest.
  **`Ratchet: unchanged`**, and it **retired a `GAP(xpath)` marker** outright,
  9 → 8 by raw census.

Milestones, read from GitHub this pass.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 86 | 45 | **active** |
| M5 — Instance validation (XML) | 13 | 12 | **active** |
| M6–M12 | 0 | 0 | not filed |

M4 moved **85 → 86 closed** and **44 → 45 open**, and it reconciles as
44 − 1 + 1 + 1: #883 closed out of it, #901 filed into it by #883's own
post-land pass, #904 filed into it by this one. **#868 carried no milestone and
is counted in neither column** — it is one of the 168 unmilestoned, which is why
the closed figure moved by one against two landings. M5 and M0–M2 are unchanged.

**M3's `open_issues` counter still says 1 and the row above still says 0. The
row is right — this was resolved at the 2026-08-18 stamp and is carried, not
re-derived.** No item carrying M3 is open; the counter was left stale by #875
being reassigned M3 → M4 without decrementing. **A GitHub milestone counter is
not a source; the issue list is.**

Queue: **225 open issues — 201 `ready`, 24 `blocked`, 0 `needs-replan`,
2 `epic`** (both `blocked`, so both counted inside the 24), against
**321 closed**. 201 + 24 = 225 exactly, and **every one of the 225 carries a
queue label** — the class #773 and #774 fell into is empty for the second
consecutive stamp. Read the milestone table as feature progress and not as the
queue: **168** of the 225 carry no milestone (225 − 45 − 12).

**The move reconciles exactly: 231 + 3 − 9 = 225.** The nine closures are the
four landings above (#859, #892, #883, #868), the three issues those landings
closed alongside them (#668 and #840 with #892; #879, filed and closed inside
the previous stamp's own minute) and the two retired as superseded (#527, #682).
The three filings are **#900** and **#901**, both from the develop loop's own
post-land passes, and **#904** from this one.

**The unblock sweep moved nothing, for the eighth consecutive pass, and it was
run as a parse rather than by eye.** All 24 `blocked` bodies were matched
against the two issues that just closed: **no open body names #868 or #883.**
The only open body mentioning either is **#901**, which names #883 to record
that the dependency is already discharged — it was filed `ready` and stays
`ready`. Of the 24, sixteen name at least one still-open issue and eight name a
**trigger** rather than an issue (a `/retro` pass, a ruling, an epic reaching
zero), which is the distribution #779's script is built to keep separate.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, fed from #840's recipe:

```
ISSUE  BRANCH         TIP AGE   VERDICT  REASON
822    wip/issue-822  71h23m0s  RETIRED  wip/issue-822: issue #822 is closed
867    wip/issue-867  main's    CLAIMED  wip/issue-867: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  37h25m0s  RETIRED  wip/issue-872: issue #872 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-867` and `wip/issue-872`. Nothing is EXPIRED and there is no
`parked/*` ref. **`wip/issue-859` and `wip/issue-872`'s sibling are gone by
GitHub's auto-delete on merge**, which is the whole of the change since the
previous stamp bar one new entrant.

**`wip/issue-867` is the new entrant, it is an EMPTY claim, and the thread
settles it: no work exists on that branch.** Its tip is `ea0650a` — #859's
landing, the commit it branched from — so it has pushed nothing of its own and
is never EXPIRED and never resumable on age (#722). Its thread carries **one
comment, a completed GROUNDING at 2026-08-18T16:18:06Z**, and no RESUME note and
nothing since: 14 hours at this stamp. **The grounding is durable and is the
asset** — a session taking #867 reads it and starts from it rather than
re-paying an oracle round. This is deliberately NOT `needs-replan`: there is no
work to supersede, only a claim that was never cashed, and #867 is banded below
on that basis.

**`wip/issue-822` @ `cc2d54e` and `wip/issue-872` @ `0b34c21` are both RETIRED
and both deliberately kept**, superseded by #851 and #878 respectively. Carried
unchanged from the previous stamp, which verified their content absent from
`main` by reading `main` rather than by ahead/behind arithmetic the shallow
clone (#802) forbids. They are never force-pushed, never renamed, never a base
to branch from, and **their deletion is a human's call, not a session's**.

**`go tool gapaudit`: 64 `GAP(` markers across 5 areas** — `xsd` 37,
`validate` 14, `xpath` 6, `xml` 4, `value` 3 — run **with reconciliation**, not
census-only. **Group 1 is EMPTY: every marker in the tree has an open tracking
issue.** The total is unchanged against the previous stamp and the composition
is not, which reconciles against the diffs exactly: **#859 retired one
`GAP(xpath)`** and **#883 added one `GAP(xsd)`**, verified by censusing
`f17da28`, `ea0650a`, `e0dd7c1`, `156ffd1` and `76ddaa8` in turn — the step
change is at those two commits and nowhere else, and **#868 added and retired
none**, consistent with a landing that minted no rule ID. #883's new marker
cites `#901` in its own text, which is why Group 1 stayed empty. Group 2's nine
entries are all `kind/gap` issues that never carried a marker — conformance lane
gaps, and in #398's case a tracker that says so in its own title — which is
where that group is permanent.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 201 of
them. Take from the top. **Three of the previous band's top four have landed** —
row 2 (#892 + #668 + #840), row 3 (#883) and row 4 (#868) — and row 1 (#867)
survives at row 3 below. The band is re-cut rather than shifted up, because the
lane freeze that set the previous ordering is over and two rows have been
measured since.

**Rows 1 and 4 are `kind/process` and they outrank both measured lane slices on
purpose.** The rule is to band process and tooling work on the sessions it costs
rather than on the lane it does not move (#527, #565): friction the log records
in *consecutive* landings compounds every pass, while the fix is one session. Both
rows are doc-only, both were paid again in this window, and #900's payment is
silent corruption of the durable cross-session channel, which is the worst class
of tax this repo has.

| # | Issue | Why here |
|---:|---|---|
| 1 | #900 | **`gh api -f body=@file` posts the literal string `@/path`; `-F` is the flag that expands a file, and no document says so.** Third witness and **second independently reproduced** one: #892's grounding comment, then #868's, on two threads in two consecutive landings by two different sessions — which closes the question of whether the first was one session's slip. The corruption is silent, on the channel `docs/WORKFLOW.md` names as one of the three durable things, and both instances were caught only because the author happened to read the comment back. The deliverable is prose in one file and the body already forbids a wrapper |
| 2 | #904 | **`schema` +4, measured, and the conformance gate needs no change.** `produceComplexType` dispatches on `childElement`'s FIRST hit, so a `complexType` carrying two content wrappers is accepted: `ctB005`, `ctB006`, `ctB019`, `ctB020`, all suite-`invalid`, all recorded `fail`, all returning `<nil>` from `parser.Parse`, and **all four decided rather than declined** — so the flips are directly bankable. Filed by this pass out of #868's deliberately-unfiled follow-up, which named one of the four; the 15470-file scan found the other three. `parser/produce_complex.go` has been read twice in two days and `rejectRepeatedAnnotations` (#836) is the guard shape to copy |
| 3 | #867 | **`schema` +2, measured, and the oracle round is already paid.** An `annotation` carrying `annotation` CHILDREN is still accepted — `annotation`'s own content model (`:5755`), not `xs:annotated`'s cardinality, which is what #836 landed. `annotB001` and `annotB005` are the suite's only two such documents, both `invalid`, both `fail`, neither declined. **The GROUNDING is done and on this issue's own thread**; `wip/issue-867` is an empty claim holding no work, so a session starts from the grounding and pushes the first commit that branch will carry. Same family as row 2 and cheap to take second |
| 4 | #659 | **Seventeen landings, and the payer is the arbiter's container, which reads neither document the original Acceptance nominated.** A fresh checkout starts with `testdata/xsdtests` unpopulated and the gate fails on the missing-suite guard (#309); #868 paid it again, and its entry counts **nine of the last ten landings**. The body has already been rewritten to name `.claude/agents/arbiter.md` as the correction that matters and to forbid the mason-copy-plus-WORKFLOW-copy restatement. One line, one session, and its escape hatch — *close it with the finding if the only real fix is outside the repo* — has been open seventeen landings without being taken |
| 5 | #901 | **#883's own follow-up, and the file is still warm.** §3.4.2.3.3 clause 2.1.2 answers before 2.1.4, so an EMPTY `sequence`/`all` carrying `maxOccurs="0"` at the TOP model-group position escapes `p-props-correct` while the identical element one level down is charged. `Ratchet:` **unmeasured and expected unchanged** — a census during #883's grounding found no witness among 39 candidates — which is why it sits below the two measured rows and not above them. It owns the one `GAP(xsd)` marker #883 landed, whose text carries a typo to fix rather than carry forward |
| 6 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in **three** rules at once — the ·initial value· charge, the ID/IDREF binding and the ·key-sequence· member. `instance` candidate, unmeasured, direction can only be up because all three decline today. First step is an oracle question: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 7 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced, and its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. Same function family as #830, #836, #867 and #904, all landed or banded; read those diffs first, and #868's in particular, which is the most recent demonstration that one of these declines was collateral rather than forced |
| 8 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. It **gates #56**, and it decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. #56's `## Depends on` now names it alone, #842 having been struck when it landed |
| 9 | #669 → #625 → #748 → #896 → #492 | **README's Library block, ONE row in the order the issues themselves name.** Splitting it across band rows is why it sat seven passes, and the five overlap by paragraph rather than partition cleanly, so whichever lands second rebases on the first. #669 fixes the "works TODAY" snippet that does not compile and the example list that omits `xsd/example_test.go`; #625 the `SchemaBuilder` pointer at closed #203; #748 the M5 block that denies a shipped API; **#896** the package doc that never says which accessor is the verdict; #492 folds `ParseReport` in. **#748 led the libuser report for the THIRD consecutive consultation.** #896's is the sharp one: `Result.Err()` is a walk-fault indicator, not a verdict, and a caller taking README's own `err`-named variable at face value **silently passes documents carrying real `cvc-*` violations** |
| 10 | #870 + #747 + #514 + #687 + #672 | **The CLI contract, all five decided BEFORE #472.** The 2026-08-18 cliuser reconfirmed every one and filed nothing new — which is itself the result: the gap is disclosure, not discovery. #870 is the one a user hits first (Quickstart's `go build ./...` writes no binary; the stub's own `go doc` remedy fails wherever an installed CLI runs), #747 the missing "Implemented today" paragraph, #514 typo-versus-unbuilt, #687 scoped help (also carrying `goxsd8 -help validate`, the flag-first spelling), #672 `-version`. Each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 11 | #885 | **The scope rule that already cost a reject round, and today it survives only in a verdict comment.** #876's round 1 classified a surviving hole as out-of-scope pre-existing; the arbiter REJECTED and had to re-derive that it was not. The discriminator, in that arbiter's words — *a fault escaping an early return is a REPAIR when the offending element is the direct dispatch target of the function holding the return, and a SEPARATE ISSUE when it is not* — belongs in `docs/WORKFLOW.md`'s Scope section. One landing produced both answers (#876 the repair, #883 the follow-up), and #868 then applied the same test by hand to leave the both-alternants accept alone — a third payer for a rule nobody has written down |
| 12 | #820 + #797 + #600 | **The landing mechanics.** #830's LOG entry was lost by a squash-merge and re-landed by hand; #858's and #887's Next lists each had to warn the next session in as many words to re-verify the entry is in the diff; **#868's merge forward hit the same file as its only real conflict**, resolved by keeping both entries in landing order. #820 is the emptiness check that reads PRESENT while the entry is absent, #797 is where a code-free iteration puts its entry, #600 is the one-append-point merge tax on a file now past 2.4 MB — and #868 is the newest evidence for #600 specifically |

**Deliberately unbanded, and why.** **#888**, **#889** and **#894** are the three
`area/xpath` gaps #858 and #886 filed; they are correct, startable and still
below the fold, but the reason has narrowed — not *"`area/xpath` landings move
nothing"*, which this window's evidence no longer supports as a rule, but that
**no census has been taken of what the suite holds in their range**, and #889
states a warden pre-flight as its first step, which is **#484**'s standing
condition. **#871** stays `blocked` on **#831**, which its arrival re-priced.
**#881** is `blocked` on the next `/retro`. **#875** is #706's own follow-up and
**#884** is #876's and #883's neighbourhood — read each beside its parent.
**#843–#849** are the 2026-08-16 architecture audit's seven findings; **#843** is
still the one whose cost of delay is stated as increasing steeply. **#852** is
`gapaudit`'s matcher, and it stays out of the band for the second consecutive
stamp because the tool again ran with reconciliation and Group 1 empty. **#744**,
**#773** and **#721** are still held out on one shared condition, and **#484**
owns it.

### Next planning action

**Census the suite before banding the next correctness slice, and make the
census a tool rather than a habit.** This window is the argument: the two
landings that converted were banded off measured witness counts (nine and six),
the five that did not were banded off mechanism, and this pass filed **#904** on
the strength of a 15470-file scan that turned a one-document note into a
four-document `schema` +4. **#836 is the standing warning about how to do it
wrong** — a +9 forecast taken over `msData/annotations/` alone paid +53, so
*state the population beside the number or the number is not a measurement* —
and **#510** already says a cartographer must grep the suite before an
Acceptance asserts "no suite case reaches X". Neither is a tool, and every
census in this window was a throwaway script.

**The general form is #570, and its cost of delay is now quantified.** Bank a
per-lane decline baseline so every landing announces the cases it just made
decidable; **#571** is its soundness half. The standing `schema` decline count is
**893** as of `c116408` and has not been re-derived since — it now predates
twenty-three landings, and it is by a wide margin the oldest measurement this
plan still argues from. #868 is the sharpest demonstration of what that costs:
its six flips were **declines nobody knew were reachable**, found by hand-running
`GOXSD_DECLINES=1` per case, and the same hand-run had to be repeated four times
by this pass to establish that #904's four cases are *not* declines. That is the
same measurement paid five times in two days.

**The CTA cohort's 45 banked `instance` failures are still unattributed**, and
this stamp does not promote that question — it was the previous stamp's next
action, no landing has touched it, and nothing on record says why any of the 45
fails. It stays open and stays true; it is below the census question only
because the census question is what would tell a session which of the 45 to
start with.

**The queue is 225 and the band is twelve rows, and the gap is not a backlog
problem.** `ready` means filed and unblocked; its size is an output and never a
target (#347). Every one of the 225 carries a queue label for the second
consecutive stamp.

Take from the top: **start at row 1 (#900)** — one file, one session, and it
stops a silent corruption of the durable channel that two consecutive landings
have now paid. If the next session must move a number instead, **row 2 (#904)**
is `schema` +4 already measured.

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
