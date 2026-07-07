# goxsd8 Go Style — Non-Negotiable

Violations are grounds for the arbiter to reject a change even if tests
pass. Each rule has an ID; reviews cite rule IDs. The rationale behind
these rules lives in docs/PRINCIPLES.md.

## Control flow

**S1. Happy path on the left.** The success path runs down the left margin;
deviations exit early.

**S2. No `else` blocks.** Invert the condition and return/continue early.
`else` after a terminating `if` body is dead weight; `else` after a
non-terminating one usually hides a second function trying to get out.

```go
// BAD
if ok {
    doThing()
} else {
    return err
}

// GOOD
if !ok {
    return err
}
doThing()
```

**S3. Errors are never dropped — especially in loops.** A `for` body that
can fail must either return the error, or accumulate into an explicit
`errs []error` / `errors.Join` that the function returns. `_ =` on an error
value requires a comment proving it cannot matter.

## Errors

**E1. Every error is decorated.** Wrap with what you were doing and to what:
`fmt.Errorf("resolving base type of %s: %w", name, err)`. An error that
surfaces to a user must let them find the schema construct that caused it
without a debugger.

**E2. Errors map to spec validation rules.** Anything that represents a
schema or instance validity violation is an `*xsderr.Error` carrying the
spec rule ID (`cvc-…`, `cos-…`, `src-…`, `derivation-ok-…`). One rule ID per
error; if you can't name the rule, you haven't read the spec section yet.

**E3. Errors carry location.** Schema errors carry the schema document URI +
line + column (from `parser/xmltree` positions). Instance errors carry the
instance location. `xsderr.Loc` is threaded, not reconstructed.

## Data & determinism

**D1. Deterministic output, always.** Identical inputs produce byte-identical
output: generated code, canonical serializations, error lists, iteration
order of reported problems.

**D2. Never iterate a map into output.** Maps are allowed only as internal
lookup indexes. Anything ordered — child components, facets, errors,
generated declarations — lives in a slice, in document order (or a
spec-defined order). If you must drain a map, sort first and justify why a
slice wasn't kept alongside.

**D3. One fact, one encoding — no derivable state.** Do not store what can
be computed from what you already store, and never keep two encodings of
the same fact. No `Primitive bool` next to fundamental facets that already
imply it — a type that defines its own fundamental facets *is* a primitive;
expose `IsPrimitive()` as a derived method if callers need the answer. No
memoized caches without a profile showing a hot path. Two encodings of one
fact will drift; fewer fields, fewer invariants, fewer bugs.

**D4. No cycle checks — build in phases.** Structure construction so cycles
cannot exist at traversal time: parse into raw documents, resolve references
via named placeholders, then finalize components in dependency order.
A traversal that needs a `seen` set is a design smell; fix the construction
phase instead. (Where the spec itself permits cycles — e.g. circular
substitution-group or union checks the spec forbids — detect them once at
construction with a named `src-`/`cos-` rule error, then never again.)

**D5. No concurrency.** The parser, validator, and generators are pure
single-threaded transforms: no goroutines, channels, or locks in library
code. Determinism and simplicity outrank parallel speed; revisit only with
a measured, documented need — and then behind a seam, never scattered.

## Types & APIs

**T1. Illegal states unrepresentable.** Unexported fields + constructors
that validate. Closed sets are types with private tag fields, not `string`.
Mutually exclusive fields become a sum-style interface or separate types.
If a comment says "only valid when…", redesign.

**T2. Capabilities are interfaces, not type switches.** Value comparison,
length, digit counting, timezone-awareness etc. are small interfaces
(`value.Ordered`, `value.Lengthed`, …). A `switch v := v.(type)` over
concrete value types outside the defining package is a bug factory —
it silently excludes user-defined types.

*Exception — closed sums:* a set closed by the schema itself (an
`xs:choice` group in generated code, a variety) is a **sealed interface**
(unexported marker method), and consumers type-switch over its branches.
That is the Go sum type and it serves T1: the open/capability rule applies
to *extensible* sets, the sealed/switch rule to *closed* ones. Never mix
them up in either direction.

**T3. Minimal interfaces at boundaries.** Expose the narrowest capability
the consumer needs (a schema view that answers only `ElementByName`), not
the whole object.

**T4. No duplicate structures.** Before adding a type/function that looks
like an existing one, unify or explain in the commit message why they must
differ. Parallel near-identical code paths (two matchers, two resolvers)
rot independently.

**T5. Export nothing without a consumer.** Every exported identifier needs
a justification — a real caller, or a documented contract it fulfills — and
a doc comment (the lint gate enforces the comment; the arbiter reviews the
justification). The exported surface IS the product the library user sees
via `go doc`; every addition is a compatibility promise. Reviews inspect
the exported-surface diff of every change.

## Spec fidelity

**P1. Stick to the spec.** The local specs in `docs/specs/md/` are ground
truth, not intuition, not other implementations. When behavior is surprising,
quote the clause in the commit message.

**P2. Comment only constraints.** Code comments state what the code cannot:
spec rule being implemented, invariants, why a spec-deviation is deliberate.
Never narrate the next line.

**P3. Fail-open XPath gaps are tracked.** Every unsupported-construct
fallback carries `// GAP(<area>): <construct>` so gaps are greppable and
ratchetable.

**P4. Stream from the start.** Bounded memory on every input path: no
`io.ReadAll`, no whole-document buffering. Position tracking uses an
offset index over the stream, not retained content.

## Enforcement

The machine-checkable subset runs via `.golangci.yml`
(`go tool golangci-lint run` if installed as a module tool, otherwise the
`golangci-lint` binary): errcheck/errorlint/nilerr (S3, E1), revive
early-return/superfluous-else/indent-error-flow/exported (S1, S2, T5),
exhaustive (T1/T2 closed sums), sloglint no-global (L1), forbidigo banning
io.ReadAll and fmt.Print* in library code (P4, L1), plus govet/staticcheck/
unused/ineffassign/bodyclose and gofmt. Everything needing judgment — T4
duplicates, D2 map-iteration-into-output, D3 derivable state, D5, E2 rule
mapping, T5 justification — is the arbiter's and warden's job; keep the
linter set lean rather than approximating those.

## Logging

**L1. `log/slog` only,** through a logger accepted at construction
(`WithLogger` options), never a package-global. Components log under
namespaced groups (`parser`, `validate.cvc`, `xpath`). Debug logs must be
rich enough for an agent to localize a conformance failure without adding
prints: include rule ID, component QName, and location in every message.
Silent by default (`slog.DiscardHandler` when nil).
