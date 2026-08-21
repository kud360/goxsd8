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

## Status — 2026-08-21 (backlog pass; six landings absorbed, lanes/milestones/queue and both surveys re-derived, this pass's persona findings folded, one duplicate closed)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10746 | 15615 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12983 | 2415 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**`schema` moved 12969 → 12983 (+14) and no other lane moved at all.** The
attribution is per commit and verified against the tree, not summed from
verdicts: `git show <commit>:…/schema.txt` counted across `bf3753d..7e73bfc`
puts **+2 on #867** (`c2ba631`) and **+12 on #928** (`7e73bfc`), and **zero on
#375, #382, #387 and #389** — exactly as each declared. Both moves are clean in
the strong sense: the flipped lines are `fail` → `pass` only, zero downward, and
the 15398-line case-ID column is byte-identical before and after each, so nothing
was added or removed. #867's two lines are `MS-Annotations2006-07-15/annotB001`
and `annotB005`; #928's twelve are `MS-Notations2006-07-15/notatF003`, `F007`,
`F011`, `F023`, `F025`, `F037`, `F041`, `F045`, `F049`, `F053`, `F057`, `F061`.
`instance` (10746) and `datatypes` (1161) are byte-identical across all six
commits.

**The two forecasts diverged in opposite directions, and the reason is the
census, not luck.** #867 forecast +2 and measured +2 — its issue had read all
15470 suite `.xsd` documents for the shape the CHECK tests and found exactly two.
#928's title forecast 25 and measured 12: **the parser rejects all 25**
(`TestProduceMisplacedNotationRejected` has a row per shape), and the other
thirteen never reach an executor because the schema lane's own decidability gate
declines them first. That residue is not a debt against #928's landing — it has
an owner and its own ratchet attribution as **#945**.

Landings absorbed by this stamp, all six at `bf3753d..7e73bfc`:

- **#867** (`c2ba631`) — an `<annotation>` carrying an `<annotation>` CHILD is
  rejected on annotation's OWN content model `(appinfo|documentation)*`, not on
  `xs:annotated`'s cardinality that #836 landed; the executable change is a
  NARROWED skip, so one walk now carries two faults with their two footings spelled
  apart. `fmt.Errorf`, no rule ID to mint (§3.15.3/.4/.5 all answer "None as
  such."). Arbiter ACCEPT round 1, one non-blocking finding, no repair round.
  `Ratchet: schema 12969 -> 12971 (+2)` banked at `df87efa`. **A bare claim from
  2026-08-18 taken over on the thread's clock** — see #946.
- **#375** (`d4b73f0`) — the suite-absent fatal/skip decision becomes a pure
  `unusableSuiteEnd(err, optedOut, ratcheting)` pinned by a committed four-row
  table, **including the arm CLAUDE.md's one rule rests on: opted out AND
  ratcheting still FAILS**. From 2026-08-01 to this landing the only thing standing
  between `GOXSD_SUITE_OPTIONAL=1 GOXSD_RATCHET=1` and a silent "no movement" was
  that nobody had edited the helper. Arbiter ACCEPT twice (a full second round,
  not a gate recheck, because the diff changed after the accept).
  `Ratchet: unchanged`, measured.
- **#382** (`f969577`) — 32 phantom `STYLE ` tokens across 31 sites renumbered onto
  IDs `docs/STYLE.md` actually defines; **option (b) ruled before any file was
  edited**, so STYLE.md is not opened. The executable content is SPLITTING the
  existing guard: `TestStyleCitationsNameARealRule` goes unconditional with no
  allow-list, `TestPositionalStyleCitationsAreAllowListed` keeps the list scoped to
  #540's three surviving sites. `Ratchet: unchanged`, measured — `expectations/`
  byte-identical across the commit, re-verified this pass.
- **#387** (`07117dc`) — `Element.Parent()` **deleted, not renamed** (Go rejects
  field-and-method with the same name, and `Element` already had `parent`). The
  warden's diff review **blocked its own pre-flight framing**: mason wrote what the
  pre-flight said, and the pre-flight's "for this module's own conformance harness"
  was false of `Attributes` and `BaseURI`, in the very paragraph the issue existed
  to add. `surface: +0 -1`. `Ratchet: unchanged`, measured.
