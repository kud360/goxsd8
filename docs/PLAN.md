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

## Status — 2026-09-09 (`/backlog`. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a fresh `git fetch -p origin` plus a `git ls-remote --heads`, the marker census a fresh `gapaudit`, and the milestone and queue counts a fresh page-numbered `state=all` fetch taken after this pass's own writes. **The window is FOURTEEN landings closing NINETEEN issues in two days** — band rows **1 through 10 and 12** in order, plus the two issues those rows' own landings carved or unblocked (**#1349**, **#456**); only row 11 (**#1167**) is left. **`schema` moved 13942 → 13968**, the first non-flat window in three, and **both movers banked from a population their own body had not identified**: #456's named case did NOT flip and twenty unnamed ones did, while #931 — banded LOWEST of the lane slices precisely because its population was a hypothesis — skipped the census its band row instructed in as many words and banked +6 where its body named 4. The namespace carries **ONE LIVE claim** (`wip/issue-413`) and **one EXPIRED, resumable** (`wip/issue-1167`). The marker census holds at **68** with **ZERO untracked sites for a TENTH consecutive stamp**. The **NINETEENTH persona consultation**: **six findings — five filed (#1367, #1368, #1369, #1370, #1371), one deduped into the issue that already owns it (#1286)**)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 13968 | 1430 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### The band ran to exhaustion, `schema` moved +26, and neither mover banked the population its body named

**Eleven of the last band's twelve rows landed, in order.** Rows 1–10 and 12;
row 11 (**#1167**) is the only survivor and is banded again below. Two more
issues landed with them, each carved or unblocked by a row's own landing:
**#456** (row 3's dependant, `blocked` on it) and **#1349** (filed by row 9's
landing).

| landing | commit | predicted | banked |
|---|---|---|---|
| **#1007** +#1231 +#1123 +#1189 +#1261 +#1290 | `84bbed7` | unchanged, structural | unchanged |
| **#1304** | `6d3b94e` | unchanged, measured AND structural | unchanged |
| **#1300** | `34aee07` | unchanged | unchanged |
| **#455** | `8aca2a3` | unchanged, structurally excluded then measured twice | unchanged |
| **#931** | `c5e5e37` | **`schema` +4**, from an eighteen-day-old hand probe | **`schema` +6** |
| **#725** | `6b3759e` | *"UP only"*, by construct census before implementing | unchanged (vacuously satisfied) |
| **#1312** | `84da227` | unchanged, structural and measured | unchanged |
| **#1247** | `77b0419` | unchanged, doc-only, run in write mode | unchanged |
| **#1251** | `3e9b8eb` | unchanged, structural and measured | unchanged |
| **#1203** | `f4ead6c` | unchanged, run in write mode | unchanged |
| **#1313** | `665437a` | unchanged | unchanged |
| **#1227** | `e36927d` | unchanged | unchanged |
| **#1349** | `c5e7a57` | unchanged, predicted by the mechanism | unchanged |
| **#456** | `5b84206` | **movement CONFIRMED on a named case** | **`schema` +20** |

**The finding is that BOTH movers banked a population their body had not
identified, in opposite directions of the same error.**

- **#456 named one suite case by ID and that case did not flip.** Its Acceptance
  item 6 required `Open/open013/instance/open013.v1.xml` to flip; the defect it
  named IS fixed (`mixed=" 1 "` reads its ·actual value·), but a correct `mixed`
  makes the ·effective content· a particle (§3.4.2.3.3 clause 2.1), routing to
  clause 5.2.1's default open content, which `xsd.Schema.ContentMatcher`
  declines under a `GAP(xsd)` owned by **#717**. The case left #1160's
  decided-and-wrong class for the decline-candidate class at the same banked
  score. The **+20** came from twenty other fixtures — every suite-declared
  `invalid` case writing an `xs:boolean` schema attribute outside `booleanRep` —
  which the member censused directly, by attribute VALUE, because `suiteindex`
  censuses by construct and cannot take that census. **Naming the instrument's
  limit is the part that worked.**
- **#931 was banded LOWEST of the lane slices for exactly the right reason and
  then skipped the step that banding asked of it.** Its band row read *"Census
  by construct FIRST, with #1282's `Parent` field, and re-band on what it says"*;
  the census was never run, an eighteen-day-old hand probe went to the verdict
  intact, and the lane moved **+6** where the body named 4, with 5 of the 6
  flips unclassified by the body. **#1332** owns that defect and is band row 2.

