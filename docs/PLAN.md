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

## Status — 2026-08-16 (full `/backlog`)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1330 | 25031 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12859 | 2539 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This stamp absorbs two landings**, both `schema` and both since the previous
stamp's baseline of 12850. **#830** rejected a nested `xs:simpleType` carrying
`name` or `final`, which `xs:localSimpleType` prohibits, for **+8** at `b0de54b`
— two more than its own measured floor, because `annotB026` and `annotB031` are
invalid on a second ground as well. **#823** charged `e-props-correct` clause 7
at a new Phase D step for **+1** at `2120ea1`, the ·xs:error· component #821
seeded being what made clause 7.2's exemption decidable at last. `instance` and
`datatypes` are untouched. The per-landing narrative is `docs/LOG/2026-08.md`,
not here.

**The lesson worth carrying is #830's, and it is about estimates rather than
about `simpleType`.** Its filing put six cases in the cohort and checked a
`GOXSD_DECLINES=1` census to prove none of the six sat in the decline list, so
`schema +6` was a **floor**. The lane paid +8. **A decline census bounds what a
fix cannot win; nothing bounds what it will win, because a case may be invalid
more than once.** An estimate worth trusting says which direction it can be
wrong in — which is the counterpart to #821's rule that a headroom figure counts
documents a change TOUCHES while a lane counts decisions it REVERSES.

Milestones, read from GitHub this pass. **The table is unchanged, and that is a
measurement rather than a carry-over**: both landings and all five issues filed
this pass are unmilestoned, matching the practice around them, so no cell could
move. M4's milestone `updated_at` is 2026-08-15T21:46Z and M5's is
2026-08-16T02:08Z — both predate this pass, confirming it.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 83 | 42 | **active** |
| M5 — Instance validation (XML) | 13 | 12 | **active** |
| M6–M12 | 0 | 0 | not filed |

Queue: **216 open issues — 193 `ready`, 23 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `blocked`, so both counted inside the 23), against **299 closed**.
193 + 23 = 216 exactly, and every one of the 216 carries a queue label —
**the arithmetic proves the sweep rather than a sweep asserting it, and the
previous stamp's one unexplained issue is gone**, resolved by this pass reading
all 216 rather than carrying a delta. Read the milestone table as feature
progress and not as the queue: **162** of the 216 carry no milestone
(216 − 42 − 12).

**Both halves of the move are stated.** #830 and #823 closed `ready`;
**#822 closed `not_planned` as superseded**, the first `needs-replan` this
roadmap has been able to report at zero; and this pass filed **#851**
(`blocked`), **#852**, **#853** and **#854** (all three `ready`). **No open issue changed queue
label.** All 23 `blocked` bodies were read for `## Depends on`: eight name an
open issue, seven open with *"A trigger, not an issue"*, two are the epics, and
none names a dependency that has closed. **That is the fourth consecutive pass
whose unblock step moved nothing**, at the cost of reading 23 bodies — recorded
on #779, which is the check that would replace the read.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed a hand-shaped issue list — #840 — because
`gh` is 403 on both GraphQL and REST here and the MCP channel served everything
else, which is docs/ROUTINES.md's fall-through rule working as written):

```
ISSUE  BRANCH         TIP AGE  VERDICT  REASON
822    wip/issue-822  unknown  RETIRED  wip/issue-822: issue #822 is closed
```

