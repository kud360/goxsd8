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

## Status — 2026-11-11 (`/backlog`, on `main` at `24289b7f`)

**M10, the codec, is the north star.** The largest decision in its census is still a content sequence or choice occurring other than once, given codegen's occurrence field: sequence 129 and choice 120, 249 cases. #3593 holds it and is now `ready`, since #3587 landed and the #3587 post-land pass unblocked it. It is Tier A row 1. The nested row is at 120. #3594 (4) is `ready`, and #3588 (48) and #3589 (49) stay `blocked` on #3593. Since the 2026-11-10 stamp, nine landings closed band rows 1-7, 9 and 10: #3587, #3579, #3623, #3557, #3599, #3618, #3584, #3580 and #3581. #3619 (row 8) parked on its second reject and was closed `not_planned` for #3643. No lane moved. M10's measure went from 13750 decoded to **13909**. The post-land passes filed #3630 (shape K), #3635 (xsi-namespace uses), #3640, #3643, #3644 and #3648. This pass files #3651, the largest M10 row no ruling holds ({open content}, 55). The xsi:type row (86) is larger, but a ruling holds it (see below). **Lean: product, M10 first.** Seven of the window's nine landings were product: four codec slices and three message defects. The other two were #3557, a banded refactor, and #3584, a ruling written up as doc. Two process costs each lost a round: #3644 (the reviewer-side census miss that led to #3619's cap round) and #3640 (a classifier-refused landing that owed a takeover and a re-judge). They lead Tier C.

### Conformance lanes

