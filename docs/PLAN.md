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

## Status — 2026-09-01 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **SIX** landings — the whole top of the last band, rows 1 through 5 plus the row excluded as in-flight — measured **588 verdicts, every one of them from a single landing**, found **the last stamp's armed trigger resolving by NOT firing**, found the census residual's **584 standing beside #1126's 588 in units nobody has reconciled**, filed **#1164** for exactly that, corrected **#438**'s falsified premise, and folded a persona consultation — libuser and cliuser, run fresh against the tail this section pointed them at, ending the streak-of-twelve-then-one-skip that the first draft of this stamp recorded)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13830 | 1568 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### One landing moved 588 verdicts, and the gate — not the charge — was what had been withholding them

**`schema` 13355 → 13830 (+475); `instance` 10916 → 11029 (+113); `datatypes`
flat.** `git diff --stat 6123653 HEAD -- conformance/testdata/expectations/` is
588 insertions and 588 deletions across `schema.txt` and `instance.txt`, which is
the paste above, and `docs/LOG/2026-09.md` records all 588 as `fail` → `pass`
with **zero `pass` → `fail` on any lane**.

| landing | commit | lane movement | declared as |
|---|---|---|---|
| #1108 | `55c416c` | unchanged | tooling |
| **#1126** | **`d433f7f`** | **`schema` +475, `instance` +113** | **measured** |
| #1145 | `fe40c87` | unchanged | process |
| #815 | `93cd789` | unchanged | process, comment text only |
| #1120 | `690c201` | unchanged | process |
| #409 | `c5b789b` | unchanged | docs |

**#1126 deleted one predicate.** `conformance/schema.go`'s
`anonymousComplexTypeDecidable` withheld every non-implicit-content inline
`<complexType>` from the executor in both lanes; both element paths now route
§3.3.2.1 clause 1's inline `<complexType>` through `complexTypeDecidable`, the
predicate a top-level named type and an `<alternative>`'s inline one already
took. `grep -rn anonymousComplexTypeDecidable --include=*.go .` returns **zero**.

**The last stamp's trigger resolved by NOT firing, which is the second trigger in
this file's history to resolve at all.** It was armed as *"if #1126 lands and
`instance` is still flat, the thing to re-examine is whether #414 and #1116
widened anything the W3C suite actually contains."* `instance` moved +113, so
they had, and **two landings banked behind that gate are now attributed**: #414's
finalize folds (`54c13b3`) and #1116's cvc-complex-type clause 2 charge
(`9919faf`), each of which measured flat and each of which said in its own
`Ratchet:` trailer that the flatness was ENTAILED by this gate. **Flatness
declared as entailed was right both times, and the queue held the gate issue that
proved it** — that is the discipline paying out, not a lucky guess.

