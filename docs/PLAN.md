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

## Status — 2026-10-01 (post-land restamp after #2063, on `main` at `b2f7da4`)

**One landing since the last stamp (`05f91cf`).** **#2063** landed at `b2f7da4`.
`rootStart` now admits a DOCTYPE whose DTD can default no attribute: no
ExternalID, and no `<!ATTLIST` or `%` in the internal subset. **`instance` +14**
(`Id` id017–id021, `ID_IDREF` s3_3_4v04i and s3_3_4v05i), exactly the grounding's
measured prediction; the filed bound of 12 was corrected at grounding. M5's
north star moves to **#2071**. **Lean: product, unchanged.**

**Two post-land passes are still not on `main`:** #1816's (open PR #1871) and
#1908's (open PR #1929). #1823's LOG entry still has no branch. #2007 owns all
three.

### Conformance lanes

**This table is `go tool lanestatus`, pasted verbatim, on `main` at `b2f7da4`.**
It is the committed expectations census, which `docs/WORKFLOW.md` names as the
lane score (#1120).

| Lane | Pass | Fail | Total |
|---|---:|---:|---:|
| `ber` | — | — | 0 |
| `datatypes` | 1163 | 10 | 1173 |
| `instance` | 25957 | 404 | 26361 |
| `json` | — | — | 0 |
| `schema` | 15275 | 123 | 15398 |
| `xpath` | — | — | 0 |

**`instance` moved +14; nothing else moved.** **`datatypes ⊆ instance` is
intended** (#1507). **An em dash means a lane with no cases yet, not a lane
scoring zero.** `datatypes` is M3 and complete. `schema` is M4 and `instance` is
M5, and both are active. `xpath`, `json` and `ber` wait on M6/M7, M8 and M11.

### What holds each active lane's failures

**Measured on `b2f7da4`** (suite `7bc3365`), with one read-only
`GOXSD_DECLINES=1` conformance run fed to `go tool lanepartition -log <run>
<lane>`. Every figure is a part of the banked fails and a bound from above,
never a prediction of flips. The tool's owner matching is lexical, so the owners
below are this restamp's reading.

| lane | banked fail | declined | indeterminate (#277) | decided against the suite |
|---|---:|---:|---:|---:|
| `instance` | 404 | 371 | 5 | 28 |
| `schema` | 123 | 11 | 11 | 101 |
| `datatypes` | 10 | 6 | — | 4 |

- **Recorded PRINCIPLES 25 divergences stay banked `fail` and have no work to
  take, so no `/backlog` names them a north star:** `schema`'s **`MS-Wildcards`
  49 (#1809)** and **#1988's 6** (`Simple` simple090–simple094 and
  `Override/over027`), and `instance`'s **`MS-Regex` 18 (#1899)** and **#1912's
  7**. Bugzilla 4113 leaving `queried` reopens #1899's ruling. Subtract #1912's
  7 (`xsd021.n02`, `pscontents00102m1` and `pscontents00102m2` Negative,
  `attgD024.i`, `ctL020.i`, `wildO006.i`, `wildP002.i`) from any declined
  cluster they fall in.
- **M5's north star is #2071: cvc-complex-type clause 5 for a wildcard child.**
  `Wild` declines 17 (10 invalid, 7 valid), the largest cluster outside M6's
  XPath that no open issue names. `resolvedChild` refuses a strict or lax
  wildcard's child whose ·locally declared type· may be non-·absent·
  (`locallyDeclared`), because the walk never checks clause 5. **Bound 17,
  measured** by a scratch probe that forced `locallyDeclared` false: 8 flip on
  the lift alone (5 `Wild`, wildZ003.v, addB135.v, s3_10_1v04i), and 9
  suite-invalid cases (8 `Wild`, s3_8_6v01i) turn decided-wrong unless the walk
  charges clause 5. So the lift does not land without the charge. The other 4
  `Wild` declines (wild062.n3, wild063.n2, wild063.v2, wild082.v1) are
  unattributed.
- **M4's north star is #2067: `fabricatedRejection`'s read arm.** `schema`
  declines 11, 10 of them suite-invalid. **schB4, schE5 and schH5** are the one
  cluster with one decision: each schemaLocation resolves to not-well-formed
  XML, and the read arm declines any failure beside an unreadable document as a
  possible reader limitation. Bound 3. `iri-001`, the one suite-valid decline,
  is an unreadable `<include>` too and is #2067's over-lift guard. The other 7
  (groupB007, mgB008, mgB010, mgP058, particlesZ009, xsd003-1.e, xsd003-2.e)
  carry no src-resolve error, and their refusing arm is unprobed.
- **The XPath clusters are M6's:** `Assert` 65 valid and 45 invalid,
  `assertion` 24 and 26, and `CTA` 29. #1042 owns them and is `blocked`.
  `TypeAlternativeTests` 11 valid declines are unowned and read as the same
  dependency.
- **`VC` declines 18 valid and 10 invalid** (#1002, #1042, both `blocked`).
- **`MS-Additional` declines 28** (16 invalid, 12 valid). **#2058** holds
  addB138, addB139 and addB063: each hinted schema is, or once #2016 lands will
  be, rejected as `xml-wf`, and the case waits on an oracle ruling for how that
  is scored. addB135.v is #2071's. The hint-less groups stay declined under
  #2025's ruling. **`MS-Attribute` declines 11 invalid** (attMd001–attMd011),
  also #2025's.
- **`MS-IdentityConstraint` declines 14** (9 invalid, 5 valid). #2062 owns
  `idF018.i`: the walk declines cvc-identity-constraint clause 3 for a field
  node of a determined complex type. The other 13 are unattributed, a manual
  probe until #2008 lands.
- **`IRI` declines 12** (6 and 6), every one in group `iri-001`, whose schema
  document is #2067's unreadable `<include>`. **`Id` declines 2** (id022.v01 and
  id022.n01, no DOCTYPE), unattributed.
- **#2053's residue is owned:** `elemZ033b.v` is #1978's named candidate (the
  `MS-Element` 3 valid left), and the lax `GAP(conformance)` at `nilValue` is
  #2061's (fail-closed, suite-invalid only).
- **#771 owns the hint-only shape:** its four decided false rejects and the two
  suite-valid declines `attgD034.v` and `ctL021.v`.
- **`MS-Particles` 7 decided are not a north star.** Five of them
  (particlesOb001/002/004/008/009) are exactly the cases where a lax wildcard
  restricts a strict one. This restamp reads them as held by the
  strict-over-lax keyword remainder that `xsd/contentrestricts.go`'s ·default
  binding· marker records as **RULED permanent by #345** (fail-open). That
  reading is unprobed. Next come `MS-Schema` 6 indeterminate and `Open` 5
  (#1374). #1379 carries a measured `schema` +2 (addB108, attO025).
- **Every false reject has an owner, unchanged:** `schema`'s 5 suite-valid
  decided cases (#462, #921, #1002) and `instance`'s 6 (#771's four and #921's
  two).
- **`datatypes`' 10** are 6 declines and 4 decided (#1848, #1794, #594),
  unchanged.

### Branch namespace, `origin` (report-only; a session never deletes a ref)

`go tool wipsurvey` was fed this restamp's issue walk (`rows 2069 distinct 2069
min 1 max 2069`, no gaps) in a full clone (unshallowed this pass). It prints:

- **No `wip/issue-<N>` lease is LIVE.** `wip/issue-2063` was deleted at merge.
- **`wip/issue-2013` is RETIRED**, because #2013 is closed `not_planned`. Its
  content is on `main` as `6e5a545`, and a human may delete it.
- **`meta/post-land-1908`** (PR #1929, 1 ahead) and **`meta/post-land-1816`**
  (PR #1871, 1 ahead) are open post-land passes. #2007 owns landing them.
- **`meta/backlog-2026-10-01-b`** was squash-merged by PR #2011 as `a0adf07`.
  #1897 owns the survey gap that calls it STRANDED, and a human may delete the
  branch.
- **`chronicler-345`** (4 ahead), **`parked/untriaged-20260930-215455`** and
  **`parked/untriaged-20260930-110432`** are unchanged, and a human may delete
  all three after triage.
- `wip/issue-1861-mason`, `wip/issue-1861-chronicler`, `wip/issue-1926-mason`,
  `wip/issue-1926-chronicler`, `wip/issue-1988-mason-repair` and
  `wip/issue-1988-chron2` are skipped because they do not match
  `wip/issue-<N>`. All three issues are closed and landed, so a human may
  delete all six.

### Marker census

`go tool gapaudit`, fed the same walk, reports **81 markers across 10 areas, 16
in group 1 and 34 in group 2, with zero dead ends.** #2063 added and removed no
marker. Group 2 loses #2063 and gains #2067. #2071 was filed after the walk; it
is a conformance-lane gap, and like #2067 it carries no marker by design.

- **The group-1 rows are unchanged.** `conformance/subtreeroot.go`'s `nilValue`
  marker is #2061's and `resolvedChild`'s is #1978's; #2063's insertion moved
  both line numbers. `xsd/defaultbinding.go:491` and `:688` are #1379's, which
  does not yet cite them. The others include the `parser/doc.go:228` §5.3
  policy row, `xsd/resolve.go:723`, and three `validate` rows whose owners do
  not yet cite them (`assess.go:1113` #1892, `cvcid.go:258` #1093,
  `cvcidentityconstraint.go:131` #1887).
- **`conformance/schema.go:401-404` restates the `parser/doc.go:228` gap with
  no marker beside it.** #785 owns tying it to that marker.
- **`GAP(value): narrowed primitive mappings`** on `ConstraintMatches`' doc
  cites #2045 and so sits in neither group.

### Milestones and queue

**Counted from the REST walk above, plus #2070 and #2071: 331 open and 740
closed.**

| milestone | open | closed | state |
|---|---:|---:|---|
| M1 — Spec infrastructure | 0 | 3 | done |
| M2 — Foundation leaves | 0 | 5 | done |
| M3 — Datatypes vertical slice | 0 | 12 | complete |
| **M4 — Schema parsing** | **64** | **169** | active |
| **M5 — Instance validation (XML)** | **21** | **79** | active |
| M6 — XPath required subset | 1 | 0 | not started |
| M7–M12 | 0 | 0 | not started |

Open issues carry **317 `ready`, 12 `blocked`, 0 `needs-replan` and 2
`epic`**, which sums to 331. `kind/gap` is 64.

- **`blocked` is 12:** #16, #555, #1002, #1042, #1051, #1374, #1609, #1790,
  #1880, #1885, #1923 and #2022. None names #2063. **#1609 is unfired:**
  `schema` Pass reads 15275, not above 15292, its Total is unchanged, and
  `b2f7da4` changes no `xsd/` file. **Five human decisions are outstanding:**
  #1880 (which gates #1002), #1885, #1790, #1923 and #2022.
- **Measured refactors, re-run on `b2f7da4`, all flat:** #2041 has 3
  definitions of the ·special· identity test, #1958 2 definitions, #1865 2
  `importWording` values, #1757 1 arm, #1770 2 `isNotationName` lines (1 call
  and the definition) and #363 2 copies. None enters the band on its figure.

### Persona consultations: not re-run this pass

**The cartographer role-plays no persona and does not spawn one** (#416).
Nothing was handed to this restamp. #2063's surface is none, so the published
surface has not changed. **Eighteen persona findings stay open and
unconsumed:** #1568–#1571, #1593–#1596, #1626, #1684, #1685, #1687, #1688,
#1843, #1845, #1894, #1895 and #1898.

### Working band

This is ordered for a `/develop` session: take the highest row you can start.
**Run `wipsurvey` fed, before starting**, because this is a snapshot and an
unfed run cannot print RETIRED. Each row names one issue (#1636). **Rows 1, 6
and 7 share `conformance/subtreeroot.go`**, also with #2061, and rows 1 and 6
edit the same function, `resolvedChild`; the session that takes one absorbs
another when it can. **Rows 3 and 4 share addB063**: #2058 decides how it is
scored and #2016 makes its schema not well-formed, so a session can take both
under WORKFLOW's Scope rule. **Rows 2 and 3 meet at `xml-wf`:** #2058's oracle
ruling on a hinted schema rejected as `xml-wf` may bind #2067's read arm too.

**The ordering principle for this stamp:** each milestone's north star first,
then the named bounds and the reproduced false accepts, then the two process
fixes whose sightings cost a round.

| # | issue | why here |
|---|---|---|
| 1 | #2071 | **M5's north star.** The walk charges cvc-complex-type clause 5 for a wildcard-attributed child whose ·governing type definition· is not ·validly substitutable· for its ·locally declared type·, and `resolvedChild`'s `locallyDeclared` refusal is lifted. `instance` **bound 17, measured** (8 flip on the lift, 9 need the charge). Same file as rows 6 and 7. **Startable now** |
| 2 | #2067 | **M4's north star.** `fabricatedRejection`'s read arm decides a composed document that is genuinely not well-formed and still declines a reader limitation. `schema` **bound 3** named (schB4, schE5, schH5), with `iri-001` as the over-lift guard. **Startable now** |
| 3 | #2058 | **M5, #2015's follow-up.** An oracle ruling on how a hinted schema rejected as `xml-wf` is scored, then `execInstanceCase` decides that `perr` while still declining a resolver fault. `instance` **bound 3** named (addB138, addB139 now, addB063 after #2016). **Startable now** |
| 4 | #2016 | **M5, a parser false accept.** Character data after the document element is rejected as `xml-wf`, and `assembleHints`' undeclared-root decline is lifted or narrowed. `Ratchet: unchanged` on its own; it carries addB063 into #2058's reach, and absorbs #2063's `instancehints_test.go` comment fix. **Startable now** |
| 5 | #1379 | **`schema` lane (no milestone), a measured bound.** The `defaultbinding.go` fixed-value residuals; decline 1's ·special· pair takes #2040's RULING with `schema` +2 measured (addB108, attO025). Same file as #2041 and #2045 (`value/valuespace.go`). **Startable now** |
| 6 | #1978 | **M5.** Admit a strict or lax wildcard's unresolved child whose xsi:type resolves, and retire `resolvedChild`'s group-1 marker. One named candidate, `elemZ033b.v`; otherwise unmeasured. Same file as rows 1 and 7, same function as row 1. **Startable now** |
| 7 | #2008 | **M5 tooling that names the next north star.** An `instance` decline names its refusing `subtreeGate` arm. `MS-IdentityConstraint`'s 13 unattributed declines, `Wild`'s other 4 and `id022` each still need a manual probe without it. Same file as rows 1 and 6. **Startable now** |
| 8 | #2026 | **Process: its first sighting cost a round and parked an accepted change (#2013).** `/develop` step 5 runs `go vet ./...` on a merged-forward tree before delegating any round. **Startable now** |
| 9 | #1989 | **Process: seven sightings, one of which cost the oracle a pass (#446).** `/develop` step 3 populates `testdata/xsdtests` before the oracle runs. **Startable now** |

**Named below the band, on purpose.**
- **#2063's follow-up:** #2070 (`suiteindex` cannot census a DOCTYPE or its
  internal subset, so a DTD measurement falls back to a byte grep that misses
  the UTF-16 fixtures; second sighting of #1206's class, no round lost). Same
  tool as #1536, #1794 and #1464.
- **#2054's follow-ups:** #2066 (`commentwrap` skips a list item's second
  paragraph as verbatim; it reflows the two `conformance/schema.go` lines that
  then report; same file as #678) and #785's item 6 (the unmarked §5.3
  sentence).
- **#2053's follow-ups, both M5 `kind/gap`:** #2061 (the walk charges nothing
  for an xsi:nil with no ·actual value· on a declaration-less element; it
  retires `nilValue`'s marker and can flip only suite-invalid cases) and #2062
  (`elementKeyMember` declines clause 3 for a determined complex field node;
  bound 1, `idF018.i`, after an oracle ruling).
- **M4, #1982's follow-up:** #2052 (a foreign child under a compositor,
  `xs:union`, or a group or attributeGroup ref is still skipped). Hypothesis,
  unreproduced; its ratchet is unmeasured.
- **`parser/produce_s4sorder.go` doc fixes:** #1343 also carries #1982's
  STYLE D3 finding (the two `##other` facts stated once, at `s4sSlot`'s doc).
  Comment-only.
- **Process, for the `/retro` on 2026-10-04:** **#1868 and #1948 first** (with
  #1798 and #1781, the same class), then #1812 (the arbiter's ratchet duty in
  step 5), #2021 (the oracle's spec scope), #1990, #1993, #1996, #2010 and
  #2047 (the Survey input recipe under the isolation guard and the permission
  system; its remedy is restated to literal one-page `gh api` calls), then
  #1913 and #1791. #1880, #1885, #1923 and #2022 wait on the owner.
- **#2006's trackers:** #2018 (5th-edition Names, xv001.v01 and xv004.v01) and
  #2020 (nine `strings.Fields` splits over list lexicals).
- **Tooling:** #2070, #2066 and #678 (`commentwrap`), #1935 (casejoin misses
  include/import-reached fixtures), #1897 (STRANDED PASS misses post-land
  heads), #1464, #1838 and #1794.
