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

## Status — 2026-09-10 (`/backlog`, the twenty-first. Replaced whole per step 6: the lane table is a fresh `go tool lanestatus` paste, the namespace a fresh `wipsurvey` against a `git fetch --unshallow origin` plus `git ls-remote --heads`, the marker census a fresh `gapaudit`, and the milestone and queue counts a page-numbered `state=all` fetch taken **after** this pass's own writes. **The window is SIX landings and the band was consumed from the top** — rows 1, 2, 3, 5 and 6 all landed and row 4 is held LIVE right now — so this is a fresh ordering rather than a re-cut. **`schema` 13968 → 14012 (+44), and the attribution is exact**: #1328 +18, #1369 +17, #1380 +9. `datatypes` and `instance` are unchanged. **The namespace carries ONE held claim**: `wip/issue-1332` is LIVE at 1h52m, and it is band row 4 of the last band. **The marker census holds at 68 and group 1 falls 18 → 16** — #1376's landing removed both `dead end: cites CLOSED #413` rows, which is this stamp's only tool-visible landing effect. **#731 was CARVED**, its own first Acceptance bullet discharged after four weeks: four mechanism issues (#1411–#1414) and the parent re-scoped to a `blocked` ledger. The **TWENTY-FIRST persona consultation**: **ten findings — five filed (#1406–#1410), one deduped (#1142), four dismissed on measurement or against a landed ruling (#1003, #1144, #1232, #672)**)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 11029 | 15332 | 26361 |
| `json` | — | — | 0 |
| `schema` | 14012 | 1386 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### The window: six landings, and the last stamp's own table was already stale

**`schema` moved 13968 → 14012, +44 pass and −44 fail, and every one of those
44 is attributed**: **#1328 +18**, **#1369 +17**, **#1380 +9**. The three sum
exactly, so no landing banked an unattributed case and none regressed.

**#1328's +18 is in that sum although the last stamp published 13968**, and the
reason is worth one sentence because it is the second time this record has hit
it. The 2026-09-09 second stamp measured the lanes while `wip/issue-1328` was
still CLAIMED, and #1328 merged before that stamp's own commit did — so the
published table was stale at the moment it was committed, by exactly one
landing. The same is true of that stamp's M4 counts (52 open / 124 closed, where
#1328 counted gives 51 / 125). **Nothing about the stamp was wrong when taken;
it was wrong when landed.** A `/backlog` that runs inside a window pays this,
and the only defence is what this section already does — one date stamp for the
whole section, so a reader tells staleness from wrongness.

**The other three landings moved nothing and are not filler.** #1167 (a mason
scope rule), #1361 (a memo keyed by declaration rather than expanded name) and
#1359 (a landcheck test that reported `ok` while skipping) all ran
`Ratchet: unchanged` in write mode rather than inferring it. #1376 likewise, and
its effect is the one visible in this stamp's own tool output rather than in a
lane.

**Two prediction data points, and they point the same way as the last two
stamps.** #1380's body predicted a lane move from the parser charge alone;
**mason measured it at ZERO** and found the twelve suite witnesses were all
declined by `conformance/schema.go`'s top-level `default` arm, so the harness
widening had to be absorbed to bank anything. #1369 predicted from a direct
15,470-file corpus census — hand-rolled, because `suiteindex` censuses by
construct and cannot reach an attribute-name axis — and banked +17. **Both
confirm the last stamp's ranking criterion and sharpen it in the same
direction**: the prediction is only as good as the census behind it, and the
census is only available where a tool can reach the axis. That is why row 1 of
this band is a tool.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, against `git fetch --unshallow origin` and
this pass's 779-issue pre-write feed:

```
ISSUE  BRANCH          LEASE AGE  VERDICT  REASON
732    wip/issue-732   428h58m0s  RETIRED  wip/issue-732: issue #732 is closed
822    wip/issue-822   606h56m0s  RETIRED  wip/issue-822: issue #822 is closed
846    wip/issue-846   379h17m0s  RETIRED  wip/issue-846: issue #846 is closed
872    wip/issue-872   572h57m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933   main's     RETIRED  wip/issue-933: issue #933 is closed
968    wip/issue-968   main's     RETIRED  wip/issue-968: issue #968 is closed
993    wip/issue-993   384h38m0s  RETIRED  wip/issue-993: issue #993 is closed
1332   wip/issue-1332  1h52m0s    LIVE     wip/issue-1332: tip pushed 1h52m0s ago, within the 2h0m0s claim TTL
```

