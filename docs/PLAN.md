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

## Status — 2026-08-23 (post-land pass for #950; THREE landings absorbed — #956, #958, #950 — lanes/milestones/queue and both surveys re-derived, the band's top three rows dropped and its head replaced, nothing filed)

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

**One lane moved, and all of the movement belongs to ONE of the three landings.**
`schema` **13076 → 13225 (+149)**, fail 2322 → 2173, banked by **#956** at its
own landing and confirmed there by the arbiter's `GOXSD_RATCHET=1` run.
`instance` and `datatypes` are unchanged. #958 and #950 each landed
`Ratchet: unchanged` with the arbiter checking
`git status --porcelain -- conformance/testdata/expectations/` before and after
its own writing run: empty both times, nothing banked, no expectations file in
either branch diff.

**#950's flat ratchet is unchanged BY CONSTRUCTION, not by observation, and the
distinction is worth keeping.** `decidableRules` (`conformance/instance.go:357`)
scores neither `w-props-correct` nor `cvc-datatype-valid`, so no conformance case
can see which of the two an out-of-enumeration `processContents` is charged. An
attribution fix whose lane movement is structurally zero is otherwise
indistinguishable from one that silently failed to move.

Landings absorbed by this stamp, three, at `311ada8..ea00f84`:

- **#956** (`eb66861`) — `checkS4SChildOrder` walks an element's children against
  an ordered list of content-model **positions** — not a fixed linear name order,
  because `xs:attrDecls` is one `maxOccurs="unbounded"` choice over
  `attribute`/`attributeGroup` — over eight models transcribed from the XML
  Representation Summaries, wired into `produceComplexType`,
  `produceSimpleContent` and `produceComplexContent` ahead of every `src-ct`
  clause. **No rule ID minted and none exists**: §2.4 clause 1 `sd-valid` and
  §5.1's first bullet carry no citable identifier, so the charge is a plain
  wrapped `fmt.Errorf` on `rejectProhibitedAttrs`'s footing. Arbiter ACCEPT in
  one round. `Ratchet: schema 13076 → 13225 (+149)`, zero regressions across
  41,759 cases.
- **#958** (`e617526`) — the decidability gate stops admitting `<xs:assertions>`,
  a name the s4s grammar has no ELEMENT for. The defect was **worse than the
  issue claimed**: the arbiter drove a `<simpleContent><restriction>` carrying
  that child through `produce` and got `err = <nil>` and a **built schema**, so
  the harness was reporting valid for an s4s-invalid document — a fabricated
  verdict in the unsafe direction, not the "admitted then dropped in silence" the
  body described. Fixed at the `kind != xsd.FacetAssertions` seam the producer's
  own `facetKindOf` uses. Arbiter ACCEPT in one round, zero findings.
  `Ratchet: unchanged`, measured.
- **#950** (`ea00f84`) — an out-of-enumeration `processContents` is charged
  `cvc-datatype-valid`, not `w-props-correct`, because `produceWildcard` raises
  the fault **before** `xsd.NewWildcard` builds the component the constraint
  quantifies over. The same two-layer split #932 settled one attribute group
  over, settled here by the identical argument. `w-props-correct` clause 1 stays
  charged, untouched, by `xsd.NewWildcard` on the component-exists path;
  `ruleWildcardCorr` had exactly one use in `parser` and is **deleted with it**.
  Arbiter ACCEPT in one round, zero findings. `Ratchet: unchanged`.

**All three follow-up ledgers are disposed and this pass filed NOTHING — which
is a first for a post-land stamp and is stated rather than left to be read as an
omission.** #956's and #958's own passes filed **#966** and **#968** before this
one ran; both are verified present, `ready`, and banded at rows 2 and 1 below.
#950's ledger held three items and every one was checked rather than accepted:

- **The arbiter's `strings.TrimSpace` non-finding is upheld, and the residue it
  does not cover was already filed as #455.** `processContents=" strict "` is
  accepted and that is correct — `whiteSpace` is `collapse(fixed)` on
  `xs:NMTOKEN` and §4.1.4's note fixes the order, pre-lexical facets normalize
  before lexical-space membership is tested. The repo has already written that
  ruling down, in `facetFixed`'s own doc comment. What the reading does **not**
  cover is the character class: `strings.TrimSpace` cuts `unicode.IsSpace`, which
  includes U+0085 and U+00A0, where §4.3.6's whitespace set is exactly
  `#x9 #xA #xD #x20` — so `processContents="&#xA0;strict"` is accepted today and
  the spec rejects it. That is **#455**, filed 2026-08-02, whose site list names
  `processContentsOf` explicitly.
- **The chronicler's friction note is correctly not its own issue, and is now
  attached to the issue that owns the class.** Third consecutive landing where an
  agent account disagreed with a checkable source (#956's 70× attribution error,
  #958's mutual misquote of the body, #950's false brief) — but **#681** already
  binds hand-off notes and log entries and already asks its reader to pick one
  remedy, and it is one of the ten issues `blocked` on the next `/retro`. An
  eleventh would pre-empt the retro's diagnosis and make a fifth unlanded
  correction to one `docs/WORKFLOW.md` paragraph. The new evidence — a
  **positive witness** for a reader-side mechanism neither of #681's two
  writer-side candidates describes — is on #681's thread.
- **Mason's absorbed `parser/doc.go` edit leaves nothing dangling**, verified in
  the tree rather than taken from the account: the contract paragraph now
  enumerates "a processContents outside the enumeration skip/lax/strict" among
  the `cvc-datatype-valid` sites, `processContentsOf`'s deferral comment is
  rewritten rather than merely repointed, and `grep -rn 'ruleWildcardCorr'
  parser/` returns nothing.

**One open body was EDITED rather than commented on, because this landing made
two of its premises false.** **#455**'s Spec section listed `p-props-correct` and
`w-props-correct` among the rules "already charged at these sites"; neither is
charged at any of them any more (#932 and #950 moved both to
`cvc-datatype-valid`, and #950 deleted the constant). Its item 3 enumerated eight
line-numbered sites; re-derived at `ea00f84` the set is **ten** by function name,
the two it missed being `checkDefaultOpenContent` and — in a file the list did
not reach at all — `parser/redefine.go`'s `checkSelfReferenceOccurs`. The item
now says that item 1's `grep -n 'TrimSpace' parser/*.go`, not the list, defines
the set. Substance untouched.

Milestones, read from `repos/kud360/goxsd8/milestones` this pass and
cross-checked against the paginated issue list, which agrees exactly.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 97 | 45 | **active** |
| M5 — Instance validation (XML) | 15 | 13 | **active** |
| M6–M12 | 0 | 0 | not filed |

**M4 is the only row that moved, by exactly the three landings**: closed 94 → 97
and open 48 → 45, all three of #956, #958 and #950 carrying the M4 milestone.
M5 is untouched. **172 of the 230 open issues carry no milestone** (230 − 45 −
13), so the milestone rows are feature progress and the queue paragraph below is
the queue.

Queue: **230 open — 203 `ready`, 27 `blocked`, 0 `needs-replan`, 2 `epic`**
(both `epic`s are `blocked`, counted inside the 27), against **347 closed**.
203 + 27 = 230 exactly, and **every one of the 230 carries a queue label** — the
class #773/#774 fell into is empty for the tenth consecutive stamp. Both figures
were re-derived by paginating the issue list (page-numbered, not `--paginate`,
whose Link header uses numeric-ID URLs the proxy blocks), raising the page count
until a page came back empty, and discarding pull requests, which share the
endpoint. **The move reconciles exactly**: closed 344 → 347 is #956, #958 and
#950 and nothing else; open 231 → 230 is those three closing against **#966** and
**#968** filing, neither of them this pass's.

**The unblock sweep moved nothing again, and this one is a PARSE of every
dependency rather than a search for a number.** All 230 open bodies were
fetched over `gh api` — byte-faithful, where MCP `issue_read` is
lossy (#764) — and searched for `#950`, `#956` and `#958`: **`#950` appears
nowhere in the queue at all**, and the only hits for the other two are #966
citing #956 and #968 citing #958, both as evidence in a `ready` body, neither in
a dependency line. Every `## Depends on` section of all 27 `blocked` issues was
then parsed and its issue numbers resolved against the full state map: **not one
blocked issue has all its named dependencies closed.** Five bodies (#960, #946,
#881, #796, #681) name a `/retro` trigger and no issue; three more (#696, #692,
#548) name a trigger and cite a closed issue as **provenance only**, which a
number-matching sweep would mis-read as a satisfied dependency — they were read
rather than matched. No `## Depends on` was repaired and no label changed.

**No duplicate was closed and no body other than #455's was rewritten.** Nothing
was filed, so nothing needed a duplicate check.

**No persona stories were folded, because none were handed to this pass.** A
post-land pass is not a consultation: the cartographer has read the source and a
persona it role-played itself would launder an insider's opinion as an
outsider's (#416). The 2026-08-22 `/backlog` consultation's findings stand as
that stamp recorded them, and rows 12 and 13 of the band below carry them
unchanged.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed the paginated issue JSON):

```
ISSUE  BRANCH         TIP AGE    VERDICT  REASON
822    wip/issue-822  165h57m0s  RETIRED  wip/issue-822: issue #822 is closed
862    wip/issue-862  main's     CLAIMED  wip/issue-862: no commits of its own; tip age is main's, not the claim's -- do not retire on age, settle it from the issue thread
872    wip/issue-872  131h59m0s  RETIRED  wip/issue-872: issue #872 is closed
933    wip/issue-933  60h12m0s   RETIRED  wip/issue-933: issue #933 is closed
```

`git ls-remote --heads origin` returns exactly `main`, `wip/issue-822`,
`wip/issue-862`, `wip/issue-872` and `wip/issue-933`, re-read this pass — the
same five as the previous stamp, and **none of the three landings left a ref
behind**, GitHub's auto-delete having taken `wip/issue-956`, `wip/issue-958` and
`wip/issue-950` at merge. Nothing is EXPIRED, no `parked/*` ref exists, and the
four rows are unchanged in verdict.

**`wip/issue-862` is still a LIVE empty claim and its clock keeps running.** It
sits at `c2ba631`, which is not `main`'s tip and is still not a commit of its
own, so it stays `ahead 0`, can never be EXPIRED, and can never be retired on
age (#722) — the "main's" in the TIP AGE column is the tool saying it has no
clock of its own to read, not a claim that the branch is current. Its thread's
last comment is the GROUNDING posted **2026-08-20T20:20Z** and nothing since —
**~57 hours as this stamp was written**, up from ~45 at the previous stamp and
now past the ~2-day threshold #867's takeover used and that **no document
states**. That rule is **#946**'s to settle and #946 is `blocked` on the next
`/retro`; until then #862 is off-limits by the same judgment the previous four
stamps applied, and the grounding remains the asset a session taking it would
start from. **`wip/issue-933` is RETIRED and kept** — #933 closed as #862's
duplicate and the branch is the grounding session's, at the same SHA.
**`wip/issue-822` and `wip/issue-872` are RETIRED and kept**, superseded by #851
and #878. All three deletions are a human's call.

**A leftover agent worktree is live in the landing container and it doubles
`grep -r`.** `git worktree list` after PR #969's merge shows
`.claude/worktrees/agent-a17352d3814a561f4` still checked out — on a branch of
its **own** this time (`worktree-agent-…`, not `wip/issue-950`), so #675's
two-writers condition did not recur and #696's "make the name distinct by
construction" candidate may already hold. The discard half did not: a surviving
worktree is a second copy of the tree, so `grep -rn "GAP(" --include='*.go' .`
returns **192** where `git grep` returns **96**. Recorded on #696 with the
commands; it is a container artefact, not a repo fact, and no ref was touched.

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
  #968 conformance: simpleTypeDecidable never consults facetElement, so an XSD-namespace child Part 2's <restriction> content model has no position for — <xs:assertions>, an <xs:attribute>, a particle — is ADMITTED and silently dropped, fabricating a valid verdict at the datatype-level sibling of #958's site
```

**The marker census did not move across three landings that touched `.go` files,
and Group 2 grew by exactly one tracker.** The raw `GAP(` token count is **96**
at `9c10af8`, `311ada8` and `ea00f84` alike, measured with `git grep` — the tool
still reports 64 and the same five-area composition, **Group 1 is EMPTY for the
tenth consecutive stamp**, and Group 2 is 9 → **10** with **#968** added. #968 is
a conformance-lane gap, which carries no `GAP(` marker by nature and belongs in
Group 2 permanently — it is not a stale tracker. **#852** still owns the matcher
qualification (the raw token count against the tool's 64) and stays below the
fold because the tool again ran with reconciliation and Group 1 empty. **#960**
still owns the class the census structurally cannot see: a fail-open disclosed in
PROSE carries no `GAP(` marker, so it appears in neither the census nor Group 1.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 203 of
them. Take from the top. **The previous band's top THREE rows all landed**
(#956, #958, #950); the band is re-cut by dropping them and re-deriving every
cross-reference by ISSUE, never by row number, which decays at each re-cut.
**Rows 5–14 are carried with their arguments intact and were NOT re-derived by
this pass** — what this pass established is rows 1–4, the three departures, and
the two promotion discriminators below that were checked and did not fire.

**Neither standing discriminator fired, and both were checked rather than
assumed.** **#963**'s row said the next landing that misses landing precondition
1 moves it to the head; all three landings carry `docs/LOG/2026-08.md` **inside
the squash itself** (`git show --stat` on `eb66861`, `e617526`, `ea00f84`), so
none is a #924 and it stays below the warm rows — recorded on its thread so the
next pass does not re-run the check. **#846**'s row said a third witness moves it
above the lane rows; neither #956's nor #958's log entry records the shadow tax,
so it did not grow one.

| # | Issue | Why here |
|---:|---|---|
| 1 | #968 | **A live FALSE ACCEPT in the file #958 just left, filed by that landing's own pass on a finding its arbiter verified by running it.** `simpleTypeDecidable` never consults `facetElement` at all, so the same `<xs:assertions>` under a *datatype-level* `<simpleType><restriction>` is still admitted and still dropped — the sibling one layer down from the site #958 fixed, in the same function family, with the same fabricated-valid-verdict consequence the arbiter demonstrated there with `err = <nil>` and a built schema. Warmest files in the tree, one-session slice, and the §3.4.2.2/Part 2 seam the last two landings read is still legible |
| 2 | #966 | **The footing under 149 newly banked `schema` cases is a precedent no document states, and each landing inherits it from the last.** `xsderr/doc.go` says free-form errors are "never for validity verdicts" and STYLE E2 says an unnameable rule means you have not read the spec; **eight** s4s-grammar rejections are plain `fmt.Errorf` because §5.1's first bullet is genuinely unnumbered. #956's GROUNDING flagged the tension and ruled it out of that issue's scope in the same breath — correctly, a producer slice is not where a module-wide error-currency policy gets decided. Ranked here on CLAUDE.md's cost rule: successive producer landings have each inherited it from the last — #340, #904, #928 and now #956, per the site list in #966's own body — and the alternative to deciding it is another silent one. Doc-only, one session, `Ratchet: unchanged` expected |
| 3 | #884 | **#950's own adjacent shape, named in its body as *"the adjacent shape and closes nothing here"* — and the session that just did the two-layer split has the context.** Every malformed named `<group>` body (a nested `group ref`, an `<element>`, annotation-only, empty) collapses into ONE `mgd-props-correct` message naming an internal invariant, located at the definition rather than the fault; all four verdicts are reproduced in the body against a named tip. **Sequence it AFTER row 2**: #884's Spec section concludes the fault carries no rule ID and is a plain `fmt.Errorf` on §5.1's first bullet, which is exactly the footing #966 exists to rule on — landing it first adds a ninth site to #966's list instead of reading its answer. This ordering is not recorded in either body; see #884's thread |
| 4 | #963 | **The tax falls on every landing, and it has two witnessed failures — but its own discriminator did NOT fire this window and the row says so rather than promoting it quietly.** #820 landed the *form* of landing precondition 1; nothing checks that the check was run, which is **#924**'s shape (`53bf113` squashed eight files and no `docs/LOG/` path; `aeae89f` carried the entry afterwards) and is not reachable by prose. One session's work — `tools/landcheck`, three real-history fixtures, one WORKFLOW line — and explicitly **not** a gate part, since #304 struck `logguard` and made CLAUDE.md's Commands block the sole gate definition. Three consecutive clean landings is weak evidence that prose suffices and no evidence that #924 decays; **the discriminator stands unchanged** |
| 5 | #846 | **#909 paid the shadow tax and said so: 183 lines of `conformance/schema.go` shadowing 364 of `parser/produce_complex.go`, correct only because the arbiter walked `attrDeclsDecidable` against `main` by hand.** Row 1 will pay it again. Ranked BELOW it and the tension is stated rather than hidden: landing this first would make row 1 cheaper, but it is a ~700-line refactor with no evidence it fits one session, and a warm measured slice should not wait behind an unsized one. **No third witness appeared in this window** — neither #956's nor #958's entry records it — so its promotion rule did not fire |
| 6 | #941 | **#387's own unfiled debt, and the coldest files of any band row** (`07117dc`, 2026-08-21). `Element.Attributes` and `BaseURI` stand exactly where `Parent` stood: zero out-of-package non-test callers, no `Node` obligation, both prongs of STYLE T5 failed. The two are NOT the same shape — `BaseURI` is a delete (the field/method collision that forced #387's) and `Attributes` survives as an unexported delegation — and **three** black-box test sites move, not the two warden named. `surface: +0 -2`, strictly shrinking. **Warden pre-flight required**; do NOT start from #387's item-2 table, which cites a file that no longer contains either call |
| 7 | #953 | **#924's other post-land filing, and a doc claim that is FALSE rather than merely stale.** `xsd/valuespace.go`'s FAIL-OPEN CONTRACT says "every caller turns a decided negative into a schema rejection"; `validate/cvcattribute.go` charges an INSTANCE violation and has since before #924's branch. Because the claim is unenumerated it also violates STYLE P3a, so one edit settles two rules. Pre-existing on `main`, out of scope for #924 by its arbiter's own ruling, and discharged there only by a commit body naming it in prose |
| 8 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in three rules at once — and all three markers name no issue (STYLE P3). `instance` candidate, unmeasured, direction can only be up. **First step is an oracle question**, not code: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 9 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured, and Group 2's only entry that is a live engine gap rather than a lane gap. It **gates #56** and decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse. Nothing gates it; startable on `main` today |
| 10 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced; its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. **Read it beside row 1**: #968 is the same function's *other* defect and touches the same seam, so whichever lands second rebases onto the first. Its measured delta shrank in an earlier window (`notatF067` was one of its witnesses and #945 banked it); the measurement is still the thing to run, and #868's diff is the place to start |
| 11 | #907 | **A repo tool catalogues every `childElement` sibling-alternative site and reports which is unguarded** — hand-written guards accumulated over months, each after a suite case tripped over it. `kind/tooling`, banded below the rows above because the tax was paid over months rather than in consecutive sessions. **Its census is now stale by at least FOUR landings and is NOT re-derived here**: #909 rewrote 418 lines of `produce_complex.go`, #957 moved `produce_typetable.go`, and #956 added `produce_s4sorder.go` outright — re-run the census before designing from the body's figures |
| 12 | #669 → #625 → #748 → #896 → #492 (+ #934) | **README's Library block, ONE row in the order the issues name; splitting it is why it sat.** 2026-08-22 libuser reconfirmed all six and filed nothing: **#669** the "works TODAY" snippet still fails to compile (`parser.WithLogger(logger)` names an identifier the block never declares, beside three unused-variable errors); **#625** still points at closed #203 while `xsd.Example_buildFinalizeQuery` exists and passes; **#748** still says shipped `validate.New`/`xmlsrc.Validate` "do not exist yet", with all three signature mismatches holding; **#896** the package "Contract" prose still never names `Err()`; **#492** README omits `ParseReport`/`AssemblyReport`/`ReadDocument`/`Produce`; **#934** the violation example still shows `[cvc-datatype-valid]` where #913/#914 now charge `[cvc-type]`. Citations were re-checked at the 2026-08-22 stamp against `79b0bd8` (2026-08-16, the commit that last touched `README.md`) and are **carried, not re-run, here** |
| 13 | #870 + #747 + #514 + #687 + #672 | **The CLI is unreachable from its own documentation; 2026-08-22 cliuser reconfirmed all five and filed nothing — fourth consecutive such verdict, so the gap is disclosure not discovery.** **#687 gained two behaviours and its body a third Acceptance question**, both reproduced at `9c10af8`: `goxsd8 -xyz -help` prints full help and exits **0**, silently swallowing the bogus flag, and `-help=true` — the stdlib boolean-flag idiom — is NOT recognized and falls to the stub with exit 2. Both follow from `wantsHelp` being a raw token scan with three exact string comparisons rather than flag parsing. **#870** Quickstart's `go build ./...` writes no executable and the stub's own `go doc` remedy fails outside the module root; **#747** `-help` is a strict subset of `go doc`; **#514** every non-help input is byte-identical stderr plus exit 2; **#672** `-version` in any spelling hits the stub |
| 14 | #885 | **The scope rule that has cost a reject round TWICE and still survives only in verdict comments.** #876 and #904 each shipped half a Goal because an Acceptance exemption was read as a scope exemption; #374's finding 1 added a third datum. Three discriminators, one sighting each — the body says if the third will not fit the sentence, state the two-datum rule and why |

**Deliberately unbanded, and why.** **#409** is `ready` since 2026-08-02 and
carries a **third** independent sighting — the 2026-08-02 steward audit that
filed it, and the 2026-08-11 and 2026-08-22 libuser passes, both of which reached
it from the published surface alone with no knowledge that it existed —
`codegen/doc.go` prints `Generate` and `Target` in a code block while the package
exports nothing, and `#203`'s landed `xsd/doc.go:213` heading is the exact
spelling to copy. It stays unbanded only because it is one row of a five-file
convention landing and no session has been blocked by it. **#937** is correct and
`ready` but says in its own body that it is naturally folded by the next landing
touching `rejectRepeatedAnnotations`. **#920** and **#921** are conformance-
bookkeeping follow-ups below the fold. **#929** and **#931** are the small parser
occurrence / rule-mapping gaps #901 exposed (#932 took the third, and #950 took
#932's own follow-up); read each beside #901's thread. **#455** is now edited to
match the tree after #932 and #950 and is the live owner of the
`strings.TrimSpace`-versus-§4.3.6 character class at **ten** sites — unbanded
because it is a pure false-accept narrowing with a provably flat ratchet, and
**#456** stays `blocked` on it. **#862** is `ready` and its grounding is banked,
but its branch is a LIVE empty claim whose clock has now run ~57 hours past its
last comment — off-limits until #946 rules, and it is the worked example #946
asks for. **#888**, **#889**, **#894** are the three `area/xpath` gaps still
awaiting a suite census in their range (#889 states a warden pre-flight per
#484). **#843–#849** are the 2026-08-16 audit's findings, **six open** — #847
closed `not_planned` on 2026-08-17 — of which **#843** has the steepest cost of
delay and **#846** is banded above. **#566** is #565's open sibling, routed
nowhere by #565's landing and correctly so. **#871** stays `blocked` on #831.
**#881**, **#548**, **#622**, **#681**, **#692**, **#696**, **#796**, **#841**,
**#925**, **#946**, **#960** are `blocked` on the next `/retro` (or a ruling),
not on any landing — **ELEVEN of the 27, not the ten the previous four stamps
carried**. The list was re-derived this pass by parsing every `## Depends on`
for the word `retro` rather than by copying it forward, and **#681 was missing
from it** — an omission, not a new arrival. **#681** and **#696** each gained a
fresh datum from this landing rather than a new sibling issue.
**#570** carries the standing `schema` decline-count argument at 893; the
previous stamp measured 788 at `311ada8` and **this pass re-measured after
#956's +149 and got 788 again**: gate part 4 at `ea00f84` reports *"lane schema:
788 declined case(s) recorded fail"* plus 11 more declined as indeterminate
(#277). **The decline count did not move while the lane gained 149**, which is
the expected reading of #956 and worth stating: its cases were
decided-and-DISAGREEING before it, not declined, so a lane gain and a decline
harvest are different events and #570's argument must say which one it counts.
Whether 893 counted the same quantity is #570's to confirm — but it starts from
788, not from a number four landings old.

### Next planning action

**Take from the top: start at #968**, and read #786 beside it — both are
`simpleTypeDecidable`, the same function's two open defects, and whichever lands
second rebases onto the first. #968 is the warmer and the more serious: it is a
live false accept that fabricates a valid verdict, filed by #958's own post-land
pass on a finding its arbiter reproduced by running it, in the file two of the
last three landings already read.

**#966 is the row to take if a doc-only session is wanted, and it should not sit
long.** Its cost is the only one in the band that **compounds per landing**:
each producer landing has inherited an error-currency footing no document
states from the one before it (#966's body lists the eight sites and their
issues), #956 put 149 banked `schema` cases behind it, and the alternative to
ruling on it is another silent landing on precedent. That is
CLAUDE.md's "band process work on the sessions it costs" applied to a tax that is
still growing, where **#963**'s has been flat for three landings.

**Row 3 carries an ordering that neither issue body records: #884 after #966.**
#884's Spec section concludes its fault carries no rule ID and is a plain
`fmt.Errorf` on §5.1's first bullet — the precedent #966 exists to rule on — so
landing #884 first adds a ninth site to the list #966 is trying to settle.
Recorded on #884's thread as well, because a band row is replaced at every
re-cut and a thread is not.

**The two standing promotion discriminators were both checked this pass and
neither fired** — #963's (three consecutive landings carry their `docs/LOG` entry
inside the squash; none is a #924) and #846's (no third shadow-tax witness in
#956's or #958's entries). Both checks are recorded — #963's on its own thread —
so the next pass re-runs them only against NEW landings.

**The next `/retro` has ELEVEN `blocked` issues waiting on it, and the number
rose by a correction rather than by an arrival** — #881, #548, #622, #681, #692,
#696, #796, #841, #925, #946, #960. The list was re-derived by parsing every
`## Depends on` for the word `retro`; the ten carried by the previous four
stamps had **#681** missing. Nothing has left the list. Two of them (#946, #960)
are the reason a live branch claim and prose-only gaps go unadjudicated today,
and **two more gained evidence this window without gaining siblings**: #681 has
its third sighting and its first positive witness, #696 its second sighting with
the two-writers half refuted and the discard half measured.

**Four unlanded corrections still target one paragraph of `docs/WORKFLOW.md`'s
filing discipline** — **#510**, **#646**, **#679**, **#912** — and whichever
lands last rebases three times. Decide one issue or four before filing a fifth;
this pass declined to file that fifth and put its evidence on #681 instead.
**The CTA cohort's 45 banked `instance` failures remain unattributed**, ninth
consecutive stamp carrying it. **`gate.yml` runs but is still not a required
status check**, which only the repository owner can change. All three stay open
and stay true.

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
family are **#471** (a local `<element ref=>` carrying `substitutionGroup=`,
silently accepted), **#931** (occurrence attributes on a named `<group>`'s
child compositor), **#929** and **#455**. The GitHub milestone holds the feature
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
