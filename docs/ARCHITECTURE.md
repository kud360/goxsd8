# goxsd8 Architecture

## Dependency rule

Packages form a strict DAG. `xsderr` (the error currency) is the **pure
leaf**: it imports nothing from this module. `xsd` (the component model)
imports only `xsderr`; every other package builds above the two of them.
Value implementations, parsing, validation, and generation live above them.

```
                 xsderr          (leaf: errors, rule IDs, locations)
                 xsd             (leaf: component model + query/walk APIs; imports xsderr only)
                 internal/...    (leaves: stdlib-only helpers two packages must agree on
                                  byte for byte, unexportable because they are nobody's API —
                                  internal/schemaloc, the schemaLocation resolver)
                 value           (value-space contracts, facet pipeline; imports xsd, xsderr, regex)
                 value/backendtest (conformance kit for any backend)
   builtin/strict  builtin/native  <user backends>   (implement value contracts)
                 regex           (one engine, XSD + F&O flavors)
                 parser/xmltree  (position-tracking XML; independent)
                 loader          (schema resolution interfaces)
                 parser          (schema docs -> xsd components; imports xmltree, loader,
                                  xsd, value, builtin — and builtin/strict, as the DEFAULT
                                  backend Parse seeds when the caller supplies none)
                 xpath           (XPath 2.0 engine; imports value)
                 validate        (instance validation; adapters xmlsrc, jsonsrc, bersrc)
                 codegen  codec  (generation; dataset ser/de)
                 conformance     (harness + ratchet; test-only)
                 cmd/goxsd8      (the CLI)
```

Nothing imports `conformance`. Nothing in the library imports an adapter's
decoder (`encoding/xml`, `encoding/json`, BER) except that adapter.

## Lexical space vs value space

The load-bearing separation of the whole design (Datatypes §2.1–2.3):

- **Lexical space**: strings. Whitespace normalization and `pattern` facets
  operate here, *before* any parsing.
- **Value space**: typed Go values. Ordering, equality, identity, and all
  value-based facets (`minInclusive`, `totalDigits`, `length` on lists,
  `enumeration`, `assertion`) operate here, *after* the lexical mapping.
- The bridge is the pair of **mappings** per type: lexical → value and
  value → canonical lexical. These are defined normatively as function
  definitions ("hfn") in Datatypes Appendix E, and our builtins are
  bootstrapped from those definitions (extracted from
  `docs/specs/md/xmlschema11-2.md` by the hfn tooling), not
  hand-transcribed.

### The facet pipeline

Validation of a literal against a simple type is a fixed pipeline. The two
stage kinds are named as interfaces (`value.LexicalFacet`,
`value.ValueFacet`), but there is **no composition seam yet**: every
implementation is unexported inside `value`, the function that assembles a
type's stages is unexported, and no exported API accepts or returns a stage.
User-composed stages are a destination, not a shipped capability (steward
audit 2026-08-02) — see the note under "Exported surface without a
consumer" below.

```
raw literal
  → whiteSpace normalization        (lexical; from the type's ws facet)
  → pattern facets                  (lexical; every step of the derivation chain)
  → lexical mapping                 (string → value.Value, via the backend)
  → value facets                    (bounds, digits, scale, length, enumeration)
  → assertions                      (XPath, fail-open; per-item for lists,
                                     per-member for unions, at every level)
```