**ONE claim is held and the last stamp shows a different one.** Both branches
the last stamp reported — `wip/issue-1167` LIVE and `wip/issue-1328` CLAIMED —
are **gone**, their issues landed and closed, which is what a merge is supposed
to do to a `wip/` ref. **Re-run `wipsurvey` before starting anything.**

**`wip/issue-1332` is LIVE**, tip `dc3df2a` (*"wip #1332: takeover heartbeat —
resume from arbiter…"*) 1h52m before this survey. It is **band row 4 of the last
band and is not banded here.** Its tip message says a previous holder was taken
over; the lease is current either way and no dating from the thread is needed.

**The same seven RETIRED refs, unchanged row for row — EIGHT stamps now — and
this pass re-measured the content question directly rather than inheriting it.**
`git merge-base --is-ancestor <tip> origin/main` over all seven: **`wip/issue-933`
and `wip/issue-968` ARE ancestors of `main`; the other five are NOT**, their tips
dated 2026-08-16 to 2026-08-25. **That is the expected shape, not a discrepancy**
— all seven closed `not_planned` and were superseded rather than merged, so their
content is not supposed to be in `main`. Closure is re-verified from this pass's
own feed; the supersede chain (#732→#1001/#1002, #822→#851, #846→#1029/#1030,
#872→#878, #993→#1018) is inherited from the last two stamps' verification, with
**#1002 re-confirmed OPEN and properly `blocked`** on the repo owner's ruling.
**No supersede is owed and none is missing.**

**Zero `needs-replan`. Zero `parked/*`. Zero `meta/*`.** **FIVE non-`wip`
`claude/*` refs stand**, unchanged in tip from the last stamp — `-39rk64`,
`-3xu0ki`, `-8jq9o6`, `-adewly`, `-kk1f7v`. The last stamp described `-kk1f7v` as
`origin/main`'s predecessor; it is `1ba9b40` and `main` is now `2fa48cf`, so it
is **seven commits back and no longer adjacent to anything**. Listed for human
triage, not acted on. `wipsurvey` reads `wip/*` and `parked/*` only, so these are
found by `git ls-remote --heads` and never by the survey.

### Marker census — 68 markers, group 1 falls 18 → 16, zero NEW untracked sites

**`go tool gapaudit`** over the whole 779-issue pre-write feed: **68 `GAP(`
markers across 8 areas** — `xsd` 32, `validate` 17, `xpath` 6, `xml` 4, `parser`
3, `value` 3, `conformance` 2, `cmd` 1. **Nothing added, nothing deleted**, which
is what six landings that touched no marker look like.

**Group 1 falls 18 → 16, and that fall is #1376's landing measured from the
outside.** Both `dead end: cites CLOSED #413` rows are gone; #1374 simultaneously
left group 2 (*"OPEN `kind/gap` issues no marker cites"*), the same
reconciliation read from the other end.

**The last stamp's finding is discharged and did not recur.** Its two greps were
re-run — `No issue owns`, `owns this residual` — and return **five hits, all
accounted for**: `xsd/wildcard.go:121` cites #248 and is fine; `contentrestricts.go:696`
is **#1378**'s; `defaultbinding.go:355` and `:549` are **#1379**'s;
`contentrestricts.go:840` cites **#499**. **Zero new prose disclaimers.** The
three that #1378 and #1379 own still sit in group 1 and **will until those issues
land** — the tool joins on citations and the numbers are not written into the
markers yet, which is exactly the work those issues own.

**Two group-1 rows are #1156's, not new findings, and this pass checked rather
than assumed.** `contentrestricts.go:794` **does** cite #499 — at line `839`, five
paragraphs below the marker head, past the blank `//` at `:800` that
`gapaudit`'s `paragraph()` stops at. `contentrestricts.go:1099` is the
element-side `key-dft-binding` cases 4/5 marker #345 owns. **#1156 names both
explicitly**, so nothing is filed; the placement defect is the same one #1376
just landed the fix pattern for, and **#1378 was given that pattern in a comment**
because its own row now carries a `dead end: cites CLOSED #501` annotation
(provenance, not ownership — but a route-(b) rewrite that leaves it unmarked
leaves the row standing).

**So: zero untracked GAP sites — a judgment on the annotations, not the tool's
mechanical rule.** The last stamp ENDED a ten-stamp zero streak by finding that
the streak had been an artifact of applying that rule; this is the first stamp
since, and it reports zero having re-run the greps the artifact was found with
rather than by inheriting either answer.

### Milestones and queue

Counts from a page-numbered `state=all` REST fetch taken **after** this pass's
writes, 15 pages (14 full at 100, page 15 at 14), PRs filtered locally:
**1414 rows, 626 PRs excluded, 788 issues — 298 open, 490 closed.** Coverage
verified rather than assumed: 1414 distinct numbers, min 1, max 1414, no gaps.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **51** | **128** | active |
| **M5 — Instance validation (XML)** | **15** | **24** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Queue labels, open only: **280 `ready`, 16 `blocked`, 0 `needs-replan`, 2
`epic`** — summing to 298 with no gap, and **every open issue carries exactly
one, verified over all 298 by grouping each issue's own label set rather than by
summing two counts**. **#779** owns the mechanical check and carries this as its
fifth class.

By kind: `kind/refactor` 80, `kind/gap` 57, `kind/process` 56, `kind/tooling`
39, `kind/story` 30, `kind/bug` 27, `kind/docs` 23, `kind/feature` 4. By area:
`parser` 82, `meta` 79, `xsd` 63, `docs` 35, `conformance` 31, `validate` 26,
`cmd` 24, `value` 16, `builtin` 11, `xsderr` 6, `xpath` 6, `model` 4, `regex` 2,
`loader` 2, `cli` 1.

**M4's open count is FLAT at 51 and that is the interesting number.** Three M4
landings closed (#1369, #1380, #1361) and **three M4 issues were opened by their
own post-land passes** (#1389, #1390, #1397) — one for one, which is what a
window of clean landings against a rich seam produces and is not a stall. The
closed count reads 124 → 128 rather than → 127 because the last stamp's own
figure did not yet include #1328; see the window section. **M5 is unchanged at
15 / 24** — nothing this window touched it.

`ready` moved **272 → 280**: nine filed by this pass (#1406–#1410 from the
consultation, #1411–#1414 from the #731 carve) and one retired from the label
(#731 itself, `ready` → `blocked`). That is #347's shape — `ready` is an output,
not a target — and neither milestone's count is its lane's remaining work.

**`ready` 280 is the honest startable count with ONE to subtract**: #1332 carries
a live claim, so **279** are startable without colliding.

**The unblock sweep measured a clean zero for the THIRTEENTH consecutive stamp**,
measured at the source over all **16** open `blocked` bodies rather than inherited
from any post-land pass. **No `## Depends on` in the partition names any of the
window's six closures** (#1167, #1328, #1359, #1361, #1369, #1376, #1380 — seven
counting #1328). Every named issue dependency — **#1324**, **#407**, **#250**,
**#831**, **#591**, **#248**, and now **#1411**/**#1412**/**#1413**/**#1414** —
was checked individually and is **open**. **Zero relabelled.** The set is 16, up
one from 15: **#731** joined it as a carved ledger and nothing left. Re-read the
partition below rather than re-deriving it:

- **FIVE are triggers rather than issues** and say so in their own
  `## Depends on` — **#1374**, **#1224**, **#555**, **#1002** and **#1042**.
  **Do not re-scan these on the next sweep.**
- **TWO have every named issue dependency closed and each states in its own body
  why that is not a discharge** — **#16** and **#1051**.
- **NINE still have at least one OPEN named dependency** — **#248**, **#267**
  and **#345** (all on #250), **#415** (#407), **#593** (#591), **#717**
  (#248), **#871** (#831), **#1344** (#1324) and **#731** (the four carves).
  **#731 is the one the band can discharge fastest**: rows 3, 4 and 5 are three
  of its four dependencies, so a window taking all three leaves it one issue from
  closing. **#1344's dependency is band row 11**, and it discharges in EITHER
  direction — #1324 closing `not_planned` is a recorded ruling just as much as
  #1324 landing is.

### #731 CARVED — the split this pass performed, and why it is step 3 work

**#731's own first Acceptance bullet had said *"Triage first, then split. Four
mechanisms in three packages is not one session"* since 2026-08-12, and no
session had performed it in four weeks** — because performing it is a
cartographer act and the issue sat in the `ready` queue where only the develop
loop looks. That is the whole lesson: **a `ready` issue whose first Acceptance
bullet is a carve is mis-labelled, and it will sit until a `/backlog` reads its
body rather than its label.**

The eleven `schema`-lane cases #442's widening reclassified from *declined* to
*decided-and-DISAGREEING* are now four issues, each carrying #731's case table
for its own rows verbatim, its own spec sections and #731's ratchet-integrity
clause:

| mechanism | cases | issue | why it is where it is in the band |
|---|---:|---|---|
| 1 — invalid F&O regex in a `<pattern>` facet | 3 | **#1411** | must precede #1414, which it makes witnessless |
| 2 — `<simpleType>` bound-facet consistency | 2 | **#1412** | smallest, one rule family, no fork |
| 3 — s4s grammar faults in `<simpleType>`/`<element>` | 5 | **#1413** | biggest prize, owes the #444 grounding |
| 4 — `substitutionGroup=` absent / non-element | 2 | **#1414** | **banks nothing if #1411 lands first** |

**All eleven are UNDER-rejections** — the suite says invalid, this processor
accepts — so **every carve's direction is `schema` UP or flat by construction**,
and a downward move means the rejection is too wide. **The overlaps are recorded
in each carve rather than lost in the split**, because they decide attribution
and in one case decide whether an issue banks anything at all: `elemE007`/`E008`/
`E009` each carry mechanism 1's `[0-9]{,5}` pattern as well as their own fault,
and `annotB026`/`annotB031` carry `stA012`'s `name=` fault as well as their own.

**#731 stays open as the ledger**, `blocked` on the four, because the check it
was filed for — *"none returns to being an unexplained `fail`"* — is the one
thing no single carve can answer.

**The evidence is TRANSCRIBED, not re-measured, and every carve says so.**
`testdata/xsdtests` is uninitialized in this container (`go tool suiteindex`
reports corpus-absent mode), so all four case tables are #731's 2026-08-12
measurement at `1dfd13c` with an instruction to re-confirm at grounding. **#1413
carries a sharper version of that instruction**: #1369 and #1380 landed s4s
tightenings this window and may already have flipped some of its five.

### Persona consultations — the TWENTY-FIRST ran, and the trigger fired correctly

The cartographer role-plays no persona and does not spawn one (#416): it has read
the source, so a verdict it produced would launder an insider's opinion as an
outsider's. **The orchestrating session ran both personas fresh against the
published surface** (README plus `go doc`, never source) and handed the reports
here.

**The trigger the last stamp proposed FIRED, and correctly.** Its rule was
*"consult when the surface changes, or when two windows pass without one,
whichever comes first"*. #1380 changed observable CLI behaviour — a typo'd
top-level construct went from `components: 0`, exit 0 to a charge — and #1369
changed which schema documents compile at all. So the surface arm fired on its
own merits, and the floor was never tested. **Keep both halves; this window is
evidence for the trigger, not for the floor.**

**Ten findings, disposed of as follows.**

**FIVE FILED:**

| # | finding | why it survived |
|---|---|---|
| **#1406** | `Violations()` returns `*xsderr.Error` (Rule/Loc/Msg as FIELDS) while `Unevaluated()` returns `validate.Unevaluated` (the same three as METHODS) — a consumer folding both into one response hand-writes an adapter | No open issue owns the shape asymmetry. **This pass added the constraint that decides the design**: a Go struct cannot carry a field `Rule` and a method `Rule()`, so `*xsderr.Error` cannot satisfy an accessor interface while #486's unexport is unlanded. The persona's *"without making `Unevaluated` implement `error`"* is carried into the body as a prohibition, because its own doc gives the reason |
| **#1407** | the CLI exit-code contract is five codes plus a severity rule spread over four paragraphs with no table, and *4 is less severe than 1* is disclosed last | Measured: `grep -n "^//\t|" cmd/goxsd8/doc.go` returns **zero**. Presentation only — the persona recorded the contract as *"accurate and thorough"* and ~30 adversarial probes found no drift, so the issue forbids changing any verdict |
| **#1408** | `M<N>` is a status marker on the published surface and is defined nowhere on it | Generalized from the persona's `gen`-only report by measurement: **40 sites across 19 `doc.go` files**, and `grep -rn "PLAN.md" --include=*.go .` returns one unrelated site. A `go install` consumer cannot resolve M8, M9, M11 or M12 either |
| **#1409** | no counts and no structured output from `goxsd8 validate`, so CI volume needs a regex | Measured: five flags, one of them (`-q`) read by nothing. Filed as a **decision** first — counts line, structured mode, or decline it the way #672 declined `-version` — with #1291 named as the precondition for the structured arm, since no violation carries an instance path |
| **#1410** | `-schema -` is refused for a correct reason with no escape hatch and no documented workaround | The refusal is correct and the body says so twice. What is missing is one sentence, exactly where `doc.go` already writes one for its neighbour (*"`./-` is what names that file"*) |

**ONE DEDUPED:**

- **#1142** takes libuser's worst-ranked finding (*"no documented path from valid
  to a usable Go value"*) as a repeat sighting. **The stronger claim came back
  FALSE for the third consecutive consultation** — `go doc ./builtin/strict New`
  documents, per type, exactly which `value` capability interfaces its value
  satisfies and says *"The concrete value types are unexported"* — which is what
  #1142's Acceptance bullet 2 already warns a session about. What the sighting
  adds is **rate**: three personas have now reached that enumeration and still
  concluded there is no route, and `grep -n "Override\|own backend\|arithmetic"
  builtin/strict/doc.go builtin/strict/strict.go` still returns zero. The
  conversion-helper half was routed to **#721**, the unbuilt-package half to
  **#1003**.

**FOUR DISMISSED, each on the thread that owns the decision:**

- **#1003** — *"`validate/jsonsrc` and `validate/bersrc` export ZERO symbols, not
  even a stub `Validate` returning `ErrNotImplemented`"*. The measurement is
  right; the ask is refused by **#409**, which settled the convention for those
  exact two files (*"describe what the package will be, not what it already is,
  until it exports something"*) and named them as the COMPLIANT examples, and by
  **#1123**, which landed the scoping. A stub export would violate STYLE 8 to let
  a caller compile against a signature M8 has not fixed.
- **#1144** — *"README's recipe fails: it sanctions `go doc -src` and `go doc`
  never indexes `_test.go`"*. **The premise is false**: `grep -rn '\-src'
  README.md docs/*.md cmd/goxsd8/doc.go` returns nothing, and README already says
  the opposite correctly. **The residue is real and is #1144's own**: README's
  actual instruction is *"read the example tests directly"*, which only works
  **inside the clone** — a third site with the precondition defect #1144 was
  filed for, and the sweep that fixes its two fixes this one free.
- **#1232** — *"one `.json` anywhere in a mixed-format batch aborts the ENTIRE
  run before any instance"*. **Measured false against a binary built from the
  tree**, six batches with the unsupported sibling in every position: XML
  siblings are assessed in all of them, the fault is reported per instance, exit
  is 2. `README.md:97-99` already states it in bold. The plausible source of the
  persona's reading is the short-circuit that DOES exist — a non-compiling
  `-schema` set exiting 3 — which is #1232's own subject and one more argument
  for putting the precedence in words.
- **#672** — *"no `-version`, and `go version -m $(which goxsd8)` needs a Go
  toolchain"*. Dismissed for the **second** consecutive consultation against the
  landed decline. **But the new half was recorded rather than waved off**: the
  workaround assumes a toolchain, so for a minimal CI image carrying only the
  binary the decline is total. That does not make the ruling wrong; it is what a
  1.0 revisit has to weigh, and it is the first sighting from that reader.

**Three of the ten reports were corrected in MECHANISM before being recorded** —
libuser 1, libuser 3 and cliuser 2 — and in two of those the correction reversed
the disposition. Every claim that could be probed was probed against a binary
built from the tree or against `go doc`, never taken on the report's word.

### Working band

Ordered for a `/develop` session: take the highest row you can start. **Row 4 of
the last band (#1332) is HELD LIVE and is not here.** Re-run `wipsurvey` before
starting anything.

| # | issue | why here |
|---|---|---|
| 1 | #1391 | **A tool, banded above every lane slice by the `kind/process`/`kind/tooling` clause, on friction two landings paid.** `suiteindex` censuses by construct and has no ATTRIBUTE axis, so **#456 and #1369 each hand-rolled a 15,470-file corpus census** — #1369's log entry says so in as many words. That is friction compounding across consecutive sessions, which the clause says outranks a lane slice. It is also the instrument the last two stamps' ranking criterion depends on: *prefer a CENSUSABLE population* is unusable on an attribute-valued population until this lands, and #929, #606 and this band's own rows 3–5 all live there. One session, `Ratchet: unchanged` by construction |
| 2 | #434 | **The best lane slice in the queue, and the only banded row that names the cases it will flip.** §5.3 Missing Sub-components: the remaining by-name reference slots hard-fail `src-resolve` at finalize instead of retaining an ·absent· value, and `saxonData/Missing/missing001`, `missing003` and `missing006` move to **pass**. Direction is removal of false rejects, so up or flat. **#281 landed the precedent and the prose** — read `resolveElementDecl`'s comment first. Its §5.3 second half (·lax assessment· at assessment time) rides on #250 and its body explicitly scopes it OUT; **#271 is the reason `missing006` is harder than `missing001`, so sequence deliberately** |
| 3 | #1412 | **The cheapest lane movement in the queue.** #731 mechanism 2: two cases, one rule family, no grammar-versus-`src-*` fork to ground, no overlap with its siblings. Under-rejection, so `schema` up or flat by construction. Carries the one thing about it that is easy to get wrong — the fixtures are the `Schemas/` copies, **not** the same-named files under `Facets/integer/` |
| 4 | #1411 | **#731 mechanism 1, and it must precede row 6.** Three cases, guaranteed direction. Its ordering is not preference: `elemE007`/`E008`/`E009` carry its `[0-9]{,5}` pattern as well as their own faults, so landing this first banks them and leaves #1414 witnessless. Read **#546** first — the `src-pattern-value` rejection mechanism already exists in `value/facets.go` and #546 is working on its `Loc` |
| 5 | #1413 | **#731 mechanism 3 — five cases, the biggest of the four carves, and the one that owes a grounding.** The grammar-versus-`src-*` fork is **#444**'s question and must be answered before a line is written. **Re-check against `main` first**: #1369 and #1380 landed s4s tightenings this window and may already reject some of the five. Its ratchet prediction is required per fault, not in aggregate, because three of the five share `stA012`'s `name=` fault |
| 6 | #1354 | **Take this before rows 7 and 8 — it rules on the MEMBERSHIP those two describe.** `violationLine`'s doc enumerates two members of the no-rule-ID branch and a third prints through it, reachable through the CLI's own `loader.Dir`. Doc-only and the cheapest of the three. **Its evidence table was corrected by this pass** — `parser/parse.go:737` → `:775`, `:489`/`:592` → `:522`/`:626`; the four `cmd/goxsd8/parse.go` citations were re-read and are exact |
| 7 | #1367 | **A false sentence in the copy `README.md` itself calls authoritative, about the one thing an operator scripts against.** All three copies promise the bare message; both members print `parser: `, an internal package name where every other stderr line writes the program name and subcommand. **Its wording is constrained by row 6's ruling**, so take them in that order or together |
| 8 | #1368 | **The `validate`-side half, and the one that puts an operator in front of a filename that does not exist.** Re-reproduced by this pass at `2fa48cf`: a `-schema` document whose root is not `xs:schema` is charged `[src-include]` against `goxsd8-schema-set.xsd:2:1`, while the same input through `parse` prints a bare, rule-free `parser: assembling a schema requires a <schema> document root…`. Distinct from **#1224**, which owns the hint path |
| 9 | #1356 | **Promoted on this pass's own evidence, and the evidence is now a THIRD file family.** #1354's `parser/parse.go` cells had drifted +38/+33/+34 across #1349, #1361 and #1380 — while that same body's four `cmd/goxsd8/parse.go` citations stayed exact. **So the defect is not bounded by this issue's three named files**: it tracks whichever files a window touched, which a `citecheck` scoped to three filenames would miss entirely. **This pass also corrected #1354 in place** rather than deferring as the last stamp did for #1156 and #345 — the difference is banding, and the ruling should say which rule produced it |
| 10 | #1283 | **The largest row here, banded on a second independent sighting from the audience it is written for.** It **retires #1286 item 3, unblocks #1051's unexport, and deletes the wrapper-as-a-string copy** that #1224, #1250 and #1346 all work around. Needs a warden pre-flight. Read #1251's and #1260's two signature facts in the body before designing the entry point — the report half and the per-root provenance are both load-bearing |
| 11 | #1324 | **The only row that discharges a `blocked` entry, and it discharges in either direction.** It challenges the eighth `/retro`'s deliberate one-copy ruling on the foreground-gate rule, asking whether `CLAUDE.md`'s gate block — the arrival path #1279 governs — should carry a pointer. #1344 waits on the ruling and closes with it whichever way it goes. The body states both outcomes as checkable `grep` results, which is what makes it one session |
| 12 | #1400 | **Filed by #1359's own post-land pass, in the code that landing rewrote.** The fixture guard probes with a short-circuiting `merge-base --is-ancestor`, so an undiffable pair reaches `checkLanding` as a bare exit 128 — plus three same-file residuals. `landcheck` mechanizes WORKFLOW's landing preconditions, so a misdiagnosis here is a misdiagnosis about landings |

**Named below the band, deliberately:** **#1414** (#731 mechanism 4) is startable
and is **not** banded, because rows 4 and it are sequenced — if #1411 lands
first, #1414's entire named cohort is already `pass` and it banks nothing. It is
still real; it just has to predict `schema` unchanged and build a Go-level test.
**#1406–#1410** are this pass's consultation filings and none is banded: four are
doc- or decision-shaped and one (#1406) is gated on #486's unexport.

### Next planning action

1. **Band the instrument before the measurement it enables.** Row 1 is a tool and
   rows 2–5 are lane slices, which inverts this section's own default. The
   justification is measured, not stylistic: two landings hand-rolled the same
   15,470-file census, and the ranking criterion the last two stamps established
   — *censusable beats enumerated beats named* — is inapplicable to an
   attribute-valued population until **#1391** exists. **If row 1 lands, re-rank
   #929 and #606 at the next stamp**; both are attribute-valued and both
   currently predict `unchanged` for want of a census.
2. **A `ready` issue whose first Acceptance bullet is a CARVE will sit forever.**
   **#731** said *"Triage first, then split — four mechanisms in three packages is
   not one session"* on 2026-08-12 and no session performed it for four weeks,
   because the carve is cartographer work and the label sends the issue to the
   develop loop. **The general question is whether the label set can express
   this** — `needs-replan` retires an issue in place and is the wrong tool. Put
   it to the next `/retro`; until then, **a `/backlog` reading a `ready` body that
   asks to be split performs the split in that pass**, as this one did.
3. **The prediction question has a third data point and it is about the census,
   not the predictor.** #1380 predicted movement, mason MEASURED zero, and the
   cause was a harness decline no reading of the body could have found. #1369
   predicted from a hand-rolled census and banked +17. **#1332 is held LIVE and
   will land against exactly this**; when it does, the next stamp should ask
   whether its rule reaches the #1380 case at all — a body-versus-banked check
   catches a wrong figure and not a census that never ran.
4. **The persona trigger fired on its surface arm and the floor was untested.**
   Keep both halves as the last stamp wrote them. **The dismissal rate is the
   thing to watch**: four of ten this window, three of them against decisions
   already recorded on threads (#409/#1123, #672, and README's own text). **Each
   dismissal was written onto the owning thread precisely so the next
   consultation is not re-triaged from scratch**, which is the only mechanism
   that keeps that rate from compounding.
5. **#849's steward re-ranking is owed for a SIXTH consecutive stamp and the
   deadline is now three days out.** The standing condition is *"file it at the
   next `/retro`, or at the next `/backlog` if the retro passes over it again"*.
   The last `/retro` was **2026-09-06** and the routine is weekly, so **Sunday
   2026-09-13** is the deadline: if that retro passes over #849 again, the next
   `/backlog` files the routing issue — the subject being that a routed
   re-ranking has no ledger and therefore never happens, which is #330's rule
   applied to the steward's own hand-offs. **Item 2 above is the same defect in a
   different clothing**, and the two should be put to the retro together.
6. **The human decision blocking #1002 is unchanged and is now carried for a
   SEVENTEENTH stamp.** #1002 waits on a ruling between (a) a constitutional
   "superseded pass" ratchet class alongside `GOXSD_RATCHET_REMOVALS`, enumerated
   by case ID, and (b) holding §4.2.2's `vc:maxVersion` arm until real assertion
   evaluation lands. CLAUDE.md puts (a) beyond any agent — *"changes only via a
   human-filed issue"* — and (b) depends on **#1042**, filed and `blocked`.
   **No agent should attempt either.**
7. **M6's carve is still owed and #1042 is still its only member.** #1042
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
inherits six**: the fold-the-five-species question (#635, #912, #609, #510,
#646), the `[tests that cannot fail]` **pattern** (routed by #472's post-land,
with no filed carrier), **#849's owed steward re-ranking**, which item 5 above
carries to a dated deadline, the **band-versus-window throughput** question, the
**persona-trigger floor**, and now **item 2's carve-labelling question**. The
`gh --paginate` page-2 403 trap stays a `/retro` datum and not a filing. The CTA
cohort's 45 banked `instance` failures remain unattributed. `gate.yml` runs and
is still not a required status check, which only the repository owner can change.

**Environment, one witness each.** **`main` did NOT move under this pass** —
`2fa48cf` at the session brief and at every measurement. **The checkout was
SHALLOW and was unshallowed before any range reading** (`git fetch --unshallow
origin`, #802); the seven-branch ancestor check under Branch namespace is
impossible without it. **`gh` REST is fully available and the session brief's
premise that it is not was corrected by probe**: the GraphQL path 403s exactly as
`docs/ROUTINES.md` records, and `gh api repos/kud360/goxsd8/...` served every
read and every write. **The paginate recipe was run TWICE and needed ZERO
retries** — 15 pages each time, every non-final page full at 100, coverage
verified by distinct-number count — which is the first clean pair in this record
and is one data point against #1145's transient, not a refutation of it.
**Twenty writes across eighteen distinct issues**: **nine filed** (#1406–#1414),
**one body PATCHed** (#1354's two drifted coordinates), **one re-scoped and
relabelled** (#731, `ready` → `blocked`, body carve-banner and `## Depends on`
rewritten), and **nine thread comments** (#1142, #1003, #1144, #1232 and #672 for
the consultation; #1354 and #1356 for the citation drift; #1378 for the
`gapaudit` dead-end pattern; #731 for the carve). **Probes were run against a
binary built from the tree** — six mixed-format `validate` batches, `gen` with
and without `-out`, `gen -help`, `-version`, and the #1367/#1368 contrast — and
**two of them reversed a finding's disposition**. **No conformance measurement
was taken by this pass**: the lane table above is the committed expectations,
which `docs/WORKFLOW.md` names as the lane score (#1120). **`testdata/xsdtests`
is NOT initialized here**, which is why every #731 carve marks its case table
transcribed rather than measured.

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
unmet acceptance item was dispositioned to the existing #717, not filed.
**#1328 has now LANDED too — 2026-09-09, `51d9cf9`, `schema` +18 (13968 →
13986, fail → pass), zero regressions** — `use=`, `form=` and the two
`*FormDefault` attributes read as ·actual values· rather than raw literals, all
four being `xs:NMTOKEN` restrictions Appendix A enumerates and `cvc-datatype-valid`
stating no fallback clause. **It is the THIRD consecutive non-zero** (#931 +6,
#456 +20, now +18) and **the first member of this family to predict its flip set
EXACTLY**: banded on 24 enumerated fixtures plus ten measured shapes, it banked
the body's 18 `fail` rows to the letter and left the 6 `pass` rows the body
flagged as the risk unmoved — the census re-taken against the tree rather than
trusted from the body. **It carved NO producer-side successor.**
`effectiveDerivationSet`'s silent drop of unrecognized
`block`/`final`/`blockDefault`/`finalDefault` tokens is ruled out of scope AND
unfiled by its own Notes, on a different spec reading (`blockDefault` *"may
include values other than"* the relevant set); its arbiter's one non-blocking
observation was ruled not a STYLE T4 defect and recorded rather than filed; and
#653, whose items 3 and 4 this landing discharged, took a thread comment and a
body re-statement rather than a new issue. Live producer-decides-and-accepts
members are now **#929** alone — which #455's landing widened to own
`minOccursZero`'s clause-2.1.3 compare as well as `maxOccursZero`'s. **The family
carved a successor at
every landing from #471 through #1243, and #1242 is the first that did not** —
one reading is the tail reaching zero, the other is that this pair was narrower
than its predecessors; two more landings decide it. **#1270 was the first of
those two and came back empty; #455 is the SECOND and came back with #1328**, so
the test the last stamp set is answered — but **#1328 is itself the THIRD
CONSECUTIVE member to carve nothing**, after #931 and #456, so *"the family is
still carving"* is no longer the reading the record supports: the generator is
producing landings and lane movement, not successors, and the queue this family
feeds has narrowed to #929 alone. Note what carried the one carve there was —
#455's successor came out of an ARBITER's non-blocking scope note, not out of a
construct census, which is a different generator from the one #1270 and #1242
exhausted; #1328's arbiter produced a non-blocking observation too, and that one
was ruled not a defect, so the generator fired and came back empty. A
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

**Two more of this family landed 2026-09-09/10, and between them they closed
the two AXES rather than two more sites.** **#1369** rejects an unprefixed
attribute no Appendix A production of its element declares (`schema` **+17**),
on a hand-transcribed roster of all 49 productions; **#1380** rejects an
XSD-namespace `<schema>` child Appendix A admits at no top-level position
(**+9**), and its load-bearing finding was that **the parser charge alone banks
ZERO** — all twelve suite witnesses were declined by `conformance/schema.go`'s
top-level `default` arm, so the harness widening had to be absorbed. **What
remains after them is neither axis but the VALUES on them**: `maxOccurs="00"`
(#929), `##defined` (#606) and the eleven cases #731 carved into #1411–#1414 are
all attribute-VALUE faults, and none of them is censusable until **#1391** gives
`suiteindex` an attribute axis — which is why two landings running hand-rolled a
15,470-file corpus census instead.

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