**This table is `go tool lanestatus`, pasted verbatim, on `main` at `24289b7f`.** It is the committed expectations census, which `docs/WORKFLOW.md` names as the lane score (#1120).

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1163 | 10 | 1173 |
| `instance` | 26290 | 71 | 26361 |
| `json` | 43 | 0 | 43 |
| `schema` | 15303 | 95 | 15398 |
| `xpath` | — | — | 0 |

**No lane moved since `9caabc00`.** `git diff 557f7482 24289b7f -- conformance/testdata/expectations/` is empty. **`datatypes ⊆ instance` is intended** (#1507). **An em dash means a lane with no cases yet, not a lane scoring zero.** `datatypes` is M3 and is complete. `schema` is M4, `instance` is M5 and `json` is M8, and all three are active. `json` moves only by the curated cases a landing adds. `xpath` waits on M7 and `ber` on M11. M9 and M10 have no lane: M9 is read by `go tool gencensus` and M10 by `go tool decodecensus`.

### What holds each lane's failures

**The W3C lanes: no fresh partition in this pass.** This doc-only pass ran gate part 4 green on its tree, without `GOXSD_DECLINES=1`, so it has no declines run to partition. Every landing since `ea869445` banked `Ratchet: unchanged` under the arbiter's ratchet run, and no expectations file changed, so the partition at `ea869445` stands. #3546 (`validate/`), #3574 (`parser/`), #3580 (`xsd/`), #3581 (`value/`, `parser/`) and #3618 (`value/`) touched lane code. The next pass that runs part 4 re-partitions. `instance` has 71 banked fails (39 declined, 5 indeterminate, 27 decided against the suite) and `schema` has 95 (1, 11 and 83). The clusters of five or more are:

- `schema`: `MS-Wildcards` 49 (#1809's recorded divergence), `MS-Schema` 6 indeterminate, `Open` 5 (#1374, `blocked`) and `Simple` 5.
- `instance`: `MS-Regex` 18 (#1899's ruling), `MS-Attribute` 11 (#2151's ruling) and `MS-Additional` 5 (decided against the suite; `lanepartition` names no open issue).

**No session lever is left on a W3C lane.** What remains are rulings: #2081 (the owner's, which #3477 also waits on) and #1809's divergence. M4 and M5 stay active at their ruled floors.

**M9: `go tool gencensus` at `24289b7f`.** It found 15470 documents and assembled 11867. Of those, 11865 generate, 2 decline and 0 fail any other way, unchanged since `9caabc00`. #3286 makes "generated" mean the output type-checks.

| first decline | documents | held by |
|---|---:|---|
| struct-value recursion | 2 | permanent (#3256); no finite instance exists |

**M10: `go tool decodecensus` at `24289b7f`:** 26361 cases, 26209 generate, **13909 decoded** (13750 at the last stamp), 690 declined, 4 undecided, 0 rejected codegen's own type, 11606 failed other (validate's charged instances), and 0 decoded where validate rejects. Each count bounds from above what lifting that construct alone decodes. The nested row's split is #3538's post-land throwaway measure at `b92222b3`, restated by #3579's post-land pass. No committed tool reproduces it.

| first decline | cases | held by |
|---|---:|---|
| a content sequence with {min occurs} 0 and {max occurs} 1 whose model group is emptiable | 1 | #3630 (shape K, spliced 0..1: `isDefault072`); measured at #3593's landing, which lifted the 129 and 120 of the two rows this one replaces |
| a nested model group particle | 120 | #3589 (49) and #3588 (48), both `blocked` on #3593; #3594 (4) |
| an element carrying xsi:type | 86 | **ruling**: refused permanently on a concrete selected type (#3291 RULING R5, issuecomment-6560566007; `codec/doc.go`). Both sampled cases, `elemT002` and `ctI028`, name a derived type. |
| {open content} | 55 | #3651 (this pass) |
| an element declaration with a {type table} | 42 | none filed; R5 rules the shape (take the branch from `Validator.ConditionallySelected`) |
| mixed content | 41 | none filed |
| an element declaration with {nillable} true | 37 | none filed |
| a union simple type two of whose basic members get one Go type | 19 | none filed |
| two particles taking one expanded name | 12 | none filed |
| two branches of a sealed interface whose structs have one shape | 11 | none filed |
| a content all group with {min occurs} 0 whose model group is not emptiable | 8 | none filed (#3623's residue) |
| an attribute use in the xsi namespace | 7 | #3635 (at most 5 decode) |
| an element declaration whose complex {type definition} has {abstract} true | 3 | none filed |

### Branch namespace, `origin` (report-only; this pass deletes no ref)

**`go tool wipsurvey`, fed by `go tool issuefeed`** (`rows 3650 distinct 3650 min 1 max 3650 no gaps`). No `wip/issue-<N>` is LIVE or CLAIMED. Every wip branch of this window's nine landings was auto-deleted at merge.

- **RETIRED, deletable now:** `wip/issue-2013`, `wip/issue-2332`, `wip/issue-2696`, `wip/issue-2699`, `wip/issue-2722`, `wip/issue-2896`, `wip/issue-2899` and **`wip/issue-3577`** (tip `349569e4`). `wipsurvey` reads each issue as closed, and an earlier stamp verified the first seven. #3577 closed `not_planned`, and #3587 ported its diff by hand and landed it as `0adce11c`. The prior art that kept the ref is consumed.
- **RETIRED, keep until #3643 lands:** `wip/issue-3619` (tip `dd9e9b2c`, last checkpoint `b8356909`). #3619 closed `not_planned`, and its content is not in `main`. #3643's body starts from `b8356909`.
- **BLOCKED:** `wip/issue-2081`, a parked grounding with no code, and `wip/issue-3477` (tip `9d9540e2`, the claim commit alone). Both wait on the owner's PRINCIPLES 25 ruling.
- **Deletable now, superseded passes:** `meta/backlog-2026-10-01-b` (tip `a9e2b09b`) and `meta/backlog-20261013-0808` (tip `71738b02`). `backlog.md`'s test prints the landed `meta: backlog` commits `04f29430` and `596ed053` for each. `meta/backlog-20261111-1407` is this pass's claim.
- **Parked, deletable:** `parked/untriaged-20261026-084521` (tip `0d07455d`), `parked/untriaged-20261103-194004` (`1f7552f2`), `parked/untriaged-20261106-124613` (`cdeff01f`) and `parked/untriaged-20261107-124629` (`14f3030b`). Each one's single commit removes a landed `docs/LOG` entry, which is #3398. No new park since the last stamp.
- **Certified for deletion and still standing:** `chronicler-345`, `meta/post-land-1816`, `meta/post-land-1908`, `postland-2099` and `postland/issue-2284` (the 2026-10-19 triage).
- **For human triage:** `parked/untriaged-20260930-110432` and `parked/untriaged-20260930-215455`. The survey skips eight subagent branches: `wip/issue-1861-{mason,chronicler}`, `wip/issue-1926-{mason,chronicler}`, `wip/issue-2008-{mason,chronicler}`, `wip/issue-1988-mason-repair` and `wip/issue-1988-chron2`. Each one's issue is closed and landed.
- **Open pull requests from outside the loop, for human triage:** #3337 (`pr-go/3332-…`) answers #3332, which is closed, so it is superseded. #2735 (`pr-scout/2574-…`) answers #2574, still open and `ready`.

### Marker census

**`go tool gapaudit` on `24289b7f`, fed the same walk:** 200 markers across 11 areas (xpath 66, jsonsrc 41, xsd 37, validate 23, value 11, conformance 7, parser 7, xml 3, landcheck 2, regex 2, cmd 1). This is unchanged since the last stamp, and no landing added or removed a marker.

- **Group 1 has 10 rows**, unchanged. Eight are #2719's. The other two have an owner written into the body: `value/valuespace.go:152` (#462) and `tools/landcheck/keywords.go:48` (#2541).
- **Group 2 has 26 rows**, unchanged. #3019 (three unmarked `xsd` disclosures) stays there until it lands its markers.

### Milestones and queue

**Read from GitHub after this pass: 360 open issues.** `issuefeed` read `rows 3650 distinct 3650 min 1 max 3650 no gaps` before this pass filed #3651.

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 1 | 13 | complete; #3477 (xs:duration's grammar) is `blocked` on the owner |
| **M4 — Schema parsing** | **63** | **181** | active, at its ruled floor |
| **M5 — Instance validation (XML)** | **8** | **137** | active, at its ruled floor |
| **M6 — XPath required subset** | **6** | **46** | active |
| M7 — XPath 2.0 growth | 1 | 1 | #2619 open |
| **M8 — JSON instance adapter** | **2** | **55** | active; #2670 `blocked`, #3044 `ready` |
| M9 — Codegen | 3 | 87 | **done**. Residue: #3286 (tooling), #3178 (low), #2977 (`blocked`) |
| **M10 — Codec** | **9** | **57** | **north star**: #3593, #3594, #3630, #3635, #3651, #3556 and #3541 `ready`; #3588 and #3589 `blocked` |
| M11–M12 | 0 | 0 | not started |

The open issues are **340 `ready`, 18 `blocked` and 2 `epic`**, which sums to 360. No issue is `needs-replan` or unlabelled. Open `kind/process` is 8: #3644, #3640, #3606, #3596, #3590, #3550, #3547 and #1885 (`blocked`).

- **`blocked` is 18:** #16, #555, #1042, #1051, #1374, #1609, #1790, #1885, #2022, #2081, #2533, #2670, #2749, #2977, #3031, #3477, #3588 and #3589. #3593 and #3594 left the list at #3587's post-land pass, and #3648 entered and left it with #3581. No trigger fired since the last stamp, and every named dependency is still open or its trigger unfired. `go tool issuefeed -owner-comments` prints "No owner comment since the blocked label on any of the 18 open blocked issues". #1609's arms read `schema` Fail 95 (arm 1 fires at 0) and Total 15398. No landing since `557f7482` changed `xsd/contentrestricts.go`, so arm 3 has not fired either. #2977's consumer trigger has not fired.
- **Measured refactors, all re-run at `24289b7f`:**
  - The shared codec→codegen figure, `git grep codegen -- 'codec/*.go' ':!*_test.go' ':!codec/testdata/**' | wc -l`, reads **225**. It was 203 at the last stamp. It rose to 232 across #3587, #3579 and #3623, and fell to 225 with #3557 and #3599. #3556 quotes it, and its body now carries 225, so #3556 enters the band.
  - #2101's first figure retired: its Acceptance (1) is discharged on `main` (`errAttrScan`), so the body was restated. Its second figure reads 1.
  - None of the others grew: #3648 2, #3541 13, #3551 1, #3533 5, #3532 2, #3531 6, #3301 2, #3294 5, #3033 14/3, #2793 3, #2663 4, #2601 0, #2600 2, #2375 1, #2374 22, #2373 16, #2372 27, #2359 14, #2329 1, #2239 2, #2237 24, #2205 2, #1958 2, #1865 2, #1080 8 and #363 2.
  - #3392, #3307 and #3324 are unmeasured.

### Persona consultations

**No round this pass.** `go tool surface -base 557f7482` reports `surface: unchanged` since the last consultation (2026-11-10 at `557f7482`), so no persona round is due. That round's findings stand as filed, folded or dismissed in the 2026-11-10 stamp. **Consumed since then:** #3618 (cliuser 1) landed as `ceba2ffb`. #3619 (cliuser 4) parked and was replaced by #3643, which is `ready`. **Still open and banded:** #3643, #3622, #3620 and #3621, and the 2026-11-09 round's #3582 and #3583. The 2026-11-09 round's #3580, #3581 and #3584 landed. The folds (#1088, #1454, #1371, #1594, #1685, #3258, #3533, #1409) are unchanged.

- **Next round:** due when a Tier A slice changes `codec`'s exported surface, or when #3643 or #3622 lands. `surface` sees neither a doc nor a message change (#2203), so the orchestrating session decides by that trigger, not by `surface` alone.

M10 faces the API through `codec`. M9 faces the API through `codegen` and the CLI through `goxsd8 gen`. M4, M5, M6 and M8 face the API through `parser`, `xsd`, `validate`, `xpath` and `validate/jsonsrc`, and the CLI through `goxsd8 parse` and `validate`.

### Working band

This is ordered for a `/develop` session: take the highest row you can start. **Run `wipsurvey` fed before starting**; its WORKING BAND section prints each row's state. Each row names one issue (#1636).

**Tier A: the north star, M10.**

| # | issue | why here |
|---|---|---|
| 1 | #3593 | **A content sequence or choice occurring other than once, decoded into codegen's occurrence field** (129 + 120 = 249, the largest decision). It unblocks #3588 and #3589. |
| 2 | #3651 | **{open content} into codegen's OpenContent field** (55, the largest row no ruling holds). Attribution is `NextParticle`'s `*OpenContent` answer. |
| 3 | #3594 | **A model group particle codegen flattens beyond #3587's** (4). It retires two guard rows #3587 added. It shares #3593's decline list and Decode item 5, so the later landing rebases. |
| 4 | #3630 | **Shape K: a spliced 0..1 content sequence** (count unmeasured; the grounding measures it). `contentGroup` asks `goshape.Splices` above its compositor switch. |
| 5 | #3635 | **xsi:type, xsi:schemaLocation and xsi:noNamespaceSchemaLocation attribute uses into codegen's field** (7; at most 5 decode). The oracle rules what the field holds. |
| 6 | #3556 | **One home for the anonymous-type unit key** (shared figure 225, which grew 22 since the last stamp). |
| 7 | #3541 | **Retire the 13 datatype mirror structs onto `internal/goshape`** (13). It takes #3557's zero-struct hazard. |

**Tier B: persona-facing defects and documentation.** No W3C lane has a session lever left (see above).

| # | issue | why here |
|---|---|---|
| 8 | #3643 | **`Violations`' same-location order sentence**, which replaces parked #3619 (cliuser 4). It starts from `wip/issue-3619` at `b8356909` and owes only the round-2 findings, backed by a census of charge sites. |
| 9 | #3582 | **`cvc-enumeration-valid` names the members.** |
| 10 | #3622 | **A facet {test}'s decline names its reason.** Cliuser 2. It needs a warden ruling on `value.AssertionEvaluator` first. |
| 11 | #3463 | **QName-valued attribute charges at the attribute.** It is the `parser` sibling of #3580. |
| 12 | #3620 | **`FacetAssertions`' doc.** Libuser 1. |
| 13 | #3621 | **`codec`'s synopsis promises encode.** Libuser 4. |
| 14 | #3595 | **`ParticleEmptiable`'s "exported for package codegen" doc** is false. It is an `xsd` edit, and its body was restated at `24289b7f`. |
| 15 | #3583 | **README says the generated types enforce no facet.** |
| 16 | #3258 | **README's program as one runnable file.** It takes libuser 9's `Sprintf` nit. |
| 17 | #3074 | **README restates `-help`'s JSON mapping.** |
| 18 | #3331 | **`ApplyWhiteSpace`'s resolver sentence and an Example.** Low. |
| 19 | #3044 | **jsonsrc's trailing-content label.** Low severity: the exit code and class are right. |

**Tier C: tooling, refactors and process.** The two process issues that each cost a round in this window lead the tier. The rest keep their order behind product.

| # | issue | why here |
|---|---|---|
| 20 | #3644 | **A reviewer dictates or clears a universal claim over code sites from probed shapes.** It led to #3619's cap round. |
| 21 | #3640 | **A landing whose PR open or squash-merge the classifier refuses.** It cost #3584 a takeover, a merge-forward and an arbiter re-judge. |
| 22 | #3596 | **A repair round re-creates its predecessor's defect class at sites its own mechanism adds.** It cost #3577's cap round. |
| 23 | #3606 | **A "scrutinize hardest" question a STYLE letter decides reaches handoff.** It cost #3546 a mason repair round. |
| 24 | #3314 | **Assemble brief parts by command.** Sightings 19-21 came in this window (#3623, and two on #3584), and none cost a round. |
| 25 | #3550 | **A body's guard-row premise is taken at base, or marked a hypothesis.** This pass's bodies follow it by hand. |
| 26 | #3590 | **A warden pre-flight's unmarked inference.** |
| 27 | #3547 | **mason.md carries the every-commit-green rule.** It shares mason.md's paragraph with row 22. |
| 28 | #3398 | **A landed `docs/LOG` entry left as a removal and parked.** Four sightings, each an undeletable ref. |
| 29 | #3648 | **One exported xsderr constructor for a delegating charge** (2). It also replaces `countValue`'s old text. |
| 30 | #3551 | **One exported `<group ref>` resolver** (1). |
| 31 | #3533 | **One home for the anonymous-type label** (5). Cliuser 3 asks for the located wording. |
| 32 | #3531 | **`UnfollowedDirective` carries its kind, schemaLocation and namespace** (6). |
| 33 | #3532 | **`LookupNamespace` renamed `LookupPrefix`** (2). |
| 34 | #3446 | **A phrase census that sees a phrase wrapped across comment lines.** |
| 35 | #3393 | **`rulecat` emits no dotted rule IDs.** |
| 36 | #3363 | **Pin doc.go's `"12,50"` exemplar.** |
| 37 | #3286 | **`gencensus` type-checks what it counts.** M9's measure. |
| 38 | #3324 | **Pin the text of `validate`'s remaining unpinned messages.** |
| 39 | #3204 | **A § number cited beside an anchor ID is unchecked.** |
| 40 | #2203 | **`surface` is blind to signature changes.** It decides when a persona round is due. |
| 41 | #3294 | **Retire `xsd`'s planned Walker** (5). |
| 42 | #3214 | **`commentwrap -fix` reflows at the edited line's width.** |
| 43 | #2748 | **`issuefeed -comments`.** It closes #1153 and #1217. |
| 44 | #3273 | **STYLE's `exhaustive` overstatement.** |
| 45 | #3301 | **The third whitespace copy moves onto `value.ApplyWhiteSpace`** (2). |

**Named below the band, on purpose.**
- **M10, `blocked`:** #3588 and #3589 (on #3593).
- **M10, unfiled:** a {type table} (42), mixed content (41), nillable (37), a union two of whose basic members get one Go type (19), a shared expanded name (12), a shared branch shape (11), the min-0 non-emptiable all (8) and an abstract type (3). The next `/backlog` files the largest of them. The xsi:type row (86) is a ruling, not unfiled work.
- **M9 residue:** #3178 (a `<redefine>`d original's shared locals; low). #2977 waits on a consumer. #3179 (`value/doc.go`'s "API frozen").
- **M4:** #3306 (a `<redefine>` original's inline simple type escapes `st-props-correct`; an oracle question first) and #3307 (a doubled walk of a case-2 default type).
- **M8:** #2670 (waits on #1601's ruling reopening).
- **`cmd/goxsd8`'s `usage` constant:** #1433, #1421, #1089, #1626, #1409 (`-q` and a summary), #1432 (no invocation compiles a set) and #1549 (README's `cvc-…` bullet).
- **T5a sweeps:** #2948, #2949, #2950, #2953 (it takes `UnmappedConstruct`'s names), #2954 (it takes #1496's paragraph) and #2955. #2985 goes with #2955 or alone. #1408's README tokens go with it.
- **Marker bookkeeping:** #3019 and #2719 (gapaudit's 8 Group 1 rows). A session in `value/valuespace.go`, `xsd/defaultbinding.go` or `xsd/contentrestricts.go` absorbs its #2719 row.
- **Library contract:** #1382 (the backend-must-match invariant), #1371 (the s4s-grammar classification route), #1454 (rule constants), #1594 (goroutine safety), #1685 (pre-1.0 mobility), #1088 (`ExampleResult` pointer), #1005 (multi-error collection), #1291 (a violation's instance path), #1430, #1094 (a declined {test}'s schema Loc; pairs with row 10), #1234 and #486.
- **Locators and messages:** #1453 (a facet charge at its simple type; the sibling #3580 and #3581 left) and #1851 (a count facet's rule; #3648's Notes order it).
- **M5, `conformance/subtreeroot.go`:** #2237 (24) and #755.
- **Persona documentation, re-sighted:**
  - `validate`: #1595 and #1408.
  - `cmd/goxsd8`: #2419, #1434, #1144, #1185 and #398.
  - Library: #1593.
- **M7:** #2619 (it needs a warden ruling on its `xpath` export).
- **`xpath/cta.go` and `xpath/ctaparser.go`:** #2595, #2675, #2416 and #2533.
- **M6:** #2663 (4) and #2662 share `validate/cvcassertion.go`, as row 8 may. #2455, #2437 and #2394 are in `xpath/assertion.go`; take them one at a time, together with #889.
- **STYLE E4 and E5 sweeps:** #2372 (27), #2373 (16), #2374 (22), #2375 (1) and #3392 (unmeasured), which shares lines with #2372 and #2373. Row 11 edits lines these sweeps count.
- **M5 residue:** #2764, #2494, #1825, #1250 and #920.
- **M4, where `Ratchet: unchanged` is expected or the bound is small:** #2087, #2052, #1873, #1877, #1883, #1887, #1801 and #1777. #1998 and #1953 are in `xsd/contentrestricts.go`, so either one fires #1609's arm 3.
- **Docs and bookkeeping:** #2679, #2624 with #793, #1803, #1881, #785, #1343, #1502 and #656. **`value`:** #721, #2045 and #462. **`regex`:** #2148 and #2147.
- **Tooling, after row 43:** #1897, #2542, #2539, #2605, #2541, #1935, #2070, #2066, #678, #1464, #1838, #1794, #1735 and #839 (`tools/logstats`).
- **Refactors:** #3033 (14/3), #2601 (0), #2600 (2), #2793 (3), #2239 (2), #2359 (14), #2329 (1), #2205 (2), #2101 (1, its first half discharged), and the E5 list above. #3034, #989, #1736, #1701, #845, #1757, #1825, #2416, #2419, #2437 and #3307 are unmeasured.

### Next planning action

1. **#3593's post-land pass unblocks #3588 and #3589** and writes their figures at its landing commit. Each M10 slice's post-land pass re-runs `decodecensus` and restates the moved rows in the open M10 bodies. #3593, #3594, #3630 and #3651 share `contentGroup`, the `codec/doc.go` decline list and Decode item 5, and each later landing rebases.
2. **The next `/backlog` files the largest unfiled M10 row**, which today is a {type table} (42) on R5's ruled shape, unless a post-land pass has filed it already. It also runs a fresh `GOXSD_DECLINES=1` partition, since #3546, #3574, #3580, #3581 and #3618 touched lane code after the last one.
3. **The next persona round is due** when a Tier A slice changes `codec`'s exported surface, or when #3643 or #3622 lands.

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
all attribute-VALUE faults. **#1391 landed the attribute axis they needed**, so
`go tool suiteindex '*@maxOccurs'` and its siblings now census them; #929 and
#606 both still predict `unchanged` for want of a census that is no longer
missing, and re-ranking them on one is owed.

**A schema-construction slice can sit OUTSIDE §5.1's grammar-and-`src-*` frame
entirely, and the expensive case is §5.3.** **That question is now RULED**:
§5.3 governs a failed `·resolution·`, so `Finalize`'s hard-fail is a deliberate
implementation policy choice rather than a spec requirement, and reversing it
costs 35 measured `schema` cases with no ratchet mechanism able to record the
loss — a call CLAUDE.md reserves to a human-filed issue. The ruling is on #1426's
thread; #1426 is closed `not_planned` and **#1498** carries the marker prose that
records it. **#1429** owns the harness guard the attempt broke. **Both missing
built-in component sets have LANDED** — **#1427** on 2026-09-12 (`c1f31b5`,
`schema` +7, `instance` +15) and **#1428** on 2026-09-12 (`cd36ad8`, `schema`
+2). **Neither waited on the ruling, and that is the finding** — the withheld
flips were a missing component set, not a §5.3 deferral, which is now measured
rather than argued.

**Read the milestone count as a floor.** The GitHub milestone holds the feature
slices; the comment-accuracy, doc and process issues that post-land passes file
against the same packages sit outside it — #1135 is M4 work carrying no
milestone today, and #1136 was too until it landed. **The sharpest instance for
LANE MOVEMENT is #1126**, which moved `schema` +475 and `instance` +113 in one
landing and carried no milestone, so the count did not move at all. (**#1411
passed it on the `schema` side on 2026-09-13 with +609**, banking the
`MS-Regex2006-07-15` corpus whole; #1126 remains the largest that moved two lanes
at once.) **The sharpest for SCOPE is #434**: a core §5.3
schema-construction issue, banded as the best lane slice in the queue, carried no
milestone for its whole life and closed outside M4's count.

### M5 — Instance validation (XML) — [epic #250](https://github.com/kud360/goxsd8/issues/250)

`validate` engine plus `validate/xmlsrc`: non-backtracking content matching,
identity constraints, `xsi:type`/`xsi:nil`, wildcards, open content, default
and fixed values. **`instance` lane.** *"Greedy"* is struck here and from
PRINCIPLES 14: #782 replaced the greedy walk with a live SET of partitions,
which `cvc-accept` clause 3.1's existential requires, and the invariant that
survives is no-backtracking rather than greed. **"Bounded" is struck too, and
by #1557 rather than by #782**: arm (b) made `partitionsBounded` an ESTIMATE of
the cover instead of an upper bound on live entries, and what survives
unconditionally is a per-item BUDGET — one item's cost is a constant of the
SCHEMA, never a function of the items already taken.

Carved 2026-08-12 into ten slices, #710–#719, and now **eighteen** — #766 was
split out of #714 by a warden pre-flight, #773, #774 and #775 out of #766 by
another, #782 and #783 out of #715 by its own, **#790** filed from #715's
warden verdict, which had named the seam and been read by nobody who filed it,
and **#1516** for the `{open content}` clauses. **All eighteen have landed.**
The last two, #773 (`c1851ae`, `instance` +9) and #774 (`5fb2cd5`, `Ratchet:
unchanged`), landed on 2026-09-24. #1516, #782 and #783 all landed 2026-09-16/17
and took `instance` 11151 → 11203. Of the thirteen that had
landed before this window, **#719** joined them on 2026-08-27 at `bd887cd`, wiring
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
into the recursive walk (key-sva (§3.3.4.6) clause 3.1) — the largest single
move any lane has recorded; identity constraints and the ID/IDREF table (**#718**,
1017 → 1133); `xsi:type` and `xsi:nil` deciding rather than declining (**#716**,
+183); and a union-governed item classified by its ·validating type· (**#813**,
+9, unioned onto #716's). #913's cvc-type clause 3.1 landing added **9409**,
itself M5. **#1738** (`f6ab5b1`, 2026-09-27) added **10,508**, the largest single
lane move this project has recorded: the lane's first `valid` observation, for a
simple leaf root. Its second slice, **#1808** (`cb0f6e3`, 2026-09-28, +34),
admitted the complex empty leaf root. The third, **#1841** (`aad7b2c`, 2026-09-29,
+2204), admitted an assessed subtree root WITH content. Its successors lift that
gate's exclusions one decision at a time: the ID family (#1857), the content-less
root (#1855, which folded both leaf gates into `assessedSubtreeRoot`), identity
constraints (#1858) and xsi:type (#1859) have landed. Read the rest off the
queue by `conformance/subtreeroot.go`.

**Landings OUTSIDE this milestone keep moving this lane, by four distinct
mechanisms, and a running total of them is not maintained here** — take the
figure from the Status section's table (#646). The four:

- **A slice PRODUCES a component the engine could not previously see** — #733
  (a top-level `<xs:attribute>`'s inline `<xs:simpleType>`), #909
  (`<simpleContent>` `<restriction>`), the CTA pair #842/#851. **#1427 LANDED as
  the next of these on 2026-09-12** (`c1f31b5`) and is the mechanism's cleanest
  instance: §3.2.7's four built-in `xsi:` attribute declarations were seeded as
  COMPONENTS, a schema referencing one stopped hard-failing `src-resolve`, and
  **`instance` banked +15** alongside `schema` +7. **The figure this paragraph
  used to carry — 14 — was an inference transcribed from `wip/issue-434` and was
  WITHDRAWN at grounding** before the landing measured 15; read it as the worked
  example of why a transcribed prediction is re-derived rather than inherited
  (#1332).
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
what the number means.** The lane observes "valid" only for a violation-free
`Result` with nothing unevaluated whose root passes one of the shape gates
`conformance/instance.go` names (#1738, #1808, #1841). Each gate admits only
shapes where the lane gate or the walk decides every applicable clause. Any
other violation-free `Result` DECLINES rather than passing, because `Assess` does
not evaluate `e-validity`'s other conjuncts for it. **Every other passing case is
an expected-INVALID one by construction**, not by measurement, and the failures
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

**Every suite-VALID `instance` case the lane decides wrong has an owner, and the
suite-INVALID ones it decides valid are a class the subtree gate made.** Read the
counts from the Status section's partition, never from this paragraph. The
suite-valid false rejects are #921's (two `gMonth` cases with an unmodeled
`queried` status); #771 decided the ones whose schema documents only the
instance's own `xsi:schemaLocation` names. The suite-invalid
admissions are cases #1841's gate admits on a `value.ValidateLexical` verdict
inside its trust boundary. Each scores Fail, and none is a false pass. The
18 `MS-Regex` cases among them, each holding an astral code point, are #1899's
permanent suite-expectation divergence.

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

What remains of the CTA subset is **#889** (value-level numeric widening, §B.1
rule 1.1's float→double promotion, which #858 withheld rather than faked) and
**#2631**, the cast-target static error a later decline discards. **#894**
(err:XPST0051/XPST0080, the static remainder #886 did not charge) landed
2026-10-14 at `4914723`, `schema` +1 (`s3_12ii06s`).
**#888**, a cast target that is `xs:QName`, was absorbed by the assertion slice
#2587 (`6f714c3`, 2026-10-13). **#859**, the wildcard `ta-AttrName`
arms, **landed 2026-08-18 at `ea0650a`** — this paragraph carried it as
remaining work for fifteen days, arguing about a stale lease on a branch whose
issue had already closed the same day. **#871**, the §3.12.4 clause 1.1.3
·inherited attributes· merge, landed on 2026-09-25 at `2e9ab84`; #1682 (the
`ref=` use-level `{inheritable}` fallback) now has it as a live reader. Neither
of the two still open (#889, #2631) carries a milestone, which is the same
pattern M4's tail records.

**Assertion evaluation is #1042, the GAP tracker, `blocked` on its slices.** It
owns `cvc-assertion` (§3.13.4.1) and `cvc-assertions-valid` (§4.3.13.3) and
retires the `GAP(validate)` markers **#719** landed on 2026-08-27 at `bd887cd`,
one milestone early, because the `instance` lane must decline every case whose
outcome turns on an assertion. #1042 closes when no assertion decline is left.

**Tier 2 is carved one slice at a time.** Landed: **#2232** (`e3490bd`,
`instance` +8, complex-type `cvc-assertion` over the grammar `CompileCTATest`
parses, attributes typed), **#2245** (`08eb2de`, +1, `$value` and the value
comparators), **#2244** (`01bfdab`, +8, xs:untypedAtomic attributes), **#2267**
(`21311ef`, +12, child-axis paths), **#2243** (`3990401`, unchanged, the
implicit timezone), **#2246** (`9f8d73d`, +11, simple-type
`cvc-assertions-valid` through an injected `value.AssertionEvaluator`),
**#2268** (`7fa21b3`, +19, `fn:count` and `.//`), **#2269** (`43661a5`, +17,
arithmetic), **#2295** (`9af63ae`, +15, the F&O string and sequence core),
**#2298** (`645657f`, unchanged, a fixed value over an asserting type),
**#2340** (`0ec64ec`, +8, `fn:exists`/`fn:empty` and child paths `a/b`),
**#2273** (`47ca5b3`, +4, identity and up-the-hierarchy casts), **#2342**
(`ba56979`, unchanged, a union's own facet), **#2322** (`9bf4e48`, unchanged,
·skipped· subtrees in a count) and **#2339** (`4e4e228`, unchanged, float/double
casts now decline), **#2321** (`747b425`, +3, existence EBV), **#2383**
(`354d3d7`, unchanged, a literal cast that decided wrong now declines),
**#2318** (`2328932`, +6, predicates and `|` inside `fn:count`), **#2320**
(`cb842d0`, +2, `count($value)` and `distinct-values`), **#2387** (`c473241`,
+2, a mixed child's string-value), **#2358** (`610f599`, +2, `fn:string` over a
typed date/time), **#2341** (`5b18339`, +2, `if` in a boolean position),
**#2448** (`b8486b1`, +3, `.` and `to`, replacing the parked #2332), **#2451**
(`1a1922c`, +9, `fn:current-date`), **#2450** (`a6e20db`, +7, `castable as`),
**#2452** (`c7b3054`, +2, `instance of` and `fn:data`), **#2472** (`c895157`,
+3, a string-literal sequence), **#2468** (`ec19d7c`, +2, `.` under
element-only content), **#2453** (`bb84f11`, +2, facet `position()`/`last()`),
**#2473** (`e0c02cf`, +2, `namespace-uri`/`in-scope-prefixes`), **#2442**
(`b79e639`, +1, `fn:concat`), **#2445** (`bc6dedb`, unchanged, `N/*` and
`N[a and b]`), **#2319** (`f819aed`, +6, a quantifier over E's children decided
by attribute or next sibling), **#2522** (`2c089be`, +3, a `preceding::` step in
a nested predicate), **#2520** (`265de89`, +2, a quantifier over an atomized
`$value`), **#2462** (`ad2874a`, unchanged, a Type Alternative's cast
declining an assertions facet rather than deciding it at a zero instant),
**#2446** (`6af2d74`, +1, `fn:sum` over an attribute path), **#2521**
(`f6e9811`, +1, a quantifier over a positional child window), **#2477**
(`3e715c7`, +1, `data(a/b)`), **#2476** (`e4d6086`, +1, `child::N` and `N[k]`
behind `fn:data`), **#2478** (`e77235b`, +1, an AttributeTest under `instance
of`), **#2509** (`dda7940`, unchanged, a typed EBV operand decided by
`ctaCarriedType`), **#2579** (`52342d6`, +2, `fn:doc` raises), **#2578**
(`e63cf8c`, +2, a `comment()` step selects nothing), **#2553** (`d1113f8`,
+1, a bound child's own attributes, with `fn:number` and
`fn:substring-after`) and **#2587** (`6f714c3`, +1, a StringLiteral cast to
`xs:QName`, absorbing #888).
#1042's `## Depends on` lists the open slices and the residue not yet filed;
this paragraph does not repeat them.