- **#389** (`a2abc12`) — the repo's **first `.github/` file**. `gate.yml` runs
  CLAUDE.md's four-part gate on every `pull_request` targeting `main`, read-only
  throughout. Arbiter REJECT round 1 on prose-versus-mechanism — the file claimed
  the PR head while `actions/checkout@v4` with no `ref:` takes `refs/pull/<n>/merge`
  — repaired with `ref: ${{ github.event.pull_request.head.sha }}`.
  `Ratchet: unchanged`, measured. **Still UNENFORCED**: required-status-check is a
  repository setting no session can make, stated on the thread at land time.
- **#928** (`7e73bfc`) — an `xs:notation` the s4s grammar does not admit in its
  position is REJECTED instead of silently discarded. Legal parents are **exactly
  two** — `xs:schema` (:4562) and `xs:override` (:5577), the only content models
  referencing `xs:schemaTop`; `xs:redefine` reaches `xs:redefinable` and is not a
  third. The guard hangs off `rejectS4SFaults`, a document-wide walk **extracted**
  from `rejectRepeatedAnnotations` rather than copied. Arbiter ACCEPT in one round,
  zero findings. `Ratchet: schema 12971 -> 12983 (+12)` banked at `857e20b`.

**Every landing's follow-up ledger is disposed, and five of six had already been
disposed before this pass opened.** #387's warden hand-off → **#941**; #928's
thirteen harness declines → **#945**; #382's ledger landed on **#540**
(`5364939725`) and **#543** (`5364944142`) rather than on its own thread, and
**#657** was rewritten around the split guard; #389's cross-references landed on
#195/#315/#354/#350 with the live premise corrected in **#550**'s and **#484**'s
bodies, and its #390 conditional is recorded as never having fired. **#867's
thread carried no post-land pass and this one closed it** (`5371241379`): Next 2
(the §2.4 acknowledgement) is **#937**, filed within minutes of the landing; Next
3 (`notatF003`) is settled — #928 took it, as #908's forecast asked; Friction 1
(the takeover threshold) is **#946**, filed this pass; Friction 2 is #925's
family; the arbiter's non-blocking `produce.go:539` finding is dismissed with its
ruling recorded so a later reader does not re-derive it. **#933** was closed
`not_planned` on 2026-08-20 as a duplicate of **#862**, which owns the same three
regex fixtures and refutes the "stale fixture" framing.

**One filing was a duplicate and is closed: #943 → #565.** The #389 post-land pass
filed #943 for the missing-IMPLEMENTED-comment pattern, searching the queue for
#400 but not for #565 — which was open, `ready`, specified, and carrying the same
evidence table. #943's one genuine asset, an API comment census across all five
threads, is **folded into #565's body**, where it discharges a verification
obligation #565's own Acceptance had flagged as unmet. #565 now carries seven
first-hand rows, five of them consecutive landings. See the band, row 1.

Milestones, read from `repos/kud360/goxsd8/milestones` this pass.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 90 | 48 | **active** |
| M5 — Instance validation (XML) | 14 | 14 | **active** |
| M6–M12 | 0 | 0 | not filed |

**Five of the six landings carried NO milestone, so the rows barely moved while
`schema` gained 14.** M4 is the only row that changed: #928 closed on it (+1
closed) and #945 was filed on it (+1 open), which is why 89/48 reads 90/48. M5 is
untouched. **169 of the 231 open issues carry no milestone** (231 − 48 − 14), so
the milestone rows are feature progress and the queue paragraph below is the
queue. #867 is the sharpest instance: a parser change that moved the `schema`
lane and belongs to M4 by subject, closed with no milestone. Recorded rather than
retro-assigned — churning closed issues' milestones would break comparability
with every prior stamp for no gain a reader gets.

