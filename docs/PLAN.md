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

## Status — 2026-08-31 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin main` at `6c50c62`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **FIVE** landings — the top five rows of the last band, in near order — found **the trigger that stamp set FIRED on the cause it named**, found **the survey recipe silently truncating its own input by 448 issues**, found **#815's mechanism reproducing twice more while #815 sat open**, found **a session landing #1108 concurrently with this pass**, filed **#1142**, **#1143**, **#1144** and **#1145**, corrected **#815** and **#1108**, and folded the **twelfth consecutive** persona consultation)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10916 | 15445 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13355 | 2043 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### The trigger FIRED, on the cause it named, and the table barely moved

**`schema` 13353 → 13355 (+2); `instance` FLAT at 10916; `datatypes` flat.** Two
verdicts across five landings, against 162 across nine last stamp.
`git diff --stat c1e2cd7 HEAD -- conformance/testdata/expectations/` is two
insertions and two deletions in `schema.txt` and nothing else, which is the
paste above.

| landing | commit | lane movement | declared as |
|---|---|---|---|
| #1099 | `397842a` | `schema` **+2** (`MS-Element2006-07-15/elemP007`, `/elemP008`) | measured |
| #1116 | `9919faf` | `instance` unchanged | **ENTAILED** |
| #1097 | `32070b8` | unchanged | **ENTAILED** |
| #1109 | `48310e8` | unchanged | process |
| #1117 | `c12d83c` | unchanged | tooling, unreachable by construction |

**The last stamp wrote: *"if row 1 (#1116) or row 3 (#1099) lands flat, the
thing to re-examine is `conformance/schema.go`'s shape gates, which #414's flat
landing already implicated by name."* #1116 landed flat and its own `Ratchet:`
trailer names that gate — `anonymousComplexTypeDecidable` — as the reason.** The
trigger fired on the cause it predicted rather than on a surprise — the
previous stamp's trigger did not fire at all, so this is the first one in the
file to resolve by firing. **#1126 was filed by #1116's own post-land pass to
own the gate**, and it is band row 1 below.

**Two landings are now banked behind that one gate and neither could move a
verdict.** #414 (`54c13b3`, the finalize folds) and #1116 (`9919faf`, the
cvc-complex-type clause 2 charge) both widened what the processor decides for an
anonymous complex type; `conformance/schema.go` withholds every
non-implicit-content inline `<complexType>` from the executor, so no assembled
case can exercise either. **Flatness that is ENTAILED is not a null result and
both landings said so before measuring** — which is the discipline working, and
also the reason the queue now holds a gate issue rather than an unexplained
pair of zeroes.

**The invalid-corpus predictor is five-for-five on DIRECTION and just delivered
its smallest figure.** #1099 was banded on the same criterion that took #1076 to
+21 four days earlier — *"every affected document is in the suite's invalid
corpus"* — and returned **+2**. Nothing is falsified: the criterion has only ever
been claimed as a predictor of direction, and it predicted the direction. **What
this stamp adds is the other bound.** The last two stamps recorded that a
measured document count does not RANK candidates; this one records that the
corpus criterion does not SIZE them either. Between #1076's +21 and #1099's +2
sits no signal the band currently has. **Rank on the criterion, promise nothing
about magnitude, and stop reading a small figure as a failed prediction.**

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin main` at
`6c50c62` and this pass's 655-issue fetch:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
732    wip/issue-732   188h56m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   366h54m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   139h15m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   332h55m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   261h8m0s   RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   200h13m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   144h36m0s  RETIRED  wip/issue-993: issue #993 is closed
1108   wip/issue-1108  33m0s      LIVE     wip/issue-1108: tip pushed 33m0s ago, within the 2h0m0s claim TTL
```

**ONE LIVE lease, the first in three stamps, and a concurrent session is landing
#1108 while this pass runs.** `wip/issue-1108` carries one commit — `36c9e03`,
*"tools/gapaudit: drop CLOSED file resemblances, rank group-1 annotations"* —
and the thread carries a GROUNDING at 12:51Z and a MASON landing account at
14:03Z. **#1108 is therefore NOT banded below**: it is claimed, in flight, and
about to close. This pass edited its body anyway, because every absolute figure
in it predated #1117, and **posted a diff table on the thread naming exactly
what moved so the in-flight session is not misled by a body that changed under
it.** The rule the issue decides was not touched.