List and union varieties recurse: lists apply the pipeline per item against
the item type before list-level facets; unions try DirectMembers in order
(not flattened members — intervening restrictions carry facets, and pattern
normalization uses the *validating member's* whiteSpace).

## Builtin types: generated table + pluggable backends

The builtin type system separates **what the spec says** from **how Go
represents it**:

- **`builtin.TypeSpec` table** (generated, data only): name, base type,
  variety, fundamental facets, applicable constraining facets and their
  defaults, for all 49 builtins **including `precisionDecimal`**. Emitted
  from the hfn definitions and per-type property tables in the local
  Datatypes spec by a deterministic generator; byte-identical on
  regeneration; contains no function values.
- **Primitives are the mandatory floor; derived mappings are optional.**
  Derived builtins (`unsignedShort`, `token`, …) are data by default —
  restrictions of a primitive plus facets from the table, inheriting
  operations — so a minimal backend implements ~25 primitive mappings
  (several share a value space; the Gregorian types ride the temporal
  model). A backend may ALSO map derived builtins to give them narrower
  representations; each type's governing mapping resolves by walking up
  the base chain to the nearest mapped ancestor.
- **The widest-space rule.** A derived type's own mapping governs only
  the value the application receives. Inherited facet checks —
  enumeration and bounds declared anywhere on the derivation chain — are
  compared in the declaring type's space via ITS governing (wider)
  mapping, and schema-build restriction checks always run in the base's
  space: a narrower representation must never distort base-chain
  semantics (overflow, collapsed precision, different ordering). A
  lexical that passes the wide checks but doesn't fit the narrow
  representation is a mapping error on that type, never a false validity
  verdict.
- **`value.Backend`** answers `Mapping(typ)` → a
  `value.Mapping{Parse(lexical, ctx), Canonical(v)}` pair. `Parse` takes a
  context because QName/NOTATION need in-scope namespace bindings.
  Comparison, length, digits, scale, identity are **not** backend methods —
  they are capability interfaces discovered on the returned values.
- **`builtin.Seed(backend)`** composes the generated table with a backend
  at schema-construction time; `value.Override(base, partial)` swaps
  individual types (back only `xs:decimal` with a money type, keep strict
  for the rest).
- Two backends are planned; **one ships today**:
  - `builtin/strict` (ships) — spec-exact: arbitrary-precision
    decimal/integer, `precisionDecimal` (coefficient/scale/sign identity,
    NaN/±INF), the 7-property date/time model, XSD-exact float/double
    behavior.
  - `builtin/native` (**M12 — doc-only contract, exports nothing today**)
    — Go-friendly: `int64`, `float64`, `string`, `time.Time`; documented,
    deliberate deviations from the spec value spaces (range limits,
    timezone folding). Its `doc.go` is the normative contract and already
    marks itself planned.
- **Third-party backends are a supported surface.** `value/backendtest`
  is the public conformance kit: `backendtest.Run(t, backend)` drives
  spec-derived vectors (lexical→value→canonical round-trips, order and
  identity cases, the capability set each type's facets require) plus a
  primitive-coverage check. `builtin/strict` passes it in-repo (that is the
  whole of "our backends" until M12); a custom backend that passes it is
  first-class.

## Component model (`xsd`)

- Components are constructed in **phases** so no traversal ever needs a
  cycle check (STYLE D4): (1) parse schema documents into raw form,
  (2) resolve QName references through a symbol table,
  (3) finalize in dependency order — a component's base/item/member types
  are complete before it is. Spec-forbidden circularities are rejected with
  their named rule, once.

  **Where they are rejected is currently split, and that split is not the
  design — it is drift to be repaired** (steward audits 2026-07-26 and
  2026-08-02; tracked as **#271**). The phase-3 home holds the references
  `xsd` stores as QNames: the complex-type base chain
  (`ct-props-correct` cl. 3), the `<group ref>` graph
  (`mg-props-correct` cl. 2), and substitution-group affiliation
  (`e-props-correct` cl. 5), all in `xsd/resolve.go`. But `xsd` stores a
  simple type's `{base type definition}` as a **live pointer**, not a QName,
  and inlines `<attributeGroup ref>` at mapping time with no ref component —
  so those two must be resolved eagerly by the producer, and their
  circularity rejection (`st-props-correct` cl. 2) lives in
  `parser/produce.go`, carrying its own copy of `resolve.go`'s color-map
  idiom. One spec concern, two homes, chosen by per-component representation
  accident; `src-resolve` cl. 1.1 is consequently charged from both packages.

  **The split has since widened, as predicted.** `parser/produce.go`'s
  `buildComplexType` now also charges `ct-props-correct` cl. 3 from the
  producer — the same rule with the same verdict that `xsd/resolve.go`'s
  `checkComplexBaseAcyclic` charges — because demand-driven eager base
  construction would otherwise not terminate. The code documents that as
  two entry points for two construction paths, and on its own terms it is
  right; but the reason a second entry point is needed at all is the
  eager-pointer representation #271 exists to remove. So the count is now
  **two rules charged from both packages**, not one.

  Unifying on the QName-plus-index representation is a pre-1.0 refactor.
  Its original ordering advice was "land before `<list>`/`<union>` (which
  add item/member pointers) and before
  `<import>`/`<redefine>`/`<override>` (which re-point base references)".
  **Half of that window has closed**: `<import>` (#182) and `<override>`
  (#183) landed 2026-07-27 with the refactor still undone. `<list>`/
  `<union>` in the producer are still ahead of it
  (`parser/produce.go` declines them today), so that half of the ordering
  argument is still live and is now the last cheap moment.
- All child collections are slices in document order. Maps exist only as
  internal indexes and never determine any order.
- Nothing derivable is stored (STYLE D3): no effective-facet caches —
  compute `Merge(base.EffectiveFacets(), declared)` on demand; no status
  booleans beside the facts that imply them.
- The model is **read-only** after construction; mutation/editing APIs are
  out of scope. `Finalize` performs exactly **one** mutation before it
  returns — `xsd/attributeusefold.go` materialises §3.4.2.4 clause 3's
  inherited `{attribute uses}` into every complex type (#401). It is a
  property OVERWRITE, not a cache of derivable state (STYLE D3): afterwards
  the producer's partial value is gone rather than kept beside the correct
  one. Read-only means read-only *after* `Finalize`, and there is one write
  site, not a growing set.

### Why the finalize machinery lives in `xsd` (a steward ruling, 2026-08-02)

`xsd` is by far the largest package, and roughly 6,700 of its non-test lines
export **nothing at all**: `derivation.go`, `complexderivation.go`,
`complexextension.go`, `contentrestricts.go`, `defaultbinding.go`,
`elementconsistent.go`, `particleattribution.go`, `effectivetotalrange.go`,
`substitutiongroup.go`, `valueconstraintvalid.go`, `wildcardadmit.go`,
`attributeusefold.go`, `resolve.go`. That looks like a candidate for an
`xsd/finalize` sub-package. **It is not; do not propose the split.**

The constraint machinery reads and writes the components' *unexported*
fields — `attributeusefold.go` reads `ComplexType.prohibitedAttributeNames`
and writes `attributeUses` back. Moving it to a sibling package would force
those fields onto the exported surface (or force a mutator API), trading a
big-but-sealed package for a permanently wider public API and a
representable illegal state (STYLE T1). Go's package boundary is exactly the
tool that keeps the component invariants unforgeable, and the zero-export
files are the evidence it is working, not evidence of a missing seam. Judge
`xsd` by its exported surface and its doc.go contract, not by line count.

### Query and walk

Two access styles over the compiled model, one shared core:

- **Query**: direct lookups — element/attribute/type by QName — exposed
  through minimal capability views (STYLE T3), so a consumer that needs
  only `ElementByName` receives only that.
- **Walk**: traversal of a type's effective content model. The algebra
  ships (type-derivation validity, substitution-group acceptance, wildcard
  admission, attribute-use lookup, all unexported in `xsd`); **the two
  drivers do not exist yet** and `xsd/doc.go` says so by name:
  - a **push** driver — `Walker`, the exhaustive, schema-only visitor of
    every particle reachable through sequences/choices/all-groups and
    named-group references (the codegen consumer) — **M9**, and
  - a **pull** driver — `Matcher`, the instance-guided advance of the
    content model one child at a time (the validation consumer) — **M5**.
  When they land, substitution groups will not be expanded at walk time
  (instance-time concern), and both drivers will reuse the same algebra
  rather than reimplement it. This paragraph is a destination; `go doc
  ./xsd` is the contract.

## Parsing & loading

- `parser/xmltree`: streaming, bounded-memory XML reader that records
  line/column for every node; the origin of every `xsderr.Loc`. No
  `io.ReadAll` (STYLE P4).
- `loader`: the IO seam. `Resolver` answers "give me the schema document
  for (namespace, location hint)"; helpers provided for files, HTTP, and
  in-memory maps, plus a chaining/catalog resolver. `xsi:schemaLocation`
  instance hints route through the same interface so multi-schema loading
  stays in one place. Multiple root schemas load into one set; the loader
  dedupes by resolved location.

- `parser`: the schema-document compiler — the M4 spine, and the only
  writer of `xsd` components. `Parse(location, opts…)` reads the root
  document through one `loader.Resolver`, walks the
  `<xs:include>`/`<xs:import>`/`<xs:override>` closure (§4.2.3, §4.2.6.2,
  §4.2.5) depth-first in document order with a load-once index keyed by
  resolved location, the namespace the document was reached under and the
  override applied to it (document identity, *not* a cycle guard — include
  cycles are spec-legal), applies chameleon coercion to a
  no-`targetNamespace` included or overridden document (§F.1), carries
  override pre-processing (§F.2) as data beside the effective namespace,
  then produces every document into
  one shared `xsd.SchemaBuilder` and finalizes. `Produce(doc, backend)` is
  the single-document entry point and follows no inter-document reference.
  `<redefine>` is skipped, not rejected
  (§3.1.2), so a schema needing it assembles short rather than wrongly.
  The assembly-wide symbol table is seeded with the builtins exactly once
  (`builtin.Seed`), which is why seeding is assembly-scoped and not
  per-document: per-document seeding would re-add `xs:string` per included
  document and trip `sch-props-correct` cl. 2.

  Two properties of this seam are worth stating because consumers depend on
  them: the backend is a caller-supplied `value.Backend` (`Produce` demands
  it explicitly; only `Parse` defaults it to `builtin/strict`, which is the
  one policy edge from `parser` to a concrete backend), and the assembled
  **document set is not reported** — `Parse` returns components, not the
  list of documents they came from. The conformance schema lane needs that
  list to gate every document in a closure, so it re-walks the closure
  itself. Location resolution is no longer duplicated to do so: both walks
  call `internal/schemaloc.Resolve` (#259), so the byte-for-byte agreement
  the harness depends on is structural rather than a comment promise. The
  rest of the walk is still a copy — `assembly.discover`/`.include`/`.fetch`
  and `attrValue`, tracked as **#272** — and it is **growing on schedule**:
  `conformance/schema_closure.go` went from 315 to 396 lines (+26%) between
  2026-07-27 and 2026-08-01 as `<import>`/`<override>`/multi-document gating
  landed. Every composition feature raises the price of that port.

## Regex (`regex`)

One recursive-descent engine translating to Go's RE2, with a **flavor
flag** (PRINCIPLES 10):

- **XSD flavor** (pattern facets): implicitly anchored, `^`/`$` literal,
  non-capturing groups, no flags, `.` excludes `\n` and `\r`.
- **F&O flavor** (`fn:matches`/`fn:replace`/`fn:tokenize`): unanchored,
  real anchors, capturing groups; `i`/`s`/`m` map to RE2 inline flags and
  `x` strips insignificant whitespace before parsing; `q` (undefined in the
  local F&O edition) and any other flag are `err:FORX0001`, and
  back-references — legal F&O grammar but with no RE2 form — are
  `err:FORX0002`, surfaced, never silently accepted.

Character-class handling (`\d \w \p{…}`, subtraction `[a-z-[m]]`) is
shared. The package sits just above the leaves: it imports only `xsderr`
(so its `FORX0001`/`FORX0002`/`src-pattern-value` failures are rule-tagged
per STYLE T2), otherwise stdlib.

## XPath (`xpath`)

Full XPath 2.0 is the destination; the engine grows outward from the
XSD-required subset:

1. the CTA restricted subset (the `test` attribute of `xs:alternative`),
2. assertion essentials — axes, predicates, quantified expressions, typed
   comparisons, the F&O function core,
3. the full grammar and function library, tracked by its own conformance
   lane.

One lexer, one parser, one AST — the evaluator walks the same tree the
static analyzer sees. **Fail-open**: an unsupported construct can never
cause a false rejection; every fallback site is a greppable
`// GAP(xpath): …`. Dynamic errors (type mismatch, bad pattern) make an
assertion definitively unsatisfied — they are NOT fail-open (PRINCIPLES
20). `$value` binds a typed atom `{Lexical, Kind}`. F&O regex functions
use `regex`'s F&O flavor, never the pattern-facet flavor.

## Validation (`validate`)

- Abstract infoset via marker interfaces; sources plug in as adapters:
  - `validate/xmlsrc` — XML instances via `parser/xmltree` (first),
  - `validate/jsonsrc` — JSON instances mapped onto the same infoset
    (schema-aware member classification, scalar shorthand for simple
    content, arrays as repeated elements, null as `xsi:nil`),
  - `validate/bersrc` — BER-encoded instances (last; same infoset, TLV
    decode).
  The engine never imports a source's decoder; adapters build infoset
  values and hand them over.
- Content-model matching is greedy and deterministic (UPA makes
  backtracking unnecessary); explicit content beats open-content
  wildcards at the current state.
- Streaming-oriented; parent element context is threaded from day one
  (ID/IDREF harvesting, EDC's post-`xsi:type` governing type, namespace
  context for identity constraints).
- Every violation is an `xsderr.Error` with a cvc rule ID + instance
  and/or schema location, reported in document order.

## Codegen & codec

**Status: neither package ships anything.** `codegen` (M9) and `codec`
(M10) are `doc.go`-only today — `go doc` renders no exported identifier for
either, and `value.Emitter` does not exist (its API is frozen for M9, not
declared). Everything below is the destination, stated in present tense
because it is a design contract; read it as "will", not "does" (steward
audit 2026-08-02).

- `codegen` emits Go types from a compiled schema, deterministically
  (D1/D2). Multiple schemas map to multiple output directories — one
  package per (schema set, target dir) pairing declared by the caller.
- **Type narrowing in interfaces** is the generated-code idiom:
  - **Choices are sealed interfaces.** An `xs:choice` becomes an interface
    with an unexported marker method; each branch is a concrete type
    implementing it, and consumers use type switches. This is the
    closed-sum exception to STYLE T2: exactly one branch can exist, so
    "N pointer fields, exactly one non-nil" never appears in generated
    code.
  - Generated readers/views expose the narrowest interface a consumer
    needs; optionality and nillability are modeled in types, not comments.
- **Anonymous types get ancestor-context names.** A single namer component
  owns all XSD-name → Go-identifier decisions. Anonymous types are named
  by walking up their schema ancestors to the nearest named declaration
  (element `shipTo` under element `purchaseOrder` → `PurchaseOrderShipTo`),
  extending the path only as far as uniqueness requires; residual
  collisions (case folding, Go keywords, XML-legal-but-Go-illegal names)
  are disambiguated deterministically by document order (D1/D2). Every
  generated type's header comment records its schema Loc + original QName.
- `codec` is the dataset serializer/deserializer: schema-directed decode of
  instance documents into generated (or reflective) Go values and canonical
  encode back out.

### Two decode paths, one semantics

`codec` is built for **minimal allocation**:

- **Runtime path** (always available): the facet pipeline +
  `value.Mapping`, driven by the compiled schema. General, reflective,
  allocation-tolerant.
- **Generated fast path** (planned): backends will export **code emitters**
  (`value.Emitter` — API frozen in M9, **not yet declared**; to be
  implemented by `builtin/strict` and `builtin/native`, and by user
  backends that want it). At codegen time the emitter
  contributes specialized decode/encode code for its types — parsing
  directly from the reader's byte window into the target field, no
  intermediate string, no boxed `value.Value`, facet checks inlined.
  A backend without an emitter simply falls back to the runtime path for
  its types.
- Runtime hot-path APIs will follow the appender convention
  (`AppendCanonical(dst []byte, v) []byte`, `ParseBytes(b []byte)`) so
  even the non-generated path can be allocation-frugal. **Neither method
  exists yet**; the convention is named here and in `codec/doc.go` so the
  first implementation does not invent a second spelling.

The two paths must implement the *same* pipeline stages with the *same*
spec rule IDs, which is what will make them **differentially testable**:
for every type, property tests feed identical input to both paths and
require identical values and identical error rule IDs, and
`testing.AllocsPerRun` benchmarks pin the fast path's allocation budget. A
fast path that disagrees with the runtime path is wrong by definition.
(No such test exists today — there is no second path to differ from.)

### Debuggability of parsing

When a value fails to parse, the error must localize the failure without
a debugger (extending E1–E3). These are **M10 requirements on `codec`**,
not descriptions of shipped behaviour — `GOXSD_DEBUG=codec` traces nothing
today because `codec` is empty:

- every decode error carries the **pipeline stage** that rejected
  (whitespace / pattern / lexical-map / facet / assertion), the type
  QName, the offending input fragment, and the instance Loc + byte offset;
- `GOXSD_DEBUG=codec` traces stage transitions per value through the
  injected slog logger (rule ID, type, input) so an agent can watch one
  value flow through the pipeline;
- generated code preserves this: emitted fast paths report the same
  stage/rule metadata as the runtime path, and generated files map cleanly
  back to the emitting backend and schema construct (a header comment per
  emitted decode function naming type QName + schema Loc).

## Exported surface without a consumer

STYLE T5 is the rule ("export nothing without a consumer"); this is the
standing list of places where the surface currently outruns it, so each
audit re-checks the same set instead of rediscovering it. Everything here
compiles, is documented, and has **zero** callers module-wide.

- **The annotation subsystem** — `xsd.Annotation`/`AppInfo`/`Documentation`/
  `Attr` plus their constructors and accessors, `SchemaBuilder.AddAnnotation`,
  a trailing `annotations []Annotation` parameter on **13** component
  constructors, and `parser`'s `Text`/`Node` character-data retention that
  exists (per `parser/tree.go`) to round-trip `<xs:documentation>`. No
  producer builds an `Annotation`: all 36 `parser` call sites pass `nil`,
  and no accessor is read. No milestone in docs/PLAN.md owns populating
  it. Filed for a decision — populate it or unexport it — because the 13
  trailing positional slots are #405's "last slot to wave through"
  tripwire, once per constructor.
- **`value.LexicalFacet` / `value.ValueFacet`** — the two pipeline-stage
  interfaces. Every implementation is unexported inside `value`, the
  assembling function (`compile`) is unexported, and no exported API takes
  or returns a stage, so no consumer can exist. Either grow the composition
  seam the docs promise, or unexport the interfaces.
- **`regex.FlavorFO`** and **`loader.FS`/`HTTP`/`Chain`** — no consumer, but
  both are **justified and not to be filed**: the F&O flavor is fully
  implemented against its M6/M7 consumer (`xpath`), and the resolver
  helpers are declared library surface in `loader/doc.go` for external
  users, which is the "documented contract it fulfills" half of T5.
  Re-check, do not re-file.
- **`xsderr.IsValidRule`** — zero non-test consumers, and `xsderr/doc.go`
  claims a module-wide test enforces it that does not exist. Tracked as
  **#273**, together with the 63 sites (50 in `builtin/strict`, 13 in
  `value`) that still pass bare string literals where the rest of the
  module uses typed `xsderr.Rule` constants.

## Conformance & ratchet

- W3C suite at `testdata/xsdtests` (submodule, pinned).
- Expectations committed at `conformance/testdata/expectations/*.txt`, one
  line per test case, one lane per file (`datatypes`, `schema`, `instance`,
  `xpath`, `json`, `ber`); diffs make regressions obvious and `git blame`
  bisectable.
- `go test ./conformance -run TestConformance -count=1` compares;
  the same command under `GOXSD_RATCHET=1` re-baselines **upward only**.
  A regression fails loudly and must never be committed.

### Cohort isolation is deliberate (a steward ruling, 2026-07-19)

`conformance/datatypes.go` claims its cases as separate **cohorts** —
lexical, `<item>`, QName/NOTATION-context, Facets, NOTATION-Facets,
Saxon `PDecimal` — each with its own reader/decoder/executor triple and
its own XML decode structs (`lexicalInstance`, `itemInstance`,
`facetsInstance`, `notationStep`, …). This is isolation-over-DRY **by
design**, not accreted duplication, and future audits should leave it be:

- Each triple decodes a *distinct, static* W3C fixture shape. Those shapes
  are frozen input data, so the triples never have to change together — the
  upkeep coupling that makes duplication expensive is absent.
- All the actual datatype *semantics* funnel into shared library entry
  points (`value.ValidateLexical`, `value.Mapping`); a spec or pipeline
  change flows through the library, not through N decoders. Genuinely
  shared harness machinery (`childBindings`, `nsContext`, `facetChild`,
  `buildOwnFacets`) is already factored out and reused.
- The separation is a **regression firewall** serving the one rule: a new
  cohort (a new issue claiming a new suite directory) adds a triple without
  touching landed ones, so it cannot silently regress the ratchet.

The cost of merging the triples into one "universal" decoder — coupling
unrelated fixture shapes so one cohort's change can break another — exceeds
the cost of the parallel shapes. Re-flag only if a semantic rule ever has
to be edited in two triples to stay correct.

## Logging

`log/slog` injected at construction, namespaced groups, silent by default.
The debug level is designed for agents: messages carry rule ID, component
QName, and location so a conformance failure can be localized from logs
alone (`GOXSD_DEBUG=parser,validate` in tests).
