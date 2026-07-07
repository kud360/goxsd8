# goxsd8 Architecture

## Dependency rule

Packages form a strict DAG. `xsd` (the component model) and `xsderr` (the
error currency) are **pure leaves**: they import nothing from this module.
Value implementations, parsing, validation, and generation live above them.

```
                 xsderr          (leaf: errors, rule IDs, locations)
                 xsd             (leaf: component model + query/walk APIs; imports xsderr only)
                 value           (value-space contracts, facet pipeline; imports xsd, xsderr)
                 value/backendtest (conformance kit for any backend)
   builtin/strict  builtin/native  <user backends>   (implement value contracts)
                 regex           (one engine, XSD + F&O flavors)
                 parser/xmltree  (position-tracking XML; independent)
                 loader          (schema resolution interfaces)
                 parser          (schema docs -> xsd components)
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

Validation of a literal against a simple type is a fixed pipeline; each
stage is a value users can compose for their own types:

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
- **Only primitives carry code.** Derived builtins (`unsignedShort`,
  `token`, …) are pure data — restrictions of a primitive plus facets from
  the table; they inherit the primitive's operations. A backend implements
  ~25 primitive mappings, and several of those share a value space (the
  Gregorian types ride the temporal model).
- **`value.Backend`** answers `Mapping(primitive)` → a
  `value.Mapping{Parse(lexical, ctx), Canonical(v)}` pair. `Parse` takes a
  context because QName/NOTATION need in-scope namespace bindings.
  Comparison, length, digits, scale, identity are **not** backend methods —
  they are capability interfaces discovered on the returned values.
- **`builtin.Seed(backend)`** composes the generated table with a backend
  at schema-construction time; `value.Override(base, partial)` swaps
  individual types (back only `xs:decimal` with a money type, keep strict
  for the rest).
- Ships with two backends:
  - `builtin/strict` — spec-exact: arbitrary-precision decimal/integer,
    `precisionDecimal` (coefficient/scale/sign identity, NaN/±INF), the
    7-property date/time model, XSD-exact float/double behavior.
  - `builtin/native` — Go-friendly: `int64`, `float64`, `string`,
    `time.Time`; documented, deliberate deviations from the spec value
    spaces (range limits, timezone folding).
- **Third-party backends are a supported surface.** `value/backendtest`
  is the public conformance kit: `backendtest.Run(t, backend)` drives
  spec-derived vectors (lexical→value→canonical round-trips, order and
  identity cases, the capability set each type's facets require) plus a
  primitive-coverage check. Our own backends pass it in-repo; a custom
  backend that passes it is first-class.

## Component model (`xsd`)

- Components are constructed in **phases** so no traversal ever needs a
  cycle check (STYLE D4): (1) parse schema documents into raw form,
  (2) resolve QName references through a symbol table,
  (3) finalize in dependency order — a component's base/item/member types
  are complete before it is. Spec-forbidden circularities (`st-props-correct`
  circular unions, circular substitution groups, …) are rejected at phase 3
  with their named rule.
- All child collections are slices in document order. Maps exist only as
  internal indexes and never determine any order.
- Nothing derivable is stored (STYLE D3): no effective-facet caches —
  compute `Merge(base.EffectiveFacets(), declared)` on demand; no status
  booleans beside the facts that imply them.
- The model is **read-only** after construction; mutation/editing APIs are
  out of scope.

### Query and walk

Two access styles over the compiled model, one shared core:

- **Query**: direct lookups — element/attribute/type by QName — exposed
  through minimal capability views (STYLE T3), so a consumer that needs
  only `ElementByName` receives only that.
- **Walk**: traversal of a type's effective content model. The reusable
  core is an *algebra* (type-derivation validity, substitution-group
  acceptance, wildcard admission, attribute-use lookup) with two drivers:
  - a **push** driver — the exhaustive, schema-only Walker that visits
    every particle reachable through sequences/choices/all-groups and
    named-group references (the codegen consumer), and
  - a **pull** driver — the instance-guided Matcher that advances the
    content model one child at a time (the validation consumer).
  Substitution groups are not expanded at walk time (instance-time
  concern). Both drivers reuse the same algebra; neither reimplements it.

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

## Regex (`regex`)

One recursive-descent engine translating to Go's RE2, with a **flavor
flag** (PRINCIPLES 10):

- **XSD flavor** (pattern facets): implicitly anchored, `^`/`$` literal,
  non-capturing groups, no flags, `.` excludes `\n` and `\r`.
- **F&O flavor** (`fn:matches`/`fn:replace`/`fn:tokenize`): unanchored,
  real anchors, capturing groups, `i`/`s` flags honored; `m`/`x`/`q` and
  back-references are not expressible in RE2 and are flavor errors —
  surfaced, never silently accepted.

Character-class handling (`\d \w \p{…}`, subtraction `[a-z-[m]]`) is
shared. The package is a pure leaf (stdlib only).

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
- **Generated fast path**: backends export **code emitters**
  (`value.Emitter`, implemented by `builtin/strict` and `builtin/native`;
  user backends may implement it too). At codegen time the emitter
  contributes specialized decode/encode code for its types — parsing
  directly from the reader's byte window into the target field, no
  intermediate string, no boxed `value.Value`, facet checks inlined.
  A backend without an emitter simply falls back to the runtime path for
  its types.
- Runtime hot-path APIs follow the appender convention
  (`AppendCanonical(dst []byte, v) []byte`, `ParseBytes(b []byte)`) so
  even the non-generated path can be allocation-frugal.

The two paths implement the *same* pipeline stages with the *same* spec
rule IDs, which makes them **differentially testable**: for every type,
property tests feed identical input to both paths and require identical
values and identical error rule IDs, and `testing.AllocsPerRun` benchmarks
pin the fast path's allocation budget. A fast path that disagrees with the
runtime path is wrong by definition.

### Debuggability of parsing

When a value fails to parse, the error must localize the failure without
a debugger (extending E1–E3):

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

## Conformance & ratchet

- W3C suite at `testdata/xsdtests` (submodule, pinned).
- Expectations committed at `conformance/testdata/expectations/*.txt`, one
  line per test case, one lane per file (`datatypes`, `schema`, `instance`,
  `xpath`, `json`, `ber`); diffs make regressions obvious and `git blame`
  bisectable.
- `go test ./conformance -run TestConformance -count=1` compares;
  the same command under `GOXSD_RATCHET=1` re-baselines **upward only**.
  A regression fails loudly and must never be committed.

## Logging

`log/slog` injected at construction, namespaced groups, silent by default.
The debug level is designed for agents: messages carry rule ID, component
QName, and location so a conformance failure can be localized from logs
alone (`GOXSD_DEBUG=parser,validate` in tests).