Queue: **231 open — 205 `ready`, 26 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 26), against **337 closed**.
205 + 26 = 231 exactly, and **every one of the 231 carries a queue label** — the
class #773/#774 fell into is empty for the seventh consecutive stamp. Both figures
were re-derived by paginating the issue list (page-numbered, not `--paginate`,
whose Link header uses numeric-ID URLs the proxy blocks), raising the page count
until two consecutive pages came back empty, deduplicating by issue number and
discarding pull requests. **The move reconciles exactly**: closed 329 → 337 is the
six landings plus #933 plus #943; open 234 → 231 is those seven pre-pass closures
against four pre-pass filings (#937, #941, #943, #945) plus this pass's #946
against this pass's one closure.

**The unblock sweep moved nothing, for the thirteenth consecutive pass, run as a
parse rather than by eye.** Every one of the 26 `blocked` bodies was fetched and
its `## Depends on` scanned for the seven just-closed issues (#867, #375, #382,
#387, #389, #928, #933): **not one names any of them**. Every live dependency line
still names an OPEN issue — #472, #250, #79, #407, #414, #455, #591, #831, #248,
#719 — and the rest are triggers (the next `/retro`, an epic target, a ruling).
**No `## Depends on` was repaired.** The one scan hit worth naming is #941's,
which reads *"none — #387 and #241 are both closed and this is startable now"*: a
discharge note, not a pending blocker.

**Three open bodies carried stale line citations and were corrected in place, not
commented at.** #669 (`README.md:78-104` → `:90-110`, `:129-136` → `:135-142`),
#625 (`:113-118` → `:119-124`) and #492 (`:114-115` → `:116-117`, `:78-119` →
`:86-142`) all pointed at README paragraphs they do not own — #625's old range
covered #492's sentence, and #492's old range stopped before two thirds of the
section a `ParseReport` mention has to fit into. `README.md` has not changed since
`2bb7133`, so all five were wrong rather than freshly stale — and the most recent
consultation comment on #669 cites a *third* wrong range (`:106-127`) for the same
snippet, so the decay is in what gets re-cited, not in the file. **A line citation
is the one thing a persona report can check that a reconfirmation cannot**, which
is most of what this pass's consultation was worth. #907's census citations were
corrected the same way (`produce.go:2820` → `:2900`, 39 call sites → 40).

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed the paginated issue JSON):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
822    wip/issue-822  127h13m0s  RETIRED  wip/issue-822: issue #822 is closed
862    wip/issue-862  main's     CLAIMED  wip/issue-862: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  93h14m0s   RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  21h27m0s   RETIRED  wip/issue-933: issue #933 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-862`, `wip/issue-872` and `wip/issue-933`, re-read this pass.
**`wip/issue-867` is gone** — the previous stamp's CLAIMED row landed and GitHub's
auto-delete took the ref, as it did for the other five landings. Nothing is
EXPIRED, no `parked/*` ref exists.

