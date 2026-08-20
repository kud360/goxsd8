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

## Status — 2026-08-20 (backlog pass; five landings absorbed, lanes/milestones/queue and both surveys re-derived, this pass's persona findings folded)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 10746 | 15615 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12969 | 2429 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**`instance` moved 1337 → 10746 (+9409), the largest single lane movement this
project has recorded, and one landing drove all of it.** The movement is
verified against the tree, not copied: `git show <commit>:…/instance.txt`
counted per commit across `6a37d5e..bf3753d` puts the entire jump on **#913**
(`abd64bf`) and nothing on the other four — `#900` (docs), `#914`, `#908` and
`#901` each moved `instance` by zero, exactly as each declared. `schema` moved
12931 → 12969 (+38), all of it **#908** (`5f5ce54`), which is **more than the
+2 its band row forecast** — the content-model guard it landed rejects the whole
`(annotation?)`-only shape, not just the two `notatF0*` witnesses named. #913's
own ratchet-bank verdict corroborates the instance figure: 9409 lines flip
`fail` → `pass`, zero flip down, the 26361-line case-ID column byte-identical
before and after (nothing added or removed), and the instance decline set falls
25018 → 15597.

**What the jump means — read it as M5's prose says, not as "40% of the suite
passes."** The `instance` lane counts documents this engine can honestly call
not-valid; every passing case is an expected-INVALID one by construction. #913
implemented `cvc-type` (§3.3.4.4) **clause 3.1** for a simple ·governing type
definition·, so an element like `<xs:element name="amount" type="xs:decimal"/>`
carrying `12,50` — **the commonest shape in XSD authoring**, previously declined
because a simple-typed leaf's ·initial value· was never assembled — is now
charged. 10746 is a higher floor of honestly-declinable documents, not a
correctness rate: the same soundness caveat that governed 1337 governs it.

Landings absorbed by this stamp, all five at `6a37d5e..bf3753d`:

- **#900** (`e99702e`, docs/ROUTINES) — the `gh api -f body=@FILE` corruption
  (posts the literal `@/path`) vs `-F body=@FILE` (expands the file) is now
  written down, in the GitHub-channel bullet. `Ratchet: unchanged`. This pass
  wrote every GitHub body through `-F body=@FILE` and read each back
  byte-faithful — a fifth positive witness.
- **#913** (`abd64bf`) — `cvc-type` clause 3.1, above. Arbiter REJECT then ACCEPT
  (one repair round): the first attempt drove gate part 4 to 981s past the 10m
  default as a panic, traced not to a missing cache but to `regex.runeSet.addTable`
  walking `\p{C}`'s 965,096 code points one at a time; adding a `Stride==1`
  interval whole removed the cost (part 4 landed at 300.5s, below the base's own
  319.5s). `Ratchet: instance 1337 -> 10746 (+9409)` banked at `0d4c242`.
- **#914** (`4f3a24f`) — a delegating charge now carries the delegated verdict as
  a real wrapped `Err` (via `causedBy`), so `errors.Unwrap` reaches the inner
  `cvc-datatype-valid` in one hop; `xsderr.Wrap` stays used zero times in
  `validate/`. Arbiter REJECT then ACCEPT (one doc-only repair round). The review
  found a FOURTH String Valid delegation, `cvc-complex-type` clause 4
  (`defaultedAttribute`), that wraps nothing because `xsd.ValueSpace.ValidDefault`
  returns two booleans and holds no verdict — filed this pass's absorbed work as
  **#924**. `Ratchet: unchanged`.
- **#908** (`5f5ce54`) — `<xs:notation>` stops silently swallowing children; a
  plain `fmt.Errorf` on §5.1's first bullet (no rule ID to mint, §3.14.3 reads
  "None as such."). `Ratchet: schema 12931 -> 12969 (+38)` banked at `91f3919`.
- **#901** (`bf3753d`) — an EMPTY `sequence`/`all` with `maxOccurs="0"` at the top
  model-group position is charged `p-props-correct`; the fix is arm order (2.1.4
  before 2.1.2), one `if` moved. `Ratchet: unchanged` (17 of 39 candidates are
  `xsd:group ref` and take 2.1.4 either way — no witness of the flipped shape).
  The `GAP(xsd)` marker was removed, not repointed. Exposed a `maxOccurs="00"`
  elision gap, filed as **#929**.

