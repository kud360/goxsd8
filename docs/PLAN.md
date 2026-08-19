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

## Status — 2026-08-19 (post-land pass for #374, with the libuser and cliuser consultations folded)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1337 | 25024 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12931 | 2467 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**Every lane is unchanged, and that is a verified fact rather than a copied
table.** `git diff e8d99e3..HEAD -- conformance/testdata/expectations/` is empty:
the six files are byte-identical to the previous stamp's commit. The one landing
in between declared `Ratchet: unchanged` and it held. The three-consecutive
lane-moving run the previous stamp opened (`schema` 12913 → 12931 across #883,
#868, #904) therefore **stands at three and did not extend** — this landing was
not a lane slice and did not claim to be.

Landing absorbed by this stamp:

- **#374** at `57b4a75` — twenty `t.Skipf` bodies interpolated `suiteRoot`
  (`"../" + suiteModulePath`), printing `git submodule update --init
  ../testdata/xsdtests`, a package-relative path that fails from where any reader
  of a `go test ./...` skip line stands. All twenty collapse into one
  `skipWithoutSuite` helper over one `suiteAbsentSkipMsg` const, pinned by
  `TestSuiteAbsentSkipNamesModuleRootPath`. **The landing deletes more lines than
  it adds**; the defect was duplication and the wrong string was its most visible
  symptom. Arbiter **ACCEPT in one round**, two non-blocking findings, both
  disposed by this pass. `Ratchet: unchanged`, verified above.

**This landing was not in the previous stamp's band, and the previous stamp's
band was not taken.** That stamp closed *"start at row 1 (#659)"*; #374 appears
nowhere in it — not as a row, not in the unbanded paragraph. Nothing on the
record says why, and this pass does not speculate. It is stated here because the
ordering is this pass's own deliverable, and a deliverable nothing consumed is a
fact the next stamp needs. See the Next planning action.

Milestones, read from GitHub this pass.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 87 | 46 | **active** |
| M5 — Instance validation (XML) | 13 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**Every milestone counter now agrees with the issue list, for the first time in
four stamps — M3's drift is REPAIRED, not carried again.** The GitHub counter had
said `open_issues: 1` for three consecutive stamps (`f17da28`, `4ec8870`,
`e8d99e3`), each recording the row as right and the counter as wrong. The cause
those stamps named — #875 reassigned M3 → M4 without a decrement — is carried as
their hypothesis and was **not** re-derived here; only the repair was. It
is fixed by a **milestone round trip on any issue**: assigning #912 to M3 left
the counter at 1, removing it again dropped it to 0, and GitHub's stored count
now matches the twelve closed and zero open items the paginated issue list
holds. Recorded because a future drift is repairable the same way, not because
the number moved — the row never changed.

M4 is 87/46, unchanged. **M5 moved 12 → 13 open**, by this pass filing #913 into
it — the only milestone assignment this pass made, and the only milestone row
that moved. Both figures were re-read from `repos/kud360/goxsd8/milestones` and
independently re-derived by grouping the paginated issue list on
`.milestone.title`; the two derivations agree on every row, and M3's repaired
counter held through three further issue writes.

Queue: **229 open issues — 205 `ready`, 24 `blocked`, 0 `needs-replan`,
2 `epic`** (both `blocked`, so both counted inside the 24), against
**323 closed**. 205 + 24 = 229 exactly, and **every one of the 229 carries a
queue label** — the class #773 and #774 fell into is empty for the fourth
consecutive stamp. Read the milestone table as feature progress and not as the
queue: **170** of the 229 carry no milestone (229 − 46 − 13).

**The move reconciles exactly: 227 − 1 + 3 = 229.** The closure is #374; closed
moved 322 → 323 by the same one. The three filings are **#912**, out of this
landing's own follow-up list, and **#913** and **#914**, both out of the libuser
consultation and both reproduced against the tree by this pass before filing —
see the band.

**The unblock sweep moved nothing, for the tenth consecutive pass, and it was
run as a parse rather than by eye.** All open bodies were matched for `#374`:
six mention it — #912, #885, #659, #591, #400, #375, the first three of them
this pass's own writes — and **not one names it as a live dependency**; #659 and
#375 list it as related and already landed. All 24 `blocked` bodies were then
re-read for their
`## Depends on`: **fourteen name at least one still-open issue** and **ten name a
trigger** rather than an issue — a `/retro` pass, a ruling, an epic reaching
zero. That split is 14/10 where the previous stamp recorded 16/8. **No label
changed and no dependency closed; the difference is classification, not
movement** — #548 and #79 both say in their own words that they name a trigger
and that the unblock sweep must not re-scan them, and this pass counts them that
way. #779's script is the thing that would stop the two passes disagreeing.

**Seven `## Depends on` sections were repaired this pass and none of the repairs
unblocked anything.** #715 closed 2026-08-14 and was still named live by #717
(`blocked`, also on #248 — stays `blocked`), by #720 (`blocked`, also on #472 —
stays `blocked`) and by #719 (`ready`, where it was the whole section); #766
closed 2026-08-14 and was still the whole section on #773 and #774, both
`ready`. All five are now struck and marked DISCHARGED. **#743 and #744 had no
`## Depends on` section at all** — the only two in the queue, found by parsing
every open body — and both now carry one reading `none`, with the adjacency their
Notes already stated. **All 229 open bodies now carry the section**, re-verified
after every write this pass made.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim**, fed from #840's recipe:

```
ISSUE  BRANCH         TIP AGE   VERDICT  REASON
822    wip/issue-822  79h10m0s  RETIRED  wip/issue-822: issue #822 is closed
867    wip/issue-867  main's    CLAIMED  wip/issue-867: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  45h11m0s  RETIRED  wip/issue-872: issue #872 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-867` and `wip/issue-872`. Nothing is EXPIRED and there is no
`parked/*` ref. **The namespace is unchanged since the previous stamp** — no ref
was added, removed or moved; #374 landed by PR squash-merge and GitHub's
auto-delete took its branch.

**The three verdicts were confirmed against GitHub's own compare, not inferred.**
`repos/kud360/goxsd8/compare/main...<branch>` reports `wip/issue-822` **ahead 3,
behind 31**, `wip/issue-872` **ahead 2, behind 15**, and `wip/issue-867`
**ahead 0, behind 7**. The previous stamp reached the same three answers by
reading `main`, on the ground that the shallow clone (#802) forbids ahead/behind
arithmetic. It forbids it **locally**; the compare endpoint computes it
server-side against the full history and is available to every session.

**`wip/issue-867` is still an EMPTY claim, 22 hours old at this stamp**, and `ahead 0` is
the mechanical proof: it has pushed nothing of its own, so it is never EXPIRED
and never resumable on age (#722). Its thread still carries exactly one comment,
the completed GROUNDING at 2026-08-18T16:18:06Z. **That grounding is durable and
is the asset** — a session taking #867 starts from it instead of re-paying an
oracle round. Deliberately NOT `needs-replan`: there is no work to supersede,
only a claim that was never cashed.

**`wip/issue-822` @ `cc2d54e` and `wip/issue-872` @ `0b34c21` are both RETIRED
and both deliberately kept**, superseded by #851 and #878, their content
confirmed absent from `main` by the compare above. They are never force-pushed,
never renamed, never a base to branch from, and **their deletion is a human's
call, not a session's**.

**`go tool gapaudit`: 64 `GAP(` markers across 5 areas** — `xsd` 37,
`validate` 14, `xpath` 6, `xml` 4, `value` 3 — run **with reconciliation**, not
census-only. **Group 1 is EMPTY for the fourth consecutive stamp: every marker
in the tree has an open tracking issue.** Total, composition and Group 2 (ten
issues) are all unchanged, which is what a `conformance`-test-only landing that
minted no rule ID and left no marker should produce.

**A marker census is still not a debt census**, and the previous stamp's finding
stands unchanged: #909 is a whole declined representation form carrying no
`GAP(` marker, which `gapaudit` could not see and never could. Treat "Group 1
empty" as "every *marked* site is tracked". **#852** owns the matcher and is
where that qualification gets built or written down.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 203 of
them. Take from the top. **Row 1 of the previous band did not land and nor did
row 2** — the landing was #374, unbanded — so the band is carried rather than
re-cut, with **three new rows**: #375 on this landing's evidence, and #913 and
#914 on the libuser consultation's. Nothing was promoted out of order and nothing
was dropped; every previous row shifts down by the number of new rows above it.

**Rows 14 and 15 are the persona rows and they are FINAL, not pending.** Both
consultations ran on 2026-08-19 against the published surface only, and their
verdicts are folded into the rows and onto all ten threads. The headline is that
**all ten issues were reconfirmed and not one body needed a correction** — every
premise, file citation and quoted line was re-read against `main@57b4a75` this
pass and every one holds. **The gap is disclosure, not discovery, on both
sides** — which is what the previous stamp said about the CLI half and is now
measured for the library half too.

**Rows 1 and 2 are `kind/process` and they outrank the measured lane slices on
purpose** (#527, #565): friction the log records in consecutive landings
compounds every pass, while the fix is one session.

| # | Issue | Why here |
|---:|---|---|
| 1 | #659 | **Paid again by #374 — 22 landings on record, 11 of them re-derivable by grep — and #374's arbiter paid it before a gate that was ABOUT that command's spelling.** A fresh checkout starts with `testdata/xsdtests` unpopulated and the gate fails on the missing-suite guard (#309). Its body was repaired this pass and is now materially easier to work: the hand-maintained landing table and the running count in its title are gone, replaced by the two greps over `docs/LOG/*.md` that re-derive both (**11** entries record paying it, **1** records it already present), and the #842 row is struck because that entry records no submodule friction at all. **#374 also shrank the remaining work**: the message a payer reads is now correct, so what is left is exactly this issue's own claim — make the step known BEFORE the gate is run, in `.claude/agents/arbiter.md`, the document the dominant payer reads. One line, one session; the escape hatch (*close it with the finding if the only real fix is outside the repo*) is still open and still untaken |
| 2 | #900 | **`gh api -f body=@file` posts the literal string `@/path`; `-F` is the flag that expands a file, and no document says so.** Not paid by #374 and not paid by this pass — because this pass had read the issue. It now has a **positive** fourth witness instead of a corruption: twelve writes through `-F body=@file` (eight body rewrites, three comments, one issue creation), every one read back and diffed against the file on disk, every one byte-faithful through backticks, `~~strikethrough~~`, angle-bracketed tokens and fenced code. That settles its acceptance item 2 with measurement rather than argument. The corruption remains silent, on the channel `docs/WORKFLOW.md` names as one of the three durable things; the deliverable is prose in one file and the body already forbids a wrapper |
| 3 | #913 | **NEW, and it is a silent FALSE ACCEPT on the commonest idiom in XSD authoring.** An element whose ·governing type definition· is a **simple type** is never checked: `<xs:element name="amount" type="xs:decimal"/>` containing `12,50` returns `Err() == nil` and **zero violations**, byte-for-byte indistinguishable from a valid document, while the identical lexical wrapped in `complexType/simpleContent/extension` IS charged. Reproduced this pass against `main@57b4a75` in a throwaway module — leaf `xs:decimal` and leaf `xs:int` both, plus the wrapped control. `cvc-type` (§3.3.4.4) **clause 3.1** is unimplemented: `governance.complexType()` (`validate/assess.go:150-161`) returns nil for a simple governing type and `walk.text` (`:719-727`) discards every run, so the ·initial value· is never assembled. `value.ValidateLexical` is called from exactly two places in `validate/`, neither on this path. **Ranked above two measured rows on SEVERITY, not on magnitude** — a departure from this band's usual rule that the next pass should check: `instance` moves up (`conformance/instance.go:176-195` declines the whole shape today, so the direction cannot be down) but the magnitude needs a `GOXSD_DECLINES=1` census, which is the session's first step |
| 4 | #914 | **Two format verbs, `Ratchet: unchanged`, and it is placed here for the coupling rather than its size.** Every `validate` charge flattens its cause into `Msg`: `xsderr.Wrap` is used **zero** times in the package (8 uses tree-wide — 2 `parser`, 1 `parser/xmltree`, 2 `value`), so a `cvc-complex-type` violation over an invalid decimal has `viol.Err == nil` and `errors.Unwrap(viol) == nil`, with the inner `cvc-datatype-valid` rule and lexical reachable only by regex-scraping the message. Reproduced this pass. **Take it before row 3**: #913 adds a third charge site, and settling the spelling first removes a rebase. It also repairs an assumption **#486** already makes — that Acceptance says *"`Unwrap`/`errors.Is` covers `Err`"*, true in `parser`/`xsd`/`value` and false in the package a library consumer actually receives errors from |
| 5 | #867 | **`schema` +2, measured, and the oracle round is already paid.** An `annotation` carrying `annotation` CHILDREN is still accepted — `annotation`'s own content model (`:5755`), not `xs:annotated`'s cardinality, which is what #836 landed. `annotB001` and `annotB005` are the suite's only two such documents, both `invalid`, both `fail`, neither declined. **The GROUNDING is done and on this issue's own thread**; `wip/issue-867` is confirmed `ahead 0` this pass, so a session starts from the grounding and pushes the first commit that branch will carry |
| 6 | #908 | **`schema` +2, decided, and the conformance gate needs no change.** `produceNotation` (`parser/produce.go:2383`-`:2396`) reads three attributes and never touches `Children()`, so every child under an `<xs:notation>` is ignored. `xs:notation` (`:5696`) extends `xs:annotated` (`:4426`), whose content is `<xs:annotation>?` and nothing else; **§3.14.3 reads "None as such."**, so it is #884's shape exactly — a plain `fmt.Errorf` on §5.1's first bullet, no rule ID to mint. `notatF018` and `notatF066` are both suite-`invalid`, both `fail`, and both admitted unconditionally by `conformance/schema.go:618`. Same family as row 3 and cheap to take beside it |
| 7 | #901 | **#883's own follow-up, and the file is still warm.** §3.4.2.3.3 clause 2.1.2 answers before 2.1.4, so an EMPTY `sequence`/`all` carrying `maxOccurs="0"` at the TOP model-group position escapes `p-props-correct` while the identical element one level down is charged. `Ratchet:` **unmeasured and expected unchanged** — a census during #883's grounding found no witness among 39 candidates — which is why it sits below the measured rows. It owns the one `GAP(xsd)` marker #883 landed, whose text carries a typo to fix rather than carry forward |
| 8 | #909 | **The last complex-type representation the producer declines outright, and the largest unbanded `schema` movement in M4.** `<simpleContent>` with `<restriction>` — §3.4.2.2 cases 1-2 synthesize an anonymous simple type from the restriction's facet children — errors at `parser/produce_complex.go:586` and is declined gate-side at `conformance/schema.go:980`. **103 suite `.xsd` files carry the shape**, which is an upper bound and not a forecast. It sits below the measured rows for exactly that reason: **census the decline set (`GOXSD_DECLINES=1`) and settle the sizing before starting**, because the two base arms plus the facet synthesis may not be one landing, and a half-built form must not reach `main` |
| 9 | #375 | **NEW to the band, on #374's evidence: the files are warm and the overlap is now known.** It carries the other two of #309's three non-blocking findings, so taking it closes that landing's ledger entirely. Its subject is the guard on CLAUDE.md's one rule — that a `GOXSD_RATCHET=1` run over an absent suite can never report "no movement" — and today that three-way policy is pinned only by an env matrix an arbiter ran by hand on #309, with `TestConformance` correct solely because its callee always halts the goroutine. Its body was corrected this pass: it claimed *"no file overlap, either order"* with #374, and #374 in fact touched both `conformance_test.go` and `runner_test.go`. Rebase and re-read both before designing; nothing is blocked, because `endUnusableSuiteRun` and `checkSuitePresent` are untouched |
| 10 | #907 | **Four hand-written guards against one mechanism, across 39 `childElement` call sites, each written only after a suite case tripped over it.** `rejectRepeatedAnnotations` (#836), `rejectBothInlineTypes` (#340), and #904's two. The catalogue this asks for reproduces all four and the one the #904 grounding names as still unguarded (`restrictionType`'s inner choice, `:4835`-`:4842`). **Banded below the lane rows and not with rows 1-2, deliberately:** the tax here was paid four times over months, not in consecutive sessions, and #904 has just guarded the three hottest sites — real debt on a cooling seam, not compounding friction. It carries an escape hatch to close with the catalogue and no tool |
| 11 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in **three** rules at once — the ·initial value· charge, the ID/IDREF binding and the ·key-sequence· member. `instance` candidate, unmeasured, direction can only be up because all three decline today. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type. It sits below row 7 for that reason — row 7 is startable at the keyboard |
| 12 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced, and its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. Same function family as #830, #836, #867 and #904, all landed or banded; read those diffs first, and #868's in particular, which is the most recent demonstration that one of these declines was collateral rather than forced |
| 13 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. It **gates #56**, and it decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Its `## Depends on` was repaired this pass: it named #715, closed 2026-08-14, as its whole dependency section. **Nothing gates it; it is startable on `main` today** |
| 14 | #669 → #625 → #748 → #896 → #492 | **The 2026-08-19 libuser consultation reconfirmed all five and corrected none — every premise in every body was re-read against `main@57b4a75` this pass and holds.** README's Library block, ONE row in the order the issues themselves name; splitting it across band rows is why it sat seven passes, and the five overlap by paragraph rather than partition cleanly, so whichever lands second rebases on the first. Per-issue, as verified: **#669** — the headline "works TODAY" snippet was pasted verbatim and built, still failing with `declared and not used: builtins`, `err`, `schema`. **#625** — `README.md:122-124` still points a reader at #203 for the worked example; `gh api` says #203 is `closed, completed`, so it names a closed issue as pending work that has in fact shipped. **#748** — `README.md:127` still reads *"`validate.New` / `xmlsrc.Validate` (M5) do not exist yet"*; the persona compiled and ran both. **THIRD consecutive consultation to lead with it**, and still open 24 hours after filing. **#896** — reproduced directly: `res.Err() == nil` while `res.Violations()` carried a real `cvc-complex-type` charge, and the one sentence that prevents the mistake surfaces only under `go doc … Result.Err`, never on the `go doc ./validate` landing page. **#492** — README names `parser.Parse` alone while `go doc ./parser` documents it as *"ParseReport without the report"*, so the published docs themselves identify the primitive README omits. **Row 3 (#913) came out of this same consultation** and is the reason this row matters more than it did: a corrected README sends the reader to a validator that silently passes the idiom the README's own example is about |
| 15 | #870 + #747 + #514 + #687 + #672 | **The 2026-08-19 cliuser consultation reconfirmed all five and filed nothing new — second consecutive consultation to return that verdict, which is itself the result: the gap is disclosure, not discovery.** All five decided BEFORE #472. **#870** is the one a user hits first, and this pass is where its second half stopped being an argument: Quickstart's `go build ./...` exits 0 and writes no binary anywhere (cwd, `$(go env GOPATH)/bin`, every PATH entry), and the persona then copied the built binary to a directory with no module and ran the stub's own remedy from there — `go doc github.com/kud360/goxsd8/cmd/goxsd8` fails with *"no required module provides package … go.mod file not found"*, at exactly the location that message is read. **#747** — `-help` and bare-invocation stdout is a strict **subset** of `go doc -all ./cmd/goxsd8`, verified byte-for-byte; the *"Implemented today"* paragraph exists only in the text a binary user never sees. **#514** — `./goxsd8 validate -schema foo.xsd bar.xml` and `./goxsd8 bogus-subcommand` produce **byte-identical stderr and identical exit 2**. **#687** — `validate -help`, `-help validate` and `validate -h` all print the full generic usage; scoped help exists in no spelling. **#672** — `-version`, `version` and `--version` all land in the same generic bucket, indistinguishable from a typo, and README never mentions `-version` at all; it compounds #514 and shares its dispatch branch. For balance the persona confirmed one thing works exactly as contracted: `-h`/`-help`/`--help` in any position and bare invocation print full usage to stdout and exit 0. Each remaining item is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |
| 16 | #885 | **The scope rule that has cost a reject round TWICE, and it still survives only in verdict comments.** #876's round 1 classified a surviving hole as out-of-scope pre-existing and the arbiter had to re-derive that it was not; #904's round 1 read an Acceptance exemption from the *ratchet figure* as an exemption from *scope* and shipped half the Goal. **A third datum was added to its thread this pass and its body now points at it**: #374's finding 1 was a one-string, no-behaviour-change defect raised by an arbiter, filed, and landed **eighteen days later** — the same axis asked of the accepting round rather than the implementing session. Three discriminators, one sighting each, one issue; the body says explicitly that if the third will not fit the sentence, state the two-datum rule and say why |

**Deliberately unbanded, and why.** **#912** is this pass's own filing and is
below the fold on its own evidence: it cost #374's session nothing — the
grep-shaped Acceptance absorbed the stale census — so it is a class worth a rule
and not friction that compounds, which is the discriminator rows 1-2 use.
**#888**, **#889** and **#894** are the three `area/xpath` gaps #858 and #886
filed; correct, startable and still below the fold, because **no census has been
taken of what the suite holds in their range**, and #889 states a warden
pre-flight as its first step, which is **#484**'s standing condition. **#444** is
annotated but not banded: #904's grounding answers its question 2 in the general
form, which shrinks its oracle round without closing it. **#871** stays `blocked`
on **#831**. **#881** is `blocked` on the next `/retro`. **#875** is #706's own
follow-up and **#884** is #876's and #883's neighbourhood — read each beside its
parent. **#843–#849** are the 2026-08-16 architecture audit's seven findings;
**#843** is still the one whose cost of delay is stated as increasing steeply.
**#846** is the root cause under row 8 — the ~700-line hand-maintained shadow of
producer coverage that #909 must edit in lockstep — and it stays unbanded only
because #909 is the slice that proves the tax rather than the one that removes
it. **#852** is `gapaudit`'s matcher, out of the band for the fourth consecutive
stamp because the tool again ran with reconciliation and Group 1 empty.
**#744**, **#773** and **#721** are still held out on one shared condition, and
**#484** owns it — #744's and #773's bodies were repaired this pass, which
changes their readability and not their ranking.

### Next planning action

**Settle what the band is for, because this stamp cannot tell whether it works.**
The previous stamp ended *"start at row 1 (#659)"* and named rows 3 and 4 as the
lane-moving alternatives; the landing was **#374, which appears nowhere in that
section**. That is not a complaint about the choice — #374 was a real defect, it
landed clean in one arbiter round, and it shrank row 1's remaining work. It is
that **the ordering is this pass's deliverable and nothing on the record says
whether it was read.** Either a landing that departs from the band says so in one
line on its own thread, or the band stops claiming to be the queue's entry point
and becomes what it can prove it is: a shortlist with reasons attached. **#400**
already owns the neighbouring defect — a post-land pass leaving no signal on
`main` — and is the place to decide this without a new issue.

**Four unlanded corrections now target one paragraph of `docs/WORKFLOW.md`'s
filing discipline, and a fifth will collide with them.** **#510** (grep the suite
before an Acceptance asserts "no case reaches X"), **#646** (no absolute banked
ratchet figure in a body), **#679** (enumerate a suite-coverage grep's constructs
from Appendix A) and **#912** (state the criterion that re-derives a site census,
not the list). Each is a sentence, each is `ready`, each states a rule the other
three do not, and **whichever lands last rebases three times.** Decide one issue
or four before filing a fifth — and this pass deliberately did not merge them
itself, having just filed one of the four.

