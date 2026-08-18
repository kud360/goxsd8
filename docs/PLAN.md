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

## Status — 2026-08-18 (`/backlog`)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1337 | 25024 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12913 | 2485 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This stamp absorbs five landings and NOT ONE MOVED A LANE — and every one of
the five measured that rather than omitting it.** Both active lanes are
byte-identical to the previous stamp, and `git log` over
`conformance/testdata/expectations/` confirms it from the other side: its most
recent touch is `c116408`, the landing the *previous* stamp closed on. Five
consecutive `Ratchet: unchanged` is the longest such run on record and it is the
fact this stamp exists to publish.

- **#706** at `85fe690` — an `xs:override`-substituted `xs:redefine` child
  finally reaches the prohibited-attribute guard (§F.2 clause 1, src-override
  clause 3's Note binding D_old′). Four lines in `prescanRedefine`.
  **Unchanged, MEASURED not inherited.** Its one comment-only finding is worth
  more than its size: the doc comment claimed the guard *"runs exactly once per
  redefining position"*, the two charges are in fact ADDITIVE, and the false
  half is a live false reject the issue deliberately does not own.
- **#876** at `2175749` — a local `xs:group ref=` / `xs:attributeGroup ref=`
  carrying `name` is rejected, the REFERENCE half of #684/#699/#706's family.
  **Unchanged, and the reason is a number: 0 witnesses across all 15470 suite
  `.xsd` files.** The repair round is the durable content — §3.4.2.3.3 clause
  2.1.4's `maxOccurs="0"` elision returns before `explicitContent` dispatches,
  so two of the three charges left the fault reachable. **#883** is what that
  discovered.
- **#858** at `d02aa59` — the three cast-shaped §3.12.6 CTA constructs
  EVALUATE. **Unchanged, and established rather than assumed**: its arbiter
  extracted all 100 distinct `alternative/@test` values in the pinned suite and
  showed no case was in the new decline's domain. Round 1 was REJECTED on two
  P1s, both reduced to failing witnesses; four `kind/gap` issues (#886–#889)
  were filed mid-flight so no marker shipped unowned.
- **#886** at `3867fb5` — `ta-props-correct` clause 2 over `xpath-valid` clause
  2 charged at last, at schema CONSTRUCTION time. **Unchanged.** It was
  overtaken by #858 off the same base and had to forward-merge one
  name-resolution rewrite onto another; warden's post-merge round then found a
  REAL over-charging false reject the merge had shipped — an order-dependent
  verdict on a byte-identical `{test}` — repaired at `613302a`.
- **#887** at `f1250c0` — the CTA comparator stops asking the VALUES which
  operators they admit and asks `xpath20.md` Appendix B.2's rows, through a
  160-row generated table. **Unchanged, MEASURED in both directions**: 80
  distinct `{test}` strings in the suite, none naming a type whose B.2 rows
  changed. The arbiter re-derived all 160 rows with its own parser rather than
  trusting `go generate`.

**Read the five together and they say one thing: closing a grammar gap
correctly can convert nothing.** Three of them are `area/xpath` and all three
are correctness work with no suite case in range; the two `area/parser` ones
found their real content in a repair round and a comment-only finding rather
than in a lane figure. Against that, the *previous* window's two `instance`
moves (#842 +4, #851 +3) both came from `{type table}` **plumbing** reaching
cases already in the lanes — not from grammar. **The seam that pays and the
seam that is being worked are not the same seam**, and the next planning action
below is about closing that gap rather than picking a sixth `area/xpath` issue.

Milestones, read from GitHub this pass.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 85 | 44 | **active** |
| M5 — Instance validation (XML) | 13 | 12 | **active** |
| M6–M12 | 0 | 0 | not filed |

M4 moved **83 → 85 closed** and **42 → 44 open** (#876 closed into it; #875,
#883 and #884 filed into it). M5 is unchanged, and so is M0–M2, which is M1 (3)
plus M2 (5): **`M0 — Scaffold` carries no issues at all**, which is why the row
has always been a group.

**M3's `open_issues` counter says 1 and the row above says 0. The row is right,
and this resolves a question the previous stamp left open — do not re-derive
it.** No item in the repository, issue or PR, carries the M3 milestone and is
open: `repos/{o}/{r}/issues?milestone=4&state=open` returns nothing, and a local
filter over the whole paginated issue-and-pull-request listing agrees — the
twelve items carrying M3 are all closed. The milestone's own `updated_at` is
`2026-08-17T15:47:15Z`, which **matches #875's `updated_at` to the second** — an
issue assigned M3 and moved to M4 without the counter decrementing. **A GitHub
milestone counter is not a source; the issue list is.**

Queue: **231 open issues — 206 `ready`, 25 `blocked`, 0 `needs-replan`,
2 `epic`** (both `blocked`, so both counted inside the 25), against
**312 closed**. 206 + 25 = 231 exactly, and **every one of the 231 carries a
queue label** — the class #773 and #774 fell into is empty this pass. Read the
milestone table as feature progress and not as the queue: **175** of the 231
carry no milestone (231 − 44 − 12).

**The move decomposes into sixteen filings and seven closures, and it reconciles
exactly**: 222 + 16 − 7 = 231. The closures are the five landings plus **#872**
(parked `needs-replan`, then closed `not_planned` when **#878** superseded it)
and **#878** itself. The filings are #875, #876, #878, #879, #881, #883, #884,
#885, #886, #887, #888, #889, #892, #893, #894 — fifteen from the develop loop's
own post-land passes — and **#896** from this pass.

**Eleven open issue bodies were CORRECTED this pass, and that is the headline
process fact.** The previous three `/backlog` passes each edited **zero**, on the
recorded belief that no faithful read-write path existed. That belief is false
and was falsified by measurement, not argument: **repository-scoped `gh api`
REST is byte-faithful in both directions here**, verified by a read-modify-write
round trip that came back identical (the proxy appends one Claude Code
attribution footer, and the append is idempotent, so a read-modify-write loop
does not stack them). The corrected bodies are **#79**, **#250**, **#625**,
**#668**, **#669**, **#748**, **#779**, **#840**, **#857**, **#892** and
**#893**, two of them epics. **#892** is the issue that makes the
fix durable; until it lands, `docs/ROUTINES.md:42-52` keeps telling every session
the corrupting channel is the only one.

**Two of the corrections were epics, and both had been wrong for weeks.**
**#79** (M4) carried `## Depends on: none (it is the dependency target)` while
wearing `blocked` — a label and a dependency list contradicting each other in
one line, recorded as a probe by six consecutive stamps and repaired by none of
them — plus a `Blocks:` list whose six targets (#46, #51, #52, #63, #70, #72)
had **all closed in July**, and a standing instruction not to start M4 ahead of
M3. **#250** (M5) still said *"Do not carve this into sub-slices yet"* — the
carve landed 2026-08-12 and twelve of its slices have shipped — and predicted
`validate.New(*xsd.Schema, ...Option)`, which is not the signature that shipped.
**An epic is the one issue kind nothing forces anybody to re-read**, and these
two are the demonstration.

**#893's mechanism was re-measured and is FALSIFIED as filed, which is why
re-measurement is in its own Acceptance.** It said `mcp__github__search_issues`
returns `total_count: 0` for every query and diagnosed a swallowed 403. Measured
today: `repo:kud360/goxsd8 is:issue is:open parser` → **0**, while
`parser prohibited attribute guard` scoped through the `owner`/`repo` parameters
→ **2 correct hits**, and `README Library section snippet does not compile` →
**#669, exactly right**. The tool does **natural-language semantic matching**; a
GitHub qualifier-syntax query is matched as literal prose and returns an honest
zero. **The harm survives and the diagnosis does not** — the fix is a calling
convention nobody wrote down, and the sharp edge the old framing hid is that a
**file path or identifier**, which all three mandating sites tell a session to
search for, is the input semantic matching is worst at.

**The unblock sweep moved nothing, for the seventh consecutive pass, and this
time it was run as a script rather than by eye.** All 25 `blocked` bodies were
parsed for `## Depends on`: **no open body names any of the seven issues that
just closed.** Sixteen name at least one open issue, eight name a trigger rather
than an issue, and the one flagged row was **#79** — repaired here, as above.
**The script's first version was wrong about three rows and the corrections are
worth more than the result**, so they are on #779: anchor the heading to the
start of a line (#16 and #548 both discuss `## Depends on` in prose) and ignore
struck-through text (#56 and #250 both record a discharged blocker that way).
With both fixed: exactly one flagged row, no false positives.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** — and fed, for the first time, from a
reproducible command rather than a hand-shaped file (see #840):

```
ISSUE  BRANCH         TIP AGE   VERDICT  REASON
822    wip/issue-822  54h57m0s  RETIRED  wip/issue-822: issue #822 is closed
859    wip/issue-859  1h1m0s    LIVE     wip/issue-859: tip pushed 1h1m0s ago, within the 2h0m0s claim TTL
872    wip/issue-872  20h58m0s  RETIRED  wip/issue-872: issue #872 is closed
```

**A develop session is in flight on #859 right now** — the tip was pushed an
hour before this survey ran, inside the claim TTL. Nothing about #859 is this
pass's to touch, and it is deliberately absent from the band below even though
its own thread would otherwise put it there.

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-859` and `wip/issue-872`. Nothing is EXPIRED and there is no
`parked/*` ref. As always, `wipsurvey` reads `ls-remote` rather than local refs
(PRINCIPLES 28) — this container is a fresh clone and holds no stale
remote-tracking ref, which the previous two stamps both did.

**`wip/issue-822` @ `cc2d54e` and `wip/issue-872` @ `0b34c21` are both RETIRED
and both deliberately kept.** Each was parked and each has been SUPERSEDED BY A
LANDING — #851 for the first, #878 (`4cd678a`) for the second — so each is
re-planning evidence about a decision that is now settled. They are never
force-pushed, never renamed, never a base to branch from, and **their deletion is
a human's call, not a session's**.

**Their content is verified absent from `main` by reading `main`, not by
counting commits.** `git diff origin/main...origin/wip/issue-872` answers
`no merge base` — the shallow-clone finding (#802), which forbids any
ahead/behind arithmetic here, so none is published. The content check needs no
merge base: that branch's tip is *"process: restore `WebFetch` as the body
re-read, with a fall-through"*, and `docs/WORKFLOW.md` on `main` today leads with
the repair path and carries no such fall-through. **The rejected rewrite is
absent because the arbiter rejected it**, which is the correct state.

**`go tool gapaudit`: 64 `GAP(` markers across 5 areas** — `xsd` 36,
`validate` 14, `xpath` 7, `xml` 4, `value` 3 — and it ran **with
reconciliation**, not census-only, because #840's recipe produced the
`kind/gap` list. **Group 1 is EMPTY: every marker in the tree has an open
tracking issue.** Against the previous stamp's 62 that is `xpath` +2, and it
reconciles against the two diffs exactly: #858 and #886 added three between
them, taking `xpath` to 8 at `d02aa59` (measured on #879), and **#887 retired
`holdsBetween`'s marker outright** rather than narrowing it, 8 → 7. Group 2's
nine entries are all `kind/gap` issues that never carried a marker — conformance
lane gaps and, in #398's case, a tracker that says so in its own title — which is
where that group is supposed to be permanent.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 206 of
them. Take from the top. **The previous band's row 1 (#872) LANDED, as #878**,
and row 2's first half (#858) landed too. The band below is re-cut rather than
carried, because five zero-movement landings, an eleven-body correction sweep
and sixteen filings all changed what is cheapest next.

**Row 1 is a lane slice and row 2 is `kind/process`, which INVERTS the previous
stamp's ordering on purpose.** That stamp put process first and was right to:
the body-edit blocker was real and it landed as #878. It is now down, and what
replaced it at the top is the five-landing lane freeze. **The process row does
not drop out** — it is row 2 with a cost that compounds every pass — but a queue
that has gone five landings without a number should be told where the number is
first.

| # | Issue | Why here |
|---:|---|---|
| 1 | #867 | **The only MEASURED lane figure in the whole queue: `schema` +2, and it is a floor.** After five landings that moved nothing, take the one issue that already carries a number. An `<annotation>` carrying `<annotation>` children is still ACCEPTED — a different s4s fault from #836's, being `<annotation>`'s own content model rather than `xs:annotated`'s cardinality. `annotB001` and `annotB005` are the suite's only two such documents, both `invalid`, both `fail`, and **neither declined**. The grounding is done and on #836's thread, and the acceptance any fix must preserve is already a green row in `parser/produce_annotation_test.go` |
| 2 | #892 + #668 + #840 | **`docs/ROUTINES.md:42-52` is false, and it is the most expensive false paragraph in the repo.** It tells every session the corrupting MCP channel is the only one available. Measured cost while it stood: **three consecutive `/backlog` passes edited zero issue bodies** and four hand-shaped the survey input, while `gh api` served both losslessly the whole time. This pass paid the discovery and corrected eleven bodies with it — **the next pass pays the tax again unless the document changes**. #892 owns the paragraph, #668 the CLAUDE.md spelling that 403s (`gh issue list` is GraphQL), #840 the producer recipe with its two traps. They land together or the recipe has no home |
| 3 | #883 | **The largest measured escape surface in `parser`, and #876 found it the expensive way.** §3.4.2.3.3 clause 2.1.4's `maxOccurs="0"` elision returns before ANY per-element grammar check, so an ENTIRE elided subtree is unvalidated and four fault classes escape — #876's own guard was one, which is why that landing needed a repair round. Unmeasured against the lane but the direction can only be up, and the neighbourhood is fresh: read `2175749` before the body |
| 4 | #868 | `complexTypeDecidable` declines a `<simpleContent>`/`<complexContent>` carrying NEITHER alternant — a grammar fault the producer rejects genuinely — and the `<simpleContent>` arm's diagnostic names a construct the author never wrote. **Six declined cases measured alongside it** by #836's post-land pass, `annotB030` among them, whose three prior records all misattributed it to `<xs:override>` production. Converts declines into decisions in `schema` |
| 5 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in **three** rules at once — the ·initial value· charge, the ID/IDREF binding and the ·key-sequence· member. `instance` candidate, unmeasured, direction can only be up because all three decline today. First step is an oracle question: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 6 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced, and its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. Same function family as #830, #836 and #867, all landed or banded; read those diffs first |
| 7 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. It **gates #56**, and it decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. #56's `## Depends on` now names it alone, #842 having been struck when it landed |
| 8 | #669 → #625 → #748 → #896 → #492 | **README's Library block, ONE row in the order the issues themselves name — now FIVE, because the 2026-08-18 libuser produced a fifth.** Splitting it across band rows is why it sat seven passes, and the five overlap by paragraph rather than partition cleanly, so whichever lands second rebases on the first. #669 fixes the "works TODAY" snippet that does not compile and the example list that omits `xsd/example_test.go`; #625 the `SchemaBuilder` pointer at closed #203; #748 the M5 block that denies a shipped API; **#896** the package doc that never says which accessor is the verdict; #492 folds `ParseReport` in. **#748 led the libuser report for the THIRD consecutive consultation.** The new one is #896's: `Result.Err()` is a walk-fault indicator, not a verdict, and a caller taking README's own `err`-named variable at face value **silently passes documents carrying real `cvc-*` violations** |
| 9 | #870 + #747 + #514 + #687 + #672 | **The CLI contract, all five decided BEFORE #472.** The 2026-08-18 cliuser reconfirmed every one and filed nothing new — which is itself the result: the gap is disclosure, not discovery. #870 is the one a user hits first (Quickstart's `go build ./...` writes no binary; the stub's own `go doc` remedy fails wherever an installed CLI runs), #747 the missing "Implemented today" paragraph, #514 typo-versus-unbuilt, #687 scoped help (now also carrying `goxsd8 -help validate`, the flag-first spelling), #672 `-version`. Each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 10 | #885 | **The scope rule that already cost a reject round, and today it survives only in a verdict comment.** #876's round 1 classified a surviving hole as out-of-scope pre-existing; the arbiter REJECTED and had to re-derive that it was not. The discriminator, in that arbiter's words — *a fault escaping an early return is a REPAIR when the offending element is the direct dispatch target of the function holding the return, and a SEPARATE ISSUE when it is not* — belongs in `docs/WORKFLOW.md`'s Scope section. One landing produced both answers (#876 the repair, #883 the follow-up), which is why the test earns its place |
| 11 | #820 + #797 + #600 | **The landing mechanics, and this window paid them again.** #830's LOG entry was lost by a squash-merge and re-landed by hand; #858's and #887's Next lists each had to warn the next session in as many words to re-verify the entry is in the diff. #820 is the emptiness check that reads PRESENT while the entry is absent, #797 is where a code-free iteration puts its entry — and #878, a `.md`-only landing, met that question again — and #600 is the one-append-point merge tax on a file now past 2.4 MB |

**Deliberately unbanded, and why.** **#859** is `ready` and would otherwise be
banded, but a develop session holds its lease right now. **#888**, **#889** and
**#894** are the three `area/xpath` gaps #858 and #886 filed; they are correct,
startable and deliberately below the fold, because three consecutive `area/xpath`
landings moved nothing and #889 states a warden pre-flight as its first step,
which is **#484**'s standing condition rather than a cheap next slice. **#871**
stays `blocked` on **#831**, which its arrival re-priced: #831 was filed as
*"startable and correct to do, not valuable to do next"* with `Ratchet:
unchanged`, and is now the precondition for §3.12.4 clause 1.1.3. **#881** is
`blocked` on the next `/retro` and is the third sighting of the same friction —
three full four-part gates, each with a ~370-450s conformance run, spent on one
prose bullet. **#875** is #706's own follow-up — that landing's comment-only
finding named a live false reject it deliberately did not own, and #875 owns it;
**#884** is #876's and #883's neighbourhood. Read each beside its parent, not
before it. **#843–#849** are the 2026-08-16
architecture audit's seven findings; **#843** is still the one whose cost of
delay is stated as increasing steeply. **#852** is `gapaudit`'s matcher, and it
dropped out of the band because the tool ran **with reconciliation and Group 1
empty** this pass — its cost is now hypothetical rather than paid. **#744**,
**#773** and **#721** are still held out on one shared condition, and **#484**
owns it.

**One file neighbourhood is moving fast enough that an issue body describing it
is the wrong thing to read — read the files.** `xpath/cta.go` and
`xpath/ctaparser.go` took **three landings in about thirty hours** (#858, #886,
#887), two of which had to forward-merge a sibling before judgment, and one of
which shipped a real false reject in the merge that only a post-merge warden
round caught. #888, #889 and #894 all open the same two files. That is a queue
shape the cartographer chose, and it is priced above by demoting all three.

### Next planning action

**Attribute the CTA cohort's banked `instance` failures before carving another
`area/xpath` slice.** Measured directly from
`conformance/testdata/expectations/` this pass, over the `CTA/` case-name
prefix: `schema` **23 pass / 7 fail** across 30 rows, `instance` **6 pass /
45 fail** across 51. **Forty-five banked `instance` failures in the cohort, and
nothing on record says why any of them fails.** Five `area/xpath` and CTA
landings in this window and the previous one produced +7 between them, all of it
from `{type table}` plumbing and none from grammar — so the question is no longer
which construct to support next, it is which of those 45 the engine already has
the parts for. **#858 is the demonstration that a grammar gap can be closed
correctly and convert nothing**, and #887 is the demonstration that a
correctness fix can be right and unobservable.

**The general form of that is #570, and this window is the argument for it.**
Bank a per-lane decline baseline so every landing announces the cases it just
made decidable; **#571** is its soundness half. The standing `schema` decline
count is **893** as of `c116408` and has not been re-derived since — it now
predates twenty-one landings, and it is by a wide margin the oldest measurement
this plan still argues from. **#836 is the standing warning about how to
re-derive it**: its forecast of +9 was a census taken over `msData/annotations/`
alone and the guard paid +53, so **an estimate is bounded by the population it
was taken over before it is bounded by anything about the cases** — state the
population beside the number or the number is not a measurement.

**The thing that stopped being true this pass is worth naming, because three
stamps in a row argued from it.** A `/backlog` can correct an issue body again:
the read path and the write path are both byte-faithful through
`gh api repos/{owner}/{repo}/...`, and eleven bodies were corrected here to prove
it, two of them epics that had been wrong for weeks. **That capability is
undocumented until #892 lands**, which is why it is band row 2 and not a
footnote — every pass that does not read this stamp will re-derive the false
belief straight out of `docs/ROUTINES.md`.

**The queue is 231 and the band is eleven rows, and the gap is not a backlog
problem.** `ready` means filed and unblocked; its size is an output and never a
target (#347). Every one of the 231 carries a queue label this pass, which is
the first time that class has been empty — but the hygiene classes #779 exists
to catch are not empty, and this pass found a **fourth** by hand: `blocked` with
a `## Depends on` that is present, non-empty, and explicitly declares no
dependency. #79 was that, for six stamps running.

Take from the top: **start at row 1 (#867)** for a measured lane figure, or
**row 2 (#892)** if the next session should stop the tax before another pass
pays it. Both are one session.

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