**All seven RETIRED refs closed `not_planned`** — re-checked through
`state_reason` on this pass's own fetch, not carried from the last stamp — so
they are parks and supersedes, their content is *supposed* not to be in `main`,
and none owes a supersede. Cloud containers cannot delete remote refs, so these
accumulate by design and are not a finding. **Zero `parked/*`.**

**Two non-`wip` refs exist now where there was one, and `wipsurvey` correctly
classifies neither.** `claude/eloquent-cerf-8jq9o6` points at `7841e98` and
`claude/eloquent-cerf-39rk64` at `0abeab6` — **both ancestors of `main`**, so
both are session scratch refs carrying nothing. `wipsurvey` reads the
`wip/issue-<N>` namespace alone; that is its contract and not a gap.

**The shallow-clone premise is unchanged.** This container sees **50 commits** of
`origin/main`, so `git log --grep` reaches none of the seven closures and every
disposition above came from GitHub's `state_reason`. **#802** owns this and is
open.

### Marker census

`go tool gapaudit` over the whole 655-issue feed: **67 markers across 6 areas** —
`xsd` 34, `validate` 17, `xpath` 6, `xml` 4, `parser` 3, `value` 3. Four fewer
than last stamp: #1116 retired three `validate` markers and #1099 retired one in
`parser`.

**Report 959 lines: 28 group-1 rows carrying 718 annotation lines, then 34
group-2 rows carrying 159.** Group 1 fell 30 → 28 and group 2 held at 34.

**ZERO rows carry no annotation at all, so the tool's own filing rule selects
nothing and there are zero untracked GAP sites — second consecutive stamp.**
That rule is unreachable by construction and #1108 bullet 4 owns saying so.

**#1117's landing is visible in this census and it did exactly what it
promised.** `parser/redefine.go:401` printed for two stamps as citing only
CLOSED #503/#504 while its live `kind/gap` owner **#744** sat in a sentence
`paragraph()` was eating, because `commentwrap` had reflowed `// #744 owns the
retirement` to a line start. **The marker never drifted; the survey was reading
it wrong.** That row is gone, and the eight rows still citing only a closed
issue were each re-checked against the same possibility before being counted.

**The unowned-marker family GREW while the issue owning it sat `ready`, and that
is this stamp's largest queue finding.** #815 was measured at thirteen rows and
is **fourteen**, re-composed as **six + eight** rather than four + nine:

| class | rows | change |
|---|---:|---|
| says *"unowned"* while an **OPEN** issue owns it | **6** | 4 → 6: `parser/produce_complex.go:2023` (#471) and `validate/cvcelt.go:178` (#1119) |
| cites **only a CLOSED** issue | **8** | 9 → 8: `parser/redefine.go:401` discharged by #1117, `:2317` renumbered to `:2351` |

**Both new rows are the mechanism reproducing, not new defects.** Each marker
predates the measurement; each owner was filed by a post-land pass that did not
then repoint the marker. **#815's own Notes name that seam and it has now been
observed operating twice more since the Notes were written.** Body and title
corrected this stamp; `#265` alone still accounts for six of the eight dead
citations.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes: **659 issues, 253 open, 406 closed**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **49** | **106** | active |
| **M5 — Instance validation (XML)** | **12** | **18** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **233 `ready`, 20 `blocked`** — every open issue carries
one or the other, verified mechanically, and the two sum to 253 with no gap. By
kind: `kind/refactor` 69, `kind/gap` 52, `kind/process` 53, `kind/tooling` 27,
`kind/bug` 24, `kind/story` 22, `kind/docs` 11, `kind/feature` 8, `epic` 2. By
area: `parser` 71, `meta` 64, `xsd` 62, `conformance` 29, `validate` 26,
`docs` 23, `value` 14, `builtin` 10, `cmd` 10, `xpath` 6, `xsderr` 3,
`loader` 2, `regex` 2.

**M4 fell 50 → 49 and M5 held at 12.** Two M4 issues closed (#1097, #1099) and
one opened into the milestone (#1133); #1126, #1135 and #1136 are M4-area work
carrying no milestone, which is why the count moved by one and the work by
four. **The census family replenished itself again** — #1099's
post-land pass filed #1135 and #1136 against the very function it widened. Read
both milestone counts as a floor and never as the lane's remaining work: #853's
+141 sat outside M5 for exactly this reason.

**`ready` overstates startable work by five.** #625, #748, #492 and #934 are
discharged in `main` and still `ready` (see below); **#1108 is claimed and in
flight**. The honest startable count is **228**.

### #1023's four are still four — NINTH stamp, and the count is now the finding

All four re-verified against `6c50c62` and tabulated on
[#1023](https://github.com/kud360/goxsd8/issues/1023). **README has not been
touched since the last stamp** (`git log c1e2cd7..HEAD -- README.md` is empty),
so where the numbers below differ from the eighth stamp's, the eighth stamp's
reading was wrong:

| issue | verified at `6c50c62` | last stamp said |
|---|---|---|
| #625 | PRODUCER-surface caveat at `README.md:182-184`; `grep -c "issues/203\|#203"` → **0** | `:182` |
| #748 | `:191` *"Instance validation runs today"*; `:203` `validate.New`; `:217` `xmlsrc.Validate` | `:184` / `:203` / `:216` — **two of three wrong** |
| #492 | `:165` carries `ParseReport`'s signature; `:176` what it lifts | `:165-167` |
| #934 | `:98` prints `[cvc-type]` as the outer rule | `:98` |

**#896 remains the fifth and remains a short landing, not an API close** — its
2026-08-28 re-scope made it a verification requiring the gate green.

**Nine stamps is a fact about the mechanism, not about the four.** Each has been
verified discharged on nine separate passes and each remains `open` and `ready`.
The cartographer cannot close them — `.claude/agents/cartographer.md` reserves
closing-as-done to the develop loop — and the develop loop picks by band
position, where four zero-work closes never outrank a lane slice. **Naming it
eight times has not broken the loop.** Recorded on #1023 rather than re-argued
here; it is `/retro`'s to diagnose.

### Persona consultations — the TWELFTH consecutive

Handed to this pass by the orchestrating session; the cartographer role-plays no
persona and verified every claim against the tree before recording it.
**Four filings, five threads updated, one persona claim corrected — the lowest
correction rate of any consultation so far**, against four wrong claims last
stamp. Both reports were accurate on every runtime probe.

**Filed.** **#1142** — `builtin/strict` exports one symbol and its doc prose
describes `math/big`, a `(coefficient, scale, sign)` triple and a seven-property
temporal model that no consumer can reach; the documented route out
(`value.Override` with a caller-owned mapping) is named in `value/doc.go:23-26`
and on `go doc ./value Override` and **nowhere in `builtin/strict`** —
`grep -rn "Override" builtin/strict/` returns **zero matches across the whole
package**, tests included.
**#1143** — README's two worked examples name **six** files that exist nowhere
in the repository (`order.xsd`, `items.xsd`, `order1.xml`, `order2.json`,
`order.xml`, and `order.xsd` again in the Library block at `:145`), and nothing
is offered as a substitute; the CLI half is moot today, **the Library half is
not** — `:191` opens *"Instance validation runs today"* and libuser ran it only
by inventing its input. **#1144** — `README.md:86` and `:231` both call
`go doc <import path>` authoritative and neither carries the
inside-the-clone precondition; from `/tmp` both fail with `go.mod file not
found`.

**#1145 was filed by this pass rather than by a persona, and it is the one that
would have wrecked this stamp.** Running the `docs/ROUTINES.md` **Survey input**
loop a second time to recount after this pass's writes, **page 4 returned 58
items and the loop stopped there** — 210 non-PR issues instead of 658, `gh` exit
status **0**, valid JSON, no signal of any kind. Re-requested seconds later that
same page returned 100, and an immediate re-run of the identical loop returned
all 12 pages. **`length < per_page` is the recipe's only stop condition and it
cannot tell a transient short page from a last one.** Every count in this
section would have been wrong by two thirds; it was caught only because a
cartographer compared two runs by eye.

**Reconfirmed, with the increment recorded and no re-scope.** **#409** — **ninth
sighting, by two personas across twelve consultations, none told the issue
existed.** The new increment is an in-repo contrast on ONE concept rather than a
whole-file property: `value/doc.go:192` writes *"implement an Emitter (API
frozen in M9; **not yet declared here**)"* while `codegen/doc.go:46` writes
*"# The Emitter seam (value.Emitter; API frozen in M9)"* — same phrase, same
symbol, and the caveat is `value`'s alone. **The hedge this issue asks for has
already been written by a sibling about the same symbol**, which settles the
wording question its Acceptance leaves open. **#16** — seven documented CLI
behaviours re-probed against a binary built at `6c50c62`, **all seven matched**,
including the `-q -v` row a persona got wrong last stamp; the `-q`/`-v`
precedence criterion stays **unobservable at runtime** and must be decided, not
discovered. **#1000**, **#721**, **#1119** each gained a cross-reference to a
filing above.

**One persona claim was wrong and is corrected inside #1142's own body.** libuser
reported *"no documented way to discover the concrete Go type a validated value
comes back as."* `go doc ./builtin/strict New` says *"The concrete value types
are unexported:"* and then enumerates, per type, which `value` capability
interfaces that type's value satisfies and which it deliberately does not, for
all twenty. **That is the second consecutive consultation in which a persona
read a package-level symbol listing and missed a full doc comment one level
down** — a pattern worth naming, and #1088 is where the runnable half of it
lives. The surviving half of the report is #1142's narrower claim.

**Two reports were dismissed with reasons rather than filed.** cliuser's item 1
(*"zero subcommands are functional"*) is **correctly and completely disclosed**
at `README.md:53`, in `-help`, and in `go doc ./cmd/goxsd8`, and the persona said
so; it is not a defect and must not be re-verified as one next pass. cliuser's
item 4 (seven adversarial flag probes, all matching) is a **confirmation**, and
it is recorded on #16 so the next consultation can spend its slots elsewhere.

### Working band

**Re-derived from this pass's evidence; not shifted.** **Five** of the last
band's twelve rows landed (#1116, #1109, #1099, #1097, #1117) and are gone.
**#1108 was row 6 and is IN FLIGHT — do not take it.** Take from the top;
re-run `wipsurvey` first.

| # | issue | why here |
|---|---|---|
| 1 | #1126 | **The band's lane row, and the only issue in the queue that can make two banked landings measurable.** `conformance/schema.go`'s `anonymousComplexTypeDecidable` withholds every non-implicit-content inline `<complexType>` from the executor, so #414's finalize folds and #1116's cvc-complex-type clause 2 charge sit behind one gate and neither moved a verdict. The last stamp's trigger named this gate before either landed, #1116's `Ratchet:` trailer named it again on the way out, and #1116's post-land pass filed this issue to own it. **Its Acceptance demands the movement be MEASURED and attributed here, and warns that the widening admits documents to BOTH lanes** — report `schema` in the same breath. A case flipping to failing is a real fold, match or derivation bug and is the finding, not a reason to restore the narrowing |
| 2 | #1145 | **The survey that is step 1 of every `/backlog` silently lost 448 issues today, with exit status 0 and valid JSON.** Filed by this pass from a live observation, not a hypothesis: the `docs/ROUTINES.md` recipe's `length < per_page` stop condition accepted a transient short page as the last one; a re-request of that page returned 100 and an immediate re-run returned all 12. **Ranked here on the reason the last stamp ranked #1117 above #1108** — this makes the instrument report something false — **and above #1117's because the blast radius is total rather than one row**: `wipsurvey` would report UNKNOWN, `gapaudit` would print every low-numbered owner as untracked, and every count in a stamp would be wrong. It happens rarely and costs everything when it does. The fix direction is deliberately left open |
| 3 | #815 | **Fourteen markers name no live owner, up from thirteen, and this pass measured the mechanism reproducing TWICE more while the issue sat `ready`.** Six say *"unowned"* while #471, #725, #782, #783, #812 and #1119 own them; eight cite only a CLOSED issue, six of those `#265`. Every one is confirmed both ways — a group-1 row AND a group-2 uncited tracker — which is the title's claim, measured. **The two new rows are not new defects**: each marker predates the measurement and each owner was filed by a post-land pass that did not repoint it, which is the seam this issue's Notes describe. **One landing, one convention, do not split.** Its own Notes say why it has sat: the cartographer files the owner and cannot edit code, the mason who wrote the marker has landed, so the repoint is owed by nobody. Line drift in the owners (#267 `:81`→`:90`, #345 `:236`→`:251`) is named in the Acceptance |
| 4 | #1120 | **Three different `instance`-lane failing figures on one tree, on the project's one rule, and this stamp again pasted the one PLAN.md's contract names.** File census **15445** (`lanestatus` and `instance.txt`), runner decline census **15428**, arbiter verdicts **15420**. `docs/WORKFLOW.md`'s *"Take a figure from the instrument that produces it"* does not say which instrument IS the lane score, so a `Ratchet:` trailer and a `docs/LOG` entry can honestly disagree by 25. Carried a second stamp because nothing in this window disturbed it — the reconciliation must be reproduced by arithmetic, not argued |
| 5 | #409 | **NINTH independent sighting, by two personas across twelve consultations, none told the issue existed — the most-corroborated doc defect in this repo, and it has sat in the band across consecutive stamps without being taken.** `codegen` and `codec` print `Generate`/`Target`/`AppendCanonical` in `go doc` code blocks while exporting **zero** symbols, and are the only two library packages for which `grep -in "not yet\|planned"` finds nothing. **Ranked here on the sessions it costs the persona instrument**, which is a compounding tax like any other: it consumes a report slot in every consultation that reaches it, and nine of twelve have. This stamp's increment removes the last open question — `value/doc.go:192` already writes the exact hedge, about the same symbol, in the same words. Four sites, one convention, **do not split** |
| 6 | #1115 | **#414's other follow-up, and the half its landing could not reach.** `ownedTypeFold` walks no ATTRIBUTE-side slot, so an anonymous complex type seated at an Attribute Declaration's `{type definition}` is folded for neither §3.4.2.4 clause 3 nor §3.4.2.5 clause 2 — and §3.2.1's simple-type-only typing is unenforced, so the shape is not even rejected. Owns `ownedTypeFold.schema`'s `GAP(xsd)`. One decision, two outcomes: make it unrepresentable, or fold it. Ranked below row 1 because it is the `xsd`-side half where row 1 is the harness-side one, and row 1 can move a lane where this cannot |
| 7 | #1036 | **The silence #1029's landing exposed, carried a third stamp, and STYLE P3 does not permit it to stay.** A top-level `<xs:schema>` child outside the XSD namespace is neither charged as the §5.1 first-bullet grammar fault it is — §A's `<xs:schema>` content model has no wildcard arm, and `xs:openAttrs` admits foreign *attributes*, not foreign element children — nor reported through `parser.AssembledDocument.Unmapped`. **One settled disposition, either direction.** #1047's body already names this issue as the owner of the foreign-namespace skip, and #1133 (filed this window) rewrites the §5.1-first-bullet paragraph this decision would join |
| 8 | #1136 | **A WRONG-ORDER defect in the function this family keeps widening, filed by the landing that widened it last.** Two element producers charge `src-element` BEFORE `checkS4SChildOrder` and `elementParticleTerm` charges AFTER, so a schema document violating both gets whichever fault class its producer happens to reach first. #1099's own account records that running the walk first is *"the load-bearing decision"* — because in that order the new check only ever converts an ACCEPT into a reject — and the other two producers do not do it. **The rationale exists and is recorded somewhere that is not the site**, which is the same defect #1135 names for the walks themselves; take them in either order but not together, since #1135 is a T4 extraction and this is a ruling |
| 9 | #1129 | **The landing contract this very pass runs without, and it bites the `/backlog` and post-land PRs specifically.** `docs/WORKFLOW.md`'s Landing preconditions 3 and 4 are unsatisfiable by construction for a PR that closes no issue, and the verifier clause names an agent that is not the one merging it. #1109 landed `48310e8` requiring the post-land PR at its two silent delegation sites — this is the half that landing exposed rather than closed. Cheap, and every doc-only pass pays it |
| 10 | #1007 | **Fourth sighting, and the first with runtime evidence that no observation can substitute for the missing sentence.** `parse`'s blurb names exit 0 and 1 (`doc.go:9-10`); `gen`'s names none at all (`:25-27`); `validate`'s names 0/1/2. `goxsd8 parse -zzz foo.xsd` and `goxsd8 parse foo.xsd` are byte-identical on stderr and both exit 2, so a script author cannot discover exit 2 by probing. **Take it WITH #1123**, which edits the closing sentence of the same file through the same three copies, and `TestUsageCoversContract` couples them by twelve substrings |
| 11 | #1143 | **Filed today, and the only new persona finding reached INDEPENDENTLY by both personas.** cliuser reported the CLI block's four absent filenames as a defect; libuser hit the same absence in the Library block, worked around it by inventing files, and reported the block as *working*. That split is the finding — the consumer who can run the code silently patches the gap. **The Library half is the live one**: `:191` says instance validation runs today and the snippet does run, against input the repository does not supply. Ranked below #1007 because #1007 has four sightings to this one's first |
| 12 | #1140 | **Take it WITH #1145 or immediately after.** `docs/ROUTINES.md`'s **Survey input** section spells a repository-scoped REST recipe and names no auth caveat, so a reader entering the file at `:104` never meets `:57-61` and one mason built a substitute feed on a `gh auth status` it should have ignored. Same section, same reader, same paragraph #1145's fix will edit; two landings here would rebase against each other |

**Below the band, and why**: **#1108 is IN FLIGHT** (`wip/issue-1108`, LIVE lease,
mason account posted) and is excluded for that reason alone — it would otherwise
sit near the top. #1102 states in its own Acceptance that its shapes are
**absent from the W3C suite**, so zero is the expected result; it wants a warden
pre-flight. #1093 (governingType's four silent exits), #1098 (the hardcoded
article), #1133 and #1135 are correct and cheap and move nothing. #1087 (the
arbiter's `## Acceptance` output form) is still **one** sighting after fourteen
landings. #1137, #1084, #1105 and #1111 are process singletons. #1122, #1123,
#1142 and #1144 were filed in the last two days and none has a second sighting.
#1088, #1089, #1003, #1033 and #1006 are the persona-family tail.

### Next planning action

1. **Take #1126 and settle the anonymous-complex-type gate, because two banked
   landings are waiting behind it and neither can be attributed until it
   moves.** This is the only band row that can move a lane. **Trigger set
   here**: if #1126 lands and `instance` is still flat, the thing to re-examine
   is not the queue and not the census — it is whether #414 and #1116 widened
   anything the W3C suite actually contains, which no measurement in this
   project has yet established for that shape.
2. **The invalid-corpus criterion predicts DIRECTION and nothing else, and this
   stamp bounds it from the other side.** #1076 delivered +21 and #1099 — same
   criterion, same family, four days later — delivered **+2**. Combined with the
   last two stamps' finding that a measured document count does not rank
   candidates, the band now has **no magnitude signal at all** and should stop
   implying one. Rank on the criterion; promise nothing about size; and never
   read a small figure as a failed prediction.
3. **#1145 is a survey-integrity defect that this pass caught by luck, and luck
   is not a control.** No gate part reads the issue feed, both surveys accept a
   partial one confidently, and the recipe's own stop condition is what
   truncated it. Until it lands, **a `/backlog` should run the paginate loop
   twice and compare the totals** — that is what caught it here, it costs one
   minute, and it is the only check that exists.
4. **#815's family grew while #815 sat `ready`, which is the second consecutive
   stamp in which a filed, specified, correctly-scoped issue failed to stop the
   behaviour it describes** — #1109 was the first. Two instances is a pattern
   and `/retro` should start from both rather than diagnose either afresh. The
   diagnosis exists in each body; what is missing is the step after filing.
5. **The human decision blocking #1002 is unchanged and is now carried for a
   seventh stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID with a per-case justification, and (b) holding §4.2.2's
   `vc:maxVersion` arm until real assertion evaluation lands. CLAUDE.md puts (a)
   beyond any agent — *"changes only via a human-filed issue"* — and (b) depends
   on **#1042**, filed and `blocked`. **No agent should attempt either.** The
   ruling is a comment on #1002; that comment is what moves it.
6. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3) and retires #719's `GAP(validate)` markers.
   **M6 tier 2 itself is uncarved** — `$value` binding, an F&O function library,
   typed comparison — and is too big for one issue. That carve is a `/backlog`
   act at the M6 opening, and #1042 is the thing it must slice around rather
   than a blank page.