- **M4 in `xsd/contentrestricts.go`, `Ratchet: unchanged` expected:** #1998
  (the keyword-half false reject) and #1953 (`coveringWildcardUnion`'s
  residual). Both are `ready` and startable, and either one fires #1609's arm 3,
  so its session owes the series point.
- **M4 with a small bound or `Ratchet: unchanged` expected:** #1873, #1877,
  #1883, #1887, #1820, #1801 and #1777. **Conformance bookkeeping:** #1803,
  #1881, #785 (`conformance/schema.go`'s stale element bullet and the unmarked
  §5.3 sentence), and #458 (comment fixes, including `conformance/schema.go`'s
  four stale "UTF-16 not yet implemented" sites). **#1987** is an unmeasured
  message refactor in `value`.
- **`value`, `Ratchet: unchanged`:** #2045 (a narrowing primitive `Override`
  makes #2040's ·special· branch fail-CLOSED; strict narrows nothing). Same
  file as #2041 and #1379.
- **`internal/xmlenc`, both low priority:** #361 (BOM-less UTF-16 and other
  `EncName`s, the leaf's `GAP(xml)` owner) and #363 (the §4.3.3 message's 2
  copies across two packages, now also pinning `Mark.String`'s byte order).
- **M5 rulings and residue:** #1856 (`MS-Regex` 2), #1892, #1093, #1848,
  #1849, #1824, #1825, and #771, whose four false rejects and two hint-only
  declines #2025 does not reach.
