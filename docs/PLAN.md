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

## Status — 2026-08-27 (`/backlog`. Replaced whole, per step 6: the lane table is a fresh `go tool lanestatus` paste, the branch namespace a fresh `wipsurvey` against a fresh `git fetch -p origin main`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh 622-issue page-numbered `state=all` fetch taken after this pass's own writes. **A `/backlog` re-derives the band ordering rather than shifting it**, so every row's rank below is argued from this pass's evidence. This pass follows **TEN** landings — the tenth, #941, landed WHILE the pass was running, so `origin/main` moved `9150d2f` → `72e40a5` under it and every count below is re-derived at the later SHA. It found **ELEVEN of the last band's twelve rows already cleared**, found **#809 closed BY THE REPO OWNER** and did not reverse it, found **THREE develop sessions in flight in the namespace**, filed **#1066**, corrected FOUR open bodies and folded the **ninth consecutive** persona consultation)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10760 | 15601 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13259 | 2139 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**The table is byte-identical for the THIRD consecutive stamp, and the last
stamp's explanation has expired.** That stamp said the reason was that no
landing touched library code. **This window landed FOUR library changes and the
table still did not move** — and, unlike a bookkeeping gap, every one of the
ten landings *declared* it:

| landing | commit | area touched | `Ratchet:` line |
|---|---|---|---|
| #1029 | `9164bc4` | `parser` (produce, report, doc, census) | `unchanged` |
| #1034 | `732ffa7` | `docs/WORKFLOW.md` only | `unchanged` |
| #719 | `bd887cd` | `validate` (validate, assess, cvcassertion, doc) | `unchanged` |
| #1030 | `adb6d57` | `parser` (produce, report, census, doc) | `unchanged` |
| #963 | `48f98aa` | `tools/landcheck` | `unchanged` |
| #1032 | `aa147fd` | `xsd/identityconstraint.go`, `xpath/doc.go` | `unchanged` |
| #1004 | `0fabb26` | `validate/doc.go`, `xsd/doc.go` | `unchanged` |
| #852 | `b9849bc` | `tools/gapaudit` | `unchanged` |
| #819 | `ce87f92` | `value` (union.go), `validate/cvcid.go` | `unchanged` |
| #941 | `72e40a5` | `parser` (document, tree, parse, override, produce_xpath) | `unchanged` |