**The last stamp ruled the band has no magnitude signal at all. That ruling is
now in doubt and this pass filed the issue that settles it.** #1030's
2026-08-27 residual bucketing measured **584** document discoveries declining at
`anonymousComplexTypeDecidable`; #1126 flipped **588** expectation lines four
days later. **A document discovery at a gate predicate and an expectation line in
a lane file are different units, one document can reach both lanes, and nobody
has reconciled them** — so the near-agreement is either the first magnitude
instrument this project has ever had or an arithmetic coincidence. **#1164 owns
the reconciliation and must rule one way or the other**, because the same
bucketing carries live per-class counts for the 55 documents that remain
(10 / 9 / 14 / 11 / 9 / 2, only the first of them filed, as #786). Until it
rules, **do not band on those counts and do not cite 584-predicted-588 as a
prediction.**

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and this
pass's 665-issue fetch:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
732    wip/issue-732   213h6m0s   RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   391h4m0s   RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   163h25m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   357h6m0s   RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   285h19m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   224h24m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   168h46m0s  RETIRED  wip/issue-993: issue #993 is closed
1115   wip/issue-1115  32m0s      LIVE     wip/issue-1115: tip pushed 32m0s ago, within the 2h0m0s claim TTL
```

**ONE LIVE lease, and #1115 is mid-repair as this pass runs.** `wip/issue-1115`
carries one commit — `88ebe56`, *"xsd: NewAttributeDeclaration rejects an inline
ComplexType at {type definition} (#441)"* — and the thread carries an oracle
GROUNDING at 13:11Z, a mason account at 13:55Z and an **arbiter round-1 REJECT at
14:14Z**. **#1115 is therefore NOT banded below**: it is claimed, in flight, and
its body was left untouched by this pass for that reason. It also closes **#441**,
which is `ready` and must not be taken either.

**All seven RETIRED refs closed `not_planned`** — re-read from `state_reason` on
this pass's own fetch, not carried from the last stamp — so they are parks and
supersedes, their content is *supposed* not to be in `main`, and none owes a
supersede. Cloud containers cannot delete remote refs, so these accumulate by
design and are not a finding. **Zero `parked/*`.**

**Two non-`wip` refs, both ancestors of `main` and both carrying nothing.**
`claude/eloquent-cerf-8jq9o6` (`7841e98`) and `claude/eloquent-cerf-39rk64`
(`0abeab6`) each pass `git merge-base --is-ancestor … origin/main`. `wipsurvey`
reads the `wip/issue-<N>` namespace alone; that is its contract and not a gap.

**The shallow-clone premise is unchanged.** This container sees **50 commits** of
`origin/main`, so `git log --grep` reaches none of the seven closures and every
disposition above came from GitHub's `state_reason`. **#802** owns this and is
open.

### Marker census

`go tool gapaudit` over this pass's whole 665-issue feed: **66 markers across 6
areas** — `xsd` 33, `validate` 17, `xpath` 6, `xml` 4, `parser` 3, `value` 3. One
fewer than last stamp, and #815's landing deleted exactly one marker outright.

**Group 1 fell 28 → 17 and group 2 fell 34 → 25**, which is #815 landing and
nothing else: ten markers repointed at issues open today, one deleted, three
ruled unowned in prose naming no `#N`.

**ZERO rows carry no annotation at all, so the tool's own filing rule selects
nothing and there are zero untracked GAP sites — third consecutive stamp.**

**Two of the seventeen surviving group-1 rows are instrument defects, not
ownership defects, and #1156 owns both.** `xsd/contentrestricts.go:742` DOES name
open #499, four paragraphs and three blank comment lines below the marker head,
where `paragraph()` never reaches; `:1047` names nobody while #345 owns it. Both
were filed by #815's own post-land pass, with the #267-versus-#345 ownership
dispute ruled on both threads.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 12 pages: **665 issues, 253 open, 412 closed**.

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
kind: `kind/refactor` 68, `kind/process` 53, `kind/gap` 52, `kind/tooling` 27,
`kind/bug` 24, `kind/story` 22, `kind/docs` 12, `kind/feature` 7, `epic` 2. By
area: `parser` 70, `meta` 66, `xsd` 62, `conformance` 29, `validate` 25,
`docs` 22, `value` 14, `builtin` 10, `cmd` 10, `xpath` 6, `xsderr` 3,
`loader` 2, `regex` 2.

**BOTH milestone counts are unchanged across six landings, and that is this
stamp's sharpest instance of the floor caveat both milestone sections make.**
M4 held at 49 and M5 at 12 because **not one of the six landings carried a
milestone** — including #1126, the largest lane movement this project has
recorded. Read neither count as the lane's remaining work.

**`ready` overstates startable work by six.** #625, #748, #492 and #934 are
discharged in `main` and still `ready` (see below); **#1115 is claimed and in
flight**, and **#441** is `ready` but closes with it, so taking it would collide.
The honest startable count is **227**.

### #1023's five are now FIVE ZERO-WORK CLOSES — ELEVENTH stamp, banded, and confirmed by a reader who cannot see the tree

All five re-verified against `4e1d49b`/`5d216e9` and tabulated on
[#1023](https://github.com/kud360/goxsd8/issues/1023). **README has not been
touched since the last stamp** — `git log 6123653..HEAD -- README.md` is empty —
so every line number is the ninth stamp's, re-read rather than carried: #625 at
`README.md:182` (`#203` still returns zero hits); #748 at `:191`, `:203`, `:217`;
#492 at `:165` and `:176`; #934 at `:98`.

**#896 is no longer the fifth-and-different one.** Ten stamps carried it as *"a
short landing, not an API close"*, which was right while #56's post-land re-scope
left a verification owing. **That verification is now performed** — twice, once
by a source-blind libuser off the default `go doc ./validate` page and once
against `validate/doc.go:90-104`, where the three-accessor text survived the two
landings that touched the file since `3160813`. No gap survives, its second
Acceptance bullet does not fire, and there is no prose to write. The discharging
session writes **nothing**: five closes as `completed` and one command against
`34a8043`.

**The one thing ten stamps never did was band it, and the diagnosis they each
wrote is the reason.** The cartographer cannot close an issue as done —
`.claude/agents/cartographer.md` reserves that to the develop loop — and the
develop loop picks by band position, where five zero-work closes never outrank a
lane slice. **#1023 is band row 3 below.** If the next stamp still finds all five
open, the failure is no longer band position and `/retro` inherits a cleaner
question than it has had for ten passes.

### Persona consultations — the THIRTEENTH, after one skipped pass, folded into nineteen comments and one body edit

**The first draft of this stamp recorded NONE and pointed at the tail below; the
orchestrating session then ran both personas against exactly that tail and handed
the reports back, so this section is replaced rather than corrected.** The
cartographer role-plays no persona and does not spawn one (#416): it has read the
source, so a verdict it produced would launder an insider's opinion as an
outsider's. What it does is fold, and it folded.

**libuser** — README and `go doc` only, never the source, against the fifteen-row
library tail. **Five reported already fixed** (#492, #625, #748, #896, #934),
each reproduced against the published surface, which is the first outsider
confirmation of the #1023 residue and is recorded on that thread. **Ten confirmed
still open with no drift from the filed claim** (#513, #670, #688, #721, #854,
#1006, #1088, #1094, #1122, #1142), each carrying a dated re-measurement comment.

**cliuser** — a fresh build, against the seven-row CLI tail. **All seven held
unchanged**; the two standing dismissals were reconfirmed by seven adversarial
flag probes and are recorded on #16. **#1007 is now a FIFTH sighting and a
consultation rather than an observation** — see band row 11, whose rationale this
consultation changed.

**One persona inference was wrong in an instructive way, and correcting it edited
an issue body.** Reasoning from the published signature alone, libuser read
`NewTypeAlternative(loc xsderr.Loc, …)` as a schema location #1094's carriers
hold and decline to expose. The tree says otherwise: none of `Assertion`,
`XPathExpression` or `TypeAlternative` stores a `Loc`, and that constructor
accepts one only to build the two errors it may return, discarding it on the
success path. So #1094's branch (a) is a field-and-thread change on three types
rather than an accessor over held state — a session sizing it off the signature
would under-scope it — and branch (b) now owes an answer for the discarded
parameter. **#1094's `## Acceptance` is edited to say so.**

**What the next consultation should be pointed at**, in the orchestrating
session's gift and not this pass's: the tails above minus whatever #1023
discharges. **#472 is band row 5 below** and is still the landing that would
change what a cliuser can observe at all — the CLI tail remains seven issues
about documenting a stub and none about behaviour.

### Working band

**Re-derived from this pass's evidence.** The last band's rows **1–5 all landed**
(#1126, #1145, #815, #1120, #409) and so did the row excluded as in-flight
(#1108) — the last stamp recorded five of twelve landing SCATTERED through the
band; this one is the first in which the top five landed contiguously. **#1115 is
IN FLIGHT and #441 closes with it — take neither.** Take from the top; re-run
`wipsurvey` first.

| # | issue | why here |
|---|---|---|
| 1 | #438 | **#1126 turned this from a latent fail-open into a live one, over the 584 documents it just admitted, and this pass corrected the body that still said otherwise.** Three read-only finalize passes — `cos-nonambig`, `cos-element-consistent`, `ct-props-correct` clause 4 — range `s.types` and never visit an anonymous complex type reached only through an `InlineTypeDefinition`. Until `d433f7f` the conformance gate withheld every such document; it no longer exists, so those documents are in both lanes' executors with all three constraints unenforced on their anonymous types. #1126 admitted them on the express ground that a never-reached verdict is an UNDER-rejection and never a false accept, so nothing banked is corrupt — the exposure is live, not silent, and it is the direct successor of the landing that just paid +588. **Its `## Depends on` says #584 supersedes it and that #584's landing closes it; #584 is the §3.4.6 constraints plus both folds across every owning slot, and names neither `cos-nonambig` nor `cos-element-consistent` in its Goal. Rule which you are taking at grounding and take exactly one** |
| 2 | #1129 | **The landing contract this very pass runs without, for the seventh session in this window alone.** `docs/WORKFLOW.md`'s Landing preconditions 3 and 4 are unsatisfiable by construction for a PR that closes no issue, and the verifier clause names an agent that is not the one merging it. Six post-land PRs and this `/backlog` PR all paid it since the last stamp. **Ranked above every lane slice on the rule that a per-session tax compounds** — it is one session's work, it is fully specified, and it was band row 9 last stamp and went untaken |
| 3 | #1023 | **TENTH stamp, five zero-work closes, and band position is its own diagnosed cause — so this stamp supplies the missing act rather than the count.** #625, #748, #492 and #934 are discharged in `main`, re-verified at `4e1d49b` against a README untouched since the last stamp; **#896's owed verification is now performed and recorded, so all FIVE are zero-work closes** — the discharging session writes no code and edits no doc. The cartographer cannot close them and ten stamps of naming it changed nothing. Cheap and bounded: five closes and one command against `34a8043`, and together they take five rows off a `ready` queue this stamp measures as overstating startable work by six |
| 4 | #786 | **`simpleTypeDecidable`'s LAST decline, and #1126 handed it the measurement it has been waiting for.** `simpleTypeDecidable` declines a `<simpleType>` naming none of §3.16.2.1's three alternatives — a document `parser.simpleTypeBody` rejects outright, so the decline is conservative rather than forced. **#1126 measured the schema gate declining ZERO suite-valid documents, down from 372**, which is the strongest evidence this issue has ever had that admitting the shape cannot cost a banked win. Its body already carries the instruction to retire `TestSchemaExecutorDeclinesUndecidableSuiteCase` rather than repoint it a fourth time. The census predicts **10** documents — **read that as a candidate filter and not a size, until #1164 rules** |
| 5 | #472 | **The only `ready` `kind/feature` on M4, and the single ref that gates the CLI's entire remaining queue.** `goxsd8 parse` is the first non-stub subcommand; #720 (`goxsd8 validate`) waits on it alone, and #16 waits on #720. **The whole open CLI story queue — #16, #1003, #1007, #1089, #1123, #1144 and the CLI half of #1143 — is seven issues about DOCUMENTING a stub and not one about behaviour**, which is the ceiling this pass's cliuser consultation hit again: all seven held unchanged and it found no eighth, because there is no behaviour to find; this is the same compounding tax on the persona instrument that carried #409 through nine sightings to its landing this window. It moves no conformance lane and is banded on the two issues it unblocks and the ceiling it lifts |
| 6 | #1160 | **This file asserts EIGHT decided-and-wrong `instance` cases and the tree measures TWELVE, so `docs/PLAN.md`'s own accuracy is the defect.** Filed by #1120's post-land pass from a figure both mason and the arbiter reproduced independently to the character. Four cases are unaccounted and only `ST_targetNS00101m2_p` has no owner. **This pass did not fix the paragraph** — either Acceptance branch needs a measurement a cartographer cannot take without running the suite, and half-doing it would strip the issue of its cheaper branch — so `M5`'s section below carries a correctness marker and nothing more |
| 7 | #1153 | **The Survey input recipe is executable text that every survey in this repo depends on, that no gate part reads, and that has no test.** #1145 paid for two independent throwaway harnesses to judge a six-line change, and the rejected first fix dropped 760 of 1160 issues while every live run against the real endpoint passed. Its Acceptance is unusually complete — extract the recipe rather than copy it, mock `gh` per `(page, attempt)`, and six named scenarios. One sighting, but #1145's own banding rationale applies unchanged: the blast radius is total and the instrument reports something false |
| 8 | #1136 | **A wrong-order defect in the function this family keeps widening, filed by the landing that widened it last.** Two element producers charge `src-element` BEFORE `checkS4SChildOrder` and `elementParticleTerm` charges AFTER, so a schema document violating both gets whichever fault class its producer happens to reach first. #1099's own account records running the walk first as *"the load-bearing decision"*, because in that order the new check only ever converts an ACCEPT into a reject. **The rationale exists and is recorded somewhere that is not the site.** Take it in either order with #1135, never together — #1135 is a T4 extraction and this is a ruling |
| 9 | #1036 | **The silence #1029's landing exposed, carried a fourth stamp, and STYLE P3 does not permit it to stay.** A top-level `<xs:schema>` child outside the XSD namespace is neither charged as the §5.1 first-bullet grammar fault it is — §A's `<xs:schema>` content model has no wildcard arm, and `xs:openAttrs` admits foreign *attributes*, not foreign element children — nor reported through `parser.AssembledDocument.Unmapped`. **One settled disposition, either direction.** #1047's body already names this issue as the owner of the foreign-namespace skip, and #1133 rewrites the §5.1-first-bullet paragraph this decision would join |
| 10 | #1156 | **Two of the seventeen surviving group-1 rows are the SURVEY misreading the tree, not the tree drifting** — the same defect class as #1117, which landed last window. `contentrestricts.go:742` names open #499 four paragraphs below the marker head where `paragraph()` never reaches; `:1047` names nobody while #345 owns it. Its Acceptance carries the before figures (census 66, group 1 17, group 2 25) and the exact after (17 → 15, 25 → 24), and the #267-versus-#345 ownership ruling is already on both threads. Comment text only |
| 11 | #1007 | **FIFTH sighting, and the first in three windows that is a consultation rather than an observation: cliuser reproduced the unevenness from `-help` output alone, without sight of `doc.go`.** It also carries runtime evidence no observation can substitute for. `parse`'s blurb names exit 0 and 1; `gen`'s names none; `validate`'s names 0/1/2. `goxsd8 parse -zzz foo.xsd` and `goxsd8 parse foo.xsd` are byte-identical on stderr and both exit 2, so a script author cannot discover exit 2 by probing. **Take it WITH #1123**, which edits the closing sentence of the same file through the same three copies, and `TestUsageCoversContract` couples them by twelve substrings |
| 12 | #1140 | **Take it now, while the rebase is free.** `docs/ROUTINES.md`'s **Survey input** section spells a repository-scoped REST recipe and names no auth caveat, so a reader entering the file at `:104` never meets `:57-61` and one mason built a substitute feed on a `gh auth status` it should have ignored. **#1145 landed in that exact section this window and sharpened this issue rather than touching it**; two landings here would rebase against each other, and the second one has already happened |

**Below the band, and why**: **#1115 is IN FLIGHT** (`wip/issue-1115`, LIVE
lease, arbiter round-1 reject posted) and **#441 closes with it** — both excluded
for that reason alone. **#1164** was filed by this pass and has one sighting;
it is a ruling the band needs but not one the band can rank above work that is
already specified. #1102 states in its own Acceptance that its shapes are
**absent from the W3C suite**, so zero is the expected result; it wants a warden
pre-flight. #1093, #1098, #1133 and #1135 are correct and cheap and move nothing.
#1087 is still **one** sighting after twenty landings. #1148 and #1157 were filed
in the last two days by post-land passes and neither has a second sighting.
#1088, #1089, #1003, #1033, #1006, #1142, #1143 and #1144 are the persona-family
tail; **the consultation has now run and confirmed every one of them still open
with no drift**, so the bar they were held below is cleared and they are ranked
on ordinary grounds from the next stamp. Only #1033 went unlooked-at, its package
falling outside both tails.

### Next planning action

1. **Take #438 and give the anonymous complex type the three verdicts it now
   escapes, because #1126 put 584 documents in front of a fail-open that used to
   sit behind a gate.** **Trigger set here**: if #438 lands and both lanes are
   flat, the thing to re-examine is not the queue and not the census — it is
   whether the suite's anonymous-type documents violate `cos-nonambig`,
   `cos-element-consistent` or `ct-props-correct` clause 4 *at all*, which no
   measurement in this project has established. Rule that before widening a
   fourth pass on the same reasoning.
2. **#1164 must rule before any stamp cites 584-predicted-588 as a prediction.**
   The last stamp wrote that the band has no magnitude signal; four days earlier
   an instrument produced a figure that came within four of the result. **Both
   cannot stand.** Until #1164 reconciles the units, the six per-class counts for
   the remaining 55 documents are candidate filters and not sort keys — which is
   the same rule the last two stamps wrote for document counts, held one more
   window rather than relaxed on a near-match.
3. **Three of the top four band rows are process work, and that is a deliberate
   reading of the compounding-tax rule rather than a lane-starving accident.**
   #1129 is paid by every doc-only PR including this one, #1023 is at its eleventh
   stamp and its five rows are now all zero-work, and both are single sessions. **The rule says a tax that compounds
   outranks a lane slice, and this is what obeying it looks like** — if the next
   stamp finds the `schema` and `instance` lanes flat because of it, that is the
   cost to weigh, not evidence the rule is wrong.
4. **The five unfiled gate widenings stay unfiled, on #1051's own standing
   instruction** — *"file them as they are taken rather than fanning them out
   speculatively now"* — and #1164's verdict is what should reopen that decision.
   #786 is the one that is filed and it is band row 4.
5. **The human decision blocking #1002 is unchanged and is now carried for an
   eighth stamp.** #1002 waits on a ruling between (a) a constitutional
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
7. **The unblock sweep measured a clean zero for the fourth consecutive stamp.**
   All 20 `blocked` bodies were fetched over REST and their `## Depends on`
   sections read: **no open issue names any of this window's six closures**
   (#1108, #1126, #1145, #815, #1120, #409) **as an open dependency**. #1051
   names #1126 and had already struck it through — its remaining chain (#438,
   #786 and the five unfiled) is open, so it correctly stays `blocked`. Seven of
   the 20 are **triggers rather than issues** (#79, #555, #692, #841, #925,
   #1002, #1080) and say so in their own `## Depends on`; **do not re-scan those
   on the next sweep**. #16 is a hand-off, not a trigger, and its two lifts
   (#472, #720) are both open — #472 is band row 5.

**Standing, and re-checked rather than restated.** Four unlanded corrections still
target one paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**,
**#646**, **#679**, **#912** — and whichever lands last rebases three times.
**The next `/retro` inherits seven**: #692, #925, #841, #1080, the
fold-the-five-species question (#635, #912, #609, #510, #646), and the two
measured recurrences the last stamp handed it (#1109's and #815's) — **#815
landed this window, so its recurrence is now a closed case study rather than an
open one, which is the first of the seven to change state in three stamps**.
**#841 is still the counter-example the steward-ranking rule cannot reach**: a
`kind/refactor` with a steward ranking, `blocked` because its trigger has no
mailbox, fired twice without a ruling. **There is no `Increasing` steward ranking
anywhere in the band.** **Eleven open bodies cite #409 as a live sibling now that
it has closed** — #472, #492, #513, #625, #670, #671, #748, #896, #1003, #1123,
#1144 — and every one is provenance or an *"adjacent, not gating"* note; **none
sits in a `## Depends on`**, checked mechanically, so none is corrected and none
is a finding. The CTA cohort's 45 banked `instance` failures remain
unattributed, twenty-first consecutive stamp. `gate.yml` runs and is still not a
required status check, which only the repository owner can change.

**Environment, one witness each.** Repository-scoped `gh api` REST served every
read and **five writes** here; `gh issue list` and `gh api --paginate` were not
attempted. **The paginate recipe as `fe40c87` left it ran twice and succeeded
twice** — the first `/backlog` since #1145 landed. 12 pages both times, pages 1–11
full at 100 on both runs, page 12 short at **63** then **64** and re-requested
each time per the recipe with no larger answer coming back, which is the
genuine-last-page case the retry arm is supposed to fall through. The
post-loop fullness check passed on both runs. The second was taken after this
pass's writes. The shallow clone truncated `origin/main` to 50 commits,
which is why the retired branches' dispositions were taken from GitHub rather
than `git log` (**#802**). No conformance measurement was taken by this pass: the
lane table above is the committed expectations, and
`git diff --stat 6123653 HEAD -- conformance/testdata/expectations/` accounts for
every verdict in it — the instrument `docs/WORKFLOW.md` now names as the lane
score, which is what #1120 landed this window.

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