- **Persona docs:** #1843, #1895, #1894 and #1898. **#1283 follow-ups:** #1864
  and #1865.
- **Refactors.** #2041, #1958, #1865, #1770, #1757 and #363 are measured and
  flat at `b2f7da4`. #2041 needs a warden pre-flight on the `xsd` export it
  adds and rebases onto `d276f4b`'s `value/valuespace.go`. #1770 is the route
  #1765 and #1745 should take. #1735, #1736, #1701, #848 and #845 are
  unmeasured. #1736 also renames `inlineDeclarationMatchesName`
  (`xsd/wildcardadmit.go`). #755 owns the single home of the xsi hint reader,
  which has three copies, and its body says where they disagree.
- **#1790** waits on a human.

### Next planning action

1. **The orchestrating session lands or supersedes the open post-land passes
   (#2007):** PR #1929 (#1908), PR #1871 (#1816) and #1823's missing LOG entry.
   A human triages both `parked/untriaged-20260930-*` branches,
   `chronicler-345`, `meta/backlog-2026-10-01-b`, the six subagent branches
   under `wip/issue-1861-*`, `wip/issue-1926-*` and `wip/issue-1988-*`, and
   `wip/issue-2013`.
2. **#2071's grounding asks the oracle** how ·validly substitutable·
   ·without limitation· reads for a wildcard child whose ·locally declared
   type· comes from a base type (key-ldt-elem case 3), and whether the walk
   needs an `xsd` export to evaluate it (a warden pre-flight if so). Its bound
   of 17 is probed; the 9 suite-invalid cases are the lift's regression guard.
   **#2067's grounding probes whether the read error tells `xml-wf` from an
   encoding refusal**, and what `iri-001`'s read error is. **#2058's grounding
   asks the oracle first**: its bound of 3 means nothing until the outcome is
   ruled.
3. **The next `/retro` is Sunday 2026-10-04.** It reads four windows since the
   last retro: 73.7 cases per session at 89% product, 177.7 at 100%, 44.4 at
   90%, and 10.0 at 90% with two repair rounds. It also reads these landings:
   #1911's at 132 cases, #2006's at 25, #2013's park after an accepted change,
   #1979's first-round accept at 62, #1988's, #1899's and #1912's no-code
   divergences, #2025's first-round accept at 15, #2019's at 17, #2029's at
   10, whose doc-only revise was repaired after the verdict, #2040's
   first-round accept at 14, #2000's first-round accept at 0, #1982's accept at
   0, #2015's first-round accept at 0 after grounding re-stated a false +2
   bar, #2053's first-round accept at 38, #2054's at 4 and #2063's at 14,
   each exactly its measured prediction. It rules #1868 and #1948 together
   with #1798 and #1781, then #1812, then the process filings #1989, #1990,
   #1993, #1996, #2010, #2021, #2026 and #2047.
4. **Human decisions outstanding:** #1880, #1885, #1790, #1923 and #2022.

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
into the recursive walk (§3.3.4.6 clause 3.1) — the largest single move any lane
has recorded; identity constraints and the ID/IDREF table (**#718**,
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
suite-valid false rejects are #771's (led by schema documents that only the
instance's own `xsi:schemaLocation` names, which the harness never assembles) and
#921's (two `gMonth` cases with an unmodeled `queried` status). The suite-invalid
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

What remains of the CTA subset is **#888** (a cast target that is `xs:QName`),
**#889** (value-level numeric widening, §B.1 rule 1.1's float→double promotion,
which #858 withheld rather than faked) and **#894** (err:XPST0051/XPST0080, the
static remainder #886 did not charge). **#859**, the wildcard `ta-AttrName`
arms, **landed 2026-08-18 at `ea0650a`** — this paragraph carried it as
remaining work for fifteen days, arguing about a stale lease on a branch whose
issue had already closed the same day. **#871**, the §3.12.4 clause 1.1.3
·inherited attributes· merge, landed on 2026-09-25 at `2e9ab84`; #1682 (the
`ref=` use-level `{inheritable}` fallback) now has it as a live reader. None of
the three still open (#888, #889, #894) carries a milestone, which is the same pattern M4's tail
records.

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
rather than paralleled (STYLE D3), exactly as #842's warden pre-flight had ruled
on D3. **Surface: none new.** STYLE P3's fail-open discipline is only honest if a
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
