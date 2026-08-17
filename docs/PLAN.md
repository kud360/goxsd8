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

## Status — 2026-08-17 (`/backlog`)

Conformance lanes — **paste `go tool lanestatus` verbatim**, never a
hand-count:

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1161 | 12 | 1173 |
| `instance` | 1337 | 25024 | 26361 |
| `json` | — | — | 0 |
| `schema` | 12913 | 2485 | 15398 |
| `xpath` | — | — | 0 |

An em dash is a lane with no cases yet, which is a different claim from a lane
scoring zero. `datatypes` is M3 and **complete**; `schema` is M4 and active;
`instance` is M5 and active; `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

**This stamp absorbs four landings and both active lanes moved: `schema`
12859 → 12913 (+54) and `instance` 1330 → 1337 (+7).** Each figure below is
the arbiter's, copied from the verdict rather than recomputed, and the four sum
exactly to the table above.

- **#842** at `3509de6` — the §3.12.6 required-subset CTA evaluator, `ta-Test`
  through `ta-AttrName`, so a `{type table}` ·conditionally selects· a type
  instead of `walk.governingType` declining the element and everything below it.
  **`instance` +4**, all four cases taking §3.12.2's `type` attribute arm.
  `xpath` stopped being a `doc.go`-only package; surface `+4 −0`.
- **#851** at `0740b51` — `TypeAlternative`'s `{type definition}` became the
  `TypeDefinitionOrRef` sum, so declare-ta's INLINE arm mints the anonymous type
  the element declaration owns instead of withholding the whole `{type table}`.
  **`schema` +1, `instance` +3**, all four the inline-arm shape.
- **#861** at `8565e80` — a raised CTA `{test}` error falsifies the WHOLE
  `{test}`, not the comparison node that raised it (key-cta-ta-select clause 2).
  **Unchanged, and honestly so**: the pinned suite holds 226 `<alternative>`
  tags and zero containing `not(`, verified rather than assumed.
- **#836** at `c116408` — a second sibling `<annotation>` is rejected everywhere
  but `<schema>` and `<redefine>`. **`schema` +53.**

**#836 is the estimate lesson the previous stamp reached for, arriving from the
opposite direction.** That stamp carried #830's finding — a decline census
bounds what a fix cannot win, and nothing bounds what it will win, because a
case may be invalid more than once. #836 forecast **+9 as a measured floor** and
paid **+53**. The 45 unforecast flips were not a second invalidity; they were a
**census taken over the wrong set**. The guard is a property of every element's
child list, and the forecast counted `msData/annotations/` alone, so the
under-count was six-fold and mechanical. **An estimate is bounded by the
population it was taken over before it is bounded by anything about the cases.**
The eighth forecast witness, `annotB020`, did not flip at all — the guard makes
its parse reject, and an unfollowed `_annotA014.xsd` then makes the decline
conjunction swallow the case (census 892 → 893). Both corrections were mason's,
made mid-implementation and confirmed independently by the arbiter.

Milestones, read from GitHub this pass. **Every cell is unchanged, and that is a
measurement**: none of the four landed issues carries a milestone, and neither
does any of the eleven filed since the previous stamp — if one did, M4's or M5's
open count would have moved and it did not. The M4 milestone's own
`updated_at` is 2026-08-15 and M5's 2026-08-16, both older than the landings —
so no cell could have moved.

| Milestone | Closed | Open | State |
|---|---:|---:|---|
| M0–M2 | 8 | 0 | done |
| M3 — Datatypes vertical slice | 12 | 0 | **complete** |
| M4 — Schema parsing | 83 | 42 | **active** |
| M5 — Instance validation (XML) | 13 | 12 | **active** |
| M6–M12 | 0 | 0 | not filed |

The M0–M2 row is M1 (3) plus M2 (5): **`M0 — Scaffold` carries no issues at
all**, which is why the row has always been a group and is recorded here so the
8 stops looking like an unexplained constant.

Queue: **222 open issues — 199 `ready`, 23 `blocked`, 0 `needs-replan`,
2 `epic`** (both `blocked`, so both counted inside the 23), against
**305 closed**. 199 + 23 = 222 exactly, and every one of the 222 carries a
queue label. Read the milestone table as feature progress and not as the
queue: **168** of the 222 carry no milestone (222 − 42 − 12).

**The move decomposes into five closures and eleven filings, and it reconciles
exactly**: 216 + 11 − 5 = 222. The closures are the four landings plus **#847**,
closed as moot by #851's post-land pass because that landing rewrote the
paragraph #847 existed to correct. The filings are **#858**, **#859**, **#861**,
**#862** and **#863** (#842's post-land pass, #861 since landed), **#867** and
**#868** (#836's), and **#869**, **#870**, **#871**, **#872** (this pass). One
issue was retitled (**#409**) and no open issue changed queue label.

**All four landings' follow-ups were disposed by their own post-land passes, and
this pass re-checked each disposition against the tree rather than trusting the
Next list.** #842's six: #861 filed and since landed, #862 and #863 filed, #56
correctly left `blocked` on #719 with #842 struck from its `## Depends on`, #859
unblocked to `ready`, and **#858's Acceptance rewritten to say two markers, not
three** — verified in the body today. #851's three: #847 closed as moot, #825
re-scoped to its surviving item rather than closed with it, #831 left untouched.
#836's four: #404's body corrected and given its first witness, #867 and #868
filed, the 45 unforecast flips recorded as needing none. **The one thing none of
them caught is `validate/cta.go:80`'s ownership — filed here as #871.**
**All four post-land passes ran entirely on GitHub and left no commit on `main`**
(`git log` between `3509de6` and `c116408` holds the four landings and nothing
else) — #400's exact shape, and until this stamp there was no durable record on
`main` that they ran at all.

**The unblock sweep moved nothing, for the sixth consecutive pass.** All 22
`blocked` bodies standing at the start of the pass were read for
`## Depends on` — the 23rd is #871, filed here: **10 name an open issue**, seven
say *"a trigger, not an issue"* (five of those additionally instruct the sweep
not to re-scan them, which it cannot honour without reading them), four carry
prose instead of a dependency list (#720, #548, #16, #250), and **#79 answers
`none (it is the dependency target)` while wearing the `blocked` label**. No
open body names any of the four issues that just closed. #717 still waits
correctly on the open half of `#715, #248`. The full distribution is on #779's
thread, and **#79 is the sharpest probe that issue has**: a label and a
`## Depends on` contradicting each other in one line, catchable by a check with
no judgment in it.

### Branch namespace, `origin` — report-only; a session never deletes a ref

**`go tool wipsurvey`, verbatim** (fed a hand-shaped issue list — **#840**,
paid again — because `gh` is a standing 403 on both GraphQL and REST here and
the MCP channel served everything else, which is docs/ROUTINES.md's fall-through
rule working as written):

```
ISSUE  BRANCH         TIP AGE   VERDICT  REASON
706    wip/issue-706  12m0s     LIVE     wip/issue-706: tip pushed 12m0s ago, within the 2h0m0s claim TTL
822    wip/issue-822  30h59m0s  RETIRED  wip/issue-822: issue #822 is closed
```

**A develop session is in flight on #706 right now** — the tip was pushed twelve
minutes before this survey ran, well inside the claim TTL. Nothing about #706 is
this pass's to touch, and it is deliberately absent from the band below.

**The tip ages are figures only because they were fetched.** A first run reported
`UNKNOWN — tip not fetched` for `wip/issue-706`: this checkout is shallow and
carries no `wip/*` tips, so
`git fetch origin 'refs/heads/wip/*:refs/remotes/origin/wip/*' --depth=1` is a
precondition for a `wipsurvey` that says anything about a live branch, not an
optimization. The shallow-clone finding (#802) still binds any ahead/behind
arithmetic here, so none is published.

**`wip/issue-822` @ `cc2d54e` is RETIRED and deliberately kept.** It was parked
on the second arbiter rejection (PRINCIPLES 30); **#851 superseded it and has now
LANDED**, which makes the branch re-planning evidence about a design decision
that is already settled rather than one still pending. It is never force-pushed,
never renamed, never a base to branch from, and **its deletion is a human's call,
not a session's**. Nothing is EXPIRED and there is no `parked/*` ref.

**`git ls-remote --heads origin` returns exactly `main`, `wip/issue-706` and
`wip/issue-822`** — while this checkout also holds
`refs/remotes/origin/claude/dazzling-cerf-x6w1wg`, which the remote does not
have. Second consecutive stamp at which a local remote-tracking ref outlives its
remote counterpart, and the standing reason `wipsurvey` reads `ls-remote` rather
than local refs (PRINCIPLES 28). Observed, not assumed — and not a claim that
this particular ref would have entered the survey, since `wipsurvey` scans
`wip/*` only.

**`go tool gapaudit`: 62 `GAP(` markers across 5 areas** — `xsd` 36,
`validate` 14, `xpath` 5, `xml` 4, `value` 3. Against the previous stamp's 60
that is **xpath +3** (#842's three declined CTA constructs) and **xsd −1**
(#851 deleted `typeTableRepresentable`'s marker with the function), which is
exactly what the two diffs predict. It ran **census-only**: reconciliation needs
a `kind/gap` issue list on stdin and #852 is why its untracked group must then be
hand-verified row by row anyway.

**#851's LOG entry publishes `95 → 94` for what it calls the gapaudit census,
and that is a mislabel, not a contradiction — dismissed here rather than
filed.** `grep -rn "GAP(" --include=*.go .` returns **94** today, matching that
entry exactly — a raw line count over the whole tree, `_test.go` files and
`tools/gapaudit`'s own `GAP(oops)` / `GAP(real)` fixtures included. Whatever
narrower set `gapaudit` counts, it is not that one, and the two figures are not
comparable: the entry names the wrong instrument for the number it publishes.
**This stamp is the authority for the census, and it says 62.** A pass that
wants the two reconciled should do it inside #852, which is already opening the
matcher.

**One marker turned out to be genuinely unowned, and two stamps had said
otherwise.** `validate/cta.go:80` — §3.12.4 clause 1.1.3's `{inherited
attributes}` merge — cites **#831** as *"the precondition for merging
correctly"*, and #842's and #851's LOG entries each record *"#831 keeps the
`{inherited attributes}` direction"*. #831 is a one-line `produceAttribute` fix
whose own Acceptance says §3.3.5.6 attribute inheritance *"this engine does not
implement at all"* and whose expected outcome is `Ratchet: unchanged`. It is
named at the marker as related, and was read as its owner. **#871** now owns the
retirement, `blocked` on #831. This is #815's shape, caught the same way #815's
own `produce_complex.go` row was caught: by reading the issue the marker names
instead of trusting that it fits.

### Working band

Dependency-ordered top of the `ready` queue, so a session need not scan 199 of
them. Take from the top. **Rows 1 and 2 of the previous band both landed** —
#842 and #836 — and the band below is re-cut rather than carried, because four
landings, two lane moves and eleven filings all changed what is cheapest next.

**Row 1 is `kind/process` and outranks a lane slice on purpose** (#527, #565).
Its tax is not hypothetical: the 2026-08-16 `/backlog` touched eighteen threads
and edited **zero** bodies, and this pass could not edit #409's either. Two
consecutive `/backlog` passes, same outcome, different diagnosis each time —
which is the shape that says the diagnosis is what is missing.

| # | Issue | Why here |
|---:|---|---|
| 1 | #872 + #857 | **No open issue body can be corrected today, and this is the second consecutive `/backlog` to pay it.** #764 landed the rule that an `issue_read` body is never written back, and named `WebFetch` on the issue URL as the safe re-read. Measured this pass: that returns **HTTP 404**, twice, on a **public** repo whose root page fetches fine — so the documented remedy does not exist here, and authoring from scratch (which discards the original) is all that is left. #872 owns the WORKFLOW correction and holds the measurement; #857 owns the round-trip probe and should widen to report the read path's **availability** beside its fidelity. Two half-answers today, one verdict after. **Re-run the 404 before writing — a 404 that has changed is a different issue** |
| 2 | #858 + #859 | **The CTA seam is the one demonstrably converting declines into passes, and these are its two remaining declines.** #842 paid `instance` +4 and #851 +3 more inside 24 hours, both by giving a `{type table}` a real verdict where `governingType` had withheld one. #858 is the three cast-shaped §3.12.6 constructs (`ta-CastExpr`'s `cast as QName ?` tail, `ta-ConstructorFunction`, a `ta-BooleanFunction` naming anything but `fn:not`); #859 is the wildcard arms of `[17] ta-AttrName` and the `AttributeValue` shape a sequence-valued attribute step needs. Both unmeasured, both say where to measure, and **both bodies are already correct** — #858's Acceptance was rewritten by #842's own post-land pass to say it retires **two** markers (`xpath/ctaparser.go:521` and `:552`) across three spellings, since §3.12.6 clause 3's Note resolves `QName '(' … ')'` by function name. Checked against the tree this pass, not carried |
| 3 | #867 | **The only MEASURED lane figure in the queue: `schema` +2, a floor.** An `<annotation>` carrying `<annotation>` children is still accepted — a different s4s fault from the one #836 just fixed, being `<annotation>`'s own content model rather than `xs:annotated`'s cardinality. `annotB001` and `annotB005` are the suite's only two such documents, both `invalid`, both `fail`, and **neither declined**. The grounding is done and in #836's thread, and the acceptance any fix must preserve is already a green row in `parser/produce_annotation_test.go` (a literally-named `{XSD}annotation` inside `<appinfo>` is lax content, not a violation) |
| 4 | #820 + #797 + #600 | **The landing mechanics, and every one of the four landings in this window paid them.** #830's LOG entry was lost by a squash-merge and re-landed by hand at `a40639a`; #823's, #836's and #861's Next lists then each had to warn the next session in as many words. #820 is the emptiness check that reads PRESENT while the entry is absent. #797 is where a code-free iteration puts its entry — and this trigger, again, has nowhere. #600 is the one-append-point merge tax on a 2.4 MB file |
| 5 | #852 + #840 | **The survey instruments, paid again by this pass.** #840: the `wipsurvey` input JSON was hand-shaped for the third consecutive pass, since `gh` is a standing 403 and nothing produces it otherwise. #852: `gapaudit`'s matcher never reads the `#N` a marker prints, so its untracked group must be hand-verified row by row — more expensive than the greps PRINCIPLES 27 replaced with it — and this pass ran it census-only for that reason. They land together or #852 lands on input nobody can reproduce |
| 6 | #868 | `complexTypeDecidable` declines a `<simpleContent>`/`<complexContent>` carrying NEITHER alternant — a grammar fault the producer rejects genuinely — and the `<simpleContent>` arm's diagnostic names a construct the author never wrote. Six declined cases measured alongside it by #836's post-land pass, `annotB030` among them, whose three prior records (issue body, thread and LOG entry) all misattributed it to `<xs:override>` production. Converts declines into decisions in `schema` |
| 7 | #786 | `simpleTypeDecidable`'s last decline is conservative, not forced, and its premise — that a `simpleType` naming none of §3.16.2.1's three alternatives needs a conservative decline — is a candidate for expiry now that #447 and #738 landed `list` and `union`. The same function family as #830 and #836, both landed; read those diffs first |
| 8 | #719 | `cvc-assertion` wired fail-open at every variety level — the M6 seam, marked and measured. It **gates #56**, whose `## Depends on` #842's post-land pass correctly left pointing here rather than unblocking it, and it decides the "genuine PASS versus unevaluated" encoding once (STYLE D4) for the CTA withhold to reuse |
| 9 | #853 | `cvc-elt` clause 5.1 is unimplemented, so an EMPTY element whose declaration carries a `{value constraint}` declines in **three** rules at once — the ·initial value· charge, the ID/IDREF binding and the ·key-sequence· member. `instance` candidate, unmeasured, direction can only be up because all three decline today. First step is an oracle question: whether #463's `checkSimpleDefault` is reusable at assessment time under clause 5.1.1's ·instance-specified· type |
| 10 | #625 → #669 → #748 → #492 | **README's Library block, ONE row in file order.** Splitting it across band rows is why it sat six passes. #625 fixes the `SchemaBuilder` pointer at closed #203; #669 the "works TODAY" snippet that does not compile; #748 the M5 block that denies a shipped API; #492 folds `ParseReport` into `README.md:116`. **#748 led the libuser report for the second consecutive consultation** — it is the finding a fresh reader hits first, and it concludes from it that validation is unusable |
| 11 | #747 + #870 + #514 + #687 + #672 | **The CLI contract, all five decided BEFORE #472.** #870 is new and is the one a user hits first: README's Quickstart runs `go build ./...`, which writes no binary, and the very next section invokes `goxsd8` — while the help path is documented as working today, so "nothing is implemented yet" does not cover it; its second half is the stub's own `go doc <import path>` remedy, which fails from outside the module tree, i.e. exactly where an installed CLI runs. #747 is the missing "Implemented today" paragraph, #514 typo-versus-unbuilt, #687 scoped help (now also carrying the declined `-help=true` finding), #672 `-version`. Each is a sentence or a dispatch branch while the CLI surface is still empty, and a change to shipped behaviour afterwards |

**Deliberately unbanded, and why.** **#871** is `blocked` on #831 and is this
pass's one blocked filing — but its arrival **re-prices #831**, which had been
filed as *"startable and correct to do, not valuable to do next"* with
`Ratchet: unchanged` in its own Acceptance. It is now the precondition for
§3.12.4 clause 1.1.3, so a session picking it should land it knowing what it
unblocks. **#869** is one false sentence in `builtin/strict`'s `New()` doc plus
a pointer to a `GAP(` marker that does not exist — real, one line, and it
compounds no cost, so it stays `ready` and unbanded (#347). **#862** and **#863**
are #842's other two post-land follow-ups: #862 stays latent until #858 or #859
widens the token set, and #863 is doc-only and belongs to a `/retro`. **#404**
gained its first known witness (`annotB020`) and had its "no suite case is known
to exhibit it" premise struck, both by #836's post-land pass. **#744**, **#773**
and **#721** are still held out on
one shared condition — each needs a warden pre-flight on an entry point that
does not exist — and **#484** owns that condition, which is the standing
argument for a process issue outranking its label: it unblocks three
band-grade issues at once, and **#871** is now a fourth wanting the same
thing. **#825** survived #851's landing correctly re-scoped to its item 2 and
did NOT close as moot, which is the outcome the previous stamp flagged as the
risk. **#843–#849** are the
2026-08-16 architecture audit's seven findings; **#843** is still the one whose
cost of delay is stated as increasing steeply. **#805**, **#806**, **#809**,
**#810** are `wipsurvey` hygiene on a tool that worked correctly again this pass.

**Two file neighbourhoods are moving fast enough that an issue body describing
them is the wrong thing to read — read the files.** #836 added
`rejectRepeatedAnnotations` to `parser/produce.go` and created
`parser/produce_annotation_test.go`, and **band row 3 (#867) opens exactly those
two**. `parser/redefine.go` has been rewritten five times in a week — #686,
#699, #506, #503 and #504 each moved it — and #744, #706 and #726 open it next,
with **#706 open in a live session right now**.

### Next planning action

**Re-derive the decline census before the next carve — carried from the previous
stamp, and this window is what turns the argument into a demonstration.** The
standing census predates sixteen landings. **The two filings in this window that
carry a number at all came out of a census a post-land pass ran on the spot**:
**#867** (`schema` +2, a floor — both cases decided-and-disagreeing rather than
declined) and **#868** (six declined cases, `annotB030` among them, whose
attribution three separate records had wrong). Against that, **#836** shows the
instrument's other edge — its forecast of +9 was taken over
`msData/annotations/` alone and the guard paid +53, so a census is only as good
as the population it is taken over, and **the population must be stated beside
the number**. The `schema` decline count is **893** as of `c116408`, measured by
#836's arbiter, up one from 892 because that landing's guard pushed `annotB020`
from a scored fail into the decline conjunction. **#570** is the issue that makes
this permanent — bank a per-lane decline baseline so every landing announces the
cases it just made decidable — and **#571** is its soundness half.

**The queue is 222 and the band is eleven rows, and the gap is not a backlog
problem.** `ready` means filed and unblocked; its size is an output and never a
target (#347). What the size argues for is still #779: a sixth consecutive full
read of every `blocked` body to conclude that nothing could move, now with #79's
`## Depends on: none` as a probe that needs no judgment to run.

**The one thing a session cannot currently do is correct an issue body, and that
is band row 1 rather than a footnote.** The read path is lossy (#764) and the
documented re-read path 404s (#872, measured this pass). The visible cost so far
is small and entirely of one kind — a stale premise gets a comment beside it
instead of a correction inside it, which is precisely what `/backlog`'s filing
discipline forbids. #409 is this pass's instance: retitled from the tree, with
its body's package list left stale and a comment saying so.

Everything else this queue needs is a develop iteration, and the band means what
it says: **take from the top, so start at row 1 (#872).** It is doc-only and one
session, and until it lands every later pass keeps paying the same tax in
comments-instead-of-corrections. Note that it is also a **code-free iteration**,
which is #797's open question about where such an entry goes — expect to meet
that problem while landing this one, and read #797 before improvising. If a
lane-moving slice is wanted instead, take **#858** at row 2: the CTA seam has
paid `instance` twice in twenty-four hours and these are the constructs it still
declines.

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
producer widening, finalize validity and composition edge cases. The
GitHub milestone holds the feature slices; the comment-accuracy, doc and
process issues that post-land passes file against the same packages sit
outside it, so the milestone is a floor on M4's remaining work and not
the whole of it.

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
+9, unioned onto #716's). `instance` stands at **1337**, and thirteen of those
cases are not M5's: **landings outside this milestone keep moving this lane** —
#740 took it 520 → 532 on a merged tree neither parent could measure, #821 added
1 (·xs:error·), #733 added 5 (a top-level `<xs:attribute>`'s inline
`<xs:simpleType>`), and the CTA pair added 7 more (#842 +4, #851 +3). A slice
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
**#851** gave §3.12.2's inline arm a real anonymous component. What remains of
the CTA subset is **#858** (the three cast-shaped constructs) and **#859** (the
wildcard `ta-AttrName` arms); **#871** is the §3.12.4 clause 1.1.3 ·inherited
attributes· merge, blocked on M4's #831. None of the four carries a milestone,
which is the same pattern M4's tail records.

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