`git diff --stat b0ff625 72e40a5 -- conformance/testdata/expectations/` is
**empty**, so the paste above is the committed expectations and no measurement
was taken or owed by this pass. **The last landing to move a lane is `e491ddb`
(#972), 2026-08-24 21:12 — `schema` 13233 → 13259 — which makes this the
THIRTEENTH consecutive `Ratchet: unchanged` landing** (#1018, #981, #493, #1029,
#1034, #719, #1030, #963, #1032, #1004, #852, #819, #941) across roughly 65
hours.

**The diagnosis this stamp acts on is different from the last one's, and
sharper.** The five library landings are census (#1029, #1030), fail-open
wiring (#719), a duplicate-scan deletion (#819) and an exported-surface
deletion (#941) — **seam work, which is lane-flat by construction**: a census
reports, a fail-open declines where it already declined, a deleted duplicate
scan decides what the original decided, and deleting an accessor with no
out-of-package caller changes no behaviour at all. Nothing in the window *decided a new rule on a shape the engine was
declining*, which is the only thing that moves a lane. **The queue could not
offer one at the time, and now it can**: #1030's stage-1 measurement filed
**#1047 (52 documents)**, **#1048 (16)** and **#1046 (31)**, the first issues in
this project's queue to carry a measured suite-document count *and* a
producer-side fix whose gate-side alternative was measured and rejected. The
band's top rows are those three.

### The ten landings, and what each left behind

**Four of the ten came out of one park.** #846 parked at the two-rejection cap
on 2026-08-25 and the 2026-08-26 pass split it into #1029 + #1030; **both
landed within nineteen hours**, and #1030's stage 1 alone filed seven follow-up
issues — the largest follow-up ledger any post-land pass has carried, and the
reason the band below is cut from measurement rather than from argument.

**#1030 landed STAGE 1 ONLY, and the stop is the finding.** 584 of the 670
remaining residual discoveries are documents where an anonymous complex type
receives no Phase D verdict at all, so repointing `assembleCase` at the census
would have been sound-by-construction and proved nothing. Stage 2 is replanned
as **#1051** (`blocked`) on a four-link chain — **#414 → #438** (584),
**#1046** (31), and **#786 plus five unfiled gate widenings** (55) — with its
bar changed from "an unchanged ratchet" to "the parity residual reaching 0",
because the original bar was unsatisfiable in principle.

**#719 landed the `Unevaluated` channel and unblocked #56.** The window's whole
unblock ledger is two — #1029's post-land pass unblocked #1030, #719's unblocked
#56, and the other seven passes each reported a measured zero.
`validate.Unevaluated` carries `Rule()`/`Loc()`/`Msg()`, `Result.Unevaluated()`
returns the document-order slice, and `validate/doc.go` now states that an empty
`Violations()` beside a non-empty `Unevaluated()` is not a pass. #56's whole design question is settled by it; what remains is the
CTA-side reuse. Assertion EVALUATION is filed for the first time as **#1042**,
`blocked`, and carries **M6 — the first issue this project has ever put on that
milestone**.

**#963 turned a hand-run check into `go tool landcheck`.** Landing preconditions
1 (the `docs/LOG/` added-lines grep naming the issue) and 2 (`merge-base`
currency) now run as one tool with `tools/lint`'s 0-1-2 exit contract. Its
sibling-precondition question — whether #1034's post-merge read-back folds into
the same tool — was answered by construction rather than deferred, and disposed
of on #992.

**#852 fixed `gapaudit`'s matcher and its own post-land pass measured the half
nobody had.** Group 2's bar now weighs a phrase run by direction, and the census
below shows it working (#592, #731 and #921 each print the collision that used
to delete their row). **Group 1's bar was deliberately left at `found()`**, and
#1060 is what that costs — see the census section.

**#1004 and #1032 closed two persona-family issues in one day each**, both
doc-only, both from the sixth-through-eighth consultations. `xsd` and `validate`
now state explicit concurrency contracts; `xsd/identityconstraint.go` no longer
says a shipped feature is deferred to M6+, and `xpath/doc.go`'s topic sentence
defers identity-constraint evaluation to `validate`. **Both were verified in the
tree by this pass rather than taken from the persona's report**, which is how
the ninth consultation's "confirmed closed" claim was checked.

**#819 landed `value.ValidatingType` and closed #844 as a DUPLICATE, not as
landed work.** #844 was row 8 of the last band; its consumer argument transfers
to #819, which is itself closed, so the row is vacated rather than inherited.

**#941 landed at `72e40a5` DURING this pass**, deleting `Element.Attributes` and
`Element.BaseURI` — both with zero out-of-package non-test callers (STYLE T5),
the fact pattern #387 settled for `Element.Parent`. Its warden pre-flight
**overruled the issue body's own plan** (delete both outright rather than keep
`Attributes` as an unexported wrapper), and its arbiter accepted in one round
with zero repair. `surface` strictly shrinks; `Ratchet: unchanged`, declared.
**Its post-land pass had not run when this stamp was written**, so its follow-up
ledger is not reflected below and the next `/backlog` inherits whatever it
files.

### #809 was closed BY THE REPO OWNER, and this pass does not reverse it

**The 2026-08-26 stamp's largest act was reopening #809** after PR #1026's
squash body — whose own heading reads *"Issues this body names and does NOT
close:"* — bound GitHub's closing keyword across a line break. That reopen was
correct about the mechanism, and the mechanism landed as **#1034** (`732ffa7`):
`docs/WORKFLOW.md` now forbids a closing keyword in a negated position and
spells the safe heading (*"names and leaves open"*, which contains no keyword).

**#809 was then closed again, by hand, by `kud360` at 2026-08-26T21:28:27Z —
no commit, no comment, four hours after the reopen.** That is a human ruling and
a cartographer never reverses one. The issue stays closed, its thread carries
the record, and **the tool defect it described is not re-filed under a new
number**: filing a near-identical issue one day after the owner closed it is
exactly what `docs/WORKFLOW.md`'s filing discipline forbids.

**A third symptom shape was nevertheless measured today and is recorded on
#802**, which is open and owns the shallow-clone premise. `wipsurvey` ran four
times in this container against an **unchanged** namespace:

| run | what changed before it | `wip/issue-953` | `wip/issue-968` |
|---|---|---|---|
| 1 | `git fetch -p origin main` only | `CLAIMED`, `main's` | `RETIRED`, `unknown` |
| 2–3 | `git fetch --depth=5` of three `wip/*` refs | `CLAIMED`, `main's` | `RETIRED`, `unknown` |
| 4 | `git fetch --depth=60 origin main` | **`UNKNOWN`, "tip not fetched"** | `RETIRED`, **`main's`** |
| 5 | `git fetch -p origin main` (picking up #941) | `UNKNOWN` | `RETIRED`, `main's` |
| 6 | `git fetch --depth=3` of `wip/issue-953` | `LIVE`, `26m0s` | `RETIRED`, **`104h46m0s`** |

Run 6's `wip/issue-953` change is real — the session pushed a commit between
runs — but `wip/issue-968`'s is not: that ref has not moved all day and its
LEASE AGE has now printed `unknown`, `main's` and `104h46m0s` in one session.
In run 4 `git cat-file -t 9150d2f` answers `commit`,
`git merge-base --is-ancestor 9150d2f origin/main` exits **0**, and
`git log -1 --format=%ct origin/wip/issue-953` answers — while the tool reports
that tip as unfetched. **This is the opposite direction from the previously
recorded symptom**: not an absent tip read as present, but a present tip read as
absent. The mechanism is recorded as a **hypothesis, not reproduced** (deepening
moved the graft boundary and the tip-time query is sensitive to it where
`merge-base` is not). The paste below is run 4, verbatim, as step 6 requires.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (re-run against a fresh `git fetch -p origin
main` at `72e40a5` — #941's own squash — and the 622-issue `state=all` fetch
re-taken after it, so this paste already reflects that landing):

```
ISSUE  BRANCH         LEASE AGE  VERDICT  REASON
732    wip/issue-732  unknown    RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822  unknown    RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846  unknown    RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872  unknown    RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  165h41m0s  RETIRED  wip/issue-933: issue #933 is closed
953    wip/issue-953  26m0s      LIVE     wip/issue-953: tip pushed 26m0s ago, within the 2h0m0s claim TTL
968    wip/issue-968  104h46m0s  RETIRED  wip/issue-968: issue #968 is closed
975    wip/issue-975  33m0s      LIVE     wip/issue-975: tip pushed 33m0s ago, within the 2h0m0s claim TTL
993    wip/issue-993  unknown    RETIRED  wip/issue-993: issue #993 is closed
```

**Three refs arrived, one left by landing, and for the first time in four stamps
the namespace is NOT all-RETIRED.** `wip/issue-941`, `wip/issue-953` and
`wip/issue-975` were all created today and **all three are develop sessions that
ran CONCURRENTLY with this pass**:

| branch | last seen | pushed | outcome by the end of this pass |
|---|---|---|---|
| `wip/issue-941` | `3b3878d` | 14:07:54Z | **LANDED at `72e40a5`, 14:45:04Z** — ref auto-deleted at merge, issue closed |
| `wip/issue-975` | `f87370d` | 14:15:35Z | **LIVE** — `1cf9ad6` (`parser: every s4s-grammar rejection names its Appendix A production (#975)`) plus a merge-forward |
| `wip/issue-953` | `8306dc7` | 14:22:33Z | **LIVE** — `xsd: enumerate ValueSpace's FAIL-OPEN CONTRACT readers by identifier (#953)` |

**#975 and #953 are LIVE, and neither is in the band below.** Both look likely
to land before this stamp is next read — #941 already did, mid-pass — which is
the ordinary way a band row goes stale and is not a defect. **A session reading
this band should re-run `wipsurvey` before picking.**

**#953's earlier row is the #981 landing working exactly as designed, and the
record is worth keeping even though the row has since changed.** When this pass
first surveyed, `wip/issue-953` carried **no commits of its own** and its thread
held a `GROUNDING:` at 14:05:02Z and no heartbeat. The second, comment-enriched
pass under `docs/ROUTINES.md`'s *"Dating an empty claim"* was run and reported
*"no commits of its own and no `RESUME:`/`TAKEOVER:` comment ever posted — a
claim is born undated, so this is not a lapsed lease"*. A `GROUNDING:` does not
date a lease and the tool now says so in as many words. Twenty minutes later the
session pushed `8306dc7` and the row became LIVE on its own.

**Nothing is EXPIRED, no `parked/*` ref exists, and no RETIRED row needs a
supersede check.** The six RETIRED refs were seven on the last stamp;
`wip/issue-941` left the namespace the only way the workflow ever removes one —
GitHub's auto-delete at merge — and the remaining six are unchanged. `wip/issue-846`'s content is in `main` by way of #1029's landing
(`9164bc4` applied `ad036ed`'s diff with zero conflicts, per that landing's own
log entry), which is the last of the seven that needed verifying.

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON —
142 issues, 50 of them open; both counts unchanged from the last stamp):

```
gapaudit: 69 GAP marker(s) across 6 area(s)

=== Per-area census ===
  parser           3
  validate         16
  value            3
  xml              4
  xpath            6
  xsd              37

=== Group 1: markers with no OPEN tracking issue found ===
(the file and phrase signals are heuristic, so a row here means "needs a
look", not proven untracked. A row with NO annotation under it cites no
issue at all — that is the list to file against; read the annotation on
any other row first.)
(none)

=== Group 2: OPEN kind/gap issues with no surviving marker ===
(a stale tracker if the gap was a marked fail-open site — but
kind/gap also labels conformance-lane gaps, which never carry a
marker and belong here permanently)
  #398 cmd: the notImplemented stub is untracked P3 debt (no GAP(cmd) marker), and TestUsageCoversContract guards four substrings of a whole-block coupling
  #404 conformance: closureScan.unresolved is scan-scoped, so the decline conjunction is coarser than the hazard it models
  #591 conformance: an instanceTest caseSpec DROPS its sibling schemaTest <schemaDocument>, so readFacetsCase declines time_minInclusive006_1163.i for a schema the catalog names explicitly
  #592 conformance: string_pattern002_1031.i — a <list itemType> over a USER-DEFINED item type in a target namespace, with a multi-leaf instance shape no cohort reader matches
      phrase-matches validate/cvcassertion.go:96 — too weak to retire this tracker; check whether it is the same gap
  #593 conformance: decimal_totalDigits004_1060.v carries its tested value on the ROOT attribute, a shape readFacetsCase exactly-one-<foo> guard declines
  #731 conformance/parser: the 11 cases #442 reclassified from declined to decided-and-DISAGREEING — invalid regex facets, unenforced bound consistency, three grammar faults and a substitutionGroup head, all pre-existing and all under-rejections
      phrase-matches parser/produce.go:873 — too weak to retire this tracker; check whether it is the same gap
      phrase-matches parser/produce_complex.go:1957 — too weak to retire this tracker; check whether it is the same gap
  #787 value: an enumeration member outside the base's value space is charged enumeration-valid-restriction (§4.3.5.5) at schema construction and never src-enumeration-value (§4.3.5.3) — which rule the spec assigns is ungrounded, and restriction.go's wrap shadows newEnumFacet's remap
  #921 conformance: <current status="queried"> is unmodeled, so the two gMonth XSD-1.0-lexical cases are permanently unbankable disagreements with no owner and no stated reason
      phrase-matches parser/conditional.go:208 — too weak to retire this tracker; check whether it is the same gap
```

**The census moved for the first time in three stamps: 66 → 69 markers, and the
group-2 report is visibly a different instrument.** `parser` gained one and
`validate` gained two, from #1029/#1030's and #719's landings. Group 2 lost
**#569** and **#719** and gained **#731** and **#921** — and #592, #731 and #921
now each print the phrase collision that used to *delete* their row silently,
which is #852's landing (`b9849bc`) working in the field.

**Group 1's `(none)` is STILL false, and this pass paid the caveat paragraph for
a fourth consecutive stamp.** #852 raised group 2's bar and deliberately left
group 1's at `found()`, so a five-word phrase run still suppresses a group-1
row. **Four markers that cite no issue at all sit inside that `(none)`** —
`validate/cvccomplexcontent.go:445`, `validate/cvcid.go:232`,
`validate/cvcidentityconstraint.go:390` and `validate/assess.go:874`, each read
from the tree by this pass with `grep -n "GAP(validate)"`. **This pass filed no
`kind/gap` issue against them and the reason is that all four are already
disposed**: the first three are #853's own Acceptance sites (whose two drifted
line numbers this pass corrected), and the fourth is the marker #853's Notes
deliberately leave unfiled with the reason stated (*"its cause is #790's
descent, not clause 5.1 … it wants a P3 issue reference and nothing else"*).
**#1060 owns the instrument.** Group 1's `(none)` therefore reads *"nothing the
heuristic could see"*, and every `/backlog` writes that sentence down until
#1060 lands.

### Milestones and queue

Both columns and the queue below are RE-DERIVED this pass from one page-numbered
`state=all` fetch — 11 pages, **1067 items, 622 issues** once the 445 pull
requests are dropped — taken after this pass's own writes **and after #941's
mid-pass landing**, not from the milestones endpoint. `gh api --paginate` is still unusable (numeric-ID
repository paths in the Link header, 403 from the proxy, after writing the pages
it did fetch), and `gh issue list` is unusable for a different reason: it is a
GraphQL path. Repository-scoped REST served **every** read and write this pass
made — **16 writes**: one issue creation, five body PATCHes and ten comments,
plus three full eleven-page listings and every body and comment thread read
behind them. `docs/ROUTINES.md` is accurate on all three.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 98 | 48 | **active** |
| M5 — Instance validation (XML) | 17 | 12 | **active** |
| M6 — XPath required subset | 0 | 1 | **opened** |
| M7–M12 | 0 | 0 | not filed |

**Three rows moved, ending five consecutive flat tables.** M5 closed two (#719
and #819, both in the nine) and gained one (#1043), 15/13 → 17/12. **M6 gains
its first issue ever — #1042**, assertion evaluation, `blocked`; the milestone
existed as a heading with nothing under it since the roadmap was written. **M4's
open count moved for the first time in six passes and moved UPWARD, 45 → 48** —
#1046, #1047 and #1048, every one of them a measured `schema`-lane candidate, so
the rise is the census finding work rather than the tail growing.

**176 of the 237 open issues carry no milestone** (237 − 48 − 12 − 1), so the
rows above are feature progress and the paragraph below is the queue.

Queue: **237 open — 216 `ready`, 21 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 21), against **385 closed**.
216 + 21 + 0 = 237 exactly, and **every one of the 237 carries exactly one queue
label** — checked mechanically, not by eye. **The hygiene sweep is clean on all
four classes this stamp**: zero open issues with no queue label, zero with two,
zero with no `area/` label, and zero with no `kind/` label. The class #773/#774
fell into stays empty for the nineteenth consecutive stamp.

**Every move reconciles.** From the previous stamp's 233 open / 212 `ready` /
21 `blocked` / 0 `needs-replan` / 373 closed:

| move | open | `ready` | `blocked` | closed |
|---|---:|---:|---:|---:|
| previous stamp | 233 | 212 | 21 | 373 |
| eight `ready` landings closed (#1029 #1034 #719 #963 #1032 #1004 #852 #819) | −8 | −8 | | +8 |
| #1030 landed — it was `blocked`, filed so by the last stamp | −1 | | −1 | +1 |
| #844 closed as a DUPLICATE by #819's post-land | −1 | −1 | | +1 |
| **#809 closed BY THE REPO OWNER**, 2026-08-26T21:28:27Z | −1 | −1 | | +1 |
| #56 unblocked by #719's landing | | +1 | −1 | |
| fourteen `ready` filings (#1036 #1038 #1040 #1043 #1046 #1047 #1048 #1049 #1050 #1052 #1060 #1061 #1062 **#1066**) | +14 | +14 | | |
| two `blocked` filings (#1042, #1051) | +2 | | +2 | |
| **#941 landed at `72e40a5` WHILE this pass ran** | −1 | −1 | | +1 |
| **this stamp** | **237** | **216** | **21** | **385** |

**#1066 is this pass's only filing.** Every other number in that ledger was
filed by a post-land pass, which is the disposition rule working: nine landings
produced fifteen follow-ups and none of them was handed to `/backlog` as a
hand-off.

**Five `ready` issues are STILL done, and the count above still overstates what
is startable by five — seven, counting the two LIVE claims. This is the SIXTH stamp carrying it.** **#625**, **#748**,
**#492**, **#934** and **#896** were discharged by `34a8043` and judged with it,
and all five remain open because GitHub bound `Closes #669, #625, …` to the one
reference following the keyword. **#1023** tracks them as a filed issue. Each of
the five also carries a comment naming the landing and telling a session not to
take it. **They are not closed here**: the cartographer files, unblocks and
restamps, and never closes an issue as done.

**The unblock sweep relabelled nothing, and the zero is measured.** All 21
`blocked` bodies were fetched over `gh api` — byte-faithful, where MCP
`issue_read` is lossy (#764) and, per **#1014**, silently truncating — every
`## Depends on` section was split out and read, and **not one names a closed
issue.** Every named dependency was checked back against the live state:
#414, #438, #786, #831, #472, #248, #591, #455, #407, #250, #79 and #1046 are
all open. Nine of the 21 name a trigger rather than an issue (#79, #250, #555,
#692, #841, #925, #1002, #1042 and the epic pair's own line), and each says in
its body not to re-scan it. **#56's unblock was made by #719's post-land pass,
not here** — this pass found it already `ready`.

**Four open bodies were corrected rather than commented at**, per
`docs/WORKFLOW.md`'s filing discipline:

- **#409** — three of its five files STRUCK. `validate`, `validate/xmlsrc` and
  `xpath` all export a real surface now (25, 3 and 5 symbols), so their
  present-tense prose is accurate and rewording it would make the docs worse.
  Every heading citation in its evidence block had drifted
  (`xsd/doc.go:190` → `:234` and `M5/M9` → `M9`; `parser/doc.go:226` → `:266`;
  `validate/doc.go:41` and `validate/xmlsrc/doc.go:4` now read different
  headings entirely). One site ADDED: the root `doc.go` package list.
  `issuecomment-5440579942`.
- **#1006** — its Acceptance was **unsatisfiable**. `Run`'s signature is
  `func Run(t *testing.T, …)`, so *"an `Example` that calls `Run`"* cannot be
  written at all; the tree's own `ExampleRun` had already discovered that and
  written the reason into a comment. Re-scoped to a `TestXxx` for the call and
  an `Example` for the capability. Two further claims corrected: `Run` **is**
  exercised (`builtin/strict/strict_test.go:21`, on the module's own backend),
  and `Canonical` on a caller-owned type **is** already demonstrated.
  `issuecomment-5440600448`.
- **#16** — its repeated-flag notation criterion LIFTED to #1066 and marked so
  in the body, on the ground #720 used for three other criteria: this issue is a
  reference and is never worked directly, so a fileable item parked here has no
  taker. `issuecomment-5440630149`.
- **#853** — two of its three Acceptance-table line numbers drifted when #719's
  landing edited `cvccomplexcontent.go` and `cvcid.go` (`:394` → `:445`,
  `:389` → `:390`); the marker text is unchanged and is what to match on.
  `issuecomment-5440677784`.

### Persona consultations — the NINTH consecutive, and the first to file only ONE

Run by the orchestrating session, never by the cartographer (#416): each persona
saw only the README, `go doc` output and, for cliuser, the built binary. **One
new issue filed, two bodies corrected from persona evidence, and — for the first
time — two persona-family issues confirmed CLOSED by landings inside the same
window.**

**Both of libuser's "confirmed closed" claims were verified against GitHub and
the tree before anything was written down**, because a persona reads only the
published surface and cannot see an issue's state. **#1004** (`0fabb26`) and
**#1032** (`aa147fd`) are genuinely closed, and the doc text libuser quoted is
in the tree: `xsd/doc.go` and `validate/doc.go` carry explicit concurrency
contracts, and `xpath/doc.go:1-7` now defers identity-constraint evaluation to
`validate` in its topic sentence. That is **two of the eight-issue persona
family discharged in one window**, against a family that had never had one
taken.

**libuser's new finding is the SIXTH sighting of #409, and it was not filed
again.** `codegen` and `codec` present `Generate`/`Target`/`Option` and
`AppendCanonical`/`ParseBytes` in `go doc` code blocks with no *"not yet
implemented"* banner, where `builtin/native`, `validate/jsonsrc`,
`validate/bersrc` and `xsd` all carry one —
`grep -in "not yet\|planned" codegen/doc.go codec/doc.go` returns zero matches
in both, the only two library packages for which that is true. **#409 has owned
this since 2026-08-02.** What the ninth consultation added is a **site** —
the root `doc.go`'s six-bullet package list, which `README.md:224` sends a
consumer to first and which marks nothing — and a stronger check
(`go list -json` reporting `"GoFiles": ["doc.go"]`). Both are in #409's body;
so is the three-file strike above, which the persona's own report made visible
by naming which siblings get it right.

**libuser's second finding cost #1006 its Acceptance.** `value/backendtest/example_test.go`
exists, is named `ExampleRun`, and its body does `_ = backendtest.Run` and then
hand-rolls the round-trip — because `Run` takes a `*testing.T` and **no runnable
`Example` can construct one**. The issue asked for something no implementation
could deliver. This is a **#1052** instance found by a persona rather than by a
mason mid-implementation, which is the cheapest place the class has ever been
caught, and it is the fifth subject on that issue.

**cliuser filed #1066: the `validate` blurb is unscriptable for a batch.**
Three holes, verified against `cmd/goxsd8/doc.go` and `README.md` before filing:
(a) `-schema <schema.xsd>...` reads as one flag taking many values, a form Go's
`flag` cannot parse, against `gen`'s `[-schema <s2> -out <d2>]...` for the same
flag — **lifted from #16**, whose bullet already carried the `flag`-package
argument cliuser re-derived independently; (b) exit-code aggregation across
`<instance>...` is stated nowhere, where #720 owns only the *reporting* half and
is `blocked`; (c) the contract page names a stream for help (stdout), the three
dispatch diagnoses (stderr) and `-v` (stderr), and names none for `parse`'s
summary or `validate`'s violation lines — the two outputs a script consumes.
`parse`'s answer already exists inside **#472**'s Acceptance and has never
reached the contract page.

**Filed as one issue rather than three**, on #1007's and #1031's precedent and
against `docs/PLAN.md`'s own recorded hazard: three one-sentence issues editing
the same three copies would each rebase on the other two plus #1007 and #1031.

**#1033 reconfirmed with a measurement, and nothing else moved.** `builtin/doc.go:44`
still says *"the ~25 primitives"*; libuser ran `builtin.Seed(strict.New())` and
got 51 components, which agrees with `typespec_test.go:20-21` (49) and `:64`
(20 primitives) and with nothing near 25 — three independent counts, one answer.
**#1003**, **#1005**, **#1007**, **#1031**, **#812**, **#472** and **#16** were
all reconfirmed unchanged and not re-litigated; cliuser additionally reconfirmed
#16's dispatch matrix byte-for-byte and correctly routed `gen`'s missing
exit-code sentence to #1007 rather than re-filing it.

**Every sharper argument was banked on its thread**, not only here, because
`docs/PLAN.md` is replaced each pass and an issue thread is not:
`issuecomment-5440579942` on #409, `-5440600448` on #1006, `-5440630149` on #16,
`-5440636124` on #1033, `-5440636373` on #1031, `-5440638294` on #1007,
`-5440677784` on #853, `-5440711090` on #809, `-5440715805` on #802,
`-5440806844` on #1043.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
of it. Take from the top. Cross-references are stated by ISSUE rather than by row
number, which decays at each re-cut. **This is a `/backlog` re-cut: every rank is
re-derived from this pass's evidence.**

**ELEVEN of the last band's TWELVE rows left it, and that is the strongest
result the band mechanism has produced.** Seven landed (#1029, #1034, #719,
#963, #852, #1032, #1004), one closed as a duplicate of a landing (#844 → #819),
and **three were claimed by develop sessions running concurrently with this
pass** (#941, #975, #953) — of which **#941 landed before the pass finished**,
making it the eighth landing of the twelve. The twelfth row, **#853**, is the
sole survivor and is banded again at row 7. Nothing below is inherited.

**Two `wip/*` refs are LIVE, so this band is NOT "all startable" — the first
stamp in five where that sentence does not hold.** #975 and #953 are excluded
from every row below, and both are likely to have landed by the time this is
read.

| # | Issue | Why here |
|---:|---|---|
| 1 | #1047 | **The largest measured lane candidate this queue has ever carried, and the direct answer to thirteen consecutive `Ratchet: unchanged` landings.** `checkS4SChildOrder` SKIPS a child no position of the chosen model admits, so **52 suite documents** are decided on a document the producer dropped a child from — 43 for a wrapped `<complexType>` carrying a sibling the other `xs:complexTypeModel` disjunct admits (`<attribute>` 9, `<anyAttribute>` 9, `<attributeGroup>` 8, a model group 16, a `<sequence>` under `<complexContent>` 1), plus 9 for `<element>`/`<attribute>`/`<simpleType>`'s own children. Every one is in the suite's INVALID corpus, so the direction can only be up. **The obvious fix is already measured and ruled out**: widening the shape gate instead costs a banked ratchet win (`TypeAlternativeTests/s3_12si01/schema/s3_12si01s` → decline), and the body leads with that so nobody rediscovers it in a rejected round. `produce_s4sorder.go:228-235`'s own doc states the skip deliberately and this issue does not overturn it — it asks for the different fault to be charged alongside the order check, with `:209` as prior art eleven lines away. The 9-document second group is marked in the body as a **hypothesis to confirm at grounding** |
| 2 | #1048 | **The cheapest measured lane mover in the band, and the same landing shape as the row above.** A named `<group>` with TWO compositor children silently loses the second: `compositorChild` (`parser/produce_complex.go:1837`) returns the first and stops, `xs:namedGroup`'s inner choice is `minOccurs="1" maxOccurs="1"` and admits exactly one, and **16 suite documents** are decided on a partial read. `rejectNamedGroupBody` (`:1823`) already carries the right error text and the right citation and is only reached when there is NO compositor at all, so the narrowest fix is to charge the second child where `compositorChild` passes it over. Carries the same measured "fix the producer, not the gate" ruling as #1047. **The census structurally cannot reach this and says so** — it is NAME-based, and a second compositor is a name whose position was already filled — so a producer rejection is the only instrument that gets there. Read beside **#931** (occurrence attributes on the compositor child), which is adjacent and distinct |
| 3 | #1052 | **FIVE consecutive sessions have now paid this, and CLAUDE.md bands `kind/process` on the sessions it costs rather than on the lane it does not move.** Four landings discovered an `## Acceptance` criterion unsatisfiable only AFTER implementing — #1030 (a subset relation mistaken for a soundness one, which stopped stage 2 outright), #1029 Friction 1 and 2, #719 Friction 1 (a bullet that would have REGRESSED the ratchet) — and **this pass added the fifth from a different direction entirely**: #1006's *"an `Example` that calls `Run`"* could not be satisfied by anyone, and a persona found it in one reading. The tax compounds and the fix is one session. **Its Acceptance is a mandate, not a rule** — a named step at which an `## Acceptance` bullet is ruled satisfiable, able to answer *"this bar cannot prove what it claims"*, which is neither #635's (wrong citation) nor #912's (stale site list). `/retro` may fold the five species into the genus instead, and the body says so |
| 4 | #1046 | **The third census bucket, 31 documents, and the only one of the three that also discharges a link of #1051's chain.** `<schema defaultAttributes=>` is unmodelled at all — §3.4.2.4 clause 3's `{attribute uses}` fold is skipped, `conformance/schema.go:597` declines every document whose root carries the attribute, and a document invalid only because of the folded-in uses would false-ACCEPT. Ranked below its two siblings because it is a genuine FEATURE rather than a rejection the producer already nearly makes, and its `## Surface` is `TBD` with a warden pre-flight flagged. **It is the one bucket the census structurally cannot name** (`UnmappedConstruct.Name` is documented as an ELEMENT's expanded name and this is an attribute-level omission), so the body records the explicit alternative — a new `UnmappedReason` that can name an attribute — and **either direction discharges #1051's dependency**. Take one and say which |
| 5 | #1043 | **A live FALSE REJECT in `validate`, and the ratchet cannot register the damage.** `walk.attributeType` (`validate/cvcid.go:574`) falls through to `topLevelAttributeType` (`:586`) for every attribute matching no `{attribute use}`, with no wildcard check and no `{process contents}` check at all — so an attribute admitted by a ***skip*** wildcard binds an ·ID value· and `cvc-id` clause 2 charges a duplicate on a document §3.10.4.1's Note says has no ·governing· declaration. **Its own doc is precise about the strict/lax reading and does not act on it.** `walk.keyMember` has the identical exposure for a `@NameTest` field. `xsd.ProcessContents` and `xsd.ProcessSkip` already exist and `validate` reads neither; reading them is the whole change. Ratchet unchanged-or-upward — it can only WITHDRAW a charge, which is the same trade #771's family records and the reason a wrong decision outranks a decline |
| 6 | #1060 | **The tool the cartographer runs every pass reports something FALSE, and this stamp paid the caveat paragraph for a fourth consecutive time.** Group 1 printed `(none)` above across 69 markers while **four markers cite no issue at all** — `validate/cvccomplexcontent.go:445`, `validate/cvcid.go:232`, `validate/cvcidentityconstraint.go:390`, `validate/assess.go:874`, each read from the tree here. #852 landed (`b9849bc`) and **deliberately raised group 2's bar only**, so `anyOpenMatch` still uses `matchKind.found()` and a five-word phrase run still empties a group-1 row. Its second half is measured too: `matchFile` retires a tracker on a bare path mention, witnessed by an exclusion — forcing `#972` open moves group 1 **69 → 64** on markers #972 owns none of. **Both directions are one landing**, and #852's own out-of-band citation report is unobservable until this lands. Banded on cost: the caveat is written into every stamp until it is fixed |
| 7 | #853 | **The band's only `instance` row, the sole survivor of the last band, and now the named owner of three of row 6's four hidden markers.** `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once, and all three markers name no issue (STYLE P3). Its two drifted citations were corrected by this pass; the marker TEXT is what to match on. **Start with an oracle question, not with code**: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type. `instance` candidate, **UNMEASURED** — run `GOXSD_DECLINES=1` and count before promising a figure; the direction can only be up. The ·nilled· shape is explicitly out of scope and each marker keeps its ·nilled· bullet |
| 8 | #56 | **Newly unblocked, its whole design question settled by a landing 12 hours old, and the consumer half is all that is left.** #719 shipped `validate.Unevaluated` with `Rule()`/`Loc()`/`Msg()` and `Result.Unevaluated()`, and `validate/doc.go:88` now states that an empty `Violations()` beside a non-empty `Unevaluated()` is not a pass. This issue records the CTA withhold into that same slice under a third rule ID — **no second type and no `Evaluated bool`**, which #842's warden pre-flight ruled out on D3. Expected surface: none new. **It moves no lane by construction and is banded anyway**, because STYLE 9's fail-open discipline is only honest if a fail-open answer is distinguishable from a real pass, and `validate` currently knows the type was withheld and tells no one. Take it while #719's seam is still the freshest thing in the log |
| 9 | #414 | **Head of the longest dependency chain in the queue, and the 584-document bucket sits behind it.** Two bare `GAP(` markers in `xsd/complextype.go` — on `AttributeUses` and `AttributeWildcard` — name no owning issue, and the §3.4.2.4 clause 3 / §3.4.2.5 clause 2 folds walk the finalized Schema's type definitions only, so an anonymous complex type nested in a particle tree is folded for neither property. **One decision, taken once, applied to both folds**: widen the reach, or record the narrow reach as a decided permanent under-approximation. It gates **#438**, **#584** and — through both — **#1051**, whose parity residual is 584 documents of exactly this shape. Ranked below the direct movers because it is a decision rather than an implementation and its own lane movement is zero; ranked here rather than below the fold because **nothing else in the queue moves 584 documents' worth of anything** |
| 10 | #1066 | **This consultation's only filing, and the CLI-contract doc family's cheapest landing — take it WITH #1031.** Three holes in the `validate`/`parse` blurb pair, each one sentence, all in the same three copies (`cmd/goxsd8/doc.go`, `main.go`'s `usage`, `README.md`): the `-schema` repetition notation (lifted from #16, which could never be worked directly), exit-code aggregation across `<instance>...`, and the unstated stream for `parse`'s summary and `validate`'s violation lines. **#1031 rewrites the same blurb from the eighth consultation** and taken separately the second rebases on the first for no gain. Banded above #409 because a script author is blocked TODAY by a contract page, where #409's reader is merely misled |
| 11 | #409 | **SIXTH independent sighting, by two personas, none of which was told the issue existed — the most-corroborated doc defect this repo has.** `codegen` and `codec` print signatures in `go doc` code blocks with no *"not yet implemented"* banner, the only two library packages for which `grep -in "not yet\|planned"` returns nothing, and the root `doc.go` bullet list — the page `README.md:224` sends a consumer to first — marks them not at all. **This pass STRUCK three of its five files** (`validate`, `validate/xmlsrc` and `xpath` now export 25, 3 and 5 symbols, so their present tense is accurate) and added the root doc, so what was a five-file landing is now four sites and strictly smaller. Still one landing and one convention; **do not split** |
| 12 | #1036 | **The silence #1029's landing exposed and STYLE P3 does not permit to stay.** A top-level `<xs:schema>` child outside the XSD namespace is neither charged as the §5.1 first-bullet grammar fault it is — §A's `<xs:schema>` content model has no wildcard arm, and `xs:openAttrs` admits foreign *attributes*, not foreign element children — nor reported through `parser.AssembledDocument.Unmapped`. **One settled disposition, either direction**, and the landing that exposed it deliberately left it: #1029's post-land pass confirmed it open *because STYLE P3 requires it*. Adjacent and distinct: #1047, whose own body says foreign-namespace children stay skipped and names this issue as the owner |

**Left the band this pass, and why.** **#1029**, **#1034**, **#719**, **#963**,
**#852**, **#1032** and **#1004** — all seven landed, all seven were band rows
on the last stamp. **#941** — LIVE when this pass surveyed and **landed at
`72e40a5` before it finished**, its warden pre-flight overruling the issue
body's own plan; eighth landing of the twelve rows. **#844** — closed as a
**duplicate** of #819 rather than landed, so its row is vacated and not
inherited; the consumer argument transferred to #819, which is itself closed.
**#975** and **#953** — both LIVE in the namespace, so both are excluded rather
than demoted; each carries a real implementation commit pushed within
half an hour of this stamp. **#853** is the only row that stayed, and it moved
from 12 to 7.

**Deliberately unbanded, and why.** **#1051** is `blocked` on a four-link chain
and belongs to no band: **#414** at row 9 and **#1046** at row 4 are two of the
four, and taking either is what moves it — it is **not** a "one more session"
issue and its body says so twice. **#1042** is `blocked` and is the M6 assertion
evaluator; it is the largest unfiled-until-now thing this project had and is now
filed, but nothing can start it before an XPath 2.0 evaluator exists.
**#1002** is `blocked` on a human ruling and nothing else. **#1023** is the
five-issue residue tracker and is the first item of the next planning action
rather than a band row, because closing an issue as done is the develop loop's
act. **#1007** and **#1031** are the CLI-doc family's other two rows and are
named at row 10 rather than banded separately — take #1031 with #1066 or take
neither. **#1049** and **#1050** are #1030's two round-2 `[P2]`s, kept apart
because #1049 needs a test (the `mappedFacetElement`/`s4sFacetElement` seam is
one name wide and nothing pins it) and #1050 is doc-only. **#1038**, **#1040**
and **#1061** are one-line doc corrections from three different post-land
passes; **#1062** is `gapaudit`'s input-shape widening and should be read beside
#1060. **#1014** is the MCP `issue_read` truncation; below the fold only because
`gh api` is the documented byte-faithful path and every read and write in this
stamp used it, without a failure. **#1005**, **#1003**, **#1006** and **#812**
are the persona family's surviving members, all reconfirmed by the ninth
consultation; **#1033** gained a third independent count and no rank.
**#1019** names four labels never used in 622 issues and omits `area/xsd` at the
top of the usage table; a decision plus two edits, unbanded only because no
session has ever been misled by it in a way the log records. **#1011** is #843's
follow-up with **#725** as its live victim. **#999** is #398's sibling and most
likely *deletes* a test. **#996** and **#992** stay unbanded on the
discriminator that #992's cost was paid-and-absorbed. **#989** is #979's
post-land filing, decision-first and cheap. **#894** is clear to proceed and
stays unbanded only because it is a lane-flat correctness fix whose first step is
an unsettled oracle question. **#888** and **#889** still await a suite census in
their range. **#907**'s `childElement` census is stale by at least twenty
landings. **#885**'s three discriminators still have one sighting each. **#854**
did not gain a third sighting this pass. **#670** asks for
`parser/example_test.go` — read it beside **#1006**, whose Acceptance this pass
re-scoped. **#937** is naturally folded by the next landing touching
`rejectRepeatedAnnotations`. **#920** and **#921** are conformance-bookkeeping
follow-ups. **#929** and **#931** are the small parser occurrence / rule-mapping
gaps #901 exposed; **#931** should be read beside row 2. **#455** is the live
owner of the `strings.TrimSpace`-versus-§4.3.6 character class at **ten** sites,
and **#456** stays `blocked` on it. **#845**, **#848** and **#849** are the
2026-08-16 audit's remaining open findings, all ranked *stable debt* by the
steward. **#841** is `blocked` on a trigger that has now fired twice without a
ruling. **#566** is #565's open sibling. **#692** and **#925** are still
`blocked` on a `/retro` trigger that fired without ruling on them. **#16** and
**#472** were both touched this pass and neither moves: #16 is a `blocked`
reference issue that is never worked directly and just had one criterion lifted
out of it, and #472 is the `goxsd8 parse` implementation whose position is set by
M4's tail. **#570** carries the standing `schema` decline-count argument at 893
against a re-measured 788, and it is now **eleven landings staler** — no
conformance measurement was taken here either.

### Next planning action

1. **Close the five. SIXTH stamp.** #625, #748, #492, #934 and #896 are
   discharged by `34a8043` and judged with it; only #669 closed. **#1023**
   tracks them. Closing them is the develop loop's act — the cartographer never
   closes an issue as done — and it is one
   `gh api ... -f state=closed -f state_reason=completed` per issue. Until it
   happens the `ready` count above overstates what is startable by five.
2. **THE RANKING QUESTION THIS STAMP EXISTS TO ANSWER: thirteen consecutive
   landings have declared `Ratchet: unchanged`, and the band's top two rows are
   the first measured answer the queue has been able to offer.** #1047 (52
   documents) and #1048 (16) are producer-side fixes whose gate-side alternative
   was measured and rejected, in the suite's invalid corpus, where the direction
   can only be up. **If a third consecutive stamp publishes a flat table after
   those two land, the diagnosis is wrong and the thing to re-examine is the
   measurement, not the queue.** That is a trigger, and it is set here.
3. **The human decision blocking #1002 is unchanged and is now carried for a
   fourth stamp.** **#1002** waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID with a per-case justification in the verdict, and (b) holding
   §4.2.2's `vc:maxVersion` arm until real assertion evaluation lands. CLAUDE.md
   puts (a) beyond any agent — *"changes only via a human-filed issue"* — and (b)
   now depends on **#1042**, which is filed and `blocked`. **No agent should
   attempt either.** Record the ruling as a comment on #1002; that comment is
   what moves it off `blocked`.
4. **Assertion evaluation is FILED, and the item this section carried for four
   stamps is discharged.** **#1042** (`blocked`, `kind/gap`, **M6**) owns
   `cvc-assertion` (§3.13.4.1) and `cvc-assertions-valid` (§4.3.13.3) and
   retires #719's `GAP(validate)` markers. It is the first issue ever placed on
   M6. **What is still uncarved is M6 tier 2 itself** — `$value` binding, an F&O
   function library, typed comparison — which is too big for one issue and whose
   carve is a `/backlog` act at the M6 opening. #1042 is now the thing that carve
   must slice, rather than a blank.
5. **The namespace is NOT all-startable for the first time in five stamps, and
   this stamp watched one of the three claims land while it was being written.**
   #941 went LIVE → landed (`72e40a5`) inside the pass; **#975 and #953 are still
   LIVE** and each pushed an implementation commit within half an hour of this
   stamp. **Re-run `wipsurvey` before picking from the band.** #941's post-land
   pass had not run when this was written, so whatever it files is inherited by
   the next `/backlog` rather than reflected here.
6. **`gh issue list` is still a GraphQL 403 and `gh api --paginate` still
   truncates.** Both are documented in `docs/ROUTINES.md` and both were hit
   again here. **#1062** would make the survey input carry labels; read it beside
   #1060.

**The two standing promotion discriminators: one fired and LANDED, and the other
still could not fire.** **#963's** (did the landing carry its `docs/LOG` entry
inside the squash?) is no longer a discriminator — **#963 itself landed at
`48f98aa` as `go tool landcheck`**, so the question is now executable rather
than judged, and the nine-window clean run it accumulated is what justified
building it. **#846's** (did a landing's entry record the shadow tax — a
`conformance/schema.go` edit mirroring a `parser/produce_complex.go` one?):
**it could not fire for the eighth consecutive window**, and the last stamp's
re-examination held — #1029 and #1030 are the producer widenings it wanted, and
both touched `parser` without touching `conformance/schema.go` at all, because
stage 2 (which is where that file changes) stopped. It carries to **#1051**
unchanged.

**The steward-ranking rule has now produced four landings out of five bandings.**
#979 landed 08-23, #978 and #843 on 08-24, and **#1029 + #1030 both landed
08-26/27** — the park's replacement pair, which is the rule's first case of a
parked subject re-planned and delivered. **#841 remains the counter-example the
rule cannot reach** — a `kind/refactor` with a steward ranking that stays
`blocked` because its trigger has no mailbox, now fired twice without a ruling.
That gap is on #841's thread and is still not filed, because the fix belongs to
whichever pass gives Part 2 of the `/retro` the mailbox Part 1 has. **There is
no `Increasing` steward ranking left in the band**: #844 carried the last one
and closed as a duplicate.

**Standing, unchanged, and still true.** Four unlanded corrections target one
paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**, **#646**,
**#679**, **#912** — and whichever lands last rebases three times; #992, #1019,
#1040 and #1052 are not among them, all targeting different paragraphs of the
same file. **The next `/retro` inherits four issues it already owed** — #692,
#925, #841 and now **#1052**, whose body explicitly offers `/retro` the
fold-the-five-species option. **The CTA cohort's 45 banked `instance` failures
remain unattributed**, seventeenth consecutive stamp carrying it. **`gate.yml`
runs but is still not a required status check**, which only the repository owner
can change.

**Environment costs stay in the log at one witness each.** Uncached conformance
test binaries hang under the default sandbox, so conformance runs must be issued
unsandboxed — this pass took no conformance measurement, so it neither
corroborated nor weakened it. `gh issue list` still fails with
`HTTP 403: This GraphQL query is not enabled for this session` while
repository-scoped `gh api` REST served **16 writes and every read behind them**
here without one failure, which is exactly what `docs/ROUTINES.md` says. **The
shallow-clone premise gained a THIRD witness and it runs the other way** — a
`git fetch --depth=60 origin main`, taken to see the window's landings after
this shallow clone had truncated `origin/main` to five visible commits, made `wipsurvey` report an already-present tip as
unfetched while making a genuinely absent one appear. It is recorded on **#802**,
which is open, and on the closed **#809**, and it is filed nowhere new because
the repo owner closed #809 by hand yesterday.

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
for **`schema` +26**, the family's largest single move. Live ones in the same
family are **#471** (a local `<element ref=>` carrying `substitutionGroup=`,
silently accepted), **#931** (occurrence attributes on a named `<group>`'s child
compositor), **#929** and **#455**. A second, narrower family has opened beside
it: the rejections the producer already makes
**correctly but describes badly**, whose bar `xsderr/doc.go` set with #966 and
whose nine deficient sites **#975** owns.

**A THIRD family opened on 2026-08-27, and it is the first whose members arrive
with a measured case count.** #1030's unmapped-construct census turned the
"decides and ACCEPTS" family from a shape into a list: **#1047** (52 suite
documents — `checkS4SChildOrder` skips a child no position of the chosen model
admits), **#1048** (16 — a named `<group>` with two compositor children loses
the second) and **#1046** (31 — `<schema defaultAttributes=>` is unmodelled, so
§3.4.2.4 clause 3's `{attribute uses}` fold never runs). All three are producer
widenings, and for all three the gate-side alternative was **measured and ruled
out**: widening `conformance/schema.go`'s shape gate costs a banked ratchet win.
They are why M4's open count rose 45 → 48 while nothing closed — the census
found work rather than the tail growing. The GitHub milestone holds the feature
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