**So the last stamp's criterion — prefer a NAMED failing case or a
construction-guaranteed direction over a hypothesis — selected the right issue
and was right for the wrong reason.** It is kept, and sharpened by what this
window measured: **a named single case is weaker evidence than an ENUMERATED
candidate set**, because a single case can be blocked two layers downstream by a
gap nobody tested against, while an enumeration cannot. Row 1 below is the first
member of this family whose body carries both — ten shapes driven through
`parser.Produce` and a complete set of **24 named suite fixtures**.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against a fresh `git fetch -p origin` and
this pass's 760-issue pre-write feed:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
413    wip/issue-413   3m0s       LIVE     wip/issue-413: tip pushed 3m0s ago, within the 2h0m0s claim TTL
732    wip/issue-732   393h48m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   571h46m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   344h7m0s   RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   537h47m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   466h0m0s   RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   405h5m0s   RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   349h28m0s  RETIRED  wip/issue-993: issue #993 is closed
1167   wip/issue-1167  2h54m0s    EXPIRED  wip/issue-1167: tip pushed 2h54m0s ago, past the 2h0m0s claim TTL
```

**`wip/issue-413` is LIVE and mid-repair-round**, tip `d2a14da`
(*"xsd: repair the {open content} deferral marker's spec claims (#413)"*) pushed
minutes before this survey, with an arbiter verdict and a `MASON: repair round`
comment on the thread. **#413 is off-limits and is not banded**, and so is
anything else editing `xsd/`'s open-content deferral markers — **#717** in
particular, which the #456 analysis above puts one step from that code.

**`wip/issue-1167` is EXPIRED and therefore RESUMABLE, not abandoned.** Its
thread carries a `TAKEOVER:` comment timed 9 seconds after the branch tip
(2026-09-09T00:14), so the claim was dated once and then lapsed at the 2h TTL —
a session that died, not a branch nobody ever held. **Taking it means posting a
second `TAKEOVER:` naming the tip you found, then pushing the heartbeat.** Read
the thread's earlier `PROCESS NOTE` first: the branch's pre-takeover tip was a
stray post-land commit for a DIFFERENT issue whose content is already on `main`,
so none of that diff is #1167's history. **Not relabelled `needs-replan`** —
under three hours is not "stale for days", and the label retires an issue rather
than releasing a lease.

**`wip/issue-1349` VANISHED at merge** — `git fetch -p` reported it deleted.
That is GitHub's auto-delete working as the workflow says it does, and it is the
first deletion observed in several stamps; the seven RETIRED refs below are
parks and supersedes, which are supposed to persist.

**The same seven RETIRED refs, unchanged row for row — SIX stamps now.** All
seven closed `not_planned`; none owes a supersede. Cloud containers cannot
delete remote refs, so these accumulate by design and are not a finding.
**Zero `parked/*`. Zero `meta/*`.**

**FIVE non-`wip` `claude/*` refs stand, one of them new** — `eloquent-cerf-39rk64`
(`0abeab6`), `-3xu0ki` (`62d5143`), `-8jq9o6` (`7841e98`), `-adewly` (`5d03049`)
unchanged in tip, and **`-kk1f7v` (`1ba9b40`), which is `origin/main` exactly**:
the session branch the last post-land pass landed through, left behind after its
PR merged. Listed for human triage, not acted on. `wipsurvey` reads `wip/*` and
`parked/*` only, so these and any `meta/*` are found by `git ls-remote --heads`
and never by the survey.

### Marker census

`go tool gapaudit` over this pass's whole 760-issue pre-write feed: **68 markers
across 8 areas** — `xsd` 32, `validate` 17, `xpath` 6, `xml` 4, `parser` 3,
`value` 3, `conformance` 2, `cmd` 1.

**The census total held at 68 across fourteen landings, and the per-area split
is identical to the last stamp cell for cell.** Two of those landings touched
markers in both directions — #725 deleted its `GAP(xsd)` at `redefinition.go`,
#413's live branch is repairing another — so **the total holding is not the same
claim as the SET holding**, and this pass did not diff marker by marker against
the last stamp, which recorded counts only. Whoever next needs the composition
must take it from `gapaudit` directly.

**Group 1 moved 17 → 16 and group 2 26 → 23. ZERO group-1 rows carry no
annotation at all, so the tool's own filing rule selects nothing and there are
ZERO untracked GAP sites — TENTH consecutive stamp.** No `kind/gap` issue was
filed from the census this pass, and that is the census reporting a clean tree
rather than a pass skipping the step.

**One gap this pass found is invisible to this instrument, and that is worth
recording beside the clean census.** #1369 (below) is an unimplemented s4s check
with no `GAP(parser)` marker anywhere, so `gapaudit` cannot see it and neither
can a reader of the tree — it was found from outside, by a persona compiling a
schema with a one-letter typo in it. A clean census means every MARKED site has
an owner; it has never meant the tree has no unmarked ones.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 14 pages (13 full at 100, page 14 at 71), PRs filtered locally:
**1371 rows, 606 PRs excluded, 765 issues — 283 open, 482 closed.**

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **51** | **124** | active |
| **M5 — Instance validation (XML)** | **14** | **24** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **267 `ready`, 14 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 283 with no gap, and **every open issue carries exactly
one, verified over all 283 by grouping each issue's own label set rather than by
summing two counts**. That distinction is the repair from two stamps ago
holding: a check that adds `ready` and `blocked` and balances against the open
total cannot see an `epic` hiding inside `blocked`. **#779** owns the mechanical
check and carries this as its fifth class.

By kind: `kind/refactor` 80, `kind/process` 56, `kind/gap` 49, `kind/tooling`
39, `kind/bug` 29, `kind/story` 24, `kind/docs` 21, `kind/feature` 4. By area:
`parser` 79, `meta` 78, `xsd` 58, `conformance` 30, `docs` 30, `validate` 24,
`cmd` 19, `value` 14, `builtin` 11, `xpath` 6, `xsderr` 6, `model` 4, `loader`
2, `regex` 2, `cli` 1.

**M4 moved 117 → 124 closed and 50 → 51 open**: seven closed against it, and the
open count still rose because this window's post-land passes and this pass filed
more onto it than the landings retired — the shape #347 says to expect, `ready`
being an output. **M5 moved 23 → 24 closed and is unchanged at 14 open**, third
consecutive stamp at 14: every landing this window was `parser`, `cmd`, `xsd`,
`tools/` or `meta`. Neither milestone's count is its lane's remaining work; the
M4 and M5 scope sections below say so in their own words.

**`ready` 267 is the honest startable count with ONE to subtract**: #413 carries
a LIVE claim, so **266** are startable without colliding. **#1167's EXPIRED
claim is NOT subtracted** — an expired lease is takeable, which is the whole
point of the TTL.

**The unblock sweep measured a clean zero for the ELEVENTH consecutive stamp.**
All **14** open `blocked` bodies were read and their `## Depends on` sections
checked against every issue this window closed — the nineteen the landings
closed, plus **#726** (closed moot by #725's post-land pass) and **#531**
(closed obsolete by #1300's). **Not one names any of them**,
and every named issue dependency in the partition — #250, #407, #831, #591,
#248, #1324 — is open. **Zero relabelled.** The set stayed at 14 with one
exchange: **#456** left it by landing, **#1344** joined it. Re-read the
partition below rather than re-deriving it:

- **FOUR are triggers rather than issues** and say so in their own
  `## Depends on` — **#1224**, **#555**, **#1002** and **#1042**. **Do not
  re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **EIGHT still have at least one OPEN named dependency** — **#248**, **#267**
  and **#345** (all on #250), **#415** (#407), **#593** (#591), **#717**
  (#248), **#871** (#831) and **#1344** (#1324). **#1344's is band row 8**,
  which is the one place in this partition the band can reach — and it
  discharges in EITHER direction, #1324 closing `not_planned` being a recorded
  ruling just as much as #1324 landing is.

### Persona consultations — the NINETEENTH ran, and the named trigger fired again

The cartographer role-plays no persona and does not spawn one (#416): it has
read the source, so a verdict it produced would launder an insider's opinion as
an outsider's. **The orchestrating session ran both personas against the
published surface — README, `go doc`, and a CLI built from `faf2017` — and
handed this pass their reports; what follows is the folding, not the
consultation.**

**The trigger the last stamp named fired exactly as named, for the FIFTH window
running.** It was *"whatever next changes the observable surface"*, with band
rows 3, 5, 6, 7 and 10 listed as the ones that would. **Row 10 — #1313 — landed
and fired it**, and cliuser went straight back at the sentence that landing had
just widened.

**Six findings: FIVE filed, ONE deduped into the issue that already owns it, and
THREE reproduced by this pass rather than taken on the reports' word.**

- **cliuser, two findings, both filed and both reproduced from a binary built
  out of the tree.** **#1367** — the no-rule-ID class does not print the bare
  message all three contract copies promise; both members print
  `parser: <message>`, an internal package name standing where every other
  stderr line in the binary writes `goxsd8: ` or `goxsd8: <subcommand>: `.
  **This pass corrected the report's central claim**: cliuser called it a
  regression against #1313's widening, and it is not — `665437a` touched no
  rendering, the prefix predates it on both members and is pinned by two
  assertions in `parse_test.go`. What the widening did was make one inaccurate
  sentence cover two members instead of one. **#1368** — a `-schema` document
  whose root is not `xs:schema` is rejected on `validate` as
  `goxsd8-schema-set.xsd:2:1: [src-include] …`, a `<loc>` naming the synthesized
  wrapper (`schemaSetLocation`) that exists in no file and is documented
  nowhere. **This pass traced the mechanism and found the sharper fact the
  report did not state**: `readSchemaDoc` reads the document for its
  `targetNamespace` alone and never asks what the root element is, so the
  document is `xs:include`d and charged at the wrapper — while the OTHER class
  member keeps its bare line on the same subcommand. **The two members diverge
  by subcommand, so #1313's carve-out covers one of them on `validate`.**
- **libuser, four findings, three filed and one deduped.** **#1369** — an
  UNDECLARED no-namespace attribute is silently dropped: `xs:openAttrs`
  (`xmlschema11-1.md:4412`-`:4425`) admits `##other` only, so `minOccur="2"` on a
  local `xs:element` compiles with the DEFAULT occurrence and exits 0.
  Reproduced and grounded in the spec by this pass. **The report's framing was
  sharpened**: it read `xsderr`'s *"a prohibited or missing attribute"* as the
  overclaim, but both of those ARE enforced — the unenforced case is the
  undeclared one, which no doc in the tree names at all, so the surface
  under-bounds the class rather than overclaiming it. **#1370** —
  `builtin/doc.go:44` says a minimal backend implements *"the ~25 primitives"*
  where five other sites, two of them in the same `go doc` output, say 19 (+
  `precisionDecimal`). **#1371** — an s4s-grammar rejection offers a library
  caller no classification route: `errors.As`, `RuleOf` and `LocOf` are all
  false by design and no sentinel is exported anywhere.

**ONE finding deduped rather than filed, and this is the first consultation in
this record where a persona independently hit an issue already open.**

- **`builtin/native` called a "shipped" backend by the module root doc → this is
  #1286 ITEM 1, open since the 2026-09-06 steward audit**, and the disposition is
  a comment on that thread (`issuecomment-5595240764`) rather than a sixth
  filing. The persona reached it cold, from `go doc github.com/kud360/goxsd8` —
  the exact entry point README sends a new consumer to — having never seen items
  2 through 5. **It adds one banding fact the body did not have**: the SAME doc
  block gets the distinction right two lines later, annotating `codegen, codec`
  as *"(M9, M10 — not yet implemented)"*, so item 1 has a working template
  inside the paragraph being fixed and is independently landable.

**Two cross-reference comments were posted so the family is sequenced rather
than rediscovered**: on **#1313** (closed — recording that the landing did what
it promised and that #1367/#1368 are what the consultation found), and on
**#1354** (naming #1367 and #1371 as the two issues that read its membership
ruling as an input).

**#1033 remains the one row no persona has ever looked at.**

### Working band

**Re-derived from this pass's evidence.** The last band is exhausted — eleven of
twelve rows landed — so this is a fresh ordering, not a re-shuffle. **#413
carries a LIVE claim and is not here**; re-run `wipsurvey` before starting
anything.

| # | issue | why here |
|---|---|---|
| 1 | #1328 | **The first M4 slice in this record whose body carries BOTH a measurement and an enumeration, which is exactly what this window proved a named single case is not.** `use`, `form`, `elementFormDefault` and `attributeFormDefault` are compared RAW at every site: ten shapes were driven through `parser.Produce` by the pass that filed it and all ten are accepted today — `use=" required "` MEASURES `{required}`=false, `form=" qualified "` mints a local name in NO namespace, `use="foo"`/`form="Qualified"`/`form=""` are silently taken as the defaults — and the enumeration half carries a complete candidate set of **24 named suite fixtures**, listed by name. It is #455's carved successor and the direct sibling of #456, which banked **+20** on the same ·actual value· footing. Direction is UP: every shape it changes is an acceptance today |
| 2 | #1332 | **This window's own measured cost, and the second consecutive window whose finding is about ratchet predictions.** #931's band row instructed a `suiteindex` census in as many words; nobody re-read it, an eighteen-day-old hand probe went to the verdict intact, and the lane moved +6 where the body named 4. The last stamp's #1205 embarrassment was a prediction made before the instrument existed; this one is a prediction the instrument could have corrected, with the instruction sitting in the band row the session was working from. The body already picks its own hard part — ONE named owner, no third copy of the `suiteindex` rule — so it is one doc-only session |
| 3 | #1359 | **A check that reports `ok` while its cases never execute, which is the same species as #1300 (last stamp's row 2, landed) and worse than a check that fails.** `TestCheckLandingRealHistory` SKIPS in a fresh container: #813's fixture pair sits on a deleted branch, `requireFixtures` discards the two REACHABLE cases along with it, and `go test ./tools/landcheck/` returns `ok`. `landcheck` is what mechanizes WORKFLOW's landing preconditions, so the thing not running is the thing that guards landings. Three arms, argued in the body, and the issue owes the choice |
| 4 | #1354 | **Take this before either row 5 or row 7 — it rules on the MEMBERSHIP those two describe.** `violationLine`'s doc enumerates two members of the no-rule-ID branch and a third prints through it: `parse.go:737`'s non-`ErrNotFound` resolver fault, reachable through the CLI's own `loader.Dir`. Filed by #1313's post-land pass and already carrying the arbiter's independent trace. Doc-only, and it is the cheapest of the three |
| 5 | #1367 | **A false sentence in the copy `README.md` itself calls authoritative, about the one thing an operator scripts against — and this consultation is the second outside reader to land on that same sentence.** All three copies promise the bare message; both members print `parser: `, an internal package name where every other stderr line writes the program name and subcommand. Two honest routes, and the body picks neither. **Its wording is constrained by row 4's ruling**, so take them in that order or take them together |
| 6 | #1369 | **An M4 under-rejection whose direction is guaranteed by construction, found from outside, with NO marker anywhere in the tree.** `xs:openAttrs` admits `##other` only, so an undeclared no-namespace attribute is an s4s fault — and `minOccur="2"` compiles today with the default occurrence and exits 0. Below row 1 deliberately: the population is unmeasured and `suiteindex` cannot census it (it censuses by construct, not by attribute name), so **census the attribute names directly, the way #456 did**. It also carries a real design bar — the roster must be GENERATED from Appendix A, never hand-typed (PRINCIPLES 26) — and a scope judgment the taker must settle first, which is why it is not row 1 despite being the larger correctness win |
| 7 | #1368 | **The `validate`-side half of rows 4 and 5, and the one that puts an operator in front of a filename that does not exist.** A `-schema` document whose root is not `xs:schema` is charged `[src-include]` against `goxsd8-schema-set.xsd:2:1`; the same input through `parse` is bare and rule-free. `readSchemaDoc` reads only `targetNamespace` and never the root name, which is the whole mechanism. Distinct from **#1224**, which owns the hint path and the opposite behaviour (elision), and from the #1251 landing's TOCTOU ruling, which is about a different line on a path this one does not use |
| 8 | #1324 | **The only row in this band that discharges a `blocked` entry, and it discharges in either direction.** It challenges the eighth `/retro`'s deliberate one-copy ruling on the foreground-gate rule, asking whether `CLAUDE.md`'s gate block — the arrival path #1279 governs — should carry a pointer. #1344 waits on the ruling and closes with it whichever way it goes. The body states both outcomes as checkable `grep` results, which is what makes it one session |
| 9 | #1361 | **A correctness defect #1349's landing exposed rather than introduced, in the code that landing rewrote.** The name-keyed component memos hand back the ORIGINAL for a redefining `xs:group`/`xs:complexType` whose expanded name is also contributed, so `sch-props-correct` clause 2 cites one location twice and `src-redefine` 6.2.2 goes vacuous. Freshest evidence in the queue, and the session that owns the surrounding code has just landed |
| 10 | #1167 | **Two measured sightings (#414, #1115), unchanged after fourteen landings, and PRINCIPLES 27 says a repeated grep wants a tool.** `gapaudit` reconciles marker → issue and issue → marker; nothing audits prose that points AT a marker or its file. **Still exactly two sightings**, kept honest deliberately — this window added none. **Its branch is EXPIRED, not free**: post the `TAKEOVER:` naming the tip you found before pushing, and read the thread's `PROCESS NOTE` first, because the pre-takeover tip is a stray commit for a different issue |
| 11 | #1356 | **A ruling the queue owes itself, measured at 55 open bodies.** They cite `file:line` into `cmd/goxsd8/doc.go`, `main.go` and `README.md` — files that took six landings in five days — against a WORKFLOW rule that already calls a line number a one-session shelf life. Two post-land passes have re-anchored citations as precedent with no document asking them to. Banded below rows 2 and 3 because its deliverable is a RULING among at least three arms rather than a fix, and one arm (a `citecheck` over the existing survey feed) is a session of its own that the ruling has to authorize first |
| 12 | #1318 | **A contract copy that denies a behaviour `flag.Parse` gives it, in the argument vocabulary `go doc` calls authoritative.** `doc.go` denies `--` any end-of-options meaning while the flag package supplies one, and the valued-help sentence miscounts the same way. Same reader and same three files as rows 5 and 7 — take it with them if any of that group is opened, since whichever lands last rebases the other two |

**Below the band, and why**: **#1297** pairs with the landed #1304 — that
landing consolidated `renderParent` into `renderName`, so #1297's namespace
rendering ruling now reaches `parent=` and every child from one site, and it
should ride whoever next opens `tools/suiteindex`. **#1266** and **#1267** are
the other two `suiteindex` follow-ups; #1266 carries a measured 232-vs-236
corpus discrepancy. **#1370** is this pass's one-line `builtin` doc fix and is a
pure ride-along for any session opening `builtin/`. **#1371** is the fourth
libuser finding and is a SURFACE ruling — route 1 mints a discoverable marker
and wants a warden pre-flight plus row 4's ruling first, route 2 is a disclaimer
and is takeable today. **#1286** rose on this consultation's independent hit
(item 1 alone is now independently landable) and is the natural partner for row
5, sharing a reader if not a line. **#1301** and **#1307** are the two
inline-copy refactors #1270 and #1246 carved, the same shape in the attribute
and element producers. **#1310** and **#1346** both pin `cmd` behaviour that
shipped untested and should ride row 7. **#1333** and **#1342** and **#1343**
are #931's and #1247's carved follow-ups and each names its own file.
**#1339**, **#1351**, **#1355** and **#1319** are this window's process
findings, all four about accounts and closures rather than about code; **#1355**
is the sharpest (an agent account that outlives the git state it describes) and
should be read with #1227's landed precondition. **#1325** and **#1363** are
citation-integrity tooling and pair with row 11. **#1336** pairs with the landed
#725. **#1276** is the derived check that makes `census.go`'s Scope section
unfalsifiable by drift and now has a second reason to exist: row 6 is a hole on
the axis that section does not cover. **#929** is the s4s family's other live
member and is banded below row 1 on the same criterion — census first.
**#1314** and **#1315** are the eighteenth consultation's unlanded filings;
#1315 wants a warden pre-flight and must be read with **#1283** and **#1051**.
**#1250** and **#1257** are `cmd` rows on the same code path as row 7.
**#1291**, **#1292** and **#1293** are the seventeenth consultation's and are
unmoved. **#1285** and **#1287** are the 2026-09-06 steward audit's other
findings. **#1201** retires the two `GAP(conformance)` markers. **#1183** and
**#1196** both narrow #1051. **#1122**, **#1232**, **#1233**, **#1234**,
**#1235**, **#1236**, **#513**, **#1003** are one-sentence fixes with settled
directions. **#1033** is still the only row no persona has ever looked at.

**There is no `Increasing` steward ranking anywhere in the band, for the FOURTH
consecutive stamp.** A scan of all 283 open bodies for `Cost of delay` returns
exactly **three** — **#849**, **#848** and **#845** — and none reads
`Increasing`: #848 and #845 are *"stable debt"*, and #849's own body records
that its ranking is falsified and that *"a steward re-ranking is warranted"*.
The `kind/refactor` cost-of-delay clause of the banding rule therefore selects
nothing again. **No `/retro` ran this window** — the last was 2026-09-06 and the
routine is weekly — so the ranked backlog did not grow because the one
instrument that can grow it did not run, which is a different fact from the last
two stamps' finding that it ran and produced nothing.

### Next planning action

1. **Prefer an ENUMERATED candidate set over a named single case, and say which
   one a slice has.** This window's two movers both banked populations their
   bodies had not identified: #456's named case was blocked two layers
   downstream by #717's `GAP(xsd)` and did not flip, while twenty unnamed
   fixtures did. A named case proves the population is non-empty; it does not
   bound it, and it can be intercepted by a gap nobody tested against. **Row 1
   (#1328) is banded on 24 named fixtures plus ten measured shapes**, and row 6
   (#1369) is banded below it precisely because its direction is guaranteed and
   its population is not yet counted.
2. **Fix the re-reading gap before the next lane slice, not after it.** #931's
   band row named the census, the census was never run, and nothing in the
   process re-reads a body's ratchet prediction before it is acted on.
   **#1332 is row 2** and is doc-only. This is the second consecutive window
   whose headline finding is a prediction defect, and the first where the
   correction was already written down and simply not read.
3. **`wip/issue-413` is LIVE and `wip/issue-1167` is EXPIRED — treat them
   differently.** #413 is off-limits, mid-repair-round, and so is the
   open-content deferral code around it. #1167 is takeable by posting a
   `TAKEOVER:` and pushing a heartbeat; it is band row 10 and its branch tip is
   NOT its own history. **Re-run `wipsurvey` before starting anything** — this
   pass's own survey watched `main` move four commits between the session brief
   and the first measurement.
4. **The nineteenth persona consultation is folded and nothing from it is handed
   off.** Six findings: five filed (#1367, #1368, #1369, #1370, #1371), one
   deduped into #1286 with a comment. **The next consultation is owed against
   whatever next changes the observable surface** — naming the trigger rather
   than scheduling a pass has now worked five windows running and correctly
   predicted the firing landing twice. Band rows 1, 4, 5, 6, 7, 9 and 12 change
   it; rows 2, 3, 8, 10 and 11 do not.
5. **A clean marker census is not a clean tree, and #1369 is the witness.** Ten
   consecutive stamps of zero untracked GAP sites mean every MARKED fail-open
   has an owner. #1369 is an unimplemented s4s check with no marker at all,
   found by a persona compiling a schema with a one-letter typo — invisible to
   `gapaudit` by construction. **#1276** is the nearest instrument and covers
   element positions only; whoever lands it should rule whether the attribute
   axis is in its scope, which today is out by silence rather than by decision.
6. **#849's steward re-ranking is owed for a FOURTH consecutive stamp, and the
   condition the last stamp set has NOT yet been met.** It said *"file it at the
   next `/retro`, or at the next `/backlog` if the retro passes over it again"*.
   No `/retro` has run since 2026-09-06, so the retro has not had another turn
   and nothing is filed here. **The next `/retro` is the deadline**: if it
   passes over #849 again, the next `/backlog` files the routing issue — the
   subject being that a routed re-ranking has no ledger and therefore never
   happens, which is #330's rule applied to the steward's own hand-offs.
7. **The human decision blocking #1002 is unchanged and is now carried for a
   FIFTEENTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`,
   enumerated by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until
   real assertion evaluation lands. CLAUDE.md puts (a) beyond any agent —
   *"changes only via a human-filed issue"* — and (b) depends on **#1042**,
   filed and `blocked`. **No agent should attempt either.**
8. **M6's carve is still owed and #1042 is still its only member.** #1042
   (`blocked`, `kind/gap`, M6) owns `cvc-assertion` (§3.13.4.1) and
   `cvc-assertions-valid` (§4.3.13.3). **M6 tier 2 itself is uncarved** —
   `$value` binding, an F&O function library, typed comparison — and is too big
   for one issue. That carve is a `/backlog` act at the M6 opening, and #1042 is
   the thing it must slice around rather than a blank page. #1042's
   `## Depends on` names #719, which is **closed**: the live dependency is the
   XPath evaluator itself, a trigger, and the body says so.

**Standing, and re-checked rather than restated.** Four unlanded corrections
still target one paragraph of `docs/WORKFLOW.md`'s filing discipline —
**#510**, **#646**, **#679**, **#912** — and whichever lands last rebases three
times; the 2026-09-06 `/retro` rewrote a neighbouring bullet of that same
paragraph, so re-read it before grounding any of the four. **The next `/retro`
inherits four**: the fold-the-five-species question (#635, #912, #609, #510,
#646), the `[tests that cannot fail]` **pattern** (routed by #472's post-land,
with no filed carrier), **#849's owed steward re-ranking**, which item 6 above
escalates, and now the **fourteen-landings-in-one-window throughput** this
stamp measured — the band was sized 12 and consumed 11, so the band is no longer
larger than a window and a `/backlog` that runs daily against a queue moving
this fast is banding for less than one dispatch ahead. The `gh --paginate`
page-2 403 trap stays a `/retro` datum and not a filing. The CTA cohort's 45
banked `instance` failures remain unattributed. `gate.yml` runs and is still not
a required status check, which only the repository owner can change.

**Environment, one witness each.** **`main` MOVED under this pass, and the
session brief was stale about it**: the brief named `faf2017` as
`origin/main`, and `origin/main` was `1ba9b40`, four commits ahead — #1349 and
#456 with their post-land passes. The working branch was **fast-forwarded**
(`git merge --ff-only`, no commit created, nothing pushed) before any
measurement was taken, and **every number in this section is at `1ba9b40`**; the
first `lanestatus` run, against the stale tree, read `schema` 13948/1450 and is
not what is tabled above. Repository-scoped REST served every read and every
write: **8 writes across 8 distinct issues** — **5 filed** (#1367, #1368, #1369,
#1370, #1371) and **3 thread comments** (#1286's dedupe disposition, #1313's and
#1354's cross-references) — plus this section's own replacement. **No issue body
was PATCHed and none needed to be**: this window's fourteen post-land passes
corrected bodies as they went, and the `## Depends on` sweep and the
landed-issue citation scan over `## Milestones` both came back clean. **The
paginate recipe ran to 14 pages twice**, 13 full at 100 each time and the last
page short at 66 then 71, with the post-loop fullness check passing on both.
**Three probes were run against a binary built from the tree** rather than
reasoned about — #1367's prefix on both class members and both subcommands,
#1368's wrapper-cited rejection, and #1369's silently-accepted `minOccur` typo —
and two of the three corrected their report in mechanism while confirming it in
outcome. **No conformance measurement was taken by this pass**: the lane table
above is the committed expectations, which `docs/WORKFLOW.md` names as the lane
score (#1120).

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
`schema` +34 on 2026-08-28. **#471** landed the local `<element ref=>` carrying
`substitutionGroup=` on 2026-09-03 (`31263ed`, `Ratchet: unchanged` — the
grammar-level `use="prohibited"` shape rejects with no case in the corpus to
bank) and its grounding carved two siblings from its scope: **#1205**
(`final=`/`abstract=`, the other grammar-prohibited pair) and **#1206**, which
**landed 2026-09-04** (`1780670`, `schema` **+16**) — `src-element` clause 2.2's
prose-only eight attributes on a local `<element ref=>`. **#1205 LANDED
2026-09-06** (`24ac7d3`) for `schema` **+5**, and it retired this family's
"ratchet-neutral by construction" inheritance: #471's neutrality came from
reaching the `ref=` form ALONE, while #1205 also reaches the inline form, where
six local fixtures exist and five moved. Its `ref=` arm IS neutral as predicted
(**0** fixtures suite-wide pair `ref=` with either attribute), and the inherited
prediction never shipped as a claim because that issue's own Acceptance made
measurement a precondition. **#1206's own landing then
carved two more**, from a UTF-16LE corner of the suite no UTF-8 grep had
reached: **#1215** (`src-element` clause 4, `ed-with-ns`) and **#1216**
(`src-attribute` clause 6, `att-with-ns`). **Both LANDED 2026-09-04** —
`976bdc2` for `schema` **+4** and `aeaf59e` for **+5**, each banking one and two
cases past its own prediction — and **each carved a successor from its own
boundary**: #1216's two `GAP(xsd)` markers became **#1242** (clause 6.1's one
reachable failure, a local `<attribute ref=>`) and **#1243** (an
`<attribute use="prohibited">` escaping clause 6 entirely), and **both have now
LANDED** — #1243 on 2026-09-05 (`3febb22`) and #1242 on 2026-09-06 (`6e95f25`),
**each `Ratchet: unchanged` and each measured that way by `go tool suiteindex`
construct census rather than assumed** — ratchet-neutral like #471 before them
(NOT like #1205, which banked **+5** on its inline arm), but the first two of
the family to prove it with an instrument instead of a grep. **#1243 carved one successor, #1242 carved none**,
which ENDS the run below: #1270 (`src-attribute` clause 4 charged nowhere on the
`use="prohibited"` branch) was #1243's, and #1242's landing closed its arm with
no residue. **#1270 has itself now LANDED — 2026-09-06, `d0fd4c6`,
`Ratchet: unchanged` and measured that way by construct census — and it carved
NO successor of this family either.** Its two residues are #1301 (a duplicate
clause-4 predicate surviving at the top-level producer, still open) and #1300 (a
process defect about how a gate command is invoked — **LANDED 2026-09-07,
`34aee07`**, `Ratchet: unchanged`); neither is a producer that decides and
ACCEPTS, so neither extends this family. **#455 has now LANDED too — 2026-09-07,
`8aca2a3`, `Ratchet: unchanged`, structurally excluded before it was measured
twice — and it CARVED a successor: #1328** (`use`, `form` and the two
`*FormDefault` attributes compared raw at every site, measured accepting
`use="foo"` and reading `use=" required "` as `{required}`=false). It also
unblocked **#456**, which was filed long before and is a discharge rather than a
carve. **#931 has now LANDED too — 2026-09-08, `c5e5e375`, `schema` +6
(fail → pass), zero regressions** — the occurrence attributes on a named
`<group>`'s body compositor, and this chain's first non-zero since #1205's +5.
**It ends the zero run**: the four landings before it all read
`Ratchet: unchanged` (#1243, #1242, #1270, #455) and the fifth banked six, so
ratchet-neutrality is a property of the individual member and not of the family,
and no further member may inherit the prediction. **It is also the one member
whose count was NOT taken by construct census** — its prediction came from a
hand-written producer-level probe, was wrong on three of its four named
candidates and under-predicted the lane by two, which is the defect **#1332**
owns. It carved no producer-side successor: #1332 is `kind/process` and **#1333**
pins the `xs:redefine` entry into `buildDefinitionModelGroup` by test, and
neither decides and ACCEPTS. **#456 has now LANDED too — 2026-09-08,
`5b84206`, `schema` +20 (fail → pass), zero regressions** — every one of the
twenty a suite-declared-`invalid` fixture writing an xs:boolean schema
attribute outside `booleanRep`. **It is the SECOND consecutive non-zero and the
largest single bank in this chain's record** (against `unchanged`, #1205's +5
and #931's +6), which settles what #931 opened: the run of four ratchet-neutral
landings before it was the anomaly, not the rule, and no member may inherit
either prediction. **It also answers #931's census defect from the other
side.** `go tool suiteindex` could not take this census either — it censuses by
construct, and this population is defined by attribute VALUE — but the member
said so before being asked and predicted the twenty EXACTLY by censusing the
attribute values directly. **Naming the instrument's limit is what #931 did not
do**, which is the distinction #1332 now owns. It carved no successor: its one
unmet acceptance item was dispositioned to the existing #717, not filed. Live
producer-decides-and-accepts members are now **#929** — which #455's landing
widened to own `minOccursZero`'s clause-2.1.3 compare as well as
`maxOccursZero`'s — and **#1328**. **The family
carved a successor at
every landing from #471 through #1243, and #1242 is the first that did not** —
one reading is the tail reaching zero, the other is that this pair was narrower
than its predecessors; two more landings decide it. **#1270 was the first of
those two and came back empty; #455 is the SECOND and came back with #1328**, so
the test the last stamp set is answered and the tail reading is not the one to
plan on: the family is still carving. Note what carried it — #455's successor
came out of an ARBITER's non-blocking scope note, not out of a construct census,
which is a different generator from the one #1270 and #1242 exhausted. A
second, narrower family opened beside it — the rejections the
producer already makes **correctly but describes badly**, whose bar
`xsderr/doc.go` set with #966 — and it is **discharged**: #975 landed at
`1dcffbf`, so every s4s-grammar rejection now names its Appendix A production.

**A THIRD family is the one that has been paying, and it has SIX landed
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

**The sixth member is #1275 (`34c39be`, 2026-09-07) and it is the first to bank
NOTHING — because it did not use the criterion, it measured.** A twelfth
`s4sModel` orders an `<alternative>`'s own children against `xs:altType`, the
same section 5.1 first-bullet fault #1047 and #1076 charge at their positions;
the corpus was censused before implementation and holds **no** `<alternative>`
carrying an out-of-model child across 232 occurrences in 81 fixtures, so
`Ratchet: unchanged` was predicted and then measured. **The criterion is a proxy
for a census that `go tool suiteindex` can now take directly**, and where the
two disagree the census is the fact — that is the methodological change this
member carries, not a sixth data point for the proxy. Read the criterion's
five-for-five as five-for-five *among issues banded on it*, which is a smaller
claim than it looks.

**The family replenishes itself as it is worked, which is what to expect and
not tail growth** — every issue it has added was filed by the post-land pass of
a landing that moved the lane. **Its shape is also changing**: #1047, #1076 and
#1099 all widened `checkS4SChildOrder` or its callers, and the issues arriving
now — #1097 (`32070b8`) and **#1136** (`d854e1d`, 2026-09-05), both landed, and
open #1098, #1133 and #1135 — are defects *in* that widening rather than further
sites for it. **A producer that decides more is a producer with more to get
wrong, and the census does not name that class.** #1136 is the one that stopped
the accumulation: it replaced four independent local run-order guesses with ONE
recorded decision, and its own follow-ups (#1246, landed 2026-09-07 as
`a363592`, and #1247, landed 2026-09-08 as `77b0419`) are about that record
rather than about a fifth guess. **That record now states a rule scoped by
WHERE rather than a family roster**, and #1247's own follow-ups (#1342, #1343)
are about its wording rather than its membership.

**#1275 shows the site-widening half is NOT exhausted, and it arrived from a
third route again.** It is a further SITE — a twelfth `s4sModel` at a position
nothing had ordered — not a defect in the widening, and it came from neither
#1030's census list nor a run-order defect: #1050's reconciliation of
`parser/census.go`'s Scope prose NAMED the silence in the tree and tracked it
with nothing, and #1275 was the filing that was owed. **Every widening site the
producer still owes is discoverable from that Scope section**, whose remaining
live silences it enumerates; the open **#1276** would make that section's truth a
DERIVED test rather than prose, which is what would turn the remainder into a
list the way #1030 did once already.

**Read the milestone count as a floor.** The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it — #1135 is M4 work carrying no
milestone today, and #1136 was too until it landed. **The sharpest instance is
#1126**, which moved `schema` +475 and `instance` +113 — the largest lane
movement this project has recorded — and carried no milestone either, so the
count did not move at all.

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
queues for a day. **The CLI's own `validate` subcommand, #720, LANDED
2026-09-04** (`f4d8f76`) and moved no lane by construction — `go list -deps
./conformance | grep -c cmd/goxsd8` is `0`. It is the reason this milestone's
count moves independently of its lane, and the 2026-09-05 window is the cleanest
demonstration yet: **three of its four subcommand issues LANDED** — #1223
(`4ab65d6`, exit 4 for an instance the assessment declined to decide), #1229
(`20010ed`) and #1230 (`38ade3e`) — **three more were filed onto it** (#1250,
#1251, #1257), and the milestone's open count did not move at all while
`instance` stayed flat at 11029. **Every M5 issue that entered or left in that
window was `cmd/goxsd8`.** **The roster of open subcommand issues is not written
here.** It went stale twice inside four days — #1260 and #1261 landed, then
#1251 — for the reason the lane figure below is not written here either (#646).
Read it off the queue: `area/cmd` on this milestone. The durable half is the
milestone rule behind that roster, and it is the reason the count reads low: **a
subcommand issue spanning `parse` (M4) as well as `validate` carries NO
milestone deliberately**, so this milestone's `cmd/goxsd8` work is undercounted
here rather than overcounted. Read the count as a floor and never as the lane's
remaining work.

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

**All ELEVEN decided-and-wrong `instance` cases carry an owner, and the two
classes below account for nine of them.** At `5b84206`: 15332 fail = 15316
decline candidates + 5 declined indeterminate + **11** decided-and-wrong. The
other two are owned outside these paragraphs (#1160) —
`MS-DataTypes2006-07-15/gMonth002_2061/instance/gMonth002_2061.v` and
`gMonth004_2063.v`, both by **#921**.

**It was TWELVE, and `Open/open013/instance/open013.v1.xml` is the case that
left — reclassified by #456, not fixed by it.** #456 landed the `boolAttr` fix
(2026-09-08, `5b84206`), so the fixture's `<xs:complexType mixed=" 1 ">` reads
its ·actual value· `true` and the false reject is gone; but a correct `mixed`
makes the ·effective content· a particle (§3.4.2.3.3 clause 2.1), routing to
clause 5.2.1's default open content, and `xsd.Schema.ContentMatcher` declines
any `{content type}` carrying `{open content}` under a `GAP(xsd)` owned by
**#717**. Same banked line, a decline candidate now rather than a wrong
decision — correctness up, lane score flat. The banked figure is
`go tool lanestatus`'s; the decline figures are the arbiter's, taken at the
same tree.

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
