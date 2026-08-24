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

## Status — 2026-08-24 (POST-LAND PASS for the #669 six-issue chain, landed as PR #991 / `34a8043`. Replaced whole, per step 6, never patched in place: the lane table is a fresh `go tool lanestatus` paste, the branch namespace a fresh `wipsurvey` against a fresh `git fetch`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh 583-issue page-numbered `state=all` fetch taken at 02:58Z. **Three band rows leave at once** — row 1 #732 retired to `needs-replan`, row 2 the six-issue chain landed, row 14 #979 landed — and every surviving row's ordering and argument is CARRIED verbatim: a post-land pass removes and shifts, it does not re-rank. **Also carried, not re-derived**: the persona consultations. **Dropped rather than carried**: the previous stamp's two mid-window rule subsections, whose subject — #884's and #862's leases — is spent, and whose rules are now in the agent files where they belong. The next `/backlog` replaces the whole section)

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

**No lane moved, and the table is byte-identical to the previous stamp** —
carried on a re-run, not on an inference. Three landings fall between the two
stamps and none could have moved a lane: **#979** (`9092a6d`) routed
`ctaScanNCName` and `icScanNCName` onto `regex`'s generated NameStartChar /
NameChar classes, a deduplication whose accepting verdict measured the character
sets as identical; **#669** and its post-land sibling are doc-only. The last
movement remains `instance` +3 at `109beb9` (#862, 2026-08-23). The `instance`
lane's indeterminate census stays at 5.

### The landing this pass follows

**`34a8043` (PR #991) closed a six-issue documentation-accuracy chain that had
sat five consecutive passes** — `README.md` and `validate/doc.go`, no Go code,
`go tool surface -base origin/main` unchanged, `Ratchet: unchanged` measured
twice. The README's "works TODAY" Library snippet is now a `func main` that
compiles and runs as pasted; the example-tests pointer list gains the `xsd`
entries; `parser.ParseReport` / `AssemblyReport` are documented; the *"instance
validation is PLANNED"* framing is replaced by real `validate.New` /
`xmlsrc.Validate` usage now that M5 has shipped; and `validate/doc.go` states
that `Result.Violations` is the verdict while `Result.Err` reports only whether
the walk finished. Arbiter: reject round 1 on one finding, accept round 2 with
zero (`issuecomment-5390125441` on #669).

**Only #669 actually closed.** See the queue section below — the other five are
`ready` and done, and **#993** owns the mechanism.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (re-run this pass against a fresh
`git fetch` and the 583-issue page-numbered fetch; ages read at 02:59Z):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
732    wip/issue-732  9h39m0s    RETIRED  wip/issue-732: issue #732 is labelled needs-replan
822    wip/issue-822  187h37m0s  RETIRED  wip/issue-822: issue #822 is closed
872    wip/issue-872  153h39m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  81h52m0s   RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  20h57m0s   RETIRED  wip/issue-968: issue #968 is closed
978    wip/issue-978  main's     CLAIMED  wip/issue-978: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
```

**One ref left, one arrived, and one row changed verdict without moving.**
`wip/issue-669` is gone — `origin` deleted it at the squash-merge, exactly as the
scheme intends. `wip/issue-978` is new: created from `main` at `dd2d802`
(19:19:16Z) and carrying no commits of its own. `wip/issue-732` is the row that
changed: CLAIMED at the previous stamp, now **RETIRED**, because its session
parked the issue `needs-replan` on the arbiter's reject (`issuecomment` at
17:37:38Z) rather than spending a repair round. The branch is retired **in
place**; nothing is renamed or deleted.

**`wip/issue-978`'s lease has LAPSED and the issue is takeable.** Settled by hand
from the thread, as the survey's verdict instructs: under `docs/WORKFLOW.md`'s
empty-claim rule an unpushed claim is dated by its newest issue-thread comment,
which is a **GROUNDING at 19:22:56Z** — **7h 36m** against the 2h TTL. **This is
the third consecutive sighting of #981's tool half**: `wipsurvey` reads
`{number, state, labels}` from stdin, never sees a comment timestamp, and so
prints *"settle it from the issue thread"* for every CLAIMED row it meets. What
#978 does **not** corroborate is #981's rule half — the grounding here is old
enough that dating the lease by it changes no answer — so that side still stands
on one sighting, `wip/issue-884`'s.

**The four closed-issue RETIRED rows are unchanged and remain a human's call.**
`wip/issue-822` and `wip/issue-872` show commits absent from `main`; that is a
squash-merge artefact, not an alarm, and both supersessions (#851, #878) landed
long ago. `wip/issue-933` (superseded by #862) and `wip/issue-968` (superseded by
#972) are the same shape. No `parked/*` ref exists and nothing is EXPIRED.

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON —
138 issues, 50 of them open):

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
tool.** The measurement is worth more than it was: **#979 is a `.go` landing that
touched exactly the kind of site the census counts** — it deleted one of two
byte-identical NCName scanners and rerouted the other — and it added, removed and
re-sited no `GAP(` marker, so 64 stands on evidence and not on "doc-only, nothing
can have changed". **#972 is still absent from Group 2 where it belongs**, the
file-path matcher artefact owned by **#852**. **Group 1's emptiness stays
qualified**: a file path cited in a body for any reason matches every marker in
that file, so an empty Group 1 is not proof that every marker has an owner.
**#960** still owns the class the census structurally cannot see — a fail-open
disclosed in PROSE carries no `GAP(` marker and appears in neither group.

### Milestones and queue

Both columns and the queue below are RE-DERIVED this pass from one page-numbered
`state=all` fetch (583 issues, pull requests dropped), not from the milestones
endpoint.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 98 | 45 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**No milestone row moved, and that is the correct reading rather than a stall.**
Neither #979 nor any of the six chain issues carries a milestone at all: one is
an internal deduplication and six are documentation accuracy. **167 of the 227
open issues carry no milestone** (227 − 45 − 13), so the rows above are feature
progress and the paragraph below is the queue.

Queue: **227 open — 206 `ready`, 20 `blocked`, 1 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 20), against **358 closed**.
206 + 20 + 1 = 227 exactly, and **every one of the 227 carries a queue label** —
the class #773/#774 fell into is empty for the fourteenth consecutive stamp. The
figures were re-derived by paginating page-numbered rather than with
`--paginate`, whose Link header uses numeric-ID URLs the proxy blocks, raising
the page count until a page came back empty (page 11) and discarding pull
requests, which share the endpoint. The fetch read 225 open / 204 `ready`; the
two issues **this pass filed** (#992, #993) are added to it by hand and are the
only difference.

**Every move reconciles.** From the previous stamp's 226 open / 206 `ready` /
20 `blocked` / 0 `needs-replan` / 356 closed: #979 closed (−1 open, −1 `ready`,
+1 closed); #669 closed (−1, −1, +1); #989 filed `ready` by #979's post-land pass
(+1, +1); **#732 relabelled `ready` → `needs-replan`** (−1 `ready`, +1
`needs-replan`) when its session parked it; #992 and #993 filed `ready` here
(+2, +2). 226 − 1 − 1 + 1 + 2 = **227**; 206 − 1 − 1 + 1 − 1 + 2 = **206**;
`blocked` untouched at 20; closed 356 + 2 = **358**.

**Five `ready` issues are already done, and that is a defect rather than a
backlog.** **#625**, **#748**, **#492**, **#934** and **#896** were all
discharged by `34a8043` and judged with it, and all five are still open: both the
PR body and the squash commit carry `Closes #669, #625, #748, #492, #934,
#896.`, and GitHub binds a closing keyword to the ONE reference that follows it,
so **#669 closed and the other five did not** (`closed_at` is null on all five,
which rules out a close-then-reopen). **#993 is filed against the mechanism**,
with PR #991 as its regression fixture and these five as its named live
subjects; each of the five carries a comment naming the landing and telling a
session not to take it. **They are not closed here**: the cartographer files,
unblocks and restamps, and never closes an issue as done — that is the develop
loop's act, and until it happens the 206 above overstates what is startable by
five.

**The unblock sweep relabelled nothing, and could not have — the zero case,
stated.** All 225 open bodies were fetched over `gh api` — byte-faithful, where
MCP `issue_read` is lossy (#764) — and searched for each of the six landed
numbers as a token. Twelve open issues cite one or more of them (#409, #513,
#670, #688, #854, #857, #893, #953 and four members of the chain itself); **not
one `## Depends on` outside the chain names any of the six**, and none of the 20
`blocked` bodies contains any of the six numbers at all. So no issue was
unblocked, and none could have been. The only `## Depends on` hits are internal
to the chain — #492 on #669 and #896 sequenced after #748 — both moot now.

**One stale premise found and fixed in the body, not commented at.** **#854**'s
Notes read *"Its right home is beside band row 9, not inside it. #625, #669,
#748 and #492 are the README's Library block in file order, all four in one
row"* — a row that no longer exists, and a cross-reference by row number that
`docs/PLAN.md`'s own convention forbids. The bullet now records that the README
half landed at `34a8043`, that #625's `INTENDED CALLER` ask is discharged there
as the *"PRODUCER surface, not application surface"* paragraph, and that #854 is
the untouched godoc half, standalone and sequenced behind nothing. **#670**'s
premise was checked and stands exactly: it asks for `parser/example_test.go` in
the README's pointer list *only*, and the landing added the `xsd` entries and no
`parser` one, because that file does not exist yet. The other ten citations are
"adjacent, not gating" by their own words and none went stale.

### Persona consultations — none handed to this pass, and none invented

**Nothing was handed here, so nothing is folded.** A post-land pass runs no
persona; the cartographer has read the source, so a persona it role-played would
launder an insider's opinion as an outsider's (#416's option (a)). The standing
record is the 2026-08-23 `/backlog`'s consultation — the **fifth consecutive**,
the first in five to change a verdict, and it filed nothing while refreshing
twelve bodies with re-derived evidence. That consultation's own top-line finding
was that the README's Library block lies to a first-time reader; **`34a8043` is
that finding paid off**, which is the strongest thing this pass can say about
it. The sixth consultation belongs to the next `/backlog`.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
of it. Take from the top. Cross-references are stated by ISSUE rather than by row
number, which decays at each re-cut.

**Three rows left and one arrived; everything else shifted up unchanged.** The
departures: **#732** (row 1) is `needs-replan` and out of the `ready` queue
entirely; **the six-issue chain** (row 2) landed; **#979** (row 14) landed. The
arrival is **#993** at row 6. Every surviving row's argument is carried verbatim
from the 2026-08-23 `/backlog` — a post-land pass shifts and removes, it does not
re-rank, and the next `/backlog` re-derives the ordering.

**Nothing in the band is claimed.** The whole `wip/*` namespace is five RETIRED
refs and `wip/issue-978`, whose lease lapsed 7h ago (above). Row 2 is #978
itself: it is takeable, and a session taking it should read its grounding first
rather than re-buying one.

| # | Issue | Why here |
|---:|---|---|
| 1 | #870 + #747 + #514 + #687 + #672 | **PROMOTED from row 13 by the 2026-08-23 `/backlog`, and #720's body independently asks for three of them first**: *"Take #514, #672 and #687 before this lands. Each is a sentence or a dispatch branch while the CLI surface is still empty; taken afterwards every one of them is a change to shipped behaviour."* #472 is `ready` and is what makes that true. Fifth consecutive reconfirmation; **#687 now carries four triggers of one raw-token scan and #672 a spelling collision that narrows its own decision**. The CLI is still unreachable from its own documentation |
| 2 | #978 | **The first issue ever banded on `.claude/agents/cartographer.md`'s steward-ranking rule.** `xpath` charges XPath/F&O error codes as message SUBSTRINGS while `regex` tags the same class as an `xsderr.Rule`. Ranked above #843 for one reason: **#894 is open and `ready`**, adds `err:XPST0051`/`err:XPST0080` to this exact function, and in #978's own words *"doubles this issue's diff"* if it lands first. Today the change is one error site, one constant, one wrap. The ordering #978-before-#894 is a cost ordering, not a dependency; neither `## Depends on` names the other and neither should. **Its branch carries an oracle grounding and no commits — read the thread, do not re-ground** |
| 3 | #843 | **The steward rule's own named exemplar, and the steepest ranking of the three.** Four hand-maintained copies of one complex-type descent have **already diverged** on the redefine-original containment edge — two of the four do not descend `c.Base()` at all. M4's open count is **45**, up from the 42 the 2026-08-16 audit recorded, so each new finalize-time constraint is a fifth copy picking its edges by eye. The bugs are **fail-open**, invisible to the ratchet, which is why an audit and not the suite found this one. **Sizing is the open question, not value** — a session returning only a design comment naming the parameterization has done the right amount |
| 4 | #981 | **Filed by the 2026-08-23 `/backlog`, which charged its friction to itself.** `docs/WORKFLOW.md`'s empty-claim lease is dated by ANY thread comment, so an oracle grounding posted 44 minutes earlier locked that pass's band head for two hours — and `wipsurvey` cannot apply the rule at all, printing *"settle it from the issue thread"* for every CLAIMED row. Banded on CLAUDE.md's cost rule: one session, doc-first, and the tool arm may split off. **The tool half now has a THIRD sighting** (`wip/issue-978`, this pass) against the rule half's one; a session may reasonably land the tool arm alone |
| 5 | #963 | **The tax falls on every landing and its discriminator STILL could not fire.** #820 landed the *form* of landing precondition 1; nothing checks that the check was run, which is **#924**'s shape and is not reachable by prose. One session — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part (#304 made CLAUDE.md's Commands block the sole gate definition). **Both landings this window carried their entries**, so the discriminator did not fire on evidence again; it is banded on cost, not on sightings |
| 6 | #993 | **NEW, filed by this pass, and the only band row with an unpaid cost sitting in the queue right now.** A multi-issue landing's `Closes #A, #B, #C` closes only #A — PR #991 named six and closed one, so **#625, #748, #492, #934 and #896 are `ready` and done**, and a session may pick one tomorrow and spend a container discovering it. One WORKFLOW sentence plus a post-merge check. Banded on CLAUDE.md's cost rule at ONE witness, above three lane slices, because the cost is live and unpaid rather than paid and absorbed — the discriminator that keeps **#992** out of the band. Read beside #963: same *"the form landed and nothing runs it"* family, but this is the only member whose check must run AFTER the merge, so folding it into `landcheck` is a design option and not the obvious answer |
| 7 | #493 | **On CLAUDE.md's cost rule; the friction was paid by hand three windows ago and the payment is still unlanded.** `docs/WORKFLOW.md`'s park step names neither the close reason nor the `ready` clear, so #968's session had to discover that a parked issue carrying `ready` is pickable. Doc-only, one session, and it targets the **park** paragraph — not the filing-discipline paragraph #510, #646, #679 and #912 all target — so it rebases against none of them. **A second live corroboration this pass**: #732 was parked `needs-replan` and its `ready` label was correctly cleared, but `wipsurvey`'s RETIRED rows still include #933, closed `not_planned` and still carrying `ready` |
| 8 | #846 | **#909 paid the shadow tax and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`.** Its promotion rule asks for a third landing that pays the tax; **#732's attempt is the nearest thing this window and it did not pay it** — it was rejected on a regressing instance case, not on the shadow. Its structural argument from #968's family A stands unchanged: the fault deciding those four documents lives outside `<restriction>`'s child list, so no predicate over that list can tell a fabricated verdict from a correct one. Still a ~700-line refactor with no evidence it fits one session |
| 9 | #941 | **#387's own unfiled debt, and the coldest files of any band row** (`07117dc`, 2026-08-21). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete, `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 10 | #953 | **A doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says "every caller turns a decided negative into a schema rejection"; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules |
| 11 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 12 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 13 | #975 | Nine s4s-grammar rejection messages name no Appendix A production — the bar `xsderr/doc.go` states as of #966 — censused at `ebf6fa2` with the re-deriving criterion stated ahead of the list. Cheap, mechanical, moves no lane. Its #884 ordering is DISCHARGED and the 21-site criterion was re-run at `b3f295a` to the same nine sites at the same line numbers, so any cheap session can take it now. **#444 already owns the tenth deficient site** (`rejectBothInlineTypes`) on a stronger footing and is excluded rather than duplicated |

**Left the band this pass, and why.** **#732** — parked `needs-replan` on the
arbiter's reject, a repair round deliberately not spent. It is out of the `ready`
queue by label, its branch is RETIRED in place, and **re-planning it is a
`/backlog` act, not a post-land one**: see the next planning action. **The
#669 → #625 → #748 → #896 → #492 → #934 chain** — landed at `34a8043`, five
members still open on #993's mechanism, none of them takeable. **#979** — landed
at `9092a6d`, the last of the three steward-ranked refactors to be banded and the
first to land.

**Deliberately unbanded, and why.** **#992** is this pass's other filing —
landing precondition 3 accepts a `MASON:` comment whose whole body is a dead
container-local path, which cost this session one extra mason round to
reconstruct the account for `5d4426c` before landing could proceed. One witness,
and **the cost was paid and absorbed** rather than left standing, which is what
separates it from #993; it also owes a doc-versus-tooling decision it does not
start with. It waits for a `/backlog` or a second witness. **#989** is the other
recent filing, by #979's post-land pass: whether PRINCIPLES 26 requires
generating `regex`'s NameStartChar/NameChar tables from `docs/specs/md/xml.md`.
Decision-first, cheap, and strengthened by #979 having routed two more callers
into those tables — but it moves no lane and carries no steward ranking, so a
`/backlog` places it rather than this pass. **#972** stays `blocked` on #732 and
is now blocked on a **retired** issue, which is a worse position than it held at
the previous stamp; **#786**'s deletion branch waits on #972 in turn, so that
whole chain is stalled behind the re-plan below. **#907**'s `childElement`
census is stale by at least four landings and must be re-derived before anything
is designed from it. **#885**'s three discriminators still have one sighting
each. **#409** is `ready` since 2026-08-02 with a fourth independent sighting; it
stays unbanded only because it is one row of a five-file convention landing.
**#894** should be taken **after** #978 — see #978's thread; **#888** and **#889**
still await a suite census in their range. **#854** is the godoc half of the
landing above, refreshed in-body this pass and now sequenced behind nothing.
**#670** asks for `parser/example_test.go` and is untouched by the landing.
**#937** is naturally folded by the next landing touching
`rejectRepeatedAnnotations`. **#920** and **#921** are conformance-bookkeeping
follow-ups below the fold. **#929** and **#931** are the small parser occurrence
/ rule-mapping gaps #901 exposed. **#455** is the live owner of the
`strings.TrimSpace`-versus-§4.3.6 character class at **ten** sites, and **#456**
stays `blocked` on it. **#843–#849** are the 2026-08-16 audit's findings, **six
open**, of which #843 is banded and **#841** is `blocked` on a trigger that fired
without a ruling. **#566** is #565's open sibling, routed nowhere by #565's
landing and correctly so — note that #992 is filed against the same landed
check and does not subsume it. **#852** owns both directions of the `gapaudit`
matcher defect and stays below the fold because the tool again ran with
reconciliation. **#692** and **#925** are still `blocked` on a `/retro` trigger
that fired without ruling on them. **#570** carries the standing `schema`
decline-count argument at 893 against a re-measured 788, unchanged this pass
because no conformance measurement was taken here.

### Next planning action

**Close the five, then re-plan #732.**

1. **Five issues are done and open.** #625, #748, #492, #934 and #896 are
   discharged by `34a8043` and judged with it; only #669 closed. Closing them is
   the develop loop's act — the cartographer never closes an issue as done — and
   it is one `gh api ... -f state=closed -f state_reason=completed` per issue.
   Until it happens the `ready` count above overstates what is startable by five.
   The mechanism is **#993**, banded at row 6.
2. **#732 is `needs-replan` and needs the re-plan.** `docs/WORKFLOW.md` is
   explicit: after re-planning, **the cartographer closes the issue as superseded
   and files a replacement**. That is a `/backlog` act — it needs the arbiter's
   reject read in full, the regressing `instance` case understood, and §4.2.2
   scoped afresh — and this pass deliberately did not attempt it. It is the
   **next `/backlog`'s first item**, and it unblocks **#972**, which unblocks the
   argument **#786** has been waiting on.
3. **The band's head is startable and unclaimed.** Row 1 (#870 + #747 + #514 +
   #687 + #672) has been reconfirmed five consecutive passes and the CLI is still
   unreachable from its own documentation. Row 2 (#978) is takeable with a
   grounding already bought and on its thread.

**The two standing promotion discriminators both had a subject this window and
neither is FIRED here** — a post-land pass reports, and it is the next
`/backlog` that fires a promotion. The facts, already measured, for it to start
from. **#963's** (did the landing carry its `docs/LOG` entry inside the squash?):
`git show --stat 9092a6d` is eight files with `docs/LOG/2026-08.md` among them,
and `34a8043` is three files with `docs/LOG/2026-08.md` among them — **both
carried**, so the discriminator did not fire on evidence for the third
consecutive window, and #963 stays banded on cost. **#846's** (did the entry
record the shadow tax?): neither did, and neither could — #979 is a scanner
deduplication and #669 is documentation. Say which way both fell on the next
stamp.

**The persona-promotion discriminator FIRED, in the direction the promotion
predicted, and the next `/backlog` should record it as answered.** The previous
stamp set it out explicitly: *"if a session took either row, the consultation
cost stops."* A session took one — the six-issue README/`validate` chain, row 2,
promoted from row 12 on the consultation cost rather than on lane movement — and
landed all six in one window. **Six of the eleven standing persona defects are
discharged**, and the sixth consultation should reconfirm five rather than
eleven. The other promoted row, #870 + #747 + #514 + #687 + #672, is untaken and
is this stamp's row 1; the discriminator survives for it.

**The steward-ranking rule is producing landings rather than rankings.** All
three refactors banded under it on 2026-08-23 have moved: **#979** landed the
next window, **#978** has a branch and a grounding, **#843** is row 3. **#841
remains the counter-example the rule cannot reach** — a `kind/refactor` with a
steward ranking that stays `blocked` because its trigger has no mailbox. That gap
is on #841's thread and is still not filed as an issue, because the fix belongs
to whichever pass gives Part 2 of the `/retro` the mailbox Part 1 has.

**Standing, unchanged, and still true.** Four unlanded corrections target one
paragraph of `docs/WORKFLOW.md`'s filing discipline — **#510**, **#646**,
**#679**, **#912** — and whichever lands last rebases three times; #493 is not a
fifth and #992 is not a sixth, both targeting different paragraphs of the same
file. **The next `/retro` inherits three issues it already owed** — #692, #925
and #841. **The CTA cohort's 45 banked `instance` failures remain
unattributed**, twelfth consecutive stamp carrying it. **`gate.yml` runs but is
still not a required status check**, which only the repository owner can change.

**One environment cost is recorded and deliberately NOT filed.** This session's
arbiter first-witnessed that uncached conformance test binaries hang under the
default sandbox — the `go test` driver spins with no child process and never
returns — so every conformance run had to be issued unsandboxed. It is
reproducible and costs a wasted timeout each time a session meets it.
`docs/ROUTINES.md`'s **Environment requirements** owns it and does not mention
it; the sandbox paragraph already there is about `gh`, a different problem
sharing one word. It is **one witness**, recorded in `docs/LOG/2026-08.md`'s
2026-08-24 entry, and this pass searched all 225 open bodies for `default
sandbox`, `unsandboxed`, `no child process` and sandbox/hang co-occurrence and
found **no second witness**: every hit uses those words for something else. A
second sighting makes it a filing and a `docs/ROUTINES.md` paragraph of its own;
until then it stays in the log.

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