**Each landing's follow-up ledger is disposed, and the develop sessions had
already filed most of it.** #913's test-gap → **#920**; its gMonth stale
fixtures → **#921**; its regex Unicode-vintage fixtures (`reS38`/`reT51`/`reZ004v`)
→ **#933**, filed this pass. #914's fourth delegation → **#924**. #908's
`notatF0*`-in-illegal-parent family → **#928**. #901's two pre-existing findings
→ **#931** (named-group prohibited occurrence attribute) and **#932** (lexical
occurrence-fault rule mapping); its `maxOccurs="00"` gap → **#929**. #913's live
false-reject on seven CTA documents is tracked by **#831**/#871 and escalated on
#831's thread this pass. Two items are dismissed as `/retro` material carried in
the LOG (the mechanical-check-for-corrected-sentence-copies question from #914,
and the tail-grep-retirement question from #908) — neither is an issue by its
author's own disposition.

Milestones, read from `repos/kud360/goxsd8/milestones` this pass.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 89 | 48 | **active** |
| M5 — Instance validation (XML) | 14 | 14 | **active** |
| M6–M12 | 0 | 0 | not filed |

M4 gained two closures (#908, #901) and M5 one (#913 — #914 carries no
milestone); each milestone's open count rose in step as this window's follow-up
filings landed. **172 of the 234 open issues carry no milestone** (234 − 48 − 14),
so the milestone rows are feature progress and the queue paragraph below is the
queue.

Queue: **234 open — 209 `ready`, 25 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 25), against **329 closed**.
209 + 25 = 234 exactly, and **every one of the 234 carries a queue label** —
the class #773/#774 fell into is empty for the sixth consecutive stamp. Both
figures were re-derived by paginating the issue list (page-numbered, not
`--paginate`, whose Link header uses numeric-ID URLs the proxy blocks) and
discarding pull requests, not read from a counter. The move reconciles: closed
324 → 329 is the five landings; open 228 → 234 is those five closures against
nine new filings plus this pass's #933 and #934.

