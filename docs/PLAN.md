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

## Status — 2026-08-24 (POST-LAND PASS for #978, landed as PR #995 / `bd22026`. Replaced whole, per step 6, never patched in place: the lane table is a fresh `go tool lanestatus` paste, the branch namespace a fresh `wipsurvey` against a fresh `git fetch -p`, the marker census a fresh `gapaudit` fed the same `kind/gap`-filtered input as the previous stamp, and the milestone and queue counts a fresh 586-issue page-numbered `state=all` fetch taken at 09:42Z. **One band row leaves and none arrives** — row 2 #978 landed — and every surviving row's ordering and argument is CARRIED verbatim: a post-land pass removes and shifts, it does not re-rank. **Also carried, not re-derived**: the persona consultations, and the whole "Deliberately unbanded" and "Next planning action" apparatus except where a fact under it moved. The next `/backlog` replaces the whole section)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10755 | 15606 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13225 | 2173 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**No lane moved, and the table is byte-identical to the previous stamp for the
second consecutive pass** — carried on a re-run, not on an inference. The one
landing between the stamps is **#978** (`bd22026`), which changed an error's
internal shape and one message string and could not move a lane: the accepting
verdict measured `Ratchet: unchanged` with the expectations tree clean
immediately after the run. The last movement remains `instance` +3 at `109beb9`
(#862, 2026-08-23). The `instance` lane's indeterminate census stays at 5.

### The landing this pass follows

**`bd22026` (PR #995) settled the error currency at `xpath`'s one static-error
site.** `CTATestStaticError` returns an `*xsderr.Error` carrying
`err:XPST0081` as its rule instead of an `errors.New` over a formatted string,
`ctaDefect` carries that verdict from the detection site rather than a rendered
`detail string`, and `parser`'s `ta-props-correct` charge **wraps** it
(`charge.Err = serr`, `validate/validate.go:139`'s `causedBy` shape) instead of
interpolating it — so `errors.Unwrap` + `xsderr.RuleOf` recovers the code and no
consumer scrapes a message substring for it. Arbiter **ACCEPT round 1**, zero
blocking findings, `Ratchet: unchanged`, `surface: unchanged`
(`issuecomment-5393242390` on #978). The one non-blocking finding — two
disagreeing enumerations of `xpath`'s imports in `docs/ARCHITECTURE.md` — was
disposed of in-branch as `2ca1bba`, and its durable half is filed here as
**#996**.

**Three sessions bought one landing, and the thread is why that cost was
bounded.** 08-23 evening bought the oracle grounding and let its lease lapse
with no commits; 08-24 early took the branch over, bought mason's round and
parked at the fourth consecutive transient arbiter-launch failure; the third
session bought the accepting verdict. **The grounding was written two containers
ago and used unchanged** — no second oracle round was needed.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (re-run this pass against a fresh `git fetch
-p` and the 586-issue `state=all` fetch; ages read at 09:42Z):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
732    wip/issue-732  16h18m0s   RETIRED  wip/issue-732: issue #732 is labelled needs-replan
822    wip/issue-822  194h16m0s  RETIRED  wip/issue-822: issue #822 is closed
872    wip/issue-872  160h18m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  88h31m0s   RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  27h36m0s   RETIRED  wip/issue-968: issue #968 is closed
```

**One ref left and none arrived: the namespace is now five RETIRED rows and
nothing else.** `wip/issue-978` is gone — `origin` deleted it at the
squash-merge, exactly as the scheme intends, and its content is in `main` at
`bd22026`. **No branch is CLAIMED, none is EXPIRED, no `parked/*` ref exists,
and nothing is UNKNOWN**, which is the cleanest this survey has read: every
surviving ref is retired in place and every one of them is a human's call, not a
session's.

**The lease-dating friction did not recur, and that is a fact about the window
rather than about #981.** The previous stamp's third sighting of #981's tool
half came from `wip/issue-978`'s CLAIMED row; with that row gone there is
**no CLAIMED row for the tool to mis-serve this pass**, so neither half of #981
gained a sighting. It stays banded on the three it has.

**The four closed-issue RETIRED rows are unchanged and remain a human's call.**
`wip/issue-822` and `wip/issue-872` show commits absent from `main`; that is a
squash-merge artefact, not an alarm, and both supersessions (#851, #878) landed
long ago. `wip/issue-933` (superseded by #862) and `wip/issue-968` (superseded by
#972) are the same shape. **`wip/issue-732` is the one row whose issue is open**,
`needs-replan`, and its re-plan is the next `/backlog`'s first item below.

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON —
138 issues, 50 of them open, the same input shape as the previous stamp):

```
gapaudit: 64 GAP marker(s) across 5 area(s)

=== Per-area census ===
  validate         14
  value            3
  xml              4
  xpath            6
  xsd              37

=== Group 1: markers with no OPEN tracking issue found ===
(matching is heuristic — file path or a distinctive phrase; treat a
listing here as "needs a human look", not as proven untracked)
(none)

=== Group 2: OPEN kind/gap issues with no surviving marker ===
(a stale tracker if the gap was a marked fail-open site — but
kind/gap also labels conformance-lane gaps, which never carry a
marker and belong here permanently)
  #398 cmd: the notImplemented stub is untracked P3 debt (no GAP(cmd) marker), and TestUsageCoversContract guards four substrings of a whole-block coupling
  #404 conformance: closureScan.unresolved is scan-scoped, so the decline conjunction is coarser than the hazard it models
  #569 parser: the ROOT half of the pre-scan's `<annotation>` exclusion is unpinned — a top-level `<xs:annotation><xs:appinfo><xs:key name="…">` was falsely indexed before #384 and is correctly excluded after, in both cases with no test
  #591 conformance: an instanceTest caseSpec DROPS its sibling schemaTest <schemaDocument>, so readFacetsCase declines time_minInclusive006_1163.i for a schema the catalog names explicitly
  #592 conformance: string_pattern002_1031.i — a <list itemType> over a USER-DEFINED item type in a target namespace, with a multi-leaf instance shape no cohort reader matches
  #593 conformance: decimal_totalDigits004_1060.v carries its tested value on the ROOT attribute, a shape readFacetsCase exactly-one-<foo> guard declines
  #719 validate: cvc-assertion wired fail-open at every variety level — the M6 seam, marked and measured
  #787 value: an enumeration member outside the base's value space is charged enumeration-valid-restriction (§4.3.5.5) at schema construction and never src-enumeration-value (§4.3.5.3) — which rule the spec assigns is ungrounded, and restriction.go's wrap shadows newEnumFacet's remap
  #921 conformance: <current status="queried"> is unmodeled, so the two gMonth XSD-1.0-lexical cases are permanently unbankable disagreements with no owner and no stated reason
```

**Both groups are byte-identical to the previous stamp, and this pass RE-RAN the
tool against a `.go` landing.** #978 touched `xpath/cta.go`,
`xpath/ctaparser.go` and `parser/produce_typetable.go` — three files inside the
two areas the census counts most heavily — and added, removed and re-sited no
`GAP(` marker, so **64 stands on evidence for the second consecutive pass**.
Note what it did *not* do: #894, the issue that will retire `xpath`'s
`ctatypes.go:152` marker, is untouched, so `xpath`'s 6 is unmoved by design.
**#972 is still absent from Group 2 where it belongs**, the file-path matcher
artefact owned by **#852**. **Group 1's emptiness stays qualified**: a file path
cited in a body for any reason matches every marker in that file, so an empty
Group 1 is not proof that every marker has an owner. **#960** still owns the
class the census structurally cannot see — a fail-open disclosed in PROSE
carries no `GAP(` marker and appears in neither group. **#996**, filed this
pass, is a third member of that unseeable class in a different medium: a fact
`go list` decides, hand-typed into prose four times, with no marker and no
check.

### Milestones and queue

Both columns and the queue below are RE-DERIVED this pass from one page-numbered
`state=all` fetch (586 issues, pull requests dropped), not from the milestones
endpoint.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 98 | 45 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**No milestone row moved, and that is the correct reading rather than a stall.**
Neither #978 nor #996 carries a milestone: one is an error-currency refactor
inside `xpath`/`parser` and the other is repo infrastructure. **169 of the 227
open issues carry no milestone** (227 − 45 − 13), so the rows above are feature
progress and the paragraph below is the queue. *(The previous stamp wrote 167
for this same subtraction and the arithmetic was wrong; 169 is the figure, and
it is re-derived here rather than carried.)*

Queue: **227 open — 206 `ready`, 20 `blocked`, 1 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 20), against **359 closed**.
206 + 20 + 1 = 227 exactly, and **every one of the 227 carries a queue label** —
the class #773/#774 fell into is empty for the fifteenth consecutive stamp. The
figures were re-derived by paginating page-numbered rather than with
`--paginate`, whose Link header uses numeric-ID URLs the proxy blocks, raising
the page count until a page came back empty and discarding pull requests, which
share the endpoint.

**Every move reconciles, and the totals are unchanged because two moves
cancelled.** From the previous stamp's 227 open / 206 `ready` / 20 `blocked` /
1 `needs-replan` / 358 closed: **#978 closed** (−1 open, −1 `ready`, +1 closed);
**#996 filed `ready` by this pass** (+1 open, +1 `ready`). 227 − 1 + 1 = **227**;
206 − 1 + 1 = **206**; `blocked` untouched at 20; `needs-replan` untouched at 1;
closed 358 + 1 = **359**. An unchanged open count across a landing is a
coincidence of arithmetic here, not a stalled window — say so, because two
consecutive stamps reading 227 otherwise looks like a survey that was carried
rather than re-run.

**Five `ready` issues are STILL done, and the count above still overstates what
is startable by five.** **#625**, **#748**, **#492**, **#934** and **#896** were
discharged by `34a8043` and judged with it, and all five remain open because
GitHub bound `Closes #669, #625, …` to the one reference following the keyword.
**#993** owns the mechanism and is banded; each of the five carries a comment
naming the landing and telling a session not to take it. **They are not closed
here**: the cartographer files, unblocks and restamps, and never closes an issue
as done — that is the develop loop's act, and it is still the first item of the
next planning action below, now carried for a second stamp.

**The unblock sweep relabelled nothing, and could not have — the zero case,
stated mechanically.** All 227 open bodies were fetched over `gh api` —
byte-faithful, where MCP `issue_read` is lossy (#764) — and searched for `#978`
as a token in title and body, and separately inside each body's `## Depends on`
section. **Exactly one open issue cites #978 at all, and it is #996, filed by
this pass**; **no open `## Depends on` names #978**; and **none of the 20
`blocked` bodies contains the string `978` anywhere**. #894, whose sequencing
behind #978 this pass discharged, does not cite it in its body either — correct,
and by design: that was a cost ordering, recorded on the two threads, and
marking #894 `blocked` on it would have been wrong.

**No stale premise was found in a body this pass.** #978's landing changed one
package's internal error shape and no exported identifier (`surface:
unchanged`), so no open body's claim about a surface or a call site could have
gone stale on it. The one body-level fact the landing *does* bear on — #894's
Notes on the cast-target codes — was checked and **stands exactly as written**;
what the landing changed for it is recorded on its thread rather than written
into its body, because the body states the decision and the thread holds the
reasoning.

### Persona consultations — none handed to this pass, and none invented

**Nothing was handed here, so nothing is folded.** A post-land pass runs no
persona; the cartographer has read the source, so a persona it role-played would
launder an insider's opinion as an outsider's (#416's option (a)). The standing
record is the 2026-08-23 `/backlog`'s consultation — the **fifth consecutive**,
the first in five to change a verdict — and its top-line finding was paid off by
`34a8043`. **Six of the eleven standing persona defects are discharged**, so the
sixth consultation should reconfirm five rather than eleven. That consultation
belongs to the next `/backlog`.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
of it. Take from the top. Cross-references are stated by ISSUE rather than by row
number, which decays at each re-cut.

**One row left and none arrived; everything below it shifted up unchanged.** The
departure is **#978** (row 2), landed. Every surviving row's argument is carried
verbatim from the 2026-08-23 `/backlog` — a post-land pass shifts and removes, it
does not re-rank, and the next `/backlog` re-derives the ordering.

**Nothing in the band is claimed, and nothing in the `wip/*` namespace is
live.** All five refs are RETIRED (above), so **every row here is takeable by
the next session that reads this table**.

| # | Issue | Why here |
|---:|---|---|
| 1 | #870 + #747 + #514 + #687 + #672 | **PROMOTED from row 13 by the 2026-08-23 `/backlog`, and #720's body independently asks for three of them first**: *"Take #514, #672 and #687 before this lands. Each is a sentence or a dispatch branch while the CLI surface is still empty; taken afterwards every one of them is a change to shipped behaviour."* #472 is `ready` and is what makes that true. Fifth consecutive reconfirmation; **#687 now carries four triggers of one raw-token scan and #672 a spelling collision that narrows its own decision**. The CLI is still unreachable from its own documentation |
| 2 | #843 | **The steward rule's own named exemplar, and the steepest ranking of the three it banded.** Four hand-maintained copies of one complex-type descent have **already diverged** on the redefine-original containment edge — two of the four do not descend `c.Base()` at all. M4's open count is **45**, unchanged this pass and up from the 42 the 2026-08-16 audit recorded, so each new finalize-time constraint is a fifth copy picking its edges by eye. The bugs are **fail-open**, invisible to the ratchet, which is why an audit and not the suite found this one. **Sizing is the open question, not value** — a session returning only a design comment naming the parameterization has done the right amount. **It is now the last of the three steward-ranked refactors still unlanded** (#979 landed 08-23, #978 landed 08-24) |
| 3 | #981 | **Filed by the 2026-08-23 `/backlog`, which charged its friction to itself.** `docs/WORKFLOW.md`'s empty-claim lease is dated by ANY thread comment, so an oracle grounding posted 44 minutes earlier locked that pass's band head for two hours — and `wipsurvey` cannot apply the rule at all, printing *"settle it from the issue thread"* for every CLAIMED row. Banded on CLAUDE.md's cost rule: one session, doc-first, and the tool arm may split off. Standing at **three sightings of the tool half against one of the rule half**; this pass added neither, because the namespace holds no CLAIMED row for the tool to mis-serve |
| 4 | #963 | **The tax falls on every landing and its discriminator STILL could not fire.** #820 landed the *form* of landing precondition 1; nothing checks that the check was run, which is **#924**'s shape and is not reachable by prose. One session — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part (#304 made CLAUDE.md's Commands block the sole gate definition). **`bd22026` carried `docs/LOG/2026-08.md` inside the squash**, so the discriminator did not fire on evidence for the **fourth** consecutive window; it is banded on cost, not on sightings |
| 5 | #993 | **The only band row with an unpaid cost sitting in the queue right now, and it is now a stamp old.** A multi-issue landing's `Closes #A, #B, #C` closes only #A — PR #991 named six and closed one, so **#625, #748, #492, #934 and #896 are `ready` and done**, and a session may pick one tomorrow and spend a container discovering it. One WORKFLOW sentence plus a post-merge check. Banded on CLAUDE.md's cost rule at ONE witness, above three lane slices, because the cost is live and unpaid rather than paid and absorbed — the discriminator that keeps **#992** and **#996** out of the band. Read beside #963: same *"the form landed and nothing runs it"* family, but this is the only member whose check must run AFTER the merge, so folding it into `landcheck` is a design option and not the obvious answer |
| 6 | #493 | **On CLAUDE.md's cost rule; the friction was paid by hand three windows ago and the payment is still unlanded.** `docs/WORKFLOW.md`'s park step names neither the close reason nor the `ready` clear, so #968's session had to discover that a parked issue carrying `ready` is pickable. Doc-only, one session, and it targets the **park** paragraph — not the filing-discipline paragraph #510, #646, #679 and #912 all target — so it rebases against none of them. **A second live corroboration stands from the previous pass**: #732 was parked `needs-replan` and its `ready` label was correctly cleared, but `wipsurvey`'s RETIRED rows still include #933, closed `not_planned` and still carrying `ready` |
| 7 | #846 | **#909 paid the shadow tax and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`.** Its promotion rule asks for a third landing that pays the tax; **#732's attempt is the nearest thing and it did not pay it** — it was rejected on a regressing instance case, not on the shadow, and #978 could not pay it either (see the discriminator below). Its structural argument from #968's family A stands unchanged: the fault deciding those four documents lives outside `<restriction>`'s child list, so no predicate over that list can tell a fabricated verdict from a correct one. Still a ~700-line refactor with no evidence it fits one session |
| 8 | #941 | **#387's own unfiled debt, and the coldest files of any band row** (`07117dc`, 2026-08-21). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete, `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 9 | #953 | **A doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says "every caller turns a decided negative into a schema rejection"; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules |
| 10 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 11 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 12 | #975 | Nine s4s-grammar rejection messages name no Appendix A production — the bar `xsderr/doc.go` states as of #966 — censused at `ebf6fa2` with the re-deriving criterion stated ahead of the list. Cheap, mechanical, moves no lane. Its #884 ordering is DISCHARGED and the 21-site criterion was re-run at `b3f295a` to the same nine sites at the same line numbers, so any cheap session can take it now. **#444 already owns the tenth deficient site** (`rejectBothInlineTypes`) on a stronger footing and is excluded rather than duplicated |

**Left the band this pass, and why.** **#978** — landed at `bd22026`, the second
of the three steward cost-of-delay refactors to land and the first ever banded
under that rule. **The rule's own prediction held and is worth recording once**:
it was ranked above two lane slices because #894 would double its diff, #894 did
not land first, and the diff that landed is the small one the ranking priced —
`xpath/cta.go` +34/−12, `parser/produce_typetable.go` +15/−5, one unexported
constant, one wrap. Nothing else left the band.

**Deliberately unbanded, and why.** **#996** is this pass's one filing —
`docs/ARCHITECTURE.md` states each package's in-module imports in up to four
places, no gate part reads any of them, and `go list -f '{{.Imports}}'` decides
the fact in one command. **Two witnesses one landing apart** (#979 corrected note
[1] and left the graph line stale; #978 hit the same defect and the arbiter
caught it only because that diff touched both copies) plus a **third instance
standing on `main` right now**: `parser`'s graph line at `:29-32` omits `xsderr`
and `internal/schemaloc`, found by running `go list` here. It is unbanded on the
#992/#993 discriminator — the two paid witnesses were **paid and absorbed**
(`2ca1bba` fixed the xpath copies), and the live third is one wrong noun in a
paragraph rather than a wrong row in the queue — and it owes a
prose-parseability decision it does not start with. Note for whoever takes it:
**the wrong `parser` line is deliberately left in place as the regression
fixture**, so the check fails on `main` before it passes. **#992** is unbanded on
the same discriminator, unchanged. **#989** is #979's post-land filing —
whether PRINCIPLES 26 requires generating `regex`'s NameStartChar/NameChar
tables from `docs/specs/md/xml.md`; decision-first and cheap, but it moves no
lane and carries no steward ranking, so a `/backlog` places it. **#972** stays
`blocked` on #732, a **retired** issue, and **#786**'s deletion branch waits on
#972 in turn, so that whole chain is stalled behind the re-plan below.
**#894** is now **clear to proceed** — its #978 ordering is discharged, the
discharge and what the landing changed for it are on its thread, its
`## Depends on` is `none` and always was, and the two codes it adds now cost two
constants beside `ruleXPST0081` rather than a second plumbing diff; it stays
unbanded only because it is a lane-flat correctness fix whose first step is an
unsettled oracle question (its Note 2), not because anything is waiting on it.
**#888** and **#889** still await a suite census in their range. **#907**'s
`childElement` census is stale by at least four landings and must be re-derived
before anything is designed from it. **#885**'s three discriminators still have
one sighting each. **#409** is `ready` since 2026-08-02 with a fourth independent
sighting; it stays unbanded only because it is one row of a five-file convention
landing. **#854** is the godoc half of the previous landing, refreshed in-body
last pass and sequenced behind nothing. **#670** asks for
`parser/example_test.go` and is untouched. **#937** is naturally folded by the
next landing touching `rejectRepeatedAnnotations`. **#920** and **#921** are
conformance-bookkeeping follow-ups below the fold. **#929** and **#931** are the
small parser occurrence / rule-mapping gaps #901 exposed. **#455** is the live
owner of the `strings.TrimSpace`-versus-§4.3.6 character class at **ten** sites,
and **#456** stays `blocked` on it. **#843–#849** are the 2026-08-16 audit's
findings, **six open**, of which #843 is banded and **#841** is `blocked` on a
trigger that fired without a ruling. **#566** is #565's open sibling, routed
nowhere by #565's landing and correctly so — note that #992 is filed against the
same landed check and does not subsume it. **#852** owns both directions of the
`gapaudit` matcher defect and stays below the fold because the tool again ran
with reconciliation. **#692** and **#925** are still `blocked` on a `/retro`
trigger that fired without ruling on them. **#570** carries the standing `schema`
decline-count argument at 893 against a re-measured 788, unchanged this pass
because no conformance measurement was taken here.

### Next planning action

**Close the five, then re-plan #732.** Both items are carried unchanged from the
previous stamp because neither has been taken; a post-land pass cannot do
either.

1. **Five issues are done and open, for a second consecutive stamp.** #625,
   #748, #492, #934 and #896 are discharged by `34a8043` and judged with it;
   only #669 closed. Closing them is the develop loop's act — the cartographer
   never closes an issue as done — and it is one
   `gh api ... -f state=closed -f state_reason=completed` per issue. Until it
   happens the `ready` count above overstates what is startable by five. The
   mechanism is **#993**, banded at row 5.
2. **#732 is `needs-replan` and needs the re-plan.** `docs/WORKFLOW.md` is
   explicit: after re-planning, **the cartographer closes the issue as superseded
   and files a replacement**. That is a `/backlog` act — it needs the arbiter's
   reject read in full, the regressing `instance` case understood, and §4.2.2
   scoped afresh. It is the **next `/backlog`'s first item**, and it unblocks
   **#972**, which unblocks the argument **#786** has been waiting on.
3. **The whole band is startable and unclaimed**, for the first time in several
   stamps: no ref is CLAIMED and no ref is EXPIRED. Row 1
   (#870 + #747 + #514 + #687 + #672) has been reconfirmed five consecutive
   passes and the CLI is still unreachable from its own documentation.

**The two standing promotion discriminators both had a subject this window and
neither is FIRED here** — a post-land pass reports, and it is the next
`/backlog` that fires a promotion. **#963's** (did the landing carry its
`docs/LOG` entry inside the squash?): `git show --stat bd22026` is eight files
with `docs/LOG/2026-08.md` among them — **carried**, so the discriminator did not
fire on evidence for the fourth consecutive window, and #963 stays banded on
cost. **#846's** (did the entry record the shadow tax?): it did not, and it could
not — #978 touches `xpath` and `parser/produce_typetable.go` and comes nowhere
near `conformance/schema.go`'s shadow of `produce_complex.go`. Say which way both
fell on the next stamp.

**The steward-ranking rule has now produced two landings out of three bandings,
and the third is row 2.** #979 landed 08-23, #978 landed 08-24, #843 stands.
**#841 remains the counter-example the rule cannot reach** — a `kind/refactor`
with a steward ranking that stays `blocked` because its trigger has no mailbox.
That gap is on #841's thread and is still not filed as an issue, because the fix
belongs to whichever pass gives Part 2 of the `/retro` the mailbox Part 1 has.

**Standing, unchanged, and still true.** Four unlanded corrections target one
paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**, **#646**,
**#679**, **#912** — and whichever lands last rebases three times; #493 is not a
fifth and #992 is not a sixth, both targeting different paragraphs of the same
file. **The next `/retro` inherits three issues it already owed** — #692, #925
and #841. **The CTA cohort's 45 banked `instance` failures remain
unattributed**, thirteenth consecutive stamp carrying it. **`gate.yml` runs but
is still not a required status check**, which only the repository owner can
change.

**Two environment costs are recorded and deliberately NOT filed, each at one
witness.** The first is carried: uncached conformance test binaries hang under
the default sandbox, so conformance runs must be issued unsandboxed
(`docs/LOG/2026-08.md`, 2026-08-24 post-land entry) — **still one witness**, and
this pass took no conformance measurement, so it neither corroborated nor
weakened it. The second is new: **four consecutive arbiter launches died on
transient platform errors** before any reached the gate, and **no document says
how many is enough** — WORKFLOW makes ending at a checkpoint a first-class
outcome and names *"a wall"*, but neither it nor ROUTINES, `develop.md` or
`arbiter.md` mentions a transient launch failure at all, so the threshold
actually used (three retries, park on the fourth) was invented by that session
and is unwritten. This pass searched `docs/LOG` for prior sightings — `529`,
`Overloaded`, `launch fail`, `transient platform` — and **found no earlier
witness**, so it is one, and #978's own log entry set the bar: *"If a second
session pays it, it belongs in `docs/WORKFLOW.md`'s checkpoint paragraph as its
own sentence — what counts as a wall when the wall is infrastructure."* Both
stay in the log until a second sighting.

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
the producer drops), both landed on 2026-08-22/23; live ones in the same
family are **#884** (every malformed named `<group>` body collapsing into one
`mgd-props-correct` message that names an internal invariant rather than the
grammar it broke — the working band's head, and grounded against the
`xs:namedGroup` production this window), **#471** (a local `<element ref=>`
carrying `substitutionGroup=`, silently accepted), **#931** (occurrence
attributes on a named `<group>`'s child compositor), **#929**, **#455**, and
**#972** (an XSD-namespace child §4.1.2's `<simpleType><restriction>` has no
position for, dropped by `restrictionFacets` — `blocked` on #732, which owns the
§4.2.2 conditional inclusion the same site needs first). A second, narrower
family has opened beside it: the rejections the producer already makes
**correctly but describes badly**, whose bar `xsderr/doc.go` set with #966 and
whose nine deficient sites **#975** owns. The GitHub milestone holds the feature
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
+9, unioned onto #716's). `instance` stands at **10752** — #913's cvc-type
clause 3.1 landing added **9409**, itself M5 and the largest single lane move
this project has recorded — and **twenty-five** of the pre-#913 cases were not
M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure (+12), #821
added 1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). (The
*thirteen* this paragraph carried predated the CTA pair and did not sum; the
five figures do, and the sum is the number.) **It happened again after #913:
#909 — an M4 landing — took `instance` 10746 → 10752 (+6) by producing
`<simpleContent>` `<restriction>`, so the outside-M5 total is now 31.** A slice
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

**10752 is still a floor built for soundness, and #913's +9409 jump did not
change what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the 15609 that
still fail are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read 10752 as 41% of the suite passing.** It is the count of documents
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
