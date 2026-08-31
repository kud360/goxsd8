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

## Status — 2026-08-29 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin main` at `7841e98`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh 648-issue page-numbered `state=all` fetch taken after this pass's own writes. This pass follows **NINE** landings, found **NINE of the last band's twelve rows already cleared**, and re-derived the whole ordering rather than shifting it. It found **the lane table moving for the second consecutive stamp and by 162 verdicts**, found **the last stamp's #1076 trigger did not fire**, found **#1109's process break repeated TWICE after that issue was filed**, found **thirteen GAP markers naming no live owner where the last stamp's tool reported four**, filed **#1122** and **#1123**, corrected **#1109**, **#815** and **#748**, and folded the **eleventh consecutive** persona consultation)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10916 | 15445 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13353 | 2045 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### The table moved again, and the trigger the last stamp set did not fire

**`schema` 13332 → 13353 (+21) and `instance` 10775 → 10916 (+141)** — 162
verdicts across nine landings, and the second consecutive stamp with movement.
Two of the nine moved a lane:

| landing | commit | expectations diff | lane movement |
|---|---|---|---|
| #1076 | `5fbbf4b` | `schema.txt` 21±21 | `schema` **+21** |
| #853 | `2310710` | `instance.txt` 141±141 | `instance` **+141** |

`git diff --stat 032d402 7841e98 -- conformance/testdata/expectations/` totals
162 insertions and 162 deletions across the two files, which is the paste above.

**#1076 was the last stamp's named test and it passed.** That stamp said: *"#1076
is that test — the same mechanism at three new elements, but UNMEASURED, where
all three of the delivered rows arrived with a count. If #1076 lands flat, the
thing to re-examine is whether 'in the suite's invalid corpus' is doing the work
the census was credited with."* It landed at **+21**. So #1030's census
generalizes off the family it was derived from, and *"the shape is in the suite's
invalid corpus"* is now a **four-for-four** predictor of direction. It remains a
predictor of direction only: #1076's body promised no figure at all, which is
the discipline the last stamp asked for and got.

**#853 is the larger result and it was UNMEASURED when it was banded.** The last
stamp put it at row 6 with *"run `GOXSD_DECLINES=1` and count before promising a
figure"*, and it delivered **+141** — the largest single `instance` move since
#913's +9409, from an issue whose own body predicted nothing. That is the second
consecutive stamp in which an unmeasured candidate outperformed the measured
ones, and it is worth stating plainly: **a measured document count bounds the
shape; it does not rank the candidates.** Two stamps of evidence now say the
census is a filter, not a sort key.