**#570 is carried unadvanced, and the reason is honest: this pass ran no suite
census at all.** The previous stamp made census-as-a-tool the next action on the
strength of four throwaway censuses in one pass. A `conformance`-test-only
landing needed none, so the argument gained no evidence and lost none. The
standing `schema` decline count is still **893** as of `c116408`, still not
re-derived, and now predates **13 landings** (`docs/LOG/2026-08.md` entries after
#836's; `git log c116408..HEAD` is 17 commits) — by a wide margin the oldest
measurement this plan argues from. The previous stamp put that figure at
twenty-four, which nothing in the tree reproduces; it is corrected here rather
than carried. Row 6 (#909) remains the sharpest case for
it: **its Acceptance cannot be written without a decline census.**

**The CTA cohort's 45 banked `instance` failures are still unattributed.** Third
consecutive stamp carrying it; no landing has touched it, nothing on record says
why any of the 45 fails. It stays open and stays true.

**The personas found a CORRECTNESS defect this time, not a documentation one, and
that changes what the consultation is for.** Ten consultations across three
passes have produced reconfirmations and doc issues; this one produced **#913** —
`cvc-type` clause 3.1 unimplemented, so a leaf simple-typed element is never
checked and an invalid document is indistinguishable from a valid one. **No
internal survey found it**, and the reason is instructive: `gapaudit` cannot see
it (no `GAP(` marker), the conformance lane declines the whole shape rather than
failing it, and `validate/assess.go`'s own doc comment states the gap 400+ words
in, where it reads as scope rather than as a defect. **A persona working only
from the published surface hit it in one session.** The planning consequence is
that the consultation belongs on a lane-facing schedule and not only on an
API-facing one — and that **"the engine documents its own withholding" is not
the same as "the withholding is tracked"**, which is the same lesson #909 taught
this band one stamp ago in the marker-census form.

**The queue is 229 and the band is sixteen rows, and the gap is not a backlog
problem.** `ready` means filed and unblocked; its size is an output and never a
target (#347). Every one of the 229 carries a queue label for the fourth
consecutive stamp, and every one now carries a `## Depends on` section as well.

Take from the top: **start at row 1 (#659)** — one line in one agent file, its
body cleaned this pass, and #374 both paid its tax and removed half its work.
**If the next session takes correctness over process, start at row 3 (#913)** —
it is a silent false accept on the idiom README's own example uses, reproduced
rather than argued, and rows 3 and 4 are one session together in either order.
It is not the only unimplemented check in the queue — row 11 (#853) is another
and `validate/assess.go`'s doc comment lists more — but it is the one whose
declined shape is the commonest thing anybody writes in a schema. If it must move
a measured number instead, **row 5 (#867)** is `schema` +2 with its grounding
already banked on the thread and **row 6 (#908)** is another +2 beside it.

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
+9, unioned onto #716's). `instance` stands at **1337**, and **twenty-five** of those
cases are not M5's: **landings outside this milestone keep moving this lane** —
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

**1337 is still a floor built for soundness, and no jump has changed what the
number means.** The lane emits only "not valid" observations; a violation-free
`Result` DECLINES rather than passing, because `Assess` evaluates none of
`e-validity`'s other conjuncts. **Every passing case is an expected-INVALID one
by construction**, not by measurement, and the 25024 that still fail are
overwhelmingly declines rather than disagreements. The milestone's remaining
slices are what turn declines into decisions.

**Do not read 1337 as 5% of the suite passing.** It is the count of documents
this engine can honestly call not-valid, and it grew because the same rules now
reach every node instead of one. A slice that decides a *new* rule will move the
number far less than #790 did and be worth more.

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

The decline census that separated harvest candidates from indeterminates
predates #766, #715, #740, #790, #718, #716 and #813 — and is not re-derived
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