**The unblock sweep moved nothing, for the twelfth consecutive pass, run as a
parse rather than by eye.** Every one of the 25 `blocked` bodies was fetched and
its `## Depends on` scanned for the five just-closed issues (#900, #908, #901,
#913, #914): **not one names any of them**. Every live dependency line still
names an OPEN issue — #472, #250, #719, #407, #414, #455, #591, #831, #248 — and
the rest are triggers (the next `/retro`, an epic target, a ruling). **No
`## Depends on` was repaired**: #505 closed and is named in #584's prose as a
grounding citation, not as a live blocker (#584's dependency is #414 alone), so
nothing there is stale-as-pending.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed the paginated issue JSON):

```
ISSUE  BRANCH         TIP AGE  VERDICT  REASON
822    wip/issue-822  unknown  RETIRED  wip/issue-822: issue #822 is closed
867    wip/issue-867  main's   CLAIMED  wip/issue-867: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  unknown  RETIRED  wip/issue-872: issue #872 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-867` and `wip/issue-872`, re-read this pass. **The namespace is
unchanged since the previous stamp** — the five landings all merged by PR squash
and GitHub's auto-delete took their `wip/*` refs. Nothing is EXPIRED, no
`parked/*` ref exists.

**`wip/issue-867` is still an EMPTY claim** (`ahead 0`, no commit of its own), so
it is never EXPIRED and never resumable on age (#722). Its GROUNDING is durable
on its own thread and is the asset — a session taking #867 starts from it.
Deliberately NOT `needs-replan`: there is no work to supersede, only a claim
never cashed. **`wip/issue-822` and `wip/issue-872` are RETIRED and kept**,
superseded by #851 and #878; their deletion is a human's call, not a session's.

**`go tool gapaudit`, verbatim** (fed `--label kind/gap --state all`-shaped JSON):

```
gapaudit: 64 GAP marker(s) across 5 area(s)
Per-area: validate 14, value 3, xml 4, xpath 6, xsd 37
Group 1 (markers with no OPEN tracker): (none)
Group 2 (OPEN kind/gap issues with no surviving marker): #398 #404 #569 #591 #592 #593 #719 #787 #867 #921 #928  (11 entries)
```

**Group 1 is EMPTY for the sixth consecutive stamp: every marked site has an
open tracker.** 64 markers, composition unchanged from the previous stamp, which
is what five landings touching parser/validate but adding no new fail-open site
should produce. Group 2 grew 10 → 11 (#928 joined as a marker-less `kind/gap`
tracker; #901's landing REMOVED its marker, closing that gap). A marker census
is still not a debt census — **#852** owns the matcher qualification and stays
below the fold because the tool again ran with reconciliation and Group 1 empty.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 209 of
them. Take from the top. **Five rows of the previous band LANDED** (#900, #913,
#914, #908, #901); the band is re-cut by dropping them and re-deriving every
cross-reference by ISSUE, never by row number, which decays at each re-cut.

The two persona rows ran a fresh 2026-08-20 consultation against the published
surface only. **cliuser reconfirmed all five and filed nothing — the third
consecutive such verdict.** **libuser reconfirmed all five AND found one new
correctness-adjacent doc defect**, filed this pass as **#934**: README's
violation-message example (`README.md:82`) still shows `[cvc-datatype-valid]` as
the outer rule, but since #913/#914 a simple-typed leaf is charged `[cvc-type]`
wrapping it, so a reader keying a dispatcher on the example misses every
simple-content lexical failure. The nine unchanged reconfirmations are
substantively identical to the 2026-08-19 evidence already on their threads, so
no re-stamp was posted (the "if not already current" rule); the one substantive
update — that #914's wrapped-cause change is orthogonal to #896's `Err()`-is-nil
defect — is posted on #896.

| # | Issue | Why here |
|---:|---|---|
| 1 | #867 | **`schema` +2, measured, oracle round already paid.** An `annotation` carrying `annotation` CHILDREN is still accepted (annotation's own content model, not `xs:annotated`'s cardinality which #836 landed); `annotB001`/`annotB005` are the suite's only two, both `invalid`, both `fail`. The GROUNDING is done on this issue's own thread and `wip/issue-867` is `ahead 0`, so a session starts from the grounding and pushes the branch's first commit |
| 2 | #928 | **`schema` candidate, and the same notation family #908 just worked while the file is warm.** An `xs:notation` nested where the s4s grammar does not admit it is silently discarded — 25 suite cases, one per illegal parent, no producer ever visits the element. Needs the general per-element child guard; #908 built the sibling guard (notation's OWN children) and this is the mirror. Below #867 only because its magnitude is a census, not a banked number |
| 3 | #924 | **Completes #914's wrapped-cause contract while `validate/` is warm.** `cvc-complex-type` clause 4 (`defaultedAttribute`) is the ONE String Valid delegation that wraps nothing, because `xsd.ValueSpace.ValidDefault` returns `(valid, decided bool)` and holds no verdict. First step is the surface question — widening `ValidDefault` is an `xsd`+`value` interface change against `xsd`'s standing rationale for dropping the reason — so a **warden pre-flight** (#484's condition), not the keyboard. `Ratchet: unchanged`; the witness is `issuecomment-5352327544`'s four-site table |
| 4 | #909 | **The one whole complex-type representation form still declined outright, and the largest unbanded `schema` movement left in M4.** `<simpleContent>` with `<restriction>` — §3.4.2.2 cases 1-2 synthesize an anonymous simple type from the facet children — errors at `parser/produce_complex.go` and declines gate-side. 103 suite `.xsd` files carry the shape (an upper bound). Below the startable measured rows because **the sizing needs a `GOXSD_DECLINES=1` census first** — the two base arms plus facet synthesis may not be one landing, and a half-built form must not reach `main`. #846 is its ~700-line producer-coverage shadow that must edit in lockstep |
| 5 | #375 | **On #374's evidence, files warm.** Carries the other two of #309's three non-blocking findings, so taking it closes that landing's ledger. Subject is the guard on CLAUDE.md's one rule — that a `GOXSD_RATCHET=1` run over an absent suite can never report "no movement". Rebase and re-read `conformance_test.go`/`runner_test.go` (both touched by #374) before designing; nothing gates it |
| 6 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once. `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 7 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 8 | #907 | **A repo tool catalogues every `childElement` sibling-alternative site and reports which is unguarded** — four hand-written guards written across months, each after a suite case tripped over it, with `restrictionType`'s inner choice still unguarded. `kind/tooling`, banded below the lane rows because the tax was paid four times over months, not in consecutive sessions. #928 (row 2) and #908's exposed `notatF0*` family are fresh witnesses its census should reproduce — the census is one short |
| 9 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced; its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. Read #868's diff first (the most recent demonstration one of these declines was collateral) |
| 10 | #669 → #625 → #748 → #896 → #492 (+ #934) | **README's Library block, ONE row in the order the issues name; splitting it is why it sat.** 2026-08-20 libuser reconfirmed all five, corrected none: **#669** the "works TODAY" snippet still fails to compile; **#625** still points at closed #203 for its worked example; **#748** still says shipped `validate.New`/`xmlsrc.Validate` "do not exist yet" (fourth consecutive consultation to lead with it); **#896** `res.Err()` is nil while `res.Violations()` carries the charge, and #914's landing did NOT fix it (orthogonal); **#492** README omits `parser.ParseReport`. **#934** joins here — the violation-message example now shows the wrong outer rule ID after #913/#914 |
| 11 | #870 + #747 + #514 + #687 + #672 | **The CLI is unreachable from its own documentation; 2026-08-20 cliuser reconfirmed all five and filed nothing — third consecutive such verdict, so the gap is disclosure not discovery.** **#870** Quickstart's `go build` writes no binary and the stub's own `go doc` remedy fails where a binary runs; **#747** `-help` output is a strict subset of `go doc` and drops the status paragraph; **#514** a typo and a real subcommand are byte-identical stderr+exit 2; **#687** no scoped help in any spelling; **#672** `-version` in any form hits the notImplemented stub. Each is a sentence or a dispatch branch while the CLI surface is still empty |
| 12 | #885 | **The scope rule that has cost a reject round TWICE and still survives only in verdict comments.** #876 and #904 each shipped half a Goal because an Acceptance exemption was read as a scope exemption; #374's finding 1 added a third datum (a no-behaviour-change defect filed and landed eighteen days later). Three discriminators, one sighting each — the body says if the third will not fit the sentence, state the two-datum rule and why |

**Deliberately unbanded, and why.** **#920** (cvc-type per-attribute charging is
unpinned) and **#933** (three regex Unicode-vintage fixtures) and **#921** (the
`<current status>` modeling question behind the gMonth pair) are this window's
conformance-bookkeeping and test-hardening follow-ups — correct, `ready`, below
the fold. **#929**, **#931** and **#932** are the small parser occurrence /
rule-mapping gaps #901 exposed; read each beside #901's thread. **#888**,
**#889**, **#894** are the three `area/xpath` gaps still awaiting a suite census
in their range (#889 states a warden pre-flight per #484). **#843–#849** are the
2026-08-16 audit's seven findings; **#843** is the one whose cost of delay
climbs steeply. **#846** is #909's producer-coverage shadow and stays unbanded
only because #909 is the slice that proves the tax. **#871** stays `blocked` on
#831. **#881**, **#548**, **#622**, **#692**, **#696**, **#796**, **#841**,
**#925** are `blocked` on the next `/retro` (or a ruling), not on any landing.
**#570** carries the standing `schema` decline-count argument, still 893 as of
`c116408`, now predating even more landings.

### Next planning action

**Take from the top: start at #867** — `schema` +2, its grounding already banked
on the thread, `wip/issue-867` confirmed `ahead 0`, and the same
annotation/notation family #908 and #909's neighbourhood work in. **#928 is the
warm follow-on** (25 notation cases, the mirror of the guard #908 just built),
and **#924 completes #914's wrapped-cause contract** while `validate/` is warm —
but #924's first move is a warden pre-flight on the `ValidDefault` surface, not
code.

**The band's process-over-lane head is empty for the first time in several
stamps.** #659 and #900 both landed; no `kind/process` friction is currently
compounding in consecutive LOG entries, so this band leads with measured lane
slices (#867, #928) rather than a process row. If the next consultation or retro
surfaces recurring friction, it re-earns the head (#527, #565) — the rule is
unchanged, the queue is momentarily clear.

**The consultation belongs on a lane-facing schedule, not only an API-facing
one.** The 2026-08-19 personas produced #913 — a silent false-accept that no
internal survey found, because `gapaudit` cannot see a gap with no `GAP(`
marker, the lane declined the whole shape rather than failing it, and
`validate/assess.go`'s own doc stated the gap 400 words in where it reads as
scope. #913 has now landed and moved the lane +9409. The 2026-08-20 personas
produced #934, a documentation consequence of that same landing. "The engine
documents its own withholding" is not "the withholding is tracked" — the lesson
#909 also teaches in the marker-census form.

**Four unlanded corrections still target one paragraph of `docs/WORKFLOW.md`'s
filing discipline** — **#510**, **#646**, **#679**, **#912** — and whichever
lands last rebases three times. Decide one issue or four before filing a fifth.
**The CTA cohort's 45 banked `instance` failures remain unattributed**, fifth
consecutive stamp carrying it. Both stay open and stay true.

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