**#56 LANDED on 2026-08-28 at `3160813`**, ten days after #719 unblocked it and
`Ratchet: unchanged` as its body predicted. It records the CTA compile-time
withhold into `Result.Unevaluated` under `key-cta-ta-select` (§3.12.4), with no
second type and no `Evaluated bool` — the encoding #719 shipped
(`Rule()`/`Loc()`/`Msg()`, `Result.Unevaluated()`), reused
rather than paralleled (STYLE D3), exactly as #842's warden pre-flight had ruled
on D3. **Surface: none new.** STYLE P3's fail-open discipline is only honest if a
fail-open answer is distinguishable from a real pass, and both of this
milestone's fail-open channels — the assertion sites #719 collects and the CTA
withhold #56 records — now reach the same slice under their own rule IDs.

**Its first consequence was a documentation defect, not a code one.**
`validate/doc.go`'s Contract states that an empty `Violations()` beside a
non-empty `Unevaluated()` is not a pass, while README's validation snippet — the
module's only working one, since `validate` ships no runnable `Example` (#1088)
— named `res.Err()` as the sole incompleteness signal. #56's landing is what
made that reachable on an ordinary conditionally-typed document rather than only
on one carrying an assertion. **#1122** landed the fix at `a458893`; the
snippet's remaining wording is #1292's.