**The namespace holds `main` and one retired ref, and RETIRED is the correct
outcome rather than a leak.** `wip/issue-822` @ `cc2d54e` was parked on the
second arbiter rejection (PRINCIPLES 30) and is **re-planning evidence
deliberately kept**: #851 supersedes it and points at that SHA for the design
that survived both reviews — the §3.4.2.1 two-mint ownership split, the
`TypeDefinitionOrRef` growth, the `key-equiv-ta` re-derivation. It is never
force-pushed, never renamed, never a base to branch from, and **its deletion is
a human's call, not a session's**. Nothing is EXPIRED, there is no `parked/*`
ref. The shallow-clone finding (#802) still binds any ahead/behind arithmetic
here, which is why the tip age reads `unknown` and nothing is inferred from it.

**`go tool gapaudit` ran with a REAL `kind/gap` list for the first time in five
stamps**, and both halves of the audit are below rather than the census alone.
Census: **60 `GAP(` markers across 5 areas** — `xsd` 37, `validate` 14, `xml` 4,
`value` 3, `xpath` 2, down one from the previous stamp's 61 as #823 retired the
clause-7 marker at the former `xsd/resolve.go:740`, which discharges that
landing's last owed item. **Group 2 is nine `kind/gap` issues with no surviving
marker, all of them conformance-lane or coverage gaps that never carry one —
no stale tracker.** Group 1 is **ten markers, every one in `validate`**, and the
ten decompose exactly:

- **Four are instrument error — #852.** They cite an OPEN `kind/gap` tracker
  verbatim (`#414` once, `#774` three times) and `matches` never reads the `#N`
  a marker prints. Two more cite `#717` and `#56`, which are `kind/feature` and
  `kind/story` and so cannot be in a `--label kind/gap` input at all.
- **Four name no issue and say nothing about it — #853.** Three of those four
  are one unimplemented rule, `cvc-elt` clause 5.1, and the lane cost is an
  empty element with a default declining in three rules at once.

**Five markers still declare themselves unowned in as many words, and an issue
owns each** — `parser/produce_complex.go` (**#471**), `validate/icpath.go`
(#812), `xsd/redefinition.go` (#725), `xsd/contentmatcher.go` twice (#782,
#783). That is #815's list, re-derived: two of its five rows are already
discharged by #800's and #813's landings, and **its claim that the
`produce_complex.go` marker is "genuinely unowned and wants a filing at the next
`/backlog`" is false — #471 has owned it since 2026-08-04**, eleven days before
#815 was filed. This roadmap carried that claim for four stamps and it is
retired here, not re-filed. #725 is retitled to drop the same census claim from
its own title.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 193 of
them. Take from the top. **Re-derived in full this pass, not carried**: the
previous band's row 1 (#822) is closed, its row 2 (#830) and row 3 (#823) have
landed, and rows 4–9 had been carried through four stamps without re-argument.

**The first four rows are `kind/process` and `kind/tooling`, on purpose.** They
are banded on the sessions they cost, never on the lane they do not move (#527,
#565). Every one of them was paid by *this* pass or by the two landings it
absorbs, in writing, and each is one session.

| # | Issue | Why here |
|---:|---|---|
| 1 | #764 | **The cartographer's own instrument is broken, and this pass paid it six times.** The MCP `issue_read` path strips angle-bracketed tokens, so a body read and written back silently deletes every `xs:element` in it — and `gh` is 403 on both routes, so that read path is the only one a session has. `/backlog`'s mandate is that *a stale premise in an open body is fixed by editing that body, not by commenting only*; **this pass could not edit a single body and left six comments instead** (#584, #250, #725, #815, #471, #822). The premises are now correct **and unfindable from the bodies**, which is strictly worse than the drift it replaced. Nothing else in this band gets cheaper until it lands |
| 2 | #842 | **The single highest lane value in the queue, and the only row that ships a new package surface.** The §3.12.6 required-subset CTA evaluator, `ta-Test` through `ta-AttrName`, so a `{type table}` ·conditionally selects· a type instead of `walk.governingType` declining the element **and everything below it**. Every suite CTA case whose alternatives take the `@type` arm has a built table since #800 and declines today; #823 gave those tables a real clause-7 verdict. It also **gates #851** and decides the encoding **#56** and **#719** both need. Ratchet is to be MEASURED, not promised — the issue says so and says where |
| 3 | #836 | **The only MEASURED lane figure in the queue: `schema` +9, a floor.** Duplicate sibling `xs:annotation` children are silently accepted where `xs:annotated` admits one; all nine cases are suite-`invalid`, `fail`, absent from the 892-entry decline list, and probed individually through `parser.ParseReport`. Two suite-VALID cases (`annotB025`, `annotB027`) pin the exception list and a uniform guard regresses them — the negative half is in the Acceptance. Same `fmt.Errorf`-on-§5.1 footing #830 just used, so the grounding is done |
| 4 | #820 + #797 + #600 | **The landing mechanics, all three paid in the last two landings.** #830's `docs/LOG` entry was **lost by the squash-merge and re-landed by hand** at `a40639a`, and #823's own Next item 1 had to warn its successor about it in as many words; #820 is the emptiness check that reads PRESENT while the entry is absent, stated as a landing precondition by the arbiter in two consecutive landings. #797 is where a code-free iteration puts its entry. #600 is the one-append-point merge tax on a file now 2.4 MB — three retros old and the file has tripled since it was filed |
| 5 | #852 + #840 | **The survey instruments, both measured by this pass.** #852: `gapaudit`'s matcher ignores the `#N` a marker prints, so four correct markers report untracked and its untracked group must be hand-verified row by row — which is more expensive than the greps PRINCIPLES 27 replaced with it. #840: no non-`gh` producer for either survey's input JSON, hand-shaped twice today. They land together or #852 lands on input nobody can reproduce |
| 6 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced — and its premise, that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline, is a candidate for expiry now that #447 and #738 landed `list` and `union`. Converting it turns declines into decisions in `schema`. **It touches the same function family as #830**, which has now landed, so read that diff first |
| 7 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. Below #842 because #842 is what makes the "genuine PASS vs unevaluated" encoding a shared decision rather than a second one (STYLE D4), and below #786 because it converts silent wrongness into honest declines, which the lane scores the same |
| 8 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in **three** rules — the ·initial value· charge, the ID/IDREF binding, and the ·key-sequence· member. `instance` candidate, unmeasured, and the direction can only be up because all three decline today. Its first step is an oracle question: whether **#463**'s `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 9 | #625 → #669 → #748 → #492 | **README's Library block, ONE row in file order.** Splitting it across band rows is why it sat five passes. #625 fixes the `SchemaBuilder` pointer at closed #203; #669 the "works TODAY" snippet that does not compile; #748 the M5 block that denies a shipped API; #492 folds `ParseReport` into the sentence at `README.md:116` |
| 10 | #747 + #514 + #687 + #672 | the CLI contract, all four decided **before** #472 — the missing "Implemented today" paragraph, typo-vs-unbuilt, scoped help, `-version`. Each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards. #472 (`goxsd8 parse`) follows them |

**Deliberately unbanded, and why.** **#744** (chained-redefine successor, highest-
value M4 gap), **#773** and **#721** are held out on one shared condition: each
needs a warden pre-flight on an entry point that does not exist. **That condition
has an owner — #484** — which is the standing argument for a process issue
outranking its label: it unblocks three band-grade issues at once. **#774** is
three fail-open declines to close or rule permanent, and which way it moves a
lane is unknown until the first is decided; **#795** pins the element-side one
either way. **#771** is the `instance` lane's LAST decided-and-disagreeing case.
**#782**, **#783**, **#785**, **#787**, **#788**, **#793**, **#794**, **#796**
are earlier landings' follow-ups; **#805**, **#806**, **#809** and **#810** are
`tools/wipsurvey` hygiene on a tool that now works; **#812**, **#814**, **#817**,
**#819** are #718's and #716's; **#825** is #800's own and **closes as moot if
#851 lands first**, since #851 deletes the marker whose wording #825 corrects.
**#815** is the standing marker-repoint seam, now four issues over five sites
with its census re-derived above. **#854** is the 2026-08-16 libuser
consultation's one new filing — `xsd`'s package doc has six sections and no
index, with the Query API second of six — and it lands **beside** band row 9
rather than inside it, being the godoc half of that row's README question. **#843–#849** are the 2026-08-16 architecture
audit's seven findings, ranked by the steward's own cost-of-delay read; **#843**
(four hand-copied component descents, already diverged on the redefine-original
edge, probed live) is the one whose cost is stated as increasing steeply and is
the first of them a session should take.

**#831 and #851 are unbanded ON PURPOSE, for opposite reasons.** #831 is a real
one-line defect — `produceAttribute` hardcodes `{inheritable}` false where
`produceLocalAttribute` reads it — that **moves no lane today**: the suite holds
exactly one top-level `xs:attribute` carrying `inheritable`, its schema case
already passes, and its two instance cases are CTA-gated. **A `ready` issue with
`Ratchet: unchanged` in its Acceptance is the honest shape for that, not a reason
to withhold the label** (#347). #851 is unbanded because it is `blocked`: it is
#822's replacement and cannot be measured on a tree without #842, which is the
single fact that killed #822 after two sound implementations.

**`parser/redefine.go` has been rewritten five times in a week, so read it rather
than an issue body that describes it.** #686, #699, #506, #503 and #504 each
moved it; #744, #706 and #726 all open it next.

### Next planning action

**Re-derive the decline census before the next carve.** It predates #766, #715,
#740, #790, #718, #716, #813, #800, #821, #733, #830 and #823 — twelve landings
— and is by a wide margin the oldest measurement this roadmap argues from. Rows
6 and 7 of the band above are ordered on *reasoning* about which declines are
convertible, not on a measurement, and **#836 and #830 are the pair that shows
what the instrument buys**: one `GOXSD_DECLINES=1` run turned "a pair of cases
worth an issue" into a nine-case cohort with a floor attached, twice, and took a
single command each time. **#570** is the issue that makes it permanent — bank a
per-lane decline baseline so every landing announces the cases it just made
decidable — and **#571** is its soundness half.

**The queue is 215 and the ready band is ten rows, and that gap is not a
backlog problem.** `ready` means filed and unblocked; its size is an output and
never a target (#347). What the size does argue for is #779: this pass spent a
full read of 23 `blocked` bodies to conclude that nothing could be unblocked,
for the fourth consecutive pass. Two new instances of a shape #779's body does
not yet name — an epic carrying `blocked` as a parking brake rather than against
a live dependency, #79 and #250 — are recorded on its thread with a suggested
third probe.

**This pass's own finding, and the reason the band opens with #764.** Six stale
premises were found in open bodies and **none of them was fixed in the body**,
because the only channel available deletes XML tokens on write. `/backlog`'s
rule that a stale premise is fixed by editing the body — not by commenting only
— is currently unfollowable, and the six comments this pass left are the
evidence. **#851 and #853 were written to survive the defect**: both spell
element names as `xs:alternative` rather than in angle brackets, so a future
round-trip cannot silently empty them. That is a workaround at filing time and
not a fix, and it does nothing for the 213 bodies already written.

Everything else this queue needs is a develop iteration. **Start with #764**, or
with **#842** if the session wants lane movement and can accept that its own
filings will not be editable afterwards.

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
+9, unioned onto #716's). `instance` stands at **1330**, and the last six of those
cases are not M5's: **M4 landings keep moving this lane** — #740 took it 520 → 532
on a merged tree neither parent could measure, #821 added 1 (·xs:error·) and #733
added 5 (a top-level `<xs:attribute>`'s inline `<xs:simpleType>`). A slice that
produces a component the engine could not previously see moves `instance` without
deciding a new `cvc-` rule.

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

**1330 is still a floor built for soundness, and no jump has changed what the
number means.** The lane emits only "not valid" observations; a violation-free
`Result` DECLINES rather than passing, because `Assess` evaluates none of
`e-validity`'s other conjuncts. **Every passing case is an expected-INVALID one
by construction**, not by measurement, and the 25031 that still fail are
overwhelmingly declines rather than disagreements. The milestone's remaining
slices are what turn declines into decisions.

**Do not read 1330 as 5% of the suite passing.** It is the count of documents
this engine can honestly call not-valid, and it grew because the same rules now
reach every node instead of one. A slice that decides a *new* rule will move the
number far less than #790 did and be worth more.

**ONE case is decided and decided WRONG — #771**, a root whose declaring schema
is reachable only through the instance's own `xsi:schemaLocation`. It was four,
and #800 retired two of them: `Assert/assert_019/instance/assert_019_2` and
`CTA/typeAlternatives_001/instance/typeAlternatives_001_2` now decline honestly
instead of rejecting a document the ·conditionally selected· type admits.
**That is two, not three, and `CTA/cta0008.v01` was never among them** — it
takes §3.12.2's inline arm, which #800 deferred to #822 and which **#851** now
owns behind #842 (#822 is closed, superseded), and the count was
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
a fail-open "unevaluated") is blocked on **#842**, the §3.12.6 required-subset
CTA evaluator — filed 2026-08-16, so the "not-yet-filed evaluator issue" this
line waited on for a month now exists and #56's `## Depends on` names it. #842
covers the **CTA half only**; assertion evaluation is still unfiled, and #56
stays blocked on that too. Its design question is no longer M6's alone: **#719**
needs the
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