**The other seven landings were lane-flat and declared it.** #56 (`3160813`, the
CTA withhold into `Result.Unevaluated`), #1082 (`88e8eae`, the clause 3.1 union),
#1043 (`5d3d222`, the skip-wildcard decline), #1060 (`28a0ee3`) and #1062
(`58470bd`, both `gapaudit`), #414 (`54c13b3`, the two finalize folds) and #1066
(`15654cb`, the CLI blurb) are seam, tooling and doc work. **#414's flat result
is a finding, not a null** — its own `Ratchet:` line records that the widening
reaches no banked case while `conformance/schema.go`'s
`anonymousComplexTypeDecidable` still declines the shape, which is what makes
band row 1 below a live candidate rather than a repeat.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin main`
at `7841e98` and this pass's 646-issue fetch:

```
ISSUE  BRANCH         LEASE AGE  VERDICT  REASON
732    wip/issue-732  140h52m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  318h51m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846  91h12m0s   RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872  284h52m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  213h5m0s   RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  152h10m0s  RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993  96h32m0s   RETIRED  wip/issue-993: issue #993 is closed
```

**Zero LIVE, zero CLAIMED, zero `parked/*` — the namespace is entirely startable,
for the second consecutive stamp.** Nine landings arrived and left no ref: GitHub's
auto-delete-on-merge is working. **All seven RETIRED refs closed `not_planned`**
— re-checked through `state_reason` on this pass's own fetch, not carried from the
last stamp — so they are parks and supersedes, their content is *supposed* not to
be in `main`, and none owes a supersede. Cloud containers cannot delete remote
refs, so these accumulate by design and are not a finding.

**One non-`wip` ref exists and `wipsurvey` does not classify it, correctly.**
`git ls-remote --heads origin` returns `main`, these seven, and
`claude/eloquent-cerf-8jq9o6` — which points at **`7841e98`, byte-identical to
`main`**, so it is a session scratch ref carrying nothing, not work. `wipsurvey`
reads the `wip/issue-<N>` namespace alone and says nothing about it; that is the
tool's contract and not a gap.

**The shallow-clone premise bit again.** This container sees **50 commits** of
`origin/main`, back to 2026-08-24, so `git log --grep` cannot reach any of the
seven closures. Every disposition above came from GitHub's `state_reason`.
**#802** owns this and is open.

### Marker census

`go tool gapaudit` over the whole 646-issue feed: **71 markers across 6 areas** —
`xsd` 34, `validate` 20, `xpath` 6, `xml` 4, `parser` 4, `value` 3.

**Group 1 is no longer empty and no longer lying — #1060 landed, and the number
it exposes is 30, not four.** Last stamp's group 1 printed `(none)` while four
markers named no issue; `28a0ee3` made a citation the only thing that keeps a
marker out, and the row count went `(none)` → **30**. Every one of the four the
last stamp read by hand is now printed by the tool. **This is the single largest
correction any survey in this repo has produced, and it is what row 7 below now
ranks on.**

**The 30 rows resolve into three classes, and only one of them is a filing
candidate:**

| class | rows | disposition |
|---|---:|---|
| cites **only a CLOSED** issue (`#51`, `#230`, `#265`×5, `#501`, `#503`) | **9** | folded into **#815** as its new section B |
| says *"unowned"* while an **OPEN** issue owns it | **4** | #815's original table, re-measured: #725, #782, #783, #812 |
| genuinely unowned, or owned by an issue filed this window | 17 | each already carries a `kind/gap` tracker filed by its landing's post-land pass |

**Exactly ONE row carries no annotation at all** — `validate/cvcelt.go:154`, whose
owner **#1119** was filed hours earlier by #853's post-land pass and whose
Acceptance already requires the number be written into the marker. So the tool's
filing rule as it then read (*"only a row with NO annotation at all is a
candidate for filing"*, replaced in #1108) selects one row, and that row is
already owned. **Zero untracked GAP sites. First time that has been true.**

**#265 alone owns five of the nine dead citations**, and its successors — #267,
#345, #413, #584 — sit in group 2 as trackers no marker cites. **`gapaudit`
prints both halves of that and cannot join them**, which is #815's title claim,
now measured rather than predicted.

Group 2 lists **35** open trackers with no surviving marker, up from eight, for
the same reason: #1060 raised the bar on both sides.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes: **648 issues, 247 open, 401 closed**.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **50** | **104** | active |
| **M5 — Instance validation (XML)** | **12** | **18** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **227 `ready`, 20 `blocked`** — every open issue carries
one or the other, and the two sum to 247 with no gap. By kind: `kind/refactor` 68,
`kind/gap` 54, `kind/process` 50, `kind/tooling` 27, `kind/bug` 24,
`kind/story` 19, `kind/docs` 10, `kind/feature` 7, `epic` 2. By area:
`parser` 70, `meta` 62, `xsd` 62, `conformance` 28, `validate` 27, `docs` 20,
`value` 14, `builtin` 9, `cmd` 9, `xpath` 6, `xsderr` 3, `loader` 2, `regex` 2.

**M4 grew by two and the growth is the instrument working.** Four M4 issues closed
(#1076, #1082, and #414/#853 outside the milestone) and six opened — #1097, #1098,
#1099, #1102, #1115, #1116 — every one filed by the post-land pass of a landing in
this window. **The census family replenishes itself as it is worked** and now has
four landed members; see the M4 section below.

**`ready` overstates startable work by four, not five.** #625, #748, #492 and #934
are discharged in `main` and still `ready`. **#896 is no longer one of them** — its
body was re-scoped on 2026-08-28 into a *verification* whose "Done when" requires
the full gate green, so it is a short landing and not an API call. The honest
startable count is **223**.

### #1023's five are now FOUR plus a verification — corrected this stamp

All four were re-read against `7841e98` by this pass, and the line numbers drifted
from the seventh stamp's `032d402` reading and are corrected in the table on
[#1023](https://github.com/kud360/goxsd8/issues/1023#issuecomment-5462950445):

| issue | verified at `7841e98` |
|---|---|
| #625 | `README.md:182` carries the PRODUCER-surface caveat; `grep -c "issues/203\|#203" README.md` → **0** |
| #748 | `README.md:184` — *"Instance validation runs today"*; `:203` calls `validate.New(schema, backend)`; `:216` `res, err := xmlsrc.Validate(…)` |
| #492 | `README.md:165-167` carries `ParseReport`'s full signature and what it lifts |
| #934 | `README.md:98` prints `[cvc-type]` as the outer rule wrapping `[cvc-datatype-valid]` |

**#896 is the correction, and it is the eighth stamp that would have got it
wrong.** #56's post-land pass rewrote its body on 2026-08-28 (`3160813`, PR #1092):
the finding is now *"a verification, not a documentation change"* and its Done-when
requires the verification to be **run and stated with the gate green**. A bare
`-f state=closed` call would skip an acceptance the issue acquired after #1023's
was written. **Four API closes plus one short landing** discharges #1023, and
neither is the cartographer's act.

**#748's body was corrected** — it quoted `README.md:126-133`'s denial text as
current, and `34a8043` replaced the whole block. An agent starting from that body
alone would have gone looking for a paragraph that no longer exists.

### Persona consultations — the ELEVENTH consecutive

Handed to this pass by the orchestrating session; the cartographer role-plays no
persona and verified every claim below against the tree before recording it.
**Two filings, seven threads updated, four persona claims corrected.**

**Filed.** **#1122** — `README.md:217-219`'s validation snippet names `res.Err()`
as the only incompleteness signal (*"a non-nil res.Err() means the assessment is
INCOMPLETE, so an empty Violations() then proves nothing"*) while
`validate/doc.go:101-104` says an empty `Violations()` beside a non-empty
`Unevaluated()` is **also** not a pass; `grep -n "Unevaluated" README.md` returns
nothing, and #56's landing this window made the omission reachable on an ordinary
conditionally-typed document. **#1123** — `cmd/goxsd8/doc.go:64-66` closes the
contract with *"every capability here is reachable through the public packages,
and the README documents both routes"*, which is **false for JSON and BER**:
`validate/jsonsrc` and `validate/bersrc` export **zero** symbols each and
`grep -n "jsonsrc\|bersrc" README.md` returns nothing, while README names the
`-format xml|json|ber` vocabulary at `:78` and runs `order2.json` at `:67`.

**Reconfirmed, with the increment recorded on the thread and no re-scope.**
**#409** (eighth sighting, two personas) gains a mechanical signal: `go doc ./codec`
and `./codegen` end with **no symbol index block at all**, where every other
library package ends with one — and that silence sits *below* the signature blocks
a reader meets first. **#1007** (fourth sighting) gains its runtime half: `goxsd8
parse -zzz foo.xsd` and `goxsd8 parse foo.xsd` are **byte-identical on stderr and
both exit 2**, so no observation an author can make today reveals that exit 2
exists — the documentation is the only channel and it is the one omitting it.
**#1089** re-probed: `goxsd8 -help | grep -i "go doc\|github"` is still empty
while every error path prints the repo URL (`main.go:59`). **#1088** unchanged.
**#1006** re-verified: `backendtest.Run(` is still called once, on the module's own
backend; `value/example_test.go` still carries one `Example` asserting no
capability.

**Four persona claims were wrong and are corrected on the threads, which is why
the verification step exists.** (1) libuser reported `value.Backend`'s method set
as *"fully elided"* and five accessors — `Matcher.Next`/`Accepting`,
`AttributeUse.DeclarationName`, `Schema.ContentMatcher`, `Result.Unevaluated` — as
*"named only in prose, never in a signature"*. **Every one prints a full
signature**: `go doc ./value Backend`, `go doc ./xsd Matcher`, `go doc ./xsd
AttributeUse`, `go doc ./xsd ContentMatcher`, `go doc ./validate Result`. The
persona read the package-level index, where `go doc` collapses **every** type in
**every** Go package to `type X struct{ … }`. Recorded on **#1088**, whose
Acceptance must stay on the *runnable* half. (2) libuser reported
`type Unevaluated struct{ … }` as leaving a reader unable to tell a method from a
field or to learn what it carries; `go doc ./validate Unevaluated` prints `Loc()`,
`Msg()` and `Rule()` under a doc comment explaining why the fields are unexported.
The **README** half of that report survived and is **#1122**. (3) cliuser reported
*"confirmed binary silently accepts `-q` and `-v` together"*; it does not —
`goxsd8 -q -v` exits **2** with `no subcommand given`, the leading-flag branch,
reached before any flag is parsed. The criterion is **unobservable at runtime**,
not merely undocumented, and is corrected on **#16**. (4) libuser attributed
*"Planned mapping (M8 — not yet implemented)"* to both `jsonsrc` and `bersrc`; it
is `jsonsrc`'s alone, and `bersrc/doc.go:9` reads *"# Mapping (contract; detailed
design lands with M11)"* — a weaker marker, which #409's body already spells
correctly. The contrast the sighting rests on survives.

**Two claims were true and are covered by decisions already taken, so neither was
filed.** cliuser's *"exit 2 conflates usage errors with I/O errors"* argues against
a contract **#472** already fixed in as many words (*"Usage / IO errors (no schema
argument, unreadable file, bad flag) → exit 2"*) — recorded there as an uncoached
outside report of what that decision costs a CI script. cliuser's *"`-backend` has
no stated default"* is carried by **#16**'s `gen` bullet, which gains the in-module
contrast: `README.md:146` states the library's equivalent default inline
(`parser.WithBackend(backend), // default: builtin/strict`) one screen away.

**#1006's `money` report was dismissed with reasons rather than filed.**
`README.md:158-160` is prose with inline code spans, not a snippet; `money` is a
stand-in for the reader's own backend in a sentence that never claims to compile.

### Working band

**Re-derived from this pass's evidence; not shifted.** **Nine** of the last band's
twelve rows landed (#1076, #1082, #1043, #1060, #1062, #853, #414, #56, #1066) and
are gone. Take from the top; re-run `wipsurvey` first, though the namespace is
currently empty.

| # | issue | why here |
|---|---|---|
| 1 | #1116 | **The band's `instance` row, and #414's landing six hours ago is what made it startable.** `validate`'s `attributePropertiesFolded` declines cvc-complex-type clause 2 on every ANONYMOUS ·governing type definition·, and it declines for an `xsd`-side under-report **#414 closed at `54c13b3`** — so what was a hedge is now a pure over-decline. Retiring it can only WITHDRAW a decline: unchanged-or-upward by construction. **UNMEASURED, and its body says so and names why**: `conformance/schema.go`'s `anonymousComplexTypeDecidable` is the second gate, still declining the non-implicit inline `<complexType>` forms, and retiring only this one may leave the movement hidden — which is exactly what happened to #414. Run the gate with `-v` and bank what the lane gives; a flat figure is a result to explain, not a pass. The identity-constraint half (`walk.idAttributes`, `icCheck.fieldAttributes`) moves with it |
| 2 | #1109 | **Banded on the sessions it costs, and this pass measured the cost compounding after the issue was already filed.** THREE post-land passes committed straight to `main` with no PR — `0b42115` (#1060), `b5a2e06` (#414), `4f9647c` (#1066) — against `docs/WORKFLOW.md`'s *"Nothing is ever committed directly to `main`."* **Two of the three happened AFTER #1109 was filed**, six and eight hours later; it was created at 01:35:43Z, two minutes after the first. Its own Notes named the trigger — *"if a future post-land search finds more instances, that changes this from a stated-but-unenforced rule to a systematic gap"* — and this pass fired it. **A stated sentence is now known not to hold**, which is why its Acceptance was widened here to require the mechanical check alongside. Body and title corrected this stamp; the detection must query `commits/{sha}/pulls`, because half the window's post-land subjects carry no `(#N)` and two of those DID land through PRs |
| 3 | #1099 | **The successor to the mechanism that moved `schema` +21 four days running, at the site that landing left behind.** `src-element` clause 2.2 is charged nowhere, so an `xs:element ref="x"` carrying an XSD-namespace child other than `xs:annotation` is silently ACCEPTED — and the `GAP(parser)` marker **#1076 itself landed** at `parser/produce_complex.go:2029` still reads *"Unowned: no issue tracks it yet."* Every affected document is in the suite's INVALID corpus, which is now a **four-for-four** predictor of direction. **UNMEASURED**, and after #853 that is not a demerit: two consecutive stamps have had an unmeasured candidate outperform the measured ones. Either charge the clause or rule it not this producer's and rewrite the marker permanently |
| 4 | #1097 | **A WRONG rejection, and #1076's landing made it unreachable rather than fixing it.** `simpleTypeBody`'s more-than-one-alternative branch charges `ruleSrcSimpleType` for a condition **none of `src-simple-type`'s four clauses states** — a rejection with no rule behind it — and since #1076 no consumer can read the charge at all, so the defect is now silent as well as wrong. A wrong decision outranks a decline, and a charge citing a clause that does not exist is the worst kind: it survives review by looking like a citation. Ranked below #1099 because #1099's shape reaches the suite and this one's reachability is what has to be established first |
| 5 | #1117 | **`gapaudit` reports something FALSE, and this pass is the one that would have been misled.** `paragraph` (`tools/gapaudit/main.go:306`) ends a marker's text at any comment line beginning `#`, so a `#N` citation that happens to wrap to line start is read as a Markdown heading and DROPPED — the marker then reads as citing nothing and lands in group 1 as untracked. Reproduced live twice in one session by its filer. **Ranked above #1108 for the reason the last stamp ranked #1060 above #1062**: this makes the tool report something false where that makes it report something noisy. The survey is step 1 of every `/backlog`, so a false row costs a filing decision every pass |
| 6 | #1108 | **Banded on the sessions it costs, and this pass paid it in full.** `gapaudit`'s group 1 went `(none)` → **30 rows carrying 834 annotation lines** when #1060 landed, and **416 of those lines are CLOSED *file* resemblances** — an annotation class that is neither an owner nor an action, since #1060 itself established that a path is named as readily to EXCLUDE a site as to own it. The report is **1068 lines** and yields **one** actionable row. The cartographer runs this tool every pass and read all 1068 lines here to extract the census above; nothing else in the queue will ever lift that. Its Acceptance is measured on both sides — 834 → 418 annotation lines, 1109 → 693 report lines — and it names the keep/drop split rather than a blunt filter |
| 7 | #815 | **Thirteen markers name no live owner, where the last stamp's tool could see four, and this stamp widened the issue to hold all of them.** Four say *"no issue owns its retirement yet"* while #725, #782, #783 and #812 own them — every one confirmed BOTH ways here, as a group-1 row and as a group-2 uncited tracker. **Nine more cite only a CLOSED issue** (`#51`, `#230`, `#265` five times, `#501`, `#503`), and for at least three the live successor (#248, #267, #345) is itself a group-2 row. **One landing, one convention, do not split.** Its own Notes say why it has sat: the cartographer files the owner and cannot edit code, the mason who wrote the marker has landed, so the repoint is owed by nobody. Three issue bodies' line numbers drifted the other way (#267 `:81`→`:90`, #345 `:236`→`:251`) and are named in the Acceptance |
| 8 | #1120 | **Three different `instance`-lane failing figures on one tree, on the project's one rule.** File census **15445** (`lanestatus` and `instance.txt`, the figure pasted above), runner decline census **15428**, arbiter verdicts **15420** — all on `2310710`, all in this window. `docs/WORKFLOW.md`'s *"Take a figure from the instrument that produces it"* does not say which instrument IS the lane score, so a `Ratchet:` trailer and a `docs/LOG` entry can honestly disagree by 25. **This section's own table is the file census by PLAN's stated contract**, so this pass is compliant and the ambiguity is still live. The reconciliation must be reproduced by arithmetic, not argued |
| 9 | #1115 | **#414's other follow-up, and the half its landing could not reach.** `ownedTypeFold` walks no ATTRIBUTE-side slot, so an anonymous complex type seated at an Attribute Declaration's `{type definition}` is folded for neither §3.4.2.4 clause 3 nor §3.4.2.5 clause 2 — and §3.2.1's simple-type-only typing is unenforced, so the shape is not even rejected. Owns `ownedTypeFold.schema`'s `GAP(xsd)`. One decision, two outcomes: make it unrepresentable, or fold it. Ranked below row 1 because it is the `xsd`-side half where row 1 is the `validate`-side one, and row 1 can move a lane where this cannot |
| 10 | #1036 | **The silence #1029's landing exposed, carried a second stamp, and STYLE P3 does not permit it to stay.** A top-level `<xs:schema>` child outside the XSD namespace is neither charged as the §5.1 first-bullet grammar fault it is — §A's `<xs:schema>` content model has no wildcard arm, and `xs:openAttrs` admits foreign *attributes*, not foreign element children — nor reported through `parser.AssembledDocument.Unmapped`. **One settled disposition, either direction.** Adjacent to and distinct from rows 3 and 4: #1047's body already named this issue as the owner of the foreign-namespace skip |
| 11 | #409 | **Eighth independent sighting, by two personas, none told the issue existed — the most-corroborated doc defect in this repo.** `codegen` and `codec` print `Generate`/`Target`/`AppendCanonical` in `go doc` code blocks while exporting **zero** symbols, and are the only two library packages for which `grep -in "not yet\|planned"` finds nothing about the surface shown. Four sites, one convention, **do not split**. This stamp added the mechanical signal: both pages end with **no symbol index block**, the one thing distinguishing them from every sibling, and it is below the signature blocks a reader meets first |
| 12 | #1007 | **Fourth sighting, and the first with runtime evidence that no observation can substitute for the missing sentence.** `parse`'s blurb names exit 0 and 1 (`doc.go:9-10`); `gen`'s names none at all (`:25-27`); `validate`'s names 0/1/2. `goxsd8 parse -zzz foo.xsd` and `goxsd8 parse foo.xsd` are byte-identical on stderr and both exit 2, so a script author cannot discover exit 2 by probing. **Take it WITH #1123**, which edits the closing sentence of the same file through the same three copies, and `TestUsageCoversContract` couples them by twelve substrings |

**Below the band, and why**: #1087 (the arbiter's `## Acceptance` output form) was
row 10 last stamp and is unpaid — still **one** sighting, and nine landings passed
through the arbiter this window without a second. #1102 states in its own
Acceptance that its shapes are **absent from the W3C suite**, so zero is the
expected and reportable result; it is a design question with a settled interim
ruling, not a bugfix, and it wants a warden pre-flight. #1093 (governingType's
four silent exits) and #1098 (the hardcoded article) are correct and cheap and
move nothing. #1084, #1105 and #1111 are process singletons. #1122 and #1123 were
filed today and neither has a second sighting. #1088, #1089, #1003, #1033 and
#1006 are the persona-family tail.

### Next planning action

1. **Close the four, and land the fifth. EIGHTH stamp, and the composition
   changed under it.** #625, #748, #492 and #934 are discharged in `main` — each
   re-verified here at `7841e98` with corrected line numbers, all tabulated on
   **#1023** — and are four `gh api repos/kud360/goxsd8/issues/<n> -X PATCH -f
   state=closed -f state_reason=completed` calls, the develop loop's act and not
   the cartographer's. **#896 is no longer one of them**: its 2026-08-28 re-scope
   made it a verification requiring the gate green, so it needs a short landing.
   Until both happen `ready` reads 227 where 223 is true.
2. **The census instrument is a FILTER, not a sort key, and two stamps now say
   so.** #1076 delivered its predicted direction (+21) and #853 — banded
   UNMEASURED at row 6 — delivered **+141**. Band on the shape being in the
   suite's invalid corpus, which is four-for-four; **stop treating a measured
   document count as a ranking signal**, and never quote one in a `Ratchet:`
   line. **Trigger set here**: if row 1 (#1116) or row 3 (#1099) lands flat, the
   thing to re-examine is `conformance/schema.go`'s shape gates, which #414's
   flat landing already implicated by name.
3. **#1109 is the first process issue in this queue to be measured recurring
   after it was filed, and that is a fact about the queue, not about #1109.** A
   filed, specified, correctly-scoped issue did not stop the behaviour twice in
   eight hours. Whatever `/retro` concludes about post-land landing discipline
   should start from that rather than from a fresh diagnosis — the diagnosis
   exists and was ignored, which is a different failure from not having one.
4. **The human decision blocking #1002 is unchanged and is now carried for a
   sixth stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID with a per-case justification, and (b) holding §4.2.2's
   `vc:maxVersion` arm until real assertion evaluation lands. CLAUDE.md puts (a)
   beyond any agent — *"changes only via a human-filed issue"* — and (b) depends
   on **#1042**, filed and `blocked`. **No agent should attempt either.** The
   ruling is a comment on #1002; that comment is what moves it.
5. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3) and retires #719's `GAP(validate)` markers.
   **M6 tier 2 itself is uncarved** — `$value` binding, an F&O function library,
   typed comparison — and is too big for one issue. That carve is a `/backlog`
   act at the M6 opening, and #1042 is the thing it must slice around rather than
   a blank page. #56's landing this window closed the last M6-adjacent item that
   was startable without it.
6. **The unblock sweep measured a clean zero for the second consecutive stamp,
   and the method is worth keeping.** All 20 `blocked` bodies were fetched over
   REST and their `## Depends on` sections read: **no open issue names any of
   this window's nine closures as a dependency**. #1051's body already records
   #414 as DISCHARGED — its post-land pass wrote that in — and correctly stays
   `blocked` on #438 (584 documents) and #786. Eight of the 20 are **triggers
   rather than issues** (#79, #692, #841, #925, #1002, #1080, #555, #16) and say
   so in their own `## Depends on`; **do not re-scan those on the next sweep** —
   each states the instruction in its own body.

**Standing, and re-checked rather than restated.** Four unlanded corrections still
target one paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**,
**#646**, **#679**, **#912** — and whichever lands last rebases three times. **The
next `/retro` inherits six**: #692, #925, #841, #1080, the fold-the-five-species
question (#635, #912, #609, #510, #646), and now **#1109**'s measured recurrence.
**#841 is still the counter-example the steward-ranking rule cannot reach**: a
`kind/refactor` with a steward ranking, `blocked` because its trigger has no
mailbox, fired twice without a ruling. **There is no `Increasing` steward ranking
anywhere in the band.** The CTA cohort's 45 banked `instance` failures remain
unattributed, nineteenth consecutive stamp. `gate.yml` runs and is still not a
required status check, which only the repository owner can change.

**Environment, one witness each.** Repository-scoped `gh api` REST served **every
read and 12 writes** here without a failure, exactly as `docs/ROUTINES.md` says;
`gh issue list` and `gh api --paginate` were not attempted. **The paginate recipe
held for the first time in three stamps** — #1062 landed at `58470bd`, replacing
the fixed `seq 1 9` with a stop-on-short-page loop, and it read all **12** pages
where the old recipe would have been short by three. The shallow clone truncated
`origin/main` to 50 commits, which is why the retired branches' dispositions were
taken from GitHub rather than `git log` (**#802**). No conformance measurement was
taken by this pass: the lane table above is the committed expectations, and
`git diff --stat 032d402 7841e98 -- conformance/testdata/expectations/` accounts
for every verdict in it — a figure whose instrument is itself band row 8 (#1120).

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

**A FOURTH member landed on 2026-08-28 and it was the family's first PREDICTION,
not its first measurement.** **#1076** (`5fbbf4b`, `schema` **+21**) carried the
same missing s4s content model at `xs:element`, `xs:attribute` and
`xs:simpleType`, and unlike #1046/#1047/#1048 it arrived with **no measured
document count at all** — the previous `/backlog` banded it explicitly as the
test of whether *"the shape is in the suite's invalid corpus"* does the work the
census had been credited with, and set the trigger that a flat landing would
falsify. It did not land flat. **That criterion is now four-for-four on
direction**, and the practical consequence is stated in the Status section: the
census is a filter on candidates, not a sort key among them.

**M4 grew 48 → 50 in the same window, and every one of the six new issues came
from a post-land pass.** #1076 and #1082 closed; **#1097** (`simpleTypeBody`
charges `ruleSrcSimpleType` for a condition none of `src-simple-type`'s four
clauses states — a rejection with no rule behind it, and since #1076 no consumer
can read it), **#1098** (`checkS4SChildOrder`'s message hardcodes the article
"a"), **#1099** (`src-element` clause 2.2 charged nowhere, owning the very
`GAP(parser)` marker #1076 landed), **#1102**, **#1115** and **#1116** opened.
**The family's shape is changing as it is worked**: #1047 and #1076 both widened
`checkS4SChildOrder`, and three of the six new issues are defects *in* that
widening rather than further sites for it. A producer that decides more is a
producer with more to get wrong, and the census does not name that class.

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

**#853 dwarfs all of them and forces the paragraph's own framing to be
restated. It took `instance` 10775 → 10916 (+141) at `2310710` on 2026-08-29** —
the largest single move this lane has recorded since #913's +9409 — by deciding
`cvc-elt` clause 5.1 for an EMPTY element whose declaration carries a `{value
constraint}`. **It carries no milestone**, so by the bookkeeping above the
outside-M5 total is now **180**, of which #853 alone is 141. That figure is not
a finding about M5's carve; it is a finding about the LABEL. #853 decides a
`cvc-` rule at assessment time on an XML instance, which is M5's definition of
its own scope, and it sat outside the milestone only because a `kind/gap` filed
by a post-land pass never acquired one. **Read the M5 milestone count as a floor
and never as the lane's remaining work** — the same caveat M4's section makes,
for the same mechanical reason, and the `instance` lane is where it costs most.

**#853 also repeats #913's lesson rather than #790's, and the band had it
right for the wrong reason.** It was banded UNMEASURED, with an instruction to
run `GOXSD_DECLINES=1` before promising a figure, and it outperformed every
candidate that arrived with a count. A slice that decides a *new* rule on a
commonly-declined shape moves the number far more than its rule count suggests;
one empty-element-with-a-default shape is common enough in the suite to be worth
141 documents. Two consecutive stamps now show an unmeasured candidate beating
the measured ones, which is recorded in the Status section as a change to how
the band is ordered.

**#1043 landed in the same window, inside M5, and moved nothing — correctly.**
`5d3d222` declined the ·governing type definition· of a skip-wildcard attribute,
withdrawing a `cvc-id` charge §3.10.4.1's Note says was never owed. Withdrawing
a wrong charge on a document already banked `fail` cannot register as movement,
which is what the ratchet's zero-flip-down means and not a null result.

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
