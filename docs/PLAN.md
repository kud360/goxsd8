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

## Status — 2026-09-02 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **SIX** landings — the last band's rows 1 through 5, taken in order, plus the pair the last stamp excluded as in-flight — measured **`schema` +38 across two of them**, found **the census producing its first EXACT magnitude prediction** (#786 predicted 10 and banked +10), found the last stamp's armed trigger **resolving by NOT firing for the second consecutive stamp**, re-scoped **#999** against a landing that falsified its `## Spec` quote, corrected **#1156**'s census baseline, and folded the **FOURTEENTH** persona consultation — which, for the first time in this project's history, returned **CLI behaviour defects rather than documentation gaps**, filed here as **#1188** and **#1189**)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13868 | 1530 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### The census made an EXACT prediction, and the 584/588 near-match is a coincidence

**`schema` 13830 → 13868 (+38); `instance` flat at 11029; `datatypes` flat.**
Two of the six landings moved it and both are attributable to the line.

| landing | commit | lane movement | milestone |
|---|---|---|---|
| #1115 with **#441** | `8e89122` | unchanged | none |
| **#438** | `21849d5` | **`schema` +28** | none |
| #1129 | `ab3e2ba` | unchanged | none |
| #1023, closing **#625, #748, #492, #934, #896** | `2355fe0` | unchanged | #748 was M5 |
| **#786** | `74ea322` | **`schema` +10** | none |
| #472 | `69569fa` | unchanged | M4 |

**#1030's residual bucketing predicted 10 discoveries for the `<simpleType>`
naming none of §3.16.2.1's three alternatives bucket. #786 banked +10 exactly,
zero regressed, on `schema` alone.** That is the first time this project has
predicted a magnitude and hit it, and it is **not** the same evidence as the
584-predicted-588 pair that filed #1164: that one compares a whole-gate document
figure against a two-file expectation diff, this one compares one named bucket
against one lane on a landing whose entire executable change was `return false`
→ `return true` in one predicate.