7. **The unblock sweep measured a clean zero for the third consecutive stamp.**
   All 20 `blocked` bodies were fetched over REST and their `## Depends on`
   sections read: **no open issue names any of this window's five closures**
   (#1097, #1099, #1109, #1116, #1117) **as a dependency**. Seven of the 20 are
   **triggers rather than issues** (#79, #555, #692, #841, #925, #1002, #1080)
   and say so in their own `## Depends on`; **do not re-scan those on the next
   sweep** — each states the instruction in its own body. #16 is a hand-off, not
   a trigger, and its two lifts (#472, #720) are both open.

**Standing, and re-checked rather than restated.** Four unlanded corrections still
target one paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**,
**#646**, **#679**, **#912** — and whichever lands last rebases three times.
**#646 earned a fresh instance this stamp**: #1108's Acceptance carried three
generations of absolute figures, all stale, and was rewritten to demand a
measurement rather than quote one. **The next `/retro` inherits seven**: #692,
#925, #841, #1080, the fold-the-five-species question (#635, #912, #609, #510,
#646), #1109's measured recurrence, and now #815's. **#841 is still the
counter-example the steward-ranking rule cannot reach**: a `kind/refactor` with a
steward ranking, `blocked` because its trigger has no mailbox, fired twice
without a ruling. **There is no `Increasing` steward ranking anywhere in the
band.** The CTA cohort's 45 banked `instance` failures remain unattributed,
twentieth consecutive stamp. `gate.yml` runs and is still not a required status
check, which only the repository owner can change.

**Environment, one witness each.** Repository-scoped `gh api` REST served every
read and **14 writes** here; `gh issue list` and `gh api --paginate` were not
attempted. **The paginate recipe FAILED once and succeeded twice** — see #1145
and item 3 above; this is the first stamp in which the survey input itself was
the finding. The shallow clone truncated `origin/main` to 50 commits, which is
why the retired branches' dispositions were taken from GitHub rather than
`git log` (**#802**). No conformance measurement was taken by this pass: the
lane table above is the committed expectations, and
`git diff --stat c1e2cd7 HEAD -- conformance/testdata/expectations/` accounts for
every verdict in it — a figure whose instrument is itself band row 4 (#1120).

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
against the same packages sit outside it — #1126, #1135 and #1136 are all M4
work carrying no milestone today.

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
null one** — #1116's explanation is band row 1.

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
here is stale before the next landing (#646). It grew most because #913 decided `cvc-type` clause 3.1 — the commonest simple-typed-leaf
shape the lane had declined outright — which is the counterpart to #790's
lesson, not a contradiction of it: a slice that decides a *new* rule moves the
number far MORE than its rule count suggests when the declined shape is common,
and #913 moved it more than #790's descent did.

**TWO classes are decided and decided WRONG, and the first is a single case —
#771**, a root whose declaring schema is reachable only through the instance's
own `xsi:schemaLocation`. It was four,
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
