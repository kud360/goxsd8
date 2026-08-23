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

## Status — 2026-08-23 (full `/backlog` derivation: lanes, branches, markers, milestones and queue all re-read this pass, and the persona consultations folded)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10752 | 15609 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13225 | 2173 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**No lane moved, the figures are RE-DERIVED rather than carried, and the
impossibility is measured in two commands.** The previous stamp carried these
numbers forward without re-reading them and said so; this one ran the tool.
`git diff --name-only a7531f7 HEAD` is six files, **not one of them a `.go`
file**, and reaching further back to the last stamp that actually derived a lane
score, `git diff --name-only af5c6ac HEAD -- '*.go'` returns exactly
`xsderr/doc.go` while `git diff --stat af5c6ac HEAD --
conformance/testdata/expectations/` returns **nothing**. So every figure above
is byte-identical to the previous three stamps' by construction, and the one
`.go` file touched in that whole span is a package comment.

**What this window was: the seventh weekly `/retro`, both parts, and no code at
all.** Part 2 landed first — `e6f8170`, the steward architecture audit (**#976**)
— and Part 1 second, `155ec39`, the chronicler's process retro (**#980**). Six
issues closed, three filed by those two passes, one more filed here. The window
is unusual in a way worth naming: **the retro changed the rules this very pass
runs under**, and two of those changes bind the cartographer directly. Both were
applied below, and one of them broke on first contact with reality.

### The two rules that landed mid-window, and what they did

**`.claude/agents/cartographer.md` gained a banding rule.** *"A `kind/refactor`
carrying a steward cost-of-delay ranking is banded on that ranking … a
divergence the steward measured as increasing outranks a lane slice, and nothing
else will ever lift it, because a refactor moves no lane by construction and
costs no per-session friction to compound."* Its stated motivation was that six
of the seven 2026-08-16 steward findings had gone untouched. **Three issues carry
such a ranking and all three are banded below** — **#843** ("Steeply
increasing"), **#978** ("Steeply rising, and uniquely cheap right now"),
**#979** ("Rising with M6/M7") — the first two above lane slices. This is the
first stamp in which a refactor outranks a conformance slice on anything other
than a session's judgment.

**`docs/WORKFLOW.md` gained a lease rule for empty claims, and its first
application locked the band head.** #946's ruling: a `wip/issue-<N>` with no
commits of its own is dated *"by its newest issue-thread comment instead,
against the same 2h TTL."* Applied literally to the two such branches:

| | `wip/issue-862` | `wip/issue-884` |
|---|---|---|
| newest thread comment | 2026-08-20T20:20:10Z | 2026-08-23T13:46:34Z |
| age at 14:30:57Z | 66h 10m | **44m** |
| verdict | takeable | **LIVE, off-limits** — but read the correction below; #884 no longer reads under this rule |

**#884 was this band's head under that rule**, and the comment holding its lease
was an oracle **GROUNDING** — the preparation that makes an issue more ready,
counted as evidence that it is already claimed. The rule's own justification is
*"a session still working would have checkpointed"*, which is sound in the
direction it is written and does not invert: a grounding, a verdict, an audit
note and a planning correction are all comments from sessions that are not
attempting the issue. **#981** was filed for it, and **`wip/issue-862` is now its
only live subject.** This pass deliberately left no comment on #884, because
under that rule doing so would have re-dated the lease and extended the lockout
by two more hours — the incentive gradient is on #981's thread.

**Correction, mid-pass: #884 is no longer an empty claim, and its lease is no
longer dated by a comment.** While the log entry for this pass was being written,
mason pushed `9c59a2e` — the implementation itself — and merged `main` into the
branch at **`834f5c5`, 2026-08-23T14:30:46Z**. A branch with commits of its own
has a tip time of its own, so `docs/WORKFLOW.md`'s primary rule — *"the name is
the claim; the tip time is the lease"* — governs it, and the empty-claim
exception (which reaches only a branch that has pushed *"no commits of its
own"*) does not. **The verdict is unchanged — LIVE, off-limits — but the basis
is the tip, not the thread**: the TTL lapses at **2026-08-23T16:30:46Z**, not the
15:46:34Z stated above, and any takeover names tip **`834f5c5`**, never
`a7531f7`. #884 is still `open`/`ready` with no PR. Band row 1 and the next
planning action below are stated on the corrected basis; the `wipsurvey` paste
that follows is left exactly as it ran and is stale in this one row.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed the paginated issue JSON, 581 issues — the
survey ran before #981 was filed, so it is one short of the 582 the queue
paragraph below counts; no `wip/*` ref belongs to #981):

```
ISSUE  BRANCH         TIP AGE  VERDICT  REASON
822    wip/issue-822  unknown  RETIRED  wip/issue-822: issue #822 is closed
862    wip/issue-862  main's   CLAIMED  wip/issue-862: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  unknown  RETIRED  wip/issue-872: issue #872 is closed
884    wip/issue-884  main's   CLAIMED  wip/issue-884: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
933    wip/issue-933  69h7m0s  RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968  8h12m0s  RETIRED  wip/issue-968: issue #968 is closed
```

`git ls-remote --heads origin` returns exactly `main` and those six `wip/*` refs,
re-read this pass. Nothing is EXPIRED, no `parked/*` ref exists, and no verdict
changed from the previous stamp.

**The survey cannot apply the rule that decided its two CLAIMED rows when it
ran.** Both still print *"settle it from the issue thread"* — correct output
before #946 ruled, and now a tool declining the only rule that answers the
question. (`wip/issue-884`'s row is stale a second way: that branch has had a tip
of its own since 14:30:46Z, so it is no longer a CLAIMED-by-comment row at all —
see the correction above. The paste stands as it ran.) It
reads `{number, state, labels}` from stdin and never sees a comment timestamp, so
this is a change to `docs/ROUTINES.md`'s calling convention as much as to the
tool. That is **#981**'s Acceptance item 2; the two lease verdicts above were
settled by hand, as every stamp since 2026-08-18 has settled #862's.

**`wip/issue-822` and `wip/issue-872` show commits absent from `main`, and that
is a squash-merge artefact, not an alarm.** `git log origin/main..origin/wip/issue-822`
lists four and `…872` five, because PRs land squashed and a `wip` branch's own
commits are never `main`'s ancestors. Both issues are closed **`not_planned`
carrying `needs-replan`** — parked, not landed — and both supersessions have
since landed: **#851** and **#878**, closed `completed`. So the content question
those rows raise is answered and stays answered. `wip/issue-933` (superseded by
#862's duplicate ruling) and `wip/issue-968` (superseded by #972) are the same
shape. **All six deletions remain a human's call.**

**`go tool gapaudit`, verbatim** (fed `kind/gap`-filtered, `state=all` JSON — 138
issues, 50 of them open):

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

**Both groups are byte-identical to the previous stamp, and that is expected
rather than reassuring**: no `.go` file changed, so the 64-marker census could
not move, and the three issues filed this window (#978, #979, #981) are all
`kind/refactor`/`kind/process`, so none entered the `kind/gap` input. **#972 is
still absent from Group 2 where it belongs** — the file-path matcher artefact the
previous stamp reproduced, owned by **#852**, unchanged because nothing has
landed against it. **Group 1's emptiness stays qualified, not repeated**: a file
path cited in a body for any reason at all — including to say "do not touch this
site" — matches every marker in that file, so an empty Group 1 is not proof that
every marker has an owner. **#960**, unblocked to `ready` by this window's retro,
still owns the class the census structurally cannot see: a fail-open disclosed in
PROSE carries no `GAP(` marker and appears in neither group.

### Milestones and queue

Milestones, read from `repos/kud360/goxsd8/milestones` this pass and
cross-checked against the paginated issue list, which agrees exactly on every
row once pull requests are dropped.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 97 | 46 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**One milestone row moved: M4 open 45 → 46, and it is #975 alone** — filed by the
previous stamp's own pass and the only issue filed in this window with a
milestone at all. #978, #979 and #981 carry none, following the convention that
the milestones hold feature slices while process, tooling and cross-cutting
refactor work sits outside them. **169 of the 228 open issues carry no
milestone** (228 − 46 − 13), so the rows above are feature progress and the
paragraph below is the queue.

Queue: **228 open — 208 `ready`, 20 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 20), against **354 closed**.
208 + 20 = 228 exactly, and **every one of the 228 carries a queue label** — the
class #773/#774 fell into is empty for the twelfth consecutive stamp. Both
figures were re-derived by paginating the issue list page-numbered rather than
with `--paginate`, whose Link header uses numeric-ID URLs the proxy blocks,
raising the page count until a page came back empty (page 11) and discarding
pull requests, which share the endpoint.

**Every move reconciles, and the previous stamp's own carried figures are the
baseline being reconciled against.** Closed 348 → 354 is six: **#966** (which
closed 20 minutes *before* the previous stamp was written and which that stamp
flagged as already having moved its counts), then **#681**, **#796**, **#881**,
**#946** and **#400**, all closed `completed` by the retro at 14:01Z. Open 230 →
228 is those six against four filings — **#975**, **#978**, **#979**, **#981**.
`ready` 202 → 208 is −2 closed (#966 and #400 both carried `ready`, not
`blocked`), +4 filed, +4 unblocked; `blocked` 28 → 20 is −4 closed (#681, #796,
#881, #946) and −4 relabelled. **The four unblocked are #548, #622, #696 and
#960**, all moved `blocked` → `ready` by the retro that ruled on them.
`needs-replan` was 0 and stays 0.

**The unblock sweep relabelled nothing, and for once the interesting cases are
the ones where a TRIGGER fired.** All 20 `blocked` bodies were fetched over
`gh api` — byte-faithful, where MCP `issue_read` is lossy (#764) — and their
`## Depends on` sections read in full. Fifteen name an open issue that is still
open (#455, #591, #414 ×2, #407, #250 ×3, #472, #831, #719, #248, #732, #591)
and are untouched. The remaining five name a trigger, and **three of those
triggers fired this window**:

- **#692** and **#925** name *"the next `/retro` process audit"*. It ran. **It
  did not rule on them**, and the retro session said so on both threads: they
  were outside the `kind/process`/`kind/tooling`-scoped evidence list handed to
  the chronicler, which is the same *"dependency target with no mailbox"* the
  retro's own structural finding names. Both stay `blocked` per their bodies'
  explicit instruction, flagged for the next retro. The retro's fix —
  `.claude/agents/chronicler.md` now has the retro read the `blocked` queue
  itself — should catch them next time.
- **#841** names *"the next `/retro` steward drift review"*. **It also ran**
  (`e6f8170`), also did not take #841 up, and unlike the two above **left no
  comment**, so the firing would have been invisible. This pass left one. The
  retro's mailbox fix landed in `chronicler.md` and reaches Part 1 only;
  **nothing equivalent landed for Part 2**, so an issue blocked on the steward
  review still has no mailbox. Concrete cost, stated on the thread: #841 is a
  `kind/refactor` that would be banded on this stamp's new steward-ranking rule
  exactly as its sibling #843 is, and cannot be, because it is `blocked`.
- **#79**, **#250** and **#555** name triggers that did not fire (M4's feature
  tail reaching zero; the M5 epic; a discharge half B). Untouched, and their
  bodies ask not to be re-scanned.

**No duplicate was closed and one issue was filed.** #981's filing search ran
over all 227 open bodies for `wipsurvey`, `2h`, `heartbeat`, `takeover`, `empty
claim` and `no commits of its own`; the only `wipsurvey` hits are **#805** and
**#809**, both in `gitAncestry`/`ancestryFromExit` — the shallow-clone ancestry
path that decides RETIRED versus EXPIRED for branches that *do* carry commits,
disjoint from the empty-claim path. #722, the closed predecessor that made an
empty claim undatable in the first place, is named in #981's Notes.

### Persona consultations — folded, and the first pass in five to change a verdict

**Both personas were run by the orchestrating session and handed here** (#416's
option (a): the cartographer has read the source, so a persona it role-played
would launder an insider's opinion as an outsider's). This is the **fifth
consecutive** consultation, and the previous four all read *"reconfirmed, filed
nothing"* on both rows.

**This one filed nothing either — and that is now a finding rather than a
formality.** Every one of the eleven standing defects reproduced, at `155ec39`,
by running things rather than reading them; **twelve issue bodies were refreshed
with today's evidence** rather than left carrying citations from `f1250c0` and
`9c10af8`; three previously-unmeasured behaviours were added; and **three new
friction points were dispositioned without a new issue, each in a body where it
will be read**. The consultation cost is real and repeating: ten session-halves
across five passes, re-measuring defects that are one doc-editing session each.
That is the argument that moves both persona rows up the band below.

What changed in the twelve:

- **#748** — the underlying work is **RESOLVED and the README is now stale in the
  OPPOSITE direction.** It no longer under-describes an unbuilt API; it tells a
  reader not to try a shipped one. The libuser ignored the warning, built the
  real path, and then hit the signature mismatch as a **second, undocumented**
  compile error. Its `instance`-lane figure was **1337** and is **10752** — the
  API README calls nonexistent has banked eight thousand cases since the body
  last said so.
- **#669** — defect 1 is **two independent compile errors, not one**, isolated
  this pass in a throwaway module: `undefined: logger` (the snippet binds it
  nowhere) and `declared and not used` (the results are unread). Only the second
  was on record, and **it is the one that does not matter** — it vanishes the
  moment a reader uses `schema`, which every real program does. A fix that only
  adds error handling leaves the block still not building.
- **#896** — reproduced end to end: `<price>12,50</price>` against `xs:decimal`
  yields `len(res.Violations()) == 1` and **`res.Err() == nil`**, so the
  idiomatic `if res.Err() != nil { reject }` accepts an invalid document. Also
  measured: plain `go doc ./validate` renders `Result` as one collapsed line with
  **no method list**, so the accessor is not merely unexplained on the default
  page, it is invisible.
- **#934** — confirmed by a **run** where the evidence was previously a reading:
  the emitted top-level bracket is `[cvc-type]`, with `[cvc-datatype-valid]` only
  as the wrapped cause. Charge site re-read at `validate/cvccomplexcontent.go:507`.
- **#687** — a **fourth** trigger of the same raw-token scan: `goxsd8 -- -help`
  prints full help and exits 0, so `--` is not honoured as end-of-options.
- **#672** — `-v` is **not available** as a version spelling; `usage` and README
  both assign it to debug logging. That constrains the decision the issue exists
  to make.
- **#625**, **#492**, **#409**, **#870**, **#747**, **#514** — all reconfirmed
  with re-derived line citations and at least one newly mechanical check apiece
  (`go test ./xsd -run Example_buildFinalizeQuery` → `ok`; four `grep -c` → `0`;
  `go doc -all ./codegen | grep -c '^func \|^type '` → `0`; a byte-identical
  `diff` across four CLI invocations).

**The three new friction points, and why none became an issue:**

- **`Result.Err()`'s inverted-from-Go-idiom semantics** — this is **#896**,
  whose Goal is that exact sentence. Folded into its Acceptance. The persona's
  broader remark (three fallibility signals is a lot of surface) is on #896's
  thread and explicitly fenced there as **#56**'s territory, not a licence to
  widen #896.
- **`go test ./...` failing without the submodule step** — **intended landed
  behaviour**, not a defect. **#309** (closed `completed`) deliberately made an
  absent suite a run-ending condition for `TestConformance` so that *"a suite
  that executed zero cases must not be indistinguishable from a green run"*; the
  fixture-driven tests skip cleanly and `GOXSD_SUITE_OPTIONAL` exists. The
  submodule line already precedes the build line inside the same fence.
  Dismissed on **#870**'s body, in the section that owns that fence, with an
  explicit "do not soften #309 under this issue".
- **`goxsd8 -q` indistinguishable from a typo** — the same root cause as
  **#514** (`args` never inspected) and its **third** distinct meaning, a
  *documented* flag rather than a subcommand. Recorded on #514 as a table row and
  as a new case in its Acceptance test matrix.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan the whole
of it. Take from the top. Cross-references are stated by ISSUE rather than by row
number, which decays at each re-cut. **This is a full re-cut, not a shift**: four
issues enter, three leave to the unbanded list, and the two persona rows are
promoted on the consultation cost measured above.

| # | Issue | Why here |
|---:|---|---|
| 1 | #884 | **Still the head, and now the best-prepared issue in the queue — but its lease is LIVE until 2026-08-23T16:30:46Z, two hours after its OWN tip `834f5c5` (2026-08-23T14:30:46Z).** Corrected mid-pass: this row was cut when `wip/issue-884` had no commits of its own and read under the empty-claim exception, lapsing 15:46:34Z; mason then pushed `9c59a2e` and merged `main`, so the primary tip-time rule governs and both the clock and the tip below changed. An oracle GROUNDING landed on the thread at 13:46Z answering all four of its spec questions: `xs:namedGroup` is `xmlschema11-1.md:5187-5216`, the `<choice>` is pinned `minOccurs="1" maxOccurs="1"` at `:5192`, neither a nested `<group ref>` nor a bare `<element>` is among its three alternants, and `mgd-props-correct` (`:2302`) is a post-construction component-tableau check that presupposes the very `{model group}` slot the malformed body failed to fill. So the issue's whole thesis is confirmed and its Spec section is settled twice over — by #966's ruling and now by the grammar itself. **After the lease lapses, take it with a takeover comment naming the branch tip (`834f5c5`) and RESUME that branch — `9c59a2e` is an implementation already pushed against this issue, so re-deriving it from `main` would redo work that already exists; before the lapse, do not touch it.** `xsderr/doc.go` gives the message shape, `rejectProhibitedAttrs`/`checkS4SChildOrder` the exemplars |
| 2 | #732 | **The band's only false-reject, which CLAUDE.md's conformance stance treats as never acceptable**, and it gates #972. §4.2.2 `vc:minVersion`/`vc:maxVersion` conditional inclusion is unimplemented, falsely rejecting `VC/vc007` and `VC/vc_003_1`, both reproduced in its body by running `parser.Parse`; four more (`VC/vc003`–`vc006`) are banked `pass` today only through a double negative that expires the moment either half is fixed alone. **Unsized and it says so** — its first step is an oracle grounding, not code, and a session returning only a grounding comment has done the right amount. Unchanged from the previous stamp |
| 3 | #669 → #625 → #748 → #492 → #934 (+ #896) | **PROMOTED from row 12, on the consultation cost rather than on lane movement.** One session closes six issues and ends a tax that has now run five consecutive passes. The argument that held it down — *"splitting it is why it sat"* — is spent: they are one ordered row, every body carries evidence re-derived at `155ec39` today, and #748's is the only one whose *shape* changed (resolved work, README stale the other way). Take them in the order named; **#896 is the one non-README member** (`validate/doc.go`) and should land beside #748, which sends readers to it |
| 4 | #870 + #747 + #514 + #687 + #672 | **PROMOTED from row 13, same argument, and #720's body independently asks for three of them first**: *"Take #514, #672 and #687 before this lands. Each is a sentence or a dispatch branch while the CLI surface is still empty; taken afterwards every one of them is a change to shipped behaviour."* #472 is `ready` and is what makes that true. Fifth consecutive reconfirmation; **#687 now carries four triggers of one raw-token scan and #672 a spelling collision that narrows its own decision**. The CLI is still unreachable from its own documentation |
| 5 | #978 | **NEW to the band, and the first issue ever banded on `.claude/agents/cartographer.md`'s steward-ranking rule** — which landed in `155ec39` about half an hour before this pass ran. `xpath` charges XPath/F&O error codes as message SUBSTRINGS while `regex` tags the same class as an `xsderr.Rule`. Ranked above #843 for one reason: **#894 is open and `ready`**, adds `err:XPST0051`/`err:XPST0080` to this exact function, and in #978's own words *"doubles this issue's diff"* if it lands first. Today the change is one error site, one constant, one wrap. The ordering #978-before-#894 is a cost ordering, not a dependency; neither `## Depends on` names the other and neither should |
| 6 | #843 | **NEW to the band, the rule's own named exemplar, and the steepest ranking of the three.** Four hand-maintained copies of one complex-type descent have **already diverged** on the redefine-original containment edge — two of the four do not descend `c.Base()` at all. Its ranking re-read this pass: M4's open count is **46**, up from the 42 the 2026-08-16 audit recorded, so each new finalize-time constraint is a fifth copy picking its edges by eye. The bugs are **fail-open**, invisible to the ratchet, which is why an audit and not the suite found this one. **Sizing is the open question, not value** — a session returning only a design comment naming the parameterization has done the right amount |
| 7 | #981 | **NEW, filed by this pass, and it charged its friction to this pass.** `docs/WORKFLOW.md`'s empty-claim lease is dated by ANY thread comment, so an oracle grounding posted 44 minutes earlier locked the band head for two hours — and `wipsurvey` cannot apply the rule at all, printing *"settle it from the issue thread"* for both CLAIMED rows. Banded on CLAUDE.md's cost rule: one session, doc-first, and the tool arm may split off. **A second sighting promotes it above the refactors** |
| 8 | #963 | **The tax falls on every landing and its discriminator STILL could not fire.** #820 landed the *form* of landing precondition 1; nothing checks that the check was run, which is **#924**'s shape and is not reachable by prose. One session — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part (#304 made CLAUDE.md's Commands block the sole gate definition). **No code landed this window**, so the only landings are two doc passes that carry their entries by definition; the retro also narrowed precondition 1's domain to exclude a pass that closes no issue, which is a change #963's fixtures must respect |
| 9 | #493 | **On CLAUDE.md's cost rule; the friction was paid by hand two windows ago and the payment is still unlanded.** `docs/WORKFLOW.md`'s park step names neither the close reason nor the `ready` clear, so #968's session had to discover that a parked issue carrying `ready` is pickable. Doc-only, one session, and it targets the **park** paragraph — not the filing-discipline paragraph #510, #646, #679 and #912 all target — so it rebases against none of them. **Live corroboration this pass**: `wipsurvey`'s RETIRED rows include #933, closed `not_planned` and still carrying `ready`, exactly the state #493 exists to prevent |
| 10 | #846 | **#909 paid the shadow tax and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`.** Its promotion rule asks for a third landing that pays the tax and **no landing occurred this window, so the discriminator did not fire** — a different statement from "did not fire on evidence". Its structural argument from #968's family A stands unchanged: the fault deciding those four documents lives outside `<restriction>`'s child list, so no predicate over that list can tell a fabricated verdict from a correct one. Still a ~700-line refactor with no evidence it fits one session |
| 11 | #941 | **#387's own unfiled debt, and the coldest files of any band row** (`07117dc`, 2026-08-21). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete, `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 12 | #953 | **A doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says "every caller turns a decided negative into a schema rejection"; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules |
| 13 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 14 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 15 | #979 | **NEW to the band, third of the steward-ranked refactors and deliberately last of them.** `ctaScanNCName` and `icScanNCName` are byte-identical and both approximate a character class `regex` already owns generated. Ranked below #978 and #843 **on its own words**: its cost is *"Rising with M6/M7"*, and `lanestatus` reads `xpath` as a lane with no cases at all — the clock has not started. That also makes it the cheapest, *"a mechanical move of 13 lines"*, and the best pick for a session wanting a small complete landing. Safe to wait because only one copy can drift: `validate/icpath.go` is frozen at the §3.11.6.2/§3.11.6.3 subset by an explicit ruling |
| 16 | #975 | **Placed here, discharging the previous stamp's "the next `/backlog` places it".** Nine s4s-grammar rejection messages name no Appendix A production — the bar `xsderr/doc.go` states as of #966 — censused at `ebf6fa2` with the re-deriving criterion stated ahead of the list. Cheap, mechanical, moves no lane, and **one of its nine sites (`produceModelGroupDefinition`) sits beside #884's in the same file**, so #884 first and #975 whenever a cheap session wants it. **#444 already owns the tenth deficient site** (`rejectBothInlineTypes`) on a stronger footing and is excluded rather than duplicated |

**Left the band this pass, and why.** **#786** — its deletion branch now waits on
#972, which is `blocked` on #732, so the strongest argument it has acquired is
also the one it cannot act on yet; read it beside #972's Notes when #732 lands.
**#907** — its `childElement` census is stale by at least four landings (#909
rewrote 418 lines of `produce_complex.go`, #957 moved `produce_typetable.go`,
#956 added `produce_s4sorder.go`), so the body's figures must be re-derived
before anything is designed from them, and that re-derivation is the real first
step. **#885** — three discriminators with one sighting each, and none fired this
window; it stays exactly where its own body says it should sit until a fourth.
All three are still `ready` and none is deprioritized on merit.

**Deliberately unbanded, and why.** **#972** is `blocked` on #732 and belongs
nowhere in a `ready` band; when #732 lands it enters at the rank #968 held.
**#561** is `ready`, one test and one sentence, with a live sequencing relation
to #972 rather than a standing one — a session taking it should read #972's
Notes first. **#409** is `ready` since 2026-08-02 with a **fourth** independent
sighting this pass (`go doc -all ./codegen | grep -c '^func \|^type '` → 0 while
the package doc prints `Generate` and `Target` in a code block); it stays
unbanded only because it is one row of a five-file convention landing, not
because the evidence is thin. **#894** is `ready` and should be taken **after**
#978 — see #978's thread; it is one of the three `area/xpath` gaps with **#888**
and **#889** still awaiting a suite census in their range. **#937** is naturally
folded by the next landing touching `rejectRepeatedAnnotations`. **#920** and
**#921** are conformance-bookkeeping follow-ups below the fold. **#929** and
**#931** are the small parser occurrence / rule-mapping gaps #901 exposed; read
each beside #901's thread. **#455** is the live owner of the
`strings.TrimSpace`-versus-§4.3.6 character class at **ten** sites — a pure
false-accept narrowing with a provably flat ratchet — and **#456** stays
`blocked` on it. **#862** is `ready`, its grounding is banked, and its lease is
now **takeable** at 66 hours: #946 ruled, the rule landed, and the five stamps
of "off-limits until #946 rules" are discharged. Taking it means a takeover
comment naming tip `c2ba631` first. **#843–#849** are the 2026-08-16 audit's
findings, **six open**, of which #843 is now banded and **#841** is `blocked` on a
trigger that fired without a ruling. **#566** is #565's open sibling, routed
nowhere by #565's landing and correctly so. **#871** stays `blocked` on #831.
**#852** owns both directions of the `gapaudit` matcher defect and stays below
the fold because the tool again ran with reconciliation. **#548**, **#622**,
**#696** and **#960** were **unblocked to `ready`** by this window's retro and
enter the general queue unbanded; **#963**, **#857**, **#852**, **#839**,
**#802**, **#779**, **#735** and **#540** were routed back by the same retro as
needing Go changes or live probes a retro cannot honestly perform, of which only
#963 is banded. **#692**, **#925** and **#841** are the three whose triggers
fired unruled. **#570** carries the standing `schema` decline-count argument at
893 against a re-measured 788 — third consecutive reading of that figure, and the
823 in #968's account is a never-committed branch's narrowed count, not a
movement.

### Next planning action

**Take from the top, but read the clock first: #884 is the head and is
off-limits until 2026-08-23T16:30:46Z** — two hours after its own tip
`834f5c5`, under `docs/WORKFLOW.md`'s primary *"the name is the claim; the tip
time is the lease"*. A `/develop` session starting after that takes it with a
takeover comment naming tip **`834f5c5`** and **resumes that branch**, because
`9c59a2e` is a pushed implementation and re-deriving it from `main` would redo
work that already exists; one starting before it takes **#732** or drops to row
3. **Corrected
mid-pass**: when this section was cut the head was held instead by #946's
empty-claim lease, dated by an oracle grounding and lapsing 15:46:34Z, and that
basis died the moment the branch acquired a tip. So the first lease ever to hold
this band head turned out to be an ordinary one. **#981** still owns the rule
defect — its only live subject is now `wip/issue-862` — and a session that hits
the same wall should add its sighting there rather than re-diagnose it.

**#884 is also the most implementation-ready issue this project has had banded.**
Its oracle grounding landed this window and confirmed all four premises against
`xmlschema11-1.md` by line number; #966's ruling settled its footing; all four
malformed-body verdicts are reproduced in its body against a named tip. The
session implements a stated answer, from a read spec, with the message shape
given in `xsderr/doc.go` — and as of `834f5c5` that implementation is already
begun on the branch, so the taking session resumes it rather than re-derives it.

**The two persona rows are promoted, and this is the pass that should be judged
on whether that was right.** Five consecutive consultations have reconfirmed
eleven defects and filed nothing; twelve bodies now carry evidence re-derived
today rather than citations from `f1250c0` and `9c10af8`; and two of the eleven
changed shape this pass (#748's direction, #669's split). The next `/backlog`
has a clean discriminator available to it: **if a session took either row, the
consultation cost stops; if neither was taken and a sixth consultation
reconfirms them again, the promotion was not the binding constraint and the band
is not where the problem lives.** Say which on the next stamp.

**Three refactors are banded on a steward ranking for the first time**, under a
rule 30 minutes old at the time of banding. Its motivation was that six of seven
2026-08-16 steward findings had gone untouched; #843 is now banded, #978 and
#979 are banded the day they were filed, and **#841 is the counter-example the
rule cannot reach** — a `kind/refactor` with a steward ranking that stays
`blocked` because its trigger has no mailbox. That gap is on #841's thread and is
not filed as an issue, because the fix belongs to whichever pass gives Part 2 of
the retro the mailbox Part 1 just got.

**Both standing promotion discriminators were checked and NEITHER COULD FIRE,
for the second consecutive window.** #963's asks whether a landing carried its
`docs/LOG` entry inside the squash; #846's asks whether a landing's entry
recorded the shadow tax. **No code landed** — the whole window is two doc passes
— so both questions had no subject. The next pass runs them against the next
real landing.

**Four unlanded corrections still target one paragraph of `docs/WORKFLOW.md`'s
filing discipline** — **#510**, **#646**, **#679**, **#912** — and whichever
lands last rebases three times. **#493** is not a fifth: it targets the park
paragraph. **The next `/retro` inherits three issues it already owed** — #692,
#925 and #841, all three flagged on their threads this window — down from eleven,
which is the largest single reduction in that backlog the project has recorded.
**The CTA cohort's 45 banked `instance` failures remain unattributed**, eleventh
consecutive stamp carrying it. **`gate.yml` runs but is still not a required
status check**, which only the repository owner can change. All of these stay
open and stay true.

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