**Never cite 584-predicted-588 as a prediction. The two counts measure sets that
do not correspond and their agreement is arithmetic accident (#1164).**
Re-derived at `adb6d57`, the commit the bucket was measured on: 670 residual
discoveries, of which **584** (`simpleContent-forbidden` 416,
`complexContent-forbidden` 168) declined at `anonymousComplexTypeDecidable`,
sitting in **580 distinct schema documents**. #1126's 588 flipped expectation
lines are **475 distinct documents** — 475 `schema` lines, exactly one per
document, plus 113 `instance` lines over 92 of those same 475. All 475 were in
the 580, so the bucket contains the movers and contains them entirely; the other
**105 bucket documents moved nothing**, their 130 cases still `fail` at
`4e1d49b`.

**No conversion in that chain is one.** 584 discoveries → 580 documents → 475
movers → 588 lines is **0.993** documents per discovery times a yield of
**0.819** movers per bucket document times a fan-out of **1.238** lines per
mover, and their product, 1.007, is the whole of the near-match. Nothing
measurable before a landing fixes any of the three: #786's exact 10 → +10 hit a
combined factor of 1.00 on a bucket that moved `schema` alone with `instance`
flat, which is a property of that bucket and not of the instrument.

**Read a residual bucket count as a candidate filter and never as a sort key —
the six per-class counts for the remaining 55 (10 / 9 / 14 / 11 / 9 / 2)
included.** Each bounds the documents a widening can reach and predicts no lane
movement, because the factors that convert documents to expectation lines are
measurable only after the landing. That is the same rule four stamps have held,
now with a case-by-case reconciliation under it rather than an open question.

**Repeat the reconciliation rather than re-quoting it**: extract `adb6d57` to a
scratch tree, name the reason at each arm of `anonymousComplexTypeDecidable`,
re-run `TestUnmappedCensusSoundAgainstShapeGate`'s loop recording that reason per
residual discovery, then map the keys `git diff` flips under
`conformance/testdata/expectations/` between `6123653` and `4e1d49b` to
documents through `censusRoot`.

**The last stamp's armed trigger resolved by NOT firing, second consecutive
stamp.** It read: *"if #438 lands and both lanes are flat, the thing to
re-examine is whether the suite's anonymous-type documents violate
`cos-nonambig`, `cos-element-consistent` or `ct-props-correct` clause 4 at
all."* `schema` moved +28 on 28 new rejections with zero regressed, so they do,
and the widening was right on grounds no measurement had established before.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and this
pass's 677-issue post-write fetch:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
732    wip/issue-732   237h17m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   415h15m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   187h37m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   381h17m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   309h30m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   248h35m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   192h57m0s  RETIRED  wip/issue-993: issue #993 is closed
1160   wip/issue-1160  37m0s      LIVE     wip/issue-1160: tip pushed 37m0s ago, within the 2h0m0s claim TTL
```

**`wip/issue-1115` is GONE** — deleted on merge, so the last stamp's one LIVE
lease closed cleanly and the namespace's only churn this window was a clean
open-and-close.

**ONE LIVE lease, and #1160 is mid-repair as this pass runs.** `wip/issue-1160`
carries two commits — `ba0c797`, then `ed0b0e2` after an **arbiter round-1
REJECT at 13:46Z** — with grounding, mason account, verdict and repair all
posted today. **#1160 is therefore NOT banded below.** It also **edits
`docs/PLAN.md`'s M5 section at `:585-612`**, the EIGHT-versus-TWELVE paragraph,
which is why this pass left those paragraphs untouched: they are a claimed
issue's deliverable, not this stamp's.

**All seven RETIRED refs closed `not_planned`** — re-read from `state_reason` on
this pass's own fetch — so they are parks and supersedes, their content is
*supposed* not to be in `main`, and none owes a supersede. Cloud containers
cannot delete remote refs, so these accumulate by design and are not a finding.
**Zero `parked/*`.** Both non-`wip` refs, `claude/eloquent-cerf-8jq9o6`
(`7841e98`) and `claude/eloquent-cerf-39rk64` (`0abeab6`), still pass
`git merge-base --is-ancestor … origin/main` and carry nothing.

**The shallow-clone premise is unchanged.** This container sees **50 commits** of
`origin/main`, so every retired ref's disposition came from GitHub rather than
`git log`. **#802** owns this and is open.

### Marker census

`go tool gapaudit` over this pass's whole 677-issue feed: **67 markers across 7
areas** — `xsd` 33, `validate` 17, `xpath` 6, `xml` 4, `parser` 3, `value` 3,
**`cmd` 1**.

**Census 66 → 67, and a SEVENTH area appeared.** #472 landed the first
`GAP(cmd)` in the project at `cmd/goxsd8/parse.go:67`, citing **#1185**, so it
sits in neither unreconciled group — the marker convention working on its first
use in a new package. Net of #1115's one deletion and #438's additions.

**Group 1 held at 17 and group 2 at 25**, both unchanged from the last stamp
across six landings.

**ZERO rows carry no annotation at all, so the tool's own filing rule selects
nothing and there are zero untracked GAP sites — fourth consecutive stamp.**

**Two of the seventeen surviving group-1 rows are instrument defects, not
ownership defects, and #1156 owns both** — unchanged, and **this pass corrected
that issue's one stale figure**: its Acceptance pinned census **66** before and
after, which the `GAP(cmd)` arrival falsified. Both groups are unmoved, so its
row deltas (group 1 17 → 15, group 2 25 → 24) stand exactly as filed.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 12 pages: **677 issues, 253 open, 424 closed**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **48** | **107** | active |
| **M5 — Instance validation (XML)** | **11** | **19** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **234 `ready`, 19 `blocked`** — every open issue carries
one or the other, verified mechanically, and the two sum to 253 with no gap. By
kind: `kind/refactor` 70, `kind/process` 56, `kind/gap` 51, `kind/tooling` 30,
`kind/bug` 25, `kind/story` 17, `kind/docs` 12, `kind/feature` 7, `epic` 2. By
area: `parser` 70, `meta` 69, `xsd` 58, `conformance` 32, `validate` 22,
`docs` 18, `value` 14, `cmd` 12, `builtin` 10, `xpath` 6, `xsderr` 3,
`loader` 2, `regex` 2.

**Both milestone counts moved for the first time in two windows, and NEITHER
mover was a lane mover.** M4 fell 49 → 48 on **#472** and M5 fell 12 → 11 on
**#748**, a README fix. **#438 (+28) and #786 (+10) — the only two landings that
moved a lane — carried no milestone at all.** That is the sharpest instance yet
of the floor caveat both milestone sections make: read neither count as its
lane's remaining work.

**`ready` overstates startable work by ONE, which is the smallest gap this
section has recorded.** #1160 is `ready` and claimed. The honest startable count
is **252**. The four-discharged-and-still-`ready` residue that stood for eleven
stamps is gone: **#1023 closed all five on 2026-09-02**, and #472's post-land
pass left **zero open issues carrying no state label**, verified again here.

### #1023 closed after ELEVEN stamps, and band position was the diagnosis all along

Ten stamps named it and did not band it; the eleventh banded it at row 3 and it
landed the next day. `2355fe0` closed **#625, #748, #492, #934, #896** as
`completed` — five zero-work API closes, no code, no doc, **eight seconds of
wall clock** against ten days of carrying. The cartographer cannot close an
issue as done, so naming the residue could never discharge it; only a band row
could, and the moment one existed the queue paid out.

**This is the cleanest evidence the band has produced that position is a
mechanism and not a description**, and it is the reason three of this stamp's
top five rows are again process work.

### Persona consultations — the FOURTEENTH, and the first ever to return CLI BEHAVIOUR

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. What it does is fold, and it folded.

**The last stamp predicted this exact result and named the mechanism.** It wrote
that #472 *"is still the landing that would change what a cliuser can observe at
all — the CLI tail remains seven issues about DOCUMENTING a stub and not one
about behaviour."* #472 landed. The next consultation returned **two behaviour
defects**, both filed:

- **#1188** — `parse`'s summary omits the `namespace:` line entirely for a schema
  declaring a `targetNamespace` and no components, while a schema with no
  `targetNamespace` and one component prints `namespace: (absent)`. **The spec
  fact settles the issue's shape and makes it more than a bug report**: §3.17.1's
  Schema component has no `{target namespace}` property, so `xsd.Schema`
  publishes no accessor for one and `namespacesOf` derives the list from
  component names because that is all the surface permits. Reporting the
  namespace is therefore an `xsd` surface addition and a **warden question**
  before it is a `cmd` change; documenting the derivation is the other branch.
  This is the residual #472's post-land pass reproduced and deliberately did not
  file, wanting exactly this judgment.
- **#1189** — `doc.go:61-64` claims `-help=true` and `-h=1` are *"usage errors
  naming the three spellings"* **wherever they stand**. True after a subcommand
  (`parse.go:33`'s `helpNotAFlagValue`); **false before one**, where both fall
  through to `noSubcommand` and name nothing. The answer a user hits first is the
  one the contract describes wrongly.

**One standing piece of evidence was FALSIFIED and is not carried forward.** The
last stamp banded #1007 partly on *"`goxsd8 parse -zzz foo.xsd` and `goxsd8 parse
foo.xsd` are byte-identical on stderr"*. #472 made them distinguishable — two
distinct texts under one `goxsd8: parse:` prefix — and the byte-identity was a
property of the stub, not of the subcommand. Recorded on #1007; the
`gen`-only argument is what survives, and it is the honest one.

**Four documentation findings reconfirmed, each UPGRADED in kind rather than
restated**: #1123 (now shown checkable from published docs alone — `go doc
./validate/jsonsrc` and `./validate/bersrc` say *"not yet implemented"*, so the
claim is self-refuting to a reader who obeys it), #1144 (upgraded from doc
inspection to direct reproduction: `go.mod file not found` outside the module —
and `main.go:76` already documents the fact README does not), #1143 (upgraded
from a file census to running the commands, which only became possible when
`parse` started working), #1089 (unchanged, plus the new datum that `# Argument
vocabulary` has **zero** hits in README). **#1185 came back a clean non-issue** —
disclosure accurate, behaviour matching — which narrows its disposition without
discharging it.

**libuser was given a narrow ask and one of its two answers corrected the ask
rather than an issue.** README's Library section is byte-unchanged since the last
consultation (`git log bfc650b..c720206 -- README.md` touches only the CLI
section), so the ten surviving tail rows were not re-run.

- **#1168 is NOT mooted, and the conflation was this pass's.** libuser confirmed
  that both `NewAttributeDeclaration` and `NewElementDeclaration` document the
  `InlineTypeDefinition`-wrapping-`ComplexType` reject and its charged class, and
  reported it as possibly discharging the issue. It does not: #1168's own body
  says *"nothing was lost on this landing, and that is the point"* — the claim is
  that `develop.md` step 4's trigger is unfalsifiable, not that the change went
  undocumented. **A T5 surface-quality question was asked of an issue about a
  process trigger**, and no persona can settle the latter, because the trigger
  lives in `.claude/commands/develop.md`. Recorded there so no future
  consultation is aimed at it again.
- **#1094's Acceptance is EDITED, and the edit is a house-convention finding a
  persona surfaced and the tree confirmed.** **Seven** constructors carry the
  formula *"loc is the source position charged to any rejection AND retained: Loc
  reports it back"*; `NewTypeAlternative` writes the same sentence with the second
  half **absent**. Non-retention is encoded as a missing clause, discoverable only
  by a reader who already knows the convention — and `TypeAlternative` exposes
  `Test`, `TypeDefinition`, `Annotations` and **no `Loc`**, confirmed against the
  tree for all three of #35's carriers. Branch (b) now owes that constructor an
  explicit *"not retained"*, not only a statement at the type.
- **#1122 reconfirmed and sharpened into its real cost**: a developer copying
  README's runnable snippet verbatim writes logic that treats
  reached-but-not-decided as a clean pass, contradicting `validate/doc.go`'s own
  stated semantics in the one direction STYLE 9 exists to keep from being silent.

**What the next consultation should be pointed at**, in the orchestrating
session's gift and not this pass's: **#720's landing**, which is band row 8 and
is the next thing that changes what a cliuser can observe. The library tail needs
no fresh pass until something moves README's Library section or the `xsd` query
surface — #1033 remains the one row no persona has ever looked at.

### Working band

**Re-derived from this pass's evidence.** The last band's rows **1–5 all landed,
in order** (#438, #1129, #1023, #786, #472) — the second consecutive stamp in
which the top of the band landed contiguously, and the first in which it was
consumed to the bottom of row 5. **#1160 is IN FLIGHT — take it and you collide
with a repair round mid-flight.** Take from the top; re-run `wipsurvey` first.

| # | issue | why here |
|---|---|---|
| 1 | #1174 | **SIX sightings in one window — the most-recorded friction in the log — and the tax falls on `/retro` itself.** #881 was ruled and closed 2026-08-23; #1145's entry re-raised it from scratch eight days later and five 2026-09-01 entries counted data points toward re-deciding a decided question, two of them both calling themselves *"the fourth"*. A `/retro` reading only its own window sees five sightings of live friction and no ruling, so the instrument that corrects process is the one being misinformed. **This pass probed its labelled class-generalization hypothesis and narrowed it**: #659 is cited nine times this month and is NOT a second instance — it is the correct one-clause form — so the leak is in rulings that CLOSE a question, not in issues that document a cost. One session, three named options, and its Acceptance forbids reopening the gate question. Banded on the compounding-tax rule, not on a lane |
| 2 | #1164 | **Fourth stamp holding *"candidate filters, not sort keys"*, and this pass handed it the measurement that makes the question answerable.** #786 predicted 10 and delivered **+10 exactly**, standing beside #1126's 584-predicted-588 across an unreconciled unit boundary — one clean single-lane single-bucket experiment and one whole-gate near-match, which cannot both be weighed until someone rules whether a document discovery and an expectation line are the same unit. It is the only thing that can turn #1051's four remaining bucket counts into sort keys, and every band until it rules pays the caveat |
| 3 | #1182 | **Two census buckets, 9 + 9, in the family that is five-for-five on DIRECTION.** `groupDecidable`'s and `modelGroupDecidable`'s bare-group declines are conservative rather than forced — the producer already rejects the prohibited-attribute forms as s4s violations, verbatim the argument #786 landed on today for +10. It is ON the census, so its counts are the ones #1164 will rule on. **Read 18 as a candidate filter, not a size** |
| 4 | #1181 | **Filed by #786's own arbiter round-2 ruling**, which reframed `simpleContentExtensionDecidable` as a conservative decline while accepting the landing beside it — an arbiter verdict is stronger provenance than a mason account. Ranked below #1182 on one ground only: it **did not appear in the 670-document bucketing at all**, and its own body records that as an open question, which makes it the less predictable slice rather than the safer one. Either order is defensible |
| 5 | #999 | **The `[tests that cannot fail]` pattern is two consecutive code landings deep and this is its only filed instance.** #786 repaired a guard fixture with a witness carrying the identical defect; #472 shipped two tests its own mason account called load-bearing that could not fail. Both times the arbiter found it by running a mutation mason had described and not executed; both times it cost a reject round. #472's post-land pass routed the *pattern* to `/retro` and rightly declined to file a fifty-seventh `kind/process` issue, which leaves this the one queue row that discharges any of it. **Re-scoped by this pass**: #472 falsified its `## Spec` quote (three diagnoses became four) and moved both mutation recipes, and the defect survived the re-cut *wider* — `leadingFlagFmt` interpolates its argument too |
| 6 | #1188 | **The first CLI BEHAVIOUR defect this queue has ever carried, found by an outsider on the subcommand #472 shipped four hours earlier.** `parse`'s summary reports no namespace for a schema that has one and `(absent)` for a schema that has none, and nothing in the three contract copies says the list is component-derived. **§3.17.1 gives the Schema component no `{target namespace}` property**, so the reporting branch is an `xsd` surface addition and a warden question, and the documenting branch is one sentence in three places — a genuine fork, which is why it is banded above the doc rows rather than with them |
| 7 | #1167 | **Two measured sightings, #414 and #1115, the second in this window.** `gapaudit` reconciles marker → issue and issue → marker; nothing audits prose that points *at* a marker or its file. #1115's stale-marker sweep asserted completeness and missed `xsd/complextype.go:1012` — inside an exported method's doc, in the same package as the deletion — which the arbiter found with one `grep -rn ownedtypefold`. PRINCIPLES 27 says a repeated grep wants a tool |
| 8 | #720 | **`goxsd8 validate` — unblocked 2026-09-02 when #472 landed, and the second of the two lifts #16 has carried since 2026-07-07.** Its `## Depends on` now reads `none` in the body, not only on the thread, and #472 answered the reserved-but-unbuilt exit-code question it deferred. It moves no conformance lane by construction (`go list -deps ./conformance \| grep -c cmd/goxsd8` is `0`). Ranked below the process rows because **#472 was the discontinuity and this is the second increment of the same ceiling** — but it is what the next persona consultation should be pointed at |
| 9 | #1156 | **Comment text only, and this pass corrected the one figure that had gone stale.** Two group-1 rows are the survey misreading the tree: `contentrestricts.go:742` names open #499 four paragraphs below the marker head where `paragraph()` never reaches, and `:1047` names nobody while #345 owns it. Baseline restamped to census **67** at `c720206`; the deltas are unchanged, and row 1 has a landed precedent in #815's repair `627ed25`. The #267-versus-#345 ownership ruling is already on both threads |
| 10 | #1136 | **A wrong-order defect in the function this family keeps widening, filed by the landing that widened it last.** Two element producers charge `src-element` BEFORE `checkS4SChildOrder` and `elementParticleTerm` charges AFTER, so a schema violating both gets whichever fault class its producer reaches first. #1099's own account records running the walk first as *"the load-bearing decision"*, because in that order the new check only ever converts an ACCEPT into a reject. **The rationale exists, recorded somewhere that is not the site.** Take it in either order with #1135, never together |
| 11 | #1007 with #1123 and #1189 | **Three issues, one session, three contract copies, one coupling test.** All three edit `cmd/goxsd8/doc.go`, `main.go`'s `usage` const and `README.md` together, and `TestUsageCoversContract` pins the first two by **sixteen** substrings — up from twelve, which #472 made worse and #398 owns. #1007 is now a clean one-of-three (`gen` alone names no exit code); #1123's claim is self-refuting to a reader who follows it; #1189 is a false sentence in the section both others touch. Taking them apart means rebasing the same three files three times |
| 12 | #1036 | **The silence #1029's landing exposed, carried a FIFTH stamp, and STYLE P3 does not permit it to stay.** A top-level `<xs:schema>` child outside the XSD namespace is neither charged as the §5.1 first-bullet grammar fault it is — §A's `<xs:schema>` content model has no wildcard arm, and `xs:openAttrs` admits foreign *attributes*, not foreign element children — nor reported through `parser.AssembledDocument.Unmapped`. **One settled disposition, either direction.** #1047's body already names this issue as the owner of the foreign-namespace skip, and #1133 rewrites the §5.1-first-bullet paragraph this decision would join |

**Below the band, and why**: **#1160 is IN FLIGHT**, the sole exclusion on that
ground. **#1183** and **#1171** are cheap and correct and move nothing; both sit
in `conformance/` and should be taken **opportunistically** by whoever takes
#1181 or #1182 rather than banded — #1183 makes #1051's stage-2 deletion three
predicates smaller. **#1140** is carried a **fourth** stamp having been ranked
*"take it now, while the rebase is free"* three times while two landings went
into that section ahead of it; **if the next stamp still finds it unbanded and
untaken, close it as accepted rather than carry it a fifth time** — that is a
disposition and carrying is not. **#1177, #1178, #1168, #1157, #1153** each have
one sighting, and #1177's hazard measured **zero** this pass, which is evidence
its instance was repaired and not its mechanism. **#1087** is still one sighting
after twenty-plus landings. **#1148** changes what `gapaudit` prints and settles
nothing about any marker. **#1102** states in its own Acceptance that its shapes
are **absent from the W3C suite**, so zero is the expected result; it wants a
warden pre-flight. **#1093, #1098, #1133, #1135** are correct and cheap and move
nothing. **#849** carries a `## Cost of delay` reading *"a steward re-ranking is
warranted"* — an unranked issue asking for a rank, which no cartographer can
supply without taking the steward's judgment; commented and routed to `/retro`'s
steward drift review as a **third** item beside #841 and #1080. **#1051** stays
`blocked` on three unfiled buckets. The persona-family tail is ranked on ordinary
grounds; **#1033** is still the only row no persona has ever looked at.

**There is no `Increasing` steward ranking anywhere in the band.** #849 is the
nearest, and its own body records the ranking as falsified and unreplaced.

### Next planning action

1. **Take #1174, on the compounding-tax rule, and hold the diagnosis to account.**
   **Trigger set here**: if #1174 lands and the next window's Friction clauses
   still queue questions that are already decided, the lookup diagnosis was wrong
   and the class is larger than discoverability — re-examine whether the defect is
   that a `/retro` ruling closes an issue at all, rather than where the ruling is
   written down.
2. **#1164 has ruled — the near-match is a coincidence — and band row 3 is still
   the sharpest test of what survives it.** **Trigger set here**: **if #1182 lands
   and `schema` moves by anything other than 18, read 18 as the upper bound the
   ruling says it is.** #786 predicted 10 and delivered exactly 10; #1182's two
   buckets predict 9 + 9 on the same instrument. A second exact hit would say the
   yield is 1.00 on single-lane single-bucket widenings specifically; a shortfall
   is the measured 0.819 reappearing. **Do not band on the counts either way** —
   see the ruling above.
3. **The CLI ceiling lifted exactly as the last stamp said it would, and the
   prediction is worth banking as a method rather than a lucky call.** Twelve
   consultations produced seven documentation issues about a stub; the thirteenth
   said the ceiling was the stub itself; #472 landed; the fourteenth produced two
   behaviour defects and falsified one standing piece of evidence. **Trigger**: if
   the consultation after #720 produces no behaviour findings, the ceiling
   explanation was wrong and the CLI queue's shape is about the personas' brief
   rather than about what the binary does.
4. **Three of the five unfiled gate widenings remain, down from five, and
   #1051's standing instruction is unchanged** — *"file them as they are taken
   rather than fanning them out speculatively now"*. #786 discharged one bucket
   and #1182 filed two more; what is left unfiled is the named `<group>` body
   outside its content model (14), the model group's non-particle child (11) and
   the `<redefine>` child outside its content model (2). #1164's verdict has
   landed and leaves that instruction standing: those three counts bound what a
   widening reaches and rank nothing.
5. **The human decision blocking #1002 is unchanged and is now carried for a
   NINTH stamp.** #1002 waits on a ruling between (a) a constitutional
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
   than a blank page. **This stamp corrected M6's remaining-CTA-subset paragraph**,
   which had carried #859 as open work for fifteen days after it landed at
   `ea0650a`, arguing about a stale lease on a branch whose issue had already
   closed.
7. **The unblock sweep measured a clean zero for the FIFTH consecutive stamp.**
   All 19 open `blocked` bodies were fetched over REST and their `## Depends on`
   sections read, plus a control grep of all 19 whole bodies for the twelve
   closures. Only #16 (#472, named as a hand-off in its own words), #841 (#438,
   already folded by that landing's post-land pass) and #1051 (#438 and #786,
   both already struck through) mention any of them, and none as a live
   dependency. The window's one unblock — **#720**, `blocked` → `ready` — was
   performed by #472's post-land pass on 2026-09-02, not here. Seven of the 19
   are **triggers rather than issues** (#79, #555, #692, #841, #925, #1002,
   #1080) and say so in their own `## Depends on`; **do not re-scan those on the
   next sweep.**

**Standing, and re-checked rather than restated.** Four unlanded corrections still
target one paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**,
**#646**, **#679**, **#912** — and whichever lands last rebases three times.
**The next `/retro` inherits seven**: #692, #925, #841, #1080, the
fold-the-five-species question (#635, #912, #609, #510, #646), the
`[tests that cannot fail]` pattern routed by #472's post-land pass, and **#849's
owed steward re-ranking**, which is new this stamp and is the third item for the
steward's drift review. **#841 is still the counter-example the steward-ranking
rule cannot reach**: a `kind/refactor` with a steward ranking, `blocked` because
its trigger has no mailbox, fired twice without a ruling. **#659 is closed and
cited nine times this month as an ongoing per-session cost** — checked and ruled
NOT a defect: every citation is the one-clause provenance form `chronicler.md`
asks for, and the datum is folded into #1174 rather than filed. The CTA cohort's
45 banked `instance` failures remain unattributed, twenty-second consecutive
stamp. `gate.yml` runs and is still not a required status check, which only the
repository owner can change.

**Environment, one witness each.** Repository-scoped `gh api` REST served every
read and **SEVENTEEN writes** here — **2** issues filed (#1188, #1189), **3**
bodies edited (#999, #1156, #1094) and **12** comments posted (#742, #849,
#1164, #1174, #1168, #1007, #1123, #1089, #1144, #1143, #1185, #1122). The
three counts are enumerated rather than summarized because the last four stamps
each recorded a hand-counted write ledger disagreeing with its own record;
2 + 3 + 12 = 17, and every number above is a thread this section names.
`gh issue list` and
`gh api --paginate` were not attempted. **The paginate recipe ran twice and
succeeded twice**: 12 pages both times, pages 1–11 full at 100 on both runs, page
12 short at **87** then **89** and re-requested each time per the recipe with no
larger answer coming back — the genuine-last-page case the retry arm falls
through. The post-loop fullness check passed on both runs, and the second was
taken after this pass's writes. The shallow clone truncated `origin/main` to 50
commits (**#802**). No conformance measurement was taken by this pass: the lane
table above is the committed expectations, which `docs/WORKFLOW.md` names as the
lane score (#1120).

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