**`wip/issue-862` is a LIVE empty claim and is off-limits today.** It sits at
`c2ba631`, `main`'s own tip, so it is `ahead 0`, can never be EXPIRED, and can
never be retired on age (#722). Its thread carries a GROUNDING posted
2026-08-20T20:20Z and nothing since — under a day old, which is well inside the
~2-day clock #867's takeover used. The grounding is the asset; a session taking
#862 starts from it. **`wip/issue-933` is RETIRED and kept** — #933 closed as
#862's duplicate, and the branch is the grounding session's, `ahead 0` at the same
SHA. **`wip/issue-822` and `wip/issue-872` are RETIRED and kept**, superseded by
#851 and #878. All three deletions are a human's call, not a session's.

**The rule that settles a CLAIMED row is not written down, and this stamp files
that.** `docs/WORKFLOW.md:61-70` says an empty claim *"is never EXPIRED and never
resumable on age (#722)"* and names no successor rule;
`.claude/commands/develop.md:22-31` calls CLAIMED branches *"off-limits"* four
sentences before handing settlement to *"its issue thread's own clock"*. #867 was
taken over on a threshold neither document states, correctly and by judgment.
**#946** carries the decision, with #867 and today's `wip/issue-862` as its two
witnesses.

**`go tool gapaudit`, verbatim** (fed `--label kind/gap --state all`-shaped JSON):

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

**Group 1 is EMPTY for the seventh consecutive stamp: every marked site has an
open tracker.** 64 markers, composition unchanged from the previous stamp, and the
raw `GAP(` token count is 96 at `bf3753d`, `c2ba631` and `7e73bfc` alike — six
landings touching `parser/` and `conformance/` added no new fail-open site and
removed none. Group 2 shrank 11 → 9: **#867 and #928 closed** and dropped out,
which is the healthy direction and the first time in six stamps this list has
gone down. A marker census is still not a debt census — **#852** owns the matcher
qualification and stays below the fold because the tool again ran with
reconciliation and Group 1 empty.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 205 of
them. Take from the top. **Three rows of the previous band LANDED** (#867, #928,
#375); the band is re-cut by dropping them and re-deriving every cross-reference
by ISSUE, never by row number, which decays at each re-cut.

**The head is a process row, and the previous stamp was wrong to say the head was
empty.** The 2026-08-20 stamp read *"no `kind/process` friction is currently
compounding in consecutive LOG entries"*. Four of the five consecutive entries
that prove otherwise were already on `main` when it was written, each routing the
same absence to `/retro` as *"data, deliberately not filed from here"*, while
#565 sat open, `ready` and specified. CLAUDE.md's rule is not conditional: an
issue whose friction the log records in consecutive sessions outranks a lane slice.

The two persona rows ran a fresh 2026-08-21 consultation against the published
surface only, by the orchestrating session — a cartographer that has read the
source cannot role-play a consumer. **Both personas reconfirmed all eleven issues
and filed NOTHING NEW; cliuser's is the third consecutive such verdict and
libuser's the first.** Comments were posted on exactly three threads (#669, #625,
#492) and only because the personas' fresh line references exposed decayed
citations in those bodies; the other eight carry substantively current evidence
already (a 2026-08-19 comment on seven, a 2026-08-20 one on #896, and #934's own
2026-08-20 body), so no re-stamp was posted — the "if not already current" rule.

| # | Issue | Why here |
|---:|---|---|
| 1 | #565 | **The process row that outranks every lane slice below it, on CLAUDE.md's own rule, and it has been `ready` and specified the whole time.** No implementation account from the implementing session reaches the issue thread — the repo's stated cross-session channel. **Seven first-hand rows, FIVE of them consecutive landings** (#867, #375, #382, #387, #389), every one verified by API comment census rather than copied. The tax is now measured: on #382 the arbiter re-derived two `T7`→`T2` conversions from the diff because mason's hand-off named three of four departures from the body's formula. It was re-filed as a duplicate (#943) on 2026-08-21 by a session that did not find it, which is the leak the body names. **The streak ENDED at #928**, whose thread carries a full mason account (`issuecomment-5365393708`) naming absorbed work and three review flags that exist in no commit body and no verdict — six landings, five without and one with, which reads toward the duty being achievable rather than ceremonial. Acceptance (4) is one decision and one file; Acceptance (2)'s #319/#320/#321 premise is still unverified and must be checked first |
| 2 | #945 | **`schema` candidate up to +13, and the files are warm from a landing hours old.** The thirteen `notatF0*` cases the parser already rejects (`TestProduceMisplacedNotationRejected` proves all 25) but `schemaShapeDecidable`/`assembleCase` decline before any executor runs. Four distinct decline arms are mapped in the body **as a hypothesis, not a measurement** — acceptance 1 replaces it with a per-case `GOXSD_DECLINES=1` run. Floor is 0 with thirteen written dismissals, and the soundness rule the widening must obey is stated so it is not relitigated. `notatF035`/`notatF055` may be #404's and `notatF067` is shared with row 9 |
| 3 | #924 | **Completes #914's wrapped-cause contract while `validate/` is warm.** `cvc-complex-type` clause 4 (`defaultedAttribute`) is the ONE String Valid delegation that wraps nothing, because `xsd.ValueSpace.ValidDefault` returns `(valid, decided bool)` and holds no verdict. First step is the surface question — widening `ValidDefault` is an `xsd`+`value` interface change against `xsd`'s standing rationale for dropping the reason — so a **warden pre-flight** (#484's condition), not the keyboard. `Ratchet: unchanged`; the witness is `issuecomment-5352327544`'s four-site table |
| 4 | #909 | **The one whole complex-type representation form still declined outright, and the largest unbanded `schema` movement left in M4.** `<simpleContent>` with `<restriction>` — §3.4.2.2 cases 1-2 synthesize an anonymous simple type from the facet children — errors at `parser/produce_complex.go` and declines gate-side. 103 suite `.xsd` files carry the shape (an upper bound). Below the startable measured rows because **the sizing needs a `GOXSD_DECLINES=1` census first** — the two base arms plus facet synthesis may not be one landing, and a half-built form must not reach `main`. #846 is its ~700-line producer-coverage shadow that must edit in lockstep |
| 5 | #941 | **#387's own unfiled debt, filed hours after that landing, and the files could not be warmer.** `Element.Attributes` and `BaseURI` now stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete (the same field/method collision that forced #387's) and `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 6 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once. `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 7 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 8 | #907 | **A repo tool catalogues every `childElement` sibling-alternative site and reports which is unguarded** — four hand-written guards written across months, each after a suite case tripped over it, with `restrictionType`'s inner choice (`:4835`-`:4842`) still unguarded at `7e73bfc`. `kind/tooling`, banded below the lane rows because the tax was paid over months rather than in consecutive sessions — the discriminator row 1 satisfies and this one does not. Its census is now stale by TWO: #908's `rejectNotationContent` and #928's `rejectMisplacedNotation`, the second of which hangs off an extracted document-wide walk rather than `childElement` dispatch at all |
| 9 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced; its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. Shares `notatF067` with row 2; whichever lands first takes it and neither orders the other. Read #868's diff first (the most recent demonstration one of these declines was collateral) |
| 10 | #669 → #625 → #748 → #896 → #492 (+ #934) | **README's Library block, ONE row in the order the issues name; splitting it is why it sat.** 2026-08-21 libuser reconfirmed all six and filed nothing: **#669** the "works TODAY" snippet still fails to compile, and this pass adds a FOURTH unbuildable token — `parser.WithLogger(logger)` names an identifier the block never declares — beside the three unused-variable errors; **#625** still points at closed #203 while `xsd.Example_buildFinalizeQuery` exists and passes; **#748** still says shipped `validate.New`/`xmlsrc.Validate` "do not exist yet", with all three signature mismatches holding; **#896** the package "Contract" prose still never names `Err()`, and the disambiguating sentence lives only on the method doc; **#492** README omits `ParseReport`/`AssemblyReport`/`ReadDocument`/`Produce`; **#934** the violation example still shows `[cvc-datatype-valid]` where #913/#914 now charge `[cvc-type]`. Three bodies were citation-corrected this pass |
| 11 | #870 + #747 + #514 + #687 + #672 | **The CLI is unreachable from its own documentation; 2026-08-21 cliuser reconfirmed all five and filed nothing — third consecutive such verdict, so the gap is disclosure not discovery.** **#870** Quickstart's `go build ./...` writes no executable anywhere in the tree (re-verified here) and the stub's own `go doc` remedy fails from outside the module root; **#747** `-help` is a strict subset of `go doc` and drops the "Implemented today" paragraph a CLI-only reader most needs; **#514** a typo'd subcommand and a real unimplemented one are byte-identical stderr plus exit 2; **#687** no scoped help in any spelling, `help parse` included; **#672** `-version` in any form hits the notImplemented stub. The persona's new permutations this pass — bogus flag before `-help`, bare `help`, stream hygiene, flag position — all matched the documented contract exactly. Each issue is a sentence or a dispatch branch while the CLI surface is still empty |
| 12 | #885 | **The scope rule that has cost a reject round TWICE and still survives only in verdict comments.** #876 and #904 each shipped half a Goal because an Acceptance exemption was read as a scope exemption; #374's finding 1 added a third datum (a no-behaviour-change defect filed and landed eighteen days later). Three discriminators, one sighting each — the body says if the third will not fit the sentence, state the two-datum rule and why |

**Deliberately unbanded, and why.** **#937** (the §2.4 acknowledgement in
`rejectRepeatedAnnotations`' doc comment) is correct and `ready` but says in its
own body that it is naturally folded by the next landing touching that function —
no session need be spent solely on it. **#920** (cvc-type per-attribute charging
is unpinned) and **#921** (the `<current status>` modeling question behind the
gMonth pair) are conformance-bookkeeping follow-ups below the fold. **#929**,
**#931** and **#932** are the small parser occurrence / rule-mapping gaps #901
exposed; read each beside #901's thread. **#862** is `ready` and its grounding is
banked, but its branch is a LIVE empty claim — off-limits until the claim resolves
or #946 rules, and it is the worked example #946 asks for. **#888**, **#889**,
**#894** are the three `area/xpath` gaps still awaiting a suite census in their
range (#889 states a warden pre-flight per #484). **#843–#849** are the 2026-08-16
audit's findings, now **six open** — #847 closed `not_planned` on **2026-08-17**
and the two stamps since still counted seven; **#843** is the one whose cost of
delay climbs steeply. **#846** is #909's producer-coverage
shadow and stays unbanded only because #909 is the slice that proves the tax.
**#871** stays `blocked` on #831. **#881**, **#548**, **#622**, **#692**, **#696**,
**#796**, **#841**, **#925**, **#946** are `blocked` on the next `/retro` (or a
ruling), not on any landing — nine of the 26, and #946 is this pass's addition to
that list. **#570** carries the standing `schema` decline-count argument, still
893 as of `c116408`, now predating even more landings.

### Next planning action

**Take from the top: start at #565**, and do not read past it to a lane row. It is
`ready`, gated by nothing, one decision in one file, and five consecutive landing
entries have now reported the same friction — four of them routing it to a
`/retro` that has not run, in the words *"data, deliberately not filed from
here"*, while the issue was filed and specified the whole time. The band led with lane slices for one stamp on a reading of the
evidence that was already false when it was written; ranking this is the fix, and
it is the fix CLAUDE.md's cartographer section names by number (#527, #565).

**#945 is the warm lane follow-on** — thirteen notation cases the parser already
rejects, the residue of a landing hours old, with the measurement step written
into its own Acceptance so no session designs from the hypothesis table.
**#941 is the other warm row**, #387's own unfiled debt against files that
landed the same day, and its first move is a warden pre-flight rather than code.

**A duplicate reached the queue this pass, and the mechanism that let it through
is named rather than handed on.** #943 was filed against a defect #565 already
owned, by a session that searched the queue for one neighbour (#400) and not the
other. Both bodies carry the searchable words; what failed is that a
`/retro`-routed friction line in a LOG entry and a `ready` issue about the same
thing do not look alike. **It belongs to no open issue and is not being handed to
one** — it is recorded in #565's Notes as the sixth re-observation of a filed and
specified issue, which is where a session reading that issue will meet it, and it
is the second instance the next `/retro` has for that pattern rather than the
first. No issue filed, deliberately: a filing about not finding filings is the
shape #565 already is.

**The consultation belongs on a lane-facing schedule, not only an API-facing
one.** The 2026-08-19 personas produced #913, which moved `instance` +9409; the
2026-08-20 personas produced #934, a documentation consequence of that landing;
the 2026-08-21 personas produced nothing new and instead exposed five wrong README
line citations across three bodies, plus a sixth in the newest consultation comment
on #669. That is a real result — a reconfirmation pass whose value was entirely in
the citations it forced somebody to check against the tree — and it is the same
lesson
#909 teaches in the marker-census form: an engine that documents its own
withholding is not an engine whose withholding is tracked.

**Four unlanded corrections still target one paragraph of `docs/WORKFLOW.md`'s
filing discipline** — **#510**, **#646**, **#679**, **#912** — and whichever
lands last rebases three times. Decide one issue or four before filing a fifth.
**The CTA cohort's 45 banked `instance` failures remain unattributed**, sixth
consecutive stamp carrying it. **`gate.yml` runs but is not a required status
check**, which only the repository owner can change. All three stay open and stay
true.

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

The original carve (#167–#183) is landed. Open work is the long tail of
producer widening, finalize validity and composition edge cases —
**`<simpleContent>` with `<restriction>` (#909) is the one whole
representation form still declined outright**, and everything else in
§3.4.2 is produced. The GitHub milestone holds the feature slices; the
comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it, so the milestone is a floor on
M4's remaining work and not the whole of it.

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
+9, unioned onto #716's). `instance` stands at **10746** — #913's cvc-type
clause 3.1 landing added **9409**, itself M5 and the largest single lane move
this project has recorded — and **twenty-five** of the pre-#913 cases were not
M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure (+12), #821
added 1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). (The
*thirteen* this paragraph carried predated the CTA pair and did not sum; the
five figures do, and the sum is the number.) A slice
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

**10746 is still a floor built for soundness, and #913's +9409 jump did not
change what the number means.** The lane emits only "not valid" observations; a
violation-free `Result` DECLINES rather than passing, because `Assess` evaluates
none of `e-validity`'s other conjuncts. **Every passing case is an
expected-INVALID one by construction**, not by measurement, and the 15615 that
still fail are overwhelmingly declines rather than disagreements. The
milestone's remaining slices are what turn declines into decisions.

**Do not read 10746 as 41% of the suite passing.** It is the count of documents
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
register, escalated on #831's thread this pass.

The decline census that separated harvest candidates from indeterminates
predates #766, #715, #740, #790, #718, #716, #813 and now #913 — and is not
re-derived
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