### M7 — XPath 2.0 growth

Grammar completion toward full XPath 2.0 plus the F&O function library
(`docs/specs/md/xpath20.md`, `xpath-functions.md`). **`xpath` lane.**

### M8 — JSON instance adapter

`validate/jsonsrc` mapping JSON onto the abstract infoset. **`json` lane**
— curated cases; the W3C suite has no JSON lane.

### M9 — Codegen — done

Deterministic emission, namer, sealed choice sums, multiple schemas to
multiple output dirs, golden-file tests. Its exit sentence (#3234, #3235
and #3243 closed) fired on 2026-11-01, with `go tool gencensus` declining
only the permanent `struct-value recursion`. Open residue is defects and
docs, banded like any product issue.

### M10 — Codec

Runtime path plus generated fast path; differential tests (identical
values, identical error rule IDs) and `testing.AllocsPerRun` budgets.
Generated capability-view readers. The public `value.Emitter` API freezes
here.
Its first slice, #3291, landed `codec.Decode` for README's `order.xsd`
shapes and declines every other construct; #3299 lifted value
constraints and `xs:anyURI`. Its measure is `go tool decodecensus`
(#3292), which decodes the suite's instances into their generated types,
read the way `gencensus` reads M9. Further slices are filed from that
census's first-decline table, largest first, passing over a row a ruling
holds: an xsi:type on a concrete selected type is refused permanently
(`codec/doc.go`; #3291's ruling R5).
codec and codegen each encode the XSD→Go mapping today. #3530 gave it one
home, `internal/goshape`; #3538, #3578, #3540, #3587 and #3557 moved the
splice, attribute-field, shape, flattening and atomic-type rules there, and
#3541 and #3556 move the remaining twins before the generated fast path adds
a third encoding.

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
earlier where a milestone says so — `value.Emitter` at M10.

## Non-goals

- Schema mutation and editing APIs.
- XSD 1.0 compatibility quirks — this is an XSD 1.1 processor.
